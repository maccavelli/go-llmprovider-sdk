package llmprovider_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// reporter is what checkGemini429 reports to; *testing.T is one.
type reporter interface {
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
}

// checkGemini429 checks a Gemini 429, as the Interactions API sent it, and
// the error Generate returned for it (0027-MADR, amendment of 2026-10-07;
// 0026-MADR F12, D12):
//   - DRIFT: the body is one of the two 429 shapes known, bare or in a
//     one-element array: Google's documented one, status RESOURCE_EXHAUSTED
//     with a google.rpc.RetryInfo retryDelay, or the Interactions API's own,
//     code "too_many_requests"; and the 429 gives a delay, in a Retry-After
//     header of seconds or a retryDelay;
//   - err is an *APIError whose RetryAfter is that delay, the header's when
//     there is one;
//   - err is ErrQuotaExhausted, terminal, exactly when a QuotaFailure
//     violation names a per-day quota, and a retryable ErrRateLimited
//     otherwise.
func checkGemini429(r reporter, header http.Header, body []byte, err error) {
	envelope, inner := "bare", body
	var elems []json.RawMessage
	if json.Unmarshal(body, &elems) == nil && len(elems) == 1 && bytes.HasPrefix(elems[0], []byte("{")) {
		envelope, inner = "in a one-element array", elems[0]
	}
	var shape struct {
		Error struct {
			Status  string          `json:"status"`
			Code    json.RawMessage `json:"code"`
			Details []struct {
				Type       string `json:"@type"`
				RetryDelay string `json:"retryDelay"`
				Violations []struct {
					QuotaID string `json:"quotaId"`
				} `json:"violations"`
			} `json:"details"`
		} `json:"error"`
	}
	if jsonErr := json.Unmarshal(inner, &shape); jsonErr != nil {
		r.Errorf("DRIFT: the 429 body is not a Gemini error: %v", jsonErr)
		return
	}
	var retryDelay string
	perDay := false
	for _, d := range shape.Error.Details {
		if strings.HasSuffix(d.Type, "google.rpc.RetryInfo") {
			retryDelay = d.RetryDelay
		}
		for _, v := range d.Violations {
			perDay = perDay || strings.Contains(v.QuotaID, "PerDay")
		}
	}
	var code string
	_ = json.Unmarshal(shape.Error.Code, &code)
	switch {
	case shape.Error.Status == "RESOURCE_EXHAUSTED" && retryDelay != "":
		r.Logf("the 429 is Google's documented google.rpc shape, %s", envelope)
	case code == "too_many_requests":
		r.Logf("the 429 is the Interactions API's too_many_requests shape, %s", envelope)
	default:
		r.Errorf("DRIFT: a 429 of neither known shape (%s): error.status %q, error.code %s",
			envelope, shape.Error.Status, shape.Error.Code)
		return
	}

	var want time.Duration
	var from string
	switch retryAfter := header.Get("Retry-After"); {
	case retryAfter != "":
		secs, convErr := strconv.Atoi(retryAfter)
		if convErr != nil || secs < 0 {
			r.Errorf("DRIFT: Retry-After %q is not a count of seconds", retryAfter)
			return
		}
		want, from = time.Duration(secs)*time.Second, "the Retry-After header"
	case retryDelay != "":
		d, parseErr := time.ParseDuration(retryDelay)
		if parseErr != nil {
			r.Errorf("DRIFT: retryDelay %q is not a duration: %v", retryDelay, parseErr)
			return
		}
		want, from = d, "the body's retryDelay"
	default:
		r.Errorf("DRIFT: the 429 gives no delay: no Retry-After header and no retryDelay")
		return
	}

	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok {
		r.Errorf("Generate's error = %v; want an *APIError", err)
		return
	}
	if apiErr.RetryAfter != want {
		r.Errorf("RetryAfter = %s; want %s, from %s", apiErr.RetryAfter, want, from)
	}
	if quota := errors.Is(err, llmprovider.ErrQuotaExhausted); quota != perDay {
		r.Errorf("a per-day quota = %t, but ErrQuotaExhausted = %t: %v", perDay, quota, err)
	}
	if !perDay && (!errors.Is(err, llmprovider.ErrRateLimited) || !apiErr.Retryable()) {
		r.Errorf("a per-minute 429 = %v; want a retryable ErrRateLimited", err)
	}
}

// recordingReporter keeps what checkGemini429 reports.
type recordingReporter struct{ errs, logs []string }

func (r *recordingReporter) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

func (r *recordingReporter) Logf(format string, args ...any) {
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

// documentedGemini429 is a Gemini 429 in Google's documented shape, as
// api_error_gemini_0026_test.go builds it in package llmprovider (0026-MADR,
// amendment "Q6 is (b)"), wrapped in the Interactions API's array (D12).
func documentedGemini429(quotaID, retryDelay string) string {
	return `[{"error":{"code":429,"message":"You exceeded your current quota.","status":"RESOURCE_EXHAUSTED","details":[` +
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{` +
		`"quotaMetric":"generativelanguage.googleapis.com/generate_content_free_tier_requests",` +
		`"quotaId":"` + quotaID + `","quotaDimensions":{"location":"global","model":"gemini-pro-latest"},"quotaValue":"5"}]},` +
		`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"` + retryDelay + `"}]}}]`
}

// capturedGemini429 is the Interactions API's 429 as captured live, through
// Generate, with its Retry-After header (0027-PLAN D1).
const capturedGemini429 = `{"error":{"message":"Rate limit exceeded for model gemini-3.1-pro (limit: 25 requests per minute on Tier 1). ` +
	`Please retry in 11s or upgrade your tier at https://ai.dev/rate-limit.","code":"too_many_requests"}}`

func classify429(service, body string, header http.Header) error {
	return llmprovider.ClassifyHTTPError(service, &http.Response{StatusCode: http.StatusTooManyRequests,
		Status: "429 Too Many Requests", Header: header, Body: io.NopCloser(strings.NewReader(body))})
}

// TestCheckGemini429 (0027-PLAN Phase 2, D1): the live 429 test's checks
// pass on both 429 shapes known, with the error the SDK classifies each as:
// Google's documented one, in the Interactions API's array, and the one
// captured live, bare with a Retry-After header. They fail on an error that
// ignores the delay or the quota, on another shape, and on a 429 with no
// delay.
func TestCheckGemini429(t *testing.T) {
	retryAfter11 := http.Header{"Retry-After": {"11"}}
	for _, c := range []struct {
		name, body, wantLog string
		header              http.Header
	}{
		{"documented, per minute", documentedGemini429("GenerateRequestsPerMinutePerProjectPerModel-FreeTier", "39s"),
			"Google's documented google.rpc shape, in a one-element array", http.Header{}},
		{"documented, per day", documentedGemini429("GenerateRequestsPerDayPerProjectPerModel-FreeTier", "52s"),
			"Google's documented google.rpc shape, in a one-element array", http.Header{}},
		{"captured", capturedGemini429, "the Interactions API's too_many_requests shape, bare", retryAfter11},
	} {
		r := &recordingReporter{}
		checkGemini429(r, c.header, []byte(c.body), classify429("gemini", c.body, c.header))
		if len(r.errs) > 0 {
			t.Errorf("%s: the checks failed on the SDK's own classification:\n\t%s", c.name, strings.Join(r.errs, "\n\t"))
		}
		if len(r.logs) != 1 || !strings.Contains(r.logs[0], c.wantLog) {
			t.Errorf("%s: logged %q; want %q", c.name, r.logs, c.wantLog)
		}
	}

	day := documentedGemini429("GenerateRequestsPerDayPerProjectPerModel-FreeTier", "52s")
	for _, c := range []struct {
		name, body string
		header     http.Header
		err        error
		want       []string
	}{
		{"documented per day, classified without its body", day, http.Header{}, classify429("gemini", `{}`, http.Header{}),
			[]string{"RetryAfter = 0s", "ErrQuotaExhausted = false"}},
		{"captured, classified without its header", capturedGemini429, retryAfter11, classify429("gemini", `{}`, http.Header{}),
			[]string{"RetryAfter = 0s; want 11s"}},
		{"another shape", `{"error":{"code":429}}`, retryAfter11, classify429("gemini", `{"error":{"code":429}}`, retryAfter11),
			[]string{"DRIFT: a 429 of neither known shape"}},
		{"no delay", capturedGemini429, http.Header{}, classify429("gemini", capturedGemini429, http.Header{}),
			[]string{"DRIFT: the 429 gives no delay"}},
	} {
		r := &recordingReporter{}
		checkGemini429(r, c.header, []byte(c.body), c.err)
		got := strings.Join(r.errs, "\n")
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: failures %q; want one containing %q", c.name, r.errs, w)
			}
		}
	}
}
