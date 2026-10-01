// Package generatecontent is Gemini's generateContent wire, which OpenCode's
// google route speaks (0015-MADR D2), with its thinking request shape. The
// gemini provider speaks the Interactions API instead, in providers/gemini
// (MADR 0014).
package generatecontent

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const (
	// roleModel is Gemini's assistant role.
	roleModel = "model"
	// skipThoughtSignature is Gemini's documented placeholder for a replayed
	// call it did not issue (accepted live, 2026-09-27).
	skipThoughtSignature = "skip_thought_signature_validator"

	keyThinkingBudget = "thinkingBudget"
)

// dynamicThinkingBudget (-1) lets the model size its own thinking budget.
const dynamicThinkingBudget = -1

// SystemInstruction is generateContent's systemInstruction for the system
// items, or nil when there are none (MADR 0014 §3).
func SystemInstruction(items []llmprovider.Item) map[string]any {
	system := wire.SystemPrompt(items)
	if system == "" {
		return nil
	}
	return map[string]any{"parts": []map[string]any{{wire.KeyText: system}}}
}

// Contents is generateContent's contents, for OpenCode's google route.
func Contents(items []llmprovider.Item) []map[string]any {
	// Gemini pairs a functionResponse with its functionCall by name, so a
	// result takes the name of the call it answers (MADR 0012 §2).
	names := map[string]string{}
	for _, item := range items {
		if call, ok := item.(llmprovider.FunctionCallItem); ok {
			names[call.CallID] = call.Name
		}
	}
	var contents []map[string]any
	lastIsResponses := false
	appendPart := func(role string, part map[string]any, merge bool) {
		if n := len(contents); merge && n > 0 && contents[n-1][wire.KeyRole] == role {
			if parts, ok := contents[n-1]["parts"].([]map[string]any); ok {
				contents[n-1]["parts"] = append(parts, part)
				return
			}
		}
		contents = append(contents, map[string]any{wire.KeyRole: role, "parts": []map[string]any{part}})
	}
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			if v.Role == wire.RoleSystem {
				continue // systemInstruction; see SystemInstruction
			}
			role := v.Role
			if role == "" || role == wire.RoleUser {
				role = wire.RoleUser
			} else {
				role = roleModel
			}
			appendPart(role, map[string]any{wire.KeyText: v.Text}, false)
			lastIsResponses = false
		case llmprovider.FunctionCallItem:
			// Gemini refuses a replayed call without its thoughtSignature. A call
			// Gemini did not issue (synthetic, or from another provider) carries
			// the documented skip value instead.
			signature := v.Signature
			if signature == "" {
				signature = skipThoughtSignature
			}
			appendPart(roleModel, map[string]any{
				"functionCall":     map[string]any{wire.KeyName: v.Name, "args": wire.ToolArguments(v.Arguments)},
				"thoughtSignature": signature,
			}, true)
			lastIsResponses = false
		case llmprovider.FunctionCallOutputItem:
			name := names[v.CallID]
			if name == "" {
				name = v.CallID
			}
			// A turn's results share one user turn, as its calls share one model turn.
			appendPart(wire.RoleUser, map[string]any{
				"functionResponse": map[string]any{
					wire.KeyName: name,
					"response":   map[string]any{wire.KeyOutput: v.Output},
				},
			}, lastIsResponses)
			lastIsResponses = true
		}
	}
	return contents
}

// Decode decodes a generateContent response.
func Decode(body io.Reader) (*llmprovider.Response, error) {
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

	result := &llmprovider.Response{}
	for _, part := range raw.Candidates[0].Content.Parts {
		// A thought summary is flagged thought: true, with its text in text.
		if part.Thought {
			if part.Text != "" {
				result.Output = append(result.Output, llmprovider.ReasoningItem{Text: part.Text})
			}
			continue
		}
		if part.Text != "" {
			result.Output = append(result.Output, llmprovider.MessageItem{Role: wire.RoleAssistant, Text: part.Text})
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
			result.Output = append(result.Output, llmprovider.FunctionCallItem{
				CallID:    part.FunctionCall.Name,
				Name:      part.FunctionCall.Name,
				Arguments: string(argsBytes),
				Signature: part.ThoughtSignature,
			})
		}
	}

	return result, nil
}

// geminiLegacyRE matches Gemini 1.x and 2.x ids, which take thinkingBudget but
// not thinkingLevel (HTTP 400 on gemini-2.5-flash, measured 2026-09-27). It is
// OpenCode's GEMINI_LEGACY_RE.
var geminiLegacyRE = regexp.MustCompile(`(?i)gemini-(?:(?:flash|pro)-)?[12](?:[.-]|$)`)

// ThinkingConfig returns a Gemini thinkingConfig. A configured budget wins.
// Otherwise "low" is thinkingLevel "low" on Gemini 3 and later and a
// wire.LowEffortThinkingBudget budget on 1.x and 2.x, and any other effort
// keeps the dynamic budget (MADR 0013 Q1, B9).
func ThinkingConfig(model, effort string, budget int) map[string]any {
	// Thought summaries come back only when asked for, as OpenCode's client
	// asks (transform.ts:1280-1288, MADR 0014 §3).
	low := string(llmprovider.EffortLow)
	cfg := map[string]any{"includeThoughts": true}
	switch {
	case budget > 0:
		cfg[keyThinkingBudget] = budget
	case effort != low:
		cfg[keyThinkingBudget] = dynamicThinkingBudget
	case geminiLegacyRE.MatchString(model):
		cfg[keyThinkingBudget] = wire.LowEffortThinkingBudget
	default:
		cfg["thinkingLevel"] = low
	}
	return cfg
}
