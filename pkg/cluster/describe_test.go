package cluster

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"testing"
)

// newTestCluster assembles a *cmv1.Cluster while handling the error to help out with inline test-case generation
func newTestCluster(t *testing.T, cb *cmv1.ClusterBuilder) *cmv1.Cluster {
	cluster, err := cb.Build()
	if err != nil {
		t.Fatalf("failed to build cluster: %s", err)
	}

	return cluster
}

func TestFindHyperShiftMgmtSvcClusters(t *testing.T) {
	tests := []struct {
		name         string
		cluster      *cmv1.Cluster
		expectedMgmt string
		expectedSvc  string
	}{
		{
			name:    "Not HyperShift",
			cluster: newTestCluster(t, cmv1.NewCluster().Hypershift(cmv1.NewHypershift().Enabled(false))),
		},
	}

	for _, test := range tests {
		mgmt, svc := findHyperShiftMgmtSvcClusters(nil, test.cluster)
		if test.expectedMgmt != mgmt {
			t.Errorf("expected %s, got %s", test.expectedMgmt, mgmt)
		}
		if test.expectedSvc != svc {
			t.Errorf("expected %s, got %s", test.expectedSvc, svc)
		}
	}
}
