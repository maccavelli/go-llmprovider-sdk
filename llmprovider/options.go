package llmprovider

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

// providerConfig holds what the common options set; Settings reads it.
type providerConfig struct {
	HTTPClient *http.Client
	MaxTokens  int
	BaseURL    string // For Ollama URL and test injection
	// ModelMetadataURL overrides the models.dev-format document the open
	// catalogs are ranked with, and OpenCode's chat route reads
	// reasoning_options from. Empty uses LLMPROVIDER_MODELS_METADATA_URL, then
	// https://models.opencode.ai/api.json.
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
}

// WithHTTPClient sets a custom HTTP client for connection pooling.
func WithHTTPClient(c *http.Client) Option {
	return commonOption("WithHTTPClient", func(cfg *providerConfig) {
		cfg.HTTPClient = c
	})
}

// WithMaxTokens sets the maximum response tokens for the provider.
func WithMaxTokens(n int) Option {
	return commonOption("WithMaxTokens", func(cfg *providerConfig) {
		cfg.MaxTokens = n
	})
}

// WithBaseURL sets a custom base URL for the provider (e.g., Ollama endpoint or test URL).
func WithBaseURL(url string) Option {
	return commonOption("WithBaseURL", func(cfg *providerConfig) {
		cfg.BaseURL = url
	})
}

// WithModelProbes enables or disables listing probes. With probes, which is
// the default, ListModels on OpenAI (API key), Claude, Gemini, Grok and
// Ollama sends one short generation to each listed model, up to
// MaxListedModels, and keeps those that answer. Every probe is a billed
// request. Other providers never probe (0016-MADR A5).
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
