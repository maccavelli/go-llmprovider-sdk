package claude

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the Claude (Anthropic) descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:             llmprovider.ProviderClaude,
		Label:          "Claude (Anthropic)",
		EnvVar:         llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude],
		RequiresAPIKey: true,
		StaticModels:   catalog.Static(llmprovider.ProviderClaude),
	}
}
