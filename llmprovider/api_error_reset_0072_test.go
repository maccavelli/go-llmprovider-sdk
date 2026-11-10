package llmprovider

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// The fixtures in this file are synthetic, built from the shapes
// 0072-MADR cites: ChatGPT's usage limit (X3 to X5), OpenCode's metadata
// (O1, O4 to O6), Kilo's 402 (K1, K2), Claude's reset headers (C6).

// usageLimitBody is a ChatGPT 429 usage-limit body (0072-MADR X3 to X5).
func usageLimitBody(resetsAt int64) string {
	return `{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","plan_type":"plus","resets_at":` +
		strconv.FormatInt(resetsAt, 10) + `}}`
}

// apiErrorOf returns err's *APIError, failing the test without one.
func apiErrorOf(t *testing.T, err error) *APIError {
	t.Helper()
	apiErr, ok := errors.AsType[*APIError](err)
	if !ok {
		t.Fatalf("%v; want an *APIError", err)
	}
	return apiErr
}

// TestResetAt_ResetsAt_0072 (0072-MADR D1): a ChatGPT usage limit's
// resets_at is kept as its instant, in UTC; RetryAfter is as before.
func TestResetAt_ResetsAt_0072(t *testing.T) {
	reset := time.Now().Add(time.Hour).Unix()
	e := apiErrorOf(t, classifyFixture("openai", 429, usageLimitBody(reset), nil))
	if want := time.Unix(reset, 0); !e.ResetAt.Equal(want) {
		t.Errorf("ResetAt = %v; want resets_at's %v", e.ResetAt, want)
	}
	if e.ResetAt.Location() != time.UTC {
		t.Errorf("ResetAt in %v; want UTC", e.ResetAt.Location())
	}
	if e.RetryAfter <= 59*time.Minute || e.RetryAfter > time.Hour {
		t.Errorf("RetryAfter = %s; want just under an hour, as before", e.RetryAfter)
	}
}

// TestResetAt_PastKept_0072 (0072-MADR D1, Q5): a resets_at that has passed
// is kept as stated, and gives no RetryAfter, as before.
func TestResetAt_PastKept_0072(t *testing.T) {
	reset := time.Now().Add(-time.Hour).Unix()
	e := apiErrorOf(t, classifyFixture("openai", 429, usageLimitBody(reset), nil))
	if want := time.Unix(reset, 0); !e.ResetAt.Equal(want) {
		t.Errorf("ResetAt = %v; want the past resets_at %v, kept as stated", e.ResetAt, want)
	}
	if e.RetryAfter != 0 {
		t.Errorf("RetryAfter = %s; want 0 for a reset that has passed", e.RetryAfter)
	}
}

// TestResetAt_Order_0072 (0072-MADR D1): the body's instant outranks a
// header's; an HTTP-date alone names one; Claude's reset header names one
// on a 429 and is not read on another status.
func TestResetAt_Order_0072(t *testing.T) {
	reset := time.Now().Add(time.Hour).Truncate(time.Second)
	e := apiErrorOf(t, classifyFixture("openai", 429, usageLimitBody(reset.Unix()), http.Header{"Retry-After": {"30"}}))
	if !e.ResetAt.Equal(reset) || e.RetryAfter != 30*time.Second {
		t.Errorf("resets_at with Retry-After 30: ResetAt %v, RetryAfter %s; want the body's %v and the header's 30s", e.ResetAt, e.RetryAfter, reset)
	}
	date := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	e = apiErrorOf(t, classifyFixture("openai", 429, `{"error":{"type":"usage_limit_reached","resets_at":`+strconv.FormatInt(reset.Unix(), 10)+`}}`,
		http.Header{"Retry-After": {date.Format(http.TimeFormat)}}))
	if !e.ResetAt.Equal(reset) {
		t.Errorf("resets_at with an HTTP-date: ResetAt %v; want the body's %v", e.ResetAt, reset)
	}
	e = apiErrorOf(t, classifyFixture("openai", 429, `{}`, http.Header{"Retry-After": {date.Format(http.TimeFormat)}}))
	if !e.ResetAt.Equal(date) {
		t.Errorf("an HTTP-date alone: ResetAt %v; want %v", e.ResetAt, date)
	}
	claudeReset := time.Now().Add(40 * time.Second).UTC().Truncate(time.Second)
	claude := http.Header{"Anthropic-Ratelimit-Requests-Reset": {claudeReset.Format(time.RFC3339)}}
	e = apiErrorOf(t, classifyFixture("claude", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`, claude))
	if !e.ResetAt.Equal(claudeReset) {
		t.Errorf("Claude's 429 with its reset header: ResetAt %v; want %v", e.ResetAt, claudeReset)
	}
	e = apiErrorOf(t, classifyFixture("claude", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"bad"}}`, claude))
	if !e.ResetAt.IsZero() {
		t.Errorf("Claude's 400 with the reset header: ResetAt %v; want none", e.ResetAt)
	}
	e = apiErrorOf(t, classifyFixture("openai", 429, `{}`, http.Header{"X-Ratelimit-Reset-Requests": {"20s"}}))
	if !e.ResetAt.IsZero() || e.RetryAfter != 20*time.Second {
		t.Errorf("a duration reset header: ResetAt %v, RetryAfter %s; want none and 20s", e.ResetAt, e.RetryAfter)
	}
}

// TestResetAt_OpenCodeMetadata_0072 (0072-MADR D1, O4 to O6): OpenCode's
// metadata.resetsAt, beside error, inside it, or in a Google ErrorInfo,
// gives ResetAt, and RetryAfter when no header did; a malformed one gives
// neither.
func TestResetAt_OpenCodeMetadata_0072(t *testing.T) {
	reset := time.Now().Add(3 * time.Hour).UTC().Truncate(time.Second)
	stamp := reset.Format(time.RFC3339)
	cases := []struct{ name, body string }{
		{"beside error", `{"type":"error","error":{"type":"BudgetLimitError","message":"A budget policy for this model has been reached."},` +
			`"metadata":{"coverage":"models","resetsAt":"` + stamp + `"}}`},
		{"inside error", `{"error":{"type":"BudgetLimitError","message":"budget","metadata":{"resetsAt":"` + stamp + `"}}}`},
		{"in an ErrorInfo", `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"Policy budget exceeded","details":[` +
			`{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"BudgetLimitError","domain":"opencode.ai",` +
			`"metadata":{"coverage":"all","resetsAt":"` + stamp + `"}}]}}`},
	}
	for _, c := range cases {
		e := apiErrorOf(t, classifyFixture("opencode-zen/chat_completions", 429, c.body, nil))
		if !e.ResetAt.Equal(reset) {
			t.Errorf("%s: ResetAt %v; want %v", c.name, e.ResetAt, reset)
		}
		if e.RetryAfter < 3*time.Hour-time.Minute || e.RetryAfter > 3*time.Hour {
			t.Errorf("%s: RetryAfter %s; want about 3h, from resetsAt", c.name, e.RetryAfter)
		}
	}
	e := apiErrorOf(t, classifyFixture("opencode-zen/chat_completions", 429, cases[0].body, http.Header{"Retry-After": {"120"}}))
	if !e.ResetAt.Equal(reset) || e.RetryAfter != 2*time.Minute {
		t.Errorf("with Retry-After 120: ResetAt %v, RetryAfter %s; want %v and the header's 2m0s", e.ResetAt, e.RetryAfter, reset)
	}
	e = apiErrorOf(t, classifyFixture("opencode-zen/chat_completions", 429,
		`{"type":"error","error":{"type":"BudgetLimitError","message":"budget"},"metadata":{"resetsAt":"next month"}}`, nil))
	if !e.ResetAt.IsZero() || e.RetryAfter != 0 {
		t.Errorf("a malformed resetsAt: ResetAt %v, RetryAfter %s; want neither", e.ResetAt, e.RetryAfter)
	}
}

// TestAPIError_TextUnchanged_0072 (0072-MADR D1, D6): Error does not print
// ResetAt: each error's text is that of a copy without it, and one is pinned.
func TestAPIError_TextUnchanged_0072(t *testing.T) {
	reset := time.Now().Add(time.Hour).Unix()
	claude := http.Header{"Anthropic-Ratelimit-Requests-Reset": {time.Now().Add(time.Minute).UTC().Format(time.RFC3339)}}
	pinned := classifyFixture("openai", 429, usageLimitBody(reset), http.Header{"Retry-After": {"30"}})
	errs := map[string]error{
		"a ChatGPT usage limit": pinned,
		"a Gemini per-day 429":  classifyFixture("gemini", 429, geminiQuota429("GenerateRequestsPerDayPerProjectPerModel-FreeTier", "52s"), nil),
		"an OpenCode Go 429": classifyFixture("opencode-go/chat_completions", 429,
			`{"type":"error","error":{"type":"GoUsageLimitError","message":"Go usage limit exceeded"},"metadata":{"workspace":"wrk_example","limitName":"5 hour"}}`,
			http.Header{"Retry-After": {"120"}}),
		"a Kilo 402": classifyFixture("kilo", 402,
			`{"error":{"message":"Insufficient balance. Please add credits to continue.","code":402,"metadata":{"buyCreditsUrl":"https://app.kilo.ai/credits"}}}`, nil),
		"a Claude 429":     classifyFixture("claude", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`, claude),
		"a stream failure": ClassifyStreamFailure("openai", "rate_limit_exceeded", "", "Slow down"),
	}
	for name, err := range errs {
		e := apiErrorOf(t, err)
		bare := *e
		bare.ResetAt = time.Time{}
		if got, want := e.Error(), bare.Error(); got != want {
			t.Errorf("%s: Error() = %q; want %q, as without ResetAt", name, got, want)
		}
	}
	const want = "llmprovider: quota exhausted: openai HTTP 429 usage_limit_reached (retry-after 30s): The usage limit has been reached"
	if got := pinned.Error(); got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}
