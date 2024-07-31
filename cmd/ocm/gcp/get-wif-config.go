package gcp

import (
	"fmt"
	"os"

	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/urls"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var GetWorkloadIdentityConfigurationOpts struct {
	single bool
}

// NewCreateWorkloadIdentityConfiguration provides the "create-wif-config" subcommand
func NewGetWorkloadIdentityConfiguration() *cobra.Command {
	getWorkloadIdentityPoolCmd := &cobra.Command{
		Use:     "wif-config [ID]",
		Short:   "Send a GET request for wif-config.",
		RunE:    getWorkloadIdentityConfigurationCmd,
		PreRunE: validationForGetWorkloadIdentityConfigurationCmd,
	}

	fs := getWorkloadIdentityPoolCmd.Flags()
	fs.BoolVar(
		&GetWorkloadIdentityConfigurationOpts.single,
		"single",
		false,
		"Return the output as a single line.",
	)

	return getWorkloadIdentityPoolCmd
}

func getWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	id, err := urls.Expand(argv)
	if err != nil {
		return errors.Wrapf(err, "could not create URI")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	resp, err := connection.Get().Path(fmt.Sprintf("/api/clusters_mgmt/v1/gcp/wif_configs/%s", id)).Send()
	if err != nil {
		return errors.Wrapf(err, "can't send request")
	}
	status := resp.Status()
	body := resp.Bytes()
	if status < 400 {
		if GetWorkloadIdentityConfigurationOpts.single {
			err = dump.Single(os.Stdout, body)
		} else {
			err = dump.Pretty(os.Stdout, body)
		}
	} else {
		if GetWorkloadIdentityConfigurationOpts.single {
			err = dump.Single(os.Stderr, body)
		} else {
			err = dump.Pretty(os.Stderr, body)
		}
	}
	if err != nil {
		return errors.Wrapf(err, "can't print body")
	}
	return nil
}

func validationForGetWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	if len(argv) != 1 {
		return fmt.Errorf("Expected exactly one command line parameter containing the id of the WIF config.")
	}
	return nil
}
