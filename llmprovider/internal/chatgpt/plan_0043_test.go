package chatgpt

import (
	"context"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestIsPlanSession_0043: a plan session is a plan session and not a ChatGPT
// session; nothing else is a plan session (0043-MADR D1).
func TestIsPlanSession_0043(t *testing.T) {
	plan := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		Resource: "https://api.openai.com/v1"}
	if !IsPlanSession(plan) || IsSession(plan) {
		t.Errorf("a plan session: IsPlanSession %v, IsSession %v; want true, false", IsPlanSession(plan), IsSession(plan))
	}
	for name, src := range map[string]llmprovider.TokenSource{
		"a ChatGPT session": &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer},
		"the Codex CLI":     &auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI},
		"a Grok resource":   &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Resource: "https://api.x.ai"},
		"a key":             llmprovider.NewStaticToken("k"),
		"nil":               nil,
	} {
		if IsPlanSession(src) {
			t.Errorf("%s: IsPlanSession true", name)
		}
	}
}

// TestPlanModelsRequest_0043: GET /models with the bearer token, and none
// of the Codex backend's headers or its client_version (0043-MADR D5).
func TestPlanModelsRequest_0043(t *testing.T) {
	req, err := PlanModelsRequest(context.Background(), "http://example.test/v1/", "agent/1", llmprovider.Token{Value: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodGet || req.URL.String() != "http://example.test/v1/models" {
		t.Errorf("%s %s; want GET http://example.test/v1/models", req.Method, req.URL)
	}
	for header, want := range map[string]string{
		"User-Agent": "agent/1", "Authorization": "Bearer t", "Accept": "application/json",
		HeaderOriginator: "", HeaderAccount: "", HeaderFedRAMP: "",
	} {
		if got := req.Header.Get(header); got != want {
			t.Errorf("%s: %q; want %q", header, got, want)
		}
	}
	if _, err := PlanModelsRequest(context.Background(), "http://[::1", "", llmprovider.Token{}); err == nil {
		t.Error("a base URL that does not parse: no error")
	}
}
