package catalog

import (
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestMetadata_ReasoningEffortsIsACopy (0026-MADR F53): the efforts returned
// are the caller's, so editing them does not change a later read of the same
// document, which every reader shares (R29).
func TestMetadata_ReasoningEffortsIsACopy(t *testing.T) {
	doc := `{"opencode":{"models":{"m1":{"id":"m1","reasoning_options":[{"type":"effort","values":["low","medium","high"]}]}}}}`
	first, err := ParseMetadata(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	efforts := first.ReasoningEfforts(llmprovider.ProviderOpencodeZen, "m1")
	if !slices.Equal(efforts, []string{"low", "medium", "high"}) {
		t.Fatalf("efforts = %q; want low, medium, high", efforts)
	}
	efforts[0] = "POISONED"
	if got := first.ReasoningEfforts(llmprovider.ProviderOpencodeZen, "m1"); got[0] != "low" {
		t.Fatalf("a later lookup = %q; want it unchanged by the caller's edit", got)
	}
}
