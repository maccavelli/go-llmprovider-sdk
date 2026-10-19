package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"syscall"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// errPrompt0048 is a Prompter's own error, which ConfigureLLM passes on.
var errPrompt0048 = errors.New("prompter: the terminal went away")

// failOnce0048 is a fakePrompter whose first call of the method named fail
// returns errPrompt0048.
type failOnce0048 struct {
	*fakePrompter
	fail string
}

func (f *failOnce0048) take(method string) bool {
	if f.fail != method {
		return false
	}
	f.fail = ""
	return true
}

func (f *failOnce0048) Select(title string, choices []Choice, defaultIdx int) (int, error) {
	if f.take("Select") {
		return 0, errPrompt0048
	}
	return f.fakePrompter.Select(title, choices, defaultIdx)
}

func (f *failOnce0048) Confirm(question string, def bool) (bool, error) {
	if f.take("Confirm") {
		return false, errPrompt0048
	}
	return f.fakePrompter.Confirm(question, def)
}

func (f *failOnce0048) Input(prompt, def string) (string, error) {
	if f.take("Input") {
		return "", errPrompt0048
	}
	return f.fakePrompter.Input(prompt, def)
}

func (f *failOnce0048) Secret(prompt string) (string, error) {
	if f.take("Secret") {
		return "", errPrompt0048
	}
	return f.fakePrompter.Secret(prompt)
}

// label0048 is a built-in provider's menu label.
func label0048(t *testing.T, id llmprovider.ProviderID) string {
	t.Helper()
	return providers.Default().Descriptors()[providerIdx(t, id)].Label
}

// TestConfigureLLM_SelectErrorWrapped_0048: a Prompter's error at the
// provider menu is returned wrapped, and matches with errors.Is (0048-MADR
// D4).
func TestConfigureLLM_SelectErrorWrapped_0048(t *testing.T) {
	p := &failOnce0048{fakePrompter: newFake(t, fakePrompter{}), fail: "Select"}
	_, err := ConfigureLLM(context.Background(), p, Options{})
	if !errors.Is(err, errPrompt0048) || err.Error() != "select provider: "+errPrompt0048.Error() {
		t.Errorf("ConfigureLLM = %v; want the Prompter's error, wrapped by select provider", err)
	}
}

// TestConfigureLLM_SecretErrorWrapped_0048: a Prompter's error at the API key
// prompt is returned wrapped, and matches with errors.Is (0048-MADR D4).
func TestConfigureLLM_SecretErrorWrapped_0048(t *testing.T) {
	p := &failOnce0048{fakePrompter: newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderClaude)}}),
		fail: "Secret"}
	_, err := ConfigureLLM(context.Background(), p, Options{})
	if !errors.Is(err, errPrompt0048) || err.Error() != "enter API key: "+errPrompt0048.Error() {
		t.Errorf("ConfigureLLM = %v; want the Prompter's error, wrapped by enter API key", err)
	}
}

// TestConfigureLLM_InputErrorWrapped_0048: a Prompter's error at the endpoint
// prompt is returned wrapped, and matches with errors.Is (0048-MADR D4).
func TestConfigureLLM_InputErrorWrapped_0048(t *testing.T) {
	p := &failOnce0048{fakePrompter: newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOllama)}}),
		fail: "Input"}
	_, err := ConfigureLLM(context.Background(), p, Options{HTTPClient: nettest.Refusing()})
	if !errors.Is(err, errPrompt0048) || err.Error() != "enter base URL: "+errPrompt0048.Error() {
		t.Errorf("ConfigureLLM = %v; want the Prompter's error, wrapped by enter base URL", err)
	}
}

// TestConfigureLLM_ConfirmErrorBare_0048: a Prompter's error at the
// keep-the-saved-key question is returned as it is (0048-MADR D4).
func TestConfigureLLM_ConfirmErrorBare_0048(t *testing.T) {
	p := &failOnce0048{fakePrompter: newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderClaude)}}),
		fail: "Confirm"}
	_, err := ConfigureLLM(context.Background(), p, Options{
		Existing: Result{Provider: llmprovider.ProviderClaude, APIKey: "existing-key-abcd"},
	})
	if !errors.Is(err, errPrompt0048) || err.Error() != errPrompt0048.Error() {
		t.Errorf("ConfigureLLM = %v; want the Prompter's error as it is", err)
	}
}

// TestConfigureLLM_UnknownProvidersIsPlain_0048: Options.Providers that
// match no registered provider is a plain error (0048-MADR D4).
func TestConfigureLLM_UnknownProvidersIsPlain_0048(t *testing.T) {
	_, err := ConfigureLLM(context.Background(), newFake(t, fakePrompter{}),
		Options{Providers: []llmprovider.ProviderID{"nonexistent"}})
	assertPlain0048(t, err, "wizard: none of the requested providers exist: [nonexistent]")
}

// TestConfigureLLM_OutOfRangeIndexIsPlain_0048: a Prompter answering an index
// out of range is a plain error (0048-MADR D4).
func TestConfigureLLM_OutOfRangeIndexIsPlain_0048(t *testing.T) {
	_, err := ConfigureLLM(context.Background(), &badIndex{newFake(t, fakePrompter{})}, Options{})
	assertPlain0048(t, err, fmt.Sprintf(`select provider: wizard: "Choose an LLM provider:": the prompter chose -1 of %d choices`,
		len(providers.Default().Descriptors())))
}

// TestConfigureLLM_NoModelsBlankIDIsPlain_0048: no models found and a blank
// id entered is a plain error (0048-MADR D4).
func TestConfigureLLM_NoModelsBlankIDIsPlain_0048(t *testing.T) {
	f := newFake(t, fakePrompter{
		selects:  []int{providerIdx(t, llmprovider.ProviderOllama)},
		inputs:   []string{nettest.RefusedURL, "  "},
		confirms: []bool{false},
	})
	_, err := ConfigureLLM(context.Background(), f, Options{HTTPClient: nettest.Refusing()})
	assertPlain0048(t, err, fmt.Sprintf("wizard: no model available for %s and none entered",
		label0048(t, llmprovider.ProviderOllama)))
}

// TestConfigureLLM_DeviceLoginErrorBare_0048: a failed device sign-in's
// error, the transport's *url.Error refusing the connection, is returned as
// the login gave it (0048-MADR D4).
func TestConfigureLLM_DeviceLoginErrorBare_0048(t *testing.T) {
	// The provider, then OpenAI's device-code method.
	f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 2}})
	_, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), HTTPClient: nettest.Refusing()})
	var urlErr *url.Error
	if !errors.As(err, &urlErr) || !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("ConfigureLLM = %v; want the login's *url.Error refusing the connection", err)
	}
	_, loginErr := auth.LoginDeviceOAuth(context.Background(), llmprovider.ProviderOpenAI,
		auth.OAuthFlowOptions{HTTPClient: nettest.Refusing()})
	if loginErr == nil || err.Error() != loginErr.Error() {
		t.Errorf("ConfigureLLM = %q; want the login's own error %q, unwrapped", err, loginErr)
	}
	if len(f.seenSelect) != 2 || !strings.Contains(f.seenSelect[1], "authenticate") {
		t.Errorf("menus = %q; want the provider and the sign-in method", f.seenSelect)
	}
}

// zeroSelect0048 answers every Select with 0 and records the choices given.
type zeroSelect0048 struct {
	*fakePrompter
	seen [][]Choice
}

func (z *zeroSelect0048) Select(_ string, choices []Choice, _ int) (int, error) {
	z.seen = append(z.seen, choices)
	return 0, nil
}

// TestConfigureLLM_EmptyRegistryIsPlain_0048: an empty Registry asks the
// Prompter to Select from no choices; an answer of 0 is a plain error, and
// TextPrompter's own refusal is passed on (0048-MADR D4).
func TestConfigureLLM_EmptyRegistryIsPlain_0048(t *testing.T) {
	defer func() {
		if v := recover(); v != nil {
			t.Fatalf("ConfigureLLM panicked on an empty Registry: %v", v)
		}
	}()
	z := &zeroSelect0048{fakePrompter: newFake(t, fakePrompter{})}
	_, err := ConfigureLLM(context.Background(), z, Options{Registry: llmprovider.NewRegistry()})
	assertPlain0048(t, err, `select provider: wizard: "Choose an LLM provider:": the prompter chose 0 of 0 choices`)
	if len(z.seen) != 1 || len(z.seen[0]) != 0 {
		t.Errorf("Select calls = %v; want one, with no choices", z.seen)
	}
	tp := &TextPrompter{In: strings.NewReader("1\n"), Out: io.Discard}
	_, err = ConfigureLLM(context.Background(), tp, Options{Registry: llmprovider.NewRegistry()})
	assertPlain0048(t, err, "select provider: wizard: Select called with no choices")
}
