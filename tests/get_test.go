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

var _ = Describe("Get", func() {
	var ctx context.Context

	BeforeEach(func() {
		// Create a context:
		ctx = context.Background()
	})

	When("Config file doesn't exist", func() {
		It("Fails", func() {
			getResult := NewCommand().
				Args(
					"get", "/api/my_service/v1/my_object",
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
					"get", "/api/my_service/v1/my_object",
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

		It("Writes the JSON returned by the server", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				RespondWithJSON(
					http.StatusOK,
					`{ "my_field": "my_value" }`,
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args("get", "/api/my_service/v1/my_object").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.OutString()).To(MatchJSON(`{ "my_field": "my_value" }`))
		})

		It("Honours the --parameter flag", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				CombineHandlers(
					VerifyFormKV("my_param", "my_value"),
					RespondWithJSON(http.StatusOK, `{}`),
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"get",
					"--parameter", "my_param=my_value",
					"/api/my_service/v1/my_object",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
		})

		It("Honours the -p flag as alias to --parameter", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				CombineHandlers(
					VerifyFormKV("my_param", "my_value"),
					RespondWithJSON(http.StatusOK, `{}`),
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"get",
					"-p", "my_param=my_value",
					"/api/my_service/v1/my_object",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
		})

		It("Honours the --header flag", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				CombineHandlers(
					VerifyHeaderKV("my_header", "my_value"),
					RespondWithJSON(http.StatusOK, `{}`),
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"get",
					"--header", "my_header=my_value",
					"/api/my_service/v1/my_object",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
		})

		It("Indents by default", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				CombineHandlers(
					RespondWithJSON(http.StatusOK, `{
						"my_field": "my_value",
						"your_field": "your_value"
					}`),
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"get",
					"/api/my_service/v1/my_object",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.OutString()).To(Equal(RemoveLeadingTabs(
				`{
				  "my_field": "my_value",
				  "your_field": "your_value"
				}
				`,
			)))
		})

		It("Honours the --single flag", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				CombineHandlers(
					RespondWithJSON(http.StatusOK, `{
						"my_field": "my_value",
						"your_field": "your_value"
					}`),
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"get",
					"--single",
					"/api/my_service/v1/my_object",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.OutString()).To(Equal(RemoveLeadingTabs(
				`{"my_field":"my_value","your_field":"your_value"}
				`,
			)))
		})

		It("Preserves long integers", func() {
			// Prepare the server:
			apiServer.AppendHandlers(
				CombineHandlers(
					RespondWithJSON(http.StatusOK, `{
						"my_field": 340282366920938463463374607431768211455
					}`),
				),
			)

			// Run the command:
			result := NewCommand().
				ConfigString(config).
				Args(
					"get",
					"/api/my_service/v1/my_object",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.OutString()).To(Equal(RemoveLeadingTabs(
				`{
				  "my_field": 340282366920938463463374607431768211455
				}
				`,
			)))
		})
	})
})
