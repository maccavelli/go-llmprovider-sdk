package wire

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

func postCall(url string) Call {
	return Call{
		Provider: "p",
		Client:   &http.Client{},
		Logger:   slog.New(slog.DiscardHandler),
		URL:      url,
		Body:     map[string]any{"q": 1},
		Prepare: func(r *http.Request, token llmprovider.Token) {
			r.Header.Set("Authorization", "Bearer "+token.Value)
		},
	}
}

func readAllString(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

// TestPost_SendsAndDecodes (0021-MADR W11): the body, its content type and the
// prepared headers go out, and a 2xx reply is decoded.
func TestPost_SendsAndDecodes(t *testing.T) {
	var got map[string]any
	var auth, contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		auth, contentType = r.Header.Get("Authorization"), r.Header.Get("Content-Type")
		_, _ = io.WriteString(w, "answer")
	}))
	defer srv.Close()
	out, err := Post(context.Background(), postCall(srv.URL), llmprovider.Token{Value: "t"}, readAllString)
	if err != nil || out != "answer" {
		t.Fatalf("Post = %q, %v; want the answer", out, err)
	}
	if !reflect.DeepEqual(got, map[string]any{"q": float64(1)}) || auth != "Bearer t" || contentType != "application/json" {
		t.Errorf("sent %v, %q, %q", got, auth, contentType)
	}
}

// TestPost_Failures: a non-2xx reply is classified, a body that cannot be
// marshalled or sent is an error, and a decode error gets its kind.
func TestPost_Failures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/limited" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, "not json")
	}))
	defer srv.Close()
	token := llmprovider.Token{Value: "t"}
	if _, err := Post(context.Background(), postCall(srv.URL+"/limited"), token, readAllString); !errors.Is(err, llmprovider.ErrRateLimited) {
		t.Errorf("a 429: %v; want ErrRateLimited", err)
	}
	bad := postCall(srv.URL)
	bad.Body = map[string]any{"c": make(chan int)}
	if _, err := Post(context.Background(), bad, token, readAllString); err == nil || !strings.Contains(err.Error(), "marshal request") {
		t.Errorf("an unmarshallable body: %v", err)
	}
	if _, err := Post(context.Background(), postCall("http://127.0.0.1:1"), token, readAllString); err == nil {
		t.Error("a refused connection gave no error")
	}
	if _, err := Post(context.Background(), postCall("://bad"), token, readAllString); err == nil {
		t.Error("a bad URL gave no error")
	}
	decode := func(r io.Reader) (any, error) {
		var v any
		return v, json.NewDecoder(r).Decode(&v)
	}
	if _, err := Post(context.Background(), postCall(srv.URL), token, decode); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("an undecodable 200: %v; want ErrIncomplete", err)
	}
}

// TestPost_IdleLimit (0021-MADR D2): a body that sends nothing for the idle
// limit ends the request, as a failure after the reply.
func TestPost_IdleLimit(t *testing.T) {
	defer SetIdleTimeout(50 * time.Millisecond)()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "{")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	_, err := Post(ctx, postCall(srv.URL), llmprovider.Token{}, readAllString)
	if !errors.Is(err, transport.ErrIdleTimeout) || !transport.IsAfterReply(err) || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Fatalf("an idle body: %v; want ErrProviderUnavailable, ErrIdleTimeout, after the reply", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("an idle body took %s, want about 50 ms", elapsed)
	}
}

// TestPost_StreamHasNoWholeBodyLimit (0021-MADR W2): an event stream is not
// cut at ReplyLimit; a JSON reply is.
func TestPost_StreamHasNoWholeBodyLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", ReplyLimit+1))
	}))
	defer srv.Close()
	call := postCall(srv.URL)
	call.Stream = true
	if out, err := Post(context.Background(), call, llmprovider.Token{}, readAllString); err != nil || len(out) != ReplyLimit+1 {
		t.Errorf("a stream: %d bytes, %v; want it whole", len(out), err)
	}
	call.Stream = false
	if _, err := Post(context.Background(), call, llmprovider.Token{}, readAllString); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("a JSON reply over the limit: %v; want ErrIncomplete", err)
	}
}
