package kilo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestNew_AcceptsAKiloDeviceSession: the session auth's Kilo device login
// returns is a Kilo credential, applied like a key in Authorization: Bearer
// (0017-MADR D2). A 401 is ErrAuthFailure, telling the caller to log in again.
// Another provider's session is still refused (TestNew_RefusesAnOAuthSession).
func TestNew_AcceptsAKiloDeviceSession(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
	}{{"accepted", http.StatusOK}, {"rejected", http.StatusUnauthorized}} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var authz string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				authz = r.Header.Get("Authorization")
				mu.Unlock()
				w.WriteHeader(tc.status)
				if tc.status == http.StatusOK {
					_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
					return
				}
				_, _ = io.WriteString(w, `{"error":{"message":"invalid token"}}`)
			}))
			t.Cleanup(srv.Close)
			session := &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: "kilo-device-token"}
			p, err := New(llmprovider.WithTokenSource(session), llmprovider.WithModel("kilo-auto/free"),
				llmprovider.WithBaseURL(srv.URL))
			if err != nil {
				t.Fatalf("New with a Kilo device-login session: %v", err)
			}
			_, err = llmprovider.GenerateText(context.Background(), p, text("hi"))
			mu.Lock()
			defer mu.Unlock()
			if authz != "Bearer kilo-device-token" {
				t.Errorf("Authorization = %q, want the session's token as a bearer", authz)
			}
			switch {
			case tc.status == http.StatusOK && err != nil:
				t.Errorf("Generate: %v", err)
			case tc.status == http.StatusUnauthorized && !errors.Is(err, llmprovider.ErrAuthFailure):
				t.Errorf("Generate after a 401 = %v, want ErrAuthFailure", err)
			}
		})
	}
}
