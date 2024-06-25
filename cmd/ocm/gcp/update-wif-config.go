package gcp

import (
	"github.com/spf13/cobra"
)

var UpdateWifConfigOpts struct {
	wifId      string
	templateId string
}

// NewCreateWorkloadIdentityConfiguration provides the "create-wif-config" subcommand
func NewUpdateWorkloadIdentityConfiguration() *cobra.Command {
	updateWifConfigCmd := &cobra.Command{
		Use:              "wif-config",
		Short:            "Update wif-config.",
		Run:              updateWorkloadIdentityConfigurationCmd,
		PersistentPreRun: validationForUpdateWorkloadIdentityConfigurationCmd,
	}

	updateWifConfigCmd.PersistentFlags().StringVar(&UpdateWifConfigOpts.wifId, "wif-id", "",
		"Workload Identity Federation ID")
	updateWifConfigCmd.MarkPersistentFlagRequired("wif-id")
	updateWifConfigCmd.PersistentFlags().StringVar(&UpdateWifConfigOpts.templateId, "template-id", "",
		"Template ID")
	updateWifConfigCmd.MarkPersistentFlagRequired("template-id")

	return updateWifConfigCmd
}

func validationForUpdateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	// No validation needed
}

func updateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	// No implementation yet
}
