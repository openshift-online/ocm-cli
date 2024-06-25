package models

import (
	"encoding/json"
	"fmt"
)

type WifConfigOutput struct {
	Metadata *WifConfigMetadata `json:"metadata,omitempty"`
	Spec     *WifConfigInput    `json:"spec,omitempty"`
	Status   *WifConfigStatus   `json:"status,omitempty"`
}

type WifConfigOutputList struct {
	Items []WifConfigOutput `json:"items"`
}

func WifConfigOutputFromJson(rawWifConfigOutput []byte) (WifConfigOutput, error) {
	var output WifConfigOutput
	err := json.Unmarshal(rawWifConfigOutput, &output)
	return output, err
}

func WifConfigOutputListFromJson(data []byte) (*WifConfigOutputList, error) {
	var wifConfigListOutput WifConfigOutputList
	err := json.Unmarshal(data, &wifConfigListOutput)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling data: %v", err)
	}
	return &wifConfigListOutput, nil

}
