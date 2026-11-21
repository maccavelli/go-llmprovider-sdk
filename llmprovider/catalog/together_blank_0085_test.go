package catalog

// 0085-MADR ST82 (Phase 1b): Together's listing, like a Chat Completions
// server's, sends no Authorization when the token's value is "". The listing
// is answered in process by a RoundTripper; the key is blank or made up.

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// togetherAuth0085 lists Together with src and returns the Authorization of
// each request the listing made ("" when it carried none), and the headers of
// the first, for a failure message.
func togetherAuth0085(t *testing.T, src llmprovider.TokenSource) (auths []string, first http.Header) {
	t.Helper()
	var mu sync.Mutex
	client := &http.Client{Transport: rt0085(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		auths = append(auths, r.Header.Get("Authorization"))
		if first == nil {
			first = r.Header.Clone()
		}
		return &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
	})}
	_, _ = List(context.Background(), llmprovider.ProviderTogether, src,
		llmprovider.WithHTTPClient(client), WithCache(&Cache{}), llmprovider.WithoutModelMetadata())
	mu.Lock()
	defer mu.Unlock()
	return auths, first
}

type rt0085 func(*http.Request) (*http.Response, error)

func (f rt0085) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestTogetherListingEmptyTokenSendsNoAuthorization_0085 (ST82): a token whose
// value is "", built with NewStaticToken from "" or from a blank, sends the
// Together listing no Authorization; a token that is not empty is sent as
// before.
func TestTogetherListingEmptyTokenSendsNoAuthorization_0085(t *testing.T) {
	for name, key := range map[string]string{"empty": "", "spaces": "   ", "tab": "\t", "nbsp": "\u00a0"} {
		t.Run(name, func(t *testing.T) {
			auths, first := togetherAuth0085(t, llmprovider.NewStaticToken(key))
			if len(auths) == 0 {
				t.Fatal("the listing made no request")
			}
			for _, a := range auths {
				if a != "" {
					t.Errorf("Authorization %q on a listing with an empty token; want none (headers %v)", a, first)
				}
			}
		})
	}
	t.Run("a key is sent as given", func(t *testing.T) {
		auths, _ := togetherAuth0085(t, llmprovider.NewStaticToken("made-up-key-0085"))
		if len(auths) == 0 || auths[0] != "Bearer made-up-key-0085" {
			t.Errorf("Authorization %q; want \"Bearer made-up-key-0085\"", auths)
		}
	})
}
