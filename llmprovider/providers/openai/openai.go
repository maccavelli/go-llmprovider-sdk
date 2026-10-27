// Package openai is the OpenAI provider (0015-MADR D2): the Responses API at
// api.openai.com with an API key, or the ChatGPT backend with a ChatGPT
// session (an OpenAI OAuth session, or the Codex CLI's login), or the
// Responses API with a ChatGPT plan session (auth.LoginChatGPTPlan).
//
// Build it with New, or providers.New(llmprovider.ProviderOpenAI, …). The
// credential decides the backend: WithAPIKey, or WithTokenSource with a
// static token, uses the platform; a ChatGPT session uses the backend at
// chatgpt.com; a plan session uses the platform, billed to the user's
// ChatGPT plan (0043-MADR D5).
//
// Capabilities: tools, forced tool choice and reasoning are Supported.
// Continuation is Supported with an API key, and Unsupported with a ChatGPT
// session, which stores no responses (MADR 0012 §4.2), and with a plan
// session, for which OpenAI says to omit previous_response_id. Native
// streaming is Supported: Stream asks the Responses API for an event stream
// and yields its events as they arrive (0031-MADR D1).
//
// Images are Supported with an API key: an ImageItem goes as a Responses
// input_image, its image_url a data URL of its bytes or its URL, joined to
// the user's message before it, whose text becomes an input_text part. On a
// ChatGPT or plan session they are Unsupported, refused before the network,
// since nothing documents images on the Codex backend; a live test decides
// (Q11 (a)). A model whose facts state its input without image is refused
// before the network, with ErrUnsupported (0061-MADR D5).
//
// Degradations:
//   - A tool result's IsError is not sent: the Responses API has no error
//     field, so Output is sent as it is (0032-MADR D3).
//   - A tool's Strict is sent as strict (0032-MADR D2).
//   - Reasoning takes an effort. A Reasoning with only a Budget sends the
//     default effort, medium; the budget is not sent.
//   - The effort, max included, is sent by the model's facts and what it
//     refused (0062-MADR D2, D3): one the facts do not list, or the model
//     refused, steps to the nearest it takes, lower first, with a
//     WarningEffortDegraded naming what was sent; a model that takes no
//     effort (unsupported_parameter) is sent no reasoning.
//     A refusal of the effort sent, in a form 0062-PLAN Phase 1 recorded,
//     is remembered for the rest of the process, in memory only, and the
//     request is sent again, at most 7 times in all; the Response, or a
//     stream's done event, carries a WarningReasoningLearned whose Refused a
//     caller can keep, and pass back with WithReasoningRefusals on its next
//     start (0062-MADR D4 to D6).
//   - A ChatGPT session sends no output limit: the backend rejects
//     max_output_tokens (gate G-C, 2026-09-27), so MaxOutputTokens and
//     WithMaxTokens have no effect there. It always sends store false, so
//     WithStore has no effect there either.
//   - A plan session follows OpenAI's rules for plan usage (0043-MADR D5):
//     every request streams, Generate reading the stream; it sends store
//     false and no output limit, so WithStore, MaxOutputTokens and
//     WithMaxTokens have no effect there; and it sends no tool_search, which
//     OpenAI does not support there, so deferred tools are appended, with
//     WarningDeferralEmulated. A system message in Input is refused before
//     the network, matching llmprovider.ErrUnsupported: OpenAI rejects one
//     there, and system prompts go in Request.Instructions (0043-MADR A4).
//
// ListModels returns the curated listing. With an API key it probes each of
// the first catalog.MaxListed listed models with one short, billed
// generation (0038-MADR D4), unless
// llmprovider.WithModelProbes(false) says otherwise (0016-MADR A5); a ChatGPT
// session is never probed (MADR 0012 §1.6). A plan session lists the account's
// models from the "models" array of GET https://api.openai.com/v1/models,
// those with visibility "list", in the listing's order, and is never probed:
// each probe would spend the user's plan (0043-MADR D5).
//
// catalog.ListRaw returns the listing as OpenAI sends it: with an API key,
// GET https://api.openai.com/v1/models; with a ChatGPT session, the Codex
// backend's GET https://chatgpt.com/backend-api/codex/models?client_version=…;
// with a plan session, GET https://api.openai.com/v1/models and its "models"
// array; unsorted and unfiltered, each entry's slug as its id where the
// array is "models" (0041-MADR D9).
//
// OpenAI caches prompts on its own. Request.Cache sends the provider's
// session id (llmprovider.WithSessionID, else a random id fixed for the
// provider) as prompt_cache_key, which routes a conversation to one cache;
// CacheLong sends the same as CacheShort, since OpenAI offers no longer
// lifetime to ask for (0039-MADR D3). A ChatGPT session sends the key on
// every request, as Codex does; a plan session sends it as a key does.
//
// Deferred tools (Tool.Deferred) go out natively with a key from GPT-5.4 on,
// and in a ChatGPT session for every model (0040-MADR D3, A3), but never in
// a plan session (above): the Search tool as a client tool_search entry, and
// the deferred tools only in the
// tool_search_output that replays a search's result; that result's text is
// not sent. A load from any other result is appended to the tools. An earlier
// model, and a forced choice of the Search tool, which tool_choice cannot
// name there, append each loaded tool instead; with PreviousResponseID that
// form sends every deferred tool (D6). Each of these carries
// WarningDeferralEmulated (D7). A hosted tool_search (execution "server") in
// a reply is dropped.
//
// Model facts (0041-MADR D1, D2, A3). Each request reads what the caches
// already hold of its model: a caller's catalog.WithModelLayer, a listing
// that ListModels or catalog.List read, and the models.dev document once
// something has fetched it. A request never fetches, and never waits.
// With no MaxOutputTokens, the output limit is the default, cut to the
// model's own when that is smaller; it is never raised.
// A ChatGPT session and a plan session send no output limit, as before.
package openai

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

const (
	headerAuthorization = "Authorization"
	// headerSession carries the conversation id, as Codex sends it
	// (codex-api/src/requests/headers.rs:8).
	headerSession   = "session-id"
	headerResidency = "x-openai-internal-codex-residency"

	jsonKeyModel       = "model"
	jsonKeyInput       = "input"
	jsonKeyTools       = "tools"
	jsonKeyType        = "type"
	jsonKeyFunction    = "function"
	jsonKeyName        = "name"
	jsonKeyDescription = "description"
	jsonKeyParameters  = "parameters"
	jsonKeyToolChoice  = "tool_choice"

	// probeMaxOutputTokens is the output limit a listing probe sends: the
	// default of the old API's probe provider, kept so the probe's request
	// is unchanged (0015-MADR D1).
	probeMaxOutputTokens = 8192
	probePrompt          = "Respond with ONLY the word Hello"
)

// The OpenAI endpoints.
const (
	// PlatformBaseURL is the API-key endpoint.
	PlatformBaseURL = "https://api.openai.com/v1"
	// ChatGPTBaseURL is the ChatGPT subscription endpoint, the Codex backend.
	ChatGPTBaseURL = chatgpt.BaseURL
)

// ChatGPTUsageURL is the page where a user manages their ChatGPT plan's usage:
// what a "Manage usage" action opens when a plan session's limit is reached,
// an error matching llmprovider.ErrQuotaExhausted (0043-MADR D6). OpenAI's
// guidance names the action, not the URL; pi links this page.
const ChatGPTUsageURL = "https://chatgpt.com/settings/usage"

// storeOption is WithStore's value.
type storeOption bool

// WithStore sets whether an API key's responses are stored (MADR 0012 §6).
// Without it the service default applies, which keeps continuation working;
// callers under zero data retention pass false, after which a
// PreviousResponseID has nothing to chain from. A ChatGPT session and a plan
// session always send false.
func WithStore(store bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderOpenAI, "openai.WithStore", storeOption(store))
}

type provider struct {
	src     llmprovider.TokenSource
	chatGPT bool
	// plan is a ChatGPT plan session: the platform's URL, with OpenAI's rules
	// for plan usage (0043-MADR D5).
	plan      bool
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	store     *bool                  // WithStore's value, or nil for the service default
	userAgent string
	session   string
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

// New builds the OpenAI provider. It needs a credential: WithAPIKey, or
// WithTokenSource. No credential, an empty key, and an option
// llmprovider.ResolveOptions refuses give an error matching
// llmprovider.ErrInvalidRequest; an OAuth session that is not OpenAI's, or
// another vendor's CLI session, one matching llmprovider.ErrUnsupported.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderOpenAI, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	if src == nil {
		return nil, fmt.Errorf("%w: openai needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	}
	switch s := src.(type) {
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: openai api key is required", llmprovider.ErrInvalidRequest) // 0020-MADR F52
		}
	case *auth.OAuthSession:
		// R16: another provider's session is refused, so its token never
		// reaches OpenAI (0026-MADR F2).
		if owner := s.Owner(); owner != llmprovider.ProviderOpenAI {
			return nil, fmt.Errorf("%w: openai takes an API key, a ChatGPT sign-in or the Codex CLI's login, not a %q session (0026-MADR F2)",
				llmprovider.ErrUnsupported, owner)
		}
	case *auth.VendorCLISession:
		if s.Provider != llmprovider.ProviderOpenAI {
			return nil, fmt.Errorf("%w: openai takes an API key, a ChatGPT sign-in or the Codex CLI's login, not the %q CLI's login (0026-MADR F2)",
				llmprovider.ErrUnsupported, s.Provider)
		}
	}
	p := &provider{
		src:       src,
		chatGPT:   isChatGPTSession(src),
		plan:      chatgpt.IsPlanSession(src),
		model:     st.Model(),
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		session:   st.SessionID(),
		probe:     st.ModelProbes(),
		cache:     cacheref.From(st.Values()),
		refusals:  st.ReasoningRefusals(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.Supported,
			Reasoning:        llmprovider.Supported,
			Continuation:     llmprovider.Supported,
			NativeStreaming:  llmprovider.Supported,
			Images:           llmprovider.Supported,
		},
	}
	for _, v := range st.Values() {
		if store, ok := v.(storeOption); ok {
			s := bool(store)
			p.store = &s
		}
	}
	p.baseURL = PlatformBaseURL
	if p.chatGPT {
		p.baseURL = ChatGPTBaseURL
		p.caps.Continuation = llmprovider.Unsupported
	}
	if p.plan {
		// "Omit previous_response_id over HTTP" (0043-MADR D5).
		p.caps.Continuation = llmprovider.Unsupported
	}
	if p.chatGPT || p.plan {
		// Nothing documents images on the Codex backend; a live test decides
		// (0061-MADR D5, Q11 (a)).
		p.caps.Images = llmprovider.Unsupported
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
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(p.session), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderOpenAI }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Responses API. A 401 from a session that can
// refresh is retried once with a new token.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the effort sent is learned, and the
	// request built again (0062-MADR D5).
	effort := p.learner(req)
	return wire.Send(ctx, string(llmprovider.ProviderOpenAI), p.src,
		func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, false, effort) }, p.decode(req), effort.Check)
}

// Stream streams req from the Responses API as it is generated (0031-MADR
// D1): the request asks for an event stream, a ChatGPT session's as always,
// and each event is yielded as it arrives. A 401 before the first event
// renews a session's token and sends once more.
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	if err := p.check(req); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	_, decoder := p.deferForm(req)
	return func(yield func(llmprovider.Event, error) bool) {
		// Each range is its own request, with its own effort learner.
		effort := p.learner(req)
		for ev, err := range wire.StreamEventsRetry(ctx, string(llmprovider.ProviderOpenAI), p.src,
			func(token llmprovider.Token) wire.Call { return p.call(ctx, req, token, true, effort) }, decoder.Events,
			effort.Check) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// learner is req's effort, learned from the model's refusals of it
// (0062-MADR D2 to D6).
func (p *provider) learner(req *llmprovider.Request) *wire.Learner {
	return wire.NewLearner(p.cache.LearnedThinking(), string(llmprovider.ProviderOpenAI), p.baseURL,
		cmp.Or(req.Model, p.model), p.refusals, responses.MatchRefusal)
}

// check refuses, before the network, what the capabilities refuse, and a
// system message in a plan session's input, which OpenAI rejects there:
// system prompts go in Request.Instructions (0043-MADR A4).
func (p *provider) check(req *llmprovider.Request) error {
	if err := p.caps.Check(req); err != nil {
		return err
	}
	if p.plan && slices.ContainsFunc(req.Input, func(item llmprovider.Item) bool {
		message, ok := item.(llmprovider.MessageItem)
		return ok && message.Role == llmprovider.RoleSystem
	}) {
		return fmt.Errorf("%w: openai: a ChatGPT plan session takes no system message in Input; "+
			"put it in Request.Instructions (0043-MADR A4)", llmprovider.ErrUnsupported)
	}
	return nil
}

// facts is what the caches know of req's model, read for each request and
// never waited for (0041-MADR D1). The request's own token finds the cached
// listing, so the token source is not asked again.
func (p *provider) facts(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) catalog.ModelInfo {
	info, _ := catalog.CachedFacts(ctx, llmprovider.ProviderOpenAI, cmp.Or(req.Model, p.model),
		slices.Concat(p.listing, []llmprovider.Option{llmprovider.WithAPIKey(token.Value)})...)
	return info
}

// deferForm is req's deferred-tool form, and the decoder its reply needs: a
// ChatGPT session takes the native form for every model, a key from GPT-5.4
// on, and the native form's decoder names the Search tool (0040-MADR D1, D3).
// A plan session appends: OpenAI lists tool_search as unsupported there
// (0043-MADR D5).
func (p *provider) deferForm(req *llmprovider.Request) (deferral.Form, responses.Decoder) {
	if p.plan {
		return deferral.Append, responses.Decoder{}
	}
	model := req.Model
	if model == "" {
		model = p.model
	}
	route := deferral.Responses
	if p.chatGPT {
		route = deferral.ChatGPT
	}
	form := deferral.ForRequest(route, model, req)
	var decoder responses.Decoder
	if form == deferral.ResponsesNative {
		decoder.Search = deferral.SearchName(req.Tools)
	}
	return form, decoder
}

// body is the Responses request for req.
// includeEncryptedReasoning asks a stateless Responses request for its
// reasoning, encrypted, so the next turn can replay it.
const includeEncryptedReasoning = "reasoning.encrypted_content"

func (p *provider) body(req *llmprovider.Request, facts catalog.ModelInfo, learner *wire.Learner) (map[string]any,
	[]llmprovider.Warning) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	form, _ := p.deferForm(req)
	input := responses.Input(req.Input)
	if form == deferral.ResponsesNative {
		input = responses.InputLoading(req.Input, req)
	}
	body := map[string]any{
		jsonKeyModel: model,
		jsonKeyInput: input,
	}
	if req.Instructions != "" {
		body["instructions"] = req.Instructions
	}
	if p.chatGPT {
		// The ChatGPT backend takes only Codex's shape: it rejects a
		// non-streaming request, a missing store:false and max_output_tokens
		// (gate G-C, 2026-09-27; codex core/src/client.rs:1007-1008).
		body["stream"] = true
		body["store"] = false
		body["include"] = []string{includeEncryptedReasoning}
		body["prompt_cache_key"] = p.session
	} else if p.plan {
		// OpenAI's rules for plan usage: stream, store false, and no
		// max_output_tokens, which it says to omit though M5 measured it
		// accepted (0043-MADR D5). Stateless, so reasoning comes back
		// encrypted, as with WithStore(false); the cache key goes as a key's.
		body["stream"] = true
		body["store"] = false
		body["include"] = []string{includeEncryptedReasoning}
		wire.AddPromptCacheKey(body, req.Cache, p.session)
	} else {
		maxTokens := p.maxTokens
		if req.MaxOutputTokens > 0 {
			maxTokens = req.MaxOutputTokens
		} else if limit := facts.MaxOutputTokens; limit > 0 && facts.Has(catalog.FieldMaxOutputTokens) {
			// The default, cut to the model's limit, never raised (0041-MADR D2).
			maxTokens = min(maxTokens, limit)
		}
		body["max_output_tokens"] = maxTokens
		if p.store != nil {
			body["store"] = *p.store
			if !*p.store {
				// Stateless, so reasoning comes back encrypted to be replayed,
				// as Codex asks on every request (0021-MADR D4).
				body["include"] = []string{includeEncryptedReasoning}
			}
		}
		// The ChatGPT backend sends the key always, above (0039-MADR D3).
		wire.AddPromptCacheKey(body, req.Cache, p.session)
	}
	if tools := deferral.Tools(req, form); form == deferral.ResponsesNative {
		wire.AddResponsesToolsNative(body, tools, req.ToolChoice)
	} else {
		wire.AddResponsesTools(body, tools, req.ToolChoice)
	}
	want, _ := p.effort(req)
	// The effort by D3's ladder, over the model's refusals and facts; none
	// left out leaves out reasoning (0062-MADR D2, D3).
	effort, warnings := learner.Effort(want, facts.Reasoning.Efforts, facts.Has(catalog.FieldReasoningEfforts))
	if effort != "" {
		// The summary makes reasoning items carry text (0020-MADR F24).
		body["reasoning"] = map[string]any{"effort": effort, "summary": "auto"}
	}
	if req.PreviousResponseID != "" {
		body["previous_response_id"] = req.PreviousResponseID
	}
	return body, warnings
}

// effort is the reasoning effort to send, and false for none: the request's
// Reasoning, else WithReasoning's, with medium when neither names an effort.
func (p *provider) effort(req *llmprovider.Request) (llmprovider.Effort, bool) {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return "", false
	}
	switch {
	case r.Effort != "":
		return r.Effort, true
	case p.reasoning != nil && p.reasoning.Effort != "":
		return p.reasoning.Effort, true
	default:
		return llmprovider.EffortMedium, true
	}
}

// decode is req's reply decoder. A ChatGPT session's reply, and a plan
// session's, is an event stream, bounded per event rather than as a whole.
func (p *provider) decode(req *llmprovider.Request) func(io.Reader) (*llmprovider.Response, error) {
	// A failed reply is classified by OpenAI's vocabulary, as the stream's
	// is (0026-MADR F28).
	_, decoder := p.deferForm(req)
	if p.chatGPT || p.plan {
		return func(r io.Reader) (*llmprovider.Response, error) {
			return decoder.ReadStream(string(llmprovider.ProviderOpenAI), r)
		}
	}
	return decoder.DecodeFor(string(llmprovider.ProviderOpenAI))
}

// call is the Responses request for req, asking for an event stream when
// stream is set; a ChatGPT session's and a plan session's always do. Only a
// ChatGPT session sends the Codex backend's headers. effort chooses the
// effort.
func (p *provider) call(ctx context.Context, req *llmprovider.Request, token llmprovider.Token, stream bool,
	effort *wire.Learner) wire.Call {
	facts := p.facts(ctx, req, token)
	body, warnings := p.body(req, facts, effort)
	if stream {
		body["stream"] = true
	}
	streamed := stream || p.chatGPT || p.plan
	form, _ := p.deferForm(req)
	return wire.Call{
		Provider: string(llmprovider.ProviderOpenAI),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/responses",
		Body:     body,
		Stream:   streamed,
		// Deferred tools appended, before GPT-5.4 or for a non-search load
		// (0040-MADR D7); an effort sent lower, and a refusal learned
		// (0062-MADR D3, D6).
		Warnings: append(deferral.Warnings(form, req, cmp.Or(req.Model, p.model)), warnings...),
		// A model whose facts take no image is refused before the network
		// (0061-MADR D5).
		Refusal: wire.RefuseImages(cmp.Or(req.Model, p.model), req.Input, facts.Has(catalog.FieldInput),
			slices.Contains(facts.Input, catalog.ModalityImage)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			token.Apply(r, headerAuthorization, "Bearer")
			if streamed {
				r.Header.Set("Accept", "text/event-stream")
			}
			if p.chatGPT {
				r.Header.Set(headerSession, p.session)
				setChatGPTHeaders(r, p.src)
				if residency := residency(token.Value); residency != "" {
					r.Header.Set(headerResidency, residency)
				}
			}
		},
	}
}

// ListModels returns the curated chat models this credential can use, never
// the raw /v1/models dump. An API key's models are probed unless
// WithModelProbes(false); a ChatGPT session's are not (MADR 0012 §1.6), nor
// a plan session's (0043-MADR D5).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	// A plan session lists its account's catalog, with no static fallback,
	// since those models may not be on the plan, and no probe, which the
	// plan would pay for. A 401 renews the token and lists once more.
	if p.plan {
		return wire.Reauth(ctx, string(llmprovider.ProviderOpenAI), p.src, func(token llmprovider.Token) ([]string, error) {
			models, err := p.listPlan(ctx, token)
			return listlimit.Cut(models, p.limit), err
		})
	}
	// A ChatGPT session lists from the Codex backend, with no static catalog
	// (MADR 0008 D11), and its subscription meters every call: no generation
	// probe (MADR 0012 §1.6). A 401 renews the session's token and lists
	// once more, as Generate does (0020-MADR F2, F38).
	if p.chatGPT {
		return wire.Reauth(ctx, string(llmprovider.ProviderOpenAI), p.src, func(token llmprovider.Token) ([]string, error) {
			models, err := p.listChatGPT(ctx, token)
			return listlimit.Cut(models, p.limit), err
		})
	}
	cat, err := catalog.List(ctx, llmprovider.ProviderOpenAI, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(llmprovider.ProviderOpenAI), p.limit)
	}
	if !p.probe {
		return listed, nil
	}
	healthy := wire.ProbeHealth(ctx, p.cache.ProbeStore(), llmprovider.ProviderOpenAI, p.baseURL, p.src, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
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

// residency reads chatgpt_compute_residency from the access token for the
// residency header. Its source is OpenCode's Codex plugin
// (plugin/openai/codex.ts:83, :426), not Codex (MADR 0012 §4.4).
func residency(accessToken string) string {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Residency *string `json:"chatgpt_compute_residency"`
		Auth      struct {
			Residency *string `json:"chatgpt_compute_residency"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	value := claims.Auth.Residency
	if value == nil {
		value = claims.Residency
	}
	if value == nil || *value == "" || *value == "no_constraint" {
		return ""
	}
	return *value
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
