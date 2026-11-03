package gcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/openshift-online/ocm-cli/pkg/gcp"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"
)

type capacityClient struct {
	gcp.GcpClient
	policy      *cloudresourcemanager.Policy
	readErr     error
	readCalls   int
	writeCalls  int
	conflict    bool
	lastRequest *cloudresourcemanager.GetIamPolicyRequest
}

func (client *capacityClient) GetProjectIamPolicy(
	_ context.Context, _ string, request *cloudresourcemanager.GetIamPolicyRequest,
) (*cloudresourcemanager.Policy, error) {
	client.readCalls++
	client.lastRequest = request
	if client.readErr != nil || client.policy == nil {
		return client.policy, client.readErr
	}
	policy := *client.policy
	policy.Bindings = make([]*cloudresourcemanager.Binding, 0, len(client.policy.Bindings))
	for _, binding := range client.policy.Bindings {
		bindingCopy := *binding
		bindingCopy.Members = append([]string(nil), binding.Members...)
		policy.Bindings = append(policy.Bindings, &bindingCopy)
	}
	return &policy, nil
}

func (client *capacityClient) SetProjectIamPolicy(
	_ context.Context, _ string, request *cloudresourcemanager.SetIamPolicyRequest,
) (*cloudresourcemanager.Policy, error) {
	client.writeCalls++
	if client.conflict && client.writeCalls == 1 {
		return nil, errors.New("concurrent policy update")
	}
	client.policy = request.Policy
	return request.Policy, nil
}

func capacityConfig(t *testing.T, accountRoles, supportRoles []*cmv1.WifRoleBuilder) *cmv1.WifConfig {
	t.Helper()
	config, err := cmv1.NewWifConfig().Gcp(cmv1.NewWifGcp().ProjectId("test-project").
		Support(cmv1.NewWifSupport().Principal("support@example.com").Roles(supportRoles...)).
		ServiceAccounts(cmv1.NewWifServiceAccount().ServiceAccountId("worker").Roles(accountRoles...))).Build()
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func capacityPolicy(count int) *cloudresourcemanager.Policy {
	policy := &cloudresourcemanager.Policy{}
	for index := 0; index < count; index++ {
		policy.Bindings = append(policy.Bindings, &cloudresourcemanager.Binding{
			Role:    fmt.Sprintf("projects/test-project/roles/existing%d", index),
			Members: []string{"serviceAccount:existing@test-project.iam.gserviceaccount.com"},
		})
	}
	return policy
}

func TestProjectIAMCapacity(t *testing.T) {
	t.Parallel()
	predefined := func(name string) *cmv1.WifRoleBuilder {
		return cmv1.NewWifRole().RoleId(name).Predefined(true)
	}
	worker := "serviceAccount:worker@test-project.iam.gserviceaccount.com"
	conditional := &cloudresourcemanager.Binding{
		Role: "roles/viewer", Members: []string{worker},
		Condition: &cloudresourcemanager.Expr{Expression: "false"},
	}
	testCases := []struct {
		name         string
		count        int
		accountRoles []*cmv1.WifRoleBuilder
		supportRoles []*cmv1.WifRoleBuilder
		bindings     []*cloudresourcemanager.Binding
		auditConfigs []*cloudresourcemanager.AuditConfig
		wantError    string
	}{
		{
			name: "overflow at 1498", count: 1498,
			accountRoles: []*cmv1.WifRoleBuilder{
				predefined("viewer"), predefined("editor"), predefined("owner"),
				predefined("compute.viewer"), predefined("dns.reader"),
			},
			wantError: "current=1498 additional=5 projected=1503 limit=1500",
		},
		{
			name: "exact fit at 1500", count: 1498,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer"), predefined("editor")},
		},
		{
			name: "existing grant at limit", count: 1499,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			bindings:     []*cloudresourcemanager.Binding{{Role: "roles/viewer", Members: []string{worker}}},
		},
		{name: "no desired additions above limit", count: 1501},
		{
			name: "repeated principal across roles counts each occurrence", count: 1500,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			wantError:    "current=1500 additional=1 projected=1501 limit=1500",
		},
		{
			name: "all members within each binding count", count: 1497,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("editor")},
			bindings: []*cloudresourcemanager.Binding{{
				Role: "roles/viewer", Members: []string{worker, "group:readers@example.com", "user:reader@example.com"},
			}},
			wantError: "current=1500 additional=1 projected=1501 limit=1500",
		},
		{
			name: "duplicate desired grants count once", count: 1499,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer"), predefined("viewer")},
		},
		{
			name: "duplicate existing role-member occurrences each count", count: 1498,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("editor")},
			bindings:     []*cloudresourcemanager.Binding{{Role: "roles/viewer", Members: []string{worker, worker}}},
			wantError:    "current=1500 additional=1 projected=1501 limit=1500",
		},
		{
			name: "conditional grant does not satisfy unconditional grant", count: 1499,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			bindings:     []*cloudresourcemanager.Binding{conditional},
			wantError:    "current=1500 additional=1 projected=1501 limit=1500",
		},
		{
			name: "unconditional grant after conditional grant satisfies desired grant", count: 1498,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			bindings: []*cloudresourcemanager.Binding{
				conditional, {Role: "roles/viewer", Members: []string{worker}},
			},
		},
		{
			name: "audit exemptions count every occurrence", count: 1498,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			auditConfigs: []*cloudresourcemanager.AuditConfig{{
				Service: "allServices", AuditLogConfigs: []*cloudresourcemanager.AuditLogConfig{
					{LogType: "DATA_READ", ExemptedMembers: []string{worker}},
					{LogType: "DATA_WRITE", ExemptedMembers: []string{worker}},
				},
			}},
			wantError: "current=1500 additional=1 projected=1501 limit=1500",
		},
		{
			name: "support roles included and deduplicated", count: 1499,
			supportRoles: []*cmv1.WifRoleBuilder{predefined("viewer"), predefined("viewer"), predefined("editor")},
			wantError:    "current=1499 additional=2 projected=1501 limit=1500",
		},
		{
			name: "existing support grant at limit", count: 1499,
			supportRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			bindings: []*cloudresourcemanager.Binding{{
				Role: "roles/viewer", Members: []string{"group:support@example.com"},
			}},
		},
		{
			name: "same role for support and account are different occurrences", count: 1499,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			supportRoles: []*cmv1.WifRoleBuilder{predefined("viewer")},
			wantError:    "current=1499 additional=2 projected=1501 limit=1500",
		},
		{
			name: "resource-specific roles excluded", count: 1500,
			accountRoles: []*cmv1.WifRoleBuilder{predefined("iam.serviceAccountUser").ResourceBindings(
				cmv1.NewWifResourceBinding().Type("iam.serviceAccounts").Name("target"),
			)},
		},
		{
			name: "existing custom role uses project-qualified identity", count: 1499,
			accountRoles: []*cmv1.WifRoleBuilder{cmv1.NewWifRole().RoleId("custom")},
			bindings: []*cloudresourcemanager.Binding{{
				Role: "projects/test-project/roles/custom", Members: []string{worker},
			}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			policy := capacityPolicy(testCase.count)
			policy.Bindings = append(policy.Bindings, testCase.bindings...)
			policy.AuditConfigs = testCase.auditConfigs
			client := &capacityClient{policy: policy}
			clientShim := &shim{
				wifConfig: capacityConfig(t, testCase.accountRoles, testCase.supportRoles), gcpClient: client,
			}
			err := clientShim.checkProjectIAMCapacity(context.Background())
			if testCase.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
				t.Fatalf("expected capacity diagnostic %q, got %v", testCase.wantError, err)
			}
			if err != nil && (strings.Contains(err.Error(), "@") || !strings.Contains(err.Error(), "test-project")) {
				t.Fatal("capacity diagnostic must identify the project without exposing principals")
			}
			if client.readCalls != 1 || client.lastRequest.Options.RequestedPolicyVersion != 3 {
				t.Fatal("preflight must read the policy once with conditions enabled")
			}
			if client.writeCalls != 0 {
				t.Fatal("preflight must not mutate the policy")
			}
		})
	}
}

func TestGrantSupportAccessPreflightStopsBeforeMutation(t *testing.T) {
	t.Parallel()
	readErr := &googleapi.Error{Code: 403, Message: `policy unavailable: user:private@example.com {"bindings":[]}`}
	for _, testCase := range []struct {
		name    string
		policy  *cloudresourcemanager.Policy
		readErr error
	}{
		{name: "overflow", policy: capacityPolicy(1500)},
		{name: "read failure", readErr: readErr},
		{name: "empty response"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			client := &capacityClient{policy: testCase.policy, readErr: testCase.readErr}
			clientShim := NewGcpClientWifConfigShim(GcpClientWifConfigShimSpec{
				GcpClient: client,
				WifConfig: capacityConfig(t, nil, []*cmv1.WifRoleBuilder{cmv1.NewWifRole().RoleId("custom")}),
			})
			err := clientShim.GrantSupportAccess(context.Background(), log.New(io.Discard, "", 0))
			if err == nil {
				t.Fatal("expected preflight rejection")
			}
			if testCase.readErr != nil && !errors.Is(err, readErr) {
				t.Fatal("policy read error must remain available to callers")
			}
			var preflightError *projectIAMPreflightError
			if !errors.As(err, &preflightError) {
				t.Fatal("callers must distinguish preflight rejection from partial cloud provisioning")
			}
			var policyError *googleapi.Error
			if testCase.readErr != nil && (!errors.As(err, &policyError) || policyError != readErr) {
				t.Fatal("typed policy-read causes must remain available without exposing their text")
			}
			if strings.Contains(err.Error(), "@") || strings.Contains(err.Error(), "bindings") {
				t.Fatal("preflight errors must not expose upstream policy data or principal identities")
			}
			if client.readCalls != 1 || client.writeCalls != 0 {
				t.Fatal("preflight rejection must not retry or mutate cloud resources")
			}
		})
	}
}

func TestProjectIAMCapacityMatchesAccountAccessMethods(t *testing.T) {
	t.Parallel()
	for _, accessMethod := range []cmv1.WifAccessMethod{
		cmv1.WifAccessMethodImpersonate, cmv1.WifAccessMethodWif, cmv1.WifAccessMethodVm,
	} {
		t.Run(string(accessMethod), func(t *testing.T) {
			t.Parallel()
			config, err := cmv1.NewWifConfig().Gcp(cmv1.NewWifGcp().ProjectId("test-project").ServiceAccounts(
				cmv1.NewWifServiceAccount().ServiceAccountId("worker").AccessMethod(accessMethod).Roles(
					cmv1.NewWifRole().RoleId("viewer").Predefined(true),
					cmv1.NewWifRole().RoleId("iam.serviceAccountUser").Predefined(true).ResourceBindings(
						cmv1.NewWifResourceBinding().Type("iam.serviceAccounts").Name("target"),
					),
				),
			)).Build()
			if err != nil {
				t.Fatal(err)
			}
			client := &capacityClient{policy: capacityPolicy(1500)}
			clientShim := &shim{wifConfig: config, gcpClient: client}
			err = clientShim.checkProjectIAMCapacity(context.Background())
			if err == nil || !strings.Contains(err.Error(), "current=1500 additional=1 projected=1501") {
				t.Fatal("each access method must plan exactly one project grant, excluding resource grants")
			}
			client.policy = capacityPolicy(0)
			if err := clientShim.bindRolesToServiceAccount(context.Background(), log.New(io.Discard, "", 0),
				config.Gcp().ServiceAccounts()[0]); err != nil {
				t.Fatal(err)
			}
			if len(client.policy.Bindings) != 1 || client.policy.Bindings[0].Role != "roles/viewer" ||
				len(client.policy.Bindings[0].Members) != 1 ||
				client.policy.Bindings[0].Members[0] != "serviceAccount:worker@test-project.iam.gserviceaccount.com" {
				t.Fatal("planned grants must match the actual project-policy writer for every access method")
			}
		})
	}
}

func TestGrantSupportAccessStillRetriesPolicyConflicts(t *testing.T) {
	client := &capacityClient{policy: capacityPolicy(1), conflict: true}
	clientShim := NewGcpClientWifConfigShim(GcpClientWifConfigShimSpec{
		GcpClient: client, FailFast: true,
		WifConfig: capacityConfig(t, nil, []*cmv1.WifRoleBuilder{
			cmv1.NewWifRole().RoleId("viewer").Predefined(true),
		}),
	})
	err := clientShim.GrantSupportAccess(context.Background(), log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	if client.writeCalls != 2 {
		t.Fatalf("expected one failed write followed by a successful retry, got %d writes", client.writeCalls)
	}
	if client.readCalls != 3 {
		t.Fatalf("expected preflight plus two attempts, got %d policy reads", client.readCalls)
	}
	if len(client.policy.Bindings) != 2 || len(client.policy.Bindings[0].Members) != 1 ||
		client.policy.Bindings[0].Members[0] != "serviceAccount:existing@test-project.iam.gserviceaccount.com" ||
		client.policy.Bindings[1].Role != "roles/viewer" || len(client.policy.Bindings[1].Members) != 1 ||
		client.policy.Bindings[1].Members[0] != "group:support@example.com" {
		t.Fatal("retry must persist the desired grant and preserve existing members")
	}
}

func TestProjectIAMBindingsPreserveConditions(t *testing.T) {
	t.Parallel()
	conditional := &cloudresourcemanager.Binding{
		Role: "roles/viewer", Members: []string{"group:other@example.com"},
		Condition: &cloudresourcemanager.Expr{Expression: "false"},
	}
	client := &capacityClient{policy: &cloudresourcemanager.Policy{Bindings: []*cloudresourcemanager.Binding{conditional}}}
	config := capacityConfig(t, nil, []*cmv1.WifRoleBuilder{cmv1.NewWifRole().RoleId("viewer").Predefined(true)})
	clientShim := &shim{wifConfig: config, gcpClient: client}
	modified, err := clientShim.ensurePolicyBindingsForProject(context.Background(), config.Gcp().Support().Roles(),
		"group:support@example.com", "test-project")
	if err != nil || !modified {
		t.Fatalf("expected unconditional grant, got modified=%t error=%v", modified, err)
	}
	if len(client.policy.Bindings) != 2 || client.policy.Bindings[0].Condition == nil ||
		client.policy.Bindings[0].Condition.Expression != "false" ||
		len(client.policy.Bindings[0].Members) != 1 || client.policy.Bindings[0].Members[0] != "group:other@example.com" {
		t.Fatal("conditional grant must remain unchanged and a separate unconditional grant must be added")
	}
	if client.policy.Bindings[1].Condition != nil || client.policy.Bindings[1].Role != "roles/viewer" ||
		len(client.policy.Bindings[1].Members) != 1 || client.policy.Bindings[1].Members[0] != "group:support@example.com" ||
		client.lastRequest.Options.RequestedPolicyVersion != 3 {
		t.Fatal("project policy updates must preserve conditions and add an unconditional grant")
	}
}

func TestGrantSupportAccessReusesGrantInLaterBinding(t *testing.T) {
	t.Parallel()
	policy := capacityPolicy(1498)
	policy.Bindings = append(policy.Bindings,
		&cloudresourcemanager.Binding{Role: "roles/viewer", Members: []string{"group:other@example.com"}},
		&cloudresourcemanager.Binding{Role: "roles/viewer", Members: []string{"group:support@example.com"}},
	)
	client := &capacityClient{policy: policy}
	clientShim := NewGcpClientWifConfigShim(GcpClientWifConfigShimSpec{
		GcpClient: client,
		WifConfig: capacityConfig(t, nil, []*cmv1.WifRoleBuilder{
			cmv1.NewWifRole().RoleId("viewer").Predefined(true),
		}),
	})
	if err := clientShim.GrantSupportAccess(context.Background(), log.New(io.Discard, "", 0)); err != nil {
		t.Fatal(err)
	}
	if client.writeCalls != 0 || client.readCalls != 2 {
		t.Fatal("existing grants in any unconditional binding must require no policy write, even at capacity")
	}
}
