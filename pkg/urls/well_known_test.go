package urls

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/ocm-cli/pkg/config"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

var _ = Describe("Gateway URL Resolution", func() {

	var nilConfig *config.Config = nil
	var emptyConfig *config.Config = &config.Config{}
	var emptyURLConfig *config.Config = &config.Config{URL: ""}
	var nonEmptyURLConfig *config.Config = &config.Config{URL: "https://api.example.com"}
	validUrlOverrides := []string{
		"https://api.example.com", "http://api.example.com", "http://localhost",
		"http://localhost:8080", "https://localhost:8080/", "unix://my.server.com/tmp/api.socket",
		"unix+https://my.server.com/tmp/api.socket", "h2c://api.example.com",
		"unix+h2c://my.server.com/tmp/api.socket",
	}
	invalidUrlOverrides := []string{
		//nolint:misspell // intentional misspellings
		"productin", "PRod", //alias typo
		"localhost", "192.168.1.1", "api.openshift.com", //ip address/hostname without protocol
	}

	It("Prority 1 - cli arg valid url aliases", func() {
		for alias, url := range OCMURLAliases {
			resolved, err := ResolveGatewayURL(alias, nilConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(url))

			resolved, err = ResolveGatewayURL(alias, emptyConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(url))

			resolved, err = ResolveGatewayURL(alias, emptyURLConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(url))

			resolved, err = ResolveGatewayURL(alias, nonEmptyURLConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(url))
		}
	})

	It("Priority 2 - cli arg valid url", func() {
		for _, urlOverride := range validUrlOverrides {
			resolved, err := ResolveGatewayURL(urlOverride, nilConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(urlOverride))

			resolved, err = ResolveGatewayURL(urlOverride, emptyConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(urlOverride))

			resolved, err = ResolveGatewayURL(urlOverride, emptyURLConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(urlOverride))

			resolved, err = ResolveGatewayURL(urlOverride, nonEmptyURLConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(urlOverride))
		}
	})

	It("Priority 3 - valid config url", func() {
		for _, urlOverride := range validUrlOverrides {
			resolved, err := ResolveGatewayURL("", &config.Config{URL: urlOverride})
			Expect(err).ToNot(HaveOccurred())
			Expect(resolved).To(Equal(urlOverride))
		}
	})

	It("Priority 4 - api.openshift.com", func() {
		resolved, err := ResolveGatewayURL("", nilConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(resolved).To(Equal(sdk.DefaultURL))

		resolved, err = ResolveGatewayURL("", emptyConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(resolved).To(Equal(sdk.DefaultURL))

		resolved, err = ResolveGatewayURL("", emptyURLConfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(resolved).To(Equal(sdk.DefaultURL))
	})

	It("Invalid url alias throws an error", func() {
		for _, urlOverride := range invalidUrlOverrides {
			_, err := ResolveGatewayURL(urlOverride, nilConfig)
			Expect(err).To(HaveOccurred())
		}
	})

	It("Invalid cfg.URL throws an error", func() {
		for _, urlOverride := range invalidUrlOverrides {
			_, err := ResolveGatewayURL("", &config.Config{URL: urlOverride})
			Expect(err).To(HaveOccurred())
		}
	})
})
