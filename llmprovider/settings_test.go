package llmprovider

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestResolveOptions_RefusesAForeignOption(t *testing.T) {
	_, err := ResolveOptions(ProviderClaude, []Option{ScopedOption(ProviderKilo, "kilo.WithOrganization", "org")})
	if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), `kilo.WithOrganization is for provider "kilo", not "claude"`) {
		t.Fatalf("err = %v, want the foreign option named", err)
	}
}

func TestResolveOptions_KeepsTheProvidersOwnValuesInOrder(t *testing.T) {
	st, err := ResolveOptions(ProviderKilo, []Option{
		ScopedOption(ProviderKilo, "kilo.WithOrganization", "org"),
		WithModel("m"),
		ScopedOption(ProviderKilo, "kilo.WithOther", 2),
	})
	if err != nil {
		t.Fatal(err)
	}
	values := st.Values()
	if len(values) != 2 || values[0] != "org" || values[1] != 2 {
		t.Fatalf("Values() = %v, want [org 2]", values)
	}
	values[0] = "changed"
	if st.Values()[0] != "org" {
		t.Fatal("a caller's change reached the Settings")
	}
}

func TestResolveOptions_CommonOptions(t *testing.T) {
	client := &http.Client{}
	logger := slog.New(slog.DiscardHandler)
	reasoning := &Reasoning{Effort: EffortLow}
	src := NewStaticToken("tok")
	st, err := ResolveOptions(ProviderOpenAI, []Option{
		WithModel("m"),
		WithAPIKey("key"),
		WithTokenSource(src), // later wins over WithAPIKey
		WithHTTPClient(client),
		WithBaseURL("http://base"),
		WithMaxTokens(77),
		WithLogger(logger),
		WithReasoning(reasoning),
		WithClientInfo("app", "1.2.3"),
		WithSessionID("sess"),
		{}, // a zero Option changes nothing
	})
	if err != nil {
		t.Fatal(err)
	}
	if st.Model() != "m" || st.TokenSource() != src || st.HTTPClient() != client || st.BaseURL() != "http://base" ||
		st.MaxTokens() != 77 || st.Logger() != logger || st.SessionID() != "sess" {
		t.Fatalf("settings not applied: %+v", st.s)
	}
	if ua := st.UserAgent(); !strings.HasPrefix(ua, "app/1.2.3 (") || !strings.Contains(ua, " go-llmprovider-sdk/") {
		t.Fatalf("UserAgent() = %q", ua)
	}
	reasoning.Effort = EffortHigh
	got := st.Reasoning()
	if got.Effort != EffortLow {
		t.Fatal("a caller's change to the reasoning reached the Settings")
	}
	got.Effort = EffortHigh
	if st.Reasoning().Effort != EffortLow {
		t.Fatal("a change to Reasoning()'s result reached the Settings")
	}
}

func TestResolveOptions_Defaults(t *testing.T) {
	st, err := ResolveOptions(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.Model() != "" || st.TokenSource() != nil || st.HTTPClient() == nil || st.MaxTokens() != 8192 || st.Reasoning() != nil {
		t.Fatalf("defaults %+v", st.s)
	}
	if st.Logger() == nil || st.Logger().Enabled(context.Background(), slog.LevelError) {
		t.Fatal("the default logger must exist and discard everything")
	}
	other, _ := ResolveOptions(ProviderOpenAI, nil)
	if st.SessionID() == "" || st.SessionID() == other.SessionID() {
		t.Fatalf("session ids %q and %q; want a random one per Settings", st.SessionID(), other.SessionID())
	}
	if !strings.HasPrefix(st.UserAgent(), "go-llmprovider-sdk/") {
		t.Fatalf("UserAgent() = %q; want this module by default", st.UserAgent())
	}
	if other.HTTPClient() == st.HTTPClient() {
		t.Fatal("two Settings share a default client; each provider gets its own (0016-MADR D8)")
	}
	if WithReasoning(nil).apply(&st.s); st.Reasoning() != nil {
		t.Fatal("WithReasoning(nil) must clear it")
	}
}

// TestResolveOptions_ModelProbesAreCommon: A5's probe options reach the new
// API as common options (0015-MADR amendment of 2026-09-30, D5 step 1).
func TestResolveOptions_ModelProbesAreCommon(t *testing.T) {
	for _, c := range []struct {
		name string
		env  string // "" leaves LLMPROVIDER_PROBES unset
		opts func() []Option
		want bool
	}{
		{"default", "", func() []Option { return nil }, true},
		{"option off", "", func() []Option { return []Option{WithModelProbes(false)} }, false},
		{"option on after off", "", func() []Option { return []Option{WithModelProbes(false), WithModelProbes(true)} }, true},
		{"env false through the helper", "false", func() []Option { return []Option{ModelProbesFromEnv()} }, false},
		{"env not a boolean", "sometimes", func() []Option { return []Option{ModelProbesFromEnv()} }, true},
		{"env unset through the helper", "", func() []Option { return []Option{ModelProbesFromEnv()} }, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(envModelProbes, c.env)
			if c.env == "" {
				if err := os.Unsetenv(envModelProbes); err != nil {
					t.Fatal(err)
				}
			}
			st, err := ResolveOptions(ProviderOpenAI, c.opts())
			if err != nil {
				t.Fatalf("ResolveOptions: %v", err)
			}
			if got := st.ModelProbes(); got != c.want {
				t.Fatalf("ModelProbes() = %v, want %v", got, c.want)
			}
		})
	}
}
