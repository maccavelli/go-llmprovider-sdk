package wire_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

// streamCall is a streamed call to url, through a client with its own
// transport and no idle connections, so its goroutines end with its reply.
func streamCall(url string) wire.Call {
	return wire.Call{
		Provider: "p",
		Client:   &http.Client{Transport: &http.Transport{DisableKeepAlives: true}},
		Logger:   slog.New(slog.DiscardHandler),
		URL:      url,
		Body:     map[string]any{"stream": true},
		Stream:   true,
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("Authorization", "Bearer "+token.Value)
		},
	}
}

// sseEvent frames one Responses event.
func sseEvent(payload string) string { return "event: x\ndata: " + payload + "\n\n" }

const (
	textDelta = `{"type":"response.output_text.delta","delta":"hi"}`
	itemDone  = `{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"hi"}]}}`
	completed = `{"type":"response.completed","response":{"id":"r1"}}`
)

// streamEvents opens c and reads it with responses.Events, collecting what
// it yields.
func streamEvents(ctx context.Context, c wire.Call, token llmprovider.Token) ([]llmprovider.Event, error) {
	s, err := wire.OpenStream(ctx, c, token)
	if err != nil {
		return nil, err
	}
	var events []llmprovider.Event
	err = s.Read(func(body io.Reader) error {
		return responses.Events(c.Provider, body, func(ev llmprovider.Event) bool {
			events = append(events, ev)
			return true
		})
	})
	return events, err
}

// postStream is today's path for the same reply: Post with ReadStream.
func postStream(c wire.Call) error {
	_, err := wire.Post(context.Background(), c, llmprovider.Token{}, func(body io.Reader) (*llmprovider.Response, error) {
		return responses.ReadStream(c.Provider, body)
	})
	return err
}

// TestOpenStream_NonSuccessClassified (0031-MADR D3): a non-2xx is the
// *APIError Post gives for it, from the open step, before any event.
func TestOpenStream_NonSuccessClassified(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusForbidden, http.StatusInternalServerError} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":{"message":"no"}}`)
		}))
		_, openErr := wire.OpenStream(context.Background(), streamCall(srv.URL), llmprovider.Token{})
		postErr := postStream(streamCall(srv.URL))
		srv.Close()
		got, okGot := errors.AsType[*llmprovider.APIError](openErr)
		want, okWant := errors.AsType[*llmprovider.APIError](postErr)
		if !okGot || !okWant || got.Status != want.Status || got.Kind != want.Kind || //nolint:errorlint // the kind itself, not what it wraps
			got.RetryAfter != want.RetryAfter {
			t.Errorf("HTTP %d: OpenStream %v, Post %v; want the same *APIError", status, openErr, postErr)
		}
	}
}

// renewingSource is a token source that can be told its token was refused.
type renewingSource struct {
	invalidated int
}

func (s *renewingSource) Token(context.Context) (llmprovider.Token, error) {
	if s.invalidated > 0 {
		return llmprovider.Token{Value: "fresh"}, nil
	}
	return llmprovider.Token{Value: "stale"}, nil
}

func (s *renewingSource) Invalidate() { s.invalidated++ }

// TestOpenStream_ReauthBeforeEvents (0031-MADR D3): a 401 arrives before any
// event, so Reauth renews the credential once and the second request streams.
func TestOpenStream_ReauthBeforeEvents(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer fresh" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, sseEvent(textDelta)+sseEvent(itemDone)+sseEvent(completed))
	}))
	defer srv.Close()
	src := &renewingSource{}
	c := streamCall(srv.URL)
	s, err := wire.Reauth(context.Background(), "p", src, func(token llmprovider.Token) (*wire.Stream, error) {
		return wire.OpenStream(context.Background(), c, token)
	})
	if err != nil {
		t.Fatal(err)
	}
	var events []llmprovider.Event
	err = s.Read(func(body io.Reader) error {
		return responses.Events("p", body, func(ev llmprovider.Event) bool {
			events = append(events, ev)
			return true
		})
	})
	if err != nil || len(events) != 3 || events[2].Type != llmprovider.EventDone {
		t.Fatalf("after a renewal: %d events, %v; want a delta, an item and EventDone", len(events), err)
	}
	if src.invalidated != 1 || requests != 2 {
		t.Errorf("invalidated %d times over %d requests; want once over 2", src.invalidated, requests)
	}
}

// TestStream_Failures (0031-MADR D4): an idle stall, an event over the limit
// and an early end each end the stream with the error Post and ReadStream
// give for the same reply, and no EventDone.
func TestStream_Failures(t *testing.T) {
	defer wire.SetIdleTimeout(50 * time.Millisecond)()
	cases := []struct {
		name    string
		handler http.HandlerFunc
		kind    error
		after   bool
	}{
		{"idle stall", func(w http.ResponseWriter, r *http.Request) {
			// The server sees the client go only once it has read the
			// request body.
			_, _ = io.Copy(io.Discard, r.Body)
			_, _ = io.WriteString(w, sseEvent(textDelta))
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
		}, llmprovider.ErrProviderUnavailable, true},
		{"event too large", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, sseEvent(`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"`+
				strings.Repeat("x", 16<<20)+`"}]}}`))
		}, llmprovider.ErrIncomplete, false},
		{"ends early", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, sseEvent(textDelta))
		}, llmprovider.ErrProviderUnavailable, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			events, err := streamEvents(context.Background(), streamCall(srv.URL), llmprovider.Token{})
			postErr := postStream(streamCall(srv.URL))
			if !errors.Is(err, tc.kind) || transport.IsAfterReply(err) != tc.after {
				t.Fatalf("stream error %v; want %v, after the reply %v", err, tc.kind, tc.after)
			}
			if !errors.Is(postErr, tc.kind) || transport.IsAfterReply(postErr) != tc.after {
				t.Fatalf("Post's error %v differs from the stream's %v", postErr, err)
			}
			for _, ev := range events {
				if ev.Type == llmprovider.EventDone {
					t.Fatalf("EventDone after a failure: %+v", events)
				}
			}
		})
	}
}

// heldServer writes one delta, then holds the stream open until the client
// goes, and reports that it went.
func heldServer(t *testing.T) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	gone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The server sees the client go only once it has read the request
		// body.
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, sseEvent(textDelta))
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
			close(gone)
		case <-time.After(5 * time.Second):
		}
	}))
	return srv, gone
}

// settled waits for the goroutine count to fall back to at most n.
func settled(n int) bool {
	for range 200 {
		if runtime.NumGoroutine() <= n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TestStream_StopClosesBody (0031-MADR D3): a caller that stops ranging
// closes the reply: the server sees the client go, and no goroutine is left.
func TestStream_StopClosesBody(t *testing.T) {
	srv, gone := heldServer(t)
	defer srv.Close()
	before := runtime.NumGoroutine()
	s, err := wire.OpenStream(context.Background(), streamCall(srv.URL), llmprovider.Token{})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = s.Read(func(body io.Reader) error {
		return responses.Events("p", body, func(llmprovider.Event) bool {
			calls++
			return false
		})
	})
	if err != nil || calls != 1 {
		t.Fatalf("stopped after one event: %d calls, %v", calls, err)
	}
	select {
	case <-gone:
	case <-time.After(2 * time.Second):
		t.Fatal("the server never saw the client go: the body was left open")
	}
	if !settled(before) {
		t.Errorf("%d goroutines, %d before the stream", runtime.NumGoroutine(), before)
	}
}

// TestStream_ContextCancel (0031-MADR D3): cancelling ctx mid-stream ends it
// with the context's error, closes the reply, and leaves no goroutine.
func TestStream_ContextCancel(t *testing.T) {
	srv, gone := heldServer(t)
	defer srv.Close()
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := wire.OpenStream(ctx, streamCall(srv.URL), llmprovider.Token{})
	if err != nil {
		t.Fatal(err)
	}
	err = s.Read(func(body io.Reader) error {
		return responses.Events("p", body, func(llmprovider.Event) bool {
			cancel()
			return true
		})
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled stream: %v; want context.Canceled", err)
	}
	select {
	case <-gone:
	case <-time.After(2 * time.Second):
		t.Fatal("the server never saw the client go")
	}
	if !settled(before) {
		t.Errorf("%d goroutines, %d before the stream", runtime.NumGoroutine(), before)
	}
}
