//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// 0032-PLAN Phase 1's measurements. They run only with LLMPROVIDER_LIVE_0032
// set. A recording transport rewrites each provider's own request to carry
// the flag being measured, and logs the reply's status and, where the reply
// echoes its tools, the strict value applied; never text or a key. They fail
// only on a transport error.

// flagRecorder rewrites a request body with rewrite, and keeps the reply.
type flagRecorder struct {
	rewrite func(body map[string]any)

	mu     sync.Mutex
	status int
	body   []byte
}

func (r *flagRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
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
		r.rewrite(body)
		if raw, err = json.Marshal(body); err != nil {
			return nil, err
		}
		req = req.Clone(req.Context())
		req.Body = io.NopCloser(bytes.NewReader(raw))
		req.ContentLength = int64(len(raw))
		req.Header.Set("Content-Length", strconv.Itoa(len(raw)))
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
	r.status, r.body = resp.StatusCode, b
	return resp, nil
}

func (r *flagRecorder) result() (int, []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, r.body
}

// cityTool has an optional property, which strict mode would make required.
var cityTool = llmprovider.Tool{Name: "record_city", Description: "Records a city name.",
	Schema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"},"country":{"type":"string"}},"required":["city"]}`)}

// eachTool calls f with each tool definition of body.
func eachTool(body map[string]any, f func(tool map[string]any)) {
	tools, _ := body["tools"].([]any)
	for _, t := range tools {
		if tool, ok := t.(map[string]any); ok {
			f(tool)
		}
	}
}

// echoedStrict is the strict value of each tool a Responses JSON reply
// echoes, or "none" when it echoes none.
func echoedStrict(reply []byte) string {
	var r struct {
		Tools []struct {
			Strict *bool `json:"strict"`
		} `json:"tools"`
	}
	if json.Unmarshal(reply, &r) != nil || len(r.Tools) == 0 {
		return "none"
	}
	var vals []string
	for _, t := range r.Tools {
		if t.Strict == nil {
			vals = append(vals, "absent")
		} else {
			vals = append(vals, strconv.FormatBool(*t.Strict))
		}
	}
	return strings.Join(vals, ",")
}

func require0032(t *testing.T) {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_0032") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0032 to run 0032-PLAN Phase 1's measurements")
	}
}

// TestLive_0032Strict is T1: does each Responses and Chat Completions target
// accept strict: false on a tool, and, where it echoes, apply it. Billed: 7
// requests of at most 64 output tokens.
func TestLive_0032Strict(t *testing.T) {
	require0032(t)
	responses := func(body map[string]any) { eachTool(body, func(tool map[string]any) { tool["strict"] = false }) }
	chat := func(body map[string]any) {
		eachTool(body, func(tool map[string]any) {
			if fn, ok := tool["function"].(map[string]any); ok {
				fn["strict"] = false
			}
		})
	}
	for _, target := range []struct {
		name    string
		id      llmprovider.ProviderID
		model   string
		rewrite func(map[string]any)
		opts    []llmprovider.Option
	}{
		{"openai", llmprovider.ProviderOpenAI, "gpt-4.1-mini", responses, nil},
		{"grok", llmprovider.ProviderGrok, "grok-4.6", responses, nil},
		{"opencode-go/responses", llmprovider.ProviderOpencodeGo, "gpt-6-luna", responses,
			[]llmprovider.Option{opencode.WithRoute(opencode.RouteResponses)}},
		{"huggingface", llmprovider.ProviderHuggingFace, catalog.Static(llmprovider.ProviderHuggingFace)[0], chat, nil},
		{"kilo", llmprovider.ProviderKilo, catalog.Static(llmprovider.ProviderKilo)[0], chat, nil},
		{"together", llmprovider.ProviderTogether, catalog.Static(llmprovider.ProviderTogether)[0], chat, nil},
		{"opencode-go/chat", llmprovider.ProviderOpencodeGo, "glm-5.3-flash", chat,
			[]llmprovider.Option{opencode.WithRoute(opencode.RouteChatCompletions)}},
	} {
		t.Run(target.name, func(t *testing.T) {
			key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[target.id])
			rec := &flagRecorder{rewrite: target.rewrite}
			p, err := providers.New(target.id, append([]llmprovider.Option{llmprovider.WithAPIKey(key),
				llmprovider.WithModel(target.model), llmprovider.WithMaxTokens(64), llmprovider.WithModelProbes(false),
				llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: rec})}, target.opts...)...)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			_, genErr := p.Generate(ctx, &llmprovider.Request{Tools: []llmprovider.Tool{cityTool},
				Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Record the city Paris."}}})
			status, reply := rec.result()
			if status == 0 {
				t.Fatalf("T1 %s: no reply: %v", target.name, genErr)
			}
			t.Logf("T1 %s (%s): status %d, echoed strict %s", target.name, target.model, status, echoedStrict(reply))
			if status/100 != 2 {
				t.Logf("T1 %s: error: %v", target.name, genErr)
			}
		})
	}
}

// failedTurn is a request whose last item is a failed result of a call to
// cityTool.
func failedTurn(callID string) *llmprovider.Request {
	return &llmprovider.Request{Tools: []llmprovider.Tool{cityTool}, Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Record the city Paris, then say whether it worked."},
		llmprovider.FunctionCallItem{CallID: callID, Name: cityTool.Name, Arguments: `{"city":"Paris"}`},
		llmprovider.FunctionCallOutputItem{CallID: callID, Output: "the city registry is offline"},
	}}
}

// mentionsFailure reports whether a reply's text says the call failed, as a
// yes or no; the text itself is not logged.
func mentionsFailure(resp *llmprovider.Response) string {
	if resp == nil {
		return "no reply"
	}
	text := strings.ToLower(resp.OutputText())
	for _, w := range []string{"offline", "fail", "error", "unable", "could not", "couldn't", "didn't", "did not", "not "} {
		if strings.Contains(text, w) {
			return "yes"
		}
	}
	return "no"
}

// TestLive_0032ResultError is T2: does each wire with a documented error form
// accept a failed tool result in it. Billed: 3 requests.
func TestLive_0032ResultError(t *testing.T) {
	require0032(t)
	messages := func(body map[string]any) {
		msgs, _ := body["messages"].([]any)
		for _, m := range msgs {
			msg, _ := m.(map[string]any)
			blocks, _ := msg["content"].([]any)
			for _, b := range blocks {
				if block, ok := b.(map[string]any); ok && block["type"] == "tool_result" {
					block["is_error"] = true
				}
			}
		}
	}
	google := func(body map[string]any) {
		contents, _ := body["contents"].([]any)
		for _, c := range contents {
			content, _ := c.(map[string]any)
			parts, _ := content["parts"].([]any)
			for _, p := range parts {
				part, _ := p.(map[string]any)
				if fr, ok := part["functionResponse"].(map[string]any); ok {
					if resp, ok := fr["response"].(map[string]any); ok {
						fr["response"] = map[string]any{"error": resp["output"]}
					}
				}
			}
		}
	}
	interactions := func(body map[string]any) {
		steps, _ := body["input"].([]any)
		for _, s := range steps {
			if step, ok := s.(map[string]any); ok && step["type"] == "function_result" {
				step["is_error"] = true
			}
		}
	}
	low := &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
	for _, target := range []struct {
		name      string
		id        llmprovider.ProviderID
		keyVar    string
		model     string
		rewrite   func(map[string]any)
		reasoning *llmprovider.Reasoning
		maxTokens int
		opts      []llmprovider.Option
	}{
		{"claude", llmprovider.ProviderClaude, "ANTHROPIC_API_KEY", "claude-haiku-4-5", messages, nil, 64, nil},
		{"google-endpoint", llmprovider.ProviderOpencodeZen, "GEMINI_API_KEY", catalog.Static(llmprovider.ProviderGemini)[0], google, low, 1024,
			[]llmprovider.Option{opencode.WithRoute(opencode.RouteGoogle), llmprovider.WithBaseURL("https://generativelanguage.googleapis.com/v1beta")}},
		{"gemini", llmprovider.ProviderGemini, "GEMINI_API_KEY", catalog.Static(llmprovider.ProviderGemini)[0], interactions, low, 1024, nil},
	} {
		t.Run(target.name, func(t *testing.T) {
			key := llmprovider.LiveEnvKey(t, target.keyVar)
			rec := &flagRecorder{rewrite: target.rewrite}
			p, err := providers.New(target.id, append([]llmprovider.Option{llmprovider.WithAPIKey(key),
				llmprovider.WithModel(target.model), llmprovider.WithMaxTokens(target.maxTokens), llmprovider.WithModelProbes(false),
				llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: rec})}, target.opts...)...)
			if err != nil {
				t.Fatal(err)
			}
			req := failedTurn("call_0032")
			req.Reasoning = target.reasoning
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			resp, genErr := p.Generate(ctx, req)
			status, _ := rec.result()
			if status == 0 {
				t.Fatalf("T2 %s: no reply: %v", target.name, genErr)
			}
			t.Logf("T2 %s (%s): status %d, the reply says the call failed: %s", target.name, target.model, status, mentionsFailure(resp))
			if status/100 != 2 {
				t.Logf("T2 %s: error: %v", target.name, genErr)
			}
		})
	}
}

// TestLive_0032StrictHuggingFaceControl is T1c: Hugging Face answered T1's
// strict request with a 500. The same request once more with strict: false,
// and once without it, tells a refusal of the field from a transient failure.
// Billed: 2 requests.
func TestLive_0032StrictHuggingFaceControl(t *testing.T) {
	require0032(t)
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderHuggingFace])
	// LLMPROVIDER_LIVE_0032_HF_MODEL names another model: the default one
	// answered every tool request with a 500 (0032-PLAN deviation D1).
	model := catalog.Static(llmprovider.ProviderHuggingFace)[0]
	if m := os.Getenv("LLMPROVIDER_LIVE_0032_HF_MODEL"); m != "" {
		model = m
	}
	for _, withStrict := range []bool{true, false} {
		name := map[bool]string{true: "with strict", false: "without strict"}[withStrict]
		rec := &flagRecorder{rewrite: func(body map[string]any) {
			if !withStrict {
				return
			}
			eachTool(body, func(tool map[string]any) {
				if fn, ok := tool["function"].(map[string]any); ok {
					fn["strict"] = false
				}
			})
		}}
		p, err := providers.New(llmprovider.ProviderHuggingFace, llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
			llmprovider.WithMaxTokens(64), llmprovider.WithModelProbes(false), llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := llmprovider.LiveCtx(t)
		_, genErr := p.Generate(ctx, &llmprovider.Request{Tools: []llmprovider.Tool{cityTool},
			Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Record the city Paris."}}})
		cancel()
		status, _ := rec.result()
		t.Logf("T1c huggingface %s (%s): status %d (error: %v)", name, model, status, genErr)
	}
}
