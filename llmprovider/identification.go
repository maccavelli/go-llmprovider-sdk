package llmprovider

import (
	"crypto/rand"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// Client identification (MADR 0012 §1.4): every request names this module and the
// consuming application honestly, and never a reference client. The identity
// and its User-Agent are internal/transport's; the options that set it are
// here.

// WithClientInfo names the consuming application: it leads User-Agent and is
// Kilo's editor name. An empty name keeps "go-llmprovider-sdk"; an empty version keeps the
// build's own.
func WithClientInfo(name, version string) ProviderOption {
	return commonOption("WithClientInfo", func(c *ProviderConfig) {
		c.ClientName = name
		c.ClientVersion = version
	})
}

// WithSessionID sets the conversation id sent as x-opencode-session and Kilo's
// task id. Empty keeps a random id, fixed for the provider's lifetime.
func WithSessionID(id string) ProviderOption {
	return commonOption("WithSessionID", func(c *ProviderConfig) {
		c.SessionID = id
	})
}

// identityOf resolves cfg's identity: the default name is this module at its own
// version; a named application defaults to the main module's version; a missing
// session is a fresh random id.
func identityOf(cfg ProviderConfig) transport.Identity {
	sdk, main := transport.BuildVersions()
	id := transport.Identity{Name: cfg.ClientName, Version: cfg.ClientVersion, Session: cfg.SessionID}
	if id.Name == "" {
		id.Name = transport.DefaultClientName
	}
	if id.Version == "" {
		id.Version = main
		if id.Name == transport.DefaultClientName {
			id.Version = sdk
		}
	}
	if id.Session == "" {
		id.Session = rand.Text()
	}
	return id
}
