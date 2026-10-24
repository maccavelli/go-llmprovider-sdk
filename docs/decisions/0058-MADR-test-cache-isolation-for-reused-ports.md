---
status: accepted
date: 2026-10-23
decision-makers: repository owner
consulted: "CI run 39890209652 (the v1.22.0 tag, attempts 1 and 2); 0057-PLAN-wizard-device-code-and-paste-prompt.md (its deviation of 2026-10-22); 0048-MADR-adopt-go-sdk-best-practices.md (D1, catalog.WithCache); 0049-MADR-v1-16-0-follow-ups.md (D3); probes on darwin/arm64 and linux/amd64 (method below); a census of the suite on a scratch clone (P7); the owner's answers to Q1 and Q2, 2026-10-22; this repository at b66ecdf"
informed: anyone who writes a test that serves a provider, a listing or a metadata document from an httptest server
---

<!-- markdownlint-disable MD013 -->

# Give Every Test That Points the Catalog's Cache at an httptest Server a Cache of Its Own

## Context and Problem Statement

CI's run for the `v1.22.0` tag failed on Linux in a test that the release
did not touch:

```text
--- FAIL: TestOpencode_RoutesFromMetadata/openai (0.00s)
    metadata_route_test.go:118: path = "/chat/completions", want "/responses"
FAIL	github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode
```

The run's shuffle seed was `1792722845670550151`. macOS and Windows passed, and
a re-run of the same commit passed (attempt 2). The owner accepted the re-run
for the release, and
[0057-PLAN-wizard-device-code-and-paste-prompt.md](0057-PLAN-wizard-device-code-and-paste-prompt.md)
left "the test's isolation" to a record of its own (C18). This is that record.
No GitHub issue is open: issues 6, 7 and 8 were closed by that PLAN.

The PLAN's deviation named a suspect: the metadata cache belongs to the
process, keyed by URL. An `httptest` server given a port an earlier server had
would find that server's document cached. This record shows the suspect to
be the mechanism, and decides how the tests stop depending on the OS never
reusing a port.

### Facts (observed 2026-10-22 at `b66ecdf`)

* **F1. Three of the catalog's caches are the process's, keyed by a
  server's URL.** Without `catalog.WithCache`, every listing and provider
  shares `processStore` (C3). Its entries are keyed by URL:
  * the metadata document, by its URL (C2), fresh for 10 minutes (C1);
  * a listing, by provider, base URL, organization and credential
    fingerprint (C6);
  * a listing's probes, by provider, base URL and credential fingerprint
    (C7).

  The learned tool-choice refusals are keyed by provider and model, not by
  URL (C16). They are out of this record's scope **[read]**. *(Amended
  2026-10-23: there is a fourth, Ollama's show replies (C20). See the
  amendment at the end.)*
* **F2. A test's server is identified only by its port.** `httptest`
  servers listen on `127.0.0.1:0`, and the route test caches the document
  at `srv.URL + "/api.json"` (C8, C9). Two servers that get the same port
  in one process share one cache entry **[read]**.
* **F3. A stale document gives exactly CI's failure.** The provider reads
  the cached document (C10). A document at that URL that does not give
  `brand-new-model` the `@ai-sdk/openai` package falls through to the
  model's own route (C11), which for a model the table does not know is
  the heuristic's chat completions. Every other document the opencode tests
  serve does so **[read]**.
* **F4. Linux reuses a closed server's port within one process. macOS did
  not.** A scratch program (P4 below) started `httptest` servers one after
  another and closed each one **[probed]**:

  | Host | Each server | Servers | Ports reused | First reuse |
  | :--- | :--- | ---: | ---: | :--- |
  | linux/amd64 | no request | 2,000 | 258 | server 90, after 9.6 ms |
  | linux/amd64 | one GET | 300 | 0 | none |
  | linux/amd64 | one GET | 2,000 | 26 | server 399, after 367 ms |
  | darwin/arm64 | no request | 2,000 | 0 | none |
  | darwin/arm64 | one GET | 2,000 | 0 | none |

  That fits a failure seen on ubuntu only.
* **F5. The mechanism reproduces deterministically.** A scratch test (P5)
  served one document from server A, cached it, closed A, and listened on
  A's port with server B, which served another document. With the process's
  cache it failed with CI's text: `path = "/chat/completions", want
  "/responses"`. With a `catalog.WithCache(new(catalog.Cache))` per server,
  given to the warm-up and to the provider, it passed **[probed]**.
* **F6. The failure recurs on Linux, in more than one test, but not by
  its seed** **[probed]**:
  * the opencode package, shuffled with a new seed each run, failed
    4 times in 300 runs on linux/amd64 (P6), in three tests.
    Each failure is a request shaped by a document its test did not
    serve:
    * `TestOpencode_UnknownModelWaitsForMetadata`, twice: `paths =
      [/chat/completions]; want the document's /messages` (seeds
      `1792725440193340307`, `1792725764032791980`);
    * `TestOpencode_RoutesFromMetadata/over_table`: `path = "/messages",
      want "/chat/completions"` (seed `1792725008519607222`);
    * `TestOpencodeChat_ReplaysInterleavedReasoning`, where the document
      picks the reasoning field to replay, not the route: `assistant
      reasoning_content = []interface {}{interface {}(nil), interface
      {}(nil)}, want ["Call the tool." ""]` (seed
      `1792725895849796753`);
  * a second measurement, by the session that ran 0057, failed 3 times
    in 150 runs (P8). One was CI's own failure,
    `TestOpencode_RoutesFromMetadata/openai`, one the `over_table`
    failure P6 also saw, and one a fourth test:
    `TestOpencode_ColdCacheUsesTableRoute`: `paths = [/messages
    /messages]; want the table's route, then the document's: [/messages
    /chat/completions]` (seed `1792725343227899698`);
  * nine other exposed packages, 100 shuffled runs each, failed none
    (P9). `catalog`, `wizard` and the root `providers` package are among
    them. That bounds each one's rate below about 3 % (95 %), not at
    zero. A package can be exposed and not show it: its documents may
    happen to agree, or its assertions may not depend on the entry.
    Failures have been seen only in the opencode package;
  * the opencode package with CI's seed, 40 times on linux/amd64 (P2):
    no failure;
  * the route test alone, `-count=2000`, on darwin/arm64 and linux/amd64
    (P3): no failure.

  The failure needs a reused port *and* a fresh entry under it. That
  depends on what the kernel does at that moment, not on the test order,
  so a seed does not bring it back. Why the route test alone never
  failed is not established.
* **F7. Package `catalog` can clear the cache, and the provider packages
  cannot.** Catalog's tests can clear `processStore`, and some do (C12,
  F11). A provider package
  sees only the exported API, so its tests depend on every server URL
  being unique **[read]**.
* **F8. The isolation the decision needs already exists, and some tests
  use it.** `catalog.Cache` was added so "a test or a process with more
  than one configuration can keep them apart" (C4), through `WithCache`
  (C5). The opencode refusal tests give each provider its own (C15)
  **[read]**.
* **F9. How many tests are exposed: 207, by a census of the suite**
  **[probed]**. A scratch clone was instrumented to print each write of
  a loopback URL into the process's metadata, listing or probe store,
  and its whole suite was run serially (P7). It passed. 207 top-level
  tests wrote such an entry:

  | Package | Tests |
  | :--- | ---: |
  | `llmprovider/catalog` | 110 |
  | `wizard` | 16 |
  | `llmprovider/providers/opencode` | 16 |
  | `llmprovider/providers/openai` | 11 |
  | `llmprovider/providers/ollama` | 10 |
  | `llmprovider/providers/claude` | 9 |
  | `llmprovider/providers/grok` | 9 |
  | `llmprovider/providers/gemini` | 8 |
  | `llmprovider/providers/together` | 6 |
  | `llmprovider/providers/huggingface` | 5 |
  | `llmprovider/providers/kilo` | 5 |
  | `llmprovider/providers` | 1 |
  | `llmprovider/internal/cmd/catalogs` | 1 |

  Outside `catalog` that is 97 tests in 12 packages. Besides the
  listing and metadata tests, they include every provider package's
  wire goldens, its token-header test and its caller-identity test, and
  the five `TestProbeCache_ProviderProbesIntoItsCache_*` tests. Each of
  those five has a step that uses the process's cache on purpose, to
  show that a nil `Cache` is the process's. The tests that already
  give their own cache, such as opencode's refusal tests (C15), do not
  appear. So the census tells the two apart. The four tests that failed
  in P6 and P8 (F6) are all among its 16 opencode tests.

  An earlier search of the `_test.go` files outside `catalog`, by text
  **[read]**, found fewer:
  * 8 files call `catalog.List`, `ListRaw`, `LookupMetadata*`,
    `CachedMetadataWith` or `CachedFacts` and also start an `httptest`
    server. None passes `WithCache`. They are
    `providers/facts_0041_test.go`, `providers/hygiene_test.go`,
    `providers/grok/efforts_0041_test.go`,
    `providers/huggingface/listing_test.go`,
    `providers/together/listing_test.go`, and three in
    `providers/opencode/`: `listing_test.go`,
    `metadata_cold_0028_test.go` and `metadata_route_test.go`;
  * 15 files pass `WithModelMetadataURL`. None passes `WithCache`;
  * in `providers/opencode`, metadata is turned on at 10 call sites in
    5 files: 6 calls of `enableMetadata` (C17) and 4 of `routeServer`,
    which calls it.

  A test that builds a provider and lets the provider list or probe goes
  through the same caches without naming the catalog. The text search
  missed those, and the census counts them.
* **F10. No test in the opencode package runs in parallel** (C13). A
  process-wide reset between tests is therefore possible there. Whether
  it is possible in every package is not checked **[read]**.
* **F11. Package `catalog` knows the hazard, and does not always clear.**
  Its helper says "httptest reuses ports, so a cache keyed by URL must
  not outlive a test" (C19). Of its 28 test files that start a server,
  13 call none of its clearing helpers and pass no `Cache`. The census
  counts 110 catalog tests that write a loopback URL into the process's
  store (F9). Which of them clear before and after is not established
  **[read]** **[probed]**.

## Decision Drivers

* A test's outcome must not depend on which port the kernel assigns.
  CI shuffles test order to catch order dependence (C14). This failure is
  the same hazard, by another route.
* No change to the library's behaviour. A real URL that serves a new
  document is the TTL's job, and is correct as it is.
* No exported name for tests only (R55).
* Fix the hazard once, where tests get their options, rather than in each
  test.
* The owner's rules rule out a retry, a skip or a loosened assertion.

## Considered Options

* **T1.** Each test that serves the catalog from an `httptest` server
  gives all of its calls one new `catalog.Cache`, through the package's
  shared test helpers.
* **T2.** Make each server's URL unique by a path segment, for example
  `srv.URL + "/" + t.Name() + "/api.json"`.
* **T3.** Clear the process's cache after each test, through an internal
  package that `catalog` registers its reset with.
* **T4.** Leave the tests as they are, and re-run CI when one fails.

## Decision Outcome

Chosen option: **T1**, because it removes the dependency on the port for
all three URL-keyed caches. It uses an option that exists for this purpose
(F8), changes no library code, and is how the refusal tests already isolate
themselves. The owner answered Q1 and Q2 on 2026-10-22 and Q3 on
2026-10-23, each with the recommended answer. The PLAN's Phase 0 marked
this record `accepted`.

### D1. The rule

A test outside package `catalog` that makes the catalog cache anything
under an `httptest` server's URL gives every call of that test one new
`catalog.Cache`, through `catalog.WithCache`: a metadata document, a listing,
or a listing's probes. "Every call" means the test's own catalog calls (a
warm-up such as C8) and the providers it builds, so that they share that
one cache and nothing else. Package `catalog`'s own tests are Q3's
(F11).

A step that tests the process's cache on purpose, such as the
`WithCache(nil)` step of the probe-cache tests (F9), is kept. It gives a
credential unique to the run, so its entry's key (C7) can meet no other
test's.

### D2. Where

The option goes in each package's shared test helpers, so that a test gets
the cache by default and a new test cannot forget it. Each package's
`build` adds the test's own cache, one per test, and a test's direct
catalog calls, such as the route test's warm-up (C8), pass the same one.
The PLAN lists every helper and file, from F9's census.

### D3. A regression test

The reproduction F5 describes is kept as a test in `providers/opencode`:
server B listens on closed server A's port, serves another document, and
the request must route by B's. Only the half with its own cache is kept. The
half with the process's cache would leave the stale entry it proves, which is
the hazard itself. Removing the cache from the helper is the planted break:
it must fail with CI's text.

### D4. Nothing else changes

No library code changes and no exported name is added. No release is
needed for this change alone. CI's commands are unchanged.

### D5. The census, as a maintainer tool

P7's instrumentation becomes `tools/cachecensus.py`. It runs on a
scratch clone, never in the tree, and CI does not run it (Q2 (a)). It is
how each phase shows that its packages no longer write a test server's
URL into the process's caches, and how a later reviewer can check the
same. It fails, rather than reporting nothing, when the code it
instruments has moved.

### Consequences

* Good, because the route test, and every test D1 covers, no longer
  depends on which port the kernel assigns, for the metadata, listings and
  probes alike.
* Good, because each test starts from an empty cache, so a test cannot
  pass only because an earlier one warmed the cache.
* Neutral, because each covered test now fetches its own document and
  listing, as it would on a cold cache. The servers are local, so the
  added time is small; the PLAN measures it.
* Bad, because the rule is a convention (Q2 (a)). A test that calls
  `catalog.List` directly, outside the helpers, can still miss it. D5's
  census finds such a test when someone runs it, not on every push.
* Bad, because a test that relies on another test warming the process's
  cache will fail once it has its own. Such a test depends on order
  already. The PLAN stops on one and puts it to the owner, rather than
  changing its assertion.

### Confirmation

* D3's test passes, and fails on its planted break with
  `path = "/chat/completions", want "/responses"`.
* The route test passes `-count=2000` on linux/amd64, as it does now. This
  shows nothing broke, not that the flake is gone (F6). The deterministic
  proof is D3.
* D5's census reports no test in scope, apart from the deliberate steps
  of D1.
* The opencode package, run 300 times shuffled on linux/amd64 as P6 ran
  it, does not fail.
* `python3 -B tools/gate.py` passes, and CI passes on the three platforms.
* The PLAN's record lists each test of F9's census, and each test found
  beyond it, with what changed or why nothing did.

## Pros and Cons of the Options

### T1. A Cache per test

* Good, because it covers all three URL-keyed caches with one option.
* Good, because it is the documented purpose of `catalog.Cache` (C4), and
  already in use (C15).
* Good, because it works whether or not a package's tests run in parallel.
* Bad, because it touches many test files, and holds only by convention
  (Q2).

### T2. A unique path per server

* Good, because each change is one line.
* Bad, because it covers the metadata only. Listings and probes are keyed
  by base URL (C6, C7), and a provider's base URL has no free path to make
  unique.
* Bad, because a test whose metadata URL is not unique is still exposed,
  and the cause is hidden behind the path.

### T3. Clear the process's cache after each test

* Good, because each test file changes by one cleanup.
* Bad, because it adds production code, a reset hook, used only by tests.
* Bad, because a reset would race any test running in parallel, and only
  the opencode package is known to have none (F10).
* Bad, because a test could still pass only because the cache was warm
  within it.

### T4. Leave it and re-run

* Good, because it costs nothing now.
* Bad, because a release's CI can fail again on a test it did not touch,
  and someone has to judge whether a re-run is enough. That happened for
  `v1.22.0`.
* Bad, because a re-run is a workaround, which the owner's rules exclude as
  a fix.

## Owner questions

* **Q1 (D1). Scope.**
  * **(a)** Every test outside `catalog` that reaches any of the three
    URL-keyed caches through an `httptest` server: F9's files, and the
    tests that reach the caches by listing or probing through a provider
    (recommended).
  * **(b)** The metadata tests in `providers/opencode` only: the 10 call
    sites of F9's last item.

  **Answered 2026-10-22: (a).**
* **Q2 (D2). How the rule is held.**
  * **(a)** By the helpers and D3's test, with no new gate (recommended). A
    gate would have to work out from the text of a test file whether its
    calls reach a cache, and a file that mentions `WithCache` once would pass
    it.
  * **(b)** Add a gate to `scripts/`, run by CI and by `gate-selftest`: it
    refuses a `_test.go` file outside `llmprovider/catalog` that starts an
    `httptest` server and calls the catalog, or passes
    `WithModelMetadataURL`, without `WithCache`. It needs its own planted
    breaks.

  **Answered 2026-10-22: (a).**
* **Q3 (D1). Package `catalog`'s own tests (F11).**
  * **(a)** In this record's scope. Each of the census's 110 tests takes
    its own cache, unless it reads `processStore` directly. A test that
    does clears every map it writes, before it starts and when it ends
    (recommended). It is the largest part of the work. No catalog test
    has been seen to fail (P9), so this is for the hazard F11 names, not
    for an observed failure.
  * **(b)** Out of scope, for a later record. Catalog's tests stay as
    they are, and are exposed as F11 describes.

  **Answered 2026-10-23: (a).**

## More Information

### Probes

None of the probes ran in the working tree.

* **P1.** CI's log, `gh run view 39890209652 --attempt 1 --log-failed`.
  It is quoted above.
* **P2.** On linux/amd64 (kernel 7.0, Go 1.27.3), in a copy of `3b63351`
  extracted by `git archive` (`b66ecdf` differs from it in `docs/`
  only):
  `go test -count=1 -shuffle=1792722845670550151 ./llmprovider/providers/opencode/`,
  40 times. None failed.
* **P3.** `go test -run 'TestOpencode_RoutesFromMetadata$' -count=2000`
  on both hosts. Both passed, in 9.7 s on darwin and 52.4 s on linux.
* **P4.** A scratch program in its own module, outside the tree.
  It ran `httptest.NewServer` in a loop and recorded each port. In mode
  `get` each server answered one `http.Get` before `Close`; in mode `idle`
  it answered none. Its results are F4's table.
* **P5.** A scratch clone of `b66ecdf`, with one new test file in
  `llmprovider/providers/opencode`. Server B was started by
  `httptest.NewUnstartedServer` with a listener from
  `net.Listen("tcp", <A's address>)`. Its two subtests: `process cache`
  FAIL with CI's text; `own Cache` PASS.
* **P6.** On linux/amd64, a copy of `3b63351`: the package's test binary,
  built once, run 300 times with `-test.shuffle=on` and `-test.count=1`.
  4 failed, as F6 quotes.
* **P7.** The census. A scratch clone of `b66ecdf` was instrumented
  at four points: `startMetadataFetch` once its fetch is registered,
  `storeListing`, `storeFetched`, and
  `transport.ProbeGenerateHealthCached`. Each printed `CENSUS <kind>
  <key>` on stderr when the store was the process's and the key held
  `127.0.0.1`. `go test -count=1 -v -p 1 -parallel 1 ./...` passed, and
  printed 1,160 such lines. A script attributed each line to the
  package and top-level test that the log's order put it in.
* **P8.** By the session that ran 0057-PLAN, on the owner's ask, and
  checked here against its logs: a depth-1 clone of `3b63351` on the same
  linux/amd64 host. The opencode test binary was built once and run 150
  times with `-test.shuffle=on` and `-test.count=1`: 3 failed, each in
  another test, as F6 quotes. Its loop ran on that host at the same
  time as P6's, so the two measured under each other's load. Its port
  probe, 500 servers with no request and none open at once, gave 98
  distinct ports on Linux and 500 on darwin, as F4 found. Its
  reproduction matched P5 on both hosts.
* **P9.** By the same session, in the same clone, checked here against its
  log: each package's test binary was built once and run 100 times with
  `-test.shuffle=on` and `-test.count=1`. None failed in
  `llmprovider/catalog`, `llmprovider/providers`, `providers/together`,
  `providers/claude`, `providers/grok`, `providers/huggingface`, `wizard`,
  `llmprovider` or `llmprovider/internal/cmd/catalogs`. The gemini, kilo,
  ollama and openai packages were not measured.

### Related

* [0057-PLAN-wizard-device-code-and-paste-prompt.md](0057-PLAN-wizard-device-code-and-paste-prompt.md):
  the deviation that found the failure and left it to this record.
* [0048-MADR-adopt-go-sdk-best-practices.md](0048-MADR-adopt-go-sdk-best-practices.md)
  (D1) and [0049-MADR-v1-16-0-follow-ups.md](0049-MADR-v1-16-0-follow-ups.md)
  (D3): `catalog.Cache`, and the refusals it holds.

### Code claims

<!-- claims: self=b66ecdf -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | A document is fresh for 10 minutes | `llmprovider/catalog/model_metadata.go:30` | `modelMetadataTTL        = 10 * time.Minute` |
| C2 | The document is looked up by URL | `llmprovider/catalog/model_metadata.go:459` | `e := c.metadata[url]` |
| C3 | The process's store | `llmprovider/catalog/cache.go:85` | `var processStore = newStore()` |
| C4 | Cache exists to keep configurations apart | `llmprovider/catalog/cache.go:16` | `process with more than one configuration can keep them apart.` |
| C5 | The option | `llmprovider/catalog/cache.go:38` | `func WithCache(c *Cache) llmprovider.Option {` |
| C6 | A listing is keyed by base URL | `llmprovider/catalog/listing_cache.go:25` | `baseURL  string` |
| C7 | Probes are keyed by base URL | `llmprovider/internal/transport/probe.go:102` | `Provider, BaseURL, Credential string` |
| C8 | The route test warms the process's cache | `llmprovider/providers/opencode/metadata_route_test.go:113` | `if _, err := catalog.LookupMetadataWith(context.Background(), tc.gateway,` |
| C9 | at the server's own URL | `llmprovider/providers/opencode/metadata_route_test.go:76` | `opts = append([]llmprovider.Option{llmprovider.WithModelMetadataURL(srv.URL + "/api.json")}, opts...)` |
| C10 | A request reads the cached document | `llmprovider/providers/opencode/opencode.go:499` | `meta, cached = catalog.CachedMetadataWith(ctx, p.gateway, p.listing...)` |
| C11 | With no package for the model, its own route | `llmprovider/providers/opencode/opencode.go:512` | `if model == p.model {` |
| C12 | Catalog's tests clear the process's cache | `llmprovider/catalog/model_metadata_test.go:29` | `clear(processStore.metadata)` |
| C13 | No opencode test runs in parallel | `llmprovider/providers/opencode/helpers_test.go:194` | `No test here runs in parallel.` |
| C14 | CI shuffles | `.github/workflows/ci.yml:46` | `go test -shuffle=on ./...` |
| C15 | The refusal tests give each provider its own Cache | `llmprovider/providers/opencode/learn_0034_test.go:65` | `catalog.WithCache(new(catalog.Cache))` |
| C16 | Refusals are keyed by provider and model | `llmprovider/internal/wire/messages/learn.go:17` | `by provider and model, oldest first` |
| C17 | The opencode switch for metadata | `llmprovider/providers/opencode/helpers_test.go:198` | `func enableMetadata(t *testing.T) {` |
| C18 | 0057's PLAN leaves it here | `docs/decisions/0057-PLAN-wizard-device-code-and-paste-prompt.md:666` | `test's isolation is a new record's, the next number, not this PLAN's. No` |
| C19 | Catalog's helper names the hazard | `llmprovider/catalog/model_metadata_test.go:23` | `httptest reuses ports, so a cache keyed by` |
| C20 | A fourth map keyed by listing | `llmprovider/catalog/cache.go:59` | `ollamaShows map[listingKey]map[string]showReply` |
| C21 | A later listing reads it back | `llmprovider/catalog/listing_cache.go:157` | `for name, r := range c.ollamaShows[key] {` |
| C22 | resetLookupCaches clears listings, not the shows | `llmprovider/catalog/lookup_0033_test.go:104` | `clear(processStore.listing)` |

## Amendment 2026-10-23: a fourth cache, and catalog's own tests

* **Found** at the PLAN's Phase 4, before any change to package
  `catalog`.
  * **F1 named three of the store's maps keyed by a server's URL. There
    are four.** `ollamaShows` holds Ollama's `/api/show` replies, keyed by
    listing, and so by base URL (C20). A later listing reuses a reply
    whose age and digest still match (C21). D5's census did not watch it.
    No test helper clears it. A test with its own `Cache` (D1) has its own
    `ollamaShows` too, so Phases 1 to 3 were not exposed through it.
  * **Of the census's 110 catalog tests, 72 reach `processStore` through
    the package's own helpers:** `enableModelMetadata`,
    `resetLookupCaches`, the wait helpers, and `useSteadyClock` and
    `replaySnapshot`, which call them. Several of them wait on the
    process's in-flight fetches. The helpers clear only part of the store.
    `enableModelMetadata` clears the metadata (C12). `resetLookupCaches`
    also clears the listings (C22). Neither clears `ollamaShows`. The other
    38 tests use none of these helpers.
* **The owner chose (2026-10-23), "option a":**
  * the 72 tests keep `processStore`. The helpers they call clear every
    map keyed by a URL, `ollamaShows` included, before the test starts and
    when it ends. Q3 (a) said a test keeps the process's store "unless it
    reads `processStore` directly"; it now also covers a test that reaches
    it through these helpers;
  * the 38 take their own cache, as Q3 (a) said;
  * D5's census gains `ollamaShows`.

  Moving the 72 to their own caches, with their waits rewritten, was put
  to the owner and not chosen. D1 stands for every package outside
  `catalog`.

## Amendment 2026-10-24 (proposed): a provider's default URL, and a test run twice

* **Found** during 0059-PLAN Phase 5 (its record, "Found, not in this
  PLAN's scope"). The PLAN's Out of scope left out "tests that share the
  process's cache under a URL that is not a loopback one, such as a
  provider's default URL served by a stubbed transport. They do not depend
  on a port." That they do not depend on a port holds. But a listing's key
  is the provider, the base URL, the Kilo organization and the
  credential's fingerprint (`llmprovider/catalog/listing_cache.go`,
  `listingKey`). A stubbed transport at a provider's default URL leaves an
  entry that any later listing with the same key literal reads: the test
  itself when it runs again in the same process, or another test that
  types the same key.
* **Facts,** at `7f75456`:
  * **G1.** `go test -count=2 ./...`: 35 packages pass and one fails,
    `wizard`, with two tests on their second run:
    `TestResolveCredential_AsksNoProviderNoModel_0056`, "catalog live
    true, usable ["claude-haiku-5-5"], after 0 listings; want the one live
    listing", and `TestResolveCredential_ErrorKinds_0056/ctx_ended_during_the_listing`,
    "error <nil>; want context.Canceled, not the static catalog as a
    checked key". Both list Claude through `claudeRefusals`, at Claude's
    default URL, with the process's cache. On the second run the listing
    is a cache hit: no request is made, so the first counts none and the
    second's transport never cancels its context.
  * **G2.** D5's census, copied to a scratch file with its loopback filter
    removed and the whole listing key printed, over the whole module at
    `-count=1 -p 1`: in the wizard, four tests write the process's cache
    at a URL that is not a loopback one, all at Claude's default URL:
    the two of G1, `TestConfigureLLM_RefusedTypedKeyIsAskedAgain`
    (`wizard/configure_0028_test.go`) and
    `TestConfigureLLM_UnusableListingNotice`
    (`wizard/model_select_edge_test.go`). The other two pass at
    `-count=2`, as their assertions do not count listings, but they leave
    the same kind of entry.
  * **G3.** In that census, four entries are written by more than one
    test: Claude's listing and its listed set at Claude's default URL (two
    `catalog` tests), the opencode metadata URL (eleven `catalog` tests),
    and one probe key (two `llmprovider/internal/transport` tests). Each of
    those tests clears the process's store first, through
    `resetLookupCaches` or `resetProbeCache`, as this record's amendment of
    2026-10-23 decided for `catalog`. No two wizard tests share a key, so CI,
    at `-count=1`, is not exposed today; a repeat, or a new test that types
    a key another already lists with, is.
  * **G4.** Outside the wizard, the other writers at such a URL are the
    deliberate `WithCache(nil)` steps D1 keeps, in claude, gemini, grok and
    openai, each under a credential unique to the run, and they pass at
    `-count=2`.
  * **G5.** The gate runs the suite at `-count=1` (`test-race-cover`,
    `test-shuffle`); only `g-wire-stable` repeats, and only the wire
    goldens. Nothing local runs a test twice in one process.
* **Proposed,** for the owner's answers:
  * **D6.** D1's rule extends to every wizard test that lists or probes
    through the catalog, whatever the URL: its `Options.ProviderOptions`
    carries `testCache(t)`. Under Q4 (a), the four tests of G2.
  * **D7.** Under Q5, how a recurrence is caught:
    * D5's census gains `--any-url`: no loopback filter, the listing key
      printed whole (the credential's fingerprint, never the credential),
      and each key that more than one test writes reported. Its default
      run is unchanged.
    * `tools/gate.py` gains `test-repeat`, `go test -count=2 ./...`. At
      `7f75456` it took about 157 s of package time, 41 s of it
      `catalog`'s, the packages in parallel. Not CI's (Q2 (a) stands).
  * As D4, no library code changes, no exported name is added, and no
    release is needed.
* **Confirmation,** for D6 and D7:
  * `go test -count=2 ./...` passes, where G1 failed;
  * a planted removal of the cache from one of G1's tests fails it again
    at `-count=2`, with G1's text;
  * under Q5 (a), the census's `--any-url` reports no wizard test, and
    fails its own test on a planted break;
  * under Q5 (a) or (b), the gate's `test-repeat` fails on `7f75456`'s
    tests and passes after D6.
* **Owner questions:**
  * **Q4 (D6). Which wizard tests.**
    * **(a)** The four of G2 (recommended): the two that fail on a repeat,
      and the two that leave the same entry.
    * **(b)** The two of G1 only.
  * **Q5 (D7). What catches the next one.**
    * **(a)** Both the census's `--any-url` and the gate's `test-repeat`
      (recommended). The repeat catches a test that reads its own entry;
      the census catches two tests sharing a key, which a repeat does not
      and a shuffle finds only by chance.
    * **(b)** The gate's `test-repeat` only.
    * **(c)** Neither: the tests of Q4 only.
