package messages

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The encoder as it was at 0028-PLAN Phase 9b's start (HEAD 657d00f), frozen as
// the reference the typed encoder must match JSON for JSON. Generated from
// git show HEAD:llmprovider/internal/wire/messages/messages.go, the functions renamed only.

// refFromItems is FromItems, by maps.
func refFromItems(items []llmprovider.Item) []map[string]any {
	items = wire.RemapCallIDs(items, wire.AnthropicCallIDs)
	var messages []map[string]any
	// appendBlock adds a content block to the previous message when it has the
	// same role and already holds blocks (or, for the assistant, text), so a
	// turn's calls and a turn's results each stay in one message (MADR 0012
	// §2); otherwise it opens a message.
	appendBlock := func(role string, block map[string]any) {
		if n := len(messages); n > 0 && messages[n-1][wire.KeyRole] == role {
			switch content := messages[n-1][wire.KeyContent].(type) {
			case []map[string]any:
				messages[n-1][wire.KeyContent] = append(content, block)
				return
			case string:
				if role == wire.RoleAssistant {
					messages[n-1][wire.KeyContent] = []map[string]any{{wire.KeyType: wire.KeyText, wire.KeyText: content}, block}
					return
				}
			}
		}
		messages = append(messages, map[string]any{wire.KeyRole: role, wire.KeyContent: []map[string]any{block}})
	}
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			role := string(v.Role)
			if role == wire.RoleSystem {
				continue // top-level system field; see wire.SystemPrompt
			}
			if role == "" || role == wire.RoleUser {
				role = wire.RoleUser
			} else {
				role = wire.RoleAssistant
			}
			messages = append(messages, map[string]any{
				wire.KeyRole:    role,
				wire.KeyContent: v.Text,
			})
		case llmprovider.ReasoningItem:
			// Anthropic needs a turn's thinking back, signature included, when
			// thinking and tools are combined. Unsigned reasoning, and another
			// wire's, cannot be replayed (0020-MADR F7; 0021-MADR W7).
			switch {
			case !wire.Replays(v, wire.FormatMessages):
			case v.Signature != "":
				appendBlock(wire.RoleAssistant, map[string]any{
					wire.KeyType: keyThinking, keyThinking: v.Text, keySignature: v.Signature,
				})
			case v.Encrypted != "":
				appendBlock(wire.RoleAssistant, map[string]any{wire.KeyType: blockRedactedThinking, keyData: v.Encrypted})
			}
		case llmprovider.FunctionCallItem:
			appendBlock(wire.RoleAssistant, map[string]any{
				wire.KeyType: "tool_use",
				"id":         v.CallID,
				wire.KeyName: v.Name,
				keyInput:     wire.ToolArguments(v.Arguments),
			})
		case llmprovider.FunctionCallOutputItem:
			appendBlock(wire.RoleUser, map[string]any{
				wire.KeyType:    "tool_result",
				"tool_use_id":   v.CallID,
				wire.KeyContent: v.Output,
			})
		}
	}
	return messages
}
