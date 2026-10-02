// Package wirecase is G-wire's scenarios through the new API, shared by the
// provider packages' tests (0015-MADR D12; 0015-PLAN S2, S7). Each scenario
// sends what its 0002-PLAN Phase 7 counterpart sent, and maps the new API's
// result back to that counterpart's, so the P7 goldens still compare. It is
// imported only by tests.
package wirecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const (
	jsonType     = "type"
	sseItemEvent = `event: response.output_item.done`
)

// Session pins the conversation id, so the goldens show every place it is
// sent.
const Session = "wire-session"

// Prompt is the text scenarios' user message.
const Prompt = "What is the weather in Paris?"

// user is a user message.
func user(text string) llmprovider.MessageItem {
	return llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: text}
}

// Items is a conversation using every Item type, a system message and a
// replayed call signature.
func Items() []llmprovider.Item {
	return []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."},
		user(Prompt),
		llmprovider.ReasoningItem{Text: "The user wants the weather."},
		llmprovider.FunctionCallItem{CallID: "call_1", Name: "get_weather", Arguments: `{"city":"Paris"}`, Signature: "sig-1"},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: `{"sky":"sunny"}`},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "It is sunny."},
		user("And tomorrow?"),
	}
}

// WeatherTool is the tool the tool scenarios offer.
func WeatherTool() llmprovider.Tool {
	return llmprovider.Tool{Name: "get_weather", Description: "Weather for a city",
		Schema: map[string]any{jsonType: "object", "properties": map[string]any{"city": map[string]any{jsonType: "string"}}}}
}

// Opts are the options every case is built with.
func Opts(baseURL string, extra ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithBaseURL(baseURL), llmprovider.WithSessionID(Session)}, extra...)
}

// ListingDataIDs is an OpenAI-style {"data":[{"id":…}]} listing.
func ListingDataIDs(ids ...string) string {
	entries := make([]string, len(ids))
	for i, id := range ids {
		entries[i] = fmt.Sprintf(`{"id":%q,"object":"model"}`, id)
	}
	return `{"object":"list","data":[` + strings.Join(entries, ",") + `]}`
}

// Canned replies. Each carries reasoning, text ("hello", which the listing
// probes look for) and a get_weather call, so every scenario decodes from one
// reply per wire.
const (
	ResponsesReply = `{"id":"resp_wire","status":"completed","output":[` +
		`{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking"}]},` +
		`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]},` +
		`{"type":"function_call","call_id":"call_wire","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}]}`
	MessagesReply = `{"id":"msg_wire","type":"message","role":"assistant","stop_reason":"tool_use","content":[` +
		`{"type":"thinking","thinking":"thinking","signature":"sig_wire"},` +
		`{"type":"text","text":"hello"},` +
		`{"type":"tool_use","id":"toolu_wire","name":"get_weather","input":{"city":"Paris"}}]}`
	InteractionsReply = `{"id":"v1_int_wire","status":"completed","steps":[` +
		`{"type":"thought","signature":"sig_wire","summary":[{"type":"text","text":"thinking"}]},` +
		`{"type":"model_output","content":[{"type":"text","text":"hello"}]},` +
		`{"type":"function_call","id":"call_wire","name":"get_weather","arguments":{"city":"Paris"}}]}`
	GoogleReply = `{"candidates":[{"finishReason":"STOP","content":{"role":"model","parts":[` +
		`{"text":"thinking","thought":true},{"text":"hello"},` +
		`{"functionCall":{"name":"get_weather","args":{"city":"Paris"}},"thoughtSignature":"sig_wire"}]}}]}`
	ChatReply = `{"id":"chat_wire","choices":[{"index":0,"finish_reason":"tool_calls","message":{` +
		`"role":"assistant","content":"hello","reasoning_content":"thinking",` +
		`"tool_calls":[{"id":"call_wire","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]}}]}`
)

// SSEReply is ResponsesReply as the ChatGPT backend streams it.
var SSEReply = strings.Join([]string{
	`event: response.created`,
	`data: {"type":"response.created","response":{"id":"resp_wire"}}`,
	``,
	sseItemEvent,
	`data: {"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking"}]}}`,
	``,
	sseItemEvent,
	`data: {"type":"response.output_item.done","item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}}`,
	``,
	sseItemEvent,
	`data: {"type":"response.output_item.done","item":{"type":"function_call","call_id":"call_wire","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}`,
	``,
	`event: response.completed`,
	`data: {"type":"response.completed","response":{"id":"resp_wire"}}`,
	``, ``,
}, "\n")

// Case is one provider, or one route of a gateway, and the canned replies
// its wire needs.
type Case struct {
	Name    string
	Build   func(baseURL string, extra ...llmprovider.Option) (llmprovider.Provider, error)
	Listing string // body for GET .../models or /api/tags
	SSE     bool   // /responses answers with an event stream (the ChatGPT backend)
	// ContinueOpts are added for the continuation scenario.
	ContinueOpts []llmprovider.Option
	// NoContinuation marks a provider whose P7 type had no Continue, and so
	// no continuation golden.
	NoContinuation bool
}

// Reply is the canned reply for r.
func (c Case) Reply(r *http.Request) wiretest.Reply {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && (strings.HasSuffix(path, "/models") || path == "/api/tags"):
		return wiretest.Reply{Header: map[string]string{"Content-Type": "application/json"}, Body: c.Listing}
	case path == "/api/version":
		return wiretest.Reply{Body: `{"version":"0.12.0"}`}
	case strings.HasSuffix(path, "/responses") && c.SSE:
		return wiretest.Reply{Body: SSEReply}
	case strings.HasSuffix(path, "/responses"):
		return wiretest.Reply{Body: ResponsesReply}
	case strings.HasSuffix(path, "/messages"):
		return wiretest.Reply{Body: MessagesReply}
	case strings.HasSuffix(path, "/interactions"):
		return wiretest.Reply{Body: InteractionsReply}
	case strings.HasSuffix(path, ":generateContent"):
		return wiretest.Reply{Body: GoogleReply}
	case strings.HasSuffix(path, "/chat/completions"):
		return wiretest.Reply{Body: ChatReply}
	}
	return wiretest.Reply{Status: http.StatusNotFound, Body: `{"error":{"message":"no canned reply for this path"}}`}
}

// errSkip marks a scenario that has no golden for this case.
var errSkip = errors.New("scenario not recorded for this case")

// scenario is one of 0015-PLAN S2's scenarios, through the new API. Each
// returns what its P7 method returned: a string for text and tool
// arguments, a *Response for items, a []string for the listing.
type scenario struct {
	name string
	run  func(ctx context.Context, c Case, p llmprovider.Provider) (any, error)
}

func toolArguments(ctx context.Context, p llmprovider.Provider, reasoning *llmprovider.Reasoning) (any, error) {
	call, err := llmprovider.GenerateToolCall(ctx, p, &llmprovider.Request{
		Input: []llmprovider.Item{user(Prompt)}, Tools: []llmprovider.Tool{WeatherTool()}, Reasoning: reasoning})
	return call.Arguments, err
}

var scenarios = []scenario{
	{"text", func(ctx context.Context, _ Case, p llmprovider.Provider) (any, error) {
		return llmprovider.GenerateText(ctx, p, &llmprovider.Request{Input: []llmprovider.Item{user(Prompt)}})
	}},
	{"tool", func(ctx context.Context, _ Case, p llmprovider.Provider) (any, error) {
		return toolArguments(ctx, p, nil)
	}},
	{"thinking", func(ctx context.Context, _ Case, p llmprovider.Provider) (any, error) {
		return llmprovider.GenerateText(ctx, p, &llmprovider.Request{
			Input: []llmprovider.Item{user(Prompt)}, Reasoning: &llmprovider.Reasoning{}})
	}},
	{"thinking-tool", func(ctx context.Context, _ Case, p llmprovider.Provider) (any, error) {
		return toolArguments(ctx, p, &llmprovider.Reasoning{})
	}},
	{"items", func(ctx context.Context, _ Case, p llmprovider.Provider) (any, error) {
		return p.Generate(ctx, &llmprovider.Request{Input: Items()})
	}},
	{"continuation", func(ctx context.Context, c Case, p llmprovider.Provider) (any, error) {
		if c.NoContinuation {
			return nil, errSkip
		}
		return p.Generate(ctx, &llmprovider.Request{
			PreviousResponseID: "resp_previous", Input: []llmprovider.Item{user("And tomorrow?")}})
	}},
	{"listing", func(ctx context.Context, _ Case, p llmprovider.Provider) (any, error) {
		lister, ok := p.(llmprovider.ModelLister)
		if !ok {
			return nil, errSkip
		}
		return lister.ListModels(ctx)
	}},
}

// result makes a return value comparable with its P7 golden: a Response's
// items are tagged with their type, and only the fields P7 had are kept.
func result(v any) any {
	resp, ok := v.(*llmprovider.Response)
	if !ok || resp == nil {
		return v // a nil *Response is recorded as null, as P7 recorded it
	}
	items := make([]map[string]any, len(resp.Output))
	for i, item := range resp.Output {
		items[i] = map[string]any{jsonType: strings.TrimPrefix(fmt.Sprintf("%T", item), "llmprovider."), "item": item}
	}
	return map[string]any{"id": resp.ID, "finish_reason": resp.FinishReason, "output": items}
}

// Run is G-wire for cases: each scenario's requests, as normalised JSON,
// match testdata/wire/<case>/<scenario>.json in the calling package.
func Run(t *testing.T, cases []Case, update bool) {
	t.Helper()
	for _, c := range cases {
		for _, s := range scenarios {
			t.Run(c.Name+"/"+s.name, func(t *testing.T) {
				srv := wiretest.NewServer(t, c.Reply)
				var extra []llmprovider.Option
				if s.name == "continuation" {
					extra = c.ContinueOpts
				}
				p, err := c.Build(srv.URL, extra...)
				if err != nil {
					t.Fatal(err)
				}
				got, err := s.run(context.Background(), c, p)
				if errors.Is(err, errSkip) {
					t.Skip("no golden for this scenario")
				}
				rec := wiretest.Record{Requests: srv.Requests(), Result: result(got)}
				if err != nil {
					rec.Error = err.Error()
				}
				wiretest.Check(t, filepath.Join("testdata", "wire", c.Name, s.name+".json"), rec, update)
			})
		}
	}
}
