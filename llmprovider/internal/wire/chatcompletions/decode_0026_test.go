package chatcompletions

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecodeFor_ErrorIn200Classified (0026-MADR F5): an OpenRouter-style
// error inside a 200 carries the HTTP status in error.code. It is classified
// as that status, through the service's own table, and names the provider
// the caller gave, not the wire.
func TestDecodeFor_ErrorIn200Classified(t *testing.T) {
	for _, c := range []struct {
		name, provider, body string
		want                 error
		retryable            bool
	}{
		{"400 context overflow", "kilo", `{"error":{"code":400,"message":"This endpoint's maximum context length is 8192 tokens. However, you requested about 9000 tokens."}}`, llmprovider.ErrContextOverflow, false},
		{"401", "kilo", `{"error":{"code":401,"message":"No auth credentials found"}}`, llmprovider.ErrAuthFailure, false},
		{"402", "together", `{"error":{"code":402,"message":"Insufficient credits"}}`, llmprovider.ErrQuotaExhausted, false},
		{"403 on kilo", "kilo", `{"error":{"code":403,"message":"flagged by moderation"}}`, llmprovider.ErrNotPermitted, false},
		{"429", "huggingface", `{"error":{"code":429,"message":"rate limited"}}`, llmprovider.ErrRateLimited, true},
		{"503 in a choice", "kilo", `{"choices":[{"finish_reason":"error","error":{"code":503,"message":"upstream down"}}]}`, llmprovider.ErrProviderUnavailable, true},
		{"OpenCode's own type", "opencode-zen/chat", `{"error":{"type":"FreeUsageLimitError","message":"free usage limit reached"}}`, llmprovider.ErrQuotaExhausted, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := DecodeFor(c.provider)(strings.NewReader(c.body))
			var apiErr *llmprovider.APIError
			if !errors.As(err, &apiErr) || !errors.Is(err, c.want) {
				t.Fatalf("DecodeFor(%q) = %v; want an *APIError of kind %v", c.provider, err, c.want)
			}
			if apiErr.Provider != c.provider {
				t.Errorf("Provider = %q, want %q", apiErr.Provider, c.provider)
			}
			if apiErr.Retryable() != c.retryable {
				t.Errorf("Retryable() = %v, want %v", apiErr.Retryable(), c.retryable)
			}
		})
	}
}
