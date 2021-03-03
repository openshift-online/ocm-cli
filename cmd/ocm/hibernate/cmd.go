package hibernate

import (
	"github.com/openshift-online/ocm-cli/cmd/ocm/hibernate/cluster"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "hibernate [flags] RESOURCE",
	Short: "Hibernate a specific resource (currently only supported for clusters)",
	Long:  "Hibernate a specific resource (currently only supported for clusters)",
}

func init() {
	Cmd.AddCommand(cluster.Cmd)
}
