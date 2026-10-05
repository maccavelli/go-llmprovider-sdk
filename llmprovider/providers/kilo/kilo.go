// Package kilo is the Kilo Gateway provider (0015-MADR D2), the inference API
// behind the Kilo Code agent.
//
// Build it with New, or providers.New(llmprovider.ProviderKilo, …). The
// credential is a Kilo API key, WithAPIKey, or the session auth's Kilo device
// login returns, WithTokenSource (0017-MADR D2); without one, Kilo's anonymous
// token is sent, which free models accept (MADR 0012 §1.7). Another
// provider's OAuth session, or a vendor CLI login, is refused. The token goes
// in Authorization as a bearer token, or in its own Header (R16).
// A URL-prefixed token ("https://host/prefix:secret") selects that backend,
// and may name the organization (MADR 0012 §3.3).
//
// Only POST {base}/chat/completions is used. Kilo also answers /responses and
// /messages with the same model — it translates formats, unlike OpenCode —
// but those routes are undocumented, and Kilo's /responses puts reasoning
// where the Responses decoder does not read it. One route reaches the whole
// catalog.
//
// Capabilities: tools are Supported. Forced tool choice and reasoning are
// BestEffort: they are sent only when the model accepts them, as WithCapabilities
// declares (all are sent when it is not given). Continuation is Unsupported:
// Chat Completions is stateless. There is no native streaming;
// llmprovider.Stream emits Generate's result.
//
// Degradations:
//   - Reasoning takes an effort: Kilo's reasoning object {effort}, or
//     {enabled: true} with none, when the model accepts "reasoning"; else
//     reasoning_effort (medium with none) when it accepts that. A Budget is
//     not sent.
//   - A tool choice is sent only when the model accepts tool_choice;
//     otherwise the tools are offered unforced.
//   - Instructions are sent as a leading system message.
//
// Requests carry provider.data_collection "deny" unless WithDataCollection(true)
// (MADR 0012 §3.3). ListModels returns the curated gateway listing; it never
// probes, as the gateway meters every call (MADR 0012 §1.6).
package kilo

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/chatcompletions"
)

// The wire shapes here were measured against the live gateway on the date
// llmprovider's wireShapesProbedOnKilo holds, which its live probes check.

const (
	// anonymousToken is the bearer Kilo's own client sends without a login;
	// free models answer it and paid ones fail with a typed error (MADR 0012
	// §1.7).
	anonymousToken = "anonymous"

	headerAuthorization = "Authorization"
	// Kilo keys the editor, the task and the organization on these headers
	// (llmprovider's identification.go has the same names, for its tests).
	headerEditor       = "X-KILOCODE-EDITORNAME"
	headerTask         = "X-KiloCode-TaskId"
	headerOrganization = "X-KILOCODE-ORGANIZATIONID"

	paramTools           = "tools"
	paramToolChoice      = "tool_choice"
	paramReasoning       = "reasoning"
	paramReasoningEffort = "reasoning_effort"
)

// Kilo's scoped options' values.
type (
	organizationOption   string
	capabilitiesOption   []string
	dataCollectionOption bool
)

// WithOrganization scopes generation and listing to a Kilo organization:
// requests carry X-KILOCODE-ORGANIZATIONID and the listing is the
// organization's, as Kilo's client does (MADR 0012 §3.3). A URL-prefixed token
// whose path is .../api/organizations/{id} names the organization without it.
func WithOrganization(id string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithOrganization", organizationOption(id))
}

// WithCapabilities declares the request parameters the model accepts, as
// published in its supported_parameters (catalog.KiloModelCapabilities).
// Without it every parameter is sent: the gateway is the authority, and not
// sending one only because it cannot be confirmed would degrade requests.
func WithCapabilities(params ...string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithCapabilities",
		capabilitiesOption(append([]string(nil), params...)))
}

// WithDataCollection lets Kilo route to upstreams that may train on prompts
// when allow is true: it drops the data_collection "deny" preference every
// request otherwise sends (MADR 0012 §3.3). A model that requires collection is
// refused without it, with ErrNotPermitted and data_collection_required.
func WithDataCollection(allow bool) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "kilo.WithDataCollection", dataCollectionOption(allow))
}

type provider struct {
	src       llmprovider.TokenSource
	model     string
	baseURL   string // WithBaseURL's, or "" for Kilo Gateway
	org       string // WithOrganization's, or ""
	client    *http.Client
	capsKnown bool
	caps      map[string]bool
	collect   bool
	maxTokens int
	reasoning *llmprovider.Reasoning // WithReasoning's default, or nil
	userAgent string
	editor    string
	session   string
	logger    *slog.Logger
	// listing carries the caller's options, the session and the transport to
	// catalog's listing.
	listing []llmprovider.Option
}

// New builds the Kilo provider.
func New(opts ...llmprovider.Option) (llmprovider.Provider, error) {
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, opts)
	if err != nil {
		return nil, err
	}
	src := st.TokenSource()
	switch s := src.(type) {
	case nil:
		src = llmprovider.NewStaticToken(anonymousToken)
	case *llmprovider.StaticToken:
		if s.Value == "" {
			src = &llmprovider.StaticToken{Value: anonymousToken, Header: s.Header}
		}
	case *auth.OAuthSession:
		// The Kilo device login's session is a Kilo credential, applied like
		// a key (0017-MADR D2). R16: another provider's session is refused.
		if s.Provider != llmprovider.ProviderKilo {
			return nil, fmt.Errorf("%w: kilo takes an API key or a Kilo device-login session, not a %s session (0016-MADR D10)",
				llmprovider.ErrUnsupported, s.Provider)
		}
	case *auth.VendorCLISession:
		return nil, fmt.Errorf("%w: kilo takes an API key or a Kilo device-login session, not a vendor CLI login (0016-MADR D10)",
			llmprovider.ErrUnsupported)
	}
	p := &provider{
		src:       src,
		model:     st.Model(),
		baseURL:   st.BaseURL(),
		client:    st.HTTPClient(),
		maxTokens: st.MaxTokens(),
		reasoning: st.Reasoning(),
		userAgent: st.UserAgent(),
		editor:    st.ClientName(),
		session:   st.SessionID(),
		logger:    st.Logger(),
	}
	for _, v := range st.Values() {
		switch v := v.(type) {
		case organizationOption:
			p.org = string(v)
		case capabilitiesOption:
			p.capsKnown, p.caps = true, map[string]bool{}
			for _, c := range v {
				p.caps[c] = true
			}
		case dataCollectionOption:
			p.collect = bool(v)
		}
	}
	p.listing = append(append([]llmprovider.Option(nil), opts...),
		llmprovider.WithSessionID(p.session), llmprovider.WithHTTPClient(p.client))
	if p.org != "" {
		p.listing = append(p.listing, catalog.WithKiloOrganization(p.org))
	}
	return p, nil
}

func (p *provider) ID() llmprovider.ProviderID { return llmprovider.ProviderKilo }

// Capabilities reports what Kilo offers. Tools are always offered, as before;
// a tool choice and reasoning are sent only where the model accepts them.
func (p *provider) Capabilities() llmprovider.Capabilities {
	return llmprovider.Capabilities{
		Tools:            llmprovider.Supported,
		ForcedToolChoice: llmprovider.BestEffort,
		Reasoning:        llmprovider.BestEffort,
		Continuation:     llmprovider.Unsupported,
		NativeStreaming:  llmprovider.Unsupported,
	}
}

// supports reports whether the model accepts a request parameter. Unknown
// capabilities send everything.
func (p *provider) supports(param string) bool { return !p.capsKnown || p.caps[param] }

// Generate sends req to the gateway's chat completions.
func (p *provider) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := p.Capabilities().Check(req); err != nil {
		return nil, err
	}
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2).
	return wire.Reauth(ctx, string(llmprovider.ProviderKilo), p.src, func(token llmprovider.Token) (*llmprovider.Response, error) {
		return p.generateOnce(ctx, req, token)
	})
}

// generateOnce sends req once, with token. The token names the gateway and
// the organization.
func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) (*llmprovider.Response, error) {
	endpoints := kiloendpoint.Resolve(p.baseURL, token.Value, p.org)
	gateway, org := endpoints.Gateway, endpoints.Org
	return wire.Post(ctx, wire.Call{
		Provider: string(llmprovider.ProviderKilo),
		Client:   p.client,
		Logger:   p.logger,
		URL:      gateway + "/chat/completions",
		Body:     p.body(req),
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			r.Header.Set(headerEditor, p.editor)
			r.Header.Set(headerTask, p.session)
			token.Apply(r, headerAuthorization, "Bearer")
			if org != "" {
				r.Header.Set(headerOrganization, org)
			}
		},
	}, token, chatcompletions.Decode)
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
	effort, reasoning := p.reasoningFields(req)
	body := chatcompletions.Body(model, maxTokens, input, chatcompletions.Opts{
		ReasoningEffort: effort,
		Reasoning:       reasoning,
		Tools:           req.Tools,
		ToolChoice:      req.ToolChoice,
		// 301 of 366 models accept "tools" but only 279 accept
		// "tool_choice"; offering the tools unforced is strictly better than
		// a 400, and ToolChoiceNone sends none (0020-MADR F40).
		NoToolChoice: !p.supports(paramToolChoice),
		// The gateway's reasoning_details go back as they came (0021-MADR W9).
		ReplayReasoningDetails: true,
	})
	if !p.collect {
		// Kilo's opt-out from upstreams that train on prompts, which its client
		// sends with hide_prompt_training_models (MADR 0012 §3.3).
		body["provider"] = map[string]any{"data_collection": "deny"}
	}
	return body
}

// reasoningFields are one call's reasoning fields (MADR 0009 §6): with
// reasoning requested, Kilo's reasoning object when the model accepts
// "reasoning" — {effort} with an effort, else {enabled: true} — else
// reasoning_effort, medium with none, when it accepts that.
func (p *provider) reasoningFields(req *llmprovider.Request) (effort string, reasoning map[string]any) {
	r := req.Reasoning
	if r == nil {
		r = p.reasoning
	}
	if r == nil {
		return "", nil
	}
	e := r.Effort
	if e == "" && p.reasoning != nil {
		e = p.reasoning.Effort
	}
	switch {
	case p.supports(paramReasoning):
		if e != "" {
			return "", map[string]any{"effort": string(e)}
		}
		return "", map[string]any{"enabled": true}
	case p.supports(paramReasoningEffort):
		if e != "" {
			return string(e), nil
		}
		return string(llmprovider.EffortMedium), nil
	}
	return "", nil
}

// ListModels returns the curated gateway listing, falling back to the static
// catalog. It spends no generation on probes (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	opts := p.listing
	if p.baseURL != "" {
		opts = append(append([]llmprovider.Option(nil), opts...), llmprovider.WithBaseURL(p.baseURL))
	}
	cat, err := catalog.List(ctx, llmprovider.ProviderKilo, p.src, opts...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = catalog.Static(llmprovider.ProviderKilo)
	}
	return listed, nil
}
