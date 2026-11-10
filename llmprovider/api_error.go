package llmprovider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
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
	// ErrWorkspaceRequired marks Anthropic's refusal of a request that names
	// no workspace from a key not scoped to one. Naming a workspace the key
	// may use can succeed; retrying unchanged cannot. It also matches
	// ErrInvalidRequest (0064-MADR D3).
	ErrWorkspaceRequired error = &sentinelError{msg: "llmprovider: workspace required", parent: ErrInvalidRequest}
	// ErrWorkspaceNotFound marks Anthropic's refusal of the workspace a
	// request named: not a valid workspace ID, or one that does not exist or
	// the key may not use. It also matches ErrInvalidRequest (0064-MADR D3).
	ErrWorkspaceNotFound error = &sentinelError{msg: "llmprovider: workspace not found", parent: ErrInvalidRequest}
	// ErrIncomplete marks a response the service cut short, or one without
	// the tool call the request forced (0015-MADR D7). Retrying it unchanged
	// cannot help, so it also matches ErrInvalidRequest (MADR 0012 §1.5;
	// 0015-MADR D7, amendment of 2026-10-01).
	ErrIncomplete error = &sentinelError{msg: "llmprovider: incomplete response", parent: ErrInvalidRequest}
	// ErrEmptyAnswer marks an answer with nothing usable in it. It also
	// matches ErrIncomplete, and so ErrInvalidRequest: retrying it unchanged
	// cannot help. A refusal or a safety stop is this kind too, and so is an
	// answer cut while the model was still reasoning; the APIError's Reason,
	// the service's finish reason (content_filter, length, …), tells them
	// apart (0061-MADR D1).
	ErrEmptyAnswer error = &sentinelError{msg: "llmprovider: empty answer", parent: ErrIncomplete}
	// ErrOffline marks a request that could not be sent for want of a
	// network: the host's name did not resolve, or the dial timed out or
	// found the port refused, the network down or unreachable, or the host
	// unreachable. It also matches ErrProviderUnavailable, and is retried. A
	// dial the caller's context ended keeps the context's error; a TLS
	// failure and any HTTP response, a 5xx included, are not offline
	// (0061-MADR D2).
	ErrOffline error = &sentinelError{msg: "llmprovider: offline", parent: ErrProviderUnavailable}
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
	// page whole is wasted work (0021-MADR Z1); four times the kept bytes
	// leaves room for redaction to shorten what comes first (0028-MADR D-A10).
	apiErrorRedactLimit = 2 << 10
)

// APIError is a failure the service reported, with its own classification
// (MADR 0012 §1.1, 0015-MADR D7). It is the only structured error: a rate
// limit is one of kind ErrRateLimited with its RetryAfter, the instant the
// service named for it to lift in ResetAt, and what the service stated
// about it in Limit (0072-MADR D1, D2); a response the service cut short is
// one of kind ErrIncomplete with its Reason. It unwraps to
// its Kind and, when that differs, to the sentinel the status alone mapped to
// before, so every existing errors.Is check still matches (MADR 0012 §7). It
// is safe for concurrent use only while no goroutine changes it.
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
	// ResetAt is the instant the service named for its limit to lift, in
	// UTC: a usage limit's reset time in its error body (ChatGPT's
	// resets_at, OpenCode's metadata.resetsAt), else an HTTP-date
	// Retry-After, else, on a 429, the rate-limit reset header naming a
	// time that gave RetryAfter. It is zero when the service named only a
	// delay, or nothing. It is kept as stated, so it may have passed, by the
	// service's clock or a late reply. Error does not print it (0072-MADR
	// D1).
	ResetAt time.Time
	// Limit is what the service stated about the limit the request hit,
	// such as its window, the account's plan, a Gemini quota or a link to
	// add credits; nil when it stated nothing beyond its code. Error does
	// not print it (0072-MADR D2).
	Limit *Limit
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
// (0026-MADR F12). ResetAt is the body's reset time (resets_at,
// metadata.resetsAt), else an HTTP-date Retry-After, else, on a 429, the
// rate-limit reset header naming a time that gave RetryAfter; Limit is what
// the body states about the limit, or nil (0072-MADR D1, D2).
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
	now := time.Now()
	retryAfter, resetAt := transport.RetryAfterAt(resp.Header, now)
	e := &APIError{
		Provider:   provider,
		Status:     resp.StatusCode,
		Code:       boundCode(envelope.errType()),
		Message:    boundMessage(redact.String(redact.StripControl(redactInput(message)))),
		RetryAfter: retryAfter,
		ResetAt:    resetAt,
		Limit:      limitOf(envelope),
	}
	e.terminal, e.Kind = classifyAPIError(serviceOf(provider), resp.StatusCode, envelope, body)
	noteCompletionFillsContext(e, envelope.message())
	// A usage limit says when it resets, as Codex reads it (MADR 0012 §4.4).
	// The body's instant outranks a header's, and gives the delay when no
	// header did (0072-MADR D1).
	if !envelope.resetAt.IsZero() {
		e.ResetAt = envelope.resetAt
		if d := envelope.resetAt.Sub(now); e.RetryAfter == 0 && d > 0 {
			e.RetryAfter = d
		}
	}
	// Google's body may carry a RetryInfo saying how long to wait, when no
	// header did (0026-MADR F12; Gemini's Interactions API sends Retry-After,
	// 0027-MADR amendment of 2026-10-07).
	if e.RetryAfter == 0 {
		e.RetryAfter = envelope.retryDelay
	}
	// Last, a rate limit's reset header, as OpenAI, Hugging Face and Claude
	// send them (0028-MADR D-H5).
	if e.RetryAfter == 0 && resp.StatusCode == http.StatusTooManyRequests {
		var at time.Time
		e.RetryAfter, at = transport.RateLimitResetAt(resp.Header, now)
		if e.ResetAt.IsZero() {
			e.ResetAt = at
		}
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
// APIError's Status is 0: there is no HTTP status. A rate limit whose
// message states a delay, such as "Please try again in 11.054s.", carries it
// in RetryAfter (0072-MADR D3). It is ClassifyStreamFailureBody with no error
// object, and returns a non-nil *APIError. It is part of the error model, for
// any provider's stream reader (0015-MADR, amendment "S7b's import graph").
func ClassifyStreamFailure(provider, code, errType, message string) error {
	return ClassifyStreamFailureBody(provider, code, errType, message, nil)
}

// ClassifyStreamFailureBody classifies a failure reported inside a 200 event
// stream as ClassifyStreamFailure does, and also reads raw, the stream's
// error object as it came, with the parser that reads an HTTP error body
// (0072-MADR D3). Its types join code's and errType's, after them, and it
// sets RetryAfter, ResetAt and Limit from the fields ClassifyHTTPError reads.
// The delay comes from, in order: the object's headers member, read as HTTP
// headers; its reset time (resets_at, metadata.resetsAt) or a Google
// RetryInfo; and, for a rate limit that is not exhausted quota, its message,
// as Codex reads it ("try again in 11.054s"). A raw that is not a JSON
// object is ignored. It returns a non-nil *APIError. It is part of the error
// model, for any provider's stream reader, as ClassifyStreamFailure is.
func ClassifyStreamFailureBody(provider, code, errType, message string, raw []byte) error {
	env := apiErrorEnvelope{msg: message}
	for _, t := range []string{code, errType} {
		if t != "" && !env.hasType(t) {
			env.types = append(env.types, t)
		}
	}
	var headers http.Header
	if obj := bytes.TrimSpace(raw); len(obj) > 0 && obj[0] == '{' {
		body := parseAPIErrorBody(append(append([]byte(`{"error":`), obj...), '}'))
		for _, t := range body.types {
			if !env.hasType(t) {
				env.types = append(env.types, t)
			}
		}
		env.resetAt, env.retryDelay, env.perDayQuota = body.resetAt, body.retryDelay, body.perDayQuota
		env.plan, env.window, env.limitName, env.creditsURL, env.quotas = body.plan, body.window, body.limitName, body.creditsURL, body.quotas
		headers = streamErrorHeaders(obj)
	}
	e := &APIError{Provider: provider, Code: boundCode(env.errType()),
		Message: boundMessage(redact.String(redact.StripControl(redactInput(message)))),
		Limit:   limitOf(env)}
	setStreamKind(e, code, errType, message, env)
	now := time.Now()
	if headers != nil {
		e.RetryAfter, e.ResetAt = transport.RetryAfterAt(headers, now)
	}
	if !env.resetAt.IsZero() {
		e.ResetAt = env.resetAt
		if d := env.resetAt.Sub(now); e.RetryAfter == 0 && d > 0 {
			e.RetryAfter = d
		}
	}
	if e.RetryAfter == 0 {
		e.RetryAfter = env.retryDelay
	}
	if kind := e.kind(); e.RetryAfter == 0 && errors.Is(kind, ErrRateLimited) && !errors.Is(kind, ErrQuotaExhausted) {
		e.RetryAfter = textDelay(message)
	}
	return e
}

// tryAgainIn is the delay a rate limit's message states, as Codex reads it
// (codex-rs/codex-api/src/sse/responses_error.rs; 0072-MADR X13).
var tryAgainIn = regexp.MustCompile(`(?i)try again in\s*(\d+(?:\.\d+)?)\s*(s|ms|seconds?)`)

// textDelay is the delay message states, or 0; a delay too long for a
// Duration is the longest one.
func textDelay(message string) time.Duration {
	m := tryAgainIn.FindStringSubmatch(message)
	if m == nil {
		return 0
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil || n <= 0 {
		return 0
	}
	unit := time.Second
	if strings.EqualFold(m[2], "ms") {
		unit = time.Millisecond
	}
	if d := n * float64(unit); d < math.MaxInt64 {
		return time.Duration(d)
	}
	return math.MaxInt64
}

// streamErrorHeaders is a stream error object's headers member as HTTP
// headers, as Codex reads a failed response's (0072-MADR X11), or nil.
func streamErrorHeaders(obj []byte) http.Header {
	var m struct {
		Headers map[string]json.RawMessage `json:"headers"`
	}
	if json.Unmarshal(obj, &m) != nil || len(m.Headers) == 0 {
		return nil
	}
	h := http.Header{}
	for name, value := range m.Headers {
		if s := scalarText(value); s != "" {
			h.Set(name, s)
		}
	}
	return h
}

// setStreamKind sets e's kind and terminality from code, errType, message
// and env, as ClassifyStreamFailure has since 0026-MADR F5.
func setStreamKind(e *APIError, code, errType, message string, env apiErrorEnvelope) {
	provider := e.Provider
	// A gateway that answers 200 with an error, as OpenRouter does, puts the
	// HTTP status in code: classify it as that status, through the service's
	// own table and the overflow check (0026-MADR F5).
	if status, err := strconv.Atoi(code); err == nil && status >= http.StatusBadRequest && status <= 599 {
		e.terminal, e.Kind = classifyAPIError(serviceOf(provider), status, env, nil)
		noteCompletionFillsContext(e, message)
		return
	}
	switch {
	case env.hasType("rate_limit_exceeded") || env.hasType("slow_down"):
		e.Kind = ErrRateLimited
	case env.hasType("context_length_exceeded"):
		e.terminal, e.Kind = true, ErrContextOverflow
	case env.hasType("invalid_prompt"):
		e.terminal, e.Kind = true, ErrInvalidRequest
	default:
		// 500 stands in for "no status": the table's codes win; else the
		// error's type decides (0028-MADR D-H8); else retryable.
		e.terminal, e.Kind = classifyAPIError(serviceOf(provider), http.StatusInternalServerError, env, nil)
		if errors.Is(e.Kind, ErrProviderUnavailable) {
			if terminal, ok, kind := classifyStreamType(env, errType); ok {
				e.terminal, e.Kind = terminal, kind
			}
		}
	}
	noteCompletionFillsContext(e, message)
}

// classifyStreamType classifies a failure inside a 200 by its OpenAI- or
// Anthropic-style type, when its code said nothing (0028-MADR D-H8). An
// invalid request, or a failure with no type, is checked for overflow first.
// ok is false when the type decides nothing.
func classifyStreamType(env apiErrorEnvelope, errType string) (terminal, ok bool, kind error) {
	overflow := contextOverflow(env) && !completionFillsContext(env.message())
	switch errType {
	case "invalid_request_error":
		if overflow {
			return true, true, ErrContextOverflow
		}
		return true, true, ErrInvalidRequest
	case "":
		if overflow {
			return true, true, ErrContextOverflow
		}
		return false, false, nil
	case "authentication_error":
		return true, true, ErrAuthFailure
	case "permission_error":
		return true, true, ErrNotPermitted
	case "rate_limit_error":
		return false, true, ErrRateLimited
	case "overloaded_error", "api_error", "server_error":
		return false, true, ErrProviderUnavailable
	default:
		return false, false, nil
	}
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
	// BudgetLimitError and CreditLimitExceeded are quota as OpenCode's own
	// client treats them, which does not retry a budget (0072-MADR D4, O7).
	opencodeQuotaTypes     = []string{"FreeUsageLimitError", "GoUsageLimitError", "BlackUsageLimitError", "CreditsError", "MonthlyLimitError", "UserLimitError", "BudgetLimitError", "CreditLimitExceeded"}
	opencodeForbiddenTypes = []string{"RegionError", "DataPolicyError", "FreeTierError"}
	// subscription_sharing_usage_limit_exceeded is a ChatGPT plan session's
	// usage limit, which a retry within minutes does not pass (0043-MADR D6).
	openAIQuotaTypes = []string{"usage_limit_reached", "insufficient_quota", "subscription_sharing_usage_limit_exceeded"}
	// openAIPlanRefusals and openAIPlanInvalid are a plan session's other
	// codes, named so that they mean in a stream what their 403 and 400
	// mean in a reply (0043-MADR A4).
	openAIPlanRefusals = []string{"subscription_sharing_user_not_eligible", "subscription_sharing_route_not_supported"}
	openAIPlanInvalid  = []string{"subscription_sharing_unsupported_capability"}
	// geminiAuthReasons are the details reasons Gemini gives a refused key
	// with HTTP 400, as captured live (0020-MADR F23, Q6 a).
	geminiAuthReasons = []string{"API_KEY_INVALID"}
	// credential403Codes are, per service, the codes a provider was measured
	// sending with HTTP 403 for a refused credential; any other 403 is
	// ErrNotPermitted (0028-MADR D-H2). 0028-PLAN Phase 1's T1 found none:
	// every provider refused a key with 401 or 400.
	credential403Codes = map[string][]string{}
	// terminalUnavailable are the 5xx statuses no retry can mend: 501 and
	// 505 refuse the request itself, and 525 and 526 the origin's TLS
	// (0068-MADR D2).
	terminalUnavailable = []int{http.StatusNotImplemented, http.StatusHTTPVersionNotSupported, 525, 526}
)

// classifyAPIError applies MADR 0012 §1.1's table (with its 2026-09-27 rows)
// and falls back to the status-only mapping.
func classifyAPIError(service string, status int, env apiErrorEnvelope, body []byte) (terminal bool, sentinel error) {
	has := func(types ...string) bool {
		return slices.ContainsFunc(types, env.hasType)
	}
	switch workspace := claudeWorkspaceRefusal(service, env); {
	case service == serviceOpencode && has(opencodeQuotaTypes...),
		service == string(ProviderKilo) && (has("PROMOTION_MODEL_LIMIT_REACHED") || bytes.Contains(body, []byte("FreeUsageLimitError"))),
		service == string(ProviderOpenAI) && has(openAIQuotaTypes...),
		status == http.StatusPaymentRequired:
		return true, ErrQuotaExhausted
	// OpenCode refuses a model or a route with 403, typed or not; a bad
	// key is its 401 (0012-MADR, amendment 2026-10-05; 0021-MADR L3).
	case service == serviceOpencode && (status == http.StatusForbidden || has(opencodeForbiddenTypes...)),
		service == string(ProviderOpenAI) && (has("usage_not_included") || has(openAIPlanRefusals...)),
		service == string(ProviderKilo) && (status == http.StatusForbidden || has("data_collection_required")):
		return true, ErrNotPermitted
	case service == string(ProviderGemini) && has(geminiAuthReasons...):
		return true, ErrAuthFailure
	// xAI refuses a key with HTTP 400 invalid-argument, naming the key
	// (0028-PLAN Phase 1, T1; 0028-MADR H11); an invalid argument that does
	// not is the request's.
	case service == string(ProviderGrok) && status == http.StatusBadRequest && has("invalid-argument") &&
		strings.Contains(strings.ToLower(env.message()), "api key"):
		return true, ErrAuthFailure
	// Anthropic's workspace refusals (0064-MADR D3); any other 400 or 404 is
	// the request's.
	case workspace != nil:
		return true, workspace
	// A 403 refuses something the key may not do, not the key, unless it
	// carries a code a provider was measured sending for a refused
	// credential (0028-MADR D-H2). The legacy status sentinel still matches
	// ErrAuthFailure.
	case status == http.StatusForbidden && !has(credential403Codes[service]...):
		return true, ErrNotPermitted
	// A per-day quota cannot be passed by a retry within the day
	// (0026-MADR F12).
	case service == string(ProviderGemini) && status == http.StatusTooManyRequests && env.perDayQuota:
		return true, ErrQuotaExhausted
	case service == serviceOpencode && has("ModelError"),
		service == string(ProviderKilo) && has("PAID_MODEL_AUTH_REQUIRED"),
		service == string(ProviderOpenAI) && has(openAIPlanInvalid...):
		return true, ErrInvalidRequest
	case status == http.StatusTooManyRequests:
		return false, ErrRateLimited
	case status == http.StatusRequestTimeout:
		return false, ErrProviderUnavailable
	// A conflict is a lock timeout to OpenAI and Anthropic, whose SDKs retry
	// it (0021-MADR T14).
	case status == http.StatusConflict && (service == string(ProviderOpenAI) || service == string(ProviderClaude)):
		return false, ErrProviderUnavailable
	case slices.Contains(terminalUnavailable, status):
		return true, ErrProviderUnavailable
	case status >= http.StatusInternalServerError:
		return false, ErrProviderUnavailable
	// An overflow whose completion alone fills the context cannot be mended
	// by shortening the input, so it stays an invalid request
	// (0028-MADR D-H3).
	case errors.Is(statusSentinel(status), ErrInvalidRequest) && contextOverflow(env) && !completionFillsContext(env.message()):
		return true, ErrContextOverflow
	default:
		return true, statusSentinel(status)
	}
}

// claudeWorkspaceRefusal is the class of an Anthropic workspace refusal, each
// matched by its type and a fixed phrase, or nil: ErrWorkspaceRequired for a
// key not scoped to a workspace that names none; ErrWorkspaceNotFound for a
// value that is not a workspace ID (its 400, measured) and for a workspace
// that does not exist or the key may not use (its 404, as the documentation
// words it) (0064-MADR D3, E5).
func claudeWorkspaceRefusal(service string, env apiErrorEnvelope) error {
	if service != string(ProviderClaude) {
		return nil
	}
	msg := env.message()
	switch {
	case env.hasType("invalid_request_error") && strings.Contains(msg, "is not scoped to a workspace"):
		return ErrWorkspaceRequired
	case env.hasType("invalid_request_error") && strings.Contains(msg, "must be a valid workspace ID"),
		env.hasType("not_found_error") && strings.HasPrefix(msg, "Workspace") && strings.HasSuffix(msg, "not found."):
		return ErrWorkspaceNotFound
	}
	return nil
}

// apiErrorEnvelope is the union of the error bodies MADR 0012 §1.1 lists:
// OpenCode and Claude {type:"error",error:{type,message}}, Kilo
// {error:{code,message}}, {code} or {error,error_type,message},
// OpenAI/Codex {error:{type,code,message}},
// xAI nested or flat {code,error}, Gemini {error:{code,message,status,
// details:[{reason}]}}, which its Interactions API sends as a one-element
// array, [{error:{…}}] (0026-PLAN D12).
type apiErrorEnvelope struct {
	types []string // candidate classifications, most specific first
	msg   string
	// resetAt is the instant the body names for its limit to lift, in UTC,
	// or zero: usage_limit_reached's resets_at, else OpenCode's
	// metadata.resetsAt (0072-MADR D1).
	resetAt time.Time
	// retryDelay is a google.rpc.RetryInfo's delay, or 0 (0026-MADR F12).
	retryDelay time.Duration
	// perDayQuota reports a google.rpc.QuotaFailure naming a per-day quota.
	perDayQuota bool
	// What the body states about its limit, raw, for limitOf (0072-MADR
	// D2): ChatGPT's plan_type and limit_window_minutes, OpenCode's
	// metadata.limitName, Kilo's error.metadata.buyCreditsUrl, and a
	// QuotaFailure's violations.
	plan       string
	window     time.Duration
	limitName  string
	creditsURL string
	quotas     []Quota
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

// soleObject returns the element of a JSON array holding one object, and any
// other body as it is: Gemini's Interactions API wraps each error in such an
// array (0026-PLAN D12).
func soleObject(body []byte) []byte {
	var elems []json.RawMessage
	if json.Unmarshal(body, &elems) != nil || len(elems) != 1 || !bytes.HasPrefix(elems[0], []byte("{")) {
		return body
	}
	return elems[0]
}

func parseAPIErrorBody(body []byte) apiErrorEnvelope {
	var top struct {
		Type      string          `json:"type"`
		Code      json.RawMessage `json:"code"`
		ErrorType string          `json:"error_type"`
		Message   string          `json:"message"`
		Error     json.RawMessage `json:"error"`
		Detail    json.RawMessage `json:"detail"`
		// Metadata is OpenCode's, beside error (0072-MADR O1, O4).
		Metadata json.RawMessage `json:"metadata"`
	}
	if json.Unmarshal(soleObject(body), &top) != nil {
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
		// PlanType and LimitWindowMinutes are ChatGPT's usage limit's
		// (0072-MADR X3, X5), kept raw so that a value of another type
		// spoils nothing else.
		PlanType           json.RawMessage `json:"plan_type"`
		LimitWindowMinutes json.RawMessage `json:"limit_window_minutes"`
		// Metadata is Kilo's, and OpenCode's in some envelopes, inside
		// error (0072-MADR K1, O6).
		Metadata json.RawMessage `json:"metadata"`
		// Details are Google's: a reason is the most specific type; a
		// RetryInfo carries the delay and a QuotaFailure the violated
		// quotas (google/rpc/error_details.proto; 0026-MADR F12); an
		// ErrorInfo carries metadata (0072-MADR O5).
		Details []struct {
			Type       string `json:"@type"`
			Reason     string `json:"reason"`
			RetryDelay string `json:"retryDelay"`
			Violations []struct {
				QuotaID     string          `json:"quotaId"`
				QuotaMetric string          `json:"quotaMetric"`
				QuotaValue  json.RawMessage `json:"quotaValue"`
			} `json:"violations"`
			Metadata json.RawMessage `json:"metadata"`
		} `json:"details"`
	}
	// metadata are the error's metadata objects, in the order a reset is
	// read from them (0072-MADR D1): beside error, inside it, then in a
	// Google ErrorInfo.
	metadata := []json.RawMessage{top.Metadata}
	var text string
	switch {
	case json.Unmarshal(top.Error, &inner) == nil:
		metadata = append(metadata, inner.Metadata)
		for _, d := range inner.Details {
			add(d.Reason)
			if strings.HasSuffix(d.Type, "google.rpc.RetryInfo") {
				env.retryDelay = protoDuration(d.RetryDelay)
			}
			if strings.HasSuffix(d.Type, "google.rpc.ErrorInfo") {
				metadata = append(metadata, d.Metadata)
			}
			for _, v := range d.Violations {
				env.perDayQuota = env.perDayQuota || strings.Contains(v.QuotaID, "PerDay")
				env.quotas = append(env.quotas, Quota{ID: v.QuotaID, Metric: v.QuotaMetric, Value: scalarText(v.QuotaValue)})
			}
		}
		add(jsonString(inner.Code), inner.Type, inner.Status)
		env.msg = inner.Message
		if inner.ResetsAt > 0 {
			env.resetAt = time.Unix(inner.ResetsAt, 0).UTC()
		}
		env.plan = jsonString(inner.PlanType)
		env.window = windowMinutes(inner.LimitWindowMinutes)
		env.creditsURL = jsonString(readMetadata(inner.Metadata).BuyCreditsURL)
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
	for _, raw := range metadata {
		if !env.resetAt.IsZero() {
			break
		}
		env.resetAt = metadataResetAt(readMetadata(raw))
	}
	// A limit's name is OpenCode's, beside error or inside it (0072-MADR
	// D2, O1).
	if env.limitName = jsonString(readMetadata(top.Metadata).LimitName); env.limitName == "" {
		env.limitName = jsonString(readMetadata(inner.Metadata).LimitName)
	}
	return env
}

// metadataResetAt is an error metadata object's resetsAt, an RFC 3339
// instant, in UTC, or zero when it is absent or does not parse (0072-MADR
// D1, O4).
func metadataResetAt(m errorMetadata) time.Time {
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(jsonString(m.ResetsAt)))
	if err != nil {
		return time.Time{}
	}
	return at.UTC()
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

// redactSeparators end a word: a cut made at one never splits a secret.
const redactSeparators = " \t\r\n,;\"'"

// redactInput is the part of a message that is redacted and kept: at most
// apiErrorRedactLimit bytes, cut back to the last separator before the bound,
// so that no secret is cut into a fragment too short to be recognised, and on
// a rune boundary. A bound with no separator before it, one word of 2 KiB, is
// cut at the bound (0028-MADR D-A10).
func redactInput(s string) string {
	if len(s) <= apiErrorRedactLimit {
		return s
	}
	cut := apiErrorRedactLimit
	if i := strings.LastIndexAny(s[:cut], redactSeparators); i > 0 {
		cut = i
	}
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
