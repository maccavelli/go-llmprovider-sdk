package llmprovider_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// cutThenAnswer answers the first request with the start of a reply, in the
// format the request asks for, and then cuts the connection; every later
// request gets the whole answer "ok".
func cutThenAnswer(t *testing.T) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	requests := 0
	const (
		sseHead  = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"par\"}\n\n"
		sseWhole = "data:{\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r2\"}}\n\n"
		jsonHead  = `{"id":"r1","output":[{"type":"message","content":[{"type":"output_text","text":"par`
		jsonWhole = `{"id":"r2","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		first := requests == 1
		mu.Unlock()
		streamed := r.Header.Get("Accept") == "text/event-stream"
		if !first {
			if streamed {
				fmt.Fprint(w, sseWhole)
			} else {
				fmt.Fprint(w, jsonWhole)
			}
			return
		}
		head, contentType := jsonHead, "application/json"
		if streamed {
			head, contentType = sseHead, "text/event-stream"
		}
		conn, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		// A longer Content-Length than is sent: the client reads the head,
		// then the cut.
		fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s", contentType, len(head)+100, head)
		_ = buf.Flush()
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)
	return srv, func() int {
		mu.Lock()
		defer mu.Unlock()
		return requests
	}
}

// TestWithRetry_StreamIsBuffered (0031-MADR D6, Q1): WithRetry over a
// provider that streams natively does not stream: it is no Streamer, reports
// NativeStreaming Unsupported, and Stream over it retries a reply cut part
// way, yielding events only from the attempt that succeeds.
func TestWithRetry_StreamIsBuffered(t *testing.T) {
	srv, requests := cutThenAnswer(t)
	inner, err := openai.New(llmprovider.WithAPIKey("k"), llmprovider.WithModel("gpt-4.1-mini"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := inner.(llmprovider.Streamer); !ok {
		t.Fatal("the inner provider does not stream natively; the test proves nothing")
	}
	p := llmprovider.WithRetry(inner, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	if _, ok := p.(llmprovider.Streamer); ok {
		t.Error("WithRetry's result is a Streamer")
	}
	if got := p.Capabilities().NativeStreaming; got != llmprovider.Unsupported {
		t.Errorf("WithRetry's NativeStreaming is %v, want Unsupported", got)
	}
	var kinds []string
	for ev, err := range llmprovider.Stream(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}) {
		if err != nil {
			t.Fatalf("after %v: %v", kinds, err)
		}
		kind := string(ev.Type)
		if ev.Type == llmprovider.EventTextDelta {
			kind += ":" + ev.Text
		}
		kinds = append(kinds, kind)
	}
	if want := []string{"text_delta:ok", "item", "done"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events %v, want %v: only the attempt that succeeded", kinds, want)
	}
	if n := requests(); n != 2 {
		t.Errorf("%d requests, want 2: the cut reply, then its one retry", n)
	}
}
