//go:build live_gateways

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestLive_BuiltinIssuersPublishVerifiableKeys pins 0016-MADR "D7 as
// decided": both built-in issuers' discovery documents name the JWKS this
// package falls back to, and advertise only algorithms it verifies with the
// standard library (RS256 for OpenAI, ES256 for xAI, probed 2026-09-30). A
// change on either side fails here, loudly. It reads public documents only.
func TestLive_BuiltinIssuersPublishVerifiableKeys(t *testing.T) {
	for _, tc := range []struct{ issuer, jwks string }{
		{DefaultOpenAIIssuer, defaultOpenAIJWKSURL},
		{DefaultGrokOAuthIssuer, defaultGrokJWKSURL},
	} {
		t.Run(tc.issuer, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, tc.issuer+"/.well-known/openid-configuration", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := transport.DefaultClient().Do(req)
			if err != nil {
				t.Fatalf("discovery: %v", err)
			}
			defer closeResponseBody(resp)
			var doc struct {
				Issuer string   `json:"issuer"`
				JWKS   string   `json:"jwks_uri"`
				Algs   []string `json:"id_token_signing_alg_values_supported"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
				t.Fatalf("decode discovery: %v", err)
			}
			if doc.Issuer != tc.issuer || doc.JWKS != tc.jwks {
				t.Errorf("issuer %q jwks_uri %q; want %q and %q", doc.Issuer, doc.JWKS, tc.issuer, tc.jwks)
			}
			if len(doc.Algs) == 0 {
				t.Error("no id_token_signing_alg_values_supported")
			}
			for _, alg := range doc.Algs {
				if !slices.Contains([]string{"RS256", "ES256"}, alg) {
					t.Errorf("advertises %q, outside RS256 and ES256", alg)
				}
			}
			t.Logf("%s: jwks_uri %s, algorithms %v", tc.issuer, doc.JWKS, doc.Algs)
		})
	}
}
