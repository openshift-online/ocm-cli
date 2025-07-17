package cluster

import (
	"fmt"
	"os"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "cluster {NAME|ID|EXTERNAL_ID}",
	Short: "Initiate cluster hibernation",
	Long: "Initiates cluster hibernation. While hibernating a cluster will not consume any cloud provider infrastructure" +
		"but will be counted for quota.",
	RunE: run,
}

func run(cmd *cobra.Command, argv []string) error {
	// Check that there is exactly one cluster name, identifir or external identifier in the
	// command line arguments:
	if len(argv) != 1 {
		fmt.Fprintf(
			os.Stderr,
			"Expected exactly one cluster name, identifier or external identifier "+
				"is required\n",
		)
		os.Exit(1)
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := argv[0]
	if !c.IsValidClusterKey(clusterKey) {
		return fmt.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	// Get the client for the cluster management api
	clusterCollection := connection.ClustersMgmt().V1().Clusters()

	// Verify the cluster exists in OCM.
	cluster, err := c.GetCluster(connection, clusterKey)
	if err != nil {
		return fmt.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
	}

	_, err = ocm.SendTypedAndHandleDeprecation(clusterCollection.Cluster(cluster.ID()).Hibernate())
	if err != nil {
		return err
	}
	return nil
}
