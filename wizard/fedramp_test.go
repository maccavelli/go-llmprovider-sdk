package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestConfigureLLM_KeepsFedRAMP: a kept ChatGPT session keeps its FedRAMP
// flag in the Result, which consumers persist and hand back as Existing.
func TestConfigureLLM_KeepsFedRAMP(t *testing.T) {
	store := newMemoryTokenStore()
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderOpenAI)},
		confirms: []bool{true},
		inputs:   []string{acceptDefault},
	})
	res, err := ConfigureLLM(context.Background(), f, Options{
		Existing: storedExisting(t, store, Result{
			Provider:    llmprovider.ProviderOpenAI,
			Kind:        CredOAuth,
			TokenExpiry: time.Now().Add(time.Hour),
			Issuer:      auth.DefaultOpenAIIssuer,
			ClientID:    auth.DefaultOpenAIClientID,
			AccountID:   "acct_test",
			FedRAMP:     true,
			Model:       "kept-chatgpt-model",
		}, "existing-access-abcd", "existing-refresh"),
		TokenStore: store,
	})
	if err != nil {
		t.Fatalf("ConfigureLLM() error = %v", err)
	}
	if !res.FedRAMP {
		t.Fatal("Result.FedRAMP = false, want the kept session's true")
	}
}
