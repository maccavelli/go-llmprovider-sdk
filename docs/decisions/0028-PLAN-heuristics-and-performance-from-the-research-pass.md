---
status: in-progress
date: 2026-10-08
associated-madr: "0028-MADR-heuristics-and-performance-from-the-research-pass.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Six Heuristic Corrections and the Five Measured Cost Reductions

Associated MADR: [0028-MADR-heuristics-and-performance-from-the-research-pass.md](0028-MADR-heuristics-and-performance-from-the-research-pass.md)

## Goal

When it is done, every decision of the MADR holds, each proven red first,
and `v1.4.0` is released:

* **D-H1:** a generation that fails after its request was written is sent
  at most twice under `WithRetry`.
* **D-H2:** a 403 without a measured credential code is `ErrNotPermitted`,
  and is neither renewed nor resent; the legacy sentinel still matches
  `ErrAuthFailure`.
* **D-H3:** an overflow message whose completion alone fills the context is
  `ErrInvalidRequest`.
* **D-H5:** a 429's `RetryAfter` carries the reset header each provider was
  measured sending.
* **D-H8:** a typed failure inside a 200 is classified by its type, and an
  overflow inside a 200 is `ErrContextOverflow`.
* **D-H10:** `configure` re-prompts a typed key the service refused, up to
  three times, and fails for a refused key from any other source.
* **D-A1:** a cold OpenCode cache does not hold up a model the route table
  knows; a refresh revalidates by `ETag`; the document is decoded as a
  stream.
* **D-A2:** default clients share the transport a process-level manager
  keeps per configuration, and cannot close it.
* **D-A3:** probe results are cached for ten minutes.
* **D-A6:** Chat Completions encodes typed messages and tools, sending the
  same JSON.
* **D-A10:** classifying a 16 KiB error takes under 1 ms, with every
  redaction unchanged.

## Scope

### In scope

| Phase | Decisions | Files (production) |
| :--- | :--- | :--- |
| 0 | records | this PLAN, 0028-MADR, `docs/README.md` |
| 1 | measurements for D-H2 and D-H5 | `llmprovider/live_0028_measure_test.go` (new, `live_gateways`) |
| 2 | D-H1, D-H5 | `llmprovider/internal/wire/post.go`, `llmprovider/internal/transport/transport.go`, `llmprovider/api_error.go`, `llmprovider/retry.go` (comments) |
| 3 | D-H2, D-H3, D-H8 | `llmprovider/api_error.go`, `llmprovider/context_overflow.go` |
| 4 | D-A10 | `internal/redact/redact.go`, `llmprovider/api_error.go`, `llmprovider/context_overflow.go` |
| 5 | D-H10 | `wizard/configure.go` |
| 6 | D-A1 | `llmprovider/catalog/model_metadata.go`, `llmprovider/catalog/discovery.go`, `llmprovider/providers/opencode/opencode.go` |
| 7 | D-A3 | `llmprovider/internal/transport/probe.go`, the five probing providers |
| 8 | D-A6 | `llmprovider/internal/wire/chatcompletions/chatcompletions.go`, `llmprovider/internal/wire/wire.go` |
| 9 | D-A2 | `llmprovider/internal/transport/transport.go` (the manager) |
| 10 | docs, release `v1.4.0` | `docs/architecture.md`, `README.md`, `docs/guides/api-standards.md` if a rule is touched, the amended records |

Each phase adds the tests named in its steps, beside the code they test.

### Out of scope

* The shortlist items the owner did not choose (MADR, More Information).
* Messages, Responses and generateContent typed bodies, unless step 8.6's
  measurement and the owner's answer add them.
* Any change to `WithRetry`'s `MaxDelay` rule, `credentialRefused`'s rule,
  0016-MADR amendment A5's probe default, or a caller's `WithHTTPClient`.
* Consumers: prepare-commit-msg and gobble-cli move under their own records.
* Push and tags, which are the owner's.

## Rules for every phase

1. **Approval.** A phase starts on the owner's "proceed" (or "do phase
   N"). A phase that sends a billed request says so, and how many, when it
   asks.
2. **Red first.** Every new test is run on the tree before the change and
   its FAIL line recorded; then it passes. A test that cannot fail on the
   tree (a guard of existing behaviour) is seen failing on a scratch copy
   with the behaviour planted out. Plants are never made in the tree.
3. **The gate** at the end of every phase that changes Go code:
   * `make pre-add-check`;
   * `CGO_ENABLED=0 go vet ./...` for linux, darwin and windows, with and
     without `-tags live_gateways`;
   * `go test -race -cover ./...` and `go test -shuffle=on ./...`;
   * `go mod tidy -diff`;
   * `make lint parity-check dep-check coverage-check api-check
     generate-check records-check gate-selftest`;
   * markdownlint on the lint scope, relative links, the identifier scan;
   * `go test -count=3 ./llmprovider/... -run TestWireGoldens` without
     `-update`: the goldens are unchanged unless the phase says otherwise.
4. **Coverage floors** (`scripts/coverage-floors.txt`) hold, among them
   `internal/redact` at 100.0 and `wizard` at 86.4. A floor changes only by
   a record.
5. **Benchmarks.** A performance phase records each named benchmark with
   `-benchmem -count=5`, before and after, on the same host, and the
   median of each figure.
6. **Live checks** read keys from the environment, checked for presence
   only, never printed. A deliberately invalid key is the literal
   `invalid-0028-not-a-key`.
7. **Deviations stop and prompt,** with evidence and resolutions, and are
   recorded here as a dated entry, and in the MADR when a decision or
   asserted fact changes, before work continues.
8. **Commits.** One commit per phase, `git commit --no-edit`: the owner's,
   or the agent's on a same-turn "commit to main".
9. **Identifiers.** Nothing committed carries a hostname, an account name or
   a machine path.

## Implementation Steps

### Phase 0: records

1. 0028-MADR is `accepted`, with the owner's answers and the amendment of
   2026-10-08; this PLAN is `proposed`.
2. `docs/README.md` indexes both, 61 records.
3. `make records-check` clean; the owner commits.

### Phase 1: measurements (live; unbilled, plus at most seven 16-token generations if step 2 needs them)

The tables this phase writes are the only source of D-H2's credential codes
and D-H5's headers.

1. **T1, refused credentials.** `TestLive_0028InvalidKeyReplies`, in
   `llmprovider/live_0028_measure_test.go` (`live_gateways`; runs when
   `LLMPROVIDER_LIVE_0028` is set):
   * for each of `openai`, `claude`, `gemini`, `grok`, `together`,
     `huggingface`, `kilo` and `opencode-go`: `providers.New(id,
     WithAPIKey("invalid-0028-not-a-key"), WithModel(catalog.Static(id)[0]))`,
     with a recording transport that keeps the reply's status and the
     first 2 KiB of its body;
   * one `Generate` of `"hi"`;
   * it logs one line per provider: status, `APIError.Code`, `Kind`, and the
     body redacted. It fails only on a transport error.

   T1 records each line. **Rule:** the 403 credential codes of D-H2 are
   the codes in T1 whose status is 403. An empty set is a valid outcome:
   then every 403 is `ErrNotPermitted`.
2. **T2, rate-limit headers.** `TestLive_0028RateLimitHeaders`, same file:
   * for each of `openai`, `claude`, `gemini`, `grok`, `together`,
     `huggingface` and `kilo`, with the owner's key from the environment:
     `ListModels` with probes off, through a recording transport that
     keeps every response header whose lower-cased name starts with
     `x-ratelimit-`, `anthropic-ratelimit-`, `retry-after` or `ratelimit`;
   * a provider whose listing carries none of them gets one `Generate`
     with `WithMaxTokens(16)` (billed), only if the owner approved it at
     the phase's start.

   T2 records header names and value formats, never a key. **Rule:** D-H5
   reads exactly the reset headers T2 shows, with the formats T2 shows; a
   format other than a Go duration or an RFC 3339 time is a deviation.
3. **Recorded as not measured:** an expired credential (MADR amendment of
   2026-10-08), and the headers of a real 429 except Gemini's, captured in
   0027-PLAN D1 (`Retry-After`, no reset header).
4. Gate; record T1 and T2; stage the measurement test.

### Phase 2: retry (D-H1, D-H5)

1. **D-H1 tests, red first** (`llmprovider/internal/wire/post_0028_test.go`
   and `llmprovider/retry_0028_test.go`):
   * `TestPost_FailureAfterWriteIsAfterReply`: an `httptest.Server` that
     reads the body, then sleeps 300 ms; `wire.Post` through a client whose
     `ResponseHeaderTimeout` is 50 ms. Want `transport.IsAfterReply(err)`.
     Today: false.
   * `TestPost_FailureBeforeWriteIsNot`: a server closed before the call
     (connection refused). Want `IsAfterReply(err)` false, and `err` a
     `*url.Error`.
   * `TestWithRetry_HeaderTimeoutSentTwice`, HTTP/1.1 and HTTP/2
     subtests (an `httptest` TLS server with `EnableHTTP2`): the OpenAI
     provider with `WithHTTPClient` (`ResponseHeaderTimeout` 50 ms, the
     server's root CAs) under `RetryPolicy{MaxAttempts: 3, BaseDelay:
     time.Millisecond}`. Want 2 requests received. Today: 3.
2. **D-H1 change, `post.go`.** Before `client.Do`, attach
   `httptrace.ClientTrace{WroteRequest: func(i httptrace.WroteRequestInfo)
   { if i.Err == nil { wrote.Store(true) } }}` with
   `httptrace.WithClientTrace`. When `Do` fails, `Unsendable` is false and
   `wrote.Load()` is true, return `transport.AfterReply(err)`. Update
   `retryable`'s and `WithRetry`'s comments to say so (0028-MADR D-H1).
3. **D-H5 tests, red first:**
   * `TestRateLimitReset` (`llmprovider/internal/transport`), a table over
     T2's headers: `x-ratelimit-reset-requests: 20s` → 20 s; `6m0s` → 6 m;
     `250ms` → 250 ms; both `-requests` and `-tokens` → the longer;
     `anthropic-ratelimit-requests-reset` at a fixed clock + 30 s → 30 s;
     a past time, a negative, an empty or a malformed value → 0. Only the
     header names T2 shows have rows.
   * `TestClassifyHTTPError_429ReadsResetHeader`: a 429 with the reset
     header 20 s → `RetryAfter` 20 s; with `Retry-After: 7` as well →
     7 s; with a body `RetryInfo` of 39 s and the header → 39 s (today's
     order wins; the header is last).
   * `TestWithRetry_ResetPastMaxDelayReturnsAtOnce`: the Together provider,
     a 429 with the header at 60 s, default policy (`MaxDelay` 30 s). Want 1
     request. Today: 3.
4. **D-H5 change.** `transport.RateLimitReset(h http.Header, now time.Time)
   time.Duration`, beside `transport.RetryAfter`, reading T2's headers.
   `ClassifyHTTPError` sets `e.RetryAfter` from it when the status is 429
   and every earlier source gave 0. A 429 with no delay at all keeps the
   backoff.
5. **Proofs on scratch copies:** the `AfterReply` wrap removed: the H1
   tests fail with 3 requests; `RateLimitReset` returning 0: the D-H5 tests
   fail.
6. Gate; record; stage.

### Phase 3: classification (D-H2, D-H3, D-H8)

1. **D-H2 tests, red first** (`llmprovider/api_error_0028_test.go`):
   * `TestClassify_403IsNotPermittedUnlessCredentialCode`: for each of
     `openai`, `claude`, `gemini`, `grok`, `together`, `huggingface` and
     `ollama`, a 403 with
     `{"error":{"type":"request_forbidden","code":"unsupported_country_region_territory","message":"Country, region, or territory not supported"}}`.
     Want `Kind` `ErrNotPermitted`, not retryable, and `errors.Is(err,
     ErrAuthFailure)` still true through the legacy sentinel. For each T1
     credential code (if any), the same 403 carrying it: want
     `ErrAuthFailure`. Today: `ErrAuthFailure` for all.
   * `TestReauth_RegionForbiddenNotRenewed`
     (`internal/wire/reauth_0028_test.go`): the classified region 403
     through `Reauth` with a counting `InvalidatingSource`. Want 1 send, 0
     invalidations. Today: 2 and 1.
   * `TestList_Forbidden403NotRenewed` (`catalog`): a listing answering
     the region 403, with a counting source. Want 1 request, 0
     invalidations, `Catalog.Err` `ErrNotPermitted`.
   * Unchanged: OpenCode's and Kilo's 403 tests; `TestReauth_RerunsOnAuthFailureKind`,
     whose `"403 auth failure"` case builds the kind by hand and so still
     renews.
2. **D-H2 change, `classifyAPIError`.** A case after OpenCode's and Kilo's
   403 rows: `status == http.StatusForbidden && !has(credential403Codes[service]...)`
   → `true, ErrNotPermitted`, where `credential403Codes` is T1's table.
   `statusSentinel` is unchanged, so `Unwrap`'s legacy sentinel keeps a
   403 matching `ErrAuthFailure`.
3. **D-H3 tests, red first** (`llmprovider/context_overflow_0028_test.go`),
   `TestContextOverflow_CompletionAlone`, through `ClassifyHTTPError` for
   `together`, 400:
   * limit 8192, 10 in the messages, 8192 in the completion → `ErrInvalidRequest`,
     not `ErrContextOverflow`, message contains `max_tokens`;
   * limit 8192, 8000 and 1000 → `ErrContextOverflow`;
   * limit 8192, 0 and 9000 → `ErrInvalidRequest`;
   * an overflow message with no split → `ErrContextOverflow`, as today.
4. **D-H3 change.** `completionFillsContext(msg string) bool` in
   `context_overflow.go`: `(?i)maximum context length is (\d+)` and
   `(\d+) in the messages, (\d+) in the completion`; true when both match
   and the completion ≥ the limit. `classifyAPIError`'s overflow case, and
   D-H8's, return `true, ErrInvalidRequest` when it is true, with
   `max_tokens` named in the message.
5. **D-H8 tests, red first:**
   * `TestClassifyStreamFailure_ByType`, a table, code `""`: type
     `invalid_request_error`, plain message → `ErrInvalidRequest`,
     terminal; the same type, an overflow message → `ErrContextOverflow`;
     the same type, the completion-alone message → `ErrInvalidRequest`;
     `authentication_error` → `ErrAuthFailure`; `permission_error` →
     `ErrNotPermitted`; `rate_limit_error` → `ErrRateLimited`, retryable;
     `overloaded_error`, `api_error`, `server_error` →
     `ErrProviderUnavailable`, retryable; `unknown_error` → as today.
   * `TestTogether_TypedErrorInside200SentOnce` (`providers/together`): a
     200 carrying each of the two bodies of the MADR's H8 measurement, under
     `WithRetry`. Want 1 request each, `ErrContextOverflow` and
     `ErrInvalidRequest`. Today: 3 each, `ErrProviderUnavailable`.
6. **D-H8 change, `ClassifyStreamFailure`.** In the default branch, before
   the 500 stand-in: classify by `errType` per step 5's table; and run the
   overflow check (and `completionFillsContext`) for any failure inside a
   200 whose type is `invalid_request_error` or empty.
7. **Existing tests** that pin a typed failure inside a 200 as retryable
   are a deviation: run `go test ./llmprovider/...` after step 6 and
   record any.
8. **Proofs on scratch copies:** the 403 case removed; `completionFillsContext`
   returning false; the type table removed: each set of tests fails.
9. **MADR 0012 §1.1** gains an amendment, dated, naming D-H2's row and
   D-H8's classes.
10. Gate; record; stage.

### Phase 4: classification cost (D-A10)

1. **Reference implementations, frozen in tests.** `internal/redact/redact_ref_test.go`
   holds today's `Redact` as `redactReference` (a copy of `redact.go`'s
   passes, unexported, test-only). `llmprovider/context_overflow_ref_test.go`
   holds today's `contextOverflow` as `contextOverflowReference`.
2. **Benchmarks, before** (committed): `BenchmarkClassify_16KiBTokens`
   (`llmprovider/api_error_bench_0028_test.go`: a JSON 400 whose message is
   `"This request used too many tokens. " + strings.Repeat("detail ", 2300)`),
   `BenchmarkClassify_2KiBOverflow`, and `BenchmarkRedact_16KiBTokens`
   (`internal/redact`). Record the medians; the MADR measured 11.6 ms.
3. **Tests, red first:**
   * `TestClassify_16KiBTokensUnder1ms`: not a timing test; it asserts the
     work bound: `redactInput(msg)` (step 4) returns at most 2048 bytes.
     Today: no such function, compile failure counts as red; on a scratch
     copy with the cut at 16 KiB it fails.
   * `TestRedactInput_NoSecretFragmentAtTheCut`: a message whose
     `sk-proj-` key of 48 characters starts 20 bytes before the 2 KiB
     mark. Want the classified message to contain no substring of 4 or
     more characters of the key. Today's cut at 16 KiB passes it; on a
     scratch copy cutting at exactly 2048 bytes with no separator rule, it
     fails.
4. **The cut.** `redactInput(s string) string` in `api_error.go`: at most
   2048 bytes (`apiErrorRedactLimit` becomes `2 << 10`), cut back to the
   last ASCII space, newline, comma, semicolon or quote at or before 2048,
   and at a rune start. `cutMessage` is replaced by it.
5. **`reKV`, keyword first.** `redact.go`'s `reKV` pass becomes: find each
   keyword occurrence with a literal-alternation regexp of the keywords;
   extend left over `[a-z0-9_-]` to the word start; match the assignment
   and value from there with the remainder of today's expression anchored
   at that start. The pass's replacement function is unchanged.
   * `TestRedact_MatchesReference`: every case of `redact_test.go`, and a
     corpus of 200 generated strings (seed 28) mixing the keywords,
     separators, quotes and secret shapes: `Redact(x) == redactReference(x)`
     byte for byte.
   * `FuzzRedact_MatchesReference`, seeded with that corpus, run 60 s in
     this phase: no difference.
6. **The overflow prefilter.** `contextOverflow` first checks, on the
   lower-cased message, for one literal each pattern requires (the table
   is built per pattern in the code, beside it). The input is the first
   2048 bytes of the message.
   * `TestContextOverflow_PrefilterAdmitsEveryPattern`: each pattern's
     documented sample (`context_overflow_test.go`'s) passes the prefilter
     and matches.
   * `TestContextOverflow_MatchesReference`: every sample and the redaction
     corpus: `contextOverflow == contextOverflowReference`.
7. **Benchmarks, after:** `BenchmarkClassify_16KiBTokens` median under
   1 ms; `BenchmarkRedact_Clean4KiB` (existing) not slower than before by
   more than 10 %. Either missed is a deviation.
8. **0021-MADR Z1** gains an amendment, dated: the redaction bound is 2 KiB,
   cut at a separator.
9. Gate (the `internal/redact` floor is 100.0); record; stage.

### Phase 5: the refused key in `configure` (D-H10)

1. **Tests, red first** (`wizard/configure_0028_test.go`, using
   `fakePrompter` and a listing `roundTripFunc`):
   * `TestConfigureLLM_RefusedTypedKeyIsAskedAgain`: Claude; the listing
     answers 401 `authentication_error` twice, then 200 with one model;
     secrets `["bad-1", "bad-2", "good-3"]`. Want no error,
     `Result.APIKey == "good-3"`, three `Secret` prompts, two notices
     containing `refused`. Today: one prompt, `Result.APIKey == "bad-1"`.
   * `TestConfigureLLM_RefusedTypedKeyThreeTimesFails`: three 401s. Want an
     error matching `llmprovider.ErrAuthFailure`, naming the provider, and a
     zero `Result`.
   * `TestConfigureLLM_RefusedEnvKeyFails`: `AllowEnv`, the environment
     key confirmed, the listing 401. Want an error naming the variable, no
     `Secret` prompt.
   * `TestConfigureLLM_RefusedSavedKeyFails`: `Existing` with Claude's key,
     kept, the listing 401. Want an error saying the saved key was refused,
     no `Secret` prompt.
   * Unchanged: `TestConfigureLLM_EmptyDiscoveryFallsBackToStatic` (a
     listing that fails without a refusal still warns and continues).
2. **The change, `configure.go`.**
   * `resolveAPIKey` returns the key's source as well: `keyTyped`,
     `keyEnvironment` or `keySaved`.
   * `discoverModels` returns the listing's failure with the catalog,
     instead of dropping it.
   * `ConfigureLLM`, after `discoverModels`: when `o.Discover` is set and
     the failure matches `ErrAuthFailure` and not `ErrNotPermitted`: a
     typed key gets a notice ("the service refused this key") and
     `resolveAPIKey`'s `Secret` prompt again, then the listing again, up to
     three keys in all; any other source returns an error naming it. An
     OAuth or vendor-CLI credential refused at listing returns an error
     naming the sign-in.
3. **Proof on a scratch copy:** the refusal branch removed: the four tests
   fail.
4. Gate (the `wizard` floor is 86.4); record; stage.

### Phase 6: OpenCode's metadata (D-A1)

1. **Tests, red first:**
   * `TestCachedMetadataWith_ColdStartsFetchWithoutWaiting` (`catalog`): a
     metadata server sleeping 450 ms. Want `CachedMetadataWith` to return
     `false` within 50 ms; after the fetch completes (wait on the test's
     fetch hook, not a sleep), `true` and the document.
   * `TestLoadModelMetadata_RevalidatesWithETag`: a server answering
     `ETag: "v1"` and the document, then 304 to `If-None-Match: "v1"`. At
     the TTL plus 1 s (the package's clock seam): want one request carrying
     `If-None-Match: "v1"`, the same document, a renewed fetch time, and
     the decode count unchanged (a counter seam in `decodeModelMetadata`).
   * `TestDecodeModelMetadata_StreamedLimit`: a body of `metadataLimit + 1`
     bytes → an error naming the limit; a valid body decodes as today.
   * `TestOpencode_ColdCacheUsesTableRoute` (`providers/opencode`): a model
     in the route table whose metadata `provider.npm` names a different
     route; the metadata server sleeps 2 s. The first `Generate` returns
     within 200 ms on the table's route; after the fetch, the next uses the
     metadata's.
   * `TestOpencode_UnknownModelWaitsForMetadata`: a model not in the table:
     the first request waits for the document and uses its route.
2. **Changes.**
   * `catalog.CachedMetadataWith(ctx, id, opts...) (Metadata, bool)`, an
     exported addition: the cached document, fresh or stale, without
     waiting; with none, it starts the fetch and reports `false`.
   * The cache entry keeps `etag`; `fetchModelMetadata` sends
     `If-None-Match` when the entry has a document and an `etag`; a 304
     keeps the document and renews `fetched`.
   * `decodeModelMetadata` decodes with `json.NewDecoder` over a reader
     that fails past `metadataLimit`.
   * `opencode.requestRoute`: pinned route first; then
     `CachedMetadataWith`; on `false`, the table's route if the table has
     the model, else `LookupMetadataWith` as today.
3. **Fixture change, recorded:** `TestOpencode_RoutesFromMetadata`'s cases
   warm the cache (`LookupMetadataWith`, awaited) before `Generate`, so
   they keep asserting the metadata route. Its assertions do not change.
4. **Benchmarks:** `BenchmarkDecodeModelMetadata` over a generated
   document (deterministic, seed 28, about 5 MB, in the models.dev shape),
   before and after: allocations and peak bytes recorded.
5. **Proofs on scratch copies:** `requestRoute` back to
   `LookupMetadataWith`: the cold test fails; the 304 branch removed: the
   ETag test fails.
6. **MADR 0012 §3.1** gains an amendment, dated (D-A1).
7. Gate (`api-check` shows the one addition); record; stage.

### Phase 7: probe results (D-A3)

1. **Tests, red first** (`llmprovider/internal/transport/probe_0028_test.go`
   and `providers/openai`):
   * `TestListModels_SecondListingSendsNoProbes`: the OpenAI provider with
     probes on, against an `httptest` server that lists 3 models and counts
     generation requests. Two `ListModels` calls: want 3 generations, then
     0. Today: 3 and 3.
   * `TestProbeCache_KeyedByCredentialAndBaseURL`: a second provider with a
     different key, and one with a different base URL: each probes again.
   * `TestProbeCache_ExpiresAfterTTL`: the cache's clock seam at 10 min +
     1 s: probes again.
   * `TestProbeCache_EmptyResultNotCached`: every probe fails: the next
     listing probes again.
   * `TestProbeCache_FingerprintOnly`: the cache holds no key: its keys are
     `sha256` hex of the token value.
2. **Change.** `transport.ProbeGenerateHealthCached(ctx, key ProbeKey,
   preferred, limit, generate)`, with `ProbeKey{Provider, BaseURL,
   Credential string}`, a process-wide map under a mutex, TTL 10 min, a
   non-empty result cached. Each of OpenAI, Claude, Gemini, Grok and Ollama
   calls it, with `Credential` the `sha256` of the token its `src` gives
   (Ollama: the empty string, as it has no credential).
3. **Proof on a scratch copy:** the cache bypassed: the first test fails
   (3 and 3).
4. Gate; record; stage.

### Phase 8: Chat Completions bodies (D-A6)

1. **Reference, frozen in tests.** `chatcompletions_ref_test.go` holds
   today's `itemsToChatMessagesReplaying` and `toolList` as references.
2. **Tests, red first:**
   * `TestChatMessages_SameJSONAsReference`: a corpus of item sequences
     (seed 28, 300 cases: user, assistant, system messages; calls joining an
     assistant turn and opening one; outputs; reasoning with and without a
     replay field, with and without details) and every wire-case input:
     `json.Marshal` of the new messages and of the reference decode to equal
     values. Today: there is no typed encoder, so the new path is the
     reference and the test passes; it is red on a scratch copy whose typed
     encoder drops `content` on a call message.
   * `TestToolList_SameJSONAsReference`, likewise, including a nil schema.
   * `BenchmarkPlainMessages` and `BenchmarkBodyMarshal` (the MADR's,
     committed): want allocations cut at least fivefold for 100 plain
     messages.
3. **Change.** `chatMessage{Role string \`json:"role"\`; Content string
   \`json:"content"\`; ToolCalls []chatToolCall \`json:"tool_calls,omitempty"\`;
   ToolCallID string \`json:"tool_call_id,omitempty"\`; ReasoningDetails
   []json.RawMessage \`json:"reasoning_details,omitempty"\`; replayField,
   replay string}` with a `MarshalJSON` that adds `replayField: replay`
   only when `replayField` is set; `chatTool` and `chatToolCall` likewise.
   `Body` keeps returning `map[string]any`, with `messages` and `tools`
   typed, so Kilo's `body["provider"]` still works.
4. **`ToolArguments`** compacts once: `json.Compact` into a buffer; on
   error, today's fallback. `TestToolArguments_*` unchanged.
5. **The other wires, measured only.** `BenchmarkPlainMessages`'s
   counterpart for Messages (`messages.FromItems`), Responses
   (`responses.Input`) and generateContent (`generatecontent.Contents`),
   maps against a typed prototype in the test. A wire whose typed prototype
   is at least 3 times faster and cuts allocations at least tenfold is
   reported to the owner as a candidate (a scope change); the others are
   recorded as measured, not done.
6. Gate (the goldens unchanged); record; stage.

### Phase 9: the transport manager (D-A2)

1. **Tests, red first** (`llmprovider/internal/transport/manager_0028_test.go`):
   * `TestDefaultClient_SharesOneTransport`, HTTP/1.1 and HTTP/2 (TLS,
     `EnableHTTP2`, the test server's root CAs given to the manager's
     default configuration through a test seam): two `DefaultClient()`
     clients make one call each, one after the other. Want 1 connection.
     Today: 2.
   * `TestDefaultClient_CloseIdleConnectionsIsScoped`: client A calls
     `CloseIdleConnections`; client B's next call reuses the idle
     connection. Want 1 connection in all.
   * `TestManager_OneTransportPerConfig`: the same configuration twice →
     the same transport; two differing configurations → two.
   * `TestDefaultClients_GoroutinesBounded`: 20 clients, one call each,
     then `runtime.NumGoroutine()` within the baseline plus 8 (0020-MADR
     F28's measure).
   * Unchanged: `TestResolveOptions_Defaults` (distinct clients) and
     `auth`'s transport tests.
2. **Change.** In `transport.go`: `type Config struct` (the comparable
   settings `DefaultClient` sets today, `MaxIdleConnsPerHost` 16);
   `manager.transport(Config) *http.Transport`, created once per `Config`
   under a mutex; `DefaultClient()` returns `&http.Client{Transport:
   scoped{manager.transport(defaultConfig)}}`, where `scoped` implements
   `RoundTrip` only, so `http.Client.CloseIdleConnections` finds no
   `CloseIdleConnections` to call. `catalog.listingClient` is unchanged in
   code and now rides the shared transport.
3. **Proof on a scratch copy:** `manager.transport` returning a new
   transport each call: the first test fails (2 connections).
4. **0016-MADR D8** gains an amendment, dated (D-A2).
5. Gate; record; stage.

### Phase 10: documentation and release

1. **`docs/architecture.md`:** the transport manager, the probe cache, the
   metadata revalidation and cold-cache routing, the redaction bound.
2. **`docs/guides/api-standards.md`:** only if a rule's text no longer
   holds (none is expected); `catalog.CachedMetadataWith` follows its
   naming rules.
3. **Release notes** in this PLAN's Execution Record: every behaviour
   change, by decision, as `v1.4.0`; `README.md`'s Status names `v1.4.0`.
4. **The live suites** on the release commit, as 0026-PLAN D11 ran them:
   `LLMPROVIDER_LIVE_TOGETHER=1 go test -tags live_gateways -count=1 -v
   ./llmprovider/... -run Live`; a failure is a deviation, a transient one
   rerun once alone.
5. **The owner** commits, pushes, and tags `v1.4.0`.
6. **After the tag:** CI on the tag; `go list -m …@v1.4.0` through the
   proxy; apidiff between `v1.3.2` and `v1.4.0`, as 0026-PLAN's Phase 7
   ran it: additions only; a scratch consumer using every public package,
   and a scratch copy of prepare-commit-msg, built and tested against
   `v1.4.0`.
7. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** for every test the phases name, with both lines
  in the Execution Record; every plant's FAIL line.
* **V2. The gate** clean at the end of each phase, the goldens unchanged.
* **V3. Measurements:** T1 and T2 recorded before Phases 2–3 use them.
* **V4. Benchmarks:** A1, A6 and A10's before and after medians; the
  targets of steps 4.7, 6.4 and 8.2 met.
* **V5. API:** `api-check` against `v1.3.2` shows additions only:
  `catalog.CachedMetadataWith`, and nothing else unless recorded.
* **V6. Live:** the live suites on the release commit.
* **V7. Records:** the amendments to MADR 0012 §1.1 and §3.1, 0016-MADR D8
  and 0021-MADR Z1 are made in their phases.

## Rollout and Rollback

* **Rollout.** Ten commits after the records, one per phase; then
  `v1.4.0`. Consumers move under their own records; prepare-commit-msg's
  next adoption record cites this one.
* **Rollback, before the tag:** each phase reverts alone, newest first.
  Phases 3 and 4 change the same classification functions, so Phase 3
  reverts only after Phase 4; Phases 5–9 depend on no other.
* **Rollback, after the tag:** fix forward in `v1.4.x`. A consumer that
  meets a defect pins `v1.3.2`; nothing on disk changes format (the probe
  and metadata caches are in memory).

## Execution Record

### Phase 0: records (2026-10-08)

* 0028-MADR `accepted` ("Acvept the madr"), with the owner's answers and
  its amendment of 2026-10-08; this PLAN; `docs/README.md`, 61 records.
* `records-check`: `61 records, 0 problem(s)`; relative links: `120
  relative link(s), 0 problem(s)`; markdownlint on `docs/README.md`: 0
  issues; the identifier scan of the three files: 0 hits.
* **Approval.** "Commit to main. Proceed. Approved to run the billed
  tests", 2026-10-08: Phase 1 may send step 2's billed fallback, at most
  seven 16-token generations.
* **Commit.** By the agent, to `main`, on that ask, with `git commit
  --no-edit`.

### Phase 1: measurements (2026-10-08)

* **Before it.** Phase 0 committed by the agent as `e150455`, on the
  owner's ask.
* **The test.** `llmprovider/live_0028_measure_test.go`
  (`live_gateways`, switched on by `LLMPROVIDER_LIVE_0028`):
  `TestLive_0028InvalidKeyReplies` (T1) and `TestLive_0028RateLimitHeaders`
  (T2). Keys are read from the environment, checked for presence only; the
  test logs statuses, codes, kinds, redacted bodies, and rate-limit header
  names and values, never a key.
* **Proof.** On a scratch copy whose recorder refuses every request:
  `live_0028_measure_test.go:108: no reply: Post "https://api.openai.com/v1/responses": planted: refused`,
  `--- FAIL`.
* **Run:** `LLMPROVIDER_LIVE_0028=1 go test -tags live_gateways -count=1 -v
  -run TestLive_0028 ./llmprovider/`: `ok … 18.371s`. Billed: seven
  16-token generations, step 2's fallback, as approved.

#### T1: replies to a key that is not a key (unbilled)

| Provider | Status | `APIError.Code` | Kind | Body (redacted) |
| :--- | ---: | :--- | :--- | :--- |
| openai | 401 | `invalid_api_key` | `ErrAuthFailure` | OpenAI's error object |
| claude | 401 | `authentication_error` | `ErrAuthFailure` | `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},…}` |
| gemini | 400 | `API_KEY_INVALID` | `ErrAuthFailure` | the Interactions array (0026-PLAN D12) |
| grok | 400 | `invalid-argument` | `ErrInvalidRequest` | `{"code":"[REDACTED]","error":"Incorrect API key provided. …"}` |
| together | 401 | `invalid_api_key` | `ErrAuthFailure` | `{… "type":"invalid_request_error", "code":"invalid_api_key"}` |
| huggingface | 401 | (none) | `ErrAuthFailure` | `{"error":"Invalid username or password."}` |
| kilo | 401 | `INVALID_TOKEN` | `ErrAuthFailure` | `{"error":{"code":"[REDACTED]","message":"Your authentication token is invalid. …"},"error_type":"authentication_required"}` |
| opencode-go | 401 | `AuthError` | `ErrAuthFailure` | `{"type":"error","error":{"type":"AuthError","message":"Invalid API key."}}` |

**Applying step 1's rule:** no provider answered a refused credential with
403, so D-H2's 403 credential-code table is **empty**: every 403 on the
seven direct providers becomes `ErrNotPermitted`. An expired credential is
not measured (MADR amendment of 2026-10-08).

#### T2: rate-limit headers

| Provider | On the listing (unbilled) | On one 16-token generation (billed) |
| :--- | :--- | :--- |
| openai | none | `x-ratelimit-limit-requests`, `-limit-tokens`, `-remaining-requests`, `-remaining-tokens`; `x-ratelimit-reset-requests: 12ms`, `x-ratelimit-reset-tokens: 0s` |
| claude | none | `anthropic-ratelimit-{requests,tokens,input-tokens,output-tokens}-{limit,remaining}`; `anthropic-ratelimit-{requests,tokens,input-tokens,output-tokens}-reset: 2026-10-08T20:25:21Z` |
| gemini | none | none |
| grok | none | `x-ratelimit-limit-requests`, `-limit-tokens`, `-remaining-requests`, `-remaining-tokens`; no reset |
| together | none | none |
| huggingface | none | `x-ratelimit-limit-*`, `-remaining-*`; `x-ratelimit-reset-requests: 100ms`, `x-ratelimit-reset-tokens: 1ms` |
| kilo | none | none |

**Applying step 2's rule:** D-H5 reads exactly these reset headers:

* `x-ratelimit-reset-requests` and `x-ratelimit-reset-tokens`, as Go
  durations (OpenAI, Hugging Face);
* `anthropic-ratelimit-requests-reset`, `-tokens-reset`,
  `-input-tokens-reset` and `-output-tokens-reset`, as RFC 3339 times
  (Claude).

Both formats are the two step 2 allows, so there is no deviation. The
MADR's D-H5 named the requests and tokens resets of Claude; T2 adds its
input- and output-token resets, which the rule takes in. Gemini, Grok,
Together and Kilo send no reset header on a 200; a 429's headers are not
measured beyond Gemini's (0027-PLAN D1: `Retry-After`).

**Found, outside this PLAN (not done):**

* **Grok's refused key is `ErrInvalidRequest`.** xAI answers an invalid key
  with 400 `invalid-argument`, "Incorrect API key provided", which the table
  classifies by its status. So `Reauth` does not renew a refused Grok
  credential, and a caller that stops on `ErrAuthFailure`, as
  prepare-commit-msg does, tries every fallback model with the same key:
  the defect 0026's F6 and D12 fixed for Gemini.
* **Redaction masks a `code` value.** Grok's and Kilo's bodies show
  `"code":"[REDACTED]"`: `reKV`'s `code` keyword takes a diagnostic code
  for a secret. Only the logged text is affected; classification reads the
  raw body, and `APIError.Code` keeps the value.

#### Gate

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `463 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -cover`, `-shuffle=on`, `go mod tidy -diff`, `make lint` | 0 each |
| `parity-check`, `dep-check`, `generate-check`, `gate-selftest` | 0 each |
| `coverage-check` | `28 packages, 0 problem(s)` |
| `api-check` | `against v1.3.2, 0 incompatible change(s) outside llmprovider/x/` |
| `records-check` | `61 records, 0 problem(s)` |
| markdownlint, G-wire stable, links | 0 problems; 616 relative links in 71 files |
| identifier scan of the new file | 0 hits |

### Deviation D1 (2026-10-08): Grok's refused key is classified as an invalid request (H11)

* **Found** in Phase 1's T1: xAI answers a key that is not a key with HTTP
  400, `{"code":"invalid-argument","error":"Incorrect API key provided. You
  can obtain an API key from https://console.x.ai."}`, which the table
  classifies by status as `ErrInvalidRequest`. `Reauth` does not renew it,
  and a caller that stops on `ErrAuthFailure` tries every fallback with the
  same key.
* **Options put to the owner:** add it to 0028, in Phase 3; a later record;
  or leave it.
* **Decision.** "Add to 0028, Phase 3". 0028-MADR's amendment adds **H11**.
* **Added to Phase 3,** after step 2:
  * **Test, red first,** `TestClassify_GrokRefusedKey`
    (`llmprovider/api_error_0028_test.go`): T1's body, as a 400 for `grok`.
    Want `Kind` `ErrAuthFailure`, terminal. Today: `ErrInvalidRequest`.
    And `TestClassify_GrokInvalidArgumentStaysInvalid`: a 400 with code
    `invalid-argument` and a message that does not name an API key (`"Invalid
    request content: max_output_tokens must be positive"`): want
    `ErrInvalidRequest`, as today.
  * **`TestReauth_GrokRefusedKeyRenews`:** T1's body through the Grok
    provider with a counting `InvalidatingSource`: want 2 sends and 1
    invalidation. Today: 1 and 0.
  * **Change,** in `classifyAPIError`: `service == "grok" && status == 400
    && has("invalid-argument") && message contains "API key"` (case
    folded) → `true, ErrAuthFailure`, beside `geminiAuthReasons`'s row.
  * **Proof on a scratch copy:** the row removed: both red tests fail.
* **Files:** none beyond Phase 3's.

### Deviation D2 (2026-10-08): redaction masks diagnostic codes (A10b)

* **Found** in Phase 1's T1: Grok's `"code":"invalid-argument"` and Kilo's
  `"code":"INVALID_TOKEN"` are logged as `"code":"[REDACTED]"`.
  `reKVLong`'s bare `code` key keeps a value only when `reDiagnostic`
  (`internal/redact/redact.go:66`, `^(?:\d+|[a-z]+(?:_[a-z]+)+)$`, 0021-MADR
  Z2) matches it, and that takes lower-case snake case only.
* **Options put to the owner:** a step in Phase 4 after the rewrite is
  proven equivalent; a later record; or leave it.
* **Decision.** "Phase 4, after equivalence". 0028-MADR's amendment adds
  **A10b**.
* **The rule, narrower than the question put it.** The question offered
  "letters, digits, hyphens and underscores, 8–40 characters, no digit run
  of 8 or more". Checked before recording, that would leave a 32-character
  hexadecimal token unmasked, whose digit runs are short. The rule recorded
  admits **letters only**: `^(?:\d+|[A-Za-z]+(?:[-_][A-Za-z]+)+)$`, at most 40
  bytes. It keeps `invalid-argument`, `INVALID_TOKEN` and today's snake case;
  no value with a digit and a letter is kept. It is narrower than what the
  owner approved, never wider.
* **Added to Phase 4,** as step 5b, after step 5's equivalence holds:
  * **Test, red first,** `TestRedact_KeepsDiagnosticCodes`: T1's Grok and
    Kilo bodies keep their codes; `{"code":"model-not-found"}` keeps it.
    Today: all three masked.
  * **`TestRedact_StillMasksSecretShapedCodes`:** `code` values of a
    32-character hexadecimal string, `sk-proj-` plus 40 characters, a
    base64url string of 43 characters, `abcd-1234-efgh-5678`, and a
    41-byte letters-and-hyphens string: all masked.
  * **Change:** `reDiagnostic` as above, with the length bound in
    `redactKV` and `redactKVLong`. `TestRedact_MatchesReference` then
    excludes, by name, the corpus cases this step changes, and asserts
    exactly those differ.
  * **Proof on a scratch copy:** `reDiagnostic` back to today's: the first
    test fails.
* **Files:** none beyond Phase 4's.

### Deviation D3 (2026-10-08): key formats researched; redaction and the wizard code for them (A10c, H10b)

* **Asked by the owner** after D2: "search the public repos for all of the
  providers we support and get the style, format, and length of their
  api-keys and tokens so we can actually code for them", then "do the
  research we need to enhance this heuristic".
* **Research, 2026-10-08,** read-only; no provider was sent a request:
  * **Sources:** gitleaks `config/gitleaks.toml` (MIT; master 09242ce9,
    2025-11-20; release v8.30.1); betterleaks
    `cmd/generate/config/rules/*.go` (MIT; main 30c423ca); Nosey Parker and
    Kingfisher rules as kept in praetorian-inc/titus (Apache-2.0;
    ccae0195); GitHub's "Supported secret scanning patterns"; trufflehog
    detectors (AGPL-3.0; cited, never copied); vendor code (openai/codex
    515c291d; xai-org/grok-build 2bdd1d6a and xai-sdk-python's
    `.gitleaks.toml`; google-gemini/gemini-cli; huggingface/huggingface_hub;
    anthropics/claude-cookbooks; Kilo-Org/kilocode 97472345 and
    Kilo-Org/cloud 1127c407; anomalyco/opencode `packages/console/core/src/key.ts`
    ac2fa668); vendor documentation (Google API keys, Gemini API keys,
    Google OAuth 2.0, Anthropic authentication and Admin API, Hugging Face
    tokens, OpenRouter keys, xAI inference).
  * **No vendor publishes a full format.** Prefixes are documented; lengths
    and character sets come from scanner rules, their fixtures and vendor
    code.
* **Measured on the owner's keys,** shape only (`key_shapes.py` and
  `key_match.py`, scratch; lengths, character classes and allow-listed
  prefixes printed, never a key), each matching its researched shape:

  | Variable | Shape |
  | :--- | :--- |
  | `OPENAI_API_KEY` | `sk-proj-` + 74 + `T3BlbkFJ` + 74, `[A-Za-z0-9_-]`, 164 in all |
  | `ANTHROPIC_API_KEY` | `sk-ant-api03-` + 93 + `AA`, 108 in all |
  | `GEMINI_API_KEY` | `AIzaSy` + 33, 39 in all |
  | `XAI_API_KEY` | `xai-` + 80 `[A-Za-z0-9]`, 84 in all |
  | `TOGETHER_API_KEY` | `tgp_v1_` + 43 `[A-Za-z0-9_-]`, 50 in all |
  | `HF_TOKEN` | `hf_` + 34 letters, 37 in all |
  | `KILO_API_KEY` | a JWT, three base64url segments |
  | `OPENCODE_API_KEY` | `sk-` + 64 `[A-Za-z0-9]`, 67 in all |

* **Today's redaction, measured** (a scratch test over `redact.String`):
  * **masks ordinary text:** `sk-learn-tutorial-for-beginners`,
    `xai-grok-login-device-code-flow-tests`,
    `(id)hf_requiredCharacteristicTypesForDisplayMetadata`;
  * **misses:** a Gemini auth key `AQ.Ab8RN6…` (Google: "Starting May 28,
    2026, all new API keys created in Google AI Studio are automatically
    created as auth keys"), a Google refresh token `1//0g…`, a Hugging Face
    org token `api_org_…`; a Google access token `ya29.…` is masked only
    when the word "token" precedes it (`reAuth`), not by its shape.
* **The wizard** (`wizard/auth.go:378-380`) does not catch an OpenRouter
  `sk-or-` key, an OpenCode `sk-` + 64 key, a Gemini `AQ.` key or a Hugging
  Face `api_org_` token pasted for OpenAI.
* **Options put to the owner:** Phase 4 and Phase 5; Phase 4 only; a later
  record.
* **Decision.** "Phase 4 + Phase 5". 0028-MADR's amendment adds **A10c**
  and **H10b**.

**Added to Phase 4, as step 5c, after step 5b (A10c).** The principle:
shapes add precision, and nothing masked today goes unmasked unless a
tighter rule masks it.

1. **Shape rows,** each with its source in a comment, in `reToken` or, where
   a check is needed after the match, in a new pass with a replace
   function:

   | Row | Pattern | Source |
   | :--- | :--- | :--- |
   | OpenAI legacy | `\bsk-[A-Za-z0-9]{20}T3BlbkFJ[A-Za-z0-9]{20}\b` | gitleaks; codex |
   | OpenAI typed | `\bsk-(?:proj\|svcacct\|admin\|None)-[A-Za-z0-9_-]{16,}` | gitleaks; codex; OpenAI OpenAPI spec |
   | Anthropic | `\bsk-ant-[a-z]{2,8}[0-9]{2}-[A-Za-z0-9_-]{16,}` | Anthropic docs (`api03`, `admin01`, `oat01`); gitleaks |
   | OpenRouter | `\bsk-or-v1-[0-9a-f]{64}\b` | OpenRouter docs |
   | OpenCode | `\bsk-[A-Za-z0-9]{64}\b` | opencode `key.ts` |
   | `sk-` fallback | `\bsk-[A-Za-z0-9_-]{32,}`, masked only if the body holds a letter and a digit | replaces today's `\bsk-[A-Za-z0-9_-]{16,}` |
   | Google API key | `\bAIza[0-9A-Za-z_-]{35}` (unchanged) | Google docs; gitleaks |
   | Gemini auth key | `\bAQ\.Ab[0-9A-Za-z_-]{30,}` | Gemini docs (prefix, forum); betterleaks |
   | Google access token | `\bya29\.[0-9A-Za-z_-]{20,}` | Google OAuth docs; Nosey Parker |
   | Google refresh token | `\b1//0[0-9A-Za-z_-]{40,}` | Google OAuth docs (`1//`); `0` avoids Python floor division |
   | xAI | `\bxai-[A-Za-z0-9]{80}\b` | trufflehog; measured |
   | `xai-` fallback | `\bxai-[A-Za-z0-9_-]{32,}`, masked only if the body holds a letter and a digit | replaces today's `\bxai-[A-Za-z0-9_-]{16,}` |
   | Together | `\btgp_v1_[A-Za-z0-9_-]{43}`; today's `\btgp_[A-Za-z0-9_-]{16,}` kept | betterleaks; measured |
   | Hugging Face | `\bhf_[A-Za-z0-9]{34}\b`, `\bapi_org_[A-Za-z0-9]{34}\b` | gitleaks; trufflehog (digits); replaces `\bhf_[A-Za-z0-9]{16,}` |
   | Ollama | `\b[0-9a-f]{32}\.[A-Za-z0-9_-]{24}\b`, only when the text holds `ollama` | betterleaks; Kingfisher |

   The pass anchors gain `aq.ab`, `ya29.`, `1//0`, `api_org_` and `ollama`.
2. **Tests, red first** (`internal/redact/redact_shapes_0028_test.go`):
   * `TestRedact_KeyShapes`: for each row, a synthetic value built to its
     published shape (never a real key), inside a sentence, in JSON and
     after `Bearer`: masked whole. Today: the `AQ.`, `1//0`, `api_org_`,
     bare `ya29.` and Ollama cases are not masked.
   * `TestRedact_NotKeys`: `sk-learn-tutorial-for-beginners`,
     `xai-grok-login-device-code-flow-tests`,
     `(id)hf_requiredCharacteristicTypesForDisplayMetadata`, `FAQ.md`,
     `x = a 1//long_python_variable_name_over_forty_characters`, and an MD5
     hash, a dot and a 24-letter word without `ollama`: unchanged. Today:
     the first three are masked.
   * `TestRedact_MatchesReference` keeps excluding, by name, only the cases
     A10b and A10c change, and asserts exactly those differ.
   * `redact_test.go:234`'s `tgp_v1_` + 24 fixture stays masked through
     the kept `tgp_` row.
3. **Proof on a scratch copy:** each new row removed in turn: its
   positive case fails; the letter-and-digit check removed: the
   `sk-learn` and `xai-grok` negatives fail.
4. The `internal/redact` floor stays 100.0.

**Added to Phase 5, as step 2b (H10b).**

1. **`foreignKeyPrefixes`** gains `sk-or-` ("an OpenRouter"), `AQ.`
   ("a Google Gemini"), and `api_org_` ("a Hugging Face"); `sk-ant-` stays
   first, `sk-or-` before any bare `sk-` rule.
2. **An OpenCode key pasted for OpenAI:** `sk-` + 64 `[A-Za-z0-9]` with no
   `T3BlbkFJ` is named as "an OpenCode" key.
3. **The chosen provider's shape:** after a typed key, a key that does not
   match the provider's confirmed shape gets a warning notice naming the
   expected shape; the wizard continues (H10's listing check then decides).
   Confirmed shapes: OpenAI (`T3BlbkFJ` typed or legacy), Anthropic
   (`sk-ant-api03-` or `sk-ant-admin01-` + 93 + `AA`), Gemini (`AIza` + 35,
   or `AQ.` prefix), Together (`tgp_v1_` + 43), Hugging Face (`hf_` + 34),
   OpenCode (`sk-` + 64). Prefix only, where the length is unconfirmed:
   xAI (`xai-`). Kilo (a JWT) and Ollama: no check.
4. **Tests, red first** (`wizard/auth_shapes_0028_test.go`):
   `TestForeignKey_Researched` (each new prefix, and the OpenCode shape for
   OpenAI, is named; today: not); `TestProviderShapeWarning` (a malformed
   key for each confirmed provider warns and continues; a well-shaped
   synthetic key does not warn). Proof on a scratch copy: the new entries
   removed: the tests fail.
