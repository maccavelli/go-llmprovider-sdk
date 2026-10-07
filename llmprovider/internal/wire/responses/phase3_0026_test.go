package responses

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestResponses_FailedStatusClassified (0026-MADR F28): a non-streamed reply
// with "status":"failed" is classified from its error, as response.failed is
// in a stream, and keeps the service's message.
func TestResponses_FailedStatusClassified(t *testing.T) {
	for _, c := range []struct {
		name, provider, body string
		kind                 error
		retryable            bool
	}{
		{"server error", "grok", `{"status":"failed","error":{"code":"server_error","message":"boom"},"output":[]}`,
			llmprovider.ErrProviderUnavailable, true},
		{"rate limit", "grok", `{"status":"failed","error":{"code":"rate_limit_exceeded","message":"slow down"}}`,
			llmprovider.ErrRateLimited, true},
		{"openai quota", "openai", `{"status":"failed","error":{"code":"insufficient_quota","message":"quota"}}`,
			llmprovider.ErrQuotaExhausted, false},
		{"no error object", "grok", `{"status":"failed"}`, llmprovider.ErrProviderUnavailable, true},
	} {
		_, err := DecodeFor(c.provider)(strings.NewReader(c.body))
		apiErr, ok := errors.AsType[*llmprovider.APIError](err)
		if !ok || !errors.Is(err, c.kind) || apiErr.Retryable() != c.retryable || errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("%s: %v; want %v, retryable=%t, not ErrIncomplete", c.name, err, c.kind, c.retryable)
			continue
		}
		if apiErr.Provider != c.provider {
			t.Errorf("%s: Provider %q; want %q", c.name, apiErr.Provider, c.provider)
		}
	}
	_, err := Decode(strings.NewReader(`{"status":"failed","error":{"code":"server_error","message":"boom"}}`))
	if apiErr, ok := errors.AsType[*llmprovider.APIError](err); !ok || !strings.Contains(apiErr.Message, "boom") {
		t.Errorf("Decode: %v; want the service's message kept", err)
	}
}

// TestResponses_EmptyArguments (0026-MADR F31): empty function arguments are
// "{}", on Decode and on ReadStream, as on every other wire (0021-MADR W1).
func TestResponses_EmptyArguments(t *testing.T) {
	call := `{"type":"function_call","call_id":"c1","name":"now","arguments":""}`
	res, err := Decode(strings.NewReader(`{"status":"completed","output":[` + call + `]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output[0].(llmprovider.FunctionCallItem).Arguments; got != "{}" {
		t.Errorf("Decode: arguments %q; want {}", got)
	}
	stream := "data: {\"type\":\"response.output_item.done\",\"item\":" + call + "}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r1\"}}\n\n"
	res, err = ReadStream("grok", strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Output[0].(llmprovider.FunctionCallItem).Arguments; got != "{}" {
		t.Errorf("ReadStream: arguments %q; want {}", got)
	}
}
