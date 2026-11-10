package llmprovider

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
)

// Limit is what a service stated about the limit a request hit, read from
// its error (0072-MADR D2). Every field is the service's; none is inferred,
// and a window is never inferred from how long the wait is. A field the
// service did not state is zero. Each string is stripped of control
// characters and bounded, as APIError.Code is. It is safe for concurrent use
// only while no goroutine changes it.
type Limit struct {
	// Name is the service's own name for the limit, such as OpenCode Go's
	// metadata.limitName: "5 hour", "weekly" or "monthly".
	Name string
	// Window is the limit's length, when the service stated it as a number:
	// ChatGPT's limit_window_minutes, or a Name of the form "<n> hour" or
	// "<n> hours". A length outside 1 minute to 366 days is not kept. A
	// Name such as "weekly" gives no Window.
	Window time.Duration
	// Plan is the account's plan, as ChatGPT's plan_type names it, such as
	// "plus".
	Plan string
	// PerDay reports a Gemini QuotaFailure whose quota is counted per day,
	// which is what makes its 429 ErrQuotaExhausted (0026-MADR F12).
	PerDay bool
	// Quotas are the quotas a Gemini QuotaFailure names, in its order.
	Quotas []Quota
	// CreditsURL is where the service says to add credits: Kilo's
	// error.metadata.buyCreditsUrl. It is kept only as an absolute https URL
	// with no user information, query or fragment, which redaction leaves
	// unchanged; otherwise it is empty.
	CreditsURL string
}

// Quota is one violated quota of a Gemini QuotaFailure (0072-MADR D2). It is
// safe for concurrent use only while no goroutine changes it.
type Quota struct {
	// ID is its quotaId, such as
	// "GenerateRequestsPerDayPerProjectPerModel-FreeTier".
	ID string
	// Metric is its quotaMetric.
	Metric string
	// Value is its quotaValue, the limit, as the text the service sent: a
	// JSON string's value, or a JSON number written in decimal.
	Value string
}

// Bounds on a stated window (0072-MADR D2).
const (
	minLimitWindow = time.Minute
	maxLimitWindow = 366 * 24 * time.Hour
)

// hoursName is a limit name that states its length in hours, as OpenCode
// Go's "5 hour" does (0072-MADR O1).
var hoursName = regexp.MustCompile(`^(\d+) hours?$`)

// errorMetadata is an error's metadata object: OpenCode's beside error or
// inside it, Kilo's inside error, or a Google ErrorInfo's. Each member is
// kept raw, so that one of an unexpected type spoils no other.
type errorMetadata struct {
	ResetsAt      json.RawMessage `json:"resetsAt"`
	LimitName     json.RawMessage `json:"limitName"`
	BuyCreditsURL json.RawMessage `json:"buyCreditsUrl"`
}

// readMetadata decodes raw as an error's metadata; an absent or malformed
// object gives a zero one.
func readMetadata(raw json.RawMessage) errorMetadata {
	var m errorMetadata
	if len(raw) == 0 || json.Unmarshal(raw, &m) != nil {
		return errorMetadata{}
	}
	return m
}

// windowMinutes is a JSON integer count of minutes as a window, or 0 when
// raw is not one or is outside 1 minute to 366 days (0072-MADR X5). The
// bound is checked on the count, before it can overflow a Duration.
func windowMinutes(raw json.RawMessage) time.Duration {
	n, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || n < int64(minLimitWindow/time.Minute) || n > int64(maxLimitWindow/time.Minute) {
		return 0
	}
	return time.Duration(n) * time.Minute
}

// windowOfName is the length a limit name states in hours, or 0 when it
// states none or one outside 1 hour to 366 days (0072-MADR O1).
func windowOfName(name string) time.Duration {
	m := hoursName.FindStringSubmatch(name)
	if m == nil {
		return 0
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || n < 1 || n > int64(maxLimitWindow/time.Hour) {
		return 0
	}
	return time.Duration(n) * time.Hour
}

// scalarText is a JSON scalar's text: a string's value, or a number's
// decimal text; anything else gives "". It reads a quotaValue, and a stream
// error's header value.
func scalarText(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if strings.HasPrefix(text, `"`) {
		return jsonString(raw)
	}
	if _, err := strconv.ParseFloat(text, 64); err != nil {
		return ""
	}
	return text
}

// creditsURL is s when it is an absolute https URL with a host and no user
// information, query or fragment, which redaction and the field bound leave
// unchanged; otherwise "" (0072-MADR D2, Q6).
func creditsURL(s string) string {
	if s == "" || redact.Field(s) != s || redact.String(s) != s {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.Opaque != "" {
		return ""
	}
	return s
}

// limitOf is what env states about its limit, or nil when it states nothing
// (0072-MADR D2).
func limitOf(env apiErrorEnvelope) *Limit {
	l := &Limit{
		Name:       redact.Field(env.limitName),
		Window:     env.window,
		Plan:       redact.Field(env.plan),
		PerDay:     env.perDayQuota,
		CreditsURL: creditsURL(env.creditsURL),
	}
	if l.Window == 0 {
		l.Window = windowOfName(env.limitName)
	}
	for _, q := range env.quotas {
		l.Quotas = append(l.Quotas, Quota{ID: redact.Field(q.ID), Metric: redact.Field(q.Metric), Value: redact.Field(q.Value)})
	}
	if l.Name == "" && l.Window == 0 && l.Plan == "" && !l.PerDay && len(l.Quotas) == 0 && l.CreditsURL == "" {
		return nil
	}
	return l
}
