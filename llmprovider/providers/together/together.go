// Package together is the Together AI provider (0015-MADR D2; 0017-MADR D1),
// through its OpenAI-compatible Chat Completions API.
//
// Build it with New, or providers.New(llmprovider.ProviderTogether, …), with a
// Together API key: WithAPIKey, or WithTokenSource with a source that yields
// one. An OAuth session is refused: Together has none. The key goes in
// Authorization as a bearer token, or in its own Header (R16).
//
// Only POST {base}/chat/completions is used. The wire shapes are from
// Together's API reference, read 2026-09-30, and llmprovider's live tests
// confirmed them on 2026-10-03: text, a named tool, "none", both reasoning
// shapes and the listing (172 chat models) pass (0017-PLAN, V2).
//
// Capabilities: tools, forced tool choice and reasoning are BestEffort, as
// support is per model on Together (0017-MADR D1). Continuation is
// Unsupported: Chat Completions is stateless. Native streaming is Supported:
// Stream asks for an event stream, with its usage, and yields its events as
// they arrive (0031-MADR E1, E5).
//
// Images are Supported: an ImageItem goes as a Chat Completions image_url
// part, {"url": …}, its bytes as a data URL, joined to the user's message
// before it, whose text becomes a text part. A model whose facts state its
// input without image is refused before the network, with ErrUnsupported
// (0061-MADR D5).
//
// Degradations:
//   - A tool result's IsError is not sent: Chat Completions has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - A tool's Strict is sent as function.strict (0032-MADR D2).
//   - Reasoning sends reasoning {"enabled": true}, which Together's toggleable
//     models read, and reasoning_effort only when an effort is set: Together
//     accepts it for some models only (openai/gpt-oss-*, DeepSeek-V4-Pro). A
//     Budget is not sent. A request with no reasoning sends neither, so the
//     model's own default applies.
//   - The effort, max included, is sent by the model's facts (0062-MADR D2,
//     D3): one the facts do not list steps to the nearest they list, lower
//     first, with a WarningEffortDegraded naming what was sent. Together
//     accepts an effort a model does not take, a bogus one included
//     (0062-REPORT-thinking-refusals.md E5), so it learns no refusal.
//   - Instructions are sent as a leading system message.
//   - ToolChoiceRequired and ToolChoiceNone are sent as "required" and
//     "none", the Chat Completions values. Measured 2026-10-03: "required" is
//     honoured by DeepSeek-V4.1-Flash, GLM-5.3, GLM-5.3-Flash and Kimi-K3, but
//     openai/gpt-oss-120b answers it with HTTP 500 (ErrProviderUnavailable,
//     which WithRetry retries in vain); a named tool works on every model.
//
// finish_reason "eos" is a normal stop; only "length" is truncation.
//
// ListModels returns the curated chat models, ranked by the metadata
// llmprovider.WithModelMetadataURL names, and falls back to the static
// catalog. It never probes, as every call is metered (0016-MADR D9).
//
// catalog.ListRaw returns the listing as Together sends it:
// GET https://api.together.ai/v1/models, a bare array (0041-MADR D9).
//
// Together caches prompts on its own, and a write is free. Request.Cache sends
// the provider's session id as prompt_cache_key, which routes a conversation
// to one cache; CacheLong sends the same as CacheShort (0039-MADR D3).
//
// Deferred tools (Tool.Deferred) have no native form here (0040-MADR D6):
// each tool a ToolLoadItem loads is appended to the tools, in load order,
// and a ToolLoadItem itself sends nothing. The response carries
// WarningDeferralEmulated (D7), and the prompt cache breaks once at each
// load.
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, a listing
// that ListModels or catalog.List read, and the models.dev document once
// something has fetched it. A request never fetches, and never waits.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// Prior reasoning is replayed under the field the model's interleaved.field
// names, and under none when no source names one.
package together

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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
)

const (
	// defaultBaseURL is Together's API (llmprovider's togetherBaseURL).
	defaultBaseURL      = "https://api.together.ai/v1"
	headerAuthorization = "Authorization"
)

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	userAgent string
	logger    *slog.Logger
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
	// session is the conversation id, sent as prompt_cache_key when a
	// request asks for caching (0039-MADR D3).
	session string
	// limit is catalog.WithLimit's n, which cuts ListModels's static
	// fallback; zero is no limit (0038-MADR A3).
	limit int
	// thinking is the memo of refused thinking settings, catalog.WithCache's
	// or the process's; refusals are WithReasoningRefusals' (0062-MADR D5,
	// D6).
	thinking *wire.Learned
	refusals []llmprovider.ReasoningRefusal
}

// New builds the Together AI provider. It needs a key. No key, an empty key,
// and an option llmprovider.ResolveOptions refuses give an error matching
// llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one matching
// llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderTogether, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: together needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: together api key is required", llmprovider.ErrInvalidRequest)
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: together takes an API key, not an OAuth session (0017-MADR D1)", llmprovider.ErrUnsupported)
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
		thinking:  cacheref.From(st.Values()).LearnedThinking(),
		refusals:  st.ReasoningRefusals(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.BestEffort,
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Supported,
			Images:           llmprovider.Supported,
		},
	}
	if b := st.BaseURL(); b != "" {
		p.baseURL = strings.TrimRight(b, "/")
	}
	p.limit = listlimit.Of(st.Values())
	p.session = st.SessionID()
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderTogether }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to Together's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the effort sent is learned, and the
	// request built again (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(llmprovider.ProviderTogether), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) },
		chatcompletions.DecodeFor(string(llmprovider.ProviderTogether)), effort.Check)
}

// learner is req's effort, learned from the model's refusals of it
// (0062-MADR D2 to D6).
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.thinking, string(llmprovider.ProviderTogether), p.baseURL, cmp.Or(req.Model, p.model), p.refusals,
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
		for ev, err := range wire.StreamEventsRetry(ctx, string(llmprovider.ProviderTogether), p.src,
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
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderTogether, cmp.Or(req.Model, p.model),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// replayField is the field the model's prior reasoning is replayed under,
// its facts' ReplayField, when the input has an assistant turn to replay onto
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

// call is the Chat Completions request for req, asking for an event stream
// with its usage when stream is set (0031-MADR E5).
func (p *provider) call(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	effort *wire.Learner) wire.Call {
	facts := p.facts(ctx, req, token)
	body, warnings := p.body(req, facts, effort)
	if stream {
		body["stream"] = true
		body["stream_options"] = map[string]any{"include_usage": true}
	}
	model := cmp.Or(req.Model, p.model)
	return wire.Call{
		Provider: string(llmprovider.ProviderTogether),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/chat/completions",
		Body:     body,
		Stream:   stream,
		Warnings: append(deferral.Warnings(deferral.ForRequest(deferral.Other, model, req), req, model), warnings...), // 0040-MADR D7; 0062-MADR D3, D6
		// A model whose facts take no image is refused before the network
		// (0061-MADR D5).
		Refusal: wire.RefuseImages(cmp.Or(req.Model, p.model), req.Input, facts.Has(catalog.FieldInput),
			slices.Contains(facts.Input, catalog.ModalityImage)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			token.Apply(r, headerAuthorization, "Bearer")
			if stream {
				r.Header.Set("Accept", "text/event-stream")
			}
		},
	}
}

// body is the Chat Completions request for req.
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
	// Deferred tools are appended as loaded (0040-MADR D6).
	o := chatcompletions.Opts{Tools: deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)),
		ToolChoice: req.ToolChoice, SendStrict: true}
	var warnings []llmprovider.Warning
	if r := p.reasoningFor(req); r != nil {
		o.Reasoning = map[string]any{"enabled": true}
		// By D3's ladder over the model's refusals and facts (0062-MADR D2,
		// D3). Together accepts a value it does not take (0062-MADR F4, as
		// measured), so it learns nothing.
		var effort llmprovider.Effort
		effort, warnings = learner.Effort(llmprovider.Effort(p.effort(r)), facts.Reasoning.Efforts,
			facts.Has(catalog.FieldReasoningEfforts))
		o.ReasoningEffort = string(effort)
	}
	o.ReplayReasoningField = replayField(input, facts)
	body := chatcompletions.Body(model, maxTokens, input, o)
	wire.AddPromptCacheKey(body, req.Cache, p.session) // 0039-MADR D3
	return body, warnings
}

// reasoningFor is req's Reasoning, else WithReasoning's; nil for none.
func (p *provider) reasoningFor(req *llmprovider.Request) *llmprovider.Reasoning {
	if req.Reasoning != nil {
		return req.Reasoning
	}
	return p.reasoning
}

// effort is reasoning_effort for r: its effort, else WithReasoning's, else ""
// (not sent).
func (p *provider) effort(r *llmprovider.Reasoning) string {
	switch {
	case r.Effort != "":
		return string(r.Effort)
	case p.reasoning != nil:
		return string(p.reasoning.Effort)
	default:
		return ""
	}
}

// ListModels returns the curated chat models, falling back to the static
// catalog. It spends no generation on probes (0016-MADR D9).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderTogether, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderTogether), p.limit)
	}
	return listed, nil
}
