package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// unsupportedRevocation is a session RevokeOAuthSession cannot revoke
// (0076-MADR D1, C1 to C3), with the requests its revocation may send.
type unsupportedRevocation struct {
	name     string
	session  *OAuthSession
	issuer   string
	requests func() []string
	want     []string
}

// unsupportedRevocations gives the three: a Kilo session, which has no
// revocation; a Grok session, and a plan session, each at a test issuer that
// publishes no revocation_endpoint. OpenAI's own issuer fills a missing
// endpoint from its built-in one (C36), so the plan's needs another issuer.
func unsupportedRevocations(t *testing.T) []unsupportedRevocation {
	t.Helper()
	kilo := &hostRecorder{}
	grok, grokRequests := noRevocationIssuer(t)
	plan, planRequests := noRevocationIssuer(t)
	planSession := planTestSession(plan.URL + "/oauth2/token")
	planSession.Issuer, planSession.HTTPClient = plan.URL, plan.Client()
	discovery := []string{"GET /.well-known/openid-configuration"}
	return []unsupportedRevocation{
		{name: "a Kilo session", requests: kilo.requests,
			session: &OAuthSession{Provider: llmprovider.ProviderKilo, Access: "kilo-token", HTTPClient: &http.Client{Transport: kilo}}},
		{name: "a Grok session", issuer: grok.URL, requests: grokRequests, want: discovery,
			session: &OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a", Refresh: "rt", Issuer: grok.URL,
				ClientID: "grok-client", HTTPClient: grok.Client()}},
		{name: "a plan session", issuer: plan.URL, requests: planRequests, want: discovery, session: planSession},
	}
}

// noRevocationIssuer serves a discovery document whose endpoints are all on
// its own origin and which names no revocation_endpoint. It records every
// request it receives.
func noRevocationIssuer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.URL.Path != "/.well-known/openid-configuration" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": srv.URL, "authorization_endpoint": srv.URL + "/oauth2/authorize",
			"token_endpoint": srv.URL + "/oauth2/token", "jwks_uri": srv.URL + "/jwks",
		})
	})
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// TestRevokeUnsupportedKind_0076 (0076-MADR D1): each revocation that cannot
// revoke matches llmprovider.ErrUnsupported, the kind R25 names, and still
// the standard library's. The Kilo revocation sends no request, and the
// other two send only the discovery request.
func TestRevokeUnsupportedKind_0076(t *testing.T) {
	for _, c := range unsupportedRevocations(t) {
		err := RevokeOAuthSession(context.Background(), c.session)
		if !errors.Is(err, llmprovider.ErrUnsupported) || !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("%s: RevokeOAuthSession = %v; want an error matching llmprovider.ErrUnsupported and errors.ErrUnsupported", c.name, err)
		}
		if got := c.requests(); !slices.Equal(got, c.want) {
			t.Errorf("%s: sent %q; want %q", c.name, got, c.want)
		}
	}
}

// TestRevokeMessages_0076 (0076-MADR D1): each error keeps its own words, and
// only its wrapped part, printed through the final %w, is the SDK's
// sentinel's. The test issuer's URL is replaced by ISSUER.
func TestRevokeMessages_0076(t *testing.T) {
	want := map[string]string{
		"a Kilo session": "oauth: revoke: kilo sessions have no revocation: llmprovider: unsupported",
		"a Grok session": "oauth: revoke: ISSUER publishes no revocation_endpoint: llmprovider: unsupported",
		"a plan session": "oauth: revoke: ISSUER publishes no revocation_endpoint: llmprovider: unsupported",
	}
	for _, c := range unsupportedRevocations(t) {
		err := RevokeOAuthSession(context.Background(), c.session)
		if err == nil {
			t.Errorf("%s: RevokeOAuthSession = nil; want %q", c.name, want[c.name])
			continue
		}
		got := err.Error()
		if c.issuer != "" {
			got = strings.ReplaceAll(got, c.issuer, "ISSUER")
		}
		if got != want[c.name] {
			t.Errorf("%s: Error() = %q; want %q", c.name, got, want[c.name])
		}
	}
}

// steppedOAuthClock moves on by a tick at every read, as time passes between
// two reads of a real clock, and by the whole duration at every sleep.
type steppedOAuthClock struct {
	mu      sync.Mutex
	current time.Time
	tick    time.Duration
	slept   []time.Duration
}

func (c *steppedOAuthClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := c.current
	c.current = c.current.Add(c.tick)
	return t
}

func (c *steppedOAuthClock) sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	c.current = c.current.Add(d)
	c.slept = append(c.slept, d)
	c.mu.Unlock()
	return ctx.Err()
}

func (c *steppedOAuthClock) sleeps() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.slept...)
}

// TestOpenAIDeviceExpiryKind_0076 (0076-MADR D2): the OpenAI poll's own
// deadline check gives the pacer's expiry, which matches
// llmprovider.ErrAuthFailure as Wait's doc says (C7), and names the last
// transient failure when there was one.
//
// The clock passes the deadline between the pacer's check (C8) and the
// loop's (C5). With a one-second tick and an 896 s interval, its reads are:
// the start, T0, so the deadline is T0+900s; the loop's check, T0+1s; the
// pacer's remaining time, T0+2s, before its 896 s sleep; the pacer's check
// after the sleep, T0+899s, still before the deadline; and the loop's next
// check, T0+900s, at it.
func TestOpenAIDeviceExpiryKind_0076(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		last   string
	}{
		{name: "a pending poll", status: http.StatusNotFound},
		{name: "a failed poll", status: http.StatusServiceUnavailable,
			last: "; the last poll failed: oauth: OpenAI device poll failed: Service Unavailable"},
	} {
		clock := &steppedOAuthClock{current: time.Unix(1_700_000_000, 0), tick: time.Second}
		var mu sync.Mutex
		var polls int
		mux := http.NewServeMux()
		srv := httptest.NewServer(mux)
		ti := newTestIssuer(mux, srv.URL)
		mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.discovery(t, nil))
		})
		mux.HandleFunc("/api/accounts/deviceauth/usercode", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, `{"device_auth_id":"device-auth-id","user_code":"OPEN-AI","interval":"896"}`)
		})
		mux.HandleFunc("/api/accounts/deviceauth/token", func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			polls++
			mu.Unlock()
			w.WriteHeader(c.status)
		})

		login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderOpenAI, OAuthFlowOptions{
			HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL, now: clock.now, sleep: clock.sleep,
		})
		if err != nil {
			srv.Close()
			t.Fatalf("%s: StartDeviceOAuth = %v", c.name, err)
		}
		_, err = login.Wait(context.Background())
		srv.Close()
		if !errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("%s: Wait = %v; want an error matching llmprovider.ErrAuthFailure", c.name, err)
		}
		if err != nil && !strings.HasSuffix(err.Error(), "device code expired"+c.last) {
			t.Errorf("%s: Wait = %q; want it to end %q", c.name, err, "device code expired"+c.last)
		}
		mu.Lock()
		n := polls
		mu.Unlock()
		if got, want := clock.sleeps(), []time.Duration{896 * time.Second}; n != 1 || !slices.Equal(got, want) {
			t.Errorf("%s: %d poll(s) and sleeps %v; want 1 and %v", c.name, n, got, want)
		}
	}
}
