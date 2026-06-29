/*
Copyright (c) 2025 Red Hat, Inc.

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

package opaquetoken

import (
	"os"

	. "github.com/onsi/ginkgo/v2" // nolint
	. "github.com/onsi/gomega"    // nolint
	"github.com/spf13/pflag"

	"github.com/openshift-online/ocm-cli/pkg/properties"
)

var _ = Describe("Enabled", func() {
	BeforeEach(func() {
		enabled = false
		os.Unsetenv(properties.OpaqueTokenEnvKey)
	})

	AfterEach(func() {
		enabled = false
		os.Unsetenv(properties.OpaqueTokenEnvKey)
	})

	It("Returns false by default", func() {
		Expect(Enabled()).To(BeFalse())
	})

	It("Returns true when env var is 'true'", func() {
		os.Setenv(properties.OpaqueTokenEnvKey, "true")
		Expect(Enabled()).To(BeTrue())
	})

	It("Returns true when env var is '1'", func() {
		os.Setenv(properties.OpaqueTokenEnvKey, "1")
		Expect(Enabled()).To(BeTrue())
	})

	It("Returns false when env var is 'false'", func() {
		os.Setenv(properties.OpaqueTokenEnvKey, "false")
		Expect(Enabled()).To(BeFalse())
	})

	It("Returns false when env var is invalid", func() {
		os.Setenv(properties.OpaqueTokenEnvKey, "notabool")
		Expect(Enabled()).To(BeFalse())
	})

	It("Returns true when flag is set", func() {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		AddFlag(fs)
		err := fs.Parse([]string{"--opaque-token"})
		Expect(err).ToNot(HaveOccurred())
		Expect(Enabled()).To(BeTrue())
	})

	It("Returns true when flag is set even if env var is false", func() {
		os.Setenv(properties.OpaqueTokenEnvKey, "false")
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		AddFlag(fs)
		err := fs.Parse([]string{"--opaque-token"})
		Expect(err).ToNot(HaveOccurred())
		Expect(Enabled()).To(BeTrue())
	})
})
