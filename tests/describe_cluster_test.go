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

	. "github.com/onsi/ginkgo/v2"    // nolint
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
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "ClusterList",
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
			Expect(result.ErrString()).To(ContainSubstring(
				"There are no subscriptions or clusters with identifier or name 'nonexist'",
			))
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
							"cluster_id": "111"
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

		It("Describe a cluster with channel field", func() {
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
							"id": "222",
							"kind": "Subscription",
							"href": "/api/accounts_mgmt/v1/subscriptions/222",
							"display_name": "test-cluster-with-channel",
							"plan": {
							  "id": "OSD",
							  "kind": "Plan",
							  "href": "/api/accounts_mgmt/v1/plans/OSD",
							  "type": "OSD"
							},
							"creator": {
								"id": "222",
								"kind": "Account",
								"href": "/api/accounts_mgmt/v1/accounts/222"
							  },
							"status": "Active",
							"cluster_id": "222"
						  }
						]
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "Cluster",
						"id": "222",
						"href": "/api/clusters_mgmt/v1/clusters/222",
						"name": "test-with-channel",
						"external_id": "77e5d48c-7afd-475f-9236-e862071f899f",
						"infra_id": "test-channel-abc",
						"creation_timestamp": "2026-03-10T10:00:00Z",
						"cloud_provider": {
						  "kind": "CloudProviderLink",
						  "id": "aws",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws"
						},
						"openshift_version": "4.16.1",
						"channel": "stable-4.16",
						"subscription": {
							"kind": "SubscriptionLink",
							"id": "222",
							"href": "/api/accounts_mgmt/v1/subscriptions/222"
						},
						"region": {
						  "kind": "CloudRegionLink",
						  "id": "us-east-1",
						  "href": "/api/clusters_mgmt/v1/cloud_providers/aws/regions/us-east-1"
						},
						"console": {
						  "url": "https://console-openshift-console.apps.test-channel.example.org"
						},
						"api": {
						  "url": "https://api.test-channel.example.org:6443",
						  "listening": "external"
						},
						"nodes": {
						  "master": 3,
						  "infra": 2,
						  "compute": 2
						},
						"state": "ready",
						"flavour": {
						  "kind": "FlavourLink",
						  "id": "osd-4",
						  "href": "/api/clusters_mgmt/v1/flavours/osd-4"
						},
						"aws": {
						  "private_link": false
						},
						"multi_az": true,
						"managed": true,
						"ccs": {
						  "enabled": true
						},
						"version": {
						  "kind": "Version",
						  "id": "openshift-v4.16.1",
						  "href": "/api/clusters_mgmt/v1/versions/openshift-v4.16.1",
						  "channel_group": "stable"
						},
						"product": {
						  "kind": "ProductLink",
						  "id": "osd",
						  "href": "/api/clusters_mgmt/v1/products/osd"
						},
						"status": {
						  "state": "ready"
						},
						"network": {
						  "type": "OVNKubernetes"
						}
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"id": "222",
						"kind": "Subscription",
						"href": "/api/accounts_mgmt/v1/subscriptions/222",
						"display_name": "test-cluster-with-channel",
						"creator": {
						  "id": "222",
						  "kind": "Account",
						  "href": "/api/accounts_mgmt/v1/accounts/222"
						},
						"managed": true,
						"status": "Active"
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"id": "222",
						"kind": "Account",
						"href": "/api/accounts_mgmt/v1/accounts/222",
						"username": "testuser",
						"email": "testuser@example.com",
						"organization": {
						  "id": "222",
						  "kind": "Organization",
						  "href": "/api/accounts_mgmt/v1/organizations/222",
						  "name": "Test Org With Channel"
						}
					  }`,
				),
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "ProvisionShard",
						"id": "222",
						"href": "/api/clusters_mgmt/v1/provision_shards/222",
						"hive_config": {
						  "server": "https://api.shard2.example.com:6443"
						}
					  }`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"describe", "cluster", "test-with-channel",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())

			// Verify Channel Group is displayed with correct value
			Expect(result.OutString()).To(MatchRegexp(`(?m)^\s*Channel Group:\s+stable\s*$`))

			// Verify Channel is displayed with correct value
			Expect(result.OutString()).To(MatchRegexp(`(?m)^\s*Channel:\s+stable-4\.16\s*$`))
		})

		It("Describe a cluster with multiple matching subscriptions", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "SubscriptionList",
						"page": 1,
						"size": 1,
						"total": 2,
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
							"cluster_id": "111"
						  }
						]
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
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring(
				"There are 2 subscriptions with cluster identifier or name 'test'",
			))
		})
	})
})
