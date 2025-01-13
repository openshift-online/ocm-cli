package utils

import (
	"encoding/json"
	"reflect"

	"github.com/MicahParks/jwkset"
)

// Checks if two string parameters represent equal json web key sets. false is
// returned if the two jwks do not have equivalent values, or if there is an
// error processing the expected fields of either parameter.
func JwksEqual(
	jwksStrA string,
	jwksStrB string,
) bool {
	var jwksA, jwksB jwkset.JWKSMarshal

	if err := json.Unmarshal(json.RawMessage(jwksStrA), &jwksA); err != nil {
		return false
	}
	if err := json.Unmarshal(json.RawMessage(jwksStrB), &jwksB); err != nil {
		return false
	}

	return reflect.DeepEqual(jwksA, jwksB)
}
