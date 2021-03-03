package resume

import (
	"github.com/openshift-online/ocm-cli/cmd/ocm/resume/cluster"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "resume [flags] RESOURCE",
	Short: "Resumes a hibernating resource (currently only supported for clusters)",
	Long:  "Resumes a hibernating resource (currently only supported for clusters)",
}

func init() {
	Cmd.AddCommand(cluster.Cmd)
}
