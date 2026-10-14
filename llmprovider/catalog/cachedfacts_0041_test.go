package catalog

import (
	"context"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-PLAN Phase 1, 0041-MADR D1: a request reads the facts already cached,
// and never starts a listing fetch.

// TestCachedFacts_NoListingFetch_0041: with a cold cache, CachedFacts makes no
// listing request, where CachedModel makes one, and starts no document fetch
// either (0041-MADR A3): it reads the document only once something else has
// fetched it.
func TestCachedFacts_NoListingFetch_0041(t *testing.T) {
	resetLookupCaches(t)
	s := newLookupServer(t, lookupDoc, 200, claudeListing0033)
	opts := append(lookupOpts(s), llmprovider.WithAPIKey("k"))
	for range 2 {
		if _, ok := CachedFacts(context.Background(), llmprovider.ProviderClaude, "claude-x", opts...); ok {
			t.Error("CachedFacts on a cold cache knew the model")
		}
		// A fetch it had started would have finished, and cached the document.
		waitListingFetches()
		waitMetadataFetches()
	}
	if n := s.listings.Load(); n != 0 {
		t.Fatalf("CachedFacts made %d listing requests; want 0", n)
	}
	if _, err := LookupMetadataWith(context.Background(), llmprovider.ProviderClaude, opts...); err != nil {
		t.Fatal(err)
	}
	info, ok := CachedFacts(context.Background(), llmprovider.ProviderClaude, "claude-x", opts...)
	if !ok || info.MaxOutputTokens != 128000 || info.SourceOf(FieldMaxOutputTokens) != sourceModelsDev {
		t.Errorf("CachedFacts once the document is fetched: %+v, %t; want models.dev's 128000", info, ok)
	}
	CachedModel(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-x", opts...)
	waitListingFetches()
	if n := s.listings.Load(); n != 1 {
		t.Errorf("CachedModel made %d listing requests; want 1", n)
	}
}

// TestCachedFacts_Precedence_0041: field by field, a caller layer beats the
// listing a List already read, which beats models.dev.
func TestCachedFacts_Precedence_0041(t *testing.T) {
	resetLookupCaches(t)
	s := newLookupServer(t, lookupDoc, 200, claudeListing0033)
	opts := append(lookupOpts(s), llmprovider.WithAPIKey("k"))
	// The listing a List call reads, and the document, both cached.
	if _, err := LookupModel(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-x",
		opts...); err != nil {
		t.Fatal(err)
	}
	layer := staticLayer{name: "caller", info: ModelInfo{ID: "claude-x", ContextWindow: 7}.Known(FieldContextWindow)}
	info, ok := CachedFacts(context.Background(), llmprovider.ProviderClaude, "claude-x",
		append(opts, WithModelLayer(layer))...)
	if !ok {
		t.Fatal("CachedFacts knew nothing")
	}
	for _, tc := range []struct {
		field  Field
		source string
		got    any
		want   any
	}{
		{FieldContextWindow, "caller", info.ContextWindow, 7},
		{FieldMaxOutputTokens, "listing:claude", info.MaxOutputTokens, 100000},
		{FieldCostInput, sourceModelsDev, info.Cost.Input, 3.0},
	} {
		if src := info.SourceOf(tc.field); src != tc.source || tc.got != tc.want {
			t.Errorf("field %v: %v from %q; want %v from %q", tc.field, tc.got, src, tc.want, tc.source)
		}
	}
	if n := s.listings.Load(); n != 1 {
		t.Errorf("listing requests %d; want LookupModel's 1, and none from CachedFacts", n)
	}
}

// TestModelReasoning_ReplayField_0041 (0041-MADR A3): models.dev's
// interleaved.field is ModelReasoning.ReplayField; interleaved true, or
// absent, sets nothing; and a caller layer's value wins.
func TestModelReasoning_ReplayField_0041(t *testing.T) {
	meta, err := ParseMetadata(strings.NewReader(`{"togetherai":{"models":{` +
		`"a":{"name":"A","interleaved":{"field":"reasoning_content"}},` +
		`"b":{"name":"B","interleaved":true},` +
		`"c":{"name":"C"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"a": "reasoning_content", "b": "", "c": ""} {
		info, ok := meta.Model(llmprovider.ProviderTogether, id)
		if !ok || info.Reasoning.ReplayField != want || info.Has(FieldReasoningReplay) != (want != "") {
			t.Errorf("%s: %q, known %t; want %q", id, info.Reasoning.ReplayField, info.Has(FieldReasoningReplay), want)
		}
	}
	resetLookupCaches(t)
	s := newLookupServer(t, `{"togetherai":{"models":{"a":{"name":"A","interleaved":{"field":"reasoning_content"}}}}}`,
		200, "[]")
	opts := lookupOpts(s)
	if _, err := LookupMetadataWith(context.Background(), llmprovider.ProviderTogether, opts...); err != nil {
		t.Fatal(err)
	}
	layer := staticLayer{name: "caller", info: ModelInfo{ID: "a",
		Reasoning: ModelReasoning{ReplayField: "reasoning"}}.Known(FieldReasoningReplay)}
	for _, tc := range []struct {
		opts   []llmprovider.Option
		want   string
		source string
	}{
		{opts, "reasoning_content", sourceModelsDev},
		{append(opts, WithModelLayer(layer)), "reasoning", "caller"},
	} {
		info, _ := CachedFacts(context.Background(), llmprovider.ProviderTogether, "a", tc.opts...)
		if info.Reasoning.ReplayField != tc.want || info.SourceOf(FieldReasoningReplay) != tc.source {
			t.Errorf("CachedFacts: %q from %q; want %q from %q", info.Reasoning.ReplayField,
				info.SourceOf(FieldReasoningReplay), tc.want, tc.source)
		}
	}
}
