package catalog

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
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
	userAgent        string
	session          string
	metadataOff      bool         // WithoutModelMetadata
	logger           *slog.Logger // Settings.Logger
}

// profileOption is WithProfile's value; kiloOrganizationOption is
// WithKiloOrganization's.
type (
	profileOption          Profile
	kiloOrganizationOption string
)

// builtinIDs are the ids WithProfile is scoped to: every built-in provider,
// so that one option list carrying it is valid for each.
var builtinIDs = []llmprovider.ProviderID{
	llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok,
	llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo, llmprovider.ProviderHuggingFace,
	llmprovider.ProviderKilo, llmprovider.ProviderTogether, llmprovider.ProviderOllama,
}

// WithProfile selects how the open catalogs (Kilo, OpenCode Zen and Go,
// Hugging Face, Together) rank their recommended models. The zero value is
// ProfileUtility. It is scoped to every built-in provider: the open-catalog
// providers' ListModels rank with it, and the others ignore it.
func WithProfile(p Profile) llmprovider.Option {
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithProfile", profileOption(p))
}

// WithKiloOrganization lists a Kilo organization's catalog: the request
// carries X-KILOCODE-ORGANIZATIONID and asks for the organization's models
// (MADR 0012 §3.3). It is scoped to kilo. kilo.WithOrganization is the
// generation option, and kilo's ListModels passes its organization on.
func WithKiloOrganization(org string) llmprovider.Option {
	return llmprovider.ScopedOption(llmprovider.ProviderKilo, "catalog.WithKiloOrganization", kiloOrganizationOption(org))
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
	}
	for _, v := range st.Values() {
		switch v := v.(type) {
		case profileOption:
			cfg.ModelProfile = Profile(v)
		case kiloOrganizationOption:
			cfg.KiloOrganization = string(v)
		}
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

// tokenHeader is the header name and value that send tok, by R16's rule
// (Token.Apply), for a request built later.
func tokenHeader(tok llmprovider.Token, header, scheme string) (name, value string) {
	req := &http.Request{Header: http.Header{}}
	tok.Apply(req, header, scheme)
	for k, v := range req.Header {
		return k, v[0]
	}
	return header, ""
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
