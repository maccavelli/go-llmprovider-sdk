package signin_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// stall0080 answers nothing: each request waits for its own context to end,
// as a profile that does not answer does. It counts the requests.
type stall0080 struct{ requests atomic.Int32 }

func (s *stall0080) RoundTrip(r *http.Request) (*http.Response, error) {
	s.requests.Add(1)
	<-r.Context().Done()
	return nil, r.Context().Err()
}

// TestVerifyKilo_Stall_0080 (0080-MADR D10, Q5; the owner's ruling of
// 2026-11-17, relayed by the coordinator: "Its own 10 s bound; a stall is
// "cannot say""): a profile that does not answer within VerifyKilo's own bound
// is one that cannot say, so nil; a caller's context that ends first is
// ctx's error. The bound is shortened here (signin.SetVerifyLimit, a test-only
// seam), so the test does not wait 10 s. Each case carries a 3 s caller
// deadline of its own as a guard, so a missing bound fails the test instead of
// hanging it.
func TestVerifyKilo_Stall_0080(t *testing.T) {
	t.Run("the function's own bound passes", func(t *testing.T) {
		defer signin.SetVerifyLimit(150 * time.Millisecond)()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s := &stall0080{}
		start := time.Now()
		err := signin.VerifyKilo(ctx, kiloGood0080, "", llmprovider.WithHTTPClient(&http.Client{Transport: s}))
		if err != nil || s.requests.Load() != 1 {
			t.Errorf("error %v, %d requests; want nil (it cannot say) after one request", err, s.requests.Load())
		}
		if took := time.Since(start); took < 100*time.Millisecond || took > 2*time.Second {
			t.Errorf("took %v; want about the bound, 150 ms", took)
		}
	})
	t.Run("the caller's context ends first", func(t *testing.T) {
		defer signin.SetVerifyLimit(5 * time.Second)()
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()
		s := &stall0080{}
		start := time.Now()
		err := signin.VerifyKilo(ctx, kiloGood0080, "", llmprovider.WithHTTPClient(&http.Client{Transport: s}))
		if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("error %v; want the caller's deadline, not nil", err)
		}
		if took := time.Since(start); took > 2*time.Second {
			t.Errorf("took %v; want the caller's 150 ms, not the 5 s bound", took)
		}
	})
}

// TestVerifyKilo_Mistakes_0080 (D10): a mistake of the caller's, in the
// options, is an error matching ErrInvalidRequest and no request, not a
// profile that cannot say, which would pass a wrong key.
func TestVerifyKilo_Mistakes_0080(t *testing.T) {
	for name, opts := range map[string][]llmprovider.Option{
		"an option scoped to another provider": {catalog.WithClaudeWorkspace("ws-0080")},
		"a base URL that does not parse":       {llmprovider.WithBaseURL("http://[::1")},
	} {
		t.Run(name, func(t *testing.T) {
			k := &kilo0080{accepts: kiloGood0080}
			opts = append(opts, llmprovider.WithHTTPClient(&http.Client{Transport: k}))
			err := signin.VerifyKilo(context.Background(), kiloBad0080, "", opts...)
			if !errors.Is(err, llmprovider.ErrInvalidRequest) || k.requests.Load() != 0 {
				t.Errorf("error %v, %d requests; want ErrInvalidRequest and no request", err, k.requests.Load())
			}
		})
	}
}

// TestVerifyKilo_OrganizationFromTheCredential_0080 (D4; the owner's ruling of
// 2026-11-17, relayed by the coordinator: "Treat it as having an
// organisation: no profile read"): a token whose embedded URL names an
// organization, and a base URL whose path does, are checked by the
// organization's listing, so no profile request is made, whatever the key.
func TestVerifyKilo_OrganizationFromTheCredential_0080(t *testing.T) {
	for name, tc := range map[string]struct {
		key  string
		opts []llmprovider.Option
		want int32
	}{
		"a token's embedded URL":           {"https://kilo.example.test/api/organizations/org-0080:fake-secret-0080", nil, 0},
		"a provider option's organization": {kiloBad0080, []llmprovider.Option{catalog.WithKiloOrganization("org-opt-0080")}, 0},
		"the base URL's path": {kiloBad0080,
			[]llmprovider.Option{llmprovider.WithBaseURL("https://kilo.example.test/api/organizations/org-0080")}, 0},
		"a token's embedded URL with no organization": {"https://kilo.example.test/prefix/api:fake-secret-0080", nil, 1},
	} {
		t.Run(name, func(t *testing.T) {
			k := &kilo0080{accepts: kiloGood0080}
			opts := append([]llmprovider.Option{llmprovider.WithHTTPClient(&http.Client{Transport: k})}, tc.opts...)
			err := signin.VerifyKilo(context.Background(), tc.key, "", opts...)
			if tc.want == 0 && err != nil || k.requests.Load() != tc.want {
				t.Errorf("error %v, %d requests; want %d requests, and no error where none is made", err, k.requests.Load(), tc.want)
			}
		})
	}
}

// TestVerifyKilo_OptionForms_0080 (0080-MADR D4; the owner's ruling of
// 2026-11-17, relayed by the coordinator: "Share the type internally and fix it
// now (Recommended)"): the organization argument and the options decide in the
// order the model listing applies them: options outside a For first, then each
// For's, the argument being a For of its own put before the caller's Fors, and an
// empty argument adding nothing. Each row says whether the check read the
// profile (one request) or left the credential to its organization's listing
// (none).
func TestVerifyKilo_OptionForms_0080(t *testing.T) {
	direct := func(id string) llmprovider.Option { return catalog.WithKiloOrganization(id) }
	forKilo := func(id string) llmprovider.Option {
		return llmprovider.For(llmprovider.ProviderKilo, catalog.WithKiloOrganization(id))
	}
	for _, tc := range []struct {
		name     string
		argument string
		opts     []llmprovider.Option
		want     int32
	}{
		{"nothing", "", nil, 1},
		{"a direct option", "", []llmprovider.Option{direct("org-x-0080")}, 0},
		{"a For option", "", []llmprovider.Option{forKilo("org-x-0080")}, 0},
		{"the argument replaces a direct empty option", "org-a-0080", []llmprovider.Option{direct("")}, 0},
		{"an empty For option replaces the argument", "org-a-0080", []llmprovider.Option{forKilo("")}, 1},
		{"a For option replaces the argument", "org-a-0080", []llmprovider.Option{forKilo("org-x-0080")}, 0},
		{"an empty For option replaces a direct one", "", []llmprovider.Option{direct("org-x-0080"), forKilo("")}, 1},
		{"an empty For option replaces a direct one that follows it", "", []llmprovider.Option{forKilo(""), direct("org-x-0080")}, 1},
		{"a For for another provider is no organization", "",
			[]llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, catalog.WithKiloOrganization("org-c-0080"))}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := &kilo0080{accepts: kiloGood0080}
			opts := append([]llmprovider.Option{llmprovider.WithHTTPClient(&http.Client{Transport: k})}, tc.opts...)
			err := signin.VerifyKilo(context.Background(), kiloGood0080, tc.argument, opts...)
			if err != nil || k.requests.Load() != tc.want {
				t.Errorf("error %v, %d requests; want nil and %d", err, k.requests.Load(), tc.want)
			}
		})
	}
}
