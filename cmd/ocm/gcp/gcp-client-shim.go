package gcp

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

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
	// The time in seconds in which IAM inconsistencies may occur.
	IamApiRetrySeconds = 600
)

const (
	impersonatorRole         = "roles/iam.serviceAccountTokenCreator"
	workloadIdentityUserRole = "roles/iam.workloadIdentityUser"
)

// All operations that modify cloud resources should be logged to the user.
// For this reason, all methods of this interface take a logger as a parameter.
type GcpClientWifConfigShim interface {
	CreateServiceAccounts(ctx context.Context, log *log.Logger) error
	CreateWorkloadIdentityPool(ctx context.Context, log *log.Logger) error
	CreateWorkloadIdentityProvider(ctx context.Context, log *log.Logger) error
	GrantSupportAccess(ctx context.Context, log *log.Logger) error

	DeleteServiceAccounts(ctx context.Context, log *log.Logger) error
	DeleteWorkloadIdentityPool(ctx context.Context, log *log.Logger) error
	UnbindServiceAccounts(ctx context.Context, log *log.Logger) error
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

func (c *shim) CreateServiceAccounts(
	ctx context.Context,
	log *log.Logger,
) error {
	for _, serviceAccount := range c.wifConfig.Gcp().ServiceAccounts() {
		var (
			sa      *adminpb.ServiceAccount
			lastErr error
		)
		if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
			log.Printf("Ensuring service account '%s' exists...", serviceAccount.ServiceAccountId())
			sa, lastErr = c.createServiceAccount(ctx, log, serviceAccount)
			return lastErr != nil, lastErr
		}, IamApiRetrySeconds, log); err != nil {
			return lastErr
		}

		if sa.Disabled {
			if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
				log.Printf("Enabling service account '%s'...", serviceAccount.ServiceAccountId())
				lastErr := c.enableServiceAccount(
					ctx,
					log,
					serviceAccount,
				)
				return lastErr != nil, lastErr
			}, IamApiRetrySeconds, log); err != nil {
				return lastErr
			}
			log.Printf("IAM service account '%s' has been enabled", serviceAccount.ServiceAccountId())
		}

		if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
			log.Printf("Ensuring roles exist for service account '%s'...", serviceAccount.ServiceAccountId())
			lastErr = c.createOrUpdateRoles(ctx, log, serviceAccount.Roles())
			return lastErr != nil, lastErr
		}, IamApiRetrySeconds, log); err != nil {
			return lastErr
		}

		if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
			log.Printf("Ensuring role bindings for service account '%s'...", serviceAccount.ServiceAccountId())
			lastErr = c.bindRolesToServiceAccount(ctx, log, serviceAccount)
			return lastErr != nil, lastErr
		}, IamApiRetrySeconds, log); err != nil {
			return lastErr
		}

		if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
			log.Printf("Ensuring access to service account '%s'...", serviceAccount.ServiceAccountId())
			lastErr = c.grantAccessToServiceAccount(ctx, log, serviceAccount)
			return lastErr != nil, lastErr
		}, IamApiRetrySeconds, log); err != nil {
			return lastErr
		}
	}
	return nil
}

func (c *shim) CreateWorkloadIdentityPool(
	ctx context.Context,
	log *log.Logger,
) error {
	var lastErr error
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		log.Print("Ensuring workload identity pool exists...")
		lastErr = c.createWorkloadIdentityPool(ctx, log)
		return lastErr != nil, lastErr
	}, IamApiRetrySeconds, log); err != nil {
		return lastErr
	}
	return nil
}

func (c *shim) CreateWorkloadIdentityProvider(
	ctx context.Context,
	log *log.Logger,
) error {
	var lastErr error
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		log.Print("Ensuring workload identity pool OIDC provider exists...")
		lastErr = c.createWorkloadIdentityProvider(ctx, log)
		return lastErr != nil, lastErr
	}, IamApiRetrySeconds, log); err != nil {
		return lastErr
	}
	return nil
}

func (c *shim) GrantSupportAccess(
	ctx context.Context,
	log *log.Logger,
) error {
	var lastErr error
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		log.Print("Ensuring Red Hat support has access to project...")
		lastErr = c.grantSupportAccess(ctx, log)
		return lastErr != nil, lastErr
	}, IamApiRetrySeconds, log); err != nil {
		return lastErr
	}
	return nil
}

func (c *shim) createWorkloadIdentityPool(
	ctx context.Context,
	log *log.Logger,
) error {
	description := fmt.Sprintf(wifDescription, c.wifConfig.DisplayName())
	poolId := c.wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	project := c.wifConfig.Gcp().ProjectId()

	parentResourceForPool := fmt.Sprintf("projects/%s/locations/global", project)
	poolResource := fmt.Sprintf("%s/workloadIdentityPools/%s", parentResourceForPool, poolId)

	resp, err := c.gcpClient.GetWorkloadIdentityPool(ctx, poolResource)

	if err != nil {
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
				return errors.Wrapf(err, "failed to create workload identity pool '%s'", poolId)
			}
			log.Printf("Workload identity pool created with name '%s'", poolId)

			return nil
		}

		return errors.Wrapf(err, "failed to check if there is existing workload identity pool '%s'", poolId)
	}

	if resp != nil && resp.State == "DELETED" {
		_, err := c.gcpClient.UndeleteWorkloadIdentityPool(
			ctx, poolResource, &iamv1.UndeleteWorkloadIdentityPoolRequest{},
		)
		if err != nil {
			return errors.Wrapf(err, "failed to undelete workload identity pool '%s'", poolId)
		}
		log.Printf("Undeleted Workload identity pool '%s'", poolId)
	}

	// Enable the pool if it exists but is disabled.
	if resp != nil && resp.Disabled {
		if err := c.gcpClient.EnableWorkloadIdentityPool(
			ctx,
			resp.Name,
		); err != nil {
			return errors.Wrapf(err, "failed to enabled workload identity pool '%s'", poolId)
		}
		log.Printf("Workload identity pool '%s' has been re-enabled", resp.DisplayName)
	}

	return nil
}

func (c *shim) createWorkloadIdentityProvider(
	ctx context.Context,
	log *log.Logger,
) error {
	attributeMap := map[string]string{
		"google.subject": "assertion.sub",
	}
	audiences := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().AllowedAudiences()
	description := fmt.Sprintf(wifDescription, c.wifConfig.DisplayName())
	issuerUrl := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IssuerUrl()
	jwks := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().Jwks()
	poolId := c.wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	projectId := c.wifConfig.Gcp().ProjectId()
	providerId := c.wifConfig.Gcp().WorkloadIdentityPool().IdentityProvider().IdentityProviderId()
	state := "ACTIVE"

	parent := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", projectId, poolId)
	providerResource := fmt.Sprintf("%s/providers/%s", parent, providerId)

	resp, err := c.gcpClient.GetWorkloadIdentityProvider(ctx, providerResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			provider := &iamv1.WorkloadIdentityPoolProvider{
				Name:        providerId,
				DisplayName: providerId,
				Description: description,
				State:       state,
				Disabled:    false,
				Oidc: &iamv1.Oidc{
					AllowedAudiences: audiences,
					IssuerUri:        issuerUrl,
					JwksJson:         jwks,
				},
				AttributeMapping: attributeMap,
			}

			_, err := c.gcpClient.CreateWorkloadIdentityProvider(ctx, parent, providerId, provider)
			if err != nil {
				return errors.Wrapf(err, "failed to create workload identity provider '%s'", providerId)
			}
			log.Printf("Workload identity provider created with name '%s' for pool '%s'", providerId, poolId)
			return nil
		}
		return errors.Wrapf(err, "failed to check if there is existing workload identity provider '%s' in pool '%s'",
			providerId, poolId)
	}

	var needsUpdate bool
	if resp.Description != description ||
		resp.Disabled ||
		resp.DisplayName != providerId ||
		resp.State != state ||
		resp.Oidc.IssuerUri != issuerUrl ||
		!utils.JwksEqual(resp.Oidc.JwksJson, jwks) ||
		!reflect.DeepEqual(resp.AttributeMapping, attributeMap) ||
		!reflect.DeepEqual(resp.Oidc.AllowedAudiences, audiences) {
		needsUpdate = true
	}

	if needsUpdate {
		if err := c.gcpClient.UpdateWorkloadIdentityPoolOidcIdentityProvider(ctx,
			&iamv1.WorkloadIdentityPoolProvider{
				Name:        providerResource,
				DisplayName: providerId,
				Description: description,
				State:       state,
				Disabled:    false,
				Oidc: &iamv1.Oidc{
					AllowedAudiences: audiences,
					IssuerUri:        issuerUrl,
					JwksJson:         jwks,
				},
				AttributeMapping: attributeMap,
			},
		); err != nil {
			return errors.Wrapf(
				err,
				"failed to updated identity provider '%s' for workload identity pool '%s'",
				providerId, poolId,
			)
		}
		log.Printf("Workload identity pool '%s' identity provider '%s' was updated", poolId, providerId)
	}
	return nil
}

func (c *shim) grantSupportAccess(
	ctx context.Context,
	log *log.Logger,
) error {
	support := c.wifConfig.Gcp().Support()
	if err := c.createOrUpdateRoles(ctx, log, support.Roles()); err != nil {
		return err
	}
	if err := c.bindRolesToGroup(ctx, log, support.Principal(), support.Roles()); err != nil {
		return err
	}
	return nil
}

// Returns the internal representation of the specified gcp service account on
// successful creation. If the service account already exists, the current
// instance of the service account is returned without error.
func (c *shim) createServiceAccount(
	ctx context.Context,
	log *log.Logger,
	serviceAccount *cmv1.WifServiceAccount,
) (*adminpb.ServiceAccount, error) {
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
	sa, err := c.gcpClient.CreateServiceAccount(ctx, request)
	if err != nil {
		pApiError, ok := err.(*apierror.APIError)
		if ok {
			if pApiError.GRPCStatus().Code() == codes.AlreadyExists {
				return c.gcpClient.GetServiceAccount(
					ctx,
					&adminpb.GetServiceAccountRequest{
						Name: gcp.FmtSaResourceId(
							serviceAccount.ServiceAccountId(),
							c.wifConfig.Gcp().ProjectId(),
						)},
				)
			}
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create IAM service account")
	}
	log.Printf("IAM service account '%s' has been created", serviceAccountId)
	return sa, nil
}

func (c *shim) enableServiceAccount(
	ctx context.Context,
	log *log.Logger,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	if err := c.gcpClient.EnableServiceAccount(
		ctx,
		serviceAccount.ServiceAccountId(),
		c.wifConfig.Gcp().ProjectId(),
	); err != nil {
		return err
	}
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
		existingRole, err := c.getRole(ctx, c.formatRoleResourceId(role))
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
					return errors.Wrap(err, fmt.Sprintf("Failed to create role '%s'", roleID))
				}
				log.Printf("Role '%s' has been created", roleID)
				continue
			} else {
				return errors.Wrap(err, "Failed to check if role exists")
			}
		}

		// Undelete role if it was deleted
		if existingRole.Deleted {
			_, err = c.undeleteRole(ctx, c.formatRoleResourceId(role))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to undelete custom role '%s'", roleID))
			}
			existingRole.Deleted = false
			log.Printf("Role '%s' has been undeleted", roleID)
		}

		// If role was disabled, enable role
		if existingRole.Stage == adminpb.Role_DISABLED {
			existingRole.Stage = adminpb.Role_GA
			_, err := c.updateRole(ctx, existingRole, c.formatRoleResourceId(role))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to enable role '%s'", roleID))
			}
			log.Printf("Role '%s' has been enabled", roleID)
		}

		if addedPermissions, needsUpdate := c.missingPermissions(permissions, existingRole.IncludedPermissions); needsUpdate {
			// Add missing permissions
			existingRole.IncludedPermissions = append(existingRole.IncludedPermissions, addedPermissions...)
			sort.Strings(existingRole.IncludedPermissions)

			_, err := c.updateRole(ctx, existingRole, c.formatRoleResourceId(role))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Failed to update role '%s'", roleID))
			}
			log.Printf("Role '%s' has been updated", roleID)
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
	log *log.Logger,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	return c.bindRolesToPrincipal(
		ctx,
		log,
		c.formatServiceAccountId(serviceAccount),
		serviceAccount.Roles(),
	)
}

func (c *shim) bindRolesToGroup(
	ctx context.Context,
	log *log.Logger,
	groupEmail string,
	roles []*cmv1.WifRole,
) error {
	return c.bindRolesToPrincipal(
		ctx,
		log,
		fmt.Sprintf("group:%s", groupEmail),
		roles,
	)
}

func (c *shim) bindRolesToPrincipal(
	ctx context.Context,
	log *log.Logger,
	principal string,
	roles []*cmv1.WifRole,
) error {
	modified, err := c.ensurePolicyBindingsForProject(
		ctx,
		c.formatRoles(roles),
		principal,
		c.wifConfig.Gcp().ProjectId(),
	)
	if err != nil {
		return errors.Errorf("Failed to bind roles to principal %s: %s", principal, err)
	}
	if modified {
		log.Printf("Bound roles to principal '%s'", principal)
	}
	return nil
}

func (c *shim) unbindRolesFromPrincipal(
	ctx context.Context,
	log *log.Logger,
	principal string,
	roles []*cmv1.WifRole,
) error {
	modified, err := c.removePolicyBindingsFromProject(
		ctx,
		c.formatRoles(roles),
		principal,
		c.wifConfig.Gcp().ProjectId(),
	)
	if err != nil {
		return errors.Errorf("Failed to unbind roles from principal %s: %s", principal, err)
	}
	if modified {
		log.Printf("Unbound roles from principal '%s'", principal)
	}
	return nil
}

func (c *shim) grantAccessToServiceAccount(
	ctx context.Context,
	log *log.Logger,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	switch serviceAccount.AccessMethod() {
	case cmv1.WifAccessMethodImpersonate:
		return c.attachImpersonator(ctx, serviceAccount)
	case cmv1.WifAccessMethodWif:
		return c.attachWorkloadIdentityPool(ctx, serviceAccount)
	case cmv1.WifAccessMethodVm:
		// Service accounts with the "vm" access method require no external access
		return nil
	default:
		log.Printf("Warning: %s is not a supported access type\n", serviceAccount.AccessMethod())
	}
	return nil
}

func (c *shim) formatRoles(
	roles []*cmv1.WifRole,
) []string {
	formattedRoles := make([]string, 0, len(roles))
	for _, role := range roles {
		formattedRoles = append(formattedRoles, c.formatRoleResourceId(role))
	}
	return formattedRoles
}

func (c *shim) formatRoleResourceId(
	role *cmv1.WifRole,
) string {
	if role.Predefined() {
		return fmt.Sprintf("roles/%s", role.RoleId())
	} else {
		return fmt.Sprintf("projects/%s/roles/%s", c.wifConfig.Gcp().ProjectId(), role.RoleId())
	}
}
func (c *shim) formatServiceAccountId(
	serviceAccount *cmv1.WifServiceAccount,
) string {
	return fmt.Sprintf("serviceAccount:%s@%s.iam.gserviceaccount.com",
		serviceAccount.ServiceAccountId(),
		c.wifConfig.Gcp().ProjectId(),
	)
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
// This method modifies cloud resources.
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
// This method modifies cloud resources.
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
// This method modifies cloud resources.
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
// Return value indicates whether a modification occurred.
func (c *shim) ensurePolicyBindingsForProject(
	ctx context.Context,
	roles []string,
	member string,
	projectName string,
) (bool, error) {
	needPolicyUpdate := false

	policy, err := c.gcpClient.GetProjectIamPolicy(ctx, projectName, &cloudresourcemanager.GetIamPolicyRequest{})

	if err != nil {
		return false, errors.Wrap(err, "Failed to fetch policy for project")
	}

	// Validate that each role exists, and add the policy binding as needed
	for _, definedRole := range roles {
		roleBindingAdded := c.ensureRoleBindingInPolicy(policy, definedRole)
		if roleBindingAdded {
			needPolicyUpdate = true
		}
		memberAddedToRoleBinding := c.applyMemberToRoleInPolicy(policy, definedRole, member, c.addMemberToBindingForProject)
		if memberAddedToRoleBinding {
			needPolicyUpdate = true
		}

	}

	if needPolicyUpdate {
		return true, c.setProjectIamPolicy(ctx, policy)
	}

	// If we made it this far there were no updates needed
	return false, nil
}

// removePolicyBindingsFromProject ensures that given member no longer has the
// following roles bound within the project's iam policy
// Roles should be in the format projects/{project}/roles/{role_id} for custom roles and roles/{role_id}
// for predefined roles.
// Return value indicates whether a modification occurred.
func (c *shim) removePolicyBindingsFromProject(
	ctx context.Context,
	roles []string,
	member string,
	projectName string,
) (bool, error) {
	needPolicyUpdate := false

	policy, err := c.gcpClient.GetProjectIamPolicy(ctx, projectName, &cloudresourcemanager.GetIamPolicyRequest{})
	if err != nil {
		return false, errors.Wrap(err, "Failed to fetch policy for project")
	}

	for _, definedRole := range roles {
		modified := c.applyMemberToRoleInPolicy(policy, definedRole, member, c.removeMemberFromBinding)
		if modified {
			needPolicyUpdate = true
		}
	}

	if needPolicyUpdate {
		return true, c.setProjectIamPolicy(ctx, policy)
	}

	// If we made it this far there were no updates needed
	return false, nil
}

// This method modifies cloud resources.
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

// Applies the passed in function to the policy entries which have the desired
// roles and members. The application function will return a boolean indicating
// whether a modification to the binding was made. The method returns whether a
// modification occurred.
func (c *shim) applyMemberToRoleInPolicy(
	policy *cloudresourcemanager.Policy,
	roleName string,
	memberName string,
	applyFunc func(string, *cloudresourcemanager.Binding) bool,
) bool {
	for i, binding := range policy.Bindings {
		if binding.Role == roleName {
			return applyFunc(memberName, policy.Bindings[i])
		}
	}
	// if we didn't find the role in the project policy, then no modification was needed.
	return false
}

// adds member to existing binding. Returns bool indicating whether a new entry was made.
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

func (c *shim) removeMemberFromBinding(
	memberName string,
	binding *cloudresourcemanager.Binding,
) bool {
	newMembers := []string{}
	for _, member := range binding.Members {
		if member != memberName {
			newMembers = append(newMembers, member)
		}
	}
	if len(newMembers) != len(binding.Members) {
		binding.Members = newMembers
		return true
	}
	return false
}

// Ensure that the project iam policy contains a binding entry for the specified role.
// If the binding is not yet present in the policy, it will be appended.
// The return value indicates whether the policy was modified.
func (c *shim) ensureRoleBindingInPolicy(
	policy *cloudresourcemanager.Policy,
	roleName string,
) bool {
	for _, binding := range policy.Bindings {
		if binding.Role == roleName {
			return false
		}
	}
	policy.Bindings = append(policy.Bindings, &cloudresourcemanager.Binding{
		Members: []string{},
		Role:    roleName,
	})
	return true
}

func (c *shim) attachImpersonator(
	ctx context.Context,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	policy, err := c.gcpClient.GetServiceAccountAccessPolicy(
		ctx,
		gcp.FmtSaResourceId(
			serviceAccount.ServiceAccountId(),
			c.wifConfig.Gcp().ProjectId(),
		),
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to determine access policy of service account '%s'",
			serviceAccount.ServiceAccountId())
	}
	if policy.HasRole(
		gcp.PolicyMember(fmt.Sprintf("serviceAccount:%s", c.wifConfig.Gcp().ImpersonatorEmail())),
		impersonatorRole,
	) {
		return nil
	}

	policy.AddRole(
		gcp.PolicyMember(fmt.Sprintf("serviceAccount:%s", c.wifConfig.Gcp().ImpersonatorEmail())),
		impersonatorRole,
	)
	if err := c.gcpClient.SetServiceAccountAccessPolicy(
		ctx,
		policy,
	); err != nil {
		return errors.Wrapf(err, "Failed to attach impersonator to service account '%s'",
			serviceAccount.ServiceAccountId())
	}

	log.Printf("Impersonation access granted to service account '%s'",
		serviceAccount.ServiceAccountId())
	return nil
}

func (c *shim) attachWorkloadIdentityPool(
	ctx context.Context,
	serviceAccount *cmv1.WifServiceAccount,
) error {
	policy, err := c.gcpClient.GetServiceAccountAccessPolicy(
		ctx,
		gcp.FmtSaResourceId(
			serviceAccount.ServiceAccountId(),
			c.wifConfig.Gcp().ProjectId(),
		),
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to determine access policy of service account '%s'",
			serviceAccount.ServiceAccountId())
	}

	var modified bool
	openshiftNamespace := serviceAccount.CredentialRequest().SecretRef().Namespace()
	for _, openshiftServiceAccount := range serviceAccount.CredentialRequest().ServiceAccountNames() {
		principal := fmt.Sprintf(
			"principal://iam.googleapis.com/projects/%s/"+
				"locations/global/workloadIdentityPools/%s/"+
				"subject/system:serviceaccount:%s:%s",
			c.wifConfig.Gcp().ProjectNumber(),
			c.wifConfig.Gcp().WorkloadIdentityPool().PoolId(),
			openshiftNamespace, openshiftServiceAccount,
		)

		if !policy.HasRole(
			gcp.PolicyMember(principal),
			gcp.RoleName(workloadIdentityUserRole),
		) {
			modified = true

			policy.AddRole(
				gcp.PolicyMember(principal),
				gcp.RoleName(workloadIdentityUserRole),
			)
		}
	}

	if modified {
		if err := c.gcpClient.SetServiceAccountAccessPolicy(
			ctx,
			policy,
		); err != nil {
			return errors.Wrapf(err, "Failed to attach federated access on service account '%s'",
				serviceAccount.ServiceAccountId())
		}
		log.Printf("Federated access granted to service account '%s'",
			serviceAccount.ServiceAccountId())
	}
	return nil
}

func (c *shim) DeleteServiceAccounts(
	ctx context.Context,
	log *log.Logger,
) error {
	log.Println("Deleting service accounts...")
	projectId := c.wifConfig.Gcp().ProjectId()

	var lastErr error
	for _, serviceAccount := range c.wifConfig.Gcp().ServiceAccounts() {
		serviceAccountID := serviceAccount.ServiceAccountId()
		if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
			log.Printf("Deleting service account '%s'...", serviceAccountID)
			lastErr = c.gcpClient.DeleteServiceAccount(ctx, serviceAccountID, projectId, true)
			return lastErr != nil, lastErr
		}, IamApiRetrySeconds, log); err != nil {
			return errors.Wrapf(lastErr, "Failed to delete service account '%s'", serviceAccountID)
		}
	}
	return nil
}

func (c *shim) DeleteWorkloadIdentityPool(
	ctx context.Context,
	log *log.Logger,
) error {
	var lastErr error
	if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
		log.Print("Deleting workload identity pool...")
		lastErr = c.deleteWorkloadIdentityPool(ctx, log)
		return lastErr != nil, lastErr
	}, IamApiRetrySeconds, log); err != nil {
		return lastErr
	}
	return nil
}

func (c *shim) deleteWorkloadIdentityPool(
	ctx context.Context,
	log *log.Logger,
) error {
	log.Println("Deleting workload identity pool...")
	projectId := c.wifConfig.Gcp().ProjectId()
	poolName := c.wifConfig.Gcp().WorkloadIdentityPool().PoolId()
	poolResource := fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", projectId, poolName)

	_, err := c.gcpClient.DeleteWorkloadIdentityPool(ctx, poolResource)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 &&
			strings.Contains(gerr.Message, "Requested entity was not found") {
			log.Printf("Workload identity pool '%s' not found", poolName)
			return nil
		}
		return errors.Wrapf(err, "Failed to delete workload identity pool '%s'", poolName)
	}

	log.Printf("Workload identity pool '%s' deleted", poolName)
	return nil
}

func (c *shim) UnbindServiceAccounts(
	ctx context.Context,
	log *log.Logger,
) error {
	var lastErr error

	log.Print("Unbinding service accounts from project IAM policy...")
	for _, serviceAccount := range c.wifConfig.Gcp().ServiceAccounts() {
		if err := utils.RetryWithBackoffandTimeout(func() (bool, error) {
			log.Printf("Unbinding service account '%s'...", serviceAccount.ServiceAccountId())
			lastErr = c.unbindRolesFromPrincipal(
				ctx,
				log,
				c.formatServiceAccountId(serviceAccount),
				serviceAccount.Roles(),
			)
			return lastErr != nil, lastErr
		}, IamApiRetrySeconds, log); err != nil {
			return errors.Wrapf(lastErr, "Failed to unbind service account '%s'", serviceAccount.ServiceAccountId())
		}
	}
	return nil
}
