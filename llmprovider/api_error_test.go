package llmprovider

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestClassifyHTTPError_Table pins MADR 0012 §1.1's classification, with the
// rows its 2026-09-27 amendment added: sentinel, Terminal, Type, a bounded
// Message, and the pre-0012 status sentinel still matching a terminal error
// (MADR 0012 §7, and its amendment of 2026-10-07).
func TestClassifyHTTPError_Table(t *testing.T) {
	opencode := func(errType, msg string) string {
		return `{"type":"error","error":{"type":"` + errType + `","message":"` + msg + `"}}`
	}
	for _, test := range []struct {
		name     string
		provider string
		status   int
		body     string
		want     error
		terminal bool
		errType  string
		message  string
	}{
		{"opencode free usage limit", "opencode-zen/responses", 429, opencode("FreeUsageLimitError", "free limit reached"), ErrQuotaExhausted, true, "FreeUsageLimitError", "free limit reached"},
		{"opencode go usage limit", "opencode-go/chat_completions", 429, opencode("GoUsageLimitError", "go limit"), ErrQuotaExhausted, true, "GoUsageLimitError", "go limit"},
		{"opencode black usage limit", "opencode-zen/messages", 429, opencode("BlackUsageLimitError", "limit"), ErrQuotaExhausted, true, "BlackUsageLimitError", "limit"},
		{"opencode credits", "opencode-zen/responses", 401, opencode("CreditsError", "no credits"), ErrQuotaExhausted, true, "CreditsError", "no credits"},
		{"opencode monthly limit", "opencode-zen/responses", 401, opencode("MonthlyLimitError", "monthly"), ErrQuotaExhausted, true, "MonthlyLimitError", "monthly"},
		{"opencode user limit", "opencode-zen/responses", 401, opencode("UserLimitError", "user"), ErrQuotaExhausted, true, "UserLimitError", "user"},
		{"opencode region", "opencode-go/messages", 403, opencode("RegionError", "region"), ErrNotPermitted, true, "RegionError", "region"},
		{"opencode data policy", "opencode-go/messages", 403, opencode("DataPolicyError", "policy"), ErrNotPermitted, true, "DataPolicyError", "policy"},
		{"opencode free tier", "opencode-zen/chat_completions", 403, opencode("FreeTierError", "OpenCode's free tier can only be used from within OpenCode"), ErrNotPermitted, true, "FreeTierError", "free tier"},
		{"opencode model", "opencode-zen/responses", 401, opencode("ModelError", "unknown model"), ErrInvalidRequest, true, "ModelError", "unknown model"},
		{"opencode upstream 402", "opencode-zen/google", 402, `{"type":"error","error":{"type":"error","message":"Upstream request failed: Insufficient account funds"}}`, ErrQuotaExhausted, true, "", "Insufficient account funds"},
		{"kilo promotion limit", "kilo", 429, `{"error":{"code":"PROMOTION_MODEL_LIMIT_REACHED","message":"promo"}}`, ErrQuotaExhausted, true, "PROMOTION_MODEL_LIMIT_REACHED", "promo"},
		{"kilo free usage text", "kilo", 429, `{"error":{"message":"FreeUsageLimitError: try later"}}`, ErrQuotaExhausted, true, "", "try later"},
		{"kilo 402", "kilo", 402, `{"error":{"code":"insufficient_balance","message":"top up"}}`, ErrQuotaExhausted, true, "insufficient_balance", "top up"},
		{"kilo 403", "kilo", 403, `{"error":{"message":"denied"}}`, ErrNotPermitted, true, "", "denied"},
		{"kilo paid model auth", "kilo", 401, `{"code":"PAID_MODEL_AUTH_REQUIRED"}`, ErrInvalidRequest, true, "PAID_MODEL_AUTH_REQUIRED", ""},
		{"codex usage limit", "openai", 429, `{"error":{"type":"usage_limit_reached","message":"reached"}}`, ErrQuotaExhausted, true, "usage_limit_reached", "reached"},
		{"openai insufficient quota", "openai", 429, `{"error":{"type":"insufficient_quota","code":"insufficient_quota","message":"quota"}}`, ErrQuotaExhausted, true, "insufficient_quota", "quota"},
		{"codex usage not included", "openai", 403, `{"error":{"type":"usage_not_included","message":"plan"}}`, ErrNotPermitted, true, "usage_not_included", "plan"},
		{"401 otherwise", "grok", 401, `{"code":"unauthenticated","error":"bad key"}`, ErrAuthFailure, true, "unauthenticated", "bad key"},
		{"403 otherwise", "claude", 403, `{"type":"error","error":{"type":"permission_error","message":"no"}}`, ErrAuthFailure, true, "permission_error", "no"},
		{"408", "huggingface", 408, `{"error":"timeout"}`, ErrProviderUnavailable, false, "", "timeout"},
		{"500", "gemini", 500, `{"error":{"code":500,"message":"internal","status":"INTERNAL"}}`, ErrProviderUnavailable, false, "INTERNAL", "internal"},
		{"525", "kilo", 525, "", ErrProviderUnavailable, true, "", ""},
		{"526", "kilo", 526, "", ErrProviderUnavailable, true, "", ""},
		{"400 otherwise", "ollama", 400, `{"error":"model not found"}`, ErrInvalidRequest, true, "", "model not found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := classifyFixture(test.provider, test.status, test.body, nil)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error %T is not an *APIError", err)
			}
			// A terminal error still matches its status's pre-0012 sentinel;
			// a retryable one leaves it out, so none of these rows' retryable
			// errors matches ErrInvalidRequest (0012-MADR amendment
			// 2026-10-07, 0026-MADR F23).
			legacy := statusSentinel(test.status)
			switch {
			case apiErr.Retryable() && errors.Is(err, ErrInvalidRequest):
				t.Errorf("error = %v is retryable and matches %v", err, ErrInvalidRequest)
			case !apiErr.Retryable() && !errors.Is(err, legacy):
				t.Errorf("error = %v no longer matches the pre-0012 sentinel %v", err, legacy)
			}
			if apiErr.terminal != test.terminal || apiErr.Code != test.errType || apiErr.Status != test.status {
				t.Errorf("terminal/Code/Status = %t/%q/%d, want %t/%q/%d",
					apiErr.terminal, apiErr.Code, apiErr.Status, test.terminal, test.errType, test.status)
			}
			if !strings.Contains(apiErr.Message, test.message) || !strings.Contains(err.Error(), test.provider) {
				t.Errorf("error = %q, want the provider and the message %q", err, test.message)
			}
		})
	}
}

// TestClassifyHTTPError_PlainRateLimit: a 429 with no quota classification is
// a retryable *APIError of kind ErrRateLimited, with its retry-after and the
// service's message (0015-MADR D7).
func TestClassifyHTTPError_PlainRateLimit(t *testing.T) {
	err := classifyFixture("kilo", 429, `{"error":{"message":"slow down"}}`, http.Header{"Retry-After": {"7"}})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !errors.Is(apiErr.Kind, ErrRateLimited) || errors.Is(err, ErrQuotaExhausted) ||
		!apiErr.Retryable() || apiErr.Status != 429 || apiErr.RetryAfter != 7*time.Second || apiErr.Message != "slow down" {
		t.Fatalf("error = %#v, want a retryable *APIError of kind ErrRateLimited, with retry-after 7s and the message", err)
	}
}

func TestClassifyHTTPError_BoundsMessage(t *testing.T) {
	err := classifyFixture("kilo", 400, `{"error":{"message":"`+strings.Repeat("é", 600)+`"}}`, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.Message) > apiErrorMessageLimit+len("…") {
		t.Fatalf("message is %d bytes, want at most %d", len(apiErr.Message), apiErrorMessageLimit+len("…"))
	}
}

func TestClassifyHTTPError_RedactsMessage(t *testing.T) {
	err := classifyFixture("openai", 400,
		`{"error":{"message":"bad token eyJhbGciOiJub25lIn0.eyJzdWIiOiJ4In0.sig"}}`, nil)
	if err == nil || strings.Contains(err.Error(), "eyJ") {
		t.Fatalf("error = %v, want the token redacted", err)
	}
}

// classifyFixture classifies a synthetic response and closes its body.
func classifyFixture(provider string, status int, body string, header http.Header) error {
	if header == nil {
		header = http.Header{}
	}
	resp := &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}
	defer closeResponseBody(resp)
	return ClassifyHTTPError(provider, resp)
}

// TestClassifyHTTPStatus pins the status -> sentinel mapping shared by every
// gateway provider, including Kilo's documented 402.
func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		retryAfter string
		wantErr    error
		wantNil    bool
	}{
		{"200 is success", http.StatusOK, "", nil, true},
		{"429 without Retry-After", http.StatusTooManyRequests, "", ErrRateLimited, false},
		{"429 with Retry-After", http.StatusTooManyRequests, "7", ErrRateLimited, false},
		{"401", http.StatusUnauthorized, "", ErrAuthFailure, false},
		{"403", http.StatusForbidden, "", ErrAuthFailure, false},
		{"402 insufficient balance is non-retryable", http.StatusPaymentRequired, "", ErrInvalidRequest, false},
		{"400", http.StatusBadRequest, "", ErrInvalidRequest, false},
		{"404", http.StatusNotFound, "", ErrInvalidRequest, false},
		{"500", http.StatusInternalServerError, "", ErrProviderUnavailable, false},
		{"503", http.StatusServiceUnavailable, "", ErrProviderUnavailable, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status, Header: http.Header{}}
			if tc.retryAfter != "" {
				resp.Header.Set("Retry-After", tc.retryAfter)
			}
			err := ClassifyHTTPError("gw/route", resp)
			if tc.wantNil {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want wrapping %v", err, tc.wantErr)
			}
			// Every branch names the provider/route so a failure is
			// attributable from the error alone, 429 included.
			if !strings.Contains(err.Error(), "gw/route") {
				t.Errorf("error must name the provider/route: %v", err)
			}
			if tc.status == http.StatusTooManyRequests {
				var rl *APIError
				if !errors.As(err, &rl) || !errors.Is(rl.Kind, ErrRateLimited) {
					t.Fatalf("429 must yield an *APIError of kind ErrRateLimited, got %#v", err)
				}
				want := time.Duration(0)
				if tc.retryAfter == "7" {
					want = 7 * time.Second
				}
				if rl.RetryAfter != want {
					t.Errorf("RetryAfter = %v, want %v", rl.RetryAfter, want)
				}
			}
		})
	}
}

// htmlErrorPage is a gateway's 64 KiB HTML error page, with one link.
func htmlErrorPage() string {
	const para = "<p>The upstream server returned an invalid response while handling this request. Please wait a moment and try again.</p>\n"
	head := `<!DOCTYPE html><html><head><title>502 Bad Gateway</title><style>body{font-family:sans-serif}</style></head>` +
		`<body><h1>Bad Gateway</h1><p>See <a href="https://status.example.com/incidents">the status page</a>.</p>\n`
	body := head + strings.Repeat(para, (64<<10-len(head))/len(para)+1)
	return body[:64<<10]
}

// BenchmarkClassifyHTTPError_64KiBHTML (0021-MADR Z1): the cost of
// classifying a 502 whose body is a 64 KiB HTML page.
func BenchmarkClassifyHTTPError_64KiBHTML(b *testing.B) {
	page := htmlErrorPage()
	b.ReportAllocs()
	for b.Loop() {
		resp := &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(page))}
		_ = ClassifyHTTPError("p", resp)
	}
}

// controlRune reports a rune a terminal may act on: C0 except tab, DEL, or C1.
func controlRune(r rune) bool {
	return r < 0x20 && r != '\t' || r == 0x7f || r >= 0x80 && r <= 0x9f
}

// TestClassify_StripsControlCharacters (0021-MADR Z3): an error body's
// escape sequences never reach Error(), Code or Message, and Code is
// bounded.
func TestClassify_StripsControlCharacters(t *testing.T) {
	body := `{"error":{"type":"x\u001b[31m","message":"\u001b]0;owned\u0007\u001b[2Jhello\u009b"}}`
	resp := &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
	var apiErr *APIError
	if !errors.As(ClassifyHTTPError("p", resp), &apiErr) {
		t.Fatal("not an APIError")
	}
	stream := ClassifyStreamFailure("p", "c\u001b[0m", "", "\u001b[2Jbye")
	var streamErr *APIError
	if !errors.As(stream, &streamErr) {
		t.Fatal("stream: not an APIError")
	}
	for _, s := range []string{apiErr.Error(), apiErr.Code, apiErr.Message, streamErr.Error(), streamErr.Code, streamErr.Message} {
		if strings.ContainsFunc(s, controlRune) {
			t.Errorf("%q holds a control character", s)
		}
	}
	if !strings.Contains(apiErr.Message, "hello") {
		t.Errorf("Message = %q, want the text kept", apiErr.Message)
	}
	long := `{"error":{"type":"` + strings.Repeat("a", 300) + `","message":"m"}}`
	resp = &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(long))}
	if errors.As(ClassifyHTTPError("p", resp), &apiErr) && len(apiErr.Code) > 128 {
		t.Errorf("Code is %d bytes, want at most 128", len(apiErr.Code))
	}
}
