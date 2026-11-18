package wizard

// 0082-PLAN Phase 1's second pass (the amendment of 2026-11-17): the wizard's
// save of a new sign-in takes the store's refresh lock when the store has one,
// and the plan session it signs in to again is handed over only when it is the
// same sign-in as the one being started. Every issuer is in process.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// lockEvents0082 is a TokenStore and RefreshLocker that records the order of
// its lock, save and unlock, and can make its lock wait for the context.
type lockEvents0082 struct {
	mu     sync.Mutex
	events []string
	hold   bool // LockRefresh waits for ctx, as a lock held by another process
	saved  *auth.OAuthSession
}

func (s *lockEvents0082) log(e string) {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
}

func (s *lockEvents0082) got() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.events...)
}

func (s *lockEvents0082) Load(context.Context, llmprovider.ProviderID) (*auth.OAuthSession, error) {
	return nil, nil
}

func (s *lockEvents0082) Save(_ context.Context, _ llmprovider.ProviderID, session *auth.OAuthSession) error {
	s.log("save")
	s.mu.Lock()
	s.saved = session
	s.mu.Unlock()
	return nil
}

func (s *lockEvents0082) Delete(context.Context, llmprovider.ProviderID) error { return nil }

func (s *lockEvents0082) LockRefresh(ctx context.Context, _ llmprovider.ProviderID) (func(), error) {
	if s.hold {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	s.log("lock")
	return func() { s.log("unlock") }, nil
}

// plainStore0082 is a TokenStore that is not a RefreshLocker.
type plainStore0082 struct{ lockEvents0082 }

// LockRefresh hides the embedded method: plainStore0082 is no RefreshLocker.
func (*plainStore0082) LockRefresh() {}

func newSession0082() *auth.OAuthSession {
	return &auth.OAuthSession{TokenURL: "https://issuer.invalid/never", Provider: llmprovider.ProviderOpenAI, Kind: auth.SessionChatGPT, Access: "new-access",
		Refresh: "new-refresh", Expiry: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer,
		ClientID: auth.DefaultOpenAIClientID, AccountID: "acct-new"}
}

func descriptor0082(t *testing.T) llmprovider.Descriptor {
	t.Helper()
	d, ok := (Options{}).registry().Descriptor(llmprovider.ProviderOpenAI)
	if !ok {
		t.Fatal("no OpenAI descriptor")
	}
	return d
}

// R-1: the save of a new sign-in takes the store's refresh lock, saves inside
// it and releases it, so a refresh in flight, which holds the lock across its
// request and its own save, finishes first.
func TestSaveTakesTheRefreshLock_0082(t *testing.T) {
	store := &lockEvents0082{}
	c := resolvedCredential{session: newSession0082(), saveTo: store}
	if err := c.save(context.Background(), nil, descriptor0082(t)); err != nil {
		t.Fatal(err)
	}
	if got := store.got(); len(got) != 3 || got[0] != "lock" || got[1] != "save" || got[2] != "unlock" {
		t.Errorf("events %q; want lock, save, unlock", got)
	}
}

// R-1 (risk b): a wait for a lock another process holds ends with the context,
// which is cancellable: nothing is saved, and the error is the context's.
func TestSaveCancelledWhileWaitingSavesNothing_0082(t *testing.T) {
	store := &lockEvents0082{hold: true}
	c := resolvedCredential{session: newSession0082(), saveTo: store}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.save(ctx, nil, descriptor0082(t)) }()
	select {
	case err := <-done:
		t.Fatalf("save returned %v before the lock was available or the context ended", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("save = %v; want an error matching context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the wait for the lock did not end with the context")
	}
	if got := store.got(); len(got) != 0 {
		t.Errorf("events %q; want nothing saved", got)
	}
}

// R-1 (risk c): a store that is not a RefreshLocker is saved to as before.
func TestSaveWithoutALockerSaves_0082(t *testing.T) {
	store := &plainStore0082{}
	if _, ok := any(store).(auth.RefreshLocker); ok {
		t.Fatal("plainStore0082 is a RefreshLocker")
	}
	c := resolvedCredential{session: newSession0082(), saveTo: store}
	if err := c.save(context.Background(), nil, descriptor0082(t)); err != nil {
		t.Fatal(err)
	}
	if got := store.got(); len(got) != 1 || got[0] != "save" {
		t.Errorf("events %q; want one save and no lock", got)
	}
}

// R-1 (risk d): a refresh in flight in the same process, through another
// session on the same FileTokenStore, holds the lock across its request; the
// save of a new sign-in waits for it, then saves, and nothing deadlocks.
func TestSaveWaitsForARefreshInFlightInOneProcess_0082(t *testing.T) {
	release := make(chan struct{})
	received := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case received <- struct{}{}:
		default:
		}
		<-release
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh-access", "refresh_token": "fresh-refresh", "expires_in": 3600})
	}))
	defer srv.Close()
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	ctx := context.Background()
	fs := newClientStore(t)
	old := auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: auth.SessionChatGPT, Access: "old-access", Refresh: "old-refresh",
		Expiry: time.Now().Add(-time.Minute), Issuer: auth.DefaultOpenAIIssuer, ClientID: "app_same", AccountID: "acct-old",
		TokenURL: srv.URL + "/token"}
	if err := fs.Save(ctx, llmprovider.ProviderOpenAI, &old); err != nil {
		t.Fatal(err)
	}
	refresher, err := fs.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || refresher == nil {
		t.Fatalf("Load = %v, %v", refresher, err)
	}
	refresher.HTTPClient = srv.Client()
	go func() { _, _ = refresher.Token(ctx) }()
	select {
	case <-received:
	case <-time.After(5 * time.Second):
		t.Fatal("the refresh did not reach the issuer")
	}

	newer := newSession0082()
	newer.ClientID, newer.AccountID = "app_newer", "acct-newer"
	saved := make(chan error, 1)
	go func() { saved <- (resolvedCredential{session: newer, saveTo: fs}).save(ctx, nil, descriptor0082(t)) }()
	select {
	case err := <-saved:
		t.Fatalf("the save returned %v while a refresh held the lock", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-saved:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the save did not finish after the refresh did: a deadlock")
	}
	if err := refresher.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	rec, err := fs.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || rec == nil || rec.ClientID != "app_newer" {
		t.Errorf("the store holds %v, %v; want the new sign-in, saved after the refresh", rec, err)
	}
}

// R-3 (the review's S4): the plan session the wizard signs in to again is
// handed over as Previous only when it is the same sign-in as the one being
// started, so another application's plan record, and its issued client and
// host id, are not reused. With a client record (0075-MADR D7) the client is
// the record's; without one a stored plan session is the app's own, as before.
func TestPlanPreviousOnlyForTheSameSignIn_0082(t *testing.T) {
	ctx := context.Background()
	record := auth.ClientRecord{ClientID: "oaiapp_mine", HostID: "urn:uuid:oaiapp_mine"}
	cases := []struct {
		name         string
		session      *auth.OAuthSession
		withRecord   bool
		wantPrevious bool
	}{
		{"the app's own plan session, the record naming its client", planSessionFor("oaiapp_mine"), true, true},
		{"the app's own plan session, no client record", planSessionFor("oaiapp_mine"), false, true},
		{"another application's plan session, the record naming this app's client", planSessionFor("oaiapp_other"), true, false},
		{"a plan session of another issuer", func() *auth.OAuthSession {
			s := planSessionFor("oaiapp_mine")
			s.Issuer = "https://issuer.example"
			return s
		}(), true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := newClientStore(t)
			if c.withRecord {
				if err := store.SaveClient(ctx, llmprovider.ProviderOpenAI, record); err != nil {
					t.Fatal(err)
				}
			}
			if err := store.Save(ctx, llmprovider.ProviderOpenAI, c.session); err != nil {
				t.Fatal(err)
			}
			var given auth.ChatGPTPlanOptions
			stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
				given = opts
				return planSessionFor("oaiapp_mine"), nil
			})
			f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}, allowLeftover: true})
			if _, _, err := ResolveCredential(ctx, f, Options{TokenStore: store, AgentName: planTestApp, NoPastePrompt: true},
				llmprovider.ProviderOpenAI); err != nil {
				t.Fatal(err)
			}
			if got := given.Previous != nil; got != c.wantPrevious {
				t.Errorf("Previous = %v; want a hand-over: %v", given.Previous, c.wantPrevious)
			}
			if c.wantPrevious && given.Previous.ClientID != c.session.ClientID {
				t.Errorf("Previous client %q, want the stored session's", given.Previous.ClientID)
			}
			if !c.wantPrevious && c.withRecord && given.ClientID != record.ClientID {
				t.Errorf("a refused Previous left client %q; want the record's %q reused", given.ClientID, record.ClientID)
			}
		})
	}
}

// R-3: a stored plan session that names another provider (a store that does
// not stamp the provider it was asked for) is not handed over either.
func TestPlanPreviousNotFromAnotherProvider_0082(t *testing.T) {
	store := newMemoryTokenStore()
	other := planSessionFor("oaiapp_mine")
	other.Provider, other.Kind = llmprovider.ProviderGrok, auth.SessionChatGPTPlan
	store.sessions[llmprovider.ProviderOpenAI] = other
	var given auth.ChatGPTPlanOptions
	stubPlanLogin(t, func(_ context.Context, opts auth.ChatGPTPlanOptions) (*auth.OAuthSession, error) {
		given = opts
		return planSessionFor("oaiapp_mine"), nil
	})
	f := newFake(t, fakePrompter{selects: []int{planMethodIdx(t), 0}, allowLeftover: true})
	if _, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: store, AgentName: planTestApp, NoPastePrompt: true},
		llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if given.Previous != nil {
		t.Errorf("Previous = %v; want none for another provider's session", given.Previous)
	}
}
