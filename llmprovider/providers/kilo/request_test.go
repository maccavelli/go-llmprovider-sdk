package kilo

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

// TestNew_RefusesAnOAuthSession: R16 refuses a source Kilo does not accept
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

// TestNew_KilosOptionsAreKilos: the three options reach Kilo, and another
// provider's is refused.
func TestNew_KilosOptionsAreKilos(t *testing.T) {
	if _, err := New(WithOrganization("o"), WithCapabilities(paramTools), WithDataCollection(true)); err != nil {
		t.Errorf("Kilo's options: %v", err)
	}
	_, err := New(llmprovider.ScopedOption(llmprovider.ProviderGrok, "grok.WithStore", true))
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a grok option: err = %v, want ErrInvalidRequest", err)
	}
}

// countingSource returns key-1, key-2, … on successive calls.
type countingSource struct{ n int }

func (s *countingSource) Token(context.Context) (llmprovider.Token, error) {
	s.n++
	return llmprovider.Token{Value: fmt.Sprintf("key-%d", s.n), Type: llmprovider.TokenAPIKey}, nil
}

// TestGenerate_ReadsTheKeyPerRequest: a rotating source's new key is used.
func TestGenerate_ReadsTheKeyPerRequest(t *testing.T) {
	srv, seen := headerServer(t)
	p := build(t, llmprovider.WithTokenSource(&countingSource{}), llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
	}
	reqs := seen()
	if got := []string{reqs[0].Get("Authorization"), reqs[1].Get("Authorization")}; fmt.Sprint(got) != "[Bearer key-1 Bearer key-2]" {
		t.Fatalf("Authorization sent %v, want [Bearer key-1 Bearer key-2]", got)
	}
}

// TestGenerate_RequestFields: the model, output limit and instructions a
// request names.
func TestGenerate_RequestFields(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxKiloChat)
	req := &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."}, user("hi")}}
	req.Model, req.MaxOutputTokens, req.Instructions = "other/model", 77, "Answer in French."
	if _, err := build(t, apiKey(srv.URL, "base/model", llmprovider.WithMaxTokens(500))...).Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if body["model"] != "other/model" || body["max_tokens"] != float64(77) {
		t.Errorf("model %v, max_tokens %v; want other/model, 77", body["model"], body["max_tokens"])
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 3 || fmt.Sprint(msgs[0]) != "map[content:Answer in French. role:system]" {
		t.Errorf("messages = %v, want the instructions as a leading system message", msgs)
	}
}

// TestGenerate_ToolChoices: every tool is offered; each choice is sent when
// the model accepts tool_choice, and none when it does not.
func TestGenerate_ToolChoices(t *testing.T) {
	for _, tc := range []struct {
		choice llmprovider.ToolChoice
		caps   []string // nil: unknown
		want   string
	}{
		{llmprovider.ToolChoiceAuto, nil, "<nil>"},
		{llmprovider.ToolChoiceRequired, nil, "required"},
		{llmprovider.ToolChoiceNone, nil, "none"},
		{llmprovider.ForceTool("get_weather"), nil, "map[function:map[name:get_weather] type:function]"},
		{llmprovider.ForceTool("get_weather"), []string{paramTools}, "<nil>"},
		{llmprovider.ToolChoiceRequired, []string{paramTools}, "<nil>"},
	} {
		var body map[string]any
		srv := captureServer(t, &body, fxKiloChat)
		var opts []llmprovider.Option
		if tc.caps != nil {
			opts = append(opts, WithCapabilities(tc.caps...))
		}
		req := text("hi")
		req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}},
			{Name: "get_time", Schema: map[string]any{"type": "object"}}}
		req.ToolChoice = tc.choice
		if _, err := build(t, apiKey(srv.URL, "m", opts...)...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%q: Generate: %v", tc.choice, err)
		}
		if got := fmt.Sprint(body["tool_choice"]); got != tc.want {
			t.Errorf("%q with caps %v: tool_choice = %s, want %s", tc.choice, tc.caps, got, tc.want)
		}
		if tools, _ := body["tools"].([]any); len(tools) != 2 {
			t.Errorf("%q: %d tools sent, want 2", tc.choice, len(tools))
		}
	}
}

// TestGenerate_Reasoning: a request's effort wins over WithReasoning's; a
// budget alone is the model's default reasoning, the budget not sent.
func TestGenerate_Reasoning(t *testing.T) {
	for _, tc := range []struct {
		name     string
		def, req *llmprovider.Reasoning
		want     string
	}{
		{"request wins", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}, "map[effort:high]"},
		{"budget alone", nil, &llmprovider.Reasoning{Budget: 4096}, "map[enabled:true]"},
		{"budget with a default effort", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, &llmprovider.Reasoning{Budget: 4096}, "map[effort:low]"},
	} {
		var body map[string]any
		srv := captureServer(t, &body, fxKiloChat)
		var opts []llmprovider.Option
		if tc.def != nil {
			opts = append(opts, llmprovider.WithReasoning(tc.def))
		}
		req := text("hi")
		req.Reasoning = tc.req
		if _, err := build(t, apiKey(srv.URL, "m", opts...)...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.name, err)
		}
		if got := fmt.Sprint(body["reasoning"]); got != tc.want {
			t.Errorf("%s: reasoning = %s, want %s", tc.name, got, tc.want)
		}
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request never reaches the
// gateway.
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, fxKiloChat)
	req := text("hi")
	req.ToolChoice = "sometimes"
	if _, err := build(t, apiKey(srv.URL, "m")...).Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if body != nil {
		t.Errorf("a request was sent: %v", body)
	}
}
