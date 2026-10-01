package opencode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from the OpenCode parts of llmprovider's identification_test.go,
// identification_options_test.go, keyless_test.go, api_error_message_test.go
// and interface_test.go (0015-PLAN S7).

var routes = []Route{RouteResponses, RouteMessages, RouteChatCompletions, RouteGoogle}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestOpencode_Identification: generation names this module honestly and
// vouches for no other client on any route (MADR 0012 §1.4). It was the
// opencode rows of TestIdentification_UserAgent and
// TestIdentification_NoForbiddenHeaders.
func TestOpencode_Identification(t *testing.T) {
	rec := newHeaderRecorder(t)
	_, _ = build(t, llmprovider.ProviderOpencodeGo, apiKey(rec.srv.URL, "glm-5.3-flash")...).Generate(context.Background(), text("hello"))
	for _, route := range routes {
		p := build(t, llmprovider.ProviderOpencodeZen, apiKey(rec.srv.URL, "some-model", WithRoute(route))...)
		_, _ = p.Generate(context.Background(), text("hello"))
	}
	reqs := rec.requests()
	if len(reqs) != 1+len(routes) {
		t.Fatalf("requests = %d, want %d", len(reqs), 1+len(routes))
	}
	for _, r := range reqs {
		if ua := r.header.Get("User-Agent"); !userAgentPattern.MatchString(ua) {
			t.Errorf("%s: User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", r.path, ua)
		}
		for name := range r.header {
			lower := strings.ToLower(name)
			if lower == "x-opencode-client" || lower == "x-xai-token-auth" || strings.HasPrefix(lower, "x-grok-") {
				t.Errorf("%s: forbidden header %s", r.path, name)
			}
		}
		if ua := strings.ToLower(r.header.Get("User-Agent")); strings.Contains(ua, "codex") ||
			strings.Contains(ua, "kilo-code") || strings.Contains(ua, "opencode/") {
			t.Errorf("%s: User-Agent %q impersonates a reference client", r.path, ua)
		}
	}
}

// TestOpencode_ClientInfoAndSession: WithClientInfo leads User-Agent and
// WithSessionID is the session header. It was the opencode half of
// TestIdentification_ClientInfoAndSessionOptions.
func TestOpencode_ClientInfoAndSession(t *testing.T) {
	rec := newHeaderRecorder(t)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(rec.srv.URL, "glm-5.3-flash",
		llmprovider.WithClientInfo("pcm", "1.2.3"), llmprovider.WithSessionID("s-1"))...)
	_, _ = p.Generate(context.Background(), text("hello"))
	reqs := rec.requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	if ua := reqs[0].header.Get("User-Agent"); !strings.HasPrefix(ua, "pcm/1.2.3 (") || !strings.Contains(ua, ") go-llmprovider-sdk/") {
		t.Errorf("User-Agent = %q, want pcm/1.2.3 (…) go-llmprovider-sdk/…", ua)
	}
	if s := reqs[0].header.Get(sessionHeader); s != "s-1" {
		t.Errorf("session = %q, want s-1", s)
	}
}

// TestOpencode_ListingCarriesSession: a gateway listing sends
// x-opencode-session, and one provider uses one session id for every request
// it makes (0013 B7). It was TestIdentification_OpencodeListingCarriesSession.
func TestOpencode_ListingCarriesSession(t *testing.T) {
	rec := newHeaderRecorder(t)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(rec.srv.URL, "glm-5.3-flash",
		llmprovider.WithModelMetadataURL(rec.srv.URL+"/api.json"))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	_, _ = p.Generate(context.Background(), text("hello"))
	sessions := map[string]bool{}
	for _, r := range rec.requests() {
		if strings.HasSuffix(r.path, "/api.json") {
			continue
		}
		s := r.header.Get(sessionHeader)
		if s == "" {
			t.Errorf("%s: no %s", r.path, sessionHeader)
		}
		sessions[s] = true
	}
	if len(sessions) != 1 {
		t.Errorf("the provider used %d session ids, want one: %v", len(sessions), sessions)
	}
}

// TestOpencode_KeylessSendsThePublicToken: without a key, OpenCode Zen and Go
// send their "public" token, in the header each route reads (MADR 0012
// §1.7). It was the opencode half of TestKeyless_KiloAndOpencodeSendAnonymousToken.
func TestOpencode_KeylessSendsThePublicToken(t *testing.T) {
	rec := newHeaderRecorder(t)
	for _, route := range routes {
		p := build(t, llmprovider.ProviderOpencodeZen, llmprovider.WithModel("some-model"),
			llmprovider.WithBaseURL(rec.srv.URL), WithRoute(route))
		_, _ = p.Generate(context.Background(), text("hello"))
	}
	reqs := rec.requests()
	if len(reqs) != len(routes) {
		t.Fatalf("requests = %d, want %d", len(reqs), len(routes))
	}
	want := []struct{ name, value string }{
		{"Authorization", "Bearer public"},
		{"x-api-key", "public"},
		{"Authorization", "Bearer public"},
		{"x-goog-api-key", "public"},
	}
	for i, w := range want {
		if got := reqs[i].header.Get(w.name); got != w.value {
			t.Errorf("%s: %s = %q, want %q", reqs[i].path, w.name, got, w.value)
		}
	}
}

// TestOpencode_ErrorCarriesServiceMessage: on every route, the error keeps the
// service's explanation and still matches the sentinel its status maps to
// (MADR 0012 §1.1, 0013 B3). It was the opencode rows of
// TestProviders_ErrorCarriesServiceMessage.
func TestOpencode_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	for _, route := range routes {
		t.Run(string(route), func(t *testing.T) {
			p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "some-model", WithRoute(route))...)
			_, err := llmprovider.GenerateText(context.Background(), p, text("hello"))
			if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
				t.Fatalf("error = %v, want it to carry the service's message", err)
			}
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
			}
		})
	}
}

// TestOpencode_Capabilities is what the interface assertions declared: text,
// tools, reasoning, items and listing, and no continuation.
func TestOpencode_Capabilities(t *testing.T) {
	for _, gateway := range family {
		p := build(t, gateway, llmprovider.WithModel("m"))
		if p.ID() != gateway {
			t.Errorf("ID = %q, want %q", p.ID(), gateway)
		}
		caps := p.Capabilities()
		if caps.Tools != llmprovider.Supported || caps.ForcedToolChoice != llmprovider.Supported ||
			caps.Reasoning != llmprovider.BestEffort || caps.Continuation != llmprovider.Unsupported ||
			caps.NativeStreaming != llmprovider.Unsupported {
			t.Errorf("%s: Capabilities = %+v", gateway, caps)
		}
		if _, ok := p.(llmprovider.ModelLister); !ok {
			t.Errorf("%s: the provider is not a ModelLister", gateway)
		}
	}
}
