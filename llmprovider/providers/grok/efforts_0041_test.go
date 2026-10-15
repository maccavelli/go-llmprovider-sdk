package grok

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0041-MADR D2, R6: Grok's efforts from the model's facts, the listing above
// models.dev, and effortMenu when there are none.

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
// xhigh), and xhigh by effortMenu with no facts.
func TestGrokEfforts_Facts_0041(t *testing.T) {
	ask := &llmprovider.Request{Input: []llmprovider.Item{user("hi")},
		Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortXHigh}}
	for _, tc := range []struct {
		name    string
		listing bool
		off     bool
		want    string
	}{
		{"the listing cached", true, false, "xhigh"},
		{"models.dev only", false, false, "high"},
		{"no facts", false, true, "xhigh"},
	} {
		srv, sent := effortServer(t)
		opts := apiKey(srv.URL, "grok-4.3", llmprovider.WithModelMetadataURL(srv.URL+"/api.json"),
			llmprovider.WithModelProbes(false))
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
		if _, err := p.Generate(context.Background(), ask); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := sent(); got != tc.want {
			t.Errorf("%s: sent effort %v; want %q", tc.name, got, tc.want)
		}
	}
}

// agreementSnapshotDir is the snapshot the agreement test reads: the one
// LLMPROVIDER_SNAPSHOT_DIR names, as tools/catalogcheck.py sets it, else the
// committed one (0041-MADR D7; 0041-PLAN Phase 7).
func agreementSnapshotDir() string {
	if dir := os.Getenv("LLMPROVIDER_SNAPSHOT_DIR"); dir != "" {
		return dir
	}
	return filepath.Join("..", "..", "catalog", "testdata", "ranking-2026-10-12")
}

// TestAgreement0041_GrokEfforts is D7's offline half for R6: on the committed
// snapshot, the menu from models.dev's facts equals effortMenu for every xAI
// model both answer, except the listed set.
func TestAgreement0041_GrokEfforts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(agreementSnapshotDir(), "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	meta, err := catalog.ParseMetadata(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	// Each exception, and why (0041-MADR §3).
	except := map[string]string{
		"grok-4.3":                   "xAI's listing gives xhigh, and a request accepted it; models.dev omits it",
		"grok-4.20-multi-agent-0309": "models.dev lists efforts the menu omits",
	}
	var differ []string
	for _, info := range meta.Models(llmprovider.ProviderGrok) {
		if !info.Reasoning.Supported || !info.Has(catalog.FieldReasoningEfforts) {
			continue
		}
		for _, effort := range []string{"low", "medium", "high", "xhigh"} {
			if clampReasoningEffortFor(info.ID, effort, factsEfforts(info), true) != clampReasoningEffort(info.ID, effort) {
				differ = append(differ, info.ID)
				break
			}
		}
	}
	slices.Sort(differ)
	want := []string{"grok-4.20-multi-agent-0309", "grok-4.3"}
	if !slices.Equal(differ, want) {
		t.Errorf("R6: facts and the menu differ on %v; want only %v (%v)", differ, want, except)
	}
}
