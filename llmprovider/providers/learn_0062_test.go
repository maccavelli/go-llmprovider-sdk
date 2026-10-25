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
// sends the request again; Ollama is recorded in Phase 5 (Q9 (c)).

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
)

// refusedResponses is the Responses wire's unsupported_value for max on model.
func refusedResponses(model string) string {
	return `{"error":{"message":"Unsupported value: 'max' is not supported with the '` + model + `' model. ` +
		`Supported values are: 'none', 'low', 'medium', 'high', and 'xhigh'.","type":"invalid_request_error",` +
		`"param":"reasoning.effort","code":"unsupported_value"}}`
}

// learnServer refuses its first request with refusal when refuse is set, and
// answers every other with wirecase's canned reply; it keeps each request's
// body in order.
type learnServer struct {
	*httptest.Server
	mu     sync.Mutex
	bodies []map[string]any
}

func newLearnServer(t *testing.T, refusal string) *learnServer {
	t.Helper()
	s := &learnServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		s.mu.Lock()
		s.bodies = append(s.bodies, body)
		first := len(s.bodies) == 1
		s.mu.Unlock()
		if first && refusal != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, refusal)
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
// before its first event. The google route has no matcher: its refusal is
// returned after one send.
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
