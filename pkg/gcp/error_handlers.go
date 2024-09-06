package gcp

import (
	"fmt"

	"github.com/googleapis/gax-go/v2/apierror"
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
