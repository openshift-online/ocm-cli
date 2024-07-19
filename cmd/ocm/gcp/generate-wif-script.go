package gcp

import (
	"context"
	"log"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
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
		Use:              "generate [wif-config ID]",
		Short:            "Generate script based on a wif-config",
		Args:             cobra.ExactArgs(1),
		Run:              generateCreateScriptCmd,
		PersistentPreRun: validationForGenerateCreateScriptCmd,
	}

	generateScriptCmd.PersistentFlags().StringVar(&GenerateScriptOpts.TargetDir, "output-dir", "",
		"Directory to place generated files (defaults to current directory)")

	return generateScriptCmd
}

func validationForGenerateCreateScriptCmd(cmd *cobra.Command, argv []string) {
	if len(argv) != 1 {
		log.Fatal(
			"Expected exactly one command line parameters containing the id " +
				"of the WIF config.",
		)
	}
}

func generateCreateScriptCmd(cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		log.Fatalf("failed to initiate GCP client: %v", err)
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()

	wifConfigId := argv[0]
	if wifConfigId == "" {
		log.Fatal("WIF config ID is required")
	}

	response, err := connection.ClustersMgmt().V1().WifConfigs().WifConfig(wifConfigId).Get().Send()
	if err != nil {
		log.Fatalf("failed to get wif-config: %v", err)
	}
	wifConfig := response.Body()

	projectNum, err := gcpClient.ProjectNumberFromId(wifConfig.Gcp().ProjectId())
	if err != nil {
		log.Fatalf("failed to get project number from id: %v", err)
	}

	log.Printf("Writing script files to %s", GenerateScriptOpts.TargetDir)
	if err := createScript(GenerateScriptOpts.TargetDir, wifConfig, projectNum); err != nil {
		log.Fatalf("failed to generate create script: %v", err)
	}
	if err := createDeleteScript(GenerateScriptOpts.TargetDir, wifConfig); err != nil {
		log.Fatalf("failed to generate delete script: %v", err)
	}
}
