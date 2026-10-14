package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-PLAN Phase 2: a listing's date is a release date only where measured
// to be one (0041-MADR D1): OpenAI's, Anthropic's, xAI's, Kilo's and Hugging
// Face's, by the step 3 table; never Zen's, Go's, Together's or Gemini's.

// datesDoc is a document that gives no release date, so a Released can only
// come from a listing.
const datesDoc = `{"openai":{"models":{"m-a":{"name":"A"},"m-b":{"name":"B"}}},` +
	`"anthropic":{"models":{"m-a":{"name":"A"},"m-b":{"name":"B"}}},` +
	`"xai":{"models":{"m-a":{"name":"A"},"m-b":{"name":"B"}}},` +
	`"kilo":{"models":{"m-a":{"name":"A"},"m-b":{"name":"B"}}},` +
	`"huggingface":{"models":{"m-a":{"name":"A"},"m-b":{"name":"B"}}},` +
	`"togetherai":{"models":{"m-a":{"name":"A"},"m-b":{"name":"B"}}}}`

// datedListing is a listing of m-a and m-b in provider's shape, dated a and
// b: a Unix time, or Anthropic's RFC 3339.
func datedListing(provider llmprovider.ProviderID, a, b int64) string {
	switch provider {
	case llmprovider.ProviderClaude:
		at := func(s int64) string { return time.Unix(s, 0).UTC().Format(time.RFC3339) }
		return `{"data":[{"id":"m-a","created_at":"` + at(a) + `"},{"id":"m-b","created_at":"` + at(b) + `"}],"has_more":false}`
	case llmprovider.ProviderTogether:
		return `[{"id":"m-a","type":"chat","created":` + itoa(a) + `},{"id":"m-b","type":"chat","created":` + itoa(b) + `}]`
	case llmprovider.ProviderHuggingFace:
		row := func(id string, c int64) string {
			return `{"id":"` + id + `","created":` + itoa(c) + `,"providers":[{"status":"live","supports_tools":true}]}`
		}
		return `{"data":[` + row("m-a", a) + `,` + row("m-b", b) + `]}`
	}
	return `{"data":[{"id":"m-a","created":` + itoa(a) + `},{"id":"m-b","created":` + itoa(b) + `}]}`
}

func itoa(n int64) string { raw, _ := json.Marshal(n); return string(raw) }

// TestListingDates_0041: each qualifying listing's dates set Released from the
// listing; one date on every row sets none; a zero sets none; Together's,
// which did not qualify, sets none.
func TestListingDates_0041(t *testing.T) {
	const a, b = 1780000000, 1785000000
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		reads    bool
	}{
		{llmprovider.ProviderOpenAI, true},
		{llmprovider.ProviderClaude, true},
		{llmprovider.ProviderGrok, true},
		{llmprovider.ProviderKilo, true},
		{llmprovider.ProviderHuggingFace, true},
		{llmprovider.ProviderTogether, false},
	} {
		for _, dates := range []struct {
			name string
			a, b int64
			want bool
		}{
			{"two dates", a, b, tc.reads},
			{"one date on every row", a, a, false},
			{"a zero", 0, b, false},
		} {
			resetLookupCaches(t)
			s := newLookupServer(t, datesDoc, 200, datedListing(tc.provider, dates.a, dates.b))
			info, err := LookupModel(context.Background(), tc.provider, llmprovider.NewStaticToken("k"), "m-a", lookupOpts(s)...)
			if err != nil {
				t.Fatalf("%s, %s: %v", tc.provider, dates.name, err)
			}
			got := info.Has(FieldReleased) && info.SourceOf(FieldReleased) == "listing:"+string(tc.provider)
			if got != dates.want || got && !info.Released.Equal(time.Unix(dates.a, 0).UTC().Truncate(24*time.Hour)) {
				t.Errorf("%s, %s: Released %v from %q; want from the listing %t", tc.provider, dates.name, info.Released,
					info.SourceOf(FieldReleased), dates.want)
			}
		}
	}
}

// firstPartyDir is the first-party listings Phase 2 recorded.
var firstPartyDir = filepath.Join("testdata", "firstparty-2026-10-14")

// firstPartyPage is the first page of a recorded first-party listing.
func firstPartyPage(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(firstPartyDir, name))
	if err != nil {
		t.Fatal(err)
	}
	var pages []json.RawMessage
	if err := json.Unmarshal(raw, &pages); err != nil || len(pages) == 0 {
		t.Fatalf("%s: %v", name, err)
	}
	return string(pages[0])
}

// TestListingDates_FirstParty_0041: on the recorded listings, OpenAI's,
// Anthropic's and xAI's dates set Released, Gemini's listing, which has no
// date, sets none, and OpenAI's shutdown_date is Retires.
func TestListingDates_FirstParty_0041(t *testing.T) {
	for _, tc := range []struct {
		provider     llmprovider.ProviderID
		file, model  string
		releasedFrom bool
	}{
		{llmprovider.ProviderOpenAI, "openai.json", "gpt-4.1", true},
		{llmprovider.ProviderClaude, "claude.json", "claude-haiku-4-5-20251001", true},
		{llmprovider.ProviderGrok, "grok.json", "grok-4.6", true},
		{llmprovider.ProviderGemini, "gemini.json", "gemini-2.5-flash", false},
	} {
		resetLookupCaches(t)
		s := newLookupServer(t, `{}`, 200, firstPartyPage(t, tc.file))
		info, err := LookupModel(context.Background(), tc.provider, llmprovider.NewStaticToken("k"), tc.model, lookupOpts(s)...)
		if err != nil {
			t.Fatalf("%s: %v", tc.provider, err)
		}
		if got := info.Has(FieldReleased); got != tc.releasedFrom || got && info.Released.IsZero() {
			t.Errorf("%s/%s: Released %v, known %t; want known %t", tc.provider, tc.model, info.Released, got, tc.releasedFrom)
		}
	}
	// A recorded OpenAI row with a shutdown_date is Retires.
	page := firstPartyPage(t, "openai.json")
	var listing struct {
		Data []struct {
			ID           string  `json:"id"`
			ShutdownDate *string `json:"shutdown_date"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(page), &listing); err != nil {
		t.Fatal(err)
	}
	retiring := ""
	for _, row := range listing.Data {
		if row.ShutdownDate != nil && *row.ShutdownDate != "" {
			retiring = row.ID
			break
		}
	}
	if retiring == "" {
		t.Fatal("no recorded OpenAI row has a shutdown_date")
	}
	resetLookupCaches(t)
	s := newLookupServer(t, `{}`, 200, page)
	info, err := LookupModel(context.Background(), llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"), retiring, lookupOpts(s)...)
	if err != nil || !info.Has(FieldRetires) || info.Retires.IsZero() {
		t.Errorf("%s: Retires %v, known %t, %v; want the shutdown_date", retiring, info.Retires, info.Has(FieldRetires), err)
	}
}

// TestCanonicalID_0041: canonical_model_id is CanonicalID; a gateway model
// whose entry lacks cache_read takes its canonical first-party model's, and
// one whose canonical model has none, or that has its own, takes nothing.
func TestCanonicalID_0041(t *testing.T) {
	meta, err := ParseMetadata(strings.NewReader(`{` +
		`"openai":{"models":{"gpt-x":{"name":"X","cost":{"input":1,"output":4,"cache_read":0.25}},` +
		`"gpt-y":{"name":"Y","cost":{"input":1,"output":4}}}},` +
		`"kilo":{"models":{` +
		`"openai/gpt-x":{"name":"X","canonical_model_id":"openai/gpt-x","cost":{"input":1.1,"output":4.4}},` +
		`"openai/gpt-x-own":{"name":"X own","canonical_model_id":"openai/gpt-x","cost":{"input":1.1,"output":4.4,"cache_read":0.5}},` +
		`"openai/gpt-y":{"name":"Y","canonical_model_id":"openai/gpt-y","cost":{"input":1.1,"output":4.4}},` +
		`"vendor/plain":{"name":"Plain","cost":{"input":1,"output":2}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id        string
		canonical string
		read      float64
		known     bool
	}{
		{"openai/gpt-x", "openai/gpt-x", 0.25, true},
		{"openai/gpt-x-own", "openai/gpt-x", 0.5, true},
		{"openai/gpt-y", "openai/gpt-y", 0, false},
		{"vendor/plain", "", 0, false},
	} {
		info, ok := meta.Model(llmprovider.ProviderKilo, tc.id)
		if !ok {
			t.Fatalf("%s not found", tc.id)
		}
		if info.CanonicalID != tc.canonical || info.Has(FieldCanonicalID) != (tc.canonical != "") {
			t.Errorf("%s: CanonicalID %q, known %t; want %q", tc.id, info.CanonicalID, info.Has(FieldCanonicalID), tc.canonical)
		}
		if info.Has(FieldCostCacheRead) != tc.known || info.Cost.CacheRead != tc.read {
			t.Errorf("%s: cache_read %v, known %t; want %v, %t", tc.id, info.Cost.CacheRead, info.Has(FieldCostCacheRead),
				tc.read, tc.known)
		}
	}
	if rates, ok := pricesByCanonical(meta.doc, "openai/gpt-x"); !ok || rates.CacheRead != 0.25 {
		t.Errorf("pricesByCanonical(openai/gpt-x) = %+v, %t; want cache_read 0.25", rates, ok)
	}
	if _, ok := pricesByCanonical(meta.doc, "nowhere/model"); ok {
		t.Error("pricesByCanonical found a canonical model no first-party section lists")
	}
}
