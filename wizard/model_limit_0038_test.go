package wizard

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0038-MADR D5 and A2: the menu is the whole curated list by default, and
// Options.ModelLimit cuts it on every path: a built-in listing, a ChatGPT
// session, a third party's ListModels and the built-in catalog.

// zenMenu runs the configuration for OpenCode Zen over zenSearchIDs, with
// limit, and returns the ids its model menu recommends.
func zenMenu(t *testing.T, limit int) []string {
	t.Helper()
	srv := zenServer(t, http.StatusOK, zenListing(zenSearchIDs))
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpencodeZen), 0},
		secrets: []string{testKey}, inputs: []string{srv.URL, ""}})
	opts := zenOptions()
	opts.ModelLimit = limit
	if _, err := ConfigureLLM(context.Background(), f, opts); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	return shown(f.seenSelectItems[len(f.seenSelectItems)-1], llmprovider.ProviderOpencodeZen, zenSearchIDs)
}

// thirdPartyMenu runs the configuration for a third party listing models,
// with limit and providerOptions, and returns the ids its model menu
// recommends and the notices.
func thirdPartyMenu(t *testing.T, models []string, limit int, providerOptions ...llmprovider.Option) ([]string, []string) {
	t.Helper()
	var built []*thirdParty
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{0, 0}, secrets: []string{testKey}})
	if _, err := ConfigureLLM(context.Background(), f, Options{
		Registry: thirdPartyRegistry(t, thirdParty{models: models}, true, &built), Discover: true,
		ModelLimit: limit, ProviderOptions: providerOptions,
	}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	return shown(f.seenSelectItems[len(f.seenSelectItems)-1], thirdPartyID, append(slices.Clone(models), thirdPartyStatic...)), f.seenNotify
}

// staticMenu runs the configuration for Claude without discovery, with
// limit, and returns the ids its model menu recommends.
func staticMenu(t *testing.T, limit int) []string {
	t.Helper()
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		secrets: []string{testKey}})
	if _, err := ConfigureLLM(context.Background(), f, Options{ModelLimit: limit}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	static := catalog.Static(llmprovider.ProviderClaude)
	return shown(f.seenSelectItems[len(f.seenSelectItems)-1], llmprovider.ProviderClaude, static)
}

// shown is the ids among candidates whose label a menu shows, in the menu's
// order.
func shown(menu []Choice, provider llmprovider.ProviderID, candidates []string) []string {
	var out []string
	for _, c := range menu {
		for _, id := range candidates {
			if c.Label == catalog.Label(provider, id) || c.Label == id {
				out = append(out, id)
				break
			}
		}
	}
	return out
}

// chatGPTRecommended is chatGPTCatalog's recommended ids for a session that
// lists models, with limit.
func chatGPTRecommended(t *testing.T, models []string, limit int) []string {
	t.Helper()
	var built []*thirdParty
	o := Options{Registry: thirdPartyRegistry(t, thirdParty{models: models}, true, &built), ModelLimit: limit}
	cat, err := chatGPTCatalog(context.Background(), o, llmprovider.Descriptor{ID: thirdPartyID, Label: "Acme"},
		Result{}, llmprovider.NewStaticToken("k"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cat.Usable, models) {
		t.Errorf("Usable = %v; want every listed model %v", cat.Usable, models)
	}
	return cat.Recommended
}

// TestWizard_MenuIsFullListByDefault: with no ModelLimit, the menu holds
// every listed model on each listing path.
func TestWizard_MenuIsFullListByDefault(t *testing.T) {
	if got := zenMenu(t, 0); len(got) != len(zenSearchIDs) {
		t.Errorf("built-in listing: the menu shows %v; want all %d listed models", got, len(zenSearchIDs))
	}
	models := listedIDs(catalog.MaxListed + 2)
	if got, _ := thirdPartyMenu(t, models, 0); !slices.Equal(got, models) {
		t.Errorf("third party: the menu shows %v; want %v", got, models)
	}
	if got := chatGPTRecommended(t, listedIDs(14), 0); len(got) != 14 {
		t.Errorf("ChatGPT session: %d recommended; want all 14", len(got))
	}
}

// TestWizard_ModelLimit: ModelLimit cuts the menu on every path, the
// built-in catalog included.
func TestWizard_ModelLimit(t *testing.T) {
	models := listedIDs(catalog.MaxListed + 2)
	for _, limit := range []int{2, catalog.MaxListed} {
		if got := zenMenu(t, limit); len(got) != limit {
			t.Errorf("built-in listing, ModelLimit %d: the menu shows %v", limit, got)
		}
		if got, _ := thirdPartyMenu(t, models, limit); !slices.Equal(got, models[:limit]) {
			t.Errorf("third party, ModelLimit %d: the menu shows %v; want %v", limit, got, models[:limit])
		}
		if got := chatGPTRecommended(t, listedIDs(14), limit); len(got) != limit {
			t.Errorf("ChatGPT session, ModelLimit %d: %d recommended", limit, len(got))
		}
	}
	static := catalog.Static(llmprovider.ProviderClaude)
	if got := staticMenu(t, 2); !slices.Equal(got, static[:2]) {
		t.Errorf("built-in catalog, ModelLimit 2: the menu shows %v; want %v", got, static[:2])
	}
}

// TestWizard_ProviderOptionsLimitRefused documents A2: catalog.WithLimit in
// ProviderOptions reaches a third party's New, which refuses it, so the
// wizard warns and offers the static catalog rather than cutting silently.
func TestWizard_ProviderOptionsLimitRefused(t *testing.T) {
	got, notes := thirdPartyMenu(t, listedIDs(catalog.MaxListed+2), 0, catalog.WithLimit(2))
	if !noticeContaining(notes, "could not list") || !slices.Equal(got, thirdPartyStatic) {
		t.Errorf("menu %v, notices %v; want the static catalog %v after a listing warning", got, notes, thirdPartyStatic)
	}
}
