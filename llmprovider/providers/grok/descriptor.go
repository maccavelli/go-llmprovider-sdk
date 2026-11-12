package grok

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// The method facts below repeat these values of the sign-in contract
// (0074-MADR D1); they are strings here so this package imports no v0 module.
const (
	factAPIKey = "api_key"
	factNone   = "none"
)

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
				ID:             llmprovider.AuthAPIKey,
				Label:          "xAI API key",
				Detail:         "console.x.ai billing (`api.x.ai`)",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           factAPIKey,
				Completion:     "value",
				Yields:         factAPIKey,
				ClientIdentity: factNone,
			},
			{
				ID:             llmprovider.AuthBrowserOAuth,
				Label:          "Sign in with xAI",
				Interactive:    true,
				Kind:           "browser",
				Completion:     "browser_loopback",
				Yields:         "oauth_session",
				ClientIdentity: "vendor_borrowed",
				Slot:           "grok",
				Refreshable:    true,
				Revocable:      true,
			},
			{
				ID:             llmprovider.AuthDeviceCode,
				Label:          "Sign in with xAI (device code)",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "device_code",
				Completion:     "device",
				Yields:         "oauth_session",
				ClientIdentity: "vendor_borrowed",
				Slot:           "grok",
				Refreshable:    true,
				Revocable:      true,
			},
			{
				ID:             llmprovider.AuthTokenStdin,
				Label:          "Paste an xAI API key",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "token_paste",
				Completion:     "value",
				Yields:         factAPIKey,
				ClientIdentity: factNone,
			},
			{
				ID:             llmprovider.AuthImportVendorCLI,
				Label:          "Use the Grok CLI login (~/.grok/auth.json)",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "vendor_import",
				Completion:     "local_file",
				Yields:         "read_through",
				ClientIdentity: factNone,
			},
		},
		StaticModels: catalog.Static(llmprovider.ProviderGrok),
	}
}
