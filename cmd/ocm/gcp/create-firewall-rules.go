package gcp

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	cmv1 "github.com/openshift-online/ocm-api-model/clientapi/clustersmgmt/v1"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	gcppkg "github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type CreateFirewallRulesArgs struct {
	Mode        string
	Interactive bool
	Name        string
	Profile     string
	ProjectID   string
	VpcName     string
	MachineCidr string
	WifConfigID string
	OutputFile  string
}

var createFwArgs CreateFirewallRulesArgs

// NewCreateFirewallRules provides the "gcp create firewall-rules" subcommand
func NewCreateFirewallRules() *cobra.Command {
	createFirewallRulesCmd := &cobra.Command{
		Use:   "firewall-rules",
		Short: "Create firewall rules",
		Long: `Create firewall rules.

Firewall rules control incoming and outgoing traffic to GCP resources.
This command creates firewall rules for OpenShift clusters.
`,
		Example: `
  # Create firewall rules
  ocm gcp create firewall-rules --name my-firewall-rule --wif-config <wif-config-id> \
    --project-id my-project --vpc-name my-vpc --machine-cidr 10.0.0.0/16

  # Create firewall rules with interactive mode
  ocm gcp create firewall-rules -i

  # Generate bash script and save to file (manual mode)
  ocm gcp create firewall-rules --name my-firewall-rule --wif-config <wif-config-id> \
    --project-id my-project --vpc-name my-vpc --machine-cidr 10.0.0.0/16 \
    --mode manual --output-file firewall-rules.sh
`,
		PreRunE: validationForCreateFirewallRulesCmd,
		RunE:    createFirewallRulesCmd,
	}

	arguments.AddInteractiveFlag(
		createFirewallRulesCmd.PersistentFlags(),
		&createFwArgs.Interactive,
	)

	createFirewallRulesCmd.PersistentFlags().StringVar(
		&createFwArgs.Name,
		"name",
		"",
		"Firewall rule name",
	)

	createFirewallRulesCmd.PersistentFlags().StringVarP(
		&createFwArgs.Mode,
		"mode",
		"m",
		ModeManual,
		modeFlagDescription,
	)

	createFirewallRulesCmd.PersistentFlags().StringVar(
		&createFwArgs.Profile,
		"profile",
		"public",
		"Firewall rule profile (default: public)",
	)

	createFirewallRulesCmd.PersistentFlags().StringVar(
		&createFwArgs.ProjectID,
		"project-id",
		"",
		"GCP project ID",
	)

	createFirewallRulesCmd.PersistentFlags().StringVar(
		&createFwArgs.VpcName,
		"vpc-name",
		"",
		"VPC network name",
	)

	createFirewallRulesCmd.PersistentFlags().StringVar(
		&createFwArgs.MachineCidr,
		"machine-cidr",
		"",
		"Machine CIDR block (e.g., 10.0.0.0/16)",
	)

	createFirewallRulesCmd.PersistentFlags().StringVar(
		&createFwArgs.WifConfigID,
		"wif-config",
		"",
		"Workload Identity Federation configuration ID",
	)

	createFirewallRulesCmd.PersistentFlags().StringVarP(
		&createFwArgs.OutputFile,
		"output-file",
		"o",
		"",
		"Output file path for the generated bash script (manual mode only)",
	)

	return createFirewallRulesCmd
}

func validationForCreateFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	// Validate mode
	if createFwArgs.Mode != ModeManual {
		return fmt.Errorf("Only manual mode is supported at this time")
	}

	// Validate output file is provided
	if createFwArgs.OutputFile == "" {
		return fmt.Errorf("--output-file flag is required")
	}

	if err := promptFirewallRuleName(); err != nil {
		return err
	}
	if err := promptWifConfigID(); err != nil {
		return err
	}
	if err := promptFirewallProjectID(); err != nil {
		return err
	}
	if err := promptVpcName(); err != nil {
		return err
	}
	if err := promptMachineCidr(); err != nil {
		return err
	}
	return nil
}

func promptFirewallRuleName() error {
	const help = "The name for the firewall rule."
	if createFwArgs.Name == "" {
		if createFwArgs.Interactive {
			prompt := &survey.Input{
				Message: "Firewall rule name:",
				Help:    help,
			}
			return survey.AskOne(
				prompt,
				&createFwArgs.Name,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'name' is required")
	}
	return nil
}

func promptWifConfigID() error {
	const help = "The Workload Identity Federation configuration ID."
	if createFwArgs.WifConfigID == "" {
		if createFwArgs.Interactive {
			prompt := &survey.Input{
				Message: "WIF configuration ID:",
				Help:    help,
			}
			return survey.AskOne(
				prompt,
				&createFwArgs.WifConfigID,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'wif-config' is required")
	}
	return nil
}

func promptFirewallProjectID() error {
	const help = "The GCP project ID that will host the firewall rules."
	if createFwArgs.ProjectID == "" {
		if createFwArgs.Interactive {
			prompt := &survey.Input{
				Message: "GCP Project ID:",
				Help:    help,
			}
			return survey.AskOne(
				prompt,
				&createFwArgs.ProjectID,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'project-id' is required")
	}
	return nil
}

func promptVpcName() error {
	const help = "The VPC network name."
	if createFwArgs.VpcName == "" {
		if createFwArgs.Interactive {
			prompt := &survey.Input{
				Message: "VPC network name:",
				Help:    help,
			}
			return survey.AskOne(
				prompt,
				&createFwArgs.VpcName,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'vpc-name' is required")
	}
	return nil
}

func promptMachineCidr() error {
	const help = "The machine CIDR block (e.g., 10.0.0.0/16)."
	if createFwArgs.MachineCidr == "" {
		if createFwArgs.Interactive {
			prompt := &survey.Input{
				Message: "Machine CIDR:",
				Help:    help,
				Default: "10.0.0.0/16",
			}
			return survey.AskOne(
				prompt,
				&createFwArgs.MachineCidr,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'machine-cidr' is required")
	}
	return nil
}

func createFirewallRulesCmd(cmd *cobra.Command, argv []string) error {
	// Validation is now done in PreRunE, so mode is guaranteed to be ModeManual here

	ctx := context.Background()
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create OCM connection")
	}
	defer connection.Close()

	// Build the GCP firewall rule
	firewallRule, err := cmv1.NewGcpFirewallRule().
		Name(createFwArgs.Name).
		Profile(createFwArgs.Profile).
		GcpNetwork(
			cmv1.NewGcpFirewallRuleGcpNetwork().
				ProjectId(createFwArgs.ProjectID).
				VpcName(createFwArgs.VpcName).
				MachineCidr(createFwArgs.MachineCidr)).
		WifConfig(cmv1.NewWifConfig().ID(createFwArgs.WifConfigID)).
		Build()
	if err != nil {
		return errors.Wrapf(err, "failed to build GCP firewall rule")
	}

	// Create the firewall rule via the API
	response, err := connection.ClustersMgmt().V1().
		GCP().
		FirewallRules().
		Add().
		Body(firewallRule).
		SendContext(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to create GCP firewall rule")
	}

	createdRule, ok := response.GetBody()
	if !ok {
		return fmt.Errorf("error processing firewall rule response")
	}

	// Generate the bash script for manual execution
	bashScript := gcppkg.GenerateCreateBashScript(gcppkg.NewFirewallRuleset(createdRule))

	// Write to file
	err = os.WriteFile(createFwArgs.OutputFile, []byte(bashScript), 0600)
	if err != nil {
		return fmt.Errorf(
			"error writing script to file %s: %v\n"+
				"please use the following command to regenerate the firewall rule creation script:\n"+
				"'ocm gcp update firewall-rules %s --output-file <path>'",
			createFwArgs.OutputFile, err, createdRule.ID())
	}
	fmt.Printf("\nBash script written to: %s\n", createFwArgs.OutputFile)

	return describeFirewallRule(createdRule)
}
