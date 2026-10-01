package llmprovider

// The authentication paths a Descriptor lists. Each provider package declares
// its own Descriptor (0015-MADR D10; 0015-PLAN S8, commit 1).

// AuthMethodID identifies a provider authentication method.
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
)

// AuthMethod describes one authentication path a provider offers.
type AuthMethod struct {
	ID          AuthMethodID
	Label       string
	Detail      string
	Interactive bool
	HeadlessOK  bool
}
