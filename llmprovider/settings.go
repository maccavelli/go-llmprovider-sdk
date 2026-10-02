package llmprovider

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// Option configures a provider when it is built (0015-MADR D5). The common
// options are in this package. A provider package defines its own with
// ScopedOption or ScopedOptionFor, and New refuses one meant for another
// provider unless For scopes it. Options apply in order, so a later one wins;
// a For's options apply after all the others.
type Option struct {
	name      string       // the constructor's name, for errors
	scoped    bool         // a provider-specific option
	providers []ProviderID // the ids a provider-specific option is for
	apply     func(*settings)
	value     any // a provider-specific option's value

	overlay bool       // a For
	target  ProviderID // the id a For is for
	inner   []Option   // a For's options
}

// For scopes opts to the provider id (0015-MADR D5): they apply only when
// building id, after every option given outside a For, so the more specific
// wins wherever it sits in the list. For any other id they are skipped. A
// provider-specific option inside For(id, …) may therefore sit in a list
// shared by several providers; given bare, it is still refused by another
// provider's New.
//
// Every New refuses a For with no id, and a For nested inside a For of
// another id, with an error matching ErrInvalidRequest.
func For(id ProviderID, opts ...Option) Option {
	return Option{name: fmt.Sprintf("For(%q, …)", id), overlay: true, target: id, inner: slices.Clone(opts)}
}

// overlayOptions is a For's options, with any For of the same id nested in
// it flattened in place.
func overlayOptions(o Option) ([]Option, error) {
	if o.target == "" {
		return nil, fmt.Errorf("%w: For needs a provider id", ErrInvalidRequest)
	}
	var out []Option
	for _, inner := range o.inner {
		if !inner.overlay {
			out = append(out, inner)
			continue
		}
		if inner.target != o.target {
			return nil, fmt.Errorf("%w: %s inside %s; a For applies to one provider", ErrInvalidRequest, inner.name, o.name)
		}
		nested, err := overlayOptions(inner)
		if err != nil {
			return nil, err
		}
		out = append(out, nested...)
	}
	return out, nil
}

// settings is what the options build.
type settings struct {
	cfg       providerConfig // what the common options set
	model     string
	tokens    TokenSource
	logger    *slog.Logger
	reasoning *Reasoning
	values    []any
}

func newSettings() settings {
	return settings{cfg: providerConfig{HTTPClient: transport.DefaultClient(), MaxTokens: 8192}}
}

// commonOption is a common option that sets the providerConfig.
func commonOption(name string, set func(*providerConfig)) Option {
	return Option{name: name, apply: func(s *settings) { set(&s.cfg) }}
}

// newOption is a common option that sets the rest of the settings.
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
	identity transport.Identity
}

// ResolveOptions applies opts for the provider id, for that provider's New:
// the options outside any For in order, then those of each For(id, …) in
// order. It refuses, with an error matching ErrInvalidRequest, an option
// scoped to another provider that no For scopes away, and a malformed For.
func ResolveOptions(id ProviderID, opts []Option) (*Settings, error) {
	s := newSettings()
	var overlays []Option
	for _, opt := range opts {
		if !opt.overlay {
			if err := s.take(id, opt); err != nil {
				return nil, err
			}
			continue
		}
		inner, err := overlayOptions(opt)
		if err != nil {
			return nil, err
		}
		if opt.target == id {
			overlays = append(overlays, inner...)
		}
	}
	for _, opt := range overlays {
		if err := s.take(id, opt); err != nil {
			return nil, err
		}
	}
	return &Settings{s: s, identity: identityOf(s.cfg)}, nil
}

// take applies one option that is not a For, refusing one scoped to another
// provider.
func (s *settings) take(id ProviderID, opt Option) error {
	switch {
	case opt.scoped && !slices.Contains(opt.providers, id):
		return fmt.Errorf("%w: option %s is for %s, not %q", ErrInvalidRequest, opt.name, scopeText(opt.providers), id)
	case opt.scoped:
		s.values = append(s.values, opt.value)
	}
	if opt.apply != nil {
		opt.apply(s)
	}
	return nil
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
func (st *Settings) UserAgent() string { return st.identity.UserAgent() }

// SessionID is the id from WithSessionID, or a random one fixed for these
// Settings.
func (st *Settings) SessionID() string { return st.identity.Session }

// ClientName is the application WithClientInfo named, or "go-llmprovider-sdk":
// what User-Agent leads with, and Kilo's editor name (MADR 0012 §1.4).
func (st *Settings) ClientName() string { return st.identity.Name }

// ModelProbes reports whether ListModels probes each listed model, as
// WithModelProbes and ModelProbesFromEnv set it; true by default
// (0016-MADR A5).
func (st *Settings) ModelProbes() bool { return !st.s.cfg.DisableModelProbes }

// ModelMetadataURL is the metadata document from WithModelMetadataURL, or
// empty for the default.
func (st *Settings) ModelMetadataURL() string { return st.s.cfg.ModelMetadataURL }

// Values returns the values of the provider-specific options, in order.
func (st *Settings) Values() []any { return append([]any(nil), st.s.values...) }
