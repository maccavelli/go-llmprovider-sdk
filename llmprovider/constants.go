package llmprovider

// Canonical provider identifiers.
const (
	ProviderGemini = "gemini"
	ProviderOpenAI = "openai"
	ProviderClaude = "claude"
	ProviderGrok   = "grok"
	// ProviderOpencodeZen is the OpenCode Zen gateway (pay-as-you-go).
	ProviderOpencodeZen = "opencode-zen"
	// ProviderOpencodeGo is the OpenCode Go gateway (subscription).
	ProviderOpencodeGo = "opencode-go"
	// ProviderHuggingFace is the Hugging Face Inference Providers router.
	ProviderHuggingFace = "huggingface"
	// ProviderKilo is the Kilo Gateway (the API behind the Kilo Code agent).
	// models.dev registers this gateway as "kilo"; this package follows that
	// registry key. See docs/decisions/0004-MADR-add-gateway-llm-providers.md revision 4.
	ProviderKilo = "kilo"
	// ProviderTogether is Together AI, an OpenAI Chat Completions service
	// (docs/decisions/0017-MADR-together-provider-and-auth-extensions.md D1).
	ProviderTogether = "together"
	// ProviderOllama is a local Ollama instance, reached through its
	// OpenAI-compatible endpoint. It is the only provider needing no credential.
	ProviderOllama = "ollama"
)

// Reasoning effort level values shared across providers.
const (
	effortLow    = "low"
	effortMedium = "medium"
	effortHigh   = "high"
	effortXHigh  = "xhigh"
)

// Item envelope types for Responses API and canonical item models.
const (
	itemTypeMessage            = "message"
	itemTypeFunctionCall       = "function_call"
	itemTypeFunctionCallOutput = "function_call_output"
	itemTypeReasoning          = "reasoning"
)
