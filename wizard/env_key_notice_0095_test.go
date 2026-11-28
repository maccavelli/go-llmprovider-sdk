package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// TestEnvKeyControlNotice_0095 (0095-MADR FU08): an environment value that
// holds a control character is still offered and, kept, used as it is
// (0085-MADR ST39), and the person is told first, by the variable's name and
// never its value, that the provider refuses it; a clean value brings no
// notice.
func TestEnvKeyControlNotice_0095(t *testing.T) {
	d, _ := providers.Default().Descriptor(llmprovider.ProviderTogether)
	for _, tc := range []struct {
		name, value string
		notice      bool
	}{
		{"a line feed at its end", "made-up-key\n", true},
		{"a tab inside", "made-up\tkey", true},
		{"clean", "made-up-key", false},
	} {
		o := options0085(t)
		o.AllowEnv, o.LookupEnv = true, func(string) string { return tc.value }
		f := newFake(t, fakePrompter{confirms: []bool{true}})
		res, _, err := ResolveCredential(context.Background(), f, o, d.ID)
		if err != nil || res.APIKey != tc.value {
			t.Errorf("%s: err %v, key %q; want the value as it is (ST39)", tc.name, err, res.APIKey)
		}
		var named []string
		for _, n := range f.seenNotify {
			if strings.Contains(n, "control character") {
				named = append(named, n)
			}
		}
		if got := len(named) == 1; got != tc.notice || len(named) > 1 {
			t.Errorf("%s: notices %q; want a control-character notice: %t", tc.name, f.seenNotify, tc.notice)
		}
		for _, n := range named {
			if !strings.Contains(n, d.EnvVar) || strings.Contains(n, "made-up") {
				t.Errorf("%s: notice %q; want the variable %s named and no part of its value", tc.name, n, d.EnvVar)
			}
		}
	}
}
