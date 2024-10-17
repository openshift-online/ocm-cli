package provider

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func getWifConfigs(client *cmv1.Client, filter string) (wifConfigs []*cmv1.WifConfig, err error) {
	collection := client.GCP().WifConfigs()
	page := 1
	size := 100
	for {
		var response *cmv1.WifConfigsListResponse
		response, err = collection.List().
			Page(page).
			Size(size).
			Search(filter).
			Send()
		if err != nil {
			return
		}
		wifConfigs = append(wifConfigs, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}

	if len(wifConfigs) == 0 {
		return nil, fmt.Errorf("no WIF configurations available")
	}
	return
}

func GetWifConfigs(client *cmv1.Client) (wifConfigs []*cmv1.WifConfig, err error) {
	return getWifConfigs(client, "")
}

// GetWifConfig returns the WIF configuration where the key is the wif config id or name
func GetWifConfig(client *cmv1.Client, key string) (wifConfig *cmv1.WifConfig, err error) {
	query := fmt.Sprintf(
		"id = '%s' or display_name = '%s'",
		key, key,
	)
	wifs, err := getWifConfigs(client, query)
	if err != nil {
		return nil, err
	}

	if len(wifs) == 0 {
		return nil, fmt.Errorf("WIF configuration with identifier or name '%s' not found", key)
	}
	if len(wifs) > 1 {
		return nil, fmt.Errorf("there are %d WIF configurations found with identifier or name '%s'", len(wifs), key)
	}
	return wifs[0], nil
}

// GetWifConfigNameOptions returns the wif config options for the cluster
// with display name as the value and id as the description
func GetWifConfigNameOptions(client *cmv1.Client) (options []arguments.Option, err error) {
	wifConfigs, err := getWifConfigs(client, "")
	if err != nil {
		err = fmt.Errorf("failed to retrieve WIF configurations: %s", err)
		return
	}

	for _, wc := range wifConfigs {
		options = append(options, arguments.Option{
			Value:       wc.DisplayName(),
			Description: wc.ID(),
		})
	}
	return
}
