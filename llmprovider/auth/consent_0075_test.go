package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const testConsentID0075 = "chatgpt_plan.background_use"

// TestConsents_ParseAndFormat_0075 (0075-MADR D2): Consent reads granted,
// declined and unanswered ids; the entries a change writes are sorted by id;
// an entry this version does not understand is ignored.
func TestConsents_ParseAndFormat_0075(t *testing.T) {
	s := &OAuthSession{Consents: "a.b=granted c.d=declined e.f=maybe junk"}
	for _, tc := range []struct {
		id                string
		granted, answered bool
	}{
		{"a.b", true, true},
		{"c.d", false, true},
		{"e.f", false, false},
		{"junk", false, false},
		{"missing", false, false},
	} {
		if granted, answered := s.Consent(tc.id); granted != tc.granted || answered != tc.answered {
			t.Errorf("Consent(%q) = %t, %t; want %t, %t", tc.id, granted, answered, tc.granted, tc.answered)
		}
	}
	if granted, answered := (*OAuthSession)(nil).Consent("a.b"); granted || answered {
		t.Errorf("a nil session answered %t, %t", granted, answered)
	}
	if got, want := withConsent("z.z=granted a.b=declined e.f=maybe junk", "m.m", true), "a.b=declined m.m=granted z.z=granted"; got != want {
		t.Errorf("withConsent = %q, want %q (sorted, unknown entries dropped)", got, want)
	}
	if got, want := withConsent("a.b=granted", "a.b", false), "a.b=declined"; got != want {
		t.Errorf("withConsent changing an answer = %q, want %q", got, want)
	}
	text := (&OAuthSession{Provider: llmprovider.ProviderOpenAI, Consents: "a.b=granted"}).String()
	if !strings.Contains(text, `Consents:"a.b=granted"`) {
		t.Errorf("String = %s, want the consents shown", text)
	}
}

// TestConsents_StoreRoundTrip_0075 (0075-MADR D2): the answers round-trip
// through FileTokenStore as "consents"; a record without the field loads
// unanswered.
func TestConsents_StoreRoundTrip_0075(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	in := testSession("a-1", "rt-1", time.Now().Add(time.Hour))
	in.Consents = testConsentID0075 + "=granted"
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, in); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "openai.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil || rec["consents"] != in.Consents {
		t.Errorf("the record = %s (%v), want a \"consents\" member %q", raw, err, in.Consents)
	}
	out, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || out == nil {
		t.Fatalf("Load = %v, %v", out, err)
	}
	if granted, answered := out.Consent(testConsentID0075); !granted || !answered {
		t.Errorf("the loaded session answers %t, %t; want granted", granted, answered)
	}

	in.Consents = ""
	if err := store.Save(ctx, llmprovider.ProviderOpenAI, in); err != nil {
		t.Fatal(err)
	}
	if raw, err = os.ReadFile(filepath.Join(dir, "openai.json")); err != nil || strings.Contains(string(raw), "consents") {
		t.Errorf("a session with no answers wrote %s (%v); want no \"consents\" member", raw, err)
	}
	if out, err = store.Load(ctx, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if _, answered := out.Consent(testConsentID0075); answered {
		t.Error("a record without consents loads answered")
	}
}

// orderStore is a TokenStore and RefreshLocker that records the order of its
// calls.
type orderStore struct {
	events  []string
	held    *OAuthSession
	loadErr error
}

func (o *orderStore) Load(context.Context, llmprovider.ProviderID) (*OAuthSession, error) {
	o.events = append(o.events, "load")
	if o.loadErr != nil {
		return nil, o.loadErr
	}
	return o.held, nil
}

func (o *orderStore) Save(_ context.Context, _ llmprovider.ProviderID, s *OAuthSession) error {
	o.events = append(o.events, "save")
	o.held = s
	return nil
}

func (o *orderStore) Delete(context.Context, llmprovider.ProviderID) error { return nil }

func (o *orderStore) LockRefresh(context.Context, llmprovider.ProviderID) (func(), error) {
	o.events = append(o.events, "lock")
	return func() { o.events = append(o.events, "unlock") }, nil
}

// TestSetConsent_0075 (0075-MADR D3): SetConsent holds the refresh lock
// throughout; no session, an empty id and an id the format cannot hold are
// ErrInvalidRequest; the store's own error is returned as it is.
func TestSetConsent_0075(t *testing.T) {
	ctx := context.Background()
	store := &orderStore{held: testSession("a-1", "rt-1", time.Now().Add(time.Hour))}
	if err := SetConsent(ctx, store, llmprovider.ProviderOpenAI, testConsentID0075, true); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(store.events, ","), "lock,load,save,unlock"; got != want {
		t.Errorf("the calls were %s, want %s: the lock is held throughout", got, want)
	}
	if granted, answered := store.held.Consent(testConsentID0075); !granted || !answered {
		t.Errorf("the stored session answers %t, %t; want granted", granted, answered)
	}
	if err := SetConsent(ctx, store, llmprovider.ProviderOpenAI, testConsentID0075, false); err != nil {
		t.Fatal(err)
	}
	if got := store.held.Consents; got != testConsentID0075+"=declined" {
		t.Errorf("Consents = %q after a second call, want the answer changed", got)
	}

	empty := &orderStore{}
	for name, id := range map[string]string{"empty": "", "space": "a b", "equals": "a=b"} {
		if err := SetConsent(ctx, empty, llmprovider.ProviderOpenAI, id, true); !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("an %s id: err = %v, want ErrInvalidRequest", name, err)
		}
	}
	if err := SetConsent(ctx, empty, llmprovider.ProviderOpenAI, testConsentID0075, true); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("no stored session: err = %v, want ErrInvalidRequest", err)
	}
	if strings.Contains(strings.Join(empty.events, ","), "save") || empty.held != nil {
		t.Errorf("the calls were %v and held %v; a missing session must not be saved as a new one", empty.events, empty.held)
	}
	if err := SetConsent(ctx, nil, llmprovider.ProviderOpenAI, testConsentID0075, true); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a nil store: err = %v, want ErrInvalidRequest", err)
	}

	broken := errors.New("planted load failure")
	failing := &orderStore{loadErr: broken}
	if err := SetConsent(ctx, failing, llmprovider.ProviderOpenAI, testConsentID0075, true); !errors.Is(err, broken) || err.Error() != broken.Error() {
		t.Errorf("a failing Load: err = %v, want the store's error as it is", err)
	}
	file, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := SetConsent(ctx, file, "../bad", testConsentID0075, true); !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("a bad provider id: err = %v, want ErrInvalidProvider from the store", err)
	}
}

// consentRefreshSession loads the openai session of a fresh FileTokenStore,
// which holds a due session whose refresh the issuer answers, and returns it
// with a second handle on the same directory, as another process has.
func consentRefreshSession(t *testing.T) (session *OAuthSession, other *FileTokenStore) {
	t.Helper()
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	due := testSession("a-old", "rt-old", time.Now().Add(-time.Minute))
	due.TokenURL = srv.URL
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, due); err != nil {
		t.Fatal(err)
	}
	if session, err = store.Load(context.Background(), llmprovider.ProviderOpenAI); err != nil || session == nil {
		t.Fatalf("Load = %v, %v", session, err)
	}
	session.HTTPClient = srv.Client()
	return session, &FileTokenStore{Dir: store.Dir}
}

// requireStoredConsent fails unless the store holds the answer granted.
func requireStoredConsent(t *testing.T, stage string, store TokenStore) {
	t.Helper()
	s, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || s == nil {
		t.Fatalf("%s: Load = %v, %v", stage, s, err)
	}
	if granted, answered := s.Consent(testConsentID0075); !granted || !answered {
		t.Errorf("%s: the store holds %t, %t; want the answer kept", stage, granted, answered)
	}
}

// requireSessionConsent fails unless the live session reports granted.
func requireSessionConsent(t *testing.T, stage string, s *OAuthSession) {
	t.Helper()
	if granted, answered := s.Consent(testConsentID0075); !granted || !answered {
		t.Errorf("%s: the session reports %t, %t; want the answer", stage, granted, answered)
	}
}

// TestConsentSurvivesRefresh_0075 (0075-MADR D3, C16, C17): a SetConsent made
// through another handle, then the session's own refresh: the store still
// holds the answer and the live session reports it.
func TestConsentSurvivesRefresh_0075(t *testing.T) {
	ctx := context.Background()
	session, other := consentRefreshSession(t)
	if err := SetConsent(ctx, other, llmprovider.ProviderOpenAI, testConsentID0075, true); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Token(ctx); err != nil {
		t.Fatalf("Token: %v", err)
	}
	if err := session.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if session.Refresh != "rt-new" {
		t.Fatalf("the session's refresh token = %q; the test needs a refresh to have run", session.Refresh)
	}
	requireStoredConsent(t, "after the refresh", other)
	requireSessionConsent(t, "after the refresh", session)
}

// failingSaves makes every FileTokenStore rename fail until the returned func
// restores it.
func failingSaves(t *testing.T) (restore func()) {
	t.Helper()
	previous := tokenStoreRename
	t.Cleanup(func() { tokenStoreRename = previous })
	tokenStoreRename = func(string, string) error { return errors.New("planted rename failure") }
	return func() { tokenStoreRename = previous }
}

// rotateWithFailedSave refreshes the session while every save fails, so its
// rotation is unsaved, then restores saving and answers the consent through
// the other handle.
func rotateWithFailedSave(t *testing.T, session *OAuthSession, other *FileTokenStore) {
	t.Helper()
	ctx := context.Background()
	restore := failingSaves(t)
	if _, err := session.Token(ctx); err != nil {
		t.Fatalf("Token with a failing save: %v", err)
	}
	if err := session.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	restore()
	if err := SetConsent(ctx, other, llmprovider.ProviderOpenAI, testConsentID0075, true); err != nil {
		t.Fatal(err)
	}
}

// TestConsentSurvivesResave_0075 (0075-MADR D3, C18): a rotation whose save
// failed, a SetConsent through another handle, then the retried save: the
// store holds the rotated token and the answer.
func TestConsentSurvivesResave_0075(t *testing.T) {
	ctx := context.Background()
	session, other := consentRefreshSession(t)
	rotateWithFailedSave(t, session, other)
	if _, err := session.Token(ctx); err != nil { // retries the save
		t.Fatalf("Token retrying the save: %v", err)
	}
	if err := session.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if stored, err := other.Load(ctx, llmprovider.ProviderOpenAI); err != nil || stored.Refresh != "rt-new" {
		t.Fatalf("the store holds %v, %v; want the rotated token saved", stored, err)
	}
	requireStoredConsent(t, "after the re-save", other)
	requireSessionConsent(t, "after the re-save", session)
}

// TestConsentSurvivesUnsavedRotationRefresh_0075 (0075-MADR D3, C40): a
// rotation whose save failed, a SetConsent through another handle, then a
// refresh once the token is due: the record holds the token the unsaved
// rotation spent, and the store keeps the answer.
func TestConsentSurvivesUnsavedRotationRefresh_0075(t *testing.T) {
	ctx := context.Background()
	session, other := consentRefreshSession(t)
	rotateWithFailedSave(t, session, other)
	session.Invalidate() // the token is due again
	if _, err := session.Token(ctx); err != nil {
		t.Fatalf("Token refreshing again: %v", err)
	}
	if err := session.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	requireStoredConsent(t, "after the refresh", other)
	requireSessionConsent(t, "after the refresh", session)
}

// TestConsentAdoptedFromSibling_0075 (0075-MADR D3): a sibling's rotation
// adopted from the store brings its consents.
func TestConsentAdoptedFromSibling_0075(t *testing.T) {
	ctx := context.Background()
	session, other := consentRefreshSession(t)
	sibling := testSession("a-sibling", "rt-sibling", time.Now().Add(time.Hour))
	sibling.Consents = testConsentID0075 + "=declined"
	if err := other.Save(ctx, llmprovider.ProviderOpenAI, sibling); err != nil {
		t.Fatal(err)
	}
	token, err := session.Token(ctx)
	if err != nil || token.Value != "a-sibling" {
		t.Fatalf("Token = %q, %v; want the sibling's session adopted", token.Value, err)
	}
	if granted, answered := session.Consent(testConsentID0075); granted || !answered {
		t.Errorf("the adopted session reports %t, %t; want the sibling's declined", granted, answered)
	}
}
