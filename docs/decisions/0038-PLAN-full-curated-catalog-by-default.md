---
status: complete
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
  before any live run. ~~This PLAN plans none: Phase 4 lists only, with
  probes off.~~ *Phase 4 lists only, with probes off; its two Together
  listings are billed and owner-approved (Phase 4's deviation 1 of
  2026-10-13).* Keys are checked for presence only.
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

   ~~`TestListModels_ProbesAtMostMaxListedModels` in
   `providers/ollama/listing_test.go` gains an assertion: with
   `MaxListed+3` installed models, all answering, `ListModels` returns all
   `MaxListed+3`, in order.~~ *(Moved to Phase 2, step 7: deviation 2 of
   2026-10-13.)* Each test fails before the change.

   *(Deviation 1 of 2026-10-13.)* `TestProbeGenerateHealth_Limit`
   (`transport/probe_test.go:39-47`, existing) wants `[a b c]`, the probed
   head and the unprobed tail, in place of `[a b]`. It also counts exactly
   two probes, so it asserts its stated meaning, "no more than limit
   candidates are probed, in order", directly.
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
   | `TestListModels_ProbesAtMostMaxListedModels` (`providers/ollama/listing_test.go`, existing; deviation 2 of 2026-10-13) | with `catalog.WithLimit(0)` and `MaxListed+3` installed models, all answering, `ListModels` returns all `MaxListed+3`, in order, after exactly `MaxListed` probes |

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
   (`live_gateways`; listings only, `WithModelProbes(false)`, ~~not billed~~;
   *two billed Together listings, owner-approved: Phase 4's deviation 1 of
   2026-10-13*).
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

The owner committed Phase 0 as `2ef6f25`.

### Phase 1: the probes keep the unprobed tail (2026-10-13)

**Research** confirmed the PLAN's facts:

* All five probing providers pass `catalog.MaxListed` to `wire.ProbeHealth`:
  `gemini.go:298`, `claude.go:331`, `grok.go:288`, `ollama.go:245` and
  `openai.go:354`.
* Each returns `listed` when no probe answered: `gemini.go:310-313`,
  `claude.go:341-344`, `grok.go:300-303`, `ollama.go:257-260` and
  `openai.go:364-367`.
* `wire.ProbeHealth` calls the cached probe when the token resolves, and
  the uncached one otherwise (`internal/wire/probe.go:22-27`).

It also found two facts the PLAN had wrong.

#### Deviations

**Deviation 1 (2026-10-13): an existing test asserts the old result.**

* **Found:** `TestProbeGenerateHealth_Limit`
  (`llmprovider/internal/transport/probe_test.go:39-47`) probes `[a b c]`
  with limit 2. It requires the result `[a b]`. Under 0038-MADR D4 the
  result is `[a b c]`. The PLAN said this phase changes no existing test.
  No other test probes more ids than its limit.
* **Decision (the owner):** the test follows D4. It wants `[a b c]` and
  counts exactly two probes, so it asserts its stated meaning directly.
  The MADR does not change.

**Deviation 2 (2026-10-13): the planned Ollama assertion cannot hold in this
phase.**

* **Found:** `ollamaCatalog` recommends the first `MaxListed` installed
  models (`llmprovider/catalog/discovery.go:466`) until Phase 2. So
  `ListModels` sees six ids, and "returns all `MaxListed+3`" fails
  whatever the probe does.
* **Decision (the owner):** the assertion moves to Phase 2, step 7, with
  `catalog.WithLimit(0)`. This phase's transport tests cover the tail
  directly.

#### Tests, change and checks

**Tests** (`llmprovider/internal/transport/probe_tail_0038_test.go`, new):

* `TestProbe_KeepsTheTail`;
* `TestProbe_AllHeadFail`;
* `TestProbe_ShortList`;
* `TestProbeCached_KeepsTheTail`.

`TestProbeGenerateHealth_Limit` follows D4 (deviation 1).

**Red first:**

```text
--- FAIL: TestProbe_KeepsTheTail (0.00s)
    probe_tail_0038_test.go:37: ProbeGenerateHealth = [m1 m3 m4 m5 m6] after 6 probes; want [m1 m3 m4 m5 m6 m7 m8 m9] after 6
--- FAIL: TestProbeCached_KeepsTheTail (0.00s)
    probe_tail_0038_test.go:69: listing 1: [m1 m3 m4 m5 m6]; want [m1 m3 m4 m5 m6 m7 m8 m9]
    probe_tail_0038_test.go:69: listing 2: [m1 m3 m4 m5 m6]; want [m1 m3 m4 m5 m6 m7 m8 m9]
--- FAIL: TestProbeGenerateHealth_Limit (0.00s)
    probe_test.go:45: ProbeGenerateHealth = [a b] after 2 probes, want [a b c] after 2
```

`TestProbe_AllHeadFail` and `TestProbe_ShortList` passed before the change,
as they must: they hold today's behavior. The plants show
`TestProbe_AllHeadFail` failing.

**Change:**

* **`llmprovider/internal/transport/probe.go`:**
  * `ProbeGenerateHealth` is `withTail(probeHead(…), preferred, limit)`.
  * `probeHead` is the old body: it probes the first `limit` ids.
  * `withTail` appends `preferred[limit:]`, unprobed, to a non-empty head,
    skipping an id the head already holds, case-insensitively, as the
    probe's own match does. An empty head stays `nil`, so each provider
    still returns its listing when no probe answered.
  * `ProbeGenerateHealthCached` caches `probeHead`'s result, and appends
    the current tail to a fresh or cached head.
  * The comment names `catalog.MaxListed`, not the missing
    `llmprovider.MaxListedModels`.
* **`llmprovider/options.go`:** the `WithModelProbes` doc says the first
  `catalog.MaxListed` are probed, and the later ones returned unprobed.

No list a provider passes today is longer than six, so no result changes.

**Green:** `go test ./llmprovider/internal/transport/
./llmprovider/internal/wire/ ./llmprovider/providers/...`, every package
`ok`.

**Plants** (`tools/plant.py`, 4 of 4 caught):

| Plant | FAIL line |
| :--- | :--- |
| the tail dropped | `TestProbe_KeepsTheTail`: `ProbeGenerateHealth = [m1 m3 m4 m5 m6] after 6 probes; want [m1 m3 m4 m5 m6 m7 m8 m9] after 6`; also `TestProbeCached_KeepsTheTail` and `TestProbeGenerateHealth_Limit` |
| the limit ignored, so every id is probed | `TestProbe_KeepsTheTail`: `… after 9 probes; want … after 6`; `TestProbe_AllHeadFail`: `ProbeGenerateHealth = [] after 9 probes; want nil after 6`; `TestProbeCached_KeepsTheTail`: `9 probes in all; want 6`; `TestProbeGenerateHealth_Limit`: `… after 3 probes, want [a b c] after 2` |
| an all-failed head returning the tail | `TestProbe_AllHeadFail`: `ProbeGenerateHealth = [m7 m8 m9] after 6 probes; want nil after 6` |
| the cached path dropping the tail | `TestProbeCached_KeepsTheTail`: `listing 2: [m1 m3 m4 m5 m6]; want [m1 m3 m4 m5 m6 m7 m8 m9]` |

**Checks:** `make pre-add-check` on the four Go files: clean (gofmt,
golangci-lint, go vet, go test, govulncheck).

**Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 1 as `df90f3d`.

### Phase 2: `catalog.WithLimit` and the full list, with the default still six (2026-10-13)

**Research** confirmed the PLAN's facts, and found what follows.

* **Built-in providers take the option; others refuse it.** Every built-in
  `New` resolves an option scoped to `builtinIDs`. A third party's refuses
  it (`settings.go:138-145`).
* **Where the providers read their values.** `gemini`, `grok`, `kilo`,
  `openai` and `opencode` already read `Settings.Values()` in `New`.
  `claude`, `huggingface` and `together` did not.
* **Ollama has no static fallback.** Its `ListModels` returns the listing's
  error (`ollama.go:235-240`), so it reads no limit of its own.
* **OpenCode already excluded unservable ids.** Its fallback curation left
  out `excludedAtRequest` ids itself (0021-MADR D3).
* **The metadata is received once.** `metadataCurate` receives the metadata
  document from a channel, once (`res := <-meta`). The full list runs the
  curation twice, the six and the uncapped one, so the closure now receives
  it through `sync.OnceValue`.

**The measurement** was a throwaway test, run on a scratch clone at
`df90f3d`, not in the tree. It replayed the 2026-10-12 snapshot and counted,
for each list and profile, `Usable` and D1's exclusions. It used the
existing predicates (`excludedAtRequest`, `utilityExcluded`, `listedOnly`
against the snapshot's document), with no ranking code:

| List | Utility: usable → full | Capable: usable → full | Left out |
| :--- | ---: | ---: | :--- |
| Hugging Face | 119 → 102 | 119 → 119 | 17 dense ≤ 14B (utility) |
| Kilo | 285 → 272 | 285 → 285 | 13 (utility) |
| OpenCode Go | 34 → 28 | 34 → 28 | the four region-gated DeepSeek ids, two `-contributor` |
| OpenCode Zen | 79 → 77 | 79 → 77 | two `-contributor-free` |
| Together | 172 → 19 | 172 → 21 | `MiniMaxAI/MiniMax-M2.7`; ids models.dev does not list; utility's exclusions |

No `excludedAtRequest` id is in any list's six.

**Step A, the option, with no behavior yet:**

* `llmprovider/internal/listlimit`: `Value`, `Of` and `Cut`, with
  `listlimit_test.go` (coverage 100 %).
* `catalog.WithLimit` and `config`'s `limit` and `limitSet` (`config.go`).

**Red first,** with the option defined and ignored:

```text
--- FAIL: TestWithLimit_FullList
    full_catalog_0038_test.go:112: opencode-zen profile 0: the full list holds 6 ids; want the 77 usable ids less D1's exclusions
    full_catalog_0038_test.go:115: opencode-zen profile 0: 6 ids; want 77, as measured
    (the same two lines for each ranked list and profile)
    full_catalog_0038_test.go:122: openai: the full list [gpt-6-luna gpt-5.6-luna gpt-4.1-mini gpt-5.6-terra gpt-6.1-sol gpt-6-astra]; want the six […] first, then the rest of Usable […, gpt-5.6-sol gpt-6-sol gpt-4.1]
    full_catalog_0038_test.go:125: ollama: the full list [m1 m2 m3 m4 m5 m6]; want every installed model [m1 m2 m3 m4 m5 m6 m7 m8]
--- FAIL: TestWithLimit_Cuts
    full_catalog_0038_test.go:135: WithLimit(0) holds 6 ids; want Kilo's full list, longer than nine
--- FAIL: TestWithLimit_FallbackExcludes
    full_catalog_0038_test.go:162: together (0 options): Recommended [MiniMaxAI/MiniMax-M2.7 t/a t/b], Usable [MiniMaxAI/MiniMax-M2.7 t/a t/b]; want MiniMaxAI/MiniMax-M2.7 searchable but not recommended
--- FAIL: TestWithLimit_StaticFallback
    full_catalog_0038_test.go:175: Live false, Recommended [gpt-6-luna … gpt-6-astra], Usable […]; want the static catalog's first two in both, not live
--- FAIL: TestListModels_WithLimit/gemini (and openai, claude, grok, opencode-zen, opencode-go, huggingface, kilo, together)
    limit_0038_test.go:64: 1 options: ListModels = [gemini-3.5-flash-lite gemini-3.8-flash gemini-3.6-flash]; want [gemini-3.5-flash-lite gemini-3.8-flash]
--- FAIL: TestListModels_FullListProbesSix
    limit_0038_test.go:43: ListModels = 6 ids after 6 probes; want 9 after 6: […]
--- FAIL: TestListModels_ChatGPTWithLimit
    limit_0038_test.go:69: 1 options: ListModels = [gpt-6-astra gpt-5.6-sol gpt-5.6-luna]/<nil>; want [gpt-6-astra gpt-5.6-sol]
--- FAIL: TestListModels_ProbesAtMostMaxListedModels (ollama)
    listing_test.go:89: ListModels = [model-0:latest … model-5:latest]; want every installed model [model-0:latest … model-8:latest]
```

Three tests passed before the change, as they must: they guard today's
behavior or the scope. They are `TestWithLimit_SixIsToday`, the OpenCode Go
half of `TestWithLimit_FallbackExcludes`, and `TestWithLimit_Scope`. The
plants show each failing.

The first red run of `TestWithLimit_Cuts` panicked, slicing the six at nine.
The test now fails with a message when the full list is not longer than
nine.

**Change:**

* **`models_catalog.go`:**
  * `curateFromCatalog(provider, catalog, available, usable, rankFn, limit)`.
    `limit` zero or less backfills every usable id. An `excludedAtRequest`
    id is never chosen (D1).
  * The `MaxListed` doc (D6).
* **`model_ranking.go`:** `rankRecommended(…, fill, limit)`, where `limit`
  zero or less is no cap.
* **`discovery.go`:**
  * `type curator func(usable []string, limit int) []string`, and every
    curate function takes `limit` and passes it to its fallback.
  * `fullList` is the six, `curate(usable, MaxListed)`, then
    `curate(usable, 0)`'s other ids (A1).
  * `recommend` cuts the six when the limit is within six. Otherwise it cuts
    the full list.
  * `catalogFrom`, `staticCatalog` and `ollamaCatalog` take the listing's
    limit. A static fallback is cut in both views, which stay equal (D3).
  * OpenCode's curation passes `isUsableOpencodeModel` to
    `curateFromCatalog`, which now applies the exclusion its own predicate
    did.
* **`model_metadata.go`:** `metadataCurate` receives the document once
  (`sync.OnceValue`) and passes `limit` on.
* **`config.go`:**
  * `recommendLimit()`: `MaxListed` without the option (until Phase 3),
    else `WithLimit`'s n.
  * `WithLimit`'s doc says a static fallback's `Usable` is cut with it.
* **The providers** (`claude`, `gemini`, `grok`, `huggingface`, `kilo`,
  `openai`, `opencode`, `together`):
  * Each `New` sets `p.limit` from `listlimit.Of(st.Values())`.
  * `ListModels` cuts its `catalog.Static` fallback with `listlimit.Cut`.
  * OpenAI also cuts a ChatGPT session's listing. `listlimit.Cut` with
    zero is no cut, so a listing without the option is unchanged.

**A bug found on the way to green:** `rankRecommended` broke at once when
`limit` was zero, since `len(out) == limit` held for an empty result.
`TestRankRecommended_UncappedFillDiffers` failed with
`uncapped = []; want [s/one s/two r/first r/second]`, and every open
catalog's full list was six long. Both loops now test `limit > 0 &&`.

**Tests added:**

| Test | File |
| :--- | :--- |
| `TestWithLimit_SixIsToday`, `_FullList`, `_Cuts`, `_FallbackExcludes`, `_StaticFallback` | `llmprovider/catalog/full_catalog_0038_test.go` |
| `TestFullList_SixFirst`; `TestRankRecommended_UncappedFillDiffers`, which pins A1's finding that today's six cannot be read off the uncapped ranking | `llmprovider/catalog/full_list_0038_test.go` |
| `TestWithLimit_Scope`, `TestListModels_WithLimit` | `llmprovider/providers/limit_0038_test.go` |
| `TestListModels_FullListProbesSix`, `TestListModels_ChatGPTWithLimit` | `llmprovider/providers/openai/limit_0038_test.go` |
| `TestOf`, `TestCut` | `llmprovider/internal/listlimit/listlimit_test.go` |

`TestListModels_ProbesAtMostMaxListedModels` (Ollama) takes
`catalog.WithLimit(0)` and requires all `MaxListed+3` models in order
(Phase 1's deviation 2).

**Done differently from the steps' wording.** No decision changed:

* **`TestWithLimit_Scope`** is in `llmprovider/providers`, not the catalog
  file. It builds every built-in provider, and `catalog`'s internal tests
  cannot import the providers.
* **`TestListModels_FullListProbesSix`** is in the new
  `openai/limit_0038_test.go`, beside the ChatGPT test the PLAN's step 6
  needed, not in `listing_test.go`.
* **Three tests are added beyond the table:** `TestFullList_SixFirst`,
  `TestRankRecommended_UncappedFillDiffers` and
  `TestListModels_ChatGPTWithLimit`.

**Existing tests,** each changed only in how it calls, so each keeps its
meaning (R46). A script matched each call's parentheses and counted the
edits:

* 14 `rankRecommended` calls, two `curate*` calls and five
  `curateFromCatalog` calls, across seven files, take `MaxListed`. Each
  `curateFromCatalog` call names its provider.
* `replaySnapshot` takes options after its own.

**Step 8, the fallback exclusion,** changed no existing test's expectation.
`TestStatic_EqualsLatestSnapshot` and `TestListModelCatalog_Snapshot20261012`
passed unchanged.

**Green:** `go test ./...`, all 30 packages `ok`.

**Plants** (`tools/plant.py`, 6 of 6 caught):

| Plant | FAIL line |
| :--- | :--- |
| the six not first (`fullList` from the uncapped curation alone) | `TestFullList_SixFirst`: `fullList = [b a c d e g f h]; want [a b c d e f g h]` |
| `curateFromCatalog` without the exclusion | `TestWithLimit_FallbackExcludes`: `together (0 options): Recommended [MiniMaxAI/MiniMax-M2.7 t/a t/b] …`, and `opencode-go (0 options): Recommended [qwen3.8-flash glm-5.3-flash deepseek-v4-flash] …` |
| `gemini`'s fallback not reading the limit | `TestListModels_WithLimit/gemini`: `1 options: ListModels = [gemini-3.5-flash-lite gemini-3.8-flash gemini-3.6-flash]; want [gemini-3.5-flash-lite gemini-3.8-flash]` |
| a negative n read as the six | `TestWithLimit_Cuts`: `WithLimit(-1) holds 6 ids; want the full list's 272` |
| the option scoped to one provider | `TestWithLimit_SixIsToday` and the others: `option catalog.WithLimit is for provider "kilo", not "huggingface"`; `TestWithLimit_Scope`: `gemini: New refused catalog.WithLimit` |
| `Of` keeping the first value | `TestOf`: `the last wins: Of = 6, true; want 0, true` |

**Checks:**

* `make pre-add-check` on the 28 Go files: clean (gofmt, golangci-lint, go
  vet, go test, govulncheck).
* `make api-check`: 0 incompatible changes against `v1.10.2`.
  `tools/apidiff.py v1.10.2` printed one compatible change,
  `./llmprovider/catalog.WithLimit: added`.
* `make coverage-check`: `llmprovider/internal/listlimit: 100.0% against
  80.0%`. No floor line was needed.

**Not done in this phase,** as planned:

* the `Catalog.Recommended` and `List` docs, which still say "at most
  `MaxListed`" (Phase 3);
* the wizard (Phase 3).

**Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 2 as `8f99c2f`.

### Phase 3: the default becomes the full list; the wizard follows (2026-10-13)

**Research: the default flipped alone.**

* **The run:** on a scratch clone at `8f99c2f`, `recommendLimit` was
  changed to return `WithLimit`'s n without the `MaxListed` default, and
  `go test ./...` was run. The tree was not touched.
* **The result:** 15 tests failed, 13 in `catalog` and 2 in `wizard`.
  * **(a), their meaning is "the six": all 15.**
    * `catalog`:
      * `TestListModelCatalog_OpenAIRecommendedIsCurated`;
      * `_UsableIsUncapped`;
      * `_OllamaSplit`;
      * `_KiloRanksByProfile`;
      * `_OpencodeGoGates`;
      * `_Snapshot20260926`;
      * `_Snapshot20261012`;
      * `TestStatic_EqualsLatestSnapshot`;
      * Phase 2's `TestWithLimit_SixIsToday`.
    * `wizard`:
      * `TestConfigureLLM_BlankFallbackRoundWithNothingLeft`: the five
        fallbacks are the six without the primary;
      * `TestConfigureLLM_FallbackSearchLoops`: "second menu … want 5 rows".
  * **(b), a G-wire listing golden:** none. `TestWireGoldens` passed, and no
    file under `testdata/wire` changed.
  * **(c), anything else:** none.
* **Read but not failing:** `wizard/configure.go:380-512`,
  `model_select.go:40-100` and their tests. Two wizard tests assert the
  wizard's own cut at `MaxListed`, which the flip alone did not touch:
  * `TestChatGPTCatalog_Capped`;
  * `TestConfigureLLM_ThirdPartyListsThroughItsLister`.

  Both are (a). They take `ModelLimit: catalog.MaxListed`, and a plant
  below shows the second failing without it.

**Step 1, `wizard.Options.ModelLimit`:** the field was added first, with no
behavior, so the new tests compiled.

**Red first:**

```text
--- FAIL: TestList_DefaultIsFullList
    full_catalog_0038_test.go:111: together profile 0: no option recommends 6 ids; want the full list's 19
    (the same for each ranked list and profile: Zen 77, Go 28, Kilo 272/285, Hugging Face 102/119, Together 19/21)
    full_catalog_0038_test.go:116: openai: no option recommends [gpt-6-luna … gpt-6-astra]; want the full list [… gpt-6-sol gpt-5.6-sol gpt-4.1]
    full_catalog_0038_test.go:119: ollama: no option recommends [m1 m2 m3 m4 m5 m6]; want every installed model [m1 … m8]
--- FAIL: TestWizard_MenuIsFullListByDefault
    model_limit_0038_test.go:97: built-in listing: the menu shows [deepseek-v4.1-flash qwen3.8-flash glm-5.3-flash deepseek-v4-flash gemini-3.5-flash-lite gemini-3.8-flash]; want all 8 listed models
    model_limit_0038_test.go:101: third party: the menu shows [acme-model-1 … acme-model-6]; want [acme-model-1 … acme-model-8]
    model_limit_0038_test.go:104: ChatGPT session: 6 recommended; want all 14
--- FAIL: TestWizard_ModelLimit
    model_limit_0038_test.go:114: built-in listing, ModelLimit 2: the menu shows [six ids]
    model_limit_0038_test.go:117: third party, ModelLimit 2: the menu shows [acme-model-1 … acme-model-6]; want [acme-model-1 acme-model-2]
    model_limit_0038_test.go:120: ChatGPT session, ModelLimit 2: 6 recommended
    model_limit_0038_test.go:125: built-in catalog, ModelLimit 2: the menu shows [claude-haiku-5-5 claude-sonnet-5-5 claude-sonnet-5 claude-opus-5-5 claude-opus-4-8]; want [claude-haiku-5-5 claude-sonnet-5-5]
```

`TestWizard_ProviderOptionsLimitRefused` passed before the change, as it
must. It documents A2: a third party's `New` refuses `catalog.WithLimit`
in `ProviderOptions`, so the wizard warns, "could not list", and offers the
static catalog.

**Change:**

* **`catalog`:** the default is the full list. `listlimit.Of` returns only
  n, zero when unset, since nothing reads "set" once the default is no limit
  (`unparam`). `config.limitSet` and `recommendLimit` are removed, and the
  listings read `cfg.limit`. The providers read `p.limit =
  listlimit.Of(st.Values())`.
* **`wizard/configure.go`:**
  * `Options.ModelLimit`, with its doc. It names `ProviderOptions` as the
    wrong place for the limit (A2).
  * `discoverModels` passes `catalog.WithLimit(o.ModelLimit)` beside
    `WithProfile`, and cuts the static fallback in both views.
  * `chatGPTCatalog` and `providerCatalog` cut with `cutModels(models,
    o.ModelLimit)` in place of `catalog.MaxListed`.

**The category (a) tests,** each taking the limit and so keeping its meaning
(R46):

* the catalog's nine (`WithLimit(MaxListed)`);
* the wizard's four (`ModelLimit: catalog.MaxListed`).

Phase 2's two 0038 tests also changed:

* `TestWithLimit_SixIsToday` now compares `WithLimit(MaxListed)` with the
  default list's first six, with `staticOpenAI`, and with Ollama's first six.
  The open catalogs' sixes are pinned by `TestListModelCatalog_Snapshot20261012`.
* `TestWithLimit_FullList` takes its six from `WithLimit(MaxListed)`.

**Docs:**

* `Catalog.Recommended` and `List`;
* `ModelLister` in `contract.go`;
* the package docs of `gemini`, `claude`, `grok` and `openai`, which said
  each listed model is probed;
* `gemini`'s `ListModels`, which said "a short, curated list";
* `docs/architecture.md`: the wizard paragraph, the `catalog.List` bullet
  and the `ListModels` bullet;
* `README.md`'s Listing bullet.

`MaxListed`'s doc changed in Phase 2.

**Green:** `go test ./...`, all 30 packages `ok`. `internal/cmd/catalogs`'s
tests passed unchanged.

**Plants** (`tools/plant.py`, 4 of 4 caught):

| Plant | FAIL line |
| :--- | :--- |
| the default left at six | `TestList_DefaultIsFullList`: `ollama: no option recommends [m1 m2 m3 m4 m5 m6]; want every installed model …`; `TestWithLimit_FullList`: `together profile 0: the full list holds 6 ids; want the 19 …` |
| the wizard ignoring `ModelLimit` on the third-party path | `TestConfigureLLM_ThirdPartyListsThroughItsLister`: `"acme-model-7" shown = true, want the first 6 listed ids only`; `TestWizard_ModelLimit`: `third party, ModelLimit 2: the menu shows [acme-model-1 … acme-model-8]` |
| the wizard cutting at `MaxListed` when `ModelLimit` is zero | `TestWizard_MenuIsFullListByDefault`: `third party: the menu shows [acme-model-1 … acme-model-6]`; `ChatGPT session: 6 recommended; want all 14` |
| the wizard not passing `ModelLimit` to `catalog.List` | `TestWizard_ModelLimit`: `built-in listing, ModelLimit 2: the menu shows [deepseek-v4.1-flash … claude-opus-5]` |

The fourth plant is beyond the PLAN's three. It shows that the built-in
path, which the other three do not touch, is covered.

**Checks:**

* `make pre-add-check` on the 23 Go files: clean.
* `tools/apidiff.py v1.10.2`, compatible changes only:
  `./wizard.Options.ModelLimit: added` and
  `./llmprovider/catalog.WithLimit: added`. These are V6's two.
* markdownlint on `README.md` and `docs/architecture.md`: no issue in
  either.

**Gate:** `tools/gate.py`, 23 of 23 passed.

The owner committed Phase 3 as `8e36d03`.

### Phase 4: live confirmation and release (2026-10-13)

#### Deviations

**Deviation 1 (2026-10-13): Together's listing is billed.**

* **Found:** the PLAN says this phase sends no billed request. But this
  repository treats a live Together listing as billed:
  `llmprovider/catalog/live_listing_test.go:25-37` skips Together unless
  `LLMPROVIDER_LIVE_TOGETHER=1`, because "live Together calls are billed".
  The measurement lists each provider twice, so Together costs two billed
  listing requests. No other provider's listing is billed.
* **Decision (the owner):** approve the two billed Together listings. The
  run sets `LLMPROVIDER_LIVE_TOGETHER=1` for that one invocation. The
  measurement uses the existing `liveEnvKey` gate, so a later run without
  the variable skips Together.

#### The measurement

`llmprovider/catalog/live_0038_full_catalog_measure_test.go`
(`TestLive_FullCatalog`) ran once, on 2026-10-13, with
`LLMPROVIDER_LIVE_TOGETHER=1` for that invocation only.

* **What it sent:** two listings per subtest, with the metadata document on
  as at runtime, and no generation. Two of the listings were Together's,
  billed and approved.
* **Keys:** all eight key variables were set, checked for presence only.
* **Result:** 13 of 13 subtests passed. In each, the six are the full list's
  first six, the full list holds no `excludedAtRequest` id, and it is a
  subset of `Usable`.

| Listing | Usable | Full | Six |
| :--- | ---: | ---: | :--- |
| Claude (keyed) | 14 | 14 | `claude-haiku-5-5`, `claude-sonnet-5-5`, `claude-sonnet-5`, `claude-opus-5-5`, `claude-opus-4-8`, `claude-haiku-4-5-20251001` |
| Gemini (keyed) | 12 | 12 | `gemini-3.5-flash-lite`, `gemini-3.8-flash`, `gemini-3.6-flash`, `gemini-3.7-flash`, `gemini-3.5-flash`, `gemini-3.1-flash-lite` |
| Grok (keyed) | 8 | 8 | `grok-4.6`, `grok-4.5`, `grok-4.7`, `grok-4.3`, `grok-build-0.1`, `grok-4.20-0309-non-reasoning` |
| OpenAI (keyed) | 73 | 73 | `staticOpenAI`, in order |
| Hugging Face (keyed and public) | 112 | 100 | the snapshot's utility six |
| Kilo (keyed and public) | 285 | 272 | the static six |
| OpenCode Go (keyed and public) | 35 | 29 | the static six |
| OpenCode Zen (public) | 80 | 78 | the static six |
| OpenCode Zen (keyed) | 42 | 40 | `claude-haiku-5-5`, `deepseek-v4.1-flash`, `qwen3.8-flash`, `glm-5.3-flash`, `gemini-3.8-flash`, `gemini-3.5-flash-lite` |
| Together (keyed) | 172 | 19 | the static six |

* **The keyed Zen listing is shorter than the public one,** as
  0033-MADR F1 found: the key's workspace has models turned off. Its six
  therefore differ from the public one's in one id.
* **Every vendor list is full:** the vendors' usability filters already hold
  only text models.

**Plants** (`tools/plant.py`, run with the live command, and with
`LLMPROVIDER_LIVE_TOGETHER` unset so that no plant run was billed):

| Plant | Result |
| :--- | :--- |
| the ranking's fill admitting an `excludedAtRequest` id | caught: `opencode-go (keyed)`: `the full list holds "deepseek-v4-flash", measured not to answer`, and the other five Go ids; the same for `opencode-go (public)` |
| `listlimit.Cut` keeping the last n | not caught. Neither live path runs the cut: `WithLimit(MaxListed)` cuts a curation that already holds at most six, and no option cuts nothing. `TestCut` covers it offline. |
| the full list not led by the six (reversed, in its place) | caught: `claude (keyed)`: `the six [claude-haiku-5-5 …] are not the full list's first 6: [claude-fable-5 claude-opus-4-6 …]`, and every other listing |

#### Release notes: `v1.11.0`

`v1.11.0` lists the full curated catalog by default, and makes today's six
models an option (0038-MADR, with its amendment A1–A3). The exported API
adds two identifiers and changes none.

**What changes by default:**

* **`catalog.List`'s `Catalog.Recommended`** is the full curated catalog.
  It holds every usable id the listing serves, less those measured not to
  answer and the profile's exclusions, ranked, with the best six first. It
  was at most six. For Kilo that is 272 models under the default profile.
* **Every built-in provider's `ListModels`** returns the same full list.
* **`wizard.ConfigureLLM`'s menu** shows the full list, on every path.

**To keep today's six,** add one option:

* `catalog.WithLimit(catalog.MaxListed)` to `catalog.List`, or to a
  provider's `New`;
* `ModelLimit: catalog.MaxListed` in `wizard.Options`. Pass the limit
  there, not in `ProviderOptions`: a third-party provider refuses the
  option.

With the limit, the six are today's six. The one exception: the fallback
curation (model metadata off or unavailable) no longer recommends an id
measured not to answer.

**Probes** (OpenAI with an API key, Claude, Gemini, Grok, Ollama) test only
the first `catalog.MaxListed` models of a listing, as before, and drop those
that fail. The later models are returned unprobed. A listing costs no more
billed requests than before.

**New API:**

* `catalog.WithLimit(n int) llmprovider.Option`, scoped to every built-in
  provider. With n zero or less there is no limit.
* `wizard.Options.ModelLimit int`. Zero is the whole list.

**Unchanged:** the static catalogs, `Catalog.Usable`, `catalog.Search` and
the ranker's rules.

**Consumers,** each in its own repository and its own record (0038-MADR
D7):

* prepare-commit-msg adds the opt-in to its `--yes` listing and its wizard
  options. Its primary model and fallbacks do not change without it, but its
  menu would.
* gobble-cli lists through `catalog` rather than `Descriptor.StaticModels`,
  with no limit.

**Upgrading:** a caller that wants six adds the option. No other change is
needed.

**Step 2:** `README.md` Status names `v1.11.0`, with a bullet.

**Gate:** `tools/gate.py`, 23 of 23 passed.

**Release checks:** `make lint vuln parity-check dep-check coverage-check api-check generate-check records-check gate-selftest` exit 0: lint 0 issues (host and Windows), no vulnerabilities, dep-check 0 problems, coverage-check 31 packages 0 problems, api-check 0 incompatible changes against `v1.10.2`, generate-check and records-check 0 problems.

**Phase 4, step 5, the release.**

* The owner committed steps 1 to 4 as `d1fbad5`.
* **The disclosure guard** ran over the five outgoing commits, `2ef6f25` to
  `d1fbad5`, when the owner asked ("run the guard"). It exited 0 with no
  finding.
* **The push and tag:** the owner pushed `main`, then tagged `v1.11.0`,
  annotated (`f0d7c60`), on `d1fbad5`, and pushed the tag.

**Step 6, after the tag:**

| Check | Result |
| :--- | :--- |
| CI on `main` | run `38417684107`, push, success |
| CI on the tag | run `38424063440`, push, success |
| The proxy | `"Version":"v1.11.0"`, `Ref refs/tags/v1.11.0`, hash `d1fbad5…` |
| `tools/apidiff.py v1.10.2 v1.11.0` | exit 0. Compatible changes only: `./llmprovider/catalog.WithLimit: added` and `./wizard.Options.ModelLimit: added` |
| `tools/consumer.py v1.11.0` | 15 packages linked, built for six platforms |
| `tools/offline.py v1.11.0` | passed: 19 fields, `requests=0`, six platforms. Both its plants were caught: "models.dev left on" and "no caller layer" |

**Step 7:** this PLAN is `complete`, and `docs/README.md` says so.

#### Verification, confirmed

| | Result |
| :--- | :--- |
| V1. The six | `WithLimit(MaxListed)` is the full list's first six, and the snapshot sixes are unchanged (Phases 2 and 3) |
| V2. The default | `TestList_DefaultIsFullList`, `TestWithLimit_FullList` against the measured lengths, and the live run, 13 of 13 |
| V3. Working models only | `TestWithLimit_FallbackExcludes`, and the live run's exclusion check |
| V4. Probes | Phase 1's tests and `TestListModels_FullListProbesSix`: at most six probes, the tail kept |
| V5. The wizard | `TestWizard_MenuIsFullListByDefault` and `TestWizard_ModelLimit`, on every path |
| V6. API | exactly `catalog.WithLimit` and `wizard.Options.ModelLimit` added |
| V7. Plants | every new assertion failed on a plant. Phase 4 named the one plant that could not reach its code |

#### Deviations, in all

All are dated 2026-10-13, each in its phase's record:

* **Phase 1:** an existing probe test asserted the old result, and now
  follows D4. The planned Ollama assertion moved to Phase 2.
* **Phase 4:** Together's two listings were billed, and the owner approved
  them.

#### Not done, and why

* **The consumers' adoption,** as 0038-MADR D7 decides. Each needs its own
  record in its own repository.
  * prepare-commit-msg: add `catalog.WithLimit(catalog.MaxListed)` to its
    `--yes` listing, and `ModelLimit: catalog.MaxListed` to its wizard.
  * gobble-cli: list through `catalog` instead of `Descriptor.StaticModels`,
    and widen its ACP model option.
* **The static catalogs** stay at most six (D3).
* **No live test asserts that a full list is longer than six.** Under a
  profile that excludes much, a list can hold six or fewer. The offline
  tests pin each length against the snapshot.

### Follow-up (2026-10-13): 0038-MADR amendment A4

0038-MADR's amendment A4 changes the full list's order and its exclusions,
and adds `Catalog.Menu`, for 0041-MADR. This PLAN stays `complete`: the work
is 0041-PLAN-decide-per-model-behaviour-from-metadata.md's, as follows.

* Phase 3 updates the static lists and this PLAN's snapshot tests.
* Phase 4 builds the menu and the new full list.
* Phase 4 re-runs this PLAN's offline tests:
  * the `_0038` tests in `llmprovider/catalog`;
  * `llmprovider/providers/limit_0038_test.go`;
  * `llmprovider/providers/openai/limit_0038_test.go`;
  * `wizard/model_limit_0038_test.go`.

  Each assertion A4 changes is updated in the same commit, and recorded
  there.

**Done in 0041-PLAN Phase 4** (2026-10-14), in the commit after `7f34e31`,
with 0038-MADR amendment A5. That PLAN's Phase 4 record lists each changed
assertion: `TestWithLimit_FullList`'s set and lengths, and
`TestFullList_SixFirst`, removed with `fullList`. The other `_0038` tests,
in `llmprovider/catalog`, `llmprovider/providers`, `providers/openai` and
`wizard`, pass unchanged.

### Follow-up (2026-10-16): `internal/listlimit` in the package tables

Step 2's new package, `llmprovider/internal/listlimit`, was not added to
`docs/guides/api-standards.md` R2's table or to `docs/architecture.md`.
This PLAN stays `complete`. 0041-PLAN-decide-per-model-behaviour-from-metadata.md
Phase 9, deviation 1, adds both: R2's row, and `catalog`'s import of it,
and in `architecture.md` its tree line, its row, its place in the package
graph, and the rows of `catalog` and the eight providers that import it.
