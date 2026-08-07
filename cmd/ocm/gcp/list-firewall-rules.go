package gcp

import (
	"context"
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/output"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var ListFirewallRulesOpts struct {
	columns     string
	noHeaders   bool
	wifConfigID string
	unused      bool
}

// NewListFirewallRules provides the "gcp list firewall-rules" subcommand
func NewListFirewallRules() *cobra.Command {
	listFirewallRulesCmd := &cobra.Command{
		Use:     "firewall-rules",
		Aliases: []string{"firewall-rule"},
		Short:   "List firewall rules",
		Long: `List firewall rules.

Lists firewall rules for OpenShift clusters in GCP.
The caller of the command will only view firewall rules that belong to the user's organization.
`,
		Example: `
  # List all firewall rules
  ocm gcp list firewall-rules

  # List firewall rules for a specific WIF configuration
  ocm gcp list firewall-rules --wif-config <wif-config-id>

  # List unused firewall rules
  ocm gcp list firewall-rules --unused

  # List firewall rules with custom columns
  ocm gcp list firewall-rules --columns id,name,profile
`,
		RunE: listFirewallRulesCmd,
	}

	fs := listFirewallRulesCmd.Flags()
	fs.StringVar(
		&ListFirewallRulesOpts.columns,
		"columns",
		"id,name,profile,gcp_network.project_id,gcp_network.vpc_name",
		`Specify which columns to display separated by commas.
The path is based on firewall-rule struct.
`,
	)
	fs.BoolVar(
		&ListFirewallRulesOpts.noHeaders,
		"no-headers",
		false,
		"Don't print header row",
	)
	fs.StringVar(
		&ListFirewallRulesOpts.wifConfigID,
		"wif-config",
		"",
		"Filter by WIF configuration ID",
	)
	fs.BoolVar(
		&ListFirewallRulesOpts.unused,
		"unused",
		false,
		"Filter to show only unused firewall rules",
	)

	return listFirewallRulesCmd
}

func listFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	// Load the configuration:
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create OCM connection")
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
		Name("firewall_rules").
		Columns(ListFirewallRulesOpts.columns).
		Build(ctx)
	if err != nil {
		return err
	}
	defer table.Close()

	// Unless noHeaders set, print header row:
	if !ListFirewallRulesOpts.noHeaders {
		table.WriteHeaders()
	}

	// Create the request
	request := connection.ClustersMgmt().V1().
		GCP().
		FirewallRules().
		List()

	// Add WIF config filter if specified
	if ListFirewallRulesOpts.wifConfigID != "" {
		request.Parameter("wif_id", ListFirewallRulesOpts.wifConfigID)
	}

	// Add unused filter if specified
	if ListFirewallRulesOpts.unused {
		request.Parameter("unused", "true")
	}

	size := 100
	index := 1
	for {
		// Fetch the next page:
		request.Size(size)
		request.Page(index)
		response, err := request.Send()
		if err != nil {
			return errors.Wrapf(err, "can't retrieve firewall rules")
		}

		// Display the items of the fetched page:
		response.Items().Each(func(firewallRule *cmv1.GcpFirewallRule) bool {
			err = table.WriteObject(firewallRule)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing firewall rule: %v\n", err)
			}
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
