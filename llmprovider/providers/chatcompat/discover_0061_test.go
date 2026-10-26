package chatcompat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0061-PLAN Phase 4e: Discover (0061-MADR E5; its amendment "E5's ports and a
// keyed server, confirmed before Phase 4e").

// stubServer is a loopback server whose GET /v1/models answers with status
// and body, recording each request's host and Authorization header.
type stubServer struct {
	*httptest.Server
	mu    sync.Mutex
	hosts []string
	auth  []string
}

func newStub(t *testing.T, status int, body string) *stubServer {
	t.Helper()
	s := &stubServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.hosts = append(s.hosts, r.Host)
		s.auth = append(s.auth, r.Header.Get("Authorization"))
		s.mu.Unlock()
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

// port is s's port.
func (s *stubServer) port(t *testing.T) int {
	t.Helper()
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// withPorts makes Discover ask ports alone, for t.
func withPorts(t *testing.T, ports ...int) {
	t.Helper()
	saved := discoverPorts
	discoverPorts = ports
	t.Cleanup(func() { discoverPorts = saved })
}

// refusedPort is a loopback port nothing listens on, internal/nettest's
// RefusedURL's: never a freed port, which another package's server may take
// (0045-MADR).
func refusedPort(t *testing.T) int {
	t.Helper()
	u, err := url.Parse(nettest.RefusedURL)
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

const ollamaListing = `{"object":"list","data":[{"id":"qwen3:8b","object":"model","created":1783878353,"owned_by":"library"},` +
	`{"id":"gemma3:12b","object":"model","created":1783179663,"owned_by":"library"}]}`

// TestDiscover_Loopback_0061: each port is asked on 127.0.0.1, and the
// servers that answer with a listing are returned in the ports' order, each
// labelled by its dialect or its host and port; a port nothing listens on,
// a 404 and a reply that is no listing are none.
func TestDiscover_Loopback_0061(t *testing.T) {
	ollama := newStub(t, http.StatusOK, ollamaListing)
	plain := newStub(t, http.StatusOK, `{"data":[{"id":"m1"},{"id":""},{"id":"m2"}]}`)
	empty := newStub(t, http.StatusOK, `{"data":[]}`)
	notFound := newStub(t, http.StatusNotFound, `{}`)
	webPage := newStub(t, http.StatusOK, `<html>hello</html>`)
	noData := newStub(t, http.StatusOK, `{"models":[]}`)
	withPorts(t, plain.port(t), refusedPort(t), ollama.port(t), notFound.port(t), webPage.port(t), noData.port(t), empty.port(t))
	got, err := Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	plainHost := "127.0.0.1:" + strconv.Itoa(plain.port(t))
	emptyHost := "127.0.0.1:" + strconv.Itoa(empty.port(t))
	want := []Server{
		{BaseURL: "http://" + plainHost + "/v1", Label: plainHost, Models: []string{"m1", "m2"}},
		{BaseURL: "http://127.0.0.1:" + strconv.Itoa(ollama.port(t)) + "/v1", Label: "Ollama", Models: []string{"qwen3:8b", "gemma3:12b"}},
		{BaseURL: "http://" + emptyHost + "/v1", Label: emptyHost},
	}
	if !slices.EqualFunc(got, want, func(a, b Server) bool {
		return a.BaseURL == b.BaseURL && a.Label == b.Label && slices.Equal(a.Models, b.Models) && a.NeedsKey == b.NeedsKey
	}) {
		t.Errorf("Discover = %+v\nwant %+v", got, want)
	}
	for _, s := range []*stubServer{ollama, plain, notFound, webPage} {
		for _, h := range s.hosts {
			if !strings.HasPrefix(h, "127.0.0.1:") {
				t.Errorf("asked host %q, want 127.0.0.1 only", h)
			}
		}
	}
}

// TestDiscover_NoKey_0061: no Authorization header is sent, though the
// options hold a key.
func TestDiscover_NoKey_0061(t *testing.T) {
	s := newStub(t, http.StatusOK, ollamaListing)
	withPorts(t, s.port(t))
	if _, err := Discover(context.Background(), llmprovider.WithAPIKey("sk-test")); err != nil {
		t.Fatal(err)
	}
	if len(s.auth) != 1 || s.auth[0] != "" {
		t.Errorf("Authorization headers %q, want one request with none", s.auth)
	}
}

// TestDiscover_Keyed_0061: a 401 or a 403 is a server that needs a key,
// with no models; a 500 is none.
func TestDiscover_Keyed_0061(t *testing.T) {
	unauthorized := newStub(t, http.StatusUnauthorized, `{"error":"missing key"}`)
	forbidden := newStub(t, http.StatusForbidden, ``)
	failing := newStub(t, http.StatusInternalServerError, ``)
	withPorts(t, unauthorized.port(t), forbidden.port(t), failing.port(t))
	got, err := Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].NeedsKey || !got[1].NeedsKey || got[0].Models != nil || got[1].Models != nil ||
		got[0].BaseURL != "http://127.0.0.1:"+strconv.Itoa(unauthorized.port(t))+"/v1" {
		t.Errorf("Discover = %+v; want the 401 and the 403, each needing a key, with no models", got)
	}
}

// TestDiscover_Bound_0061: a port that does not answer is given up after
// the bound, and the others are returned; an ended ctx is Discover's error;
// a refused option is too.
func TestDiscover_Bound_0061(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// At most 3 s: past the test's 2 s, so that a Discover with no bound
		// fails the test rather than hanging it.
		select {
		case <-release:
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	t.Cleanup(slow.Close)
	t.Cleanup(func() { close(release) })
	u, err := url.Parse(slow.URL)
	if err != nil {
		t.Fatal(err)
	}
	slowPort, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	fast := newStub(t, http.StatusOK, ollamaListing)
	withPorts(t, slowPort, fast.port(t))
	start := time.Now()
	got, err := Discover(context.Background())
	if elapsed := time.Since(start); err != nil || len(got) != 1 || got[0].Label != "Ollama" || elapsed > 2*time.Second {
		t.Errorf("Discover = %+v, %v after %v; want the fast server alone, within the bound", got, err, elapsed)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Discover(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("an ended ctx: %v, want context.Canceled", err)
	}
	if _, err := Discover(context.Background(), llmprovider.ScopedOption("other", "other.Option", 1)); !errors.Is(err,
		llmprovider.ErrInvalidRequest) {
		t.Errorf("a refused option: %v, want ErrInvalidRequest", err)
	}
}
