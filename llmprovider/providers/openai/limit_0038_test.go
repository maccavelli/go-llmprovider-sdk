package openai

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0038-MADR D4 and A3: ListModels returns the full list with
// catalog.WithLimit(0), probing at most catalog.MaxListed, and a ChatGPT
// session's listing is cut by the limit.

// nineServed is an API-key listing longer than six, every id usable.
var nineServed = append(catalog.Static(llmprovider.ProviderOpenAI), "gpt-5.6-sol", "gpt-6-sol", "gpt-4.1")

// TestListModels_FullListProbesSix: nine served ids come back, and at most
// six probes are sent. Every probe fails here, so the listing is kept whole.
func TestListModels_FullListProbesSix(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"` + strings.Join(nineServed, `"},{"id":"`) + `"}]}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4.1-mini"), catalog.WithLimit(0))...)
	models, err := list(t, p)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != len(nineServed) || posts.Load() != int32(catalog.MaxListed) {
		t.Errorf("ListModels = %d ids after %d probes; want %d after %d: %v", len(models), posts.Load(),
			len(nineServed), catalog.MaxListed, models)
	}
}

// TestListModels_ChatGPTWithLimit: a ChatGPT session's listing, never cut
// before 0038, is cut when the caller sets a limit.
func TestListModels_ChatGPTWithLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[
			{"slug":"gpt-6-astra","visibility":"list","priority":1,"supported_in_api":true},
			{"slug":"gpt-5.6-sol","visibility":"list","priority":4,"supported_in_api":true},
			{"slug":"gpt-5.6-luna","visibility":"list","priority":8,"supported_in_api":true}
		]}`))
	}))
	t.Cleanup(srv.Close)
	session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)}
	for _, tc := range []struct {
		opts []llmprovider.Option
		want []string
	}{
		{[]llmprovider.Option{catalog.WithLimit(2)}, []string{"gpt-6-astra", "gpt-5.6-sol"}},
		{nil, []string{"gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-luna"}},
	} {
		p := sessionProvider(t, session, "gpt-6-astra", append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL)}, tc.opts...)...)
		if models, err := list(t, p); err != nil || !slices.Equal(models, tc.want) {
			t.Errorf("%d options: ListModels = %v/%v; want %v", len(tc.opts), models, err, tc.want)
		}
	}
}
