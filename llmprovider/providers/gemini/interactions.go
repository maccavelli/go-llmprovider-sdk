package gemini

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

// The provider's wire is the Interactions API, POST {base}/interactions
// (MADR 0014 §1). Every shape below was measured against gemini-3.7-flash on
// 2026-09-27; OpenCode's google route keeps generateContent
// (llmprovider/generatecontent_wire.go).
const (
	interactionsPath              = "/interactions"
	interactionStepThought        = "thought"
	interactionStepModelOutput    = "model_output"
	interactionStepUserInput      = "user_input"
	interactionStepFunctionCall   = "function_call"
	interactionStepFunctionResult = "function_result"
	interactionSummariesAuto      = "auto"
	statusIncomplete              = "incomplete"

	// skipThoughtSignature is Gemini's documented placeholder for a call it
	// did not issue. generateContent uses the same value in llmprovider.
	skipThoughtSignature = "skip_thought_signature_validator"

	jsonKeyType        = "type"
	jsonKeyText        = "text"
	jsonKeyContent     = "content"
	jsonKeyName        = "name"
	jsonKeyModel       = "model"
	jsonKeyInput       = "input"
	jsonKeyTools       = "tools"
	jsonKeyFunction    = "function"
	jsonKeyDescription = "description"
	jsonKeyParameters  = "parameters"
	jsonKeyArguments   = "arguments"
	jsonKeyCallID      = "call_id"
)

// interactionTextStep is a user_input or model_output step holding one text.
func interactionTextStep(stepType, text string) map[string]any {
	return map[string]any{jsonKeyType: stepType, jsonKeyContent: []map[string]any{{jsonKeyType: jsonKeyText, jsonKeyText: text}}}
}

// interactionsInput converts items to Interactions steps. System items go to
// system_instruction instead (SystemPrompt). A function call is preceded by
// the thought step whose signature Gemini requires to replay it (a replayed
// call without one is refused with 400), carrying the item's Signature or,
// for a call Gemini did not issue, the placeholder it accepts. A result is
// keyed by call_id and named after its call.
func interactionsInput(items []llmprovider.Item) []map[string]any {
	names := map[string]string{}
	for _, item := range items {
		if call, ok := item.(llmprovider.FunctionCallItem); ok {
			names[call.CallID] = call.Name
		}
	}
	var steps []map[string]any
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			switch v.Role {
			case llmprovider.RoleSystem:
				continue
			case "", llmprovider.RoleUser:
				steps = append(steps, interactionTextStep(interactionStepUserInput, v.Text))
			default:
				steps = append(steps, interactionTextStep(interactionStepModelOutput, v.Text))
			}
		case llmprovider.FunctionCallItem:
			signature := v.Signature
			if signature == "" {
				signature = skipThoughtSignature
			}
			steps = append(steps,
				map[string]any{jsonKeyType: interactionStepThought, "signature": signature},
				map[string]any{jsonKeyType: interactionStepFunctionCall, "id": v.CallID, jsonKeyName: v.Name,
					jsonKeyArguments: wire.ToolArguments(v.Arguments)})
		case llmprovider.FunctionCallOutputItem:
			name := names[v.CallID]
			if name == "" {
				name = v.CallID
			}
			step := map[string]any{jsonKeyType: interactionStepFunctionResult, jsonKeyCallID: v.CallID,
				jsonKeyName: name, "result": v.Output}
			// A failed call says so (0032-MADR D3).
			if v.IsError {
				step["is_error"] = true
			}
			steps = append(steps, step)
		}
	}
	return steps
}

// decodeInteraction maps an Interaction to a Response: a thought step's
// summary becomes a ReasoningItem and its signature the Signature of the
// calls after it; model_output text becomes a MessageItem; function_call a
// FunctionCallItem. incomplete is an *APIError of kind ErrIncomplete (MADR
// 0012 §1.5, 0015-MADR D7);
// failed and cancelled are retryable failures.
func decodeInteraction(body io.Reader) (*llmprovider.Response, error) {
	var raw interaction
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("gemini: decode interaction: %w", err)
	}
	return raw.response()
}

// interaction is an Interaction, whole, or assembled from its stream.
type interaction struct {
	ID     string            `json:"id"`
	Model  string            `json:"model"`
	Status string            `json:"status"`
	Steps  []interactionStep `json:"steps"`
	Usage  interactionUsage  `json:"usage"`
}

// interactionStep is one step of an Interaction.
type interactionStep struct {
	Type      string          `json:"type"`
	Signature string          `json:"signature"`
	Summary   []stepText      `json:"summary"`
	Content   []stepText      `json:"content"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// stepText is one typed text of a step's summary or content.
type stepText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// response is raw as a Response, or the error its status makes it.
func (raw interaction) response() (*llmprovider.Response, error) {
	switch raw.Status {
	case "completed", "requires_action":
	case statusIncomplete:
		// An Interaction gives no reason (measured at max_output_tokens).
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: statusIncomplete}
	default:
		return nil, fmt.Errorf("%w: gemini interaction %s", llmprovider.ErrProviderUnavailable, raw.Status)
	}
	result := &llmprovider.Response{ID: raw.ID, Model: raw.Model, Usage: raw.Usage.counts()}
	signature := ""
	for _, step := range raw.Steps {
		switch step.Type {
		case interactionStepThought:
			signature = step.Signature
			var sb strings.Builder
			for _, s := range step.Summary {
				sb.WriteString(s.Text)
			}
			if sb.Len() > 0 {
				result.Output = append(result.Output, llmprovider.ReasoningItem{Text: sb.String(), Format: wire.FormatGenerateContent})
			}
		case interactionStepModelOutput:
			var sb strings.Builder
			for _, c := range step.Content {
				if c.Type == jsonKeyText {
					sb.WriteString(c.Text)
				}
			}
			if sb.Len() > 0 {
				result.Output = append(result.Output,
					llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: sb.String()})
			}
		case interactionStepFunctionCall:
			// Compact, as the generateContent decoder returns arguments.
			args, err := wire.CompactArguments(step.Arguments)
			if err != nil {
				return nil, fmt.Errorf("gemini: function_call arguments: %w", err)
			}
			result.Output = append(result.Output, llmprovider.FunctionCallItem{CallID: step.ID, Name: step.Name,
				Arguments: args, Signature: signature})
		}
	}
	if len(result.Output) == 0 {
		return nil, wire.EmptyAnswer("gemini interaction", "")
	}
	// Stop, or tool_calls when the model wants a tool (0020-MADR F11).
	result.FinishReason = llmprovider.FinishStop
	for _, item := range result.Output {
		if _, ok := item.(llmprovider.FunctionCallItem); ok {
			result.FinishReason = llmprovider.FinishToolCalls
		}
	}
	return result, nil
}

// interactionEvent is the part of one Interactions stream event the reader
// uses (0031-PLAN-native-streaming-on-the-other-wires, Phase 1).
type interactionEvent struct {
	EventType   string          `json:"event_type"`
	Index       int             `json:"index"`
	Interaction interaction     `json:"interaction"`
	Step        interactionStep `json:"step"`
	Delta       struct {
		Type      string          `json:"type"`
		Text      string          `json:"text"`
		Signature string          `json:"signature"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"delta"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// interactionStream is an Interaction as its events arrive.
type interactionStream struct {
	provider string
	raw      interaction
	text     []*strings.Builder // each step's text deltas
	args     []*strings.Builder // each step's argument fragments
	sig      []*strings.Builder // each step's signature deltas (0053-MADR D1)
	yield    func(llmprovider.Event) bool
}

// interactionEvents reads an Interactions event stream and yields its events
// as they arrive (0031-MADR E3): a model_output step's text delta as
// EventTextDelta, and at interaction.completed the items decodeInteraction
// returns for the same interaction, in its order, then EventDone, carrying
// that Response. A thought step's signature and a call's arguments_delta are
// assembled into their items; no thought summary delta was seen streaming, so
// none is yielded (0031-PLAN-native-streaming-on-the-other-wires, Phase 1). A
// call's arguments may arrive as text fragments or as an object. An error
// event is classified by the status its code names. It returns nil once
// EventDone is yielded or yield returns false, and otherwise the error that
// ended the stream; what it yielded before an error is not a response
// (0031-MADR D4).
func interactionEvents(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	s := &interactionStream{provider: provider, yield: yield}
	return wire.ReadEvents(provider, body, "interaction.completed", s.event)
}

// event handles one event.
func (s *interactionStream) event(payload []byte) (bool, error) {
	if string(payload) == "[DONE]" {
		return false, nil
	}
	var ev interactionEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return false, fmt.Errorf("%s: decode stream event: %w", s.provider, err)
	}
	if e := ev.Error; e != nil {
		return false, llmprovider.ClassifyStreamFailure(s.provider, strconv.Itoa(e.Code), e.Status, e.Message)
	}
	switch ev.EventType {
	case "interaction.created":
		s.raw.ID, s.raw.Model = ev.Interaction.ID, ev.Interaction.Model
	case "step.start":
		step, err := s.step(ev.Index)
		if err != nil {
			return false, err
		}
		*step = ev.Step
	case "step.delta":
		return s.delta(ev)
	case "interaction.completed":
		return s.completed(ev.Interaction)
	}
	return false, nil
}

// step is the step at index, grown to it. The index is the service's:
// checked before it grows the steps (0053-MADR D2).
func (s *interactionStream) step(index int) (*interactionStep, error) {
	if err := wire.StreamIndex(s.provider, "step", index); err != nil {
		return nil, err
	}
	for len(s.raw.Steps) <= index {
		s.raw.Steps = append(s.raw.Steps, interactionStep{})
		s.text = append(s.text, &strings.Builder{})
		s.args = append(s.args, &strings.Builder{})
		s.sig = append(s.sig, &strings.Builder{})
	}
	return &s.raw.Steps[index], nil
}

// delta adds one step delta.
func (s *interactionStream) delta(ev interactionEvent) (bool, error) {
	step, err := s.step(ev.Index)
	if err != nil {
		return false, err
	}
	switch ev.Delta.Type {
	case jsonKeyText:
		s.text[ev.Index].WriteString(ev.Delta.Text)
		if step.Type == interactionStepModelOutput {
			return !s.yield(llmprovider.Event{Type: llmprovider.EventTextDelta, Text: ev.Delta.Text}), nil
		}
	case "thought_signature":
		s.sig[ev.Index].WriteString(ev.Delta.Signature)
	case "arguments_delta":
		var fragment string
		if err := json.Unmarshal(ev.Delta.Arguments, &fragment); err == nil {
			s.args[ev.Index].WriteString(fragment)
		} else if len(ev.Delta.Arguments) > 0 {
			// The whole object, in place of any fragments.
			s.args[ev.Index].Reset()
			s.args[ev.Index].Write(ev.Delta.Arguments)
		}
	}
	return false, nil
}

// completed finishes the interaction: its items, then EventDone.
func (s *interactionStream) completed(final interaction) (bool, error) {
	s.raw.Status, s.raw.Usage = final.Status, final.Usage
	if final.ID != "" {
		s.raw.ID = final.ID
	}
	if final.Model != "" {
		s.raw.Model = final.Model
	}
	for i := range s.raw.Steps {
		step := &s.raw.Steps[i]
		if text := s.text[i].String(); text != "" {
			step.Content = append(step.Content, stepText{Type: jsonKeyText, Text: text})
		}
		if s.args[i].Len() > 0 {
			step.Arguments = json.RawMessage(s.args[i].String())
		}
		step.Signature += s.sig[i].String()
	}
	resp, err := s.raw.response()
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

// interactionUsage is an Interaction's token counts (ai.google.dev/api/
// interactions-api, read 2026-10-01). total_input_tokens holds the cached
// tokens. total_thought_tokens is outside total_output_tokens, as in
// generateContent: measured 2026-10-02 by
// TestLive_GeminiInteractionsThoughtTokens, where 17 input, 3 output and 114
// thought tokens made a total of 134 (0015-MADR amendment "what `Usage`
// counts"). total_tool_use_tokens is not counted.
type interactionUsage struct {
	Input    int `json:"total_input_tokens"`
	Output   int `json:"total_output_tokens"`
	Cached   int `json:"total_cached_tokens"`
	Thoughts int `json:"total_thought_tokens"`
}

func (u interactionUsage) counts() llmprovider.Usage {
	return llmprovider.Usage{InputTokens: u.Input, OutputTokens: u.Output + u.Thoughts,
		ReasoningTokens: u.Thoughts, CachedTokens: u.Cached}
}
