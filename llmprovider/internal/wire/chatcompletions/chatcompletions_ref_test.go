package chatcompletions

import (
	"encoding/json"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The encoder as it was at 0028-PLAN Phase 8's start (HEAD 211d686), frozen as
// the reference the typed encoder must match JSON for JSON. Generated from
// git show HEAD:llmprovider/internal/wire/chatcompletions/chatcompletions.go, the functions renamed, and the two
// errcheck directives, which a test file does not need, removed.

// refItemsToChatMessagesReplaying is itemsToChatMessagesReplaying, by maps.
func refItemsToChatMessagesReplaying(items []llmprovider.Item, field string, details bool) []map[string]any {
	var messages []map[string]any
	var pending strings.Builder
	var pendingDetails []json.RawMessage
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.ReasoningItem:
			pending.WriteString(v.Text)
			if detail := strings.TrimSpace(v.Encrypted); details && v.Format == wire.FormatChatCompletions &&
				strings.HasPrefix(detail, "{") && json.Valid([]byte(detail)) {
				pendingDetails = append(pendingDetails, json.RawMessage(detail))
			}
		case llmprovider.MessageItem:
			role := string(v.Role)
			if role == "" {
				role = wire.RoleUser
			}
			// Reasoning no assistant message followed, such as an answer
			// cut while reasoning, ends with its turn (0026-MADR F32).
			if role == wire.RoleUser {
				pending.Reset()
				pendingDetails = nil
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
		if field == "" && !details {
			continue
		}
		// The reasoning belongs to the assistant turn it precedes.
		if n := len(messages); n > 0 && messages[n-1][wire.KeyRole] == wire.RoleAssistant {
			if _, isReasoning := item.(llmprovider.ReasoningItem); !isReasoning {
				if field != "" {
					prior, _ := messages[n-1][field].(string)
					messages[n-1][field] = prior + pending.String()
				}
				pending.Reset()
				if len(pendingDetails) > 0 {
					prior, _ := messages[n-1][keyReasoningDetail].([]json.RawMessage)
					messages[n-1][keyReasoningDetail] = append(prior, pendingDetails...)
					pendingDetails = nil
				}
			}
		}
	}
	return messages
}

// refToolList is toolList, by maps.
func refToolList(tools []llmprovider.Tool) []map[string]any {
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
