package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

const version = "v1.15.0"

var generatedFile = regexp.MustCompile("/*generated.*\\.go$")

// test injection point to enable test coverage of exit behavior
var exitFunction = os.Exit

// delegates to runGoTestAndCheckCoverage, so we have an easy to test method
func main() {
	argv := os.Args[1:len(os.Args)] // remove go-testcov

	// print out version instead of go version when asked
	if len(argv) == 1 && argv[0] == "version" {
		fmt.Println(version)
		exitFunction(0)
	} else { // wrapping in else in case exitFunction was stubbed
		exitFunction(runGoTestAndCheckCoverage(argv))
	}
}

// run go test with given arguments + coverage and inspect coverage after run
func runGoTestAndCheckCoverage(argv []string) (exitCode int) {
	coveragePath := "coverage.out"
	_ = os.Remove(coveragePath) // remove file if it exists, to avoid confusion when test run fails

	// allow users to keep the coverage.out file when they passed -cover manually
	// TODO: parse options to find the location the user wanted and use+keep that
	if !slices.Contains(argv, "-cover") {
		defer os.Remove(coveragePath)
	}

	// run test
	exitCode = runCommand(buildTestCommand(argv, coveragePath)...)
	if exitCode != 0 {
		return exitCode
	}

	return checkCoverage(coveragePath)
}

// build the `go test` (or `ginkgo`) command that writes coverage to coveragePath
func buildTestCommand(argv []string, coveragePath string) []string {
	// user trying to use ginkgo binary, or locally installed one ?
	if len(argv) >= 1 && strings.HasSuffix("/"+argv[0], "/ginkgo") {
		// - files (i.e. ./...) need to come last
		// - subcommands need to come first, see https://github.com/onsi/ginkgo/issues/1531
		length := len(argv)
		command := argv[0 : length-1]
		return append(command, "-cover", "-coverprofile", coveragePath, argv[length-1])
	} else {
		return append(append([]string{"go", "test"}, argv...), "-coverprofile", coveragePath)
	}
}

// check coverage for each path that has coverage
func checkCoverage(coverageFilePath string) (exitCode int) {
	exitCode = 0
	sectionsByPath := groupSectionsByPath(getSections(coverageFilePath))

	wd, err := os.Getwd()
	check(err)

	iterateBySortedKey(sectionsByPath, func(path string, sections []Section) {
		// skip generated files since their coverage does not matter and would often have gaps
		if generatedFile.MatchString(path) {
			return
		}

		displayPath, readPath := normalizeCoveredPath(path, wd)
		configuredUntestedValue, configuredUntestedPercent, configuredUntestedAtLine := configuredUntestedForFile(readPath)
		lines := strings.Split(readFile(readPath), "\n")

		blockIgnores := findBlockIgnores(lines)
		inlineIgnores := findInlineIgnores(lines)

		// print warnings for parts that incorrectly claim to be untestedSections
		warnCoveredBlockIgnore(displayPath, sections, blockIgnores)
		warnCoveredInlineIgnore(displayPath, sections, inlineIgnores)

		// find untestedSections sections
		untestedSections := filter(sections, func(section Section) bool { return section.callCount == 0 })
		untestedSections = removeSectionsInBlockIgnore(untestedSections, blockIgnores)
		untestedSections = removeSectionsInInlineIgnore(untestedSections, inlineIgnores)

		// compare config against what we found
		untested := newUntested(len(untestedSections), len(lines), configuredUntestedValue, configuredUntestedPercent)
		if untested.isAsConfigured() {
			// nothing to do
		} else if untested.isMoreThanConfigured() {
			printUntestedSections(untestedSections, displayPath, untested.String())
			exitCode = 1 // at least 1 failure, add more tests
		} else { // less than configured which does not happen when % is used
			_, _ = fmt.Fprintf(
				os.Stderr,
				"%v has less untested sections %v, decrement configured untested?\nconfigured on: %v:%v",
				displayPath, untested.String(), readPath, configuredUntestedAtLine)
		}
	})

	return exitCode
}

func groupSectionsByPath(sections []Section) (grouped map[string][]Section) {
	grouped = map[string][]Section{}
	for _, section := range sections {
		grouped[section.path] = append(grouped[section.path], section)
	}
	return
}

// get all sections from coverage file
func getSections(coverageFilePath string) (sections []Section) {
	sections = []Section{}
	content := readFile(coverageFilePath)

	lines := splitWithoutEmpty(content, '\n')

	// remove the initial `set: mode` line
	if len(lines) == 0 {
		return
	}
	lines = lines[1:]

	for _, line := range lines {
		sections = append(sections, NewSection(line))
	}

	return
}

// find relative path of file in current directory
func findFile(path string) (readPath string) {
	parts := strings.Split(path, string(os.PathSeparator))
	for len(parts) > 0 {
		_, err := os.Stat(strings.Join(parts, string(os.PathSeparator)))
		if err != nil {
			parts = parts[1:] // shift directory to continue to look for file
		} else {
			break
		}
	}
	return strings.Join(parts, string(os.PathSeparator))
}

// remove path prefix like "github.com/user/lib", but cache the call to os.Get
func normalizeCoveredPath(path string, workingDirectory string) (displayPath string, readPath string) {
	modulePrefixSize := 3 // foo.com/bar/baz + file.go
	separator := string(os.PathSeparator)
	parts := strings.SplitN(path, separator, modulePrefixSize+1)
	goPath, hasGoPath := os.LookupEnv("GOPATH")
	inGoPath := false
	goPrefixedPath := joinPath(goPath, "src", path)

	if hasGoPath {
		_, err := os.Stat(goPrefixedPath)
		inGoPath = !os.IsNotExist(err)
	}

	// path too short, return a good guess
	if len(parts) <= modulePrefixSize {
		if inGoPath {
			return path, goPrefixedPath
		} else {
			return path, path
		}
	}

	prefix := strings.Join(parts[:modulePrefixSize], separator)
	demodularized := findFile(strings.SplitN(path, prefix+separator, 2)[1])

	// folder is not in go path ... remove module nesting
	if !inGoPath {
		return demodularized, demodularized
	}

	// we are in a nested folder ... remove module nesting and expand full goPath
	if strings.HasSuffix(workingDirectory, prefix) {
		return demodularized, goPrefixedPath
	}

	// testing remote package, don't expand display but expand full goPath
	return path, goPrefixedPath
}
