// Package kilo is the Kilo Gateway provider (0015-MADR D2), the inference API
// behind the Kilo Code agent.
//
// Build it with New, or providers.New(llmprovider.ProviderKilo, …). The
// credential is a Kilo API key, WithAPIKey, or the session auth's Kilo device
// login returns, WithTokenSource (0017-MADR D2); without one, Kilo's anonymous
// token is sent, which free models accept (MADR 0012 §1.7). Another
// provider's OAuth session, or a vendor CLI login, is refused. The token goes
// in Authorization as a bearer token, or in its own Header (R16).
// A URL-prefixed token ("https://host/prefix:secret") selects that backend,
// and may name the organization (MADR 0012 §3.3).
//
// Only POST {base}/chat/completions is used. Kilo also answers /responses and
// /messages with the same model — it translates formats, unlike OpenCode —
// but those routes are undocumented, and Kilo's /responses puts reasoning
// where the Responses decoder does not read it. One route reaches the whole
// catalog.
//
// Capabilities: tools are Supported. Forced tool choice and reasoning are
// BestEffort: they are sent only when the model accepts them, as WithCapabilities
// declares (all are sent when it is not given). Continuation is Unsupported:
// Chat Completions is stateless. Native streaming is Supported:
// Stream asks for an event stream, with its usage, and yields its events as
// they arrive (0031-MADR E1, E5).
//
// Degradations:
//   - A tool result's IsError is not sent: Chat Completions has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - A tool's Strict is sent as function.strict (0032-MADR D2).
//   - Reasoning takes an effort: Kilo's reasoning object {effort}, or
//     {enabled: true} with none, when the model accepts "reasoning"; else
//     reasoning_effort (medium with none) when it accepts that. A Budget is
//     not sent.
//   - The effort, max included, in either field, is sent by the model's
//     facts and what it refused (0062-MADR D2, D3): one the facts do not
//     list, or the model refused, steps to the nearest it takes, lower
//     first, with a WarningEffortDegraded naming what was sent; with none
//     left, the reasoning object is sent as {enabled: true}.
//     A refusal of the effort sent, in a form 0062-PLAN Phase 1 recorded,
//     is remembered for the rest of the process, in memory only, and the
//     request is sent again, at most 7 times in all; the Response, or a
//     stream's done event, carries a WarningReasoningLearned whose Refused a
//     caller can keep, and pass back with WithReasoningRefusals on its next
//     start (0062-MADR D4 to D6).
//   - A tool choice is sent only when the model accepts tool_choice;
//     otherwise the tools are offered unforced, and ToolChoiceNone sends no
//     tools (0020-MADR F40).
//   - Instructions are sent as a leading system message.
//
// Requests carry provider.data_collection "deny" unless WithDataCollection(true)
// (MADR 0012 §3.3). ListModels returns the curated gateway listing; it never
// probes, as the gateway meters every call (MADR 0012 §1.6).
//
// catalog.ListRaw returns the listing as Kilo sends it:
// GET https://api.kilo.ai/api/gateway/models, or an organization's
// /api/organizations/{id}/models (0041-MADR D9).
//
// Request.Cache, for a model whose id starts "anthropic/", sends OpenRouter's
// top-level cache_control with Anthropic's TTL: "5m" for CacheShort, "1h"
// for CacheLong (0039-MADR D4). Another model gets nothing for it. Kilo
// caches Claude on its own too, keyed by X-KiloCode-TaskId, which is sent
// either way.
//
// Deferred tools (Tool.Deferred) have no native form here (0040-MADR D6):
// each tool a ToolLoadItem loads is appended to the tools, in load order,
// and a ToolLoadItem itself sends nothing. The response carries
// WarningDeferralEmulated (D7), and the prompt cache breaks once at each
// load.
// OpenRouter's deferred_tools is not sent: Kilo's gateway does not honour it
// (A2).
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, a listing
// that ListModels or catalog.List read, and the models.dev document once
// something has fetched it. A request never fetches, and never waits.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// Its reasoning_details replay is unchanged.
package kilo

import (
	"cmp"
	"context"
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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

// The wire shapes here were measured against the live gateway on the date
// llmprovider's wireShapesProbedOnKilo holds, which its live probes check.

const (
	// anonymousToken is the bearer Kilo's own client sends without a login;
	// free models answer it and paid ones fail with a typed error (MADR 0012
	// §1.7).
	anonymousToken = "anonymous"

	headerAuthorization = "Authorization"
	// Kilo keys the editor, the task and the organization on these headers
	// (llmprovider's identification.go has the same names, for its tests).
	headerEditor       = "X-KILOCODE-EDITORNAME"
	headerTask         = "X-KiloCode-TaskId"
	headerOrganization = "X-KILOCODE-ORGANIZATIONID"

	paramTools           = "tools"
	paramToolChoice      = "tool_choice"
	paramReasoning       = "reasoning"
	paramReasoningEffort = "reasoning_effort"
)

// Kilo's scoped options' values.
type (
	organizationOption   string
	capabilitiesOption   []string
	dataCollectionOption bool
)

// WithOrganization scopes generation and listing to a Kilo organization:
// requests carry X-KILOCODE-ORGANIZATIONID and the listing is the
// organization's, as Kilo's client does (MADR 0012 §3.3). A URL-prefixed token
// whose path is .../api/organizations/{id} names the organization without it.
func WithOrganization(id string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithOrganization", organizationOption(id))
}

// WithCapabilities declares the request parameters the model accepts, as
// published in its supported_parameters (catalog.KiloModelCapabilities).
// Without it every parameter is sent: the gateway is the authority, and not
// sending one only because it cannot be confirmed would degrade requests.
func WithCapabilities(params ...string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithCapabilities",
		capabilitiesOption(append([]string(nil), params...)))
}

// WithDataCollection lets Kilo route to upstreams that may train on prompts
// when allow is true: it drops the data_collection "deny" preference every
// request otherwise sends (MADR 0012 §3.3). A model that requires collection is
// refused without it, with ErrNotPermitted and data_collection_required.
func WithDataCollection(allow bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithDataCollection", dataCollectionOption(allow))
}

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string // WithBaseURL's, or "" for Kilo Gateway
	org       string // WithOrganization's, or ""
	client    *http.Client
	capsKnown bool
	caps      map[string]bool
	collect   bool
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	userAgent string
	editor    string
	session   string
	logger    *slog.Logger
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
	// limit is catalog.WithLimit's n, which cuts ListModels's static
	// fallback; zero is no limit (0038-MADR A3).
	limit int
	// thinking is the memo of refused thinking settings, catalog.WithCache's
	// or the process's; refusals are WithReasoningRefusals' (0062-MADR D5,
	// D6).
	thinking *wire.Learned
	refusals []llmprovider.ReasoningRefusal
}

// New builds the Kilo provider. Without a key it sends Kilo's anonymous
// token. An option llmprovider.ResolveOptions refuses gives an error matching
// llmprovider.ErrInvalidRequest; an OAuth session for another provider, or a
// vendor CLI session, one matching llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		src = llmprovider.NewStaticToken(anonymousToken)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			src = &llmprovider.StaticToken{Value: anonymousToken, Header: s.Header}
		}
	case *auth.OAuthSession:
		// The Kilo device login's session is a Kilo credential, applied like
		// a key (0017-MADR D2). R16: another provider's session is refused.
		if s.Provider != llmprovider.ProviderKilo {
			return nil, fmt.Errorf("%w: kilo takes an API key or a Kilo device-login session, not a %s session (0016-MADR D10)",
				llmprovider.ErrUnsupported, s.Provider)
		}
	case *auth.VendorCLISession:
		return nil, fmt.Errorf("%w: kilo takes an API key or a Kilo device-login session, not a vendor CLI login (0016-MADR D10)",
			llmprovider.ErrUnsupported)
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   st.BaseURL(),
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		editor:    st.ClientName(),
		session:   st.SessionID(),
		logger:    st.Logger(),
		thinking:  cacheref.From(st.Values()).LearnedThinking(),
		refusals:  st.ReasoningRefusals(),
	}
	for _, v := range st.Values() {
		switch v := v.(type) {
		case organizationOption:
			p.org = string(v)
		case capabilitiesOption:
			p.capsKnown, p.caps = true, map[string]bool{}
			for _, c := range v {
				p.caps[c] = true
			}
		case dataCollectionOption:
			p.collect = bool(v)
		}
	}
	p.limit = listlimit.Of(st.Values())
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(p.session), llmprovider.WithHTTPClient(p.client))
	if p.org != "" {
		p.listing = append(p.listing, catalog.WithKiloOrganization(p.org))
	}
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderKilo }

// Capabilities reports what Kilo offers. Tools are always offered, as before;
// a tool choice and reasoning are sent only where the model accepts them.
func (p *provider) Capabilities() llmprovider.Capabilities {
	return llmprovider.Capabilities{
		Tools:            llmprovider.Supported,
		ForcedToolChoice: llmprovider.BestEffort,
		Reasoning:        llmprovider.BestEffort,
		Continuation:     llmprovider.Unsupported,
		NativeStreaming:  llmprovider.Supported,
	}
}

// supports reports whether the model accepts a request parameter. Unknown
// capabilities send everything.
func (p *provider) supports(param string) bool { return !p.capsKnown || p.caps[param] }

// Generate sends req to the gateway's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.Capabilities().Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the effort sent is learned, and the
	// request built again (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(llmprovider.ProviderKilo), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) },
		chatcompletions.DecodeFor(string(llmprovider.ProviderKilo)), effort.Check)
}

// learner is req's effort, learned from the model's refusals of it
// (0062-MADR D2 to D6).
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.thinking, string(llmprovider.ProviderKilo), p.baseURL, cmp.Or(req.Model, p.model), p.refusals,
		chatcompletions.MatchRefusal)
}

// Stream streams req from the Chat Completions API as it is generated
// (0031-MADR E1, E5): the request asks for an event stream with its usage,
// and each event is yielded as it arrives. A 401 before the first event
// renews a refreshable token and sends once more.
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	if err := p.Capabilities().Check(req); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	return func(yield func(llmprovider.Event, error) bool) {
		// Each range is its own request, with its own effort learner.
		effort := p.learner(req)
		for ev, err := range wire.StreamEventsRetry(ctx, string(llmprovider.ProviderKilo), p.src,
			func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, true, effort) }, chatcompletions.Events,
			effort.Check) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// call is the Chat Completions request for req, asking for an event stream
// with its usage when stream is set (0031-MADR E5).
func (p *provider) call(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	effort *wire.Learner) wire.Call {
	endpoints := kiloendpoint.Resolve(p.baseURL, token.Value, p.org)
	gateway, org := endpoints.Gateway, endpoints.Org
	body, warnings := p.body(req, p.facts(ctx, req, token), effort)
	if stream {
		body["stream"] = true
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	model := cmp.Or(req.Model, p.model)
	return wire.Call{
		Provider: string(llmprovider.ProviderKilo),
		Client:   p.client,
		Logger:   p.logger,
		URL:      gateway + "/chat/completions",
		Body:     body,
		Stream:   stream,
		Warnings: append(deferral.Warnings(deferral.ForRequest(deferral.Other, model, req), req, model), warnings...), // 0040-MADR D7; 0062-MADR D3, D6
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			r.Header.Set(headerEditor, p.editor)
			r.Header.Set(headerTask, p.session)
			token.Apply(r, headerAuthorization, "Bearer")
			if org != "" {
				r.Header.Set(headerOrganization, org)
			}
			if stream {
				r.Header.Set("Accept", "text/event-stream")
			}
		},
	}
}

// facts is what the caches know of req's model, read for each request and
// never waited for (0041-MADR D1). The request's own token finds the cached
// listing, so the token source is not asked again.
func (p *provider) facts(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) catalog.ModelInfo {
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderKilo, cmp.Or(req.Model, p.model),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// body is the Chat Completions request for req, with its model's facts.
func (p *provider) body(req *llmprovider.Request, facts catalog.ModelInfo, learner *wire.Learner) (map[string]any,
	[]llmprovider.Warning) {
	model := req.Model
	if model == "" {
		model = p.model
	}
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
	effort, reasoning, warnings := p.reasoningFields(req, facts, learner)
	body := chatcompletions.Body(model, maxTokens, input, chatcompletions.Opts{
		ReasoningEffort: effort,
		Reasoning:       reasoning,
		// Deferred tools are appended as loaded: the gateway does not
		// forward defer_loading (0040-MADR D6, A2).
		Tools:      deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)),
		SendStrict: true,
		ToolChoice: req.ToolChoice,
		// 301 of 366 models accept "tools" but only 279 accept
		// "tool_choice"; offering the tools unforced is strictly better than
		// a 400, and ToolChoiceNone sends none (0020-MADR F40).
		NoToolChoice: !p.supports(paramToolChoice),
		// The gateway's reasoning_details go back as they came (0021-MADR W9).
		ReplayReasoningDetails: true,
	})
	if !p.collect {
		// Kilo's opt-out from upstreams that train on prompts, which its client
		// sends with hide_prompt_training_models (MADR 0012 §3.3).
		body["provider"] = map[string]any{"data_collection": "deny"}
	}
	// A Claude model gets OpenRouter's top-level cache_control, with
	// Anthropic's TTL (0039-MADR D4).
	if req.Cache != "" && strings.HasPrefix(model, "anthropic/") {
		body["cache_control"] = messages.CacheMarker(req.Cache)
	}
	return body, warnings
}

// reasoningFields are one call's reasoning fields (MADR 0009 §6): with
// reasoning requested, Kilo's reasoning object when the model accepts
// "reasoning" — {effort} with an effort, else {enabled: true} — else
// reasoning_effort, medium with none, when it accepts that. The effort is
// learner's, by D3's ladder over the model's refusals and facts, with its
// warnings (0062-MADR D2, D3, D6).
func (p *provider) reasoningFields(req *llmprovider.Request, facts catalog.ModelInfo, learner *wire.Learner) (effort string,
	reasoning map[string]any, warnings []llmprovider.Warning) {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return "", nil, nil
	}
	e := r.Effort
	if e == "" && p.reasoning != nil {
		e = p.reasoning.Effort
	}
	switch {
	case p.supports(paramReasoning):
		if e != "" {
			// By D3's ladder; none left keeps reasoning on, with no effort
			// (0062-MADR D2, D3).
			if e, warnings = learner.Effort(e, facts.Reasoning.Efforts, facts.Has(catalog.FieldReasoningEfforts)); e != "" {
				return "", map[string]any{"effort": string(e)}, warnings
			}
			return "", map[string]any{"enabled": true}, warnings
		}
		return "", map[string]any{"enabled": true}, nil
	case p.supports(paramReasoningEffort):
		if e != "" {
			e, warnings = learner.Effort(e, facts.Reasoning.Efforts, facts.Has(catalog.FieldReasoningEfforts))
			return string(e), nil, warnings
		}
		return string(llmprovider.EffortMedium), nil, nil
	}
	return "", nil, nil
}

// ListModels returns the curated gateway listing, falling back to the static
// catalog. It spends no generation on probes (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	opts := p.listing
	if p.baseURL != "" {
		opts = append(append([]llmprovider.Option(nil), opts...), llmprovider.WithBaseURL(p.baseURL))
	}
	cat, err := catalog.List(ctx, llmprovider.ProviderKilo, p.src, opts...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderKilo), p.limit)
	}
	return listed, nil
}
