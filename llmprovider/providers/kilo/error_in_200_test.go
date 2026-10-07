package kilo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestKilo_ErrorIn200IsNotRetried (0026-MADR F5): a context overflow that
// Kilo reports inside a 200, with the status in error.code, is
// ErrContextOverflow and is sent once through WithRetry, not resent as an
// outage; the error names kilo.
func TestKilo_ErrorIn200IsNotRetried(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error":{"code":400,"message":"This endpoint's maximum context length is 8192 tokens. However, you requested about 9000 tokens."}}`)
	}))
	defer srv.Close()
	p := build(t, llmprovider.WithAPIKey("kilo_test"), llmprovider.WithModel("kilo-auto/free"), llmprovider.WithBaseURL(srv.URL))
	retrying := llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond})
	_, err := llmprovider.GenerateText(context.Background(), retrying, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
	})
	var apiErr *llmprovider.APIError
	if !errors.Is(err, llmprovider.ErrContextOverflow) || !errors.As(err, &apiErr) || apiErr.Provider != "kilo" {
		t.Fatalf("GenerateText = %v; want ErrContextOverflow from kilo", err)
	}
	if n := hits.Load(); n != 1 {
		t.Errorf("%d requests, want 1: an overflow cannot succeed on a retry", n)
	}
}
