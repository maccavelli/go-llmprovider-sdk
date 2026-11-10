package llmprovider

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// The error objects in this file are synthetic, from the shapes 0072-MADR
// cites: Codex's failed response (X11 to X14) and Google's 429 (0026-MADR
// F12).

// TestClassifyStreamFailureBody_Headers_0072 (0072-MADR D3, X11): the error
// object's headers member is read as HTTP headers: a delay gives RetryAfter,
// an HTTP-date ResetAt too.
func TestClassifyStreamFailureBody_Headers_0072(t *testing.T) {
	e := apiErrorOf(t, ClassifyStreamFailureBody("openai", "rate_limit_exceeded", "", "Slow down",
		[]byte(`{"code":"rate_limit_exceeded","message":"Slow down","headers":{"retry-after":"7"}}`)))
	if e.RetryAfter != 7*time.Second || !e.ResetAt.IsZero() {
		t.Errorf("a delay in headers: RetryAfter %s, ResetAt %v; want 7s and none", e.RetryAfter, e.ResetAt)
	}
	date := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	e = apiErrorOf(t, ClassifyStreamFailureBody("openai", "rate_limit_exceeded", "", "Slow down",
		[]byte(`{"code":"rate_limit_exceeded","headers":{"Retry-After":"`+date.Format(http.TimeFormat)+`"}}`)))
	if !e.ResetAt.Equal(date) || e.RetryAfter <= 59*time.Minute {
		t.Errorf("an HTTP-date in headers: RetryAfter %s, ResetAt %v; want about 1h and %v", e.RetryAfter, e.ResetAt, date)
	}
}

// TestClassifyStreamFailureBody_Text_0072 (0072-MADR D3, X12, X13): a rate
// limit's message gives its stated delay, as Codex reads it, after any
// structured delay; a quota or another kind takes none from its text.
func TestClassifyStreamFailureBody_Text_0072(t *testing.T) {
	for message, want := range map[string]time.Duration{
		"Rate limit reached. Please try again in 11.054s.": 11054 * time.Millisecond,
		"try again in 250ms":     250 * time.Millisecond,
		"Try again in 3 seconds": 3 * time.Second,
		"Try again in 5 minutes": 0,
	} {
		if got := apiErrorOf(t, ClassifyStreamFailure("openai", "rate_limit_exceeded", "", message)).RetryAfter; got != want {
			t.Errorf("%q: RetryAfter %s; want %s", message, got, want)
		}
	}
	const text = "Please try again in 20s."
	e := apiErrorOf(t, ClassifyStreamFailureBody("openai", "rate_limit_exceeded", "", text,
		[]byte(`{"headers":{"retry-after":"7"}}`)))
	if e.RetryAfter != 7*time.Second {
		t.Errorf("headers and text: RetryAfter %s; want the headers' 7s", e.RetryAfter)
	}
	for _, code := range []string{"usage_limit_reached", "server_error"} {
		if got := apiErrorOf(t, ClassifyStreamFailure("openai", code, "", text)).RetryAfter; got != 0 {
			t.Errorf("%s: RetryAfter %s from its text; want none", code, got)
		}
	}
}

// TestClassifyStreamFailureBody_UsageLimit_0072 (0072-MADR D3, X14): a
// usage limit's resets_at, plan_type and limit_window_minutes inside a
// stream are read as in an HTTP body.
func TestClassifyStreamFailureBody_UsageLimit_0072(t *testing.T) {
	reset := time.Now().Add(2 * time.Hour).Unix()
	raw := `{"type":"usage_limit_reached","code":"usage_limit_reached","message":"The usage limit has been reached",` +
		`"plan_type":"plus","resets_at":` + strconv.FormatInt(reset, 10) + `,"limit_window_minutes":300}`
	err := ClassifyStreamFailureBody("openai", "usage_limit_reached", "usage_limit_reached", "The usage limit has been reached", []byte(raw))
	e := apiErrorOf(t, err)
	if !errors.Is(err, ErrQuotaExhausted) || !e.ResetAt.Equal(time.Unix(reset, 0)) || e.Limit == nil ||
		e.Limit.Plan != "plus" || e.Limit.Window != 5*time.Hour {
		t.Errorf("%v: ResetAt %v, Limit %+v; want ErrQuotaExhausted, resets_at, Plan plus, Window 5h", err, e.ResetAt, e.Limit)
	}
}

// TestClassifyStreamFailureBody_Gemini_0072 (0072-MADR D3; 0026-MADR F12):
// Google's per-day quota inside a stream is ErrQuotaExhausted, with
// RetryInfo's delay and Limit.PerDay, as in an HTTP reply.
func TestClassifyStreamFailureBody_Gemini_0072(t *testing.T) {
	raw := `{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaId":"GenerateRequestsPerDayPerProjectPerModel-FreeTier","quotaValue":"5"}]},` +
		`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"52s"}]}`
	err := ClassifyStreamFailureBody("gemini", "429", "RESOURCE_EXHAUSTED", "You exceeded your current quota.", []byte(raw))
	e := apiErrorOf(t, err)
	if !errors.Is(err, ErrQuotaExhausted) || e.RetryAfter != 52*time.Second || e.Limit == nil || !e.Limit.PerDay {
		t.Errorf("%v: RetryAfter %s, Limit %+v; want ErrQuotaExhausted, 52s, PerDay", err, e.RetryAfter, e.Limit)
	}
}

// TestClassifyStreamFailure_Same_0072 (0072-MADR D3): with no error object
// and a message stating no delay, ClassifyStreamFailure is
// ClassifyStreamFailureBody with nil, as it was before.
func TestClassifyStreamFailure_Same_0072(t *testing.T) {
	cases := []struct{ provider, code, errType, message string }{
		{"openai", "rate_limit_exceeded", "", "slow down"},
		{"openai", "server_error", "", "boom"},
		{"openai", "context_length_exceeded", "", "too long"},
		{"openai", "insufficient_quota", "", "quota"},
		{"openai", "usage_limit_reached", "", "The usage limit has been reached"},
		{"openrouter", "429", "", "Rate limited"},
		{"gemini", "429", "RESOURCE_EXHAUSTED", "You exceeded your current quota."},
		{"claude", "", "overloaded_error", "Overloaded"},
		{"claude", "", "invalid_request_error", "prompt is too long: 300000 tokens > 200000 maximum"},
		{"opencode-zen/messages", "", "GoUsageLimitError", "limit"},
		{"together", "subscription_sharing_user_not_eligible", "", "refused"},
	}
	for _, c := range cases {
		a := apiErrorOf(t, ClassifyStreamFailure(c.provider, c.code, c.errType, c.message))
		b := apiErrorOf(t, ClassifyStreamFailureBody(c.provider, c.code, c.errType, c.message, nil))
		sameKind := errors.Is(a.kind(), b.kind()) && errors.Is(b.kind(), a.kind())
		if a.Error() != b.Error() || !sameKind || a.Retryable() != b.Retryable() || a.RetryAfter != b.RetryAfter ||
			a.Code != b.Code || a.Limit != nil || b.Limit != nil || !a.ResetAt.IsZero() || !b.ResetAt.IsZero() {
			t.Errorf("%+v: ClassifyStreamFailure %v (retryable %t), Body(nil) %v (retryable %t); want the same", c, a, a.Retryable(), b, b.Retryable())
		}
	}
}
