package ollama

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the Ollama (local) descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderID(llmprovider.ProviderOllama),
		Label:           "Ollama (local)",
		DefaultBaseURL:  defaultBaseURL,
		SupportsBaseURL: true,
		IsLocal:         true,
		StaticModels:    llmprovider.StaticModels(llmprovider.ProviderOllama),
		Notes:           "runs on your machine; no API key",
	}
}
