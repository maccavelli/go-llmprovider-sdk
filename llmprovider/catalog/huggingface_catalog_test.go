package catalog

import "testing"

// The catalog tests that were in huggingface_test.go, which stay with the
// catalog when the provider moved to providers/huggingface (0015-PLAN S7).

func TestSplitHuggingFaceModelPolicy(t *testing.T) {
	tests := []struct{ id, wantBase, wantPolicy string }{
		{"openai/gpt-oss-120b", "openai/gpt-oss-120b", ""},
		{"openai/gpt-oss-120b:groq", "openai/gpt-oss-120b", "groq"},
		{"openai/gpt-oss-120b:cheapest", "openai/gpt-oss-120b", "cheapest"},
		{"openai/gpt-oss-120b:fastest", "openai/gpt-oss-120b", "fastest"},
		{"no-slash", "no-slash", ""},
	}
	for _, tc := range tests {
		base, policy := splitHuggingFaceModelPolicy(tc.id)
		if base != tc.wantBase || policy != tc.wantPolicy {
			t.Errorf("split(%q) = (%q,%q), want (%q,%q)", tc.id, base, policy, tc.wantBase, tc.wantPolicy)
		}
	}
}

func TestStaticHuggingFace_Count(t *testing.T) {
	if len(staticHuggingFace) == 0 || len(staticHuggingFace) > MaxListed {
		t.Errorf("staticHuggingFace has %d entries, want 1..%d", len(staticHuggingFace), MaxListed)
	}
	for _, m := range staticHuggingFace {
		if !isUsableHuggingFaceModel(m) {
			t.Errorf("%q fails its own usability filter", m)
		}
	}
}
