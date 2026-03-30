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

	Context("MapToSortedString", func() {
		It("handles empty map", func() {
			Expect(MapToSortedString(map[string][]string{})).To(Equal(""))
		})

		It("handles nil map", func() {
			Expect(MapToSortedString(nil)).To(Equal(""))
		})

		It("correctly sorts mapping", func() {
			Expect(MapToSortedString(map[string][]string{
				"a":  {"a", "a1"},
				"b":  {"b", "b1", "b2"},
				"a1": {"a1"},
			})).To(Equal("{{a: [a, a1]}, {b: [b, b1, b2]}, {a1: [a1]}}"))
		})

		It("renders empty value slices as []", func() {
			Expect(MapToSortedString(map[string][]string{
				"key": {},
			})).To(Equal("{{key: []}}"))
		})

		It("does not mutate input slices", func() {
			input := map[string][]string{
				"k": {"b", "a"},
			}
			_ = MapToSortedString(input)
			Expect(input["k"]).To(Equal([]string{"b", "a"}))
		})
	})
})
