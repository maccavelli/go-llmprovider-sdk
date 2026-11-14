package llmprovider

import (
	"strings"
	"testing"
)

// versionOfUserAgent0079 is the version a User-Agent leads with: what follows
// the first "/" up to the platform.
func versionOfUserAgent0079(t *testing.T, ua string) string {
	t.Helper()
	lead, _, ok := strings.Cut(ua, " (")
	_, version, ok2 := strings.Cut(lead, "/")
	if !ok || !ok2 {
		t.Fatalf("User-Agent %q does not lead with name/version (", ua)
	}
	return version
}

// TestSettingsClientVersion_0079 (0079-MADR D4): ClientVersion is the version
// WithClientInfo gave, or the version UserAgent shows when it gave none.
func TestSettingsClientVersion_0079(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []Option
		want string // empty: the version UserAgent shows
	}{
		{"given", []Option{WithClientInfo("a", "1.2.3")}, "1.2.3"},
		{"name only", []Option{WithClientInfo("a", "")}, ""},
		{"no option", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, err := ResolveOptions(ProviderOpenAI, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.want
			if want == "" {
				want = versionOfUserAgent0079(t, st.UserAgent())
			}
			if got := st.ClientVersion(); got != want || got == "" {
				t.Errorf("ClientVersion() = %q, want %q", got, want)
			}
		})
	}
}
