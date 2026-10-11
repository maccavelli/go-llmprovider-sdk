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

// TestStream_OtherRoutesFallBack (0031-MADR D1, Q2): the routes that do not
// stream yet send one request without "stream", and yield Generate's result
// as events. The messages and chat_completions routes stream since 0031-PLAN-
// native-streaming-on-the-other-wires Phases 3 and 4.
func TestStream_OtherRoutesFallBack(t *testing.T) {
	replies := map[Route]string{
		RouteGoogle: `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"ok"}]}}]}`,
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

// TestStream_MessagesRouteNative (0031-MADR E1, E3): on the messages route
// Stream asks for an event stream and yields the Messages decoder's events.
func TestStream_MessagesRouteNative(t *testing.T) {
	stream := sseEvent(`{"type":"message_start","message":{"model":"qwen3.8-flash","usage":{"input_tokens":9}}}`) +
		sseEvent(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
		sseEvent(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`) +
		sseEvent(`{"type":"content_block_stop","index":0}`) +
		sseEvent(`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`) +
		sseEvent(`{"type":"message_stop"}`)
	for _, gateway := range family {
		t.Run(string(gateway), func(t *testing.T) {
			var mu sync.Mutex
			var accept, path string
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				mu.Lock()
				accept, path = r.Header.Get("Accept"), r.URL.Path
				_ = json.Unmarshal(raw, &body)
				mu.Unlock()
				_, _ = io.WriteString(w, stream)
			}))
			defer srv.Close()
			p := build(t, gateway, apiKey(srv.URL, "some-model", WithRoute(RouteMessages))...)
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
			mu.Lock()
			defer mu.Unlock()
			if path != "/messages" || accept != "text/event-stream" || body["stream"] != true {
				t.Errorf("sent %s, Accept %q, stream %v; want /messages, text/event-stream, true", path, accept, body["stream"])
			}
		})
	}
}

// TestStream_ChatRouteNative (0031-MADR E1, E3, E5): on the chat_completions
// route Stream asks for an event stream with its usage, and yields the Chat
// Completions decoder's events.
func TestStream_ChatRouteNative(t *testing.T) {
	stream := sseEvent(`{"id":"c1","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":null}]}`) +
		sseEvent(`{"id":"c1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`) +
		sseEvent(`{"id":"c1","model":"m","choices":[],"usage":{"prompt_tokens":9,"completion_tokens":2}}`) +
		"data: [DONE]\n\n"
	for _, gateway := range family {
		t.Run(string(gateway), func(t *testing.T) {
			var mu sync.Mutex
			var accept, path string
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				mu.Lock()
				accept, path = r.Header.Get("Accept"), r.URL.Path
				_ = json.Unmarshal(raw, &body)
				mu.Unlock()
				_, _ = io.WriteString(w, stream)
			}))
			defer srv.Close()
			p := build(t, gateway, apiKey(srv.URL, "some-model", WithRoute(RouteChatCompletions))...)
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
			mu.Lock()
			defer mu.Unlock()
			options, _ := body["stream_options"].(map[string]any)
			if path != "/chat/completions" || accept != "text/event-stream" || body["stream"] != true || options["include_usage"] != true {
				t.Errorf("sent %s, Accept %q, stream %v, stream_options %v; want a stream with its usage", path, accept,
					body["stream"], body["stream_options"])
			}
		})
	}
}
