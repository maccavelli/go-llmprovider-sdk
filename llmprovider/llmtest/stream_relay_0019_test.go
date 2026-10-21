package llmtest

import (
	"context"
	"io"
	"iter"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// headerProbe streams as streamRef does, after one request of its own that
// keeps the reply's headers.
type headerProbe struct {
	*streamRef
	got http.Header
}

func (p *headerProbe) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/stream", http.NoBody)
		if err != nil {
			yield(llmprovider.Event{}, err)
			return
		}
		resp, err := p.client.Do(httpReq)
		if err != nil {
			yield(llmprovider.Event{}, err)
			return
		}
		p.got = resp.Header.Clone()
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		p.streamRef.Stream(ctx, req)(yield)
	}
}

// TestStreamStaged_RelaysHeaders_0019 (0019-MADR, amendment of 2026-10-20):
// the server streamStaged starts sends the headers StreamText set, so a
// provider that reads one, such as Content-Type, sees what the harness wrote.
func TestStreamStaged_RelaysHeaders_0019(t *testing.T) {
	h := streamHarness(twoDeltas0051, false)
	h.StreamText = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Llmtest-Probe", "kept")
		_, _ = io.WriteString(w, twoDeltas0051)
	}
	var probe *headerProbe
	base := h.New
	h.New = func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		p, err := base(baseURL, opts...)
		if err != nil {
			return nil, err
		}
		probe = &headerProbe{streamRef: p.(*streamRef)}
		return probe, nil
	}
	rec := newRecorder()
	if _, ok := streamStaged(rec, h, newOnce(), false, func(int, int) {}); !ok || probe == nil || probe.got == nil {
		t.Fatalf("streamStaged did not stream: failures %q", rec.failures())
	}
	if probe.got.Get("X-Llmtest-Probe") != "kept" || probe.got.Get("Content-Type") != "text/event-stream" {
		t.Errorf("the relayed reply's headers are %v; want StreamText's X-Llmtest-Probe and Content-Type", probe.got)
	}
}
