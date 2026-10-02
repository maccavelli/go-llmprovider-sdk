package llmprovider

import "testing"

// TestProviderEnvVars_Opencode pins the single evidenced credential name: both
// gateways share OPENCODE_API_KEY (models.dev declares it for each; neither
// docs page names any other variable).
func TestProviderEnvVars_Opencode(t *testing.T) {
	for _, gw := range []ProviderID{ProviderOpencodeZen, ProviderOpencodeGo} {
		got, ok := providerEnvVars[gw]
		if !ok {
			t.Errorf("providerEnvVars missing %q", gw)
			continue
		}
		if got != "OPENCODE_API_KEY" {
			t.Errorf("providerEnvVars[%q] = %q, want OPENCODE_API_KEY", gw, got)
		}
	}
}
