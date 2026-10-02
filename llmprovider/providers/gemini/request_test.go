package gemini

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Request fields the old methods could not express, and New's refusals
// (0015-PLAN S7).

func TestNew_NeedsACredential(t *testing.T) {
	if _, err := New(llmprovider.WithModel("gemini-3.7-flash")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

// TestNew_RefusesAnOAuthSession: R16 refuses a source the service does not
// accept (0016-MADR D10).
func TestNew_RefusesAnOAuthSession(t *testing.T) {
	for name, src := range map[string]llmprovider.TokenSource{
		"oauth":  &llmprovider.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a"},
		"vendor": &llmprovider.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "unused"},
	} {
		if _, err := New(llmprovider.WithTokenSource(src)); !errors.Is(err, llmprovider.ErrUnsupported) {
			t.Errorf("%s: err = %v, want ErrUnsupported", name, err)
		}
	}
}

// TestNew_AcceptsAnEmptyKey: NewGemini("") built a provider, which the service
// then refused; New keeps that.
func TestNew_AcceptsAnEmptyKey(t *testing.T) {
	if _, err := New(llmprovider.WithAPIKey("")); err != nil {
		t.Fatalf("New with an empty key: %v", err)
	}
}

func TestNew_RefusesAForeignOption(t *testing.T) {
	_, err := New(llmprovider.WithAPIKey("k"), llmprovider.ScopedOption(llmprovider.ProviderOpenAI, "openai.WithStore", true))
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
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
	srv, c := interactionServer(t, interactionText)
	p := build(t, llmprovider.WithTokenSource(&countingSource{}), llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	got := []string{c.headers[0].Get(googleKeyHeader), c.headers[1].Get(googleKeyHeader)}
	if fmt.Sprint(got) != "[key-1 key-2]" {
		t.Fatalf("x-goog-api-key sent %v, want [key-1 key-2]", got)
	}
}

// TestGenerate_RequestFields: the model, output limit and instructions a
// request names, and each tool choice.
func TestGenerate_RequestFields(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-base", llmprovider.WithMaxTokens(500))...)
	req := items(llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."}, user("hi"))
	req.Model, req.MaxOutputTokens, req.Instructions = "gemini-other", 77, "Answer in French."
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, body := c.last()
	if body["model"] != "gemini-other" || generationConfig(body)["max_output_tokens"] != float64(77) {
		t.Errorf("model %v, max_output_tokens %v; want gemini-other, 77", body["model"], generationConfig(body)["max_output_tokens"])
	}
	if body["system_instruction"] != "Answer in French.\n\nBe brief." {
		t.Errorf("system_instruction = %q, want the instructions, then the system items", body["system_instruction"])
	}

	for _, tc := range []struct {
		choice llmprovider.ToolChoice
		want   string
	}{
		{llmprovider.ToolChoiceAuto, "<nil>"},
		{"", "<nil>"},
		{llmprovider.ToolChoiceRequired, "any"},
		{llmprovider.ToolChoiceNone, "none"},
		{llmprovider.ForceTool("get_weather"), "map[allowed_tools:map[mode:any tools:[get_weather]]]"},
	} {
		req := text("hi")
		req.Tools, req.ToolChoice = []llmprovider.Tool{weatherTool, {Name: "get_time", Schema: map[string]any{"type": "object"}}}, tc.choice
		if _, err := p.Generate(context.Background(), req); err != nil {
			t.Fatalf("%q: Generate: %v", tc.choice, err)
		}
		_, body := c.last()
		if got := fmt.Sprint(generationConfig(body)["tool_choice"]); got != tc.want {
			t.Errorf("%q: tool_choice = %s, want %s", tc.choice, got, tc.want)
		}
		if tools, _ := body["tools"].([]any); len(tools) != 2 {
			t.Errorf("%q: %d tools sent, want 2", tc.choice, len(tools))
		}
	}
}

// TestGenerate_Reasoning: a request's effort wins over WithReasoning's; a
// budget alone asks for summaries and sends no level, since the API has no
// budget; with neither, nothing is sent.
func TestGenerate_Reasoning(t *testing.T) {
	for _, tc := range []struct {
		name      string
		def, req  *llmprovider.Reasoning
		summaries bool
		level     string
	}{
		{"none", nil, nil, false, "<nil>"},
		{"default only", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, nil, true, "low"},
		{"request wins", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}, true, "high"},
		{"request budget, default effort", &llmprovider.Reasoning{Effort: llmprovider.EffortMedium}, &llmprovider.Reasoning{Budget: 4096}, true, "medium"},
		{"budget alone", nil, &llmprovider.Reasoning{Budget: 4096}, true, "<nil>"},
	} {
		srv, c := interactionServer(t, interactionText)
		var opts []llmprovider.Option
		if tc.def != nil {
			opts = append(opts, llmprovider.WithReasoning(tc.def))
		}
		p := build(t, apiKey(srv.URL, "gemini-3.7-flash", opts...)...)
		req := text("hi")
		req.Reasoning = tc.req
		if _, err := p.Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.name, err)
		}
		_, body := c.last()
		gen := generationConfig(body)
		if _, has := gen["thinking_summaries"]; has != tc.summaries || fmt.Sprint(gen["thinking_level"]) != tc.level {
			t.Errorf("%s: generation_config = %v, want summaries %v and level %s", tc.name, gen, tc.summaries, tc.level)
		}
		if _, ok := gen["thinking_budget"]; ok {
			t.Errorf("%s: thinking_budget sent, which the API refuses", tc.name)
		}
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request never reaches the
// service.
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	srv, c := interactionServer(t, interactionText)
	p := build(t, apiKey(srv.URL, "gemini-3.7-flash")...)
	req := text("hi")
	req.ToolChoice = "sometimes"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if n := c.count(); n != 0 {
		t.Errorf("%d requests, want none", n)
	}
}
