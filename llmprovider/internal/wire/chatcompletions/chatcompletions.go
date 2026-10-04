// Package chatcompletions is the OpenAI Chat Completions wire, shared by
// Hugging Face, Kilo, Together, Ollama and OpenCode's chat route (0015-MADR
// D2).
package chatcompletions

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// Chat Completions field names only this format uses.
const (
	keyModel           = "model"
	keyMessages        = "messages"
	keyMaxTokens       = "max_tokens"
	keyTools           = "tools"
	keyToolChoice      = "tool_choice"
	keyToolCalls       = "tool_calls"
	keyFunction        = "function"
	keyDescription     = "description"
	keyParameters      = "parameters"
	keyReasoningEffort = "reasoning_effort"
	keyReasoning       = "reasoning"
	roleTool           = "tool"
)

// Opts carries the per-gateway variations of a Chat Completions request. Zero
// values omit the corresponding field entirely.
type Opts struct {
	// Tools are offered to the model as functions.
	Tools []llmprovider.Tool
	// ToolChoice is sent as tool_choice: a named tool as an object,
	// "required" and "none" as strings, and auto as nothing.
	ToolChoice llmprovider.ToolChoice
	// NoToolChoice is for a service or model that does not take tool_choice:
	// none is sent, and ToolChoiceNone is kept by sending no tools, so the
	// model cannot call one (0020-MADR F40). Ollama never takes it; Kilo
	// gates it on the model's supported_parameters.
	NoToolChoice bool
	// ReasoningEffort, when non-empty, is sent as reasoning_effort. OpenCode's
	// chat route sets it only when the model's published reasoning_options
	// list the configured effort (MADR 0009 §6).
	ReasoningEffort string
	// Reasoning, when non-nil, is sent as the OpenRouter-style reasoning
	// object Kilo reads: {"effort": …} or {"enabled": true}.
	Reasoning map[string]any
	// ReplayReasoningField, when non-empty, replays prior reasoning on every
	// assistant message under this field (OpenCode interleaved models).
	ReplayReasoningField string
}

// itemsToChatMessages converts canonical items to OpenAI Chat Completions
// messages. A function call becomes the assistant turn's tool_calls entry,
// and its result a role:"tool" message keyed by tool_call_id, the Chat
// Completions equivalent of the Responses API's function_call_output item.
func itemsToChatMessages(items []llmprovider.Item) []map[string]any {
	return itemsToChatMessagesReplaying(items, "")
}

// itemsToChatMessagesReplaying is itemsToChatMessages that, when field is set,
// puts the reasoning preceding each assistant message into that field, and
// sets it (possibly "") on every assistant message, as OpenCode's client does
// for interleaved models (MADR 0012 §2, O5).
func itemsToChatMessagesReplaying(items []llmprovider.Item, field string) []map[string]any {
	var messages []map[string]any
	var pending strings.Builder
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.ReasoningItem:
			pending.WriteString(v.Text)
		case llmprovider.MessageItem:
			role := string(v.Role)
			if role == "" {
				role = wire.RoleUser
			}
			messages = append(messages, map[string]any{
				wire.KeyRole:    role,
				wire.KeyContent: v.Text,
			})
		case llmprovider.FunctionCallItem:
			call := map[string]any{
				"id":         v.CallID,
				wire.KeyType: keyFunction,
				keyFunction: map[string]any{
					wire.KeyName:      v.Name,
					wire.KeyArguments: v.Arguments,
				},
			}
			// A call joins the assistant turn it follows (its text, or the
			// calls before it); otherwise it opens one (MADR 0012 §2).
			if n := len(messages); n > 0 && messages[n-1][wire.KeyRole] == wire.RoleAssistant {
				calls, ok := messages[n-1][keyToolCalls].([]map[string]any)
				if !ok {
					calls = nil
				}
				messages[n-1][keyToolCalls] = append(calls, call)
				continue
			}
			messages = append(messages, map[string]any{
				wire.KeyRole:    wire.RoleAssistant,
				wire.KeyContent: "",
				keyToolCalls:    []map[string]any{call},
			})
		case llmprovider.FunctionCallOutputItem:
			messages = append(messages, map[string]any{
				wire.KeyRole:    roleTool,
				"tool_call_id":  v.CallID,
				wire.KeyContent: v.Output,
			})
		}
		if field == "" {
			continue
		}
		// The reasoning belongs to the assistant turn it precedes.
		if n := len(messages); n > 0 && messages[n-1][wire.KeyRole] == wire.RoleAssistant {
			if _, isReasoning := item.(llmprovider.ReasoningItem); !isReasoning {
				prior, _ := messages[n-1][field].(string) //nolint:errcheck // absent is ""
				messages[n-1][field] = prior + pending.String()
				pending.Reset()
			}
		}
	}
	return messages
}

// toolList is tools as Chat Completions functions.
func toolList(tools []llmprovider.Tool) []map[string]any {
	list := make([]map[string]any, len(tools))
	for i, tool := range tools {
		list[i] = map[string]any{
			wire.KeyType: keyFunction,
			keyFunction: map[string]any{
				wire.KeyName:   tool.Name,
				keyDescription: tool.Description,
				keyParameters:  wire.ToolSchema(tool.Schema),
			},
		}
	}
	return list
}

// toolChoice is choice as tool_choice, or nil for auto.
func toolChoice(choice llmprovider.ToolChoice) any {
	if name, forced := choice.Tool(); forced {
		return map[string]any{wire.KeyType: keyFunction, keyFunction: map[string]any{wire.KeyName: name}}
	}
	if choice == llmprovider.ToolChoiceRequired || choice == llmprovider.ToolChoiceNone {
		return string(choice)
	}
	return nil
}

// Body builds an OpenAI Chat Completions request body.
func Body(model string, maxTokens int, input []llmprovider.Item, o Opts) map[string]any {
	body := map[string]any{
		keyModel:     model,
		keyMessages:  itemsToChatMessagesReplaying(input, o.ReplayReasoningField),
		keyMaxTokens: maxTokens,
	}
	if len(o.Tools) > 0 && (!o.NoToolChoice || o.ToolChoice != llmprovider.ToolChoiceNone) {
		body[keyTools] = toolList(o.Tools)
		if choice := toolChoice(o.ToolChoice); choice != nil && !o.NoToolChoice {
			body[keyToolChoice] = choice
		}
	}
	if o.ReasoningEffort != "" {
		body[keyReasoningEffort] = o.ReasoningEffort
	}
	if o.Reasoning != nil {
		body[keyReasoning] = o.Reasoning
	}
	return body
}

// Decode decodes an OpenAI Chat Completions envelope into a canonical
// Response.
//
// Reasoning has two competing vendor spellings, both undocumented, both measured
// 2026-08-28/29:
//
//	message.reasoning_content  — OpenCode Zen (verified on big-pickle)
//	message.reasoning          — Kilo Gateway, the OpenRouter convention
//
// Both are accepted; reasoning_content wins when both are present. Kilo also
// sends message.reasoning_details[] ({type:"reasoning.text", text}), a structured
// restatement of the same trace; it is deliberately NOT decoded, because
// ReasoningItem carries a single Text field and parsing both would create two
// sources of truth for one value.
//
// Absent reasoning is normal, never an error.
func Decode(body io.Reader) (*llmprovider.Response, error) {
	var raw struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role             string `json:"role"`
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				Reasoning        string `json:"reasoning"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage usage `json:"usage"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw.Choices) == 0 {
		return nil, fmt.Errorf("%w: chat completions: the answer has no choices", llmprovider.ErrIncomplete)
	}

	msg := raw.Choices[0].Message
	finish := llmprovider.FinishReason(raw.Choices[0].FinishReason)
	// A tool call cut by the token limit has unusable arguments (MADR 0012 §1.5).
	if finish == llmprovider.FinishLength && len(msg.ToolCalls) > 0 {
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}
	// The response id is not a resumable conversation handle on any gateway
	// that speaks this format, so it is carried for logging only.
	result := &llmprovider.Response{ID: raw.ID, Model: raw.Model, FinishReason: finish, Usage: raw.Usage.counts()}

	reasoning := msg.ReasoningContent
	if reasoning == "" {
		reasoning = msg.Reasoning
	}
	if strings.TrimSpace(reasoning) != "" {
		result.Output = append(result.Output, llmprovider.ReasoningItem{Text: reasoning})
	}
	if msg.Content != "" {
		role := msg.Role
		if role == "" {
			role = wire.RoleAssistant
		}
		result.Output = append(result.Output, llmprovider.MessageItem{Role: llmprovider.Role(role), Text: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		result.Output = append(result.Output, llmprovider.FunctionCallItem{
			CallID:    tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	if len(result.Output) == 0 {
		return nil, fmt.Errorf("%w: chat completions: the answer has no usable content", llmprovider.ErrIncomplete)
	}
	return result, nil
}

// usage is Chat Completions' token counts. prompt_tokens holds the cached
// tokens and completion_tokens the reasoning tokens, as Usage's do. The
// cached count has three spellings, read in pi's order: OpenAI's
// prompt_tokens_details.cached_tokens, DeepSeek's prompt_cache_hit_tokens,
// and a top-level cached_tokens (0017-REPORT P7).
type usage struct {
	Prompt        int  `json:"prompt_tokens"`
	Completion    int  `json:"completion_tokens"`
	CacheHit      *int `json:"prompt_cache_hit_tokens"`
	Cached        *int `json:"cached_tokens"`
	PromptDetails struct {
		Cached *int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionDetails struct {
		Reasoning int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

func (u usage) counts() llmprovider.Usage {
	cached := 0
	for _, c := range []*int{u.PromptDetails.Cached, u.CacheHit, u.Cached} {
		if c != nil {
			cached = *c
			break
		}
	}
	return llmprovider.Usage{InputTokens: u.Prompt, OutputTokens: u.Completion,
		ReasoningTokens: u.CompletionDetails.Reasoning, CachedTokens: cached}
}
