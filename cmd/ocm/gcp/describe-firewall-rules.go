package gcp

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewDescribeFirewallRules provides the "gcp describe firewall-rules" subcommand
func NewDescribeFirewallRules() *cobra.Command {
	describeFirewallRulesCmd := &cobra.Command{
		Use:   "firewall-rules [ID]",
		Short: "Describe firewall rules",
		Long: `Describe firewall rules.

Shows detailed information about a specific firewall rule including its configuration,
network settings, and rendered GCP firewall rule specifications.
`,
		Example: `
  # Describe firewall rules by ID
  ocm gcp describe firewall-rules <firewall-rule-id>
`,
		Args: cobra.ExactArgs(1),
		RunE: describeFirewallRulesCmd,
	}

	return describeFirewallRulesCmd
}

func describeFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	if len(argv) != 1 {
		return errors.New("expected one command line parameter containing the ID of the firewall rule")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create OCM connection")
	}
	defer connection.Close()

	// Get the firewall rule from OCM
	firewallRule, err := getFirewallRule(ctx, connection, argv[0])
	if err != nil {
		return err
	}

	return describeFirewallRule(firewallRule)
}

func getFirewallRule(
	ctx context.Context,
	connection *sdk.Connection,
	firewallRuleId string,
) (*cmv1.GcpFirewallRule, error) {

	response, err := connection.ClustersMgmt().V1().
		GCP().
		FirewallRules().
		GcpFirewallRule(firewallRuleId).
		Get().
		SendContext(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get firewall rule")
	}

	firewallRule, ok := response.GetBody()
	if !ok {
		return nil, fmt.Errorf("response body is empty")
	}

	return firewallRule, nil
}

func describeFirewallRule(
	firewallRule *cmv1.GcpFirewallRule,
) error {

	w := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', 0)

	// Basic information
	fmt.Fprintf(w, "ID:\t%s\n", firewallRule.ID())
	fmt.Fprintf(w, "Name:\t%s\n", firewallRule.Name())
	fmt.Fprintf(w, "Profile:\t%s\n", firewallRule.Profile())
	fmt.Fprintf(w, "HREF:\t%s\n", firewallRule.HREF())

	// Network configuration
	if network, ok := firewallRule.GetGcpNetwork(); ok {
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "Network Configuration:\t\n")
		if projectId, ok := network.GetProjectId(); ok {
			fmt.Fprintf(w, "  Project ID:\t%s\n", projectId)
		}
		if vpcName, ok := network.GetVpcName(); ok {
			fmt.Fprintf(w, "  VPC Name:\t%s\n", vpcName)
		}
		if cidr, ok := network.GetMachineCidr(); ok {
			fmt.Fprintf(w, "  Machine CIDR:\t%s\n", cidr)
		}
	}

	// WIF Config
	if wifConfig, ok := firewallRule.GetWifConfig(); ok {
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "WIF Configuration:\t\n")
		fmt.Fprintf(w, "  ID:\t%s\n", wifConfig.ID())
		if href, ok := wifConfig.GetHREF(); ok {
			fmt.Fprintf(w, "  HREF:\t%s\n", href)
		}
	}

	// Status and rendered rules
	if status, ok := firewallRule.GetStatus(); ok {
		if rules, ok := status.GetRules(); ok && len(rules) > 0 {
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, "Rendered Firewall Rules:\t\n")
			for i, rule := range rules {
				fmt.Fprintf(w, "  Rule %d:\t\n", i+1)
				if name, ok := rule.GetName(); ok {
					fmt.Fprintf(w, "    Name:\t%s\n", name)
				}
				if network, ok := rule.GetNetwork(); ok {
					fmt.Fprintf(w, "    Network:\t%s\n", network)
				}
				if direction, ok := rule.GetDirection(); ok {
					fmt.Fprintf(w, "    Direction:\t%s\n", direction)
				}
				if priority, ok := rule.GetPriority(); ok {
					fmt.Fprintf(w, "    Priority:\t%d\n", priority)
				}
				if allowed, ok := rule.GetAllowed(); ok && len(allowed) > 0 {
					fmt.Fprintf(w, "    Allowed:\t\n")
					for j, allow := range allowed {
						protocol := allow.IPProtocol()
						if ports, ok := allow.GetPorts(); ok && len(ports) > 0 {
							fmt.Fprintf(w, "      [%d] Protocol:\t%s, Ports: %v\n", j+1, protocol, ports)
						} else {
							fmt.Fprintf(w, "      [%d] Protocol:\t%s\n", j+1, protocol)
						}
					}
				}

				if sourceRanges, ok := rule.GetSourceRanges(); ok && len(sourceRanges) > 0 {
					fmt.Fprintf(w, "    Source Ranges:\t%v\n", sourceRanges)
				}
				if sourceAccounts, ok := rule.GetSourceServiceAccounts(); ok && len(sourceAccounts) > 0 {
					fmt.Fprintf(w, "    Source Service Accounts:\t%v\n", sourceAccounts)
				}
				if targetAccounts, ok := rule.GetTargetServiceAccounts(); ok && len(targetAccounts) > 0 {
					fmt.Fprintf(w, "    Target Service Accounts:\t%v\n", targetAccounts)
				}
			}
		}
	}

	return w.Flush()
}
