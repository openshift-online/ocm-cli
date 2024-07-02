package models

type WifConfigList struct {
	Items []WifConfigOutput `json:"items,omitempty"`
	Page  int32             `json:"page,omitempty"`
	Total int32             `json:"total,omitempty"`
}
