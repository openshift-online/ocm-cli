package rhRegion

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	sdk "github.com/openshift-online/ocm-sdk-go"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/openshift-online/ocm-cli/pkg/urls"
	"github.com/spf13/cobra"
)

var args struct {
	discoveryURL string
}

var Cmd = &cobra.Command{
	Use:   "rh-regions",
	Short: "List available OCM regions",
	Long:  "List available OCM regions",
	Example: `  # List all supported OCM regions 
ocm list rh-regions`,
	RunE:   run,
	Hidden: true,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.discoveryURL,
		"discovery-url",
		"",
		"URL of the OCM API gateway. If not provided, will reuse the URL from the configuration "+
			"file or "+sdk.DefaultURL+" as a last resort. The value should be a complete URL "+
			"or a valid URL alias: "+strings.Join(urls.ValidOCMUrlAliases(), ", "),
	)
}

func run(cmd *cobra.Command, argv []string) error {

	cfg, _ := config.Load()

	gatewayURL, err := urls.ResolveGatewayURL(args.discoveryURL, cfg)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Discovery URL: %s\n\n", gatewayURL)
	regions, err := sdk.GetRhRegions(gatewayURL)
	if err != nil {
		return fmt.Errorf("Failed to get OCM regions: %w", err)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintf(writer, "RH Region\t\tGateway URL\n")
	for regionName, region := range regions {
		fmt.Fprintf(writer, "%s\t\t%v\n", regionName, region.URL)
	}

	err = writer.Flush()
	return err

}
