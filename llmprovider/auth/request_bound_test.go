package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// stallServer sends headers and the start of a body, then nothing, until the
// request ends.
func stallServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "{")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestAuthRequests_Bounded (0021-MADR D2, amendment item 6): with no total
// timeout on the default client, every auth request still ends at
// authRequestTimeout when the issuer stalls its body: the device poll and the
// token exchange, the issuer's keys, and Kilo's device login. The refresh
// ends at oauthRefreshAttemptTimeout, its own bound (0021-MADR T8).
func TestAuthRequests_Bounded(t *testing.T) {
	old, oldAttempt := authRequestTimeout, oauthRefreshAttemptTimeout
	authRequestTimeout, oauthRefreshAttemptTimeout = 100*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { authRequestTimeout, oauthRefreshAttemptTimeout = old, oldAttempt })
	srv := stallServer(t)
	client := &http.Client{} // no timeout of its own
	readAll := func(resp *http.Response) error {
		_, err := io.ReadAll(resp.Body)
		return err
	}
	for name, run := range map[string]func() error{
		"json post": func() error {
			resp, err := postOAuthJSON(context.Background(), client, srv.URL, map[string]string{"a": "b"})
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			return readAll(resp)
		},
		"form post": func() error {
			resp, err := postOAuthForm(context.Background(), client, srv.URL, url.Values{"a": {"b"}})
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			return readAll(resp)
		},
		"kilo": func() error {
			resp, err := kiloDeviceRequest(context.Background(), oauthFlowConfig{httpClient: client}, http.MethodGet, srv.URL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			return readAll(resp)
		},
		"keys": func() error {
			_, err := fetchJWKS(context.Background(), client, srv.URL)
			return err
		},
		"refresh": func() error {
			session := expiredSession(srv, "")
			session.HTTPClient = client
			_, err := session.Token(context.Background())
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			err := run()
			if err == nil {
				t.Fatal("a stalled body gave no error")
			}
			// The refresh tries up to three times, each bounded.
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("a stalled body took %s, want each request ended at 100 ms", elapsed)
			}
		})
	}
}
