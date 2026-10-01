package llmprovider

import (
	"context"
	"testing"
)

// TestKeyless_KiloSendsAnonymousToken: without a key, Kilo sends its
// "anonymous" token (MADR 0012 §1.7). OpenCode's half moved to
// providers/opencode (0015-PLAN S7).
func TestKeyless_KiloSendsAnonymousToken(t *testing.T) {
	rec := newHeaderRecorder(t)
	base := WithBaseURL(rec.srv.URL)
	kilo, err := NewKilo("", "some/model", base)
	if err != nil {
		t.Fatalf("NewKilo without a key: %v", err)
	}
	_, _ = kilo.Generate(context.Background(), "hello")
	reqs := rec.requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	if got := reqs[0].header.Get("Authorization"); got != "Bearer anonymous" {
		t.Errorf("%s: Authorization = %q, want Bearer anonymous", reqs[0].path, got)
	}
}

// TestKeyless_KeyStillWins: a supplied key is sent as is.
func TestKeyless_KeyStillWins(t *testing.T) {
	rec := newHeaderRecorder(t)
	p, err := NewKilo("real-key", "some/model", WithBaseURL(rec.srv.URL))
	if err != nil {
		t.Fatalf("NewKilo: %v", err)
	}
	_, _ = p.Generate(context.Background(), "hello")
	if got := rec.requests()[0].header.Get("Authorization"); got != "Bearer real-key" {
		t.Fatalf("Authorization = %q, want Bearer real-key", got)
	}
}
