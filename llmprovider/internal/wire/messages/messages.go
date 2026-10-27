// Package messages is the Anthropic Messages wire, shared by the claude
// provider and OpenCode's messages route (0015-MADR D2), with its thinking
// request shape.
package messages

import (
	"encoding/base64"
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
	blockImage            = "image"
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
const defaultThinkingBudget = wire.DefaultThinkingBudget

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
// redacted_thinking, tool_use, tool_result, tool_reference and image. A field
// a block does not have is nil, and omitted; Type is always sent.
type Block struct {
	Content   ResultContent `json:"content,omitzero"`
	Data      *string       `json:"data,omitempty"`
	ID        *string       `json:"id,omitempty"`
	Input     any           `json:"input,omitempty"`
	IsError   bool          `json:"is_error,omitempty"`
	Name      *string       `json:"name,omitempty"`
	Signature *string       `json:"signature,omitempty"`
	Source    *ImageSource  `json:"source,omitempty"`
	Text      *string       `json:"text,omitempty"`
	Thinking  *string       `json:"thinking,omitempty"`
	ToolName  *string       `json:"tool_name,omitempty"`
	ToolUseID *string       `json:"tool_use_id,omitempty"`
	Type      string        `json:"type"`
}

// ImageSource is an image block's source: base64 data with its media type,
// or a URL the service fetches (0061-MADR D5). A field a source does not have
// is nil, and omitted.
type ImageSource struct {
	Data      *string `json:"data,omitempty"`
	MediaType *string `json:"media_type,omitempty"`
	Type      string  `json:"type"`
	URL       *string `json:"url,omitempty"`
}

// imageSource is img's source: its bytes base64-encoded, or its URL.
func imageSource(img llmprovider.ImageItem, arena *wire.Arena) *ImageSource {
	if img.URL != "" {
		return &ImageSource{Type: "url", URL: arena.Keep(img.URL)}
	}
	return &ImageSource{Type: "base64", MediaType: arena.Keep(img.MediaType),
		Data: arena.Keep(base64.StdEncoding.EncodeToString(img.Data))}
}

// ResultContent is a tool_result's content: its text, as every result had
// before 0040, or blocks, the tool_reference blocks of a result that loads
// deferred tools (0040-MADR D2, A1). The zero value is omitted. Like Content,
// it is not an interface, so encoding it allocates nothing (0028-PLAN D16).
type ResultContent struct {
	text   *string
	blocks *[]Block
}

// IsZero reports a block with no content, which omitzero leaves out.
func (c ResultContent) IsZero() bool { return c.text == nil && c.blocks == nil }

// Text is the result's text, or "" when it holds blocks or nothing.
func (c ResultContent) Text() string {
	if c.text == nil {
		return ""
	}
	return *c.text
}

// MarshalJSONTo writes the blocks as a list, or the text as a string.
func (c ResultContent) MarshalJSONTo(enc *jsontext.Encoder) error {
	if c.blocks != nil {
		return jsonv2.MarshalEncode(enc, *c.blocks)
	}
	if c.text == nil {
		return enc.WriteToken(jsontext.Null)
	}
	return enc.WriteToken(jsontext.String(*c.text))
}

// MarshalJSON is MarshalJSONTo for an encoder that does not call it.
func (c ResultContent) MarshalJSON() ([]byte, error) {
	if c.blocks != nil {
		return json.Marshal(*c.blocks)
	}
	if c.text == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*c.text)
}

// FromItems converts items to Anthropic messages. System items are left out:
// they go to the top-level system field (see wire.SystemPrompt). Call ids
// another wire made are sent in a form Anthropic accepts, unique in the
// request, and each result keeps its call (0026-MADR F8).
//
// The strings the blocks carry are kept in one arena, and every message's
// blocks in one backing array: only the last message ever takes a block, so
// each message's blocks are contiguous (0028-PLAN D16).
//
// FromItems sends no ToolLoadItem: a request whose form appends its loaded
// tools has nothing to say of them in the messages (0040-MADR D6).
func FromItems(items []llmprovider.Item) []Message { return encodeItems(items, false) }

// FromItemsLoading is FromItems for a request that defers tools natively
// (0040-MADR D2, A1). A result that ToolLoadItems follow holds one
// tool_reference per loaded tool, and its text goes after the turn's
// results, as its own text block: Anthropic refuses text beside a
// tool_reference in one tool_result. With no load, it is FromItems.
func FromItemsLoading(items []llmprovider.Item) []Message { return encodeItems(items, true) }

// encodeItems is FromItems, and with loading, FromItemsLoading.
func encodeItems(items []llmprovider.Item, loading bool) []Message {
	items = wire.RemapCallIDs(items, wire.AnthropicCallIDs)
	var loads map[string][]string // a result's call id → the tools it loads
	if loading {
		for _, item := range items {
			if load, ok := item.(llmprovider.ToolLoadItem); ok {
				if loads == nil {
					loads = map[string][]string{}
				}
				loads[load.CallID] = append(loads[load.CallID], load.Name)
			}
		}
	}
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
	// appendImage adds an image to the user message just before it, whose
	// text becomes a text block, or opens a user message of its own
	// (0061-MADR D5).
	appendImage := func(block Block) {
		if n := len(messages); n > 0 && messages[n-1].Role == wire.RoleUser && messages[n-1].Content.blocks == nil {
			messages[n-1].Content = openList(Block{Type: wire.KeyText, Text: messages[n-1].Content.text}, block)
			return
		}
		appendBlock(wire.RoleUser, block)
	}
	// pending holds the texts of the turn's loading results, which go after
	// all of the turn's results (0040-MADR A1).
	var pending []*string
	flush := func() {
		for _, text := range pending {
			appendBlock(wire.RoleUser, Block{Type: wire.KeyText, Text: text})
		}
		pending = pending[:0]
	}
	for _, item := range items {
		switch item.(type) {
		case llmprovider.FunctionCallOutputItem, llmprovider.ToolLoadItem:
		default:
			flush()
		}
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
		case llmprovider.ImageItem:
			appendImage(Block{Type: blockImage, Source: imageSource(v, arena)})
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
			names := loads[v.CallID]
			if len(names) == 0 {
				appendBlock(wire.RoleUser, Block{Type: "tool_result", ToolUseID: arena.Keep(v.CallID),
					Content: ResultContent{text: arena.Keep(v.Output)}, IsError: v.IsError})
				continue
			}
			refs := make([]Block, len(names))
			for i, name := range names {
				refs[i] = Block{Type: "tool_reference", ToolName: arena.Keep(name)}
			}
			appendBlock(wire.RoleUser, Block{Type: "tool_result", ToolUseID: arena.Keep(v.CallID),
				Content: ResultContent{blocks: &refs}, IsError: v.IsError})
			if v.Output != "" {
				pending = append(pending, arena.Keep(v.Output))
			}
		}
	}
	flush()
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
	// at is the block a content_block event's index names. The index is the
	// service's: checked before it grows the blocks (0053-MADR D2).
	at := func(i int) (*block, *blockText, error) {
		if err := wire.StreamIndex(provider, "content_block", i); err != nil {
			return nil, nil, err
		}
		for len(m.Content) <= i {
			m.Content = append(m.Content, block{})
			texts = append(texts, &blockText{})
		}
		return &m.Content[i], texts[i], nil
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
			b, _, err := at(ev.Index)
			if err != nil {
				return false, err
			}
			*b = ev.ContentBlock
		case "content_block_delta":
			_, t, err := at(ev.Index)
			if err != nil {
				return false, err
			}
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
			b, t, err := at(ev.Index)
			if err != nil {
				return false, err
			}
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
				{&m.Usage.CacheRead, &ev.Usage.CacheRead}, {&m.Usage.CacheWrite, &ev.Usage.CacheWrite},
				{&m.Usage.CacheCreation.Ephemeral1h, &ev.Usage.CacheCreation.Ephemeral1h},
				{&m.Usage.OutputDetails.Thinking, &ev.Usage.OutputDetails.Thinking}} {
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

// A Claude id's version, which deferred tools read (0040-MADR D2). The
// thinking shape no longer does: 0062-MADR D8 removed R1.

// claudeVersionRE reads a Claude version, family-first (claude-opus-4-8) or
// version-first (claude-4.7-opus). A minor has at most two digits, so the date
// in claude-sonnet-4-20250514 is not read as one. OpenCode's
// anthropicUsesModernAdaptiveThinking uses the same pattern.
var claudeVersionRE = regexp.MustCompile(`(?i)claude-(?:[a-z]+-)?(\d+)(?:[.-](\d{1,2}))?(?:[.@-]|$)`)

// ClaudeVersion reads a Claude id's major and minor version, such as 4 and 5
// in claude-haiku-4-5 or 5 and 0 in claude-sonnet-5, by claudeVersionRE. It
// reports false for an id without "claude-" or with no readable version.
func ClaudeVersion(model string) (major, minor int, ok bool) {
	lower := strings.ToLower(model)
	if !strings.Contains(lower, "claude-") {
		return 0, 0, false
	}
	m := claudeVersionRE.FindStringSubmatch(lower)
	if m == nil {
		return 0, 0, false
	}
	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	if m[2] != "" {
		if minor, err = strconv.Atoi(m[2]); err != nil {
			return 0, 0, false
		}
	}
	return major, minor, true
}

// AddThinking adds the thinking fields of an Anthropic Messages request to
// body, in D2's shape for a request with no facts, and returns its max_tokens:
// AddThinkingFor with no facts and PreferredShape's shape (0062-MADR D2).
func AddThinking(body map[string]any, effort string, budget, maxTokens int) int {
	return AddThinkingFor(body, effort, budget, maxTokens, Thinking{}, PreferredShape(Thinking{}, budget))
}

// Thinking is what a model's facts say of its Messages thinking (0041-MADR
// D2, R3), resolved by the provider so that this package reads no facts.
type Thinking struct {
	// Known reports that the facts answer. Without it the request decides
	// the shape (0062-MADR D2).
	Known bool
	// Adaptive is thinking.type "adaptive", not an enabled budget.
	// EffortOnly, with it, sends the effort as output_config.effort.
	Adaptive, EffortOnly bool
	// BudgetMin and BudgetMax bound an enabled budget; zero is unstated.
	BudgetMin, BudgetMax int
	// MaxOutput is the model's output limit, which a budget's raise of
	// max_tokens never passes; zero is unknown.
	MaxOutput int
}

// FactsThinking is the Thinking a model's reasoning facts give: a budget
// option means an enabled budget within its bounds; efforts without one mean
// adaptive thinking with the effort; a toggle alone means adaptive thinking
// with none. With none of them the facts do not answer. maxOutput is the
// model's output limit, or zero.
func FactsThinking(toggle bool, efforts []string, budget bool, budgetMin, budgetMax, maxOutput int) Thinking {
	switch {
	case budget:
		return Thinking{Known: true, BudgetMin: budgetMin, BudgetMax: budgetMax, MaxOutput: maxOutput}
	case len(efforts) > 0:
		return Thinking{Known: true, Adaptive: true, EffortOnly: true, MaxOutput: maxOutput}
	case toggle:
		return Thinking{Known: true, Adaptive: true, MaxOutput: maxOutput}
	}
	return Thinking{MaxOutput: maxOutput}
}

// PreferredShape is D2's thinking shape: the facts', when they answer; else a
// budget's, when the request sets one; else adaptive thinking (0062-MADR D2).
// D3's ladder may then send the other, when the model refused this one.
func PreferredShape(facts Thinking, budget int) llmprovider.ThinkingShape {
	switch {
	case facts.Known && facts.Adaptive, !facts.Known && budget <= 0:
		return llmprovider.ThinkingAdaptive
	default:
		return llmprovider.ThinkingBudget
	}
}

// AddThinkingFor adds the thinking fields of shape to body, with the model's
// facts (0041-MADR D2; 0062-MADR D3), and returns its max_tokens. Adaptive
// thinking is thinking.type "adaptive", with output_config.effort when an
// effort is set, unless the facts name adaptive thinking without one; a
// budget does not apply to it. The budget shape is an enabled budget: the
// request's, else wire.LowEffortThinkingBudget for "low", else
// defaultThinkingBudget (R3), held to the facts' bounds and below the output
// limit, with max_tokens raised above it when needed (Anthropic requires
// max_tokens > budget_tokens).
func AddThinkingFor(body map[string]any, effort string, budget, maxTokens int, facts Thinking,
	shape llmprovider.ThinkingShape) int {
	if shape == llmprovider.ThinkingAdaptive {
		fields := map[string]any{keyThinking: map[string]any{wire.KeyType: "adaptive"}}
		toggleOnly := facts.Known && facts.Adaptive && !facts.EffortOnly
		if !toggleOnly && effort != "" {
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
	if facts.BudgetMax > 0 {
		budget = min(budget, facts.BudgetMax)
	}
	budget = max(budget, facts.BudgetMin)
	if maxTokens <= budget {
		maxTokens = budget + defaultThinkingBudget
	}
	if facts.MaxOutput > 0 && maxTokens > facts.MaxOutput {
		// The raise stops at the limit, and the budget stays below it, as
		// Anthropic requires.
		maxTokens = facts.MaxOutput
		budget = min(budget, maxTokens-1)
	}
	body[keyThinking] = map[string]any{wire.KeyType: keyEnabled, "budget_tokens": budget}
	return maxTokens
}

// usage is the Messages API's token counts. input_tokens leaves out the cache
// reads and writes, so the input count adds them. cache_creation splits the
// writes by lifetime, and its one-hour count is CacheWrite1hTokens (0036-MADR
// D2). output_tokens_details.thinking_tokens is the part of output_tokens
// that was reasoning; a stream sends it on its last message_delta only
// (0037-MADR D1).
type usage struct {
	Input         int `json:"input_tokens"`
	Output        int `json:"output_tokens"`
	CacheRead     int `json:"cache_read_input_tokens"`
	CacheWrite    int `json:"cache_creation_input_tokens"`
	CacheCreation struct {
		Ephemeral1h int `json:"ephemeral_1h_input_tokens"`
	} `json:"cache_creation"`
	OutputDetails struct {
		Thinking int `json:"thinking_tokens"`
	} `json:"output_tokens_details"`
}

func (u usage) counts() llmprovider.Usage {
	return llmprovider.Usage{InputTokens: u.Input + u.CacheRead + u.CacheWrite, OutputTokens: u.Output,
		ReasoningTokens: u.OutputDetails.Thinking, CachedTokens: u.CacheRead, CacheWriteTokens: u.CacheWrite,
		CacheWrite1hTokens: u.CacheCreation.Ephemeral1h}
}
