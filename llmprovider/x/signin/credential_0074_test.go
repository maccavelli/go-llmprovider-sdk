package signin

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestCredentialRedacts_0074: an APIKey, and a DeviceKey, with a planted key
// print no key in %v, %+v, %#v, %s, %q or slog, by value or by pointer.
func TestCredentialRedacts_0074(t *testing.T) {
	const planted = "sk-PLANTED-0074-KEY-VALUE"
	for name, credential := range map[string]Credential{
		"APIKey":    APIKey{Key: planted},
		"DeviceKey": DeviceKey{Key: planted},
	} {
		var outputs []string
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
			outputs = append(outputs, fmt.Sprintf(verb, credential), fmt.Sprintf(verb, &credential))
		}
		for _, handler := range []func(*bytes.Buffer) slog.Handler{
			func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
			func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
		} {
			var buf bytes.Buffer
			slog.New(handler(&buf)).Info("signed in", "credential", credential)
			outputs = append(outputs, buf.String())
		}
		for i, out := range outputs {
			if strings.Contains(out, planted) {
				t.Errorf("%s output %d shows the key: %s", name, i, out)
			}
		}
		token, err := credential.TokenSource().Token(background())
		if err != nil || token.Value != planted {
			t.Errorf("%s: Token = %+v, %v; want the key", name, token, err)
		}
	}
}

// TestCredentialYields_0074: each of the five types reports its contract
// name and a token source that is the credential's.
func TestCredentialYields_0074(t *testing.T) {
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "access-0123456789"}
	vendor := &auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "/nowhere/auth.json"}
	for _, c := range []struct {
		credential Credential
		yields     authcontract.Yields
		source     llmprovider.TokenSource
	}{
		{APIKey{Key: "k"}, authcontract.YieldsAPIKey, nil},
		{Session{OAuth: session}, authcontract.YieldsOAuthSession, session},
		{AccessOnly{OAuth: session}, authcontract.YieldsAccessOnly, session},
		{DeviceKey{Key: "k"}, authcontract.YieldsDeviceKey, nil},
		{ReadThrough{Source: vendor}, authcontract.YieldsReadThrough, vendor},
	} {
		if got := c.credential.Yields(); got != c.yields {
			t.Errorf("%T.Yields = %q; want %q", c.credential, got, c.yields)
		}
		source := c.credential.TokenSource()
		if source == nil {
			t.Errorf("%T.TokenSource is nil", c.credential)
		}
		if c.source != nil && source != c.source {
			t.Errorf("%T.TokenSource is not the credential's own", c.credential)
		}
	}
}

// TestStart_VendorImport_0074: a fixture Codex auth.json under a temporary
// CODEX_HOME gives done and a ReadThrough; a file with no token fails, kind
// auth; Grok's path comes from GROK_AUTH_PATH (0074-MADR D3, D10).
func TestStart_VendorImport_0074(t *testing.T) {
	dir := t.TempDir()
	codex := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(codex, []byte(`{"tokens":{"access_token":"codex-access-token-0074"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	env := func(key string) string {
		if key == "CODEX_HOME" {
			return dir
		}
		return ""
	}
	run, err := Start(background(), llmprovider.ProviderOpenAI, llmprovider.AuthImportVendorCLI, Options{LookupEnv: env})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	credential, err := run.Wait(background())
	if err != nil {
		t.Fatalf("Wait = %v", err)
	}
	through, ok := credential.(ReadThrough)
	if !ok || through.Source == nil || through.Source.Path != codex {
		t.Fatalf("credential = %#v; want a ReadThrough of %s", credential, codex)
	}
	if token, err := credential.TokenSource().Token(background()); err != nil || token.Value != "codex-access-token-0074" {
		t.Errorf("Token = %+v, %v; want the file's token", token, err)
	}
	steps := collect(t, run)
	if len(steps) != 1 || steps[0].Kind != authcontract.StepDone || steps[0].Seq != 1 || steps[0].Yields != authcontract.YieldsReadThrough {
		t.Errorf("steps = %+v; want done #1 with read_through", steps)
	}

	grok := filepath.Join(dir, "grok.json")
	scope := strings.TrimRight(auth.DefaultGrokOAuthIssuer, "/") + "::" + auth.DefaultGrokOAuthClientID
	if err := os.WriteFile(grok, []byte(fmt.Sprintf(`{%q:{"key":"grok-key-0074"}}`, scope)), 0o600); err != nil {
		t.Fatal(err)
	}
	grokEnv := func(key string) string {
		if key == "GROK_AUTH_PATH" {
			return grok
		}
		return ""
	}
	run, err = Start(background(), llmprovider.ProviderGrok, llmprovider.AuthImportVendorCLI, Options{LookupEnv: grokEnv})
	if err != nil {
		t.Fatalf("Start(Grok) = %v", err)
	}
	if credential, err = run.Wait(background()); err != nil || credential.Yields() != authcontract.YieldsReadThrough {
		t.Fatalf("Wait(Grok) = %v, %v", credential, err)
	}

	empty := t.TempDir()
	emptyEnv := func(key string) string {
		if key == "CODEX_HOME" {
			return empty
		}
		return ""
	}
	run, err = Start(background(), llmprovider.ProviderOpenAI, llmprovider.AuthImportVendorCLI, Options{LookupEnv: emptyEnv})
	if err != nil {
		t.Fatalf("Start(empty) = %v", err)
	}
	_, err = run.Wait(background())
	if ErrorKindOf(err) != authcontract.FailureAuth {
		t.Errorf("Wait with no auth file = %v; want kind auth", err)
	}
	steps = collect(t, run)
	if len(steps) != 1 || steps[0].Kind != authcontract.StepFailed || steps[0].Error == nil || steps[0].Error.Kind != authcontract.FailureAuth {
		t.Errorf("steps = %+v; want one failed with kind auth", steps)
	}
}
