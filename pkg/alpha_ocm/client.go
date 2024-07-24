package alphaocm

/* This package will be replaced with calls to the sdk once available */

import (
	"encoding/json"
	"fmt"
	"os"

	sdk "github.com/openshift-online/ocm-sdk-go"

	"github.com/openshift-online/ocm-cli/pkg/dump"
	"github.com/openshift-online/ocm-cli/pkg/models"
	"github.com/openshift-online/ocm-cli/pkg/ocm"
)

type OcmClient interface {
	CreateWifConfig(models.WifConfigInput) (models.WifConfigOutput, error)
	GetWifConfig(string) (models.WifConfigOutput, error)
	ListWifConfigs() ([]models.WifConfigOutput, error)
	DeleteWifConfig(string) error
}

type ocmClient struct {
	connection *sdk.Connection
}

func NewOcmClient() (OcmClient, error) {
	// Create the connection:
	connection, err := ocm.NewConnection().Build()
	if err != nil {
		return nil, fmt.Errorf("can't create connection: %v", err)
	}
	return &ocmClient{
		connection: connection,
	}, nil
}

func (c *ocmClient) CreateWifConfig(input models.WifConfigInput) (models.WifConfigOutput, error) {
	var wifConfigOutput models.WifConfigOutput

	inputJson, err := json.Marshal(input)
	if err != nil {
		return wifConfigOutput, fmt.Errorf("failed to marshal wif input: %v", err)
	}

	resp, err := c.connection.Post().Path("/api/clusters_mgmt/v1/gcp/wif_configs").Bytes(inputJson).Send()
	if err != nil {
		return wifConfigOutput, fmt.Errorf("can't send request: %v", err)
	}

	status := resp.Status()
	body := resp.Bytes()

	if status >= 400 {
		dump.Pretty(os.Stderr, body)
		return wifConfigOutput, fmt.Errorf("failed to create WIF config: %s", string(body))
	}

	wifConfigOutput, err = models.WifConfigOutputFromJson(body)
	return wifConfigOutput, err
}

func (c *ocmClient) GetWifConfig(id string) (models.WifConfigOutput, error) {
	var wifConfigOutput models.WifConfigOutput
	resp, err := c.connection.Get().Path(fmt.Sprintf("/api/clusters_mgmt/v1/gcp/wif_configs/%s", id)).Send()
	if err != nil {
		return wifConfigOutput, fmt.Errorf("can't send request: %v", err)
	}
	if resp.Status() >= 400 {
		body := resp.Bytes()
		dump.Pretty(os.Stderr, body)
		return wifConfigOutput, fmt.Errorf("failed to list WIF configs: %s", string(body))
	}

	wifConfigOutput, err = models.WifConfigOutputFromJson(resp.Bytes())
	return wifConfigOutput, err
}

func (c *ocmClient) ListWifConfigs() ([]models.WifConfigOutput, error) {
	var wifConfigs []models.WifConfigOutput
	resp, err := c.connection.Get().Path("/api/clusters_mgmt/v1/gcp/wif_configs").Send()
	if err != nil {
		return wifConfigs, fmt.Errorf("can't send request: %v", err)
	}
	if resp.Status() >= 400 {
		body := resp.Bytes()
		dump.Pretty(os.Stderr, body)
		return wifConfigs, fmt.Errorf("failed to list WIF configs: %s", string(body))
	}

	wifConfigsList, err := models.WifConfigOutputListFromJson(resp.Bytes())
	return wifConfigsList.Items, err
}

func (c *ocmClient) DeleteWifConfig(id string) error {
	resp, err := c.connection.Delete().Path(fmt.Sprintf("/api/clusters_mgmt/v1/gcp/wif_configs/%s", id)).Send()
	if err != nil {
		return fmt.Errorf("can't send request: %v", err)
	}
	if resp.Status() >= 400 {
		body := resp.Bytes()
		dump.Pretty(os.Stderr, body)
		return fmt.Errorf("failed to delete WIF config: %s", string(body))
	}
	return nil
}
