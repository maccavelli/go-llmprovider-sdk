package llmprovider

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// overflowSamples holds, for each message form, a message the vendor sent,
// keyed by the form's seen label. The samples are pi's examples
// (overflow.ts:11-35), with numbers filled in.
var overflowSamples = map[string]string{
	"Anthropic, z.ai and Ollama (pi :11, :30, :35, :38)": "prompt is too long: 213462 tokens > 200000 maximum",
	"Amazon Bedrock (pi :40)":                            "Input is too long for requested model.",
	"OpenAI (pi :13, :41)":                               "Your input exceeds the context window of this model.",
	"OpenAI-compatible proxies (pi :14-15, :42)":         "Requested token count exceeds the model's maximum context length of 131072 tokens",
	"Google Gemini (pi :16, :43)":                        "The input token count (1196265) exceeds the maximum number of tokens allowed (1048575)",
	"xAI (pi :17, :44)":                                  "This model's maximum prompt length is 131072 but the request contains 537812 tokens",
	"Groq (pi :18, :45)":                                 "Please reduce the length of the messages or completion.",
	"OpenRouter (pi :19, :46)":                           "This endpoint's maximum context length is 200000 tokens. However, you requested about 250000 tokens.",
	"OpenRouter's Poolside route (pi :20, :47)":          "Input length 300000 exceeds the maximum allowed input length of 262144 tokens.",
	"Together AI (pi :21, :48)":                          "The input (140000 tokens) is longer than the model's context length (131072 tokens).",
	"GitHub Copilot (pi :24, :49)":                       "prompt token count of 130000 exceeds the limit of 128000",
	"llama.cpp (pi :22, :50)":                            "the request exceeds the available context size, try increasing it",
	"LM Studio (pi :23, :51)":                            "tokens to keep from the initial prompt is greater than the context length",
	"MiniMax (pi :25, :52)":                              "invalid params, context window exceeds limit",
	"Kimi (pi :26, :53)":                                 "Your request exceeded model token limit: 262144 (requested: 300000)",
	"Mistral (pi :29, :54)":                              "Prompt contains 140000 tokens and 0 draft tokens, too large for model with 131072 maximum context length",
	"DS4 (pi :27, :55)":                                  "Prompt has 70000 tokens, but the configured context size is 65536 tokens",
	"DashScope and Qwen (pi :34, :58)":                   "Range of input length should be [1, 129024]",
	"any service, as a message (pi :59)":                 "context length exceeded",
	"any service (pi :60)":                               "Request has too many tokens",
	"any service (pi :61)":                               "token limit exceeded for this model",
}

// TestContextOverflow_EveryFormHasASample keeps the table and its samples in
// step: a form without a sample would have no red test.
func TestContextOverflow_EveryFormHasASample(t *testing.T) {
	for _, form := range contextOverflowMessages {
		if _, ok := overflowSamples[form.seen]; !ok {
			t.Errorf("form %q (%s) has no sample", form.pattern, form.seen)
		}
	}
	if len(overflowSamples) != len(contextOverflowMessages) {
		t.Errorf("%d samples for %d forms", len(overflowSamples), len(contextOverflowMessages))
	}
}

// TestContextOverflow_Messages sends each sample as an HTTP 400 and expects
// ErrContextOverflow, which is still an ErrInvalidRequest and not retryable.
func TestContextOverflow_Messages(t *testing.T) {
	for seen, msg := range overflowSamples {
		t.Run(seen, func(t *testing.T) {
			err := classifyBody(ProviderTogether, http.StatusBadRequest, `{"error":{"message":`+quote(msg)+`}}`)
			assertOverflow(t, err)
		})
	}
}

func TestContextOverflow_Types(t *testing.T) {
	for _, c := range []struct{ name, status, body string }{
		{"OpenAI's code", "400", `{"error":{"type":"invalid_request_error","code":"context_length_exceeded","message":"This model's context is full."}}`},
		{"Anthropic's request_too_large", "413", `{"type":"error","error":{"type":"request_too_large","message":"Request exceeds the maximum size"}}`},
		{"z.ai's code", "400", `{"error":{"code":"model_context_window_exceeded","message":"x"}}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			status := http.StatusBadRequest
			if c.status == "413" {
				status = http.StatusRequestEntityTooLarge
			}
			assertOverflow(t, classifyBody(ProviderOpenAI, status, c.body))
		})
	}
}

func TestContextOverflow_StreamFailure(t *testing.T) {
	err := streamFailure(ProviderOpenAI, "context_length_exceeded", "invalid_request_error", "Your input exceeds the context window of this model.")
	assertOverflow(t, err)
	if err := streamFailure(ProviderOpenAI, "invalid_prompt", "", "bad"); errors.Is(err, ErrContextOverflow) || !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid_prompt: %v; want ErrInvalidRequest only", err)
	}
}

func TestContextOverflow_NotEveryInvalidRequest(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		msg    string
		want   error
	}{
		{"an unrelated 400", http.StatusBadRequest, "Unknown parameter: temperature_x", ErrInvalidRequest},
		{"a throttle that names tokens", http.StatusBadRequest, "Too many tokens, rate limit reached; please wait", ErrInvalidRequest},
		{"a 400 that says too many requests", http.StatusBadRequest, "too many requests, too many tokens", ErrInvalidRequest},
		{"a 429 that names tokens", http.StatusTooManyRequests, "too many tokens per minute", ErrRateLimited},
		{"a 500 that names the prompt", http.StatusInternalServerError, "prompt is too long to log", ErrProviderUnavailable},
		{"an empty message", http.StatusBadRequest, "", ErrInvalidRequest},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := classifyBody(ProviderGrok, c.status, `{"error":{"message":`+quote(c.msg)+`}}`)
			if errors.Is(err, ErrContextOverflow) || !errors.Is(err, c.want) {
				t.Fatalf("%v; want %v and not ErrContextOverflow", err, c.want)
			}
		})
	}
}

func assertOverflow(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrContextOverflow) || !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("%v; want ErrContextOverflow, beneath ErrInvalidRequest", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Retryable() || !errors.Is(apiErr.Kind, ErrContextOverflow) {
		t.Fatalf("%#v; want a non-retryable APIError of kind ErrContextOverflow", err)
	}
	if !strings.HasPrefix(err.Error(), "llmprovider: context window exceeded: ") {
		t.Fatalf("message %q", err.Error())
	}
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
