package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestListModels_ConcurrentProbes_0053 (0053-MADR D6, M7): five concurrent
// cold ListModels send one run of billed probes, six, where each sent its
// own: thirty. A later, warm call sends none.
func TestListModels_ConcurrentProbes_0053(t *testing.T) {
	var probes atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models") {
			_, _ = io.WriteString(w, `{"data":[{"id":"gpt-5.5"},{"id":"gpt-5.4"},{"id":"gpt-5.2"},{"id":"gpt-5"},`+
				`{"id":"gpt-4.1"},{"id":"gpt-4.1-mini"},{"id":"o3"}]}`)
			return
		}
		probes.Add(1)
		time.Sleep(50 * time.Millisecond)
		_, _ = io.WriteString(w, `{"id":"resp_0053","model":"m","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`)
	}))
	t.Cleanup(srv.Close)
	p, err := New(llmprovider.WithAPIKey("sk-0053"), llmprovider.WithModel("gpt-5.5"), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithoutModelMetadata(), catalog.WithCache(&catalog.Cache{}))
	if err != nil {
		t.Fatal(err)
	}
	lister := p.(llmprovider.ModelLister)
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			if _, err := lister.ListModels(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	cold := probes.Load()
	if _, err := lister.ListModels(context.Background()); err != nil {
		t.Fatal(err)
	}
	if cold != 6 || probes.Load() != cold {
		t.Errorf("%d probes for five concurrent cold listings, then %d for a warm one; want 6, then 0", cold, probes.Load()-cold)
	}
}
