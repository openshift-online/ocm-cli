package provider

import (
	"fmt"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
)

func GetGcpKmsKeyLocations(client *cmv1.Client, ccs cluster.CCS,
	gcpAuth cluster.GcpAuthentication, region string) ([]string, error) {

	gcpDataBuilder, err := getGcpCloudProviderDataBuilder(gcpAuth, ccs)
	if err != nil {
		return nil, err
	}

	cloudProviderData, err := gcpDataBuilder.Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build GCP provider data")
	}

	response, err := client.GCPInquiries().Regions().Search().
		Page(1).Size(-1).Body(cloudProviderData).Send()

	if err != nil {
		return nil, err
	}

	var keyLocations []string
	for _, cloudRegion := range response.Items().Slice() {
		keyLocationsStr, ok := cloudRegion.GetKMSLocationID()
		if !ok {
			return nil, errors.Wrapf(err, "failed to build GCP provider data")
		}
		keyLocations = strings.Split(keyLocationsStr, ",")
	}

	return keyLocations, err
}

func GetGcpKmsKeyRings(client *cmv1.Client, ccs cluster.CCS,
	gcpAuth cluster.GcpAuthentication, keyLocation string) ([]*cmv1.KeyRing, error) {

	gcpDataBuilder, err := getGcpCloudProviderDataBuilder(gcpAuth, ccs)
	if err != nil {
		return nil, err
	}

	cloudProviderData, err := gcpDataBuilder.
		KeyLocation(keyLocation).
		Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build GCP provider data")
	}
	response, err := client.GCPInquiries().KeyRings().Search().
		Page(1).Size(-1).Body(cloudProviderData).Send()

	if err != nil {
		return nil, err
	}

	return response.Items().Slice(), err
}

func GetGcpKmsKeys(client *cmv1.Client, ccs cluster.CCS,
	gcpAuth cluster.GcpAuthentication, keyLocation string, keyRing string) ([]*cmv1.EncryptionKey, error) {

	gcpDataBuilder, err := getGcpCloudProviderDataBuilder(gcpAuth, ccs)
	if err != nil {
		return nil, err
	}

	cloudProviderData, err := gcpDataBuilder.
		KeyLocation(keyLocation).
		KeyRingName(keyRing).
		Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build GCP provider data")
	}

	response, err := client.GCPInquiries().EncryptionKeys().Search().
		Page(1).Size(-1).Body(cloudProviderData).Send()

	if err != nil {
		return nil, err
	}

	return response.Items().Slice(), err
}

func getGcpCloudProviderDataBuilder(gcpAuth cluster.GcpAuthentication,
	ccs cluster.CCS) (*cmv1.CloudProviderDataBuilder, error) {

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
		return nil, errors.New(
			fmt.Sprintf("failed to build GCP provider data, unexpected GCP authentication method %q", gcpAuth.Type))
	}

	return cmv1.NewCloudProviderData().GCP(gcpBuilder), nil
}
