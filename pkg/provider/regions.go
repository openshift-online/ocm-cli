/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"

	"github.com/openshift-online/ocm-cli/pkg/cluster"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// GetRegions queries either `aws/available_regions` or `regions` depending on CCS flags.
// Does not filter by .Enabled() flag; whether caller should filter depends on CCS.
func GetRegions(client *cmv1.Client, provider string, ccs cluster.CCS) (regions []*cmv1.CloudRegion, err error) {
	if ccs.Enabled && provider == "aws" {
		// Build cmv1.AWS object to get list of available regions:
		awsCredentials, err := cmv1.NewAWS().
			AccessKeyID(ccs.AWS.AccessKeyID).
			SecretAccessKey(ccs.AWS.SecretAccessKey).
			Build()
		if err != nil {
			return nil, fmt.Errorf("Failed to build AWS credentials: %v", err)
		}

		response, err := ocm.SendTypedAndHandleDeprecation(
			client.CloudProviders().CloudProvider(provider).AvailableRegions().Search().
				Page(1).
				Size(-1).
				Body(awsCredentials))
		if err != nil {
			return nil, err
		}
		regions = response.Items().Slice()
	} else {
		response, err := ocm.SendTypedAndHandleDeprecation(client.CloudProviders().CloudProvider(provider).Regions().List().
			Page(1).
			Size(-1))
		if err != nil {
			return nil, err
		}
		regions = response.Items().Slice()
	}
	return regions, nil
}
