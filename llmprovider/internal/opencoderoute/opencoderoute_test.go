package opencoderoute

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOf: the table's route for a listed model, matched without case or
// surrounding space, else the heuristic's (0089-PLAN deviation 2). The
// provider package's tests pin every snapshot row and the heuristic in full.
func TestOf(t *testing.T) {
	for _, c := range []struct {
		gateway llmprovider.ProviderID
		model   string
		want    string
	}{
		{llmprovider.ProviderOpencodeZen, "claude-opus-4-6", Messages},
		{llmprovider.ProviderOpencodeZen, " Claude-Opus-4-6 ", Messages},
		{llmprovider.ProviderOpencodeZen, "claude-not-listed", Messages},
		{llmprovider.ProviderOpencodeZen, "gemini-not-listed", Google},
		{llmprovider.ProviderOpencodeGo, "claude-not-listed", ChatCompletions},
		{llmprovider.ProviderOpencodeGo, "minimax-not-listed", Messages},
		{llmprovider.ProviderOpencodeGo, "gpt-not-listed", Responses},
		{llmprovider.ProviderOpencodeZen, "qwen9-max", ChatCompletions},
		{llmprovider.ProviderOpencodeZen, "qwen9-flash", Messages},
		{llmprovider.ProviderOpencodeZen, "qwen9", Messages},
		{llmprovider.ProviderOpencodeGo, "qwen9", ChatCompletions},
		{llmprovider.ProviderOpencodeZen, "other", ChatCompletions},
		{llmprovider.ProviderKilo, "claude-opus-4-6", ChatCompletions},
	} {
		if got := Of(c.gateway, c.model); got != c.want {
			t.Errorf("Of(%s, %q) = %q, want %q", c.gateway, c.model, got, c.want)
		}
	}
}

// TestForNPM: each AI SDK package's route, and the chat route for any other.
func TestForNPM(t *testing.T) {
	for npm, want := range map[string]string{"@ai-sdk/openai": Responses, "@ai-sdk/anthropic": Messages,
		"@ai-sdk/google": Google, "": ChatCompletions, "@ai-sdk/other": ChatCompletions} {
		if got := ForNPM(npm); got != want {
			t.Errorf("ForNPM(%q) = %q, want %q", npm, got, want)
		}
	}
}

// TestBuildTable: the embedded snapshot builds, both gateways in it; a
// malformed snapshot, or one with a section for no gateway, panics.
func TestBuildTable(t *testing.T) {
	if len(Table[llmprovider.ProviderOpencodeZen]) == 0 || len(Table[llmprovider.ProviderOpencodeGo]) == 0 {
		t.Fatalf("the table lacks a gateway: %d sections", len(Table))
	}
	for name, raw := range map[string]string{"malformed": "{", "unknown section": `{"other":{"m":""}}`} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("BuildTable accepted it")
				}
			}()
			BuildTable([]byte(raw))
		})
	}
}
