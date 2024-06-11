package models

type ServiceAccount struct {
	AccessMethod      string            `json:"access_method,omitempty"`
	CredentialRequest CredentialRequest `json:"credential_request,omitempty"`
	Id                string            `json:"id,omitempty"`
	Kind              string            `json:"kind,omitempty"`
	OsdRole           string            `json:"osd_role,omitempty"`
	Roles             []Role            `json:"roles,omitempty"`
}

type CredentialRequest struct {
	SecretRef           SecretRef
	ServiceAccountNames []string
}

type SecretRef struct {
	Name      string
	Namespace string
}

func (s ServiceAccount) GetId() string {
	serviceAccountID := "z-" + s.Id
	if len(serviceAccountID) > 30 {
		serviceAccountID = serviceAccountID[:30]
	}
	return serviceAccountID
}

func (s ServiceAccount) GetSecretName() string {
	return s.CredentialRequest.SecretRef.Name
}

func (s ServiceAccount) GetSecretNamespace() string {
	return s.CredentialRequest.SecretRef.Namespace
}

func (s ServiceAccount) GetServiceAccountNames() []string {
	return s.CredentialRequest.ServiceAccountNames
}
