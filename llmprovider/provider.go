// Package llmprovider provides an SDK-free LLM provider abstraction: provider
// adapters, OAuth sessions, token storage and model discovery. All providers
// use raw net/http for maximum control over connection pooling and timeouts.
package llmprovider

import (
	"errors"
	"time"
)

// Tool defines an LLM function/tool schema, for Request.Tools.
type Tool struct {
	Name        string
	Description string
	Schema      any
}

// Typed errors for programmatic classification (go.dev/doc/effective-go).
var (
	ErrRateLimited         = errors.New("llmprovider: rate limited")
	ErrProviderUnavailable = errors.New("llmprovider: provider unavailable")
	ErrAuthFailure         = errors.New("llmprovider: authentication failed")
	// ErrInvalidRequest marks a non-retryable client error (4xx other than 429
	// and 401/403). Retrying it can never succeed.
	ErrInvalidRequest = errors.New("llmprovider: invalid request")
)

// retryBackoffCap is RetryPolicy's default MaxDelay. A server asking for
// longer gets its error back instead, so the caller can reschedule (MADR 0012
// §1.2).
const retryBackoffCap = 30 * time.Second

// serverRetryAfter is the delay a failed attempt asked for, or 0.
func serverRetryAfter(err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

// ProviderEnvVars returns a copy of the map from each provider id to its
// standard environment variable. The package reads none of them itself
// (0015-MADR D9).
func ProviderEnvVars() map[ProviderID]string {
	out := make(map[ProviderID]string, len(providerEnvVars))
	for id, name := range providerEnvVars {
		out[id] = name
	}
	return out
}

// providerEnvVars maps canonical provider names to their standard environment variable.
var providerEnvVars = map[ProviderID]string{
	ProviderGemini: "GEMINI_API_KEY",
	ProviderClaude: "CLAUDE_API_KEY",
	ProviderOpenAI: "OPENAI_API_KEY",
	ProviderGrok:   "XAI_API_KEY",
	// One credential serves both OpenCode gateways; models.dev declares
	// OPENCODE_API_KEY for each, and neither docs page names any other variable.
	ProviderOpencodeZen: "OPENCODE_API_KEY",
	ProviderOpencodeGo:  "OPENCODE_API_KEY",
	ProviderHuggingFace: "HF_TOKEN",
	ProviderKilo:        "KILO_API_KEY",
	ProviderTogether:    "TOGETHER_API_KEY",
}
