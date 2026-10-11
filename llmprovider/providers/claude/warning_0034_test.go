package claude

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestGenerate_WarnsWhenDegraded_0034 (0034-MADR D7): a degraded forced choice
// puts a warning on the Response; a request sent as asked puts none.
func TestGenerate_WarnsWhenDegraded_0034(t *testing.T) {
	for _, c := range []struct {
		name, model string
		think       bool
		warns       bool
	}{
		{"a refusing model", "claude-sonnet-5-5", false, true},
		{"thinking on an accepting model", "claude-sonnet-5", true, true},
		{"an accepting model", "claude-sonnet-5", false, false},
	} {
		var body map[string]any
		srv := captureServer(t, &body, callReply0034)
		req := weatherRequest()
		req.ToolChoice = llmprovider.ForceTool("get_weather")
		if c.think {
			req.Reasoning = &llmprovider.Reasoning{}
		}
		resp, err := build(t, apiKey(srv.URL, c.model)...).Generate(context.Background(), req)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := len(resp.Warnings) == 1 && resp.Warnings[0].Kind == llmprovider.WarningForcedChoiceDegraded &&
			resp.Warnings[0].Model == c.model; got != c.warns || (!c.warns && len(resp.Warnings) != 0) {
			t.Errorf("%s: warnings %+v, want a degraded warning: %v", c.name, resp.Warnings, c.warns)
		}
	}
}

// TestStream_WarnsWhenDegraded_0034 (0034-MADR D7): the stream's done event
// carries the same warning.
func TestStream_WarnsWhenDegraded_0034(t *testing.T) {
	srv, _ := sseServer(t, messagesStream)
	req := weatherRequest()
	req.ToolChoice = llmprovider.ForceTool("get_weather")
	var warnings []llmprovider.Warning
	for ev, err := range build(t, apiKey(srv.URL, "claude-opus-5-5")...).(llmprovider.Streamer).Stream(context.Background(), req) {
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
