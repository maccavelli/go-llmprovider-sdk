package openai

import (
	"context"
	"reflect"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestBody_StatelessAsksForEncryptedReasoning (0021-MADR D4, W8): a request
// sent with store false asks for its reasoning encrypted, so a tool loop can
// replay it; a stored one, or one that sets no store, does not.
func TestBody_StatelessAsksForEncryptedReasoning(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		want any
	}{
		{"store false", []llmprovider.Option{WithStore(false)}, []any{"reasoning.encrypted_content"}},
		{"store true", []llmprovider.Option{WithStore(true)}, nil},
		{"no store", nil, nil},
	} {
		var body map[string]any
		srv := captureServer(t, &body, okResponse)
		p := build(t, apiKey(srv.URL, append([]llmprovider.Option{llmprovider.WithModel("gpt-6-luna")}, tc.opts...)...)...)
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := body["include"]; !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: include = %v, want %v", tc.name, got, tc.want)
		}
	}
}
