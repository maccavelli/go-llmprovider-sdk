//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
)

// geminiBurst is the default number of requests TestLive_GeminiRateLimitShape
// sends to reach a 429; LLMPROVIDER_LIVE_GEMINI_429_BURST overrides it
// (0027-MADR Q1 a).
const geminiBurst = 20

// geminiResponse passes a request on, and keeps its response's status,
// header and body.
type geminiResponse struct {
	status int
	header http.Header
	body   []byte
}

func (g *geminiResponse) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	g.status, g.header, g.body = resp.StatusCode, resp.Header, b
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, nil
}

// TestLive_GeminiRateLimitShape (0026-MADR F12; 0027-MADR): a real Gemini
// 429, sent by the Interactions API that Generate calls, is classified as
// checkGemini429 requires: RetryAfter from the body's RetryInfo, and a
// per-day quota ErrQuotaExhausted.
//
// It deliberately exhausts a rate limit, so it runs only with
// LLMPROVIDER_LIVE_GEMINI_429 set, as well as GEMINI_API_KEY. It sends one
// request first and logs its usage, the cost of each request in the burst;
// then LLMPROVIDER_LIVE_GEMINI_429_BURST (default geminiBurst; 0 for the one
// request only) concurrent one-token Generate calls to
// LLMPROVIDER_LIVE_GEMINI_429_MODEL (default gemini-pro-latest), stopping at
// the first 429. The 429's body is logged redacted, with -v.
func TestLive_GeminiRateLimitShape(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_GEMINI_429") == "" {
		t.Skip("set LLMPROVIDER_LIVE_GEMINI_429 to exhaust a Gemini rate limit on purpose")
	}
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	model := os.Getenv("LLMPROVIDER_LIVE_GEMINI_429_MODEL")
	if model == "" {
		model = "gemini-pro-latest"
	}
	burst := geminiBurst
	if v := os.Getenv("LLMPROVIDER_LIVE_GEMINI_429_BURST"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			t.Fatalf("LLMPROVIDER_LIVE_GEMINI_429_BURST = %q; want a count, 0 or more", v)
		}
		burst = n
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	rec, err := geminiOnce(ctx, t, key, model)
	switch rec.status {
	case http.StatusOK:
		var reply struct {
			Usage json.RawMessage `json:"usage"`
		}
		_ = json.Unmarshal(rec.body, &reply)
		t.Logf("%s: one request's usage: %s", model, reply.Usage)
	case http.StatusTooManyRequests:
		reportGemini429(t, model, "the first request", rec, err)
		return
	default:
		t.Fatalf("%s: HTTP %d (%v): %s", model, rec.status, err, redact.String(string(rec.body)))
	}
	if burst == 0 {
		t.Skipf("%s: LLMPROVIDER_LIVE_GEMINI_429_BURST is 0: one request sent, no burst", model)
	}

	got, sent, gotErr := firstGemini429(ctx, t, key, model, burst)
	if t.Failed() {
		t.FailNow() // a status other than 200 or 429: the model is unusable, not unlimited
	}
	if got == nil {
		t.Skipf("%s: no 429 after a burst of %d; the key's limit is higher", model, sent)
	}
	reportGemini429(t, model, fmt.Sprintf("a burst of %d sent at once", sent), got, gotErr)
}

// reportGemini429 logs a 429, and the requests it came within, and checks it.
func reportGemini429(t *testing.T, model, within string, rec *geminiResponse, err error) {
	t.Helper()
	t.Logf("%s: 429 within %s; Retry-After header %q; error %v; body:\n%s",
		model, within, rec.header.Get("Retry-After"), err, redact.String(string(rec.body)))
	checkGemini429(t, rec.header, rec.body, err)
}

// liveGemini429Request asks for one output token.
func liveGemini429Request() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
}

// geminiOnce sends one one-token Generate, through a provider of its own, and
// returns its response and Generate's error.
func geminiOnce(ctx context.Context, t *testing.T, key, model string) (*geminiResponse, error) {
	t.Helper()
	rec := &geminiResponse{}
	p, err := gemini.New(llmprovider.WithAPIKey(key), llmprovider.WithModel(model), llmprovider.WithMaxTokens(1),
		llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Generate(ctx, liveGemini429Request())
	return rec, err
}

// firstGemini429 sends burst Generate calls at once, each through a provider
// of its own, and returns the first 429's response, how many calls were
// sent, and Generate's error for that 429. The response is nil when none
// was refused.
func firstGemini429(ctx context.Context, t *testing.T, key, model string, burst int) (*geminiResponse, int, error) {
	t.Helper()
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	var (
		mu     sync.Mutex
		got    *geminiResponse
		gotErr error
		sent   int
		wg     sync.WaitGroup
	)
	for range burst {
		wg.Go(func() {
			mu.Lock()
			sent++
			mu.Unlock()
			rec, err := geminiOnce(ctx, t, key, model)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case rec.status == http.StatusTooManyRequests && got == nil:
				got, gotErr = rec, err
				stop()
			case rec.status != 0 && rec.status != http.StatusOK && rec.status != http.StatusTooManyRequests:
				t.Errorf("%s: HTTP %d: %s", model, rec.status, redact.String(string(rec.body)))
			}
		})
	}
	wg.Wait()
	return got, sent, gotErr
}
