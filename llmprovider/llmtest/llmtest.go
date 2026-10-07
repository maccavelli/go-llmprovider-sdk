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
	// AuthFailure, when set, writes the service's own refusal of a key, its
	// status and body, for the R16 reauth check: Gemini's is HTTP 400
	// API_KEY_INVALID. Nil is Error with status 401 (0026-MADR F6).
	AuthFailure func(w http.ResponseWriter, r *http.Request)
	// NoReauth, when set, says why the R16 reauth check does not apply: for
	// example, the credential New is given decides the provider's mode, so
	// the check's own token source would build a different provider. Empty
	// runs the check.
	NoReauth string

	// The fields below each switch on one more check (0021-MADR D5, W12). A
	// Harness that sets none passes as before.

	// Fidelity checks that what a request asks for reaches the wire: a
	// request with Model FidelityModel, Instructions FidelityInstructions
	// and a FunctionCallOutputItem whose Output is FidelityOutput must send
	// all three, in its path or its body.
	Fidelity bool
	// Garbled, when set, writes a 200 reply that cannot be decoded. Generate
	// must fail with ErrIncomplete, and, through WithRetry, send the request
	// once: the service answered.
	Garbled func(w http.ResponseWriter, r *http.Request)
	// Truncated, when set, writes a reply the service cut at its output
	// limit, holding a partial call. Generate must fail with ErrIncomplete,
	// and Reason TruncatedReason.
	Truncated func(w http.ResponseWriter, r *http.Request)
	// TruncatedReason is the Reason a cut answer reports on the provider's
	// wire: the service's own (APIError.Reason), such as "max_output_tokens"
	// on the Responses wire. Empty means "length".
	TruncatedReason string
	// StrictTools checks the ToolCall reply: FinishReason FinishToolCalls,
	// and each call's Arguments non-empty, valid JSON.
	StrictTools bool
	// ReasoningCut, when set, writes a reply the service cut at its output
	// limit while the model was still reasoning: reasoning, and no text or
	// call. Generate must fail with ErrIncomplete, and Reason
	// TruncatedReason, never succeed with no text (0026-MADR F7).
	ReasoningCut func(w http.ResponseWriter, r *http.Request)
}

// The values the Fidelity check sends and looks for on the wire.
const (
	FidelityModel        = "llmtest-model-7f3a"
	FidelityInstructions = "llmtest-instructions-7f3a"
	FidelityOutput       = "llmtest-output-7f3a"
)

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
//   - concurrent use (R20), which the race detector checks;
//   - when the Harness asks: request fidelity, an undecodable reply, a cut
//     answer, and a strict tool call (0021-MADR W12), and an answer cut while
//     reasoning (0026-MADR F7).
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
	if h.Fidelity {
		r.Run("W12-fidelity", func(r reporter) { checkFidelity(r, h) })
	}
	if h.Garbled != nil {
		r.Run("W12-garbled", func(r reporter) { checkGarbled(r, h) })
	}
	if h.Truncated != nil {
		r.Run("W12-truncated", func(r reporter) { checkTruncated(r, h) })
	}
	if h.StrictTools {
		r.Run("W12-strict-tools", func(r reporter) { checkStrictTools(r, h) })
	}
	if h.ReasoningCut != nil {
		r.Run("F7-reasoning-cut", func(r reporter) { checkReasoningCut(r, h) })
	}
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
	refuse := h.AuthFailure
	if refuse == nil {
		refuse = func(w http.ResponseWriter, req *http.Request) { h.Error(w, req, http.StatusUnauthorized) }
	}
	fs := serve(func(w http.ResponseWriter, req *http.Request) {
		if carries(req, reauthRefused) {
			refuse(w, req)
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
		r.Errorf("R16 (a refused token is renewed once; 0017-MADR D3; 0026-MADR F6): after the service refused the key, "+
			"%d invalidation(s) and %d request(s), error %v; "+
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

// recordingServer serves handler and keeps each request's path and body.
func recordingServer(handler func(http.ResponseWriter, *http.Request)) (*fakeServer, func() []string) {
	var mu sync.Mutex
	var seen []string
	fs := serve(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
		if err != nil {
			body = []byte("(request body unreadable: " + err.Error() + ")")
		}
		mu.Lock()
		seen = append(seen, req.URL.Path+" "+string(body))
		mu.Unlock()
		handler(w, req)
	})
	return fs, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

// checkFidelity sends a request's Model, Instructions and a tool's output,
// and requires each on the wire (0021-MADR W12).
func checkFidelity(r reporter, h Harness) {
	r.Helper()
	fs, seen := recordingServer(h.Text)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	req := toolRequest(llmprovider.ToolChoiceAuto)
	req.Model, req.Instructions = FidelityModel, FidelityInstructions
	req.Input = append(req.Input,
		llmprovider.FunctionCallItem{CallID: "call_llmtest", Name: llmtestTool, Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "call_llmtest", Output: FidelityOutput})
	if _, err := generate(r, p, req); err != nil {
		r.Errorf("W12 (fidelity): Generate failed: %v", err)
		return
	}
	wire := strings.Join(seen(), "\n")
	for _, want := range []struct{ field, value string }{
		{"Model", FidelityModel}, {"Instructions", FidelityInstructions}, {"a tool's output", FidelityOutput},
	} {
		if !strings.Contains(wire, want.value) {
			r.Errorf("W12 (fidelity, 0021-MADR D5): the request's %s, %q, never reached the wire: %.300s", want.field, want.value, wire)
		}
	}
}

// checkGarbled sends a request whose 200 reply cannot be decoded: it is
// ErrIncomplete, and WithRetry does not send it again.
func checkGarbled(r reporter, h Harness) {
	r.Helper()
	fs := serve(h.Garbled)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	retrying := llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond})
	_, err := generate(r, retrying, textRequest())
	if !errors.Is(err, llmprovider.ErrIncomplete) {
		r.Errorf("W12 (an undecodable reply, 0021-MADR D1): Generate returned %v; want an error matching ErrIncomplete", err)
	}
	if n := fs.count.Load(); n != 1 {
		r.Errorf("W12 (an undecodable reply, 0021-MADR D1): WithRetry sent %d request(s); want 1: the service answered", n)
	}
}

// checkTruncated sends a request whose reply was cut with a partial call: it
// is ErrIncomplete with Reason "length".
func checkTruncated(r reporter, h Harness) {
	r.Helper()
	fs := serve(h.Truncated)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	want := h.TruncatedReason
	if want == "" {
		want = string(llmprovider.FinishLength)
	}
	_, err := generate(r, p, toolRequest(llmprovider.ToolChoiceAuto))
	var apiErr *llmprovider.APIError
	if !errors.Is(err, llmprovider.ErrIncomplete) || !errors.As(err, &apiErr) || apiErr.Reason != want {
		r.Errorf("W12 (a cut answer, 0021-MADR W3): Generate returned %v; want an APIError of kind ErrIncomplete with Reason %q",
			err, want)
	}
}

// checkReasoningCut checks that an answer cut while the model was still
// reasoning is ErrIncomplete, with the wire's cut reason.
func checkReasoningCut(r reporter, h Harness) {
	r.Helper()
	fs := serve(h.ReasoningCut)
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	want := h.TruncatedReason
	if want == "" {
		want = string(llmprovider.FinishLength)
	}
	resp, err := generate(r, p, textRequest())
	var apiErr *llmprovider.APIError
	if !errors.Is(err, llmprovider.ErrIncomplete) || !errors.As(err, &apiErr) || apiErr.Reason != want {
		r.Errorf("0026-MADR F7 (an answer cut while reasoning): Generate returned %+v, %v; want an APIError of kind "+
			"ErrIncomplete with Reason %q", resp, err, want)
	}
}

// checkStrictTools checks a call reply finishes tool_calls with non-empty,
// valid JSON arguments.
func checkStrictTools(r reporter, h Harness) {
	r.Helper()
	if h.ToolCall == nil {
		r.Errorf("llmtest: StrictTools needs the Harness's ToolCall")
		return
	}
	fs := serve(func(w http.ResponseWriter, req *http.Request) { h.ToolCall(w, req, llmtestTool) })
	defer fs.Close()
	p := build(r, h, fs)
	if p == nil {
		return
	}
	resp, err := generate(r, p, toolRequest(llmprovider.ToolChoiceAuto))
	if err != nil || resp == nil {
		r.Errorf("W12 (strict tools): a call reply returned %v", err)
		return
	}
	if resp.FinishReason != llmprovider.FinishToolCalls {
		r.Errorf("W12 (strict tools, 0021-MADR W3): a call reply finished %q; want %q", resp.FinishReason, llmprovider.FinishToolCalls)
	}
	calls := 0
	for _, item := range resp.Output {
		if call, ok := item.(llmprovider.FunctionCallItem); ok {
			calls++
			if call.Arguments == "" || !json.Valid([]byte(call.Arguments)) {
				r.Errorf("W12 (strict tools, 0021-MADR W1): the call to %q has arguments %q; want non-empty, valid JSON", call.Name, call.Arguments)
			}
		}
	}
	if calls == 0 {
		r.Errorf("W12 (strict tools): a call reply has no FunctionCallItem")
	}
}
