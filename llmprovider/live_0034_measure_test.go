//go:build live_gateways

package llmprovider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0034-PLAN Phase 1's measurements (M1 to M9). They run only with
// LLMPROVIDER_LIVE_0034 set. A recording transport rewrites each request as
// the step needs, and the test logs each reply's status, stop_reason, whether
// it holds a tool_use block, and an error's type and message; never text or a
// key. Billed: 22 requests of at most 256 output tokens.

func require0034(t *testing.T) {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_0034") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0034 to run 0034-PLAN Phase 1's measurements")
	}
}

// strictCityTool is cityTool's schema as strict mode compiles it:
// additionalProperties false on every object.
var strictCityTool = llmprovider.Tool{Name: "record_city", Description: "Records a city name.",
	Schema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"],"additionalProperties":false}`)}

// turnInstruction is 0034-MADR D2's mid-conversation system message text.
const turnInstruction = "Tool-use requirement for the current turn: the application requires a call to the record_city " +
	"tool in your response to the user's latest message. Begin your response with the record_city tool call. " +
	"Do not reply with text only."

// cityRequest is a request offering tool, with choice.
func cityRequest(tool llmprovider.Tool, choice llmprovider.ToolChoice) *llmprovider.Request {
	return &llmprovider.Request{Tools: []llmprovider.Tool{tool}, ToolChoice: choice,
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Record the city Paris."}}}
}

// Body rewrites for the steps.
var (
	appendTurnInstruction = func(body map[string]any) {
		msgs, _ := body["messages"].([]any)
		body["messages"] = append(msgs, map[string]any{"role": "system", "content": turnInstruction})
	}
	topLevelInstruction = func(body map[string]any) {
		if s, ok := body["system"].(string); ok && s != "" {
			body["system"] = s + "\n\n" + turnInstruction
			return
		}
		body["system"] = turnInstruction
	}
	strictTools = func(body map[string]any) { eachTool(body, func(tool map[string]any) { tool["strict"] = true }) }
	both        = func(fs ...func(map[string]any)) func(map[string]any) {
		return func(body map[string]any) {
			for _, f := range fs {
				f(body)
			}
		}
	}
)

// replyShape is a Messages reply's status, stop_reason, whether it holds a
// tool_use block, and an error's type and message.
func replyShape(status int, body []byte) string {
	var r struct {
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type  string          `json:"type"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &r)
	if status/100 != 2 {
		if r.Error.Type == "" && r.Error.Message == "" {
			// Not Anthropic's error shape: the body, bounded, as the service
			// sent it (M9 on OpenCode's gateway).
			return fmt.Sprintf("status %d, body %s", status, compact(body, 400))
		}
		return fmt.Sprintf("status %d, error %s: %q", status, r.Error.Type, r.Error.Message)
	}
	call := "no tool_use"
	for _, c := range r.Content {
		if c.Type == "tool_use" {
			var in map[string]any
			valid := json.Unmarshal(c.Input, &in) == nil
			_, hasCity := in["city"].(string)
			call = fmt.Sprintf("tool_use (input valid JSON %v, city a string %v, keys %d)", valid, hasCity, len(in))
		}
	}
	return fmt.Sprintf("status %d, stop_reason %s, %s", status, r.StopReason, call)
}

// measure0034 sends req to model through provider id with rewrite applied,
// and logs the reply's shape under label.
func measure0034(t *testing.T, label string, id llmprovider.ProviderID, key, model string, req *llmprovider.Request,
	rewrite func(map[string]any), forced bool, opts ...llmprovider.Option) {
	t.Helper()
	if rewrite == nil {
		rewrite = func(map[string]any) {}
	}
	rec := &flagRecorder{rewrite: rewrite}
	p, err := providers.New(id, append([]llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
		llmprovider.WithMaxTokens(256), llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata(),
		llmprovider.WithHTTPClient(&http.Client{Transport: rec})}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	var genErr error
	if forced {
		_, genErr = llmprovider.GenerateToolCall(ctx, p, req)
	} else {
		_, genErr = p.Generate(ctx, req)
	}
	status, body := rec.result()
	if status == 0 {
		t.Fatalf("%s %s: no reply: %v", label, model, genErr)
	}
	t.Logf("%s %s: %s", label, model, replyShape(status, body))
}

// TestLive_0034ForcedChoice is M1 to M7, on the Claude provider. Billed: 19
// requests.
func TestLive_0034ForcedChoice(t *testing.T) {
	require0034(t)
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude])
	claude := llmprovider.ProviderClaude
	refusing := []string{"claude-sonnet-5-5", "claude-opus-5-5", "claude-fable-5-1"}

	t.Run("M1", func(t *testing.T) {
		measure0034(t, "M1 forced", claude, key, "claude-fable-5-1", cityRequest(cityTool, ""), nil, true)
	})
	t.Run("M2", func(t *testing.T) {
		measure0034(t, "M2 forced", claude, key, "claude-sonnet-5-5", cityRequest(cityTool, ""), nil, true)
	})
	for _, model := range refusing {
		t.Run("M3/"+model, func(t *testing.T) {
			measure0034(t, "M3 auto alone", claude, key, model, cityRequest(cityTool, llmprovider.ToolChoiceAuto), nil, false)
		})
	}
	for _, model := range refusing {
		for i := 1; i <= 3; i++ {
			t.Run(fmt.Sprintf("M4/%s/%d", model, i), func(t *testing.T) {
				measure0034(t, fmt.Sprintf("M4 auto+turn instruction #%d", i), claude, key, model,
					cityRequest(cityTool, llmprovider.ToolChoiceAuto), appendTurnInstruction, false)
			})
		}
	}
	t.Run("M5", func(t *testing.T) {
		measure0034(t, "M5 auto+top-level system", claude, key, "claude-sonnet-5-5",
			cityRequest(cityTool, llmprovider.ToolChoiceAuto), topLevelInstruction, false)
	})
	for _, model := range []string{"claude-haiku-5-5", "claude-sonnet-5"} {
		t.Run("M6/"+model, func(t *testing.T) {
			measure0034(t, "M6 forced+strict", claude, key, model, cityRequest(strictCityTool, ""), strictTools, true)
		})
	}
	t.Run("M6/claude-sonnet-5-5", func(t *testing.T) {
		measure0034(t, "M6 degraded+strict", claude, key, "claude-sonnet-5-5",
			cityRequest(strictCityTool, llmprovider.ToolChoiceAuto), both(appendTurnInstruction, strictTools), false)
	})
	t.Run("M7", func(t *testing.T) {
		measure0034(t, "M7 forced+strict, schema without additionalProperties false", claude, key, "claude-haiku-5-5",
			cityRequest(cityTool, ""), strictTools, true)
	})
}

// TestLive_0034OpencodeMessages is M8 and M9, on OpenCode Go's messages
// route. Billed: 3 requests.
func TestLive_0034OpencodeMessages(t *testing.T) {
	require0034(t)
	key := llmprovider.LiveOpencodeKey(t)
	route := opencode.WithRoute(opencode.RouteMessages)
	for _, model := range []string{"qwen3.8-flash", "minimax-m3"} {
		t.Run("M8/"+model, func(t *testing.T) {
			measure0034(t, "M8 auto+strict", llmprovider.ProviderOpencodeGo, key, model,
				cityRequest(strictCityTool, llmprovider.ToolChoiceAuto), strictTools, false, route)
		})
	}
	t.Run("M9", func(t *testing.T) {
		measure0034(t, "M9 forced", llmprovider.ProviderOpencodeGo, key, "qwen3.8-flash", cityRequest(cityTool, ""), nil, true, route)
	})
}

// soShape is a Messages reply's status, stop_reason, its content blocks'
// types, and whether its text parses as {"city": string}; an error's type and
// message. Never the text itself.
func soShape(status int, body []byte) string {
	var r struct {
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &r)
	if status/100 != 2 {
		return fmt.Sprintf("status %d, error %s: %q", status, r.Error.Type, r.Error.Message)
	}
	var types []string
	parsed := "no text"
	for _, c := range r.Content {
		types = append(types, c.Type)
		if c.Type == "text" {
			var v map[string]any
			if err := json.Unmarshal([]byte(c.Text), &v); err == nil {
				_, isCity := v["city"].(string)
				parsed = fmt.Sprintf("JSON object, city a string %v, keys %d", isCity, len(v))
			} else {
				parsed = "text not JSON"
			}
		}
	}
	return fmt.Sprintf("status %d, stop_reason %s, blocks %v, %s", status, r.StopReason, types, parsed)
}

// structuredOutput sets output_config.format to schema, keeping any other
// output_config field (an effort).
func structuredOutput(schema string) func(map[string]any) {
	return func(body map[string]any) {
		cfg, _ := body["output_config"].(map[string]any)
		if cfg == nil {
			cfg = map[string]any{}
		}
		cfg["format"] = map[string]any{"type": "json_schema", "schema": json.RawMessage(schema)}
		body["output_config"] = cfg
	}
}

const (
	strictCitySchema = `{"type":"object","properties":{"city":{"type":"string"}},"required":["city"],"additionalProperties":false}`
	looseCitySchema  = `{"type":"object","properties":{"city":{"type":"string"},"country":{"type":"string"}},"required":["city"]}`
)

// measureSO sends req to a Claude model with rewrite, at most 512 output
// tokens, and logs soShape under label.
func measureSO(t *testing.T, label, key, model string, req *llmprovider.Request, rewrite func(map[string]any)) {
	t.Helper()
	rec := &flagRecorder{rewrite: rewrite}
	p, err := providers.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
		llmprovider.WithMaxTokens(512), llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata(),
		llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	_, genErr := p.Generate(ctx, req)
	status, body := rec.result()
	if status == 0 {
		t.Fatalf("%s %s: no reply: %v", label, model, genErr)
	}
	t.Logf("%s %s: %s", label, model, soShape(status, body))
}

// replayedCall is a conversation in which record_city was called, with the
// synthesised id 0034-MADR D8 would give it, and answered.
func replayedCall() []llmprovider.Item {
	return []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Record the city Paris."},
		llmprovider.FunctionCallItem{CallID: "toolu_so_1", Name: "record_city", Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: "toolu_so_1", Output: "recorded"},
	}
}

// TestLive_0034StructuredOutputs is 0034-PLAN Phase 4b (S1 to S5): a forced
// named tool sent as structured outputs on the models that refuse forcing.
// Billed: 7 requests of at most 512 output tokens.
func TestLive_0034StructuredOutputs(t *testing.T) {
	require0034(t)
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude])
	for _, model := range []string{"claude-sonnet-5-5", "claude-opus-5-5", "claude-fable-5-1"} {
		t.Run("S1/"+model, func(t *testing.T) {
			measureSO(t, "S1 output_config.format, tools with tool_choice none", key, model,
				cityRequest(strictCityTool, llmprovider.ToolChoiceNone), structuredOutput(strictCitySchema))
		})
	}
	t.Run("S2", func(t *testing.T) {
		measureSO(t, "S2 a schema without additionalProperties false", key, "claude-sonnet-5-5",
			cityRequest(cityTool, llmprovider.ToolChoiceNone), structuredOutput(looseCitySchema))
	})
	t.Run("S3", func(t *testing.T) {
		req := &llmprovider.Request{Tools: []llmprovider.Tool{strictCityTool},
			Input: append(replayedCall(), llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Did that work? Answer in one word."})}
		measureSO(t, "S3 a synthesised call replayed with its result", key, "claude-sonnet-5-5", req, func(map[string]any) {})
	})
	t.Run("S4", func(t *testing.T) {
		req := &llmprovider.Request{
			Input: append(replayedCall(), llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Now record the city London."})}
		measureSO(t, "S4 no tools, a tool_use in the history, output_config.format", key, "claude-sonnet-5-5", req,
			structuredOutput(strictCitySchema))
	})
	t.Run("S5", func(t *testing.T) {
		req := cityRequest(strictCityTool, llmprovider.ToolChoiceNone)
		req.Reasoning = &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
		measureSO(t, "S5 S1 with Reasoning (adaptive thinking, effort low)", key, "claude-opus-5-5", req,
			structuredOutput(strictCitySchema))
	})
}
