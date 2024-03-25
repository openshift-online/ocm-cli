package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Slices", func() {
	Context("Validates Contains", func() {
		It("Return false when input is empty", func() {
			Expect(false).To(Equal(Contains([]string{}, "any")))
		})

		It("Return true when input is populated and present", func() {
			Expect(true).To(Equal(Contains([]string{"test", "any"}, "any")))
		})

		It("Return false when input is populated and not present", func() {
			Expect(false).To(Equal(Contains([]string{"test", "any"}, "none")))
		})
	})
})
