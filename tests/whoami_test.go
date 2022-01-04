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

	"github.com/golang-jwt/jwt/v4"

	. "github.com/onsi/ginkgo/v2"                      // nolint
	. "github.com/onsi/gomega"                         // nolint
	. "github.com/onsi/gomega/ghttp"                   // nolint
	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Whoami", func() {
	var ctx context.Context

	BeforeEach(func() {
		// Create a context:
		ctx = context.Background()
	})

	When("Offline user session not found", func() {
		var ssoServer *Server
		var apiServer *Server

		BeforeEach(func() {
			// Create the servers:
			ssoServer = MakeTCPServer()
			apiServer = MakeTCPServer()

			// Prepare the server:
			ssoServer.AppendHandlers(
				RespondWithJSON(
					http.StatusBadRequest,
					`{
						"error": "invalid_grant",
						"error_description": "Offline user session not found"
					}`,
				),
			)
		})

		AfterEach(func() {
			// Close the servers:
			ssoServer.Close()
			apiServer.Close()
		})

		It("Writes user friendly message", func() {
			// Create a valid offline token:
			tokenObject := MakeTokenObject(
				jwt.MapClaims{
					"typ": "Offline",
					"exp": nil,
				},
			)
			tokenString := tokenObject.Raw

			// Prepare the SSO server so that it will respond saying that the offline
			// session doesn't exist. This is something that happens some times in
			// `sso.redhat.com` when the servers are restarted.
			ssoServer.AppendHandlers(
				RespondWithJSON(
					http.StatusBadRequest, `{
						"error_code": "invalid_grant",
						"error_message": "Offline user session not found"
					}`,
				),
			)

			// Run the command:
			whoamiResult := NewCommand().
				ConfigString(
					`{
						"client_id": "cloud-services",
						"insecure": true,
						"refresh_token": "{{ .Token }}",
						"scopes": [
							"openid"
						],
						"token_url": "{{ .TokenURL }}",
						"url": "{{ .URL }}"
					}`,
					"Token", tokenString,
					"TokenURL", ssoServer.URL(),
					"URL", apiServer.URL(),
				).
				Args("whoami").
				Run(ctx)
			Expect(whoamiResult.ExitCode()).ToNot(BeZero())
			Expect(whoamiResult.ErrString()).To(Equal(
				"Offline access token is no longer valid. Go to " +
					"https://console.redhat.com/openshift/token to get a new " +
					"one and then use the 'ocm login --token=...' command to " +
					"log in with that new token.\n",
			))
		})
	})
})
