package wizard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0059-MADR W8 to W11: an ended context, a plain-http endpoint the caller
// holds, raw mode, and the pasted token's client.

// TestConfigure_EndedContextAsksNothing_0059 (P6): ConfigureLLM on an ended
// ctx asks nothing and returns ctx's error.
func TestConfigure_EndedContextAsksNothing_0059(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := newFake(t, fakePrompter{})
	_, err := ConfigureLLM(ctx, f, Options{})
	if !errors.Is(err, context.Canceled) || len(f.allText) != 0 {
		t.Errorf("err %v, shown %q; want context.Canceled and no question", err, f.allText)
	}
}

// TestBaseURL_NoPromptPlainHTTP_0059: under NoEndpointPrompt, the plain-http
// remote URL Existing holds is used with a warning and no question; a typed
// one is still asked about.
func TestBaseURL_NoPromptPlainHTTP_0059(t *testing.T) {
	const plain = "http://models.example.test/v1"
	existing := Result{Provider: llmprovider.ProviderTogether, BaseURL: plain}
	f := newFake(t, fakePrompter{secrets: []string{testKey}})
	res, _, err := ResolveCredential(context.Background(), f,
		Options{Existing: existing, NoEndpointPrompt: true}, llmprovider.ProviderTogether)
	if err != nil || res.BaseURL != plain || len(f.seenConfirm) != 0 ||
		countContaining(f.seenNotify, "plain http to a remote host") != 1 {
		t.Errorf("base URL %q, %v, confirms %q, notices %q; want %s, no confirm, one warning",
			res.BaseURL, err, f.seenConfirm, f.seenNotify, plain)
	}
	f = newFake(t, fakePrompter{inputs: []string{plain}, confirms: []bool{true}, secrets: []string{testKey}})
	res, _, err = ResolveCredential(context.Background(), f, Options{Existing: existing}, llmprovider.ProviderTogether)
	if err != nil || res.BaseURL != plain || len(f.seenConfirm) != 1 ||
		!strings.Contains(f.seenConfirm[0], "plain http to a remote host") {
		t.Errorf("typed: base URL %q, %v, confirms %q; want %s, asked about first", res.BaseURL, err, f.seenConfirm, plain)
	}
}

// rawSeam0059 replaces the terminal functions for t: a terminal, raw mode
// refused with rawErr, and the restore failing with restoreErr. It counts
// the raw-mode calls and the restores.
func rawSeam0059(t *testing.T, rawErr, restoreErr error) (made, restored *int) {
	t.Helper()
	origTerm, origRaw, origRestore := isTerminal, makeRaw, restoreTerm
	t.Cleanup(func() { isTerminal, makeRaw, restoreTerm = origTerm, origRaw, origRestore })
	made, restored = new(int), new(int)
	isTerminal = func(int) bool { return true }
	makeRaw = func(int) (*term.State, error) {
		*made++
		if rawErr != nil {
			return nil, rawErr
		}
		return &term.State{}, nil
	}
	restoreTerm = func(int, *term.State) error {
		*restored++
		return restoreErr
	}
	return made, restored
}

// pipe0059 is a TextPrompter reading typed from an os.Pipe, its output in
// the returned buffer.
func pipe0059(t *testing.T, typed string) (*TextPrompter, *bytes.Buffer) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if _, err := w.WriteString(typed); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	return &TextPrompter{In: r, Out: out}, out
}

// TestSecret_RawMode_0059: on a terminal, secretOnce reads masked in raw mode
// and restores it; when raw mode is refused it reads visibly and says so; a
// failed restore is an error notice. The key is never shown.
func TestSecret_RawMode_0059(t *testing.T) {
	const key = "sk-raw-0059-keyAA"
	for _, c := range []struct {
		name                string
		rawErr, restoreErr  error
		typed               string
		wantRestores        int
		wantShown, notShown string
	}{
		{"raw mode", nil, nil, key + "\r", 1, "Key: ", "hidden input unavailable"},
		{"raw mode refused", errors.New("not a console"), nil, key + "\n", 0,
			"hidden input unavailable: not a console", "failed to restore"},
		{"the restore failing", nil, errors.New("bad descriptor"), key + "\r", 1,
			"error: failed to restore terminal mode: bad descriptor", "hidden input unavailable"},
	} {
		t.Run(c.name, func(t *testing.T) {
			made, restored := rawSeam0059(t, c.rawErr, c.restoreErr)
			p, out := pipe0059(t, c.typed)
			got, err := p.secretOnce("Key")
			shown := out.String()
			if err != nil || got != key || *made != 1 || *restored != c.wantRestores {
				t.Errorf("secretOnce = %q, %v; %d raw, %d restores; want the key, 1 raw, %d restores",
					got, err, *made, *restored, c.wantRestores)
			}
			if !strings.Contains(shown, c.wantShown) || strings.Contains(shown, c.notShown) {
				t.Errorf("shown %q; want %q and not %q", shown, c.wantShown, c.notShown)
			}
			if c.rawErr == nil && strings.Contains(shown, key) {
				t.Errorf("the masked entry shows the key: %q", shown)
			}
		})
	}
}

// TestPastedToken_KeepsClient_0059: the session of a pasted ChatGPT access
// token keeps Options.HTTPClient.
func TestPastedToken_KeepsClient_0059(t *testing.T) {
	client := &http.Client{}
	d := llmprovider.Descriptor{ID: llmprovider.ProviderOpenAI, Label: "OpenAI"}
	f := newFake(t, fakePrompter{secrets: []string{testJWT(fmt.Sprintf(`{"exp":%d}`, time.Now().Add(time.Hour).Unix()))}})
	res, err := resolveTokenStdin(context.Background(), f, d, Options{TokenStore: newMemoryTokenStore(), HTTPClient: client})
	if err != nil || res.session == nil || res.session.HTTPClient != client {
		t.Errorf("resolveTokenStdin: %v; session %v; want one holding the caller's client", err, res.session)
	}
}
