// Package responses is the OpenAI Responses wire, shared by the openai and
// grok providers and OpenCode's responses route (0015-MADR D2), and by a
// chatcompat provider built with chatcompat.WithResponses (0091-MADR ST52).
package responses

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
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

	// contentReasoningText is a reasoning item's content part that carries
	// its text, where a server sends no summary (0091-MADR ST10).
	contentReasoningText = "reasoning_text"
)

// InputItem is every Responses input item in one struct: a message, an
// encrypted reasoning item, a function call and its output. A field an item
// does not have is nil, and omitted. The fields are in the order json.Marshal
// wrote the maps they replace, so the JSON is the same byte for byte
// (0028-MADR D-A6, amendments 2026-10-09).
type InputItem struct {
	Arguments        Arguments         `json:"arguments,omitzero"`
	CallID           *string           `json:"call_id,omitempty"`
	Content          MessageContent    `json:"content,omitzero"`
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

// MessageContent is an input message's content: its text, sent as a string
// as before 0061, or its parts, input_text and input_image, once an image
// joins it (0061-MADR D5). The zero value is omitted.
type MessageContent struct {
	text  *string
	parts []contentPart
}

// IsZero reports an item with no content, which omitzero leaves out.
func (c MessageContent) IsZero() bool { return c.text == nil && c.parts == nil }

// MarshalJSON writes the parts as a list, or the text as a string.
func (c MessageContent) MarshalJSON() ([]byte, error) {
	if c.parts != nil {
		return json.Marshal(c.parts)
	}
	if c.text == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*c.text)
}

// contentPart is an input_text or an input_image part.
type contentPart struct {
	ImageURL *string `json:"image_url,omitempty"`
	Text     *string `json:"text,omitempty"`
	Type     string  `json:"type"`
}

// joinImage adds img, as an input_image whose image_url is a data URL of its
// bytes or its URL, to the user message just before it, whose text becomes
// an input_text part; else it opens a user message of its own (0061-MADR D5).
func joinImage(input []InputItem, img llmprovider.ImageItem) []InputItem {
	url := img.URL
	if url == "" {
		url = wire.DataURL(img)
	}
	part := contentPart{Type: "input_image", ImageURL: &url}
	if n := len(input); n > 0 && input[n-1].Type == nil && input[n-1].Role != nil && *input[n-1].Role == wire.RoleUser {
		last := &input[n-1]
		if last.Content.parts == nil {
			last.Content.parts = []contentPart{{Type: "input_text", Text: last.Content.text}}
		}
		last.Content.parts = append(last.Content.parts, part)
		return input
	}
	return append(input, InputItem{Role: &roleUser, Content: MessageContent{parts: []contentPart{part}}})
}

// appendInput is input with item sent as Input sends it: an image joined to
// the message before it, any other item as an item of its own.
func appendInput(input []InputItem, item llmprovider.Item) []InputItem {
	if img, ok := item.(llmprovider.ImageItem); ok {
		return joinImage(input, img)
	}
	return append(input, Input([]llmprovider.Item{item})...)
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
			input = append(input, InputItem{Role: role, Content: MessageContent{text: arena.Keep(v.Text)}})
		case llmprovider.ImageItem:
			input = joinImage(input, v)
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
			input = appendInput(input, v)
		}
	}
	return input
}

// SearchLoaded is the names of the caller's tools InputLoading sends inside
// the tool_search_output items of items, once each, in order: the tools a
// request holds outside its tools list (0084-MADR D4). It reads what
// InputLoading encoded, so it names exactly the definitions that went out.
func SearchLoaded(items []llmprovider.Item, req *llmprovider.Request) []string {
	var out []string
	for _, in := range InputLoading(items, req) {
		if in.Tools == nil {
			continue
		}
		for _, tool := range *in.Tools {
			if name, ok := tool[wire.KeyName].(string); ok && name != "" && !slices.Contains(out, name) {
				out = append(out, name)
			}
		}
	}
	return out
}

// InputDispatch is Input for a request whose deferred tools go out through
// deferral.UseTool (0040-MADR D5). A call to a deferred tool is sent as the
// call to the wrapper the model made, with tool_name and tool_input; a
// result that ToolLoadItems follow is sent with its text, then the loaded
// tools' definitions (name, description, input_schema) as JSON, as
// grok-build's search returns them. Every other item is sent as Input sends
// it.
func InputDispatch(items []llmprovider.Item, req *llmprovider.Request) []InputItem {
	dispatched, loads := deferral.Dispatched(req.Tools), deferral.Loads(items)
	defs := map[string]llmprovider.Tool{}
	for _, tool := range req.Tools {
		defs[tool.Name] = tool
	}
	var input []InputItem
	for _, item := range items {
		switch v := item.(type) {
		case llmprovider.FunctionCallItem:
			if dispatched[v.Name] {
				v.Arguments, v.Name = useToolArguments(v.Name, v.Arguments), deferral.UseTool
			}
			input = append(input, Input([]llmprovider.Item{v})...)
		case llmprovider.FunctionCallOutputItem:
			if names := loads[v.CallID]; len(names) > 0 {
				v.Output = loadedText(v.Output, names, defs)
			}
			input = append(input, Input([]llmprovider.Item{v})...)
		default:
			input = appendInput(input, v)
		}
	}
	return input
}

// useToolArguments is a call to name, with arguments, as a call to
// deferral.UseTool: {"tool_name":…,"tool_input":…}. Arguments that are not a
// JSON object are sent as {}, as a tool_search_call's are.
func useToolArguments(name, arguments string) string {
	input := json.RawMessage("{}")
	if trimmed := strings.TrimSpace(arguments); json.Valid([]byte(trimmed)) && strings.HasPrefix(trimmed, "{") {
		input = json.RawMessage(trimmed)
	}
	out, err := json.Marshal(struct {
		ToolName  string          `json:"tool_name"`
		ToolInput json.RawMessage `json:"tool_input"`
	}{name, input})
	if err != nil {
		// Unreachable: a string and a valid object always encode.
		return `{"tool_name":"","tool_input":{}}`
	}
	return string(out)
}

// loadedText is a result's text, then a blank line and the named tools'
// definitions as a JSON array.
func loadedText(output string, names []string, defs map[string]llmprovider.Tool) string {
	type definition struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		InputSchema any    `json:"input_schema"`
	}
	list := make([]definition, 0, len(names))
	for _, name := range names {
		tool := defs[name]
		list = append(list, definition{Name: tool.Name, Description: tool.Description, InputSchema: wire.ToolSchema(tool.Schema)})
	}
	raw, err := json.Marshal(list)
	if err != nil {
		// Unreachable: deferral.CheckDispatch refuses a deferred tool whose
		// schema does not encode. The names still say what was loaded.
		raw = []byte("Loaded: " + strings.Join(names, ", "))
	}
	if output == "" {
		return string(raw)
	}
	return output + "\n\n" + string(raw)
}

// DecodeFor is Decode for provider, the label the caller gives wire.Call: a
// failed response names provider, and is classified by that service's error
// vocabulary, as ReadStream's is (0026-MADR F28).
func DecodeFor(provider string) func(io.Reader) (*llmprovider.Response, error) {
	return Decoder{}.DecodeFor(provider)
}

// Decoder decodes Responses replies for one request.
type Decoder struct {
	// Search names the request's Search tool, when its deferred tools go
	// out natively: a client tool_search_call then decodes as a
	// FunctionCallItem calling it, its arguments the object as JSON text.
	// With no Search, such a call is dropped, as every unknown item is
	// (0040-MADR D3).
	Search string
	// Dispatch holds the deferred tools' names, when they go out through
	// deferral.UseTool: a call to the wrapper whose tool_name is one of them
	// decodes as that tool's call, with tool_input as its arguments. Any
	// other call to the wrapper stays one (0040-MADR D5).
	Dispatch map[string]bool
	// Logger is the provider's WithLogger logger. A stream that took items
	// from its terminal response, which no output_item.done delivered,
	// writes one Debug line to it: the provider, the terminal event's type
	// and the count taken of each item type, never an item's text, id or
	// arguments. Nil writes nothing (0092-MADR ST20).
	Logger *slog.Logger
	// Limit is the output limit the request's attempt sent, set only by
	// chatcompat's Responses path (0091-MADR ST33, ST55): a reply the
	// decoder would finish as completed, its status completed, absent or
	// any other but failed and incomplete (ST69), whose output_tokens
	// reached Limit is read as one the limit cut, as a reply incomplete for
	// max_output_tokens is. On a stream it is read before the held calls
	// are released and before an announced item's check (ST34, ST65). A
	// refusal is read first and finishes content_filter (ST70); a call
	// announced and never delivered is then named in a WarningToolCallCut
	// (ST74). Zero, as openai, grok and OpenCode leave it, reads no reply as
	// cut (ST68).
	Limit int
}

// atLimit reports a reply whose output_tokens, u's, reached d.Limit, so
// that the limit cut it however its service finished it (0091-MADR ST55).
func (d Decoder) atLimit(u usage) bool { return d.Limit > 0 && u.Output >= d.Limit }

// DecodeFor is the package's DecodeFor, with d's Search.
func (d Decoder) DecodeFor(provider string) func(io.Reader) (*llmprovider.Response, error) {
	return func(body io.Reader) (*llmprovider.Response, error) { return decodeAs(body, provider, d) }
}

// ReadStream is the package's ReadStream, with d's Search.
func (d Decoder) ReadStream(provider string, body io.Reader) (*llmprovider.Response, error) {
	var result *llmprovider.Response
	err := readEvents(provider, body, handledEvents, d, func(ev llmprovider.Event) bool {
		if ev.Type == llmprovider.EventDone {
			result = ev.Response
		}
		return true
	})
	return finished(result, err)
}

// Events is the package's Events, with d's Search.
func (d Decoder) Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	return readEvents(provider, body, streamedEvents, d, yield)
}

// Decode decodes a Responses API JSON body into a Response.
func Decode(body io.Reader) (*llmprovider.Response, error) {
	return decodeAs(body, "responses", Decoder{})
}

// decodeAs is Decode, with a failure labelled and classified for provider.
func decodeAs(body io.Reader, provider string, d Decoder) (*llmprovider.Response, error) {
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
	switch {
	case raw.Status == statusFailed:
		// The status is read first, as ReadStream reads response.failed.
		return nil, failed(provider, raw.Error)
	case raw.Status == statusIncomplete && raw.IncompleteDetails.Reason != reasonOutputLimit:
		// An answer incomplete for another reason, such as a content
		// filter, is an error, never an empty success (MADR 0012 §1.5;
		// 0088-MADR ST81).
		return nil, incomplete(raw.IncompleteDetails.Reason)
	}

	result := &llmprovider.Response{ID: raw.ID, Model: raw.Model, Usage: raw.Usage.counts()}
	refused := false
	for _, out := range raw.Output {
		refused = appendOutput(result, out, d) || refused
	}
	if raw.Status == statusIncomplete || !refused && d.atLimit(raw.Usage) {
		// A call's arguments stream in pieces (0088-MADR ST107). A reply
		// at the limit sent is cut whatever its status said (0091-MADR
		// ST55, ST69), unless it is a refusal, which is read first (ST70).
		return wire.CutAnswer("responses", result, reasonOutputLimit, true)
	}
	return completed(result, refused)
}

// reasonOutputLimit is the incomplete reason of an answer the output limit
// cut, which is returned with what is usable (0088-MADR ST75 to ST79).
const reasonOutputLimit = "max_output_tokens"

// statusCut is an output item's status when the answer was cut inside it: a
// call of that status was cut before its arguments were whole (0088-MADR
// ST78).
const statusCut = "incomplete"

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
// tool_calls when it calls a tool, else stop (0020-MADR F11; 0021-MADR W4).
// One the output limit cut finishes length (CutAnswer); one incomplete for
// another reason is an error, never a Response.
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

// incomplete is the MADR 0012 §1.5 error for a Responses answer incomplete
// for a reason other than the output limit (0088-MADR ST81). The reason is
// the service's text, so it is stripped and bounded (0026-MADR F18).
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
	// ID is the item's own id, which a stream's events name it by.
	ID     string `json:"id"`
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	// Status is the item's: "incomplete" when the answer was cut inside it.
	Status string `json:"status"`
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
func appendOutput(r *llmprovider.Response, out outputItem, d Decoder) (refused bool) {
	// A call the output limit cut is dropped and named (0088-MADR ST77,
	// ST78).
	if out.Status == statusCut && (out.Type == itemTypeFunctionCall || out.Type == itemTypeToolSearchCall) {
		r.Warnings = append(r.Warnings, wire.ToolCallCut(r.Model, out.Name, out.CallID))
		return false
	}
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
		name := out.Name
		if name == deferral.UseTool && len(d.Dispatch) > 0 {
			name, args = d.dispatched(args)
		}
		r.Output = append(r.Output, llmprovider.FunctionCallItem{
			CallID:    out.CallID,
			Name:      name,
			Arguments: args,
		})
	case itemTypeToolSearchCall:
		// A client search the caller runs, as a call to its Search tool; a
		// hosted one, or one with no Search tool to name, is dropped
		// (0040-MADR D3).
		if d.Search == "" || out.Execution != executionClientValue {
			break
		}
		args := argumentsText(out.Arguments)
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		r.Output = append(r.Output, llmprovider.FunctionCallItem{CallID: out.CallID, Name: d.Search, Arguments: args})
	case itemTypeReasoning:
		var sb strings.Builder
		for _, s := range out.Summary {
			if s.Type == "summary_text" || s.Type == wire.KeyText {
				sb.WriteString(s.Text)
			}
		}
		// A server that sends its reasoning as text puts it in content, as
		// reasoning_text: it is read when the summary holds none (0091-MADR
		// ST10, ST63). Its text is never sent back; only encrypted content is.
		if sb.Len() == 0 {
			for _, c := range out.Content {
				if c.Type == contentReasoningText {
					sb.WriteString(c.Text)
				}
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

// dispatched is a call to deferral.UseTool with arguments, as the call it
// makes: the deferred tool tool_name names, with tool_input, an object or a
// string holding one, as its arguments, "{}" when there is none. A call naming
// no tool in d.Dispatch, or with input that is not an object, stays a call to
// the wrapper, with its arguments as they came (0040-MADR D5).
func (d Decoder) dispatched(arguments string) (name, args string) {
	var call struct {
		ToolName  string          `json:"tool_name"`
		ToolInput json.RawMessage `json:"tool_input"`
	}
	if json.Unmarshal([]byte(arguments), &call) != nil || !d.Dispatch[call.ToolName] {
		return deferral.UseTool, arguments
	}
	input := argumentsText(call.ToolInput)
	switch trimmed := strings.TrimSpace(input); {
	case trimmed == "":
		return call.ToolName, "{}"
	case strings.HasPrefix(trimmed, "{") && json.Valid([]byte(trimmed)):
		var compact bytes.Buffer
		if json.Compact(&compact, []byte(trimmed)) == nil {
			return call.ToolName, compact.String()
		}
	}
	return deferral.UseTool, arguments
}

// streamEvent is the part of one Responses stream event the reader uses. The
// type is read from the payload, not the SSE "event:" line.
type streamEvent struct {
	Type string `json:"type"`
	// Code and Message are a top-level error event's (0020-MADR F39).
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Item    outputItem `json:"item"`
	// OutputIndex is an output_item.added's or .done's place in the
	// response's output, nil when the event names none (0092-MADR D2); it is
	// read by outputIndex, so a value of another form never fails the event.
	OutputIndex json.RawMessage `json:"output_index"`
	Response    struct {
		ID                string         `json:"id"`
		Model             string         `json:"model"`
		Error             *responseError `json:"error"`
		IncompleteDetails struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		// Output is the terminal response's items, read on a cut (0088-MADR
		// ST104) and on a completed stream, for the items no
		// output_item.done delivered (0092-MADR D1).
		Output []outputItem `json:"output"`
		Usage  usage        `json:"usage"`
	} `json:"response"`
}

// responseError is a failed response's error, and the object as it came.
type responseError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
	// raw is the error object, which the classifier reads for its headers,
	// reset time and limit (0072-MADR D3).
	raw json.RawMessage
}

// UnmarshalJSON decodes the error's fields and keeps the object.
func (e *responseError) UnmarshalJSON(b []byte) error {
	type fields responseError
	var f fields
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	*e = responseError(f)
	e.raw = append(json.RawMessage(nil), b...)
	return nil
}

// failed classifies a failed response, streamed or not, by its error as
// Codex does; one with no error is retryable (0026-MADR F28).
func failed(provider string, e *responseError) error {
	if e == nil {
		return llmprovider.ClassifyStreamFailure(provider, "", "", "response.failed event received")
	}
	return llmprovider.ClassifyStreamFailureBody(provider, e.Code, e.Type, e.Message, e.raw)
}

// ReadStream reads a Responses API event stream into a Response, as Codex
// does (codex-api/src/sse/responses.rs:343-450): each
// response.output_item.done adds its item, and response.created and
// response.completed carry the id. At response.completed, and at
// response.incomplete for the output limit, the answer also takes each item
// the terminal response carries that no output_item.done delivered, after
// the delivered ones (0092-MADR D1, D2; 0088-MADR ST104).
// response.failed maps onto MADR 0012 §1.1; response.incomplete for the
// output limit is the answer it cut, finishing length (0088-MADR ST75 to
// ST80), and for another reason MADR 0012 §1.5's error; and a stream that
// ends before response.completed is retryable, once (MADR 0012 §4.1;
// 0021-MADR D1). So is a completed stream that announced a message, a call
// or a client tool search that neither an output_item.done nor the terminal
// response delivered: ErrProviderUnavailable, marked after the reply, naming
// item_not_finished with the item's type and id (0092-MADR ST11). Each
// event is bounded, not the stream (0021-MADR W2). It does not rely on
// Content-Type, which the ChatGPT backend does not send. It collects Events,
// skipping the deltas it does not need (0031-MADR D2).
func ReadStream(provider string, body io.Reader) (*llmprovider.Response, error) {
	return Decoder{}.ReadStream(provider, body)
}

// Events reads a Responses API event stream and yields its events as they
// arrive (0031-MADR D2): a text delta as EventTextDelta, a reasoning summary
// delta (0031-PLAN Phase 1) and a reasoning text delta (0091-MADR ST63) as
// EventReasoningDelta, each
// response.output_item.done's item as EventItem, and response.completed, or
// response.incomplete for the output limit, as EventDone, carrying the
// Response ReadStream returns. A call whose arguments are {} or not whole is
// held until a later item or the end (wire.HeldCalls), so a call the limit
// cut is never yielded (0088-MADR ST80). The items it takes from the
// terminal response, which no output_item.done delivered, are yielded after
// the held calls and before EventDone, with no delta of their own (0092-MADR
// D1). A call-argument delta and every other event yield nothing (0031-MADR
// Q4). It returns nil once EventDone is yielded or yield returns false, and
// otherwise the error that ended the stream, as ReadStream's, among them
// ErrProviderUnavailable naming item_not_finished, for an announced item
// never delivered (0092-MADR ST11); what it yielded before an error is not a
// response (0031-MADR D4).
func Events(provider string, body io.Reader, yield func(llmprovider.Event) bool) error {
	return Decoder{}.Events(provider, body, yield)
}

// readEvents is Events, decoding only the event types in handled.
func readEvents(provider string, body io.Reader, handled map[string]bool, d Decoder, yield func(llmprovider.Event) bool) error {
	s := &streamState{result: &llmprovider.Response{}, d: d, provider: provider}
	result := s.result
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
			return false, llmprovider.ClassifyStreamFailureBody(provider, event.Code, "", event.Message, payload)
		case "response.output_item.added":
			if event.Item.Type == itemTypeFunctionCall || event.Item.Type == itemTypeToolSearchCall {
				s.open = append(s.open, event.Item)
			}
			s.announce(event.Item, outputIndex(event.OutputIndex))
		case "response.output_item.done":
			return s.itemDone(event.Item, outputIndex(event.OutputIndex), yield), nil
		case "response.failed":
			return false, failed(provider, event.Response.Error)
		case "response.incomplete":
			if event.Response.IncompleteDetails.Reason != reasonOutputLimit {
				return false, incomplete(event.Response.IncompleteDetails.Reason)
			}
			return s.end(event, true, yield)
		case "response.completed":
			return s.end(event, false, yield)
		}
		return false, nil
	}
	// The framing, the per-event bound and the early end are the shared
	// reader's (0031-MADR E2).
	return wire.ReadEvents(provider, body, "response.completed", dispatch)
}

// streamState is a Responses stream's answer as its events arrive.
type streamState struct {
	result   *llmprovider.Response
	d        Decoder
	provider string
	refused  bool
	// open are the calls added and not yet done: at an output limit's cut,
	// each is a call the limit cut (0088-MADR ST77). done are the items an
	// output_item.done delivered, in order, each as pairing reads it
	// (0092-MADR D2). announced are the items an output_item.added announced
	// of the types a completed stream must deliver (ST11). held are the
	// done calls a cut could drop, not yet yielded (0088-MADR ST80).
	open      []outputItem
	done      []doneItem
	announced []announcedItem
	// announcedKeys are the announced items' keys, by id, call id and
	// output_index with type, so a repeat is found without a scan (ST34,
	// ST38).
	announcedKeys map[string]bool
	held          wire.HeldCalls
}

// ref is an item as pairing reads it: its id, "" for none; its place in the
// response's output, -1 for none; and its type (0092-MADR D2).
type ref struct {
	id    string
	index int
	typ   string
}

// doneItem is an output_item.done's item; empty when it added nothing to
// the answer, a message or a reasoning item with nothing in it, so that its
// terminal partner is taken (0092-MADR ST12).
type doneItem struct {
	ref
	empty bool
}

// announcedItem is an output_item.added's item, with its call id, which
// names the item when it has no id of its own.
type announcedItem struct {
	ref
	callID string
}

// outputIndex is an event's output_index: an integer JSON number at least
// 0, else -1, none, as an absent one is; 1.0 or "1" is none, never an error
// (0092-MADR ST35).
func outputIndex(raw json.RawMessage) int {
	n, err := strconv.Atoi(string(bytes.TrimSpace(raw)))
	if err != nil || n < 0 {
		return -1
	}
	return n
}

// announce records an announced item of a type a completed stream must
// deliver: a message, a call, or a client tool search when the request has
// a Search tool, the types appendOutput keeps; not a reasoning item, a
// hosted search or an unknown type (0092-MADR ST11). A repeated
// announcement is the same item, recorded once: the same id (ST34); else,
// with no id, the same call id; else, with neither, the same output_index
// and type (ST38). One with none of the three is recorded each time. Once
// the announced items pass wire.MaxStreamIndex, where ST11's check is off
// (ST33), no more are recorded, so each announcement costs constant work.
func (s *streamState) announce(item outputItem, index int) {
	switch item.Type {
	case itemTypeMessage, itemTypeFunctionCall:
	case itemTypeToolSearchCall:
		if s.d.Search == "" || item.Execution != executionClientValue {
			return
		}
	default:
		return
	}
	if len(s.announced) > wire.MaxStreamIndex || s.announcedKeys[announceKey(item, index)] {
		return
	}
	if s.announcedKeys == nil {
		s.announcedKeys = map[string]bool{}
	}
	if item.ID != "" {
		s.announcedKeys["id:"+item.ID] = true
	}
	if item.CallID != "" {
		s.announcedKeys["call:"+item.CallID] = true
	}
	if item.ID == "" && item.CallID == "" && index >= 0 {
		s.announcedKeys["at:"+strconv.Itoa(index)+":"+item.Type] = true
	}
	s.announced = append(s.announced, announcedItem{ref: ref{id: item.ID, index: index, typ: item.Type},
		callID: item.CallID})
}

// announceKey is the key a repeat of item, announced at index, is found by:
// its id; else its call id; else its output_index and type; "" for none
// (0092-MADR ST34, ST38).
func announceKey(item outputItem, index int) string {
	switch {
	case item.ID != "":
		return "id:" + item.ID
	case item.CallID != "":
		return "call:" + item.CallID
	case index >= 0:
		return "at:" + strconv.Itoa(index) + ":" + item.Type
	}
	return ""
}

// closes reports whether item, finished by an output_item.done or taken
// from the terminal response, is the open call o: by call id when both
// carry one; else by id, when item carries one (0088-MADR ST115, ST116). Two
// different call ids never match through an id. A message's or a reasoning
// item's .done, which carries no call id, closes no call announced without
// one unless their ids match.
func closes(o, item outputItem) bool {
	if item.CallID != "" && o.CallID != "" {
		return o.CallID == item.CallID
	}
	return item.ID != "" && o.ID == item.ID
}

// itemDone adds the item an output_item.done delivers, and yields it unless
// it is held. It reports true when yield stopped the stream.
func (s *streamState) itemDone(item outputItem, index int, yield func(llmprovider.Event) bool) bool {
	s.open = slices.DeleteFunc(s.open, func(o outputItem) bool { return closes(o, item) })
	done := doneItem{ref: ref{id: item.ID, index: index, typ: item.Type}}
	if !s.held.Later(yield) {
		s.done = append(s.done, done)
		return true
	}
	n := len(s.result.Output)
	s.refused = appendOutput(s.result, item, s.d) || s.refused
	done.empty = (item.Type == itemTypeMessage || item.Type == itemTypeReasoning) && len(s.result.Output) == n
	s.done = append(s.done, done)
	if len(s.result.Output) > n && !s.held.Hold(s.result.Output[n]) {
		return !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: s.result.Output[n]})
	}
	return false
}

// pairRefs pairs each of a with at most one of b, in 0092-MADR D2's three
// passes, each over what the earlier ones left unpaired: by id, where both
// carry one; by index, where both carry one and are of one type; and in
// order within each type, the k-th unpaired of a type in a with the k-th
// unpaired of that type in b. match[i] is a[i]'s partner in b, -1 for none.
// With oq5, a type with k ≥ 1 unpaired in a and more than k unpaired in b
// is paired no further, and is named in ambiguous (OQ5).
func pairRefs(a, b []ref, oq5 bool) (match []int, ambiguous map[string]bool) {
	match = make([]int, len(a))
	used := make([]bool, len(b))
	for i := range match {
		match[i] = -1
	}
	pair := func(i int, ok func(y ref) bool) {
		for j, y := range b {
			if !used[j] && ok(y) {
				match[i], used[j] = j, true
				return
			}
		}
	}
	for i, x := range a {
		if x.id != "" {
			pair(i, func(y ref) bool { return y.id == x.id })
		}
	}
	for i, x := range a {
		if match[i] < 0 && x.index >= 0 {
			pair(i, func(y ref) bool { return y.index == x.index && y.typ == x.typ })
		}
	}
	var types []string
	for _, r := range slices.Concat(a, b) {
		if !slices.Contains(types, r.typ) {
			types = append(types, r.typ)
		}
	}
	for _, typ := range types {
		var ua, ub []int
		for i, x := range a {
			if match[i] < 0 && x.typ == typ {
				ua = append(ua, i)
			}
		}
		for j, y := range b {
			if !used[j] && y.typ == typ {
				ub = append(ub, j)
			}
		}
		if oq5 && len(ua) >= 1 && len(ub) > len(ua) {
			if ambiguous == nil {
				ambiguous = map[string]bool{}
			}
			ambiguous[typ] = true
			continue
		}
		for k := range min(len(ua), len(ub)) {
			match[ua[k]], used[ub[k]] = ub[k], true
		}
	}
	return match, ambiguous
}

// terminalRefs are the terminal response's items as pairing reads them, each
// at its place in the output.
func terminalRefs(output []outputItem) []ref {
	refs := make([]ref, len(output))
	for i, item := range output {
		refs[i] = ref{id: item.ID, index: i, typ: item.Type}
	}
	return refs
}

// doneRefs are the delivered items as pairing reads them.
func (s *streamState) doneRefs() []ref {
	refs := make([]ref, len(s.done))
	for i, d := range s.done {
		refs[i] = d.ref
	}
	return refs
}

// takeTerminal adds each item the terminal response carries that no
// output_item.done delivered, on a completed stream as on a cut (0092-MADR
// D1; 0088-MADR ST104): a service such as xAI ends a cut stream with
// response.incomplete and no .done events, and a compatible server may omit
// them on a completed one. The .done items and the terminal's are paired by
// D2's passes; an unpaired terminal item is taken, and so is the partner of
// a .done that added nothing (ST12), but nothing of a type OQ5's case makes
// ambiguous (ST6 (f)); a .done item that added one stands against a
// differing partner (ST7). Taken items go through appendOutput, after the
// delivered ones, in the terminal's order (ST9). A call it takes is no
// longer an open one; the cut rules then apply to it as to any other. It
// returns the items it added, and their count by type. Past the bound,
// nothing is taken (ST33).
func (s *streamState) takeTerminal(output []outputItem) ([]llmprovider.Item, map[string]int) {
	if !s.bounded(output) {
		return nil, nil
	}
	match, ambiguous := pairRefs(s.doneRefs(), terminalRefs(output), true)
	take := make([]bool, len(output))
	for j := range take {
		take[j] = !ambiguous[output[j].Type]
	}
	for i, j := range match {
		if j >= 0 && !s.done[i].empty {
			take[j] = false
		}
	}
	n := len(s.result.Output)
	var counts map[string]int
	for j, item := range output {
		if !take[j] {
			continue
		}
		s.open = slices.DeleteFunc(s.open, func(o outputItem) bool { return closes(o, item) })
		before := len(s.result.Output)
		s.refused = appendOutput(s.result, item, s.d) || s.refused
		if len(s.result.Output) > before {
			if counts == nil {
				counts = map[string]int{}
			}
			counts[item.Type]++
		}
	}
	return slices.Clone(s.result.Output[n:]), counts
}

// bounded reports whether the items pairing reads are within
// wire.MaxStreamIndex, the bound every wire puts on a stream's indexes: the
// terminal's items, the .done items and the announced ones. Past it, the
// stream keeps what the .done events gave, and ST11's check is not run
// (0092-MADR ST33).
func (s *streamState) bounded(output []outputItem) bool {
	return len(output) <= wire.MaxStreamIndex && len(s.done) <= wire.MaxStreamIndex &&
		len(s.announced) <= wire.MaxStreamIndex
}

// logTaken writes one Debug line through the decoder's logger for a stream
// that took items: the provider, the terminal event's type, and the count
// taken of each type; never an item's text, id or arguments (0092-MADR
// ST20).
func (s *streamState) logTaken(event string, counts map[string]int) {
	if s.d.Logger == nil || len(counts) == 0 {
		return
	}
	args := []any{"provider", s.provider, "event", event}
	for _, typ := range []string{itemTypeReasoning, itemTypeMessage, itemTypeFunctionCall, itemTypeToolSearchCall} {
		if n := counts[typ]; n > 0 {
			args = append(args, typ, n)
		}
	}
	s.d.Logger.Debug("llmprovider: "+s.provider+": items taken from the terminal response", args...)
}

// unfinished is the error for a completed stream that announced an item
// neither an output_item.done nor the terminal response delivered: an
// announced item is paired, by D2's passes, with the delivered items and
// the terminal's items no .done delivered, and the first left unpaired is
// named. It is ErrProviderUnavailable, marked after the reply as an early
// end is, naming item_not_finished with the item's type and id, each the
// service's, so stripped and bounded (0092-MADR ST11). Nil when every
// announced item was delivered, or when the items are past the bound (ST33).
func (s *streamState) unfinished(output []outputItem) error {
	if len(s.announced) == 0 || !s.bounded(output) {
		return nil
	}
	dones, terminal := s.doneRefs(), terminalRefs(output)
	paired, _ := pairRefs(dones, terminal, true)
	delivered := slices.Clone(dones)
	for j, r := range terminal {
		if !slices.Contains(paired, j) {
			delivered = append(delivered, r)
		}
	}
	announced := make([]ref, len(s.announced))
	for i, a := range s.announced {
		announced[i] = a.ref
	}
	match, _ := pairRefs(announced, delivered, false)
	for i, j := range match {
		if j >= 0 {
			continue
		}
		a := s.announced[i]
		detail := "item_not_finished: " + redact.Field(a.typ)
		if id := redact.Field(cmp.Or(a.id, a.callID)); id != "" {
			detail += " " + id
		}
		return fmt.Errorf("%w: %s: %w", llmprovider.ErrProviderUnavailable, s.provider,
			transport.AfterReply(errors.New(detail)))
	}
	return nil
}

// end finishes the stream at its last event, response.completed, or
// response.incomplete for the output limit when cut: the result takes the
// event's id, model and usage, and the terminal response's undelivered
// items (0092-MADR D1), before any decision on how to finish (ST14). A cut
// answer names each call added and never done as the call the limit cut,
// and is finished by CutAnswer, the reply's last item read in the server's
// order (0092-MADR ST37); a refusal at the limit sent names them too, and
// finishes content_filter (0091-MADR ST74); a completed one fails when an
// announced item was never delivered (ST11), and is otherwise finished by
// completed; then the held calls and the taken items the response kept are
// yielded, and EventDone (0088-MADR ST77, ST80). Its results are a stream
// dispatch's.
func (s *streamState) end(event streamEvent, cut bool, yield func(llmprovider.Event) bool) (bool, error) {
	result := s.result
	if event.Response.ID != "" {
		result.ID = event.Response.ID
	}
	if event.Response.Model != "" {
		result.Model = event.Response.Model
	}
	result.Usage = event.Response.Usage.counts()
	taken, counts := s.takeTerminal(event.Response.Output)
	s.logTaken(event.Type, counts)
	// A completed stream at the limit sent is a cut, decided before the
	// announced items' check (0091-MADR ST55, ST65); a refusal is read
	// first, and is not one (ST70).
	atLimit := s.d.atLimit(event.Response.Usage)
	cut = cut || !s.refused && atLimit
	var resp *llmprovider.Response
	var err error
	switch {
	case cut:
		s.nameOpen(result)
		resp, err = wire.CutAnswerLast("responses", result, reasonOutputLimit, true, s.lastIndex(event.Response.Output, taken))
	case s.refused && atLimit:
		// The refusal finishes the reply; a call announced and never
		// delivered is the limit's, named (0091-MADR ST74).
		s.nameOpen(result)
		resp, err = completed(result, true)
	default:
		if err := s.unfinished(event.Response.Output); err != nil {
			return false, err
		}
		resp, err = completed(result, s.refused)
	}
	if err != nil {
		return false, err
	}
	if !s.held.Done(resp, yield) {
		return true, nil
	}
	for _, item := range taken {
		if slices.Contains(resp.Output, item) && !yield(llmprovider.Event{Type: llmprovider.EventItem, Item: item}) {
			return true, nil
		}
	}
	yield(llmprovider.Event{Type: llmprovider.EventDone, Response: resp})
	return true, nil
}

// nameOpen names each call added and never done in a WarningToolCallCut
// on result (0088-MADR ST77, ST108).
func (s *streamState) nameOpen(result *llmprovider.Response) {
	for _, o := range s.open {
		result.Warnings = append(result.Warnings, wire.ToolCallCut(result.Model, o.Name, o.CallID))
	}
}

// lastIndex is the place in s.result.Output of the reply's last item, where
// the limit fell, in the server's order (0092-MADR ST37): the terminal
// output's last item. When it is a call, it is the response's call with its
// call id; when it is not a call, no item is last, -1. With no terminal
// output, past the bound (ST33), or a last call whose call id the response
// does not hold, it is the response's own last item, as before. The cut
// applies only to a call not yet yielded, one still held or one taken from
// the terminal: a call a later .done released was yielded, so it is kept,
// and no item is last, -1 (ST9; 0088-MADR ST109).
func (s *streamState) lastIndex(output []outputItem, taken []llmprovider.Item) int {
	fallback := len(s.result.Output) - 1
	if len(output) == 0 || !s.bounded(output) {
		return fallback
	}
	last := output[len(output)-1]
	if last.Type != itemTypeFunctionCall && last.Type != itemTypeToolSearchCall {
		return -1
	}
	if last.CallID == "" {
		return fallback
	}
	for i, item := range s.result.Output {
		if call, ok := item.(llmprovider.FunctionCallItem); ok && call.CallID == last.CallID {
			if !s.held.Holds(item) && !slices.Contains(taken, item) {
				return -1
			}
			return i
		}
	}
	return fallback
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
	"response.created": true, "response.output_item.added": true, "response.output_item.done": true,
	"response.failed": true, "response.incomplete": true, "response.completed": true,
}

// deltaEvents are the delta event types Events yields, and the Event type of
// each. The reasoning summary delta is the one 0031-PLAN Phase 1 measured;
// the reasoning text delta, which probes of 2026-11-23 saw, is passed on on
// every provider (0091-MADR ST63, ST67). A
// refusal's delta is a text delta, since appendOutput keeps a refusal's text
// as the answer's (0021-MADR W4); its name is OpenAI's documented one (0031-MADR,
// amendment of 2026-10-10).
var deltaEvents = map[string]llmprovider.EventType{
	"response.output_text.delta":            llmprovider.EventTextDelta,
	"response.refusal.delta":                llmprovider.EventTextDelta,
	"response.reasoning_summary_text.delta": llmprovider.EventReasoningDelta,
	"response.reasoning_text.delta":         llmprovider.EventReasoningDelta,
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
