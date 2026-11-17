package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/kiloendpoint"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestKiloListing_OrganizationPrecedence_0080 (0080-MADR D4): the organization
// a Kilo listing is made for, and the request it sends, follow one precedence:
//
//  1. the explicit organization: the last WithKiloOrganization applied, an
//     empty one included, which replaces an earlier one. The options outside
//     a For apply first, in order, then those inside llmprovider.For(Kilo, ...),
//     in order, so a For option replaces a direct one wherever it stands;
//  2. else the one a token's embedded URL names;
//  3. else the one the base URL's own path names.
//
// A request for an organization is its listing, at /api/organizations/{id}/models,
// with the organization header except for the third source, which sends none.
// The sign-in check decides whether a credential has an organization by
// kiloendpoint.OptionOrganization and kiloendpoint.Organization over the
// resolved options, so each case asserts both: what the listing sent, and what
// the check decides, which must agree. TestKiloVerify_OptionForms_0080, in
// wizard, holds the same against the options the wizard itself builds. A listing the check would read the profile for is
// the public one, at /models, with no header.
func TestKiloListing_OrganizationPrecedence_0080(t *testing.T) {
	const orgToken = "/api/organizations/org-token-0080"
	const orgBase = "/api/organizations/org-base-0080"
	org := func(id string) llmprovider.Option { return WithKiloOrganization(id) }
	forKilo := func(id string) llmprovider.Option {
		return llmprovider.For(llmprovider.ProviderKilo, WithKiloOrganization(id))
	}
	for _, tc := range []struct {
		name     string
		options  []llmprovider.Option
		tokenURL string // a token embedding the server's URL and this path; "" is a plain token
		basePath string // the path of the base URL
		want     string // the organization the listing is for
		wantURL  string // the path the listing was requested at
		wantHdr  string // the organization header, "" when none
	}{
		{"none", nil, "", "", "", "/models", ""},
		{"a provider option", []llmprovider.Option{org("org-opt-0080")}, "", "",
			"org-opt-0080", "/api/organizations/org-opt-0080/models", "org-opt-0080"},
		{"a later provider option wins", []llmprovider.Option{org("org-a-0080"), org("org-b-0080")}, "", "",
			"org-b-0080", "/api/organizations/org-b-0080/models", "org-b-0080"},
		{"a later empty option replaces an earlier one", []llmprovider.Option{org("org-a-0080"), org("")}, "", "",
			"", "/models", ""},
		{"a token's embedded URL", nil, orgToken, "",
			"org-token-0080", "/api/organizations/org-token-0080/models", "org-token-0080"},
		{"the base URL's path", nil, "", orgBase,
			"org-base-0080", "/api/organizations/org-base-0080/models", ""},
		{"a For option replaces a direct one before it", []llmprovider.Option{org("org-a-0080"), forKilo("org-b-0080")}, "", "",
			"org-b-0080", "/api/organizations/org-b-0080/models", "org-b-0080"},
		{"a For option replaces a direct one after it", []llmprovider.Option{forKilo("org-a-0080"), org("org-b-0080")}, "", "",
			"org-a-0080", "/api/organizations/org-a-0080/models", "org-a-0080"},
		{"an empty For option replaces a direct one", []llmprovider.Option{org("org-a-0080"), forKilo("")}, "", "",
			"", "/models", ""},
		{"a For option for another provider is no organization",
			[]llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, WithKiloOrganization("org-c-0080"))}, "", "",
			"", "/models", ""},
		{"an option beats a token's", []llmprovider.Option{org("org-opt-0080")}, orgToken, "",
			"org-opt-0080", "/api/organizations/org-opt-0080/models", "org-opt-0080"},
		{"an option beats the base URL's", []llmprovider.Option{org("org-opt-0080")}, "", orgBase,
			"org-opt-0080", "/api/organizations/org-opt-0080/models", "org-opt-0080"},
		{"a token's beats the base URL's", nil, orgToken, orgBase,
			"org-token-0080", "/api/organizations/org-token-0080/models", "org-token-0080"},
		{"an option beats a token's and the base URL's", []llmprovider.Option{org("org-opt-0080")}, orgToken, orgBase,
			"org-opt-0080", "/api/organizations/org-opt-0080/models", "org-opt-0080"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var path, header string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				path, header = r.URL.Path, r.Header.Get(kiloendpoint.OrganizationHeader)
				mu.Unlock()
				_, _ = w.Write([]byte(`{"data":[{"id":"kilo-auto/free"}]}`))
			}))
			t.Cleanup(srv.Close)

			token := "plain-fake-key-0080"
			if tc.tokenURL != "" {
				token = srv.URL + tc.tokenURL + ":fake-secret-0080"
			}
			base := srv.URL + tc.basePath
			opts := append([]llmprovider.Option{llmprovider.WithBaseURL(base), testCache(t)}, tc.options...)
			if _, err := listT(context.Background(), llmprovider.ProviderKilo, llmprovider.NewStaticToken(token), opts...); err != nil {
				t.Fatalf("List: %v", err)
			}
			mu.Lock()
			gotPath, gotHeader := path, header
			mu.Unlock()
			if gotPath != tc.wantURL || gotHeader != tc.wantHdr {
				t.Errorf("listing requested %q with header %q; want %q with %q", gotPath, gotHeader, tc.wantURL, tc.wantHdr)
			}

			// What the check decides from the same inputs.
			st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, opts)
			if err != nil {
				t.Fatalf("ResolveOptions: %v", err)
			}
			explicit := kiloendpoint.OptionOrganization(st.Values())
			if got := kiloendpoint.Organization(st.BaseURL(), token, explicit); got != tc.want {
				t.Errorf("the check decides organization %q; want %q, the listing's", got, tc.want)
			}
		})
	}
}
