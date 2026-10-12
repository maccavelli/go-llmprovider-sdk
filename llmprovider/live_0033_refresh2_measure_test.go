//go:build live_gateways

package llmprovider_test

import (
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0033-PLAN-refresh-the-curated-catalogs.md Phase 7's measurement: each id the
// second refresh adds answers a request before it is listed (0033-MADR D4).
// It runs only with LLMPROVIDER_LIVE_0033_REFRESH set. Each key is read from
// its provider's variable and sent only in its request header; only the
// status and the error are logged. Model metadata is left on, so each model
// is routed as the SDK routes it at runtime: OpenCode Go sends a Claude model
// to Messages only through models.dev's npm (Phase 7's deviation).

// refreshCandidates0033 are the additions the command printed on 2026-10-12,
// by provider, after 0009-MADR §3 items 10, 11 and 12 (Phases 7a to 7c). The
// Together picks that could not be served are in Phase 7's record.
var refreshCandidates0033 = map[llmprovider.ProviderID][]string{
	llmprovider.ProviderOpencodeZen: {"claude-haiku-5-5"},
	llmprovider.ProviderOpencodeGo:  {"claude-haiku-5-5", "space-bunny"},
	llmprovider.ProviderKilo:        {"openai/gpt-6.1-sol"},
	llmprovider.ProviderTogether:    {"zai-org/GLM-5.3-Flash", "deepseek-ai/DeepSeek-V4-Flash-0731"},
}

// statusRecorder keeps the status of the last reply to a POST.
type statusRecorder struct {
	mu     sync.Mutex
	status int
}

func (r *statusRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil && req.Method == http.MethodPost {
		r.mu.Lock()
		r.status = resp.StatusCode
		r.mu.Unlock()
	}
	return resp, err
}

func (r *statusRecorder) last() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// TestLive_0033RefreshCandidates is Phase 7, step 2. Billed: one generation of
// at most 16 output tokens per candidate, 6 in all.
func TestLive_0033RefreshCandidates(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0033_REFRESH") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0033_REFRESH to run 0033-PLAN-refresh-the-curated-catalogs Phase 7's measurement")
	}
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo,
		llmprovider.ProviderKilo, llmprovider.ProviderTogether} {
		for _, model := range refreshCandidates0033[id] {
			t.Run(string(id)+"/"+model, func(t *testing.T) {
				key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id])
				rec := &statusRecorder{}
				p, err := providers.New(id, llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
					llmprovider.WithMaxTokens(16), llmprovider.WithModelProbes(false),
					llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := llmprovider.LiveCtx(t)
				defer cancel()
				_, genErr := p.Generate(ctx, &llmprovider.Request{
					Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hello."}}})
				status := rec.last()
				t.Logf("%s %s: status %d, error: %v", id, model, status, genErr)
				if status != http.StatusOK {
					t.Errorf("%s %s did not answer: status %d, error: %v", id, model, status, genErr)
				}
			})
		}
	}
}
