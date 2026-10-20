package llmtest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

const (
	// partStep is how long the live check waits, after each part of
	// StreamText served to the provider itself, for its first text delta.
	partStep = 50 * time.Millisecond
	// liveLimit is how long the live check holds the rest of StreamText back
	// for the wrapped provider's first text delta.
	liveLimit = 2 * time.Second
)

// streamRetryPolicy is the policy the stream retry check wraps a provider
// in: its waits are a test's.
func streamRetryPolicy() llmprovider.RetryPolicy {
	return llmprovider.RetryPolicy{BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}
}

// checkStreamRetry checks a native stream through llmprovider.WithStreamRetry
// (0051-MADR D5): a 503 before the first event is retried, with one notice,
// and the stream then succeeds as D7 has it; a failure after the first event
// ends the stream, unretried; and the stream is live, with the provider's
// own events (checkStreamLive).
func checkStreamRetry(r reporter, h Harness) {
	r.Helper()
	text := serve(h.Text)
	defer text.Close()
	p := build(r, h, text)
	if p == nil {
		return
	}
	if _, ok := p.(llmprovider.Streamer); !ok {
		return // the check is for a provider that streams natively
	}
	want, err := generate(r, p, textRequest())
	if err != nil {
		r.Errorf("D7-stream-retry (0051-MADR D5): Generate on the Text reply returned %v", err)
		return
	}

	var sent atomic.Bool
	retried := serve(func(w http.ResponseWriter, req *http.Request) {
		if !sent.Swap(true) {
			h.Error(w, req, http.StatusServiceUnavailable)
			return
		}
		h.StreamText(w, req)
	})
	defer retried.Close()
	run, ok := streamRetried(r, h, retried)
	if !ok {
		return
	}
	done := lastDone(run.events)
	switch {
	case len(run.errs) > 0:
		r.Errorf("D7-stream-retry (0051-MADR D2): a 503, then the StreamText reply: %v; want the 503 retried and the stream to succeed", run.errs[0])
	case retried.count.Load() != 2 || run.notices != 1:
		r.Errorf("D7-stream-retry (0051-MADR D2): a 503, then the StreamText reply: %d requests and %d notices; want 2 and 1",
			retried.count.Load(), run.notices)
	case done == nil || !reflect.DeepEqual(done, want):
		r.Errorf("D7-stream-retry (0051-MADR D2): after the retry the stream's events are %v, EventDone's Response %+v; want it last, with Generate's %+v",
			eventTypes(run.events), done, want)
	case joinedDeltas(run.events) != messageText(done):
		r.Errorf("D7-stream-retry (0051-MADR D2): after the retry the text deltas joined are %q; EventDone's message text is %q",
			joinedDeltas(run.events), messageText(done))
	}
	if run.tracker.open() {
		r.Errorf("D7-stream-retry (0031-MADR D3): a retried stream left a reply open")
	}

	failing := serve(h.StreamFailure)
	defer failing.Close()
	run, ok = streamRetried(r, h, failing)
	if !ok {
		return
	}
	switch {
	case len(run.errs) != 1 || run.at[0] != len(run.events):
		r.Errorf("D7-stream-retry (0051-MADR D2): a stream failing after a delta yielded %d errors, events %v; want one error, last",
			len(run.errs), eventTypes(run.events))
	case failing.count.Load() != 1 || run.notices != 0:
		r.Errorf("D7-stream-retry (0051-MADR D2): a stream failing after a delta was sent %d times, with %d notices; want once, with none: a failure after the first event is not retried",
			failing.count.Load(), run.notices)
	}
	if hasDone(run.events) {
		r.Errorf("D7-stream-retry (0031-MADR D4): a failing stream yielded EventDone")
	}
	if run.tracker.open() {
		r.Errorf("D7-stream-retry (0031-MADR D3): a failed stream left its reply open")
	}

	checkStreamLive(r, h)
}

// retriedRun is what a stream through WithStreamRetry yielded: its events,
// each error with the index it came at, the notices sent, and the bodies the
// client opened.
type retriedRun struct {
	events  []llmprovider.Event
	errs    []error
	at      []int
	notices int
	tracker *bodyTracker
}

// streamRetried streams the text request against fs from the provider h
// builds, wrapped in WithStreamRetry with a counting notify, through a
// tracking client. It reports false when the provider cannot be built.
func streamRetried(r reporter, h Harness, fs *fakeServer) (retriedRun, bool) {
	r.Helper()
	run := retriedRun{tracker: &bodyTracker{}}
	defer run.tracker.base.CloseIdleConnections()
	p := build(r, h, fs, llmprovider.WithHTTPClient(&http.Client{Transport: run.tracker}))
	if p == nil {
		return run, false
	}
	var notices atomic.Int32
	w := llmprovider.WithStreamRetry(p, streamRetryPolicy(), func(context.Context, llmprovider.RetryNotice) { notices.Add(1) })
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	for ev, err := range llmprovider.Stream(ctx, w, textRequest()) {
		if err != nil {
			run.errs, run.at = append(run.errs, err), append(run.at, len(run.events))
			continue
		}
		run.events = append(run.events, ev)
	}
	run.notices = int(notices.Load())
	return run, true
}

// checkStreamLive checks that a stream through WithStreamRetry is live, by
// 0051-MADR M5's heuristic. StreamText, split into the parts that end in a
// blank line, is served to the provider itself one part at a time, to find
// the part after which its first text delta arrives; then served to the
// wrapped provider up to that part, the rest held back until the caller has
// a text delta, or liveLimit has passed. A wrapper that holds the stream
// back has none by then, since the provider's own stream has not ended. The
// wrapped stream's events must also be the provider's own, in order, which
// shows, when StreamText has two deltas, that none is merged or dropped.
func checkStreamLive(r reporter, h Harness) {
	r.Helper()
	var parts, after atomic.Int64
	after.Store(-1)
	ownSeen := newOnce()
	own, ok := streamStaged(r, h, ownSeen, false, func(i, n int) {
		parts.Store(int64(n))
		if i == 0 || after.Load() >= 0 {
			return
		}
		select {
		case <-ownSeen.c:
			after.Store(int64(i - 1))
		case <-time.After(partStep):
		}
	})
	if !ok {
		return
	}
	if len(own.errs) > 0 {
		r.Errorf("D7-stream-retry (0051-MADR D5): the StreamText reply, served in parts, failed: %v", own.errs[0])
		return
	}
	n, k := int(parts.Load()), int(after.Load())
	switch {
	case n < 2:
		r.Logf("D7-stream-retry (0051-MADR D5): StreamText is written in one part, so the stream through WithStreamRetry was not checked live; write it as an event stream, each event ending in a blank line")
		return
	case k < 0:
		r.Logf("D7-stream-retry (0051-MADR D5): the provider's own first text delta came only with the last of StreamText's %d parts, so the stream through WithStreamRetry was not checked live", n)
		return
	}
	if deltas := countDeltas(own.events); deltas == 1 {
		r.Logf("D7-stream-retry (0051-MADR D5): StreamText carries one text delta; two or more would also show that the stream through WithStreamRetry merges none")
	}

	var held atomic.Bool
	wrappedSeen := newOnce()
	wrapped, ok := streamStaged(r, h, wrappedSeen, true, func(i, _ int) {
		if i != k+1 {
			return
		}
		select {
		case <-wrappedSeen.c:
		case <-time.After(liveLimit):
			held.Store(true)
		}
	})
	if !ok {
		return
	}
	if held.Load() {
		r.Errorf("D7-stream-retry (0051-MADR D5): no text delta through WithStreamRetry before StreamText's part %d of %d was sent; the stream is held back, not live",
			k+2, n)
	}
	if !reflect.DeepEqual(wrapped.events, own.events) || len(wrapped.errs) > 0 {
		r.Errorf("D7-stream-retry (0051-MADR D2): through WithStreamRetry the stream's events are %v (errors %v); the provider's own are %v",
			eventTypes(wrapped.events), wrapped.errs, eventTypes(own.events))
	}
}

// once is a channel closed at most once.
type once struct {
	c    chan struct{}
	done sync.Once
}

func newOnce() *once { return &once{c: make(chan struct{})} }

func (o *once) close() { o.done.Do(func() { close(o.c) }) }

// streamStaged streams the text request from the provider h builds, wrapped
// in WithStreamRetry when wrap is set, against a server that writes
// StreamText's parts one by one, calling hold(i, n) before part i of n.
// seen is closed at the first text delta the caller receives, or at the end.
func streamStaged(r reporter, h Harness, seen *once, wrap bool, hold func(i, n int)) (retriedRun, bool) {
	r.Helper()
	fs := serve(func(w http.ResponseWriter, req *http.Request) {
		rec := httptest.NewRecorder()
		h.StreamText(rec, req)
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)
		flusher, canFlush := w.(http.Flusher)
		chunks := splitParts(rec.Body.Bytes())
		for i, c := range chunks {
			hold(i, len(chunks))
			if _, err := w.Write(c); err != nil {
				return // the client has gone
			}
			if canFlush {
				flusher.Flush()
			}
		}
	})
	defer fs.Close()
	defer seen.close()
	p := build(r, h, fs)
	if p == nil {
		return retriedRun{}, false
	}
	if wrap {
		p = llmprovider.WithStreamRetry(p, streamRetryPolicy(), nil)
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	var run retriedRun
	for ev, err := range llmprovider.Stream(ctx, p, textRequest()) {
		if err != nil {
			run.errs = append(run.errs, err)
			continue
		}
		if ev.Type == llmprovider.EventTextDelta {
			seen.close()
		}
		run.events = append(run.events, ev)
	}
	return run, true
}

// splitParts splits an event stream into its parts, each ending in a blank
// line; a body with none is one part.
func splitParts(body []byte) [][]byte {
	chunks := bytes.SplitAfter(body, []byte("\n\n"))
	if n := len(chunks); n > 1 && len(chunks[n-1]) == 0 {
		chunks = chunks[:n-1]
	}
	return chunks
}

// lastDone is the Response of events' last event when it is EventDone and
// the only one.
func lastDone(events []llmprovider.Event) *llmprovider.Response {
	n := len(events)
	if n == 0 || events[n-1].Type != llmprovider.EventDone || hasDone(events[:n-1]) {
		return nil
	}
	return events[n-1].Response
}

// hasDone reports whether any of events is EventDone.
func hasDone(events []llmprovider.Event) bool {
	for _, ev := range events {
		if ev.Type == llmprovider.EventDone {
			return true
		}
	}
	return false
}

// joinedDeltas is the text of events' text deltas, joined.
func joinedDeltas(events []llmprovider.Event) string {
	var b strings.Builder
	for _, ev := range events {
		if ev.Type == llmprovider.EventTextDelta {
			b.WriteString(ev.Text)
		}
	}
	return b.String()
}

// countDeltas counts events' text deltas.
func countDeltas(events []llmprovider.Event) int {
	n := 0
	for _, ev := range events {
		if ev.Type == llmprovider.EventTextDelta {
			n++
		}
	}
	return n
}

// messageText is resp's message text, joined.
func messageText(resp *llmprovider.Response) string {
	if resp == nil {
		return ""
	}
	var b strings.Builder
	for _, item := range resp.Output {
		if m, ok := item.(llmprovider.MessageItem); ok {
			b.WriteString(m.Text)
		}
	}
	return b.String()
}
