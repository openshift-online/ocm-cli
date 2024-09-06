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

var (
	// CreateWorkloadIdentityPoolOpts captures the options that affect creation of the workload identity pool
	GenerateScriptOpts = options{
		TargetDir: "",
	}
)

func NewGenerateCommand() *cobra.Command {
	generateScriptCmd := &cobra.Command{
		Use:     "generate [wif-config ID]",
		Short:   "Generate script based on a wif-config",
		Args:    cobra.ExactArgs(1),
		RunE:    generateCreateScriptCmd,
		PreRunE: validationForGenerateCreateScriptCmd,
	}

	generateScriptCmd.PersistentFlags().StringVar(&GenerateScriptOpts.TargetDir, "output-dir", "",
		"Directory to place generated files (defaults to current directory)")

	return generateScriptCmd
}

func validationForGenerateCreateScriptCmd(cmd *cobra.Command, argv []string) error {
	if len(argv) != 1 {
		return fmt.Errorf(
			"Expected exactly one command line parameters containing the id " +
				"of the WIF config.",
		)
	}
	return nil
}

func generateCreateScriptCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		errors.Wrapf(err, "failed to initiate GCP client")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	wifConfigId := argv[0]
	if wifConfigId == "" {
		return fmt.Errorf("WIF config ID is required")
	}

	response, err := connection.ClustersMgmt().V1().GCP().WifConfigs().WifConfig(wifConfigId).Get().Send()
	if err != nil {
		return errors.Wrapf(err, "failed to get wif-config")
	}
	wifConfig := response.Body()

	projectNum, err := gcpClient.ProjectNumberFromId(ctx, wifConfig.Gcp().ProjectId())
	if err != nil {
		return errors.Wrapf(err, "failed to get project number from id")
	}

	log.Printf("Writing script files to %s", GenerateScriptOpts.TargetDir)
	if err := createScript(GenerateScriptOpts.TargetDir, wifConfig, projectNum); err != nil {
		return errors.Wrapf(err, "failed to generate create script")
	}
	if err := createDeleteScript(GenerateScriptOpts.TargetDir, wifConfig); err != nil {
		return errors.Wrapf(err, "failed to generate delete script")
	}
	return nil
}
