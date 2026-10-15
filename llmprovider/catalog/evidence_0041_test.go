package catalog

import (
	"bytes"
	"cmp"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/generatecontent"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

// 0041-PLAN Phase 0: each number 0041-MADR states from the committed
// snapshot, asserted through the code's own functions, offline, with the
// ranking clock at the snapshot's recorded_at. A later phase that changes a
// function under test updates its assertion in the same commit. R6 is
// asserted in providers/grok, which imports this package (Phase 0,
// deviation 2).

// evidence0041Dir is the snapshot 0041-MADR measured, or the one
// LLMPROVIDER_SNAPSHOT_DIR names, as tools/catalogcheck.py sets it for the
// agreement tests (0041-MADR D7; 0041-PLAN Phase 7). The evidence tests pin
// the measured snapshot's figures, and fail on any other.
var evidence0041Dir = snapshotDirOr(filepath.Join("testdata", "ranking-2026-10-12"))

// snapshotDirOr is LLMPROVIDER_SNAPSHOT_DIR when it is set, else dir.
func snapshotDirOr(dir string) string {
	if v := os.Getenv("LLMPROVIDER_SNAPSHOT_DIR"); v != "" {
		return v
	}
	return dir
}

// evidence0041Doc decodes the snapshot's models.dev document and pins the
// ranking clock at its recorded_at.
func evidence0041Doc(t *testing.T) *modelMetadataDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(evidence0041Dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := decodeModelMetadata(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(evidence0041Dir, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		RecordedAt string `json:"recorded_at"`
	}
	if err := json.Unmarshal(manifest, &m); err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse(time.RFC3339, m.RecordedAt)
	if err != nil {
		t.Fatal(err)
	}
	pinRankingNow(t, at)
	return doc
}

// evidence0041Section is one models.dev section of doc, which must exist.
func evidence0041Section(t *testing.T, doc *modelMetadataDoc, key string) map[string]modelMetadata {
	t.Helper()
	models, ok := doc.models(key)
	if !ok {
		t.Fatalf("the snapshot has no %q section", key)
	}
	return models
}

// reasons reports whether m is a reasoning model.
func reasons(m modelMetadata) bool { return m.Reasoning != nil && *m.Reasoning }

// optionsOf is m's reasoning_options of kind.
func optionsOf(m modelMetadata, kind string) []modelReasoningOption {
	var out []modelReasoningOption
	for _, o := range m.ReasoningOptions {
		if o.Type == kind {
			out = append(out, o)
		}
	}
	return out
}

// TestEvidence0041_ClaudeAdaptive is §3's R1: the Messages wire sends
// adaptive thinking exactly where models.dev lists no budget_tokens option,
// on every Claude reasoning model of the three sections.
func TestEvidence0041_ClaudeAdaptive(t *testing.T) {
	doc := evidence0041Doc(t)
	want := map[string]int{"anthropic": 17, "opencode": 16, "opencode-go": 1}
	total := 0
	for sec, n := range want {
		agree, seen := 0, 0
		for id, m := range evidence0041Section(t, doc, sec) {
			bare := id[strings.LastIndex(id, "/")+1:]
			if !strings.Contains(strings.ToLower(bare), "claude") || !reasons(m) {
				continue
			}
			seen++
			body := map[string]any{}
			messages.AddThinking(body, bare, "high", 0, 64000)
			thinking, _ := body["thinking"].(map[string]any)
			if adaptive := thinking["type"] == "adaptive"; adaptive == (len(optionsOf(m, "budget_tokens")) == 0) {
				agree++
			} else {
				t.Errorf("%s/%s: adaptive %t, but models.dev's budget_tokens options are %v", sec, id, adaptive,
					optionsOf(m, "budget_tokens"))
			}
		}
		if seen != n || agree != n {
			t.Errorf("%s: %d of %d Claude reasoning models agree; want %d of %d", sec, agree, seen, n, n)
		}
		total += agree
	}
	if total != 34 {
		t.Errorf("R1: %d agree; want 34 of 34", total)
	}
}

// TestEvidence0041_GeminiLegacy is §3's R4: at low effort the generateContent
// wire sends a thinking budget exactly where models.dev lists a budget and no
// effort, on 27 of Google's 29 reasoning models; the other two are an image
// model and a computer-use model.
func TestEvidence0041_GeminiLegacy(t *testing.T) {
	doc := evidence0041Doc(t)
	agree, seen := 0, 0
	var differ []string
	for id, m := range evidence0041Section(t, doc, "google") {
		if !reasons(m) {
			continue
		}
		seen++
		_, budget := generatecontent.ThinkingConfig(id, "low", 0)["thinkingBudget"]
		var efforts []string
		for _, o := range optionsOf(m, "effort") {
			efforts = append(efforts, o.Values...)
		}
		if budget == (len(optionsOf(m, "budget_tokens")) > 0 && len(efforts) == 0) {
			agree++
		} else {
			differ = append(differ, id)
		}
	}
	slices.Sort(differ)
	if want := []string{"gemini-2.5-computer-use-preview-10-2025", "gemini-2.5-flash-image"}; agree != 27 || seen != 29 ||
		!slices.Equal(differ, want) {
		t.Errorf("R4: %d of %d agree, differing %v; want 27 of 29, differing %v", agree, seen, differ, want)
	}
}

// TestEvidence0041_OutputLimits is §3's R7, as amendment A1 states it: 52
// entries in the eight sections record a limit.output below the 8192 default,
// 10 of them image and video models at a stated 0, so 42 text models; and
// every Claude model allows at least 64,000.
func TestEvidence0041_OutputLimits(t *testing.T) {
	doc := evidence0041Doc(t)
	// The default every provider sends with no WithMaxTokens (R7, E16).
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, nil)
	if err != nil {
		t.Fatal(err)
	}
	limit := st.MaxTokens()
	below, low := 0, 0
	var zero []string
	for _, sec := range []string{"anthropic", "openai", "google", "xai", "opencode", "kilo", "togetherai", "huggingface"} {
		for id, m := range evidence0041Section(t, doc, sec) {
			if m.Limit.Output == nil || *m.Limit.Output >= limit {
				continue
			}
			below++
			if *m.Limit.Output == 0 {
				zero = append(zero, id)
			} else {
				low++
			}
		}
	}
	slices.Sort(zero)
	wantZero := []string{"chatgpt-image-latest", "gpt-image-1", "gpt-image-1-mini", "gpt-image-1.5", "gpt-image-2",
		"grok-imagine-image", "grok-imagine-image-quality", "grok-imagine-video", "grok-imagine-video-1.5",
		"grok-imagine-video-1.5-lite"}
	if below != 52 || low != 42 || !slices.Equal(zero, wantZero) {
		t.Errorf("R7: %d below %d, %d of them text models, at 0 %v; want 52 below 8192, 42, and %v", below,
			limit, low, zero, wantZero)
	}
	smallest := math.MaxInt
	for _, m := range evidence0041Section(t, doc, "anthropic") {
		if m.Limit.Output != nil {
			smallest = min(smallest, *m.Limit.Output)
		}
	}
	if smallest != 64000 {
		t.Errorf("the smallest Claude output limit is %d; want 64000", smallest)
	}
}

// TestEvidence0041_FullListFlags is §4: the first-party curators, over their
// models.dev sections, keep deprecated and tool-less models.
func TestEvidence0041_FullListFlags(t *testing.T) {
	doc := evidence0041Doc(t)
	full := func(key string, curate func([]string, int) []string) ([]string, map[string]modelMetadata) {
		models := evidence0041Section(t, doc, key)
		ids := make([]string, 0, len(models))
		for id := range models {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		return curate(ids, 0), models
	}
	openai, models := full("openai", curateOpenAI)
	deprecated := 0
	for _, id := range openai {
		if models[id].Status == "deprecated" {
			deprecated++
		}
	}
	at := slices.Index(openai, "gpt-3.5-turbo")
	if tools := models["gpt-3.5-turbo"].ToolCall; len(models) != 53 || len(openai) != 42 || deprecated != 11 || at != 31 ||
		tools == nil || *tools {
		t.Errorf("OpenAI: section %d, full list %d, %d deprecated, gpt-3.5-turbo at #%d (tool_call %v); "+
			"want 53, 42, 11, #32 without tools", len(models), len(openai), deprecated, at+1, tools)
	}
	grok, models := full("xai", curateGrok)
	if n := len(grok); n != 8 || grok[n-1] != "grok-4.20-multi-agent-0309" || models[grok[n-1]].ToolCall == nil ||
		*models[grok[n-1]].ToolCall {
		t.Errorf("Grok: full list %v; want 8, the last grok-4.20-multi-agent-0309 without tools", grok)
	}
}

// listingDates0041 is one row of §4's table: a recorded listing, its rows,
// its distinct created values, the rows whose created is 0, and how many of
// the rows with a created and a models.dev release_date fall within 31 days.
type listingDates0041 struct {
	file, section                         string
	rows, distinct, zeroes, within, dated int
}

// TestEvidence0041_ListingDates is §4's table: a listing's created is a
// release date on Kilo and Hugging Face, but not on Zen and Go, whose rows
// share one value.
func TestEvidence0041_ListingDates(t *testing.T) {
	doc := evidence0041Doc(t)
	for _, want := range []listingDates0041{
		// §4 gives no zero count for Kilo: its 5 were measured in Phase 0.
		{"kilo.json", "kilo", 391, 368, 5, 365, 382},
		{"hf.json", "huggingface", 125, 125, 0, 66, 69},
		{"together.json", "togetherai", 265, 231, 29, 11, 16},
		{"zen.json", "opencode", 83, 1, 0, 13, 80},
		{"go.json", "opencode-go", 38, 1, 0, 8, 34},
	} {
		raw, err := os.ReadFile(filepath.Join(evidence0041Dir, want.file))
		if err != nil {
			t.Fatal(err)
		}
		var rows []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		}
		if err := json.Unmarshal(raw, &rows); err != nil {
			var wrapped struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(raw, &wrapped); err != nil || json.Unmarshal(wrapped.Data, &rows) != nil {
				t.Fatalf("%s: neither a list nor {\"data\": [...]}", want.file)
			}
		}
		models := evidence0041Section(t, doc, want.section)
		got := listingDates0041{file: want.file, section: want.section, rows: len(rows)}
		distinct := map[int64]bool{}
		for _, r := range rows {
			distinct[r.Created] = true
			if r.Created == 0 {
				got.zeroes++
				continue
			}
			released, ok := releaseDay(models[r.ID].ReleaseDate)
			if !ok {
				continue
			}
			got.dated++
			created := time.Unix(r.Created, 0).UTC().Truncate(24 * time.Hour)
			if days := math.Abs(created.Sub(released).Hours() / 24); days <= 31 {
				got.within++
			}
		}
		got.distinct = len(distinct)
		if got != want {
			t.Errorf("%s: %+v; want %+v", want.file, got, want)
		}
	}
}

// releaseDay is a models.dev release_date as a day: YYYY-MM-DD, or YYYY-MM
// read as its first day.
func releaseDay(s string) (time.Time, bool) {
	if len(s) == len("2006-01") {
		s += "-01"
	}
	if len(s) < len("2006-01-02") {
		return time.Time{}, false
	}
	day, err := time.Parse(time.DateOnly, s[:len("2006-01-02")])
	return day, err == nil
}

// TestEvidence0041_CostKeyReach is §5's table: how many eligible pairs
// ProfileCapable's order ties on every key before cost.
func TestEvidence0041_CostKeyReach(t *testing.T) {
	doc := evidence0041Doc(t)
	for _, tc := range []struct {
		provider        llmprovider.ProviderID
		usable          func(string) bool
		eligible, reach int
	}{
		// Phase 3: D4's small by price moves OpenAI's and Kilo's counts, and
		// the unusable leave Kilo's eligible set (§5 measured 13 and 144).
		{llmprovider.ProviderOpenAI, isUsableOpenAIChatModel, 24, 8},
		{llmprovider.ProviderClaude, isUsableClaudeTextModel, 17, 3},
		{llmprovider.ProviderTogether, func(string) bool { return true }, 13, 0},
		{llmprovider.ProviderKilo, isUsableKiloModel, 209, 111},
	} {
		models := evidence0041Section(t, doc, modelMetadataKey(tc.provider))
		var ids []string
		for id := range models {
			if tc.usable(id) {
				ids = append(ids, id)
			}
		}
		slices.Sort(ids)
		var eligible []rankCandidate
		for _, id := range ids {
			if c := metadataCandidate(id, models[id], true, rankingNow()); c.eligible(tc.provider) {
				eligible = append(eligible, c)
			}
		}
		norm := newRankNorm(eligible)
		reach := 0
		for i := range eligible {
			for j := i + 1; j < len(eligible); j++ {
				a, b := eligible[i], eligible[j]
				if cmp.Or(boolFirst(a.signal(), b.signal()), cmp.Compare(b.benchOrZero(), a.benchOrZero()),
					cmp.Compare(a.preferredOrInf(), b.preferredOrInf()), boolFirst(!a.small, !b.small),
					cmp.Compare(norm.age(a), norm.age(b))) == 0 {
					reach++
				}
			}
		}
		n := len(eligible)
		if pairs := n * (n - 1) / 2; n != tc.eligible || reach != tc.reach {
			t.Errorf("%s: %d of %d pairs reach the cost key, %d eligible; want %d of %d", tc.provider, reach, pairs, n,
				tc.reach, tc.eligible*(tc.eligible-1)/2)
		}
	}
}

// cachedTurn0041 is D5's price of one cached turn, per million tokens: a
// share o of the tokens is output, and of the rest a share h is read from the
// cache and the remainder sent new, at newPrice.
func cachedTurn0041(read, newPrice, output, h, o float64) float64 {
	return (1-o)*(h*read+(1-h)*newPrice) + o*output
}

// TestEvidence0041_CachedTurn is §7's cross-check against OpenRouter's
// Sonnet 5 example: $2 input, $10 output, $0.20 read, $2.50 write, with
// h = 0.9 and o = 0.01.
func TestEvidence0041_CachedTurn(t *testing.T) {
	const input, output, read, write, h, o = 2.0, 10.0, 0.20, 2.50, 0.90, 0.01
	if uncached := cachedTurn0041(input, input, output, 0, o); math.Abs(uncached-2.08) > 1e-9 {
		t.Errorf("uncached: %v; want 2.08", uncached)
	}
	low, high := cachedTurn0041(read, input, output, h, o), cachedTurn0041(read, write, output, h, o)
	if low < 0.47 || high > 0.53 || low >= high {
		t.Errorf("cached: %v to %v; want both within [0.47, 0.53], the new input at the input price lower", low, high)
	}
}

// TestEvidence0041_AgentTrial is D5's trial: the cached turn choosing the
// members, not only ordering them, on every models.dev section with D4's
// rules. Each eligible model is ordered by a signal, bench, preferred index,
// not small, the release quarter, the cheaper turn, then id, and the six
// taken with D4's group cap. It keeps few of capable's six where no signal
// of strength exists, which is why ProfileAgent's members are capable's.
// Over the 4 x 4 grid of h and o, it counts the distinct bands.
func TestEvidence0041_AgentTrial(t *testing.T) {
	doc := evidence0041Doc(t)
	for _, tc := range []struct {
		provider             llmprovider.ProviderID
		usable               func(string) bool
		kept, size, distinct int
	}{
		{llmprovider.ProviderOpenAI, isUsableOpenAIChatModel, 6, 6, 1},
		{llmprovider.ProviderClaude, isUsableClaudeTextModel, 6, 6, 1},
		{llmprovider.ProviderGemini, func(id string) bool { return isUsableGeminiTextModel(id, []string{"generateContent"}) }, 4, 5, 1},
		{llmprovider.ProviderGrok, isUsableGrokModel, 3, 3, 1},
		{llmprovider.ProviderOpencodeZen, isUsableOpencodeModel, 1, 6, 4},
		{llmprovider.ProviderOpencodeGo, isUsableOpencodeModel, 5, 6, 3},
		{llmprovider.ProviderHuggingFace, isUsableHuggingFaceModel, 6, 6, 1},
		{llmprovider.ProviderTogether, func(string) bool { return true }, 6, 6, 3},
		{llmprovider.ProviderKilo, isUsableKiloModel, 1, 6, 12},
	} {
		models := evidence0041Section(t, doc, modelMetadataKey(tc.provider))
		var ids []string
		for id := range models {
			if tc.usable(id) {
				ids = append(ids, id)
			}
		}
		slices.Sort(ids)
		var cands, eligible []rankCandidate
		for _, id := range ids {
			c := metadataCandidate(id, models[id], true, rankingNow())
			if f := models[id].Family; oneVendor[tc.provider] && f != "" {
				c.group = f
			}
			c.prices = pricesOf(models[id], doc)
			cands = append(cands, c)
			if c.eligible(tc.provider) {
				eligible = append(eligible, c)
			}
		}
		capable := rankRecommended(ProfileCapable, tc.provider, cands, nil, MaxListed)
		band := func(w workload) []string {
			ordered := make([]rankCandidate, len(eligible))
			for i, c := range eligible {
				ordered[i] = c.withTurn(w)
			}
			slices.SortFunc(ordered, func(a, b rankCandidate) int {
				return cmp.Or(boolFirst(a.signal(), b.signal()), cmp.Compare(b.benchOrZero(), a.benchOrZero()),
					cmp.Compare(a.preferredOrInf(), b.preferredOrInf()), boolFirst(!a.small, !b.small),
					cmp.Compare(a.quarter(), b.quarter()), boolFirst(a.turnKnown, b.turnKnown), cmp.Compare(a.turn, b.turn),
					strings.Compare(a.id, b.id))
			})
			var six []string
			groups := map[string]int{}
			for _, c := range ordered {
				if len(six) < MaxListed && groups[c.group] < maxPerRankGroup {
					groups[c.group]++
					six = append(six, c.id)
				}
			}
			return six
		}
		at := band(defaultWorkload)
		kept := 0
		for _, id := range at {
			if slices.Contains(capable, id) {
				kept++
			}
		}
		distinct := map[string]bool{}
		for _, h := range []float64{0.6, 0.75, 0.9, 0.96} {
			for _, o := range []float64{0.005, 0.01, 0.05, 0.12} {
				distinct[strings.Join(band(workload{cacheShare: h, outputShare: o}), ",")] = true
			}
		}
		if kept != tc.kept || len(at) != tc.size || len(capable) != tc.size || len(distinct) != tc.distinct {
			t.Errorf("%s: the price's band %v keeps %d of capable's %d %v, %d distinct over the grid; want %d of %d, %d",
				tc.provider, at, kept, len(capable), capable, len(distinct), tc.kept, tc.size, tc.distinct)
		}
	}
}
