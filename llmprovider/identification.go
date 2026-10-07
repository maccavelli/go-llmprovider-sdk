package llmprovider

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// Client identification (MADR 0012 §1.4): every request names this module and the
// consuming application honestly, and never a reference client. The identity
// and its User-Agent are internal/transport's; the options that set it are
// here.

// WithClientInfo names the consuming application: it leads User-Agent and is
// Kilo's editor name. An empty name keeps "go-llmprovider-sdk"; an empty version keeps the
// build's own.
func WithClientInfo(name, version string) Option {
	return commonOption("WithClientInfo", func(c *providerConfig) {
		c.ClientName = name
		c.ClientVersion = version
	})
}

// WithSessionID sets the conversation id sent as x-opencode-session, Kilo's
// task id, and the ChatGPT backend's session-id. Empty keeps a random id,
// fixed for the provider's lifetime.
func WithSessionID(id string) Option {
	return commonOption("WithSessionID", func(c *providerConfig) {
		c.SessionID = id
	})
}

// identityOf resolves cfg's identity with transport.NewIdentity.
func identityOf(cfg providerConfig) transport.Identity {
	return transport.NewIdentity(cfg.ClientName, cfg.ClientVersion, cfg.SessionID)
}
