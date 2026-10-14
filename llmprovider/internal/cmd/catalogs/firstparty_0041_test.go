package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0041-PLAN Phase 2 step 2: -firstparty records the listings of OpenAI,
// Anthropic, Gemini and xAI, every page, redacted, with a manifest.

// Pages is the 0033 tests' fakeFetcher's one page per listing.
func (f *fakeFetcher) Pages(p llmprovider.ProviderID) ([][]byte, bool) {
	body, ok := f.bodies[p]
	if !ok {
		return nil, false
	}
	return [][]byte{body}, true
}

// pagedFetcher is a fetcher whose listings may have several pages.
type pagedFetcher struct {
	pages map[llmprovider.ProviderID][][]byte
}

func (f *pagedFetcher) Metadata(context.Context) (catalog.Metadata, error) {
	return catalog.ParseMetadata(strings.NewReader(runDoc))
}

func (f *pagedFetcher) Catalog(_ context.Context, p llmprovider.ProviderID, _ string, _ bool) (catalog.Catalog, error) {
	return catalog.Catalog{Live: true}, nil
}

func (f *pagedFetcher) Listing(p llmprovider.ProviderID) ([]byte, bool) {
	if pages := f.pages[p]; len(pages) == 1 {
		return pages[0], true
	}
	return nil, false
}

func (f *pagedFetcher) Pages(p llmprovider.ProviderID) ([][]byte, bool) {
	pages, ok := f.pages[p]
	return pages, ok
}

// firstPartyPages is a canned listing for each first-party provider: OpenAI's
// with a fine-tune and an organisation owner, Anthropic's in two pages.
func firstPartyPages() map[llmprovider.ProviderID][][]byte {
	return map[llmprovider.ProviderID][][]byte{
		llmprovider.ProviderOpenAI: {[]byte(`{"object":"list","data":[` +
			`{"id":"gpt-6-luna","created":1780000000,"owned_by":"system"},` +
			`{"id":"ft:gpt-6-luna:acme-corp::abc123","created":1780000001,"owned_by":"org-acme"},` +
			`{"id":"gpt-custom","created":1780000002,"owned_by":"user-xyz"}]}`)},
		llmprovider.ProviderClaude: {
			[]byte(`{"data":[{"id":"claude-haiku-5-5","created_at":"2026-09-01T00:00:00Z"}],"has_more":true,"last_id":"claude-haiku-5-5"}`),
			[]byte(`{"data":[{"id":"claude-opus-5-5","created_at":"2026-09-02T00:00:00Z"}],"has_more":false}`)},
		llmprovider.ProviderGemini: {[]byte(`{"models":[{"name":"models/gemini-3.8-flash","inputTokenLimit":1048576}]}`)},
		llmprovider.ProviderGrok:   {[]byte(`{"data":[{"id":"grok-4.6","created":1780000000,"owned_by":"xai"}]}`)},
	}
}

// TestFirstParty_0041: the four listings are written as arrays of their pages,
// redacted, with a manifest of their sums; an existing directory is refused;
// a listing not read writes nothing.
func TestFirstParty_0041(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "firstparty-2026-10-14")
	var out, errOut bytes.Buffer
	if code := run([]string{"-firstparty", dir}, &out, &errOut, allKeys(),
		&pagedFetcher{pages: firstPartyPages()}); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var manifest struct {
		RecordedAt string            `json:"recorded_at"`
		Files      map[string]string `json:"files"`
	}
	raw, err := os.ReadFile(filepath.Join(dir, "firstparty.json"))
	if err != nil || json.Unmarshal(raw, &manifest) != nil || manifest.RecordedAt == "" {
		t.Fatalf("manifest %s, %v", raw, err)
	}
	for _, name := range []string{"openai.json", "claude.json", "gemini.json", "grok.json"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if manifest.Files[name] != hex.EncodeToString(sum[:]) {
			t.Errorf("%s: manifest sum %q does not match", name, manifest.Files[name])
		}
		var pages []json.RawMessage
		if err := json.Unmarshal(body, &pages); err != nil {
			t.Errorf("%s is not an array of pages: %v", name, err)
		}
		if name == "claude.json" && len(pages) != 2 {
			t.Errorf("claude.json holds %d pages; want 2", len(pages))
		}
	}
	openai, _ := os.ReadFile(filepath.Join(dir, "openai.json"))
	for _, leak := range []string{"ft:", "acme", "org-acme", "user-xyz"} {
		if bytes.Contains(openai, []byte(leak)) {
			t.Errorf("openai.json keeps %q: %s", leak, openai)
		}
	}
	for _, kept := range []string{`"gpt-6-luna"`, `"gpt-custom"`, `1780000000`, `"owned_by":"system"`, `"owned_by":"<owner>"`} {
		if !bytes.Contains(openai, []byte(kept)) {
			t.Errorf("openai.json lost %s: %s", kept, openai)
		}
	}
	if code := run([]string{"-firstparty", dir}, &out, &errOut, allKeys(),
		&pagedFetcher{pages: firstPartyPages()}); code == 0 {
		t.Error("an existing directory was not refused")
	}
	missing := firstPartyPages()
	delete(missing, llmprovider.ProviderGemini)
	other := filepath.Join(t.TempDir(), "firstparty-missing")
	if code := run([]string{"-firstparty", other}, &out, &errOut, allKeys(),
		&pagedFetcher{pages: missing}); code == 0 {
		t.Error("a listing not read was not refused")
	}
	if _, err := os.Stat(other); !os.IsNotExist(err) {
		t.Errorf("a refused run left %s: %v", other, err)
	}
}
