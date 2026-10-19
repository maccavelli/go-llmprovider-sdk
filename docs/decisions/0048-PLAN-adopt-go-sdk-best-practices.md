---
status: in-progress
date: 2026-10-19
associated-madr: "0048-MADR-adopt-go-sdk-best-practices.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Closing the SDK's Gaps Against Go SDK Practice

Associated MADR: [0048-MADR-adopt-go-sdk-best-practices.md](0048-MADR-adopt-go-sdk-best-practices.md)

## Goal

* **D1:** a caller owns its caches: `catalog.Cache`, `catalog.WithCache`,
  and `providers.DefaultWith`, with today's process-wide cache the default.
* **D2:** every public package has a runnable example, and the package
  comments of `llmprovider`, `providers` and `wizard` show the simplest call.
* **D3, D4:** every exported type states its concurrency contract, and every
  exported function returning `error` its errors.
* **D5:** `CHANGELOG.md`, a gate on it, and a CI job that makes each tag's
  GitHub Release from it.
* **D6 (A1):** the Go floor stays `go 1.27.2`, its reasons stated.
* **D7:** each new export justified in the record that adds it.
* **D8:** `api-standards.md` rules R49–R55, X1–X3's reasons, and the checks:
  `sdkaudit` kept as `llmprovider/internal/cmd/sdkaudit`, behind
  `make docs-check`.
* **Release** `v1.16.0`, which makes the first Release by D5's job.

## Scope

### In scope

* **New:**
  * `llmprovider/internal/cmd/sdkaudit/` (with `testdata/`);
  * `llmprovider/internal/cacheref/`;
  * `catalog.Cache`, `catalog.WithCache`, `providers.DefaultWith`;
  * an `example_test.go` in each of the 15 public packages;
  * `CHANGELOG.md`;
  * `scripts/check_changelog.py`, `scripts/check_docs.py`.
* **Changed:**
  * **code:** `llmprovider/catalog/` (listing and metadata caches),
    `llmprovider/internal/transport/probe.go`,
    `llmprovider/internal/wire/probe.go`, the five probing providers, and
    `llmprovider/providers/providers.go`;
  * **doc comments:** in every public package (D3, D4);
  * **CI and gates:** `.github/workflows/ci.yml`, `Makefile`,
    `tools/gate.py`, `tools/test_tools.py`, `scripts/test_gates.py`;
  * **docs:** `docs/guides/api-standards.md`, `docs/architecture.md`,
    `AGENTS.md`, `README.md`, `tools/README.md`, `docs/README.md`.

### Out of scope

* `go.mod`'s `go` line (A1); go-selfupdate-lib (the MADR's Context); a `v2`;
  X1–X3's code.
* The consumers' code. Phase 7 proves the consumer case on a scratch copy and
  tells the consumers; adoption is their own record.

## Rules for every phase

* **Approval.** 0048-MADR is accepted with A1, and the owner approves this
  PLAN, whole or by phase. A phase starts after the owner has committed the
  last.
* **The gate, before staging:** `python3 -B tools/gate.py` passes every
  check; `make pre-add-check FILES="…"` on the phase's Go files;
  `tools/denyscan.py --changed`; `make records-check`; `tools/claims.py` on
  this PLAN; `tools/recordlint.py --changed`. Each exit status is captured
  before any filter.
* **Red first.** Every new test, check and gate case is seen to fail on a
  planted break, by `tools/plant.py` (its spec's text in files beside it where
  it spans lines) or `scripts/test_gates.py`, on a scratch clone, never in the
  tree. The phase's record holds each plant and its failure line.
* **The wire and the floors.** G-wire's goldens do not change (R45);
  `make coverage-check` holds every floor (R47); `tools/apidiff.py v1.15.0`
  shows only the additions this PLAN names.
* **D7.** Each exported name a phase adds is listed in its record with the
  call site that needs it.
* **Stage, then the owner commits** with `git commit --no-edit`; push and tags
  on the owner's word.
* **Stop conditions:**
  * a fact of the MADR or a claim below no longer holds;
  * a step would change an exported signature, or make `apidiff` report a
    change other than an addition;
  * a golden changes;
  * a doc comment would state a behaviour no test shows.

  Stop, record a dated deviation here (and an amendment in the MADR when a
  decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. 0048-MADR `accepted` (`tools/records.py status … accepted`); this PLAN
   indexed (`tools/records.py add …`). `make records-check`.

### Phase 1: the audit command (D8)

1. **Port** the scratch `sdkaudit` (the MADR's method) to
   `llmprovider/internal/cmd/sdkaudit/`, `package main`, the standard library
   only:
   * `audit.go`: the parse and the measures, as the MADR's table lists them;
   * `main.go`: `-report` (the JSON, the default) and `-check`, with
     `-rules` (default `R50,R51,R52`) and a root argument (default `.`).
2. **`-check`'s rules,** each violation printed as `path:line Rnn what`, exit
   1 on any, 0 clean, 2 on a parse error:
   * **R50a:** each public package (outside `internal/`) has at least one
     `Example…` function, in a `_test.go` file, whose body ends with an
     `// Output:` comment.
   * **R50b:** the package comments of `llmprovider`, `llmprovider/providers`
     and `wizard` hold an indented (tab) code block.
   * **R51:** each exported type of a public package has a doc comment that
     contains `concurrent use` (case-insensitive).
   * **R52:** each exported function or method of a public package returning
     `error` has a doc comment matching
     `` \bErr[A-Z]\w*|\berrors?\b|\bfail|\brefus|\bAPIError\b ``; the methods
     `MarshalJSON`, `UnmarshalJSON`, `MarshalText`, `UnmarshalText`,
     `Error`, `String`, `GoString`, `LogValue` and `Format` are exempt.
3. **Tests** (`main_test.go`), on `testdata/` modules: `good/`, which `-check`
   passes; `bad/`, with one of each defect the MADR's planted package held and
   one of each R50–R52 violation, each counted once by `-report` and named once
   by `-check`. Coverage at least 80 % (C11).
4. **Tables:** a row for `llmprovider/internal/cmd/sdkaudit` in
   `docs/architecture.md`'s package table and in R2's, beside
   `internal/cmd/catalogs` (C17, C18), importing the standard library only.
5. **`tools/README.md`** and AGENTS.md's tools list name it, and why it is Go
   (MADR D8).
6. **Red first:** plants of each rule's test (R50a's `// Output:` match; R51's
   phrase; R52's exemption list emptied), the command `go test
   ./llmprovider/internal/cmd/sdkaudit/`.
7. **The baseline:** `go run ./llmprovider/internal/cmd/sdkaudit -check .` on
   the tree; its counts per rule go into the record. They are the work list
   of Phases 3 and 4. A count of R52 other than 27, or of R50a other than 15,
   is recorded with each extra or missing name, not a stop.

### Phase 2: caches a caller can own (D1)

1. **The probe store.** `llmprovider/internal/transport/probe.go`: the cache
   becomes `type ProbeStore struct` (its mutex, its map) with
   `NewProbeStore()` and the process's `defaultProbes`;
   `ProbeGenerateHealthCached` (C6) takes a `*ProbeStore`, nil meaning the
   process's. `probeCacheNow` stays the clock.
2. **The catalog store.** `llmprovider/catalog/`: the listing state
   (`listing_cache.go`, C1 of the MADR) and the metadata state
   (`model_metadata.go`) move into one unexported `store` struct, with the
   process's `processStore`. `config` (configFor, C3) gains `store *store`,
   the process's unless an option names one; every function that reads the
   state reads `cfg.store`. `defaultConfig` (C4) keeps the process's. The
   clocks `listingNow` (C12) and `metadataNow` stay package seams. The test
   helpers (`resetListingCache`, C19; `ResetModelMetadataCache`, C20) reset
   `processStore`.
3. **The handle.** `llmprovider/internal/cacheref`: `type Value struct {
   Catalog any; Probes *transport.ProbeStore }` and `From(values []any)
   *Value`. A row in both package tables.
4. **The API:**
   * `catalog.Cache`: a struct whose zero value is ready, built once on first
     use (`sync.Once`), safe for concurrent use, not to be copied after first
     use (its doc says so; `go vet`'s copylocks holds it).
   * `catalog.WithCache(c *Cache) llmprovider.Option`, a
     `ScopedOptionFor(builtinIDs, "catalog.WithCache", …)` (C2) whose value is
     the `*cacheref.Value`; a nil `c` is the process's.
   * Each provider reads `cacheref.From(st.Values())` in `New` and passes its
     `Probes` to `wire.ProbeHealth` (C5), which gains a `*transport.ProbeStore`
     parameter (internal). The five callers: `gemini.go:355`, `claude.go:418`,
     `grok.go:373`, `ollama.go:309`, `openai.go:521`. Catalog calls get the
     cache through `p.listing`, which already carries every option (C9).
   * `providers.DefaultWith(opts ...llmprovider.Option) *llmprovider.Registry`
     (C7, C8): `Default()`'s registry, each built-in factory called with
     `opts` first and the caller's options after, so a later option wins (C10).
     `Default()` and `providers.New` are unchanged.
5. **Tests:**
   * two `catalog.Cache` values do not share a listing (a fake server's
     request count: one per cache);
   * a zero `Cache` works; `WithCache(nil)` is the process's;
   * a provider built `WithCache(c)` serves `CachedFacts` from `c` after its
     `ListModels`, and not from another cache;
   * two `ProbeStore`s do not share a probe;
   * `DefaultWith`: an option reaches every built-in; a caller's later option
     wins; a factory the caller registers is untouched;
   * 32 goroutines listing through one `Cache`, under `-race`.
6. **Red first:** `configFor` ignoring `WithCache` (the isolation test
   fails); `DefaultWith` dropping `opts`; `ProbeHealth` ignoring its store.
7. **D7:** `catalog.Cache`, `catalog.WithCache` and `providers.DefaultWith`,
   each with its call site: a test that isolates its listings
   (prepare-commit-msg's case, the MADR's C18); a process that keeps more
   than one configuration; a registry whose providers share one cache.
8. **Docs:** `docs/architecture.md` (the caches, the new package); the
   README, a short "Caches" section.

### Phase 3: stated concurrency and errors (D3, D4)

1. **Each exported type** of Phase 1's R51 list (76 at the MADR, Appendix A)
   gets a sentence with `concurrent use`. The sentence says what the code
   does:
   * **a value type** (`Request`, `ModelInfo`, …): "Safe for concurrent use
     only while no goroutine changes it";
   * **an interface a caller implements:** what an implementation must be:
     "Implementations must be safe for concurrent use: a provider calls
     `Token` from many goroutines at once" (`TokenSource`); likewise
     `TokenStore`, `RefreshLocker`, `Prompter`, `Layer`, `Streamer`,
     `ModelLister`;
   * **a stateful type** (`FileTokenStore`, `VendorCLISession`,
     `TextPrompter`, `Catalog`): its behaviour as measured first, by a test of
     concurrent use under `-race`. A type found unsafe is documented "not safe
     for concurrent use", and is not changed in this PLAN. (Amended by step 5:
     a `-race` test also for four interface and function-type sentences.)
2. **Each function** of Phase 1's R52 list (27 at the MADR, Appendix B) gets
   a sentence naming its error kinds, read from its code paths and its tests:
   each provider's `New`: `ErrInvalidRequest` for a missing or empty
   credential or a foreign option, `ErrUnsupported` for a credential kind the
   service does not take; `GenerateText`: `Generate`'s errors and
   `ErrIncomplete` for an answer with no text; and so on. A kind with no test
   that shows it is a stop condition. (Met; amended by step 5: the missing
   tests are written in this phase.)
3. **Check:** `go run ./llmprovider/internal/cmd/sdkaudit -check -rules
   R51,R52 .` exits 0.
4. **Red first:** a plant removing one R51 sentence, and one R52 sentence;
   the command the check above.
5. **Deviation (2026-10-19), chosen by the owner.** Step 2's stop condition
   was met. Read from the code, the error kinds of about 14 of the 27
   functions have no test through the function itself:
   * the logins: `StartDeviceOAuth`, `DeviceLogin.Wait`, `LoginDeviceOAuth`
     and `LoginBrowserOAuth` (context ends, transport, status and decode
     failures);
   * `NewFileTokenStore`, `ValidateOllamaURLWith`, `Metadata.Encode`;
   * `ConfigureLLM` (no test asserts a Prompter's error with `errors.Is`) and
     `TextPrompter`'s `Select`, `MultiSelect`, `Confirm` and `Input`;
   * no `errors.Is` for `ValidateOAuthSession`'s `ErrAuthFailure`,
     `FileTokenStore.Delete`'s `ErrInvalidProvider`, or `CommandToken.Token`'s
     context errors.

   Four R51 sentences would also state a behaviour no test shows: a provider
   calls a `Layer`'s `Model` from many goroutines; two `Registry.New` calls
   run one `Factory` at once; `ConfigureLLM` calls `Notify` while an `Input`
   waits; `VendorCLISession` is safe for concurrent use (step 1 already asks
   for that one's test).

   The resolution: step 2 writes, for every kind a doc names, a test through
   that function where none exists, each red first; step 1 writes a `-race`
   test for each of the four. No code changes: a doc states what the code
   does now. By the same rule, `Settings`, `Option` and `Metadata`, whose
   sentences say they are safe, get a `-race` test as the stateful types do. Rejected: docs that name only what is tested, and docs that
   state an implementation's requirement without its reason.

   Found and left out of this PLAN, for a record of their own: a browser
   login's `access_denied` is a plain error where the device login's is
   `ErrAuthFailure`; `ValidateOllamaURL`'s errors are not classified as the
   Ollama listing's are; an unsupported provider in a login is a plain
   error; a typed-nil `*StaticToken` makes `ResolveOptions` panic, and so
   does a zero `DeviceLogin`'s `Wait`.

### Phase 4: examples and package docs (D2)

1. **One `example_test.go` per public package** (`package <name>_test`), each
   offline, each with `// Output:`:
   * `llmprovider`: `ExampleGenerateText` over `llmtest.Fake`;
     `ExampleWithRetry`;
   * `llmprovider/auth`: `ExampleNewFileTokenStore`, a Save and Load in a
     temporary directory, printing the provider;
   * `llmprovider/catalog`: `ExampleStatic`, printing that the list is not
     empty; `ExampleWithCache`, two caches and their request counts;
   * `llmprovider/llmtest`: `ExampleFake`;
   * `llmprovider/providers`: `ExampleNew` and `ExampleDefaultWith`;
   * each of the nine provider packages: `ExampleNew`, against an `httptest`
     server answering that wire's shape, printing the reply's text;
   * `wizard`: `ExampleConfigureLLM`, a `TextPrompter` over a
     `strings.Reader`, printing the chosen provider and model.
2. **Package comments** of `llmprovider`, `providers` and `wizard`: an
   indented example, the README's quick start in `llmprovider`'s.
3. **Check:** `sdkaudit -check -rules R50` exits 0; `go test ./...` runs the
   examples.
4. **Red first:** an example's `// Output:` line changed (its `go test`
   fails); one package's example removed (`sdkaudit` fails).

### Phase 5: the changelog and the Releases (D5)

1. **`CHANGELOG.md`,** in the Keep a Changelog 1.1.0 form:
   * an `## [Unreleased]` section, then one `## [vX.Y.Z] - YYYY-MM-DD` per tag,
     newest first, 23 sections;
   * each date the tag's commit date (`git log -1 --format=%cs vX.Y.Z`);
   * the text, from the README's Status bullets (C14) for the releases they
     name; from the PLANs' release-notes steps where a bullet is thin; and,
     for a tag neither covers, the commit subjects outside `docs/` between it
     and the tag before, under "Changed";
   * comparison links at the end.
2. **`scripts/check_changelog.py`** (`make changelog-check`):
   * every `v*` tag has a section, newest first, its date the tag's;
   * an `[Unreleased]` section is allowed;
   * `--section TAG` prints that section's body, exit 1 if none.
3. **Wiring:** `changelog-check` in the Makefile and its `.PHONY` (C15), CI's
   list (C13), `tools/gate.py` after `fixtures-check` (C16),
   `scripts/test_gates.py`'s `GATES` (C21), and `tools/test_tools.py`'s order
   test. Selftest cases: a tag with no section; two sections out of order;
   `--section` of an unknown tag.
4. **The release job,** a second job in `ci.yml`:
   * `release`, `needs: validate`;
   * `if: github.event_name == 'push' && startsWith(github.ref,
     'refs/tags/v')`;
   * `runs-on: ubuntu-24.04`, `permissions: contents: write` (the workflow's
     own stays `contents: read`, C22);
   * the same pinned checkout with `fetch-depth: 0`;
   * `python3 -B scripts/check_changelog.py --section "$GITHUB_REF_NAME" >
     notes.md`;
   * `gh release create "$GITHUB_REF_NAME" --title "$GITHUB_REF_NAME"
     --notes-file notes.md --verify-tag`, with `GH_TOKEN: ${{ github.token }}`.

   CI already runs on `v*` tags (C23), so `validate`'s `changelog-check` fails
   a tag without a section before the job runs.
5. **The README's Status** keeps the current release and its Go line (C24),
   and links `CHANGELOG.md` for the rest.
6. **Red first:** the selftest's three cases; a plant of `--section`'s exit
   code.

### Phase 6: the standard and the docs gate (D6–D8)

1. **`api-standards.md`:**
   * **R49 caches** (D1): what a cache holds, `WithCache`, the process's
     default.
   * **R50 examples** (D2).
   * **R51 concurrency** (D3).
   * **R52 errors** (D4).
   * **R53 changelog and Releases** (D5).
   * **R54 the Go floor** (D6, A1): `go 1.27.2`, because `encoding/json/v2`
     needs Go 1.27, and 1.27.2 carries the 2026-10-08 security fixes; raised
     with each security patch.
   * **R55 a new export is justified** (D7).

   Each cites 0048-MADR. The Construction section gains X1–X3's reasons. The
   Checks table: R50–R52 `make docs-check`; R53 `make changelog-check`; R54
   `go.mod` and CI's `go-version-file`. R48 (C25) is unchanged.
2. **`scripts/check_docs.py`** (`make docs-check`): runs `go run
   ./llmprovider/internal/cmd/sdkaudit -check .` and returns its exit code; 2
   when `go` is missing. Wired as Phase 5 step 3 wires `changelog-check`.
   Selftest cases, in the copied tree: an exported type's concurrency
   sentence removed; an R52 sentence removed; a package's example removed.
3. **AGENTS.md:** the CI list gains `docs-check` and `changelog-check`; "API
   work" says a PLAN adding exports lists each with its call site (R55); the
   Go line's reason, beside "Requires Go 1.27.2" (C26).
4. **`docs/architecture.md`:** the checks and the CI job.
5. **Red first:** the selftest's three cases.

### Phase 7: release `v1.16.0`

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.16.0] - <date>`:
   * Added: `catalog.Cache`, `catalog.WithCache` and `providers.DefaultWith`;
     the examples;
   * Changed: the stated concurrency and errors;
   * Repository: `CHANGELOG.md` and the Releases.

   The README's Status names `v1.16.0`.
2. **Before the push:**
   * the gate;
   * `tools/apidiff.py v1.15.0`: the three additions, nothing else;
   * the disclosure guard over the outgoing commits.

   The owner pushes; CI green on three systems.
3. **The owner tags `v1.16.0`** and pushes the tag. CI's `release` job makes
   the Release. Check `gh release view v1.16.0 --json body` against
   `check_changelog.py --section v1.16.0`, which must be equal.
4. **After the tag:**
   * `tools/consumer.py v1.16.0` (six platforms);
   * `tools/offline.py v1.16.0`;
   * on a scratch copy of prepare-commit-msg moved to `v1.16.0`, its listing
     tests changed to pass a `catalog.WithCache(new(catalog.Cache))` per test
     and no per-run key: they pass. This is the MADR's confirmation of D1; the
     copy is never committed.
5. **The 23 past Releases,** on the owner's word: each tag's section, written
   to a file and scanned with `tools/denyscan.py`, shown first; then `gh
   release create <tag> --notes-file … --verify-tag --latest=false`. Each URL
   goes into the record.
6. **The consumers,** on the owner's word: a comment or an issue in
   prepare-commit-msg (its per-run keys can go), and in gobble-cli if it lists.
   Each is written to a file, scanned and shown first.
7. **Close:** the MADR's Confirmation items checked, each with its evidence;
   this PLAN set `complete` with `tools/records.py`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | `sdkaudit` reports and checks, on a planted bad module and a good one | 1 | `go test ./llmprovider/internal/cmd/sdkaudit/` |
| V2 | Two caches never share a listing or a probe; the default is the process's | 2 | Phase 2's tests, `-race` |
| V3 | `apidiff` shows only `catalog.Cache`, `catalog.WithCache`, `providers.DefaultWith` | 2, 7 | `tools/apidiff.py v1.15.0` |
| V4 | Every exported type states its concurrency, every error-returning export its errors | 3 | `sdkaudit -check -rules R51,R52` |
| V5 | Every public package has a running example; three package comments show code | 4 | `sdkaudit -check -rules R50`; `go test ./...` |
| V6 | Every tag has a changelog section; a tag without one fails | 5 | `make changelog-check`; `gate-selftest` |
| V7 | `docs-check` and `changelog-check` run in CI and the local gate | 5, 6 | CI's log; `tools/gate.py` |
| V8 | The goldens and the coverage floors hold | all | G-wire; `make coverage-check` |
| V9 | `v1.16.0`'s Release body is its changelog section, made by CI | 7 | `gh release view`; `--section` |
| V10 | A consumer's listing tests isolate with a `Cache` and no per-run key | 7 | step 4's scratch run |
| V11 | Every new check failed first | 1–6 | the execution record |

## Rollout and Rollback

* Consumers get `v1.16.0` by `go get`. Nothing changes for a caller that
  passes no `WithCache`: the process's cache is the default.
* **Rollback before the tag:** revert the phase's commit.
* **After the tag:** a `v1.16.1` can restore any behaviour. The three
  additions stay, since R48 forbids their removal in `v1`.
* **A Release made wrong** is edited in place (`gh release edit`); a tag is
  never moved.

## Code claims

<!-- claims: self=f8ab6bf -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | A scoped option outside its providers is refused; inside, it is kept | `llmprovider/settings.go:257` | `func (s *settings) take(id ProviderID, opt Option) error {` |
| C2 | Catalog options are scoped to the built-in ids | `llmprovider/catalog/config.go:91` | `var builtinIDs = []llmprovider.ProviderID{` |
| C3 | Catalog's configuration is built from options | `llmprovider/catalog/config.go:158` | `func configFor(` |
| C4 | The no-option configuration | `llmprovider/catalog/config.go:199` | `func defaultConfig() config {` |
| C5 | The probe's entry point takes no options today | `llmprovider/internal/wire/probe.go:13` | `func ProbeHealth(` |
| C6 | The probe cache's one reader and writer | `llmprovider/internal/transport/probe.go:139` | `func ProbeGenerateHealthCached(` |
| C7 | `Default()` today | `llmprovider/providers/providers.go:43` | `func Default() *llmprovider.Registry {` |
| C8 | A factory's signature | `llmprovider/registry.go:9` | `type Factory func(opts ...Option) (Provider, error)` |
| C9 | A provider keeps every option for its catalog calls | `llmprovider/providers/claude/claude.go:191` | `p.listing = append(append([]llmprovider.Option(nil), opts...),` |
| C10 | Options apply in order; a later one wins | `llmprovider/settings.go:17` | `Options apply in order, so a later one wins;` |
| C11 | A new package's coverage floor is 80 % | `scripts/check_coverage.py:28` | `DEFAULT_FLOOR = 80.0` |
| C12 | The listing cache's clock | `llmprovider/catalog/listing_cache.go:74` | `listingNow = time.Now` |
| C13 | CI's list of gates | `.github/workflows/ci.yml:60` | `records-check tables-check fixtures-check gate-selftest` |
| C14 | The README's Status section | `README.md:284` | `## Status` |
| C15 | The Makefile's `.PHONY` | `Makefile:14` | `.PHONY: all help test` |
| C16 | The local gate's `fixtures-check` | `tools/gate.py:100` | `make("fixtures-check"),` |
| C17 | `internal/cmd/catalogs`'s R2 row | `docs/guides/api-standards.md:39` | `` the maintainer tool's command, run with `go run` `` |
| C18 | `internal/cmd/catalogs`'s architecture row | `docs/architecture.md:125` | `` `llmprovider/internal/cmd/catalogs` `` |
| C19 | The listing cache's test reset | `llmprovider/catalog/lookup_0033_test.go:100` | `func resetListingCache(` |
| C20 | The metadata cache's test reset, exported to `catalog_test` | `llmprovider/catalog/export_test.go:10` | `ResetModelMetadataCache` |
| C21 | `gate-selftest`'s map of gates | `scripts/test_gates.py:39` | `"fixtures-check": "scripts/check_fixtures.py",` |
| C22 | The workflow's permission | `.github/workflows/ci.yml:19` | `contents: read` |
| C23 | CI runs on `v*` tags | `.github/workflows/ci.yml:5` | `tags: ['v*']` |
| C24 | The README's release and Go line | `README.md:286` | `` The current release is `v1.15.0`. The module requires Go 1.27.2. `` |
| C25 | R48 | `docs/guides/api-standards.md:277` | `` **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is `` |
| C26 | AGENTS.md's Go line | `AGENTS.md:10` | `Requires Go` |

## Appendix A: the 76 types without a concurrency note (the MADR's audit)

* `llmprovider` (41): `APIError`, `AuthMethod`, `AuthMethodID`,
  `CacheRetention`, `Capabilities`, `Descriptor`, `Effort`, `Event`,
  `EventType`, `Factory`, `FinishReason`, `ForcedChoiceFallback`,
  `FunctionCallItem`, `FunctionCallOutputItem`, `InvalidatingSource`, `Item`,
  `MessageItem`, `ModelLister`, `Option`, `ProviderID`, `Reasoning`,
  `ReasoningItem`, `Request`, `Response`, `RetryNotice`, `RetryPolicy`,
  `Role`, `Settings`, `StaticToken`, `Streamer`, `Support`, `Token`,
  `TokenInvalidator`, `TokenSource`, `TokenType`, `Tool`, `ToolChoice`,
  `ToolLoadItem`, `Usage`, `Warning`, `WarningKind`.
* `llmprovider/auth` (8): `ChatGPTPlanOptions`, `FileTokenStore`,
  `KiloAccount`, `KiloOrganization`, `OAuthFlowOptions`, `RefreshLocker`,
  `TokenStore`, `VendorCLISession`.
* `llmprovider/catalog` (18): `Catalog`, `Caveat`, `Field`, `Layer`,
  `ListSource`, `Match`, `MenuEntry`, `Metadata`, `Modality`, `ModelCost`,
  `ModelCostTier`, `ModelInfo`, `ModelRates`, `ModelReasoning`, `ModelStatus`,
  `Profile`, `RawListing`, `RawModel`.
* `llmprovider/llmtest` (1): `Harness`. `llmprovider/providers/opencode`
  (1): `Route`.
* `wizard` (7): `Choice`, `CredentialKind`, `Level`, `Options`, `Prompter`,
  `Result`, `TextPrompter`.

These are the audit's lists at `b8ffd4b`; Phase 1's baseline, run with the
kept command's exact R51 phrase, is the list of record.

## Appendix B: the 27 error-returning exports whose doc names no error

* `llmprovider`: `CommandToken.Token`, `GenerateText`, `StaticToken.Token`.
* `llmprovider/auth`: `StartDeviceOAuth`, `DeviceLogin.Wait`,
  `LoginDeviceOAuth`, `LoginBrowserOAuth`, `ValidateOAuthSession`,
  `NewFileTokenStore`, `FileTokenStore.Delete`.
* `llmprovider/catalog`: `ValidateOllamaURLWith`, `Metadata.Encode`.
* Each provider's `New`: `claude`, `gemini`, `grok`, `huggingface`, `kilo`,
  `ollama`, `openai`, `together`; `opencode.NewZen`, `opencode.NewGo`.
* `wizard`: `ConfigureLLM`, `TextPrompter.Select`, `TextPrompter.MultiSelect`,
  `TextPrompter.Confirm`, `TextPrompter.Input`.

## Execution record

### Phase 0: records (2026-10-19)

* **No deviation.**
* The owner committed the MADR as `f8ab6bf` and this PLAN, with the MADR's
  A1, as `21f9c73`, and pushed. The owner then said "accept and approve,
  proceed to phase 0".
* After this PLAN was written, the owner asked again for a `go 1.26.9`
  floor, was shown that it needs a build-tag split or the removal of
  `encoding/json/v2`, and kept `go 1.27.2` (A1 stands). The owner also chose
  `toolchain go1.27.2`; on a scratch module, `go mod tidy` removes that line
  when it equals the `go` line, and this repository's `go mod tidy -diff`
  would fail it, so `go.mod` gets no toolchain line, as A1 says.
* `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`,
  both index rows with them; "records-check: 0 problem(s)".

### Phase 1: the audit command (2026-10-19)

* **No deviation.** Phase 0 was committed by the owner as `72e70a0`.
* **Steps 1–2.** `llmprovider/internal/cmd/sdkaudit/`: `main.go` (flags,
  `run`, and a `writer` that keeps the first write error, as
  `internal/cmd/catalogs`'s does), `audit.go` (`load`, and `-check`'s R50,
  R51, R52), `measure.go` (the report). The standard library only.
* **Step 3, the tests,** five on `testdata/bad` and `testdata/good` and one on
  a failing writer:
  * the report counts each planted defect once, and classes each of eight
    package-level variables;
  * `-check` names exactly R51 `Doer`, R52 `Quiet` and R50's two;
  * the good module passes, its `internal/` package unread;
  * `-rules` selects;
  * six tool errors exit 2; a failed write exits 2.

  An unparsable file is written by the test into a temporary directory, not
  committed: a committed one fails `gofmt -l .`, which CI runs. Coverage
  91.2 % (`make coverage-check`, against R47's 80 %).
* **Step 4.** Rows in `docs/architecture.md`'s package table and tree, and in
  R2's table, "the standard library"; `make tables-check`: "36 packages, 0
  problem(s)".
* **Step 5.** `tools/README.md`'s section "The one tool in Go", and AGENTS.md's
  tools list.
* **Step 6, red first,** `tools/plant.py`, the `_0048$` tests the command,
  five plants, all caught:
  * R50's `Output:` match made always true;
  * R51's phrase emptied;
  * R52's exemption list emptied;
  * internal packages counted public;
  * a failed write ignored.

  The R52 plant was first not caught: the good module's `MarshalJSON` doc
  said "error", which R52's pattern matches without the exemption. The doc
  was reworded so only the exemption lets it pass, and the plant then failed
  `TestCheckPassesTheGoodModule_0048`.
* **Step 7, the baseline,** `go run ./llmprovider/internal/cmd/sdkaudit
  -check .`, exit 1:
  * R50: 18, the 15 public packages with no example and the package comments
    of `llmprovider`, `llmprovider/providers` and `wizard`;
  * R51: 78. The MADR's audit counted 76, as it took "concurrent" or
    "goroutine"; the rule's phrase is "concurrent use", which the docs of
    `auth.DeviceLogin` and `llmprovider.CommandToken` lack. Phase 3's list is
    the 78;
  * R52: 27, as the MADR.

  The report on the tree gives the MADR's numbers: 15 packages, 199 exported
  functions and methods, 82 types, 58 returning an error and 27 of them
  naming one, 6 types speaking of concurrency.
* **The gate:** `make pre-add-check` on the four Go files, "4 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)"; `tools/gate.py`, 25
  of 25 passed.

### Phase 2: caches a caller can own (2026-10-19)

* **No deviation.** Phase 1 was committed by the owner as `0e6736a`. A first
  draft named the catalog's state `cacheState`, read it through a `config`
  method, gave `ProbeStore` a `GenerateHealth` method and gave
  `wire.ProbeHealth` the `*cacheref.Value`. Before the tests were planted,
  the code was brought to steps 1, 2 and 4 as written: `store`,
  `processStore` and the field `cfg.store`; `ProbeGenerateHealthCached(ctx,
  s *ProbeStore, …)`; `wire.ProbeHealth(ctx, probes *transport.ProbeStore,
  …)`.
* **Step 1.** `internal/transport/probe.go`: `ProbeStore` (its mutex and
  map), `NewProbeStore`, the process's `defaultProbes`;
  `ProbeGenerateHealthCached` takes the store, nil meaning the process's.
  Its eight callers in tests pass nil. `probeCacheNow` stays the clock.
* **Step 2.** `catalog/cache.go`: the unexported `store` (the listing state
  of `listing_cache.go` and the metadata state of `model_metadata.go`, each
  under its own lock), `newStore`, `processStore`, and `storeIn`, which reads
  a `*cacheref.Value` from the options. `configFor` sets `cfg.store`; the
  listing and metadata functions became methods on `*store`, and their
  callers read `cfg.store`. `defaultConfig` builds through `configFor`, so it
  keeps the process's. `listingNow` and `metadataNow` stay the clocks.
  `resetListingCache` and `resetModelMetadataCache` reset `processStore`;
  one test that builds a `config` literal (`menu_0041_test.go`) names
  `processStore`.
* **Step 3.** `internal/cacheref`: `Value{Catalog any; Probes
  *transport.ProbeStore}`, `From` (the last `*Value` wins) and a nil-safe
  `ProbeStore`. Rows in both package tables; `make tables-check`: "37
  packages, 0 problem(s)". Its own test (`cacheref_0048_test.go`), 100 %:
  without one `make coverage-check` measures it at 0 % against R47's 80 %.
* **Step 4.** `catalog.Cache` (a `sync.Once` and the `*cacheref.Value`;
  zero value ready; safe for concurrent use; not to be copied after first
  use) and `catalog.WithCache`, scoped to `builtinIDs`. The five probing
  providers (`gemini`, `claude`, `grok`, `ollama`, `openai`) keep
  `cacheref.From(st.Values())` in `New` and pass `p.cache.ProbeStore()` to
  `wire.ProbeHealth`. `providers.DefaultWith(opts...)`: `Default()` is
  `DefaultWith()`; each built-in's factory is wrapped to call it with a copy
  of `opts` and then the call's own.
* **Step 5, the tests** (`_0048$`), all passing, also under `-race`:
  * `catalog/cache_0048_test.go`: two zero `Cache` values and the process's
    each fetch Together's listing once; `WithCache(nil)` is the process's; of
    two `WithCache`, the last wins; a provider from `providers.New(Together,
    …, WithCache(&a))` lists into `a`, where `CachedFacts` finds the model,
    and not into `b`; 32 goroutines listing through one `Cache` send one
    request;
  * `internal/transport/probe_store_0048_test.go`: two stores and the
    process's probe separately; `ProbeGenerateHealthCached(nil)` reads the
    process's;
  * `providers/defaultwith_0048_test.go`: an option scoped to no built-in is
    refused by each of the ten built-ins' `New`, so it reached each; a later
    `WithAPIKey` wins (Claude's `x-api-key` on a fake server); a factory the
    caller registers is given no option;
  * beyond the step's list, so that the probe plumbing is tested where it
    runs: `internal/wire/probe_0048_test.go` (`ProbeHealth` probes through
    the store it is given) and `providers/openai/probe_cache_0048_test.go` (a
    provider built `WithCache(&a)` keeps its probes in `a`: 3, 0, then 3
    through `b` and 3 through the process's); and
    `internal/cacheref/cacheref_0048_test.go`.
* **Step 6, red first,** `tools/plant.py`, nine plants, all caught:
  * `configFor` ignoring `WithCache`: "cache_0048_test.go:74: step 2: 1
    listing request(s), want 2";
  * `DefaultWith` dropping `opts`: "defaultwith_0048_test.go:32: gemini: New
    = \<nil\>; want the foreign option refused", and the nine others;
  * `ProbeHealth` ignoring its store: "probe_0048_test.go:29: step 2: 2
    probes in all; want 4";
  * a nil `ProbeStore` not the process's: "probe_store_0048_test.go:26: step
    4: 12 probes in all, want 9";
  * `storeIn` dropping the value it finds: "cache_0048_test.go:99:
    CachedFacts found the model in a cache the provider was not given";
  * OpenAI passing no store, and OpenAI's `New` dropping the cache:
    "probe_cache_0048_test.go:27: step 2: 0 probes; want 3";
  * `From` taking the first `*Value`: "cacheref_0048_test.go:26: case 3";
  * a nil `Value` given a store: "cacheref_0048_test.go:36: nil Value's
    ProbeStore = …; want nil".
* **Not covered:** the probe call sites of Gemini, Claude, Grok and Ollama
  are the same line as OpenAI's, `wire.ProbeHealth(ctx,
  p.cache.ProbeStore(), …)`, but no test lists through those providers with
  a `Cache`; a plant there would not be caught. Their listings and facts are
  covered through `catalog`, which every provider's `p.listing` reaches.
* **Step 7, D7,** each export with the call site that needs it:
  * `catalog.Cache`: a test that keeps its listings apart from every other
    test's, prepare-commit-msg's case (the MADR's C18), here
    `TestCache_KeepsListingsApart_0048`;
  * `catalog.WithCache`: a process that keeps more than one configuration,
    each provider given its own cache;
  * `providers.DefaultWith`: a `Registry` whose providers share one cache,
    `providers.DefaultWith(catalog.WithCache(&c))`, as the README shows.
* **Step 8.** `docs/architecture.md`: the tree, the rows of `providers`,
  `catalog`, `internal/wire`, `internal/transport` and the new
  `internal/cacheref`, the five providers' imports, and a paragraph on whose
  caches a listing reads. R2's table: the `catalog` row (`Cache`, and
  `internal/cacheref` among its imports), the `providers` row
  (`DefaultWith`), the `internal/cacheref` row, and its citation. The
  README: a section "Caches".
* **The API:** `tools/apidiff.py v1.15.0`: "Compatible changes:
  ./llmprovider/providers.DefaultWith: added, ./llmprovider/catalog.Cache:
  added, ./llmprovider/catalog.WithCache: added", and nothing else. No golden
  changed.
* **The gate:** `go test -count=1 ./...`, 34 packages ok; `-race` on
  `catalog`, `providers/...`, `internal/transport` and `internal/wire`, ok;
  `make pre-add-check` on the phase's 31 Go files, in two runs, "30 file(s)
  clean" and, for `cacheref`'s two, "2 file(s) clean (gofmt, golangci-lint,
  go vet, go test, govulncheck)"; `tools/claims.py`, this PLAN 26 of 26 and
  the MADR 24 of 24; `tools/gate.py` on the final tree, 25 of 25 passed,
  `coverage-check` among them.

### Phase 3: stated concurrency and errors (2026-10-19)

* **One deviation**, step 5, chosen by the owner: the missing tests are
  written in this phase. Phase 2 was committed by the owner as `b7523d9`.
* **The baseline,** `sdkaudit -check -rules R51,R52 .`: 105 violations, R51
  78 and R52 27, Phase 1's. After: "sdkaudit: 0 violation(s) of R51,R52",
  exit 0 (step 3).
* **Step 1, R51, 78 types.** Each sentence was read from the code; every
  changed line outside a test file is a comment:
  * 21 string or integer kinds: "It is a string (an integer), safe for
    concurrent use." The step's value-type sentence, "safe only while no
    goroutine changes it", is true of them but says less, as a value of
    those kinds cannot be changed in place;
  * 35 structs of plain values: "It is safe for concurrent use only while no
    goroutine changes it."; and the sealed `Item`'s implementations;
  * the interfaces a caller implements: `TokenSource`, `TokenInvalidator`,
    `Layer`, `TokenStore` and `RefreshLocker` say why an implementation must
    be safe; `InvalidatingSource`, `ModelLister` and `Streamer` state the
    requirement, inherited from `TokenSource` and `Provider`; `Prompter` must
    allow one concurrent use, `Notify` while an `Input` waits; `Factory`
    must be safe, as `Registry.New` does not serialise it;
  * the stateful types, as measured: `FileTokenStore`, `VendorCLISession`,
    `DeviceLogin`, `CommandToken`, `StaticToken`, `Settings`, `Option`,
    `Catalog`, `Metadata` and `Harness` safe, each with its condition;
    `TextPrompter` **not safe**: on a scratch copy of the module, two
    goroutines in `Input` at once gave "WARNING: DATA RACE … Read at … by
    goroutine 9: wizard.(*TextPrompter).bufReader() text_prompter.go:120":
    15 reports with a reader that pauses, and races in 6 of 7 runs with a
    `strings.Reader`; the copy was deleted, and the type is not changed, as
    step 1 says.
* **Step 2, R52, 27 functions.** Each sentence names the kinds its tests
  show: the ten providers' `New` (credential, foreign option, session
  kinds; `ResolveOptions`'s refusals); `GenerateText`; `StaticToken.Token`
  ("never returns an error"); `CommandToken.Token`; `ValidateOllamaURLWith`;
  `Metadata.Encode`; the four logins; `ValidateOAuthSession`;
  `NewFileTokenStore` and `FileTokenStore.Delete`; `ConfigureLLM`; and
  `TextPrompter`'s `Select`, `MultiSelect`, `Confirm` and `Input`.
* **Step 5, the tests** (`_0048$`), 12 new files, all passing under
  `-race`:
  * `llmprovider`: `command_token_0048_test.go` (a caller's and a waiter's
    context, the shared run's error), `concurrency_0048_test.go` (`Factory`
    run by two `New` calls at once, one option list shared by 16 builds, 16
    concurrent readers of one `Settings`);
  * `llmprovider/catalog`: `concurrency_0048_test.go` (catalogs listed at
    once share nothing; one `Metadata` read by 16 goroutines),
    `errors_0048_test.go` (`ValidateOllamaURLWith`, `Encode`),
    `layer_0048_test.go` (two concurrent Claude requests call one `Layer`
    at once);
  * `llmprovider/auth`: `kinds_helpers_0048_test.go`,
    `device_kinds_0048_test.go`, `browser_kinds_0048_test.go`,
    `session_store_kinds_0048_test.go`, `vendor_session_race_0048_test.go`
    (16 goroutines on one `VendorCLISession`);
  * `wizard`: `configure_errors_0048_test.go` (a Prompter's error wrapped
    and as it is, a failed device-code sign-in's `*url.Error`, the plain
    errors, an empty `Registry`), `text_prompter_errors_0048_test.go`.

  `Prompter`'s concurrent use is shown by the existing
  `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt`: its `Input`
  returns only once a `Notify` came while it waited. A planted break of it
  hangs until the test timeout rather than failing an assertion.
* **Not named, for want of a test:** `ConfigureLLM`'s browser and ChatGPT
  plan sign-ins return their errors as the device-code one does, but only
  the device-code one is tested, so only it is named; `LoginBrowserOAuth`'s
  own ten-minute bound takes the same branch as a caller's deadline, which
  is tested, so the doc names the caller's only; its `crypto/rand` failures
  cannot be reached.
* **Red first,** `tools/plant.py`, 92 plants in six specs, all caught, each
  by a failing test, none by a build failure:
  * `llmprovider` and `catalog`, 13, under `-race`: for example
    "concurrency_0048_test.go:50: the two New calls never ran the factory at
    once", "layer_0048_test.go:70: two concurrent requests never called the
    layer at once (2 calls)";
  * `auth`, 50, and 1 under `-race` (the lock around `VendorCLISession`'s
    account write removed);
  * `wizard`, 25, and 1 for the sign-in prompt (`drain` waiting for the
    read);
  * step 4, the check itself, 2: one R51 sentence removed gives
    "llmprovider/auth/oauth_device.go:81 R51 type DeviceLogin does not say
    whether it is safe for concurrent use", one R52 sentence removed
    "llmprovider/token.go:117 R52 StaticToken.Token returns an error its doc
    does not name", each exit 1.
* **Found, left for a record of its own,** as step 5 lists: the error kinds
  that differ between like paths, and the two panics.
* **The API:** `tools/apidiff.py v1.15.0` lists only Phase 2's three
  additions. No golden changed.
* **The gate:** `go vet ./...`, also with `GOOS=windows`, clean; `go test
  -count=1 ./...`, 35 packages ok; `-race` on `llmprovider/...` and
  `wizard/...`, ok; `make pre-add-check` on the phase's 59 Go files, "59
  file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)";
  `tools/gate.py` on the final tree, 25 of 25 passed.
