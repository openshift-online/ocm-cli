/*
Copyright (c) 2026 Red Hat, Inc.

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
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("GCP Firewall Rules", func() {
	var ctx context.Context
	var ssoServer *Server
	var apiServer *Server
	var config string

	BeforeEach(func() {
		// Create a context:
		ctx = context.Background()

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

	Describe("Create firewall rules", func() {
		When("Not logged in", func() {
			It("Fails with authentication error", func() {
				tempDir, err := os.MkdirTemp("", "ocm-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				result := NewCommand().
					Args(
						"gcp", "create", "firewall-rules",
						"--name", "test-firewall",
						"--wif-config", "wif-123",
						"--project-id", "test-project",
						"--vpc-name", "test-vpc",
						"--machine-cidr", "10.0.0.0/16",
						"--output-file", outputFile,
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("Not logged in"))
			})
		})

		When("Missing required flags", func() {
			It("Fails when name is missing", func() {
				tempDir, err := os.MkdirTemp("", "ocm-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "create", "firewall-rules",
						"--wif-config", "wif-123",
						"--project-id", "test-project",
						"--vpc-name", "test-vpc",
						"--machine-cidr", "10.0.0.0/16",
						"--output-file", outputFile,
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("flag 'name' is required"))
			})

			It("Fails when wif-config is missing", func() {
				tempDir, err := os.MkdirTemp("", "ocm-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "create", "firewall-rules",
						"--name", "test-firewall",
						"--project-id", "test-project",
						"--vpc-name", "test-vpc",
						"--machine-cidr", "10.0.0.0/16",
						"--output-file", outputFile,
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("flag 'wif-config' is required"))
			})

			It("Fails when project-id is missing", func() {
				tempDir, err := os.MkdirTemp("", "ocm-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "create", "firewall-rules",
						"--name", "test-firewall",
						"--wif-config", "wif-123",
						"--vpc-name", "test-vpc",
						"--machine-cidr", "10.0.0.0/16",
						"--output-file", outputFile,
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("flag 'project-id' is required"))
			})

			It("Fails when vpc-name is missing", func() {
				tempDir, err := os.MkdirTemp("", "ocm-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "create", "firewall-rules",
						"--name", "test-firewall",
						"--wif-config", "wif-123",
						"--project-id", "test-project",
						"--machine-cidr", "10.0.0.0/16",
						"--output-file", outputFile,
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("flag 'vpc-name' is required"))
			})

			It("Fails when machine-cidr is missing", func() {
				tempDir, err := os.MkdirTemp("", "ocm-test-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})
				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "create", "firewall-rules",
						"--name", "test-firewall",
						"--wif-config", "wif-123",
						"--project-id", "test-project",
						"--vpc-name", "test-vpc",
						"--output-file", outputFile,
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("flag 'machine-cidr' is required"))
			})
		})

		When("Creating firewall rules successfully", func() {
			It("Creates firewall rules with manual mode", func() {
				// Create a temporary directory for the output file
				tempDir, err := os.MkdirTemp("", "ocm-test-firewall-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})

				outputFile := filepath.Join(tempDir, "create-firewall.sh")

				// Prepare the server to respond to the create request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/gcp/firewall_rules"),
						VerifyJQ(`.name`, "test-firewall"),
						VerifyJQ(`.wif_config.id`, "wif-123"),
						VerifyJQ(`.gcp_network.project_id`, "test-project"),
						VerifyJQ(`.gcp_network.vpc_name`, "test-vpc"),
						VerifyJQ(`.gcp_network.machine_cidr`, "10.0.0.0/16"),
						RespondWithJSON(
							http.StatusCreated,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall",
								"profile": "public",
								"gcp_network": {
									"project_id": "test-project",
									"vpc_name": "test-vpc",
									"machine_cidr": "10.0.0.0/16"
								},
								"wif_config": {
									"id": "wif-123",
									"href": "/api/clusters_mgmt/v1/gcp/wif_configs/wif-123"
								},
								"status": {
									"rules": [
										{
											"name": "test-firewall-ingress",
											"direction": "INGRESS",
											"priority": 1000
										}
									]
								}
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "create", "firewall-rules",
						"--name", "test-firewall",
						"--wif-config", "wif-123",
						"--project-id", "test-project",
						"--vpc-name", "test-vpc",
						"--machine-cidr", "10.0.0.0/16",
						"--mode", "manual",
						"--output-file", outputFile,
					).
					Run(ctx)

				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutString()).To(ContainSubstring("ID:"))
				Expect(result.OutString()).To(ContainSubstring("firewall-123"))
				Expect(result.OutString()).To(ContainSubstring("Name:"))
				Expect(result.OutString()).To(ContainSubstring("test-firewall"))
				Expect(result.OutString()).To(ContainSubstring("Bash script written to"))

				// Verify the script file was created
				_, err = os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())

				// Read and verify script content
				scriptContent, err := os.ReadFile(outputFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(scriptContent)).To(ContainSubstring("#!/bin/bash"))
				Expect(string(scriptContent)).To(ContainSubstring("--quiet"))
			})
		})
	})

	Describe("Delete firewall rules", func() {
		When("Missing firewall rule ID", func() {
			It("Fails when ID is not provided", func() {
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "delete", "firewall-rules",
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("accepts 1 arg(s), received 0"))
			})
		})

		When("Attempting to use auto mode", func() {
			It("Fails with error message", func() {
				// Run the command with auto mode:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "delete", "firewall-rules",
						"firewall-123",
						"--mode", "auto",
						"--output-file", "/tmp/delete.sh",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("Only manual mode is supported"))
			})
		})

		When("Missing output file in manual mode", func() {
			It("Fails with error message", func() {
				// Prepare the server (though it shouldn't be called):
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"name": "test-firewall"
							}`,
						),
					),
				)

				// Run the command without output file:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "delete", "firewall-rules",
						"firewall-123",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--output-file flag is required"))
			})
		})

		When("Deleting firewall rules in manual mode", func() {
			It("Generates a bash script and deletes from OCM backend", func() {
				// Create a temporary directory for the output file
				tempDir, err := os.MkdirTemp("", "ocm-test-firewall-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})

				outputFile := filepath.Join(tempDir, "delete-firewall.sh")

				// Prepare the server to respond to the get request (to fetch firewall details):
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall",
								"profile": "public",
								"gcp_network": {
									"project_id": "test-project",
									"vpc_name": "test-vpc",
									"machine_cidr": "10.0.0.0/16"
								},
								"status": {
									"rules": [
										{
											"name": "test-firewall-rule",
											"direction": "INGRESS",
											"priority": 1000
										}
									]
								}
							}`,
						),
					),
				)

				// Prepare the server to respond to the delete request (to delete from OCM backend):
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodDelete, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusNoContent,
							`{}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "delete", "firewall-rules",
						"firewall-123",
						"--mode", "manual",
						"--output-file", outputFile,
					).
					Run(ctx)

				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutString()).To(ContainSubstring("Bash script written to"))
				Expect(result.OutString()).To(ContainSubstring("Execute the script to delete firewall rules from GCP"))
				Expect(result.OutString()).To(ContainSubstring("deleted from OCM backend successfully"))

				// Verify the script file was created
				_, err = os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())

				// Read and verify script content
				scriptContent, err := os.ReadFile(outputFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(scriptContent)).To(ContainSubstring("#!/bin/bash"))
				Expect(string(scriptContent)).To(ContainSubstring("GCP Firewall Rules Delete Script"))
				Expect(string(scriptContent)).To(ContainSubstring("--quiet"))
			})

			It("Handles non-existent firewall rule", func() {
				// Prepare the server to respond with an error:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/nonexistent"),
						RespondWithJSON(
							http.StatusNotFound,
							`{
								"kind": "Error",
								"id": "404",
								"href": "/api/clusters_mgmt/v1/errors/404",
								"code": "CLUSTERS-MGMT-404",
								"reason": "Firewall rule with id 'nonexistent' not found"
							}`,
						),
					),
				)

				// Create a temporary directory for the output file
				tempDir, err := os.MkdirTemp("", "ocm-test-firewall-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})

				outputFile := filepath.Join(tempDir, "delete-firewall.sh")

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "delete", "firewall-rules",
						"nonexistent",
						"--output-file", outputFile,
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("failed to get GCP firewall rule"))
			})
		})
	})

	Describe("List firewall rules", func() {
		When("Listing firewall rules", func() {
			It("Lists all firewall rules successfully", func() {
				// Prepare the server to respond to the list request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRuleList",
								"page": 1,
								"size": 2,
								"total": 2,
								"items": [
									{
										"kind": "GcpFirewallRule",
										"id": "firewall-123",
										"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
										"name": "test-firewall-1",
										"profile": "public"
									},
									{
										"kind": "GcpFirewallRule",
										"id": "firewall-456",
										"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-456",
										"name": "test-firewall-2",
										"profile": "private"
									}
								]
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "list", "firewall-rules",
					).
					Run(ctx)

				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutString()).To(ContainSubstring("firewall-123"))
				Expect(result.OutString()).To(ContainSubstring("firewall-456"))
			})
		})
	})

	Describe("Update firewall rules", func() {
		When("Missing firewall rule ID", func() {
			It("Fails when ID is not provided", func() {
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "update", "firewall-rules",
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("accepts 1 arg(s), received 0"))
			})
		})

		When("Attempting to use auto mode", func() {
			It("Fails with error message", func() {
				// Run the command with auto mode:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "update", "firewall-rules",
						"firewall-123",
						"--mode", "auto",
						"--output-file", "/tmp/update.sh",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("Only manual mode is supported"))
			})
		})

		When("Missing output file in manual mode", func() {
			It("Fails with error message", func() {
				// Run the command without output file:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "update", "firewall-rules",
						"firewall-123",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("--output-file flag is required"))
			})
		})

		When("Updating firewall rules in manual mode", func() {
			It("Generates an update bash script successfully", func() {
				// Create a temporary directory for the output file
				tempDir, err := os.MkdirTemp("", "ocm-test-firewall-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})

				outputFile := filepath.Join(tempDir, "update-firewall.sh")

				// Prepare the server to respond to the get request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall",
								"profile": "public",
								"gcp_network": {
									"project_id": "test-project",
									"vpc_name": "test-vpc",
									"machine_cidr": "10.0.0.0/16"
								},
								"status": {
									"rules": [
										{
											"name": "test-firewall-ingress",
											"direction": "INGRESS",
											"priority": 1000,
											"network": "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/test-vpc",
											"allowed": [
												{
													"ip_protocol": "tcp",
													"ports": ["22"]
												}
											],
											"source_ranges": ["10.0.0.0/16"],
											"target_service_accounts": ["osd-control-plane@test-project.iam.gserviceaccount.com"]
										}
									]
								}
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "update", "firewall-rules",
						"firewall-123",
						"--output-file", outputFile,
					).
					Run(ctx)

				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutString()).To(ContainSubstring("Bash script written to"))
				Expect(result.OutString()).To(ContainSubstring("update firewall rules in GCP"))

				// Verify the script file was created
				_, err = os.Stat(outputFile)
				Expect(err).ToNot(HaveOccurred())

				// Read and verify script content
				scriptContent, err := os.ReadFile(outputFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(scriptContent)).To(ContainSubstring("#!/bin/bash"))
				Expect(string(scriptContent)).To(ContainSubstring("GCP Firewall Rules Update Script"))
				Expect(string(scriptContent)).To(ContainSubstring("firewall-rules update"))
				Expect(string(scriptContent)).To(ContainSubstring("test-firewall-ingress"))
				Expect(string(scriptContent)).To(ContainSubstring("--priority=1000"))
				Expect(string(scriptContent)).To(ContainSubstring("--rules='tcp:22'"))
				Expect(string(scriptContent)).To(ContainSubstring("--source-ranges='10.0.0.0/16'"))
				Expect(string(scriptContent)).To(ContainSubstring(
					"--target-service-accounts='osd-control-plane@test-project.iam.gserviceaccount.com'"))
				Expect(string(scriptContent)).To(ContainSubstring("--quiet"))
			})

			It("Handles non-existent firewall rule", func() {
				// Prepare the server to respond with an error:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/nonexistent"),
						RespondWithJSON(
							http.StatusNotFound,
							`{
								"kind": "Error",
								"id": "404",
								"href": "/api/clusters_mgmt/v1/errors/404",
								"code": "CLUSTERS-MGMT-404",
								"reason": "Firewall rule with id 'nonexistent' not found"
							}`,
						),
					),
				)

				// Create a temporary directory for the output file
				tempDir, err := os.MkdirTemp("", "ocm-test-firewall-*")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					Expect(os.RemoveAll(tempDir)).To(Succeed())
				})

				outputFile := filepath.Join(tempDir, "update-firewall.sh")

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "update", "firewall-rules",
						"nonexistent",
						"--output-file", outputFile,
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("failed to get GCP firewall rule"))
			})
		})
	})

	Describe("Verify firewall rules", func() {
		When("Missing firewall rule ID", func() {
			It("Fails when ID is not provided", func() {
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "verify", "firewall-rules",
					).
					Run(ctx)
				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("accepts 1 arg(s), received 0"))
			})
		})

		When("Verifying properly configured firewall rules", func() {
			It("Succeeds when all rules are configured correctly", func() {
				// Prepare the server to respond to the get request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall"
							}`,
						),
					),
				)

				// Prepare the server to respond to the status request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123/status"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRulesStatus",
								"state": "ready",
								"description": "All firewall rules are configured correctly",
								"rules": [
									{
										"name": "test-firewall-ingress",
										"exists": true,
										"configured_correctly": true,
										"description": "Rule is properly configured"
									}
								]
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "verify", "firewall-rules",
						"firewall-123",
					).
					Run(ctx)

				Expect(result.ExitCode()).To(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("Firewall rules are properly configured"))
			})
		})

		When("Verifying misconfigured firewall rules", func() {
			It("Fails when rules are misconfigured", func() {
				// Prepare the server to respond to the get request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall"
							}`,
						),
					),
				)

				// Prepare the server to respond to the status request with misconfiguration:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123/status"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRulesStatus",
								"state": "misconfigured",
								"description": "Some firewall rules are misconfigured",
								"rules": [
									{
										"name": "test-firewall-ingress",
										"exists": true,
										"configured_correctly": false,
										"description": "Priority is incorrect: expected 1000, found 2000"
									}
								]
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "verify", "firewall-rules",
						"firewall-123",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("misconfigured"))
			})
		})

		When("Verifying missing firewall rules", func() {
			It("Fails when rules are missing", func() {
				// Prepare the server to respond to the get request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall"
							}`,
						),
					),
				)

				// Prepare the server to respond to the status request with missing rules:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123/status"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRulesStatus",
								"state": "missing",
								"description": "Some firewall rules are missing",
								"rules": [
									{
										"name": "test-firewall-ingress",
										"exists": false,
										"configured_correctly": false,
										"description": "Firewall rule not found in GCP project"
									}
								]
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "verify", "firewall-rules",
						"firewall-123",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("missing"))
			})
		})

		When("Verifying non-existent firewall rule", func() {
			It("Handles non-existent firewall rule", func() {
				// Prepare the server to respond with an error:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/nonexistent"),
						RespondWithJSON(
							http.StatusNotFound,
							`{
								"kind": "Error",
								"id": "404",
								"href": "/api/clusters_mgmt/v1/errors/404",
								"code": "CLUSTERS-MGMT-404",
								"reason": "Firewall rule with id 'nonexistent' not found"
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "verify", "firewall-rules",
						"nonexistent",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
				Expect(result.ErrString()).To(ContainSubstring("failed to get GCP firewall rule"))
			})
		})
	})

	Describe("Describe firewall rules", func() {
		When("Describing a firewall rule", func() {
			It("Shows firewall rule details successfully", func() {
				// Prepare the server to respond to the get request:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123"),
						RespondWithJSON(
							http.StatusOK,
							`{
								"kind": "GcpFirewallRule",
								"id": "firewall-123",
								"href": "/api/clusters_mgmt/v1/gcp/firewall_rules/firewall-123",
								"name": "test-firewall",
								"profile": "public",
								"gcp_network": {
									"project_id": "test-project",
									"vpc_name": "test-vpc",
									"machine_cidr": "10.0.0.0/16"
								},
								"wif_config": {
									"id": "wif-123",
									"href": "/api/clusters_mgmt/v1/gcp/wif_configs/wif-123"
								},
								"status": {
									"rules": [
										{
											"name": "test-firewall-ingress",
											"direction": "INGRESS",
											"priority": 1000
										}
									]
								}
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "describe", "firewall-rules",
						"firewall-123",
					).
					Run(ctx)

				Expect(result.ExitCode()).To(BeZero())
				Expect(result.OutString()).To(ContainSubstring("firewall-123"))
				Expect(result.OutString()).To(ContainSubstring("test-firewall"))
			})

			It("Handles non-existent firewall rule", func() {
				// Prepare the server to respond with an error:
				apiServer.AppendHandlers(
					CombineHandlers(
						VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/gcp/firewall_rules/nonexistent"),
						RespondWithJSON(
							http.StatusNotFound,
							`{
								"kind": "Error",
								"id": "404",
								"href": "/api/clusters_mgmt/v1/errors/404",
								"code": "CLUSTERS-MGMT-404",
								"reason": "Firewall rule with id 'nonexistent' not found"
							}`,
						),
					),
				)

				// Run the command:
				result := NewCommand().
					ConfigString(config).
					Args(
						"gcp", "describe", "firewall-rules",
						"nonexistent",
					).
					Run(ctx)

				Expect(result.ExitCode()).ToNot(BeZero())
			})
		})
	})
})
