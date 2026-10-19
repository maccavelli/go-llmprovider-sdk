package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// browserLogin0048 runs a Grok browser login against f, with opts' hooks,
// under ctx, and returns its error.
func browserLogin0048(t *testing.T, ctx context.Context, f *fake0048, hooks OAuthFlowOptions) error {
	t.Helper()
	opts := f.options(nil)
	opts.OpenURL, opts.InputCode = hooks.OpenURL, hooks.InputCode
	session, err := LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, opts)
	if session != nil {
		t.Errorf("LoginBrowserOAuth returned a session with its error")
	}
	return err
}

// TestLoginBrowserOAuth_PlainFailures_0048 shows that LoginBrowserOAuth's
// unsupported provider, caller-issuer discovery failure, IdP error in the
// callback (access_denied included, which is not ErrAuthFailure), failed
// token exchange and rejected id_token are plain errors naming their step
// (0048-MADR D4).
func TestLoginBrowserOAuth_PlainFailures_0048(t *testing.T) {
	t.Run("unsupported provider", func(t *testing.T) {
		_, err := LoginBrowserOAuth(context.Background(), llmprovider.ProviderClaude, OAuthFlowOptions{})
		assertPlain0048(t, err, "is not supported")
	})
	ctx := func(t *testing.T) context.Context {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		t.Cleanup(cancel)
		return ctx
	}
	t.Run("caller issuer discovery fails", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderGrok, reply0048(t, "/.well-known/openid-configuration", http.StatusInternalServerError, "down"))
		assertPlain0048(t, browserLogin0048(t, ctx(t), f, OAuthFlowOptions{}), "discovery for")
	})
	for _, tc := range []struct{ name, query, step string }{
		{"IdP access_denied", "error=access_denied", "authorization failed: access_denied"},
		{"IdP error with a description", "error=server_error&error_description=upstream+down", "authorization failed (server_error): upstream down"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake0048(t, llmprovider.ProviderGrok, nil)
			err := browserLogin0048(t, ctx(t), f, OAuthFlowOptions{OpenURL: visit0048(f.ti, tc.query)})
			assertPlain0048(t, err, tc.step)
		})
	}
	t.Run("token exchange non-2xx", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusBadRequest, `{"error":"invalid_grant"}`))
		err := browserLogin0048(t, ctx(t), f, OAuthFlowOptions{OpenURL: visit0048(f.ti, "code=c")})
		assertPlain0048(t, err, "token exchange failed")
	})
	t.Run("id_token rejected", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderGrok, routes0048("/token", func(f *fake0048) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				idToken := signTestJWT(t, "ES256", "ec-1", map[string]any{"iss": f.ti.base, "aud": "test-client",
					"exp": time.Now().Add(time.Hour).Unix(), "nonce": "not-the-sent-nonce"})
				writeTestJSON(t, w, `{"access_token":"access","refresh_token":"refresh","expires_in":3600,"id_token":"`+idToken+`"}`)
			}
		}))
		err := browserLogin0048(t, ctx(t), f, OAuthFlowOptions{OpenURL: visit0048(f.ti, "code=c")})
		assertPlain0048(t, err, "id_token rejected: nonce does not match")
	})
}

// TestLoginBrowserOAuth_ContextEnds_0048 shows that the login's deadline,
// the caller's or its own 10 minutes, and a canceled context end it with an
// error wrapping the context's (0048-MADR D4).
func TestLoginBrowserOAuth_ContextEnds_0048(t *testing.T) {
	t.Run("deadline", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderGrok, nil)
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		err := browserLogin0048(t, ctx, f, OAuthFlowOptions{}) // the browser never comes back
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want context.DeadlineExceeded", err)
		}
		assertPlain0048(t, err, "browser login timed out")
	})
	t.Run("canceled", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderGrok, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := browserLogin0048(t, ctx, f, OAuthFlowOptions{OpenURL: func(string) error {
			cancel() // the caller gives up once the browser opens
			return nil
		}})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		assertPlain0048(t, err, "browser login timed out")
	})
}

// errInput0048 is an InputCode hook's own failure.
var errInput0048 = errors.New("the prompt was closed")

// TestLoginBrowserOAuth_InputCodeError_0048 shows that InputCode's own error
// ends the login, and errors.Is matches it (0048-MADR D4).
func TestLoginBrowserOAuth_InputCodeError_0048(t *testing.T) {
	f := newFake0048(t, llmprovider.ProviderGrok, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := browserLogin0048(t, ctx, f, OAuthFlowOptions{InputCode: func(context.Context) (string, error) {
		return "", errInput0048
	}})
	if !errors.Is(err, errInput0048) {
		t.Fatalf("err = %v, want InputCode's error", err)
	}
	assertPlain0048(t, err, errInput0048.Error())
}
