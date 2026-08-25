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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("behavior for `ocm gcp wif-config update`", func() {
	Context("Flag validation", func() {
		var ctx = context.Background()

		When("Using --versions flag with auto mode", func() {
			It("Fails with an error requiring manual mode", func() {
				result := NewCommand().
					Args(
						"gcp", "update", "wif-config",
						"test-wif",
						"--mode", "auto",
						"--versions", "v4.20,v4.21",
					).
					Run(ctx)

				Expect(result.ExitCode()).NotTo(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--versions flag requires --mode=manual"))
			})
		})

		When("Using --versions flag without --output-dir", func() {
			It("Fails with an error requiring output-dir", func() {
				result := NewCommand().
					Args(
						"gcp", "update", "wif-config",
						"test-wif",
						"--mode", "manual",
						"--versions", "v4.20,v4.21",
					).
					Run(ctx)

				Expect(result.ExitCode()).NotTo(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--mode=manual requires --output-dir"))
			})
		})

		When("Using both --version and --versions flags", func() {
			It("Fails with an error about mutually exclusive flags", func() {
				result := NewCommand().
					Args(
						"gcp", "update", "wif-config",
						"test-wif",
						"--mode", "manual",
						"--output-dir", "foobar",
						"--version", "v4.20",
						"--versions", "v4.21,v4.22",
					).
					Run(ctx)

				Expect(result.ExitCode()).NotTo(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("Cannot specify both --version and --versions"))
			})
		})

		When("Using manual mode without --output-dir", func() {
			It("Fails with an error requiring output-dir", func() {
				result := NewCommand().
					Args(
						"gcp", "update", "wif-config",
						"test-wif",
						"--mode", "manual",
					).
					Run(ctx)

				Expect(result.ExitCode()).NotTo(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--mode=manual requires --output-dir"))
			})
		})

		When("Using --versions with empty string", func() {
			It("Fails with an error about empty value", func() {
				result := NewCommand().
					Args(
						"gcp", "update", "wif-config",
						"test-wif",
						"--mode", "manual",
						"--versions", "",
					).
					Run(ctx)

				Expect(result.ExitCode()).NotTo(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--versions cannot be empty or whitespace-only"))
			})
		})

		When("Using --versions with whitespace-only string", func() {
			It("Fails with an error about empty value", func() {
				result := NewCommand().
					Args(
						"gcp", "update", "wif-config",
						"test-wif",
						"--mode", "manual",
						"--versions", "   ",
					).
					Run(ctx)

				Expect(result.ExitCode()).NotTo(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--versions cannot be empty or whitespace-only"))
			})
		})
	})
})
