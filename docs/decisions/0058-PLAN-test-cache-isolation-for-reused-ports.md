---
status: complete
date: 2026-10-23
associated-madr: "0058-MADR-test-cache-isolation-for-reused-ports.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Cache of Its Own for Every Test That Points the Catalog's Cache at an httptest Server

Associated MADR: [0058-MADR-test-cache-isolation-for-reused-ports.md](0058-MADR-test-cache-isolation-for-reused-ports.md)

## Goal

The MADR's D1 to D5, under the owner's answers to Q1 and Q2 (2026-10-22),
and to Q3 once given. The aim is that no test writes an `httptest` server's
URL into the process's metadata, listing or probe cache, apart from the
steps that test the process's cache on purpose. Those steps use a
credential unique to the run. Three things show it:

* the cache census (D5) reports no other test, in any package in scope;
* D3's regression test passes, and fails on its planted break with CI's
  text;
* the opencode package, run 300 times shuffled on linux/amd64, does not
  fail. Before the change, P6 failed 4 times in 300 runs, and P8 3
  times in 150 (MADR F6).

## Scope

### In scope

* **The test helpers.** Each of these packages gets `testCache(t)` in its
  shared test helpers, and its `build` adds it first:
  * `llmprovider/providers/{claude,gemini,grok,huggingface,kilo,ollama,openai,opencode,together}`,
    in `helpers_test.go`;
  * `llmprovider/providers`, in a new `cache_0058_test.go`. It has no
    `build`, so its direct `New` calls take it;
  * `wizard`, in `fake_prompter_test.go`, beside the other helpers.
* **The tests the census names** (MADR F9): 97 tests in 12 packages outside
  `catalog`. Each test gets the cache through `build` or explicitly:
  * a direct `New` or `catalog.*` call;
  * each package's `llmtest` `Harness.New` and wire-golden constructor;
  * the wizard's `Options.ProviderOptions`;
  * `TestLiveFetcher`'s `extra` options.
* **The tests that use the process's cache on purpose.** These steps
  are kept, and give a credential unique to the run (MADR D1):
  * the `WithCache(nil)` step of `TestProbeCache_ProviderProbesIntoItsCache_0049`
    in claude, gemini, grok and ollama;
  * the same step of `TestProbeCache_ProviderProbesIntoItsCache_0048` in
    openai;
  * any other step the census finds that tests the process's cache, named
    in the phase's record.
* **D3's regression test,** `llmprovider/providers/opencode/reuse_0058_test.go`.
* **D5's tool:** `tools/cachecensus.py`, its test in `tools/test_tools.py`,
  and its row in `tools/README.md`.
* **Package `catalog`'s tests, under Q3 (a) only** (Phase 4): the census's
  110 tests there.
* **Records:** this pair, and its index rows.

### Out of scope

* Any change outside `_test.go` files, `tools/` and the records. Library
  code, CI and the `Makefile` are unchanged (MADR D4).
* Any exported name. The PLAN adds none, so R55's table is empty.
* A gate that CI runs (MADR Q2 (a)). The census is a maintainer tool, as
  `tools/` is (0035-MADR).
* Tests that share the process's cache under a URL that is not a loopback
  one, such as a provider's default URL served by a stubbed transport.
  They do not depend on a port. Any the census finds by chance are listed
  in the record, not changed. *(Amended 2026-10-24, proposed: the wizard's
  are Phase 6's. See the MADR's amendment of that date.)*
* `CHANGELOG.md`. The change is to tests and tools only, and no release is
  made. `changelog-check` checks only the tags' sections.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase, once Q3 is
  answered. A phase starts after the owner has committed the last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `make tools-test` when `tools/` changes;
  * `tools/claims.py` on the MADR, the record with an evidence table;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter.
* **The census, after each phase:** `python3 -B tools/cachecensus.py` lists
  no test of the phase's packages, apart from the deliberate steps named
  in the phase. Its report goes in the phase's record.
* **Red first.** Every new test, and the census tool's test, is seen to
  fail on a planted break by `tools/plant.py`, on a scratch clone, never in
  the tree. The phase's record holds each plant and its failure line.
* **No assertion changes.** A test changes only by the cache option, or by
  the run-unique credential of a deliberate step. A test that then fails
  for another reason is a stop condition. It is not loosened, skipped or
  rewritten.
* **Stage, then the owner commits** with `git commit --no-edit`. Push
  happens on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * a test fails once given its own cache, because it relied on another
    test warming the process's cache;
  * a test outside the census needs a change;
  * the census tool cannot find one of its instrumentation anchors.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. The owner's answer to Q3 is written into the MADR's "Owner questions".
   Under (b), Phase 4 is struck, with the date.
2. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.

### Phase 1: the census tool, opencode, and the regression test (D2, D3, D5)

1. **`tools/cachecensus.py`,** written as the MADR's P7 ran:
   * a scratch clone of the working tree (`_common.scratch_clone`);
   * the instrumentation, by exact replacement, each anchor asserted
     present once:
     * `startMetadataFetch`, once the fetch is registered;
     * `storeListing`, before its lock;
     * `storeFetched`, past its empty-listing return;
     * `transport.ProbeGenerateHealthCached`, at entry.

     Each prints `CENSUS <kind> <key>` on stderr when the store is the
     process's and the key holds `127.0.0.1`;
   * `go test -count=1 -v -p 1 -parallel 1 ./...` in the clone, its exit
     status captured, and its log kept in the tool's output directory;
   * each line attributed to its package and top-level test by the log's
     order, and printed by package, with `--json` for agents;
   * exit 0 when the suite passed, 1 when it failed, 2 when an anchor is
     missing.
2. **Its test** in `tools/test_tools.py`:
   * the attribution, over a short sample log;
   * every anchor present once in the tree.
3. **`tools/README.md`:** its row.
4. **`llmprovider/providers/opencode/helpers_test.go`:** `testCache(t)`,
   one `catalog.Cache` per test, the same for every call in that test. It
   is held in a `sync.Map` keyed by `t` and dropped by `t.Cleanup`, with a
   comment citing this record. `build` puts it before the caller's
   options, so that a test's own `WithCache` comes later and wins.
5. **The opencode tests of the census** (16), where they bypass `build`:
   * `metadata_route_test.go`: the warm-up at line 113 passes
     `testCache(t)`;
   * `listing_test.go` and `metadata_cold_0028_test.go`: their
     `catalog.*` calls;
   * `opencode_test.go`: its direct `New` calls that reach a server;
   * `llmtest_test.go` and `wire_test.go`: the `Harness.New` and the
     wire-golden constructor.
6. **`reuse_0058_test.go`:** `TestOpencode_ReusedPortRoutesByItsOwnDocument_0058`,
   two subtests run in order. They model two tests on one port, so each
   has its own `t` and so its own `testCache(t)`. Within one `t` the two
   servers would share a cache by design, and B would see A's document:
   * `first`: server A serves a document that gives `brand-new-model`
     `@ai-sdk/openai-compatible`. The subtest warms it through
     `testCache(t)` and generates once through `build`
     (`/chat/completions`). It closes A and hands its address to the
     second;
   * `second`: server B starts with `httptest.NewUnstartedServer` and a
     listener from `net.Listen("tcp", <A's address>)`, and serves
     `@ai-sdk/openai`. The URLs are asserted equal. Through `build` and
     the same warm-up, the request goes to `/responses`.

   The test does not use the process's cache at any step (MADR D3). A
   failed re-listen, the port taken in between, fails the test with that
   reason. It is not skipped.
7. **Plants:**
   * `testCache` returning `catalog.WithCache(nil)`: the regression test
     fails with `path = "/chat/completions", want "/responses"`;
   * `build` without `testCache`: the same failure;
   * the census tool with one anchor changed: exit 2, naming it;
   * the census tool's attribution off by one line: its test fails.
8. **The census** lists no opencode test.

### Phase 2: the other provider packages, and `internal/cmd/catalogs` (D1, D2)

1. **`testCache(t)`** in the `helpers_test.go` of claude, gemini, grok,
   huggingface, kilo, ollama, openai and together, as Phase 1 step 4
   wrote it, and in `llmprovider/providers/cache_0058_test.go`. Each
   package's `build` puts it first. The root package has no `build`,
   so its direct `New` calls take it, by step 2.
2. **The census's tests of these packages** (64 in 9 packages), where they
   bypass `build`:
   * direct `New` calls (`ollama/listing_0026_test.go`,
     `together/listing_test.go`, `providers/facts_0041_test.go`, and those
     the census names);
   * direct `catalog.*` calls (`huggingface/listing_test.go`,
     `together/listing_test.go`, `grok/efforts_0041_test.go`,
     `providers/facts_0041_test.go`);
   * each package's `llmtest` `Harness.New` and wire-golden constructor.
3. **The deliberate steps** (Scope): the `WithCache(nil)` step's provider
   is given a key from `crypto/rand.Text()`, generated for that step. The
   step still asserts that the process's cache probes again. With a key
   unique to the run, its entry cannot meet any other test's. It
   cannot meet its own from an earlier `-count` iteration either.
4. **`internal/cmd/catalogs/main_refresh_0033_test.go`:**
   `TestLiveFetcher`'s `extra` gains `catalog.WithCache(new(catalog.Cache))`.
5. **The drift check:** `go test -race -count=1` on each package of the
   phase passes, with no assertion changed.
6. **Plants:**
   * for one package of each kind, its `build` without `testCache`. The
     census is expected to list that package's tests again: claude for
     probes, together for listings, the root package for metadata;
   * a deliberate step given the earlier steps' cache in place of nil.
     Its test is expected to fail on "want probes true", which shows the
     step still tests the process's cache. The credential's part is shown
     by the key's construction, which holds its fingerprint (MADR C7), not
     by a plant.
7. **The census** lists none of the phase's tests, apart from the
   deliberate steps.

### Phase 3: the wizard (D1, D2)

1. **`wizard/fake_prompter_test.go`:** `testCache(t)`.
2. **The census's 16 wizard tests:** each `Options` they pass to
   `ConfigureLLM` gets `testCache(t)` in `ProviderOptions`. `zenOptions()`
   becomes `zenOptions(t)`, and its callers pass `t`. A test that already
   sets `ProviderOptions` appends to them.
3. **The drift check:** `go test -race -count=1 ./wizard/`.
4. **Plant:** `zenOptions` without the cache; the census lists its callers
   again.
5. **The census** lists no wizard test.

### Phase 4: package `catalog`'s own tests (Q3 (a) only)

1. **`testCache(t)`** in `llmprovider/catalog/model_metadata_test.go`,
   beside `resetModelMetadataCache`.
2. **The census's 110 catalog tests.** *(Amended 2026-10-23, the owner's
   "option a": a test that reaches `processStore` through the package's
   clearing and waiting helpers keeps it too, and those helpers clear every
   map keyed by a URL, `ollamaShows` included. See Phase 4's record.)* Each
   test does one of two things, recorded per test:
   * it takes `testCache(t)` in its options. This is the default;
   * it keeps the process's store, because it reads `processStore`
     directly (`metadata_request_test.go`, `model_metadata_test.go`,
     `lookup_0033_test.go`, `menu_0041_test.go`, and any the phase
     finds). Such a test clears every map it writes, before it starts and
     in `t.Cleanup`, through the existing helpers.
3. **The drift check:** `go test -race -count=1 ./llmprovider/catalog/`.
4. **Plant:** one converted file back on the process's store; the census
   lists its tests again.
5. **The census** lists only the tests of step 2's second kind, and the
   record shows that each one clears.

### Phase 5: close

1. **The census, whole:** its report holds only the deliberate steps, and
   under Q3 (a) step 4.2's clearing tests.
2. **On linux/amd64,** in a copy of the phase's commit outside the tree:
   * the opencode package's test binary, 300 runs with
     `-test.shuffle=on` and `-test.count=1`: no failure;
   * the route test with `-count=2000`: no failure.
3. **The gate,** and CI on the three platforms.
4. **The MADR's Confirmation, item by item;** this PLAN `complete`.

### Phase 6: a provider's default URL, and a test run twice (MADR amendment of 2026-10-24; proposed)

Waits for the owner's answers to Q4 and Q5, and approval. The rules for
every phase hold, with "the census" meaning, here, its `--any-url` run.

1. **Records.** `tools/records.py status`: this PLAN `in-progress`; the
   MADR's amendment marked decided, with the answers.
2. **The tests of Q4** (D6). Each `Options` they pass gains `testCache(t)`
   in `ProviderOptions`, appended to any already there:
   * `TestResolveCredential_AsksNoProviderNoModel_0056` and
     `TestResolveCredential_ErrorKinds_0056`'s "ctx ended during the
     listing" (`wizard/resolve_0056_test.go`);
   * under Q4 (a), `TestConfigureLLM_RefusedTypedKeyIsAskedAgain`
     (`wizard/configure_0028_test.go`) and
     `TestConfigureLLM_UnusableListingNotice`
     (`wizard/model_select_edge_test.go`).

   No assertion changes (the rule "No assertion changes").
3. **Under Q5 (a): `tools/cachecensus.py --any-url`** (D7). Without the
   flag, the run and its report are as today. With it:
   * no loopback filter, in the catalog's store and the probe store;
   * a listing, a listed set or a show is reported by its whole key,
     provider, base URL, organization and credential fingerprint;
   * the report adds, for each key, the tests that wrote it, and lists
     apart each key written by more than one test.

   Its test in `tools/test_tools.py` covers the filter, the key and the
   shared-key report. `tools/README.md`'s row says what the flag does.
4. **Under Q5 (a) or (b): `tools/gate.py`'s `test-repeat`,** `go test
   -count=2 ./...`, after `test-shuffle`. `tools/README.md` and any test
   that lists the gate's checks follow.
5. **Red first, and plants** (`tools/plant.py`, a scratch clone):
   * at `7f75456`, `go test -count=2 ./wizard/` fails G1's two tests (the
     MADR's G1 quotes them); after step 2, it passes;
   * the cache removed from `TestResolveCredential_AsksNoProviderNoModel_0056`,
     run by `go test -count=2` on the package: "after 0 listings";
   * under Q5 (a), the loopback filter kept under `--any-url`: the tool's
     test fails, and the census, run on `7f75456`, lists G2's four tests
     before step 2 and none after;
   * under Q5 (a) or (b), `test-repeat` run on a scratch clone of
     `7f75456`: it fails, with G1's lines.
6. **Close:**
   * `go test -race -count=1 -shuffle=on ./wizard/` and `go test -count=2
     ./...` pass;
   * under Q5 (a), `tools/cachecensus.py --any-url` reports no wizard test,
     and its shared keys are G3's, each test of which clears the store;
   * `python3 -B tools/gate.py` passes, with `test-repeat` under Q5 (a) or
     (b);
   * the Confirmation of the MADR's amendment of 2026-10-24, item by
     item, and this PLAN `complete` again.
7. **Not in scope:** library code, CI, the `Makefile`, `CHANGELOG.md`
   (tests and tools only, as Out of scope says), and the deliberate and
   clearing tests of G3 and G4, which stay as they are.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | A request on a reused port routes by its own server's document | 1 | `TestOpencode_ReusedPortRoutesByItsOwnDocument_0058`, and its plants |
| V2 | No test outside `catalog` writes a loopback URL into the process's caches, apart from the deliberate steps | 1–3 | `tools/cachecensus.py` |
| V3 | Each deliberate step still asserts the process's cache, under a credential unique to the run | 2 | the steps' tests, and step 2.6's second plant |
| V4 | Under Q3 (a), catalog's tests isolate or clear | 4 | `tools/cachecensus.py`, and the record per test |
| V5 | No assertion changed | 1–4 | the drift checks, and the diffs |
| V6 | The census tool fails on a missing anchor, and its test on a broken attribution | 1 | its plants, `make tools-test` |
| V7 | The opencode package does not fail in 300 shuffled runs on Linux | 5 | step 5.2 |
| V8 | No exported name, and no library code changed | 1–4 | `tools/apidiff.py v1.22.0`, and `git diff --stat` outside `_test.go`, `tools/` and `docs/` |
| V9 | No wizard test leaves an entry in the process's cache at any URL; the whole module passes `-count=2` | 6 | step 6.6 |
| V10 | Under Q5, the census's `--any-url` and the gate's `test-repeat` each fail on their planted break | 6 | step 6.5 |

## Rollout and Rollback

* **Rollout.** Commits on `main`, staged by the agent and committed by the
  owner: the records, then one commit per phase. No tag, no release.
* **Rollback.** Each phase reverts on its own, newest first. A revert puts
  its packages back on the process's cache, and with it the flake. No
  user of the module sees any of it.

## Execution record

### Phase 0: records (2026-10-23)

* **Approval.** The owner answered Q1 and Q2 on 2026-10-22 ("questions:
  follow recommendations"), and Q3 on 2026-10-23 with "Q3: a and
  proceed". Phase 4 stays.
* **Step 1.** The three answers are in the MADR's "Owner questions", and
  its Decision Outcome names T1 as chosen.
* **Step 2.** `tools/records.py status`: the MADR `accepted` and this PLAN
  `in-progress`, both dated 2026-10-23, with their index rows.
* **Not done here:** no code. Phase 1 starts after the owner has committed
  this phase.
* **Committed** as `2160673`, on the owner's word ("Commit to main, then
  proceed").

### Phase 1: the census tool, opencode, and the regression test (2026-10-23)

* **Step 1, `tools/cachecensus.py`.** As written, with two choices the step
  left open:
  * the anchors are checked in the source tree before the clone is made, so
    a moved anchor costs no clone;
  * the census functions sit in two new files in the clone (`catalog` and
    `transport`), so no import line is edited.

  `go test` runs with stderr merged into stdout, since the attribution
  depends on the order of the two. Its first run, on the opencode package
  at `2160673`, listed the same 16 tests as the MADR's P7.
* **Step 2, its tests:** `TestCacheCensus` in `tools/test_tools.py`, four
  tests. The fourth, `test_an_anchor_found_twice_is_named`, was added
  after step 7's plant "an anchor found twice taken as once" came back NOT
  CAUGHT (exit 0) against the first three. With it, the plant was caught.
* **Step 3:** its row in `tools/README.md`.
* **Step 4:** `testCache(t)` and `build` in `helpers_test.go`, as written.
  Its doc also says why one cache serves the whole test: a test's servers
  stay open until it ends, so none shares a port with another within it.
* **Step 5, where the tests bypass `build`.** After step 4 alone, the census
  listed 5 opencode tests, not the 16 of P7. One test failed:
  `TestOpencode_RoutesFromMetadata/over_table`, `path = "/messages", want
  "/chat/completions"`. That is the state step 5 exists to end, not a stop
  condition. The route test's warm-up still filled the process's cache,
  while its provider read its own cache, which was empty, and a cold cache
  routes `minimax-m3` by the table, as 0028-MADR D-A1 requires.
* **Deviation (2026-10-23): step 5's file list was a guess, and the census
  gave the true one.** Changed:
  * `metadata_route_test.go`: the warm-up passes `testCache(t)`;
  * `metadata_cold_0028_test.go`: the `LookupMetadataWith` call of
    `TestOpencode_ColdCacheUsesTableRoute`;
  * `metadata_ua_0026_test.go`: its direct `NewGo`;
  * `listing_test.go`: `TestListModels_HonoursTheMetadataURLOption`'s
    options, which its `catalog.List` and its `build` share;
  * `wire_test.go`: the wire-golden constructor.

  `opencode_test.go` and `llmtest_test.go`, which step 5 named, needed no
  change: once step 4 was in, the census listed none of their tests.
* **Deviation (2026-10-23): the wire goldens take a cache per build, not
  `testCache(t)`.** `wirecase.Run` builds once per scenario, in a subtest of
  its own, against a server that the subtest closes before the next one
  starts. One cache for `TestWireGoldens` would be shared by servers that
  can have the same port, which is the hazard itself. So each `Build`
  gives `catalog.WithCache(new(catalog.Cache))`. This is D1's rule, a cache
  for each server's life, and no decision of the MADR changes.
* **Step 6, the regression test:** `reuse_0058_test.go`. To start B on A's
  address, `routeServer` became a call to a new `routeServerAt(t, addr, …)`,
  whose `addr` of "" is any port. The handler and every assertion are
  unchanged.
* **Step 7, the plants** (`tools/plant.py`, scratch clones):

  ```text
  == testCache returns the process's cache: caught (exit 1)
     --- FAIL: TestOpencode_ReusedPortRoutesByItsOwnDocument_0058/second (0.00s)
     reuse_0058_test.go:45: path = "/chat/completions", want "/responses"
  == build without testCache: caught (exit 1)
     --- FAIL: TestOpencode_ReusedPortRoutesByItsOwnDocument_0058/second (0.00s)
     reuse_0058_test.go:45: path = "/chat/completions", want "/responses"
  plant: all 2 plant(s) caught

  == a missing anchor not reported: caught (exit 1)
  == an anchor found twice taken as once: caught (exit 1)
     AssertionError: Lists differ: [] != ["a.go: anchor found 2 times: 'x()'"]
  == attribution carried over the package line: caught (exit 1)
  == a subtest taken as its own test: caught (exit 1)
  == CONT not followed: caught (exit 1)
  plant: all 5 plant(s) caught
  ```

  The failure text is CI's for `v1.22.0`, as the MADR's Confirmation asks.
* **Step 8, the census:** `tools/cachecensus.py
  ./llmprovider/providers/opencode/` gave "0 test(s) in 0 package(s) wrote a
  loopback URL into the process's caches; go test exit 0".
* **The checks:**
  * `go test -race -count=1 -shuffle=on ./llmprovider/providers/opencode/`:
    ok;
  * `make pre-add-check` on the 7 Go files: "7 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)";
  * `make tools-test`: 73 tests, OK;
  * `python3 -B tools/gate.py`: "gate: 27 of 27 passed".
* **Not done here:** the other packages, which Phases 2 to 4 cover. Until
  then they still write to the process's cache, so a full census lists
  them.
* **Committed** as `65e3cf6`, on the owner's word ("Commit to main then
  proceed").

### Phase 2: the other provider packages, and `internal/cmd/catalogs` (2026-10-23)

* **Step 1.** `testCache(t)` in the `helpers_test.go` of claude, gemini,
  grok, huggingface, kilo, ollama, openai and together, with the doc
  Phase 1 gave it, and each `build` puts it first. The root package gets
  it in the new `cache_0058_test.go`. After this step alone, the census
  over the phase's ten packages listed 19 tests, not P7's 65. Two of them
  failed:
  * `TestGrokEfforts_Facts_0041`, whose warm-up filled the process's cache
    while its provider read its own. This is the state step 2 ends, as in
    Phase 1;
  * `TestProbeCache_ProviderKeyIsFingerprinted` (openai), which reads the
    process's probe store itself, `transport.ProbeGenerateHealthCached`
    with a nil store, after a listing. That makes it a deliberate step, as
    Scope's last item foresees: "any other step the census finds that
    tests the process's cache".
* **Step 2, where the tests bypass `build`:**
  * every package's wire-golden constructors go through a new `wireOpts`
    in its `wire_test.go`, which adds a cache per build. This is Phase 1's
    deviation for opencode, for the same reason;
  * `grok/efforts_0041_test.go`: `testCache(t)` in the options its warm-up
    and its provider share;
  * `huggingface/listing_test.go` and `together/listing_test.go`: the same,
    in `TestListModels_HonoursTheMetadataURLOption`;
  * `providers/facts_0041_test.go`: the same, in
    `TestReplayField_Facts_0041`.

  The files step 2 named for their direct `New` (`ollama/listing_0026_test.go`,
  `together/listing_test.go`'s other tests) needed no change: the census
  listed none of their tests once step 1 was in.
* **Step 3, the deliberate steps.** Each keeps the process's cache and
  takes a key from `crypto/rand.Text()`, made once per test as
  `processKey`:
  * the `WithCache(nil)` step of `TestProbeCache_ProviderProbesIntoItsCache_0049`
    in claude, gemini, grok and ollama, and of `_0048` in openai. Ollama
    sends no key, but its probe key holds the fingerprint of the one it is
    given, so a key of the run's own isolates it too;
  * `TestProbeCache_ProviderKeyIsFingerprinted`: `listProbes` takes
    options, the test gives `catalog.WithCache(nil)`, and its key is
    `"k-fingerprint-" + rand.Text()`.

  No assertion changed.
* **Step 4:** `TestLiveFetcher`'s `extra` gains
  `catalog.WithCache(new(catalog.Cache))`.
* **Deviation (2026-10-23): `TestLiveFetcher`'s metadata read cannot take a
  cache.**
  * **Found.** With step 4 in, the census still listed it, for
    "metadata". `liveFetcher.Metadata` is `return
    catalog.LookupMetadata(ctx, f.metadataURL, nil)`
    (`internal/cmd/catalogs/main.go:105`), which takes no option. It was
    so before this record. The PLAN's scope rules out a change outside
    `_test.go` files.
  * **Options put to the owner:**
    * a metadata URL of the run's own, in the test only (recommended);
    * `liveFetcher.Metadata` given the fetcher's options, a change to the
      command and an amendment of this PLAN's scope.
  * **Decision.** "Option 1". The test's `metadataURL` is `srv.URL + "/" +
    rand.Text() + "/api.json"`, which its handler serves by the
    `/api.json` suffix as before. Its entry's key can meet no other test's,
    as the deliberate steps' keys cannot. The census still lists it, and
    Phase 5 names it beside them. The MADR's T2 was rejected as the general
    fix, since it reaches only the metadata; here the metadata is all that
    is left, so no decision of the MADR changes.
  * **Seen:** the census line became `CENSUS metadata
    http://127.0.0.1:64038/BWHOYYOI2NHGCYFE2IHXENROV3/api.json`, and
    `go test -race -count=3 -run TestLiveFetcher` passed.
* **Step 5, the drift check:** `go test -race -count=1 -shuffle=on` on the
  phase's ten packages: ok, each.
* **Step 6, the plants.** `tools/plant.py`, a scratch clone, `-run
  TestProbeCache_`:

  ```text
  == claude: the nil step given cache a: caught (exit 1)
     probe_cache_0049_test.go:48: step 3: 0 probes; want probes true
  == gemini: the nil step given cache a: caught (exit 1)
     probe_cache_0049_test.go:48: step 3: 0 probes; want probes true
  == grok: the nil step given cache a: caught (exit 1)
     probe_cache_0049_test.go:48: step 3: 0 probes; want probes true
  == ollama: the nil step given cache a: caught (exit 1)
     probe_cache_0049_test.go:35: step 3: 0 probes; want probes true
  == openai 0048: the nil step given cache a: caught (exit 1)
     probe_cache_0048_test.go:35: step 3: 0 probes; want 3
  == openai fingerprint: the provider on the test's own cache: caught (exit 1)
     probe_cache_0028_test.go:99: by the fingerprint: 3 probes; want the cached entry
  plant: all 6 plant(s) caught
  ```

  The census plants, each on a scratch copy given to
  `tools/cachecensus.py --src`, with the census expected to list the tests
  again:

  ```text
  == claude: build without testCache (probes): caught; listed [TestClaude_TokenHeaderOverride,
     TestListModels, TestListModels_CarriesTheCallersIdentity, TestListModels_DefaultBaseShape_0041,
     TestListModels_ListingBounded, TestListModels_ProbeBodyIsTheOldProbes,
     TestListModels_ProbesFollowDefaultOptionAndEnv, TestProbeCache_ProviderProbesIntoItsCache_0049]
  == together: build without testCache (listings): caught; listed [TestListModels_CarriesTheCallersIdentity,
     TestListModels_NeverProbes, TestTogether_Listing, TestTogether_TokenHeaderOverride]
  == root: TestReplayField_Facts_0041 without testCache (metadata): caught; listed [TestReplayField_Facts_0041]
  ```

* **Step 7, the census** over the phase's ten packages, every test passing,
  lists only the deliberate steps and the deviation's test:

  ```text
  llmprovider/internal/cmd/catalogs: TestLiveFetcher metadata
  llmprovider/providers/claude: TestProbeCache_ProviderProbesIntoItsCache_0049 listed,listing,probes
  llmprovider/providers/gemini: TestProbeCache_ProviderProbesIntoItsCache_0049 listed,listing,probes
  llmprovider/providers/grok: TestProbeCache_ProviderProbesIntoItsCache_0049 listed,listing,probes
  llmprovider/providers/ollama: TestProbeCache_ProviderProbesIntoItsCache_0049 probes
  llmprovider/providers/openai: TestProbeCache_ProviderKeyIsFingerprinted listed,listing,probes
  llmprovider/providers/openai: TestProbeCache_ProviderProbesIntoItsCache_0048 listed,listing,probes
  ```

* **The checks:**
  * `make pre-add-check` on the phase's 28 Go files: clean, and again on
    the deviation's file once it changed;
  * `python3 -B tools/gate.py`: "gate: 27 of 27 passed".
* **Not done here:** the wizard (Phase 3) and package `catalog` (Phase 4).
* **Committed** as `35164d1`, on the owner's word ("Commit to main and
  proceed").

### Phase 3: the wizard (2026-10-23)

* **Step 1.** `testCache(t)` in `fake_prompter_test.go`, with the doc of
  Phases 1 and 2.
* **Step 2.** `zenOptions(t)` adds `testCache(t)` to its
  `ProviderOptions`, and its 14 callers in 7 files pass `t`. That covers
  `describe_0056_test.go`'s, whose test the census did not list. After
  this alone, the census listed 3 of P7's 16 wizard tests, each with its
  `Options` written inline. They got `testCache(t)` in their
  `ProviderOptions`:
  * `TestConfigureLLM_OllamaLeavesOutEmbedding_0042` (three subtests) and
    `TestConfigureLLM_OllamaToolless_0042`, in `model_select_edge_test.go`;
  * `TestConfigureLLM_ProviderOptionsReachTheListing`, beside its metadata
    URL.
* **Deviation (2026-10-23), within the step: one caller is in a
  goroutine.** `TestConfigure_EmptyRecommendedHonoursCtx` calls
  `ConfigureLLM` in a goroutine that can outlive the test, whose deadline is
  300 ms. `testCache` registers a `t.Cleanup`, which must not run after the
  test has ended, so the options are built on the test's goroutine,
  before the `go`. No assertion changed.
* **Step 3, the drift check:** `go test -race -count=1 -shuffle=on
  ./wizard/`: ok.
* **Step 4, the plant:** `zenOptions` without the cache, on a scratch copy.
  The census listed its 13 callers again:

  ```text
  == wizard: zenOptions without the cache: caught; listed [TestConfigureLLM_BlankFallbackRoundWithNothingLeft,
     TestConfigureLLM_FallbackSearch, TestConfigureLLM_FallbackSearchLoops,
     TestConfigureLLM_PreselectsExistingFallbacks, TestConfigureLLM_ProfileReachesListing,
     TestConfigureLLM_SearchResultsCapped, TestConfigureLLM_SearchUsesLiveCorpus,
     TestConfigure_EmptyRecommendedHonoursCtx, TestConfigure_EmptyRecommendedOpensSearch,
     TestConfigure_EmptyRecommendedSearchStillWorks, TestConfigure_TiersOnlyAreOffered,
     TestWizard_MenuIsFullListByDefault, TestWizard_ModelLimit]
  ```

* **Step 5, the census:** "0 test(s) in 0 package(s) wrote a loopback URL
  into the process's caches; go test exit 0".
* **The checks:** `make pre-add-check` on the phase's 9 Go files, "9
  file(s) clean"; `python3 -B tools/gate.py`, "gate: 27 of 27 passed".
* **Not done here:** package `catalog` (Phase 4).
* **Committed** as `b110d5d`, by the owner ("committed, proceed").

### Phase 4: package `catalog`'s own tests (2026-10-23)

* **Deviation (2026-10-23): a fourth cache, and 72 tests already on the
  process's store.**
  * **Found,** before any change, by sorting the census's 110 tests by
    whether they reach `processStore` themselves or through a test helper
    that does:
    * the store's `ollamaShows`, Ollama's `/api/show` replies keyed by
      listing (`cache.go:59`), is a fourth map keyed by a URL. The census
      did not watch it, and no helper clears it;
    * 72 tests reach `processStore` through `enableModelMetadata`,
      `resetLookupCaches`, the wait helpers, `useSteadyClock` or
      `replaySnapshot`. Several wait on its in-flight fetches.
      `enableModelMetadata` clears only the metadata;
    * 38 reach it through none of them.
  * **Options put to the owner:**
    * A: the 72 keep the process's store; the helpers clear every map
      keyed by a URL, before and after; the 38 take their own cache; the
      census gains `ollamaShows` (recommended);
    * B: all 110 take their own cache, the 72's waits rewritten to the
      test's cache.
  * **Decision.** "option a, proceed". The MADR's amendment of 2026-10-23
    records it, with F1 corrected to four maps. Step 2 is annotated, and
    steps 1 and 5 hold for the 38 and the 72 in turn.
* **The census tool** gains a fifth instrument, `storeShow` (kind
  `shows`). Its doc and its `tools/README.md` row say five, and its tests
  pass. On `catalog`, `providers/ollama` and `wizard` together, 15 catalog
  tests wrote a show reply. None in the other two did, so Phases 1 to 3
  were not exposed through that map.
* **Step 1, the helpers** in `model_metadata_test.go`:
  * `resetProcessStore`: the metadata, then the listings and the show
    replies, each after its fetches in flight. `resetListingCache` also
    clears `ollamaShows` now;
  * `clearProcessStore(t)`: `resetProcessStore`, now and in `t.Cleanup`;
  * `enableModelMetadata` clears through it, so its tests no longer leave
    listings behind;
  * `testCache(t)`, as in the other packages.
* **Step 2, the 110.** Of the 38 that reach none of the helpers:
  * 36 in package `catalog` take `testCache(t)`. A script added it within
    those test functions only, after a line's first `WithBaseURL` or else
    `WithModelMetadataURL`: 45 places in 13 files. One, a single-value
    assignment in `identification_test.go`, failed to compile, and the
    option was moved to the call that uses the value;
  * `TestListModels_HonoursRankingOptions` (`catalog_test`) takes a cache
    of its subtest's own, which its `List` and its provider share;
  * `TestCache_KeepsListingsApart_0048` (`catalog_test`) is a deliberate
    step: its no-cache and `WithCache(nil)` steps test the process's cache.
    `key0048` became a key from `crypto/rand.Text()`, one per test. It was
    `"key-0048-" + t.Name()`, which another `-count` run of the same test
    shares.

  Of the 72, and of the tests below that the census still listed:
  * `TestLoadModelMetadata_Disabled` and `TestOptionsFromEnv` cleared the
    metadata before and never after. Each now starts with
    `clearProcessStore(t)`;
  * `TestMetadata_ReasoningEffortsIsACopy`, which the first sort put among
    the 38, calls `LookupMetadata`. That takes no cache, and the test is
    about the process-wide cache, so it keeps the process's store and
    starts with `clearProcessStore(t)`.

  No assertion changed.
* **Step 3, the drift check:** `go test -race -count=1 -shuffle=on
  ./llmprovider/catalog/`: ok.
* **Step 4, the plants.** On the clearing, through a new test,
  `TestResetProcessStore_EmptiesEveryURLMap_0058` (`clear_0058_test.go`),
  which fills each of the four maps and calls `resetProcessStore`:

  ```text
  == the show replies not cleared: caught (exit 1)
     clear_0058_test.go:30: after resetProcessStore: 0 metadata, 0 listing and 1 show entries; want none
  == resetProcessStore clears only the metadata: caught (exit 1)
     clear_0058_test.go:30: after resetProcessStore: 0 metadata, 1 listing and 1 show entries; want none
  == resetProcessStore clears only the listings: caught (exit 1)
     clear_0058_test.go:30: after resetProcessStore: 1 metadata, 0 listing and 0 show entries; want none
  plant: all 3 plant(s) caught
  ```

  On the cache: `testCache` giving the process's cache, on a scratch copy.
  The census listed 110 tests again, the 36 among them.
* **Step 5, the census** of `./llmprovider/catalog/`: 74 tests. A script
  followed each one's calls through the test helpers to
  `clearProcessStore`:
  * 73 reach it: 25 through `enableModelMetadata`, 24 through
    `resetLookupCaches`, 11 through `replaySnapshot`, 10 through
    `useSteadyClock`, and 3 directly (the three tests above);
  * one does not: `TestCache_KeepsListingsApart_0048`, the deliberate
    step, under its key of the run's own.
* **The checks:** `make pre-add-check` on the phase's 19 Go files, "19
  file(s) clean"; `make tools-test`, OK; `python3 -B tools/gate.py`, "gate:
  27 of 27 passed".
* **Not done here:** Phase 5, the close.
* **Committed** as `54e71fb`, by the owner ("committed, proceed").

### Phase 5: close (2026-10-23)

* **Step 1, the census, whole:** `tools/cachecensus.py` over `./...` at
  `54e71fb`, every test passing, listed 82 tests in 7 packages:
  * outside `catalog`, only the deliberate steps of Phase 2 and
    `TestLiveFetcher` (Phase 2's deviation):
    `TestProbeCache_ProviderProbesIntoItsCache_0049` in claude, gemini,
    grok and ollama, `_0048` and `TestProbeCache_ProviderKeyIsFingerprinted`
    in openai, and `TestLiveFetcher` in `internal/cmd/catalogs`;
  * in `catalog`, 75: Phase 4's 74 and
    `TestResetProcessStore_EmptiesEveryURLMap_0058`, which fills the store
    on purpose. Phase 4's script found that 74 reach `clearProcessStore`.
    The one that does not is `TestCache_KeepsListingsApart_0048`, the
    deliberate step under its key of the run's own.
* **Step 2, on linux/amd64,** in a copy of `54e71fb` made by `git archive`,
  outside the tree:
  * the opencode package's test binary, 300 runs with `-test.shuffle=on`
    and `-test.count=1`: "loop runs=300 fails=0". Before the change, the
    MADR's P6 failed 4 times in 300 and P8 3 times in 150;
  * `go test -count=2000 -run 'TestOpencode_RoutesFromMetadata$'`: ok,
    58.4 s.

  The copy was removed afterwards.
* **Step 3, the gate:** `python3 -B tools/gate.py`, "gate: 27 of 27
  passed". The disclosure guard over the six commits not yet pushed
  (`b66ecdf` to `54e71fb`) exited 0, with nothing flagged.
* **Not done yet:** CI on the three platforms, which runs on a push and so
  waits for the owner's word, and step 4, the MADR's Confirmation item by
  item and this PLAN `complete`, which follows CI.
* **Committed and pushed** by the owner, as `dfd65f9` ("committed and
  pushed"). "CI", run 39970710624: `validate` passed on macos-15,
  windows-2025 and ubuntu-24.04, and `release` was skipped. "Docs", run
  39970710621, passed.
* **Step 4, the MADR's Confirmation, item by item:**
  * **"D3's test passes, and fails on its planted break with `path =
    \"/chat/completions\", want \"/responses\"`."** Met, Phase 1 step 7: two
    plants, each failing with that text.
  * **"The route test passes `-count=2000` on linux/amd64."** Met, step 2.
  * **"D5's census reports no test in scope, apart from the deliberate
    steps of D1."** Met, step 1. As the amendment of 2026-10-23 decided, it
    also lists `TestLiveFetcher` (Phase 2's deviation) and the catalog
    tests that keep the process's store and clear it.
  * **"The opencode package, run 300 times shuffled on linux/amd64 as P6
    ran it, does not fail."** Met, step 2: 0 of 300.
  * **"`python3 -B tools/gate.py` passes, and CI passes on the three
    platforms."** Met, step 3 and run 39970710624.
  * **"The PLAN's record lists each test of F9's census, and each test
    found beyond it, with what changed or why nothing did."** Met in part.
    The records name, test by test, every test that still writes to the
    process's caches and why, and every test that needed more than a
    package's helper (Phases 1 to 4). The tests that a helper covered are
    recorded by helper and by the census before and after: 16 to 0 in
    opencode, 65 to 7 in Phase 2's ten packages, 16 to 0 in the wizard,
    and 110 to 74 in `catalog`, the 74 each shown to clear. They are not
    listed one by one. P7's census holds their names.
* **The PLAN is `complete`.** The MADR stays `accepted`.
