package chatcompletions

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// Fixtures below are real bodies measured against the live gateways on
// 2026-08-28/29, not invented shapes.

// measured on OpenCode Zen, model big-pickle: reasoning arrives as
// message.reasoning_content.
const fixtureOpencodeReasoningContent = `{"id":"router-5a84a0cf","object":"chat.completion","created":1787968292,
"model":"big-pickle","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant",
"content":"Hi!","reasoning_content":"We need answer to user. They said \"say hi\".","tool_calls":null}}],
"usage":{"prompt_tokens":85,"completion_tokens":22,"total_tokens":107},"cost":"0"}`

// measured on Kilo Gateway, model kilo-auto/free: reasoning arrives as
// message.reasoning, plus a structured reasoning_details[] restatement.
const fixtureKiloReasoning = `{"id":"gen_01M16Q","object":"chat.completion","created":1788005835,
"model":"stepfun/step-3.7-flash","choices":[{"index":0,"message":{"role":"assistant","content":"ALPHA",
"reasoning":"Got it, the user said to say ALPHA only.",
"reasoning_details":[{"type":"reasoning.text","text":"Got it, the user said to say ALPHA only."}]}}]}`

func TestDecodeChatCompletions_Message(t *testing.T) {
	body := `{"id":"cmpl-1","choices":[{"message":{"role":"assistant","content":"hello world"}}]}`
	resp, err := Decode(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "cmpl-1" {
		t.Errorf("ID = %q, want cmpl-1", resp.ID)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Output))
	}
	if _, ok := resp.Output[0].(llmprovider.MessageItem); !ok {
		t.Errorf("output[0] = %T, want MessageItem", resp.Output[0])
	}
	if got := resp.OutputText(); got != "hello world" {
		t.Errorf("OutputText() = %q", got)
	}
}

// TestDecodeChatCompletions_ReasoningFieldNames is the core of the shared
// decoder: two gateways spell reasoning differently and both must decode.
func TestDecodeChatCompletions_ReasoningFieldNames(t *testing.T) {
	tests := []struct {
		name          string
		body          string
		wantReasoning string // "" means no ReasoningItem expected
		wantText      string
	}{
		{"reasoning_content only (OpenCode big-pickle, measured)",
			fixtureOpencodeReasoningContent,
			`We need answer to user. They said "say hi".`, "Hi!"},
		{"reasoning only (Kilo kilo-auto/free, measured)",
			fixtureKiloReasoning,
			"Got it, the user said to say ALPHA only.", "ALPHA"},
		{"both present: reasoning_content wins",
			`{"choices":[{"message":{"role":"assistant","content":"x","reasoning_content":"RC","reasoning":"R"}}]}`,
			"RC", "x"},
		{"reasoning_details only, no string field: not decoded",
			`{"choices":[{"message":{"role":"assistant","content":"x","reasoning_details":[{"type":"reasoning.text","text":"ignored"}]}}]}`,
			"", "x"},
		{"neither: absent reasoning is normal",
			`{"choices":[{"message":{"role":"assistant","content":"x"}}]}`,
			"", "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := Decode(strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			var reasoning []llmprovider.ReasoningItem
			for _, it := range resp.Output {
				if r, ok := it.(llmprovider.ReasoningItem); ok {
					reasoning = append(reasoning, r)
				}
			}
			if tc.wantReasoning == "" {
				if len(reasoning) != 0 {
					t.Errorf("expected no ReasoningItem, got %d: %q", len(reasoning), reasoning[0].Text)
				}
			} else {
				if len(reasoning) != 1 {
					t.Fatalf("expected exactly 1 ReasoningItem, got %d", len(reasoning))
				}
				if reasoning[0].Text != tc.wantReasoning {
					t.Errorf("reasoning = %q, want %q", reasoning[0].Text, tc.wantReasoning)
				}
				// Reasoning precedes the message, matching the Responses API order.
				if _, ok := resp.Output[0].(llmprovider.ReasoningItem); !ok {
					t.Errorf("output[0] = %T, want ReasoningItem first", resp.Output[0])
				}
			}
			if got := resp.OutputText(); got != tc.wantText {
				t.Errorf("OutputText() = %q, want %q", got, tc.wantText)
			}
		})
	}
}

// TestDecodeChatCompletions_ToolCalls uses the tool_calls shape measured on Kilo.
func TestDecodeChatCompletions_ToolCalls(t *testing.T) {
	body := `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[
	{"id":"chatcmpl-tool-8c37b719","type":"function","function":{"name":"get_weather","arguments":"{\"city\": \"Paris\"}"}}]}}]}`
	resp, err := Decode(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Output) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Output))
	}
	fc, ok := resp.Output[0].(llmprovider.FunctionCallItem)
	if !ok {
		t.Fatalf("output[0] = %T, want FunctionCallItem", resp.Output[0])
	}
	if fc.CallID != "chatcmpl-tool-8c37b719" || fc.Name != "get_weather" {
		t.Errorf("call = %+v", fc)
	}
	if fc.Arguments != `{"city": "Paris"}` {
		t.Errorf("arguments = %q", fc.Arguments)
	}
}

func TestDecodeChatCompletions_Empty(t *testing.T) {
	if _, err := Decode(strings.NewReader(`{"choices":[]}`)); err == nil {
		t.Error("expected error for no choices")
	}
	empty := `{"choices":[{"message":{"role":"assistant","content":"","tool_calls":null}}]}`
	if _, err := Decode(strings.NewReader(empty)); err == nil {
		t.Error("expected error when content, reasoning and tool_calls are all empty")
	}
	if _, err := Decode(strings.NewReader(`not json`)); err == nil {
		t.Error("expected error for malformed json")
	}
}

func TestItemsToChatMessages(t *testing.T) {
	msgs := itemsToChatMessages([]llmprovider.Item{
		llmprovider.MessageItem{Text: "no role"},
		llmprovider.MessageItem{Role: wire.RoleAssistant, Text: "assistant text"},
		llmprovider.FunctionCallOutputItem{CallID: "call_1", Output: `{"ok":true}`},
		llmprovider.FunctionCallItem{CallID: "c", Name: "n", Arguments: "{}"}, // its own assistant turn (MADR 0012 §2)
	})
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if calls, ok := msgs[3][keyToolCalls].([]map[string]any); !ok || len(calls) != 1 || msgs[3][wire.KeyRole] != wire.RoleAssistant {
		t.Errorf("function call message = %v, want an assistant turn with one tool call", msgs[3])
	}
	if msgs[0][wire.KeyRole] != wire.RoleUser {
		t.Errorf("empty role should default to user, got %v", msgs[0][wire.KeyRole])
	}
	if msgs[1][wire.KeyRole] != wire.RoleAssistant {
		t.Errorf("assistant role not preserved: %v", msgs[1][wire.KeyRole])
	}
	if msgs[2][wire.KeyRole] != roleTool || msgs[2]["tool_call_id"] != "call_1" {
		t.Errorf("tool result message = %v", msgs[2])
	}
}

func TestChatCompletionsBody(t *testing.T) {
	tool := &llmprovider.Tool{Name: "get_weather", Description: "Get weather", Schema: map[string]any{"type": "object"}}
	input := []llmprovider.Item{llmprovider.MessageItem{Role: wire.RoleUser, Text: "hi"}}

	t.Run("no tool, no reasoning", func(t *testing.T) {
		b := Body("big-pickle", 128, input, Opts{})
		if _, ok := b[keyTools]; ok {
			t.Error("tools must be absent")
		}
		if _, ok := b[keyToolChoice]; ok {
			t.Error("tool_choice must be absent")
		}
		if _, ok := b[keyReasoningEffort]; ok {
			t.Error("reasoning_effort must be absent")
		}
		if b[keyMaxTokens] != 128 || b[keyModel] != "big-pickle" {
			t.Errorf("body = %v", b)
		}
	})

	tools := []llmprovider.Tool{*tool}

	t.Run("tools with the auto choice omit tool_choice", func(t *testing.T) {
		b := Body("kilo-auto/free", 1, input, Opts{Tools: tools})
		list, ok := b[keyTools].([]map[string]any)
		if !ok || len(list) != 1 || list[0][wire.KeyType] != keyFunction {
			t.Fatalf("tools = %v", b[keyTools])
		}
		if fn, _ := list[0][keyFunction].(map[string]any); fn[wire.KeyName] != "get_weather" ||
			fn[keyDescription] != "Get weather" || fn[keyParameters] == nil {
			t.Errorf("tools[0].function = %v", list[0][keyFunction])
		}
		if _, ok := b[keyToolChoice]; ok {
			t.Error("tool_choice must be absent for the auto choice")
		}
	})

	t.Run("a forced tool sends both", func(t *testing.T) {
		b := Body("openai/gpt-oss-20b", 1, input, Opts{Tools: tools, ToolChoice: llmprovider.ForceTool("get_weather")})
		if _, ok := b[keyTools]; !ok {
			t.Error("tools must be present")
		}
		tc, ok := b[keyToolChoice].(map[string]any)
		if !ok {
			t.Fatalf("tool_choice = %v", b[keyToolChoice])
		}
		fn, ok := tc[keyFunction].(map[string]any)
		if !ok || fn[wire.KeyName] != "get_weather" {
			t.Errorf("tool_choice.function = %v", tc[keyFunction])
		}
	})

	t.Run("required and none are sent as strings", func(t *testing.T) {
		for _, choice := range []llmprovider.ToolChoice{llmprovider.ToolChoiceRequired, llmprovider.ToolChoiceNone} {
			b := Body("m", 1, input, Opts{Tools: tools, ToolChoice: choice})
			if b[keyToolChoice] != string(choice) {
				t.Errorf("%s: tool_choice = %v", choice, b[keyToolChoice])
			}
		}
	})

	// 0020-MADR F40: without tool_choice, "none" is kept by sending no tools.
	t.Run("without tool_choice, none sends no tools", func(t *testing.T) {
		b := Body("m", 1, input, Opts{Tools: tools, ToolChoice: llmprovider.ToolChoiceNone, NoToolChoice: true})
		if _, ok := b[keyTools]; ok {
			t.Error("tools must be absent")
		}
		b = Body("m", 1, input, Opts{Tools: tools, ToolChoice: llmprovider.ToolChoiceRequired, NoToolChoice: true})
		if _, ok := b[keyTools]; !ok {
			t.Error("tools must be offered, unforced")
		}
		if _, ok := b[keyToolChoice]; ok {
			t.Error("tool_choice must be absent")
		}
	})

	t.Run("reasoning effort passthrough", func(t *testing.T) {
		b := Body("deepseek-v4-pro", 1, input, Opts{ReasoningEffort: string(llmprovider.EffortXHigh)})
		if b[keyReasoningEffort] != string(llmprovider.EffortXHigh) {
			t.Errorf("reasoning_effort = %v, want %q", b[keyReasoningEffort], string(llmprovider.EffortXHigh))
		}
	})
}

// TestDecodeChat_LengthToolCallIsError: a tool call cut off by the token limit
// has unusable arguments, so it is an error: an *APIError of kind
// ErrIncomplete, which also matches ErrInvalidRequest (0015-MADR D7).
func TestDecodeChat_LengthToolCallIsError(t *testing.T) {
	body := `{"id":"c1","choices":[{"finish_reason":"length","message":{"role":"assistant",
		"tool_calls":[{"id":"t1","function":{"name":"commit","arguments":"{\"subject\":\"fix: tru"}}]}}]}`
	res, err := Decode(strings.NewReader(body))
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) || !errors.Is(apiErr.Kind, llmprovider.ErrIncomplete) || apiErr.Reason != "length" ||
		!errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("decode = %+v/%v, want an *APIError of kind ErrIncomplete, reason length", res, err)
	}
}

// TestDecodeChat_LengthTextKeepsText: a text answer cut by the token limit is
// still returned, and says so in FinishReason.
func TestDecodeChat_LengthTextKeepsText(t *testing.T) {
	body := `{"id":"c1","choices":[{"finish_reason":"length","message":{"role":"assistant","content":"partial ans"}}]}`
	res, err := Decode(strings.NewReader(body))
	if err != nil || res.OutputText() != "partial ans" || res.FinishReason != llmprovider.FinishLength {
		t.Fatalf("decode = %+v/%v, want the text and FinishReason length", res, err)
	}
}
