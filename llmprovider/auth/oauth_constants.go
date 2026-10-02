package auth

// The issuers and the default OAuth client ids; the package doc says whose
// the client ids are.
const (
	// DefaultOpenAIIssuer is the issuer used for ChatGPT OAuth sessions.
	DefaultOpenAIIssuer = "https://auth.openai.com"
	// DefaultOpenAIClientID is the public OAuth client used by Codex-compatible flows.
	DefaultOpenAIClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	// DefaultGrokOAuthIssuer is the production xAI OAuth issuer.
	DefaultGrokOAuthIssuer = "https://auth.x.ai"
	// DefaultGrokOAuthClientID is the public OAuth client used by Grok-compatible flows.
	DefaultGrokOAuthClientID = "b1a00492-073a-47ea-816f-4c329264a828"

	defaultGrokOAuthRefreshURL = "https://auth.x.ai/oauth2/token"
	defaultGrokOAuthDeviceURL  = "https://auth.x.ai/oauth2/device/code"

	// The built-in issuers' published keys, from their discovery documents
	// (probed 2026-09-30; 0016-MADR, "D7 as decided"). They are the fallback
	// when discovery of the built-in issuer fails.
	defaultOpenAIJWKSURL = "https://auth.openai.com/.well-known/jwks.json"
	defaultGrokJWKSURL   = "https://auth.x.ai/.well-known/jwks.json"
)
