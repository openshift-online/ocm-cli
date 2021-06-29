/*
Copyright (c) 2019 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains functions that help displaying information in a tabular form

package table

import (
	"fmt"
	"io"
	"strings"
)

// FindMapValue will find a key and retrieve its value from the given map. The key has to be
// a string and can be multilayered, for example `foo.bar`. Returns the value and a boolean
// indicating if the value was found.
func FindMapValue(data map[string]interface{}, key string) (string, bool) {

	// Split key into array
	keys := strings.Split(key, ".")

	// loop though elements in sliced string:
	for _, element := range keys {

		// if key is found, continue:
		if val, ok := data[element]; ok {

			switch typed := val.(type) {

			// If key points to interface insance:
			case map[string]interface{}:
				data = typed

			// If key points to an end value:
			default:
				return fmt.Sprintf("%v", typed), true

			}

		} else { // Key not in map
			return "", false
		}
	}

	return "", false
}

// PrintPadded turns an array into a padded string and outputs it into the given writer.
func PrintPadded(w io.Writer, columns []string, padding []int) error {
	updated := updateRowPad(columns, padding)
	var finalString string
	for _, str := range updated {
		finalString = fmt.Sprint(finalString, str)
	}
	_, err := fmt.Fprint(w, finalString+"\n")
	return err
}

func updateRowPad(columnList []string, columnPad []int) []string {
	st := columnList
	fixLen := len(columnPad) - len(st)
	if fixLen < 0 {
		valueToUse := columnPad[len(columnPad)-1]
		for i := 0; i < fixLen*(-1); i++ {
			columnPad = append(columnPad, valueToUse)
		}
	}
	for i := range st {
		if len(st[i]) < columnPad[i] {
			st[i] = st[i] + strings.Repeat(" ", columnPad[i]-len(st[i]))
		} else {
			st[i] = st[i][:columnPad[i]-2] + "  "
		}
	}
	return st
}
