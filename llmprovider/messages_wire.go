package llmprovider

import (
	"encoding/json"
	"fmt"
	"io"
)

// The Anthropic Messages wire, shared by the claude provider package and
// OpenCode's messages route. It left claude.go when Claude moved to its own
// package (0015-PLAN S7); S7b moves it to internal/wire.

// defaultClaudeThinkingBudget is used by GenerateThinking when no budget is configured.
const defaultClaudeThinkingBudget = 4096

// MessagesFromItems converts items to Anthropic messages. System items
// are left out: they go to the top-level system field (see SystemPrompt).
// Temporary export for the provider packages (0015-PLAN S7); S7b moves it to internal/wire.
func MessagesFromItems(items []Item) []map[string]any {
	var messages []map[string]any
	// appendBlock adds a content block to the previous message when it has the
	// same role and already holds blocks (or, for the assistant, text), so a
	// turn's calls and a turn's results each stay in one message (MADR 0012
	// §2); otherwise it opens a message.
	appendBlock := func(role string, block map[string]any) {
		if n := len(messages); n > 0 && messages[n-1][jsonKeyRole] == role {
			switch content := messages[n-1][jsonKeyContent].(type) {
			case []map[string]any:
				messages[n-1][jsonKeyContent] = append(content, block)
				return
			case string:
				if role == jsonRoleAssistant {
					messages[n-1][jsonKeyContent] = []map[string]any{{jsonKeyType: jsonKeyText, jsonKeyText: content}, block}
					return
				}
			}
		}
		messages = append(messages, map[string]any{jsonKeyRole: role, jsonKeyContent: []map[string]any{block}})
	}
	for _, item := range items {
		switch v := item.(type) {
		case MessageItem:
			role := v.Role
			if role == jsonRoleSystem {
				continue // top-level system field; see SystemPrompt
			}
			if role == "" || role == jsonRoleUser {
				role = jsonRoleUser
			} else {
				role = jsonRoleAssistant
			}
			messages = append(messages, map[string]any{
				jsonKeyRole:    role,
				jsonKeyContent: v.Text,
			})
		case FunctionCallItem:
			appendBlock(jsonRoleAssistant, map[string]any{
				jsonKeyType:  "tool_use",
				"id":         v.CallID,
				jsonKeyName:  v.Name,
				jsonKeyInput: toolArguments(v.Arguments),
			})
		case FunctionCallOutputItem:
			appendBlock(jsonRoleUser, map[string]any{
				jsonKeyType:    "tool_result",
				"tool_use_id":  v.CallID,
				jsonKeyContent: v.Output,
			})
		}
	}
	return messages
}

// DecodeMessagesResponse decodes a Messages API response. The API is
// stateless, so the Response has no ID.
// Temporary export for the provider packages (0015-PLAN S7); S7b moves it to internal/wire.
func DecodeMessagesResponse(body io.Reader) (*Response, error) {
	var result struct {
		Content []struct {
			Type     string         `json:"type"`
			Text     string         `json:"text"`
			Thinking string         `json:"thinking"`
			ID       string         `json:"id"`
			Name     string         `json:"name"`
			Input    map[string]any `json:"input"`
		} `json:"content"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty content")
	}

	res := &Response{ID: ""} // Claude Messages API is stateless
	for _, b := range result.Content {
		switch b.Type {
		case "thinking":
			text := b.Thinking
			if text == "" {
				text = b.Text
			}
			res.Output = append(res.Output, ReasoningItem{Text: text})
		case jsonKeyText, "":
			if b.Text != "" {
				res.Output = append(res.Output, MessageItem{Role: jsonRoleAssistant, Text: b.Text})
			}
		case "tool_use":
			var argsStr string
			if b.Input != nil {
				argsBytes, err := json.Marshal(b.Input)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal claude tool input: %w", err)
				}
				argsStr = string(argsBytes)
			}
			res.Output = append(res.Output, FunctionCallItem{
				CallID:    b.ID,
				Name:      b.Name,
				Arguments: argsStr,
			})
		}
	}

	if len(res.Output) == 0 {
		return nil, fmt.Errorf("claude returned no usable content")
	}

	return res, nil
}
