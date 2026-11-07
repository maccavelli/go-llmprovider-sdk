---
status: complete
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
   failure line is recorded. *(As run, it failed 20 of 20 times on its
   store check alone, never on the removal. On Windows, Go's `testing`
   retries a removal for 2 s, and the test's issuer answers within it. A
   scratch experiment, held past 2 s, showed the removal failure. See
   Phase 4's entry.)*
3. On the primary Linux host, the same `-race` run.

### Phase 4b: the Windows removal test (amendment 2026-11-07)

The owner added this phase after Phase 4 (0071-MADR, amendment "a Windows
test of the removal"), and chose that the test skips outside Windows.

1. **The test.** `TestOAuthSessionWait_WindowsRemoval_0071` goes in
   `llmprovider/auth/oauth_wait_0071_test.go`, made from Phase 4's scratch
   experiment:
   * an issuer that answers after 3 s;
   * a `FileTokenStore` in `t.TempDir()`, holding the session;
   * a caller whose `Token` gives up at 200 ms;
   * `Wait`, under a 10 s context, in a cleanup registered after
     `t.TempDir`, which fails the test if `Wait` returns an error.

   The test's first statement skips it unless `runtime.GOOS` is
   `windows`, and names the reason.
2. **On macOS,** `go test -v -run` shows `--- SKIP` with that reason, and
   the package passes.
3. **On the Windows host,** as in Phase 4: a `git bundle` of `HEAD`,
   cloned into a new scratch directory, with the new test file copied over
   the clone's.
   * As built, `go test -count=5 -v -run` passes 5 of 5.
   * With `Wait` planted to return at once (`plantwait71.py`, in a second
     scratch clone), it fails 5 of 5. Each failure must be `TempDir
     RemoveAll cleanup` with "being used by another process".
     `tools/plant.py` runs only on this machine, so this plant runs by
     hand there, and its failure lines are recorded.
   * `go test -race -count=1 ./llmprovider/auth/...` passes.
   * The scratch clones and the bundle are then removed.
4. **Before staging,** `make pre-add-check` on the test file and the full
   gate.

If the planted `Wait` does not fail the test 5 of 5 times on Windows,
stop. A test that is not seen to fail is not committed.

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

### Phase 3 (2026-11-07)

#### Phase 2's last check

Before Phase 2 was staged, on the tree the owner then committed at
`f8fd810`, `python3 -B tools/gate.py` reported "gate: 29 of 29 passed". It
started at load 9.6.

#### What Phase 3 did

* **Step 1.**
  * `Cache`'s doc, in `llmprovider/catalog/cache.go`, describes its
    lifetime: the background work its calls may start, and that `Close`
    stops, cancels and waits for it. It says the process's cache has none,
    and that `Close` is safe for concurrent use.
  * `WithCache`'s doc says what a closed cache does: `List` and the lookups
    refuse it, the cached reads answer from it, and `ListRaw` is unaffected.
* **Step 2.** R49, in `docs/guides/api-standards.md`, gains three
  sentences:
  * `Close` ends a `Cache`'s life;
  * the process's cache has no lifetime;
  * background work a new cache starts is counted on the cache, under the
    lock its start takes.

  It cites "0048 D1 and its A5; 0049 D3; 0071 D1". The third sentence is
  the rule Phase 1's count under the lock follows, written into the guide
  so a later cache keeps to it.
* **Step 3.** `docs/architecture.md`:
  * the catalog row names `Cache`'s `Close`;
  * the auth row names `OAuthSession.Wait`;
  * the paragraph on the caches says what `Close` does, and that a caller
    that also owns an `OAuthSession` calls `Close`, then `Wait`.
* **Step 4.** `CHANGELOG.md`'s `[Unreleased]` section:
  * Added: the two methods;
  * Fixed: the two counts taken after their lock (MADR D1), and the second
    listing request (Deviation 4).
* **Step 5.** `make docs-check` reported "sdkaudit: 0 violation(s) of
  R50,R51,R52", and `make changelog-check` "40 release section(s), 0
  problem(s)". `make links-check` and `make markdownlint` returned 0.

No test changes in this phase, and no new check, so it has no planted
breaks. The full gate runs before staging, and its line is recorded at the
start of Phase 4's entry.

### Phase 4 (2026-11-07)

#### Phase 3's last checks

Before Phase 3 was staged, on the tree the owner then committed at
`731efa1`:

* `make pre-add-check` on `llmprovider/catalog/cache.go` reported "1
  file(s) clean";
* `python3 -B tools/gate.py` reported "gate: 29 of 29 passed". It started
  at load 9.4.

#### The hosts

The `available-dev-hosts` skill's verify-before-trust probe ran on both
hosts, over SSH with key authentication and `BatchMode`:

* **The Windows host:** MINGW64 on Windows, x86_64; git 2.56.0.windows.1;
  go1.27.3 windows/amd64. `CGO_ENABLED=1`, with gcc, so `-race` builds.
* **The Linux host:** Linux x86_64; git 2.56.0; go1.27.3 linux/amd64.

`go.mod` names `go 1.27.3`, and both hosts have it.

The 0071 commits are not pushed. So a `git bundle` of `main` at `731efa1`
was copied to each host and cloned into a new scratch directory, after a
check that the paths were free. `git bundle verify` returned 0. Each
clone's `git log -1` read `731efa1`.

#### Step 1 and step 3: `-race` on Windows and Linux

`go test -race -count=1 ./llmprovider/catalog/... ./llmprovider/auth/...
./llmprovider/internal/transport/...`. Step 1 named `catalog` and `auth`,
and `internal/transport` was added to match the Verification section.

| Host | `catalog` | `auth` | `internal/transport` |
| :--- | :--- | :--- | :--- |
| Windows | ok, 45.156s | ok, 10.266s | ok, 2.492s |
| Linux | ok, 56.049s | ok, 10.607s | ok, 2.563s |

#### Step 2: the planted `Wait` on Windows

* **The plant.** A second scratch clone on the Windows host got a copy of
  `oauth_session.go` with `Wait` planted to return nil at once. The copy
  was made locally by `plantwait71.py` (session scratch), which asserts
  the text it replaces occurs once and never writes the tree. `git diff
  --stat` in that clone showed that one file, with 3 insertions.
* **The run.** `go test -count=20 -v -run
  'TestOAuthSessionWait_TempDir_0071$'` failed 20 of 20 times, each with
  `oauth_wait_0071_test.go:121: after Wait the store holds … want the
  rotated refresh-1`.
* **No removal failure.** None of the 20 failures came from removing the
  temporary directory, the failure issue 12 reports. Go 1.27.3's
  `testing.removeAll` (`testing.go:1651-1670`) retries a Windows removal
  for up to 2 s (`arbitraryTimeout`), and the test's issuer answers in
  milliseconds. The refresh therefore ends within the retry, and the test
  as written cannot show the Windows failure.
* **The scratch experiment.** To prove MADR fact 3, the experiment
  `scratch_removal_0071_test.go` (session scratch, never committed) ran
  in both Windows clones. Its issuer answers after 3 s, and its caller gives
  up at 200 ms, so the refresh holds the store's refresh lock past the
  test body and past the 2 s retry. `Wait` runs in a cleanup that is
  registered after `t.TempDir`.

  | Clone | Result |
  | :--- | :--- |
  | `Wait` as built | `Wait = <nil> after 2.808s`; `--- PASS: TestScratchRemoval0071 (3.02s)` |
  | `Wait` planted | `Wait = <nil> after 0s`; `testing.go:1619: TempDir RemoveAll cleanup: unlinkat <path>: The process cannot access the file because it is being used by another process.`; `--- FAIL: TestScratchRemoval0071 (3.01s)` |

  (`<path>` stands for the host's temporary path, which is not recorded.)
  The failure is issue 12's: a refresh holding its lock file blocks the
  removal on Windows. `Wait` ends it, which is MADR fact 3.

#### After the runs

Both hosts' scratch clones and bundles were listed, removed, and checked
gone. Nothing was pushed from either host, and nothing was left on them.

### Phase 4b (2026-11-07)

#### Phase 4's last check

Phase 4's record and the amendment adding this phase were staged after
`python3 -B tools/gate.py --markdown` reported "gate: 11 of 11 passed". It
started at load 9.2. The owner committed them at `5f21797`.

#### Step 1: the test

`TestOAuthSessionWait_WindowsRemoval_0071` is in
`llmprovider/auth/oauth_wait_0071_test.go`, as step 1 describes. Its body
also fails if the caller's `Token` does not return
`context.DeadlineExceeded`. That shows the refresh is still running when
the body ends. `go vet` passes on the host and with `GOOS=windows`.

#### Step 2: macOS

`go test -count=1 -v -run 'TestOAuthSessionWait_WindowsRemoval_0071$'
./llmprovider/auth/`:

* `oauth_wait_0071_test.go:141: only Windows refuses to remove a file held
  open; elsewhere this test cannot fail`;
* `--- SKIP: TestOAuthSessionWait_WindowsRemoval_0071`;
* `ok`.

#### Step 3: the Windows host

* **The clones.** A `git bundle` of `main` at `5f21797` was copied to the
  Windows host, after a check that the paths were free, and cloned twice.
  The new test file was copied into both clones, and the planted
  `oauth_session.go` (`plantwait71.py`) into the second. `git diff --stat`
  showed the test file alone in the first clone (45 insertions), and the
  test file plus `oauth_session.go` (3 insertions) in the second.
* **As built,** `go test -count=5 -v -run …` showed 5 × `--- PASS:
  TestOAuthSessionWait_WindowsRemoval_0071`.
* **With `Wait` planted,** it showed 5 × `--- FAIL`. Each failure was
  `testing.go:1619: TempDir RemoveAll cleanup: unlinkat <path>\openai.oslock:
  The process cannot access the file because it is being used by another
  process.` `openai.oslock` is the store's refresh lock, which the refresh
  held open. (`<path>` stands for the host's temporary directory, which is
  not recorded.)
* **Race.** `go test -race -count=1 ./llmprovider/auth/...` reported `ok`.
* **Cleanup.** The clones and the bundle were listed, removed, and checked
  gone.
* **The SSH warning.** The host's SSH server answered with a warning that
  its key exchange is not post-quantum. That is the host's own
  configuration, outside this PLAN, and was left as it is.

#### Step 4

The pre-add check and the full gate run before staging. Their lines are
recorded at the start of Phase 5's entry.

### Phase 5 (2026-11-07)

#### Phase 4b's last checks

Before Phase 4b was staged, on the tree the owner then committed at
`ec37421`:

* `tools/denyscan.py --changed` reported "0 hit(s) in 2 file(s)";
* `make pre-add-check` on the test file reported "1 file(s) clean";
* `python3 -B tools/gate.py` reported "gate: 29 of 29 passed". It started at
  load 9.2.

#### Steps 1 and 2: the release's section

* **Step 1.** `CHANGELOG.md`'s `[Unreleased]` became `## [v1.28.0] -
  2026-11-07`, under an empty `[Unreleased]`. The compare links read
  `[Unreleased]: …compare/v1.28.0...HEAD` and `[v1.28.0]:
  …compare/v1.27.0...v1.28.0`.
* **The README.** `README.md:356` reads "The current release is
  `v1.28.0`." No other file outside the changelog and the records names
  `v1.27.0` as the current release.
* **The checks:**
  * `scripts/check_changelog.py` reported "41 release section(s), 0
    problem(s)";
  * `--section v1.28.0` printed the section: Added, the two methods; Fixed,
    the counts under the lock and the second listing request;
  * `tools/apidiff.py v1.27.0` reported "Compatible changes:
    ./llmprovider/catalog.(*Cache).Close: added;
    ./llmprovider/auth.(*OAuthSession).Wait: added".
* **Step 2,** the full gate, runs before staging, and its line is recorded
  with the release's.

#### Deviation 5 (2026-11-07): three Releases past the five newest

* **Found** after the tag's release job. `gh release list` showed 8
  Releases. The three past the five newest were `v1.24.1`, `v1.24.0` and
  `v1.23.0`.
* **The rule.** R53 keeps a Release for the five newest tags only. CI's
  `release` job creates the new tag's Release and deletes none. The
  release's PLAN then deletes the Release of the tag that falls to sixth
  (`0055-MADR-go-1-27-3-and-the-v1-20-0-release.md`, which chose a manual
  command over a destructive CI step). 0068-PLAN did so for `v1.26.0`.
* **The misses.** The releases of `v1.26.1` (0069-PLAN) and `v1.27.0`
  (0070-PLAN) did not delete theirs, `v1.23.0` and `v1.24.0`. Neither PLAN
  contains `gh release delete`. This PLAN's Phase 5 copied 0070's steps,
  and missed `v1.24.1`'s.
* **Before any choice,** each Release's JSON (tag, name, body, created
  date) was saved in the session's scratch directory: `v1.24.1` 7 lines,
  `v1.24.0` 18, `v1.23.0` 190. Any can be made again from its save or its
  `CHANGELOG.md` section.
* **Offered:**
  * delete all three (recommended);
  * delete `v1.24.1` alone;
  * delete none for now.
* **The owner chose** to delete all three, keeping their tags and sections.
  0069-PLAN and 0070-PLAN have dated notes. The deletion and its check are
  recorded below.

### The release (2026-11-07)

* **The release commit.** Phase 5's files were staged after `python3 -B
  tools/gate.py` reported "gate: 29 of 29 passed", started at load 9.6. The
  owner committed them at `e98c248`.
* **The push.** The owner pushed `main`, `abe3626..e98c248`, the eight 0071
  commits from `ab8db9e` to `e98c248`, through the pre-push disclosure
  guard. The push came before this record's own run of the guard. That run,
  afterwards, covered `v1.27.0..e98c248`, 10 commits, and returned 0.
* **CI on `main`** at `e98c248`:
  * CI run 42422080849, success: validate on windows-2025, macos-15 and
    ubuntu-24.04, and release skipped, as on any branch;
  * Docs run 42422080838, success.

  `auth` passed on each platform: 13.403s on Windows, 8.093s on macOS,
  10.525s on Ubuntu. CI runs `go test` without `-v`, so its log does not
  show that `TestOAuthSessionWait_WindowsRemoval_0071` ran on Windows.
  Phase 4b showed that on the Windows host.
* **The tag.** The owner made `v1.28.0`, an annotated tag (`64a53cf`), on
  `e98c248`, dated 2026-11-07 (`%cs`), the section's date, and pushed it.
  `git ls-remote` showed the same tag object and commit. CI on the tag, run
  42422640759: success, with validate on the three platforms, and release.
* **The Release.** `relcheck.py` (session scratch):
  * `v1.28.0`, Latest, is not a draft or a prerelease;
  * "Release body equals the section, 26 lines", with each `docs/` link
    written as the tag's blob URL;
  * the five newest tags each carry notes.
* **R53's deletion** (Deviation 5). `gh release delete v1.24.1 --yes`, then
  `v1.24.0` and `v1.23.0`, each returned 0. `gh release list` then showed
  five: `v1.28.0`, `v1.27.0`, `v1.26.1`, `v1.26.0`, `v1.25.0`. `git
  ls-remote` still lists the three tags.
* **`tools/consumer.py v1.28.0` and `tools/offline.py v1.28.0`.**
  sum.golang.org listed the version at the first poll, 200. In each, `go
  get` reported "added … v1.28.0". Both reported "every step passed":
  * consumer built for the six platforms and linked 16 packages;
  * offline printed "offline: ok" with "fields=19" and "requests=0", and
    built for the six platforms;
  * offline's `FAIL:` lines are its two planted steps, "caught: models.dev
    left on" and "caught: no caller layer", which are meant to fail.

### Closing (2026-11-07)

* **Goal:**
  * D1 and D2: `Cache.Close`, which stops, cancels and waits for the
    cache's background work. The waiting calls refuse after it, and the
    cached reads answer (Phase 1, with Deviations 1 to 4).
  * D3: `OAuthSession.Wait` (Phase 2). Its effect on Windows, a removal no
    longer blocked by a refresh's lock file, is shown by a test there
    (Phases 4 and 4b).
  * D4: the docs, the amended records, `CHANGELOG.md`, and the release
    `v1.28.0`, Q6's answer (Phases 0, 3 and 5).
* **The MADR's Confirmation:**
  * the D4 tests pass, each seen to fail on its planted breaks: 20 in
    Phase 1, 4 in Phase 2, and the Windows plant 5 of 5 in Phase 4b;
  * `go test -race` passed over `catalog`, `auth` and `internal/transport`
    on macOS (Phases 1 and 2), Windows and Linux (Phase 4);
  * `tools/apidiff.py v1.27.0` lists exactly `(*Cache).Close` and
    `(*OAuthSession).Wait`;
  * `Wait`, and a `t.TempDir` removal, ran on a Windows host before the
    tag (Phases 4 and 4b);
  * `tools/claims.py` passes 36 of 36 on the MADR.
* **Deviations,** each the owner's choice, recorded where they arose:
  1. the probe store's close and wait are exported within
     `internal/transport`;
  2. `cacheref` is unchanged;
  3. a provider's `ListModels` takes its own failure path, a fact the MADR
     corrects;
  4. a gap in 0053-MADR D4's single fetch per key, closed;
  5. three Releases past the five newest, deleted.

  The owner also added Phase 4b, by amendment.
* **Not done, and why:**
  * a `Wait` for `CommandToken`: Q3's answer leaves it until a caller needs
    one, and its doc names the early run as detached;
  * clearing a closed cache's contents: Q2's answer keeps them readable;
  * issue 12, which the release answers. Its comment and closing are the
    owner's word.
    (2026-11-07, after this entry: on the owner's word, issue 12 was
    answered with a comment linking the release and this record's pair,
    and closed as completed.)
* `tools/records.py status` sets this PLAN `complete`.
