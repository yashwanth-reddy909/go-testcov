package main

import "os"

func run(a, b int) int {
	x := a +
		b
	// untested block
	if x > 5 {
		return 1
	}
	return 2
}

func main() {
	os.Exit(run(len(os.Args), 0))
}
