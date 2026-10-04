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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// replyServerReq is replyServer with the request in hand.
func replyServerReq(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		reply(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

var hiRequest = &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}

// TestGenerate_CutBodyIsUnavailable (0021-MADR W6, D1): a 200 whose body is cut
// short is a failure to reach the service, marked after the reply, and
// WithRetry sends it once more, not to MaxAttempts.
func TestGenerate_CutBodyIsUnavailable(t *testing.T) {
	srv, hits := replyServer(t, func(w http.ResponseWriter) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","mes`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
			_ = conn.Close()
		}
	})
	_, err := replyProvider(t, srv.URL).Generate(context.Background(), hiRequest)
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !transport.IsAfterReply(err) {
		t.Errorf("a cut body: %v; want ErrProviderUnavailable after the reply", err)
	}
	if hits.Load() != 2 {
		t.Errorf("a cut body was sent %d time(s), want 2: retried once", hits.Load())
	}
}

// TestGenerate_OverLimitIsIncomplete (0021-MADR W6): a 200 over ReplyLimit is
// ErrIncomplete, says so, and is sent once.
func TestGenerate_OverLimitIsIncomplete(t *testing.T) {
	srv, hits := replyServer(t, func(w http.ResponseWriter) {
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"`+
			strings.Repeat("x", wire.ReplyLimit)+`"}}]}`)
	})
	_, err := replyProvider(t, srv.URL).Generate(context.Background(), hiRequest)
	if !errors.Is(err, llmprovider.ErrIncomplete) || transport.IsAfterReply(err) || err == nil ||
		!strings.Contains(err.Error(), "over 16 MiB") {
		t.Errorf("a reply over the limit: %v; want ErrIncomplete, over 16 MiB", err)
	}
	if hits.Load() != 1 {
		t.Errorf("a reply over the limit was sent %d time(s), want 1", hits.Load())
	}
}

// TestGenerate_IdleBodyIsUnavailable (0021-MADR D2): a body that sends nothing
// for the idle limit ends the request, whichever client sends it.
func TestGenerate_IdleBodyIsUnavailable(t *testing.T) {
	defer wire.SetIdleTimeout(50 * time.Millisecond)()
	for name, opts := range map[string][]llmprovider.Option{
		"default client": nil,
		"caller client":  {llmprovider.WithHTTPClient(&http.Client{})},
	} {
		t.Run(name, func(t *testing.T) {
			srv, _ := replyServerReq(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, "{")
				w.(http.Flusher).Flush()
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
				}
			})
			p, err := New(append([]llmprovider.Option{llmprovider.WithAPIKey("tgp-test"), llmprovider.WithModel("m"),
				llmprovider.WithBaseURL(srv.URL)}, opts...)...)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			start := time.Now()
			_, err = p.Generate(ctx, hiRequest)
			if !errors.Is(err, transport.ErrIdleTimeout) || !transport.IsAfterReply(err) {
				t.Errorf("an idle body: %v; want ErrIdleTimeout after the reply", err)
			}
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Errorf("an idle body took %s, want the 50 ms idle limit", elapsed)
			}
		})
	}
}
