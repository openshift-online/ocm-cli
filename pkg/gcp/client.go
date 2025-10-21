package gcp

import (
	"context"
	"fmt"

	iamadmin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"cloud.google.com/go/iam/apiv1/iampb"
	"cloud.google.com/go/storage"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"

	iamv1 "google.golang.org/api/iam/v1"
	secretmanager "google.golang.org/api/secretmanager/v1"
)

//nolint:lll
type GcpClient interface {
	CreateRole(context.Context, *adminpb.CreateRoleRequest) (*adminpb.Role, error)
	CreateServiceAccount(ctx context.Context, request *adminpb.CreateServiceAccountRequest) (*adminpb.ServiceAccount, error)
	CreateWorkloadIdentityPool(ctx context.Context, parent, poolID string, pool *iamv1.WorkloadIdentityPool) (*iamv1.Operation, error)
	CreateWorkloadIdentityProvider(ctx context.Context, parent, providerID string, provider *iamv1.WorkloadIdentityPoolProvider) (*iamv1.Operation, error)
	DeleteServiceAccount(ctx context.Context, saName string, project string, allowMissing bool) error
	DeleteWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.Operation, error)
	EnableServiceAccount(ctx context.Context, serviceAccountId string, projectId string) error
	EnableWorkloadIdentityPool(ctx context.Context, poolId string) error
	GetProjectIamPolicy(ctx context.Context, projectName string, request *cloudresourcemanager.GetIamPolicyRequest) (*cloudresourcemanager.Policy, error)
	GetRole(context.Context, *adminpb.GetRoleRequest) (*adminpb.Role, error)
	GetServiceAccount(ctx context.Context, request *adminpb.GetServiceAccountRequest) (*adminpb.ServiceAccount, error)
	GetServiceAccountAccessPolicy(ctx context.Context, saId string) (Policy, error)
	GetWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPool, error)
	GetWorkloadIdentityProvider(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPoolProvider, error)
	ProjectNumberFromId(ctx context.Context, projectId string) (int64, error)
	SetProjectIamPolicy(ctx context.Context, svcAcctResource string, request *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error)
	SetServiceAccountAccessPolicy(ctx context.Context, policy Policy) error
	UndeleteRole(context.Context, *adminpb.UndeleteRoleRequest) (*adminpb.Role, error)
	UndeleteWorkloadIdentityPool(ctx context.Context, resource string, request *iamv1.UndeleteWorkloadIdentityPoolRequest) (*iamv1.Operation, error)
	UpdateRole(context.Context, *adminpb.UpdateRoleRequest) (*adminpb.Role, error)
	UpdateWorkloadIdentityPoolOidcIdentityProvider(ctx context.Context, provider *iamv1.WorkloadIdentityPoolProvider) error
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

func (c *gcpClient) CreateRole(ctx context.Context, request *adminpb.CreateRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.CreateRole(ctx, request)
}

func (c *gcpClient) CreateServiceAccount(
	ctx context.Context,
	request *adminpb.CreateServiceAccountRequest,
) (*adminpb.ServiceAccount, error) {
	svcAcct, err := c.iamClient.CreateServiceAccount(ctx, request)
	return svcAcct, err
}

//nolint:lll
func (c *gcpClient) CreateWorkloadIdentityPool(ctx context.Context, parent, poolID string, pool *iamv1.WorkloadIdentityPool) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Create(parent, pool).WorkloadIdentityPoolId(poolID).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) CreateWorkloadIdentityProvider(ctx context.Context, parent, providerID string, provider *iamv1.WorkloadIdentityPoolProvider) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Providers.Create(parent, provider).WorkloadIdentityPoolProviderId(providerID).Context(ctx).Do()
}

func (c *gcpClient) DeleteServiceAccount(ctx context.Context, saName string, project string, allowMissing bool) error {
	name := fmt.Sprintf("projects/%s/serviceAccounts/%s@%s.iam.gserviceaccount.com", project, saName, project)
	err := c.iamClient.DeleteServiceAccount(ctx, &adminpb.DeleteServiceAccountRequest{
		Name: name,
	})
	if err != nil {
		return c.handleDeleteServiceAccountError(err, allowMissing)
	}
	return nil
}

//nolint:lll
func (c *gcpClient) DeleteWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Delete(resource).Context(ctx).Do()
}

func (c *gcpClient) EnableServiceAccount(
	ctx context.Context,
	serviceAccountId string,
	projectId string,
) error {
	_, err := c.oldIamClient.Projects.ServiceAccounts.Enable(
		FmtSaResourceId(serviceAccountId, projectId),
		&iamv1.EnableServiceAccountRequest{},
	).Do()
	if err != nil {
		return c.fmtGoogleApiError(err)
	}
	return nil
}

func (c *gcpClient) EnableWorkloadIdentityPool(
	ctx context.Context,
	poolId string,
) error {
	_, err := c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Patch(
		poolId,
		&iamv1.WorkloadIdentityPool{
			Disabled: false,
		},
	).UpdateMask("disabled").Do()
	if err != nil {
		return c.fmtGoogleApiError(err)
	}
	return nil
}

//nolint:lll
func (c *gcpClient) GetProjectIamPolicy(
	ctx context.Context,
	projectName string,
	request *cloudresourcemanager.GetIamPolicyRequest,
) (*cloudresourcemanager.Policy, error) {
	return c.cloudResourceManager.Projects.GetIamPolicy(projectName, request).Context(context.Background()).Do()
}

func (c *gcpClient) GetRole(ctx context.Context, request *adminpb.GetRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.GetRole(ctx, request)
}

func (c *gcpClient) GetServiceAccount(
	ctx context.Context,
	request *adminpb.GetServiceAccountRequest,
) (*adminpb.ServiceAccount, error) {
	return c.iamClient.GetServiceAccount(ctx, request)
}

func (c *gcpClient) GetServiceAccountAccessPolicy(
	ctx context.Context,
	saId string,
) (Policy, error) {
	libPolicy, err := c.iamClient.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
		Resource: saId,
	})
	if err != nil {
		return nil, c.handleApiNotFoundError(err)
	}
	return &policy{
		resourceId: saId,
		policy:     libPolicy,
	}, nil
}

//nolint:lll
func (c *gcpClient) GetWorkloadIdentityPool(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPool, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Get(resource).Context(ctx).Do()
}

//nolint:lll
func (c *gcpClient) GetWorkloadIdentityProvider(ctx context.Context, resource string) (*iamv1.WorkloadIdentityPoolProvider, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Providers.Get(resource).Context(ctx).Do()
}

func (c *gcpClient) ProjectNumberFromId(ctx context.Context, projectId string) (int64, error) {
	project, err := c.cloudResourceManager.Projects.Get(projectId).Do()
	if err != nil {
		return 0, err
	}
	return project.ProjectNumber, nil
}

//nolint:lll
func (c *gcpClient) SetProjectIamPolicy(ctx context.Context, svcAcctResource string, request *cloudresourcemanager.SetIamPolicyRequest) (*cloudresourcemanager.Policy, error) {
	return c.cloudResourceManager.Projects.SetIamPolicy(svcAcctResource, request).Context(ctx).Do()
}

func (c *gcpClient) SetServiceAccountAccessPolicy(
	ctx context.Context,
	policy Policy,
) error {
	_, err := c.iamClient.SetIamPolicy(ctx, &iamadmin.SetIamPolicyRequest{
		Resource: policy.ResourceId(),
		Policy:   policy.IamPolicy(),
	})
	if err != nil {
		return c.handleApiError(err)
	}
	return nil
}

func (c *gcpClient) UndeleteRole(ctx context.Context, request *adminpb.UndeleteRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.UndeleteRole(ctx, request)
}

//nolint:lll
func (c *gcpClient) UndeleteWorkloadIdentityPool(ctx context.Context, resource string, request *iamv1.UndeleteWorkloadIdentityPoolRequest) (*iamv1.Operation, error) {
	return c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Undelete(resource, request).Context(ctx).Do()
}

func (c *gcpClient) UpdateRole(ctx context.Context, request *adminpb.UpdateRoleRequest) (*adminpb.Role, error) {
	return c.iamClient.UpdateRole(ctx, request)
}

func (c *gcpClient) UpdateWorkloadIdentityPoolOidcIdentityProvider(
	ctx context.Context,
	provider *iamv1.WorkloadIdentityPoolProvider,
) error {
	_, err := c.oldIamClient.Projects.Locations.WorkloadIdentityPools.Providers.Patch(
		provider.Name,
		provider,
	).UpdateMask("attributeMapping,description,displayName,disabled,state,oidc").Do()
	if err != nil {
		return c.fmtGoogleApiError(err)
	}
	return nil
}
