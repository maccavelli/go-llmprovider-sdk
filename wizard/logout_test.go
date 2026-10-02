package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// revokeServer answers OpenAI's revocation endpoint with status, and counts
// the tokens it was sent.
func revokeServer(t *testing.T, status int) (*httptest.Server, *[]string) {
	t.Helper()
	var revoked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/revoke" {
			t.Errorf("request %s %s, want POST /oauth/revoke", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode revoke body: %v", err)
		}
		revoked = append(revoked, body["token"])
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, &revoked
}

// storedOpenAISession saves an OpenAI session whose revocation goes to srv.
func storedOpenAISession(t *testing.T, store *memoryTokenStore, srv *httptest.Server) {
	t.Helper()
	s := testOAuthSession(llmprovider.ProviderOpenAI)
	s.Refresh, s.TokenURL = "rt-logout", srv.URL+"/oauth/token"
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, s); err != nil {
		t.Fatal(err)
	}
}

// TestLogout_RevokesThenDeletes (0016-MADR D11, A8): logout revokes the
// stored session, then deletes it; a failed revocation is reported and the
// local copy is deleted anyway.
func TestLogout_RevokesThenDeletes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		notice string
	}{
		{"revoked", http.StatusOK, "Logged out of OpenAI"},
		{"revocation fails", http.StatusInternalServerError, "could not revoke"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, revoked := revokeServer(t, tc.status)
			store := newMemoryTokenStore()
			storedOpenAISession(t, store, srv)
			f := &fakePrompter{t: t, confirms: []bool{true}}
			if err := Logout(context.Background(), f, Options{TokenStore: store}, llmprovider.ProviderOpenAI); err != nil {
				t.Fatalf("Logout() error = %v", err)
			}
			if len(*revoked) != 1 || (*revoked)[0] != "rt-logout" {
				t.Errorf("revoked %q, want the refresh token once", *revoked)
			}
			if _, ok := store.sessions[llmprovider.ProviderOpenAI]; ok {
				t.Error("the session is still stored")
			}
			if countContaining(f.seenNotify, tc.notice) != 1 {
				t.Errorf("notices %q, want one containing %q", f.seenNotify, tc.notice)
			}
			assertTextMasksSecret(t, f.allText, "oauth-access")
		})
	}
}

// TestLogout_NoRevocationForKilo: Kilo's login has no revocation endpoint, so
// logout says so and deletes the local copy.
func TestLogout_NoRevocationForKilo(t *testing.T) {
	store := newMemoryTokenStore()
	if err := store.Save(context.Background(), llmprovider.ProviderKilo,
		&llmprovider.OAuthSession{Provider: llmprovider.ProviderKilo, Access: kiloWizardToken}); err != nil {
		t.Fatal(err)
	}
	f := &fakePrompter{t: t, confirms: []bool{true}}
	if err := Logout(context.Background(), f, Options{TokenStore: store}, llmprovider.ProviderKilo); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, ok := store.sessions[llmprovider.ProviderKilo]; ok {
		t.Error("the session is still stored")
	}
	if countContaining(f.seenNotify, "no revocation") != 1 {
		t.Errorf("notices %q, want the missing revocation said once", f.seenNotify)
	}
}

// TestLogout_LeavesTheStoreAlone covers the ways logout does nothing: the user
// declines, or there is no session.
func TestLogout_LeavesTheStoreAlone(t *testing.T) {
	srv, revoked := revokeServer(t, http.StatusOK)
	store := newMemoryTokenStore()
	storedOpenAISession(t, store, srv)
	f := &fakePrompter{t: t, confirms: []bool{false}}
	if err := Logout(context.Background(), f, Options{TokenStore: store}, llmprovider.ProviderOpenAI); err != nil {
		t.Fatalf("declined: Logout() error = %v", err)
	}
	if len(*revoked) != 0 || store.sessions[llmprovider.ProviderOpenAI] == nil {
		t.Errorf("declined: revoked %q, stored %v; want nothing changed", *revoked, store.sessions)
	}

	f = &fakePrompter{t: t}
	if err := Logout(context.Background(), f, Options{TokenStore: store}, llmprovider.ProviderGrok); err != nil {
		t.Fatalf("no session: Logout() error = %v", err)
	}
	if len(f.seenConfirm) != 0 || countContaining(f.seenNotify, "No saved") != 1 {
		t.Errorf("no session: confirms %q, notices %q; want only a notice", f.seenConfirm, f.seenNotify)
	}
}

// deleteFailingStore is a store whose Delete fails.
type deleteFailingStore struct{ *memoryTokenStore }

func (deleteFailingStore) Delete(context.Context, llmprovider.ProviderID) error {
	return errors.New("disk on fire")
}

// TestLogout_Errors: logout needs a store, and reports a failed delete.
func TestLogout_Errors(t *testing.T) {
	if err := Logout(context.Background(), &fakePrompter{t: t}, Options{}, llmprovider.ProviderOpenAI); err == nil ||
		!strings.Contains(err.Error(), "TokenStore") {
		t.Errorf("no store: Logout() error = %v, want it named", err)
	}
	srv, _ := revokeServer(t, http.StatusOK)
	store := newMemoryTokenStore()
	storedOpenAISession(t, store, srv)
	err := Logout(context.Background(), &fakePrompter{t: t, confirms: []bool{true}},
		Options{TokenStore: deleteFailingStore{store}}, llmprovider.ProviderOpenAI)
	if err == nil || !strings.Contains(err.Error(), "disk on fire") {
		t.Errorf("failed delete: Logout() error = %v, want it returned", err)
	}
}
