package llmprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

// togetherCapture records the last request a stub Together server received.
type togetherCapture struct {
	mu     sync.Mutex
	path   string
	header http.Header
	body   map[string]any
}

func (c *togetherCapture) last() (string, http.Header, map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.path, c.header, c.body
}

// togetherServer answers /models with listing and everything else with reply.
func togetherServer(t *testing.T, reply, listing string) (*httptest.Server, *togetherCapture) {
	t.Helper()
	c := &togetherCapture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/models" {
			_, _ = io.WriteString(w, listing)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		c.mu.Lock()
		c.path, c.header, c.body = r.URL.Path, r.Header.Clone(), body
		c.mu.Unlock()
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, c
}

const togetherText = `{"id":"t-1","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}]}`

// TestTogether_RequestShapes pins MADR 0017 D1: the plain path sends no
// reasoning field; the thinking path sends reasoning {"enabled": true}, and
// reasoning_effort only when an effort was set; a forced tool pins
// tool_choice; the key is a bearer on POST /chat/completions with max_tokens.
func TestTogether_RequestShapes(t *testing.T) {
	enabled := map[string]any{"enabled": true}
	for _, tc := range []struct {
		name          string
		effort        string
		thinking      bool
		tool          bool
		wantReasoning any
		wantEffort    any
	}{
		{name: "plain"},
		{name: "thinking", thinking: true, wantReasoning: enabled},
		{name: "thinking with effort", effort: "high", thinking: true, wantReasoning: enabled, wantEffort: "high"},
		{name: "effort set, plain path", effort: "high"},
		{name: "forced tool", tool: true},
		{name: "forced tool, thinking", tool: true, thinking: true, wantReasoning: enabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, c := togetherServer(t, togetherText, "[]")
			p, err := NewTogether("tg-key", "openai/gpt-oss-120b", WithBaseURL(srv.URL), WithReasoningEffort(tc.effort))
			if err != nil {
				t.Fatal(err)
			}
			var tool *Tool
			if tc.tool {
				tool = &weatherTool
			}
			if _, err := p.doGenerateItems(context.Background(), []Item{MessageItem{Role: jsonRoleUser, Text: "hi"}}, tool, tc.thinking); err != nil {
				t.Fatalf("generate: %v", err)
			}
			path, header, body := c.last()
			if path != "/chat/completions" || header.Get("Authorization") != "Bearer tg-key" {
				t.Errorf("path %q, Authorization %q", path, header.Get("Authorization"))
			}
			if body["max_tokens"] != float64(8192) || body["model"] != "openai/gpt-oss-120b" {
				t.Errorf("max_tokens %v, model %v", body["max_tokens"], body["model"])
			}
			assertThinkingFields(t, body, map[string]any{"reasoning": tc.wantReasoning, "reasoning_effort": tc.wantEffort})
			choice, forced := body["tool_choice"]
			if forced != tc.tool {
				t.Errorf("tool_choice present = %v, want %v (%v)", forced, tc.tool, choice)
			}
		})
	}
}

// TestTogether_EOSIsAStop: finish_reason "eos" keeps its text and is no error;
// only "length" is truncation.
func TestTogether_EOSIsAStop(t *testing.T) {
	srv, _ := togetherServer(t, `{"choices":[{"finish_reason":"eos","message":{"role":"assistant","content":"done"}}]}`, "[]")
	p, err := NewTogether("k", "m", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.GenerateItems(context.Background(), MessageItem{Role: jsonRoleUser, Text: "hi"})
	if err != nil || resp.OutputText() != "done" || resp.FinishReason != "eos" {
		t.Fatalf("GenerateItems = %+v, %v; want text \"done\", finish \"eos\", no error", resp, err)
	}
}

// TestTogether_Listing: GET /models is a bare array; only "chat" models are
// kept, the static order leads, and a failed listing falls back to the
// static catalog. No generation is sent.
func TestTogether_Listing(t *testing.T) {
	listing := `[{"id":"openai/gpt-oss-120b","type":"chat"},{"id":"BAAI/bge-large-en-v1.5","type":"embedding"},` +
		`{"id":"black-forest-labs/FLUX.2","type":"image"},{"id":"zai-org/GLM-5.3","type":"chat"},{"id":"new/Chat-Model","type":"chat"}]`
	srv, c := togetherServer(t, togetherText, listing)
	p, err := NewTogether("k", "m", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.DiscoverModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"zai-org/GLM-5.3", "openai/gpt-oss-120b", "new/Chat-Model"}; !slices.Equal(got, want) {
		t.Errorf("DiscoverModels = %v, want %v", got, want)
	}
	if path, _, _ := c.last(); path != "" {
		t.Errorf("listing sent a generation to %q", path)
	}

	for name, body := range map[string]string{"not an array": `{"data":[]}`, "no chat model": `[{"id":"x","type":"image"}]`} {
		t.Run(name, func(t *testing.T) {
			srv, _ := togetherServer(t, togetherText, body)
			p, err := NewTogether("k", "m", WithBaseURL(srv.URL))
			if err != nil {
				t.Fatal(err)
			}
			got, err := p.DiscoverModels(context.Background())
			if err != nil || !slices.Equal(got, staticTogether) {
				t.Errorf("DiscoverModels = %v, %v; want the static catalog", got, err)
			}
		})
	}
}

func TestTogether_RequiresKey(t *testing.T) {
	if _, err := NewTogether("", "m"); err == nil {
		t.Error("NewTogether with no key: want an error")
	}
}
