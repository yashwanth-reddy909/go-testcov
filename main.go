package main

import (
	"fmt"
	"os"

	"github.com/grosser/go-testcov/internal/testcov"
)

const version = "v1.16.0"

// test injection point to enable test coverage of exit behavior
var exitFunction = os.Exit

// delegates to testcov.RunGoTestAndCheckCoverage, so we have an easy to test method
func main() {
	argv := os.Args[1:len(os.Args)] // remove go-testcov

	// print out version instead of go version when asked
	if len(argv) == 1 && argv[0] == "version" {
		fmt.Println(version)
		exitFunction(0)
	} else { // wrapping in else in case exitFunction was stubbed
		exitFunction(testcov.RunGoTestAndCheckCoverage(argv))
	}
}
