package ollama

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

// TestOllama_TokenHeaderOverride is 0016-PLAN T3 step 1 (R16; 0016-MADR D2,
// A6): Ollama's own header is none, so a token with no Header is not sent; one
// naming a Header goes there, "Bearer "-prefixed only for a TokenBearer, in
// generation and in the listing alike.
func TestOllama_TokenHeaderOverride(t *testing.T) {
	for _, tc := range []struct {
		name   string
		token  llmprovider.Token
		header string
		want   string
	}{
		{"no header", llmprovider.Token{Value: "v", Type: llmprovider.TokenAPIKey}, "X-Custom", ""},
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
					_, _ = w.Write([]byte(ollamaTags("llama3.3:latest")))
					return
				}
				_, _ = w.Write([]byte(fxOllamaChat))
			}))
			t.Cleanup(srv.Close)
			p := build(t, local(srv.URL, "m", llmprovider.WithTokenSource(headerSource(tc.token)), llmprovider.WithModelProbes(false))...)
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
				case h.Get("Authorization") != "":
					t.Errorf("%s: Authorization = %q, want none", method, h.Get("Authorization"))
				}
			}
		})
	}
}
