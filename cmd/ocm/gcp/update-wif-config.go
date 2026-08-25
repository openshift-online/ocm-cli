package gcp

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	UpdateWifConfigOpts = options{
		Mode:              ModeAuto,
		TargetDir:         "",
		OpenshiftVersion:  "",
		OpenshiftVersions: "",
		FederatedProject:  "",
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
	updateWifConfigCmd.PersistentFlags().StringVar(
		&UpdateWifConfigOpts.OpenshiftVersions,
		"versions",
		"",
		versionsFlagDescription,
	)

	updateWifConfigCmd.PersistentFlags().StringVar(
		&UpdateWifConfigOpts.FederatedProject,
		"federated-project",
		"",
		federatedProjectFlagDescription,
	)

	addWifConfigFailFastFlag(updateWifConfigCmd)

	return updateWifConfigCmd
}

func validationForUpdateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	var err error

	if UpdateWifConfigOpts.Mode != ModeAuto && UpdateWifConfigOpts.Mode != ModeManual {
		return fmt.Errorf("Invalid mode. Allowed values are %s", Modes)
	}

	versionChanged := cmd.Flags().Changed("version")
	versionsChanged := cmd.Flags().Changed("versions")

	if versionChanged && versionsChanged {
		return fmt.Errorf("Cannot specify both --version and --versions flags")
	}

	if versionsChanged {
		if strings.TrimSpace(UpdateWifConfigOpts.OpenshiftVersions) == "" {
			return fmt.Errorf("--versions cannot be empty or whitespace-only")
		}
		if UpdateWifConfigOpts.Mode != ModeManual {
			return fmt.Errorf("--versions flag requires --mode=manual")
		}
	}

	if UpdateWifConfigOpts.Mode == ModeManual && UpdateWifConfigOpts.TargetDir == "" {
		return fmt.Errorf("--mode=manual requires --output-dir to be specified")
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
	originalWifConfig, err := findWifConfig(connection.ClustersMgmt().V1(), key)
	if err != nil {
		return errors.Wrapf(err, "failed to get wif-config")
	}

	wifBuilder := cmv1.NewWifConfig()
	// Update the WIF configuration
	if UpdateWifConfigOpts.OpenshiftVersion != "" {
		// Additive mode: append the new version to existing templates
		wifTemplate := versionToTemplateID(UpdateWifConfigOpts.OpenshiftVersion)

		existingTemplates, _ := originalWifConfig.GetWifTemplates()
		wifBuilder.WifTemplates(append(existingTemplates, wifTemplate)...)
	} else if UpdateWifConfigOpts.OpenshiftVersions != "" {
		// Declarative mode: replace all templates with the specified versions
		versionsList := strings.Split(UpdateWifConfigOpts.OpenshiftVersions, ",")
		wifTemplates := make([]string, 0, len(versionsList))
		for _, version := range versionsList {
			version = strings.TrimSpace(version)
			if version != "" {
				wifTemplates = append(wifTemplates, versionToTemplateID(version))
			}
		}
		if len(wifTemplates) == 0 {
			return fmt.Errorf("--versions must contain at least one version")
		}
		wifBuilder.WifTemplates(wifTemplates...)
	}

	// Create the client for the GCP API
	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	if UpdateWifConfigOpts.FederatedProject != "" &&
		originalWifConfig.Gcp().FederatedProjectId() != UpdateWifConfigOpts.FederatedProject {
		projectNumInt64, err := gcpClient.ProjectNumberFromId(ctx, UpdateWifConfigOpts.FederatedProject)
		if err != nil {
			return errors.Wrapf(err, "failed to get GCP project number from project id")
		}

		wifBuilder.Gcp(cmv1.NewWifGcp().
			FederatedProjectId(UpdateWifConfigOpts.FederatedProject).
			FederatedProjectNumber(strconv.FormatInt(projectNumInt64, 10)))
	}

	updatedWifConfig, err := wifBuilder.Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create wif-config body")
	}

	resp, err := connection.ClustersMgmt().V1().GCP().WifConfigs().
		WifConfig(originalWifConfig.ID()).Update().Body(updatedWifConfig).Send()
	if err != nil {
		return errors.Wrapf(err, "failed to update wif-config")
	}
	updatedWifConfig = resp.Body()

	if UpdateWifConfigOpts.Mode == ModeManual {
		log.Printf("Writing script files to %s", UpdateWifConfigOpts.TargetDir)
		projectNumInt64, err := strconv.ParseInt(getFederatedProjectNumber(updatedWifConfig), 10, 64)
		if err != nil {
			return errors.Wrapf(err, "failed to parse project number from WifConfig")
		}

		if err := createUpdateScript(
			UpdateWifConfigOpts.TargetDir, projectNumInt64, originalWifConfig, updatedWifConfig,
		); err != nil {
			return errors.Wrapf(err, "failed to generate script files")
		}
		return nil
	}

	// Re-apply WIF resources
	gcpClientWifConfigShim := NewGcpClientWifConfigShim(GcpClientWifConfigShimSpec{
		GcpClient: gcpClient,
		WifConfig: updatedWifConfig,
		FailFast:  wifConfigOptions.FailFast,
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

	retryTimeout := IamApiRetrySeconds
	if wifConfigOptions.FailFast {
		retryTimeout = IamApiFailFastRetrySeconds
	}

	//The IAM API is eventually consistent. If the user created the service
	//accounts needed for cluster deployment within too brief a period, then
	//our backend will not yet have access to it. To avoid confusing error
	//messages being returned, we will verify that the backend can see the
	//resources before we consider the wif-config creation complete.
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		log.Printf("Verifying wif-config '%s'...", updatedWifConfig.ID())
		if err := verifyWifConfig(connection, updatedWifConfig.ID()); err != nil {
			return true, err
		}
		return false, nil
	}, retryTimeout, log); err != nil {
		return fmt.Errorf("Timed out verifying wif-config resources\n"+
			"Please try 'ocm gcp update wif-config %s' again in a few minutes, "+
			"or contact Red Hat support.", updatedWifConfig.ID())
	}

	log.Printf("wif-config '%s' updated successfully.", updatedWifConfig.ID())
	return nil
}
