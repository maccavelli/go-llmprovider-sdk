package wizard

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0059-MADR S1 (W1, W3): a typed key is named before the listing only when
// it is plainly another provider's; a key's shape is named only once the
// provider has refused it. The keys are built to a shape; none is a key.

// claudeListing0059 answers Anthropic's model listing with status, counting
// the requests: one model on 200, an authentication error otherwise.
func claudeListing0059(status int, calls *atomic.Int32) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		body := `{"data":[{"id":"claude-sonnet-5","type":"model","display_name":"Claude Sonnet 5"}],"has_more":false}`
		if status != http.StatusOK {
			body = `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`
		}
		return &http.Response{StatusCode: status, Status: http.StatusText(status),
			Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
}

// options0059 lists Claude through client, in t's own cache and with no
// metadata document.
func options0059(t *testing.T, client *http.Client, discover bool) Options {
	return Options{Discover: discover, HTTPClient: client,
		ProviderOptions: []llmprovider.Option{llmprovider.WithoutModelMetadata(), testCache(t)}}
}

// TestShape_AcceptedKeyNoWarning_0059 (P1): a Claude key the listing accepts
// gets no notice, whether or not it has the confirmed shape, and is the key
// returned.
func TestShape_AcceptedKeyNoWarning_0059(t *testing.T) {
	for name, key := range map[string]string{
		"another version tag": "sk-ant-api04-" + shaped(shapeB64, 93) + "AA",
		"another length":      "sk-ant-api03-" + shaped(shapeB64, 94) + "AA",
		"the confirmed shape": "sk-ant-api03-" + shaped(shapeB64, 93) + "AA",
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			f := newFake(t, fakePrompter{secrets: []string{key}})
			res, _, err := ResolveCredential(context.Background(), f,
				options0059(t, claudeListing0059(http.StatusOK, &calls), true), llmprovider.ProviderClaude)
			if err != nil || res.APIKey != key || calls.Load() != 1 {
				t.Fatalf("ResolveCredential: err %v, key kept %t, %d listings; want the key, listed once",
					err, res.APIKey == key, calls.Load())
			}
			if len(f.seenNotify) != 0 {
				t.Errorf("notices %q; want none for a key the listing accepted", f.seenNotify)
			}
		})
	}
}

// TestShape_ForeignKey_0059: a key that is another provider's is named, with
// the listing's promise when the run lists; a provider's own key, of any
// shape, is not.
func TestShape_ForeignKey_0059(t *testing.T) {
	for _, c := range []struct {
		id     llmprovider.ProviderID
		key    string
		vendor string // "" for no notice
	}{
		{llmprovider.ProviderGrok, "sk-ant-api03-" + shaped(shapeB64, 93) + "AA", "an Anthropic"},
		{llmprovider.ProviderClaude, "xai-" + shaped(shapeAlnum, 80), "an xAI"},
		{llmprovider.ProviderOpenAI, "sk-" + shaped(shapeAlnum, 64), "an OpenCode"},
		{llmprovider.ProviderClaude, "sk-ant-api04-" + shaped(shapeB64, 10), ""},
		{llmprovider.ProviderOpencodeGo, "sk-" + shaped(shapeAlnum, 64), ""},
		{llmprovider.ProviderGemini, "AQ.Ab" + shaped(shapeB64, 50), ""},
	} {
		d := llmprovider.Descriptor{ID: c.id, Label: string(c.id), RequiresAPIKey: true}
		f := newFake(t, fakePrompter{secrets: []string{c.key}})
		got, _, err := resolveAPIKey(f, d, Options{Discover: true})
		if err != nil || got != c.key {
			t.Fatalf("%s: resolveAPIKey: err %v; want the typed key", c.id, err)
		}
		switch {
		case c.vendor == "" && len(f.seenNotify) != 0:
			t.Errorf("%s: its own key: notices %q; want none", c.id, f.seenNotify)
		case c.vendor != "" && (len(f.seenNotify) != 1 || !strings.Contains(f.seenNotify[0], c.vendor) ||
			!strings.Contains(f.seenNotify[0], "the model listing will check it")):
			t.Errorf("%s: %s key: notices %q; want one naming the vendor and the listing", c.id, c.vendor, f.seenNotify)
		}
		for _, n := range f.seenNotify {
			if strings.Contains(n, c.key) {
				t.Errorf("%s: a notice shows the key: %q", c.id, n)
			}
		}
	}
}

// TestShape_NoDiscoverNoPromise_0059 (P2): without Discover nothing is listed,
// so the notice about another vendor's key promises no check.
func TestShape_NoDiscoverNoPromise_0059(t *testing.T) {
	var calls atomic.Int32
	f := newFake(t, fakePrompter{secrets: []string{"xai-" + shaped(shapeAlnum, 80)}})
	if _, _, err := ResolveCredential(context.Background(), f,
		options0059(t, claudeListing0059(http.StatusOK, &calls), false), llmprovider.ProviderClaude); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 || len(f.seenNotify) != 1 || strings.Contains(f.seenNotify[0], "listing") ||
		!strings.Contains(f.seenNotify[0], "an xAI") {
		t.Errorf("%d listings, notices %q; want none, and one notice naming xAI with no listing promised",
			calls.Load(), f.seenNotify)
	}
}

// TestShape_RefusalNamesShape_0059: each refusal of a typed key without the
// provider's shape names the shape, the last included; a refused key with the
// shape is not told about its shape. No notice shows a key.
func TestShape_RefusalNamesShape_0059(t *testing.T) {
	for _, c := range []struct {
		name       string
		key        string
		wantShapes int
	}{
		{"without the shape", "not-a-key-" + shaped(shapeAlnum, 20), maxTypedKeys},
		{"with the shape", "sk-ant-api03-" + shaped(shapeB64, 93) + "AA", 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			keys := []string{c.key, c.key, c.key}
			f := newFake(t, fakePrompter{secrets: keys})
			_, _, err := ResolveCredential(context.Background(), f,
				options0059(t, claudeListing0059(http.StatusUnauthorized, &calls), true), llmprovider.ProviderClaude)
			if !errors.Is(err, llmprovider.ErrAuthFailure) {
				t.Fatalf("err %v; want one matching ErrAuthFailure", err)
			}
			if n := countContaining(f.seenNotify, "API keys look like sk-ant-api03- or sk-ant-admin01-"); n != c.wantShapes {
				t.Errorf("%d notices name the shape; want %d: %q", n, c.wantShapes, f.seenNotify)
			}
			for _, n := range f.seenNotify {
				if strings.Contains(n, c.key) {
					t.Errorf("a notice shows the key: %q", n)
				}
			}
		})
	}
}
