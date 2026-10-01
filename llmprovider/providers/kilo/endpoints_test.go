package kilo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from llmprovider's kilo_endpoints_test.go, kilo_organization_test.go
// and kilo_data_collection*_test.go (0015-PLAN S7). DiscoverModels is
// ListModels.

// gatewayURL is Kilo Gateway's default generation base.
const gatewayURL = "https://api.kilo.ai/api/gateway"

// kiloCall is one request a Kilo provider made.
type kiloCall struct {
	method, url, auth, org string
}

// absentHeader marks a header the request did not carry.
const absentHeader = "<absent>"

// kiloCalls runs one Generate and one ListModels through a recording
// transport that answers without a network, and returns the requests made.
func kiloCalls(t *testing.T, token string, opts ...llmprovider.Option) []kiloCall {
	t.Helper()
	var (
		mu    sync.Mutex
		calls []kiloCall
	)
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		org := absentHeader
		if v, ok := r.Header[http.CanonicalHeaderKey("X-KILOCODE-ORGANIZATIONID")]; ok {
			org = strings.Join(v, ",")
		}
		mu.Lock()
		calls = append(calls, kiloCall{r.Method, r.URL.String(), r.Header.Get("Authorization"), org})
		mu.Unlock()
		body := `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`
		if r.Method == http.MethodGet {
			body = `{"data":[{"id":"deepseek/deepseek-v4.1-flash","architecture":{"input_modalities":["text"],` +
				`"output_modalities":["text"]},"supported_parameters":["tools"],"pricing":{"completion":"0.1"}}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	p := build(t, append([]llmprovider.Option{llmprovider.WithAPIKey(token), llmprovider.WithModel("deepseek/deepseek-v4.1-flash"),
		llmprovider.WithHTTPClient(client)}, opts...)...)
	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	return slices.Clone(calls)
}

func wantKiloCalls(t *testing.T, got []kiloCall, want ...kiloCall) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("requests:\n got  %+v\n want %+v", got, want)
	}
}

// TestKilo_URLTokenSelectsBase: a token "{url}:{secret}" sends generation and
// the listing to {origin}{prefix}/api/gateway (url.ts route()), with the whole
// token as the bearer.
func TestKilo_URLTokenSelectsBase(t *testing.T) {
	for _, tc := range []struct{ token, base string }{
		{"https://kilo.example.test/tenant:secret", "https://kilo.example.test/tenant/api/gateway"},
		{"http://127.0.0.1:8080:secret", "http://127.0.0.1:8080/api/gateway"},
		{"https://kilo.example.test/tenant/api/openrouter/:secret", "https://kilo.example.test/tenant/api/gateway"},
	} {
		t.Run(tc.token, func(t *testing.T) {
			wantKiloCalls(t, kiloCalls(t, tc.token),
				kiloCall{http.MethodPost, tc.base + "/chat/completions", "Bearer " + tc.token, absentHeader},
				kiloCall{http.MethodGet, tc.base + "/models", "Bearer " + tc.token, absentHeader})
		})
	}
}

// TestKilo_URLTokenOrganization: an /api/organizations/{id} token path scopes
// both requests to that organization and lists the organization's models
// (models.ts:219-229).
func TestKilo_URLTokenOrganization(t *testing.T) {
	const token = "https://kilo.example.test/api/organizations/org-9:secret"
	wantKiloCalls(t, kiloCalls(t, token),
		kiloCall{http.MethodPost, "https://kilo.example.test/api/gateway/chat/completions", "Bearer " + token, "org-9"},
		kiloCall{http.MethodGet, "https://kilo.example.test/api/organizations/org-9/models", "Bearer " + token, "org-9"})
}

// TestKilo_PlainTokenUsesDefaults: an ordinary key keeps the default gateway
// and sends no organization, even when a URL appears after its start.
func TestKilo_PlainTokenUsesDefaults(t *testing.T) {
	const token = "sk-plain:https://elsewhere.test/p:q"
	wantKiloCalls(t, kiloCalls(t, token),
		kiloCall{http.MethodPost, gatewayURL + "/chat/completions", "Bearer " + token, absentHeader},
		kiloCall{http.MethodGet, gatewayURL + "/models", "Bearer " + token, absentHeader})
}

// TestKilo_OrganizationOption: WithOrganization, which was WithKiloOrganization,
// scopes generation with X-KILOCODE-ORGANIZATIONID and lists
// /api/organizations/{id}/models.
func TestKilo_OrganizationOption(t *testing.T) {
	wantKiloCalls(t, kiloCalls(t, "sk-plain", WithOrganization("org-1")),
		kiloCall{http.MethodPost, gatewayURL + "/chat/completions", "Bearer sk-plain", "org-1"},
		kiloCall{http.MethodGet, "https://api.kilo.ai/api/organizations/org-1/models", "Bearer sk-plain", "org-1"})
}

// TestKilo_RotatingTokenPicksItsBase: the endpoints follow the token each
// request reads, so a source that changes its URL-prefixed token moves the
// requests (new with WithTokenSource).
func TestKilo_RotatingTokenPicksItsBase(t *testing.T) {
	var urls []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		urls = append(urls, r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)), Request: r}, nil
	})}
	src := &tokenSequence{values: []string{"https://a.example.test:s1", "https://b.example.test:s2"}}
	p := build(t, llmprovider.WithTokenSource(src), llmprovider.WithModel("m"), llmprovider.WithHTTPClient(client))
	for range 2 {
		if _, err := p.Generate(context.Background(), text("hi")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
	}
	if !slices.Equal(urls, []string{"https://a.example.test/api/gateway/chat/completions", "https://b.example.test/api/gateway/chat/completions"}) {
		t.Fatalf("requests = %v, want one per token's base", urls)
	}
}

// tokenSequence returns its values in turn.
type tokenSequence struct {
	mu     sync.Mutex
	values []string
	n      int
}

func (s *tokenSequence) Token(context.Context) (llmprovider.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.values[s.n%len(s.values)]
	s.n++
	return llmprovider.Token{Value: v, Type: llmprovider.TokenAPIKey}, nil
}

// kiloBody runs one Generate against a stub gateway and returns the request
// body it sent.
func kiloBody(t *testing.T, opts ...llmprovider.Option) map[string]any {
	t.Helper()
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	t.Cleanup(srv.Close)
	if _, err := build(t, apiKey(srv.URL, "deepseek/deepseek-v4.1-flash", opts...)...).Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return body
}

// TestKilo_DeniesDataCollectionByDefault: Kilo's opt-out from upstreams that
// train on prompts is sent by default, matching the listing's policy.
func TestKilo_DeniesDataCollectionByDefault(t *testing.T) {
	provider, _ := kiloBody(t)["provider"].(map[string]any)
	if provider["data_collection"] != "deny" {
		t.Fatalf("provider = %v, want data_collection deny", provider)
	}
}

// TestKilo_DataCollectionAllowed: WithDataCollection(true), which was
// WithKiloDataCollection(true), sends no provider preference, as Kilo's client
// does without its privacy setting.
func TestKilo_DataCollectionAllowed(t *testing.T) {
	if provider, ok := kiloBody(t, WithDataCollection(true))["provider"]; ok {
		t.Fatalf("provider = %v, want absent", provider)
	}
}
