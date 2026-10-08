package grok

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// renewingSource is an InvalidatingSource that counts its invalidations.
type renewingSource struct{ invalidated atomic.Int32 }

func (s *renewingSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{Value: "xai-test", Type: llmprovider.TokenBearer}, nil
}
func (s *renewingSource) Invalidate() { s.invalidated.Add(1) }

// TestReauth_GrokRefusedKeyRenews (0028-MADR H11): xAI refuses a key with
// HTTP 400 invalid-argument, "Incorrect API key provided" (0028-PLAN Phase 1,
// T1). That is a refused credential, so a renewable source is renewed once
// and the request sent again.
func TestReauth_GrokRefusedKeyRenews(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"code":"invalid-argument","error":"Incorrect API key provided. You can obtain an API key from https://console.x.ai."}`)
			return
		}
		_, _ = io.WriteString(w, `{"id":"r","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}]}`)
	}))
	defer srv.Close()
	src := &renewingSource{}
	p, err := New(llmprovider.WithTokenSource(src), llmprovider.WithModel("grok-test"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Generate(context.Background(), &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}})
	if err != nil || requests.Load() != 2 || src.invalidated.Load() != 1 {
		t.Fatalf("err %v after %d request(s) and %d invalidation(s); want success after 2 and 1", err, requests.Load(), src.invalidated.Load())
	}
}
