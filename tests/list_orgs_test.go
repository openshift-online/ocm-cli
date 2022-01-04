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

var _ = Describe("List orgs", func() {
	var ctx context.Context

	BeforeEach(func() {
		// Create a context:
		ctx = context.Background()
	})

	When("Config file doesn't exist", func() {
		It("Fails", func() {
			getResult := NewCommand().
				Args(
					"list", "orgs",
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
					"list", "orgs",
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

		It("Writes the organizations returned by the server", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{
						"kind": "OrganizationList",
						"page": 1,
						"size": 2,
						"total": 2,
						"items": [
							{
								"kind": "Organization",
								"id": "123",
								"href": "/api/accounts_mgmt/v1/organizations/123",
								"name": "my_org"
							},
							{
								"kind": "Organization",
								"id": "456",
								"href": "/api/accounts_mgmt/v1/organizations/456",
								"name": "your_org"
							}
						]
					}`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args("list", "orgs").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			lines := result.OutLines()
			Expect(lines).To(HaveLen(3))
			Expect(lines[0]).To(MatchRegexp(
				`^\s*ID\s+NAME\s*$`,
			))
			Expect(lines[1]).To(MatchRegexp(
				`^\s*123\s+my_org\s*$`,
			))
			Expect(lines[2]).To(MatchRegexp(
				`^\s*456\s+your_org\s*$`,
			))
		})
	})
})
