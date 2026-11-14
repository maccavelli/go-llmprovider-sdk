package catalog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// The catalog hands WithClientInfo to a session before it reads the session's
// token, in a listing and in the lookups (0079-MADR D4 and its amendment
// "the catalog lookups take D4's hand-off"; 0079-PLAN Phase 3, steps 3 and
// 3a).

// issuer0079 is a fake issuer: it answers every request as a refresh, and
// keeps the User-Agent of each.
type issuer0079 struct {
	*httptest.Server
	mu     sync.Mutex
	agents []string
}

func newIssuer0079(t *testing.T) *issuer0079 {
	t.Helper()
	i := &issuer0079{}
	i.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i.mu.Lock()
		i.agents = append(i.agents, r.Header.Get("User-Agent"))
		i.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
	}))
	t.Cleanup(i.Close)
	return i
}

func (i *issuer0079) seen() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]string(nil), i.agents...)
}

// storedGrok0079 is a Grok session saved by a FileTokenStore and loaded again,
// so that it has no identity and no client, expiring at expiry and refreshing
// at iss.
func storedGrok0079(t *testing.T, iss *issuer0079, expiry time.Time) *auth.OAuthSession {
	t.Helper()
	store, err := auth.NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	saved := &auth.OAuthSession{
		Provider: llmprovider.ProviderGrok,
		Issuer:   auth.DefaultGrokOAuthIssuer,
		ClientID: auth.DefaultGrokOAuthClientID,
		Access:   "old-access",
		Refresh:  "old-refresh",
		Expiry:   expiry,
		TokenURL: iss.URL + "/oauth/token",
	}
	if err := store.Save(t.Context(), llmprovider.ProviderGrok, saved); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(t.Context(), llmprovider.ProviderGrok)
	if err != nil || loaded == nil {
		t.Fatalf("Load() = %v, %v", loaded, err)
	}
	return loaded
}

// wantAgent0079 fails unless iss saw exactly one request, a refresh, and its
// User-Agent is the application's, or this module's when named is false.
func wantAgent0079(t *testing.T, label string, iss *issuer0079, named bool) {
	t.Helper()
	agents := iss.seen()
	if len(agents) != 1 {
		t.Errorf("%s: the issuer saw %d requests %q, want one refresh", label, len(agents), agents)
		return
	}
	got := agents[0]
	if named {
		if !strings.HasPrefix(got, "probe-app/1.2.3 (") || !strings.Contains(got, "go-llmprovider-sdk/") {
			t.Errorf("%s: refresh User-Agent = %q, want it to begin %q and name this module", label, got, "probe-app/1.2.3 (")
		}
		return
	}
	if !strings.HasPrefix(got, "go-llmprovider-sdk/") || strings.Contains(got, "probe-app") {
		t.Errorf("%s: refresh User-Agent = %q, want this module's", label, got)
	}
}

func named0079(named bool) []llmprovider.Option {
	if !named {
		return nil
	}
	return []llmprovider.Option{llmprovider.WithClientInfo("probe-app", "1.2.3")}
}

// TestListingNamesSession_0079: a listing names the application on the
// refresh its token read causes.
func TestListingNamesSession_0079(t *testing.T) {
	for _, named := range []bool{true, false} {
		t.Run(fmt.Sprintf("named=%t", named), func(t *testing.T) {
			resetLookupCaches(t)
			s := newLookupServer(t, lookupDoc, 200, grokListing0033)
			iss := newIssuer0079(t)
			// Within its refresh window, not past its expiry.
			session := storedGrok0079(t, iss, time.Now().Add(10*time.Second))
			opts := append(lookupOpts(s), named0079(named)...)
			if _, err := ListRaw(context.Background(), llmprovider.ProviderGrok, session, opts...); err != nil {
				t.Fatalf("ListRaw: %v", err)
			}
			wantAgent0079(t, "ListRaw", iss, named)
		})
	}
}

// TestLookupNamesSession_0079: the lookups name the application on the
// refresh they start, which is the first request at the issuer.
func TestLookupNamesSession_0079(t *testing.T) {
	ctx := context.Background()
	for _, named := range []bool{true, false} {
		t.Run(fmt.Sprintf("named=%t", named), func(t *testing.T) {
			t.Run("CachedModel", func(t *testing.T) {
				resetLookupCaches(t)
				s := newLookupServer(t, lookupDoc, 200, grokListing0033)
				iss := newIssuer0079(t)
				session := storedGrok0079(t, iss, time.Now().Add(-time.Minute))
				CachedModel(ctx, llmprovider.ProviderGrok, session, "grok-x", append(lookupOpts(s), named0079(named)...)...)
				// The background listing, waited for, sends the same.
				waitListingFetches()
				wantAgent0079(t, "CachedModel", iss, named)
			})
			t.Run("LookupModel", func(t *testing.T) {
				resetLookupCaches(t)
				s := newLookupServer(t, lookupDoc, 200, grokListing0033)
				iss := newIssuer0079(t)
				session := storedGrok0079(t, iss, time.Now().Add(-time.Minute))
				if _, err := LookupModel(ctx, llmprovider.ProviderGrok, session, "grok-x",
					append(lookupOpts(s), named0079(named)...)...); err != nil {
					t.Fatalf("LookupModel: %v", err)
				}
				waitListingFetches()
				wantAgent0079(t, "LookupModel", iss, named)
			})
			t.Run("CachedFacts", func(t *testing.T) {
				resetLookupCaches(t)
				s := newLookupServer(t, lookupDoc, 200, grokListing0033)
				iss := newIssuer0079(t)
				session := storedGrok0079(t, iss, time.Now().Add(-time.Minute))
				CachedFacts(ctx, llmprovider.ProviderGrok, "grok-x",
					append(append(lookupOpts(s), llmprovider.WithTokenSource(session)), named0079(named)...)...)
				wantAgent0079(t, "CachedFacts", iss, named)
			})
		})
	}
	t.Run("CachedFacts with a static key after the session", func(t *testing.T) {
		resetLookupCaches(t)
		s := newLookupServer(t, lookupDoc, 200, grokListing0033)
		iss := newIssuer0079(t)
		session := storedGrok0079(t, iss, time.Now().Add(-time.Minute))
		// As a provider passes it: its request's own token follows its options.
		CachedFacts(ctx, llmprovider.ProviderGrok, "grok-x", append(lookupOpts(s),
			llmprovider.WithTokenSource(session), llmprovider.WithClientInfo("probe-app", "1.2.3"), llmprovider.WithAPIKey("static"))...)
		if agents := iss.seen(); len(agents) != 0 {
			t.Errorf("the issuer saw %q, want no request: the static key is the credential", agents)
		}
	})
}

// countingSource is a token source that is no session: it counts its reads and
// fails when err is set.
type countingSource0079 struct {
	reads atomic.Int32
	err   error
}

func (c *countingSource0079) Token(context.Context) (llmprovider.Token, error) {
	c.reads.Add(1)
	if c.err != nil {
		return llmprovider.Token{}, c.err
	}
	return llmprovider.Token{Value: "static-token", Type: llmprovider.TokenBearer}, nil
}

// TestLookupHandOffOnlyNames_0079: the lookups' hand-off names the session and
// does nothing else: no client given, no change to an error's text, no more
// reads of the source's token (the amendment's L1, not L2).
func TestLookupHandOffOnlyNames_0079(t *testing.T) {
	ctx := context.Background()
	t.Run("no client handed", func(t *testing.T) {
		for _, lookup := range []string{"CachedModel", "LookupModel"} {
			resetLookupCaches(t)
			s := newLookupServer(t, lookupDoc, 200, grokListing0033)
			iss := newIssuer0079(t)
			session := storedGrok0079(t, iss, time.Now().Add(time.Hour))
			opts := append(lookupOpts(s), llmprovider.WithClientInfo("probe-app", "1.2.3"))
			// A listing fetch hands the session its client (listingToken), as
			// it did before 0079. The listing is cached first, under the
			// session's own token, so that the lookup fetches nothing and
			// only the hand-off is left to show.
			if _, err := LookupModel(ctx, llmprovider.ProviderGrok, llmprovider.NewStaticToken(session.Access), "grok-x", opts...); err != nil {
				t.Fatalf("warming the listing: %v", err)
			}
			waitListingFetches()
			warm := s.listings.Load()
			if lookup == "CachedModel" {
				CachedModel(ctx, llmprovider.ProviderGrok, session, "grok-x", opts...)
				waitListingFetches()
			} else if _, err := LookupModel(ctx, llmprovider.ProviderGrok, session, "grok-x", opts...); err != nil {
				t.Fatalf("LookupModel: %v", err)
			}
			if n := s.listings.Load(); n != warm {
				t.Fatalf("%s fetched a listing (%d requests, %d after warming); the cache was to answer it", lookup, n, warm)
			}
			if session.HTTPClient != nil {
				t.Errorf("%s gave the session a client: HTTPClient = %v, want nil", lookup, session.HTTPClient)
			}
		}
	})
	t.Run("LookupModel's error text", func(t *testing.T) {
		resetLookupCaches(t)
		s := newLookupServer(t, lookupDoc, 200, grokListing0033)
		boom := errors.New("token source is down")
		_, err := LookupModel(ctx, llmprovider.ProviderGrok, &countingSource0079{err: boom}, "grok-x", lookupOpts(s)...)
		if err == nil || !strings.HasSuffix(err.Error(), "model metadata: listing grok: acquire token: "+boom.Error()) ||
			strings.Contains(err.Error(), "model listing") {
			t.Errorf("LookupModel error = %v, want it to end %q", err, "model metadata: listing grok: acquire token: "+boom.Error())
		}
		if !errors.Is(err, boom) {
			t.Errorf("LookupModel error = %v, want the source's error wrapped", err)
		}
	})
	t.Run("token reads", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			call func(src llmprovider.TokenSource, opts []llmprovider.Option)
		}{
			{"CachedModel", func(src llmprovider.TokenSource, opts []llmprovider.Option) {
				CachedModel(ctx, llmprovider.ProviderGrok, src, "grok-x", opts...)
				waitListingFetches()
			}},
			{"LookupModel", func(src llmprovider.TokenSource, opts []llmprovider.Option) {
				_, _ = LookupModel(ctx, llmprovider.ProviderGrok, src, "grok-x", opts...)
				waitListingFetches()
			}},
		} {
			resetLookupCaches(t)
			s := newLookupServer(t, lookupDoc, 200, grokListing0033)
			src := &countingSource0079{}
			tc.call(src, append(lookupOpts(s), llmprovider.WithClientInfo("probe-app", "1.2.3")))
			// Each reads the token once for its own listing layer, and the
			// fetch it starts or makes reads it once more for the request.
			if got := src.reads.Load(); got != 2 {
				t.Errorf("%s read the source's token %d times, want 2, as at 5d08770", tc.name, got)
			}
		}
	})
}
