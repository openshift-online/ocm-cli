/*
Copyright (c) 2026 Red Hat, Inc.
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

package ingress

import (
	"fmt"
	"strings"
)

func ExtractExcludedNamespaceSelectors(selectorsStr string) (map[string][]string, error) {
	if len(selectorsStr) == 0 {
		return nil, nil
	}
	excludedNamespaceSelectors := make(map[string][]string)
	for _, selector := range strings.Split(selectorsStr, ",") {
		if !strings.Contains(selector, "=") {
			return nil, fmt.Errorf("Expected key=value format for excluded-namespace-selectors")
		}
		tokens := strings.Split(selector, "=")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("Invalid excluded namespace selector: '%s'", selector)
		}
		key := strings.TrimSpace(tokens[0])
		if key == "" {
			return nil, fmt.Errorf("Invalid excluded namespace selector: '%s'", selector)
		}
		value := strings.TrimSpace(tokens[1])
		if value == "" {
			return nil, fmt.Errorf("Invalid excluded namespace selector: '%s'", selector)
		}
		excludedNamespaceSelectors[key] = append(excludedNamespaceSelectors[key], value)
	}
	return excludedNamespaceSelectors, nil
}
