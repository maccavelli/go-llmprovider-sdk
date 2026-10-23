package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0041-MADR D2: the output limit (R7) and the Chat Completions replay field,
// from the model's facts, across the built-in providers.

// limitLayer is a caller layer that gives one model an output limit.
type limitLayer struct {
	model string
	limit int
}

func (l limitLayer) Name() string { return "test" }

func (l limitLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	if id != l.model {
		return catalog.ModelInfo{}, false
	}
	return catalog.ModelInfo{ID: id, MaxOutputTokens: l.limit}.Known(catalog.FieldMaxOutputTokens), true
}

// sentLimit is the output limit a body asks for, whatever its wire's name for
// it, and wherever the wire puts it.
func sentLimit(v any) (float64, bool) {
	switch v := v.(type) {
	case map[string]any:
		for _, key := range []string{"max_tokens", "max_output_tokens", "max_completion_tokens", "maxOutputTokens"} {
			// wiretest keeps numbers exact, as json.Number.
			if n, ok := v[key].(json.Number); ok {
				f, err := n.Float64()
				return f, err == nil
			}
		}
		for _, inner := range v {
			if n, ok := sentLimit(inner); ok {
				return n, true
			}
		}
	case []any:
		for _, inner := range v {
			if n, ok := sentLimit(inner); ok {
				return n, true
			}
		}
	}
	return 0, false
}

// TestOutputLimit_Facts_0041 (R7): with no MaxOutputTokens, a model whose
// limit is 4096 is asked for 4096, and one whose limit is 128,000 for the
// 8192 default, which is never raised.
func TestOutputLimit_Facts_0041(t *testing.T) {
	for _, tc := range []struct {
		id    llmprovider.ProviderID
		model string
	}{
		{llmprovider.ProviderClaude, "claude-sonnet-5"},
		{llmprovider.ProviderOpenAI, "gpt-6-luna"},
		{llmprovider.ProviderGrok, "grok-4.6"},
		{llmprovider.ProviderGemini, "gemini-3.8-flash"},
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5"},
		{llmprovider.ProviderOpencodeZen, "gpt-5.5"},
		{llmprovider.ProviderOpencodeZen, "glm-5.3"},
		{llmprovider.ProviderOpencodeZen, "gemini-3.8-flash"},
		{llmprovider.ProviderKilo, "openai/gpt-6-luna"},
		{llmprovider.ProviderHuggingFace, "deepseek-ai/DeepSeek-V4.1-Flash"},
		{llmprovider.ProviderTogether, "openai/gpt-oss-120b"},
		{llmprovider.ProviderOllama, "llama3.3:latest"},
	} {
		for _, limits := range [][2]int{{4096, 4096}, {128000, 8192}} {
			srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
			p, err := New(tc.id, llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model), llmprovider.WithBaseURL(srv.URL),
				llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false),
				catalog.WithModelLayer(limitLayer{tc.model, limits[0]}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.Generate(context.Background(), &llmprovider.Request{
				Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}); err != nil {
				t.Fatalf("%s/%s: %v", tc.id, tc.model, err)
			}
			if got, ok := sentLimit(lastBody(srv)); !ok || int(got) != limits[1] {
				t.Errorf("%s/%s, a limit of %d: sent %v (found %t); want %d", tc.id, tc.model, limits[0], got, ok, limits[1])
			}
		}
	}
}

// TestReplayField_Facts_0041: Together and Hugging Face replay prior
// reasoning under the field the model's interleaved.field names, read from
// the cached document; with no such fact, under none.
func TestReplayField_Facts_0041(t *testing.T) {
	for _, tc := range []struct {
		id      llmprovider.ProviderID
		section string
		model   string
	}{
		{llmprovider.ProviderTogether, "togetherai", "org/think"},
		{llmprovider.ProviderHuggingFace, "huggingface", "org/think"},
	} {
		for _, field := range []string{"reasoning_content", ""} {
			interleaved := ""
			if field != "" {
				interleaved = `,"interleaved":{"field":"` + field + `"}`
			}
			doc := `{"` + tc.section + `":{"models":{"` + tc.model + `":{"name":"Think","reasoning":true,` +
				`"limit":{"context":100000,"output":8192}` + interleaved + `}}}}`
			meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, doc)
			}))
			t.Cleanup(meta.Close)
			srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
			opts := []llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelMetadataURL(meta.URL), llmprovider.WithModelProbes(false),
				testCache(t)}
			// The document cached, as a request finds it once fetched.
			if _, err := catalog.LookupMetadataWith(context.Background(), tc.id, opts...); err != nil {
				t.Fatal(err)
			}
			p, err := New(tc.id, opts...)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.Generate(context.Background(), &llmprovider.Request{Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "a"},
				llmprovider.ReasoningItem{Text: "a thought"},
				llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "b"},
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "c"},
			}}); err != nil {
				t.Fatalf("%s: %v", tc.id, err)
			}
			var assistant map[string]any
			for _, m := range lastBody(srv)["messages"].([]any) {
				if msg := m.(map[string]any); msg["role"] == "assistant" {
					assistant = msg
				}
			}
			replayed := ""
			for key, v := range assistant {
				if s, ok := v.(string); ok && strings.Contains(s, "a thought") {
					replayed = key
				}
			}
			if replayed != field {
				t.Errorf("%s, interleaved.field %q: the reasoning was replayed under %q; want %q (assistant %v)", tc.id,
					field, replayed, field, assistant)
			}
		}
	}
}
