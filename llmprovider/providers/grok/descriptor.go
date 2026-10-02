package grok

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the Grok (xAI) descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:             llmprovider.ProviderGrok,
		Label:          "Grok (xAI)",
		EnvVar:         llmprovider.ProviderEnvVars()[llmprovider.ProviderGrok],
		RequiresAPIKey: true,
		AuthMethods: []llmprovider.AuthMethod{
			{
				ID:          llmprovider.AuthAPIKey,
				Label:       "xAI API key",
				Detail:      "console.x.ai billing (`api.x.ai`)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthBrowserOAuth,
				Label:       "Sign in with xAI",
				Interactive: true,
			},
			{
				ID:          llmprovider.AuthDeviceCode,
				Label:       "Sign in with xAI (device code)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthTokenStdin,
				Label:       "Paste an xAI API key",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthImportVendorCLI,
				Label:       "Use the Grok CLI login (~/.grok/auth.json)",
				Interactive: true,
				HeadlessOK:  true,
			},
		},
		StaticModels: catalog.Static(llmprovider.ProviderGrok),
	}
}
