package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// rawShape is 0041-MADR D9's table: where a provider's page keeps its model
// array ("" for a bare array), and which field is the provider's id.
type rawShape struct{ array, id string }

var rawShapes = map[string]rawShape{
	"claude": {"data", "id"}, "openai": {"data", "id"}, "chatgpt": {"models", "slug"}, "gemini": {"models", "name"},
	"grok": {"data", "id"}, "opencode-zen": {"data", "id"}, "opencode-go": {"data", "id"}, "kilo": {"data", "id"},
	"huggingface": {"data", "id"}, "together": {"", "id"}, "ollama": {"models", "name"},
}

// rawProvider is the provider a recorded envelope lists.
var rawProvider = map[string]llmprovider.ProviderID{
	"claude": llmprovider.ProviderClaude, "openai": llmprovider.ProviderOpenAI, "chatgpt": llmprovider.ProviderOpenAI,
	"gemini": llmprovider.ProviderGemini, "grok": llmprovider.ProviderGrok, "opencode-zen": llmprovider.ProviderOpencodeZen,
	"opencode-go": llmprovider.ProviderOpencodeGo, "kilo": llmprovider.ProviderKilo,
	"huggingface": llmprovider.ProviderHuggingFace, "together": llmprovider.ProviderTogether,
	"ollama": llmprovider.ProviderOllama,
}

// rawEnvelopes reads each recorded listing, its pages as the provider sent
// them: the 2026-10-12 ranking snapshot, the 2026-10-14 first-party
// recording, and the 2026-10-15 recording of a ChatGPT session's and a local
// Ollama's (0041-PLAN Phase 8).
func rawEnvelopes(t *testing.T) map[string][][]byte {
	t.Helper()
	read := func(parts ...string) []byte {
		raw, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, parts...)...))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	out := map[string][][]byte{}
	for name, file := range map[string]string{"opencode-zen": "zen.json", "opencode-go": "go.json", "kilo": "kilo.json",
		"huggingface": "hf.json", "together": "together.json"} {
		out[name] = [][]byte{read("ranking-2026-10-12", file)}
	}
	for _, name := range []string{"claude", "gemini", "grok", "openai"} {
		var pages []json.RawMessage
		if err := json.Unmarshal(read("firstparty-2026-10-14", name+".json"), &pages); err != nil {
			t.Fatal(err)
		}
		for _, p := range pages {
			out[name] = append(out[name], []byte(p))
		}
	}
	for _, name := range []string{"chatgpt", "ollama"} {
		out[name] = [][]byte{read("rawlistings-2026-10-15", name+".json")}
	}
	return out
}

// pagedServer answers the n-th request with pages[n], the last repeated,
// and records each request.
type pagedServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []*http.Request
}

func newPagedServer(t *testing.T, pages ...[]byte) *pagedServer {
	t.Helper()
	s := &pagedServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		n := len(s.requests)
		s.requests = append(s.requests, r.Clone(context.Background()))
		s.mu.Unlock()
		_, _ = w.Write(pages[min(n, len(pages)-1)])
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *pagedServer) sent() []*http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// chatGPTSession is a ChatGPT login, for the Codex backend's listing.
func chatGPTSession() llmprovider.TokenSource {
	return &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)}
}

// rawSource is name's credential: a ChatGPT session for chatgpt, none for
// Ollama, a key for the rest.
func rawSource(name string) llmprovider.TokenSource {
	switch name {
	case "chatgpt":
		return chatGPTSession()
	case "ollama":
		return llmprovider.NewStaticToken("")
	}
	return llmprovider.NewStaticToken("k")
}

// splitPage is the test's own reading of a page's model array, from D9's
// table.
func splitPage(t *testing.T, shape rawShape, page []byte) []json.RawMessage {
	t.Helper()
	var models []json.RawMessage
	if shape.array == "" {
		if err := json.Unmarshal(page, &models); err != nil {
			t.Fatal(err)
		}
		return models
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal(page, &env); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(env[shape.array], &models); err != nil {
		t.Fatal(err)
	}
	return models
}

// TestListRaw_Verbatim_0041: for every provider, on its recorded envelope,
// Pages are the served bodies byte for byte, Models is one entry per array
// element, each the element's bytes, and ID is the provider's own field
// (0041-MADR D9).
func TestListRaw_Verbatim_0041(t *testing.T) {
	for name, pages := range rawEnvelopes(t) {
		t.Run(name, func(t *testing.T) {
			s := newPagedServer(t, pages...)
			got, err := ListRaw(context.Background(), rawProvider[name], rawSource(name), llmprovider.WithBaseURL(s.URL))
			if err != nil {
				t.Fatalf("ListRaw: %v", err)
			}
			if got.Provider != rawProvider[name] || got.FetchedAt.IsZero() || strings.Contains(got.Endpoint, "?") ||
				!strings.HasPrefix(got.Endpoint, s.URL) {
				t.Errorf("Provider %s, Endpoint %q, FetchedAt %v", got.Provider, got.Endpoint, got.FetchedAt)
			}
			if len(got.Pages) != len(pages) {
				t.Fatalf("%d pages; want %d", len(got.Pages), len(pages))
			}
			var want []json.RawMessage
			for i, p := range pages {
				if !bytes.Equal(got.Pages[i], p) {
					t.Errorf("page %d differs from the served body", i)
				}
				want = append(want, splitPage(t, rawShapes[name], p)...)
			}
			if len(got.Models) != len(want) || len(want) == 0 {
				t.Fatalf("%d models; want %d, and at least one", len(got.Models), len(want))
			}
			for i, m := range got.Models {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(want[i], &fields); err != nil {
					t.Fatal(err)
				}
				var id string
				_ = json.Unmarshal(fields[rawShapes[name].id], &id)
				if !bytes.Equal(m.Raw, want[i]) || m.ID != id {
					t.Errorf("model %d: ID %q, %d bytes; want %q, the element's %d bytes", i, m.ID, len(m.Raw), id, len(want[i]))
				}
			}
		})
	}
}

// TestListRaw_NoCuration_0041: an embedding model, an image model, a
// duplicate id, an unknown field, and an entry with no id all come back, in
// the body's order, the last with an empty ID (0041-MADR D9).
func TestListRaw_NoCuration_0041(t *testing.T) {
	body := []byte(`{"object":"list","data":[{"id":"text-embedding-3-large","x-new":1},{"id":"gpt-image-2"},` +
		`{"id":"gpt-4o"},{"id":"gpt-4o"},{"id":"dall-e-2"},{"object":"model"}],"x-envelope":true}`)
	s := newPagedServer(t, body)
	got, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(s.URL))
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got.Models))
	for i, m := range got.Models {
		ids[i] = m.ID
	}
	if want := []string{"text-embedding-3-large", "gpt-image-2", "gpt-4o", "gpt-4o", "dall-e-2", ""}; !slices.Equal(ids, want) {
		t.Errorf("ids %q; want every entry in order %q", ids, want)
	}
	if string(got.Models[0].Raw) != `{"id":"text-embedding-3-large","x-new":1}` || !bytes.Equal(got.Pages[0], body) {
		t.Errorf("the unknown fields were not kept: %s", got.Models[0].Raw)
	}
}

// TestListRaw_Pages_0041: Claude's pages by after_id and Gemini's by
// pageToken come back in order; an eleventh page is an error (0041-MADR D9).
func TestListRaw_Pages_0041(t *testing.T) {
	claude := [][]byte{
		[]byte(`{"data":[{"id":"c1"}],"has_more":true,"last_id":"c1"}`),
		[]byte(`{"data":[{"id":"c2"}],"has_more":true,"last_id":"c2"}`),
		[]byte(`{"data":[{"id":"c3"}],"has_more":false,"last_id":"c3"}`),
	}
	gemini := [][]byte{
		[]byte(`{"models":[{"name":"models/g1"}],"nextPageToken":"p2"}`),
		[]byte(`{"models":[{"name":"models/g2"}],"nextPageToken":"p3"}`),
		[]byte(`{"models":[{"name":"models/g3"}]}`),
	}
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		pages    [][]byte
		ids      []string
		query    string
	}{
		{llmprovider.ProviderClaude, claude, []string{"c1", "c2", "c3"}, "after_id"},
		{llmprovider.ProviderGemini, gemini, []string{"models/g1", "models/g2", "models/g3"}, "pageToken"},
	} {
		s := newPagedServer(t, tc.pages...)
		got, err := ListRaw(context.Background(), tc.provider, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(s.URL))
		if err != nil {
			t.Fatalf("%s: %v", tc.provider, err)
		}
		ids := make([]string, len(got.Models))
		for i, m := range got.Models {
			ids[i] = m.ID
		}
		if !slices.Equal(ids, tc.ids) || len(got.Pages) != 3 {
			t.Errorf("%s: ids %q over %d pages; want %q over 3", tc.provider, ids, len(got.Pages), tc.ids)
		}
		if sent := s.sent(); len(sent) != 3 || !sent[2].URL.Query().Has(tc.query) {
			t.Errorf("%s: %d requests; want 3, the last with %s", tc.provider, len(sent), tc.query)
		}
	}
	endless := newPagedServer(t, []byte(`{"data":[{"id":"c"}],"has_more":true,"last_id":"c"}`))
	if _, err := ListRaw(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(endless.URL)); err == nil {
		t.Error("an eleventh page: no error")
	}
}

// TestListRaw_Bound_0041: a page over 8 MiB is an error (0041-MADR D9).
func TestListRaw_Bound_0041(t *testing.T) {
	srv := paddedServer(t, 9<<20)
	got, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(srv.URL))
	if err == nil || !strings.Contains(err.Error(), "8 MiB") || got.Pages != nil {
		t.Errorf("ListRaw = %d pages, %v; want an error naming the 8 MiB bound", len(got.Pages), err)
	}
}

// byteCountingBody counts the bytes read from a reply's body.
type byteCountingBody struct {
	io.ReadCloser
	n *atomic.Int64
}

func (b byteCountingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.n.Add(int64(n))
	return n, err
}

// byteCountingTransport hands each reply's body on through a byteCountingBody.
type byteCountingTransport struct{ n *atomic.Int64 }

func (c byteCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		resp.Body = byteCountingBody{resp.Body, c.n}
	}
	return resp, err
}

// TestListRaw_ReadsBounded_0041: ListRaw's recording reads a page no further
// than one byte past the 8 MiB bound, so a page of any size costs no more
// than the bound to refuse.
func TestListRaw_ReadsBounded_0041(t *testing.T) {
	chunk := bytes.Repeat([]byte("a"), 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"pad":"`))
		for range 32 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	var read atomic.Int64
	_, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(srv.URL), llmprovider.WithHTTPClient(&http.Client{Transport: byteCountingTransport{&read}}))
	if err == nil || read.Load() > listingPageLimit+1 {
		t.Errorf("a 32 MiB page: %d bytes read, %v; want an error and at most %d bytes", read.Load(), err, listingPageLimit+1)
	}
}

// TestListRaw_NoFallback_0041: a 503 is a zero RawListing and the error; a
// 401 renews an invalidating source once, then returns the error
// (0041-MADR D9).
func TestListRaw_NoFallback_0041(t *testing.T) {
	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(unavailable.Close)
	got, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(unavailable.URL))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || got.Provider != "" || got.Pages != nil {
		t.Errorf("503: %+v, %v; want a zero RawListing and ErrProviderUnavailable", got, err)
	}
	var requests int
	var mu sync.Mutex
	refused := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(refused.Close)
	src := &invalidatingSource{}
	if _, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, src, llmprovider.WithBaseURL(refused.URL)); !errors.Is(err,
		llmprovider.ErrAuthFailure) {
		t.Errorf("401: %v; want ErrAuthFailure", err)
	}
	if requests != 2 || src.invalidated.Load() != 1 {
		t.Errorf("401: %d requests, %d renewals; want 2 and 1", requests, src.invalidated.Load())
	}
}

// TestListRaw_NoSideEffects_0041: ListRaw fetches no models.dev document,
// and leaves nothing CachedFacts can read (0041-MADR D9).
func TestListRaw_NoSideEffects_0041(t *testing.T) {
	resetLookupCaches(t)
	meta, served := metadataServer(t, http.StatusOK, `{}`)
	s := newPagedServer(t, []byte(`{"data":[{"id":"vendor/raw-x","context_length":131072,"pricing":{"prompt":"0.000001"}}]}`))
	opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), llmprovider.WithModelMetadataURL(meta.URL)}
	if _, err := ListRaw(context.Background(), llmprovider.ProviderKilo, llmprovider.NewStaticToken("k"), opts...); err != nil {
		t.Fatal(err)
	}
	if n := served.Load(); n != 0 {
		t.Errorf("%d models.dev requests; want 0", n)
	}
	if info, ok := CachedFacts(context.Background(), llmprovider.ProviderKilo, "vendor/raw-x",
		append(opts, llmprovider.WithAPIKey("k"))...); ok && info.Has(FieldContextWindow) {
		t.Errorf("CachedFacts knows the listed model after ListRaw: %+v", info)
	}
}

// TestListRaw_SameTransportAsList_0041: ListRaw and List send the same
// method, path, query and credential header (0041-MADR D9).
func TestListRaw_SameTransportAsList_0041(t *testing.T) {
	envelopes := rawEnvelopes(t)
	for name, pages := range envelopes {
		if name == "chatgpt" {
			continue // List does not list a ChatGPT session: TestListRaw_ChatGPT_0041
		}
		t.Run(name, func(t *testing.T) {
			resetLookupCaches(t)
			listSrv, rawSrv := newPagedServer(t, pages...), newPagedServer(t, pages...)
			if _, err := listT(context.Background(), rawProvider[name], rawSource(name), llmprovider.WithBaseURL(listSrv.URL)); err != nil {
				t.Fatal(err)
			}
			if _, err := ListRaw(context.Background(), rawProvider[name], rawSource(name), llmprovider.WithBaseURL(rawSrv.URL)); err != nil {
				t.Fatal(err)
			}
			describe := func(r *http.Request) string {
				return fmt.Sprintf("%s %s?%s %q %q %q", r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"),
					r.Header.Get("x-api-key"), r.Header.Get("x-goog-api-key"))
			}
			// List's Ollama listing also asks /api/show per model: only its
			// listing requests are compared.
			var listed []string
			for _, r := range listSrv.sent() {
				if r.URL.Path != "/api/show" {
					listed = append(listed, describe(r))
				}
			}
			var raw []string
			for _, r := range rawSrv.sent() {
				raw = append(raw, describe(r))
			}
			if !slices.Equal(listed, raw) {
				t.Errorf("List sent %q; ListRaw %q", listed, raw)
			}
		})
	}
}

// TestListRaw_ChatGPT_0041: a ChatGPT session lists the Codex backend with
// client_version, and gets every entry back unsorted and unfiltered,
// visibility and supported_in_api included, and one whose slug is not a
// string with an empty ID (0041-MADR D9).
func TestListRaw_ChatGPT_0041(t *testing.T) {
	body := []byte(`{"models":[{"slug":"gpt-low","priority":9,"visibility":"list"},` +
		`{"slug":"gpt-hidden","priority":1,"visibility":"hide"},{"slug":"gpt-off","priority":2,"supported_in_api":false},` +
		`{"slug":7}]}`)
	s := newPagedServer(t, body)
	got, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, chatGPTSession(), llmprovider.WithBaseURL(s.URL))
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got.Models))
	for i, m := range got.Models {
		ids[i] = m.ID
	}
	if want := []string{"gpt-low", "gpt-hidden", "gpt-off", ""}; !slices.Equal(ids, want) {
		t.Errorf("slugs %q; want every entry in the body's order %q", ids, want)
	}
	sent := s.sent()
	if len(sent) != 1 || sent[0].URL.Path != "/models" || !sent[0].URL.Query().Has("client_version") ||
		sent[0].Header.Get("originator") == "" || sent[0].Header.Get("Authorization") != "Bearer a" {
		t.Errorf("requests %d: %v; want one GET /models?client_version=…, with the session's headers", len(sent), sent)
	}
}

// TestListRaw_Ollama_0041: Ollama's listing is /api/tags, with name as the
// id, and no /api/show (0041-MADR D9).
func TestListRaw_Ollama_0041(t *testing.T) {
	s := newPagedServer(t, []byte(`{"models":[{"name":"llama3:latest","model":"llama3:latest"}]}`))
	got, err := ListRaw(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""),
		llmprovider.WithBaseURL(s.URL))
	if err != nil {
		t.Fatal(err)
	}
	sent := s.sent()
	if len(sent) != 1 || sent[0].URL.Path != "/api/tags" || len(got.Models) != 1 || got.Models[0].ID != "llama3:latest" {
		t.Errorf("requests %v, models %+v; want one /api/tags and llama3:latest", sent, got.Models)
	}
}

// TestListRaw_FailedPageNamed_0041: a second page that is not JSON, or that
// fails, is an error naming page 2, the failure's class kept (0041-MADR D9).
func TestListRaw_FailedPageNamed_0041(t *testing.T) {
	first := []byte(`{"data":[{"id":"c1"}],"has_more":true,"last_id":"c1"}`)
	for _, tc := range []struct {
		name   string
		status int
		body   []byte
		is     error
	}{
		{"not JSON", http.StatusOK, []byte(`<html>`), nil},
		{"503", http.StatusServiceUnavailable, nil, llmprovider.ErrProviderUnavailable},
	} {
		var n atomic.Int64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if n.Add(1) == 1 {
				_, _ = w.Write(first)
				return
			}
			w.WriteHeader(tc.status)
			_, _ = w.Write(tc.body)
		}))
		got, err := ListRaw(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
			llmprovider.WithBaseURL(srv.URL))
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), "claude page 2") || got.Pages != nil ||
			(tc.is != nil && !errors.Is(err, tc.is)) {
			t.Errorf("%s on page 2: %d pages, %v; want an error naming page 2", tc.name, len(got.Pages), err)
		}
	}
}

// TestListRaw_Refusals_0041: an unknown provider, a missing source and a
// page with no model array are errors.
func TestListRaw_Refusals_0041(t *testing.T) {
	if _, err := ListRaw(context.Background(), "nope", llmprovider.NewStaticToken("k")); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Errorf("unknown provider: %v; want ErrUnsupported", err)
	}
	if _, err := ListRaw(context.Background(), llmprovider.ProviderOpenAI, nil); err == nil {
		t.Error("nil source: no error")
	}
	s := newPagedServer(t, []byte(`{"object":"list","items":[]}`))
	if _, err := ListRaw(context.Background(), llmprovider.ProviderKilo, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(s.URL)); err == nil || !strings.Contains(err.Error(), "page 1") {
		t.Errorf("no model array: %v; want an error naming page 1", err)
	}
}
