package gcp_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGcp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gcp Suite")
}
