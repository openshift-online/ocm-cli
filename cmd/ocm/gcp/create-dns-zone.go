package gcp

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type Args struct {
	Interactive  bool
	DomainPrefix string
	ProjectId    string
	NetworkId    string
	// This is required only for creating dns-zone
	NetworkProjectId string
}

var args Args

// NewCreateDnsZone provides the "gcp create dns-zone" subcommand
func NewCreateDnsZone() *cobra.Command {
	createDnsDomainCmd := &cobra.Command{
		Use:   "dns-zone",
		Short: "Create a DNS zone",
		Long: `Create a DNS zone.

DNS zone objects represent a Cloud DNS Zone in GCP.
A zone is a subtree of the DNS namespace under one administrative responsibility.
A ManagedZone is a resource that represents a DNS zone hosted by the Cloud DNS service.
`,
		Example: `
  # Create a DNS zone
  ocm gcp create dns-zone --domain-prefix "my-domain" --project-id "my-project" 
  --network-id "my-network" --network-project-id "my-network-project"

  # Create a DNS zone with interactive mode
  ocm gcp create dns-zone -i
`,
		PreRunE: validationForCreateDnsDomainCmd,
		RunE:    createDnsDomainCmd,
	}

	arguments.AddInteractiveFlag(
		createDnsDomainCmd.PersistentFlags(),
		&args.Interactive,
	)

	createDnsDomainCmd.PersistentFlags().StringVar(
		&args.DomainPrefix,
		"domain-prefix",
		"",
		domainPrefixFlagDescription,
	)

	createDnsDomainCmd.PersistentFlags().StringVar(
		&args.ProjectId,
		"project-id",
		"",
		dnsZoneProjectIdFlagDescription,
	)

	createDnsDomainCmd.PersistentFlags().StringVar(
		&args.NetworkId,
		"network-id",
		"",
		networkIdFlagDescription,
	)

	createDnsDomainCmd.PersistentFlags().StringVar(
		&args.NetworkProjectId,
		"network-project-id",
		"",
		networkProjectIdFlagDescription,
	)

	return createDnsDomainCmd
}

func validationForCreateDnsDomainCmd(cmd *cobra.Command, argv []string) error {
	if err := promptDomainPrefix(); err != nil {
		return err
	}
	if err := promptDnsZoneProjectId(); err != nil {
		return err
	}
	if err := promptNetworkId(); err != nil {
		return err
	}
	if err := promptNetworkProjectId(); err != nil {
		return err
	}
	return nil
}

func promptDomainPrefix() error {
	const domainPrefixHelp = "The prefix for the DNS zone. "
	if args.DomainPrefix == "" {
		if args.Interactive {
			prompt := &survey.Input{
				Message: "DNS zone prefix:",
				Help:    domainPrefixHelp,
			}
			return survey.AskOne(
				prompt,
				&args.DomainPrefix,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'domain-prefix' is required")
	}
	return nil
}

func promptDnsZoneProjectId() error {
	const projectIdHelp = "The GCP Project Id that will be used by the DNS zone."
	if args.ProjectId == "" {
		if args.Interactive {
			prompt := &survey.Input{
				Message: "Gcp Project ID:",
				Help:    projectIdHelp,
			}
			return survey.AskOne(
				prompt,
				&args.ProjectId,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'project-id' is required")
	}
	return nil
}

func promptNetworkId() error {
	const networkIdHelp = "The ID of the Shared VPC network that will be used by the DNS zone."
	if args.NetworkId == "" {
		if args.Interactive {
			prompt := &survey.Input{
				Message: "Google cloud network ID:",
				Help:    networkIdHelp,
			}
			return survey.AskOne(
				prompt,
				&args.NetworkId,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'network-id' is required")
	}
	return nil
}

func promptNetworkProjectId() error {
	const networkProjectIdHelp = "The ID of the GCP project that is used as network project to VPC Network."
	if args.NetworkProjectId == "" {
		if args.Interactive {
			prompt := &survey.Input{
				Message: "Gcp Project ID that is used as network project to VPC Network:",
				Help:    networkProjectIdHelp,
			}
			return survey.AskOne(
				prompt,
				&args.NetworkProjectId,
				survey.WithValidator(survey.Required),
			)
		}
		return fmt.Errorf("flag 'network-project-id' is required")
	}
	return nil
}

func createDnsDomainCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return errors.Wrapf(err, "failed to create OCM connection")
	}
	defer connection.Close()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	// Create the DNS domain object in OCM
	dnsDomain, err := createDnsDomain(
		connection,
		args.DomainPrefix,
		args.ProjectId,
		args.NetworkId,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create dns-domain")
	}

	// Create the DNS zone in GCP
	_, err = gcpClient.CreateDnsZone(ctx, dnsDomain, args.NetworkProjectId)
	if err != nil {
		cleanupErr := deleteDnsDomain(connection, dnsDomain.ID())
		if cleanupErr != nil {
			return errors.Wrapf(
				err,
				"failed to create dns-zone and failed to rollback dns-domain '%s': %v",
				dnsDomain.ID(),
				cleanupErr,
			)
		}
		return errors.Wrapf(err, "failed to create dns-zone")
	}

	// Describe the DNS zone
	if err := describeDnsZoneCmd(cmd, []string{dnsDomain.ID()}); err != nil {
		return errors.Wrap(err, "dns-zone created but failed to describe")
	}

	return nil
}

func createDnsDomain(
	connection *sdk.Connection,
	domainPrefix string,
	projectId string,
	networkId string,
) (*cmv1.DNSDomain, error) {

	dnsDomainBuilder := cmv1.NewDNSDomain().
		CloudProvider(ProviderGCP).
		ClusterArch(ClusterArchClassic)

	gcpDnsDomainBuilder := cmv1.NewGcpDnsDomain().
		DomainPrefix(domainPrefix).
		ProjectId(projectId).
		NetworkId(networkId)

	dnsDomainBuilder.Gcp(gcpDnsDomainBuilder)

	dnsDomainInput, err := dnsDomainBuilder.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build dns-domain")
	}

	response, err := connection.ClustersMgmt().V1().
		DNSDomains().
		Add().
		Body(dnsDomainInput).
		Send()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dns-domain")
	}

	return response.Body(), nil
}
