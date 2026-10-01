package llmprovider

import (
	"encoding/json"
	"fmt"
	"io"
)

// Gemini's generateContent wire, which OpenCode's google route speaks
// (opencode.go). The gemini provider speaks the Interactions API instead
// (providers/gemini, MADR 0014); this stays here until 0015-PLAN S7b moves
// it to internal/wire.

// dynamicGeminiThinkingBudget (-1) lets the model size its own thinking budget.
const dynamicGeminiThinkingBudget = -1

// geminiSystemInstruction is generateContent's systemInstruction for the
// system items, or nil when there are none (MADR 0014 §3).
func geminiSystemInstruction(items []Item) map[string]any {
	system := SystemPrompt(items)
	if system == "" {
		return nil
	}
	return map[string]any{"parts": []map[string]any{{jsonKeyText: system}}}
}

// geminiItemsToContents is generateContent's contents, for OpenCode's google
// route.
func geminiItemsToContents(items []Item) []map[string]any {
	// Gemini pairs a functionResponse with its functionCall by name, so a
	// result takes the name of the call it answers (MADR 0012 §2).
	names := map[string]string{}
	for _, item := range items {
		if call, ok := item.(FunctionCallItem); ok {
			names[call.CallID] = call.Name
		}
	}
	var contents []map[string]any
	lastIsResponses := false
	appendPart := func(role string, part map[string]any, merge bool) {
		if n := len(contents); merge && n > 0 && contents[n-1][jsonKeyRole] == role {
			if parts, ok := contents[n-1]["parts"].([]map[string]any); ok {
				contents[n-1]["parts"] = append(parts, part)
				return
			}
		}
		contents = append(contents, map[string]any{jsonKeyRole: role, "parts": []map[string]any{part}})
	}
	for _, item := range items {
		switch v := item.(type) {
		case MessageItem:
			if v.Role == jsonRoleSystem {
				continue // systemInstruction; see geminiSystemInstruction
			}
			role := v.Role
			if role == "" || role == jsonRoleUser {
				role = jsonRoleUser
			} else {
				role = geminiRoleModel
			}
			appendPart(role, map[string]any{jsonKeyText: v.Text}, false)
			lastIsResponses = false
		case FunctionCallItem:
			// Gemini refuses a replayed call without its thoughtSignature. A call
			// Gemini did not issue (synthetic, or from another provider) carries
			// the documented skip value instead.
			signature := v.Signature
			if signature == "" {
				signature = geminiSkipThoughtSignature
			}
			appendPart(geminiRoleModel, map[string]any{
				"functionCall":     map[string]any{jsonKeyName: v.Name, "args": ToolArguments(v.Arguments)},
				"thoughtSignature": signature,
			}, true)
			lastIsResponses = false
		case FunctionCallOutputItem:
			name := names[v.CallID]
			if name == "" {
				name = v.CallID
			}
			// A turn's results share one user turn, as its calls share one model turn.
			appendPart(jsonRoleUser, map[string]any{
				"functionResponse": map[string]any{
					jsonKeyName: name,
					"response":  map[string]any{jsonKeyOutput: v.Output},
				},
			}, lastIsResponses)
			lastIsResponses = true
		}
	}
	return contents
}

// decodeGeminiResponse decodes a generateContent response.
func decodeGeminiResponse(body io.Reader) (*Response, error) {
	var raw struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					Thought      bool   `json:"thought"`
					FunctionCall *struct {
						Name string         `json:"name"`
						Args map[string]any `json:"args"`
					} `json:"functionCall"`
					ThoughtSignature string `json:"thoughtSignature"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}

	if len(raw.Candidates) == 0 || len(raw.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content")
	}

	result := &Response{}
	for _, part := range raw.Candidates[0].Content.Parts {
		// A thought summary is flagged thought: true, with its text in text.
		if part.Thought {
			if part.Text != "" {
				result.Output = append(result.Output, ReasoningItem{Text: part.Text})
			}
			continue
		}
		if part.Text != "" {
			result.Output = append(result.Output, MessageItem{Role: jsonRoleAssistant, Text: part.Text})
		}
		if part.FunctionCall != nil {
			args := part.FunctionCall.Args
			if args == nil {
				args = map[string]any{} // a tool without parameters
			}
			argsBytes, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal gemini function args: %w", err)
			}
			result.Output = append(result.Output, FunctionCallItem{
				CallID:    part.FunctionCall.Name,
				Name:      part.FunctionCall.Name,
				Arguments: string(argsBytes),
				Signature: part.ThoughtSignature,
			})
		}
	}

	return result, nil
}
