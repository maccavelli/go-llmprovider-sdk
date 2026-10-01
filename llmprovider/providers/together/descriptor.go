package together

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the Together AI descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderID(llmprovider.ProviderTogether),
		Label:           "Together AI",
		EnvVar:          llmprovider.ProviderEnvVars()[llmprovider.ProviderTogether],
		DefaultBaseURL:  defaultBaseURL,
		SupportsBaseURL: true,
		RequiresAPIKey:  true,
		StaticModels:    catalog.Static(llmprovider.ProviderTogether),
		Notes:           "pay-as-you-go",
	}
}
