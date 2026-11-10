package llmprovider

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// The fixtures in this file are synthetic, built from the shapes 0072-MADR
// cites: ChatGPT's usage limit (X3 to X5), OpenCode Go's metadata (O1, O2),
// Gemini's QuotaFailure (0026-MADR F12), Kilo's 402 (K1, K2).

// chatGPTLimitBody is a ChatGPT 429 usage-limit body with extra members,
// such as `"limit_window_minutes":300` (0072-MADR X3 to X5).
func chatGPTLimitBody(resetsAt int64, extra string) string {
	return `{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","plan_type":"plus","resets_at":` +
		strconv.FormatInt(resetsAt, 10) + extra + `}}`
}

// goLimitBody is an OpenCode Go 429 naming its limit (0072-MADR O1).
func goLimitBody(limitName string) string {
	return `{"type":"error","error":{"type":"GoUsageLimitError","message":"Go usage limit exceeded"},` +
		`"metadata":{"workspace":"wrk_example","limitName":"` + limitName + `"}}`
}

// kiloCreditsBody is Kilo's documented 402 with a link (0072-MADR K1, K2).
func kiloCreditsBody(link string) string {
	return `{"error":{"message":"Insufficient balance. Please add credits to continue.","code":402,"metadata":{"buyCreditsUrl":"` + link + `"}}}`
}

// limitOfErr returns err's Limit, failing the test when err has none.
func limitOfErr(t *testing.T, err error) *Limit {
	t.Helper()
	l := apiErrorOf(t, err).Limit
	if l == nil {
		t.Fatalf("%v: Limit is nil; want one", err)
	}
	return l
}

// TestLimit_ChatGPT_0072 (0072-MADR D2, X3, X5): plan_type and
// limit_window_minutes are kept as Plan and Window.
func TestLimit_ChatGPT_0072(t *testing.T) {
	reset := time.Now().Add(time.Hour).Unix()
	l := limitOfErr(t, classifyFixture("openai", 429, chatGPTLimitBody(reset, `,"limit_window_minutes":300`), nil))
	if l.Plan != "plus" || l.Window != 5*time.Hour {
		t.Errorf("Limit = %+v; want Plan plus and Window 5h", *l)
	}
}

// TestLimit_NeverInferred_0072 (0072-MADR D2; M1): with no stated length, a
// long wait or a word such as "weekly" gives no Window.
func TestLimit_NeverInferred_0072(t *testing.T) {
	reset := time.Now().Add(4 * time.Hour).Unix()
	cases := map[string]error{
		"a ChatGPT limit with a 4 h wait": classifyFixture("openai", 429, chatGPTLimitBody(reset, ""), http.Header{"Retry-After": {"14400"}}),
		"OpenCode Go's weekly limit": classifyFixture("opencode-go/chat_completions", 429, goLimitBody("weekly"),
			http.Header{"Retry-After": {"345600"}}),
		"a Gemini per-day quota": classifyFixture("gemini", 429, geminiQuota429("GenerateRequestsPerDayPerProjectPerModel-FreeTier", "52s"), nil),
	}
	for name, err := range cases {
		e := apiErrorOf(t, err)
		if e.RetryAfter == 0 {
			t.Errorf("%s: RetryAfter 0; the case needs a wait to tempt an inference", name)
		}
		if e.Limit != nil && e.Limit.Window != 0 {
			t.Errorf("%s: Window = %s; want none, since the service stated no length", name, e.Limit.Window)
		}
	}
	if l := limitOfErr(t, cases["OpenCode Go's weekly limit"]); l.Name != "weekly" {
		t.Errorf("OpenCode Go's weekly limit: Name = %q; want weekly", l.Name)
	}
}

// TestLimit_WindowBounds_0072 (0072-MADR D2): a stated length outside 1
// minute to 366 days, or not an integer, gives no Window; "<n> hour(s)"
// gives its hours.
func TestLimit_WindowBounds_0072(t *testing.T) {
	reset := time.Now().Add(time.Hour).Unix()
	for _, minutes := range []string{"0", "-5", "600000", "1.5", `"300"`} {
		e := apiErrorOf(t, classifyFixture("openai", 429, chatGPTLimitBody(reset, `,"limit_window_minutes":`+minutes), nil))
		if e.Limit == nil || e.Limit.Window != 0 {
			t.Errorf("limit_window_minutes %s: Limit = %+v; want Plan only, with no Window", minutes, e.Limit)
		}
	}
	for name, want := range map[string]time.Duration{"five hour": 0, "0 hours": 0, "1 hour": time.Hour, "5 hours": 5 * time.Hour, "9000 hours": 0} {
		l := limitOfErr(t, classifyFixture("opencode-go/chat_completions", 429, goLimitBody(name), nil))
		if l.Window != want {
			t.Errorf("limitName %q: Window = %s; want %s", name, l.Window, want)
		}
	}
}

// TestLimit_OpenCodeGo_0072 (0072-MADR D2, O1): Go's limitName is kept as
// Name, with its stated hours as Window; its workspace is not exposed.
func TestLimit_OpenCodeGo_0072(t *testing.T) {
	err := classifyFixture("opencode-go/chat_completions", 429, goLimitBody("5 hour"), http.Header{"Retry-After": {"120"}})
	l := limitOfErr(t, err)
	if l.Name != "5 hour" || l.Window != 5*time.Hour {
		t.Errorf("Limit = %+v; want Name \"5 hour\" and Window 5h", *l)
	}
	if s := fmt.Sprintf("%+v %s", *l, err); strings.Contains(s, "wrk_") {
		t.Errorf("the workspace id is exposed: %s", s)
	}
}

// TestLimit_Gemini_0072 (0072-MADR D2; 0026-MADR F12): a QuotaFailure's
// violations are kept in order, a numeric quotaValue as its decimal text;
// PerDay is the flag that classifies the 429.
func TestLimit_Gemini_0072(t *testing.T) {
	body := `{"error":{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[` +
		`{"quotaMetric":"generativelanguage.googleapis.com/generate_content_free_tier_requests","quotaId":"GenerateRequestsPerMinutePerProjectPerModel-FreeTier","quotaValue":"5"},` +
		`{"quotaMetric":"generativelanguage.googleapis.com/generate_content_free_tier_requests","quotaId":"GenerateRequestsPerDayPerProjectPerModel-FreeTier","quotaValue":5}]},` +
		`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"52s"}]}}`
	l := limitOfErr(t, classifyFixture("gemini", 429, body, nil))
	want := []Quota{
		{ID: "GenerateRequestsPerMinutePerProjectPerModel-FreeTier", Metric: "generativelanguage.googleapis.com/generate_content_free_tier_requests", Value: "5"},
		{ID: "GenerateRequestsPerDayPerProjectPerModel-FreeTier", Metric: "generativelanguage.googleapis.com/generate_content_free_tier_requests", Value: "5"},
	}
	if fmt.Sprint(l.Quotas) != fmt.Sprint(want) || !l.PerDay {
		t.Errorf("Limit = %+v; want Quotas %v and PerDay", *l, want)
	}
	e := apiErrorOf(t, classifyFixture("gemini", 429, geminiQuota429("GenerateRequestsPerMinutePerProjectPerModel-FreeTier", "52s"), nil))
	if e.Limit == nil || e.Limit.PerDay || !e.Retryable() {
		t.Errorf("a per-minute quota: Limit %+v, Retryable %t; want PerDay false and a retryable rate limit", e.Limit, e.Retryable())
	}
}

// TestLimit_KiloCredits_0072 (0072-MADR D2, Q6, K1, K2): Kilo's link to add
// credits is kept only when it is a plain https URL that redaction leaves
// unchanged; Code stays empty, as before.
func TestLimit_KiloCredits_0072(t *testing.T) {
	const link = "https://app.kilo.ai/credits"
	e := apiErrorOf(t, classifyFixture("kilo", 402, kiloCreditsBody(link), nil))
	if e.Limit == nil || e.Limit.CreditsURL != link || e.Code != "" || !errors.Is(e, ErrQuotaExhausted) {
		t.Errorf("Kilo's 402: Code %q, Limit %+v, %v; want ErrQuotaExhausted, no Code, and CreditsURL %s", e.Code, e.Limit, e, link)
	}
	for _, bad := range []string{
		"http://app.kilo.ai/credits",
		"https://user:pass@app.kilo.ai/credits",
		"https://app.kilo.ai/credits?token=abc",
		"https://app.kilo.ai/credits#top",
		"javascript:alert(1)",
		"https://app.kilo.ai/credits/sk-abcdefghijklmnopqrstuv",
	} {
		e := apiErrorOf(t, classifyFixture("kilo", 402, kiloCreditsBody(bad), nil))
		if e.Limit != nil && e.Limit.CreditsURL != "" {
			t.Errorf("buyCreditsUrl %q: CreditsURL = %q; want none", bad, e.Limit.CreditsURL)
		}
	}
}

// TestLimit_Redacted_0072 (0072-MADR D2; R35): a stated string is stripped
// of control characters and bounded, on a rune boundary.
func TestLimit_Redacted_0072(t *testing.T) {
	long := "plus\u001b[31m" + strings.Repeat("é", 150)
	reset := time.Now().Add(time.Hour).Unix()
	body := `{"error":{"type":"usage_limit_reached","message":"limit","plan_type":"` + strings.ReplaceAll(long, "\u001b", `\u001b`) +
		`","resets_at":` + strconv.FormatInt(reset, 10) + `}}`
	name := strings.ReplaceAll(long, "\u001b", `\u001b`)
	for label, l := range map[string]*Limit{
		"plan_type": limitOfErr(t, classifyFixture("openai", 429, body, nil)),
		"limitName": limitOfErr(t, classifyFixture("opencode-go/chat_completions", 429, goLimitBody(name), nil)),
	} {
		s := l.Plan + l.Name
		if strings.ContainsRune(s, '\u001b') || len(s) > 128 || !utf8.ValidString(s) || s == "" {
			t.Errorf("%s: %q (%d bytes); want no control character, at most 128 bytes, valid UTF-8", label, s, len(s))
		}
	}
}

// TestLimit_Nil_0072 (0072-MADR D2): an error stating nothing about its
// limit has no Limit.
func TestLimit_Nil_0072(t *testing.T) {
	for name, err := range map[string]error{
		"a 429 with only a message": classifyFixture("openai", 429, `{"error":{"type":"rate_limit_exceeded","message":"slow down"}}`, nil),
		"a 500":                     classifyFixture("claude", 500, `{"type":"error","error":{"type":"api_error","message":"oops"}}`, nil),
		"a 401":                     classifyFixture("kilo", 401, `{"error":{"message":"bad key","code":401}}`, nil),
	} {
		if l := apiErrorOf(t, err).Limit; l != nil {
			t.Errorf("%s: Limit = %+v; want nil", name, *l)
		}
	}
}
