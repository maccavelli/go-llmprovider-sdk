package openai

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

// streamRequest is what a stream test's server received.
type streamRequest struct {
	mu     sync.Mutex
	path   string
	accept string
	body   map[string]any
}

// sseServer records each request and answers with reply as an event stream.
func sseServer(t *testing.T, reply string) (*httptest.Server, *streamRequest) {
	t.Helper()
	got := &streamRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.mu.Lock()
		got.path, got.accept, got.body = r.URL.Path, r.Header.Get("Accept"), nil
		_ = json.Unmarshal(raw, &got.body)
		got.mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

// streamKinds streams req from p and returns each event's type, a delta with
// its text, and the stream's error.
func streamKinds(t *testing.T, p llmprovider.Provider, req *llmprovider.Request) ([]string, error) {
	t.Helper()
	s, ok := p.(llmprovider.Streamer)
	if !ok {
		t.Fatal("the provider does not stream natively")
	}
	var kinds []string
	for ev, err := range s.Stream(context.Background(), req) {
		if err != nil {
			return kinds, err
		}
		switch ev.Type {
		case llmprovider.EventTextDelta, llmprovider.EventReasoningDelta:
			kinds = append(kinds, string(ev.Type)+":"+ev.Text)
		case llmprovider.EventItem:
			kinds = append(kinds, string(ev.Type)+":"+reflect.TypeOf(ev.Item).Name())
		default:
			kinds = append(kinds, string(ev.Type))
		}
	}
	return kinds, nil
}

// TestStream_APIKey (0031-MADR D1): with an API key, Stream asks /responses
// for an event stream and yields its events as D2 maps them.
func TestStream_APIKey(t *testing.T) {
	srv, got := sseServer(t, fixture(t, "chatgpt-text.sse"))
	p := build(t, llmprovider.WithAPIKey("k"), llmprovider.WithModel("gpt-4.1-mini"), llmprovider.WithBaseURL(srv.URL))
	kinds, err := streamKinds(t, p, text("hi"))
	if want := []string{"text_delta:ok", "item:MessageItem", "done"}; err != nil || !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events %v, %v; want %v", kinds, err, want)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if got.path != "/responses" || got.accept != "text/event-stream" || got.body["stream"] != true {
		t.Errorf("sent %s, Accept %q, stream %v; want /responses, text/event-stream, true", got.path, got.accept, got.body["stream"])
	}
}

// TestStream_ChatGPT (0031-MADR D1): a ChatGPT session streams with the
// headers and body Generate sends it.
func TestStream_ChatGPT(t *testing.T) {
	client, c := chatGPTClient(t, fixture(t, "chatgpt-text.sse"))
	p := sessionProvider(t, chatGPTSession(), "gpt-5.4-mini", llmprovider.WithHTTPClient(client))
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	genHeader, genBody := c.header, c.body
	c.mu.Unlock()
	kinds, err := streamKinds(t, p, text("hi"))
	if want := []string{"text_delta:ok", "item:MessageItem", "done"}; err != nil || !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events %v, %v; want %v", kinds, err, want)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !reflect.DeepEqual(c.header, genHeader) || !reflect.DeepEqual(c.body, genBody) {
		t.Errorf("Stream sent %v %v; Generate sent %v %v", c.header, c.body, genHeader, genBody)
	}
}

// TestGenerate_Unchanged (0031-MADR D5): an API key's Generate asks for no
// stream.
func TestGenerate_Unchanged(t *testing.T) {
	srv, got := sseServer(t, okResponse)
	p := build(t, llmprovider.WithAPIKey("k"), llmprovider.WithModel("gpt-4.1-mini"), llmprovider.WithBaseURL(srv.URL))
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if _, ok := got.body["stream"]; ok || got.accept == "text/event-stream" {
		t.Errorf("Generate sent stream %v, Accept %q; want neither", got.body["stream"], got.accept)
	}
}
