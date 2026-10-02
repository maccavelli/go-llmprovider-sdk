//go:build live_gateways

package auth

import (
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_GrokDiscoveryPublishesRevocation: xAI's discovery document names
// a revocation endpoint (https://auth.x.ai/oauth2/revoke on 2026-09-27),
// which RevokeOAuthSession uses. Read-only: nothing is revoked.
func TestLive_GrokDiscoveryPublishesRevocation(t *testing.T) {
	ctx, cancel := liveCtx(t)
	defer cancel()
	endpoints, err := oauthEndpointsFor(ctx, oauthFlowConfig{provider: llmprovider.ProviderGrok, issuer: DefaultGrokOAuthIssuer,
		httpClient: http.DefaultClient})
	if err != nil {
		t.Skipf("discovery unreachable: %v", err)
	}
	if endpoints.Revocation != DefaultGrokOAuthIssuer+"/oauth2/revoke" {
		t.Fatalf("revocation_endpoint = %q", endpoints.Revocation)
	}
}
