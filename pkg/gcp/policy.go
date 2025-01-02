// Defines a Policy type which wraps the iam.Policy object. This enables
// callers of the gcp package to process iam policies without needing to make
// additional imports.
package gcp

import (
	"cloud.google.com/go/iam"
)

// The resource name belonging to the policy.
//
// For service accounts, this would take the forms:
// * `projects/{PROJECT_ID}/serviceAccounts/{EMAIL_ADDRESS}`
// * `projects/{PROJECT_ID}/serviceAccounts/{UNIQUE_ID}`
// * `projects/-/serviceAccounts/{EMAIL_ADDRESS}`
// * `projects/-/serviceAccounts/{UNIQUE_ID}`
//
// It is recommended that wildcard `-` form is avoided due to the potential for
// misleading error messages. The client helper FmtSaResourceId produces a
// string that may be used as a policy member.
type PolicyMember string

// The name of the role belonging to the policy.
//
// Values of this type take two different forms, depending on whether it is
// predefined.
//
// For predefined roles:
// * `roles/{role_id}`
//
// For custom roles:
// * `projects/{project}/roles/{role_id}`
type RoleName string

type Policy interface {
	HasRole(member PolicyMember, roleName RoleName) bool
	AddRole(member PolicyMember, roleName RoleName)

	// Getters
	IamPolicy() *iam.Policy
	ResourceId() string
}

type policy struct {
	policy     *iam.Policy
	resourceId string
}

func (p *policy) AddRole(member PolicyMember, roleName RoleName) {
	p.policy.Add(string(member), iam.RoleName(roleName))
}

func (p *policy) HasRole(member PolicyMember, roleName RoleName) bool {
	return p.policy.HasRole(string(member), iam.RoleName(roleName))
}

func (p *policy) IamPolicy() *iam.Policy {
	return p.policy
}

func (p *policy) ResourceId() string {
	return p.resourceId
}
