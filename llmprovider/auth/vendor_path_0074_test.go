package auth

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestVendorCLIPath_0074 pins each CLI's own resolution (0074-MADR D10;
// 0012-MADR §5.1): the environment is read only through lookupEnv, the home
// directory is HOME or, on Windows, USERPROFILE, and a nil lookupEnv finds
// none.
func TestVendorCLIPath_0074(t *testing.T) {
	const home = "/home/user"
	env := func(vars map[string]string) func(string) string {
		return func(name string) string { return vars[name] }
	}
	for _, tc := range []struct {
		name     string
		provider llmprovider.ProviderID
		goos     string
		vars     map[string]string
		want     string
	}{
		{"codex home", llmprovider.ProviderOpenAI, "linux", map[string]string{"CODEX_HOME": "/x/codex", "HOME": home}, filepath.Join("/x/codex", "auth.json")},
		{"codex default", llmprovider.ProviderOpenAI, "linux", map[string]string{"HOME": home}, filepath.Join(home, ".codex", "auth.json")},
		{"codex default on windows", llmprovider.ProviderOpenAI, "windows", map[string]string{"USERPROFILE": `C:\u`, "HOME": home}, filepath.Join(`C:\u`, ".codex", "auth.json")},
		{"grok auth path before home", llmprovider.ProviderGrok, "linux",
			map[string]string{"GROK_AUTH_PATH": "/y/login.json", "GROK_HOME": "/x/grok", "HOME": home}, "/y/login.json"},
		{"grok home", llmprovider.ProviderGrok, "linux", map[string]string{"GROK_HOME": "/x/grok", "HOME": home}, filepath.Join("/x/grok", "auth.json")},
		{"grok default", llmprovider.ProviderGrok, "linux", map[string]string{"HOME": home}, filepath.Join(home, ".grok", "auth.json")},
		{"grok default on windows", llmprovider.ProviderGrok, "windows", map[string]string{"USERPROFILE": `C:\u`, "HOME": home}, filepath.Join(`C:\u`, ".grok", "auth.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := vendorPathOn(tc.provider, env(tc.vars), tc.goos)
			if err != nil || got != tc.want {
				t.Fatalf("vendorPathOn = %q, %v; want %q", got, err, tc.want)
			}
		})
	}

	t.Run("the exported function reads this OS's home", func(t *testing.T) {
		name := "HOME"
		if runtime.GOOS == "windows" {
			name = "USERPROFILE"
		}
		got, err := VendorCLIPath(llmprovider.ProviderOpenAI, env(map[string]string{name: home}))
		if want := filepath.Join(home, ".codex", "auth.json"); err != nil || got != want {
			t.Fatalf("VendorCLIPath = %q, %v; want %q", got, err, want)
		}
	})
	t.Run("a nil lookupEnv finds no home", func(t *testing.T) {
		for _, provider := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGrok} {
			path, err := VendorCLIPath(provider, nil)
			if err == nil || path != "" {
				t.Errorf("%s: VendorCLIPath(nil) = %q, %v; want an error", provider, path, err)
			}
		}
	})
	t.Run("no home variable is an error", func(t *testing.T) {
		if path, err := VendorCLIPath(llmprovider.ProviderOpenAI, env(nil)); err == nil || path != "" {
			t.Errorf("VendorCLIPath with no HOME = %q, %v; want an error", path, err)
		}
	})
	t.Run("Kilo has no vendor CLI", func(t *testing.T) {
		path, err := VendorCLIPath(llmprovider.ProviderKilo, env(map[string]string{"HOME": home}))
		if !errors.Is(err, llmprovider.ErrUnsupported) || path != "" {
			t.Errorf("VendorCLIPath(kilo) = %q, %v; want ErrUnsupported", path, err)
		}
	})
}
