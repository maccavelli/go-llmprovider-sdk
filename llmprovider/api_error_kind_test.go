package llmprovider

import (
	"errors"
	"net/http"
	"testing"
)

func TestAPIError_KindAndCodeFromClassification(t *testing.T) {
	err := classifyBody(ProviderOpencodeZen, http.StatusForbidden, `{"type":"error","error":{"type":"RegionError","message":"not here"}}`)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("%v is not an APIError", err)
	}
	if !errors.Is(apiErr.Kind, ErrNotPermitted) || apiErr.Code != "RegionError" || apiErr.Type != apiErr.Code || apiErr.Retryable() {
		t.Fatalf("%+v; want kind ErrNotPermitted, code RegionError in Code and Type, not retryable", apiErr)
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
		{"a final unavailable service", &APIError{Status: 525, Kind: ErrProviderUnavailable, Terminal: true}, false},
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
// its status decides, and Code wins over the deprecated Type.
func TestAPIError_WithoutKind(t *testing.T) {
	err := &APIError{Provider: "p", Status: 401, Code: "bad_key", Type: "old"}
	if !errors.Is(err, ErrAuthFailure) || err.Error() != "llm: authentication failed: p HTTP 401 bad_key" {
		t.Fatalf("%q", err.Error())
	}
	old := &APIError{Provider: "p", Status: 0, Type: "old"}
	if !errors.Is(old, ErrProviderUnavailable) || old.Error() != "llm: provider unavailable: p stream old" {
		t.Fatalf("%q", old.Error())
	}
}

func TestSentinels_Beneath(t *testing.T) {
	if !errors.Is(ErrContextOverflow, ErrInvalidRequest) {
		t.Error("ErrContextOverflow must match ErrInvalidRequest")
	}
	if !errors.Is(ErrUnsupported, errors.ErrUnsupported) {
		t.Error("ErrUnsupported must match errors.ErrUnsupported")
	}
	incomplete := &IncompleteError{Reason: "length"}
	if !errors.Is(incomplete, ErrIncomplete) || !errors.Is(incomplete, ErrInvalidRequest) {
		t.Error("IncompleteError must match ErrIncomplete and ErrInvalidRequest")
	}
	for _, err := range []error{ErrContextOverflow, ErrIncomplete, ErrUnsupported} {
		if msg := err.Error(); len(msg) < len("llmprovider: ") || msg[:len("llmprovider: ")] != "llmprovider: " {
			t.Errorf("%q does not start llmprovider: (R27)", msg)
		}
	}
}
