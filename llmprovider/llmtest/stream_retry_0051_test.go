package llmtest

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// streamRef is the reference provider with a native stream, on a wire of its
// own: a line "data: TEXT" is a text delta, "data: !fail" a failure, and
// "data: [DONE]" the end. With collect set, it reads the whole reply before
// yielding any of it, so its own stream is not live.
type streamRef struct {
	*refProvider
	collect bool
}

func (p *streamRef) Capabilities() llmprovider.Capabilities {
	caps := p.refProvider.Capabilities()
	caps.NativeStreaming = llmprovider.Supported
	return caps
}

func (p *streamRef) Stream(ctx context.Context, _ *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	return func(yield func(llmprovider.Event, error) bool) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/stream", http.NoBody)
		if err != nil {
			yield(llmprovider.Event{}, err)
			return
		}
		httpReq.Header.Set("User-Agent", p.userAgent)
		resp, err := p.client.Do(httpReq)
		if err != nil {
			yield(llmprovider.Event{}, fmt.Errorf("%w: %w", llmprovider.ErrProviderUnavailable, err))
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			yield(llmprovider.Event{}, p.failure(resp))
			return
		}
		var body io.Reader = resp.Body
		if p.collect {
			all, _ := io.ReadAll(resp.Body)
			body = bytes.NewReader(all)
		}
		var text strings.Builder
		for lines := bufio.NewScanner(body); lines.Scan(); {
			data, ok := strings.CutPrefix(lines.Text(), "data: ")
			switch {
			case !ok:
			case data == "[DONE]":
				item := llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: text.String()}
				if yield(llmprovider.Event{Type: llmprovider.EventItem, Item: item}, nil) {
					yield(llmprovider.Event{Type: llmprovider.EventDone, Response: &llmprovider.Response{
						FinishReason: llmprovider.FinishStop, Output: []llmprovider.Item{item}}}, nil)
				}
				return
			case data == "!fail":
				yield(llmprovider.Event{}, fmt.Errorf("%w: reference: the stream failed", llmprovider.ErrProviderUnavailable))
				return
			default:
				text.WriteString(data)
				if !yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: data}, nil) {
					return
				}
			}
		}
		yield(llmprovider.Event{}, fmt.Errorf("%w: reference: the stream ended early", llmprovider.ErrProviderUnavailable))
	}
}

// The reference's StreamText scripts, each the text Text writes.
const (
	twoDeltas0051 = "data: hello from \n\ndata: the reference provider\n\ndata: [DONE]\n\n"
	oneDelta0051  = "data: hello from the reference provider\n\ndata: [DONE]\n\n"
	onePart0051   = "data: hello from the reference provider\ndata: [DONE]\n\n"
)

// streamHarness is the reference harness, its provider streaming natively
// with script as StreamText.
func streamHarness(script string, collect bool) Harness {
	h := refHarness(flaws{})
	base := h.New
	h.New = func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		p, err := base(baseURL, opts...)
		if err != nil {
			return nil, err
		}
		return &streamRef{refProvider: p.(*refProvider), collect: collect}, nil
	}
	h.StreamText = func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, script) }
	h.StreamFailure = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "data: hello\n\ndata: !fail\n\n")
	}
	return h
}

// streamRetryLogs runs every check on h, and returns the failures and the
// D7-stream-retry lines logged.
func streamRetryLogs(h Harness) (failures, logs []string) {
	rec := newRecorder()
	runChecks(rec, h)
	for _, line := range rec.logged() {
		if strings.Contains(line, "D7-stream-retry") {
			logs = append(logs, line)
		}
	}
	return rec.failures(), logs
}

// TestStreamRetryCheck_TwoDeltas_0051 (0051-MADR D5): a provider that streams
// live, with two deltas, passes every check, the stream retry one included,
// and nothing is logged.
func TestStreamRetryCheck_TwoDeltas_0051(t *testing.T) {
	failures, logs := streamRetryLogs(streamHarness(twoDeltas0051, false))
	if len(failures) > 0 || len(logs) > 0 {
		t.Errorf("failures %q, logs %q; want neither", failures, logs)
	}
}

// TestStreamRetryCheck_OneDelta_0051 (0051-MADR D5): one delta passes, and is
// told that two would show more.
func TestStreamRetryCheck_OneDelta_0051(t *testing.T) {
	failures, logs := streamRetryLogs(streamHarness(oneDelta0051, false))
	if len(failures) > 0 || len(logs) != 1 || !strings.Contains(logs[0], "carries one text delta") {
		t.Errorf("failures %q, logs %q; want none, and the one-delta line", failures, logs)
	}
}

// TestStreamRetryCheck_OnePart_0051 (0051-MADR D5): a StreamText written in one
// part cannot show liveness: it passes, and the log says why.
func TestStreamRetryCheck_OnePart_0051(t *testing.T) {
	failures, logs := streamRetryLogs(streamHarness(onePart0051, false))
	if len(failures) > 0 || len(logs) != 1 || !strings.Contains(logs[0], "written in one part") {
		t.Errorf("failures %q, logs %q; want none, and the one-part line", failures, logs)
	}
}

// TestStreamRetryCheck_ProviderNotLive_0051 (0051-MADR D5): a provider whose
// own stream gives no delta before its reply ends cannot show the wrapper
// live: it passes, and the log says why.
func TestStreamRetryCheck_ProviderNotLive_0051(t *testing.T) {
	failures, logs := streamRetryLogs(streamHarness(twoDeltas0051, true))
	if len(failures) > 0 || len(logs) != 1 || !strings.Contains(logs[0], "came only with the last") {
		t.Errorf("failures %q, logs %q; want none, and the not-checked line", failures, logs)
	}
}
