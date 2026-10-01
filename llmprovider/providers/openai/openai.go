// Package openai is the OpenAI provider (0015-MADR D2): the Responses API at
// api.openai.com with an API key, or the ChatGPT backend with a ChatGPT
// session (an OpenAI OAuth session, or the Codex CLI's login).
//
// Build it with New, or providers.New(llmprovider.ProviderOpenAI, …). The
// credential decides the backend: WithAPIKey, or WithTokenSource with a
// static token, uses the platform; a ChatGPT session uses the backend at
// chatgpt.com.
//
// Capabilities: tools, forced tool choice and reasoning are Supported.
// Continuation is Supported with an API key, and Unsupported with a ChatGPT
// session, which stores no responses (MADR 0012 §4.2). There is no native
// streaming; llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - Reasoning takes an effort. A Reasoning with only a Budget sends the
//     default effort, medium; the budget is not sent.
//   - A ChatGPT session sends no output limit: the backend rejects
//     max_output_tokens (gate G-C, 2026-09-27), so MaxOutputTokens and
//     WithMaxTokens have no effect there. It always sends store false, so
//     WithStore has no effect there either.
//
// ListModels returns the curated listing. With an API key it probes each
// listed model with one short, billed generation, unless
// llmprovider.WithModelProbes(false) says otherwise (0016-MADR A5); a ChatGPT
// session is never probed (MADR 0012 §1.6).
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
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

// storeOption is WithStore's value.
type storeOption bool

// WithStore sets whether an API key's responses are stored (MADR 0012 §6).
// Without it the service default applies, which keeps continuation working;
// callers under zero data retention pass false, after which a
// PreviousResponseID has nothing to chain from. A ChatGPT session always
// sends false.
func WithStore(store bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderOpenAI, "openai.WithStore", storeOption(store))
}

type provider struct {
	src       llmprovider.TokenSource
	chatGPT   bool
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
	// listing carries the caller's options, the session and the transport to
	// the listing, which lives in llmprovider until 0015-PLAN S8b.
	listing []llmprovider.Option
}

// New builds the OpenAI provider. It needs a credential: WithAPIKey, or
// WithTokenSource.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderOpenAI, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	if src == nil {
		return nil, fmt.Errorf("%w: openai needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	}
	p := &provider{
		src:       src,
		chatGPT:   llmprovider.IsChatGPTSession(src),
		model:     st.Model(),
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
		session:   st.SessionID(),
		probe:     st.ModelProbes(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.Supported,
			Reasoning:        llmprovider.Supported,
			Continuation:     llmprovider.Supported,
			NativeStreaming:  llmprovider.Unsupported,
		},
	}
	for _, v := range st.Values() {
		if store, ok := v.(storeOption); ok {
			s := bool(store)
			p.store = &s
		}
	}
	p.baseURL = llmprovider.DefaultOpenAIPlatformBaseURL
	if p.chatGPT {
		p.baseURL = llmprovider.DefaultOpenAIChatGPTBaseURL
		p.caps.Continuation = llmprovider.Unsupported
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	llmprovider.ShareHTTPClient(src, p.client)
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(p.session), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderOpenAI }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Responses API. A 401 from a session that can
// refresh is retried once with a new token.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	resp, err := p.generateOnce(ctx, req)
	var apiErr *llmprovider.APIError
	if err == nil || !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized || !llmprovider.ExpireSession(p.src) {
		return resp, err
	}
	return p.generateOnce(ctx, req)
}

// body is the Responses request for req.
func (p *provider) body(req *llmprovider.Request) map[string]any {
	model := req.Model
	if model == "" {
		model = p.model
	}
	body := map[string]any{
		jsonKeyModel: model,
		jsonKeyInput: responses.Input(req.Input),
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
		body["include"] = []string{"reasoning.encrypted_content"}
		body["prompt_cache_key"] = p.session
	} else {
		maxTokens := p.maxTokens
		if req.MaxOutputTokens > 0 {
			maxTokens = req.MaxOutputTokens
		}
		body["max_output_tokens"] = maxTokens
		if p.store != nil {
			body["store"] = *p.store
		}
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]any{
				jsonKeyType:        jsonKeyFunction,
				jsonKeyName:        tool.Name,
				jsonKeyDescription: tool.Description,
				jsonKeyParameters:  tool.Schema,
			}
		}
		body[jsonKeyTools] = tools
		if name, forced := req.ToolChoice.Tool(); forced {
			body[jsonKeyToolChoice] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyName: name}
		} else if req.ToolChoice == llmprovider.ToolChoiceRequired || req.ToolChoice == llmprovider.ToolChoiceNone {
			body[jsonKeyToolChoice] = string(req.ToolChoice)
		}
	}
	if effort, ok := p.effort(req); ok {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	if req.PreviousResponseID != "" {
		body["previous_response_id"] = req.PreviousResponseID
	}
	return body
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

func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	reqBody, err := json.Marshal(p.body(req))
	if err != nil {
		return nil, fmt.Errorf("llmprovider: openai: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", p.userAgent)
	token, err := p.src.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmprovider: openai: acquire token: %w", err)
	}
	token.Apply(httpReq, headerAuthorization, "Bearer")
	if p.chatGPT {
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set(headerSession, p.session)
		httpReq.Header.Set(llmprovider.ChatGPTOriginatorHeader, llmprovider.ChatGPTOriginatorValue)
		if accountID := llmprovider.ChatGPTSessionAccountID(p.src); accountID != "" {
			httpReq.Header.Set(llmprovider.ChatGPTAccountHeader, accountID)
		}
		if llmprovider.ChatGPTSessionFedRAMP(p.src) {
			httpReq.Header.Set(llmprovider.ChatGPTFedRAMPHeader, "true")
		}
		if residency := residency(token.Value); residency != "" {
			httpReq.Header.Set(headerResidency, residency)
		}
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.logger.Debug("llmprovider: openai: close response body", "error", err)
		}
	}()
	if err := llmprovider.ClassifyHTTPError(llmprovider.ProviderOpenAI, resp); err != nil {
		return nil, err
	}
	if p.chatGPT {
		return responses.ReadStream(llmprovider.ProviderOpenAI, resp.Body)
	}
	return responses.Decode(io.LimitReader(resp.Body, 1<<20))
}

// ListModels returns the curated chat models this credential can use, never
// the raw /v1/models dump. An API key's models are probed unless
// WithModelProbes(false); a ChatGPT session's are not (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	listed, err := llmprovider.ListAvailableModelsWithSource(ctx, llmprovider.ProviderOpenAI, p.src, p.listing...)
	if err != nil || len(listed) == 0 {
		// A ChatGPT session has no static catalog (MADR 0008 D11).
		if p.chatGPT {
			return nil, err
		}
		listed = llmprovider.StaticModels(llmprovider.ProviderOpenAI)
	}
	// A ChatGPT subscription meters every call: no generation probe (MADR 0012 §1.6).
	if p.chatGPT || !p.probe {
		return listed, nil
	}
	healthy := transport.ProbeGenerateHealth(ctx, listed, llmprovider.MaxListedModels, func(ctx context.Context, model string) (string, error) {
		probe := *p
		probe.model, probe.maxTokens, probe.store, probe.reasoning = model, probeMaxOutputTokens, nil, nil
		resp, err := probe.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: string(llmprovider.RoleUser), Text: probePrompt}}})
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
