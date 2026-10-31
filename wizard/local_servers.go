package wizard

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// withLocalServers is o for a run under Options.LocalServers: a Registry of
// the caller's descriptors, each built by the caller's Registry, then each
// server chatcompat.Discover finds on this machine, and Providers, when set,
// with the found ids. o is a copy, and Providers is cloned before it grows,
// so the caller's Options are not changed (0066-MADR V5, Q2 (a); its
// amendment "Q2 (a)'s details"). An ended ctx gives its error, wrapped.
func withLocalServers(ctx context.Context, o Options) (Options, error) {
	var opts []llmprovider.Option
	if o.HTTPClient != nil {
		// Discover is given the client only: ProviderOptions may hold an
		// option it refuses (the amendment, C45).
		opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
	}
	servers, err := chatcompat.Discover(ctx, opts...)
	if err != nil {
		return Options{}, fmt.Errorf("wizard: look for the servers on this machine: %w", err)
	}
	callers, reg := o.registry(), llmprovider.NewRegistry()
	offered := map[string]bool{}
	for _, d := range callers.Descriptors() {
		if err := reg.Register(d, func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return callers.New(d.ID, opts...)
		}); err != nil {
			return Options{}, err
		}
		if port, ok := localPort(d); ok {
			offered[port] = true
		}
	}
	allow := slices.Clone(o.Providers)
	for _, s := range servers {
		u, err := url.Parse(s.BaseURL)
		if err != nil || offered[u.Port()] {
			continue
		}
		d := chatcompat.Descriptor(llmprovider.ProviderID("local-"+u.Port()), serverLabel(s, u.Host))
		d.DefaultBaseURL, d.Notes = s.BaseURL, serverNotes(s)
		if reg.Register(d, func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return chatcompat.New(append(slices.Clone(opts), chatcompat.WithID(d.ID))...)
		}) != nil {
			// The caller registered this id itself: its own stays.
			continue
		}
		if len(allow) > 0 {
			allow = append(allow, d.ID)
		}
	}
	o.Registry, o.Providers = reg, allow
	return o, nil
}

// localPort is the port of d's default endpoint when d runs on this machine
// at a loopback host, as Ollama does: a server found there is d.
func localPort(d llmprovider.Descriptor) (string, bool) {
	if !d.IsLocal {
		return "", false
	}
	u, err := url.Parse(d.DefaultBaseURL)
	if err != nil || !loopbackHost(u.Hostname()) || u.Port() == "" {
		return "", false
	}
	return u.Port(), true
}

// serverLabel names a found server in the menu: its dialect's name, or
// "Chat Completions server", with its address.
func serverLabel(s chatcompat.Server, host string) string {
	if s.Label == host {
		return fmt.Sprintf("Chat Completions server (%s)", host)
	}
	return fmt.Sprintf("%s (%s)", s.Label, host)
}

// serverNotes is a found server's menu detail.
func serverNotes(s chatcompat.Server) string {
	switch {
	case s.NeedsKey:
		return "running on this machine, wants a key"
	case len(s.Models) == 1:
		return "running on this machine, 1 model"
	}
	return fmt.Sprintf("running on this machine, %d models", len(s.Models))
}
