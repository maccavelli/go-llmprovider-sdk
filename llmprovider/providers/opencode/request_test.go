package opencode

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Request fields the old methods could not express, and the constructors'
// refusals (0015-PLAN S7).

// TestNew_RefusesAnOAuthSession: R16 refuses a source the gateways do not
// accept (0016-MADR D10).
func TestNew_RefusesAnOAuthSession(t *testing.T) {
	for _, gateway := range family {
		for name, src := range map[string]llmprovider.TokenSource{
			"oauth":  &llmprovider.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a"},
			"vendor": &llmprovider.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "unused"},
		} {
			if _, err := newFor(gateway)(llmprovider.WithTokenSource(src)); !errors.Is(err, llmprovider.ErrUnsupported) {
				t.Errorf("%s/%s: err = %v, want ErrUnsupported", gateway, name, err)
			}
		}
	}
}

// TestNew_WithRouteIsForOpencodeOnly: the route option reaches both gateways,
// and another provider's option is refused.
func TestNew_WithRouteIsForOpencodeOnly(t *testing.T) {
	for _, gateway := range family {
		if _, err := newFor(gateway)(WithRoute(RouteMessages)); err != nil {
			t.Errorf("%s: WithRoute: %v", gateway, err)
		}
		_, err := newFor(gateway)(llmprovider.ScopedOption(llmprovider.ProviderGrok, "grok.WithStore", true))
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: a grok option: err = %v, want ErrInvalidRequest", gateway, err)
		}
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
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("x-api-key"))
		_, _ = w.Write([]byte(fxMessages))
	}))
	t.Cleanup(srv.Close)
	p := build(t, llmprovider.ProviderOpencodeZen, llmprovider.WithTokenSource(&countingSource{}),
		llmprovider.WithModel("claude-sonnet-5"), llmprovider.WithBaseURL(srv.URL))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
	}
	if fmt.Sprint(keys) != "[key-1 key-2]" {
		t.Fatalf("x-api-key sent %v, want [key-1 key-2]", keys)
	}
}

// TestGenerate_ToolChoicesPerRoute: each route's form of each choice, with
// every tool offered.
func TestGenerate_ToolChoicesPerRoute(t *testing.T) {
	type form func(map[string]any) string
	topLevel := func(b map[string]any) string { return fmt.Sprint(b["tool_choice"]) }
	google := func(b map[string]any) string { return fmt.Sprint(b["toolConfig"]) }
	named := llmprovider.ForceTool("get_weather")
	for _, tc := range []struct {
		model, fixture string
		choice         form
		want           map[llmprovider.ToolChoice]string
	}{
		{"gpt-5.5", fxResponses, topLevel, map[llmprovider.ToolChoice]string{
			llmprovider.ToolChoiceAuto: "<nil>", llmprovider.ToolChoiceRequired: "required", llmprovider.ToolChoiceNone: "none",
			named: "map[name:get_weather type:function]"}},
		{"claude-sonnet-5", fxMessages, topLevel, map[llmprovider.ToolChoice]string{
			llmprovider.ToolChoiceAuto: "<nil>", llmprovider.ToolChoiceRequired: "map[type:any]", llmprovider.ToolChoiceNone: "map[type:none]",
			named: "map[name:get_weather type:tool]"}},
		{"gemini-3.7-flash", fxGoogle, google, map[llmprovider.ToolChoice]string{
			llmprovider.ToolChoiceAuto: "<nil>", llmprovider.ToolChoiceRequired: "map[functionCallingConfig:map[mode:ANY]]",
			llmprovider.ToolChoiceNone: "map[functionCallingConfig:map[mode:NONE]]",
			named:                      "map[functionCallingConfig:map[allowedFunctionNames:[get_weather] mode:ANY]]"}},
		{deepSeekV4Pro, fxChat, topLevel, map[llmprovider.ToolChoice]string{
			llmprovider.ToolChoiceAuto: "<nil>", llmprovider.ToolChoiceRequired: "required", llmprovider.ToolChoiceNone: "none",
			named: "map[function:map[name:get_weather] type:function]"}},
	} {
		for choice, want := range tc.want {
			var body map[string]any
			srv := captureServer(t, &body, tc.fixture)
			req := text("hi")
			req.Tools = []llmprovider.Tool{{Name: "get_weather", Schema: map[string]any{"type": "object"}},
				{Name: "get_time", Schema: map[string]any{"type": "object"}}}
			req.ToolChoice = choice
			if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model)...).Generate(context.Background(), req); err != nil {
				t.Fatalf("%s %q: Generate: %v", tc.model, choice, err)
			}
			if got := tc.choice(body); got != want {
				t.Errorf("%s %q: tool choice = %s, want %s", tc.model, choice, got, want)
			}
			if got := fmt.Sprint(body["tools"]); !strings.Contains(got, "get_weather") || !strings.Contains(got, "get_time") {
				t.Errorf("%s %q: tools = %s, want both", tc.model, choice, got)
			}
		}
	}
}

// TestGenerate_RequestFields: the output limit a request names reaches each
// route's field.
func TestGenerate_RequestFields(t *testing.T) {
	for _, tc := range []struct {
		model, fixture string
		limit          func(map[string]any) any
	}{
		{"gpt-5.5", fxResponses, func(b map[string]any) any { return b["max_output_tokens"] }},
		{"claude-sonnet-5", fxMessages, func(b map[string]any) any { return b["max_tokens"] }},
		{"gemini-3.7-flash", fxGoogle, func(b map[string]any) any { return b["generationConfig"].(map[string]any)["maxOutputTokens"] }},
		{deepSeekV4Pro, fxChat, func(b map[string]any) any { return b["max_tokens"] }},
	} {
		var body map[string]any
		srv := captureServer(t, &body, tc.fixture)
		req := text("hi")
		req.MaxOutputTokens = 77
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, tc.model, llmprovider.WithMaxTokens(500))...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: Generate: %v", tc.model, err)
		}
		if got := tc.limit(body); got != float64(77) {
			t.Errorf("%s: output limit = %v, want 77", tc.model, got)
		}
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request never reaches the
// gateway.
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	var path string
	srv := pathCapture(t, &path, fxChat)
	req := text("hi")
	req.ToolChoice = "sometimes"
	if _, err := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "glm-5.3")...).Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want ErrInvalidRequest", err)
	}
	if path != "" {
		t.Errorf("a request was sent to %s", path)
	}
}
