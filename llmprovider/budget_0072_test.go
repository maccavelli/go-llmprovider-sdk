package llmprovider

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// This file calls only API that exists at v1.28.0, so that 0072-PLAN can
// show its tests red there. Its fixtures are synthetic, from OpenCode's
// client's (0072-MADR O3 to O5, O7).

// budgetBody is OpenCode's budget 429 (0072-MADR O4).
const budgetBody = `{"type":"error","error":{"type":"BudgetLimitError","message":"A budget policy for this model has been reached."},` +
	`"metadata":{"coverage":"models","resetsAt":"2026-11-01T00:00:00.000Z"}}`

// TestClassify_OpenCodeBudget_0072 (0072-MADR D4): OpenCode's budget and
// credit limit are ErrQuotaExhausted and not retryable, in OpenCode's
// envelope and in Google's.
func TestClassify_OpenCodeBudget_0072(t *testing.T) {
	googleBudget := `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"Policy budget exceeded","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"BudgetLimitError","domain":"opencode.ai",` +
		`"metadata":{"coverage":"all","resetsAt":"2026-11-01T00:00:00.000Z"}}]}}`
	cases := []struct {
		name, provider string
		status         int
		body           string
	}{
		{"a budget", "opencode-zen/chat_completions", 429, budgetBody},
		{"a budget in Google's envelope", "opencode-go/google", 429, googleBudget},
		{"a credit limit", "opencode-zen/chat_completions", 402,
			`{"type":"error","error":{"type":"CreditLimitExceeded","message":"Credit limit exceeded."}}`},
	}
	for _, c := range cases {
		err := classifyFixture(c.provider, c.status, c.body, nil)
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok || !errors.Is(err, ErrQuotaExhausted) || apiErr.Retryable() {
			t.Errorf("%s: %v; want a terminal ErrQuotaExhausted", c.name, err)
		}
	}
}

// TestRetry_OpenCodeBudgetNotResent_0072 (0072-MADR D4): WithRetry does not
// resend a request an OpenCode budget refused.
func TestRetry_OpenCodeBudgetNotResent_0072(t *testing.T) {
	var sends atomic.Int32
	p := &stubProvider{id: "opencode-zen", gen: func(context.Context, *Request) (*Response, error) {
		sends.Add(1)
		return nil, classifyFixture("opencode-zen/chat_completions", 429, budgetBody, http.Header{})
	}}
	r := WithRetry(p, RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond})
	if _, err := r.Generate(context.Background(), &Request{Input: textInput("hi")}); !errors.Is(err, ErrQuotaExhausted) {
		t.Errorf("err = %v; want ErrQuotaExhausted", err)
	}
	if n := sends.Load(); n != 1 {
		t.Errorf("sent %d times; want once: a budget is not retried", n)
	}
}
