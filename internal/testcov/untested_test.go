package testcov

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("untested", func() {
	Describe("configuredUntestedForFile", func() {
		It("returns 0,0 when not configured", func() {
			inTempDir(func() {
				writeFile(joinPath("foo"), "")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(0))
				Expect(percent).To(Equal(false))
				Expect(line).To(Equal(0))
			})
		})

		It("returns number of untested and line number of comment when configured", func() {
			inTempDir(func() {
				writeFile("foo", "// untested sections: 12")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(12))
				Expect(percent).To(Equal(false))
				Expect(line).To(Equal(1))
			})
		})

		It("returns number of untested and line number of comment when configured with multiple lines", func() {
			inTempDir(func() {
				writeFile("foo", "... bork ... \n // untested sections: 12 \n ... bork ...")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(12))
				Expect(percent).To(Equal(false))
				Expect(line).To(Equal(2))
			})
		})

		It("returns ignored when configured", func() {
			inTempDir(func() {
				writeFile("foo", "... bork ... \n // untested sections: ignore \n ... bork ...")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(100))
				Expect(percent).To(Equal(true))
				Expect(line).To(Equal(2))
			})
		})

		It("returns percent when configured", func() {
			inTempDir(func() {
				writeFile("foo", "... bork ... \n // untested sections: 10% \n ... bork ...")
				count, percent, line := configuredUntestedForFile("foo")
				Expect(count).To(Equal(10))
				Expect(percent).To(Equal(true))
				Expect(line).To(Equal(2))
			})
		})
	})
})
