package chatcompat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0066-PLAN Phase 2 (V4; Q1 (a)): the provider gives its whole catalog
// through catalog.Lister, as catalog.List lists it.

// listing30 is a /models reply of 30 models, local-model-00 on.
func listing30() string {
	data := make([]string, 30)
	for i := range data {
		data[i] = fmt.Sprintf(`{"id":"local-model-%02d","object":"model"}`, i)
	}
	return `{"object":"list","data":[` + strings.Join(data, ",") + `]}`
}

// TestListCatalog_0066: ListCatalog is catalog.List with the provider's id
// and base URL, its options after the provider's, an option List refuses
// refused; a refusal is Live false, its Err matching ErrAuthFailure, and no
// error.
func TestListCatalog_0066(t *testing.T) {
	ctx := context.Background()
	s := newServer(t, func(int, *http.Request) (int, string) { return http.StatusOK, listing30() })
	l, ok := build(t, s.URL).(catalog.Lister)
	if !ok {
		t.Fatal("the provider is not a catalog.Lister")
	}
	got, err := l.ListCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want, err := catalog.List(ctx, testID, llmprovider.NewStaticToken(""), testCache(t), WithID(testID),
		llmprovider.WithBaseURL(s.URL), llmprovider.WithoutModelMetadata())
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != catalog.ListLive || !slices.Equal(got.Usable, want.Usable) ||
		!slices.Equal(got.Recommended, want.Recommended) || len(got.Menu) != len(want.Menu) || len(got.Usable) != 30 {
		t.Errorf("ListCatalog: Source %q, Usable %d, Recommended %d, Menu %d; want List's: %q, %d, %d, %d",
			got.Source, len(got.Usable), len(got.Recommended), len(got.Menu),
			want.Source, len(want.Usable), len(want.Recommended), len(want.Menu))
	}

	// WithLimit is scoped to the built-ins, so List refuses it here, and the
	// wizard cuts the catalog itself (0066-MADR, amendment "the Lister's
	// options").
	if _, err := l.ListCatalog(ctx, catalog.WithLimit(3)); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Errorf("WithLimit(3): err %v; want ErrUnsupported, as List refuses it", err)
	}

	refused := newServer(t, func(int, *http.Request) (int, string) {
		return http.StatusUnauthorized, `{"error":{"message":"no key"}}`
	})
	cat, err := build(t, refused.URL).(catalog.Lister).ListCatalog(ctx)
	if err != nil || cat.Live || !errors.Is(cat.Err, llmprovider.ErrAuthFailure) {
		t.Errorf("a 401: err %v, Live %v, Err %v; want nil, false, ErrAuthFailure", err, cat.Live, cat.Err)
	}
}
