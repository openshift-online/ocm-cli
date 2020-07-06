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

package urls

import (
	"fmt"
)

// Resources that return a list of multiple items
var listResourceURLs = map[string]string{
	"accounts":       "/api/accounts_mgmt/v1/accounts",
	"accts":          "/api/accounts_mgmt/v1/accounts",
	"subscriptions":  "/api/accounts_mgmt/v1/subscriptions",
	"subs":           "/api/accounts_mgmt/v1/subscriptions",
	"organizations":  "/api/accounts_mgmt/v1/organizations",
	"orgs":           "/api/accounts_mgmt/v1/organizations",
	"clusters":       "/api/clusters_mgmt/v1/clusters",
	"role_bindings":  "/api/accounts_mgmt/v1/role_bindings",
	"resource_quota": "/api/accounts_mgmt/v1/resource_quota",
	"roles":          "/api/accounts_mgmt/v1/roles",
	"skus":           "/api/accounts_mgmt/v1/skus",
	"sku_rules":      "/api/accounts_mgmt/v1/sku_rules",
}

// Resources that apply to a specific item and require an appended argument
var individualResourceURLs = map[string]string{
	"account":      "/api/accounts_mgmt/v1/accounts/",
	"acct":         "/api/accounts_mgmt/v1/accounts/",
	"subscription": "/api/accounts_mgmt/v1/subscriptions/",
	"sub":          "/api/accounts_mgmt/v1/subscriptions/",
	"organization": "/api/accounts_mgmt/v1/organizations/",
	"org":          "/api/accounts_mgmt/v1/organizations/",
	"cluster":      "/api/clusters_mgmt/v1/clusters/",
	"role_binding": "/api/accounts_mgmt/v1/role_bindings/",
	"role":         "/api/accounts_mgmt/v1/roles/",
	"sku":          "/api/accounts_mgmt/v1/skus/",
	"sku_rule":     "/api/accounts_mgmt/v1/sku_rules",
}

// Expand returns full URI to UHC resources based on an alias. An alias
// allows for shortcuts on the CLI, such as replace "accts" with the
// full URI of the resource. Lists of resources require just the alias as
// a parameter, while getting/posting individual resources requires the additional
// ID of the resource.
func Expand(argv []string) (string, error) {
	if len(argv) < 1 || len(argv) > 2 {
		msg := fmt.Errorf("Expected 1 (for Lists) or 2 (for a specific resource) but got %d", len(argv))
		return "", msg
	}

	preParsePath := argv[0]

	if path, ok := listResourceURLs[preParsePath]; ok {
		return path, nil
	}

	if path, ok := individualResourceURLs[preParsePath]; ok {
		// append the argument ID to the URL
		url, err := expandResourceWithID(path, argv)
		if err != nil {
			return "", err
		}
		return url, err
	}

	return preParsePath, nil
}

func Resources() []string {
	resources := make([]string, 0)
	for r := range listResourceURLs {
		resources = append(resources, r)
	}
	for r := range individualResourceURLs {
		resources = append(resources, r)
	}
	return resources
}

func expandResourceWithID(path string, argv []string) (string, error) {
	if len(argv) != 2 {
		return "", fmt.Errorf("Resource requires an ID, but got none")
	}
	return path + argv[1], nil
}
