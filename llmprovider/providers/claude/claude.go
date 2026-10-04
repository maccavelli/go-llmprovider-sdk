// Package claude is the Anthropic Claude provider (0015-MADR D2): the
// Messages API with an API key.
//
// Build it with New, or providers.New(llmprovider.ProviderClaude, …), with
// WithAPIKey, or WithTokenSource and a source that yields an API key, such as
// a CommandToken. An OAuth session is refused: this module does not sign in to
// Anthropic subscriptions (0016-MADR D10).
//
// Capabilities: tools and reasoning are Supported. Forced tool choice is
// BestEffort: with reasoning it is sent as "auto" (see Degradations,
// 0020-MADR F43).
// Continuation is Unsupported: the Messages API is stateless, so a caller
// replays prior items instead. There is no native streaming;
// llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - With Reasoning, a forced tool choice (ForceTool or ToolChoiceRequired)
//     is sent as "auto": Anthropic refuses to force a tool while thinking.
//   - Reasoning's shape follows the model. Claude 4.7 and later take adaptive
//     thinking, with the effort as output_config.effort; a Budget is not sent
//     to them. Older models take a budget: Reasoning.Budget, else 1024 for
//     EffortLow, else 4096, with the output limit raised above it.
//
// Instructions and system items go to the top-level system field, in that
// order.
//
// ListModels returns the curated listing, probing each listed model with one
// short, billed generation unless llmprovider.WithModelProbes(false) says
// otherwise (0016-MADR A5).
package claude

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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

const (
	defaultBaseURL   = "https://api.anthropic.com/v1"
	anthropicVersion = "2023-06-01"

	jsonKeyType = "type"
	jsonKeyName = "name"

	// probeMaxTokens is the output limit a listing probe sends: the default
	// of the old API's probe provider, kept so the probe's request is
	// unchanged (0015-MADR D1).
	probeMaxTokens = 8192
	probePrompt    = "Respond with ONLY the word Hello"
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
	probe     bool
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
}

// New builds the Claude provider. It needs an API key: WithAPIKey, or
// WithTokenSource with a source that yields one.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		return nil, fmt.Errorf("%w: claude needs WithAPIKey or WithTokenSource", llmprovider.ErrInvalidRequest)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			return nil, fmt.Errorf("%w: claude api key is required", llmprovider.ErrInvalidRequest)
		}
	case *auth.OAuthSession, *auth.VendorCLISession:
		// R16: a source of a kind the service does not accept is refused.
		return nil, fmt.Errorf("%w: claude takes an API key, not an OAuth session (0016-MADR D10)", llmprovider.ErrUnsupported)
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
			ForcedToolChoice: llmprovider.BestEffort,
			Reasoning:        llmprovider.Supported,
			Continuation:     llmprovider.Unsupported,
			NativeStreaming:  llmprovider.Unsupported,
		},
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(st.SessionID()), llmprovider.WithHTTPClient(p.client), llmprovider.WithBaseURL(p.baseURL))
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderClaude }

func (p *provider) Capabilities() llmprovider.Capabilities { return p.caps }

// Generate sends req to the Messages API.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.caps.Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2).
	return wire.Reauth(ctx, string(llmprovider.ProviderClaude), p.src, func(token llmprovider.Token) (*llmprovider.Response, error) {
		return p.generateOnce(ctx, req, token)
	})
}

// generateOnce sends req once, with token.
func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) (*llmprovider.Response, error) {
	return wire.Post(ctx, wire.Call{
		Provider: string(llmprovider.ProviderClaude),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/messages",
		Body:     p.body(req),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			token.Apply(r, "x-api-key", "")
			r.Header.Set("anthropic-version", anthropicVersion)
		},
	}, token, messages.Decode)
}

// body is the Messages request for req.
func (p *provider) body(req *llmprovider.Request) map[string]any {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := p.maxTokens
	if req.MaxOutputTokens > 0 {
		maxTokens = req.MaxOutputTokens
	}
	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages.FromItems(req.Input),
	}
	var system []string
	for _, part := range []string{req.Instructions, wire.SystemPrompt(req.Input)} {
		if part != "" {
			system = append(system, part)
		}
	}
	if len(system) > 0 {
		body["system"] = strings.Join(system, "\n\n")
	}
	thinking := p.addReasoning(body, req, model, maxTokens)
	if len(req.Tools) > 0 {
		body["tools"] = wire.MessagesTools(req.Tools)
		if choice := toolChoice(req.ToolChoice, thinking); choice != nil {
			body["tool_choice"] = choice
		}
	}
	return body
}

// addReasoning adds the thinking fields for the request's Reasoning, or
// WithReasoning's, and reports whether it did. Each of effort and budget
// falls back to WithReasoning's.
func (p *provider) addReasoning(body map[string]any, req *llmprovider.Request, model string, maxTokens int) bool {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return false
	}
	effort, budget := r.Effort, r.Budget
	if p.reasoning != nil {
		if effort == "" {
			effort = p.reasoning.Effort
		}
		if budget == 0 {
			budget = p.reasoning.Budget
		}
	}
	body["max_tokens"] = messages.AddThinking(body, model, string(effort), budget, maxTokens)
	return true
}

// toolChoice is the tool_choice for choice. With thinking, forcing is sent
// as "auto": Anthropic refuses a forced tool while thinking.
func toolChoice(choice llmprovider.ToolChoice, thinking bool) map[string]any {
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

// ListModels returns curated Claude text models (Models API and catalog),
// probed unless WithModelProbes(false). It falls back to the static catalog.
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	cat, err := catalog.List(ctx, llmprovider.ProviderClaude, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = catalog.Static(llmprovider.ProviderClaude)
	}
	// Probes are billed; they are on by default, and callers can turn them off (0016-MADR A5).
	if !p.probe {
		return listed, nil
	}
	healthy := transport.ProbeGenerateHealth(ctx, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
		probe := *p
		probe.model, probe.maxTokens, probe.reasoning = model, probeMaxTokens, nil
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
