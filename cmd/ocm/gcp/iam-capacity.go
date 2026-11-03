package gcp

import (
	"context"
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
)

const projectIAMPrincipalLimit = 1500

type projectIAMGrant struct {
	role   string
	member string
}

type projectIAMPreflightError struct {
	cause error
}

func (preflightError *projectIAMPreflightError) Error() string {
	return preflightError.cause.Error()
}

func (preflightError *projectIAMPreflightError) Unwrap() error {
	return preflightError.cause
}

type projectIAMPolicyReadError struct {
	project string
	cause   error
}

func (readError *projectIAMPolicyReadError) Error() string {
	return fmt.Sprintf("cannot check project IAM capacity for project %s: "+
		"unable to read IAM policy; verify project IAM read permissions and retry", readError.project)
}

func (readError *projectIAMPolicyReadError) Unwrap() error {
	return readError.cause
}

func unconditionalProjectIAMGrants(policy *cloudresourcemanager.Policy) map[projectIAMGrant]bool {
	existing := make(map[projectIAMGrant]bool)
	for _, binding := range policy.Bindings {
		if binding.Condition == nil {
			for _, member := range binding.Members {
				existing[projectIAMGrant{role: binding.Role, member: member}] = true
			}
		}
	}
	return existing
}

func (clientShim *shim) checkProjectIAMCapacity(ctx context.Context) error {
	project := clientShim.wifConfig.Gcp().ProjectId()
	policy, err := clientShim.gcpClient.GetProjectIamPolicy(ctx, project, &cloudresourcemanager.GetIamPolicyRequest{
		Options: &cloudresourcemanager.GetPolicyOptions{RequestedPolicyVersion: 3},
	})
	if err != nil {
		return &projectIAMPolicyReadError{project: project, cause: err}
	}
	if policy == nil {
		return fmt.Errorf("cannot check project IAM capacity for project %s: empty policy response", project)
	}

	current := 0
	existing := unconditionalProjectIAMGrants(policy)
	for _, binding := range policy.Bindings {
		current += len(binding.Members)
	}
	for _, auditConfig := range policy.AuditConfigs {
		for _, logConfig := range auditConfig.AuditLogConfigs {
			current += len(logConfig.ExemptedMembers)
		}
	}

	desired := make(map[projectIAMGrant]bool)
	addRoles := func(member string, roles []*cmv1.WifRole) {
		for _, role := range roles {
			if role.ResourceBindings() == nil {
				desired[projectIAMGrant{role: clientShim.formatRoleResourceId(role), member: member}] = true
			}
		}
	}
	support := clientShim.wifConfig.Gcp().Support()
	addRoles("group:"+support.Principal(), support.Roles())
	for _, serviceAccount := range clientShim.wifConfig.Gcp().ServiceAccounts() {
		addRoles(clientShim.formatServiceAccountId(serviceAccount), serviceAccount.Roles())
	}

	additional := 0
	for grant := range desired {
		if !existing[grant] {
			additional++
		}
	}
	projected := current + additional
	if additional > 0 && projected > projectIAMPrincipalLimit {
		return fmt.Errorf("project IAM capacity exceeded for project %s: "+
			"current=%d additional=%d projected=%d limit=%d; "+
			"reduce policy principal occurrences before retrying; no GCP resources were changed by this operation",
			project, current, additional, projected, projectIAMPrincipalLimit)
	}
	return nil
}
