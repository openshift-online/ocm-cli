package gcp

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/urls"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewDescribeWorkloadIdentityConfiguration provides the "gcp describe wif-config" subcommand
func NewDescribeWorkloadIdentityConfiguration() *cobra.Command {
	describeWorkloadIdentityPoolCmd := &cobra.Command{
		Use:     "wif-config [ID]",
		Short:   "Show details of a wif-config.",
		RunE:    describeWorkloadIdentityConfigurationCmd,
		PreRunE: validationForDescribeWorkloadIdentityConfigurationCmd,
	}

	return describeWorkloadIdentityPoolCmd
}

func describeWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	id, err := urls.Expand(argv)
	if err != nil {
		return errors.Wrapf(err, "could not create URI")
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	response, err := connection.ClustersMgmt().V1().GCP().WifConfigs().WifConfig(id).Get().Send()
	if err != nil {
		return errors.Wrapf(err, "failed to get wif-config")
	}
	wifConfig := response.Body()

	// Print output
	w := tabwriter.NewWriter(os.Stdout, 8, 0, 2, ' ', 0)

	fmt.Fprintf(w, "ID:\t%s\n", wifConfig.ID())
	fmt.Fprintf(w, "Display Name:\t%s\n", wifConfig.DisplayName())
	fmt.Fprintf(w, "Project:\t%s\n", wifConfig.Gcp().ProjectId())
	fmt.Fprintf(w, "Issuer URL:\t%s\n", wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IssuerUrl())

	return w.Flush()
}

func validationForDescribeWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	if len(argv) != 1 {
		return fmt.Errorf("Expected exactly one command line parameters containing the id of the WIF config")
	}
	return nil
}
