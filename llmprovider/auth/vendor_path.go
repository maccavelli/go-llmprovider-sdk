package auth

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// VendorCLIPath resolves a vendor CLI's auth file as the CLI does, for
// VendorCLISession's Path (0074-MADR D10; 0012-MADR §5.1). Codex, for
// llmprovider.ProviderOpenAI: $CODEX_HOME/auth.json, else
// ~/.codex/auth.json. Grok, for llmprovider.ProviderGrok: a non-empty
// $GROK_AUTH_PATH verbatim, else $GROK_HOME/auth.json, else
// ~/.grok/auth.json. The home directory is HOME, or USERPROFILE on Windows.
//
// It reads the environment only through lookupEnv (0015-MADR D9), such as
// os.Getenv, and reads no file. A nil lookupEnv finds no variable and no
// home directory, and the error says to pass one. A provider with no vendor
// CLI gives an error matching llmprovider.ErrUnsupported. It is safe for
// concurrent use when lookupEnv is.
func VendorCLIPath(provider llmprovider.ProviderID, lookupEnv func(string) string) (string, error) {
	return vendorPathOn(provider, lookupEnv, runtime.GOOS)
}

// vendorPathOn is VendorCLIPath for the named operating system.
func vendorPathOn(provider llmprovider.ProviderID, lookupEnv func(string) string, goos string) (string, error) {
	env := lookupEnv
	if env == nil {
		env = func(string) string { return "" }
	}
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
		return "", fmt.Errorf("auth: %w: provider %q has no vendor CLI login", llmprovider.ErrUnsupported, provider)
	}
	if dir := env(homeEnv); dir != "" {
		return filepath.Join(dir, "auth.json"), nil
	}
	home, err := vendorHomeDir(lookupEnv, goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultDir, "auth.json"), nil
}

// vendorHomeDir is the home directory from the caller's environment:
// USERPROFILE on Windows, HOME elsewhere, as os.UserHomeDir reads it. With no
// lookupEnv there is no home directory, and the error says what to pass
// (0020-MADR F25, Q5 a).
func vendorHomeDir(lookupEnv func(string) string, goos string) (string, error) {
	if lookupEnv == nil {
		return "", errors.New("auth: a vendor CLI's default auth path is under the home directory; " +
			"pass a lookupEnv, such as os.Getenv, to find it")
	}
	name := "HOME"
	if goos == "windows" {
		name = "USERPROFILE"
	}
	if home := lookupEnv(name); home != "" {
		return home, nil
	}
	return "", fmt.Errorf("auth: resolve home directory: %s is not set", name)
}
