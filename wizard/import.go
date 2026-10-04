package wizard

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// vendorAuthPath resolves a vendor CLI's auth file as the CLI does (MADR 0012
// §5.1). Codex: $CODEX_HOME/auth.json, else ~/.codex/auth.json. Grok: a
// non-empty $GROK_AUTH_PATH verbatim, else $GROK_HOME/auth.json, else
// ~/.grok/auth.json (grok-build xai-grok-login/src/storage.rs:45-55,
// xai-dirs/src/lib.rs:43-58).
func vendorAuthPath(provider llmprovider.ProviderID, o Options) (string, error) {
	env := o.lookupEnv()
	var homeEnv, defaultDir string
	switch provider {
	case llmprovider.ProviderOpenAI:
		homeEnv, defaultDir = "CODEX_HOME", ".codex"
	case llmprovider.ProviderGrok:
		if path := env("GROK_AUTH_PATH"); path != "" {
			return path, nil
		}
		homeEnv, defaultDir = "GROK_HOME", ".grok"
	default:
		return "", fmt.Errorf("wizard: provider %q has no vendor CLI login", provider)
	}
	if dir := env(homeEnv); dir != "" {
		return filepath.Join(dir, "auth.json"), nil
	}
	home, err := homeDir(o)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultDir, "auth.json"), nil
}

// homeDir is the home directory from the caller's environment: USERPROFILE
// on Windows, HOME elsewhere, as os.UserHomeDir reads it. The wizard reads no
// ambient state (0015-MADR D9), so with no LookupEnv there is no home
// directory, and the error says what to pass (0020-MADR F25, Q5 a).
func homeDir(o Options) (string, error) {
	if o.LookupEnv == nil {
		return "", errors.New("wizard: a vendor CLI's default auth path is under the home directory; " +
			"pass Options.LookupEnv, such as os.Getenv, to find it")
	}
	name := "HOME"
	if runtime.GOOS == "windows" {
		name = "USERPROFILE"
	}
	if home := o.LookupEnv(name); home != "" {
		return home, nil
	}
	return "", fmt.Errorf("wizard: resolve home directory: %s is not set", name)
}
