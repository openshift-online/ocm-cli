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
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo" // nolint
	. "github.com/onsi/gomega" // nolint
)

var _ = Describe("Plugin list", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("Writes the list of plugins", func() {
		// Create a temporary directory for the plugins:
		tmp, err := ioutil.TempDir("", "ocm-test-*.d")
		Expect(err).ToNot(HaveOccurred())
		defer func() {
			err = os.RemoveAll(tmp)
			Expect(err).ToNot(HaveOccurred())
		}()

		// Create a collection of empty scripts that will be used as plugins:
		names := []string{
			"my-plugin",
			"your-plugin",
		}
		for _, name := range names {
			path := filepath.Join(tmp, "ocm-"+name)
			file, err := os.OpenFile(path, os.O_CREATE, 0700)
			Expect(err).ToNot(HaveOccurred())
			err = file.Close()
			Expect(err).ToNot(HaveOccurred())
		}

		// Run the command replacing the `PATH` environment variable with the temporary
		// directory for plugins, so that it will not accidentally find other plugins that
		// may be available in the machine where the tests run.
		result := NewCommand().
			Env("PATH", tmp).
			Args("plugin", "list").
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ErrString()).To(BeEmpty())
		lines := result.OutLines()
		Expect(lines).To(HaveLen(3))
		Expect(lines[0]).To(MatchRegexp(
			`^\s*NAME\s+PATH\s*$`,
		))
		Expect(lines[1]).To(MatchRegexp(
			`^\s*ocm-my-plugin\s+%s\s*$`, tmp,
		))
		Expect(lines[2]).To(MatchRegexp(
			`^\s*ocm-your-plugin\s+%s\s*$`, tmp,
		))
	})
})
