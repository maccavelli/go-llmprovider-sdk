//go:build live_gateways

package llmprovider_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0040-PLAN Phase 7's measurement: Phase 0's paths that stayed native, and
// Grok's dispatch, through the SDK. TestLive_0040Deferral runs only with
// LLMPROVIDER_LIVE_0040=1, and each path only with its key; the ChatGPT
// session's path also needs LLMPROVIDER_LIVE_CHATGPT=1, which only the owner
// sets, since it reads the Codex CLI's login (0040-MADR A3). Each path sends
// three requests with Request.Cache: the model calls the search, the search's
// result loads get_weather, and the model calls it; then the third answers.
// Each logs its status, stop reason, the tool called and the cached token
// count, and never a key, a header or a prompt. Billed: three requests per
// path, five keyed paths, about 3,000 input tokens each.

// deferralWant is what a path's three replies must show: probeWant's calls
// and cache reads, and whether each reply says its deferred tools were
// emulated (0040-MADR D7).
type deferralWant struct {
	probeWant
	emulated bool
}

// deferralRun is one path's replies: probeStep's fields, and whether each
// reply carried WarningDeferralEmulated.
type deferralRun struct {
	steps  []probeStep
	warned []bool
}

// problems lists what a run breaks: probeWant's problems, and a reply whose
// warning is not what the path's form gives.
func (w deferralWant) problems(run deferralRun) []string {
	out := w.probeWant.problems(run.steps)
	for i, warned := range run.warned {
		if warned != w.emulated {
			out = append(out, fmt.Sprintf("step %d: WarningDeferralEmulated %t, want %t", i+1, warned, w.emulated))
		}
	}
	return out
}

// TestDeferralChecks_0040 proves the measurement's checks on canned runs,
// with no key and no network.
func TestDeferralChecks_0040(t *testing.T) {
	native := deferralWant{probeWant{search: "tool_search", loaded: "get_weather", cacheRequired: true}, false}
	steps := []probeStep{{Status: 200, Tool: "tool_search"}, {Status: 200, Tool: "get_weather", Cached: 900},
		{Status: 200, Cached: 900}}
	quiet := []bool{false, false, false}
	if p := native.problems(deferralRun{steps, quiet}); len(p) != 0 {
		t.Fatalf("a good native run: %v", p)
	}
	dispatched := native
	dispatched.emulated = true
	for name, tc := range map[string]struct {
		want deferralWant
		run  deferralRun
		find string
	}{
		"a native path warned":     {native, deferralRun{steps, []bool{false, true, false}}, "step 2: WarningDeferralEmulated true, want false"},
		"a dispatched path silent": {dispatched, deferralRun{steps, quiet}, "step 1: WarningDeferralEmulated false, want true"},
		"no cache read":            {native, deferralRun{[]probeStep{steps[0], {Status: 200, Tool: "get_weather"}, steps[2]}, quiet}, "step 2 read nothing"},
		"the loaded tool not called": {native, deferralRun{[]probeStep{steps[0], {Status: 200, Tool: "read_file", Cached: 900}, steps[2]}, quiet},
			"want the loaded tool"},
		"a failed request": {native, deferralRun{[]probeStep{{Err: "refused"}}, []bool{false}}, "1 steps, want 3"},
	} {
		p := tc.want.problems(tc.run)
		if !strings.Contains(strings.Join(p, "\n"), tc.find) {
			t.Errorf("%s: problems %q, want one containing %q", name, p, tc.find)
		}
	}
}

// deferralTools0040 is Phase 0's tools: a direct tool, the search, and two
// deferred tools.
func deferralTools0040() []llmprovider.Tool {
	return []llmprovider.Tool{
		{Name: "read_file", Description: probeReadDesc, Schema: probeReadFile},
		{Name: "tool_search", Description: probeSearchDesc, Schema: probeSearchSchema, Search: true},
		{Name: "get_weather", Description: probeWeatherDesc, Schema: probeWeather, Deferred: true},
		{Name: "lookup_stock", Description: probeStockDesc, Schema: probeStock, Deferred: true},
	}
}

// runDeferral sends a path's three requests through p. The search's result
// loads get_weather; any other call is answered with probeResult. It stops
// at a failed request or an answer.
func runDeferral(t *testing.T, p llmprovider.Provider, path string) deferralRun {
	t.Helper()
	var run deferralRun
	input := []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: probeUser}}
	for range 3 {
		req := &llmprovider.Request{Instructions: prompt0039("0040 sdk " + path), Tools: deferralTools0040(),
			Input: input, Cache: llmprovider.CacheShort}
		ctx, cancel := llmprovider.LiveCtx(t)
		resp, err := p.Generate(ctx, req)
		cancel()
		if err != nil {
			llmprovider.SkipIfTransient(t, err)
			run.steps = append(run.steps, probeStep{Err: err.Error()})
			run.warned = append(run.warned, false)
			return run
		}
		step := probeStep{Status: 200, Stop: string(resp.FinishReason), Cached: resp.Usage.CachedTokens}
		var call *llmprovider.FunctionCallItem
		for _, item := range resp.Output {
			if c, ok := item.(llmprovider.FunctionCallItem); ok && call == nil {
				call, step.Tool, step.Args = &c, c.Name, c.Arguments
			}
		}
		warned := false
		for _, w := range resp.Warnings {
			warned = warned || w.Kind == llmprovider.WarningDeferralEmulated
		}
		run.steps, run.warned = append(run.steps, step), append(run.warned, warned)
		if call == nil {
			return run
		}
		input = append(input, resp.Output...)
		if call.Name == "tool_search" {
			input = append(input, llmprovider.FunctionCallOutputItem{CallID: call.CallID, Output: "Loaded get_weather."},
				llmprovider.ToolLoadItem{CallID: call.CallID, Name: "get_weather"})
			continue
		}
		input = append(input, llmprovider.FunctionCallOutputItem{CallID: call.CallID, Output: probeResult})
	}
	return run
}

// TestLive_0040Deferral is 0040-PLAN Phase 7's measurement.
func TestLive_0040Deferral(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0040") != "1" {
		t.Skip("set LLMPROVIDER_LIVE_0040=1 to run 0040-PLAN Phase 7's measurement (billed)")
	}
	cached := probeWant{search: "tool_search", loaded: "get_weather", cacheRequired: true}
	native, dispatched := deferralWant{cached, false}, deferralWant{cached, true}
	key := func(id llmprovider.ProviderID) func(t *testing.T) []llmprovider.Option {
		return func(t *testing.T) []llmprovider.Option {
			return []llmprovider.Option{llmprovider.WithAPIKey(llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id]))}
		}
	}
	zen := func(t *testing.T) []llmprovider.Option {
		return []llmprovider.Option{llmprovider.WithAPIKey(llmprovider.LiveOpencodeKey(t))}
	}
	paths := []struct {
		name  string
		id    llmprovider.ProviderID
		opts  func(t *testing.T) []llmprovider.Option
		model string
		want  deferralWant
	}{
		{"claude", llmprovider.ProviderClaude, key(llmprovider.ProviderClaude), "claude-haiku-5-5", native},
		// A1's form, not yet measured through Zen (0040-MADR A4).
		{"zen-messages", llmprovider.ProviderOpencodeZen, zen, "claude-haiku-5-5", native},
		{"openai", llmprovider.ProviderOpenAI, key(llmprovider.ProviderOpenAI), "gpt-6-luna", native},
		{"zen-responses", llmprovider.ProviderOpencodeZen, zen, "gpt-6-luna", native},
		{"grok", llmprovider.ProviderGrok, key(llmprovider.ProviderGrok), "grok-4.6", dispatched},
		// The owner's path (0040-MADR A3): it skips unless
		// LLMPROVIDER_LIVE_CHATGPT=1.
		{"chatgpt", llmprovider.ProviderOpenAI, func(t *testing.T) []llmprovider.Option {
			return []llmprovider.Option{llmprovider.WithTokenSource(liveChatGPTSession(t))}
		}, "gpt-5.4-mini", native},
	}
	for _, pt := range paths {
		t.Run(pt.name, func(t *testing.T) {
			p, err := providers.New(pt.id, append(pt.opts(t), llmprovider.WithModel(pt.model), llmprovider.WithModelProbes(false),
				llmprovider.WithMaxTokens(256), llmprovider.WithSessionID("live-0040-"+pt.name))...)
			if err != nil {
				t.Fatal(err)
			}
			run := runDeferral(t, p, pt.name)
			for i, s := range run.steps {
				t.Logf("%s step %d: stop=%s tool=%q args=%s cached=%d emulated=%t err=%q",
					pt.name, i+1, s.Stop, s.Tool, strings.TrimSpace(s.Args), s.Cached, run.warned[i], s.Err)
			}
			for _, problem := range pt.want.problems(run) {
				t.Errorf("%s: %s", pt.name, problem)
			}
		})
	}
}
