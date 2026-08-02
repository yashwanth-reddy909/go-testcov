# go-testcov [![Test](https://github.com/grosser/go-testcov/actions/workflows/test.yml/badge.svg)](https://github.com/grosser/go-testcov/actions?query=branch%3Amaster) [![coverage](https://img.shields.io/badge/coverage-100%25-success.svg)](https://github.com/grosser/go-testcov)

`go test` that fails on untested lines and shows them

 - 🎉 **Instant** and **actionable** feedback on 💚 test run
 - 🚀 Fast PRs: avoid comments and CI failures
 - 💰 No 3rd-party payment / integration / security-leaks 
 - Highlight untested code sections with inline `// untested section` comment (or on the line above as a separate comment)
 - Onboard untested code (top of the file `// untested sections: 5` comment, warns when below)
 - Ignore untested files (top of the file `// untested sections: ignore` comment)
 - Ignore large amounts of poorly tested code (top of the file `// untested sections: 50%` comment, does not warn when below that %)
 - Ignore untested functions with `// untested block` comment in function header
 - Run `ginkgo` with `go-testcov ginkgo ./...`

```
go get github.com/grosser/go-testcov
go-testcov . # same arguments as `go test` uses, so for example `go-testcov ./...` for everything
...
test output
...
pkg.go new untested sections introduced (2 current vs 0 configured)
pkg.go:20.14,21.11
pkg.go:54.5,56.5
```


## Notes

 - Docs for [coverage in go](https://blog.golang.org/cover)
 - Runtime overhead for coverage is about 3%
 - Use `-covermode atomic` when testing parallel algorithms
 - Use `// untested section random` or `// untested block random` to skip tested warnings (goroutines, timing, randomness)
 - To keep the `coverage.out` file run with `-cover`
 - `go-testcov version` to see current version


## Makefile setup to use a consistent version of go-testcov

```
.PHONY: test
test: go-testcov ## Unit test
	$(GOTESTCOV) ./... -covermode atomic

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
GOTESTCOV ?= $(LOCALBIN)/go-testcov
GOTESTCOV_VERSION ?= v1.15.0

.PHONY: go-testcov
go-testcov: $(LOCALBIN) # Download go-testcov (replace existing if incorrect version)
	@(test -f $(GOTESTCOV) && $(GOTESTCOV) version | grep "$(GOTESTCOV_VERSION)" >/dev/null) || \
	(rm -f $(GOTESTCOV) && echo "Installing $(GOTESTCOV) $(GOTESTCOV_VERSION)" && \
	GOBIN=$(LOCALBIN) go install github.com/grosser/go-testcov@$(GOTESTCOV_VERSION))
```

## Architecture

### Execution

```mermaid
graph TD;
GoTestCov(go-testcov .)
GoTest(go test . -cover --coverprofile coverage.out)
Coverage(coverage.out)
GoTestCov--runs-->GoTest;
GoTest--produces-->Coverage;
GoTestCov--parses-->Coverage;
```

`coverage.out` shows the coverage of each file "section".

### Section

A "section" is a chunk of uninterrupted code, for example:

```go
func main() {
  if foo(1) {
      fmt.Print("Hi")
  }
  fmt.Print("Ho")
}
```

has 3 sections:
```
github.com/foo/bar/main.go:1.13,2.13 1 1
github.com/foo/bar/main.go:2.13,3.4 1 1
github.com/foo/bar/main.go:5.3,5.18 1 1
```

1. `1.13,2.13`: after opening `main() {` until after `if foo(1) {`
2. `2.13,3.4`: after `if foo(1) {` until after `if` closing `}`
3. `5.3,5.18`: `fmt.Print("Ho")`

- the `else` case (aka "what if foo(1) returns false") has no coverage information
- when not using modules the path is `/full/path/to/main.go`

### Block

A "block" is everything from a `// untested block` comment until the closing `}` at the same indentation.

```go
// untested block
func main() {
  if foo(1) {
      fmt.Print("Hi")
  }
  fmt.Print("Ho")
}
```

Marking the example above ignores all 3 of its sections.

- put `// untested block` above a function, or above an `if` when all of its sections should be ignored — at the same indentation as that block's closing `}`


## Development

Run `go-testcov` on itself:

```
make
```

### inspecting coverage output

- create a new `foo/main.go` file with the code you want to inspect
- `go test -cover -coverprofile cov.out foo/main.go`
- `cat cov.out`

## Release

- never release a major version unless absolutely necessary, since that requires a /v2 path
- make new version commit that changes version in readme "Makefile setup" + main.go
- push and tag the commit

Author
======
[Michael Grosser](http://grosser.it)<br/>
michael@grosser.it<br/>
License: MIT<br/>
