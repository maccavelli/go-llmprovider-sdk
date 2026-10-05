package opencode

import (
	"context"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestResponsesBody_AsksForEncryptedReasoning (0021-MADR D4, W8): every
// responses-route request is stateless, so it asks for its reasoning
// encrypted, and for a summary whenever it asks for reasoning.
func TestResponsesBody_AsksForEncryptedReasoning(t *testing.T) {
	for name, req := range map[string]*llmprovider.Request{"text": text("hi"), "thinking": thinking(text("hi"))} {
		var body map[string]any
		srv := captureServer(t, &body, fxResponses)
		if _, err := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "gpt-5.5")...).Generate(context.Background(), req); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := body["include"]; !reflect.DeepEqual(got, []any{"reasoning.encrypted_content"}) {
			t.Errorf("%s: include = %v, want encrypted reasoning", name, got)
		}
		if reasoning, ok := body["reasoning"].(map[string]any); ok && reasoning["summary"] != "auto" {
			t.Errorf("%s: reasoning = %v, want summary auto", name, reasoning)
		}
		if name == "thinking" && body["reasoning"] == nil {
			t.Errorf("thinking: no reasoning sent")
		}
	}
}
