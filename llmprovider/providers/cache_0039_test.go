package providers

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/wiretest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// 0039-MADR D2-D4: what each built-in provider sends for Request.Cache. The
// routing key is the provider's session id, sent only when Cache is set; the
// top-level cache_control goes to Anthropic's Messages and to Kilo's
// anthropic/ models; nothing else is sent.

// cacheSession pins the session id the routing key carries.
const cacheSession = "session-0039"

// TestCache_PerProvider sends one request with CacheShort and one without,
// to each provider and model, and checks the cache fields of each body.
func TestCache_PerProvider(t *testing.T) {
	for _, tc := range []struct {
		id        llmprovider.ProviderID
		model     string
		key       bool // prompt_cache_key, with Cache
		topMarker bool // the top-level cache_control, with Cache
	}{
		{llmprovider.ProviderOpenAI, "gpt-6-luna", true, false},
		{llmprovider.ProviderGrok, "grok-4.6", true, false},
		{llmprovider.ProviderTogether, "openai/gpt-oss-120b", true, false},
		{llmprovider.ProviderOpencodeZen, "gpt-5.5", true, false},
		{llmprovider.ProviderOpencodeZen, "claude-sonnet-5", false, true},
		{llmprovider.ProviderOpencodeZen, "glm-5.3", false, false},
		{llmprovider.ProviderKilo, "anthropic/claude-haiku-5.5", false, true},
		{llmprovider.ProviderKilo, "openai/gpt-6-luna", false, false},
		{llmprovider.ProviderClaude, "claude-haiku-5-5", false, true},
		{llmprovider.ProviderGemini, "gemini-3.8-flash", false, false},
		{llmprovider.ProviderHuggingFace, "deepseek-ai/DeepSeek-V4.1-Flash", false, false},
		{llmprovider.ProviderOllama, "llama3.3:latest", false, false},
	} {
		t.Run(string(tc.id)+"/"+tc.model, func(t *testing.T) {
			for _, cached := range []bool{true, false} {
				body := cacheBody(t, tc.id, tc.model, cached)
				_, hasKey := body["prompt_cache_key"]
				_, hasMarker := body["cache_control"]
				if wantKey := cached && tc.key; hasKey != wantKey || (hasKey && body["prompt_cache_key"] != cacheSession) {
					t.Errorf("Cache %t: prompt_cache_key = %v (sent %t); want sent %t, as %q", cached, body["prompt_cache_key"],
						hasKey, wantKey, cacheSession)
				}
				if wantMarker := cached && tc.topMarker; hasMarker != wantMarker {
					t.Errorf("Cache %t: top-level cache_control sent %t; want %t", cached, hasMarker, wantMarker)
				}
			}
		})
	}
}

// cacheBody is the body of the generation request id sends for model, with
// CacheShort when cached.
func cacheBody(t *testing.T, id llmprovider.ProviderID, model string, cached bool) map[string]any {
	t.Helper()
	srv := wiretest.NewServer(t, wirecase.Case{}.Reply)
	p, err := New(id, llmprovider.WithAPIKey("cache-key"), llmprovider.WithModel(model), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithSessionID(cacheSession), llmprovider.WithoutModelMetadata(), llmprovider.WithModelProbes(false))
	if err != nil {
		t.Fatal(err)
	}
	req := &llmprovider.Request{Instructions: "Be brief.", Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
	if cached {
		req.Cache = llmprovider.CacheShort
	}
	_, _ = p.Generate(context.Background(), req)
	var last map[string]any
	for _, r := range srv.Requests() {
		if b, ok := r.Body.(map[string]any); ok && r.Method == "POST" {
			last = b
		}
	}
	if last == nil {
		t.Fatalf("%s sent no generation request", id)
	}
	return last
}
