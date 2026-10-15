package catalog

import (
	"errors"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-PLAN Phase 5, 0041-MADR D5: ProfileAgent's band has ProfileCapable's
// members, ordered by signal, bench, preferred index, release quarter, the
// cheaper cached turn, then id.

// TestProfileAgent_ReasoningEffort_0041: as ProfileCapable's, "".
func TestProfileAgent_ReasoningEffort_0041(t *testing.T) {
	if got := ProfileAgent.ReasoningEffort(); got != "" {
		t.Errorf("ProfileAgent.ReasoningEffort() = %q; want \"\"", got)
	}
}

// TestProfileAgent_Members_0041: on the snapshot, every ranked provider's
// and every first-party section's agent band holds capable's band's members.
func TestProfileAgent_Members_0041(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for provider := range snapshotFile {
		capable := replaySnapshot(t, dir, provider, ProfileCapable, WithLimit(MaxListed)).Recommended
		agent := replaySnapshot(t, dir, provider, ProfileAgent, WithLimit(MaxListed)).Recommended
		if !sameSet(agent, capable) || len(agent) != MaxListed {
			t.Errorf("%s: agent band %v; want capable's members %v", provider, agent, capable)
		}
	}
	for _, provider := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderClaude,
		llmprovider.ProviderGemini, llmprovider.ProviderGrok} {
		capable, _ := sectionSix(t, provider, ProfileCapable)
		agent, _ := sectionSix(t, provider, ProfileAgent)
		if !sameSet(agent, capable) {
			t.Errorf("%s section: agent band %v; want capable's members %v", provider, agent, capable)
		}
	}
}

// priced is a candidate with a release age and prices, its cached turn at w.
func priced(id string, ageDays int, input, output, read float64, w workload) rankCandidate {
	c := rankCandidate{id: id, group: id, ageDays: ageDays, ageKnown: true, reasoning: true, reasoningKnown: true,
		cost: input + output, costKnown: true,
		prices: turnPrices{input: input, output: output, read: read, readKnown: read >= 0, known: true}}
	return c.withTurn(w)
}

// TestProfileAgent_Order_0041: within a quarter the cheaper cached turn
// comes first; a newer quarter beats a cheaper one; an unknown price sorts
// last.
func TestProfileAgent_Order_0041(t *testing.T) {
	w := defaultWorkload
	cands := []rankCandidate{
		priced("dear-new", 10, 3, 15, 0.3, w),
		priced("cheap-new", 40, 1, 5, 0.1, w),
		priced("cheapest-old", 100, 0.1, 0.4, 0.01, w),
		{id: "unpriced-new", group: "unpriced-new", ageDays: 5, ageKnown: true},
	}
	got := rankRecommended(ProfileAgent, llmprovider.ProviderTogether, cands, nil, MaxListed)
	if want := []string{"cheap-new", "dear-new", "unpriced-new", "cheapest-old"}; !slices.Equal(got, want) {
		t.Errorf("agent order %v; want %v", got, want)
	}
	if capable := rankRecommended(ProfileCapable, llmprovider.ProviderTogether, cands, nil, MaxListed); !sameSet(got, capable) {
		t.Errorf("agent members %v; want capable's %v", got, capable)
	}
}

// TestCachedTurn_0041 is §7's Sonnet 5 example, $2 input, $10 output, $0.20
// read, $2.50 write: 2.08 with no cached price, about 0.51 cached; a missing
// cache_read takes the canonical model's, and with none the input price.
func TestCachedTurn_0041(t *testing.T) {
	w := defaultWorkload
	read, write := 0.20, 2.50
	if got, ok := w.turn(turnPrices{input: 2, output: 10, known: true}); !ok || math.Abs(got-2.08) > 1e-9 {
		t.Errorf("uncached turn %v, %t; want 2.08", got, ok)
	}
	got, ok := w.turn(turnPrices{input: 2, output: 10, read: read, readKnown: true, write: write, writeKnown: true, known: true})
	if !ok || got < 0.47 || got > 0.53 {
		t.Errorf("cached turn %v, %t; want within [0.47, 0.53]", got, ok)
	}
	if _, ok := w.turn(turnPrices{}); ok {
		t.Error("an unknown price gave a turn")
	}
	doc, err := decodeModelMetadata(strings.NewReader(`{
"anthropic":{"models":{"claude-x":{"name":"X","cost":{"input":2,"output":10,"cache_read":0.2}}}},
"togetherai":{"models":{
 "a/claude-x":{"name":"X","canonical_model_id":"anthropic/claude-x","cost":{"input":2,"output":10}},
 "a/plain":{"name":"P","cost":{"input":2,"output":10}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	models, _ := doc.models("togetherai")
	if p := pricesOf(models["a/claude-x"], doc); !p.readKnown || p.read != 0.2 {
		t.Errorf("a/claude-x: read %v, known %t; want the canonical model's 0.2", p.read, p.readKnown)
	}
	plain := pricesOf(models["a/plain"], doc)
	if got, ok := w.turn(plain); plain.readKnown || !ok || math.Abs(got-2.08) > 1e-9 {
		t.Errorf("a/plain: read known %t, turn %v; want no read price, and the input price's 2.08", plain.readKnown, got)
	}
}

// TestWithWorkload_0041: a caller's workload reorders a band where it
// should, and a share outside (0, 1) is refused.
func TestWithWorkload_0041(t *testing.T) {
	order := func(w workload) []string {
		cands := []rankCandidate{
			priced("x/cached", 10, 2, 2, 0.1, w),
			priced("y/uncached", 20, 0.5, 0.5, -1, w),
		}
		return rankRecommended(ProfileAgent, llmprovider.ProviderTogether, cands, nil, MaxListed)
	}
	cfg, err := configFor(llmprovider.ProviderTogether, []llmprovider.Option{WithWorkload(0.5, 0.05)})
	if err != nil {
		t.Fatal(err)
	}
	if got := order(defaultWorkload); !slices.Equal(got, []string{"x/cached", "y/uncached"}) {
		t.Errorf("default workload: %v; want the cached model first", got)
	}
	if got := order(cfg.workload); !slices.Equal(got, []string{"y/uncached", "x/cached"}) {
		t.Errorf("WithWorkload(0.5, 0.05): %v; want the uncached model first", got)
	}
	for _, w := range [][2]float64{{0, 0.5}, {1, 0.5}, {math.NaN(), 0.5}, {0.5, 0}, {0.5, 1}, {0.5, math.Inf(1)}} {
		if _, err := configFor(llmprovider.ProviderTogether, []llmprovider.Option{WithWorkload(w[0], w[1])}); !errors.Is(err,
			llmprovider.ErrInvalidRequest) {
			t.Errorf("WithWorkload(%v, %v): %v; want ErrInvalidRequest", w[0], w[1], err)
		}
	}
}

// TestObservedWorkload_0041: the shares measured from usages; none, or a
// share WithWorkload would refuse, gives ok false.
func TestObservedWorkload_0041(t *testing.T) {
	h, o, ok := ObservedWorkload(
		llmprovider.Usage{InputTokens: 1000, OutputTokens: 10, CachedTokens: 900},
		llmprovider.Usage{InputTokens: 3000, OutputTokens: 30, CachedTokens: 2700},
		llmprovider.Usage{InputTokens: 6000, OutputTokens: 60, CachedTokens: 5400},
	)
	if !ok || math.Abs(h-0.9) > 1e-12 || math.Abs(o-100.0/10100) > 1e-12 {
		t.Errorf("ObservedWorkload = %v, %v, %t; want 0.9, %v, true", h, o, ok, 100.0/10100)
	}
	if _, _, ok := ObservedWorkload(); ok {
		t.Error("no usages: ok true")
	}
	if _, _, ok := ObservedWorkload(llmprovider.Usage{InputTokens: 100, OutputTokens: 5}); ok {
		t.Error("no cached tokens, a share of 0: ok true")
	}
}

// TestProfileAgent_Snapshot_0041 pins each ranked provider's agent band, as
// List gives it on the 2026-10-12 snapshot, at the published workload and
// at (0.5, 0.05), measured in 0041-PLAN Phase 5. Kilo's quality signals
// decide before the price, so its order is capable's at both.
func TestProfileAgent_Snapshot_0041(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for _, tc := range []struct {
		provider              llmprovider.ProviderID
		published, halfCached []string
	}{
		{llmprovider.ProviderOpencodeZen,
			[]string{"mistral-large-4", "gpt-6.1-sol", "claude-sonnet-5-5", "gpt-6-sol", "grok-4.7", "claude-opus-5-5"},
			[]string{"mistral-large-4", "grok-4.7", "gpt-6.1-sol", "claude-sonnet-5-5", "gpt-6-sol", "claude-opus-5-5"}},
		{llmprovider.ProviderOpencodeGo,
			[]string{"glm-5.3", "qwen3.8-max", "grok-4.6", "grok-4.7", "kimi-k3", "glm-5.2"},
			[]string{"glm-5.3", "grok-4.6", "grok-4.7", "qwen3.8-max", "kimi-k3", "glm-5.2"}},
		{llmprovider.ProviderKilo,
			[]string{"openai/gpt-6-astra", "anthropic/claude-fable-5.1", "openai/gpt-5.6-sol", "deepseek/deepseek-v4.1-flash",
				"google/gemini-3.8-flash", "x-ai/grok-4.6"},
			[]string{"openai/gpt-6-astra", "anthropic/claude-fable-5.1", "openai/gpt-5.6-sol", "deepseek/deepseek-v4.1-flash",
				"google/gemini-3.8-flash", "x-ai/grok-4.6"}},
		{llmprovider.ProviderHuggingFace,
			[]string{"Qwen/Qwen3.8-27B", "thinkingmachines/Inkling", "deepseek-ai/DeepSeek-V4-Pro-0813", "zai-org/GLM-5.3",
				"Qwen/Qwen3.8-2.4T-A95B", "moonshotai/Kimi-K3"},
			[]string{"Qwen/Qwen3.8-27B", "thinkingmachines/Inkling", "deepseek-ai/DeepSeek-V4-Pro-0813", "zai-org/GLM-5.3",
				"Qwen/Qwen3.8-2.4T-A95B", "moonshotai/Kimi-K3"}},
		{llmprovider.ProviderTogether,
			[]string{"deepseek-ai/DeepSeek-V4-Pro-0813", "thinkingmachines/Inkling", "zai-org/GLM-5.3", "moonshotai/Kimi-K3",
				"nvidia/nemotron-3-ultra-550b-a55b", "zai-org/GLM-5.2"},
			[]string{"thinkingmachines/Inkling", "deepseek-ai/DeepSeek-V4-Pro-0813", "zai-org/GLM-5.3", "moonshotai/Kimi-K3",
				"nvidia/nemotron-3-ultra-550b-a55b", "zai-org/GLM-5.2"}},
	} {
		if got := replaySnapshot(t, dir, tc.provider, ProfileAgent, WithLimit(MaxListed)).Recommended; !slices.Equal(got, tc.published) {
			t.Errorf("%s:\n got  %v\n want %v", tc.provider, got, tc.published)
		}
		got := replaySnapshot(t, dir, tc.provider, ProfileAgent, WithLimit(MaxListed), WithWorkload(0.5, 0.05)).Recommended
		if !slices.Equal(got, tc.halfCached) {
			t.Errorf("%s at (0.5, 0.05):\n got  %v\n want %v", tc.provider, got, tc.halfCached)
		}
	}
}

// TestKiloTurnPrices_0041: Kilo's per-token prices, its cache read and write
// among them, become a turn's prices per million tokens; a missing one is
// unknown.
func TestKiloTurnPrices_0041(t *testing.T) {
	var e kiloCatalogEntry
	e.ID = "v/m"
	e.Pricing.Prompt, e.Pricing.Completion = "0.000002", "0.00001"
	e.Pricing.InputCacheRead, e.Pricing.InputCacheWrite = "0.0000002", "0.0000025"
	p := kiloCandidate(e, rankingNow()).prices
	if !p.known || !p.readKnown || !p.writeKnown || math.Abs(p.input-2) > 1e-9 || math.Abs(p.output-10) > 1e-9 ||
		math.Abs(p.read-0.2) > 1e-9 || math.Abs(p.write-2.5) > 1e-9 {
		t.Errorf("prices %+v; want 2, 10, 0.2 and 2.5, each known", p)
	}
	e.Pricing.InputCacheRead, e.Pricing.InputCacheWrite = "", ""
	if p := kiloCandidate(e, rankingNow()).prices; p.readKnown || p.writeKnown {
		t.Errorf("prices %+v; want no cache price without one", p)
	}
}
