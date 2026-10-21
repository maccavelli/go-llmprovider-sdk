package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0056-MADR D3: a notice shows a service's error by its class and status,
// never its text.

// maskedEcho0056 is a refusal's message that repeats a key the service has
// masked itself, which redaction leaves as it is (0056-MADR F7).
const maskedEcho0056 = "Incorrect API key provided: sk-proj-********************************abcd."

// assertNoServiceText fails t if anything shown holds the message.
func assertNoServiceText(t *testing.T, f *fakePrompter) {
	t.Helper()
	for _, s := range f.allText {
		if strings.Contains(s, "sk-proj") || strings.Contains(s, "Incorrect API key") {
			t.Errorf("shown %q; want none of the service's message", s)
		}
	}
}

func TestDescribe_0056(t *testing.T) {
	apiErr := func(status int, kind error) error {
		return fmt.Errorf("list: %w", &llmprovider.APIError{Provider: "acme", Status: status, Kind: kind, Message: maskedEcho0056})
	}
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"401", apiErr(http.StatusUnauthorized, nil), "authentication failed, HTTP 401"},
		{"403 not permitted", apiErr(http.StatusForbidden, llmprovider.ErrNotPermitted), "not permitted, HTTP 403"},
		{"429 quota, the child first", apiErr(http.StatusTooManyRequests, llmprovider.ErrQuotaExhausted), "quota exhausted, HTTP 429"},
		{"500", apiErr(http.StatusInternalServerError, nil), "provider unavailable, HTTP 500"},
		{"a deadline", fmt.Errorf("list: %w", context.DeadlineExceeded), "timed out"},
		{"cancelled", fmt.Errorf("list: %w", context.Canceled), "cancelled"},
		{"a transport error", errors.New("dial tcp 127.0.0.1:1: connection refused"), "the request failed"},
		{"a sentinel with no status", fmt.Errorf("%w: %s", llmprovider.ErrAuthFailure, maskedEcho0056), "authentication failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := describe(tc.err); got != tc.want {
				t.Errorf("describe = %q; want %q", got, tc.want)
			}
		})
	}
}

// echoRefusals answers Claude's listing with a 401 whose message is the
// masked echo, refusals times, then with one model.
func echoRefusals(refusals int32, requests *atomic.Int32) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/v1/models") {
			return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}
		if requests.Add(1) <= refusals {
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(
				`{"type":"error","error":{"type":"authentication_error","message":"` + maskedEcho0056 + `"}}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"claude-x"}],"has_more":false}`))}, nil
	})}
}

// TestRefusalNotice_NoProviderText_0056: a refused key's notice is its class
// and status; after three, the error still matches ErrAuthFailure.
func TestRefusalNotice_NoProviderText_0056(t *testing.T) {
	var requests atomic.Int32
	f := newFake(t, fakePrompter{secrets: []string{"bad-1", "bad-2", "bad-3"}})
	_, _, err := ResolveCredential(context.Background(), f, Options{Discover: true, DiscoverLimit: 2 * time.Second,
		HTTPClient: echoRefusals(3, &requests)}, llmprovider.ProviderClaude)
	if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Errorf("error %v; want ErrAuthFailure", err)
	}
	if n := countContaining(f.seenNotify, "refused this key (authentication failed, HTTP 401); enter another"); n != 2 {
		t.Errorf("notices %q; want two worded by class and status", f.seenNotify)
	}
	assertNoServiceText(t, f)
}

// registry0056 holds one provider whose ListModels fails with err, and whose
// descriptor's static catalog is static.
func registry0056(t *testing.T, static []string, err error) *llmprovider.Registry {
	t.Helper()
	r := llmprovider.NewRegistry()
	d := llmprovider.Descriptor{ID: thirdPartyID, Label: "Acme", RequiresAPIKey: true, StaticModels: static}
	if regErr := r.Register(d, func(...llmprovider.Option) (llmprovider.Provider, error) {
		return listingThirdParty{&thirdParty{err: err}}, nil
	}); regErr != nil {
		t.Fatal(regErr)
	}
	return r
}

// TestListingWarnings_NoProviderText_0056: each listing warning, and Kilo's
// profile notice, shows the class and status, not the message.
func TestListingWarnings_NoProviderText_0056(t *testing.T) {
	unavailable := &llmprovider.APIError{Provider: "acme", Status: http.StatusInternalServerError, Message: maskedEcho0056}
	t.Run("no static catalog", func(t *testing.T) {
		f := newFake(t, fakePrompter{secrets: []string{"acme-key"}})
		_, _, err := ResolveCredential(context.Background(), f, Options{Registry: registry0056(t, nil, unavailable), Discover: true},
			thirdPartyID)
		if err != nil {
			t.Fatal(err)
		}
		if countContaining(f.seenNotify, "could not list models for Acme (provider unavailable, HTTP 500)") != 1 {
			t.Errorf("notices %q; want the warning by class and status", f.seenNotify)
		}
		assertNoServiceText(t, f)
	})
	t.Run("a static catalog", func(t *testing.T) {
		f := newFake(t, fakePrompter{secrets: []string{"acme-key"}})
		_, _, err := ResolveCredential(context.Background(), f, Options{Registry: registry0056(t, thirdPartyStatic, unavailable),
			Discover: true}, thirdPartyID)
		if err != nil {
			t.Fatal(err)
		}
		if countContaining(f.seenNotify, "could not list models for Acme (provider unavailable, HTTP 500); using the built-in catalog") != 1 {
			t.Errorf("notices %q; want the warning by class and status", f.seenNotify)
		}
		assertNoServiceText(t, f)
	})
	t.Run("the live listing unavailable", func(t *testing.T) {
		srv := zenServer(t, http.StatusInternalServerError, `{"error":{"message":"`+maskedEcho0056+`"}}`)
		f := newFake(t, fakePrompter{inputs: []string{srv.URL}, secrets: []string{testKey}})
		_, _, err := ResolveCredential(context.Background(), f, zenOptions(), llmprovider.ProviderOpencodeZen)
		if err != nil {
			t.Fatal(err)
		}
		if countContaining(f.seenNotify, "live model listing for OpenCode Zen is unavailable (provider unavailable, HTTP 500)") != 1 {
			t.Errorf("notices %q; want the warning by class and status", f.seenNotify)
		}
		assertNoServiceText(t, f)
	})
	t.Run("Kilo's profile", func(t *testing.T) {
		stubKilo(t, kiloAccount0056, unavailable)
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		_, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore()}, llmprovider.ProviderKilo)
		if err != nil {
			t.Fatal(err)
		}
		if countContaining(f.seenNotify, "could not read the Kilo account's organizations (provider unavailable, HTTP 500)") != 1 {
			t.Errorf("notices %q; want the notice by class and status", f.seenNotify)
		}
		assertNoServiceText(t, f)
	})
}

// TestLogoutNotice_NoProviderText_0056: a failed revocation's notice shows
// no message the service sent.
func TestLogoutNotice_NoProviderText_0056(t *testing.T) {
	srv := zenServer(t, http.StatusBadRequest, `{"error":"invalid_request","error_description":"`+maskedEcho0056+`"}`)
	store := newMemoryTokenStore()
	storedOpenAISession(t, store, srv)
	f := newFake(t, fakePrompter{confirms: []bool{true}})
	if err := Logout(context.Background(), f, Options{TokenStore: store}, llmprovider.ProviderOpenAI); err != nil {
		t.Fatal(err)
	}
	if countContaining(f.seenNotify, "could not revoke the OpenAI session (") != 1 {
		t.Errorf("notices %q; want the revocation's failure", f.seenNotify)
	}
	assertNoServiceText(t, f)
}
