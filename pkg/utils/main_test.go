package utils

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEditCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils suite")
}
