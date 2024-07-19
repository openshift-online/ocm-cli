package gcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func createScript(targetDir string, wifConfig *cmv1.WifConfig, projectNum int64) error {
	// Write the script content to the path
	scriptContent := generateScriptContent(wifConfig, projectNum)
	err := os.WriteFile(filepath.Join(targetDir, "script.sh"), []byte(scriptContent), 0600)
	if err != nil {
		return err
	}
	// Write jwk json file to the path
	jwkPath := filepath.Join(targetDir, "jwk.json")
	err = os.WriteFile(jwkPath, []byte(wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().Jwks()), 0600)
	if err != nil {
		return err
	}
	return nil
}

func createDeleteScript(targetDir string, wifConfig *cmv1.WifConfig) error {
	// Write the script content to the path
	scriptContent := generateDeleteScriptContent(wifConfig)
	err := os.WriteFile(filepath.Join(targetDir, "delete.sh"), []byte(scriptContent), 0600)
	if err != nil {
		return err
	}
	return nil
}

func generateDeleteScriptContent(wifConfig *cmv1.WifConfig) string {
	scriptContent := "#!/bin/bash\n"

	// Append the script to delete the service accounts
	scriptContent += deleteServiceAccountScriptContent(wifConfig)

	// Append the script to delete the workload identity pool
	scriptContent += deleteIdentityPoolScriptContent(wifConfig)

	return scriptContent
}
func deleteServiceAccountScriptContent(wifConfig *cmv1.WifConfig) string {
	var sb strings.Builder
	sb.WriteString("\n# Delete service accounts:\n")
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		project := wifConfig.Gcp().ProjectId()
		serviceAccountID := sa.ServiceAccountId()
		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts delete %s --project=%s\n",
			serviceAccountID, project))
	}
	return sb.String()
}

func deleteIdentityPoolScriptContent(wifConfig *cmv1.WifConfig) string {
	pool := wifConfig.Gcp().WorkloadIdentityPool()
	// Delete the workload identity pool
	return fmt.Sprintf(`
# Delete the workload identity pool
gcloud iam workload-identity-pools delete %s --project=%s --location=global
`, pool.PoolId(), wifConfig.Gcp().ProjectId())
}

func generateScriptContent(wifConfig *cmv1.WifConfig, projectNum int64) string {
	poolSpec := gcp.WorkloadIdentityPoolSpec{
		PoolName:               wifConfig.Gcp().WorkloadIdentityPool().PoolId(),
		ProjectId:              wifConfig.Gcp().ProjectId(),
		Jwks:                   wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().Jwks(),
		IssuerUrl:              wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IssuerUrl(),
		PoolIdentityProviderId: wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IdentityProviderId(),
	}

	scriptContent := "#!/bin/bash\n"

	// Create a script to create the workload identity pool
	scriptContent += createIdentityPoolScriptContent(poolSpec)

	// Append the script to create the identity provider
	scriptContent += createIdentityProviderScriptContent(poolSpec)

	// Append the script to create the service accounts
	scriptContent += createServiceAccountScriptContent(wifConfig, projectNum)

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
func createServiceAccountScriptContent(wifConfig *cmv1.WifConfig, projectNum int64) string {
	// For each service account, create a service account and bind it to the workload identity pool
	var sb strings.Builder

	sb.WriteString("\n# Create service accounts:\n")
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		project := wifConfig.Gcp().ProjectId()
		serviceAccountID := sa.ServiceAccountId()
		serviceAccountName := wifConfig.DisplayName() + "-" + serviceAccountID
		serviceAccountDesc := poolDescription + " for WIF config " + wifConfig.DisplayName()
		//nolint:lll
		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts create %s --display-name=%s --description=\"%s\" --project=%s\n",
			serviceAccountID, serviceAccountName, serviceAccountDesc, project))
	}
	sb.WriteString("\n# Create roles:\n")
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		for _, role := range sa.Roles() {
			if !role.Predefined() {
				roleId := strings.ReplaceAll(role.RoleId(), "-", "_")
				project := wifConfig.Gcp().ProjectId()
				permissions := strings.Join(role.Permissions(), ",")
				roleName := roleId
				serviceAccountDesc := roleDescription + " for WIF config " + wifConfig.DisplayName()
				//nolint:lll
				sb.WriteString(fmt.Sprintf("gcloud iam roles create %s --project=%s --title=%s --description=\"%s\" --stage=GA --permissions=%s\n",
					roleId, project, roleName, serviceAccountDesc, permissions))
			}
		}
	}
	sb.WriteString("\n# Bind service account roles:\n")
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		for _, role := range sa.Roles() {
			project := wifConfig.Gcp().ProjectId()
			member := fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", sa.ServiceAccountId(), project)
			sb.WriteString(fmt.Sprintf("gcloud projects add-iam-policy-binding %s --member=%s --role=roles/%s\n",
				project, member, role.RoleId()))
		}
	}
	sb.WriteString("\n# Grant access:\n")
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		if sa.AccessMethod() == "wif" {
			project := wifConfig.Gcp().ProjectId()
			serviceAccount := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", sa.ServiceAccountId(), project)
			members := fmtMembers(sa, projectNum,
				wifConfig.Gcp().WorkloadIdentityPool().PoolId())
			for _, member := range members {
				//nolint:lll
				sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=%s --role=roles/iam.workloadIdentityUser --project=%s\n",
					serviceAccount, member, project))
			}
		} else if sa.AccessMethod() == "impersonate" {
			project := wifConfig.Gcp().ProjectId()
			serviceAccount := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", sa.ServiceAccountId(), project)
			impersonator := fmt.Sprintf("serviceAccount:%s", impersonatorEmail)
			//nolint:lll
			sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=%s --role=roles/iam.serviceAccountTokenCreator --project=%s\n",
				serviceAccount, impersonator, wifConfig.Gcp().ProjectId()))
		}
	}
	return sb.String()
}

func fmtMembers(sa *cmv1.WifServiceAccount, projectNum int64, poolId string) []string {
	members := []string{}
	for _, saName := range sa.CredentialRequest().ServiceAccountNames() {
		//nolint:lll
		members = append(members, fmt.Sprintf(
			"principal://iam.googleapis.com/projects/%d/locations/global/workloadIdentityPools/%s/subject/system:serviceaccount:%s:%s",
			projectNum, poolId, sa.CredentialRequest().SecretRef().Namespace(), saName))
	}
	return members
}
