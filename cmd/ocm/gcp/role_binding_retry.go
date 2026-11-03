package gcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/googleapi"
)

const roleBindingMaxAttempts = 12

type roleBindingDiagnostic struct {
	code    int
	reason  string
	message string
	advice  string
	retry   bool
}

func (diagnostic roleBindingDiagnostic) String() string {
	return fmt.Sprintf("code=%d reason=%s message=%q; %s",
		diagnostic.code, diagnostic.reason, diagnostic.message, diagnostic.advice)
}

type roleBindingError struct {
	diagnostic roleBindingDiagnostic
	cause      error
}

func (failure *roleBindingError) Error() string {
	stop := "non-retryable API failure"
	switch {
	case errors.Is(failure.cause, context.Canceled):
		stop = "retry cancelled"
	case errors.Is(failure.cause, context.DeadlineExceeded):
		stop = "retry deadline exceeded"
	case failure.diagnostic.retry:
		stop = "retry budget exhausted"
	}
	return fmt.Sprintf("IAM role binding failed (%s): %s", stop, failure.diagnostic)
}

func (failure *roleBindingError) Unwrap() error {
	return failure.cause
}

func classifyRoleBindingError(err error) roleBindingDiagnostic {
	diagnostic := roleBindingDiagnostic{
		reason: "unknown", message: "[redacted]",
		advice: "Check API permissions, resource configuration, and quota before retrying.",
	}
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return diagnostic
	}
	if apiErr.Code >= 100 && apiErr.Code <= 599 {
		diagnostic.code = apiErr.Code
	}
	for _, phrase := range []string{
		"Permission denied", "The caller does not have permission", "Rate limit exceeded",
		"Quota exceeded", "Internal error encountered", "Service unavailable",
	} {
		if strings.Contains(apiErr.Message, phrase) {
			diagnostic.message = phrase
			break
		}
	}
	var reasons []string
	for _, item := range apiErr.Errors {
		reasons = append(reasons, item.Reason)
	}
	quotaFailure := false
	for _, detail := range apiErr.Details {
		fields, ok := detail.(map[string]interface{})
		if !ok {
			continue
		}
		switch fields["@type"] {
		case "type.googleapis.com/google.rpc.ErrorInfo":
			if reason, ok := fields["reason"].(string); ok {
				reasons = append(reasons, reason)
			}
		case "type.googleapis.com/google.rpc.QuotaFailure":
			quotaFailure = true
		}
	}
	throttled, unknownReason := false, false
	for _, reason := range reasons {
		switch reason {
		case "forbidden", "authError", "invalid", "notFound", "conditionNotMet":
			diagnostic.reason = reason
			return diagnostic
		case "quotaExceeded", "dailyLimitExceeded", "dailyLimitExceeded402", "quotaExceeded402",
			"dailyLimitExceededUnreg", "rateLimitExceededUnreg", "userRateLimitExceededUnreg",
			"QUOTA_EXCEEDED", "RESOURCE_QUOTA_EXCEEDED":
			diagnostic.reason = reason
			diagnostic.advice = "Check quota, capacity, billing, or API registration before retrying; " +
				"this failure is not classified as transient."
			return diagnostic
		case "rateLimitExceeded", "userRateLimitExceeded", "servingLimitExceeded", "RATE_LIMIT_EXCEEDED":
			diagnostic.reason = reason
			throttled = true
		case "backendError", "internalError", "conflict":
			diagnostic.reason = reason
		default:
			unknownReason = true
		}
	}
	if quotaFailure && !throttled {
		diagnostic.reason = "QuotaFailure"
		diagnostic.advice = "Inspect quota limits; structured quota exhaustion is not assumed to be temporary throttling."
		return diagnostic
	}
	if unknownReason {
		diagnostic.reason = "unknown"
		return diagnostic
	}
	switch apiErr.Code {
	case 403, 429:
		if throttled {
			diagnostic.retry = true
			diagnostic.advice = "Request rate limited; retry with bounded backoff."
		}
	case 500, 502, 503, 504:
		diagnostic.retry = true
		diagnostic.advice = "Temporary API failure; retry the policy read-modify-write with backoff."
	case 409:
		var response struct {
			Error struct {
				Status string `json:"status"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(apiErr.Body), &response) == nil && response.Error.Status == "ABORTED" {
			diagnostic.reason = "ABORTED"
			diagnostic.retry = true
			diagnostic.advice = "Policy revision conflict; reread the policy and reapply bindings before writing."
		}
	}
	return diagnostic
}

func retryRoleBinding(
	ctx context.Context,
	logger *log.Logger,
	timeout time.Duration,
	operation func(context.Context) error,
	wait func(context.Context, time.Duration) error,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	backoff := gax.Backoff{Initial: time.Second, Max: 32 * time.Second, Multiplier: 2}
	var lastErr error
	var diagnostic roleBindingDiagnostic
	for attempt := 1; attempt <= roleBindingMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr == nil {
				return err
			}
			return &roleBindingError{diagnostic: diagnostic, cause: errors.Join(err, lastErr)}
		}
		lastErr = operation(ctx)
		if lastErr == nil {
			return nil
		}
		diagnostic = classifyRoleBindingError(lastErr)
		action := "stopping"
		if diagnostic.retry && attempt < roleBindingMaxAttempts && ctx.Err() == nil {
			action = "retrying"
		}
		if logger != nil {
			logger.Printf("IAM role binding attempt=%d %s: %s", attempt, action, diagnostic)
		}
		if action == "stopping" {
			return &roleBindingError{diagnostic: diagnostic, cause: errors.Join(ctx.Err(), lastErr)}
		}
		if err := wait(ctx, backoff.Pause()); err != nil {
			return &roleBindingError{diagnostic: diagnostic, cause: errors.Join(err, lastErr)}
		}
	}
	return &roleBindingError{diagnostic: diagnostic, cause: lastErr}
}

func waitRoleBinding(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
