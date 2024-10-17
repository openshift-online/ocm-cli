package gcp

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

var (
	// CreateWifConfigOpts captures the options that affect creation of the workload identity configuration
	CreateWifConfigOpts = options{
		Mode:       ModeAuto,
		Name:       "",
		Project:    "",
		RolePrefix: "",
		TargetDir:  "",
	}
)

const (
	poolDescription = "Created by the OLM CLI"
	roleDescription = "Created by the OLM CLI"
)

// NewCreateWorkloadIdentityConfiguration provides the "gcp create wif-config" subcommand
func NewCreateWorkloadIdentityConfiguration() *cobra.Command {
	createWifConfigCmd := &cobra.Command{
		Use:     "wif-config",
		Short:   "Create workload identity configuration",
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
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.RolePrefix, "role-prefix", "",
		"Prefix for naming custom roles")

	createWifConfigCmd.PersistentFlags().StringVarP(
		&CreateWifConfigOpts.Mode,
		"mode",
		"m",
		ModeAuto,
		"How to perform the operation. Valid options are:\n"+
			"auto (default): Resource changes will be automatic applied using the current GCP account\n"+
			"manual: Commands necessary to modify GCP resources will be output to be run manually",
	)
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.TargetDir, "output-dir", "",
		"Directory to place generated files (defaults to current directory)")

	return createWifConfigCmd
}

func validationForCreateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	if err := promptWifDisplayName(); err != nil {
		return err
	}
	if err := promptProjectId(); err != nil {
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
	const wifNameHelp = "The name can be used as the identifier of the WifConfig resource."
	if CreateWifConfigOpts.Name == "" {
		if CreateWifConfigOpts.Interactive {
			prompt := &survey.Input{
				Message: "WifConfig name",
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
	const projectIdHelp = "The GCP Project Id that will be used by the WifConfig"
	if CreateWifConfigOpts.Project == "" {
		if CreateWifConfigOpts.Interactive {
			prompt := &survey.Input{
				Message: "Gcp Project ID",
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

func createWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
	log := log.Default()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	log.Println("Creating workload identity configuration...")
	wifConfig, err := createWorkloadIdentityConfiguration(
		ctx,
		gcpClient,
		CreateWifConfigOpts.Name,
		CreateWifConfigOpts.Project,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create WIF config")
	}

	if CreateWifConfigOpts.Mode == ModeManual {
		log.Printf("Writing script files to %s", CreateWifConfigOpts.TargetDir)

		projectNum, err := gcpClient.ProjectNumberFromId(ctx, wifConfig.Gcp().ProjectId())
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
	return nil
}

func createWorkloadIdentityConfiguration(
	ctx context.Context,
	client gcp.GcpClient,
	displayName string,
	projectId string,
) (*cmv1.WifConfig, error) {
	projectNum, err := client.ProjectNumberFromId(ctx, projectId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get GCP project number from project id")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	wifBuilder := cmv1.NewWifConfig()
	gcpBuilder := cmv1.NewWifGcp().
		ProjectId(projectId).
		ProjectNumber(strconv.FormatInt(projectNum, 10))

	if CreateWifConfigOpts.RolePrefix != "" {
		gcpBuilder.RolePrefix(CreateWifConfigOpts.RolePrefix)
	}
	wifBuilder.Gcp(gcpBuilder)

	wifBuilder.DisplayName(displayName)
	wifConfigInput, err := wifBuilder.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build WIF config")
	}

	response, err := connection.ClustersMgmt().V1().GCP().
		WifConfigs().
		Add().
		Body(wifConfigInput).
		Send()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create WIF config")
	}

	return response.Body(), nil
}
