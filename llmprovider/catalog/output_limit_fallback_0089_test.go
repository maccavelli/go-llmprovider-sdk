package catalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// fallbackServer records each POST's max_tokens and answers it with
// reply(n), n counting the POSTs from 0; a GET is a listing, answered with
// listing and counted.
type fallbackServer struct {
	URL   string
	mu    sync.Mutex
	sent  []float64
	lists int
}

func newFallbackServer(t *testing.T, listing string, reply func(n int) (int, string)) *fallbackServer {
	t.Helper()
	s := &fallbackServer{}
	srv := httptest.NewServer(wiretest.StreamReplies(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		s.mu.Lock()
		defer s.mu.Unlock()
		if r.Method != http.MethodPost {
			s.lists++
			_, _ = io.WriteString(w, listing)
			return
		}
		var body struct {
			MaxTokens float64 `json:"max_tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		n := len(s.sent)
		s.sent = append(s.sent, body.MaxTokens)
		status, answer := reply(n)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, answer)
	})))
	t.Cleanup(srv.Close)
	s.URL = srv.URL
	return s
}

// state is what the server saw: each POST's max_tokens, and the GETs.
func (s *fallbackServer) state() ([]float64, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]float64(nil), s.sent...), s.lists
}

const fallbackOK = `{"content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`

// fallbackRefusal is a Messages 400 carrying msg.
func fallbackRefusal(msg string) string {
	return fmt.Sprintf(`{"type":"error","error":{"type":"invalid_request_error","message":%q}}`, msg)
}

// TestOutputLimit_Fallback_0089 (OQ2 (1): DL50, DL51, DL77; 0088-MADR ST11,
// ST12, ST34, ST54): a request never waits for a listing: on a fresh cache
// it reads the embedded extract; a model no source states takes the
// smallest limit any current model on the route is stated to take, basis
// fallback, the retired not counted; a refusal stating its cap resends at
// it, and one stating no number is returned; on OpenCode's messages route
// the fallback counts only the models the shared route table sends there;
// and where no source states any limit, the request is refused before the
// network, saying how to state one. It is in catalog's external test
// package, which can build the Claude and OpenCode providers and replace
// the extract (0089-PLAN, annotated 2026-11-23).
func TestOutputLimit_Fallback_0089(t *testing.T) {
	ctx := context.Background()
	hi := func() *llmprovider.Request {
		return &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
	}
	base := func(url string, cache *catalog.Cache, model string) []llmprovider.Option {
		return []llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithoutModelMetadata(),
			llmprovider.WithModelProbes(false), catalog.WithCache(cache), llmprovider.WithBaseURL(url), llmprovider.WithModel(model)}
	}
	ok := func(int) (int, string) { return http.StatusOK, fallbackOK }

	// (1) A fresh cache: the first request reads the extract, and sends no
	// listing request.
	listing := `{"data":[{"id":"claude-haiku-4-5","max_tokens":32000},{"id":"claude-sonnet-fb-0089","max_tokens":64000},` +
		`{"id":"claude-opus-fb-0089","max_tokens":128000}],"has_more":false}`
	fresh := newFallbackServer(t, listing, ok)
	p, err := claude.New(base(fresh.URL, new(catalog.Cache), "claude-opus-4-6")...)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(ctx, hi())
	sent, lists := fresh.state()
	if err != nil || len(sent) != 1 || sent[0] != 128000 || lists != 0 {
		t.Errorf("a fresh cache: sent %v, %d listing request(s), err %v; want 128000 from the extract, no listing request",
			sent, lists, err)
	}
	if want := "models.dev (bundled 2026-10-12)"; err == nil && (resp.OutputLimit.Basis != llmprovider.LimitBasisFact ||
		resp.OutputLimit.Source != want) {
		t.Errorf("a fresh cache: OutputLimit = %+v; want basis fact from %s", resp.OutputLimit, want)
	}

	// (2) A model no source states, the listing cached: the smallest current
	// limit, 64,000; claude-haiku-4-5's 32,000 is retired.
	const unstated = "claude-fallback-0089"
	for _, tc := range []struct {
		name    string
		refusal string
		sent    []float64
		basis   llmprovider.LimitBasis
	}{
		{"the fallback", "", []float64{64000}, llmprovider.LimitBasisFallback},
		{"refused stating 4,096", "max_tokens: 64000 > 4096, which is the maximum allowed number of output tokens for " + unstated,
			[]float64{64000, 4096}, llmprovider.LimitBasisRefusal},
		{"refused stating no number", "max_tokens: the value is too large for this model", []float64{64000}, ""},
	} {
		reply := ok
		if tc.refusal != "" {
			reply = func(n int) (int, string) {
				if n == 0 {
					return http.StatusBadRequest, fallbackRefusal(tc.refusal)
				}
				return http.StatusOK, fallbackOK
			}
		}
		srv := newFallbackServer(t, listing, reply)
		p, err := claude.New(base(srv.URL, new(catalog.Cache), unstated)...)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.(llmprovider.ModelLister).ListModels(ctx); err != nil {
			t.Fatalf("%s: ListModels: %v", tc.name, err)
		}
		resp, err := p.Generate(ctx, hi())
		sent, _ := srv.state()
		if fmt.Sprint(sent) != fmt.Sprint(tc.sent) {
			t.Errorf("%s: sent %v; want %v", tc.name, sent, tc.sent)
		}
		switch {
		case tc.basis == "" && !errors.Is(err, llmprovider.ErrInvalidRequest):
			t.Errorf("%s: err %v; want the refusal returned", tc.name, err)
		case tc.basis != "" && (err != nil || resp.OutputLimit.Basis != tc.basis):
			t.Errorf("%s: %+v, %v; want basis %s", tc.name, resp, err, tc.basis)
		case tc.basis == llmprovider.LimitBasisFallback && resp.OutputLimit.Source != "listing:claude":
			t.Errorf("%s: OutputLimit = %+v; want the listing's evidence", tc.name, resp.OutputLimit)
		}
	}

	// (3) OpenCode's messages route, a model no source states: the smallest
	// limit among the models the route table sends there, from the extract
	// (DL77): 64,000 on Zen, 131,072 on Go.
	for _, gw := range []struct {
		name  string
		build func(...llmprovider.Option) (llmprovider.Provider, error)
		want  float64
	}{
		{"zen", opencode.NewZen, 64000},
		{"go", opencode.NewGo, 131072},
	} {
		srv := newFallbackServer(t, "", ok)
		p, err := gw.build(append(base(srv.URL, new(catalog.Cache), "claude-unlisted-0089"),
			opencode.WithRoute(opencode.RouteMessages))...)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := p.Generate(ctx, hi())
		sent, _ := srv.state()
		if err != nil || len(sent) != 1 || sent[0] != gw.want || resp.OutputLimit.Basis != llmprovider.LimitBasisFallback {
			t.Errorf("opencode %s messages, no source: sent %v, err %v; want %v, basis fallback", gw.name, sent, err, gw.want)
		}
	}

	// (4) No source states any limit for the model or the route's models:
	// refused before the network, saying how to state one (DL51).
	catalog.ReplaceExtract(t, `{"date":"2026-10-12","sections":{}}`)
	srv := newFallbackServer(t, "", ok)
	p, err = opencode.NewZen(append(base(srv.URL, new(catalog.Cache), "claude-nosource-0089"),
		opencode.WithRoute(opencode.RouteMessages))...)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Generate(ctx, hi())
	if sent, _ := srv.state(); len(sent) != 0 || !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("no source at all: %d request(s), err %v; want none sent and ErrInvalidRequest", len(sent), err)
	}
	if err == nil || !strings.Contains(err.Error(), "maxTokens") || !strings.Contains(err.Error(), "WithModelLayer") {
		t.Errorf("the refusal's message %q; want it to name maxTokens and WithModelLayer", err)
	}
}
