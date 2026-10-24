package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0059-MADR W6 and W7: a saved session that cannot be used, and a saved
// fallback the menu does not recommend.

// savedOpenAI0059 puts an OpenAI session in store, refresh as given, its
// access expired an hour ago when expired is set, and names it in Existing.
func savedOpenAI0059(t *testing.T, store *memoryTokenStore, refresh string, expired bool) Options {
	t.Helper()
	expiry := time.Now().Add(time.Hour)
	if expired {
		expiry = time.Now().Add(-time.Hour)
	}
	o := Options{TokenStore: store}
	o.Existing = storedExisting(t, store, Result{Provider: llmprovider.ProviderOpenAI, Kind: CredOAuth,
		TokenExpiry: expiry, Issuer: auth.DefaultOpenAIIssuer, ClientID: auth.DefaultOpenAIClientID},
		"saved-access-0059", refresh)
	return o
}

// methodIdx0059 is method's place in the menu o offers for id.
func methodIdx0059(t *testing.T, o Options, id llmprovider.ProviderID, method llmprovider.AuthMethodID) int {
	t.Helper()
	d, ok := o.registry().Descriptor(id)
	if !ok {
		t.Fatalf("no descriptor for %q", id)
	}
	i := slices.IndexFunc(offeredAuthMethods(d, o), func(m llmprovider.AuthMethod) bool { return m.ID == method })
	if i < 0 {
		t.Fatalf("%q does not offer %q", id, method)
	}
	return i
}

// TestKept_UnusableNotOffered_0059 (P7): a saved session that cannot be used
// is not offered; a notice gives the check's reason, and the method menu
// follows, where a browser sign-in completes the run.
func TestKept_UnusableNotOffered_0059(t *testing.T) {
	store := newMemoryTokenStore()
	o := savedOpenAI0059(t, store, "", true)
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		return testOAuthSession(provider), nil
	})
	f := newFake(t, fakePrompter{selects: []int{methodIdx0059(t, o, llmprovider.ProviderOpenAI, llmprovider.AuthBrowserOAuth)}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderOpenAI)
	if err != nil || res.Kind != CredOAuth || store.saves != 2 {
		t.Fatalf("ResolveCredential = %q, %v with %d saves; want the new session saved after the setup's",
			res.Kind, err, store.saves)
	}
	if len(f.seenConfirm) != 0 || countContaining(f.seenNotify, "the saved OpenAI session cannot be used") != 1 ||
		countContaining(f.seenNotify, "no refresh token") != 1 ||
		!slices.Contains(f.seenSelect, "Choose how to authenticate with OpenAI:") {
		t.Errorf("confirms %q, notices %q, selects %q; want no confirm, one notice with the reason, and the method menu",
			f.seenConfirm, f.seenNotify, f.seenSelect)
	}
	for _, n := range f.seenNotify {
		if strings.Contains(n, "saved-access-0059") {
			t.Errorf("a notice shows the token: %q", n)
		}
	}
}

// TestKept_UsableStillOffered_0059: a saved session that can be used is
// offered, and kept, with no menu.
func TestKept_UsableStillOffered_0059(t *testing.T) {
	store := newMemoryTokenStore()
	o := savedOpenAI0059(t, store, "saved-refresh-0059", false)
	f := newFake(t, fakePrompter{confirms: []bool{true}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderOpenAI)
	if err != nil || res.Kind != CredOAuth || store.saves != 1 {
		t.Fatalf("ResolveCredential = %q, %v with %d saves; want the saved session kept", res.Kind, err, store.saves)
	}
	if len(f.seenConfirm) != 1 || !strings.Contains(f.seenConfirm[0], "Keep the existing session") ||
		len(f.seenSelect) != 0 || len(f.seenNotify) != 0 {
		t.Errorf("confirms %q, selects %q, notices %q; want the keep question alone", f.seenConfirm, f.seenSelect, f.seenNotify)
	}
}

// blankMulti0059 answers each MultiSelect with its preselection, as Enter
// does, and records it; the rest is fakePrompter's.
type blankMulti0059 struct {
	*fakePrompter
	preselected [][]int
}

func (b *blankMulti0059) MultiSelect(_ string, choices []Choice, preselected []int) ([]int, error) {
	b.seenMultiSelectItems = append(b.seenMultiSelectItems, choices)
	b.preselected = append(b.preselected, preselected)
	return preselected, nil
}

// TestFallbacks_SavedOffered_0059 (P5): a saved fallback that is not
// recommended is offered in the blank round, marked saved and preselected,
// and a blank answer keeps it.
func TestFallbacks_SavedOffered_0059(t *testing.T) {
	const legacy = "claude-legacy-fallback"
	b := &blankMulti0059{fakePrompter: newFake(t, fakePrompter{blankSearches: true,
		selects: []int{providerIdx(t, llmprovider.ProviderClaude), 0}, confirms: []bool{true}})}
	res, err := ConfigureLLM(context.Background(), b, Options{NeedFallbacks: true,
		Existing: Result{Provider: llmprovider.ProviderClaude, APIKey: "existing-key-abcd", Fallbacks: []string{legacy}}})
	if err != nil || !slices.Contains(res.Fallbacks, legacy) {
		t.Fatalf("ConfigureLLM: fallbacks %q, %v; want %s kept", res.Fallbacks, err, legacy)
	}
	if len(b.seenMultiSelectItems) != 1 {
		t.Fatalf("%d fallback rounds; want one", len(b.seenMultiSelectItems))
	}
	items := b.seenMultiSelectItems[0]
	at := slices.IndexFunc(items, func(c Choice) bool { return c.Detail == savedFallbackDetail })
	if at < 0 || at != len(items)-1 || !slices.Equal(b.preselected[0], []int{at}) {
		t.Errorf("choices %+v, preselected %v; want the saved row last, marked %q, and preselected",
			items, b.preselected[0], savedFallbackDetail)
	}
}

// TestFallbacks_SavedOnlyRound_0059: with no recommended row left, a saved
// fallback still remaining keeps the blank round.
func TestFallbacks_SavedOnlyRound_0059(t *testing.T) {
	const legacy = "claude-legacy-fallback"
	d, _ := (Options{}).registry().Descriptor(llmprovider.ProviderClaude)
	cat := catalog.Catalog{Recommended: []string{"claude-primary"}, Usable: []string{"claude-primary", "claude-other"}}
	b := &blankMulti0059{fakePrompter: newFake(t, fakePrompter{blankSearches: true})}
	got, err := selectFallbacks(context.Background(), b, d, cat, "claude-primary", []string{legacy})
	if err != nil || !slices.Equal(got, []string{legacy}) || len(b.seenMultiSelectItems) != 1 {
		t.Errorf("selectFallbacks = %q, %v after %d rounds; want the saved fallback, from one round",
			got, err, len(b.seenMultiSelectItems))
	}
}
