package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0080-MADR D4 (the owner's ruling of 2026-11-17, relayed by the coordinator:
// "Treat it as having an organisation: no profile read"): a credential whose
// listing is an organization's is checked by that listing alone, whether the
// organization comes from the caller, from a token's embedded URL or from the
// base URL's own path. The profile is not read for it, since a valid
// organization token must not be refused by a check the organization's listing
// makes redundant. The fake credentials are fixed values.

const (
	// kiloOrgBad0080 and kiloOrgGood0080 are tokens of the form "{backend URL}:{secret}" whose URL
	// names an organization; the listing is that organization's.
	kiloOrgBad0080  = "https://kilo.example.test/api/organizations/org-embedded-0080:fake-org-bad-0080"
	kiloOrgGood0080 = "https://kilo.example.test/api/organizations/org-embedded-0080:fake-org-good-0080"
)

// TestKiloVerify_EmbeddedOrganizationReadsNoProfile_0080: a typed token that
// embeds an organization's URL, with no organization entered, makes no profile
// request; the organization's listing refuses the first and accepts the second.
func TestKiloVerify_EmbeddedOrganizationReadsNoProfile_0080(t *testing.T) {
	k := newKilo0080(kiloOrgGood0080)
	d := descriptor0056(t, llmprovider.ProviderKilo)
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{kiloOrgBad0080, kiloOrgGood0080}})
	res, _, err := ResolveCredential(context.Background(), f, k.options(t), llmprovider.ProviderKilo)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	want := d.Label + " refused this key (authentication failed, HTTP 401); enter another"
	if n := countContaining(f.seenNotify, want); n != 1 || res.APIKey != kiloOrgGood0080 {
		t.Errorf("notices %q, key shown as the second %t; want %q once and the second token returned",
			f.seenNotify, res.APIKey == kiloOrgGood0080, want)
	}
	if reads := k.profileReads(); len(reads) != 0 {
		t.Errorf("%d profile reads; want none for a token that names an organization", len(reads))
	}
	if len(k.orgLists) != 2 || len(k.public) != 0 {
		t.Errorf("organization listings %d, public listings %d; want the organization's listing twice and no public one",
			len(k.orgLists), len(k.public))
	}
}

// TestKiloVerify_BaseURLOrganizationReadsNoProfile_0080: an endpoint whose own
// path is an organization's makes the listing the organization's, so a typed
// key is checked by it and the profile is not read.
func TestKiloVerify_BaseURLOrganizationReadsNoProfile_0080(t *testing.T) {
	k := newKilo0080(kiloGood0080)
	f := newFake(t, fakePrompter{inputs: []string{"https://kilo.example.test/api/organizations/org-base-0080"},
		secrets: []string{kiloBad0080, kiloGood0080}})
	res, _, err := ResolveCredential(context.Background(), f, k.options(t), llmprovider.ProviderKilo)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	if res.APIKey != kiloGood0080 || len(f.seenSecret) != 2 {
		t.Errorf("key shown as the second %t, %d keys asked; want the second returned after one refusal",
			res.APIKey == kiloGood0080, len(f.seenSecret))
	}
	if reads := k.profileReads(); len(reads) != 0 {
		t.Errorf("%d profile reads; want none for an endpoint that is an organization's", len(reads))
	}
	paths := []string{}
	for _, seen := range k.orgLists {
		paths = append(paths, seen.path)
	}
	if want := []string{"/api/organizations/org-base-0080/models", "/api/organizations/org-base-0080/models"}; !slices.Equal(paths, want) {
		t.Errorf("organization listings at %q; want %q", paths, want)
	}
}

// TestKiloVerify_ProviderOptionOrganizationReadsNoProfile_0080 (the owner's
// ruling of 2026-11-17, relayed by the coordinator: "Share the type internally
// and fix it now (Recommended)"): an organization the calling program gives in
// Options.ProviderOptions, with For(Kilo, catalog.WithKiloOrganization), is
// the listing's organization, so the check reads no profile: the
// organization's listing alone refuses the first key and accepts the second.
func TestKiloVerify_ProviderOptionOrganizationReadsNoProfile_0080(t *testing.T) {
	k := newKilo0080(kiloGood0080)
	o := k.options(t)
	o.ProviderOptions = append(o.ProviderOptions,
		llmprovider.For(llmprovider.ProviderKilo, catalog.WithKiloOrganization("org-opt-0080")))
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{kiloBad0080, kiloGood0080}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
	if err != nil {
		t.Fatalf("ResolveCredential: %v", err)
	}
	if res.APIKey != kiloGood0080 || len(f.seenSecret) != 2 {
		t.Errorf("key shown as the second %t, %d keys asked; want the second returned after one refusal",
			res.APIKey == kiloGood0080, len(f.seenSecret))
	}
	if reads := k.profileReads(); len(reads) != 0 {
		t.Errorf("%d profile reads; want none for an organization given in ProviderOptions", len(reads))
	}
	if len(k.orgLists) != 2 || len(k.public) != 0 || k.orgLists[0].org != "org-opt-0080" {
		t.Errorf("organization listings %+v, public listings %d; want the organization's listing twice, with its header, and no public one",
			k.orgLists, len(k.public))
	}
}
