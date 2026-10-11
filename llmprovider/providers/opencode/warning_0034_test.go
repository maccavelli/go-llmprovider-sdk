package opencode

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestMessagesRoute_WarnsWhenDegraded_0034 (0034-MADR D7): on Zen's messages
// route, a degraded forced choice warns on the Response and on the stream's
// done event; an accepting model's does not. The degraded choice is
// ToolChoiceRequired, opted in to (D10).
func TestMessagesRoute_WarnsWhenDegraded_0034(t *testing.T) {
	for _, c := range []struct {
		model string
		warns int
	}{{"claude-opus-5-5", 1}, {"claude-sonnet-5", 0}} {
		var body map[string]any
		srv := captureServer(t, &body, callReply0034)
		req := weatherRequest0034()
		req.ToolChoice = llmprovider.ToolChoiceRequired
		p := build(t, llmprovider.ProviderOpencodeZen,
			apiKey(srv.URL, c.model, llmprovider.WithForcedChoiceFallback(llmprovider.ForcedChoiceDegrade))...)
		resp, err := p.Generate(context.Background(), req)
		if err != nil {
			t.Fatalf("%s: %v", c.model, err)
		}
		if len(resp.Warnings) != c.warns {
			t.Errorf("%s: warnings %+v, want %d", c.model, resp.Warnings, c.warns)
		}
	}
	srv, _ := refusingServer(t, llmtestMessagesStream, refusal0034, false)
	req := weatherRequest0034()
	req.ToolChoice = llmprovider.ToolChoiceRequired
	p := build(t, llmprovider.ProviderOpencodeZen,
		apiKey(srv.URL, "claude-opus-5-5", llmprovider.WithForcedChoiceFallback(llmprovider.ForcedChoiceDegrade))...)
	var warnings []llmprovider.Warning
	for ev, err := range p.(llmprovider.Streamer).Stream(context.Background(), req) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if ev.Type == llmprovider.EventDone {
			warnings = ev.Response.Warnings
		}
	}
	if len(warnings) != 1 || warnings[0].Kind != llmprovider.WarningForcedChoiceDegraded {
		t.Errorf("done event's warnings = %+v, want one degraded warning", warnings)
	}
}
