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
// model metadata (llmprovider.WithModelMetadataURL), else the built-in table,
// else a prefix heuristic (MADR 0012 §3.1). Sending a model to the wrong route
// fails with an opaque HTTP 500, which classifies as the retryable
// ErrProviderUnavailable; WithRoute pins a route.
//
// Capabilities: tools and forced tool choice are Supported. Reasoning is
// BestEffort: the chat route sends reasoning_effort only when the model's
// published reasoning_options list the effort (MADR 0009 §6). Continuation is
// Unsupported: the gateway rejects previous_response_id with HTTP 400
// (measured 2026-08-28), so a caller replays prior items. There is no native
// streaming; llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - Reasoning: the responses route takes an effort (medium when none is
//     named); the messages and google routes take the effort or the budget,
//     shaped by model as Claude's and Gemini's; the chat route, see above.
//   - On the messages route, a forced tool choice is sent as "auto" with
//     reasoning: Anthropic refuses to force a tool while thinking.
//   - Instructions are sent as a leading system item on every route: the
//     system field on the messages route, systemInstruction on the google
//     route, a system message elsewhere.
//   - ToolChoiceRequired and ToolChoiceNone use each route's documented form;
//     only a named tool was measured against the gateways.
//
// ListModels returns the curated gateway listing. It never probes: the
// gateways meter every call (MADR 0012 §1.6).
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
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
	gateway     llmprovider.ProviderID
	src         llmprovider.TokenSource
	model       string
	baseURL     string
	client      *http.Client
	caps        llmprovider.Capabilities
	maxTokens   int
	reasoning   *llmprovider.Reasoning // WithReasoning's default, or nil
	route       Route                  // WithRoute's, or the table's for model
	routePinned bool
	metadataURL string
	userAgent   string
	session     string
	logger      *slog.Logger
	// listing carries the caller's options, the session and the transport to
	// the listing, which lives in llmprovider until 0015-PLAN S8b.
	listing []llmprovider.Option
}

// NewZen builds the OpenCode Zen provider.
func NewZen(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return newGateway(llmprovider.ProviderOpencodeZen, zenBaseURL, opts)
}

// NewGo builds the OpenCode Go provider.
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
	case *llmprovider.OAuthSession, *llmprovider.VendorCLISession:
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
		reasoning:   st.Reasoning(),
		metadataURL: st.ModelMetadataURL(),
		userAgent:   st.UserAgent(),
		session:     st.SessionID(),
		logger:      st.Logger(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.Supported,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Unsupported,
		},
	}
	var override Route
	for _, v := range st.Values() {
		if r, ok := v.(routeOption); ok {
			override = Route(r)
		}
	}
	if p.route, err = resolveRoute(gateway, p.model, override); err != nil {
		return nil, err
	}
	p.routePinned = override != ""
	if b := st.BaseURL(); b != "" {
		p.baseURL = strings.TrimRight(b, "/")
	}
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
}

// Generate sends req to the gateway on the model's route.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	c := call{model: req.Model, input: req.Input, maxTokens: p.maxTokens, reasoning: p.effectiveReasoning(req), req: req}
	if c.model == "" {
		c.model = p.model
	}
	if req.MaxOutputTokens > 0 {
		c.maxTokens = req.MaxOutputTokens
	}
	if req.Instructions != "" {
		c.input = append([]llmprovider.Item{llmprovider.MessageItem{Role: string(llmprovider.RoleSystem), Text: req.Instructions}}, c.input...)
	}
	c.route = p.requestRoute(ctx, c.model)
	var body map[string]any
	switch c.route {
	case RouteResponses:
		body = p.responsesBody(c)
	case RouteMessages:
		body = p.messagesBody(c)
	case RouteGoogle:
		body = p.googleBody(c)
	default:
		body = p.chatBody(ctx, c)
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("llmprovider: opencode: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+c.route.path(c.model), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	token, err := p.src.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmprovider: opencode: acquire token: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", p.userAgent)
	// Each route reads the key from its vendor's header (MADR 0007 §1c); the
	// key stays in a header, never the URL.
	header, scheme := keyHeader(c.route)
	llmprovider.SetTokenHeader(httpReq, token, header, scheme)
	// x-opencode-session is fixed for the provider's lifetime (MADR 0012 §1.4).
	httpReq.Header.Set(sessionHeader, p.session)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.logger.Debug("llmprovider: opencode: close response body", "error", err)
		}
	}()
	if err := llmprovider.ClassifyHTTPError(string(p.gateway)+"/"+string(c.route), resp); err != nil {
		return nil, err
	}
	// 1 MiB bounds a runaway reply.
	limited := io.LimitReader(resp.Body, 1<<20)
	switch c.route {
	case RouteResponses:
		return llmprovider.DecodeResponsesAPIOutput(limited)
	case RouteMessages:
		return llmprovider.DecodeMessagesResponse(limited)
	case RouteGoogle:
		return llmprovider.DecodeGeminiResponse(limited)
	default:
		return llmprovider.DecodeChatCompletionsResponse(limited)
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
// model's provider.npm route in the metadata, else the table's or the
// heuristic's (MADR 0012 §3.1). The lookup waits at most 5 s.
func (p *provider) requestRoute(ctx context.Context, model string) Route {
	if p.routePinned {
		return p.route
	}
	if meta, err := llmprovider.LookupModelMetadata(ctx, p.metadataURL, p.client); err == nil {
		if npm, ok := meta.NPM(string(p.gateway), model); ok {
			return routeForNPM(npm)
		}
	}
	if model == p.model {
		return p.route
	}
	return tableRoute(p.gateway, model)
}

// responsesBody is the OpenAI Responses shape.
func (p *provider) responsesBody(c call) map[string]any {
	body := map[string]any{
		"model":             c.model,
		"input":             llmprovider.ItemsToInput(c.input),
		"max_output_tokens": c.maxTokens,
		// OpenCode's client stores nothing for @ai-sdk/openai models
		// (transform.ts:1235-1243, MADR 0012 §3.2); items are replayed.
		"store": false,
	}
	if tools := c.req.Tools; len(tools) > 0 {
		list := make([]map[string]any, len(tools))
		for i, tool := range tools {
			list[i] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyName: tool.Name,
				jsonKeyDescription: tool.Description, jsonKeyParameters: tool.Schema}
		}
		body[jsonKeyTools] = list
		if name, forced := c.req.ToolChoice.Tool(); forced {
			body[jsonKeyToolChoice] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyName: name}
		} else if c.req.ToolChoice == llmprovider.ToolChoiceRequired || c.req.ToolChoice == llmprovider.ToolChoiceNone {
			body[jsonKeyToolChoice] = string(c.req.ToolChoice)
		}
	}
	if c.reasoning != nil {
		effort := c.reasoning.Effort
		if effort == "" {
			effort = llmprovider.EffortMedium
		}
		body["reasoning"] = map[string]any{"effort": effort}
	}
	return body
}

// messagesBody is the Anthropic Messages shape.
func (p *provider) messagesBody(c call) map[string]any {
	maxTokens := c.maxTokens
	body := map[string]any{
		"model":    c.model,
		"messages": llmprovider.MessagesFromItems(c.input),
	}
	if system := llmprovider.SystemPrompt(c.input); system != "" {
		body["system"] = system
	}
	thinking := c.reasoning != nil
	if thinking {
		maxTokens = llmprovider.AddMessagesThinking(body, c.model, string(c.reasoning.Effort), c.reasoning.Budget, maxTokens)
	}
	body[jsonKeyMaxTokens] = maxTokens
	if tools := c.req.Tools; len(tools) > 0 {
		list := make([]map[string]any, len(tools))
		for i, tool := range tools {
			list[i] = map[string]any{jsonKeyName: tool.Name, jsonKeyDescription: tool.Description, "input_schema": tool.Schema}
		}
		body[jsonKeyTools] = list
		if choice := messagesToolChoice(c.req.ToolChoice, thinking); choice != nil {
			body[jsonKeyToolChoice] = choice
		}
	}
	return body
}

// messagesToolChoice is the Messages tool_choice for choice. With thinking,
// forcing is sent as "auto": extended thinking is incompatible with a forced
// tool_choice.
func messagesToolChoice(choice llmprovider.ToolChoice, thinking bool) map[string]any {
	name, named := choice.Tool()
	forced := named || choice == llmprovider.ToolChoiceRequired
	switch {
	case forced && thinking:
		return map[string]any{jsonKeyType: "auto"}
	case named:
		return map[string]any{jsonKeyType: "tool", jsonKeyName: name}
	case choice == llmprovider.ToolChoiceRequired:
		return map[string]any{jsonKeyType: "any"}
	case choice == llmprovider.ToolChoiceNone:
		return map[string]any{jsonKeyType: "none"}
	default:
		return nil
	}
}

// googleBody is the Gemini generateContent shape.
func (p *provider) googleBody(c call) map[string]any {
	genCfg := map[string]any{"maxOutputTokens": c.maxTokens}
	if c.reasoning != nil {
		genCfg["thinkingConfig"] = llmprovider.GeminiThinkingConfig(c.model, string(c.reasoning.Effort), c.reasoning.Budget)
	}
	body := map[string]any{
		"contents":         llmprovider.GeminiItemsToContents(c.input),
		"generationConfig": genCfg,
	}
	if system := llmprovider.GeminiSystemInstruction(c.input); system != nil {
		body["systemInstruction"] = system
	}
	if tools := c.req.Tools; len(tools) > 0 {
		decls := make([]map[string]any, len(tools))
		for i, tool := range tools {
			decls[i] = map[string]any{jsonKeyName: tool.Name, jsonKeyDescription: tool.Description, jsonKeyParameters: tool.Schema}
		}
		body[jsonKeyTools] = []map[string]any{{"functionDeclarations": decls}}
		if cfg := googleToolConfig(c.req.ToolChoice); cfg != nil {
			body["toolConfig"] = map[string]any{"functionCallingConfig": cfg}
		}
	}
	return body
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

// chatBody is the Chat Completions shape. It carries reasoning_effort only
// when chatReasoningEffort resolves one (MADR 0009 §6), and replays prior
// reasoning under the field the model's metadata declares.
func (p *provider) chatBody(ctx context.Context, c call) map[string]any {
	body := llmprovider.ChatCompletionsBody(c.model, c.maxTokens, c.input, llmprovider.ChatCompletionsOpts{
		ReasoningEffort:      p.chatReasoningEffort(ctx, c),
		ReplayReasoningField: p.chatReplayField(ctx, c),
	})
	if tools := c.req.Tools; len(tools) > 0 {
		list := make([]map[string]any, len(tools))
		for i, tool := range tools {
			list[i] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyFunction: map[string]any{
				jsonKeyName: tool.Name, jsonKeyDescription: tool.Description, jsonKeyParameters: tool.Schema}}
		}
		body[jsonKeyTools] = list
		if name, forced := c.req.ToolChoice.Tool(); forced {
			body[jsonKeyToolChoice] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyFunction: map[string]any{jsonKeyName: name}}
		} else if c.req.ToolChoice == llmprovider.ToolChoiceRequired || c.req.ToolChoice == llmprovider.ToolChoiceNone {
			body[jsonKeyToolChoice] = string(c.req.ToolChoice)
		}
	}
	return body
}

// chatReasoningEffort is the chat route's reasoning_effort: the effort, when
// the call reasons and the model's published reasoning_options list it;
// otherwise "". Metadata that is unavailable, disabled or silent on the model
// sends nothing (MADR 0009 §6).
func (p *provider) chatReasoningEffort(ctx context.Context, c call) string {
	if c.reasoning == nil || c.reasoning.Effort == "" {
		return ""
	}
	meta, err := llmprovider.LookupModelMetadata(ctx, p.metadataURL, p.client)
	if err != nil || !slices.Contains(meta.ReasoningEfforts(string(p.gateway), c.model), string(c.reasoning.Effort)) {
		return ""
	}
	return string(c.reasoning.Effort)
}

// chatReplayField is the interleaved reasoning field the model's metadata
// declares, looked up only when the input holds an assistant turn to replay
// onto (MADR 0012 §2, O5). Unavailable metadata replays nothing.
func (p *provider) chatReplayField(ctx context.Context, c call) string {
	if !slices.ContainsFunc(c.input, isAssistantTurn) {
		return ""
	}
	meta, err := llmprovider.LookupModelMetadata(ctx, p.metadataURL, p.client)
	if err != nil {
		return ""
	}
	return meta.InterleavedField(string(p.gateway), c.model)
}

// isAssistantTurn reports whether an item is part of an assistant turn.
func isAssistantTurn(item llmprovider.Item) bool {
	switch v := item.(type) {
	case llmprovider.FunctionCallItem:
		return true
	case llmprovider.MessageItem:
		return v.Role == string(llmprovider.RoleAssistant)
	}
	return false
}

// ListModels returns the curated gateway listing, falling back to the static
// catalog. It spends no generation on probes (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	listed, err := llmprovider.ListAvailableModelsWithSource(ctx, string(p.gateway), p.src, p.listing...)
	if err != nil || len(listed) == 0 {
		listed = llmprovider.StaticModels(string(p.gateway))
	}
	return listed, nil
}
