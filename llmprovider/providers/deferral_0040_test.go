package providers

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
)

// 0040-MADR D1's table, D6 and D7 across the built-in providers: which tools
// each sends for a request with loaded deferred tools, in what order, and
// whether its response carries WarningDeferralEmulated. 0040-PLAN Phase 4's
// TestAppend_Order_0040 and TestWarning_Emulated_0040 are
// TestDeferral_PerProvider_0040's two checks. Grok's dispatch is Phase 5's.

// deferralTools declares three deferred tools: get_weather, lookup_stock and
// send_email, in that order.
func deferralTools() []llmprovider.Tool {
	return []llmprovider.Tool{
		{Name: "read_file"},
		{Name: "tool_search", Search: true},
		{Name: "get_weather", Deferred: true},
		{Name: "lookup_stock", Deferred: true},
		{Name: "send_email", Deferred: true},
	}
}

// deferralRequest loads lookup_stock, then get_weather: the reverse of their
// declared order. send_email is never loaded.
func deferralRequest() *llmprovider.Request {
	return &llmprovider.Request{
		Tools: deferralTools(),
		Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "stock, then weather in Paris"},
			llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"stock weather"}`},
			llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded lookup_stock and get_weather."},
			llmprovider.ToolLoadItem{CallID: "c1", Name: "lookup_stock"},
			llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
		},
	}
}

// sentTools is the tool names a body sends, whatever its wire: a tool's name,
// a Chat Completions function's, a Gemini declaration's, or a tool_search
// entry's type.
func sentTools(body map[string]any) []string {
	var out []string
	list, _ := body["tools"].([]any)
	for _, raw := range list {
		tool, _ := raw.(map[string]any)
		switch {
		case tool["name"] != nil:
			out = append(out, tool["name"].(string))
		case tool["function"] != nil:
			out = append(out, tool["function"].(map[string]any)["name"].(string))
		case tool["functionDeclarations"] != nil:
			for _, d := range tool["functionDeclarations"].([]any) {
				out = append(out, d.(map[string]any)["name"].(string))
			}
		default:
			out = append(out, tool["type"].(string))
		}
	}
	return out
}

// The smallest complete stream on each wire: one text delta and its end.
const (
	messagesStream = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"m\"}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	chatStream = "data: {\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hello\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	interactionsStream = "event: interaction.created\ndata: {\"event_type\":\"interaction.created\",\"interaction\":{\"id\":\"v1_i\",\"model\":\"m\",\"status\":\"in_progress\"}}\n\n" +
		"event: step.start\ndata: {\"event_type\":\"step.start\",\"index\":0,\"step\":{\"type\":\"model_output\"}}\n\n" +
		"event: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"hello\"}}\n\n" +
		"event: step.stop\ndata: {\"event_type\":\"step.stop\",\"index\":0}\n\n" +
		"event: interaction.completed\ndata: {\"event_type\":\"interaction.completed\",\"interaction\":{\"id\":\"v1_i\",\"model\":\"m\",\"status\":\"completed\"}}\n\n" +
		"event: done\ndata: [DONE]\n\n"
	googleStream = "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"hello\"}]},\"finishReason\":\"STOP\",\"index\":0}],\"modelVersion\":\"m\"}\n\n"
)

// streamReply answers a POST with the stream its path's wire sends, and
// anything else as wirecase does. A reply cannot read the request's body, so
// a stream is served by a server of its own.
func streamReply(r *http.Request) wiretest.Reply {
	if r.Method != http.MethodPost {
		return wirecase.Case{}.Reply(r)
	}
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/responses"):
		return wiretest.Reply{Body: wirecase.SSEReply}
	case strings.HasSuffix(path, "/messages"):
		return wiretest.Reply{Body: messagesStream}
	case strings.HasSuffix(path, "/interactions"):
		return wiretest.Reply{Body: interactionsStream}
	case strings.HasSuffix(path, ":streamGenerateContent"):
		return wiretest.Reply{Body: googleStream}
	case strings.HasSuffix(path, "/chat/completions"):
		return wiretest.Reply{Body: chatStream}
	}
	return wiretest.Reply{Status: http.StatusNotFound, Body: `{"error":{"message":"no stream for this path"}}`}
}

// lastBody is the last POST body srv recorded.
func lastBody(srv *wiretest.Server) map[string]any {
	var body map[string]any
	for _, r := range srv.Requests() {
		if b, ok := r.Body.(map[string]any); ok && r.Method == http.MethodPost {
			body = b
		}
	}
	return body
}

// conversation is the part of a body that carries the turns, whatever its
// wire: messages, input or contents.
func conversation(body map[string]any) any {
	for _, key := range []string{"messages", "input", "contents"} {
		if v, ok := body[key]; ok {
			return v
		}
	}
	return nil
}

func emulated(ws []llmprovider.Warning) bool {
	for _, w := range ws {
		if w.Kind == llmprovider.WarningDeferralEmulated {
			return true
		}
	}
	return false
}

// TestDeferral_PerProvider_0040 sends the request to each provider and model,
// by Generate and by Stream, and checks the tools sent and the warning.
func TestDeferral_PerProvider_0040(t *testing.T) {
	// The loaded tools follow the others, in load order; the unloaded one is
	// not sent (D6).
	appended := []string{"read_file", "tool_search", "lookup_stock", "get_weather"}
	every := []string{"read_file", "tool_search", "get_weather", "lookup_stock", "send_email"}
	for _, tc := range []struct {
		id    llmprovider.ProviderID
		model string
		tools []string
		warn  bool
	}{
		{llmprovider.ProviderClaude, "claude-haiku-5-5", every, false},
		{llmprovider.ProviderClaude, "claude-opus-4-1", appended, true},
		{llmprovider.ProviderOpenAI, "gpt-6-luna", []string{"read_file", "tool_search"}, false},
		{llmprovider.ProviderOpenAI, "gpt-4.1-mini", appended, true},
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5", every, false},
		{llmprovider.ProviderOpencodeZen, "gpt-5.5", []string{"read_file", "tool_search"}, false},
		{llmprovider.ProviderOpencodeZen, "glm-5.3", appended, true},
		{llmprovider.ProviderOpencodeZen, "gemini-3.8-flash", appended, true},
		{llmprovider.ProviderKilo, "openai/gpt-6-luna", appended, true},
		{llmprovider.ProviderGemini, "gemini-3.8-flash", appended, true},
		{llmprovider.ProviderHuggingFace, "deepseek-ai/DeepSeek-V4.1-Flash", appended, true},
		{llmprovider.ProviderTogether, "openai/gpt-oss-120b", appended, true},
		{llmprovider.ProviderOllama, "llama3.3:latest", appended, true},
		// The wrapper, whatever is loaded (D5).
		{llmprovider.ProviderGrok, "grok-4.6", []string{"read_file", "tool_search", "use_tool"}, true},
	} {
		t.Run(string(tc.id)+"/"+tc.model, func(t *testing.T) {
			provider := func(srv *wiretest.Server) llmprovider.Provider {
				p, err := New(tc.id, llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model), llmprovider.WithBaseURL(srv.URL),
					llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false))
				if err != nil {
					t.Fatal(err)
				}
				return p
			}
			srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
			resp, err := provider(srv).Generate(context.Background(), deferralRequest())
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := sentTools(lastBody(srv)); !equalNames(got, tc.tools) {
				t.Errorf("Generate: tools %v, want %v", got, tc.tools)
			}
			if got := emulated(resp.Warnings); got != tc.warn {
				t.Errorf("Generate: WarningDeferralEmulated %t, want %t (warnings %+v)", got, tc.warn, resp.Warnings)
			}
			if tc.warn && tc.id != llmprovider.ProviderGrok {
				// Appended, a ToolLoadItem sends nothing: the conversation is
				// the one sent with the loads taken out (A5). Dispatched, the
				// loads' definitions ride in the result (D5).
				bare := wiretest.NewServer(t, wirecase.Case{}.Reply)
				req := deferralRequest()
				req.Input = slices.DeleteFunc(req.Input, func(item llmprovider.Item) bool {
					_, load := item.(llmprovider.ToolLoadItem)
					return load
				})
				if _, err := provider(bare).Generate(context.Background(), req); err != nil {
					t.Fatalf("Generate without the loads: %v", err)
				}
				if got, want := conversation(lastBody(srv)), conversation(lastBody(bare)); got == nil || !reflect.DeepEqual(got, want) {
					t.Errorf("the loads sent something: %v, want %v", got, want)
				}
			}
			streamed := wiretest.NewServer(t, streamReply)
			var done *llmprovider.Response
			for ev, err := range llmprovider.Stream(context.Background(), provider(streamed), deferralRequest()) {
				if err != nil {
					t.Fatalf("Stream: %v", err)
				}
				if ev.Type == llmprovider.EventDone {
					done = ev.Response
				}
			}
			if got := sentTools(lastBody(streamed)); !equalNames(got, tc.tools) {
				t.Errorf("Stream: tools %v, want %v", got, tc.tools)
			}
			if done == nil || emulated(done.Warnings) != tc.warn {
				t.Errorf("Stream: done %+v; want WarningDeferralEmulated %t", done, tc.warn)
			}
		})
	}
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAppend_PreviousResponseID_0040: a continuation's earlier loads are held
// by the service, not in Input, so the append form sends every deferred tool
// and says so; the native form sends none at the top level, and warns of
// nothing (D3, D6, D7).
func TestAppend_PreviousResponseID_0040(t *testing.T) {
	every := []string{"read_file", "tool_search", "get_weather", "lookup_stock", "send_email"}
	for _, tc := range []struct {
		id    llmprovider.ProviderID
		model string
		opts  []llmprovider.Option
		tools []string
		warn  string
	}{
		{llmprovider.ProviderOpenAI, "gpt-4.1-mini", nil, every, "every deferred tool"},
		{llmprovider.ProviderOpenAI, "gpt-6-luna", nil, []string{"read_file", "tool_search"}, ""},
		{llmprovider.ProviderGemini, "gemini-3.8-flash", []llmprovider.Option{gemini.WithStore(true)}, every, "every deferred tool"},
		// The wrapper's list needs no history (D5).
		{llmprovider.ProviderGrok, "grok-4.6", nil, []string{"read_file", "tool_search", "use_tool"}, "use_tool"},
	} {
		srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
		p, err := New(tc.id, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model),
			llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false)}, tc.opts...)...)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := p.Generate(context.Background(), &llmprovider.Request{Tools: deferralTools(), PreviousResponseID: "resp_1",
			Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "and tomorrow?"}}})
		if err != nil {
			t.Fatalf("%s/%s: %v", tc.id, tc.model, err)
		}
		if got := sentTools(lastBody(srv)); !equalNames(got, tc.tools) {
			t.Errorf("%s/%s: tools %v, want %v", tc.id, tc.model, got, tc.tools)
		}
		var detail string
		for _, w := range resp.Warnings {
			if w.Kind == llmprovider.WarningDeferralEmulated {
				detail = w.Detail
			}
		}
		if tc.warn == "" && detail != "" || tc.warn != "" && !strings.Contains(detail, tc.warn) {
			t.Errorf("%s/%s: warning %q, want one containing %q", tc.id, tc.model, detail, tc.warn)
		}
	}
}

// TestDeferral_NoWarningWithoutDeferral_0040: a request that defers nothing
// earns no warning anywhere.
func TestDeferral_NoWarningWithoutDeferral_0040(t *testing.T) {
	for _, tc := range []struct {
		id    llmprovider.ProviderID
		model string
	}{
		{llmprovider.ProviderClaude, "claude-opus-4-1"}, {llmprovider.ProviderGemini, "gemini-3.8-flash"},
		{llmprovider.ProviderTogether, "openai/gpt-oss-120b"}, {llmprovider.ProviderOpenAI, "gpt-4.1-mini"},
	} {
		srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
		p, err := New(tc.id, llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model), llmprovider.WithBaseURL(srv.URL),
			llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := p.Generate(context.Background(), &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "read_file"}},
			Input: []llmprovider.Item{llmprovider.MessageItem{Text: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		if emulated(resp.Warnings) {
			t.Errorf("%s/%s: a warning with nothing deferred: %+v", tc.id, tc.model, resp.Warnings)
		}
	}
}
