package gemini

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's gemini_interactions_test.go (0015-PLAN S7). Each
// old method is the Request it sent.

// TestGeminiInteractions_Request: the request shape measured on 2026-09-27
// (MADR 0014 §1): system_instruction, typed steps, a thought step with the
// call's signature before it, a result keyed by call_id, the tool and its
// forced choice in generation_config, and store false.
func TestGeminiInteractions_Request(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	_, genErr := p.Generate(context.Background(), withTool(items(
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."},
		user("weather?"),
		llmprovider.FunctionCallItem{CallID: "call_1", Name: "get_weather", Arguments: `{"city":"Paris"}`, Signature: "sig-1"},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: "sunny"}), weatherTool))
	path, body := c.last()
	if path != "/interactions" {
		t.Fatalf("path = %q, want /interactions", path)
	}
	if got := fmt.Sprint(stepTypes(body)); got != "[user_input thought function_call function_result]" {
		t.Errorf("steps = %s", got)
	}
	steps, _ := body["input"].([]any)
	if len(steps) == 4 {
		thought, call, result := steps[1].(map[string]any), steps[2].(map[string]any), steps[3].(map[string]any)
		if thought["signature"] != "sig-1" || call["id"] != "call_1" || fmt.Sprint(call["arguments"]) != "map[city:Paris]" ||
			result["call_id"] != "call_1" || result["name"] != "get_weather" || result["result"] != "sunny" {
			t.Errorf("thought %v, call %v, result %v", thought, call, result)
		}
	}
	gen := generationConfig(body)
	if body["system_instruction"] != "Be brief." || body["store"] != false || body["model"] != "gemini-3.7-flash" {
		t.Errorf("system_instruction %v, store %v, model %v", body["system_instruction"], body["store"], body["model"])
	}
	if fmt.Sprint(gen["tool_choice"]) != "map[allowed_tools:map[mode:any tools:[get_weather]]]" {
		t.Errorf("tool_choice = %v", gen["tool_choice"])
	}
	tools, _ := body["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["type"] != "function" || tools[0].(map[string]any)["name"] != "get_weather" {
		t.Errorf("tools = %v", body["tools"])
	}
	if genErr != nil {
		t.Errorf("Generate: %v", genErr)
	}
}

// TestGeminiInteractions_SyntheticCallCarriesPlaceholder: a call Gemini did not
// issue replays after a thought step with the placeholder signature.
func TestGeminiInteractions_SyntheticCallCarriesPlaceholder(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	_, _ = p.Generate(context.Background(), items(user("q"),
		llmprovider.FunctionCallItem{CallID: "c", Name: "f", Arguments: "{}"}, llmprovider.FunctionCallOutputItem{CallID: "c", Output: "r"}))
	_, body := c.last()
	steps, _ := body["input"].([]any)
	if len(steps) < 2 || steps[1].(map[string]any)["signature"] != skipThoughtSignature {
		t.Fatalf("steps = %v, want a placeholder thought before the call", steps)
	}
}

// TestGeminiInteractions_Response: a thought summary is reasoning, its
// signature rides on the calls after it, arguments are compact JSON, and the
// interaction id is the ID.
func TestGeminiInteractions_Response(t *testing.T) {
	srv, _ := interactionServer(t, `{"id":"v1_int_9","status":"requires_action","steps":[
{"type":"thought","signature":"sig-9","summary":[{"type":"text","text":"Look up the weather."}]},
{"type":"function_call","id":"call_7","name":"get_weather","arguments":{"city": "Paris"}},
{"type":"function_call","id":"call_8","name":"get_weather","arguments":{"city":"Rome"}}]}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	res, err := p.Generate(context.Background(), withTool(text("q"), weatherTool))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want := []llmprovider.Item{
		llmprovider.ReasoningItem{Text: "Look up the weather."},
		llmprovider.FunctionCallItem{CallID: "call_7", Name: "get_weather", Arguments: `{"city":"Paris"}`, Signature: "sig-9"},
		llmprovider.FunctionCallItem{CallID: "call_8", Name: "get_weather", Arguments: `{"city":"Rome"}`, Signature: "sig-9"},
	}
	if res.ID != "v1_int_9" || fmt.Sprint(res.Output) != fmt.Sprint(want) {
		t.Fatalf("ID %q, output %+v; want v1_int_9, %+v", res.ID, res.Output, want)
	}
}

// TestGeminiInteractions_Thinking: summaries on every thinking call; an effort
// becomes thinking_level, xhigh as high; gemini-2.5-flash-lite takes only high.
// Each case runs with the effort set at construction, as the old
// WithReasoningEffort did, and on the request.
func TestGeminiInteractions_Thinking(t *testing.T) {
	for _, tc := range []struct {
		model  string
		effort llmprovider.Effort
		want   string
	}{
		{"gemini-3.7-flash", "", "<nil>"},
		{"gemini-3.7-flash", llmprovider.EffortLow, "low"},
		{"gemini-3.7-flash", llmprovider.EffortXHigh, "high"},
		{"gemini-2.5-flash-lite", llmprovider.EffortLow, "<nil>"},
		{"gemini-2.5-flash-lite", llmprovider.EffortHigh, "high"},
	} {
		for _, where := range []string{"construction", "request"} {
			srv, c := interactionServer(t, interactionText)
			var p llmprovider.Provider
			req := text("hi")
			if where == "construction" {
				p = build(t, apiKey(srv.URL, tc.model, llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: tc.effort}))...)
				thinking(req)
			} else {
				p = build(t, apiKey(srv.URL, tc.model)...)
				req.Reasoning = &llmprovider.Reasoning{Effort: tc.effort}
			}
			_, _ = p.Generate(context.Background(), req)
			_, body := c.last()
			gen := generationConfig(body)
			if gen["thinking_summaries"] != "auto" || fmt.Sprint(gen["thinking_level"]) != tc.want {
				t.Errorf("%s/%s at %s: generation_config = %v, want summaries auto and level %s",
					tc.model, tc.effort, where, gen, tc.want)
			}
		}
	}
}

// TestGeminiInteractions_PlainCallSendsNoExtras: no thinking, system or tool
// fields on a plain call.
func TestGeminiInteractions_PlainCallSendsNoExtras(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	_, _ = p.Generate(context.Background(), text("hi"))
	_, body := c.last()
	gen := generationConfig(body)
	for _, k := range []string{"thinking_summaries", "thinking_level", "tool_choice"} {
		if v, ok := gen[k]; ok {
			t.Errorf("generation_config.%s = %v, want absent", k, v)
		}
	}
	for _, k := range []string{"system_instruction", "tools", "previous_interaction_id"} {
		if v, ok := body[k]; ok {
			t.Errorf("%s = %v, want absent", k, v)
		}
	}
}

// TestGeminiInteractions_Status: incomplete is an *APIError of kind
// ErrIncomplete (as at max_output_tokens), failed is retryable.
func TestGeminiInteractions_Status(t *testing.T) {
	srv, _ := interactionServer(t, `{"id":"v1_x","status":"incomplete","steps":[{"type":"model_output","content":[{"type":"text","text":"1, 2, "}]}]}`)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	_, err := p.Generate(context.Background(), text("count"))
	var inc *llmprovider.APIError
	if !errors.As(err, &inc) || !errors.Is(inc.Kind, llmprovider.ErrIncomplete) || inc.Reason != "incomplete" {
		t.Errorf("incomplete: err = %v, want an *APIError of kind ErrIncomplete", err)
	}
	srv2, _ := interactionServer(t, `{"id":"v1_y","status":"failed","steps":[]}`)
	p2 := build(t, apiKey(srv2.URL, "gemini-3.7-flash")...)
	if _, err := p2.Generate(context.Background(), text("hi")); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("failed: err = %v, want ErrProviderUnavailable", err)
	}
}

// TestGemini_ContinueNeedsStore: without WithStore(true) nothing is stored, so
// a continuation is refused before any request. It was
// TestGemini_ContinueNeedsStore with ErrInvalidRequest; continuation is now
// Unsupported there, so the refusal is ErrUnsupported (0015-MADR D4).
func TestGemini_ContinueNeedsStore(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	if got := p.Capabilities().Continuation; got != llmprovider.Unsupported {
		t.Errorf("Continuation = %v, want Unsupported", got)
	}
	req := text("more")
	req.PreviousResponseID = "v1_prev"
	_, err := p.Generate(context.Background(), req)
	if path, _ := c.last(); !errors.Is(err, llmprovider.ErrUnsupported) || path != "" {
		t.Fatalf("err = %v after request %q, want ErrUnsupported and none", err, path)
	}
}

// TestGemini_ContinueChainsWhenStored: with WithStore(true), a continuation
// sends only the new items, previous_interaction_id and store true.
func TestGemini_ContinueChainsWhenStored(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash", WithStore(true))...)
	if got := p.Capabilities().Continuation; got != llmprovider.Supported {
		t.Errorf("Continuation = %v, want Supported", got)
	}
	req := text("more")
	req.PreviousResponseID = "v1_prev"
	res, err := p.Generate(context.Background(), req)
	path, body := c.last()
	if path != "/interactions" || body["previous_interaction_id"] != "v1_prev" || body["store"] != true ||
		fmt.Sprint(stepTypes(body)) != "[user_input]" {
		t.Fatalf("path %q, body %v", path, body)
	}
	if err != nil || res.ID != "v1_int_1" {
		t.Fatalf("Generate = %+v, %v", res, err)
	}
}
