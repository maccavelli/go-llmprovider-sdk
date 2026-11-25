// Package opencode is the OpenCode Zen and OpenCode Go gateway provider
// (0015-MADR D2). Both gateways are multi-protocol: they dispatch each model
// to one of four upstream wire formats (Responses, Anthropic Messages, Gemini
// generateContent, Chat Completions) and do not normalize them, so the
// provider picks the request shape, path, header and decoder per request.
//
// Build Zen with NewZen and Go with NewGo, or providers.New with
// llmprovider.ProviderOpencodeZen or ProviderOpencodeGo (0015-MADR,
// amendment "the OpenCode family"). The credential is an API key, WithAPIKey
// or WithTokenSource; without one, the gateway's public token is sent, which
// free models accept (MADR 0012 §1.7). An OAuth session is refused. Each
// route reads the key from its vendor's header, or the token's own Header
// (R16).
//
// The route for a request is WithRoute's, else the model's provider.npm in the
// models.dev document given to New with catalog.WithMetadata (0093-MADR D6),
// else the built-in table,
// else a prefix heuristic (MADR 0012 §3.1). Sending a model to the wrong route
// fails with an opaque HTTP 500, which classifies as the retryable
// ErrProviderUnavailable; WithRoute pins a route.
//
// Capabilities: tools are Supported. Forced tool choice is BestEffort: it
// is sent as asked, and some models refuse it (below). Reasoning is
// BestEffort: what a model takes is learned from its refusals, and the
// google route learns nothing (below). Continuation is
// Unsupported: the gateway rejects previous_response_id with HTTP 400
// (measured 2026-08-28), so a caller replays prior items. Native streaming is
// BestEffort: Stream asks every route for an event stream and yields its
// events as they arrive, but the google route's stream was measured on
// Google's own endpoint, not through either gateway, whose account was
// refused (0031-PLAN-native-streaming-on-the-other-wires D1), and Zen's routes
// stand on Go's measurements (0031-MADR D1, E1).
//
// Images are BestEffort, sent in each route's form, as the provider sends it
// alone: messages as claude's, responses as openai's, chat_completions as an
// image_url object, and google as inline_data, an image's bytes only, an
// image by URL dropped, as Degradations lists. OpenCode documents images on
// no route (E15). A model whose facts state its input without image is
// refused before the network, with ErrUnsupported (0061-MADR D5).
//
// Degradations:
//   - On the google route an image by URL is not sent: no generateContent
//     form for one is confirmed, so it is dropped, with a WarningImageDropped
//     naming its index in Request.Input (0061-MADR D5, E7).
//   - A tool result's IsError is sent as is_error on the messages route and
//     as functionResponse.response {"error": …} on the google route; the
//     responses and chat_completions routes have no error field, so Output
//     is sent as it is (0032-MADR D3).
//   - A tool's Strict is sent as strict on the responses and messages
//     routes and as function.strict on the chat_completions route, and not
//     on the google route, which has no such field (0032-MADR D2;
//     0034-MADR D4).
//   - Reasoning: the responses route takes an effort (medium when none is
//     named); the messages and google routes take the effort or the budget,
//     in the shape the model's facts give, else the request's: on the
//     messages route as the claude package sends it, and on the google route
//     an effort as thinkingLevel and a Budget as thinkingBudget, where
//     0013-MADR B9 sent a dynamic budget (0062-MADR D2, amendment "Phase 3's
//     google-route levels, and 0013's B9"); the chat route takes
//     reasoning_effort, by the efforts the model's published
//     reasoning_options list, and as asked when they list none or no
//     document is handed in (0062-MADR Q7 (a); it sent none, MADR 0009 §6).
//   - The effort, max included on every route, is sent by the model's facts
//     and what it refused (0062-MADR D2, D3): one the facts do not list, or
//     the model refused, steps to the nearest it takes, lower first, with a
//     WarningEffortDegraded naming what was sent. A refusal of the effort
//     sent, in the form its route's wire was recorded refusing one
//     (0062-PLAN Phase 1), is remembered for the rest of the process, in
//     memory only, and the request is sent again, at most 7 times in all;
//     the Response, or a stream's done event, carries a
//     WarningReasoningLearned whose Refused a caller can keep, and pass back
//     with WithReasoningRefusals (D4 to D6). The messages route learns a
//     refused thinking shape as the claude package does (D3); the google
//     route a refused thinkingLevel, as Zen's gemini-3.8-flash refused xhigh
//     (0062-REPORT-thinking-refusals.md E11).
//   - On the messages route, a forced tool choice is sent as "auto" with
//     reasoning: Anthropic refuses to force a tool while thinking, and the
//     Response carries a WarningForcedChoiceDegraded (0034-MADR D7). On
//     claude-fable-5-1, claude-mythos-5-1, claude-opus-5-5 and
//     claude-sonnet-5-5, as the claude provider does: a named tool is sent
//     as structured outputs and returned as its call (D8); ToolChoiceRequired
//     fails before the network with ErrUnsupported unless
//     WithForcedChoiceFallback(ForcedChoiceDegrade) sends it as "auto" with a
//     turn instruction and a warning (D2, D10); WithModelForcedChoice
//     overrides the list (D9). Another model that refuses a forced choice in
//     Anthropic's form is remembered for the process, and handled as above,
//     with a WarningForcedChoiceLearned (D3); the qwen refusals below have
//     another form, and are not.
//   - Instructions are sent as a leading system item on every route: the
//     system field on the messages route, systemInstruction on the google
//     route, a system message elsewhere.
//   - ToolChoiceRequired and ToolChoiceNone use each route's documented form,
//     measured on Go's chat, messages and responses routes on 2026-10-02
//     (TestLive_OpencodeToolChoices).
//   - A tool choice is sent as asked, and models differ in honouring it.
//     Measured 2026-10-02 on Go's messages route: qwen3.8-flash and
//     qwen3.7-plus refuse a forced choice (ToolChoiceRequired or a named
//     tool) with an error matching ErrInvalidRequest; the minimax models
//     accept every choice but may ignore it (minimax-m2.7 honours
//     ToolChoiceRequired and ignores ToolChoiceNone).
//   - Request.Cache, on the messages route, sends Anthropic's markers as the
//     claude package does: the top-level cache_control and one breakpoint
//     on the system prompt or the last tool, with "5m" or "1h"
//     (0039-MADR D2, D4). The x-opencode-session header is sent either way.
//   - Deferred tools (Tool.Deferred) go out natively on the messages route
//     from Claude 4.5 on, as the claude package sends them, and on the
//     responses route from GPT-5.4 on, as the openai package sends them
//     (0040-MADR D2, D3). The chat and google routes, and every other
//     model, append each loaded tool, and the response carries
//     WarningDeferralEmulated (D6, D7).
//   - Model facts (0041-MADR D1, D2, A3): each request reads what the caches
//     hold of its model, a caller's catalog.WithModelLayer and the models.dev
//     document given to New (0093-MADR D2), and never fetches. From them come the messages
//     route's thinking shape, as the claude package sends it; the google
//     route's thinkingLevel, from the levels the model lists. With no
//     facts the request decides the shape (0062-MADR D2).
//   - The output limit (0089-MADR D6, D7): a request's own MaxOutputTokens
//     is sent as given, and WithMaxTokens' value cut to the model's stated
//     limit, never raised. Otherwise the messages route, whose wire requires
//     a limit, sends the model's chosen output fact (catalog.CachedLimits),
//     held to its window and to a cap the gateway stated; with no fact, the
//     smallest limit among the models OpenCode's route table sends to that
//     route (DL51, DL77); with none of those, the request is refused before
//     the network with ErrInvalidRequest, its message saying how to state
//     the limit. The fact or that smallest limit is held to the room the
//     stated window, else the stated input limit, leaves the request's
//     input, estimated from its content as pi does (0088-MADR ST87, ST105).
//     A refusal of a limit the SDK chose, in Anthropic's form and
//     validated as 0089-MADR D4 says, is sent once more at the stated cap,
//     which is kept for the process unless llmprovider.WithoutLimitLearning
//     is given, and reported in a WarningLimitLearned, or, on a call that
//     fails, as its error's APIError.Observed (DL78). A caller's own limit
//     refused in a cap form, Anthropic's on the messages route, OpenAI's on
//     the chat route or Gemini's range on the google route, is returned, not
//     resent, and its cap kept alike; a call that fails after a cap was kept
//     carries it as its error's APIError.Observed (DL78, DL86). The
//     responses, chat and google routes send none unless the caller gives one
//     (DL47). The Response, and a stream's done event, carry the limit sent
//     in OutputLimit.
//
// A refusal of a tool's schema is an *APIError of kind
// llmprovider.ErrToolSchemaRefused when its reply carries a marker the
// gateway was recorded sending, on any route: the code
// invalid_function_parameters, as OpenAI sends it and Zen's responses route
// did, with the tool named in the message's function '<name>'; or
// INVALID_ARGUMENT naming a functionDeclaration parameters schema, with the
// tool in backticks before it, which the google route was recorded sending.
// Both are read on every route of the gateway, by marker. Tool is that name when the
// request sent a tool of it. Anthropic's form is not read here: the messages
// route was recorded with no refusal. A refusal of anything else, such as
// the google route's additionalProperties, stays an invalid request.
// A failure inside a 200 body or a stream is not classified as the
// kind and keeps the kind it had, and a reply marked x-should-retry: true,
// or one that says the prompt is too long, keeps the kind those give it
// (0084-MADR D1 to D4, N4, N5).
//
// ListModels returns the curated gateway listing. It never probes: the
// gateways meter every call (MADR 0012 §1.6).
//
// catalog.ListRaw returns the listing as the gateway sends it:
// GET https://opencode.ai/zen/v1/models for Zen, and
// GET https://opencode.ai/zen/go/v1/models for Go (0041-MADR D9).
package opencode

import (
	"cmp"
	"context"
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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/limitmemo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/deferral"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/generatecontent"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

const (
	zenBaseURL = "https://opencode.ai/zen/v1"
	goBaseURL  = "https://opencode.ai/zen/go/v1"

	// sessionHeader carries a stable conversation id. OpenCode Go rejects
	// requests without it (400 MissingSessionID, measured 2026-09-26).
	sessionHeader = "x-opencode-session"
	// publicToken is the key the Zen/Go server treats as anonymous; free
	// models answer it and paid ones fail with a typed error (MADR 0012 §1.7).
	publicToken = "public"

	headerAuthorization = "Authorization"

	jsonKeyType        = "type"
	jsonKeyName        = "name"
	jsonKeyFunction    = "function"
	jsonKeyDescription = "description"
	jsonKeyParameters  = "parameters"
	jsonKeyTools       = "tools"
	jsonKeyToolChoice  = "tool_choice"
	jsonKeyMaxTokens   = "max_tokens"
	jsonKeyMode        = "mode"
)

// family is the ids opencode's options are scoped to.
var family = []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo}

// routeOption is WithRoute's value.
type routeOption Route

// WithRoute pins the wire format, over the metadata, the table and the
// heuristic. Use it when the gateway adds a model before this package's table
// knows it. Both gateways take it; NewZen and NewGo refuse an unknown route.
func WithRoute(route Route) llmprovider.Option {
	return llmprovider.ScopedOptionFor(family, "opencode.WithRoute", routeOption(route))
}

type provider struct {
	gateway   llmprovider.ProviderID
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	// maxGiven reports that WithMaxTokens gave maxTokens (0088-MADR ST15);
	// learn is false under llmprovider.WithoutLimitLearning (0089-MADR DL73).
	maxGiven bool
	learn    bool
	// limits is catalog.WithCache's memo of output caps, nil for the
	// process's; defaultBase is the gateway's own base URL (0089-MADR D5).
	limits      *limitmemo.Memo
	defaultBase string
	reasoning   *llmprovider.Reasoning // WithReasoning's default, or nil
	forcing     messages.Policy        // WithForcedChoiceFallback's and WithModelForcedChoice's
	route       Route                  // WithRoute's, or the table's for model
	routePinned bool
	// meta is the models.dev document given to New with catalog.WithMetadata,
	// the only one its requests read (0093-MADR D2, Q4); zero for none.
	meta      catalog.Metadata
	userAgent string
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

// NewZen builds the OpenCode Zen provider. Without a key it sends the
// gateway's public token, and so for a key that is empty once trimmed
// (llmprovider.NewStaticToken). An option llmprovider.ResolveOptions refuses,
// and a WithRoute naming no route, give an error matching
// llmprovider.ErrInvalidRequest; an OAuth or vendor CLI session, one matching
// llmprovider.ErrUnsupported.
func NewZen(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return newGateway(llmprovider.ProviderOpencodeZen, zenBaseURL, opts)
}

// NewGo builds the OpenCode Go provider. Without a key, or with one that is
// empty once trimmed (llmprovider.NewStaticToken), it sends the gateway's public
// token. Its errors are NewZen's.
func NewGo(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return newGateway(llmprovider.ProviderOpencodeGo, goBaseURL, opts)
}

func newGateway(gateway llmprovider.ProviderID, base string, opts []llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(gateway, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		src = llmprovider.NewStaticToken(publicToken)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			src = &llmprovider.StaticToken{Value: publicToken, Header: s.Header}
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: %s takes an API key, not an OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported, gateway)
	}
	p := &provider{
		gateway:     gateway,
		src:         src,
		model:       st.Model(),
		baseURL:     base,
		client:      st.HTTPClient(),
		maxTokens:   st.MaxTokens(),
		maxGiven:    st.MaxTokensGiven(),
		learn:       !limitmemo.LearningOff(st.Values()),
		limits:      cacheref.From(st.Values()).LimitMemo(),
		defaultBase: base,
		reasoning:   st.Reasoning(),
		forcing:     messages.NewPolicy(st.ForcedChoiceFallback(), st.ModelForcedChoices(), cacheref.From(st.Values()).LearnedRefusals()),
		userAgent:   st.UserAgent(),
		session:     st.SessionID(),
		logger:      st.Logger(),
		thinking:    cacheref.From(st.Values()).LearnedThinking(),
		refusals:    st.ReasoningRefusals(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.BestEffort,
			Images:           llmprovider.BestEffort,
		},
	}
	var override Route
	for _, v := range st.Values() {
		switch v := v.(type) {
		case routeOption:
			override = Route(v)
		case catalog.Metadata:
			// The last given wins, as catalog reads it (0093-MADR D2).
			p.meta = v
		}
	}
	if p.route, err = resolveRoute(gateway, p.model, override); err != nil {
		return nil, err
	}
	p.routePinned = override != ""
	if b := st.BaseURL(); b != "" {
		p.baseURL = strings.TrimRight(b, "/")
	}
	p.limit = listlimit.Of(st.Values())
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(p.session), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return p.gateway }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// call is one request's resolved values.
type call struct {
	model     string
	route     Route
	input     []llmprovider.Item
	maxTokens int
	reasoning *llmprovider.Reasoning // nil for none
	req       *llmprovider.Request
	// facts is what the caches know of the model, read once per request
	// (0041-MADR D1).
	facts catalog.ModelInfo
	// limits is what the caches state of the model's limits, read once per
	// request, and limit the messages route's output limit across its
	// sends (0089-MADR D6); nil on another route.
	limits catalog.Limits
	limit  *wire.LimitRetry
	// dropped is a warning for each image the route did not send
	// (0061-MADR D5).
	dropped []llmprovider.Warning
	// tools are the tools the route's body sends, in order, and held what the
	// body holds of the caller's tools: both computed once per request, for
	// the encoder and for naming a refused tool (0084-MADR D4).
	tools []llmprovider.Tool
	held  wire.Held
}

// Generate sends req to the gateway on the model's route. When the SDK
// chose its output limit or left it out, it is read over the route's stream
// and returned whole (0088-MADR ST88).
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	c := p.resolve(ctx, req)
	if streams(c.route) && wire.OverStream(req, p.maxGiven) {
		return wire.Collect(string(p.gateway), p.Stream(ctx, req))
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2); a refusal of the effort sent is learned, and the
	// request built again (0062-MADR D5).
	effort := p.learner(c)
	// The call reads c as it is when sent, its limit set below.
	build := func(llmprovider.Token) wire.Call { return p.routeCall(ctx, c, false, effort) }
	if c.route != RouteMessages {
		// A refusal of a tool's schema names the tool it refused, when the
		// body held it (0084-MADR D2, D4).
		c.limit = p.limitRetry(c, false)
		resp, err := wire.Send(ctx, "opencode", p.src, build, p.decode(c), wire.Retries(c.limit.Check, effort.Check))
		// The limit sent, and a cap learned, are reported: on the response, or on
		// the error that ends the call (0089-MADR D5, D7, DL78).
		err = c.limit.Finish(resp, err, p.observe(c, false))
		return resp, wire.NameRefusedTool(err, c.held)
	}
	f := p.forcing.For(string(p.gateway), c.model)
	// A required choice a model refuses fails here unless the caller opted in
	// to degrading it (0034-MADR D10; R11).
	if err := messages.CheckForcing(req, f); err != nil {
		return nil, err
	}
	// On the messages route, a refusal of a forced tool choice from a model
	// no layer knows is learned, and handled by policy (0034-MADR D3); a
	// refusal learned, and a forced choice sent as auto, are reported (D7).
	retry := messages.NewRefusalRetry(f, req.ToolChoice)
	// A refusal of the output limit that states a cap resends once at it,
	// outside that bound (0089-MADR D4 to D6).
	c.limit = p.limitRetry(c, false)
	resp, err := wire.Send(ctx, "opencode", p.src, build, messages.Decode,
		wire.Retries(c.limit.Check, retry.Check, effort.Check))
	err = wire.NameRefusedTool(err, c.held)
	// The limit sent, and a cap learned, are reported: on the response, or on
	// the error that ends the call (0089-MADR D5, D7, DL78).
	err = c.limit.Finish(resp, err, p.observe(c, false))
	if err == nil {
		messages.StructuredCall(req, f, resp)
		for _, warn := range []func() (llmprovider.Warning, bool){retry.Warning,
			func() (llmprovider.Warning, bool) { return messages.ToolChoiceWarning(req, c.reasoning != nil, f) }} {
			if w, ok := warn(); ok {
				resp.Warnings = append(resp.Warnings, w)
			}
		}
	}
	return resp, err
}

// Stream streams req from the gateway, on every route (0031-MADR D1, Q2, E1):
// the request asks for an event stream, and each event is yielded as it
// arrives; a 401 before the first event renews the token and sends once
// more. A route this package does not know yields Generate's result as
// events, as llmprovider.Stream does for a provider without a Streamer.
func (p *provider) Stream(ctx context.Context, req *llmprovider.Request) iter.Seq2[llmprovider.Event, error] {
	if err := p.caps.Check(req); err != nil {
		return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
	}
	c := p.resolve(ctx, req)
	if c.route == RouteMessages {
		f := p.forcing.For(string(p.gateway), c.model)
		if err := messages.CheckForcing(req, f); err != nil {
			return func(yield func(llmprovider.Event, error) bool) { yield(llmprovider.Event{}, err) }
		}
		// A refusal of a forced tool choice is handled as Generate handles
		// it (0034-MADR D3). Each range is its own request, with its own
		// retry checks.
		return func(yield func(llmprovider.Event, error) bool) {
			retry, effort := messages.NewRefusalRetry(f, req.ToolChoice), p.learner(c)
			c := c
			c.limit = p.limitRetry(c, true)
			build := func(llmprovider.Token) wire.Call { return p.routeCall(ctx, c, true, effort) }
			stream := wire.StreamEventsRetry(ctx, string(p.gateway)+"/"+string(c.route), p.src, build, messages.Events,
				wire.Retries(c.limit.Check, retry.Check, effort.Check))
			stream = wire.NameRefusedStream(stream, c.held)
			stream = messages.StructuredStream(req, f, stream)
			stream = c.limit.FinishStream(stream, p.observe(c, true))
			for ev, err := range messages.WarnOnDone(stream, retry.Warning, func() (llmprovider.Warning, bool) {
				return messages.ToolChoiceWarning(req, c.reasoning != nil, f)
			}) {
				if !yield(ev, err) {
					return
				}
			}
		}
	}
	var events func(string, io.Reader, func(llmprovider.Event) bool) error
	switch c.route {
	case RouteResponses:
		events = p.responsesDecoder(c).Events
	case RouteChatCompletions:
		events = chatcompletions.Events
	case RouteGoogle:
		events = generatecontent.Events
	default:
		return llmprovider.Stream(ctx, generateOnly{p}, req)
	}
	return func(yield func(llmprovider.Event, error) bool) {
		// Each range is its own request, with its own effort learner.
		effort := p.learner(c)
		c := c
		c.limit = p.limitRetry(c, true)
		stream := wire.StreamEventsRetry(ctx, string(p.gateway)+"/"+string(c.route), p.src,
			func(llmprovider.Token) wire.Call { return p.routeCall(ctx, c, true, effort) }, events,
			wire.Retries(c.limit.Check, effort.Check))
		stream = c.limit.FinishStream(stream, p.observe(c, true))
		for ev, err := range wire.NameRefusedStream(stream, c.held) {
			if !yield(ev, err) {
				return
			}
		}
	}
}

// streams reports whether Stream reads route's own event stream, rather
// than Generate's result.
func streams(route Route) bool {
	return route == RouteMessages || route == RouteResponses || route == RouteChatCompletions || route == RouteGoogle
}

// generateOnly is a provider without its Stream, so that llmprovider.Stream
// falls back to Generate.
type generateOnly struct{ p *provider }

func (g generateOnly) ID() llmprovider.ProviderID { return g.p.ID() }

func (g generateOnly) Capabilities() llmprovider.Capabilities { return g.p.Capabilities() }

func (g generateOnly) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	return g.p.Generate(ctx, req)
}

// resolve is req's call: its model, input, limits, reasoning and route.
func (p *provider) resolve(ctx context.Context, req *llmprovider.Request) call {
	c := call{model: req.Model, input: req.Input, maxTokens: p.maxTokens, reasoning: p.effectiveReasoning(req), req: req}
	if c.model == "" {
		c.model = p.model
	}
	// The gateways are not listing layers, so the facts need no token, and
	// are never waited for (0041-MADR D1).
	c.facts, _ = catalog.CachedFacts(ctx, p.gateway, c.model, p.listing...)
	c.route = p.requestRoute(ctx, c.model)
	if c.route == RouteMessages || (p.maxGiven && req.MaxOutputTokens <= 0) {
		// The model's limits: the messages route's output limit, and on
		// another route the stated limit a given option is cut to
		// (0089-MADR D6).
		c.limits = catalog.CachedLimits(ctx, p.gateway, c.model, p.listing...)
	}
	if c.route == RouteGoogle {
		// No generateContent form for an image by URL is confirmed, so the
		// google route drops one, warned (0061-MADR D5, E7).
		c.input, c.dropped = wire.DropImageURLs(c.model, c.input, "the google route takes an image's bytes only, not its URL")
	}
	if req.Instructions != "" {
		c.input = append([]llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: req.Instructions}}, c.input...)
	}
	// The tools the route's body holds, once (0084-MADR D4): its tools list,
	// and on the Responses route a native deferral's definitions inside the
	// input's tool_search_output items.
	form := c.deferForm()
	c.tools = deferral.Tools(req, form)
	c.held = wire.Held{Listed: deferral.Listed(c.tools, form)}
	if c.route == RouteResponses && form == deferral.ResponsesNative {
		c.held.Inside = responses.SearchLoaded(c.input, req)
	}
	return c
}

// learner is c's thinking shape and effort, learned from the model's
// refusals of them by its route's wire (0062-MADR D2 to D6; the google
// route's, its amendment "Phase 3's google-route levels, and 0013's B9").
func (p *provider) learner(c call) *wire.Learner {
	var match wire.Matcher
	switch c.route {
	case RouteMessages:
		match = messages.MatchRefusal
	case RouteResponses:
		match = responses.MatchRefusal
	case RouteGoogle:
		match = generatecontent.MatchRefusal
	default:
		match = chatcompletions.MatchRefusal
	}
	return wire.NewLearner(p.thinking, string(p.gateway), p.baseURL, c.model, p.refusals, match)
}

// decode is the decoder of c's route. The chat and responses routes' failures
// are classified by OpenCode's own table (0026-MADR F5, F28).
func (p *provider) decode(c call) func(io.Reader) (*llmprovider.Response, error) {
	label := string(p.gateway) + "/" + string(c.route)
	switch c.route {
	case RouteResponses:
		return p.responsesDecoder(c).DecodeFor(label)
	case RouteMessages:
		return messages.Decode
	case RouteGoogle:
		return generatecontent.Decode
	default:
		return chatcompletions.DecodeFor(label)
	}
}

// routeCall is the request for c on its route, at the effort effort chooses,
// asking for an event stream when stream is set.
func (p *provider) routeCall(ctx context.Context, c call, stream bool, effort *wire.Learner) wire.Call {
	var body map[string]any
	var warnings []llmprovider.Warning
	// The output limit on an optional route: none unless the caller gives one
	// (0089-MADR D6, DL47); the messages route chooses its own below.
	var optional llmprovider.SentLimit
	if c.route != RouteMessages {
		optional = c.limit.Optional(c.req.MaxOutputTokens, p.maxTokens, p.maxGiven, func() (int, string) {
			return statedOutput(c.limits, stream)
		})
		c.maxTokens = optional.Tokens
	}
	switch c.route {
	case RouteResponses:
		body, warnings = p.responsesBody(c, effort)
		wire.DropLimit(body, optional, "max_output_tokens")
	case RouteMessages:
		chosen, refused := c.limit.Choose(p.limitInput(c, stream))
		body, warnings = p.messagesBody(c, effort, chosen.Tokens)
		if sent, ok := body[jsonKeyMaxTokens].(int); ok {
			c.limit.Sent(sent)
		}
		if refused != nil {
			// No limit can be chosen: refused before the network (0089-MADR
			// DL51).
			call := p.wireCall(c, body, stream, warnings)
			call.Refusal = refused
			return call
		}
	case RouteGoogle:
		body, warnings = p.googleBody(c, effort)
		wire.DropLimit(body, optional, "generationConfig", "maxOutputTokens")
	default:
		body, warnings = p.chatBody(ctx, c, effort)
		wire.DropLimit(body, optional, jsonKeyMaxTokens)
	}
	switch {
	case !stream, c.route == RouteGoogle:
		// generateContent streams at its own path, not by a body field
		// (0031-PLAN-native-streaming-on-the-other-wires, Phase 1).
	case c.route == RouteChatCompletions:
		body["stream"] = true
		// Usage comes in a stream only when asked for (0031-MADR E5).
		body["stream_options"] = map[string]any{"include_usage": true}
	default:
		body["stream"] = true
	}
	call := p.wireCall(c, body, stream, warnings)
	if stream && c.route == RouteGoogle {
		call.URL = p.baseURL + fmt.Sprintf("/models/%s:streamGenerateContent?alt=sse", c.model)
	}
	return call
}

// wireCall is the request for c with body, on c's route, asking for an event
// stream when stream is set, with the effort's warnings.
func (p *provider) wireCall(c call, body map[string]any, stream bool, warnings []llmprovider.Warning) wire.Call {
	return wire.Call{
		// Errors name the gateway and the route (0020-MADR F9).
		Provider: string(p.gateway) + "/" + string(c.route),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + c.route.path(c.model),
		Body:     body,
		Stream:   stream,
		// 0040-MADR D7; 0062-MADR D3, D6; 0061-MADR D5.
		Warnings: slices.Concat(deferral.Warnings(c.deferForm(), c.req, c.model), warnings, c.dropped),
		// A model whose facts take no image is refused before the network
		// (0061-MADR D5).
		Refusal: wire.RefuseImages(c.model, c.req.Input, c.facts.Has(catalog.FieldInput),
			slices.Contains(c.facts.Input, catalog.ModalityImage)),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			// Each route reads the key from its vendor's header (MADR 0007
			// §1c); the key stays in a header, never the URL.
			header, scheme := keyHeader(c.route)
			token.Apply(r, header, scheme)
			// x-opencode-session is fixed for the provider's lifetime (MADR
			// 0012 §1.4).
			r.Header.Set(sessionHeader, p.session)
			if stream {
				r.Header.Set("Accept", "text/event-stream")
			}
		},
	}
}

// effectiveReasoning is the request's Reasoning, else WithReasoning's, with
// an effort or budget the request leaves empty taken from WithReasoning; nil
// for none.
func (p *provider) effectiveReasoning(req *llmprovider.Request) *llmprovider.Reasoning {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return nil
	}
	out := *r
	if p.reasoning != nil {
		if out.Effort == "" {
			out.Effort = p.reasoning.Effort
		}
		if out.Budget == 0 {
			out.Budget = p.reasoning.Budget
		}
	}
	return &out
}

// keyHeader is the header and scheme the Zen/Go server reads the key from on
// route r. Each route parses only its vendor's header (MADR 0007 §1c).
func keyHeader(r Route) (header, scheme string) {
	switch r {
	case RouteMessages:
		return "x-api-key", ""
	case RouteGoogle:
		return "x-goog-api-key", ""
	default:
		return headerAuthorization, "Bearer"
	}
}

// requestRoute is one request's wire format: the pinned route, else the
// model's provider.npm route in the document given to New, else the table's
// or the heuristic's (MADR 0012 §3.1), at once: no document is fetched or
// waited for (0093-MADR D6).
func (p *provider) requestRoute(_ context.Context, model string) Route {
	if p.routePinned {
		return p.route
	}
	if npm, ok := p.meta.NPM(p.gateway, model); ok {
		return routeForNPM(npm)
	}
	if model == p.model {
		return p.route
	}
	return tableRoute(p.gateway, model)
}

// deferForm is how c's deferred tools go out on its route: natively on the
// messages route from Claude 4.5 and on the responses route from GPT-5.4,
// and appended as loaded otherwise (0040-MADR D1, D6).
func (c call) deferForm() deferral.Form {
	switch c.route {
	case RouteMessages:
		return deferral.Of(deferral.Messages, c.model)
	case RouteResponses:
		return deferral.ForRequest(deferral.Responses, c.model, c.req)
	default:
		return deferral.ForRequest(deferral.Other, c.model, c.req)
	}
}

// responsesDecoder is the decoder c's reply needs: one that names the
// Search tool when c's deferred tools go out natively (0040-MADR D3), and
// writes to p's logger (0092-MADR ST20).
func (p *provider) responsesDecoder(c call) responses.Decoder {
	if c.deferForm() == deferral.ResponsesNative {
		return responses.Decoder{Search: deferral.SearchName(c.req.Tools), Logger: p.logger}
	}
	return responses.Decoder{Logger: p.logger}
}

// responsesBody is the OpenAI Responses shape, at the effort learner chooses,
// with its warnings.
func (p *provider) responsesBody(c call, learner *wire.Learner) (map[string]any, []llmprovider.Warning) {
	// Deferred tools go out natively from GPT-5.4 on, and are appended as
	// loaded before it (0040-MADR D1, D3, D6).
	form := c.deferForm()
	input := responses.Input(c.input)
	if form == deferral.ResponsesNative {
		input = responses.InputLoading(c.input, c.req)
	}
	body := map[string]any{
		"model":             c.model,
		"input":             input,
		"max_output_tokens": c.maxTokens,
		// OpenCode's client stores nothing for @ai-sdk/openai models
		// (transform.ts:1235-1243, MADR 0012 §3.2); items are replayed, so
		// reasoning comes back encrypted (0021-MADR D4).
		"store":   false,
		"include": []string{"reasoning.encrypted_content"},
	}
	if form == deferral.ResponsesNative {
		wire.AddResponsesToolsNative(body, c.tools, c.req.ToolChoice)
	} else {
		wire.AddResponsesTools(body, c.tools, c.req.ToolChoice)
	}
	var warnings []llmprovider.Warning
	if c.reasoning != nil {
		want := c.reasoning.Effort
		if want == "" {
			want = llmprovider.EffortMedium
		}
		// By D3's ladder; none left leaves out reasoning (0062-MADR D3).
		var effort llmprovider.Effort
		effort, warnings = learner.Effort(want, c.facts.Reasoning.Efforts, c.facts.Has(catalog.FieldReasoningEfforts))
		if effort != "" {
			// The summary makes reasoning items carry text, as for OpenAI
			// (0020-MADR F24; 0021-MADR D4).
			body["reasoning"] = map[string]any{"effort": effort, "summary": "auto"}
		}
	}
	wire.AddPromptCacheKey(body, c.req.Cache, p.session) // 0039-MADR D3, D4
	return body, warnings
}

// messagesBody is the Anthropic Messages shape, at the effort learner
// chooses, with its warnings, and maxTokens, the output limit chosen
// (0089-MADR D6), which a thinking budget may raise (DL55).
func (p *provider) messagesBody(c call, learner *wire.Learner, maxTokens int) (map[string]any, []llmprovider.Warning) {
	// Deferred tools go out in the model's form: a Claude model's natively,
	// any other model's appended as loaded (0040-MADR D1, D2, D6).
	form := c.deferForm()
	msgs := messages.FromItems(c.input)
	if form == deferral.MessagesNative {
		msgs = messages.FromItemsLoading(c.input)
	}
	body := map[string]any{
		"model":    c.model,
		"messages": msgs,
	}
	if system := wire.SystemPrompt(c.input); system != "" {
		body["system"] = system
	}
	thinking := c.reasoning != nil
	var warnings []llmprovider.Warning
	if thinking {
		var effort llmprovider.Effort
		effort, warnings = learner.Effort(c.reasoning.Effort, c.facts.Reasoning.Efforts,
			c.facts.Has(catalog.FieldReasoningEfforts))
		facts := messagesThinking(c.facts)
		shape := learner.Shape(messages.PreferredShape(facts, c.reasoning.Budget))
		maxTokens = messages.AddThinkingFor(body, string(effort), c.reasoning.Budget, maxTokens, facts, shape)
	}
	body[jsonKeyMaxTokens] = maxTokens
	if tools := c.tools; len(tools) > 0 {
		if form == deferral.MessagesNative {
			body[jsonKeyTools] = wire.MessagesToolsNative(tools)
		} else {
			body[jsonKeyTools] = wire.MessagesTools(tools)
		}
		messages.AddToolChoice(body, c.req.ToolChoice, thinking, p.forcing.For(string(p.gateway), c.model))
	}
	// Last, once the system and tools are set (0039-MADR D2).
	messages.AddCache(body, c.req.Cache)
	return body, warnings
}

// effortsOf is info's reasoning efforts, as strings.
func effortsOf(info catalog.ModelInfo) []string {
	out := make([]string, len(info.Reasoning.Efforts))
	for i, e := range info.Reasoning.Efforts {
		out[i] = string(e)
	}
	return out
}

// messagesThinking is the Messages thinking info's facts give (0041-MADR
// D2, R3), with its output limit.
func messagesThinking(info catalog.ModelInfo) messages.Thinking {
	limit := 0
	if info.Has(catalog.FieldMaxOutputTokens) {
		limit = info.MaxOutputTokens
	}
	return messages.FactsThinking(info.Reasoning.Toggle, effortsOf(info), info.Has(catalog.FieldReasoningBudget),
		info.Reasoning.BudgetMin, info.Reasoning.BudgetMax, limit)
}

// geminiThinking is the Gemini thinking info's facts give (0041-MADR
// D2).
func geminiThinking(info catalog.ModelInfo) generatecontent.Thinking {
	return generatecontent.FactsThinking(effortsOf(info), info.Has(catalog.FieldReasoningBudget))
}

// googleBody is the Gemini generateContent shape, at the effort learner
// chooses, with its warnings.
func (p *provider) googleBody(c call, learner *wire.Learner) (map[string]any, []llmprovider.Warning) {
	genCfg := map[string]any{"maxOutputTokens": c.maxTokens}
	var warnings []llmprovider.Warning
	if c.reasoning != nil {
		var effort llmprovider.Effort
		effort, warnings = learner.Effort(c.reasoning.Effort, c.facts.Reasoning.Efforts,
			c.facts.Has(catalog.FieldReasoningEfforts))
		shape := learner.Shape(generatecontent.PreferredShape(geminiThinking(c.facts), c.reasoning.Budget))
		genCfg["thinkingConfig"] = generatecontent.ThinkingConfigFor(string(effort), c.reasoning.Budget, shape)
	}
	body := map[string]any{
		"contents":         generatecontent.Contents(c.input),
		"generationConfig": genCfg,
	}
	if system := generatecontent.SystemInstruction(c.input); system != nil {
		body["systemInstruction"] = system
	}
	if tools := c.tools; len(tools) > 0 {
		decls := make([]map[string]any, len(tools))
		for i, tool := range tools {
			decls[i] = map[string]any{jsonKeyName: tool.Name, jsonKeyDescription: tool.Description, jsonKeyParameters: wire.ToolSchema(tool.Schema)}
		}
		body[jsonKeyTools] = []map[string]any{{"functionDeclarations": decls}}
		if cfg := googleToolConfig(c.req.ToolChoice); cfg != nil {
			body["toolConfig"] = map[string]any{"functionCallingConfig": cfg}
		}
	}
	return body, warnings
}

// googleToolConfig is generateContent's functionCallingConfig for choice, or
// nil for auto.
func googleToolConfig(choice llmprovider.ToolChoice) map[string]any {
	if name, named := choice.Tool(); named {
		return map[string]any{jsonKeyMode: "ANY", "allowedFunctionNames": []string{name}}
	}
	switch choice {
	case llmprovider.ToolChoiceRequired:
		return map[string]any{jsonKeyMode: "ANY"}
	case llmprovider.ToolChoiceNone:
		return map[string]any{jsonKeyMode: "NONE"}
	default:
		return nil
	}
}

// chatBody is the Chat Completions shape, with chatEffort's reasoning_effort,
// replaying prior reasoning under the field the model's metadata declares.
func (p *provider) chatBody(ctx context.Context, c call, learner *wire.Learner) (map[string]any, []llmprovider.Warning) {
	effort, warnings := p.chatEffort(c, learner)
	return chatcompletions.Body(c.model, c.maxTokens, c.input, chatcompletions.Opts{
		ReasoningEffort:      string(effort),
		ReplayReasoningField: p.chatReplayField(ctx, c),
		Tools:                c.tools,
		SendStrict:           true,
		ToolChoice:           c.req.ToolChoice,
	}), warnings
}

// chatEffort is the chat route's reasoning_effort, when the call reasons with
// an effort: by D3's ladder over what the model refused and the efforts its
// published reasoning_options list in the document given to New. No
// document, or one silent on the model, lists none, and the effort asked is
// sent, and a refusal learned (0062-MADR D8, Q7 (a); it sent nothing, MADR
// 0009 §6; 0093-MADR D2).
func (p *provider) chatEffort(c call, learner *wire.Learner) (llmprovider.Effort, []llmprovider.Warning) {
	if c.reasoning == nil || c.reasoning.Effort == "" {
		return "", nil
	}
	var listed []llmprovider.Effort
	for _, e := range p.meta.ReasoningEfforts(p.gateway, c.model) {
		listed = append(listed, llmprovider.Effort(e))
	}
	return learner.Effort(c.reasoning.Effort, listed, len(listed) > 0)
}

// chatReplayField is the interleaved reasoning field the model's metadata
// declares in the document given to New, read only when the input holds an
// assistant turn to replay onto (MADR 0012 §2, O5). With no document, it
// replays nothing.
func (p *provider) chatReplayField(_ context.Context, c call) string {
	if !slices.ContainsFunc(c.input, isAssistantTurn) {
		return ""
	}
	return p.meta.InterleavedField(p.gateway, c.model)
}

// isAssistantTurn reports whether an item is part of an assistant turn.
func isAssistantTurn(item llmprovider.Item) bool {
	switch v := item.(type) {
	case llmprovider.FunctionCallItem:
		return true
	case llmprovider.MessageItem:
		return v.Role == llmprovider.RoleAssistant
	}
	return false
}

// ListModels returns the curated gateway listing, falling back to the static
// catalog. It spends no generation on probes (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, p.gateway, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = listlimit.Cut(catalog.Static(p.gateway), p.limit)
	}
	return listed, nil
}

// memoKey is c's key in the memo of output caps (0089-MADR D5).
func (p *provider) memoKey(c call, stream bool) limitmemo.Key {
	base := p.baseURL
	if strings.EqualFold(base, p.defaultBase) {
		base = ""
	}
	return limitmemo.Key{Provider: string(p.gateway), BaseURL: base, Model: c.model, Stream: stream}
}

// routeLimits is what route reads of a refusal of its output limit
// (0089-MADR DL68).
func routeLimits(route Route) wire.LimitWire {
	switch route {
	case RouteMessages:
		return wire.MessagesLimits
	case RouteResponses:
		return wire.ResponsesLimits
	case RouteGoogle:
		return wire.GenerateContentLimits
	}
	return wire.ChatCompletionsLimits
}

// statedOutput is the output limits states for a streamed or a plain
// request, and its evidence's source; 0 when no source states it.
func statedOutput(limits catalog.Limits, stream bool) (int, string) {
	out := limits.Output
	if stream {
		out = limits.StreamedOutput
	}
	if !out.Known {
		return 0, ""
	}
	return out.Tokens, out.Evidence[0].Source
}

// limitRetry is c's output limit across its sends, on c's route.
func (p *provider) limitRetry(c call, stream bool) *wire.LimitRetry {
	given := p.baseURL
	if strings.EqualFold(given, p.defaultBase) {
		given = ""
	}
	r := &wire.LimitRetry{Wire: routeLimits(c.route), Model: c.model,
		Observed: llmprovider.LimitObservation{Model: c.model, BaseURL: given}}
	if p.learn {
		memo, key := p.limits, p.memoKey(c, stream)
		r.Keep = func(tokens int) bool { return memo.Keep(key, tokens) }
	}
	return r
}

// observe is c's memo's drop of a cap a response shows wrong (DL39).
func (p *provider) observe(c call, stream bool) func(int) {
	memo, key := p.limits, p.memoKey(c, stream)
	return func(output int) { memo.Observe(key, output) }
}

// limitInput is what c's output limit is chosen from on the messages route,
// which requires one: the request's own value, the option, the model's
// chosen output fact, and the route's fallback (0089-MADR D6, DL51), held to
// the room the window, else the input limit, leaves the request's estimated
// input (0088-MADR ST87, ST105).
func (p *provider) limitInput(c call, stream bool) wire.LimitInput {
	in := wire.LimitInput{Request: c.req.MaxOutputTokens, Option: p.maxTokens, Given: p.maxGiven, Sends: true,
		Required: true, Provider: string(p.gateway), Model: c.model,
		Window: cmp.Or(c.limits.Window.Tokens, c.limits.Input.Tokens), Estimate: wire.EstimateInput(c.req)}
	in.Output, in.Source = statedOutput(c.limits, stream)
	// The route's fallback is this request's own read's (0093-MADR D7).
	if f := c.limits.RouteFallback; f.Known {
		in.Fallback, in.FallbackSource = f.Tokens, f.Evidence[0].Source
	}
	return in
}
