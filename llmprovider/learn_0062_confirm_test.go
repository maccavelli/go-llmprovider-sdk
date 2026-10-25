package llmprovider_test

// 0062-PLAN Phase 5 step 3's confirmation of each removed rule: its cases and
// checks. Untagged, so that the offline twin below runs the checks the live
// test (live_0062_refusals_measure_test.go) runs, as live_twins_test.go runs
// live_checks_test.go's (0046-MADR D3).

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// learnConfirm is one removed rule's live case: the refusals its first
// request must learn, in order, by Phase 1's fixtures; none where the value
// the rule avoided is accepted.
type learnConfirm struct {
	id        string
	rule      string
	provider  llmprovider.ProviderID
	route     opencode.Route
	model     string
	reasoning llmprovider.Reasoning
	learns    []llmprovider.ReasoningRefusal
}

// learnConfirms are the cases of each rule 0062-MADR D8 removed, as the
// PLAN's Phase 5 record lists them. R4 has none: removed as dead, its one
// reachable model unavailable (0062-REPORT-thinking-refusals.md E7).
func learnConfirms() []learnConfirm {
	low, high, xhigh, maxE := llmprovider.EffortLow, llmprovider.EffortHigh, llmprovider.EffortXHigh, llmprovider.EffortMax
	effort := func(e llmprovider.Effort) llmprovider.Reasoning { return llmprovider.Reasoning{Effort: e} }
	refused := func(model string, efforts ...llmprovider.Effort) []llmprovider.ReasoningRefusal {
		out := make([]llmprovider.ReasoningRefusal, len(efforts))
		for i, e := range efforts {
			out[i] = llmprovider.ReasoningRefusal{Model: model, Effort: e}
		}
		return out
	}
	shape := func(model string, s llmprovider.ThinkingShape) []llmprovider.ReasoningRefusal {
		return []llmprovider.ReasoningRefusal{{Model: model, Shape: s}}
	}
	claude, gemini, grok := llmprovider.ProviderClaude, llmprovider.ProviderGemini, llmprovider.ProviderGrok
	zen, goGateway := llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo
	return []learnConfirm{
		{"claude-sonnet-5-budget", "R1", claude, "", "claude-sonnet-5", llmprovider.Reasoning{Budget: 4096},
			shape("claude-sonnet-5", llmprovider.ThinkingBudget)},
		{"claude-sonnet-4-5-high", "R1", claude, "", "claude-sonnet-4-5-20250929", effort(high),
			shape("claude-sonnet-4-5-20250929", llmprovider.ThinkingAdaptive)},
		{"opencode-go-messages-qwen3.8-flash-high", "R1", goGateway, opencode.RouteMessages, "qwen3.8-flash", effort(high), nil},
		{"opencode-go-messages-minimax-m3-high", "R2", goGateway, opencode.RouteMessages, "minimax-m3", effort(high), nil},
		{"gemini-2.5-flash-lite-low", "R5", gemini, "", "gemini-2.5-flash-lite", effort(low),
			refused("gemini-2.5-flash-lite", low, llmprovider.EffortMedium)},
		{"gemini-3.8-flash-xhigh", "Gemini's xhigh to high", gemini, "", "gemini-3.8-flash", effort(xhigh),
			refused("gemini-3.8-flash", xhigh)},
		{"gemini-3.8-flash-max", "0061-MADR D3's Gemini row", gemini, "", "gemini-3.8-flash", effort(maxE),
			refused("gemini-3.8-flash", maxE)},
		{"grok-4.5-max", "R6 and 0061-MADR D3's Grok row", grok, "", "grok-4.5", effort(maxE), refused("grok-4.5", maxE)},
		{"grok-build-0.1-high", "R6", grok, "", "grok-build-0.1", effort(high), refused("grok-build-0.1", high)},
		{"grok-4.20-0309-non-reasoning-high", "R6", grok, "", "grok-4.20-0309-non-reasoning", effort(high),
			refused("grok-4.20-0309-non-reasoning", high)},
		{"opencode-zen-google-gemini-3.8-flash-max", "0061-MADR D3's google-route row", zen, opencode.RouteGoogle,
			"gemini-3.8-flash", effort(maxE), nil},
		{"opencode-zen-google-gemini-3.8-flash-xhigh", "0013-MADR B9", zen, opencode.RouteGoogle, "gemini-3.8-flash",
			effort(xhigh), refused("gemini-3.8-flash", xhigh)},
		{"opencode-zen-chat-glm-5.3-max", "the chat-route gate", zen, opencode.RouteChatCompletions, "glm-5.3",
			effort(maxE), nil},
	}
}

// ollamaConfirm is Ollama's case, on the model the owner names: xhigh, which
// was sent as max, is accepted, nothing learned (0062-MADR, amendment "Phase
// 5's Ollama cases").
func ollamaConfirm(model string) learnConfirm {
	return learnConfirm{"ollama-xhigh", "Ollama's xhigh to max", llmprovider.ProviderOllama, "", model,
		llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}, nil}
}

// sendCounter counts the POSTs it sends on base.
type sendCounter struct {
	base http.RoundTripper
	n    atomic.Int64
}

func (s *sendCounter) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost {
		s.n.Add(1)
	}
	return s.base.RoundTrip(req)
}

// learnedRefusals is the Refused of each WarningReasoningLearned in warnings,
// in order.
func learnedRefusals(warnings []llmprovider.Warning) []llmprovider.ReasoningRefusal {
	var out []llmprovider.ReasoningRefusal
	for _, w := range warnings {
		if w.Kind == llmprovider.WarningReasoningLearned {
			out = append(out, w.Refused)
		}
	}
	return out
}

// confirmLearn sends c's request twice through one provider, built with key
// when there is one, with no metadata and a cache of its own, on base, and
// reports the first check that fails: the first request succeeds after one
// send more than c.learns, having learned exactly c.learns; the second
// succeeds after one send, having learned nothing.
func confirmLearn(ctx context.Context, c learnConfirm, key string, base http.RoundTripper,
	opts ...llmprovider.Option) error {
	counter := &sendCounter{base: base}
	all := []llmprovider.Option{llmprovider.WithModel(c.model), llmprovider.WithModelProbes(false),
		llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: counter}),
		catalog.WithCache(new(catalog.Cache))}
	if key != "" {
		all = append(all, llmprovider.WithAPIKey(key))
	}
	if c.route != "" {
		all = append(all, opencode.WithRoute(c.route))
	}
	p, err := providers.New(c.provider, append(all, opts...)...)
	if err != nil {
		return fmt.Errorf("%s: %w", c.id, err)
	}
	for i, want := range []struct {
		sends  int64
		learns []llmprovider.ReasoningRefusal
	}{{int64(len(c.learns)) + 1, c.learns}, {1, nil}} {
		counter.n.Store(0)
		reasoning := c.reasoning
		resp, err := p.Generate(ctx, &llmprovider.Request{
			Input:     []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: refusalPrompt}},
			Reasoning: &reasoning})
		if err != nil {
			return fmt.Errorf("%s: request %d: %w", c.id, i+1, err)
		}
		if got := counter.n.Load(); got != want.sends {
			return fmt.Errorf("%s: request %d: %d send(s); want %d", c.id, i+1, got, want.sends)
		}
		if learned := learnedRefusals(resp.Warnings); !slices.Equal(learned, want.learns) {
			return fmt.Errorf("%s: request %d: learned %v; want %v", c.id, i+1, learned, want.learns)
		}
	}
	return nil
}

// twinServer refuses its nth request with the nth of refusals, an empty one
// refusing nothing, and answers every other with wirecase's canned reply.
func twinServer(t *testing.T, refusals ...string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		mu.Lock()
		n++
		i := n
		mu.Unlock()
		if i <= len(refusals) && refusals[i-1] != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, refusals[i-1])
			return
		}
		reply := wirecase.Case{}.Reply(r)
		if reply.Status != 0 {
			w.WriteHeader(reply.Status)
		}
		_, _ = io.WriteString(w, reply.Body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestLearnConfirm_Twin_0062 (0062-PLAN Phase 5, step 3): confirmLearn, the
// live confirmation's checks, against a local server in each wire's recorded
// refusal: it passes where the first request learns what the case says and
// the second is right the first time, and fails where a refusal is not
// learned or the second request is refused again. And the case table is one
// case per removed rule's recorded case, R4's none.
func TestLearnConfirm_Twin_0062(t *testing.T) {
	budget := `{"type":"error","error":{"type":"invalid_request_error","message":"\"thinking.type.enabled\" is not ` +
		`supported for this model. Use \"thinking.type.adaptive\" and \"output_config.effort\" to control thinking behavior."}}`
	level := func(sent, supported string) string {
		return `{"error":{"message":"The value '` + sent + `' is not supported for 'generation_config.thinking_level'. ` +
			`Supported values: ` + supported + `.","code":"invalid_request"}}`
	}
	cases := map[string]learnConfirm{}
	for _, c := range learnConfirms() {
		if cases[c.id].id != "" {
			t.Errorf("case %s repeated", c.id)
		}
		cases[c.id] = c
	}
	if len(cases) != 13 {
		t.Errorf("%d cases; want 13", len(cases))
	}
	xhighRefused := level("xhigh", "'minimal', 'low', 'medium', 'high'")
	wrongLearn := cases["gemini-3.8-flash-xhigh"]
	wrongLearn.learns = []llmprovider.ReasoningRefusal{{Model: "gemini-3.8-flash", Effort: llmprovider.EffortMax}}
	for _, tc := range []struct {
		name     string
		c        learnConfirm
		refusals []string
		fails    string // the check that fails, "" for none
	}{
		{"the budget refused, then adaptive", cases["claude-sonnet-5-budget"], []string{budget}, ""},
		{"nothing refused where the budget should be", cases["claude-sonnet-5-budget"], nil, "request 1: 1 send(s); want 2"},
		{"xhigh refused, then high", cases["gemini-3.8-flash-xhigh"], []string{xhighRefused}, ""},
		{"another refusal learned than the case's", wrongLearn, []string{xhighRefused}, "request 1: learned"},
		{"the second request refused again", cases["gemini-3.8-flash-xhigh"],
			[]string{xhighRefused, "", level("high", "'low'")}, "request 2: 2 send(s); want 1"},
		{"nothing to learn, nothing refused", cases["opencode-zen-chat-glm-5.3-max"], nil, ""},
		{"nothing to learn, max refused", cases["opencode-zen-chat-glm-5.3-max"],
			[]string{`{"error":{"code":"400","message":"Invalid request: [\"reasoning_effort (max): Input should be ` +
				`'low', 'medium' or 'high'\"]","type":"Bad Request"}}`}, "request 1: 2 send(s); want 1"},
		{"the request fails", cases["opencode-zen-chat-glm-5.3-max"], []string{`{"error":{"message":"bad"}}`},
			"request 1: "},
		{"Ollama's xhigh accepted", ollamaConfirm("gpt-oss:20b"), nil, ""},
	} {
		srv := twinServer(t, tc.refusals...)
		err := confirmLearn(t.Context(), tc.c, "k", http.DefaultTransport, llmprovider.WithBaseURL(srv.URL))
		if got := fmt.Sprint(err); (err == nil) != (tc.fails == "") || !strings.Contains(got, tc.fails) {
			t.Errorf("%s: %v; want it to fail at %q", tc.name, err, tc.fails)
		}
	}
}
