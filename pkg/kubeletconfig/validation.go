/*
Copyright (c) 2026 Red Hat

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

package kubeletconfig

import (
	"fmt"

	sdk "github.com/openshift-online/ocm-sdk-go"
)

const (
	MinPodPidsLimit       = 4096
	MaxPodPidsLimit       = 16384
	MaxUnsafePodPidsLimit = 3694303

	bypassPidsLimitCapability = "capability.organization.bypass_pids_limits"
)

// ValidatePodPidsLimit checks whether the requested pod-pids-limit is within the
// allowed range for the caller's organization. Values between MinPodPidsLimit and
// MaxPodPidsLimit are always accepted. Values above MaxPodPidsLimit require the
// organization to hold the bypass_pids_limits capability; if present the upper
// bound is relaxed to MaxUnsafePodPidsLimit.
func ValidatePodPidsLimit(connection *sdk.Connection, requestedPids int) error {
	if requestedPids < MinPodPidsLimit {
		return fmt.Errorf(
			"Invalid value for '--pod-pids-limit': %d. Minimum value is %d",
			requestedPids, MinPodPidsLimit,
		)
	}

	if requestedPids <= MaxPodPidsLimit {
		return nil
	}

	// Value exceeds the standard max — check org capability.
	bypassed, err := isBypassPidsLimitEnabled(connection)
	if err != nil {
		return fmt.Errorf("Failed to check organization capabilities: %v", err)
	}

	if !bypassed {
		return fmt.Errorf(
			"Invalid value for '--pod-pids-limit': %d. Maximum value is %d. "+
				"Contact Red Hat support if you require a higher limit",
			requestedPids, MaxPodPidsLimit,
		)
	}

	if requestedPids > MaxUnsafePodPidsLimit {
		return fmt.Errorf(
			"Invalid value for '--pod-pids-limit': %d. Maximum value is %d",
			requestedPids, MaxUnsafePodPidsLimit,
		)
	}

	return nil
}

// isBypassPidsLimitEnabled returns true if the current user's organization has
// the capability.organization.bypass_pids_limits capability enabled.
func isBypassPidsLimitEnabled(connection *sdk.Connection) (bool, error) {
	// Get the current account to find the organization ID.
	accountResponse, err := connection.AccountsMgmt().V1().CurrentAccount().Get().Send()
	if err != nil {
		return false, fmt.Errorf("Failed to get current account: %v", err)
	}
	org := accountResponse.Body().Organization()
	if org == nil {
		return false, fmt.Errorf("Could not determine organization ID from current account")
	}
	orgID := org.ID()
	if orgID == "" {
		return false, fmt.Errorf("Could not determine organization ID from current account")
	}

	// Search for the specific capability on the organization.
	capResponse, err := connection.AccountsMgmt().V1().Capabilities().List().
		Search(fmt.Sprintf("organization_id = '%s' AND name = '%s'", orgID, bypassPidsLimitCapability)).
		Send()
	if err != nil {
		return false, fmt.Errorf("Failed to list capabilities: %v", err)
	}

	items := capResponse.Items()
	if items == nil {
		return false, nil
	}
	for _, cap := range items.Slice() {
		if cap.Name() == bypassPidsLimitCapability {
			return cap.Value() == "true", nil
		}
	}
	return false, nil
}
