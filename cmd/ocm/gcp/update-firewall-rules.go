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

type UpdateFirewallRulesArgs struct {
	Mode       string
	OutputFile string
}

var updateFwArgs UpdateFirewallRulesArgs

// NewUpdateFirewallRules provides the "gcp update firewall-rules" subcommand
func NewUpdateFirewallRules() *cobra.Command {
	updateFirewallRulesCmd := &cobra.Command{
		Use:   "firewall-rules [ID]",
		Short: "Update firewall rules",
		Long: `Update firewall rules.

Generates a bash script to update GCP firewall rules to match the intended configuration.
This is useful to restore firewall rules to their correct state if they have been misconfigured.

Note: Only manual mode is supported. The command generates a script that you must execute manually.
`,
		Example: `
  # Generate update script for firewall rules
  ocm gcp update firewall-rules <firewall-rule-id> --output-file update-firewall-rules.sh

  # Execute the generated script
  bash update-firewall-rules.sh
`,
		Args: cobra.ExactArgs(1),
		RunE: updateFirewallRulesCmd,
	}

	updateFirewallRulesCmd.PersistentFlags().StringVarP(
		&updateFwArgs.Mode,
		"mode",
		"m",
		ModeManual,
		modeFlagDescription,
	)

	updateFirewallRulesCmd.PersistentFlags().StringVarP(
		&updateFwArgs.OutputFile,
		"output-file",
		"o",
		"",
		"Output file path for the generated bash script (required)",
	)

	return updateFirewallRulesCmd
}

func updateFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	firewallID := argv[0]
	if firewallID == "" {
		return fmt.Errorf("firewall rule ID is required")
	}

	// Validate mode - only manual mode is supported
	if updateFwArgs.Mode != ModeManual {
		return fmt.Errorf("Only manual mode is supported at this time")
	}

	// Validate output file is provided
	if updateFwArgs.OutputFile == "" {
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

	// Generate bash script for manual update
	bashScript := gcppkg.GenerateUpdateBashScript(gcppkg.NewFirewallRuleset(firewallRule))

	// Write to file
	err = os.WriteFile(updateFwArgs.OutputFile, []byte(bashScript), 0600)
	if err != nil {
		return fmt.Errorf("Error writing script to file %s: %v", updateFwArgs.OutputFile, err)
	}

	fmt.Printf("\nBash script written to: %s\n", updateFwArgs.OutputFile)
	fmt.Printf("Execute the script to update firewall rules in GCP to match the intended configuration.\n")

	return nil
}
