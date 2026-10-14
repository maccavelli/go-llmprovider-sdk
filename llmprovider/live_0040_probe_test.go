//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// 0040-PLAN-native-tool-deferral.md Phase 0: the live probe of 0040-MADR's
// Confirmation, approved by the owner (Q5). TestLive_0040Probe runs only with
// LLMPROVIDER_LIVE_0040_PROBE=1, and each path only with its key. The SDK
// cannot send these fields yet, so each request is built here by hand. Each
// path sends three requests: the model calls the search, the search's result
// loads one deferred tool, and the model calls that tool. It logs each
// reply's status, stop reason, the tool called and the cached token counts,
// and never a key, a header or a prompt. A provider's error message is logged,
// as the classification of every failure needs it.
//
// The ChatGPT session's path is not here: its credentials come from the
// Codex CLI's auth file, which only the owner opts into reading
// (LLMPROVIDER_LIVE_CHATGPT; live_chatgpt_session_test.go).

// probeStep is what one reply did: its HTTP status, its stop reason, the
// tool it called (empty for none), the arguments of that call, and the input
// tokens it read from the cache.
type probeStep struct {
	Status int
	Stop   string
	Tool   string
	Args   string
	Cached int
	Err    string
}

// probeWant is what a path's three steps must show.
type probeWant struct {
	search, loaded string
	// cacheRequired says the second and third steps must read the cache.
	cacheRequired bool
}

// problems lists what three steps break: a failed request, a first call
// that is not the search, a second that is not the loaded tool, a third
// that still calls a tool, or a required cache read missing.
func (w probeWant) problems(steps []probeStep) []string {
	var out []string
	if len(steps) != 3 {
		return []string{fmt.Sprintf("%d steps, want 3", len(steps))}
	}
	for i, s := range steps {
		if s.Status != http.StatusOK {
			out = append(out, fmt.Sprintf("step %d: HTTP %d: %s", i+1, s.Status, s.Err))
		}
	}
	if len(out) > 0 {
		return out
	}
	if steps[0].Tool != w.search {
		out = append(out, fmt.Sprintf("step 1 called %q, want the search %q", steps[0].Tool, w.search))
	}
	if steps[1].Tool != w.loaded {
		out = append(out, fmt.Sprintf("step 2 called %q, want the loaded tool %q", steps[1].Tool, w.loaded))
	}
	if steps[2].Tool != "" {
		out = append(out, fmt.Sprintf("step 3 called %q, want an answer", steps[2].Tool))
	}
	if w.cacheRequired {
		for i := 1; i < 3; i++ {
			if steps[i].Cached == 0 {
				out = append(out, fmt.Sprintf("step %d read nothing from the cache", i+1))
			}
		}
	}
	return out
}

// TestProbeChecks_0040 proves the probe's checks on canned steps, with no key
// and no network.
func TestProbeChecks_0040(t *testing.T) {
	want := probeWant{search: "tool_search", loaded: "get_weather", cacheRequired: true}
	good := []probeStep{{Status: 200, Tool: "tool_search", Cached: 0}, {Status: 200, Tool: "get_weather", Cached: 900},
		{Status: 200, Cached: 900}}
	if p := want.problems(good); len(p) != 0 {
		t.Fatalf("a good run: %v", p)
	}
	for name, tc := range map[string]struct {
		steps []probeStep
		want  string
	}{
		"refused":       {[]probeStep{{Status: 400, Err: "bad"}, good[1], good[2]}, "step 1: HTTP 400: bad"},
		"no search":     {[]probeStep{{Status: 200, Tool: "get_weather"}, good[1], good[2]}, "want the search"},
		"not loaded":    {[]probeStep{good[0], {Status: 200, Tool: "read_file", Cached: 900}, good[2]}, "want the loaded tool"},
		"still calling": {[]probeStep{good[0], good[1], {Status: 200, Tool: "get_weather", Cached: 900}}, "want an answer"},
		"no cache":      {[]probeStep{good[0], {Status: 200, Tool: "get_weather"}, good[2]}, "step 2 read nothing"},
		"short":         {good[:2], "2 steps, want 3"},
	} {
		p := want.problems(tc.steps)
		if !strings.Contains(strings.Join(p, "\n"), tc.want) {
			t.Errorf("%s: problems %q, want one containing %q", name, p, tc.want)
		}
	}
}

const (
	probeUser   = "Use tool_search to find a tool for the weather, then use it to get the weather in Paris. Answer in one sentence."
	probeResult = "Sunny, 21 C."
)

var (
	probeReadFile = map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}},
		"required": []string{"path"}}
	probeSearchSchema = map[string]any{"type": "object", "properties": map[string]any{
		"query": map[string]any{"type": "string", "description": "What the tool should do"}}, "required": []string{"query"}}
	probeWeather = map[string]any{"type": "object", "properties": map[string]any{
		"location": map[string]any{"type": "string", "description": "A city"}}, "required": []string{"location"}}
	probeStock = map[string]any{"type": "object", "properties": map[string]any{
		"symbol": map[string]any{"type": "string"}}, "required": []string{"symbol"}}
)

const (
	probeSearchDesc  = "Searches the deferred tools by what they do, and loads the ones it finds."
	probeWeatherDesc = "Gets the current weather for a city."
	probeStockDesc   = "Gets the latest price of a stock."
	probeReadDesc    = "Reads a file from the workspace."
)

// post sends body as JSON to url with headers, and decodes the reply.
func post(t *testing.T, url string, headers map[string]string, body any) (int, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, map[string]any{"error": map[string]any{"message": err.Error()}}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		out = map[string]any{"error": map[string]any{"message": fmt.Sprintf("not JSON (%d bytes)", len(data))}}
	}
	return resp.StatusCode, out
}

// errText is a reply's error message, if any.
func errText(m map[string]any) string {
	if e, ok := m["error"].(map[string]any); ok {
		if s, ok := e["message"].(string); ok {
			return s
		}
	}
	if e, ok := m["error"].(string); ok {
		return e
	}
	return ""
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

// messagesPath probes Anthropic's Messages wire: deferred tools with
// defer_loading, a search tool the caller runs, and its result as a text
// block then a tool_reference (0040-MADR D2).
func messagesPath(t *testing.T, url string, headers map[string]string, model, name string) []probeStep {
	return messagesPathWith(t, url, headers, model, name, false)
}

// messagesPathWith is messagesPath; with textOutside, the search's result
// holds only the tool_reference, and its text follows the tool_result as its
// own block in the same user turn. The first run found Anthropic refusing a
// text block inside it: "Tool definitions/code execution functions cannot be
// mixed with other content" (0040-PLAN Phase 0).
func messagesPathWith(t *testing.T, url string, headers map[string]string, model, name string, textOutside bool) []probeStep {
	tools := []map[string]any{
		{"name": "read_file", "description": probeReadDesc, "input_schema": probeReadFile},
		{"name": "tool_search", "description": probeSearchDesc, "input_schema": probeSearchSchema},
		{"name": "get_weather", "description": probeWeatherDesc, "input_schema": probeWeather, "defer_loading": true},
		{"name": "lookup_stock", "description": probeStockDesc, "input_schema": probeStock, "defer_loading": true},
	}
	messages := []any{map[string]any{"role": "user", "content": probeUser}}
	var steps []probeStep
	for i := 0; i < 3; i++ {
		body := map[string]any{"model": model, "max_tokens": 1024, "system": prompt0039("0040 " + name),
			"tools": tools, "messages": messages, "cache_control": map[string]any{"type": "ephemeral"}}
		status, reply := post(t, url, headers, body)
		step := probeStep{Status: status, Err: errText(reply), Stop: fmt.Sprint(reply["stop_reason"])}
		if u, ok := reply["usage"].(map[string]any); ok {
			step.Cached = num(u["cache_read_input_tokens"])
		}
		var callID string
		content, _ := reply["content"].([]any)
		for _, c := range content {
			b, _ := c.(map[string]any)
			if b["type"] == "tool_use" {
				step.Tool, _ = b["name"].(string)
				callID, _ = b["id"].(string)
				args, _ := json.Marshal(b["input"])
				step.Args = string(args)
			}
		}
		steps = append(steps, step)
		if status != http.StatusOK || step.Tool == "" {
			break
		}
		messages = append(messages, map[string]any{"role": "assistant", "content": content})
		var result any = probeResult
		turn := []map[string]any{}
		if step.Tool == "tool_search" {
			ref := map[string]any{"type": "tool_reference", "tool_name": "get_weather"}
			text := map[string]any{"type": "text", "text": "Loaded get_weather."}
			result = []map[string]any{text, ref}
			if textOutside {
				result = []map[string]any{ref}
			}
		}
		turn = append(turn, map[string]any{"type": "tool_result", "tool_use_id": callID, "content": result})
		if textOutside && step.Tool == "tool_search" {
			turn = append(turn, map[string]any{"type": "text", "text": "Loaded get_weather."})
		}
		messages = append(messages, map[string]any{"role": "user", "content": turn})
	}
	return steps
}

// responsesPath probes OpenAI's Responses wire: a client tool_search, the
// deferred tool only in tool_search_output, and the whole history replayed
// with store false (0040-MADR D3).
func responsesPath(t *testing.T, url string, headers map[string]string, model, name string) []probeStep {
	tools := []map[string]any{
		{"type": "function", "name": "read_file", "description": probeReadDesc, "parameters": probeReadFile},
		{"type": "tool_search", "execution": "client", "description": probeSearchDesc, "parameters": probeSearchSchema},
	}
	input := []any{map[string]any{"role": "user", "content": probeUser}}
	var steps []probeStep
	for i := 0; i < 3; i++ {
		body := map[string]any{"model": model, "store": false, "instructions": prompt0039("0040 " + name),
			"tools": tools, "input": input, "include": []string{"reasoning.encrypted_content"},
			"prompt_cache_key": "probe-0040-" + name}
		status, reply := post(t, url, headers, body)
		step := probeStep{Status: status, Err: errText(reply), Stop: fmt.Sprint(reply["status"])}
		if u, ok := reply["usage"].(map[string]any); ok {
			if d, ok := u["input_tokens_details"].(map[string]any); ok {
				step.Cached = num(d["cached_tokens"])
			}
		}
		var call map[string]any
		output, _ := reply["output"].([]any)
		for _, o := range output {
			item, _ := o.(map[string]any)
			switch item["type"] {
			case "tool_search_call":
				step.Tool, call = "tool_search", item
				args, _ := json.Marshal(item["arguments"])
				step.Args = string(args)
			case "function_call":
				step.Tool, call = fmt.Sprint(item["name"]), item
				step.Args = fmt.Sprint(item["arguments"])
			}
		}
		steps = append(steps, step)
		if status != http.StatusOK || call == nil {
			break
		}
		input = append(input, output...)
		if step.Tool == "tool_search" {
			input = append(input, map[string]any{"type": "tool_search_output", "execution": "client",
				"call_id": call["call_id"], "status": "completed", "tools": []map[string]any{
					{"type": "function", "name": "get_weather", "description": probeWeatherDesc,
						"parameters": probeWeather, "defer_loading": true}}})
		} else {
			input = append(input, map[string]any{"type": "function_call_output", "call_id": call["call_id"],
				"output": probeResult})
		}
	}
	return steps
}

// kiloPath probes OpenRouter's deferred_tools protocol through Kilo's
// gateway, with a custom search (0040-MADR D4).
func kiloPath(t *testing.T, url string, headers map[string]string, model string) []probeStep {
	fn := func(name, desc string, params map[string]any, deferred bool) map[string]any {
		f := map[string]any{"type": "function", "function": map[string]any{"name": name, "description": desc,
			"parameters": params}}
		if deferred {
			f["defer_loading"] = true
		}
		return f
	}
	tools := []map[string]any{fn("read_file", probeReadDesc, probeReadFile, false),
		fn("tool_search", probeSearchDesc, probeSearchSchema, false),
		fn("get_weather", probeWeatherDesc, probeWeather, true), fn("lookup_stock", probeStockDesc, probeStock, true)}
	messages := []any{map[string]any{"role": "system", "content": prompt0039("0040 kilo")},
		map[string]any{"role": "user", "content": probeUser}}
	var steps []probeStep
	for i := 0; i < 3; i++ {
		body := map[string]any{"model": model, "max_tokens": 1024, "messages": messages, "tools": tools,
			"deferred_tools": map[string]any{"protocol": "v1", "profile": "portable", "validation": "runtime",
				"search": map[string]any{"type": "custom", "tool_id": "tool_search", "result_key": "tools"}}}
		status, reply := post(t, url, headers, body)
		step := probeStep{Status: status, Err: errText(reply)}
		if u, ok := reply["usage"].(map[string]any); ok {
			if d, ok := u["prompt_tokens_details"].(map[string]any); ok {
				step.Cached = num(d["cached_tokens"])
			}
		}
		choices, _ := reply["choices"].([]any)
		var msg map[string]any
		var callID string
		if len(choices) > 0 {
			c, _ := choices[0].(map[string]any)
			step.Stop = fmt.Sprint(c["finish_reason"])
			msg, _ = c["message"].(map[string]any)
			calls, _ := msg["tool_calls"].([]any)
			if len(calls) > 0 {
				tc, _ := calls[0].(map[string]any)
				callID, _ = tc["id"].(string)
				f, _ := tc["function"].(map[string]any)
				step.Tool, _ = f["name"].(string)
				step.Args = fmt.Sprint(f["arguments"])
			}
		}
		steps = append(steps, step)
		if status != http.StatusOK || step.Tool == "" {
			break
		}
		messages = append(messages, msg)
		content := probeResult
		if step.Tool == "tool_search" {
			content = `{"tools":["get_weather"]}`
		}
		messages = append(messages, map[string]any{"role": "tool", "tool_call_id": callID, "content": content})
	}
	return steps
}

// grokPath probes grok-build's dispatch on xAI's Responses wire: a fixed
// use_tool wrapper, and the loaded schema sent as text (0040-MADR D5).
func grokPath(t *testing.T, url string, headers map[string]string, model string) []probeStep {
	useTool := map[string]any{"type": "object", "properties": map[string]any{
		"tool_name":  map[string]any{"type": "string", "description": "The name of a tool tool_search found"},
		"tool_input": map[string]any{"type": "object", "description": "Its arguments, by the input schema tool_search gave"}},
		"required": []string{"tool_name", "tool_input"}}
	tools := []map[string]any{
		{"type": "function", "name": "read_file", "description": probeReadDesc, "parameters": probeReadFile},
		{"type": "function", "name": "tool_search", "description": probeSearchDesc, "parameters": probeSearchSchema},
		{"type": "function", "name": "use_tool", "description": "Runs a tool that tool_search found.", "parameters": useTool},
	}
	schema, _ := json.Marshal(probeWeather)
	found := fmt.Sprintf(`Loaded get_weather. [{"name":"get_weather","description":%q,"input_schema":%s}] Call it through use_tool.`,
		probeWeatherDesc, schema)
	input := []any{map[string]any{"role": "user", "content": probeUser}}
	var steps []probeStep
	for i := 0; i < 3; i++ {
		body := map[string]any{"model": model, "store": false, "instructions": prompt0039("0040 grok"),
			"tools": tools, "input": input, "include": []string{"reasoning.encrypted_content"}}
		status, reply := post(t, url, headers, body)
		step := probeStep{Status: status, Err: errText(reply), Stop: fmt.Sprint(reply["status"])}
		if u, ok := reply["usage"].(map[string]any); ok {
			if d, ok := u["input_tokens_details"].(map[string]any); ok {
				step.Cached = num(d["cached_tokens"])
			}
		}
		var call map[string]any
		output, _ := reply["output"].([]any)
		for _, o := range output {
			item, _ := o.(map[string]any)
			if item["type"] == "function_call" {
				call = item
				step.Tool = fmt.Sprint(item["name"])
				step.Args = fmt.Sprint(item["arguments"])
			}
		}
		// A use_tool call names the tool it runs: that is the call the
		// caller would see (0040-MADR D5).
		if step.Tool == "use_tool" {
			var a struct {
				ToolName string `json:"tool_name"`
			}
			if json.Unmarshal([]byte(step.Args), &a) == nil {
				step.Tool = a.ToolName
			}
		}
		steps = append(steps, step)
		if status != http.StatusOK || call == nil {
			break
		}
		input = append(input, output...)
		out := probeResult
		if call["name"] == "tool_search" {
			out = found
		}
		input = append(input, map[string]any{"type": "function_call_output", "call_id": call["call_id"], "output": out})
	}
	return steps
}

// TestLive_0040Probe runs each path whose key is set, and logs and checks it.
func TestLive_0040Probe(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0040_PROBE") != "1" {
		t.Skip("set LLMPROVIDER_LIVE_0040_PROBE=1 to run 0040-PLAN Phase 0's probe (billed)")
	}
	want := probeWant{search: "tool_search", loaded: "get_weather"}
	cached := want
	cached.cacheRequired = true
	paths := []struct {
		name string
		env  string
		want probeWant
		run  func(t *testing.T, key string) []probeStep
	}{
		{"claude", "ANTHROPIC_API_KEY", cached, func(t *testing.T, key string) []probeStep {
			return messagesPath(t, "https://api.anthropic.com/v1/messages",
				map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"}, "claude-haiku-5-5", "claude")
		}},
		{"zen-messages", "OPENCODE_API_KEY", cached, func(t *testing.T, key string) []probeStep {
			return messagesPath(t, "https://opencode.ai/zen/v1/messages",
				map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01", "x-opencode-session": "probe-0040-zen-messages"},
				"claude-haiku-5-5", "zen-messages")
		}},
		{"openai", "OPENAI_API_KEY", cached, func(t *testing.T, key string) []probeStep {
			return responsesPath(t, "https://api.openai.com/v1/responses",
				map[string]string{"Authorization": "Bearer " + key}, "gpt-6-luna", "openai")
		}},
		{"zen-responses", "OPENCODE_API_KEY", cached, func(t *testing.T, key string) []probeStep {
			return responsesPath(t, "https://opencode.ai/zen/v1/responses",
				map[string]string{"Authorization": "Bearer " + key, "x-opencode-session": "probe-0040-zen-responses"},
				"gpt-6-luna", "zen-responses")
		}},
		{"kilo", "KILO_API_KEY", want, func(t *testing.T, key string) []probeStep {
			return kiloPath(t, "https://api.kilo.ai/api/gateway/chat/completions",
				map[string]string{"Authorization": "Bearer " + key}, "anthropic/claude-haiku-5.5")
		}},
		{"grok", "XAI_API_KEY", want, func(t *testing.T, key string) []probeStep {
			return grokPath(t, "https://api.x.ai/v1/responses", map[string]string{"Authorization": "Bearer " + key}, "grok-4.6")
		}},
	}
	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			key := os.Getenv(p.env)
			if key == "" {
				t.Skipf("%s is not set", p.env)
			}
			steps := p.run(t, key)
			for i, s := range steps {
				t.Logf("%s step %d: status=%d stop=%s tool=%q args=%s cached=%d err=%q",
					p.name, i+1, s.Status, s.Stop, s.Tool, strings.TrimSpace(s.Args), s.Cached, s.Err)
			}
			for _, problem := range p.want.problems(steps) {
				t.Errorf("%s: %s", p.name, problem)
			}
		})
	}
}

// kiloVisibleTools asks a Kilo model, under deferred_tools, to name every tool
// it can call, without calling one. A deferred tool among them means Kilo did
// not honour the protocol; the first run could not tell, because its prompt
// sends the model to the search first (0040-PLAN Phase 0).
func kiloVisibleTools(t *testing.T, url string, headers map[string]string, model string) (int, string, string) {
	fn := func(name, desc string, params map[string]any, deferred bool) map[string]any {
		f := map[string]any{"type": "function", "function": map[string]any{"name": name, "description": desc,
			"parameters": params}}
		if deferred {
			f["defer_loading"] = true
		}
		return f
	}
	body := map[string]any{"model": model, "max_tokens": 300,
		"messages": []any{map[string]any{"role": "user", "content": "Do not call any tool. Reply with only the exact names of every tool you can call right now, comma-separated."}},
		"tools": []map[string]any{fn("read_file", probeReadDesc, probeReadFile, false),
			fn("tool_search", probeSearchDesc, probeSearchSchema, false),
			fn("get_weather", probeWeatherDesc, probeWeather, true), fn("lookup_stock", probeStockDesc, probeStock, true)},
		"deferred_tools": map[string]any{"protocol": "v1", "profile": "portable", "validation": "runtime",
			"search": map[string]any{"type": "custom", "tool_id": "tool_search", "result_key": "tools"}}}
	status, reply := post(t, url, headers, body)
	text := ""
	if choices, _ := reply["choices"].([]any); len(choices) > 0 {
		c, _ := choices[0].(map[string]any)
		msg, _ := c["message"].(map[string]any)
		text = fmt.Sprint(msg["content"])
	}
	return status, text, errText(reply)
}

// TestLive_0040ProbeFollowUp runs the two follow-ups the first run called for,
// within the owner's approval of 21 requests: Claude with the text outside the
// tool_result (three requests), and the Kilo discriminator (one).
func TestLive_0040ProbeFollowUp(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0040_PROBE") != "1" || os.Getenv("LLMPROVIDER_LIVE_0040_FOLLOWUP") != "1" {
		t.Skip("set LLMPROVIDER_LIVE_0040_PROBE=1 and LLMPROVIDER_LIVE_0040_FOLLOWUP=1 (billed)")
	}
	t.Run("claude-text-outside", func(t *testing.T) {
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			t.Skip("ANTHROPIC_API_KEY is not set")
		}
		steps := messagesPathWith(t, "https://api.anthropic.com/v1/messages",
			map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"}, "claude-haiku-5-5", "claude-text-outside", true)
		for i, s := range steps {
			t.Logf("claude-text-outside step %d: status=%d stop=%s tool=%q args=%s cached=%d err=%q",
				i+1, s.Status, s.Stop, s.Tool, strings.TrimSpace(s.Args), s.Cached, s.Err)
		}
		for _, problem := range (probeWant{search: "tool_search", loaded: "get_weather", cacheRequired: true}).problems(steps) {
			t.Errorf("claude-text-outside: %s", problem)
		}
	})
	t.Run("kilo-visible-tools", func(t *testing.T) {
		key := os.Getenv("KILO_API_KEY")
		if key == "" {
			t.Skip("KILO_API_KEY is not set")
		}
		status, text, errMsg := kiloVisibleTools(t, "https://api.kilo.ai/api/gateway/chat/completions",
			map[string]string{"Authorization": "Bearer " + key}, "anthropic/claude-haiku-5.5")
		t.Logf("kilo-visible-tools: status=%d tools=%q err=%q", status, strings.TrimSpace(text), errMsg)
		if status != http.StatusOK {
			t.Errorf("kilo-visible-tools: HTTP %d: %s", status, errMsg)
		}
		if strings.Contains(text, "get_weather") || strings.Contains(text, "lookup_stock") {
			t.Errorf("kilo-visible-tools: a deferred tool is visible before any search: %q", text)
		}
	})
}
