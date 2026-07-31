package testcov

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
)

// reused regex
var perFileIgnore = regexp.MustCompile("// *untested sections: *(\\S+)")

// compares how many sections are actually untested against what's configured
type Untested struct {
	actualCount       int
	actualPercent     int
	configuredValue   int
	configuredPercent bool
}

func newUntested(actualCount int, totalLines int, configuredValue int, configuredPercent bool) Untested {
	return Untested{
		actualCount:       actualCount,
		actualPercent:     int(math.Round(float64(actualCount) / float64(totalLines) * 100)),
		configuredValue:   configuredValue,
		configuredPercent: configuredPercent,
	}
}

// untested is exactly as much as we expected, ignored (0%), or <= % than configured: nothing to do
func (m Untested) isAsConfigured() bool {
	if m.configuredPercent {
		return m.actualPercent <= m.configuredValue
	} else {
		return m.actualCount == m.configuredValue
	}
}

func (m Untested) isMoreThanConfigured() bool {
	if m.configuredPercent {
		return m.actualPercent > m.configuredValue
	} else {
		return m.actualCount > m.configuredValue
	}
}

func (m Untested) String() string {
	if m.configuredPercent {
		return fmt.Sprintf("(%v%% current vs %v%% configured)", m.actualPercent, m.configuredValue)
	} else {
		return fmt.Sprintf("(%v current vs %v configured)", m.actualCount, m.configuredValue)
	}
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
