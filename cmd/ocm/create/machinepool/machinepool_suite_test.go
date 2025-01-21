package machinepool_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMachinepool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machinepool Suite")
}
