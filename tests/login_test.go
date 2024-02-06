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
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Login", func() {
	var ctx context.Context
	var ssoServer *Server

	BeforeEach(func() {
		// Create the context:
		ctx = context.Background()

		// Create the server:
		ssoServer = MakeTCPServer()
	})

	AfterEach(func() {
		// Close the server:
		ssoServer.Close()
	})

	When("Using offline token", func() {
		It("Creates the configuration file", func() {
			// Create the tokens:
			accessToken := MakeTokenString("Bearer", 15*time.Minute)

			// Run the command:
			result := NewCommand().
				Args(
					"login",
					"--token", accessToken,
					"--token-url", ssoServer.URL(),
				).
				Run(ctx)

			// Check the content of the configuration file:
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ConfigFile()).ToNot(BeEmpty())
			Expect(result.ConfigString()).To(MatchJSONTemplate(
				`{
					"url": "{{ .url }}",
					"token_url": "{{ .tokenURL }}",
					"client_id": "{{ .clientID }}",
					"scopes": [
						{{ range $i, $scope := .scopes }}
							{{ if gt $i 0 }},{{ end }}
							"{{ $scope }}"
						{{ end }}
					],
					"access_token": "{{ .accessToken }}",
					"auth_method": "token"
				}`,
				"url", sdk.DefaultURL,
				"tokenURL", ssoServer.URL(),
				"clientID", sdk.DefaultClientID,
				"scopes", sdk.DefaultScopes,
				"accessToken", accessToken,
			))
		})
	})

	When("Using client credentials grant", func() {
		It("Creates the configuration file", func() {
			// Create the token:
			accessToken := MakeTokenString("Bearer", 15*time.Minute)

			// Prepare the server:
			ssoServer.AppendHandlers(
				RespondWithAccessToken(accessToken),
			)

			// Run the command:
			result := NewCommand().
				Args(
					"login",
					"--client-id", "my-client",
					"--client-secret", "my-secret",
					"--token-url", ssoServer.URL(),
				).
				Run(ctx)

			// Check the content of the configuration file:
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ConfigFile()).ToNot(BeEmpty())
			Expect(result.ConfigString()).To(MatchJSONTemplate(
				`{
					"url": "{{ .url }}",
					"token_url": "{{ .tokenURL }}",
					"client_id": "my-client",
					"client_secret": "my-secret",
					"scopes": [
						{{ range $i, $scope := .scopes }}
							{{ if gt $i 0 }},{{ end }}
							"{{ $scope }}"
						{{ end }}
					],
					"access_token": "{{ .accessToken }}",
					"auth_method": "client-credentials"
				}`,
				"url", sdk.DefaultURL,
				"tokenURL", ssoServer.URL(),
				"scopes", sdk.DefaultScopes,
				"accessToken", accessToken,
			))
		})
	})

	When("Using password grant", func() {
		It("Creates the configuration file", func() {
			// Create the token:
			accessToken := MakeTokenString("Bearer", 15*time.Minute)
			refreshToken := MakeTokenString("Refresh", 10*time.Hour)

			// Prepare the server:
			ssoServer.AppendHandlers(
				RespondWithAccessAndRefreshTokens(accessToken, refreshToken),
			)

			// Run the command:
			result := NewCommand().
				Args(
					"login",
					"--user", "my-user",
					"--password", "my-password",
					"--token-url", ssoServer.URL(),
				).
				Run(ctx)

			// Check the content of the configuration file. Note that currently the CLI
			// doesn't save the user and password to the configuration file and it
			// generates a warning in the standard error output.
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("deprecated"))
			Expect(result.ConfigFile()).ToNot(BeEmpty())
			Expect(result.ConfigString()).To(MatchJSONTemplate(
				`{
					"url": "{{ .url }}",
					"token_url": "{{ .tokenURL }}",
					"client_id": "{{ .clientID }}",
					"scopes": [
						{{ range $i, $scope := .scopes }}
							{{ if gt $i 0 }},{{ end }}
							"{{ $scope }}"
						{{ end }}
					],
					"access_token": "{{ .accessToken }}",
					"refresh_token": "{{ .refreshToken }}",
					"auth_method": "password"
				}`,
				"url", sdk.DefaultURL,
				"tokenURL", ssoServer.URL(),
				"clientID", sdk.DefaultClientID,
				"scopes", sdk.DefaultScopes,
				"accessToken", accessToken,
				"refreshToken", refreshToken,
			))
		})
	})
})
