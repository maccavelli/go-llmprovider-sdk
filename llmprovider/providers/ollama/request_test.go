package ollama

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Request fields the old methods could not express, and New's refusals
// (0015-PLAN S7).

// TestNew_RefusesAnOAuthSession: R16 refuses a source Ollama does not accept
// (0016-MADR D10).
func TestNew_RefusesAnOAuthSession(t *testing.T) {
	for name, src := range map[string]llmprovider.TokenSource{
		"oauth":  &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a"},
		"vendor": &auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "unused"},
	} {
		if _, err := New(llmprovider.WithTokenSource(src)); !errors.Is(err, llmprovider.ErrUnsupported) {
			t.Errorf("%s: err = %v, want ErrUnsupported", name, err)
		}
	}
}

func TestNew_RefusesAForeignOption(t *testing.T) {
	_, err := New(llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithOrganization", "o"))
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

// TestGenerate_RequestFields: the model, output limit and instructions a
// request names, and each tool choice offering every tool, forcing none.
func TestGenerate_RequestFields(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxOllamaChat)
	p := build(t, local(srv.URL, "base-model", llmprovider.WithMaxTokens(500))...)
	req := &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."}, user("hi")}}
	req.Model, req.MaxOutputTokens, req.Instructions = "other-model", 77, "Answer in French."
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if body["model"] != "other-model" || body["max_tokens"] != float64(77) {
		t.Errorf("model %v, max_tokens %v; want other-model, 77", body["model"], body["max_tokens"])
	}
	if msgs, _ := body["messages"].([]any); len(msgs) != 3 || fmt.Sprint(msgs[0]) != "map[content:Answer in French. role:system]" {
		t.Errorf("messages = %v, want the instructions as a leading system message", body["messages"])
	}

	for _, choice := range []llmprovider.ToolChoice{llmprovider.ToolChoiceAuto, llmprovider.ToolChoiceRequired,
		llmprovider.ToolChoiceNone, llmprovider.ForceTool("get_weather")} {
		req := text("hi")
		req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}},
			{Name: "get_time", Schema: map[string]any{"type": "object"}}}
		req.ToolChoice = choice
		if _, err := p.Generate(context.Background(), req); err != nil {
			t.Fatalf("%q: Generate: %v", choice, err)
		}
		if got, ok := body["tool_choice"]; ok {
			t.Errorf("%q: tool_choice = %v, want none sent", choice, got)
		}
		want := 2
		if choice == llmprovider.ToolChoiceNone {
			want = 0 // honoured by sending no tools (0020-MADR F40)
		}
		if tools, _ := body["tools"].([]any); len(tools) != want {
			t.Errorf("%q: %d tools sent, want %d", choice, len(tools), want)
		}
	}
}

// TestGenerate_Reasoning: a request's effort wins over WithReasoning's; an
// empty one takes WithReasoning's; a budget alone sends medium, the budget not
// sent; WithReasoning alone reasons on every request.
func TestGenerate_Reasoning(t *testing.T) {
	for _, tc := range []struct {
		name     string
		def, req *llmprovider.Reasoning
		want     any
	}{
		{"request wins", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}, "high"},
		{"empty takes the default", &llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}, &llmprovider.Reasoning{}, maxEffort},
		{"budget alone", nil, &llmprovider.Reasoning{Budget: 4096}, "medium"},
		{"default only", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, nil, "low"},
	} {
		var body map[string]any
		srv := captureServer(t, &body, fxOllamaChat)
		var opts []llmprovider.Option
		if tc.def != nil {
			opts = append(opts, llmprovider.WithReasoning(tc.def))
		}
		req := text("hi")
		req.Reasoning = tc.req
		if _, err := build(t, local(srv.URL, "m", opts...)...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.name, err)
		}
		if body["reasoning_effort"] != tc.want {
			t.Errorf("%s: reasoning_effort = %v, want %v", tc.name, body["reasoning_effort"], tc.want)
		}
		if _, sent := body["reasoning"]; sent {
			t.Errorf("%s: reasoning = %v, want no budget sent", tc.name, body["reasoning"])
		}
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request never reaches the
// instance.
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxOllamaChat)
	req := text("hi")
	req.ToolChoice = "sometimes"
	if _, err := build(t, local(srv.URL, "m")...).Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if body != nil {
		t.Errorf("a request was sent: %v", body)
	}
}
