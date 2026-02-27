package gcp

import (
	"context"
	"os"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/output"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var ListDnsZoneOpts struct {
	columns   string
	noHeaders bool
}

// NewListDnsZone provides the "gcp list dns-zone" subcommand
func NewListDnsZone() *cobra.Command {
	listDnsZoneCmd := &cobra.Command{
		Use:     "dns-zone",
		Aliases: []string{"dns-zones"},
		Short:   "List DNS zones",
		Long: `List DNS zones.

The caller of the command will only view data from DNS zone objects that
belong to the user's organization.`,
		RunE: listDnsZoneCmd,
	}

	fs := listDnsZoneCmd.Flags()
	fs.StringVar(
		&ListDnsZoneOpts.columns,
		"columns",
		"id,gcp.domain_prefix,gcp.project_id,gcp.network_id",
		`Specify which columns to display separated by commas. 
The path is based on dns-zone struct.
`,
	)
	fs.BoolVar(
		&ListDnsZoneOpts.noHeaders,
		"no-headers",
		false,
		"Don't print header row",
	)

	return listDnsZoneCmd
}

func listDnsZoneCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	// Create the output printer:
	printer, err := output.NewPrinter().
		Writer(os.Stdout).
		Pager(cfg.Pager).
		Build(ctx)
	if err != nil {
		return err
	}
	defer printer.Close()

	// Create the output table:
	table, err := printer.NewTable().
		Name("dns_zones").
		Columns(ListDnsZoneOpts.columns).
		Build(ctx)
	if err != nil {
		return err
	}
	defer table.Close()

	// Unless noHeaders set, print header row:
	if !ListDnsZoneOpts.noHeaders {
		table.WriteHeaders()
	}

	// Create the request
	request := connection.ClustersMgmt().V1().DNSDomains().List()

	size := 100
	index := 1
	for {
		// Fetch the next page:
		request.Size(size)
		request.Page(index)
		response, err := request.Send()
		if err != nil {
			return errors.Wrapf(err, "can't retrieve dns zones")
		}

		// Display the items of the fetched page:
		response.Items().Each(func(dnsZone *cmv1.DNSDomain) bool {
			err = table.WriteObject(dnsZone)
			return err == nil
		})
		if err != nil {
			return err
		}

		// If the number of fetched items is less than requested, then this was the last
		// page, otherwise process the next one:
		if response.Size() < size {
			break
		}
		index++
	}
	return nil
}
