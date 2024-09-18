package gcp

import (
	"context"
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
		Use:     "generate [wif-config ID|Name]",
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
	if err := wifKeyArgCheck(argv); err != nil {
		return err
	}
	return nil
}

func generateCreateScriptCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()
	key := argv[0]

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		errors.Wrapf(err, "failed to initiate GCP client")
	}

	// Verify the WIF configuration exists
	wifConfig, err := findWifConfig(connection.ClustersMgmt().V1(), key)
	if err != nil {
		return errors.Wrapf(err, "failed to get wif-config")
	}

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
