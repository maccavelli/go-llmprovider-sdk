package providers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0062-MADR D2 to D6: each provider and route learns a refusal of the effort
// it sent, as its wire was recorded refusing one (0062-PLAN Phase 1), and
// sends the request again. Ollama, recorded in Phase 5, has no matcher (the
// amendment "Phase 5's Ollama cases").

// Refusals of max, in each wire's recorded form.
const (
	refusedMessages = `{"type":"error","error":{"type":"invalid_request_error",` +
		`"message":"output_config.effort: Input should be 'low', 'medium', 'high' or 'xhigh'"}}`
	refusedGrok        = `{"code":"invalid-argument","error":"Invalid reasoning effort."}`
	refusedInteraction = `{"error":{"message":"The value 'max' is not supported for 'generation_config.thinking_level'. ` +
		`Supported values: 'minimal', 'low', 'medium', 'high'.","code":"invalid_request"}}`
	refusedHuggingFace = `{"error":{"code":"400","message":"Invalid request: [\"reasoning_effort (max): Input should be ` +
		`'minimal', 'low', 'medium', 'high' or 'none'\"]","type":"Bad Request"}}`
	refusedKilo = `{"error":{"message":"Invalid option: expected one of \"none\"|\"low\"|\"medium\"|\"high\"",` +
		`"type":"invalid_request_error","param":"reasoning.effort","code":"invalid_request_error"}}`
	refusedGoogle = `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"thinking refused"}}`
	refusedOllama = `{"error":{"message":"invalid reasoning value: \"max\" (must be \"minimal\", \"low\", ` +
		`\"medium\", \"high\", \"xhigh\", \"ultra\", \"max\", or \"none\")","type":"invalid_request_error",` +
		`"param":null,"code":null}}`
)

// refusedResponses is the Responses wire's unsupported_value for max on model.
func refusedResponses(model string) string {
	return `{"error":{"message":"Unsupported value: 'max' is not supported with the '` + model + `' model. ` +
		`Supported values are: 'none', 'low', 'medium', 'high', and 'xhigh'.","type":"invalid_request_error",` +
		`"param":"reasoning.effort","code":"unsupported_value"}}`
}

// learnServer refuses its nth request with the nth of its refusals, an empty
// one refusing nothing, and answers every other with wirecase's canned reply;
// it keeps each request's body in order.
type learnServer struct {
	*httptest.Server
	mu     sync.Mutex
	bodies []map[string]any
}

func newLearnServer(t *testing.T, refusals ...string) *learnServer {
	t.Helper()
	s := &learnServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		s.mu.Lock()
		s.bodies = append(s.bodies, body)
		n := len(s.bodies)
		s.mu.Unlock()
		if n <= len(refusals) && refusals[n-1] != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, refusals[n-1])
			return
		}
		reply := wirecase.Case{}.Reply(r)
		if reply.Status != 0 {
			w.WriteHeader(reply.Status)
		}
		_, _ = io.WriteString(w, reply.Body)
	}))
	t.Cleanup(s.Close)
	return s
}

// shapes is the thinking each request sent, in order: "adaptive", "budget",
// "level=<l>", or "" for none.
func (s *learnServer) shapes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.bodies))
	for i, b := range s.bodies {
		if th, ok := b["thinking"].(map[string]any); ok {
			out[i] = map[any]string{"adaptive": "adaptive", "enabled": "budget"}[th["type"]]
		}
		gen, _ := b["generationConfig"].(map[string]any)
		if tc, ok := gen["thinkingConfig"].(map[string]any); ok {
			switch {
			case tc["thinkingLevel"] != nil:
				out[i] = "level=" + tc["thinkingLevel"].(string)
			case tc["thinkingBudget"] != nil:
				out[i] = "budget"
			}
		}
	}
	return out
}

// efforts is the effort each request sent, in order.
func (s *learnServer) efforts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.bodies))
	for i, b := range s.bodies {
		out[i] = sentEffort(b)
	}
	return out
}

// TestLearn_Effort_0062: a request at max, refused in its wire's recorded
// form, is learned and sent again within the bound at the next effort, with
// a WarningReasoningLearned carrying the refusal and a WarningEffortDegraded;
// the next request sends the learned effort first; WithReasoningRefusals
// makes a fresh provider skip max from its first request; a Stream learns
// before its first event. On the google route, a refusal in another form than
// the one recorded (0062-REPORT-thinking-refusals.md E11) is returned after
// one send, nothing learned; and on Ollama, which has no matcher, any refusal
// is. On both, a refusal passed back with WithReasoningRefusals is never
// sent.
func TestLearn_Effort_0062(t *testing.T) {
	route := func(r opencode.Route) []llmprovider.Option { return []llmprovider.Option{opencode.WithRoute(r)} }
	for _, tc := range []struct {
		id      llmprovider.ProviderID
		opts    []llmprovider.Option
		model   string
		refusal string
		next    string // the effort sent once max is refused; "" when nothing is learned
	}{
		{llmprovider.ProviderClaude, nil, "claude-sonnet-5", refusedMessages, "xhigh"},
		{llmprovider.ProviderOpencodeZen, route(opencode.RouteMessages), "claude-sonnet-5", refusedMessages, "xhigh"},
		{llmprovider.ProviderOpenAI, nil, "gpt-6-luna", refusedResponses("gpt-6-luna"), "xhigh"},
		{llmprovider.ProviderGrok, nil, "grok-4.6", refusedGrok, "xhigh"},
		{llmprovider.ProviderOpencodeZen, route(opencode.RouteResponses), "gpt-5.5", refusedResponses("gpt-5.5"), "xhigh"},
		{llmprovider.ProviderGemini, nil, "gemini-3.8-flash", refusedInteraction, "high"},
		{llmprovider.ProviderOpencodeGo, route(opencode.RouteChatCompletions), "glm-5.3-flash", refusedHuggingFace, "high"},
		{llmprovider.ProviderHuggingFace, nil, "deepseek-ai/DeepSeek-V4.1-Flash", refusedHuggingFace, "high"},
		{llmprovider.ProviderTogether, nil, "openai/gpt-oss-120b", refusedHuggingFace, "high"},
		{llmprovider.ProviderKilo, nil, "openai/gpt-6-luna", refusedKilo, "high"},
		{llmprovider.ProviderOpencodeZen, route(opencode.RouteGoogle), "gemini-3.8-flash", refusedGoogle, ""},
		{llmprovider.ProviderOllama, nil, "gpt-oss:20b", refusedOllama, ""},
	} {
		name := string(tc.id) + "/" + tc.model
		build := func(srv *learnServer, more ...llmprovider.Option) llmprovider.Provider {
			opts := append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false),
				testCache(t)}, tc.opts...)
			p, err := New(tc.id, append(opts, more...)...)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			return p
		}
		ask := func() *llmprovider.Request {
			return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
				Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortMax}}
		}

		srv := newLearnServer(t, tc.refusal)
		p := build(srv)
		resp, err := p.Generate(context.Background(), ask())
		if tc.next == "" {
			if err == nil || len(srv.efforts()) != 1 {
				t.Errorf("%s: %d request(s), %v; want the refusal after one, nothing learned", name, len(srv.efforts()), err)
			}
			passed := newLearnServer(t)
			refused := llmprovider.ReasoningRefusal{Model: tc.model, Effort: llmprovider.EffortMax}
			if _, err := build(passed, llmprovider.WithReasoningRefusals(refused)).Generate(context.Background(),
				ask()); err != nil || !slices.Equal(passed.efforts(), []string{"xhigh"}) {
				t.Errorf("%s: with max passed back: %v after %v; want xhigh, the next rung, sent first", name, err,
					passed.efforts())
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := srv.efforts(); !slices.Equal(got, []string{"max", tc.next}) {
			t.Errorf("%s: efforts sent %v; want max, then %s", name, got, tc.next)
		}
		refused := llmprovider.ReasoningRefusal{Model: tc.model, Effort: llmprovider.EffortMax}
		if !slices.ContainsFunc(resp.Warnings, func(w llmprovider.Warning) bool {
			return w.Kind == llmprovider.WarningReasoningLearned && w.Refused == refused
		}) || !slices.ContainsFunc(resp.Warnings, func(w llmprovider.Warning) bool {
			return w.Kind == llmprovider.WarningEffortDegraded
		}) {
			t.Errorf("%s: warnings %v; want a WarningReasoningLearned of %+v and a WarningEffortDegraded", name, resp.Warnings,
				refused)
		}
		if _, err := p.Generate(context.Background(), ask()); err != nil {
			t.Errorf("%s: the second request: %v", name, err)
		} else if got := srv.efforts(); len(got) != 3 || got[2] != tc.next {
			t.Errorf("%s: the second request sent %v; want %s first", name, got[2:], tc.next)
		}

		passed := newLearnServer(t, "")
		if _, err := build(passed, llmprovider.WithReasoningRefusals(refused)).Generate(context.Background(), ask()); err != nil {
			t.Errorf("%s: with the refusal passed back: %v", name, err)
		} else if got := passed.efforts(); !slices.Equal(got, []string{"xhigh"}) {
			t.Errorf("%s: with the refusal passed back, sent %v; want xhigh, the next rung, first", name, got)
		}

		streamed := newLearnServer(t, tc.refusal)
		for _, err := range llmprovider.Stream(context.Background(), build(streamed), ask()) {
			if apiErr, ok := errors.AsType[*llmprovider.APIError](err); ok && apiErr.Status == http.StatusBadRequest {
				t.Errorf("%s: the stream yielded the refusal %v; want it learned before the first event", name, err)
			}
			if err != nil {
				break
			}
		}
		if got := streamed.efforts(); len(got) != 2 || got[0] != "max" || got[1] != tc.next {
			t.Errorf("%s: the stream sent %v; want max, then %s", name, got, tc.next)
		}
	}
}

// Refusals of a thinking shape, in Messages' recorded forms, and a level the
// google route refuses (0062-REPORT-thinking-refusals.md E1, E11).
const (
	refusedAdaptive = `{"type":"error","error":{"type":"invalid_request_error",` +
		`"message":"adaptive thinking is not supported on this model"}}`
	refusedBudget = `{"type":"error","error":{"type":"invalid_request_error","message":` +
		`"\"thinking.type.enabled\" is not supported for this model. Use \"thinking.type.adaptive\" and ` +
		`\"output_config.effort\" to control thinking behavior."}}`
	refusedLevel = `{"error":{"code":400,"status":"INVALID_ARGUMENT","message":"Upstream request failed: [400] ` +
		`Invalid value at 'generation_config.thinking_config.thinking_level' (type.googleapis.com/` +
		`google.cloud.aiplatform.v1.GenerationConfig.ThinkingConfig.ThinkingLevel), "}}`
)

// TestLearn_Shape_0062 (0062-MADR D3, D5, D6): a thinking shape the model
// refuses is learned and the other sent, within the request: adaptive refused
// then the budget accepted, and the reverse, on Claude and OpenCode's messages
// route; on the google route, a level refused steps to the next. The second
// request is right the first time, WithReasoningRefusals makes a fresh
// provider skip the refused shape, a model refusing both shapes has the second
// refusal returned, and a Stream learns before its first event.
func TestLearn_Shape_0062(t *testing.T) {
	route := func(r opencode.Route) []llmprovider.Option { return []llmprovider.Option{opencode.WithRoute(r)} }
	adaptive, budget := llmprovider.ThinkingAdaptive, llmprovider.ThinkingBudget
	for _, tc := range []struct {
		id        llmprovider.ProviderID
		opts      []llmprovider.Option
		model     string
		reasoning llmprovider.Reasoning
		refusal   string
		sent      []string // the first send's, and the resend's
		refused   llmprovider.ReasoningRefusal
	}{
		{llmprovider.ProviderClaude, nil, "claude-sonnet-4-5-20250929", llmprovider.Reasoning{Effort: llmprovider.EffortHigh},
			refusedAdaptive, []string{"adaptive", "budget"}, llmprovider.ReasoningRefusal{Shape: adaptive}},
		{llmprovider.ProviderClaude, nil, "claude-sonnet-5", llmprovider.Reasoning{Budget: 4096},
			refusedBudget, []string{"budget", "adaptive"}, llmprovider.ReasoningRefusal{Shape: budget}},
		{llmprovider.ProviderOpencodeZen, route(opencode.RouteMessages), "claude-sonnet-5", llmprovider.Reasoning{Budget: 4096},
			refusedBudget, []string{"budget", "adaptive"}, llmprovider.ReasoningRefusal{Shape: budget}},
		{llmprovider.ProviderOpencodeGo, route(opencode.RouteMessages), "qwen3.8-flash",
			llmprovider.Reasoning{Effort: llmprovider.EffortHigh}, refusedAdaptive, []string{"adaptive", "budget"},
			llmprovider.ReasoningRefusal{Shape: adaptive}},
		{llmprovider.ProviderOpencodeZen, route(opencode.RouteGoogle), "gemini-3.8-flash",
			llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}, refusedLevel, []string{"level=xhigh", "level=high"},
			llmprovider.ReasoningRefusal{Effort: llmprovider.EffortXHigh}},
	} {
		name := string(tc.id) + "/" + tc.model
		tc.refused.Model = tc.model
		build := func(srv *learnServer, more ...llmprovider.Option) llmprovider.Provider {
			opts := append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(tc.model),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false),
				testCache(t)}, tc.opts...)
			p, err := New(tc.id, append(opts, more...)...)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			return p
		}
		ask := func() *llmprovider.Request {
			r := tc.reasoning
			return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
				Reasoning: &r}
		}

		srv := newLearnServer(t, tc.refusal)
		p := build(srv)
		resp, err := p.Generate(context.Background(), ask())
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := srv.shapes(); !slices.Equal(got, tc.sent) {
			t.Errorf("%s: sent %v; want %v", name, got, tc.sent)
		}
		if !slices.ContainsFunc(resp.Warnings, func(w llmprovider.Warning) bool {
			return w.Kind == llmprovider.WarningReasoningLearned && w.Refused == tc.refused
		}) {
			t.Errorf("%s: warnings %v; want a WarningReasoningLearned of %+v", name, resp.Warnings, tc.refused)
		}
		if _, err := p.Generate(context.Background(), ask()); err != nil {
			t.Errorf("%s: the second request: %v", name, err)
		} else if got := srv.shapes(); len(got) != 3 || got[2] != tc.sent[1] {
			t.Errorf("%s: the second request sent %v; want %s first", name, got[2:], tc.sent[1])
		}

		passed := newLearnServer(t)
		if _, err := build(passed, llmprovider.WithReasoningRefusals(tc.refused)).Generate(context.Background(),
			ask()); err != nil {
			t.Errorf("%s: with the refusal passed back: %v", name, err)
		} else if got := passed.shapes(); !slices.Equal(got, tc.sent[1:]) {
			t.Errorf("%s: with the refusal passed back, sent %v; want %s first", name, got, tc.sent[1])
		}
	}

	// Both shapes refused: the second refusal is returned, after two sends.
	both := newLearnServer(t, refusedAdaptive, refusedBudget)
	p, err := New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-x"),
		llmprovider.WithBaseURL(both.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false), testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Generate(context.Background(), &llmprovider.Request{
		Input:     []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
		Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}})
	if apiErr, ok := errors.AsType[*llmprovider.APIError](err); !ok || apiErr.Status != http.StatusBadRequest ||
		!slices.Equal(both.shapes(), []string{"adaptive", "budget"}) {
		t.Errorf("both shapes refused: %v after %v; want the budget's refusal after adaptive, then budget", err, both.shapes())
	}

	// A stream learns before its first event.
	streamed := newLearnServer(t, refusedAdaptive)
	sp, err := New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-sonnet-4-5-20250929"),
		llmprovider.WithBaseURL(streamed.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false),
		testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range llmprovider.Stream(context.Background(), sp, &llmprovider.Request{
		Input:     []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
		Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortHigh}}) {
		if apiErr, ok := errors.AsType[*llmprovider.APIError](err); ok && apiErr.Status == http.StatusBadRequest {
			t.Errorf("the stream yielded the refusal %v; want it learned before the first event", err)
		}
		if err != nil {
			break
		}
	}
	if got := streamed.shapes(); !slices.Equal(got, []string{"adaptive", "budget"}) {
		t.Errorf("the stream sent %v; want adaptive, then budget", got)
	}
}

// TestProbeReasoning_Provider_0062 (0062-MADR D7): through Claude, the probe
// sends max, and returns what the model refused, the effort or the adaptive
// shape, after exactly one accepted send; a model that refuses nothing gives
// none after one send; a model refusing both shapes gives the error.
func TestProbeReasoning_Provider_0062(t *testing.T) {
	for _, tc := range []struct {
		model    string
		refusals []string
		want     []llmprovider.ReasoningRefusal
		efforts  []string
		shapes   []string
	}{
		{"claude-sonnet-5", []string{refusedMessages},
			[]llmprovider.ReasoningRefusal{{Model: "claude-sonnet-5", Effort: llmprovider.EffortMax}},
			[]string{"max", "xhigh"}, []string{"adaptive", "adaptive"}},
		{"claude-sonnet-4-5-20250929", []string{refusedAdaptive},
			[]llmprovider.ReasoningRefusal{{Model: "claude-sonnet-4-5-20250929", Shape: llmprovider.ThinkingAdaptive}},
			[]string{"max", ""}, []string{"adaptive", "budget"}},
		{"claude-opus-5-5", nil, nil, []string{"max"}, []string{"adaptive"}},
	} {
		srv := newLearnServer(t, tc.refusals...)
		p, err := New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-default"),
			llmprovider.WithBaseURL(srv.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false),
			testCache(t))
		if err != nil {
			t.Fatal(err)
		}
		got, err := llmprovider.ProbeReasoning(context.Background(), p, tc.model)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, %v; want %v", tc.model, got, err, tc.want)
		}
		if efforts, shapes := srv.efforts(), srv.shapes(); !slices.Equal(efforts, tc.efforts) ||
			!slices.Equal(shapes, tc.shapes) || len(efforts) != len(tc.refusals)+1 {
			t.Errorf("%s: sent efforts %v, shapes %v; want %v, %v, one accepted after %d refused", tc.model, efforts, shapes,
				tc.efforts, tc.shapes, len(tc.refusals))
		}
	}

	both := newLearnServer(t, refusedAdaptive, refusedBudget)
	p, err := New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("claude-x"),
		llmprovider.WithBaseURL(both.URL), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false), testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := llmprovider.ProbeReasoning(context.Background(), p, ""); got != nil ||
		!errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("both shapes refused: %v, %v; want no refusals and the budget's refusal", got, err)
	}
	both.mu.Lock()
	b := slices.Clone(both.bodies)
	both.mu.Unlock()
	if len(b) != 2 || b[0]["model"] != "claude-x" {
		t.Errorf("both shapes refused: %d send(s), the first to %v; want two, to the provider's own model", len(b), b[0]["model"])
	}
}
