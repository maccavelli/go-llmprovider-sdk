package together

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

func replyServer(t *testing.T, reply func(w http.ResponseWriter)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		reply(w)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func replyProvider(t *testing.T, base string) llmprovider.Provider {
	t.Helper()
	p, err := New(llmprovider.WithAPIKey("tgp-test"), llmprovider.WithModel("m"), llmprovider.WithBaseURL(base))
	if err != nil {
		t.Fatal(err)
	}
	return llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
}

// TestGenerate_AnsweredOnceIsNotBoughtAgain (0020-MADR F9, Q2 a): an answer
// that could not be used is ErrIncomplete and sent once; a valid reply of
// 2 MiB is read; a network failure is still retried.
func TestGenerate_AnsweredOnceIsNotBoughtAgain(t *testing.T) {
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}

	srv, hits := replyServer(t, func(w http.ResponseWriter) { _, _ = io.WriteString(w, `{"choices":[]}`) })
	_, err := replyProvider(t, srv.URL).Generate(context.Background(), req)
	if !errors.Is(err, llmprovider.ErrIncomplete) || hits.Load() != 1 {
		t.Errorf("an empty answer: %d request(s), %v; want one, ErrIncomplete", hits.Load(), err)
	}

	big := strings.Repeat("x", 2<<20)
	srv, hits = replyServer(t, func(w http.ResponseWriter) {
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"`+big+`"}}]}`)
	})
	resp, err := replyProvider(t, srv.URL).Generate(context.Background(), req)
	if err != nil || len(resp.OutputText()) != len(big) || hits.Load() != 1 {
		t.Errorf("a 2 MiB answer: %d request(s), %v; want it read once", hits.Load(), err)
	}

	srv, hits = replyServer(t, func(w http.ResponseWriter) {
		if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
			_ = conn.Close() // the connection drops before any reply
		}
	})
	_, _ = replyProvider(t, srv.URL).Generate(context.Background(), req)
	if hits.Load() != 3 {
		t.Errorf("a dropped connection: %d request(s), want 3: a network failure is retried", hits.Load())
	}
}
