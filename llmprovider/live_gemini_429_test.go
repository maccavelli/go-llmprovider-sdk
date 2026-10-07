//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// geminiBurst bounds the requests TestLive_GeminiRateLimitShape sends to
// reach a 429; each asks for one output token.
const geminiBurst = 20

// TestLive_GeminiRateLimitShape (0026-MADR F12, its PLAN's Q6 a): a real
// Gemini 429 carries no Retry-After header; its body's google.rpc.RetryInfo
// gives the delay, and a QuotaFailure names the quota. The classification
// reads both: RetryAfter is the retryDelay, and a per-day quota is
// ErrQuotaExhausted.
//
// It deliberately exhausts a rate limit, so it runs only with
// LLMPROVIDER_LIVE_GEMINI_429 set, as well as GEMINI_API_KEY. It sends up to
// geminiBurst one-token requests at once to LLMPROVIDER_LIVE_GEMINI_429_MODEL
// (default gemini-pro-latest, an alias that follows the current pro model),
// and stops at the first 429. The body is logged redacted, with -v, so a
// fixture can be taken from it.
func TestLive_GeminiRateLimitShape(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_GEMINI_429") == "" {
		t.Skip("set LLMPROVIDER_LIVE_GEMINI_429 to exhaust a Gemini rate limit on purpose")
	}
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	model := os.Getenv("LLMPROVIDER_LIVE_GEMINI_429_MODEL")
	if model == "" {
		model = "gemini-pro-latest"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	header, body, sent := firstGemini429(ctx, t, key, model)
	if t.Failed() {
		t.FailNow() // a status other than 200 or 429: the model is unusable, not unlimited
	}
	if body == nil {
		t.Skipf("%s: no 429 after %d requests; the key's limit is higher than the burst", model, sent)
	}
	t.Logf("%s: 429 after %d requests; Retry-After header %q; body:\n%s",
		model, sent, header.Get("Retry-After"), redact.String(string(body)))

	var shape struct {
		Error struct {
			Status  string           `json:"status"`
			Details []map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &shape); err != nil {
		t.Fatalf("DRIFT: the 429 body is not JSON: %v", err)
	}
	if shape.Error.Status != "RESOURCE_EXHAUSTED" {
		t.Errorf("DRIFT: error.status = %q; want RESOURCE_EXHAUSTED", shape.Error.Status)
	}
	var retryDelay string
	for _, d := range shape.Error.Details {
		if strings.HasSuffix(d["@type"].(string), "google.rpc.RetryInfo") {
			retryDelay, _ = d["retryDelay"].(string)
		}
	}
	if retryDelay == "" {
		t.Fatalf("DRIFT: no google.rpc.RetryInfo retryDelay in the 429 body")
	}
	want, err := time.ParseDuration(retryDelay)
	if err != nil {
		t.Fatalf("DRIFT: retryDelay %q is not a duration: %v", retryDelay, err)
	}

	err = llmprovider.ClassifyHTTPError("gemini", &http.Response{StatusCode: http.StatusTooManyRequests,
		Status: "429 Too Many Requests", Header: header, Body: io.NopCloser(bytes.NewReader(body))})
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		t.Fatalf("ClassifyHTTPError: %v; want an *APIError", err)
	}
	if header.Get("Retry-After") == "" && apiErr.RetryAfter != want {
		t.Errorf("RetryAfter = %s; want the body's retryDelay, %s", apiErr.RetryAfter, want)
	}
	perDay := strings.Contains(string(body), "PerDay")
	if perDay != errors.Is(err, llmprovider.ErrQuotaExhausted) {
		t.Errorf("a per-day quota = %t, but ErrQuotaExhausted = %t: %v", perDay, !perDay, err)
	}
}

// firstGemini429 sends up to geminiBurst requests at once, and returns the
// first 429's header and body, and how many requests were sent. The body is
// nil when none was refused.
func firstGemini429(ctx context.Context, t *testing.T, key, model string) (http.Header, []byte, int) {
	t.Helper()
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	url := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent"
	const request = `{"contents":[{"parts":[{"text":"hi"}]}],"generationConfig":{"maxOutputTokens":1}}`
	var (
		mu     sync.Mutex
		header http.Header
		body   []byte
		sent   int
		wg     sync.WaitGroup
	)
	for range geminiBurst {
		wg.Go(func() {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(request))
			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-goog-api-key", key)
			mu.Lock()
			sent++
			mu.Unlock()
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return // cancelled after the first 429, or a transport failure
			}
			defer func() { _ = resp.Body.Close() }()
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case resp.StatusCode == http.StatusTooManyRequests && body == nil:
				header, body = resp.Header, b
				stop()
			case resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests:
				t.Errorf("%s: HTTP %d: %s", model, resp.StatusCode, redact.String(string(b)))
			}
		})
	}
	wg.Wait()
	return header, body, sent
}
