//go:build darwin
// +build darwin

/*
Copyright (c) 2024 Red Hat, Inc.

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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	"github.com/openshift-online/ocm-cli/cmd/ocm/login"
	"github.com/openshift-online/ocm-cli/pkg/properties"
	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Keychain Keyring", func() {
	var ctx context.Context
	var ssoServer *Server

	BeforeEach(func() {
		// Create the context
		ctx = context.Background()

		// Create the server
		ssoServer = MakeTCPServer()

		// Remove the configuration from the keyring
		result := NewCommand().
			Args(
				"logout",
			).
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ErrString()).To(BeEmpty())
	})

	AfterEach(func() {
		// Close the server
		ssoServer.Close()
	})

	When("Using OCM_KEYRING", func() {
		AfterEach(func() {
			// reset keyring
			os.Setenv(properties.KeyringEnvKey, "")
		})

		It("Stores/Removes configuration in Keychain", func() {
			// Create the token
			accessToken := MakeTokenString("Bearer", 15*time.Minute)

			// Prepare the server
			ssoServer.AppendHandlers(
				RespondWithAccessToken(accessToken),
			)

			os.Setenv(properties.KeyringEnvKey, "keychain")

			// Run login
			result := NewCommand().
				Args(
					"login",
					"--client-id", "my-client",
					"--client-secret", "my-secret",
					"--token-url", ssoServer.URL(),
				).
				Run(ctx)

			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			// Verify no config file data exists
			Expect(result.ConfigFile()).To(BeEmpty())
			Expect(result.ConfigString()).To(BeEmpty())

			// Check the content of the keyring
			result = NewCommand().
				Args(
					"config",
					"get",
					"access_token",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.OutLines()[0]).To(ContainSubstring(accessToken))

			// Remove the configuration from the keyring
			result = NewCommand().
				Args(
					"logout",
				).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.OutLines()).To(BeEmpty())

			// Ensure the keyring is empty
			result = NewCommand().
				Args(
					"config",
					"get",
					"access_token",
				).
				Run(ctx)
			Expect(result.ErrString()).To(BeEmpty())
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutLines()[0]).To(BeEmpty())
		})

		Context("Using auth code", func() {
			It("Stores/Removes configuration in Keychain", func() {
				os.Setenv(properties.KeyringEnvKey, "keychain")

				login.InitiateAuthCode = InitiateAuthCodeMock

				cmd := login.Cmd
				cmd.SetArgs([]string{"--use-auth-code"})
				err := cmd.Execute()
				Expect(err).NotTo(HaveOccurred())

				// Check the content of the keyring
				result := NewCommand().
					Args(
						"config",
						"get",
						"access_token",
					).
					Run(ctx)
				Expect(result.ExitCode()).To(BeZero())
				Expect(result.ErrString()).To(BeEmpty())
				Expect(result.OutLines()[0]).NotTo(BeEmpty())

				// Remove the configuration from the keyring
				result = NewCommand().
					Args(
						"logout",
					).
					Run(ctx)
				Expect(result.ExitCode()).To(BeZero())
				Expect(result.ErrString()).To(BeEmpty())
				Expect(result.OutLines()).To(BeEmpty())

				// Ensure the keyring is empty
				result = NewCommand().
					Args(
						"config",
						"get",
						"access_token",
					).
					Run(ctx)
				Expect(result.ErrString()).To(BeEmpty())
				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutLines()[0]).To(BeEmpty())
			})
		})
	})
})
