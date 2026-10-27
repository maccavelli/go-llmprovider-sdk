package wire

import (
	"context"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// ProbeHealth probes a listing's models through probes, the process's store
// when probes is nil (0048-MADR D1), keyed by provider, base URL, scope
// and the fingerprint of the token src gives (0028-MADR D-A3). scope is
// Anthropic's workspace for the Claude provider, and empty for every other
// (0064-MADR, amendment "the probe cache"). A token that cannot be read is
// probed for, uncached.
func ProbeHealth(
	ctx context.Context,
	probes *transport.ProbeStore,
	provider llmprovider.ProviderID,
	baseURL, scope string,
	src llmprovider.TokenSource,
	preferred []string,
	limit int,
	generate func(ctx context.Context, modelID string) (string, error),
) []string {
	token, err := src.Token(ctx)
	if err != nil {
		return transport.ProbeGenerateHealth(ctx, preferred, limit, generate)
	}
	key := transport.ProbeKey{Provider: string(provider), BaseURL: baseURL, Credential: transport.Fingerprint(token.Value), Scope: scope}
	return transport.ProbeGenerateHealthCached(ctx, probes, key, preferred, limit, generate)
}
