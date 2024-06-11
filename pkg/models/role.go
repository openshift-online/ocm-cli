package models

type Role struct {
	Id          string   `json:"id,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Predefined  bool     `json:"predefined,omitempty"`
}
