package cluster

import (
	"testing"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	slv1 "github.com/openshift-online/ocm-sdk-go/servicelogs/v1"
)

func TestIsWarningSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity slv1.Severity
		want     bool
	}{
		{name: "legacy Warning", severity: slv1.SeverityWarning, want: true},
		{name: "HCC Moderate", severity: slv1.SeverityModerate, want: true},
		{name: "Info", severity: slv1.SeverityInfo, want: false},
		{name: "Debug", severity: slv1.SeverityDebug, want: false},
		{name: "empty", severity: "", want: false},
		{name: "Important", severity: slv1.SeverityImportant, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isWarningSeverity(test.severity); got != test.want {
				t.Errorf("isWarningSeverity(%q) = %v, want %v", test.severity, got, test.want)
			}
		})
	}
}

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
