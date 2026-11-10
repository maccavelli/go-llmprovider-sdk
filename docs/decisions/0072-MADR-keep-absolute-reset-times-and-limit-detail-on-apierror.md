---
status: accepted
date: 2026-11-09
decision-makers: repository owner
consulted: this module at d62142c (v1.28.0 and one records commit); GitHub issue 13 of this repository, opened 2026-11-09; the OpenAI Codex CLI source at 57e64682f8; the OpenCode source at 4ce137b39e, and its gateway as last published at d21b0790a6; the Kilo Code source at 58072bd696; gobble-cli at 74d7f66879 and its issue 14; magic-cli-remote at 119a03383b and its 0243-MADR
informed: anyone who reads an *APIError's RetryAfter, Code or Message to tell a user which limit a request hit; gobble-cli and magic-cli-remote
---

<!-- markdownlint-disable MD013 -->

# Keep a Limit's Absolute Reset Time, and What the Service Said About the Limit, on APIError

## Context and Problem Statement

GitHub issue 13 asks the SDK to keep what a provider says about a usage
limit, so that a consumer can tell a user exactly which limit a call hit and
when it lifts. The limit can be:

* a rolling window, such as 5 hours;
* a daily, weekly or monthly quota;
* a free tier;
* a credit balance;
* plain throttling.

The consumers are gobble-cli (its issue 14) and magic-cli-remote
(`0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md`).
Both state one rule that this record adopts: **a window is never inferred from
how long the wait is** (M1). A 4-hour wait does not make a 5-hour window.

`*APIError` carries the provider's `Code`, a redacted `Message`, and a
relative `RetryAfter` (C1). It is the only structured error (R24, C27). What a
provider says beyond those is dropped, or turned into a delay.

### What the code does at d62142c

Every site the issue cites holds. The table adds the sites the issue does not
name (rows 7 to 10).

| # | What is lost | Where | Claims |
| :--- | :--- | :--- | :--- |
| 1 | A body's `resets_at` (Unix seconds) becomes a delay; the instant is lost, and a past instant leaves nothing | `ClassifyHTTPError` | C3, C7 |
| 2 | An HTTP-date `Retry-After` becomes a delay | `transport.ParseRetryAfter` | C5 |
| 3 | Anthropic's RFC 3339 reset headers become a delay | `transport.RateLimitReset` | C4, C6 |
| 4 | ChatGPT's `plan_type` is in a fixture body but never read | the envelope parser | C12 |
| 5 | Gemini's `QuotaFailure`: only `quotaId` is read, to set a private per-day flag | the envelope parser | C8, C9, C10, C17 |
| 6 | A failure inside a 200 stream gets no `RetryAfter`: the exported signature has no room for one, and none of the five stream parsers reads a delay | `ClassifyStreamFailure` and its callers | C14, C15, C18 to C24 |
| 7 | A numeric `error.code`, as Kilo sends it, is dropped from `Code` | the envelope parser | C11, K2 |
| 8 | Error metadata is not parsed: OpenCode's top-level `metadata`, Kilo's `error.metadata` | the envelope parser | O1, O4, K1 |
| 9 | Gemini's details inside a stream are dropped, so a per-day quota in a stream is a retryable rate limit | `generatecontent`, `interactions` | C20, C21, C22 |
| 10 | OpenCode's `BudgetLimitError` (429) is a retryable rate limit, and `CreditLimitExceeded` is not in the quota table | `opencodeQuotaTypes` | C16, O3, O5 |

Every generation request passes through one function, `wire.open`, where a
non-2xx reply is classified (C25). So one change to `ClassifyHTTPError`
reaches every provider. Row 7 is recorded but not changed (D2).

### What the providers send

Read from each reference client's source. "Proven" means the client parses
the field, so a server sends it. "Inferred" means only a test fixture or
documentation shows it.

#### ChatGPT (Codex, the reference client for the ChatGPT backend)

* **The 429 usage-limit body is proven** (X1 to X5). It is recognised by
  `error.type == "usage_limit_reached"`, and carries:
  * `plan_type`;
  * `resets_at` in Unix seconds, which Codex keeps as an absolute instant (X2);
  * `limit_window_minutes`, documented as "Server-selected window responsible
    for the limit, in minutes" (X6).

  Codex shows the user the reset as a local time ("Try again at 3:40 PM").
* **Window headers are proven, on success and on a 429** (X7, X8, X10):
  * `x-codex-{primary,secondary}-{used-percent,window-minutes,reset-at}`,
    where `reset-at` is absolute;
  * credit headers;
  * `x-codex-active-limit`, which names the header family that was hit.

  Codex derives each window's label ("5h", "weekly") from the server's
  `window-minutes` (X9), never from the reset distance.
* **Delays inside a stream are proven** (X11 to X13):
  * a `response.failed` error's `headers` object gives `Retry-After`;
  * for `rate_limit_exceeded` and `slow_down` with no header, the delay is
    read from the message with `(?i)try again in\s*(\d+(?:\.\d+)?)\s*(s|ms|seconds?)`.

  Codex's fixture message is "Please try again in 11.054s."
* **Usage limits inside a stream are only inferred.** `response.failed`
  decodes `plan_type` and `resets_at` but never uses them (X14).
* **`resets_in_seconds` is fixture-only** in Codex. No production code reads
  it.

#### OpenCode Zen, Go and Black

* **The gateway's code is no longer public.**
  * Commit `83f2b6f67a` (2026-10-11) made the Zen routes a proxy to a private
    service.
  * Commit `3efbdad965` (2026-11-06) removed the console package.
  * The last public logic is at `d21b0790a6`, and it shows what OpenCode sent
    up to 2026-10-11 (`packages/console/app/src/routes/zen/util/handler.ts:494-532`):
    * `CreditsError`, `MonthlyLimitError`, `UserLimitError` and `ModelError`:
      401, `{type:"error", error:{type, message}}`, and **no headers**.
    * `RateLimitError`, `FreeUsageLimitError`, `GoUsageLimitError` and
      `BlackUsageLimitError`: 429, `retry-after` in delta seconds, and a
      **top-level** `metadata` beside `error`: `{workspace, limitName}` for Go
      (`limitName` is `"5 hour"`, `"weekly"` or `"monthly"`), and `{}` for the
      others.
    * Go's messages name the window ("5-hour usage limit reached. Resets in
      …", "Weekly …", "Monthly …"). Black's do not ("Subscription quota
      exceeded. Retry in …"). The reset was sent only as relative text.
* **Today's service, inferred from the client's fixtures at HEAD:**
  * `GoUsageLimitError`: 429, `metadata.limitName`, `retry-after` in delta
    seconds (O1, O2).
  * New codes:
    * `CreditLimitExceeded`, at 402 (O3);
    * `BudgetLimitError`, at 429, not retryable, with
      `metadata: {coverage: "models" | "all", resetsAt: <RFC 3339>}` (O4,
      O7). In Google's envelope the same metadata sits in a
      `google.rpc.ErrorInfo` detail whose `reason` is `BudgetLimitError` (O5).
  * The client's own comment: "OpenCode Console names it `resetsAt` in every
    protocol's error envelope" (O6).
* **Recorded live by magic-cli-remote** (its `0073-MADR` and `0138-MADR`): Go
  bodies reading "Weekly usage limit reached. Resets in 4 days." and "Monthly
  usage limit reached. Resets in 4 days". So Go's messages named their window
  as late as those captures.

#### Kilo

* **The gateway's code is not in the Kilo Code repository.**
* **Its public documentation shows a 402 for an empty balance** (K1, K2):
  `{"error": {"message": …, "code": 402, "metadata": {"buyCreditsUrl": "https://app.kilo.ai/credits"}}}`.
  * `metadata` sits **inside** `error`, unlike OpenCode's.
  * `code` is a number.
* **No client code reads `buyCreditsUrl`.** The client hard-codes the same URL.
* **No error body or stream event carries a reset time or a window** (K3). The
  documentation states two limits in prose only: a daily member limit that
  resets at midnight UTC, and 200 requests per hour per IP for free models.

#### Gemini

* **The `QuotaFailure` and `RetryInfo` shapes are documented, never captured**
  (0026-MADR F12 accepted that risk). Each `violations[]` entry has:
  * `quotaId`;
  * `quotaMetric`;
  * `quotaValue`, a string in the JSON form of an `int64`;
  * `quotaDimensions`.
* `TestLive_GeminiRateLimitShape` checks the shape against a real 429 when the
  owner runs it.

#### Claude, OpenAI, Grok, Hugging Face

* **0028-PLAN's T2 measured these headers on a billed success:**
  * the `-limit-` and `-remaining-` headers on OpenAI, Claude, Grok and
    Hugging Face;
  * the reset headers on OpenAI, Claude and Hugging Face;
  * Claude's resets as RFC 3339 instants.
* No body field names a window.

### What the consumers need

* **magic-cli-remote 0243-MADR's requirement 6** asks the SDK for (M2):
  * an absolute `ResetAt` from `resets_at` and an HTTP-date;
  * ChatGPT's `plan_type`;
  * Gemini's `quotaId`, `quotaMetric`, `quotaValue` and the per-day flag;
  * Kilo's `metadata`;
  * `RetryAfter` on stream failures.

  It rules out remaining quota before it runs out: "no agent reports it" (M3).
* **gobble-cli's issue 14** lists the same dependencies. gobble copies
  `RetryAfter` from `*APIError` into its own error (G1), and would copy the new
  fields the same way. gobble owns its scope table (`free_tier`, `monthly`,
  `credits`, `plan`, `daily`, `rate`). It notes that `window` and `weekly`
  "need a source that states the window, which the SDK does not have yet."
* **Both parse `APIError.Error()`'s text today.** magic-cli-remote's
  classifier, uncommitted in its working tree at `119a03383b`, matches
  `<provider> HTTP <status> <code> (retry-after <d>):`, followed by the
  message (C13). `Error()`'s text
  is therefore a contract until gobble sends the structured fields.
* **Neither asks** to show a plan name or a buy-credits link, or for
  `quotaDimensions`. The issue lists them, and so do 0243-MADR's
  dependencies, except `quotaDimensions`.

### Code claims

<!-- claims: self=d62142c codex=57e64682f8 opencode=4ce137b39e kilocode=58072bd696 gobble-cli=74d7f66879 magic-cli-remote=119a03383b -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `RetryAfter` is a delay | `llmprovider/api_error.go:114` | `RetryAfter time.Duration` |
| C2 | A header's delay is read first | `llmprovider/api_error.go:251` | `RetryAfter: transport.RetryAfter(resp.Header),` |
| C3 | `resets_at` becomes a delay | `llmprovider/api_error.go:257` | `if d := time.Until(time.Unix(envelope.resetsAt, 0)); d > 0 {` |
| C4 | Reset headers become a delay | `llmprovider/api_error.go:270` | `e.RetryAfter = transport.RateLimitReset(resp.Header, time.Now())` |
| C5 | An HTTP-date becomes a delay | `llmprovider/internal/transport/transport.go:235` | `if d := time.Until(t); d > 0 {` |
| C6 | Claude's resets are RFC 3339 instants | `llmprovider/internal/transport/transport.go:273` | `if at, err := time.Parse(time.RFC3339, strings.TrimSpace(h.Get(name))); err == nil {` |
| C7 | The envelope keeps only `resets_at` | `llmprovider/api_error.go:504` | `resetsAt int64 // usage_limit_reached's reset time, Unix seconds, or 0` |
| C8 | The per-day flag is private | `llmprovider/api_error.go:508` | `perDayQuota bool` |
| C9 | Only `quotaId` is read | `llmprovider/api_error.go:583` | `QuotaID string` |
| C10 | Per-day is read from the id | `llmprovider/api_error.go:596` | `strings.Contains(v.QuotaID, "PerDay")` |
| C11 | A numeric code is dropped | `llmprovider/api_error.go:599` | `add(jsonString(inner.Code), inner.Type, inner.Status)` |
| C12 | `plan_type` is in a fixture | `llmprovider/chatgpt_errors_test.go:35` | `"plan_type":"plus"` |
| C13 | `Error()` prints the delay | `llmprovider/api_error.go:144` | `where = append(where, fmt.Sprintf("(retry-after %s)", e.RetryAfter))` |
| C14 | The stream classifier's signature | `llmprovider/api_error.go:301` | `func ClassifyStreamFailure(provider, code, errType, message string) error {` |
| C15 | A stream failure gets no delay | `llmprovider/api_error.go:308` | `e := &APIError{Provider: provider, Code: boundCode(env.errType()),` |
| C16 | OpenCode's quota codes | `llmprovider/api_error.go:383` | `opencodeQuotaTypes     = []string{"FreeUsageLimitError", "GoUsageLimitError", "BlackUsageLimitError", "CreditsError", "MonthlyLimitError", "UserLimitError"}` |
| C17 | A Gemini per-day 429 is quota | `llmprovider/api_error.go:445` | `case service == string(ProviderGemini) && status == http.StatusTooManyRequests && env.perDayQuota:` |
| C18 | A failed response's error has three fields | `llmprovider/internal/wire/responses/responses.go:595` | `type responseError struct {` |
| C19 | The Responses stream's error event | `llmprovider/internal/wire/responses/responses.go:664` | `llmprovider.ClassifyStreamFailure(provider, event.Code, "", event.Message)` |
| C20 | Google's stream error has no details | `llmprovider/internal/wire/generatecontent/generatecontent.go:220` | `type replyError struct {` |
| C21 | generateContent's stream failure | `llmprovider/internal/wire/generatecontent/generatecontent.go:307` | `llmprovider.ClassifyStreamFailure(provider, strconv.Itoa(e.Code), e.Status, e.Message)` |
| C22 | The Interactions stream's failure | `llmprovider/providers/gemini/interactions.go:276` | `llmprovider.ClassifyStreamFailure(s.provider, strconv.Itoa(e.Code), e.Status, e.Message)` |
| C23 | The Messages stream's failure | `llmprovider/internal/wire/messages/messages.go:524` | `llmprovider.ClassifyStreamFailure(provider, "", ev.Error.Type, ev.Error.Message)` |
| C24 | The Chat Completions stream's failure | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:879` | `return llmprovider.ClassifyStreamFailure(provider, code, e.Type, e.Message)` |
| C25 | Every reply is classified in one place | `llmprovider/internal/wire/post.go:181` | `if err := llmprovider.ClassifyHTTPError(c.Provider, resp); err != nil {` |
| C26 | A retry notice says the wait was the server's | `llmprovider/retry.go:47` | `ServerDirected bool` |
| C27 | R24 | `docs/guides/api-standards.md:207` | `is the only structured error.` |
| C28 | R55 | `docs/guides/api-standards.md:360` | `A new export is justified.` |
| X1 | Codex recognises the usage limit by type | `codex:codex-rs/codex-api/src/api_bridge.rs:210` | `Some("usage_limit_reached")` |
| X2 | and keeps `resets_at` absolute | `codex:codex-rs/codex-api/src/api_bridge.rs:227` | `DateTime::<Utc>::from_timestamp(seconds, 0)` |
| X3 | The body's `plan_type` | `codex:codex-rs/codex-api/src/api_bridge.rs:371` | `plan_type: Option<PlanType>,` |
| X4 | The body's `resets_at` | `codex:codex-rs/codex-api/src/api_bridge.rs:372` | `resets_at: Option<i64>,` |
| X5 | The body's `limit_window_minutes` | `codex:codex-rs/codex-api/src/api_bridge.rs:373` | `limit_window_minutes: Option<Value>,` |
| X6 | The window is the server's | `codex:codex-rs/protocol/src/error.rs:781` | `Server-selected window responsible for the limit, in minutes.` |
| X7 | A window-length header | `codex:codex-rs/codex-api/src/rate_limits.rs:71` | `-primary-window-minutes` |
| X8 | An absolute reset header | `codex:codex-rs/codex-api/src/rate_limits.rs:72` | `-primary-reset-at` |
| X9 | A label comes from the window's minutes | `codex:codex-rs/tui/src/chatwidget/rate_limits.rs:120` | `Some("5h".to_string())` |
| X10 | Window headers are read on a success | `codex:codex-rs/codex-api/src/sse/responses.rs:45` | `parse_all_rate_limits(&stream_response.headers)` |
| X11 | A stream failure's `headers` give `Retry-After` | `codex:codex-rs/codex-api/src/sse/responses_error.rs:31` | `RetryAfter::from_headers(&json_headers_to_http_headers(headers))` |
| X12 | A rate limit's delay is read from its text | `codex:codex-rs/codex-api/src/sse/responses_error.rs:95` | `try_parse_retry_delay(&error)` |
| X13 | by this pattern | `codex:codex-rs/codex-api/src/sse/responses_error.rs:144` | `try again in` |
| X14 | A stream failure's `resets_at` is decoded, unused | `codex:codex-rs/codex-api/src/sse/responses_error.rs:21` | `resets_at: Option<i64>,` |
| O1 | Go's 429 names its limit in metadata | `opencode:packages/core/test/session-error.test.ts:105` | `limitName: "5 hour"` |
| O2 | with a delay in seconds | `opencode:packages/core/test/session-error.test.ts:107` | `{ "retry-after": "120" }` |
| O3 | `CreditLimitExceeded` | `opencode:packages/core/test/session-error.test.ts:124` | `type: "CreditLimitExceeded"` |
| O4 | A budget names its coverage and reset | `opencode:packages/core/test/session-error.test.ts:149` | `coverage: "models", resetsAt: "2026-11-01T00:00:00.000Z"` |
| O5 | In Google's envelope, as a reason | `opencode:packages/core/test/session-error.test.ts:169` | `reason: "BudgetLimitError",` |
| O6 | `resetsAt` in every envelope | `opencode:packages/tui/src/util/usage-limits.ts:94` | `OpenCode Console names it` |
| O7 | A budget is not retried | `opencode:packages/core/test/session-error.test.ts:152` | `expect(SessionRunnerRetry.isRetryable(model)).toBe(false)` |
| K1 | Kilo's 402 carries a link | `kilocode:packages/kilo-docs/pages/gateway/usage-and-billing.md:49` | `"buyCreditsUrl": "https://app.kilo.ai/credits"` |
| K2 | with a numeric code | `kilocode:packages/kilo-docs/pages/gateway/usage-and-billing.md:47` | `"code": 402,` |
| K3 | Kilo's stream error has no delay | `kilocode:packages/kilo-docs/pages/gateway/streaming.md:186` | `"finish_reason":"error"` |
| G1 | gobble copies `RetryAfter` | `gobble-cli:llm/provider/provider.go:689` | `return &llm.APIError{Status: ae.Status, RetryAfter: ae.RetryAfter, Err: wrapped}` |
| M1 | The consumers' rule | `magic-cli-remote:docs/decisions/0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md:151` | `A window is never inferred from how long the wait is.` |
| M2 | Its dependencies on this SDK | `magic-cli-remote:docs/decisions/0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md:279` | `Dependencies on go-llmprovider-sdk` |
| M3 | Remaining quota is out of its scope | `magic-cli-remote:docs/decisions/0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md:192` | `Showing remaining quota before it runs out: no agent reports it` |

## Decision Drivers

* **Stated, never inferred.** A field holds only what the service said:
  * no window from a wait's length;
  * no scope from a code;
  * no reset from a delay.
* **Compatible.** The change only adds API (R48).
  * `Retryable`, `Unwrap`, `Code` and the request on the wire are unchanged
    (R45). `RetryAfter` is unchanged wherever it is set today. It is newly
    set only where it was 0: OpenCode's `metadata.resetsAt`, and a stream
    failure that states a delay.
  * `Error()`'s format is unchanged, because both consumers parse it. It
    prints neither `ResetAt` nor `Limit`. A `RetryAfter` newly set is printed
    as any `RetryAfter` is, `(retry-after <d>)`, which the consumers already
    read.
* **The API standards.**
  * `*APIError` stays the only structured error (R24).
  * Wire knowledge stays in the SDK (R3).
  * Each new export has a call site (R55).
  * Text from a service is redacted and bounded (R35).
  * Each new type says whether it is safe for concurrent use (R51).
* **One parser.** A stream failure is read by the parser that reads an HTTP
  error body, so that a field is learnt once.
* **No identifiers.** An account's or workspace's id is not exposed.
  OpenCode's `metadata.workspace` and the Go console URL carry one.

## Considered Options

* **A. `ResetAt` and a `Limit` on `*APIError`**, filled by the one envelope
  parser, from HTTP replies and stream failures alike. Each field is a fact
  the service stated.
* **B. Flat fields on `*APIError`:** `ResetAt`, `Plan`, `Window`, `QuotaID`
  and the rest, each a field of the error.
* **C. The raw body:** an `APIError.Body json.RawMessage`, or a
  `map[string]any` of the error's metadata, for consumers to parse.
* **D. A scope enum** (`rate`, `window`, `daily`, `weekly`, `monthly`,
  `credits`, `free_tier`, `plan`), decided in the SDK from codes and fields.
* **E. A typed error per limit kind,** such as `*UsageLimitError`, beside
  `*APIError`.
* **F. Leave the SDK as it is:** consumers parse `Error()`'s text.

## Decision Outcome

Chosen option: "A", because it carries every fact the consumers asked for
under one nil-able field. It keeps `*APIError` the only structured error, and
keeps each provider's vocabulary inside the SDK. It adds nothing a service did
not state, and it changes no existing field, method or text.

### D1. `APIError.ResetAt time.Time`

* **What it is.** The instant the service named for the limit to lift, in
  UTC. It is the zero time when the service named only a delay, or nothing.
* **Sources, most specific first.** The first present wins:
  1. **The body's reset:**
     * ChatGPT's `error.resets_at`, in Unix seconds (X4);
     * OpenCode's `metadata.resetsAt`, RFC 3339, read at the top level, in
       `error.metadata`, or in a `google.rpc.ErrorInfo` detail's `metadata`
       (O4, O5, O6).
  2. **An HTTP-date `Retry-After`,** when it is the header `RetryAfter` read.
     `retry-after-ms` and delta seconds name no instant.
  3. **On a 429, a reset header naming an instant:** the Anthropic RFC 3339
     header that gave `RetryAfter`, when that header won (C4, C6). A Go
     duration header (`x-ratelimit-reset-*`) names no instant.
* **A past instant is kept as stated** (Q5). `RetryAfter` stays 0 for it, as
  today. Its doc says the instant may have passed, and why: the service's
  clock, or a late reply.
* **`RetryAfter` is unchanged** in every case it is set today (C2 to C5), with
  one addition. OpenCode's `metadata.resetsAt` sets it when no header did,
  exactly as `resets_at` does (C3). It is the same rule over a second spelling.
* **`Error()` is unchanged.** It does not print `ResetAt` (C13).

### D2. `APIError.Limit *Limit`

`Limit` describes the limit a request hit, as the service stated it. It is
nil when the service stated nothing beyond its code. Its fields:

| Field | Type | Set from | Evidence |
| :--- | :--- | :--- | :--- |
| `Name` | `string` | OpenCode's `metadata.limitName` (`"5 hour"`, `"weekly"`, `"monthly"`) | O1 |
| `Window` | `time.Duration` | ChatGPT's `error.limit_window_minutes`; OpenCode's `limitName` when it reads `<n> hour` or `<n> hours` | X5, X6, O1 |
| `Plan` | `string` | ChatGPT's `error.plan_type` (`"plus"`) | X3, C12 |
| `PerDay` | `bool` | a Gemini `QuotaFailure` whose `quotaId` names `PerDay`: today's private flag (C8, C10) | 0026-MADR F12 |
| `Quotas` | `[]Quota` | Gemini's `QuotaFailure.violations[]` | 0026-MADR F12 |
| `CreditsURL` | `string` | Kilo's `error.metadata.buyCreditsUrl` | K1 |

`Quota` has three fields:

* `ID`, from `quotaId`;
* `Metric`, from `quotaMetric`;
* `Value`, from `quotaValue`, kept as the string the service sent. A JSON
  number is written in decimal.

#### The rules on those fields

* **`Window` is a length the service stated as a number.** It is never
  derived from `RetryAfter`, `ResetAt`, a code, or a word such as "weekly".
  * `"weekly"` and `"monthly"` stay in `Name`, where a consumer's table maps
    them.
  * A length outside 1 minute to 366 days is ignored.
* **Every string is redacted and bounded** as `Code` is (`redact.Field`):
  control characters stripped, 128 bytes at most (R35).
* **`CreditsURL` is kept only if all of these hold:**
  * it parses as an absolute `https` URL;
  * it has no user information, query or fragment;
  * `redact.String` leaves it unchanged.

  Otherwise it is empty. A link with a token in it is dropped, not shown.
* **What is not exposed:**
  * OpenCode's `metadata.workspace`, an account identifier;
  * OpenCode's `metadata.coverage` (`"models"` or `"all"`, O4) and Gemini's
    `quotaDimensions`, which no consumer asks for (R55). Either can be added
    later without breaking anyone;
  * any other metadata key.
* **A numeric `error.code`, as Kilo sends it, stays out of `Code`** (row 7
  is observed, not changed). The number repeats the HTTP status, which
  `Status` already holds. Putting it in `Code` would change `Code` and
  `Error()`'s text for errors that exist today.
* **`Limit` is set on any kind,** not only on rate limits. Kilo's link arrives
  with a 402, which is `ErrQuotaExhausted`.

### D3. A stream failure is read as an error body

* **The new function.**
  `ClassifyStreamFailureBody(provider, code, errType, message string, raw []byte) error`
  classifies as `ClassifyStreamFailure` does (C14). It also reads `raw`, the
  stream's error object as it came, with the HTTP body's envelope parser, and
  sets `RetryAfter`, `ResetAt` and `Limit` from the same fields D1 and D2 name.
* **`ClassifyStreamFailure` keeps its signature.** It becomes
  `ClassifyStreamFailureBody(provider, code, errType, message, nil)`. Its
  result changes in one case only: a rate limit whose message states a delay
  now carries that delay in `RetryAfter`, by source 3 below. Its kind,
  `Retryable` and `Code` are unchanged.
* **Sources of a delay inside a stream,** most specific first:
  1. the error's `headers` object, read as HTTP headers (X11);
  2. the body's reset (`resets_at`, `metadata.resetsAt`) or a Google
     `RetryInfo`;
  3. for a failure classified `ErrRateLimited` and not `ErrQuotaExhausted`,
     with no delay from (1) or (2): the message, read with Codex's pattern
     (X12, X13). A delay in the text is one the service stated.
* **A Gemini per-day quota in a stream** becomes `ErrQuotaExhausted`, as in
  an HTTP reply (C17): the details now reach the classifier.
* **Every stream parser passes its error object:**
  * Responses' `error` event, and `response.failed`'s `response.error`;
  * generateContent's `error`;
  * Interactions' `error`;
  * Messages' `error`;
  * Chat Completions' gateway `error`.

  A wire format with no such field passes what it has. The parser ignores
  what it does not know.

### D4. OpenCode's two new codes are quota

* `BudgetLimitError` and `CreditLimitExceeded` join `opencodeQuotaTypes` (C16).
* So a budget's 429 is `ErrQuotaExhausted` and not retryable, as OpenCode's
  own client treats it (O7). Today it is `ErrRateLimited`, and `WithRetry`
  waits and resends it.
* In Google's envelope the code is a detail's `reason`, which the parser
  already adds as a type, so the same row matches (O5).
* This amends 0012-MADR §1.1's table.

### D5. Requirement 5 is not built

* The rate-limit `limit` and `remaining` headers of a successful reply are not
  added to `Response`.
* No consumer asks for them: 0243-MADR rules them out (M3), and gobble's
  issue 14 does not ask (R55). OpenCode strips them, and Gemini, Together and
  Kilo send none (0028-PLAN T2).
* A later record adds them once a caller needs them. The PLAN's measurement
  phase records which headers ChatGPT sends, for that record.

### D6. Tests, docs and the release

* **Tests.** Each is seen to fail on a planted break, on a scratch clone:
  * D1's sources and their order;
  * D2's fields, with the "never inferred" case: a 429 with a 4-hour delay
    and no stated window has a zero `Window`;
  * the redaction and the URL rule;
  * D3 on each of the five stream parsers;
  * D4's classification;
  * a test that `Error()` is byte-for-byte the same for an error with and
    without `ResetAt` and `Limit`, with one result pinned as a literal.

  Every existing test passes unchanged.
* **Fixtures.** Built from the shapes this record cites, and named as
  synthetic. None is recorded live (see Q-risk below).
* **Docs:**
  * R24's text, by an amendment to 0015-MADR D7;
  * 0012-MADR §1.1's table, for D4;
  * a note on 0026-MADR F12 that its flag is now `Limit.PerDay`;
  * `docs/architecture.md`;
  * the `APIError` rows of `docs/guides/migrating-from-mcplib.md`;
  * the changelog;
  * an `Example` showing a usage limit read from an `*APIError`.
* **The release** is a minor one (Q7).

### Consequences

* Good, because a consumer reads which limit was hit, and when it lifts, from
  fields rather than from `Error()`'s text.
* Good, because the SDK keeps the instant the service named, so a consumer's
  reset no longer drifts with its own clock.
* Good, because an OpenCode budget is no longer retried, and a Gemini per-day
  quota in a stream is no longer retried.
* Good, because a stream failure gains the delay Codex has read for its own
  users all along.
* Neutral, because a consumer still owns its table from `Code` and `Name` to a
  scope (Q2). Both have one today.
* Neutral, because OpenCode's live service is private. Its shapes are taken
  from its client's fixtures. A difference surfaces as a missing field, never
  a wrong one: each field is read only when it is present and well formed.
* Bad, because 14 exported names are added, under the compatibility promise
  (R48): two fields of `*APIError`, `Limit` and its six fields, `Quota` and
  its three, and `ClassifyStreamFailureBody`.
* Bad, because the Gemini, Kilo and OpenCode fixtures are synthetic. A field
  the live service spells differently is found only by a live run or a
  caller, as 0026-MADR F12 accepted for Gemini.

### Confirmation

* The PLAN's tests pass, each seen to fail on its planted break.
* `apidiff` against `v1.28.0` lists exactly D1 to D3's additions.
* `make docs-check` is clean: each new type says it is safe for concurrent use
  while unchanged, and `ClassifyStreamFailureBody` names the kind it returns.
* `tools/claims.py` passes on this record.
* The `Error()` test holds, with its text pinned.

## Pros and Cons of the Options

### A. `ResetAt` and a `Limit`

* Good, because the limit's facts sit together, and a nil `Limit` says "the
  service stated nothing".
* Good, because it answers 0243-MADR's requirement 6 and issue 13's
  requirements 1 to 4 in full.
* Neutral, because `Limit` holds provider-specific fields side by side. Each is
  zero for a provider that does not send it.
* Bad, because it adds two types.

### B. Flat fields on `*APIError`

* Good, because there are no new types.
* Bad, because `*APIError` gains eight fields, most zero on most errors.
* Bad, because a consumer cannot test once whether anything was stated.

### C. The raw body

* Good, because nothing is lost.
* Bad, because each consumer re-learns each provider's envelope. That is the
  wire knowledge R3 keeps inside the SDK.
* Bad, because a raw body escapes redaction (R35). It may hold a workspace id,
  or a key echoed back.

### D. A scope enum

* Good, because a consumer would need no table.
* Bad, because a scope from a code is the SDK's inference, not the service's
  statement, and the consumers chose to own that table.
* Bad, because a closed set would need a record for each new provider code.

### E. A typed error per limit kind

* Good, because `errors.As` finds it.
* Bad, because it breaks R24: two structured errors, and every existing
  `errors.As(*APIError)` misses the new one.

### F. Leave it

* Good, because nothing changes.
* Bad, because consumers keep parsing a text that drops the instant, the plan
  and the window, and a stream failure has no delay at all.

## Owner questions

*Answered by the owner on 2026-11-09, with the record's acceptance: each
question as recommended. The answers were chosen from three options each:*

* *Q1, `ResetAt` and a `*Limit` (offered also: flat fields on `APIError`; or
  `ResetAt` alone now);*
* *Q2, stated facts only (also: a scope enum now; or a scope in a later
  additive record);*
* *Q3, requirement 5 deferred to a later record (also: built in this record;
  or rejected outright);*
* *Q4, both OpenCode codes are quota (also: `BudgetLimitError` only; or
  neither, in a separate record);*
* *Q5, a past reset kept as stated (also: zeroed when past; or kept, with a
  helper method);*
* *Q6, `CreditsURL` only, vetted (also: a map of every key; or no Kilo field
  yet);*
* *Q7, `v1.29.0` (also: `v1.28.1`; or no release yet);*
* *Q-risk, accepted, with Phase 5 measuring (also: block the tag on a Gemini
  live run; or block it on a live capture for every provider).*

Each has a recommendation. The PLAN follows the recommendations, and is
amended before Phase 1 if an answer differs.

* **Q1. The shape.** Recommended: A, `ResetAt` on `*APIError` and the rest in
  `*Limit`. `ResetAt` sits beside `RetryAfter` because it is general: Claude's
  headers and an HTTP-date give one with no `Limit`. Alternative: B.
* **Q2. A scope in the SDK.** Recommended: no. Expose the stated facts only.
  Both consumers own a scope table (gobble's issue 14, requirement 2;
  0243-MADR D3), and a scope from a code is an inference. Alternative: D, as
  a later, additive record.
* **Q3. Requirement 5, remaining-quota headers on `Response`.** Recommended:
  not now (D5). No caller exists (M3, R55). The measurement phase records
  ChatGPT's headers for a later record.
* **Q4. OpenCode's new codes.** Recommended: yes, `BudgetLimitError` and
  `CreditLimitExceeded` are quota (D4). The evidence is OpenCode's own client,
  which does not retry them (O7). The risk is a code the private service no
  longer sends, which costs nothing.
* **Q5. A past reset.** Recommended: keep it as stated. The field's name
  promises the service's instant. A consumer can compare it with its clock.
  Alternative: zero it when it is not after `time.Now()`.
* **Q6. Kilo's metadata.** Recommended: the one documented key, as
  `CreditsURL`, under the URL rule. Alternative: a `map[string]string` of
  every key, each redacted. That exposes keys no one has documented, and is
  harder to hold to R35.
* **Q7. The release.** Recommended: `v1.29.0`, a minor release, since names
  are exported.

**Q-risk, accepted unless the owner says otherwise.** No live capture of the
new shapes is required before the release:

* a ChatGPT usage limit, a Kilo 402 and an OpenCode budget cannot be produced
  on demand;
* Gemini's 429 needs a burst the owner runs (`LLMPROVIDER_LIVE_GEMINI_429`).

The PLAN's Phase 5 measures what can be measured without exhausting a limit,
and extends the Gemini live check.

## More Information

* GitHub issue 13, "Keep absolute reset times and limit-scope detail on
  APIError". Its "Uncertain" items, answered here:
  * **Do OpenCode Go or Black messages name the window?** Go's do, and so does
    its `metadata.limitName` (O1). Black's did not, in the last public code.
  * **Does ChatGPT send `x-codex-*` headers, or `resets_in_seconds`?** Codex
    reads `x-codex-*` on a success and on a 429 (X10). `resets_in_seconds` is
    fixture-only in Codex. Whether this SDK's ChatGPT requests receive the
    headers is measured in the PLAN's Phase 5.
  * **Does OpenCode send `retry-after` with its 401 quota codes?** Not in the
    last public code (`d21b0790a6`). Those replies set no headers.
* **Out of scope, and noted for gobble.** gobble's issue 14, requirement 4, is
  about OpenCode's 401 quota codes. They unwrap to `ErrAuthFailure` as well as
  `ErrQuotaExhausted`: the pre-0012 status sentinel (MADR 0012 §7), kept for
  compatibility. A consumer that tests `ErrQuotaExhausted` before
  `ErrAuthFailure` words them correctly. Changing `Unwrap` would break the
  compatibility promise, and is not proposed.
* **gobble's issue 14, requirement 3,** needs nothing new here.
  `RetryNotice.ServerDirected` and `Err` exist (C26).
* Related records:
  * 0012-MADR-conform-providers-to-reference-clients.md, §1.1 (the quota
    table) and §4.4 (`resets_at`);
  * 0015-MADR-canonical-sdk-api-and-module-layout.md, D7 (R24);
  * 0026-MADR-remediate-v1-2-debugging-pass-findings.md, F12 (Gemini's 429);
  * 0028-MADR-heuristics-and-performance-from-the-research-pass.md, D-H5,
    and 0028-PLAN's T2 (the measured headers);
  * 0043-MADR-openai-chatgpt-plan-sign-in.md (the plan session's codes).
* PLAN:
  [0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md](0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md).
