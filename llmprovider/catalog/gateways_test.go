package catalog

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Moved from llmprovider's opencode_gateway_test.go (0015-PLAN S8, commit 2).

// TestOpencodeBaseURL verifies the documented default base URLs.
func TestOpencodeBaseURL(t *testing.T) {
	zen, err := opencodeBaseURL(llmprovider.ProviderOpencodeZen)
	if err != nil || zen != "https://opencode.ai/zen/v1" {
		t.Errorf("zen base = %q, err = %v", zen, err)
	}
	goURL, err := opencodeBaseURL(llmprovider.ProviderOpencodeGo)
	if err != nil || goURL != "https://opencode.ai/zen/go/v1" {
		t.Errorf("go base = %q, err = %v", goURL, err)
	}
	if _, err := opencodeBaseURL("nope"); err == nil {
		t.Error("expected error for unknown gateway")
	}
}

// TestIsUsableOpencodeModel_DeniesSystemone: jev-* models use OpenCode's
// "systemone" route, which has no encoder (MADR 0012 §3.1).
func TestIsUsableOpencodeModel_DeniesSystemone(t *testing.T) {
	for _, id := range []string{"jev-1.13", "jev-1.13-free", "JEV-2"} {
		if isUsableOpencodeModel(id) {
			t.Errorf("isUsableOpencodeModel(%q) = true, want false", id)
		}
	}
	if !isUsableOpencodeModel("kimi-k2.6") {
		t.Error("isUsableOpencodeModel(kimi-k2.6) = false, want true")
	}
}
