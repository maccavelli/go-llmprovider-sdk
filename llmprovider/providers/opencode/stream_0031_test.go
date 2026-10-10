package opencode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// sseEvent frames one Responses event.
func sseEvent(payload string) string { return "data: " + payload + "\n\n" }

// streamKind is an event's type, with a delta's text.
func streamKind(ev llmprovider.Event) string {
	if ev.Type == llmprovider.EventTextDelta || ev.Type == llmprovider.EventReasoningDelta {
		return string(ev.Type) + ":" + ev.Text
	}
	return string(ev.Type)
}

// streamer is p as a Streamer, failing the test when it is not one.
func streamer(t *testing.T, p llmprovider.Provider) llmprovider.Streamer {
	t.Helper()
	s, ok := p.(llmprovider.Streamer)
	if !ok {
		t.Fatal("the provider does not implement Streamer")
	}
	return s
}

// TestStream_ResponsesRouteNative (0031-MADR D1): on the responses route a
// delta reaches the caller while the reply is still open. The server holds
// the stream after its first delta until the test has read it.
func TestStream_ResponsesRouteNative(t *testing.T) {
	for _, gateway := range family {
		t.Run(string(gateway), func(t *testing.T) {
			release := make(chan struct{})
			var mu sync.Mutex
			var accept string
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				mu.Lock()
				accept = r.Header.Get("Accept")
				_ = json.Unmarshal(raw, &body)
				mu.Unlock()
				_, _ = io.WriteString(w, sseEvent(`{"type":"response.output_text.delta","delta":"ok"}`))
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-time.After(5 * time.Second):
					return // the delta never reached the caller before the end
				}
				_, _ = io.WriteString(w, sseEvent(`{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"ok"}]}}`)+
					sseEvent(`{"type":"response.completed","response":{"id":"r1"}}`))
			}))
			defer srv.Close()
			p := build(t, gateway, apiKey(srv.URL, "some-model", WithRoute(RouteResponses))...)
			var kinds []string
			for ev, err := range streamer(t, p).Stream(context.Background(), text("hi")) {
				if err != nil {
					t.Fatalf("after %v: %v", kinds, err)
				}
				if len(kinds) == 0 {
					close(release)
				}
				kinds = append(kinds, streamKind(ev))
			}
			if want := []string{"text_delta:ok", "item", "done"}; !reflect.DeepEqual(kinds, want) {
				t.Fatalf("events %v, want %v", kinds, want)
			}
			mu.Lock()
			defer mu.Unlock()
			if accept != "text/event-stream" || body["stream"] != true {
				t.Errorf("sent Accept %q, stream %v; want text/event-stream and true", accept, body["stream"])
			}
		})
	}
}

// TestStream_OtherRoutesFallBack (0031-MADR D1, Q2): the messages,
// chat_completions and google routes do not stream yet: Stream sends one
// request without "stream", and yields Generate's result as events.
func TestStream_OtherRoutesFallBack(t *testing.T) {
	replies := map[Route]string{
		RouteMessages:        `{"stop_reason":"end_turn","content":[{"type":"text","text":"ok"}]}`,
		RouteChatCompletions: `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`,
		RouteGoogle:          `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"ok"}]}}]}`,
	}
	for _, gateway := range family {
		for route, reply := range replies {
			t.Run(string(gateway)+"/"+string(route), func(t *testing.T) {
				var body map[string]any
				requests := 0
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					raw, _ := io.ReadAll(r.Body)
					_ = json.Unmarshal(raw, &body)
					_, _ = io.WriteString(w, reply)
				}))
				defer srv.Close()
				p := build(t, gateway, apiKey(srv.URL, "some-model", WithRoute(route))...)
				var kinds []string
				for ev, err := range streamer(t, p).Stream(context.Background(), text("hi")) {
					if err != nil {
						t.Fatal(err)
					}
					kinds = append(kinds, streamKind(ev))
				}
				if want := []string{"text_delta:ok", "item", "done"}; !reflect.DeepEqual(kinds, want) {
					t.Fatalf("events %v, want %v", kinds, want)
				}
				if _, ok := body["stream"]; ok || requests != 1 {
					t.Errorf("%d requests, stream %v; want one, without stream", requests, body["stream"])
				}
			})
		}
	}
}

// TestGenerate_Unchanged (0031-MADR D5): Generate on the responses route asks
// for no stream.
func TestGenerate_Unchanged(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "some-model", WithRoute(RouteResponses))...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["stream"]; ok {
		t.Errorf("Generate sent stream %v; want none", body["stream"])
	}
}
