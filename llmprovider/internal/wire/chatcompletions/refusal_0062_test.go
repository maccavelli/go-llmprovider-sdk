package chatcompletions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// TestRefusal_Fixtures_0062 (0062-MADR D4): on every recorded failure of the
// Chat Completions wire, MatchRefusal reads exactly the refusals the MADR's
// amendment "Phase 1's evidence" lists, with their values and lists, and
// matches no failure of another cause; nor either of Ollama's, recorded in
// Phase 5, which has no matcher (the amendment "Phase 5's Ollama cases").
func TestRefusal_Fixtures_0062(t *testing.T) {
	all := []llmprovider.Effort{"low", "medium", "high", "xhigh", "max"}
	want := map[string]wire.Refusal{
		"huggingface-deepseek-v4.1-flash-bogus": {Effort: "bogus", Accepted: all, Listed: true},
		"kilo-gpt-6-luna-bogus":                 {Effort: "bogus", Accepted: all, Listed: true},
	}
	seen, ollama := 0, 0
	for _, r := range refusalFixtures(t, filepath.Join("..", "testdata", "refusals-0062", "chatcompletions")) {
		if strings.HasPrefix(r.Case, "ollama-") {
			ollama++
		}
		got, ok := MatchRefusal(r.err(), r.sent())
		w, refused := want[r.Case]
		switch {
		case refused && (!ok || got.Effort != w.Effort || got.Shape != w.Shape || got.Field != w.Field ||
			got.Listed != w.Listed || !slices.Equal(got.Accepted, w.Accepted)):
			t.Errorf("%s: %+v, %t; want %+v", r.Case, got, ok, w)
		case !refused && ok:
			t.Errorf("%s (%s %d %s): matched %+v; a failure of another cause", r.Case, r.Outcome, r.Status, r.Code, got)
		}
		if refused {
			seen++
		}
	}
	if seen != len(want) {
		t.Errorf("%d of the %d refusals were recorded", seen, len(want))
	}
	if ollama != 2 {
		t.Errorf("%d of Ollama's failures were read; want its two, bogus and the empty input, neither matched", ollama)
	}
}

// TestMatchRefusal_OtherList_0062 (0062-MADR D4, amendment "Phase 1's
// evidence"): a refusal of the value sent whose list is not made of effort
// words is another parameter's, and is not read as the effort's.
func TestMatchRefusal_OtherList_0062(t *testing.T) {
	err := &llmprovider.APIError{Provider: "p", Status: 400, Code: "invalid_request_error",
		Message: `Invalid option: expected one of "auto"|"concise"|"detailed"`}
	if got, ok := MatchRefusal(err, llmprovider.EffortMax); ok {
		t.Errorf("matched %+v; the list is another parameter's", got)
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

// sent is the effort the case sent: its own, else the recorder's request's,
// high.
func (r refusalFixture) sent() llmprovider.Effort {
	for _, key := range []string{"output_config.effort", "reasoning.effort", "generation_config.thinking_level", "reasoning_effort"} {
		if v, ok := r.Set[key].(string); ok {
			return llmprovider.Effort(v)
		}
	}
	return llmprovider.EffortHigh
}

// refusalFixtures are the recorded failures in dir, one wire's (0062-PLAN
// Phase 1).
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
