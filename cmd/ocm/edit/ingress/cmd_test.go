package ingress

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse component routes", func() {
	It("Parses input string for component routes", func() {
		componentRouteBuilder, err := parseComponentRoutes(
			//nolint:lll
			"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
		)
		Expect(err).To(BeNil())
		for key, builder := range componentRouteBuilder {
			expectedHostname := fmt.Sprintf("%s-host", key)
			expectedTlsRef := fmt.Sprintf("%s-secret", key)
			componentRoute, err := builder.Build()
			Expect(err).To(BeNil())
			Expect(componentRoute.Hostname()).To(Equal(expectedHostname))
			Expect(componentRoute.TlsSecretRef()).To(Equal(expectedTlsRef))
		}
	})
	Context("Fails to parse input string for component routes", func() {
		It("fails due to invalid component route", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"unknown: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("'unknown' is not a valid component name. Expected include [oauth, console, downloads]"))
		})
		It("fails due to wrong amount of component routes", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("the expected amount of component routes is 3, but 2 have been supplied"))
		})
		It("fails if it can split ':' in more than they key separation", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth:-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("only the name of the component should be followed by ':'"))
		})
		It("fails due to invalid parameter", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: unknown=oauth-host;tlsSecretRef=oauth-secret,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("'unknown' is not a valid parameter for a component route. Expected include [hostname, tlsSecretRef]"))
		})
		It("fails due to wrong amount of parameters", func() {
			_, err := parseComponentRoutes(
				//nolint:lll
				"oauth: hostname=oauth-host,downloads: hostname=downloads-host;tlsSecretRef=downloads-secret,console: hostname=console-host;tlsSecretRef=console-secret",
			)
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal("only 2 parameters are expected for each component"))
		})
	})
})
