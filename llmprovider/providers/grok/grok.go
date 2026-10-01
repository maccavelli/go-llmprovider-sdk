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
// Capabilities: tools, forced tool choice, reasoning and continuation are
// Supported. There is no native streaming; llmprovider.Stream emits
// Generate's result.
//
// Degradations:
//   - Reasoning takes an effort, clamped to the Grok CLI's menu for the
//     model (MADR 0012 §6): grok-4.6 offers low to xhigh, grok-4.5 low to
//     high, grok-4.6-build and grok-3-mini low and high. An effort off the
//     menu is sent as the nearest lower one, and no effort as high. Other
//     models reason on their own, and get no reasoning field. A Budget is
//     not sent.
//   - Instructions are sent as a leading system message, the shape the old
//     API already sent for a system item; the Responses instructions field
//     is not sent. Neither form has been measured against xAI.
//   - ToolChoiceRequired and ToolChoiceNone are sent as "required" and
//     "none", the Responses API's values; only a named tool was measured
//     against xAI.
//
// ListModels returns the curated listing, probing each listed model with one
// short, billed generation unless llmprovider.WithModelProbes(false) says
// otherwise (0016-MADR A5).
package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/responses"
)

const (
	headerAuthorization = "Authorization"

	jsonKeyType        = "type"
	jsonKeyName        = "name"
	jsonKeyFunction    = "function"
	jsonKeyDescription = "description"
	jsonKeyParameters  = "parameters"

	// probeMaxOutputTokens is the output limit a listing probe sends: the
	// default of the old API's probe provider, kept so the probe's request
	// is unchanged (0015-MADR D1).
	probeMaxOutputTokens = 8192
	probePrompt          = "Respond with ONLY the word Hello"
)

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
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   llmprovider.DefaultGrokBaseURL,
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		logger:    st.Logger(),
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
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	llmprovider.ShareHTTPClient(src, p.client)
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
	maxTokens := p.maxTokens
	if req.MaxOutputTokens > 0 {
		maxTokens = req.MaxOutputTokens
	}
	input := req.Input
	if req.Instructions != "" {
		input = append([]llmprovider.Item{llmprovider.MessageItem{Role: string(llmprovider.RoleSystem), Text: req.Instructions}}, input...)
	}
	body := map[string]any{
		"model":             model,
		"input":             responses.Input(input),
		"max_output_tokens": maxTokens,
	}
	if p.store != nil {
		body["store"] = *p.store
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
		body["tools"] = tools
		if name, forced := req.ToolChoice.Tool(); forced {
			body["tool_choice"] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyName: name}
		} else if req.ToolChoice == llmprovider.ToolChoiceRequired || req.ToolChoice == llmprovider.ToolChoiceNone {
			body["tool_choice"] = string(req.ToolChoice)
		}
	}
	if effort, ok := p.effort(req, model); ok {
		body["reasoning"] = map[string]any{"effort": effort}
	}
	if req.PreviousResponseID != "" {
		body["previous_response_id"] = req.PreviousResponseID
	}
	return body
}

// effort is the reasoning effort to send, and false for none: the request's
// Reasoning, else WithReasoning's, an empty effort taken from WithReasoning,
// clamped to the model's menu.
func (p *provider) effort(req *llmprovider.Request, model string) (string, bool) {
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
	clamped := clampReasoningEffort(model, string(effort))
	return clamped, clamped != ""
}

func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	reqBody, err := json.Marshal(p.body(req))
	if err != nil {
		return nil, fmt.Errorf("llmprovider: grok: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	token, err := p.src.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmprovider: grok: acquire token: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", p.userAgent)
	token.Apply(httpReq, headerAuthorization, "Bearer")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			p.logger.Debug("llmprovider: grok: close response body", "error", err)
		}
	}()
	if err := llmprovider.ClassifyHTTPError(llmprovider.ProviderGrok, resp); err != nil {
		return nil, err
	}
	// 1 MiB bounds a runaway reply.
	return responses.Decode(io.LimitReader(resp.Body, 1<<20))
}

// ListModels returns curated Grok text models available to this credential,
// probed unless WithModelProbes(false). It falls back to the static catalog.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderGrok, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = catalog.Static(llmprovider.ProviderGrok)
	}
	// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := transport.ProbeGenerateHealth(ctx, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		// The old API's probe provider: the default output limit, no store,
		// no reasoning (0015-MADR D1).
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
