package llmprovider

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// geminiQuota429 is a Gemini 429 in Google's documented shape: a QuotaFailure
// naming the violated quota, and a RetryInfo with the delay
// (google/rpc/error_details.proto). It is not a capture (0026-MADR, amendment
// "Q6 is (b), F12 from Google's documented shape").
func geminiQuota429(quotaID, retryDelay string) string {
	return `{"error":{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{` +
		`"quotaMetric":"generativelanguage.googleapis.com/generate_content_free_tier_requests",` +
		`"quotaId":"` + quotaID + `","quotaDimensions":{"location":"global","model":"gemini-pro-latest"},"quotaValue":"5"}]},` +
		`{"@type":"type.googleapis.com/google.rpc.Help","links":[{"description":"Learn more","url":"https://ai.google.dev/gemini-api/docs/rate-limits"}]},` +
		`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"` + retryDelay + `"}]}}`
}

// TestClassify_GeminiRetryInfo (0026-MADR F12): Gemini sends no Retry-After
// header; its 429 body's RetryInfo gives RetryAfter, and a per-day quota is
// ErrQuotaExhausted, which no retry within the day can pass.
func TestClassify_GeminiRetryInfo(t *testing.T) {
	for _, c := range []struct {
		name, quotaID, delay string
		want                 time.Duration
		kind                 error
		retryable            bool
	}{
		{"per minute", "GenerateRequestsPerMinutePerProjectPerModel-FreeTier", "39s", 39 * time.Second, ErrRateLimited, true},
		{"fractional", "GenerateContentPaidTierInputTokensPerModelPerMinute", "1.5s", 1500 * time.Millisecond, ErrRateLimited, true},
		{"per day", "GenerateRequestsPerDayPerProjectPerModel-FreeTier", "52s", 52 * time.Second, ErrQuotaExhausted, false},
	} {
		err := classifyFixture("gemini", 429, geminiQuota429(c.quotaID, c.delay), nil)
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok {
			t.Fatalf("%s: %v; want an *APIError", c.name, err)
		}
		if apiErr.RetryAfter != c.want || !errors.Is(err, c.kind) || apiErr.Retryable() != c.retryable {
			t.Errorf("%s: RetryAfter=%s kind ok=%t retryable=%t (%v); want %s, %v, retryable=%t",
				c.name, apiErr.RetryAfter, errors.Is(err, c.kind), apiErr.Retryable(), err, c.want, c.kind, c.retryable)
		}
	}
}

// TestClassify_GeminiRetryInfoEdges: a header wins over the body; a 429 with
// neither detail, as Google's forum reports, keeps today's classification;
// a malformed or non-positive delay is ignored; another service's body is not
// read for Gemini's quota ids.
func TestClassify_GeminiRetryInfoEdges(t *testing.T) {
	minute := "GenerateRequestsPerMinutePerProjectPerModel-FreeTier"
	header := map[string][]string{"Retry-After": {"7"}}
	if e, _ := errors.AsType[*APIError](classifyFixture("gemini", 429, geminiQuota429(minute, "39s"), header)); e == nil || e.RetryAfter != 7*time.Second {
		t.Errorf("Retry-After 7 with retryDelay 39s: %v; want the header's 7s", e)
	}
	bare := `{"error":{"code":429,"message":"Resource exhausted","status":"RESOURCE_EXHAUSTED"}}`
	if e, _ := errors.AsType[*APIError](classifyFixture("gemini", 429, bare, nil)); e == nil || e.RetryAfter != 0 || !e.Retryable() {
		t.Errorf("a bare 429: %v; want a retryable rate limit with no delay", e)
	}
	for _, delay := range []string{"soon", "-5s", "0s", "39", "NaNs"} {
		if e, _ := errors.AsType[*APIError](classifyFixture("gemini", 429, geminiQuota429(minute, delay), nil)); e == nil || e.RetryAfter != 0 {
			t.Errorf("retryDelay %q: %v; want it ignored", delay, e)
		}
	}
	day := geminiQuota429("GenerateRequestsPerDayPerProjectPerModel-FreeTier", "52s")
	if err := classifyFixture("kilo", 429, day, nil); errors.Is(err, ErrQuotaExhausted) || !strings.Contains(err.Error(), "kilo") {
		t.Errorf("kilo with a Google per-day body: %v; want its own rate limit", err)
	}
}
