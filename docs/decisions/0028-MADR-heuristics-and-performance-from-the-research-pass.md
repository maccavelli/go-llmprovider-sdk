---
status: accepted
date: 2026-10-08
decision-makers: repository owner
consulted: 0012-MADR-conform-providers-to-reference-clients.md (§1.1, §3.1), 0016-MADR-provider-auth-and-support-baseline.md (D8, amendment A5), 0020-MADR-remediate-v1-debugging-pass-findings.md (F9, F28), 0021-MADR-harden-and-tune-after-the-v1-1-review.md (D1, L3, T5, Z1), 0026-MADR-remediate-v1-2-debugging-pass-findings.md (F5, F6, F11, F12), 0027-MADR-live-test-skips-and-gemini-429-path.md
informed: consumers of go-llmprovider-sdk v1, among them prepare-commit-msg (on v1.3.2) and gobble-cli (on v1.1.1)
---

<!-- markdownlint-disable MD013 MD024 -->

# Correct Six Error and Retry Heuristics, and Remove Five Measured Costs

## Context and Problem Statement

On 2026-10-08 the owner asked for a research pass over this repository:
heuristics that could judge better, and algorithms that could make the SDK
faster or cheaper. Two read-only research agents surveyed the code, and
reported about twenty heuristics and fourteen performance candidates. The
owner chose eleven of them from a shortlist of twenty:

* **heuristics** H1, H2, H3, H5, H8 and H10;
* **performance** A1, A2, A3, A6 and A10.

The identifiers are the shortlist's, so the gaps (H4, H6, H7, H9; A4, A5,
A7–A9) are items the owner did not choose; they are listed under More
Information.

Every item below was checked before this record was written: either read
in the code at `HEAD` `0b2e6ea`, or reproduced and measured on a scratch
copy of the repository, never in the tree. Measurements ran with Go 1.27.1
on an Apple M1 Pro. Where a fact rests on a vendor's documentation rather
than on this repository's code, records or a measurement, it is marked
*vendor-documented, not measured here*.

### H1. A reply timeout after the request was sent is resent as if it was never sent

* **Where:** `llmprovider/retry.go:178-182`; `llmprovider/internal/wire/post.go:66-75`; `llmprovider/internal/transport/transport.go:50`.
* **Rule.** `Post` returns `client.Do`'s error as it is, unless
  `transport.Unsendable` says no retry can mend it. `retryable` then treats
  any `*url.Error` as a failure to send:

  ```go
  // A failure to send is retried, unless no retry can mend it
  // (0026-MADR F11).
  var urlErr *url.Error
  return errors.As(err, &urlErr) && !transport.Unsendable(err)
  ```

  It is not marked `transport.AfterReply`, so the "once per Generate" rule
  of 0021-MADR D1 (`retry.go:104-110`) does not apply: it is retried up to
  `MaxAttempts`, 3 by default (`retry.go:30-32`).
* **What it misjudges.** `net/http` also returns a `*url.Error` after the
  request was written in full: `net/http: timeout awaiting response
  headers` (and HTTP/2's `http2: timeout awaiting response headers`). Only
  the ChatGPT path streams (`openai.go:222`, the only `"stream"` key in a
  request body); every other provider receives no headers until the
  generation is complete, and the default `ResponseHeaderTimeout` is 300 s.
  So a generation that runs past it is resent, and billed again, up to
  three times. The repository has met this timeout live: 0009-MADR X5
  ("reasoning on a non-streaming call meets the 30 s header timeout") and
  0013-MADR D6 (`http2: timeout awaiting response headers` on
  `TestLive_OpencodeChatReasoningEffort/glm-5.3-flash`). 0021-MADR T5
  treated the same cost as a defect when it happened after a 200.
* **Measured.** A server that reads the whole request body and answers
  after a client `ResponseHeaderTimeout` of 100 ms, under `WithRetry`
  with `MaxAttempts: 3`: **3 requests received**; the error is `llmprovider:
  openai failed after 3 attempts: Post "…/responses": net/http: timeout
  awaiting response headers`.
* **Pinned by:** `TestWithRetry_KindlessRetriedOnlyFromTheNetwork`
  (`retry_kindless_test.go:18`) pins a refused connection as retried. No
  test covers a failure after the request was written.

### H2. Any 403 the table does not map is a refused credential, so the credential is renewed and the request resent

* **Where:** `llmprovider/api_error.go:178-189` (`statusSentinel`), `:361-362` (the default case); `llmprovider/internal/wire/reauth.go:57-66` (`credentialRefused`); the same rule in `llmprovider/catalog/discovery.go:146-160`.
* **Rule.** `statusSentinel` maps 401 and 403 to `ErrAuthFailure`. Only
  OpenCode and Kilo map 403 to `ErrNotPermitted` (`api_error.go:334-336`,
  0012-MADR §1.1 amendment 2026-10-05, 0021-MADR L3). On OpenAI, Claude,
  Gemini, Grok, Together, Hugging Face and Ollama every 403 is
  `ErrAuthFailure`, and `credentialRefused` reruns the credential on that
  kind (0026-MADR F6).
* **What it misjudges.** A 403 for a region, an entitlement or an
  organisation policy is not a refused credential. Renewing forces an
  OAuth refresh (rotating the refresh token) or reruns a `CommandToken`,
  and the resend gets the same 403. The message then says "authentication
  failed", which sends a user to sign in again.
* **Measured.** The OpenAI provider with a counting `InvalidatingSource`,
  against a 403 with OpenAI's documented region refusal
  (`{"error":{"type":"request_forbidden","code":"unsupported_country_region_territory",…}}`,
  *vendor-documented, not measured here*): **2 requests, 1
  invalidation**; `errors.Is(err, ErrAuthFailure)` true, `ErrNotPermitted`
  false; the message `llmprovider: authentication failed: openai HTTP 403
  unsupported_country_region_territory: …`.
* **Pinned by:** `TestReauth_RerunsOnAuthFailureKind`, case `"403 auth
  failure"` (`reauth_0026_test.go:24`), which wants 2 sends and 1
  invalidation.
* **Note.** `APIError.Unwrap` keeps the legacy status sentinel, so a 403 of
  kind `ErrNotPermitted` still matches `ErrAuthFailure` through it
  (`reauth.go:23-26`). A consumer that tests `errors.Is(err,
  ErrAuthFailure) && !errors.Is(err, ErrNotPermitted)`, as prepare-commit-msg
  does (`main.go:284` there), already tells the two apart.

### H3. A request whose completion alone fills the context is classified as context overflow

* **Where:** `llmprovider/context_overflow.go:33-34` (and `:30`, `:42`); the gate at `llmprovider/api_error.go:359`.
* **Rule.** Among others, `(?i)reduce the length of the messages` and
  `(?i)maximum context length is \d+ tokens` make a 4xx
  `ErrContextOverflow`, whose documented meaning is that shortening the
  input can succeed.
* **What it misjudges.** The OpenAI-compatible message reports input plus
  requested completion: "This model's maximum context length is 8192
  tokens. However, you requested 8202 tokens (10 in the messages, 8192 in
  the completion). Please reduce the length of the messages or
  completion." When the completion alone is at or above the limit, no
  shortening of the input can succeed; only a smaller `max_tokens` can.
  The SDK's default `max_tokens` is 8192 (`settings.go:82`), so an
  8K-context model on Together or Hugging Face meets this with a short
  prompt. A caller that compacts its history on overflow compacts for
  ever.
* **Measured.** That body, as a 400, through `ClassifyHTTPError` for
  `together` and for `huggingface`: `ErrContextOverflow` both times.
* **Pinned by:** `TestContextOverflow_Messages`, whose samples include the
  two patterns.

### H5. A 429 without a delay is retried after 0.5–2 s, and the services' reset headers are not read

* **Where:** `llmprovider/retry.go:149-160` (`wait`); `llmprovider/api_error.go:213-238` (`RetryAfter` from `retry-after-ms`, `Retry-After`, a body's reset time or `RetryInfo`). No code outside tests reads an `x-ratelimit-*` header.
* **Rule.** With no delay from the service, a 429 is retried on the
  backoff: `BaseDelay` 1 s doubling, each wait between half and all of it.
* **What it misjudges.** OpenCode Zen's and Kilo's 429s carry no
  `Retry-After` (measured: `0004-MADR-add-gateway-llm-providers.md:237`,
  `:1429`). A per-minute window is not passed in 0.5–2 s, so the two
  retries are spent. Several services publish when the window resets:
  OpenAI and xAI in `x-ratelimit-reset-requests` and
  `x-ratelimit-reset-tokens` (xAI's `x-ratelimit-limit-requests` is
  measured in `0006-MADR-subscription-auth-for-llm-providers.md:1025-1026`;
  the reset headers are *vendor-documented, not measured here*), and
  Anthropic in `anthropic-ratelimit-*-reset` (*vendor-documented, not
  measured here*).
* **Measured.** The Together provider, against a 429 with no `Retry-After`
  and `x-ratelimit-reset-requests: 20s`, under `WithRetry` with the default
  policy: **3 requests**, gaps of **0.90 s and 1.42 s**, 2.32 s in all;
  `RetryAfter` 0.
* **Cost.** A 429 is not billed; the cost is two requests that cannot
  succeed, and a caller that is not told how long to wait.

### H8. A failure inside a 200 with no code is retried, even an invalid request or a context overflow

* **Where:** `llmprovider/api_error.go:268-292` (`ClassifyStreamFailure`); `llmprovider/internal/wire/chatcompletions/chatcompletions.go:267-283, 371-386` (a 200 carrying `{"error":…}`, as OpenRouter-style gateways send, 0021-MADR W5); the overflow gate at `api_error.go:359`.
* **Rule.** A code that is not a known name or an HTTP status is classified
  as status 500, "the table's codes win, else retryable". The overflow check
  runs only for a 4xx.
* **What it misjudges.** OpenAI-style error envelopes name their class in
  `type` (`invalid_request_error`, `authentication_error`, …) and often send
  `code: null`. Such a failure inside a 200 is `ErrProviderUnavailable`,
  retryable, whatever its type says.
* **Measured.** The Together provider, answering 200 with
  `{"error":{"type":"invalid_request_error","code":null,…}}`, under
  `WithRetry`:
  * an overflow message (8000 in the messages, 1000 in the completion):
    **3 requests**, `ErrProviderUnavailable`, not `ErrContextOverflow`;
  * `'messages' must contain at least one message`: **3 requests**,
    `ErrProviderUnavailable`, not `ErrInvalidRequest`.
* **Related.** 0026-MADR F5 classifies a numeric code inside a 200 as that
  status; a typed envelope with no code was left to the fallback.

### H10. A key the service refuses during `configure` is reported as a listing outage, and returned to be saved

* **Where:** `wizard/configure.go:132-202` (`ConfigureLLM`), `:322-400` (`discoverModels`, the notice at `:389-391`); `llmprovider/catalog/discovery.go:160-171` (`catalogFrom` keeps the failure in `Catalog.Err` and serves the built-in catalog).
* **Rule.** A listing that fails for any reason, a refused credential
  included, falls back to the built-in catalog; the wizard warns that the
  listing is unavailable, and returns the `Result` with the credential.
* **What it misjudges.** A mistyped or revoked key is a user error the
  wizard could catch at the moment it was typed. Returned in `Result`, it
  is saved by the caller, and every later request fails.
* **Measured.** `ConfigureLLM` with the wizard's own `fakePrompter`, Claude
  chosen, and a listing client answering 401 `authentication_error`: **no
  error**; `Result.APIKey` is the refused key; `Result.Model` is the first
  built-in model; one notice, `live model listing for Claude (Anthropic)
  is unavailable (llmprovider: authentication failed: claude HTTP 401
  authentication_error: invalid x-api-key); search covers the built-in
  catalog only`.

### A1. OpenCode waits for a 5.4 MB document before a new process's first request, and refetches it whole every ten minutes

* **Where:** `llmprovider/providers/opencode/opencode.go:306-318` (`requestRoute`, which looks the metadata up for every request unless the route is pinned), `:448-473`, `:501-508`; `llmprovider/catalog/model_metadata.go:23-33` (TTL 10 min, a failure remembered 1 min, a lookup bounded at 5 s), `:276-305` (an empty cache waits for the fetch), `:345-384` (a plain GET, then `decodeLimited`: `io.ReadAll` and `json.Unmarshal`, `discovery.go:38-50`).
* **Measured.**
  * The default document, `https://models.opencode.ai/api.json`:
    5,366,509 bytes; three fetches took 0.68 s, 0.46 s and 0.42 s. Go's
    transport asks for gzip, which the server serves at 535,824 bytes.
  * The server sends `ETag` and `cache-control: public, max-age=0,
    must-revalidate`; a request with `If-None-Match` returned **304, 0
    bytes, in 0.11 s**.
  * `BenchmarkDecodeModelMetadata` on the real document: **15.6 ms, 11.3 MB
    and about 2,300 allocations** per decode.
  * The first lookup on an empty cache, against a server that takes the
    measured 450 ms: **486 ms** before the lookup returns, and so before the
    generation request is sent.
  * A warm lookup (`BenchmarkLookupMetadataWarm`): **1.2 µs and 16
    allocations**. The up to three lookups a request makes cost under 4 µs,
    so they are not worth changing.
* **Who pays.** A short-lived process starts with an empty cache: a git
  hook pays the wait on every commit, even for a model the built-in route
  table knows. A long-lived one pays the full download and the decode
  every ten minutes, where a 304 would do.
* **Rule this touches.** MADR 0012 §3.1: the route comes from the
  metadata's `provider.npm`, "the table remains the fallback when metadata
  is unavailable", and `WithOpencodeRoute` overrides both.

### A2. Every provider instance builds its own connection pool

* **Where:** `llmprovider/settings.go:203-207`, which calls `transport.DefaultClient()` for each `ResolveOptions` without `WithHTTPClient`; `llmprovider/internal/transport/transport.go:43-55`, which builds a new `http.Transport` each time.
* **Measured.** An `httptest` server counting new connections: one provider
  making two calls opened **1 connection**; two providers built the same
  way, one call each, opened **2**. A TLS handshake to a public host took
  45 ms with DNS and connections warm, and 230 ms cold (curl's
  `time_appconnect` against `models.opencode.ai`).
* **Who pays.** A caller that builds a provider per request, per user or
  per model. prepare-commit-msg builds one per model it tries
  (`main.go:272` there), so each fallback model pays a new handshake to
  the same host. Idle pools live 90 s; 0020-MADR F28 measured the same
  pattern in listings ("20 listings gave 60 goroutines") before the
  catalog moved to one shared `listingClient` (`catalog/config.go:83-89`).
* **Rule this touches.** 0016-MADR D8: "One default client per provider
  instance serves requests, listing and refresh."
  `TestResolveOptions_Defaults` (`settings_test.go:96-97`) compares client
  pointers, not transports.
* **Measured: three transport strategies** (Q6). Against a TLS test server
  taking 20 ms a call, counting connections, in three patterns: *burst*, 10
  instances × 5 calls started at once; *sequential*, 10 instances built one
  after another with one call each, as a fallback loop or a provider per
  request does; *staggered*, 10 instances × 5 calls started 5 ms apart.
  Two runs agreed on every count:

  | Pattern | Protocol | A transport per instance (today) | One shared transport | A manager of 4 identical transports |
  | :--- | :--- | ---: | ---: | ---: |
  | burst | HTTP/1.1, HTTP/2 | 10 | 10 | 10 |
  | sequential | HTTP/1.1, HTTP/2 | 10 | 1 | 4 |
  | staggered | HTTP/2 | 10 | 1 | 4 |
  | staggered | HTTP/1.1 | 10 | 10–16 | 12 |

  Wall times were within run-to-run noise in every pattern, 109–188 ms,
  with no strategy consistently faster. One `http.Transport` already serves
  concurrent requests: each connection runs its own read and write
  goroutines, HTTP/2 multiplexes requests on one connection, and the
  runtime spreads goroutines over OS threads. Sharding identical
  transports adds no parallelism, and costs a handshake per transport.

### A3. `ListModels` sends up to six billed probes on every call, and caches nothing

* **Where:** `llmprovider/providers/openai/openai.go:323-339`, and the same pattern in Claude, Gemini, Grok and Ollama; `llmprovider/internal/transport/probe.go:20-50`; `llmprovider/options.go:59-66`.
* **Rule.** Probes are on by default (`WithModelProbes`: "With probes,
  which is the default, ListModels … sends one short generation to each
  listed model, up to MaxListedModels … Every probe is a billed request";
  0016-MADR D9 was replaced by amendment A5, "probes are on by default
  again"). `catalog.MaxListed` is 6. Each probe sends `"Respond with ONLY
  the word Hello"` with a 8192-token output limit (`probeMaxOutputTokens`
  in OpenAI, Grok and Ollama; `probeMaxTokens` in Claude).
* **What it costs.** Every `ListModels` call spends up to six generations,
  and a UI or a wizard that lists again spends them again. Nothing is
  remembered between calls.

### A6. Chat Completions bodies are built as nested maps, at a cost that grows with the history

* **Where:** `llmprovider/internal/wire/chatcompletions/chatcompletions.go:66-155` (each item becomes one to three `map[string]any`), `:184-203` (`Body`); `llmprovider/internal/wire/post.go:51` (`json.Marshal`, which sorts each map's keys by reflection); `llmprovider/internal/wire/wire.go:55-66` (`ToolArguments` runs `json.Valid`, then `json.Compact`, and `json.Marshal` validates the `RawMessage` again).
* **Measured** (`BenchmarkBodyMarshal`, a history in turns of user message,
  assistant message, a call with about 1 KiB of arguments, its output):

  | Items | Time | Bytes | Allocations |
  | :--- | :--- | :--- | :--- |
  | 40 | 42–50 µs | 41 KB | 540 |
  | 100 | 101–105 µs | 104 KB | 1,322 |
  | 200 | 207–238 µs | 218 KB | 2,624 |

  The cost is linear in the history, and an agent loop sends the whole
  history every turn, so a conversation's total grows with the square of
  its length.
* **Measured** (`BenchmarkPlainMessages`, 100 plain messages encoded
  through the codec's maps against a typed struct of the same JSON):
  **maps 54–55 µs, 57 KB, 1,008 allocations; typed 12.3–12.6 µs, 11.7 KB,
  4 allocations**: 4.4 times the time, and 250 times the allocations.
* **Goldens.** The G-wire goldens decode each body and compare structurally
  (`internal/wiretest/wiretest.go:73-94`, `:250`), so key order does not
  change them; a dropped field does.

### A10. Classifying an error body costs up to 11.6 ms, almost all of it in regular expressions

The shortlist named this item for repeated JSON decoding. Profiling showed
the decoding is not the cost:

* **Measured.**
  * `ClassifyHTTPError` on a 2 KiB JSON 400: **0.59–0.60 ms** when the
    message is an overflow, **0.82 ms** when it is not; 33 allocations.
  * Its parts: `parseAPIErrorBody` **9 µs**; one `json.Unmarshal` of the
    same body **3 µs**. The several decodes cost about 1 % of the total.
  * A CPU profile of the 0.82 ms case: **about 70 %** in `redact.String`
    (`internal/redact`'s regular expressions), **about 23 %** in
    `contextOverflow`'s.
  * At the redaction limit, a 16 KiB message that says "tokens", as every
    overflow message does: **11.6 ms** and 158 KB per classification.
    Eight times the size cost nineteen times the time.
  * For comparison, the existing `BenchmarkClassifyHTTPError_64KiBHTML`
    measures 0.89 ms: an HTML page holds none of the redaction anchors.
* **Why.** 0021-MADR Z1 measured redaction at about 400 ns per byte and
  added two bounds: an anchor prefilter, so a pass runs only when the text
  holds one of its literal words, and a 16 KiB redaction limit
  (`api_error.go:62-65`), though only `apiErrorMessageLimit`, 512 bytes, is
  kept. "token" is an anchor of two passes (`internal/redact/redact.go:135,
  137`), and English error text says "tokens", so the prefilter does not
  stop them. `reKV` (`redact.go:49`) begins `\b((?:[a-z0-9]+[_-]?)*(?:…))`,
  a repetition tried from every word boundary.
* **Cost.** Error path only, under `WithRetry` once per attempt. 11.6 ms
  is small next to a request's network time, and large next to the 3 µs a
  decode needs.

## Decision Drivers

* **A failure is classified as what it is,** so a retry is spent only when
  it can succeed, and a caller acting on a kind (renew, compact, fall back,
  wait) does the right thing.
* **No billed request is sent twice by a heuristic** (0021-MADR T5's
  principle).
* **A credential is renewed only when the service refused the credential.**
* **A user learns of a refused key when they type it,** not at the next
  request.
* **The common path is cheap:** a short-lived process does not wait on a
  document its route table already answers; a provider does not pay a
  handshake its process already paid; a listing does not pay for the same
  probes twice.
* **Each change is proven red first,** and each performance change by a
  benchmark that records before and after.
* **Existing decisions change only by amendment:** MADR 0012 §1.1 and §3.1,
  0016-MADR D8 and amendment A5, 0021-MADR D1 and Z1.
* **The public API only adds** (R48): any new exported identifier is an
  addition, and behaviour changes are listed in the release notes.

## Considered Options

* **A. One decision, eleven changes, implemented in phases by risk, and
  released as `v1.4.0`.**
* **B. Two records: the heuristics now, the performance changes under a
  later record.**
* **C. The heuristics only; the performance items recorded as found, not
  done.**
* **D. Change nothing; document the eleven behaviours.**

## Decision Outcome

Chosen option: **"A"**, because the eleven share their evidence, their
amendments touch the same classification and transport code, and one
`v1.4.0` lets consumers take every behaviour change at once. The PLAN
phases them so that the measurement-first items land before the wider
changes, and A2, which changes connection behaviour for every caller,
lands last.

Each decision below carries a recommendation where a design choice is
open; the owner questions follow.

### D-H1. A failure after the request was written is retried at most once

* `Post` traces the request with `httptrace.ClientTrace.WroteRequest`.
  When `client.Do` fails after the request was written, the error is marked
  `transport.AfterReply`, so `WithRetry` retries it at most once per
  `Generate`, as 0021-MADR D1 does for a reply cut short (Q1).
* A failure before the request was written (refused connection, DNS, TLS
  handshake) is retried as today.

### D-H2. A 403 is a refused credential only when it says so

* `statusSentinel`'s 403 becomes `ErrNotPermitted`; a 403 whose body
  carries a credential code stays `ErrAuthFailure` (Q2).
* The credential codes are measured, not assumed: the PLAN sends each
  provider a deliberately invalid and an expired credential (unbilled
  requests) and records status, type and code. Only codes seen there enter
  the table.
* `credentialRefused` (`reauth.go`, `catalog/discovery.go`) is unchanged:
  it renews on `ErrAuthFailure` alone, so a region or entitlement 403 is no
  longer renewed or resent.
* `TestReauth_RerunsOnAuthFailureKind`'s `"403 auth failure"` case changes
  with an amendment to MADR 0012 §1.1.

### D-H3. Overflow means the input can be shortened

* When an overflow message states its split, as `(N in the messages, M in
  the completion)` with the limit L, and M ≥ L, the failure is
  `ErrInvalidRequest`, its message naming `max_tokens`; otherwise it stays
  `ErrContextOverflow`.
* A message that states no split is classified as today.

### D-H5. A 429 reads the reset the service published

* `RetryAfter` falls back, in order, to `retry-after-ms`, `Retry-After`,
  the body's reset time or `RetryInfo` (as today), then the reset headers:
  `x-ratelimit-reset-requests` and `x-ratelimit-reset-tokens` (durations,
  such as `20s` or `6m0s`), and `anthropic-ratelimit-requests-reset` and
  `-tokens-reset` (RFC 3339 times), the longer of the pair.
* The PLAN first measures which headers each provider sends on a 200 and
  on a 429, and reads only those (Q3). `WithRetry`'s `MaxDelay` rule is
  unchanged: a reset past it returns at once, for the caller to reschedule.
* A 429 with no delay of any kind keeps today's backoff.

### D-H8. A failure inside a 200 is classified by its type

* `ClassifyStreamFailure`, when the code is empty or unknown and `errType`
  names an OpenAI- or Anthropic-style class, classifies by the class:
  `invalid_request_error` as `ErrInvalidRequest`, after the overflow check;
  `authentication_error` as `ErrAuthFailure`; `permission_error` as
  `ErrNotPermitted`; `rate_limit_error` as `ErrRateLimited`;
  `overloaded_error`, `api_error` and `server_error` as
  `ErrProviderUnavailable`.
* The overflow check runs on a failure inside a 200 whatever the stand-in
  status.
* An unknown type keeps the 500 fallback.

### D-H10. The wizard asks again when the service refuses the key

* When the listing fails with `ErrAuthFailure` for a key the user typed,
  `ConfigureLLM` says the service refused the key and asks for it again, up
  to three times, then returns an error (Q4).
* A listing that fails for another reason keeps today's warning and the
  built-in catalog.
* A credential the user did not type (an environment variable, a vendor
  CLI file, a sign-in) is not re-prompted: the wizard says which source was
  refused, and returns an error.

### D-A1. OpenCode does not wait for metadata its route table can stand in for, and revalidates instead of refetching

* **Cold cache, model in the table:** `requestRoute` uses the table's route
  and starts the fetch in the background; the next request uses the
  document once it is cached (Q5). A model the table does not know waits,
  as today.
* **Refresh:** the cache keeps the document's `ETag`, and the refresh sends
  `If-None-Match`; a 304 renews the entry's time without a download or a
  decode.
* **Decode:** `json.NewDecoder` over the limited reader replaces
  `io.ReadAll` followed by `json.Unmarshal`, keeping the 32 MiB limit.
* MADR 0012 §3.1 is amended: the table is the route while the metadata is
  being fetched, as well as when it is unavailable.

### D-A2. A process-level transport manager, one transport per configuration

* **The manager.** `internal/transport` gains a process-level manager that
  owns the default transports: one `*http.Transport` for each distinct
  transport configuration (proxy, TLS settings, timeouts, connection
  limits), created on first use and kept for the process. Today every
  default configuration is the same, so there is one; a configuration that
  differs gets its own, and never shares connections with the others.
  `MaxIdleConnsPerHost` rises from 4 to 16.
* **Per-instance clients.** `transport.DefaultClient()` returns a new
  `*http.Client` for each provider instance, so
  `TestResolveOptions_Defaults` holds. The client's transport is a thin
  wrapper over the manager's that does not implement
  `CloseIdleConnections`, so one instance's client cannot close idle
  connections the others would reuse.
* **Not sharded.** The manager does not spread identical traffic over
  several transports: measured above, that opened 4 connections where one
  transport needed 1, with no gain in wall time (Q6).
* **Unchanged:** a caller's `WithHTTPClient`; the catalog's
  `listingClient`, which becomes the manager's default client.
* 0016-MADR D8 is amended: "one default client per provider instance, over
  the transport the process-level manager keeps for its configuration".

### D-A3. Probe results are cached for ten minutes

* The models that answered a probe are cached for 10 minutes, keyed by
  provider, base URL and a SHA-256 fingerprint of the credential the probes
  used, held in memory only (Q7).
* A listing within the TTL sends no probes; a different credential, base
  URL or provider probes again.
* `WithModelProbes(false)` and 0016-MADR amendment A5's default are
  unchanged.

### D-A6. Chat Completions encodes typed structs

* Chat Completions' messages, tool calls and tools are typed structs; the
  JSON each request sends is unchanged, field for field, and the goldens
  prove it.
* `ToolArguments` compacts once: `json.Compact` validates as it compacts,
  so the separate `json.Valid` goes.
* The other three wires (Messages, Responses, generateContent) move only if
  the PLAN's benchmarks show a gain of the same order; otherwise they are
  recorded as measured and not done.

### D-A10. Classification bounds its regular-expression work to what it keeps

* **Redaction runs on what can be kept.** The message is cut to 2 KiB (four
  times `apiErrorMessageLimit`) before redaction, at a separator, so no
  fragment of a secret straddles the cut; then it is redacted, then cut to
  512 bytes as today. `apiErrorRedactLimit` goes from 16 KiB to 2 KiB, and
  0021-MADR Z1 is amended.
* **`reKV` finds its keyword first.** The pass looks up the keyword
  literally and matches the assignment around it, instead of trying the
  leading repetition from every word boundary. Every case of the redaction
  tests passes unchanged.
* **The overflow patterns share one prefilter:** a literal word every
  pattern needs ("context", "token", "too large", "too long", …) before
  any pattern runs.
* The JSON decoding is left as it is: it is about 1 % of the cost.
* This re-scopes the shortlist's A10 on the evidence; Q8 asks the owner to
  confirm it.

### Owner questions

Each has a recommendation, written into the decisions above.

* **Q1 (D-H1).** A failure after the request was written: retried at most
  once (recommended), or never?
* **Q2 (D-H2).** 403 is `ErrNotPermitted` unless its body carries a
  measured credential code (recommended); or 403 stays `ErrAuthFailure`,
  and `credentialRefused` alone stops renewing on a 403 without such a
  code?
* **Q3 (D-H5).** Read the reset headers the PLAN measures each provider
  sending (recommended); or also lengthen the backoff of a 429 that gives
  no delay at all?
* **Q4 (D-H10).** Re-prompt up to three times, then fail (recommended);
  fail at once; or return the key with the listing's failure in a new
  `Result` field, for the caller to decide?
* **Q5 (D-A1).** On a cold cache, use the table's route for a model it
  knows and fetch in the background (recommended); or keep waiting, and
  take only the revalidation and the streamed decode?
* **Q6 (D-A2).** Share one default transport (recommended), or keep one
  transport per instance? The owner proposed a third design, a single
  process-level transport manager over several transports on their own
  goroutines, to be evaluated; the measurement under A2 led to the
  manager with one transport per configuration.
* **Q7 (D-A3).** Cache probes 10 minutes keyed by provider, base URL and
  credential fingerprint (recommended); per provider instance only; or
  another TTL?
* **Q8 (D-A10).** Re-scope A10 from the JSON decoding to the regular
  expressions, as measured (recommended), or drop A10?

**Answered 2026-10-08.** The decisions above are written to these answers:

* **Q1:** "At most once".
* **Q2:** "NotPermitted unless code".
* **Q3:** "Read reset headers".
* **Q4:** "Re-prompt, then fail".
* **Q5:** "Table route + background".
* **Q6:** the owner proposed "a single transport process manager, but
  multi-threaded transports, using separate goroutines" and asked for it to
  be evaluated; after the experiment under A2, the owner chose "Manager,
  one per config".
* **Q7:** "10 min, process-wide".
* **Q8:** "Re-scope to regexes".

**Accepted 2026-10-08** ("Acvept the madr").

### Consequences

* Good, because a slow generation is billed at most twice, not three
  times (H1).
* Good, because a region or entitlement 403 no longer rotates a refresh
  token, reruns a command, or tells the user to sign in again (H2).
* Good, because a caller that shortens its input on overflow is not sent
  round a loop that shortening cannot end (H3), and a failure inside a 200
  that says it is an invalid request is not sent three times (H8).
* Good, because a 429 from a service that says when its window resets
  waits that long, or is handed back to the caller, rather than spending
  two requests (H5).
* Good, because a mistyped key is caught while the user is still at the
  keyboard (H10).
* Good, because a git hook on OpenCode stops paying about half a second
  per commit for a document its table can stand in for, and a long-lived
  process stops re-downloading 5.4 MB and decoding it in 15.6 ms every ten
  minutes (A1).
* Good, because a second provider instance reuses the first one's
  connections (A2), and a second listing reuses the first one's probes
  (A3).
* Good, because a Chat Completions request allocates a small fraction of
  what it does today, in a cost that every turn of a conversation repeats
  (A6), and a 16 KiB error is classified in a fraction of 11.6 ms (A10).
* Neutral, because `v1.4.0` is a minor release with behaviour changes:
  `errors.Is` answers differently for a 403 on seven providers, a
  completion-only overflow, and a typed failure inside a 200. The release
  notes list them.
* Bad, because a 403 that is a credential problem but carries no code in
  the measured table becomes `ErrNotPermitted`, and is not renewed. The
  table is measured to keep that set small, and the legacy sentinel still
  matches `ErrAuthFailure`.
* Bad, because a model whose route the gateway changed after the table's
  snapshot is sent to the old route on a cold cache, once per process,
  until the background fetch lands (A1). A wrong route answers 500, which
  is retried.
* Neutral, because a provider's client no longer closes idle connections
  when asked: the wrapper hides `CloseIdleConnections`, so the shared pool
  is released by its idle timeout, 90 s, or when the process ends (A2).
* Bad, because the manager keeps one transport per configuration for the
  process's life; a process that builds many differently configured
  default clients keeps that many pools (A2).
* Bad, because a model that stops answering within the probe TTL stays in
  the listing for up to ten minutes (A3).
* Bad, because cutting a message to 2 KiB before redaction drops text past
  2 KiB even when redaction would have shortened the first 2 KiB below 512
  bytes; the kept message is then shorter, never less redacted (A10).

### Confirmation

* **Red first, every item:** each has a test that fails on today's code
  for the measured reason above, and passes after; the PLAN records both
  FAIL and PASS lines.
  * H1: the reproduction above, wanting at most 2 requests.
  * H2: the region 403 through `Reauth` with a counting source, wanting 1
    send and 0 invalidations; a credential 403 from the measured table
    still renews.
  * H3: the 10/8192/8192 body is `ErrInvalidRequest`; an 8000/1000 body on
    an 8192 limit stays `ErrContextOverflow`.
  * H5: the 429 with `x-ratelimit-reset-requests: 20s` has `RetryAfter`
    20 s, and under the default `MaxDelay` returns after one request.
  * H8: both 200 bodies above end after one request, as
    `ErrContextOverflow` and `ErrInvalidRequest`.
  * H10: the reproduction above re-prompts, and with three refusals
    returns an error and no key.
* **Benchmarks, before and after, recorded in the PLAN:**
  * A1: the first request on a cold cache, for a table model, against a
    450 ms metadata server, in under 50 ms; a refresh against a 304 makes
    no decode.
  * A2: two providers, one call each, open 1 connection; a client's
    `CloseIdleConnections` leaves the other instance's idle connection open;
    a differently configured default gets its own transport.
  * A3: a second `ListModels` within the TTL sends 0 probes.
  * A6: `BenchmarkPlainMessages` and `BenchmarkBodyMarshal` show at least
    a fivefold cut in allocations.
  * A10: `ClassifyHTTPError` on the 16 KiB "tokens" message in under 1 ms,
    and every redaction test unchanged.
* **The gate** (`make pre-add-check` and CI's checks, `api-check` showing
  additions only), the goldens unchanged, and the live suites on the
  release tag.

## Pros and Cons of the Options

### A. One decision, eleven changes, phased by risk, released as `v1.4.0`

* Good, because one release carries every behaviour change, with one set
  of notes for consumers.
* Good, because the classification items (H2, H3, H8, A10) touch the same
  functions, and land together without conflicting.
* Bad, because the record is large, and a deviation in one phase can hold
  the release.

### B. Two records, heuristics first

* Good, because each record is smaller, and the heuristics, which carry the
  billed and user-facing costs, ship sooner.
* Bad, because consumers take two minor releases.
* Bad, because H3 and A10 both change overflow detection, and would land in
  different records.

### C. The heuristics only

* Good, because the smallest change that removes the billed costs.
* Bad, because A1's half-second per commit for the hook, and A3's repeated
  billed probes, stay.

### D. Document the behaviours, change nothing

* Good, because nothing breaks.
* Bad, because the measured costs stay: a slow generation billed three
  times, renewals on region refusals, overflow loops, retried invalid
  requests, refused keys saved, and the latency and probe costs.

## More Information

### Method

* **Research:** two read-only research agents, 2026-10-08: one on
  heuristics (about twenty found), one on algorithms and performance
  (fourteen). Their reports are not records; every fact taken from them
  was checked in the code or reproduced before it entered this record.
* **Reproductions and benchmarks** ran on scratch copies of the
  repository, with test files that were never committed: the H1, H2 and
  H3 reproductions; `TestH5_RateLimitWithoutDelay`, `TestH8_ErrorInside200`
  and `TestH10_RefusedKeyIsReturned`; `BenchmarkDecodeModelMetadata`,
  `BenchmarkLookupMetadataWarm` and the cold-lookup test;
  `TestConnectionsPerProviderInstance` and the three-strategy transport
  experiment (`TestStrategies`, a standalone module); `BenchmarkBodyMarshal` and
  `BenchmarkPlainMessages`; `BenchmarkClassify_2KiB*`, `_16KiBTokens`,
  `BenchmarkParseOnly`, `BenchmarkOverflowOnly` and `BenchmarkOneDecode`;
  and a CPU profile of the 2 KiB classification. The PLAN turns each into a
  committed test or benchmark.
* **Network measurements:** three `curl` fetches of the metadata document,
  one conditional request, and one gzip request. No provider was sent a
  billed request.

### Measured and not chosen

* OAuth `Token()` parses the access JWT on every call
  (`auth/oauth_session.go:403`): 3.6 µs and 5 allocations, 4.1 µs with ten
  goroutines on one session. Not worth changing.
* `Capabilities.Check` marshals every tool schema to check it, and
  discards the bytes (`contract.go:216`): 0.29 ms and 86 KB per `Generate`
  with twenty 2 KiB schemas, equal to the wire's own marshal (A4 in the
  shortlist).

### On the shortlist, not chosen

H4 (Gemini's per-day quota cannot be seen on the Interactions API), H6 (a
TLS certificate error is retried), H7 (a Gemini Interactions status other
than the three known is retried after a 200), H9 (`lockWait`, 40 s, is
shorter than a legitimate refresh hold, about 45.6 s); A4 (above), A5
(`VendorCLISession` re-reads and parses its file on every `Token`), A7 (the
file lock polls), A8 (bodies are closed undrained), A9 (ranking scores
inside the sort comparator).

### Found by the research, not on the shortlist

Anthropic's 413 `request_too_large` is a byte limit pinned as overflow
(`context_overflow.go:18`); the vendor menus ignore the model profile;
refresh retries ignore `Retry-After`; an OpenRouter key passes the
wizard's OpenAI key check; OpenAI's backfill can list a dated snapshot
beside its alias. None is decided here.

### Relationship to other records

* **MADR 0012 §1.1** (the classification table): amended by D-H2.
* **MADR 0012 §3.1** (OpenCode routes): amended by D-A1.
* **0016-MADR D8** (the default client): amended by D-A2, a process-level
  transport manager behind per-instance clients. **Amendment A5**
  (probes on by default) stands; D-A3 caches its results.
* **0021-MADR D1** (a reply cut short is retried once): extended by D-H1.
  **Z1** (redaction bounds): amended by D-A10. **L3** (OpenCode's 403):
  generalised by D-H2.
* **0026-MADR F5** (a numeric code inside a 200): extended by D-H8.

## Amendment 2026-10-08: D-H2's table is measured from invalid credentials

Made while writing
[0028-PLAN-heuristics-and-performance-from-the-research-pass.md](0028-PLAN-heuristics-and-performance-from-the-research-pass.md),
before any phase ran.

* **Corrects D-H2's method.** D-H2 says the PLAN sends each provider "a
  deliberately invalid and an expired credential". An expired credential
  cannot be produced without a person signing in and waiting out its
  lifetime: an API key does not expire, and a forged JWT is refused for
  its signature, not its age.
* **Now:** the credential-code table is built from each provider's reply to
  a deliberately invalid API key (unbilled), and from the refusals this
  repository's records already captured (0020-MADR F23's `API_KEY_INVALID`;
  0026-PLAN D12's Interactions reply). A 403 code enters the table only if
  a provider sent it for a refused credential. The PLAN records what an
  expired credential returns as not measured.

## Amendment 2026-10-08: H11, Grok's refused key, and A10b, diagnostic codes

Made by the 0028 PLAN's deviations D1 and D2, chosen by the owner ("Add to
0028, Phase 3"; "Phase 4, after equivalence"), from Phase 1's measurement.

* **H11.** xAI answers a refused key with HTTP 400, code
  `invalid-argument`, "Incorrect API key provided", classified today as
  `ErrInvalidRequest`. So `Reauth` does not renew a refused Grok credential,
  and a caller stopping on `ErrAuthFailure` tries every fallback with it,
  as Gemini's did before 0026-MADR F6 and D12. **Decision:** a Grok 400 with
  code `invalid-argument` whose message names an API key is
  `ErrAuthFailure`; one whose message does not stays `ErrInvalidRequest`.
* **A10b.** Redaction masks diagnostic code values, such as Grok's
  `invalid-argument` and Kilo's `INVALID_TOKEN`, because 0021-MADR Z2's
  diagnostic rule takes lower-case snake case only. **Decision:** a value
  made of two or more letter-only words joined by `-` or `_`, at most 40
  bytes, is a diagnostic; any value with a digit and a letter is still
  masked. 0021-MADR Z2 is amended in Phase 4.
* Both ship in `v1.4.0`.

## Amendment 2026-10-08: A10c, key shapes in redaction, and H10b, in the wizard

Made by the 0028 PLAN's deviation D3, chosen by the owner ("Phase 4 +
Phase 5"), after a research pass on every supported provider's key and
token formats, checked against the owner's keys by shape only.

* **Found.** Today's redaction leaves a Gemini auth key (`AQ.`, the default
  for new AI Studio keys since 2026-05-28), a Google refresh token (`1//`)
  and a Hugging Face org token (`api_org_`) unmasked, and masks ordinary
  text such as `sk-learn-…` and `xai-grok-…`. The wizard does not name an
  OpenRouter, OpenCode or Gemini `AQ.` key pasted for OpenAI.
* **A10c.** Redaction gains a sourced row per published shape, and its
  generic `sk-` and `xai-` rows mask only a body of 32 or more characters
  holding a letter and a digit. Nothing masked today is unmasked except by
  a tighter row.
* **H10b.** The wizard names the newly researched foreign keys, and warns,
  without refusing, when a typed key does not match its provider's
  confirmed shape.
* The formats are the vendors' practice, not contracts: a vendor's new
  format falls to the generic rows and the keyword rules, which stay.
* Both ship in `v1.4.0`.

## Amendment 2026-10-08: D-H1 refines 0020-MADR F9

Made by the 0028 PLAN's deviation D4, chosen by the owner ("Test follows
D-H1"). 0020-MADR F9 (Q2 a) retries a kindless failure from the network on
`WithRetry`'s backoff. D-H1 keeps that for a failure before the request was
written (a refused connection, DNS, a TLS handshake), and resends one after
it, such as a connection dropped after the service received the request, at
most once. `TestGenerate_AnsweredOnceIsNotBoughtAgain` follows.
