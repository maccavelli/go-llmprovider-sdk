package wizard

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0066-PLAN Phase 6 (V11, V12, V14): an unknown Options.Providers id is
// named, the guide's recommended ids, and the two paths no test ran.

// TestProviders_UnknownNamed_0066: the MADR's probe H. Each id the registry
// lacks is named once, and the menu offers the rest; when it lacks every
// one, the error is as before, with no notice.
func TestProviders_UnknownNamed_0066(t *testing.T) {
	f := newFake(t, fakePrompter{})
	_, err := ConfigureLLM(context.Background(), f, Options{Providers: []llmprovider.ProviderID{
		llmprovider.ProviderClaude, "claud", "claud", "nonexistent"}})
	if err == nil || len(f.seenSelectItems) != 1 || len(f.seenSelectItems[0]) != 1 {
		t.Fatalf("err %v, menus %v; want the run to end at a menu of one", err, f.seenSelectItems)
	}
	want := []string{`no provider "claud" here; it is left out of the menu`,
		`no provider "nonexistent" here; it is left out of the menu`}
	if !slices.Equal(f.seenNotify, want) {
		t.Errorf("notices %q; want %q", f.seenNotify, want)
	}

	g := newFake(t, fakePrompter{})
	_, err = ConfigureLLM(context.Background(), g, Options{Providers: []llmprovider.ProviderID{"nonexistent"}})
	if err == nil || err.Error() != "wizard: none of the requested providers exist: [nonexistent]" || len(g.seenNotify) != 0 {
		t.Errorf("err %v, notices %q; want the error as before, and no notice", err, g.seenNotify)
	}
}

// TestGuide_EveryListedRecommended_0066: the guide's sentence (V12). With
// ModelLimit zero, a provider's own ListModels has every listed id
// recommended, in its order.
func TestGuide_EveryListedRecommended_0066(t *testing.T) {
	var built []*thirdParty
	models := listedIDs(catalog.MaxListed + 4)
	f := newFake(t, fakePrompter{secrets: []string{testKey}})
	_, cat, err := ResolveCredential(context.Background(), f, Options{
		Registry: thirdPartyRegistry(t, thirdParty{models: models}, true, &built), Discover: true}, thirdPartyID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cat.Recommended, models) {
		t.Errorf("Recommended %q; want every listed id, in order: %q", cat.Recommended, models)
	}
}

// TestFallbacks_CapMatches_0066: V14. A fallback search with 25 matches
// shows the first 20, and says so.
func TestFallbacks_CapMatches_0066(t *testing.T) {
	const primary = "primary-model"
	usable := []string{primary}
	for i := range 25 {
		usable = append(usable, fmt.Sprintf("fallback-model-%02d", i))
	}
	d, _ := (Options{}).registry().Descriptor(llmprovider.ProviderClaude)
	cat := catalog.Catalog{Recommended: []string{primary}, Usable: usable}
	f := newFake(t, fakePrompter{inputs: []string{"fallback-model"}, multiSelects: [][]int{{}}, confirms: []bool{false}})
	if _, err := selectFallbacks(context.Background(), f, d, cat, primary, nil); err != nil {
		t.Fatal(err)
	}
	if want := []string{"showing 20 of 25 matches; refine the search to narrow them"}; !slices.Equal(f.seenNotify, want) {
		t.Errorf("notices %q; want %q", f.seenNotify, want)
	}
	if len(f.seenMultiSelectItems) != 1 || len(f.seenMultiSelectItems[0]) != 20 {
		t.Errorf("menus %v; want one of 20", f.seenMultiSelectItems)
	}
}

// errStore0066 is the failing store's error.
var errStore0066 = errors.New("store: disk full")

// failingStore0066 holds nothing and fails every Save.
type failingStore0066 struct{}

func (failingStore0066) Load(context.Context, llmprovider.ProviderID) (*auth.OAuthSession, error) {
	return nil, nil
}

func (failingStore0066) Save(context.Context, llmprovider.ProviderID, *auth.OAuthSession) error {
	return errStore0066
}

func (failingStore0066) Delete(context.Context, llmprovider.ProviderID) error { return nil }

// TestSave_StoreFails_0066: V14. A store whose Save fails ends a Grok
// browser sign-in and a Kilo device sign-in with an error that names the
// save and wraps the store's.
func TestSave_StoreFails_0066(t *testing.T) {
	t.Run("Grok", func(t *testing.T) {
		client := client0059(200, 200, nil)
		browserGrok0059(t, client, false)
		_, err := grok0059(t, failingStore0066{}, client, false, 0)
		if !errors.Is(err, errStore0066) || err.Error() != "wizard: save OAuth session: store: disk full" {
			t.Errorf("err = %v; want the save named, wrapping the store's error", err)
		}
	})
	t.Run("Kilo", func(t *testing.T) {
		stubKilo(t, auth.KiloAccount{}, nil)
		o := Options{TokenStore: failingStore0066{}}
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault},
			selects: []int{methodIdx0059(t, o, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode)}})
		_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
		if !errors.Is(err, errStore0066) || err.Error() != "wizard: save Kilo login: store: disk full" {
			t.Errorf("err = %v; want the save named, wrapping the store's error", err)
		}
	})
}
