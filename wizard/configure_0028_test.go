package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0028-MADR D-H10: a key the service refuses while listing is not saved.
// A typed key is asked for again, up to three keys; a key from the
// environment or the saved configuration is an error naming where it came
// from.

// claudeRefusals answers Claude's model listing with a 401 for each of the
// first refusals requests, then with one model. It counts the requests.
func claudeRefusals(refusals int32, requests *atomic.Int32) *http.Client {
	model := catalog.Static(llmprovider.ProviderClaude)[0]
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		n := requests.Add(1)
		if n <= refusals {
			return &http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"` + model + `"}],"has_more":false}`))}, nil
	})}
}

// TestConfigureLLM_RefusedTypedKeyIsAskedAgain: two refused keys, then one
// the listing accepts, which the Result carries.
func TestConfigureLLM_RefusedTypedKeyIsAskedAgain(t *testing.T) {
	var requests atomic.Int32
	f := newFake(t, fakePrompter{
		blankSearches: true,
		selects:       []int{providerIdx(t, llmprovider.ProviderClaude), 0},
		secrets:       []string{"bad-1", "bad-2", "good-3"},
	})
	res, err := ConfigureLLM(context.Background(), f, Options{
		Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: claudeRefusals(2, &requests),
	})
	if err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if res.APIKey != "good-3" {
		t.Errorf("APIKey = %q; want the third key", res.APIKey)
	}
	if len(f.seenSecret) != 3 {
		t.Errorf("Secret prompts = %d; want 3", len(f.seenSecret))
	}
	if n := countContaining(f.seenNotify, "refused"); n != 2 {
		t.Errorf("notices = %v; want 2 saying the key was refused", f.seenNotify)
	}
	// A refusal is not an outage: no notice says the listing is unavailable.
	if n := countContaining(f.seenNotify, "unavailable"); n != 0 {
		t.Errorf("notices = %v; want none calling the listing unavailable", f.seenNotify)
	}
	if res.Model != catalog.Static(llmprovider.ProviderClaude)[0] {
		t.Errorf("Model = %q; want the listed model", res.Model)
	}
}

// TestConfigureLLM_RefusedTypedKeyThreeTimesFails: three refused keys end
// the run with an authentication error naming the provider.
func TestConfigureLLM_RefusedTypedKeyThreeTimesFails(t *testing.T) {
	var requests atomic.Int32
	f := newFake(t, fakePrompter{
		selects: []int{providerIdx(t, llmprovider.ProviderClaude)},
		secrets: []string{"bad-1", "bad-2", "bad-3"},
	})
	res, err := ConfigureLLM(context.Background(), f, Options{
		Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: claudeRefusals(3, &requests),
	})
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "Claude") {
		t.Fatalf("err = %v; want ErrAuthFailure naming Claude", err)
	}
	if !reflect.DeepEqual(res, Result{}) {
		t.Errorf("Result = %+v; want the zero Result", res)
	}
	if requests.Load() != 3 {
		t.Errorf("listings = %d; want 3", requests.Load())
	}
}

// TestConfigureLLM_RefusedEnvKeyFails: a refused key from the environment is
// not replaced by a prompt; the error names the variable.
func TestConfigureLLM_RefusedEnvKeyFails(t *testing.T) {
	var requests atomic.Int32
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderClaude)},
		confirms: []bool{true},
	})
	_, err := ConfigureLLM(context.Background(), f, Options{
		Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: claudeRefusals(1, &requests),
		AllowEnv: true, LookupEnv: func(name string) string {
			if name == "ANTHROPIC_API_KEY" {
				return "env-key-refused"
			}
			return ""
		},
	})
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("err = %v; want ErrAuthFailure naming ANTHROPIC_API_KEY", err)
	}
	if len(f.seenSecret) != 0 {
		t.Errorf("Secret prompts = %v; want none", f.seenSecret)
	}
}

// TestConfigureLLM_RefusedSavedKeyFails: a refused saved key, kept, is an
// error saying so; no prompt replaces it.
func TestConfigureLLM_RefusedSavedKeyFails(t *testing.T) {
	var requests atomic.Int32
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderClaude)},
		confirms: []bool{true},
	})
	_, err := ConfigureLLM(context.Background(), f, Options{
		Discover: true, DiscoverLimit: 2 * time.Second, HTTPClient: claudeRefusals(1, &requests),
		Existing: Result{Provider: llmprovider.ProviderClaude, Kind: CredAPIKey, APIKey: "saved-key-refused"},
	})
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "saved") {
		t.Fatalf("err = %v; want ErrAuthFailure saying the saved key was refused", err)
	}
	if len(f.seenSecret) != 0 {
		t.Errorf("Secret prompts = %v; want none", f.seenSecret)
	}
}

// TestRefusalError_NamesTheSource: each source of a refused credential is
// named, and the refusal stays an ErrAuthFailure.
func TestRefusalError_NamesTheSource(t *testing.T) {
	d := llmprovider.Descriptor{Label: "Claude", EnvVar: "ANTHROPIC_API_KEY"}
	refused := llmprovider.ErrAuthFailure
	for _, c := range []struct {
		cred resolvedCredential
		want string
	}{
		{resolvedCredential{keyFrom: keyTyped}, "refused 3 keys"},
		{resolvedCredential{keyFrom: keyEnvironment}, "the key in ANTHROPIC_API_KEY"},
		{resolvedCredential{keyFrom: keySaved}, "the saved key"},
		{resolvedCredential{kind: CredVendorCLI, vendorPath: "/home/<user>/.claude/auth.json"}, "CLI login in /home/<user>/.claude/auth.json"},
		{resolvedCredential{kind: CredOAuth}, "the sign-in; sign in again"},
		{resolvedCredential{kind: CredAPIKey}, "the pasted credential"},
	} {
		err := refusalError(d, c.cred, 3, refused)
		if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("refusalError(%+v) = %v; want ErrAuthFailure saying %q", c.cred, err, c.want)
		}
	}
}

// TestRefusedCredential_NotPermittedIsNotARefusal: a 403 the service sends
// for a permission is not a refused credential (0028-MADR D-H2), and an
// outage is neither.
func TestRefusedCredential_NotPermittedIsNotARefusal(t *testing.T) {
	if !refusedCredential(llmprovider.ErrAuthFailure) {
		t.Error("ErrAuthFailure is not a refusal; want one")
	}
	notPermitted := &llmprovider.APIError{Kind: llmprovider.ErrNotPermitted}
	if refusedCredential(notPermitted) || refusedCredential(errors.New("connection reset")) || refusedCredential(nil) {
		t.Error("a permission, an outage or nil counted as a refusal")
	}
}
