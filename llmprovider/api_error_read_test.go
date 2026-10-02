package llmprovider

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// failingBody reads part of an error body, then fails.
type failingBody struct{ done bool }

func (b *failingBody) Read(p []byte) (int, error) {
	if b.done {
		return 0, errors.New("connection reset")
	}
	b.done = true
	return copy(p, `{"error":{"message":"quota`), nil
}

func (*failingBody) Close() error { return nil }

// TestClassifyHTTPError_SaysTheBodyWasUnreadable (0015-PLAN S10): a body that
// fails to read is said in the error's message, where it used to go to the
// global logger.
func TestClassifyHTTPError_SaysTheBodyWasUnreadable(t *testing.T) {
	err := ClassifyHTTPError("kilo", &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: &failingBody{}})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || !strings.Contains(apiErr.Message, "error body unreadable: connection reset") {
		t.Fatalf("err = %v, want the unreadable body in the message", err)
	}
}
