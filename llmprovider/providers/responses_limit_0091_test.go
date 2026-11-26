package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0091-MADR ST68 (D3, the coordinator's call): the providers that leave the
// Responses decoder's limit unset, openai, grok and OpenCode's responses
// route (ST33 (built)), read a completed reply whose output_tokens reached
// the limit sent as they did: finishing stop, or tool_calls with a call,
// never length. Only chatcompat's opt-in reads it as cut (ST55).

// atLimitLimit is the limit each request sends, WithMaxTokens' value, and
// the output_tokens each reply reports.
const atLimitLimit = 16

// atLimitRoutes are the three Responses routes the pin covers.
var atLimitRoutes = []struct {
	name  string
	id    llmprovider.ProviderID
	model string
	opts  []llmprovider.Option
}{
	{"openai", llmprovider.ProviderOpenAI, "gpt-5", nil},
	{"grok", llmprovider.ProviderGrok, "grok-4.5", nil},
	{"opencode responses", llmprovider.ProviderOpencodeZen, "gpt-5.5",
		[]llmprovider.Option{opencode.WithRoute(opencode.RouteResponses)}},
}

// atLimitItems are a completed reply's items: a message, and a whole call
// when call is set.
func atLimitItems(call bool) []any {
	items := []any{map[string]any{"type": "message", "id": "msg_1", "role": "assistant", "status": "completed",
		"content": []any{map[string]any{"type": "output_text", "text": "hello"}}}}
	if call {
		items = append(items, map[string]any{"type": "function_call", "id": "fc_1", "call_id": "call_1",
			"name": "get_weather", "status": "completed", "arguments": `{"city":"Paris"}`})
	}
	return items
}

// atLimitServer answers each POST with a completed reply at the limit, as
// JSON or as its event stream, and keeps each body's max_output_tokens.
func atLimitServer(t *testing.T, call bool) (*httptest.Server, func() []any) {
	t.Helper()
	usage := map[string]any{"input_tokens": 3, "output_tokens": atLimitLimit}
	var (
		mu   sync.Mutex
		sent []any
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		sent = append(sent, body["max_output_tokens"])
		mu.Unlock()
		if body["stream"] != true {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, js(map[string]any{"id": "resp_1", "model": "m", "status": "completed",
				"output": atLimitItems(call), "usage": usage}))
			return
		}
		var b strings.Builder
		ev := func(v map[string]any) { b.WriteString("event: " + v["type"].(string) + "\ndata: " + js(v) + "\n\n") }
		ev(map[string]any{"type": "response.created", "response": map[string]any{"id": "resp_1", "model": "m"}})
		for _, item := range atLimitItems(call) {
			ev(map[string]any{"type": "response.output_item.added", "item": item})
			ev(map[string]any{"type": "response.output_item.done", "item": item})
		}
		ev(map[string]any{"type": "response.completed", "response": map[string]any{"id": "resp_1", "model": "m",
			"status": "completed", "usage": usage}})
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, b.String())
	}))
	t.Cleanup(srv.Close)
	return srv, func() []any {
		mu.Lock()
		defer mu.Unlock()
		return append([]any(nil), sent...)
	}
}

// TestResponses_CompletedAtLimitUnchecked_0091 (0091-MADR ST68): on openai,
// grok and OpenCode's responses route, plain and stream, a completed reply
// at the limit sent finishes stop, or tool_calls with a call, as before.
func TestResponses_CompletedAtLimitUnchecked_0091(t *testing.T) {
	for _, route := range atLimitRoutes {
		for _, call := range []bool{false, true} {
			want := llmprovider.FinishStop
			if call {
				want = llmprovider.FinishToolCalls
			}
			for _, stream := range []bool{false, true} {
				name := route.name + "/" + string(want)
				if stream {
					name += "/stream"
				}
				t.Run(name, func(t *testing.T) {
					srv, sent := atLimitServer(t, call)
					resp, err := atLimitRun(t, route.id, route.model, route.opts, srv.URL, stream)
					if got := sent(); len(got) != 1 || got[0] != float64(atLimitLimit) {
						t.Fatalf("max_output_tokens sent %v; want %d once", got, atLimitLimit)
					}
					if err != nil || resp.FinishReason != want || resp.OutputText() != "hello" {
						t.Errorf("%+v, %v; want hello, finishing %s: the decoder's limit is chatcompat's alone", resp, err, want)
					}
				})
			}
		}
	}
}

// atLimitRun sends one request to the provider id built for url, by Generate
// or by Stream, and returns its response or its error.
func atLimitRun(t *testing.T, id llmprovider.ProviderID, model string, opts []llmprovider.Option, url string,
	stream bool) (*llmprovider.Response, error) {
	t.Helper()
	p, err := New(id, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
		llmprovider.WithBaseURL(url), llmprovider.WithModelProbes(false),
		llmprovider.WithMaxTokens(atLimitLimit), testCache(t)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	if !stream {
		return p.Generate(context.Background(), overStreamHi())
	}
	for ev, err := range llmprovider.Stream(context.Background(), p, overStreamHi()) {
		if err != nil {
			return nil, err
		}
		if ev.Type == llmprovider.EventDone {
			return ev.Response, nil
		}
	}
	return nil, errors.New("the stream ended with no done event")
}
