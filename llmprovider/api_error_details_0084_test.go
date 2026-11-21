package llmprovider

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// manyDetails0084 is a Google-shaped error body of about total bytes whose
// "details" holds one entry for each of item(0), item(1), ...: the shape the
// review of 0084-PLAN Phase 4 measured (B1).
func manyDetails0084(total int, item func(i int) string) string {
	var b strings.Builder
	b.WriteString(`{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"quota","details":[`)
	for i := 0; b.Len() < total-200; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(item(i))
	}
	b.WriteString(`]}}`)
	return b.String()
}

// bestOf0084 is the shortest of five classifications of body, and the last
// error, so that a slow moment of the host does not fail the ratio.
func bestOf0084(body string) (time.Duration, *APIError) {
	var best time.Duration
	var last *APIError
	for i := range 5 {
		start := time.Now()
		err := ClassifyHTTPError("gemini", &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(body))})
		if d := time.Since(start); i == 0 || d < best {
			best = d
		}
		last, _ = err.(*APIError) //nolint:errorlint // the error ClassifyHTTPError returns is the *APIError itself
	}
	return best, last
}

// TestClassify_ManyDetails_0084 (MADR ST87; the owner's ruling "Set-based
// check + cap the lists (Recommended)"): at the 1 MiB read, a body of many
// distinct details costs time in proportion to its size, not its square, and
// leaves a bounded number of entries on the error. A body of distinct reasons
// costs time in proportion to its size: the best time at about 1 MiB over the
// best time at 64 KiB, measured in the same run, is under 64 (about 16 as
// built; the code that scanned the list it built gave 140 to 250). A ratio is
// used and not an absolute bound because the race detector slows the parse
// about twentyfold, which an absolute bound cannot allow for without losing
// its headroom. The entries kept are counted apart, which is what catches a cap
// removed: a 1 MiB body of distinct reasons keeps at most 32 among the types;
// one of distinct quota violations keeps at most 32 on Limit (it kept 8,134),
// and at most 32 in all, not 32 for each detail; an error with a few details
// keeps them all.
func TestClassify_ManyDetails_0084(t *testing.T) {
	const size = 1000 << 10
	reasons := func(i int) string { return fmt.Sprintf(`{"@type":"x","reason":"R%d"}`, i) }
	violations := func(i int) string {
		return fmt.Sprintf(`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaId":"Q%d","quotaMetric":"m","quotaValue":"1"}]}`, i)
	}

	body := manyDetails0084(size, reasons)
	small, _ := bestOf0084(manyDetails0084(64<<10, reasons))
	large, _ := bestOf0084(body)
	if ratio := float64(large) / float64(max(small, time.Microsecond)); ratio >= 64 {
		t.Errorf("a %d-byte body of distinct reasons took %v, and one of 64 KiB %v: a ratio of %.0f; want under 64 (the cost grows in proportion to the size, 16 times)",
			len(body), large, small, ratio)
	}
	if env := parseAPIErrorBody([]byte(body)); len(env.types) > 32+1 {
		t.Errorf("a body of distinct reasons kept %d types; want at most 32 reasons and the status", len(env.types))
	}

	body = manyDetails0084(size, violations)
	_, apiErr := bestOf0084(body)
	if apiErr == nil || apiErr.Limit == nil || len(apiErr.Limit.Quotas) > 32 {
		t.Errorf("a body of distinct violations kept %v quotas; want at most 32", apiErr)
	}
	if env := parseAPIErrorBody([]byte(body)); len(env.quotas) > 32 {
		t.Errorf("a body of distinct violations kept %d quotas in the envelope; want at most 32", len(env.quotas))
	}

	// The quota cap holds for one detail that carries all the violations, too:
	// the entry cap does not bound it, the violation cap does.
	var one strings.Builder
	one.WriteString(`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[`)
	for i := 0; one.Len() < size-400; i++ {
		if i > 0 {
			one.WriteString(",")
		}
		fmt.Fprintf(&one, `{"quotaId":"Q%d","quotaMetric":"m","quotaValue":"1"}`, i)
	}
	one.WriteString(`]}`)
	body = manyDetails0084(0, nil)
	body = strings.Replace(body, `"details":[`, `"details":[`+one.String(), 1)
	if env := parseAPIErrorBody([]byte(body)); len(env.quotas) > 32 {
		t.Errorf("one detail with many violations kept %d quotas in the envelope; want at most 32", len(env.quotas))
	}

	// The quota cap is in all, not for each detail (the re-check of Phase 4,
	// E1): two details of 40 violations each keep at most 32 in all.
	detail := func(prefix string) string {
		var v []string
		for i := range 40 {
			v = append(v, fmt.Sprintf(`{"quotaId":"%s%d","quotaMetric":"m","quotaValue":"1"}`, prefix, i))
		}
		return `{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[` + strings.Join(v, ",") + `]}`
	}
	two := strings.Replace(manyDetails0084(0, nil), `"details":[`, `"details":[`+detail("A")+","+detail("B"), 1)
	if env := parseAPIErrorBody([]byte(two)); len(env.quotas) > 32 {
		t.Errorf("two details of 40 violations kept %d quotas; want at most 32 in all", len(env.quotas))
	}

	// The caps are not zero: an error with a few details keeps each.
	few := manyDetails0084(0, nil)
	few = strings.Replace(few, `"details":[`, `"details":[`+reasons(0)+","+reasons(1)+","+reasons(2)+","+violations(0)+","+violations(1)+","+violations(2), 1)
	env := parseAPIErrorBody([]byte(few))
	if !slices.Contains(env.types, "R0") || !slices.Contains(env.types, "R2") || len(env.quotas) != 3 {
		t.Errorf("an error with three reasons and three violations kept types %v and %d quotas; want all of them", env.types, len(env.quotas))
	}
}
