package grok

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0041-MADR D2: Grok's efforts from the model's facts, the listing above
// models.dev, and the effort asked when there are none (0062-MADR D2, D3;
// R6 removed, amendment "Phase 2's Grok cases").

const (
	grok43Listing = `{"data":[{"id":"grok-4.3","capabilities":{"reasoning_effort":["low","medium","high","xhigh"]}}]}`
	grok43Doc     = `{"xai":{"models":{"grok-4.3":{"name":"Grok 4.3","reasoning":true,"limit":{"context":1000000,"output":64000},` +
		`"reasoning_options":[{"type":"effort","values":["none","low","medium","high"]}]}}}}`
)

// effortServer serves the listing at /models, the document at /api.json, and
// records the effort of each /responses request.
func effortServer(t *testing.T) (*httptest.Server, func() any) {
	t.Helper()
	var mu sync.Mutex
	var effort any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, grok43Listing)
		case "/api.json":
			_, _ = io.WriteString(w, grok43Doc)
		default:
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			mu.Lock()
			effort = effortOf(body)
			mu.Unlock()
			_, _ = io.WriteString(w, okResponse)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() any { mu.Lock(); defer mu.Unlock(); return effort }
}

// TestGrokEfforts_Facts_0041: grok-4.3 asked for xhigh sends xhigh when the
// cached listing gives it, high when only models.dev is cached (it lists no
// xhigh), warned, and xhigh as asked with no facts.
func TestGrokEfforts_Facts_0041(t *testing.T) {
	ask := &llmprovider.Request{Input: []llmprovider.Item{user("hi")},
		Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}}
	for _, tc := range []struct {
		name    string
		listing bool
		off     bool
		want    string
		warned  bool
	}{
		{"the listing cached", true, false, "xhigh", false},
		{"models.dev only", false, false, "high", true},
		{"no facts", false, true, "xhigh", false},
	} {
		srv, sent := effortServer(t)
		opts := apiKey(srv.URL, "grok-4.3", llmprovider.WithModelMetadataURL(srv.URL+"/api.json"),
			llmprovider.WithModelProbes(false), testCache(t))
		if tc.off {
			opts = append(opts, llmprovider.WithoutModelMetadata())
		} else {
			// The document cached, as a request finds it once fetched.
			if _, err := catalog.LookupMetadataWith(context.Background(), llmprovider.ProviderGrok, opts...); err != nil {
				t.Fatal(err)
			}
		}
		p := build(t, opts...)
		if tc.listing {
			if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		resp, err := p.Generate(context.Background(), ask)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := sent(); got != tc.want {
			t.Errorf("%s: sent effort %v; want %q", tc.name, got, tc.want)
		}
		warned := false
		for _, w := range resp.Warnings {
			warned = warned || w.Kind == llmprovider.WarningEffortDegraded
		}
		if warned != tc.warned {
			t.Errorf("%s: degraded warning %t; want %t (%v)", tc.name, warned, tc.warned, resp.Warnings)
		}
	}
}
