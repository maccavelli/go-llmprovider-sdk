//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0041-PLAN Phase 9's live check: one generation per thinking shape, each
// through the fact path (0041-MADR D2). TestLive_0041Facts runs only with
// LLMPROVIDER_LIVE_0041=1, and each path only with its key. Each path lists
// the provider's models with probes off, which warms the facts and sends no
// generation, and checks that catalog.CachedFacts then answers for its model.
// It sends one generation, keeps the request's body, and checks its thinking
// fields against the shape the facts give, and that the reply has text. It
// logs the shape, the stop reason and the token counts, never a key, a header,
// a prompt or a body. Billed: three requests, a few hundred tokens each.

// factsWant is what a path's request and reply must show: the thinking
// shape its body carries, and which fact the shape comes from.
type factsWant struct {
	shape string
	fact  catalog.Field
}

// factsRun is one path's outcome: whether the facts answered for the model,
// with the wanted fact; the shape the request body carried; the reply's text;
// and the error, if the request failed.
type factsRun struct {
	known bool
	shape string
	text  string
	err   string
}

// problems lists what a run breaks.
func (w factsWant) problems(run factsRun) []string {
	var out []string
	if run.err != "" {
		return []string{"the request failed: " + run.err}
	}
	if !run.known {
		out = append(out, "the facts did not answer with the shape's field: the request took the rule path")
	}
	if run.shape != w.shape {
		out = append(out, fmt.Sprintf("sent %q, want %q", run.shape, w.shape))
	}
	if strings.TrimSpace(run.text) == "" {
		out = append(out, "the reply has no text")
	}
	return out
}

// thinkingShape is the thinking a Messages or Interactions request body
// carries: "adaptive effort=<e>", "enabled budget=<n>", "level=<l>", or
// "none".
func thinkingShape(body map[string]any) string {
	if thinking, ok := body["thinking"].(map[string]any); ok {
		switch thinking["type"] {
		case "adaptive":
			effort, _ := body["output_config"].(map[string]any)
			return fmt.Sprintf("adaptive effort=%v", effort["effort"])
		case "enabled":
			return fmt.Sprintf("enabled budget=%v", thinking["budget_tokens"])
		}
		return fmt.Sprintf("thinking %v", thinking["type"])
	}
	if gen, ok := body["generation_config"].(map[string]any); ok {
		if level, ok := gen["thinking_level"]; ok {
			return fmt.Sprintf("level=%v", level)
		}
	}
	return "none"
}

// TestFactsChecks_0041 proves the live check's checks on canned runs and
// bodies, with no key and no network.
func TestFactsChecks_0041(t *testing.T) {
	for body, want := range map[string]string{
		`{"thinking":{"type":"adaptive"},"output_config":{"effort":"low"}}`: "adaptive effort=low",
		`{"thinking":{"type":"enabled","budget_tokens":1024}}`:              "enabled budget=1024",
		`{"thinking":{"type":"enabled","budget_tokens":512}}`:               "enabled budget=512",
		`{"generation_config":{"thinking_level":"low"}}`:                    "level=low",
		`{"generation_config":{"max_output_tokens":8}}`:                     "none",
		`{"max_tokens":8}`: "none",
	} {
		var m map[string]any
		if err := json.Unmarshal([]byte(body), &m); err != nil {
			t.Fatal(err)
		}
		if got := thinkingShape(m); got != want {
			t.Errorf("thinkingShape(%s) = %q; want %q", body, got, want)
		}
	}
	want := factsWant{"enabled budget=1024", catalog.FieldReasoningBudget}
	good := factsRun{known: true, shape: "enabled budget=1024", text: "ok"}
	if p := want.problems(good); len(p) != 0 {
		t.Fatalf("a good run: %v", p)
	}
	for name, tc := range map[string]struct {
		run  factsRun
		find string
	}{
		"the rule path":  {factsRun{shape: good.shape, text: "ok"}, "the request took the rule path"},
		"another shape":  {factsRun{known: true, shape: "enabled budget=512", text: "ok"}, `sent "enabled budget=512", want "enabled budget=1024"`},
		"no text":        {factsRun{known: true, shape: good.shape, text: " "}, "the reply has no text"},
		"a failed reply": {factsRun{err: "llmprovider: invalid request"}, "the request failed: llmprovider: invalid request"},
	} {
		p := want.problems(tc.run)
		if !strings.Contains(strings.Join(p, "\n"), tc.find) {
			t.Errorf("%s: problems %q, want one containing %q", name, p, tc.find)
		}
	}
}

// bodyRecorder keeps the body of each POST it sends.
type bodyRecorder struct {
	base http.RoundTripper
	mu   sync.Mutex
	last []byte
}

func (r *bodyRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && req.Body != nil {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		r.mu.Lock()
		r.last = body
		r.mu.Unlock()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	return r.base.RoundTrip(req)
}

// lastShape is the thinking shape of the last POST's body.
func (r *bodyRecorder) lastShape(t *testing.T) string {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var body map[string]any
	if err := json.Unmarshal(r.last, &body); err != nil {
		t.Fatalf("the request body: %v", err)
	}
	return thinkingShape(body)
}

// TestLive_0041Facts is 0041-PLAN Phase 9's live check.
func TestLive_0041Facts(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0041") != "1" {
		t.Skip("set LLMPROVIDER_LIVE_0041=1 to run 0041-PLAN Phase 9's live check (billed)")
	}
	paths := []struct {
		name      string
		id        llmprovider.ProviderID
		model     string
		reasoning llmprovider.Reasoning
		maxTokens int
		want      factsWant
	}{
		// Efforts with no budget option: adaptive thinking with the effort.
		{"claude-adaptive", llmprovider.ProviderClaude, "claude-haiku-5-5",
			llmprovider.Reasoning{Effort: llmprovider.EffortLow}, 1024,
			factsWant{"adaptive effort=low", catalog.FieldReasoningEfforts}},
		// A budget option with a 1024 minimum: the request's 512 is raised to
		// it, where the rule sends 512.
		{"claude-budget", llmprovider.ProviderClaude, "claude-sonnet-4-5-20250929",
			llmprovider.Reasoning{Budget: 512}, 2048,
			factsWant{"enabled budget=1024", catalog.FieldReasoningBudget}},
		// A listed level: thinking_level.
		{"gemini-level", llmprovider.ProviderGemini, "gemini-3.1-flash-lite",
			llmprovider.Reasoning{Effort: llmprovider.EffortLow}, 1024,
			factsWant{"level=low", catalog.FieldReasoningEfforts}},
	}
	for _, pt := range paths {
		t.Run(pt.name, func(t *testing.T) {
			key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[pt.id])
			rec := &bodyRecorder{base: http.DefaultTransport}
			p, err := providers.New(pt.id, llmprovider.WithAPIKey(key), llmprovider.WithModel(pt.model),
				llmprovider.WithModelProbes(false), llmprovider.WithMaxTokens(pt.maxTokens),
				llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
			if err != nil {
				t.Fatal(err)
			}
			lister, ok := p.(llmprovider.ModelLister)
			if !ok {
				t.Fatalf("%s lists no models", pt.id)
			}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			if _, err := lister.ListModels(ctx); err != nil {
				llmprovider.SkipIfTransient(t, err)
				t.Fatalf("ListModels: %v", err)
			}
			info, cached := catalog.CachedFacts(ctx, pt.id, pt.model, llmprovider.WithAPIKey(key))
			run := factsRun{known: cached && info.Has(pt.want.fact)}
			resp, err := p.Generate(ctx, &llmprovider.Request{
				Reasoning: &pt.reasoning,
				Input:     []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Reply with the one word: ok"}},
			})
			if err != nil {
				llmprovider.SkipIfTransient(t, err)
				run.err = err.Error()
			} else {
				run.shape = rec.lastShape(t)
				for _, item := range resp.Output {
					if m, ok := item.(llmprovider.MessageItem); ok {
						run.text += m.Text
					}
				}
				t.Logf("%s %s: %s, stop=%s, input=%d output=%d reasoning=%d", pt.name, pt.model, run.shape,
					resp.FinishReason, resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.ReasoningTokens)
			}
			for _, problem := range pt.want.problems(run) {
				t.Errorf("%s: %s", pt.name, problem)
			}
		})
	}
}
