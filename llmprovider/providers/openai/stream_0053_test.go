package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

const (
	delta0053  = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"par\"}\n\n"
	failed0053 = "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"server_error\",\"message\":\"0053\"}}}\n\n"
)

// sseServer0053 answers every request with body, as an event stream, and
// counts them.
func sseServer0053(t *testing.T, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func request0053() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
}

// TestGenerate_InStreamFailureOnce_0053 (0053-MADR D3, M4): a ChatGPT
// session's Generate reads an event stream; a failure inside it, after a
// delta, is retried at most once, whatever MaxAttempts says. It was retried
// up to MaxAttempts times.
func TestGenerate_InStreamFailureOnce_0053(t *testing.T) {
	srv, n := sseServer0053(t, delta0053+failed0053)
	p, err := New(llmprovider.WithTokenSource(&auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "chatgpt-access",
		Expiry: time.Now().Add(time.Hour), TokenURL: "http://127.0.0.1:1/never-refresh"}),
		llmprovider.WithModel("gpt-5.5"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, attempts := range []int{3, 5} {
		n.Store(0)
		w := llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: attempts, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond})
		_, err := w.Generate(context.Background(), request0053())
		var apiErr *llmprovider.APIError
		if n.Load() != 2 || !errors.Is(err, llmprovider.ErrProviderUnavailable) || !errors.As(err, &apiErr) || !transport.IsAfterReply(err) {
			t.Errorf("MaxAttempts %d: %d requests, %v (after the reply: %v); want 2, ErrProviderUnavailable, marked",
				attempts, n.Load(), err, transport.IsAfterReply(err))
		}
	}
}

// TestStreamRetry_InStreamFailureBeforeFirstEvent_0053 (0053-MADR D3): a
// stream whose first event is response.failed is retried at most once under
// WithStreamRetry; one that fails after a delta is passed through, as
// 0051-MADR D2 has it.
func TestStreamRetry_InStreamFailureBeforeFirstEvent_0053(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int32
	}{
		{"before the first event", failed0053, 2},
		{"after a delta", delta0053 + failed0053, 1},
	} {
		srv, n := sseServer0053(t, tc.body)
		p, err := New(llmprovider.WithAPIKey("sk-0053"), llmprovider.WithModel("gpt-5.5"), llmprovider.WithBaseURL(srv.URL))
		if err != nil {
			t.Fatal(err)
		}
		w := llmprovider.WithStreamRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, nil)
		var last error
		for _, err := range llmprovider.Stream(context.Background(), w, request0053()) {
			if err != nil {
				last = err
			}
		}
		if n.Load() != tc.want || !errors.Is(last, llmprovider.ErrProviderUnavailable) {
			t.Errorf("%s: %d requests, %v; want %d and ErrProviderUnavailable", tc.name, n.Load(), last, tc.want)
		}
	}
}
