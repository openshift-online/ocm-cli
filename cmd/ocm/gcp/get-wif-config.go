package gcp

import (
	"fmt"
	"os"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var GetWorkloadIdentityConfigurationOpts struct {
	single    bool
	parameter []string
}

func NewGetWorkloadIdentityConfiguration() *cobra.Command {
	getWorkloadIdentityPoolCmd := &cobra.Command{
		Use:     "wif-config [ID]",
		Short:   "Send a GET request for wif-config.",
		RunE:    getWorkloadIdentityConfigurationCmd,
		PreRunE: validationForGetWorkloadIdentityConfigurationCmd,
		Aliases: []string{"wif-configs"},
	}

	fs := getWorkloadIdentityPoolCmd.Flags()
	arguments.AddParameterFlag(fs, &GetWorkloadIdentityConfigurationOpts.parameter)
	fs.BoolVar(
		&GetWorkloadIdentityConfigurationOpts.single,
		"single",
		false,
		"Return the output as a single line.",
	)

	return getWorkloadIdentityPoolCmd
}

func getWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	var path string
	if len(argv) == 0 {
		path = "/api/clusters_mgmt/v1/gcp/wif_configs"
	} else if len(argv) == 1 {
		id := argv[0]
		path = fmt.Sprintf("/api/clusters_mgmt/v1/gcp/wif_configs/%s", id)
	} else {
		return fmt.Errorf("unexpected number of arguments")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	request := connection.Get().Path(path)
	arguments.ApplyParameterFlag(request, GetWorkloadIdentityConfigurationOpts.parameter)

	resp, err := request.Send()
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
	if len(argv) > 1 {
		return fmt.Errorf("expected at most one command line parameter containing the id of the WIF config")
	}
	return nil
}
