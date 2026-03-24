package gcp

import (
	"fmt"
	"strings"
)

func FmtSaResourceId(accountId, projectId string) string {
	return fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", projectId, accountId, projectId)
}

func FmtNetworkResourceId(projectId, networkId string) string {
	// network resource id pattern:
	// https://compute.googleapis.com/compute/v1/projects/<project-id>/global/networks/<network-id>
	return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s", projectId, networkId)
}

func FmtDnsName(domainPrefix, baseDomain string) string {
	// dnsName pattern: <domain-prefix>.<base-domain>.
	dnsName := fmt.Sprintf("%s.%s", domainPrefix, baseDomain)
	return dnsName + "."
}

func FmtDnsZoneName(domainPrefix, baseDomain string) string {
	// dnsName pattern: <domain-prefix>.<base-domain> -> replace "." with "-"
	dnsName := fmt.Sprintf("%s.%s", domainPrefix, baseDomain)
	return strings.ReplaceAll(dnsName, ".", "-")
}
