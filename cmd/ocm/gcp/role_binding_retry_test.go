package gcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/gax-go/v2/apierror"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"

	gcputils "github.com/openshift-online/ocm-cli/pkg/gcp"
)

func bindingAPIError(code int, reason string) *googleapi.Error {
	failure := &googleapi.Error{Code: code}
	if reason != "" {
		failure.Errors = []googleapi.ErrorItem{{Reason: reason}}
	}
	return failure
}

func bindingErrorInfo(reason string) interface{} {
	return map[string]interface{}{
		"@type":    "type.googleapis.com/google.rpc.ErrorInfo",
		"reason":   reason,
		"metadata": map[string]interface{}{"consumer": "projects/private-example"},
	}
}

func TestRoleBindingErrorClassification(t *testing.T) {
	t.Parallel()
	quotaDetail := map[string]interface{}{
		"@type":      "type.googleapis.com/google.rpc.QuotaFailure",
		"violations": []interface{}{map[string]interface{}{"description": "private-example"}},
	}
	tests := []struct {
		name  string
		err   error
		retry bool
	}{
		{"permission", bindingAPIError(403, "forbidden"), false},
		{"authentication", bindingAPIError(401, "authError"), false},
		{"validation", bindingAPIError(400, "invalid"), false},
		{"not found without propagation evidence", bindingAPIError(404, "notFound"), false},
		{"unknown", errors.New("unexpected private-example"), false},
		{"quota", bindingAPIError(403, "quotaExceeded"), false},
		{"quota on 429", bindingAPIError(429, "quotaExceeded"), false},
		{"daily quota", bindingAPIError(429, "dailyLimitExceeded"), false},
		{"capacity quota", bindingAPIError(503, "RESOURCE_QUOTA_EXCEEDED"), false},
		{"unregistered rate", bindingAPIError(403, "rateLimitExceededUnreg"), false},
		{"bare 403", bindingAPIError(403, ""), false},
		{"bare 429", bindingAPIError(429, ""), false},
		{"unknown 429", bindingAPIError(429, "private-example"), false},
		{"rate on 403", bindingAPIError(403, "rateLimitExceeded"), true},
		{"user rate on 429", bindingAPIError(429, "userRateLimitExceeded"), true},
		{"overall API rate", bindingAPIError(429, "servingLimitExceeded"), true},
		{"structured rate", &googleapi.Error{Code: 429,
			Details: []interface{}{bindingErrorInfo("RATE_LIMIT_EXCEEDED")}}, true},
		{"structured quota", &googleapi.Error{Code: 429,
			Details: []interface{}{bindingErrorInfo("RESOURCE_QUOTA_EXCEEDED")}}, false},
		{"quota violation", &googleapi.Error{Code: 429,
			Details: []interface{}{quotaDetail}}, false},
		{"explicit rate with quota details", &googleapi.Error{Code: 429,
			Details: []interface{}{bindingErrorInfo("RATE_LIMIT_EXCEEDED"), quotaDetail}}, true},
		{"allocation overrides rate", &googleapi.Error{Code: 429,
			Details: []interface{}{bindingErrorInfo("RATE_LIMIT_EXCEEDED"),
				bindingErrorInfo("RESOURCE_QUOTA_EXCEEDED"), quotaDetail}}, false},
		{"unknown structured reason", &googleapi.Error{Code: 503,
			Details: []interface{}{bindingErrorInfo("private-example")}}, false},
		{"internal", bindingAPIError(500, "internalError"), true},
		{"gateway", bindingAPIError(502, ""), true},
		{"unavailable", bindingAPIError(503, "backendError"), true},
		{"timeout", bindingAPIError(504, ""), true},
		{"generic conflict", bindingAPIError(409, ""), false},
		{"aborted conflict", &googleapi.Error{Code: 409,
			Body: `{"error":{"status":"ABORTED","message":"private-example"}}`}, true},
		{"already exists conflict", &googleapi.Error{Code: 409,
			Body: `{"error":{"status":"ALREADY_EXISTS"}}`}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			attempts, waits := 0, 0
			err := retryRoleBinding(context.Background(), nil, time.Minute, func(context.Context) error {
				attempts++
				if attempts == 1 {
					return fmt.Errorf("wrapped: %w", test.err)
				}
				return nil
			}, func(context.Context, time.Duration) error { waits++; return nil })
			if test.retry {
				if err != nil || attempts != 2 || waits != 1 {
					t.Fatalf("expected retry and success: err=%v attempts=%d waits=%d", err, attempts, waits)
				}
			} else if !errors.Is(err, test.err) || attempts != 1 || waits != 0 {
				t.Fatalf("expected immediate failure with cause: err=%v attempts=%d waits=%d", err, attempts, waits)
			}
		})
	}
}

func TestRoleBindingDecodedSDKResponses(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		code  int
		body  string
		retry bool
	}{
		{"revision conflict", 409,
			`{"error":{"code":409,"status":"ABORTED","errors":[{"reason":"conflict"}]}}`, true},
		{"generic conflict", 409,
			`{"error":{"code":409,"errors":[{"reason":"conflict"}]}}`, false},
		{"structured throttle", 429,
			`{"error":{"code":429,"details":[` +
				`{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RATE_LIMIT_EXCEEDED"}]}}`, true},
		{"structured resource quota", 429,
			`{"error":{"code":429,"details":[` +
				`{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RESOURCE_QUOTA_EXCEEDED"}]}}`, false},
		{"explicit throttle with quota details", 429,
			`{"error":{"code":429,"details":[` +
				`{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RATE_LIMIT_EXCEEDED"},` +
				`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[]}]}}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			decoded := googleapi.CheckResponseWithBody(&http.Response{StatusCode: test.code}, []byte(test.body))
			if diagnostic := classifyRoleBindingError(decoded); diagnostic.retry != test.retry {
				t.Fatalf("retry=%v want=%v diagnostic=%s", diagnostic.retry, test.retry, diagnostic)
			}
		})
	}
}

func TestRoleBindingInitialSuccess(t *testing.T) {
	t.Parallel()
	attempts := 0
	err := retryRoleBinding(context.Background(), nil, time.Minute, func(context.Context) error {
		attempts++
		return nil
	}, func(context.Context, time.Duration) error {
		t.Fatal("unexpected wait after success")
		return nil
	})
	if err != nil || attempts != 1 {
		t.Fatalf("err=%v attempts=%d", err, attempts)
	}
}

func TestRoleBindingBoundedRetries(t *testing.T) {
	t.Parallel()
	attempts, waits := 0, 0
	var last *googleapi.Error
	var output bytes.Buffer
	err := retryRoleBinding(context.Background(), log.New(&output, "", 0), time.Minute,
		func(context.Context) error {
			attempts++
			last = bindingAPIError(503, "backendError")
			return last
		}, func(_ context.Context, delay time.Duration) error {
			waits++
			if delay <= 0 || delay > 32*time.Second {
				t.Fatalf("unbounded backoff %s", delay)
			}
			return nil
		})
	var typed *googleapi.Error
	if !errors.Is(err, last) || !errors.As(err, &typed) || typed != last ||
		attempts != roleBindingMaxAttempts || waits != roleBindingMaxAttempts-1 {
		t.Fatalf("lost final cause or retry bound: err=%v attempts=%d waits=%d", err, attempts, waits)
	}
	if strings.Count(output.String(), "IAM role binding attempt=") != attempts ||
		!strings.Contains(output.String(), "attempt=12 stopping") {
		t.Fatalf("missing per-attempt or exhaustion diagnostic: %s", output.String())
	}
	if !strings.Contains(err.Error(), "retry budget exhausted") {
		t.Fatal("missing safe exhaustion summary")
	}
}

func TestRoleBindingCancellation(t *testing.T) {
	t.Parallel()
	t.Run("before first attempt", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := retryRoleBinding(ctx, nil, time.Minute, func(context.Context) error {
			t.Fatal("operation called after cancellation")
			return nil
		}, waitRoleBinding)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("missing cancellation: %v", err)
		}
	})
	t.Run("during wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		last := bindingAPIError(503, "backendError")
		err := retryRoleBinding(ctx, nil, time.Minute,
			func(context.Context) error { return last },
			func(ctx context.Context, delay time.Duration) error {
				cancel()
				return waitRoleBinding(ctx, time.Hour+delay)
			})
		if !errors.Is(err, context.Canceled) || !errors.Is(err, last) {
			t.Fatalf("missing cancellation or final cause: %v", err)
		}
	})
	t.Run("deadline during operation", func(t *testing.T) {
		last := bindingAPIError(503, "backendError")
		err := retryRoleBinding(context.Background(), nil, time.Millisecond,
			func(ctx context.Context) error {
				<-ctx.Done()
				return last
			}, func(context.Context, time.Duration) error {
				t.Fatal("wait called after deadline")
				return nil
			})
		if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, last) {
			t.Fatalf("missing deadline or final cause: %v", err)
		}
	})
	t.Run("deadline during wait", func(t *testing.T) {
		last := bindingAPIError(503, "backendError")
		err := retryRoleBinding(context.Background(), nil, 10*time.Millisecond,
			func(context.Context) error { return last },
			func(ctx context.Context, _ time.Duration) error { return waitRoleBinding(ctx, time.Hour) })
		if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, last) ||
			!strings.Contains(err.Error(), "retry deadline exceeded") {
			t.Fatalf("missing deadline, final cause, or safe summary: %v", err)
		}
	})
}

func TestRoleBindingDiagnosticsAreSanitized(t *testing.T) {
	t.Parallel()
	const sensitive = "secret-example-principal@example.invalid"
	tests := []*googleapi.Error{
		{Code: 403, Message: "Permission denied for " + sensitive,
			Body: sensitive, Errors: []googleapi.ErrorItem{{Reason: sensitive, Message: sensitive}},
			Details: []interface{}{bindingErrorInfo(sensitive)}},
		{Code: 429, Message: "Rate limit exceeded Bearer " + sensitive,
			Body:    `{"bindings":["` + sensitive + `"]}`,
			Details: []interface{}{bindingErrorInfo("RATE_LIMIT_EXCEEDED")}},
		{Code: 403, Message: "Bearer " + sensitive + "\nforged-log-line",
			Errors: []googleapi.ErrorItem{{Reason: "quotaExceeded"}}},
	}
	for _, failure := range tests {
		var output bytes.Buffer
		wrapped := fmt.Errorf("raw wrapper %s: %w", sensitive, failure)
		sdkError, ok := apierror.ParseError(wrapped, true)
		if !ok {
			t.Fatal("could not create SDK error")
		}
		err := retryRoleBinding(context.Background(), log.New(&output, "", 0), time.Minute,
			func(context.Context) error { return sdkError },
			func(context.Context, time.Duration) error { return context.DeadlineExceeded })
		if !errors.Is(err, failure) {
			t.Fatal("lost typed SDK cause")
		}
		combined := output.String() + err.Error()
		for _, forbidden := range []string{sensitive, "private-example", "Bearer", "bindings", "forged-log-line"} {
			if strings.Contains(combined, forbidden) {
				t.Fatalf("sensitive data in diagnostic: %s", forbidden)
			}
		}
		if !strings.Contains(combined, "attempt=1") || !strings.Contains(combined, "code=") ||
			!strings.Contains(combined, "reason=") {
			t.Fatal("missing safe API diagnostic")
		}
		if strings.HasPrefix(failure.Message, "Permission denied") &&
			!strings.Contains(combined, `message="Permission denied"`) {
			t.Fatal("lost safe message evidence")
		}
	}
}

type bindingPolicyClient struct {
	gcputils.GcpClient
	reads  int
	writes int
	fail   error
}

func (client *bindingPolicyClient) GetProjectIamPolicy(
	ctx context.Context, _ string, _ *cloudresourcemanager.GetIamPolicyRequest,
) (*cloudresourcemanager.Policy, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	client.reads++
	return &cloudresourcemanager.Policy{Etag: fmt.Sprintf("revision-%d", client.reads)}, nil
}

func (client *bindingPolicyClient) SetProjectIamPolicy(
	_ context.Context, _ string, request *cloudresourcemanager.SetIamPolicyRequest,
) (*cloudresourcemanager.Policy, error) {
	client.writes++
	if request.Policy.Etag != fmt.Sprintf("revision-%d", client.writes) {
		return nil, errors.New("stale policy revision")
	}
	if client.writes == 1 {
		return nil, client.fail
	}
	return request.Policy, nil
}

func TestRoleBindingRetriesEntirePolicyTransaction(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		fail *googleapi.Error
		want int
	}{
		{"conflict", &googleapi.Error{Code: 409, Body: `{"error":{"status":"ABORTED"}}`}, 2},
		{"permanent", bindingAPIError(403, "quotaExceeded"), 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &bindingPolicyClient{fail: test.fail}
			config, err := cmv1.NewWifConfig().Gcp(cmv1.NewWifGcp().ProjectId("test-project")).Build()
			if err != nil {
				t.Fatal(err)
			}
			role, err := cmv1.NewWifRole().RoleId("viewer").Predefined(true).Build()
			if err != nil {
				t.Fatal(err)
			}
			subject := &shim{gcpClient: client, wifConfig: config}
			err = subject.bindRolesToPrincipal(context.Background(), log.New(io.Discard, "", 0),
				"serviceAccount:test@example.invalid", []*cmv1.WifRole{role})
			if test.want == 2 && err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if test.want == 1 && !errors.Is(err, test.fail) {
				t.Fatalf("lost API cause: %v", err)
			}
			if client.reads != test.want || client.writes != test.want {
				t.Fatalf("reads=%d writes=%d want=%d", client.reads, client.writes, test.want)
			}
		})
	}
}

func TestSupportRoleBindingPermanentFailureIsNotRetriedByOuterHelper(t *testing.T) {
	t.Parallel()
	failure := bindingAPIError(403, "quotaExceeded")
	client := &bindingPolicyClient{fail: failure}
	config, err := cmv1.NewWifConfig().Gcp(cmv1.NewWifGcp().ProjectId("test-project").Support(
		cmv1.NewWifSupport().Principal("support@example.invalid").Roles(
			cmv1.NewWifRole().RoleId("viewer").Predefined(true)))).Build()
	if err != nil {
		t.Fatal(err)
	}
	subject := &shim{gcpClient: client, wifConfig: config}
	err = subject.GrantSupportAccess(context.Background(), log.New(io.Discard, "", 0))
	if !errors.Is(err, failure) || client.reads != 1 || client.writes != 1 {
		t.Fatalf("expected single attempt retaining API cause: err=%v reads=%d writes=%d",
			err, client.reads, client.writes)
	}
}
