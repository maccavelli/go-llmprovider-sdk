package gemini

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

// scriptedInteraction is an Interactions event stream in the shape 0031-PLAN-
// native-streaming-on-the-other-wires Phase 1 measured.
const scriptedInteraction = "event: interaction.created\ndata: {\"event_type\":\"interaction.created\",\"interaction\":{\"id\":\"int_1\",\"model\":\"gemini-3.7-flash\",\"status\":\"in_progress\"}}\n\n" +
	"event: step.start\ndata: {\"event_type\":\"step.start\",\"index\":0,\"step\":{\"type\":\"model_output\"}}\n\n" +
	"event: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"ok\"}}\n\n" +
	"event: step.stop\ndata: {\"event_type\":\"step.stop\",\"index\":0}\n\n" +
	"event: interaction.completed\ndata: {\"event_type\":\"interaction.completed\",\"interaction\":{\"id\":\"int_1\",\"model\":\"gemini-3.7-flash\",\"status\":\"completed\"}}\n\n" +
	"event: done\ndata: [DONE]\n\n"

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

// TestStream (0031-MADR E1, E3): Stream asks /interactions for an event
// stream and yields the Interactions decoder's events, the interaction's id
// on EventDone's Response.
func TestStream(t *testing.T) {
	srv, got := sseServer(t, scriptedInteraction)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	s, ok := p.(llmprovider.Streamer)
	if !ok {
		t.Fatal("the provider does not stream natively")
	}
	var kinds []string
	var done *llmprovider.Response
	for ev, err := range s.Stream(context.Background(), text("hi")) {
		if err != nil {
			t.Fatal(err)
		}
		kind := string(ev.Type)
		if ev.Type == llmprovider.EventTextDelta {
			kind += ":" + ev.Text
		}
		if ev.Type == llmprovider.EventDone {
			done = ev.Response
		}
		kinds = append(kinds, kind)
	}
	if want := []string{"text_delta:ok", "item", "done"}; !reflect.DeepEqual(kinds, want) || done == nil || done.ID != "int_1" {
		t.Fatalf("events %v, Response %+v; want %v, id int_1", kinds, done, want)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if got.accept != "text/event-stream" || got.body["stream"] != true {
		t.Errorf("sent Accept %q, stream %v; want text/event-stream and true", got.accept, got.body["stream"])
	}
}

// TestGenerate_Unchanged (0031-MADR E4): Generate asks for no stream.
func TestGenerate_Unchanged(t *testing.T) {
	srv, got := sseServer(t, `{"id":"int_1","model":"m","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"ok"}]}]}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	got.mu.Lock()
	defer got.mu.Unlock()
	if _, ok := got.body["stream"]; ok || got.accept == "text/event-stream" {
		t.Errorf("Generate sent stream %v, Accept %q; want neither", got.body["stream"], got.accept)
	}
}
