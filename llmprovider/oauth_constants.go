package llmprovider

// The default OAuth client ids are the vendor CLIs' own public clients:
// Codex's for OpenAI and the Grok CLI's for xAI. Subscription OAuth works only
// with them, so they are borrowed, and a vendor may restrict them at any time
// (0016-MADR D7 and Consequences). OAuthFlowOptions.ClientID overrides them.
// This note becomes the auth package's doc when this file moves there
// (0015-PLAN S7b).
const (
	// DefaultOpenAIIssuer is the issuer used for ChatGPT OAuth sessions.
	DefaultOpenAIIssuer = "https://auth.openai.com"
	// DefaultOpenAIClientID is the public OAuth client used by Codex-compatible flows.
	DefaultOpenAIClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	// DefaultOpenAIChatGPTBaseURL is the ChatGPT subscription API endpoint.
	DefaultOpenAIChatGPTBaseURL = "https://chatgpt.com/backend-api/codex"
	// DefaultOpenAIPlatformBaseURL is the OpenAI API-key endpoint.
	DefaultOpenAIPlatformBaseURL = "https://api.openai.com/v1"
	// DefaultGrokOAuthIssuer is the production xAI OAuth issuer.
	DefaultGrokOAuthIssuer = "https://auth.x.ai"
	// DefaultGrokOAuthClientID is the public OAuth client used by Grok-compatible flows.
	DefaultGrokOAuthClientID = "b1a00492-073a-47ea-816f-4c329264a828"
	// DefaultGrokBaseURL is the xAI API endpoint used by API keys and OAuth sessions.
	DefaultGrokBaseURL = "https://api.x.ai/v1"

	defaultGrokOAuthRefreshURL = "https://auth.x.ai/oauth2/token"
	defaultGrokOAuthDeviceURL  = "https://auth.x.ai/oauth2/device/code"

	// The built-in issuers' published keys, from their discovery documents
	// (probed 2026-09-30; 0016-MADR, "D7 as decided"). They are the fallback
	// when discovery of the built-in issuer fails.
	defaultOpenAIJWKSURL = "https://auth.openai.com/.well-known/jwks.json"
	defaultGrokJWKSURL   = "https://auth.x.ai/.well-known/jwks.json"
)
