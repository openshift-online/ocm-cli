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
	"time"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("Cluster update", Ordered, func() {
	var ctx context.Context

	var ssoServer *Server
	var apiServer *Server
	var config string

	BeforeEach(func() {
		ctx = context.Background()

		ssoServer = MakeTCPServer()
		apiServer = MakeTCPServer()

		accessToken := MakeTokenString("Bearer", 15*time.Minute)
		ssoServer.AppendHandlers(
			RespondWithAccessToken(accessToken),
		)
		clientSecret := MakeTokenString("Bearer", 15*time.Minute)

		result := NewCommand().
			Args(
				"login",
				"--client-id", "my-client",
				"--client-secret", clientSecret,
				"--token-url", ssoServer.URL(),
				"--url", apiServer.URL(),
			).
			Run(ctx)
		Expect(result.ExitCode()).To(BeZero())
		config = result.ConfigString()
	})

	AfterEach(func() {
		ssoServer.Close()
		apiServer.Close()
	})

	It("creates a manual update policy for an available version", func() {
		apiServer.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/subscriptions"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "SubscriptionList",
					"total": 1,
					"items": [
						{
							"kind": "Subscription",
							"id": "sub-1",
							"cluster_id": "cluster-123",
							"status": "Active"
						}
					]
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-123"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "Cluster",
					"id": "cluster-123",
					"name": "my-cluster",
					"version": {
						"kind": "Version",
						"id": "openshift-v4.22.17-candidate",
						"available_upgrades": [
							"4.22.18",
							"5.0.0-rc.6"
						]
					}
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-123/upgrade_policies"),
				VerifyJSON(`{
					"kind": "UpgradePolicy",
					"schedule_type": "manual",
					"version": "5.0.0-rc.6",
					"next_run": "2099-01-01T00:00:00Z"
				}`),
				RespondWithJSON(http.StatusCreated, `{}`),
			),
		)

		result := NewCommand().
			ConfigString(config).
			Args(
				"cluster", "update", "my-cluster",
				"--version", "5.0.0-rc.6",
				"--next-run", "2099-01-01T00:00:00Z",
			).
			Run(ctx)

		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ErrString()).To(BeEmpty())
		Expect(result.OutString()).To(ContainSubstring(
			"Scheduled update for cluster 'my-cluster' to version '5.0.0-rc.6'",
		))
	})

	It("updates the channel before creating an update policy", func() {
		apiServer.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/subscriptions"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "SubscriptionList",
					"total": 1,
					"items": [
						{
							"kind": "Subscription",
							"id": "sub-1",
							"cluster_id": "cluster-123",
							"status": "Active"
						}
					]
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-123"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "Cluster",
					"id": "cluster-123",
					"name": "my-cluster",
					"channel": "candidate-4.22",
					"version": {
						"kind": "Version",
						"id": "openshift-v4.22.17-candidate",
						"available_upgrades": [
							"4.22.18"
						],
						"available_channels": [
							"candidate-4.22",
							"candidate-5.0"
						]
					}
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodPatch, "/api/clusters_mgmt/v1/clusters/cluster-123"),
				VerifyJSON(`{
					"kind": "Cluster",
					"channel": "candidate-5.0"
				}`),
				RespondWithJSON(http.StatusOK, `{
					"kind": "Cluster",
					"id": "cluster-123",
					"name": "my-cluster",
					"channel": "candidate-5.0",
					"version": {
						"kind": "Version",
						"id": "openshift-v4.22.17-candidate",
						"available_upgrades": [
							"5.0.0-rc.6"
						],
						"available_channels": [
							"candidate-4.22",
							"candidate-5.0"
						]
					}
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-123/upgrade_policies"),
				VerifyJSON(`{
					"kind": "UpgradePolicy",
					"schedule_type": "manual",
					"version": "5.0.0-rc.6",
					"next_run": "2099-01-01T00:00:00Z"
				}`),
				RespondWithJSON(http.StatusCreated, `{}`),
			),
		)

		result := NewCommand().
			ConfigString(config).
			Args(
				"cluster", "update", "my-cluster",
				"--channel", "candidate-5.0",
				"--version", "5.0.0-rc.6",
				"--next-run", "2099-01-01T00:00:00Z",
			).
			Run(ctx)

		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ErrString()).To(BeEmpty())
		Expect(result.OutString()).To(ContainSubstring(
			"Scheduled update for cluster 'my-cluster' to version '5.0.0-rc.6'",
		))
	})

	It("does not patch the cluster when the selected channel is current", func() {
		apiServer.AppendHandlers(
			CombineHandlers(
				VerifyRequest(http.MethodGet, "/api/accounts_mgmt/v1/subscriptions"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "SubscriptionList",
					"total": 1,
					"items": [
						{
							"kind": "Subscription",
							"id": "sub-1",
							"cluster_id": "cluster-123",
							"status": "Active"
						}
					]
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodGet, "/api/clusters_mgmt/v1/clusters/cluster-123"),
				RespondWithJSON(http.StatusOK, `{
					"kind": "Cluster",
					"id": "cluster-123",
					"name": "my-cluster",
					"channel": "candidate-4.22",
					"version": {
						"kind": "Version",
						"id": "openshift-v4.22.17-candidate",
						"available_upgrades": [
							"4.22.18"
						],
						"available_channels": [
							"candidate-4.22",
							"candidate-5.0"
						]
					}
				}`),
			),
			CombineHandlers(
				VerifyRequest(http.MethodPost, "/api/clusters_mgmt/v1/clusters/cluster-123/upgrade_policies"),
				VerifyJSON(`{
					"kind": "UpgradePolicy",
					"schedule_type": "manual",
					"version": "4.22.18",
					"next_run": "2099-01-01T00:00:00Z"
				}`),
				RespondWithJSON(http.StatusCreated, `{}`),
			),
		)

		result := NewCommand().
			ConfigString(config).
			Args(
				"cluster", "update", "my-cluster",
				"--channel", "candidate-4.22",
				"--version", "4.22.18",
				"--next-run", "2099-01-01T00:00:00Z",
			).
			Run(ctx)

		Expect(result.ExitCode()).To(BeZero())
		Expect(result.ErrString()).To(BeEmpty())
	})

	It("rejects a version that is not listed as an available update", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"kind": "SubscriptionList",
				"total": 1,
				"items": [
					{
						"kind": "Subscription",
						"id": "sub-1",
						"cluster_id": "cluster-123",
						"status": "Active"
					}
				]
			}`),
			RespondWithJSON(http.StatusOK, `{
				"kind": "Cluster",
				"id": "cluster-123",
				"name": "my-cluster",
				"version": {
					"kind": "Version",
					"id": "openshift-v4.22.17-candidate",
					"available_upgrades": [
						"4.22.18"
					]
				}
			}`),
		)

		result := NewCommand().
			ConfigString(config).
			Args(
				"cluster", "update", "my-cluster",
				"--version", "5.0.0-rc.6",
				"--next-run", "2099-01-01T00:00:00Z",
			).
			Run(ctx)

		Expect(result.ExitCode()).ToNot(BeZero())
		Expect(result.ErrString()).To(ContainSubstring("A valid --version must be specified"))
	})

	It("reports no available updates when the cluster has no version", func() {
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, `{
				"kind": "SubscriptionList",
				"total": 1,
				"items": [
					{
						"kind": "Subscription",
						"id": "sub-1",
						"cluster_id": "cluster-123",
						"status": "Active"
					}
				]
			}`),
			RespondWithJSON(http.StatusOK, `{
				"kind": "Cluster",
				"id": "cluster-123",
				"name": "my-cluster"
			}`),
		)

		result := NewCommand().
			ConfigString(config).
			Args(
				"cluster", "update", "my-cluster",
				"--next-run", "2099-01-01T00:00:00Z",
			).
			Run(ctx)

		Expect(result.ExitCode()).ToNot(BeZero())
		Expect(result.ErrString()).To(ContainSubstring("There are no available updates for cluster 'my-cluster'"))
	})
})
