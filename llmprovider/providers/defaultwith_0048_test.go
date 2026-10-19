package providers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// foreign0048 is scoped to no built-in, so every built-in's New refuses it:
// a refusal shows that the option reached that New.
var foreign0048 = llmprovider.ScopedOption("not-a-provider", "test.foreign0048", 1)

// TestDefaultWith_ReachesEveryBuiltin_0048: DefaultWith's options reach every
// built-in provider's New (0048-MADR D1).
func TestDefaultWith_ReachesEveryBuiltin_0048(t *testing.T) {
	r := providers.DefaultWith(foreign0048)
	ds := r.Descriptors()
	if len(ds) != len(providers.Default().Descriptors()) {
		t.Fatalf("DefaultWith holds %d providers, Default %d", len(ds), len(providers.Default().Descriptors()))
	}
	for _, d := range ds {
		_, err := r.New(d.ID, llmprovider.WithAPIKey("key-0048"))
		if !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(err.Error(), "test.foreign0048") {
			t.Errorf("%s: New = %v; want the foreign option refused", d.ID, err)
		}
	}
}

// TestDefaultWith_CallerOptionWins_0048: the call's own option comes after
// DefaultWith's, so it wins: the request carries the caller's key.
func TestDefaultWith_CallerOptionWins_0048(t *testing.T) {
	var key atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key.Store(r.Header.Get("x-api-key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_0048","type":"message","role":"assistant","model":"claude-x",` +
			`"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(srv.Close)
	r := providers.DefaultWith(llmprovider.WithAPIKey("default-0048"))
	p, err := r.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("caller-0048"), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithModel("claude-x"), llmprovider.WithoutModelMetadata())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := llmprovider.GenerateText(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
	}); err != nil {
		t.Fatalf("GenerateText: %v", err)
	}
	if got := key.Load(); got != "caller-0048" {
		t.Errorf("x-api-key %v; want the caller's, caller-0048", got)
	}
}

// TestDefaultWith_CallerFactoryUntouched_0048: a provider the caller
// registers is not given DefaultWith's options.
func TestDefaultWith_CallerFactoryUntouched_0048(t *testing.T) {
	r := providers.DefaultWith(foreign0048)
	const id llmprovider.ProviderID = "custom-0048"
	given := -1
	err := r.Register(llmprovider.Descriptor{ID: id, Label: "Custom"}, func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
		given = len(opts)
		return llmtest.NewFake(id, llmprovider.Capabilities{}), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.New(id); err != nil || given != 0 {
		t.Fatalf("New(custom) = %v, given %d option(s); want no error and none", err, given)
	}
}
