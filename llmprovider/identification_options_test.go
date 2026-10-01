package llmprovider

import (
	"context"
	"strings"
	"testing"
)

// TestIdentification_ClientInfoAndSessionOptions: WithClientInfo leads
// User-Agent and names Kilo's editor; WithSessionID reaches Kilo. OpenCode's
// half moved to providers/opencode (0015-PLAN S7).
func TestIdentification_ClientInfoAndSessionOptions(t *testing.T) {
	rec := newHeaderRecorder(t)
	opts := []ProviderOption{WithBaseURL(rec.srv.URL), WithClientInfo("pcm", "1.2.3"), WithSessionID("s-1")}
	kilo, err := NewKilo("k", "some/model", opts...)
	if err != nil {
		t.Fatalf("NewKilo: %v", err)
	}
	_, _ = kilo.Generate(context.Background(), "hello")

	reqs := rec.requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	for _, r := range reqs {
		if ua := r.header.Get("User-Agent"); !strings.HasPrefix(ua, "pcm/1.2.3 (") || !strings.Contains(ua, ") go-llmprovider-sdk/") {
			t.Errorf("%s: User-Agent = %q, want pcm/1.2.3 (…) go-llmprovider-sdk/…", r.path, ua)
		}
	}
	if h := reqs[0].header; h.Get(kiloEditorHeader) != "pcm" || h.Get(kiloTaskHeader) != "s-1" {
		t.Errorf("kilo editor/task = %q/%q, want pcm/s-1", h.Get(kiloEditorHeader), h.Get(kiloTaskHeader))
	}
}
