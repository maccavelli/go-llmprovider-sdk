package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// steadyReply is one scripted listing reply.
type steadyReply struct {
	status int
	body   string
}

// steadyServer answers each request with the next scripted reply, the last
// repeated, after delay, and records each request's path.
type steadyServer struct {
	*httptest.Server
	mu      sync.Mutex
	replies []steadyReply
	paths   []string
	delay   time.Duration
}

func newSteadyServer(t *testing.T, delay time.Duration, replies ...steadyReply) *steadyServer {
	t.Helper()
	s := &steadyServer{replies: replies, delay: delay}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		reply := s.replies[min(len(s.paths), len(s.replies)-1)]
		s.paths = append(s.paths, r.URL.Path)
		s.mu.Unlock()
		time.Sleep(s.delay)
		if reply.status != http.StatusOK {
			w.WriteHeader(reply.status)
			_, _ = w.Write([]byte(`{"error":{"message":"scripted failure"}}`))
			return
		}
		_, _ = w.Write([]byte(reply.body))
	}))
	t.Cleanup(s.Close)
	return s
}

// requests is how many requests s has answered.
func (s *steadyServer) requests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.paths)
}

// steadyClock is the listing cache's clock, moved by hand.
type steadyClock struct {
	mu  sync.Mutex
	now time.Time
}

func useSteadyClock(t *testing.T) *steadyClock {
	t.Helper()
	resetLookupCaches(t)
	c := &steadyClock{now: time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)}
	saved := listingNow
	listingNow = c.Now
	t.Cleanup(func() { listingNow = saved })
	return c
}

func (c *steadyClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *steadyClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// steadyList is List of id against s, with the key "k".
func steadyList(t *testing.T, id llmprovider.ProviderID, s *steadyServer, opts ...llmprovider.Option) Catalog {
	t.Helper()
	cat, err := listT(context.Background(), id, llmprovider.NewStaticToken("k"),
		append([]llmprovider.Option{llmprovider.WithBaseURL(s.URL)}, opts...)...)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return cat
}

var steadyOpenAI = steadyReply{http.StatusOK, openAIStyleListing("gpt-4.1-mini", "gpt-4o", "gpt-5.1")}

// TestList_Cached_0041: a listing read within listingMetadataTTL is served
// again without a request; past it, the listing is fetched again; callers
// at once share one fetch (0041-MADR D6).
func TestList_Cached_0041(t *testing.T) {
	clock := useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyOpenAI)
	first := steadyList(t, llmprovider.ProviderOpenAI, s)
	clock.advance(5 * time.Minute)
	second := steadyList(t, llmprovider.ProviderOpenAI, s)
	if n := s.requests(); n != 1 {
		t.Errorf("%d listing requests for two Lists within the TTL; want 1", n)
	}
	if !slices.Equal(first.Recommended, second.Recommended) || second.Source != ListLive ||
		!second.FetchedAt.Equal(first.FetchedAt) {
		t.Errorf("cached: %q, %s at %v; want %q, live at %v", second.Recommended, second.Source, second.FetchedAt,
			first.Recommended, first.FetchedAt)
	}
	clock.advance(6 * time.Minute)
	steadyList(t, llmprovider.ProviderOpenAI, s)
	if n := s.requests(); n != 2 {
		t.Errorf("%d listing requests once the TTL passed; want 2", n)
	}

	concurrent := newSteadyServer(t, 100*time.Millisecond, steadyOpenAI)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { steadyList(t, llmprovider.ProviderOpenAI, concurrent) })
	}
	wg.Wait()
	if n := concurrent.requests(); n != 1 {
		t.Errorf("%d listing requests for 8 concurrent Lists; want 1", n)
	}
}

// steadyOpenAIDated is an OpenAI listing whose created times are facts:
// each model's release date (0041-PLAN Phase 2).
var steadyOpenAIDated = steadyReply{http.StatusOK, `{"data":[{"id":"gpt-4.1-mini","created":1744675200},` +
	`{"id":"gpt-4o","created":1715558400},{"id":"gpt-5.1","created":1762992000}]}`}

// TestList_CachedRebuildsWithProfile_0041: a cached listing is built as a
// live one is: ranked with the caller's own profile, and with the facts the
// listing gave (0041-MADR A6).
func TestList_CachedRebuildsWithProfile_0041(t *testing.T) {
	useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyOpenAIDated)
	steadyList(t, llmprovider.ProviderOpenAI, s, WithProfile(ProfileUtility))
	cached := steadyList(t, llmprovider.ProviderOpenAI, s, WithProfile(ProfileCapable))
	if n := s.requests(); n != 1 {
		t.Fatalf("%d listing requests; want 1", n)
	}
	live := steadyList(t, llmprovider.ProviderOpenAI, newSteadyServer(t, 0, steadyOpenAIDated), WithProfile(ProfileCapable))
	if !slices.Equal(cached.Recommended, live.Recommended) {
		t.Errorf("a cached listing under ProfileCapable recommends %q; a live one %q", cached.Recommended, live.Recommended)
	}
	dated := 0
	for i, e := range cached.Menu {
		l := live.Menu[i]
		if e.ID != l.ID || e.Info.Has(FieldReleased) != l.Info.Has(FieldReleased) || !e.Info.Released.Equal(l.Info.Released) {
			t.Errorf("menu entry %d: cached %s released %v (%t); live %s released %v (%t)", i, e.ID, e.Info.Released,
				e.Info.Has(FieldReleased), l.ID, l.Info.Released, l.Info.Has(FieldReleased))
		}
		if l.Info.Has(FieldReleased) {
			dated++
		}
	}
	if dated == 0 {
		t.Error("the live listing gave no release date; the fixture does not test the facts")
	}
}

// TestList_StaleOn503_0041: a transient failure serves the last good
// listing: Source ListStale, Live true, Err the failure, FetchedAt when it
// was read (0041-MADR D6).
func TestList_StaleOn503_0041(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		want   error
	}{
		{"503", http.StatusServiceUnavailable, llmprovider.ErrProviderUnavailable},
		{"429", http.StatusTooManyRequests, llmprovider.ErrRateLimited},
		// The listers return a send failure as the transport gives it, a
		// net.Error; only Ollama's wraps it.
		{"network", 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clock := useSteadyClock(t)
			s := newSteadyServer(t, 0, steadyOpenAI, steadyReply{tc.status, ""})
			first := steadyList(t, llmprovider.ProviderOpenAI, s)
			clock.advance(11 * time.Minute)
			if tc.status == 0 {
				s.Close()
			}
			stale := steadyList(t, llmprovider.ProviderOpenAI, s)
			var netErr net.Error
			matched := errors.Is(stale.Err, tc.want)
			if tc.want == nil {
				matched = errors.As(stale.Err, &netErr)
			}
			if stale.Source != ListStale || !stale.Live || !matched {
				t.Errorf("Source %s, Live %t, Err %v; want stale, live, %v", stale.Source, stale.Live, stale.Err, tc.want)
			}
			if !slices.Equal(stale.Usable, first.Usable) || !slices.Equal(stale.Recommended, first.Recommended) ||
				!stale.FetchedAt.Equal(first.FetchedAt) {
				t.Errorf("stale %q / %q at %v; want the first listing's %q / %q at %v", stale.Usable, stale.Recommended,
					stale.FetchedAt, first.Usable, first.Recommended, first.FetchedAt)
			}
		})
	}
}

// TestList_NoStaleOn401_0041: a refused credential never serves the last
// good listing: the static list, Source ListStatic, Live false (0041-MADR D6).
func TestList_NoStaleOn401_0041(t *testing.T) {
	clock := useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyOpenAI, steadyReply{http.StatusUnauthorized, ""})
	steadyList(t, llmprovider.ProviderOpenAI, s)
	clock.advance(11 * time.Minute)
	cat := steadyList(t, llmprovider.ProviderOpenAI, s)
	if cat.Source != ListStatic || cat.Live || !errors.Is(cat.Err, llmprovider.ErrAuthFailure) || !cat.FetchedAt.IsZero() {
		t.Errorf("Source %s, Live %t, Err %v, FetchedAt %v; want static, not live, an auth failure, zero", cat.Source,
			cat.Live, cat.Err, cat.FetchedAt)
	}
	if !slices.Equal(cat.Recommended, Static(llmprovider.ProviderOpenAI)) {
		t.Errorf("Recommended %q; want the static list", cat.Recommended)
	}
}

// TestList_StaleExpires_0041: a listing older than 24 hours is not served
// stale; the static list is the floor (0041-MADR D6).
func TestList_StaleExpires_0041(t *testing.T) {
	clock := useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyOpenAI, steadyReply{http.StatusServiceUnavailable, ""})
	steadyList(t, llmprovider.ProviderOpenAI, s)
	clock.advance(25 * time.Hour)
	cat := steadyList(t, llmprovider.ProviderOpenAI, s)
	if cat.Source != ListStatic || cat.Live || !errors.Is(cat.Err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("Source %s, Live %t, Err %v; want the static floor", cat.Source, cat.Live, cat.Err)
	}
}

// TestList_LastGood_0041: with nothing cached, WithLastGood's listing is the
// last good one, within 24 hours, on a transient failure only; a cached
// listing comes before it (0041-MADR D6).
func TestList_LastGood_0041(t *testing.T) {
	ids := []string{"gpt-4o", "gpt-5.1"}
	for _, tc := range []struct {
		name   string
		status int
		age    time.Duration
		source ListSource
	}{
		{"503", http.StatusServiceUnavailable, time.Hour, ListStale},
		{"401", http.StatusUnauthorized, time.Hour, ListStatic},
		{"older than 24 hours", http.StatusServiceUnavailable, 25 * time.Hour, ListStatic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clock := useSteadyClock(t)
			at := clock.Now().Add(-tc.age)
			s := newSteadyServer(t, 0, steadyReply{tc.status, ""})
			cat := steadyList(t, llmprovider.ProviderOpenAI, s, WithLastGood(ids, at))
			if cat.Source != tc.source {
				t.Fatalf("Source %s; want %s", cat.Source, tc.source)
			}
			if tc.source == ListStale && (!slices.Equal(cat.Usable, ids) || !cat.FetchedAt.Equal(at) || !cat.Live) {
				t.Errorf("Usable %q at %v, Live %t; want %q at %v, live", cat.Usable, cat.FetchedAt, cat.Live, ids, at)
			}
		})
	}
	t.Run("a cached listing first", func(t *testing.T) {
		clock := useSteadyClock(t)
		s := newSteadyServer(t, 0, steadyOpenAI, steadyReply{http.StatusServiceUnavailable, ""})
		first := steadyList(t, llmprovider.ProviderOpenAI, s)
		clock.advance(11 * time.Minute)
		cat := steadyList(t, llmprovider.ProviderOpenAI, s, WithLastGood(ids, clock.Now()))
		if cat.Source != ListStale || !slices.Equal(cat.Usable, first.Usable) {
			t.Errorf("Source %s, Usable %q; want the cached %q, stale", cat.Source, cat.Usable, first.Usable)
		}
	})
}

// TestList_SourceAlways_0041: every path sets Source, and FetchedAt where a
// listing was read (0041-MADR D6).
func TestList_SourceAlways_0041(t *testing.T) {
	clock := useSteadyClock(t)
	start := clock.Now()
	s := newSteadyServer(t, 0, steadyOpenAI, steadyReply{http.StatusServiceUnavailable, ""})
	live := steadyList(t, llmprovider.ProviderOpenAI, s)
	clock.advance(time.Minute)
	cached := steadyList(t, llmprovider.ProviderOpenAI, s)
	clock.advance(11 * time.Minute)
	stale := steadyList(t, llmprovider.ProviderOpenAI, s)
	static := steadyList(t, llmprovider.ProviderClaude, newSteadyServer(t, 0, steadyReply{http.StatusUnauthorized, ""}))
	for _, tc := range []struct {
		name   string
		cat    Catalog
		source ListSource
		at     time.Time
	}{
		{"live", live, ListLive, start},
		{"cached", cached, ListLive, start},
		{"stale", stale, ListStale, start},
		{"static", static, ListStatic, time.Time{}},
	} {
		if tc.cat.Source != tc.source || !tc.cat.FetchedAt.Equal(tc.at) {
			t.Errorf("%s: Source %q at %v; want %q at %v", tc.name, tc.cat.Source, tc.cat.FetchedAt, tc.source, tc.at)
		}
	}
}

// TestList_KiloStaleRanksAsLive_0041: a stale Kilo listing is ranked from
// the cached entries, prices included, so its band is the live one
// (0041-MADR A6).
func TestList_KiloStaleRanksAsLive_0041(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	raw, err := os.ReadFile(filepath.Join(dir, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		RecordedAt time.Time `json:"recorded_at"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	pinRankingNow(t, manifest.RecordedAt)
	body, err := os.ReadFile(filepath.Join(dir, snapshotFile[llmprovider.ProviderKilo]))
	if err != nil {
		t.Fatal(err)
	}
	// Measured in Phase 6's research: without the entries, ProfileCapable's
	// six differ entirely, and ProfileUtility's full list from its seventh
	// id; ProfileUtility's six do not, the ids being cheapest first.
	for _, tc := range []struct {
		name string
		opts []llmprovider.Option
	}{
		{"capable six", []llmprovider.Option{WithProfile(ProfileCapable), WithLimit(MaxListed)}},
		{"utility full list", []llmprovider.Option{WithProfile(ProfileUtility), WithLimit(0)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clock := useSteadyClock(t)
			s := newSteadyServer(t, 0, steadyReply{http.StatusOK, string(body)},
				steadyReply{http.StatusServiceUnavailable, ""})
			live := steadyList(t, llmprovider.ProviderKilo, s, tc.opts...)
			clock.advance(11 * time.Minute)
			stale := steadyList(t, llmprovider.ProviderKilo, s, tc.opts...)
			if stale.Source != ListStale || !slices.Equal(stale.Recommended, live.Recommended) {
				t.Errorf("stale Kilo list (%s, %d ids) differs from the live one (%d ids): %q…; want %q…", stale.Source,
					len(stale.Recommended), len(live.Recommended), stale.Recommended[:min(8, len(stale.Recommended))],
					live.Recommended[:min(8, len(live.Recommended))])
			}
		})
	}
}

// TestList_KiloOrganizationKeyed_0041: a Kilo organization's listing is
// cached apart from the personal account's (0041-MADR A6).
func TestList_KiloOrganizationKeyed_0041(t *testing.T) {
	useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyReply{http.StatusOK, kiloListingFixture})
	steadyList(t, llmprovider.ProviderKilo, s)
	steadyList(t, llmprovider.ProviderKilo, s, WithKiloOrganization("org-1"))
	steadyList(t, llmprovider.ProviderKilo, s)
	steadyList(t, llmprovider.ProviderKilo, s, WithKiloOrganization("org-1"))
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.paths) != 2 || s.paths[0] == s.paths[1] {
		t.Errorf("listing requests %q; want one personal, one for the organization, each once", s.paths)
	}
}

// TestList_OllamaOutsideD6_0041: Ollama lists every time, and returns its
// error rather than a stale listing (0041-MADR A6).
func TestList_OllamaOutsideD6_0041(t *testing.T) {
	clock := useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyReply{http.StatusOK, ollamaTags("llama3:latest")})
	first := steadyList(t, llmprovider.ProviderOllama, s)
	steadyList(t, llmprovider.ProviderOllama, s)
	tags := 0
	s.mu.Lock()
	for _, p := range s.paths {
		if p == "/api/tags" {
			tags++
		}
	}
	s.mu.Unlock()
	if tags != 2 || first.Source != ListLive || !first.FetchedAt.Equal(clock.Now()) {
		t.Errorf("/api/tags asked %d times, Source %s at %v; want 2, live at %v", tags, first.Source, first.FetchedAt,
			clock.Now())
	}
	s.Close()
	if _, err := listT(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""),
		llmprovider.WithBaseURL(s.URL)); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("Ollama down: err %v; want ErrProviderUnavailable, not a stale listing", err)
	}
}
