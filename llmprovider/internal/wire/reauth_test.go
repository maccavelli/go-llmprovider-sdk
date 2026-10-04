package wire

import (
	"context"
	"errors"
	"net/http"
	"testing"

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
			_, err := Reauth(c.src, func() (int, error) {
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
