package gcp

import (
	"fmt"
	"net/http"

	"github.com/googleapis/gax-go/v2/apierror"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
)

func (c *gcpClient) handleAttachImpersonatorError(err error) error {
	pApiError, ok := err.(*apierror.APIError)
	if !ok {
		return fmt.Errorf("Unexpected error")
	}
	return fmt.Errorf(pApiError.Details().String())
}

func (c *gcpClient) handleAttachWorkloadIdentityPoolError(err error) error {
	pApiError, ok := err.(*apierror.APIError)
	if !ok {
		return fmt.Errorf("Unexpected error")
	}
	fmt.Println(pApiError.Error())
	return fmt.Errorf(pApiError.Error())
}

func (c *gcpClient) handleListServiceAccountError(err error) error {
	pApiError, ok := err.(*apierror.APIError)
	if !ok {
		return fmt.Errorf("Unexpected error")
	}
	return fmt.Errorf(pApiError.Details().String())
}

func (c *gcpClient) handleDeleteServiceAccountError(err error, allowMissing bool) error {
	pApiError, ok := err.(*apierror.APIError)
	if !ok {
		return fmt.Errorf("Unexpected error")
	}
	if pApiError.GRPCStatus().Code() == codes.NotFound && allowMissing {
		return nil
	}
	return fmt.Errorf(pApiError.Details().String())
}

func (c *gcpClient) handleRetrieveSecretError(err error) ([]byte, error) {
	gApiError, ok := err.(*googleapi.Error)
	if !ok {
		return []byte{}, fmt.Errorf("Unexpected error")
	}
	return []byte{}, gApiError
}

// Errors that can't be converted to *googleapi.Error are unexpected
// If the secret already exists, this is not considered an error
func (c *gcpClient) handleSaveSecretError(err error) error {
	gApiError, ok := err.(*googleapi.Error)
	if !ok {
		return fmt.Errorf("Unexpected error")
	}
	if gApiError.Code == http.StatusConflict {
		return nil
	}
	return gApiError
}
