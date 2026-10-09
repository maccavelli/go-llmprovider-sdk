package responses

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The encoder as it was at 0028-PLAN Phase 9b's start (HEAD 657d00f), frozen as
// the reference the typed encoder must match JSON for JSON. Generated from
// git show HEAD:llmprovider/internal/wire/responses/responses.go, the functions renamed only.

// refInput is Input, by maps.
func refInput(items []llmprovider.Item) []map[string]any {
	var input []map[string]any
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			role := string(v.Role)
			if role == "" {
				role = wire.RoleUser // an empty Role is the user's (0020-MADR F10)
			}
			input = append(input, map[string]any{
				wire.KeyRole:    role,
				wire.KeyContent: v.Text,
			})
		case llmprovider.ReasoningItem:
			// Encrypted reasoning goes back as it came (0020-MADR F24), to
			// this wire only (0021-MADR W7); plain text reasoning cannot be
			// replayed.
			if v.Encrypted != "" && wire.Replays(v, wire.FormatResponses) {
				input = append(input, map[string]any{
					wire.KeyType:        itemTypeReasoning,
					"encrypted_content": v.Encrypted,
					"summary":           []any{},
				})
			}
		case llmprovider.FunctionCallItem:
			input = append(input, map[string]any{
				wire.KeyType:      itemTypeFunctionCall,
				keyCallID:         v.CallID,
				wire.KeyName:      v.Name,
				wire.KeyArguments: v.Arguments,
			})
		case llmprovider.FunctionCallOutputItem:
			input = append(input, map[string]any{
				wire.KeyType:   itemTypeFunctionCallOutput,
				keyCallID:      v.CallID,
				wire.KeyOutput: v.Output,
			})
		}
	}
	return input
}
