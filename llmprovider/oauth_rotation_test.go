package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// siblingRotates answers a refresh of rt-old by first saving, as another
// process would, a session rotated to rt-sibling (expiring at siblingExpiry),
// then rejecting rt-old as reused. A refresh of rt-sibling succeeds.
func siblingRotates(t *testing.T, store *FileTokenStore, siblingExpiry time.Time) (*http.Request, func() []string) {
	t.Helper()
	var sent []string
	var tokenURL string // the server's own URL, so the sibling never refreshes against a real issuer
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode refresh: %v", err)
		}
		sent = append(sent, body[oauthRefreshToken])
		switch body[oauthRefreshToken] {
		case "rt-old":
			sibling := testSession("a-sibling", "rt-sibling", siblingExpiry)
			sibling.TokenURL = tokenURL
			if err := store.Save(context.Background(), ProviderOpenAI, sibling); err != nil {
				t.Errorf("sibling save: %v", err)
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"refresh_token_reused"}`))
		default:
			refreshOK(w, "a-refreshed")
		}
	})
	tokenURL = srv.URL
	req, _ := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	return req, func() []string { return sent }
}

// TestOAuthRefresh_RejectedAdoptsSiblingRotation (0016-MADR amendment A1): a
// refresh rejected because a sibling already rotated the token re-reads the
// store and adopts the sibling's session instead of failing.
func TestOAuthRefresh_RejectedAdoptsSiblingRotation(t *testing.T) {
	store, session, _ := sharedStoreSessions(t, "")
	req, sent := siblingRotates(t, store, time.Now().Add(time.Hour))
	session.TokenURL = req.URL.String()

	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "a-sibling" || session.Refresh != "rt-sibling" {
		t.Fatalf("Token = %q, %v (refresh %q); want the sibling's session adopted", tok.Value, err, session.Refresh)
	}
	if got := sent(); len(got) != 1 {
		t.Errorf("refreshes sent %v, want only the rejected one", got)
	}
}

// TestOAuthRefresh_RejectedRefreshesWithSiblingToken: when the sibling's
// session is itself due, it is refreshed once with the sibling's token.
func TestOAuthRefresh_RejectedRefreshesWithSiblingToken(t *testing.T) {
	store, session, _ := sharedStoreSessions(t, "")
	req, sent := siblingRotates(t, store, time.Now().Add(-time.Minute))
	session.TokenURL = req.URL.String()

	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "a-refreshed" {
		t.Fatalf("Token = %q, %v; want a refresh with the sibling's token", tok.Value, err)
	}
	if got := strings.Join(sent(), ","); got != "rt-old,rt-sibling" {
		t.Errorf("refreshes sent %s, want rt-old,rt-sibling", got)
	}
}

// TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer: a save retried after
// a failure is dropped, not written, when another process has since saved a
// newer session.
func TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer(t *testing.T) {
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, session, _ := sharedStoreSessions(t, srv.URL)
	var logged bytes.Buffer
	session.Logger = slog.New(slog.NewTextHandler(&logged, nil))

	tokenStoreRename = func(string, string) error { return errors.New("planted rename failure") }
	if _, err := session.Token(context.Background()); err != nil {
		t.Fatalf("Token with a failing save: %v", err)
	}
	tokenStoreRename = os.Rename
	t.Cleanup(func() { tokenStoreRename = os.Rename })

	newer := testSession("a-newer", "rt-newer", time.Now().Add(time.Hour))
	if err := store.Save(context.Background(), ProviderOpenAI, newer); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Token(context.Background()); err != nil {
		t.Fatalf("Token retrying the save: %v", err)
	}
	stored, err := store.Load(context.Background(), ProviderOpenAI)
	if err != nil || stored.Refresh != "rt-newer" {
		t.Fatalf("store holds %+v, %v; the newer session must not be overwritten", stored, err)
	}
	if !strings.Contains(logged.String(), "newer session") {
		t.Errorf("log = %q, want the dropped retry reported", logged.String())
	}
	saves := logged.Len()
	if _, err := session.Token(context.Background()); err != nil || logged.Len() != saves {
		t.Errorf("the dropped retry was attempted again (log grew to %q), err %v", logged.String(), err)
	}
}
