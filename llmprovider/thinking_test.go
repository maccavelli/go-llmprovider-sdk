package llmprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captureServer returns an httptest server that records the decoded JSON request body of
// the last call and replies with the given canned response body.
func captureServer(t *testing.T, lastBody *map[string]any, respBody string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		m := map[string]any{}
		_ = json.Unmarshal(raw, &m)
		*lastBody = m
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, respBody)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestGoogleRouteThinking_RequestBody verifies a thinkingConfig is nested in
// generationConfig with the configured budget on OpenCode's google route, and
// that the default path omits it. The gemini provider has no budget (MADR 0014).
func TestGoogleRouteThinking_RequestBody(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`)

	p, _ := NewOpencode(ProviderOpencodeZen, "k", "gemini-x", WithBaseURL(srv.URL), WithThinkingBudget(1234))
	if _, err := p.GenerateThinking(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	gc, ok := body["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing generationConfig: %v", body)
	}
	tc, ok := gc["thinkingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("missing thinkingConfig: %v", gc)
	}
	if b := tc["thinkingBudget"].(float64); int(b) != 1234 {
		t.Errorf("thinkingBudget = %v, want 1234", b)
	}

	// Default path: no thinkingConfig.
	body = nil
	if _, err := p.Generate(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	gc2 := body["generationConfig"].(map[string]any)
	if _, present := gc2["thinkingConfig"]; present {
		t.Errorf("non-thinking request must not contain thinkingConfig: %v", gc2)
	}
}

// TestGoogleRouteThinking_DynamicBudgetDefault verifies an unset budget maps to
// -1 (dynamic) on OpenCode's google route.
func TestGoogleRouteThinking_DynamicBudgetDefault(t *testing.T) {
	var body map[string]any
	srv := captureServer(t, &body, `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`)

	p, _ := NewOpencode(ProviderOpencodeZen, "k", "gemini-x", WithBaseURL(srv.URL))
	if _, err := p.GenerateThinking(context.Background(), "hi"); err != nil {
		t.Fatal(err)
	}
	gc := body["generationConfig"].(map[string]any)
	tc := gc["thinkingConfig"].(map[string]any)
	if b := tc["thinkingBudget"].(float64); int(b) != dynamicGeminiThinkingBudget {
		t.Errorf("thinkingBudget = %v, want %d (dynamic)", b, dynamicGeminiThinkingBudget)
	}
}
