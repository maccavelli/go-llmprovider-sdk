package generatecontent

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The encoder as it was at 0028-PLAN Phase 9b's start (HEAD 657d00f), frozen as
// the reference the typed encoder must match JSON for JSON. Generated from
// git show HEAD:llmprovider/internal/wire/generatecontent/generatecontent.go, the functions renamed only.

// refContents is Contents, by maps.
func refContents(items []llmprovider.Item) []map[string]any {
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
			role := string(v.Role)
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
				name = callName(v.CallID)
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

// refSystemInstruction is SystemInstruction, by maps.
func refSystemInstruction(items []llmprovider.Item) map[string]any {
	system := wire.SystemPrompt(items)
	if system == "" {
		return nil
	}
	return map[string]any{"parts": []map[string]any{{wire.KeyText: system}}}
}
