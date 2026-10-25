package catalog

import (
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Canonical ids (0061-MADR E2; its amendment "E2 and E4's menu, made exact
// before Phase 4c", X1, X2, X5): a server's model id, and models.dev's, in
// one form, so that a server's model takes models.dev's facts only where
// the two forms are equal, never by a near match.

// sourceCanonical names the layer of models.dev's facts read by canonical id.
const sourceCanonical = "models.dev (canonical id)"

// quantToken is a quantisation or format token, dropped from a canonical
// id: q or iq and a digit, then letters, digits and _ (q4_k_m, iq3_xxs);
// awq, gptq, fp8, fp16, bf16, int4, int8, gguf, mlx; and a number of bits.
var quantToken = regexp.MustCompile(`^(?:i?q\d[a-z0-9_]*|awq|gptq|fp8|fp16|bf16|int4|int8|gguf|mlx|\d+bit)$`)

// roleTokens are dropped from the variant key, not from the canonical id
// (decision 1).
var roleTokens = []string{"instruct", "chat", "it", "base"}

// canonicalTokens is id's canonical tokens (X1): lower case; up to the last
// "/" dropped; an Ollama tag split at the last ":", latest dropped; split on
// "-", ".", ":" and spaces; a quantisation or format token dropped; a "_"
// within a token becoming "-".
func canonicalTokens(id string) []string {
	s := strings.ToLower(strings.TrimSpace(id))
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, ":"); i >= 0 && s[i+1:] == "latest" {
		s = s[:i]
	}
	var out []string
	for _, t := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune("-.: ", r) }) {
		if !quantToken.MatchString(t) {
			out = append(out, strings.ReplaceAll(t, "_", "-"))
		}
	}
	return out
}

// canonicalID is id's canonical form (X1): its canonical tokens joined by
// "-".
func canonicalID(id string) string { return strings.Join(canonicalTokens(id), "-") }

// variantKey is the key id's variants share (X2, decision 1): its canonical
// tokens less the role tokens.
func variantKey(id string) string {
	return strings.Join(slices.DeleteFunc(canonicalTokens(id), func(t string) bool { return slices.Contains(roleTokens, t) }), "-")
}

// plainness is how many tokens id has, split on "-", "_", ".", ":" and "/":
// the plainest variant has the fewest (X2).
func plainness(id string) int {
	return len(strings.FieldsFunc(id, func(r rune) bool { return strings.ContainsRune("-_.:/", r) }))
}

// canonicalIndex is a document's facts by canonical id, merged as X5 says,
// built once per document the first time it is read.
type canonicalIndex struct {
	once    sync.Once
	byCanon map[string]ModelInfo
}

// canonicalLimits are the fields a tie resolves to the smaller of (X5).
var canonicalLimits = []Field{FieldContextWindow, FieldMaxInputTokens, FieldMaxOutputTokens}

// modelFor is what the document says of model: in provider's section, or,
// for a chatcompat provider's, by its canonical id (0061-MADR E2, X1).
func (m Metadata) modelFor(cfg config, provider llmprovider.ProviderID, model string) (ModelInfo, bool) {
	if cfg.compat {
		return m.canonicalModel(model)
	}
	return m.Model(provider, model)
}

// canonicalModel is what the document says of a server's model by its
// canonical id (X1, X5): each field the value most of the matching entries
// give; a tied limit the smaller; any other tie unknown. Its fields' source
// is sourceCanonical, and its ID is id.
func (m Metadata) canonicalModel(id string) (ModelInfo, bool) {
	if m.doc == nil {
		return ModelInfo{}, false
	}
	m.doc.canonical.once.Do(func() { m.doc.canonical.byCanon = buildCanonical(m) })
	info, ok := m.doc.canonical.byCanon[canonicalID(id)]
	if !ok {
		return ModelInfo{}, false
	}
	info.ID = id
	return info, true
}

// buildCanonical groups every entry of every section by canonical id, the
// sections in name order, and merges each group (X5).
func buildCanonical(m Metadata) map[string]ModelInfo {
	keys := make([]string, 0, len(m.doc.sections))
	for k := range m.doc.sections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	groups := map[string][]ModelInfo{}
	for _, k := range keys {
		models, ok := m.doc.models(k)
		if !ok {
			continue
		}
		ids := make([]string, 0, len(models))
		for id := range models {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if c := canonicalID(id); c != "" {
				groups[c] = append(groups[c], withCanonicalCacheRead(m.doc, modelInfoFrom(id, models[id])))
			}
		}
	}
	out := make(map[string]ModelInfo, len(groups))
	for c, infos := range groups {
		if merged := mergeCanonical(infos); merged.known != 0 {
			out[c] = merged
		}
	}
	return out
}

// mergeCanonical merges the entries one canonical id names (X5).
func mergeCanonical(infos []ModelInfo) ModelInfo {
	merged := ModelInfo{layer: sourceCanonical}
	var known []Field
	for f := range fieldCount {
		var values []ModelInfo // f's value alone, one per entry stating it
		var counts []int
		for _, info := range infos {
			if !info.Has(f) {
				continue
			}
			var v ModelInfo
			copyField(&v, info, f)
			if i := slices.IndexFunc(values, func(w ModelInfo) bool { return reflect.DeepEqual(w, v) }); i >= 0 {
				counts[i]++
				continue
			}
			values, counts = append(values, v), append(counts, 1)
		}
		if winner, ok := majority(f, values, counts); ok {
			copyField(&merged, winner, f)
			known = append(known, f)
		}
	}
	return merged.Known(known...)
}

// majority is the value most entries give for f; on a tie, a limit's
// smaller value, and otherwise none.
func majority(f Field, values []ModelInfo, counts []int) (ModelInfo, bool) {
	if len(values) == 0 {
		return ModelInfo{}, false
	}
	top := slices.Max(counts)
	var tied []ModelInfo
	for i, n := range counts {
		if n == top {
			tied = append(tied, values[i])
		}
	}
	if len(tied) == 1 {
		return tied[0], true
	}
	if !slices.Contains(canonicalLimits, f) {
		return ModelInfo{}, false
	}
	return slices.MinFunc(tied, func(a, b ModelInfo) int { return limitOf(a, f) - limitOf(b, f) }), true
}

// limitOf is a limit field's value.
func limitOf(info ModelInfo, f Field) int {
	switch f {
	case FieldMaxInputTokens:
		return info.MaxInputTokens
	case FieldMaxOutputTokens:
		return info.MaxOutputTokens
	}
	return info.ContextWindow
}
