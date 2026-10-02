package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's thinking_test.go, thinking_wire_test.go,
// generatecontent_wire_test.go, claude_system_test.go,
// reasoning_replay_test.go, metadata_request_test.go and provider_test.go:
// the OpenCode cases, and the generateContent wire as OpenCode's google route
// sends it (0015-PLAN S7).

// dynamicThinkingBudget is generateContent's "let the model size it" budget
// (llmprovider's dynamicGeminiThinkingBudget).
const dynamicThinkingBudget = -1

// TestGoogleRouteThinking_RequestBody verifies a thinkingConfig is nested in
// generationConfig with the configured budget on OpenCode's google route, and
// that the default path omits it.
func TestGoogleRouteThinking_RequestBody(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxGoogle)
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gemini-x", WithRoute(RouteGoogle),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Budget: 1234}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	gc, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing generationConfig: %v", body)
	}
	tc, ok := gc["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing thinkingConfig: %v", gc)
	}
	if b := tc["thinkingBudget"].(float64); int(b) != 1234 {
		t.Errorf("thinkingBudget = %v, want 1234", b)
	}

	// No reasoning on the request or the provider: no thinkingConfig.
	plain := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gemini-x", WithRoute(RouteGoogle))...)
	body = nil
	if _, err := plain.Generate(context.Background(), text("hi")); err != nil {
		t.Fatal(err)
	}
	gc2 := body["generationConfig"].(map[string]any)
	if _, present := gc2["thinkingConfig"]; present {
		t.Errorf("non-thinking request must not contain thinkingConfig: %v", gc2)
	}
}

// TestGoogleRouteThinking_DynamicBudgetDefault verifies an unset budget maps to
// -1 (dynamic) on OpenCode's google route.
func TestGoogleRouteThinking_DynamicBudgetDefault(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxGoogle)
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gemini-x", WithRoute(RouteGoogle))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatal(err)
	}
	gc := body["generationConfig"].(map[string]any)
	tc := gc["thinkingConfig"].(map[string]any)
	if b := tc["thinkingBudget"].(float64); int(b) != dynamicThinkingBudget {
		t.Errorf("thinkingBudget = %v, want %d (dynamic)", b, dynamicThinkingBudget)
	}
}

func assertThinkingFields(t *testing.T, body map[string]any, want map[string]any) {
	t.Helper()
	for k, v := range want {
		got, present := body[k]
		switch {
		case v == nil && present:
			t.Errorf("%s = %v, want it absent", k, got)
		case v != nil && !reflect.DeepEqual(got, v):
			t.Errorf("%s = %v, want %v", k, got, v)
		}
	}
}

// TestThinkingWire_Gemini pins MADR 0013 Q1 on the generateContent wire, which
// only OpenCode's google route speaks since MADR 0014: "low" is thinkingLevel
// on Gemini 3 and later and a 1024 budget on 2.x (thinkingLevel is HTTP 400
// there); other efforts keep the dynamic budget; a budget wins.
func TestThinkingWire_Gemini(t *testing.T) {
	for _, tc := range []struct {
		model  string
		effort llmprovider.Effort
		budget int
		want   map[string]any
	}{
		{"gemini-3.7-flash", llmprovider.EffortLow, 0, map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingLevel": "low"}}},
		{"gemini-2.5-flash", llmprovider.EffortLow, 0, map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingBudget": float64(1024)}}},
		{"gemini-3.7-flash", "", 0, map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingBudget": float64(-1)}}},
		{"gemini-3.7-flash", llmprovider.EffortHigh, 0, map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingBudget": float64(-1)}}},
		{"gemini-2.5-flash", llmprovider.EffortLow, 512, map[string]any{"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingBudget": float64(512)}}},
	} {
		t.Run(tc.model+"/"+string(tc.effort), func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, fxGoogle)
			p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model,
				llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: tc.effort, Budget: tc.budget}))...)
			if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			gc, _ := body["generationConfig"].(map[string]any)
			assertThinkingFields(t, gc, tc.want)
		})
	}
}

// TestThinkingWire_OpencodeRoutes pins the same shapes on OpenCode's messages
// and google routes, which forward to the same upstream APIs. Each case runs
// with the effort set at construction, as the old WithReasoningEffort did,
// and on the request.
func TestThinkingWire_OpencodeRoutes(t *testing.T) {
	for _, tc := range []struct {
		gateway        llmprovider.ProviderID
		model, fixture string
		inGenCfg       bool
		want           map[string]any
	}{
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5", fxMessages, false, map[string]any{
			"thinking": map[string]any{"type": "adaptive"}, "output_config": map[string]any{"effort": "low"}}},
		{llmprovider.ProviderOpencodeGo, "qwen3.8-flash", fxMessages, false, map[string]any{
			"thinking": map[string]any{"type": "enabled", "budget_tokens": float64(1024)}}},
		{llmprovider.ProviderOpencodeZen, "gemini-3.8-flash", fxGoogle, true, map[string]any{
			"thinkingConfig": map[string]any{"includeThoughts": true, "thinkingLevel": "low"}}},
	} {
		for _, where := range []string{"construction", "request"} {
			t.Run(string(tc.gateway)+"/"+tc.model+"/"+where, func(t *testing.T) {
				var body map[string]any
				srv := captureServer(t, &body, tc.fixture)
				req := text("hi")
				var opts []llmprovider.Option
				if where == "construction" {
					opts = append(opts, effort(llmprovider.EffortLow))
					thinking(req)
				} else {
					req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
				}
				if _, err := build(t, tc.gateway, apiKey(srv.URL, tc.model, opts...)...).Generate(context.Background(), req); err != nil {
					t.Fatalf("Generate: %v", err)
				}
				if tc.inGenCfg {
					body, _ = body["generationConfig"].(map[string]any)
				}
				assertThinkingFields(t, body, tc.want)
			})
		}
	}
}

// googleBody returns the request body OpenCode's google route sends for one
// call. It was llmprovider's geminiWireBodies.
func googleBody(t *testing.T, reasons bool, input ...llmprovider.Item) map[string]any {
	t.Helper()
	var body map[string]any
	srv := captureServer(t, &body, `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`)
	req := items(input...)
	if reasons {
		thinking(req)
	}
	if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gemini-3.7-flash")...).Generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	return body
}

// TestGeminiThinking_AsksForThoughts: a reasoning call sets includeThoughts,
// as OpenCode's client does (transform.ts:1280-1288).
func TestGeminiThinking_AsksForThoughts(t *testing.T) {
	gen, _ := googleBody(t, true, user("hi"))["generationConfig"].(map[string]any)
	tc, _ := gen["thinkingConfig"].(map[string]any)
	if tc["includeThoughts"] != true {
		t.Errorf("thinkingConfig = %v, want includeThoughts true", tc)
	}
}

// TestGeminiThinking_NotAskedWithoutThinking: a plain call sends no
// thinkingConfig at all.
func TestGeminiThinking_NotAskedWithoutThinking(t *testing.T) {
	gen, _ := googleBody(t, false, user("hi"))["generationConfig"].(map[string]any)
	if _, ok := gen["thinkingConfig"]; ok {
		t.Errorf("generationConfig = %v, want no thinkingConfig", gen)
	}
}

// TestGemini_SystemInstruction: system items go to systemInstruction and
// never become a model turn.
func TestGemini_SystemInstruction(t *testing.T) {
	body := googleBody(t, false,
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Reply in French."},
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."},
		user("hello"))
	si, _ := body["systemInstruction"].(map[string]any)
	parts, _ := si["parts"].([]any)
	if len(parts) != 1 || parts[0].(map[string]any)["text"] != "Reply in French.\n\nBe brief." {
		t.Errorf("systemInstruction = %v", body["systemInstruction"])
	}
	contents, _ := body["contents"].([]any)
	if len(contents) != 1 || contents[0].(map[string]any)["role"] != "user" {
		t.Errorf("contents = %v, want only the user turn", contents)
	}
}

// TestGemini_NoSystemInstructionWithoutSystem: no system item, no field.
func TestGemini_NoSystemInstructionWithoutSystem(t *testing.T) {
	if v, ok := googleBody(t, false, user("hello"))["systemInstruction"]; ok {
		t.Errorf("systemInstruction = %v, want absent", v)
	}
}

// TestOpencodeMessages_SystemMessageIsTopLevel: system items are the Messages
// system field, joined, and never a message. It read messagesBody directly.
func TestOpencodeMessages_SystemMessageIsTopLevel(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxMessages)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "qwen3.8-flash", WithRoute(RouteMessages))...)
	if _, err := p.Generate(context.Background(), items(
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Always answer in French."},
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."},
		user("Say hello."))); err != nil {
		t.Fatal(err)
	}
	if body["system"] != "Always answer in French.\n\nBe brief." {
		t.Errorf("system = %#v, want the two system items joined", body["system"])
	}
	if got, _ := json.Marshal(body["messages"]); string(got) != `[{"content":"Say hello.","role":"user"}]` {
		t.Errorf("messages = %s, want only the user turn", got)
	}
}

// replayServer serves a metadata document declaring interleaved for one Go
// model, and records the chat request body.
func replayServer(t *testing.T, interleaved string) (*httptest.Server, *map[string]any) {
	t.Helper()
	enableMetadata(t)
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api.json") {
			_, _ = w.Write([]byte(`{"opencode-go":{"models":{"kimi-k2.6":{"id":"kimi-k2.6"` + interleaved + `}}}}`))
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"sunny"}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &body
}

func replayConversation() *llmprovider.Request {
	return items(
		user("weather?"),
		llmprovider.ReasoningItem{Text: "Call the tool."},
		llmprovider.FunctionCallItem{CallID: "call_1", Name: "get_weather", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: "sunny"},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "It is sunny."},
		user("thanks"),
	)
}

// TestOpencodeChat_ReplaysInterleavedReasoning: the declared field is set on
// every assistant message, with the reasoning that preceded it or "" (as
// OpenCode's client does, transform.ts:321-349).
func TestOpencodeChat_ReplaysInterleavedReasoning(t *testing.T) {
	srv, body := replayServer(t, `,"interleaved":{"field":"reasoning_content"}`)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "kimi-k2.6",
		llmprovider.WithModelMetadataURL(srv.URL+"/api.json"), WithRoute(RouteChatCompletions))...)
	if _, err := p.Generate(context.Background(), replayConversation()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var assistants []any
	for _, m := range (*body)["messages"].([]any) {
		if msg := m.(map[string]any); msg["role"] == string(llmprovider.RoleAssistant) {
			assistants = append(assistants, msg["reasoning_content"])
		}
	}
	if len(assistants) != 2 || assistants[0] != "Call the tool." || assistants[1] != "" {
		t.Fatalf("assistant reasoning_content = %#v, want [\"Call the tool.\" \"\"]", assistants)
	}
}

// TestOpencodeChat_NoReplayWithoutInterleaved: a model the metadata does not
// mark interleaved gets no reasoning field.
func TestOpencodeChat_NoReplayWithoutInterleaved(t *testing.T) {
	srv, body := replayServer(t, "")
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "kimi-k2.6",
		llmprovider.WithModelMetadataURL(srv.URL+"/api.json"), WithRoute(RouteChatCompletions))...)
	if _, err := p.Generate(context.Background(), replayConversation()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, m := range (*body)["messages"].([]any) {
		if _, ok := m.(map[string]any)["reasoning_content"]; ok {
			t.Fatalf("message %v carries reasoning_content for a non-interleaved model", m)
		}
	}
}

// TestChatReasoningEffort_FailureBackoff pins MADR 0013 A6: with the metadata
// host failing, three chat-route reasoning calls make one fetch, not three.
func TestChatReasoningEffort_FailureBackoff(t *testing.T) {
	enableMetadata(t)
	meta, hits := metadataServer(t, http.StatusServiceUnavailable, "")
	var body map[string]any
	srv := captureServer(t, &body, fxChat)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "glm-5.3-flash",
		llmprovider.WithModelMetadataURL(meta.URL), effort(llmprovider.EffortLow))...)
	for range 3 {
		if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if got, ok := body["reasoning_effort"]; ok {
			t.Fatalf("reasoning_effort = %v with metadata down, want none", got)
		}
	}
	// One request makes two lookups (route, effort); the failure is cached.
	if n := hits.Load(); n != 1 {
		t.Errorf("3 reasoning calls made %d metadata fetches, want 1", n)
	}
}

// TestChatReasoningEffort_LookupHasDeadline pins MADR 0013 A6: the lookup
// inside a request carries its own deadline of at most 5 s, whatever the
// caller's context.
func TestChatReasoningEffort_LookupHasDeadline(t *testing.T) {
	enableMetadata(t)
	meta, _ := metadataServer(t, http.StatusOK, `{"opencode-go":{"models":{"glm-5.3-flash":{"id":"glm-5.3-flash","reasoning":true}}}}`)
	var body map[string]any
	srv := captureServer(t, &body, fxChat)
	rec := &deadlineTransport{}
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "glm-5.3-flash", llmprovider.WithModelMetadataURL(meta.URL+"/api.json"),
		effort(llmprovider.EffortLow), llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...)
	if _, err := p.Generate(context.Background(), thinking(text("hi"))); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	left, ok := rec.left("GET /api.json")
	switch {
	case !ok:
		t.Fatal("no metadata request was made")
	case left < 0:
		t.Error("metadata lookup ran without a deadline, want one within 5s")
	case left > 5*time.Second:
		t.Errorf("metadata lookup had %v left, want at most 5s", left.Round(time.Second))
	}
}

// TestNew_RoutesResolved was TestNewOpencode_RoutesResolved: the route is
// resolved when the provider is built, so a misroute is visible before any
// request.
func TestNew_RoutesResolved(t *testing.T) {
	p := build(t, llmprovider.ProviderOpencodeZen, llmprovider.WithModel("claude-sonnet-5"))
	if got := p.(*provider).route; got != RouteMessages {
		t.Errorf("route = %q, want %q", got, RouteMessages)
	}
}

// TestGenerate_InstructionsAreALeadingSystemItem: on every route, a Request's
// Instructions go before the system items, in each route's system form.
func TestGenerate_InstructionsAreALeadingSystemItem(t *testing.T) {
	for _, tc := range []struct {
		model, fixture string
		system         func(map[string]any) string
	}{
		{"gpt-5.5", fxResponses, func(b map[string]any) string { return fmt.Sprint(b["input"]) }},
		{"claude-sonnet-5", fxMessages, func(b map[string]any) string { return fmt.Sprint(b["system"]) }},
		{"gemini-3.7-flash", fxGoogle, func(b map[string]any) string { return fmt.Sprint(b["systemInstruction"]) }},
		{deepSeekV4Pro, fxChat, func(b map[string]any) string { return fmt.Sprint(b["messages"]) }},
	} {
		var body map[string]any
		srv := captureServer(t, &body, tc.fixture)
		req := items(llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "SECOND"}, user("hi"))
		req.Instructions = "FIRST"
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model)...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.model, err)
		}
		got := tc.system(body)
		if i, j := strings.Index(got, "FIRST"), strings.Index(got, "SECOND"); i < 0 || j < 0 || i > j {
			t.Errorf("%s: system = %s, want FIRST before SECOND", tc.model, got)
		}
	}
}
