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

var _ = Describe("Token", func() {
	var ctx context.Context
	var cmd *CommandRunner

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		cmd.Close()
	})

	When("Logged in", func() {
		var accessToken string
		var refreshToken string

		BeforeEach(func() {
			// Create the tokens:
			accessToken = MakeTokenString("Bearer", 10*time.Minute)
			refreshToken = MakeTokenString("Refresh", 10*time.Hour)

			// Create the command:
			cmd = NewCommand().
				Config(
					`{
						"refresh_token": "{{ .refreshToken }}",
						"access_token": "{{ .accessToken }}"
					}`,
					"accessToken", accessToken,
					"refreshToken", refreshToken,
				).
				Arg("token")

		})

		It("Displays current access token", func() {
			cmd.Run(ctx)
			Expect(cmd.OutString()).To(Equal(accessToken + "\n"))
			Expect(cmd.ErrString()).To(BeEmpty())
			Expect(cmd.ExitCode()).To(BeZero())
		})

		It("Displays current refresh token", func() {
			cmd.Arg("--refresh").Run(ctx)
			Expect(cmd.OutString()).To(Equal(refreshToken + "\n"))
			Expect(cmd.ErrString()).To(BeEmpty())
			Expect(cmd.ExitCode()).To(BeZero())
		})
	})

	When("Not logged in", func() {
		BeforeEach(func() {
			// Create the command:
			cmd = NewCommand().Arg("token")
		})

		It("Fails", func() {
			cmd.Run(ctx)
			Expect(cmd.OutString()).To(BeEmpty())
			Expect(cmd.ErrString()).To(ContainSubstring("Not logged in"))
			Expect(cmd.ExitCode()).ToNot(BeZero())
		})
	})
})
