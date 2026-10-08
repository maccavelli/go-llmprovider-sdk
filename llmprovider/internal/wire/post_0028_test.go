package wire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestPost_FailureAfterWriteIsAfterReply (0028-MADR D-H1): the service read
// the whole request and has not answered when the client stops waiting, as a
// long non-streaming generation does. The request was sent, and may be
// billed, so the failure is marked as after the reply began: WithRetry
// resends it at most once.
func TestPost_FailureAfterWriteIsAfterReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()
	call := postCall(srv.URL)
	call.Client = &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 50 * time.Millisecond}}
	_, err := Post(context.Background(), call, llmprovider.Token{Value: "t"}, readAllString)
	if err == nil || !transport.IsAfterReply(err) {
		t.Fatalf("Post = %v; want a failure marked after the reply, since the request was written", err)
	}
}

// TestPost_FailureBeforeWriteIsNot: a request that never reached the service,
// here a refused connection, keeps its own *url.Error, unmarked, so
// WithRetry retries it as before (0020-MADR F9).
func TestPost_FailureBeforeWriteIsNot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.URL
	srv.Close()
	_, err := Post(context.Background(), postCall(addr), llmprovider.Token{Value: "t"}, readAllString)
	var urlErr *url.Error
	if err == nil || transport.IsAfterReply(err) || !errors.As(err, &urlErr) {
		t.Fatalf("Post = %v; want an unmarked *url.Error", err)
	}
}
