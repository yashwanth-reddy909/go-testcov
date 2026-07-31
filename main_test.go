package main

import (
	"os"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("go-testcov", func() {
	Describe("main", func() {
		It("exits", func() {
			withFakeGo("touch coverage.out\necho go \"$@\"", func() {
				exitCode := -1

				// fake the exit function so we can test it
				exitFunction = func(got int) { exitCode = got }
				defer func() { exitFunction = os.Exit }()

				withOsArgs([]string{"executable-name", "some", "arg"}, func() {
					expectCommand(
						func() int {
							main()
							return exitCode
						},
						[]interface{}{0, "go test some arg -coverprofile coverage.out\n", ""},
					)
				})
			})
		})

		It("shows version", func() {
			exitCode := -1
			exitFunction = func(got int) { exitCode = got }
			defer func() { exitFunction = os.Exit }()

			withOsArgs([]string{"executable-name", "version"}, func() {
				expectCommand(
					func() int {
						main()
						return exitCode
					},
					[]interface{}{0, version + "\n", ""},
				)
			})
		})
	})
})
