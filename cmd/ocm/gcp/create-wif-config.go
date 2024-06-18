package gcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/googleapis/gax-go/v2/apierror"
	alphaocm "github.com/openshift-online/ocm-cli/pkg/alpha_ocm"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/models"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	iamv1 "google.golang.org/api/iam/v1"
	"google.golang.org/grpc/codes"
)

var (
	// CreateWorkloadIdentityPoolOpts captures the options that affect creation of the workload identity pool
	CreateWorkloadIdentityConfigurationOpts = options{
		Name:      "",
		Project:   "",
		TargetDir: "",
	}

	impersonatorServiceAccount = "projects/sda-ccs-3/serviceAccounts/osd-impersonator@sda-ccs-3.iam.gserviceaccount.com"
	impersonatorEmail          = "osd-impersonator@sda-ccs-3.iam.gserviceaccount.com"
)

const (
	poolDescription = "Created by the OLM CLI"

	openShiftAudience = "openshift"
)

// NewCreateWorkloadIdentityConfiguration provides the "create-wif-config" subcommand
func NewCreateWorkloadIdentityConfiguration() *cobra.Command {
	createWorkloadIdentityPoolCmd := &cobra.Command{
		Use:              "wif-config",
		Short:            "Create workload identity configuration",
		Run:              createWorkloadIdentityConfigurationCmd,
		PersistentPreRun: validationForCreateWorkloadIdentityConfigurationCmd,
	}

	createWorkloadIdentityPoolCmd.PersistentFlags().StringVar(&CreateWorkloadIdentityConfigurationOpts.Name, "name", "", "User-defined name for all created Google cloud resources")
	createWorkloadIdentityPoolCmd.MarkPersistentFlagRequired("name")
	createWorkloadIdentityPoolCmd.PersistentFlags().StringVar(&CreateWorkloadIdentityConfigurationOpts.Project, "project", "", "ID of the Google cloud project")
	createWorkloadIdentityPoolCmd.MarkPersistentFlagRequired("project")
	createWorkloadIdentityPoolCmd.PersistentFlags().BoolVar(&CreateWorkloadIdentityConfigurationOpts.DryRun, "dry-run", false, "Skip creating objects, and just save what would have been created into files")
	createWorkloadIdentityPoolCmd.PersistentFlags().StringVar(&CreateWorkloadIdentityConfigurationOpts.TargetDir, "output-dir", "", "Directory to place generated files (defaults to current directory)")

	return createWorkloadIdentityPoolCmd
}

func createWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		log.Fatalf("failed to initiate GCP client: %v", err)
	}

	log.Println("Creating workload identity configuration...")
	wifConfig, err := createWorkloadIdentityConfiguration(models.WifConfigInput{
		DisplayName: CreateWorkloadIdentityConfigurationOpts.Name,
		ProjectId:   CreateWorkloadIdentityConfigurationOpts.Project,
	})
	if err != nil {
		log.Fatalf("failed to create WIF config: %v", err)
	}

	poolSpec := gcp.WorkloadIdentityPoolSpec{
		PoolName:               wifConfig.Status.WorkloadIdentityPoolData.PoolId,
		ProjectId:              wifConfig.Status.WorkloadIdentityPoolData.ProjectId,
		Jwks:                   wifConfig.Status.WorkloadIdentityPoolData.Jwks,
		IssuerUrl:              wifConfig.Status.WorkloadIdentityPoolData.IssuerUrl,
		PoolIdentityProviderId: wifConfig.Status.WorkloadIdentityPoolData.IdentityProviderId,
	}

	if CreateWorkloadIdentityConfigurationOpts.DryRun {
		log.Printf("Writing script files to %s", CreateWorkloadIdentityConfigurationOpts.TargetDir)

		err := createScript(CreateWorkloadIdentityConfigurationOpts.TargetDir, wifConfig)
		if err != nil {
			log.Fatalf("Failed to create script files: %s", err)
		}
		return
	}

	if err = createWorkloadIdentityPool(ctx, gcpClient, poolSpec); err != nil {
		log.Fatalf("Failed to create workload identity pool: %s", err)
	}

	if err = createWorkloadIdentityProvider(ctx, gcpClient, poolSpec); err != nil {
		log.Fatalf("Failed to create workload identity provider: %s", err)
	}

	if err = createServiceAccounts(ctx, gcpClient, wifConfig); err != nil {
		log.Fatalf("Failed to create IAM service accounts: %s", err)
	}

}

func validationForCreateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	if CreateWorkloadIdentityConfigurationOpts.Name == "" {
		log.Fatal("Name is required")
	}
	if CreateWorkloadIdentityConfigurationOpts.Project == "" {
		log.Fatal("Project is required")
	}

	if CreateWorkloadIdentityConfigurationOpts.TargetDir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %s", err)
		}

		CreateWorkloadIdentityConfigurationOpts.TargetDir = pwd
	}

	fPath, err := filepath.Abs(CreateWorkloadIdentityConfigurationOpts.TargetDir)
	if err != nil {
		log.Fatalf("Failed to resolve full path: %s", err)
	}

	sResult, err := os.Stat(fPath)
	if os.IsNotExist(err) {
		log.Fatalf("Directory %s does not exist", fPath)
	}
	if !sResult.IsDir() {
		log.Fatalf("file %s exists and is not a directory", fPath)
	}

}

func createWorkloadIdentityConfiguration(input models.WifConfigInput) (*models.WifConfigOutput, error) {
	ocmClient, err := alphaocm.NewOcmClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create backend client")
	}
	output, err := ocmClient.CreateWifConfig(input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wif config")
	}
	return &output, nil
}

func createWorkloadIdentityPool(ctx context.Context, client gcp.GcpClient, spec gcp.WorkloadIdentityPoolSpec) error {
	name := spec.PoolName
	project := spec.ProjectId

	parentResourceForPool := fmt.Sprintf("projects/%s/locations/global", project)
	poolResource := fmt.Sprintf("%s/workloadIdentityPools/%s", parentResourceForPool, name)
	resp, err := client.GetWorkloadIdentityPool(ctx, poolResource)
	if resp != nil && resp.State == "DELETED" {
		log.Printf("Workload identity pool %s was deleted", name)
		_, err := client.UndeleteWorkloadIdentityPool(ctx, poolResource, &iamv1.UndeleteWorkloadIdentityPoolRequest{})
		if err != nil {
			return errors.Wrapf(err, "failed to undelete workload identity pool %s", name)
		}
	} else if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 && strings.Contains(gerr.Message, "Requested entity was not found") {
			pool := &iamv1.WorkloadIdentityPool{
				Name:        name,
				DisplayName: name,
				Description: poolDescription,
				State:       "ACTIVE",
				Disabled:    false,
			}

			_, err := client.CreateWorkloadIdentityPool(ctx, parentResourceForPool, name, pool)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity pool %s", name)
			}
			log.Printf("Workload identity pool created with name %s", name)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity pool %s", name)
		}
	} else {
		log.Printf("Workload identity pool %s already exists", name)
	}

	return nil
}

func createWorkloadIdentityProvider(ctx context.Context, client gcp.GcpClient, spec gcp.WorkloadIdentityPoolSpec) error {
	providerResource := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s/providers/%s", spec.ProjectId, spec.PoolName, spec.PoolName)
	_, err := client.GetWorkloadIdentityProvider(ctx, providerResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 && strings.Contains(gerr.Message, "Requested entity was not found") {
			provider := &iam.WorkloadIdentityPoolProvider{
				Name:        spec.PoolName,
				DisplayName: spec.PoolName,
				Description: poolDescription,
				State:       "ACTIVE",
				Disabled:    false,
				Oidc: &iam.Oidc{
					AllowedAudiences: []string{openShiftAudience},
					IssuerUri:        spec.IssuerUrl,
					JwksJson:         spec.Jwks,
				},
				AttributeMapping: map[string]string{
					"google.subject": "assertion.sub",
				},
			}

			_, err := client.CreateWorkloadIdentityProvider(ctx, fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", spec.ProjectId, spec.PoolName), spec.PoolName, provider)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity provider %s", spec.PoolName)
			}
			log.Printf("workload identity provider created with name %s", spec.PoolName)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity provider %s in pool %s", spec.PoolName, spec.PoolName)
		}
	} else {
		log.Printf("Workload identity provider %s already exists in pool %s", spec.PoolName, spec.PoolName)
	}

	return nil
}

func createServiceAccounts(ctx context.Context, gcpClient gcp.GcpClient, wifOutput *models.WifConfigOutput) error {
	projectId := wifOutput.Spec.ProjectId
	fmtRoleResourceId := func(role models.Role) string {
		return fmt.Sprintf("roles/%s", role.Id)
	}

	// Create service accounts
	for _, serviceAccount := range wifOutput.Status.ServiceAccounts {
		serviceAccountID := serviceAccount.GetId()
		serviceAccountName := wifOutput.Spec.DisplayName + "-" + serviceAccountID
		serviceAccountDesc := poolDescription + " for WIF config " + wifOutput.Spec.DisplayName

		fmt.Println("Creating service account", serviceAccountID)
		_, err := CreateServiceAccount(gcpClient, serviceAccountID, serviceAccountName, serviceAccountDesc, projectId, true)
		if err != nil {
			return errors.Wrap(err, "Failed to create IAM service account")
		}
		log.Printf("IAM service account %s created", serviceAccountID)
	}

	// Bind roles and grant access
	for _, serviceAccount := range wifOutput.Status.ServiceAccounts {
		serviceAccountID := serviceAccount.GetId()

		fmt.Printf("\t\tBinding roles to %s\n", serviceAccount.Id)
		roles := make([]string, 0, len(serviceAccount.Roles))
		for _, role := range serviceAccount.Roles {
			if !role.Predefined {
				fmt.Printf("Skipping role %q for service account %q as custom roles are not yet supported.", role.Id, serviceAccount.Id)
				continue
			}
			roles = append(roles, fmtRoleResourceId(role))
		}
		err := EnsurePolicyBindingsForProject(gcpClient, roles, fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccountID, projectId), projectId)
		if err != nil {
			log.Fatalf("Failed to bind roles to service account %s: %s", serviceAccountID, err)
		}
		fmt.Printf("\t\tRoles bound to %s\n", serviceAccount.Id)

		fmt.Printf("\t\tGranting access to %s...\n", serviceAccount.Id)
		switch serviceAccount.AccessMethod {
		case "impersonate":
			if err := gcpClient.AttachImpersonator(serviceAccount.Id, projectId, impersonatorServiceAccount); err != nil {
				panic(err)
			}
		case "wif":
			if err := gcpClient.AttachWorkloadIdentityPool(serviceAccount, wifOutput.Status.WorkloadIdentityPoolData.PoolId, projectId); err != nil {
				panic(err)
			}
		default:
			fmt.Printf("Warning: %s is not a supported access type\n", serviceAccount.AccessMethod)
		}
		fmt.Printf("\t\tAccess granted to %s\n", serviceAccount.Id)
	}

	return nil
}

func CreateServiceAccount(gcpClient gcp.GcpClient, svcAcctID, svcAcctName, svcAcctDescription, projectName string, allowExisting bool) (*adminpb.ServiceAccount, error) {
	request := &adminpb.CreateServiceAccountRequest{
		Name:      fmt.Sprintf("projects/%s", projectName),
		AccountId: svcAcctID,
		ServiceAccount: &adminpb.ServiceAccount{
			DisplayName: svcAcctName,
			Description: svcAcctDescription,
		},
	}
	svcAcct, err := gcpClient.CreateServiceAccount(context.TODO(), request)
	if err != nil {
		pApiError, ok := err.(*apierror.APIError)
		if ok {
			if pApiError.GRPCStatus().Code() == codes.AlreadyExists && allowExisting {
				return svcAcct, nil
			}
		}
	}
	return svcAcct, err
}

func generateServiceAccountID(serviceAccount models.ServiceAccount) string {
	serviceAccountID := "z-" + serviceAccount.Id
	if len(serviceAccountID) > 30 {
		serviceAccountID = serviceAccountID[:30]
	}
	return serviceAccountID
}

// EnsurePolicyBindingsForProject ensures that given roles and member, appropriate binding is added to project
func EnsurePolicyBindingsForProject(gcpClient gcp.GcpClient, roles []string, member string, projectName string) error {
	needPolicyUpdate := false

	policy, err := gcpClient.GetProjectIamPolicy(projectName, &cloudresourcemanager.GetIamPolicyRequest{})

	if err != nil {
		return fmt.Errorf("error fetching policy for project: %v", err)
	}

	// Validate that each role exists, and add the policy binding as needed
	for _, definedRole := range roles {
		// Earlier we've verified that the requested roles already exist.

		// Add policy binding
		modified := addPolicyBindingForProject(policy, definedRole, member)
		if modified {
			needPolicyUpdate = true
		}

	}

	if needPolicyUpdate {
		return setProjectIamPolicy(gcpClient, policy, projectName)
	}

	// If we made it this far there were no updates needed
	return nil
}

func addPolicyBindingForProject(policy *cloudresourcemanager.Policy, roleName, memberName string) bool {
	for i, binding := range policy.Bindings {
		if binding.Role == roleName {
			return addMemberToBindingForProject(memberName, policy.Bindings[i])
		}
	}

	// if we didn't find an existing binding entry, then make one
	createMemberRoleBindingForProject(policy, roleName, memberName)

	return true
}

func createMemberRoleBindingForProject(policy *cloudresourcemanager.Policy, roleName, memberName string) {
	policy.Bindings = append(policy.Bindings, &cloudresourcemanager.Binding{
		Members: []string{memberName},
		Role:    roleName,
	})
}

// adds member to existing binding. returns bool indicating if an entry was made
func addMemberToBindingForProject(memberName string, binding *cloudresourcemanager.Binding) bool {
	for _, member := range binding.Members {
		if member == memberName {
			// already present
			return false
		}
	}

	binding.Members = append(binding.Members, memberName)
	return true
}

func setProjectIamPolicy(gcpClient gcp.GcpClient, policy *cloudresourcemanager.Policy, projectName string) error {
	policyRequest := &cloudresourcemanager.SetIamPolicyRequest{
		Policy: policy,
	}

	_, err := gcpClient.SetProjectIamPolicy(projectName, policyRequest)
	if err != nil {
		return fmt.Errorf("error setting project policy: %v", err)
	}
	return nil
}
