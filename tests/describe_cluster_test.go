/*
Copyright (c) 2021 Red Hat, Inc.

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

package tests

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"       // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Describe clusters", func() {
	var ctx context.Context

	BeforeEach(func() {
		// Create a context:
		ctx = context.Background()

	})

	When("Describe with no cluster", func() {
		It("Fails", func() {
			getResult := NewCommand().
				Args(
					"describe", "cluster",
				).Run(ctx)
			Expect(getResult.ExitCode()).ToNot(BeZero())
			Expect(getResult.ErrString()).To(ContainSubstring("identifier or external identifier is required"))
		})
	})
	When("Describe clusters", func() {
		var ssoServer *Server
		var apiServer *Server
		var config string

		BeforeEach(func() {
			// Create the servers:
			ssoServer = MakeTCPServer()
			apiServer = MakeTCPServer()

			// Create the token:
			accessToken := MakeTokenString("Bearer", 15*time.Minute)

			// Prepare the server:
			ssoServer.AppendHandlers(
				RespondWithAccessToken(accessToken),
			)

			// Login:
			result := NewCommand().
				Args(
					"login",
					"--client-id", "my-client",
					"--client-secret", "my-secret",
					"--token-url", ssoServer.URL(),
					"--url", apiServer.URL(),
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			config = result.ConfigString()
		})

		AfterEach(func() {
			// Close the servers:
			ssoServer.Close()
			apiServer.Close()
		})

		It("Describe a non-exist cluster", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "SubscriptionList",
						"page": 1,
						"size": 0,
						"total": 0,
						"items": []
					  }`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args("describe", "cluster", "nonexist").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("There is no cluster with identifier or name"))
		})

		It("Describe an exist cluster", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "SubscriptionList",
						"page": 1,
						"size": 1,
						"total": 1,
						"items": [
						  {
							"id": "111",
							"kind": "Subscription",
							"href": "/api/accounts_mgmt/v1/subscriptions/111",
							"plan": {
							  "id": "OSD",
							  "kind": "Plan",
							  "href": "/api/accounts_mgmt/v1/plans/OSD",
							  "type": "OSD"
							},
							"creator": {
								"id": "111",
								"kind": "Account",
								"href": "/api/accounts_mgmt/v1/accounts/111"
							  },
							"status": "Active",
							"status": "Active",
							"cluster_id": "111",
							"status": "Active"
						  }
						]
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "Cluster",
						"id": "111",
						"href": "/api/clusters_mgmt/v1/clusters/111",
						"name": "test",
						"external_id": "66e5d48c-6afd-475f-9236-e862071f899f",
						"infra_id": "test-wtjvx",
						"display_name": "test",
						"creation_timestamp": "2021-07-05T03:27:18.264654Z",
						"activity_timestamp": "2021-07-13T06:55:32Z",
						"expiration_timestamp": "2021-07-18T21:34:46Z",
						"cloud_provider": {
						  "kind": "CloudProviderLink",
						  "id": "aws",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						},
						"openshift_version": "4.7.18",
						"subscription": {
							"kind": "SubscriptionLink",
							"id": "111",
							"href": "/api/accounts_mgmt/v1/subscriptions/111"
						},
						"region": {
						  "kind": "CloudRegionLink",
						  "id": "ap-southeast-2",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/ap-southeast-2"
						},
						"console": {
						  "url": "https://console-openshift-console.apps.test.example.org"
						},
						"api": {
						  "url": "https://api.-test.example.org:6443",
						  "listening": "external"
						},
						"nodes": {
						  "master": 3,
						  "infra": 2,
						  "compute": 2,
						  "availability_zones": [
							"ap-southeast-2a"
						  ]
						},
						"state": "ready",
						"flavour": {
						  "kind": "FlavourLink",
						  "id": "osd-4",
						  "href": "/api/clusters_mgmt/v1/flavours/osd-4"
						},
						"groups": {
						  "kind": "GroupListLink",
						  "href": "/api/clusters_mgmt/v1/clusters/111/groups"
						},
						"aws": {
						  "private_link": false
						},
						
						"multi_az": false,
						"managed": true,
						"ccs": {
						  "enabled": true,
						  "disable_scp_checks": false
						},
						"version": {
						  "kind": "Version",
						  "id": "openshift-v4.7.18",
						  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.7.18",
						  "raw_id": "",
						  "channel_group": "stable",
						  "available_upgrades": [
							"4.7.19"
						  ]
						},
						"product": {
						  "kind": "ProductLink",
						  "id": "osd",
						  "href": "/api/clusters_mgmt/v1/products/osd"
						},
						"status": {
						  "state": "ready",
						  "dns_ready": true,
						  "provision_error_message": "",
						  "provision_error_code": "",
						  "configuration_mode": "full"
						}
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"id": "111",
						"kind": "Subscription",
						"href": "/api/accounts_mgmt/v1/subscriptions/111",
						"support_level": "Premium",
						"display_name": "test",
						"creator": {
						  "id": "111",
						  "kind": "Account",
						  "href": "/api/accounts_mgmt/v1/accounts/111"
						},
						"managed": true,
						"status": "Active"
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"id": "111",
						"kind": "Account",
						"href": "/api/accounts_mgmt/v1/accounts/111",
						"first_name": "Test",
						"last_name": "Test",
						"username": "test",
						"email": "test@example.com",
						"created_at": "2021-04-13T05:10:46.277747Z",
						"updated_at": "2021-07-09T05:49:31.562355Z",
						"organization": {
						  "id": "111",
						  "kind": "Organization",
						  "href": "/api/accounts_mgmt/v1/organizations/111",
						  "name": "Example Org"
						}
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "ProvisionShard",
						"id": "111",
						"href": "/api/clusters_mgmt/v1/provision_shards/111",
						"hive_config": {
						  "server": "https://api.shard1.example.com:6443"
						},
						"aws_account_operator_config": {
						  "server": "https://api.shard1.example.com:6443"
						},
						"gcp_project_operator_config": {
						  "server": "https://api.shard1.example.com:6443"
						},
						"aws_base_domain": "s1.test.org",
						"gcp_base_domain": "s2.test.org",
						"status": "active"
					  }`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"describe", "cluster", "test",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			lines := result.OutLines()
			Expect(result.ErrString()).To(BeEmpty())
			Expect(lines[1]).To(MatchRegexp(
				`^\s*ID:\s+111\s*$`,
			))

			Expect(result.OutString()).To(ContainSubstring("https://console-openshift-console.apps.test.example.org"))
			Expect(result.OutString()).To(ContainSubstring("https://api.shard1.example.com:6443"))
			Expect(result.OutString()).To(ContainSubstring("Example Org"))
			Expect(result.OutString()).To(ContainSubstring("test@example.com"))

		})
	})
})
