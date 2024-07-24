package ocm

import (
	"fmt"
	"os"

	"github.com/openshift-online/ocm-cli/pkg/debug"
	"github.com/openshift-online/ocm-cli/pkg/info"
	conn "github.com/openshift-online/ocm-cli/pkg/ocm/connection-builder"
	"github.com/openshift-online/ocm-cli/pkg/properties"
)

func NewConnection() *conn.ConnectionBuilder {
	connection := conn.NewConnection()
	connection = connection.AsAgent("OCM-CLI/" + info.Version)

	// overwrite the config URL if the environment variable is set
	if overrideUrl := os.Getenv(properties.URLEnvKey); overrideUrl != "" {
		if debug.Enabled() {
			fmt.Fprintf(os.Stderr, "INFO: %s is overridden via environment variable. This functionality is considered tech preview and may cause unexpected issues.\n", properties.URLEnvKey)                                          //nolint:lll
			fmt.Fprintf(os.Stderr, "      If you experience issues while %s is set, unset the %s environment variable and attempt to log in directly to the desired OCM environment.\n\n", properties.URLEnvKey, properties.URLEnvKey) //nolint:lll
		}
		connection = connection.WithApiUrl(overrideUrl)
	}

	return connection
}
