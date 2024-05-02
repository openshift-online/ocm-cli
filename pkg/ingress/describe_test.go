package ingress

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Get min width for output", func() {
	It("retrieves the min width", func() {
		minWidth := getMinWidth([]string{"a", "ab", "abc", "def"})
		Expect(minWidth).To(Equal(3))
	})
	When("empty slice", func() {
		It("retrieves the min width as 0", func() {
			minWidth := getMinWidth([]string{})
			Expect(minWidth).To(Equal(0))
		})
	})
})

var _ = Describe("Retrieve map of entries for output", func() {
	It("retrieves map", func() {
		cluster, err := cmv1.NewCluster().ID("123").Build()
		Expect(err).To(BeNil())
		ingress, err := cmv1.NewIngress().
			ID("123").
			Default(true).
			Listening(cmv1.ListeningMethodExternal).
			LoadBalancerType(cmv1.LoadBalancerFlavorNlb).
			RouteWildcardPolicy(cmv1.WildcardPolicyWildcardsAllowed).
			RouteNamespaceOwnershipPolicy(cmv1.NamespaceOwnershipPolicyStrict).
			RouteSelectors(map[string]string{
				"test-route": "test-selector",
			}).
			ExcludedNamespaces("test", "test2").
			ComponentRoutes(map[string]*cmv1.ComponentRouteBuilder{
				string(cmv1.ComponentRouteTypeOauth): v1.NewComponentRoute().
					Hostname("oauth-hostname").TlsSecretRef("oauth-secret"),
			}).
			Build()
		Expect(err).To(BeNil())
		mapOutput := generateEntriesOutput(cluster, ingress)
		Expect(mapOutput).To(HaveLen(10))
	})
})
