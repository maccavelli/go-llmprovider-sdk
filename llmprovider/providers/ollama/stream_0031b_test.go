package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// chatStream is a Chat Completions event stream in the shape 0031-PLAN-
// native-streaming-on-the-other-wires Phase 1 measured: reasoning, text, the
// finish, the usage chunk, then [DONE].
const chatStream = "data: {\"id\":\"c1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"reasoning_content\":\"think\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"c1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"c1\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: {\"id\":\"c1\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":3}}\n\n" +
	"data: [DONE]\n\n"

// streamRequest is what a stream test's server received.
type streamRequest struct {
	mu     sync.Mutex
	path   string
	accept string
	body   map[string]any
}

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

// TestStream (0031-MADR E1, E3, E5): Stream asks the chat completions path
// for an event stream with its usage, and yields the Chat Completions
// decoder's events.
func TestStream(t *testing.T) {
	srv, got := sseServer(t, chatStream)
	p := build(t, llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL))
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
	if want := []string{"reasoning_delta:think", "text_delta:ok", "item", "item", "done"}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events %v, want %v", kinds, want)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	options, _ := got.body["stream_options"].(map[string]any)
	if !strings.HasSuffix(got.path, "/chat/completions") || got.accept != "text/event-stream" || got.body["stream"] != true ||
		options["include_usage"] != true {
		t.Errorf("sent %s, Accept %q, stream %v, stream_options %v; want a stream with its usage", got.path, got.accept,
			got.body["stream"], got.body["stream_options"])
	}
}

// TestGenerate_Unchanged (0031-MADR E4): Generate asks for no stream.
func TestGenerate_Unchanged(t *testing.T) {
	srv, got := sseServer(t, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`)
	p := build(t, llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL))
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	_, stream := got.body["stream"]
	_, options := got.body["stream_options"]
	if stream || options || got.accept == "text/event-stream" {
		t.Errorf("Generate sent stream %v, stream_options %v, Accept %q; want none", got.body["stream"],
			got.body["stream_options"], got.accept)
	}
}
