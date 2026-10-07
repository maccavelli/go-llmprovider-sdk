package opencode

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOpencode_MetadataLookupUsesClientInfo (0026-MADR F33): the metadata
// request a generation makes to pick its route names the caller's
// application, as every other request does (0020-MADR F37).
func TestOpencode_MetadataLookupUsesClientInfo(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.Header.Get("User-Agent"))
		mu.Unlock()
		_, _ = io.WriteString(w, `{}`)
	}))
	defer meta.Close()
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hi"}}]}`)
	}))
	defer gateway.Close()

	p, err := NewGo(llmprovider.WithAPIKey("k"), llmprovider.WithModel("glm-5.3-flash"), llmprovider.WithBaseURL(gateway.URL),
		llmprovider.WithModelMetadataURL(meta.URL), llmprovider.WithClientInfo("acme-cli", "4.2.0"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) == 0 {
		t.Fatal("no metadata request was made")
	}
	for _, ua := range agents {
		if !strings.HasPrefix(ua, "acme-cli/4.2.0 ") {
			t.Errorf("metadata User-Agent = %q; want it led by the caller's acme-cli/4.2.0", ua)
		}
	}
}
