package wizard

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestKiloVerify_EndedContext_0080 (D6): a ctx that ends while the profile is
// read is neither a refusal nor a warning, and nothing is saved after it: the
// step returns ctx's error, as ConfigureLLM and ResolveCredential say an ended
// ctx does. The stand-in answers the profile only when its request's context
// is done, after the test has cancelled it.
func TestKiloVerify_EndedContext_0080(t *testing.T) {
	d := descriptor0056(t, llmprovider.ProviderKilo)
	endsDuringRead := func(k *kilo0080, cancel context.CancelFunc) {
		k.profile = func(r *http.Request) (*http.Response, error) {
			cancel()
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
	}
	assertEnded := func(t *testing.T, err error, f *fakePrompter) {
		t.Helper()
		if !errors.Is(err, context.Canceled) || errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("error %v; want ctx's error, not a refusal", err)
		}
		for _, notice := range f.seenNotify {
			if strings.Contains(notice, "could not verify") || strings.Contains(notice, "could not read") ||
				strings.Contains(notice, "refused") {
				t.Errorf("notice %q; want neither a warning nor a refusal for an ended ctx", notice)
			}
		}
	}
	t.Run("typed key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		k := newKilo0080(kiloGood0080)
		endsDuringRead(k, cancel)
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, secrets: []string{kiloGood0080}})
		_, _, err := credentialStep(ctx, f, d, k.options(t))
		assertEnded(t, err, f)
		if len(f.seenSecret) != 1 {
			t.Errorf("%d keys asked; want one, not asked again", len(f.seenSecret))
		}
	})
	t.Run("device session", func(t *testing.T) {
		stubKiloDeviceLogin0080(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		k := newKilo0080()
		endsDuringRead(k, cancel)
		store := newStore0059()
		o := k.options(t)
		o.TokenStore = store
		f := newFake(t, fakePrompter{inputs: []string{acceptDefault}, selects: []int{1}})
		_, _, err := credentialStep(ctx, f, d, o)
		assertEnded(t, err, f)
		if saves := store.recorded(); len(saves) != 0 || store.held[llmprovider.ProviderKilo] != nil {
			t.Errorf("saves %+v; want none after ctx ended", saves)
		}
	})
}

// TestKiloVerify_RefusalShowsNoKey_0080 (0056-MADR D3): a service that echoes
// the Authorization header, and the key in the shapes the SDK's redaction
// knows (a token after "token", a bearer), in its 401 answer shows nothing of
// the key: not in a notice, not in the error the run returns. Redaction is by
// pattern, not by the key's value (0080-MADR, Consequences, N-4), so this
// holds for those shapes only; notices never show a key whatever the shape.
func TestKiloVerify_RefusalShowsNoKey_0080(t *testing.T) {
	k := newKilo0080()
	k.profile = func(r *http.Request) (*http.Response, error) {
		resp := respond0080(r, http.StatusUnauthorized,
			`{"error":"Invalid token `+kiloBad0080+`","echo":"`+r.Header.Get("Authorization")+`"}`)
		resp.Header.Set("X-Echo", r.Header.Get("Authorization"))
		resp.Header.Set("WWW-Authenticate", `Bearer error="invalid_token", token="`+kiloBad0080+`"`)
		return resp, nil
	}
	f := newFake(t, fakePrompter{inputs: []string{acceptDefault},
		secrets: []string{kiloBad0080, kiloBad0080, kiloBad0080}})
	_, _, err := ResolveCredential(context.Background(), f, k.options(t), llmprovider.ProviderKilo)
	wantRefused0080(t, err, "refused 3 keys")
	if err != nil && strings.Contains(err.Error(), kiloBad0080) {
		t.Errorf("the error's text shows the key")
	}
	assertNoKiloServiceText0080(t, f)
}

// TestDiscoverLimit_0080 (Q5): a check is bound by Options.DiscoverLimit, and
// by 10 s, the one constant signin.VerifyKilo's bound is too, when it is zero.
func TestDiscoverLimit_0080(t *testing.T) {
	got := (Options{}).discoverLimit()
	if got != defaultDiscoverLimit {
		t.Errorf("default limit %v; want defaultDiscoverLimit, %v", got, defaultDiscoverLimit)
	}
	if got != 10*time.Second {
		t.Errorf("default limit %v; want 10 s", got)
	}
	if got := (Options{DiscoverLimit: 3 * time.Second}).discoverLimit(); got != 3*time.Second {
		t.Errorf("limit %v; want the caller's 3 s", got)
	}
}
