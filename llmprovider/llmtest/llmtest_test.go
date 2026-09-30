package llmtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

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
}

// refProvider is a small conformant provider over a JSON wire: it POSTs
// {"text","tool"} and reads {"text"} or {"call":{"name","arguments"}}.
type refProvider struct {
	baseURL   string
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
		return &refProvider{baseURL: baseURL, client: st.HTTPClient(), userAgent: st.UserAgent(), flaws: f}, nil
	}
}

func (p *refProvider) ID() llmprovider.ProviderID { return refID }

func (p *refProvider) Capabilities() llmprovider.Capabilities { return refCaps }

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
	tool, _ := req.ToolChoice.Tool()
	body, err := json.Marshal(map[string]string{"text": req.Input[0].(llmprovider.MessageItem).Text, "tool": tool})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", p.userAgent)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &llmprovider.APIError{Provider: string(refID), Status: resp.StatusCode, Kind: p.kindFor(resp.StatusCode)}
	}
	var out struct {
		Text string `json:"text"`
		Call *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"call"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: %w", llmprovider.ErrProviderUnavailable, err)
	}
	result := &llmprovider.Response{FinishReason: llmprovider.FinishStop}
	if out.Call != nil {
		result.Output = append(result.Output, llmprovider.FunctionCallItem{CallID: "call_1", Name: out.Call.Name, Arguments: out.Call.Arguments})
	}
	if out.Text != "" {
		result.Output = append(result.Output, llmprovider.MessageItem{Role: string(llmprovider.RoleAssistant), Text: out.Text})
	}
	return result, nil
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
