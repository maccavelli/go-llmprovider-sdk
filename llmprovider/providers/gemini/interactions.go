package gemini

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
			steps = append(steps, map[string]any{jsonKeyType: interactionStepFunctionResult, jsonKeyCallID: v.CallID,
				jsonKeyName: name, "result": v.Output})
		}
	}
	return steps
}

// thinkingLevel maps an effort to thinking_level: low, medium or high, with
// xhigh sent as high; anything else is omitted, leaving the model's default.
// gemini-2.5-flash-lite refuses medium and fails on low (measured
// 2026-09-27), so it gets high or nothing.
func thinkingLevel(model string, effort llmprovider.Effort) string {
	level := llmprovider.Effort(strings.ToLower(string(effort)))
	switch level {
	case llmprovider.EffortXHigh:
		level = llmprovider.EffortHigh
	case llmprovider.EffortLow, llmprovider.EffortMedium, llmprovider.EffortHigh:
	default:
		return ""
	}
	if strings.HasPrefix(strings.ToLower(model), "gemini-2.5-flash-lite") && level != llmprovider.EffortHigh {
		return ""
	}
	return string(level)
}

// decodeInteraction maps an Interaction to a Response: a thought step's
// summary becomes a ReasoningItem and its signature the Signature of the
// calls after it; model_output text becomes a MessageItem; function_call a
// FunctionCallItem. incomplete is an *APIError of kind ErrIncomplete (MADR
// 0012 §1.5, 0015-MADR D7);
// failed and cancelled are retryable failures.
func decodeInteraction(body io.Reader) (*llmprovider.Response, error) {
	var raw struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Steps  []struct {
			Type      string `json:"type"`
			Signature string `json:"signature"`
			Summary   []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"summary"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"steps"`
		Usage interactionUsage `json:"usage"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("gemini: decode interaction: %w", err)
	}
	switch raw.Status {
	case "completed", "requires_action":
	case statusIncomplete:
		// An Interaction gives no reason (measured at max_output_tokens).
		return nil, &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: statusIncomplete}
	default:
		return nil, fmt.Errorf("%w: gemini interaction %s", llmprovider.ErrProviderUnavailable, raw.Status)
	}
	result := &llmprovider.Response{ID: raw.ID, Usage: raw.Usage.counts()}
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
				result.Output = append(result.Output, llmprovider.ReasoningItem{Text: sb.String()})
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
			args := "{}"
			if a := bytes.TrimSpace(step.Arguments); len(a) > 0 && string(a) != "null" {
				var buf bytes.Buffer
				if err := json.Compact(&buf, a); err != nil {
					return nil, fmt.Errorf("gemini: function_call arguments: %w", err)
				}
				args = buf.String()
			}
			result.Output = append(result.Output, llmprovider.FunctionCallItem{CallID: step.ID, Name: step.Name,
				Arguments: args, Signature: signature})
		}
	}
	if len(result.Output) == 0 {
		return nil, errors.New("gemini returned no content")
	}
	return result, nil
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
