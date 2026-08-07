package gcp

import (
	"context"
	"fmt"
	"os"

	gcppkg "github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type DeleteFirewallRulesArgs struct {
	Mode       string
	OutputFile string
}

var deleteFwArgs DeleteFirewallRulesArgs

// NewDeleteFirewallRules provides the "gcp delete firewall-rules" subcommand
func NewDeleteFirewallRules() *cobra.Command {
	deleteFirewallRulesCmd := &cobra.Command{
		Use:   "firewall-rules [ID]",
		Short: "Delete firewall rules",
		Long: `Delete firewall rules.

Generates a bash script to delete firewall rules from GCP and removes the
firewall rule metadata from the OCM backend.

The generated script must be executed manually to delete the actual firewall
rules from GCP. The OCM backend metadata is deleted automatically.
`,
		Example: `
  # Delete firewall rules by ID
  ocm gcp delete firewall-rules <firewall-rule-id>

  # Generate bash script for manual deletion and save to file
  ocm gcp delete firewall-rules <firewall-rule-id> --mode manual --output-file delete-firewall-rules.sh
`,
		Args: cobra.ExactArgs(1),
		RunE: deleteFirewallRulesCmd,
	}

	deleteFirewallRulesCmd.PersistentFlags().StringVarP(
		&deleteFwArgs.Mode,
		"mode",
		"m",
		ModeManual,
		modeFlagDescription,
	)

	deleteFirewallRulesCmd.PersistentFlags().StringVarP(
		&deleteFwArgs.OutputFile,
		"output-file",
		"o",
		"",
		"Output file path for the generated bash script (manual mode only)",
	)

	return deleteFirewallRulesCmd
}

func deleteFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	firewallID := argv[0]
	if firewallID == "" {
		return fmt.Errorf("firewall rule ID is required")
	}

	// Validate mode - only manual mode is supported
	if deleteFwArgs.Mode != ModeManual {
		return fmt.Errorf("Only manual mode is supported at this time")
	}

	// Validate output file is provided
	if deleteFwArgs.OutputFile == "" {
		return fmt.Errorf("--output-file flag is required")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create OCM connection")
	}
	defer connection.Close()

	// Get the firewall rule details to generate the script
	getResponse, err := connection.ClustersMgmt().V1().
		GCP().
		FirewallRules().
		GcpFirewallRule(firewallID).
		Get().
		SendContext(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get GCP firewall rule")
	}

	firewallRule, ok := getResponse.GetBody()
	if !ok {
		return fmt.Errorf("response body is empty")
	}

	// Generate bash script for manual deletion from GCP
	bashScript := gcppkg.GenerateDeleteBashScript(gcppkg.NewFirewallRuleset(firewallRule))

	// Write to file
	err = os.WriteFile(deleteFwArgs.OutputFile, []byte(bashScript), 0600)
	if err != nil {
		return fmt.Errorf("Error writing script to file %s: %v", deleteFwArgs.OutputFile, err)
	}

	fmt.Printf("\nBash script written to: %s\n", deleteFwArgs.OutputFile)
	fmt.Printf("Execute the script to delete firewall rules from GCP.\n\n")

	// Delete the firewall rule metadata from OCM backend
	deleteResponse, err := connection.ClustersMgmt().V1().
		GCP().
		FirewallRules().
		GcpFirewallRule(firewallID).
		Delete().
		SendContext(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to delete GCP firewall rule from OCM backend")
	}

	fmt.Printf("Firewall rule '%s' deleted from OCM backend successfully (status: %d)\n",
		firewallID, deleteResponse.Status())

	return nil
}
