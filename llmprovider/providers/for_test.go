package providers

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/kilo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// TestFor_OneListBuildsEveryProvider (0015-MADR D5 steps 2-3): one option
// list, with each provider's own options inside For, builds every built-in
// provider; the same options given bare are refused by the others.
func TestFor_OneListBuildsEveryProvider(t *testing.T) {
	overlays := []llmprovider.Option{
		llmprovider.For(llmprovider.ProviderKilo, kilo.WithOrganization("org-1"), catalog.WithKiloOrganization("org-1")),
		llmprovider.For(llmprovider.ProviderOpencodeZen, opencode.WithRoute(opencode.RouteMessages)),
		llmprovider.For(llmprovider.ProviderOpencodeGo, opencode.WithRoute(opencode.RouteChatCompletions)),
		llmprovider.For(llmprovider.ProviderOpenAI, openai.WithStore(true)),
		llmprovider.For(llmprovider.ProviderGemini, gemini.WithStore(true)),
		llmprovider.For(llmprovider.ProviderGrok, grok.WithStore(false)),
	}
	registry := Default()
	for _, d := range registry.Descriptors() {
		shared := []llmprovider.Option{llmprovider.WithModel("m"), catalog.WithProfile(catalog.ProfileCapable)}
		if d.RequiresAPIKey {
			shared = append(shared, llmprovider.WithAPIKey("k"))
		}
		if _, err := registry.New(d.ID, append(shared, overlays...)...); err != nil {
			t.Errorf("%s: New with the shared list: %v", d.ID, err)
		}
		if d.ID != llmprovider.ProviderKilo {
			_, err := registry.New(d.ID, append(shared, kilo.WithOrganization("org-1"))...)
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("%s: New with a bare kilo.WithOrganization: err = %v, want it refused", d.ID, err)
			}
		}
	}
}
