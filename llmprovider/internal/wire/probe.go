package wire

import (
	"context"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// ProbeHealth probes a listing's models through the process's probe cache,
// keyed by provider, base URL and the fingerprint of the token src gives
// (0028-MADR D-A3). A token that cannot be read is probed for, uncached.
func ProbeHealth(
	ctx context.Context,
	provider llmprovider.ProviderID,
	baseURL string,
	src llmprovider.TokenSource,
	preferred []string,
	limit int,
	generate func(ctx context.Context, modelID string) (string, error),
) []string {
	token, err := src.Token(ctx)
	if err != nil {
		return transport.ProbeGenerateHealth(ctx, preferred, limit, generate)
	}
	key := transport.ProbeKey{Provider: string(provider), BaseURL: baseURL, Credential: transport.Fingerprint(token.Value)}
	return transport.ProbeGenerateHealthCached(ctx, key, preferred, limit, generate)
}
