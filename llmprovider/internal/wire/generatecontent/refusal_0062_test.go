package generatecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// TestRefusal_Fixtures_0062 (0062-MADR D4, amendment "Phase 3's google-route
// levels, and 0013's B9"): on every recorded failure of the generateContent
// wire, MatchRefusal reads exactly the refusal of xhigh, the level sent with no
// list, and matches no failure of another cause: the empty contents, the
// string schema, and gemini-2.5-flash's UNKNOWN "Model is unavailable."
func TestRefusal_Fixtures_0062(t *testing.T) {
	want := map[string]wire.Refusal{
		"opencode-zen-google-gemini-3.8-flash-level-xhigh": {Effort: "xhigh"},
	}
	seen, failures := 0, 0
	for _, r := range refusalFixtures(t, filepath.Join("..", "testdata", "refusals-0062", "generatecontent")) {
		failures++
		got, ok := MatchRefusal(r.err(), r.sent())
		w, refused := want[r.Case]
		switch {
		case refused && (!ok || got.Effort != w.Effort || got.Shape != w.Shape || got.Field != w.Field ||
			got.Listed != w.Listed || len(got.Accepted) != 0):
			t.Errorf("%s: %+v, %t; want %+v", r.Case, got, ok, w)
		case !refused && ok:
			t.Errorf("%s (%s %d %s): matched %+v; a failure of another cause", r.Case, r.Outcome, r.Status, r.Code, got)
		}
		if refused {
			seen++
		}
	}
	if seen != len(want) || failures != 4 {
		t.Errorf("%d of the %d refusals, among %d failures, were recorded; want among 4", seen, len(want), failures)
	}
}

// TestMatchRefusal_Bounds_0062 (0062-MADR D4): the level's refusal is matched
// only as a 400 INVALID_ARGUMENT, and only when an effort was sent.
func TestMatchRefusal_Bounds_0062(t *testing.T) {
	msg := "Upstream request failed: [400] Invalid value at 'generation_config.thinking_config.thinking_level'"
	for _, tc := range []struct {
		name   string
		status int
		code   string
		sent   llmprovider.Effort
	}{
		{"another status", 500, "INVALID_ARGUMENT", "xhigh"},
		{"another code", 400, "UNKNOWN", "xhigh"},
		{"no effort sent", 400, "INVALID_ARGUMENT", ""},
	} {
		err := &llmprovider.APIError{Provider: "p", Status: tc.status, Code: tc.code, Message: msg}
		if got, ok := MatchRefusal(err, tc.sent); ok {
			t.Errorf("%s: matched %+v", tc.name, got)
		}
	}
	err := &llmprovider.APIError{Provider: "p", Status: 400, Code: "INVALID_ARGUMENT", Message: msg}
	if got, ok := MatchRefusal(err, "xhigh"); !ok || got.Effort != "xhigh" {
		t.Errorf("the refusal: %+v, %t; want xhigh", got, ok)
	}
}

// refusalFixture is the part of a 0062 record a matcher reads.
type refusalFixture struct {
	Case    string         `json:"case"`
	Set     map[string]any `json:"set"`
	Outcome string         `json:"outcome"`
	Status  int            `json:"status"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
}

// err is the *APIError the record's reply made.
func (r refusalFixture) err() error {
	return &llmprovider.APIError{Provider: "p", Status: r.Status, Code: r.Code, Message: r.Message}
}

// sent is the effort the case sent: its level, else the recorder's request's,
// high.
func (r refusalFixture) sent() llmprovider.Effort {
	if v, ok := r.Set["generationConfig.thinkingConfig.thinkingLevel"].(string); ok {
		return llmprovider.Effort(v)
	}
	return llmprovider.EffortHigh
}

// refusalFixtures are the recorded failures in dir, one wire's (0062-PLAN
// Phase 1, Phase 3).
func refusalFixtures(t *testing.T, dir string) []refusalFixture {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no fixtures in %s: %v", dir, err)
	}
	var out []refusalFixture
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			t.Fatal(err)
		}
		var r refusalFixture
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if r.Status/100 != 2 {
			out = append(out, r)
		}
	}
	return out
}
