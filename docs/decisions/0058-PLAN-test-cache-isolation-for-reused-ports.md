---
status: in-progress
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
  in the record, not changed.
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
2. **The census's 110 catalog tests.** Each test does one of two things,
   recorded per test:
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
