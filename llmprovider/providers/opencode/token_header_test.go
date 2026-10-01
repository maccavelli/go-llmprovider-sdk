package opencode

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

// TestOpencode_TokenHeaderOverride is 0016-PLAN T3 step 1 (R16; 0016-MADR D2,
// A6) on every route: a token with no Header goes in the route's own header
// and scheme; one naming a Header goes there instead, "Bearer "-prefixed only
// for a TokenBearer, in generation and in the listing alike.
func TestOpencode_TokenHeaderOverride(t *testing.T) {
	own := map[Route][2]string{
		RouteResponses:       {"Authorization", "Bearer v"},
		RouteMessages:        {"x-api-key", "v"},
		RouteGoogle:          {"x-goog-api-key", "v"},
		RouteChatCompletions: {"Authorization", "Bearer v"},
	}
	for _, route := range routes {
		for _, tc := range []struct {
			name         string
			token        llmprovider.Token
			header, want string
		}{
			{"no header", llmprovider.Token{Value: "v", Type: llmprovider.TokenAPIKey}, own[route][0], own[route][1]},
			{"override, bearer", llmprovider.Token{Value: "v", Type: llmprovider.TokenBearer, Header: "X-Custom"}, "X-Custom", "Bearer v"},
			{"override, api key", llmprovider.Token{Value: "v", Type: llmprovider.TokenAPIKey, Header: "X-Custom"}, "X-Custom", "v"},
		} {
			t.Run(string(route)+"/"+tc.name, func(t *testing.T) {
				var mu sync.Mutex
				seen := map[string]http.Header{}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					seen[r.Method] = r.Header.Clone()
					mu.Unlock()
					_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.3-flash"}]}`))
				}))
				t.Cleanup(srv.Close)
				p := build(t, llmprovider.ProviderOpencodeZen, llmprovider.WithTokenSource(headerSource(tc.token)),
					llmprovider.WithModel("glm-5.3-flash"), llmprovider.WithBaseURL(srv.URL), WithRoute(route))
				_, _ = p.Generate(context.Background(), text("hi")) // the reply's shape does not matter here
				if _, err := list(t, p); err != nil {
					t.Fatalf("ListModels: %v", err)
				}
				mu.Lock()
				defer mu.Unlock()
				checks := map[string][2]string{http.MethodPost: {tc.header, tc.want}}
				// The listing is GET /models with Authorization, whatever the route.
				if tc.token.Header == "" {
					checks[http.MethodGet] = [2]string{"Authorization", "Bearer v"}
				} else {
					checks[http.MethodGet] = [2]string{tc.header, tc.want}
				}
				for method, want := range checks {
					h, ok := seen[method]
					switch {
					case !ok:
						t.Errorf("%s: no request", method)
					case h.Get(want[0]) != want[1]:
						t.Errorf("%s: %s = %q, want %q", method, want[0], h.Get(want[0]), want[1])
					case tc.token.Header != "" && (h.Get("Authorization") != "" || h.Get("x-api-key") != "" || h.Get("x-goog-api-key") != ""):
						t.Errorf("%s: a route header was sent beside the override: %v", method, h)
					}
				}
			})
		}
	}
}
