//go:build live_gateways

package llmprovider_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// 0028-PLAN Phase 1's measurements. They run only with LLMPROVIDER_LIVE_0028
// set, and log what D-H2's credential codes and D-H5's reset headers are
// built from. They fail only on a transport error.

// invalid0028Key is a literal that is no provider's key.
const invalid0028Key = "invalid-0028-not-a-key"

// measured0028 are the providers Phase 1 measures.
var measured0028 = []llmprovider.ProviderID{
	llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGemini, llmprovider.ProviderGrok,
	llmprovider.ProviderTogether, llmprovider.ProviderHuggingFace, llmprovider.ProviderKilo, llmprovider.ProviderOpencodeGo,
}

// rateHeaderPrefixes are the lower-cased header-name prefixes T2 records.
var rateHeaderPrefixes = []string{"x-ratelimit-", "anthropic-ratelimit-", "retry-after", "ratelimit"}

// measureRecorder passes requests on and keeps each reply's status, its
// rate-limit headers, and the first 2 KiB of its body.
type measureRecorder struct {
	mu      sync.Mutex
	status  []int
	headers []map[string]string
	bodies  [][]byte
}

func (r *measureRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(b))
	kept := map[string]string{}
	for name, values := range resp.Header {
		lower := strings.ToLower(name)
		if slices.ContainsFunc(rateHeaderPrefixes, func(p string) bool { return strings.HasPrefix(lower, p) }) {
			kept[lower] = strings.Join(values, ", ")
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = append(r.status, resp.StatusCode)
	r.headers = append(r.headers, kept)
	r.bodies = append(r.bodies, b[:min(len(b), 2048)])
	return resp, nil
}

func (r *measureRecorder) last() (int, map[string]string, []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.status) == 0 {
		return 0, nil, nil
	}
	n := len(r.status) - 1
	return r.status[n], r.headers[n], r.bodies[n]
}

func require0028(t *testing.T) {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_0028") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0028 to run 0028-PLAN Phase 1's measurements")
	}
}

// TestLive_0028InvalidKeyReplies is T1: each provider's reply to a key that
// is not a key. Unbilled.
func TestLive_0028InvalidKeyReplies(t *testing.T) {
	require0028(t)
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}
	for _, id := range measured0028 {
		t.Run(string(id), func(t *testing.T) {
			rec := &measureRecorder{}
			p, err := providers.New(id, llmprovider.WithAPIKey(invalid0028Key), llmprovider.WithModel(catalog.Static(id)[0]),
				llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			_, err = p.Generate(ctx, req)
			status, _, body := rec.last()
			if status == 0 {
				t.Fatalf("no reply: %v", err)
			}
			apiErr, _ := errors.AsType[*llmprovider.APIError](err)
			code, kind := "", error(nil)
			if apiErr != nil {
				code, kind = apiErr.Code, apiErr.Kind
			}
			t.Logf("T1 %s: status %d, code %q, kind %v; body: %s", id, status, code, kind, redact.String(string(body)))
		})
	}
}

// TestLive_0028RateLimitHeaders is T2: the rate-limit headers each provider
// sends on its model listing (unbilled), and, where the listing carries none,
// on one 16-token generation (billed, approved 2026-10-08).
func TestLive_0028RateLimitHeaders(t *testing.T) {
	require0028(t)
	for _, id := range measured0028 {
		if id == llmprovider.ProviderOpencodeGo {
			continue // one key serves both gateways; measured as T1 only
		}
		t.Run(string(id), func(t *testing.T) {
			key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id])
			rec := &measureRecorder{}
			client := &http.Client{Transport: rec}
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			_, _ = catalog.List(ctx, id, llmprovider.NewStaticToken(key), llmprovider.WithHTTPClient(client),
				llmprovider.WithModelProbes(false))
			status, headers, _ := rec.last()
			t.Logf("T2 %s listing: status %d, headers %v", id, status, headers)
			if len(headers) > 0 {
				return
			}
			p, err := providers.New(id, llmprovider.WithAPIKey(key), llmprovider.WithModel(catalog.Static(id)[0]),
				llmprovider.WithMaxTokens(16), llmprovider.WithHTTPClient(client))
			if err != nil {
				t.Fatal(err)
			}
			_, genErr := p.Generate(ctx, &llmprovider.Request{Input: []llmprovider.Item{
				llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say ok."}}})
			status, headers, _ = rec.last()
			t.Logf("T2 %s generation: status %d, headers %v (error: %v)", id, status, headers, genErr)
		})
	}
}
