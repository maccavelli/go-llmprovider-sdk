package huggingface

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

// TestHuggingFace_TokenHeaderOverride is 0016-PLAN T3 step 1 (R16; 0016-MADR
// D2, A6): a token with no Header goes in Authorization as a bearer token; one
// naming a Header goes there instead, "Bearer "-prefixed only for a
// TokenBearer, in generation and in the listing alike.
func TestHuggingFace_TokenHeaderOverride(t *testing.T) {
	for _, tc := range []struct {
		name   string
		token  llmprovider.Token
		header string
		want   string
	}{
		{"no header", llmprovider.Token{Value: "v", Type: llmprovider.TokenAPIKey}, headerAuthorization, "Bearer v"},
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
					_, _ = w.Write([]byte(hfListing))
					return
				}
				_, _ = w.Write([]byte(fxHFChat))
			}))
			t.Cleanup(srv.Close)
			p := build(t, llmprovider.WithTokenSource(headerSource(tc.token)), llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL))
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
				case tc.header != headerAuthorization && h.Get(headerAuthorization) != "":
					t.Errorf("%s: Authorization = %q, want none beside the override", method, h.Get(headerAuthorization))
				}
			}
		})
	}
}
