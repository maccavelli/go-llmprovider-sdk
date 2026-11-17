package wizard

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestKiloVerify_OptionForms_0080 (0080-MADR D4; the owner's ruling of
// 2026-11-17, relayed by the coordinator: "Share the type internally and fix it
// now (Recommended)"): whatever way an organization is given, the sign-in
// check and the listing agree, because the check is given the options the
// listing is given, built in one place (listingOptions). Each row runs a
// credential the stand-in accepts and says what the check did (profile
// requests) and where the listing went (URL and organization header); and, in
// every row, either the profile was read or the listing was an organization's:
// never neither, which would accept a wrong key with no check. llmprovider's
// ResolveOptions applies the options outside a For first, then each For's, so
// a caller's option outside a For is replaced by the wizard's own For, which
// carries the Result's organization, and that by a For of the caller's.
func TestKiloVerify_OptionForms_0080(t *testing.T) {
	forKilo := func(id string) llmprovider.Option {
		return llmprovider.For(llmprovider.ProviderKilo, catalog.WithKiloOrganization(id))
	}
	direct := func(id string) llmprovider.Option { return catalog.WithKiloOrganization(id) }
	const (
		public  = "/api/gateway/models"
		orgPath = "/api/organizations/%s/models"
		typed   = "fake-kilo-typed-key-0080"
		orgBase = "https://kilo.example.test/api/organizations/org-b-0080"
		tokURL  = "https://kilo.example.test/prefix/api:fake-secret-0080"
		tokOrg  = "https://kilo.example.test/api/organizations/org-t-0080:fake-secret-0080"
	)
	org := func(id string) string { return strings.Replace(orgPath, "%s", id, 1) }

	type row struct {
		name     string
		endpoint string
		key      string
		opts     []llmprovider.Option
		kept     string // a kept session's organization; "" is a typed key
		device   bool   // a device session that chooses org-dev-0080
		profile  int    // the profile requests, the check's or the menu's
		path     string // where the listing went
		header   string // its organization header
	}
	rows := []row{
		// The reviewer's ten, with a typed key.
		{name: "a direct option", key: typed, opts: []llmprovider.Option{direct("org-x-0080")},
			profile: 1, path: public},
		{name: "a For option", key: typed, opts: []llmprovider.Option{forKilo("org-x-0080")},
			profile: 0, path: org("org-x-0080"), header: "org-x-0080"},
		{name: "a For org-x then a For empty", key: typed, opts: []llmprovider.Option{forKilo("org-x-0080"), forKilo("")},
			profile: 1, path: public},
		{name: "a direct org-x then a For empty", key: typed, opts: []llmprovider.Option{direct("org-x-0080"), forKilo("")},
			profile: 1, path: public},
		{name: "a For empty then a direct org-x", key: typed, opts: []llmprovider.Option{forKilo(""), direct("org-x-0080")},
			profile: 1, path: public},
		{name: "a token with a URL and no organization, under an empty For", key: tokURL,
			opts: []llmprovider.Option{forKilo("")}, profile: 1, path: "/prefix/api/gateway/models"},
		{name: "a token with an organization path, under an empty For", key: tokOrg,
			opts: []llmprovider.Option{forKilo("")}, profile: 0, path: org("org-t-0080"), header: "org-t-0080"},
		{name: "a base path org-b and a For org-y", endpoint: orgBase, key: typed,
			opts: []llmprovider.Option{forKilo("org-y-0080")}, profile: 0, path: org("org-y-0080"), header: "org-y-0080"},
		{name: "a For for another provider carrying a Kilo organization", key: typed,
			opts:    []llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, catalog.WithKiloOrganization("org-c-0080"))},
			profile: 1, path: public},
		{name: "a direct empty option", key: typed, opts: []llmprovider.Option{direct("")},
			profile: 1, path: public},
		// The Result's own organization, a kept session's, against each form.
		{name: "a kept session's organization", kept: "org-r-0080",
			profile: 0, path: org("org-r-0080"), header: "org-r-0080"},
		{name: "a kept session's organization and a direct option", kept: "org-r-0080",
			opts: []llmprovider.Option{direct("org-x-0080")}, profile: 0, path: org("org-r-0080"), header: "org-r-0080"},
		{name: "a kept session's organization and a direct empty option", kept: "org-r-0080",
			opts: []llmprovider.Option{direct("")}, profile: 0, path: org("org-r-0080"), header: "org-r-0080"},
		{name: "a kept session's organization and a For option", kept: "org-r-0080",
			opts: []llmprovider.Option{forKilo("org-x-0080")}, profile: 0, path: org("org-x-0080"), header: "org-x-0080"},
		{name: "a kept session's organization and an empty For option", kept: "org-r-0080",
			opts: []llmprovider.Option{forKilo("")}, profile: 1, path: public},
		// A device session's choice.
		{name: "a device session's choice", device: true,
			profile: 1, path: org("org-dev-0080"), header: "org-dev-0080"},
		{name: "a device session's choice and an empty For option", device: true,
			opts: []llmprovider.Option{forKilo("")}, profile: 1, path: public},
		{name: "a device session's choice and a direct option", device: true,
			opts: []llmprovider.Option{direct("org-x-0080")}, profile: 1, path: org("org-dev-0080"), header: "org-dev-0080"},
		{name: "a device session's choice and a For option", device: true,
			opts: []llmprovider.Option{forKilo("org-x-0080")}, profile: 1, path: org("org-x-0080"), header: "org-x-0080"},
	}
	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			var k *kilo0080
			var o Options
			var f *fakePrompter
			switch {
			case tc.kept != "":
				k = newKilo0080(keptAccess0066)
				o = k.options(t)
				o.TokenStore = keptKilo0080(keptAccess0066)
				o.Existing = Result{Provider: llmprovider.ProviderKilo, Kind: CredOAuth, Organization: tc.kept}
				f = newFake(t, fakePrompter{confirms: []bool{true}, inputs: []string{acceptDefault}})
			case tc.device:
				stubKiloDeviceLogin0080(t)
				k = newKilo0080(kiloWizardToken)
				k.profile = func(r *http.Request) (*http.Response, error) {
					return respond0080(r, http.StatusOK,
						`{"organizations":[{"id":"org-dev-0080","name":"Dev","role":"member"}],"hasPersonalAccount":true}`), nil
				}
				o = k.options(t)
				o.TokenStore = newStore0059()
				f = newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1, 1}})
			default:
				k = newKilo0080(tc.key)
				o = k.options(t)
				endpoint := tc.endpoint
				if endpoint == "" {
					endpoint = acceptDefault
				}
				f = newFake(t, fakePrompter{inputs: []string{endpoint}, secrets: []string{tc.key}})
			}
			o.ProviderOptions = append(o.ProviderOptions, tc.opts...)
			if _, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo); err != nil {
				t.Fatalf("ResolveCredential: %v", err)
			}
			profile := len(k.profileReads())
			listings := append(append([]kiloRead0080(nil), k.orgLists...), k.public...)
			if len(listings) != 1 {
				t.Fatalf("%d listing requests; want one", len(listings))
			}
			got := listings[0]
			if profile != tc.profile || got.path != tc.path || got.org != tc.header {
				t.Errorf("profile requests %d, listing %q with header %q; want %d, %q with %q",
					profile, got.path, got.org, tc.profile, tc.path, tc.header)
			}
			// Never neither: a wrong key would be accepted unchecked.
			if profile == 0 && !strings.Contains(got.path, "/organizations/") {
				t.Errorf("no profile request and the listing %q is not an organization's: a wrong key would be accepted", got.path)
			}
		})
		if tc.kept != "" || tc.device {
			continue
		}
		// The same options with a key the service refuses: refused, whichever
		// way the check or the listing found it.
		t.Run(tc.name+", a wrong key", func(t *testing.T) {
			k := newKilo0080()
			o := k.options(t)
			o.ProviderOptions = append(o.ProviderOptions, tc.opts...)
			endpoint := tc.endpoint
			if endpoint == "" {
				endpoint = acceptDefault
			}
			f := newFake(t, fakePrompter{inputs: []string{endpoint}, secrets: []string{tc.key, tc.key, tc.key}})
			_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
			wantRefused0080(t, err, "refused 3 keys")
		})
	}
}
