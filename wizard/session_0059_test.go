package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// 0059-MADR W2 and its amendment "a refresh can outlast the listing (Phase
// 2)": a new session is saved once the listing has not refused it, after any
// refresh the listing left running.

// saved0059 is what a save held when it was made, as a file store keeps it,
// not the session the wizard goes on changing.
type saved0059 struct {
	access, refresh string
	store           auth.TokenStore
}

// store0059 holds a session for Load and records each save.
type store0059 struct {
	mu    sync.Mutex
	held  map[llmprovider.ProviderID]*auth.OAuthSession
	saves []saved0059
}

func newStore0059() *store0059 {
	return &store0059{held: map[llmprovider.ProviderID]*auth.OAuthSession{}}
}

func (s *store0059) Load(_ context.Context, provider llmprovider.ProviderID) (*auth.OAuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.held[provider], nil
}

func (s *store0059) Save(_ context.Context, provider llmprovider.ProviderID, session *auth.OAuthSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saves = append(s.saves, saved0059{session.Access, session.Refresh, session.Store})
	s.held[provider] = &auth.OAuthSession{Provider: provider, Access: session.Access, Refresh: session.Refresh}
	return nil
}

func (s *store0059) Delete(_ context.Context, provider llmprovider.ProviderID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, provider)
	return nil
}

func (s *store0059) recorded() []saved0059 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]saved0059(nil), s.saves...)
}

// answer is a JSON response with status.
func answer0059(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status),
		Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}

// client0059 lists with listStatus, and answers the token endpoint once
// release is closed (at once when it is nil), with tokenStatus: rotated
// tokens on 200, invalid_grant otherwise.
func client0059(listStatus, tokenStatus int, release <-chan struct{}) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			if release != nil {
				<-release
			}
			if tokenStatus != http.StatusOK {
				return answer0059(r, tokenStatus, `{"error":"invalid_grant"}`), nil
			}
			return answer0059(r, http.StatusOK, `{"access_token":"at-new","refresh_token":"rt-new","expires_in":3600}`), nil
		}
		if listStatus != http.StatusOK {
			return answer0059(r, listStatus, `{"error":{"message":"invalid credentials"}}`), nil
		}
		return answer0059(r, http.StatusOK, `{"data":[{"id":"grok-4.5"}]}`), nil
	})}
}

// browserGrok0059 stubs the browser sign-in to return a Grok session that
// refreshes through client, expired when expired is set, and returns it.
func browserGrok0059(t *testing.T, client *http.Client, expired bool) **auth.OAuthSession {
	t.Helper()
	var session *auth.OAuthSession
	original := loginBrowserOAuth
	t.Cleanup(func() { loginBrowserOAuth = original })
	loginBrowserOAuth = func(_ context.Context, provider llmprovider.ProviderID, _ auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		session = testOAuthSession(provider)
		session.TokenURL, session.HTTPClient = "https://issuer.test/oauth/token", client
		if expired {
			session.Expiry = time.Now().Add(-time.Minute)
		}
		return session, nil
	}
	return &session
}

// grok0059 runs Grok's browser sign-in (the menu's second method).
func grok0059(t *testing.T, store auth.TokenStore, client *http.Client, discover bool, limit time.Duration) (*fakePrompter, error) {
	t.Helper()
	f := newFake(t, fakePrompter{selects: []int{1}})
	_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, Discover: discover,
		DiscoverLimit: limit, HTTPClient: client,
		ProviderOptions: []llmprovider.Option{llmprovider.WithoutModelMetadata(), testCache(t)}}, llmprovider.ProviderGrok)
	return f, err
}

// TestSession_RefusedKeepsSaved_0059 (P8): a new sign-in the listing refuses
// leaves the store as it was, unwritten: the previous session, or none. The
// refusal's refresh is not saved either.
func TestSession_RefusedKeepsSaved_0059(t *testing.T) {
	for name, previous := range map[string]*auth.OAuthSession{
		"a previous session": {Provider: llmprovider.ProviderGrok, Access: "prev-access", Refresh: "prev-refresh"},
		"an empty store":     nil,
	} {
		t.Run(name, func(t *testing.T) {
			store := newStore0059()
			if previous != nil {
				store.held[llmprovider.ProviderGrok] = previous
			}
			client := client0059(http.StatusUnauthorized, http.StatusOK, nil)
			browserGrok0059(t, client, false)
			_, err := grok0059(t, store, client, true, 0)
			if !errors.Is(err, llmprovider.ErrAuthFailure) {
				t.Fatalf("err %v; want the refusal", err)
			}
			if saves, held := store.recorded(), store.held[llmprovider.ProviderGrok]; len(saves) != 0 || held != previous {
				t.Errorf("saves %+v, the store holds %v; want none, and what it held before", saves, held)
			}
		})
	}
}

// TestSession_AcceptedSavedOnce_0059: a sign-in the listing accepts is saved
// once, with its Store set.
func TestSession_AcceptedSavedOnce_0059(t *testing.T) {
	store := newStore0059()
	client := client0059(http.StatusOK, http.StatusOK, nil)
	browserGrok0059(t, client, false)
	if _, err := grok0059(t, store, client, true, 0); err != nil {
		t.Fatal(err)
	}
	if saves := store.recorded(); len(saves) != 1 || saves[0].access != "oauth-access" || saves[0].store != store {
		t.Errorf("saves %+v; want one, of the new session, with its Store set", saves)
	}
}

// TestSession_NoDiscoverSaved_0059: with no listing, the sign-in is saved once.
func TestSession_NoDiscoverSaved_0059(t *testing.T) {
	store := newStore0059()
	client := client0059(http.StatusUnauthorized, http.StatusOK, nil)
	browserGrok0059(t, client, false)
	if _, err := grok0059(t, store, client, false, 0); err != nil {
		t.Fatal(err)
	}
	if saves := store.recorded(); len(saves) != 1 || saves[0].store != store {
		t.Errorf("saves %+v; want one, with its Store set", saves)
	}
}

// TestSession_RotatedDuringCheckSaved_0059: a session that refreshes during
// the listing is saved once, with the rotated tokens.
func TestSession_RotatedDuringCheckSaved_0059(t *testing.T) {
	store := newStore0059()
	client := client0059(http.StatusOK, http.StatusOK, nil)
	browserGrok0059(t, client, true)
	if _, err := grok0059(t, store, client, true, 0); err != nil {
		t.Fatal(err)
	}
	if saves := store.recorded(); len(saves) != 1 || saves[0].access != "at-new" || saves[0].refresh != "rt-new" {
		t.Errorf("saves %+v; want one, of the rotated tokens", saves)
	}
}

// TestSession_KiloRefusedNotSaved_0059: Kilo's device sign-in, its listing
// refused, saves nothing.
func TestSession_KiloRefusedNotSaved_0059(t *testing.T) {
	stubKilo(t, auth.KiloAccount{HasPersonalAccount: true}, nil)
	store := newStore0059()
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
	_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, Discover: true,
		HTTPClient:      client0059(http.StatusUnauthorized, http.StatusOK, nil),
		ProviderOptions: []llmprovider.Option{llmprovider.WithoutModelMetadata(), testCache(t)}}, llmprovider.ProviderKilo)
	if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Fatalf("err %v; want the refusal", err)
	}
	if saves := store.recorded(); len(saves) != 0 {
		t.Errorf("saves %+v; want none", saves)
	}
}

// TestSession_RefreshAfterLimitSaved_0059: a refresh that answers after the
// listing's limit is waited for, and the session saved once, with the
// rotated refresh token, not the one the refresh spent.
func TestSession_RefreshAfterLimitSaved_0059(t *testing.T) {
	release := make(chan struct{})
	after := time.AfterFunc(400*time.Millisecond, func() { close(release) })
	t.Cleanup(func() { after.Stop() })
	store := newStore0059()
	client := client0059(http.StatusOK, http.StatusOK, release)
	browserGrok0059(t, client, true)
	f, err := grok0059(t, store, client, true, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if countContaining(f.seenNotify, "timed out") != 1 {
		t.Errorf("notices %q; want the listing to have run out of time", f.seenNotify)
	}
	if saves := store.recorded(); len(saves) != 1 || saves[0].refresh != "rt-new" {
		t.Errorf("saves %+v; want one, of the rotated refresh token", saves)
	}
}

// TestSession_RefreshRefusedNotSaved_0059: a refresh the issuer refuses
// after the listing's limit saves nothing and ends the run.
func TestSession_RefreshRefusedNotSaved_0059(t *testing.T) {
	release := make(chan struct{})
	after := time.AfterFunc(400*time.Millisecond, func() { close(release) })
	t.Cleanup(func() { after.Stop() })
	store := newStore0059()
	client := client0059(http.StatusOK, http.StatusBadRequest, release)
	browserGrok0059(t, client, true)
	_, err := grok0059(t, store, client, true, 100*time.Millisecond)
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "was not saved") {
		t.Fatalf("err %v; want one matching ErrAuthFailure, saying the session was not saved", err)
	}
	if saves := store.recorded(); len(saves) != 0 {
		t.Errorf("saves %+v; want none", saves)
	}
}
