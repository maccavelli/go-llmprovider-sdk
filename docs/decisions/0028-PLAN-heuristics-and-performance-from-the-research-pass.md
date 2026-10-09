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

### Deviation D4 (2026-10-08): 0020's dropped-connection test meets D-H1

* **Found** at Phase 2's gate, after D-H1's change: `go test -race` (and
  `-shuffle`, `coverage-check`, `gate-selftest`, `pre-add-check`, all for
  the same test) failed
  `TestGenerate_AnsweredOnceIsNotBoughtAgain`
  (`llmprovider/providers/together/reply_test.go:76-85`, 0020-MADR F9 Q2 a):
  `reply_test.go:65: a dropped connection: 2 request(s), want 3: a network
  failure is retried`. Its server receives the whole request, then hijacks
  and closes the connection before any reply: a failure after the request
  was written, which D-H1 marks and resends at most once.
* **Pre-existing:** the test predates 0028; it encodes 0020's rule, which
  D-H1 refines for failures after the write.
* **Options put to the owner:** the test follows D-H1, keeping 0020's
  retry for a failure before the write; or D-H1 narrows to header timeouts
  alone, and the test stays.
* **Decision.** "Test follows D-H1". The dropped-connection case wants 2
  requests; a new case, a refused connection (the request never reached
  the service), wants 3 attempts, so 0020 F9's "a network failure is
  retried" keeps its coverage where it still holds. 0028-MADR names 0020-MADR
  F9 as refined.
* **Added to Phase 2:** `llmprovider/providers/together/reply_test.go`.

### Phase 2: retry (2026-10-08)

* **Before it.** Phase 1 and D1–D3 committed by the owner as `7a76323`.
* **Approval.** "committed, proceed", 2026-10-08.

**D-H1, red first:**

```text
--- FAIL: TestPost_FailureAfterWriteIsAfterReply (0.30s)
    post_0028_test.go:32: Post = Post "http://127.0.0.1:…": net/http: timeout awaiting response headers; want a failure marked after the reply, since the request was written
--- FAIL: TestWithRetry_HeaderTimeoutSentTwice/HTTP/1.1 (0.42s)
    retry_0028_test.go:52: the service received 3 request(s), error llmprovider: openai failed after 3 attempts: Post "https://127.0.0.1:…/responses": net/http: timeout awaiting response headers; want 2 and an error
--- FAIL: TestWithRetry_HeaderTimeoutSentTwice/HTTP/2 (0.17s)
    retry_0028_test.go:52: the service received 3 request(s), error … http2: timeout awaiting response headers; want 2 and an error
```

`TestPost_FailureBeforeWriteIsNot` passed on the tree (a guard).

**D-H1, the change.** `wire.Post` traces the request with
`httptrace.ClientTrace.WroteRequest`; a `client.Do` failure after a clean
write, not `Unsendable`, is returned as `transport.AfterReply(err)`.
`AfterReply`'s, `retryable`'s and `WithRetry`'s comments say so.

**D-H5, red first:**

```text
llmprovider/internal/transport/ratelimit_0028_test.go:46:13: undefined: RateLimitReset
--- FAIL: TestClassifyHTTPError_429ReadsResetHeader (0.00s)
    api_error_ratelimit_0028_test.go:24: 429 with the reset header: RetryAfter = 0s; want 20s
    api_error_ratelimit_0028_test.go:39: Claude's 429 with its reset 30 s ahead: RetryAfter = 0s; want about 30s
--- FAIL: TestWithRetry_ResetPastMaxDelayReturnsAtOnce (0.01s)
    retry_0028_test.go:78: the service received 3 request(s), error llmprovider: together failed after 3 attempts: … HTTP 429 rate_limit_exceeded: Rate limit reached; want 1 and the rate limit
```

**D-H5, the change.** `transport.RateLimitReset(h, now)` reads T2's six
headers, `X-Ratelimit-Reset-{Requests,Tokens}` as Go durations and
`Anthropic-Ratelimit-{Requests,Tokens,Input-Tokens,Output-Tokens}-Reset` as
RFC 3339 times, and returns the longest wait ahead. `ClassifyHTTPError`
uses it last, for a 429 only. The comment above `RetryInfo`'s fallback no
longer says "Gemini sends no Retry-After" (0027-MADR's capture).

**Green:** the four touched packages pass in full; then D4 (above), and the
updated `TestGenerate_AnsweredOnceIsNotBoughtAgain` passes.

**Proofs on scratch copies** (`p28_plants2.py`, `p28_plant_d4.py`):

| Plant | Result |
| :--- | :--- |
| the `AfterReply` wrap removed | `TestPost_FailureAfterWriteIsAfterReply` FAIL; `TestWithRetry_HeaderTimeoutSentTwice` FAIL, 3 requests on HTTP/1.1 and HTTP/2 |
| every failure marked (`wrote.Load() \|\| true`) | `TestPost_FailureBeforeWriteIsNot` FAIL, `connection refused; want an unmarked *url.Error`; `TestGenerate_AnsweredOnceIsNotBoughtAgain` FAIL, `a refused connection: 2 attempt(s), want 3` |
| `RateLimitReset` returning 0 | `TestRateLimitReset` FAIL (each duration row); `TestClassifyHTTPError_429ReadsResetHeader` FAIL; `TestWithRetry_ResetPastMaxDelayReturnsAtOnce` FAIL, 3 requests |

**The gate** (`p26_gate.py`), after D4:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `467 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -cover`, `-shuffle=on`, `go mod tidy -diff`, `make lint` | 0 each |
| `parity-check` | `409 identifiers, 409 rows, 409 with an SDK equivalent, 0 problem(s)` |
| `dep-check`, `generate-check` | 0 each |
| `coverage-check` | `28 packages, 0 problem(s)` |
| `api-check` | `against v1.3.2, 0 incompatible change(s) outside llmprovider/x/` |
| `records-check` | `61 records, 0 problem(s)` |
| `gate-selftest` | `Ran 14 tests`, OK |
| markdownlint, G-wire stable, links | 0 problems; 616 relative links in 71 files |
| identifier scan of the changed files | 0 hits in 12 files |

The first gate run, before D4, failed five checks on the one test D4
records; nothing else failed.

### Deviation D5 (2026-10-08): a ChatGPT session's 403 was refreshed by an existing test (pending a measurement)

* **Found** in Phase 3, after D-H2's change: `go test ./llmprovider/...`
  failed `TestListModels_ChatGPTFailureIsAnAPIError`
  (`providers/openai/chatgpt_listing_errors_test.go:60-77`, 0020-MADR F38):
  `listing/refresh calls = 1/0, want 2/1`. Its comment states the rule D-H2
  replaces: "A 403 is ErrAuthFailure on OpenAI, so the session is refreshed
  once and the listing sent once more before the error is returned
  (0026-MADR F6)". Its own subject, F38's "a listing that is not 200 is
  classified like any other answer", still holds: the error is an
  `*APIError` with status 403. Session renewal on a 401 has its own test,
  `TestListModels_ChatGPT401RefreshesOnce`, which passes.
* **Not measured:** Phase 1's T1 sent API keys only; what the ChatGPT
  backend answers a refused or expired session was never captured.
* **Options put to the owner:** the test follows D-H2; the ChatGPT backend
  is exempted, keeping its 403 a credential refusal; or measure first.
* **Decision.** "Measure first". Phase 3 pauses, uncommitted, until the
  measurement is recorded here and the owner decides.
* **The measurement, in two parts:**
  * **A refused session token** (no sign-in needed, unbilled):
    `TestLive_0028ChatGPTRefusedSession` in `live_0028_measure_test.go`
    sends the ChatGPT backend an `OAuthSession` whose access and refresh
    tokens are literals that are not tokens, through `ListModels` and
    `Generate`, and logs each reply's host, path, status and redacted body.
  * **An expired real session** needs the owner signed in, and waits for
    the owner.
* **Measured (2026-10-08).** `LLMPROVIDER_LIVE_0028=1 go test -tags
  live_gateways -run TestLive_0028ChatGPTRefusedSession ./llmprovider/`
  (unbilled; no sign-in):

  | Request | Status | Body (redacted) |
  | :--- | ---: | :--- |
  | `chatgpt.com/backend-api/codex/models` (the listing) | 401 | `{"detail":"Could not parse your authentication token. Please try signing in again."}` |
  | `auth.openai.com/oauth/token` (the refresh the 401 started) | 401 | `"Could not validate your token. Please try signing in again.", "type":"invalid_request_error"` |
  | `auth.openai.com/oauth/token` (`Generate`'s renewal) | 401 | the same |

  A refused ChatGPT session is a 401, as every refused key in T1 was; it is
  renewed under D-H2, as it was before, and as
  `TestListModels_ChatGPT401RefreshesOnce` pins. No refused credential has
  been seen to answer 403. **Not measured:** an expired real session, which
  needs the owner signed in.
* **Resolution.** "Test follows D-H2". `TestListModels_ChatGPTFailureIsAnAPIError`'s
  403 case wants 1 listing and 0 refreshes; its F38 assertion, an
  `*APIError` with status 403, is unchanged; its comment cites D-H2 and
  this measurement. Phase 3 continues.
* **Added to Phase 3:** `llmprovider/providers/openai/chatgpt_listing_errors_test.go`,
  and the measurement in `llmprovider/live_0028_measure_test.go`.

### Phase 3: classification (2026-10-08)

* **Before it.** Phase 2 and D4 committed by the owner as `276496b`.
* **Approval.** "i committed, proceed", 2026-10-08; D5 decided during the
  phase (above).

**Red first.** The package's tests named `credential403Codes`, so the
table was added first, empty per T1 and changing nothing, to let them run:

```text
--- FAIL: TestClassify_403IsNotPermittedUnlessCredentialCode
    api_error_0028_test.go:26: openai 403: kind llmprovider: authentication failed, retryable false, matches ErrAuthFailure true; want ErrNotPermitted, not retryable, true
    (and the same for claude, gemini, grok, together, huggingface, ollama)
--- FAIL: TestClassify_GrokRefusedKey
    api_error_0028_test.go:47: Grok's refused key: llmprovider: invalid request: grok HTTP 400 invalid-argument: Incorrect API key provided. …
--- FAIL: TestContextOverflow_CompletionAlone
    api_error_0028_test.go:84: 10 + 8192 on 8192: overflow = true; want false …
    api_error_0028_test.go:84: 0 + 9000 on 8192: overflow = true; want false …
--- FAIL: TestClassifyStreamFailure_ByType
    api_error_stream_0028_test.go:37: invalid request: kind llmprovider: provider unavailable, retryable true; want llmprovider: invalid request, false
    (and invalid request with overflow, completion alone, no type with overflow, authentication, permission, rate limit)
--- FAIL: TestReauth_RegionForbiddenNotRenewed
    reauth_0028_test.go:26: sends = 2, invalidations = 1; want 1 and 0
--- FAIL: TestList_Forbidden403NotRenewed
    reauth_0028_test.go:33: requests=2 invalidations=1 Err=llmprovider: authentication failed: together HTTP 403 unsupported_country_region_territory: …; want 1, 0 and ErrNotPermitted
--- FAIL: TestReauth_GrokRefusedKeyRenews
    reauth_0028_test.go:46: err llmprovider: invalid request: grok HTTP 400 invalid-argument: … after 1 request(s) and 0 invalidation(s); want success after 2 and 1
--- FAIL: TestTogether_TypedErrorInside200SentOnce/overflow and /invalid
    reply_0028_test.go:35: 3 request(s), llmprovider: together failed after 3 attempts: llmprovider: provider unavailable: together stream invalid_request_error: …
```

Guards that passed on the tree: `TestClassify_GrokInvalidArgumentStaysInvalid`;
`TestClassifyStreamFailure_ByType`'s overloaded, api, server and unknown
rows.

**The changes** (`llmprovider/api_error.go`, `llmprovider/context_overflow.go`):

* **D-H2:** `credential403Codes` (empty, T1); a `classifyAPIError` row after
  OpenCode's, Kilo's and Gemini's: a 403 without such a code is
  `ErrNotPermitted`, terminal. `statusSentinel` is unchanged, so the legacy
  sentinel still matches `ErrAuthFailure`.
* **H11:** a row before it: a Grok 400 with `invalid-argument` whose message
  holds "api key" (case folded) is `ErrAuthFailure`, terminal.
* **D-H3:** `completionFillsContext`, from `maximum context length is (\d+)`
  and `(\d+) in the messages, (\d+) in the completion`; the overflow row
  requires it false; `noteCompletionFillsContext` sets `Reason` to "max_tokens
  alone fills the context window: lower it" on such an invalid request, in
  `ClassifyHTTPError` and `ClassifyStreamFailure`.
* **D-H8:** `classifyStreamType`, run in `ClassifyStreamFailure`'s default
  branch when the service's codes left the stand-in `ErrProviderUnavailable`.
* `golangci-lint` asked for the error last in `classifyStreamType`'s results
  (`terminal, ok bool, kind error`) and for no local named `max`; both done.
* **Step 7:** no existing test pinned a typed failure inside a 200 as
  retryable. D5's test is the one existing test the phase changed.
* **Step 9:** `0012-MADR-conform-providers-to-reference-clients.md` gains
  "Amendment 2026-10-08: 403s, Grok's refused key, and failures inside a 200
  (0028)".

**Green:** `go test ./llmprovider/... ./wizard/...`: 24 packages ok.

**Proofs on scratch copies** (`p28_plants3b.py`, against the final code):

| Plant | Tests that failed |
| :--- | :--- |
| the 403 row off | `TestClassify_403IsNotPermittedUnlessCredentialCode`, `TestReauth_RegionForbiddenNotRenewed`, `TestList_Forbidden403NotRenewed`, `TestListModels_ChatGPTFailureIsAnAPIError` (`listing/refresh calls = 2/1, want 1/0`) |
| `completionFillsContext` always false | `TestContextOverflow_CompletionAlone`, `TestClassifyStreamFailure_ByType` |
| `classifyStreamType` deciding nothing | `TestClassifyStreamFailure_ByType`, `TestTogether_TypedErrorInside200SentOnce` |
| the Grok row off | `TestClassify_GrokRefusedKey`, `TestReauth_GrokRefusedKeyRenews` |
| the Grok row without its "api key" check | `TestClassify_GrokInvalidArgumentStaysInvalid` (`… authentication failed: grok HTTP 400 invalid-argument: Invalid request content: …`) |

**The gate** (`p26_gate.py`): the first run failed `make lint` (and
`pre-add-check`, which runs it) on the two findings above; after them:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `473 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet` for darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -cover`, `-shuffle=on`, `go mod tidy -diff`, `make lint` | 0 each |
| `parity-check`, `dep-check`, `generate-check` | 0 each |
| `coverage-check` | `28 packages, 0 problem(s)` |
| `api-check` | `against v1.3.2, 0 incompatible change(s) outside llmprovider/x/` |
| `records-check` | `61 records, 0 problem(s)` |
| `gate-selftest` | `Ran 14 tests`, OK |
| markdownlint, G-wire stable, links | 0 problems; 617 relative links in 71 files |
| identifier scan of the changed files | 0 hits |

### Deviation D6 (2026-10-08): D3's generic rows unmasked what is masked today

* **Found** in Phase 4, step 5c, running `internal/redact`'s tests after
  A10c as D3 recorded it (generic `sk-`/`xai-` rows of 32 or more characters
  holding a letter and a digit; `hf_` only at exactly 34):
  * `TestRedact_ProviderKeysAndTokenFields`: `String("Incorrect API key
    sk-FAKEabcdefghijklmnop0123 given")` kept the key; so did the
    `sk-or-v1-FAKE…0123`, `xai-FAKE…0123` and `hf_FAKE…0123` cases, each a
    24-character body;
  * `TestRedact_PlantedSecrets`: `case 0: "sk-QsIaQ85OZwvK7de7ni5uZ3tG"
    survived`.
* **Cause.** D3 states a principle, "nothing masked today goes unmasked
  unless a tighter rule masks it", and a 32-character floor, which unmasks
  today's 16–31-character digit-bearing values. The two conflict; the
  existing tests hold the principle.
* **Options put to the owner:** keep today's floor of 16 and require a
  letter and a digit, for `sk-`, `xai-` and `hf_`, with the exact `hf_` +
  34 row for real, letter-only tokens; or keep 32 and the exact `hf_`
  length, changing the fixtures and amending the principle.
* **Decision.** "Keep 16, require a digit". `reTokenFallback` is
  `\b(?:(?:sk|xai)-[A-Za-z0-9_-]{16,}|hf_[A-Za-z0-9]{16,})`, masked only
  when its body holds a letter and a digit; `\bhf_[A-Za-z0-9]{34}\b` stays in
  `reToken`. Every value masked today with a digit stays masked; the only
  values newly kept are digit-free `sk-`/`xai-`/`hf_` strings, the false
  positives D3 set out to fix. D3's table rows "`sk-` fallback", "`xai-`
  fallback" and "Hugging Face" read accordingly; the tests stay as written.

### Deviation D7 (2026-10-08): D6's digit rule unmasked random secrets

* **Found** in Phase 4, step 5c, after D6: `TestRedact_PlantedSecrets`
  (`internal/redact/redact_test.go:278`) failed, `case 46:
  "sk-AngeaXah_PHLGPjNKveTuAuT" survived`. Its generator plants `sk-`,
  `xai-` and `hf_` secrets with random 24-character bodies; about 1.5–1.7 %
  of them hold no digit, and D6 kept every digit-free body. So D6 unmasked
  random-looking secrets, not only the word-like false positives it meant.
* **Options put to the owner:** keep only word-like bodies, returning
  `hf_`'s generic row to today's; or keep D6 and plant realistic shapes.
* **Decision.** "Keep only word-like". D6's digit rule is replaced:
  * `reTokenFallback` is `\b(?:sk|xai)-[A-Za-z0-9_-]{16,}`, today's floor; a
    body is kept only when it is two or more lower-case-letter words joined
    by `-` (`sk-learn-tutorial-for-beginners`,
    `xai-grok-login-device-code-flow-tests`); any other body, with a digit
    or without, mixed case or with `_`, is masked;
  * `hf_`'s generic row is today's, `\bhf_[A-Za-z0-9]{16,}`, in `reToken`;
    the exact `hf_` + 34 row stays beside it, documented;
  * `(id)hf_requiredCharacteristicTypesForDisplayMetadata` stays masked, a
    known false positive, **not fixed**: a 45-letter mixed-case identifier
    has the shape a Hugging Face token could take, and keeping it would
    unmask such tokens;
  * `TestRedact_PlantedSecrets` stays as written.
* Everything masked before 0028 stays masked, except `sk-`/`xai-` values
  whose body is word-like.

### Deviation D8 (2026-10-08): A10b applies to `code` only

* **Found** in Phase 4, step 5b. D2 changes `reDiagnostic`, which bare
  `code`, `key` and `token` all keep, to admit letter-only words joined by
  `-` or `_`. That would keep a `key` or `token` value such as
  `abc-DEF-ghij`: a random value with no digit can take that shape, and those
  names hold secrets far more often than codes.
* **Built, narrower than approved, never wider.** `reDiagnostic` is
  unchanged. A new `reDiagnosticCode`, `^[A-Za-z]+(?:[-_][A-Za-z]+)+$`, keeps
  a value in `redactKVLong` only when the name is `code` (case folded) and
  the value is at most 40 bytes (`diagnosticCodeLimit`). `key` and `token`
  keep today's rule. `TestRedact_StillMasksSecretShapedCodes` gains
  `{"key":"abc-DEF-ghij"}`, which must stay masked.
* **Not put to the owner before building,** as D2's own narrowing was not:
  it masks more than the approval allows, never less. It is recorded here,
  in 0028-MADR's amendment "A10b keeps diagnostic words for `code` only",
  and in 0021-MADR's Z2 amendment, and is named in the handoff for the
  owner to confirm.
* **Confirmed** by the owner, 2026-10-08: "Confirmed".

### Deviation D9 (2026-10-08): A10b and A10c slowed clean-text redaction past step 7's bound

* **Found** in Phase 4, step 7. HEAD and the tree, as test binaries run
  alternately for nine rounds on a quiet host: `BenchmarkRedact_Clean4KiB`
  61.1 µs on HEAD, 71.3 µs on the tree, 1.167 times; the bound is 1.10.
  Before A10b and A10c it measured 64.5 µs against 64.8 µs.
* **Cause.** The profile puts 92 % of the time in `containsAny`, which runs
  `bytes.Contains` over the lower-cased text once per anchor of each pass.
  On clean text every anchor is absent, so each is a full scan. HEAD scans
  43; A10b and A10c bring it to 50 (`api_org_`, `aq.ab`, `ya29.`, `1//0`,
  `ollama`, and the fallback's `sk-` and `xai-`), 1.16 times. Five of the 50
  repeat an anchor another pass already scanned for (`token`, `code`,
  `key`, `sk-`, `xai-`). `bytes.Index` steps through each occurrence of an
  anchor's first byte, and most anchors begin with a common letter.
* **Measured on scratch copies**, the same nine alternated rounds:
  each distinct anchor looked up once per call, 66.4 µs (1.087); that and
  each anchor searched by its rarest byte in prose, the whole anchor checked
  around each hit, 26.3 µs (0.430). `internal/redact` and `llmprovider`
  passed under both; a 20 s fuzz of the rarest-byte search against
  `bytes.Contains`, 6.25 M executions, found no difference.
* **Options put to the owner:** both, as measured; or the lookup once per
  call alone, inside the bound by about 1 µs.
* **Decision.** "Memo + rare byte". `redact.go` numbers the distinct
  anchors of `passes` at init (`indexAnchors`, at most 64), and `Redact`
  looks each up at most once per call, on the stack (`anchorsSeen`), by its
  rarest byte (`anchorIn`, `rarestOffsets`). Redaction output is unchanged.
  `FuzzAnchorIn_MatchesContains` holds the search to `bytes.Contains`;
  `TestRedact_MatchesReference` and its companions hold the output. Not
  offered: amending step 7's bound, or dropping A10c's rows.
* **Files:** none beyond Phase 4's; the fuzz test goes in a new
  `internal/redact/redact_anchor_0028_test.go`.

### Deviation D10 (2026-10-08): redaction panics on a Kelvin sign, and misses ſ and K

* **Found** in Phase 4, by D9's plant "an absent anchor remembered as
  present", which panicked rather than failing an assertion. Probed on
  scratch copies of `HEAD` and the `v1.3.2` tag, with the same result on
  both, so the fault was released:
  * `String("key api_Key=abcd1234efgh")` panics, `index out of range [2]
    with length 0`, in `redactKVLong` (`internal/redact/redact.go:363`).
    `ReplaceAllFunc` hands it the match, and it matches `reKVLong` again on
    the match alone. `(?i)` folds the Kelvin sign (U+212A) into `k`, and Go's
    `\b` is ASCII-only: in context the `\b` before it holds after `_`; alone,
    at the start of the match, it does not. `FindSubmatch` returns nil.
    `ClassifyHTTPError` redacts service replies, so a reply holding such
    text crashes the caller. The ASCII anchor gate hides it unless the text
    also holds a plain `key` or `code`.
  * `redactCookie` re-matches the same way: `x_ſet-cookie: a=b` becomes
    `x_cookie: a=[REDACTED]`, the `ſet-` dropped.
    `redactAuth` and `redactKV` re-match too.
  * The gate, lower-casing ASCII only, leaves `x_ſecret=abcd1234efgh`
    unmasked; `Key=abcd1234efgh` at the start of the text, or after a
    space, is unmasked even with the gate open, because `\b` sees no
    boundary before a non-ASCII letter.
* **Options put to the owner:** fix in Phase 4, or a `v1.3.3` hotfix under a
  new pair first; for the misses, record them, or fold. A first answer chose
  folding the gate's lower-cased copy; a scratch prototype showed that
  leaves `Key=…` at the start of the text unmasked, and the question was
  put again with that corrected.
* **Decision.** "Fix in 0028 Phase 4" and "Fold the text":
  * the four functions read their groups from the match in context
    (`replaceSubmatches`, over `FindAllSubmatchIndex`), never by matching
    the match again, so they cannot miss;
  * when a message holds ſ (U+017F) or K (U+212A), `Redact` works on a copy
    with them replaced by `s` and `k`, the only non-ASCII letters `(?i)`
    folds into ASCII ones; the redacted text shows `s` and `k` there. The
    caller's buffer is never changed.
* **Tests:** `TestRedact_FoldedLetters` (no panic; each secret masked;
  the cookie keeps its name), `TestReplaceSubmatches_InContext` (the
  unfolded Kelvin case through `reKVLong`, no panic), and
  `FuzzRedact_NeverPanics`. The corpus cases "Kelvin sign" and "long s" join
  `changedBy0028`: they are now masked. `FuzzRedact_MatchesReference`'s
  oracle becomes the frozen `reKV` with today's in-context replacement,
  since the reference's re-matching is the fault.
* **Files:** `internal/redact/redact.go`, `redact_equiv_0028_test.go`, and a
  new `internal/redact/redact_fold_0028_test.go`. Ships in `v1.4.0`;
  `v1.3.x` keeps the fault.

### Deviation D11 (2026-10-08): Phase 4's code fails `make lint`

* **Found** by 0029-PLAN-go-1-27-2-for-standard-library-fixes.md's gate
  (its deviation D2). `52709e4`, Phase 4 committed before step 9's gate ran,
  fails `make lint` with three `goconst` findings, the same under
  `go1.27.1` and `go1.27.2`: `token` and `code`, three occurrences each, in
  `internal/redact/redact.go`'s anchor lists and `kvKeywords`; and
  `maximum context length`, three anchors in
  `llmprovider/context_overflow.go`'s `contextOverflowMessages`. CI runs
  govulncheck first, so it did not reach lint on `52709e4`.
* **Options put to the owner:** fix them here, in Phase 4's follow-up; or
  in 0029's change.
* **Decision.** "Fix under 0028 Phase 4": each literal is named once,
  `kwToken` and `kwCode` in `redact.go`, `anchorMaxContextLength` in
  `context_overflow.go`; behaviour unchanged. With those named, goconst
  reported a fourth, `key` (three occurrences, `redact.go`), which it had
  not listed before; it is `kwKey`. CI is green once 0029's
  commit and this follow-up are both committed.

### Phase 4: classification cost (2026-10-08)

* **Before it.** Phase 3 committed by the owner as `ad92bf7`.
* **Approval.** "i committed, proceed", 2026-10-08. D6 to D11 were decided
  during the phase (above). The owner committed the phase's work in
  progress as `52709e4`, after D9 and before step 9's gate. D10 and D11 are
  the follow-up to that commit.

**Step 1, the references.** `internal/redact/redact_ref_test.go` is
generated from `git show HEAD:internal/redact/redact.go`, its identifiers
renamed `ref…` and `Redact` renamed `redactReference`.
`llmprovider/context_overflow_ref_test.go` freezes `contextOverflow` the
same way. The first generation renamed the string `"pass"`, reKV's anchor,
along with the type `pass`, so the reference skipped reKV on `password=`.
It was caught because the first equivalence run differed, then regenerated
renaming the type only. Every quoted and backtick literal of the reference
was then checked as identical to `HEAD`'s.

**Step 2, before** (`-count=5`, medians, `ad92bf7`):

| Benchmark | Median |
| :--- | ---: |
| `Classify_16KiBTokens` | 12.67 ms |
| `Classify_2KiBOverflow` | 718 µs |
| `ClassifyHTTPError_64KiBHTML` | 908 µs |
| `Redact_16KiBTokens` | 5.34 ms |
| `Redact_Clean4KiB` | 64.8 µs |

**Step 3, red first.** `TestClassify_16KiBTokensUnder1ms` did not compile
(`undefined: redactInput`), as the step allows. The A10b and A10c tests,
before their code:

```text
--- FAIL: TestRedact_KeepsDiagnosticCodes
    String("{\"code\":\"invalid-argument\",…}") = "{\"code\":\"[REDACTED]\",…"
    (and INVALID_TOKEN, model-not-found)
--- FAIL: TestRedact_KeyShapes
    Gemini auth key, Google refresh token, Hugging Face org, Google access token: kept, in a sentence and in JSON
    Ollama: kept
--- FAIL: TestRedact_NotKeys
    "see sk-learn-tutorial-for-beginners for details" = "see [REDACTED] for details"
    (and xai-grok-login-device-code-flow-tests)
```

`TestRedactInput_NoSecretFragmentAtTheCut` differs from the step as
written. The key is 47 characters and starts at byte 2036, 12 bytes before
the bound rather than 20, and twenty redacted keys come first. That puts
the text at the cut inside the 512 bytes kept, where a fragment would be
seen. The test asserts that neither the fragment's start, `sk-proj-Q`, nor
`QWER` appears. Nothing after the cut reaches the message, so that is the
whole of what could leak.

**Step 4, the cut.** `redactInput` in `api_error.go` takes at most 2048
bytes (`apiErrorRedactLimit = 2 << 10`). It cuts back to the last of
`" \t\r\n,;\"'"` before the bound, then to a rune start, and replaces
`cutMessage` at both of its call sites.

**Measured, not planned.** After the cut, `Classify_16KiBTokens` was still
1.54 ms. The profile put 68 % of it in `completionFillsContext`, Phase 3's
D-H3, which ran two regexes over the whole 16 KiB message. It now reads at
most 2 KiB, and returns false unless the lower-cased text holds "in the
completion". That brought it to 0.319 ms.

**Step 5, `reKV` keyword first.** `redactKVPass` finds each of
`kvKeywords`. For each one it extends left over name bytes, then tries
`reKVAnchored` at each word boundary of that run, taking the first match.
Text with a non-ASCII byte takes `reKV`'s own search.

* **`TestRedact_MatchesReference`:** 18 named edge cases and 200 generated
  ones (PCG seed 28, 0). The cases that differ are exactly
  `changedBy0028`: `gen-192` (A10b), and "Kelvin sign" and "long s" (D10).
* **`TestRedact_MatchesReferenceOnExistingCases`:** this test covers the
  step's "every case of `redact_test.go`". It parses `redact_test.go`,
  `redact_providers_test.go` and `redact_0026_test.go`, and runs each of
  their 353 string literals, and sums of literals, through both. No
  difference.
* **`FuzzRedact_MatchesReference`:** the keyword-first pass against the
  frozen `reKV`, with today's replacement read in context (D10). 60 s,
  670,609 executions before D10; 60 s, 211,394 after; no failure.
* **The boundary plant.** Removing the word-boundary check from
  `redactKVPass` failed no test at first. Three corpus cases were added,
  each a keyword mid-word after `__` (`a__token=abcd1234efgh`,
  `x__client_secret: s3cr3tv4lue`, `9__password=hunter2hunter2`). The plant
  then failed `TestRedact_MatchesReference` and the fuzz seeds.

**Step 5b, A10b**, as D2 and D8 record. `reDiagnosticCode` keeps a `code`
value of letter-only words joined by `-` or `_`, at most 40 bytes. `key` and
`token` keep `reDiagnostic`.

**Step 5c, A10c**, as D3, D6 and D7 record:

* `reToken` gains the researched rows;
* `reTokenFallback` with `redactTokenFallback` (word-like bodies kept);
* `reOllamaKey` behind the anchor "ollama".

**Step 6, the overflow prefilter.** Each entry of `contextOverflowMessages`
carries its anchor. `contextOverflow` reads at most 2 KiB and runs a form
only when the lower-cased text holds its anchor. Text with a non-ASCII byte
runs every form, because `(?i)` folds `ſ` and `K`.
`TestContextOverflow_PrefilterAdmitsEveryPattern` and
`TestContextOverflow_MatchesReference` pass.

**Step 7, after.** D9 records the miss on `Redact_Clean4KiB` and its fix.
The final measurement, `ad92bf7` against the tree with D9, D10 and D11, nine
alternated rounds on a quiet host:

| Benchmark | `ad92bf7` | Tree | Ratio |
| :--- | ---: | ---: | ---: |
| `Classify_16KiBTokens` | 12,688 µs | 309 µs | 0.024 |
| `Classify_2KiBOverflow` | 724 µs | 173 µs | 0.239 |
| `ClassifyHTTPError_64KiBHTML` | 899 µs | 119 µs | 0.133 |
| `Redact_16KiBTokens` | 5,368 µs | 1,354 µs | 0.252 |
| `Redact_Clean4KiB` | 61.8 µs | 26.8 µs | 0.434 |

Both criteria are met: under 1 ms, and not more than 10 % slower.

**Step 8.** 0021-MADR gains "Amendment 2026-10-08: Z1's bound is 2 KiB,
and Z2 keeps a code's diagnostic words".

**Proofs on scratch copies.** Every plant failed:

| Plant | Tests that failed |
| :--- | :--- |
| the cut with no separator rule | `TestRedactInput_NoSecretFragmentAtTheCut` |
| the redaction bound at 16 KiB | `TestClassify_16KiBTokensUnder1ms` |
| the keyword pass without its boundary check | `TestRedact_MatchesReference`, `FuzzRedact_MatchesReference` (after the three cases above) |
| a wrong overflow anchor | `TestContextOverflow_PrefilterAdmitsEveryPattern`, `TestContextOverflow_MatchesReference`, the overflow message tests |
| no code diagnostic | `TestRedact_KeepsDiagnosticCodes` |
| the code diagnostic for any name, unbounded | `TestRedact_StillMasksSecretShapedCodes`, `TestRedact_MatchesReference` |
| no new shape rows | `TestRedact_KeyShapes` |
| no word-like exception | `TestRedact_NotKeys` |
| the Google key row dropped | `TestRedact_MatchesReferenceOnExistingCases` |
| a case file missing | `TestRedact_MatchesReferenceOnExistingCases` (`open no_such_test.go`) |
| D9: the rare byte alone, no whole-anchor check | `TestRedact_MatchesReference` |
| D9: the first hit only | `TestRedact_MatchesReference`, `TestRedact_PlantedSecrets`, `TestRedact_PrefixedKeys`, `TestRedact_ProviderKeysAndTokenFields` |
| D9: the wrong offset | `TestRarestOffsets` |
| D9: an absent anchor remembered as present | `TestRedact_MatchesReference` (a panic, which found D10) |
| D9: anchors not shared | `TestIndexAnchors_SharedOnce` |
| D9: no bound on the count | `TestIndexAnchors_Bound` |
| D10: no fold | `TestRedact_FoldedLetters`, `TestRedact_MatchesReference` |
| D10: groups read by matching the match again | `TestReplaceSubmatches_InContext` (the panic) |
| D10: the fold done in the caller's buffer | `TestRedact_FoldLeavesTheCallersBuffer`, `FuzzRedact_NeverPanics` |
| D10: a group that did not take part read anyway | `TestRedact_Coverage0021`, `TestRedact_KeepsOrdinaryWords`, both reference tests |

D11's check is `make lint` itself. It failed on `52709e4` with the three
findings, then a fourth, and passes after.

**Fuzzing** (60 s each, after D10): `FuzzRedact_NeverPanics`, 614,820
executions; `FuzzRedact_MatchesReference`, 211,394;
`FuzzAnchorIn_MatchesContains`, 18,330,364. No failures.

**Step 9, the gate** (`p26_gate.py`, a scratch copy of the tree,
`GOTOOLCHAIN=go1.27.2`, with 0029's change present):

| Check | Result |
| :--- | :--- |
| `make pre-add-check` | `483 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `CGO_ENABLED=0 go vet`, darwin, linux and windows, with and without `live_gateways` | 0 each |
| `go test -race -cover`, `-shuffle=on`, `go mod tidy -diff`, `make lint` | 0 each; `internal/redact` 100.0 %, `llmprovider` 98.2 % |
| `parity-check`, `dep-check`, `generate-check` | 0 each |
| `coverage-check` | `28 packages, 0 problem(s)` |
| `api-check` | `against v1.3.2, 0 incompatible change(s) outside llmprovider/x/` |
| `records-check` | `63 records, 0 problem(s)` |
| `gate-selftest` | `Ran 14 tests`, OK |
| markdownlint, G-wire stable | 0 |
| links | `0 problem(s), 626 relative link(s) in 73 file(s)` |
| deny scan | `0 hit(s) in 15 file(s)` |

**Not done in this phase:**

* The `hf_requiredCharacteristic…` identifier stays masked, a known false
  positive (D7).
* `v1.3.x` keeps D10's panic. The fix ships in `v1.4.0`.
* The phase's own commit was the owner's `52709e4`, made before the gate.
  The follow-up (D10, D11 and this record) is staged after 0029's commit,
  so that the two stay separate.
