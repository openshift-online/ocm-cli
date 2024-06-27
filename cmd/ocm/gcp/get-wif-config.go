package gcp

import (
	"fmt"
	"log"
	"os"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/urls"
	"github.com/spf13/cobra"
)

var GetWorkloadIdentityConfigurationOpts struct {
	single bool
}

// NewCreateWorkloadIdentityConfiguration provides the "create-wif-config" subcommand
func NewGetWorkloadIdentityConfiguration() *cobra.Command {
	getWorkloadIdentityPoolCmd := &cobra.Command{
		Use:              "wif-config [ID]",
		Short:            "Send a GET request for wif-config.",
		Run:              getWorkloadIdentityConfigurationCmd,
		PersistentPreRun: validationForGetWorkloadIdentityConfigurationCmd,
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

func getWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	id, err := urls.Expand(argv)
	if err != nil {
		log.Fatalf("could not create URI: %v", err)
	}

	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Can't load config file: %v", err)
	}
	if cfg == nil {
		log.Fatalf("Not logged in, run the 'login' command")
	}

	// Check that the configuration has credentials or tokens that don't have expired:
	armed, reason, err := cfg.Armed()
	if err != nil {
		log.Fatalf(err.Error())
	}
	if !armed {
		log.Fatalf("Not logged in, %s, run the 'login' command", reason)
	}

	// Create the connection:
	connection, err := cfg.Connection()
	if err != nil {
		log.Fatalf("Can't create connection: %v", err)
	}
	defer connection.Close()

	resp, err := connection.Get().Path(fmt.Sprintf("/api/clusters_mgmt/v1/gcp/wif_configs/%s", id)).Send()
	if err != nil {
		log.Fatalf("can't send request: %v", err)
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
		log.Fatalf("Can't print body: %v", err)
	}
}

func validationForGetWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	if len(argv) != 1 {
		log.Fatalf("Expected exactly one command line parameter containing the id of the WIF config.")
	}
}
