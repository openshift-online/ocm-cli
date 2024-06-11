package models

type WifConfigStatus struct {
	ServiceAccounts          []ServiceAccount            `json:"service_accounts,omitempty"`
	State                    string                      `json:"state,omitempty"`
	Summary                  string                      `json:"summary,omitempty"`
	TimeData                 WifTimeData                 `json:"time_data,omitempty"`
	WorkloadIdentityPoolData WifWorkloadIdentityPoolData `json:"workload_identity_pool,omitempty"`
}

type WifWorkloadIdentityPoolData struct {
	IdentityProviderId string `json:"identity_provider_id,omitempty"`
	IssuerUrl          string `json:"issuer_url,omitempty"`
	Jwks               string `json:"jwks,omitempty"`
	PoolId             string `json:"pool_id,omitempty"`
	ProjectId          string `json:"gcp_project_name,omitempty"`
	ProjectNumber      int64  `json:"gcp_project_num,omitempty"`
}

type WifTimeData struct {
	LastChecked string `json:"last_checked,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}
