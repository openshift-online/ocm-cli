package tests

import (
	"time"

	. "github.com/openshift-online/ocm-sdk-go/testing" // nolint
)

func InitiateAuthCodeMock(clientID string) (string, error) {
	accessToken := MakeTokenString("Bearer", 15*time.Minute)
	return accessToken, nil
}
