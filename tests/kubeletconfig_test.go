/*
Copyright (c) 2026 Red Hat

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
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

const (
	kubeletConfigClusterID   = "my-cluster"
	kubeletConfigClusterName = "mycluster"
	kubeletConfigSubsID      = "subsID"
)

var kubeletConfigSubscriptionList = fmt.Sprintf(`{
	"items": [{
		"kind": "Subscription",
		"cluster_id": %q,
		"id": %q
	}]
}`, kubeletConfigClusterID, kubeletConfigSubsID)

// currentAccountResponse returns a minimal /api/accounts_mgmt/v1/current_account response
// with the given orgID embedded.
func currentAccountResponse(orgID string) string {
	return fmt.Sprintf(`{
		"kind": "Account",
		"id": "account-1",
		"organization": {
			"kind": "Organization",
			"id": %q
		}
	}`, orgID)
}

// capabilitiesListResponse returns a /api/accounts_mgmt/v1/capabilities response.
// Set bypassEnabled=true to simulate the org having the bypass capability.
func capabilitiesListResponse(bypassEnabled bool) string {
	value := "false"
	if bypassEnabled {
		value = "true"
	}
	return fmt.Sprintf(`{
		"kind": "CapabilityList",
		"page": 1,
		"size": 1,
		"total": 1,
		"items": [{
			"name": "capability.organization.bypass_pids_limits",
			"value": %q,
			"inherited": false
		}]
	}`, value)
}

var kubeletConfigReadyCluster = fmt.Sprintf(`{
	"kind": "ClusterList",
	"total": 1,
	"items": [{
		"kind": "Cluster",
		"id": %q,
		"name": %q,
		"subscription": {"id": %q},
		"state": "ready"
	}]
}`, kubeletConfigClusterID, kubeletConfigClusterName, kubeletConfigSubsID)

var kubeletConfigNotReadyCluster = fmt.Sprintf(`{
	"kind": "ClusterList",
	"total": 1,
	"items": [{
		"kind": "Cluster",
		"id": %q,
		"name": %q,
		"subscription": {"id": %q},
		"state": "installing"
	}]
}`, kubeletConfigClusterID, kubeletConfigClusterName, kubeletConfigSubsID)

var existingKubeletConfig = `{
	"kind": "KubeletConfig",
	"id": "kc-id-1",
	"name": "default",
	"href": "/api/clusters_mgmt/v1/clusters/my-cluster/kubelet_config",
	"pod_pids_limit": 5000
}`

// setupKubeletConfigServers creates the ssoServer and apiServer and performs login,
// returning the servers and the resulting config string.
func setupKubeletConfigServers(ctx context.Context) (*Server, *Server, string) {
	ssoServer := MakeTCPServer()
	apiServer := MakeTCPServer()

	accessToken := MakeTokenString("Bearer", 15*time.Minute)
	ssoServer.AppendHandlers(RespondWithAccessToken(accessToken))

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
	return ssoServer, apiServer, result.ConfigString()
}

var _ = Describe("Describe kubeletconfig", Ordered, func() {
	var ctx context.Context
	var ssoServer *Server
	var apiServer *Server
	var config string

	BeforeEach(func() {
		ctx = context.Background()
		ssoServer, apiServer, config = setupKubeletConfigServers(ctx)
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
	})

	When("no --cluster flag is provided", func() {
		It("fails with a missing flag error", func() {
			result := NewCommand().
				Args("describe", "kubeletconfig").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("cluster"))
		})
	})

	When("the cluster exists and has a kubeletconfig", func() {
		It("displays the kubeletconfig details", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("describe", "kubeletconfig", "--cluster", kubeletConfigClusterName).
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("kc-id-1"))
			Expect(result.OutString()).To(ContainSubstring("default"))
			Expect(result.OutString()).To(ContainSubstring("5000"))
		})

		It("outputs raw JSON with --json flag", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("describe", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--json").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring(`"kind": "KubeletConfig"`))
			Expect(result.OutString()).To(ContainSubstring(`"id": "kc-id-1"`))
		})
	})

	When("the cluster has no kubeletconfig", func() {
		It("fails with a not found message", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
			)

			result := NewCommand().
				ConfigString(config).
				Args("describe", "kubeletconfig", "--cluster", kubeletConfigClusterName).
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("ocm create kubeletconfig"))
		})
	})
})

var _ = Describe("Create kubeletconfig", Ordered, func() {
	var ctx context.Context
	var ssoServer *Server
	var apiServer *Server
	var config string

	BeforeEach(func() {
		ctx = context.Background()
		ssoServer, apiServer, config = setupKubeletConfigServers(ctx)
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
	})

	When("no --cluster flag is provided", func() {
		It("fails with a missing flag error", func() {
			result := NewCommand().
				Args("create", "kubeletconfig", "--pod-pids-limit", "5000").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("cluster"))
		})
	})

	When("no --pod-pids-limit flag is provided", func() {
		It("fails with a missing flag error", func() {
			result := NewCommand().
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName).
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("pod-pids-limit"))
		})
	})

	When("--pod-pids-limit is below the minimum", func() {
		It("fails with a validation error", func() {
			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "100").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("4096"))
		})
	})

	When("--pod-pids-limit is above the maximum and org has no bypass capability", func() {
		It("fails with a validation error", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, currentAccountResponse("org-1")),
				RespondWithJSON(http.StatusOK, capabilitiesListResponse(false)),
			)
			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "99999").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("16384"))
		})
	})

	When("--pod-pids-limit is above the standard max and org has bypass capability", func() {
		It("creates successfully with the elevated limit", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, currentAccountResponse("org-1")),
				RespondWithJSON(http.StatusOK, capabilitiesListResponse(true)),
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
				RespondWithJSON(http.StatusCreated, existingKubeletConfig),
			)
			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "50000").
				InString("y\n").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("Successfully created"))
		})
	})

	When("--pod-pids-limit exceeds the absolute maximum even with bypass capability", func() {
		It("fails with a validation error referencing the hard cap", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, currentAccountResponse("org-1")),
				RespondWithJSON(http.StatusOK, capabilitiesListResponse(true)),
			)
			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "3694304").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("3694303"))
		})
	})

	When("the cluster is not ready", func() {
		It("fails with a cluster not ready error", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigNotReadyCluster),
			)

			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("not yet ready"))
		})
	})

	When("a kubeletconfig already exists for the cluster", func() {
		It("fails and directs user to edit", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("ocm edit kubeletconfig"))
		})
	})

	When("no kubeletconfig exists and user confirms reboot", func() {
		It("creates the kubeletconfig successfully", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
				RespondWithJSON(http.StatusCreated, existingKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000").
				InString("y\n").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("Successfully created"))
		})
	})

	When("no kubeletconfig exists and user declines reboot", func() {
		It("aborts without creating", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
			)

			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000").
				InString("n\n").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).ToNot(ContainSubstring("Successfully created"))
		})
	})

	When("no kubeletconfig exists and --yes flag is provided", func() {
		It("creates without prompting for confirmation", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
				RespondWithJSON(http.StatusCreated, existingKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("create", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000", "--yes").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("Successfully created"))
		})
	})
})

var _ = Describe("Edit kubeletconfig", Ordered, func() {
	var ctx context.Context
	var ssoServer *Server
	var apiServer *Server
	var config string

	BeforeEach(func() {
		ctx = context.Background()
		ssoServer, apiServer, config = setupKubeletConfigServers(ctx)
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
	})

	When("no --cluster flag is provided", func() {
		It("fails with a missing flag error", func() {
			result := NewCommand().
				Args("edit", "kubeletconfig", "--pod-pids-limit", "5000").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("cluster"))
		})
	})

	When("no --pod-pids-limit flag is provided", func() {
		It("fails with a missing flag error", func() {
			result := NewCommand().
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName).
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("pod-pids-limit"))
		})
	})

	When("--pod-pids-limit is below the minimum", func() {
		It("fails with a validation error", func() {
			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "100").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("4096"))
		})
	})

	When("--pod-pids-limit is above the maximum and org has no bypass capability", func() {
		It("fails with a validation error", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, currentAccountResponse("org-1")),
				RespondWithJSON(http.StatusOK, capabilitiesListResponse(false)),
			)
			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "99999").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("16384"))
		})
	})

	When("--pod-pids-limit is above the standard max and org has bypass capability", func() {
		It("updates successfully with the elevated limit", func() {
			updatedKubeletConfig := `{
				"kind": "KubeletConfig",
				"id": "kc-id-1",
				"href": "/api/clusters_mgmt/v1/clusters/my-cluster/kubelet_config",
				"pod_pids_limit": 50000
			}`
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, currentAccountResponse("org-1")),
				RespondWithJSON(http.StatusOK, capabilitiesListResponse(true)),
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
				RespondWithJSON(http.StatusOK, updatedKubeletConfig),
			)
			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "50000").
				InString("y\n").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("Successfully updated"))
		})
	})

	When("the cluster is not ready", func() {
		It("fails with a cluster not ready error", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigNotReadyCluster),
			)

			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("not yet ready"))
		})
	})

	When("no kubeletconfig exists for the cluster", func() {
		It("fails and directs user to create", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusNotFound, `{"kind":"Error","reason":"not found"}`),
			)

			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "5000").
				Run(ctx)
			Expect(result.ExitCode()).ToNot(BeZero())
			Expect(result.ErrString()).To(ContainSubstring("ocm create kubeletconfig"))
		})
	})

	When("a kubeletconfig exists and user confirms reboot", func() {
		It("updates the kubeletconfig successfully", func() {
			updatedKubeletConfig := `{
				"kind": "KubeletConfig",
				"id": "kc-id-1",
				"href": "/api/clusters_mgmt/v1/clusters/my-cluster/kubelet_config",
				"pod_pids_limit": 8000
			}`
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
				RespondWithJSON(http.StatusOK, updatedKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "8000").
				InString("y\n").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("Successfully updated"))
		})
	})

	When("a kubeletconfig exists and user declines reboot", func() {
		It("aborts without updating", func() {
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "8000").
				InString("n\n").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).ToNot(ContainSubstring("Successfully updated"))
		})
	})

	When("a kubeletconfig exists and --yes flag is provided", func() {
		It("updates without prompting for confirmation", func() {
			updatedKubeletConfig := `{
				"kind": "KubeletConfig",
				"id": "kc-id-1",
				"href": "/api/clusters_mgmt/v1/clusters/my-cluster/kubelet_config",
				"pod_pids_limit": 8000
			}`
			apiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, kubeletConfigSubscriptionList),
				RespondWithJSON(http.StatusOK, kubeletConfigReadyCluster),
				RespondWithJSON(http.StatusOK, existingKubeletConfig),
				RespondWithJSON(http.StatusOK, updatedKubeletConfig),
			)

			result := NewCommand().
				ConfigString(config).
				Args("edit", "kubeletconfig", "--cluster", kubeletConfigClusterName, "--pod-pids-limit", "8000", "--yes").
				Run(ctx)
			Expect(result.ExitCode()).To(BeZero())
			Expect(result.OutString()).To(ContainSubstring("Successfully updated"))
		})
	})
})
