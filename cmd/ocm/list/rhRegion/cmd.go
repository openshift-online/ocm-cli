package rhRegion

import (
	"fmt"
	sdk "github.com/openshift-online/ocm-sdk-go"

	"github.com/openshift-online/ocm-cli/pkg/config"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "rh-regions",
	Short: "List available OCM regions",
	Long:  "List available OCM regions",
	Example: `  # List all supported OCM regions 
ocm list rh-regions`,
	RunE:   run,
	Hidden: true,
}

func run(cmd *cobra.Command, argv []string) error {
	// Load the configuration file:
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("Can't load config file: %v", err)
	}
	if cfg == nil {
		return fmt.Errorf("Not logged in, run the 'login' command")
	}

	regions, err := sdk.GetRhRegions(cfg.URL)
	if err != nil {
		return fmt.Errorf("Failed to get OCM regions: %w", err)
	}

	for regionName := range regions {
		fmt.Println(regionName)
	}
	return nil

}
