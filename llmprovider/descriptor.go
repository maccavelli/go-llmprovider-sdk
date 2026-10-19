package llmprovider

// The authentication paths a Descriptor lists. Each provider package declares
// its own Descriptor (0015-MADR D10; 0015-PLAN S8, commit 1).

// AuthMethodID identifies a provider authentication method. It is a string,
// safe for concurrent use.
type AuthMethodID string

const (
	// AuthAPIKey selects provider API-key authentication.
	AuthAPIKey AuthMethodID = "api_key"
	// AuthBrowserOAuth selects interactive browser OAuth.
	AuthBrowserOAuth AuthMethodID = "browser_oauth"
	// AuthDeviceCode selects a headless device-code flow.
	AuthDeviceCode AuthMethodID = "device_code"
	// AuthTokenStdin selects a credential pasted through standard input.
	AuthTokenStdin AuthMethodID = "token_stdin"
	// AuthImportVendorCLI imports a session from the provider's CLI.
	AuthImportVendorCLI AuthMethodID = "import_vendor_cli"
	// AuthChatGPTPlan selects OpenAI's Sign in with ChatGPT for agents: the
	// user's ChatGPT plan, under the consumer's own app name
	// (auth.LoginChatGPTPlan; 0043-MADR D7).
	AuthChatGPTPlan AuthMethodID = "chatgpt_plan"
)

// AuthMethod describes one authentication path a provider offers. It is safe
// for concurrent use only while no goroutine changes it.
type AuthMethod struct {
	ID          AuthMethodID
	Label       string
	Detail      string
	Interactive bool
	HeadlessOK  bool
}
