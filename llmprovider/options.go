package llmprovider

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// providerConfig holds what the common options set; Settings reads it.
type providerConfig struct {
	HTTPClient *http.Client
	MaxTokens  int
	BaseURL    string // For Ollama URL and test injection
	// ModelMetadataURL overrides the models.dev-format document the open
	// catalogs are ranked with, and OpenCode's chat route reads
	// reasoning_options from. Empty uses https://models.opencode.ai/api.json;
	// catalog.OptionsFromEnv sets it from LLMPROVIDER_MODELS_METADATA_URL.
	ModelMetadataURL string
	// ClientName and ClientVersion name the consuming application in
	// User-Agent (MADR 0012 §1.4); see WithClientInfo.
	ClientName    string
	ClientVersion string
	// SessionID is the conversation id OpenCode and Kilo receive; see
	// WithSessionID.
	SessionID string
	// DisableModelProbes stops ListModels probing each listed model; the
	// zero value probes. See WithModelProbes and ModelProbesFromEnv.
	DisableModelProbes bool
	// DisableModelMetadata stops the metadata fetch; see
	// WithoutModelMetadata.
	DisableModelMetadata bool
	// ForcedChoiceFallback and ModelForcedChoice are what a provider does
	// with a forced tool choice a model refuses, and the caller's per-model
	// facts; see WithForcedChoiceFallback and WithModelForcedChoice
	// (0034-MADR D9, D10).
	ForcedChoiceFallback ForcedChoiceFallback
	ModelForcedChoice    map[string]Support
	// ReasoningRefusals are the thinking settings the caller passed back;
	// see WithReasoningRefusals (0062-MADR D6).
	ReasoningRefusals []ReasoningRefusal
}

// WithHTTPClient sets the HTTP client the provider sends with: its requests,
// its listings and a session's refresh (R32). Nil is the provider's own
// default client, NewHTTPClient's. To add request headers, wrap
// NewHTTPClient's Transport in an http.RoundTripper that sets them, and give
// that client; what the transport needs to know per request, such as the
// request's model, goes in the request's context (0061-MADR D6; 0066-MADR
// D2).
func WithHTTPClient(c *http.Client) Option {
	return commonOption("WithHTTPClient", func(cfg *providerConfig) {
		cfg.HTTPClient = c
	})
}

// NewHTTPClient returns the client a provider builds when it is given none:
// a new http.Client on each call, over the transport the process shares, so
// that every client shares its connections. Its bounds are the transport's:
// a connection within 30 s, a TLS handshake within 10 s, the first byte of a
// reply within 300 s, idle connections closed after 90 s and 16 kept per
// host, and no total timeout, so a streamed reply runs while it sends. Its
// Transport is an http.RoundTripper to wrap, not an *http.Transport to
// configure: wrap it to add request headers, and give the client to
// WithHTTPClient (0066-MADR D2). It returns no error, and the client is safe
// for concurrent use, as an http.Client is.
func NewHTTPClient() *http.Client {
	return transport.DefaultClient()
}

// WithMaxTokens sets the maximum response tokens for the provider. It must be
// positive: New refuses another value with ErrInvalidRequest (R23).
func WithMaxTokens(n int) Option {
	return commonOption("WithMaxTokens", func(cfg *providerConfig) {
		cfg.MaxTokens = n
	})
}

// WithBaseURL sets a custom base URL for the provider (e.g., Ollama endpoint or
// test URL). A trailing slash is dropped.
func WithBaseURL(url string) Option {
	return commonOption("WithBaseURL", func(cfg *providerConfig) {
		cfg.BaseURL = url
	})
}

// WithModelProbes enables or disables listing probes. With probes, which is
// the default, ListModels on OpenAI (API key), Claude, Gemini, Grok and
// Ollama sends one short generation to each of the first catalog.MaxListed
// listed models, and drops those that do not answer. The later models are
// returned unprobed (0038-MADR D4). Every probe is a billed request. Other
// providers never probe (0016-MADR A5).
func WithModelProbes(enabled bool) Option {
	return commonOption("WithModelProbes", func(cfg *providerConfig) {
		cfg.DisableModelProbes = !enabled
	})
}

// envModelProbes names the variable ModelProbesFromEnv reads.
const envModelProbes = "LLMPROVIDER_PROBES"

// ModelProbesFromEnv returns WithModelProbes set from LLMPROVIDER_PROBES
// ("true" or "false", as strconv.ParseBool reads them), or an option that
// changes nothing when the variable is unset or not a boolean. It is the only
// way the variable takes effect: the package reads no environment variable
// unless a caller asks (0015-MADR D9). Options apply in order, so pass it
// before any explicit WithModelProbes that should win.
func ModelProbesFromEnv() Option {
	value, ok := os.LookupEnv(envModelProbes)
	if !ok {
		return Option{name: "ModelProbesFromEnv"}
	}
	enabled, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return Option{name: "ModelProbesFromEnv"}
	}
	return WithModelProbes(enabled)
}

// WithModelMetadataURL overrides the model metadata document (MADR 0009 §2)
// that the open-catalog providers read: OpenCode, Hugging Face, Kilo and
// Together. Every provider takes it; one that reads no metadata ignores it
// (0015-MADR, amendment "the OpenCode family"). WithoutModelMetadata turns
// the fetch off whatever the URL.
func WithModelMetadataURL(url string) Option {
	return commonOption("WithModelMetadataURL", func(cfg *providerConfig) {
		cfg.ModelMetadataURL = url
	})
}

// WithoutModelMetadata turns the model metadata fetch off, for every provider
// and listing: the open catalogs rank without it, and OpenCode routes by its
// table. It is how a caller, or a test, keeps a listing off the network
// (0015-MADR amendment "no ambient state, in detail"); catalog.OptionsFromEnv
// sets it from LLMPROVIDER_DISABLE_MODELS_METADATA.
func WithoutModelMetadata() Option {
	return commonOption("WithoutModelMetadata", func(cfg *providerConfig) {
		cfg.DisableModelMetadata = true
	})
}
