// Defines helper objects used by the machine pool creation command and unit tests.
//
//go:generate mockgen -source=helpers.go -package=machinepool -destination=mock_helpers.go
package machinepool

import (
	"github.com/openshift-online/ocm-cli/pkg/arguments"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	"github.com/openshift-online/ocm-cli/pkg/provider"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/spf13/pflag"
)

type FlagSet interface {
	Changed(flagId string) bool
	CheckOneOf(flagName string, options []arguments.Option) error
}

type flagSet struct {
	data *pflag.FlagSet
}

func (f *flagSet) Changed(flagId string) bool { return f.data.Changed(flagId) }
func (f *flagSet) CheckOneOf(flagId string, options []arguments.Option) error {
	return arguments.CheckOneOf(f.data, flagId, options)
}

type MachineTypeListGetter interface {
	GetMachineTypeOptions(ocm.Cluster) ([]arguments.Option, error)
}

type machineTypeListGetter struct {
	connection *sdk.Connection
}

func (m *machineTypeListGetter) GetMachineTypeOptions(cluster ocm.Cluster) ([]arguments.Option, error) {
	return provider.GetMachineTypeOptions(
		m.connection.ClustersMgmt().V1(),
		cluster.CloudProviderId(),
		cluster.CcsEnabled(),
	)
}
