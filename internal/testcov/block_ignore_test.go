package testcov

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("block ignore", func() {
	Describe("findBlockIgnores", func() {
		It("finds nothing when there is no block ignore", func() {
			Expect(findBlockIgnores([]string{"func foo() {", "\tbar()", "}"})).To(Equal([]BlockIgnore{}))
		})

		It("finds the block below the comment", func() {
			Expect(findBlockIgnores([]string{"", "// untested block", "func foo() {", "\tbar()", "}"})).To(Equal(
				[]BlockIgnore{{2, 3, 5, false}},
			))
		})

		It("matches the indentation of the comment", func() {
			lines := []string{"func foo() {", "\t// untested block", "\tif bar {", "\t\tbaz()", "\t}", "}"}
			Expect(findBlockIgnores(lines)).To(Equal([]BlockIgnore{{2, 3, 5, false}}))
		})

		It("marks blocks with a random suffix", func() {
			Expect(findBlockIgnores([]string{"// untested block random", "func foo() {", "}"})).To(Equal(
				[]BlockIgnore{{1, 2, 3, true}},
			))
		})

		It("finds multiple blocks", func() {
			lines := []string{"// untested block", "func foo() {", "}", "// untested block", "func bar() {", "}"}
			Expect(findBlockIgnores(lines)).To(Equal([]BlockIgnore{{1, 2, 3, false}, {4, 5, 6, false}}))
		})

		It("warns and skips when the end of the block cannot be found", func() {
			var ignores []BlockIgnore
			stderr := captureStderr(func() {
				ignores = findBlockIgnores([]string{"\t\t// untested block", "func foo() {", "}"})
			})
			Expect(ignores).To(Equal([]BlockIgnore{}))
			Expect(stderr).To(Equal(
				"go-testcov: unable to find the end of the `// untested block` on line 1, a line starting with \t\t}",
			))
		})
	})

	Describe("warnCoveredBlockIgnore", func() {
		It("warns when an untested block is fully covered", func() {
			stderr := captureStderr(func() {
				warnCoveredBlockIgnore(
					"foo.go",
					[]Section{
						{"foo.go", 3, 2, 4, 3, 300002, 1},
						{"foo.go", 4, 3, 4, 8, 400003, 1},
					},
					findBlockIgnores([]string{"", "// untested block", "func foo() {", "\tbar()", "}"}),
				)
			})
			Expect(stderr).To(Equal("go-testcov (warn): foo.go:2 has `// untested block` but the block is tested\n"))
		})

		It("does not warn when an untested block is partially covered", func() {
			stderr := captureStderr(func() {
				warnCoveredBlockIgnore(
					"foo.go",
					[]Section{
						{"foo.go", 2, 2, 3, 3, 200002, 1},
						{"foo.go", 3, 3, 3, 8, 300003, 0},
					},
					findBlockIgnores([]string{"// untested block", "func foo() {", "\tbar()", "}"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})

		It("does not warn when an untested block is uncovered", func() {
			stderr := captureStderr(func() {
				warnCoveredBlockIgnore(
					"foo.go",
					[]Section{{"foo.go", 2, 2, 3, 3, 200002, 0}},
					findBlockIgnores([]string{"// untested block", "func foo() {", "\tbar()", "}"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})

		It("does not warn when an untested block has a random suffix", func() {
			stderr := captureStderr(func() {
				warnCoveredBlockIgnore(
					"foo.go",
					[]Section{{"foo.go", 2, 2, 3, 3, 200002, 1}},
					findBlockIgnores([]string{"// untested block random", "func foo() {", "\tbar()", "}"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})
	})
})
