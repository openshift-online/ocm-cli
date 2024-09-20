package gcp

import (
	"context"
	"fmt"
	"log"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var UpdateWifConfigOpts struct {
}

// NewUpdateWorkloadIdentityConfiguration provides the "gcp update wif-config" subcommand
func NewUpdateWorkloadIdentityConfiguration() *cobra.Command {
	updateWifConfigCmd := &cobra.Command{
		Use:   "wif-config [ID|Name]",
		Short: "Update wif-config.",
		RunE:  updateWorkloadIdentityConfigurationCmd,
	}

	return updateWifConfigCmd
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
