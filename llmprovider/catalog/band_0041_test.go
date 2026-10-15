package catalog

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-PLAN Phase 3, 0041-MADR D4: "small" from the 3:1 blended price, the
// unusable never eligible, one-vendor catalogs grouped by family, and the
// first-party providers ranked by facts.

func ptrBool(v bool) *bool { return &v }

// TestSmallFromPrice_0041: with a known positive price, small is a 3:1 blend
// of $1 or less; with none, the name decides.
func TestSmallFromPrice_0041(t *testing.T) {
	priced := func(in, out float64) modelMetadata {
		return modelMetadata{Name: "M", Cost: &modelCost{Input: in, Output: out}, Limit: struct {
			Context int  `json:"context"`
			Input   *int `json:"input"`
			Output  *int `json:"output"`
		}{Context: 1000000}}
	}
	for _, tc := range []struct {
		id    string
		m     modelMetadata
		small bool
	}{
		{"gpt-6-luna", priced(0.10, 0.50), true},    // blend 0.20
		{"gpt-5.4-mini", priced(0.75, 4.50), false}, // blend 1.69, though named mini
		{"some-mini", modelMetadata{Name: "M"}, true},
		{"some-pro", modelMetadata{Name: "M"}, false},
		{"free-mini", priced(0, 0), true}, // no positive price: the name decides
	} {
		if got := metadataCandidate(tc.id, tc.m, true, rankingNow()).small; got != tc.small {
			t.Errorf("%s: small %t, want %t", tc.id, got, tc.small)
		}
	}
	for _, tc := range []struct {
		name, prompt, completion string
		small                    bool
	}{
		{"cheap-pro", "0.0000001", "0.0000005", true}, // blend 0.20
		{"dear-mini", "0.00000075", "0.0000045", false},
		{"unpriced-mini", "-1", "-1", true},
	} {
		var e kiloCatalogEntry
		e.ID, e.Name = "vendor/"+tc.name, tc.name
		e.Pricing.Prompt, e.Pricing.Completion = tc.prompt, tc.completion
		if got := kiloCandidate(e, rankingNow()).small; got != tc.small {
			t.Errorf("kilo %s: small %t, want %t", tc.name, got, tc.small)
		}
	}
}

// TestUnusableIneligible_0041: a model that takes no tools, or writes no
// text, is never in the band, but stays in the listing's usable ids.
func TestUnusableIneligible_0041(t *testing.T) {
	ok := modelMetadata{Name: "M", Reasoning: ptrBool(true), Cost: &modelCost{Input: 1, Output: 5}, ToolCall: ptrBool(true)}
	ok.Limit.Context = 1000000
	noTools := ok
	noTools.ToolCall = ptrBool(false)
	image := ok
	image.Modalities = &struct {
		Input  []string `json:"input"`
		Output []string `json:"output"`
	}{Input: []string{"text"}, Output: []string{"image"}}
	now := rankingNow()
	if !metadataCandidate("ok", ok, true, now).eligible(llmprovider.ProviderTogether) {
		t.Fatal("a usable model is not eligible")
	}
	for name, m := range map[string]modelMetadata{"no tools": noTools, "image output": image} {
		if metadataCandidate(name, m, true, now).eligible(llmprovider.ProviderTogether) {
			t.Errorf("%s: eligible", name)
		}
	}
	// In a ranking, the two stay in usable but never in the band.
	doc := `{"togetherai":{"models":{` +
		`"a/one":{"name":"1","reasoning":true,"tool_call":true,"cost":{"input":1,"output":5},"limit":{"context":1000000}},` +
		`"a/two":{"name":"2","reasoning":true,"tool_call":true,"cost":{"input":2,"output":6},"limit":{"context":1000000}},` +
		`"a/notools":{"name":"N","reasoning":true,"tool_call":false,"cost":{"input":9,"output":9},"limit":{"context":1000000}},` +
		`"a/image":{"name":"I","reasoning":true,"tool_call":true,"modalities":{"output":["image"]},"cost":{"input":9,"output":9},"limit":{"context":1000000}}}}}`
	meta, err := ParseMetadata(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	usable := []string{"a/one", "a/two", "a/notools", "a/image"}
	band := rankRecommended(ProfileCapable, llmprovider.ProviderTogether, candidatesFor(meta, llmprovider.ProviderTogether, usable),
		nil, MaxListed)
	if slices.Contains(band, "a/notools") || slices.Contains(band, "a/image") || len(band) != 2 {
		t.Errorf("band %v; want the two usable models only", band)
	}
}

// candidatesFor is metadataCurate's candidates for usable on provider's section.
func candidatesFor(meta Metadata, provider llmprovider.ProviderID, usable []string) []rankCandidate {
	models, _ := meta.doc.models(modelMetadataKey(provider))
	out := make([]rankCandidate, 0, len(usable))
	for _, id := range usable {
		m, covered := models[id]
		out = append(out, metadataCandidate(id, m, covered, rankingNow()))
	}
	return out
}

// sectionSix ranks provider's whole models.dev section, its usable ids, as
// the first-party providers now rank (D4), on the committed snapshot.
func sectionSix(t *testing.T, provider llmprovider.ProviderID, profile Profile) ([]string, map[string]modelMetadata) {
	t.Helper()
	doc := evidence0041Doc(t)
	usable := map[llmprovider.ProviderID]func(string) bool{
		llmprovider.ProviderOpenAI: isUsableOpenAIChatModel, llmprovider.ProviderClaude: isUsableClaudeTextModel,
		llmprovider.ProviderGemini: func(id string) bool { return isUsableGeminiTextModel(id, []string{"generateContent"}) },
		llmprovider.ProviderGrok:   isUsableGrokModel,
	}[provider]
	models, _ := doc.models(modelMetadataKey(provider))
	var ids []string
	for id := range models {
		if usable(id) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	meta := func() modelMetadataResult { return modelMetadataResult{doc: doc} }
	return metadataCurate(provider, profile, defaultWorkload, meta, firstPartyCurator(provider))(ids, MaxListed), models
}

// TestOneVendorGroups_0041: OpenAI's band holds at most two of each family,
// and no o3-pro.
func TestOneVendorGroups_0041(t *testing.T) {
	for _, profile := range []Profile{ProfileUtility, ProfileCapable} {
		six, models := sectionSix(t, llmprovider.ProviderOpenAI, profile)
		perFamily := map[string]int{}
		for _, id := range six {
			perFamily[models[id].Family]++
		}
		for family, n := range perFamily {
			if n > maxPerRankGroup {
				t.Errorf("profile %d: %d of family %q in %v", profile, n, family, six)
			}
		}
		if slices.Contains(six, "o3-pro") {
			t.Errorf("profile %d: o3-pro in %v", profile, six)
		}
	}
}

// TestFirstPartyRankByFacts_0041: on their sections, the four first-party
// providers' sixes are D4's table; with metadata off, a listing curates as
// today's name curator does.
func TestFirstPartyRankByFacts_0041(t *testing.T) {
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		profile  Profile
		want     []string
	}{
		{llmprovider.ProviderOpenAI, ProfileUtility,
			[]string{"gpt-6-luna", "gpt-5.6-luna", "gpt-5.4-nano", "gpt-5-nano", "gpt-5-mini", "gpt-6.1-sol"}},
		{llmprovider.ProviderOpenAI, ProfileCapable,
			[]string{"gpt-6.1-sol", "gpt-6-sol", "gpt-6-astra", "gpt-5.6-terra", "gpt-5.5-pro", "gpt-5.5"}},
	} {
		if six, _ := sectionSix(t, tc.provider, tc.profile); !slices.Equal(six, tc.want) {
			t.Errorf("%s profile %d:\n got  %v\n want %v", tc.provider, tc.profile, six, tc.want)
		}
	}
	// D4's other rows: what each keeps of today's static list, and adds.
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		profile  Profile
		kept     int
		adds     []string
	}{
		{llmprovider.ProviderClaude, ProfileUtility, 4, []string{"claude-haiku-4-5", "claude-opus-5"}},
		{llmprovider.ProviderClaude, ProfileCapable, 3, []string{"claude-fable-5-1", "claude-opus-5", "claude-fable-5"}},
		{llmprovider.ProviderGemini, ProfileUtility, 3, nil},
		{llmprovider.ProviderGemini, ProfileCapable, 3, nil},
		{llmprovider.ProviderGrok, ProfileUtility, 5, nil},
		{llmprovider.ProviderGrok, ProfileCapable, 5, nil},
	} {
		six, _ := sectionSix(t, tc.provider, tc.profile)
		kept := 0
		for _, id := range Static(tc.provider) {
			if slices.Contains(six, id) {
				kept++
			}
		}
		if kept != tc.kept || len(six) != MaxListed {
			t.Errorf("%s profile %d: %v keeps %d of the static %v; want %d, and six", tc.provider, tc.profile, six, kept,
				Static(tc.provider), tc.kept)
		}
		for _, id := range tc.adds {
			if !slices.Contains(six, id) {
				t.Errorf("%s profile %d: %v lacks %s", tc.provider, tc.profile, six, id)
			}
		}
	}
	// Metadata off: each recorded first-party listing curates as its name
	// curator always did.
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		file     string
	}{
		{llmprovider.ProviderOpenAI, "openai.json"}, {llmprovider.ProviderClaude, "claude.json"},
		{llmprovider.ProviderGemini, "gemini.json"}, {llmprovider.ProviderGrok, "grok.json"},
	} {
		resetLookupCaches(t)
		s := newLookupServer(t, `{}`, 200, firstPartyPage(t, tc.file))
		cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken("k"),
			llmprovider.WithBaseURL(s.URL), llmprovider.WithoutModelMetadata(), WithLimit(MaxListed))
		if err != nil || cat.Err != nil {
			t.Fatalf("%s: %v, %v", tc.provider, err, cat.Err)
		}
		if want := firstPartyCurator(tc.provider)(slices.Clone(cat.Usable), MaxListed); !slices.Equal(cat.Recommended, want) {
			t.Errorf("%s with metadata off:\n got  %v\n want the name curator's %v", tc.provider, cat.Recommended, want)
		}
		// Today's static order leads; the curator fills the rest.
		if static := Static(tc.provider); len(cat.Recommended) < len(static) ||
			!slices.Equal(cat.Recommended[:len(static)], static) {
			t.Errorf("%s with metadata off: %v does not start with the static %v", tc.provider, cat.Recommended, static)
		}
	}
}

// TestFirstPartyListRanksByFacts_0041: List, on each recorded first-party
// listing with the committed document, ranks by facts, not by name.
func TestFirstPartyListRanksByFacts_0041(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(evidence0041Dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc := evidence0041Doc(t)
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		file     string
	}{
		{llmprovider.ProviderOpenAI, "openai.json"}, {llmprovider.ProviderClaude, "claude.json"},
		{llmprovider.ProviderGemini, "gemini.json"}, {llmprovider.ProviderGrok, "grok.json"},
	} {
		resetLookupCaches(t)
		s := newLookupServer(t, string(raw), 200, firstPartyPage(t, tc.file))
		cat, err := List(context.Background(), tc.provider, llmprovider.NewStaticToken("k"),
			llmprovider.WithBaseURL(s.URL), llmprovider.WithModelMetadataURL(s.URL+"/api.json"), WithLimit(MaxListed))
		if err != nil || cat.Err != nil {
			t.Fatalf("%s: %v, %v", tc.provider, err, cat.Err)
		}
		meta := func() modelMetadataResult { return modelMetadataResult{doc: doc} }
		want := metadataCurate(tc.provider, ProfileUtility, defaultWorkload, meta, firstPartyCurator(tc.provider))(slices.Clone(cat.Usable), MaxListed)
		if !slices.Equal(cat.Recommended, want) {
			t.Errorf("%s:\n got  %v\n want the ranking by facts %v", tc.provider, cat.Recommended, want)
		}
		if byName := firstPartyCurator(tc.provider)(slices.Clone(cat.Usable), MaxListed); slices.Equal(want, byName) {
			t.Errorf("%s: the ranking by facts %v is the name curator's; the case tells nothing", tc.provider, want)
		}
	}
}
