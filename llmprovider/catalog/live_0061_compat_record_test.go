//go:build live_gateways

package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLive_RecordCompat_0061 records a Chat Completions server's replies for
// E2's dialects (0061-PLAN Phase 4c, step 1). LLMPROVIDER_LIVE_0061_COMPAT
// names the server's base URL, such as http://127.0.0.1:11434/v1, and
// LLMPROVIDER_LIVE_0061_COMPAT_DIALECT the dialect's directory under
// testdata/compat. It writes {base}/models byte for byte as models.json. For
// the ollama dialect it also writes the root's /api/tags as tags.json and,
// for each model LLMPROVIDER_LIVE_0061_COMPAT_SHOW names (comma separated),
// its /api/show as show-<model>.json, kept to the fields the catalog reads:
// the modelfile names its blob by a path under the local account, and the
// license, template, parameters and tensors are unread bulk (0061-MADR,
// amendment "E2 and E4's menu, made exact before Phase 4c", X7). It sends no
// key and generates nothing.
func TestLive_RecordCompat_0061(t *testing.T) {
	base := strings.TrimRight(os.Getenv("LLMPROVIDER_LIVE_0061_COMPAT"), "/")
	dialect := os.Getenv("LLMPROVIDER_LIVE_0061_COMPAT_DIALECT")
	if base == "" || dialect == "" {
		t.Skip("LLMPROVIDER_LIVE_0061_COMPAT and LLMPROVIDER_LIVE_0061_COMPAT_DIALECT name the server and its dialect")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir := filepath.Join("testdata", "compat", dialect)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, body []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s, %d bytes", filepath.Join(dir, name), len(body))
	}
	write("models.json", recordGet(ctx, t, base+"/models"))
	if dialect != "ollama" {
		return
	}
	root := strings.TrimSuffix(base, "/v1")
	write("tags.json", recordGet(ctx, t, root+"/api/tags"))
	for _, model := range strings.Split(os.Getenv("LLMPROVIDER_LIVE_0061_COMPAT_SHOW"), ",") {
		if model = strings.TrimSpace(model); model == "" {
			continue
		}
		body, err := json.Marshal(map[string]string{"model": model})
		if err != nil {
			t.Fatal(err)
		}
		reply := recordDo(ctx, t, http.MethodPost, root+"/api/show", body)
		var full map[string]json.RawMessage
		if err := json.Unmarshal(reply, &full); err != nil {
			t.Fatalf("%s: /api/show: %v", model, err)
		}
		kept := map[string]json.RawMessage{}
		for _, k := range []string{"capabilities", "details", "model_info", "modified_at"} {
			if v, ok := full[k]; ok {
				kept[k] = v
			}
		}
		out, err := json.MarshalIndent(kept, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		write("show-"+strings.NewReplacer(":", "_", "/", "_").Replace(model)+".json", append(out, '\n'))
	}
}

// recordGet is url's body, which must answer 200.
func recordGet(ctx context.Context, t *testing.T, url string) []byte {
	t.Helper()
	return recordDo(ctx, t, http.MethodGet, url, nil)
}

// recordDo sends one request with no credential and returns its body, which
// must answer 200.
func recordDo(ctx context.Context, t *testing.T, method, url string, body []byte) []byte {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	reply, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("%s %s: %d, %v", method, url, resp.StatusCode, err)
	}
	return reply
}
