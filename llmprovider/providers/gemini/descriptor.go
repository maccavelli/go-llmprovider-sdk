package gemini

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the Gemini (Google) descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:             llmprovider.ProviderGemini,
		Label:          "Gemini (Google)",
		EnvVar:         llmprovider.ProviderEnvVars()[llmprovider.ProviderGemini],
		RequiresAPIKey: true,
		StaticModels:   catalog.Static(llmprovider.ProviderGemini),
	}
}
