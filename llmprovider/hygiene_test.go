package llmprovider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// TestResolveOptions_NilHTTPClientIsTheDefault (0020-MADR F13): nil means the
// provider's default client, as Settings.HTTPClient says, not a nil one that
// panics on the first request.
func TestResolveOptions_NilHTTPClientIsTheDefault(t *testing.T) {
	st, err := ResolveOptions(ProviderClaude, []Option{WithHTTPClient(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if st.HTTPClient() == nil {
		t.Error("WithHTTPClient(nil) left no client; want the default")
	}
}

// TestResolveOptions_RefusesANonPositiveMaxTokens (0020-MADR F29, R23).
func TestResolveOptions_RefusesANonPositiveMaxTokens(t *testing.T) {
	for _, n := range []int{0, -1} {
		if _, err := ResolveOptions(ProviderClaude, []Option{WithMaxTokens(n)}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("WithMaxTokens(%d) = %v, want ErrInvalidRequest", n, err)
		}
	}
}

// TestResolveOptions_TrimsTheBaseURL (0020-MADR F30): a trailing slash is
// trimmed once, so no provider or listing requests "//models".
func TestResolveOptions_TrimsTheBaseURL(t *testing.T) {
	st, err := ResolveOptions(ProviderOllama, []Option{WithBaseURL("http://127.0.0.1:11434/")})
	if err != nil {
		t.Fatal(err)
	}
	if got := st.BaseURL(); got != "http://127.0.0.1:11434" {
		t.Errorf("BaseURL = %q, want it without the trailing slash", got)
	}
}

// TestCheck_RefusesPointerAndNilItems (0020-MADR F22): every encoder drops an
// item that is not one of the four value types, so Check refuses it.
func TestCheck_RefusesPointerAndNilItems(t *testing.T) {
	for name, item := range map[string]Item{
		"a *MessageItem":      &MessageItem{Role: RoleUser, Text: "hi"},
		"a *FunctionCallItem": &FunctionCallItem{CallID: "c", Name: "f"},
		"nil":                 nil,
	} {
		if err := (Capabilities{}).Check(&Request{Input: []Item{item}}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: Check = %v, want ErrInvalidRequest", name, err)
		}
	}
}

// TestClassifyHTTPError_NilResponse (0020-MADR F33).
func TestClassifyHTTPError_NilResponse(t *testing.T) {
	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ClassifyHTTPError(p, nil) panicked: %v", v)
		}
	}()
	if err := ClassifyHTTPError("p", nil); !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("ClassifyHTTPError(p, nil) = %v, want ErrProviderUnavailable", err)
	}
}

// geminiInvalidKeyBody is Gemini's answer to an invalid x-goog-api-key, HTTP
// 400, captured live on 2026-10-03 (0020-MADR F23, Q6 a).
const geminiInvalidKeyBody = `{
  "error": {
    "code": 400,
    "message": "API key not valid. Please pass a valid API key.",
    "status": "INVALID_ARGUMENT",
    "details": [
      {
        "@type": "type.googleapis.com/google.rpc.ErrorInfo",
        "reason": "API_KEY_INVALID",
        "domain": "googleapis.com",
        "metadata": {
          "service": "generativelanguage.googleapis.com"
        }
      },
      {
        "@type": "type.googleapis.com/google.rpc.LocalizedMessage",
        "locale": "en-US",
        "message": "API key not valid. Please pass a valid API key."
      }
    ]
  }
}
`

// anthropicInvalidKeyBody is Anthropic's answer to an invalid x-api-key, HTTP
// 401, captured live on 2026-10-03 (0020-MADR F23, Q6 a).
const anthropicInvalidKeyBody = `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},` +
	`"request_id":"req_011CfgSYDoyQNARy39ws7ip8"}`

// TestClassifyHTTPError_CapturedInvalidKeys (0020-MADR F23): an invalid key
// is a terminal ErrAuthFailure on the captured bodies. Gemini says so only in
// its details; Anthropic says so with 401.
func TestClassifyHTTPError_CapturedInvalidKeys(t *testing.T) {
	for _, c := range []struct {
		provider ProviderID
		status   int
		body     string
	}{
		{ProviderGemini, http.StatusBadRequest, geminiInvalidKeyBody},
		{ProviderClaude, http.StatusUnauthorized, anthropicInvalidKeyBody},
	} {
		err := ClassifyHTTPError(string(c.provider), &http.Response{StatusCode: c.status, Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(c.body))})
		apiErr, ok := errors.AsType[*APIError](err)
		if !errors.Is(err, ErrAuthFailure) || !ok || apiErr.Retryable() {
			t.Errorf("%s: %v, want a terminal ErrAuthFailure", c.provider, err)
		}
	}
}

// listingStub is a provider that lists models.
type listingStub struct{ stubProvider }

func (listingStub) ListModels(context.Context) ([]string, error) { return []string{"m"}, nil }

// TestWithRetry_KeepsListingAndDropsNativeStreaming (0020-MADR F32): the
// wrapper lists when its provider lists, and has no Stream of its own.
func TestWithRetry_KeepsListingAndDropsNativeStreaming(t *testing.T) {
	inner := &listingStub{stubProvider{id: "stub", caps: Capabilities{Tools: Supported, NativeStreaming: Supported}}}
	wrapped := WithRetry(inner, RetryPolicy{})
	lister, ok := wrapped.(ModelLister)
	if !ok {
		t.Fatal("WithRetry hid ModelLister")
	}
	if models, err := lister.ListModels(context.Background()); err != nil || !slices.Equal(models, []string{"m"}) {
		t.Errorf("ListModels = %v, %v; want the inner listing", models, err)
	}
	if caps := wrapped.Capabilities(); caps.NativeStreaming != Unsupported || caps.Tools != Supported {
		t.Errorf("Capabilities = %+v, want the inner's without NativeStreaming", caps)
	}
	if _, ok := WithRetry(&stubProvider{id: "stub"}, RetryPolicy{}).(ModelLister); ok {
		t.Error("a provider that does not list became a ModelLister")
	}
}
