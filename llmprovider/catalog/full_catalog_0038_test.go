package catalog

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0038-MADR D1, D2 and its amendment A1: WithLimit(n) cuts the recommended
// list; with n zero or less it is the full curated catalog, today's six
// first. Since 0038-PLAN Phase 3, no option is the full list.

// vendorIDs is an OpenAI listing longer than six, every id usable.
var vendorIDs = append(slices.Clone(staticOpenAI), "gpt-5.6-sol", "gpt-6-sol", "gpt-4.1")

// ollamaNames is an Ollama instance with more models installed than six.
var ollamaNames = []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8"}

// snapshotFull is each ranked list's full length in the 2026-10-12 snapshot,
// by profile: Usable less D1's exclusions, as 0038-PLAN Phase 2 measured
// them before the change.
var snapshotFull = map[llmprovider.ProviderID][2]int{
	llmprovider.ProviderHuggingFace: {102, 119}, llmprovider.ProviderKilo: {272, 285},
	llmprovider.ProviderOpencodeGo: {28, 28}, llmprovider.ProviderOpencodeZen: {77, 77},
	llmprovider.ProviderTogether: {19, 21},
}

// listVendor lists vendorIDs from OpenAI, with opts.
func listVendor(t *testing.T, opts ...llmprovider.Option) Catalog {
	t.Helper()
	srv := serveBody(t, openAIStyleListing(vendorIDs...))
	return listCatalog(context.Background(), t, llmprovider.ProviderOpenAI, append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL)}, opts...)...)
}

// listOllama lists ollamaNames, with opts.
func listOllama(t *testing.T, opts ...llmprovider.Option) Catalog {
	t.Helper()
	srv := serveBody(t, ollamaTags(ollamaNames...))
	return listCatalog(context.Background(), t, llmprovider.ProviderOllama, append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL)}, opts...)...)
}

// expectedFull is the set D1 says the full list holds: usable, less the ids
// measured not to answer, the profile's exclusions, and on Together the ids
// models.dev does not list. It reads the snapshot's document itself, apart
// from the ranking.
func expectedFull(t *testing.T, dir string, provider llmprovider.ProviderID, profile Profile, usable []string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := decodeModelMetadata(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	known, _ := doc.models(modelMetadataKey(provider))
	var out []string
	for _, id := range usable {
		_, listed := known[id]
		if !excludedAtRequest(provider, id) && !utilityExcluded(profile, provider, id) && (!listedOnly(provider) || listed) {
			out = append(out, id)
		}
	}
	return out
}

// sameSet reports whether a and b hold the same ids, a with no id twice.
func sameSet(a, b []string) bool {
	sa, sb := slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b))
	return len(slices.Compact(slices.Clone(sa))) == len(sa) && slices.Equal(sa, sb)
}

// TestWithLimit_SixIsToday: WithLimit(MaxListed) recommends today's six, the
// first six of the default list, on the open catalogs, a vendor and Ollama.
// The open catalogs' sixes themselves are pinned by
// TestListModelCatalog_Snapshot20261012.
func TestWithLimit_SixIsToday(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for provider := range snapshotFile {
		for _, p := range []Profile{ProfileUtility, ProfileCapable} {
			full := replaySnapshot(t, dir, provider, p).Recommended
			six := full[:min(len(full), MaxListed)]
			if got := replaySnapshot(t, dir, provider, p, WithLimit(MaxListed)).Recommended; !slices.Equal(got, six) {
				t.Errorf("%s profile %d: WithLimit(MaxListed) = %v; want %v", provider, p, got, six)
			}
		}
	}
	// A vendor's six is the first six of its default list, ranked by facts
	// since 0041-MADR D4.
	vendor := listVendor(t).Recommended
	if got := listVendor(t, WithLimit(MaxListed)).Recommended; !slices.Equal(got, vendor[:min(len(vendor), MaxListed)]) {
		t.Errorf("openai: WithLimit(MaxListed) = %v; want the default list's first six %v", got,
			vendor[:min(len(vendor), MaxListed)])
	}
	if got := listOllama(t, WithLimit(MaxListed)).Recommended; !slices.Equal(got, ollamaNames[:MaxListed]) {
		t.Errorf("ollama: WithLimit(MaxListed) = %v; want %v", got, ollamaNames[:MaxListed])
	}
}

// TestList_DefaultIsFullList: with no option, a listing recommends the full
// list, as WithLimit(0) does, on the open catalogs, a vendor and Ollama
// (0038-MADR D1).
func TestList_DefaultIsFullList(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for provider := range snapshotFile {
		for _, p := range []Profile{ProfileUtility, ProfileCapable} {
			want := replaySnapshot(t, dir, provider, p, WithLimit(0)).Recommended
			if got := replaySnapshot(t, dir, provider, p).Recommended; !slices.Equal(got, want) {
				t.Errorf("%s profile %d: no option recommends %d ids; want the full list's %d", provider, p, len(got), len(want))
			}
		}
	}
	if got, want := listVendor(t).Recommended, listVendor(t, WithLimit(0)).Recommended; !slices.Equal(got, want) {
		t.Errorf("openai: no option recommends %v; want the full list %v", got, want)
	}
	if got := listOllama(t).Recommended; !slices.Equal(got, ollamaNames) {
		t.Errorf("ollama: no option recommends %v; want every installed model %v", got, ollamaNames)
	}
}

// TestWithLimit_FullList: WithLimit(0) is today's six, then the rest of the
// curated list: every usable id less D1's exclusions, once each.
func TestWithLimit_FullList(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for provider, lengths := range snapshotFull {
		for _, p := range []Profile{ProfileUtility, ProfileCapable} {
			six := replaySnapshot(t, dir, provider, p, WithLimit(MaxListed)).Recommended
			cat := replaySnapshot(t, dir, provider, p, WithLimit(0))
			full := cat.Recommended
			if len(full) < len(six) || !slices.Equal(full[:len(six)], six) {
				t.Errorf("%s profile %d: the full list does not start with the six %v: %v", provider, p, six, full[:min(len(full), MaxListed)])
			}
			if want := expectedFull(t, dir, provider, p, cat.Usable); !sameSet(full, want) {
				t.Errorf("%s profile %d: the full list holds %d ids; want the %d usable ids less D1's exclusions", provider, p, len(full), len(want))
			}
			if len(full) != lengths[p] {
				t.Errorf("%s profile %d: %d ids; want %d, as measured", provider, p, len(full), lengths[p])
			}
		}
	}
	vendor := listVendor(t, WithLimit(0))
	if six := listVendor(t, WithLimit(MaxListed)).Recommended; !slices.Equal(vendor.Recommended[:min(len(vendor.Recommended), len(six))], six) ||
		!sameSet(vendor.Recommended, vendor.Usable) {
		t.Errorf("openai: the full list %v; want the six %v first, then the rest of Usable %v", vendor.Recommended, six, vendor.Usable)
	}
	if got := listOllama(t, WithLimit(0)).Recommended; !slices.Equal(got, ollamaNames) {
		t.Errorf("ollama: the full list %v; want every installed model %v", got, ollamaNames)
	}
}

// TestWithLimit_Cuts: WithLimit(n) is the full list's first n, beyond six as
// within it, and a negative n is no limit.
func TestWithLimit_Cuts(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	full := replaySnapshot(t, dir, llmprovider.ProviderKilo, ProfileUtility, WithLimit(0)).Recommended
	if len(full) <= 9 {
		t.Fatalf("WithLimit(0) holds %d ids; want Kilo's full list, longer than nine", len(full))
	}
	for _, n := range []int{3, 9} {
		if got := replaySnapshot(t, dir, llmprovider.ProviderKilo, ProfileUtility, WithLimit(n)).Recommended; !slices.Equal(got, full[:n]) {
			t.Errorf("WithLimit(%d) = %v; want the full list's first %d %v", n, got, n, full[:n])
		}
	}
	if got := replaySnapshot(t, dir, llmprovider.ProviderKilo, ProfileUtility, WithLimit(-1)).Recommended; !slices.Equal(got, full) {
		t.Errorf("WithLimit(-1) holds %d ids; want the full list's %d", len(got), len(full))
	}
}

// TestWithLimit_FallbackExcludes: with no metadata, the fallback curation
// recommends no id measured not to answer, cut or not (D1).
func TestWithLimit_FallbackExcludes(t *testing.T) {
	together := serveBody(t, `[{"id":"MiniMaxAI/MiniMax-M2.7","type":"chat"},{"id":"t/a","type":"chat"},{"id":"t/b","type":"chat"}]`)
	goGateway := serveBody(t, openAIStyleListing(opencodeDeepSeekV4Flash, "glm-5.3-flash", "qwen3.8-flash"))
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		url, id  string
	}{
		{llmprovider.ProviderTogether, together.URL, "MiniMaxAI/MiniMax-M2.7"},
		{llmprovider.ProviderOpencodeGo, goGateway.URL, opencodeDeepSeekV4Flash},
	} {
		for _, opts := range [][]llmprovider.Option{nil, {WithLimit(0)}} {
			cat := listCatalog(context.Background(), t, tc.provider, append([]llmprovider.Option{llmprovider.WithBaseURL(tc.url)}, opts...)...)
			if slices.Contains(cat.Recommended, tc.id) || !slices.Contains(cat.Usable, tc.id) {
				t.Errorf("%s (%d options): Recommended %v, Usable %v; want %s searchable but not recommended",
					tc.provider, len(opts), cat.Recommended, cat.Usable, tc.id)
			}
		}
	}
}

// TestWithLimit_StaticFallback: a failed listing's static catalog is cut too,
// and Usable still equals Recommended (D3).
func TestWithLimit_StaticFallback(t *testing.T) {
	srv := httptestStatus(t, http.StatusInternalServerError)
	cat := listCatalog(context.Background(), t, llmprovider.ProviderOpenAI, llmprovider.WithBaseURL(srv), WithLimit(2))
	if cat.Live || !slices.Equal(cat.Recommended, staticOpenAI[:2]) || !slices.Equal(cat.Usable, cat.Recommended) {
		t.Errorf("Live %v, Recommended %v, Usable %v; want the static catalog's first two in both, not live",
			cat.Live, cat.Recommended, cat.Usable)
	}
}

// httptestStatus is a server that answers every request with status, and
// returns its URL.
func httptestStatus(t *testing.T, status int) string {
	t.Helper()
	srv, _ := metadataServer(t, status, "")
	return srv.URL
}
