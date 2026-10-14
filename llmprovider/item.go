package llmprovider

import (
	"iter"
	"strings"
)

// Item is a sealed interface representing a single typed element in an LLM
// response or input. Only types defined in this package may implement Item;
// the unexported marker method itemKind() prevents external implementations.
//
// Callers switch on concrete types: MessageItem, FunctionCallItem,
// FunctionCallOutputItem, ReasoningItem, and ToolLoadItem, which only a
// request's Input holds.
type Item interface {
	itemKind() string // sealed marker — unexported, prevents external implementation
}

// MessageItem represents a text message from a role (typically "assistant").
type MessageItem struct {
	Role Role // RoleUser, RoleAssistant or RoleSystem
	Text string
}

func (MessageItem) itemKind() string { return itemTypeMessage }

// FunctionCallItem represents the model requesting a function/tool call.
type FunctionCallItem struct {
	CallID    string // provider-issued call identifier
	Name      string // function name
	Arguments string // JSON-encoded arguments
	// Signature is an opaque token the provider issued with the call and
	// requires back when the call is replayed (Gemini's thoughtSignature).
	// Empty when the provider issues none.
	Signature string
}

func (FunctionCallItem) itemKind() string { return itemTypeFunctionCall }

// FunctionCallOutputItem represents the caller's response to a function call.
type FunctionCallOutputItem struct {
	CallID string // matches FunctionCallItem.CallID
	Output string // the function's return value (typically JSON)
	// IsError says the call failed, and Output says how. It is sent where the
	// wire has an error form: Messages' is_error, generateContent's
	// {"error": …}, Interactions' is_error. Elsewhere Output is sent as it
	// is, never rewritten; each provider's package doc says where
	// (0032-MADR D3).
	IsError bool
}

func (FunctionCallOutputItem) itemKind() string { return itemTypeFunctionCallOutput }

// ToolLoadItem loads one deferred tool from a tool result: the caller's
// answer to a call, given in Input after that call's FunctionCallOutputItem.
// One item loads one tool; a result that loads several is followed by
// several. A loaded tool stays loaded for the rest of the conversation, so
// later requests carry the items that loaded it. Each provider sends the
// load in its service's form, or appends the tool to the request's tools, as
// its package doc says. No response holds one (0040-MADR D1, amendment A5).
type ToolLoadItem struct {
	CallID string // the FunctionCallOutputItem's, which comes before it
	Name   string // a Tool of the request with Deferred set
}

func (ToolLoadItem) itemKind() string { return itemTypeToolLoad }

// ReasoningItem represents a model's internal reasoning/thinking trace.
// Not all providers or models emit this; when absent, the Output slice
// simply contains no ReasoningItem values.
type ReasoningItem struct {
	Text string
	// Signature is the service's proof that Text is its own, which it needs
	// back when the reasoning is replayed: Claude's thinking signature
	// (0020-MADR F7, Q1 a). Empty where the service sends none.
	Signature string
	// Encrypted is reasoning the service returned only in encrypted form,
	// replayed as is: Claude's redacted_thinking data, OpenAI's
	// encrypted_content (0020-MADR F7, F24, Q1 a). Text may then be empty.
	Encrypted string
	// Format names the wire that produced the item: "responses",
	// "messages", "chatcompletions" or "generatecontent". Signature and
	// Encrypted are replayed only to the wire that issued them, or when
	// Format is empty, as on an item built before this field; another wire
	// sees the item as text only (0021-MADR D5, W7).
	Format string
}

func (ReasoningItem) itemKind() string { return itemTypeReasoning }

// Response is the result of a generation (0015-MADR D3).
type Response struct {
	// ID is the provider-issued response identifier, used for server-side
	// conversation chaining (OpenAI/xAI: response ID, Gemini: interaction ID,
	// which can be continued only when stored; see gemini.WithStore).
	// Empty for providers that do not support server-side state (Claude).
	ID string

	// Model is the model that answered, as the service names it. Empty where
	// the service does not say.
	Model string

	// Output contains the typed items returned by the model.
	Output []Item

	// FinishReason is the Chat Completions finish_reason, e.g. "stop" or
	// "length"; a length-truncated text answer keeps its text and reports
	// it here (MADR 0012 §1.5). Empty where the service does not report one.
	FinishReason FinishReason

	// Usage counts the response's tokens; zero where the service reports
	// none.
	Usage Usage

	// Warnings are non-fatal notes: the request was answered, but not
	// exactly as asked, such as a forced tool choice sent as auto. Nil when
	// it was (0034-MADR D7).
	Warnings []Warning
}

// OutputText returns the concatenation of all MessageItem texts in Output,
// mirroring the OpenAI/xAI SDK's output_text convenience accessor.
// Returns "" if no MessageItem is present.
func (r *Response) OutputText() string {
	var sb strings.Builder
	for _, item := range r.Output {
		if m, ok := item.(MessageItem); ok {
			sb.WriteString(m.Text)
		}
	}
	return sb.String()
}

// Items returns an iterator over r.Output for use with range-over-func.
func (r *Response) Items() iter.Seq[Item] {
	return func(yield func(Item) bool) {
		for _, item := range r.Output {
			if !yield(item) {
				return
			}
		}
	}
}
