package gcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/googleapis/gax-go/v2/apierror"
	alphaocm "github.com/openshift-online/ocm-cli/pkg/alpha_ocm"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/models"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi"
	iamv1 "google.golang.org/api/iam/v1"
	"google.golang.org/grpc/codes"
)

var (
	// CreateWifConfigOpts captures the options that affect creation of the workload identity configuration
	CreateWifConfigOpts = options{
		DryRun:    false,
		Name:      "",
		Project:   "",
		TargetDir: "",
	}

	//nolint:lll
	impersonatorServiceAccount = "projects/sda-ccs-3/serviceAccounts/osd-impersonator@sda-ccs-3.iam.gserviceaccount.com"
	impersonatorEmail          = "osd-impersonator@sda-ccs-3.iam.gserviceaccount.com"
)

const (
	poolDescription = "Created by the OLM CLI"
	roleDescription = "Created by the OLM CLI"

	openShiftAudience = "openshift"
)

// NewCreateWorkloadIdentityConfiguration provides the "gcp create wif-config" subcommand
func NewCreateWorkloadIdentityConfiguration() *cobra.Command {
	createWifConfigCmd := &cobra.Command{
		Use:              "wif-config",
		Short:            "Create workload identity configuration",
		Run:              createWorkloadIdentityConfigurationCmd,
		PersistentPreRun: validationForCreateWorkloadIdentityConfigurationCmd,
	}

	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.Name, "name", "",
		"User-defined name for all created Google cloud resources")
	createWifConfigCmd.MarkPersistentFlagRequired("name")
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.Project, "project", "",
		"ID of the Google cloud project")
	createWifConfigCmd.MarkPersistentFlagRequired("project")
	createWifConfigCmd.PersistentFlags().BoolVar(&CreateWifConfigOpts.DryRun, "dry-run", false,
		"Skip creating objects, and just save what would have been created into files")
	createWifConfigCmd.PersistentFlags().StringVar(&CreateWifConfigOpts.TargetDir, "output-dir", "",
		"Directory to place generated files (defaults to current directory)")

	return createWifConfigCmd
}

func createWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	ctx := context.Background()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		log.Fatalf("failed to initiate GCP client: %v", err)
	}

	log.Println("Creating workload identity configuration...")
	wifConfig, err := createWorkloadIdentityConfiguration(models.WifConfigInput{
		DisplayName: CreateWifConfigOpts.Name,
		ProjectId:   CreateWifConfigOpts.Project,
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

	if CreateWifConfigOpts.DryRun {
		log.Printf("Writing script files to %s", CreateWifConfigOpts.TargetDir)

		err := createScript(CreateWifConfigOpts.TargetDir, wifConfig)
		if err != nil {
			log.Fatalf("Failed to create script files: %s", err)
		}
		return
	}

	if err = createWorkloadIdentityPool(ctx, gcpClient, poolSpec); err != nil {
		log.Printf("Failed to create workload identity pool: %s", err)
		log.Fatalf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.Metadata.Id)
	}

	if err = createWorkloadIdentityProvider(ctx, gcpClient, poolSpec); err != nil {
		log.Printf("Failed to create workload identity provider: %s", err)
		log.Fatalf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.Metadata.Id)
	}

	if err = createServiceAccounts(ctx, gcpClient, wifConfig); err != nil {
		log.Printf("Failed to create IAM service accounts: %s", err)
		log.Fatalf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.Metadata.Id)
	}

}

func validationForCreateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) {
	if CreateWifConfigOpts.Name == "" {
		log.Fatal("Name is required")
	}
	if CreateWifConfigOpts.Project == "" {
		log.Fatal("Project is required")
	}

	if CreateWifConfigOpts.TargetDir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %s", err)
		}

		CreateWifConfigOpts.TargetDir = pwd
	}

	fPath, err := filepath.Abs(CreateWifConfigOpts.TargetDir)
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

func createWorkloadIdentityPool(ctx context.Context, client gcp.GcpClient,
	spec gcp.WorkloadIdentityPoolSpec) error {
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
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
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

func createWorkloadIdentityProvider(ctx context.Context, client gcp.GcpClient,
	spec gcp.WorkloadIdentityPoolSpec) error {
	//nolint:lll
	providerResource := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s/providers/%s", spec.ProjectId, spec.PoolName, spec.PoolName)
	_, err := client.GetWorkloadIdentityProvider(ctx, providerResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			provider := &iamv1.WorkloadIdentityPoolProvider{
				Name:        spec.PoolName,
				DisplayName: spec.PoolName,
				Description: poolDescription,
				State:       "ACTIVE",
				Disabled:    false,
				Oidc: &iamv1.Oidc{
					AllowedAudiences: []string{openShiftAudience},
					IssuerUri:        spec.IssuerUrl,
					JwksJson:         spec.Jwks,
				},
				AttributeMapping: map[string]string{
					"google.subject": "assertion.sub",
				},
			}

			parent := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s",
				spec.ProjectId, spec.PoolName)
			_, err := client.CreateWorkloadIdentityProvider(ctx, parent, spec.PoolName, provider)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity provider %s", spec.PoolName)
			}
			log.Printf("workload identity provider created with name %s", spec.PoolName)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity provider %s in pool %s",
				spec.PoolName, spec.PoolName)
		}
	} else {
		return errors.Errorf("workload identity provider %s already exists in pool %s", spec.PoolName, spec.PoolName)
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
		serviceAccountID := serviceAccount.Id
		serviceAccountName := wifOutput.Spec.DisplayName + "-" + serviceAccountID
		serviceAccountDesc := poolDescription + " for WIF config " + wifOutput.Spec.DisplayName

		_, err := createServiceAccount(gcpClient, serviceAccountID, serviceAccountName, serviceAccountDesc, projectId, true)
		if err != nil {
			return errors.Wrap(err, "Failed to create IAM service account")
		}
		log.Printf("IAM service account %s created", serviceAccountID)
	}

	// Create roles that aren't predefined
	for _, serviceAccount := range wifOutput.Status.ServiceAccounts {
		for _, role := range serviceAccount.Roles {
			if role.Predefined {
				continue
			}
			roleID := role.Id
			roleName := role.Id
			permissions := role.Permissions
			existingRole, err := GetRole(gcpClient, roleID, projectId)
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
					strings.Contains(gerr.Message, "Requested entity was not found") {
					existingRole, err = CreateRole(gcpClient, permissions, roleName,
						roleID, roleDescription, projectId)
					if err != nil {
						return errors.Wrap(err, fmt.Sprintf("Failed to create %s", roleName))
					}
					log.Printf("Role %s created", roleID)
				} else {
					return errors.Wrap(err, "Failed to check if role exists")
				}
			}
			// Update role if permissions have changed
			if !reflect.DeepEqual(existingRole.IncludedPermissions, permissions) {
				existingRole.IncludedPermissions = permissions
				_, err := UpdateRole(gcpClient, existingRole, roleName)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to update %s", roleName))
				}
				log.Printf("Role %s updated", roleID)
			}
		}
	}

	// Bind roles and grant access
	for _, serviceAccount := range wifOutput.Status.ServiceAccounts {
		serviceAccountID := serviceAccount.Id

		roles := make([]string, 0, len(serviceAccount.Roles))
		for _, role := range serviceAccount.Roles {
			roles = append(roles, fmtRoleResourceId(role))
		}
		member := fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccountID, projectId)
		err := EnsurePolicyBindingsForProject(gcpClient, roles, member, projectId)
		if err != nil {
			return errors.Errorf("Failed to bind roles to service account %s: %s", serviceAccountID, err)
		}

		switch serviceAccount.AccessMethod {
		case "impersonate":
			if err := gcpClient.AttachImpersonator(serviceAccount.Id, projectId,
				impersonatorServiceAccount); err != nil {
				return errors.Wrapf(err, "Failed to attach impersonator to service account %s", serviceAccount.Id)
			}
		case "wif":
			if err := gcpClient.AttachWorkloadIdentityPool(serviceAccount,
				wifOutput.Status.WorkloadIdentityPoolData.PoolId, projectId); err != nil {
				return errors.Wrapf(err, "Failed to attach workload identity pool to service account %s", serviceAccount.Id)
			}
		default:
			log.Printf("Warning: %s is not a supported access type\n", serviceAccount.AccessMethod)
		}
	}

	return nil
}

func createServiceAccount(gcpClient gcp.GcpClient, svcAcctID, svcAcctName, svcAcctDescription,
	projectName string, allowExisting bool) (*adminpb.ServiceAccount, error) {
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
