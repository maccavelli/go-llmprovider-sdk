// Package gemini is the Google Gemini provider (0015-MADR D2): the
// Interactions API with an API key (MADR 0014).
//
// Build it with New, or providers.New(llmprovider.ProviderGemini, …), with
// WithAPIKey, or WithTokenSource and a source that yields an API key, such as
// a CommandToken. An OAuth session is refused: this module does not sign in to
// Gemini subscriptions (0016-MADR D10). The key goes in x-goog-api-key, or the
// token's own Header (R16), never in the URL.
//
// Capabilities: tools and forced tool choice are Supported. Reasoning is
// BestEffort: a level a model does not take is refused, and stepped from
// (below).
// Continuation is Supported only with WithStore(true): Gemini chains only from
// a stored interaction, and nothing is stored by default (MADR 0014 §2).
// Native streaming is Supported: Stream asks the Interactions API for an event
// stream and yields its events as they arrive; a thought summary is not
// streamed, and arrives only in its item (0031-MADR E1).
//
// Images are Supported: an ImageItem goes as an Interactions image block, its
// bytes base64-encoded as data with their mime_type, or its URL as uri,
// joined to the user_input step before it (E14). A model whose facts state
// its input without image is refused before the network, with ErrUnsupported
// (0061-MADR D5).
//
// Degradations:
//   - A tool result's IsError is sent as the function_result step's is_error
//     (0032-MADR D3).
//   - A tool's Strict is not sent: the Interactions API has no such field
//     (0032-MADR D2).
//   - Reasoning takes an effort, sent as thinking_level; the Interactions API
//     has no thinking budget, so Reasoning.Budget is not sent. The level,
//     xhigh and max included, is sent by the model's facts and what it
//     refused (0062-MADR D2, D3): one the facts do not list, or the model
//     refused, steps to the nearest it takes, lower first, with a
//     WarningEffortDegraded naming what was sent. Gemini 3 refuses xhigh and
//     max with the levels it takes, so a first request at them steps to
//     high; gemini-2.5-flash-lite refuses medium, and low with a budget's
//     message, so it is sent high. No rule of model names decides (R5 and
//     the mapping of xhigh to high removed).
//     A refusal of the effort sent, in a form 0062-PLAN Phase 1 recorded,
//     is remembered for the rest of the process, in memory only, and the
//     request is sent again, at most 7 times in all; the Response, or a
//     stream's done event, carries a WarningReasoningLearned whose Refused a
//     caller can keep, and pass back with WithReasoningRefusals on its next
//     start (0062-MADR D4 to D6).
//   - ToolChoiceNone is sent as tool_choice "none", from the API reference;
//     only "any" and allowed_tools were measured (MADR 0014).
//
// Instructions and system items go to system_instruction, in that order.
//
// ListModels returns the curated listing, probing each of the first
// catalog.MaxListed listed models with one short, billed generation unless
// llmprovider.WithModelProbes(false) says otherwise (0016-MADR A5; 0038-MADR
// D4).
//
// catalog.ListRaw returns the listing as Google sends it: every page of
// GET https://generativelanguage.googleapis.com/v1beta/models, by
// pageToken, each entry's name, models/ prefix included, as its id
// (0041-MADR D9).
//
// Request.Cache sends nothing: Gemini caches prompts implicitly, and its
// explicit cachedContents resource is out of scope (0039-MADR D4, D6).
//
// Deferred tools (Tool.Deferred) have no native form here (0040-MADR D6):
// each tool a ToolLoadItem loads is appended to the tools, in load order,
// and a ToolLoadItem itself sends nothing. The response carries
// WarningDeferralEmulated (D7), and the prompt cache breaks once at each
// load.
// With PreviousResponseID, every deferred tool is sent, since the earlier
// loads are held by the service.
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, a listing
// that ListModels or catalog.List read, and the models.dev document once
// something has fetched it. A request never fetches, and never waits.
// From them, thinking_level is sent only when the model lists it; an effort
// off the list steps to the nearest listed.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// With no facts, the level asked is sent, and a refusal learned.
package gemini

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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
)

const (
	defaultBaseURL  = "https://generativelanguage.googleapis.com/v1beta"
	googleKeyHeader = "x-goog-api-key"
	probePrompt     = "Respond with ONLY the word Hello"
)

// storeOption is WithStore's value.
type storeOption bool

// WithStore sets whether interactions are stored (MADR 0014 §2). Gemini stores
// nothing unless given true, and continuing from a PreviousResponseID needs
// it: without it, continuation is Unsupported.
func WithStore(store bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderGemini, "gemini.WithStore", storeOption(store))
}

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	store     bool
	userAgent string
	logger    *slog.Logger
	probe     bool
	// cache is catalog.WithCache's, nil for the process's (0048-MADR D1).
	cache *cacheref.Value
	// refusals are WithReasoningRefusals' (0062-MADR D6).
	refusals []llmprovider.ReasoningRefusal
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
	// limit is catalog.WithLimit's n, which cuts ListModels's static
	// fallback; zero is no limit (0038-MADR A3).
	limit int
}

// New builds the Gemini provider. It needs an API key: WithAPIKey, or
// WithTokenSource with a source that yields one. No key, an empty key, and an
// option llmprovider.ResolveOptions refuses give an error matching
// llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one matching
// llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderGemini, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: gemini needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: gemini api key is required", llmprovider.ErrInvalidRequest) // 0020-MADR F52
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: gemini takes an API key, not an OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported)
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   defaultBaseURL,
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		probe:     st.ModelProbes(),
		cache:     cacheref.From(st.Values()),
		refusals:  st.ReasoningRefusals(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.Supported,
			// gemini-2.5-flash-lite does not think at low or medium effort,
			// and the Interactions API takes no budget (0026-MADR F38).
			Reasoning:       llmprovider.BestEffort,
			Continuation:    llmprovider.Unsupported,
			NativeStreaming: llmprovider.Supported,
			Images:          llmprovider.Supported,
		},
	}
	for _, v := range st.Values() {
		if store, ok := v.(storeOption); ok {
			p.store = bool(store)
		}
	}
	if p.store {
		p.caps.Continuation = llmprovider.Supported
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	p.limit = listlimit.Of(st.Values())
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderGemini }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Interactions API.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the level sent is learned, and the request
	// built again (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(llmprovider.ProviderGemini), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) }, decodeInteraction,
		effort.Check)
}

// learner is req's level, learned from the model's refusals of it
// (0062-MADR D2 to D6).
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.cache.LearnedThinking(), string(llmprovider.ProviderGemini), p.baseURL,
		cmp.Or(req.Model, p.model), p.refusals, matchRefusal)
}

// Stream streams req from the Interactions API as it is generated (0031-MADR
// E1): the request asks for an event stream, and each event is yielded as it
// arrives.
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	if err := p.caps.Check(req); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	return func(yield func(llmprovider.Event, error) bool) {
		// Each range is its own request, with its own effort learner.
		effort := p.learner(req)
		for ev, err := range wire.StreamEventsRetry(ctx, string(llmprovider.ProviderGemini), p.src,
			func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, true, effort) }, interactionEvents,
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
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderGemini, cmp.Or(req.Model, p.model),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// call is the Interactions request for req, asking for an event stream when
// stream is set, at the level effort chooses.
func (p *provider) call(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	effort *wire.Learner) wire.Call {
	facts := p.facts(ctx, req, token)
	body, warnings := p.body(req, facts, effort)
	if stream {
		body["stream"] = true
	}
	model := cmp.Or(req.Model, p.model)
	return wire.Call{
		Provider: string(llmprovider.ProviderGemini),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + interactionsPath,
		Body:     body,
		Stream:   stream,
		Warnings: append(deferral.Warnings(deferral.ForRequest(deferral.Other, model, req), req, model), warnings...), // 0040-MADR D7; 0062-MADR D3, D6
		// A model whose facts take no image is refused before the network
		// (0061-MADR D5).
		Refusal: wire.RefuseImages(cmp.Or(req.Model, p.model), req.Input, facts.Has(catalog.FieldInput),
			slices.Contains(facts.Input, catalog.ModalityImage)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			// In a header, not the URL, so it cannot leak through a *url.Error.
			token.Apply(r, googleKeyHeader, "")
			if stream {
				r.Header.Set("Accept", "text/event-stream")
			}
		},
	}
}

// body is the Interactions request for req (MADR 0014 §1), at the level
// learner chooses.
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
	gen := map[string]any{"max_output_tokens": maxTokens}
	var warnings []llmprovider.Warning
	if r := p.effectiveReasoning(req); r != nil {
		// Without it a thought step carries only its signature.
		gen["thinking_summaries"] = interactionSummariesAuto
		// The level by D3's ladder over what the model refused and the levels
		// it lists, when it lists any (0041-MADR D2; 0062-MADR D2, D3); none
		// leaves the model's default. A level the model does not take is
		// refused, learned and stepped from (0062-MADR D8: R5 and the xhigh
		// mapping removed).
		var level llmprovider.Effort
		level, warnings = learner.Effort(r.Effort, facts.Reasoning.Efforts, len(facts.Reasoning.Efforts) > 0)
		if level != "" {
			gen["thinking_level"] = string(level)
		}
	}
	body := map[string]any{
		jsonKeyModel:        model,
		jsonKeyInput:        interactionsInput(req.Input),
		"store":             p.store,
		"generation_config": gen,
	}
	var system []string
	for _, part := range []string{req.Instructions, wire.SystemPrompt(req.Input)} {
		if part != "" {
			system = append(system, part)
		}
	}
	if len(system) > 0 {
		body["system_instruction"] = strings.Join(system, "\n\n")
	}
	// Deferred tools are appended as loaded (0040-MADR D6).
	if sent := deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)); len(sent) > 0 {
		tools := make([]map[string]any, len(sent))
		for i, tool := range sent {
			tools[i] = map[string]any{
				jsonKeyType:        jsonKeyFunction,
				jsonKeyName:        tool.Name,
				jsonKeyDescription: tool.Description,
				jsonKeyParameters:  wire.ToolSchema(tool.Schema),
			}
		}
		body[jsonKeyTools] = tools
		// tool_choice belongs in generation_config; top level is refused.
		if choice := toolChoice(req.ToolChoice); choice != nil {
			gen["tool_choice"] = choice
		}
	}
	if req.PreviousResponseID != "" {
		body["previous_interaction_id"] = req.PreviousResponseID
	}
	return body, warnings
}

// effectiveReasoning is the request's Reasoning, else WithReasoning's, with
// an effort the request leaves empty taken from WithReasoning; nil for none.
func (p *provider) effectiveReasoning(req *llmprovider.Request) *llmprovider.Reasoning {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return nil
	}
	out := *r
	if out.Effort == "" && p.reasoning != nil {
		out.Effort = p.reasoning.Effort
	}
	return &out
}

// toolChoice is generation_config.tool_choice for choice, or nil for auto.
func toolChoice(choice llmprovider.ToolChoice) any {
	if name, named := choice.Tool(); named {
		return map[string]any{"allowed_tools": map[string]any{"mode": "any", "tools": []string{name}}}
	}
	switch choice {
	case llmprovider.ToolChoiceRequired:
		return "any"
	case llmprovider.ToolChoiceNone:
		return "none"
	default:
		return nil
	}
}

// ListModels returns the curated list of production text models: the static
// catalog's models the Models API serves, then the rest it serves, in full
// unless catalog.WithLimit cuts it (0038-MADR D4), probed unless
// WithModelProbes(false). It falls back to the static catalog, and returns
// the curated list when every probe fails.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderGemini, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderGemini), p.limit)
	}
	// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := wire.ProbeHealth(ctx, p.cache.ProbeStore(), llmprovider.ProviderGemini, p.baseURL, p.src, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		// The old API's probe: the provider's output limit, nothing stored, no
		// reasoning (0015-MADR D1).
		probe := *p
		probe.model, probe.store, probe.reasoning = model, false, nil
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
