package grok

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// 0041-PLAN Phase 0, deviation 2: 0041-MADR §3's R6, asserted here because
// effortMenu is unexported and this package imports catalog. It reads the
// snapshot catalog's evidence tests read.

// TestEvidence0041_GrokMenus is R6: effortMenu equals models.dev's effort
// values on 5 of xAI's 7 reasoning models. grok-4.3's menu has xhigh, which
// xAI's listing gives and a request accepted, where models.dev has none and
// no xhigh; grok-4.20-multi-agent-0309 has efforts in models.dev that the
// menu omits.
func TestEvidence0041_GrokMenus(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "catalog", "testdata", "ranking-2026-10-12", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]struct {
		Models map[string]struct {
			Reasoning        *bool `json:"reasoning"`
			ReasoningOptions []struct {
				Type   string   `json:"type"`
				Values []string `json:"values"`
			} `json:"reasoning_options"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	agree, seen := 0, 0
	differ := map[string][2][]string{}
	for id, m := range doc["xai"].Models {
		if m.Reasoning == nil || !*m.Reasoning {
			continue
		}
		seen++
		var data []string
		for _, o := range m.ReasoningOptions {
			if o.Type == "effort" {
				data = append(data, o.Values...)
			}
		}
		rule := slices.Clone(effortMenu(id))
		slices.Sort(rule)
		slices.Sort(data)
		if slices.Equal(rule, data) {
			agree++
		} else {
			differ[id] = [2][]string{rule, data}
		}
	}
	want := map[string][2][]string{
		"grok-4.3":                   {{"high", "low", "medium", "xhigh"}, {"high", "low", "medium", "none"}},
		"grok-4.20-multi-agent-0309": {nil, {"high", "low", "medium", "xhigh"}},
	}
	if agree != 5 || seen != 7 || len(differ) != len(want) {
		t.Fatalf("R6: %d of %d agree, differing %v; want 5 of 7, differing %v", agree, seen, differ, want)
	}
	for id, w := range want {
		if got, ok := differ[id]; !ok || !slices.Equal(got[0], w[0]) || !slices.Equal(got[1], w[1]) {
			t.Errorf("R6: %s: menu and models.dev %v; want %v", id, got, w)
		}
	}
}
