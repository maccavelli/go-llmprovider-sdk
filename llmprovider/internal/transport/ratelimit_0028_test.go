package transport

import (
	"net/http"
	"testing"
	"time"
)

// TestRateLimitReset (0028-MADR D-H5): the reset headers the services were
// measured sending (0028-PLAN Phase 1, T2) give the wait: OpenAI's and Hugging
// Face's x-ratelimit-reset-* as Go durations, Claude's
// anthropic-ratelimit-*-reset as RFC 3339 times; of several, the longest.
// Anything else gives 0.
func TestRateLimitReset(t *testing.T) {
	now := time.Date(2026, 10, 8, 20, 0, 0, 0, time.UTC)
	at := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339) }
	for _, c := range []struct {
		name   string
		header map[string]string
		want   time.Duration
	}{
		{"requests 20s", map[string]string{"x-ratelimit-reset-requests": "20s"}, 20 * time.Second},
		{"requests 6m0s", map[string]string{"x-ratelimit-reset-requests": "6m0s"}, 6 * time.Minute},
		{"requests 250ms", map[string]string{"x-ratelimit-reset-requests": "250ms"}, 250 * time.Millisecond},
		{"tokens 1.5s", map[string]string{"x-ratelimit-reset-tokens": "1.5s"}, 1500 * time.Millisecond},
		{"both, the longer", map[string]string{"x-ratelimit-reset-requests": "12ms", "x-ratelimit-reset-tokens": "0s"}, 12 * time.Millisecond},
		{"both, tokens longer", map[string]string{"x-ratelimit-reset-requests": "1s", "x-ratelimit-reset-tokens": "6m0s"}, 6 * time.Minute},
		{"anthropic requests", map[string]string{"anthropic-ratelimit-requests-reset": at(30 * time.Second)}, 30 * time.Second},
		{"anthropic, the longest of four", map[string]string{
			"anthropic-ratelimit-requests-reset":      at(5 * time.Second),
			"anthropic-ratelimit-tokens-reset":        at(10 * time.Second),
			"anthropic-ratelimit-input-tokens-reset":  at(45 * time.Second),
			"anthropic-ratelimit-output-tokens-reset": at(20 * time.Second),
		}, 45 * time.Second},
		{"a past time", map[string]string{"anthropic-ratelimit-requests-reset": at(-time.Minute)}, 0},
		{"a negative duration", map[string]string{"x-ratelimit-reset-requests": "-5s"}, 0},
		{"malformed", map[string]string{"x-ratelimit-reset-requests": "soon", "anthropic-ratelimit-requests-reset": "tomorrow"}, 0},
		{"empty", map[string]string{"x-ratelimit-reset-requests": ""}, 0},
		{"no header", nil, 0},
		{"an unmeasured header", map[string]string{"x-ratelimit-reset": "20s"}, 0},
	} {
		h := http.Header{}
		for k, v := range c.header {
			h.Set(k, v)
		}
		if got := RateLimitReset(h, now); got != c.want {
			t.Errorf("%s: RateLimitReset = %s; want %s", c.name, got, c.want)
		}
	}
}
