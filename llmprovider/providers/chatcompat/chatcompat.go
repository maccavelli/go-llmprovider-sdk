// Package chatcompat is a provider for any server that speaks OpenAI's Chat
// Completions API, under an id the caller picks (0061-MADR D4): LM Studio,
// llama.cpp's server, vLLM, LocalAI, a gateway, or any other.
//
// Build it with New, WithID and WithBaseURL. It is not in providers.Default();
// a caller registers it in its own Registry with Descriptor:
//
//	reg.Register(chatcompat.Descriptor("lmstudio", "LM Studio"),
//		func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
//			return chatcompat.New(append(opts, chatcompat.WithID("lmstudio"))...)
//		})
//
// The key is optional, unlike a built-in's: with WithAPIKey or WithTokenSource
// it goes in Authorization as a bearer token, or in its own Header (R16); with
// neither, or an empty key, no Authorization header is sent, for a local
// server that takes none. A remote server that needs one answers 401, which is
// ErrAuthFailure. An OAuth or vendor CLI session is refused: its token is a
// vendor's, never sent to another server.
//
// Options: WithID and catalog.WithCache are shared options
// (llmprovider.SharedOption), which every New takes. New reads WithID first,
// then resolves the options under that id, so For(id, …) reaches the
// provider; a WithID inside For is not read, so give it bare, as the factory
// above does. The other catalog options are scoped to the built-in providers,
// and New refuses them (0061-MADR, amendment "Phase 4's options").
//
// Generation uses POST {base}/chat/completions. ListModels is catalog.List's
// listing of GET {base}/models, as a built-in's: cached for ten minutes by
// base URL and key, one request for callers at once, and the last good
// listing, at most 24 hours old, after a transient failure; the server's own
// order, probing each of the first catalog.MaxListed models with one short
// generation only under llmprovider.WithModelProbes(true) (0061-MADR,
// amendment "chatcompat, provider-agnostic", E1).
//
// Capabilities: tools are Supported, sent with function.strict as the caller
// sets it (0032-MADR D2). Forced tool choice is BestEffort: the server decides.
// Reasoning is BestEffort: only some servers and models reason. Continuation
// is Unsupported: Chat Completions is stateless. Native streaming is
// Supported: Stream asks for an event stream, with its usage, and yields its
// events as they arrive (0031-MADR E1, E5).
//
// Degradations:
//   - A tool result's IsError is not sent: Chat Completions has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - Reasoning takes an effort, sent as reasoning_effort (medium with none);
//     a Budget is not sent.
//   - The effort, max included, is sent as asked, the server's model having
//     no facts here. A refusal of the effort sent, in a form 0062-PLAN Phase 1
//     recorded for the Chat Completions wire, is remembered for the rest of
//     the process, in memory only, and the request is sent again, at most 7
//     times in all, at the next effort; the Response, or a stream's done
//     event, carries a WarningReasoningLearned whose Refused a caller can
//     keep, and pass back with WithReasoningRefusals (0062-MADR D2 to D6).
//   - Instructions are sent as a leading system message.
//   - ToolChoiceRequired and ToolChoiceNone are sent as "required" and
//     "none", the Chat Completions values.
//
// Request.Cache sends nothing: a server's caching parameter, where it has one,
// is its own (0039-MADR D4).
//
// Deferred tools (Tool.Deferred) have no native form here (0040-MADR D6):
// each tool a ToolLoadItem loads is appended to the tools, in load order, and
// the response carries WarningDeferralEmulated (D7).
//
// Model facts (0041-MADR D1): each request reads what catalog.CachedFacts
// holds of its model, never fetching: a caller's catalog layers
// (catalog.WithModelLayer), then what the cached listing read of the server,
// then models.dev's facts of the model's canonical id, from the cached
// document. The listing reads a server's details where its /models reply is
// a recorded dialect's: Ollama's /api/show, for each model's window, tools,
// vision and thinking. The canonical id drops an organisation, an Ollama tag
// "latest", and quantisation and format tokens, and matches only an equal
// id (0061-MADR E2; its amendment "E2 and E4's menu, made exact before Phase
// 4c"). A model none of these knows has the default output limit or
// MaxOutputTokens, an effort checked against no list, and prior reasoning
// replayed under no field.
package chatcompat

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/compatid"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
)

const (
	// scope is the id New first resolves the options under, to read WithID.
	scope llmprovider.ProviderID = "chatcompat"

	headerAuthorization = "Authorization"

	// probeMaxOutputTokens and probePrompt are a listing probe's request, as
	// Ollama's.
	probeMaxOutputTokens = 8192
	probePrompt          = "Respond with ONLY the word Hello"
)

// builtins are the built-in providers' ids, which a chatcompat provider may
// not take (0061-MADR D4); a test pins them to providers.Default().
var builtins = []llmprovider.ProviderID{
	llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok,
	llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo, llmprovider.ProviderHuggingFace,
	llmprovider.ProviderKilo, llmprovider.ProviderTogether, llmprovider.ProviderOllama,
}

// WithID sets the provider's id, which New requires: the id a Registry holds
// it under, and the one its ID returns. It is a shared option
// (llmprovider.SharedOption), so another provider's New takes it and ignores
// it, and catalog's listing and lookups read it, to list the server.
func WithID(id llmprovider.ProviderID) llmprovider.Option {
	return llmprovider.SharedOption("chatcompat.WithID", compatid.Value(id))
}

type provider struct {
	id        llmprovider.ProviderID
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	probe     bool
	// cache is catalog.WithCache's, nil for the process's (0048-MADR D1).
	cache     *cacheref.Value
	userAgent string
	logger    *slog.Logger
	// listing carries the caller's options, WithID among them, the session,
	// the transport and the base URL to catalog's listing and lookups.
	listing []llmprovider.Option
	// thinking is the memo of refused thinking settings, catalog.WithCache's
	// or the process's; refusals are WithReasoningRefusals' (0062-MADR D5,
	// D6).
	thinking *wire.Learned
	refusals []llmprovider.ReasoningRefusal
}

// New builds a Chat Completions provider. It needs WithID and WithBaseURL. No
// WithID, an empty id, or a built-in provider's id, compared without case,
// gives an error matching llmprovider.ErrInvalidProvider; no WithBaseURL, and
// an option llmprovider.ResolveOptions refuses, one matching
// llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one matching
// llmprovider.ErrUnsupported. It makes no network call.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	// Probes are off unless the caller's WithModelProbes(true) follows: a
	// server's generations are its own to bill (0061-MADR D4).
	opts = append([]llmprovider.Option{llmprovider.WithModelProbes(false)}, opts...)
	first, err := llmprovider.ResolveOptions(scope, opts)
	if err != nil {
		return nil, err
	}
	id, err := idOf(first.Values())
	if err != nil {
		return nil, err
	}
	st, err := llmprovider.ResolveOptions(id, opts)
	if err != nil {
		return nil, err
	}
	if st.BaseURL() == "" {
		return nil, fmt.Errorf("%w: chatcompat %q needs WithBaseURL", llmprovider.ErrInvalidRequest, id)
	}
	src := st.TokenSource()
	switch src.(type) {
	case nil:
		src = llmprovider.NewStaticToken("")
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a vendor's session is never sent to another server.
		return nil, fmt.Errorf("%w: chatcompat %q takes a key, not an OAuth or vendor CLI session", llmprovider.ErrUnsupported, id)
	}
	ref := cacheref.From(st.Values())
	listing := append(slices.Clone(opts), llmprovider.WithSessionID(st.SessionID()),
		llmprovider.WithHTTPClient(st.HTTPClient()), llmprovider.WithBaseURL(st.BaseURL()))
	return &provider{
		id:        id,
		src:       src,
		model:     st.Model(),
		baseURL:   st.BaseURL(),
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		probe:     st.ModelProbes(),
		cache:     ref,
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		listing:   listing,
		thinking:  ref.LearnedThinking(),
		refusals:  st.ReasoningRefusals(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Supported,
		},
	}, nil
}

// idOf is the last WithID's id among values, refused as New says.
func idOf(values []any) (llmprovider.ProviderID, error) {
	id, found := compatid.Of(values)
	switch {
	case !found:
		return "", fmt.Errorf("%w: chatcompat needs WithID", llmprovider.ErrInvalidProvider)
	case strings.TrimSpace(id) == "":
		return "", fmt.Errorf("%w: chatcompat: WithID with an empty id", llmprovider.ErrInvalidProvider)
	case slices.ContainsFunc(builtins, func(b llmprovider.ProviderID) bool { return strings.EqualFold(string(b), id) }):
		return "", fmt.Errorf("%w: chatcompat: %q is a built-in provider's id", llmprovider.ErrInvalidProvider, id)
	}
	return llmprovider.ProviderID(id), nil
}

func (p *provider) ID() llmprovider.ProviderID { return p.id }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the server's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the effort sent is learned, and the
	// request built again (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(p.id), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) },
		chatcompletions.DecodeFor(string(p.id)), effort.Check)
}

// learner is req's effort, learned from the model's refusals of it
// (0062-MADR D2 to D6), read by the Chat Completions wire's matcher.
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.thinking, string(p.id), p.baseURL, cmp.Or(req.Model, p.model), p.refusals,
		chatcompletions.MatchRefusal)
}

// Stream streams req from the Chat Completions API as it is generated
// (0031-MADR E1, E5): the request asks for an event stream with its usage,
// and each event is yielded as it arrives. A 401 before the first event
// renews a refreshable token and sends once more.
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	if err := p.caps.Check(req); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	return func(yield func(llmprovider.Event, error) bool) {
		// Each range is its own request, with its own effort learner.
		effort := p.learner(req)
		for ev, err := range wire.StreamEventsRetry(ctx, string(p.id), p.src,
			func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, true, effort) }, chatcompletions.Events,
			effort.Check) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// facts is what the caches know of req's model, read for each request and
// never waited for (0041-MADR D1). The request's own token finds the cached
// listing, so the token source is not asked again.
func (p *provider) facts(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) catalog.ModelInfo {
	info, _ := catalog.CachedFacts(ctx, p.id, cmp.Or(req.Model, p.model),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// call is the Chat Completions request for req, asking for an event stream
// with its usage when stream is set (0031-MADR E5).
func (p *provider) call(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	effort *wire.Learner) wire.Call {
	body, warnings := p.body(req, p.facts(ctx, req, token), effort)
	if stream {
		body["stream"] = true
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	model := cmp.Or(req.Model, p.model)
	return wire.Call{
		Provider: string(p.id),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/chat/completions",
		Body:     body,
		Stream:   stream,
		Warnings: append(deferral.Warnings(deferral.ForRequest(deferral.Other, model, req), req, model), warnings...), // 0040-MADR D7; 0062-MADR D3, D6
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			p.authorize(r, token)
			if stream {
				r.Header.Set("Accept", "text/event-stream")
			}
		},
	}
}

// authorize sets token on r, unless it is empty: a server that takes no key
// gets no Authorization header (0061-MADR D4).
func (p *provider) authorize(r *http.Request, token llmprovider.Token) {
	if token.Value != "" {
		token.Apply(r, headerAuthorization, "Bearer")
	}
}

// body is the Chat Completions request for req, by its model's facts, and
// the effort's warnings.
func (p *provider) body(req *llmprovider.Request, facts catalog.ModelInfo, learner *wire.Learner) (map[string]any,
	[]llmprovider.Warning) {
	model := cmp.Or(req.Model, p.model)
	maxTokens := p.maxTokens
	if req.MaxOutputTokens > 0 {
		maxTokens = req.MaxOutputTokens
	} else if limit := facts.MaxOutputTokens; limit > 0 && facts.Has(catalog.FieldMaxOutputTokens) {
		// The default, cut to the model's limit, never raised (0041-MADR D2).
		maxTokens = min(maxTokens, limit)
	}
	input := req.Input
	if req.Instructions != "" {
		input = append([]llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: req.Instructions}}, input...)
	}
	// By D3's ladder over the model's refusals and facts (0062-MADR D2, D3).
	effort, warnings := learner.Effort(llmprovider.Effort(p.effort(req)), facts.Reasoning.Efforts,
		facts.Has(catalog.FieldReasoningEfforts))
	// Deferred tools are appended as loaded (0040-MADR D6).
	return chatcompletions.Body(model, maxTokens, input, chatcompletions.Opts{ReasoningEffort: string(effort),
		ReplayReasoningField: replayField(input, facts),
		Tools:                deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)), ToolChoice: req.ToolChoice,
		SendStrict: true}), warnings
}

// replayField is the field the model's prior reasoning is replayed under, its
// facts' ReplayField, when the input has an assistant turn to replay onto
// (0041-MADR D2, A3); "" otherwise.
func replayField(input []llmprovider.Item, facts catalog.ModelInfo) string {
	if !slices.ContainsFunc(input, isAssistantTurn) {
		return ""
	}
	return facts.Reasoning.ReplayField
}

// isAssistantTurn reports an item the model wrote: a call, or an assistant
// message.
func isAssistantTurn(item llmprovider.Item) bool {
	switch v := item.(type) {
	case llmprovider.FunctionCallItem:
		return true
	case llmprovider.MessageItem:
		return v.Role == llmprovider.RoleAssistant
	}
	return false
}

// effort is reasoning_effort for req: the request's Reasoning, else
// WithReasoning's, an empty effort taken from WithReasoning, else medium; ""
// for no reasoning.
func (p *provider) effort(req *llmprovider.Request) string {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return ""
	}
	switch {
	case r.Effort != "":
		return string(r.Effort)
	case p.reasoning != nil && p.reasoning.Effort != "":
		return string(p.reasoning.Effort)
	default:
		return string(llmprovider.EffortMedium)
	}
}

// ListModels returns the models catalog.List lists of the server, in its
// order, probed under WithModelProbes(true): the first catalog.MaxListed
// that answer a short generation, when any does. A listing within ten
// minutes is reused, and the last good one served after a transient failure.
// A failed listing returns its error: an *llmprovider.APIError of the reply's
// kind, one matching llmprovider.ErrOffline when the server cannot be reached
// for want of a network, llmprovider.ErrIncomplete for a reply it cannot
// read, or the context's error.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, p.id, p.src, p.listing...)
	if err != nil {
		return nil, err
	}
	if !cat.Live {
		// No static catalog stands in for a server's: its failure is the
		// caller's (none, for a listing with no model).
		return nil, listingError(p.id, cat.Err)
	}
	listed := cat.Recommended
	if !p.probe || len(listed) == 0 {
		return listed, nil
	}
	healthy := wire.ProbeHealth(ctx, p.cache.ProbeStore(), p.id, p.baseURL, p.src, listed, catalog.MaxListed,
		func(ctx context.Context, model string) (string, error) {
			probe := *p
			probe.model, probe.maxTokens, probe.reasoning = model, probeMaxOutputTokens, nil
			resp, err := probe.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: probePrompt}}})
			if err != nil {
				return "", err
			}
			return resp.OutputText(), nil
		})
	if len(healthy) > 0 {
		return healthy, nil
	}
	return listed, nil
}

// kinds are the errors a listing's failure keeps as it came: a kind, or the
// context's.
var kinds = []error{llmprovider.ErrRateLimited, llmprovider.ErrAuthFailure, llmprovider.ErrNotPermitted,
	llmprovider.ErrInvalidRequest, llmprovider.ErrProviderUnavailable, llmprovider.ErrUnsupported,
	llmprovider.ErrIncomplete, context.Canceled, context.DeadlineExceeded}

// listingError is a failed listing's error: as it came when it has a kind,
// else ErrIncomplete, the server having answered with what cannot be read.
func listingError(id llmprovider.ProviderID, err error) error {
	if err == nil || slices.ContainsFunc(kinds, func(k error) bool { return errors.Is(err, k) }) {
		return err
	}
	return fmt.Errorf("%w: chatcompat %s: the listing: %w", llmprovider.ErrIncomplete, id, err)
}
