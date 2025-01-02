package utils_test

import (
	"encoding/json"
	"fmt"

	"github.com/MicahParks/jwkset"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-online/ocm-cli/pkg/utils"
)

var _ = Describe("Jwks helpers", func() {
	type equalityTestSpec struct {
		JwksStrA string
		JwksStrB string
		Equal    bool
		ValidA   bool
		ValidB   bool
	}
	DescribeTable("checking equality", func(spec equalityTestSpec) {
		var jwksA, jwksB jwkset.JWKSMarshal
		if spec.ValidA {
			Expect(json.Unmarshal(json.RawMessage(spec.JwksStrA), &jwksA)).ToNot(HaveOccurred(),
				"failed to unmarshall the first jwks parameter")
		}
		if spec.ValidB {
			Expect(json.Unmarshal(json.RawMessage(spec.JwksStrB), &jwksB)).ToNot(HaveOccurred(),
				"failed to unmarshall the first jwks parameter")
		}
		Expect(utils.JwksEqual(spec.JwksStrA, spec.JwksStrB)).To(Equal(spec.Equal))
	},
		Entry("returns true for two equal jwks", equalityTestSpec{
			JwksStrA: generateJwksStr("test1"),
			JwksStrB: generateJwksStr("test1"),
			Equal:    true,
			ValidA:   true,
			ValidB:   true,
		}),
		Entry("returns false for two unqeual jwks", equalityTestSpec{
			JwksStrA: generateJwksStr("test2"),
			JwksStrB: generateJwksStr("foobar"),
			Equal:    false,
			ValidA:   true,
			ValidB:   true,
		}),
		Entry("returns false if the first parameter is invalid", equalityTestSpec{
			JwksStrA: "foobar",
			JwksStrB: generateJwksStr("test3"),
			Equal:    false,
			ValidA:   false,
			ValidB:   true,
		}),
		Entry("returns false if the second parameter is invalid", equalityTestSpec{
			JwksStrA: generateJwksStr("test4"),
			JwksStrB: "foobar",
			Equal:    false,
			ValidA:   true,
			ValidB:   false,
		}),
	)
})

func generateJwksStr(kid string) string {
	//nolint:lll
	return fmt.Sprintf(`{
    "keys": [
        {
            "e": "AQAB",
            "n": "ubDivpwTQ84zcmhCC7Dlun34pv8N-kd44Kx1ohYa3HqAGFrYGVvxAc4bRgfFD1_Rt03uNGFy0lBkZ_Jv5mjGJ97eBXACuU1wiIX_C6gT7gwH9WlmbUnNndWz5CN7mvtspcHWv0E_uM08LJCNkThFe7dQESbWkyS8RO-dfJBUOwZH0N38AUnXOkNKfvFTMQr_-I9YHgaWr7rZxhoPV5viE1aM_H-kaLsgqFbD2VSGC48qpeO4FktnRrvM92mtK8RyqM0w4BnbNAlIk22SIWK0H_2nusdtYHnkPTY1nBk4f-TvcHLA1hEbGKK3eM9IQwWZFtmSlxwPQAD7l_gNREPPtDSuy2q5qHH6Ew3rKFxE2PTF0UNH1oHsgCf6DpRIcrqQ7rSuUAghmFgkqXBneiI-lCFqSHeFYUr2LpyF4LJ5GyWaEIyG54Rv9vpfpJYd6RmTfEferzwgCnm2fmClZQVa_fQMxluzw6UldrDzUF-rOko4klZNeSZita-5IZ3C9zU4ZVWEr4RR4_F3gnYLXYwz7asIFckIpXoPTggB29OpoWJPoYulMMedhZ3A1IjCCx7_Nxgj6qqwrJuyubkURpBsnhxVIsfumn5eNaDy2D8N2cpRxZmnCIJZ2ffEaLj0UNp4M3MAQONRKCaPNx-GbRxit3PvNDCr_LQ9C5fI-zUeUb8",
            "alg": "RS256",
            "kid": "%s",
            "kty": "RSA",
            "use": "sig"
        }
    ]}`, kid)
}
