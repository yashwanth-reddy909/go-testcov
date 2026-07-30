package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const version = "v1.15.0"

// reused regex
var perFileIgnore = regexp.MustCompile("// *untested sections: *(\\S+)")
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

	var command []string
	// user trying to use ginkgo binary, or locally installed one ?
	if len(argv) >= 1 && strings.HasSuffix("/"+argv[0], "/ginkgo") {
		// - files (i.e. ./...) need to come last
		// - subcommands need to come first, see https://github.com/onsi/ginkgo/issues/1531
		length := len(argv)
		command = argv[0 : length-1]
		command = append(command, "-cover", "-coverprofile", coveragePath, argv[length-1])
	} else {
		command = append(append([]string{"go", "test"}, argv...), "-coverprofile", coveragePath)
	}

	exitCode = runCommand(command...)

	if exitCode != 0 {
		return exitCode
	}
	return checkCoverage(coveragePath)
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
		configuredUntested, percentUntested, configuredUntestedAtLine := configuredUntestedForFile(readPath)
		lines := strings.Split(readFile(readPath), "\n")

		// find ignores once, so warnings about them and their effect on coverage stay in sync
		blockIgnores := findBlockIgnores(lines)
		inlineIgnores := findInlineIgnores(lines)

		// print warnings logs for covered blocks and sections
		warnCoveredBlockIgnore(displayPath, sections, blockIgnores)
		warnCoveredInlineIgnore(displayPath, sections, inlineIgnores)

		untested := removeSectionsInInlineIgnore(
			removeSectionsInBlockIgnore(untestedFromSections(sections), blockIgnores), inlineIgnores,
		)
		actualUntested := len(untested)
		actualUntestedPercent := int(math.Round(float64(actualUntested) / float64(len(lines)) * 100))

		// what to show the user
		var details string
		if percentUntested {
			details = fmt.Sprintf("(%v%% current vs %v%% configured)", actualUntestedPercent, configuredUntested)
		} else {
			details = fmt.Sprintf("(%v current vs %v configured)", actualUntested, configuredUntested)
		}

		if (!percentUntested && actualUntested == configuredUntested) || (percentUntested && actualUntestedPercent <= configuredUntested) {
			// exactly as much as we expected, ignored (0%), or <= % than configured: nothing to do
		} else if actualUntested > configuredUntested {
			printUntestedSections(untested, displayPath, details)
			exitCode = 1 // at least 1 failure, so say to add more tests
		} else { // never hit in % case
			_, _ = fmt.Fprintf(
				os.Stderr,
				"%v has less untested sections %v, decrement configured untested?\nconfigured on: %v:%v",
				displayPath, details, readPath, configuredUntestedAtLine)
		}
	})

	return exitCode
}

func printUntestedSections(sections []Section, displayPath string, details string) {
	// TODO: color when tty
	_, _ = fmt.Fprintf(os.Stderr, "%v new untested sections introduced %v\n", displayPath, details)

	// sort sections since go coverage output is not sorted
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].sortValue < sections[j].sortValue
	})

	// print copy-paste friendly snippets
	for _, section := range sections {
		_, _ = fmt.Fprintln(os.Stderr, displayPath+":"+section.Location())
	}
}

// keep only the items for which keep returns true
func filter[T any](items []T, keep func(T) bool) []T {
	kept := []T{}
	for _, item := range items {
		if keep(item) {
			kept = append(kept, item)
		}
	}
	return kept
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

// keep only sections that were not covered (callCount == 0)
func untestedFromSections(sections []Section) []Section {
	return filter(sections, func(section Section) bool {
		return section.callCount == 0
	})
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

// How many sections are expected to be untested ?
//
// - 0 if not configured
// - count when configured with "x"
// - percentage when configured with "x%"
// - 100% if "ignore"
//
// also returns at what line we found the comment, so we can point the user to it
func configuredUntestedForFile(path string) (count int, percent bool, lineNumber int) {
	content := readFile(path)
	match := perFileIgnore.FindStringSubmatch(content)
	if len(match) == 2 { // found a config ?
		config := match[1]
		line := lineNumberOfMatch(content)

		if config == "ignore" {
			return 100, true, line // 100% which does not warn for any amount, so basically ignored
		} else if strings.HasSuffix(config, "%") {
			return stringToInt(config[:len(config)-1]), true, line // percent
		} else {
			return stringToInt(config), false, line // count
		}
	} else {
		return 0, false, 0
	}
}
