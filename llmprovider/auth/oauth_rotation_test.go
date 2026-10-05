package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
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
			if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, sibling); err != nil {
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
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, newer); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Token(context.Background()); err != nil {
		t.Fatalf("Token retrying the save: %v", err)
	}
	stored, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
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

// TestOAuthRefresh_TolerantDecodeKeepsRotation (0021-MADR T4): a 200 whose
// expires_in is a string, a float or unreadable still yields the rotated
// refresh token, adopted and saved. An unreadable one takes the 3600 s
// default.
func TestOAuthRefresh_TolerantDecodeKeepsRotation(t *testing.T) {
	for _, c := range []struct{ name, expiresIn string }{
		{"a string", `"3600"`},
		{"a float", `3600.0`},
		{"unreadable", `"abc"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"access_token":"a-new","refresh_token":"rt-new","expires_in":`+c.expiresIn+`}`)
			})
			store, session, _ := sharedStoreSessions(t, srv.URL)
			before := time.Now()
			tok, err := session.Token(context.Background())
			if err != nil || tok.Value != "a-new" || session.Refresh != "rt-new" {
				t.Fatalf("Token = %q, %v (refresh %q); want a-new and rt-new kept", tok.Value, err, session.Refresh)
			}
			if stored, err := store.Load(context.Background(), llmprovider.ProviderOpenAI); err != nil || stored.Refresh != "rt-new" {
				t.Errorf("store holds %v, %v; want rt-new saved", stored, err)
			}
			if tok.Expiry.Before(before.Add(time.Hour-time.Minute)) || tok.Expiry.After(time.Now().Add(time.Hour+time.Minute)) {
				t.Errorf("Expiry = %v, want an hour from now", tok.Expiry)
			}
		})
	}
}

// TestOAuthRefresh_LargeReplyIsBounded (0021-MADR T4): a 2 MiB refresh reply
// is an error, after reading at most oauthResponseLimit + 1 bytes of it.
func TestOAuthRefresh_LargeReplyIsBounded(t *testing.T) {
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"`+strings.Repeat("a", 2<<20)+`","refresh_token":"rt-new"}`)
	})
	session := expiredSession(srv, "")
	var read atomic.Int64
	session.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err == nil {
			resp.Body = &countingBody{ReadCloser: resp.Body, n: &read}
		}
		return resp, err
	})}
	if tok, err := session.Token(context.Background()); err == nil {
		t.Fatalf("Token = %d bytes, <nil>; want an error for the oversized reply", len(tok.Value))
	}
	if n := read.Load(); n > oauthResponseLimit+1 {
		t.Errorf("read %d bytes of the reply, want at most %d", n, oauthResponseLimit+1)
	}
}

// countingBody counts the bytes read from a reply body.
type countingBody struct {
	io.ReadCloser
	n *atomic.Int64
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.n.Add(int64(n))
	return n, err
}
