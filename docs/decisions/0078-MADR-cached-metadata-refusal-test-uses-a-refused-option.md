---
status: accepted
date: 2026-11-12
decision-makers: repository owner
consulted: "ci.yml run 43295522137 on e5eea9d (job validate (ubuntu-24.04), step race detector), read with gh on 2026-11-12; reproductions on scratch clones of e5eea9d on this Mac, 2026-11-12; 0015-MADR-canonical-sdk-api-and-module-layout.md (D5, For); 0028-MADR-heuristics-and-performance-from-the-research-pass.md (D-A1, the test's origin); 0058-MADR-test-cache-isolation-for-reused-ports.md (F4, ports reused on Linux); the owner's decision of 2026-11-12, root-cause pair first"
informed: anyone who writes a test that counts the requests a catalog call sends
---

<!-- markdownlint-disable MD013 MD024 -->

# Test CachedMetadataWith's Refusal with an Option It Refuses, and Count Requests Only After Every Fetch Has Ended

## Context and Problem Statement

CI failed once on `main` at `e5eea9d`, a commit that changed only
`scripts/test_gates.py` and a record. In `ci.yml` run 43295522137, job
`validate (ubuntu-24.04)`, step "race detector (R44; 0020-MADR F27)", which
runs `go test -race -shuffle=on ./...` (C16):

```text
-test.shuffle 1794513551380402489
--- FAIL: TestCachedMetadataWith_Unavailable (0.00s)
    model_metadata_0028_test.go:68: requests = 1; want none yet
FAIL
FAIL	github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog	47.414s
```

`validate (macos-15)` and `validate (windows-2025)` passed; `release` was
skipped. The failure blocks the green `main` run that closes
[0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md)
Phase 4, and the start of
[0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)'s
code.

The test (C1 to C6), added with 0028-MADR D-A1 at `fd99677`, makes two
calls that it expects to report false with no fetch, then asserts that its
`httptest` server has seen no request (C3, C4). The first call turns
metadata off. The second passes, in the test's words, "an option for
another provider" (C2): `llmprovider.For(llmprovider.ProviderKilo,
WithKiloOrganization("org"))` (C1).

### Facts (observed 2026-11-12)

| # | Observation | Source |
| :--- | :--- | :--- |
| F1 | CI's run failed as quoted above, at line 68, with seed `1794513551380402489`, on Ubuntu only | `gh run view 43295522137 --log-failed`; `--json jobs` |
| F2 | The same seed passes on this Mac: the coordinator's three runs of the package, and a fourth here, `go test -count=1 -shuffle=1794513551380402489 -v ./llmprovider/catalog/`, exit 0. The test is timing-dependent, not order-dependent | the runs |
| F3 | `For` skips its options for every other id; only a provider-specific option given bare is refused (C7 to C10). `WithKiloOrganization` is scoped to `kilo` (C11). So `For(kilo, WithKiloOrganization("org"))` resolves for `opencode-zen` with no error and no Kilo value, and `CachedMetadataWith` goes on to the cache (C12), finds nothing, and starts a fetch (C13), in a goroutine (C14) | the code |
| F4 | **The second call sends the request.** A scratch test made each of the two calls alone, under `enableModelMetadata`, waited for every fetch with `waitMetadataFetches` (C15), and logged what reached the server. Metadata off: `configFor` error none, a fetch in flight on return `false`, 0 requests. `For(kilo, WithKiloOrganization)`: error none, a fetch in flight on return `true`, 1 request, `GET /` with the SDK's own User-Agent. Bare `WithKiloOrganization("org")`: error "option catalog.WithKiloOrganization is for provider \"kilo\", not \"opencode-zen\"", no fetch, 0 requests | scratch clone of `e5eea9d`, `go test -race -run TestRepro0078 -v` |
| F5 | **CI's text, reproduced at will.** With `time.Sleep(100 * time.Millisecond)` planted before the check (C3), the unchanged test fails at once: `requests = 1; want none yet`, exit 1. With `waitMetadataFetches()` planted there instead, it fails the same way, exit 1. Unpatched, 300 runs (`-race -count=300`) pass: the check reads the counter before the fetch's request arrives, except when the scheduler holds the test's goroutine long enough **[that CI's runner did so is inferred from F1, F2 and F5; nothing in the log shows it]** | scratch clones of `e5eea9d` |
| F6 | **The correction holds.** With the second call given bare `WithKiloOrganization("org")` and `waitMetadataFetches()` before the check (D1), the test passes 300 runs under `-race`, and the package passes with CI's seed and with `-shuffle=on`, each exit 0. Three plants each fail it with `requests = 1; want none yet`, `tools/plant.py` "all 3 plant(s) caught": the For option restored; the first call's `WithoutModelMetadata()` replaced; `cachedMetadata`'s `metadataOff` test turned off | scratch clone of `e5eea9d` with the edit |
| F7 | **No request crossed from an earlier test.** A probe on a scratch clone wrapped the client of every background start (`startModelMetadata`, `startMetadataFetch`, `startListingFetch`, `claimListing`) and logged any request sent after the top-level test that started it had ended. On a planted stray, a delayed fetch on a test's own cache, it logged one (`STRAY kind=startMetadataFetch origin=TestZZProbeA now=TestZZProbeB`). Over four full runs of the package under `-race`, CI's seed and three `-shuffle=on`, it logged none | scratch clone of `e5eea9d` |
| F8 | The second call's `For` has been in the test since it was written (`git log -S`, one commit, `fd99677`); no other test in the module passes a `For` of another id where it expects a refusal (`git grep`) | `git log`, `git grep` |

The coordinator's hypothesis, that a request from an earlier test reached
this test's server on a port `httptest` reused (0058-MADR F4 measured such
reuse on Linux), does not explain F4 to F6: the request comes from the
test's own second call, and F7 found no request crossing tests. The two
mechanisms that hypothesis named are real, and are observed under D3.

## Decision Drivers

* CI on `main` must pass in full before 0073-PLAN Phase 4 closes and 0074's
  code starts.
* The fix must remove the cause, not the symptom: the test's second case
  should exercise a refusal, and its "none yet" check should hold for that
  reason.
* A check that a call started no fetch is worth something only if it fails
  whenever a call does start one, not when the scheduler happens to let the
  request through first.
* `For`'s contract, one option list shared by several providers, is
  decided (0015-MADR D5) and relied on by callers (C7, C8).
* No assertion is loosened (the owner's rule on deviations).

## Considered Options

* **A. Correct the test:** give the second call the bare
  `WithKiloOrganization("org")`, which `configFor` refuses, and call
  `waitMetadataFetches()` before the "none yet" check, so that any fetch
  either call started is counted every time.
* **B. Correct the option only,** with no wait before the check.
* **C. Refuse a `For` of another id** in `CachedMetadataWith` (or in
  `ResolveOptions`), so the test as written is right.
* **D. Bind every background request to its test:** count
  `startModelMetadata`'s goroutine, and close each `testCache` at the end
  of its test, as the coordinator's hypothesis proposed.
* **E. Make the test tolerant:** accept up to one request before the third
  call, or drop the "none yet" check.

## Decision Outcome

Chosen option: "A. Correct the test", because the request comes from the
test's own second call (F4), whose option the library skips by its
documented contract (F3); the bare option is the refusal the test names
(C6) and the library makes (C10); and the wait turns a check that F5 shows
catches a fetch only by chance into one that catches it every time (F6).
No exported name, no non-test file and no other assertion changes.

The owner accepted this record on 2026-11-12, in the coordinator session, and
approved its PLAN the same day.

### D1. The edit

In `llmprovider/catalog/model_metadata_0028_test.go`:

* the second call's last argument becomes `WithKiloOrganization("org")`
  (C1), and its message "an option scoped to another provider: ok = true"
  (C2);
* `waitMetadataFetches()` is called before the check at C3, with a comment
  naming this record.

The rest of the test, the third call, its wait, and the count of one
request remembered (C5), is unchanged.

### D2. What proves it

* **Red before:** the test at `e5eea9d`, with only the wait of D1 added,
  fails `requests = 1; want none yet` on every run (F5).
* **Green after:** the edited test passes under `-race -count=300`, and the
  package passes `go test -race -shuffle=on`, with CI's seed and with a new
  one (F6).
* **Plants:** `tools/plant.py` on a scratch clone, never in the tree, with
  the three plants of F6, each of which must fail.

### D3. Observed, not decided: background work a test does not wait for

Two mechanisms of the coordinator's hypothesis exist in the code, though
neither caused this failure (F7):

* **O1.** `startModelMetadata` starts a goroutine that no WaitGroup counts
  until it reaches `startMetadataFetch` (C17, C18). A listing that fails, or
  lists nothing, builds the static menu from `peekMetadata` (C19) and never
  waits for that goroutine. Under a test that turns metadata on, its fetch
  can start after `waitMetadataFetches` has returned.
* **O2.** `testCache(t)` gives each test its own `Cache`, whose background
  fetches and refreshes nothing waits for: its cleanup only forgets it
  (C20), and `waitMetadataFetches` waits on the process's store alone.

Either can let a request outlive its test, and on Linux reach a later
test's server on a reused port. Neither changes here. A record of its own
decides them, if a failure or `tools/cachecensus.py` points at one.

### Consequences

* Good, because the test's second case tests what its comment says (C6):
  an option `CachedMetadataWith` refuses, refused.
* Good, because a future change that makes either case fetch fails this
  test on every run, not once in many.
* Good, because CI on `main` can pass again, which 0073-PLAN Phase 4 and
  0074-PLAN Phase 1 wait for.
* Neutral, because the test no longer covers a `For` of another id, which
  no assertion in it ever checked: the call reported false only because
  nothing was cached.
* Neutral, because O1 and O2 stay as they are (D3).
* Bad, because the edit is a test change made for a flake seen once, and
  F5's account of CI's timing is inferred, not observed.

### Confirmation

* D2's red and green runs, each exit status captured; the red run is a
  scratch clone of `e5eea9d`, never the tree.
* `tools/plant.py` reports "all 3 plant(s) caught".
* `python3 -B tools/gate.py` passes on the change.
* `ci.yml` on `main`, at the commit that holds D1, passes all three
  `validate` jobs.
* `tools/claims.py` passes on this record and on its PLAN.

## Pros and Cons of the Options

### A. Correct the test

* Good, because it removes the cause F4 found, in the one place it lives
  (F8).
* Good, because it strengthens the check: F6's three plants each fail it.
* Good, because it changes one test file and no code.
* Bad, because the wait covers the process's store alone (C15): a call
  given a cache of its own would start a fetch the check does not wait
  for. Neither of the test's calls is given one.

### B. Correct the option only

* Good, because it is the smallest edit that removes the cause.
* Bad, because the check stays a race: F5 shows the unchanged check passed
  300 runs with the fetch in flight, so a later regression would show only
  as a rare flake, as this one did.

### C. Refuse a `For` of another id

* Good, because the test would pass as written.
* Bad, because it breaks `For`'s contract, an option list shared by
  several providers (0015-MADR D5; C7, C8), which callers rely on, for a
  test's mistake.
* Bad, because it is an exported behaviour change, under R48, for no
  caller's need.

### D. Bind every background request to its test

* Good, because O1 and O2 are real, and a later failure may need it.
* Bad, because it does not touch this failure: the request came from the
  test's own call (F4), and no request crossed tests in four runs (F7).
* Bad, because counting O1's goroutine changes non-test code, and closing
  each test's cache in `testCache` changes the end of every test that uses
  it (50 calls in the package's tests), with no failure to show for
  either.

### E. Make the test tolerant

* Good, because it is one line.
* Bad, because it loosens an assertion, which the owner's rules forbid as a
  workaround, and leaves the second case testing nothing.

## More Information

* The PLAN:
  [0078-PLAN-cached-metadata-refusal-test-uses-a-refused-option.md](0078-PLAN-cached-metadata-refusal-test-uses-a-refused-option.md).
* The test's origin: 0028-MADR D-A1, at `fd99677`.
* Port reuse on Linux, and the test caches it led to:
  [0058-MADR-test-cache-isolation-for-reused-ports.md](0058-MADR-test-cache-isolation-for-reused-ports.md).
* The scratch scripts, outputs and plant spec are kept in the session's
  scratch directory, not committed. The probe of F7 marked each top-level
  test of the package at its start and, last of its cleanups, at its end.

### Not verified

* Why CI's runner let the request through first (F5): the log has no
  timing for it.
* F7 on Linux: the probe ran on this Mac only. It reads when a request is
  sent, not which port it reaches, so it does not depend on reuse.
* O1 and O2 under load on Linux.

### Code claims

<!-- claims: self=e5eea9d -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The second call's option | `llmprovider/catalog/model_metadata_0028_test.go:64` | `llmprovider.For(llmprovider.ProviderKilo, WithKiloOrganization("org"))); ok {` |
| C2 | and its message | `llmprovider/catalog/model_metadata_0028_test.go:65` | `t.Error("an option for another provider: ok = true")` |
| C3 | The check CI failed | `llmprovider/catalog/model_metadata_0028_test.go:67` | `if n := hits.Load(); n != 0 {` |
| C4 | at line 68 | `llmprovider/catalog/model_metadata_0028_test.go:68` | `t.Fatalf("requests = %d; want none yet", n)` |
| C5 | The count after the third call | `llmprovider/catalog/model_metadata_0028_test.go:75` | `if n := hits.Load(); n != 1 {` |
| C6 | What the test says it checks | `llmprovider/catalog/model_metadata_0028_test.go:54` | `// and an option CachedMetadataWith refuses all report false without a fetch.` |
| C7 | `For` skips its options for another id | `llmprovider/settings.go:39` | `wins wherever it sits in the list. For any other id they are skipped. A` |
| C8 | and only a bare option is refused | `llmprovider/settings.go:41` | `shared by several providers; given bare, it is still refused by another` |
| C9 | The skip | `llmprovider/settings.go:205` | `if opt.target == id {` |
| C10 | The refusal | `llmprovider/settings.go:288` | `case opt.scoped && !slices.Contains(opt.providers, id):` |
| C11 | `WithKiloOrganization` is scoped to kilo | `llmprovider/catalog/config.go:159` | `return llmprovider.ScopedOption(llmprovider.ProviderKilo, "catalog.WithKiloOrganization", kiloOrganizationOption(org))` |
| C12 | `CachedMetadataWith` reports false only on a refused option | `llmprovider/catalog/model_metadata.go:309` | `cfg, err := configFor(id, opts)` |
| C13 | otherwise, with nothing cached, it starts a fetch | `llmprovider/catalog/model_metadata.go:330` | `c.startMetadataFetch(ctx, url, cfg)` |
| C14 | which runs in a goroutine | `llmprovider/catalog/model_metadata.go:509` | `c.metadataFetching.Go(func() {` |
| C15 | The test's wait | `llmprovider/catalog/model_metadata_test.go:21` | `func waitMetadataFetches() { processStore.metadataFetching.Wait() }` |
| C16 | CI's step | `.github/workflows/ci.yml:54` | `go test -race -shuffle=on ./...` |
| C17 | O1: an uncounted goroutine | `llmprovider/catalog/model_metadata.go:680` | `go func() {` |
| C18 | started beside every listing but Kilo's and Ollama's | `llmprovider/catalog/discovery.go:341` | `meta = startModelMetadata(ctx, cfg)` |
| C19 | The static path reads the cache, not that goroutine | `llmprovider/catalog/discovery.go:538` | `if m, ok := peekMetadata(cfg); ok {` |
| C20 | O2: a test's cache is only forgotten | `llmprovider/catalog/model_metadata_test.go:60` | `t.Cleanup(func() { caches.Delete(t) })` |
