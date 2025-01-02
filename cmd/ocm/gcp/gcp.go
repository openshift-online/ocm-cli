package gcp

import (
	"github.com/spf13/cobra"
)

type options struct {
	Interactive              bool
	Mode                     string
	Name                     string
	OpenshiftVersion         string
	Project                  string
	Region                   string
	RolePrefix               string
	TargetDir                string
	WorkloadIdentityPool     string
	WorkloadIdentityProvider string
}

// NewGcpCmd implements the "gcp" subcommand for the credentials provisioning
func NewGcpCmd() *cobra.Command {
	gcpCmd := &cobra.Command{
		Use:   "gcp COMMAND",
		Short: "Manage GCP resources.",
		Long:  "Perform operations related to GCP resources.",
		Args:  cobra.MinimumNArgs(1),
	}

	gcpCmd.AddCommand(NewCreateCmd())
	gcpCmd.AddCommand(NewUpdateCmd())
	gcpCmd.AddCommand(NewDeleteCmd())
	gcpCmd.AddCommand(NewGetCmd())
	gcpCmd.AddCommand(NewListCmd())
	gcpCmd.AddCommand(NewDescribeCmd())
	gcpCmd.AddCommand(NewVerifyCmd())

	return gcpCmd
}

// NewCreateCmd implements the "create" subcommand
func NewCreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create COMMAND",
		Short: "Create resources related to GCP.",
		Long: `Create resources related to GCP.

Deployments, such as OSD-GCP WIF clusters, require resources to be created on
the user's cloud prior to cluster creation. This command set provides the
methods needed to create these resources on behalf of the user.`,
		Args: cobra.MinimumNArgs(1),
	}

	createCmd.AddCommand(NewCreateWorkloadIdentityConfiguration())

	return createCmd
}

// NewUpdateCmd implements the "update" subcommand
func NewUpdateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update COMMAND",
		Short: "Update resources related to GCP.",
		Long: `Update resources related to GCP.

Deployments, such as OSD-GCP WIF clusters, utilize resources that may require
updation between version upgrades. This command set providers the methods
needed to update GCP resources on behalf of the user.`,
		Args: cobra.MinimumNArgs(1),
	}
	updateCmd.AddCommand(NewUpdateWorkloadIdentityConfiguration())
	return updateCmd
}

// NewDeleteCmd implements the "delete" subcommand
func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete COMMAND",
		Short: "Delete resources related to GCP.",
		Long:  "Delete resources related to GCP.",
		Args:  cobra.MinimumNArgs(1),
	}
	deleteCmd.AddCommand(NewDeleteWorkloadIdentityConfiguration())
	return deleteCmd
}

// NewGetCmd implements the "get" subcommand
func NewGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get COMMAND",
		Short: "Get resources related to GCP.",
		Long:  "Get resources related to GCP.",
		Args:  cobra.MinimumNArgs(1),
	}
	getCmd.AddCommand(NewGetWorkloadIdentityConfiguration())
	return getCmd
}

// NewListCmd implements the "list" subcommand
func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list COMMAND",
		Short: "List resources related to GCP.",
		Long:  "List resources related to GCP.",
		Args:  cobra.MinimumNArgs(1),
	}
	listCmd.AddCommand(NewListWorkloadIdentityConfiguration())
	return listCmd
}

// NewDescribeCmd implements the "describe" subcommand
func NewDescribeCmd() *cobra.Command {
	describeCmd := &cobra.Command{
		Use:   "describe COMMAND",
		Short: "Describe resources related to GCP.",
		Long:  "Describe resources related to GCP.",
		Args:  cobra.MinimumNArgs(1),
	}
	describeCmd.AddCommand(NewDescribeWorkloadIdentityConfiguration())
	return describeCmd
}

// NewVerifyCmd implements the "verify" subcommand
func NewVerifyCmd() *cobra.Command {
	verifyCmd := &cobra.Command{
		Use:   "verify COMMAND",
		Short: "Verify resources related to GCP.",
		Long:  "Verify resources related to GCP.",
		Args:  cobra.MinimumNArgs(1),
	}
	verifyCmd.AddCommand(NewVerifyWorkloadIdentityConfiguration())
	return verifyCmd
}
