// Package messages is the Anthropic Messages wire, shared by the claude
// provider and OpenCode's messages route (0015-MADR D2), with its thinking
// request shape.
package messages

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"io"
	"maps"
	"regexp"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const (
	keyInput     = "input"
	keyEffort    = "effort"
	keyThinking  = "thinking"
	keyEnabled   = "enabled"
	keySignature = "signature"
	keyData      = "data"

	blockRedactedThinking = "redacted_thinking"
	blockToolUse          = "tool_use"
)

// stopReasons maps Anthropic's stop_reason to FinishReason (0020-MADR F3).
// model_context_window_exceeded is a cut answer, as the Grok CLI reads it
// (0021-MADR W3).
var stopReasons = map[string]llmprovider.FinishReason{
	"model_context_window_exceeded": llmprovider.FinishLength,
	"end_turn":                      llmprovider.FinishStop,
	"stop_sequence":                 llmprovider.FinishStop,
	"pause_turn":                    llmprovider.FinishStop,
	"max_tokens":                    llmprovider.FinishLength,
	"tool_use":                      llmprovider.FinishToolCalls,
	"refusal":                       llmprovider.FinishContentFilter,
}

// defaultThinkingBudget is the thinking budget when none is configured.
const defaultThinkingBudget = 4096

// Message is one Messages API message. The fields of Message and Block are
// in the order json.Marshal wrote the maps they replace, so the JSON is the
// same byte for byte (0028-MADR D-A6, amendments 2026-10-09).
type Message struct {
	Content Content `json:"content"`
	Role    string  `json:"role"`
}

// Content is a message's content: its text, or its blocks. It is not an
// interface: json.Marshal, which runs on encoding/json/v2, copies every
// interface value it encodes into a new one, an allocation apiece
// (0028-PLAN D16).
type Content struct {
	text   *string
	blocks *[]Block
}

// MarshalJSONTo writes the text as a string, or the blocks as a list.
func (c Content) MarshalJSONTo(enc *jsontext.Encoder) error {
	if c.blocks != nil {
		return jsonv2.MarshalEncode(enc, *c.blocks)
	}
	return enc.WriteToken(jsontext.String(*c.text))
}

// MarshalJSON is MarshalJSONTo for an encoder that does not call it; json/v2
// prefers MarshalJSONTo.
func (c Content) MarshalJSON() ([]byte, error) {
	if c.blocks != nil {
		return json.Marshal(*c.blocks)
	}
	return json.Marshal(*c.text)
}

// Block is every content block in one struct: text, thinking,
// redacted_thinking, tool_use and tool_result. A field a block does not have
// is nil, and omitted; Type is always sent.
type Block struct {
	Content   *string `json:"content,omitempty"`
	Data      *string `json:"data,omitempty"`
	ID        *string `json:"id,omitempty"`
	Input     any     `json:"input,omitempty"`
	Name      *string `json:"name,omitempty"`
	Signature *string `json:"signature,omitempty"`
	Text      *string `json:"text,omitempty"`
	Thinking  *string `json:"thinking,omitempty"`
	ToolUseID *string `json:"tool_use_id,omitempty"`
	Type      string  `json:"type"`
}

// FromItems converts items to Anthropic messages. System items are left out:
// they go to the top-level system field (see wire.SystemPrompt). Call ids
// another wire made are sent in a form Anthropic accepts, unique in the
// request, and each result keeps its call (0026-MADR F8).
//
// The strings the blocks carry are kept in one arena, and every message's
// blocks in one backing array: only the last message ever takes a block, so
// each message's blocks are contiguous (0028-PLAN D16).
func FromItems(items []llmprovider.Item) []Message {
	items = wire.RemapCallIDs(items, wire.AnthropicCallIDs)
	var (
		messages []Message
		arena    = wire.NewArena(4 * len(items))
		blocks   = make([]Block, 0, 2*len(items))
		lists    = make([][]Block, 0, len(items))
		start    int // where the last message's blocks begin in blocks
	)
	// openList starts a block list holding first, as a content.
	openList := func(first ...Block) Content {
		start = len(blocks)
		blocks = append(blocks, first...)
		lists = append(lists, blocks[start:len(blocks):len(blocks)])
		return Content{blocks: &lists[len(lists)-1]}
	}
	// appendBlock adds a content block to the previous message when it has the
	// same role and already holds blocks (or, for the assistant, text), so a
	// turn's calls and a turn's results each stay in one message (MADR 0012
	// §2); otherwise it opens a message.
	appendBlock := func(role string, block Block) {
		if n := len(messages); n > 0 && messages[n-1].Role == role {
			switch content := messages[n-1].Content; {
			case content.blocks != nil:
				blocks = append(blocks, block)
				*content.blocks = blocks[start:len(blocks):len(blocks)]
				return
			case role == wire.RoleAssistant:
				messages[n-1].Content = openList(Block{Type: wire.KeyText, Text: content.text}, block)
				return
			}
		}
		messages = append(messages, Message{Role: role, Content: openList(block)})
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
			messages = append(messages, Message{Role: role, Content: Content{text: arena.Keep(v.Text)}})
		case llmprovider.ReasoningItem:
			// Anthropic needs a turn's thinking back, signature included, when
			// thinking and tools are combined. Unsigned reasoning, and another
			// wire's, cannot be replayed (0020-MADR F7; 0021-MADR W7).
			switch {
			case !wire.Replays(v, wire.FormatMessages):
			case v.Signature != "":
				appendBlock(wire.RoleAssistant, Block{Type: keyThinking, Thinking: arena.Keep(v.Text), Signature: arena.Keep(v.Signature)})
			case v.Encrypted != "":
				appendBlock(wire.RoleAssistant, Block{Type: blockRedactedThinking, Data: arena.Keep(v.Encrypted)})
			}
		case llmprovider.FunctionCallItem:
			appendBlock(wire.RoleAssistant, Block{
				Type: "tool_use", ID: arena.Keep(v.CallID), Name: arena.Keep(v.Name), Input: wire.ToolArguments(v.Arguments),
			})
		case llmprovider.FunctionCallOutputItem:
			appendBlock(wire.RoleUser, Block{Type: "tool_result", ToolUseID: arena.Keep(v.CallID), Content: arena.Keep(v.Output)})
		}
	}
	return messages
}

// Decode decodes a Messages API response. The API is stateless, so the
// Response has no ID. stop_reason becomes FinishReason, and a tool call cut
// by max_tokens is ErrIncomplete (0020-MADR F3; MADR 0012 §1.5).
func Decode(body io.Reader) (*llmprovider.Response, error) {
	var result struct {
		Model      string `json:"model"`
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type      string          `json:"type"`
			Text      string          `json:"text"`
			Thinking  string          `json:"thinking"`
			Signature string          `json:"signature"`
			Data      string          `json:"data"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
		} `json:"content"`
		Usage usage `json:"usage"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return nil, err
	}

	hasCall := false
	for _, b := range result.Content {
		hasCall = hasCall || b.Type == blockToolUse
	}
	finish := wire.Finish(result.StopReason, stopReasons, hasCall)
	if len(result.Content) == 0 {
		return nil, wire.EmptyAnswer("messages", finish)
	}
	if finish == llmprovider.FinishLength && hasCall {
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}

	// The Messages API is stateless: no ID.
	res := &llmprovider.Response{Model: result.Model, FinishReason: finish, Usage: result.Usage.counts()}
	for _, b := range result.Content {
		switch b.Type {
		case keyThinking:
			text := b.Thinking
			if text == "" {
				text = b.Text
			}
			res.Output = append(res.Output, llmprovider.ReasoningItem{Text: text, Signature: b.Signature, Format: wire.FormatMessages})
		case blockRedactedThinking:
			res.Output = append(res.Output, llmprovider.ReasoningItem{Encrypted: b.Data, Format: wire.FormatMessages})
		case wire.KeyText, "":
			if b.Text != "" {
				res.Output = append(res.Output, llmprovider.MessageItem{Role: wire.RoleAssistant, Text: b.Text})
			}
		case blockToolUse:
			argsStr, err := wire.CompactArguments(b.Input)
			if err != nil {
				return nil, fmt.Errorf("messages: tool_use input: %w", err)
			}
			res.Output = append(res.Output, llmprovider.FunctionCallItem{
				CallID:    b.ID,
				Name:      b.Name,
				Arguments: argsStr,
			})
		}
	}

	// An answer cut while it was still thinking has nothing usable
	// (0026-MADR F7); a cut text answer keeps its text.
	if len(res.Output) == 0 || finish == llmprovider.FinishLength && res.OutputText() == "" {
		return nil, wire.EmptyAnswer("messages", finish)
	}

	return res, nil
}

// The thinking shape, budget-based, as in OpenCode's own client
// (packages/opencode/src/provider/transform.ts): the model id selects it
// (MADR 0013 Q1, B9).

// claudeVersionRE reads a Claude version, family-first (claude-opus-4-8) or
// version-first (claude-4.7-opus). A minor has at most two digits, so the date
// in claude-sonnet-4-20250514 is not read as one. OpenCode's
// anthropicUsesModernAdaptiveThinking uses the same pattern.
var claudeVersionRE = regexp.MustCompile(`(?i)claude-(?:[a-z]+-)?(\d+)(?:[.-](\d{1,2}))?(?:[.@-]|$)`)

// claudeAdaptiveOnly reports whether model refuses thinking.type "enabled":
// Claude 4.7 and later, and a claude- id with no readable version (HTTP 400 on
// claude-sonnet-5 and claude-opus-4-8, measured 2026-09-27). Other ids,
// including the non-Claude models OpenCode serves on its messages route, take
// a budget.
func claudeAdaptiveOnly(model string) bool {
	lower := strings.ToLower(model)
	if !strings.Contains(lower, "claude-") {
		return false
	}
	m := claudeVersionRE.FindStringSubmatch(lower)
	if m == nil {
		return true
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return true
	}
	minor := 0
	if m[2] != "" {
		if minor, err = strconv.Atoi(m[2]); err != nil {
			return true
		}
	}
	return major > 4 || (major == 4 && minor >= 7)
}

// minimaxAdaptiveThinking reports a MiniMax-M3 id. MiniMax's Anthropic
// interface takes thinking.type "adaptive" with no budget or effort, as
// OpenCode's client sends it (transform.ts:1293-1296, MADR 0012 §3.2).
func minimaxAdaptiveThinking(model string) bool {
	return strings.Contains(strings.ToLower(model), "minimax-m3")
}

// AddThinking adds the thinking fields of an Anthropic Messages request to
// body and returns its max_tokens. Adaptive-only models get thinking.type
// "adaptive", plus output_config.effort when an effort is set; a budget does
// not apply to them. Other models get an enabled budget: the configured one,
// else wire.LowEffortThinkingBudget for "low", else defaultThinkingBudget,
// with max_tokens raised above it when needed (Anthropic requires max_tokens >
// budget_tokens).
func AddThinking(body map[string]any, model, effort string, budget, maxTokens int) int {
	if minimaxAdaptiveThinking(model) {
		body[keyThinking] = map[string]any{wire.KeyType: "adaptive"}
		return maxTokens
	}
	if claudeAdaptiveOnly(model) {
		fields := map[string]any{keyThinking: map[string]any{wire.KeyType: "adaptive"}}
		if effort != "" {
			fields["output_config"] = map[string]any{keyEffort: effort}
		}
		maps.Copy(body, fields)
		return maxTokens
	}
	if budget <= 0 {
		budget = defaultThinkingBudget
		if effort == string(llmprovider.EffortLow) {
			budget = wire.LowEffortThinkingBudget
		}
	}
	if maxTokens <= budget {
		maxTokens = budget + defaultThinkingBudget
	}
	body[keyThinking] = map[string]any{wire.KeyType: keyEnabled, "budget_tokens": budget}
	return maxTokens
}

// usage is the Messages API's token counts. input_tokens leaves out the cache
// reads and writes, so the input count adds them; the API reports no
// reasoning count.
type usage struct {
	Input      int `json:"input_tokens"`
	Output     int `json:"output_tokens"`
	CacheRead  int `json:"cache_read_input_tokens"`
	CacheWrite int `json:"cache_creation_input_tokens"`
}

func (u usage) counts() llmprovider.Usage {
	return llmprovider.Usage{InputTokens: u.Input + u.CacheRead + u.CacheWrite, OutputTokens: u.Output,
		CachedTokens: u.CacheRead}
}
