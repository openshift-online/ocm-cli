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

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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
)

const (
	poolDescription = "Created by the OLM CLI"
	roleDescription = "Created by the OLM CLI"
)

// NewCreateWorkloadIdentityConfiguration provides the "gcp create wif-config" subcommand
func NewCreateWorkloadIdentityConfiguration() *cobra.Command {
	createWifConfigCmd := &cobra.Command{
		Use:     "wif-config",
		Short:   "Create workload identity configuration",
		RunE:    createWorkloadIdentityConfigurationCmd,
		PreRunE: validationForCreateWorkloadIdentityConfigurationCmd,
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

func createWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	ctx := context.Background()

	gcpClient, err := gcp.NewGcpClient(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to initiate GCP client")
	}

	log.Println("Creating workload identity configuration...")
	wifConfig, err := createWorkloadIdentityConfiguration(CreateWifConfigOpts.Name, CreateWifConfigOpts.Project)
	if err != nil {
		return errors.Wrapf(err, "failed to create WIF config")
	}

	if CreateWifConfigOpts.DryRun {
		log.Printf("Writing script files to %s", CreateWifConfigOpts.TargetDir)

		projectNum, err := gcpClient.ProjectNumberFromId(wifConfig.Gcp().ProjectId())
		if err != nil {
			return errors.Wrapf(err, "failed to get project number from id")
		}
		err = createScript(CreateWifConfigOpts.TargetDir, wifConfig, projectNum)
		if err != nil {
			return errors.Wrapf(err, "Failed to create script files")
		}
		return nil
	}

	if err = createWorkloadIdentityPool(ctx, gcpClient, wifConfig); err != nil {
		log.Printf("Failed to create workload identity pool: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
	}

	if err = createWorkloadIdentityProvider(ctx, gcpClient, wifConfig); err != nil {
		log.Printf("Failed to create workload identity provider: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
	}

	if err = createServiceAccounts(ctx, gcpClient, wifConfig); err != nil {
		log.Printf("Failed to create IAM service accounts: %s", err)
		return fmt.Errorf("To clean up, run the following command: ocm gcp delete wif-config %s", wifConfig.ID())
	}
	return nil
}

func validationForCreateWorkloadIdentityConfigurationCmd(cmd *cobra.Command, argv []string) error {
	if CreateWifConfigOpts.Name == "" {
		return fmt.Errorf("Name is required")
	}
	if CreateWifConfigOpts.Project == "" {
		return fmt.Errorf("Project is required")
	}

	if CreateWifConfigOpts.TargetDir == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "failed to get current directory")
		}

		CreateWifConfigOpts.TargetDir = pwd
	}

	fPath, err := filepath.Abs(CreateWifConfigOpts.TargetDir)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve full path")
	}

	sResult, err := os.Stat(fPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory %s does not exist", fPath)
	}
	if !sResult.IsDir() {
		return fmt.Errorf("file %s exists and is not a directory", fPath)
	}
	return nil
}

func createWorkloadIdentityConfiguration(displayName, projectId string) (*cmv1.WifConfig, error) {
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create OCM connection")
	}
	defer connection.Close()

	wifBuilder := cmv1.NewWifConfig()
	gcpBuilder := cmv1.NewWifGcp().ProjectId(projectId)

	wifBuilder.DisplayName(displayName)
	wifBuilder.Gcp(gcpBuilder)

	wifConfigInput, err := wifBuilder.Build()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build WIF config")
	}

	response, err := connection.ClustersMgmt().V1().GCP().
		WifConfigs().
		Add().
		Body(wifConfigInput).
		Send()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create WIF config")
	}

	return response.Body(), nil
}

func createWorkloadIdentityPool(ctx context.Context, client gcp.GcpClient,
	wifConfig *cmv1.WifConfig) error {
	poolId := wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	project := wifConfig.Gcp().ProjectId()

	parentResourceForPool := fmt.Sprintf("projects/%s/locations/global", project)
	poolResource := fmt.Sprintf("%s/workloadIdentityPools/%s", parentResourceForPool, poolId)
	resp, err := client.GetWorkloadIdentityPool(ctx, poolResource)
	if resp != nil && resp.State == "DELETED" {
		log.Printf("Workload identity pool %s was deleted", poolId)
		_, err := client.UndeleteWorkloadIdentityPool(ctx, poolResource, &iamv1.UndeleteWorkloadIdentityPoolRequest{})
		if err != nil {
			return errors.Wrapf(err, "failed to undelete workload identity pool %s", poolId)
		}
	} else if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			pool := &iamv1.WorkloadIdentityPool{
				Name:        poolId,
				DisplayName: poolId,
				Description: poolDescription,
				State:       "ACTIVE",
				Disabled:    false,
			}

			_, err := client.CreateWorkloadIdentityPool(ctx, parentResourceForPool, poolId, pool)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity pool %s", poolId)
			}
			log.Printf("Workload identity pool created with name %s", poolId)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity pool %s", poolId)
		}
	} else {
		log.Printf("Workload identity pool %s already exists", poolId)
	}

	return nil
}

func createWorkloadIdentityProvider(ctx context.Context, client gcp.GcpClient,
	wifConfig *cmv1.WifConfig) error {
	projectId := wifConfig.Gcp().ProjectId()
	poolId := wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	jwks := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().Jwks()
	audiences := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().AllowedAudiences()
	issuerUrl := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IssuerUrl()
	providerId := wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IdentityProviderId()

	parent := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", projectId, poolId)
	providerResource := fmt.Sprintf("%s/providers/%s", parent, providerId)

	_, err := client.GetWorkloadIdentityProvider(ctx, providerResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			provider := &iamv1.WorkloadIdentityPoolProvider{
				Name:        providerId,
				DisplayName: providerId,
				Description: poolDescription,
				State:       "ACTIVE",
				Disabled:    false,
				Oidc: &iamv1.Oidc{
					AllowedAudiences: audiences,
					IssuerUri:        issuerUrl,
					JwksJson:         jwks,
				},
				AttributeMapping: map[string]string{
					"google.subject": "assertion.sub",
				},
			}

			_, err := client.CreateWorkloadIdentityProvider(ctx, parent, providerId, provider)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity provider %s", providerId)
			}
			log.Printf("workload identity provider created with name %s", providerId)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity provider %s in pool %s",
				providerId, poolId)
		}
	} else {
		return errors.Errorf("workload identity provider %s already exists in pool %s", providerId, poolId)
	}

	return nil
}

func createServiceAccounts(ctx context.Context, gcpClient gcp.GcpClient, wifConfig *cmv1.WifConfig) error {
	projectId := wifConfig.Gcp().ProjectId()
	fmtRoleResourceId := func(role *cmv1.WifRole) string {
		if role.Predefined() {
			return fmt.Sprintf("roles/%s", role.RoleId())
		} else {
			return fmt.Sprintf("projects/%s/roles/%s", projectId, role.RoleId())
		}
	}

	// Create service accounts
	for _, serviceAccount := range wifConfig.Gcp().ServiceAccounts() {
		serviceAccountID := serviceAccount.ServiceAccountId()
		serviceAccountName := wifConfig.DisplayName() + "-" + serviceAccountID
		serviceAccountDesc := poolDescription + " for WIF config " + wifConfig.DisplayName()

		_, err := createServiceAccount(gcpClient, serviceAccountID, serviceAccountName, serviceAccountDesc, projectId, true)
		if err != nil {
			return errors.Wrap(err, "Failed to create IAM service account")
		}
		log.Printf("IAM service account %s created", serviceAccountID)
	}

	// Create roles that aren't predefined
	for _, serviceAccount := range wifConfig.Gcp().ServiceAccounts() {
		for _, role := range serviceAccount.Roles() {
			if role.Predefined() {
				continue
			}
			roleID := role.RoleId()
			roleTitle := role.RoleId()
			permissions := role.Permissions()
			existingRole, err := GetRole(gcpClient, fmtRoleResourceId(role))
			if err != nil {
				if gerr, ok := err.(*apierror.APIError); ok && gerr.GRPCStatus().Code() == codes.NotFound {
					_, err = CreateRole(gcpClient, permissions, roleTitle,
						roleID, roleDescription, projectId)
					if err != nil {
						return errors.Wrap(err, fmt.Sprintf("Failed to create %s", roleID))
					}
					log.Printf("Role %q created", roleID)
					continue
				} else {
					return errors.Wrap(err, "Failed to check if role exists")
				}
			}

			// Undelete role if it was deleted
			if existingRole.Deleted {
				_, err = UndeleteRole(gcpClient, fmtRoleResourceId(role))
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to undelete custom role %q", roleID))
				}
				existingRole.Deleted = false
				log.Printf("Role %q undeleted", roleID)
			}

			// Update role if permissions have changed
			if !reflect.DeepEqual(existingRole.IncludedPermissions, permissions) {
				existingRole.IncludedPermissions = permissions
				_, err := UpdateRole(gcpClient, existingRole, fmtRoleResourceId(role))
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("Failed to update %s", roleID))
				}
				log.Printf("Role %q updated", roleID)
			}
		}
	}

	// Bind roles and grant access
	for _, serviceAccount := range wifConfig.Gcp().ServiceAccounts() {
		serviceAccountID := serviceAccount.ServiceAccountId()

		roles := make([]string, 0, len(serviceAccount.Roles()))
		for _, role := range serviceAccount.Roles() {
			roles = append(roles, fmtRoleResourceId(role))
		}
		member := fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccountID, projectId)
		err := EnsurePolicyBindingsForProject(gcpClient, roles, member, projectId)
		if err != nil {
			return errors.Errorf("Failed to bind roles to service account %s: %s", serviceAccountID, err)
		}

		switch serviceAccount.AccessMethod() {
		case cmv1.WifAccessMethodImpersonate:
			if err := gcpClient.AttachImpersonator(serviceAccount.ServiceAccountId(), projectId,
				wifConfig.Gcp().ImpersonatorEmail()); err != nil {
				return errors.Wrapf(err, "Failed to attach impersonator to service account %s",
					serviceAccount.ServiceAccountId())
			}
		case cmv1.WifAccessMethodWif:
			if err := gcpClient.AttachWorkloadIdentityPool(serviceAccount,
				wifConfig.Gcp().WorkloadIdentityPool().PoolId(), projectId); err != nil {
				return errors.Wrapf(err, "Failed to attach workload identity pool to service account %s",
					serviceAccount.ServiceAccountId())
			}
		default:
			log.Printf("Warning: %s is not a supported access type\n", serviceAccount.AccessMethod())
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
