package llmprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// deviceCounts counts a fake device server's token polls and approvals.
type deviceCounts struct{ polls, approvals atomic.Int32 }

// grokDeviceServer serves discovery, a device code valid 60 s with a 1 s
// interval, and a token endpoint answering pending until approve is closed.
func grokDeviceServer(t *testing.T, approve <-chan struct{}) (*httptest.Server, *deviceCounts) {
	t.Helper()
	counts := &deviceCounts{}
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, ti.discovery(t, map[string]string{"device_authorization_endpoint": srv.URL + "/device", "token_endpoint": srv.URL + "/token"}))
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, `{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		counts.polls.Add(1)
		select {
		case <-approve:
			counts.approvals.Add(1)
			writeTestJSON(t, w, ti.tokenResponse(t, "ES256", "test-client", "grok-access", "grok-refresh"))
		default:
			w.WriteHeader(http.StatusBadRequest)
			writeTestJSON(t, w, `{"error":"authorization_pending"}`)
		}
	})
	return srv, counts
}

func grokDeviceOptions(srv *httptest.Server, clock *fakeOAuthClock) OAuthFlowOptions {
	opts := OAuthFlowOptions{HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL}
	if clock != nil {
		opts.now, opts.sleep = clock.now, clock.sleep
	}
	return opts
}

// TestDeviceLogin_HandleThenWait (0016-MADR D6): the handle shows the code,
// the URI and the expiry before any polling, and Wait returns the session.
func TestDeviceLogin_HandleThenWait(t *testing.T) {
	approve := make(chan struct{})
	close(approve)
	srv, _ := grokDeviceServer(t, approve)
	clock := newFakeOAuthClock()
	login, err := StartDeviceOAuth(context.Background(), ProviderGrok, grokDeviceOptions(srv, clock))
	if err != nil {
		t.Fatal(err)
	}
	if login.UserCode != "ABCD-EFGH" || login.VerificationURI != "https://example.test/activate" ||
		!login.Expiry.Equal(clock.now().Add(60*time.Second)) {
		t.Fatalf("handle = %q %q %v", login.UserCode, login.VerificationURI, login.Expiry)
	}
	session, err := login.Wait(context.Background())
	if err != nil || session.Access != "grok-access" {
		t.Fatalf("Wait = %v, %v", session, err)
	}
}

// TestDeviceLogin_CancelDuringPolling: Cancel from another goroutine makes a
// polling Wait return within one poll interval, with context.Canceled.
func TestDeviceLogin_CancelDuringPolling(t *testing.T) {
	srv, counts := grokDeviceServer(t, make(chan struct{}))
	login, err := StartDeviceOAuth(context.Background(), ProviderGrok, grokDeviceOptions(srv, nil))
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		err     error
		elapsed time.Duration
	}
	done := make(chan result, 1)
	go func() {
		_, err := login.Wait(context.Background())
		done <- result{err: err}
	}()
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	login.Cancel()
	select {
	case r := <-done:
		r.elapsed = time.Since(start)
		if !errors.Is(r.err, context.Canceled) || r.elapsed >= time.Second {
			t.Fatalf("Wait after Cancel = %v after %v; want context.Canceled within one 1 s interval", r.err, r.elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Wait did not return after Cancel")
	}
	if counts.approvals.Load() != 0 {
		t.Error("a canceled login was approved")
	}
}

// TestDeviceLogin_CancelBeforeWait: a canceled login's Wait returns at
// once, without polling.
func TestDeviceLogin_CancelBeforeWait(t *testing.T) {
	srv, counts := grokDeviceServer(t, make(chan struct{}))
	login, err := StartDeviceOAuth(context.Background(), ProviderGrok, grokDeviceOptions(srv, nil))
	if err != nil {
		t.Fatal(err)
	}
	login.Cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	_, err = login.Wait(ctx)
	if !errors.Is(err, context.Canceled) || time.Since(start) >= 500*time.Millisecond || counts.polls.Load() != 0 {
		t.Fatalf("Wait after Cancel = %v after %v and %d polls; want context.Canceled at once, without polling",
			err, time.Since(start), counts.polls.Load())
	}
}

// TestDeviceLogin_WaitersShareOneResult: concurrent Waits poll once and get
// the same session.
func TestDeviceLogin_WaitersShareOneResult(t *testing.T) {
	approve := make(chan struct{})
	close(approve)
	srv, counts := grokDeviceServer(t, approve)
	login, err := StartDeviceOAuth(context.Background(), ProviderGrok, grokDeviceOptions(srv, newFakeOAuthClock()))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	sessions := make([]*OAuthSession, 3)
	for i := range sessions {
		wg.Go(func() {
			s, err := login.Wait(context.Background())
			if err != nil {
				t.Errorf("Wait %d: %v", i, err)
			}
			sessions[i] = s
		})
	}
	wg.Wait()
	if counts.approvals.Load() != 1 || sessions[0] == nil || sessions[1] != sessions[0] || sessions[2] != sessions[0] {
		t.Fatalf("%d approvals, sessions %p %p %p; want one poll shared", counts.approvals.Load(), sessions[0], sessions[1], sessions[2])
	}
}

// TestDeviceLogin_OpenAIHandle: the OpenAI handle points at the issuer's
// device page and carries the 15-minute login window.
func TestDeviceLogin_OpenAIHandle(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, ti.discovery(t, nil))
	})
	mux.HandleFunc("/api/accounts/deviceauth/usercode", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, `{"device_auth_id":"dev-1","user_code":"WXYZ-1234","interval":"5"}`)
	})
	clock := newFakeOAuthClock()
	login, err := StartDeviceOAuth(context.Background(), ProviderOpenAI, grokDeviceOptions(srv, clock))
	if err != nil {
		t.Fatal(err)
	}
	if login.UserCode != "WXYZ-1234" || login.VerificationURI != srv.URL+"/codex/device" ||
		!login.Expiry.Equal(clock.now().Add(openAIDeviceTimeout)) {
		t.Fatalf("handle = %q %q %v", login.UserCode, login.VerificationURI, login.Expiry)
	}
}
