package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0033-PLAN-public-model-metadata.md Phase 4, as its deviation resolved it:
// main calls run, which these tests drive with no network and no real key.

const runDoc = `{"anthropic": {"models": {
 "claude-haiku-5-5": {"name": "Haiku", "tool_call": true, "release_date": "2026-09-01", "cost": {"input": 1, "output": 5}},
 "claude-next": {"name": "Next", "tool_call": true, "release_date": "2026-12-01", "cost": {"input": 3, "output": 15}}
}}}`

const testKey = "sk-ant-0033-never-printed"

// fakeFetcher answers with a canned listing and listing body, or an error,
// and records the keys it was given, by provider, and which it ranked.
type fakeFetcher struct {
	listed map[llmprovider.ProviderID][]string
	bodies map[llmprovider.ProviderID][]byte
	fail   error
	keys   []string
	asked  map[llmprovider.ProviderID]string
	ranked map[llmprovider.ProviderID]bool
}

func (f *fakeFetcher) Metadata(context.Context) (catalog.Metadata, error) {
	return catalog.ParseMetadata(strings.NewReader(runDoc))
}

func (f *fakeFetcher) Catalog(_ context.Context, p llmprovider.ProviderID, key string, ranked bool) (catalog.Catalog, error) {
	if key != "" {
		f.keys = append(f.keys, key)
	}
	if f.asked == nil {
		f.asked, f.ranked = map[llmprovider.ProviderID]string{}, map[llmprovider.ProviderID]bool{}
	}
	f.asked[p], f.ranked[p] = key, ranked
	if f.fail != nil {
		return catalog.Catalog{}, f.fail
	}
	return catalog.Catalog{Usable: f.listed[p], Recommended: f.listed[p], Live: true}, nil
}

func (f *fakeFetcher) Listing(p llmprovider.ProviderID) ([]byte, bool) {
	body, ok := f.bodies[p]
	return body, ok
}

// TestRun_Proposes_0033: with Claude's key set, run lists Claude, prints the
// proposals and a summary, and never prints the key.
func TestRun_Proposes_0033(t *testing.T) {
	f := &fakeFetcher{listed: map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-haiku-5-5", "claude-next"}}}
	var out, errOut bytes.Buffer
	rc := run(nil, &out, &errOut, map[llmprovider.ProviderID]string{llmprovider.ProviderClaude: testKey}, f)
	if rc != 0 {
		t.Fatalf("run = %d; stderr %s", rc, errOut.String())
	}
	if len(f.keys) != 1 || f.keys[0] != testKey {
		t.Errorf("the lister was given %d key(s), want Claude's", len(f.keys))
	}
	// Claude's is a vendor list, so its newer model is a review line
	// (0033-MADR's amendment "the second refresh, and what the command
	// reads", Q18).
	if !strings.Contains(out.String(), "claude\treview\tclaude-next\t") {
		t.Errorf("stdout lacks the review line for claude-next:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "proposal(s)") {
		t.Errorf("stdout lacks the summary:\n%s", out.String())
	}
	if strings.Contains(out.String()+errOut.String(), testKey) {
		t.Error("the key was printed")
	}
	if !strings.Contains(errOut.String(), "unset; its listing is skipped") {
		t.Errorf("stderr does not say which listings were skipped:\n%s", errOut.String())
	}
}

// TestKeysFromEnv_0033 (R30): the keys come from the providers' variables,
// and an unset one is left out. Every variable is blanked first, so the test
// never reads a key the host holds.
func TestKeysFromEnv_0033(t *testing.T) {
	for _, v := range llmprovider.ProviderEnvVars() {
		t.Setenv(v, "")
	}
	t.Setenv(llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude], testKey)
	keys := KeysFromEnv()
	if len(keys) != 1 || keys[llmprovider.ProviderClaude] != testKey {
		t.Errorf("KeysFromEnv found %d key(s), want Claude's alone", len(keys))
	}
}

// failingWriter fails every write.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

// TestRun_WriteFails_0033: a failed write is a tool error.
func TestRun_WriteFails_0033(t *testing.T) {
	var errOut bytes.Buffer
	if rc := run(nil, failingWriter{}, &errOut, nil, &fakeFetcher{}); rc != 2 {
		t.Errorf("a failed write: run = %d, want 2", rc)
	}
}

// TestRun_DocumentAndFailures_0033: a document given with -doc is read in
// place of a fetch; a listing that fails is reported, and the run goes on; a
// missing document is a tool error.
func TestRun_DocumentAndFailures_0033(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.json")
	if err := os.WriteFile(path, []byte(runDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	f := &fakeFetcher{fail: errors.New("listing down")}
	var out, errOut bytes.Buffer
	if rc := run([]string{"-doc", path}, &out, &errOut, map[llmprovider.ProviderID]string{llmprovider.ProviderClaude: testKey}, f); rc != 0 {
		t.Fatalf("run -doc = %d; stderr %s", rc, errOut.String())
	}
	if !strings.Contains(errOut.String(), "claude: listing: listing down") {
		t.Errorf("stderr does not report the failed listing:\n%s", errOut.String())
	}
	if rc := run([]string{"-doc", filepath.Join(t.TempDir(), "missing.json")}, &out, &errOut, nil, f); rc != 2 {
		t.Errorf("a missing document: run = %d, want 2", rc)
	}
	if rc := run([]string{"-no-such-flag"}, &out, &errOut, nil, f); rc != 2 {
		t.Errorf("a bad flag: run = %d, want 2", rc)
	}
}
