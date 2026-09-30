package llmprovider

import (
	"regexp"
	"slices"
)

// Context overflow (0015-MADR D7, amendment of 2026-09-30). A service's own
// error type decides where there is one; otherwise its message is matched
// against the forms each vendor was seen to use. "pi" cites
// packages/ai/src/utils/overflow.ts at pi commit 0e283203c, by line; the
// gateways (OpenCode, Kilo, Hugging Face) pass their upstream's message on,
// so an upstream's form applies through them.

// contextOverflowTypes are service error types that mean overflow.
var contextOverflowTypes = []string{
	"context_length_exceeded",       // OpenAI and the ChatGPT backend (MADR 0012 §1.1)
	"request_too_large",             // Anthropic, HTTP 413 (pi :12, :39)
	"model_context_window_exceeded", // z.ai (pi :56)
}

// contextOverflowMessages are the message forms, each with where it was seen.
var contextOverflowMessages = []struct {
	pattern *regexp.Regexp
	seen    string
}{
	{regexp.MustCompile(`(?i)prompt (?:is )?too long`), "Anthropic, z.ai and Ollama (pi :11, :30, :35, :38)"},
	{regexp.MustCompile(`(?i)input is too long for requested model`), "Amazon Bedrock (pi :40)"},
	{regexp.MustCompile(`(?i)exceeds the context window`), "OpenAI (pi :13, :41)"},
	{regexp.MustCompile(`(?i)exceeds (?:the )?(?:model'?s )?maximum context length(?: of [\d,]+ tokens?|\s*\([\d,]+\))`), "OpenAI-compatible proxies (pi :14-15, :42)"},
	{regexp.MustCompile(`(?i)input token count.*exceeds the maximum`), "Google Gemini (pi :16, :43)"},
	{regexp.MustCompile(`(?i)maximum prompt length is \d+`), "xAI (pi :17, :44)"},
	{regexp.MustCompile(`(?i)reduce the length of the messages`), "Groq (pi :18, :45)"},
	{regexp.MustCompile(`(?i)maximum context length is \d+ tokens`), "OpenRouter (pi :19, :46)"},
	{regexp.MustCompile(`(?i)exceeds (?:the )?maximum allowed input length of [\d,]+ tokens?`), "OpenRouter's Poolside route (pi :20, :47)"},
	{regexp.MustCompile(`(?i)input \(\d+ tokens\) is longer than the model'?s context length \(\d+ tokens\)`), "Together AI (pi :21, :48)"},
	{regexp.MustCompile(`(?i)exceeds the limit of \d+`), "GitHub Copilot (pi :24, :49)"},
	{regexp.MustCompile(`(?i)exceeds the available context size`), "llama.cpp (pi :22, :50)"},
	{regexp.MustCompile(`(?i)greater than the context length`), "LM Studio (pi :23, :51)"},
	{regexp.MustCompile(`(?i)context window exceeds limit`), "MiniMax (pi :25, :52)"},
	{regexp.MustCompile(`(?i)exceeded model token limit`), "Kimi (pi :26, :53)"},
	{regexp.MustCompile(`(?i)too large for model with \d+ maximum context length`), "Mistral (pi :29, :54)"},
	{regexp.MustCompile(`(?i)prompt has [\d,]+ tokens?, but the configured context size is [\d,]+ tokens?`), "DS4 (pi :27, :55)"},
	{regexp.MustCompile(`(?i)range of input length should be`), "DashScope and Qwen (pi :34, :58)"},
	{regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`), "any service, as a message (pi :59)"},
	{regexp.MustCompile(`(?i)too many tokens`), "any service (pi :60)"},
	{regexp.MustCompile(`(?i)token limit exceeded`), "any service (pi :61)"},
}

// notContextOverflow are messages that match a form above and are rate
// limits instead (pi :75-79).
var notContextOverflow = regexp.MustCompile(`(?i)rate limit|too many requests`)

// contextOverflow reports whether an invalid-request error body says the
// input is longer than the model's context window.
func contextOverflow(env apiErrorEnvelope) bool {
	if slices.ContainsFunc(contextOverflowTypes, env.hasType) {
		return true
	}
	msg := env.message()
	if msg == "" || notContextOverflow.MatchString(msg) {
		return false
	}
	for _, form := range contextOverflowMessages {
		if form.pattern.MatchString(msg) {
			return true
		}
	}
	return false
}
