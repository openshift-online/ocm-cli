package provider

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func getVPCs(client *cmv1.Client, provider string, ccs cluster.CCS,
	region string) (cloudVPCList []*cmv1.CloudVPC, err error) {
	if ccs.Enabled && provider == "aws" {

		cloudProviderData, err := cmv1.NewCloudProviderData().
			AWS(cmv1.NewAWS().AccessKeyID(ccs.AWS.AccessKeyID).SecretAccessKey(ccs.AWS.SecretAccessKey)).
			Region(cmv1.NewCloudRegion().ID(region)).
			Build()
		if err != nil {
			return nil, fmt.Errorf("Failed to build AWS cloud provider data: %v", err)
		}

		response, err := client.AWSInquiries().Vpcs().Search().
			Page(1).
			Size(-1).
			Body(cloudProviderData).
			Send()
		if err != nil {
			return nil, err
		}
		return response.Items().Slice(), err
	}
	return cloudVPCList, nil
}

func GetSubnetworks(client *cmv1.Client, provider string, ccs cluster.CCS,
	region string) (subnetworkList []*cmv1.Subnetwork, err error) {
	if ccs.Enabled && provider == "aws" {
		cloudVPCs, err := getVPCs(client, provider, ccs, region)
		if err != nil {
			return nil, err
		}

		for _, vpc := range cloudVPCs {
			subnetworkList = append(subnetworkList, vpc.AWSSubnets()...)
		}
	}
	return subnetworkList, nil
}
