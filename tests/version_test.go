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

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint

	"github.com/openshift-online/ocm-cli/pkg/info"
)

var _ = Describe("Version", func() {
	It("Prints the version", func() {
		// Create a context:
		ctx := context.Background()

		// Create the command:
		cmd := NewCommand().Args("version")
		defer cmd.Close()

		// Run the command:
		cmd.Run(ctx)

		// Check the result:
		Expect(cmd.OutString()).To(Equal(info.Version + "\n"))
		Expect(cmd.ErrString()).To(BeEmpty())
		Expect(cmd.ExitCode()).To(BeZero())
	})
})
