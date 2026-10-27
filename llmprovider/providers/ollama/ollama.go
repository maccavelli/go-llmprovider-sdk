// Package ollama is the provider for a local Ollama instance (0015-MADR D2),
// through its OpenAI-compatible Chat Completions endpoint.
//
// Build it with New, or providers.New(llmprovider.ProviderOllama, …). It needs
// no credential: Ollama's API key is "required but ignored", so no
// Authorization header is sent, with or without WithAPIKey. A token naming its
// own Header is sent there, for an instance behind a proxy that wants one
// (R16). An OAuth session is refused. The base URL defaults to
// http://localhost:11434; llmprovider.WithBaseURL names another instance.
//
// Generation uses POST {base}/v1/chat/completions. Listing stays on the native
// GET {base}/api/tags, which is stable; /v1/models is a compatibility shim.
//
// Capabilities: tools are Supported. Forced tool choice is BestEffort: Ollama
// does not support tool_choice, so tools are offered, never forced. Reasoning
// is BestEffort: only thinking models reason. Continuation is Unsupported:
// Chat Completions is stateless. Native streaming is Supported:
// Stream asks for an event stream, with its usage, and yields its events as
// they arrive (0031-MADR E1, E5).
//
// Images are BestEffort: an ImageItem's bytes go as a Chat Completions
// image_url part whose image_url is the data URL alone, a string, as Ollama
// documents it, joined to the user's message before it; an image by URL is
// dropped, as Degradations lists (E9). A model whose facts state its input
// without image is refused before the network, with ErrUnsupported (0061-MADR
// D5).
//
// Degradations:
//   - An image by URL is not sent: Ollama marks image URLs unsupported, so it
//     is dropped, with a WarningImageDropped naming its index in
//     Request.Input (0061-MADR D5, E9).
//   - A tool result's IsError is not sent: Chat Completions has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - A tool's Strict is not sent: Ollama's acceptance of function.strict
//     has not been measured (0032-MADR D2).
//   - No tool choice is sent: a named tool and ToolChoiceRequired offer the
//     tools unforced, and ToolChoiceNone sends no tools (0020-MADR F40).
//   - Reasoning takes an effort, sent as reasoning_effort (medium with none).
//     A Budget is not sent.
//   - The effort, xhigh and max included, is sent by the model's facts and
//     what it refused (0062-MADR D2, D3): one the facts do not list, or the
//     model refused, steps to the nearest it takes, lower first, with a
//     WarningEffortDegraded naming what was sent. Ollama 0.33.2 takes xhigh
//     and max (0062-REPORT-thinking-refusals.md E12), so xhigh is no longer
//     sent as max. A model that does not think refuses every effort
//     (`"<model>" does not support thinking`): that refusal is learned, with a
//     WarningReasoningLearned, and the request sent again with no effort,
//     warned, as every later one to the model in the process is (0061-MADR,
//     amendment "Ollama's refusal of thinking, learned"). No refusal of one
//     effort among others was recorded; a refusal passed back with
//     WithReasoningRefusals is never sent (0062-MADR, amendment "Phase 5's
//     Ollama cases").
//   - Instructions are sent as a leading system message.
//
// ListModels returns the installed models, probing each listed model with one
// short generation unless llmprovider.WithModelProbes(false) says otherwise
// (0016-MADR A5). There is no static catalog: installed models are
// machine-specific, so a failed listing returns its error.
//
// catalog.ListRaw returns the listing as the server sends it: GET /api/tags
// alone, each entry's name as its id, and no /api/show (0041-MADR D9).
//
// The wire shapes were measured against a running v0.31.1 on the date
// llmprovider's wireShapesProbedOnOllama holds.
//
// Request.Cache sends nothing: a local server takes no caching parameter
// (0039-MADR D4).
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
package ollama

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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
)

const (
	// defaultBaseURL is a local instance (llmprovider's ollamaBaseURL).
	defaultBaseURL = "http://localhost:11434"

	// probeMaxOutputTokens is the output limit a listing probe sends: the
	// default of the old API's probe provider, kept so the probe's request
	// is unchanged (0015-MADR D1).
	probeMaxOutputTokens = 8192
	probePrompt          = "Respond with ONLY the word Hello"
)

type provider struct {
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
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
	// thinking is the memo of refused thinking settings, catalog.WithCache's
	// or the process's; refusals are WithReasoningRefusals' (0062-MADR D5,
	// D6).
	thinking *wire.Learned
	refusals []llmprovider.ReasoningRefusal
}

// New builds the Ollama provider. It needs no credential. An option
// llmprovider.ResolveOptions refuses gives an error matching
// llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one matching
// llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderOllama, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch src.(type) {
	case nil:
		src = llmprovider.NewStaticToken("")
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: ollama takes no OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported)
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   defaultBaseURL,
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		probe:     st.ModelProbes(),
		cache:     cacheref.From(st.Values()),
		thinking:  cacheref.From(st.Values()).LearnedThinking(),
		refusals:  st.ReasoningRefusals(),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Supported,
			Images:           llmprovider.BestEffort,
		},
	}
	if b := st.BaseURL(); b != "" {
		p.baseURL = strings.TrimRight(b, "/")
	}
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderOllama }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the instance's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); the effort is learned's (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(llmprovider.ProviderOllama), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) },
		chatcompletions.DecodeFor(string(llmprovider.ProviderOllama)), effort.Check)
}

// learner is req's effort, from the model's refusals and facts (0062-MADR D2
// to D6). Its matcher is the Chat Completions wire's, which reads Ollama's
// refusal of thinking by a model that does not think as a refusal of the
// effort field (0061-MADR, amendment "Ollama's refusal of thinking,
// learned"); no refusal of one effort among others was recorded (0062-MADR,
// amendment "Phase 5's Ollama cases").
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.thinking, string(llmprovider.ProviderOllama), p.baseURL, cmp.Or(req.Model, p.model), p.refusals,
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
		for ev, err := range wire.StreamEventsRetry(ctx, string(llmprovider.ProviderOllama), p.src,
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
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderOllama, cmp.Or(req.Model, p.model),
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
		Provider: string(llmprovider.ProviderOllama),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/v1/chat/completions",
		Body:     body,
		Stream:   stream,
		Warnings: append(deferral.Warnings(deferral.ForRequest(deferral.Other, model, req), req, model), warnings...), // 0040-MADR D7; 0062-MADR D3, D6
		// A model whose facts take no image is refused before the network
		// (0061-MADR D5).
		Refusal: wire.RefuseImages(cmp.Or(req.Model, p.model), req.Input, facts.Has(catalog.FieldInput),
			slices.Contains(facts.Input, catalog.ModalityImage)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			// Ollama takes no credential: a token is sent only where it names its
			// own Header.
			if token.Header != "" {
				token.Apply(r, "", "")
			}
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
	// Ollama marks image URLs unsupported, so an image by URL is dropped,
	// warned; an image's bytes go as a data URL (0061-MADR D5, E9).
	input, dropped := wire.DropImageURLs(model, req.Input, "Ollama takes an image's bytes only, not its URL")
	if req.Instructions != "" {
		input = append([]llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: req.Instructions}}, input...)
	}
	effort, warnings := p.effort(req, facts, learner)
	// Ollama does not support tool_choice: every tool is offered and none
	// forced, and ToolChoiceNone sends no tools (0020-MADR F40). Deferred
	// tools are appended as loaded (0040-MADR D6). An image's image_url is
	// the data URL alone, a string, as Ollama documents it (0061-MADR D5, E9).
	return chatcompletions.Body(model, maxTokens, input, chatcompletions.Opts{ReasoningEffort: effort,
		ReplayReasoningField: replayField(input, facts),
		Tools:                deferral.Tools(req, deferral.ForRequest(deferral.Other, model, req)), ToolChoice: req.ToolChoice, NoToolChoice: true,
		ImageURLString: true}), append(dropped, warnings...)
}

// effort is reasoning_effort for req: the request's Reasoning, else
// WithReasoning's, an empty effort taken from WithReasoning, else medium; ""
// for no reasoning. It is sent by D3's ladder over the model's refusals and
// facts, with the warning when it is sent lower (0062-MADR D2, D3).
func (p *provider) effort(req *llmprovider.Request, facts catalog.ModelInfo, learner *wire.Learner) (string,
	[]llmprovider.Warning) {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return "", nil
	}
	effort := llmprovider.EffortMedium
	switch {
	case r.Effort != "":
		effort = r.Effort
	case p.reasoning != nil && p.reasoning.Effort != "":
		effort = p.reasoning.Effort
	}
	sent, warnings := learner.Effort(effort, facts.Reasoning.Efforts, facts.Has(catalog.FieldReasoningEfforts))
	return string(sent), warnings
}

// ListModels returns the installed models, probed unless
// WithModelProbes(false). A failed listing returns its error: there is no
// static catalog to fall back on.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderOllama, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		return nil, err
	}
	// Probes are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := wire.ProbeHealth(ctx, p.cache.ProbeStore(), llmprovider.ProviderOllama, p.baseURL, p.src, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		// The old API's probe provider: the default output limit, no
		// reasoning (0015-MADR D1).
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
