---
status: in-progress
date: 2026-11-09
associated-madr: "0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement an Absolute Reset Time and a Stated Limit on APIError

Associated MADR: [0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md](0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md)

## Goal

After this PLAN, a consumer reads from an `*APIError`:

* the instant a limit lifts, as the service named it (`ResetAt`);
* what the service stated about the limit (`Limit`).

It reads both from an HTTP reply and from a failure inside a 200 stream, and
reads nothing the service did not state. It also changes two classifications:

* OpenCode's budget is no longer retried;
* a Gemini per-day quota in a stream is no longer retried.

`Error()`'s format, `Retryable`, `Unwrap`, `Code` and the request on the wire
are unchanged. This answers GitHub issue 13, requirements 1 to 4.
Requirement 5 is deferred (MADR D5).

The steps follow the MADR's recommended answers to Q1 to Q7. If the owner
answers otherwise, the MADR and this PLAN are amended before Phase 1.
(2026-11-09: the owner answered each question, and Q-risk, as recommended,
and the MADR is accepted. This PLAN awaits approval.)

## Scope

### In scope

* **D1, `ResetAt`:**
  * `llmprovider/api_error.go`;
  * `llmprovider/internal/transport/transport.go`.
* **D2, `Limit` and `Quota`:**
  * `llmprovider/api_error.go`;
  * a new `llmprovider/api_limit.go` for the two types and their docs.
* **D3, `ClassifyStreamFailureBody`,** and its use in the five stream parsers:
  * `llmprovider/internal/wire/responses/responses.go`;
  * `llmprovider/internal/wire/generatecontent/generatecontent.go`;
  * `llmprovider/providers/gemini/interactions.go`;
  * `llmprovider/internal/wire/messages/messages.go`;
  * `llmprovider/internal/wire/chatcompletions/chatcompletions.go`.
* **D4:** two codes in `opencodeQuotaTypes`.
* **D6:** the tests, an `Example`, the docs, the changelog and the release.
* **The measurement phase:** a live test of ChatGPT's headers, and the Gemini
  live check extended.

### Exported names (R55)

| Name | Call site that needs it |
| :--- | :--- |
| `APIError.ResetAt time.Time` | gobble-cli's `promptError`, for `ErrorData.resetAt`: "or the SDK's absolute reset time, once it has one" (its issue 14, requirement 1); magic-cli-remote's 0243-MADR requirement 6 (MADR M2) |
| `APIError.Limit *Limit` | the same `ErrorData`, whose `limit` gobble fills from its table and the stated facts (issue 14, requirements 1 and 2) |
| `type Limit` | holds the fields below |
| `Limit.Name string` | gobble's `weekly` and `window` scopes, which "need a source that states the window" (issue 14, requirement 2); 0243-MADR D3's `Scope` |
| `Limit.Window time.Duration` | the same; 0243-MADR D3's `WindowMinutes` |
| `Limit.Plan string` | 0243-MADR requirement 6, "ChatGPT `plan_type`"; issue 13, requirement 2 |
| `Limit.PerDay bool` | gobble's `daily` scope, "Gemini per-day" (issue 14, requirement 2); 0243-MADR requirement 6, "the per-day flag" |
| `Limit.Quotas []Quota` | 0243-MADR requirement 6, "Gemini `quotaId`, `quotaMetric`, `quotaValue`" |
| `type Quota` | holds the three fields below |
| `Quota.ID string` | as `Limit.Quotas` |
| `Quota.Metric string` | as `Limit.Quotas` |
| `Quota.Value string` | as `Limit.Quotas` |
| `Limit.CreditsURL string` | 0243-MADR requirement 6, "Kilo's `metadata`"; issue 13, requirement 4, "for a link to add credits" |
| `ClassifyStreamFailureBody(provider, code, errType, message string, raw []byte) error` | the five stream parsers in this module (MADR C19 to C24), which `ClassifyStreamFailure` serves today for the same reason (0015-MADR, amendment "S7b's import graph") |

No other name is exported.

**Exported, but internal to the module.** Two functions are exported from
`llmprovider/internal/transport` so that `llmprovider` can call them. Being
internal, they add nothing to the module's API (as 0071-PLAN's Deviation 1):

* `RetryAfterAt`;
* `RateLimitResetAt`.

### Out of scope

* Remaining-quota headers on `Response` (MADR D5, Q3).
* A scope enum (MADR Q2).
* Exposing these:
  * OpenCode's `metadata.workspace` or `metadata.coverage`;
  * Gemini's `quotaDimensions`;
  * any other metadata key (MADR D2).
* A numeric `error.code` in `Code` (MADR D2, row 7).
* `Unwrap`'s status sentinel for OpenCode's 401 quota codes (MADR, More
  Information).
* `x-codex-*` headers, which are measured in Phase 5 and not read.
* gobble-cli's and magic-cli-remote's own changes, in their own repositories.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts only after the owner has committed the last one.
* **Red, then green.** Each new test is shown to fail one of two ways:
  * **A test that calls only existing API** is seen red at `d62142c` with
    `python3 -B tools/redgreen.py SPEC.json --expect fail --rev d62142c`, and
    green on the phase's tree with `--expect pass`.
  * **A test that calls a new name** cannot compile at `d62142c`. It is seen
    to fail on each of its planted breaks, with `python3 -B tools/plant.py
    SPEC.json`, on a scratch clone.

  The specs live in the session's scratch directory, and this record holds
  each failure line. If a break cannot be made to fail its test on every
  run, that is a deviation: stop and report it. Do not loosen the test.
* **The drift check.** Every existing test passes unchanged (R46), in
  particular:
  * `api_error_test.go`, `api_error_gemini_0026_test.go`,
    `api_error_ratelimit_0028_test.go`, `api_error_0043_test.go`,
    `chatgpt_errors_test.go` and `gemini429_check_test.go`;
  * `internal/transport`'s `Retry-After` tests;
  * every stream test in the five wire packages;
  * G-wire's goldens (R45).
* **Fixtures are synthetic.** Every fixture body is built from a shape the
  MADR cites, and its comment names that claim's id (X3, O4, K1, …). None
  carries an id, an account or a workspace. `make fixtures-check` stays
  clean.
* **Pre-add.** `make pre-add-check FILES="…"` on the phase's Go files:
  * gofmt;
  * golangci-lint on the host and with `GOOS=windows`;
  * vet;
  * the packages' tests;
  * govulncheck.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`, or `--markdown` for a records-only phase;
  * `python3 -B tools/claims.py` on the MADR;
  * `make records-check`;
  * `python3 -B tools/recordlint.py --changed`;
  * `python3 -B tools/denyscan.py --changed`;
  * `python3 -B tools/linkcheck.py`.

  Each exit status is captured before any filter, and read before anything is
  written from it. Files are staged only when the gate passes (`&&`).
* **R50 to R52:**
  * `Limit` and `Quota` each say they are safe for concurrent use while no
    goroutine changes them.
  * `ClassifyStreamFailureBody` says it returns a non-nil `*APIError`, as
    `ClassifyStreamFailure` does.
  * `make docs-check` stays at 0.
* **Stage, then the owner commits** with `git commit --no-edit`. Pushes and
  tags happen on the owner's word, after the disclosure guard has run over
  the outgoing commits.
* **Stop conditions:**
  * a fact of the MADR, or one of its claims, no longer holds;
  * an existing test needs a changed assertion;
  * `Error()`'s text changes for an error that exists today;
  * `apidiff` against `v1.28.0` lists anything beyond the R55 table;
  * a golden changes;
  * a package falls below its floor in `scripts/coverage-floors.txt` (R47);
    `llmprovider`'s is 95.9;
  * `-race` reports a race.

  On any of these: stop, record a dated deviation here (with an amendment in
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR and their rows in `docs/README.md` as records
   only.
2. Once the owner has answered Q1 to Q7:
   * the MADR records the answers, as `accepted`;
   * `tools/records.py status` sets this PLAN `in-progress`.
3. Add dated amendments, without rewriting the decisions:
   * **`0012-MADR-conform-providers-to-reference-clients.md`, §1.1:** OpenCode
     `BudgetLimitError` and `CreditLimitExceeded` are `ErrQuotaExhausted`,
     terminal (0072-MADR D4, and its evidence O3 to O5 and O7).
   * **`0015-MADR-canonical-sdk-api-and-module-layout.md`, D7:**
     * `*APIError` also carries `ResetAt` and `Limit`;
     * `ClassifyStreamFailureBody` reads a stream's error object as an
       error body;
     * R24's text after it (0072-MADR D1 to D3).
   * **`0026-MADR-remediate-v1-2-debugging-pass-findings.md`, F12:** the
     per-day flag is exposed as `Limit.PerDay`, and the violations as
     `Limit.Quotas`. A per-day quota inside a stream is classified as in an
     HTTP reply (0072-MADR D2, D3).
4. Run `python3 -B tools/gate.py --markdown`, then stage.

### Phase 1: `ResetAt` (D1)

1. **`internal/transport`.**
   * **`RetryAfterAt(h http.Header, now time.Time) (time.Duration, time.Time)`**
     returns the delay `RetryAfter` returns today, and the instant when the
     header it read was an HTTP-date, in UTC; otherwise the zero time.
     * `retry-after-ms` keeps its precedence. Neither it nor delta seconds
       names an instant.
     * The delay of an HTTP-date is `t.Sub(now)` when positive. That is
       today's `time.Until(t)` with the clock passed in.
     * `RetryAfter(h)` becomes `d, _ := RetryAfterAt(h, time.Now())`.
     * `ParseRetryAfter` is unchanged.
   * **`RateLimitResetAt(h http.Header, now time.Time) (time.Duration, time.Time)`**
     returns the longest wait, as `RateLimitReset` does. When that wait came
     from an RFC 3339 header, it also returns the instant, in UTC; when a Go
     duration header gave it, the zero time. A tie keeps the duration
     header's zero instant.
     * `RateLimitReset` becomes its wrapper, with the same result.
2. **The envelope** (`apiErrorEnvelope` and `parseAPIErrorBody`):
   * **`resetAt time.Time`:**
     * from `error.resets_at`, as `time.Unix(n, 0).UTC()` when `n > 0`;
     * else from `metadata.resetsAt` (RFC 3339), read in this order: the
       top-level `metadata`, `error.metadata`, then each `details[]` entry
       whose `@type` ends `google.rpc.ErrorInfo`.

     A value that does not parse is ignored.
   * The `resetsAt int64` field is replaced by `resetAt` (MADR C7).
3. **`ClassifyHTTPError`:**
   * **`RetryAfter`:** `RetryAfter`, `ResetAt := transport.RetryAfterAt(resp.Header, now)`.
   * **The body's reset.** When `envelope.resetAt` is set, `ResetAt` takes it,
     which outranks the header's (MADR D1, source 1). When `RetryAfter` is 0
     and the instant is ahead of `now`, `RetryAfter` takes `resetAt.Sub(now)`.
     That is today's rule (C3), for both spellings.
   * **`RetryInfo`,** unchanged.
   * **The reset headers.** When `RetryAfter` is still 0 on a 429:
     `d, at := transport.RateLimitResetAt(resp.Header, now)`. `RetryAfter`
     takes `d`, and `ResetAt` takes `at` if it is still zero.
   * **The clock.** `now` is read once, at the start, with `time.Now()`. No
     test seam is added: the tests use instants an hour or more away.
4. **`APIError.ResetAt`'s doc** follows MADR D1's wording. It says:
   * the instant may have passed;
   * it is zero when the service named only a delay;
   * `Error()` does not print it.
5. **Tests.**
   * **In `llmprovider/internal/transport/reset_at_0072_test.go`:**

     | Test | Asserts | Planted breaks |
     | :--- | :--- | :--- |
     | `TestRetryAfterAt_0072` | An HTTP-date gives its instant in UTC and the delay to it; delta seconds and `retry-after-ms` give a zero instant; `retry-after-ms` wins over an HTTP-date `Retry-After`, with a zero instant. Its delays equal `RetryAfter`'s for each case. | a delta gives `now+d` as its instant; the instant is returned in `time.Local` |
     | `TestRateLimitResetAt_0072` | Claude's RFC 3339 headers give the latest instant; a longer `x-ratelimit-reset-*` duration gives a zero instant; the delays equal `RateLimitReset`'s. | the RFC 3339 instant is returned even when a duration header won |

   * **In `llmprovider/api_error_reset_0072_test.go`:**

     | Test | Asserts | Planted breaks |
     | :--- | :--- | :--- |
     | `TestResetAt_ResetsAt_0072` | A ChatGPT 429 with `resets_at` an hour ahead (fixture from X3 to X5) has `ResetAt == time.Unix(n, 0)`, in `time.UTC`, and a `RetryAfter` between 59 m and 1 h, as today. | `ResetAt` left zero; `ResetAt` in `time.Local` |
     | `TestResetAt_PastKept_0072` | `resets_at` an hour ago gives that instant, and `RetryAfter` 0 (MADR Q5). | a past instant is zeroed |
     | `TestResetAt_Order_0072` | `resets_at` with `Retry-After: 30` gives the body's `ResetAt` and a 30 s `RetryAfter`. An HTTP-date alone gives its instant. Claude's 429 with its reset header gives the header's instant. The same header on a 400 gives none. | the header is preferred to the body; the reset header is read on any status |
     | `TestResetAt_OpenCodeMetadata_0072` | `metadata.resetsAt` (O4) at the top level, in `error.metadata`, and in a Google `ErrorInfo` detail (O5) each set `ResetAt`, and `RetryAfter` when no header did. A malformed `resetsAt` sets neither. | the top-level `metadata` is not read; the `ErrorInfo` detail is not read |
     | `TestAPIError_TextUnchanged_0072` | For six errors, `Error()` equals that of a copy with `ResetAt` and `Limit` zeroed: a ChatGPT usage limit, a Gemini per-day 429, an OpenCode Go 429, a Kilo 402, a Claude 429 with a reset header, and a stream failure. One is pinned as a literal string. | `Error()` appends the reset instant |

   * **Red at the base.** `TestResetAt_*` call the new field, so they are
     planted, not red-at-base.

### Phase 2: `Limit` and the OpenCode codes (D2, D4)

1. **`llmprovider/api_limit.go`** holds `Limit` and `Quota`, with MADR D2's
   fields.
   * Each field's doc names its source, as D2's table does.
   * `Limit`'s doc carries the rule: "every field is the service's; none is
     inferred, and a window is never inferred from how long the wait is."
2. **The envelope** gains:
   * `plan` (`error.plan_type`);
   * `window` (`error.limit_window_minutes`, a JSON integer);
   * `name` (`metadata.limitName`, at the top level or in `error.metadata`);
   * `creditsURL` (`error.metadata.buyCreditsUrl`);
   * `quotas`, from each `QuotaFailure` violation's `quotaId`, `quotaMetric`
     and `quotaValue`. The value is kept as a JSON string's text, or as a
     JSON number's decimal text.

   `perDayQuota` is unchanged.
3. **`limitOf(envelope) *Limit`** returns nil when every field is zero.
   * **`Window`** comes from `limit_window_minutes`, or from a `Name` that
     matches `^(\d+) hours?$`. It is kept only when it is from 1 minute to
     366 days.
   * **Every string** passes through `redact.Field`.
   * **`CreditsURL`** is kept only when all of these hold:
     * `url.Parse` succeeds;
     * the scheme is `https`;
     * the host is non-empty;
     * `User`, `RawQuery` and `Fragment` are empty;
     * `redact.String` leaves it unchanged.

   `ClassifyHTTPError` sets `e.Limit = limitOf(envelope)`.
4. **D4.** `"BudgetLimitError", "CreditLimitExceeded"` are appended to
   `opencodeQuotaTypes` (MADR C16). The comment cites 0072-MADR D4 and O7.
5. **Tests**, in `llmprovider/api_error_limit_0072_test.go` unless named:

   | Test | Asserts | Planted breaks |
   | :--- | :--- | :--- |
   | `TestLimit_ChatGPT_0072` | `plan_type: "plus"` and `limit_window_minutes: 300` (X3, X5) give `Plan "plus"` and `Window` 5 h. | `limit_window_minutes` not read |
   | `TestLimit_NeverInferred_0072` | These each give a zero `Window`: a ChatGPT 429 with `resets_at` 4 h ahead, `Retry-After: 14400` and no `limit_window_minutes`; an OpenCode Go 429 with `limitName: "weekly"` and `retry-after: 345600`; a Gemini per-day 429. | `Window` set from `RetryAfter` when none is stated; `"weekly"` read as 7 days |
   | `TestLimit_WindowBounds_0072` | `limit_window_minutes` of 0, -5, 600000 and 1.5, and a `limitName` of `"five hour"`, give a zero `Window`. `"1 hour"` and `"5 hours"` give 1 h and 5 h. | the bounds removed |
   | `TestLimit_OpenCodeGo_0072` | O1's body gives `Name "5 hour"` and `Window` 5 h. `fmt.Sprintf("%+v", *e.Limit)` and `e.Error()` hold no `wrk_`. | `limitName` not read |
   | `TestLimit_Gemini_0072` | A `QuotaFailure` with a per-minute and a per-day violation, one `quotaValue` a string and one a number, gives both in `Quotas`, in order, with `Value "5"` for each, and `PerDay` true. A per-minute violation alone gives `PerDay` false and a retryable `ErrRateLimited`. | `quotaMetric` not read; a numeric `quotaValue` dropped |
   | `TestLimit_KiloCredits_0072` | K1's 402 on `kilo` is `ErrQuotaExhausted`, with `CreditsURL` `https://app.kilo.ai/credits` and `Code` empty, as today. Each of these gives an empty `CreditsURL`: `http://`, user information, a query, a fragment, `javascript:`, and a URL holding a key that `redact.String` masks. | the scheme check removed; the query check removed |
   | `TestLimit_Redacted_0072` | A `plan_type` and a `limitName` holding control characters and 300 bytes come back stripped, within 128 bytes, on a rune boundary. | `Plan` not passed through `redact.Field` |
   | `TestLimit_Nil_0072` | A 429 with only a message, a 500 and a 401 have a nil `Limit`. | `Limit` always allocated |
   | `TestClassify_OpenCodeBudget_0072` | On `opencode-zen`, a 429 `BudgetLimitError` (O4) is `ErrQuotaExhausted` and not `Retryable`. So is the Google envelope's `ErrorInfo` reason (O5) on an `opencode-go` route label. A 402 `CreditLimitExceeded` (O3) is `ErrQuotaExhausted`. | red at `d62142c`: the budget is `ErrRateLimited` there |
   | `TestRetry_OpenCodeBudgetNotResent_0072` (`retry_0072_test.go`) | `WithRetry` over an `httptest` OpenCode route answering a budget 429 sends one request. | red at `d62142c`: it resends |

### Phase 3: stream failures (D3)

1. **`ClassifyStreamFailureBody(provider, code, errType, message string, raw []byte) error`:**
   * When `raw` is a JSON object, it is parsed by `parseAPIErrorBody` as
     `{"error": raw}`. Its types join the ones from `code` and `errType`, after
     them.
   * Classification is as `ClassifyStreamFailure`'s today, over the joined
     types. The status-in-code path (C14's `strconv.Atoi`) now sees the
     envelope's `perDayQuota`, so a Gemini per-day 429 in a stream is quota
     (C17).
   * `RetryAfter`, `ResetAt` and `Limit` are set from MADR D3's three sources,
     in order:
     1. the error's `headers` object, as an `http.Header` through
        `transport.RetryAfterAt`;
     2. the envelope's reset, then its `RetryInfo`;
     3. for `ErrRateLimited` that is not `ErrQuotaExhausted`, the message, by
        `(?i)try again in\s*(\d+(?:\.\d+)?)\s*(s|ms|seconds?)` (X13). The
        result is bounded as `durationOf` bounds a header's.
   * A `raw` that is not an object is ignored.
2. **`ClassifyStreamFailure`** becomes
   `ClassifyStreamFailureBody(provider, code, errType, message, nil)`. Its doc
   says what changed: a rate limit whose message states a delay now carries
   it (MADR D3).
3. **The five parsers pass their error object:**
   * **responses:**
     * `responseError` keeps its raw object, with an `UnmarshalJSON` that
       records the bytes;
     * `failed` passes it, which serves both the stream and the 200 JSON
       reply (`responses.go:389`);
     * the stream's `error` event passes its whole payload.
   * **generatecontent:** the stream callback reads the payload's `error`
     member as `json.RawMessage` beside `chunk.Error`, and passes it.
   * **gemini interactions:** the same, for `ev.Error`.
   * **messages:** `ev.Error` keeps its raw object, as in responses.
   * **chatcompletions:** `gatewayError` keeps its raw object, and `classify`
     passes it.
4. **Tests.**
   * **In `llmprovider/api_error_stream_0072_test.go`:**

     | Test | Asserts | Planted breaks |
     | :--- | :--- | :--- |
     | `TestClassifyStreamFailureBody_Headers_0072` | `{"code":"rate_limit_exceeded","message":"Slow down","headers":{"retry-after":"7"}}` gives `RetryAfter` 7 s (X11). An HTTP-date there gives `ResetAt`. | `headers` not read |
     | `TestClassifyStreamFailureBody_Text_0072` | The `RetryAfter` given by each message: "Please try again in 11.054s." gives 11.054 s; "try again in 250ms" gives 250 ms; "Try again in 3 seconds" gives 3 s. A `headers` delay wins over the text. `usage_limit_reached` with the same text gives no delay from it; so does `server_error`. | the pattern removed; the text read for a quota too |
     | `TestClassifyStreamFailureBody_UsageLimit_0072` | `usage_limit_reached` with `resets_at`, `plan_type` and `limit_window_minutes` in `raw` (X14) is `ErrQuotaExhausted`, with `ResetAt`, `Plan` and `Window`. | `raw` ignored |
     | `TestClassifyStreamFailureBody_Gemini_0072` | Code `"429"`, status `RESOURCE_EXHAUSTED`, and `raw` with a per-day `QuotaFailure` and a 52 s `RetryInfo`, give `ErrQuotaExhausted`, `RetryAfter` 52 s, and `Limit.PerDay`. | the envelope's per-day flag not passed to the status path |
     | `TestClassifyStreamFailure_Same_0072` | For each code, type and message in the existing stream tests' tables that states no delay, `ClassifyStreamFailure` and `ClassifyStreamFailureBody(…, nil)` give the same `Error()`, `Kind`, `Retryable` and `RetryAfter`. | `ClassifyStreamFailure` reads a fixed non-nil `raw` |

   * **Per wire,** in a `stream_error_0072_test.go` in each package. Each test
     calls only existing API: it decodes a stream and reads the error with
     `errors.As`. Each is seen red at `d62142c`, and also fails with its
     parser planted to pass `nil`. Each uses a field that only `raw` carries,
     so the plant shows the plumbing:

     | Test | Field only `raw` carries | Asserts |
     | :--- | :--- | :--- |
     | `responses.TestStream_FailedHeaders_0072` | `response.error.headers` | `RetryAfter` 7 s, in a stream and in a 200 JSON reply |
     | `generatecontent.TestEvents_PerDayQuota_0072` | `error.details` | `ErrQuotaExhausted`, `RetryAfter` 52 s |
     | `gemini.TestInteractionEvents_PerDayQuota_0072` | `error.details` | the same |
     | `messages.TestEvents_ErrorHeaders_0072` | `error.headers` (a plumbing fixture: Anthropic is not known to send it) | `RetryAfter` 7 s |
     | ~~`chatcompletions.TestEvents_GatewayErrorLink_0072`~~ `chatcompletions.TestEvents_GatewayErrorHeaders_0072` (Deviation 1) | ~~`error.metadata.buyCreditsUrl` with `code: 402` (K1's shape in a stream, K3)~~ `error.headers` (a plumbing fixture, as Messages') | ~~`Limit.CreditsURL` set~~ `RetryAfter` 7 s |

### Phase 4: docs and the example (D6)

1. **Docs of the new and changed names:**
   * `APIError`'s doc names `ResetAt` and `Limit`.
   * `ClassifyHTTPError`'s doc names D1's sources.
   * `ClassifyStreamFailure`'s doc says it is
     `ClassifyStreamFailureBody(…, nil)`.
2. **`ExampleAPIError_usageLimit`**, in `llmprovider/example_0072_test.go`:
   * it calls `ClassifyHTTPError` on an `http.Response` built in memory, a
     ChatGPT usage limit with a fixed `resets_at`;
   * it prints `Plan`, `Window`, `ResetAt.Format(time.RFC3339)` and
     `Retryable()`;
   * it has an `// Output:` comment.

   Planted break: the example's output, planted wrong.
3. **`docs/guides/api-standards.md` R24:** "It carries `Provider`, `Status`,
   `Kind`, `Code`, a redacted and bounded `Message`, `RetryAfter`, `ResetAt`,
   `Limit` and `Reason`. (0015 D7; 0072 D1, D2)". This is the 0015-MADR D7
   amendment made in Phase 0.
4. **`docs/architecture.md`:**
   * the errors bullets name `ResetAt` and `Limit`;
   * the stream paragraph names `ClassifyStreamFailureBody`;
   * the `internal/transport` row names `RetryAfterAt` and `RateLimitResetAt`.
5. **`docs/guides/migrating-from-mcplib.md`:**
   * the `APIError` row adds `ResetAt` and `Limit` to what the type adds;
   * the `RetryAfter` row adds "OpenCode's `metadata.resetsAt`, and a stream
     failure that states a delay".
6. **`CHANGELOG.md`,** under `[Unreleased]`:
   * **Added:** `APIError.ResetAt`, `APIError.Limit`, `Limit`, `Quota` and
     `ClassifyStreamFailureBody`.
   * **Fixed:**
     * an OpenCode budget 429 is not retried;
     * a Gemini per-day quota inside a stream is not retried;
     * a stream failure carries the delay its error states.
7. Run `make docs-check`, `make changelog-check`, `make parity-check` and
   `make links-check`.

### Phase 5: measurement (owner-run; Q-risk)

This phase reads live services and needs the owner's credentials. Its
results are recorded here. A shape that differs from the MADR's is a
deviation, and is never patched over in a test.

1. **`TestLive_0072ChatGPTHeaders`,** in `llmprovider/live_0072_measure_test.go`
   (build tag `live_gateways`).
   * It runs only with `LLMPROVIDER_LIVE_0072` and `LLMPROVIDER_LIVE_CHATGPT`
     set. It uses `liveChatGPTSession` (`live_chatgpt_session_test.go:24`).
   * It sends the smallest generation that `TestLive_ChatGPTGenerate` sends,
     through `WithHTTPClient`. That client's transport records the reply's
     header names.
   * It logs:
     * the names that start `x-codex-`;
     * the values of `x-codex-*-window-minutes` only.

     No other value is logged.
   * It answers the issue's second "Uncertain" item for this SDK's requests,
     for D5's later record.
2. **`checkGemini429`** (`gemini429_check_test.go`) also requires, when the
   body has a `QuotaFailure`, that:
   * `Limit.Quotas` holds each violation's `quotaId`, in order;
   * `Limit.PerDay` equals the per-day flag it computes.

   `TestCheckGemini429` gains a case for each. Planted break: `Quotas` left
   empty, which the check must report. The owner may then run
   `TestLive_GeminiRateLimitShape` (`LLMPROVIDER_LIVE_GEMINI_429`), which uses
   the check.
3. **`scripts/check_tables.py`** passes: `LLMPROVIDER_LIVE_0072` is this
   record's variable, named in this PLAN (AGENTS.md, "Live tests").

### Phase 6: the release (Q7: `v1.29.0`)

As 0071-PLAN's Phase 5:

1. The changelog section is renamed `v1.29.0`, and the README's current
   release is updated.
2. Run the full gate. `tools/apidiff.py v1.28.0` lists exactly the R55
   table's names.
3. Tag on the owner's word.
4. Wait for CI and the Release, then run `tools/consumer.py` and
   `tools/offline.py`.
5. On the owner's word:
   * comment on issue 13, naming requirement 5 as deferred to a later record,
     and close it;
   * comment on gobble-cli's issue 14 that the SDK's part is released.

## Verification

* Every test in Phases 1 to 3 passes. Each was seen red at `d62142c`, or seen
  to fail on each of its planted breaks. This record holds each failure line.
* Every existing test passes unchanged, and no golden changes.
* `go test -race -count=3` is clean over:
  * `./llmprovider`;
  * `./llmprovider/internal/transport/...`;
  * `./llmprovider/internal/wire/...`;
  * `./llmprovider/providers/gemini/...`.
* `apidiff` against `v1.28.0` lists exactly the R55 table.
* `TestAPIError_TextUnchanged_0072` holds.
* The MADR's claims pass. C3, C4, C7, C14, C15, C16 and C18 to C24 are
  annotated as changed by Phases 1 to 3.
* Coverage holds each package's floor.
* `make fixtures-check` is clean: no fixture carries an id.

## Rollout and Rollback

* **Rollout.** The change is additive.
  * A caller that reads neither `ResetAt` nor `Limit` sees two changes, both
    in behaviour the MADR names:
    * an OpenCode budget is no longer retried;
    * a stream failure may carry a `RetryAfter`, which `WithStreamRetry`
      honours before the first event.
  * `Error()`'s format is unchanged.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** the names stay (R48). A defect is fixed
  forward.
  * A field that proves wrong for a provider stops being set for it. That is
    a zero value, which every caller already handles.
  * A classification that proves wrong (D4) is reverted by an amendment to
    0012-MADR §1.1.

## Execution record

The owner answered Q1 to Q7 and Q-risk on 2026-11-09, each as recommended.
The owner committed the records (`27e2548`), and on 2026-11-09 accepted the
MADR, approved this PLAN and said to proceed.

### Phase 0 (2026-11-09)

* Step 1: done by the owner's commit `27e2548`.
* Step 2: the MADR was already `accepted`. `tools/records.py status` set this
  PLAN `in-progress`, with its index row.
* Step 3: three dated amendments, each appended after the record's last
  amendment, with no earlier text rewritten:
  * `0012-MADR-conform-providers-to-reference-clients.md`, "OpenCode's budget
    and credit limit are quota (0072 D4)";
  * `0015-MADR-canonical-sdk-api-and-module-layout.md`, "`*APIError` carries
    `ResetAt` and `Limit` (0072)";
  * `0026-MADR-remediate-v1-2-debugging-pass-findings.md`, "F12's quota is
    exposed, and read inside a stream".
* Step 4: `python3 -B tools/gate.py --markdown` exited 0, started at load
  9.9: "gate: 11 of 11 passed" (parity-check, records-check, tables-check,
  changelog-check, markdownlint, links-check, disclosure-check,
  gate-selftest, deny-scan, recordlint, claims). The five files were then
  staged. This entry was added after that run, so `make records-check`,
  `tools/recordlint.py --changed`, `tools/denyscan.py --changed` and
  `tools/linkcheck.py` were run again, each exiting 0, before the PLAN was
  re-staged. The PLAN holds no claims.

The owner committed Phase 0 (`a08eb37`) and said to proceed.

### Phase 1 (2026-11-09)

* **Step 1, `internal/transport`:**
  * `RetryAfterAt(h, now)` and `RateLimitResetAt(h, now)` were added.
  * `RetryAfter` and `RateLimitReset` became their wrappers.
  * `ParseRetryAfter` is unchanged.
  * An HTTP-date that has passed keeps its instant, with a zero delay, as
    MADR Q5 keeps a past body reset.
* **Step 2, the envelope:**
  * `resetsAt int64` became `resetAt time.Time`.
  * `parseAPIErrorBody` reads `metadata` beside `error`, `error.metadata`,
    and an `ErrorInfo` detail's `metadata`, in that order, after
    `resets_at`.
  * A new helper, `metadataResetAt`, parses `resetsAt` as RFC 3339.
* **Step 3, `ClassifyHTTPError`:** it reads `now` once and follows MADR D1's
  order.
* **Step 4:** `APIError.ResetAt`'s doc follows MADR D1's wording.
* **Step 5, tests:** the seven tests in the tables, in
  `llmprovider/internal/transport/reset_at_0072_test.go` and
  `llmprovider/api_error_reset_0072_test.go`. Each passes.
  * `TestAPIError_TextUnchanged_0072` zeroes `ResetAt` only, since `Limit`
    does not exist until Phase 2. Phase 2 makes it zero `Limit` too.
  * Its pinned text is
    `llmprovider: quota exhausted: openai HTTP 429 usage_limit_reached (retry-after 30s): The usage limit has been reached`.
* **The drift check:** before the new tests were written,
  `go test -count=1 ./llmprovider ./llmprovider/internal/transport/...`
  passed with every existing test unchanged.
* **Planted breaks:** `tools/plant.py`, one spec per test, each with
  `-run '^<test>$'`, on scratch clones. All 11 plants were caught, each by
  its own test. Their failure lines follow (times in the failure lines are
  left out):

  | Test | Plant | Failure line |
  | :--- | :--- | :--- |
  | `TestRetryAfterAt_0072` | a delta gives `now+d` as its instant | `reset_at_0072_test.go:34: delta seconds: RetryAfterAt = 30s, 2026-11-09 12:00:30 +0000 UTC; want 30s, 0001-01-01 00:00:00 +0000 UTC` |
  | `TestRetryAfterAt_0072` | the instant is returned in `time.Local` | `reset_at_0072_test.go:37: an HTTP-date ahead: instant in Local; want UTC` |
  | `TestRateLimitResetAt_0072` | the RFC 3339 instant is returned even when a duration header won | `reset_at_0072_test.go:72: a longer duration header: RateLimitResetAt = 5m0s, 2026-11-09 12:00:30 +0000 UTC; want 5m0s, 0001-01-01 00:00:00 +0000 UTC` (and the tie) |
  | `TestResetAt_ResetsAt_0072` | `ResetAt` left zero | `api_error_reset_0072_test.go:37: ResetAt = 0001-01-01 00:00:00 +0000 UTC; want resets_at's …` |
  | `TestResetAt_ResetsAt_0072` | `ResetAt` in `time.Local` | `api_error_reset_0072_test.go:40: ResetAt in Local; want UTC` |
  | `TestResetAt_PastKept_0072` | a past instant is zeroed | `api_error_reset_0072_test.go:53: ResetAt = 0001-01-01 00:00:00 +0000 UTC; want the past resets_at …, kept as stated` |
  | `TestResetAt_Order_0072` | the header is preferred to the body | `api_error_reset_0072_test.go:73: resets_at with an HTTP-date: ResetAt …; want the body's …` |
  | `TestResetAt_Order_0072` | the reset header is read on any status | `api_error_reset_0072_test.go:87: Claude's 400 with the reset header: ResetAt …; want none` |
  | `TestResetAt_OpenCodeMetadata_0072` | the top-level `metadata` is not read | `api_error_reset_0072_test.go:113: beside error: ResetAt 0001-01-01 00:00:00 +0000 UTC; want …` |
  | `TestResetAt_OpenCodeMetadata_0072` | the `ErrorInfo` detail is not read | `api_error_reset_0072_test.go:113: in an ErrorInfo: ResetAt 0001-01-01 00:00:00 +0000 UTC; want …` |
  | `TestAPIError_TextUnchanged_0072` | `Error()` appends the reset instant | `api_error_reset_0072_test.go:152: a ChatGPT usage limit: Error() = "llmprovider: quota exhausted: openai HTTP 429 usage_limit_reached (retry-after 30s) resets …: The usage limit has been reached"; want …` |

* **Pre-add:** `make pre-add-check` on the four Go files: "go-precheck: 4
  file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)", exit
  0.
* **Race:** `go test -race -count=3 ./llmprovider ./llmprovider/internal/transport/...`,
  exit 0.
* **The gate:** `python3 -B tools/gate.py`, started at load 9.8: "gate: 29
  of 29 passed".
  * `api-check`: "against v1.28.0, 0 incompatible change(s)".
  * `coverage-check`: `llmprovider` 98.4 % against 95.9 %,
    `internal/transport` 88.3 % against 80.0 %.
* **`tools/apidiff.py v1.28.0`** lists one compatible change:
  `./llmprovider.APIError.ResetAt: added`. `RetryAfterAt` and
  `RateLimitResetAt` are in an internal package, and are not listed.
* **Stop conditions:** none met. No existing test changed and no golden
  changed.
* This entry was added after the gate ran, so the record checks were run
  again before staging.

The owner committed Phase 1 (`8eaa21f`) and said to proceed.

### Phase 2 (2026-11-09)

* **Step 1:** `llmprovider/api_limit.go` holds `Limit`, `Quota` and their
  helpers. Each type's doc says it is safe for concurrent use only while no
  goroutine changes it (R51).
* **Step 2, the envelope:**
  * It gains `plan`, `window`, `limitName`, `creditsURL` and `quotas`.
  * The new members of `error` (`plan_type`, `limit_window_minutes`,
    `quotaValue`) and every metadata member are decoded as
    `json.RawMessage`. A value of an unexpected type then spoils none of the
    body's other fields.
  * Phase 1's `metadataResetAt` now takes the decoded metadata object.
  * `limitName` is read beside `error`, then inside it, as step 2 says, and
    not from an `ErrorInfo` detail.
* **Step 3, `limitOf`:** as written. Each bound is checked once, on the
  count, before it can overflow a `Duration`:
  * 1 to 527,040 minutes, in `windowMinutes`;
  * 1 to 8,784 hours, in `windowOfName`.

  A first draft also checked every result in a shared helper. That second
  check could not be shown to fail, since the first always rejected first.
  It was removed before the plants ran.
* **Step 4:** `BudgetLimitError` and `CreditLimitExceeded` appended to
  `opencodeQuotaTypes`, with a comment citing 0072-MADR D4 and O7.
* **Step 5, tests.** Each passes. Three differ from the table as written:
  * **`TestClassify_OpenCodeBudget_0072` and
    `TestRetry_OpenCodeBudgetNotResent_0072`** are in one file,
    `llmprovider/budget_0072_test.go`, not in `api_error_limit_0072_test.go`
    and `retry_0072_test.go`. `tools/redgreen.py --rev` copies only the
    files a spec names into a clone at that revision. A file that also used
    `Limit` could not build at `d62142c`. This file calls only API that
    exists at `v1.28.0`. The retry test uses the package's `stubProvider`,
    not `httptest`, since the classification is all it tests.
  * **`TestLimit_WindowBounds_0072`** also asserts that `"0 hours"` gives no
    `Window`, for the lower bound of `windowOfName`.
  * **`TestAPIError_TextUnchanged_0072`** now zeroes `Limit` as well as
    `ResetAt`, as Phase 1's entry said it would. It gained a plant of its
    own: `Error()` appends the limit's name.
* **The drift check:** with steps 1 to 4 in place and no new test written,
  `go test -count=1 ./llmprovider/...` passed, every existing test
  unchanged.
* **Red at the base:** `tools/redgreen.py --expect fail --rev d62142c`:
  "2 of 2 fail as expected":
  * `budget_0072_test.go:41: a budget: llmprovider: rate limited: opencode-zen/chat_completions HTTP 429 BudgetLimitError: A budget policy for this model has been reached.; want a terminal ErrQuotaExhausted`,
    and the same for Google's envelope;
  * `budget_0072_test.go:59: sent 3 times; want once: a budget is not retried`.

  `--expect pass` on the tree: "2 of 2 pass as expected".
* **Planted breaks:** one spec per test, as in Phase 1. All 13 plants were
  caught, each by its own test:

  | Test | Plant | Failure line |
  | :--- | :--- | :--- |
  | `TestLimit_ChatGPT_0072` | `limit_window_minutes` not read | `api_error_limit_0072_test.go:52: Limit = {Name: Window:0s Plan:plus …}; want Plan plus and Window 5h` |
  | `TestLimit_NeverInferred_0072` | `Window` set from `RetryAfter` when none is stated | `api_error_limit_0072_test.go:72: a Gemini per-day quota: Window = 52s; want none, since the service stated no length` (and the other two cases) |
  | `TestLimit_NeverInferred_0072` | `"weekly"` read as 7 days | `api_error_limit_0072_test.go:72: OpenCode Go's weekly limit: Window = 168h0m0s; want none, since the service stated no length` |
  | `TestLimit_WindowBounds_0072` | the minutes bound removed | `api_error_limit_0072_test.go:88: limit_window_minutes -5: Limit = &{… Window:-5m0s Plan:plus …}; want Plan only, with no Window` (and 600000) |
  | `TestLimit_WindowBounds_0072` | the hours bound removed | `api_error_limit_0072_test.go:94: limitName "9000 hours": Window = 9000h0m0s; want 0s` |
  | `TestLimit_OpenCodeGo_0072` | `limitName` not read | `api_error_limit_0072_test.go:103: … GoUsageLimitError (retry-after 2m0s): Go usage limit exceeded: Limit is nil; want one` |
  | `TestLimit_Gemini_0072` | `quotaMetric` not read | `api_error_limit_0072_test.go:127: Limit = {… Quotas:[{ID:GenerateRequestsPerMinutePerProjectPerModel-FreeTier Metric: Value:5} …]…}; want …` |
  | `TestLimit_Gemini_0072` | a numeric `quotaValue` dropped | `api_error_limit_0072_test.go:127: Limit = {Name: Window:0s Plan: PerDay:true Quotas:[{ID:GenerateRequestsPerMinutePerProjectPerModel-FreeTier Metric:generativelanguage.googleapis.com/generate_content_free_tier_requests Value:5} {ID:Generate` (the tool cuts a line at 240 characters, before the second quota's `Value`) |
  | `TestLimit_KiloCredits_0072` | the scheme check removed | `api_error_limit_0072_test.go:154: buyCreditsUrl "http://app.kilo.ai/credits": CreditsURL = "http://app.kilo.ai/credits"; want none` |
  | `TestLimit_KiloCredits_0072` | the query check removed | `api_error_limit_0072_test.go:154: buyCreditsUrl "https://app.kilo.ai/credits?token=abc": CreditsURL = "https://app.kilo.ai/credits?token=abc"; want none` |
  | `TestLimit_Redacted_0072` | `Plan` not passed through `redact.Field` | `api_error_limit_0072_test.go:173: plan_type: "plus\x1b[31méé…" (309 bytes); want no control character, at most 128 bytes, valid UTF-8` |
  | `TestLimit_Nil_0072` | `Limit` always allocated | `api_error_limit_0072_test.go:187: a 429 with only a message: Limit = {Name: Window:0s Plan: PerDay:false Quotas:[] CreditsURL:}; want nil` |
  | `TestAPIError_TextUnchanged_0072` | `Error()` appends the limit's name | `api_error_reset_0072_test.go:153: an OpenCode Go 429: Error() = "llmprovider: quota exhausted: opencode-go/chat_completions HTTP 429 GoUsageLimitError (retry-after 2m0s) 5 hour: Go usage limit exceeded"; want …` |

  An ellipsis (…) marks a part of a line left out here. The lines are cut at
  240 characters by `tools/plant.py`.

* **Pre-add:** `make pre-add-check` on the five Go files failed once, on
  golangci-lint's `errorlint`:
  `api_error_limit_0072_test.go:160:9: comparing with == will fail on wrapped errors`.
  The comparison was in a test helper. The test now uses `errors.Is`, and the
  helper is gone. The re-run: "go-precheck: 5 file(s) clean (gofmt,
  golangci-lint, go vet, go test, govulncheck)", exit 0.
  * The fix moved lines in the test file. So the plants were run again
    against the final file. All 13 were caught, and the table above holds
    that run's line numbers.
* **Race:** `go test -race -count=3 ./llmprovider ./llmprovider/internal/transport/...`,
  exit 0.
* **The gate:** `python3 -B tools/gate.py`, started at load 9.8: "gate: 29
  of 29 passed". `coverage-check`: `llmprovider` 98.3 % against 95.9 %.
* **`tools/apidiff.py v1.28.0`** lists four compatible changes:
  * `APIError.Limit`;
  * `APIError.ResetAt`;
  * `Limit`;
  * `Quota`.

  The tool names a new type, not its fields. The fields are the R55 table's.
* **Stop conditions:** none met. No existing test changed and no golden
  changed.

The owner committed Phase 2 (`e09e6ec`) and said to proceed.

### Phase 3 (2026-11-09)

#### Deviation 1 (2026-11-09): the Chat Completions wire test

* **Found:** Phase 3, step 4, says each per-wire test "calls only existing
  API" and is seen red at `d62142c`. The table's Chat Completions test,
  `TestEvents_GatewayErrorLink_0072`, asserted `Limit.CreditsURL`. `Limit`
  does not exist at `d62142c`, so the test could not build there to be seen
  red. The other four wire tests read only `RetryAfter` and the kind, which
  exist there.
* **Decided by the owner,** from three options (also offered: two tests, the
  link test proven by its plant alone beside a headers test; or the link test
  proven by its plant alone):
  * the test uses `error.headers`, as the Messages test does, and asserts
    `RetryAfter` 7 s;
  * it is renamed `TestEvents_GatewayErrorHeaders_0072`.

  It is seen red at `d62142c` and fails with its parser planted to pass
  `nil`. The table above is annotated, not rewritten.
* **Not tested in a stream:** Kilo's link inside a stream. Kilo's documented
  stream error carries no `metadata` (0072-MADR K3). The link is still read
  from an HTTP body, which `TestLimit_KiloCredits_0072` tests. In a stream it
  is read by the same parser, which `TestClassifyStreamFailureBody_*` covers.
* **No file is added to the phase's scope.** The MADR's decision and facts
  are unchanged.

#### Deviation 2 (2026-11-09): one file added, `internal/wire/wire.go`

* **Found:** step 3 has three parsers, generateContent's, Interactions' and
  Messages', read the payload's `error` member as `json.RawMessage`. The
  three reads are the same code.
* **Done:** `llmprovider/internal/wire/wire.go` gains `ErrorObject(payload)`,
  and the three parsers call it.
  * It is exported from an internal package, so it adds nothing to the
    module's API.
  * It adds a file to the phase's scope that the PLAN did not list.
* **Recorded, not asked:** it changes no behaviour step 3 describes. The
  owner is told in the handoff and may ask for the reads to be inlined.

#### Steps

* **Step 1, `ClassifyStreamFailureBody`:** as written.
  * Today's classification logic moved, unchanged, into an unexported
    `classifyStreamFailure(e, code, errType, message, env)`.
  * The `headers` member is decoded by `streamErrorHeaders`. A value may be a
    JSON string or number, read by `scalarText`, which is Phase 2's
    `quotaValue` renamed now that it has two uses.
  * The message delay is `textDelay`, over Codex's pattern.
  * **A consequence of joining the types:** a stream error object that
    carries a type the HTTP tables know, such as a Google `API_KEY_INVALID`
    reason, is now classified by it, as the same object is in an HTTP reply.
    No existing test covers such an object, and every existing test passes
    unchanged.
* **Step 2:** `ClassifyStreamFailure` calls `ClassifyStreamFailureBody(…, nil)`.
  Its doc names the text delay and says it returns a non-nil `*APIError`.
* **Step 3, the five parsers:**
  * Responses' `responseError` and Chat Completions' `gatewayError` each keep
    their object, through an `UnmarshalJSON` that decodes the fields and
    keeps the bytes.
  * The Responses `error` event passes its whole payload.
  * generateContent, Interactions and Messages pass `wire.ErrorObject(payload)`
    (Deviation 2).
* **Step 4, tests:** the five `ClassifyStreamFailureBody` tests in
  `llmprovider/api_error_stream_0072_test.go`, and the five wire tests, each
  in a `stream_error_0072_test.go`. All pass.
  * `TestClassifyStreamFailureBody_Text_0072` also asserts that "Try again in
    5 minutes" gives no delay. Codex's pattern reads seconds and
    milliseconds only.
  * Before its plant ran, `TestClassifyStreamFailure_Same_0072` was made to
    require both calls to have no `Limit` and no `ResetAt`. As first written
    it checked only the `nil` call's, so the plant, which changes the other
    call, could not have failed it.
* **The drift check:** with steps 1 to 3 in place and no new test written,
  `go test -count=1 ./llmprovider/...` passed, every existing test and G-wire
  golden unchanged.
* **Red at the base** (the five wire tests): `tools/redgreen.py --expect fail
  --rev d62142c`, "5 of 5 fail as expected":

  | Test | Failure line at `d62142c` |
  | :--- | :--- |
  | `responses.TestStream_FailedHeaders_0072` | `stream_error_0072_test.go:23: streamed: llmprovider: rate limited: openai stream rate_limit_exceeded: Slow down; want ErrRateLimited with RetryAfter 7s` (and `replied`) |
  | `generatecontent.TestEvents_PerDayQuota_0072` | `stream_error_0072_test.go:25: llmprovider: rate limited: gemini stream 429: You exceeded your current quota.; want ErrQuotaExhausted with RetryAfter 52s` |
  | `gemini.TestInteractionEvents_PerDayQuota_0072` | `stream_error_0072_test.go:24: llmprovider: rate limited: gemini stream 429: You exceeded your current quota.; want ErrQuotaExhausted with RetryAfter 52s` |
  | `messages.TestEvents_ErrorHeaders_0072` | `stream_error_0072_test.go:21: llmprovider: rate limited: claude stream rate_limit_error: slow down; want ErrRateLimited with RetryAfter 7s` |
  | `chatcompletions.TestEvents_GatewayErrorHeaders_0072` | `stream_error_0072_test.go:27: at the top: llmprovider: rate limited: kilo stream 429: Rate limited; want ErrRateLimited with RetryAfter 7s` (and `in the choice`) |

  `--expect pass` on the tree: "5 of 5 pass as expected".
* **Planted breaks:** one spec per test. All 11 were caught, each by its own
  test. The five "the parser passes nil" plants failed with the same lines as
  the red run above.

  | Test | Plant | Failure line |
  | :--- | :--- | :--- |
  | `TestClassifyStreamFailureBody_Headers_0072` | `headers` not read | `api_error_stream_0072_test.go:22: a delay in headers: RetryAfter 0s, ResetAt 0001-01-01 00:00:00 +0000 UTC; want 7s and none` (and `:28`, the HTTP-date) |
  | `TestClassifyStreamFailureBody_Text_0072` | the pattern removed | `api_error_stream_0072_test.go:43: "Rate limit reached. Please try again in 11.054s.": RetryAfter 0s; want 11.054s` (and the other two) |
  | `TestClassifyStreamFailureBody_Text_0072` | the text read for a quota too | `api_error_stream_0072_test.go:54: usage_limit_reached: RetryAfter 20s from its text; want none` |
  | `TestClassifyStreamFailureBody_UsageLimit_0072` | `raw` ignored | `api_error_stream_0072_test.go:70: llmprovider: quota exhausted: openai stream usage_limit_reached: The usage limit has been reached: ResetAt 0001-01-01 00:00:00 +0000 UTC, Limit <nil>; want …` |
  | `TestClassifyStreamFailureBody_Gemini_0072` | the envelope's per-day flag not passed to the status path | `api_error_stream_0072_test.go:84: llmprovider: rate limited: gemini stream 429 (retry-after 52s): You exceeded your current quota.: RetryAfter 52s, Limit &{… PerDay:false …}; want …` |
  | `TestClassifyStreamFailure_Same_0072` | `ClassifyStreamFailure` reads a fixed non-nil `raw` | `api_error_stream_0072_test.go:111: {provider:openai code:rate_limit_exceeded errType: message:slow down}: ClassifyStreamFailure llmprovider: rate limited: openai stream rate_limit_exceeded: slow down …` (and the other ten cases) |
  | each of the five wire tests | the parser passes `nil` | as the red run's line for that test |

* **Pre-add:** `make pre-add-check` on the fourteen Go files failed once,
  with two lint findings:
  * `revive`'s `confusing-naming`: the unexported helper was first named
    `classifyStreamFailure`, which differs from `ClassifyStreamFailure` only
    by case. It is now `setStreamKind`.
  * `errorlint`: `TestClassifyStreamFailure_Same_0072` compared two kinds
    with `!=`. It now requires `errors.Is` both ways.

  The re-run: "go-precheck: 14 file(s) clean (gofmt, golangci-lint, go vet,
  go test, govulncheck)", exit 0. The plants were run again against the
  fixed files, and all 11 were caught. The table above holds that run's line
  numbers.
* **Race:** `go test -race -count=3` over `./llmprovider`,
  `./llmprovider/internal/transport/...`, `./llmprovider/internal/wire/...`
  and `./llmprovider/providers/gemini/...`, exit 0, before the lint fixes.
  The gate's `test-race-cover` ran again after them.

PHASE3_GATE_PENDING
