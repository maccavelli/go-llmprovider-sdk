package together

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Ported from llmprovider's together_test.go (0015-PLAN S7), with
// assertThinkingFields from its thinking_wire_test.go.

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

// TestTogether_RequestShapes pins MADR 0017 D1: a request with no reasoning
// sends no reasoning field; one with reasoning sends reasoning {"enabled":
// true}, and reasoning_effort only when an effort was set; a forced tool pins
// tool_choice; the key is a bearer on POST /chat/completions with max_tokens.
// The effort was WithReasoningEffort's; it runs at construction and on the
// request now. "effort set, plain path" keeps its row: an effort is part of
// Reasoning now, so a request with no Reasoning carries none.
func TestTogether_RequestShapes(t *testing.T) {
	enabled := map[string]any{"enabled": true}
	for _, where := range []string{"construction", "request"} {
		for _, tc := range []struct {
			name          string
			effort        llmprovider.Effort
			thinking      bool
			tool          bool
			wantReasoning any
			wantEffort    any
		}{
			{name: "plain"},
			{name: "thinking", thinking: true, wantReasoning: enabled},
			{name: "thinking with effort", effort: llmprovider.EffortHigh, thinking: true, wantReasoning: enabled, wantEffort: "high"},
			{name: "effort set, plain path", effort: llmprovider.EffortHigh},
			{name: "forced tool", tool: true},
			{name: "forced tool, thinking", tool: true, thinking: true, wantReasoning: enabled},
		} {
			t.Run(where+"/"+tc.name, func(t *testing.T) {
				srv, c := togetherServer(t, togetherText, "[]")
				var opts []llmprovider.Option
				req := text("hi")
				if tc.thinking {
					if where == "construction" {
						opts = append(opts, llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: tc.effort}))
						req.Reasoning = &llmprovider.Reasoning{}
					} else {
						req.Reasoning = &llmprovider.Reasoning{Effort: tc.effort}
					}
				}
				if tc.tool {
					req.Tools, req.ToolChoice = []llmprovider.Tool{weatherTool}, llmprovider.ForceTool(weatherTool.Name)
				}
				p := build(t, append([]llmprovider.Option{llmprovider.WithAPIKey("tg-key"),
					llmprovider.WithModel("openai/gpt-oss-120b"), llmprovider.WithBaseURL(srv.URL)}, opts...)...)
				if _, err := p.Generate(context.Background(), req); err != nil {
					t.Fatalf("generate: %v", err)
				}
				path, header, body := c.last()
				if path != "/chat/completions" || header.Get("Authorization") != "Bearer tg-key" {
					t.Errorf("path %q, Authorization %q", path, header.Get("Authorization"))
				}
				if body["max_tokens"] != float64(8192) || body["model"] != "openai/gpt-oss-120b" {
					t.Errorf("max_tokens %v, model %v", body["max_tokens"], body["model"])
				}
				assertThinkingFields(t, body, map[string]any{"reasoning": tc.wantReasoning, "reasoning_effort": tc.wantEffort})
				choice, forced := body["tool_choice"]
				if forced != tc.tool {
					t.Errorf("tool_choice present = %v, want %v (%v)", forced, tc.tool, choice)
				}
			})
		}
	}
}

// TestTogether_EOSIsAStop: finish_reason "eos" keeps its text and is no error;
// only "length" is truncation.
func TestTogether_EOSIsAStop(t *testing.T) {
	srv, _ := togetherServer(t, `{"choices":[{"finish_reason":"eos","message":{"role":"assistant","content":"done"}}]}`, "[]")
	resp, err := build(t, apiKey(srv.URL, "m")...).Generate(context.Background(), text("hi"))
	if err != nil || resp.OutputText() != "done" || resp.FinishReason != "eos" {
		t.Fatalf("Generate = %+v, %v; want text \"done\", finish \"eos\", no error", resp, err)
	}
}

// TestTogether_Listing: GET /models is a bare array; only "chat" models are
// kept, the static order leads, and a failed listing falls back to the
// static catalog. No generation is sent.
func TestTogether_Listing(t *testing.T) {
	listing := `[{"id":"openai/gpt-oss-120b","type":"chat"},{"id":"BAAI/bge-large-en-v1.5","type":"embedding"},` +
		`{"id":"black-forest-labs/FLUX.2","type":"image"},{"id":"zai-org/GLM-5.3","type":"chat"},{"id":"new/Chat-Model","type":"chat"}]`
	srv, c := togetherServer(t, togetherText, listing)
	got, err := list(t, build(t, apiKey(srv.URL, "m")...))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"zai-org/GLM-5.3", "openai/gpt-oss-120b", "new/Chat-Model"}; !slices.Equal(got, want) {
		t.Errorf("ListModels = %v, want %v", got, want)
	}
	if path, _, _ := c.last(); path != "" {
		t.Errorf("listing sent a generation to %q", path)
	}

	for name, body := range map[string]string{"not an array": `{"data":[]}`, "no chat model": `[{"id":"x","type":"image"}]`} {
		t.Run(name, func(t *testing.T) {
			srv, _ := togetherServer(t, togetherText, body)
			got, err := list(t, build(t, apiKey(srv.URL, "m")...))
			if err != nil || !slices.Equal(got, catalog.Static(llmprovider.ProviderTogether)) {
				t.Errorf("ListModels = %v, %v; want the static catalog", got, err)
			}
		})
	}
}

// TestNew_NeedsAKey was TestTogether_RequiresKey: NewTogether("") was
// refused; New refuses no credential and an empty one, now with
// ErrInvalidRequest.
func TestNew_NeedsAKey(t *testing.T) {
	for name, opts := range map[string][]llmprovider.Option{
		"none":  {llmprovider.WithModel("m")},
		"empty": {llmprovider.WithAPIKey(""), llmprovider.WithModel("m")},
	} {
		if _, err := New(opts...); !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
	}
}

// TestTogether_NoContinuation: Chat Completions is stateless, so continuation
// is Unsupported, and refused before any request.
func TestTogether_NoContinuation(t *testing.T) {
	srv, c := togetherServer(t, togetherText, "[]")
	p := build(t, apiKey(srv.URL, "m")...)
	req := text("hi")
	req.PreviousResponseID = "resp_prev"
	if _, err := p.Generate(context.Background(), req); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Errorf("err = %v, want ErrUnsupported", err)
	}
	if path, _, _ := c.last(); path != "" {
		t.Errorf("a request was sent to %q", path)
	}
}

// TestTogether_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation and still matches the sentinel its status maps to (MADR 0012
// §1.1, 0013 B3).
func TestTogether_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := llmprovider.GenerateText(context.Background(), build(t, apiKey(srv.URL, "m")...), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") || !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want the service's message, matching ErrInvalidRequest", err)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestTogether_UserAgent: generation names this module honestly (MADR 0012
// §1.4).
func TestTogether_UserAgent(t *testing.T) {
	srv, c := togetherServer(t, togetherText, "[]")
	if _, err := build(t, apiKey(srv.URL, "m")...).Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, header, _ := c.last(); !userAgentPattern.MatchString(header.Get("User-Agent")) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", header.Get("User-Agent"))
	}
}

// TestTogether_Capabilities is 0017-MADR D1's: tools, forced tool choice and
// reasoning are per model.
func TestTogether_Capabilities(t *testing.T) {
	p := build(t, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"))
	caps := p.Capabilities()
	if caps.Tools != llmprovider.BestEffort || caps.ForcedToolChoice != llmprovider.BestEffort ||
		caps.Reasoning != llmprovider.BestEffort || caps.Continuation != llmprovider.Unsupported ||
		caps.NativeStreaming != llmprovider.Unsupported {
		t.Errorf("Capabilities = %+v", caps)
	}
	if p.ID() != llmprovider.ProviderTogether {
		t.Errorf("ID() = %q", p.ID())
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Error("the provider is not a ModelLister")
	}
}
