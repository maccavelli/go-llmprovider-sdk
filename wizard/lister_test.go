package wizard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0015-PLAN S12b: a provider catalog does not know is listed through its own
// ListModels (0015-MADR, amendment "the wizard lists a provider through its
// own ListModels").

const thirdPartyID llmprovider.ProviderID = "acme-test"

var thirdPartyStatic = []string{"static-a", "static-b"}

// thirdParty is a provider catalog does not know. It records what the
// wizard built it with.
type thirdParty struct {
	models   []string
	err      error
	settings *llmprovider.Settings
}

func (p *thirdParty) ID() llmprovider.ProviderID { return thirdPartyID }

func (p *thirdParty) Capabilities() llmprovider.Capabilities { return llmprovider.Capabilities{} }

func (p *thirdParty) Generate(context.Context, *llmprovider.Request) (*llmprovider.Response, error) {
	return nil, errors.New("thirdParty: Generate is not part of these tests")
}

// listingThirdParty is a thirdParty that is a ModelLister.
type listingThirdParty struct{ *thirdParty }

func (p listingThirdParty) ListModels(context.Context) ([]string, error) {
	return slices.Clone(p.models), p.err
}

// thirdPartyRegistry holds one provider built from base; lister says whether
// it is a ModelLister. built receives every provider the wizard builds.
func thirdPartyRegistry(t *testing.T, base thirdParty, lister bool, built *[]*thirdParty) *llmprovider.Registry {
	t.Helper()
	r := llmprovider.NewRegistry()
	d := llmprovider.Descriptor{ID: thirdPartyID, Label: "Acme", RequiresAPIKey: true, StaticModels: thirdPartyStatic}
	err := r.Register(d, func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
		st, err := llmprovider.ResolveOptions(thirdPartyID, opts)
		if err != nil {
			return nil, err
		}
		p := base
		p.settings = st
		*built = append(*built, &p)
		if lister {
			return listingThirdParty{&p}, nil
		}
		return &p, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func listedIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("acme-model-%d", i+1)
	}
	return ids
}

func noticeContaining(notes []string, s string) bool {
	return slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, s) })
}

// TestConfigureLLM_ThirdPartyListsThroughItsLister: the listing is offered,
// its first catalog.MaxListed ids recommended, and the provider is built
// with the run's credential, client and ProviderOptions.
func TestConfigureLLM_ThirdPartyListsThroughItsLister(t *testing.T) {
	var built []*thirdParty
	models := listedIDs(catalog.MaxListed + 2)
	client := &http.Client{}
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{0, 1}, secrets: []string{testKey}}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Registry: thirdPartyRegistry(t, thirdParty{models: models}, true, &built), Discover: true,
		HTTPClient: client, ProviderOptions: []llmprovider.Option{llmprovider.WithModel("from-provider-options")},
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Model != models[1] {
		t.Errorf("Model = %q, want the second listed id %q", res.Model, models[1])
	}
	menu := f.seenSelectItems[len(f.seenSelectItems)-1]
	var labels []string
	for _, c := range menu {
		labels = append(labels, c.Label)
	}
	for i, m := range models {
		if got := slices.ContainsFunc(labels, func(l string) bool { return strings.Contains(l, m) }); got != (i < catalog.MaxListed) {
			t.Errorf("menu %v: %q shown = %v, want the first %d listed ids only", labels, m, got, catalog.MaxListed)
		}
	}
	if noticeContaining(f.seenNotify, "could not list") || noticeContaining(f.seenNotify, "unavailable") {
		t.Errorf("notices = %v, want no listing warning", f.seenNotify)
	}
	if len(built) != 1 {
		t.Fatalf("the wizard built %d providers, want 1", len(built))
	}
	st := built[0].settings
	if tok, err := st.TokenSource().Token(context.Background()); err != nil || tok.Value != testKey {
		t.Errorf("the provider's credential = %q, %v; want the key entered", tok.Value, err)
	}
	if st.HTTPClient() != client {
		t.Error("the provider was not built with Options.HTTPClient")
	}
	if st.Model() != "from-provider-options" {
		t.Errorf("Model setting = %q, want Options.ProviderOptions applied", st.Model())
	}
}

// TestConfigureLLM_ThirdPartySearchCoversTheWholeListing: an id past the
// recommended ones is found by search.
func TestConfigureLLM_ThirdPartySearchCoversTheWholeListing(t *testing.T) {
	var built []*thirdParty
	models := listedIDs(catalog.MaxListed + 2)
	last := models[len(models)-1]
	f := &fakePrompter{t: t, selects: []int{0, 0}, secrets: []string{testKey}, inputs: []string{last}}
	res, err := ConfigureLLM(context.Background(), f, Options{
		Registry: thirdPartyRegistry(t, thirdParty{models: models}, true, &built), Discover: true,
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.Model != last {
		t.Errorf("Model = %q, want %q, found by search", res.Model, last)
	}
}

// TestConfigureLLM_ThirdPartyFallsBackToStatic: no lister, a failing one, or
// an empty listing gives StaticModels, with a warning.
func TestConfigureLLM_ThirdPartyFallsBackToStatic(t *testing.T) {
	cases := []struct {
		name   string
		base   thirdParty
		lister bool
	}{
		{"not a ModelLister", thirdParty{models: listedIDs(3)}, false},
		{"ListModels fails", thirdParty{err: errors.New("acme listing down")}, true},
		{"an empty listing", thirdParty{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var built []*thirdParty
			f := &fakePrompter{t: t, blankSearches: true, selects: []int{0, 1}, secrets: []string{testKey}}
			res, err := ConfigureLLM(context.Background(), f, Options{
				Registry: thirdPartyRegistry(t, tc.base, tc.lister, &built), Discover: true,
			})
			if err != nil {
				t.Fatalf("ConfigureLLM: %v", err)
			}
			if res.Model != thirdPartyStatic[1] {
				t.Errorf("Model = %q, want the static %q", res.Model, thirdPartyStatic[1])
			}
			if !noticeContaining(f.seenNotify, "could not list models for Acme") {
				t.Errorf("notices = %v, want the listing warning", f.seenNotify)
			}
		})
	}
}
