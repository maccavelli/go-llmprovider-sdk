package catalog

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/compatid"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// The listing's options: llmprovider's common ones, read through Settings, and
// the two only the listing takes (0015-MADR, amendment "`catalog` before the
// old API's removal").

// Names the listing shares with the rest of the module.
const (
	headerAuthorization = "Authorization"
	bearerScheme        = "Bearer"
	headerUserAgent     = "User-Agent"
	// serviceOpencode labels an OpenCode listing request.
	serviceOpencode = "opencode"

	effortLow    = "low"
	effortMedium = "medium"
	effortHigh   = "high"
	effortXHigh  = "xhigh"

	jsonKeyEffort    = "effort"
	jsonKeyReasoning = "reasoning"
	jsonKeyText      = "text"
	jsonKeyTools     = "tools"
)

// config is what one listing reads of its options.
type config struct {
	BaseURL          string
	HTTPClient       *http.Client
	ModelProfile     Profile
	ModelMetadataURL string
	KiloOrganization string
	ClaudeWorkspace  string
	userAgent        string
	session          string
	metadataOff      bool         // WithoutModelMetadata
	limit            int          // WithLimit's n; zero or less, the default, is no limit
	logger           *slog.Logger // Settings.Logger
	layers           []Layer      // WithModelLayer's, in the order given
	workload         workload     // WithWorkload's, else the published one
	lastGood         lastGood     // WithLastGood's
	// tokens is the credential the options give (WithAPIKey or
	// WithTokenSource), or nil: CachedFacts finds a listing by it.
	tokens llmprovider.TokenSource
	// sink, when set, receives what a listing says of each model it lists
	// (0033-MADR D2).
	sink func(ModelInfo)
	// store is WithCache's, else the process's (0048-MADR D1).
	store *store
	// compat reports a chatcompat provider's listing: an id no built-in has,
	// which the options' chatcompat.WithID names (0061-MADR, amendment
	// "chatcompat, provider-agnostic", E1).
	compat bool
}

// emit hands one listed model's metadata to the sink, if there is one.
func (c config) emit(info ModelInfo) {
	if c.sink != nil {
		c.sink(info)
	}
}

// profileOption is WithProfile's value; kiloOrganizationOption is
// WithKiloOrganization's; claudeWorkspaceOption is WithClaudeWorkspace's.
type (
	profileOption          Profile
	kiloOrganizationOption string
	claudeWorkspaceOption  string
	// modelLayerOption is WithModelLayer's value.
	modelLayerOption struct{ layer Layer }
	// workloadOption is WithWorkload's value.
	workloadOption workload
	// lastGoodOption is WithLastGood's value.
	lastGoodOption lastGood
)

// lastGood is a listing a caller kept from an earlier run: its usable ids,
// and when they were read (0041-MADR D6).
type lastGood struct {
	ids []string
	at  time.Time
}

// builtinIDs are the ids WithProfile and WithLimit are scoped to: every built-in provider,
// so that one option list carrying it is valid for each.
var builtinIDs = []llmprovider.ProviderID{
	llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok,
	llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo, llmprovider.ProviderHuggingFace,
	llmprovider.ProviderKilo, llmprovider.ProviderTogether, llmprovider.ProviderOllama,
}

// WithProfile selects how a listing ranks its recommended models: every
// provider's but Ollama's, which has no band (0041-MADR D4). The zero value
// is ProfileUtility. It is scoped to every built-in provider.
func WithProfile(p Profile) llmprovider.Option {
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithProfile", profileOption(p))
}

// WithLimit cuts the recommended list to its first n models, and a provider's
// ListModels to the same, its static fallback included. Without it, or with
// n zero or less, the list is the full curated catalog: every known, working
// model the listing serves, ranked, with the best six first (0038-MADR D1,
// D2). A listing with WithLimit(MaxListed) recommends the six it always had. A live
// listing's Usable, which search covers, is never cut; a static fallback's
// is cut with Recommended, which it equals. It is scoped to every built-in
// provider, as WithProfile is.
func WithLimit(n int) llmprovider.Option {
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithLimit", listlimit.Value(n))
}

// WithWorkload sets the agent turn ProfileAgent prices its band by: the
// share of a turn's input read from the cache, and the share of its tokens
// that are output (0041-MADR D5). Without it the shares are the published
// 0.90 and 0.01. ObservedWorkload measures a caller's own. Each must be
// within (0, 1): List, and a provider's ListModels, refuse another value
// with ErrInvalidRequest. It orders the band, and never chooses its
// members. It is scoped to every built-in provider, as WithProfile is.
func WithWorkload(cacheShare, outputShare float64) llmprovider.Option {
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithWorkload",
		workloadOption{cacheShare: cacheShare, outputShare: outputShare})
}

// WithLastGood hands in a listing the caller kept from an earlier run: its
// usable ids, as Catalog.Usable gave them, and when they were read, as
// Catalog.FetchedAt gave it. It is used only after a transient failure, when
// no listing for the same provider, base URL, organization or workspace and credential
// is cached, and only within 24 hours of fetchedAt; the Catalog is then
// stale (0041-MADR D6). Kilo's band is then ranked without Kilo's prices,
// which the ids do not carry (0041-MADR A6). Ollama ignores it. It is
// scoped to every built-in provider, as WithProfile is, but its ids are one
// provider's: pass it with llmprovider.For(id, …) in a list several
// providers share.
func WithLastGood(ids []string, fetchedAt time.Time) llmprovider.Option {
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithLastGood",
		lastGoodOption{ids: slices.Clone(ids), at: fetchedAt})
}

// WithKiloOrganization lists a Kilo organization's catalog: the request
// carries X-KILOCODE-ORGANIZATIONID and asks for the organization's models
// (MADR 0012 §3.3). It is scoped to kilo. kilo.WithOrganization is the
// generation option, and kilo's ListModels passes its organization on.
func WithKiloOrganization(org string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "catalog.WithKiloOrganization", kiloOrganizationOption(org))
}

// WithClaudeWorkspace lists Anthropic's models in a workspace: each page's
// request carries anthropic-workspace-id, which a key not scoped to a
// workspace needs. Empty sends none. The value is trimmed; one holding a
// control character is refused with ErrInvalidRequest, as it cannot be a
// header value. It is scoped to claude. claude.WithWorkspace is the
// generation option, and the claude provider's ListModels passes its
// workspace on (0064-MADR D2).
func WithClaudeWorkspace(id string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderClaude, "catalog.WithClaudeWorkspace", claudeWorkspaceOption(id))
}

// listingClient is the client catalog's own calls use when the caller gives
// none: one, shared, so connections are reused across listings. A provider
// passes its own client (R32), and keeps it: each provider instance has its
// own (0016-MADR D8; 0020-MADR F28).
var listingClient = sync.OnceValue(transport.DefaultClient)

// configFor resolves opts for a listing of id.
func configFor(id llmprovider.ProviderID, opts []llmprovider.Option) (config, error) {
	// The shared client goes first, so a caller's WithHTTPClient wins.
	opts = append([]llmprovider.Option{llmprovider.WithHTTPClient(listingClient())}, opts...)
	st, err := llmprovider.ResolveOptions(id, opts)
	if err != nil {
		return config{}, err
	}
	cfg := config{
		BaseURL:          st.BaseURL(),
		HTTPClient:       st.HTTPClient(),
		ModelMetadataURL: st.ModelMetadataURL(),
		userAgent:        st.UserAgent(),
		session:          st.SessionID(),
		metadataOff:      st.ModelMetadataDisabled(),
		logger:           st.Logger(),
		tokens:           st.TokenSource(),
		workload:         defaultWorkload,
	}
	for _, v := range st.Values() {
		switch v := v.(type) {
		case profileOption:
			cfg.ModelProfile = Profile(v)
		case kiloOrganizationOption:
			cfg.KiloOrganization = string(v)
		case claudeWorkspaceOption:
			workspace := strings.TrimSpace(string(v))
			if strings.ContainsFunc(workspace, unicode.IsControl) {
				return config{}, fmt.Errorf("%w: catalog.WithClaudeWorkspace: a workspace ID holds no control character",
					llmprovider.ErrInvalidRequest)
			}
			cfg.ClaudeWorkspace = workspace
		case modelLayerOption:
			cfg.layers = append(cfg.layers, v.layer)
		case lastGoodOption:
			cfg.lastGood = lastGood(v)
		case workloadOption:
			if !share(v.cacheShare) || !share(v.outputShare) {
				return config{}, fmt.Errorf("%w: catalog.WithWorkload(%v, %v): each share must be within (0, 1)",
					llmprovider.ErrInvalidRequest, v.cacheShare, v.outputShare)
			}
			cfg.workload = workload(v)
		}
	}
	cfg.limit = listlimit.Of(st.Values())
	cfg.store = storeIn(st.Values())
	// A chatcompat provider's model is the server's: models.dev is read for it
	// by canonical id, as its lowest layer (0061-MADR E2; its amendment "E2
	// and E4's menu, made exact before Phase 4c", decision 4, X1).
	cfg.compat = !listed[id] && compatid.For(st.Values(), string(id))
	return cfg, nil
}

// listingConfig is configFor for a listing of id, lowered: a built-in's, or a
// chatcompat provider's whose options name it with chatcompat.WithID and give
// a base URL (0061-MADR, amendment "chatcompat, provider-agnostic", E1). Any
// other id is unlistedProvider's, whatever its options.
func listingConfig(id llmprovider.ProviderID, opts []llmprovider.Option) (config, error) {
	if listed[id] {
		return configFor(id, opts)
	}
	cfg, err := configFor(id, opts)
	if err != nil || !cfg.compat {
		return config{}, unlistedProvider(id)
	}
	if cfg.BaseURL == "" {
		return config{}, fmt.Errorf("%w: model listing: chatcompat provider %q needs a base URL", llmprovider.ErrInvalidRequest, id)
	}
	return cfg, nil
}

// defaultConfig is a listing's configuration with no options.
func defaultConfig() config {
	cfg, err := configFor("", nil)
	if err != nil {
		// No option can be refused when there are none.
		panic(err)
	}
	return cfg
}

// setUserAgent names the client on req (MADR 0012 §1.4).
func (c config) setUserAgent(req *http.Request) {
	req.Header.Set(headerUserAgent, c.userAgent)
}

// bearerHeader is the header name and value that send tok as a bearer token
// in Authorization, by R16's rule (Token.Apply), for a request built later:
// every listing that sends a token sends it so.
func bearerHeader(tok llmprovider.Token) (name, value string) {
	req := &http.Request{Header: http.Header{}}
	tok.Apply(req, headerAuthorization, bearerScheme)
	for k, v := range req.Header {
		return k, v[0]
	}
	return headerAuthorization, ""
}

// closeBody closes an HTTP response body, logging a close failure at debug
// level to the listing's logger (0015-MADR D9). Each package keeps its own,
// so that bodyclose sees the close.
func (c config) closeBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	if err := resp.Body.Close(); err != nil && c.logger != nil {
		c.logger.Debug("catalog: close response body", "error", err)
	}
}
