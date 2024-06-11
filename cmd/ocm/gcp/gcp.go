package gcp

import (
	"github.com/spf13/cobra"
)

type options struct {
	TargetDir                string
	Region                   string
	Name                     string
	Project                  string
	WorkloadIdentityPool     string
	WorkloadIdentityProvider string
	DryRun                   bool
}

// NewGcpCmd implements the "gcp" subcommand for the credentials provisioning
func NewGcpCmd() *cobra.Command {
	gcpCmd := &cobra.Command{
		Use:   "gcp COMMAND",
		Short: "Perform actions related to GCP WIF",
		Long:  "Manage GCP Workload Identity Federation (WIF) resources.",
		Args:  cobra.MinimumNArgs(1),
	}

	gcpCmd.AddCommand(NewCreateCmd())
	gcpCmd.AddCommand(NewUpdateCmd())
	gcpCmd.AddCommand(NewDeleteCmd())
	gcpCmd.AddCommand(NewGetCmd())
	gcpCmd.AddCommand(NewListCmd())
	gcpCmd.AddCommand(NewDescribeCmd())

	return gcpCmd
}

// NewCreateCmd implements the "create" subcommand
func NewCreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create COMMAND",
		Short: "Create resources",
		Long:  "Create resources.",
		Args:  cobra.MinimumNArgs(1),
	}

	createCmd.AddCommand(NewCreateWorkloadIdentityConfiguration())

	return createCmd
}

// NewUpdateCmd implements the "update" subcommand
func NewUpdateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update COMMAND",
		Short: "Update resources",
		Long:  "Update resources.",
		Args:  cobra.MinimumNArgs(1),
	}
	updateCmd.AddCommand(NewUpdateWorkloadIdentityConfiguration())
	return updateCmd
}

// NewDeleteCmd implements the "delete" subcommand
func NewDeleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete COMMAND",
		Short: "Delete resources",
		Long:  "Delete resources.",
		Args:  cobra.MinimumNArgs(1),
	}
	deleteCmd.AddCommand(NewDeleteWorkloadIdentityConfiguration())
	return deleteCmd
}

// NewGetCmd implements the "get" subcommand
func NewGetCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get COMMAND",
		Short: "Get resources",
		Long:  "Get resources.",
		Args:  cobra.MinimumNArgs(1),
	}
	getCmd.AddCommand(NewGetWorkloadIdentityConfiguration())
	return getCmd
}

// NewListCmd implements the "list" subcommand
func NewListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list COMMAND",
		Short: "List resources",
		Long:  "List resources.",
		Args:  cobra.MinimumNArgs(1),
	}
	listCmd.AddCommand(NewListWorkloadIdentityConfiguration())
	return listCmd
}

// NewDescribeCmd implements the "describe" subcommand
func NewDescribeCmd() *cobra.Command {
	describeCmd := &cobra.Command{
		Use:   "describe COMMAND",
		Short: "Describe resources",
		Long:  "Describe resources.",
		Args:  cobra.MinimumNArgs(1),
	}
	describeCmd.AddCommand(NewDescribeWorkloadIdentityConfiguration())
	return describeCmd
}
