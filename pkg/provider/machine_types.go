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

	"github.com/openshift-online/ocm-cli/pkg/arguments"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func getMachineTypes(client *cmv1.Client, provider string) (machineTypes []*cmv1.MachineType, err error) {
	collection := client.MachineTypes()
	page := 1
	size := 100
	for {
		var response *cmv1.MachineTypesListResponse
		response, err = collection.List().
			Search(fmt.Sprintf("cloud_provider.id = '%s'", provider)).
			Order("size desc").
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return
		}
		machineTypes = append(machineTypes, response.Items().Slice()...)
		if response.Size() < size {
			break
		}
		page++
	}

	if len(machineTypes) == 0 {
		return nil, fmt.Errorf("No machine types for provider %v", err)
	}
	return
}

func GetMachineTypeOptions(client *cmv1.Client, provider string) (options []arguments.Option, err error) {
	machineTypes, err := getMachineTypes(client, provider)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve machine types: %s", err)
		return
	}

	for _, v := range machineTypes {
		options = append(options, arguments.Option{
			Value:       v.ID(),
			Description: v.Name(),
		})
	}
	return
}
