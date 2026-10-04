// Package gemini is the Google Gemini provider (0015-MADR D2): the
// Interactions API with an API key (MADR 0014).
//
// Build it with New, or providers.New(llmprovider.ProviderGemini, …), with
// WithAPIKey, or WithTokenSource and a source that yields an API key, such as
// a CommandToken. An OAuth session is refused: this module does not sign in to
// Gemini subscriptions (0016-MADR D10). The key goes in x-goog-api-key, or the
// token's own Header (R16), never in the URL.
//
// Capabilities: tools, forced tool choice and reasoning are Supported.
// Continuation is Supported only with WithStore(true): Gemini chains only from
// a stored interaction, and nothing is stored by default (MADR 0014 §2).
// There is no native streaming; llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - Reasoning takes an effort, sent as thinking_level: low, medium or high,
//     with EffortXHigh sent as high. The Interactions API has no thinking
//     budget, so Reasoning.Budget is not sent. gemini-2.5-flash-lite takes
//     only high; another effort is omitted there, leaving the model's default.
//   - ToolChoiceNone is sent as tool_choice "none", from the API reference;
//     only "any" and allowed_tools were measured (MADR 0014).
//
// Instructions and system items go to system_instruction, in that order.
//
// ListModels returns the curated listing, probing each listed model with one
// short, billed generation unless llmprovider.WithModelProbes(false) says
// otherwise (0016-MADR A5).
package gemini

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
)

const (
	defaultBaseURL  = "https://generativelanguage.googleapis.com/v1beta"
	googleKeyHeader = "x-goog-api-key"
	probePrompt     = "Respond with ONLY the word Hello"
)

// storeOption is WithStore's value.
type storeOption bool

// WithStore sets whether interactions are stored (MADR 0014 §2). Gemini stores
// nothing unless given true, and continuing from a PreviousResponseID needs
// it: without it, continuation is Unsupported.
func WithStore(store bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderGemini, "gemini.WithStore", storeOption(store))
}

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string
	client    *http.Client
	caps      llmprovider.Capabilities
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	store     bool
	userAgent string
	logger    *slog.Logger
	probe     bool
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
}

// New builds the Gemini provider. It needs an API key: WithAPIKey, or
// WithTokenSource with a source that yields one.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderGemini, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: gemini needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: gemini api key is required", llmprovider.ErrInvalidRequest) // 0020-MADR F52
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: gemini takes an API key, not an OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported)
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
		probe:     st.ModelProbes(),
		caps: llmprovider.Capabilities{
			Tools:            llmprovider.Supported,
			ForcedToolChoice: llmprovider.Supported,
			Reasoning:        llmprovider.Supported,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Unsupported,
		},
	}
	for _, v := range st.Values() {
		if store, ok := v.(storeOption); ok {
			p.store = bool(store)
		}
	}
	if p.store {
		p.caps.Continuation = llmprovider.Supported
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderGemini }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Interactions API.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2).
	return wire.Reauth(ctx, string(llmprovider.ProviderGemini), p.src, func(token llmprovider.Token) (*llmprovider.Response, error) {
		return p.generateOnce(ctx, req, token)
	})
}

// generateOnce sends req once, with token.
func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) (*llmprovider.Response, error) {
	return wire.Post(ctx, wire.Call{
		Provider: string(llmprovider.ProviderGemini),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + interactionsPath,
		Body:     p.body(req),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			// In a header, not the URL, so it cannot leak through a *url.Error.
			token.Apply(r, googleKeyHeader, "")
		},
	}, token, decodeInteraction)
}

// body is the Interactions request for req (MADR 0014 §1).
func (p *provider) body(req *llmprovider.Request) map[string]any {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := p.maxTokens
	if req.MaxOutputTokens > 0 {
		maxTokens = req.MaxOutputTokens
	}
	gen := map[string]any{"max_output_tokens": maxTokens}
	if r := p.effectiveReasoning(req); r != nil {
		// Without it a thought step carries only its signature.
		gen["thinking_summaries"] = interactionSummariesAuto
		if level := thinkingLevel(model, r.Effort); level != "" {
			gen["thinking_level"] = level
		}
	}
	body := map[string]any{
		jsonKeyModel:        model,
		jsonKeyInput:        interactionsInput(req.Input),
		"store":             p.store,
		"generation_config": gen,
	}
	var system []string
	for _, part := range []string{req.Instructions, wire.SystemPrompt(req.Input)} {
		if part != "" {
			system = append(system, part)
		}
	}
	if len(system) > 0 {
		body["system_instruction"] = strings.Join(system, "\n\n")
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]any{
				jsonKeyType:        jsonKeyFunction,
				jsonKeyName:        tool.Name,
				jsonKeyDescription: tool.Description,
				jsonKeyParameters:  wire.ToolSchema(tool.Schema),
			}
		}
		body[jsonKeyTools] = tools
		// tool_choice belongs in generation_config; top level is refused.
		if choice := toolChoice(req.ToolChoice); choice != nil {
			gen["tool_choice"] = choice
		}
	}
	if req.PreviousResponseID != "" {
		body["previous_interaction_id"] = req.PreviousResponseID
	}
	return body
}

// effectiveReasoning is the request's Reasoning, else WithReasoning's, with
// an effort the request leaves empty taken from WithReasoning; nil for none.
func (p *provider) effectiveReasoning(req *llmprovider.Request) *llmprovider.Reasoning {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return nil
	}
	out := *r
	if out.Effort == "" && p.reasoning != nil {
		out.Effort = p.reasoning.Effort
	}
	return &out
}

// toolChoice is generation_config.tool_choice for choice, or nil for auto.
func toolChoice(choice llmprovider.ToolChoice) any {
	if name, named := choice.Tool(); named {
		return map[string]any{"allowed_tools": map[string]any{"mode": "any", "tools": []string{name}}}
	}
	switch choice {
	case llmprovider.ToolChoiceRequired:
		return "any"
	case llmprovider.ToolChoiceNone:
		return "none"
	default:
		return nil
	}
}

// ListModels returns a short, curated list of production text models: the
// Models API intersected with the static catalog, probed unless
// WithModelProbes(false). It falls back to the static catalog, and returns
// the curated list when every probe fails.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderGemini, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = catalog.Static(llmprovider.ProviderGemini)
	}
	// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := transport.ProbeGenerateHealth(ctx, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		// The old API's probe: the provider's output limit, nothing stored, no
		// reasoning (0015-MADR D1).
		probe := *p
		probe.model, probe.store, probe.reasoning = model, false, nil
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
