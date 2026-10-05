package catalog

import (
	"cmp"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The Grok CLI catalog's models (MADR 0012 §6).
const (
	grokModel46 = "grok-4.6"
	grokModel45 = "grok-4.5"
)

// MaxListed is the hard cap for configure-time model menus.
// Keeps wizards short and avoids dumping dozens of unusable API IDs.
const MaxListed = 6

// Gemini generateContent method name (Models API supportedGenerationMethods).
const methodGenerateContent = "generateContent"

// Static curated catalogs — preferred production text models for commit messages
// and general generate/chat use. Ordered: fast/cheap first, then quality.
// Updated from provider docs (Gemini 2026-07, OpenAI 2026, Anthropic 2026).
//
// These are the PRIMARY source of truth for menus. Live list APIs are used only
// to confirm availability and drop IDs the key cannot access — not to dump the
// full catalog (Gemini alone exposes embeddings, TTS, Live, image, robotics, …).
//
//nolint:goconst // catalog IDs are intentionally repeated in tests and filters
var (
	// staticGemini: stable fast text models only. gemini-2.0-* and 1.5-* are shut down.
	// Prioritizes low-latency Flash and Flash-Lite models for fast Git hook execution.
	staticGemini = []string{
		"gemini-3.7-flash",
		"gemini-3.6-flash",
		"gemini-3.5-flash",
		"gemini-3.5-flash-lite",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
	}

	// staticOpenAI: chat-capable defaults; mini/nano first for cost.
	staticOpenAI = []string{
		"gpt-4.1-mini",
		"gpt-4.1-nano",
		"gpt-4o-mini",
		"gpt-4.1",
		"gpt-4o",
		"o4-mini",
	}

	// staticClaude: current aliases, each answering on the Messages API
	// (verified 2026-09-27, MADR 0013 B10).
	staticClaude = []string{
		"claude-haiku-4-5",
		"claude-sonnet-5",
		"claude-sonnet-4-6",
		"claude-opus-4-8",
	}

	// staticOpencodeZen: MADR 0009 Context §7's utility six, ranked from the
	// 2026-09-26 Zen listing and models.opencode.ai metadata. Routes verified
	// against api.json's npm packages on 2026-09-26.
	staticOpencodeZen = []string{
		"deepseek-v4.1-flash",   // chat_completions
		"qwen3.8-flash",         // messages
		"glm-5.3-flash",         // chat_completions
		"deepseek-v4-flash",     // chat_completions
		"gemini-3.5-flash-lite", // google
		"gemini-3.8-flash",      // google
	}

	// staticOpencodeGo: MADR 0009 Context §7's utility six (2026-09-26). It excludes
	// the region-gated DeepSeek models and the -contributor models.
	staticOpencodeGo = []string{
		"mimo-v2.6-flash", // chat_completions
		"qwen3.8-flash",   // messages
		"glm-5.3-flash",   // chat_completions
		"gpt-6-luna",      // responses
		"mimo-v2.6-pro",   // chat_completions
		"hy3",             // chat_completions
	}

	// staticHuggingFace: fallback only — discovery is metadata-driven. MADR
	// 0009 Context §7's utility six, from the 2026-09-26 router listing and
	// models.opencode.ai metadata: reasoning-capable, paid, recent.
	staticHuggingFace = []string{
		"deepseek-ai/DeepSeek-V4-Flash-0731",
		"zai-org/GLM-5.3-Flash",
		"deepseek-ai/DeepSeek-V4.1-Flash",
		"thinkingmachines/Inkling-Small",
		"stepfun-ai/Step-3.7-Flash",
		"stepfun-ai/Step-3.5-Flash",
	}

	// staticTogether: fallback only — discovery is metadata-driven. From
	// models.dev's togetherai entry on 2026-09-30: tool-calling chat models,
	// not deprecated, text out, one per vendor, newest first
	// (docs/decisions/0017-MADR-together-provider-and-auth-extensions.md D1).
	staticTogether = []string{
		"deepseek-ai/DeepSeek-V4.1-Flash",
		"zai-org/GLM-5.3",
		"moonshotai/Kimi-K3",
		"MiniMaxAI/MiniMax-M3",
		"Qwen/Qwen3.6-Plus",
		"openai/gpt-oss-120b",
	}

	// staticKilo: fallback only — discovery is metadata-driven. MADR 0009 Context §7's
	// utility six, from the 2026-09-26 listing: reasoning-capable, paid,
	// recent, at most two per vendor, none training on prompts.
	staticKilo = []string{
		"deepseek/deepseek-v4.1-flash",
		"z-ai/glm-5.3-flash",
		"google/gemini-3.8-flash",
		"google/gemini-3.6-flash",
		"meta/muse-spark-1.2",
		"thinkingmachines/inkling",
	}

	// staticGrok leads with the Grok CLI's defaults, grok-4.6 then grok-4.5
	// (MADR 0012 §6), then the fast models.
	staticGrok = []string{
		grokModel46,
		grokModel45,
		"grok-3-mini-fast",
		"grok-3-mini",
		"grok-4",
		"grok-4-fast-reasoning",
	}
)

// geminiDenySubstrings reject non-text / non-production / specialized endpoints.
// Matched as lowercase substrings of the model id (after models/ prefix strip).
const denyComputerUse = "computer-use"

// Deny substrings shared across provider filters. They are named rather than
// repeated as literals so that adding a new provider's deny list does not push
// the existing occurrences over goconst's threshold, which would report files
// this change deliberately leaves untouched.
const (
	denyEmbed   = "embed"
	denyVision  = "vision"
	denyImage   = "image"
	denyAudio   = "audio"
	denyPreview = "preview"
	denyTTS     = "tts"
	denyOmni    = "omni"
)

var geminiDenySubstrings = []string{
	"image", "vision", "embed", "tts", "audio", "live", "preview",
	"exp", "experimental", "gemma", "learnlm", "aqa", "robotics",
	denyComputerUse, "deep-research", "antigravity", "veo", "lyria",
	"omni", "imagen", "native-audio", "thinking", "banana",
}

// openaiDenyPrefixes exclude non-chat model families from /v1/models.
var openaiDenyPrefixes = []string{
	"dall-e-", "whisper-", "tts-", "davinci-", "babbage-", "chatgpt-image",
	"text-embedding", "embedding", "moderation", "omni-moderation",
	"sora-", "gpt-image", "computer-use",
}

// openaiAllowPrefixes: chat / reasoning completions. gpt-6 joined with
// 0021-MADR amendment "the gpt-6 family on OpenAI's listing".
var openaiAllowPrefixes = []string{
	"gpt-4", "gpt-5", "gpt-6", "gpt-3.5-turbo", "o1", "o3", "o4",
	"chatgpt-4o",
}

// openaiDenyTokens are the token runs of OpenAI models that are not chat
// models: speech, transcription, search, deep research and realtime
// (0021-MADR C4).
var openaiDenyTokens = [][]string{{denyTTS}, {"transcribe"}, {"search"}, {"deep", "research"}, {"realtime"}}

// hasTokens reports whether id's tokens (modelTokens, the matcher's) hold
// seq as a contiguous run, so a rule for "mini" matches gpt-4o-mini but not
// gemini or minimax (0021-MADR C4, C5).
func hasTokens(id string, seq ...string) bool {
	tokens := modelTokens(id)
	for i := 0; i+len(seq) <= len(tokens); i++ {
		if slices.Equal(tokens[i:i+len(seq)], seq) {
			return true
		}
	}
	return false
}

// datedOrSnapshotGemini matches dated previews and numeric snapshots we should
// not surface when a stable short alias exists (e.g. gemini-2.5-flash-001).
var datedOrSnapshotGemini = regexp.MustCompile(`(?i)(-\d{2}-\d{4}|-\d{4}-\d{2}-\d{2}|-preview-|-\d{3}$|-exp)`)

// Rank scores a model id for sorting by preference within provider's
// catalog: higher is better. It replaces the per-provider Rank*Model functions
// (0015-PLAN S5 step 3). The provider id is compared case-insensitively, as
// Static compares it. Together ranks by position in its static catalog, the
// order its fallback curation keeps (0021-MADR C12). A provider with no
// ranking scores every model 0.
func Rank(provider llmprovider.ProviderID, model string) int {
	switch llmprovider.ProviderID(strings.ToLower(string(provider))) {
	case llmprovider.ProviderGemini:
		return rankGeminiModel(model)
	case llmprovider.ProviderOpenAI:
		return rankOpenAIModel(model)
	case llmprovider.ProviderClaude:
		return rankClaudeModel(model)
	case llmprovider.ProviderGrok:
		return rankGrokModel(model)
	case llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo:
		return rankOpencodeModel(model)
	case llmprovider.ProviderHuggingFace:
		return rankHuggingFaceModel(model)
	case llmprovider.ProviderKilo:
		return rankKiloModel(model)
	case llmprovider.ProviderTogether:
		if i := slices.Index(staticTogether, model); i >= 0 {
			return len(staticTogether) - i
		}
	}
	return 0
}

// Static returns a copy of the curated catalog for provider.
func Static(provider llmprovider.ProviderID) []string {
	switch llmprovider.ProviderID(strings.ToLower(string(provider))) {
	case llmprovider.ProviderGemini:
		return append([]string(nil), staticGemini...)
	case llmprovider.ProviderOpenAI:
		return append([]string(nil), staticOpenAI...)
	case llmprovider.ProviderClaude:
		return append([]string(nil), staticClaude...)
	case llmprovider.ProviderGrok:
		return append([]string(nil), staticGrok...)
	case llmprovider.ProviderOpencodeZen:
		return append([]string(nil), staticOpencodeZen...)
	case llmprovider.ProviderOpencodeGo:
		return append([]string(nil), staticOpencodeGo...)
	case llmprovider.ProviderHuggingFace:
		return append([]string(nil), staticHuggingFace...)
	case llmprovider.ProviderKilo:
		return append([]string(nil), staticKilo...)
	case llmprovider.ProviderTogether:
		return append([]string(nil), staticTogether...)
	case llmprovider.ProviderOllama:
		// Installed models are machine-specific, so there is no meaningful
		// static catalog. listOllamaModels is the only sensible source, and
		// ConfigureLLM tolerates an empty catalog for exactly this case.
		return nil
	default:
		return nil
	}
}

// staticOpencodeCatalog returns the curation seed for a gateway (no copy;
// callers must not mutate).
func staticOpencodeCatalog(gateway llmprovider.ProviderID) []string {
	if gateway == llmprovider.ProviderOpencodeGo {
		return staticOpencodeGo
	}
	return staticOpencodeZen
}

// opencodeDenySubstrings reject non-text / non-production entries that appear in
// the gateway catalogs (e.g. deepseek-v4-flash-vision-exp, mimo-v2-omni,
// hy3-preview on OpenCode Go).
var opencodeDenySubstrings = []string{
	denyVision, denyImage, denyEmbed, denyTTS, denyAudio, denyOmni, denyPreview, "-exp",
}

// isUsableOpencodeModel filters gateway model IDs to production text models.
// Unlike the vendor filters it enforces no family prefix: the gateways
// deliberately aggregate many vendors under bare IDs.
func isUsableOpencodeModel(id string) bool {
	sm := strings.ToLower(strings.TrimSpace(id))
	// jev-* models use OpenCode's "systemone" route, which this package has
	// no encoder for (MADR 0012 §3.1).
	if sm == "" || strings.HasPrefix(sm, "jev-") {
		return false
	}
	for _, deny := range opencodeDenySubstrings {
		if strings.Contains(sm, deny) {
			return false
		}
	}
	return true
}

// rankOpencodeModel scores a gateway model for menu ordering. Higher is better.
// Prefers low-latency tiers for fast Git hook execution, penalizes heavy
// reasoning tiers, and demotes free models because the gateway rate-limits them
// aggressively (HTTP 429 FreeUsageLimitError).
func rankOpencodeModel(m string) int {
	sm := strings.ToLower(m)
	score := 0
	// The name rules match whole tokens (0021-MADR C5).
	switch {
	case hasTokens(sm, "nano"):
		score += 200
	case hasTokens(sm, "lite"):
		score += 190
	case hasTokens(sm, "flash"):
		score += 180
	case hasTokens(sm, "mini"):
		score += 170
	case hasTokens(sm, "haiku"):
		score += 160
	case hasTokens(sm, "sonnet"):
		score += 90
	case hasTokens(sm, "opus"), hasTokens(sm, "pro"), hasTokens(sm, "max"):
		score -= 300
	}
	if strings.HasSuffix(sm, "-free") {
		score -= 50 // free tier is rate-limited; usable but not a default
	}
	if strings.Contains(sm, "codex") {
		score -= 100 // code-completion specialisations, not general chat
	}
	return score
}

// isUsableGeminiTextModel reports whether id is a production text generateContent model.
func isUsableGeminiTextModel(id string, methods []string) bool {
	id = strings.TrimPrefix(id, "models/")
	sm := strings.ToLower(strings.TrimSpace(id))
	if sm == "" || !strings.HasPrefix(sm, "gemini") {
		return false
	}
	canGenerate := slices.Contains(methods, methodGenerateContent)
	if !canGenerate {
		return false
	}
	for _, deny := range geminiDenySubstrings {
		if strings.Contains(sm, deny) {
			return false
		}
	}
	// Reject dated/snapshot/experimental naming for menu stability.
	if datedOrSnapshotGemini.MatchString(sm) {
		return false
	}
	// Require flash, pro, or lite family (skip odd one-offs).
	if !strings.Contains(sm, "flash") && !strings.Contains(sm, "pro") && !strings.Contains(sm, "lite") {
		return false
	}
	return true
}

// isUsableOpenAIChatModel filters OpenAI /v1/models IDs to chat/reasoning models.
func isUsableOpenAIChatModel(id string) bool {
	sm := strings.ToLower(strings.TrimSpace(id))
	if sm == "" {
		return false
	}
	for _, p := range openaiDenyPrefixes {
		if strings.HasPrefix(sm, p) || strings.Contains(sm, p) {
			return false
		}
	}
	// Prefer undated aliases for menu; still allow dated gpt-4.1-2025-* if needed later.
	// Drop instruction-only and realtime.
	if strings.Contains(sm, "instruct") || strings.Contains(sm, "realtime") || strings.Contains(sm, "audio") {
		return false
	}
	for _, seq := range openaiDenyTokens {
		if hasTokens(sm, seq...) {
			return false
		}
	}
	for _, p := range openaiAllowPrefixes {
		if strings.HasPrefix(sm, p) {
			return true
		}
	}
	return false
}

// isUsableClaudeTextModel filters Anthropic model IDs for Messages API text use.
func isUsableClaudeTextModel(id string) bool {
	sm := strings.ToLower(strings.TrimSpace(id))
	if sm == "" || !strings.HasPrefix(sm, "claude") {
		return false
	}
	// Skip non-Messages specialties if they appear.
	for _, deny := range []string{denyComputerUse, "bedrock", "instant-1"} {
		if strings.Contains(sm, deny) {
			return false
		}
	}
	return true
}

// curateFromCatalog returns catalog models that appear in available (case-sensitive
// match first, then case-insensitive). Preserves catalog order. If catalog hits do not
// fill MaxListed, dynamically backfills with top-ranked usable models from available.
// If nothing from the catalog is available, falls back to filtering `available` with rankFn.
func curateFromCatalog(catalog, available []string, usable func(string) bool, rankFn func(string) int) []string {
	avail := make(map[string]string, len(available)) // lower -> canonical
	for _, a := range available {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if usable != nil && !usable(a) {
			continue
		}
		avail[strings.ToLower(a)] = a
	}

	out := make([]string, 0, MaxListed)
	seen := make(map[string]struct{}, MaxListed)

	for _, c := range catalog {
		key := strings.ToLower(c)
		if canon, ok := avail[key]; ok {
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, canon)
			if len(out) >= MaxListed {
				return out
			}
		}
	}

	// Backfill with remaining usable models from available if catalog hits < MaxListed.
	if len(available) > 0 {
		var remaining []string
		for _, a := range available {
			key := strings.ToLower(strings.TrimSpace(a))
			if usable != nil && !usable(a) {
				continue
			}
			if _, dup := seen[key]; !dup {
				remaining = append(remaining, a)
			}
		}
		if rankFn != nil {
			sortByRankDesc(remaining, rankFn)
		}
		for _, r := range remaining {
			if len(out) >= MaxListed {
				break
			}
			key := strings.ToLower(r)
			if _, dup := seen[key]; !dup {
				seen[key] = struct{}{}
				out = append(out, r)
			}
		}
	}

	return out
}

// sortByRankDesc sorts ids by rankFn descending (stable enough via simple insertion).
func sortByRankDesc(ids []string, rankFn func(string) int) {
	// Stable: equal ranks keep the listing's order (0020-MADR F36).
	slices.SortStableFunc(ids, func(a, b string) int { return cmp.Compare(rankFn(b), rankFn(a)) })
}

// rankGeminiModel scores a Gemini model name for sorting by preference.
// Higher is better. Prioritizes low-latency Flash and Flash-Lite models for fast Git hook execution,
// while penalizing heavy reasoning (Pro) models and deprecated generations.
func rankGeminiModel(m string) int {
	score := 0
	sm := strings.ToLower(m)

	if strings.Contains(sm, "flash-lite") || strings.HasSuffix(sm, "lite") {
		score += 200
	} else if strings.Contains(sm, "flash") {
		score += 180
	} else if strings.Contains(sm, "pro") {
		score -= 500 // Penalize heavy reasoning models for Git hook latency
	}

	if strings.Contains(sm, "preview") || strings.Contains(sm, "exp") || strings.Contains(sm, "deep-research") {
		score -= 1000
	}
	if datedOrSnapshotGemini.MatchString(sm) {
		score -= 200
	}

	// Version weights (newer generations first). A hint matches whole
	// tokens; a generation no hint names is placed among the anchors
	// (0021-MADR C6).
	switch {
	case hasRun(sm, "3.7"):
		score += 100
	case hasRun(sm, "3.6"):
		score += 90
	case hasRun(sm, "3.5"):
		score += 80
	case hasRun(sm, "3.1") || hasRun(sm, "3.0"):
		score += 40
	case hasRun(sm, "2.5"):
		score += 30
	case slices.ContainsFunc(geminiShutDown, func(v string) bool { return hasRun(sm, v) }):
		score -= 2000 // Deprecated / shut down
	default:
		score += generationScore(sm, geminiAnchors, 0)
	}

	return score
}

// geminiShutDown are the shut-down Gemini generations.
var geminiShutDown = []string{"2.0", "1.5"}

// geminiAnchors place a Gemini generation that no hint names. The 3.0, 3.1
// and gemini-3- case is one anchor, at 3.0, so that the cap never puts a
// gemini-3- id below the 40 the case gives it.
var geminiAnchors = []generationAnchor{{205, 30}, {300, 40}, {305, 80}, {306, 90}, {307, 100}}

// rankOpenAIModel prefers mini/nano for cost, then flagship chat.
func rankOpenAIModel(m string) int {
	sm := strings.ToLower(m)
	score := 0
	switch {
	case strings.Contains(sm, "nano"):
		score += 200
	case strings.Contains(sm, "mini"):
		score += 180
	case strings.HasPrefix(sm, "o4"):
		score += 120
	case strings.HasPrefix(sm, "o3"):
		score += 100
	case strings.Contains(sm, "4.1"):
		score += 160
	case strings.Contains(sm, "4o"):
		score += 140
	case strings.Contains(sm, "gpt-5"), strings.Contains(sm, "gpt-6"):
		score += 150
	}
	if strings.Contains(sm, "realtime") || strings.Contains(sm, "audio") {
		score -= 500
	}
	return score
}

// rankClaudeModel prefers Haiku/Sonnet for speed, then newer gens.
func rankClaudeModel(m string) int {
	sm := strings.ToLower(m)
	score := 0
	if strings.Contains(sm, "haiku") {
		score += 200
	} else if strings.Contains(sm, "sonnet") {
		score += 150
	} else if strings.Contains(sm, "opus") {
		score += 100
	} else if strings.Contains(sm, "fable") {
		score += 90
	}
	// Generation hints, matched as whole tokens; a generation no hint names
	// is placed among the anchors (0021-MADR C6).
	matched := false
	if hasRun(sm, "4-5") || hasRun(sm, "sonnet-5") || hasRun(sm, "haiku-4") {
		score += 40
		matched = true
	}
	if hasRun(sm, "opus-4-8") || hasRun(sm, "sonnet-4-6") {
		score += 35
		matched = true
	}
	if hasRun(sm, "3-5") {
		score += 10
		matched = true
	}
	if !matched {
		score += generationScore(sm, claudeAnchors, 0)
	}
	return score
}

// claudeAnchors place a Claude generation that no hint names. They do not
// rise with the generation, which is why generationScore caps them.
var claudeAnchors = []generationAnchor{{305, 10}, {400, 40}, {405, 40}, {406, 35}, {408, 35}, {500, 40}}

// isUsableGrokModel filters xAI model IDs for Responses API text use.
func isUsableGrokModel(id string) bool {
	sm := strings.ToLower(strings.TrimSpace(id))
	if sm == "" || !strings.HasPrefix(sm, "grok") {
		return false
	}
	// Skip non-text specialties: xAI lists grok-imagine-video, which "image"
	// does not match (MADR 0012 §6).
	for _, deny := range []string{"vision", denyImage, "embed", "imagine", "video", "voice", "stt", "tts"} {
		if strings.Contains(sm, deny) {
			return false
		}
	}
	return true
}

// rankGrokModel prefers mini-fast for cost/speed, then by generation.
func rankGrokModel(m string) int {
	sm := strings.ToLower(m)
	score := 0
	switch {
	case strings.Contains(sm, "mini-fast"):
		score += 200
	case strings.Contains(sm, "mini"):
		score += 180
	case strings.Contains(sm, "fast-reasoning"):
		score += 100
	}
	// Generation weights, matched as whole tokens. grok-4 and grok-3 are
	// anchors, not hints, so a newer grok-4.x ranks above them (0021-MADR
	// C6).
	switch {
	case hasRun(sm, "4.6"):
		score += 60
	case hasRun(sm, "4.5"):
		score += 50
	default:
		score += generationScore(sm, grokAnchors, 0)
	}
	return score
}

// grokAnchors place a Grok generation that no hint names.
var grokAnchors = []generationAnchor{{300, 30}, {400, 40}, {405, 50}, {406, 60}}

// generationAnchor is a generation hint's score at its order key, ord =
// 100·major + minor (0021-MADR C6).
type generationAnchor struct{ ord, score int }

// hasRun reports whether id holds hint's tokens as a contiguous run, so
// "3.1" matches gemini-3.1-flash but not gemini-3.10-flash.
func hasRun(id, hint string) bool { return hasTokens(id, modelTokens(hint)...) }

// parseGeneration reads the first version after an id's family name: N,
// N.M or N-M, where N is one digit and M one or two, so a date or a size is
// never a version.
func parseGeneration(id string) (major, minor int, ok bool) {
	tokens := modelTokens(id)
	for i := 1; i < len(tokens); i++ {
		n, isVersion := versionNumber(tokens[i])
		if !isVersion || len(tokens[i]) > 1 {
			continue
		}
		if i+1 < len(tokens) {
			if m, isMinor := versionNumber(tokens[i+1]); isMinor {
				return n, m, true
			}
		}
		return n, 0, true
	}
	return 0, 0, false
}

// versionNumber reads a token of one or two digits.
func versionNumber(token string) (int, bool) {
	if token == "" || len(token) > 2 || strings.Trim(token, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(token)
	return n, err == nil
}

// generationScore places id's generation among anchors, sorted by ord
// (0021-MADR C6; amendment "C6's anchors are capped"):
//   - between anchors, it takes the highest anchor at or below it, capped at
//     one less than the lowest anchor above it;
//   - above every anchor, it takes the newest anchor's score plus 10 plus
//     its distance in steps, at most 40: 20 a major, 1 a minor;
//   - below every anchor, or with no version, it is below.
func generationScore(id string, anchors []generationAnchor, below int) int {
	major, minor, ok := parseGeneration(id)
	if !ok {
		return below
	}
	ord := 100*major + minor
	newest := anchors[len(anchors)-1]
	switch {
	case ord < anchors[0].ord:
		return below
	case ord > newest.ord:
		steps := minor - newest.ord%100
		if major != newest.ord/100 {
			steps = 20*(major-newest.ord/100) + minor
		}
		return newest.score + 10 + min(40, steps)
	}
	score, ceiling := 0, math.MaxInt
	for _, a := range anchors {
		if a.ord <= ord {
			score = a.score
		} else {
			ceiling = min(ceiling, a.score-1)
		}
	}
	return min(score, ceiling)
}

// splitHuggingFaceModelPolicy splits a router model id into its bare
// "<org>/<name>" and an optional provider-selection policy suffix
// (":fastest", ":cheapest", ":preferred", or a partner name like ":groq").
// The split is at the LAST colon, and only when the suffix contains no "/",
// because org and model names may not contain colons but the policy always
// follows one.
//
// NOTE: this is Hugging Face-specific. Kilo model ids may legitimately END in
// ":free" (tencent/hy3:free), so Kilo must NOT use this helper.
func splitHuggingFaceModelPolicy(id string) (base, policy string) {
	i := strings.LastIndex(id, ":")
	if i < 0 || strings.Contains(id[i+1:], "/") {
		return id, ""
	}
	return id[:i], id[i+1:]
}

// isUsableHuggingFaceModel filters router model IDs to production text models.
// Modality filtering happens in listHuggingFaceModels from published metadata;
// this catches the static-catalog path where no metadata is available.
func isUsableHuggingFaceModel(id string) bool {
	base, _ := splitHuggingFaceModelPolicy(strings.TrimSpace(id))
	sm := strings.ToLower(base)
	if sm == "" || !strings.Contains(sm, "/") {
		return false
	}
	for _, deny := range []string{denyVision, "-vl-", denyEmbed, denyTTS, "whisper", "flux", "stable-diffusion"} {
		if strings.Contains(sm, deny) {
			return false
		}
	}
	return true
}

// rankHuggingFaceModel is a weak, name-based fallback used only when the live
// listing is unavailable and the static catalog is in play. The primary ranking
// path uses published throughput and latency; see listHuggingFaceModels.
func rankHuggingFaceModel(m string) int {
	sm := strings.ToLower(m)
	score := 0
	switch {
	case strings.Contains(sm, "-20b"), strings.Contains(sm, "8b"), strings.Contains(sm, "flash"):
		score += 150
	case strings.Contains(sm, "-120b"), strings.Contains(sm, "70b"):
		score += 80
	}
	if strings.Contains(sm, "thinking") || strings.Contains(sm, "-r1") {
		score -= 100 // reasoning-heavy; slow for hook latency
	}
	return score
}

// isUsableKiloModel filters Kilo model IDs to production text models.
//
// POLICY, not capability: models flagged mayTrainOnYourPrompts are excluded by
// listKiloModels (25 of 366 on 2026-08-29). A shared library used by commit
// hooks should not route a private diff to a model that trains on it without
// the caller asking. This is the only judgement this package makes on the
// user's behalf; TestIsUsableKiloModel_TrainingPolicy pins it so it stays
// visible and reversible. The flag lives in the listing, so the check is in
// listKiloModels; this function handles the id-only static path.
func isUsableKiloModel(id string) bool {
	sm := strings.ToLower(strings.TrimSpace(id))
	// NOTE: no colon-splitting — ":free" is part of the id, not a policy suffix.
	if sm == "" || !strings.Contains(sm, "/") {
		return false
	}
	for _, deny := range []string{denyVision, "-vl", denyEmbed, denyTTS, "whisper", denyOmni, denyImage} {
		if strings.Contains(sm, deny) {
			return false
		}
	}
	return true
}

// rankKiloModel is a weak, name-based fallback used only when the live listing
// is unavailable. The primary ranking path uses published pricing; see
// listKiloModels.
func rankKiloModel(m string) int {
	sm := strings.ToLower(m)
	score := 0
	if strings.HasPrefix(sm, "kilo-auto/") {
		score += 200 // managed tiers: stable ids, gateway-selected models
	}
	// The name rules match whole tokens (0021-MADR C5).
	switch {
	case hasTokens(sm, "flash"), hasTokens(sm, "lightning"),
		hasTokens(sm, "small"), hasTokens(sm, "mini"):
		score += 100
	case hasTokens(sm, "pro"), hasTokens(sm, "max"), hasTokens(sm, "frontier"):
		score -= 200
	}
	return score
}

// modelLabels maps a curated model id to a human-readable menu label. It lives
// beside the static catalogs it annotates so a catalog edit and its label edit
// are the same diff — the arrangement that let prepare-commit-msg's private
// 17-entry copy drift out of step with this package.
//
// Labels are advisory. Label falls back to the bare id, so a live listing
// that outruns this table degrades to raw ids rather than hiding models.
//
//nolint:goconst // model ids are intentionally repeated across catalog and labels
var modelLabels = map[string]string{
	// Gemini
	"gemini-3.7-flash":      "Gemini 3.7 Flash       [★ Recommended: frontier coding intelligence]",
	"gemini-3.6-flash":      "Gemini 3.6 Flash       [high efficiency, reduced token overhead]",
	"gemini-3.5-flash":      "Gemini 3.5 Flash       [high-speed production workhorse]",
	"gemini-3.5-flash-lite": "Gemini 3.5 Flash-Lite  [ultra-low latency]",
	"gemini-2.5-flash":      "Gemini 2.5 Flash       [proven balanced baseline]",
	"gemini-2.5-flash-lite": "Gemini 2.5 Flash-Lite  [lightweight baseline]",

	// OpenAI
	"gpt-4.1-mini": "GPT-4.1 Mini           [★ Recommended: fast & cost-effective]",
	"gpt-4.1-nano": "GPT-4.1 Nano           [ultra-low latency]",
	"gpt-4o-mini":  "GPT-4o Mini            [stable fast chat model]",
	"gpt-4.1":      "GPT-4.1                [high-capability fast tier]",
	"gpt-4o":       "GPT-4o                 [flagship multimodal]",
	"o4-mini":      "o4-mini                [fast reasoning]",

	// Claude
	"claude-haiku-4-5":  "Claude Haiku 4.5       [★ Recommended: high speed, low latency]",
	"claude-sonnet-5":   "Claude Sonnet 5        [balanced speed & precision]",
	"claude-sonnet-4-6": "Claude Sonnet 4.6      [stable high precision]",
	"claude-opus-4-8":   "Claude Opus 4.8        [maximum capability]",

	// Grok
	"grok-3-mini-fast":      "Grok 3 Mini Fast       [★ Recommended: fastest tier]",
	"grok-3-mini":           "Grok 3 Mini            [low latency]",
	"grok-4":                "Grok 4                 [flagship]",
	"grok-4.6":              "Grok 4.6               [current flagship]",
	"grok-4.5":              "Grok 4.5               [previous flagship]",
	"grok-4-fast-reasoning": "Grok 4 Fast Reasoning  [fast reasoning tier]",

	// Together
	"openai/gpt-oss-120b": "GPT-OSS 120B           [highest throughput]",
}

// Label returns a human-readable menu label for a model, or the bare model
// id when none is known. The provider argument is accepted for future
// per-provider disambiguation; ids are currently unique across providers.
func Label(provider llmprovider.ProviderID, model string) string {
	_ = provider
	if label, ok := modelLabels[model]; ok {
		return label
	}
	return model
}
