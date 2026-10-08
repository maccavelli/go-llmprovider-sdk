package llmprovider

import (
	"errors"
	"strings"
	"testing"
)

// regionForbidden is OpenAI's documented refusal for a region, a 403 that
// says nothing about the key.
const regionForbidden = `{"error":{"type":"request_forbidden","code":"unsupported_country_region_territory","message":"Country, region, or territory not supported"}}`

// TestClassify_403IsNotPermittedUnlessCredentialCode (0028-MADR D-H2): on the
// direct providers a 403 is a refusal of something the key may not do, not of
// the key, unless its body carries a credential code a provider was measured
// sending with 403 (0028-PLAN Phase 1, T1: none). The legacy status sentinel
// still matches ErrAuthFailure.
func TestClassify_403IsNotPermittedUnlessCredentialCode(t *testing.T) {
	for _, service := range []string{"openai", "claude", "gemini", "grok", "together", "huggingface", "ollama"} {
		err := classifyFixture(service, 403, regionForbidden, nil)
		apiErr, ok := errors.AsType[*APIError](err)
		if !ok {
			t.Fatalf("%s: %v; want an *APIError", service, err)
		}
		if apiErr.Kind != ErrNotPermitted || apiErr.Retryable() || !errors.Is(err, ErrAuthFailure) { //nolint:errorlint // the kind itself
			t.Errorf("%s 403: kind %v, retryable %t, matches ErrAuthFailure %t; want ErrNotPermitted, not retryable, true",
				service, apiErr.Kind, apiErr.Retryable(), errors.Is(err, ErrAuthFailure))
		}
		for _, code := range credential403Codes[service] {
			body := `{"error":{"code":"` + code + `","message":"refused"}}`
			if e, _ := errors.AsType[*APIError](classifyFixture(service, 403, body, nil)); e == nil || e.Kind != ErrAuthFailure { //nolint:errorlint // the kind itself
				t.Errorf("%s 403 %s: %v; want ErrAuthFailure", service, code, e)
			}
		}
	}
}

// grokRefusedKey is xAI's reply to a key that is not a key, as measured
// (0028-PLAN Phase 1, T1).
const grokRefusedKey = `{"code":"invalid-argument","error":"Incorrect API key provided. You can obtain an API key from https://console.x.ai."}`

// TestClassify_GrokRefusedKey (0028-MADR H11): xAI refuses a key with HTTP
// 400 invalid-argument, which is a refused credential.
func TestClassify_GrokRefusedKey(t *testing.T) {
	err := classifyFixture("grok", 400, grokRefusedKey, nil)
	if apiErr, ok := errors.AsType[*APIError](err); !ok || apiErr.Kind != ErrAuthFailure || apiErr.Retryable() { //nolint:errorlint // the kind itself
		t.Fatalf("Grok's refused key: %v; want ErrAuthFailure, not retryable", err)
	}
}

// TestClassify_GrokInvalidArgumentStaysInvalid: an invalid-argument 400 that
// names no API key is the request's fault, as today.
func TestClassify_GrokInvalidArgumentStaysInvalid(t *testing.T) {
	body := `{"code":"invalid-argument","error":"Invalid request content: max_output_tokens must be positive"}`
	err := classifyFixture("grok", 400, body, nil)
	if apiErr, ok := errors.AsType[*APIError](err); !ok || apiErr.Kind != ErrInvalidRequest { //nolint:errorlint // the kind itself
		t.Fatalf("Grok's invalid argument: %v; want ErrInvalidRequest", err)
	}
}

// overflowSplit is the OpenAI-compatible overflow message with its split.
func overflowSplit(limit, messages, completion string) string {
	return `{"error":{"message":"This model's maximum context length is ` + limit + ` tokens. However, you requested 0 tokens (` +
		messages + ` in the messages, ` + completion + ` in the completion). Please reduce the length of the messages or completion.",` +
		`"type":"invalid_request_error","code":null}}`
}

// TestContextOverflow_CompletionAlone (0028-MADR D-H3): when the requested
// completion alone fills the context, shortening the input cannot help, so
// the failure is an invalid request naming max_tokens; otherwise it stays an
// overflow.
func TestContextOverflow_CompletionAlone(t *testing.T) {
	for _, c := range []struct {
		name, body string
		overflow   bool
	}{
		{"10 + 8192 on 8192", overflowSplit("8192", "10", "8192"), false},
		{"0 + 9000 on 8192", overflowSplit("8192", "0", "9000"), false},
		{"8000 + 1000 on 8192", overflowSplit("8192", "8000", "1000"), true},
		{"no split", `{"error":{"message":"This model's maximum context length is 8192 tokens.","type":"invalid_request_error"}}`, true},
	} {
		err := classifyFixture("together", 400, c.body, nil)
		if got := errors.Is(err, ErrContextOverflow); got != c.overflow {
			t.Errorf("%s: overflow = %t; want %t (%v)", c.name, got, c.overflow, err)
		}
		if !c.overflow && (!errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), "max_tokens")) {
			t.Errorf("%s: %v; want ErrInvalidRequest naming max_tokens", c.name, err)
		}
	}
}
