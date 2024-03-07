package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validates SliceToSortedString", func() {
	It("Empty when slice is empty", func() {
		s := SliceToSortedString([]string{})
		Expect("").To(Equal(s))
	})

	It("Sorted when slice is filled", func() {
		s := SliceToSortedString([]string{"b", "a", "c", "a10", "a1", "a20", "a2", "1", "2", "10", "20"})
		Expect("[1, 2, a, b, c, 10, 20, a1, a2, a10, a20]").To(Equal(s))
	})
})
