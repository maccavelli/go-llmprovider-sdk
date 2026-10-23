package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/catalogtool"
)

// 0033-PLAN-refresh-the-curated-catalogs.md Phase 6: the command reads the
// live guard's public listings, and records a snapshot (0033-MADR's amendment
// "the second refresh, and what the command reads", D4a and D4c as refined).

// rankedBodies are a canned listing body for each ranked list.
func rankedBodies() map[llmprovider.ProviderID][]byte {
	out := map[llmprovider.ProviderID][]byte{}
	for p := range snapshotFiles {
		out[p] = []byte(`{"data":[{"id":"` + string(p) + `-model"}]}`)
	}
	return out
}

// allKeys is a distinct key for every provider.
func allKeys() map[llmprovider.ProviderID]string {
	keys := map[llmprovider.ProviderID]string{}
	for p := range llmprovider.ProviderEnvVars() {
		keys[p] = "key-" + string(p)
	}
	return keys
}

// TestRun_PublicListings: the live guard's four public gateways are listed
// with no key, and the others with their own.
func TestRun_PublicListings(t *testing.T) {
	f := &fakeFetcher{}
	var out, errOut bytes.Buffer
	if rc := run(nil, &out, &errOut, allKeys(), f); rc != 0 {
		t.Fatalf("run = %d; stderr %s", rc, errOut.String())
	}
	for _, p := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo,
		llmprovider.ProviderKilo, llmprovider.ProviderHuggingFace} {
		if key, ok := f.asked[p]; !ok || key != "" {
			t.Errorf("%s was listed with key %q (asked %t), want the public listing", p, key, ok)
		}
	}
	for _, p := range []llmprovider.ProviderID{llmprovider.ProviderTogether, llmprovider.ProviderClaude} {
		if key := f.asked[p]; key != "key-"+string(p) {
			t.Errorf("%s was listed with key %q, want its own", p, key)
		}
	}
	if !f.ranked[llmprovider.ProviderTogether] || f.ranked[llmprovider.ProviderClaude] {
		t.Errorf("ranked requests %v, want Together ranked and Claude not", f.ranked)
	}
}

// TestRun_Snapshot: -snapshot writes each ranked listing as it was read,
// Metadata.Encode's output, and a manifest of their hashes.
func TestRun_Snapshot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ranking-test")
	f := &fakeFetcher{bodies: rankedBodies()}
	var out, errOut bytes.Buffer
	if rc := run([]string{"-snapshot", dir}, &out, &errOut, allKeys(), f); rc != 0 {
		t.Fatalf("run -snapshot = %d; stderr %s", rc, errOut.String())
	}
	var manifest struct {
		RecordedAt string            `json:"recorded_at"`
		Files      map[string]string `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := time.Parse(time.RFC3339, manifest.RecordedAt); err != nil {
		t.Errorf("recorded_at %q: %v", manifest.RecordedAt, err)
	}
	want := map[string][]byte{"metadata.json": nil}
	for p, name := range snapshotFiles {
		want[name] = f.bodies[p]
	}
	for name, body := range want {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if body != nil && !bytes.Equal(got, body) {
			t.Errorf("%s = %q, want the recorded body %q", name, got, body)
		}
		sum := sha256.Sum256(got)
		if manifest.Files[name] != hex.EncodeToString(sum[:]) {
			t.Errorf("%s: manifest hash %q, want %x", name, manifest.Files[name], sum)
		}
	}
	if len(manifest.Files) != len(want) {
		t.Errorf("manifest names %d files, want %d", len(manifest.Files), len(want))
	}
	if !strings.Contains(out.String(), "snapshot: ") {
		t.Errorf("stdout does not report the snapshot:\n%s", out.String())
	}
}

// TestRun_SnapshotRefusesAnExistingDir: a directory that exists is never
// written into, and is refused before any listing is read.
func TestRun_SnapshotRefusesAnExistingDir(t *testing.T) {
	dir := t.TempDir()
	f := &fakeFetcher{bodies: rankedBodies()}
	var out, errOut bytes.Buffer
	if rc := run([]string{"-snapshot", dir}, &out, &errOut, allKeys(), f); rc != 2 {
		t.Fatalf("run -snapshot into an existing directory = %d, want 2", rc)
	}
	if len(f.asked) != 0 {
		t.Errorf("%d listing(s) were read before the directory was refused", len(f.asked))
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("the existing directory gained %d entries", len(entries))
	}
	if _, err := os.Stat(dir + ".partial"); !os.IsNotExist(err) {
		t.Errorf("a partial snapshot was left beside it: %v", err)
	}
}

// TestRun_SnapshotNeedsEveryRankedListing: a snapshot missing a ranked
// listing is a tool error, and nothing is written.
func TestRun_SnapshotNeedsEveryRankedListing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ranking-test")
	bodies := rankedBodies()
	delete(bodies, llmprovider.ProviderTogether)
	var out, errOut bytes.Buffer
	if rc := run([]string{"-snapshot", dir}, &out, &errOut, allKeys(), &fakeFetcher{bodies: bodies}); rc != 2 {
		t.Fatalf("run -snapshot without Together's listing = %d, want 2", rc)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the snapshot directory exists: %v", err)
	}
}

// TestSnapshotFiles: every ranked list has a snapshot file, and nothing else.
func TestSnapshotFiles(t *testing.T) {
	for p := range llmprovider.ProviderEnvVars() {
		if _, ok := snapshotFiles[p]; ok != catalogtool.Ranked(p) {
			t.Errorf("%s: snapshot file %t, ranked %t", p, ok, catalogtool.Ranked(p))
		}
	}
}

// TestListingRecorder: the recorder keeps the one GET whose path ends in
// /models, and refuses none or two.
func TestListingRecorder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("body of " + r.URL.Path))
	}))
	defer srv.Close()
	get := func(rec *listingRecorder, path string) {
		t.Helper()
		resp, err := (&http.Client{Transport: rec}).Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	rec := newListingRecorder()
	get(rec, "/api.json")
	if _, err := rec.listing(); err == nil {
		t.Error("no listing GET: listing() gave no error")
	}
	get(rec, "/v1/models")
	if body, err := rec.listing(); err != nil || string(body) != "body of /v1/models" {
		t.Errorf("listing() = %q, %v; want the /v1/models body", body, err)
	}
	get(rec, "/v1/models")
	if _, err := rec.listing(); err == nil {
		t.Error("two listing GETs: listing() gave no error")
	}
}

// TestLiveFetcher: the live fetcher reads the document and a listing from the
// endpoints it is given, keeps the listing body it ranked, and keeps none when
// the listing fails.
func TestLiveFetcher(t *testing.T) {
	listing := `{"object":"list","data":[{"id":"zen-a","object":"model"},{"id":"zen-b","object":"model"}]}`
	doc := `{"opencode":{"models":{"zen-a":{"name":"A","tool_call":true,"reasoning":true,` +
		`"release_date":"2026-09-01","cost":{"input":1,"output":2}}}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/fail/"):
			w.WriteHeader(http.StatusInternalServerError)
		case strings.HasSuffix(r.URL.Path, "/models"):
			_, _ = w.Write([]byte(listing))
		case strings.HasSuffix(r.URL.Path, "/api.json"):
			_, _ = w.Write([]byte(doc))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	ctx := context.Background()
	// Metadata reads into the process's cache, which takes no option of the
	// test's, so the document's URL is the run's own (0058-PLAN Phase 2).
	f := &liveFetcher{metadataURL: srv.URL + "/" + rand.Text() + "/api.json",
		extra: []llmprovider.Option{llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelMetadataURL(srv.URL + "/api.json"),
			catalog.WithCache(new(catalog.Cache))}}
	meta, err := f.Metadata(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := meta.Model(llmprovider.ProviderOpencodeZen, "zen-a"); !ok {
		t.Error("the document read has no zen-a")
	}
	cat, err := f.Catalog(ctx, llmprovider.ProviderOpencodeZen, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cat.Usable, "zen-a") || len(cat.Recommended) == 0 {
		t.Errorf("catalog %+v, want zen-a usable and a ranked list", cat)
	}
	if body, ok := f.Listing(llmprovider.ProviderOpencodeZen); !ok || string(body) != listing {
		t.Errorf("Listing = %q, %t; want the listing as served", body, ok)
	}
	if _, ok := f.Listing(llmprovider.ProviderKilo); ok {
		t.Error("Listing(kilo) is set, though Kilo was never read")
	}
	failing := &liveFetcher{extra: []llmprovider.Option{llmprovider.WithBaseURL(srv.URL + "/fail")}}
	if _, err := failing.Catalog(ctx, llmprovider.ProviderOpencodeZen, "", false); err == nil {
		t.Error("a failing listing gave no error")
	}
	if _, ok := failing.Listing(llmprovider.ProviderOpencodeZen); ok {
		t.Error("a failing listing's body was kept")
	}
}
