package llmprovider

import (
	"context"
	"fmt"
	"strings"
)

// Provider is the one generation contract every provider implements
// (0015-MADR D3). A new request feature is a new field on Request or Response,
// never a new method. A Provider is immutable after construction and safe for
// concurrent use.
type Provider interface {
	// ID returns the provider's canonical identifier.
	ID() ProviderID
	// Capabilities declares what the provider supports.
	Capabilities() Capabilities
	// Generate sends req and returns the service's response. A request that
	// needs an Unsupported capability fails before any network call.
	Generate(ctx context.Context, req *Request) (*Response, error)
}

// ModelLister is implemented by a provider that can list the models it
// serves (0015-MADR D3, amendment of 2026-09-30). ListModels returns the
// curated listing, probed where 0016-MADR A5 says; a caller type-asserts to
// it, as to Streamer.
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// ProviderID is a provider's canonical identifier: the models.dev registry
// key (0015-MADR D6). The ProviderGemini family of constants keeps its
// untyped form until 0015-PLAN S8, commit 5.
type ProviderID string

// Role is the author of a MessageItem (0015-MADR D6). MessageItem.Role keeps
// its string type until 0015-PLAN S8, commit 5.
type Role string

const (
	// RoleUser is the person or program asking.
	RoleUser Role = "user"
	// RoleAssistant is the model.
	RoleAssistant Role = "assistant"
	// RoleSystem carries instructions inside the input.
	RoleSystem Role = "system"
)

// Effort is a reasoning effort level (0015-MADR D6).
type Effort string

const (
	// EffortLow asks for little reasoning.
	EffortLow Effort = effortLow
	// EffortMedium asks for moderate reasoning.
	EffortMedium Effort = effortMedium
	// EffortHigh asks for extensive reasoning.
	EffortHigh Effort = effortHigh
	// EffortXHigh asks for the most reasoning a provider offers.
	EffortXHigh Effort = effortXHigh
)

// valid reports whether e is one of the Effort constants.
func (e Effort) valid() bool {
	switch e {
	case EffortLow, EffortMedium, EffortHigh, EffortXHigh:
		return true
	default:
		return false
	}
}

// Reasoning requests extended reasoning with either an effort level or a
// token budget. A nil *Reasoning on a Request means none.
type Reasoning struct {
	// Effort is the level to reason at. Empty leaves the provider's default.
	Effort Effort
	// Budget is a token budget for reasoning, for services that take one.
	// Zero leaves the provider's default.
	Budget int
}

// ToolChoice says whether and how the model must call a tool (0015-MADR D3).
// The empty value is ToolChoiceAuto. ForceTool names one tool.
type ToolChoice string

const (
	// ToolChoiceAuto lets the model decide whether to call a tool.
	ToolChoiceAuto ToolChoice = "auto"
	// ToolChoiceNone forbids tool calls.
	ToolChoiceNone ToolChoice = "none"
	// ToolChoiceRequired makes the model call one of the offered tools.
	ToolChoiceRequired ToolChoice = "required"
)

// toolChoicePrefix marks a ToolChoice that names one tool.
const toolChoicePrefix = "tool:"

// ForceTool returns the ToolChoice that makes the model call the named tool.
func ForceTool(name string) ToolChoice {
	return ToolChoice(toolChoicePrefix + name)
}

// Tool returns the tool c forces, and whether c forces one.
func (c ToolChoice) Tool() (string, bool) {
	return strings.CutPrefix(string(c), toolChoicePrefix)
}

// forced reports whether c makes the model call a tool.
func (c ToolChoice) forced() bool {
	_, named := c.Tool()
	return c == ToolChoiceRequired || named
}

// FinishReason says why the model stopped (0015-MADR D6). A service's own
// value that no constant names is kept as the service sent it.
type FinishReason string

const (
	// FinishStop is a natural end.
	FinishStop FinishReason = "stop"
	// FinishLength is the output-token limit; the text so far is kept.
	FinishLength FinishReason = finishReasonLength
	// FinishToolCalls is a stop to let the caller run tool calls.
	FinishToolCalls FinishReason = "tool_calls"
	// FinishContentFilter is a stop by the service's content filter.
	FinishContentFilter FinishReason = "content_filter"
)

// Usage counts a response's tokens. Each is zero when the service reports
// none (0015-MADR D3, D4).
type Usage struct {
	InputTokens     int
	OutputTokens    int
	ReasoningTokens int
	CachedTokens    int
}

// Request is everything one generation needs (0015-MADR D3). A field's zero
// value means the provider's default, or not requested.
type Request struct {
	// Model overrides the model the provider was built with. Empty uses it.
	Model string
	// Instructions is the system prompt.
	Instructions string
	// Input is the conversation so far.
	Input []Item
	// Tools are the functions the model may call.
	Tools []Tool
	// ToolChoice says whether and how the model must call one of Tools.
	ToolChoice ToolChoice
	// Reasoning requests extended reasoning. Nil means none.
	Reasoning *Reasoning
	// MaxOutputTokens bounds the output. Zero is the provider's default.
	MaxOutputTokens int
	// PreviousResponseID continues a conversation the service stored.
	PreviousResponseID string
}

// Check validates req and refuses a request that needs a capability c does
// not have. A provider calls it before any network call. An invalid value
// gives an error matching ErrInvalidRequest (0015-MADR D6); a need c marks
// Unsupported gives one matching ErrUnsupported (0015-MADR D4).
func (c Capabilities) Check(req *Request) error {
	if err := req.validate(); err != nil {
		return err
	}
	for _, need := range []struct {
		needed  bool
		support Support
		name    string
	}{
		{len(req.Tools) > 0, c.Tools, "tools"},
		{req.ToolChoice.forced(), c.ForcedToolChoice, "a forced tool choice"},
		{req.Reasoning != nil, c.Reasoning, "reasoning"},
		{req.PreviousResponseID != "", c.Continuation, "continuation"},
	} {
		if need.needed && need.support == Unsupported {
			return fmt.Errorf("%w: the request needs %s", ErrUnsupported, need.name)
		}
	}
	return nil
}

// validate rejects values no provider can send.
func (req *Request) validate() error {
	if req == nil {
		return fmt.Errorf("%w: nil request", ErrInvalidRequest)
	}
	if req.MaxOutputTokens < 0 {
		return fmt.Errorf("%w: MaxOutputTokens %d is negative", ErrInvalidRequest, req.MaxOutputTokens)
	}
	for i, tool := range req.Tools {
		if tool.Name == "" {
			return fmt.Errorf("%w: tool %d has no name", ErrInvalidRequest, i)
		}
	}
	if err := req.validateToolChoice(); err != nil {
		return err
	}
	if r := req.Reasoning; r != nil {
		if r.Effort != "" && !r.Effort.valid() {
			return fmt.Errorf("%w: unknown reasoning effort %q", ErrInvalidRequest, r.Effort)
		}
		if r.Budget < 0 {
			return fmt.Errorf("%w: reasoning budget %d is negative", ErrInvalidRequest, r.Budget)
		}
	}
	return nil
}

// validateToolChoice rejects an unknown choice, and a forced one with nothing
// to call.
func (req *Request) validateToolChoice() error {
	choice := req.ToolChoice
	name, named := choice.Tool()
	switch {
	case choice == "", choice == ToolChoiceAuto, choice == ToolChoiceNone:
		return nil
	case choice == ToolChoiceRequired:
		if len(req.Tools) == 0 {
			return fmt.Errorf("%w: tool choice %q with no tools", ErrInvalidRequest, choice)
		}
		return nil
	case named:
		for _, tool := range req.Tools {
			if tool.Name == name {
				return nil
			}
		}
		return fmt.Errorf("%w: tool choice names %q, which is not among the tools", ErrInvalidRequest, name)
	default:
		return fmt.Errorf("%w: unknown tool choice %q", ErrInvalidRequest, choice)
	}
}
