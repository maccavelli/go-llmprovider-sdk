//go:build live_gateways

package llmprovider_test

import (
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// codexHeaderRecorder passes requests on and keeps, from each reply, the
// lower-cased names of its x-codex- headers, and the values of only those
// that end -window-minutes (0072-PLAN Phase 5 step 1).
type codexHeaderRecorder struct {
	mu      sync.Mutex
	status  []int
	names   [][]string
	windows []map[string]string
}

func (r *codexHeaderRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	var names []string
	windows := map[string]string{}
	for name, values := range resp.Header {
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "x-codex-") {
			continue
		}
		names = append(names, lower)
		if strings.HasSuffix(lower, "-window-minutes") {
			windows[lower] = strings.Join(values, ", ")
		}
	}
	slices.Sort(names)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = append(r.status, resp.StatusCode)
	r.names = append(r.names, names)
	r.windows = append(r.windows, windows)
	return resp, nil
}

// TestLive_0072ChatGPTHeaders is 0072-PLAN Phase 5 step 1's measurement. It
// runs only with LLMPROVIDER_LIVE_0072 and LLMPROVIDER_LIVE_CHATGPT set, and
// sends the smallest generation TestLive_ChatGPTGenerate sends, through a
// client whose transport records the reply's headers. It logs the names of
// the x-codex- headers, and the values of the x-codex-*-window-minutes ones;
// no other value. It answers 0072-MADR D5's open question, whether ChatGPT
// sends this SDK's requests the window headers the Codex CLI reads.
func TestLive_0072ChatGPTHeaders(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0072") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0072 to run 0072-PLAN Phase 5's measurement")
	}
	session := liveChatGPTSession(t)
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	rec := &codexHeaderRecorder{}
	p := liveOpenAI(t, session, "gpt-6-astra", llmprovider.WithHTTPClient(&http.Client{Transport: rec}))
	_, err := llmprovider.GenerateText(ctx, p, userText("Reply with only the word ALPHA"))
	llmprovider.SkipIfTransient(t, err)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.status) == 0 {
		t.Fatal("Generate sent no request through the recording client")
	}
	for i, status := range rec.status {
		t.Logf("reply %d: HTTP %d; %d x-codex- headers: %s", i+1, status, len(rec.names[i]), strings.Join(rec.names[i], ", "))
		keys := make([]string, 0, len(rec.windows[i]))
		for k := range rec.windows[i] {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			t.Logf("reply %d: %s = %s", i+1, k, rec.windows[i][k])
		}
	}
}
