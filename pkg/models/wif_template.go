package models

type WifTemplate struct {
	Id              string           `json:"id,omitempty"`
	Kind            string           `json:"kind,omitempty"`
	ServiceAccounts []ServiceAccount `json:"service_accounts,omitempty"`
}
