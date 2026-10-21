package catalog

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// list0053 is List of OpenAI against s, through c.
func list0053(ctx context.Context, s *steadyServer, c *Cache) (Catalog, error) {
	return List(ctx, llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(s.URL),
		llmprovider.WithoutModelMetadata(), WithCache(c))
}

// TestListingFlight_SharesFailure_0053 (0053-MADR D4, M5): five concurrent
// Lists of one key, the listing failing after 100 ms, cost one request, and
// every caller is given the failure. Each waiter fetched again in turn: five
// requests, one after another.
func TestListingFlight_SharesFailure_0053(t *testing.T) {
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusUnauthorized} {
		s := newSteadyServer(t, 100*time.Millisecond, steadyReply{status: status})
		c := &Cache{}
		var wg sync.WaitGroup
		var failed atomic.Int32
		for range 5 {
			wg.Go(func() {
				cat, err := list0053(context.Background(), s, c)
				if err == nil && cat.Err != nil && cat.Source == ListStatic {
					failed.Add(1)
				}
			})
		}
		wg.Wait()
		if n := s.requests(); n != 1 || failed.Load() != 5 {
			t.Errorf("HTTP %d: %d requests, %d callers given the failure; want 1 and 5", status, n, failed.Load())
		}
	}
}

// TestListingFlight_CancelledClaimant_0053 (0053-MADR D4, M5): a caller that
// starts the fetch and is cancelled does not end it: a waiter that came
// after it receives the listing from the one request. The waiter fetched
// again: two requests.
func TestListingFlight_CancelledClaimant_0053(t *testing.T) {
	s := newSteadyServer(t, 100*time.Millisecond, steadyOpenAI)
	c := &Cache{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	claimed := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		close(claimed)
		_, _ = list0053(ctx, s, c)
	})
	<-claimed
	time.Sleep(5 * time.Millisecond)
	cat, err := list0053(context.Background(), s, c)
	wg.Wait()
	if err != nil || !cat.Live || s.requests() != 1 {
		t.Errorf("waiter: live %v, %v, after %d requests; want live from 1 request", cat.Live, err, s.requests())
	}
}

// ollamaServer0053 is a fake Ollama with one model, whose digest the test
// sets, counting the /api/show requests.
type ollamaServer0053 struct {
	*httptest.Server
	digest atomic.Value
	shows  atomic.Int32
}

func newOllamaServer0053(t *testing.T, digest string) *ollamaServer0053 {
	t.Helper()
	s := &ollamaServer0053{}
	s.digest.Store(digest)
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			entry := `{"name":"llama3.2:latest","capabilities":["completion","tools"]`
			if d := s.digest.Load().(string); d != "" {
				entry += `,"digest":"` + d + `"`
			}
			_, _ = io.WriteString(w, `{"models":[`+entry+`}]}`)
		case "/api/show":
			s.shows.Add(1)
			_, _ = io.WriteString(w, `{"model_info":{"llama.context_length":131072},"capabilities":["completion","tools"]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *ollamaServer0053) list(t *testing.T, c *Cache) {
	t.Helper()
	if _, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""),
		llmprovider.WithBaseURL(s.URL), llmprovider.WithoutModelMetadata(), WithCache(c)); err != nil {
		t.Fatalf("List: %v", err)
	}
}

// TestOllamaShow_AgePerReply_0053 (0053-MADR D5, M6): listings nine minutes
// apart ask /api/show at 0, 18 and 36 minutes, each reply living ten minutes
// from its own fetch. A listing restamped every reply it reused: one
// request in 36 minutes.
func TestOllamaShow_AgePerReply_0053(t *testing.T) {
	clock := useSteadyClock(t)
	s := newOllamaServer0053(t, "d1")
	c := &Cache{}
	for i := range 5 {
		if i > 0 {
			clock.advance(9 * time.Minute)
		}
		s.list(t, c)
	}
	if n := s.shows.Load(); n != 3 {
		t.Errorf("%d /api/show requests over five listings nine minutes apart; want 3", n)
	}
}

// TestOllamaShow_DigestChange_0053 (0053-MADR D5): a model whose digest
// changes is asked again at once; an unchanged digest is not; with no digest
// listed, the reply's age alone decides.
func TestOllamaShow_DigestChange_0053(t *testing.T) {
	clock := useSteadyClock(t)
	s := newOllamaServer0053(t, "d1")
	c := &Cache{}
	s.list(t, c)
	clock.advance(time.Minute)
	s.list(t, c)
	if n := s.shows.Load(); n != 1 {
		t.Errorf("an unchanged digest: %d /api/show requests; want 1", n)
	}
	s.digest.Store("d2")
	clock.advance(time.Minute)
	s.list(t, c)
	if n := s.shows.Load(); n != 2 {
		t.Errorf("a changed digest: %d /api/show requests; want 2", n)
	}
	s.digest.Store("")
	clock.advance(time.Minute)
	s.list(t, c)
	if n := s.shows.Load(); n != 2 {
		t.Errorf("no digest, the reply 1 minute old: %d /api/show requests; want 2", n)
	}
	clock.advance(10 * time.Minute)
	s.list(t, c)
	if n := s.shows.Load(); n != 3 {
		t.Errorf("no digest, the reply 11 minutes old: %d /api/show requests; want 3", n)
	}
}
