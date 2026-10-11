package claude

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestGenerate_ForcingPolicy_0034 (0034-MADR D9, D10): ToolChoiceRequired on a
// refusing model fails before the network by default; WithForcedChoiceFallback
// degrades it; WithModelForcedChoice overrides the table either way.
func TestGenerate_ForcingPolicy_0034(t *testing.T) {
	for _, c := range []struct {
		name, model string
		opts        []llmprovider.Option
		wantErr     bool
		wantChoice  string // the tool_choice type sent, "" for no request
		accepts     bool   // the fake service accepts a forced choice, as the model does
	}{
		{"strict by default", "claude-sonnet-5-5", nil, true, "", false},
		{"an alias, strict", "anthropic/claude-opus-5-5", nil, true, "", false},
		{"opted in to degrading", "claude-sonnet-5-5",
			[]llmprovider.Option{llmprovider.WithForcedChoiceFallback(llmprovider.ForcedChoiceDegrade)}, false, "auto", false},
		{"the table overridden", "claude-sonnet-5-5",
			[]llmprovider.Option{llmprovider.WithModelForcedChoice("claude-sonnet-5-5", llmprovider.Supported)}, false, "any", true},
		{"an unknown model marked", "claude-future-marked",
			[]llmprovider.Option{llmprovider.WithModelForcedChoice("claude-future-marked", llmprovider.Unsupported)}, true, "", false},
		{"an accepting model", "claude-sonnet-5", nil, false, "any", true},
	} {
		status, refusal := http.StatusBadRequest, refusal0034
		if c.accepts {
			status, refusal = http.StatusOK, callReply0034
		}
		srv, rs := newRefusingServer(t, callReply0034, false, status, refusal)
		req := weatherRequest()
		req.ToolChoice = llmprovider.ToolChoiceRequired
		_, err := build(t, apiKey(srv.URL, c.model, c.opts...)...).Generate(context.Background(), req)
		if got := errors.Is(err, llmprovider.ErrUnsupported); got != c.wantErr {
			t.Errorf("%s: err = %v, want ErrUnsupported: %v", c.name, err, c.wantErr)
		}
		choices, _ := rs.seen()
		switch {
		case c.wantChoice == "" && len(choices) != 0:
			t.Errorf("%s: %d requests sent, want none", c.name, len(choices))
		case c.wantChoice != "" && (len(choices) != 1 || choices[0] != c.wantChoice):
			t.Errorf("%s: tool_choice %v, want [%s]", c.name, choices, c.wantChoice)
		}
	}
}
