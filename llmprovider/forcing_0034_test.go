package llmprovider

import (
	"errors"
	"testing"
)

// TestForcedChoiceOptions_0034 (0034-MADR D9, D10): the fallback defaults to
// ForcedChoiceRefuse; the options set it and the caller's per-model layer,
// which Settings returns as a copy, keyed in lower case.
func TestForcedChoiceOptions_0034(t *testing.T) {
	st, err := ResolveOptions("test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.ForcedChoiceFallback() != ForcedChoiceRefuse || len(st.ModelForcedChoices()) != 0 {
		t.Errorf("defaults = %v, %v; want ForcedChoiceRefuse and no overrides", st.ForcedChoiceFallback(), st.ModelForcedChoices())
	}
	st, err = ResolveOptions("test", []Option{WithForcedChoiceFallback(ForcedChoiceDegrade),
		WithModelForcedChoice("Claude-Sonnet-5-5", Supported), WithModelForcedChoice("claude-future-6", Unsupported)})
	if err != nil {
		t.Fatal(err)
	}
	got := st.ModelForcedChoices()
	if st.ForcedChoiceFallback() != ForcedChoiceDegrade || got["claude-sonnet-5-5"] != Supported || got["claude-future-6"] != Unsupported {
		t.Errorf("= %v, %v; want ForcedChoiceDegrade and both overrides", st.ForcedChoiceFallback(), got)
	}
	got["claude-sonnet-5-5"] = Unsupported
	if st.ModelForcedChoices()["claude-sonnet-5-5"] != Supported {
		t.Error("ModelForcedChoices returns the settings' own map (R29)")
	}
}

// TestForcedChoiceOptions_Invalid_0034 (R23): a value no request could use is
// refused by New's resolution, with ErrInvalidRequest.
func TestForcedChoiceOptions_Invalid_0034(t *testing.T) {
	for name, opt := range map[string]Option{
		"an empty model":              WithModelForcedChoice("", Unsupported),
		"BestEffort":                  WithModelForcedChoice("claude-sonnet-5-5", BestEffort),
		"a control character":         WithModelForcedChoice("claude\n", Unsupported),
		"a fallback out of its range": WithForcedChoiceFallback(ForcedChoiceFallback(9)),
	} {
		if _, err := ResolveOptions("test", []Option{opt}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
	}
}
