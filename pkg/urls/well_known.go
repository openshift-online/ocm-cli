/*
Copyright (c) 2021 Red Hat, Inc.

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

// This file contains constants for well known URLs.

package urls

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/config"
	sdk "github.com/openshift-online/ocm-sdk-go"
)

// OfflineTokenPage is the URL of the page used to generate offline access tokens.
const OfflineTokenPage = "https://console.redhat.com/openshift/token" // #nosec G101

const (
	OCMProductionURL  = "https://api.openshift.com"
	OCMStagingURL     = "https://api.stage.openshift.com"
	OCMIntegrationURL = "https://api.integration.openshift.com"
)

var OCMURLAliases = map[string]string{
	"production":  OCMProductionURL,
	"prod":        OCMProductionURL,
	"prd":         OCMProductionURL,
	"staging":     OCMStagingURL,
	"stage":       OCMStagingURL,
	"stg":         OCMStagingURL,
	"integration": OCMIntegrationURL,
	"int":         OCMIntegrationURL,
}

func ValidOCMUrlAliases() []string {
	keys := make([]string, 0, len(OCMURLAliases))
	for k := range OCMURLAliases {
		keys = append(keys, k)
	}
	return keys
}

// URL Precedent (from highest priority to lowest priority):
//  1. runtime `--url` cli arg (key found in `urlAliases`)
//  2. runtime `--url` cli arg (non-empty string)
//  3. config file `URL` value (non-empty string)
//  4. sdk.DefaultURL
//
// Finally, it will try to url.ParseRequestURI the resolved URL to make sure it's a valid URL.
func ResolveGatewayURL(optionalParsedCliFlagValue string, optionalParsedConfig *config.Config) (string, error) {
	gatewayURL := sdk.DefaultURL
	source := "default"
	if optionalParsedCliFlagValue != "" {
		gatewayURL = optionalParsedCliFlagValue
		source = "flag"
		if _, ok := OCMURLAliases[optionalParsedCliFlagValue]; ok {
			gatewayURL = OCMURLAliases[optionalParsedCliFlagValue]
		}
	} else if optionalParsedConfig != nil && optionalParsedConfig.URL != "" {
		// re-use the URL from the config file
		gatewayURL = optionalParsedConfig.URL
		source = "config"
	}

	url, err := url.ParseRequestURI(gatewayURL)
	if err != nil {
		return "", fmt.Errorf(
			"%w\n\nURL Source: %s\nExpected an absolute URI/path (e.g. %s) or a case-sensitive alias, one of: [%s]",
			err, source, sdk.DefaultURL, strings.Join(ValidOCMUrlAliases(), ", "))
	}

	return url.String(), nil
}
