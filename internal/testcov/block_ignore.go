package testcov

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// reused regex
var blockIgnore = regexp.MustCompile("(?m)^([\t ]*)// *untested block(\\s|:|,|$)")
var randomBlockIgnore = regexp.MustCompile(`// *untested block\s+random(\s|:|,|$)`)

// a `// untested block` comment and the lines of the block it ignores
type BlockIgnore struct {
	commentLine int // line the `// untested block` comment is on
	startLine   int // first line of the ignored block
	endLine     int // last line of the ignored block, the closing `}`
	random      bool
}

// find all `// untested block` comments and the blocks they ignore
// warns about comments whose block end cannot be found, they ignore nothing
func findBlockIgnores(lines []string) (ignores []BlockIgnore) {
	ignores = []BlockIgnore{}

	for i, line := range lines {
		match := blockIgnore.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		commentLine := i + 1
		indentation := match[1]
		search := indentation + "}"
		endIndex := findLineStartingWith(lines, commentLine, search)
		if endIndex == -1 {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"go-testcov: unable to find the end of the `// untested block` on line %d, a line starting with %v",
				commentLine, search,
			)
			continue
		}

		ignores = append(ignores, BlockIgnore{
			commentLine: commentLine,
			startLine:   commentLine + 1,
			endLine:     endIndex + 1,
			random:      randomBlockIgnore.MatchString(line),
		})
	}
	return
}

// true when the section is inside one of the given ignored blocks
func inBlockIgnore(ignores []BlockIgnore, section Section) bool {
	for _, ignore := range ignores {
		if ignore.startLine <= section.startLine && section.endLine <= ignore.endLine {
			return true
		}
	}
	return false
}

// remove sections that are inside a `// untested block` ignore
func removeSectionsInBlockIgnore(sections []Section, blockIgnores []BlockIgnore) []Section {
	return filter(sections, func(section Section) bool {
		return !inBlockIgnore(blockIgnores, section)
	})
}

// warn when blocks are actually tested
func warnCoveredBlockIgnore(path string, sections []Section, blockIgnores []BlockIgnore) {
	for _, ignore := range blockIgnores {
		// skip flaky-coverage warnings (goroutines, timing, randomness)
		if ignore.random {
			continue
		}

		if allSectionsInRangeCovered(sections, ignore.startLine, ignore.endLine) {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"go-testcov (warn): %v:%v has `// untested block` but the block is tested\n",
				path, ignore.commentLine,
			)
		}
	}
}

// find the first line starting with the search term
// returns -1 when not found
func findLineStartingWith(lines []string, searchFromIndex int, search string) int {
	for i, line := range lines[searchFromIndex:] {
		if strings.HasPrefix(line, search) {
			return searchFromIndex + i
		}
	}
	return -1
}
