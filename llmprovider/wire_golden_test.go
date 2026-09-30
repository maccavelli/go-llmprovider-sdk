package llmprovider

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
)

// updateWire rewrites the G-wire goldens. They are recorded once, at 0002-PLAN
// Phase 7, and regenerated only for a difference a record explains
// (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireSession pins the conversation id, so the goldens show every place it is
// sent: prompt_cache_key, session-id, x-opencode-session and Kilo's task id.
const wireSession = "wire-session"

const wirePrompt = "What is the weather in Paris?"

// wireItems is a conversation using every Item type, a system message and a
// replayed call signature.
var wireItems = []Item{
	MessageItem{Role: jsonRoleSystem, Text: "Be brief."},
	MessageItem{Role: jsonRoleUser, Text: wirePrompt},
	ReasoningItem{Text: "The user wants the weather."},
	FunctionCallItem{CallID: "call_1", Name: "get_weather", Arguments: `{"city":"Paris"}`, Signature: "sig-1"},
	FunctionCallOutputItem{CallID: "call_1", Output: `{"sky":"sunny"}`},
	MessageItem{Role: jsonRoleAssistant, Text: "It is sunny."},
	MessageItem{Role: jsonRoleUser, Text: "And tomorrow?"},
}

// wireProvider is what every provider type implements at 0002-PLAN Phase 7.
type wireProvider interface {
	ThinkingToolProvider
	ThinkingProvider
	ItemProvider
	ModelDiscoverer
}

// wireCase is one provider, or one route of a gateway, and the canned replies
// its wire needs.
type wireCase struct {
	name    string
	build   func(baseURL string, extra ...ProviderOption) (wireProvider, error)
	listing string // body for GET .../models or /api/tags
	sse     bool   // /responses answers with an event stream (the ChatGPT backend)
	// continueOpts are added for the continuation scenario: Gemini continues
	// only stored interactions.
	continueOpts []ProviderOption
}

func wireOpts(baseURL string, extra ...ProviderOption) []ProviderOption {
	return append([]ProviderOption{WithBaseURL(baseURL), WithSessionID(wireSession)}, extra...)
}

func opencodeWireCase(gateway, route, model, listing string) wireCase {
	return wireCase{
		name:    gateway + "-" + route,
		listing: listing,
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewOpencode(gateway, "opencode-wire-key", model, wireOpts(u, extra...)...)
		},
	}
}

var (
	wireListingDataIDs = func(ids ...string) string {
		entries := make([]string, len(ids))
		for i, id := range ids {
			entries[i] = fmt.Sprintf(`{"id":%q,"object":"model"}`, id)
		}
		return `{"object":"list","data":[` + strings.Join(entries, ",") + `]}`
	}
	wireOpencodeListing = wireListingDataIDs("gpt-5.5", "claude-sonnet-5", "gemini-3.8-flash", "glm-5.3")
)

var wireCases = []wireCase{
	{
		name:    "claude",
		listing: `{"data":[{"id":"claude-sonnet-5","type":"model"},{"id":"claude-haiku-4-5","type":"model"}],"has_more":false}`,
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewClaude("sk-ant-wire", "claude-sonnet-5", wireOpts(u, extra...)...)
		},
	},
	{
		name: "gemini",
		listing: `{"models":[{"name":"models/gemini-3.7-flash","supportedGenerationMethods":["generateContent"]},` +
			`{"name":"models/text-embedding-005","supportedGenerationMethods":["embedContent"]}]}`,
		continueOpts: []ProviderOption{WithStore(true)},
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewGemini(context.Background(), "gemini-wire-key", "gemini-3.7-flash", wireOpts(u, extra...)...)
		},
	},
	{
		name:    "grok",
		listing: wireListingDataIDs("grok-4.5", "grok-4.7"),
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewGrok("xai-wire", "grok-4.5", wireOpts(u, extra...)...)
		},
	},
	opencodeWireCase(ProviderOpencodeZen, "responses", "gpt-5.5", wireOpencodeListing),
	opencodeWireCase(ProviderOpencodeZen, "messages", "claude-sonnet-5", wireOpencodeListing),
	opencodeWireCase(ProviderOpencodeZen, "google", "gemini-3.8-flash", wireOpencodeListing),
	opencodeWireCase(ProviderOpencodeZen, "chat", "glm-5.3", wireOpencodeListing),
	opencodeWireCase(ProviderOpencodeGo, "responses", "grok-4.7", wireListingDataIDs("grok-4.7", "qwen3.8-flash", "kimi-k3")),
	opencodeWireCase(ProviderOpencodeGo, "messages", "qwen3.8-flash", wireListingDataIDs("grok-4.7", "qwen3.8-flash", "kimi-k3")),
	opencodeWireCase(ProviderOpencodeGo, "chat", "kimi-k3", wireListingDataIDs("grok-4.7", "qwen3.8-flash", "kimi-k3")),
	{
		name: "kilo",
		listing: `{"data":[{"id":"anthropic/claude-sonnet-5","name":"Claude Sonnet 5","created":1780000000,` +
			`"architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
			`"pricing":{"prompt":"0.000003","completion":"0.000015"},"context_length":200000,` +
			`"supported_parameters":["tools","tool_choice","reasoning","reasoning_effort"]}]}`,
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewKilo("kilo-wire-key", "anthropic/claude-sonnet-5", wireOpts(u, extra...)...)
		},
	},
	{
		name: "huggingface",
		listing: `{"data":[{"id":"openai/gpt-oss-120b","architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
			`"providers":[{"status":"live","supports_tools":true,"throughput":100,"first_token_latency_ms":300}]}]}`,
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewHuggingFace("hf_wire", "openai/gpt-oss-120b", wireOpts(u, extra...)...)
		},
	},
	{
		// Added by 0017-PLAN U1 after the P7 recording: its goldens are new
		// files, never a change to another case's.
		name: "together",
		listing: `[{"id":"openai/gpt-oss-120b","object":"model","type":"chat","context_length":131072},` +
			`{"id":"BAAI/bge-large-en-v1.5","object":"model","type":"embedding"},` +
			`{"id":"zai-org/GLM-5.3","object":"model","type":"chat","context_length":202752}]`,
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewTogether("together-wire-key", "openai/gpt-oss-120b", wireOpts(u, extra...)...)
		},
	},
	{
		name:    "ollama",
		listing: `{"models":[{"name":"llama3.3:latest"},{"name":"qwen3:8b"}]}`,
		build: func(u string, extra ...ProviderOption) (wireProvider, error) {
			return NewOllama("", "llama3.3", wireOpts(u, extra...)...)
		},
	},
}

// Canned replies. Each carries reasoning, text ("hello", which the listing
// probes look for) and a get_weather call, so every scenario decodes from one
// reply per wire.
const (
	wireResponsesReply = `{"id":"resp_wire","status":"completed","output":[` +
		`{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking"}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]},` +
		`{"type":"function_call","call_id":"call_wire","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}]}`
	wireMessagesReply = `{"id":"msg_wire","type":"message","role":"assistant","stop_reason":"tool_use","content":[` +
		`{"type":"thinking","thinking":"thinking","signature":"sig_wire"},` +
		`{"type":"text","text":"hello"},` +
		`{"type":"tool_use","id":"toolu_wire","name":"get_weather","input":{"city":"Paris"}}]}`
	wireInteractionsReply = `{"id":"v1_int_wire","status":"completed","steps":[` +
		`{"type":"thought","signature":"sig_wire","summary":[{"type":"text","text":"thinking"}]},` +
		`{"type":"model_output","content":[{"type":"text","text":"hello"}]},` +
		`{"type":"function_call","id":"call_wire","name":"get_weather","arguments":{"city":"Paris"}}]}`
	wireGoogleReply = `{"candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[` +
		`{"text":"thinking","thought":true},{"text":"hello"},` +
		`{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"sig_wire"}]}}]}`
	wireChatReply = `{"id":"chat_wire","choices":[{"index":0,"finish_reason":"tool_calls","message":{` +
		`"role":"assistant","content":"hello","reasoning_content":"thinking",` +
		`"tool_calls":[{"id":"call_wire","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]}}]}`
)

// wireSSEReply is wireResponsesReply as the ChatGPT backend streams it.
var wireSSEReply = strings.Join([]string{
	`event: response.created`,
	`data: {"type":"response.created","response":{"id":"resp_wire"}}`,
	``,
	`event: response.output_item.done`,
	`data: {"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking"}]}}`,
	``,
	`event: response.output_item.done`,
	`data: {"type":"response.output_item.done","item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}}`,
	``,
	`event: response.output_item.done`,
	`data: {"type":"response.output_item.done","item":{"type":"function_call","call_id":"call_wire","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}`,
	``,
	`event: response.completed`,
	`data: {"type":"response.completed","response":{"id":"resp_wire"}}`,
	``, ``,
}, "\n")

func (c wireCase) reply(r *http.Request) wiretest.Reply {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && (strings.HasSuffix(path, "/models") || path == "/api/tags"):
		return wiretest.Reply{Header: map[string]string{"Content-Type": "application/json"}, Body: c.listing}
	case path == "/api/version":
		return wiretest.Reply{Body: `{"version":"0.12.0"}`}
	case strings.HasSuffix(path, "/responses") && c.sse:
		return wiretest.Reply{Body: wireSSEReply}
	case strings.HasSuffix(path, "/responses"):
		return wiretest.Reply{Body: wireResponsesReply}
	case strings.HasSuffix(path, "/messages"):
		return wiretest.Reply{Body: wireMessagesReply}
	case strings.HasSuffix(path, "/interactions"):
		return wiretest.Reply{Body: wireInteractionsReply}
	case strings.HasSuffix(path, ":generateContent"):
		return wiretest.Reply{Body: wireGoogleReply}
	case strings.HasSuffix(path, "/chat/completions"):
		return wiretest.Reply{Body: wireChatReply}
	}
	return wiretest.Reply{Status: http.StatusNotFound, Body: `{"error":{"message":"no canned reply for this path"}}`}
}

// wireScenarios are 0015-PLAN S2's scenarios, through the API as it stands at
// 0002-PLAN Phase 7.
var wireScenarios = []struct {
	name string
	run  func(ctx context.Context, p wireProvider) (any, error)
}{
	{"text", func(ctx context.Context, p wireProvider) (any, error) { return p.Generate(ctx, wirePrompt) }},
	{"tool", func(ctx context.Context, p wireProvider) (any, error) {
		return p.GenerateWithTool(ctx, wirePrompt, weatherTool)
	}},
	{"thinking", func(ctx context.Context, p wireProvider) (any, error) { return p.GenerateThinking(ctx, wirePrompt) }},
	{"thinking-tool", func(ctx context.Context, p wireProvider) (any, error) {
		return p.GenerateWithToolThinking(ctx, wirePrompt, weatherTool)
	}},
	{"items", func(ctx context.Context, p wireProvider) (any, error) { return p.GenerateItems(ctx, wireItems...) }},
	{"continuation", func(ctx context.Context, p wireProvider) (any, error) {
		c, ok := p.(Continuer)
		if !ok {
			return nil, errWireUnsupported
		}
		return c.Continue(ctx, "resp_previous", MessageItem{Role: jsonRoleUser, Text: "And tomorrow?"})
	}},
	{"listing", func(ctx context.Context, p wireProvider) (any, error) { return p.DiscoverModels(ctx) }},
}

// errWireUnsupported marks a scenario the provider has no method for; it
// writes no golden.
var errWireUnsupported = errors.New("scenario not supported")

// wireResult makes a provider's return value comparable across the S7 API
// change: a Response's items are tagged with their type.
func wireResult(v any) any {
	resp, ok := v.(*Response)
	if !ok || resp == nil {
		return v
	}
	items := make([]map[string]any, len(resp.Output))
	for i, item := range resp.Output {
		items[i] = map[string]any{"type": strings.TrimPrefix(fmt.Sprintf("%T", item), "llmprovider."), "item": item}
	}
	return map[string]any{"id": resp.ID, "finish_reason": resp.FinishReason, "output": items}
}

// TestWireGoldens is G-wire (0015-MADR D12): each scenario's requests, as
// normalised JSON, match testdata/wire/<provider>/<scenario>.json.
func TestWireGoldens(t *testing.T) {
	for _, c := range wireCases {
		for _, s := range wireScenarios {
			t.Run(c.name+"/"+s.name, func(t *testing.T) {
				srv := wiretest.NewServer(t, c.reply)
				var extra []ProviderOption
				if s.name == "continuation" {
					extra = c.continueOpts
				}
				p, err := c.build(srv.URL, extra...)
				if err != nil {
					t.Fatal(err)
				}
				result, err := s.run(context.Background(), p)
				if errors.Is(err, errWireUnsupported) {
					t.Skip("no such method on this provider")
				}
				rec := wiretest.Record{Requests: srv.Requests(), Result: wireResult(result)}
				if err != nil {
					rec.Error = err.Error()
				}
				wiretest.Check(t, filepath.Join("testdata", "wire", c.name, s.name+".json"), rec, *updateWire)
			})
		}
	}
}
