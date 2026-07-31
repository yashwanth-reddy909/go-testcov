package testcov

import (
	"fmt"
	"os"
	"regexp"
)

// reused regex
var inlineIgnore = "//.*untested section(\\s|:|,|$)"
var anyInlineIgnore = regexp.MustCompile(inlineIgnore)
var startsWithInlineIgnore = regexp.MustCompile("^\\s*" + inlineIgnore)
var randomInlineIgnore = regexp.MustCompile(`//.*untested section\s+random(\s|:|,|$)`)

// a `// untested section` comment, either trailing on a line or on its own line above the code
type InlineIgnore struct {
	line       int  // line the comment is on
	startsLine bool // true when the comment is the whole line, so it also ignores the line below
	random     bool
}

// find all `// untested section` comments
func findInlineIgnores(lines []string) (ignores []InlineIgnore) {
	ignores = []InlineIgnore{}

	for i, line := range lines {
		if !anyInlineIgnore.MatchString(line) {
			continue
		}

		ignores = append(ignores, InlineIgnore{
			line:       i + 1,
			startsLine: startsWithInlineIgnore.MatchString(line),
			random:     randomInlineIgnore.MatchString(line),
		})
	}
	return
}

// true when the ignore comment applies to the given line
func (ignore InlineIgnore) ignores(line int) bool {
	return ignore.line == line || (ignore.startsLine && ignore.line == line-1)
}

// true when the line is ignored by one of the given inline comments
func inInlineIgnore(ignores []InlineIgnore, line int) bool {
	return anyMatch(ignores, func(ignore InlineIgnore) bool { return ignore.ignores(line) })
}

// remove sections that are marked with a `// untested section` comment
// NOTE: this is a bit rough as it does not account for partial lines via start/end characters
func removeSectionsInInlineIgnore(sections []Section, inlineIgnores []InlineIgnore) []Section {
	return filter(sections, func(section Section) bool {
		for lineNumber := section.startLine; lineNumber <= section.endLine; lineNumber++ {
			if inInlineIgnore(inlineIgnores, lineNumber) {
				return false
			}
		}
		return true
	})
}

// warn when inline ignore markers point to code that is actually covered
func warnCoveredInlineIgnore(path string, sections []Section, inlineIgnores []InlineIgnore) {
	for _, ignore := range inlineIgnores {
		// skip flaky-coverage warnings (goroutines, timing, randomness)
		if ignore.random {
			continue
		}

		if ignore.startsLine {
			// TODO: ideally you should be allowed to have a long comment block and then the code
			if allSectionsInRangeCovered(sections, ignore.line+1, ignore.line+1) {
				_, _ = fmt.Fprintf(
					os.Stderr,
					"go-testcov (warn): %v:%v has `// untested section` but the code below is tested\n",
					path, ignore.line,
				)
			}
		} else {
			if allSectionsInRangeCovered(sections, ignore.line, ignore.line) {
				_, _ = fmt.Fprintf(
					os.Stderr,
					"go-testcov (warn): %v:%v has `// untested section` but is tested\n",
					path, ignore.line,
				)
			}
		}
	}
}
