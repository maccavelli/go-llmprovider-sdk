package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// 0066-PLAN Phase 2 (V2, V3, V4; Q1 (a)): a Chat Completions server, keyed
// or not, is configured through the wizard to a Result that builds its
// provider, with the catalog catalog builds for it.

// compatServer0066 is a Chat Completions server listing n models,
// local-model-00 on, at /v1/models. With a key, a request without
// "Bearer <key>" is answered 401. It records each listing request's
// Authorization header.
type compatServer0066 struct {
	*httptest.Server
	mu    sync.Mutex
	auths []string
}

func newCompatServer0066(t *testing.T, n int, key string) *compatServer0066 {
	t.Helper()
	s := &compatServer0066{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		s.mu.Lock()
		s.auths = append(s.auths, r.Header.Get("Authorization"))
		s.mu.Unlock()
		if key != "" && r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"message":"no key","type":"invalid_request_error"}}`)
			return
		}
		data := make([]string, n)
		for i := range data {
			data[i] = fmt.Sprintf(`{"id":"local-model-%02d","object":"model"}`, i)
		}
		_, _ = io.WriteString(w, `{"object":"list","data":[`+strings.Join(data, ",")+`]}`)
	}))
	t.Cleanup(s.Close)
	return s
}

// seen is a copy of the Authorization headers so far.
func (s *compatServer0066) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.auths)
}

// compatOptions0066 lists the registry chatcompat's package doc shows, off
// models.dev, in t's own cache.
func compatOptions0066(t *testing.T) Options {
	return Options{Registry: compatRegistry0066(t), Discover: true,
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}
}

// TestCompat_BlankEndpointAskedAgain_0066: the MADR's probe A. A provider
// with no default endpoint does not take a blank one: a notice, and the
// prompt again.
func TestCompat_BlankEndpointAskedAgain_0066(t *testing.T) {
	srv := newCompatServer0066(t, 2, "")
	f := newFake(t, fakePrompter{inputs: []string{"", srv.URL + "/v1"}})
	res, cat, err := ResolveCredential(context.Background(), f, compatOptions0066(t), compatID0066)
	if err != nil {
		t.Fatal(err)
	}
	if res.BaseURL != srv.URL+"/v1" || !cat.Live {
		t.Errorf("BaseURL %q, Live %v; want %q, true", res.BaseURL, cat.Live, srv.URL+"/v1")
	}
	if want := []string{"LM Studio endpoint", "LM Studio endpoint"}; !slices.Equal(f.seenInput, want) {
		t.Errorf("inputs %q; want %q", f.seenInput, want)
	}
	if !slices.Contains(f.seenNotify, "LM Studio needs an endpoint") {
		t.Errorf("notices %q; want \"LM Studio needs an endpoint\"", f.seenNotify)
	}
}

// TestCompat_NoEndpointPromptNeedsURL_0066: probe A under NoEndpointPrompt.
// With no Existing.BaseURL there is nothing to list: a plain error, and
// nothing asked.
func TestCompat_NoEndpointPromptNeedsURL_0066(t *testing.T) {
	f := newFake(t, fakePrompter{})
	o := compatOptions0066(t)
	o.NoEndpointPrompt = true
	_, _, err := ResolveCredential(context.Background(), f, o, compatID0066)
	const want = "wizard: LM Studio needs an endpoint: pass Options.Existing.BaseURL"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v; want %q", err, want)
	}
	if len(f.seenInput)+len(f.seenSecret)+len(f.seenConfirm) != 0 {
		t.Errorf("asked inputs %q, secrets %q, confirms %q; want nothing", f.seenInput, f.seenSecret, f.seenConfirm)
	}
}

// TestCompat_KeyAskedAfterRefusal_0066: the MADR's probe B. A server that
// refuses the request without a key is asked for one, once; the second
// listing carries it, and the Result is an API key's.
func TestCompat_KeyAskedAfterRefusal_0066(t *testing.T) {
	srv := newCompatServer0066(t, 2, "lm-key")
	f := newFake(t, fakePrompter{selects: []int{0, 0}, inputs: []string{srv.URL + "/v1"}, secrets: []string{"lm-key"},
		blankSearches: true})
	res, err := ConfigureLLM(context.Background(), f, compatOptions0066(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.seenSecret) != 1 {
		t.Errorf("secrets asked %q; want one", f.seenSecret)
	}
	if res.Kind != CredAPIKey || res.APIKey != "lm-key" || res.Model != "local-model-00" {
		t.Errorf("Kind %q, key set %v, Model %q; want api_key, true, local-model-00", res.Kind, res.APIKey == "lm-key", res.Model)
	}
	if got, want := srv.seen(), []string{"", "Bearer lm-key"}; !slices.Equal(got, want) {
		t.Errorf("listing Authorization headers %q; want %q", got, want)
	}
	if !noticeContaining(f.seenNotify, "LM Studio refused the request without a key") {
		t.Errorf("notices %q; want the refusal without a key", f.seenNotify)
	}
}

// TestCompat_ThreeKeysRefused_0066: the first key asked is the first typed;
// three refused end the run, as typed keys do (0028-MADR D-H10).
func TestCompat_ThreeKeysRefused_0066(t *testing.T) {
	srv := newCompatServer0066(t, 2, "lm-key")
	f := newFake(t, fakePrompter{inputs: []string{srv.URL + "/v1"}, secrets: []string{"wrong-1", "wrong-2", "wrong-3"}})
	_, _, err := ResolveCredential(context.Background(), f, compatOptions0066(t), compatID0066)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || err == nil || !strings.Contains(err.Error(), "LM Studio refused 3 keys") {
		t.Errorf("err = %v; want ErrAuthFailure saying \"LM Studio refused 3 keys\"", err)
	}
	if n := len(srv.seen()); n != 4 {
		t.Errorf("%d listing requests; want 4, none then three keys", n)
	}
}

// TestCompat_SavedKeyOffered_0066: Existing's key is offered to keep, as a
// required key's is; a kept key the server refuses ends the run, naming it.
func TestCompat_SavedKeyOffered_0066(t *testing.T) {
	for _, c := range []struct {
		name, saved, wantErr string
	}{
		{"kept", "lm-key", ""},
		{"refused", "old-key", "LM Studio refused the saved key"},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv := newCompatServer0066(t, 2, "lm-key")
			f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, confirms: []bool{true}})
			o := compatOptions0066(t)
			o.Existing = Result{Provider: compatID0066, Kind: CredAPIKey, APIKey: c.saved, BaseURL: srv.URL + "/v1"}
			res, _, err := ResolveCredential(context.Background(), f, o, compatID0066)
			if len(f.seenConfirm) != 1 || !strings.HasPrefix(f.seenConfirm[0], "Keep the existing key (") {
				t.Errorf("confirms %q; want the keep question", f.seenConfirm)
			}
			if got, want := srv.seen(), []string{"Bearer " + c.saved}; !slices.Equal(got, want) {
				t.Errorf("listing Authorization headers %q; want %q", got, want)
			}
			if c.wantErr != "" {
				if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), c.wantErr) {
					t.Errorf("err = %v; want ErrAuthFailure saying %q", err, c.wantErr)
				}
				return
			}
			if err != nil || res.Kind != CredAPIKey || res.APIKey != c.saved {
				t.Errorf("Kind %q, key kept %v, err %v; want api_key, true, nil", res.Kind, res.APIKey == c.saved, err)
			}
		})
	}
}

// TestOllama_RefusedNoKeyAsked_0066: a local provider sends no key, so
// none is asked; its refusal is worded as a request that carried none.
func TestOllama_RefusedNoKeyAsked_0066(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			_, _ = io.WriteString(w, `{"version":"0.6.0"}`)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"unauthorized"}`)
	}))
	t.Cleanup(srv.Close)
	f := newFake(t, fakePrompter{inputs: []string{srv.URL}})
	_, _, err := ResolveCredential(context.Background(), f, Options{Discover: true,
		ProviderOptions: []llmprovider.Option{testCache(t), llmprovider.WithoutModelMetadata()}}, llmprovider.ProviderOllama)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || err == nil ||
		!strings.Contains(err.Error(), "Ollama (local) refused a request that carried no key") {
		t.Errorf("err = %v; want ErrAuthFailure saying the request carried no key", err)
	}
	if len(f.seenSecret) != 0 {
		t.Errorf("secrets asked %q; want none", f.seenSecret)
	}
}

// TestCompat_ListingFailureNoticed_0066: a Lister's failed listing, which
// its catalog carries with no error, is told as a failed ListModels is, and
// the user is asked for a model id (0066-MADR V4).
func TestCompat_ListingFailureNoticed_0066(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	f := newFake(t, fakePrompter{selects: []int{0}, inputs: []string{srv.URL + "/v1", "typed-model"}})
	res, err := ConfigureLLM(context.Background(), f, compatOptions0066(t))
	if err != nil || res.Model != "typed-model" {
		t.Fatalf("Model %q, err %v; want typed-model", res.Model, err)
	}
	if want := "could not list models for LM Studio (provider unavailable, HTTP 502)"; !slices.Contains(f.seenNotify, want) {
		t.Errorf("notices %q; want %q", f.seenNotify, want)
	}
}

// TestCompat_ListedThroughLister_0066: the MADR's probe C. The wizard lists
// a Chat Completions server through catalog.Lister, so its catalog is the
// one catalog.List builds with chatcompat.WithID: live, with its whole menu.
// List takes no WithLimit for such an id, so the wizard cuts Recommended to
// ModelLimit itself (0066-MADR, amendment "the Lister's options").
func TestCompat_ListedThroughLister_0066(t *testing.T) {
	srv := newCompatServer0066(t, 30, "")
	f := newFake(t, fakePrompter{inputs: []string{srv.URL + "/v1"}})
	o := compatOptions0066(t)
	o.ModelLimit = 6
	_, cat, err := ResolveCredential(context.Background(), f, o, compatID0066)
	if err != nil {
		t.Fatal(err)
	}
	want, err := catalog.List(context.Background(), compatID0066, llmprovider.NewStaticToken(""), testCache(t),
		chatcompat.WithID(compatID0066), llmprovider.WithBaseURL(srv.URL+"/v1"), llmprovider.WithoutModelMetadata())
	if err != nil {
		t.Fatal(err)
	}
	if cat.Source != catalog.ListLive || len(cat.Menu) != 30 || len(cat.Recommended) != 6 || len(cat.Usable) != 30 {
		t.Errorf("Source %q, Menu %d, Recommended %d, Usable %d; want live, 30, 6, 30",
			cat.Source, len(cat.Menu), len(cat.Recommended), len(cat.Usable))
	}
	if !slices.Equal(cat.Recommended, want.Recommended[:6]) || !slices.Equal(cat.Usable, want.Usable) ||
		!slices.EqualFunc(cat.Menu, want.Menu, func(a, b catalog.MenuEntry) bool { return a.ID == b.ID && a.Rank == b.Rank }) {
		t.Errorf("catalog %q / %q; want List's, %q / %q", cat.Recommended, cat.Usable, want.Recommended, want.Usable)
	}
	if n := len(cat.Search("local-model-2")); n != 10 {
		t.Errorf("search local-model-2 found %d; want 10", n)
	}
}
