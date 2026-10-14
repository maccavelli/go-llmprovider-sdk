// Package grok is the xAI Grok provider (0015-MADR D2): the Responses API at
// api.x.ai, with an API key, an xAI OAuth session, or the Grok CLI's login.
//
// Build it with New, or providers.New(llmprovider.ProviderGrok, …), with
// WithAPIKey, or WithTokenSource and any source: a static or command-backed
// key, an OAuthSession from the Grok login, or a VendorCLISession on the
// Grok CLI's login. The token goes in Authorization as a bearer token, or in
// the token's own Header (R16). A 401 from a source that can refresh is
// retried once with a new token.
//
// Capabilities: tools, forced tool choice and continuation are Supported.
// Reasoning is BestEffort: a model off the Grok CLI's menu gets no reasoning
// field (see Degradations, 0020-MADR F43). Native streaming is Supported:
// Stream asks the Responses API for an event stream and yields its events as
// they arrive (0031-MADR D1).
//
// Degradations:
//   - A tool result's IsError is not sent: the Responses API has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - A tool's Strict is sent as strict (0032-MADR D2).
//   - Reasoning takes an effort, clamped to the Grok CLI's menu for the
//     model (MADR 0012 §6): grok-4.6 offers low to xhigh, grok-4.5 low to
//     high, grok-4.6-build and grok-3-mini low and high; grok-4.7 and
//     grok-4.3 low to xhigh, as xAI's listing reports (0033-MADR D4). An
//     effort off the menu is sent as the nearest lower one, and no effort
//     as high. Other models reason on their own, and get no reasoning
//     field. A Budget is not sent.
//   - Instructions are sent as a leading system message, the shape the old
//     API already sent for a system item; the Responses instructions field
//     is not sent. xAI obeys both forms, measured on grok-4.6 and grok-4.7
//     (0021-MADR, amendment "L2 is the test's prompt, not the wire").
//   - ToolChoiceRequired and ToolChoiceNone are sent as "required" and
//     "none", the Responses API's values; only a named tool was measured
//     against xAI.
//
// ListModels returns the curated listing, probing each of the first
// catalog.MaxListed listed models with one short, billed generation unless
// llmprovider.WithModelProbes(false) says otherwise (0016-MADR A5; 0038-MADR
// D4).
//
// xAI caches prompts on its own. Request.Cache sends the provider's session
// id as prompt_cache_key, which xAI treats as its x-grok-conv-id routing
// key; CacheLong sends the same as CacheShort (0039-MADR D3).
//
// Deferred tools (Tool.Deferred) go through grok-build's use_tool wrapper
// (0040-MADR D5): the tools sent are those not deferred, then use_tool, the
// same list before and after each load, so the cache holds. A result that
// ToolLoadItems follow is sent with its text, then the loaded tools'
// definitions as JSON. A use_tool call naming a deferred tool comes back as
// that tool's call, and is replayed as the wrapper call; one naming anything
// else, or with input that is not an object, stays a use_tool call. A caller
// tool named use_tool, beside a deferred tool, is refused with
// ErrInvalidRequest. A forced choice of a deferred tool, which tool_choice
// can name only in the list, appends it instead (D6). Either way the response
// carries WarningDeferralEmulated (D7).
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, a listing
// that ListModels or catalog.List read, and the models.dev document once
// something has fetched it. A request never fetches, and never waits.
// From them come the efforts a request may send, xAI's listing above
// models.dev; an effort off them clamps as before.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// With no facts, the efforts menu decides, as before.
package grok

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

const (
	headerAuthorization = "Authorization"

	// probeMaxOutputTokens is the output limit a listing probe sends: the
	// default of the old API's probe provider, kept so the probe's request
	// is unchanged (0015-MADR D1).
	probeMaxOutputTokens = 8192
	probePrompt          = "Respond with ONLY the word Hello"
)

// BaseURL is the xAI API endpoint, for API keys and OAuth sessions alike.
const BaseURL = "https://api.x.ai/v1"

// storeOption is WithStore's value.
type storeOption bool

// WithStore sets whether responses are stored (MADR 0012 §6). Without it the
// service default applies, which keeps continuation working; callers under
// zero data retention pass false, after which a PreviousResponseID has
// nothing to chain from.
func WithStore(store bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderGrok, "grok.WithStore", storeOption(store))
}

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	store     *bool                  // WithStore's value, or nil for the service default
	userAgent string
	logger    *slog.Logger
	probe     bool
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
	// session is the conversation id, sent as prompt_cache_key when a
	// request asks for caching (0039-MADR D3).
	session string
	// limit is catalog.WithLimit's n, which cuts ListModels's static
	// fallback; zero is no limit (0038-MADR A3).
	limit int
}

// New builds the Grok provider. It needs a credential: WithAPIKey, or
// WithTokenSource.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderGrok, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: grok needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: grok api key is required", llmprovider.ErrInvalidRequest)
		}
	case *auth.OAuthSession:
		// R16: another provider's session is refused, so its token never
		// reaches xAI (0026-MADR F2).
		if owner := s.Owner(); owner != llmprovider.ProviderGrok {
			return nil, fmt.Errorf("%w: grok takes an API key, a Grok sign-in or the Grok CLI's login, not a %q session (0026-MADR F2)",
				llmprovider.ErrUnsupported, owner)
		}
	case *auth.VendorCLISession:
		if s.Provider != llmprovider.ProviderGrok {
			return nil, fmt.Errorf("%w: grok takes an API key, a Grok sign-in or the Grok CLI's login, not the %q CLI's login (0026-MADR F2)",
				llmprovider.ErrUnsupported, s.Provider)
		}
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   BaseURL,
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		probe:     st.ModelProbes(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.Supported,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Supported,
			NativeStreaming:  llmprovider.Supported,
		},
	}
	for _, v := range st.Values() {
		if store, ok := v.(storeOption); ok {
			s := bool(store)
			p.store = &s
		}
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	// A session outlives this provider, and may serve another: a caller's
	// client replaces a provider's default there, never another caller's,
	// and the discarding default logger is not shared (0026-MADR F37).
	shareHTTPClient(src, p.client, st.HTTPClientGiven())
	if st.LoggerGiven() {
		shareLogger(src, st.Logger())
	}
	p.limit = listlimit.Of(st.Values())
	p.session = st.SessionID()
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderGrok }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Responses API. A 401 from a source that can
// refresh is retried once with a new token.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	form, decoder := p.deferForm(req)
	if err := deferral.CheckDispatch(form, req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2).
	return wire.Reauth(ctx, string(llmprovider.ProviderGrok), p.src, func(token llmprovider.Token) (*llmprovider.Response, error) {
		return wire.Post(ctx, p.call(req, p.facts(ctx, req, token), false), token,
			decoder.DecodeFor(string(llmprovider.ProviderGrok)))
	})
}

// Stream streams req from the Responses API as it is generated (0031-MADR
// D1): the request asks for an event stream, and each event is yielded as it
// arrives. A 401 before the first event renews a refreshable token and sends
// once more.
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	form, decoder := p.deferForm(req)
	err := p.caps.Check(req)
	if err == nil {
		err = deferral.CheckDispatch(form, req)
	}
	if err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	return wire.StreamEventsFor(ctx, p.src, func(token llmprovider.Token) wire.Call {
		return p.call(req, p.facts(ctx, req, token), true)
	}, decoder.Events)
}

// facts is what the caches know of req's model, read for each request and
// never waited for (0041-MADR D1). The request's own token finds the cached
// listing, so the token source is not asked again.
func (p *provider) facts(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) catalog.ModelInfo {
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderGrok, cmp.Or(req.Model, p.model),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// deferForm is req's deferred-tool form, and the decoder its reply needs:
// Grok has no native form, so its deferred tools go through grok-build's
// use_tool wrapper, and the decoder unwraps a call to it (0040-MADR D5).
func (p *provider) deferForm(req *llmprovider.Request) (deferral.Form, responses.Decoder) {
	form := deferral.ForRequest(deferral.Grok, cmp.Or(req.Model, p.model), req)
	var decoder responses.Decoder
	if form == deferral.Dispatch {
		decoder.Dispatch = deferral.Dispatched(req.Tools)
	}
	return form, decoder
}

// body is the Responses request for req, with its model's facts.
func (p *provider) body(req *llmprovider.Request, facts catalog.ModelInfo) map[string]any {
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
	// Deferred tools go through the use_tool wrapper, or are appended when a
	// forced choice names one (0040-MADR D5, D6).
	form, _ := p.deferForm(req)
	sent := responses.Input(input)
	if form == deferral.Dispatch {
		sent = responses.InputDispatch(input, req)
	}
	body := map[string]any{
		"model":             model,
		"input":             sent,
		"max_output_tokens": maxTokens,
	}
	if p.store != nil {
		body["store"] = *p.store
		if !*p.store {
			// Stateless, so reasoning comes back encrypted to be replayed,
			// as Codex asks on every request (0021-MADR D4).
			body["include"] = []string{"reasoning.encrypted_content"}
		}
	}
	wire.AddResponsesTools(body, deferral.Tools(req, form), req.ToolChoice)
	if effort, ok := p.effort(req, model, facts); ok {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	if req.PreviousResponseID != "" {
		body["previous_response_id"] = req.PreviousResponseID
	}
	wire.AddPromptCacheKey(body, req.Cache, p.session) // 0039-MADR D3
	return body
}

// effort is the reasoning effort to send, and false for none: the request's
// Reasoning, else WithReasoning's, an empty effort taken from WithReasoning,
// clamped to the model's menu: its facts' efforts when known, else
// effortMenu (0041-MADR D2, R6).
func (p *provider) effort(req *llmprovider.Request, model string, facts catalog.ModelInfo) (string, bool) {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return "", false
	}
	effort := r.Effort
	if effort == "" && p.reasoning != nil {
		effort = p.reasoning.Effort
	}
	clamped := clampReasoningEffortFor(model, string(effort), factsEfforts(facts), facts.Has(catalog.FieldReasoningEfforts))
	return clamped, clamped != ""
}

// call is the Responses request for req, asking for an event stream when
// stream is set.
func (p *provider) call(req *llmprovider.Request, facts catalog.ModelInfo, stream bool) wire.Call {
	body := p.body(req, facts)
	if stream {
		body["stream"] = true
	}
	form, _ := p.deferForm(req)
	return wire.Call{
		Provider: string(llmprovider.ProviderGrok),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/responses",
		Body:     body,
		Stream:   stream,
		// Dispatched, or appended for a forced choice (0040-MADR D7).
		Warnings: deferral.Warnings(form, req, cmp.Or(req.Model, p.model)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			token.Apply(r, headerAuthorization, "Bearer")
			if stream {
				r.Header.Set("Accept", "text/event-stream")
			}
		},
	}
}

// ListModels returns curated Grok text models available to this credential,
// probed unless WithModelProbes(false). It falls back to the static catalog.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderGrok, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderGrok), p.limit)
	}
	// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := wire.ProbeHealth(ctx, llmprovider.ProviderGrok, p.baseURL, p.src, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		// The old API's probe provider: the default output limit, no store,
		// no reasoning (0015-MADR D1).
		probe := *p
		probe.model, probe.maxTokens, probe.store, probe.reasoning = model, probeMaxOutputTokens, nil, nil
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

// clientUser is a session that refreshes with a client of its own, such as
// *auth.OAuthSession.
type clientUser interface {
	UseHTTPClient(*http.Client)
	UseDefaultHTTPClient(*http.Client)
}

// shareHTTPClient gives src the provider's client, when src takes one: a
// caller's (given) through UseHTTPClient, the provider's default through
// UseDefaultHTTPClient, so a later caller's client replaces it (0016-MADR
// D8; 0026-MADR F37).
func shareHTTPClient(src llmprovider.TokenSource, client *http.Client, given bool) {
	user, ok := src.(clientUser)
	switch {
	case !ok:
	case given:
		user.UseHTTPClient(client)
	default:
		user.UseDefaultHTTPClient(client)
	}
}

// loggerUser is a session that reports through a logger, such as
// *auth.OAuthSession.
type loggerUser interface {
	UseLogger(*slog.Logger)
}

// shareLogger gives src the provider's WithLogger logger, when it takes one
// (0020-MADR F48).
func shareLogger(src llmprovider.TokenSource, logger *slog.Logger) {
	if user, ok := src.(loggerUser); ok {
		user.UseLogger(logger)
	}
}
