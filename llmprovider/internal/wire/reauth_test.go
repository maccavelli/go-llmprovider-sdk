package wire

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

type countingSource struct{ invalidated int }

func (s *countingSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{Value: "t"}, nil
}

func (s *countingSource) Invalidate() { s.invalidated++ }

// TestReauth (0020-MADR F2): only a 401 with an InvalidatingSource sends
// again, once.
func TestReauth(t *testing.T) {
	unauthorized := &llmprovider.APIError{Status: http.StatusUnauthorized}
	for _, c := range []struct {
		name        string
		src         llmprovider.TokenSource
		errs        []error
		wantSends   int
		wantErr     bool
		invalidates int
	}{
		{"success", &countingSource{}, []error{nil}, 1, false, 0},
		{"other error", &countingSource{}, []error{errors.New("boom")}, 1, true, 0},
		{"429", &countingSource{}, []error{&llmprovider.APIError{Status: http.StatusTooManyRequests}}, 1, true, 0},
		{"401 static", llmprovider.NewStaticToken("k"), []error{unauthorized}, 1, true, 0},
		{"401 renewed", &countingSource{}, []error{unauthorized, nil}, 2, false, 1},
		{"401 twice", &countingSource{}, []error{unauthorized, unauthorized}, 2, true, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			sends := 0
			_, err := Reauth(context.Background(), "p", c.src, func(llmprovider.Token) (int, error) {
				err := c.errs[sends]
				sends++
				return sends, err
			})
			if sends != c.wantSends || (err != nil) != c.wantErr {
				t.Errorf("sends = %d, err = %v; want %d sends, error %v", sends, err, c.wantSends, c.wantErr)
			}
			if s, ok := c.src.(*countingSource); ok && s.invalidated != c.invalidates {
				t.Errorf("invalidations = %d, want %d", s.invalidated, c.invalidates)
			}
		})
	}
}

// generationSource hands out "t<generation>", and refreshes to the next
// generation when its token was invalidated. Invalidate always does;
// InvalidateToken only when the refused token is still the current one.
type generationSource struct {
	mu        sync.Mutex
	gen       int
	stale     bool
	refreshes int
}

func (s *generationSource) Token(context.Context) (llmprovider.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gen == 0 || s.stale {
		if s.gen > 0 {
			s.refreshes++
		}
		s.gen, s.stale = s.gen+1, false
	}
	return llmprovider.Token{Value: fmt.Sprintf("t%d", s.gen)}, nil
}

func (s *generationSource) Invalidate() {
	s.mu.Lock()
	s.stale = true
	s.mu.Unlock()
}

func (s *generationSource) InvalidateToken(token llmprovider.Token) {
	s.mu.Lock()
	if token.Value == fmt.Sprintf("t%d", s.gen) {
		s.stale = true
	}
	s.mu.Unlock()
}

// TestReauth_LateRefusalKeepsFreshToken (0021-MADR D5, T1): eight sends hold
// one token and each gets a 401. The 401s land one at a time, each after the
// previous send was resent, so every late 401 names a token the source has
// already replaced. The source is refreshed exactly once.
func TestReauth_LateRefusalKeepsFreshToken(t *testing.T) {
	const sends = 8
	src := &generationSource{}
	arrived, retried := make(chan struct{}, sends), make(chan struct{}, sends)
	release, done := make(chan struct{}), make(chan struct{})
	defer close(done)
	unauthorized := &llmprovider.APIError{Status: http.StatusUnauthorized}
	errs := make([]error, sends)
	var wg sync.WaitGroup
	for i := range sends {
		wg.Go(func() {
			_, errs[i] = Reauth(context.Background(), "p", src, func(token llmprovider.Token) (int, error) {
				if token.Value != "t1" {
					retried <- struct{}{}
					return 1, nil
				}
				arrived <- struct{}{}
				select {
				case <-release:
				case <-done:
				}
				return 0, unauthorized
			})
		})
	}
	wait := func(ch <-chan struct{}, what string) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %s", what)
		}
	}
	for range sends {
		wait(arrived, "the first sends")
	}
	for range sends {
		release <- struct{}{}
		wait(retried, "a resend")
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("send %d: %v", i, err)
		}
	}
	if src.refreshes != 1 {
		t.Errorf("refreshes = %d, want 1: a late 401 discarded a fresh token", src.refreshes)
	}
}
