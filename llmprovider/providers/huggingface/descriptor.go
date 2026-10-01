package huggingface

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the Hugging Face descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderID(llmprovider.ProviderHuggingFace),
		Label:           "Hugging Face",
		EnvVar:          llmprovider.ProviderEnvVars()[llmprovider.ProviderHuggingFace],
		DefaultBaseURL:  defaultBaseURL,
		SupportsBaseURL: true,
		RequiresAPIKey:  true,
		StaticModels:    llmprovider.StaticModels(llmprovider.ProviderHuggingFace),
		Notes:           "monthly credits; no free tier",
	}
}
