package claude

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

// messagesStream is a Messages event stream in the shape 0031-PLAN-native-
// streaming-on-the-other-wires Phase 1 measured: a thinking block, a text
// block, then the stop.
const messagesStream = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-haiku-4-5\",\"usage\":{\"input_tokens\":9,\"output_tokens\":1}}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"thinking\",\"thinking\":\"\",\"signature\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"think\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"signature_delta\",\"signature\":\"sig\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":1}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":4}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

// streamRequest is what a stream test's server received.
type streamRequest struct {
	mu     sync.Mutex
	accept string
	body   map[string]any
}

func sseServer(t *testing.T, reply string) (*httptest.Server, *streamRequest) {
	t.Helper()
	got := &streamRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.mu.Lock()
		got.accept, got.body = r.Header.Get("Accept"), nil
		_ = json.Unmarshal(raw, &got.body)
		got.mu.Unlock()
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

// TestStream (0031-MADR E1, E3): Stream asks /messages for an event stream
// and yields its events as the Messages decoder maps them.
func TestStream(t *testing.T) {
	srv, got := sseServer(t, messagesStream)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
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
	if got.accept != "text/event-stream" || got.body["stream"] != true {
		t.Errorf("sent Accept %q, stream %v; want text/event-stream and true", got.accept, got.body["stream"])
	}
}

// TestGenerate_Unchanged (0031-MADR E4): Generate asks for no stream.
func TestGenerate_Unchanged(t *testing.T) {
	srv, got := sseServer(t, `{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}]}`)
	p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if _, ok := got.body["stream"]; ok || got.accept == "text/event-stream" {
		t.Errorf("Generate sent stream %v, Accept %q; want neither", got.body["stream"], got.accept)
	}
}
