package machinepool_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	gomock "go.uber.org/mock/gomock"

	"github.com/openshift-online/ocm-cli/cmd/ocm/create/machinepool"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/cluster"
	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

var _ = Describe("Verification helpers", func() {
	Context("VerifyCluster", func() {
		It("returns an error for clusters that are not ready", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockCluster := ocm.NewMockCluster(ctrl)
			mockCluster.EXPECT().State().MinTimes(1).Return(cmv1.ClusterStateError)
			err := machinepool.VerifyCluster(mockCluster)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("VerifyArguments", Ordered, func() {
		const (
			machinePoolId = "mp-id"
			faultyLabel   = "foobar"
			faultyTaint   = "foobar"
			testAz        = "us-east1b"
		)
		It("returns error if no command line arguments are passed", func() {
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv: []string{},
			}, nil, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing machine pool ID"))
		})
		It("returns an error if a label does not contain '='", func() {
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:   []string{machinePoolId},
				Labels: faultyLabel,
			}, nil, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("label"))
		})
		It("returns an error if a taint does not contain '='", func() {
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:   []string{machinePoolId},
				Taints: faultyTaint,
			}, nil, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("taints"))
		})
		It("returns an error when replicas is set while autoscaling is enabled", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, true, true, true)
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:        []string{machinePoolId},
				Autoscaling: cluster.Autoscaling{Enabled: true},
			}, mockFlagSet, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--replicas"))
		})
		It("returns an error if both min and max replicas are not set", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, false, false, false)
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:        []string{machinePoolId},
				Autoscaling: cluster.Autoscaling{Enabled: true},
			}, mockFlagSet, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Both --min-replicas and --max-replicas"))
		})
		It("returns an error if max replicas is set while autoscaling is disabled", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, false, true, true)
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:        []string{machinePoolId},
				Autoscaling: cluster.Autoscaling{Enabled: false},
			}, mockFlagSet, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--min-replicas and --max-replicas are not allowed"))
		})
		It("returns an error if min replicas is set while autoscaling is disabled", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, true, false, true)
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:        []string{machinePoolId},
				Autoscaling: cluster.Autoscaling{Enabled: false},
			}, mockFlagSet, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--min-replicas and --max-replicas are not allowed"))
		})
		It("returns an error if replicas is not set while autoscaling is disabled", func() {
			ctrl := gomock.NewController(GinkgoT())
			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, false, false, false)
			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:        []string{machinePoolId},
				Autoscaling: cluster.Autoscaling{Enabled: false},
			}, mockFlagSet, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("--replicas is required"))
		})
		It("returns an error if the specified instance type is not a valid option", func() {
			ctrl := gomock.NewController(GinkgoT())

			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, false, false, true)
			mockFlagSet.EXPECT().CheckOneOf("instance-type", gomock.Any()).
				MinTimes(1).Return(fmt.Errorf("no such instance type"))

			mockMachineTypeListGetter := machinepool.NewMockMachineTypeListGetter(ctrl)
			mockMachineTypeListGetter.EXPECT().GetMachineTypeOptions(gomock.Any()).
				MinTimes(1).Return([]arguments.Option{}, nil)

			err := machinepool.VerifyArguments(machinepool.Args{
				Argv: []string{machinePoolId},
			}, mockFlagSet, mockMachineTypeListGetter, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such instance type"))
		})
		It("returns an error if the cluster is not 'GCP' and an AZ is specified", func() {
			ctrl := gomock.NewController(GinkgoT())

			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, false, false, true)
			mockFlagSet.EXPECT().CheckOneOf("instance-type", gomock.Any()).
				MinTimes(1).Return(nil)

			mockMachineTypeListGetter := machinepool.NewMockMachineTypeListGetter(ctrl)
			mockMachineTypeListGetter.EXPECT().GetMachineTypeOptions(gomock.Any()).
				MinTimes(1).Return([]arguments.Option{}, nil)

			mockCluster := ocm.NewMockCluster(ctrl)
			mockCluster.EXPECT().CloudProviderId().MinTimes(1).Return(c.ProviderAWS)

			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:             []string{machinePoolId},
				AvailabilityZone: testAz,
			}, mockFlagSet, mockMachineTypeListGetter, mockCluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("availability-zone"))
		})
		It("returns an error if the cluster is not 'AWS' and additional security groups are specified", func() {
			ctrl := gomock.NewController(GinkgoT())

			mockFlagSet := machinepool.NewMockFlagSet(ctrl)
			mockFlagSet = addChangedFlags(mockFlagSet, false, false, true)
			mockFlagSet.EXPECT().CheckOneOf("instance-type", gomock.Any()).
				MinTimes(1).Return(nil)

			mockMachineTypeListGetter := machinepool.NewMockMachineTypeListGetter(ctrl)
			mockMachineTypeListGetter.EXPECT().GetMachineTypeOptions(gomock.Any()).
				MinTimes(1).Return([]arguments.Option{}, nil)

			mockCluster := ocm.NewMockCluster(ctrl)
			mockCluster.EXPECT().CloudProviderId().MinTimes(1).Return(c.ProviderGCP)

			err := machinepool.VerifyArguments(machinepool.Args{
				Argv:                       []string{machinePoolId},
				AdditionalSecurityGroupIds: []string{"foobar"},
			}, mockFlagSet, mockMachineTypeListGetter, mockCluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("additional-security-group-ids"))
		})
	})
})

func addChangedFlags(
	mockFlagSet *machinepool.MockFlagSet,
	minReplicas bool,
	maxReplicas bool,
	replicas bool,
) *machinepool.MockFlagSet {
	mockFlagSet.EXPECT().Changed("min-replicas").MinTimes(1).Return(minReplicas)
	mockFlagSet.EXPECT().Changed("max-replicas").MinTimes(1).Return(maxReplicas)
	mockFlagSet.EXPECT().Changed("replicas").MinTimes(1).Return(replicas)
	return mockFlagSet
}
