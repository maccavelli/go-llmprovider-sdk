package transport

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime/debug"
	"testing"
	"time"
)

// TestParseRetryAfter and TestParseRetryAfter_HTTPDate were llmprovider's
// (provider_correctness_test.go and provider_test.go; 0015-PLAN S7b).
func TestParseRetryAfter(t *testing.T) {
	if got := ParseRetryAfter("120"); got != 120*time.Second {
		t.Errorf("seconds: got %v", got)
	}
	if got := ParseRetryAfter(""); got != 0 {
		t.Errorf("empty: got %v", got)
	}
	if got := ParseRetryAfter("not-a-number"); got != 0 {
		t.Errorf("garbage: got %v", got)
	}
	if got := ParseRetryAfter(time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)); got != 0 {
		t.Errorf("a date in the past: got %v", got)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	future := time.Now().Add(30 * time.Second).UTC().Format(http.TimeFormat)
	d := ParseRetryAfter(future)
	if d <= 0 || d > 35*time.Second {
		t.Errorf("ParseRetryAfter(%q) = %v, expected ~30s", future, d)
	}
}

// TestRetryAfter: retry-after-ms wins, in fractional milliseconds; otherwise
// Retry-After (MADR 0012 §1.2).
func TestRetryAfter(t *testing.T) {
	h := http.Header{}
	h.Set("Retry-After", "7")
	if got := RetryAfter(h); got != 7*time.Second {
		t.Errorf("Retry-After only: %v, want 7s", got)
	}
	h.Set("Retry-After-Ms", "1500.5")
	if got := RetryAfter(h); got != 1500500*time.Microsecond {
		t.Errorf("retry-after-ms: %v, want 1.5005s", got)
	}
}

// TestIdentity_UserAgent: the application leads, then the platform and this
// module's version (MADR 0012 §1.4).
func TestIdentity_UserAgent(t *testing.T) {
	id := Identity{Name: "app", Version: "1.2.3", Session: "s"}
	ua := id.UserAgent()
	if !regexp.MustCompile(`^app/1\.2\.3 \(\w+; \w+\) go-llmprovider-sdk/\S+$`).MatchString(ua) {
		t.Errorf("UserAgent = %q", ua)
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.invalid/", http.NoBody)
	id.SetUserAgent(req)
	if req.Header.Get("User-Agent") != ua {
		t.Errorf("User-Agent header = %q, want %q", req.Header.Get("User-Agent"), ua)
	}
}

// TestVersionsOf: this module's version comes from its dependency entry, or
// from the main module when it is the main module; a missing version is
// "(devel)".
func TestVersionsOf(t *testing.T) {
	for _, tc := range []struct {
		name      string
		info      *debug.BuildInfo
		ok        bool
		sdk, main string
	}{
		{"no build info", nil, false, DevelVersion, DevelVersion},
		{"a dependency", &debug.BuildInfo{Main: debug.Module{Path: "example.com/app", Version: "v2.0.0"},
			Deps: []*debug.Module{{Path: "other", Version: "v9"}, {Path: sdkModulePath, Version: "v1.4.0"}}}, true, "v1.4.0", "v2.0.0"},
		{"the main module", &debug.BuildInfo{Main: debug.Module{Path: sdkModulePath, Version: "v1.5.0"}}, true, "v1.5.0", "v1.5.0"},
		{"no versions", &debug.BuildInfo{Main: debug.Module{Path: "example.com/app"}}, true, DevelVersion, DevelVersion},
	} {
		if sdk, main := versionsOf(tc.info, tc.ok); sdk != tc.sdk || main != tc.main {
			t.Errorf("%s: versionsOf = %q, %q; want %q, %q", tc.name, sdk, main, tc.sdk, tc.main)
		}
	}
}
