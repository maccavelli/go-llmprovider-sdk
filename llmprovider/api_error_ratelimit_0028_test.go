package llmprovider

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// TestClassifyHTTPError_429ReadsResetHeader (0028-MADR D-H5): a 429 with no
// other delay takes the reset header's; Retry-After and a body's RetryInfo
// still win, the header coming last; a status other than 429 does not read
// it.
func TestClassifyHTTPError_429ReadsResetHeader(t *testing.T) {
	reset := http.Header{"X-Ratelimit-Reset-Requests": {"20s"}}
	retryAfterOf := func(err error) time.Duration {
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok {
			t.Fatalf("%v; want an *APIError", err)
		}
		return apiErr.RetryAfter
	}
	if got := retryAfterOf(classifyFixture("openai", 429, `{"error":{"type":"rate_limit_exceeded"}}`, reset)); got != 20*time.Second {
		t.Errorf("429 with the reset header: RetryAfter = %s; want 20s", got)
	}
	both := http.Header{"X-Ratelimit-Reset-Requests": {"20s"}, "Retry-After": {"7"}}
	if got := retryAfterOf(classifyFixture("openai", 429, `{}`, both)); got != 7*time.Second {
		t.Errorf("429 with Retry-After 7 and the reset header: RetryAfter = %s; want Retry-After's 7s", got)
	}
	gemini := `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"39s"}]}}`
	if got := retryAfterOf(classifyFixture("gemini", 429, gemini, reset)); got != 39*time.Second {
		t.Errorf("429 with RetryInfo 39s and the reset header: RetryAfter = %s; want the body's 39s", got)
	}
	if got := retryAfterOf(classifyFixture("openai", 400, `{}`, reset)); got != 0 {
		t.Errorf("400 with the reset header: RetryAfter = %s; want 0", got)
	}
	claude := http.Header{"Anthropic-Ratelimit-Requests-Reset": {time.Now().Add(30 * time.Second).Format(time.RFC3339)}}
	if got := retryAfterOf(classifyFixture("claude", 429, `{}`, claude)); got < 28*time.Second || got > 31*time.Second {
		t.Errorf("Claude's 429 with its reset 30 s ahead: RetryAfter = %s; want about 30s", got)
	}
}
