package openai

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// Descriptor is the OpenAI descriptor: what a configuration UI shows for
// it (0015-MADR D10).
func Descriptor() llmprovider.Descriptor {
	return llmprovider.Descriptor{
		ID:             llmprovider.ProviderOpenAI,
		Label:          "OpenAI",
		EnvVar:         llmprovider.ProviderEnvVars()[llmprovider.ProviderOpenAI],
		RequiresAPIKey: true,
		AuthMethods: []llmprovider.AuthMethod{
			{
				ID:          llmprovider.AuthAPIKey,
				Label:       "OpenAI API key",
				Detail:      "Platform billing (`api.openai.com`)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthBrowserOAuth,
				Label:       "Sign in with ChatGPT",
				Detail:      "Plus/Pro/Business/Edu/Enterprise plan",
				Interactive: true,
			},
			{
				ID:          llmprovider.AuthDeviceCode,
				Label:       "Sign in with ChatGPT (device code)",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthTokenStdin,
				Label:       "Paste a ChatGPT access token or API key",
				Interactive: true,
				HeadlessOK:  true,
			},
			{
				ID:          llmprovider.AuthImportVendorCLI,
				Label:       "Use the Codex CLI login (~/.codex/auth.json)",
				Interactive: true,
				HeadlessOK:  true,
			},
		},
		StaticModels: catalog.Static(llmprovider.ProviderOpenAI),
	}
}
