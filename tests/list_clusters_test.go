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

var _ = Describe("List clusters", func() {
	var ctx context.Context

	BeforeEach(func() {
		// Create a context:
		ctx = context.Background()
	})

	When("Config file doesn't exist", func() {
		It("Fails", func() {
			getResult := NewCommand().
				Args(
					"list", "clusters",
				).Run(ctx)
			Expect(getResult.ExitCode()).ToNot(BeZero())
			Expect(getResult.ErrString()).To(ContainSubstring("Not logged in"))
		})
	})

	When("Config file doesn't contain valid credentials", func() {
		It("Fails", func() {
			getResult := NewCommand().
				ConfigString(`{}`).
				Args(
					"list", "clusters",
				).Run(ctx)
			Expect(getResult.ExitCode()).ToNot(BeZero())
			Expect(getResult.ErrString()).To(ContainSubstring("Not logged in"))
		})
	})

	When("Config file contains valid credentials", func() {
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

		It("Writes the clusters returned by the server", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "ClusterList",
						"page": 1,
						"size": 2,
						"total": 2,
						"items": [
							{
								"kind": "Cluster",
								"id": "123",
								"href": "/api/clusters_mgmt/v1/clusters/123",
								"name": "my_cluster",
								"api": {
									"url": "http://api.my-cluster.com"
								},
								"openshift_version": "4.7",
								"product": {
									"kind": "ProductLink",
									"id": "osd",
									"href": "/api/clusters_mgmt/v1/products/osd"
								},
								"cloud_provider": {
									"kind": "CloudProviderLink",
									"id": "aws",
									"href": "/api/clusters_mgmt/v1/cloud_providers/aws"
								},
								"region": {
									"kind": "CloudRegion",
									"id": "us-east-1",
									"href": "/api/clusters_mgmt/v1/cloud_providers/aws/us-east-1"
								},
								"state": "ready"
							},
							{
								"kind": "Cluster",
								"id": "456",
								"href": "/api/clusters_mgmt/v1/clusters/456",
								"name": "your_cluster",
								"api": {
									"url": "http://api.your-cluster.com"
								},
								"openshift_version": "4.8",
								"product": {
									"kind": "ProductLink",
									"id": "ocp",
									"href": "/api/clusters_mgmt/v1/products/ocp"
								},
								"cloud_provider": {
									"kind": "CloudProviderLink",
									"id": "gcp",
									"href": "/api/clusters_mgmt/v1/cloud_providers/gcp"
								},
								"region": {
									"kind": "CloudRegion",
									"id": "us-west1",
									"href": "/api/clusters_mgmt/v1/cloud_providers/gcp/us-west1"
								},
								"state": "installing"
							}
						]
					}`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args("list", "clusters").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			lines := result.OutLines()
			Expect(lines).To(HaveLen(3))
			Expect(lines[0]).To(MatchRegexp(
				`^\s*ID\s+NAME\s+API URL\s+OPENSHIFT_VERSION\s+PRODUCT ID\s+CLOUD_PROVIDER\s+REGION ID\s+STATE\s*$`,
			))
			Expect(lines[1]).To(MatchRegexp(
				`^\s*123\s+my_cluster\s+http://api.my-cluster.com\s+4\.7\s+osd\s+aws\s+us-east-1\s+ready\s*$`,
			))
			Expect(lines[2]).To(MatchRegexp(
				`^\s*456\s+your_cluster\s+http://api.your-cluster.com\s+4\.8\s+ocp\s+gcp\s+us-west1\s+installing\s*$`,
			))
		})

		It("Doesn't trim `external_id` column", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "ClusterList",
						"page": 1,
						"size": 1,
						"total": 1,
						"items": [
							{
								"kind": "Cluster",
								"id": "123",
								"external_id": "e30bac0b-b337-47d7-a378-2c302b4c868a",
								"name": "my_cluster"
							}
						]
					}`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"list", "clusters",
					"--columns", "id,external_id,name",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			lines := result.OutLines()
			Expect(lines).To(HaveLen(2))
			Expect(lines[0]).To(MatchRegexp(
				`^\s*ID\s+EXTERNAL ID\s+NAME\s*$`,
			))
			Expect(lines[1]).To(MatchRegexp(
				`^\s*123\s+e30bac0b-b337-47d7-a378-2c302b4c868a\s+my_cluster\s*$`,
			))
		})
	})
})
