package wire

import (
	"encoding/json"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The estimate fit (0088-MADR ST87 as amended, ST105, ST106; OQ14 (3)): a
// request's input, estimated from its content as pi counts it, and the room
// a stated window leaves the output limit the SDK chose. pi's numbers, none
// measured (0088-MADR's "Every number" tables).
const (
	// fitMargin is the tokens left spare beyond the estimate
	// (pi:packages/ai/src/api/simple-options.ts:15).
	fitMargin = 4096
	// imageChars is an image's count, whatever its size
	// (pi:packages/ai/src/utils/estimate.ts:16).
	imageChars = 4800
)

// EstimateInput is req's input in tokens, estimated as pi estimates it
// (0088-MADR ST87): the characters (Unicode code points) of the instructions,
// of each message's text, of each reasoning item's text (not its signature
// or encrypted form), of each call's name and arguments, of each call output,
// of each loaded tool's name, and of each tool sent that is not deferred (its
// name, description and JSON-encoded schema), with each image at 4,800
// characters; at 3.5 characters a token, rounded up. A stored conversation
// the request continues (PreviousResponseID) is not counted.
func EstimateInput(req *llmprovider.Request) int {
	chars := utf8.RuneCountInString(req.Instructions)
	for _, item := range req.Input {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			chars += utf8.RuneCountInString(v.Text)
		case llmprovider.ReasoningItem:
			chars += utf8.RuneCountInString(v.Text)
		case llmprovider.FunctionCallItem:
			chars += utf8.RuneCountInString(v.Name) + utf8.RuneCountInString(v.Arguments)
		case llmprovider.FunctionCallOutputItem:
			chars += utf8.RuneCountInString(v.Output)
		case llmprovider.ToolLoadItem:
			chars += utf8.RuneCountInString(v.Name)
		case llmprovider.ImageItem:
			chars += imageChars
		}
	}
	for _, tool := range req.Tools {
		if tool.Deferred {
			continue
		}
		chars += utf8.RuneCountInString(tool.Name) + utf8.RuneCountInString(tool.Description)
		if tool.Schema != nil {
			if raw, err := json.Marshal(tool.Schema); err == nil {
				chars += utf8.RuneCount(raw)
			}
		}
	}
	// chars / 3.5, rounded up.
	return (2*chars + 6) / 7
}

// fit is tokens held to the room in's window leaves the estimated input
// (0088-MADR ST87, ST105): the window less the estimate less fitMargin, never
// below the wire's minimum; tokens as they are when no window is stated.
func fit(tokens int, in LimitInput) int {
	if in.Window <= 0 {
		return tokens
	}
	return min(tokens, max(in.Window-in.Estimate-fitMargin, in.Minimum, 1))
}
