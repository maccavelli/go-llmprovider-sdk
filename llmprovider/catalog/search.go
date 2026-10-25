package catalog

import (
	"cmp"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The search engine (0061-MADR, amendment "Model search, rebuilt"): today's
// tiers, kept whole (model_matcher.go), a tier of every word matched in some
// field, facets included, and three fallbacks, each tried only when the
// tiers before it matched nothing; inside a score, BM25F, then fzf's score,
// then the listing's order (S2).

// The tiers this engine adds below today's (S2).
const (
	// scoreAllWords is every query word matched exactly or as a prefix, in
	// some field, facets included.
	scoreAllWords = 1800
	// scoreNearAll is every query word matched, one at least only within the
	// edit bound: a fallback.
	scoreNearAll = 1500
	// scorePartial is some query words matched, the count added: a fallback.
	scorePartial = 1200
)

// searchField is one of a model's fields BM25F weighs (S2).
type searchField int

const (
	fieldID searchField = iota
	fieldLabel
	fieldFamily
	fieldFacet
	searchFields
)

// fieldWeight and fieldB are each field's BM25F weight and length
// normalisation: none for the short fields (S2).
var (
	fieldWeight = [searchFields]float64{3, 2, 1.5, 1}
	fieldB      = [searchFields]float64{0, 0, 0, 0.5}
)

// bm25K1 is BM25's term-frequency saturation, its common default.
const bm25K1 = 1.2

// The weights of a query word's kinds of match (S2).
const (
	weightExact  = 1.0
	weightPrefix = 0.8
	weightNear1  = 0.6
	weightNear2  = 0.4
	// minPrefixRunes is the shortest query word that matches as a prefix.
	minPrefixRunes = 2
)

// fzf's scores, as its algo.go sets them: a match, a gap's start and each
// extension, the bonus at a word's boundary and on a turn into a digit, a
// consecutive run's, and the first character's multiplier.
const (
	fzfMatch          = 16
	fzfGapStart       = -3
	fzfGapExtension   = -1
	fzfBoundary       = fzfMatch / 2
	fzfCamel123       = fzfBoundary + fzfGapExtension
	fzfConsecutive    = -(fzfGapStart + fzfGapExtension)
	fzfFirstCharTimes = 2
)

// contextBuckets are the context window's words, each matching a window at
// least its size (S3).
var contextBuckets = []struct {
	word   string
	tokens int
}{
	{"8k", 8_000}, {"16k", 16_000}, {"32k", 32_000}, {"64k", 64_000}, {"128k", 128_000},
	{"200k", 200_000}, {"256k", 256_000}, {"400k", 400_000}, {"1m", 1_000_000}, {"2m", 2_000_000},
}

// searchDoc is one model, as the engine reads it: its match, the forms
// today's tiers compare, and its fields' words.
type searchDoc struct {
	match   Match // ID and Label
	lowerID string
	name    string   // the label's display name, lower case
	compact string   // lowerID less its separators
	tokens  []string // lowerID's and name's tokens, for the token-prefix tier
	words   [searchFields][]string
	// length is each field's length for BM25F: its word count, the context
	// buckets counted as one, so that a larger window does not lengthen the
	// facets and lower their score.
	length [searchFields]int
	order  int
}

// searchIndex is a listing's documents, its vocabulary's document
// frequencies, the vocabulary by rune count, and each field's mean length:
// built once, then only read, so one index serves concurrent searches (S5).
// menu and usable are the slices a Catalog's was built from, so that
// Catalog.Search knows a Catalog whose Usable or Menu was replaced.
type searchIndex struct {
	docs   []searchDoc
	df     map[string]int
	byLen  map[int][]string
	avgLen [searchFields]float64
	menu   []MenuEntry
	usable []string
}

// wordsPerModel is the backing array's room per model, for an id, a label
// and their tokens; a model with more grows it. 32 halved the bytes of 16 on
// BenchmarkSearch_1000_0061 (0061-PLAN Phase 4b).
const wordsPerModel = 32

// newSearchIndex indexes n models of provider, at(i) giving the i-th's id
// and facts, nil for none: each id once, compared without case, the first
// kept, in order. A model's facts give its family and facets, and its name
// joins its label. Every document's words and tokens share one backing
// array.
func newSearchIndex(provider llmprovider.ProviderID, n int, at func(int) (string, *ModelInfo)) *searchIndex {
	ix := &searchIndex{docs: make([]searchDoc, 0, n), df: make(map[string]int, n), byLen: map[int][]string{}}
	seen := make(map[string]struct{}, n)
	words := make([]string, 0, wordsPerModel*n)
	since := func(start int) []string { return words[start:len(words):len(words)] }
	var total [searchFields]int
	for i := range n {
		id, info := at(i)
		key := strings.ToLower(id)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		label := Label(provider, id)
		name := displayName(strings.ToLower(label))
		d := searchDoc{match: Match{ID: id, Label: label}, lowerID: key, name: name, compact: compactModelText(key),
			order: len(ix.docs)}
		start := len(words)
		words = appendTokens(appendTokens(words, key), name)
		d.tokens = since(start)
		start = len(words)
		words = appendWords(words, start, key)
		d.words[fieldID] = since(start)
		start = len(words)
		words = appendWords(words, start, name)
		if info != nil && info.Has(FieldName) {
			words = appendWords(words, start, info.Name)
		}
		d.words[fieldLabel] = since(start)
		start = len(words)
		if info != nil && info.Has(FieldFamily) {
			words = appendWords(words, start, info.Family)
		}
		d.words[fieldFamily] = since(start)
		start = len(words)
		var buckets int
		words, buckets = appendFacets(words, info)
		d.words[fieldFacet] = since(start)
		for f := range searchFields {
			d.length[f] = len(d.words[f])
		}
		d.length[fieldFacet] -= max(0, buckets-1)
		for f := range searchFields {
			total[f] += d.length[f]
			for _, w := range d.words[f] {
				if slices.ContainsFunc(d.words[:f], func(earlier []string) bool { return slices.Contains(earlier, w) }) {
					continue // counted once per document
				}
				if ix.df[w] == 0 {
					n := utf8.RuneCountInString(w)
					ix.byLen[n] = append(ix.byLen[n], w)
				}
				ix.df[w]++
			}
		}
		ix.docs = append(ix.docs, d)
	}
	for f := range searchFields {
		if len(ix.docs) > 0 {
			ix.avgLen[f] = float64(total[f]) / float64(len(ix.docs))
		}
	}
	return ix
}

// Search ranks c's models against query, as the package's Search does, and
// searches each model's facts too: its name, its family, and its features
// as words (vision, audio, video, pdf, tools, reasoning, thinking, free,
// the context window from 8k to 2m, each matching a window at least its
// size, and the release year), each only where the facts state it. It
// searches Usable's ids, in Usable's order, each with the facts of Menu's
// entry of the same id, compared without case: a model WithLimit cut from
// Menu is found by its id and label alone. When Usable is empty it searches
// Menu's ids. Labels are those of the provider List listed; a Catalog List
// did not build labels each model by its id. The index List built is read,
// not rebuilt; a Catalog whose Usable or Menu was replaced since is indexed
// again, once per call (0061-MADR, amendment "Model search, rebuilt", S1,
// S3, S5, and "S1 amended in execution").
func (c Catalog) Search(query string) []Match {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	ix := c.index
	if !ix.indexes(c) {
		ix = c.newIndex()
	}
	return ix.query(q)
}

// newIndex indexes c's Usable ids, each with the facts of Menu's entry of the
// same id, the first of its case-folded id; Menu's ids when Usable is empty.
func (c Catalog) newIndex() *searchIndex {
	var ix *searchIndex
	if len(c.Usable) == 0 {
		ix = newSearchIndex(c.provider, len(c.Menu), func(i int) (string, *ModelInfo) { return c.Menu[i].ID, &c.Menu[i].Info })
	} else {
		facts := make(map[string]*ModelInfo, len(c.Menu))
		for i := range c.Menu {
			if key := strings.ToLower(c.Menu[i].ID); facts[key] == nil {
				facts[key] = &c.Menu[i].Info
			}
		}
		ix = newSearchIndex(c.provider, len(c.Usable), func(i int) (string, *ModelInfo) {
			return c.Usable[i], facts[strings.ToLower(c.Usable[i])]
		})
	}
	ix.menu, ix.usable = c.Menu, c.Usable
	return ix
}

// indexes reports whether ix was built from c's Usable and Menu as they are.
func (ix *searchIndex) indexes(c Catalog) bool {
	return ix != nil && sameSlice(ix.menu, c.Menu) && sameSlice(ix.usable, c.Usable)
}

// sameSlice reports whether a and b are one slice: the same length, and,
// when not empty, the same first element.
func sameSlice[T any](a, b []T) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

// query ranks ix's documents against q, lower case, trimmed and not empty:
// a glob, else the engine.
func (ix *searchIndex) query(q string) []Match {
	if strings.ContainsAny(q, "*?") {
		candidates := make([]Match, len(ix.docs))
		for i, d := range ix.docs {
			candidates[i] = d.match
		}
		return sortByScore(globMatches(candidates, q))
	}
	return ix.search(q)
}

// appendFacets appends a model's facts as words, each only where the facts
// state its field, none for nil (S3), and says how many are context buckets.
func appendFacets(out []string, info *ModelInfo) (_ []string, buckets int) {
	if info == nil {
		return out, 0
	}
	if info.Has(FieldInput) {
		for _, m := range info.Input {
			switch m {
			case ModalityImage:
				out = append(out, "vision")
			case ModalityAudio:
				out = append(out, "audio")
			case ModalityVideo:
				out = append(out, "video")
			case ModalityPDF:
				out = append(out, "pdf")
			}
		}
	}
	if info.Has(FieldTools) && info.Tools {
		out = append(out, "tools")
	}
	if info.Has(FieldReasoning) && info.Reasoning.Supported {
		out = append(out, "reasoning", "thinking")
	}
	if info.Has(FieldCostInput) && info.Has(FieldCostOutput) && info.Cost.Input == 0 && info.Cost.Output == 0 {
		out = append(out, "free")
	}
	if info.Has(FieldContextWindow) {
		for _, b := range contextBuckets {
			if info.ContextWindow >= b.tokens {
				out = append(out, b.word)
				buckets++
			}
		}
	}
	if info.Has(FieldReleased) && !info.Released.IsZero() {
		out = append(out, strconv.Itoa(info.Released.Year()))
	}
	return out, buckets
}

// searchWords is s's lower-case words, once each, in order (appendWords).
func searchWords(s string) []string { return appendWords(nil, 0, s) }

// appendWords appends s's lower-case words to dst, each not already in
// dst[base:]: s split on the model separators and spaces, and between a
// letter and a digit, a size such as 8b, 128k or 1m kept whole (S2).
func appendWords(dst []string, base int, s string) []string {
	for token := range strings.FieldsFuncSeq(strings.ToLower(s), isModelSeparator) {
		start, prev := 0, rune(-1)
		for i, r := range token {
			letterToDigit := unicode.IsLetter(prev) && unicode.IsDigit(r)
			digitToLetter := unicode.IsDigit(prev) && unicode.IsLetter(r) && !isSizeUnit(token[i:])
			if letterToDigit || digitToLetter {
				dst = addWord(dst, base, token[start:i])
				start = i
			}
			prev = r
		}
		dst = addWord(dst, base, token[start:])
	}
	return dst
}

// addWord appends w to dst unless it is empty or already in dst[base:].
func addWord(dst []string, base int, w string) []string {
	if w == "" || slices.Contains(dst[base:], w) {
		return dst
	}
	return append(dst, w)
}

// appendTokens appends lower-case s's tokens, as modelTokens splits them.
func appendTokens(dst []string, s string) []string {
	for token := range strings.FieldsFuncSeq(s, isModelSeparator) {
		dst = append(dst, token)
	}
	return dst
}

// isSizeUnit reports whether rest, after a digit, is a size's unit alone: b,
// k, m or t.
func isSizeUnit(rest string) bool {
	switch rest {
	case "b", "k", "m", "t":
		return true
	}
	return false
}

// editsWithin is the optimal string alignment distance of a and b, an
// adjacent transposition one edit, Damerau's, and whether it is at most
// bound. It stops once a row's every cell passes the bound.
func editsWithin(a, b []rune, bound int) (int, bool) {
	if len(a)-len(b) > bound || len(b)-len(a) > bound {
		return 0, false
	}
	prev2 := make([]int, len(b)+1)
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		rowMin := cur[0]
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
			rowMin = min(rowMin, cur[j])
		}
		if rowMin > bound {
			return 0, false
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(b)], prev[len(b)] <= bound
}

// editBound is how many edits a query word of n runes may be from a word it
// matches: none under 4, one from 4, two from 8 (S2).
func editBound(n int) int {
	switch {
	case n >= 8:
		return 2
	case n >= 4:
		return 1
	}
	return 0
}

// nearWords are the vocabulary's words within w's edit bound, other than w,
// each with its weight: weightNear1 at one edit, weightNear2 at two.
func (ix *searchIndex) nearWords(w string) map[string]float64 {
	r := []rune(w)
	bound := editBound(len(r))
	if bound == 0 {
		return nil
	}
	out := map[string]float64{}
	for n := len(r) - bound; n <= len(r)+bound; n++ {
		for _, v := range ix.byLen[n] {
			if v == w {
				continue
			}
			if d, ok := editsWithin(r, []rune(v), bound); ok {
				out[v] = weightNear1
				if d == 2 {
					out[v] = weightNear2
				}
			}
		}
	}
	return out
}

// searchQuery is a query in the forms the tiers and the engine compare.
type searchQuery struct {
	q, compact string
	tokens     []string // modelTokens, for the token-prefix tier
	words      []string // searchWords, for the engine
	near       []map[string]float64
	pattern    []rune // for fzf
}

// wordHit is how one query word matched one document: its weight in each
// field, and its best weight, field and document word, for the IDF and Why.
type wordHit struct {
	weight [searchFields]float64
	best   float64
	word   string
	field  searchField
}

// hit is how the query's word i matched d.
func (sq *searchQuery) hit(d *searchDoc, i int) wordHit {
	w := sq.words[i]
	prefixes := utf8.RuneCountInString(w) >= minPrefixRunes
	var h wordHit
	for f := range searchFields {
		for _, dw := range d.words[f] {
			var kind float64
			switch {
			case dw == w:
				kind = weightExact
			case prefixes && strings.HasPrefix(dw, w):
				kind = weightPrefix
			default:
				kind = sq.near[i][dw]
			}
			h.weight[f] = max(h.weight[f], kind)
			if kind > h.best {
				h.best, h.word, h.field = kind, dw, f
			}
		}
	}
	return h
}

// scored is one document's result: its score, BM25F's, fzf's, and why.
type scored struct {
	doc   *searchDoc
	score int
	bm25  float64
	fzf   int
	why   string
}

// search ranks ix's documents against q, lower case, trimmed, not empty and
// not a glob: today's tiers and scoreAllWords, else the fallbacks in turn
// (S2).
func (ix *searchIndex) search(q string) []Match {
	sq := &searchQuery{q: q, compact: compactModelText(q), tokens: modelTokens(q), words: searchWords(q)}
	sq.pattern = []rune(sq.compact)
	sq.near = make([]map[string]float64, len(sq.words))
	for i, w := range sq.words {
		sq.near[i] = ix.nearWords(w)
	}
	hits := make([]wordHit, len(sq.words))
	var tiers, nearAll, partial []scored
	for i := range ix.docs {
		d := &ix.docs[i]
		strong, matched := 0, 0
		for j := range sq.words {
			hits[j] = sq.hit(d, j)
			if hits[j].best >= weightPrefix {
				strong++
			}
			if hits[j].best > 0 {
				matched++
			}
		}
		s := scored{doc: d}
		group := &tiers
		switch tier, ok := tierScore(d, sq); {
		case ok:
			s.score, s.why = tier, tierWhy(tier, hits)
		case len(sq.words) > 0 && strong == len(sq.words):
			s.score, s.why = scoreAllWords, withFacets("every word", hits)
		case len(sq.words) > 0 && matched == len(sq.words):
			s.score, s.why, group = scoreNearAll, nearWhy(hits), &nearAll
		case len(sq.words) > 1 && matched > 0:
			s.score, s.why, group = scorePartial+matched, partialWhy(sq.words, hits), &partial
		default:
			continue
		}
		s.bm25, s.fzf = ix.bm25(d, hits), fzfScore([]rune(d.lowerID), sq.pattern)
		*group = append(*group, s)
	}
	for _, group := range [][]scored{tiers, nearAll, partial} {
		if len(group) > 0 {
			return ranked(group)
		}
	}
	if utf8.RuneCountInString(q) < minSubsequenceQuery {
		return nil
	}
	var subsequence []scored
	for i := range ix.docs {
		d := &ix.docs[i]
		if bonus, ok := subsequenceBonus(d.lowerID, q); ok {
			subsequence = append(subsequence, scored{doc: d, score: scoreSubsequence + bonus, why: "letters in order",
				fzf: fzfScore([]rune(d.lowerID), sq.pattern)})
		}
	}
	return ranked(subsequence)
}

// bm25 is d's BM25F score for the query words' hits: per word, its weight in
// each field times the field's weight, over the field's length where its b
// asks, summed, saturated by k1, and weighed by the IDF of the document word
// it matched best.
func (ix *searchIndex) bm25(d *searchDoc, hits []wordHit) float64 {
	n := float64(len(ix.docs))
	total := 0.0
	for _, h := range hits {
		if h.best == 0 {
			continue
		}
		tf := 0.0
		for f := range searchFields {
			if h.weight[f] == 0 {
				continue
			}
			norm := 1.0
			if fieldB[f] > 0 && ix.avgLen[f] > 0 {
				norm = 1 - fieldB[f] + fieldB[f]*float64(d.length[f])/ix.avgLen[f]
			}
			tf += fieldWeight[f] * h.weight[f] / norm
		}
		df := float64(ix.df[h.word])
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		total += idf * tf * (bm25K1 + 1) / (tf + bm25K1)
	}
	return total
}

// ranked orders a group by its score, then BM25F's, then fzf's, then the
// listing's order, and returns its matches with their scores and reasons.
func ranked(group []scored) []Match {
	slices.SortFunc(group, func(a, b scored) int {
		return cmp.Or(cmp.Compare(b.score, a.score), cmp.Compare(b.bm25, a.bm25), cmp.Compare(b.fzf, a.fzf),
			cmp.Compare(a.doc.order, b.doc.order))
	})
	out := make([]Match, len(group))
	for i, s := range group {
		out[i] = s.doc.match
		out[i].Score, out[i].Why = s.score, s.why
	}
	return out
}

// tierWhy names today's tiers, with the facets the words matched.
func tierWhy(tier int, hits []wordHit) string {
	var why string
	switch tier {
	case scoreExact:
		why = "exact id"
	case scoreIDPrefix:
		why = "id prefix"
	case scoreIDSubstring:
		why = "in the id"
	case scoreCompactSubstring:
		why = "in the id, without separators"
	case scoreLabelSubstring:
		why = "in the name"
	default:
		why = "every word"
	}
	return withFacets(why, hits)
}

// withFacets is why, then the facets the hits matched, in the query's order.
func withFacets(why string, hits []wordHit) string {
	var facets []string
	for _, h := range hits {
		if h.best > 0 && h.field == fieldFacet {
			facets = append(facets, h.word)
		}
	}
	if len(facets) == 0 {
		return why
	}
	return why + ": " + strings.Join(facets, ", ")
}

// nearWhy names the first document word matched only within the edit bound.
func nearWhy(hits []wordHit) string {
	for _, h := range hits {
		if h.best > 0 && h.best < weightPrefix {
			return "near " + h.word
		}
	}
	return "every word" // unreachable: a near tier has a near word
}

// partialWhy names the query words matched.
func partialWhy(words []string, hits []wordHit) string {
	var got []string
	for i, h := range hits {
		if h.best > 0 {
			got = append(got, words[i])
		}
	}
	return "some words: " + strings.Join(got, ", ")
}

// fzfScore is fzf's score of pattern in text, its version 1: the leftmost
// match's end, the shortest window back from it, then each matched rune's
// score and bonus (at a word's boundary, or on a turn into a digit), a
// consecutive run's bonus carried, the first rune's doubled, and each gap's
// penalty. A pattern not in text, or none, scores 0.
func fzfScore(text, pattern []rune) int {
	if len(pattern) == 0 {
		return 0
	}
	pi, end := 0, -1
	for i, r := range text {
		if r != pattern[pi] {
			continue
		}
		pi++
		if pi == len(pattern) {
			end = i
			break
		}
	}
	if end < 0 {
		return 0
	}
	pi, start := len(pattern)-1, end
	for i := end; i >= 0; i-- {
		if text[i] != pattern[pi] {
			continue
		}
		pi--
		if pi < 0 {
			start = i
			break
		}
	}
	score, inGap, consecutive, firstBonus := 0, false, 0, 0
	pi = 0
	for i := start; i <= end; i++ {
		if pi < len(pattern) && text[i] == pattern[pi] {
			bonus := 0
			switch {
			case i == 0 || isModelSeparator(text[i-1]):
				bonus = fzfBoundary
			case unicode.IsDigit(text[i]) && !unicode.IsDigit(text[i-1]):
				bonus = fzfCamel123
			}
			if consecutive == 0 {
				firstBonus = bonus
			} else {
				if bonus >= fzfBoundary && bonus > firstBonus {
					firstBonus = bonus
				}
				bonus = max(bonus, firstBonus, fzfConsecutive)
			}
			if pi == 0 {
				bonus *= fzfFirstCharTimes
			}
			score += fzfMatch + bonus
			inGap, consecutive = false, consecutive+1
			pi++
			continue
		}
		if inGap {
			score += fzfGapExtension
		} else {
			score += fzfGapStart
		}
		inGap, consecutive, firstBonus = true, 0, 0
	}
	return score
}
