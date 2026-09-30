package llmprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const kiloTestToken = "kilo-device-token-0123456789"

// kiloDeviceServer serves Kilo's device endpoints: the code, then the poll
// answers in order, the last repeating.
func kiloDeviceServer(t *testing.T, startStatus int, polls ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var polled atomic.Int32
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("POST /api/device-auth/codes", func(w http.ResponseWriter, _ *http.Request) {
		if startStatus != http.StatusOK {
			w.WriteHeader(startStatus)
			return
		}
		writeTestJSON(t, w, `{"code":"KILO-1234","verificationUrl":"https://app.kilo.ai/device","expiresIn":60}`)
	})
	mux.HandleFunc("GET /api/device-auth/codes/{code}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("code") != "KILO-1234" {
			t.Errorf("polled code %q", r.PathValue("code"))
		}
		n := int(polled.Add(1))
		status := polls[min(n, len(polls))-1]
		if status == http.StatusOK {
			writeTestJSON(t, w, `{"status":"approved","token":"`+kiloTestToken+`","userEmail":"user@example.test"}`)
			return
		}
		w.WriteHeader(status)
	})
	return srv, &polled
}

func kiloOptions(srv *httptest.Server, clock *fakeOAuthClock) OAuthFlowOptions {
	opts := OAuthFlowOptions{HTTPClient: srv.Client(), Issuer: srv.URL}
	if clock != nil {
		opts.now, opts.sleep = clock.now, clock.sleep
	}
	return opts
}

// TestKiloDevice_ApprovedAfterPending (0017-MADR D2): the handle shows the
// code and URL, polls every 3 s, and returns a session that never refreshes.
func TestKiloDevice_ApprovedAfterPending(t *testing.T) {
	srv, polled := kiloDeviceServer(t, http.StatusOK, http.StatusAccepted, http.StatusAccepted, http.StatusOK)
	clock := newFakeOAuthClock()
	login, err := StartDeviceOAuth(context.Background(), ProviderKilo, kiloOptions(srv, clock))
	if err != nil {
		t.Fatal(err)
	}
	if login.UserCode != "KILO-1234" || login.VerificationURI != "https://app.kilo.ai/device" ||
		!login.Expiry.Equal(clock.now().Add(60*time.Second)) {
		t.Fatalf("handle = %q %q %v", login.UserCode, login.VerificationURI, login.Expiry)
	}
	session, err := login.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if session.Provider != ProviderKilo || session.Access != kiloTestToken || session.Refresh != "" || !session.Expiry.IsZero() {
		t.Fatalf("session = %v", session)
	}
	if got := clock.sleeps(); !reflect.DeepEqual(got, []time.Duration{3 * time.Second, 3 * time.Second, 3 * time.Second}) || polled.Load() != 3 {
		t.Errorf("sleeps %v over %d polls, want three 3 s waits", got, polled.Load())
	}
	if strings.Contains(fmt.Sprintf("%v %+v", session, session), kiloTestToken) {
		t.Error("the Kilo token shows when the session is formatted")
	}
}

// TestKiloDevice_Outcomes: denial, expiry, a rate-limited start, and an unsafe
// verification URL each fail.
func TestKiloDevice_Outcomes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start int
		polls []int
		want  string
	}{
		{"denied", http.StatusOK, []int{http.StatusForbidden}, "denied"},
		{"expired", http.StatusOK, []int{http.StatusAccepted, http.StatusGone}, "expired"},
		{"unexpected poll status", http.StatusOK, []int{http.StatusInternalServerError}, "Kilo device poll"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := kiloDeviceServer(t, tc.start, tc.polls...)
			login, err := StartDeviceOAuth(context.Background(), ProviderKilo, kiloOptions(srv, newFakeOAuthClock()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := login.Wait(context.Background()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Wait = %v, want an error naming %q", err, tc.want)
			}
		})
	}
	t.Run("too many pending", func(t *testing.T) {
		srv, _ := kiloDeviceServer(t, http.StatusTooManyRequests)
		if _, err := StartDeviceOAuth(context.Background(), ProviderKilo, kiloOptions(srv, nil)); !errors.Is(err, ErrRateLimited) {
			t.Errorf("start = %v, want ErrRateLimited", err)
		}
	})
	t.Run("unsafe verification URL", func(t *testing.T) {
		mux := http.NewServeMux()
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		mux.HandleFunc("POST /api/device-auth/codes", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, `{"code":"KILO-1234","verificationUrl":"http://evil.example/device","expiresIn":60}`)
		})
		if _, err := StartDeviceOAuth(context.Background(), ProviderKilo, kiloOptions(srv, nil)); err == nil {
			t.Error("start accepted a plain-http, non-loopback verification URL")
		}
	})
}

// TestKiloDevice_Cancel: Cancel during polling returns within one interval.
func TestKiloDevice_Cancel(t *testing.T) {
	srv, _ := kiloDeviceServer(t, http.StatusOK, http.StatusAccepted)
	login, err := StartDeviceOAuth(context.Background(), ProviderKilo, kiloOptions(srv, nil))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := login.Wait(context.Background())
		done <- err
	}()
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	login.Cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || time.Since(start) >= kiloDevicePoll {
			t.Fatalf("Wait after Cancel = %v after %v", err, time.Since(start))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait did not return after Cancel")
	}
}

// TestKiloDevice_SavedLoginNeverRefreshes: a stored Kilo login, loaded again,
// yields its token with no network call, however much time has passed.
func TestKiloDevice_SavedLoginNeverRefreshes(t *testing.T) {
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Save(ctx, ProviderKilo, &OAuthSession{Provider: ProviderKilo, Access: kiloTestToken, Issuer: kiloAPIOrigin}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(ctx, ProviderKilo)
	if err != nil {
		t.Fatal(err)
	}
	loaded.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		t.Errorf("a stored Kilo login made a request to %s", r.URL)
		return nil, errors.New("no network")
	})}
	tok, err := loaded.Token(ctx)
	if err != nil || tok.Value != kiloTestToken {
		t.Fatalf("Token = %v, %v", tok, err)
	}
}

// TestKiloProfile reads the account's organizations from {origin}/api/profile.
func TestKiloProfile(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/profile" {
			t.Errorf("profile path %q", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		writeTestJSON(t, w, `{"user":{"email":"user@example.test","name":"U"},"organizations":[{"id":"org-1","name":"Acme","role":"member"}],"selectedOrganizationId":"org-1","hasPersonalAccount":true}`)
	}))
	t.Cleanup(srv.Close)
	account, err := KiloProfile(context.Background(), kiloTestToken, WithBaseURL(srv.URL+"/api/gateway"), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	want := KiloAccount{Email: "user@example.test", Organizations: []KiloOrganization{{ID: "org-1", Name: "Acme", Role: "member"}},
		SelectedOrganizationID: "org-1", HasPersonalAccount: true}
	if !reflect.DeepEqual(account, want) || auth != "Bearer "+kiloTestToken {
		t.Errorf("account = %+v (Authorization %q), want %+v", account, auth, want)
	}
}
