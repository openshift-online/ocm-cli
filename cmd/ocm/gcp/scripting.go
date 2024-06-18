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

func createScript(targetDir string, wifConfig *models.WifConfigOutput) error {
	// Write the script content to the path
	scriptContent := generateScriptContent(wifConfig)
	err := os.WriteFile(filepath.Join(targetDir, "script.sh"), []byte(scriptContent), 0644)
	if err != nil {
		return err
	}
	// Write jwk json file to the path
	jwkPath := filepath.Join(targetDir, "jwk.json")
	err = os.WriteFile(jwkPath, []byte(wifConfig.Status.WorkloadIdentityPoolData.Jwks), 0644)
	if err != nil {
		return err
	}
	return nil
}

func generateScriptContent(wifConfig *models.WifConfigOutput) string {
	poolSpec := gcp.WorkloadIdentityPoolSpec{
		PoolName:               wifConfig.Status.WorkloadIdentityPoolData.PoolId,
		ProjectId:              wifConfig.Status.WorkloadIdentityPoolData.ProjectId,
		Jwks:                   wifConfig.Status.WorkloadIdentityPoolData.Jwks,
		IssuerUrl:              wifConfig.Status.WorkloadIdentityPoolData.IssuerUrl,
		PoolIdentityProviderId: wifConfig.Status.WorkloadIdentityPoolData.IdentityProviderId,
	}

	scriptContent := "#!/bin/bash\n"

	// Create a script to create the workload identity pool
	scriptContent += createIdentityPoolScriptContent(poolSpec)

	// Append the script to create the identity provider
	scriptContent += createIdentityProviderScriptContent(poolSpec)

	// Append the script to create the service accounts
	scriptContent += createServiceAccountScriptContent(wifConfig)

	return scriptContent
}

func createIdentityPoolScriptContent(spec gcp.WorkloadIdentityPoolSpec) string {
	name := spec.PoolName
	project := spec.ProjectId

	return fmt.Sprintf(`
# Create a workload identity pool
gcloud iam workload-identity-pools create %s \
	--project=%s \
	--location=global \
	--description="Workload Identity Pool for %s" \
	--display-name="%s"
`, name, project, poolDescription, name)
}

func createIdentityProviderScriptContent(spec gcp.WorkloadIdentityPoolSpec) string {
	return fmt.Sprintf(`
# Create a workload identity provider
gcloud iam workload-identity-pools providers create-oidc %s \
	--display-name="%s" \
	--description="%s" \
	--location=global \
	--issuer-uri="%s" \
	--jwk-json-path="jwk.json" \
	--allowed-audiences="%s" \
	--attribute-mapping="google.subject=assertion.sub" \
	--workload-identity-pool=%s
`, spec.PoolName, spec.PoolName, poolDescription, spec.IssuerUrl, openShiftAudience, spec.PoolName)
}

// This returns the gcloud commands to create a service account, bind roles, and grant access
// to the workload identity pool
func createServiceAccountScriptContent(wifConfig *models.WifConfigOutput) string {
	// For each service account, create a service account and bind it to the workload identity pool
	var sb strings.Builder

	sb.WriteString("\n# Create service accounts:\n")
	for _, sa := range wifConfig.Status.ServiceAccounts {
		project := wifConfig.Spec.ProjectId
		serviceAccountID := sa.GetId()
		serviceAccountName := wifConfig.Spec.DisplayName + "-" + serviceAccountID
		serviceAccountDesc := poolDescription + " for WIF config " + wifConfig.Spec.DisplayName
		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts create %s --display-name=%s --description=\"%s\" --project=%s\n",
			serviceAccountID, serviceAccountName, serviceAccountDesc, project))
	}
	sb.WriteString("\n# Bind service account roles:\n")
	for _, sa := range wifConfig.Status.ServiceAccounts {
		for _, role := range sa.Roles {
			project := wifConfig.Spec.ProjectId
			member := fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", sa.GetId(), project)
			sb.WriteString(fmt.Sprintf("gcloud projects add-iam-policy-binding %s --member=%s --role=roles/%s\n",
				project, member, role.Id))
		}
	}
	sb.WriteString("\n# Grant access:\n")
	for _, sa := range wifConfig.Status.ServiceAccounts {
		if sa.AccessMethod == "wif" {
			project := wifConfig.Spec.ProjectId
			serviceAccount := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", sa.GetId(), project)
			members := fmtMembers(sa, wifConfig.Status.WorkloadIdentityPoolData.ProjectNumber, wifConfig.Status.WorkloadIdentityPoolData.PoolId)
			for _, member := range members {
				sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=%s --role=roles/iam.workloadIdentityUser --project=%s\n",
					serviceAccount, member, project))
			}
		} else if sa.AccessMethod == "impersonate" {
			// gcloud iam service-accounts add-iam-policy-binding SERVICE_ACCOUNT_EMAIL \
			// --member='serviceAccount:IMPERSONATOR_EMAIL' \
			// --role='roles/iam.serviceAccountTokenCreator' \
			// --project=PROJECT_ID
			project := wifConfig.Spec.ProjectId
			serviceAccount := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", sa.GetId(), project)
			// saResource := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", wifConfig.Spec.ProjectId, sa.Id, wifConfig.Spec.ProjectId)
			impersonator := fmt.Sprintf("serviceAccount:%s", impersonatorEmail)
			sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=%s --role=roles/iam.serviceAccountTokenCreator --project=%s\n",
				serviceAccount, impersonator, wifConfig.Spec.ProjectId))
		}
	}
	return sb.String()
}

func fmtMembers(sa models.ServiceAccount, projectNum int64, poolId string) []string {
	members := []string{}
	for _, saName := range sa.GetServiceAccountNames() {
		members = append(members, fmt.Sprintf(
			"principal://iam.googleapis.com/projects/%d/locations/global/workloadIdentityPools/%s/subject/system:serviceaccount:%s:%s",
			projectNum, poolId, sa.GetSecretNamespace(), saName))
	}
	return members
}
