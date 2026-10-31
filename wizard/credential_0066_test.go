package wizard

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// 0066-PLAN Phase 4 (V6, V7; Q5 (a), Q4 (a)): a failed listing's catalog
// says why, and a kept session the listing refuses leads to the sign-in menu.

// failingServer0066 answers every request with status.
func failingServer0066(t *testing.T, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"error":{"message":"down"}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// staticCompatRegistry0066 is compatRegistry0066 with a static catalog in
// the descriptor.
func staticCompatRegistry0066(t *testing.T) *llmprovider.Registry {
	t.Helper()
	r := llmprovider.NewRegistry()
	d := chatcompat.Descriptor(compatID0066, "LM Studio")
	d.StaticModels = thirdPartyStatic
	if err := r.Register(d, func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
		return chatcompat.New(append(opts, chatcompat.WithID(compatID0066))...)
	}); err != nil {
		t.Fatal(err)
	}
	return r
}

// TestResolve_FailedListingErr_0066: the MADR's probes L and D. A listing
// that fails is Live false with an Err matching the failure, on each path;
// with no listing, Err is nil.
func TestResolve_FailedListingErr_0066(t *testing.T) {
	unavailable := &llmprovider.APIError{Provider: "acme", Status: http.StatusInternalServerError, Message: "down"}
	offline := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}
	})}
	for _, tc := range []struct {
		name string
		run  func(t *testing.T) (catalog.Catalog, error)
		want error
	}{
		{"a provider's ListModels, HTTP 500", func(t *testing.T) (catalog.Catalog, error) {
			f := newFake(t, fakePrompter{secrets: []string{"acme-key"}})
			_, cat, err := ResolveCredential(context.Background(), f,
				Options{Registry: registry0056(t, thirdPartyStatic, unavailable), Discover: true}, thirdPartyID)
			return cat, err
		}, llmprovider.ErrProviderUnavailable},
		{"a Chat Completions server, HTTP 502", func(t *testing.T) (catalog.Catalog, error) {
			f := newFake(t, fakePrompter{inputs: []string{failingServer0066(t, http.StatusBadGateway).URL + "/v1"}})
			_, cat, err := ResolveCredential(context.Background(), f, compatOptions0066(t), compatID0066)
			return cat, err
		}, llmprovider.ErrProviderUnavailable},
		{"a Chat Completions server beside a static catalog, HTTP 502", func(t *testing.T) (catalog.Catalog, error) {
			f := newFake(t, fakePrompter{inputs: []string{failingServer0066(t, http.StatusBadGateway).URL + "/v1"}})
			o := compatOptions0066(t)
			o.Registry = staticCompatRegistry0066(t)
			_, cat, err := ResolveCredential(context.Background(), f, o, compatID0066)
			if !slices.Equal(cat.Recommended, thirdPartyStatic) {
				t.Errorf("Recommended %q; want the static %q", cat.Recommended, thirdPartyStatic)
			}
			return cat, err
		}, llmprovider.ErrProviderUnavailable},
		{"Claude offline", func(t *testing.T) (catalog.Catalog, error) {
			f := newFake(t, fakePrompter{secrets: []string{"sk-ant-api03-made-up-0066"}})
			_, cat, err := ResolveCredential(context.Background(), f, Options{Discover: true, DiscoverLimit: 2 * time.Second,
				HTTPClient: offline, ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}},
				llmprovider.ProviderClaude)
			return cat, err
		}, llmprovider.ErrOffline},
		{"no Discover", func(t *testing.T) (catalog.Catalog, error) {
			f := newFake(t, fakePrompter{secrets: []string{"acme-key"}})
			_, cat, err := ResolveCredential(context.Background(), f,
				Options{Registry: registry0056(t, thirdPartyStatic, unavailable)}, thirdPartyID)
			return cat, err
		}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cat, err := tc.run(t)
			if err != nil {
				t.Fatal(err)
			}
			if cat.Live {
				t.Errorf("Live true; want false")
			}
			if tc.want == nil && cat.Err != nil || tc.want != nil && !errors.Is(cat.Err, tc.want) {
				t.Errorf("Err = %v; want one matching %v", cat.Err, tc.want)
			}
		})
	}
}

// TestResolve_LiveMeans_0066: what Live says beyond a failure (0066-MADR,
// amendment "what Live says"). A listing answered with no usable id is the
// static catalog, Live false with a nil Err; one read again at once is
// served from the listing cache, Live true, with no request, though the
// server now fails.
func TestResolve_LiveMeans_0066(t *testing.T) {
	status, body, requests := http.StatusOK, `{"data":[]}`, 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return answer0059(r, status, body), nil
	})}
	resolve := func(t *testing.T) catalog.Catalog {
		t.Helper()
		f := newFake(t, fakePrompter{selects: []int{0}, secrets: []string{"xai-made-up-key-0066"}})
		_, cat, err := ResolveCredential(context.Background(), f, Options{Discover: true, HTTPClient: client,
			ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}, llmprovider.ProviderGrok)
		if err != nil {
			t.Fatal(err)
		}
		return cat
	}
	t.Run("nothing usable", func(t *testing.T) {
		if cat := resolve(t); cat.Live || cat.Err != nil || cat.Source != catalog.ListStatic {
			t.Errorf("Live %v, Err %v, Source %q; want false, nil, %q", cat.Live, cat.Err, cat.Source, catalog.ListStatic)
		}
	})
	t.Run("read again at once", func(t *testing.T) {
		status, body, requests = http.StatusOK, `{"data":[{"id":"grok-4.5"}]}`, 0
		if cat := resolve(t); !cat.Live || requests != 1 {
			t.Fatalf("first: Live %v after %d requests; want true after 1", cat.Live, requests)
		}
		status = http.StatusServiceUnavailable
		if cat := resolve(t); !cat.Live || cat.Err != nil || requests != 1 {
			t.Errorf("again: Live %v, Err %v, %d requests; want true, nil, still 1", cat.Live, cat.Err, requests)
		}
	})
}

// keptAccess0066 is the kept session's token; a listing that carries it is
// refused.
const keptAccess0066 = "kept-access-0066"

// accepting0066 answers a listing that carries "Bearer <accept>" with one
// model, and refuses every other request, a token refresh's among them.
func accepting0066(accept string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if accept != "" && r.Header.Get("Authorization") == "Bearer "+accept && !strings.HasSuffix(r.URL.Path, "/oauth/token") {
			return answer0059(r, http.StatusOK, `{"data":[{"id":"grok-4.5"}]}`), nil
		}
		return answer0059(r, http.StatusUnauthorized, `{"error":"invalid_grant"}`), nil
	})}
}

// keptGrok0066 puts a Grok session in a store, names it in Existing, and
// lists through client.
func keptGrok0066(t *testing.T, client *http.Client) (*store0059, Options) {
	t.Helper()
	store := newStore0059()
	kept := testOAuthSession(llmprovider.ProviderGrok)
	kept.Access, kept.Refresh, kept.Expiry = keptAccess0066, "kept-refresh-0066", time.Now().Add(2*time.Hour)
	store.held[llmprovider.ProviderGrok] = kept
	return store, Options{TokenStore: store, Discover: true, HTTPClient: client,
		Providers:       []llmprovider.ProviderID{llmprovider.ProviderGrok},
		Existing:        Result{Provider: llmprovider.ProviderGrok, Kind: CredOAuth},
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}
}

// assertKeptRefused0066 checks the one keep question, the one notice and
// the one method menu, and that no prompt shows the kept token.
func assertKeptRefused0066(t *testing.T, f *fakePrompter, label string) {
	t.Helper()
	if len(f.seenConfirm) != 1 || !strings.Contains(f.seenConfirm[0], "Keep the existing session") {
		t.Errorf("confirms %q; want the keep question once", f.seenConfirm)
	}
	if n := countContaining(f.seenNotify, "the saved "+label+" session was refused (authentication failed"); n != 1 ||
		countContaining(f.seenNotify, "); sign in again") != 1 {
		t.Errorf("notices %q; want one that the saved session was refused, by class", f.seenNotify)
	}
	if n := countContaining(f.seenSelect, "Choose how to authenticate with "+label); n != 1 {
		t.Errorf("selects %q; want the method menu once", f.seenSelect)
	}
	for _, shown := range slices.Concat(f.seenNotify, f.seenConfirm, f.seenSelect) {
		if strings.Contains(shown, keptAccess0066) {
			t.Errorf("a prompt shows the kept token: %q", shown)
		}
	}
}

// TestKept_RefusedLeadsToSignIn_0066: the MADR's probe G, then a browser
// sign-in the listing accepts. The method menu is shown, the new session is
// saved, and the Result names it.
func TestKept_RefusedLeadsToSignIn_0066(t *testing.T) {
	client := accepting0066("oauth-access")
	store, o := keptGrok0066(t, client)
	signedIn := browserGrok0059(t, client, false)
	f := newFake(t, fakePrompter{blankSearches: true, confirms: []bool{true},
		selects: []int{0, methodIdx0059(t, o, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth), 0}})
	res, err := ConfigureLLM(context.Background(), f, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != CredOAuth || res.Model != "grok-4.5" || *signedIn == nil || !res.TokenExpiry.Equal((*signedIn).Expiry) {
		t.Errorf("Result %+v; want the new session's, model grok-4.5", res)
	}
	if saves := store.recorded(); len(saves) != 1 || saves[0].access != "oauth-access" || saves[0].store != store {
		t.Errorf("saves %+v; want the new session, once", saves)
	}
	assertKeptRefused0066(t, f, "Grok (xAI)")
}

// TestKept_RefusedThenNewRefused_0066: a new sign-in the listing refuses as
// well ends the run, and the store still holds the kept session.
func TestKept_RefusedThenNewRefused_0066(t *testing.T) {
	client := accepting0066("")
	store, o := keptGrok0066(t, client)
	browserGrok0059(t, client, false)
	f := newFake(t, fakePrompter{confirms: []bool{true},
		selects: []int{methodIdx0059(t, o, llmprovider.ProviderGrok, llmprovider.AuthBrowserOAuth)}})
	_, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderGrok)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "Grok (xAI) refused the sign-in; sign in again") {
		t.Fatalf("err = %v; want the new sign-in refused", err)
	}
	if saves, held := store.recorded(), store.held[llmprovider.ProviderGrok]; len(saves) != 0 ||
		held == nil || held.Access != keptAccess0066 || held.Refresh != "kept-refresh-0066" {
		t.Errorf("saves %+v, the store holds %+v; want none, and the kept session", saves, held)
	}
	assertKeptRefused0066(t, f, "Grok (xAI)")
}

// TestKept_KiloRefusedLeadsToSignIn_0066: the same for a kept Kilo device
// session. The new one is saved, and the kept session's organization does
// not outlive it.
func TestKept_KiloRefusedLeadsToSignIn_0066(t *testing.T) {
	stubKilo(t, auth.KiloAccount{}, nil)
	store := newStore0059()
	store.held[llmprovider.ProviderKilo] = &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: keptAccess0066}
	o := Options{TokenStore: store, Discover: true, HTTPClient: accepting0066(kiloWizardToken),
		Existing:        Result{Provider: llmprovider.ProviderKilo, Kind: CredOAuth, Organization: "org-kept"},
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}
	d, _ := o.registry().Descriptor(llmprovider.ProviderKilo)
	f := newFake(t, fakePrompter{confirms: []bool{true}, inputs: []string{acceptDefault},
		selects: []int{methodIdx0059(t, o, llmprovider.ProviderKilo, llmprovider.AuthDeviceCode)}})
	res, _, err := ResolveCredential(context.Background(), f, o, llmprovider.ProviderKilo)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != CredOAuth || res.Organization != "" {
		t.Errorf("Kind %q, Organization %q; want oauth, the personal account", res.Kind, res.Organization)
	}
	if saves := store.recorded(); len(saves) != 1 || saves[0].access != kiloWizardToken {
		t.Errorf("saves %+v; want the device sign-in's token, once", saves)
	}
	assertKeptRefused0066(t, f, d.Label)
}
