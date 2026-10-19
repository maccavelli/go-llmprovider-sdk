package llmprovider

import (
	"context"
	"encoding/json"
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
// curated listing, in full unless catalog.WithLimit cuts it (0038-MADR D4),
// probed where 0016-MADR A5 says; a caller type-asserts to it, as to
// Streamer. An implementation must be safe for concurrent use, as the
// Provider that has it is.
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// ProviderID is a provider's canonical identifier: the models.dev registry
// key (0015-MADR D6). The ProviderGemini family of constants are its
// built-in values. It is a string, safe for concurrent use.
type ProviderID string

// Role is the author of a MessageItem (0015-MADR D6). An empty Role is the
// user's; any other value outside the constants is refused before any network
// call, with ErrInvalidRequest. It is a string, safe for concurrent use.
type Role string

const (
	// RoleUser is the person or program asking.
	RoleUser Role = "user"
	// RoleAssistant is the model.
	RoleAssistant Role = "assistant"
	// RoleSystem carries instructions inside the input.
	RoleSystem Role = "system"
)

func (r Role) valid() bool {
	switch r {
	case "", RoleUser, RoleAssistant, RoleSystem:
		return true
	default:
		return false
	}
}

// Effort is a reasoning effort level (0015-MADR D6). It is a string, safe for
// concurrent use.
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
// token budget. A nil *Reasoning on a Request means none. It is safe for
// concurrent use only while no goroutine changes it.
type Reasoning struct {
	// Effort is the level to reason at. Empty leaves the provider's default.
	Effort Effort
	// Budget is a token budget for reasoning, for services that take one.
	// Zero leaves the provider's default.
	Budget int
}

// CacheRetention asks the service to cache the request's prompt, for how
// long (0039-MADR D1). The zero value asks for nothing: the request is sent
// as it would be without the field, and a service that caches on its own
// still does. Each provider's package doc says what it sends. It is a string,
// safe for concurrent use.
type CacheRetention string

const (
	// CacheShort is the service's short lifetime: five minutes on Anthropic.
	CacheShort CacheRetention = "short"
	// CacheLong is the longest lifetime the service offers: one hour on
	// Anthropic. Where the service offers one lifetime, it is CacheShort's.
	CacheLong CacheRetention = "long"
)

// valid reports whether r is the zero value or a constant.
func (r CacheRetention) valid() bool {
	switch r {
	case "", CacheShort, CacheLong:
		return true
	default:
		return false
	}
}

// ToolChoice says whether and how the model must call a tool (0015-MADR D3).
// The empty value is ToolChoiceAuto. ForceTool names one tool. It is a
// string, safe for concurrent use.
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
// value that no constant names is kept as the service sent it. It is a
// string, safe for concurrent use.
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
// none (0015-MADR D3, D4). A total holds its parts, whatever the service's
// own convention, so InputTokens + OutputTokens is the response's whole count
// (0015-MADR amendment "what `Usage` counts"). CachedTokens and
// CacheWriteTokens are parts of InputTokens that do not overlap, so
// InputTokens - CachedTokens - CacheWriteTokens is the uncached input
// (0036-MADR D1). It is safe for concurrent use only while no goroutine
// changes it.
type Usage struct {
	// InputTokens is every input token, CachedTokens and CacheWriteTokens
	// included.
	InputTokens int
	// OutputTokens is every output token, ReasoningTokens included.
	OutputTokens int
	// ReasoningTokens is the part of OutputTokens spent reasoning.
	ReasoningTokens int
	// CachedTokens is the part of InputTokens read from the service's cache.
	CachedTokens int
	// CacheWriteTokens is the part of InputTokens written to the service's
	// cache. It does not overlap CachedTokens.
	CacheWriteTokens int
	// CacheWrite1hTokens is the part of CacheWriteTokens written with a
	// one-hour lifetime. Only Anthropic's Messages reports it.
	CacheWrite1hTokens int
}

// Request is everything one generation needs (0015-MADR D3). A field's zero
// value means the provider's default, or not requested. It is safe for
// concurrent use only while no goroutine changes it.
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
	// Cache asks the service to cache the prompt. The zero value asks for
	// nothing (0039-MADR D1).
	Cache CacheRetention
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
		// Every wire sends the schema as JSON, so one that cannot be
		// marshalled is refused here (0026-MADR F20).
		if _, err := json.Marshal(tool.Schema); err != nil {
			return fmt.Errorf("%w: tool %q schema: %w", ErrInvalidRequest, tool.Name, err)
		}
	}
	for i, item := range req.Input {
		// Every encoder drops another type, so it is refused here
		// (0020-MADR F22).
		switch it := item.(type) {
		case MessageItem:
			if !it.Role.valid() {
				return fmt.Errorf("%w: input %d has unknown role %q", ErrInvalidRequest, i, it.Role)
			}
		case FunctionCallItem, FunctionCallOutputItem, ReasoningItem, ToolLoadItem:
		default:
			return fmt.Errorf("%w: input %d is %T; an item is a MessageItem, FunctionCallItem, "+
				"FunctionCallOutputItem, ReasoningItem or ToolLoadItem value", ErrInvalidRequest, i, item)
		}
	}
	if err := req.validateToolChoice(); err != nil {
		return err
	}
	if err := req.validateDeferral(); err != nil {
		return err
	}
	if !req.Cache.valid() {
		return fmt.Errorf("%w: unknown cache retention %q", ErrInvalidRequest, req.Cache)
	}
	return req.Reasoning.check()
}

// check rejects a reasoning setting no provider can send: an unknown effort,
// or a negative budget. Nil is none. Request validation and WithReasoning
// both use it (0026-MADR F19).
func (r *Reasoning) check() error {
	if r == nil {
		return nil
	}
	if r.Effort != "" && !r.Effort.valid() {
		return fmt.Errorf("%w: unknown reasoning effort %q", ErrInvalidRequest, r.Effort)
	}
	if r.Budget < 0 {
		return fmt.Errorf("%w: reasoning budget %d is negative", ErrInvalidRequest, r.Budget)
	}
	return nil
}

// validateDeferral rejects what no service can send with deferred tools
// (0040-MADR D1, A5): two search tools, a deferred search tool, every tool
// deferred (Anthropic refuses it), a ToolLoadItem naming anything but a
// deferred tool of the request (Anthropic's "Tool reference … not found"), a
// ToolLoadItem with no earlier result for its call, whose load no wire could
// attach, and a forced choice of a deferred tool nothing has loaded, which
// the model cannot see.
func (req *Request) validateDeferral() error {
	deferred := map[string]bool{}
	search, deferredCount := "", 0
	for _, tool := range req.Tools {
		if tool.Search {
			if tool.Deferred {
				return fmt.Errorf("%w: search tool %q is deferred", ErrInvalidRequest, tool.Name)
			}
			if search != "" {
				return fmt.Errorf("%w: two search tools, %q and %q", ErrInvalidRequest, search, tool.Name)
			}
			search = tool.Name
		}
		if tool.Deferred {
			deferred[tool.Name] = true
			deferredCount++
		}
	}
	if deferredCount > 0 && deferredCount == len(req.Tools) {
		return fmt.Errorf("%w: every tool is deferred; at least one must not be", ErrInvalidRequest)
	}
	loaded, answered := map[string]bool{}, map[string]bool{}
	for i, item := range req.Input {
		switch it := item.(type) {
		case FunctionCallOutputItem:
			answered[it.CallID] = true
		case ToolLoadItem:
			if !deferred[it.Name] {
				return fmt.Errorf("%w: input %d loads %q, which is not a deferred tool of the request",
					ErrInvalidRequest, i, it.Name)
			}
			if !answered[it.CallID] {
				return fmt.Errorf("%w: input %d loads %q for call %q, which has no earlier result",
					ErrInvalidRequest, i, it.Name, it.CallID)
			}
			loaded[it.Name] = true
		}
	}
	if name, forced := req.ToolChoice.Tool(); forced && deferred[name] && !loaded[name] {
		return fmt.Errorf("%w: tool choice forces %q, which is deferred and not loaded", ErrInvalidRequest, name)
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
