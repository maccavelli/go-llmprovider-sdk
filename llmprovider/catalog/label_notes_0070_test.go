package catalog

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// relativeTime matches the words that place a model in time, which a newer
// release makes false, as whole words, ignoring case.
var relativeTime = regexp.MustCompile(`(?i)\b(current|previous|newest|newer|latest|older|oldest|last)\b`)

// TestLabelNotes_NoRelativeTime_0070 (0070-MADR D2, D3): no curated note says
// where its model stands in time, and grok-4.6's no longer calls it the Grok
// CLI's default (issue 11, 0070-MADR C22). It reads the bracketed part of
// Label and ranges only the keys of modelLabels, so that it builds at
// b5ce51d and is seen red there (0070-PLAN Phase 2).
func TestLabelNotes_NoRelativeTime_0070(t *testing.T) {
	for _, p := range slices.Sorted(maps.Keys(modelLabels)) {
		for _, id := range slices.Sorted(maps.Keys(modelLabels[p])) {
			label := Label(p, id)
			_, note, ok := strings.Cut(label, "[")
			if !ok {
				continue
			}
			if words := relativeTime.FindAllString(note, -1); len(words) > 0 {
				t.Errorf("%s %s: note %q places the model in time with %q", p, id, strings.TrimSuffix(note, "]"), words)
			}
		}
	}
	if got := Label(llmprovider.ProviderGrok, grokModel46); strings.Contains(got, "the Grok CLI's default]") {
		t.Errorf("Label(grok, %s) = %q, want no claim that it is the Grok CLI's default", grokModel46, got)
	}
}
