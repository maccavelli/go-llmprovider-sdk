//go:build live_gateways

package llmprovider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0033-PLAN-refresh-the-curated-catalogs Phase 1's measurements. They run
// only with LLMPROVIDER_LIVE_0033 set. Each key is read from its provider's
// variable and sent only in its request header; nothing but statuses, ids and
// listing metadata is logged. They fail only on a transport error.

// candidates0033 are the lists the owner approved for measurement (0033-MADR
// Q8), by provider.
var candidates0033 = map[llmprovider.ProviderID][]string{
	llmprovider.ProviderOpenAI: {"gpt-6-luna", "gpt-5.6-luna", "gpt-4.1-mini", "gpt-5.6-terra", "gpt-6.1-sol", "gpt-6-astra"},
	llmprovider.ProviderGemini: {"gemini-3.5-flash-lite", "gemini-3.8-flash", "gemini-3.6-flash"},
	llmprovider.ProviderClaude: {"claude-haiku-5-5", "claude-sonnet-5", "claude-opus-4-8", "claude-sonnet-4-6",
		"claude-sonnet-5-5", "claude-opus-5-5"},
	llmprovider.ProviderGrok: {"grok-4.6", "grok-4.5", "grok-4.7", "grok-4.3", "grok-build-0.1"},
}

func require0033(t *testing.T) {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_0033") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0033 to run 0033-PLAN-refresh-the-curated-catalogs Phase 1's measurements")
	}
}

// measured0033 is every id a provider's listing is checked for: its
// candidates, then today's static ids.
func measured0033(id llmprovider.ProviderID) []string {
	ids := slices.Clone(candidates0033[id])
	for _, s := range catalog.Static(id) {
		if !slices.Contains(ids, s) {
			ids = append(ids, s)
		}
	}
	return ids
}

// getListing GETs u with the header set, and returns the status and body.
func getListing(t *testing.T, u string, header map[string]string) (int, []byte) {
	t.Helper()
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", req.URL.Redacted(), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, body
}

// compact is raw JSON on one line, cut at n bytes.
func compact(raw json.RawMessage, n int) string {
	s := strings.Join(strings.Fields(string(raw)), " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	if s == "" {
		return "absent"
	}
	return s
}

// logListed logs, for each measured id, its listing entry's fields, or that
// the listing lacks it.
func logListed(t *testing.T, name string, ids []string, entries map[string]string) {
	t.Helper()
	for _, id := range ids {
		if fields, ok := entries[id]; ok {
			t.Logf("L %s %s: listed; %s", name, id, fields)
		} else {
			t.Logf("L %s %s: NOT LISTED", name, id)
		}
	}
}

// TestLive_0033Listings is step 1: each vendor's listing, for each candidate
// and each of today's static ids. Not billed.
func TestLive_0033Listings(t *testing.T) {
	require0033(t)
	t.Run("openai", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderOpenAI])
		status, body := getListing(t, "https://api.openai.com/v1/models", map[string]string{"Authorization": "Bearer " + key})
		var r struct {
			Data []struct {
				ID           string          `json:"id"`
				ShutdownDate json.RawMessage `json:"shutdown_date"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatalf("openai listing: status %d: %v", status, err)
		}
		entries := map[string]string{}
		for _, m := range r.Data {
			entries[m.ID] = "shutdown_date " + compact(m.ShutdownDate, 40)
		}
		t.Logf("L openai: status %d, %d models", status, len(r.Data))
		logListed(t, "openai", measured0033(llmprovider.ProviderOpenAI), entries)
	})
	t.Run("claude", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderClaude])
		entries, n, after := map[string]string{}, 0, ""
		for range 10 {
			u := "https://api.anthropic.com/v1/models?limit=1000"
			if after != "" {
				u += "&after_id=" + url.QueryEscape(after)
			}
			status, body := getListing(t, u, map[string]string{"x-api-key": key, "anthropic-version": "2023-06-01"})
			var r struct {
				Data []struct {
					ID             string          `json:"id"`
					Lifecycle      json.RawMessage `json:"lifecycle"`
					MaxInputTokens json.RawMessage `json:"max_input_tokens"`
					MaxTokens      json.RawMessage `json:"max_tokens"`
					Capabilities   json.RawMessage `json:"capabilities"`
				} `json:"data"`
				HasMore bool   `json:"has_more"`
				LastID  string `json:"last_id"`
			}
			if err := json.Unmarshal(body, &r); err != nil {
				t.Fatalf("claude listing: status %d: %v", status, err)
			}
			for _, m := range r.Data {
				entries[m.ID] = fmt.Sprintf("lifecycle %s, max_input_tokens %s, max_tokens %s, capabilities %s",
					compact(m.Lifecycle, 120), compact(m.MaxInputTokens, 20), compact(m.MaxTokens, 20), compact(m.Capabilities, 900))
			}
			n += len(r.Data)
			if !r.HasMore || r.LastID == "" {
				break
			}
			after = r.LastID
		}
		t.Logf("L claude: %d models", n)
		logListed(t, "claude", measured0033(llmprovider.ProviderClaude), entries)
	})
	t.Run("gemini", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGemini])
		entries, n, token := map[string]string{}, 0, ""
		for range 10 {
			u := "https://generativelanguage.googleapis.com/v1beta/models?pageSize=1000"
			if token != "" {
				u += "&pageToken=" + url.QueryEscape(token)
			}
			status, body := getListing(t, u, map[string]string{"x-goog-api-key": key})
			var r struct {
				Models []struct {
					Name             string          `json:"name"`
					InputTokenLimit  json.RawMessage `json:"inputTokenLimit"`
					OutputTokenLimit json.RawMessage `json:"outputTokenLimit"`
					Thinking         json.RawMessage `json:"thinking"`
				} `json:"models"`
				NextPageToken string `json:"nextPageToken"`
			}
			if err := json.Unmarshal(body, &r); err != nil {
				t.Fatalf("gemini listing: status %d: %v", status, err)
			}
			for _, m := range r.Models {
				entries[strings.TrimPrefix(m.Name, "models/")] = fmt.Sprintf("inputTokenLimit %s, outputTokenLimit %s, thinking %s",
					compact(m.InputTokenLimit, 20), compact(m.OutputTokenLimit, 20), compact(m.Thinking, 20))
			}
			n += len(r.Models)
			if r.NextPageToken == "" {
				break
			}
			token = r.NextPageToken
		}
		t.Logf("L gemini: %d models", n)
		logListed(t, "gemini", measured0033(llmprovider.ProviderGemini), entries)
	})
	t.Run("grok", func(t *testing.T) {
		key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGrok])
		status, body := getListing(t, "https://api.x.ai/v1/models", map[string]string{"Authorization": "Bearer " + key})
		var r struct {
			Data []struct {
				ID            string          `json:"id"`
				Aliases       []string        `json:"aliases"`
				ContextLength json.RawMessage `json:"context_length"`
				Capabilities  json.RawMessage `json:"capabilities"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			t.Fatalf("grok listing: status %d: %v", status, err)
		}
		entries := map[string]string{}
		for _, m := range r.Data {
			fields := fmt.Sprintf("context_length %s, capabilities %s", compact(m.ContextLength, 20), compact(m.Capabilities, 300))
			entries[m.ID] = fields
			for _, a := range m.Aliases {
				entries[a] = "alias of " + m.ID + "; " + fields
			}
		}
		t.Logf("L grok: status %d, %d models", status, len(r.Data))
		logListed(t, "grok", measured0033(llmprovider.ProviderGrok), entries)
	})
}

// probe0033 builds provider id for model through a recording transport that
// applies rewrite, and returns it with the recorder.
func probe0033(t *testing.T, id llmprovider.ProviderID, model string, rewrite func(map[string]any)) (llmprovider.Provider, *flagRecorder) {
	t.Helper()
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id])
	if rewrite == nil {
		rewrite = func(map[string]any) {}
	}
	rec := &flagRecorder{rewrite: rewrite}
	p, err := providers.New(id, llmprovider.WithAPIKey(key), llmprovider.WithModel(model), llmprovider.WithMaxTokens(16),
		llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
	if err != nil {
		t.Fatal(err)
	}
	return p, rec
}

// effortProbes0033 are the Grok models whose efforts step 4 measures; their
// generation is measured there, so step 2 leaves them out.
var effortProbes0033 = map[string][]string{
	"grok-4.7": {"low", "medium", "high", "xhigh"},
	"grok-4.3": {"low", "medium", "high", "xhigh"},
}

// TestLive_0033Generate is step 2: one generation of at most 16 output tokens
// per candidate. Billed: 18 requests.
func TestLive_0033Generate(t *testing.T) {
	require0033(t)
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGemini,
		llmprovider.ProviderClaude, llmprovider.ProviderGrok} {
		for _, model := range candidates0033[id] {
			if _, ok := effortProbes0033[model]; ok {
				continue
			}
			t.Run(string(id)+"/"+model, func(t *testing.T) {
				p, rec := probe0033(t, id, model, nil)
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				_, genErr := p.Generate(ctx, &llmprovider.Request{
					Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."}}})
				status, _ := rec.result()
				if status == 0 {
					t.Fatalf("G %s %s: no reply: %v", id, model, genErr)
				}
				t.Logf("G %s %s: status %d, error: %v", id, model, status, genErr)
			})
		}
	}
}

// TestLive_0033ForcedChoice is step 3: GenerateToolCall, which forces its
// tool, on each Claude candidate. Billed: 6 requests.
func TestLive_0033ForcedChoice(t *testing.T) {
	require0033(t)
	for _, model := range candidates0033[llmprovider.ProviderClaude] {
		t.Run(model, func(t *testing.T) {
			p, rec := probe0033(t, llmprovider.ProviderClaude, model, nil)
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			_, genErr := llmprovider.GenerateToolCall(ctx, p, &llmprovider.Request{Tools: []llmprovider.Tool{cityTool},
				Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Record the city Paris."}}})
			status, _ := rec.result()
			if status == 0 {
				t.Fatalf("F %s: no reply: %v", model, genErr)
			}
			t.Logf("F claude %s: forced tool choice: status %d, error: %v", model, status, genErr)
		})
	}
}

// TestLive_0033GrokEfforts is step 4: each documented effort, and xhigh, on
// grok-4.7 and grok-4.3, set on the wire as reasoning.effort. Billed: 8
// requests.
func TestLive_0033GrokEfforts(t *testing.T) {
	require0033(t)
	for _, model := range []string{"grok-4.7", "grok-4.3"} {
		for _, effort := range effortProbes0033[model] {
			t.Run(model+"/"+effort, func(t *testing.T) {
				p, rec := probe0033(t, llmprovider.ProviderGrok, model, func(body map[string]any) { body["reasoning"] = map[string]any{"effort": effort} })
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				_, genErr := p.Generate(ctx, &llmprovider.Request{
					Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."}}})
				status, _ := rec.result()
				if status == 0 {
					t.Fatalf("E %s %s: no reply: %v", model, effort, genErr)
				}
				t.Logf("E grok %s reasoning.effort %s: status %d, error: %v", model, effort, status, genErr)
			})
		}
	}
}
