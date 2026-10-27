// Package claude is the Anthropic Claude provider (0015-MADR D2): the
// Messages API with an API key.
//
// Build it with New, or providers.New(llmprovider.ProviderClaude, …), with
// WithAPIKey, or WithTokenSource and a source that yields an API key, such as
// a CommandToken. An OAuth session is refused: this module does not sign in to
// Anthropic subscriptions (0016-MADR D10).
//
// Capabilities: tools and reasoning are Supported. Forced tool choice is
// BestEffort: with reasoning, and on the models that refuse it, it is sent as
// "auto" (see Degradations; 0020-MADR F43, 0034-MADR D2).
// Continuation is Unsupported: the Messages API is stateless, so a caller
// replays prior items instead. Native streaming is
// Supported: Stream asks the Messages API for an event stream and yields its
// events as they arrive (0031-MADR E1).
//
// Images are Supported: an ImageItem goes as an image block, its bytes
// base64-encoded with their media type, or its URL, joined to the user's
// message before it, whose text becomes a text block. A model whose facts
// state its input without image is refused before the network, with
// ErrUnsupported (0061-MADR D5).
//
// Degradations:
//   - A tool result's IsError is sent as tool_result's is_error
//     (0032-MADR D3).
//   - A tool's Strict, when set, is sent as the tool's strict; nil sends
//     nothing (0034-MADR D4).
//   - With Reasoning, a forced tool choice (ForceTool or ToolChoiceRequired)
//     is sent as "auto": Anthropic refuses to force a tool while thinking.
//     The Response, and a stream's done event, carry a
//     WarningForcedChoiceDegraded (0034-MADR D7).
//   - claude-fable-5-1, claude-mythos-5-1, claude-opus-5-5 and
//     claude-sonnet-5-5 refuse a forced tool choice with HTTP 400 (0034-MADR
//     D1). A model id is matched after its gateway or platform prefix and
//     snapshot date are stripped, and WithModelForcedChoice overrides the
//     list either way (D9). On those models:
//   - A named tool (ForceTool, and so GenerateToolCall) is sent as structured
//     outputs: the tool's schema as output_config.format, the tools offered
//     with tool_choice "none". The answer's JSON is returned as a
//     FunctionCallItem for the tool, finishing tool_calls, and a stream
//     yields the call, not text deltas. It is not degraded, and carries no
//     warning (D8). The format's schema is a copy in which every object that
//     does not say otherwise is closed with additionalProperties false, as
//     structured outputs requires; the tool definition is sent as given. A
//     schema structured outputs still cannot compile, such as one with an
//     object left open, returns the API's 400 (Q9).
//   - ToolChoiceRequired fails before the network, with an error matching
//     ErrUnsupported (D10). With WithForcedChoiceFallback(ForcedChoiceDegrade)
//     it is sent as "auto", with Anthropic's turn instruction: a
//     mid-conversation system message after the last user turn, or, when the
//     input does not end with one, a line of the system field. The Response,
//     and a stream's done event, then carry a WarningForcedChoiceDegraded
//     (D2, D7).
//   - Another model that refuses a forced choice the same way (HTTP 400,
//     invalid_request_error, a message naming tool_choice) is remembered for
//     the rest of the process, in memory only, and the request handled as
//     above: a named tool is sent once more, as structured outputs;
//     ToolChoiceRequired is sent once more, degraded, under
//     ForcedChoiceDegrade, and otherwise returned as an error matching
//     ErrUnsupported and wrapping the *APIError. The Response, or a stream's
//     done event, carries a WarningForcedChoiceLearned naming the model (D3).
//   - Reasoning's shape is the model's facts', else an enabled budget when
//     Reasoning.Budget is set, else adaptive thinking, with the effort as
//     output_config.effort (0062-MADR D2). An enabled budget is
//     Reasoning.Budget, else 1024 for EffortLow, else 4096, with the output
//     limit raised above it. No rule of model names decides: a refusal of the
//     shape sent, in a form 0062-PLAN Phase 1 recorded, is remembered as an
//     effort's is, below, and the request sent again with the other shape; a
//     model that refuses both has the second refusal returned (D3).
//   - The effort, output_config.effort, max included, is sent by the model's
//     facts and what it refused (0062-MADR D2, D3): one the facts do not
//     list, or the model refused, steps to the nearest it takes, lower
//     first, with a WarningEffortDegraded naming what was sent.
//     A refusal of the effort sent, in a form 0062-PLAN Phase 1 recorded,
//     is remembered for the rest of the process, in memory only, and the
//     request is sent again, at most 7 times in all; the Response, or a
//     stream's done event, carries a WarningReasoningLearned whose Refused a
//     caller can keep, and pass back with WithReasoningRefusals on its next
//     start (0062-MADR D4 to D6). A
//     budget model sends no effort.
//
// Instructions and system items go to the top-level system field, in that
// order.
//
// Request.Cache asks Anthropic to cache the prompt (0039-MADR D2): the
// request carries the top-level cache_control, Anthropic's automatic caching,
// and one breakpoint, on the system prompt, which is then sent as one text
// block, or with none on the last tool. Both have one TTL: "5m" for
// CacheShort, "1h" for CacheLong. Two of Anthropic's four breakpoints are
// used. A prompt below the model's minimum is not cached, and no error is
// returned. Without Cache nothing is sent, and Anthropic does not cache.
//
// ListModels returns the curated listing, probing each of the first
// catalog.MaxListed listed models with one short, billed generation unless
// llmprovider.WithModelProbes(false) says otherwise (0016-MADR A5; 0038-MADR
// D4).
//
// catalog.ListRaw returns the listing as Anthropic sends it: every page of
// GET https://api.anthropic.com/v1/models, by after_id (0041-MADR D9).
//
// Deferred tools (Tool.Deferred) go out natively from Claude 4.5 on
// (0040-MADR D2, A1): each with defer_loading and no cache marker, and a
// result's loads as tool_reference blocks in its tool_result, with the
// result's text as a text block after the turn's results. An earlier model
// gets each loaded tool appended to the tools instead, and the response
// carries WarningDeferralEmulated (D6, D7). The Search tool is the caller's
// own function; Anthropic's hosted tool search is not sent.
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, a listing
// that ListModels or catalog.List read, and the models.dev document once
// something has fetched it. A request never fetches, and never waits.
// From them comes the thinking shape: a budget bound means an enabled
// budget within it, efforts mean adaptive thinking with the effort, and a
// toggle alone means adaptive thinking with none.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// With no facts, the request decides the shape, and a refusal is learned
// (0062-MADR D2, D3).
package claude

import (
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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

const (
	defaultBaseURL   = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"

	// probeMaxTokens is the output limit a listing probe sends: the default
	// of the old API's probe provider, kept so the probe's request is
	// unchanged (0015-MADR D1).
	probeMaxTokens = 8192
	probePrompt    = "Respond with ONLY the word Hello"
)

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	forcing   messages.Policy        // WithForcedChoiceFallback's and WithModelForcedChoice's
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

// New builds the Claude provider. It needs an API key: WithAPIKey, or
// WithTokenSource with a source that yields one. No key, an empty key, and an
// option llmprovider.ResolveOptions refuses give an error matching
// llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one matching
// llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: claude needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: claude api key is required", llmprovider.ErrInvalidRequest)
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: claude takes an API key, not an OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported)
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   defaultBaseURL,
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		forcing:   messages.NewPolicy(st.ForcedChoiceFallback(), st.ModelForcedChoices(), cacheref.From(st.Values()).LearnedRefusals()),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		probe:     st.ModelProbes(),
		cache:     cacheref.From(st.Values()),
		refusals:  st.ReasoningRefusals(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.Supported,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Supported,
			Images:           llmprovider.Supported,
		},
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	p.limit = listlimit.Of(st.Values())
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderClaude }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Messages API.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	f := p.forcingFor(req)
	// A required choice a model refuses fails here unless the caller opted in
	// to degrading it (0034-MADR D10; R11).
	if err := messages.CheckForcing(req, f); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2). A refusal of a forced tool choice from a model no layer
	// knows is learned, and handled by policy: the request is built again and
	// sent once more, or the refusal returned (0034-MADR D3). A refusal of the
	// effort sent is learned, and the request built again (0062-MADR D5).
	retry, effort := messages.NewRefusalRetry(f, req.ToolChoice), p.learner(req)
	resp, err := wire.Send(ctx, string(llmprovider.ProviderClaude), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(req, p.facts(ctx, req, token), false, effort) },
		messages.Decode, wire.Retries(retry.Check, effort.Check))
	if err == nil {
		// A named tool sent as structured outputs is returned as its call
		// (0034-MADR D8); a refusal learned, and a forced choice sent as
		// auto, are reported (D3, D7).
		messages.StructuredCall(req, f, resp)
		for _, warn := range []func() (llmprovider.Warning, bool){retry.Warning, func() (llmprovider.Warning, bool) { return p.toolChoiceWarning(req) }} {
			if w, ok := warn(); ok {
				resp.Warnings = append(resp.Warnings, w)
			}
		}
	}
	return resp, err
}

// toolChoiceWarning is the warning req's tool choice earns, if any.
func (p *provider) toolChoiceWarning(req *llmprovider.Request) (llmprovider.Warning, bool) {
	thinking := req.Reasoning != nil || p.reasoning != nil
	return messages.ToolChoiceWarning(req, thinking, p.forcingFor(req))
}

// forcingFor is req's forced-choice context (0034-MADR D9, D10).
func (p *provider) forcingFor(req *llmprovider.Request) messages.Forcing {
	return p.forcing.For(string(llmprovider.ProviderClaude), p.modelFor(req))
}

// Stream streams req from the Messages API as it is generated (0031-MADR
// E1): the request asks for an event stream, and each event is yielded as it
// arrives. A 401 before the first event renews a refreshable token and sends
// once more, and a refusal of a forced tool choice is handled as Generate
// handles it (0034-MADR D3).
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	if err := p.caps.Check(req); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	f := p.forcingFor(req)
	if err := messages.CheckForcing(req, f); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	return func(yield func(llmprovider.Event, error) bool) {
		// Each range is its own request, with its own retry checks.
		retry, effort := messages.NewRefusalRetry(f, req.ToolChoice), p.learner(req)
		stream := wire.StreamEventsRetry(ctx, string(llmprovider.ProviderClaude), p.src,
			func(token llmprovider.Token) wire.Call { return p.call(req, p.facts(ctx, req, token), true, effort) },
			messages.Events, wire.Retries(retry.Check, effort.Check))
		stream = messages.StructuredStream(req, f, stream)
		for ev, err := range messages.WarnOnDone(stream, retry.Warning,
			func() (llmprovider.Warning, bool) { return p.toolChoiceWarning(req) }) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// learner is req's effort, learned from the model's refusals of it
// (0062-MADR D2 to D6).
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.cache.LearnedThinking(), string(llmprovider.ProviderClaude), p.baseURL, p.modelFor(req),
		p.refusals, messages.MatchRefusal)
}

// modelFor is req's model, else the provider's.
func (p *provider) modelFor(req *llmprovider.Request) string {
	if req.Model != "" {
		return req.Model
	}
	return p.model
}

// call is the Messages request for req, asking for an event stream when
// stream is set, at the effort effort chooses.
func (p *provider) call(req *llmprovider.Request, facts catalog.ModelInfo, stream bool, effort *wire.Learner) wire.Call {
	body, warnings := p.body(req, facts, effort)
	if stream {
		body["stream"] = true
	}
	model := p.modelFor(req)
	return wire.Call{
		Provider: string(llmprovider.ProviderClaude),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/messages",
		Body:     body,
		Stream:   stream,
		// A model before Claude 4.5 has its deferred tools appended (0040-MADR
		// D7); an effort may be sent lower, and a refusal learned (0062-MADR
		// D3, D6).
		Warnings: append(deferral.Warnings(deferral.Of(deferral.Messages, model), req, model), warnings...),
		// A model whose facts take no image is refused before the network
		// (0061-MADR D5).
		Refusal: wire.RefuseImages(model, req.Input, facts.Has(catalog.FieldInput),
			slices.Contains(facts.Input, catalog.ModalityImage)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			token.Apply(r, "x-api-key", "")
			r.Header.Set("anthropic-version", anthropicVersion)
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
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderClaude, p.modelFor(req),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// thinkingFacts is the Messages thinking info's reasoning facts give
// (0041-MADR D2, R3), with its output limit.
func thinkingFacts(info catalog.ModelInfo) messages.Thinking {
	efforts := make([]string, len(info.Reasoning.Efforts))
	for i, e := range info.Reasoning.Efforts {
		efforts[i] = string(e)
	}
	return messages.FactsThinking(info.Reasoning.Toggle, efforts, info.Has(catalog.FieldReasoningBudget),
		info.Reasoning.BudgetMin, info.Reasoning.BudgetMax, outputLimit(info))
}

// outputLimit is info's output limit, or zero when no source states it.
func outputLimit(info catalog.ModelInfo) int {
	if !info.Has(catalog.FieldMaxOutputTokens) {
		return 0
	}
	return info.MaxOutputTokens
}

// body is the Messages request for req, with its model's facts, at the
// effort effort chooses.
func (p *provider) body(req *llmprovider.Request, facts catalog.ModelInfo, effort *wire.Learner) (map[string]any,
	[]llmprovider.Warning) {
	model := p.modelFor(req)
	maxTokens := p.maxTokens
	if req.MaxOutputTokens > 0 {
		maxTokens = req.MaxOutputTokens
	} else if limit := outputLimit(facts); limit > 0 {
		// The default, cut to the model's limit, never raised (0041-MADR D2).
		maxTokens = min(maxTokens, limit)
	}
	// Deferred tools go out in the model's form (0040-MADR D1, D2, D6).
	form := deferral.Of(deferral.Messages, model)
	msgs := messages.FromItems(req.Input)
	if form == deferral.MessagesNative {
		msgs = messages.FromItemsLoading(req.Input)
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   msgs,
	}
	var system []string
	for _, part := range []string{req.Instructions, wire.SystemPrompt(req.Input)} {
		if part != "" {
			system = append(system, part)
		}
	}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n\n")
	}
	thinking, warnings := p.addReasoning(body, req, maxTokens, facts, effort)
	if tools := deferral.Tools(req, form); len(tools) > 0 {
		if form == deferral.MessagesNative {
			body["tools"] = wire.MessagesToolsNative(tools)
		} else {
			body["tools"] = wire.MessagesTools(tools)
		}
		messages.AddToolChoice(body, req.ToolChoice, thinking, p.forcing.For(string(llmprovider.ProviderClaude), model))
	}
	// Last, once the system and tools are set (0039-MADR D2).
	messages.AddCache(body, req.Cache)
	return body, warnings
}

// addReasoning adds the thinking fields for the request's Reasoning, or
// WithReasoning's, in the shape and at the effort learner chooses, from what
// the model refused, its facts and the request, and reports whether it did,
// with the effort's warnings (0062-MADR D2, D3, D6). Each of effort and budget
// falls back to WithReasoning's.
func (p *provider) addReasoning(body map[string]any, req *llmprovider.Request, maxTokens int, info catalog.ModelInfo,
	learner *wire.Learner) (bool, []llmprovider.Warning) {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return false, nil
	}
	effort, budget := r.Effort, r.Budget
	if p.reasoning != nil {
		if effort == "" {
			effort = p.reasoning.Effort
		}
		if budget == 0 {
			budget = p.reasoning.Budget
		}
	}
	effort, warnings := learner.Effort(effort, info.Reasoning.Efforts, info.Has(catalog.FieldReasoningEfforts))
	facts := thinkingFacts(info)
	shape := learner.Shape(messages.PreferredShape(facts, budget))
	body["max_tokens"] = messages.AddThinkingFor(body, string(effort), budget, maxTokens, facts, shape)
	return true, warnings
}

// ListModels returns curated Claude text models (Models API and catalog),
// probed unless WithModelProbes(false). It falls back to the static catalog.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderClaude, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderClaude), p.limit)
	}
	// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := wire.ProbeHealth(ctx, p.cache.ProbeStore(), llmprovider.ProviderClaude, p.baseURL, p.src, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		probe := *p
		probe.model, probe.maxTokens, probe.reasoning = model, probeMaxTokens, nil
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
