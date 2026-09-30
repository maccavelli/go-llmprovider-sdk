package llmprovider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// keysServer serves a testIssuer's /jwks, and a 500 at /broken.
func keysServer(t *testing.T) (*httptest.Server, *testIssuer) {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/broken", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
	return srv, ti
}

func idClaims(iss, aud string, exp time.Time) map[string]any {
	return map[string]any{"iss": iss, "aud": aud, "sub": "user-1", "exp": exp.Unix()}
}

// TestVerifyIDToken_AcceptsEachAlgorithm (0016-MADR D7, A3): a valid token
// under each standard-library algorithm verifies.
func TestVerifyIDToken_AcceptsEachAlgorithm(t *testing.T) {
	srv, _ := keysServer(t)
	check := idTokenCheck{issuer: srv.URL, clientID: "client-1", jwksURL: srv.URL + "/jwks"}
	for _, alg := range []string{"RS256", "PS256", "ES256", "EdDSA"} {
		raw := signTestJWT(t, alg, kidFor(alg), idClaims(srv.URL, "client-1", time.Now().Add(time.Hour)))
		if err := verifyIDToken(context.Background(), srv.Client(), raw, check, time.Now()); err != nil {
			t.Errorf("%s: %v", alg, err)
		}
	}
}

// TestVerifyIDToken_Rejects: every failure fails verification, with a reason.
func TestVerifyIDToken_Rejects(t *testing.T) {
	srv, _ := keysServer(t)
	good := func() map[string]any { return idClaims(srv.URL, "client-1", time.Now().Add(time.Hour)) }
	with := func(k string, v any) map[string]any { c := good(); c[k] = v; return c }
	valid := signTestJWT(t, "ES256", "ec-1", good())
	parts := strings.Split(valid, ".")
	tampered := parts[0] + "." + b64([]byte(`{"iss":"`+srv.URL+`","aud":"client-1","sub":"attacker","exp":9999999999}`)) + "." + parts[2]
	base := idTokenCheck{issuer: srv.URL, clientID: "client-1", jwksURL: srv.URL + "/jwks"}

	for _, tc := range []struct {
		name  string
		raw   string
		check func(idTokenCheck) idTokenCheck
		want  string
	}{
		{"tampered payload", tampered, nil, "bad signature"},
		{"wrong audience", signTestJWT(t, "ES256", "ec-1", with("aud", "someone-else")), nil, "audience"},
		{"audience array without the client", signTestJWT(t, "ES256", "ec-1", with("aud", []string{"a", "b"})), nil, "audience"},
		{"wrong issuer", signTestJWT(t, "ES256", "ec-1", with("iss", "https://evil.example")), nil, "issuer"},
		{"expired", signTestJWT(t, "ES256", "ec-1", with("exp", time.Now().Add(-2*time.Hour).Unix())), nil, "expired"},
		{"no exp", signTestJWT(t, "ES256", "ec-1", map[string]any{"iss": srv.URL, "aud": "client-1"}), nil, "no exp"},
		{"alg none", signTestJWT(t, "none", "ec-1", good()), nil, `"none" is not allowed`},
		{"HS256", signTestJWT(t, "HS256", "ec-1", good()), nil, `"HS256" is not allowed`},
		{"no kid", signTestJWT(t, "ES256", "", good()), nil, "no key id"},
		{"alg the issuer does not advertise", signTestJWT(t, "RS256", "rsa-1", good()),
			func(c idTokenCheck) idTokenCheck { c.advertised = []string{"ES256"}; return c }, `"RS256" is not allowed`},
		{"key for another algorithm", signTestJWT(t, "ES256", "rsa-1", good()), nil, "not a P-256 key"},
		{"nonce mismatch", signTestJWT(t, "ES256", "ec-1", with("nonce", "other")),
			func(c idTokenCheck) idTokenCheck { c.nonce = "sent"; return c }, "nonce"},
		{"nonce missing", signTestJWT(t, "ES256", "ec-1", good()),
			func(c idTokenCheck) idTokenCheck { c.nonce = "sent"; return c }, "nonce"},
		{"no jwks_uri", valid, func(c idTokenCheck) idTokenCheck { c.jwksURL = ""; return c }, "no jwks_uri"},
		{"unreachable keys", valid, func(c idTokenCheck) idTokenCheck { c.jwksURL = srv.URL + "/broken"; return c }, "HTTP 500"},
		{"not a JWT", "abc.def", nil, "not a signed JWT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := base
			if tc.check != nil {
				check = tc.check(base)
			}
			err := verifyIDToken(context.Background(), srv.Client(), tc.raw, check, time.Now())
			if !errors.Is(err, errIDToken) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("verifyIDToken = %v, want an id_token rejection naming %q", err, tc.want)
			}
		})
	}
}

// TestVerifyIDToken_UnknownKidRefetchesOnce: a kid missing from the cached
// keys refetches them exactly once, then fails; a key the issuer has since
// published is found by that refetch.
func TestVerifyIDToken_UnknownKidRefetchesOnce(t *testing.T) {
	srv, ti := keysServer(t)
	check := idTokenCheck{issuer: srv.URL, clientID: "client-1", jwksURL: srv.URL + "/jwks"}
	claims := idClaims(srv.URL, "client-1", time.Now().Add(time.Hour))
	ctx := context.Background()

	ti.hideKid = "ec-1"
	if err := verifyIDToken(ctx, srv.Client(), signTestJWT(t, "RS256", "rsa-1", claims), check, time.Now()); err != nil {
		t.Fatal(err)
	}
	if hits := ti.jwksHits.Load(); hits != 1 {
		t.Fatalf("%d key fetches to populate the cache, want 1", hits)
	}
	err := verifyIDToken(ctx, srv.Client(), signTestJWT(t, "ES256", "ec-1", claims), check, time.Now())
	if !errors.Is(err, errIDToken) || ti.jwksHits.Load() != 2 {
		t.Fatalf("unknown kid: %v after %d fetches; want a rejection after exactly one refetch", err, ti.jwksHits.Load())
	}
	ti.mu.Lock()
	ti.hideKid = ""
	ti.mu.Unlock()
	if err := verifyIDToken(ctx, srv.Client(), signTestJWT(t, "ES256", "ec-1", claims), check, time.Now()); err != nil {
		t.Fatalf("a newly published key was not found by the refetch: %v", err)
	}
	if err := verifyIDToken(ctx, srv.Client(), signTestJWT(t, "ES256", "ec-1", claims), check, time.Now()); err != nil || ti.jwksHits.Load() != 3 {
		t.Errorf("cached key: %v after %d fetches; want no further fetch", err, ti.jwksHits.Load())
	}
}

// TestVerifiedSession_RequiresIDToken (0016-MADR A3): every login requests
// openid, so a token response without an id_token fails.
func TestVerifiedSession_RequiresIDToken(t *testing.T) {
	_, err := verifiedSession(context.Background(), oauthFlowConfig{provider: ProviderGrok, now: time.Now},
		oauthEndpoints{Token: "https://issuer.example/token", JWKS: "https://issuer.example/jwks"},
		oauthTokenResponse{AccessToken: "a", RefreshToken: "r"}, "")
	if !errors.Is(err, errIDToken) || !strings.Contains(err.Error(), "openid was requested") {
		t.Errorf("verifiedSession = %v, want a missing-id_token rejection", err)
	}
}

// failingDiscovery answers every request with 503.
func failingDiscovery() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable",
			Body: io.NopCloser(strings.NewReader("unavailable")), Header: make(http.Header), Request: request}, nil
	})}
}

// TestOAuthEndpointsFor_BuiltinIssuerFallsBack (0016-MADR D7): when the
// built-in issuer's discovery fails, its built-in endpoints and keys apply.
func TestOAuthEndpointsFor_BuiltinIssuerFallsBack(t *testing.T) {
	for _, tc := range []struct {
		provider, issuer string
		want             oauthEndpoints
	}{
		{ProviderGrok, DefaultGrokOAuthIssuer, oauthEndpoints{Authorization: DefaultGrokOAuthIssuer + "/oauth2/authorize",
			Token: defaultGrokOAuthRefreshURL, Device: defaultGrokOAuthDeviceURL, JWKS: defaultGrokJWKSURL}},
		{ProviderOpenAI, DefaultOpenAIIssuer, oauthEndpoints{Authorization: DefaultOpenAIIssuer + "/oauth/authorize",
			Token: DefaultOpenAIIssuer + "/oauth/token", JWKS: defaultOpenAIJWKSURL}},
	} {
		got, err := oauthEndpointsFor(context.Background(), oauthFlowConfig{provider: tc.provider, issuer: tc.issuer, httpClient: failingDiscovery()})
		if err != nil || got.Authorization != tc.want.Authorization || got.Token != tc.want.Token ||
			got.Device != tc.want.Device || got.JWKS != tc.want.JWKS {
			t.Errorf("%s: endpoints = %+v, %v; want %+v", tc.provider, got, err, tc.want)
		}
	}
}

// TestOAuthEndpointsFor_CallerIssuerDiscoveryFailureFails (0016-MADR D7,
// replacing the test that pinned finding M6's fallback): a caller's issuer
// whose discovery fails is an error, never the built-in endpoints.
func TestOAuthEndpointsFor_CallerIssuerDiscoveryFailureFails(t *testing.T) {
	for _, provider := range []string{ProviderGrok, ProviderOpenAI} {
		_, err := oauthEndpointsFor(context.Background(), oauthFlowConfig{provider: provider,
			issuer: "https://issuer.example", httpClient: failingDiscovery()})
		if err == nil || !strings.Contains(err.Error(), "503") {
			t.Errorf("%s: oauthEndpointsFor = %v, want the discovery failure", provider, err)
		}
	}
}
