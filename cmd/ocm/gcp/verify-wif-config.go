package gcp

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
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

	// Verify the WIF configuration is valid
	response, err := connection.ClustersMgmt().V1().GCP().WifConfigs().WifConfig(wif.ID()).Status().Get().Send()
	if err != nil {
		return errors.Wrapf(err, "failed to verify wif-config")
	}
	if !response.Body().Configured() {
		err := errors.New(response.Body().Description())
		helpMsg := fmt.Sprintf("Running 'ocm gcp update wif-config' may fix errors related to " +
			"cloud resource misconfiguration.")
		return fmt.Errorf("verification failed with error: %v\n%s", err, helpMsg)
	} else {
		cmd.Println("WIF configuration is valid")
	}

	return nil
}
