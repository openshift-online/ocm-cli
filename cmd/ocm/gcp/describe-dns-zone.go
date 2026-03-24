package gcp

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewDescribeDnsZone provides the "gcp describe dns-zone" subcommand
func NewDescribeDnsZone() *cobra.Command {
	describeDnsZoneCmd := &cobra.Command{
		Use:   "dns-zone [ID|BASE DOMAIN]",
		Short: "Show details of a dns-zone.",
		RunE:  describeDnsZoneCmd,
		Args:  cobra.ExactArgs(1),
	}

	return describeDnsZoneCmd
}

func describeDnsZoneCmd(cmd *cobra.Command, argv []string) error {

	if len(argv) != 1 {
		return errors.New("expected one command line parameter containing the " +
			"ID or Base Domain of the DNS zone")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	// Get the DNS domain from OCM
	dnsZone, err := getDnsDomain(connection, argv[0])
	if err != nil {
		return errors.Wrapf(err, "failed to get dns-domain")
	}

	w := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', 0)

	fmt.Fprintf(w, "ID|BASE DOMAIN:\t%s\n", dnsZone.ID())
	fmt.Fprintf(w, "Domain Prefix:\t%s\n", dnsZone.Gcp().DomainPrefix())
	fmt.Fprintf(w, "Project:\t%s\n", dnsZone.Gcp().ProjectId())
	fmt.Fprintf(w, "Network:\t%s\n", dnsZone.Gcp().NetworkId())
	fmt.Fprintf(w, "Zone Name:\t%s\n", gcp.FmtDnsZoneName(dnsZone.Gcp().DomainPrefix(), dnsZone.ID()))
	fmt.Fprintf(w, "DNS Name:\t%s\n", gcp.FmtDnsName(dnsZone.Gcp().DomainPrefix(), dnsZone.ID()))
	return w.Flush()
}
