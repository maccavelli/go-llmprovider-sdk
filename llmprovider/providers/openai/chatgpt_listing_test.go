package openai

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestListModels_ChatGPTListsCodexCatalog was catalog's
// TestListAvailableModelsWithSource_ChatGPTListsCodexCatalog, with the live,
// one-request half of TestListModelCatalogWithSource_ChatGPTListsCodexCatalog
// (0015-PLAN S8c step 1): a ChatGPT session lists the Codex catalog by
// priority, without hidden models, with the session's headers; each call
// returns a fresh slice.
func TestListModels_ChatGPTListsCodexCatalog(t *testing.T) {
	var captured []*http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = append(captured, r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[
			{"slug":"gpt-5.6-sol","visibility":"list","priority":4,"supported_in_api":true},
			{"slug":"gpt-reserve","visibility":"hide","priority":3,"supported_in_api":true},
			{"slug":"gpt-5.6-luna","visibility":"list","priority":8,"supported_in_api":true},
			{"slug":"gpt-6-astra","visibility":"list","priority":1,"supported_in_api":true},
			{"slug":"hidden-review","visibility":"hide","priority":43,"supported_in_api":true}
		]}`))
	}))
	t.Cleanup(srv.Close)

	session := &auth.OAuthSession{
		Issuer:    auth.DefaultOpenAIIssuer,
		Access:    "session-access",
		Expiry:    time.Now().Add(time.Hour),
		AccountID: "acct_live",
		FedRAMP:   true,
	}
	p := sessionProvider(t, session, "gpt-6-astra", llmprovider.WithHTTPClient(srv.Client()), llmprovider.WithBaseURL(srv.URL))
	models, err := list(t, p)
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	want := []string{"gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-luna"}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("models = %v, want %v", models, want)
	}
	if len(captured) != 1 {
		t.Fatalf("requests = %d, want one Codex /models listing", len(captured))
	}
	r := captured[0]
	if r.URL.Path != "/models" || r.URL.Query().Get("client_version") != chatgptModelsClientVersion {
		t.Fatalf("request URL = %s", r.URL.Redacted())
	}
	for header, value := range map[string]string{
		"Authorization":      "Bearer session-access",
		headerOriginator:     originatorValue,
		headerChatGPTAccount: "acct_live",
		headerFedRAMP:        "true",
	} {
		if got := r.Header.Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
	models[0] = "mutated"
	again, err := list(t, p)
	if err != nil {
		t.Fatalf("second ListModels() error = %v", err)
	}
	if again[0] == "mutated" {
		t.Fatal("ListModels returned a live slice")
	}
}

// TestListModels_ChatGPTListingBounded (MADR 0013 A5): the ChatGPT listing
// runs under the 10 s bound catalog applies to every other listing.
func TestListModels_ChatGPTListingBounded(t *testing.T) {
	var left time.Duration = -1
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if dl, ok := r.Context().Deadline(); ok {
			left = time.Until(dl)
		}
		return httpResponse(r, http.StatusOK, `{"models":[{"slug":"gpt-6-astra","visibility":"list"}]}`), nil
	})}
	session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)}
	if _, err := list(t, sessionProvider(t, session, "gpt-6-astra", llmprovider.WithHTTPClient(client))); err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if left < 0 || left > 10*time.Second {
		t.Fatalf("the listing ran with %v left, want a deadline within 10s", left)
	}
}
