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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestURLExpander(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "URL expander")
}

type urlExpanderTest struct {
	params      []string
	expectError bool
	contains    string
}

func urlExpanderTestVerify(test urlExpanderTest) {
	expanded, err := Expand(test.params)
	if !test.expectError {
		Expect(err).ToNot(HaveOccurred())
		Expect(expanded).To(ContainSubstring(test.contains))
	} else {
		Expect(err).To(HaveOccurred())
	}
}

var _ = Describe("Expand", func() {
	DescribeTable(
		"Invalid parameters",
		urlExpanderTestVerify,
		Entry(
			"Invalid parameters - too few",
			urlExpanderTest{
				params:      []string{},
				expectError: true,
			},
		),
		Entry(
			"Invalid parameters - too many",
			urlExpanderTest{
				params:      []string{"foo", "foo", "foo"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"Accounts",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - accts",
			urlExpanderTest{
				params:   []string{"accts"},
				contains: "accounts_mgmt/v1/accounts",
			},
		),
		Entry(
			"Valid list parameters - accounts",
			urlExpanderTest{
				params:   []string{"accounts"},
				contains: "accounts_mgmt/v1/accounts",
			},
		),
		Entry(
			"Valid resource parameters - acct",
			urlExpanderTest{
				params:   []string{"acct", "foo"},
				contains: "accounts_mgmt/v1/accounts",
			},
		),
		Entry(
			"Valid resource parameters - account",
			urlExpanderTest{
				params:   []string{"account", "foo"},
				contains: "accounts_mgmt/v1/accounts",
			},
		),
		Entry(
			"Invalid resource parameters - acct",
			urlExpanderTest{
				params:      []string{"acct"},
				expectError: true,
			},
		),
		Entry(
			"Invalid resource parameters - account",
			urlExpanderTest{
				params:      []string{"account"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"Subscriptions",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - subs",
			urlExpanderTest{
				params:   []string{"subs"},
				contains: "accounts_mgmt/v1/subscriptions",
			},
		),
		Entry(
			"Valid list parameters - subscriptions",
			urlExpanderTest{
				params:   []string{"subscriptions"},
				contains: "accounts_mgmt/v1/subscriptions",
			},
		),
		Entry(
			"Valid resource parameters - sub",
			urlExpanderTest{
				params:   []string{"sub", "foo"},
				contains: "accounts_mgmt/v1/subscriptions",
			},
		),
		Entry(
			"Valid resource parameters - subscription",
			urlExpanderTest{
				params:   []string{"subscription", "foo"},
				contains: "accounts_mgmt/v1/subscriptions",
			},
		),
		Entry(
			"Invalid resource parameters - sub",
			urlExpanderTest{
				params:      []string{"subscription"},
				expectError: true,
			},
		),
		Entry(
			"Invalid resource parameters - subscription",
			urlExpanderTest{
				params:      []string{"subscription"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"Organizations",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - orgs",
			urlExpanderTest{
				params:   []string{"orgs"},
				contains: "accounts_mgmt/v1/organizations",
			},
		),
		Entry(
			"Valid list parameters - organizations",
			urlExpanderTest{
				params:   []string{"organizations"},
				contains: "accounts_mgmt/v1/organizations",
			},
		),
		Entry(
			"Valid resource parameters - org",
			urlExpanderTest{
				params:   []string{"org", "foo"},
				contains: "accounts_mgmt/v1/organizations",
			},
		),
		Entry(
			"Valid resource parameters - organization",
			urlExpanderTest{
				params:   []string{"organization", "foo"},
				contains: "accounts_mgmt/v1/organizations",
			},
		),
		Entry(
			"Invalid resource parameters - org",
			urlExpanderTest{
				params:      []string{"org"},
				expectError: true,
			},
		),
		Entry(
			"Invalid resource parameters - organization",
			urlExpanderTest{
				params:      []string{"organization"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"Passthrough",
		urlExpanderTestVerify,
		Entry(
			"Paths w/o an alias are passed through",
			urlExpanderTest{
				params:   []string{"/api/accounts_mgmt/v1/quota"},
				contains: "accounts_mgmt/v1/quota",
			},
		),
	)

	DescribeTable(
		"Clusters",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - clusters",
			urlExpanderTest{
				params:   []string{"clusters"},
				contains: "clusters_mgmt/v1/clusters",
			},
		),
		Entry(
			"Valid resource parameters - cluster",
			urlExpanderTest{
				params:   []string{"cluster", "foo"},
				contains: "clusters_mgmt/v1/clusters",
			},
		),
		Entry(
			"Invalid resource parameters - cluster",
			urlExpanderTest{
				params:      []string{"cluster"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"RoleBindings",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - role_bindings",
			urlExpanderTest{
				params:   []string{"role_bindings"},
				contains: "accounts_mgmt/v1/role_bindings",
			},
		),
		Entry(
			"Valid resource parameters - role_binding",
			urlExpanderTest{
				params:   []string{"role_binding", "foo"},
				contains: "accounts_mgmt/v1/role_bindings",
			},
		),
		Entry(
			"Invalid resource parameters - role_binding",
			urlExpanderTest{
				params:      []string{"role_binding"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"ResourceQuota",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - resource_quota",
			urlExpanderTest{
				params:   []string{"resource_quota"},
				contains: "accounts_mgmt/v1/resource_quota",
			},
		),
		Entry(
			"Invalid resource parameters - resource_quota",
			urlExpanderTest{
				params:   []string{"resource_quota", "foo"},
				contains: "accounts_mgmt/v1/resource_quota",
			},
		),
	)

	DescribeTable(
		"SKUs",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - skus",
			urlExpanderTest{
				params:   []string{"skus"},
				contains: "accounts_mgmt/v1/skus",
			},
		),
		Entry(
			"Valid resource parameters - sku",
			urlExpanderTest{
				params:   []string{"sku", "foo"},
				contains: "accounts_mgmt/v1/skus",
			},
		),
		Entry(
			"Invalid resource parameters - sku",
			urlExpanderTest{
				params:      []string{"sku"},
				expectError: true,
			},
		),
	)

	DescribeTable(
		"Roles",
		urlExpanderTestVerify,
		Entry(
			"Valid list parameters - roles",
			urlExpanderTest{
				params:   []string{"roles"},
				contains: "accounts_mgmt/v1/roles",
			},
		),
		Entry(
			"Valid resource parameters - role",
			urlExpanderTest{
				params:   []string{"role", "foo"},
				contains: "accounts_mgmt/v1/roles",
			},
		),
		Entry(
			"Invalid resource parameters - role",
			urlExpanderTest{
				params:      []string{"role"},
				expectError: true,
			},
		),
	)
})
