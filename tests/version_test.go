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

	. "github.com/onsi/ginkgo/v2" // nolint
	. "github.com/onsi/gomega"    // nolint

	"github.com/openshift-online/ocm-cli/pkg/info"
	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Version", func() {
	It("Prints the version", func() {
		ctx := context.Background()
		result := NewCommand().Args("version").Run(ctx)

		Expect(result.OutString()).To(Equal(info.Version + "\n"))
		Expect(result.ErrString()).To(BeEmpty())
		Expect(result.ExitCode()).To(BeZero())
	})

	It("Lists HCP versions when a product is specified", func() {
		ctx := context.Background()
		ssoServer := MakeTCPServer()
		defer ssoServer.Close()
		apiServer := MakeTCPServer()
		defer apiServer.Close()

		ssoServer.AppendHandlers(RespondWithAccessToken(MakeTokenString("Bearer", 15*time.Minute)))
		loginResult := NewCommand().Args(
			"login", "--client-id", "my-client", "--client-secret", "my-secret",
			"--token-url", ssoServer.URL(), "--url", apiServer.URL(),
		).Run(ctx)
		Expect(loginResult.ExitCode()).To(BeZero())

		var requestedProduct string
		apiServer.AppendHandlers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedProduct = r.URL.Query().Get("product")
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{
				"kind": "VersionList",
				"page": 1,
				"size": 1,
				"total": 1,
				"items": [
					{
						"kind": "Version",
						"id": "openshift-v5.0.1-fast",
						"raw_id": "5.0.1",
						"enabled": true,
						"default": false,
						"channel_group": "fast"
					}
				]
			}`))
			Expect(err).NotTo(HaveOccurred())
		}))

		result := NewCommand().ConfigString(loginResult.ConfigString()).Args(
			"list", "versions", "--channel-group", "fast", "--product", "hcp",
		).Run(ctx)
		Expect(result.ExitCode()).To(BeZero(), result.ErrString())
		Expect(requestedProduct).To(Equal("hcp"))
		Expect(result.OutString()).To(Equal("5.0.1-fast\n"))
	})
})
