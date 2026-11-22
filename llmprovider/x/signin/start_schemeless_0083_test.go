package signin

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// codeRecorder0083 is a transport that records the code of each token request
// and passes the request on.
type codeRecorder0083 struct {
	next  http.RoundTripper
	mu    sync.Mutex
	codes []string
}

func (c *codeRecorder0083) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(strings.NewReader(string(raw)))
		if form, err := url.ParseQuery(string(raw)); err == nil && form.Get("code") != "" {
			c.mu.Lock()
			c.codes = append(c.codes, form.Get("code"))
			c.mu.Unlock()
		}
	}
	return c.next.RoundTrip(req)
}

// TestStart_SchemelessPasteCompletes_0083 (0083-MADR D11, ST76): with NoLoopback,
// the real auth code against the fake issuer, a paste of the redirect address
// with no http:// in front completes the sign-in, as the address bar shows it
// until it is tapped.
func TestStart_SchemelessPasteCompletes_0083(t *testing.T) {
	for _, p := range browserProviders {
		t.Run(string(p), func(t *testing.T) {
			fake := &fakeIssuer{}
			recorder := &codeRecorder0083{next: fake}
			run, err := Start(background(), p, llmprovider.AuthBrowserOAuth,
				Options{HTTPClient: &http.Client{Transport: recorder}, Issuer: fakeBase, ClientID: "test-client", NoLoopback: true})
			if err != nil {
				t.Fatalf("Start = %v", err)
			}
			open := nextStep(t, run)
			ask := nextStep(t, run)
			wantStep(t, open, authcontract.StepOpenURL, 1)
			wantStep(t, ask, authcontract.StepPaste, 2)
			authorize, err := url.Parse(open.URL)
			if err != nil {
				t.Fatalf("open_url's URL = %q: %v", open.URL, err)
			}
			query := authorize.Query()
			redirect := strings.TrimPrefix(query.Get("redirect_uri"), "http://")
			if redirect == query.Get("redirect_uri") || redirect == "" {
				t.Fatalf("redirect_uri = %q; want an http:// address", query.Get("redirect_uri"))
			}
			fake.setNonce(open.URL)
			pasted := redirect + "?code=code-0083&state=" + url.QueryEscape(query.Get("state"))
			if err := run.Answer(background(), authcontract.Answer{Seq: ask.Seq, Paste: pasted}); err != nil {
				t.Fatalf("Answer = %v", err)
			}
			credential, err := run.Wait(bounded(t))
			if err != nil {
				t.Fatalf("Wait = %v with the paste %q; want the sign-in completed", err, redirect)
			}
			if _, ok := credential.(Session); !ok {
				t.Errorf("credential is a %T; want a Session", credential)
			}
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			if len(recorder.codes) != 1 || recorder.codes[0] != "code-0083" {
				t.Errorf("the token request carried the codes %q; want the code of the pasted address, once", recorder.codes)
			}
		})
	}
}
