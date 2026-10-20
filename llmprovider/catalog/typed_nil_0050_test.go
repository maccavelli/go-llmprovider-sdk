package catalog_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// tokenFunc0050 is a caller's own source, of a func type.
type tokenFunc0050 func(context.Context) (llmprovider.Token, error)

func (f tokenFunc0050) Token(ctx context.Context) (llmprovider.Token, error) { return f(ctx) }

// typedNilSources0050 are a typed nil of each source the module has, and of
// a caller's own.
var typedNilSources0050 = []llmprovider.TokenSource{
	(*llmprovider.StaticToken)(nil),
	(*llmprovider.CommandToken)(nil),
	(*auth.OAuthSession)(nil),
	(*auth.VendorCLISession)(nil),
	tokenFunc0050(nil),
}

// knownLayer0050 knows one model's context window.
type knownLayer0050 struct{}

func (knownLayer0050) Name() string { return "test-0050" }

func (knownLayer0050) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	if id != "claude-x" {
		return catalog.ModelInfo{}, false
	}
	return catalog.ModelInfo{ID: id, ContextWindow: 5000}.Known(catalog.FieldContextWindow), true
}

// offline0050 are options that reach no network.
func offline0050(more ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithBaseURL(nettest.RefusedURL),
		llmprovider.WithHTTPClient(nettest.Refusing()), llmprovider.WithoutModelMetadata()}, more...)
}

// call0050 runs f, reporting a panic as a failure named name.
func call0050(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if v := recover(); v != nil {
			t.Errorf("%s panicked: %v; want no panic", name, v)
		}
	}()
	f()
}

// TestList_TypedNilSources_0050 (0050-MADR D4, Q2): List and ListRaw, for
// every built-in provider, refuse a nil source and each typed-nil one with
// ErrInvalidRequest, the typed nil's naming its type; none panics. The
// MADR's probe saw List panic for every typed-nil source of every provider.
func TestList_TypedNilSources_0050(t *testing.T) {
	ctx := context.Background()
	sources := append([]llmprovider.TokenSource{nil}, typedNilSources0050...)
	for _, d := range providers.Default().Descriptors() {
		for _, src := range sources {
			want := "TokenSource is required"
			if src != nil {
				want = fmt.Sprintf("a nil %T is not a TokenSource", src)
			}
			name := fmt.Sprintf("%s, %T", d.ID, src)
			var errList, errRaw error
			call0050(t, name+": List", func() { _, errList = catalog.List(ctx, d.ID, src, offline0050()...) })
			call0050(t, name+": ListRaw", func() { _, errRaw = catalog.ListRaw(ctx, d.ID, src, offline0050()...) })
			for fn, err := range map[string]error{"List": errList, "ListRaw": errRaw} {
				if !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(fmt.Sprint(err), want) {
					t.Errorf("%s: %s = %v; want ErrInvalidRequest, %q", name, fn, err, want)
				}
			}
		}
	}
}

// TestLookupModel_TypedNilSources_0050 (0050-MADR D4): LookupModel with a
// typed-nil source merges the caller's layer and reports the source with
// ErrInvalidRequest; with a nil one, it merges the layer with no error.
// CachedModel treats a typed nil as no source. None panics.
func TestLookupModel_TypedNilSources_0050(t *testing.T) {
	ctx := context.Background()
	opts := offline0050(catalog.WithModelLayer(knownLayer0050{}))
	for _, src := range append([]llmprovider.TokenSource{nil}, typedNilSources0050...) {
		name := fmt.Sprintf("%T", src)
		var info, cached catalog.ModelInfo
		var err error
		var known bool
		call0050(t, name+": LookupModel", func() {
			info, err = catalog.LookupModel(ctx, llmprovider.ProviderClaude, src, "claude-x", opts...)
		})
		call0050(t, name+": CachedModel", func() {
			cached, known = catalog.CachedModel(ctx, llmprovider.ProviderClaude, src, "claude-x", opts...)
		})
		if !info.Has(catalog.FieldContextWindow) || info.ContextWindow != 5000 {
			t.Errorf("%s: LookupModel = %+v; want the layer's context window", name, info)
		}
		switch {
		case src == nil && err != nil:
			t.Errorf("%s: LookupModel err = %v; want nil", name, err)
		case src != nil && (!errors.Is(err, llmprovider.ErrInvalidRequest) ||
			!strings.Contains(fmt.Sprint(err), fmt.Sprintf("a nil %T is not a TokenSource", src))):
			t.Errorf("%s: LookupModel err = %v; want ErrInvalidRequest naming its type", name, err)
		}
		if !known || !cached.Has(catalog.FieldContextWindow) || cached.ContextWindow != 5000 {
			t.Errorf("%s: CachedModel = %+v, %v; want the layer's context window, known", name, cached, known)
		}
	}
}
