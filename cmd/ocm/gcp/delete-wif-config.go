package gcp

import (
	"context"
	"fmt"
	"strings"

	"log"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"google.golang.org/api/googleapi"

	"github.com/spf13/cobra"
)

var (
	// DeleteWifConfigOpts captures the options that affect creation of the workload identity configuration
	DeleteWifConfigOpts = options{
		Mode:      ModeAuto,
		TargetDir: "",
	}
)

// NewDeleteWorkloadIdentityConfiguration provides the "gcp delete wif-config" subcommand
func NewDeleteWorkloadIdentityConfiguration() *cobra.Command {
	deleteWifConfigCmd := &cobra.Command{
		Use:   "wif-config [ID|Name]",
		Short: "Delete workload identity federation configuration (wif-config)",
		Long: `Delete workload identity federation configuration (wif-config)

wif-config resources may only be deleted if they are not in use by an OSD-GCP
cluster. Deleting a wif-config that is presently utilized is prevented and
results in an error returned. Clusters that are utilizing a specific wif-config
may be listed using the following command: 

ocm list cluster --parameter search="gcp.authentication.wif_config_id = '<WIF_ID>'"

Where <WIF_ID> is the id of the wif-config resource.

Deleting the wif-config resource will remove the OCM metadata, as well as the
GCP resources represented by the wif-config.
        `,
		RunE:    deleteWorkloadIdentityConfigurationCmd,
		PreRunE: validationForDeleteWorkloadIdentityConfigurationCmd,
	}

	deleteWifConfigCmd.PersistentFlags().StringVarP(
		&DeleteWifConfigOpts.Mode,
		"mode",
		"m",
		ModeAuto,
		modeFlagDescription,
	)
	deleteWifConfigCmd.PersistentFlags().StringVar(
		&DeleteWifConfigOpts.TargetDir,
		"output-dir",
		"",
		targetDirFlagDescription,
	)

	return deleteWifConfigCmd
}

func validationForDeleteWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	var err error

	if DeleteWifConfigOpts.Mode != ModeAuto && DeleteWifConfigOpts.Mode != ModeManual {
		return fmt.Errorf("Invalid mode. Allowed values are %s", Modes)
	}

	DeleteWifConfigOpts.TargetDir, err = getPathFromFlag(DeleteWifConfigOpts.TargetDir)
	if err != nil {
		return err
	}
	return nil
}

func deleteWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
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

	// Check if wif-config can be deleted with dry-run
	_, err = connection.ClustersMgmt().V1().GCP().WifConfigs().
		WifConfig(wifConfig.ID()).Delete().DryRun(true).Send()
	if err != nil {
		return errors.Wrapf(err, "failed to delete wif config %q", wifConfig.ID())
	}

	if DeleteWifConfigOpts.Mode == ModeManual {
		log.Printf("Writing script files to %s", DeleteWifConfigOpts.TargetDir)

		err := createDeleteScript(DeleteWifConfigOpts.TargetDir, wifConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to create script files")
		}
		return nil
	}

	gcpClient, err := gcp.NewGcpClient(context.Background())
	if err != nil {
		return err
	}

	if err := unbindServiceAccounts(ctx, gcpClient, wifConfig); err != nil {
		return err
	}

	if err := deleteServiceAccounts(ctx, gcpClient, wifConfig, true); err != nil {
		return err
	}

	if err := deleteWorkloadIdentityPool(ctx, gcpClient, wifConfig, true); err != nil {
		return err
	}

	_, err = connection.ClustersMgmt().V1().GCP().WifConfigs().
		WifConfig(wifConfig.ID()).
		Delete().
		Send()
	if err != nil {
		return errors.Wrapf(err, "failed to delete wif config %q", wifConfig.ID())
	}
	return nil
}

func deleteServiceAccounts(ctx context.Context, gcpClient gcp.GcpClient,
	wifConfig *cmv1.WifConfig, allowMissing bool) error {
	log.Println("Deleting service accounts...")
	projectId := wifConfig.Gcp().ProjectId()

	for _, serviceAccount := range wifConfig.Gcp().ServiceAccounts() {
		serviceAccountID := serviceAccount.ServiceAccountId()
		log.Println("Deleting service account", serviceAccountID)
		err := gcpClient.DeleteServiceAccount(ctx, serviceAccountID, projectId, allowMissing)
		if err != nil {
			return errors.Wrapf(err, "Failed to delete service account %q", serviceAccountID)
		}
	}

	return nil
}

func deleteWorkloadIdentityPool(ctx context.Context, gcpClient gcp.GcpClient,
	wifConfig *cmv1.WifConfig, allowMissing bool) error {
	log.Println("Deleting workload identity pool...")
	projectId := wifConfig.Gcp().ProjectId()
	poolName := wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	poolResource := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", projectId, poolName)

	_, err := gcpClient.DeleteWorkloadIdentityPool(ctx, poolResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") && allowMissing {
			log.Printf("Workload identity pool %q not found", poolName)
			return nil
		}
		return errors.Wrapf(err, "Failed to delete workload identity pool %q", poolName)
	}

	log.Printf("Workload identity pool %q deleted", poolName)
	return nil
}

func unbindServiceAccounts(
	ctx context.Context,
	gcpClient gcp.GcpClient,
	wifConfig *cmv1.WifConfig,
) error {
	log.Println("Unbinding service accounts...")

	gcpShim := &shim{
		wifConfig: wifConfig,
		gcpClient: gcpClient,
	}

	for _, serviceAccount := range wifConfig.Gcp().ServiceAccounts() {
		err := gcpShim.unbindRolesFromPrincipal(ctx, log.Default(),
			gcpShim.formatServiceAccountId(serviceAccount),
			serviceAccount.Roles())
		if err != nil {
			return errors.Wrapf(err, "Failed to unbind service account %q", serviceAccount.ServiceAccountId())
		}
	}

	return nil
}
