package auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestSignInSentinels_0074: each of the six sites' errors matches
// ErrAuthFailure and its own sentinel, ErrSignInDenied or ErrSignInExpired,
// and no other kind (0074-MADR D8).
func TestSignInSentinels_0074(t *testing.T) {
	want := map[string]error{
		"browser denial":       ErrSignInDenied,
		"Grok device denial":   ErrSignInDenied,
		"Grok device expiry":   ErrSignInExpired,
		"shared device expiry": ErrSignInExpired,
		"Kilo denial":          ErrSignInDenied,
		"Kilo expiry":          ErrSignInExpired,
	}
	runs := signInRuns0074()
	if len(runs) != len(want) {
		t.Fatalf("%d sites run, %d expected; the two tables disagree", len(runs), len(want))
	}
	for _, tc := range runs {
		t.Run(tc.name, func(t *testing.T) {
			sentinel, ok := want[tc.name]
			if !ok {
				t.Fatalf("no sentinel expected for %q", tc.name)
			}
			assertSignInKind0074(t, tc.run(t), sentinel)
		})
	}
}

// TestSignInSentinels_OpenAIDeadline_0074: the OpenAI poll's own 15-minute
// deadline is the shared expiry (0076-MADR D2), so it matches
// ErrSignInExpired too.
func TestSignInSentinels_OpenAIDeadline_0074(t *testing.T) {
	err := deviceWait0074(t, llmprovider.ProviderOpenAI, reply0048(t, openAIPoll0048, http.StatusForbidden, "{}"))
	assertSignInKind0074(t, err, ErrSignInExpired)
}

// assertSignInKind0074 fails t unless err matches llmprovider.ErrAuthFailure
// and sentinel, and neither the other sentinel nor any other kind.
func assertSignInKind0074(t *testing.T, err, sentinel error) {
	t.Helper()
	if err == nil {
		t.Fatal("err = nil, want a sign-in error")
	}
	other := ErrSignInExpired
	if errors.Is(sentinel, ErrSignInExpired) {
		other = ErrSignInDenied
	}
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !errors.Is(err, sentinel) {
		t.Errorf("err = %v; want it to match ErrAuthFailure and %v", err, sentinel)
	}
	if errors.Is(err, other) {
		t.Errorf("err = %v; matches %v too", err, other)
	}
	for _, kind := range kindSentinels0048 {
		if !errors.Is(kind, llmprovider.ErrAuthFailure) && errors.Is(err, kind) {
			t.Errorf("err = %v; matches %v", err, kind)
		}
	}
}

// TestSignInSentinels_Distinct_0074: the sentinels are not each other, and
// each matches ErrAuthFailure.
func TestSignInSentinels_Distinct_0074(t *testing.T) {
	if errors.Is(ErrSignInDenied, ErrSignInExpired) || errors.Is(ErrSignInExpired, ErrSignInDenied) {
		t.Error("ErrSignInDenied and ErrSignInExpired match each other")
	}
	for _, sentinel := range []error{ErrSignInDenied, ErrSignInExpired} {
		if !errors.Is(sentinel, llmprovider.ErrAuthFailure) {
			t.Errorf("%v does not match ErrAuthFailure", sentinel)
		}
	}
	// A wrapped sign-in error, as a caller's %w gives it, still matches.
	wrapped := errors.Join(errors.New("context"), asSignInError(errors.New("plain"), ErrSignInDenied))
	if !errors.Is(wrapped, ErrSignInDenied) {
		t.Error("a joined sign-in error no longer matches its sentinel")
	}
}
