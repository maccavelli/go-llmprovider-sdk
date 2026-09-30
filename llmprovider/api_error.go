package llmprovider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/maccavelli/go-llmprovider-sdk/internal/redact"
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
	ErrQuotaExhausted error = &sentinelError{msg: "llm: quota exhausted", parent: ErrRateLimited}
	// ErrNotPermitted marks a request the account may not make: a region,
	// data-policy, entitlement or free-tier restriction (MADR 0012 §1.1).
	ErrNotPermitted = errors.New("llm: not permitted")
	// ErrContextOverflow marks input longer than the model's context window.
	// Shortening the input can succeed; retrying it unchanged cannot. It also
	// matches ErrInvalidRequest (0015-MADR D7, amendment of 2026-09-30).
	ErrContextOverflow error = &sentinelError{msg: "llmprovider: context window exceeded", parent: ErrInvalidRequest}
	// ErrIncomplete marks a response the service cut short, or one without
	// the tool call the request forced (0015-MADR D7).
	ErrIncomplete = errors.New("llmprovider: incomplete response")
	// ErrUnsupported marks a request needing a capability the provider does
	// not have. It fails before any network call, and also matches
	// errors.ErrUnsupported (0015-MADR D4, D7).
	ErrUnsupported error = &sentinelError{msg: "llmprovider: unsupported", parent: errors.ErrUnsupported}
)

const (
	// apiErrorBodyLimit bounds how much of an error body is read.
	apiErrorBodyLimit = 64 << 10
	// apiErrorMessageLimit bounds APIError.Message.
	apiErrorMessageLimit = 512
)

// APIError is a failure the service reported, with its own classification
// (MADR 0012 §1.1, 0015-MADR D7). It unwraps to its Kind and, when that
// differs, to the sentinel the status alone mapped to before, so every
// existing errors.Is check still matches (MADR 0012 §7).
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
	// RetryAfter is the delay the service asked for, when it asked.
	RetryAfter time.Duration
	// Reason is the service's reason for a response it cut short, such as
	// "max_output_tokens".
	Reason string

	// Type is Code.
	//
	// Deprecated: Use Code. Type is removed with the old API (0015-PLAN S8).
	Type string
	// Terminal reports that retrying cannot succeed.
	//
	// Deprecated: Use Retryable. Terminal is removed with the old API
	// (0015-PLAN S8).
	Terminal bool
}

func (e *APIError) Error() string {
	var b strings.Builder
	if e.Status == 0 {
		fmt.Fprintf(&b, "%v: %s stream", e.kind(), e.Provider)
	} else {
		fmt.Fprintf(&b, "%v: %s HTTP %d", e.kind(), e.Provider, e.Status)
	}
	if code := e.code(); code != "" {
		fmt.Fprintf(&b, " %s", code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	return b.String()
}

// Unwrap returns the kind and the pre-0012 status sentinel. A stream failure
// (Status 0) has no status sentinel.
func (e *APIError) Unwrap() []error {
	kind := e.kind()
	if legacy := statusSentinel(e.Status); e.Status != 0 && !errors.Is(kind, legacy) {
		return []error{kind, legacy}
	}
	return []error{kind}
}

// Retryable reports whether the same request can succeed later: a rate limit
// other than exhausted quota, or an unavailable service, that the service did
// not mark final (0015-MADR D7).
func (e *APIError) Retryable() bool {
	if e.Terminal {
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

// code is Code, or the deprecated Type a caller may still set.
func (e *APIError) code() string {
	if e.Code != "" {
		return e.Code
	}
	return e.Type
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

// classifyHTTPError maps a non-200 response to a typed error; it returns nil
// for 200. provider names the caller for the message: a multi-route gateway
// passes "gateway/route", so a misroute is diagnosable from the error alone.
// A plain 429 stays a *RateLimitError; every other status is an *APIError.
func classifyHTTPError(provider string, resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	var body []byte
	if resp.Body != nil {
		var readErr error
		if body, readErr = io.ReadAll(io.LimitReader(resp.Body, apiErrorBodyLimit)); readErr != nil {
			slog.Debug("llmprovider: read error body", "provider", provider, "error", readErr)
		}
	}
	envelope := parseAPIErrorBody(body)
	e := &APIError{
		Provider:   provider,
		Status:     resp.StatusCode,
		Code:       envelope.errType(),
		Type:       envelope.errType(),
		Message:    boundMessage(redact.String(envelope.message())),
		RetryAfter: retryAfterFrom(resp.Header),
	}
	e.Terminal, e.Kind = classifyAPIError(serviceOf(provider), resp.StatusCode, envelope, body)
	// A usage limit says when it resets, as Codex reads it (MADR 0012 §4.4).
	if e.RetryAfter == 0 && envelope.resetsAt > 0 {
		if d := time.Until(time.Unix(envelope.resetsAt, 0)); d > 0 {
			e.RetryAfter = d
		}
	}
	// x-should-retry: false is the service saying no retry can succeed (MADR 0012 §1.2).
	if strings.EqualFold(strings.TrimSpace(resp.Header.Get("X-Should-Retry")), "false") {
		e.Terminal = true
	}
	if errors.Is(e.Kind, ErrRateLimited) && !e.Terminal {
		return &RateLimitError{RetryAfter: e.RetryAfter, Status: e.Status, Provider: provider, Message: e.Message}
	}
	return e
}

// streamFailure classifies a response.failed event, which arrives inside a 200
// stream, as Codex's parse_failed_response does
// (codex-api/src/sse/responses_error.rs): quota and entitlement codes are
// terminal (MADR 0012 §1.1), context_length_exceeded is a context overflow,
// invalid_prompt is an invalid request, rate_limit_exceeded and slow_down are
// rate limits, and anything else is retryable. The APIError's Status is 0:
// there is no HTTP status.
func streamFailure(provider, code, errType, message string) error {
	env := apiErrorEnvelope{msg: message}
	for _, t := range []string{code, errType} {
		if t != "" && !env.hasType(t) {
			env.types = append(env.types, t)
		}
	}
	e := &APIError{Provider: provider, Code: env.errType(), Type: env.errType(), Message: boundMessage(redact.String(message))}
	switch {
	case env.hasType("rate_limit_exceeded") || env.hasType("slow_down"):
		return &RateLimitError{Provider: provider, Message: e.Message}
	case env.hasType("context_length_exceeded"):
		e.Terminal, e.Kind = true, ErrContextOverflow
	case env.hasType("invalid_prompt"):
		e.Terminal, e.Kind = true, ErrInvalidRequest
	default:
		// 500 stands in for "no status": the table's codes win, else retryable.
		e.Terminal, e.Kind = classifyAPIError(serviceOf(provider), http.StatusInternalServerError, env, nil)
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
)

// classifyAPIError applies MADR 0012 §1.1's table (with its 2026-09-27 rows)
// and falls back to the status-only mapping.
func classifyAPIError(service string, status int, env apiErrorEnvelope, body []byte) (terminal bool, sentinel error) {
	has := func(types ...string) bool {
		return slices.ContainsFunc(types, env.hasType)
	}
	switch {
	case service == serviceOpencode && has(opencodeQuotaTypes...),
		service == ProviderKilo && (has("PROMOTION_MODEL_LIMIT_REACHED") || bytes.Contains(body, []byte("FreeUsageLimitError"))),
		service == ProviderOpenAI && has(openAIQuotaTypes...),
		status == http.StatusPaymentRequired:
		return true, ErrQuotaExhausted
	case service == serviceOpencode && has(opencodeForbiddenTypes...),
		service == ProviderOpenAI && has("usage_not_included"),
		service == ProviderKilo && (status == http.StatusForbidden || has("data_collection_required")):
		return true, ErrNotPermitted
	case service == serviceOpencode && has("ModelError"),
		service == ProviderKilo && has("PAID_MODEL_AUTH_REQUIRED"):
		return true, ErrInvalidRequest
	case status == http.StatusTooManyRequests:
		return false, ErrRateLimited
	case status == http.StatusRequestTimeout:
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
// xAI nested or flat {code,error}, Gemini {error:{code,message,status}}.
type apiErrorEnvelope struct {
	types    []string // candidate classifications, most specific first
	msg      string
	resetsAt int64 // usage_limit_reached's reset time, Unix seconds, or 0
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
	}
	var text string
	switch {
	case json.Unmarshal(top.Error, &inner) == nil:
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
