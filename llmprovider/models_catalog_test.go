package llmprovider

import (
	"slices"
	"strings"
	"testing"
)

func TestIsUsableGeminiTextModel(t *testing.T) {
	methods := []string{methodGenerateContent}
	cases := []struct {
		id   string
		want bool
	}{
		{"gemini-3.7-flash", true},
		{"gemini-3.6-flash", true},
		{"gemini-3.5-flash", true},
		{"gemini-3.5-flash-lite", true},
		{"gemini-2.5-flash", true},
		{"gemini-2.5-flash-lite", true},
		{"gemini-2.5-pro", true},
		{"models/gemini-3.7-flash", true},
		// Specialty / non-production
		{"gemini-2.5-flash-image", false},
		{"gemini-2.5-flash-preview-tts", false},
		{"gemini-2.5-flash-native-audio-preview-12-2025", false},
		{"gemini-embedding-001", false},
		{"gemini-2.5-computer-use-preview-10-2025", false},
		{"gemini-2.5-flash-preview-09-2025", false},
		{"gemini-2.5-flash-001", false},
		{"gemma-3-27b", false},
		{"imagen-4", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isUsableGeminiTextModel(tc.id, methods)
		if got != tc.want {
			t.Errorf("isUsableGeminiTextModel(%q)=%v want %v", tc.id, got, tc.want)
		}
	}
	if isUsableGeminiTextModel("gemini-3.7-flash", []string{"embedContent"}) {
		t.Error("must require generateContent")
	}
}

func TestIsUsableOpenAIChatModel(t *testing.T) {
	if !isUsableOpenAIChatModel("gpt-4.1-mini") {
		t.Error("gpt-4.1-mini should be allowed")
	}
	if !isUsableOpenAIChatModel("gpt-4o") {
		t.Error("gpt-4o should be allowed")
	}
	if isUsableOpenAIChatModel("dall-e-3") {
		t.Error("dall-e should be denied")
	}
	if isUsableOpenAIChatModel("text-embedding-3-small") {
		t.Error("embeddings denied")
	}
	if isUsableOpenAIChatModel("whisper-1") {
		t.Error("whisper denied")
	}
}

func TestCurateFromCatalog_Gemini(t *testing.T) {
	// API returns noise + good models
	available := []string{
		"gemini-2.5-flash-image",
		"gemini-3.7-flash",
		"gemini-3.6-flash",
		"gemini-2.5-flash-preview-09-2025",
		"gemini-3.5-flash",
		"gemini-embedding-001",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.0-flash", // shut down-ish; still text if not denied by filter
	}
	// Filter first as listGemini does
	var filtered []string
	for _, a := range available {
		if isUsableGeminiTextModel(a, []string{methodGenerateContent}) {
			filtered = append(filtered, a)
		}
	}
	out := curateFromCatalog(staticGemini, filtered, func(s string) bool {
		return isUsableGeminiTextModel(s, []string{methodGenerateContent})
	}, rankGeminiModel)

	if len(out) == 0 {
		t.Fatal("expected curated models")
	}
	if len(out) > MaxListedModels {
		t.Fatalf("cap exceeded: %d", len(out))
	}
	for _, m := range out {
		if strings.Contains(m, "image") || strings.Contains(m, "preview") || strings.Contains(m, "embed") {
			t.Errorf("junk leaked into menu: %s", m)
		}
	}
	// Catalog order preference: 3.7-flash before 2.5-flash when both present
	if idx37, idx25 := indexOf(out, "gemini-3.7-flash"), indexOf(out, "gemini-2.5-flash"); idx37 >= 0 && idx25 >= 0 && idx37 > idx25 {
		t.Errorf("expected 3.7-flash before 2.5-flash in catalog order: %v", out)
	}
}

func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}

func TestCurateFromCatalog_EmptyAvailableUsesNothingThenFallback(t *testing.T) {
	out := curateFromCatalog(staticGemini, nil, nil, rankGeminiModel)
	if len(out) != 0 {
		// no available, no fallback path without available items
		t.Fatalf("expected empty, got %v", out)
	}
	// With only unusable available, usable filter empties map → empty
	out = curateFromCatalog(staticOpenAI, []string{"dall-e-3"}, isUsableOpenAIChatModel, rankOpenAIModel)
	if len(out) != 0 {
		t.Fatalf("expected empty when only junk available: %v", out)
	}
}

func TestStaticModels_NoShutDownGemini20(t *testing.T) {
	if len(staticGemini) > MaxListedModels {
		t.Errorf("staticGemini should not exceed MaxListedModels (%d), got %d", MaxListedModels, len(staticGemini))
	}
	for _, m := range staticGemini {
		if strings.Contains(m, "2.0") {
			t.Errorf("static catalog should not recommend shut-down 2.0 models: %s", m)
		}
		if strings.Contains(m, "1.5") {
			t.Errorf("static catalog should not recommend shut-down 1.5 models: %s", m)
		}
		if strings.Contains(m, "preview") {
			t.Errorf("static catalog must not include preview: %s", m)
		}
		if strings.Contains(m, "pro") {
			t.Errorf("static catalog must not include slow reasoning pro models: %s", m)
		}
	}
}

func TestRankGeminiModel_PrefersFlashLite(t *testing.T) {
	if rankGeminiModel("gemini-3.7-flash") <= rankGeminiModel("gemini-2.5-pro") {
		t.Error("3.7 flash should rank above pro for hook latency")
	}
	if rankGeminiModel("gemini-3.7-flash") <= rankGeminiModel("gemini-2.5-flash") {
		t.Error("newer 3.7 generation should rank above 2.5")
	}
	if rankGeminiModel("gemini-2.5-flash-preview-09-2025") >= rankGeminiModel("gemini-2.5-flash") {
		t.Error("preview should rank far below stable flash")
	}
}

func TestIsUsableGrokModel(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"grok-4", true},
		{"grok-3-mini", true},
		{"grok-3-mini-fast", true},
		{"grok-4.5", true},
		{"grok-4.6", true},
		{"grok-4-fast-reasoning", true},
		{"grok-vision-beta", false},
		{"grok-image-gen", false},
		{"", false},
		{"llama-3", false},
	}
	for _, tc := range tests {
		if got := isUsableGrokModel(tc.id); got != tc.want {
			t.Errorf("isUsableGrokModel(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestRankGrokModel(t *testing.T) {
	if rankGrokModel("grok-3-mini-fast") <= rankGrokModel("grok-3-mini") {
		t.Error("mini-fast should rank above mini")
	}
	if rankGrokModel("grok-3-mini") <= rankGrokModel("grok-4") {
		t.Error("mini should rank above grok-4 for cost preference")
	}
}

func TestStaticGrok_Count(t *testing.T) {
	if len(staticGrok) > MaxListedModels {
		t.Errorf("staticGrok has %d entries, max is %d", len(staticGrok), MaxListedModels)
	}
}

func TestStaticModels(t *testing.T) {
	providers := []string{
		ProviderGemini, ProviderOpenAI, ProviderClaude, ProviderGrok,
		ProviderOpencodeZen, ProviderOpencodeGo, ProviderHuggingFace, ProviderKilo, ProviderTogether,
	}
	for _, p := range providers {
		models := StaticModels(p)
		if len(models) == 0 {
			t.Errorf("StaticModels(%q) returned empty slice", p)
		}
	}
	if StaticModels("unknown") != nil {
		t.Errorf("StaticModels('unknown') should return nil")
	}
}

func TestIsUsableClaudeTextModel(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"claude-sonnet-5", true},
		{"claude-haiku-4-5", true},
		{"claude-opus-4-8", true},
		{"claude-3-5-sonnet", true},
		{"claude-computer-use-preview", false},
		{"claude-3-bedrock", false},
		{"claude-instant-1", false},
		{"gpt-4", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isUsableClaudeTextModel(tc.id); got != tc.want {
			t.Errorf("isUsableClaudeTextModel(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestRankOpenAIModel(t *testing.T) {
	if rankOpenAIModel("gpt-4.1-nano") <= rankOpenAIModel("gpt-4.1-mini") {
		t.Error("nano should rank above mini for cost preference")
	}
	if rankOpenAIModel("gpt-4.1-mini") <= rankOpenAIModel("gpt-4.1") {
		t.Error("mini should rank above standard 4.1")
	}
	if rankOpenAIModel("o4") <= rankOpenAIModel("o3") {
		t.Error("o4 should rank above o3")
	}
	if rankOpenAIModel("gpt-4o-realtime") >= 0 {
		t.Error("realtime models should be penalized")
	}
	if rankOpenAIModel("unknown-model") != 0 {
		t.Error("unknown model should have score 0")
	}
}

func TestRankClaudeModel(t *testing.T) {
	if rankClaudeModel("claude-haiku-4-5") <= rankClaudeModel("claude-sonnet-5") {
		t.Error("haiku should rank above sonnet for speed")
	}
	if rankClaudeModel("claude-sonnet-5") <= rankClaudeModel("claude-opus-4-8") {
		t.Error("sonnet should rank above opus")
	}
	if rankClaudeModel("claude-fable-4") <= 0 {
		t.Error("fable should have positive score")
	}
	if rankClaudeModel("unknown-model") != 0 {
		t.Error("unknown model should have score 0")
	}
}

func TestSortByRankDesc(t *testing.T) {
	models := []string{"gemini-2.5-flash", "gemini-3.7-flash", "gemini-2.5-pro"}
	sortByRankDesc(models, rankGeminiModel)
	if models[0] != "gemini-3.7-flash" {
		t.Errorf("expected highest rank first, got %v", models)
	}
}

func TestCurateFromCatalog_Backfill(t *testing.T) {
	// Catalog has only 1 matching model, available has more usable models
	catalog := []string{"gpt-4.1-mini"}
	available := []string{"gpt-4.1-mini", "gpt-4.1-nano", "gpt-4o", "gpt-5"}
	out := curateFromCatalog(catalog, available, isUsableOpenAIChatModel, rankOpenAIModel)
	if len(out) != 4 {
		t.Fatalf("expected 4 backfilled models, got %d (%v)", len(out), out)
	}
	if out[0] != "gpt-4.1-mini" {
		t.Errorf("catalog hit should be first, got %s", out[0])
	}
}

func TestIsUsableOpencodeModel(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"gpt-5.4-nano", true},
		{"claude-haiku-4-5", true},
		{"kimi-k2.6", true},
		{"hy3-free", true},
		{"deepseek-v4-flash-vision-exp", false},
		{"mimo-v2-omni", false},
		{"hy3-preview", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isUsableOpencodeModel(tc.id); got != tc.want {
			t.Errorf("isUsableOpencodeModel(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestRankOpencodeModel(t *testing.T) {
	if rankOpencodeModel("gpt-5.4-nano") <= rankOpencodeModel("gpt-5.4-mini") {
		t.Error("nano should rank above mini for hook latency")
	}
	if rankOpencodeModel("gemini-3.5-flash-lite") <= rankOpencodeModel("gemini-3.7-flash") {
		t.Error("lite should rank above flash")
	}
	if rankOpencodeModel("claude-haiku-4-5") <= rankOpencodeModel("claude-sonnet-5") {
		t.Error("haiku should rank above sonnet")
	}
	if rankOpencodeModel("qwen3.8-flash") <= rankOpencodeModel("qwen3.8-max") {
		t.Error("flash should rank above max")
	}
	if rankOpencodeModel("hy3-free") >= rankOpencodeModel("hy3") {
		t.Error("free tier is rate-limited and must rank below its paid sibling")
	}
}

func TestStaticOpencode_Count(t *testing.T) {
	for name, cat := range map[string][]string{
		"staticOpencodeZen": staticOpencodeZen,
		"staticOpencodeGo":  staticOpencodeGo,
	} {
		if len(cat) == 0 || len(cat) > MaxListedModels {
			t.Errorf("%s has %d entries, want 1..%d", name, len(cat), MaxListedModels)
		}
		for _, m := range cat {
			// The docs endpoint table lists these but the live listing does not.
			if m == "qwen3.7-max" || m == "qwen3.7-plus" {
				t.Errorf("%s must not include %q: absent from the live listing", name, m)
			}
			if !isUsableOpencodeModel(m) {
				t.Errorf("%s entry %q fails its own usability filter", name, m)
			}
		}
	}
}

// TestStaticModels_OpenAIIsPlatformCatalog: the only OpenAI static catalog is
// the Platform one; ChatGPT sessions have none (MADR 0008 D11).
func TestStaticModels_OpenAIIsPlatformCatalog(t *testing.T) {
	if got := StaticModels(ProviderOpenAI); !slices.Equal(got, staticOpenAI) {
		t.Fatalf("StaticModels(openai) = %v, want staticOpenAI %v", got, staticOpenAI)
	}
}
