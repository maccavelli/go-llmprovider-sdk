//go:build live_gateways

package catalog

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// liveOllamaURL is the local Ollama the live check reads.
const liveOllamaURL = "http://localhost:11434"

// TestLive_OllamaWritesText_0042 checks 0042-MADR D1 and D2 on the local
// Ollama, when one answers. Ollama takes no key, and nothing here generates
// or is billed:
//
//  1. every model in Usable writes text, or Ollama does not say;
//  2. every installed model left out says, through /api/show or /api/tags,
//     that it writes no text;
//  3. wherever both endpoints report capabilities, they agree on whether
//     the model writes text;
//  4. every listed model whose only caveat is no tools is recommended.
func TestLive_OllamaWritesText_0042(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	probe, probeCancel := context.WithTimeout(ctx, time.Second)
	req, err := http.NewRequestWithContext(probe, http.MethodGet, liveOllamaURL+"/api/tags", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	probeCancel()
	if err != nil {
		t.Skip("no local Ollama server answered")
	}
	_ = resp.Body.Close()

	resetLookupCaches(t)
	cfg := testConfig(t, llmprovider.WithBaseURL(liveOllamaURL))
	token := llmprovider.Token{}
	tags, err := fetchOllamaTags(ctx, token, cfg)
	if err != nil {
		t.Fatalf("/api/tags: %v", err)
	}
	shows := map[string]ModelInfo{}
	failed := 0
	for _, tag := range tags {
		info, err := fetchOllamaShow(ctx, token, cfg, tag.Name)
		if err != nil {
			t.Logf("%s: /api/show failed: %v", tag.Name, err)
			failed++
			continue
		}
		shows[tag.Name] = info
	}
	cat, err := List(ctx, llmprovider.ProviderOllama, llmprovider.NewStaticToken(""),
		llmprovider.WithBaseURL(liveOllamaURL), llmprovider.WithoutModelMetadata())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, e := range cat.Menu {
		if writesNoText(e.Info) {
			t.Errorf("%s is listed, but its facts say it writes no text", e.ID)
		}
	}
	left := 0
	for _, tag := range tags {
		tagInfo, show := ollamaTagInfo(tag), shows[tag.Name]
		if !slices.Contains(cat.Usable, tag.Name) {
			left++
			if !writesNoText(show) && !writesNoText(tagInfo) {
				t.Errorf("%s is left out, but neither endpoint says it writes no text (tags %v, show %v)",
					tag.Name, tag.Capabilities, show.Output)
			}
		}
		if tagInfo.Has(FieldOutput) && show.Has(FieldOutput) && writesNoText(tagInfo) != writesNoText(show) {
			t.Errorf("%s: /api/tags (%v) and /api/show (%v) disagree on writing text", tag.Name, tag.Capabilities,
				show.Output)
		}
	}
	toolless := 0
	for _, e := range cat.Menu {
		if slices.Equal(e.Caveats, []Caveat{CaveatNoTools}) {
			toolless++
			if !slices.Contains(cat.Recommended, e.ID) {
				t.Errorf("%s takes no tools and has no other caveat, but is not recommended", e.ID)
			}
		}
	}
	t.Logf("installed %d, listed %d, left out %d, /api/show failed %d, tool-less recommended %d",
		len(tags), len(cat.Usable), left, failed, toolless)
}
