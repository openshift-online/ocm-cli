/*
Copyright (c) 2018 Red Hat, Inc.

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
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

type urlExpaderTest struct {
	params      []string
	expectError bool
	contains    string
}

var invalidParamsTests = map[string]urlExpaderTest{
	"invalid params -- too few": {
		params:      []string{},
		expectError: true,
	},
	"invalid params -- too many": {
		params:      []string{"foo", "foo", "foo"},
		expectError: true,
	},
}

var accountTests = map[string]urlExpaderTest{
	"valid list params - accts": {
		params:   []string{"accts"},
		contains: "accounts_mgmt/v1/accounts",
	},
	"valid list params - accounts": {
		params:   []string{"accounts"},
		contains: "accounts_mgmt/v1/accounts",
	},
	"valid resource params - acct": {
		params:   []string{"acct", "foo"},
		contains: "accounts_mgmt/v1/accounts",
	},
	"valid resource params - account": {
		params:   []string{"account", "foo"},
		contains: "accounts_mgmt/v1/accounts",
	},
	"invalid resource params - acct": {
		params:      []string{"acct"},
		expectError: true,
	},
	"invalid resource params - account": {
		params:      []string{"account"},
		expectError: true,
	},
}

var subscriptionTests = map[string]urlExpaderTest{
	"valid list params - subs": {
		params:   []string{"subs"},
		contains: "accounts_mgmt/v1/subscriptions",
	},
	"valid list params - subscriptions": {
		params:   []string{"subscriptions"},
		contains: "accounts_mgmt/v1/subscriptions",
	},
	"valid resource params - sub": {
		params:   []string{"sub", "foo"},
		contains: "accounts_mgmt/v1/subscriptions",
	},
	"valid resource params - subscription": {
		params:   []string{"subscription", "foo"},
		contains: "accounts_mgmt/v1/subscriptions",
	},
	"invalid resource params - sub": {
		params:      []string{"subscription"},
		expectError: true,
	},
	"invalid resource params - subscription": {
		params:      []string{"subscription"},
		expectError: true,
	},
}

var organizationTests = map[string]urlExpaderTest{
	"valid list params - orgs": {
		params:   []string{"orgs"},
		contains: "accounts_mgmt/v1/organizations",
	},
	"valid list params - organizations": {
		params:   []string{"organizations"},
		contains: "accounts_mgmt/v1/organizations",
	},
	"valid resource params - org": {
		params:   []string{"org", "foo"},
		contains: "accounts_mgmt/v1/organizations",
	},
	"valid resource params - organization": {
		params:   []string{"organization", "foo"},
		contains: "accounts_mgmt/v1/organizations",
	},
	"invalid resource params - org": {
		params:      []string{"org"},
		expectError: true,
	},
	"invalid resource params - organization": {
		params:      []string{"organization"},
		expectError: true,
	},
}

var passthroughTests = map[string]urlExpaderTest{
	"paths w/o an alias are passed through": {
		params:   []string{"/api/accounts_mgmt/v1/quota"},
		contains: "accounts_mgmt/v1/quota",
	},
}

var clusterTests = map[string]urlExpaderTest{
	"valid list params - clusters": {
		params:   []string{"clusters"},
		contains: "clusters_mgmt/v1/clusters",
	},
	"valid resource params - cluster": {
		params:   []string{"cluster", "foo"},
		contains: "clusters_mgmt/v1/clusters",
	},
	"invalid resource params - cluster": {
		params:      []string{"cluster"},
		expectError: true,
	},
}

func runTests(t *testing.T, tests map[string]urlExpaderTest) {
	RegisterTestingT(t)
	for name, test := range tests {
		expanded, err := Expand(test.params)
		if !test.expectError {
			Expect(err).To(BeNil(), "Test '%s' has unexpected error: %s", name, err)
			Expect(strings.Contains(expanded, test.contains)).To(BeTrue(),
				"Test '%s' expected path to contain '%s' but got '%s'", name, test.contains, expanded)
		} else {
			Expect(err).NotTo(BeNil(), "Test '%s' expected error but got none", name)
		}
	}
}

func TestExpand(t *testing.T) {
	runTests(t, invalidParamsTests)
	runTests(t, accountTests)
	runTests(t, subscriptionTests)
	runTests(t, organizationTests)
	runTests(t, clusterTests)
	runTests(t, passthroughTests)
}
