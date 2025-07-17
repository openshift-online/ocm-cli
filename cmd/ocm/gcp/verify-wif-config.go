package gcp

import (
	"fmt"
	"log"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewVerifyWorkloadIdentityConfiguration provides the "gcp verify wif-config" subcommand
func NewVerifyWorkloadIdentityConfiguration() *cobra.Command {
	verifyWorkloadIdentityCmd := &cobra.Command{
		Use:   "wif-config [ID|Name]",
		Short: "Verify a workload identity federation configuration (wif-config) object.",
		RunE:  verifyWorkloadIdentityConfigurationCmd,
	}

	return verifyWorkloadIdentityCmd
}

func verifyWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
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
	wif, err := findWifConfig(connection.ClustersMgmt().V1(), key)
	if err != nil {
		return errors.Wrapf(err, "failed to get wif-config")
	}

	if err := verifyWifConfig(connection, wif.ID()); err != nil {
		return errors.Wrapf(err, "failed to verify wif-config")
	}
	log.Println("wif-config is valid")
	return nil
}

func verifyWifConfig(
	connection *sdk.Connection,
	wifId string,
) error {
	// Verify the WIF configuration is valid
	response, err := ocm.SendTypedAndHandleDeprecation(connection.
		ClustersMgmt().V1().
		GCP().WifConfigs().WifConfig(wifId).Status().
		Get())
	if err != nil {
		return fmt.Errorf("failed to call wif-config verification: %s", err.Error())
	}

	if !response.Body().Configured() {
		return fmt.Errorf(
			"verification failed with error: %s\n"+
				"Running 'ocm gcp update wif-config' will fix errors related to "+
				"cloud resource misconfiguration.",
			response.Body().Description())
	}
	return nil
}
