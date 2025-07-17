package billing

import (
	"github.com/openshift-online/ocm-cli/pkg/ocm"
	sdk "github.com/openshift-online/ocm-sdk-go"
	amv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
)

const (
	StandardSubscriptionType       = "standard"
	MarketplaceGcpSubscriptionType = "marketplace-gcp"
)

var ValidSubscriptionTypes = []string{
	StandardSubscriptionType,
	MarketplaceGcpSubscriptionType,
}

func GetBillingModel(connection *sdk.Connection, billingModelID string) (*amv1.BillingModelItem, error) {
	bilingModel, err := ocm.SendTypedAndHandleDeprecation(
		connection.AccountsMgmt().V1().BillingModels().BillingModel(billingModelID).Get())
	if err != nil {
		return nil, err
	}
	return bilingModel.Body(), nil
}

func GetBillingModels(connection *sdk.Connection) ([]*amv1.BillingModelItem, error) {
	response, err := ocm.SendTypedAndHandleDeprecation(connection.AccountsMgmt().V1().BillingModels().List())
	if err != nil {
		return nil, err
	}
	billingModels := response.Items().Slice()
	var validBillingModel []*amv1.BillingModelItem
	for _, billingModel := range billingModels {
		for _, validSubscriptionTypeId := range ValidSubscriptionTypes {
			if billingModel.ID() == validSubscriptionTypeId {
				validBillingModel = append(validBillingModel, billingModel)
			}
		}
	}
	return validBillingModel, nil
}
