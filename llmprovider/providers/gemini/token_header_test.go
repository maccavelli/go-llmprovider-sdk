package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// headerSource returns one fixed Token, as a caller's own source would.
type headerSource llmprovider.Token

func (s headerSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token(s), nil
}

// TestGemini_TokenHeaderOverride is 0016-PLAN T3 step 1 (R16; 0016-MADR D2,
// A6): a token with no Header goes in x-goog-api-key, bare; one naming a
// Header goes there instead, "Bearer "-prefixed only for a TokenBearer, in
// generation and in the listing alike.
func TestGemini_TokenHeaderOverride(t *testing.T) {
	for _, tc := range []struct {
		name   string
		token  llmprovider.Token
		header string
		want   string
	}{
		{"no header", llmprovider.Token{Value: "v", Type: llmprovider.TokenAPIKey}, googleKeyHeader, "v"},
		{"override, bearer", llmprovider.Token{Value: "v", Type: llmprovider.TokenBearer, Header: "X-Custom"}, "X-Custom", "Bearer v"},
		{"override, api key", llmprovider.Token{Value: "v", Type: llmprovider.TokenAPIKey, Header: "X-Custom"}, "X-Custom", "v"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			seen := map[string]http.Header{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				seen[r.Method] = r.Header.Clone()
				mu.Unlock()
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(geminiListing))
					return
				}
				_, _ = w.Write([]byte(interactionText))
			}))
			t.Cleanup(srv.Close)
			p := build(t, llmprovider.WithTokenSource(headerSource(tc.token)), llmprovider.WithModel("gemini-3.7-flash"),
				llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false))
			if _, err := p.Generate(context.Background(), text("hi")); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if _, err := list(t, p); err != nil {
				t.Fatalf("ListModels: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			for _, method := range []string{http.MethodPost, http.MethodGet} {
				h, ok := seen[method]
				switch {
				case !ok:
					t.Errorf("%s: no request", method)
				case h.Get(tc.header) != tc.want:
					t.Errorf("%s: %s = %q, want %q", method, tc.header, h.Get(tc.header), tc.want)
				case tc.header != googleKeyHeader && h.Get(googleKeyHeader) != "":
					t.Errorf("%s: x-goog-api-key = %q, want none beside the override", method, h.Get(googleKeyHeader))
				}
			}
		})
	}
}
