package gcp

import (
	"context"
	"log"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// NewDeleteDnsZone provides the "gcp delete dns-zone" subcommand
func NewDeleteDnsZone() *cobra.Command {
	deleteDnsZoneCmd := &cobra.Command{
		Use:   "dns-zone [ID|BASE DOMAIN]",
		Short: "Delete a DNS zone",
		Long: `Delete a DNS zone.

Deleting the dns-zone resource will remove the OCM metadata, as well as the
GCP resources represented by the dns-zone.`,
		RunE: deleteDnsZoneCmd,
	}

	return deleteDnsZoneCmd
}

func deleteDnsZoneCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
	log := log.Default()

	if len(argv) != 1 {
		return errors.New("expected one command line parameter containing the " +
			"ID or Base Domain of the DNS zone")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	// Get the DNS zone from OCM
	dnsDomain, err := getDnsDomain(
		connection,
		argv[0],
	)
	if err != nil {
		return errors.Wrapf(err, "failed to get dns-domain")
	}

	// Delete the DNS zone from GCP (no-op if not found)
	err = gcpClient.DeleteDnsZone(ctx, dnsDomain)
	if err != nil {
		return errors.Wrapf(err, "failed to delete dns-zone")
	}

	log.Printf("gcp dns-zone '%s' deleted successfully.",
		gcp.FmtDnsZoneName(dnsDomain.Gcp().DomainPrefix(), dnsDomain.ID()))

	// Delete the DNS domain from OCM
	err = deleteDnsDomain(
		connection,
		dnsDomain.ID(),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to delete dns-domain")
	}

	log.Printf("dns-domain '%s' deleted successfully.", dnsDomain.ID())

	return nil
}

func getDnsDomain(
	connection *sdk.Connection,
	dnsDomainId string,
) (*cmv1.DNSDomain, error) {

	response, err := connection.ClustersMgmt().V1().
		DNSDomains().
		DNSDomain(dnsDomainId).
		Get().
		Send()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get dns-zone")
	}

	return response.Body(), nil
}

func deleteDnsDomain(
	connection *sdk.Connection,
	dnsDomainId string,
) error {

	_, err := connection.ClustersMgmt().V1().
		DNSDomains().
		DNSDomain(dnsDomainId).
		Delete().
		Send()
	if err != nil {
		return errors.Wrap(err, "failed to delete dns-domain")
	}

	return nil
}
