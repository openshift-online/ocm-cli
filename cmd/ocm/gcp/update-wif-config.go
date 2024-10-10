package gcp

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	UpdateWifConfigOpts = options{
		Mode:      ModeAuto,
		TargetDir: "",
	}
)

// NewUpdateWorkloadIdentityConfiguration provides the "gcp update wif-config" subcommand
func NewUpdateWorkloadIdentityConfiguration() *cobra.Command {
	updateWifConfigCmd := &cobra.Command{
		Use:     "wif-config [ID|Name]",
		Short:   "Update wif-config.",
		RunE:    updateWorkloadIdentityConfigurationCmd,
		PreRunE: validationForUpdateWorkloadIdentityConfigurationCmd,
	}

	updateWifConfigCmd.PersistentFlags().StringVarP(
		&UpdateWifConfigOpts.Mode,
		"mode",
		"m",
		ModeAuto,
		"How to perform the operation. Valid options are:\n"+
			"auto (default): Resource changes will be automatic applied using the current GCP account\n"+
			"manual: Commands necessary to modify GCP resources will be output to be run manually",
	)
	updateWifConfigCmd.PersistentFlags().StringVar(&UpdateWifConfigOpts.TargetDir, "output-dir", "",
		"Directory to place generated files (defaults to current directory)")

	return updateWifConfigCmd
}

func validationForUpdateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	var err error

	if UpdateWifConfigOpts.Mode != ModeAuto && UpdateWifConfigOpts.Mode != ModeManual {
		return fmt.Errorf("Invalid mode. Allowed values are %s", Modes)
	}

	UpdateWifConfigOpts.TargetDir, err = getPathFromFlag(UpdateWifConfigOpts.TargetDir)
	if err != nil {
		return err
	}
	return nil
}

func updateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
	log := log.Default()
	key, err := wifKeyFromArgs(argv)
	if err != nil {
		return err
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	// Verify the WIF configuration exists
	wifConfig, err := findWifConfig(connection.ClustersMgmt().V1(), key)
	if err != nil {
		return errors.Wrapf(err, "failed to get wif-config")
	}

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	if UpdateWifConfigOpts.Mode == ModeManual {
		log.Printf("Writing script files to %s", UpdateWifConfigOpts.TargetDir)
		projectNumInt64, err := strconv.ParseInt(wifConfig.Gcp().ProjectNumber(), 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse project number from WifConfig")
		}

		if err := createUpdateScript(UpdateWifConfigOpts.TargetDir, wifConfig, projectNumInt64); err != nil {
			return errors.Wrapf(err, "failed to generate script files")
		}
		return nil
	}

	// Re-apply WIF resources
	gcpClientWifConfigShim := NewGcpClientWifConfigShim(GcpClientWifConfigShimSpec{
		GcpClient: gcpClient,
		WifConfig: wifConfig,
	})

	if err := gcpClientWifConfigShim.GrantSupportAccess(ctx, log); err != nil {
		return fmt.Errorf("Failed to grant support access to project: %s", err)
	}

	if err := gcpClientWifConfigShim.CreateWorkloadIdentityPool(ctx, log); err != nil {
		return fmt.Errorf("Failed to update workload identity pool: %s", err)
	}

	if err = gcpClientWifConfigShim.CreateWorkloadIdentityProvider(ctx, log); err != nil {
		return fmt.Errorf("Failed to update workload identity provider: %s", err)
	}

	if err = gcpClientWifConfigShim.CreateServiceAccounts(ctx, log); err != nil {
		return fmt.Errorf("Failed to update IAM service accounts: %s", err)
	}

	return nil
}
