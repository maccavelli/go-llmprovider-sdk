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
	var m message
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		return nil, err
	}
	return m.response()
}

// message is a Messages API reply, whole, or assembled from its stream.
type message struct {
	Model      string  `json:"model"`
	StopReason string  `json:"stop_reason"`
	Content    []block `json:"content"`
	Usage      usage   `json:"usage"`
}

// block is one content block.
type block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Signature string          `json:"signature"`
	Data      string          `json:"data"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
}

// response is m as a Response, or the error its stop reason and content
// make it.
func (m message) response() (*llmprovider.Response, error) {
	hasCall := false
	for _, b := range m.Content {
		hasCall = hasCall || b.Type == blockToolUse
	}
	finish := wire.Finish(m.StopReason, stopReasons, hasCall)
	if len(m.Content) == 0 {
		return nil, wire.EmptyAnswer("messages", finish)
	}
	if finish == llmprovider.FinishLength && hasCall {
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}

	// The Messages API is stateless: no ID.
	res := &llmprovider.Response{Model: m.Model, FinishReason: finish, Usage: m.Usage.counts()}
	for _, b := range m.Content {
		item, ok, err := b.item()
		if err != nil {
			return nil, err
		}
		if ok {
			res.Output = append(res.Output, item)
		}
	}

	// An answer cut while it was still thinking has nothing usable
	// (0026-MADR F7); a cut text answer keeps its text.
	if len(res.Output) == 0 || finish == llmprovider.FinishLength && res.OutputText() == "" {
		return nil, wire.EmptyAnswer("messages", finish)
	}

	return res, nil
}

// item is the item b holds, and false for none: an empty text, or a block of
// a type this wire does not read.
func (b block) item() (llmprovider.Item, bool, error) {
	switch b.Type {
	case keyThinking:
		text := b.Thinking
		if text == "" {
			text = b.Text
		}
		return llmprovider.ReasoningItem{Text: text, Signature: b.Signature, Format: wire.FormatMessages}, true, nil
	case blockRedactedThinking:
		return llmprovider.ReasoningItem{Encrypted: b.Data, Format: wire.FormatMessages}, true, nil
	case wire.KeyText, "":
		if b.Text == "" {
			return nil, false, nil
		}
		return llmprovider.MessageItem{Role: wire.RoleAssistant, Text: b.Text}, true, nil
	case blockToolUse:
		argsStr, err := wire.CompactArguments(b.Input)
		if err != nil {
			return nil, false, fmt.Errorf("messages: tool_use input: %w", err)
		}
		return llmprovider.FunctionCallItem{CallID: b.ID, Name: b.Name, Arguments: argsStr}, true, nil
	}
	return nil, false, nil
}

// streamEvent is the part of one Messages stream event the reader uses
// (0031-PLAN-native-streaming-on-the-other-wires, Phase 1).
type streamEvent struct {
	Type    string `json:"type"`
	Index   int    `json:"index"`
	Message struct {
		Model string `json:"model"`
		Usage usage  `json:"usage"`
	} `json:"message"`
	ContentBlock block `json:"content_block"`
	Delta        struct {
		Type        string `json:"type"`
		Text        string `json:"text"`
		Thinking    string `json:"thinking"`
		Signature   string `json:"signature"`
		PartialJSON string `json:"partial_json"`
		StopReason  string `json:"stop_reason"`
	} `json:"delta"`
	Usage usage `json:"usage"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// blockText is a block's strings as its deltas arrive.
type blockText struct {
	text, thinking, signature, args strings.Builder
}

// Events reads a Messages API event stream and yields its events as they
// arrive (0031-MADR E3): a text_delta as EventTextDelta, a thinking_delta as
// EventReasoningDelta, each content block as EventItem when it stops, and
// message_stop as EventDone, carrying the Response Decode returns for the
// same answer. Signatures, a call's input_json_delta and a redacted block are
// assembled into their items, not yielded. The stream's error event is
// classified by its type. It returns nil once EventDone is yielded or yield
// returns false, and otherwise the error that ended the stream; what it
// yielded before an error is not a response (0031-MADR D4).
func Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	var m message
	var texts []*blockText
	at := func(i int) (*block, *blockText) {
		for len(m.Content) <= i {
			m.Content = append(m.Content, block{})
			texts = append(texts, &blockText{})
		}
		return &m.Content[i], texts[i]
	}
	dispatch := func(payload []byte) (bool, error) {
		var ev streamEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			return false, fmt.Errorf("%s: decode stream event: %w", provider, err)
		}
		switch ev.Type {
		case "message_start":
			m.Model, m.Usage = ev.Message.Model, ev.Message.Usage
		case "content_block_start":
			b, _ := at(ev.Index)
			*b = ev.ContentBlock
		case "content_block_delta":
			_, t := at(ev.Index)
			switch ev.Delta.Type {
			case "text_delta":
				t.text.WriteString(ev.Delta.Text)
				return !yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: ev.Delta.Text}), nil
			case "thinking_delta":
				t.thinking.WriteString(ev.Delta.Thinking)
				return !yield(llmprovider.Event{Type: llmprovider.EventReasoningDelta, Text: ev.Delta.Thinking}), nil
			case "signature_delta":
				t.signature.WriteString(ev.Delta.Signature)
			case "input_json_delta":
				t.args.WriteString(ev.Delta.PartialJSON)
			}
		case "content_block_stop":
			b, t := at(ev.Index)
			b.Text += t.text.String()
			b.Thinking += t.thinking.String()
			b.Signature += t.signature.String()
			if t.args.Len() > 0 {
				b.Input = json.RawMessage(t.args.String())
			}
			// A call cut short has no usable input: its error is the one
			// response gives at message_stop, as Decode's.
			if item, ok, err := b.item(); err == nil && ok {
				return !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: item}), nil
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				m.StopReason = ev.Delta.StopReason
			}
			// The counts are cumulative; one the event leaves out keeps
			// message_start's.
			for _, f := range []struct{ to, from *int }{{&m.Usage.Input, &ev.Usage.Input}, {&m.Usage.Output, &ev.Usage.Output},
				{&m.Usage.CacheRead, &ev.Usage.CacheRead}, {&m.Usage.CacheWrite, &ev.Usage.CacheWrite}} {
				if *f.from > 0 {
					*f.to = *f.from
				}
			}
		case "message_stop":
			resp, err := m.response()
			if err != nil {
				return false, err
			}
			yield(llmprovider.Event{Type: llmprovider.EventDone, Response: resp})
			return true, nil
		case "error":
			return false, llmprovider.ClassifyStreamFailure(provider, "", ev.Error.Type, ev.Error.Message)
		}
		return false, nil
	}
	return wire.ReadEvents(provider, body, "message_stop", dispatch)
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
