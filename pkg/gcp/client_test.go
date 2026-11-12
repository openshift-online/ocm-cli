package gcp

import (
	"context"
	"errors"
	"net/http"
	"testing"

	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

type policyTransport func(*http.Request) (*http.Response, error)

func (transport policyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestProjectIamPolicyUsesCallerContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := policyTransport(func(request *http.Request) (*http.Response, error) {
		if request.Context().Done() != ctx.Done() {
			t.Fatal("policy request did not use caller context")
		}
		cancel()
		return nil, request.Context().Err()
	})
	service, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	client := &gcpClient{cloudResourceManager: service}
	_, err = client.GetProjectIamPolicy(ctx, "test-project", &cloudresourcemanager.GetIamPolicyRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
