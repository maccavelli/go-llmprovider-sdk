---
status: in-progress
date: 2026-10-20
associated-madr: "0053-MADR-stream-retry-and-cache-defects.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Fixing Six Defects in Streaming, Retry and the Catalog's Caches

Associated MADR: [0053-MADR-stream-retry-and-cache-defects.md](0053-MADR-stream-retry-and-cache-defects.md)

## Goal

The MADR's D1 to D7, as the owner answered its questions on 2026-10-20 (Q1:
keep 0028-MADR D-A3; Q2: a stream-index bound of 4096; Q3: `v1.20.0`):

* every streamed string accumulates in a builder, so a stream of N fragments
  costs O(N) (F1);
* every index a stream sends is checked against `0 ≤ i < 4096`, and one
  outside ends the stream with `ErrIncomplete`, never a panic (F2);
* a failure event inside a 200 stream carries the after-reply mark, so
  `WithRetry` and `WithStreamRetry` retry it at most once per call (F3);
* one listing fetch runs per key, detached from its callers, and every
  caller receives its result, a failure included (F4);
* each Ollama `/api/show` reply keeps its own age and its model's digest
  (F5);
* one probe run per key, detached, its result shared with its waiters (F6);
* `v1.20.0` is released with no exported name changed, and the MADR's probe
  shows each defect gone.

## Scope

### In scope

* **Phase 1, the streams:**
  * `llmprovider/internal/wire`: `sse.go` and a new index check;
  * the decoders in `chatcompletions`, `generatecontent` and `messages`;
  * `llmprovider/providers/gemini/interactions.go`;
  * `llmprovider/retry.go` (`WithRetry`'s doc);
  * the tests and benchmarks for all of these.
* **Phase 2, the catalog's caches:**
  * `llmprovider/catalog`: `cache.go`, `listing_cache.go`, `discovery.go`,
    `menu.go`, `listing_metadata.go`;
  * `llmprovider/internal/transport/probe.go`;
  * their tests.
* **Docs:** `docs/architecture.md`, `CHANGELOG.md`, and the README's Status.
* **Records:** the dated amendments to
  `0021-MADR-harden-and-tune-after-the-v1-1-review.md` (D1) and
  `0041-MADR-decide-per-model-behaviour-from-metadata.md` (A4).

### Out of scope

* Any exported name added, changed or removed (R48, R55): `tools/apidiff.py
  v1.19.0` reports nothing.
* The wire (R45): G-wire's goldens do not change.
* Caching an empty probe result past its run (Q1: 0028-MADR D-A3 stands).
* Negative caching of listings, stale-while-revalidate, and every other idea
  in `0054-REPORT-heuristics-and-performance-opportunities.md`.
* The probe's output limit (`probeMaxTokens`), which the MADR's D6 finds is
  not a defect.
* `llmprovider.ClassifyStreamFailure`, exported, which is unchanged: the
  mark is the reader's (MADR D3).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `tools/claims.py` on both records;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter, and read before anything
  is written from it.
* **Red first.** Every new test is seen to fail on today's code, or on a
  planted return of its defect, by `tools/plant.py` on a scratch clone,
  never in the tree. The phase's record holds each plant and its failure
  line.
* **The probe.** The MADR's probes (M1 to M7) are kept as scratch files
  outside the tree. Each phase runs its findings' probes again on a copy of
  the result, and records the lines.
* **The docs.** A function whose errors or behaviour change has its doc
  changed in the same phase, naming only what a test shows (R52). `make
  docs-check` stays at 0.
* **The API and the wire.**
  * `tools/apidiff.py v1.19.0` reports no change.
  * G-wire's goldens do not change (R45).
  * `make coverage-check` holds every floor (R47).
* **The existing tests are the drift check.** Every test at `e9ea383`
  passes unchanged, except the ones a step names as changing, with the
  reason.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, and the disclosure guard runs before the
  owner pushes. A records-only commit runs the Docs workflow
  (0052-MADR-skip-full-ci-for-docs-only-pushes.md); a phase with Go runs
  the full CI.
* **Other sessions.** If another session's uncommitted changes are in the
  tree, stage only this PLAN's paths. A shared file such as `docs/README.md`
  is staged with this PLAN's rows only, built from `HEAD`, and the staged
  snapshot is checked with `check_records.py` (as 0053-MADR's record was).
* **Stop conditions:**
  * a fact of the MADR, or a claim below, no longer holds;
  * a step would add or change an exported name;
  * a golden changes;
  * an existing test would need to change for a reason no step names;
  * a doc would state a behaviour no test shows.

  On any of these, stop, record a dated deviation here (with an amendment in
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.
2. **0021-MADR's amendment**, dated 2026-10-20, after its D1 (C29): "What
   counts" gains "a failure event the service sends inside a 200 stream,
   such as Responses' `response.failed` or Messages' `error` event", by
   0053-MADR D3. D1's text is not rewritten.
3. **0041-MADR's amendment**, dated 2026-10-20, after A4 (C30):
   * each `/api/show` reply keeps the time it was fetched, whether a listing
     or a lookup asked for it;
   * it is reused only while younger than `listingMetadataTTL`, and while
     the digest it was fetched for matches the one `/api/tags` lists, when
     both are known.

   This is 0053-MADR D5. A4's text is not rewritten.

### Phase 1: the streams (D1, D2, D3)

1. **The index check** (D2): in `llmprovider/internal/wire`, a new file
   `index.go`:
   * `const MaxStreamIndex = 4096`, exported within the internal package
     because `providers/gemini` uses it;
   * `func StreamIndex(provider, field string, i int) error`, returning nil
     for `0 ≤ i < MaxStreamIndex`, and otherwise an error matching
     `llmprovider.ErrIncomplete`: "%s: stream %s index %d out of range
     (0 to %d)".

   It is used before every slice growth that a streamed index drives:
   * **Chat Completions' tool calls** (C4): `tc.Index` is checked before the
     growth loop.
   * **Chat Completions' `reasoning_details`** (C3): the float is checked
     before conversion. A value that is negative, at least `MaxStreamIndex`,
     not finite, or not an integer ends the stream. A fragment with no
     `index` keeps index 0, as today.
   * **Messages' content blocks** (C5): `at(i)` returns the error. Every
     caller (`content_block_start`, `content_block_delta`,
     `content_block_stop`) passes it up.
   * **Gemini Interactions' steps** (C6): `step(index)` returns the error,
     and so do its callers.
2. **The builders** (D1):
   * **Chat Completions' `reasoning_details`** (C1): each entry becomes a
     struct holding its fields (`map[string]any`, first value kept, as
     today) and a `strings.Builder` for each of `text`, `summary`, `data`
     and `signature`. `mergeDetail` appends to the builders. At `[DONE]`,
     `done()` puts each builder's string under its key, only for a key a
     fragment carried, and marshals the map as today. Map keys marshal
     sorted, so the replayed bytes are unchanged.
   * **generateContent** (C2): `appendPart` keeps the trailing merged
     part's text in a builder held beside the parts. The builder is written
     into the part when a part that does not merge arrives, and when
     `raw.response()` is built. The merge rule in `appendPart`'s doc is
     unchanged.
   * **Gemini Interactions' `thought_signature`** (C7): a builder per step,
     beside `s.text` and `s.args`, read where `step.Signature` is used.
3. **The mark** (D3): in `wire.ReadEvents` (C8), a non-nil error from
   `dispatch` is returned as `transport.AfterReply(err)`, at both call sites
   (`sse.go:58` and `sse.go:79`). The reader's own errors are unchanged: the
   event over the limit (`ErrIncomplete`, unmarked) and the read failure and
   early end (already marked).
4. **Docs:**
   * `ReadEvents`' doc (C9): a dispatch error, such as a failure event the
     service sent inside the stream, is also marked, so `WithRetry` and
     `WithStreamRetry` ask again at most once.
   * `WithRetry`'s doc (C10): "such as a cut connection, a stream that
     ended early, or a failure the service reports inside the stream".
   * `docs/architecture.md`:
     * the `WithRetry` entry (C23) gains the in-stream failure;
     * the `internal/wire` row gains the stream-index bound.
   * `CHANGELOG.md` `[Unreleased]`:
     * **Changed:** an in-stream failure is retried at most once by
       `WithRetry` and `WithStreamRetry` (D3); a stream whose index is out of
       range ends with `ErrIncomplete` (D2).
     * **Fixed:** the quadratic accumulation (D1); the panic on a negative
       index (D2).
5. **Tests,** `_0053$`, under `-race`:
   * **F1, linear growth:**
     * `TestEvents_ReasoningDetailsLinear_0053` (Chat Completions) and
       `TestEvents_TextPartsLinear_0053` (generateContent): the bytes
       allocated (`runtime.MemStats.TotalAlloc`) for 16,000 four-byte
       fragments are under 12 times those for 2,000. On today's code the
       ratio is 55 for Chat Completions and 62 for generateContent (M1); a
       linear one is about 8.
     * `BenchmarkEvents_ReasoningDetails_0053` and
       `BenchmarkEvents_TextParts_0053`, for the record.
   * **F1, unchanged bytes:** `TestReasoningDetailsReplay_0053`: a stream of
     fragments that split `text` and `signature` replays one entry
     byte-for-byte equal to the entry sent whole. The existing
     `reasoning_details` replay tests pass unchanged.
   * **F2, every decoder:** `TestStreamIndex_0053`, a table over the four
     decoders (C3 to C6). Index `-1`, index `4096` and index ~~`1e9`~~ 100,000 *(Phase 1 deviation: safe to plant)* each give
     an error matching `ErrIncomplete` that names the field, with no panic
     and less than 1 MiB allocated. Index `4095` is accepted.
   * **F2, through a provider:** `TestStream_IndexOutOfRange_0053` in
     `providers/huggingface`: the public `Stream` yields that error and
     returns, with no panic (M3).
   * **F3, `Generate`:** `TestGenerate_InStreamFailureOnce_0053` in
     `providers/openai`, with M4's ChatGPT-session server. `WithRetry` at
     `MaxAttempts` 3 and 5 sends 2 requests each time. The error matches
     `ErrProviderUnavailable`, is marked, and keeps its text.
   * **F3, `Stream`:** `TestStreamRetry_InStreamFailureBeforeFirstEvent_0053`
     in `providers/openai`. A stream whose first event is `response.failed`
     gets 2 requests under `WithStreamRetry` at `MaxAttempts` 3. One that
     fails after a delta still gets 1 request (0051-MADR D2).
   * **F3, the reader:** `TestReadEvents_DispatchErrorMarked_0053` in
     `internal/wire`: a dispatch error comes back marked, `errors.Is` and
     `errors.As` still find its kind and `*APIError`, and an event over the
     limit stays unmarked.
6. **The probe,** M1 to M4, run again on a copy of the result:
   * M1: linear rows;
   * M2 and M3: `ErrIncomplete` and no panic;
   * M4: 2 requests.
7. **Red first,** each caught:
   * C1's `prev + next` restored: the `reasoning_details` linear test fails.
   * C2's `+=` restored: the text-parts linear test fails.
   * The check removed from each of the four decoders, one plant each: that
     decoder's row in `TestStreamIndex_0053` fails (a recovered panic, or
     allocation over 1 MiB).
   * The mark removed from `ReadEvents`: the three F3 tests fail.
   * The builder's string not written at `[DONE]`: the replay test fails.

### Phase 2: the catalog's caches (D4, D5, D6)

1. **The listing flight** (D4):
   * **`store.listingClaims`** (C13) becomes `map[listingKey]*listingCall`.
     The call holds `done chan struct{}` and the flight's result:
     `f fetchedListing`, `models map[string]ModelInfo`, `at time.Time` and
     `err error`.
   * **`claimListing(key)`** returns `(call, leader bool)`.
   * **`cachedOrFetched`** (C11, C12):
     1. A fresh listing is returned as today.
     2. Otherwise the caller joins the key's call, or starts one as the
        leader.
     3. The leader runs `fetchListing` on `cfg.store.listingFetching.Go`, on
        `context.WithTimeout(context.WithoutCancel(ctx), modelListingTimeout)`,
        as the metadata fetch does (C14).
     4. It stores a success as `fetchClaimed` does today, records the
        result on the call, removes the call from the map and closes `done`.
   * **Every caller,** the leader included, selects on `done` and its own
     `ctx.Done()`. It receives the call's result, with its own copy of
     `models` (`maps.Clone`), since callers may keep it. A caller whose
     context ends gets `ctx.Err()`, as today; the flight goes on.
   * **Unchanged:**
     * `transientListingFailure` and the last good listing;
     * `storeFetched`;
     * the renewal of a refused credential (`renewRefused`), which lists
       again under the new token's key;
     * `startListingFetch` (`CachedModel`'s background fetch), whose
       in-flight set is separate.
2. **Ollama's replies** (D5):
   * **The tag:** `ollamaTag` (C17) gains ``Digest string `json:"digest"` ``.
   * **Each reply's own entry:** `/api/show` replies are cached per listing
     key and model name, as `{info ModelInfo; fetched time.Time; digest
     string}`. They are written by:
     * `ollamaShows` (C18), for a reply it fetched;
     * `fetchListingModel`'s Ollama path (`listing_cache.go:203`, C31), for
       a lookup's reply.

     Re-emitting a cached reply (C15) does not rewrite its time.
   * **Reuse:** a reply is reused while `listingNow()` minus its `fetched`
     time is under `listingMetadataTTL`. When both its digest and the listed
     digest are non-empty, they must also be equal.
   * **The listing's own facts** (`storeListing`, C16), which `CachedFacts`
     and `LookupModel` read, are stored as today. Their freshness no longer
     decides whether a reply is asked for again.
   * **Unchanged:**
     * a failed reply is not cached, and is asked again (0042-MADR D1);
     * four requests at a time (`ollamaShowConcurrency`);
     * `/api/tags` order;
     * the caveats.
3. **The probe flight** (D6):
   * **`ProbeStore`** (C21) gains `flights map[ProbeKey]*probeFlight`, where
     a flight holds `done chan struct{}` and the healthy list.
   * **`ProbeGenerateHealthCached`** (C19, C20):
     1. A cached healthy list is returned as today.
     2. Otherwise the caller joins the key's flight, or starts one.
     3. The flight runs `probeHead` in a goroutine, on
        `context.WithoutCancel(ctx)`. Each probe is still bounded by
        `probeTimeout`.
     4. It caches a non-empty result as today (Q1: an empty one is not
        cached), records it on the flight, removes the flight and closes
        `done`.
   * **Every caller** selects on `done` and its own context, and appends its
     own tail with `withTail`, as today. A caller whose context ends gets
     what a run with no answer gives, `withTail(nil, …)`, so its provider
     keeps its listing, as today when no probe answers.
4. **Docs:**
   * **`List`'s doc** (C26) and **the architecture's listing paragraph**
     (C22): "callers at once share one fetch, and its result, a failure
     included; a caller that is cancelled ends only its own wait".
   * **The Ollama entry** (C24): each reply is cached for ten minutes from
     when it was fetched, and asked again at once when its model's digest
     changes.
   * **The probe entry** (C25): concurrent listings share one run of probes.
   * **`CHANGELOG.md` `[Unreleased]`, Fixed:**
     * concurrent listings share one fetch and its failure, and a cancelled
       caller no longer restarts it (D4);
     * an Ollama reply expires ten minutes after it was fetched, and when its
       model's digest changes (D5);
     * concurrent listings share one run of probes (D6).
5. **Tests,** `_0053$`, under `-race`:
   * **F4, the listing flight,** in `catalog`:
     * **`TestListingFlight_SharesFailure_0053`:** 5 concurrent `List`
       calls, the server answering 503, and then 401, after 100 ms: 1 request
       each time, every caller given the failure. The 503 case takes no last
       good listing (none cached).
     * **`TestListingFlight_CancelledClaimant_0053`:** a claimant cancelled
       at 30 ms, and a waiter that started after it: 1 request, and the
       waiter's catalog is live.
     * **The existing single-flight test,** `TestList_Cached_0041`'s 8 concurrent
       `List` calls with 1 request (C27), passes unchanged.
   * **F5, Ollama's replies,** in `catalog`:
     * **`TestOllamaShow_AgePerReply_0053`:** M6's clock, listings at 0, 9,
       18, 27 and 36 minutes, gives 3 `/api/show` requests.
     * **`TestOllamaShow_DigestChange_0053`:** a listing at 0 minutes, then
       one at 1 minute whose digest differs, asks again at once: 2 requests.
       An unchanged digest asks nothing, and a missing digest leaves the age
       alone to decide.
   * **F6, the probe flight,** in `internal/transport` and
     `providers/openai`:
     * **`TestProbeFlight_SharedRun_0053`** (`transport`): 5 concurrent cold
       calls run `generate` 6 times in all.
     * **`TestListModels_ConcurrentProbes_0053`** (`openai`): M7's cold row
       sends 6 probe generations.
     * **`TestProbeFlight_CancelledClaimant_0053`:** a waiter receives the
       run's healthy list after the claimant's context is cancelled.
     * **The existing tests,** `TestProbeCache_EmptyResultNotCached` (C28)
       and the other probe tests, pass unchanged.
6. **The probe,** M5 to M7, run again on a copy of the result:
   * M5: 1 request in each row;
   * M6: 3 `/api/show` requests;
   * M7: 6 cold probes, and 6 per listing while every probe fails, as D-A3
     keeps it.
7. **Red first,** each caught:
   * **A waiter not given the flight's failure** (refetching instead): the
     shared-failure test fails.
   * **The flight run on the claimant's context:** the cancelled-claimant
     test fails.
   * **Re-emitted replies stamped now:** the age test fails.
   * **The digest ignored:** the digest test fails.
   * **No probe flight** (each caller probing): the shared-run tests fail.
   * **The probe flight on the claimant's context:** its cancelled-claimant
     test fails.

### Phase 3: release `v1.20.0` (D7)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.20.0] - <date>`, the date
   that of the commit the tag will name. It holds Phases 1 and 2's entries
   and 0052's Changed entry. An empty `[Unreleased]` goes above it, and the
   comparison links are updated. The README's Status names `v1.20.0`.
2. **Before the push:**
   * the gate;
   * `tools/apidiff.py v1.19.0`, with no change;
   * the disclosure guard over the outgoing commits, run by the agent
     before the owner pushes.
3. **The tag.** The owner tags `v1.20.0` and pushes it. CI's `release` job
   makes the Release, and its body equals `check_changelog.py --section
   v1.20.0 --repo maccavelli/go-llmprovider-sdk`.
4. **After the tag:** `tools/consumer.py v1.20.0` and `tools/offline.py
   v1.20.0`.
5. **Close:**
   * the MADR's Confirmation, item by item, on a scratch copy of `v1.20.0`,
     M1 to M7 included;
   * this PLAN marked `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | Streamed text accumulates linearly, and replays unchanged | 1 | the linear tests and benchmarks; the replay test; M1 |
| V2 | No streamed index panics or over-allocates; one out of range is `ErrIncomplete` | 1 | `TestStreamIndex_0053`; the public `Stream` test; M2, M3 |
| V3 | An in-stream failure is retried at most once by both wrappers | 1 | the three F3 tests; M4 |
| V4 | One listing fetch per key, its result shared, a cancelled claimant harmless | 2 | the two listing-flight tests; the existing 8-caller test; M5 |
| V5 | An Ollama reply ages from its own fetch, and is asked again on a new digest | 2 | the two Ollama tests; M6 |
| V6 | One probe run per key, shared; an empty result still not cached | 2 | the probe-flight tests; `TestProbeCache_EmptyResultNotCached`; M7 |
| V7 | No exported name, golden or unnamed existing test changes | all | `tools/apidiff.py v1.19.0`; G-wire; the full suite |
| V8 | `v1.20.0`'s Release body is its section | 3 | `gh release view`; `--section` |
| V9 | Every new test failed first | 1, 2 | the execution record |

## Rollout and Rollback

* **Rollout.** Four commits on `main`, staged by the agent and committed by
  the owner: the records (Docs workflow), then a commit per phase (full CI).
  Then the owner's tag.
* **Rollback.** Each phase reverts on its own, newest first.
  * Reverting Phase 2 restores the per-caller listing fetch, the reply ages
    stamped by each listing, and the per-caller probes.
  * Reverting Phase 1 restores the quadratic accumulation, the panics and
    the unmarked in-stream failures.
  * After `v1.20.0` is tagged, a revert ships as a new release.

## Code claims

<!-- claims: self=e9ea383 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | Chat Completions joins a `reasoning_details` string to the one before | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:727` | `next = prev + next` |
| C2 | generateContent joins text parts with `+=` | `llmprovider/internal/wire/generatecontent/generatecontent.go:339` | `parts[n-1].Text += part.Text` |
| C3 | The `reasoning_details` index, from a float | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:706` | `index = int(f)` |
| C4 | The tool-call index, unchecked | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:656` | `for len(s.calls) <= tc.Index {` |
| C5 | Messages' content-block index | `llmprovider/internal/wire/messages/messages.go:410` | `for len(m.Content) <= i {` |
| C6 | Gemini Interactions' step index | `llmprovider/providers/gemini/interactions.go:308` | `for len(s.raw.Steps) <= index {` |
| C7 | Gemini Interactions joins a signature with `+=` | `llmprovider/providers/gemini/interactions.go:326` | `step.Signature += ev.Delta.Signature` |
| C8 | `ReadEvents` returns a dispatch error as it is | `llmprovider/internal/wire/sse.go:58` | `if stop, err := next(); stop` |
| C9 | `ReadEvents`' doc of the mark | `llmprovider/internal/wire/sse.go:38` | `// WithRetry asks again at most once (0021-MADR D1).` |
| C10 | `WithRetry`'s doc of the once-only rule | `llmprovider/retry.go:70` | `cut connection or a stream that ended early, is retried at most once per` |
| C11 | A waiter is handed a bare channel | `llmprovider/catalog/discovery.go:364` | `done, wait := cfg.store.claimListing(key)` |
| C12 | The claimant fetches on its own context | `llmprovider/catalog/discovery.go:383` | `f, failure := fetchListing(ctx, id, token, cfg)` |
| C13 | The claims map holds channels | `llmprovider/catalog/cache.go:55` | `listingClaims map[listingKey]chan struct{}` |
| C14 | The metadata fetch is detached | `llmprovider/catalog/model_metadata.go:490` | `fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), metadataFetchTimeout)` |
| C15 | Ollama emits cached replies again | `llmprovider/catalog/discovery.go:843` | `cfg.emit(show)` |
| C16 | Then stores them all, not merged | `llmprovider/catalog/discovery.go:347` | `cfg.store.storeListing(listingKeyFor(llmprovider.ProviderOllama, cfg, token), models, false)` |
| C17 | `/api/tags` is decoded for name and capabilities | `llmprovider/catalog/discovery.go:856` | `Capabilities []string` |
| C18 | `ollamaShows` reuses a cached reply | `llmprovider/catalog/menu.go:229` | `if info, ok := cached[name]; ok {` |
| C19 | The probe runs with no shared flight | `llmprovider/internal/transport/probe.go:170` | `healthy := probeHead(ctx, preferred, limit, generate)` |
| C20 | Only a result with a model is cached | `llmprovider/internal/transport/probe.go:171` | `if len(healthy) > 0 {` |
| C21 | `ProbeStore` holds only results | `llmprovider/internal/transport/probe.go:120` | `m  map[ProbeKey]probeCacheEntry` |
| C22 | The architecture's listing cache | `docs/architecture.md:198` | `with no request, and callers at once share one fetch. A transient failure` |
| C23 | The architecture's `WithRetry` entry | `docs/architecture.md:447` | `after the request was written, is retried at most once per call` |
| C24 | The architecture's Ollama entry | `docs/architecture.md:725` | `` `/api/show` for each installed model not cached, four at a time, and `` |
| C25 | The architecture's probe entry | `docs/architecture.md:751` | `unprobed. The models that answered are cached in` |
| C26 | `List`'s doc of the shared fetch | `llmprovider/catalog/discovery.go:191` | `// and callers at once share one fetch (0041-MADR D6, A6). A transient` |
| C27 | The existing single-flight test | `llmprovider/catalog/steady_0041_test.go:135` | `for 8 concurrent Lists; want 1` |
| C28 | The existing empty-result test | `llmprovider/internal/transport/probe_0028_test.go:63` | `func TestProbeCache_EmptyResultNotCached(t *testing.T) {` |
| C29 | 0021-MADR D1's list of what counts | `docs/decisions/0021-MADR-harden-and-tune-after-the-v1-1-review.md:176` | `an event stream that ends before its completion event (0012 §4.1).` |
| C30 | 0041-MADR A4's life for a reply | `docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md:837` | `` and caches each reply as `LookupModel` does `` |
| C31 | A lookup's reply merged into the entry | `llmprovider/catalog/listing_cache.go:203` | `cfg.store.storeListing(listingKeyFor(id, cfg, token), map[string]ModelInfo{model: info}, true)` |

## Execution record

### Phase 0: records (2026-10-20)

* **No deviation.** The owner committed the MADR as `08a9336` and its
  answers as `e9ea383`. This PLAN was still staged, not committed, when
  Phase 0 began, so it goes in with Phase 0's records. Each amendment follows its own
  record's convention, so it sits apart from the passage it amends:
  * 0021-MADR's is appended as its last `## Amendment` section, as its
    others are;
  * 0041-MADR's is A9, in its own `## Amendment` section before the Pros and
    Cons, as A1 to A8 are.

  Neither rewrites the passage it amends (C29, C30).
* **Step 1,** `tools/records.py status`: the MADR `accepted`, this PLAN
  `in-progress`, each with its index row; "records-check: 0 problem(s)".
* **Step 2,** 0021-MADR's "Amendment 2026-10-20: a failure event inside a
  200 stream counts": D1's "What counts" gains a failure event the service
  sends inside a 200 stream, retried at most once per `Generate`, and per
  `Stream` before the first event, with M4's counts and D1's own reason.
* **Step 3,** 0041-MADR's "A9. An `/api/show` reply keeps its own age, and
  its model's digest": each reply ages from its own fetch, by a listing or a
  lookup, and is asked again when the listed digest differs. The digest is
  compared opaquely, and the age alone decides without one. M6's count and
  0042-MADR D1's unchanged rule are cited.
* **Lint:** 0041-MADR's first list uses `-`, so markdownlint's MD004 expects
  it; A9's three bullets use `-` and its count stays at 298, as at HEAD.
  The first draft's `*` raised it to 301, and `tools/recordlint.py` caught
  it.
* **The checks:** `tools/claims.py`, 22 of 22 and 31 of 31; `make
  records-check`, 0 problems; `tools/recordlint.py --changed`, no rule rose;
  `tools/linkcheck.py`, 0 problems; `tools/gate.py`, 27 of 27 passed.

### Phase 1: the streams (2026-10-20)

* Phase 0 was committed by the owner as `882272f`.
* **Deviations (2026-10-20), no decision changed,** offered to the owner
  before the commit:
  * **Step 5's large index is 100,000, not 1e9.** The first run of step 7's
    plants had to be stopped: with a check planted away, index 1e9 made the
    decoder allocate a slot for every index below it, which would have
    exhausted the machine's memory. Index 100,000 still proves the bound:
    unchecked it allocates about 50 MB, fifty times the test's 1 MiB limit,
    and it is safe to plant. Step 5's text is annotated, not rewritten.
  * **`TestStreamIndex_0053` is one test per package,** in
    `chatcompletions`, `messages` and `providers/gemini`, not one table.
    The four decoders live in three packages, and Gemini's
    `interactionEvents` is unexported.
  * **Step 1's helper has a sibling,** `StreamIndexNumber`, for
    `reasoning_details`' index, which arrives as a JSON number. It refuses a
    value that is not finite or not whole before converting it. Both are in
    an internal package, so no exported name changes.
* **Step 1, the bound** (D2), `internal/wire/index.go`: `MaxStreamIndex =
  4096`, `StreamIndex` and `StreamIndexNumber`, each an error matching
  `ErrIncomplete`, "stream <field> index <i> out of range (0 to 4095)". They
  are used at all four places:
  * Chat Completions' tool calls, before the growth loop;
  * Chat Completions' `reasoning_details`, on the float;
  * Messages' `at()`, which now returns an error to its three callers;
  * Gemini Interactions' `step()`, which now returns an error to its two
    callers.
* **Step 2, the builders** (D1):
  * **Chat Completions:** `streamDetail` keeps an entry's first-value fields
    and a `strings.Builder` per split string. `value()` joins them at
    `[DONE]`, and the map marshals as before.
  * **generateContent:** `streamParts` holds the text joining the last part
    in a builder, flushed when a part that does not merge arrives, and
    before the reply is built.
  * **Gemini Interactions:** a signature builder per step, added after the
    start's signature at `completed`.
* **Step 3, the mark** (D3): `ReadEvents` returns
  `transport.AfterReply(err)` at both dispatch call sites. `AfterReply(nil)`
  is nil, so a clean stop is unchanged.
* **The drift check:** after steps 1 to 3, before any new test,
  `go test -race -count=1 ./...` passed in all 36 packages, unchanged.
* **Step 4, the docs:**
  * `ReadEvents`' doc (C9) and `WithRetry`'s (C10);
  * `docs/architecture.md`'s `internal/wire` row (the mark and the bound)
    and its `WithRetry` entry (C23);
  * `CHANGELOG.md` `[Unreleased]`: Changed (D3, D2) and Fixed (D1, D2),
    beside 0052's Changed entry.

  `make docs-check`, "sdkaudit: 0 violation(s) of R50,R51,R52"; `make
  changelog-check`, "27 release section(s), 0 problem(s)"; `make
  tables-check`, "38 packages, 0 problem(s)"; `make parity-check`, "409
  identifiers, 409 rows, 409 with an SDK equivalent, 0 problem(s)";
  markdownlint on both files, no finding.
* **Step 5, the tests,** under `-race`, each passing:
  * **F1:**
    * `TestEvents_ReasoningDetailsLinear_0053` and
      `TestEvents_TextPartsLinear_0053`: the least of three runs of
      `TotalAlloc`, under 12 times for eight times the fragments;
    * `BenchmarkEvents_ReasoningDetails_0053` and
      `BenchmarkEvents_TextParts_0053`;
    * `TestReasoningDetailsReplay_0053`: `text`, `summary` and `signature`
      split, `format` kept at its first value; EventDone's Response equals
      `Decode`'s on the entry sent whole;
    * `TestInteractionSignature_0053`: the start's signature, then two
      deltas, give "abc" on the call that follows. A first draft looked for
      it on a reasoning item; a thought step's signature goes to the next
      call, and the test was corrected.
  * **F2:**
    * `TestStreamIndex_0053`, in the three packages: -1, 4096 and 100,000,
      each `ErrIncomplete` naming the field, with no panic and, in Chat
      Completions, under 1 MiB allocated. 4095 passes the bound (in
      Messages and Gemini, where the test's answer is empty, the check is
      that no error says "out of range");
    * `TestStreamIndex_Helpers_0053` (`internal/wire`);
    * `TestStream_IndexOutOfRange_0053` (Hugging Face's public `Stream`).
  * **F3:**
    * `TestReadEvents_DispatchErrorMarked_0053`: marked at an event and at
      the stream's end, with the text and kind unchanged; over the limit,
      unmarked; a clean stop, nil;
    * `TestGenerate_InStreamFailureOnce_0053`: 2 requests at `MaxAttempts` 3
      and 5;
    * `TestStreamRetry_InStreamFailureBeforeFirstEvent_0053`: 2 requests
      before the first event, 1 after a delta.
* **Step 6, the MADR's probe,** unchanged, on a copy of the working tree:
  * **M1:** Chat Completions' `reasoning_details` allocates 1,689, 3,226,
    6,406 and 12,688 KiB for 2,000 to 16,000 fragments, against 10,048 to
    552,955 KiB before. generateContent allocates 492 to 3,167 KiB, against
    8,814 to 543,276 KiB.
  * **M2:** each decoder, at index -1, returns "llmprovider: incomplete
    response: probe: stream <field> index -1 out of range (0 to 4095)",
    with no panic. Index 1,000,000 allocates 0 MiB, against 505 MiB before.
  * **M3:** Hugging Face's public `Stream`: no panic.
  * **M4:** the ChatGPT `Generate`, 2 requests at `MaxAttempts` 3 and 5,
    `afterReply=true`, against 3 and 5. Claude's `Stream` after a delta: 1
    request, passed through.
* **Step 7, red first,** ten plants, each caught by a failing test, none by
  a failed build:

  | Plant | Failure line |
  | :--- | :--- |
  | `reasoning_details` joined by copying, as before | "16,000 fragments allocated 1118685600 bytes, 2,000 allocated 18763584: 59.6 times; want under 12" |
  | generateContent parts joined with `+=`, as before | "61.8 times; want under 12" |
  | Chat Completions' tool-call index unchecked | "tool_calls -1: err <nil>, panic runtime error: index out of range [-1]" |
  | Chat Completions' `reasoning_details` index unchecked | "reasoning_details -1: … panic runtime error: index out of range [-1]" |
  | Messages' content-block index unchecked | "content_block_start index -1: … panic runtime error: index out of range [-1]" |
  | Gemini Interactions' step index unchecked | "step.start index -1: … panic" |
  | `ReadEvents`' mark removed at an event | the reader's test, `TestGenerate_InStreamFailureOnce_0053` ("MaxAttempts 3: 3 requests … want 2") and the stream retry test ("before the first event: 3 requests … want 2") |
  | `ReadEvents`' mark removed at the stream's end | "at the stream end: … (after the reply: false)" |
  | the split strings not written at `[DONE]` | the replay test, its Encrypted entry lacking text, summary and signature |
  | Gemini's signature deltas not joined | "the call's signature \"a\"; want \"abc\"" |

  The plant command carries `-timeout 180s`, after the stopped first run.
  The plants were run again after the last test edit (the lint fix
  below), all caught.
* **The checks:**
  * `tools/apidiff.py v1.19.0`: no output, exit 0;
  * `make coverage-check`: "37 packages, 0 problem(s)" (`internal/wire`
    95.8 %, `chatcompletions` 94.6 %, `generatecontent` 97.4 %, `messages`
    91.3 %, `gemini` 96.7 %);
  * `make pre-add-check` on the 14 Go files: "14 file(s) clean". A first
    run found staticcheck's ST1008 in two test helpers that returned
    their error first; they now return it last, and the plants were run
    again after the fix, all caught;
  * `tools/gate.py`: 27 of 27 passed.

### Phase 2: the catalog's caches (2026-10-20)

* **No deviation.** Phase 1 was committed by the owner as `50668bc`, with
  its three deviations accepted.
* **Step 1, the listing flight** (D4):
  * **The call:** `store.listingClaims` holds `*listingCall` (`done`, and
    the result: `f`, `models`, `at`, `err`). `claimListing` returns the call
    and whether the caller leads; `finishListing` removes it and closes
    `done`.
  * **The fetch:** `cachedOrFetched`'s leader runs `fetchClaimed` on
    `listingFetching.Go`, under `context.WithTimeout(context.WithoutCancel(ctx),
    modelListingTimeout)`.
  * **Every caller** waits on `done` or its own context, and receives the
    result with its own copy of `models` (`maps.Clone`).
  * **Unchanged:** `fetchClaimed` lost only its `done` parameter. The last
    good listing, `storeFetched`, `renewRefused` and `startListingFetch` are
    as they were.
* **Step 2, Ollama's replies** (D5):
  * **The tag:** `ollamaTag` decodes `digest`, and its doc says why.
  * **The store:** `store.ollamaShows` holds a `showReply` (`info`,
    `fetched`, `digest`) per listing key and model.
    * `freshShows` returns the replies younger than `listingMetadataTTL`
      whose digest matches the listed one, when both are known.
    * `storeShow` records a reply fetched now.
  * **The listing:** `ollamaCatalog` passes `freshShows` to `ollamaShows`,
    and stores only the replies it fetched, with their models' digests.
  * **A lookup:** `fetchListingModel`'s Ollama path stores its reply with no
    digest.
  * **The listing's facts** (`storeListing`) are stored as before. A lookup
    reads them, and they are stamped when the listing is stored, so a lookup
    may see a reply up to about two TTLs old, as the step keeps it. That is
    no longer forever: a listing reuses a reply only while it is young.
* **Step 3, the probe flight** (D6):
  * **The flight:** `ProbeStore` gains `flights` (`*probeFlight`: `done`,
    `healthy`), initialised by `NewProbeStore`. It is also made on first use,
    for a store whose results map a test resets (`resetProbeCache`).
  * **The run:** one goroutine per key runs `probeHead` on
    `context.WithoutCancel(ctx)`, caches a non-empty result, records it on
    the flight, and closes `done`.
  * **Every caller** waits on `done` or its own context. A caller whose
    context ends first gets `withTail(nil, …)`, nil.
  * **The doc** of `ProbeGenerateHealthCached` says so.
* **The drift check:** after steps 1 to 3, before any new test,
  `go test -race -count=1 ./...` passed in all 36 packages, unchanged. That
  includes `TestList_Cached_0041`'s 8 concurrent Lists (C27),
  `TestOllamaUsable_TagsNotCached_0042` (0042-MADR D1's re-ask of a failed
  reply) and `TestProbeCache_EmptyResultNotCached` (C28).
* **Step 4, the docs:**
  * `List`'s doc (C26) and `docs/architecture.md`: its listing paragraph
    (C22), its Ollama entry (C24) and its probe entry (C25);
  * `CHANGELOG.md` `[Unreleased]` Fixed: D4, D5 and D6, each with its
    measured count.

  `make docs-check`, "sdkaudit: 0 violation(s) of R50,R51,R52"; `make
  changelog-check`, "27 release section(s), 0 problem(s)"; `make
  tables-check`, "38 packages, 0 problem(s)"; `make parity-check`, "409
  identifiers, 409 rows, 409 with an SDK equivalent, 0 problem(s)";
  markdownlint on both files, no finding.
* **Step 5, the tests,** under `-race`, each passing:
  * **F4,** `catalog`:
    * `TestListingFlight_SharesFailure_0053`: HTTP 503, then 401, after
      100 ms, five concurrent Lists: 1 request, and all five given the
      static catalog with the failure in `Err`;
    * `TestListingFlight_CancelledClaimant_0053`: a claimant cancelled at
      30 ms, a waiter 5 ms after it: 1 request, the waiter's catalog live.
  * **F5,** `catalog`, on the steady clock:
    * `TestOllamaShow_AgePerReply_0053`: five listings nine minutes apart,
      3 `/api/show` requests;
    * `TestOllamaShow_DigestChange_0053`: an unchanged digest, 1; a changed
      one, 2; no digest at 1 minute, 2; at 11 minutes, 3.
  * **F6:**
    * `TestProbeFlight_SharedRun_0053` (`transport`): five concurrent cold
      calls, 6 probes, each given the head and its tail;
    * `TestProbeFlight_CancelledClaimant_0053`: the cancelled claimant nil,
      the waiter the full list after 6 probes, a later call cached;
    * `TestProbeFlight_EmptyRunShared_0053`: three concurrent calls whose
      every probe answers wrongly share one run (6 probes), and the next
      call probes again (6 more), as Q1 keeps D-A3. A first draft's
      instant-failing probe could not make the calls overlap, so it
      asserted only "at most 18"; a 50 ms probe makes it exact;
    * `TestListModels_ConcurrentProbes_0053` (`openai`): five concurrent
      cold `ListModels`, 6 probes, then 0 for a warm one.
* **Step 6, the MADR's probe,** unchanged, on a copy of the working tree:
  * **M5:** 5 concurrent Lists, 503 or 401: "requests=1 wall=100ms",
    against 5 and 510 ms. A cancelled claimant: "requests=1 waiter
    wall=100ms live=true", against 2 and 130 ms.
  * **M6:** `/api/show` 1, 1, 2, 2, 3 at 0, 9, 18, 27 and 36 minutes,
    against 1 throughout.
  * **M7:** "5 concurrent cold ListModels: probe generations=6; a 6th, warm:
    +0", against 30. Every probe failing: 6, 12, 18 over three listings, as
    D-A3 keeps it.
* **Step 7, red first,** six plants, each caught by a failing test, none by
  a failed build:

  | Plant | Failure line |
  | :--- | :--- |
  | a waiter not given the flight's failure, fetching again | "HTTP 503: 5 requests, 5 callers given the failure; want 1 and 5" |
  | the listing flight on the claimant's context | "waiter: live false, <nil>, after 1 requests; want live from 1 request" |
  | re-emitted Ollama replies stamped now | "1 /api/show requests over five listings nine minutes apart; want 3" |
  | the Ollama digest ignored | "a changed digest: 1 /api/show requests; want 2" |
  | no probe flight, each caller probing | "30 probes for five concurrent listings; want 6" |
  | the probe flight on the claimant's context | "the waiter: [] after 6 probes; want [\"a\" … \"g\"] after 6" |

  The plants were run again after the last test edit (gofmt's wrapping of
  one line), all caught.
* **The checks:**
  * `tools/apidiff.py v1.19.0`: no output, exit 0;
  * `make coverage-check`: "37 packages, 0 problem(s)" (`catalog` 95.6 %,
    `internal/transport` 87.6 %, `openai` 95.7 %);
  * `make pre-add-check` on the 7 Go files: "7 file(s) clean". A first run
    found one line gofmt wraps, in `probe_flight_0053_test.go`;
  * `tools/gate.py`: 27 of 27 passed.
