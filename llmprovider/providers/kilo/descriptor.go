package kilo

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// defaultBaseURL is Kilo's gateway, as llmprovider's kiloBaseURL, for the
// descriptor.
const defaultBaseURL = "https://api.kilo.ai/api/gateway"

// Descriptor is the Kilo Gateway descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderKilo,
		Label:           "Kilo Gateway",
		EnvVar:          llmprovider.ProviderEnvVars()[llmprovider.ProviderKilo],
		DefaultBaseURL:  defaultBaseURL,
		SupportsBaseURL: true,
		RequiresAPIKey:  true,
		AuthMethods: []llmprovider.AuthMethod{
			{
				ID:          llmprovider.AuthAPIKey,
				Label:       "Kilo API key",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				// 0017-MADR D2: the token never refreshes and is used as the key.
				ID:          llmprovider.AuthDeviceCode,
				Label:       "Sign in with Kilo (device code)",
				Interactive: true,
				HeadlessOK:  true,
			},
		},
		StaticModels: catalog.Static(llmprovider.ProviderKilo),
		Notes:        "free models available",
	}
}
