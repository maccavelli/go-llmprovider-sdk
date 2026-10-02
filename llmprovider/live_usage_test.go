//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/kilo"
)

// The two live measurements of 0015-MADR amendment "what `Usage` counts".
// Each is opt-in through its key, as the other live tests are, and reports
// what the service sent.

// replyRecorder keeps each response body, passing the response on.
type replyRecorder struct {
	mu     sync.Mutex
	bodies [][]byte
}

func (r *replyRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil || resp.Body == nil {
		return resp, err
	}
	b, err := io.ReadAll(resp.Body)
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.bodies = append(r.bodies, b)
	r.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(b))
	return resp, nil
}

// last is the last response body, decoded into v.
func (r *replyRecorder) last(t *testing.T, v any) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.bodies) == 0 {
		t.Fatal("no response was recorded")
	}
	if err := json.Unmarshal(r.bodies[len(r.bodies)-1], v); err != nil {
		t.Fatalf("decode the response: %v", err)
	}
}

// TestLive_KiloReportsUsage measures whether Kilo reports usage when the
// request does not send usage: {include: true}, as this module's does not.
// Kilo's own client always sends it (0017-REPORT K2). On 2026-10-02 Kilo
// reported usage without it. A failure would be the finding that Kilo now
// needs it; that request change goes to the owner first.
func TestLive_KiloReportsUsage(t *testing.T) {
	rec := &replyRecorder{}
	p := liveKilo(t, llmprovider.LiveKiloFreeCollecting, kilo.WithDataCollection(true),
		llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	resp, err := p.Generate(ctx, userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		rec.mu.Lock()
		for _, b := range rec.bodies {
			t.Logf("response body: %.600s", b)
		}
		rec.mu.Unlock()
		t.Fatalf("Generate: %v", err)
	}
	var raw struct {
		Usage json.RawMessage `json:"usage"`
	}
	rec.last(t, &raw)
	t.Logf("raw usage: %s; decoded %+v", raw.Usage, resp.Usage)
	if resp.Usage.InputTokens == 0 || resp.Usage.OutputTokens == 0 {
		t.Fatalf("Kilo reported no usage without usage.include (raw %s)", raw.Usage)
	}
}

// TestLive_GeminiInteractionsThoughtTokens measures whether an Interaction's
// total_output_tokens includes total_thought_tokens. On 2026-10-02 it did not:
// the decoder adds the thoughts to OutputTokens, as for generateContent.
func TestLive_GeminiInteractionsThoughtTokens(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	rec := &replyRecorder{}
	req := userText("What is 17 * 23? Reply with only the number.")
	req.Reasoning = &llmprovider.Reasoning{}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	resp, err := liveGemini(t, key, "gemini-3.7-flash", llmprovider.WithHTTPClient(&http.Client{Transport: rec})).Generate(ctx, req)
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var raw struct {
		Usage struct {
			Input    int `json:"total_input_tokens"`
			Output   int `json:"total_output_tokens"`
			Thoughts int `json:"total_thought_tokens"`
			ToolUse  int `json:"total_tool_use_tokens"`
			Total    int `json:"total_tokens"`
		} `json:"usage"`
	}
	rec.last(t, &raw)
	u := raw.Usage
	t.Logf("raw usage: %+v; decoded %+v", u, resp.Usage)
	switch {
	case u.Thoughts == 0:
		t.Skip("the model spent no thought tokens; nothing to measure")
	case u.Total == u.Input+u.Output+u.Thoughts+u.ToolUse:
		t.Logf("thoughts are outside total_output_tokens: the decoder's reading holds")
	case u.Total == u.Input+u.Output+u.ToolUse:
		t.Fatalf("thoughts are inside total_output_tokens: the decoder double-counts them (%+v)", u)
	default:
		t.Fatalf("total_tokens %d matches neither reading (%+v)", u.Total, u)
	}
}
