package gcp

import (
	"github.com/spf13/cobra"
)

var UpdateWifConfigOpts struct {
	wifId      string
	templateId string
}

// NewUpdateWorkloadIdentityConfiguration provides the "gcp update wif-config" subcommand
func NewUpdateWorkloadIdentityConfiguration() *cobra.Command {
	updateWifConfigCmd := &cobra.Command{
		Use:     "wif-config",
		Short:   "Update wif-config.",
		RunE:    updateWorkloadIdentityConfigurationCmd,
		PreRunE: validationForUpdateWorkloadIdentityConfigurationCmd,
	}

	updateWifConfigCmd.PersistentFlags().StringVar(&UpdateWifConfigOpts.wifId, "wif-id", "",
		"Workload Identity Federation ID")
	updateWifConfigCmd.MarkPersistentFlagRequired("wif-id")
	updateWifConfigCmd.PersistentFlags().StringVar(&UpdateWifConfigOpts.templateId, "template-id", "",
		"Template ID")
	updateWifConfigCmd.MarkPersistentFlagRequired("template-id")

	return updateWifConfigCmd
}

func validationForUpdateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	// No validation needed
	return nil
}

func updateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	// No implementation yet
	return nil
}
