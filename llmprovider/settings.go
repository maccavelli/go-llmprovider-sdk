package llmprovider

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

// Option configures a provider when it is built (0015-MADR D5). The common
// options are in this package. A provider package defines its own with
// ScopedOption or ScopedOptionFor, and New refuses one meant for another
// provider. Options apply in order, so a later one wins.
type Option struct {
	name      string       // the constructor's name, for errors
	scoped    bool         // a provider-specific option
	providers []ProviderID // the ids a provider-specific option is for
	legacy    bool         // an option of the old API only
	apply     func(*settings)
	value     any // a provider-specific option's value
}

// settings is what the options build.
type settings struct {
	cfg       ProviderConfig // the old API's configuration, until 0015-PLAN S8
	model     string
	tokens    TokenSource
	logger    *slog.Logger
	reasoning *Reasoning
	values    []any
}

func newSettings() settings {
	return settings{cfg: ProviderConfig{HTTPClient: defaultHTTPClient(), MaxTokens: 8192}}
}

// commonOption is an option both APIs take.
func commonOption(name string, set func(*ProviderConfig)) Option {
	return Option{name: name, apply: func(s *settings) { set(&s.cfg) }}
}

// legacyOption is an option only the old API takes.
func legacyOption(name string, set func(*ProviderConfig)) Option {
	return Option{name: name, legacy: true, apply: func(s *settings) { set(&s.cfg) }}
}

// newOption is an option only the new API takes; the old constructors ignore
// it.
func newOption(name string, set func(*settings)) Option {
	return Option{name: name, apply: set}
}

// WithModel sets the model a provider uses when a Request names none.
func WithModel(model string) Option {
	return newOption("WithModel", func(s *settings) { s.model = model })
}

// WithAPIKey sets the credential to a static API key; it is shorthand for
// WithTokenSource(NewStaticToken(key)).
func WithAPIKey(key string) Option {
	return newOption("WithAPIKey", func(s *settings) { s.tokens = NewStaticToken(key) })
}

// WithTokenSource sets where a provider gets its credential (0015-MADR D5,
// 0016-MADR D2).
func WithTokenSource(src TokenSource) Option {
	return newOption("WithTokenSource", func(s *settings) { s.tokens = src })
}

// WithLogger sets the logger a provider writes to. Without one, nothing is
// logged (0015-MADR D9).
func WithLogger(logger *slog.Logger) Option {
	return newOption("WithLogger", func(s *settings) { s.logger = logger })
}

// WithReasoning sets the reasoning a provider requests when a Request sets
// none. Nil means none.
func WithReasoning(r *Reasoning) Option {
	return newOption("WithReasoning", func(s *settings) {
		s.reasoning = nil
		if r != nil {
			copied := *r
			s.reasoning = &copied
		}
	})
}

// ScopedOption returns a provider-specific option, for a provider package to
// wrap in its own constructor, such as kilo.WithOrganization. name is that
// constructor's name. The provider's New reads value from Settings.Values;
// every other provider's New refuses the option (0015-MADR D5).
func ScopedOption(provider ProviderID, name string, value any) Option {
	return ScopedOptionFor([]ProviderID{provider}, name, value)
}

// ScopedOptionFor is ScopedOption for an option that several providers of
// one family take, such as opencode.WithRoute for both OpenCode gateways.
// Each listed provider's New reads value from Settings.Values; every other
// provider's New refuses the option, as does every New when the list is
// empty (0015-MADR, amendment "the OpenCode family").
func ScopedOptionFor(providers []ProviderID, name string, value any) Option {
	return Option{name: name, scoped: true, providers: slices.Clone(providers), value: value}
}

// scopeText names the providers a scoped option is for, for errors.
func scopeText(providers []ProviderID) string {
	if len(providers) == 1 {
		return fmt.Sprintf("provider %q", providers[0])
	}
	quoted := make([]string, len(providers))
	for i, p := range providers {
		quoted[i] = fmt.Sprintf("%q", p)
	}
	return "providers " + strings.Join(quoted, ", ")
}

// Settings is the configuration a provider's New resolved from its options.
// It cannot be changed after ResolveOptions returns it.
type Settings struct {
	s        settings
	identity clientIdentity
}

// ResolveOptions applies opts for the provider id, for that provider's New.
// It refuses, with an error matching ErrInvalidRequest, an option scoped to
// another provider, and an option only the old API takes.
func ResolveOptions(id ProviderID, opts []Option) (*Settings, error) {
	s := newSettings()
	for _, opt := range opts {
		switch {
		case opt.legacy:
			return nil, fmt.Errorf("%w: option %s belongs to the old API; %s's New does not take it", ErrInvalidRequest, opt.name, id)
		case opt.scoped && !slices.Contains(opt.providers, id):
			return nil, fmt.Errorf("%w: option %s is for %s, not %q", ErrInvalidRequest, opt.name, scopeText(opt.providers), id)
		case opt.scoped:
			s.values = append(s.values, opt.value)
		}
		if opt.apply != nil {
			opt.apply(&s)
		}
	}
	return &Settings{s: s, identity: identityOf(s.cfg)}, nil
}

// Model is the model from WithModel, or empty.
func (st *Settings) Model() string { return st.s.model }

// TokenSource is the credential from WithAPIKey or WithTokenSource, or nil.
func (st *Settings) TokenSource() TokenSource { return st.s.tokens }

// HTTPClient is the client from WithHTTPClient, or this provider's own
// default client.
func (st *Settings) HTTPClient() *http.Client { return st.s.cfg.HTTPClient }

// BaseURL is the endpoint from WithBaseURL, or empty for the provider's own.
func (st *Settings) BaseURL() string { return st.s.cfg.BaseURL }

// MaxTokens is the output limit from WithMaxTokens, or 8192.
func (st *Settings) MaxTokens() int { return st.s.cfg.MaxTokens }

// Logger is the logger from WithLogger, or one that discards everything.
func (st *Settings) Logger() *slog.Logger {
	if st.s.logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return st.s.logger
}

// Reasoning is a copy of the reasoning from WithReasoning, or nil.
func (st *Settings) Reasoning() *Reasoning {
	if st.s.reasoning == nil {
		return nil
	}
	copied := *st.s.reasoning
	return &copied
}

// UserAgent is the User-Agent every request sends: the application from
// WithClientInfo, then this module (MADR 0012 §1.4).
func (st *Settings) UserAgent() string { return st.identity.userAgent() }

// SessionID is the id from WithSessionID, or a random one fixed for these
// Settings.
func (st *Settings) SessionID() string { return st.identity.session }

// ModelProbes reports whether ListModels probes each listed model, as
// WithModelProbes and ModelProbesFromEnv set it; true by default
// (0016-MADR A5).
func (st *Settings) ModelProbes() bool { return !st.s.cfg.DisableModelProbes }

// ModelMetadataURL is the metadata document from WithModelMetadataURL, or
// empty for the default.
func (st *Settings) ModelMetadataURL() string { return st.s.cfg.ModelMetadataURL }

// Values returns the values of the provider-specific options, in order.
func (st *Settings) Values() []any { return append([]any(nil), st.s.values...) }
