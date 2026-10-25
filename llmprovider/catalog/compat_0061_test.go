package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/compatid"
)

// 0061-PLAN Phase 4a (E1): catalog lists a chatcompat provider's server
// through the listing cache, as it lists a built-in's.

// compatListing is a Chat Completions server's GET /models, eight models.
const compatListing = `{"object":"list","data":[{"id":"m1"},{"id":"m2"},{"id":"m3"},{"id":"m4"},` +
	`{"id":"m5"},{"id":"m6"},{"id":"m7"},{"id":""},{"id":"m2"}]}`

// withCompatID is chatcompat.WithID(id), which catalog cannot import.
func withCompatID(id string) llmprovider.Option {
	return llmprovider.SharedOption("chatcompat.WithID", compatid.Value(id))
}

// compatList is List of "acme" against s's /v1, through c, with key, and
// with no models.dev document, which a compat listing fetches since 0061-PLAN
// Phase 4c (decision 4): no test reaches models.opencode.ai.
func compatList(ctx context.Context, s *steadyServer, c *Cache, key string) (Catalog, error) {
	return List(ctx, "acme", llmprovider.NewStaticToken(key), withCompatID("acme"), llmprovider.WithBaseURL(s.URL+"/v1"),
		WithCache(c), llmprovider.WithoutModelMetadata())
}

// TestCompatListing_Cached_0061: a chatcompat server's listing is GET
// {base}/models, its ids once each and none empty, the first six in its order;
// a second List within ten minutes sends nothing; five at once on a cold
// cache send one request; after ten minutes, one more.
func TestCompatListing_Cached_0061(t *testing.T) {
	clock := useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyReply{status: http.StatusOK, body: compatListing})
	c := &Cache{}
	cat, err := compatList(context.Background(), s, c, "k")
	if err != nil || !cat.Live || cat.Source != ListLive || !slices.Equal(cat.Usable, []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7"}) ||
		!slices.Equal(cat.Recommended[:MaxListed], []string{"m1", "m2", "m3", "m4", "m5", "m6"}) {
		t.Fatalf("List = %+v, %v; want m1 to m7 live, the first six in order", cat, err)
	}
	if s.paths[0] != "/v1/models" {
		t.Errorf("path %q; want /v1/models", s.paths[0])
	}
	if _, err := compatList(context.Background(), s, c, "k"); err != nil || s.requests() != 1 {
		t.Errorf("a second List within ten minutes: %v, %d request(s); want 1", err, s.requests())
	}
	clock.advance(11 * time.Minute)
	if _, err := compatList(context.Background(), s, c, "k"); err != nil || s.requests() != 2 {
		t.Errorf("after eleven minutes: %v, %d request(s); want 2", err, s.requests())
	}

	cold := newSteadyServer(t, 50*time.Millisecond, steadyReply{status: http.StatusOK, body: compatListing})
	shared := &Cache{}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			if _, err := compatList(context.Background(), cold, shared, "k"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if cold.requests() != 1 {
		t.Errorf("five Lists at once: %d requests; want 1", cold.requests())
	}
}

// TestCompatListing_LastGood_0061: after a good listing, a 503 past the ten
// minutes serves it as stale, with the failure.
func TestCompatListing_LastGood_0061(t *testing.T) {
	clock := useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyReply{status: http.StatusOK, body: compatListing},
		steadyReply{status: http.StatusServiceUnavailable})
	c := &Cache{}
	if _, err := compatList(context.Background(), s, c, "k"); err != nil {
		t.Fatal(err)
	}
	clock.advance(11 * time.Minute)
	cat, err := compatList(context.Background(), s, c, "k")
	if err != nil || cat.Source != ListStale || !cat.Live || !errors.Is(cat.Err, llmprovider.ErrProviderUnavailable) ||
		len(cat.Usable) != 7 {
		t.Errorf("List after a 503 = %+v, %v; want the last good listing, stale, with the failure", cat, err)
	}
}

// TestCompatListing_UnknownRefused_0061: an id no built-in has is refused
// as before, ErrUnsupported, unless chatcompat.WithID names it, a WithID of
// another id included; one named with no base URL is ErrInvalidRequest; and
// the lookups refuse it, ErrInvalidProvider, without WithID.
func TestCompatListing_UnknownRefused_0061(t *testing.T) {
	src := llmprovider.NewStaticToken("k")
	base := llmprovider.WithBaseURL("http://127.0.0.1:1")
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
		want error
	}{
		{"no WithID", []llmprovider.Option{base}, llmprovider.ErrUnsupported},
		{"another id's WithID", []llmprovider.Option{base, withCompatID("other")}, llmprovider.ErrUnsupported},
		{"no base URL", []llmprovider.Option{withCompatID("acme")}, llmprovider.ErrInvalidRequest},
	} {
		if _, err := List(context.Background(), "acme", src, tc.opts...); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v; want %v", tc.name, err, tc.want)
		}
	}
	if _, err := LookupModel(context.Background(), "acme", src, "m1", base); !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("LookupModel without WithID: %v; want ErrInvalidProvider", err)
	}
	if _, ok := CachedFacts(context.Background(), "acme", "m1", base, llmprovider.WithAPIKey("k")); ok {
		t.Error("CachedFacts without WithID knew the model")
	}
}

// TestCompatListing_NoKey_0061: an empty key sends no Authorization header;
// a key goes as a bearer token.
func TestCompatListing_NoKey_0061(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Authorization"))
		mu.Unlock()
		_, _ = io.WriteString(w, compatListing)
	}))
	t.Cleanup(srv.Close)
	for _, key := range []string{"", "sk-test"} {
		if _, err := List(context.Background(), "acme", llmprovider.NewStaticToken(key), withCompatID("acme"),
			llmprovider.WithBaseURL(srv.URL), WithCache(&Cache{}), llmprovider.WithoutModelMetadata()); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(seen, []string{"", "Bearer sk-test"}) {
		t.Errorf("Authorization %q; want none, then the bearer token", seen)
	}
}
