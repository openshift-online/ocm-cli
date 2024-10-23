package gcp

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/googleapis/gax-go/v2/apierror"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
	iamv1 "google.golang.org/api/iam/v1"
	"google.golang.org/grpc/codes"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	"github.com/openshift-online/ocm-cli/pkg/utils"
)

const (
	maxRetries   = 10
	retryDelayMs = 500
)

type GcpClientWifConfigShim interface {
	CreateServiceAccounts(ctx context.Context, log *log.Logger) error
	CreateWorkloadIdentityPool(ctx context.Context, log *log.Logger) error
	CreateWorkloadIdentityProvider(ctx context.Context, log *log.Logger) error
	GrantSupportAccess(ctx context.Context, log *log.Logger) error
}

type shim struct {
	wifConfig *cmv1.WifConfig
	gcpClient gcp.GcpClient
}

type GcpClientWifConfigShimSpec struct {
	WifConfig *cmv1.WifConfig
	GcpClient gcp.GcpClient
}

func NewGcpClientWifConfigShim(spec GcpClientWifConfigShimSpec) GcpClientWifConfigShim {
	return &shim{
		wifConfig: spec.WifConfig,
		gcpClient: spec.GcpClient,
	}
}

func (c *shim) CreateWorkloadIdentityPool(
	ctx context.Context,
	log *log.Logger,
) error {
	poolId := c.wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	project := c.wifConfig.Gcp().ProjectId()

	parentResourceForPool := fmt.Sprintf("projects/%s/locations/global", project)
	poolResource := fmt.Sprintf("%s/workloadIdentityPools/%s", parentResourceForPool, poolId)
	resp, err := c.gcpClient.GetWorkloadIdentityPool(ctx, poolResource)
	if resp != nil && resp.State == "DELETED" {
		log.Printf("Workload identity pool %s was deleted", poolId)
		_, err := c.gcpClient.UndeleteWorkloadIdentityPool(
			ctx, poolResource, &iamv1.UndeleteWorkloadIdentityPoolRequest{},
		)
		if err != nil {
			return errors.Wrapf(err, "failed to undelete workload identity pool %s", poolId)
		}
	} else if err != nil {
		description := fmt.Sprintf(wifDescription, c.wifConfig.DisplayName())
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			pool := &iamv1.WorkloadIdentityPool{
				Name:        poolId,
				DisplayName: poolId,
				Description: description,
				State:       "ACTIVE",
				Disabled:    false,
			}

			_, err := c.gcpClient.CreateWorkloadIdentityPool(ctx, parentResourceForPool, poolId, pool)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity pool %s", poolId)
			}
			log.Printf("Workload identity pool created with name %s", poolId)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity pool %s", poolId)
		}
	}

	return nil
}

func (c *shim) CreateWorkloadIdentityProvider(
	ctx context.Context,
	log *log.Logger,
) error {
	projectId := c.wifConfig.Gcp().ProjectId()
	poolId := c.wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	jwks := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().Jwks()
	audiences := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().AllowedAudiences()
	issuerUrl := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IssuerUrl()
	providerId := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IdentityProviderId()

	parent := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", projectId, poolId)
	providerResource := fmt.Sprintf("%s/providers/%s", parent, providerId)

	_, err := c.gcpClient.GetWorkloadIdentityProvider(ctx, providerResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			description := fmt.Sprintf(wifDescription, c.wifConfig.DisplayName())
			provider := &iamv1.WorkloadIdentityPoolProvider{
				Name:        providerId,
				DisplayName: providerId,
				Description: description,
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

			_, err := c.gcpClient.CreateWorkloadIdentityProvider(ctx, parent, providerId, provider)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity provider %s", providerId)
			}
			log.Printf("Workload identity provider created with name %s", providerId)
		} else {
			return errors.Wrapf(err, "failed to check if there is existing workload identity provider %s in pool %s",
				providerId, poolId)
		}
	}

	return nil
}

func (c *shim) CreateServiceAccounts(
	ctx context.Context,
	log *log.Logger,
) error {
	for _, serviceAccount := range c.wifConfig.Gcp().ServiceAccounts() {
		if err := c.createServiceAccount(ctx, log, serviceAccount); err != nil {
			return err
		}
		if err := c.createOrUpdateRoles(ctx, log, serviceAccount.Roles()); err != nil {
			return err
		}
		if err := c.bindRolesToServiceAccount(ctx, serviceAccount); err != nil {
			return err
		}
		if err := c.grantAccessToServiceAccount(ctx, serviceAccount); err != nil {
			return err
		}
	}
	return nil
}

func (c *shim) GrantSupportAccess(
	ctx context.Context,
	log *log.Logger,
) error {
	support := c.wifConfig.Gcp().Support()
	if err := c.createOrUpdateRoles(ctx, log, support.Roles()); err != nil {
		return err
	}
	if err := c.bindRolesToGroup(ctx, support.Principal(), support.Roles()); err != nil {
		return err
	}
	return nil
}

func (c *shim) createServiceAccount(
	ctx context.Context,
	log *log.Logger,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	serviceAccountId := serviceAccount.ServiceAccountId()
	serviceAccountName := c.wifConfig.DisplayName() + "-" + serviceAccountId
	serviceAccountDescription := fmt.Sprintf(wifDescription, c.wifConfig.DisplayName())
	request := &adminpb.CreateServiceAccountRequest{
		Name:      fmt.Sprintf("projects/%s", c.wifConfig.Gcp().ProjectId()),
		AccountId: serviceAccountId,
		ServiceAccount: &adminpb.ServiceAccount{
			DisplayName: serviceAccountName,
			Description: serviceAccountDescription,
		},
	}
	_, err := c.gcpClient.CreateServiceAccount(ctx, request)
	if err != nil {
		pApiError, ok := err.(*apierror.APIError)
		if ok {
			if pApiError.GRPCStatus().Code() == codes.AlreadyExists {
				return nil
			}
		}
	}
	if err != nil {
		return errors.Wrap(err, "Failed to create IAM service account")
	}
	log.Printf("IAM service account %s created", serviceAccountId)
	return nil
}

func (c *shim) createOrUpdateRoles(
	ctx context.Context,
	log *log.Logger,
	roles []*cmv1.WifRole,
) error {
	for _, role := range roles {
		if role.Predefined() {
			continue
		}
		roleID := role.RoleId()
		roleTitle := role.RoleId()
		permissions := role.Permissions()
		existingRole, err := c.getRole(ctx, c.fmtRoleResourceId(role))
		if err != nil {
			if gerr, ok := err.(*apierror.APIError); ok && gerr.GRPCStatus().Code() == codes.NotFound {
				_, err = c.createRole(
					ctx,
					permissions,
					roleTitle,
					roleID,
					wifRoleDescription,
					c.wifConfig.Gcp().ProjectId(),
				)
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
			_, err = c.undeleteRole(ctx, c.fmtRoleResourceId(role))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to undelete custom role %q", roleID))
			}
			existingRole.Deleted = false
			log.Printf("Role %q undeleted", roleID)
		}

		if addedPermissions, needsUpdate := c.missingPermissions(permissions, existingRole.IncludedPermissions); needsUpdate {
			// Add missing permissions
			existingRole.IncludedPermissions = append(existingRole.IncludedPermissions, addedPermissions...)
			sort.Strings(existingRole.IncludedPermissions)

			_, err := c.updateRole(ctx, existingRole, c.fmtRoleResourceId(role))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to update %s", roleID))
			}
			log.Printf("Role %q updated", roleID)
		}
	}
	return nil
}

// missingPermissions returns true if there are new permissions that are not in the existing permissions
// and returns the list of missing permissions
func (c *shim) missingPermissions(
	newPermissions []string,
	existingPermissions []string,
) ([]string, bool) {
	missing := []string{}
	permissionMap := map[string]bool{}
	for _, permission := range existingPermissions {
		permissionMap[permission] = true
	}
	for _, permission := range newPermissions {
		if !permissionMap[permission] {
			missing = append(missing, permission)
		}
	}
	if len(missing) > 0 {
		return missing, true
	} else {
		return missing, false
	}
}

func (c *shim) bindRolesToServiceAccount(
	ctx context.Context,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	serviceAccountId := serviceAccount.ServiceAccountId()
	roles := serviceAccount.Roles()

	// It was found that there is a window of time between when a service
	// account creation call is made that the service account is not available
	// in adjacent API calls. The call is therefore wrapped in retry logic to
	// be robust to these types of synchronization issues.
	return utils.DelayedRetry(func() error {
		return c.bindRolesToPrincipal(
			ctx,
			fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com", serviceAccountId, c.wifConfig.Gcp().ProjectId()),
			roles,
		)
	}, maxRetries, retryDelayMs*time.Millisecond)
}

func (c *shim) bindRolesToGroup(
	ctx context.Context,
	groupEmail string,
	roles []*cmv1.WifRole,
) error {
	return c.bindRolesToPrincipal(
		ctx,
		fmt.Sprintf("group:%s", groupEmail),
		roles,
	)
}

func (c *shim) bindRolesToPrincipal(
	ctx context.Context,
	principal string,
	roles []*cmv1.WifRole,
) error {
	formattedRoles := make([]string, 0, len(roles))
	for _, role := range roles {
		formattedRoles = append(formattedRoles, c.fmtRoleResourceId(role))
	}
	err := c.ensurePolicyBindingsForProject(
		ctx,
		formattedRoles,
		principal,
		c.wifConfig.Gcp().ProjectId(),
	)
	if err != nil {
		return errors.Errorf("Failed to bind roles to principal %s: %s", principal, err)
	}
	return nil
}

func (c *shim) grantAccessToServiceAccount(
	ctx context.Context,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	switch serviceAccount.AccessMethod() {
	case cmv1.WifAccessMethodImpersonate:
		if err := c.gcpClient.AttachImpersonator(
			ctx,
			serviceAccount.ServiceAccountId(),
			c.wifConfig.Gcp().ProjectId(),
			c.wifConfig.Gcp().ImpersonatorEmail(),
		); err != nil {
			return errors.Wrapf(err, "Failed to attach impersonator to service account %s",
				serviceAccount.ServiceAccountId())
		}
	case cmv1.WifAccessMethodWif:
		if err := c.gcpClient.AttachWorkloadIdentityPool(
			ctx,
			serviceAccount,
			c.wifConfig.Gcp().WorkloadIdentityPool().PoolId(),
			c.wifConfig.Gcp().ProjectId(),
		); err != nil {
			return errors.Wrapf(err, "Failed to attach workload identity pool to service account %s",
				serviceAccount.ServiceAccountId())
		}
	case cmv1.WifAccessMethodVm:
		// Service accounts with the "vm" access method require no external access
		return nil
	default:
		log.Printf("Warning: %s is not a supported access type\n", serviceAccount.AccessMethod())
	}
	return nil
}

func (c *shim) fmtRoleResourceId(
	role *cmv1.WifRole,
) string {
	if role.Predefined() {
		return fmt.Sprintf("roles/%s", role.RoleId())
	} else {
		return fmt.Sprintf("projects/%s/roles/%s", c.wifConfig.Gcp().ProjectId(), role.RoleId())
	}
}

// GetRole fetches the role created to satisfy a credentials request.
// Custom roles should follow the format projects/{project}/roles/{role_id}.
func (c *shim) getRole(
	ctx context.Context,
	roleName string,
) (*adminpb.Role, error) {
	role, err := c.gcpClient.GetRole(ctx, &adminpb.GetRoleRequest{
		Name: roleName,
	})
	return role, err
}

// CreateRole creates a new role given permissions
func (c *shim) createRole(
	ctx context.Context,
	permissions []string,
	roleTitle string,
	roleId string,
	roleDescription string,
	projectName string,
) (*adminpb.Role, error) {
	role, err := c.gcpClient.CreateRole(ctx, &adminpb.CreateRoleRequest{
		Role: &adminpb.Role{
			Title:               roleTitle,
			Description:         roleDescription,
			IncludedPermissions: permissions,
			Stage:               adminpb.Role_GA,
		},
		Parent: fmt.Sprintf("projects/%s", projectName),
		RoleId: roleId,
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole updates an existing role given permissions.
// Custom roles should follow the format projects/{project}/roles/{role_id}.
func (c *shim) updateRole(
	ctx context.Context,
	role *adminpb.Role,
	roleName string,
) (*adminpb.Role, error) {
	updated, err := c.gcpClient.UpdateRole(ctx, &adminpb.UpdateRoleRequest{
		Name: roleName,
		Role: role,
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// UndeleteRole undeletes a previously deleted role that has not yet been pruned
func (c *shim) undeleteRole(
	ctx context.Context,
	roleName string,
) (*adminpb.Role, error) {
	role, err := c.gcpClient.UndeleteRole(ctx, &adminpb.UndeleteRoleRequest{
		Name: roleName,
	})
	return role, err
}

// EnsurePolicyBindingsForProject ensures that given roles and member, appropriate binding is added to project.
// Roles should be in the format projects/{project}/roles/{role_id} for custom roles and roles/{role_id}
// for predefined roles.
func (c *shim) ensurePolicyBindingsForProject(
	ctx context.Context,
	roles []string,
	member string,
	projectName string,
) error {
	needPolicyUpdate := false

	policy, err := c.gcpClient.GetProjectIamPolicy(ctx, projectName, &cloudresourcemanager.GetIamPolicyRequest{})

	if err != nil {
		return fmt.Errorf("error fetching policy for project: %v", err)
	}

	// Validate that each role exists, and add the policy binding as needed
	for _, definedRole := range roles {
		// Earlier we've verified that the requested roles already exist.

		// Add policy binding
		modified := c.addPolicyBindingForProject(policy, definedRole, member)
		if modified {
			needPolicyUpdate = true
		}

	}

	if needPolicyUpdate {
		return c.setProjectIamPolicy(ctx, policy)
	}

	// If we made it this far there were no updates needed
	return nil
}

func (c *shim) setProjectIamPolicy(
	ctx context.Context,
	policy *cloudresourcemanager.Policy,
) error {
	_, err := c.gcpClient.SetProjectIamPolicy(
		ctx,
		c.wifConfig.Gcp().ProjectId(),
		&cloudresourcemanager.SetIamPolicyRequest{
			Policy: policy,
		})
	if err != nil {
		return fmt.Errorf("error setting project policy: %v", err)
	}
	return nil
}

func (c *shim) addPolicyBindingForProject(
	policy *cloudresourcemanager.Policy,
	roleName string,
	memberName string,
) bool {
	for i, binding := range policy.Bindings {
		if binding.Role == roleName {
			return c.addMemberToBindingForProject(memberName, policy.Bindings[i])
		}
	}

	// if we didn't find an existing binding entry, then make one
	c.createMemberRoleBindingForProject(policy, roleName, memberName)

	return true
}

// adds member to existing binding. returns bool indicating if an entry was made
func (c *shim) addMemberToBindingForProject(
	memberName string,
	binding *cloudresourcemanager.Binding,
) bool {
	for _, member := range binding.Members {
		if member == memberName {
			// already present
			return false
		}
	}

	binding.Members = append(binding.Members, memberName)
	return true
}

func (c *shim) createMemberRoleBindingForProject(
	policy *cloudresourcemanager.Policy,
	roleName string,
	memberName string,
) {
	policy.Bindings = append(policy.Bindings, &cloudresourcemanager.Binding{
		Members: []string{memberName},
		Role:    roleName,
	})
}
