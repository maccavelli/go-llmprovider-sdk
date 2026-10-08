package llmprovider

import (
	"errors"
	"testing"
)

// TestClassifyStreamFailure_ByType (0028-MADR D-H8): a failure inside a 200
// with no code is classified by its OpenAI- or Anthropic-style type, and an
// invalid request is checked for overflow, before the 500 stand-in.
func TestClassifyStreamFailure_ByType(t *testing.T) {
	overflow := "This model's maximum context length is 8192 tokens. However, you requested 9000 tokens (8000 in the messages, 1000 in the completion)."
	alone := "This model's maximum context length is 8192 tokens. However, you requested 8202 tokens (10 in the messages, 8192 in the completion)."
	for _, c := range []struct {
		name, errType, message string
		kind                   error
		retryable              bool
	}{
		{"invalid request", "invalid_request_error", "'messages' must contain at least one message", ErrInvalidRequest, false},
		{"invalid request, overflow", "invalid_request_error", overflow, ErrContextOverflow, false},
		{"invalid request, completion alone", "invalid_request_error", alone, ErrInvalidRequest, false},
		{"no type, overflow", "", overflow, ErrContextOverflow, false},
		{"authentication", "authentication_error", "invalid x-api-key", ErrAuthFailure, false},
		{"permission", "permission_error", "not allowed", ErrNotPermitted, false},
		{"rate limit", "rate_limit_error", "slow down", ErrRateLimited, true},
		{"overloaded", "overloaded_error", "Overloaded", ErrProviderUnavailable, true},
		{"api error", "api_error", "Internal server error", ErrProviderUnavailable, true},
		{"server error", "server_error", "The server had an error", ErrProviderUnavailable, true},
		{"unknown type", "unknown_error", "something", ErrProviderUnavailable, true},
	} {
		err := ClassifyStreamFailure("together", "", c.errType, c.message)
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok {
			t.Fatalf("%s: %v; want an *APIError", c.name, err)
		}
		if apiErr.Kind != c.kind || apiErr.Retryable() != c.retryable { //nolint:errorlint // the kind itself
			t.Errorf("%s: kind %v, retryable %t; want %v, %t", c.name, apiErr.Kind, apiErr.Retryable(), c.kind, c.retryable)
		}
	}
}
