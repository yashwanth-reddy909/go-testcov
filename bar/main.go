package main

import "os"

func run(a, b int) int {
	if a > 0 {
		return -1
	} else {
		x := b +
			1
		_ = x
	}
	// untested block
	if b > 5 {
		return 1
	}
	return 2
}

func main() {
	os.Exit(run(len(os.Args), 0))
}
