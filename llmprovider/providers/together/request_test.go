package together

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// Request fields the old methods could not express, and New's refusals
// (0015-PLAN S7).

// TestNew_RefusesAnOAuthSession: R16 refuses a source Together does not
// accept (0017-MADR D1: an API key only).
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
	_, err := New(llmprovider.WithAPIKey("k"), llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithOrganization", "o"))
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
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(togetherText))
	}))
	t.Cleanup(srv.Close)
	p := build(t, llmprovider.WithTokenSource(&countingSource{}), llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
	}
	if fmt.Sprint(seen) != "[Bearer key-1 Bearer key-2]" {
		t.Fatalf("Authorization sent %v, want [Bearer key-1 Bearer key-2]", seen)
	}
}

// TestGenerate_RequestFields: the model, output limit and instructions a
// request names, and each tool choice with every tool offered.
func TestGenerate_RequestFields(t *testing.T) {
	srv, c := togetherServer(t, togetherText, "[]")
	p := build(t, apiKey(srv.URL, "base/model", llmprovider.WithMaxTokens(500))...)
	req := &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: "Be brief."}, user("hi")}}
	req.Model, req.MaxOutputTokens, req.Instructions = "other/model", 77, "Answer in French."
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	_, _, body := c.last()
	if body["model"] != "other/model" || body["max_tokens"] != float64(77) {
		t.Errorf("model %v, max_tokens %v; want other/model, 77", body["model"], body["max_tokens"])
	}
	if msgs, _ := body["messages"].([]any); len(msgs) != 3 || fmt.Sprint(msgs[0]) != "map[content:Answer in French. role:system]" {
		t.Errorf("messages = %v, want the instructions as a leading system message", body["messages"])
	}

	for _, tc := range []struct {
		choice llmprovider.ToolChoice
		want   string
	}{
		{llmprovider.ToolChoiceAuto, "<nil>"},
		{llmprovider.ToolChoiceRequired, "required"},
		{llmprovider.ToolChoiceNone, "none"},
		{llmprovider.ForceTool("get_weather"), "map[function:map[name:get_weather] type:function]"},
	} {
		req := text("hi")
		req.Tools = []llmprovider.Tool{weatherTool, {Name: "get_time", Schema: map[string]any{"type": "object"}}}
		req.ToolChoice = tc.choice
		if _, err := p.Generate(context.Background(), req); err != nil {
			t.Fatalf("%q: Generate: %v", tc.choice, err)
		}
		_, _, body := c.last()
		if got := fmt.Sprint(body["tool_choice"]); got != tc.want {
			t.Errorf("%q: tool_choice = %s, want %s", tc.choice, got, tc.want)
		}
		if tools, _ := body["tools"].([]any); len(tools) != 2 {
			t.Errorf("%q: %d tools sent, want 2", tc.choice, len(tools))
		}
	}
}

// TestGenerate_Reasoning: a request's effort wins over WithReasoning's; an
// empty one takes WithReasoning's; a budget alone sends no effort, the budget
// not sent; WithReasoning alone reasons on every request.
func TestGenerate_Reasoning(t *testing.T) {
	for _, tc := range []struct {
		name     string
		def, req *llmprovider.Reasoning
		want     any
	}{
		{"request wins", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}, "high"},
		{"empty takes the default", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, &llmprovider.Reasoning{}, "low"},
		{"budget alone", nil, &llmprovider.Reasoning{Budget: 4096}, nil},
		{"default only", &llmprovider.Reasoning{Effort: llmprovider.EffortLow}, nil, "low"},
		{"default with no effort", &llmprovider.Reasoning{}, nil, nil},
	} {
		srv, c := togetherServer(t, togetherText, "[]")
		var opts []llmprovider.Option
		if tc.def != nil {
			opts = append(opts, llmprovider.WithReasoning(tc.def))
		}
		req := text("hi")
		req.Reasoning = tc.req
		if _, err := build(t, apiKey(srv.URL, "m", opts...)...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.name, err)
		}
		_, _, body := c.last()
		if fmt.Sprint(body["reasoning"]) != "map[enabled:true]" {
			t.Errorf("%s: reasoning = %v, want {enabled: true}", tc.name, body["reasoning"])
		}
		if got, sent := body["reasoning_effort"]; (tc.want == nil && sent) || (tc.want != nil && got != tc.want) {
			t.Errorf("%s: reasoning_effort = %v (sent %t), want %v", tc.name, got, sent, tc.want)
		}
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request never reaches
// Together.
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	srv, c := togetherServer(t, togetherText, "[]")
	req := text("hi")
	req.ToolChoice = "sometimes"
	if _, err := build(t, apiKey(srv.URL, "m")...).Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if path, _, _ := c.last(); path != "" {
		t.Errorf("a request was sent to %q", path)
	}
}
