package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// ollama0042Model is one installed model of a fake Ollama: the capabilities
// JSON of its /api/tags entry ("" leaves the field out), and of its
// /api/show reply ("" for a reply without the field, "500" for a failure).
type ollama0042Model struct{ name, tags, show string }

// ollama0042Server is a fake Ollama serving models, counting the /api/show
// requests for each.
type ollama0042Server struct {
	*httptest.Server
	mu    sync.Mutex
	shows map[string]int
}

func newOllama0042Server(t *testing.T, models ...ollama0042Model) *ollama0042Server {
	t.Helper()
	s := &ollama0042Server{shows: map[string]int{}}
	byName := map[string]ollama0042Model{}
	for _, m := range models {
		byName[m.name] = m
	}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			parts := make([]string, 0, len(models))
			for _, m := range models {
				entry := fmt.Sprintf(`{"name":%q`, m.name)
				if m.tags != "" {
					entry += `,"capabilities":` + m.tags
				}
				parts = append(parts, entry+"}")
			}
			_, _ = io.WriteString(w, `{"models":[`+strings.Join(parts, ",")+`]}`)
		case "/api/show":
			var body struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			s.mu.Lock()
			s.shows[body.Model]++
			s.mu.Unlock()
			switch c := byName[body.Model].show; c {
			case "500":
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
			case "":
				_, _ = io.WriteString(w, `{"model_info":{}}`)
			default:
				_, _ = io.WriteString(w, `{"model_info":{},"capabilities":`+c+`}`)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

// asked is the /api/show requests so far, by model.
func (s *ollama0042Server) asked() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.shows)
}

// list is List for Ollama against s, as prepare-commit-msg lists it.
func (s *ollama0042Server) list(t *testing.T, opts ...llmprovider.Option) Catalog {
	t.Helper()
	opts = append([]llmprovider.Option{llmprovider.WithBaseURL(s.URL), llmprovider.WithoutModelMetadata(),
		WithLimit(MaxListed)}, opts...)
	cat, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), opts...)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return cat
}

// The capabilities the cases use, from 0042-MADR section 3.
const (
	capsEmbedding = `["embedding"]`
	capsChat      = `["completion","tools"]`
	capsCoder     = `["completion","tools","insert"]`
)

// ollama0042S1 is the issue's install: an embedding model listed first.
var ollama0042S1 = []ollama0042Model{
	{"granite-embedding:30m", "", capsEmbedding},
	{"qwen3-vl:8b", "", `["completion","vision","tools","thinking"]`},
	{"llama3.1:8b-instruct-q5_K_M", "", capsChat},
	{"qwen2.5-coder:7b", "", capsCoder},
}

// TestOllamaInfo_Output_0042: an /api/show reply's capabilities say what
// the model writes, text for completion and an image for image; a missing
// or empty list says nothing at all (0042-MADR D1).
func TestOllamaInfo_Output_0042(t *testing.T) {
	for _, tc := range []struct {
		capabilities string
		output       []Modality
		known, tools bool
	}{
		{capsCoder, []Modality{ModalityText}, true, true},
		{capsEmbedding, nil, true, false},
		{`["image"]`, []Modality{ModalityImage}, true, false},
		{`["completion","image"]`, []Modality{ModalityText, ModalityImage}, true, false},
		{`["decision"]`, nil, true, false},
		{"", nil, false, false},
		{`[]`, nil, false, false},
	} {
		body := `{"model_info":{}}`
		if tc.capabilities != "" {
			body = `{"model_info":{},"capabilities":` + tc.capabilities + `}`
		}
		var show ollamaShowReply
		if err := json.Unmarshal([]byte(body), &show); err != nil {
			t.Fatal(err)
		}
		info := ollamaInfo("m", show)
		if info.Has(FieldOutput) != tc.known || !slices.Equal(info.Output, tc.output) {
			t.Errorf("%q: Output %v, known %t; want %v, known %t", tc.capabilities, info.Output, info.Has(FieldOutput),
				tc.output, tc.known)
		}
		if info.Has(FieldTools) != tc.known || info.Tools != tc.tools {
			t.Errorf("%q: Tools %t, known %t; want %t, known %t", tc.capabilities, info.Tools, info.Has(FieldTools),
				tc.tools, tc.known)
		}
		if tc.known && info.SourceOf(FieldOutput) != "listing:ollama" {
			t.Errorf("%q: Output's source %q; want listing:ollama", tc.capabilities, info.SourceOf(FieldOutput))
		}
	}
}

// TestOllamaUsable_0042: Ollama's listing leaves out each model Ollama says
// writes no text, from /api/show or, where that fails, /api/tags, and keeps
// a model it cannot judge; a caller's layer decides over both (0042-MADR
// D1, PLAN Phase 1).
func TestOllamaUsable_0042(t *testing.T) {
	embedAsText := staticLayer{name: "caller",
		info: ModelInfo{ID: "granite-embedding:30m", Output: []Modality{ModalityText}}.Known(FieldOutput)}
	chats := []string{"qwen3-vl:8b", "llama3.1:8b-instruct-q5_K_M", "qwen2.5-coder:7b"}
	for _, tc := range []struct {
		name                      string
		models                    []ollama0042Model
		opts                      []llmprovider.Option
		usable, recommended, menu []string
		noTools                   []string
		asked                     int
	}{
		{name: "S1", models: ollama0042S1, usable: chats, recommended: chats, menu: chats, asked: 4},
		{name: "S2", models: []ollama0042Model{{"gemma3:12b", "", `["completion","vision"]`}, {"codellama:7b", "", `["completion","insert"]`}},
			usable: []string{"gemma3:12b", "codellama:7b"}, menu: []string{"gemma3:12b", "codellama:7b"},
			noTools: []string{"gemma3:12b", "codellama:7b"}, asked: 2},
		{name: "S3", models: []ollama0042Model{{"embed-tools:1", "", `["tools","embedding"]`}, {"chat:1", "", capsChat}},
			usable: []string{"chat:1"}, recommended: []string{"chat:1"}, menu: []string{"chat:1"}, asked: 2},
		{name: "S4", models: []ollama0042Model{{"granite-embedding:30m", "", "500"}, {"chat:1", "", capsChat}},
			usable: []string{"granite-embedding:30m", "chat:1"}, recommended: []string{"granite-embedding:30m", "chat:1"},
			menu: []string{"granite-embedding:30m", "chat:1"}, asked: 2},
		{name: "S4b", models: []ollama0042Model{{"granite-embedding:30m", capsEmbedding, "500"}, {"chat:1", capsChat, capsChat}},
			usable: []string{"chat:1"}, recommended: []string{"chat:1"}, menu: []string{"chat:1"}, asked: 2},
		{name: "S5", models: []ollama0042Model{{"nomic-embed-text:latest", "", ""}, {"chat:1", "", ""}},
			usable: []string{"nomic-embed-text:latest", "chat:1"}, recommended: []string{"nomic-embed-text:latest", "chat:1"},
			menu: []string{"nomic-embed-text:latest", "chat:1"}, asked: 2},
		{name: "S6", models: []ollama0042Model{{"x/flux2:1", "", `["image"]`}, {"classifier:1", "", `["decision"]`}, {"chat:1", "", capsChat}},
			usable: []string{"chat:1"}, recommended: []string{"chat:1"}, menu: []string{"chat:1"}, asked: 3},
		{name: "S7", models: []ollama0042Model{{"granite-embedding:30m", capsEmbedding, capsEmbedding}, {"qwen2.5-coder:7b", capsCoder, capsCoder}},
			usable: []string{"qwen2.5-coder:7b"}, recommended: []string{"qwen2.5-coder:7b"}, menu: []string{"qwen2.5-coder:7b"}, asked: 2},
		{name: "S8", models: []ollama0042Model{{"granite-embedding:30m", "", capsEmbedding}, {"nomic-embed-text:latest", "", capsEmbedding}},
			asked: 2},
		{name: "S9", models: ollama0042S1, opts: []llmprovider.Option{WithModelLayer(embedAsText)},
			usable: append([]string{"granite-embedding:30m"}, chats...), recommended: chats,
			menu: append(slices.Clone(chats), "granite-embedding:30m"), noTools: []string{"granite-embedding:30m"}, asked: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetLookupCaches(t)
			s := newOllama0042Server(t, tc.models...)
			cat := s.list(t, tc.opts...)
			if !cat.Live || cat.Err != nil {
				t.Errorf("Live %t, Err %v; want a live listing", cat.Live, cat.Err)
			}
			if !slices.Equal(cat.Usable, tc.usable) {
				t.Errorf("Usable %q; want %q", cat.Usable, tc.usable)
			}
			if !slices.Equal(cat.Recommended, tc.recommended) {
				t.Errorf("Recommended %q; want %q", cat.Recommended, tc.recommended)
			}
			if got := menuIDs(cat.Menu); !slices.Equal(got, tc.menu) {
				t.Errorf("Menu %q; want %q", got, tc.menu)
			}
			for _, e := range cat.Menu {
				var want []Caveat
				if slices.Contains(tc.noTools, e.ID) {
					want = []Caveat{CaveatNoTools}
				}
				if !slices.Equal(e.Caveats, want) {
					t.Errorf("%s: caveats %v; want %v", e.ID, e.Caveats, want)
				}
			}
			asked := 0
			for _, n := range s.asked() {
				asked += n
			}
			if asked != tc.asked {
				t.Errorf("/api/show asked %d times; want once for each of %d models", asked, tc.asked)
			}
		})
	}
}

// TestOllamaUsable_TagsNotCached_0042: what /api/tags says of a model whose
// /api/show failed decides one listing and is never cached, so the next
// listing asks /api/show again; a reply is cached as before (0042-MADR D1).
func TestOllamaUsable_TagsNotCached_0042(t *testing.T) {
	resetLookupCaches(t)
	s := newOllama0042Server(t, ollama0042Model{"granite-embedding:30m", capsEmbedding, "500"},
		ollama0042Model{"chat:1", capsChat, capsChat})
	if cat := s.list(t); !slices.Equal(cat.Usable, []string{"chat:1"}) {
		t.Fatalf("Usable %q; want [chat:1]", cat.Usable)
	}
	cached, _ := cachedListing(listingKeyFor(llmprovider.ProviderOllama, config{BaseURL: s.URL}, llmprovider.Token{}))
	if _, ok := cached["chat:1"]; !ok {
		t.Errorf("chat:1's reply is not cached: %v", slices.Sorted(maps.Keys(cached)))
	}
	if _, ok := cached["granite-embedding:30m"]; ok {
		t.Error("the embedding model is cached from /api/tags, though its /api/show failed")
	}
	s.list(t)
	if got, want := s.asked(), map[string]int{"granite-embedding:30m": 2, "chat:1": 1}; !maps.Equal(got, want) {
		t.Errorf("/api/show asked %v after two listings; want %v", got, want)
	}
}
