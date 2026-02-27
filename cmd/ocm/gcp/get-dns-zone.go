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

type GetDnsZoneArgs struct {
	single    bool
	parameter []string
}

var getDnsZoneArgs GetDnsZoneArgs

func NewGetDnsZone() *cobra.Command {
	getDnsZoneCmd := &cobra.Command{
		Use:   "dns-zone [ID|BASE DOMAIN]",
		Short: "Retrieve DNS zone resource data.",
		Long: `Retrieve DNS zone resource data.

The DNS zone object returned by this command is in the json format returned
by the OCM backend. It displays all of the data that is associated with the
specified DNS zone object.

Calling this command without an ID specified results in a dump of all
DNS zone objects that belongs to the user's organization.`,
		RunE:    getDnsZoneCmd,
		Aliases: []string{"dns-zones"},
	}

	fs := getDnsZoneCmd.Flags()
	arguments.AddParameterFlag(fs, &getDnsZoneArgs.parameter)
	fs.BoolVar(
		&getDnsZoneArgs.single,
		"single",
		false,
		"Return the output as a single line.",
	)
	return getDnsZoneCmd
}

func getDnsZoneCmd(cmd *cobra.Command, argv []string) error {
	var path string
	if len(argv) == 0 {
		path = "/api/clusters_mgmt/v1/dns_domains"
	} else if len(argv) == 1 {
		id := argv[0]
		path = fmt.Sprintf("/api/clusters_mgmt/v1/dns_domains/%s", id)
	} else {
		return fmt.Errorf("unexpected number of arguments")
	}

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	request := connection.Get().Path(path)
	arguments.ApplyParameterFlag(request, getDnsZoneArgs.parameter)

	resp, err := request.Send()
	if err != nil {
		return errors.Wrapf(err, "can't send request")
	}

	if resp.Status() == 404 {
		if len(argv) == 1 {
			return fmt.Errorf("dns-zone '%s' not found", argv[0])
		}
		return fmt.Errorf("dns-zone not found")
	}

	status := resp.Status()
	body := resp.Bytes()
	if status < 400 {
		if getDnsZoneArgs.single {
			err = dump.Single(os.Stdout, body)
		} else {
			err = dump.Pretty(os.Stdout, body)
		}
		return err
	}
	if getDnsZoneArgs.single {
		err = dump.Single(os.Stderr, body)
	} else {
		err = dump.Pretty(os.Stderr, body)
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("request failed with status %d", status)
}
