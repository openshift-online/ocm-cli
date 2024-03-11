package utils

import (
	"sort"
	"strings"
)

func SliceToSortedString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	SortStringRespectLength(s)
	return "[" + strings.Join(s, ", ") + "]"
}

func SortStringRespectLength(s []string) {
	sort.Slice(s, func(i, j int) bool {
		l1, l2 := len(s[i]), len(s[j])
		if l1 != l2 {
			return l1 < l2
		}
		return s[i] < s[j]
	})
}
