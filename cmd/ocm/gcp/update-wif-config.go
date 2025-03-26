package gcp

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	UpdateWifConfigOpts = options{
		Mode:             ModeAuto,
		TargetDir:        "",
		OpenshiftVersion: "",
	}
)

// NewUpdateWorkloadIdentityConfiguration provides the "gcp update wif-config" subcommand
func NewUpdateWorkloadIdentityConfiguration() *cobra.Command {
	updateWifConfigCmd := &cobra.Command{
		Use:   "wif-config [ID|Name]",
		Short: "Update workload identity federation configuration (wif-config).",
		Long: `Update workload identity federation configuration (wif-config).

wif-config that are in use by cluster deployments may require updation before
the cluster version upgrade may continue. This command may be used to update
the wif-config metadata and the GCP resources it represents.`,
		RunE:    updateWorkloadIdentityConfigurationCmd,
		PreRunE: validationForUpdateWorkloadIdentityConfigurationCmd,
	}

	updateWifConfigCmd.PersistentFlags().StringVarP(
		&UpdateWifConfigOpts.Mode,
		"mode",
		"m",
		ModeAuto,
		modeFlagDescription,
	)
	updateWifConfigCmd.PersistentFlags().StringVar(
		&UpdateWifConfigOpts.TargetDir,
		"output-dir",
		"",
		targetDirFlagDescription,
	)
	updateWifConfigCmd.PersistentFlags().StringVar(
		&UpdateWifConfigOpts.OpenshiftVersion,
		"version",
		"",
		versionFlagDescription,
	)

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

	wifBuilder := cmv1.NewWifConfig()
	// Update the WIF configuration
	if UpdateWifConfigOpts.OpenshiftVersion != "" {
		wifTemplate := versionToTemplateID(UpdateWifConfigOpts.OpenshiftVersion)

		existingTemplates, _ := wifConfig.GetWifTemplates()
		wifBuilder.WifTemplates(append(existingTemplates, wifTemplate)...)
	}

	updatedWifConfig, err := wifBuilder.Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create wif-config body")
	}

	resp, err := connection.ClustersMgmt().V1().GCP().WifConfigs().
		WifConfig(wifConfig.ID()).Update().Body(updatedWifConfig).Send()
	if err != nil {
		return errors.Wrapf(err, "failed to update wif-config")
	}
	wifConfig = resp.Body()

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

	log.Println("Updating support access...")
	if err := gcpClientWifConfigShim.GrantSupportAccess(ctx, log); err != nil {
		return fmt.Errorf("Failed to grant support access to project: %s", err)
	}

	log.Println("Updating workload identity pool...")
	if err := gcpClientWifConfigShim.CreateWorkloadIdentityPool(ctx, log); err != nil {
		return fmt.Errorf("Failed to update workload identity pool: %s", err)
	}

	log.Println("Updating oidc provider...")
	if err = gcpClientWifConfigShim.CreateWorkloadIdentityProvider(ctx, log); err != nil {
		return fmt.Errorf("Failed to update workload identity provider: %s", err)
	}

	log.Println("Updating service accounts...")
	if err = gcpClientWifConfigShim.CreateServiceAccounts(ctx, log); err != nil {
		return fmt.Errorf("Failed to update IAM service accounts: %s", err)
	}

	//The IAM API is eventually consistent. If the user created the service
	//accounts needed for cluster deployment within too brief a period, then
	//our backend will not yet have access to it. To avoid confusing error
	//messages being returned, we will verify that the backend can see the
	//resources before we consider the wif-config creation complete.
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		log.Printf("Verifying wif-config '%s'...", wifConfig.ID())
		if err := verifyWifConfig(connection, wifConfig.ID()); err != nil {
			return true, err
		}
		return false, nil
	}, IamApiRetrySeconds, log); err != nil {
		return fmt.Errorf("Timed out verifying wif-config resources\n"+
			"Please try 'ocm gcp update wif-config %s' again in a few minutes, "+
			"or contact Red Hat support.", wifConfig.ID())
	}

	log.Printf("wif-config '%s' updated successfully.", wifConfig.ID())
	return nil
}
