//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// OpenCode's live tests, through its own package (0015-PLAN S7). They REQUIRE
// OPENCODE_API_KEY (0004-PLAN deviation D3) and use paid OpenCode Go models:
// Zen's free tier refuses clients other than OpenCode (MADR 0013 D1).

// opencodeProbedOn is providers/opencode's wireShapesProbedOn, for messages.
const opencodeProbedOn = "2026-09-27"

// opencodeRecorder keeps every POST's path and last body to an opencode.ai
// host, passing the request on.
type opencodeRecorder struct {
	mu    sync.Mutex
	paths []string
	sent  []byte
}

func (o *opencodeRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.Contains(r.URL.Host, "opencode.ai") && r.Method == http.MethodPost && r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		o.mu.Lock()
		o.paths, o.sent = append(o.paths, r.URL.Path), b
		o.mu.Unlock()
		r.Body = io.NopCloser(bytes.NewReader(b))
	}
	return http.DefaultTransport.RoundTrip(r)
}

func (o *opencodeRecorder) last() ([]string, []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.paths...), o.sent
}

// liveGo builds OpenCode Go with the real key, model and the options.
func liveGo(t *testing.T, model string, opts ...llmprovider.Option) llmprovider.Provider {
	t.Helper()
	p, err := opencode.NewGo(append([]llmprovider.Option{llmprovider.WithAPIKey(llmprovider.LiveOpencodeKey(t)),
		llmprovider.WithModel(model)}, opts...)...)
	if err != nil {
		t.Fatalf("NewGo: %v", err)
	}
	return p
}

func goModel(t *testing.T, candidates ...string) string {
	t.Helper()
	return llmprovider.LiveModel(t, llmprovider.ProviderOpencodeGo, candidates...)
}

func TestLive_OpencodeChatCompletions(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveGo(t, goModel(t, "hy3", "glm-5.3-flash", "kimi-k2.6")), userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Errorf("empty output (probed %s)", opencodeProbedOn)
	}
}

func TestLive_OpencodeResponses(t *testing.T) {
	rec := &opencodeRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	resp, err := liveGo(t, goModel(t, "gpt-6-luna", "grok-4.6"), llmprovider.WithHTTPClient(&http.Client{Transport: rec})).
		Generate(ctx, userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if paths, _ := rec.last(); len(paths) != 1 || !strings.HasSuffix(paths[0], "/responses") {
		t.Fatalf("request paths = %v, want one /responses", paths)
	}
	var sawMessage bool
	for _, it := range resp.Output {
		if _, ok := it.(llmprovider.MessageItem); ok {
			sawMessage = true
		}
	}
	if !sawMessage {
		t.Errorf("no MessageItem in responses output (probed %s)", opencodeProbedOn)
	}
}

// TestLive_OpencodeRouteStillEnforced is the measurement the entire 63+26-row
// route table rests on: routes are NOT interchangeable. If this fails, OpenCode
// has become a translating gateway and the table is no longer necessary.
func TestLive_OpencodeRouteStillEnforced(t *testing.T) {
	model := goModel(t, "gpt-6-luna", "grok-4.6")
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	_, err := liveGo(t, model).Generate(ctx, userText("hi"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("%s must still succeed on its documented /responses route: %v", model, err)
	}
	if _, err := liveGo(t, model, opencode.WithRoute(opencode.RouteChatCompletions)).Generate(ctx, userText("hi")); err == nil {
		t.Errorf("DRIFT (probed %s): %s now succeeds on /chat/completions. Routes were "+
			"measured as non-interchangeable; if that changed, the route table may no "+
			"longer be needed", opencodeProbedOn, model)
	}
}

// TestLive_OpencodeChatReasoningEffort is MADR 0009 §6's OpenCode gate (as
// amended 2026-09-26): on OpenCode Go, for each chat-routed utility model whose
// reasoning_options list "low" (glm-flash and Hy families), the gateway accepts
// reasoning_effort "low". It also proves the x-opencode-session header: Go
// answers 400 MissingSessionID without it.
func TestLive_OpencodeChatReasoningEffort(t *testing.T) {
	llmprovider.LiveOpencodeKey(t)
	t.Setenv("LLMPROVIDER_DISABLE_MODELS_METADATA", "0") // the real document decides
	for _, candidate := range []string{"glm-5.3-flash", "hy3"} {
		t.Run(candidate, func(t *testing.T) {
			model := goModel(t, candidate)
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			meta, err := catalog.LookupMetadata(ctx, "", nil)
			if err != nil {
				t.Skipf("metadata unreachable: %v", err)
			}
			if !slices.Contains(meta.ReasoningEfforts(llmprovider.ProviderOpencodeGo, model), string(llmprovider.EffortLow)) {
				t.Fatalf("DRIFT: %s reasoning_options no longer list low", model)
			}
			rec := &opencodeRecorder{}
			req := userText("Reply with only the word ALPHA")
			req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
			out, err := llmprovider.GenerateText(ctx, liveGo(t, model, llmprovider.WithMaxTokens(400),
				llmprovider.WithHTTPClient(&http.Client{Transport: rec})), req)
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("DRIFT (probed %s): gateway rejected reasoning_effort on %s: %v", opencodeProbedOn, model, err)
			}
			if paths, sent := rec.last(); len(paths) != 1 || !strings.HasSuffix(paths[0], "/chat/completions") ||
				!bytes.Contains(sent, []byte(`"reasoning_effort":"low"`)) {
				t.Fatalf("DRIFT: %s requests %v sent %s; want one chat request with reasoning_effort low", model, paths, sent)
			}
			if strings.TrimSpace(out) == "" {
				t.Errorf("empty output from %s", model)
			}
		})
	}
}

// liveNPM reads one model's provider.npm from the live models.opencode.ai
// document: the package OpenCode's client routes it by. ok is false when the
// document does not list the model.
func liveNPM(t *testing.T, section, model string) (npm string, ok bool) {
	t.Helper()
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://models.opencode.ai/api.json", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("models.opencode.ai unreachable: %v", err)
	}
	defer resp.Body.Close()
	var doc map[string]struct {
		Models map[string]struct {
			Provider struct {
				NPM string `json:"npm"`
			} `json:"provider"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode models.opencode.ai: %v", err)
	}
	m, ok := doc[section].Models[model]
	return m.Provider.NPM, ok
}

// TestLive_OpencodeRoutesFromMetadata: on OpenCode Go, qwen3.8-max is sent
// to the route its live provider.npm selects, as OpenCode's own client picks
// it (0012-MADR §3.1). The route used to be pinned to chat_completions, from
// the 2026-09-27 document's empty npm; the 2026-10-04 document gives
// @ai-sdk/anthropic, so the route moved to messages, which the provider
// followed (0021-MADR L1). The expected route is computed here, written
// independently of routeForNPM: an npm of @ai-sdk/openai, @ai-sdk/anthropic
// or @ai-sdk/google selects responses, messages or the Google route, and any
// other, or none, chat_completions.
func TestLive_OpencodeRoutesFromMetadata(t *testing.T) {
	const model = "qwen3.8-max"
	npm, listed := liveNPM(t, "opencode-go", model)
	if !listed {
		t.Skipf("models.opencode.ai no longer lists %s on opencode-go", model)
	}
	want := map[string]string{
		"@ai-sdk/openai":    "/responses",
		"@ai-sdk/anthropic": "/messages",
		"@ai-sdk/google":    ":generateContent",
	}[npm]
	if want == "" {
		want = "/chat/completions"
	}
	t.Logf("%s on opencode-go: provider.npm %q, so %s", model, npm, want)
	rec := &opencodeRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveGo(t, goModel(t, model), llmprovider.WithHTTPClient(&http.Client{Transport: rec})),
		userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if paths, _ := rec.last(); len(paths) != 1 || !strings.HasSuffix(paths[0], want) {
		t.Fatalf("request paths = %v, want one ending %s (provider.npm %q)", paths, want, npm)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("empty output")
	}
}

// TestLive_OpencodeResponsesStoreFalse: OpenCode Go's Responses route accepts
// store:false (measured 2026-09-27 on gpt-6-luna, with and without it).
func TestLive_OpencodeResponsesStoreFalse(t *testing.T) {
	rec := &opencodeRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	out, err := llmprovider.GenerateText(ctx, liveGo(t, goModel(t, "gpt-6-luna", "grok-4.6"), llmprovider.WithHTTPClient(&http.Client{Transport: rec})),
		userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if _, sent := rec.last(); !bytes.Contains(sent, []byte(`"store":false`)) {
		t.Fatalf("request did not send store:false: %s", sent)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("empty output")
	}
}

// TestLive_OpencodeMinimaxM3Adaptive: minimax-m3 on OpenCode Go's Messages
// route thinks under thinking.type "adaptive" (measured 2026-09-27: budget,
// adaptive and none are all accepted, so this proves acceptance and that
// reasoning is returned, not necessity).
func TestLive_OpencodeMinimaxM3Adaptive(t *testing.T) {
	rec := &opencodeRecorder{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req := userText("What is 17 * 23? Reply with the number.")
	req.Reasoning = &llmprovider.Reasoning{}
	res, err := liveGo(t, goModel(t, "minimax-m3"), llmprovider.WithHTTPClient(&http.Client{Transport: rec})).Generate(ctx, req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, sent := rec.last(); !bytes.Contains(sent, []byte(`"thinking":{"type":"adaptive"}`)) {
		t.Fatalf("request did not think adaptively: %s", sent)
	}
	var reasoned bool
	for _, it := range res.Output {
		if r, ok := it.(llmprovider.ReasoningItem); ok && strings.TrimSpace(r.Text) != "" {
			reasoned = true
		}
	}
	if !reasoned {
		t.Errorf("no ReasoningItem in %#v", res.Output)
	}
	if !strings.Contains(res.OutputText(), "391") {
		t.Errorf("reply %q, want 391", res.OutputText())
	}
}

// TestLive_OpencodeMessagesThinking: OpenCode Go's messages route accepts the
// low-effort budget on qwen3.8-flash (one of Go's utility six).
func TestLive_OpencodeMessagesThinking(t *testing.T) {
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req := userText("Reply with only the word ALPHA")
	req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
	out, err := llmprovider.GenerateText(ctx, liveGo(t, goModel(t, "qwen3.8-flash")), req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if !strings.Contains(strings.ToUpper(out), "ALPHA") {
		t.Errorf("output %q does not contain ALPHA", out)
	}
}

// TestLive_OpencodeInterleavedReasoningReplay: an interleaved Go chat model
// accepts the replayed reasoning on the assistant tool-call turn and answers
// from the result. Measured 2026-09-27: kimi-k2.6 accepts the turn with or
// without the field, so this proves acceptance, not necessity.
func TestLive_OpencodeInterleavedReasoningReplay(t *testing.T) {
	t.Setenv("LLMPROVIDER_DISABLE_MODELS_METADATA", "0") // the real models.opencode.ai document decides
	rec := &opencodeRecorder{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := liveGo(t, goModel(t, "kimi-k2.6"), llmprovider.WithHTTPClient(&http.Client{Transport: rec})).Generate(ctx,
		&llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "What is the weather in Paris? Use the tool."},
			llmprovider.ReasoningItem{Text: "The user wants Paris weather; call get_weather."},
			llmprovider.FunctionCallItem{CallID: "call_rp_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
			llmprovider.FunctionCallOutputItem{CallID: "call_rp_1", Output: `{"forecast":"sunny, 21C"}`},
		}})
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, sent := rec.last(); !bytes.Contains(sent, []byte(`"reasoning_content":"The user wants Paris weather; call get_weather."`)) {
		t.Fatalf("request did not replay the reasoning: %s", sent)
	}
	if text := strings.ToLower(res.OutputText()); !strings.Contains(text, "sunny") && !strings.Contains(text, "21") {
		t.Fatalf("reply %q does not use the tool result", res.OutputText())
	}
}

// TestLive_OpencodeSystemMessage: a system instruction reaches the model on
// OpenCode's messages route, as the top-level system field, and the model
// follows it. It was the go-messages row of TestLive_SystemMessage. The check
// is a pair, with and without a neutral instruction, so an obeyed instruction
// is told from a reply that happens to match, and the instruction does not
// contradict the user (0024-MADR).
func TestLive_OpencodeSystemMessage(t *testing.T) {
	model := goModel(t, "qwen3.8-flash", "minimax-m3")
	for _, tc := range []struct {
		name   string
		system string
		want   bool
	}{
		{"with", "Begin every reply with the word OMEGA.", true},
		{"without", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			rec := &opencodeRecorder{}
			var input []llmprovider.Item
			if tc.system != "" {
				input = append(input, llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: tc.system})
			}
			input = append(input, llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."})
			res, err := liveGo(t, model, llmprovider.WithHTTPClient(&http.Client{Transport: rec})).Generate(ctx, &llmprovider.Request{Input: input})
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := strings.Contains(strings.ToUpper(res.OutputText()), "OMEGA"); got != tc.want {
				t.Fatalf("reply %q: contains OMEGA = %v, want %v", res.OutputText(), got, tc.want)
			}
			paths, sent := rec.last()
			var body map[string]json.RawMessage
			if err := json.Unmarshal(sent, &body); err != nil {
				t.Fatalf("request body: %v", err)
			}
			if _, hasSystem := body["system"]; len(paths) != 1 || !strings.HasSuffix(paths[0], "/messages") || hasSystem != tc.want {
				t.Fatalf("requests %v, top-level system = %v; want one /messages call, and a system field = %v", paths, hasSystem, tc.want)
			}
		})
	}
}

// TestLive_OpencodeToolRoundTrip sends a completed tool call and its result on
// each Go route, and requires the model to answer from the result (MADR 0012
// §2). It was the go-* rows of TestLive_ToolRoundTrip.
func TestLive_OpencodeToolRoundTrip(t *testing.T) {
	for name, candidates := range map[string][]string{
		"go-chat":      {"glm-5.3-flash", "glm-5.3", "kimi-k2.6"},
		"go-messages":  {"qwen3.8-flash", "minimax-m3"},
		"go-responses": {"gpt-6-luna", "grok-4.6"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			res, err := liveGo(t, goModel(t, candidates...)).Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "What is the weather in Paris? Use the tool."},
				llmprovider.FunctionCallItem{CallID: "call_rt_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
				llmprovider.FunctionCallOutputItem{CallID: "call_rt_1", Output: `{"forecast":"sunny, 21C"}`},
			}})
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if text := strings.ToLower(res.OutputText()); !strings.Contains(text, "sunny") && !strings.Contains(text, "21") {
				t.Fatalf("reply %q does not use the tool result", res.OutputText())
			}
		})
	}
}

// TestLive_OpencodeToolChoices pins the tool choices on each Go route:
// "required" makes a call, and "none" makes none (0015-PLAN S7). Forced tool
// choice is BestEffort on OpenCode, and on the messages route no listed model
// honours both choices, so each choice there has its own model, measured on
// 2026-10-02: minimax-m2.7 honours required (10/10) and ignores none;
// qwen3.8-flash honours none (5/5) and refuses required
// (TestLive_OpencodeForcedChoiceRefused).
func TestLive_OpencodeToolChoices(t *testing.T) {
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	chat, responses := []string{"glm-5.3-flash", "glm-5.3", "kimi-k2.6"}, []string{"gpt-6-luna", "grok-4.6"}
	for name, perChoice := range map[string]map[llmprovider.ToolChoice][]string{
		"go-chat":      {llmprovider.ToolChoiceRequired: chat, llmprovider.ToolChoiceNone: chat},
		"go-messages":  {llmprovider.ToolChoiceRequired: {"minimax-m2.7"}, llmprovider.ToolChoiceNone: {"qwen3.8-flash"}},
		"go-responses": {llmprovider.ToolChoiceRequired: responses, llmprovider.ToolChoiceNone: responses},
	} {
		for _, tc := range []struct {
			choice   llmprovider.ToolChoice
			wantCall bool
		}{{llmprovider.ToolChoiceRequired, true}, {llmprovider.ToolChoiceNone, false}} {
			candidates := perChoice[tc.choice]
			t.Run(name+"/"+string(tc.choice), func(t *testing.T) {
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				req := userText("What is the weather in Paris?")
				req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, tc.choice
				res, err := liveGo(t, goModel(t, candidates...)).Generate(ctx, req)
				llmprovider.SkipIfTransient(t, err)
				if err != nil {
					t.Fatalf("Generate: %v", err)
				}
				called := false
				for _, item := range res.Output {
					if _, ok := item.(llmprovider.FunctionCallItem); ok {
						called = true
					}
				}
				if called != tc.wantCall {
					t.Fatalf("a call came back: %t, want %t (%+v)", called, tc.wantCall, res.Output)
				}
			})
		}
	}
}

// TestLive_OpencodeForcedChoiceRefused pins a measured refusal (2026-10-02):
// qwen3.8-flash on Go's messages route answers a forced tool choice with HTTP
// 400, which is why forced tool choice is BestEffort on OpenCode (0015-PLAN,
// deviation "OpenCode's forced tool choice"). It fails if the service starts
// accepting it, so the degradation can be removed.
func TestLive_OpencodeForcedChoiceRefused(t *testing.T) {
	tool := llmprovider.Tool{Name: "get_weather", Description: "Get the weather for a city", Schema: map[string]any{
		"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}}, "required": []string{"city"}}}
	model := goModel(t, "qwen3.8-flash")
	for _, choice := range []llmprovider.ToolChoice{llmprovider.ToolChoiceRequired, llmprovider.ForceTool(tool.Name)} {
		t.Run(string(choice), func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			req := userText("What is the weather in Paris?")
			req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, choice
			_, err := liveGo(t, model).Generate(ctx, req)
			llmprovider.SkipIfTransient(t, err)
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Fatalf("Generate = %v, want the refusal, an error matching ErrInvalidRequest", err)
			}
		})
	}
}
