package testcov

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var generatedFile = regexp.MustCompile("/*generated.*\\.go$")

// CheckCoverage checks coverage for each path that has coverage
func CheckCoverage(coverageFilePath string) (exitCode int) {
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
		warnOnTestedBlockIgnore(displayPath, sections, blockIgnores)
		warnCoveredInlineIgnore(displayPath, sections, inlineIgnores)

		// find untestedSections sections
		untestedSections := filter(sections, func(section Section) bool { return section.callCount == 0 })
		untestedSections = withoutSectionsInBlockIgnore(untestedSections, blockIgnores)
		untestedSections = withoutSectionsInInlineIgnore(untestedSections, inlineIgnores)

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
