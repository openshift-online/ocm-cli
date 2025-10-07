package gcp

import (
	"context"
	"fmt"
	"log"
	"strconv"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/utils"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

var (
	// CreateWifConfigOpts captures the options that affect creation of the workload identity configuration
	CreateWifConfigOpts = options{
		Mode:             ModeAuto,
		Name:             "",
		Project:          "",
		FederatedProject: "",
		RolePrefix:       "",
		TargetDir:        "",
		OpenshiftVersion: "",
	}
)

const (
	// Description for wif-config-specific WIF resources
	wifDescription = "Created by the OCM CLI for WIF config %s"
	// Description for OpenShift version-specific WIF IAM roles
	wifRoleDescription = "Created by the OCM CLI for Workload Identity Federation on OpenShift"
)

// NewCreateWorkloadIdentityConfiguration provides the "gcp create wif-config" subcommand
func NewCreateWorkloadIdentityConfiguration() *cobra.Command {
	createWifConfigCmd := &cobra.Command{
		Use:   "wif-config",
		Short: "Create a workload identity federation configuration (wif-config) object.",
		Long: `Create a workload identity federation configuration (wif-config) object.

wif-config objects represent a set of GCP resources that are needed in a
deployment of WIF OSD-GCP clusters. These resources include service accounts,
custom roles, role bindings, identity and federated pools. Running this command
in auto-mode will generate these resources on the user's cloud, and create a
wif-config resource within OCM to represent those resources.`,
		PreRunE: validationForCreateWorkloadIdentityConfigurationCmd,
		RunE:    createWorkloadIdentityConfigurationCmd,
	}

	arguments.AddInteractiveFlag(
		createWifConfigCmd.PersistentFlags(),
		&CreateWifConfigOpts.Interactive,
	)

	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.Name, "name", "",
		"User-defined name for all created Google cloud resources")
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.Project, "project", "",
		"ID of the Google cloud project")
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.FederatedProject, "federated-project", "",
		"ID of the Google cloud project that will host the WIF pool")
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.RolePrefix, "role-prefix", "",
		"Prefix for naming custom roles")

	createWifConfigCmd.PersistentFlags().StringVarP(
		&CreateWifConfigOpts.Mode,
		"mode",
		"m",
		ModeAuto,
		modeFlagDescription,
	)
	createWifConfigCmd.PersistentFlags().StringVar(
		&CreateWifConfigOpts.TargetDir,
		"output-dir",
		"",
		targetDirFlagDescription,
	)
	createWifConfigCmd.PersistentFlags().StringVar(
		&CreateWifConfigOpts.OpenshiftVersion,
		"version",
		"",
		versionFlagDescription,
	)

	return createWifConfigCmd
}

func validationForCreateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	if err := promptWifDisplayName(); err != nil {
		return err
	}
	if err := promptProjectId(); err != nil {
		return err
	}
	if err := promptVersion(); err != nil {
		return err
	}
	if err := promptFederatedProjectId(); err != nil {
		return err
	}

	if CreateWifConfigOpts.Mode != ModeAuto && CreateWifConfigOpts.Mode != ModeManual {
		return fmt.Errorf("Invalid mode. Allowed values are %s", Modes)
	}

	var err error
	CreateWifConfigOpts.TargetDir, err = getPathFromFlag(CreateWifConfigOpts.TargetDir)
	if err != nil {
		return err
	}
	return nil
}

func promptWifDisplayName() error {
	const wifNameHelp = "The display name of the wif-config resource."
	if CreateWifConfigOpts.Name == "" {
		if CreateWifConfigOpts.Interactive {
			prompt := &survey.Input{
				Message: "wif-config name:",
				Help:    wifNameHelp,
			}
			return survey.AskOne(
				prompt,
				&CreateWifConfigOpts.Name,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'name' is required")
	}
	return nil
}

func promptProjectId() error {
	const projectIdHelp = "The GCP Project Id that will be used by the wif-config."
	if CreateWifConfigOpts.Project == "" {
		if CreateWifConfigOpts.Interactive {
			prompt := &survey.Input{
				Message: "Gcp Project ID:",
				Help:    projectIdHelp,
			}
			return survey.AskOne(
				prompt,
				&CreateWifConfigOpts.Project,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("Flag 'project' is required")
	}
	return nil
}

func promptVersion() error {
	const versionHelp = "The OCP version to configure the wif-config for. " +
		"Will default to the latest supported version if left unset."
	if CreateWifConfigOpts.OpenshiftVersion == "" {
		if CreateWifConfigOpts.Interactive {
			prompt := &survey.Input{
				Message: "Openshift version:",
				Help:    versionHelp,
			}
			return survey.AskOne(
				prompt,
				&CreateWifConfigOpts.OpenshiftVersion,
			)
		}
	}
	return nil
}

func promptFederatedProjectId() error {
	const federatedProjectIdHelp = "The GCP Project Id that will be used to host the WIF pool." +
		"Leave empty to use the same project as the cluster deployment project."

	if CreateWifConfigOpts.FederatedProject == "" {
		if CreateWifConfigOpts.Interactive {
			prompt := &survey.Input{
				Message: "Gcp Federated Project ID:",
				Help:    federatedProjectIdHelp,
			}
			return survey.AskOne(
				prompt,
				&CreateWifConfigOpts.FederatedProject,
			)
		}
	}
	return nil
}

func createWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
	log := log.Default()

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	wifConfig, err := createWorkloadIdentityConfiguration(
		ctx,
		connection,
		gcpClient,
		CreateWifConfigOpts.Name,
		CreateWifConfigOpts.Project,
		CreateWifConfigOpts.FederatedProject,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create wif-config")
	}

	if CreateWifConfigOpts.Mode == ModeManual {
		log.Printf("Writing script files to %s", CreateWifConfigOpts.TargetDir)

		projectNum, err := gcpClient.ProjectNumberFromId(ctx, wifConfig.Gcp().FederatedProjectId())
		if err != nil {
			return errors.Wrapf(err, "failed to get project number from id")
		}
		err = createCreateScript(CreateWifConfigOpts.TargetDir, wifConfig, projectNum)
		if err != nil {
			return errors.Wrapf(err, "Failed to create script files")
		}
		return nil
	}

	gcpClientWifConfigShim := NewGcpClientWifConfigShim(GcpClientWifConfigShimSpec{
		GcpClient: gcpClient,
		WifConfig: wifConfig,
	})

	if err := gcpClientWifConfigShim.GrantSupportAccess(ctx, log); err != nil {
		log.Printf("Failed to grant support access to project: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
	}

	if err := gcpClientWifConfigShim.CreateWorkloadIdentityPool(ctx, log); err != nil {
		log.Printf("Failed to create workload identity pool: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
	}

	if err = gcpClientWifConfigShim.CreateWorkloadIdentityProvider(ctx, log); err != nil {
		log.Printf("Failed to create workload identity provider: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
	}

	if err = gcpClientWifConfigShim.CreateServiceAccounts(ctx, log); err != nil {
		log.Printf("Failed to create IAM service accounts: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
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
			"Please run 'ocm gcp update wif-config %s' to repair potential misconfigurations "+
			"and to complete the wif-config creation process", wifConfig.ID())
	}

	log.Printf("wif-config '%s' created successfully.", wifConfig.ID())
	return nil
}

func createWorkloadIdentityConfiguration(
	ctx context.Context,
	connection *sdk.Connection,
	client gcp.GcpClient,
	displayName string,
	projectId string,
	federatedProjectId string,
) (*cmv1.WifConfig, error) {

	projectNum, err := client.ProjectNumberFromId(ctx, projectId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get GCP project number from project id")
	}

	var federatedProjectNum int64
	if federatedProjectId == "" {
		federatedProjectId = projectId
		federatedProjectNum = projectNum
	} else {
		federatedProjectNum, err = client.ProjectNumberFromId(ctx, federatedProjectId)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get GCP federated project number from project id")
		}
	}

	wifBuilder := cmv1.NewWifConfig()
	gcpBuilder := cmv1.NewWifGcp().
		ProjectId(projectId).
		ProjectNumber(strconv.FormatInt(projectNum, 10)).
		FederatedProjectId(federatedProjectId).
		FederatedProjectNumber(strconv.FormatInt(federatedProjectNum, 10))

	if CreateWifConfigOpts.RolePrefix != "" {
		gcpBuilder.RolePrefix(CreateWifConfigOpts.RolePrefix)
	}
	wifBuilder.Gcp(gcpBuilder)

	if CreateWifConfigOpts.OpenshiftVersion != "" {
		wifTemplate := versionToTemplateID(CreateWifConfigOpts.OpenshiftVersion)
		wifBuilder.WifTemplates(wifTemplate)
	}

	wifBuilder.DisplayName(displayName)
	wifConfigInput, err := wifBuilder.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build wif-config")
	}

	response, err := connection.ClustersMgmt().V1().GCP().
		WifConfigs().
		Add().
		Body(wifConfigInput).
		Send()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wif-config")
	}

	return response.Body(), nil
}
