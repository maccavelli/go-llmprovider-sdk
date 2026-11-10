package transport

import (
	"net/http"
	"testing"
	"time"
)

// TestRetryAfterAt_0072 (0072-MADR D1): an HTTP-date Retry-After names an
// instant, in UTC, kept even when it has passed; delta seconds and
// retry-after-ms name none; retry-after-ms keeps its precedence; and every
// delay is RetryAfter's.
func TestRetryAfterAt_0072(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	ahead := now.Add(90 * time.Minute)
	ago := now.Add(-time.Hour)
	cases := []struct {
		name   string
		header http.Header
		delay  time.Duration
		at     time.Time
	}{
		{"an HTTP-date ahead", http.Header{"Retry-After": {ahead.Format(http.TimeFormat)}}, 90 * time.Minute, ahead},
		{"an HTTP-date passed", http.Header{"Retry-After": {ago.Format(http.TimeFormat)}}, 0, ago},
		{"delta seconds", http.Header{"Retry-After": {"30"}}, 30 * time.Second, time.Time{}},
		{"retry-after-ms", http.Header{"Retry-After-Ms": {"1500"}}, 1500 * time.Millisecond, time.Time{}},
		{"retry-after-ms over an HTTP-date", http.Header{"Retry-After-Ms": {"250"}, "Retry-After": {ahead.Format(http.TimeFormat)}}, 250 * time.Millisecond, time.Time{}},
		{"neither", http.Header{}, 0, time.Time{}},
		{"unparseable", http.Header{"Retry-After": {"soon"}}, 0, time.Time{}},
	}
	for _, c := range cases {
		delay, at := RetryAfterAt(c.header, now)
		if delay != c.delay || !at.Equal(c.at) {
			t.Errorf("%s: RetryAfterAt = %s, %v; want %s, %v", c.name, delay, at, c.delay, c.at)
		}
		if !at.IsZero() && at.Location() != time.UTC {
			t.Errorf("%s: instant in %v; want UTC", c.name, at.Location())
		}
		// RetryAfter measures from the real clock, so it is compared only
		// where the delay does not depend on the clock.
		if got := RetryAfter(c.header); c.at.IsZero() && got != c.delay {
			t.Errorf("%s: RetryAfter = %s; want RetryAfterAt's %s", c.name, got, c.delay)
		}
	}
}

// TestRateLimitResetAt_0072 (0072-MADR D1): Claude's RFC 3339 headers name
// the latest instant; a longer Go duration header names none; the wait is
// RateLimitReset's.
func TestRateLimitResetAt_0072(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	near, far := now.Add(30*time.Second), now.Add(2*time.Minute)
	claude := http.Header{
		"Anthropic-Ratelimit-Requests-Reset": {near.Format(time.RFC3339)},
		"Anthropic-Ratelimit-Tokens-Reset":   {far.In(time.FixedZone("UTC+2", 2*3600)).Format(time.RFC3339)},
	}
	cases := []struct {
		name   string
		header http.Header
		wait   time.Duration
		at     time.Time
	}{
		{"Claude's latest", claude, 2 * time.Minute, far},
		{"a longer duration header", http.Header{"Anthropic-Ratelimit-Requests-Reset": {near.Format(time.RFC3339)}, "X-Ratelimit-Reset-Tokens": {"5m"}}, 5 * time.Minute, time.Time{}},
		{"a tie", http.Header{"Anthropic-Ratelimit-Requests-Reset": {near.Format(time.RFC3339)}, "X-Ratelimit-Reset-Requests": {"30s"}}, 30 * time.Second, time.Time{}},
		{"durations only", http.Header{"X-Ratelimit-Reset-Requests": {"20s"}}, 20 * time.Second, time.Time{}},
		{"none", http.Header{}, 0, time.Time{}},
	}
	for _, c := range cases {
		wait, at := RateLimitResetAt(c.header, now)
		if wait != c.wait || !at.Equal(c.at) {
			t.Errorf("%s: RateLimitResetAt = %s, %v; want %s, %v", c.name, wait, at, c.wait, c.at)
		}
		if !at.IsZero() && at.Location() != time.UTC {
			t.Errorf("%s: instant in %v; want UTC", c.name, at.Location())
		}
		if got := RateLimitReset(c.header, now); got != c.wait {
			t.Errorf("%s: RateLimitReset = %s; want RateLimitResetAt's %s", c.name, got, c.wait)
		}
	}
}
