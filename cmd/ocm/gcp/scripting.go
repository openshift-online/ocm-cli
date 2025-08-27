package gcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const bashShebang = "#!/bin/bash\n"

func createCreateScript(targetDir string, wifConfig *cmv1.WifConfig, projectNum int64) error {
	// Write the script content to the path
	scriptContent := generateCreateScriptContent(wifConfig, projectNum)
	err := os.WriteFile(filepath.Join(targetDir, "create.sh"), []byte(scriptContent), 0600)
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

func createUpdateScript(targetDir string, wifConfig *cmv1.WifConfig, projectNum int64) error {
	// Write the script content to the path
	scriptContent := generateUpdateScriptContent(wifConfig, projectNum)
	err := os.WriteFile(filepath.Join(targetDir, "apply.sh"), []byte(scriptContent), 0600)
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
	scriptContent := bashShebang

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
		serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", sa.ServiceAccountId(), project)
		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts delete %s --project=%s\n",
			serviceAccountEmail, project))
	}
	return sb.String()
}

func deleteIdentityPoolScriptContent(wifConfig *cmv1.WifConfig) string {
	pool := wifConfig.Gcp().WorkloadIdentityPool()
	projectId := getFederatedProjectId(wifConfig)

	// Delete the workload identity pool
	return fmt.Sprintf(`
# Delete the workload identity pool
gcloud iam workload-identity-pools delete %s --project=%s --location=global
`, pool.PoolId(), projectId)
}

func generateCreateScriptContent(wifConfig *cmv1.WifConfig, projectNum int64) string {
	scriptContent := bashShebang

	// Create a script to create the workload identity pool
	scriptContent += createIdentityPoolScriptContent(wifConfig)

	// Append the script to create the identity provider
	scriptContent += createIdentityProviderScriptContent(wifConfig)

	// Append the script to create the service accounts
	scriptContent += createServiceAccountScriptContent(wifConfig, projectNum)

	scriptContent += grantSupportAccessScriptContent(wifConfig)

	return scriptContent
}

func generateUpdateScriptContent(wifConfig *cmv1.WifConfig, projectNum int64) string {
	scriptContent := bashShebang

	// Create a script to create the workload identity pool
	scriptContent += createIdentityPoolScriptContent(wifConfig)

	// Append the script to create the identity provider
	scriptContent += createIdentityProviderScriptContent(wifConfig)

	// Append the script to create/update the service accounts
	scriptContent += updateServiceAccountScriptContent(wifConfig, projectNum)

	scriptContent += grantSupportAccessScriptContent(wifConfig)

	return scriptContent
}

func createIdentityPoolScriptContent(wifConfig *cmv1.WifConfig) string {
	name := wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	description := fmt.Sprintf(wifDescription, wifConfig.DisplayName())
	project := getFederatedProjectId(wifConfig)

	return fmt.Sprintf(`
# Create workload identity pool:
gcloud iam workload-identity-pools create %s \
	--project=%s \
	--location=global \
	--description="%s" \
	--display-name="%s"
`, name, project, description, name)
}

func createIdentityProviderScriptContent(wifConfig *cmv1.WifConfig) string {
	poolId := wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	audiences := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().AllowedAudiences()
	issuerUrl := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IssuerUrl()
	providerId := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IdentityProviderId()
	description := fmt.Sprintf(wifDescription, wifConfig.DisplayName())
	projectId := getFederatedProjectId(wifConfig)

	return fmt.Sprintf(`
# Create workload identity provider:
gcloud iam workload-identity-pools providers create-oidc %s \
	--display-name="%s" \
	--description="%s" \
	--location=global \
	--issuer-uri="%s" \
	--jwk-json-path="jwk.json" \
	--allowed-audiences="%s" \
	--attribute-mapping="google.subject=assertion.sub" \
	--workload-identity-pool=%s \
	--project=%s
`, providerId, providerId, description, issuerUrl, strings.Join(audiences, ","), poolId, projectId)
}

// This returns the gcloud commands to create a service account, bind roles, and grant access
// to the workload identity pool
func createServiceAccountScriptContent(wifConfig *cmv1.WifConfig, projectNum int64) string {
	// For each service account, create a service account and bind it to the workload identity pool
	var sb strings.Builder

	sb.WriteString("\n# Create service accounts:\n")
	sb.WriteString(createServiceAccountScript(wifConfig))

	sb.WriteString("\n# Create custom roles for service accounts:\n")
	sb.WriteString(createCustomRoleScript(wifConfig))

	sb.WriteString("\n# Bind roles to service accounts:\n")
	sb.WriteString(addRoleBindingsScript(wifConfig))

	sb.WriteString("\n# Grant access to service accounts:\n")
	sb.WriteString(grantServiceAccountAccessScript(wifConfig, projectNum))

	return sb.String()
}

func updateServiceAccountScriptContent(wifConfig *cmv1.WifConfig, projectNum int64) string {
	// For each service account, create a service account and bind it to the workload identity pool
	var sb strings.Builder

	sb.WriteString("\n# Create service accounts:\n")
	sb.WriteString(createServiceAccountScript(wifConfig))

	sb.WriteString("\n# Create custom roles for service accounts:\n")
	sb.WriteString(createCustomRoleScript(wifConfig))

	sb.WriteString("\n# Update custom roles for service accounts:\n")
	sb.WriteString(updateCustomRolesScript(wifConfig))

	sb.WriteString("\n# Bind roles to service accounts:\n")
	sb.WriteString(addRoleBindingsScript(wifConfig))

	sb.WriteString("\n# Grant access to service accounts:\n")
	sb.WriteString(grantServiceAccountAccessScript(wifConfig, projectNum))

	return sb.String()
}

func createServiceAccountScript(wifConfig *cmv1.WifConfig) string {
	var sb strings.Builder
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		project := wifConfig.Gcp().ProjectId()
		serviceAccountID := sa.ServiceAccountId()
		serviceAccountName := wifConfig.DisplayName() + "-" + serviceAccountID
		description := fmt.Sprintf(wifDescription, wifConfig.DisplayName())
		//nolint:lll
		sb.WriteString(fmt.Sprintf("gcloud iam service-accounts create %s --display-name=%s --description=\"%s\" --project=%s\n",
			serviceAccountID, serviceAccountName, description, project))
	}
	return sb.String()
}

func createCustomRoleScript(wifConfig *cmv1.WifConfig) string {
	var sb strings.Builder
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		for _, role := range sa.Roles() {
			if !role.Predefined() {
				roleId := role.RoleId()
				project := wifConfig.Gcp().ProjectId()
				permissions := strings.Join(role.Permissions(), ",")
				roleName := roleId
				roleDesc := wifRoleDescription
				//nolint:lll
				sb.WriteString(fmt.Sprintf("gcloud iam roles create %s --project=%s --title=%s --description=\"%s\" --stage=GA --permissions=%s\n",
					roleId, project, roleName, roleDesc, permissions))
			}
		}
	}
	return sb.String()
}

func updateCustomRolesScript(wifConfig *cmv1.WifConfig) string {
	var sb strings.Builder
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		for _, role := range sa.Roles() {
			if !role.Predefined() {
				project := wifConfig.Gcp().ProjectId()
				permissions := strings.Join(role.Permissions(), ",")
				//nolint:lll
				sb.WriteString(fmt.Sprintf("gcloud iam roles update %s --project=%s --permissions=%s\n",
					role.RoleId(), project, permissions))
			}
		}
	}
	return sb.String()
}

func addRoleBindingsScript(wifConfig *cmv1.WifConfig) string {
	var sb strings.Builder
	for _, sa := range wifConfig.Gcp().ServiceAccounts() {
		for _, role := range sa.Roles() {
			project := wifConfig.Gcp().ProjectId()
			member := fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", sa.ServiceAccountId(), project)
			var roleResource string
			if role.Predefined() {
				roleResource = fmt.Sprintf("roles/%s", role.RoleId())
			} else {
				roleResource = fmt.Sprintf("projects/%s/roles/%s", project, role.RoleId())
			}
			if role.ResourceBindings() != nil {
				for _, resourceBinding := range role.ResourceBindings() {
					switch resourceBinding.Type() {
					case "iam.serviceAccounts":
						saResourceId := gcp.FmtSaResourceId(resourceBinding.Name(), project)
						sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=%s --role=%s\n",
							saResourceId, member, roleResource))
					default:
						fmt.Printf("Warning: unsupported resource type '%s' for resource '%s'\n",
							resourceBinding.Type(), resourceBinding.Name())
					}
				}
			} else {
				sb.WriteString(fmt.Sprintf("gcloud projects add-iam-policy-binding %s --member=%s --role=%s\n",
					project, member, roleResource))
			}
		}
	}
	return sb.String()
}

func grantServiceAccountAccessScript(wifConfig *cmv1.WifConfig, projectNum int64) string {
	var sb strings.Builder
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
			impersonator := fmt.Sprintf("serviceAccount:%s", wifConfig.Gcp().ImpersonatorEmail())
			//nolint:lll
			sb.WriteString(fmt.Sprintf("gcloud iam service-accounts add-iam-policy-binding %s --member=%s --role=roles/iam.serviceAccountTokenCreator --project=%s\n",
				serviceAccount, impersonator, wifConfig.Gcp().ProjectId()))
		}
	}
	return sb.String()
}

func grantSupportAccessScriptContent(wifConfig *cmv1.WifConfig) string {
	var sb strings.Builder

	roles := wifConfig.Gcp().Support().Roles()
	project := wifConfig.Gcp().ProjectId()
	principal := wifConfig.Gcp().Support().Principal()

	sb.WriteString("\n# Create custom roles for support:\n")
	for _, role := range roles {
		if !role.Predefined() {
			roleId := role.RoleId()
			permissions := strings.Join(role.Permissions(), ",")
			roleName := roleId
			roleDesc := wifRoleDescription
			//nolint:lll
			sb.WriteString(fmt.Sprintf("gcloud iam roles create %s --project=%s --title=%s --description=\"%s\" --stage=GA --permissions=%s\n",
				roleId, project, roleName, roleDesc, permissions))
			//nolint:lll
			sb.WriteString(fmt.Sprintf("gcloud iam roles update %s --project=%s --permissions=%s\n",
				roleId, project, permissions))
		}
	}

	sb.WriteString("\n# Bind roles to support principal:\n")
	for _, role := range roles {
		var roleResource string
		if role.Predefined() {
			roleResource = fmt.Sprintf("roles/%s", role.RoleId())
		} else {
			roleResource = fmt.Sprintf("projects/%s/roles/%s", project, role.RoleId())
		}
		supportGroup := fmt.Sprintf("group:%s", principal)
		sb.WriteString(fmt.Sprintf("gcloud projects add-iam-policy-binding %s --member=%s --role=%s\n",
			project, supportGroup, roleResource))
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
