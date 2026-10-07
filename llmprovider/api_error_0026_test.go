package llmprovider

import (
	"errors"
	"net/http"
	"testing"
)

// TestAPIError_RetryableDoesNotMatchInvalidRequest (0026-MADR F23):
// ErrInvalidRequest says a retry "can never succeed", so a Retryable error
// leaves out its status's pre-0012 sentinel: a retryable 408, or a 409 on
// openai and claude, no longer matches ErrInvalidRequest (0012-MADR amendment
// 2026-10-07). A terminal 4xx still matches it.
func TestAPIError_RetryableDoesNotMatchInvalidRequest(t *testing.T) {
	const body = `{"error":{"message":"try again"}}`
	for _, c := range []struct {
		provider string
		status   int
	}{
		{"openai", http.StatusRequestTimeout},
		{"openai", http.StatusConflict},
		{"claude", http.StatusConflict},
	} {
		err := classifyFixture(c.provider, c.status, body, nil)
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok || !apiErr.Retryable() {
			t.Fatalf("%s %d: %v; want a retryable APIError", c.provider, c.status, err)
		}
		if errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s %d: Retryable=true and matches ErrInvalidRequest; want only its retryable kind", c.provider, c.status)
		}
	}
	if err := classifyFixture("openai", http.StatusBadRequest, body, nil); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("openai 400: %v; want ErrInvalidRequest", err)
	}
}

// TestAPIError_ShouldRetryIsNotInvalidRequest (0026-MADR F67): a reply the
// service marks x-should-retry: true is retried, so a status that would be
// ErrInvalidRequest is ErrProviderUnavailable instead, as a retryable 408 is.
// Any other kind keeps its kind.
func TestAPIError_ShouldRetryIsNotInvalidRequest(t *testing.T) {
	const body = `{"error":{"message":"m"}}`
	shouldRetry := func() http.Header { return http.Header{"X-Should-Retry": {"true"}} }
	for _, provider := range []string{"openai", "claude", "kilo"} {
		err := classifyFixture(provider, http.StatusBadRequest, body, shouldRetry())
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok || !apiErr.Retryable() || !errors.Is(err, ErrProviderUnavailable) || errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s 400, x-should-retry true: %v; want retryable ErrProviderUnavailable, not ErrInvalidRequest", provider, err)
		}
	}
	if err := classifyFixture("openai", http.StatusUnauthorized, body, shouldRetry()); !errors.Is(err, ErrAuthFailure) {
		t.Errorf("openai 401, x-should-retry true: %v; want ErrAuthFailure kept", err)
	}
	if err := classifyFixture("openai", http.StatusBadRequest, body, nil); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("openai 400: %v; want ErrInvalidRequest", err)
	}
}
