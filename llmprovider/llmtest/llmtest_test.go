package llmtest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// refID is the reference provider's id.
const refID llmprovider.ProviderID = "llmtest-reference"

// flaws are defects the reference provider can be built with, each one a
// rule Run must catch.
type flaws struct {
	ignoreCancel   bool // sends with a context of its own
	skipCheck      bool // never calls Capabilities.Check
	misclassify429 bool // reports a 429 as an unavailable service
	racy           bool // counts calls without a lock
	noReauth       bool // never renews a refused token
	noFinish       bool // never says why the answer ended
	emptyRole      bool // sends an empty Role as "role":""
	// The W12 checks' faults (0021-MADR W12).
	dropInstructions bool // never sends Request.Instructions
	decodeBypass     bool // reports an undecodable 200 as a retryable outage
	noLengthCheck    bool // takes a cut answer for a whole one
	emptyArguments   bool // reports a call's arguments as ""
	// keepReasoningCut takes an answer cut while reasoning for a whole one
	// (0026-MADR F7).
	keepReasoningCut bool
	// declaresNoForce declares ForcedToolChoice Unsupported, and sends a
	// forced choice all the same (0026-MADR F55).
	declaresNoForce bool
	// The 0026-MADR F57 checks' faults.
	notAPIError          bool // reports an HTTP failure as a plain error
	noRetryAfter         bool // drops a 429's Retry-After
	notPermittedAsAuth   bool // reports a not-permitted 403 as a refused key
	ignoresTokenHeader   bool // sends every token as Authorization: Bearer
	listingNoIdentity    bool // lists without its User-Agent
	listingIgnoresCancel bool // lists with a context of its own
}

// refProvider is a small conformant provider over a JSON wire: it POSTs
// {"text","tool","model","instructions","outputs"} and reads {"text"} or
// {"call":{"name","arguments"},"finish"}.
type refProvider struct {
	baseURL   string
	src       llmprovider.TokenSource
	client    *http.Client
	userAgent string
	flaws     flaws
	calls     int
}

var refCaps = llmprovider.Capabilities{
	Tools:            llmprovider.Supported,
	ForcedToolChoice: llmprovider.Supported,
	Reasoning:        llmprovider.BestEffort,
	Continuation:     llmprovider.Unsupported,
	NativeStreaming:  llmprovider.Unsupported,
}

func newRef(f flaws) func(string, ...llmprovider.Option) (llmprovider.Provider, error) {
	return func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		st, err := llmprovider.ResolveOptions(refID, opts)
		if err != nil {
			return nil, err
		}
		src := st.TokenSource()
		if src == nil {
			src = llmprovider.NewStaticToken("reference-key")
		}
		return &refProvider{baseURL: baseURL, src: src, client: st.HTTPClient(), userAgent: st.UserAgent(), flaws: f}, nil
	}
}

func (p *refProvider) ID() llmprovider.ProviderID { return refID }

func (p *refProvider) Capabilities() llmprovider.Capabilities {
	caps := refCaps
	if p.flaws.declaresNoForce {
		caps.ForcedToolChoice = llmprovider.Unsupported
	}
	return caps
}

func (p *refProvider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if p.flaws.racy {
		p.calls++
	}
	if !p.flaws.skipCheck {
		if err := refCaps.Check(req); err != nil {
			return nil, err
		}
	}
	if p.flaws.ignoreCancel {
		ctx = context.Background()
	}
	resp, err := p.generateOnce(ctx, req)
	var apiErr *llmprovider.APIError
	if p.flaws.noReauth || err == nil || !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		return resp, err
	}
	source, ok := p.src.(llmprovider.InvalidatingSource)
	if !ok {
		return resp, err
	}
	source.Invalidate()
	return p.generateOnce(ctx, req)
}

// generateOnce sends req once, with a token fetched for this send.
func (p *refProvider) generateOnce(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	tool, _ := req.ToolChoice.Tool()
	var outputs []string
	for _, item := range req.Input {
		if out, ok := item.(llmprovider.FunctionCallOutputItem); ok {
			outputs = append(outputs, out.Output)
		}
	}
	fields := map[string]string{"text": req.Input[0].(llmprovider.MessageItem).Text, "tool": tool,
		"model": req.Model, "instructions": req.Instructions, "outputs": strings.Join(outputs, ",")}
	if p.flaws.dropInstructions {
		delete(fields, "instructions")
	}
	if p.flaws.emptyRole {
		fields["role"] = string(req.Input[0].(llmprovider.MessageItem).Role)
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", p.userAgent)
	token, err := p.src.Token(ctx)
	if err != nil {
		return nil, err
	}
	if p.flaws.ignoresTokenHeader {
		httpReq.Header.Set("Authorization", "Bearer "+token.Value)
	} else {
		token.Apply(httpReq, "Authorization", "Bearer")
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, p.failure(resp)
	}
	var out struct {
		Text      string `json:"text"`
		Reasoning string `json:"reasoning"`
		Call      *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"call"`
		Finish string `json:"finish"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		if p.flaws.decodeBypass {
			return nil, fmt.Errorf("%w: %w", llmprovider.ErrProviderUnavailable, err)
		}
		return nil, fmt.Errorf("%w: %w", llmprovider.ErrIncomplete, err)
	}
	if out.Finish == string(llmprovider.FinishLength) && out.Call != nil && !p.flaws.noLengthCheck {
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}
	if out.Finish == string(llmprovider.FinishLength) && out.Call == nil && out.Text == "" && !p.flaws.keepReasoningCut {
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}
	result := &llmprovider.Response{FinishReason: llmprovider.FinishStop}
	if out.Call != nil {
		result.FinishReason = llmprovider.FinishToolCalls
	}
	if p.flaws.noFinish {
		result.FinishReason = ""
	}
	if out.Call != nil {
		arguments := out.Call.Arguments
		if p.flaws.emptyArguments {
			arguments = ""
		}
		result.Output = append(result.Output, llmprovider.FunctionCallItem{CallID: "call_1", Name: out.Call.Name, Arguments: arguments})
	}
	if out.Reasoning != "" {
		result.Output = append(result.Output, llmprovider.ReasoningItem{Text: out.Reasoning})
	}
	if out.Text != "" {
		result.Output = append(result.Output, llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: out.Text})
	}
	return result, nil
}

// failure is the reference wire's classified failure: an *APIError with the
// delay a 429 asked for, and a 403 that says not_permitted of kind
// ErrNotPermitted.
func (p *refProvider) failure(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	kind := p.kindFor(resp.StatusCode)
	if resp.StatusCode == http.StatusForbidden && bytes.Contains(body, []byte("not_permitted")) && !p.flaws.notPermittedAsAuth {
		kind = llmprovider.ErrNotPermitted
	}
	if p.flaws.notAPIError {
		return fmt.Errorf("%w: HTTP %d", kind, resp.StatusCode)
	}
	e := &llmprovider.APIError{Provider: string(refID), Status: resp.StatusCode, Kind: kind}
	if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && !p.flaws.noRetryAfter {
		e.RetryAfter = time.Duration(secs) * time.Second
	}
	return e
}

// ListModels lists the reference wire's one model.
func (p *refProvider) ListModels(ctx context.Context) ([]string, error) {
	if p.flaws.listingIgnoresCancel {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", http.NoBody)
	if err != nil {
		return nil, err
	}
	if !p.flaws.listingNoIdentity {
		req.Header.Set("User-Agent", p.userAgent)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return []string{"ref-model"}, nil
}

func (p *refProvider) kindFor(status int) error {
	switch {
	case status == http.StatusTooManyRequests && !p.flaws.misclassify429:
		return llmprovider.ErrRateLimited
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return llmprovider.ErrAuthFailure
	case status >= http.StatusInternalServerError || status == http.StatusTooManyRequests:
		return llmprovider.ErrProviderUnavailable
	default:
		return llmprovider.ErrInvalidRequest
	}
}

func refHarness(f flaws) Harness {
	return Harness{
		New: newRef(f),
		Text: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"text":"hello from the reference provider"}`)
		},
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
			_ = json.NewEncoder(w).Encode(map[string]any{"call": map[string]string{"name": tool, "arguments": "{}"}})
		},
		Error: func(w http.ResponseWriter, _ *http.Request, status int) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":"llmtest"}`)
		},
		Fidelity:    true,
		StrictTools: true,
		Garbled:     func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
		Truncated: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"call":{"name":"llmtest_tool","arguments":"{\"city\":"},"finish":"length"}`)
		},
		ReasoningCut: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"reasoning":"thinking about it","finish":"length"}`)
		},
		NotPermitted: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":"not_permitted"}`)
		},
	}
}

// recorder is a reporter that keeps every failure, prefixed with its
// subtest path.
type recorder struct {
	prefix string
	mu     *sync.Mutex
	msgs   *[]string
}

func newRecorder() *recorder { return &recorder{mu: &sync.Mutex{}, msgs: &[]string{}} }

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	*r.msgs = append(*r.msgs, r.prefix+fmt.Sprintf(format, args...))
}

func (r *recorder) Run(name string, f func(reporter)) {
	f(&recorder{prefix: r.prefix + name + ": ", mu: r.mu, msgs: r.msgs})
}

func (r *recorder) failures() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), *r.msgs...)
}

func TestRun_ConformantProviderPasses(t *testing.T) {
	Run(t, refHarness(flaws{}))
}

func TestRun_ConformantProviderReportsNothing(t *testing.T) {
	rec := newRecorder()
	runChecks(rec, refHarness(flaws{}))
	if got := rec.failures(); len(got) != 0 {
		t.Fatalf("a conformant provider reported %q", got)
	}
}

// TestRun_NamesTheBrokenRule checks each flaw fails the check for its rule,
// with a message naming the rule. The race flaw is caught by the race
// detector, not by a check, so it is proven in a scratch copy instead
// (0015-PLAN S6 step 3).
func TestRun_NamesTheBrokenRule(t *testing.T) {
	for _, c := range []struct {
		name  string
		flaws flaws
		want  string
	}{
		{"ignores cancellation", flaws{ignoreCancel: true}, "R40-cancellation: R40 (cancellation): Generate did not return within"},
		{"sends an unsupported need", flaws{skipCheck: true}, "R10-R11-capabilities: R11 (refusal before the network): a request needing continuation"},
		{"skips validation", flaws{skipCheck: true}, "R23-invalid-values: R23 (invalid values): a negative MaxOutputTokens"},
		{"misclassifies a 429", flaws{misclassify429: true}, "R25-R26-classification: R25 (errors by kind): HTTP 429"},
		{"never renews a refused token", flaws{noReauth: true}, "R16-reauth: R16 (a refused token is renewed once"},
		{"never says why it ended", flaws{noFinish: true}, "R7-R9-response: R7 (Response invariants, 0020-MADR F11): a text reply has no FinishReason"},
		{"sends an empty role", flaws{emptyRole: true}, `R6-empty-role: R6 (an empty Role is the user's, 0020-MADR F10): the request carries "role":""`},
		{"drops the instructions", flaws{dropInstructions: true}, `W12-fidelity: W12 (fidelity, 0021-MADR D5): the request's Instructions`},
		{"retries an undecodable reply", flaws{decodeBypass: true}, "W12-garbled: W12 (an undecodable reply, 0021-MADR D1): Generate returned"},
		{"takes a cut answer for a whole one", flaws{noLengthCheck: true}, "W12-truncated: W12 (a cut answer, 0021-MADR W3)"},
		{"drops a call's arguments", flaws{emptyArguments: true}, `W12-strict-tools: W12 (strict tools, 0021-MADR W1): the call to "llmtest_tool" has arguments ""`},
		{"takes a reasoning-only cut for a whole answer", flaws{keepReasoningCut: true}, "F7-reasoning-cut: 0026-MADR F7 (an answer cut while reasoning)"},
		{"reports an HTTP failure as a plain error", flaws{notAPIError: true}, "R25-R26-classification: R24 (*APIError is the only structured error): HTTP 400"},
		{"drops a 429's Retry-After", flaws{noRetryAfter: true}, "R25-R26-classification: R24 (RetryAfter): HTTP 429"},
		{"reports not permitted as a refused key", flaws{notPermittedAsAuth: true}, "R25-not-permitted: R25 (errors by kind): the service's not-permitted refusal"},
		{"ignores a token's own header", flaws{ignoresTokenHeader: true}, `R16-token-header: R16 (a token's own header): a "" token naming X-Llmtest-Key`},
		{"lists without its identity", flaws{listingNoIdentity: true}, "R40-R44-listing: R44 (identity): the listing sent User-Agent"},
		{"lists past its cancellation", flaws{listingIgnoresCancel: true}, "R40-R44-listing: R40 (cancellation): ListModels did not return"},
	} {
		t.Run(c.name, func(t *testing.T) {
			rec := newRecorder()
			runChecks(rec, refHarness(c.flaws))
			got := rec.failures()
			for _, msg := range got {
				if strings.HasPrefix(msg, c.want) {
					return
				}
			}
			t.Fatalf("failures %q; want one starting %q", got, c.want)
		})
	}
}

func TestRun_RefusesAnIncompleteHarness(t *testing.T) {
	rec := newRecorder()
	runChecks(rec, Harness{New: newRef(flaws{})})
	if got := rec.failures(); len(got) != 1 || !strings.Contains(got[0], "needs New, Text and Error") {
		t.Fatalf("failures %q; want the Harness refused", got)
	}
}

func TestRun_NeedsToolCallWhenToolsAreForced(t *testing.T) {
	h := refHarness(flaws{})
	h.ToolCall = nil
	rec := newRecorder()
	runChecks(rec, h)
	want := "R10-R11-capabilities: llmtest: the Harness needs ToolCall"
	for _, msg := range rec.failures() {
		if strings.HasPrefix(msg, want) {
			return
		}
	}
	t.Fatalf("failures %q; want one starting %q", rec.failures(), want)
}

func TestRun_ReportsAFailingNew(t *testing.T) {
	h := refHarness(flaws{})
	h.New = func(string, ...llmprovider.Option) (llmprovider.Provider, error) {
		return nil, llmprovider.ErrInvalidRequest
	}
	rec := newRecorder()
	runChecks(rec, h)
	if got := rec.failures(); len(got) == 0 || !strings.Contains(got[0], "the Harness's New failed") {
		t.Fatalf("failures %q; want New's failure reported", got)
	}
}

// TestRun_ChecksCapabilityConsistency declares capabilities the provider
// contradicts.
func TestRun_ChecksCapabilityConsistency(t *testing.T) {
	h := refHarness(flaws{})
	h.New = func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
		p, err := newRef(flaws{})(baseURL, opts...)
		return inconsistent{p}, err
	}
	rec := newRecorder()
	runChecks(rec, h)
	joined := strings.Join(rec.failures(), "\n")
	for _, want := range []string{"NativeStreaming is Supported, but implementing Streamer is false", "ForcedToolChoice is Supported while Tools is Unsupported"} {
		if !strings.Contains(joined, want) {
			t.Errorf("failures %q; want %q", joined, want)
		}
	}
}

type inconsistent struct{ llmprovider.Provider }

func (inconsistent) Capabilities() llmprovider.Capabilities {
	return llmprovider.Capabilities{ForcedToolChoice: llmprovider.Supported, NativeStreaming: llmprovider.Supported}
}

// TestRun_ChecksTheResponse feeds replies that break R7 and R9.
func TestRun_ChecksTheResponse(t *testing.T) {
	for _, c := range []struct {
		name string
		gen  func() (*llmprovider.Response, error)
		want string
	}{
		{"both", func() (*llmprovider.Response, error) {
			return &llmprovider.Response{}, llmprovider.ErrInvalidRequest
		}, "Generate returned both a response and the error"},
		{"neither", func() (*llmprovider.Response, error) { return nil, nil }, "Generate returned neither"},
		{"no text", func() (*llmprovider.Response, error) { return &llmprovider.Response{}, nil }, "a text reply has no output text"},
		{"bad call", func() (*llmprovider.Response, error) {
			return &llmprovider.Response{Output: []llmprovider.Item{
				llmprovider.MessageItem{Text: "x"},
				llmprovider.FunctionCallItem{Name: "t", Arguments: "{not json"},
			}}, nil
		}, "output 1 is a call"},
		{"negative usage", func() (*llmprovider.Response, error) {
			return &llmprovider.Response{Output: []llmprovider.Item{llmprovider.MessageItem{Text: "x"}}, Usage: llmprovider.Usage{InputTokens: -1}}, nil
		}, "negative usage"},
		{"cached beyond input", func() (*llmprovider.Response, error) {
			return &llmprovider.Response{Output: []llmprovider.Item{llmprovider.MessageItem{Text: "x"}},
				Usage: llmprovider.Usage{InputTokens: 5, CachedTokens: 6}}, nil
		}, "a part larger than its total"},
		{"reasoning beyond output", func() (*llmprovider.Response, error) {
			return &llmprovider.Response{Output: []llmprovider.Item{llmprovider.MessageItem{Text: "x"}},
				Usage: llmprovider.Usage{OutputTokens: 5, ReasoningTokens: 6}}, nil
		}, "a part larger than its total"},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := refHarness(flaws{})
			h.New = func(string, ...llmprovider.Option) (llmprovider.Provider, error) {
				return canned{c.gen}, nil
			}
			rec := newRecorder()
			runChecks(rec, h)
			for _, msg := range rec.failures() {
				if strings.HasPrefix(msg, "R7-R9-response: ") && strings.Contains(msg, c.want) {
					return
				}
			}
			t.Fatalf("failures %q; want a response failure containing %q", rec.failures(), c.want)
		})
	}
}

// canned is a provider with no network that returns what gen gives.
type canned struct {
	gen func() (*llmprovider.Response, error)
}

func (canned) ID() llmprovider.ProviderID { return refID }

func (canned) Capabilities() llmprovider.Capabilities { return llmprovider.Capabilities{} }

func (c canned) Generate(context.Context, *llmprovider.Request) (*llmprovider.Response, error) {
	return c.gen()
}

// TestRun_AuthFailureIsTheServicesRefusal (0026-MADR F6): with AuthFailure
// set, the R16 check refuses the key with the service's own reply. The
// reference provider renews only after a 401, so a 400 refusal fails it.
func TestRun_AuthFailureIsTheServicesRefusal(t *testing.T) {
	h := refHarness(flaws{})
	h.AuthFailure = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"API_KEY_INVALID"}`)
	}
	rec := newRecorder()
	runChecks(rec, h)
	for _, msg := range rec.failures() {
		if strings.HasPrefix(msg, "R16-reauth: R16 (a refused token is renewed once") {
			return
		}
	}
	t.Fatalf("failures %q; want R16 to fail on a 400 refusal the provider does not renew", rec.failures())
}

// TestRun_NoReauthSkipsTheCheck (0020-MADR F2): a Harness that says why the
// reauth check does not apply is not held to it.
func TestRun_NoReauthSkipsTheCheck(t *testing.T) {
	h := refHarness(flaws{noReauth: true})
	h.NoReauth = "the credential decides the mode"
	rec := newRecorder()
	runChecks(rec, h)
	for _, msg := range rec.failures() {
		if strings.HasPrefix(msg, "R16-reauth:") {
			t.Fatalf("the skipped check still reported %q", msg)
		}
	}
}

// TestRun_W12ChecksAreOptional (0021-MADR W12): a harness that sets none of
// the W12 checks runs none of them, so it passes as before, even for a
// provider that would fail them.
func TestRun_W12ChecksAreOptional(t *testing.T) {
	h := refHarness(flaws{dropInstructions: true, decodeBypass: true, noLengthCheck: true, emptyArguments: true,
		keepReasoningCut: true})
	h.Fidelity, h.StrictTools, h.Garbled, h.Truncated, h.ReasoningCut = false, false, nil, nil, nil
	rec := newRecorder()
	runChecks(rec, h)
	if got := rec.failures(); len(got) != 0 {
		t.Fatalf("a harness without the W12 checks reported %q", got)
	}
}

// TestRun_StrictToolsNeedsToolCall: StrictTools with no ToolCall is a
// harness error.
// TestRun_ForcedChoiceRefusalWithoutToolCall (0026-MADR F55): R11's refusal
// check needs no ToolCall handler, so a provider that declares forced choice
// Unsupported and sends it anyway fails R11 whether or not the Harness has
// one.
func TestRun_ForcedChoiceRefusalWithoutToolCall(t *testing.T) {
	for _, withToolCall := range []bool{true, false} {
		h := refHarness(flaws{declaresNoForce: true})
		h.StrictTools = false
		if !withToolCall {
			h.ToolCall = nil
		}
		rec := newRecorder()
		runChecks(rec, h)
		found := false
		for _, msg := range rec.failures() {
			found = found || strings.HasPrefix(msg, "R10-R11-capabilities: R11 (refusal before the network): a request needing a forced tool choice")
		}
		if !found {
			t.Errorf("ToolCall set %t: failures %q; want R11 for the forced tool choice", withToolCall, rec.failures())
		}
	}
}

func TestRun_StrictToolsNeedsToolCall(t *testing.T) {
	h := refHarness(flaws{})
	h.ToolCall = nil
	rec := newRecorder()
	runChecks(rec, h)
	for _, msg := range rec.failures() {
		if strings.Contains(msg, "StrictTools needs the Harness's ToolCall") {
			return
		}
	}
	t.Fatalf("failures %q; want StrictTools to need ToolCall", rec.failures())
}
