//go:build live_gateways

package llmprovider

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// liveVendorSession returns a read-through session on a real CLI login, or
// skips. It never writes the file and never refreshes (MADR 0012 §5.1).
func liveVendorSession(t *testing.T, provider ProviderID, optIn, homeEnv, dir string) *VendorCLISession {
	t.Helper()
	if os.Getenv(optIn) != "1" {
		t.Skipf("%s unset: this spends the CLI's subscription", optIn)
	}
	home := os.Getenv(homeEnv)
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			t.Skip(err)
		}
		home = filepath.Join(userHome, dir)
	}
	path := filepath.Join(home, "auth.json")
	if provider == ProviderGrok && os.Getenv("GROK_AUTH_PATH") != "" {
		path = os.Getenv("GROK_AUTH_PATH")
	}
	s := &VendorCLISession{Provider: provider, Path: path}
	if _, err := s.Token(t.Context()); errors.Is(err, ErrAuthFailure) {
		t.Skipf("no live %s CLI login: %v", provider, err)
	}
	return s
}
