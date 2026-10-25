// Package llmprovider is the generation contract every provider implements:
// Provider, Request, Response and Item, Capabilities, the errors and their
// kinds, the options and Settings, the Registry, retry middleware, the
// thinking settings a model refused (ReasoningRefusal, WithReasoningRefusals,
// and ProbeReasoning, which finds them with one generation), and the
// credential sources a provider reads (Token, TokenSource, StaticToken,
// CommandToken). The providers are in llmprovider/providers/..., the OAuth
// sessions and stores in llmprovider/auth, and model listing in
// llmprovider/catalog. All providers use raw net/http.
//
// A provider is built with options and asked for text; a failure is matched
// by its kind:
//
//	p, err := providers.New(llmprovider.ProviderClaude,
//		llmprovider.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
//		llmprovider.WithModel("claude-sonnet-5"))
//	if err != nil {
//		return err
//	}
//	p = llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3})
//	text, err := llmprovider.GenerateText(ctx, p, &llmprovider.Request{
//		Input: []llmprovider.Item{llmprovider.MessageItem{Text: "Say hello in one word."}},
//	})
//	if errors.Is(err, llmprovider.ErrRateLimited) {
//		return fmt.Errorf("try later: %w", err)
//	}
package llmprovider

import (
	"errors"
	"maps"
	"time"
)

// Tool defines an LLM function/tool schema, for Request.Tools. It is safe for
// concurrent use only while no goroutine changes it.
type Tool struct {
	Name        string
	Description string
	Schema      any
	// Strict, when set, asks the service to hold the call's arguments to
	// Schema (true) or not (false). Nil leaves the service's default: OpenAI's
	// Responses API applies strict mode unless a tool opts out. It is sent
	// where the wire has the field, and each provider's package doc says
	// where it is not (0032-MADR D1, D2).
	Strict *bool
	// Deferred offers the tool but keeps it out of the model's context
	// until a ToolLoadItem in Input loads it (0040-MADR A5). Each provider
	// sends it in its service's own form, through a fixed wrapper, or by
	// appending the tool once it is loaded, as its package doc says; a
	// response says when it was not the service's own form with
	// WarningDeferralEmulated (0040-MADR D1, D7). At least one tool in a
	// request is not deferred.
	Deferred bool
	// Search marks the caller's own search for deferred tools: the tool
	// whose results load them. At most one tool in a request has it, and it
	// is never deferred (0040-MADR D1).
	Search bool
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
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr.RetryAfter
	}
	return 0
}

// ProviderEnvVars returns a copy of the map from each provider id to its
// standard environment variable. The package reads none of them itself
// (0015-MADR D9).
func ProviderEnvVars() map[ProviderID]string {
	out := make(map[ProviderID]string, len(providerEnvVars))
	maps.Copy(out, providerEnvVars)
	return out
}

// providerEnvVars maps canonical provider names to their standard environment variable.
var providerEnvVars = map[ProviderID]string{
	ProviderGemini: "GEMINI_API_KEY",
	ProviderClaude: "ANTHROPIC_API_KEY",
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
