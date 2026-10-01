package llmprovider

import (
	"testing"
	"time"
)

// The tests of what stayed here when OpenCode moved to providers/opencode
// (0015-PLAN S7): the gateway base URLs the listing uses, the catalog filter,
// and two checks that were in opencode_route_test.go but test this package.

// TestOpencodeBaseURL verifies the documented default base URLs.
func TestOpencodeBaseURL(t *testing.T) {
	zen, err := opencodeBaseURL(ProviderOpencodeZen)
	if err != nil || zen != "https://opencode.ai/zen/v1" {
		t.Errorf("zen base = %q, err = %v", zen, err)
	}
	goURL, err := opencodeBaseURL(ProviderOpencodeGo)
	if err != nil || goURL != "https://opencode.ai/zen/go/v1" {
		t.Errorf("go base = %q, err = %v", goURL, err)
	}
	if _, err := opencodeBaseURL("nope"); err == nil {
		t.Error("expected error for unknown gateway")
	}
}

// TestProviderConstants_Distinct guards against a copy-paste error in the
// canonical identifier block.
func TestProviderConstants_Distinct(t *testing.T) {
	ids := []string{
		ProviderGemini, ProviderOpenAI, ProviderClaude, ProviderGrok,
		ProviderOpencodeZen, ProviderOpencodeGo, ProviderHuggingFace, ProviderKilo,
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			t.Error("empty provider identifier")
		}
		if _, dup := seen[id]; dup {
			t.Errorf("duplicate provider identifier: %q", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != 8 {
		t.Errorf("expected 8 distinct provider identifiers, got %d", len(seen))
	}
}

// TestWireShapesProbedOn validates every gateway probe-date pin still in this
// package; opencode's is tested in providers/opencode. Its second job is to
// give those constants a real, untagged use: golangci-lint analyses test
// files (run: tests: true) but not //go:build live_gateways files, so a
// reference from the live suite would not satisfy `unused`. See 0004-PLAN
// deviation D2.
func TestWireShapesProbedOn(t *testing.T) {
	pins := map[string]string{
		"huggingface": wireShapesProbedOnHuggingFace,
		"kilo":        wireShapesProbedOnKilo,
		"ollama":      wireShapesProbedOnOllama,
	}
	for name, pin := range pins {
		t.Run(name, func(t *testing.T) {
			d, err := time.Parse(time.DateOnly, pin)
			if err != nil {
				t.Fatalf("wire-shape pin %q is not a YYYY-MM-DD date: %v", pin, err)
			}
			if d.After(time.Now()) {
				t.Errorf("wire-shape pin %q is in the future", pin)
			}
		})
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
