/*
Copyright (c) 2023 Red Hat, Inc.

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

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("List machine pools", Ordered, func() {
	var ctx context.Context

	var ssoServer *Server
	var apiServer *Server
	var config string

	var subscriptionInfo string = `{
		"items": [
		  {
			"kind":"Subscription",
			"cluster_id":"my-cluster",
			"id":"subsID"
		  }]
	}`

	var clustersInfo string = `{
		"kind": "ClusterList",
		"total": 1,
		"items": [
			{
			"kind":"Cluster",
			"id":"my-cluster",
			"subscription": {"id":"subsID"},
			"state":"ready"
			}]
	  }`

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

	It("Able to list machine pools information in a cluster with no duplicates", func() {

		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, subscriptionInfo),
			RespondWithJSON(http.StatusOK, clustersInfo),
			RespondWithJSON(http.StatusOK, `{
				"kind": "MachinePoolList",
				"total": 2,
				"items": [
				  {
					"kind": "MachinePool",
					"id": "worker",
					"replicas": 4,
					"instance_type": "m5.xlarge",
					"availability_zones": [
					  "us-west-2a"
					]
				  },
				  {
					"kind": "MachinePool",
					"id": "worker1",
					"replicas": 2,
					"instance_type": "m5.2xlarge",
					"availability_zones": [
					  "us-west-2a"
					]
				  }
				]
			  }`),
		)

		// Run the command:
		result := NewCommand().
			ConfigString(config).
			Args(
				"list", "machinepools",
				"--cluster", "my-cluster",
			).Run(ctx)

		Expect(result.ExitCode()).To(BeZero())
		lines := result.OutLines()
		// The heading and 2 machinepool record information
		Expect(lines).To(HaveLen(3))
		Expect(lines[0]).To(MatchRegexp(
			`^ID\s+AUTOSCALING\s+REPLICAS\s+INSTANCE TYPE\s+LABELS\s+TAINTS\s+AVAILABILITY ZONES$`,
		))
		Expect(lines[1]).To(MatchRegexp(
			`^worker\s+No\s+4\s+m5.xlarge\s+us-west-2a$`,
		))
		Expect(lines[2]).To(MatchRegexp(
			`^worker1\s+No\s+2\s+m5.2xlarge\s+us-west-2a$`,
		))
	})

	It("Able to list information on machine pool named as default", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, subscriptionInfo),
			RespondWithJSON(http.StatusOK, clustersInfo),
			RespondWithJSON(http.StatusOK, `{
				"kind": "MachinePoolList",
				"total": 1,
				"items": [
				  {
					"kind": "MachinePool",
					"id": "default",
					"replicas": 2,
					"instance_type": "m5.xlarge",
					"availability_zones": [
					  "us-west-2a"
					]
				  }
				]
			  }`),
		)

		// Run the command:
		result := NewCommand().
			ConfigString(config).
			Args(
				"list", "machinepools",
				"--cluster", "my-cluster",
			).Run(ctx)

		Expect(result.ExitCode()).To(BeZero())
		lines := result.OutLines()
		// The heading and 1 machinepool record information
		Expect(lines).To(HaveLen(2))
		Expect(lines[0]).To(MatchRegexp(
			`^ID\s+AUTOSCALING\s+REPLICAS\s+INSTANCE TYPE\s+LABELS\s+TAINTS\s+AVAILABILITY ZONES$`,
		))
		Expect(lines[1]).To(MatchRegexp(
			`^default\s+No\s+2\s+m5.xlarge\s+us-west-2a$`,
		))
	})

	It("Fail on invalid cluster key", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, subscriptionInfo),
		)

		// Run the command:
		result := NewCommand().
			ConfigString(config).
			Args(
				"list", "machinepools",
				"--cluster", "invalid!cluster",
			).Run(ctx)

		Expect(result.ExitCode()).ToNot(BeZero())
		Expect(result.ErrString()).To(ContainSubstring("Cluster name, identifier or external identifier"))
		Expect(result.ErrString()).To(ContainSubstring("isn't valid"))
	})

	It("Fail on non-existing cluster", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, subscriptionInfo),
			RespondWith(http.StatusNotFound, `{}`),
		)

		// Run the command:
		result := NewCommand().
			ConfigString(config).
			Args(
				"list", "machinepools",
				"--cluster", "my-cluster",
			).Run(ctx)

		Expect(result.ExitCode()).ToNot(BeZero())
		Expect(result.ErrString()).To(ContainSubstring("Failed to get cluster"))
	})

	It("Fail when cluster is not in ready state", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, subscriptionInfo),
			RespondWithJSON(http.StatusOK, `{
				"kind": "ClusterList",
				"total": 1,
				"items": [
					{
					"kind":"Cluster",
					"id":"my-cluster",
					"subscription": {"id":"subsID"},
					"state":"waiting"
					}]
			  }`),
		)

		// Run the command:
		result := NewCommand().
			ConfigString(config).
			Args(
				"list", "machinepools",
				"--cluster", "my-cluster",
			).Run(ctx)

		Expect(result.ExitCode()).ToNot(BeZero())
		Expect(result.ErrString()).To(ContainSubstring("is not yet ready"))
	})
})
