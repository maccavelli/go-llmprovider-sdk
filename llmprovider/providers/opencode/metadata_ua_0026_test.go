package opencode

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOpencode_MetadataLookupUsesClientInfo (0026-MADR F33): the metadata
// request a generation makes to pick its route names the caller's
// application, as every other request does (0020-MADR F37).
func TestOpencode_MetadataLookupUsesClientInfo(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	// The model is in Go's route table, so a cold cache routes by the table
	// and the metadata request runs in the background (0028-MADR D-A1): the
	// test waits for it (0028-PLAN D13).
	requested := make(chan struct{}, 1)
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.Header.Get("User-Agent"))
		mu.Unlock()
		select {
		case requested <- struct{}{}:
		default:
		}
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
	select {
	case <-requested:
	case <-time.After(5 * time.Second):
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
