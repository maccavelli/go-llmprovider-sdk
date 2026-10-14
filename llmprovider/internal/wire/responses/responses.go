// Package responses is the OpenAI Responses wire, shared by the openai and
// grok providers and OpenCode's responses route (0015-MADR D2).
package responses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
)

const (
	keyCallID = "call_id"

	itemTypeMessage            = "message"
	itemTypeFunctionCall       = "function_call"
	itemTypeFunctionCallOutput = "function_call_output"
	itemTypeReasoning          = "reasoning"
	// itemTypeToolSearchCall and itemTypeToolSearchOutput are a client tool
	// search's call and the caller's answer (0040-MADR D3).
	itemTypeToolSearchCall   = "tool_search_call"
	itemTypeToolSearchOutput = "tool_search_output"
	executionClientValue     = "client"

	// statusIncomplete is the status of a truncated answer.
	statusIncomplete = "incomplete"
	// statusFailed is the status of a response the service failed.
	statusFailed = "failed"
)

// InputItem is every Responses input item in one struct: a message, an
// encrypted reasoning item, a function call and its output. A field an item
// does not have is nil, and omitted. The fields are in the order json.Marshal
// wrote the maps they replace, so the JSON is the same byte for byte
// (0028-MADR D-A6, amendments 2026-10-09).
type InputItem struct {
	Arguments        Arguments         `json:"arguments,omitzero"`
	CallID           *string           `json:"call_id,omitempty"`
	Content          *string           `json:"content,omitempty"`
	EncryptedContent *string           `json:"encrypted_content,omitempty"`
	Execution        *string           `json:"execution,omitempty"`
	Name             *string           `json:"name,omitempty"`
	Output           *string           `json:"output,omitempty"`
	Role             *string           `json:"role,omitempty"`
	Status           *string           `json:"status,omitempty"`
	Summary          *[]any            `json:"summary,omitempty"`
	Tools            *[]map[string]any `json:"tools,omitempty"`
	Type             *string           `json:"type,omitempty"`
}

// Arguments is an input item's arguments: a function_call's JSON text, sent
// as a string as before 0040, or a tool_search_call's object, sent as JSON
// (0040-MADR D3). The zero value is omitted.
type Arguments struct {
	text   *string
	object json.RawMessage
}

// IsZero reports an item with no arguments, which omitzero leaves out.
func (a Arguments) IsZero() bool { return a.text == nil && a.object == nil }

// MarshalJSON writes the object as it is, or the text as a string.
func (a Arguments) MarshalJSON() ([]byte, error) {
	if a.object != nil {
		return a.object, nil
	}
	if a.text == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*a.text)
}

// The constant fields InputItem points to.
var (
	typeReasoning          = itemTypeReasoning
	typeFunctionCall       = itemTypeFunctionCall
	typeFunctionCallOutput = itemTypeFunctionCallOutput
	roleUser               = wire.RoleUser
	emptySummary           = []any{}
	typeToolSearchCall     = itemTypeToolSearchCall
	typeToolSearchOutput   = itemTypeToolSearchOutput
	executionClient        = executionClientValue
	statusCompleted        = "completed"
)

// Input converts items to the Responses API input format. The strings the
// items carry are kept in one arena (0028-PLAN D16).
func Input(items []llmprovider.Item) []InputItem {
	var input []InputItem
	arena := wire.NewArena(3 * len(items))
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.MessageItem:
			role := &roleUser // an empty Role is the user's (0020-MADR F10)
			if v.Role != "" {
				role = arena.Keep(string(v.Role))
			}
			input = append(input, InputItem{Role: role, Content: arena.Keep(v.Text)})
		case llmprovider.ReasoningItem:
			// Encrypted reasoning goes back as it came (0020-MADR F24), to
			// this wire only (0021-MADR W7); plain text reasoning cannot be
			// replayed.
			if v.Encrypted != "" && wire.Replays(v, wire.FormatResponses) {
				input = append(input, InputItem{Type: &typeReasoning, EncryptedContent: arena.Keep(v.Encrypted), Summary: &emptySummary})
			}
		case llmprovider.FunctionCallItem:
			input = append(input, InputItem{Type: &typeFunctionCall, CallID: arena.Keep(v.CallID),
				Name: arena.Keep(v.Name), Arguments: Arguments{text: arena.Keep(v.Arguments)}})
		case llmprovider.FunctionCallOutputItem:
			input = append(input, InputItem{Type: &typeFunctionCallOutput, CallID: arena.Keep(v.CallID), Output: arena.Keep(v.Output)})
		}
	}
	return input
}

// InputLoading is Input for a request whose deferred tools go out natively
// (0040-MADR D3). A call to req's Search tool is sent as a client
// tool_search_call, its arguments as a JSON object, and its result as a
// tool_search_output whose tools are the definitions its ToolLoadItems load,
// each with defer_loading; the result's text has no field there, and is not
// sent. Every other item is sent as Input sends it. With no Search tool or no
// deferred tool it is Input.
func InputLoading(items []llmprovider.Item, req *llmprovider.Request) []InputItem {
	search := ""
	for _, tool := range req.Tools {
		if tool.Search {
			search = tool.Name
		}
	}
	if search == "" || !deferral.Deferring(req.Tools) {
		return Input(items)
	}
	loads := deferral.SearchLoads(req)
	defs := map[string]llmprovider.Tool{}
	for _, tool := range req.Tools {
		defs[tool.Name] = tool
	}
	searchCalls := map[string]bool{}
	var input []InputItem
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.FunctionCallItem:
			if v.Name != search {
				input = append(input, Input([]llmprovider.Item{v})...)
				continue
			}
			searchCalls[v.CallID] = true
			callID, arguments := v.CallID, json.RawMessage("{}")
			if trimmed := strings.TrimSpace(v.Arguments); json.Valid([]byte(trimmed)) && strings.HasPrefix(trimmed, "{") {
				arguments = json.RawMessage(trimmed)
			}
			input = append(input, InputItem{Type: &typeToolSearchCall, CallID: &callID, Execution: &executionClient,
				Status: &statusCompleted, Arguments: Arguments{object: arguments}})
		case llmprovider.FunctionCallOutputItem:
			if !searchCalls[v.CallID] {
				input = append(input, Input([]llmprovider.Item{v})...)
				continue
			}
			var loaded []llmprovider.Tool
			for _, name := range loads[v.CallID] {
				loaded = append(loaded, defs[name])
			}
			tools := wire.ResponsesTools(loaded)
			for _, tool := range tools {
				tool["defer_loading"] = true
			}
			callID := v.CallID
			input = append(input, InputItem{Type: &typeToolSearchOutput, CallID: &callID, Execution: &executionClient,
				Status: &statusCompleted, Tools: &tools})
		default:
			input = append(input, Input([]llmprovider.Item{v})...)
		}
	}
	return input
}

// DecodeFor is Decode for provider, the label the caller gives wire.Call: a
// failed response names provider, and is classified by that service's error
// vocabulary, as ReadStream's is (0026-MADR F28).
func DecodeFor(provider string) func(io.Reader) (*llmprovider.Response, error) {
	return Decoder{}.DecodeFor(provider)
}

// Decoder decodes Responses replies for one request. Search names the
// request's Search tool, when its deferred tools go out natively: a client
// tool_search_call then decodes as a FunctionCallItem calling it, its
// arguments the object as JSON text. With no Search, such a call is dropped,
// as every unknown item is (0040-MADR D3).
type Decoder struct{ Search string }

// DecodeFor is the package's DecodeFor, with d's Search.
func (d Decoder) DecodeFor(provider string) func(io.Reader) (*llmprovider.Response, error) {
	return func(body io.Reader) (*llmprovider.Response, error) { return decodeAs(body, provider, d.Search) }
}

// ReadStream is the package's ReadStream, with d's Search.
func (d Decoder) ReadStream(provider string, body io.Reader) (*llmprovider.Response, error) {
	var result *llmprovider.Response
	err := readEvents(provider, body, handledEvents, d.Search, func(ev llmprovider.Event) bool {
		if ev.Type == llmprovider.EventDone {
			result = ev.Response
		}
		return true
	})
	return finished(result, err)
}

// Events is the package's Events, with d's Search.
func (d Decoder) Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	return readEvents(provider, body, streamedEvents, d.Search, yield)
}

// Decode decodes a Responses API JSON body into a Response.
func Decode(body io.Reader) (*llmprovider.Response, error) {
	return decodeAs(body, "responses", "")
}

// decodeAs is Decode, with a failure labelled and classified for provider.
func decodeAs(body io.Reader, provider, search string) (*llmprovider.Response, error) {
	var raw struct {
		ID                string         `json:"id"`
		Model             string         `json:"model"`
		Status            string         `json:"status"`
		Error             *responseError `json:"error"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Output []outputItem `json:"output"`
		Usage  usage        `json:"usage"`
	}
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return nil, err
	}
	switch raw.Status {
	case statusFailed:
		// The status is read first, as ReadStream reads response.failed.
		return nil, failed(provider, raw.Error)
	case statusIncomplete:
		// A truncated answer is an error, never an empty success (MADR 0012
		// §1.5).
		return nil, incomplete(raw.IncompleteDetails.Reason)
	}

	result := &llmprovider.Response{ID: raw.ID, Model: raw.Model, Usage: raw.Usage.counts()}
	refused := false
	for _, out := range raw.Output {
		refused = appendOutput(result, out, search) || refused
	}
	return completed(result, refused)
}

// completed finishes a completed answer: an answer with nothing in it is
// ErrIncomplete, never an empty success, as every other wire has it
// (0021-MADR W4).
func completed(r *llmprovider.Response, refused bool) (*llmprovider.Response, error) {
	r.FinishReason = finishReason(r, refused)
	if len(r.Output) == 0 {
		return nil, wire.EmptyAnswer("responses", r.FinishReason)
	}
	return r, nil
}

// finishReason is a completed answer's: content_filter when it refused,
// tool_calls when it calls a tool, else stop (0020-MADR F11; 0021-MADR W4). An
// incomplete one is an error, never a Response.
func finishReason(r *llmprovider.Response, refused bool) llmprovider.FinishReason {
	if refused {
		return llmprovider.FinishContentFilter
	}
	for _, item := range r.Output {
		if _, ok := item.(llmprovider.FunctionCallItem); ok {
			return llmprovider.FinishToolCalls
		}
	}
	return llmprovider.FinishStop
}

// incomplete is the MADR 0012 §1.5 error for an incomplete Responses answer.
// The reason is the service's text, so it is stripped and bounded (0026-MADR
// F18).
func incomplete(reason string) error {
	reason = redact.Field(reason)
	if reason == "" {
		reason = "unspecified"
	}
	return &llmprovider.APIError{Kind: llmprovider.ErrIncomplete, Reason: reason}
}

// outputItem is one Responses API output entry.
type outputItem struct {
	Type    string `json:"type"`
	Content []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Refusal string `json:"refusal"`
	} `json:"content"`
	Summary []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"summary"`
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	// Arguments is a function_call's JSON text, as a string, or a
	// tool_search_call's object (0040-MADR D3).
	Arguments        json.RawMessage `json:"arguments"`
	EncryptedContent string          `json:"encrypted_content"`
	Execution        string          `json:"execution"`
}

// argumentsText is raw arguments as JSON text: a string's content, or an
// object compacted. Nothing, null and a string that is not JSON are "".
func argumentsText(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	switch {
	case len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")):
		return ""
	case trimmed[0] == '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return ""
		}
		return s
	default:
		var out bytes.Buffer
		if err := json.Compact(&out, trimmed); err != nil {
			return ""
		}
		return out.String()
	}
}

// appendOutput adds to r the item one Responses output entry carries, if any,
// and reports whether it was a refusal. A refusal part's text is kept as the
// answer's text (0021-MADR W4). It was the method Response.appendOutput
// (0015-PLAN S7b step 5).
func appendOutput(r *llmprovider.Response, out outputItem, search string) (refused bool) {
	switch out.Type {
	case itemTypeMessage:
		var sb strings.Builder
		for _, c := range out.Content {
			switch c.Type {
			case "output_text", wire.KeyText:
				sb.WriteString(c.Text)
			case "refusal":
				sb.WriteString(c.Refusal)
				refused = true
			}
		}
		if text := sb.String(); text != "" {
			r.Output = append(r.Output, llmprovider.MessageItem{Role: wire.RoleAssistant, Text: text})
		}
	case itemTypeFunctionCall:
		// Empty arguments are "{}", as on every other wire (0021-MADR W1;
		// 0026-MADR F31).
		args := argumentsText(out.Arguments)
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		r.Output = append(r.Output, llmprovider.FunctionCallItem{
			CallID:    out.CallID,
			Name:      out.Name,
			Arguments: args,
		})
	case itemTypeToolSearchCall:
		// A client search the caller runs, as a call to its Search tool; a
		// hosted one, or one with no Search tool to name, is dropped
		// (0040-MADR D3).
		if search == "" || out.Execution != executionClientValue {
			break
		}
		args := argumentsText(out.Arguments)
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		r.Output = append(r.Output, llmprovider.FunctionCallItem{CallID: out.CallID, Name: search, Arguments: args})
	case itemTypeReasoning:
		var sb strings.Builder
		for _, s := range out.Summary {
			if s.Type == "summary_text" || s.Type == wire.KeyText {
				sb.WriteString(s.Text)
			}
		}
		// An item with neither text nor encrypted content says nothing, and
		// is not kept (0020-MADR F24).
		if sb.Len() > 0 || out.EncryptedContent != "" {
			r.Output = append(r.Output, llmprovider.ReasoningItem{Text: sb.String(), Encrypted: out.EncryptedContent,
				Format: wire.FormatResponses})
		}
	}
	return refused
}

// streamEvent is the part of one Responses stream event the reader uses. The
// type is read from the payload, not the SSE "event:" line.
type streamEvent struct {
	Type string `json:"type"`
	// Code and Message are a top-level error event's (0020-MADR F39).
	Code     string     `json:"code"`
	Message  string     `json:"message"`
	Item     outputItem `json:"item"`
	Response struct {
		ID                string         `json:"id"`
		Model             string         `json:"model"`
		Error             *responseError `json:"error"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Usage usage `json:"usage"`
	} `json:"response"`
}

// responseError is a failed response's error.
type responseError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// failed classifies a failed response, streamed or not, by its error as
// Codex does; one with no error is retryable (0026-MADR F28).
func failed(provider string, e *responseError) error {
	if e == nil {
		return llmprovider.ClassifyStreamFailure(provider, "", "", "response.failed event received")
	}
	return llmprovider.ClassifyStreamFailure(provider, e.Code, e.Type, e.Message)
}

// ReadStream reads a Responses API event stream into a Response, as Codex
// does (codex-api/src/sse/responses.rs:343-450): each
// response.output_item.done adds its item, and response.created and
// response.completed carry the id. response.failed maps onto MADR 0012 §1.1,
// response.incomplete onto MADR 0012 §1.5, and a stream that ends before
// response.completed is retryable, once (MADR 0012 §4.1; 0021-MADR D1). Each
// event is bounded, not the stream (0021-MADR W2). It does not rely on
// Content-Type, which the ChatGPT backend does not send. It collects Events,
// skipping the deltas it does not need (0031-MADR D2).
func ReadStream(provider string, body io.Reader) (*llmprovider.Response, error) {
	return Decoder{}.ReadStream(provider, body)
}

// Events reads a Responses API event stream and yields its events as they
// arrive (0031-MADR D2): a text delta as EventTextDelta, a reasoning summary
// delta, the one measured, as EventReasoningDelta (0031-PLAN Phase 1), each
// response.output_item.done's item as EventItem, and response.completed as
// EventDone, carrying the Response ReadStream returns. A call-argument delta
// and every other event yield nothing (0031-MADR Q4). It returns nil once
// EventDone is yielded or yield returns false, and otherwise the error that
// ended the stream, as ReadStream's; what it yielded before an error is not a
// response (0031-MADR D4).
func Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	return Decoder{}.Events(provider, body, yield)
}

// readEvents is Events, decoding only the event types in handled.
func readEvents(provider string, body io.Reader, handled map[string]bool, search string, yield func(llmprovider.Event) bool) error {
	result := &llmprovider.Response{}
	refused := false
	// dispatch handles one event; stop ends the stream without an error,
	// after EventDone or when the caller stops.
	dispatch := func(payload []byte) (stop bool, err error) {
		if string(payload) == "[DONE]" || ignored(payload, handled) {
			return false, nil
		}
		if deltaType, ok := deltaEvents[string(eventType(payload))]; ok {
			var delta struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(payload, &delta); err != nil {
				return false, fmt.Errorf("%s: decode stream event: %w", provider, err)
			}
			return !yield(llmprovider.Event{Type: deltaType, Text: delta.Delta}), nil
		}
		var event streamEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return false, fmt.Errorf("%s: decode stream event: %w", provider, err)
		}
		switch event.Type {
		case "response.created":
			result.ID, result.Model = event.Response.ID, event.Response.Model
		case "error":
			// The stream's own error event, not a response's (0020-MADR F39).
			return false, llmprovider.ClassifyStreamFailure(provider, event.Code, "", event.Message)
		case "response.output_item.done":
			n := len(result.Output)
			refused = appendOutput(result, event.Item, search) || refused
			if len(result.Output) > n {
				return !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: result.Output[n]}), nil
			}
		case "response.failed":
			return false, failed(provider, event.Response.Error)
		case "response.incomplete":
			return false, incomplete(event.Response.IncompleteDetails.Reason)
		case "response.completed":
			if event.Response.ID != "" {
				result.ID = event.Response.ID
			}
			if event.Response.Model != "" {
				result.Model = event.Response.Model
			}
			result.Usage = event.Response.Usage.counts()
			resp, err := completed(result, refused)
			if err != nil {
				return false, err
			}
			yield(llmprovider.Event{Type: llmprovider.EventDone, Response: resp})
			return true, nil
		}
		return false, nil
	}
	// The framing, the per-event bound and the early end are the shared
	// reader's (0031-MADR E2).
	return wire.ReadEvents(provider, body, "response.completed", dispatch)
}

// finished is a stream's outcome: the response once it completed, or the
// error alone. A failed event never hands back the partial response with its
// error (R7; 0021-MADR W14).
func finished(result *llmprovider.Response, err error) (*llmprovider.Response, error) {
	if err != nil {
		return nil, err
	}
	return result, nil
}

// handledEvents are the event types ReadStream acts on.
var handledEvents = map[string]bool{
	"response.created": true, "response.output_item.done": true, "response.failed": true,
	"response.incomplete": true, "response.completed": true,
}

// deltaEvents are the delta event types Events yields, and the Event type of
// each. The reasoning delta is the one 0031-PLAN Phase 1 measured. A
// refusal's delta is a text delta, since appendOutput keeps a refusal's text
// as the answer's (0021-MADR W4); its name is OpenAI's documented one (0031-MADR,
// amendment of 2026-10-10).
var deltaEvents = map[string]llmprovider.EventType{
	"response.output_text.delta":            llmprovider.EventTextDelta,
	"response.refusal.delta":                llmprovider.EventTextDelta,
	"response.reasoning_summary_text.delta": llmprovider.EventReasoningDelta,
}

// streamedEvents are the event types Events acts on: ReadStream's, and the
// deltas.
var streamedEvents = func() map[string]bool {
	events := maps.Clone(handledEvents)
	for event := range deltaEvents {
		events[event] = true
	}
	return events
}()

// ignored reports whether an event is one the reader does not use, as handled
// lists them, such as an in-progress event, or a delta when the done events
// carry the whole items. It reads the first "type" value without decoding the
// payload. Only a value that starts "response." counts: no item or content
// type does, so a nested type can never be mistaken for the event's own, and
// a payload it cannot read is decoded in full (0021-MADR W2).
func ignored(payload []byte, handled map[string]bool) bool {
	t := eventType(payload)
	return bytes.HasPrefix(t, []byte("response.")) && !handled[string(t)]
}

// eventType is the first "type" value of payload, or nil when it has none.
// It is a slice of payload, so that a map lookup by it does not allocate.
func eventType(payload []byte) []byte {
	const key = `"type":"`
	_, rest, ok := bytes.Cut(payload, []byte(key))
	if !ok {
		return nil
	}
	t, _, ok := bytes.Cut(rest, []byte{'"'})
	if !ok {
		return nil
	}
	return t
}

// usage is the Responses API's token counts. Its input count holds the cached
// and the cache-written tokens, and its output count the reasoning tokens, as
// Usage's do (0036-MADR D2).
type usage struct {
	Input        int `json:"input_tokens"`
	Output       int `json:"output_tokens"`
	InputDetails struct {
		Cached     int `json:"cached_tokens"`
		CacheWrite int `json:"cache_write_tokens"`
	} `json:"input_tokens_details"`
	OutputDetails struct {
		Reasoning int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

func (u usage) counts() llmprovider.Usage {
	return llmprovider.Usage{InputTokens: u.Input, OutputTokens: u.Output,
		ReasoningTokens: u.OutputDetails.Reasoning, CachedTokens: u.InputDetails.Cached,
		CacheWriteTokens: u.InputDetails.CacheWrite}
}
