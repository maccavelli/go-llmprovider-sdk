// Package together is the Together AI provider (0015-MADR D2; 0017-MADR D1),
// through its OpenAI-compatible Chat Completions API.
//
// Build it with New, or providers.New(llmprovider.ProviderTogether, …), with a
// Together API key: WithAPIKey, or WithTokenSource with a source that yields
// one. An OAuth session is refused: Together has none. The key goes in
// Authorization as a bearer token, or in its own Header (R16).
//
// Only POST {base}/chat/completions is used. The wire shapes are from
// Together's API reference, read 2026-09-30; llmprovider's live tests are
// written to confirm them against the service, and have not run yet.
//
// Capabilities: tools, forced tool choice and reasoning are BestEffort, as
// support is per model on Together (0017-MADR D1). Continuation is
// Unsupported: Chat Completions is stateless. There is no native streaming;
// llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - Reasoning sends reasoning {"enabled": true}, which Together's toggleable
//     models read, and reasoning_effort only when an effort is set: Together
//     accepts it for some models only (openai/gpt-oss-*, DeepSeek-V4-Pro). A
//     Budget is not sent. A request with no reasoning sends neither, so the
//     model's own default applies.
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
package together

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
)

const (
	// defaultBaseURL is Together's API (llmprovider's togetherBaseURL).
	defaultBaseURL      = "https://api.together.ai/v1"
	headerAuthorization = "Authorization"

	jsonKeyType       = "type"
	jsonKeyFunction   = "function"
	jsonKeyName       = "name"
	jsonKeyToolChoice = "tool_choice"
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
}

// New builds the Together AI provider. It needs a key.
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
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.BestEffort,
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.BestEffort,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Unsupported,
		},
	}
	if b := st.BaseURL(); b != "" {
		p.baseURL = strings.TrimRight(b, "/")
	}
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
	reqBody, err := json.Marshal(p.body(req))
	if err != nil {
		return nil, fmt.Errorf("llmprovider: together: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	token, err := p.src.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmprovider: together: acquire token: %w", err)
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
			p.logger.Debug("llmprovider: together: close response body", "error", err)
		}
	}()
	if err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderTogether), resp); err != nil {
		return nil, err
	}
	// 1 MiB bounds a runaway reply.
	return chatcompletions.Decode(io.LimitReader(resp.Body, 1<<20))
}

// body is the Chat Completions request for req.
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
		input = append([]llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleSystem, Text: req.Instructions}}, input...)
	}
	var o chatcompletions.Opts
	if r := p.reasoningFor(req); r != nil {
		o.Reasoning = map[string]any{"enabled": true}
		o.ReasoningEffort = p.effort(r)
	}
	body := chatcompletions.Body(model, maxTokens, input, o)
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyFunction: map[string]any{
				jsonKeyName: tool.Name, "description": tool.Description, "parameters": tool.Schema}}
		}
		body["tools"] = tools
		if name, forced := req.ToolChoice.Tool(); forced {
			body[jsonKeyToolChoice] = map[string]any{jsonKeyType: jsonKeyFunction, jsonKeyFunction: map[string]any{jsonKeyName: name}}
		} else if req.ToolChoice == llmprovider.ToolChoiceRequired || req.ToolChoice == llmprovider.ToolChoiceNone {
			body[jsonKeyToolChoice] = string(req.ToolChoice)
		}
	}
	return body
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
		listed = catalog.Static(llmprovider.ProviderTogether)
	}
	return listed, nil
}
