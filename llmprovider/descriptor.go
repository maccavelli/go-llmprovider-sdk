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
//
// Every field is a string or a boolean, so the struct is comparable. The
// string facts hold the wire values of the sign-in contract module
// (github.com/maccavelli/go-llmprovider-sdk/authcontract), spelled here as
// plain strings so that this package, under the v1 promise, imports no v0
// module (0074-MADR D1). A fact that does not apply is empty or false.
type AuthMethod struct {
	ID     AuthMethodID
	Label  string
	Detail string

	// Interactive reports that the wizard may offer the method.
	//
	// Deprecated: read Kind and Completion. The value is unchanged and
	// stays until v2 (0074-MADR D7).
	Interactive bool
	// HeadlessOK reports that the method needs no browser: a device code, a
	// value on standard input, or an import.
	//
	// Deprecated: read Completion. The value is unchanged and stays until v2
	// (0074-MADR D7).
	HeadlessOK bool

	// Kind is what the method is: "api_key", "token_paste", "device_code",
	// "browser", "oauth", "vendor_import" or "host_only".
	Kind string
	// Completion is how the sign-in ends: "value", "device",
	// "browser_paste", "browser_loopback", "at_start", "local_file" or
	// "host_command".
	Completion string
	// Paste is what a person may paste to complete the method:
	// "url_or_code", "url_only" or "code". Empty means nothing may be
	// pasted.
	Paste string
	// FirstPaste is Paste for a first sign-in, when it differs from Paste.
	FirstPaste string
	// Yields is what the sign-in produces: "api_key", "oauth_session",
	// "access_only", "device_key", "read_through" or "unknown". For
	// token_stdin on OpenAI it names the stronger case, "access_only"; a
	// pasted "sk-" key is taken as an API key and saves nothing.
	Yields string
	// ClientIdentity is whose OAuth client the method signs in as: "none",
	// "vendor_borrowed", "app_registered" or "unknown".
	ClientIdentity string
	// Slot names the stored credential the method shares with others, such
	// as "openai"; empty when it stores none.
	Slot string
	// Refreshable reports that the credential refreshes itself.
	Refreshable bool
	// Revocable reports that a logout revokes the credential at the issuer.
	Revocable bool
}
