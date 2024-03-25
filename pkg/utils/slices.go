package utils

import "reflect"

func Contains[T comparable](slice []T, element T) bool {
	for _, sliceElement := range slice {
		if reflect.DeepEqual(sliceElement, element) {
			return true
		}
	}

	return false
}
