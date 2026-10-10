package grok

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// grokStream is the event stream xAI sent in 0031-PLAN Phase 1, shortened:
// a reasoning summary delta, its item, a text delta, its item, completed.
const grokStream = `data: {"type":"response.created","response":{"id":"r1","model":"grok-4.6"}}

data: {"type":"response.reasoning_summary_text.delta","delta":"think"}

data: {"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"think"}]}}

data: {"type":"response.output_text.delta","delta":"ok"}

data: {"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"ok"}]}}

data: {"type":"response.completed","response":{"id":"r1","model":"grok-4.6"}}

`

// streamRequest is what a stream test's server received.
type streamRequest struct {
	mu     sync.Mutex
	path   string
	accept string
	body   map[string]any
}

// sseServer records each request and answers with reply.
func sseServer(t *testing.T, reply string) (*httptest.Server, *streamRequest) {
	t.Helper()
	got := &streamRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.mu.Lock()
		got.path, got.accept, got.body = r.URL.Path, r.Header.Get("Accept"), nil
		_ = json.Unmarshal(raw, &got.body)
		got.mu.Unlock()
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

// TestStream (0031-MADR D1): Stream asks /responses for an event stream and
// yields its events as D2 maps them.
func TestStream(t *testing.T) {
	srv, got := sseServer(t, grokStream)
	p := build(t, apiKey(srv.URL, "grok-4.6")...)
	s, ok := p.(llmprovider.Streamer)
	if !ok {
		t.Fatal("the provider does not stream natively")
	}
	var kinds []string
	for ev, err := range s.Stream(context.Background(), text("hi")) {
		if err != nil {
			t.Fatal(err)
		}
		kind := string(ev.Type)
		if ev.Type == llmprovider.EventTextDelta || ev.Type == llmprovider.EventReasoningDelta {
			kind += ":" + ev.Text
		}
		kinds = append(kinds, kind)
	}
	if want := []string{"reasoning_delta:think", "item", "text_delta:ok", "item", "done"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events %v, want %v", kinds, want)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if got.path != "/responses" || got.accept != "text/event-stream" || got.body["stream"] != true {
		t.Errorf("sent %s, Accept %q, stream %v; want /responses, text/event-stream, true", got.path, got.accept, got.body["stream"])
	}
}

// TestGenerate_Unchanged (0031-MADR D5): Generate asks for no stream.
func TestGenerate_Unchanged(t *testing.T) {
	srv, got := sseServer(t, `{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	p := build(t, apiKey(srv.URL, "grok-4.6")...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if _, ok := got.body["stream"]; ok || got.accept == "text/event-stream" {
		t.Errorf("Generate sent stream %v, Accept %q; want neither", got.body["stream"], got.accept)
	}
}
