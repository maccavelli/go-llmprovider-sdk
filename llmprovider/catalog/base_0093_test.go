package catalog

// 0093-MADR's rows that hold of the module with no document handed in,
// written so that they build at the record's base and are seen red there:
// MH11 (no request for a document), MH16 (no extract evidence), MH23 (no
// address), MH24 (nothing embedded), MH25 (the notice) and MH27 (no
// variable).

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// moduleRoot0093 is the module's root: the nearest directory above this
// package's that holds go.mod.
func moduleRoot0093(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the package")
		}
		dir = parent
	}
}

// nonTestGoFiles0093 is every non-test Go file of the module, relative to
// its root, outside testdata.
func nonTestGoFiles0093(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "testdata" || strings.HasPrefix(d.Name(), ".")) && path != root {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no Go file found")
	}
	return files
}

// metadataAddress0093 matches a model-metadata address in a string.
var metadataAddress0093 = regexp.MustCompile(`(?i)models\.dev/|models\.opencode\.ai|api\.json`)

// TestNoMetadataFetch_0093 (MH23; D6, D14): no string literal in a non-test
// Go file of the module names a model-metadata address (models.dev's,
// models.opencode.ai's, or an api.json); a comment may name one, as the docs
// do. A literal is read as Go reads it, so a planted constant is found
// whatever its quoting.
func TestNoMetadataFetch_0093(t *testing.T) {
	root := moduleRoot0093(t)
	for _, rel := range nonTestGoFiles0093(t, root) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, filepath.Join(root, rel), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				value = lit.Value
			}
			if metadataAddress0093.MatchString(value) {
				t.Errorf("%s:%d: a string literal names a model-metadata address: %s", rel, fset.Position(lit.Pos()).Line,
					lit.Value)
			}
			return true
		})
	}
}

// TestNoMetadataEnv_0093 (MH27; D10): no non-test Go file names the two
// variables the removed fetch read.
func TestNoMetadataEnv_0093(t *testing.T) {
	root := moduleRoot0093(t)
	for _, rel := range nonTestGoFiles0093(t, root) {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"LLMPROVIDER_MODELS_METADATA_URL", "LLMPROVIDER_DISABLE_MODELS_METADATA"} {
			if strings.Contains(string(raw), name) {
				t.Errorf("%s names %s", rel, name)
			}
		}
	}
}

// TestNoEmbeddedExtract_0093 (MH24; D7): the catalog package embeds nothing,
// and holds no limits_snapshot.json.
func TestNoEmbeddedExtract_0093(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == "limits_snapshot.json" {
			t.Errorf("the catalog package holds %s", name)
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "//go:embed") {
			t.Errorf("%s embeds a file (//go:embed)", name)
		}
	}
}

// modelsDevData0093 is every file of models.dev data in the module (0093-MADR
// F6, D11, MH55), relative to the module's root.
var modelsDevData0093 = []string{
	"llmprovider/internal/opencoderoute/routes_snapshot.json",
	"llmprovider/catalog/testdata/ranking-2026-09-26/metadata.json",
	"llmprovider/catalog/testdata/ranking-2026-10-12/metadata.json",
	"llmprovider/catalog/testdata/versions-2026-11-06/families.json",
	"llmprovider/catalog/testdata/metadata-0033/fixture.json",
	"llmprovider/catalog/testdata/extract-2026-10-12/api.json",
}

// TestNoticeNamesModelsDevData_0093 (MH25; D11, Q3): NOTICE at the module's
// root names each file of models.dev data, each of which exists, and credits
// models.dev under the MIT licence, whose text LICENSE.models.dev holds.
func TestNoticeNamesModelsDevData_0093(t *testing.T) {
	root := moduleRoot0093(t)
	notice, err := os.ReadFile(filepath.Join(root, "NOTICE"))
	if err != nil {
		t.Fatalf("NOTICE: %v", err)
	}
	for _, rel := range modelsDevData0093 {
		if !strings.Contains(string(notice), rel) {
			t.Errorf("NOTICE does not name %s", rel)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
	if !strings.Contains(string(notice), "models.dev") || !strings.Contains(string(notice), "MIT") ||
		!strings.Contains(string(notice), "LICENSE.models.dev") {
		t.Error("NOTICE does not credit models.dev under the MIT licence in LICENSE.models.dev")
	}
	licence, err := os.ReadFile(filepath.Join(root, "LICENSE.models.dev"))
	if err != nil {
		t.Fatalf("LICENSE.models.dev: %v", err)
	}
	for _, want := range []string{"MIT License", "Permission is hereby granted, free of charge",
		`THE SOFTWARE IS PROVIDED "AS IS"`} {
		if !strings.Contains(string(licence), want) {
			t.Errorf("LICENSE.models.dev lacks %q", want)
		}
	}
}

// documentCounter0093 answers a listing from its own server and counts every
// request to anything else, or for an api.json, without sending it.
type documentCounter0093 struct {
	host  string
	inner http.RoundTripper
	mu    sync.Mutex
	other []string
}

func (c *documentCounter0093) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != c.host || strings.HasSuffix(req.URL.Path, "api.json") {
		c.mu.Lock()
		c.other = append(c.other, req.URL.String())
		c.mu.Unlock()
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")),
			Header: http.Header{}, Request: req}, nil
	}
	return c.inner.RoundTrip(req)
}

func (c *documentCounter0093) requests() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.other...)
}

// TestList_NoDocumentRequest_0093 (MH11; D6): a listing, a lookup and a
// limits lookup, given no document, ask nothing of anything but the
// listing's server: no request for a model-metadata document.
func TestList_NoDocumentRequest_0093(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"id":"claude-sonnet-5","type":"model"}],"has_more":false}`)
	}))
	defer srv.Close()
	counter := &documentCounter0093{host: strings.TrimPrefix(srv.URL, "http://"), inner: http.DefaultTransport}
	opts := []llmprovider.Option{llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false),
		llmprovider.WithHTTPClient(&http.Client{Transport: counter}), WithCache(new(Cache))}
	ctx := context.Background()
	src := llmprovider.NewStaticToken("k")
	if _, err := List(ctx, llmprovider.ProviderClaude, src, opts...); err != nil {
		t.Fatalf("List: %v", err)
	}
	_, _ = LookupModel(ctx, llmprovider.ProviderClaude, src, "claude-sonnet-5", opts...)
	_, _ = LookupLimits(ctx, llmprovider.ProviderClaude, src, "claude-sonnet-5", opts...)
	_, _ = CachedModel(ctx, llmprovider.ProviderTogether, nil, "m", opts...)
	if got := counter.requests(); len(got) != 0 {
		t.Errorf("requests for a document, or to another host: %q; want none", got)
	}
}

// TestLimits_NoExtract_0093 (MH16; D7): with no document, nothing listed and
// no layer, a model only the extract stated has no limit known and no
// evidence; at the base the extract stated claude-opus-4-6's 128,000.
func TestLimits_NoExtract_0093(t *testing.T) {
	l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-opus-4-6", WithCache(new(Cache)))
	for name, f := range map[string]Fact{"Output": l.Output, "StreamedOutput": l.StreamedOutput, "Window": l.Window,
		"Input": l.Input} {
		if f.Known || len(f.Evidence) != 0 {
			t.Errorf("%s = %+v; want unknown, with no evidence", name, f)
		}
	}
}
