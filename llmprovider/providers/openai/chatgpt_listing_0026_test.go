package openai

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestChatGPTListing_BoundedAndKinded (0026-MADR F34): the ChatGPT listing
// is read under the catalog's listing cap (0021-MADR C3), and a reply it
// cannot use has a kind (R25): a body over the cap, one that does not decode,
// and a catalog that lists nothing are ErrIncomplete.
func TestChatGPTListing_BoundedAndKinded(t *testing.T) {
	huge := `{"models":[` + strings.Repeat(" ", 40<<20) + `]}`
	for _, c := range []struct {
		name string
		body func() io.Reader
		want string
	}{
		{"40 MiB", func() io.Reader { return strings.NewReader(huge) }, "larger than"},
		{"undecodable", func() io.Reader { return strings.NewReader(`{"models":[{"slug":`) }, "decode"},
		{"lists nothing", func() io.Reader { return strings.NewReader(`{"models":[]}`) }, "listed no models"},
	} {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(c.body()), Request: r}, nil
		})}
		p := sessionProvider(t, chatGPTListingSession(client), "gpt-6-astra", llmprovider.WithHTTPClient(client))
		_, err := list(t, p)
		if !errors.Is(err, llmprovider.ErrIncomplete) || err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: ListModels = %v; want ErrIncomplete naming %q", c.name, err, c.want)
		}
	}
}
