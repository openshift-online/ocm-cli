package gcp

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/openshift-online/ocm-cli/pkg/gcp"

	"google.golang.org/api/cloudresourcemanager/v1"
)

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

/* Custom Role Creation */

// GetRole fetches the role created to satisfy a credentials request
func GetRole(gcpClient gcp.GcpClient, roleID, projectName string) (*adminpb.Role, error) {
	log.Printf("role id %v", roleID)
	role, err := gcpClient.GetRole(context.TODO(), &adminpb.GetRoleRequest{
		Name: fmt.Sprintf("projects/%s/roles/%s", projectName, roleID),
	})
	return role, err
}

// CreateRole creates a new role given permissions
func CreateRole(gcpClient gcp.GcpClient, permissions []string, roleName, roleID, roleDescription, projectName string) (*adminpb.Role, error) {
	role, err := gcpClient.CreateRole(context.TODO(), &adminpb.CreateRoleRequest{
		Role: &adminpb.Role{
			Title:               roleName,
			Description:         roleDescription,
			IncludedPermissions: permissions,
			Stage:               adminpb.Role_GA,
		},
		Parent: fmt.Sprintf("projects/%s", projectName),
		RoleId: roleID,
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRole updates an existing role given permissions
func UpdateRole(gcpClient gcp.GcpClient, role *adminpb.Role, roleName string) (*adminpb.Role, error) {
	role, err := gcpClient.UpdateRole(context.TODO(), &adminpb.UpdateRoleRequest{
		Name: roleName,
		Role: role,
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}

// DeleteRole deletes the role created to satisfy a credentials request
func DeleteRole(gcpClient gcp.GcpClient, roleName string) (*adminpb.Role, error) {
	role, err := gcpClient.DeleteRole(context.TODO(), &adminpb.DeleteRoleRequest{
		Name: roleName,
	})
	return role, err
}

// UndeleteRole undeletes a previously deleted role that has not yet been pruned
func UndeleteRole(gcpClient gcp.GcpClient, roleName string) (*adminpb.Role, error) {
	role, err := gcpClient.UndeleteRole(context.TODO(), &adminpb.UndeleteRoleRequest{
		Name: roleName,
	})
	return role, err
}
