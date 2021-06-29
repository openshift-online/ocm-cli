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

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Logout", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("Doesn't remove configuration file", func() {
		result := NewCommand().
			ConfigString(`{}`).
			Args("logout").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ConfigFile()).ToNot(BeEmpty())
	})

	It("Removes tokens from configuration file", func() {
		// Generate the tokens:
		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		refreshToken := MakeTokenString("Refresh", 10*time.Hour)

		// Run the command:
		result := NewCommand().
			ConfigString(
				`{
					"access_token": "{{ .accessToken }}",
					"refresh_token": "{{ .refreshToken }}"
				}`,
				"accessToken", accessToken,
				"refreshToken", refreshToken,
			).
			Args("logout").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ConfigString()).To(MatchJSON(`{}`))
	})

	It("Removes client credentials from configuration file", func() {
		result := NewCommand().
			ConfigString(`{
				"client_id": "my_client",
				"client_secret": "my_secret"
			}`).
			Args("logout").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ConfigString()).To(MatchJSON(`{}`))
	})

	It("Removes URLs from configuration file", func() {
		result := NewCommand().
			ConfigString(`{
				"token_url": "http://my-sso.example.com",
				"url": "http://my-api.example.com"
			}`).
			Args("logout").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ConfigString()).To(MatchJSON(`{}`))
	})

	It("Removes scopes from configuration file", func() {
		result := NewCommand().
			ConfigString(`{
				"scopes": [
					"my_scope",
					"your_scope"
				]
			}`).
			Args("logout").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ConfigString()).To(MatchJSON(`{}`))
	})

	It("Removes insecure flag from configuration file", func() {
		result := NewCommand().
			ConfigString(`{
				"insecure": true
			}`).
			Args("logout").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ConfigString()).To(MatchJSON(`{}`))
	})
})
