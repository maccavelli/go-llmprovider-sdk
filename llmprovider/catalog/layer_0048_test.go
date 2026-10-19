package catalog_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// meetingLayer0048 is a Layer whose Model waits, for a while, until two calls
// are in it at once; calls counts every call.
type meetingLayer0048 struct {
	in, calls atomic.Int32
	met       chan struct{}
	once      sync.Once
}

func (*meetingLayer0048) Name() string { return "test-0048" }

func (l *meetingLayer0048) Model(llmprovider.ProviderID, string) (catalog.ModelInfo, bool) {
	l.calls.Add(1)
	defer l.in.Add(-1)
	if l.in.Add(1) == 2 {
		l.once.Do(func() { close(l.met) })
	}
	select {
	case <-l.met:
	case <-time.After(2 * time.Second):
	}
	return catalog.ModelInfo{}, false
}

// TestLayer_CalledByConcurrentRequests_0048 (0048-MADR D3): a provider given
// WithModelLayer asks the layer on every request, so concurrent requests call
// its Model at once.
func TestLayer_CalledByConcurrentRequests_0048(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_0048","type":"message","role":"assistant","model":"claude-x",` +
			`"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(srv.Close)
	layer := &meetingLayer0048{met: make(chan struct{})}
	p, err := providers.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("key-0048"), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithModel("claude-x"), llmprovider.WithoutModelMetadata(), catalog.WithModelLayer(layer))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, err := llmprovider.GenerateText(context.Background(), p, &llmprovider.Request{
				Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
			}); err != nil {
				t.Errorf("GenerateText: %v", err)
			}
		})
	}
	wg.Wait()
	select {
	case <-layer.met:
	default:
		t.Errorf("two concurrent requests never called the layer at once (%d calls)", layer.calls.Load())
	}
}
