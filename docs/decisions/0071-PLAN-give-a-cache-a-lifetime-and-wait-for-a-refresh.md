---
status: in-progress
date: 2026-11-07
associated-madr: "0071-MADR-give-a-cache-a-lifetime-and-wait-for-a-refresh.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Cache's Lifetime and a Wait for an OAuth Refresh

Associated MADR: [0071-MADR-give-a-cache-a-lifetime-and-wait-for-a-refresh.md](0071-MADR-give-a-cache-a-lifetime-and-wait-for-a-refresh.md)

## Goal

A caller that owns a `catalog.Cache` can end the work that the cache
started. After `(*Cache).Close(ctx)` returns nil, nothing the cache started is
running, and nothing it will start can run. A caller that owns an
`auth.OAuthSession` can wait for its detached refresh with
`(*OAuthSession).Wait(ctx)`. The process's cache, and every caller that
gives no cache, behaves as today. This answers GitHub issue 12.

The steps below follow the MADR's recommended answers to Q1 to Q6. If the
owner answers otherwise, the MADR and this PLAN are amended before Phase 1.
(2026-11-07: the owner answered each question as recommended, and the MADR
is accepted. This PLAN awaits approval.)

## Scope

### In scope

* The cache's lifetime (MADR D1):
  * `llmprovider/catalog/cache.go`;
  * `listing_cache.go` and `discovery.go`: `Go` moves under the lock;
  * `model_metadata.go`;
  * `llmprovider/internal/cacheref/cacheref.go`;
  * `llmprovider/internal/transport/probe.go`.
* A closed cache (MADR D2):
  * the refusal in `List`, `LookupModel` and `LookupMetadataWith`;
  * no start in `CachedModel` and `CachedMetadataWith`;
  * no probe run.
* `OAuthSession.Wait` (MADR D3), in `llmprovider/auth/oauth_session.go`,
  and a sentence in `CommandToken`'s doc naming its early run as detached.
* The tests in MADR D4, an `Example` for each new method, the docs, and the
  release.

### Exported names (R55)

| Name | Call site that needs it |
| :--- | :--- |
| `(*catalog.Cache).Close(ctx context.Context) error` | gobble-cli's tests and process end, which own a `Cache` given with `WithCache` (issue 12; gobble-cli's `0046-MADR-own-background-listings-and-keep-tests-hermetic.md`, Q2) |
| `(*auth.OAuthSession).Wait(ctx context.Context) error` | gobble-cli's login tests, which leave a refresh running past their end (issue 12, `TestLoginAsksThroughQuestions`) |

No other name is exported. The closed-cache error wraps
`llmprovider.ErrInvalidRequest`, with no new sentinel (MADR Q4).

### Out of scope

* A `Wait` for `CommandToken` (MADR Q3).
* Closing the process's cache.
* `ListRaw`, which touches no cache (MADR C22).
* Any change to the detachment, the timeouts, the single-flights, or
  0026-MADR F3's rotation rule.
* gobble-cli's own change. It passes no token source in its tests, in its
  own repository.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts only after the owner has committed the last one.
* **Planted breaks.** Every new test calls a new name, so none can be red at
  `abe3626`. Each is instead seen to fail on each of its planted breaks, by
  `python3 -B tools/plant.py SPEC.json` on a scratch clone. The specs live
  in the session's scratch directory, and this record holds each failure
  line. If a break cannot be made to fail its test on every run, that is a
  deviation: stop and report it. Do not loosen the test.
* **The drift check.** Every existing test passes unchanged, the module's
  own waits on the groups (MADR C26, C27) included (R46).
* **Pre-add.** `make pre-add-check FILES="…"` on the phase's Go files: gofmt,
  golangci-lint on the host and with `GOOS=windows`, vet, the packages'
  tests, and govulncheck.
* **Race.** `go test -race -count=3` over `./llmprovider/catalog/...`,
  `./llmprovider/auth/...` and `./llmprovider/internal/transport/...`.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`, or `--markdown` for a records-only phase;
  * `python3 -B tools/claims.py` on the MADR;
  * `make records-check`, `python3 -B tools/recordlint.py --changed`,
    `python3 -B tools/denyscan.py --changed`, `python3 -B tools/linkcheck.py`.

  Each exit status is captured before any filter, and read before anything
  is written from it. Files are staged only when the gate passes (`&&`).
* **R50 to R52.** Each new method has a doc comment and an `Example` with an
  `// Output:` comment. Each doc names its error kinds: nil, or the
  context's error. `Cache`'s doc already says it is safe for concurrent use.
  It gains that `Close` is safe to call concurrently with every other use.
  `make docs-check` stays at 0.
* **Stage, then the owner commits** with `git commit --no-edit`. Pushes and
  tags happen on the owner's word, after the disclosure guard has run over
  the outgoing commits.
* **Stop conditions:**
  * a fact of the MADR, or one of its claims, no longer holds;
  * an existing test needs a changed assertion;
  * `apidiff` against `v1.27.0` lists anything beyond the two methods;
  * a golden changes;
  * a package falls below its floor in `scripts/coverage-floors.txt` (R47);
  * `-race` reports a race, or a group panics.

  On any of these: stop, record a dated deviation here (with an amendment in
  the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR and their rows in `docs/README.md` as records
   only.
2. Once the owner has answered Q1 to Q6:
   * the MADR records the answers, as `accepted`;
   * `tools/records.py status` sets this PLAN `in-progress`.
3. Add dated amendments, without rewriting the decisions:
   * **`0048-MADR-adopt-go-sdk-best-practices.md`, D1:** a `Cache` has a
     lifetime. `Close` stops new background work and waits for running work.
     The process's cache has none (0071-MADR D1).
   * **`0053-MADR-stream-retry-and-cache-defects.md`, D4 and D6:** the
     detached fetch and the probe run stay detached from their callers. Each
     is now owned by its cache, whose `Close` cancels and waits for it
     (0071-MADR D1).
4. Run `python3 -B tools/gate.py --markdown`, then stage.

### Phase 1: the cache's lifetime (D1, D2)

1. **`store`** gains:
   * `closed bool`, set under `listingMu` and under `metadataMu`;
   * `life context.Context` and `end context.CancelFunc`.

   `newStore` makes `life`. The process's store is never ended.
2. **A shared helper** gives detached work its context:
   `context.WithTimeout(context.WithoutCancel(ctx), d)`, plus
   `context.AfterFunc(store.life, cancel)`. Its `stop` is deferred, so the
   work ends when the caller's timeout or the owner's cancel comes first.
3. **`startListingFetch`:**
   * it returns at once when `closed` is set;
   * it calls `listingFetching.Go` before `listingMu` is released;
   * its context comes from step 2.
4. **`cachedOrFetched`:**
   * `claimListing` reports a closed store;
   * the leader calls `Go` under the claim's lock;
   * the context comes from step 2.
   * On a closed store, `List` has already refused (step 7), so this path is
     not reached. The check is the guard behind that refusal.
5. **`startMetadataFetch`** (lock already held):
   * it starts nothing when `closed` is set;
   * `cachedMetadata` then answers from the entry;
   * `loadModelMetadata` returns the refusal (step 7);
   * its context comes from step 2.
6. **`transport.ProbeStore`** gains:
   * `closed` and a group, under `mu`;
   * `life` and `end`;
   * ~~an unexported close-and-wait~~ a close-and-wait, exported from
     `internal/transport` so the catalog can call it. Being internal, it
     adds nothing to the module's API (Deviation 1).

   `ProbeGenerateHealthCached` on a closed store starts no run and returns
   `withTail(nil, …)`. The run goes on the group and takes the step 2 form
   of its context. `defaultProbes` is never closed.
7. **The refusal.** One unexported error helper returns
   `fmt.Errorf("catalog: the cache is closed: %w", llmprovider.ErrInvalidRequest)`.
   It is checked before the network:
   * in `List` after `listingConfig`;
   * in `LookupModel` after `lookupConfig`;
   * in `LookupMetadataWith` after `configFor`.
8. ~~**`cacheref.Value`** gains a close-and-wait over its catalog store, which
   it reaches through a small interface the catalog's store meets, and over
   its probe store. `cacheref` stays internal.~~ Struck by Deviation 2:
   `Cache.Close` reaches both stores through `c.ref`'s fields, and
   `cacheref` is unchanged.
9. **`(*Cache).Close(ctx)`** calls `c.value()` and closes once. Each call
   then waits under its own `ctx` on the three groups (MADR Q5). It returns
   nil, or `ctx.Err()`.
10. **Tests**, in `llmprovider/catalog/close_0071_test.go` and
    `llmprovider/internal/transport/probe_close_0071_test.go`. Each is listed
    with its planted breaks.

    | Test | Asserts | Planted breaks |
    | :--- | :--- | :--- |
    | `TestCacheClose_WaitsForFetch_0071` | With a `CachedModel` fetch blocked until its context is done, `Close` returns nil only after the fetch has returned, and the in-flight map is empty. | `Close` returns before the wait; the owner context is not cancelled (so `Close` waits for the 10 s timeout, past the test's 2 s bound) |
    | `TestCacheClose_MetadataFetch_0071` | The same, for a models.dev refresh. | the metadata group is left out of the wait |
    | `TestCacheClose_Deadline_0071` | With a fetch that ignores cancellation, `Close(ctx)` returns `ctx.Err()` at a 100 ms deadline. Once the fetch is released, a second `Close` returns nil. | `Close` returns nil at the deadline; a second `Close` returns the first's error |
    | `TestCacheClose_NoStartAfter_0071` | After `Close`, `CachedModel` and `CachedMetadataWith` send no request to the test server, and they start no fetch. | the closed check is removed from `startListingFetch`; the same for `startMetadataFetch` |
    | `TestCacheClose_Refuses_0071` | After `Close`, `List`, `LookupModel` and `LookupMetadataWith` return an error matching `ErrInvalidRequest`, and no request is sent. ~~A provider's `ListModels` given the cache returns it too.~~ A provider's `ListModels` given the cache sends no request: Together's returns its static list, and Ollama's returns the refusal (Deviation 3). | each of the three refusals removed, one at a time |
    | `TestCacheClose_CountUnderLock_0071` | A start that has passed its closed check, held there by a test hook, is waited for by a concurrent `Close`. | `Go` is moved back after the unlock, at each of the two sites |
    | `TestCacheClose_Race_0071` | 64 goroutines call `CachedModel`, `List` and probes while `Close` runs, under `-race`. Nothing panics, and nothing runs after `Close` returns. | none: it is a race test. Its rows above carry the plants. |
    | `TestCacheClose_Zero_0071` | A zero `Cache` closes at once with nil, and a later `List` given it is refused. | `Close` on an unused `Cache` does not mark it closed |
    | `TestProbeStoreClose_0071` | A closed probe store starts no run, and a run in flight is waited for. | the probe store's closed check is removed; its run is left uncounted |
    | `ExampleCache_Close` | A cache given to `CachedModel` over `httptest`, closed, prints `nil`. *(As written: it prints `<nil>`, what `fmt.Println` gives a nil error, then `true` for a `List` that `Close` has refused.)* | the example's output, planted wrong |

    The hook in `TestCacheClose_CountUnderLock_0071` is an unexported
    package variable that is nil outside tests. It is a test seam of the same kind as `listingNow`, the clock that tests move. If no hook
    can make the plant fail on every run, that is a stop condition, not a
    reason to drop the test.

### Phase 2: `OAuthSession.Wait` (D3)

1. **`(*OAuthSession).Wait(ctx)`** loops:
   * under `s.mu`, it reads `s.inflight`;
   * when that is nil, it returns nil;
   * otherwise it waits on its `done` channel or on `ctx`, and returns
     `ctx.Err()` if `ctx` ends first.

   The doc says that `Wait` cancels nothing, that a rotation is saved before
   it returns, and that a caller owning a `Cache` calls `Close` first (MADR
   D3).
2. **`CommandToken`'s doc** gains a sentence: an early run is detached from
   its caller and bounded by `Timeout` (MADR Q3).
3. **Tests**, in `llmprovider/auth/oauth_wait_0071_test.go`:

   | Test | Asserts | Planted breaks |
   | :--- | :--- | :--- |
   | `TestOAuthSessionWait_AfterSave_0071` | A caller's `Token` is cancelled while the issuer stub holds the refresh. Once the stub answers, `Wait` returns nil only after the store has saved the rotated token. | `Wait` returns nil at once |
   | `TestOAuthSessionWait_Deadline_0071` | While the stub still holds the refresh, `Wait` returns `ctx.Err()` at a 100 ms deadline. | `Wait` ignores `ctx` (the test's own 2 s bound fails it) |
   | `TestOAuthSessionWait_Idle_0071` | A session with nothing in flight returns nil at once. | `Wait` blocks when `inflight` is nil |
   | `TestOAuthSessionWait_TempDir_0071` | A `FileTokenStore` in `t.TempDir()`, a detached refresh, and `Wait` in `t.Cleanup` before the directory is removed. The removal succeeds. | `Wait` returns nil at once. This break fails the test on Windows only, where an open handle blocks removal; it is planted there in Phase 4. *(As written, the test also checks after `Wait` that the store holds the rotated session. So the break fails it on every platform, and it failed on macOS in Phase 2. Phase 4 still plants it on Windows, for the removal.)* |
   | `ExampleOAuthSession_Wait` | A session over `httptest`, refreshed and waited for, prints `nil`. | the example's output, planted wrong *(As written: its caller gives up at a 20 ms deadline, so it prints `context deadline exceeded`, then `<nil>` from `Wait`, then `fresh-access <nil>` from the next `Token`.)* |

### Phase 3: docs (D4)

1. **`Cache`'s doc** describes the lifetime. **`WithCache`'s doc** says that a
   closed cache refuses `List` and the lookups.
2. **`docs/guides/api-standards.md` R49** gains: "`Close` ends a `Cache`'s
   life: it stops new background work and waits for running work; the
   process's cache has none (0071 D1)." This is the 0048-MADR D1 amendment
   made in Phase 0.
3. **`docs/architecture.md`:** the catalog row names `Close`, and the auth
   row names `Wait`.
4. **`CHANGELOG.md`:** under `[Unreleased]`, Added: the two methods. Fixed:
   the two counts taken after their lock.
5. Run `make docs-check`, `make changelog-check` and `make links-check`.

### Phase 4: other platforms

The changes touch no build tag or path, but issue 12's failure was
Windows-only. Using the `available-dev-hosts` skill, after its
verify-before-trust probe:

1. On the Windows host, `go test -race` over `./llmprovider/catalog/...`
   and `./llmprovider/auth/...`, including
   `TestOAuthSessionWait_TempDir_0071`.
2. On that host, `TestOAuthSessionWait_TempDir_0071` is seen to fail with
   `Wait` planted to return at once, in a scratch clone there. The test's
   failure line is recorded.
3. On the primary Linux host, the same `-race` run.

### Phase 5: the release (Q6: `v1.28.0`)

As 0070-PLAN's Phase 5:

1. The changelog section is renamed `v1.28.0`, and the README's current
   release is updated.
2. Run the full gate. `tools/apidiff.py v1.27.0` lists the two methods.
3. Tag on the owner's word.
4. Wait for CI and the Release, then run `tools/consumer.py` and
   `tools/offline.py`.
5. Comment on issue 12, and close it, on the owner's word.

## Verification

* Every test in Phases 1 and 2 passes, and each was seen to fail on each of
  its planted breaks. This record holds each failure line.
* `go test -race -count=3` is clean over the three packages, on macOS,
  Windows and Linux.
* Every existing test passes unchanged, and no golden changes.
* `apidiff` against `v1.27.0` lists exactly `(*Cache).Close` and
  `(*OAuthSession).Wait`.
* The MADR's claims pass, with C5a, C6 and C8 annotated as changed by
  Phase 1.
* Coverage holds each package's floor.

## Rollout and Rollback

* **Rollout.** The change is additive and opt-in. A caller that never calls
  `Close` sees no change, except that a count is now taken under its lock,
  which changes no result.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** the two methods stay (R48). A defect in `Close`
  is fixed forward. If the defect cannot be fixed, the method is documented
  `Deprecated:` and made to wait without closing, through a new record.

## Execution record

### Phase 0 (2026-11-07)

* **Step 1.** The MADR, this PLAN and their index rows were committed as
  records only, at `ab8db9e`, after `tools/gate.py --markdown` passed 11 of
  11. The MADR's claims passed 32 of 32 at `abe3626`.
* **Step 2.** The owner answered Q1 to Q6 as recommended, with three options
  offered for each, and the MADR is `accepted`. The owner approved this PLAN
  ("committed proceed"), and `tools/records.py status` set it
  `in-progress`.
* **Step 3.** The amendments were appended:
  * `0048-MADR-adopt-go-sdk-best-practices.md`, "A5 (2026-11-07): a
    `Cache` has a lifetime";
  * `0053-MADR-stream-retry-and-cache-defects.md`, "Amendment 2026-11-07:
    D4 and D6, the detached work has an owner".

  Neither rewrites a decision.
* **Step 4.** `python3 -B tools/gate.py --markdown` reported "gate: 11 of
  11 passed". Each of these checks returned 0:
  * `parity-check`, `records-check`, `tables-check`, `changelog-check`;
  * `markdownlint`, `links-check`, `disclosure-check`, `gate-selftest`;
  * `deny-scan`, `recordlint`, `claims`.

  It was run again after this entry was written, and the files were then
  staged. The owner committed Phase 0 at `0312f1c`.

### Phase 1 (2026-11-07)

#### Deviation 1 (2026-11-07): the probe store's close-and-wait is exported from `internal/transport`

* **Found** before any code was written. Step 6 called the probe store's
  close-and-wait unexported. `catalog` calls it, and `catalog` is a
  different package from `internal/transport`, so it cannot reach an
  unexported method.
* **Offered:**
  * export the methods inside `internal/transport` (recommended);
  * move the probe flights into `catalog`;
  * leave probes out of `Close`, which contradicts 0071-MADR D1.
* **The owner chose** the first. `internal/` cannot be imported from
  outside the module, so the API is unchanged, and `apidiff` still lists
  only the two methods. Step 6 is annotated. The MADR is unchanged: it does
  not say whether the methods are exported.

#### Deviation 2 (2026-11-07): `cacheref` is unchanged

* **Found** before any code was written. `Cache.Close` can reach both stores
  through `c.ref`'s fields: `Catalog` is the catalog's `*store`, and `Probes`
  is the probe store. Step 8's interface would serve no other caller.
* **Offered:**
  * close in `catalog`, through those fields (recommended);
  * step 8 as written;
  * a `Value.Close` for the probe store alone.
* **The owner chose** the first. Step 8 is struck through, and
  `llmprovider/internal/cacheref` leaves this phase's scope. The MADR is
  unchanged.

#### Deviation 3 (2026-11-07): a provider's `ListModels` takes its own failure path

* **Found** after steps 1 to 7 and 9 were written, before the tests. The
  steps had built and vetted, and the existing tests had passed under
  `-race` (35 packages). MADR D2 said a built-in provider's `ListModels`
  returns the closed-cache error. Only Ollama's and chatcompat's do. The
  other eight fall back to their static list, with a nil error, on any
  error from `catalog.List`.
* **Offered:**
  * correct the fact, and assert both behaviours (recommended);
  * make all ten return the error, through an internal marker;
  * drop the claim and the assertion.
* **The owner chose** the first. The MADR has the amendment "a provider's
  `ListModels` takes its own failure path" (C33 to C35), and its D2 sentence
  is annotated. The `TestCacheClose_Refuses_0071` row is annotated, and
  `Close`'s doc says what each kind of provider does. No provider changes.
  The provider assertions need `providers`, which `catalog`'s internal tests
  cannot import, so they go in an external test file,
  `llmprovider/catalog/close_ext_0071_test.go`, beside the PLAN's
  `close_0071_test.go`.

#### Deviation 4 (2026-11-07): a second leader came through the gap before a claim

* **Found** by the full gate, after the execution record below was written:
  * `pre-add-check` failed with `cache_0048_test.go:136: 2 listing
    request(s) for 32 concurrent listings of one cache, want 1`, in
    `TestCache_ConcurrentUse_0048`, an existing test. The gate started at
    load 9.3.
  * Measured with `flake0048.py` (session scratch): the tree and a clone of
    `0312f1c` (no Phase 1 code) ran at the same time, under load, at `-cpu`
    1, 2, 4, 8 and 16.
  * The tree failed 9 times in 1,000 runs, and `0312f1c` 0 times in 1,000.
  * Each side alone, with no load, failed 0 times in 500 runs.
* **The cause**, from the code: `cachedOrFetched` checks for a fresh listing,
  releases `listingMu`, then claims the key. A leader that stores its
  listing and gives up its claim in that gap leaves the caller to lead a
  second fetch. The gap is 0053-MADR D4's, and it is present at
  `0312f1c`. Phase 1's `refuseClosed` takes `listingMu` once more for
  each `List`, which most likely widens it: that is inferred from the
  rates, not proven.
* **Offered:**
  * close the gap (recommended);
  * make the refusal lock-free and leave the gap;
  * both.
* **The owner chose** the first:
  * `claimListing` checks for the fresh listing again under its lock, and
    returns it as a finished call;
  * `listingClaimHook`, a test seam, runs between the check and the claim;
  * the new `TestCachedOrFetched_NoSecondLeader_0071` holds a caller there
    while another lists to the end;
  * `0053-MADR-stream-retry-and-cache-defects.md` has a dated note on D4.

  0071-MADR is unchanged: none of its decisions or facts changes. The
  results are recorded under "After Deviation 4" below.

#### What Phase 1 did

* **Steps 1 to 5, 7 and 9**, as written, in:
  * `llmprovider/catalog/cache.go`: the store's flags, owner context,
    `detached`, `refuseClosed`, `close` and `wait`, and `Cache.Close`;
  * `listing_cache.go`: `startListingFetch` and `claimListing` count
    under `listingMu`. `claimListing` now takes the fetch to start.
  * `discovery.go`: `List`'s refusal, and `cachedOrFetched` through the
    new `claimListing`;
  * `model_metadata.go`: `startMetadataFetch`'s check, `loadModelMetadata`'s
    guard, and `LookupMetadataWith`'s refusal;
  * `lookup.go`: `LookupModel`'s refusal.
* **Step 6** in `llmprovider/internal/transport/probe.go`, as Deviation 1
  amended it.
* **Step 8** was not done (Deviation 2).
* **Not in the steps:**
  * `Cache` keeps a typed pointer to its store, set where `value` makes
    the store. golangci-lint's `errcheck` refused an unchecked type
    assertion of `c.ref.Catalog` in `Close`, and the typed pointer needs
    none.
  * `listingStartHook`, the test seam the hook row names, is checked at
    both counted starts.
* **One test was added to the table:** `TestCacheClose_ProbeRun_0071`. It
  shows that `Close` closes and waits for the cache's probe store, a wiring
  that `TestProbeStoreClose_0071` cannot reach. Without it, two planted
  breaks below would have gone uncaught. `TestCacheClose_Refuses_0071` warms
  the cache before `Close`, so that a removed refusal answers from it and
  is seen.

#### The tests and the planted breaks

All 12 new tests (ten in `close_0071_test.go`, one in
`close_ext_0071_test.go`, one in `probe_close_0071_test.go`) and the
example pass under `-race`. `tools/plant.py` ran `plant71-p1.json`
(session scratch) five times:

* runs 1 to 3 had 19 plants, each run reporting "plant: all 19 plant(s)
  caught". Runs 1 and 2 were before the `errcheck` fix, run 3 after it.
* run 4 followed Deviation 4 and had 20 plants. One no longer matched the
  code (`"old" matches 0 time(s)`), and the other 19 were caught.
* run 5 had that plant rewritten for the new `claimListing`, and reported
  "plant: all 20 plant(s) caught".

The failure lines of run 5, on the final code:

| Planted break | Failure line |
| :--- | :--- |
| `Close` returns before the wait | `close_0071_test.go:126: Close returned with 1 fetch(es) in flight; want 0`, with `:141`, `:157` and `:282` |
| the owner context is not cancelled | `close_0071_test.go:123: Close = context deadline exceeded; want nil once the fetch has ended`, with `:138` and `:279` |
| the metadata group is left out of the wait | `close_0071_test.go:141: Close returned with 1 models.dev fetch(es) in flight; want 0` |
| the probe store is not closed | `close_0071_test.go:361: Close = context deadline exceeded; want nil once the probe run has ended`; `close_0071_test.go:340: 21 probe(s) ran after Close returned` |
| the probe group is left out of the wait | `close_0071_test.go:364: Close returned before the probe run had ended` |
| `Close` returns nil at the deadline | `close_0071_test.go:157: Close with a stubborn fetch = <nil>; want context.DeadlineExceeded` |
| a second `Close` returns the first's result | `close_0071_test.go:164: second Close after the fetch was released = context deadline exceeded; want nil` |
| `Close` on an unused `Cache` does not mark it closed | `close_0071_test.go:409: List given a Cache closed before its first use = <nil>; want the closed-cache refusal`, with `:182`, `:412` and `close_ext_0071_test.go:55` |
| `startListingFetch` has no closed check | `close_0071_test.go:182: after Close: 1 fetch(es), 0 claim(s), 0 models.dev fetch(es) started; want none`, with `:334` |
| `startMetadataFetch` has no closed check | `close_0071_test.go:182: after Close: 0 fetch(es), 0 claim(s), 1 models.dev fetch(es) started; want none`, with `:330` |
| `List` does not refuse | `close_0071_test.go:221: List after Close = <nil>; want catalog: the cache is closed: llmprovider: invalid request`, with `:409`, `close_ext_0071_test.go:55: 1 request(s) after Close; want 0`, and `--- FAIL: ExampleCache_Close` |
| `LookupModel` does not refuse | `close_0071_test.go:221: LookupModel after Close = <nil>; want catalog: the cache is closed: llmprovider: invalid request` |
| `LookupMetadataWith` does not refuse | `close_0071_test.go:221: LookupMetadataWith after Close = <nil>; want catalog: the cache is closed: llmprovider: invalid request` |
| `startListingFetch` counts after the unlock | `close_0071_test.go:282: Close returned with 1 fetch(es), 0 claim(s), 0 models.dev fetch(es) in flight; want none` |
| `claimListing` counts after the unlock | `close_0071_test.go:282: Close returned with 0 fetch(es), 1 claim(s), 0 models.dev fetch(es) in flight; want none` |
| `claimListing` does not check for the fresh listing again (Deviation 4) | `close_0071_test.go:394: 2 listing request(s) for two callers of one key; want 1` |
| the example's output is wrong | `--- FAIL: ExampleCache_Close (0.00s)` |
| the probe store has no closed check | `probe_close_0071_test.go:59: closed store: [m], 1 probe(s); want nil and none`, with `close_0071_test.go:340` |
| the probe run is uncounted | `probe_close_0071_test.go:45: Wait returned before the run's probe had ended`, with `close_0071_test.go:364` |
| the probe store's `Close` does not cancel | `probe_close_0071_test.go:42: Wait did not return after Close`, with `close_0071_test.go:361` |

#### Checks

* **The drift check.** Every existing test passed unchanged under
  `go test -race -count=1 ./llmprovider/... ./wizard/...`, in 35 packages,
  before the new tests were written. The module's own waits on the groups
  (MADR C26, C27) are unchanged.
* **Race.** `go test -race -count=3` over `catalog`, `auth` and
  `internal/transport` passed, with no `DATA RACE` report.
* **Pre-add.** `make pre-add-check` on the phase's 10 Go files reported
  "10 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)",
  after the `errcheck` fix.
* **API.** `tools/apidiff.py v1.27.0` reported one compatible change:
  "./llmprovider/catalog.(*Cache).Close: added".
* **The claims.** `tools/claims.py` passes 35 of 35 on the MADR, at its
  marker `abe3626`. C5a, C6 and C8 describe the code Phase 1 changed, as
  the Verification section foresaw.
* **The gate.** `python3 -B tools/gate.py` reported "gate: 29 of 29
  passed". It waited for the load to fall from 22.2, and started at 9.6.
  Among the 29: `test-race-cover`, `test-shuffle`, `test-repeat`, `lint`,
  `coverage-check`, `api-check`, `docs-check`, and vet on darwin, linux and
  windows, with and without `live_gateways`. After that line was added,
  the full gate ran again, since `--markdown` refuses a tree with Go
  changes. That run reported "gate: 28 of 29 passed": `pre-add-check`
  failed on `TestCache_ConcurrentUse_0048`, which is Deviation 4.

#### After Deviation 4

* `TestCachedOrFetched_NoSecondLeader_0071` and `TestCache_ConcurrentUse_0048`
  pass under `-race`.
* Re-measured with `flake0048.py`, under load, side by side at `-cpu` 1,
  2, 4, 8 and 16:
  * the tree failed 0 times in 1,000 runs;
  * the clone of `0312f1c` failed once in 1,000. So the gap was there
    before this PLAN.
* `make pre-add-check` on the 10 Go files: "10 file(s) clean (gofmt,
  golangci-lint, go vet, go test, govulncheck)".
* The plants: run 5, above.
* The full gate, and `go test -race -count=3` over the three packages, run
  again before the phase is staged: the race run above predates Deviation 4.
  Their lines are recorded at the start of Phase 2's entry, as
  0070-PLAN recorded its phases' gate lines.

### Phase 2 (2026-11-07)

#### Phase 1's last checks

Before Phase 1 was staged, on the tree the owner then committed at
`f05119d`:

* `go test -race -count=3` over `catalog`, `auth` and `internal/transport`
  passed, with no `DATA RACE` report.
* `python3 -B tools/gate.py` reported "gate: 29 of 29 passed". It started at
  load 9.7.

#### What Phase 2 did

* **Step 1.** `(*OAuthSession).Wait(ctx)` is in
  `llmprovider/auth/oauth_session.go`, beside `waitToken`. It loops on
  `s.inflight` under `s.mu`. It returns nil when nothing is in flight, and
  `ctx.Err()` when `ctx` ends first. When `ctx` ends just as the refresh
  does, it looks once more and returns nil if the refresh has finished.

  Its doc says that `Wait` cancels nothing, that the refresh lock is released
  and the save attempted before it returns nil, and that a `Cache`'s owner
  calls `Close` first (MADR D3).
* **Step 2.** `CommandToken`'s doc, in `llmprovider/command_token.go`, now
  says that the early run is detached from the caller's context, that
  nothing waits for it, and that it may outlive the caller by up to
  `Timeout` (MADR Q3).
* **Step 3.** The four tests are in
  `llmprovider/auth/oauth_wait_0071_test.go`, and `ExampleOAuthSession_Wait`
  is in `llmprovider/auth/example_test.go`. The tests reuse
  `newRotatingIssuer`, `failingStore` and `expiredStoredSession`, from
  0026-MADR F3's tests. Two rows of step 3's table are annotated: the
  temporary-directory test's prediction, and the example's output.

#### The planted breaks

`tools/plant.py` ran `plant71-p2.json` (session scratch). The first run was
stopped after 600 s: under "Wait blocks when nothing is in flight", the
example's `Wait(context.Background())` never returns, and `go test` would
have failed only at its 10-minute default. The tree was checked afterwards:
it held no plant text, and no test process was left. The spec now passes
`-timeout=60s`, and the second run reported "plant: all 4 plant(s) caught":

| Planted break | Failure line |
| :--- | :--- |
| `Wait` returns nil at once | `oauth_wait_0071_test.go:58: Wait = <nil>, with the store holding "old-refresh"; want nil once refresh-1 is saved`, with `:78` and `:121` |
| `Wait` ignores `ctx` | `oauth_wait_0071_test.go:81: Wait did not return at its 100ms deadline` |
| `Wait` blocks when nothing is in flight | `oauth_wait_0071_test.go:92: Wait on an idle session = context canceled, want nil`, with `:58`, `:117`, and the example timed out at 60 s |
| the example's output is wrong | `--- FAIL: ExampleOAuthSession_Wait (0.20s)` |

#### Checks

* **Pre-add.** `make pre-add-check` on the phase's four Go files reported
  "4 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)".
* **Race.** `go test -race -count=3` over `catalog`, `auth` and
  `internal/transport` passed, with no `DATA RACE` report.
* **API.** `tools/apidiff.py v1.27.0` reported two compatible changes, the
  two names this PLAN exports:
  * "./llmprovider/catalog.(*Cache).Close: added";
  * "./llmprovider/auth.(*OAuthSession).Wait: added".
* **The gate, first run.** It reported "gate: 27 of 29 passed". `docs-check`
  and `gate-selftest` both failed on one finding, `oauth_session.go:218 R52
  OAuthSession.Wait returns an error its doc does not name`:
  * sdkaudit's R52 pattern wants a word such as "error" in the doc, and
    "ctx.Err()" does not match it;
  * `Close`'s doc passed because it says "an error matching";
  * `Wait`'s doc now says "the context's error, ctx.Err(), if ctx ends
    first; it returns no other error", which the PLAN's R50 to R52 rule
    asks for;
  * `make docs-check` then reported "sdkaudit: 0 violation(s) of
    R50,R51,R52".
* **The gate, second run.** It runs before the phase is staged, and its
  line is recorded at the start of Phase 3's entry.
