package openai

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// New in the new API: the Request fields the old methods could not express.

func TestNew_NeedsACredential(t *testing.T) {
	if _, err := New(llmprovider.WithModel("gpt-x")); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("New without a credential: err = %v, want ErrInvalidRequest", err)
	}
}

func TestNew_RefusesAForeignOption(t *testing.T) {
	_, err := New(llmprovider.WithAPIKey("k"), llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithOrganization", "o"))
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("a Kilo option: err = %v, want ErrInvalidRequest", err)
	}
}

func TestGenerate_RequestFields(t *testing.T) {
	weather := llmprovider.Tool{Name: "get_weather", Schema: map[string]any{"type": "object"}}
	clock := llmprovider.Tool{Name: "get_time", Schema: map[string]any{"type": "object"}}
	for _, c := range []struct {
		name  string
		req   func() *llmprovider.Request
		opts  []llmprovider.Option
		check func(t *testing.T, body map[string]any)
	}{
		{"model and instructions", func() *llmprovider.Request {
			r := text("hi")
			r.Model, r.Instructions = "gpt-override", "Be brief."
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if body["model"] != "gpt-override" || body["instructions"] != "Be brief." {
				t.Errorf("model = %v, instructions = %v", body["model"], body["instructions"])
			}
		}},
		{"no instructions field when empty", func() *llmprovider.Request { return text("hi") }, nil,
			func(t *testing.T, body map[string]any) {
				if _, ok := body["instructions"]; ok || body["model"] != "gpt-base" {
					t.Errorf("instructions present or model %v", body["model"])
				}
			}},
		{"MaxOutputTokens overrides WithMaxTokens", func() *llmprovider.Request {
			r := text("hi")
			r.MaxOutputTokens = 77
			return r
		}, []llmprovider.Option{llmprovider.WithMaxTokens(500)}, func(t *testing.T, body map[string]any) {
			if body["max_output_tokens"] != float64(77) {
				t.Errorf("max_output_tokens = %v, want 77", body["max_output_tokens"])
			}
		}},
		{"tools with auto choice send no tool_choice", func() *llmprovider.Request {
			r := text("hi")
			r.Tools = []llmprovider.Tool{weather, clock}
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			tools, _ := body["tools"].([]any)
			if _, ok := body["tool_choice"]; ok || len(tools) != 2 {
				t.Errorf("tool_choice = %v, %d tools", body["tool_choice"], len(tools))
			}
		}},
		{"required", func() *llmprovider.Request {
			r := text("hi")
			r.Tools, r.ToolChoice = []llmprovider.Tool{weather, clock}, llmprovider.ToolChoiceRequired
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if body["tool_choice"] != "required" {
				t.Errorf("tool_choice = %v, want required", body["tool_choice"])
			}
		}},
		{"none", func() *llmprovider.Request {
			r := text("hi")
			r.Tools, r.ToolChoice = []llmprovider.Tool{weather}, llmprovider.ToolChoiceNone
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if body["tool_choice"] != "none" {
				t.Errorf("tool_choice = %v, want none", body["tool_choice"])
			}
		}},
		{"a request's effort wins over the default", func() *llmprovider.Request {
			r := text("hi")
			r.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
			return r
		}, []llmprovider.Option{llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh})},
			func(t *testing.T, body map[string]any) {
				if r, _ := body["reasoning"].(map[string]any); r["effort"] != "low" {
					t.Errorf("reasoning = %v, want low", body["reasoning"])
				}
			}},
		{"the default reasoning applies to a request with none", func() *llmprovider.Request { return text("hi") },
			[]llmprovider.Option{llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh})},
			func(t *testing.T, body map[string]any) {
				if r, _ := body["reasoning"].(map[string]any); r["effort"] != "high" {
					t.Errorf("reasoning = %v, want high", body["reasoning"])
				}
			}},
		{"a budget alone sends the default effort", func() *llmprovider.Request {
			r := text("hi")
			r.Reasoning = &llmprovider.Reasoning{Budget: 4096}
			return r
		}, nil, func(t *testing.T, body map[string]any) {
			if r, _ := body["reasoning"].(map[string]any); r["effort"] != "medium" || len(r) != 1 {
				t.Errorf("reasoning = %v, want only effort medium", body["reasoning"])
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var body map[string]any
			srv := captureServer(t, &body, okResponse)
			p := build(t, apiKey(srv.URL, append([]llmprovider.Option{llmprovider.WithModel("gpt-base")}, c.opts...)...)...)
			if _, err := p.Generate(context.Background(), c.req()); err != nil {
				t.Fatal(err)
			}
			c.check(t, body)
		})
	}
}

// TestGenerate_RefusesBeforeTheNetwork: an invalid request is refused with no
// request sent (0015-MADR D4, D6).
func TestGenerate_RefusesBeforeTheNetwork(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, okResponse)
	p := build(t, apiKey(srv.URL)...)
	req := text("hi")
	req.ToolChoice = llmprovider.ForceTool("missing")
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
	if body != nil {
		t.Fatalf("a request was sent: %v", body)
	}
}
