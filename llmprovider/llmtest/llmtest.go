// Package llmtest checks a provider against the llmprovider contract, and
// offers Fake, a scriptable provider for tests (0015-MADR D11).
//
// Every built-in provider passes Run, and so must a third-party provider that
// claims conformance. Run it under the race detector (go test -race), which
// is what makes the concurrency check meaningful.
package llmtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Harness wires a provider to the fake server Run starts for each check.
type Harness struct {
	// New builds the provider under test, sending every request to baseURL,
	// with opts applied after its own.
	New func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error)
	// Text writes the service's successful reply with some output text.
	Text func(w http.ResponseWriter, r *http.Request)
	// ToolCall writes the service's successful reply calling the named tool
	// with the arguments {}. It is needed unless the provider's Tools
	// capability is Unsupported.
	ToolCall func(w http.ResponseWriter, r *http.Request, tool string)
	// Error writes the service's error reply with the given status.
	Error func(w http.ResponseWriter, r *http.Request, status int)
	// Model, when set, is the model the Text reply names: the Response must
	// report it (R7; 0020-MADR F11).
	Model string
	// NoReauth, when set, says why the R16 reauth check does not apply: for
	// example, the credential New is given decides the provider's mode, so
	// the check's own token source would build a different provider. Empty
	// runs the check.
	NoReauth string
}

// Run checks the provider h builds against the contract. Each check is a
// subtest, and each failure names the standards-guide rule it breaks
// (docs/guides/api-standards.md):
//   - capabilities honoured, and a request needing an Unsupported one
//     refused before the network (R10, R11);
//   - invalid values refused before the network (R23);
//   - cancellation (R40);
//   - status-to-kind classification (R25, R26);
//   - identity headers (R44; 0012-MADR §1.4);
//   - a refused token renewed once: an HTTP 401 invalidates an
//     llmprovider.InvalidatingSource and the request is sent once more (R16;
//     0017-MADR D3; 0020-MADR F2);
//   - the Response invariants (R7, R9), FinishReason and, when the Harness
//     names it, Model (0020-MADR F11);
//   - an empty Role, the user's, never sent as "" (R6; 0020-MADR F10);
//   - concurrent use (R20), which the race detector checks.
func Run(t *testing.T, h Harness) {
	t.Helper()
	runChecks(testReporter{t}, h)
}

// reporter is the part of *testing.T the checks use, so this package's own
// tests can record what a flawed provider reports.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
	Run(name string, f func(reporter))
}

type testReporter struct{ t *testing.T }

func (r testReporter) Helper() { r.t.Helper() }

func (r testReporter) Errorf(format string, args ...any) {
	r.t.Helper()
	r.t.Errorf(format, args...)
}

func (r testReporter) Run(name string, f func(reporter)) {
	r.t.Run(name, func(t *testing.T) { f(testReporter{t}) })
}

const (
	// llmtestTool is the tool the checks offer.
	llmtestTool = "llmtest_tool"
	// callTimeout bounds every call a check makes.
	callTimeout = 10 * time.Second
	// cancelGrace is how soon Generate must return once its context ends.
	cancelGrace = 2 * time.Second
	// concurrentCalls is how many calls the concurrency check overlaps.
	concurrentCalls = 8
	clientName      = "llmtest"
	clientVersion   = "0.0.1"
)

func runChecks(r reporter, h Harness) {
	r.Helper()
	if h.New == nil || h.Text == nil || h.Error == nil {
		r.Errorf("llmtest: the Harness needs New, Text and Error")
		return
	}
	r.Run("R10-R11-capabilities", func(r reporter) { checkCapabilities(r, h) })
	r.Run("R23-invalid-values", func(r reporter) { checkInvalidValues(r, h) })
	r.Run("R40-cancellation", func(r reporter) { checkCancellation(r, h) })
	r.Run("R25-R26-classification", func(r reporter) { checkClassification(r, h) })
	r.Run("R44-identity", func(r reporter) { checkIdentity(r, h) })
	r.Run("R16-reauth", func(r reporter) { checkReauth(r, h) })
	r.Run("R7-R9-response", func(r reporter) { checkResponse(r, h) })
	r.Run("R6-empty-role", func(r reporter) { checkEmptyRole(r, h) })
	r.Run("R20-concurrency", func(r reporter) { checkConcurrency(r, h) })
}

// fakeServer counts the requests it receives and records their User-Agent.
type fakeServer struct {
	*httptest.Server
	count  atomic.Int32
	mu     sync.Mutex
	agents []string
}

func serve(handler func(http.ResponseWriter, *http.Request)) *fakeServer {
	fs := &fakeServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.count.Add(1)
		fs.mu.Lock()
		fs.agents = append(fs.agents, r.UserAgent())
		fs.mu.Unlock()
		handler(w, r)
	}))
	return fs
}

func (fs *fakeServer) userAgents() []string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return append([]string(nil), fs.agents...)
}

// build makes the provider under test against fs, or reports why not.
func build(r reporter, h Harness, fs *fakeServer, opts ...llmprovider.Option) llmprovider.Provider {
	r.Helper()
	p, err := h.New(fs.URL, opts...)
	if err != nil || p == nil {
		r.Errorf("llmtest: the Harness's New failed: %v", err)
		return nil
	}
	return p
}

// generate calls p.Generate with a bounded context, and reports a result
// that breaks R7: a panic, both a response and an error, or neither.
func generate(r reporter, p llmprovider.Provider, req *llmprovider.Request) (resp *llmprovider.Response, err error) {
	r.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	defer func() {
		if v := recover(); v != nil {
			r.Errorf("R7 (Response invariants): Generate panicked: %v", v)
			resp, err = nil, fmt.Errorf("llmtest: Generate panicked: %v", v)
		}
	}()
	resp, err = p.Generate(ctx, req)
	switch {
	case err != nil && resp != nil:
		r.Errorf("R7 (Response invariants): Generate returned both a response and the error %v", err)
	case err == nil && resp == nil:
		r.Errorf("R7 (Response invariants): Generate returned neither a response nor an error")
	}
	return resp, err
}

func textRequest() *llmprovider.Request {
	return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "llmtest"}}}
}

func toolRequest(choice llmprovider.ToolChoice) *llmprovider.Request {
	req := textRequest()
	req.Tools = []llmprovider.Tool{{
		Name:        llmtestTool,
		Description: "A tool the conformance suite offers.",
		Schema:      map[string]any{"type": "object", "properties": map[string]any{}},
	}}
	req.ToolChoice = choice
	return req
}

func checkCapabilities(r reporter, h Harness) {
	r.Helper()
	probe := serve(h.Text)
	p := build(r, h, probe)
	probe.Close()
	if p == nil {
		return
	}
	caps := p.Capabilities()
	if _, streams := p.(llmprovider.Streamer); streams != (caps.NativeStreaming != llmprovider.Unsupported) {
		r.Errorf("R10 (capabilities are data): NativeStreaming is %v, but implementing Streamer is %v", caps.NativeStreaming, streams)
	}
	if caps.ForcedToolChoice != llmprovider.Unsupported && caps.Tools == llmprovider.Unsupported {
		r.Errorf("R10 (capabilities are data): ForcedToolChoice is %v while Tools is Unsupported", caps.ForcedToolChoice)
	}
	for _, need := range []struct {
		name    string
		support llmprovider.Support
		req     *llmprovider.Request
		forced  bool
	}{
		{"tools", caps.Tools, toolRequest(llmprovider.ToolChoiceAuto), false},
		{"a forced tool choice", caps.ForcedToolChoice, toolRequest(llmprovider.ForceTool(llmtestTool)), true},
		{"reasoning", caps.Reasoning, withReasoning(textRequest()), false},
		{"continuation", caps.Continuation, withPrevious(textRequest()), false},
	} {
		handler := h.Text
		if need.forced {
			if h.ToolCall == nil {
				if need.support != llmprovider.Unsupported {
					r.Errorf("llmtest: the Harness needs ToolCall, because ForcedToolChoice is %v", need.support)
				}
				continue
			}
			handler = func(w http.ResponseWriter, req *http.Request) { h.ToolCall(w, req, llmtestTool) }
		}
		checkNeed(r, h, need.name, need.support, need.req, need.forced, handler)
	}
}

// checkNeed sends a request with one need, and checks the provider honours
// or refuses it as its Capabilities say.
func checkNeed(r reporter, h Harness, name string, support llmprovider.Support, req *llmprovider.Request, forced bool, handler func(http.ResponseWriter, *http.Request)) {
	r.Helper()
	fs := serve(handler)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	resp, err := generate(r, p, req)
	if support == llmprovider.Unsupported {
		if !errors.Is(err, llmprovider.ErrUnsupported) || !errors.Is(err, errors.ErrUnsupported) {
			r.Errorf("R11 (refusal before the network): a request needing %s, which is Unsupported, returned %v; want an error matching ErrUnsupported", name, err)
		}
		if n := fs.count.Load(); n != 0 {
			r.Errorf("R11 (refusal before the network): a request needing %s, which is Unsupported, sent %d request(s); want none", name, n)
		}
		return
	}
	switch {
	case err != nil:
		r.Errorf("R10 (capabilities honoured): a request needing %s, which is %v, failed: %v", name, support, err)
	case forced && support == llmprovider.Supported && !hasCall(resp, llmtestTool):
		r.Errorf("R10 (capabilities honoured): ForcedToolChoice is Supported, but the response has no call to %q", llmtestTool)
	}
}

func withReasoning(req *llmprovider.Request) *llmprovider.Request {
	req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}
	return req
}

func withPrevious(req *llmprovider.Request) *llmprovider.Request {
	req.PreviousResponseID = "llmtest_previous"
	return req
}

func hasCall(resp *llmprovider.Response, tool string) bool {
	if resp == nil {
		return false
	}
	for _, item := range resp.Output {
		if call, ok := item.(llmprovider.FunctionCallItem); ok && call.Name == tool {
			return true
		}
	}
	return false
}

func checkInvalidValues(r reporter, h Harness) {
	r.Helper()
	unknownChoice := textRequest()
	unknownChoice.ToolChoice = "sometimes"
	negativeLimit := textRequest()
	negativeLimit.MaxOutputTokens = -1
	unknownRole := textRequest()
	unknownRole.Input = []llmprovider.Item{llmprovider.MessageItem{Role: "model", Text: "hi"}}
	for _, c := range []struct {
		name string
		req  *llmprovider.Request
	}{
		{"an unknown tool choice", unknownChoice},
		{"a negative MaxOutputTokens", negativeLimit},
		{"an unknown role", unknownRole},
		{"a nil request", nil},
	} {
		fs := serve(h.Text)
		if p := build(r, h, fs); p != nil {
			_, err := generate(r, p, c.req)
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				r.Errorf("R23 (invalid values): %s returned %v; want an error matching ErrInvalidRequest", c.name, err)
			}
			if n := fs.count.Load(); n != 0 {
				r.Errorf("R23 (invalid values): %s sent %d request(s); want none", c.name, n)
			}
		}
		fs.Close()
	}
}

func checkCancellation(r reporter, h Harness) {
	r.Helper()
	arrived := make(chan struct{}, 1)
	release := make(chan struct{})
	fs := serve(func(w http.ResponseWriter, req *http.Request) {
		select {
		case arrived <- struct{}{}:
		default:
		}
		select {
		case <-req.Context().Done():
			return
		case <-release:
		}
		h.Text(w, req)
	})
	defer fs.Close()
	defer close(release)
	p := build(r, h, fs)
	if p == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := p.Generate(ctx, textRequest())
		done <- err
	}()
	select {
	case <-arrived:
	case err := <-done:
		r.Errorf("R40 (cancellation): Generate returned %v before its request reached the server", err)
		return
	case <-time.After(callTimeout):
		r.Errorf("R40 (cancellation): the request never reached the server")
		return
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			r.Errorf("R40 (cancellation): after its context was cancelled, Generate returned %v; want an error matching context.Canceled", err)
		}
	case <-time.After(cancelGrace):
		r.Errorf("R40 (cancellation): Generate did not return within %v of its context being cancelled", cancelGrace)
	}
}

func checkClassification(r reporter, h Harness) {
	r.Helper()
	for _, c := range []struct {
		status    int
		kind      error
		kindName  string
		retryable bool
	}{
		{http.StatusBadRequest, llmprovider.ErrInvalidRequest, "ErrInvalidRequest", false},
		{http.StatusUnauthorized, llmprovider.ErrAuthFailure, "ErrAuthFailure", false},
		{http.StatusTooManyRequests, llmprovider.ErrRateLimited, "ErrRateLimited", true},
		{http.StatusInternalServerError, llmprovider.ErrProviderUnavailable, "ErrProviderUnavailable", true},
		{http.StatusServiceUnavailable, llmprovider.ErrProviderUnavailable, "ErrProviderUnavailable", true},
	} {
		fs := serve(func(w http.ResponseWriter, req *http.Request) { h.Error(w, req, c.status) })
		if p := build(r, h, fs); p != nil {
			_, err := generate(r, p, textRequest())
			// An APIError also unwraps to the sentinel its status alone maps
			// to (MADR 0012 §7), so its Kind is checked on its own.
			var apiErr *llmprovider.APIError
			isAPIErr := errors.As(err, &apiErr)
			switch {
			case !errors.Is(err, c.kind):
				r.Errorf("R25 (errors by kind): HTTP %d returned %v; want an error matching %s", c.status, err, c.kindName)
			case isAPIErr && apiErr.Kind != nil && !errors.Is(apiErr.Kind, c.kind):
				r.Errorf("R25 (errors by kind): HTTP %d returned an APIError of kind %v; want %s", c.status, apiErr.Kind, c.kindName)
			case isAPIErr && apiErr.Retryable() != c.retryable:
				r.Errorf("R26 (Retryable): HTTP %d: Retryable() is %v; want %v", c.status, apiErr.Retryable(), c.retryable)
			}
		}
		fs.Close()
	}
}

func checkIdentity(r reporter, h Harness) {
	r.Helper()
	fs := serve(h.Text)
	defer fs.Close()
	p := build(r, h, fs, llmprovider.WithClientInfo(clientName, clientVersion))
	if p == nil {
		return
	}
	if _, err := generate(r, p, textRequest()); err != nil {
		r.Errorf("R44 (identity): Generate failed: %v", err)
		return
	}
	want := clientName + "/" + clientVersion + " "
	for _, agent := range fs.userAgents() {
		if !strings.HasPrefix(agent, want) || !strings.Contains(agent, " go-llmprovider-sdk/") {
			r.Errorf("R44 (identity, 0012-MADR §1.4): User-Agent %q; want it to start %q and name go-llmprovider-sdk", agent, want)
		}
	}
}

func checkResponse(r reporter, h Harness) {
	r.Helper()
	fs := serve(h.Text)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	resp, err := generate(r, p, textRequest())
	if err != nil || resp == nil {
		r.Errorf("R7 (Response invariants): a text reply returned %v", err)
		return
	}
	if len(resp.Output) == 0 || resp.OutputText() == "" {
		r.Errorf("R7 (Response invariants): a text reply has no output text")
	}
	if resp.FinishReason == "" {
		r.Errorf("R7 (Response invariants, 0020-MADR F11): a text reply has no FinishReason")
	}
	if h.Model != "" && resp.Model != h.Model {
		r.Errorf("R7 (Response invariants, 0020-MADR F11): Model = %q; the reply names %q", resp.Model, h.Model)
	}
	for i, item := range resp.Output {
		switch it := item.(type) {
		case llmprovider.MessageItem, llmprovider.ReasoningItem, llmprovider.FunctionCallOutputItem:
		case llmprovider.FunctionCallItem:
			if it.Name == "" || (it.Arguments != "" && !json.Valid([]byte(it.Arguments))) {
				r.Errorf("R7 (Response invariants): output %d is a call with name %q and arguments %q", i, it.Name, it.Arguments)
			}
		default:
			r.Errorf("R9 (Item is sealed): output %d is %T", i, item)
		}
	}
	u := resp.Usage
	if u.InputTokens < 0 || u.OutputTokens < 0 || u.ReasoningTokens < 0 || u.CachedTokens < 0 {
		r.Errorf("R7 (Response invariants): negative usage %+v", u)
	}
	// A total holds its part (0015-MADR amendment "what `Usage` counts").
	if u.CachedTokens > u.InputTokens || u.ReasoningTokens > u.OutputTokens {
		r.Errorf("R7 (Response invariants): usage %+v has a part larger than its total", u)
	}
}

func checkConcurrency(r reporter, h Harness) {
	r.Helper()
	fs := serve(h.Text)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	errs := make(chan error, concurrentCalls)
	var wg sync.WaitGroup
	for range concurrentCalls {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
			defer cancel()
			_, err := p.Generate(ctx, textRequest())
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			r.Errorf("R20 (concurrent use): a concurrent Generate failed: %v", err)
		}
	}
}

const (
	// reauthRefused and reauthFresh are the tokens checkReauth's source hands
	// out before and after it is invalidated.
	reauthRefused = "llmtest-refused-token"
	reauthFresh   = "llmtest-fresh-token"
	// reauthHeader is the header the source names, so every provider sends
	// the token, even one that sends none of its own (R16).
	reauthHeader = "X-Llmtest-Credential"
)

// renewableToken is an InvalidatingSource: it hands out reauthRefused until
// it is invalidated, then reauthFresh.
type renewableToken struct {
	mu          sync.Mutex
	invalidated int
}

func (s *renewableToken) Token(context.Context) (llmprovider.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.invalidated > 0 {
		return llmprovider.Token{Value: reauthFresh, Header: reauthHeader}, nil
	}
	return llmprovider.Token{Value: reauthRefused, Header: reauthHeader}, nil
}

func (s *renewableToken) Invalidate() {
	s.mu.Lock()
	s.invalidated++
	s.mu.Unlock()
}

func (s *renewableToken) invalidations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.invalidated
}

// carries reports whether any of req's headers, or its query, holds value.
func carries(req *http.Request, value string) bool {
	for _, values := range req.Header {
		for _, v := range values {
			if strings.Contains(v, value) {
				return true
			}
		}
	}
	return strings.Contains(req.URL.RawQuery, value)
}

func checkReauth(r reporter, h Harness) {
	r.Helper()
	if h.NoReauth != "" {
		return
	}
	src := &renewableToken{}
	fs := serve(func(w http.ResponseWriter, req *http.Request) {
		if carries(req, reauthRefused) {
			h.Error(w, req, http.StatusUnauthorized)
			return
		}
		h.Text(w, req)
	})
	defer fs.Close()
	p := build(r, h, fs, llmprovider.WithTokenSource(src))
	if p == nil {
		return
	}
	_, err := generate(r, p, textRequest())
	if err != nil || src.invalidations() != 1 || fs.count.Load() != 2 {
		r.Errorf("R16 (a refused token is renewed once; 0017-MADR D3): after a 401, %d invalidation(s) and %d request(s), error %v; "+
			"want the source invalidated once and the request sent once more, with the fresh token",
			src.invalidations(), fs.count.Load(), err)
	}
}

// checkEmptyRole sends a message with no Role, which is the user's, and fails
// if any request body carries it as "role":"" (R6; 0020-MADR F10).
func checkEmptyRole(r reporter, h Harness) {
	r.Helper()
	var mu sync.Mutex
	var bodies []string
	fs := serve(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
		if err != nil {
			body = []byte("(request body unreadable: " + err.Error() + ")")
		}
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		h.Text(w, req)
	})
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Text: "llmtest"}}}
	if _, err := generate(r, p, req); err != nil {
		r.Errorf("R6 (an empty Role is the user's): Generate failed: %v", err)
		return
	}
	mu.Lock()
	defer mu.Unlock()
	for _, body := range bodies {
		if strings.Contains(strings.ReplaceAll(body, " ", ""), `"role":""`) {
			r.Errorf("R6 (an empty Role is the user's, 0020-MADR F10): the request carries \"role\":\"\": %.200s", body)
		}
	}
}
