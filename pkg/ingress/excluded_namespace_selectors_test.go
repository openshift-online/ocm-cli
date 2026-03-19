package ingress_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-online/ocm-cli/pkg/ingress"
)

var _ = Describe("ExcludedNamespaceSelectors", func() {
	type testSpec struct {
		SelectorsStr string
		MapCheck     func(map[string][]string)
		ErrCheck     func(error)
	}
	DescribeTable("ExtractExcludedNamespaceSelectors", func(spec testSpec) {
		subject, err := ingress.ExtractExcludedNamespaceSelectors(spec.SelectorsStr)
		if spec.MapCheck != nil {
			spec.MapCheck(subject)
		}
		if spec.ErrCheck != nil {
			spec.ErrCheck(err)
		}
	},
		Entry("returns empty map for empty string", testSpec{
			SelectorsStr: "",
			MapCheck:     func(selectors map[string][]string) { Expect(selectors).To(Equal(map[string][]string{})) },
			ErrCheck:     func(err error) { Expect(err).ToNot(HaveOccurred()) },
		}),
		Entry("returns error for invalid string", testSpec{
			SelectorsStr: "foobar",
			MapCheck:     func(selectors map[string][]string) { Expect(selectors).To(BeNil()) },
			ErrCheck:     func(err error) { Expect(err).To(HaveOccurred()) },
		}),
		Entry("returns error for invalid entry", testSpec{
			SelectorsStr: "foo=bar,foobar",
			MapCheck:     func(selectors map[string][]string) { Expect(selectors).To(BeNil()) },
			ErrCheck:     func(err error) { Expect(err).To(HaveOccurred()) },
		}),
		Entry("returns error for empty key", testSpec{
			SelectorsStr: "=bar",
			MapCheck:     func(selectors map[string][]string) { Expect(selectors).To(BeNil()) },
			ErrCheck:     func(err error) { Expect(err).To(HaveOccurred()) },
		}),
		Entry("returns error for empty value", testSpec{
			SelectorsStr: "foo=",
			MapCheck:     func(selectors map[string][]string) { Expect(selectors).To(BeNil()) },
			ErrCheck:     func(err error) { Expect(err).To(HaveOccurred()) },
		}),
		Entry("returns error for syntax error", testSpec{
			SelectorsStr: "foo=bar=",
			MapCheck:     func(selectors map[string][]string) { Expect(selectors).To(BeNil()) },
			ErrCheck:     func(err error) { Expect(err).To(HaveOccurred()) },
		}),
		Entry("supports single value", testSpec{
			SelectorsStr: "foo=bar",
			MapCheck: func(selectors map[string][]string) {
				Expect(len(selectors)).To(Equal(1))
				Expect(len(selectors["foo"])).To(Equal(1))
				Expect(selectors["foo"][0]).To(Equal("bar"))
			},
			ErrCheck: func(err error) { Expect(err).ToNot(HaveOccurred()) },
		}),
		Entry("supports multiple values", testSpec{
			SelectorsStr: "a=1,b=2,a=3",
			MapCheck: func(selectors map[string][]string) {
				Expect(len(selectors)).To(Equal(2))
				Expect(len(selectors["a"])).To(Equal(2))
				Expect(len(selectors["b"])).To(Equal(1))
				Expect(selectors["a"]).To(ContainElement("1"))
				Expect(selectors["b"][0]).To(Equal("2"))
				Expect(selectors["a"]).To(ContainElement("3"))
			},
			ErrCheck: func(err error) { Expect(err).ToNot(HaveOccurred()) },
		}),
		Entry("gracefully handles whitespace", testSpec{
			SelectorsStr: "a =  1 ,  b =  2, a=            3",
			MapCheck: func(selectors map[string][]string) {
				Expect(len(selectors)).To(Equal(2))
				Expect(len(selectors["a"])).To(Equal(2))
				Expect(len(selectors["b"])).To(Equal(1))
				Expect(selectors["a"]).To(ContainElement("1"))
				Expect(selectors["b"][0]).To(Equal("2"))
				Expect(selectors["a"]).To(ContainElement("3"))
			},
			ErrCheck: func(err error) { Expect(err).ToNot(HaveOccurred()) },
		}),
	)

})
