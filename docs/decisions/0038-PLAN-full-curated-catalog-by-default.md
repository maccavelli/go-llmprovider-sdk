---
status: in-progress
date: 2026-10-13
associated-madr: "0038-MADR-full-curated-catalog-by-default.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Full Curated Catalog by Default, and Six Only on Request

Associated MADR: [0038-MADR-full-curated-catalog-by-default.md](0038-MADR-full-curated-catalog-by-default.md)

## Goal

When it is done:

* **The default is the full list.** A live listing's `Recommended` is the
  full curated catalog (0038-MADR D1, built as its amendment A1 says), and
  so is every built-in provider's `ListModels` (D4).
* **The six is an option.** `catalog.WithLimit(n)` cuts every list to `n`.
  `WithLimit(catalog.MaxListed)` returns today's six, apart from D1's
  fallback exclusion (D2).
* **The wizard follows the same rule.** Its menu is the full list by
  default, and `Options.ModelLimit` cuts it (D5, A2).
* **Probes stay at six.** At most `MaxListed` billed probes are sent per
  listing, and the unprobed tail is kept (D4).
* **The docs say so:**
  * the `MaxListed` doc (D6);
  * the docs of `Catalog`, `List`, `WithModelProbes` and `ListModels`;
  * `docs/architecture.md`.
* **A live run confirms it.** For every provider with a key, and the public
  listings, a live listing shows the six as the full list's prefix, with no
  id measured not to answer.
* **`v1.11.0` is released.** `tools/apidiff.py` reports exactly two
  additions: `catalog.WithLimit` and `wizard.Options.ModelLimit` (D7, A2).

## Scope

### In scope

* **New package:** `llmprovider/internal/listlimit` (A3). It holds the
  option's value type, how it is read from resolved options, and the cut.
* **`llmprovider/catalog`:**
  * `config.go`: `WithLimit`, and `config` reads the limit.
  * `model_ranking.go`: `rankRecommended` takes a cap, where zero means
    none.
  * `models_catalog.go`:
    * `curateFromCatalog` takes the provider and a cap, and leaves out
      `excludedAtRequest` ids (D1);
    * the `MaxListed` doc.
  * `discovery.go`:
    * the curate functions take a cap;
    * `catalogFrom`, `staticCatalog` and `ollamaCatalog` build the full
      list (A1) and apply the limit;
    * the `Catalog` and `List` docs.
  * `model_metadata.go`: `metadataCurate` passes the cap on.
* **The probes:** `llmprovider/internal/transport/probe.go` (the probed
  head plus the unprobed tail, and the cache) and its stale
  `llmprovider.MaxListedModels` comment.
* **The providers:** the `New` and `ListModels` of every built-in provider:
  * `gemini`, `openai` (its API-key and ChatGPT listings), `claude`, `grok`
    and `ollama`, which probe;
  * `opencode` (Zen and Go), `huggingface`, `kilo` and `together`, which do
    not.
* **`llmprovider/options.go`:** the `WithModelProbes` doc, which names a
  `MaxListedModels` that does not exist.
* **`wizard/configure.go`:** the `Options.ModelLimit` field, and the cut on
  every listing path.
* **Tests:**
  * new tests, each named in its phase;
  * existing tests whose meaning is "the six", which take the limit (R46);
  * the G-wire listing goldens whose result lists change, regenerated
    alone, each diff recorded.
* **A live measurement:** `llmprovider/catalog/live_0038_full_catalog_measure_test.go`.
  It sends listing requests only, with probes off.
* **Records and docs:**
  * `docs/architecture.md`, at the "A listing" paragraph and the
    "Discovery and ranking" bullets;
  * `README.md`, its Status and the listing paragraph near line 124;
  * this PLAN's release notes;
  * `docs/README.md`;
  * `scripts/coverage-floors.txt`, if `make coverage-check` asks for a line
    for the new package (R47).

### Out of scope

* **The static lists.** Their content and their bound stay as they are
  (D3).
* **The ranker's rules,** eligibility and `maxPerRankGroup` included.
* **`Usable` and `Search`.**
* **Any change to `Catalog`'s fields,** the `ModelLister` interface, or
  `Descriptor`.
* **gobble-cli's and prepare-commit-msg's adoption** (D7). Each needs its own
  record in its own repository.
* **`llmprovider/internal/cmd/catalogs`.** It already cuts `Recommended` to
  `MaxListed` (`main.go:259`), and by A1 the full list's first six are the
  same ids. Its tests are run in every phase, unchanged.

## Rules for every phase

* **Research first.** Each phase starts by reading the code it changes, at
  the commit it starts from. A fact that differs from this PLAN or the
  MADR is a deviation.
* **Red first,** with every FAIL line recorded.
* **Plants** go through `tools/plant.py`, on a scratch clone, never in the
  tree. Each plant's FAIL line is recorded.
* **Every script is Python.** A reusable one belongs in `tools/`. Specs and
  scripts are written with the Write tool, and exact edits go through
  `tools/edit.py`.
* **A phase is staged only after:**
  * `make pre-add-check FILES="…"` on every changed Go file;
  * `make records-check`;
  * `tools/denyscan.py --changed` and `tools/linkcheck.py`;
  * markdownlint on any changed non-record Markdown;
  * the gate, `python3 -B tools/gate.py --out <scratch>`, at 23 of 23.
* **Billed requests:** the count goes to the owner, and the owner approves,
  before any live run. This PLAN plans none: Phase 4 lists only, with probes
  off. Keys are checked for presence only.
* **A deviation stops the work:** evidence, options with the recommended
  one first, and a dated entry here, plus a MADR amendment where a fact or
  a decision changes.
* **The owner commits** each phase, pushes and tags. If a phase is still
  staged, the next one stays unstaged.

## Implementation Steps

### Phase 0: records

1. 0038-MADR is `accepted` (the owner, 2026-10-13), with its amendment
   "what the PLAN's research found". This PLAN is `in-progress`, approved
   by the owner on 2026-10-13.
2. `docs/README.md`: the PLAN's row and the count.
3. `make records-check`, `tools/linkcheck.py` and `tools/denyscan.py
   --changed`. Stage. The owner commits.

### Phase 1: the probes keep the unprobed tail

No list today is longer than six, so this phase changes no result. It
prepares D4 for Phase 3.

1. **Research.**
   * Read `transport.ProbeGenerateHealth` and `ProbeGenerateHealthCached`
     (`probe.go:17-136`), `wire.ProbeHealth` (`internal/wire/probe.go`),
     and each probing provider's `ListModels`.
   * Confirm every caller passes `catalog.MaxListed`.
   * Confirm the "every probe failed, so return the list" fallback lives in
     the providers (for example `openai.go:363-366`).
2. **Red first.** In `llmprovider/internal/transport/probe_tail_0038_test.go`:

   | Test | Input | Want |
   | :--- | :--- | :--- |
   | `TestProbe_KeepsTheTail` | 9 ids, limit 6, the 2nd fails | 6 probes; ids 1 and 3–9, in order |
   | `TestProbe_AllHeadFail` | 9 ids, limit 6, every probe fails | `nil`, so the provider returns its list unchanged, as today |
   | `TestProbe_ShortList` | 4 ids, limit 6 | 4 probes; the healthy ones, as today |
   | `TestProbeCached_KeepsTheTail` | the first test twice, same key | 6 probes in all; the second call returns the cached head plus the current tail |

   `TestListModels_ProbesAtMostMaxListedModels` in
   `providers/ollama/listing_test.go` gains an assertion: with
   `MaxListed+3` installed models, all answering, `ListModels` returns all
   `MaxListed+3`, in order. Each test fails before the change.
3. **Change** (`transport/probe.go`):
   * `ProbeGenerateHealth` probes `preferred[:min(limit, len)]`, as today.
     It returns the healthy head followed by `preferred[limit:]`, and `nil`
     when no head probe answered.
   * `ProbeGenerateHealthCached` caches the head's healthy ids only, and
     appends the current tail to a cached result.
   * The doc comments say this, and name `catalog.MaxListed`, not the
     missing `llmprovider.MaxListedModels`.
4. **`llmprovider/options.go`:** `WithModelProbes`'s doc says "to each of
   the first `catalog.MaxListed` listed models", and that the rest are
   returned unprobed.
5. **Plants:**
   * the tail dropped;
   * the limit ignored, so every id is probed;
   * an all-failed head returning the tail;
   * the cached path dropping the tail.
6. **Green:** `go test ./...`. The gate. Stage. The owner commits.

### Phase 2: `catalog.WithLimit` and the full list, with the default still six

This phase adds the whole mechanism. Without the option, every list stays
as it is today, so no existing test changes meaning. `WithLimit(0)` is the
full list in this phase only. Phase 3 makes it the default.

1. **Research.**
   * Read every curate function and its callers: `curateGemini`,
     `curateOpenAI`, `curateClaude`, `curateGrok`, `curateTogether`,
     `curateHuggingFace`, `curateKilo`, `kiloCurate`, `metadataCurate`, and
     the OpenCode one (`discovery.go:648`).
   * Read `catalogFrom`, `staticCatalog`, `ollamaCatalog`, and how each
     provider's `New` reads `Settings.Values()` (for example
     `kilo.go:170`).
   * Confirm that `ResolveOptions` accepts an option scoped to
     `builtinIDs` in every built-in `New`, and refuses it in a third
     party's (`settings.go:138-145`).
   * **Measure on a scratch clone, not the tree:** replay the 2026-10-12
     snapshot with a throwaway uncapped ranking. Record each ranked list's
     full length under both profiles, and confirm that no
     `excludedAtRequest` id appears. Record the numbers here.
2. **`llmprovider/internal/listlimit`:**
   * `type Value int`, the option's value.
   * `Of(values []any) (n int, set bool)`: the last `Value` in `values`,
     since options apply in order.
   * `Cut(ids []string, n int) []string`: a copy of `ids`, cut to `n` when
     `0 < n < len(ids)`.
   * Its test covers each branch. The package holds at least 80 % (R47).
3. **`catalog.WithLimit(n int) llmprovider.Option`** (`config.go`):
   * It is `llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithLimit",
     listlimit.Value(n))`.
   * `config` gains `limit int` and `limitSet bool`, read in `configFor`.
   * Its doc states D2: `n ≤ 0` is the full list, and the option is scoped
     to every built-in provider.
4. **The cap** (A1):
   * `rankRecommended(profile, provider, cands, fill, max)`, where
     `max ≤ 0` means no cap.
   * `curateFromCatalog(provider, catalog, available, usable, rankFn, max)`.
     It also skips `excludedAtRequest(provider, id)` (D1).
   * Each curate function takes `max`, and passes it to its fallback, so an
     uncapped ranking fills from an uncapped fallback.
5. **The full list** (A1), in one function, `fullList(usable, curate)`. It
   returns the six, `curate(usable, MaxListed)`, followed by
   `curate(usable, 0)` without the six's ids. `catalogFrom` uses it:
   * **With no option** (this phase), `Recommended` is the six.
   * **With `WithLimit(n)`,** it is `listlimit.Cut(fullList, n)`, and the
     full list when `n ≤ 0`.
   * `staticCatalog` and `ollamaCatalog` apply the same cut. `Usable` is not
     cut.
6. **The providers** (A3). Each built-in `New` reads `listlimit.Of`, and
   `ListModels` cuts its `catalog.Static` fallback to the limit. The
   ChatGPT listing is cut only when the limit is set, because today it is
   not cut at all. The probe limit stays `catalog.MaxListed`.
7. **Red first,** each test failing before the change:

   | Test (file) | Want |
   | :--- | :--- |
   | `TestWithLimit_SixIsToday` (`catalog/full_catalog_0038_test.go`) | for an open catalog (the 2026-10-12 snapshot, both profiles), a vendor (OpenAI over an `httptest` listing) and Ollama: `WithLimit(MaxListed)` equals no option |
   | `TestWithLimit_FullList` (same) | `WithLimit(0)`: the first six equal the six. The ids are `Usable` minus exactly D1's exclusions, with no id twice. Each snapshot length is as measured in step 1 |
   | `TestWithLimit_Cuts` (same) | `WithLimit(3)` is the full list's first three; `WithLimit(-1)` equals `WithLimit(0)` |
   | `TestWithLimit_FallbackExcludes` (same) | with `WithoutModelMetadata`, OpenCode Go's listing with a region-gated id, and Together's with `MiniMaxAI/MiniMax-M2.7`, recommend neither, cut or not |
   | `TestWithLimit_StaticFallback` (same) | a failed listing with `WithLimit(2)` returns the static list's first two, with `Live` false |
   | `TestWithLimit_Scope` (same) | every built-in `New` accepts `catalog.WithLimit`, and a registry's third-party `New` refuses it with `ErrInvalidRequest` |
   | `TestListModels_WithLimit` (`providers/hygiene_test.go`, or a new `providers/limit_0038_test.go`) | for each built-in provider, with probes off: `WithLimit(2)` returns at most two, its static fallback included, and `WithLimit(0)` returns at least the listing's six |
   | `TestListModels_FullListProbesSix` (`providers/openai/listing_test.go`) | with probes on and nine served ids, nine come back and at most six probes are sent |

8. **The fallback exclusion** (D1) can change an existing test's six where
   a fixture backfills an unservable id. Each such test gets a changed
   expectation, its diff recorded here. Any other failure of an existing
   test is a deviation.
9. **Plants:**
   * the six not first (`fullList` returning the uncapped ranking alone);
   * `curateFromCatalog` without the exclusion;
   * `WithLimit` not read by one provider's fallback;
   * `n < 0` read as a cut;
   * the option scoped to one provider only.
10. **Green:** `go test ./...`, with `TestStatic_EqualsLatestSnapshot` and
    `TestListModelCatalog_Snapshot20261012` unchanged. `make api-check` shows the
    addition of `WithLimit`. The gate. Stage. The owner commits.

### Phase 3: the default becomes the full list; the wizard follows

1. **Research.**
   * Rerun `go test ./...` on a scratch clone with only the default flipped
     (`limitSet` false reads as no cut), and record every failure.
   * Sort each failing test as one of:
     * (a) its meaning is "the six": it takes `catalog.WithLimit(catalog.MaxListed)`
       or `ModelLimit: catalog.MaxListed` (R46), and the listing tests in
       `discovery_catalog_test.go:153`, `:215` are examples;
     * (b) a G-wire listing golden whose result list grows;
     * (c) anything else, which is a deviation.
   * Read `wizard/configure.go:380-512`, `model_select.go:40-100` and their
     tests.
2. **Change:**
   * `catalog`: with no option, `Recommended` is the full list (D1).
   * `listlimit.Of`: unset means no cut everywhere, the ChatGPT listing
     included.
   * `wizard.Options.ModelLimit int`. Its doc: "the most models the menu
     recommends; zero recommends the whole curated list (0038-MADR D5,
     A2)". `discoverModels` passes `catalog.WithLimit(o.ModelLimit)` beside
     `WithProfile`. `chatGPTCatalog`, `providerCatalog` and the static
     fallback cut to `ModelLimit` when it is positive, in place of
     `catalog.MaxListed` (`configure.go:483`, `:511`, `:395`).
3. **Red first:**

   | Test (file) | Want |
   | :--- | :--- |
   | `TestList_DefaultIsFullList` (`catalog/full_catalog_0038_test.go`) | no option equals `WithLimit(0)`, on all three paths |
   | `TestWizard_MenuIsFullListByDefault` (`wizard/model_limit_0038_test.go`) | with `MaxListed+2` listed ids, for a built-in listing, a ChatGPT session and a third-party `ListModels`, the menu shows them all |
   | `TestWizard_ModelLimit` (same) | `ModelLimit: catalog.MaxListed` shows six on each path, and on the static fallback |
   | `TestWizard_ProviderOptionsLimitRefused` (same) | documents A2: `catalog.WithLimit` in `ProviderOptions`, for a third party, is the `New` error the wizard reports, not a silent cut |

   Then: the category-(a) tests take the limit, each listed here; and the
   category-(b) goldens are regenerated alone, with `-update -run
   TestWireGoldens/<provider>/listing`. Each diff is recorded here, and
   only the result list changes.
4. **Docs:**
   * `MaxListed` (D6);
   * `Catalog.Recommended`, `List`, and `ModelLister.ListModels` in
     `contract.go`, whose doc says "the curated listing";
   * each built-in `ListModels` doc that says "a short, curated list" (for
     example `gemini.go:284`);
   * `docs/architecture.md`, at the "A listing" paragraph and the
     "Discovery and ranking" bullets;
   * `README.md`'s listing paragraph.
5. **Plants:**
   * the default left at six;
   * the wizard ignoring `ModelLimit` on the third-party path;
   * the wizard cutting at `MaxListed` when `ModelLimit` is zero.
6. **Green:** `go test ./...`, with `internal/cmd/catalogs`'s tests
   unchanged. `make api-check` shows both additions. The gate. Stage. The
   owner commits.

### Phase 4: live confirmation and release

1. **The measurement,** `live_0038_full_catalog_measure_test.go`
   (`live_gateways`; listings only, `WithModelProbes(false)`, not billed).
   It covers every provider whose key is set (checked for presence only),
   and the public listings of OpenCode Zen, OpenCode Go, Kilo and Hugging
   Face. For each:
   * `List` with no option, and with `WithLimit(MaxListed)`;
   * the six is the full list's prefix;
   * no `excludedAtRequest` id appears;
   * the ids are a subset of `Usable`;
   * the lengths are logged.

   Each provider's lengths and its six are recorded here. A provider whose
   key is missing is recorded as skipped.
2. **`README.md` Status:** `v1.11.0`, with a bullet that names the default
   change first, then the opt-in.
3. **Release notes** in this PLAN:
   * the change and the opt-in, for `catalog.List`, `ListModels` and the
     wizard;
   * the probe rule;
   * D1's fallback exclusion;
   * the consumers' follow-up (D7).
4. **The checks** before the owner pushes: `make lint vuln parity-check
   dep-check coverage-check api-check generate-check records-check
   gate-selftest`, and the gate. The disclosure guard runs when the owner
   asks.
5. **The owner** commits, pushes, and tags `v1.11.0`.
6. **After the tag:**
   * CI on `main` and on the tag;
   * the proxy reports `v1.11.0`;
   * `tools/apidiff.py v1.10.2 v1.11.0` reports exactly the two additions;
   * `tools/consumer.py v1.11.0` and `tools/offline.py v1.11.0` pass.
7. This PLAN becomes `complete`, and `docs/README.md` says so.

## Verification

| | Acceptance criterion | Shown by |
| :--- | :--- | :--- |
| V1 | `WithLimit(MaxListed)` is today's six, apart from D1's fallback exclusion | `TestWithLimit_SixIsToday`; the snapshot tests unchanged |
| V2 | with no option, the full list: `Usable` minus D1's exclusions, with the six first | `TestWithLimit_FullList`, `TestList_DefaultIsFullList`; Phase 4's live run |
| V3 | no recommended id is measured not to answer, on any path | `TestWithLimit_FallbackExcludes`; Phase 4 |
| V4 | at most `MaxListed` probes per listing, with the tail kept | Phase 1's tests; `TestListModels_FullListProbesSix` |
| V5 | the wizard shows the full list by default and `ModelLimit` cuts it, on every path | Phase 3's wizard tests |
| V6 | API: exactly `catalog.WithLimit` and `wizard.Options.ModelLimit` added | `make api-check`; `tools/apidiff.py v1.10.2 v1.11.0` |
| V7 | every new assertion has failed on a plant | each phase's plant table |

## Rollout and Rollback

* **Rollout:**
  * `v1.11.0`, a minor release: two additions and a changed default (D7).
  * The release notes give the one-line opt-in that keeps today's six.
  * The two consumers adopt it in their own repositories:
    * prepare-commit-msg with `WithLimit(MaxListed)` and
      `ModelLimit: catalog.MaxListed`;
    * gobble-cli by listing through `catalog`.
* **Rollback:**
  * A caller that passes the opt-in is unaffected by either direction.
  * Restoring the six as the default is a patch release that changes the
    default in `listlimit.Of` back to `MaxListed`. `WithLimit(0)` still
    gives the full list, so no exported identifier is removed and R48
    holds.
  * Phases 1 and 2 change no default and need no rollback of their own.

## Execution Record

### Phase 0: records (2026-10-13)

* **The MADR:** the owner accepted 0038-MADR ("accept the madr, write the
  plan"), with its amendment "what the PLAN's research found" (A1–A3).
* **This PLAN:** the owner approved it ("accepted, proceed"), so its status
  is `in-progress`.
* **The index:** `docs/README.md` has both rows, the count at 82, and an "I
  want to…" row.
* **Checks:**
  * `make records-check`: 82 records, 0 problems;
  * `tools/linkcheck.py`: exit 0;
  * `tools/denyscan.py --changed`: 0 hits.
