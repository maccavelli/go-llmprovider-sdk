---
status: proposed
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
   * an unexported close-and-wait.

   `ProbeGenerateHealthCached` on a closed store starts no run and returns
   `withTail(nil, …)`. The run goes on the group and takes the step 2 form
   of its context. `defaultProbes` is never closed.
7. **The refusal.** One unexported error helper returns
   `fmt.Errorf("catalog: the cache is closed: %w", llmprovider.ErrInvalidRequest)`.
   It is checked before the network:
   * in `List` after `listingConfig`;
   * in `LookupModel` after `lookupConfig`;
   * in `LookupMetadataWith` after `configFor`.
8. **`cacheref.Value`** gains a close-and-wait over its catalog store, which
   it reaches through a small interface the catalog's store meets, and over
   its probe store. `cacheref` stays internal.
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
    | `TestCacheClose_Refuses_0071` | After `Close`, `List`, `LookupModel` and `LookupMetadataWith` return an error matching `ErrInvalidRequest`, and no request is sent. A provider's `ListModels` given the cache returns it too. | each of the three refusals removed, one at a time |
    | `TestCacheClose_CountUnderLock_0071` | A start that has passed its closed check, held there by a test hook, is waited for by a concurrent `Close`. | `Go` is moved back after the unlock, at each of the two sites |
    | `TestCacheClose_Race_0071` | 64 goroutines call `CachedModel`, `List` and probes while `Close` runs, under `-race`. Nothing panics, and nothing runs after `Close` returns. | none: it is a race test. Its rows above carry the plants. |
    | `TestCacheClose_Zero_0071` | A zero `Cache` closes at once with nil, and a later `List` given it is refused. | `Close` on an unused `Cache` does not mark it closed |
    | `TestProbeStoreClose_0071` | A closed probe store starts no run, and a run in flight is waited for. | the probe store's closed check is removed; its run is left uncounted |
    | `ExampleCache_Close` | A cache given to `CachedModel` over `httptest`, closed, prints `nil`. | the example's output, planted wrong |

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
   | `TestOAuthSessionWait_TempDir_0071` | A `FileTokenStore` in `t.TempDir()`, a detached refresh, and `Wait` in `t.Cleanup` before the directory is removed. The removal succeeds. | `Wait` returns nil at once. This break fails the test on Windows only, where an open handle blocks removal; it is planted there in Phase 4. |
   | `ExampleOAuthSession_Wait` | A session over `httptest`, refreshed and waited for, prints `nil`. | the example's output, planted wrong |

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

No phase has run. The record begins once the owner approves this PLAN.
