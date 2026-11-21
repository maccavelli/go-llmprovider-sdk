package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0084-MADR D2 and D4 and the state table's ST03, ST05, ST08 to ST13, ST15,
// ST16, ST19, ST28, ST38 to ST45: through each provider that reads a refusal
// of a tool's schema, an in-process server replays a recorded refusal (0062-
// PLAN Phase 1's, or 0084-PLAN Phase 1's), and the error that Generate and
// Stream return names the tool the refusal names, when the request sent it,
// and no other. The classification itself is the llmprovider package's.
//
// A named edit of a recorded body (the plan's Rules, on the owner's ruling on
// N7) is stated where it is made: a name replaced, a name added, a position
// moved, a phrase added. Every edit decodes the recorded body, changes one
// field and encodes it again, so the body stays what the service sends.

// toolRoute is one route a D3 matcher reads, as these tests drive it.
type toolRoute struct {
	name  string
	id    llmprovider.ProviderID
	opts  []llmprovider.Option
	model string
}

func zenRoute(r opencode.Route) []llmprovider.Option {
	return []llmprovider.Option{opencode.WithRoute(r)}
}

var (
	routeGrok         = toolRoute{"grok", llmprovider.ProviderGrok, nil, "grok-4.6"}
	routeGrok2        = toolRoute{"grok2", llmprovider.ProviderGrok, nil, "grok-4.5"}
	routeOpenAI       = toolRoute{"openai", llmprovider.ProviderOpenAI, nil, "gpt-6-luna"}
	routeZenResponses = toolRoute{"zen-responses", llmprovider.ProviderOpencodeZen, zenRoute(opencode.RouteResponses), "gpt-5.5"}
	routeClaude       = toolRoute{"claude", llmprovider.ProviderClaude, nil, "claude-sonnet-5"}
	routeZenGoogle    = toolRoute{"zen-google", llmprovider.ProviderOpencodeZen, zenRoute(opencode.RouteGoogle), "gemini-3.8-flash"}
	routeZenMessages  = toolRoute{"zen-messages", llmprovider.ProviderOpencodeZen, zenRoute(opencode.RouteMessages), "claude-sonnet-5"}
	// The routes D3 does not read by name but N3 reads by service and marker.
	routeZenChat   = toolRoute{"zen-chat", llmprovider.ProviderOpencodeZen, zenRoute(opencode.RouteChatCompletions), "glm-5.3"}
	routeGoMessage = toolRoute{"go-messages", llmprovider.ProviderOpencodeGo, zenRoute(opencode.RouteMessages), "qwen3.8-flash"}
)

// withModel is r on another model, for a form the model decides.
func (r toolRoute) withModel(model string) toolRoute {
	r.model = model
	return r
}

// provider builds r's provider against url.
func (r toolRoute) provider(t *testing.T, url string, more ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	opts := append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(r.model), llmprovider.WithBaseURL(url),
		llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false), testCache(t)}, r.opts...)
	p, err := New(r.id, append(opts, more...)...)
	if err != nil {
		t.Fatalf("%s: %v", r.name, err)
	}
	return p
}

// recorded0084 is a recorded refusal under internal/wire/testdata.
type recorded0084 struct {
	Case  string `json:"case"`
	Tools []struct {
		Name   string `json:"name"`
		Schema any    `json:"schema"`
	} `json:"tools"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Body    string `json:"body"`
}

func readRecorded0084(t *testing.T, rel string) recorded0084 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "internal", "wire", "testdata", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var r recorded0084
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return r
}

// noopTool is the first tool of every recorded run: a valid one, so that the
// refused tool sits at index 1.
func noopTool() llmprovider.Tool {
	return llmprovider.Tool{Name: "noop", Schema: map[string]any{"type": "object", "properties": map[string]any{}}}
}

// tools is the list the recorded run sent: Phase 1's own, or, for a 0062
// record, which sent one tool, noop first and the refused tool at index 1, as
// the plan's Rules allow (a named edit: the tool's position).
func (r recorded0084) tools() []llmprovider.Tool {
	if len(r.Tools) == 0 {
		return []llmprovider.Tool{noopTool(), {Name: "record_city", Schema: map[string]any{"type": "string"}}}
	}
	out := make([]llmprovider.Tool, len(r.Tools))
	for i, tool := range r.Tools {
		out[i] = llmprovider.Tool{Name: tool.Name, Schema: tool.Schema}
	}
	return out
}

// asked is a request with the tools given.
func asked(tools ...llmprovider.Tool) *llmprovider.Request {
	return &llmprovider.Request{Tools: tools,
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Reply with OK."}}}
}

// editBody decodes body, lets change edit its top-level object, and encodes
// it again.
func editBody(t *testing.T, body string, change func(m map[string]any)) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("body %q: %v", body, err)
	}
	change(m)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// editMessage is body with its error message changed by change: Grok's
// "error" string, or the message of the "error" object every other form has.
func editMessage(t *testing.T, body string, change func(msg string) string) string {
	t.Helper()
	return editBody(t, body, func(m map[string]any) {
		switch e := m["error"].(type) {
		case string:
			m["error"] = change(e)
		case map[string]any:
			msg, _ := e["message"].(string)
			e["message"] = change(msg)
		default:
			t.Fatalf("body has no error message to edit")
		}
	})
}

// renamed is body with the tool name from replaced by to in its message: a
// named edit, the tool's name.
func renamed(t *testing.T, body, from, to string) string {
	t.Helper()
	return editMessage(t, body, func(msg string) string {
		if !strings.Contains(msg, from) {
			t.Fatalf("message %q holds no %q to rename", msg, from)
		}
		return strings.ReplaceAll(msg, from, to)
	})
}

// appended is body with text added to its message: a named edit, a second
// name or index, or a phrase from another table.
func appended(t *testing.T, body, text string) string {
	t.Helper()
	return editMessage(t, body, func(msg string) string { return msg + text })
}

// refuse400 serves body at 400 to every POST, and wirecase's canned reply to
// anything else.
func refuse400(body string) func(*http.Request) wiretest.Reply {
	return func(r *http.Request) wiretest.Reply {
		if r.Method != http.MethodPost {
			return wirecase.Case{}.Reply(r)
		}
		return wiretest.Reply{Status: http.StatusBadRequest, Header: map[string]string{"Content-Type": "application/json"}, Body: body}
	}
}

// firstStreamError is the first error Stream yields.
func firstStreamError(p llmprovider.Provider, req *llmprovider.Request) error {
	for _, err := range llmprovider.Stream(context.Background(), p, req) {
		if err != nil {
			return err
		}
	}
	return nil
}

// refusalErrors sends req to r's provider, which a server refusing every
// POST with body answers, by Generate and by Stream, and returns each error.
func refusalErrors(t *testing.T, r toolRoute, req *llmprovider.Request, body string, more ...llmprovider.Option) (generate, stream error) {
	t.Helper()
	srv := wiretest.NewServer(t, refuse400(body))
	p := r.provider(t, srv.URL, more...)
	_, generate = p.Generate(context.Background(), req)
	return generate, firstStreamError(r.provider(t, srv.URL, more...), req)
}

// wantKind checks that err is the kind, from an *APIError of status 400 whose
// own Kind it is, and returns it.
func wantKind(t *testing.T, name string, err error) *llmprovider.APIError {
	t.Helper()
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		t.Errorf("%s: %v; want an *APIError", name, err)
		return &llmprovider.APIError{}
	}
	if !errors.Is(err, llmprovider.ErrToolSchemaRefused) || !errors.Is(err, llmprovider.ErrInvalidRequest) ||
		!errors.Is(apiErr.Kind, llmprovider.ErrToolSchemaRefused) || apiErr.Status != http.StatusBadRequest {
		t.Errorf("%s: %v (Kind %v, Status %d); want ErrToolSchemaRefused, and so ErrInvalidRequest, from a 400", name, err,
			apiErr.Kind, apiErr.Status)
	}
	return apiErr
}

// wantTool is wantKind and that the error's Tool is want, the empty string
// being no name.
func wantTool(t *testing.T, name string, err error, want string) {
	t.Helper()
	if apiErr := wantKind(t, name, err); apiErr.Tool != want {
		t.Errorf("%s: Tool %q; want %q", name, apiErr.Tool, want)
	}
}

// names0084 are the recorded refusals each route's matcher reads, by
// 0084-PLAN Phase 1c's sort (MADR ST53 to ST55, ST59, ST69), with the tool
// each names, which sits at index 1 of its list (O2's second bad tool,
// record_country, is at index 2: OpenAI and Zen Responses name it, since it
// is the only bad one; Grok and Anthropic name record_city, the union's).
var names0084 = []struct {
	route toolRoute
	file  string
	want  string
}{
	{routeGrok, "toolschema-0084/responses/grok-root-string.json", "record_city"},
	{routeGrok, "toolschema-0084/responses/grok-O1-root-union.json", "record_city"},
	{routeGrok, "toolschema-0084/responses/grok-O2-two-bad.json", "record_city"},
	{routeGrok2, "toolschema-0084/responses/grok2-root-string.json", "record_city"},
	{routeGrok2, "toolschema-0084/responses/grok2-O1-root-union.json", "record_city"},
	{routeGrok2, "toolschema-0084/responses/grok2-O2-two-bad.json", "record_city"},
	{routeOpenAI, "toolschema-0084/responses/openai-root-string.json", "record_city"},
	{routeOpenAI, "toolschema-0084/responses/openai-O2-two-bad.json", "record_country"},
	{routeZenResponses, "toolschema-0084/responses/zen-responses-root-string.json", "record_city"},
	{routeZenResponses, "toolschema-0084/responses/zen-responses-O2-two-bad.json", "record_country"},
	{routeClaude, "toolschema-0084/messages/claude-root-string.json", "record_city"},
	{routeClaude, "toolschema-0084/messages/claude-O1-root-union.json", "record_city"},
	{routeClaude, "toolschema-0084/messages/claude-O2-two-bad.json", "record_city"},
	// OpenCode Zen's Google route recorded no refusal in a form D3 reads in
	// Phase 1: its 0062 record stands, with the tool moved to index 1.
	{routeZenGoogle, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", "record_city"},
}

// TestToolSchemaRefusal_Names_0084 (MADR D2, D4; ST01 to ST06, ST09, ST15,
// ST16, ST18, ST19, ST29, ST30, ST69, ST72): through Generate and through
// Stream, each recorded refusal is the kind with Tool the tool it names,
// which the request sent; errors.As finds the *APIError and its Kind is the
// kind. The server is a proxy passing the recorded status and body through
// (ST19).
func TestToolSchemaRefusal_Names_0084(t *testing.T) {
	for _, tc := range names0084 {
		f := readRecorded0084(t, tc.file)
		gen, stream := refusalErrors(t, tc.route, asked(f.tools()...), f.Body)
		wantTool(t, f.Case+" through Generate", gen, tc.want)
		wantTool(t, f.Case+" through Stream", stream, tc.want)
	}
	// ST72: a caller's tool named use_tool, with no deferred tool beside it,
	// is an ordinary sent tool, and a refusal that names it is named by
	// string. F1 with its tool renamed use_tool, a named edit.
	f := readRecorded0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	tools := []llmprovider.Tool{noopTool(), {Name: "use_tool", Schema: map[string]any{"type": "string"}}}
	gen, stream := refusalErrors(t, routeGrok, asked(tools...), renamed(t, f.Body, "record_city", "use_tool"))
	wantTool(t, "a caller's use_tool through Generate", gen, "use_tool")
	wantTool(t, "a caller's use_tool through Stream", stream, "use_tool")
}

// TestToolSchemaRefusal_NeverNamesUnsent_0084 (MADR D2, D4; ST03, ST10): the
// kind, with Tool empty, when the reply names a tool the request did not
// send, a schema field, a deferred tool that was not sent, Grok's use_tool
// wrapper, an index past the sent list, or OpenAI's param naming index 1
// while its message names an unsent tool; and for a request that sent no
// tools. The kind and the empty Tool are asserted apart.
func TestToolSchemaRefusal_NeverNamesUnsent_0084(t *testing.T) {
	grok := readRecorded0084(t, "toolschema-0084/responses/grok-root-string.json")
	claude := readRecorded0084(t, "toolschema-0084/messages/claude-root-string.json")
	openai := readRecorded0084(t, "refusals-0062/responses/openai-string-schema-tool.json")
	google := readRecorded0084(t, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json")
	deferred := llmprovider.Tool{Name: "deferred_tool", Schema: map[string]any{"type": "object"}, Deferred: true}

	for _, tc := range []struct {
		name  string
		route toolRoute
		req   *llmprovider.Request
		body  string
	}{
		{"grok: a name the request did not send", routeGrok, asked(grok.tools()...),
			renamed(t, grok.Body, "record_city", "ghost_tool")},
		{"grok: a schema field only", routeGrok, asked(grok.tools()...), renamed(t, grok.Body, "record_city", "properties")},
		{"grok: a deferred tool that was not sent", routeGrok, asked(append(grok.tools(), deferred)...),
			renamed(t, grok.Body, "record_city", "deferred_tool")},
		{"grok: the use_tool wrapper", routeGrok, asked(append(grok.tools(), deferred)...),
			renamed(t, grok.Body, "record_city", "use_tool")},
		{"grok: a request that sent no tools", routeGrok, asked(), grok.Body},
		{"openai: a deferred tool that was not sent", routeOpenAI.withModel("gpt-4.1-mini"),
			asked(append(openai.tools(), deferred)...), renamed(t, openai.Body, "record_city", "deferred_tool")},
		// N2: param names index 1, the sent record_city; the message names an
		// unsent tool, and the message decides.
		{"openai: param names index 1, the message an unsent tool", routeOpenAI, asked(openai.tools()...),
			renamed(t, editBody(t, openai.Body, func(m map[string]any) {
				m["error"].(map[string]any)["param"] = "tools[1].parameters"
			}), "record_city", "ghost_tool")},
		{"claude: an index past the sent list", routeClaude, asked(claude.tools()...),
			renamed(t, claude.Body, "tools.1.", "tools.7.")},
		{"claude: a request that sent no tools", routeClaude, asked(), renamed(t, claude.Body, "tools.1.", "tools.0.")},
		// Before 4.5 a deferred tool not yet loaded is not sent, so the list
		// sent is shorter than Request.Tools and index 2 is past it.
		{"claude: past the sent list, a deferred tool before it", routeClaude.withModel("claude-opus-4-1"),
			asked(noopTool(), deferred, llmprovider.Tool{Name: "record_city", Schema: map[string]any{"type": "string"}}),
			renamed(t, claude.Body, "tools.1.", "tools.2.")},
		{"zen google: a name the request did not send", routeZenGoogle, asked(google.tools()...),
			renamed(t, google.Body, "record_city", "ghost_tool")},
	} {
		gen, stream := refusalErrors(t, tc.route, tc.req, tc.body)
		for _, got := range []struct {
			via string
			err error
		}{{"Generate", gen}, {"Stream", stream}} {
			wantTool(t, tc.name+" through "+got.via, got.err, "")
		}
	}
}

// TestToolSchemaRefusal_Deferral_0084 (MADR D4; ST05): Anthropic's index is
// into the list as sent after deferral, not into Request.Tools. On a Claude
// model before 4.5, the Append form, a deferred tool not yet loaded is not
// sent, so index 1 of the list sent is record_city, while index 1 of
// Request.Tools is the deferred tool.
func TestToolSchemaRefusal_Deferral_0084(t *testing.T) {
	claude := readRecorded0084(t, "toolschema-0084/messages/claude-root-string.json")
	deferred := llmprovider.Tool{Name: "deferred_tool", Schema: map[string]any{"type": "object"}, Deferred: true}
	tools := []llmprovider.Tool{noopTool(), deferred, {Name: "record_city", Schema: map[string]any{"type": "string"}}}
	for _, route := range []toolRoute{routeClaude.withModel("claude-opus-4-1"), routeZenMessages.withModel("claude-opus-4-1")} {
		srv := wiretest.NewServer(t, refuse400(claude.Body))
		_, err := route.provider(t, srv.URL).Generate(context.Background(), asked(tools...))
		// OpenCode's Messages route does not read Anthropic's marker
		// (ST66 (5)), so only Claude's own is the kind here.
		if route.name == "claude" {
			wantTool(t, route.name, err, "record_city")
		} else if errors.Is(err, llmprovider.ErrToolSchemaRefused) {
			t.Errorf("%s: %v; Anthropic's marker is read for Claude only", route.name, err)
		}
		if got := sentTools(lastBody(srv)); !slices.Equal(got, []string{"noop", "record_city"}) {
			t.Errorf("%s sent tools %v; want [noop record_city], the deferred tool left out", route.name, got)
		}
	}
	// Streamed too, and on the native form, where the list sent is
	// Request.Tools (a deferred tool is sent with defer_loading): index 1 is
	// then the deferred tool.
	for _, tc := range []struct {
		model, want string
	}{{"claude-opus-4-1", "record_city"}, {"claude-haiku-5-5", "deferred_tool"}} {
		route := routeClaude.withModel(tc.model)
		srv := wiretest.NewServer(t, refuse400(claude.Body))
		wantTool(t, tc.model+" through Stream", firstStreamError(route.provider(t, srv.URL), asked(tools...)), tc.want)
	}
}

// TestToolSchemaRefusal_LastSendKeepsName_0084 (MADR D4's refinement; C27,
// ST39): the name is filled where the provider returns the error, not in the
// resend hook, which is not consulted once MaxSends sends have been made. xAI
// refuses the effort sent (its 0062 form) and the resend is refused for its
// schema: named. The same on the last allowed send: Anthropic's learner
// resends five times for efforts, each refusal listing fewer, and once for
// the thinking shape, so the seventh send, the last, is refused for its
// schema, and its error is named, through Generate and Stream.
func TestToolSchemaRefusal_LastSendKeepsName_0084(t *testing.T) {
	grok := readRecorded0084(t, "toolschema-0084/responses/grok-root-string.json")
	claude := readRecorded0084(t, "toolschema-0084/messages/claude-root-string.json")
	req := func(tools []llmprovider.Tool) *llmprovider.Request {
		r := asked(tools...)
		r.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortMax}
		return r
	}

	// xAI: send 1 refused for its effort, send 2 for its schema.
	srv := newLearnServer(t, refusedGrok, grok.Body)
	p, err := New(llmprovider.ProviderGrok, llmprovider.WithAPIKey("k"), llmprovider.WithModel("grok-4.6"),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false), testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Generate(context.Background(), req(grok.tools()))
	wantTool(t, "grok after a refused effort", err, "record_city")
	if got := srv.efforts(); !slices.Equal(got, []string{"max", "xhigh"}) {
		t.Errorf("grok sent efforts %v; want max, then xhigh", got)
	}

	// Anthropic: seven sends, the last allowed.
	effortRefusal := func(list string) string {
		return `{"type":"error","error":{"type":"invalid_request_error","message":"output_config.effort: Input should be ` + list + `"}}`
	}
	refusals := []string{
		effortRefusal(`'low', 'medium', 'high' or 'xhigh'`), // max
		effortRefusal(`'low', 'medium' or 'high'`),          // xhigh
		effortRefusal(`'low' or 'medium'`),                  // high
		effortRefusal(`'low'`),                              // medium
		effortRefusal(`'none'`),                             // low: no effort is left
		refusedAdaptive,                                     // adaptive, with no effort
		claude.Body,                                         // the budget shape, send 7
	}
	for _, via := range []string{"Generate", "Stream"} {
		// Each subtest has its own catalog cache, so that what the first
		// learned is not known to the second.
		t.Run(via, func(t *testing.T) {
			srv := newLearnServer(t, refusals...)
			p, err := New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-sonnet-5"),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false), testCache(t))
			if err != nil {
				t.Fatal(err)
			}
			if via == "Generate" {
				_, err = p.Generate(context.Background(), req(claude.tools()))
			} else {
				err = firstStreamError(p, req(claude.tools()))
			}
			wantTool(t, "claude on the last send", err, "record_city")
			srv.mu.Lock()
			sends := len(srv.bodies)
			srv.mu.Unlock()
			if sends != wire.MaxSends {
				t.Errorf("claude made %d sends; want %d, the last allowed, refused for its schema", sends, wire.MaxSends)
			}
		})
	}
}

// TestToolSchemaRefusal_RedactedNeighbour_0084 (MADR X1; ST41): redaction
// keeps a tool's name and can eat the word after it. F1 with its tool
// renamed lookup_api_key, a named edit, reads "[invalid_client_tool_schema]
// lookup_api_key: [REDACTED] parameter root ...", and the name is still
// found.
func TestToolSchemaRefusal_RedactedNeighbour_0084(t *testing.T) {
	grok := readRecorded0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	const name = "lookup_api_key"
	tools := []llmprovider.Tool{noopTool(), {Name: name, Schema: map[string]any{"type": "string"}}}
	gen, stream := refusalErrors(t, routeGrok, asked(tools...), renamed(t, grok.Body, "record_city", name))
	wantTool(t, "through Generate", gen, name)
	wantTool(t, "through Stream", stream, name)
	if apiErr, ok := errors.AsType[*llmprovider.APIError](gen); ok && !strings.Contains(apiErr.Message, "[REDACTED]") {
		t.Errorf("Message %q holds no [REDACTED]; the case is meant to meet a redacted neighbour", apiErr.Message)
	}
}

// TestToolSchemaRefusal_RedactedName_0084 (MADR D2; ST42): a tool whose name
// redaction itself removes is not found in the message, so the kind has no
// Tool. F1 with its tool renamed to a made-up key-shaped name, a named edit;
// the name is built at run time and asserted absent from Message.
func TestToolSchemaRefusal_RedactedName_0084(t *testing.T) {
	grok := readRecorded0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	name := "key_" + strings.Repeat("a", 12)
	tools := []llmprovider.Tool{noopTool(), {Name: name, Schema: map[string]any{"type": "string"}}}
	gen, stream := refusalErrors(t, routeGrok, asked(tools...), renamed(t, grok.Body, "record_city", name))
	for _, got := range []struct {
		via string
		err error
	}{{"Generate", gen}, {"Stream", stream}} {
		wantTool(t, got.via, got.err, "")
		if apiErr, ok := errors.AsType[*llmprovider.APIError](got.err); ok && strings.Contains(apiErr.Message, name) {
			t.Errorf("%s: Message %q holds the name; the case is meant to have it redacted", got.via, apiErr.Message)
		}
	}
}

// TestToolSchemaRefusal_NameBeyondBound_0084 (MADR D2; ST43, C12): a tool
// whose name runs past Message's 512-byte bound is cut inside the name, so no
// name is read: the kind, with Tool empty, and never a prefix of the name.
// F7 with its tool renamed to a 600-byte name, a named edit.
func TestToolSchemaRefusal_NameBeyondBound_0084(t *testing.T) {
	google := readRecorded0084(t, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json")
	name := "tool_" + strings.Repeat("n", 595)
	tools := []llmprovider.Tool{noopTool(), {Name: name, Schema: map[string]any{"type": "string"}}}
	gen, stream := refusalErrors(t, routeZenGoogle, asked(tools...), renamed(t, google.Body, "record_city", name))
	for _, got := range []struct {
		via string
		err error
	}{{"Generate", gen}, {"Stream", stream}} {
		apiErr := wantKind(t, got.via, got.err)
		if apiErr.Tool != "" || len(apiErr.Message) > 520 || strings.Contains(apiErr.Message, name) {
			t.Errorf("%s: Tool %q, Message of %d bytes; want no Tool, a Message cut inside the name", got.via, apiErr.Tool,
				len(apiErr.Message))
		}
	}
}

// TestToolSchemaRefusal_SeveralNamed_0084 (MADR D2, Q4; ST12): the kind with
// Tool empty when the reply names several tools, each sent. The recorded
// O2 replies name one (Phase 1c), so each body is a recorded one with a
// second name or index added, a named edit (N7): Grok's tag again, an
// OpenAI function again, a second tools.N. path, a second backticked name.
func TestToolSchemaRefusal_SeveralNamed_0084(t *testing.T) {
	tools := []llmprovider.Tool{noopTool(), {Name: "record_city", Schema: map[string]any{"type": "string"}},
		{Name: "record_country", Schema: map[string]any{"type": "string"}}}
	for _, tc := range []struct {
		route toolRoute
		file  string
		add   string
	}{
		{routeGrok, "toolschema-0084/responses/grok-root-string.json",
			" [invalid_client_tool_schema] record_country: tool parameter root must be an object type"},
		{routeOpenAI, "refusals-0062/responses/openai-string-schema-tool.json",
			" Invalid schema for function 'record_country': schema must be a JSON Schema of 'type: \"object\"'."},
		{routeClaude, "toolschema-0084/messages/claude-root-string.json",
			"; tools.2.custom.input_schema.type: Input should be 'object'"},
		{routeZenGoogle, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json",
			" Also `record_country` functionDeclaration parameters schema should be of type OBJECT."},
	} {
		f := readRecorded0084(t, tc.file)
		gen, stream := refusalErrors(t, tc.route, asked(tools...), appended(t, f.Body, tc.add))
		wantTool(t, f.Case+" through Generate", gen, "")
		wantTool(t, f.Case+" through Stream", stream, "")
	}
}

// TestToolSchemaRefusal_NamesNothing_0084 (MADR D2, Q4; ST11): the route's
// marker and no tool named at all: the kind, with Tool empty, and never the
// last tool sent. F1, F2 and F7 with their tool renamed to the empty string,
// a named edit.
func TestToolSchemaRefusal_NamesNothing_0084(t *testing.T) {
	for _, tc := range []struct {
		route toolRoute
		file  string
	}{
		{routeGrok, "refusals-0062/responses/grok-string-schema-tool.json"},
		{routeOpenAI, "refusals-0062/responses/openai-string-schema-tool.json"},
		{routeZenGoogle, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json"},
	} {
		f := readRecorded0084(t, tc.file)
		gen, stream := refusalErrors(t, tc.route, asked(f.tools()...), renamed(t, f.Body, "record_city", ""))
		wantTool(t, f.Case+" through Generate", gen, "")
		wantTool(t, f.Case+" through Stream", stream, "")
	}
}

// TestToolSchemaRefusal_DuplicateNames_0084 (MADR D2; ST13; N6, "Name by
// string"): two sent tools of one name, and a refusal naming it: Tool is the
// name, since every entry it can mean shares it. On Anthropic's index form,
// a reply naming both indices is one name too. Nothing refuses duplicates
// before the network (H6).
func TestToolSchemaRefusal_DuplicateNames_0084(t *testing.T) {
	grok := readRecorded0084(t, "toolschema-0084/responses/grok-root-string.json")
	claude := readRecorded0084(t, "toolschema-0084/messages/claude-root-string.json")
	twice := []llmprovider.Tool{noopTool(), {Name: "record_city", Schema: map[string]any{"type": "string"}},
		{Name: "record_city", Schema: map[string]any{"type": "string"}}}
	gen, stream := refusalErrors(t, routeGrok, asked(twice...), grok.Body)
	wantTool(t, "grok through Generate", gen, "record_city")
	wantTool(t, "grok through Stream", stream, "record_city")
	both := appended(t, claude.Body, "; tools.2.custom.input_schema.type: Input should be 'object'")
	gen, stream = refusalErrors(t, routeClaude, asked(twice...), both)
	wantTool(t, "claude naming both indices through Generate", gen, "record_city")
	wantTool(t, "claude naming both indices through Stream", stream, "record_city")
}

// TestToolSchemaRefusal_SearchLoaded_0084 (MADR D2, D4; ST10, the owner's
// ruling "Name the caller's Search tool"): with native deferral on a
// Responses model from 5.4 on, the body holds the caller's Search tool as a
// nameless tool_search entry, and a tool a result loaded inside a
// tool_search_output item. A refusal that names either names a tool the body
// held: Tool is each name. F2 renamed to the loaded tool, then to the Search
// tool, each a named edit.
func TestToolSchemaRefusal_SearchLoaded_0084(t *testing.T) {
	openai := readRecorded0084(t, "refusals-0062/responses/openai-string-schema-tool.json")
	req := func() *llmprovider.Request {
		return &llmprovider.Request{
			Tools: []llmprovider.Tool{noopTool(), {Name: "tool_search", Search: true, Schema: map[string]any{"type": "object"}},
				{Name: "loaded_tool", Deferred: true, Schema: map[string]any{"type": "string"}},
				{Name: "unloaded_tool", Deferred: true, Schema: map[string]any{"type": "string"}}},
			Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "find a tool"},
				llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"loaded"}`},
				llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded loaded_tool."},
				llmprovider.ToolLoadItem{CallID: "c1", Name: "loaded_tool"},
			}}
	}
	for _, route := range []toolRoute{routeOpenAI, routeZenResponses} {
		for _, tc := range []struct{ name, named, want string }{
			{"the tool the search loaded", "loaded_tool", "loaded_tool"},
			{"the Search tool", "tool_search", "tool_search"},
			{"a tool of the list", "noop", "noop"},
			{"a deferred tool not loaded", "unloaded_tool", ""},
		} {
			srv := wiretest.NewServer(t, refuse400(renamed(t, openai.Body, "record_city", tc.named)))
			_, err := route.provider(t, srv.URL).Generate(context.Background(), req())
			wantTool(t, route.name+": "+tc.name, err, tc.want)
			body := lastBody(srv)
			if !strings.Contains(fmt.Sprint(body["input"]), "loaded_tool") || !strings.Contains(fmt.Sprint(body["input"]), "tool_search_output") {
				t.Errorf("%s: the request held no tool_search_output with loaded_tool: %v", route.name, body["input"])
			}
			wantList := []string{"noop", "tool_search"}
			if got := sentTools(body); !slices.Equal(got, wantList) {
				t.Errorf("%s: tools sent %v; want %v", route.name, got, wantList)
			}
		}
	}
}

// TestToolSchemaRefusal_SharedLabel_0084 (MADR ST08; the owner's ruling on
// N3, "By service and marker"): the kind is read on every route of a read
// service whose reply carries the service's marker. OpenCode Zen's chat route
// and OpenCode Go's Messages route each serve Zen Responses' refusal (F3),
// and OpenAI's ChatGPT session and plan session each serve F2: the kind, Tool
// the sent name. A route's stream is classified as its Generate is.
func TestToolSchemaRefusal_SharedLabel_0084(t *testing.T) {
	zen := readRecorded0084(t, "toolschema-0084/responses/zen-responses-root-string.json")
	openai := readRecorded0084(t, "toolschema-0084/responses/openai-root-string.json")
	chatGPT := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "sess", Refresh: "refresh",
		Expiry: time.Now().Add(time.Hour)}
	plan := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer, Access: "plan-access",
		Refresh: "plan-refresh", Expiry: time.Now().Add(time.Hour), ClientID: "oaiapp_test", AccountID: "acct-plan",
		Resource: "https://api.openai.com/v1", TokenURL: auth.DefaultOpenAIIssuer + "/api/accounts/oauth/token"}
	for _, tc := range []struct {
		name  string
		route toolRoute
		more  []llmprovider.Option
		f     recorded0084
	}{
		{"zen chat route", routeZenChat, nil, zen},
		{"go messages route", routeGoMessage, nil, zen},
		{"openai chatgpt session", routeOpenAI.withModel("gpt-6-astra"), []llmprovider.Option{llmprovider.WithTokenSource(chatGPT)}, openai},
		{"openai plan session", routeOpenAI.withModel("gpt-6-astra"), []llmprovider.Option{llmprovider.WithTokenSource(plan)}, openai},
	} {
		name := tc.name
		gen, stream := refusalErrors(t, tc.route, asked(tc.f.tools()...), tc.f.Body, tc.more...)
		wantTool(t, name+" through Generate", gen, "record_city")
		wantTool(t, name+" through Stream", stream, "record_city")
	}
}

// TestToolSchemaRefusal_FieldsUnchanged_0084 (MADR D4; ST28, ST44, C12): the
// fields of the error other than Tool, read through Generate and through
// Stream, so that the naming helper has run, equal those ClassifyHTTPError
// gives the same reply: Status, Code, Message (redacted and bounded as
// before), RetryAfter, ResetAt, Limit and Reason. Nothing of the request, such
// as its schema, enters the error.
func TestToolSchemaRefusal_FieldsUnchanged_0084(t *testing.T) {
	for _, tc := range []struct {
		route toolRoute
		file  string
		label string
	}{
		{routeGrok, "refusals-0062/responses/grok-string-schema-tool.json", "grok"},
		{routeOpenAI, "refusals-0062/responses/openai-string-schema-tool.json", "openai"},
		{routeClaude, "refusals-0062/messages/claude-string-schema-tool.json", "claude"},
		{routeZenGoogle, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json", "opencode"},
	} {
		f := readRecorded0084(t, tc.file)
		want, ok := errors.AsType[*llmprovider.APIError](llmprovider.ClassifyHTTPError(tc.label, &http.Response{
			StatusCode: f.Status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(f.Body))}))
		if !ok {
			t.Fatalf("%s: ClassifyHTTPError gave no *APIError", f.Case)
		}
		gen, stream := refusalErrors(t, tc.route, asked(f.tools()...), f.Body)
		for _, got := range []struct {
			via string
			err error
		}{{"Generate", gen}, {"Stream", stream}} {
			apiErr, ok := errors.AsType[*llmprovider.APIError](got.err)
			if !ok {
				t.Errorf("%s through %s: %v; want an *APIError", f.Case, got.via, got.err)
				continue
			}
			if apiErr.Status != want.Status || apiErr.Code != want.Code || apiErr.Message != want.Message ||
				apiErr.RetryAfter != want.RetryAfter || !apiErr.ResetAt.Equal(want.ResetAt) || !reflect.DeepEqual(apiErr.Limit, want.Limit) ||
				apiErr.Reason != want.Reason {
				t.Errorf("%s through %s: %+v; want the fields of %+v", f.Case, got.via, apiErr, want)
			}
		}
	}
}

// requestSchemas is the schemas a request body sent, in order, whatever its
// wire: a Responses tool's parameters, a Messages tool's input_schema, a
// Gemini declaration's parameters.
func requestSchemas(body map[string]any) []any {
	var out []any
	list, _ := body["tools"].([]any)
	for _, raw := range list {
		tool, _ := raw.(map[string]any)
		switch {
		case tool["parameters"] != nil:
			out = append(out, tool["parameters"])
		case tool["input_schema"] != nil:
			out = append(out, tool["input_schema"])
		case tool["functionDeclarations"] != nil:
			for _, d := range tool["functionDeclarations"].([]any) {
				out = append(out, d.(map[string]any)["parameters"])
			}
		}
	}
	return out
}

// TestToolSchemaRefusal_NoAdaptation_0084 (MADR D5; ST38; the standards
// guide's R45): the SDK sends the caller's tool definitions as they are and
// sends the request once. Per D3 route, through Generate, a refusal is
// returned after one request, whose tools carry exactly the schemas the
// recorded run sent: nothing dropped, rewritten or resent.
func TestToolSchemaRefusal_NoAdaptation_0084(t *testing.T) {
	for _, tc := range names0084 {
		f := readRecorded0084(t, tc.file)
		srv := wiretest.NewServer(t, refuse400(f.Body))
		if _, err := tc.route.provider(t, srv.URL).Generate(context.Background(), asked(f.tools()...)); err == nil {
			t.Fatalf("%s: no error", f.Case)
		}
		posts := 0
		for _, r := range srv.Requests() {
			if r.Method == http.MethodPost {
				posts++
			}
		}
		if posts != 1 {
			t.Errorf("%s: %d requests reached the server; want one, no resend", f.Case, posts)
		}
		var want []any
		for _, tool := range f.tools() {
			raw, _ := json.Marshal(tool.Schema)
			var decoded any
			_ = json.Unmarshal(raw, &decoded)
			want = append(want, decoded)
		}
		var got []any
		for _, schema := range requestSchemas(lastBody(srv)) {
			raw, _ := json.Marshal(schema)
			var decoded any
			_ = json.Unmarshal(raw, &decoded)
			got = append(got, decoded)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the request's schemas %v; want the caller's, %v", f.Case, got, want)
		}
	}
}

// recordedLog is a slog handler that keeps every record, at every level, as
// text.
type recordedLog struct {
	mu    *sync.Mutex
	lines *[]string
	attrs string
}

func newRecordedLog() *recordedLog { return &recordedLog{mu: new(sync.Mutex), lines: new([]string)} }

func (h *recordedLog) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordedLog) Handle(_ context.Context, r slog.Record) error {
	line := r.Message + h.attrs
	r.Attrs(func(a slog.Attr) bool {
		line += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})
	h.mu.Lock()
	*h.lines = append(*h.lines, line)
	h.mu.Unlock()
	return nil
}

func (h *recordedLog) WithAttrs(as []slog.Attr) slog.Handler {
	c := *h
	for _, a := range as {
		c.attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value)
	}
	return &c
}

func (h *recordedLog) WithGroup(string) slog.Handler { return h }

// TestToolSchemaRefusal_NoLog_0084 (MADR ST45; H8): a provider given a
// logger that records every level logs no record holding the refusal's text
// or the Tool, through Generate and Stream, on each of the four forms.
func TestToolSchemaRefusal_NoLog_0084(t *testing.T) {
	for _, tc := range []struct {
		route toolRoute
		file  string
	}{
		{routeGrok, "refusals-0062/responses/grok-string-schema-tool.json"},
		{routeOpenAI, "refusals-0062/responses/openai-string-schema-tool.json"},
		{routeClaude, "toolschema-0084/messages/claude-root-string.json"},
		{routeZenGoogle, "refusals-0062/generatecontent/opencode-zen-google-string-schema-tool.json"},
	} {
		f := readRecorded0084(t, tc.file)
		log := newRecordedLog()
		gen, stream := refusalErrors(t, tc.route, asked(f.tools()...), f.Body, llmprovider.WithLogger(slog.New(log)))
		if gen == nil || stream == nil {
			t.Fatalf("%s: Generate gave %v and Stream %v; want the refusal from each", f.Case, gen, stream)
		}
		log.mu.Lock()
		lines := slices.Clone(*log.lines)
		log.mu.Unlock()
		for _, line := range lines {
			if strings.Contains(line, "record_city") || strings.Contains(line, f.Message) {
				t.Errorf("%s: a log record holds the refusal or the tool: %q", f.Case, line)
			}
		}
	}
}

// TestToolSchemaRefusal_Concurrent_0084 (MADR ST40; the plan's step 5: the
// list is per request): two Generate calls at once on one provider, each with
// its own refused tool, are each named for their own. The server holds the
// first request until the second arrives, so both are in flight together, and
// refuses each in F1's form with the name of the tool its own request sent
// at index 1 (a named edit: the tool's name).
func TestToolSchemaRefusal_Concurrent_0084(t *testing.T) {
	grok := readRecorded0084(t, "refusals-0062/responses/grok-string-schema-tool.json")
	var mu sync.Mutex
	arrived := 0
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		}
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		arrived++
		if arrived == 2 {
			close(release)
		}
		mu.Unlock()
		select {
		case <-release:
		case <-time.After(10 * time.Second):
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		name := ""
		if len(body.Tools) > 1 {
			name = body.Tools[1].Name
		}
		w.WriteHeader(http.StatusBadRequest)
		// The named edit, made as a text replacement: the name is an
		// identifier, so the body stays valid JSON.
		_, _ = io.WriteString(w, strings.Replace(grok.Body, "record_city", name, 1))
	}))
	t.Cleanup(srv.Close)
	p := routeGrok.provider(t, srv.URL)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, name := range []string{"tool_one", "tool_two"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = p.Generate(context.Background(),
				asked(noopTool(), llmprovider.Tool{Name: name, Schema: map[string]any{"type": "string"}}))
		}()
	}
	wg.Wait()
	wantTool(t, "the first call", errs[0], "tool_one")
	wantTool(t, "the second call", errs[1], "tool_two")
}
