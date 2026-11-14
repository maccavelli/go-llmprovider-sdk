package openai

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The descriptor moved here from llmprovider's table (0015-PLAN S8, commit 1).

// The method facts below repeat these values of the sign-in contract
// (0074-MADR D1); they are strings here so this package imports no v0 module.
const (
	factOAuthSession = "oauth_session"
	factNone         = "none"
	factOpenAI       = "openai"
)

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
				ID:             llmprovider.AuthAPIKey,
				Label:          "OpenAI API key",
				Detail:         "Platform billing (`api.openai.com`)",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "api_key",
				Completion:     "value",
				Yields:         "api_key",
				ClientIdentity: factNone,
			},
			{
				ID:             llmprovider.AuthBrowserOAuth,
				Label:          "Sign in with ChatGPT",
				Detail:         "Plus/Pro/Business/Edu/Enterprise plan",
				Interactive:    true,
				Kind:           "browser",
				Completion:     "browser_paste",
				Paste:          "url_or_code",
				Yields:         factOAuthSession,
				ClientIdentity: "vendor_borrowed",
				Slot:           factOpenAI,
				Refreshable:    true,
				Revocable:      true,
			},
			{
				ID:             llmprovider.AuthDeviceCode,
				Label:          "Sign in with ChatGPT (device code)",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "device_code",
				Completion:     "device",
				Yields:         factOAuthSession,
				ClientIdentity: "vendor_borrowed",
				Slot:           factOpenAI,
				Refreshable:    true,
				Revocable:      true,
			},
			{
				ID:             llmprovider.AuthTokenStdin,
				Label:          "Paste a ChatGPT access token or API key",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "token_paste",
				Completion:     "value",
				Yields:         "access_only",
				ClientIdentity: factNone,
				Slot:           factOpenAI,
				Revocable:      true,
			},
			{
				ID:             llmprovider.AuthImportVendorCLI,
				Label:          "Use the Codex CLI login (~/.codex/auth.json)",
				Interactive:    true,
				HeadlessOK:     true,
				Kind:           "vendor_import",
				Completion:     "local_file",
				Yields:         "read_through",
				ClientIdentity: factNone,
			},
			// Last, so that every earlier method keeps its place; the wizard
			// offers it only with the consumer's app name (0043-MADR D7).
			{
				ID:             llmprovider.AuthChatGPTPlan,
				Label:          "Use your ChatGPT plan with this app",
				Detail:         "Sign in with ChatGPT for agents; usage counts toward your plan",
				Interactive:    true,
				Kind:           "browser",
				Completion:     "browser_paste",
				Paste:          "url_or_code",
				FirstPaste:     "url_only",
				Consent:        ConsentBackgroundUse,
				ConsentText:    consentBackgroundText,
				ConsentURL:     ChatGPTUsageURL,
				Yields:         factOAuthSession,
				ClientIdentity: "app_registered",
				Slot:           factOpenAI,
				Refreshable:    true,
				Revocable:      true,
			},
		},
		StaticModels: catalog.Static(llmprovider.ProviderOpenAI),
	}
}
