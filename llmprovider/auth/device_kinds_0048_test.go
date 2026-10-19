package auth

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// routes0048 replaces one route of a fake issuer.
func routes0048(pattern string, handler func(f *fake0048) http.HandlerFunc) func(f *fake0048) map[string]http.HandlerFunc {
	return func(f *fake0048) map[string]http.HandlerFunc {
		return map[string]http.HandlerFunc{pattern: handler(f)}
	}
}

// reply0048 is a route that answers status and body.
func reply0048(t *testing.T, pattern string, status int, body string) func(f *fake0048) map[string]http.HandlerFunc {
	return routes0048(pattern, func(*fake0048) http.HandlerFunc { return answer0048(t, status, body) })
}

// TestStartDeviceOAuth_PlainFailures_0048 shows that StartDeviceOAuth's
// unsupported provider, discovery, status, decode and endpoint failures are
// plain errors naming their step (0048-MADR D4).
func TestStartDeviceOAuth_PlainFailures_0048(t *testing.T) {
	t.Run("unsupported provider", func(t *testing.T) {
		_, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderClaude, OAuthFlowOptions{})
		assertPlain0048(t, err, "is not supported")
	})
	const discovery = "/.well-known/openid-configuration"
	for _, tc := range []struct {
		name     string
		provider llmprovider.ProviderID
		routes   func(f *fake0048) map[string]http.HandlerFunc
		drop     string
		step     string
	}{
		{"OpenAI discovery fails", llmprovider.ProviderOpenAI, reply0048(t, discovery, http.StatusInternalServerError, "down"), "", "discovery for"},
		{"Grok discovery fails", llmprovider.ProviderGrok, reply0048(t, discovery, http.StatusInternalServerError, "down"), "", "discovery for"},
		{"OpenAI non-2xx", llmprovider.ProviderOpenAI, reply0048(t, "/api/accounts/deviceauth/usercode", http.StatusBadRequest, "{}"), "", "OpenAI device-code request failed"},
		{"Grok non-2xx", llmprovider.ProviderGrok, reply0048(t, "/device", http.StatusBadRequest, "{}"), "", "Grok device-code request failed"},
		{"Kilo non-2xx", llmprovider.ProviderKilo, reply0048(t, kiloCodes0048, http.StatusBadRequest, "{}"), "", "Kilo device-code request failed"},
		{"OpenAI reply does not decode", llmprovider.ProviderOpenAI, reply0048(t, "/api/accounts/deviceauth/usercode", http.StatusOK, "{"), "", "decode OpenAI device-code response"},
		{"Grok reply does not decode", llmprovider.ProviderGrok, reply0048(t, "/device", http.StatusOK, "{"), "", "decode Grok device-code response"},
		{"Kilo reply does not decode", llmprovider.ProviderKilo, reply0048(t, kiloCodes0048, http.StatusOK, "{"), "", "decode Kilo device-code response"},
		{"OpenAI reply incomplete", llmprovider.ProviderOpenAI, reply0048(t, "/api/accounts/deviceauth/usercode", http.StatusOK, "{}"), "", "OpenAI device-code response is incomplete"},
		{"Grok reply incomplete", llmprovider.ProviderGrok, reply0048(t, "/device", http.StatusOK, "{}"), "", "Grok device-code response is incomplete"},
		{"Kilo reply incomplete", llmprovider.ProviderKilo, reply0048(t, kiloCodes0048, http.StatusOK, "{}"), "", "Kilo device-code response is incomplete"},
		{"Grok discovery has no device endpoint", llmprovider.ProviderGrok, nil, "device_authorization_endpoint", "lacks an endpoint this login needs"},
		{"Grok discovery has no token endpoint", llmprovider.ProviderGrok, nil, "token_endpoint", "lacks an endpoint this login needs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var drop []string
			if tc.drop != "" {
				drop = []string{tc.drop}
			}
			f := newFake0048(t, tc.provider, tc.routes, drop...)
			login, err := StartDeviceOAuth(context.Background(), tc.provider, f.options(nil))
			if login != nil {
				t.Errorf("StartDeviceOAuth returned a login with its error")
			}
			assertPlain0048(t, err, tc.step)
		})
	}
}

// TestStartDeviceOAuth_TransportFailure_0048 shows that a device-code request
// that fails in transport wraps the *url.Error, and matches context.Canceled
// when the caller's context ends (0048-MADR D4).
func TestStartDeviceOAuth_TransportFailure_0048(t *testing.T) {
	for _, tc := range []struct {
		provider   llmprovider.ProviderID
		path, step string
	}{
		{llmprovider.ProviderOpenAI, "/api/accounts/deviceauth/usercode", "oauth: request"},
		{llmprovider.ProviderGrok, "/device", "oauth: request"},
		{llmprovider.ProviderKilo, "/api/device-auth/codes", "oauth: Kilo request"},
	} {
		t.Run(string(tc.provider), func(t *testing.T) {
			f := newFake0048(t, tc.provider, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opts := f.options(nil)
			opts.HTTPClient = f.cancelOn0048(tc.path, cancel)
			_, err := StartDeviceOAuth(ctx, tc.provider, opts)
			var urlErr *url.Error
			if !errors.Is(err, context.Canceled) || !errors.As(err, &urlErr) {
				t.Fatalf("err = %v, want a *url.Error matching context.Canceled", err)
			}
			assertPlain0048(t, err, tc.step)
		})
	}
}

// TestDeviceLoginWait_ContextEnds_0048 shows that the caller's context ending
// ends Wait with an error matching the context's, whether it ends during a
// poll request or in the wait between polls (0048-MADR D4).
func TestDeviceLoginWait_ContextEnds_0048(t *testing.T) {
	pending := map[llmprovider.ProviderID]func(f *fake0048) map[string]http.HandlerFunc{
		llmprovider.ProviderOpenAI: reply0048(t, openAIPoll0048, http.StatusForbidden, "{}"),
		llmprovider.ProviderGrok:   reply0048(t, "/token", http.StatusBadRequest, `{"error":"authorization_pending"}`),
		llmprovider.ProviderKilo:   reply0048(t, kiloPoll0048, http.StatusAccepted, ""),
	}
	for _, tc := range []struct {
		provider   llmprovider.ProviderID
		path, step string
	}{
		{llmprovider.ProviderOpenAI, openAIPoll0048, "oauth: request"},
		{llmprovider.ProviderGrok, "/token", "oauth: request"},
		{llmprovider.ProviderKilo, kiloPollPath0048, "oauth: Kilo request"},
	} {
		t.Run(string(tc.provider)+" during a poll request", func(t *testing.T) {
			f := newFake0048(t, tc.provider, pending[tc.provider])
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opts := f.options(newFakeOAuthClock())
			opts.HTTPClient = f.cancelOn0048(tc.path, cancel)
			login, err := StartDeviceOAuth(ctx, tc.provider, opts)
			if err != nil {
				t.Fatal(err)
			}
			_, err = login.Wait(ctx)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Wait = %v, want context.Canceled", err)
			}
			assertPlain0048(t, err, tc.step)
		})
		t.Run(string(tc.provider)+" between polls", func(t *testing.T) {
			f := newFake0048(t, tc.provider, pending[tc.provider])
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			opts := f.options(nil)
			// The wait between polls is where the context ends.
			opts.sleep = func(sleepCtx context.Context, _ time.Duration) error {
				cancel()
				<-sleepCtx.Done()
				return sleepCtx.Err()
			}
			login, err := StartDeviceOAuth(ctx, tc.provider, opts)
			if err != nil {
				t.Fatal(err)
			}
			_, err = login.Wait(ctx)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Wait = %v, want context.Canceled", err)
			}
			assertPlain0048(t, err, "device-code wait")
		})
	}
	t.Run("OpenAI past its deadline", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderOpenAI, pending[llmprovider.ProviderOpenAI])
		login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderOpenAI, f.options(newFakeOAuthClock()))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		_, err = login.Wait(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Wait = %v, want context.DeadlineExceeded", err)
		}
		assertPlain0048(t, err, "oauth: request")
	})
}

// TestDeviceLoginWait_SecondWaiterContextEnds_0048 shows that a second Wait
// whose own context ends returns that context's error while the first keeps
// polling (0048-MADR D4).
func TestDeviceLoginWait_SecondWaiterContextEnds_0048(t *testing.T) {
	polled := make(chan struct{}, 1)
	f := newFake0048(t, llmprovider.ProviderOpenAI, routes0048(openAIPoll0048, func(*fake0048) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			select {
			case polled <- struct{}{}:
			default:
			}
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderOpenAI, f.options(nil))
	if err != nil {
		t.Fatal(err)
	}
	first := make(chan error, 1)
	go func() {
		_, err := login.Wait(context.Background())
		first <- err
	}()
	<-polled // the first Wait is polling
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	second := make(chan error, 1)
	go func() {
		_, err := login.Wait(ctx)
		second <- err
	}()
	select {
	case err := <-second:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("second Wait = %v, want its own context's context.DeadlineExceeded", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("the second Wait did not return when its context ended")
	}
	login.Cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Errorf("first Wait after Cancel = %v, want context.Canceled", err)
	}
}

// TestDeviceLoginWait_PlainFailures_0048 shows that Wait's unexpected poll
// answers, undecodable or incomplete poll replies, a failed token exchange and
// a rejected id_token are plain errors naming their step (0048-MADR D4).
func TestDeviceLoginWait_PlainFailures_0048(t *testing.T) {
	approved := `{"authorization_code":"auth-code","code_verifier":"verifier"}`
	for _, tc := range []struct {
		name     string
		provider llmprovider.ProviderID
		routes   func(f *fake0048) map[string]http.HandlerFunc
		step     string
	}{
		{"OpenAI unexpected poll status", llmprovider.ProviderOpenAI, reply0048(t, openAIPoll0048, http.StatusBadRequest, "{}"), "OpenAI device poll failed"},
		{"OpenAI poll reply does not decode", llmprovider.ProviderOpenAI, reply0048(t, openAIPoll0048, http.StatusOK, "{"), "decode OpenAI device token"},
		{"OpenAI poll reply incomplete", llmprovider.ProviderOpenAI, reply0048(t, openAIPoll0048, http.StatusOK, "{}"), "OpenAI device token response is incomplete"},
		{"OpenAI token exchange non-2xx", llmprovider.ProviderOpenAI, func(f *fake0048) map[string]http.HandlerFunc {
			return map[string]http.HandlerFunc{
				openAIPoll0048: answer0048(t, http.StatusOK, approved),
				"/oauth/token": answer0048(t, http.StatusBadRequest, `{"error":"invalid_grant"}`),
			}
		}, "token exchange failed"},
		{"Grok unknown error code", llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusBadRequest, `{"error":"unknown_code"}`), "Grok device token failed: unknown_code"},
		{"Grok poll reply does not decode", llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusOK, "{"), "decode Grok device token"},
		{"Grok error reply does not decode", llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusBadRequest, "not json"), "decode Grok device error"},
		{"Grok token reply has no id_token", llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusOK, `{"access_token":"access","refresh_token":"refresh"}`), "id_token rejected"},
		{"Kilo approval does not decode", llmprovider.ProviderKilo, reply0048(t, kiloPoll0048, http.StatusOK, "{"), "decode Kilo device approval"},
		{"Kilo approval has no usable token", llmprovider.ProviderKilo, reply0048(t, kiloPoll0048, http.StatusOK, `{"token":"short"}`), "carries no usable token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake0048(t, tc.provider, tc.routes)
			login, err := StartDeviceOAuth(context.Background(), tc.provider, f.options(newFakeOAuthClock()))
			if err != nil {
				t.Fatal(err)
			}
			session, err := login.Wait(context.Background())
			if session != nil {
				t.Errorf("Wait returned a session with its error")
			}
			assertPlain0048(t, err, tc.step)
		})
	}
}

// TestLoginDeviceOAuth_StartAndWaitErrors_0048 shows that LoginDeviceOAuth
// returns Start's error, plain for an unsupported provider, and Wait's,
// context.Canceled when the caller's context ends while it waits
// (0048-MADR D4).
func TestLoginDeviceOAuth_StartAndWaitErrors_0048(t *testing.T) {
	t.Run("Start's error", func(t *testing.T) {
		_, err := LoginDeviceOAuth(context.Background(), llmprovider.ProviderClaude, OAuthFlowOptions{})
		assertPlain0048(t, err, "is not supported")
	})
	t.Run("Wait's error", func(t *testing.T) {
		f := newFake0048(t, llmprovider.ProviderGrok,
			reply0048(t, "/token", http.StatusBadRequest, `{"error":"authorization_pending"}`))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		opts := f.options(nil)
		var shown string
		opts.NotifyDevice = func(_, userCode string) {
			shown = userCode
			cancel() // the caller gives up once the code is shown
		}
		_, err := LoginDeviceOAuth(ctx, llmprovider.ProviderGrok, opts)
		if shown != "ABCD-EFGH" || !errors.Is(err, context.Canceled) {
			t.Fatalf("LoginDeviceOAuth = %v after showing %q; want context.Canceled after showing the code", err, shown)
		}
		assertPlain0048(t, err, "device-code wait")
	})
}
