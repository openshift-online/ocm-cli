package gcp

import (
	"github.com/spf13/cobra"
)

var UpdateWorkloadIdentityConfigurationOpts struct {
	wifId      string
	templateId string
}

// NewCreateWorkloadIdentityConfiguration provides the "create-wif-config" subcommand
func NewUpdateWorkloadIdentityConfiguration() *cobra.Command {
	updateWorkloadIdentityPoolCmd := &cobra.Command{
		Use:              "wif-config",
		Short:            "Update wif-config.",
		Run:              updateWorkloadIdentityConfigurationCmd,
		PersistentPreRun: validationForUpdateWorkloadIdentityConfigurationCmd,
	}

	updateWorkloadIdentityPoolCmd.PersistentFlags().StringVar(&UpdateWorkloadIdentityConfigurationOpts.wifId, "wif-id", "", "Workload Identity Federation ID")
	updateWorkloadIdentityPoolCmd.MarkPersistentFlagRequired("wif-id")
	updateWorkloadIdentityPoolCmd.PersistentFlags().StringVar(&UpdateWorkloadIdentityConfigurationOpts.templateId, "template-id", "", "Template ID")
	updateWorkloadIdentityPoolCmd.MarkPersistentFlagRequired("template-id")

	return updateWorkloadIdentityPoolCmd
}

func validationForUpdateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	// No validation needed
}

func updateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	// No implementation yet
}
