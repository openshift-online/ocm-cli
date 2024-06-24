package gcp

import (
	"log"

	alphaocm "github.com/openshift-online/ocm-cli/pkg/alpha_ocm"
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

	generateScriptCmd.PersistentFlags().StringVar(&GenerateScriptOpts.TargetDir, "output-dir", "", "Directory to place generated files (defaults to current directory)")

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
	// Create the client for the OCM API:
	ocmClient, err := alphaocm.NewOcmClient()
	if err != nil {
		log.Fatalf("failed to create backend client: %v", err)
	}

	wifConfigId := argv[0]
	if wifConfigId == "" {
		log.Fatal("WIF config ID is required")
	}

	wifConfig, err := ocmClient.GetWifConfig(wifConfigId)
	if err != nil {
		log.Fatalf("failed to get wif-config: %v", err)
	}

	log.Printf("Writing script files to %s", GenerateScriptOpts.TargetDir)
	if err := createScript(GenerateScriptOpts.TargetDir, &wifConfig); err != nil {
		log.Fatalf("failed to generate create script: %v", err)
	}
	if err := createDeleteScript(GenerateScriptOpts.TargetDir, &wifConfig); err != nil {
		log.Fatalf("failed to generate delete script: %v", err)
	}
}
