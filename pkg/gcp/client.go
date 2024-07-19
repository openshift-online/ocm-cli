package gcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"cloud.google.com/go/iam"
	iamadmin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/storage"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"

	iamv1 "google.golang.org/api/iam/v1"
	"google.golang.org/api/iterator"
	secretmanager "google.golang.org/api/secretmanager/v1"
)

type GcpClient interface {
	ListServiceAccounts(project string, filter func(s string) bool) ([]string, error) //nolint:lll

	CreateServiceAccount(ctx context.Context, request *adminpb.CreateServiceAccountRequest) (*adminpb.ServiceAccount, error) //nolint:lll

	CreateWorkloadIdentityPool(ctx context.Context, parent, poolID string, pool *iamv1.WorkloadIdentityPool) (*iamv1.Operation, error)               //nolint:lll
	GetWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPool, error)                                               //nolint:lll
	DeleteWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.Operation, error)                                                       //nolint:lll
	UndeleteWorkloadIdentityPool(ctx context.Context, resource string, request *iamv1.UndeleteWorkloadIdentityPoolRequest) (*iamv1.Operation, error) //nolint:lll

	CreateWorkloadIdentityProvider(ctx context.Context, parent, providerID string, provider *iamv1.WorkloadIdentityPoolProvider) (*iamv1.Operation, error) //nolint:lll
	GetWorkloadIdentityProvider(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPoolProvider, error)                                         //nolint:lll

	DeleteServiceAccount(saName string, project string, allowMissing bool) error

	GetProjectIamPolicy(projectName string, request *cloudresourcemanager.GetIamPolicyRequest) (*cloudresourcemanager.Policy, error)     //nolint:lll
	SetProjectIamPolicy(svcAcctResource string, request *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error) //nolint:lll

	AttachImpersonator(saId, projectId, impersonatorResourceId string) error
	AttachWorkloadIdentityPool(sa *cmv1.WifServiceAccount, poolId, projectId string) error

	SaveSecret(secretId, projectId string, secretData []byte) error
	RetreiveSecret(secretId string, projectId string) ([]byte, error)

	ProjectNumberFromId(projectId string) (int64, error)

	GetRole(context.Context, *adminpb.GetRoleRequest) (*adminpb.Role, error)
	CreateRole(context.Context, *adminpb.CreateRoleRequest) (*adminpb.Role, error)
	UpdateRole(context.Context, *adminpb.UpdateRoleRequest) (*adminpb.Role, error)
	DeleteRole(context.Context, *adminpb.DeleteRoleRequest) (*adminpb.Role, error)
	UndeleteRole(context.Context, *adminpb.UndeleteRoleRequest) (*adminpb.Role, error)
	ListRoles(context.Context, *adminpb.ListRolesRequest) (*adminpb.ListRolesResponse, error)
}

type gcpClient struct {
	ctx                  context.Context
	iamClient            *iamadmin.IamClient
	oldIamClient         *iamv1.Service
	cloudResourceManager *cloudresourcemanager.Service
	secretManager        *secretmanager.Service
	storageClient        *storage.Client
}

func NewGcpClient(ctx context.Context) (GcpClient, error) {
	iamClient, err := iamadmin.NewIamClient(ctx)
	if err != nil {
		return nil, err
	}
	cloudResourceManager, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, err
	}
	secretManager, err := secretmanager.NewService(ctx)
	if err != nil {
		return nil, err
	}
	// The new iam client doesn't support workload identity federation operations.
	oldIamClient, err := iamv1.NewService(ctx)
	if err != nil {
		return nil, err
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &gcpClient{
		ctx:                  ctx,
		iamClient:            iamClient,
		cloudResourceManager: cloudResourceManager,
		secretManager:        secretManager,
		oldIamClient:         oldIamClient,
		storageClient:        storageClient,
	}, nil
}

func (c *gcpClient) CreateServiceAccount(ctx context.Context,
	request *adminpb.CreateServiceAccountRequest) (*adminpb.ServiceAccount, error) {
	svcAcct, err := c.iamClient.CreateServiceAccount(ctx, request)
	return svcAcct, err
}

func (c *gcpClient) DeleteServiceAccount(saName string, project string, allowMissing bool) error {
	name := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", project, saName, project)
	err := c.iamClient.DeleteServiceAccount(context.Background(), &adminpb.DeleteServiceAccountRequest{
		Name: name,
	})
	if err != nil {
		return c.handleDeleteServiceAccountError(err, allowMissing)
	}
	return nil
}

func (c *gcpClient) ListServiceAccounts(project string, filter func(string) bool) ([]string, error) {
	out := []string{}
	// Listing objects follow the iterator pattern specified here:
	// https://github.com/googleapis/google-cloud-go/wiki/Iterator-Guidelines
	saIterator := c.iamClient.ListServiceAccounts(context.Background(), &adminpb.ListServiceAccountsRequest{
		Name: fmt.Sprintf("projects/%s", project),
		// The pagesize can be adjusted for optimized network load.
		// PageSize: 5,
	})
	for sa, err := saIterator.Next(); err != iterator.Done; sa, err = saIterator.Next() {
		if err != nil {
			return nil, c.handleListServiceAccountError(err)
		}
		// Example:
		//    To list all service accounts:
		//       filter = func(s string) bool { return true }
		if filter(sa.Name) {
			out = append(out, sa.Name)
		}
	}
	return out, nil
}

func (c *gcpClient) AttachImpersonator(saId, projectId string, impersonatorId string) error {
	saResourceId := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com",
		projectId, saId, projectId)
	policy, err := c.iamClient.GetIamPolicy(context.Background(), &iampb.GetIamPolicyRequest{
		Resource: saResourceId,
	})
	if err != nil {
		return c.handleAttachImpersonatorError(err)
	}
	policy.Add(
		fmt.Sprintf("serviceAccount:%s", c.extractEmail(impersonatorId)),
		iam.RoleName("roles/iam.serviceAccountTokenCreator"))
	_, err = c.iamClient.SetIamPolicy(context.Background(), &iamadmin.SetIamPolicyRequest{
		Resource: saResourceId,
		Policy:   policy,
	})
	if err != nil {
		return c.handleAttachImpersonatorError(err)
	}
	return nil
}

func (c *gcpClient) AttachWorkloadIdentityPool(sa *cmv1.WifServiceAccount, poolId, projectId string) error {
	saResourceId := c.fmtSaResourceId(sa.ServiceAccountId(), projectId)

	projectNum, err := c.ProjectNumberFromId(projectId)
	if err != nil {
		return c.handleAttachWorkloadIdentityPoolError(err)
	}

	policy, err := c.iamClient.GetIamPolicy(context.Background(), &iampb.GetIamPolicyRequest{
		Resource: saResourceId,
	})
	if err != nil {
		return c.handleAttachWorkloadIdentityPoolError(err)
	}
	for _, openshiftServiceAccount := range sa.CredentialRequest().ServiceAccountNames() {
		policy.Add(
			//nolint:lll
			fmt.Sprintf(
				"principal://iam.googleapis.com/projects/%d/locations/global/workloadIdentityPools/%s/subject/system:serviceaccount:%s:%s",
				projectNum, poolId, sa.CredentialRequest().SecretRef().Namespace(), openshiftServiceAccount,
			),
			iam.RoleName("roles/iam.workloadIdentityUser"))
	}
	_, err = c.iamClient.SetIamPolicy(context.Background(), &iamadmin.SetIamPolicyRequest{
		Resource: saResourceId,
		Policy:   policy,
	})
	if err != nil {
		return c.handleAttachWorkloadIdentityPoolError(err)
	}
	return nil
}

//   - secretResource: The resource name of the secret is in the format
//     `projects/*/secrets/*`
//   - secretData: Can be anything.
func (c *gcpClient) SaveSecret(secretName, secretProject string, secretData []byte) error {
	_, err := c.secretManager.Projects.Secrets.Create("projects/"+secretProject, &secretmanager.Secret{
		// This is an undocumented required field.
		// https://github.com/hashicorp/terraform-provider-google/issues/11395
		Replication: &secretmanager.Replication{Automatic: &secretmanager.Automatic{}},
	}).SecretId(secretName).Do()
	if err != nil {
		err = c.handleSaveSecretError(err)
		if err != nil {
			return err
		}
	}
	_, err = c.secretManager.Projects.Locations.Secrets.AddVersion(
		fmt.Sprintf("projects/%s/secrets/%s", secretProject, secretName),
		&secretmanager.AddSecretVersionRequest{
			Payload: &secretmanager.SecretPayload{
				Data: base64.StdEncoding.EncodeToString(secretData),
			},
		}).Do()
	if err != nil {
		return c.handleSaveSecretError(err)
	}
	return nil
}

//   - name: The resource name of the secret is in the format
//     `projects/*/secrets/*/versions/*` or
//     `projects/*/locations/*/secrets/*/versions/*`.
//     `projects/*/secrets/*/versions/latest` or
//     `projects/*/locations/*/secrets/*/versions/latest` is an alias to
//     the most recently created SecretVersion.
func (c *gcpClient) RetreiveSecret(secretId string, projectId string) ([]byte, error) {
	secretResource := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectId, secretId)
	resp, err := c.secretManager.Projects.Secrets.Versions.Access(secretResource).Do()
	if err != nil {
		c.handleRetrieveSecretError(err)
	}
	return base64.StdEncoding.DecodeString(resp.Payload.Data)
}

type WorkloadIdentityPoolSpec struct {
	Audience               []string
	IssuerUrl              string
	PoolName               string
	ProjectId              string
	Jwks                   string
	PoolIdentityProviderId string
}

func (c *gcpClient) CreateWorkloadIdentityPool2(spec WorkloadIdentityPoolSpec) error {
	// Note: The parent parameter should be in the following format:
	// projects/*/locations/*
	// https://cloud.google.com/iam/docs/reference/rest/v1/projects.locations.workloadIdentityPools/create
	if _, err := c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Create(
		fmt.Sprintf("projects/%s/locations/global", spec.ProjectId), &iamv1.WorkloadIdentityPool{
			DisplayName: spec.PoolName,
			Description: "Workload Identity pool created by prototype",
		}).WorkloadIdentityPoolId(spec.PoolName).Do(); err != nil {
		if err != nil {
			return err
		}
	}
	if _, err := c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Providers.Create(
		fmt.Sprintf("projects/%s/locations/global/workloadIdentityPools/%s", spec.ProjectId, spec.PoolName),
		&iamv1.WorkloadIdentityPoolProvider{
			AttributeMapping: map[string]string{
				"google.subject": "assertion.sub",
			},
			Description: "Identity Provider created by prototype",
			Oidc: &iamv1.Oidc{
				AllowedAudiences: []string{
					"openshift",
				},
				IssuerUri: spec.IssuerUrl,
				JwksJson:  spec.Jwks,
			},
		}).WorkloadIdentityPoolProviderId(spec.PoolIdentityProviderId).Do(); err != nil {
		return err
	}
	return nil
}

//nolint:lll
func (c *gcpClient) CreateWorkloadIdentityPool(ctx context.Context, parent, poolID string, pool *iamv1.WorkloadIdentityPool) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Create(parent, pool).WorkloadIdentityPoolId(poolID).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) GetWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPool, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Get(resource).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) DeleteWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Delete(resource).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) UndeleteWorkloadIdentityPool(ctx context.Context, resource string, request *iamv1.UndeleteWorkloadIdentityPoolRequest) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Undelete(resource, request).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) CreateWorkloadIdentityProvider(ctx context.Context, parent, providerID string, provider *iamv1.WorkloadIdentityPoolProvider) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Providers.Create(parent, provider).WorkloadIdentityPoolProviderId(providerID).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) GetWorkloadIdentityProvider(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPoolProvider, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Providers.Get(resource).Context(ctx).Do()
}

func (c *gcpClient) ProjectNumberFromId(projectId string) (int64, error) {
	project, err := c.cloudResourceManager.Projects.Get(projectId).Do()
	if err != nil {
		return 0, err
	}
	return project.ProjectNumber, nil
}

//nolint:lll
func (c *gcpClient) GetProjectIamPolicy(projectName string, request *cloudresourcemanager.GetIamPolicyRequest) (*cloudresourcemanager.Policy, error) {
	return c.cloudResourceManager.Projects.GetIamPolicy(projectName, request).Context(context.Background()).Do()
}

//nolint:lll
func (c *gcpClient) SetProjectIamPolicy(svcAcctResource string, request *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error) {
	return c.cloudResourceManager.Projects.SetIamPolicy(svcAcctResource, request).Context(context.Background()).Do()
}

func (c *gcpClient) GetRole(ctx context.Context, request *adminpb.GetRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.GetRole(ctx, request)
}

func (c *gcpClient) CreateRole(ctx context.Context, request *adminpb.CreateRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.CreateRole(ctx, request)
}

func (c *gcpClient) UpdateRole(ctx context.Context, request *adminpb.UpdateRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.UpdateRole(ctx, request)
}

func (c *gcpClient) DeleteRole(ctx context.Context, request *adminpb.DeleteRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.DeleteRole(ctx, request)
}

func (c *gcpClient) UndeleteRole(ctx context.Context, request *adminpb.UndeleteRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.UndeleteRole(ctx, request)
}

//nolint:lll
func (c *gcpClient) ListRoles(ctx context.Context, request *adminpb.ListRolesRequest) (*adminpb.ListRolesResponse, error) {
	return c.iamClient.ListRoles(ctx, request)
}
