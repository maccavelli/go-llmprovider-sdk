package llmprovider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// sentinelError is a sentinel that also matches a broader one, so a new class
// can be added without breaking an existing errors.Is check.
type sentinelError struct {
	msg    string
	parent error
}

func (e *sentinelError) Error() string { return e.msg }
func (e *sentinelError) Unwrap() error { return e.parent }

var (
	// ErrQuotaExhausted marks exhausted quota, credit or balance. Retrying
	// cannot succeed before the quota resets, so the APIError carrying it is
	// terminal. It also matches ErrRateLimited (MADR 0012 §1.1).
	ErrQuotaExhausted error = &sentinelError{msg: "llmprovider: quota exhausted", parent: ErrRateLimited}
	// ErrNotPermitted marks a request the account may not make: a region,
	// data-policy, entitlement or free-tier restriction (MADR 0012 §1.1).
	ErrNotPermitted = errors.New("llmprovider: not permitted")
	// ErrContextOverflow marks input longer than the model's context window.
	// Shortening the input can succeed; retrying it unchanged cannot. It also
	// matches ErrInvalidRequest (0015-MADR D7, amendment of 2026-09-30).
	ErrContextOverflow error = &sentinelError{msg: "llmprovider: context window exceeded", parent: ErrInvalidRequest}
	// ErrIncomplete marks a response the service cut short, or one without
	// the tool call the request forced (0015-MADR D7). Retrying it unchanged
	// cannot help, so it also matches ErrInvalidRequest (MADR 0012 §1.5;
	// 0015-MADR D7, amendment of 2026-10-01).
	ErrIncomplete error = &sentinelError{msg: "llmprovider: incomplete response", parent: ErrInvalidRequest}
	// ErrUnsupported marks a request needing a capability the provider does
	// not have. It fails before any network call, and also matches
	// errors.ErrUnsupported (0015-MADR D4, D7).
	ErrUnsupported error = &sentinelError{msg: "llmprovider: unsupported", parent: errors.ErrUnsupported}
	// ErrInvalidProvider marks a provider id that is unknown, empty or not a
	// safe file name (0015-MADR D7).
	ErrInvalidProvider = errors.New("llmprovider: invalid provider id")
)

const (
	// apiErrorBodyLimit bounds how much of an error body is read.
	apiErrorBodyLimit = 64 << 10
	// apiErrorMessageLimit bounds APIError.Message.
	apiErrorMessageLimit = 512
	// apiErrorRedactLimit bounds how much of a message is redacted. Only the
	// first apiErrorMessageLimit bytes are kept, so redacting a 64 KiB HTML
	// page whole is wasted work (0021-MADR Z1).
	apiErrorRedactLimit = 16 << 10
)

// APIError is a failure the service reported, with its own classification
// (MADR 0012 §1.1, 0015-MADR D7). It is the only structured error: a rate
// limit is one of kind ErrRateLimited with its RetryAfter, and a response the
// service cut short one of kind ErrIncomplete with its Reason. It unwraps to
// its Kind and, when that differs, to the sentinel the status alone mapped to
// before, so every existing errors.Is check still matches (MADR 0012 §7).
type APIError struct {
	// Provider names the provider, or "gateway/route" for a gateway.
	Provider string
	// Status is the HTTP status: 0 for a failure inside a 200 event stream.
	Status int
	// Kind is the kind sentinel, such as ErrRateLimited. Nil means the one
	// Status maps to.
	Kind error
	// Code is the service's error type or code, e.g. "FreeUsageLimitError".
	Code string
	// Message is the service's message, redacted and bounded to 512 bytes.
	Message string
	// RetryAfter is the delay the service asked for, when it asked, in a
	// header or in its error body.
	RetryAfter time.Duration
	// Reason is the service's reason for a response it cut short, such as
	// "max_output_tokens".
	Reason string

	// terminal reports that retrying cannot succeed; Retryable reads it.
	terminal bool
	// shouldRetry reports that the service asked for a retry, with
	// x-should-retry: true; Retryable reads it first (0021-MADR T14).
	shouldRetry bool
}

// Error reads "<kind>: <provider> HTTP <status> <code> (retry-after <d>):
// <reason>: <message>", leaving out what is unset. A failure with no status
// and no reason came from a stream.
func (e *APIError) Error() string {
	var where []string
	if e.Provider != "" {
		where = append(where, e.Provider)
	}
	switch {
	case e.Status != 0:
		where = append(where, fmt.Sprintf("HTTP %d", e.Status))
	case e.Provider != "" && e.Reason == "":
		where = append(where, "stream")
	}
	if e.Code != "" {
		where = append(where, e.Code)
	}
	if e.RetryAfter > 0 {
		where = append(where, fmt.Sprintf("(retry-after %s)", e.RetryAfter))
	}
	parts := []string{e.kind().Error()}
	if len(where) > 0 {
		parts = append(parts, strings.Join(where, " "))
	}
	for _, s := range []string{e.Reason, e.Message} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ": ")
}

// Unwrap returns the kind and the pre-0012 status sentinel. A stream failure
// (Status 0) has no status sentinel, and nor does a Retryable error, such as
// a retryable 408 or 409: its status's sentinel may be ErrInvalidRequest,
// which says a retry can never succeed (0026-MADR F23).
func (e *APIError) Unwrap() []error {
	kind := e.kind()
	if e.Retryable() {
		return []error{kind}
	}
	if legacy := statusSentinel(e.Status); e.Status != 0 && !errors.Is(kind, legacy) {
		return []error{kind, legacy}
	}
	return []error{kind}
}

// Retryable reports whether the same request can succeed later: a rate limit
// other than exhausted quota, or an unavailable service, that the service did
// not mark final (0015-MADR D7).
func (e *APIError) Retryable() bool {
	if e.shouldRetry {
		return true
	}
	if e.terminal {
		return false
	}
	kind := e.kind()
	if errors.Is(kind, ErrQuotaExhausted) {
		return false
	}
	return errors.Is(kind, ErrRateLimited) || errors.Is(kind, ErrProviderUnavailable)
}

// kind is Kind, or the sentinel Status maps to when Kind is nil.
func (e *APIError) kind() error {
	switch {
	case e.Kind != nil:
		return e.Kind
	case e.Status == 0:
		return ErrProviderUnavailable
	default:
		return statusSentinel(e.Status)
	}
}

// statusSentinel is the status-only mapping every provider used before MADR
// 0012.
func statusSentinel(status int) error {
	switch {
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrAuthFailure
	case status >= http.StatusInternalServerError:
		return ErrProviderUnavailable
	default:
		return ErrInvalidRequest
	}
}

// ClassifyHTTPError maps a non-2xx response to a typed error; it returns nil
// for any 2xx (0026-MADR F24). provider names the caller for the message: a
// multi-route gateway passes "gateway/route", so a misroute is diagnosable
// from the error alone. A 429 is an *APIError of kind ErrRateLimited, with
// the delay the service asked for in RetryAfter: from retry-after-ms or
// Retry-After, else from the body, as a usage limit's reset time or Google's
// RetryInfo. Gemini's 429 for a per-day quota is ErrQuotaExhausted
// (0026-MADR F12).
// It is part of the error model, for any provider (0015-MADR, amendment
// "S7b's import graph"). A nil resp is ErrProviderUnavailable.
func ClassifyHTTPError(provider string, resp *http.Response) error {
	if resp == nil {
		return fmt.Errorf("%w: %s: no HTTP response", ErrProviderUnavailable, provider)
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	var body []byte
	var readErr error
	if resp.Body != nil {
		body, readErr = io.ReadAll(io.LimitReader(resp.Body, apiErrorBodyLimit))
	}
	envelope := parseAPIErrorBody(body)
	message := envelope.message()
	// A body that cannot be read is said in the message, not logged
	// (0015-MADR D9).
	if readErr != nil {
		message = strings.TrimSpace(message + " (error body unreadable: " + readErr.Error() + ")")
	}
	e := &APIError{
		Provider:   provider,
		Status:     resp.StatusCode,
		Code:       boundCode(envelope.errType()),
		Message:    boundMessage(redact.String(redact.StripControl(cutMessage(message)))),
		RetryAfter: transport.RetryAfter(resp.Header),
	}
	e.terminal, e.Kind = classifyAPIError(serviceOf(provider), resp.StatusCode, envelope, body)
	// A usage limit says when it resets, as Codex reads it (MADR 0012 §4.4).
	if e.RetryAfter == 0 && envelope.resetsAt > 0 {
		if d := time.Until(time.Unix(envelope.resetsAt, 0)); d > 0 {
			e.RetryAfter = d
		}
	}
	// Gemini sends no Retry-After; its body's RetryInfo says how long to
	// wait (0026-MADR F12).
	if e.RetryAfter == 0 {
		e.RetryAfter = envelope.retryDelay
	}
	// x-should-retry is the service saying whether a retry can succeed, and
	// is obeyed both ways, as the OpenAI and Anthropic SDKs do (MADR 0012
	// §1.2; 0021-MADR T14).
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("X-Should-Retry"))) {
	case "false":
		e.terminal = true
	case "true":
		e.shouldRetry = true
		// ErrInvalidRequest says a retry can never succeed, so a reply
		// the service says to retry is unavailable instead, as a
		// retryable 408 is (0026-MADR F67).
		if errors.Is(e.Kind, ErrInvalidRequest) {
			e.Kind = ErrProviderUnavailable
		}
	}
	return e
}

// ClassifyStreamFailure classifies a failure reported inside a 200 event
// stream, such as the Responses API's response.failed, as Codex's
// parse_failed_response does (codex-api/src/sse/responses_error.rs): quota and
// entitlement codes are terminal (MADR 0012 §1.1), context_length_exceeded is a
// context overflow, invalid_prompt is an invalid request, rate_limit_exceeded
// and slow_down are rate limits, and anything else is retryable. A code that
// is an HTTP status from 400 to 599, as OpenRouter-style gateways send inside
// a 200, is classified as that status (0026-MADR F5). The
// APIError's Status is 0: there is no HTTP status. It is part of the error
// model, for any provider's stream reader (0015-MADR, amendment "S7b's import
// graph").
func ClassifyStreamFailure(provider, code, errType, message string) error {
	env := apiErrorEnvelope{msg: message}
	for _, t := range []string{code, errType} {
		if t != "" && !env.hasType(t) {
			env.types = append(env.types, t)
		}
	}
	e := &APIError{Provider: provider, Code: boundCode(env.errType()),
		Message: boundMessage(redact.String(redact.StripControl(cutMessage(message))))}
	// A gateway that answers 200 with an error, as OpenRouter does, puts the
	// HTTP status in code: classify it as that status, through the service's
	// own table and the overflow check (0026-MADR F5).
	if status, err := strconv.Atoi(code); err == nil && status >= http.StatusBadRequest && status <= 599 {
		e.terminal, e.Kind = classifyAPIError(serviceOf(provider), status, env, nil)
		return e
	}
	switch {
	case env.hasType("rate_limit_exceeded") || env.hasType("slow_down"):
		e.Kind = ErrRateLimited
	case env.hasType("context_length_exceeded"):
		e.terminal, e.Kind = true, ErrContextOverflow
	case env.hasType("invalid_prompt"):
		e.terminal, e.Kind = true, ErrInvalidRequest
	default:
		// 500 stands in for "no status": the table's codes win, else retryable.
		e.terminal, e.Kind = classifyAPIError(serviceOf(provider), http.StatusInternalServerError, env, nil)
	}
	return e
}

// serviceOf reduces a provider label ("opencode-go/messages", "kilo") to the
// service whose error vocabulary applies.
func serviceOf(provider string) string {
	name, _, _ := strings.Cut(provider, "/")
	if strings.HasPrefix(name, serviceOpencode) {
		return serviceOpencode
	}
	return name
}

// serviceOpencode names both OpenCode gateways' shared error vocabulary.
const serviceOpencode = "opencode"

var (
	opencodeQuotaTypes     = []string{"FreeUsageLimitError", "GoUsageLimitError", "BlackUsageLimitError", "CreditsError", "MonthlyLimitError", "UserLimitError"}
	opencodeForbiddenTypes = []string{"RegionError", "DataPolicyError", "FreeTierError"}
	openAIQuotaTypes       = []string{"usage_limit_reached", "insufficient_quota"}
	// geminiAuthReasons are the details reasons Gemini gives a refused key
	// with HTTP 400, as captured live (0020-MADR F23, Q6 a).
	geminiAuthReasons = []string{"API_KEY_INVALID"}
)

// classifyAPIError applies MADR 0012 §1.1's table (with its 2026-09-27 rows)
// and falls back to the status-only mapping.
func classifyAPIError(service string, status int, env apiErrorEnvelope, body []byte) (terminal bool, sentinel error) {
	has := func(types ...string) bool {
		return slices.ContainsFunc(types, env.hasType)
	}
	switch {
	case service == serviceOpencode && has(opencodeQuotaTypes...),
		service == string(ProviderKilo) && (has("PROMOTION_MODEL_LIMIT_REACHED") || bytes.Contains(body, []byte("FreeUsageLimitError"))),
		service == string(ProviderOpenAI) && has(openAIQuotaTypes...),
		status == http.StatusPaymentRequired:
		return true, ErrQuotaExhausted
	// OpenCode refuses a model or a route with 403, typed or not; a bad
	// key is its 401 (0012-MADR, amendment 2026-10-05; 0021-MADR L3).
	case service == serviceOpencode && (status == http.StatusForbidden || has(opencodeForbiddenTypes...)),
		service == string(ProviderOpenAI) && has("usage_not_included"),
		service == string(ProviderKilo) && (status == http.StatusForbidden || has("data_collection_required")):
		return true, ErrNotPermitted
	case service == string(ProviderGemini) && has(geminiAuthReasons...):
		return true, ErrAuthFailure
	// A per-day quota cannot be passed by a retry within the day
	// (0026-MADR F12).
	case service == string(ProviderGemini) && status == http.StatusTooManyRequests && env.perDayQuota:
		return true, ErrQuotaExhausted
	case service == serviceOpencode && has("ModelError"),
		service == string(ProviderKilo) && has("PAID_MODEL_AUTH_REQUIRED"):
		return true, ErrInvalidRequest
	case status == http.StatusTooManyRequests:
		return false, ErrRateLimited
	case status == http.StatusRequestTimeout:
		return false, ErrProviderUnavailable
	// A conflict is a lock timeout to OpenAI and Anthropic, whose SDKs retry
	// it (0021-MADR T14).
	case status == http.StatusConflict && (service == string(ProviderOpenAI) || service == string(ProviderClaude)):
		return false, ErrProviderUnavailable
	case status == 525 || status == 526:
		return true, ErrProviderUnavailable
	case status >= http.StatusInternalServerError:
		return false, ErrProviderUnavailable
	case errors.Is(statusSentinel(status), ErrInvalidRequest) && contextOverflow(env):
		return true, ErrContextOverflow
	default:
		return true, statusSentinel(status)
	}
}

// apiErrorEnvelope is the union of the error bodies MADR 0012 §1.1 lists:
// OpenCode and Claude {type:"error",error:{type,message}}, Kilo
// {error:{code,message}}, {code} or {error,error_type,message},
// OpenAI/Codex {error:{type,code,message}},
// xAI nested or flat {code,error}, Gemini {error:{code,message,status,
// details:[{reason}]}}.
type apiErrorEnvelope struct {
	types    []string // candidate classifications, most specific first
	msg      string
	resetsAt int64 // usage_limit_reached's reset time, Unix seconds, or 0
	// retryDelay is a google.rpc.RetryInfo's delay, or 0 (0026-MADR F12).
	retryDelay time.Duration
	// perDayQuota reports a google.rpc.QuotaFailure naming a per-day quota.
	perDayQuota bool
}

// protoDuration is a protobuf Duration in its JSON form, seconds with an "s"
// suffix such as "39s" or "1.5s", or 0 when s is not a positive one. A delay
// too long for a Duration is the longest one.
func protoDuration(s string) time.Duration {
	digits, ok := strings.CutSuffix(s, "s")
	secs, err := strconv.ParseFloat(digits, 64)
	switch {
	case !ok || err != nil || math.IsNaN(secs) || secs <= 0:
		return 0
	case secs >= float64(math.MaxInt64)/float64(time.Second):
		return math.MaxInt64
	default:
		return time.Duration(secs * float64(time.Second))
	}
}

func (e apiErrorEnvelope) errType() string {
	if len(e.types) == 0 {
		return ""
	}
	return e.types[0]
}

func (e apiErrorEnvelope) message() string { return e.msg }

func (e apiErrorEnvelope) hasType(t string) bool { return slices.Contains(e.types, t) }

func parseAPIErrorBody(body []byte) apiErrorEnvelope {
	var top struct {
		Type      string          `json:"type"`
		Code      json.RawMessage `json:"code"`
		ErrorType string          `json:"error_type"`
		Message   string          `json:"message"`
		Error     json.RawMessage `json:"error"`
		Detail    json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(body, &top) != nil {
		return apiErrorEnvelope{msg: strings.TrimSpace(string(body))}
	}
	var env apiErrorEnvelope
	add := func(values ...string) {
		for _, v := range values {
			if v != "" && v != "error" && !env.hasType(v) {
				env.types = append(env.types, v)
			}
		}
	}
	var inner struct {
		Type     string          `json:"type"`
		Code     json.RawMessage `json:"code"`
		Status   string          `json:"status"`
		Message  string          `json:"message"`
		ResetsAt int64           `json:"resets_at"`
		// Details are Google's: a reason is the most specific type; a
		// RetryInfo carries the delay and a QuotaFailure the violated
		// quotas (google/rpc/error_details.proto; 0026-MADR F12).
		Details []struct {
			Type       string `json:"@type"`
			Reason     string `json:"reason"`
			RetryDelay string `json:"retryDelay"`
			Violations []struct {
				QuotaID string `json:"quotaId"`
			} `json:"violations"`
		} `json:"details"`
	}
	var text string
	switch {
	case json.Unmarshal(top.Error, &inner) == nil:
		for _, d := range inner.Details {
			add(d.Reason)
			if strings.HasSuffix(d.Type, "google.rpc.RetryInfo") {
				env.retryDelay = protoDuration(d.RetryDelay)
			}
			for _, v := range d.Violations {
				env.perDayQuota = env.perDayQuota || strings.Contains(v.QuotaID, "PerDay")
			}
		}
		add(jsonString(inner.Code), inner.Type, inner.Status)
		env.msg = inner.Message
		env.resetsAt = inner.ResetsAt
	case json.Unmarshal(top.Error, &text) == nil:
		env.msg = text
	}
	add(top.ErrorType, jsonString(top.Code), top.Type)
	if env.msg == "" {
		env.msg = top.Message
	}
	if env.msg == "" {
		// The ChatGPT backend's {"detail": ...} (gate G-C, 2026-09-27).
		env.msg = jsonString(top.Detail)
	}
	return env
}

// jsonString returns a JSON string's value; numbers and null give "".
func jsonString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

// boundCode strips a service's error code of control characters and trims
// it to redact.FieldLimit bytes on a rune boundary: the code reaches
// Error() and so a terminal (0021-MADR Z3).
func boundCode(s string) string { return redact.Field(s) }

// cutMessage trims s to apiErrorRedactLimit bytes on a rune boundary, before
// it is redacted.
func cutMessage(s string) string {
	if len(s) <= apiErrorRedactLimit {
		return s
	}
	cut := apiErrorRedactLimit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// boundMessage trims s to apiErrorMessageLimit bytes on a rune boundary.
func boundMessage(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= apiErrorMessageLimit {
		return s
	}
	cut := apiErrorMessageLimit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}
