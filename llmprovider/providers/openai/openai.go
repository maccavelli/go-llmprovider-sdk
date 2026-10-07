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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
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
	ChatGPTBaseURL = "https://chatgpt.com/backend-api/codex"
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
	// catalog's listing.
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
	p.baseURL = PlatformBaseURL
	if p.chatGPT {
		p.baseURL = ChatGPTBaseURL
		p.caps.Continuation = llmprovider.Unsupported
	}
	if base := st.BaseURL(); base != "" {
		p.baseURL = base
	}
	shareHTTPClient(src, p.client)
	shareLogger(src, st.Logger())
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
	// A 401 renews an InvalidatingSource's token and sends once more
	// (0020-MADR F2).
	return wire.Reauth(ctx, string(llmprovider.ProviderOpenAI), p.src, func(token llmprovider.Token) (*llmprovider.Response, error) {
		return p.generateOnce(ctx, req, token)
	})
}

// body is the Responses request for req.
// includeEncryptedReasoning asks a stateless Responses request for its
// reasoning, encrypted, so the next turn can replay it.
const includeEncryptedReasoning = "reasoning.encrypted_content"

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
		body["include"] = []string{includeEncryptedReasoning}
		body["prompt_cache_key"] = p.session
	} else {
		maxTokens := p.maxTokens
		if req.MaxOutputTokens > 0 {
			maxTokens = req.MaxOutputTokens
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
	}
	wire.AddResponsesTools(body, req.Tools, req.ToolChoice)
	if effort, ok := p.effort(req); ok {
		// The summary makes reasoning items carry text (0020-MADR F24).
		body["reasoning"] = map[string]any{"effort": effort, "summary": "auto"}
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

// generateOnce sends req once, with token. A ChatGPT session's reply is an
// event stream, bounded per event rather than as a whole.
func (p *provider) generateOnce(ctx context.Context, req *llmprovider.Request, token llmprovider.Token) (*llmprovider.Response, error) {
	decode := responses.Decode
	if p.chatGPT {
		decode = func(r io.Reader) (*llmprovider.Response, error) {
			return responses.ReadStream(string(llmprovider.ProviderOpenAI), r)
		}
	}
	return wire.Post(ctx, wire.Call{
		Provider: string(llmprovider.ProviderOpenAI),
		Client:   p.client,
		Logger:   p.logger,
		URL:      p.baseURL + "/responses",
		Body:     p.body(req),
		Stream:   p.chatGPT,
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("User-Agent", p.userAgent)
			token.Apply(r, headerAuthorization, "Bearer")
			if p.chatGPT {
				r.Header.Set("Accept", "text/event-stream")
				r.Header.Set(headerSession, p.session)
				setChatGPTHeaders(r, p.src)
				if residency := residency(token.Value); residency != "" {
					r.Header.Set(headerResidency, residency)
				}
			}
		},
	}, token, decode)
}

// ListModels returns the curated chat models this credential can use, never
// the raw /v1/models dump. An API key's models are probed unless
// WithModelProbes(false); a ChatGPT session's are not (MADR 0012 §1.6).
func (p *provider) ListModels(ctx context.Context) ([]string, error) {
	// A ChatGPT session lists from the Codex backend, with no static catalog
	// (MADR 0008 D11), and its subscription meters every call: no generation
	// probe (MADR 0012 §1.6). A 401 renews the session's token and lists
	// once more, as Generate does (0020-MADR F2, F38).
	if p.chatGPT {
		return wire.Reauth(ctx, string(llmprovider.ProviderOpenAI), p.src, func(token llmprovider.Token) ([]string, error) {
			return p.listChatGPT(ctx, token)
		})
	}
	cat, err := catalog.List(ctx, llmprovider.ProviderOpenAI, p.src, p.listing...)
	listed := cat.Recommended
	if err != nil || len(listed) == 0 {
		listed = catalog.Static(llmprovider.ProviderOpenAI)
	}
	if !p.probe {
		return listed, nil
	}
	healthy := transport.ProbeGenerateHealth(ctx, listed, catalog.MaxListed, func(ctx context.Context, model string) (string, error) {
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
}

// shareHTTPClient gives src the provider's client, when src takes one and has
// none (0016-MADR D8).
func shareHTTPClient(src llmprovider.TokenSource, client *http.Client) {
	if user, ok := src.(clientUser); ok {
		user.UseHTTPClient(client)
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
