// Package huggingface is the Hugging Face Inference Providers provider
// (0015-MADR D2): a routing proxy in front of partner inference providers.
//
// Build it with New, or providers.New(llmprovider.ProviderHuggingFace, …), with
// a Hugging Face token: WithAPIKey, or WithTokenSource with a source that
// yields one. An OAuth session is refused. The token goes in Authorization as
// a bearer token, or in its own Header (R16).
//
// Only POST {base}/chat/completions is used. A POST {base}/responses endpoint
// also exists, but it is deliberately NOT used: it is undocumented, the
// official docs scope the OpenAI-compatible surface to "chat completion tasks
// only", and on auth failure it returns HTTP 200 with status:"failed" and the
// error in the body (measured 2026-08-29), which status-only classification
// would decode as an empty, error-free success. Do not "fix" this omission
// without re-measuring.
//
// Capabilities: tools are Supported. Forced tool choice is BestEffort: only a
// named tool was measured (see Degradations, 0020-MADR F43). Reasoning is
// BestEffort: Hugging Face documents reasoning_effort as "provider and
// model-dependent", so an ignored effort is a normal outcome. Continuation is
// Unsupported: Chat Completions is stateless. Native streaming is Supported:
// Stream asks for an event stream, with its usage, and yields its events as
// they arrive (0031-MADR E1, E5).
//
// Images are Supported: an ImageItem goes as a Chat Completions image_url
// part, {"url": …}, its bytes as a data URL, joined to the user's message
// before it, whose text becomes a text part. That the router takes a data URL
// and an https URL is not documented; a live test pins it (0061-PLAN Phase
// 7). A model whose facts state its input without image is refused before the
// network, with ErrUnsupported (0061-MADR D5).
//
// Degradations:
//   - A tool result's IsError is not sent: Chat Completions has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - A tool's Strict is sent as function.strict (0032-MADR D2).
//   - Reasoning takes an effort, sent as reasoning_effort (medium with none);
//     a Budget is not sent.
//   - The effort, max included, is sent by the model's facts and what it
//     refused (0062-MADR D2, D3): one the facts do not list, or the model
//     refused, steps to the nearest it takes, lower first, with a
//     WarningEffortDegraded naming what was sent.
//     A refusal of the effort sent, in a form 0062-PLAN Phase 1 recorded,
//     is remembered for the rest of the process, in memory only, and the
//     request is sent again, at most 7 times in all; the Response, or a
//     stream's done event, carries a WarningReasoningLearned whose Refused a
//     caller can keep, and pass back with WithReasoningRefusals on its next
//     start (0062-MADR D4 to D6).
//   - Instructions are sent as a leading system message.
//   - ToolChoiceRequired is sent as "required", the Chat Completions value.
//     ToolChoiceNone is kept by sending no tools and no tool_choice: the
//     router's upstreams answer a call made under "none" with HTTP 400
//     tool_use_failed (0023-MADR, measured 2026-10-05). On Groq, a request
//     after a tool turn can still return ErrInvalidRequest with
//     tool_use_failed: gpt-oss writes a call from the history alone, and
//     no request form avoids it (0023-MADR, amendment 2026-10-06).
//
// ListModels returns the curated router listing, ranked by the metadata the
// router publishes (llmprovider.WithModelMetadataURL). It never probes, as the
// router meters every call (MADR 0012 §1.6).
//
// catalog.ListRaw returns the listing as the router sends it:
// GET https://router.huggingface.co/v1/models (0041-MADR D9).
//
// Request.Cache sends nothing: the router documents no caching parameter
// (0039-MADR D4).
//
// Deferred tools (Tool.Deferred) have no native form here (0040-MADR D6):
// each tool a ToolLoadItem loads is appended to the tools, in load order,
// and a ToolLoadItem itself sends nothing. The response carries
// WarningDeferralEmulated (D7), and the prompt cache breaks once at each
// load.
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, and the
// models.dev document once something has fetched it. A request never
// fetches, and never waits.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// Prior reasoning is replayed under the field the model's interleaved.field
// names, and under none when no source names one.
package huggingface

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
	// defaultBaseURL is the router's base (llmprovider's huggingFaceBaseURL).
	defaultBaseURL      = "https://router.huggingface.co/v1"
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
	// limit is catalog.WithLimit's n, which cuts ListModels's static
	// fallback; zero is no limit (0038-MADR A3).
	limit int
	// thinking is the memo of refused thinking settings, catalog.WithCache's
	// or the process's; refusals are WithReasoningRefusals' (0062-MADR D5,
	// D6).
	thinking *wire.Learned
	refusals []llmprovider.ReasoningRefusal
}

// New builds the Hugging Face provider. It needs a token. No token, an empty
// one, and an option llmprovider.ResolveOptions refuses give an error
// matching llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one
// matching llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderHuggingFace, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: huggingface needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: huggingface api key is required", llmprovider.ErrInvalidRequest)
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: huggingface takes a token, not an OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported)
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
			Tools:            llmprovider.Supported,
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
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderHuggingFace }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the router's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the effort sent is learned, and the
	// request built again (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(llmprovider.ProviderHuggingFace), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) },
		chatcompletions.DecodeFor(string(llmprovider.ProviderHuggingFace)), effort.Check)
}

// learner is req's effort, learned from the model's refusals of it
// (0062-MADR D2 to D6).
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.thinking, string(llmprovider.ProviderHuggingFace), p.baseURL, cmp.Or(req.Model, p.model), p.refusals,
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
		for ev, err := range wire.StreamEventsRetry(ctx, string(llmprovider.ProviderHuggingFace), p.src,
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
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderHuggingFace, cmp.Or(req.Model, p.model),
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
		Provider: string(llmprovider.ProviderHuggingFace),
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
	tools, choice := deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)), req.ToolChoice
	if choice == llmprovider.ToolChoiceNone {
		// The router's upstreams refuse a call made under "none" with HTTP 400
		// tool_use_failed, so none is kept by offering no tools (0023-MADR).
		tools, choice = nil, llmprovider.ToolChoiceAuto
	}
	// By D3's ladder over the model's refusals and facts (0062-MADR D2, D3).
	effort, warnings := learner.Effort(llmprovider.Effort(p.effort(req)), facts.Reasoning.Efforts,
		facts.Has(catalog.FieldReasoningEfforts))
	return chatcompletions.Body(model, maxTokens, input, chatcompletions.Opts{ReasoningEffort: string(effort),
		ReplayReasoningField: replayField(input, facts),
		Tools:                tools, ToolChoice: choice, SendStrict: true}), warnings
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

// ListModels returns the curated router listing, falling back to the static
// catalog. It spends no generation on probes (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderHuggingFace, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderHuggingFace), p.limit)
	}
	return listed, nil
}
