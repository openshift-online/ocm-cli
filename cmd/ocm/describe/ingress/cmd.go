package ingress

import (
	"bytes"
	"fmt"
	"os"

	c "github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/dump"
	i "github.com/openshift-online/ocm-cli/pkg/ingress"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
)

var args struct {
	json       bool
	output     bool
	ingressKey string
}

var Cmd = &cobra.Command{
	Use:   "ingress [flags] {CLUSTER_NAME|CLUSTER_ID|CLUSTER_EXTERNAL_ID} -i ingress_key",
	Short: "Show details of an ingress",
	Long:  "Show details of an ingress identified by name, or identifier",
	RunE:  run,
}

func init() {
	// Add flags to rootCmd:
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.output,
		"output",
		false,
		"Output result into JSON file.",
	)
	flags.BoolVar(
		&args.json,
		"json",
		false,
		"Output the entire JSON structure",
	)
	flags.StringVarP(
		&args.ingressKey,
		"ingress",
		"i",
		"",
		"Ingress identifier",
	)
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
	key := argv[0]
	if !c.IsValidClusterKey(key) {
		fmt.Fprintf(
			os.Stderr,
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores\n",
			key,
		)
		os.Exit(1)
	}
	ingressKey := args.ingressKey
	if ingressKey == "" {
		fmt.Fprintf(
			os.Stderr,
			"Ingress identifier must be supplied\n",
		)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return fmt.Errorf("Failed to create OCM connection: %v", err)
	}
	defer connection.Close()

	cluster, err := c.GetCluster(connection, key)
	if err != nil {
		return fmt.Errorf("Can't retrieve cluster for key '%s': %v", key, err)
	}

	clusterId := cluster.ID()
	response, err := ocm.SendTypedAndHandleDeprecation(connection.ClustersMgmt().V1().
		Clusters().Cluster(clusterId).
		Ingresses().
		List().Page(1).Size(-1))
	if err != nil {
		return err
	}

	ingresses := response.Items().Slice()
	var ingress *cmv1.Ingress
	for _, item := range ingresses {
		if ingressKey == "apps" && item.Default() {
			ingress = item
		}
		if ingressKey == "apps2" && !item.Default() {
			ingress = item
		}
		if item.ID() == ingressKey {
			ingress = item
		}
	}
	if ingress == nil {
		return fmt.Errorf("Failed to get ingress '%s' for cluster '%s'", ingressKey, clusterId)
	}

	if args.output {
		// Create a filename based on cluster name:
		filename := fmt.Sprintf("ingress-%s-%s.json", cluster.ID(), ingress.ID())

		// Attempt to create file:
		myFile, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("Failed to create file: %v", err)
		}

		// Dump encoder content into file:
		err = cmv1.MarshalIngress(ingress, myFile)
		if err != nil {
			return fmt.Errorf("Failed to Marshal ingress into file: %v", err)
		}
	}

	// Get full API response (JSON):
	if args.json {
		// Buffer for pretty output:
		buf := new(bytes.Buffer)
		fmt.Println()

		// Convert cluster to JSON and dump to encoder:
		err = cmv1.MarshalIngress(ingress, buf)
		if err != nil {
			return fmt.Errorf("Failed to Marshal ingress into JSON encoder: %v", err)
		}

		err = dump.Pretty(os.Stdout, buf.Bytes())
		if err != nil {
			return fmt.Errorf("Can't print body: %v", err)
		}

	} else {
		err = i.PrintIngressDescription(ingress, cluster)
		if err != nil {
			return err
		}
	}

	return nil
}
