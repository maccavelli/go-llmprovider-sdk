package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
)

// TestVerifyIDToken_StaleKeysForAReusedURL_0083 (ST64, ST65): a server on a port
// an earlier server held finds no entry of the earlier one's keys under its own
// JWKS URL, so the test that counts its fetches sees one to populate the cache,
// one refetch for the unknown kid, and no further fetch once it is cached.
func TestVerifyIDToken_StaleKeysForAReusedURL_0083(t *testing.T) {
	port := nettest.Port(t)
	staleURL := fmt.Sprintf("http://127.0.0.1:%d/jwks", port)
	// An earlier server's entry, fresh, holding the kids this test verifies.
	jwksCache.Lock()
	jwksCache.entries[staleURL] = jwksEntry{keys: map[string]jwk{"rsa-1": {}, "ec-1": {}}, fetched: time.Now()}
	jwksCache.Unlock()
	t.Cleanup(func() {
		jwksCache.Lock()
		delete(jwksCache.entries, staleURL)
		jwksCache.Unlock()
	})
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	srv, ti := keysServerOn(t, listener)
	if srv.URL+"/jwks" != staleURL {
		t.Fatalf("the server's JWKS URL = %s; want the stale entry's %s", srv.URL+"/jwks", staleURL)
	}
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
	err = verifyIDToken(ctx, srv.Client(), signTestJWT(t, "ES256", "ec-1", claims), check, time.Now())
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
