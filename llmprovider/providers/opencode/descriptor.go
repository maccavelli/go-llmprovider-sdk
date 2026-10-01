package opencode

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// DescriptorZen is the OpenCode Zen descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func DescriptorZen() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderID(llmprovider.ProviderOpencodeZen),
		Label:           "OpenCode Zen",
		EnvVar:          llmprovider.ProviderEnvVars()[llmprovider.ProviderOpencodeZen],
		DefaultBaseURL:  zenBaseURL,
		SupportsBaseURL: true,
		RequiresAPIKey:  true,
		StaticModels:    catalog.Static(llmprovider.ProviderOpencodeZen),
		Notes:           "pay-as-you-go; free models available",
	}
}

// DescriptorGo is the OpenCode Go descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func DescriptorGo() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderID(llmprovider.ProviderOpencodeGo),
		Label:           "OpenCode Go",
		EnvVar:          llmprovider.ProviderEnvVars()[llmprovider.ProviderOpencodeGo],
		DefaultBaseURL:  goBaseURL,
		SupportsBaseURL: true,
		RequiresAPIKey:  true,
		StaticModels:    catalog.Static(llmprovider.ProviderOpencodeGo),
		Notes:           "subscription",
	}
}
