package wire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestPost_StalledErrorBodyEndsAtIdleLimit (0026-MADR F4): an error reply
// whose body stalls after its headers ends at the idle limit, as a stalled
// 200 does, and keeps its status's kind. With no caller deadline it would
// otherwise wait for ever.
func TestPost_StalledErrorBodyEndsAtIdleLimit(t *testing.T) {
	defer SetIdleTimeout(100 * time.Millisecond)()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"error":{"message":"overloa`)
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := Post(ctx, postCall(srv.URL), llmprovider.Token{}, readAllString)
	elapsed := time.Since(start)
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("a stalled 503 body: %v; want ErrProviderUnavailable", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("a stalled 503 body took %s; want about the idle limit, 100 ms", elapsed)
	}
}

// TestPost_UnsendableIsInvalidRequest (0026-MADR F11): a token that no header
// can carry fails as ErrInvalidRequest, without reaching the service, so
// WithRetry does not try it again.
func TestPost_UnsendableIsInvalidRequest(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits++ }))
	defer srv.Close()
	_, err := Post(context.Background(), postCall(srv.URL), llmprovider.Token{Value: "sk-test\nX-Injected: 1"}, readAllString)
	if !errors.Is(err, llmprovider.ErrInvalidRequest) || hits != 0 {
		t.Fatalf("a token with a newline: %v after %d requests; want ErrInvalidRequest and none", err, hits)
	}
	if _, err := Post(context.Background(), postCall("http://host/%zz"), llmprovider.Token{}, readAllString); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("a URL that does not parse: %v; want ErrInvalidRequest", err)
	}
}

// TestPost_UnmarshalableBodyIsInvalidRequest (0026-MADR F20): a body that
// cannot be marshalled is the caller's request, so it is ErrInvalidRequest,
// which WithRetry does not retry.
func TestPost_UnmarshalableBodyIsInvalidRequest(t *testing.T) {
	c := postCall("http://127.0.0.1:1")
	c.Body = map[string]any{"tools": make(chan int)}
	if _, err := Post(context.Background(), c, llmprovider.Token{}, readAllString); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("a body holding a channel: %v; want ErrInvalidRequest", err)
	}
}

// TestPost_Accepts2xx (0026-MADR F24): Post's doc says it decodes a 2xx reply,
// so a 201 is decoded as a success.
func TestPost_Accepts2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()
	got, err := Post(context.Background(), postCall(srv.URL), llmprovider.Token{}, readAllString)
	if err != nil || got != `{"ok":true}` {
		t.Fatalf("a 201: %q, %v; want its body decoded", got, err)
	}
}
