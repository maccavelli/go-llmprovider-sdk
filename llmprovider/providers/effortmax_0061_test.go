package providers

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0061-MADR D3: EffortMax on each built-in provider and route, by the
// model's facts, with its warning on the Response.

// effortLayer is a caller layer that lists one model's efforts.
type effortLayer struct {
	model   string
	efforts []llmprovider.Effort
}

func (l effortLayer) Name() string { return "test" }

func (l effortLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	if id != l.model {
		return catalog.ModelInfo{}, false
	}
	return catalog.ModelInfo{ID: id, Reasoning: catalog.ModelReasoning{Efforts: l.efforts}}.Known(
		catalog.FieldReasoningEfforts), true
}

// sentEffort is the effort a body asks for, whatever its wire's name for it,
// and wherever the wire puts it.
func sentEffort(v any) string {
	switch v := v.(type) {
	case map[string]any:
		for _, key := range []string{"effort", "reasoning_effort", "thinking_level", "thinkingLevel"} {
			if s, ok := v[key].(string); ok {
				return s
			}
		}
		for _, inner := range v {
			if s := sentEffort(inner); s != "" {
				return s
			}
		}
	case []any:
		for _, inner := range v {
			if s := sentEffort(inner); s != "" {
				return s
			}
		}
	}
	return ""
}

// TestEffortMax_Mapping_0061: a request at EffortMax, with facts that list
// max (rule 1), facts that list low, medium and high (rule 2, Q9 (a)), and
// no facts (rule 3). Each want is the effort sent, then the effort the
// warning names, "" for none. By 0062-MADR D3's ladder every provider sends
// what the facts allow, and max with none: a service that refuses it is
// learned from, not guessed at (0062-MADR D8, 0061-MADR D3's rows removed).
// OpenCode's google route sends the level, max with no facts (0062-MADR,
// amendment "Phase 3's google-route levels, and 0013's B9").
func TestEffortMax_Mapping_0061(t *testing.T) {
	type want struct{ sent, warned string }
	withMax := []llmprovider.Effort{"low", "medium", "high", "xhigh", "max"}
	toHigh := []llmprovider.Effort{"low", "medium", "high"}
	for _, tc := range []struct {
		id                    llmprovider.ProviderID
		model                 string
		rule1, rule2, noFacts want
	}{
		{llmprovider.ProviderClaude, "claude-sonnet-5", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderOpenAI, "gpt-6-luna", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderGrok, "grok-4.6", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderGemini, "gemini-3.8-flash", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderOpencodeZen, "gpt-5.5", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderOpencodeZen, "gemini-3.8-flash", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderKilo, "openai/gpt-6-luna", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderHuggingFace, "deepseek-ai/DeepSeek-V4.1-Flash", want{"max", ""}, want{"high", "high"},
			want{"max", ""}},
		{llmprovider.ProviderTogether, "openai/gpt-oss-120b", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
		{llmprovider.ProviderOllama, "llama3.3:latest", want{"max", ""}, want{"high", "high"}, want{"max", ""}},
	} {
		for _, run := range []struct {
			name    string
			efforts []llmprovider.Effort
			want    want
		}{{"rule 1", withMax, tc.rule1}, {"rule 2", toHigh, tc.rule2}, {"no facts", nil, tc.noFacts}} {
			srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
			opts := []llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false)}
			if run.efforts != nil {
				opts = append(opts, catalog.WithModelLayer(effortLayer{tc.model, run.efforts}))
			}
			p, err := New(tc.id, opts...)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := p.Generate(context.Background(), &llmprovider.Request{
				Input:     []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
				Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortMax}})
			if err != nil {
				t.Fatalf("%s/%s, %s: %v", tc.id, tc.model, run.name, err)
			}
			if got := sentEffort(lastBody(srv)); got != run.want.sent {
				t.Errorf("%s/%s, %s: sent %q; want %q", tc.id, tc.model, run.name, got, run.want.sent)
			}
			var warned []llmprovider.Warning
			for _, w := range resp.Warnings {
				if w.Kind == llmprovider.WarningEffortDegraded {
					warned = append(warned, w)
				}
			}
			switch {
			case run.want.warned == "" && len(warned) != 0:
				t.Errorf("%s/%s, %s: warnings %v; want none", tc.id, tc.model, run.name, warned)
			case run.want.warned != "" && (len(warned) != 1 || warned[0].Model != tc.model ||
				warned[0].Detail != "reasoning effort max is not offered for "+tc.model+"; sent "+run.want.warned):
				t.Errorf("%s/%s, %s: warnings %v; want one naming %s", tc.id, tc.model, run.name, warned, run.want.warned)
			}
		}
	}
}
