package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"    // nolint
	. "github.com/onsi/gomega"       // nolint
	. "github.com/onsi/gomega/ghttp" // nolint

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

var _ = Describe("List users", func() {
	var ctx context.Context

	var ssoServer *Server
	var apiServer *Server
	var config string

	var subscriptionInfo string = `{
		"items": [
			{
				"kind": "Subscription",
				"cluster_id": "my-cluster",
				"id": "subsID"
			}
		]
	}`

	var clusterInfo string = `{
		"kind": "ClusterList",
		"total": 1,
		"items": [
			{
				"kind": "Cluster",
				"id": "my-cluster",
				"subscription": { "id": "subsID" },
				"state": "ready"
			}
		]
	}`

	var groupsInfo string = `{
		"kind":"GroupList",
		"href":"/api/clusters_mgmt/v1/clusters/my-cluster/groups",
		"page":1,
		"size":2,
		"total":2,
		"items": [
			{
				"kind":"Group",
				"id":"dedicated-admins",
				"users": {
					"kind":"UserList",
					"items": [
						{
							"kind":"User",
							"id":"ddddddddddddddddddddddddddddddddddddddddddd"
						},
						{
							"kind":"User",
							"id":"ssssss"
						}
					]
				}
			},
			{
				"kind":"Group",
				"id":"cluster-admins",
				"users": {
					"kind":"UserList",
					"items": [
						{
							"kind":"User",
							"id":"ddddddddddddddddddddddddddddddddddddddddddd"
						},
						{
							"kind":"User",
							"id":"ssssssssss"
						}
					]
				}
			}
		]
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

	It("Prints GROUP and USER columns aligned correctly", func() {
		// Prepare the server:
		apiServer.AppendHandlers(
			RespondWithJSON(http.StatusOK, subscriptionInfo),
			RespondWithJSON(http.StatusOK, clusterInfo),
			RespondWithJSON(http.StatusOK, groupsInfo),
		)

		result := NewCommand().
			ConfigString(config).
			Args("list", "users", "--cluster", "my-cluster").
			Run(ctx)

		lines := result.OutLines()
		header := lines[0]

		Expect(result.ExitCode()).To(BeZero())
		Expect(lines).To(HaveLen(5))
		Expect(header).To(MatchRegexp(`^GROUP\s+USER$`))

		// Skip header and read content
		userHeaderCol := strings.Index(header, "USER")
		for i := 1; i < len(lines); i++ {
			line := lines[i]

			// Split into logical columns to get the actual username value
			fields := strings.Fields(line)
			Expect(fields).To(HaveLen(2))

			userCol := strings.Index(line, fields[1])
			Expect(userCol).To(Equal(userHeaderCol), fmt.Sprintf("USER column in line %d is not aligned with header", i))
		}
	})

})
