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
	path := preParsePath
	var err error

	switch preParsePath {

	// List resources:
	case "accounts", "accts":
		path = "/api/accounts_mgmt/v1/accounts"
	case "subscriptions", "subs":
		path = "/api/accounts_mgmt/v1/subscriptions"
	case "organizations", "orgs":
		path = "/api/accounts_mgmt/v1/organizations"
	case "clusters":
		path = "/api/clusters_mgmt/v1/clusters"

	// Individual resources:
	case "account", "acct":
		path, err = expandResourceWithID("/api/accounts_mgmt/v1/accounts/", argv)
	case "subscription", "sub":
		path, err = expandResourceWithID("/api/accounts_mgmt/v1/subscriptions/", argv)
	case "organization", "org":
		path, err = expandResourceWithID("/api/accounts_mgmt/v1/organizations/", argv)
	case "cluster":
		path, err = expandResourceWithID("/api/clusters_mgmt/v1/clusters/", argv)
	}

	if err != nil {
		return "", err
	}

	return path, nil
}

func expandResourceWithID(path string, argv []string) (string, error) {
	if len(argv) != 2 {
		return "", fmt.Errorf("Resource requires an ID, but got none")
	}
	return path + argv[1], nil
}
