/*
Copyright (c) 2022 Red Hat, Inc.

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

package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWETokenHeader struct {
	Algorithm   string `json:"alg"`
	Encryption  string `json:"enc"`
	ContentType string `json:"cty,omitempty"`
}

func IsEncryptedToken(textToken string) bool {
	parts := strings.Split(textToken, ".")
	if len(parts) != 5 {
		return false
	}
	encoded := fmt.Sprintf("%s==", parts[0])
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return false
	}
	header := new(JWETokenHeader)
	err = json.Unmarshal(decoded, header)
	if err != nil {
		return false
	}
	if header.Encryption != "" && header.ContentType == "JWT" {
		return true
	}
	return false
}

func ParseToken(textToken string) (token *jwt.Token, err error) {
	parser := new(jwt.Parser)
	token, _, err = parser.ParseUnverified(textToken, jwt.MapClaims{})
	if err != nil {
		return
	}
	return token, nil
}

// tokenUsable checks if the given token is usable.
func tokenUsable(token string, margin time.Duration) (usable bool, err error) {
	parsed, err := ParseToken(token)
	if err != nil {
		return
	}
	expires, left, err := tokenExpiration(parsed)
	if err != nil {
		return
	}
	if !expires {
		usable = true
		return
	}
	if left >= margin {
		usable = true
		return
	}
	return
}

// tokenExpiration determines if the given token expires, and the time that remains till it expires.
func tokenExpiration(token *jwt.Token) (expires bool, left time.Duration, err error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("expected map claims bug got %T", claims)
		return
	}
	claim, ok := claims["exp"]
	if !ok {
		return
	}
	exp, ok := claim.(float64)
	if !ok {
		err = fmt.Errorf("expected floating point 'exp' but got %T", claim)
		return
	}
	if exp == 0 {
		return
	}
	expires = true
	left = time.Until(time.Unix(int64(exp), 0))
	return
}

// TokenType extracts the value of the `typ` claim. It returns the value as a string, or the empty
// string if there is no such claim.
func TokenType(token *jwt.Token) (typ string, err error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		err = fmt.Errorf("expected map claims but got %T", claims)
		return
	}
	claim, ok := claims["typ"]
	if !ok {
		return
	}
	value, ok := claim.(string)
	if !ok {
		err = fmt.Errorf("expected string 'typ' but got %T", claim)
		return
	}
	typ = value
	return
}
