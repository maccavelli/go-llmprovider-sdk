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
// Unsupported: Chat Completions is stateless. There is no native streaming;
// llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - Reasoning takes an effort, sent as reasoning_effort (medium with none);
//     a Budget is not sent.
//   - Instructions are sent as a leading system message.
//   - ToolChoiceRequired and ToolChoiceNone are sent as "required" and
//     "none", the Chat Completions values; only a named tool was measured.
//
// ListModels returns the curated router listing, ranked by the metadata the
// router publishes (llmprovider.WithModelMetadataURL). It never probes, as the
// router meters every call (MADR 0012 §1.6).
package huggingface

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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
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
}

// New builds the Hugging Face provider. It needs a token.
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
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
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

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderHuggingFace }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the router's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2).
	return wire.Reauth(p.src, func() (*llmprovider.Response, error) { return p.generateOnce(ctx, req) })
}

// generateOnce sends req once, with a token fetched for this send.
func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	reqBody, err := json.Marshal(p.body(req))
	if err != nil {
		return nil, fmt.Errorf("llmprovider: huggingface: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	token, err := p.src.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("llmprovider: huggingface: acquire token: %w", err)
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
			p.logger.Debug("llmprovider: huggingface: close response body", "error", err)
		}
	}()
	if err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderHuggingFace), resp); err != nil {
		return nil, err
	}
	// 1 MiB bounds a runaway reply.
	// The reply is bounded, and a failure to read it gets its kind and
	// name (0020-MADR F9).
	out, err := chatcompletions.Decode(io.LimitReader(resp.Body, wire.ReplyLimit))
	return out, wire.DecodeError("huggingface", err)
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
	return chatcompletions.Body(model, maxTokens, input, chatcompletions.Opts{ReasoningEffort: p.effort(req),
		Tools: req.Tools, ToolChoice: req.ToolChoice})
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
		listed = catalog.Static(llmprovider.ProviderHuggingFace)
	}
	return listed, nil
}
