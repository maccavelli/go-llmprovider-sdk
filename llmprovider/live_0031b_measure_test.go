//go:build live_gateways

package llmprovider_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0031-PLAN-native-streaming-on-the-other-wires Phase 1's measurement (T2).
// It runs only with LLMPROVIDER_LIVE_0031B set. For each request it logs the
// status, the Content-Type, whether the stream ended with [DONE], and the
// shape of its events: each distinct key path (arrays collapsed as []) and
// the values of the keys that name an event or a stop, never text or a key.
// It fails only on a transport error.

// shapeKeys are the keys whose values are logged: they name an event or a
// stop, and carry no text.
var shapeKeys = map[string]bool{"type": true, "event": true, "event_type": true, "finish_reason": true,
	"finishReason": true, "stop_reason": true, "status": true, "object": true}

// wireRecorder turns each provider request into a streamed one, as its wire
// asks for a stream, and records the reply's shape.
type wireRecorder struct {
	wire        string // messages, chat, google or interactions
	usageOption bool   // chat: send stream_options.include_usage

	mu          sync.Mutex
	status      int
	contentType string
	done        bool
	shape       []string
}

func (r *wireRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodPost && req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}
		req = req.Clone(req.Context())
		switch r.wire {
		case "google":
			req.URL.Path = strings.Replace(req.URL.Path, ":generateContent", ":streamGenerateContent", 1)
			q := req.URL.Query()
			q.Set("alt", "sse")
			req.URL.RawQuery = q.Encode()
		case "chat":
			body["stream"] = true
			if r.usageOption {
				body["stream_options"] = map[string]any{"include_usage": true}
			}
		default:
			body["stream"] = true
		}
		if raw, err = json.Marshal(body); err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(raw))
		req.ContentLength = int64(len(raw))
		req.Header.Set("Content-Length", strconv.Itoa(len(raw)))
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(b))
	done, shape := streamShape(b)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status, r.contentType, r.done, r.shape = resp.StatusCode, resp.Header.Get("Content-Type"), done, shape
	return resp, nil
}

func (r *wireRecorder) result() (int, string, bool, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, r.contentType, r.done, r.shape
}

// streamShape is whether the body's data lines ended with [DONE], and the
// sorted distinct key paths and named values of its data payloads.
func streamShape(body []byte) (bool, []string) {
	seen := map[string]bool{}
	done := false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64<<10), 8<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		if name, ok := bytes.CutPrefix(line, []byte("event:")); ok {
			seen["event:"+strings.TrimSpace(string(name))] = true
			continue
		}
		payload, ok := bytes.CutPrefix(line, []byte("data:"))
		if !ok {
			continue
		}
		payload = bytes.TrimSpace(payload)
		if string(payload) == "[DONE]" {
			done = true
			continue
		}
		var v any
		if json.Unmarshal(payload, &v) == nil {
			walkShape("", v, 0, seen)
		}
	}
	shape := make([]string, 0, len(seen))
	for k := range seen {
		shape = append(shape, k)
	}
	sort.Strings(shape)
	return done, shape
}

// walkShape records v's key paths, and the values of shapeKeys.
func walkShape(path string, v any, depth int, seen map[string]bool) {
	if depth > 6 {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			p := k
			if path != "" {
				p = path + "." + k
			}
			seen[p] = true
			if s, ok := child.(string); ok && shapeKeys[k] {
				seen[p+"="+s] = true
			}
			walkShape(p, child, depth+1, seen)
		}
	case []any:
		for _, child := range t {
			walkShape(path+"[]", child, depth+1, seen)
		}
	}
}

// t2Target is one T2 target.
type t2Target struct {
	name     string
	wire     string
	id       llmprovider.ProviderID
	model    string
	keyVar   string               // the key's variable; empty when none is needed
	opts     []llmprovider.Option // beyond the key, model and client
	thinking *llmprovider.Reasoning
	maxThink int // max output tokens for the reasoning request
	tool     bool
}

// ollamaModel is the first model a local Ollama server lists, or "" when no
// server answers.
func ollamaModel() string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:11434/api/tags", nil)
	if err != nil {
		return ""
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.NewDecoder(resp.Body).Decode(&tags) != nil || len(tags.Models) == 0 {
		return ""
	}
	return tags.Models[0].Name
}

// TestLive_0031bStreamReplies is T2: does each of the four wires stream, with
// which events, and, for Chat Completions, does it stream usage. Billed: at
// most 26 requests (Ollama's are local).
func TestLive_0031bStreamReplies(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0031B") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0031B to run 0031-PLAN-native-streaming-on-the-other-wires Phase 1's measurement")
	}
	messagesThinking := &llmprovider.Reasoning{Budget: 1024}
	low := &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
	targets := []t2Target{
		{name: "claude", wire: "messages", id: llmprovider.ProviderClaude, model: catalog.Static(llmprovider.ProviderClaude)[0],
			keyVar: "ANTHROPIC_API_KEY", thinking: messagesThinking, maxThink: 1040, tool: true},
		{name: "opencode-go/messages", wire: "messages", id: llmprovider.ProviderOpencodeGo, model: "qwen3.8-flash",
			keyVar: "OPENCODE_API_KEY", opts: []llmprovider.Option{opencode.WithRoute(opencode.RouteMessages)},
			thinking: messagesThinking, maxThink: 1040},
		{name: "huggingface", wire: "chat", id: llmprovider.ProviderHuggingFace, model: catalog.Static(llmprovider.ProviderHuggingFace)[0],
			keyVar: "HF_TOKEN", thinking: low, maxThink: 16},
		{name: "kilo", wire: "chat", id: llmprovider.ProviderKilo, model: catalog.Static(llmprovider.ProviderKilo)[0],
			keyVar: "KILO_API_KEY", thinking: low, maxThink: 16},
		{name: "together", wire: "chat", id: llmprovider.ProviderTogether, model: catalog.Static(llmprovider.ProviderTogether)[0],
			keyVar: "TOGETHER_API_KEY", thinking: low, maxThink: 16, tool: true},
		{name: "opencode-go/chat", wire: "chat", id: llmprovider.ProviderOpencodeGo, model: "glm-5.3-flash",
			keyVar: "OPENCODE_API_KEY", opts: []llmprovider.Option{opencode.WithRoute(opencode.RouteChatCompletions)},
			thinking: low, maxThink: 16},
		{name: "ollama", wire: "chat", id: llmprovider.ProviderOllama, model: ollamaModel(), thinking: low, maxThink: 16},
		// Google's own generateContent endpoint, through the OpenCode
		// google route's request, which is Google's own shape and header.
		{name: "google-endpoint", wire: "google", id: llmprovider.ProviderOpencodeZen, model: catalog.Static(llmprovider.ProviderGemini)[0],
			keyVar: "GEMINI_API_KEY", opts: []llmprovider.Option{opencode.WithRoute(opencode.RouteGoogle),
				llmprovider.WithBaseURL("https://generativelanguage.googleapis.com/v1beta")},
			thinking: low, maxThink: 16, tool: true},
		{name: "opencode-zen/google", wire: "google", id: llmprovider.ProviderOpencodeZen, model: "gemini-3.5-flash-lite",
			keyVar: "OPENCODE_API_KEY", opts: []llmprovider.Option{opencode.WithRoute(opencode.RouteGoogle)},
			thinking: low, maxThink: 16},
		{name: "gemini", wire: "interactions", id: llmprovider.ProviderGemini, model: catalog.Static(llmprovider.ProviderGemini)[0],
			keyVar: "GEMINI_API_KEY", thinking: low, maxThink: 16, tool: true},
	}
	tool := llmprovider.Tool{Name: "record_city", Description: "Records a city name.",
		Schema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`)}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			if target.id == llmprovider.ProviderOllama && target.model == "" {
				t.Skip("no local Ollama server answered; not measured (0031-MADR Q6)")
			}
			var opts []llmprovider.Option
			if target.keyVar != "" {
				opts = append(opts, llmprovider.WithAPIKey(llmprovider.LiveEnvKey(t, target.keyVar)))
			}
			opts = append(opts, target.opts...)
			type run struct {
				name      string
				reasoning *llmprovider.Reasoning
				maxTokens int
				tool      bool
			}
			runs := []run{{"plain", nil, 16, false}, {"reasoning", target.thinking, target.maxThink, false}}
			if target.tool {
				runs = append(runs, run{"tool", nil, 64, true})
			}
			for _, r := range runs {
				req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser,
					Text: "Say hi, then record the city Paris."}}, Reasoning: r.reasoning}
				if r.tool {
					req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, llmprovider.ForceTool(tool.Name)
				}
				attempt := func(usageOption bool) (int, string, bool, []string, error) {
					rec := &wireRecorder{wire: target.wire, usageOption: usageOption}
					p, err := providers.New(target.id, append(slices.Clone(opts), llmprovider.WithModel(target.model),
						llmprovider.WithMaxTokens(r.maxTokens), llmprovider.WithModelProbes(false),
						llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...)
					if err != nil {
						t.Fatal(err)
					}
					ctx, cancel := llmprovider.LiveCtx(t)
					defer cancel()
					_, genErr := p.Generate(ctx, req)
					status, contentType, done, shape := rec.result()
					return status, contentType, done, shape, genErr
				}
				// Only the plain request asks for streamed usage (PLAN Phase 1).
				usageOption := target.wire == "chat" && r.name == "plain"
				status, contentType, done, shape, genErr := attempt(usageOption)
				if status == 0 {
					t.Fatalf("T2 %s %s (%s): no reply: %v", target.name, r.name, target.model, genErr)
				}
				t.Logf("T2 %s %s (%s): status %d, Content-Type %q, include_usage %v, [DONE] %v, shape %v",
					target.name, r.name, target.model, status, contentType, usageOption, done, shape)
				if status/100 != 2 {
					t.Logf("T2 %s %s: error: %v", target.name, r.name, genErr)
				}
				if usageOption && status/100 == 4 {
					// The gateway may refuse stream_options: once more without it (E5).
					status, contentType, done, shape, genErr = attempt(false)
					t.Logf("T2 %s %s (%s) without include_usage: status %d, Content-Type %q, [DONE] %v, shape %v",
						target.name, r.name, target.model, status, contentType, done, shape)
					if status/100 != 2 {
						t.Logf("T2 %s %s without include_usage: error: %v", target.name, r.name, genErr)
					}
				}
			}
		})
	}
}

// TestLive_0031bGeminiShapes is T2b (0031-PLAN-native-streaming-on-the-other-
// wires, deviation D2): T2's 16- and 64-token caps went wholly to thinking on
// the Gemini wires, so no call or thought-summary delta was seen. A reasoning
// and a forced-tool request at effort low and a 1,024-token cap, on Google's
// generateContent endpoint and on gemini's Interactions. Billed: 4 requests.
func TestLive_0031bGeminiShapes(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0031B") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0031B to run 0031-PLAN-native-streaming-on-the-other-wires Phase 1's measurement")
	}
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	model := catalog.Static(llmprovider.ProviderGemini)[0]
	tool := llmprovider.Tool{Name: "record_city", Description: "Records a city name.",
		Schema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`)}
	for _, target := range []struct {
		name string
		wire string
		id   llmprovider.ProviderID
		opts []llmprovider.Option
	}{
		{"google-endpoint", "google", llmprovider.ProviderOpencodeZen, []llmprovider.Option{opencode.WithRoute(opencode.RouteGoogle),
			llmprovider.WithBaseURL("https://generativelanguage.googleapis.com/v1beta")}},
		{"gemini", "interactions", llmprovider.ProviderGemini, nil},
	} {
		for _, withTool := range []bool{false, true} {
			name := map[bool]string{false: "reasoning", true: "tool"}[withTool]
			t.Run(target.name+"/"+name, func(t *testing.T) {
				rec := &wireRecorder{wire: target.wire}
				p, err := providers.New(target.id, append(slices.Clone(target.opts), llmprovider.WithAPIKey(key),
					llmprovider.WithModel(model), llmprovider.WithMaxTokens(1024), llmprovider.WithModelProbes(false),
					llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...)
				if err != nil {
					t.Fatal(err)
				}
				req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser,
					Text: "Say hi, then record the city Paris."}}, Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortLow}}
				if withTool {
					req.Tools, req.ToolChoice = []llmprovider.Tool{tool}, llmprovider.ForceTool(tool.Name)
				}
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				_, genErr := p.Generate(ctx, req)
				status, contentType, done, shape := rec.result()
				if status == 0 {
					t.Fatalf("T2b %s %s: no reply: %v", target.name, name, genErr)
				}
				t.Logf("T2b %s %s (%s): status %d, Content-Type %q, [DONE] %v, shape %v",
					target.name, name, model, status, contentType, done, shape)
				if status/100 != 2 {
					t.Logf("T2b %s %s: error: %v", target.name, name, genErr)
				}
			})
		}
	}
}
