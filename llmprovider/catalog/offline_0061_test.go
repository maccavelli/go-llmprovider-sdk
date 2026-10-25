package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestList_RefusedDialIsOffline_0061 (0061-MADR D2): a listing whose dial is
// refused reports ErrOffline, as its error or, degraded to the static
// catalog, in Catalog.Err: OpenAI's listing, Ollama's, and ListRaw, whose
// recorder sits inside the client those listings send through.
func TestList_RefusedDialIsOffline_0061(t *testing.T) {
	token := llmprovider.NewStaticToken("k-0061")
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderOllama} {
		t.Run(string(id), func(t *testing.T) {
			cat, err := List(context.Background(), id, token, llmprovider.WithHTTPClient(nettest.Refusing()),
				llmprovider.WithBaseURL(nettest.RefusedURL), llmprovider.WithoutModelMetadata(), testCache(t))
			if !errors.Is(err, llmprovider.ErrOffline) && !errors.Is(cat.Err, llmprovider.ErrOffline) {
				t.Errorf("List: err %v, Catalog.Err %v; want ErrOffline in one", err, cat.Err)
			}
		})
	}
	t.Run("ListRaw", func(t *testing.T) {
		_, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, token,
			llmprovider.WithHTTPClient(nettest.Refusing()), llmprovider.WithBaseURL(nettest.RefusedURL), testCache(t))
		if !errors.Is(err, llmprovider.ErrOffline) {
			t.Errorf("ListRaw: %v; want ErrOffline", err)
		}
	})
}
