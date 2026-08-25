package gcp

import (
	"fmt"
	"log"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewVerifyFirewallRules provides the "gcp verify firewall-rules" subcommand
func NewVerifyFirewallRules() *cobra.Command {
	verifyFirewallRulesCmd := &cobra.Command{
		Use:   "firewall-rules [ID]",
		Short: "Verify firewall rules configuration",
		Long: `Verify firewall rules configuration.

Checks whether the firewall rules are properly configured in GCP or if they are
missing or misconfigured. This uses the status endpoint to validate the current
state of the firewall rules against the intended configuration.
`,
		Example: `
  # Verify firewall rules by ID
  ocm gcp verify firewall-rules <firewall-rule-id>
`,
		Args: cobra.ExactArgs(1),
		RunE: verifyFirewallRulesCmd,
	}

	return verifyFirewallRulesCmd
}

func verifyFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	log := log.Default()

	firewallID := argv[0]
	if firewallID == "" {
		return fmt.Errorf("firewall rule ID is required")
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	// Verify the firewall rule exists
	getResponse, err := connection.ClustersMgmt().V1().
		GCP().
		FirewallRules().
		GcpFirewallRule(firewallID).
		Get().
		Send()
	if err != nil {
		return errors.Wrapf(err, "failed to get GCP firewall rule")
	}

	firewallRule, ok := getResponse.GetBody()
	if !ok {
		return fmt.Errorf("response body is empty")
	}

	// Verify the firewall rule configuration
	if err := verifyFirewallRule(connection, firewallRule.ID()); err != nil {
		return errors.Wrapf(err, "failed to verify firewall rules")
	}

	log.Println("Firewall rules are properly configured")
	return nil
}

func verifyFirewallRule(
	connection *sdk.Connection,
	firewallID string,
) error {
	// Call the status endpoint to verify the firewall rule configuration
	response, err := connection.
		ClustersMgmt().V1().
		GCP().
		FirewallRules().
		GcpFirewallRule(firewallID).
		Status().
		Get().
		Send()
	if err != nil {
		return fmt.Errorf("failed to call firewall rule verification: %s", err.Error())
	}

	status, ok := response.GetBody()
	if !ok {
		return fmt.Errorf("verification response body is empty")
	}

	// Check overall state
	state, ok := status.GetState()
	if !ok {
		return fmt.Errorf("state not available in verification response")
	}
	if state != "ready" {
		description, _ := status.GetDescription()
		// Ensure rules are returned.
		rules, ok := status.GetRules()
		if !ok || len(rules) == 0 {
			return fmt.Errorf("no firewall rules found in status response: %s", description)
		}

		// Check if all rules are properly configured
		hasIssues := false
		var issueDetails string

		// Check individual rules
		for _, rule := range rules {
			ruleName, _ := rule.GetName()
			exists, _ := rule.GetExists()
			configuredCorrectly, _ := rule.GetConfiguredCorrectly()
			description, _ := rule.GetDescription()

			if !exists {
				hasIssues = true
				issueDetails += fmt.Sprintf("  - Rule '%s': MISSING - %s\n", ruleName, description)
			} else if !configuredCorrectly {
				hasIssues = true
				issueDetails += fmt.Sprintf("  - Rule '%s': MISCONFIGURED - %s\n", ruleName, description)
			}
		}

		if hasIssues {
			overallDescription, _ := status.GetDescription()
			return fmt.Errorf(
				"verification failed (state: %s): %s\n\n"+
					"Issues found:\n%s\n"+
					"Running 'ocm gcp update firewall-rules %s --output-file update.sh' will generate "+
					"a script to fix misconfigured firewall rules.",
				state, overallDescription, issueDetails, firewallID)
		}

		return fmt.Errorf("firewall rules state is '%s', expected 'ready': %s", state, description)
	}
	return nil

}
