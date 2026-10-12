// Package chatcompletions is the OpenAI Chat Completions wire, shared by
// Hugging Face, Kilo, Together, Ollama and OpenCode's chat route (0015-MADR
// D2).
package chatcompletions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// Chat Completions field names only this format uses.
const (
	keyModel           = "model"
	keyMessages        = "messages"
	keyMaxTokens       = "max_tokens"
	keyTools           = "tools"
	keyToolChoice      = "tool_choice"
	keyToolCalls       = "tool_calls"
	keyFunction        = "function"
	keyDescription     = "description"
	keyParameters      = "parameters"
	keyReasoningEffort = "reasoning_effort"
	keyReasoning       = "reasoning"
	keyReasoningDetail = "reasoning_details"
	keyToolCallID      = "tool_call_id"
	roleTool           = "tool"
)

// Opts carries the per-gateway variations of a Chat Completions request. Zero
// values omit the corresponding field entirely.
type Opts struct {
	// Tools are offered to the model as functions.
	Tools []llmprovider.Tool
	// ToolChoice is sent as tool_choice: a named tool as an object,
	// "required" and "none" as strings, and auto as nothing.
	ToolChoice llmprovider.ToolChoice
	// NoToolChoice is for a service or model that does not take tool_choice:
	// none is sent, and ToolChoiceNone is kept by sending no tools, so the
	// model cannot call one (0020-MADR F40). Ollama never takes it; Kilo
	// gates it on the model's supported_parameters.
	NoToolChoice bool
	// ReasoningEffort, when non-empty, is sent as reasoning_effort. OpenCode's
	// chat route sets it only when the model's published reasoning_options
	// list the configured effort (MADR 0009 §6).
	ReasoningEffort string
	// Reasoning, when non-nil, is sent as the OpenRouter-style reasoning
	// object Kilo reads: {"effort": …} or {"enabled": true}.
	Reasoning map[string]any
	// ReplayReasoningField, when non-empty, replays prior reasoning on every
	// assistant message under this field (OpenCode interleaved models).
	ReplayReasoningField string
	// ReplayReasoningDetails replays the reasoning_details entries this wire
	// decoded, as they came, on the assistant message they precede, as
	// OpenRouter-style gateways need them back: Kilo (0021-MADR W9).
	ReplayReasoningDetails bool
	// SendStrict sends each tool's Strict, when set, as function.strict: for
	// a gateway measured accepting it (0032-MADR D2, Q2).
	SendStrict bool
}

// chatMessage is one Chat Completions message. Its fields are in the order
// json.Marshal wrote the map it replaces, so the JSON is the same byte for
// byte (0028-MADR D-A6). A replayed reasoning field, named by the model's
// metadata, is not a field: replayField and replay hold it, and
// chatMessages.MarshalJSON writes it.
type chatMessage struct {
	Content          string            `json:"content"`
	ReasoningDetails []json.RawMessage `json:"reasoning_details,omitempty"`
	Role             string            `json:"role"`
	ToolCallID       *string           `json:"tool_call_id,omitempty"`
	ToolCalls        []chatToolCall    `json:"tool_calls,omitempty"`

	// replayField is the key the replay is sent under, and replaying says the
	// message carries it. When the key is reasoning_details or tool_calls, the
	// replay stands in that field's place, as it did in the map.
	replayField string
	replay      string
	replaying   bool
}

// chatToolCall is one tool_calls entry.
type chatToolCall struct {
	Function chatCallFunction `json:"function"`
	ID       string           `json:"id"`
	Type     string           `json:"type"`
}

// chatCallFunction is a tool call's function.
type chatCallFunction struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

// chatTool is one tools entry.
type chatTool struct {
	Function chatToolFunction `json:"function"`
	Type     string           `json:"type"`
}

// chatToolFunction is a tool's function.
type chatToolFunction struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Parameters  any    `json:"parameters"`
	Strict      *bool  `json:"strict,omitempty"`
}

// replayString is the string the message holds under key, as the map's
// lookup m[key].(string) read it: "" when the key is absent or holds no string.
func (m *chatMessage) replayString(key string) string {
	switch key {
	case wire.KeyContent:
		return m.Content
	case wire.KeyRole:
		return m.Role
	case keyToolCallID:
		if m.ToolCallID != nil {
			return *m.ToolCallID
		}
		return ""
	}
	if m.replaying && m.replayField == key {
		return m.replay
	}
	return ""
}

// setReplay sets the string under key, as the map's m[key] = v did.
func (m *chatMessage) setReplay(key, v string) {
	switch key {
	case wire.KeyContent:
		m.Content = v
		return
	case wire.KeyRole:
		m.Role = v
		return
	case keyToolCallID:
		m.ToolCallID = &v
		return
	}
	// Under tool_calls or reasoning_details, the replay hides that field:
	// toolCalls, details and replayJSON read the replay in its place.
	m.replayField, m.replay, m.replaying = key, v, true
}

// toolCalls is the message's calls, as the map's m["tool_calls"] read them:
// none while a replay stands in their place.
func (m *chatMessage) toolCalls() []chatToolCall {
	if m.replaying && m.replayField == keyToolCalls {
		return nil
	}
	return m.ToolCalls
}

// setToolCalls sets the calls, replacing a replay that stood in their place.
func (m *chatMessage) setToolCalls(calls []chatToolCall) {
	if m.replaying && m.replayField == keyToolCalls {
		m.replaying = false
	}
	m.ToolCalls = calls
}

// details is the message's reasoning_details, as the map read them.
func (m *chatMessage) details() []json.RawMessage {
	if m.replaying && m.replayField == keyReasoningDetail {
		return nil
	}
	return m.ReasoningDetails
}

// setDetails sets reasoning_details, replacing a replay in their place.
func (m *chatMessage) setDetails(details []json.RawMessage) {
	if m.replaying && m.replayField == keyReasoningDetail {
		m.replaying = false
	}
	m.ReasoningDetails = details
}

// chatMessages is a message list in which some message carries a replay; its
// MarshalJSON writes each such message's keys in sorted order, as the map
// did. A list with no replay is sent as []chatMessage, encoded field by field.
type chatMessages []chatMessage

// MarshalJSON writes the messages, a replay under its key in sorted order.
func (ms chatMessages) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('[')
	for i := range ms {
		if i > 0 {
			b.WriteByte(',')
		}
		var (
			raw []byte
			err error
		)
		if ms[i].replaying {
			raw, err = ms[i].replayJSON()
		} else {
			raw, err = json.Marshal(ms[i])
		}
		if err != nil {
			return nil, err
		}
		b.Write(raw)
	}
	b.WriteByte(']')
	return b.Bytes(), nil
}

// replayJSON writes one message with its replay, keys sorted.
func (m chatMessage) replayJSON() ([]byte, error) {
	fields := map[string]any{wire.KeyContent: m.Content, wire.KeyRole: m.Role, m.replayField: m.replay}
	if len(m.ReasoningDetails) > 0 && m.replayField != keyReasoningDetail {
		fields[keyReasoningDetail] = m.ReasoningDetails
	}
	if m.ToolCallID != nil {
		fields[keyToolCallID] = *m.ToolCallID
	}
	if len(m.ToolCalls) > 0 && m.replayField != keyToolCalls {
		fields[keyToolCalls] = m.ToolCalls
	}
	return json.Marshal(fields)
}

// chatMessagesValue is msgs as a body's messages: the slice itself, unless a
// message carries a replay.
func chatMessagesValue(msgs []chatMessage) any {
	for i := range msgs {
		if msgs[i].replaying {
			return chatMessages(msgs)
		}
	}
	return msgs
}

// itemsToChatMessages converts canonical items to OpenAI Chat Completions
// messages. A function call becomes the assistant turn's tool_calls entry,
// and its result a role:"tool" message keyed by tool_call_id, the Chat
// Completions equivalent of the Responses API's function_call_output item.
func itemsToChatMessages(items []llmprovider.Item) []chatMessage {
	return itemsToChatMessagesReplaying(items, "", false)
}

// itemsToChatMessagesReplaying is itemsToChatMessages that, when field is set,
// puts the reasoning preceding each assistant message into that field, and
// sets it (possibly "") on every assistant message, as OpenCode's client does
// for interleaved models (MADR 0012 §2, O5). With details, the
// reasoning_details entries this wire decoded go back on the assistant
// message they precede (0021-MADR W9).
func itemsToChatMessagesReplaying(items []llmprovider.Item, field string, details bool) []chatMessage {
	messages := make([]chatMessage, 0, len(items))
	var pending strings.Builder
	var pendingDetails []json.RawMessage
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.ReasoningItem:
			pending.WriteString(v.Text)
			if detail := strings.TrimSpace(v.Encrypted); details && v.Format == wire.FormatChatCompletions &&
				strings.HasPrefix(detail, "{") && json.Valid([]byte(detail)) {
				pendingDetails = append(pendingDetails, json.RawMessage(detail))
			}
		case llmprovider.MessageItem:
			role := string(v.Role)
			if role == "" {
				role = wire.RoleUser
			}
			// Reasoning no assistant message followed, such as an answer
			// cut while reasoning, ends with its turn (0026-MADR F32).
			if role == wire.RoleUser {
				pending.Reset()
				pendingDetails = nil
			}
			messages = append(messages, chatMessage{Role: role, Content: v.Text})
		case llmprovider.FunctionCallItem:
			call := chatToolCall{ID: v.CallID, Type: keyFunction,
				Function: chatCallFunction{Name: v.Name, Arguments: v.Arguments}}
			// A call joins the assistant turn it follows (its text, or the
			// calls before it); otherwise it opens one (MADR 0012 §2).
			if n := len(messages); n > 0 && messages[n-1].Role == wire.RoleAssistant {
				last := &messages[n-1]
				last.setToolCalls(append(last.toolCalls(), call))
				continue
			}
			messages = append(messages, chatMessage{Role: wire.RoleAssistant, ToolCalls: []chatToolCall{call}})
		case llmprovider.FunctionCallOutputItem:
			id := v.CallID
			messages = append(messages, chatMessage{Role: roleTool, ToolCallID: &id, Content: v.Output})
		}
		if field == "" && !details {
			continue
		}
		// The reasoning belongs to the assistant turn it precedes.
		if n := len(messages); n > 0 && messages[n-1].Role == wire.RoleAssistant {
			if _, isReasoning := item.(llmprovider.ReasoningItem); !isReasoning {
				last := &messages[n-1]
				if field != "" {
					last.setReplay(field, last.replayString(field)+pending.String())
				}
				pending.Reset()
				if len(pendingDetails) > 0 {
					last.setDetails(append(last.details(), pendingDetails...))
					pendingDetails = nil
				}
			}
		}
	}
	if len(messages) == 0 {
		return nil // sent as null, as the map encoder sent it
	}
	return messages
}

// toolList is tools as Chat Completions functions.
func toolList(tools []llmprovider.Tool, sendStrict bool) []chatTool {
	list := make([]chatTool, len(tools))
	for i, tool := range tools {
		list[i] = chatTool{Type: keyFunction, Function: chatToolFunction{
			Name: tool.Name, Description: tool.Description, Parameters: wire.ToolSchema(tool.Schema)}}
		if sendStrict {
			list[i].Function.Strict = tool.Strict
		}
	}
	return list
}

// toolChoice is choice as tool_choice, or nil for auto.
func toolChoice(choice llmprovider.ToolChoice) any {
	if name, forced := choice.Tool(); forced {
		return map[string]any{wire.KeyType: keyFunction, keyFunction: map[string]any{wire.KeyName: name}}
	}
	if choice == llmprovider.ToolChoiceRequired || choice == llmprovider.ToolChoiceNone {
		return string(choice)
	}
	return nil
}

// Body builds an OpenAI Chat Completions request body.
func Body(model string, maxTokens int, input []llmprovider.Item, o Opts) map[string]any {
	body := map[string]any{
		keyModel:     model,
		keyMessages:  chatMessagesValue(itemsToChatMessagesReplaying(input, o.ReplayReasoningField, o.ReplayReasoningDetails)),
		keyMaxTokens: maxTokens,
	}
	if len(o.Tools) > 0 && (!o.NoToolChoice || o.ToolChoice != llmprovider.ToolChoiceNone) {
		body[keyTools] = toolList(o.Tools, o.SendStrict)
		if choice := toolChoice(o.ToolChoice); choice != nil && !o.NoToolChoice {
			body[keyToolChoice] = choice
		}
	}
	if o.ReasoningEffort != "" {
		body[keyReasoningEffort] = o.ReasoningEffort
	}
	if o.Reasoning != nil {
		body[keyReasoning] = o.Reasoning
	}
	return body
}

// Decode decodes an OpenAI Chat Completions envelope into a canonical
// Response.
//
// Reasoning has two competing vendor spellings, both undocumented, both measured
// 2026-08-28/29:
//
//	message.reasoning_content  — OpenCode Zen (verified on big-pickle)
//	message.reasoning          — Kilo Gateway, the OpenRouter convention
//
// Both are accepted; reasoning_content wins when both are present. Kilo also
// sends message.reasoning_details[], OpenRouter's structured form of the same
// trace, which carries the signatures some models need back (measured on
// google/gemini-3.8-flash, 2026-10-04: {type:"reasoning.text", format, index,
// text}). When present, each entry is one ReasoningItem: its text, the entry
// itself in Encrypted, compacted, to be replayed as it came, and Format
// "chatcompletions"; the plain string is then not decoded as well, so the
// trace has one source (0021-MADR W9).
//
// Absent reasoning is normal, never an error.
func Decode(body io.Reader) (*llmprovider.Response, error) {
	return decodeAs(body, "chat completions")
}

// DecodeFor is Decode for provider: an error reported inside a 200 names
// provider, and is classified by that service's error vocabulary (0026-MADR
// F5). provider is the label the caller gives wire.Call.
func DecodeFor(provider string) func(io.Reader) (*llmprovider.Response, error) {
	return func(body io.Reader) (*llmprovider.Response, error) { return decodeAs(body, provider) }
}

// decodeAs is Decode, with errors labelled and classified for provider.
func decodeAs(body io.Reader, provider string) (*llmprovider.Response, error) {
	var raw chatReply
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw.response(provider)
}

// chatReply is a Chat Completions reply, whole, or assembled from its stream.
type chatReply struct {
	ID      string        `json:"id"`
	Model   string        `json:"model"`
	Error   *gatewayError `json:"error"`
	Choices []chatChoice  `json:"choices"`
	Usage   usage         `json:"usage"`
}

// chatChoice is one choice of a reply.
type chatChoice struct {
	FinishReason string        `json:"finish_reason"`
	Error        *gatewayError `json:"error"`
	Message      replyMessage  `json:"message"`
}

// replyMessage is a choice's message.
type replyMessage struct {
	Role             string            `json:"role"`
	Content          chatContent       `json:"content"`
	Refusal          string            `json:"refusal"`
	ReasoningContent string            `json:"reasoning_content"`
	Reasoning        string            `json:"reasoning"`
	ReasoningDetails []json.RawMessage `json:"reasoning_details"`
	ToolCalls        []replyCall       `json:"tool_calls"`
}

// replyCall is one of a message's calls.
type replyCall struct {
	ID       string `json:"id"`
	Function struct {
		Name      string        `json:"name"`
		Arguments chatArguments `json:"arguments"`
	} `json:"function"`
}

// response is raw as a Response, or the error it carries or makes, for
// provider.
func (raw chatReply) response(provider string) (*llmprovider.Response, error) {
	// A gateway may answer 200 with an error, at the top level or in the
	// choice, as OpenRouter does, which Kilo inherits (0021-MADR W5).
	if raw.Error != nil {
		return nil, raw.Error.classify(provider)
	}
	if len(raw.Choices) == 0 {
		return nil, wire.EmptyAnswer("chat completions", "")
	}
	if e := raw.Choices[0].Error; e != nil || raw.Choices[0].FinishReason == finishError {
		if e == nil {
			e = &gatewayError{Message: "finish_reason error"}
		}
		return nil, e.classify(provider)
	}

	msg := raw.Choices[0].Message
	finish := wire.Finish(raw.Choices[0].FinishReason, nil, len(msg.ToolCalls) > 0)
	// A tool call cut by the token limit has unusable arguments (MADR 0012 §1.5).
	if finish == llmprovider.FinishLength && len(msg.ToolCalls) > 0 {
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: string(llmprovider.FinishLength)}
	}
	// A refusal is the answer's text, flagged content_filter, as on the
	// Responses wire (0021-MADR W4; 0026-MADR F29).
	text := string(msg.Content)
	if text == "" && msg.Refusal != "" {
		text, finish = msg.Refusal, llmprovider.FinishContentFilter
	}
	// An answer cut while it was still reasoning has nothing usable
	// (0026-MADR F7).
	if finish == llmprovider.FinishLength && text == "" {
		return nil, wire.EmptyAnswer("chat completions", finish)
	}
	// The response id is not a resumable conversation handle on any gateway
	// that speaks this format, so it is carried for logging only.
	result := &llmprovider.Response{ID: raw.ID, Model: raw.Model, FinishReason: finish, Usage: raw.Usage.counts()}

	if len(msg.ReasoningDetails) > 0 {
		for _, detail := range msg.ReasoningDetails {
			item, err := reasoningDetail(detail)
			if err != nil {
				return nil, err
			}
			result.Output = append(result.Output, item)
		}
	} else {
		reasoning := msg.ReasoningContent
		if reasoning == "" {
			reasoning = msg.Reasoning
		}
		if strings.TrimSpace(reasoning) != "" {
			result.Output = append(result.Output, llmprovider.ReasoningItem{Text: reasoning, Format: wire.FormatChatCompletions})
		}
	}
	if text != "" {
		// An answer is the assistant's, whatever role a gateway spells: a
		// verbatim "Assistant" would fail the next turn's validation.
		result.Output = append(result.Output, llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: text})
	}
	for _, tc := range msg.ToolCalls {
		result.Output = append(result.Output, llmprovider.FunctionCallItem{
			CallID:    tc.ID,
			Name:      tc.Function.Name,
			Arguments: string(tc.Function.Arguments),
		})
	}

	if len(result.Output) == 0 {
		return nil, wire.EmptyAnswer("chat completions", finish)
	}
	return result, nil
}

// chatChunk is the part of one Chat Completions stream chunk the reader uses
// (0031-PLAN-native-streaming-on-the-other-wires, Phase 1).
type chatChunk struct {
	ID      string        `json:"id"`
	Model   string        `json:"model"`
	Error   *gatewayError `json:"error"`
	Choices []struct {
		FinishReason string        `json:"finish_reason"`
		Error        *gatewayError `json:"error"`
		Delta        chunkDelta    `json:"delta"`
	} `json:"choices"`
	Usage *usage `json:"usage"`
}

// chunkDelta is a chunk's choice delta.
type chunkDelta struct {
	Content          chatContent       `json:"content"`
	Refusal          string            `json:"refusal"`
	ReasoningContent string            `json:"reasoning_content"`
	Reasoning        string            `json:"reasoning"`
	ReasoningDetails []json.RawMessage `json:"reasoning_details"`
	ToolCalls        []struct {
		Index    int    `json:"index"`
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
}

// streamCall is a call as its fragments arrive.
type streamCall struct {
	id, name  string
	arguments strings.Builder
}

// chatStream is a reply as its chunks arrive.
type chatStream struct {
	provider                                      string
	raw                                           chatReply
	content, refusal, reasoningContent, reasoning strings.Builder
	calls                                         []*streamCall
	details                                       []map[string]any
	yield                                         func(llmprovider.Event) bool
}

// Events reads a Chat Completions event stream and yields its events as they
// arrive (0031-MADR E3): delta.content and delta.refusal as EventTextDelta,
// delta.reasoning_content, else delta.reasoning, as EventReasoningDelta, and
// at [DONE] the items Decode returns for the same answer, in its order, then
// EventDone, carrying that Response. Calls, assembled from their fragments by
// index, and reasoning_details, merged by index, arrive only as items. An
// error inside the 200, in a chunk or its choice, is classified as Decode
// classifies it (0021-MADR W5). It returns nil once EventDone is yielded or
// yield returns false, and otherwise the error that ended the stream; what it
// yielded before an error is not a response (0031-MADR D4).
func Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	s := &chatStream{provider: provider, raw: chatReply{Choices: []chatChoice{{}}}, yield: yield}
	return wire.ReadEvents(provider, body, "[DONE]", func(payload []byte) (bool, error) {
		if string(payload) == "[DONE]" {
			return s.done()
		}
		return s.chunk(payload)
	})
}

// done finishes the reply at [DONE]: its items, then EventDone.
func (s *chatStream) done() (bool, error) {
	msg := &s.raw.Choices[0].Message
	msg.Content, msg.Refusal = chatContent(s.content.String()), s.refusal.String()
	msg.ReasoningContent, msg.Reasoning = s.reasoningContent.String(), s.reasoning.String()
	for _, c := range s.calls {
		call := replyCall{ID: c.id}
		call.Function.Name = c.name
		call.Function.Arguments = chatArguments(c.arguments.String())
		if strings.TrimSpace(c.arguments.String()) == "" {
			call.Function.Arguments = "{}"
		}
		msg.ToolCalls = append(msg.ToolCalls, call)
	}
	for _, detail := range s.details {
		entry, err := json.Marshal(detail)
		if err != nil {
			return false, fmt.Errorf("%s: reasoning_details: %w", s.provider, err)
		}
		msg.ReasoningDetails = append(msg.ReasoningDetails, entry)
	}
	resp, err := s.raw.response(s.provider)
	if err != nil {
		return false, err
	}
	for _, item := range resp.Output {
		if !s.yield(llmprovider.Event{Type: llmprovider.EventItem, Item: item}) {
			return true, nil
		}
	}
	s.yield(llmprovider.Event{Type: llmprovider.EventDone, Response: resp})
	return true, nil
}

// chunk reads one chunk: its error, its id, model and usage, and its choice.
func (s *chatStream) chunk(payload []byte) (bool, error) {
	var chunk chatChunk
	if err := json.Unmarshal(payload, &chunk); err != nil {
		return false, fmt.Errorf("%s: decode stream chunk: %w", s.provider, err)
	}
	// A gateway may fail inside the 200, at the top level or in the choice
	// (0021-MADR W5).
	if chunk.Error != nil {
		return false, chunk.Error.classify(s.provider)
	}
	if s.raw.ID == "" {
		s.raw.ID = chunk.ID
	}
	if s.raw.Model == "" {
		s.raw.Model = chunk.Model
	}
	if chunk.Usage != nil {
		s.raw.Usage = *chunk.Usage
	}
	if len(chunk.Choices) == 0 {
		return false, nil
	}
	choice := chunk.Choices[0]
	if e := choice.Error; e != nil || choice.FinishReason == finishError {
		if e == nil {
			e = &gatewayError{Message: "finish_reason error"}
		}
		return false, e.classify(s.provider)
	}
	if choice.FinishReason != "" {
		s.raw.Choices[0].FinishReason = choice.FinishReason
	}
	return s.delta(choice.Delta)
}

// delta adds one delta: calls and reasoning_details are accumulated, and
// reasoning and text are yielded.
func (s *chatStream) delta(delta chunkDelta) (bool, error) {
	for _, tc := range delta.ToolCalls {
		for len(s.calls) <= tc.Index {
			s.calls = append(s.calls, &streamCall{})
		}
		c := s.calls[tc.Index]
		if tc.ID != "" {
			c.id = tc.ID
		}
		c.name += tc.Function.Name
		c.arguments.WriteString(tc.Function.Arguments)
	}
	for _, entry := range delta.ReasoningDetails {
		if err := s.detail(entry); err != nil {
			return false, err
		}
	}
	s.reasoningContent.WriteString(delta.ReasoningContent)
	s.reasoning.WriteString(delta.Reasoning)
	reasoning := delta.ReasoningContent
	if reasoning == "" {
		reasoning = delta.Reasoning
	}
	if reasoning != "" && !s.yield(llmprovider.Event{Type: llmprovider.EventReasoningDelta, Text: reasoning}) {
		return true, nil
	}
	// A refusal's delta is a text delta, as on the Responses wire (0031-MADR
	// amendment of 2026-10-10, D4).
	for _, part := range []struct {
		text string
		to   *strings.Builder
	}{{string(delta.Content), &s.content}, {delta.Refusal, &s.refusal}} {
		if part.text == "" {
			continue
		}
		part.to.WriteString(part.text)
		if !s.yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: part.text}) {
			return true, nil
		}
	}
	return false, nil
}

// detail merges one reasoning_details fragment into the entry its index
// names.
func (s *chatStream) detail(entry json.RawMessage) error {
	var fragment map[string]any
	if err := json.Unmarshal(entry, &fragment); err != nil {
		return fmt.Errorf("%s: reasoning_details: %w", s.provider, err)
	}
	index := 0
	if f, ok := fragment["index"].(float64); ok {
		index = int(f)
	}
	for len(s.details) <= index {
		s.details = append(s.details, map[string]any{})
	}
	mergeDetail(s.details[index], fragment)
	return nil
}

// mergeDetail adds one reasoning_details fragment to its entry: the strings a
// stream splits (text, summary, data, signature) are joined, and any other
// field keeps its first value.
func mergeDetail(entry, fragment map[string]any) {
	for k, v := range fragment {
		switch k {
		case "text", "summary", "data", "signature":
			next, ok := v.(string)
			if !ok {
				continue
			}
			if prev, ok := entry[k].(string); ok {
				next = prev + next
			}
			entry[k] = next
		default:
			if _, ok := entry[k]; !ok {
				entry[k] = v
			}
		}
	}
}

// reasoningDetail is one reasoning_details entry as a ReasoningItem: its text
// or summary, and the entry itself, compacted, to replay as it came.
func reasoningDetail(raw json.RawMessage) (llmprovider.ReasoningItem, error) {
	var entry struct {
		Text    string `json:"text"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(raw, &entry); err != nil {
		return llmprovider.ReasoningItem{}, fmt.Errorf("chat completions: reasoning_details: %w", err)
	}
	compact, err := wire.CompactArguments(raw)
	if err != nil {
		return llmprovider.ReasoningItem{}, fmt.Errorf("chat completions: reasoning_details: %w", err)
	}
	text := entry.Text
	if text == "" {
		text = entry.Summary
	}
	return llmprovider.ReasoningItem{Text: text, Encrypted: compact, Format: wire.FormatChatCompletions}, nil
}

// finishError is the finish_reason of a choice the gateway failed.
const finishError = "error"

// gatewayError is an error envelope inside a 200 reply. Its code may be a
// string or a number (OpenRouter sends the HTTP status).
type gatewayError struct {
	Message string          `json:"message"`
	Code    json.RawMessage `json:"code"`
	Type    string          `json:"type"`
}

// classify is the error's kind, as a stream failure's is classified, for
// provider: a numeric code is the HTTP status it names (0026-MADR F5).
func (e *gatewayError) classify(provider string) error {
	code := strings.Trim(strings.TrimSpace(string(e.Code)), `"`)
	if code == "null" {
		code = ""
	}
	return llmprovider.ClassifyStreamFailure(provider, code, e.Type, e.Message)
}

// chatArguments is a call's arguments. The standard sends a JSON string; some
// gateways send the object itself, which is kept compacted rather than failing
// the whole reply (0021-MADR W1). null, or nothing, is "{}".
type chatArguments string

// chatContent is a message's content: a string, null, or an array of parts
// whose text parts are joined (0026-MADR F30).
type chatContent string

func (c *chatContent) UnmarshalJSON(data []byte) error {
	var s *string
	if err := json.Unmarshal(data, &s); err == nil {
		if s != nil {
			*c = chatContent(*s)
		}
		return nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &parts); err != nil {
		return fmt.Errorf("chat completions: content is neither a string nor an array of parts: %w", err)
	}
	var sb strings.Builder
	for _, p := range parts {
		if p.Type == wire.KeyText {
			sb.WriteString(p.Text)
		}
	}
	*c = chatContent(sb.String())
	return nil
}

func (a *chatArguments) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if strings.TrimSpace(s) == "" {
			s = "{}"
		}
		*a = chatArguments(s)
		return nil
	}
	compact, err := wire.CompactArguments(data)
	if err != nil {
		return err
	}
	*a = chatArguments(compact)
	return nil
}

// usage is Chat Completions' token counts. prompt_tokens holds the cached
// tokens and completion_tokens the reasoning tokens, as Usage's do. The
// cached count has three spellings, read in pi's order: OpenAI's
// prompt_tokens_details.cached_tokens, DeepSeek's prompt_cache_hit_tokens,
// and a top-level cached_tokens (0017-REPORT P7). prompt_tokens holds the
// cache-written tokens too. Their count has two spellings, read in this
// order: prompt_tokens_details.cache_write_tokens, then Kilo's top-level
// cache_creation_input_tokens (0036-MADR D2, and its amendment of
// 2026-10-12).
type usage struct {
	Prompt        int  `json:"prompt_tokens"`
	Completion    int  `json:"completion_tokens"`
	CacheHit      *int `json:"prompt_cache_hit_tokens"`
	Cached        *int `json:"cached_tokens"`
	CacheCreation *int `json:"cache_creation_input_tokens"`
	PromptDetails struct {
		Cached     *int `json:"cached_tokens"`
		CacheWrite *int `json:"cache_write_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionDetails struct {
		Reasoning int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

func (u usage) counts() llmprovider.Usage {
	cached := 0
	for _, c := range []*int{u.PromptDetails.Cached, u.CacheHit, u.Cached} {
		if c != nil {
			cached = *c
			break
		}
	}
	written := 0
	for _, w := range []*int{u.PromptDetails.CacheWrite, u.CacheCreation} {
		if w != nil {
			written = *w
			break
		}
	}
	return llmprovider.Usage{InputTokens: u.Prompt, OutputTokens: u.Completion,
		ReasoningTokens: u.CompletionDetails.Reasoning, CachedTokens: cached, CacheWriteTokens: written}
}
