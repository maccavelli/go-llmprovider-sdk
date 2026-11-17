package signinflow

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestVerifyKilo_OrganizationFromTheCredential_0080 (0080-MADR D4; the owner's
// ruling of 2026-11-17, relayed by the coordinator: "Treat it as having an
// organisation: no profile read"): the check makes no request when the
// listing is an organization's, whichever way the organization is named, and
// one request otherwise.
func TestVerifyKilo_OrganizationFromTheCredential_0080(t *testing.T) {
	for _, tc := range []struct {
		name         string
		token        string
		organization string
		base         string
		reads        int
	}{
		{"none", "fake-kilo-key-0080", "", "https://kilo.example.test/api/gateway", 1},
		{"the caller's", "fake-kilo-key-0080", "org-0080", "https://kilo.example.test/api/gateway", 0},
		{"a token's embedded URL", "https://kilo.example.test/api/organizations/org-0080:fake-secret-0080", "", "", 0},
		{"a token's embedded URL without one", "https://kilo.example.test/prefix/api:fake-secret-0080", "", "", 1},
		{"the base URL's path", "fake-kilo-key-0080", "", "https://kilo.example.test/api/organizations/org-0080", 0},
		{"no base URL", "fake-kilo-key-0080", "", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			read := func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
				reads++
				return auth.KiloAccount{}, nil
			}
			var opts []llmprovider.Option
			if tc.base != "" {
				opts = append(opts, llmprovider.WithBaseURL(tc.base))
			}
			refused, unverified := VerifyKilo(context.Background(), read, tc.token, tc.organization, opts...)
			if reads != tc.reads || refused != nil || unverified != nil {
				t.Errorf("%d reads, refused %v, unverified %v; want %d reads and neither", reads, refused, unverified, tc.reads)
			}
		})
	}
}

// TestCheckKiloOptions_0080 (0080-MADR D10): a caller's mistake in the options
// is reported, not read as a profile that cannot say; good options are not.
func TestCheckKiloOptions_0080(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		bad  bool
	}{
		{"none", nil, false},
		{"a base URL", []llmprovider.Option{llmprovider.WithBaseURL("https://kilo.example.test/api/gateway")}, false},
		{"a client", []llmprovider.Option{llmprovider.WithHTTPClient(http.DefaultClient)}, false},
		{"an option for another provider", []llmprovider.Option{catalog.WithClaudeWorkspace("ws-0080")}, true},
		{"a base URL that does not parse", []llmprovider.Option{llmprovider.WithBaseURL("http://[::1")}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckKiloOptions(tc.opts...)
			if tc.bad != (err != nil) || (tc.bad && !errors.Is(err, llmprovider.ErrInvalidRequest)) {
				t.Errorf("CheckKiloOptions = %v; want a mistake matching ErrInvalidRequest: %t", err, tc.bad)
			}
		})
	}
}

// TestDefaultCheckLimit_0080 (MADR Q5, D10): the one limit of a check is 10 s,
// the wizard's default discovery limit and signin.VerifyKilo's own bound.
func TestDefaultCheckLimit_0080(t *testing.T) {
	if DefaultCheckLimit != 10*time.Second {
		t.Errorf("DefaultCheckLimit = %v; want 10 s", DefaultCheckLimit)
	}
}

// TestVerifyKilo_OrganizationPrecedence_0080 (0080-MADR D4; the owner's ruling
// of 2026-11-17, relayed by the coordinator: "Share the type internally and fix
// it now (Recommended)"): the check decides whether the credential has an
// organization in the order the listing applies the options: every option
// outside a For first, in order, then those inside For(Kilo, ...), in order,
// the last catalog.WithKiloOrganization winning, an empty one included. A
// non-empty organization argument is one more For option, before the For
// options of opts; an empty one adds nothing. After those, a token's embedded
// URL's organization, then the base URL's path's. The wizard gives no argument:
// its own For is among the options it passes. TestKiloListing_OrganizationPrecedence_0080,
// in catalog, holds what the listing sends for the same inputs.
func TestVerifyKilo_OrganizationPrecedence_0080(t *testing.T) {
	const plain = "fake-kilo-key-0080"
	const tokenOrg = "https://kilo.example.test/api/organizations/org-token-0080:fake-secret-0080"
	const baseOrg = "https://kilo.example.test/api/organizations/org-base-0080"
	direct := func(id string) llmprovider.Option { return catalog.WithKiloOrganization(id) }
	forKilo := func(id string) llmprovider.Option {
		return llmprovider.For(llmprovider.ProviderKilo, catalog.WithKiloOrganization(id))
	}
	for _, tc := range []struct {
		name     string
		token    string
		argument string
		options  []llmprovider.Option
		base     string
		reads    int
	}{
		{"nothing", plain, "", nil, "", 1},
		{"the argument", plain, "org-a-0080", nil, "", 0},
		{"a direct option", plain, "", []llmprovider.Option{direct("org-opt-0080")}, "", 0},
		{"a For option", plain, "", []llmprovider.Option{forKilo("org-opt-0080")}, "", 0},
		{"the argument replaces a direct option, even an empty one", plain, "org-a-0080", []llmprovider.Option{direct("")}, "", 0},
		{"a For option replaces the argument", plain, "org-a-0080", []llmprovider.Option{forKilo("org-opt-0080")}, "", 0},
		{"an empty For option replaces the argument", plain, "org-a-0080", []llmprovider.Option{forKilo("")}, "", 1},
		{"an empty For option replaces a direct one", plain, "", []llmprovider.Option{direct("org-opt-0080"), forKilo("")}, "", 1},
		{"an empty For option replaces a direct one that follows it", plain, "", []llmprovider.Option{forKilo(""), direct("org-opt-0080")}, "", 1},
		{"a For option replaces an empty For option", plain, "", []llmprovider.Option{forKilo(""), forKilo("org-opt-0080")}, "", 0},
		{"a later empty For option replaces an earlier one", plain, "", []llmprovider.Option{forKilo("org-opt-0080"), forKilo("")}, "", 1},
		{"a For option for another provider is no organization", plain, "",
			[]llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, catalog.WithKiloOrganization("org-c-0080"))}, "", 1},
		{"a token's embedded URL", tokenOrg, "", nil, "", 0},
		{"a token's embedded URL under an empty For option", tokenOrg, "", []llmprovider.Option{forKilo("")}, "", 0},
		{"the base URL's path", plain, "", nil, baseOrg, 0},
		{"the base URL's path under an empty For option", plain, "", []llmprovider.Option{forKilo("")}, baseOrg, 0},
		{"a token's embedded URL replaces the base URL's, and names none", "https://kilo.example.test/prefix/api:fake-secret-0080",
			"", nil, baseOrg, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			read := func(context.Context, string, ...llmprovider.Option) (auth.KiloAccount, error) {
				reads++
				return auth.KiloAccount{}, nil
			}
			opts := tc.options
			if tc.base != "" {
				opts = append([]llmprovider.Option{llmprovider.WithBaseURL(tc.base)}, opts...)
			}
			if refused, unverified := VerifyKilo(context.Background(), read, tc.token, tc.argument, opts...); refused != nil || unverified != nil {
				t.Fatalf("refused %v, unverified %v; want neither", refused, unverified)
			}
			if reads != tc.reads {
				t.Errorf("%d profile reads; want %d", reads, tc.reads)
			}
		})
	}
}
