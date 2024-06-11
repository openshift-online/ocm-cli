package gcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/models"
)

// const (
// 	createServiceAccountCmd       = "gcloud iam service-accounts create %s --display-name=%s --project=%s"
// 	addPolicyBindingForSvcAcctCmd = "gcloud iam service-accounts add-iam-policy-binding <POPULATE_SERVICE_ACCOUNT_EMAIL> --member=%s --role=%s"
// )

func createIdentityPoolScriptContent(spec gcp.WorkloadIdentityPoolSpec) string {
	name := spec.PoolName
	project := spec.ProjectId

	return fmt.Sprintf(`#!/bin/bash

# Create a workload identity pool
gcloud iam workload-identity-pools create %s \
	--project=%s \
	--location=global \
	--description="Workload Identity Pool for %s" \
	--display-name="%s"
`, name, project, poolDescription, name)

	// // Create the output directory if it doesn't exist
	// err := os.MkdirAll("output", 0755)
	// if err != nil {
	// 	return err
	// }

	// // Write the script content to the file
	// err = ioutil.WriteFile("output/createWorkloadIdentityPool.sh", []byte(scriptContent), 0644)
	// if err != nil {
	// 	return err
	// }

	// return nil
}

func createIdentityProviderScriptContent(spec gcp.WorkloadIdentityPoolSpec) string {
	return fmt.Sprintf(`#!/bin/bash

# Create a workload identity provider
gcloud iam workload-identity-pools providers create-oidc %s \
	--display-name="%s" \
	--description=\"%s\" \
	--location=global \
	--issuer-uri="%s" \
	--jwk-json-path="path/to/jwk.json"
	--allowed-audiences="%s" \
	--attribute-mapping=\"google.subject=assertion.sub\" \
	--workload-identity-pool=%s
`, spec.PoolName, spec.PoolName, poolDescription, spec.IssuerUrl, openShiftAudience, spec.PoolName)

}

// This returns the gcloud commands to create a service account, bind roles, and grant access
// to the workload identity pool
func createServiceAccountScriptContent(wifConfig *models.WifConfigOutput) string {
	// For each service account, create a service account and bind it to the workload identity pool
	var sb strings.Builder

	sb.WriteString("# Create service accounts:\n")
	for _, sa := range wifConfig.Status.ServiceAccounts {
		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts create %s --display-name=%s --project=%s\n",
			sa.Id, sa.Id, wifConfig.Spec.ProjectId))
	}
	sb.WriteString("# Bind service account roles:\n")
	for _, sa := range wifConfig.Status.ServiceAccounts {
		for _, role := range sa.Roles {
			sb.WriteString(fmt.Sprintf("gcloud projects add-iam-policy-binding %s --member=serviceAccount:%s --role=%s\n",
				wifConfig.Spec.ProjectId, sa.Id, role.Id))
		}
	}
	sb.WriteString("# Grant access:\n")
	// for _, sa := range wifConfig.Status.ServiceAccounts {
	// 	if sa.AccessMethod == "impersonate" {
	// 		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=serviceAccount:%s --role=roles/iam.workloadIdentityUser\n",
	// 			sa.Id, sa.Id))
	// 	} else if sa.AccessMethod == "wif" {
	// 		// gcloud iam service-accounts add-iam-policy-binding SERVICE_ACCOUNT_ID \
	// 		// --member='serviceAccount:PROJECT_ID.svc.id.goog[WORKLOAD_IDENTITY_POOL_ID/WORKLOAD_IDENTITY_PROVIDER_ID]' \
	// 		// --role='roles/iam.workloadIdentityUser' \
	// 		// --project=PROJECT_ID
	// 		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member='serviceAccount:%s.svc.id.goog[%s/%s]' --role='roles/iam.workloadIdentityUser' --project=%s\n",
	// 	}
	// }
	return sb.String()
}

func createScriptFile(targetDir string, content ...string) error {
	// Concatenate the content strings
	scriptContent := strings.Join(content, "\n")

	// Write the script content to the file
	err := os.WriteFile(filepath.Join(targetDir, "script.sh"), []byte(scriptContent), 0644)
	if err != nil {
		return err
	}

	return nil
}
