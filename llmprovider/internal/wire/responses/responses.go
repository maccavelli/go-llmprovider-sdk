// Package responses is the OpenAI Responses wire, shared by the openai and
// grok providers and OpenCode's responses route (0015-MADR D2).
package responses

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const (
	keyCallID = "call_id"

	itemTypeMessage            = "message"
	itemTypeFunctionCall       = "function_call"
	itemTypeFunctionCallOutput = "function_call_output"
	itemTypeReasoning          = "reasoning"

	// statusIncomplete is the status of a truncated answer.
	statusIncomplete = "incomplete"
)

// Input converts items to the Responses API input format.
func Input(items []llmprovider.Item) []map[string]any {
	var input []map[string]any
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			input = append(input, map[string]any{
				wire.KeyRole:    v.Role,
				wire.KeyContent: v.Text,
			})
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

// Decode decodes a Responses API JSON body into a Response.
func Decode(body io.Reader) (*llmprovider.Response, error) {
	var raw struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []outputItem `json:"output"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	// A truncated answer is an error, never an empty success (MADR 0012 §1.5).
	if raw.Status == statusIncomplete {
		return nil, incomplete(raw.IncompleteDetails.Reason)
	}

	result := &llmprovider.Response{ID: raw.ID}
	for _, out := range raw.Output {
		appendOutput(result, out)
	}
	return result, nil
}

// incomplete is the MADR 0012 §1.5 error for an incomplete Responses answer.
func incomplete(reason string) error {
	if reason == "" {
		reason = "unspecified"
	}
	return &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: reason}
}

// outputItem is one Responses API output entry.
type outputItem struct {
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Summary []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"summary"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// appendOutput adds to r the item one Responses output entry carries, if any.
// It was the method Response.appendOutput (0015-PLAN S7b step 5).
func appendOutput(r *llmprovider.Response, out outputItem) {
	switch out.Type {
	case itemTypeMessage:
		var sb strings.Builder
		for _, c := range out.Content {
			if c.Type == "output_text" || c.Type == wire.KeyText {
				sb.WriteString(c.Text)
			}
		}
		if text := sb.String(); text != "" {
			r.Output = append(r.Output, llmprovider.MessageItem{Role: wire.RoleAssistant, Text: text})
		}
	case itemTypeFunctionCall:
		r.Output = append(r.Output, llmprovider.FunctionCallItem{
			CallID:    out.CallID,
			Name:      out.Name,
			Arguments: out.Arguments,
		})
	case itemTypeReasoning:
		var sb strings.Builder
		for _, s := range out.Summary {
			if s.Type == "summary_text" || s.Type == wire.KeyText {
				sb.WriteString(s.Text)
			}
		}
		r.Output = append(r.Output, llmprovider.ReasoningItem{Text: sb.String()})
	}
}

// streamLimit bounds one Responses event stream.
const streamLimit = 16 << 20

// streamEvent is the part of one Responses stream event the reader uses. The
// type is read from the payload, not the SSE "event:" line.
type streamEvent struct {
	Type     string     `json:"type"`
	Item     outputItem `json:"item"`
	Response struct {
		ID    string `json:"id"`
		Error *struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
	} `json:"response"`
}

// ReadStream reads a Responses API event stream into a Response, as Codex
// does (codex-api/src/sse/responses.rs:343-450): each
// response.output_item.done adds its item, and response.created and
// response.completed carry the id. response.failed maps onto MADR 0012 §1.1,
// response.incomplete onto MADR 0012 §1.5, and a stream that ends before
// response.completed is retryable (MADR 0012 §4.1). It does not rely on
// Content-Type, which the ChatGPT backend does not send.
func ReadStream(provider string, body io.Reader) (*llmprovider.Response, error) {
	scanner := bufio.NewScanner(io.LimitReader(body, streamLimit))
	scanner.Buffer(make([]byte, 0, 64<<10), streamLimit)
	result := &llmprovider.Response{}
	var data strings.Builder
	dispatch := func() (done bool, err error) {
		payload := data.String()
		data.Reset()
		if payload == "" || payload == "[DONE]" {
			return false, nil
		}
		var event streamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return false, fmt.Errorf("%s: decode stream event: %w", provider, err)
		}
		switch event.Type {
		case "response.created":
			result.ID = event.Response.ID
		case "response.output_item.done":
			appendOutput(result, event.Item)
		case "response.failed":
			e := event.Response.Error
			if e == nil {
				return false, llmprovider.ClassifyStreamFailure(provider, "", "", "response.failed event received")
			}
			return false, llmprovider.ClassifyStreamFailure(provider, e.Code, e.Type, e.Message)
		case "response.incomplete":
			return false, incomplete(event.Response.IncompleteDetails.Reason)
		case "response.completed":
			if event.Response.ID != "" {
				result.ID = event.Response.ID
			}
			return true, nil
		}
		return false, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if done, err := dispatch(); done || err != nil {
				return result, err
			}
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %s: read stream: %w", llmprovider.ErrProviderUnavailable, provider, err)
	}
	if done, err := dispatch(); done || err != nil {
		return result, err
	}
	return nil, fmt.Errorf("%w: %s: stream ended before response.completed", llmprovider.ErrProviderUnavailable, provider)
}
