package llmprovider

import (
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
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

// anchorMaxContextLength is the anchor three forms share.
const anchorMaxContextLength = "maximum context length"

// contextOverflowMessages are the message forms, each with where it was seen
// and a lower-case literal it cannot match without, checked first
// (0028-MADR D-A10).
var contextOverflowMessages = []struct {
	pattern *regexp.Regexp
	seen    string
	anchor  string
}{
	{regexp.MustCompile(`(?i)prompt (?:is )?too long`), "Anthropic, z.ai and Ollama (pi :11, :30, :35, :38)", "too long"},
	{regexp.MustCompile(`(?i)input is too long for requested model`), "Amazon Bedrock (pi :40)", "too long"},
	{regexp.MustCompile(`(?i)exceeds the context window`), "OpenAI (pi :13, :41)", "context window"},
	{regexp.MustCompile(`(?i)exceeds (?:the )?(?:model'?s )?maximum context length(?: of [\d,]+ tokens?|\s*\([\d,]+\))`), "OpenAI-compatible proxies (pi :14-15, :42)", anchorMaxContextLength},
	{regexp.MustCompile(`(?i)input token count.*exceeds the maximum`), "Google Gemini (pi :16, :43)", "exceeds the maximum"},
	{regexp.MustCompile(`(?i)maximum prompt length is \d+`), "xAI (pi :17, :44)", "maximum prompt length"},
	{regexp.MustCompile(`(?i)reduce the length of the messages`), "Groq (pi :18, :45)", "reduce the length"},
	{regexp.MustCompile(`(?i)maximum context length is \d+ tokens`), "OpenRouter (pi :19, :46)", anchorMaxContextLength},
	{regexp.MustCompile(`(?i)exceeds (?:the )?maximum allowed input length of [\d,]+ tokens?`), "OpenRouter's Poolside route (pi :20, :47)", "maximum allowed input length"},
	{regexp.MustCompile(`(?i)input \(\d+ tokens\) is longer than the model'?s context length \(\d+ tokens\)`), "Together AI (pi :21, :48)", "context length"},
	{regexp.MustCompile(`(?i)exceeds the limit of \d+`), "GitHub Copilot (pi :24, :49)", "exceeds the limit"},
	{regexp.MustCompile(`(?i)exceeds the available context size`), "llama.cpp (pi :22, :50)", "available context size"},
	{regexp.MustCompile(`(?i)greater than the context length`), "LM Studio (pi :23, :51)", "context length"},
	{regexp.MustCompile(`(?i)context window exceeds limit`), "MiniMax (pi :25, :52)", "context window"},
	{regexp.MustCompile(`(?i)exceeded model token limit`), "Kimi (pi :26, :53)", "token limit"},
	{regexp.MustCompile(`(?i)too large for model with \d+ maximum context length`), "Mistral (pi :29, :54)", anchorMaxContextLength},
	{regexp.MustCompile(`(?i)prompt has [\d,]+ tokens?, but the configured context size is [\d,]+ tokens?`), "DS4 (pi :27, :55)", "configured context size"},
	{regexp.MustCompile(`(?i)range of input length should be`), "DashScope and Qwen (pi :34, :58)", "range of input length"},
	{regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`), "any service, as a message (pi :59)", "exceeded"},
	{regexp.MustCompile(`(?i)too many tokens`), "any service (pi :60)", "too many tokens"},
	{regexp.MustCompile(`(?i)token limit exceeded`), "any service (pi :61)", "token limit"},
}

// The OpenAI-compatible overflow message states the context's limit and how
// the request split between input and completion (0028-MADR D-H3).
var (
	contextLimitPattern = regexp.MustCompile(`(?i)maximum context length is (\d+)`)
	contextSplitPattern = regexp.MustCompile(`(?i)(\d+) in the messages, (\d+) in the completion`)
)

// reasonCompletionFillsContext is the Reason of an invalid request whose
// requested completion alone fills the context.
const reasonCompletionFillsContext = "max_tokens alone fills the context window: lower it"

// completionFillsContext reports whether an overflow message says the
// requested completion is at least the context's limit, so that no
// shortening of the input can succeed (0028-MADR D-H3).
func completionFillsContext(msg string) bool {
	// The split is in the message's first sentences; matching only those,
	// and only when they hold its literal, keeps a long message cheap
	// (0028-MADR D-A10). The literal holds no letter (?i) folds from
	// outside ASCII.
	if len(msg) > contextOverflowReadLimit {
		msg = msg[:contextOverflowReadLimit]
	}
	if !strings.Contains(strings.ToLower(msg), "in the completion") {
		return false
	}
	limit := contextLimitPattern.FindStringSubmatch(msg)
	split := contextSplitPattern.FindStringSubmatch(msg)
	if limit == nil || split == nil {
		return false
	}
	contextTokens, errLimit := strconv.Atoi(limit[1])
	completion, errSplit := strconv.Atoi(split[2])
	return errLimit == nil && errSplit == nil && completion >= contextTokens
}

// noteCompletionFillsContext names max_tokens on an invalid request whose
// completion alone fills the context.
func noteCompletionFillsContext(e *APIError, msg string) {
	if errors.Is(e.Kind, ErrInvalidRequest) && completionFillsContext(msg) {
		e.Reason = reasonCompletionFillsContext
	}
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
	if len(msg) > contextOverflowReadLimit {
		cut := contextOverflowReadLimit
		for cut > 0 && !utf8.RuneStart(msg[cut]) {
			cut--
		}
		msg = msg[:cut]
	}
	if msg == "" || notContextOverflow.MatchString(msg) {
		return false
	}
	// A form runs only when the message holds its literal. (?i) folds some
	// letters outside ASCII into ASCII ones, such as the long s, so a
	// message with any byte outside ASCII runs every form.
	lower, ascii := strings.ToLower(msg), isASCIIString(msg)
	for _, form := range contextOverflowMessages {
		if ascii && !strings.Contains(lower, form.anchor) {
			continue
		}
		if form.pattern.MatchString(msg) {
			return true
		}
	}
	return false
}

// contextOverflowReadLimit bounds how much of a message is matched: every
// form names the overflow in its first sentence (0028-MADR D-A10).
const contextOverflowReadLimit = 2 << 10

// isASCIIString reports whether s holds only ASCII bytes.
func isASCIIString(s string) bool {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
