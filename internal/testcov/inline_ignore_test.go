package testcov

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("inline ignore", func() {
	Describe("findInlineIgnores", func() {
		It("finds nothing when there is no inline comment", func() {
			Expect(findInlineIgnores([]string{"foo"})).To(Equal([]InlineIgnore{}))
		})

		It("finds a trailing comment", func() {
			Expect(findInlineIgnores([]string{"foo // untested section"})).To(Equal(
				[]InlineIgnore{{1, false, false}},
			))
		})

		It("finds a comment on its own line", func() {
			Expect(findInlineIgnores([]string{"// untested section", "foo"})).To(Equal(
				[]InlineIgnore{{1, true, false}},
			))
		})

		It("marks comments with a random suffix", func() {
			Expect(findInlineIgnores([]string{"foo // untested section random"})).To(Equal(
				[]InlineIgnore{{1, false, true}},
			))
		})
	})

	Describe("warnCoveredInlineIgnore", func() {
		It("warns when inline comment is on covered code", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{{"foo.go", 1, 2, 1, 3, 100002, 1}},
					findInlineIgnores([]string{"foo // untested section"}),
				)
			})
			Expect(stderr).To(Equal("go-testcov (warn): foo.go:1 has `// untested section` but is tested\n"))
		})

		It("warns when inline comment is above covered code", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{{"foo.go", 2, 2, 2, 3, 200002, 1}},
					findInlineIgnores([]string{"// untested section", "foo"}),
				)
			})
			Expect(stderr).To(Equal("go-testcov (warn): foo.go:1 has `// untested section` but the code below is tested\n"))
		})

		It("does not warn when inline comment has random suffix", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{{"foo.go", 1, 2, 1, 3, 100002, 1}},
					findInlineIgnores([]string{"foo // untested section random"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})

		It("does not warn when above-line comment has random suffix", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{{"foo.go", 2, 2, 2, 3, 200002, 1}},
					findInlineIgnores([]string{"// untested section random", "foo"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})

		It("does not warn when inline comment is on uncovered code", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{{"foo.go", 1, 2, 1, 3, 100002, 0}},
					findInlineIgnores([]string{"foo // untested section"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})

		It("warns when inline comment is on a line with no coverage information", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{},
					findInlineIgnores([]string{"foo // untested section"}),
				)
			})
			Expect(stderr).To(Equal("go-testcov (warn): foo.go:1 has `// untested section` but is tested\n"))
		})

		It("warns when above-line comment points to a line with no coverage information", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{},
					findInlineIgnores([]string{"// untested section", "foo"}),
				)
			})
			Expect(stderr).To(Equal(
				"go-testcov (warn): foo.go:1 has `// untested section` but the code below is tested\n",
			))
		})

		It("does not warn when one of multiple sections on the line is uncovered", func() {
			stderr := captureStderr(func() {
				warnCoveredInlineIgnore(
					"foo.go",
					[]Section{
						{"foo.go", 1, 2, 1, 3, 100002, 1},
						{"foo.go", 1, 4, 1, 6, 100004, 0},
					},
					findInlineIgnores([]string{"foo || bar // untested section"}),
				)
			})
			Expect(stderr).To(Equal(""))
		})

		It("keeps random suffix inline comments as ignores", func() {
			sections := removeSectionsInInlineIgnore(
				[]Section{{"foo.go", 1, 2, 1, 3, 100002, 0}},
				findInlineIgnores([]string{"foo // untested section random"}),
			)
			Expect(sections).To(Equal([]Section{}))
		})
	})
})
