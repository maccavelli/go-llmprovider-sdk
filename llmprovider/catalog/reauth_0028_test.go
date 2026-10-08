package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestList_Forbidden403NotRenewed (0028-MADR D-H2): a listing refused with a
// region 403 is not a refused credential: it is not renewed nor sent again,
// and the catalog's failure is ErrNotPermitted.
func TestList_Forbidden403NotRenewed(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":{"type":"request_forbidden","code":"unsupported_country_region_territory","message":"Country, region, or territory not supported"}}`)
	}))
	defer srv.Close()
	src := &invalidatingSource{}
	cat, err := List(context.Background(), llmprovider.ProviderTogether, src, llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithoutModelMetadata())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if requests.Load() != 1 || src.invalidated.Load() != 0 || !errors.Is(cat.Err, llmprovider.ErrNotPermitted) {
		t.Fatalf("requests=%d invalidations=%d Err=%v; want 1, 0 and ErrNotPermitted",
			requests.Load(), src.invalidated.Load(), cat.Err)
	}
}
