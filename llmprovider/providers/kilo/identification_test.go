package kilo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from the Kilo parts of llmprovider's identification_test.go,
// identification_options_test.go, keyless_test.go, api_error_message_test.go
// and interface_test.go (0015-PLAN S7).

// headerServer records the headers of every request and answers "ok".
func headerServer(t *testing.T) (*httptest.Server, func() []http.Header) {
	t.Helper()
	var mu sync.Mutex
	var seen []http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Clone())
		mu.Unlock()
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []http.Header {
		mu.Lock()
		defer mu.Unlock()
		return append([]http.Header(nil), seen...)
	}
}

var userAgentPattern = regexp.MustCompile(`^go-llmprovider-sdk/\S+ \(\w+; \w+\) go-llmprovider-sdk/\S+$`)

// TestKilo_Identification: generation names this module honestly, names the
// editor and the task, and vouches for no other client (MADR 0012 §1.4). It
// was the kilo rows of TestIdentification_UserAgent and
// TestIdentification_NoForbiddenHeaders, and TestIdentification_KiloHeaders.
func TestKilo_Identification(t *testing.T) {
	srv, seen := headerServer(t)
	_, _ = build(t, apiKey(srv.URL, "some/model")...).Generate(context.Background(), text("hello"))
	reqs := seen()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	h := reqs[0]
	if ua := h.Get("User-Agent"); !userAgentPattern.MatchString(ua) {
		t.Errorf("User-Agent = %q, want go-llmprovider-sdk/<version> (<os>; <arch>) go-llmprovider-sdk/<version>", ua)
	}
	if h.Get("X-KILOCODE-EDITORNAME") != "go-llmprovider-sdk" || h.Get("X-KiloCode-TaskId") == "" {
		t.Errorf("editor/task = %q/%q, want go-llmprovider-sdk and a task id", h.Get("X-KILOCODE-EDITORNAME"), h.Get("X-KiloCode-TaskId"))
	}
	for name := range h {
		lower := strings.ToLower(name)
		if lower == "x-opencode-client" || lower == "x-xai-token-auth" || strings.HasPrefix(lower, "x-grok-") {
			t.Errorf("forbidden header %s", name)
		}
	}
	if ua := strings.ToLower(h.Get("User-Agent")); strings.Contains(ua, "codex") ||
		strings.Contains(ua, "kilo-code") || strings.Contains(ua, "opencode/") {
		t.Errorf("User-Agent %q impersonates a reference client", ua)
	}
}

// TestKilo_ClientInfoAndSession: WithClientInfo leads User-Agent and names
// Kilo's editor; WithSessionID is the task id. It was the kilo half of
// TestIdentification_ClientInfoAndSessionOptions.
func TestKilo_ClientInfoAndSession(t *testing.T) {
	srv, seen := headerServer(t)
	p := build(t, apiKey(srv.URL, "some/model", llmprovider.WithClientInfo("pcm", "1.2.3"), llmprovider.WithSessionID("s-1"))...)
	_, _ = p.Generate(context.Background(), text("hello"))
	reqs := seen()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	if ua := reqs[0].Get("User-Agent"); !strings.HasPrefix(ua, "pcm/1.2.3 (") || !strings.Contains(ua, ") go-llmprovider-sdk/") {
		t.Errorf("User-Agent = %q, want pcm/1.2.3 (…) go-llmprovider-sdk/…", ua)
	}
	if h := reqs[0]; h.Get("X-KILOCODE-EDITORNAME") != "pcm" || h.Get("X-KiloCode-TaskId") != "s-1" {
		t.Errorf("editor/task = %q/%q, want pcm/s-1", h.Get("X-KILOCODE-EDITORNAME"), h.Get("X-KiloCode-TaskId"))
	}
}

// TestKilo_KeylessSendsTheAnonymousToken: without a key, Kilo's "anonymous"
// token is sent (MADR 0012 §1.7); a supplied key is sent as is. It was
// llmprovider's keyless_test.go.
func TestKilo_KeylessSendsTheAnonymousToken(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		want string
	}{
		{"no credential", nil, "Bearer anonymous"},
		{"empty key", []llmprovider.Option{llmprovider.WithAPIKey("")}, "Bearer anonymous"},
		{"a key", []llmprovider.Option{llmprovider.WithAPIKey("real-key")}, "Bearer real-key"},
	} {
		srv, seen := headerServer(t)
		p := build(t, append([]llmprovider.Option{llmprovider.WithModel("some/model"), llmprovider.WithBaseURL(srv.URL)}, tc.opts...)...)
		_, _ = p.Generate(context.Background(), text("hello"))
		if got := seen()[0].Get("Authorization"); got != tc.want {
			t.Errorf("%s: Authorization = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestKilo_ErrorCarriesServiceMessage: the error keeps the service's own
// explanation and still matches the sentinel its status maps to (MADR 0012
// §1.1, 0013 B3). It was the kilo row of TestProviders_ErrorCarriesServiceMessage.
func TestKilo_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	_, err := llmprovider.GenerateText(context.Background(), build(t, apiKey(srv.URL, "some/model")...), text("hello"))
	if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
		t.Fatalf("error = %v, want it to carry the service's message", err)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
	}
}

// TestKilo_Capabilities is what the interface assertions declared: text,
// tools, reasoning, items and listing, and no continuation.
func TestKilo_Capabilities(t *testing.T) {
	p := build(t, llmprovider.WithModel("m"))
	caps := p.Capabilities()
	if caps.Tools != llmprovider.Supported || caps.ForcedToolChoice != llmprovider.BestEffort ||
		caps.Reasoning != llmprovider.BestEffort || caps.Continuation != llmprovider.Unsupported ||
		caps.NativeStreaming != llmprovider.Unsupported {
		t.Errorf("Capabilities = %+v", caps)
	}
	if _, ok := p.(llmprovider.ModelLister); !ok {
		t.Error("the provider is not a ModelLister")
	}
}
