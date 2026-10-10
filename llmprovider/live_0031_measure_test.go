//go:build live_gateways

package llmprovider_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0031-PLAN Phase 1's measurement (T1). It runs only with
// LLMPROVIDER_LIVE_0031 set, and logs, per request, the status, the
// Content-Type and the event types of a streamed /responses reply: what D1's
// gate and D2's reasoning delta names are taken from. It never logs a
// payload's text or a key, and fails only on a transport error.

// streamRecorder turns each POST to /responses into a streamed one, as the
// provider sends it otherwise, and keeps the reply's status, Content-Type and
// the ordered, de-duplicated event types of its body.
type streamRecorder struct {
	mu          sync.Mutex
	status      int
	contentType string
	types       []string
}

func (r *streamRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/responses") && req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		body["stream"] = true
		if raw, err = json.Marshal(body); err != nil {
			return nil, err
		}
		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(raw))
		req.ContentLength = int64(len(raw))
		req.Header.Set("Content-Length", strconv.Itoa(len(raw)))
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(b))
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status, r.contentType, r.types = resp.StatusCode, resp.Header.Get("Content-Type"), eventTypes(b)
	return resp, nil
}

// eventTypes is each distinct "type" of the stream's data payloads, in order
// of first appearance.
func eventTypes(body []byte) []string {
	var types []string
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for scanner.Scan() {
		payload, ok := bytes.CutPrefix(scanner.Bytes(), []byte("data:"))
		if !ok {
			continue
		}
		var event struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(bytes.TrimSpace(payload), &event) == nil && event.Type != "" && !slices.Contains(types, event.Type) {
			types = append(types, event.Type)
		}
	}
	return types
}

func (r *streamRecorder) result() (int, string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, r.contentType, r.types
}

// responsesModel is the model a gateway routes to /responses: the first in
// its static catalog whose route in routes_snapshot.json is @ai-sdk/openai,
// else the snapshot's first such "nano" model by name, else its first by name.
func responsesModel(t *testing.T, id llmprovider.ProviderID) string {
	t.Helper()
	raw, err := os.ReadFile("providers/opencode/routes_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]map[string]string
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	key := map[llmprovider.ProviderID]string{llmprovider.ProviderOpencodeZen: "opencode", llmprovider.ProviderOpencodeGo: "opencode-go"}[id]
	routes := snapshot[key]
	for _, model := range catalog.Static(id) {
		if routes[model] == "@ai-sdk/openai" {
			return model
		}
	}
	var listed []string
	for model, route := range routes {
		if route == "@ai-sdk/openai" {
			listed = append(listed, model)
		}
	}
	if len(listed) == 0 {
		t.Fatalf("no responses-route model in the %s snapshot", key)
	}
	slices.Sort(listed)
	if i := slices.IndexFunc(listed, func(m string) bool { return strings.Contains(m, "nano") }); i >= 0 {
		return listed[i]
	}
	return listed[0]
}

// stream0031Target is one T1 target: how to build it, and its plain and
// reasoning models.
type stream0031Target struct {
	id        llmprovider.ProviderID
	opts      func(t *testing.T) []llmprovider.Option
	plain     string
	reasoning string
}

func apiKeyOpts(id llmprovider.ProviderID) func(t *testing.T) []llmprovider.Option {
	return func(t *testing.T) []llmprovider.Option {
		return []llmprovider.Option{
			llmprovider.WithAPIKey(llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id])),
			llmprovider.WithMaxTokens(16),
		}
	}
}

// TestLive_0031StreamReplies is T1: does each Responses speaker stream on
// /responses, and with which event types. Billed: two requests of at most 16
// output tokens per target, plus two on a ChatGPT session when
// LLMPROVIDER_LIVE_CHATGPT=1.
func TestLive_0031StreamReplies(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0031") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0031 to run 0031-PLAN Phase 1's measurement")
	}
	targets := []stream0031Target{
		{id: llmprovider.ProviderOpenAI, opts: apiKeyOpts(llmprovider.ProviderOpenAI), plain: "gpt-4.1-mini", reasoning: "o4-mini"},
		{id: llmprovider.ProviderGrok, opts: apiKeyOpts(llmprovider.ProviderGrok), plain: catalog.Static(llmprovider.ProviderGrok)[0], reasoning: "grok-3-mini"},
		{id: llmprovider.ProviderOpencodeZen, opts: apiKeyOpts(llmprovider.ProviderOpencodeZen)},
		{id: llmprovider.ProviderOpencodeGo, opts: apiKeyOpts(llmprovider.ProviderOpencodeGo)},
		{id: "chatgpt", opts: func(t *testing.T) []llmprovider.Option {
			return []llmprovider.Option{llmprovider.WithTokenSource(liveChatGPTSession(t))}
		}, plain: "gpt-5.4-mini", reasoning: "gpt-5.4-mini"},
	}
	for _, target := range targets {
		t.Run(string(target.id), func(t *testing.T) {
			opts := target.opts(t)
			id, plain, reasoning := target.id, target.plain, target.reasoning
			if id == "chatgpt" {
				id = llmprovider.ProviderOpenAI
			}
			if plain == "" {
				plain = responsesModel(t, id)
				reasoning = plain
			}
			for _, run := range []struct {
				name   string
				model  string
				effort llmprovider.Effort
			}{{"plain", plain, ""}, {"reasoning", reasoning, llmprovider.EffortLow}} {
				rec := &streamRecorder{}
				p, err := providers.New(id, append(opts, llmprovider.WithModel(run.model),
					llmprovider.WithModelProbes(false), llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...)
				if err != nil {
					t.Fatal(err)
				}
				req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
				if run.effort != "" {
					req.Reasoning = &llmprovider.Reasoning{Effort: run.effort}
				}
				ctx, cancel := llmprovider.LiveCtx(t)
				_, genErr := p.Generate(ctx, req)
				cancel()
				status, contentType, types := rec.result()
				if status == 0 {
					t.Fatalf("T1 %s %s (%s): no reply: %v", target.id, run.name, run.model, genErr)
				}
				t.Logf("T1 %s %s (%s): status %d, Content-Type %q, events %v", target.id, run.name, run.model, status, contentType, types)
				if status/100 != 2 {
					// The error is the provider's, redacted and bounded (D-A10).
					t.Logf("T1 %s %s: error: %v", target.id, run.name, genErr)
				}
			}
		})
	}
}
