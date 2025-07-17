package provider

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func getAWSVPCs(client *cmv1.Client, ccs cluster.CCS,
	region string) (cloudVPCList []*cmv1.CloudVPC, err error) {

	cloudProviderData, err := cmv1.NewCloudProviderData().
		AWS(cmv1.NewAWS().AccessKeyID(ccs.AWS.AccessKeyID).SecretAccessKey(ccs.AWS.SecretAccessKey)).
		Region(cmv1.NewCloudRegion().ID(region)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build AWS cloud provider data: %v", err)
	}

	response, err := ocm.SendTypedAndHandleDeprecation(client.AWSInquiries().Vpcs().Search().
		Page(1).
		Size(-1).
		Body(cloudProviderData))
	if err != nil {
		return nil, err
	}
	return response.Items().Slice(), err
}

func GetGCPVPCs(client *cmv1.Client, ccs cluster.CCS,
	gcpAuth cluster.GcpAuthentication, region string) (cloudVPCList []*cmv1.CloudVPC, err error) {

	gcpBuilder := cmv1.NewGCP()

	switch gcpAuth.Type {
	case cluster.AuthenticationWif:
		gcpAuth := cmv1.NewGcpAuthentication().
			Kind(cmv1.WifConfigKind).
			Id(gcpAuth.Id)
		gcpBuilder.Authentication(gcpAuth)
	case cluster.AuthenticationKey:
		gcpBuilder.ProjectID(ccs.GCP.ProjectID).
			ClientEmail(ccs.GCP.ClientEmail).
			Type(ccs.GCP.Type).
			PrivateKey(ccs.GCP.PrivateKey).
			PrivateKeyID(ccs.GCP.PrivateKeyID).
			AuthProviderX509CertURL(ccs.GCP.AuthProviderX509CertURL).
			AuthURI(ccs.GCP.AuthURI).TokenURI(ccs.GCP.TokenURI).
			ClientX509CertURL(ccs.GCP.ClientX509CertURL).
			ClientID(ccs.GCP.ClientID).TokenURI(ccs.GCP.TokenURI)
	default:
		return nil, fmt.Errorf("Failed to build GCP provider data, unexpected GCP authentication method %q", gcpAuth.Type)
	}

	cloudProviderData, err := cmv1.NewCloudProviderData().
		GCP(gcpBuilder).
		Region(cmv1.NewCloudRegion().ID(region)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("Failed to build GCP provider data: %v", err)
	}

	response, err := ocm.SendTypedAndHandleDeprecation(client.GCPInquiries().Vpcs().Search().
		Page(1).
		Size(-1).
		Body(cloudProviderData))
	if err != nil {
		return nil, err
	}
	return response.Items().Slice(), err
}

func GetAWSSubnetworks(client *cmv1.Client, ccs cluster.CCS,
	region string) (subnetworkList []*cmv1.Subnetwork, err error) {
	cloudVPCs, err := getAWSVPCs(client, ccs, region)
	if err != nil {
		return nil, err
	}

	for _, vpc := range cloudVPCs {
		subnetworkList = append(subnetworkList, vpc.AWSSubnets()...)
	}
	return subnetworkList, nil
}

func GetGCPSubnetList(client *cmv1.Client, provider string, ccs cluster.CCS,
	gcpAuth cluster.GcpAuthentication, region string) (subnetList []string, err error) {
	if ccs.Enabled && provider == "gcp" {

		cloudVPCs, err := GetGCPVPCs(client, ccs, gcpAuth, region)
		if err != nil {
			return nil, err
		}

		for _, vpc := range cloudVPCs {
			subnetList = append(subnetList, vpc.Subnets()...)
		}
	}
	return
}
