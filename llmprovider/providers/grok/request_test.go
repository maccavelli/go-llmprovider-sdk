package grok

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Request fields the old methods could not express, and New's options
// (0015-PLAN S7).

func TestNew_RefusesAForeignOption(t *testing.T) {
	_, err := New(llmprovider.WithAPIKey("k"), llmprovider.ScopedOption(llmprovider.ProviderOpenAI, "openai.WithStore", true))
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

// TestNew_AcceptsEverySourceKind: Grok takes a key, an OAuth session and the
// Grok CLI's login (MADR 0012 §5).
func TestNew_AcceptsEverySourceKind(t *testing.T) {
	for name, src := range map[string]llmprovider.TokenSource{
		"static":  llmprovider.NewStaticToken("k"),
		"command": llmprovider.NewCommandToken("true"),
		"oauth":   grokSession(),
		"vendor":  &auth.VendorCLISession{Provider: llmprovider.ProviderGrok, Path: "unused"},
	} {
		if _, err := New(llmprovider.WithTokenSource(src)); err != nil {
			t.Errorf("%s: New: %v", name, err)
		}
	}
}

// TestGenerate_RequestFields: the model, output limit and instructions a
// request names, and each tool choice.
func TestGenerate_RequestFields(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL, "grok-base", llmprovider.WithMaxTokens(500))...)
	req := items(llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."}, user("hi"))
	req.Model, req.MaxOutputTokens, req.Instructions = "grok-other", 77, "Answer in French."
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if body["model"] != "grok-other" || body["max_output_tokens"] != float64(77) {
		t.Errorf("model %v, max_output_tokens %v; want grok-other, 77", body["model"], body["max_output_tokens"])
	}
	if got := fmt.Sprint(body["input"]); got != "[map[content:Answer in French. role:system] map[content:Be brief. role:system] map[content:hi role:user]]" {
		t.Errorf("input = %s, want the instructions as a leading system message", got)
	}
	if _, ok := body["instructions"]; ok {
		t.Errorf("instructions = %v, want absent", body["instructions"])
	}

	for _, tc := range []struct {
		choice llmprovider.ToolChoice
		want   string
	}{
		{llmprovider.ToolChoiceAuto, "<nil>"},
		{"", "<nil>"},
		{llmprovider.ToolChoiceRequired, "required"},
		{llmprovider.ToolChoiceNone, "none"},
		{llmprovider.ForceTool("get_weather"), "map[name:get_weather type:function]"},
	} {
		req := text("hi")
		req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}},
			{Name: "get_time", Schema: map[string]any{"type": "object"}}}
		req.ToolChoice = tc.choice
		if _, err := p.Generate(context.Background(), req); err != nil {
			t.Fatalf("%q: Generate: %v", tc.choice, err)
		}
		if got := fmt.Sprint(body["tool_choice"]); got != tc.want {
			t.Errorf("%q: tool_choice = %s, want %s", tc.choice, got, tc.want)
		}
		if tools, _ := body["tools"].([]any); len(tools) != 2 {
			t.Errorf("%q: %d tools sent, want 2", tc.choice, len(tools))
		}
	}
}

// TestGenerate_Reasoning: a request's effort wins over WithReasoning's and is
// clamped to the model's menu; an empty effort takes WithReasoning's, else
// high; a budget is not sent; with neither, nothing is sent. Each case that
// the old WithReasoningEffort expressed runs both ways.
func TestGenerate_Reasoning(t *testing.T) {
	low, high, medium := &llmprovider.Reasoning{Effort: llmprovider.EffortLow},
		&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}, &llmprovider.Reasoning{Effort: llmprovider.EffortMedium}
	for _, tc := range []struct {
		name     string
		model    string
		def, req *llmprovider.Reasoning
		want     any
	}{
		{"none", "grok-4.5", nil, nil, nil},
		{"default only", "grok-4.5", low, nil, "low"},
		{"request only", "grok-4.5", nil, low, "low"},
		{"request wins", "grok-4.5", low, high, "high"},
		{"empty request effort takes the default", "grok-4.5", medium, &llmprovider.Reasoning{}, "medium"},
		{"no effort anywhere sends high", "grok-4.6", nil, &llmprovider.Reasoning{}, "high"},
		{"a budget alone sends high", "grok-4.6", nil, &llmprovider.Reasoning{Budget: 4096}, "high"},
		{"request clamped to the menu", "grok-3-mini", nil, medium, "low"},
		{"a model without a menu", "grok-4", nil, high, nil},
	} {
		var body map[string]any
		srv := captureServer(t, &body, okResponse)
		var opts []llmprovider.Option
		if tc.def != nil {
			opts = append(opts, llmprovider.WithReasoning(tc.def))
		}
		req := text("hi")
		req.Reasoning = tc.req
		if _, err := build(t, apiKey(srv.URL, tc.model, opts...)...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.name, err)
		}
		if got := effortOf(body); got != tc.want {
			t.Errorf("%s: reasoning.effort = %v, want %v (body %v)", tc.name, got, tc.want, body)
		}
		if r, _ := body["reasoning"].(map[string]any); len(r) > 1 {
			t.Errorf("%s: reasoning = %v, want only an effort", tc.name, r)
		}
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request never reaches the
// service.
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	req := text("hi")
	req.ToolChoice = "sometimes"
	if _, err := build(t, apiKey(srv.URL, "grok-4.6")...).Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if body != nil {
		t.Errorf("a request was sent: %v", body)
	}
}
