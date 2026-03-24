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

package cluster

import (
	"fmt"
	"sort"
	"strings"

	goVersion "github.com/hashicorp/go-version"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const prefix = "openshift-v"

func DropOpenshiftVPrefix(v string) string {
	return strings.TrimPrefix(v, prefix)
}

func EnsureOpenshiftVPrefix(v string) string {
	if !strings.HasPrefix(v, prefix) {
		return prefix + v
	}
	return v
}

// GetEnabledVersions returns the versions with enabled=true, and the one that has default=true.
// The returned strings are the IDs without "openshift-v" prefix (e.g. "4.6.0-rc.4-candidate")
// sorted in approximate SemVer order (handling of text parts is somewhat arbitrary).
//
// Parameters:
//   - channel: Y-stream update channel (e.g., "stable-4.19"). Takes precedence over channelGroup.
//   - channelGroup: Channel group name (e.g., "stable"). Used if channel is not provided (deprecated).
//   - gcpMarketplaceEnabled: Filter for GCP marketplace support ("true"/"false").
//   - additionalFilters: Additional filter expressions to apply.
//
// Returns:
//   - versions: List of enabled version IDs without "openshift-v" prefix, sorted by semver.
//   - defaultVersion: The version marked as default.
//   - err: Error if the API request fails.
func GetEnabledVersions(client *cmv1.Client,
	channel string,
	channelGroup string,
	gcpMarketplaceEnabled string,
	additionalFilters string,
) (
	versions []string, defaultVersion string, err error) {
	collection := client.Versions()
	page := 1
	size := 100
	filter := "enabled = 'true'"
	if gcpMarketplaceEnabled != "" {
		filter = fmt.Sprintf("%s AND gcp_marketplace_enabled = '%s'", filter, gcpMarketplaceEnabled)
	}
	// Handle channel and channelGroup parameters
	// Channel format is "{channel_group}-{major}.{minor}" (e.g., "stable-4.19")
	// Extract channel_group from channel since Version API only supports filtering by channel_group
	var effectiveChannelGroup string
	if channel != "" {
		// Prefer channel over channelGroup (channel is the new field)
		// Extract channel group from channel (e.g., "stable" from "stable-4.19")
		parts := strings.Split(channel, "-")
		if len(parts) > 0 {
			effectiveChannelGroup = parts[0]
		}
	} else if channelGroup != "" {
		// Fall back to channelGroup if channel not provided
		effectiveChannelGroup = channelGroup
	}

	if effectiveChannelGroup != "" {
		filter = fmt.Sprintf("%s AND channel_group = '%s'", filter, effectiveChannelGroup)
	}
	if additionalFilters != "" {
		filter = fmt.Sprintf("%s %s", filter, additionalFilters)
	}
	for {
		response, err := collection.List().
			Search(filter).
			Page(page).
			Size(size).
			Send()
		if err != nil {
			return nil, "", err
		}

		for _, version := range response.Items().Slice() {
			short := DropOpenshiftVPrefix(version.ID())
			if version.Enabled() {
				versions = append(versions, short)
			}
			if version.Default() {
				defaultVersion = short
			}
		}

		if response.Size() < size {
			break
		}
		page++
	}

	sort.Slice(versions, func(i, j int) (less bool) {
		s1, s2 := versions[i], versions[j]
		v1, err1 := goVersion.NewVersion(s1)
		v2, err2 := goVersion.NewVersion(s2)
		if err1 != nil || err2 != nil {
			// Fall back to lexicographic comparison.
			return s1 < s2
		}
		return v1.LessThan(v2)
	})
	return versions, defaultVersion, nil
}
