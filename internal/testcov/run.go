package testcov

import (
	"os"
	"slices"
	"strings"
)

// RunGoTestAndCheckCoverage runs go test with given arguments + coverage and inspects coverage after run
func RunGoTestAndCheckCoverage(argv []string) (exitCode int) {
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

	return CheckCoverage(coveragePath)
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
