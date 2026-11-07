---
status: accepted
date: 2026-11-07
decision-makers: repository owner
consulted: this module at abe3626 (v1.27.0 is the last tag; abe3626 changes only records since); GitHub issue 12 of this repository, opened 2026-11-07, and the gobble-cli measurement it reports; the Go 1.27.3 documentation of sync.WaitGroup.Go and context.AfterFunc; net/http's Server.Shutdown as prior art
informed: anyone who gives a catalog.Cache to a listing or a provider, and anyone who uses an auth.OAuthSession in a test or a short-lived process
---

<!-- markdownlint-disable MD013 -->

# Give a Cache a Lifetime, and Let a Caller Wait for an OAuth Refresh

## Context and Problem Statement

GitHub issue 12 asks for a way to end the SDK's background work. The catalog
starts network work that no caller can end or wait for. The issue lists
three kinds:

* `CachedModel`'s background listing fetch;
* `List`'s shared fetch (0053-MADR D4);
* the models.dev document's refresh.

Each runs on `context.WithoutCancel` of the caller's context (C5, C7, C9),
for up to its own timeout. Each is counted in a `sync.WaitGroup` field of the
cache's `store` (C2, C3). Those groups are unexported, and their comments say
they exist "so tests can wait": only this module's tests wait on them (C26,
C27).

The detachment is deliberate. A cancelled caller ends only its own wait, so
one caller's deadline cannot fail every caller that shares the fetch
(0053-MADR D4). What is missing is an owner who can end the work.

### What it cost a consumer

The issue reports a measurement made in gobble-cli, not repeated here. The
measurement used scratch copies of gobble-cli at `b67cbbe`, with a local
proxy that recorded each connection and forwarded nothing:

* 9 of `internal/cli`'s 264 tests tried 51 to 52 connections in one run;
* 12 of those connections were refreshes of a stored Grok login at
  `auth.x.ai`;
* a refresh's lock file appeared in a test's temporary home after the test
  ended;
* on Windows, `t.TempDir`'s removal then failed ("The directory is not
  empty"). The failure was recorded three times, and it failed gobble's
  `v0.3.0` release run once.

gobble will fix its own tests by passing no token source, which starts no
fetch (C11). The issue asks for what a consumer cannot do at all today: own
and end the SDK's background work.

### What the code does (abe3626)

Every site the issue cites holds at `abe3626`. The table lists them, then
three sites the issue does not name (rows 6 to 8).

| # | Site | Detached on | Bound | Counted by | Reached from a `Cache` |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | `CachedModel`'s listing fetch, `startListingFetch` (C6, C12) | `WithoutCancel` (C5) | 10 s | `listingFetching` (C2) | yes, its store |
| 2 | `List`'s shared fetch, `cachedOrFetched` (C8) | `WithoutCancel` (C7) | 10 s | `listingFetching` | yes, its store |
| 3 | the models.dev refresh, `startMetadataFetch` (C10) | `WithoutCancel` (C9) | `metadataFetchTimeout` | `metadataFetching` (C3) | yes, its store |
| 4 | a probe run, `ProbeGenerateHealthCached` (C18, C19) | `WithoutCancel` | `probeTimeout` per probe | nothing | yes: a `Cache`'s value holds its own probe store (C20) |
| 5 | an OAuth refresh's completion (C14) | `WithoutCancel` | per attempt | nothing | no: the session is the caller's |
| 6 | `CommandToken`'s early run (C21; 0068-MADR D4) | `WithoutCancel` | `Timeout` | nothing | no: the source is the caller's |
| 7 | `CachedModel` asks its source for a token in the foreground (C10a) | the caller's ctx | the caller's | — | no |
| 8 | `ListRaw` reads and writes no cache (C22) | — | — | — | not affected |

Three facts decide the design.

1. **Starting work and taking a count are not one step at two of the three
   catalog sites.**
   * `startListingFetch` sets its in-flight mark under `listingMu`, drops
     the lock (C5a), then calls `listingFetching.Go` (C6).
   * `cachedOrFetched` claims its key under the lock (C8a), then calls
     `Go` outside it (C8).
   * Only `startMetadataFetch` runs with its lock held (C10).

   Go's documentation says: "If the WaitGroup is empty, Go must happen
   before a WaitGroup.Wait." So a "closed" flag alone does not make a wait
   safe. The count must be taken under the same lock that `Close` takes to
   set the flag.
2. **A background listing fetch asks its source for a token** (C29). With
   an `OAuthSession`, that may start a refresh that runs on past the fetch's
   cancellation (C14). Closing the cache ends the fetch's wait for the
   token, not the refresh.
3. **The refresh releases its lock before it reports done** (C16, C15). The
   lock file is closed on unlock and stays on disk by design (C17). A wait
   for the refresh to finish therefore also ends the file handle that
   Windows' `TempDir` removal tripped on.

### Code claims

<!-- claims: self=abe3626 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `Cache` | `llmprovider/catalog/cache.go:20` | `type Cache struct {` |
| C2 | The listing group exists for tests | `llmprovider/catalog/cache.go:62` | `// listingFetching counts the background fetches, so tests can wait.` |
| C3 | The metadata group too | `llmprovider/catalog/cache.go:69` | `// metadataFetching counts the fetches in flight, so that tests can wait` |
| C4 | The process's store | `llmprovider/catalog/cache.go:87` | `var processStore = newStore()` |
| C5 | `CachedModel`'s fetch is detached | `llmprovider/catalog/listing_cache.go:230` | `fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), modelListingTimeout)` |
| C5a | after its lock is dropped | `llmprovider/catalog/listing_cache.go:229` | `c.listingMu.Unlock()` |
| C6 | and counted outside it | `llmprovider/catalog/listing_cache.go:231` | `c.listingFetching.Go(func() {` |
| C7 | `List`'s shared fetch is detached | `llmprovider/catalog/discovery.go:399` | `fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), modelListingTimeout)` |
| C8 | and counted outside the claim's lock | `llmprovider/catalog/discovery.go:400` | `cfg.store.listingFetching.Go(func() {` |
| C8a | The claim | `llmprovider/catalog/discovery.go:397` | `call, leader := cfg.store.claimListing(key)` |
| C9 | The metadata fetch is detached | `llmprovider/catalog/model_metadata.go:494` | `fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), metadataFetchTimeout)` |
| C10 | with its lock held | `llmprovider/catalog/model_metadata.go:487` | `// is none. c.metadataMu is held.` |
| C10a | `CachedModel` asks for a token in the foreground | `llmprovider/catalog/lookup.go:135` | `if token, err := src.Token(ctx); err == nil {` |
| C11 | No source starts no fetch | `llmprovider/catalog/lookup.go:134` | `if src != nil && !nilsource.Is(src) && hasListingLayer(id, cfg) {` |
| C12 | `CachedModel` starts the listing fetch | `llmprovider/catalog/lookup.go:140` | `cfg.store.startListingFetch(ctx, id, src, model, cfg, key)` |
| C13 | and the metadata fetch | `llmprovider/catalog/lookup.go:131` | `if m, ok := cachedMetadata(ctx, cfg); ok {` |
| C14 | The refresh completes detached | `llmprovider/auth/oauth_session.go:192` | `go s.complete(context.WithoutCancel(ctx), future, state, token, pending)` |
| C15 | It clears its mark before it reports done | `llmprovider/auth/oauth_session.go:257` | `s.inflight = nil` |
| C16 | after releasing the refresh lock | `llmprovider/auth/oauth_session.go:804` | `defer unlock()` |
| C17 | which closes the lock file | `llmprovider/auth/tokenstore_file.go:267` | `ignoreOAuthError(filelock.Unlock(f))` |
| C18 | A probe run is detached | `llmprovider/internal/transport/probe.go:195` | `detached := context.WithoutCancel(ctx)` |
| C19 | on a goroutine nothing counts | `llmprovider/internal/transport/probe.go:196` | `go func() {` |
| C20 | A cache's value holds its own probe store | `llmprovider/internal/cacheref/cacheref.go:31` | `return &Value{Catalog: catalog, Probes: transport.NewProbeStore(), Refusals: messages.NewRefusals(),` |
| C21 | `CommandToken`'s early run is detached | `llmprovider/command_token.go:174` | `value, err := runTokenCommand(context.WithoutCancel(ctx), argv, timeout)` |
| C22 | `ListRaw` touches no cache | `llmprovider/catalog/raw.go:66` | `no models.dev fetch; no listing cache read or write, so` |
| C23 | `List` refuses its options before the network | `llmprovider/catalog/discovery.go:236` | `cfg, err := listingConfig(id, opts)` |
| C24 | `ErrInvalidRequest` | `llmprovider/provider.go:70` | `ErrInvalidRequest = errors.New("llmprovider: invalid request")` |
| C25 | R25 | `docs/guides/api-standards.md:214` | `A new failure mode maps to an existing kind before it` |
| C26 | The module's tests wait on the listing group | `llmprovider/catalog/lookup_0033_test.go:97` | `func waitListingFetches() { processStore.listingFetching.Wait() }` |
| C27 | and on the metadata group | `llmprovider/catalog/model_metadata_test.go:21` | `func waitMetadataFetches() { processStore.metadataFetching.Wait() }` |
| C28 | R49 | `docs/guides/api-standards.md:320` | `- **R49. A caller can own the caches.**` |
| C29 | A background fetch asks for a token | `llmprovider/catalog/listing_cache.go:240` | `if _, _, err := fetchListingModel(fetchCtx, id, src, model, cfg); err != nil && cfg.logger != nil {` |
| C30 | opencode reads metadata at request time | `llmprovider/providers/opencode/opencode.go:589` | `meta, cached = catalog.CachedMetadataWith(ctx, p.gateway, p.listing...)` |
| C31 | and looks it up for a model its table lacks | `llmprovider/providers/opencode/opencode.go:870` | `return catalog.LookupMetadataWith(ctx, p.gateway, p.listing...)` |
| C32 | R55 | `docs/guides/api-standards.md:357` | `- **R55. A new export is justified.**` |

## Decision Drivers

* **Determinism.** Once `Close` returns nil, nothing the cache started is
  running, and nothing it will start can run.
* **The detachment stays.** No caller can cancel shared work. Only its
  owner can (0053-MADR D4, D6), and a rotation is still saved (0026-MADR F3).
* **Opt-in.** The process's cache keeps today's behaviour. No existing test
  changes.
* **The API standards.** A new failure maps to an existing kind first (R25,
  C25). A new export is justified by a call site (R55, C32). A blocking call
  takes a context (R40). Concurrency and error kinds are documented (R51,
  R52).
* **One mechanism.** The groups that already count the work are reused.

## Considered Options

* **A. `(*Cache).Close(ctx)`, which stops new starts and waits; and
  `(*OAuthSession).Wait(ctx)`.** This is the issue's proposal, with the
  count moved under each start's lock.
* **B. Export a `Wait` on the existing groups.**
* **C. A process-wide switch** (`catalog.SetBackground(false)`, or an
  environment variable).
* **D. A per-call `WithoutBackgroundFetch()` option.**
* **E. Undo the detachment:** run each fetch under its caller's context.

## Decision Outcome

Chosen option: "A", because it is the only option that both stops new work and
waits for running work. It keeps the detachment, it is opt-in by
construction, and it reuses the groups the cache already has.

### D1. `(*Cache).Close(ctx context.Context) error`

* **Closing happens once.**
  * The first `Close` sets a closed flag under each lock that a start takes:
    `listingMu`, `metadataMu`, and the probe store's `mu`.
  * It then cancels the cache's owner context.
* **Each start checks the flag and takes its count under its own lock.**
  * `startListingFetch` and `cachedOrFetched` call their group's `Go` before
    they release the lock (fact 1).
  * `startMetadataFetch` already holds its lock.
  * The probe store gains a group, used the same way.
  * So every count is taken before `Close` sets the flag, or the start sees
    the flag and starts nothing. Either way, `Wait` never races a `Go` on an
    empty group.
* **An owner context.** Each cache's store holds a context that `Close`
  cancels; the process's store holds one that is never cancelled. Detached
  work keeps today's context:
  `context.WithTimeout(context.WithoutCancel(ctx), …)`. It also stops when
  the owner context is cancelled: `context.AfterFunc(owner, cancel)`,
  stopped when the work ends. The work keeps the caller's values and today's
  timeouts. A fetch in flight ends at its next network wait.
* **The wait.** Each `Close` waits under its own `ctx` for the listing
  group, the metadata group and the probe group. It returns nil when all
  three are empty, or `ctx.Err()` when `ctx` ends first (Q5).
* **A zero `Cache`** makes its store on first use, as today. `Close` is a use,
  so a `Cache` closed before any other use is closed when it is later given
  to a call. It has nothing running, so `Close` returns nil at once.
* **The process's cache** has no `*Cache`, so nothing can close it.

### D2. A closed cache

* **The cached reads start nothing.** `CachedModel` and `CachedMetadataWith`
  given the cache answer from what it holds, as they do today. They start no
  fetch.
  * `CachedFacts` starts none today.
  * `CachedModel` still asks its source for the token that keys the listing
    (C10a). That is the caller's own call, in the foreground.
* **The waiting calls refuse.** `List`, `LookupModel` and
  `LookupMetadataWith` given the cache refuse before the network, as `List`
  refuses a bad option (C23). The error matches `llmprovider.ErrInvalidRequest`
  (C24) and says the cache is closed (Q1, Q4). A built-in provider's
  `ListModels` given the cache lists through `catalog.List`, so it returns
  that error too.
* **A probe through a closed cache's store** starts no run and returns what a
  run with no answer gives: the provider keeps its listing (0053-MADR D6).
* **A request is unaffected.**
  * A request reads cached facts (`CachedFacts`), which start nothing.
  * opencode's route reads the cached document (C30). For a model its route
    table lacks, it looks the document up (C31). After `Close` that lookup is
    refused, and the route falls back to the table's or the heuristic's, as
    it does today when the lookup fails.
* **What the cache holds stays readable** (Q2).
* **`ListRaw` is outside the cache** (C22) and unaffected.

### D3. `(*auth.OAuthSession).Wait(ctx context.Context) error`

* **What it waits for.** `Wait` returns nil once the session has no refresh
  or save in flight, or `ctx.Err()` when `ctx` ends first.
  * It does not cancel the refresh: a rotation must be saved (0026-MADR F3).
  * It looks again after each refresh ends, so a refresh that a waiter
    restarts is waited for too.
  * A `Token` call made after `Wait` returns may start another refresh.
* **The lock.** The refresh lock is released and its file closed before the
  refresh reports done (fact 3). So a test that calls `Wait` before its
  temporary directory is removed leaves no handle open.
* **Order.** `Cache.Close` ends a fetch's wait for its token but not the
  refresh behind it (fact 2). A caller that owns both calls `Close`, then
  `Wait`. Both docs say so.
* **`CommandToken`'s early run** (C21) is the same kind of work. It gets no
  `Wait` here: no caller needs one yet (R55). Its doc names the run as
  detached (Q3).

### D4. Tests, docs and the release

* **Tests**, one for each of the issue's acceptance points. Each is seen to
  fail on a planted break, on a scratch clone:
  * `Close` with a fetch in flight returns after the fetch ends. The fetch
    blocks until its context is done, and the test shows it ended.
  * With a fetch that ignores cancellation, `Close` returns `ctx.Err()` at
    the deadline, and a second `Close` returns nil once the fetch ends (Q5).
  * After `Close`:
    * `CachedModel` starts no fetch: no request reaches the test server, and
      the in-flight map stays empty;
    * `List`, `LookupModel` and `LookupMetadataWith` return an error
      matching `ErrInvalidRequest`;
    * a probe starts no run.
  * `Close` racing many `CachedModel`, `List` and probe calls passes `-race`
    and never panics on a group.
  * A zero `Cache` closes at once with nil.
  * `OAuthSession.Wait` returns after a detached refresh has saved, and at
    `ctx`'s deadline while it has not.
  * The process's cache is unchanged, and every existing test passes as it is.
* **Docs**:
  * `Cache`'s doc and `WithCache`'s doc;
  * a dated amendment to 0048-MADR D1, and R49's text after it (C28);
  * a dated note on 0053-MADR D4 and D6 that the detached work now has an
    owner;
  * `docs/architecture.md`;
  * the changelog;
  * an `Example` for each new method (R50).
* **The release** is a minor one (Q6).

### Consequences

* Good, because a test or a configuration can end with nothing of the SDK
  running against its credentials or files.
* Good, because the detachment and its reasons stand. Only the cache's owner
  can stop shared work.
* Good, because two latent orderings, `Go` taken after its lock, become
  correct by construction before anything waits on them.
* Neutral, because the process's cache cannot be closed. A caller who needs a
  lifetime makes a `Cache`, as R49 already allows.
* Neutral, because a `Close` whose context ended leaves a goroutine waiting
  on the groups until the work ends, at most the work's own timeout.
* Bad, because two exports are added, under the compatibility promise
  (R48).
* Bad, because a closed cache refuses `List`. A caller that still wants a
  listing must use a new cache.

### Confirmation

* The tests in D4 pass, each seen to fail on its planted break, the
  probe-store tests included.
* `go test -race` over `catalog`, `auth` and `internal/transport` passes.
* `apidiff` against `v1.27.0` lists only the two methods.
* The new `Wait` and a `t.TempDir` removal are run on a Windows host before
  the tag.
* `tools/claims.py` passes on this record.

## Pros and Cons of the Options

### A. `Close` and `Wait`

* Good, because they are deterministic: no start after the flag, and a wait
  with the count taken under the lock.
* Good, because they are opt-in, and the existing groups are reused.
* Bad, because a closed cache refuses the waiting calls, which callers must
  learn.

### B. Export a `Wait` on the groups

* Good, because it is the smallest change.
* Bad, because work can start after `Wait` returns.
* Bad, because a `Go` on an empty group that races a `Wait` breaks the
  `sync` contract. Without a closed state, it promises nothing.

### C. A process-wide switch

* Bad, because it reaches every user of the process's cache and cannot scope
  to one test.
* Bad, because an environment variable breaks R30.

### D. A per-call option

* Good, because it removes the work where it is given.
* Bad, because it cannot wait for work already started, and it covers neither
  `List`'s shared fetch nor the refresh.

### E. Undo the detachment

* Bad, because one cancelled caller would fail a fetch that others wait on.
  0053-MADR D4 fixed exactly that.
* Bad, because a cancelled refresh would lose a rotation (0026-MADR F3).

## Owner questions

*Answered by the owner on 2026-11-07, with the record's acceptance: each
question as recommended. The answers were chosen from three options each:*

* *Q1, refuse (offered also: a foreground fetch, unshared; or serve the cache
  and refuse only when a fetch is needed);*
* *Q2, keep the contents readable (also: clear them on `Close`; or keep them
  and refuse the reads too);*
* *Q3, `OAuthSession.Wait` only (also: a `CommandToken.Wait` as well; or
  `Wait` in a record of its own);*
* *Q4, wrap `ErrInvalidRequest` (also: an `ErrCacheClosed` that wraps it; or
  an `ErrCacheClosed` standing alone);*
* *Q5, each `Close` waits again (also: return the first call's result; or
  return at once);*
* *Q6, `v1.28.0` (also: `v1.27.1`; or no release yet).*

* **Q1. `List` on a closed cache.** Recommended: refuse, with an error
  matching `ErrInvalidRequest`. That is deterministic, and it matches how
  `List` refuses a bad option. The alternative is to fetch in the foreground
  under the caller's context, unshared, which keeps a closed cache usable but
  makes "closed" mean two things.
* **Q2. Contents after `Close`.** Recommended: keep them readable. Reading
  starts nothing and costs nothing. Clearing them would make `CachedFacts`
  answer differently after `Close`.
* **Q3. Scope.** Recommended:
  * `OAuthSession.Wait` belongs in this record. gobble's leftover lock comes
    from the refresh, which `Close` cannot end (fact 2).
  * `CommandToken` gets no `Wait` until a caller needs one (R55). Its doc
    names the early run.
* **Q4. The error.** Recommended: no new sentinel. The error wraps
  `ErrInvalidRequest` and says the cache is closed (R25). A
  `catalog.ErrCacheClosed` that wraps `ErrInvalidRequest`, as the issue
  proposes, can be added later without breaking anyone, once a caller needs to
  tell a closed cache from another refusal (R55). It would be the module's
  first sentinel outside `llmprovider`.
* **Q5. A second `Close`.** The issue says a second `Close` "returns the
  first's result". Recommended instead: closing happens once, and every call
  waits under its own context. Otherwise, a `Close` that hit its deadline
  could never be waited out. `http.Server.Shutdown` behaves the same way.
* **Q6. The release.** Recommended: `v1.28.0`, a minor release, since two
  methods are exported.

## More Information

* GitHub issue 12, "catalog: give a Cache a lifetime — Close(ctx) that stops
  new background fetches and waits for running ones (and OAuthSession.Wait
  for a detached refresh)". It was found in gobble-cli's
  `0046-MADR-own-background-listings-and-keep-tests-hermetic.md` (proposed
  2026-11-07), owner question Q2. Its measurements are the issue's, not
  repeated here.
* `go doc sync.WaitGroup.Go` (Go 1.27.3): "If the WaitGroup is empty, Go
  must happen before a WaitGroup.Wait."
* `context.AfterFunc` (Go 1.21 and later) runs a function once a context is
  done, and returns a `stop` function.
* Related records:
  * 0026-MADR-remediate-v1-2-debugging-pass-findings.md, F3, the detached
    refresh;
  * 0048-MADR-adopt-go-sdk-best-practices.md, D1, the `Cache`;
  * 0053-MADR-stream-retry-and-cache-defects.md, D4 and D6, the shared
    fetch and the probe run;
  * 0068-MADR-five-internal-fixes-from-the-0054-audit.md, D4, the
    `CommandToken` early run.
* PLAN:
  [0071-PLAN-give-a-cache-a-lifetime-and-wait-for-a-refresh.md](0071-PLAN-give-a-cache-a-lifetime-and-wait-for-a-refresh.md).
