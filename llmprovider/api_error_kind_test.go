package llmprovider

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAPIError_KindAndCodeFromClassification(t *testing.T) {
	err := classifyBody(ProviderOpencodeZen, http.StatusForbidden, `{"type":"error","error":{"type":"RegionError","message":"not here"}}`)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("%v is not an APIError", err)
	}
	if !errors.Is(apiErr.Kind, ErrNotPermitted) || apiErr.Code != "RegionError" || apiErr.Retryable() {
		t.Fatalf("%+v; want kind ErrNotPermitted, code RegionError, not retryable", apiErr)
	}
}

func TestAPIError_Retryable(t *testing.T) {
	for _, c := range []struct {
		name string
		err  *APIError
		want bool
	}{
		{"a rate limit", &APIError{Status: 429, Kind: ErrRateLimited}, true},
		{"exhausted quota", &APIError{Status: 429, Kind: ErrQuotaExhausted}, false},
		{"an unavailable service", &APIError{Status: 503, Kind: ErrProviderUnavailable}, true},
		{"a final unavailable service", &APIError{Status: 525, Kind: ErrProviderUnavailable, terminal: true}, false},
		{"an invalid request", &APIError{Status: 400, Kind: ErrInvalidRequest}, false},
		{"an authentication failure", &APIError{Status: 401, Kind: ErrAuthFailure}, false},
		{"no kind, a 502", &APIError{Status: 502}, true},
		{"no kind, a 400", &APIError{Status: 400}, false},
		{"no kind, a stream failure", &APIError{}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.err.Retryable(); got != c.want {
				t.Fatalf("Retryable() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestAPIError_WithoutKind covers an APIError a caller builds: with no Kind,
// its status decides, and with no status it is a stream failure.
func TestAPIError_WithoutKind(t *testing.T) {
	err := &APIError{Provider: "p", Status: 401, Code: "bad_key"}
	if !errors.Is(err, ErrAuthFailure) || err.Error() != "llmprovider: authentication failed: p HTTP 401 bad_key" {
		t.Fatalf("%q", err.Error())
	}
	stream := &APIError{Provider: "p", Code: "old"}
	if !errors.Is(stream, ErrProviderUnavailable) || stream.Error() != "llmprovider: provider unavailable: p stream old" {
		t.Fatalf("%q", stream.Error())
	}
}

// TestAPIError_ErrorText pins the message of each shape D7 folds into
// APIError (0015-MADR D7): a rate limit names its provider and retry-after,
// as RateLimitError did, and a cut-short response its reason, as
// IncompleteError did.
func TestAPIError_ErrorText(t *testing.T) {
	for _, c := range []struct {
		err  *APIError
		want string
	}{
		{&APIError{Provider: ProviderKilo, Status: 429, Kind: ErrRateLimited, RetryAfter: 5 * time.Second, Message: "slow down"},
			"llmprovider: rate limited: kilo HTTP 429 (retry-after 5s): slow down"},
		{&APIError{Status: 429, Kind: ErrRateLimited, RetryAfter: 5 * time.Second},
			"llmprovider: rate limited: HTTP 429 (retry-after 5s)"},
		{&APIError{Provider: ProviderOpenAI, Kind: ErrRateLimited, Code: "slow_down", Message: "wait"},
			"llmprovider: rate limited: openai stream slow_down: wait"},
		{&APIError{Kind: ErrIncomplete, Reason: "length"},
			"llmprovider: incomplete response: length"},
		{&APIError{Provider: ProviderGemini, Kind: ErrIncomplete, Reason: "incomplete"},
			"llmprovider: incomplete response: gemini: incomplete"},
	} {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
	}
}

func TestSentinels_Beneath(t *testing.T) {
	if !errors.Is(ErrContextOverflow, ErrInvalidRequest) {
		t.Error("ErrContextOverflow must match ErrInvalidRequest")
	}
	if !errors.Is(ErrIncomplete, ErrInvalidRequest) {
		t.Error("ErrIncomplete must match ErrInvalidRequest")
	}
	if !errors.Is(ErrUnsupported, errors.ErrUnsupported) {
		t.Error("ErrUnsupported must match errors.ErrUnsupported")
	}
	incomplete := &APIError{Kind: ErrIncomplete, Reason: "length"}
	if !errors.Is(incomplete, ErrIncomplete) || !errors.Is(incomplete, ErrInvalidRequest) || incomplete.Retryable() {
		t.Error("a cut-short APIError must match ErrIncomplete and ErrInvalidRequest, and not be retryable")
	}
	limited := &APIError{Kind: ErrRateLimited}
	if !errors.Is(limited, ErrRateLimited) || errors.Is(limited, ErrInvalidRequest) || !limited.Retryable() {
		t.Error("a stream rate limit must match ErrRateLimited alone, and be retryable")
	}
	for _, err := range []error{
		ErrRateLimited, ErrProviderUnavailable, ErrAuthFailure, ErrInvalidRequest, ErrQuotaExhausted,
		ErrNotPermitted, ErrContextOverflow, ErrIncomplete, ErrUnsupported, ErrInvalidProvider,
	} {
		if msg := err.Error(); len(msg) < len("llmprovider: ") || msg[:len("llmprovider: ")] != "llmprovider: " {
			t.Errorf("%q does not start llmprovider: (R27)", msg)
		}
	}
}
