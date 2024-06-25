package models

type WifConfigMetadata struct {
	DisplayName  string                         `json:"display_name,omitempty"`
	Id           string                         `json:"id,omitempty"`
	Organization *WifConfigMetadataOrganization `json:"organization,omitempty"`
}
