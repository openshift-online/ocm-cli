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

var ListWorkloadIdentityConfigurationOpts struct {
	columns   string
	noHeaders bool
}

// NewListWorkloadIdentityConfiguration provides the "gcp list wif-config" subcommand
func NewListWorkloadIdentityConfiguration() *cobra.Command {
	listWorkloadIdentityPoolCmd := &cobra.Command{
		Use:     "wif-config",
		Aliases: []string{"wif-configs"},
		Short:   "List wif-configs.",
		RunE:    listWorkloadIdentityConfigurationCmd,
		PreRunE: validationForListWorkloadIdentityConfigurationCmd,
	}

	fs := listWorkloadIdentityPoolCmd.Flags()
	fs.StringVar(
		&ListWorkloadIdentityConfigurationOpts.columns,
		"columns",
		"id, display_name",
		"Specify which columns to display separated by commas, path is based on wif-config struct",
	)
	fs.BoolVar(
		&ListWorkloadIdentityConfigurationOpts.noHeaders,
		"no-headers",
		false,
		"Don't print header row",
	)

	return listWorkloadIdentityPoolCmd
}

func validationForListWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	// No validation needed
	return nil
}

func listWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	// Create a context:
	ctx := context.Background()

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Create the client for the OCM API:
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
		Name("wifconfigs").
		Columns(ListWorkloadIdentityConfigurationOpts.columns).
		Build(ctx)
	if err != nil {
		return err
	}
	defer table.Close()

	// Unless noHeaders set, print header row:
	if !ListWorkloadIdentityConfigurationOpts.noHeaders {
		table.WriteHeaders()
	}

	// Create the request
	request := connection.ClustersMgmt().V1().GCP().WifConfigs().List()

	size := 100
	index := 1
	for {
		// Fetch the next page:
		request.Size(size)
		request.Page(index)
		response, err := request.Send()
		if err != nil {
			return errors.Wrapf(err, "can't retrieve wif configs")
		}

		// Display the items of the fetched page:
		response.Items().Each(func(wc *cmv1.WifConfig) bool {
			err = table.WriteObject(wc)
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
