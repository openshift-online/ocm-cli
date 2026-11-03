# WIF project IAM role binding retries

Project role bindings use a local retry policy, not the general WIF retry helper.
Both service-account and support-group bindings reread the policy on every
attempt, retain its etag, reapply the desired bindings, and write only when needed.
REST API errors remain available through `errors.As` and `errors.Is`.

## Classification

- Retry HTTP 500, 502, 503, and 504 unless structured reasons indicate a permanent
  failure or an unknown condition.
- Retry HTTP 403 or 429 only with the explicit `rateLimitExceeded`,
  `userRateLimitExceeded`, `servingLimitExceeded`, or structured
  `RATE_LIMIT_EXCEEDED` reason.
- Stop on quota, capacity, daily-limit, or registration failures. In particular,
  HTTP 429 alone does not establish temporary rate limiting. `QuotaFailure`
  details alone are non-retryable: they do not distinguish rate quotas from
  allocation quotas. An explicit rate-limit reason can establish throttling even
  with quota details; an explicit permanent reason takes precedence over throttling.
- Retry HTTP 409 only with the canonical `ABORTED` status, restarting the whole
  read-modify-write transaction rather than replaying a stale policy write.
- Stop on all other errors, including unknown errors, validation and
  authentication failures, and generic 403, 404, or 409 responses. IAM permits
  retrying some 404s for newly created resources, but this path reads an existing
  project policy: a 404 here does not establish service-account propagation.

The binding operation has a maximum of 12 attempts, jittered exponential backoff
capped at 32 seconds, and the existing WIF timeout (600 seconds, or 30 seconds
with `--fail-fast`). The caller's cancellation and deadline apply to both policy
requests and waits. An exhausted retry returns the last typed API cause;
cancellation and timeout also remain discoverable in the error chain.

## Diagnostics

Every failed attempt logs its number, stop/retry decision, HTTP code, allowlisted
reason, safe message evidence, and corrective guidance. Message evidence is
restricted to fixed recognized phrases; all other text is redacted. No arbitrary
reason, metadata, policy, principal, resource identifier, header, response body,
or error-wrapper text is copied into the new diagnostics. The final binding error
also formats only this safe summary while retaining its original cause via
`Unwrap`. Existing progress messages and diagnostics for unrelated operations are
unchanged. The shared retry helper is unchanged.

## Official API guidance

- [IAM retry strategy](https://cloud.google.com/iam/docs/retry-strategy): transient
  server errors, optional eventual-consistency 404 handling, and complete
  read-modify-write retries for 409 `ABORTED` concurrency failures.
- [Resource Manager core errors](https://cloud.google.com/resource-manager/docs/core_errors):
  rate-limit reasons versus quota, daily-limit, registration, and capacity errors.
- [Quota troubleshooting](https://cloud.google.com/docs/quotas/troubleshoot):
  HTTP 429 can represent exceeded quota; rate quotas differ from allocation quotas.
- [AIP-193](https://google.aip.dev/193): structured `ErrorInfo` reasons and
  error details; request-specific message and metadata fields are not safe to
  print indiscriminately.

Offline tests cover classification, single-attempt permanent failures with no
wait, retry then success, retry exhaustion and last-cause retention, cancellation,
wrapped SDK errors, structured quota/rate distinctions, diagnostic sanitization,
fresh policy etags after conflicts, and propagation of the caller's context to
the REST policy read. No cloud credentials or live cloud resources are required.
