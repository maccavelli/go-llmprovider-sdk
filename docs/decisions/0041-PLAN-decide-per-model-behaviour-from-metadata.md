---
status: proposed
date: 2026-10-13
associated-madr: "0041-MADR-decide-per-model-behaviour-from-metadata.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Live Model Menu, and Per-Model Behaviour, From Model Facts

Associated MADR: [0041-MADR-decide-per-model-behaviour-from-metadata.md](0041-MADR-decide-per-model-behaviour-from-metadata.md)

It also implements
[0038-MADR-full-curated-catalog-by-default.md](0038-MADR-full-curated-catalog-by-default.md)'s
amendment A4, which 0038-PLAN's follow-up note hands to this PLAN.

Every code location a step names has an id, P*n*, in this PLAN's evidence
index, checked by `tools/claims.py` in Phase 0. A step that names a location
the index does not hold must add it, and check it, before the step runs.

## Goal

When it is done:

* **Facts decide** (D1, D2):
  * Messages thinking, Gemini thinking, Grok efforts and the output limit
    follow each model's facts, merged as caller, listing, models.dev;
  * every Chat Completions adapter replays reasoning under the model's
    `interleaved.field`;
  * the name rules decide only where no fact exists, and with metadata off;
  * no generation sends a request it did not send before.
* **The live menu** (D3, 0038-MADR A4). Every provider's `Catalog` has:
  * `Menu`: every usable model, ordered by facts, with each entry's facts,
    rank and caveats;
  * `Recommended`: the band, then the rest by fact, with no caveated entry.
* **The band** (D4) is ranked by facts for every provider:
  * "small" is the absolute 3:1 price band;
  * a one-vendor catalog groups by family;
  * caveated models are never in it.
* **`ProfileAgent`** (D5) orders `ProfileCapable`'s band by the price of a
  cached turn, with published constants that a caller can replace.
* **Steady** (D6):
  * a transient listing failure serves the last good listing, for up to 24
    hours;
  * an authentication failure does not;
  * `Source` and `FetchedAt` say which happened.
* **Checked** (D7): offline agreement tests in CI, and a refresh tool that
  reports rule and fact differences and how much each six changed.
* **Proven:**
  * every number 0041-MADR measured on the committed snapshot (§3–§5, D4,
    D5) is asserted by a test;
  * every E*n* and P*n* passes `tools/claims.py`.

  The live document's totals (§1, D1) and the published figures (§7) are
  recorded with their source and date, and cannot be asserted offline.
* **Raw listings** (D9). `catalog.ListRaw` returns every provider's listing
  as the provider sent it, with the SDK's transport and none of its
  curation. That covers all ten listings, the ChatGPT session's included.
* **Released** as a minor version. `tools/apidiff.py` reports exactly the
  additions under "The exported additions" below, and nothing else.

## Scope

### In scope

* **`llmprovider/catalog`:**
  * the facts: `CachedFacts`, `ModelInfo.CanonicalID` and the listing dates;
  * the menu: `Catalog.Menu`, `MenuEntry`, `Caveat`, `Catalog.Source`,
    `Catalog.FetchedAt`, and `ListSource`;
  * the ranking and profiles: D4's ranking, `ProfileAgent`, `WithWorkload`
    and `ObservedWorkload`;
  * steadiness: `WithLastGood` and the listing cache;
  * the static lists, which 0038-MADR A4.4 requires to follow the snapshot.
* **`llmprovider/internal/wire`:** `messages.AddThinking` and
  `generatecontent.ThinkingConfig` take resolved facts. Their rules stay as
  the default.
* **Providers:** each of `claude`, `opencode`, `gemini`, `grok`, `openai`,
  `kilo`, `together`, `huggingface` and `ollama` resolves the facts for its
  requests, and documents what it reads (R12).
* **Raw listings:**
  * `catalog.ListRaw`, `RawListing` and `RawModel`;
  * a module-internal package for the ChatGPT session's check, headers and
    listing request, shared by `catalog` and `providers/openai`.
* **The snapshot command** (`internal/cmd/catalogs`) records the first-party
  listings into a directory the ranking tests do not read.
* **`tools/`:** `claims.py`, `catalogcheck.py`, their tests and their
  README rows.
* **Docs:**
  * `docs/architecture.md`;
  * `docs/guides/api-standards.md` (R6, R10);
  * `README.md`;
  * `docs/README.md`.

### Out of scope

As 0041-MADR D8 decides:

* hysteresis on the band;
* external quality signals, and a `ProfileAgent` whose price chooses its
  members;
* `cost.tiers`, `context_over_200k` and `experimental.modes`;
* `knowledge`, `open_weights` and `attachment`;
* removing any name rule;
* the consumers' adoption. Each consumer has its own repository and records;
  Phase 9 drafts the issues, and opens them only when the owner asks.
* A raw models.dev document, and per-model detail calls such as Ollama's
  `/api/show` (0041-MADR D9).

### The exported additions

`tools/apidiff.py` must report these, all `added`, and nothing else:

| Package | Addition | Phase |
| :--- | :--- | :--- |
| `catalog` | `CachedFacts` | 1 |
| `catalog` | `ModelInfo.CanonicalID`, `FieldCanonicalID` | 2 |
| `catalog` | `Catalog.Menu`, `MenuEntry` and its fields `ID`, `Info`, `Rank` and `Caveats`; `Caveat`, `CaveatDeprecated`, `CaveatNoTools`, `CaveatNoTextOutput`, `CaveatExpiring` and `CaveatUnavailable` | 4 |
| `catalog` | `ProfileAgent`, `WithWorkload`, `ObservedWorkload` | 5 |
| `catalog` | `Catalog.Source`, `Catalog.FetchedAt`, `ListSource`, `ListLive`, `ListStale`, `ListStatic`, `WithLastGood` | 6 |
| `catalog` | `ListRaw`; `RawListing` and its fields `Provider`, `Endpoint`, `FetchedAt`, `Pages` and `Models`; `RawModel` and its fields `ID` and `Raw` | 8 |

The names are this PLAN's, from the MADR's descriptions. A phase whose
research finds a name that collides, or reads badly beside its neighbours,
stops and records a deviation.

## Rules for every phase

* **Research first.** Each phase starts by reading the code it changes, at
  the commit it starts from, and by re-running `tools/claims.py` on both
  records. A fact that differs from this PLAN or the MADR is a deviation.
* **Red first.** Each new test is run before the change, and its FAIL lines
  are recorded here.
* **Plants.** Every new assertion is shown failing by `tools/plant.py` on a
  scratch clone, never in the tree. Each plant's FAIL line is recorded.
* **Python.** Every script is Python, and a reusable one goes in `tools/`
  with a test. Specs and scripts are written with the Write tool. Exact
  edits go through `tools/edit.py`.
* **The goldens.** An existing G-wire golden never changes; the wire
  scenarios run with metadata off (P27). A new golden is generated alone,
  and recorded here.
* **The snapshot tests.** `TestListModelCatalog_Snapshot20261012` and
  `TestStatic_EqualsLatestSnapshot` change only in Phase 3. Each changed
  six is recorded against 0041-MADR D4's table.
* **Before a phase is staged:**
  * `make pre-add-check FILES="…"`;
  * `make records-check`;
  * `tools/denyscan.py --changed` and `tools/linkcheck.py`;
  * `tools/claims.py` on both records;
  * the gate, `tools/gate.py`, at 23 of 23 (24 once Phase 0 adds the claims
    check).
* **Keys and requests.** Keys are checked for presence only, never printed.
  The owner approves the count of any billed request before it runs. None is
  planned.
* **A deviation stops the work.** Give the evidence, the options with the
  recommended one first, and a dated entry here, plus a MADR amendment where
  a fact or a decision changes.
* **The owner commits** each phase, pushes and tags. While one phase is
  staged, the next stays unstaged.
* **Ordering with 0040-PLAN.** Both plans change `internal/wire/messages`:
  this PLAN's Phase 1 its thinking, 0040-PLAN's Phase 2 its tools and tool
  results. They are run one at a time, in the order the owner chooses. The
  second starts by re-reading the files the first changed, and re-running
  both records' claims checks, whose line numbers the first will have moved.

## Implementation Steps

### Phase 0: records, the claims check, and the evidence tests

1. **The records.**
   * 0041-MADR was accepted on 2026-10-13. This PLAN becomes `in-progress`
     on the owner's approval.
   * If 0040-PLAN-native-tool-deferral.md's Phase 0 has already committed
     `tools/claims.py`, step 2 is skipped and recorded, and the check runs
     as it stands.
   * `docs/README.md` gets the PLAN's row, the statuses and the count.
2. **`tools/claims.py`.** It is the checker that verified 0041-MADR, made
   general:
   * **Arguments:** `claims.py RECORD.md [NAME=DIR@REV ...]`.
   * **What it reads:** every row of a table whose first cell is `E` or `P`
     and a number, whose Where cell is `` `path:line` `` (this repository) or
     `` `name:path:line` `` (another), and whose last cell is the matched
     text in backticks.
   * **How it checks:** it reads each file at the revision with
     `git show REV:PATH`, never the working tree, and requires the text on
     that exact line. A failure names the lines where the text was found.
   * **Exit codes:** 0 when every row passes, 1 on any failure, or on a
     record with no rows, and 2 on a tool error.
   * **Tests** in `tools/test_tools.py`, each in a temporary git repository:
     * a passing table;
     * a wrong line, with the line found reported;
     * a missing file;
     * a revision that lacks the file;
     * an escaped `\|` in the matched text;
     * a record with no rows.
   * **Other wiring:**
     * a row in `tools/README.md`;
     * `gate.py` gains it as a check on the changed records, which makes 24
       checks;
     * `make gate-selftest` covers the new check.
   * **Plant:** the line comparison off by one, which a test must catch.
3. **`llmprovider/catalog/evidence_0041_test.go`.** Offline, on
   `testdata/ranking-2026-10-12`, with the clock pinned at its
   `recorded_at`. It asserts each number 0041-MADR states, through the
   code's own functions:

   | Test | Asserts (0041-MADR) |
   | :--- | :--- |
   | `TestEvidence0041_ClaudeAdaptive` | R1: `claudeAdaptiveOnly` equals "no `budget_tokens` option" on 34 of 34 Claude reasoning models in `anthropic`, `opencode` and `opencode-go` (§3) |
   | `TestEvidence0041_GeminiLegacy` | R4: 27 of 29, the two others being `gemini-2.5-flash-image` and `gemini-2.5-computer-use-preview-10-2025` (§3) |
   | `TestEvidence0041_GrokMenus` | R6: 5 of 7 reasoning models agree; `grok-4.3` and `grok-4.20-multi-agent-0309` differ as stated (§3) |
   | `TestEvidence0041_OutputLimits` | R7: 52 models below 8192 across the sections; the smallest Claude limit is 64,000 (§3) |
   | `TestEvidence0041_FullListFlags` | the OpenAI curator over its section gives 42 ids, 11 of them deprecated, `gpt-3.5-turbo` at #32 without tools; Grok's gives 8, the last without tools (§4) |
   | `TestEvidence0041_ListingDates` | the per-listing table of §4: rows, distinct `created`, and matches within 31 days |
   | `TestEvidence0041_CostKeyReach` | §5's table: 13 of 276, 3 of 136, 0 of 78, 144 of 23,220 |
   | `TestEvidence0041_CachedTurn` | §7: the uncached Sonnet 5 price is 2.08; the cached one lies in [0.47, 0.53] |

   * Each later phase that changes a function under test updates these
     assertions in the same commit, to the new values, which become that
     phase's proof. For example, Phase 4 makes the deprecated count 0.
   * **Plants:** one per test, changing the function's input or output by
     one model.
4. **Green.** `go test ./llmprovider/catalog -run Evidence0041`,
   `make tools-test`, then the gate. Stage. The owner commits.

### Phase 1: facts at request time (D1, D2)

1. **Research.**
   * Read `CachedModel` and its fetch rules (P8), and OpenCode's
     `requestRoute`, the pattern to follow (P9).
   * Read every call site this phase changes:
     * `messages.AddThinking` (P1, P2, P20);
     * `generatecontent.ThinkingConfig` (P3, P21);
     * Gemini's `thinkingLevel` (P4);
     * Grok's `clampReasoningEffort` (P5);
     * `Settings.MaxTokens` (P6);
     * the Chat Completions `Body` and its replay options (P7, P22).
   * Confirm the wire scenarios run with metadata off (P27).
2. **`catalog.CachedFacts(ctx, provider, model, opts...) (ModelInfo, bool)`.**
   * It merges:
     * the caller's layers;
     * a listing already cached for the provider, base URL and credential
       in `opts`;
     * the cached document.
   * It never starts a listing fetch (0041-MADR D1). With no document
     cached, it starts that fetch only, unless metadata is off.
   * It is `CachedModel` without the listing fetch. `CachedModel` itself is
     unchanged.
3. **The wire takes facts.** Each function keeps its signature for today's
   callers, and gains a sibling that takes the facts:
   * **`messages.AddThinkingFor(body, model, effort, budget, maxTokens,
     facts messages.Thinking)`.** `Thinking` holds:
     * `Known bool`;
     * `Adaptive bool`;
     * `EffortOnly bool`;
     * `BudgetMin, BudgetMax int`.

     With `Known` false it is `AddThinking`.
   * **`generatecontent.ThinkingConfigFor(model, effort, budget, facts
     generatecontent.Thinking)`,** the same way.
   * Both are module-internal, so they are not API.
4. **The providers resolve and pass.**
   * **`claude` and `opencode`'s Messages route.** From `Reasoning`:
     * a budget bound means an enabled budget, clamped to it;
     * efforts with no budget mean adaptive thinking with the effort;
     * a toggle alone means adaptive with no effort.
   * **`gemini` and `opencode`'s Google route:** a budget, or a level.
     Gemini's `thinkingLevel` sends a level only when it is in
     `Reasoning.Efforts`.
   * **`grok`:** `clampReasoningEffort` clamps to `Reasoning.Efforts` when
     the fact is known, else `effortMenu`.
   * **Every provider, the output limit.** With no `MaxOutputTokens`, the
     request's limit is `min(st.MaxTokens(), info.MaxOutputTokens)` when the
     fact is known and positive. A raise for a thinking budget stops at it.
   * **`together`, `huggingface` and `ollama`:** `ReplayReasoningField` from
     the model's `interleaved.field`, as OpenCode's `chatReplayField` does.
     `kilo` is unchanged (P7).
5. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestCachedFacts_NoListingFetch_0041` | with a listing server counting requests, and a cold cache, `CachedFacts` makes 0 listing requests; `CachedModel` makes 1 |
   | `TestCachedFacts_Precedence_0041` | a caller layer beats the listing, which beats models.dev, field by field |
   | `TestAddThinkingFor_0041` | each of the three shapes from facts. With `Known` false the body equals `AddThinking`'s, byte for byte |
   | `TestThinkingConfigFor_0041` | a budget or a level from facts; a level not in the list is not sent |
   | `TestGrokEfforts_Facts_0041` | `grok-4.3` with a listing `reasoning_effort` holding `xhigh` sends `xhigh`; with no facts, the menu's |
   | `TestOutputLimit_Facts_0041` (per provider, through `wiretest`) | a model whose `limit.output` is 4096 sends 4096; one of 128,000 sends 8192 |
   | `TestReplayField_Facts_0041` | Together and Hugging Face replay under `reasoning_content` when the fact says so; none when unknown |
   | `TestAgreement0041_*` (the offline half of D7) | on the committed snapshot, R1, R2, R4, R5 and R6 from facts equal the rule for every model both answer, except the listed set. The set is `grok-4.3`, `grok-4.20-multi-agent-0309` and the two Gemini models of §3, each with its reason |
   | the wire goldens | unchanged: every scenario, three runs |

6. **Plants:**
   * `CachedFacts` calling the listing fetch;
   * the precedence inverted;
   * each shape's branch swapped;
   * the output cut removed;
   * the replay field ignored;
   * one agreement exception removed from the list.
7. **Docs.** Each provider's package doc names the facts it reads and its
   fallback (R12). Stage after the gate. The owner commits.

### Phase 2: listing facts (D1)

1. **Research.**
   * Read the listing decoders (P15, P16, P19).
   * Read the snapshot command's lists and keys (P17).
   * Read `latestSnapshot`'s glob (P18), which reads only `ranking-*`
     directories.
2. **Record the first-party listings.**
   * `internal/cmd/catalogs` gains `-firstparty DIR`, which writes the
     listings of OpenAI, Anthropic, Gemini and xAI, read with keys from the
     environment (checked for presence only), and a manifest with each
     file's SHA-256.
   * DIR is `llmprovider/catalog/testdata/firstparty-YYYY-MM-DD`, a name
     `latestSnapshot` does not read, so the ranked static lists do not move
     in this phase.
   * These four listing endpoints are free. No billed request.
   * **Redaction.** A listing body holds no key, but the files pass
     `tools/denyscan.py`, and any account or organisation field is replaced
     by a placeholder before they are committed.
3. **Measure each listing's dates** against models.dev, as §4 of the MADR
   did. A listing's date is read where at least 90% of its matched rows are
   within 31 days, and the measured table is recorded here. A listing whose
   `created` is the same on every row, or zero, gives no date (D1). Today's
   evidence: Kilo and Hugging Face qualify; Zen and Go do not.
4. **Decode.**
   * The dates that qualify, into `ModelInfo.Released` from the listing
     layer.
   * `canonical_model_id` into `ModelInfo.CanonicalID` (`FieldCanonicalID`).
   * An unexported helper `pricesByCanonical(doc)`, which gives a model's
     missing `cache_read` from the same canonical model's first-party
     section (D5).
   * The expiry facts already decoded (OpenAI `shutdown_date`, Kilo
     `expiration_date`), checked as `Retires`, for Phase 4's
     `CaveatExpiring`.
5. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestListingDates_0041` | Kilo's and Hugging Face's listing dates set `Released` from the listing; Zen's and Go's do not; a listing with one `created` on every row sets none; a `0` sets none |
   | `TestListingDates_FirstParty_0041` | on the recorded first-party listings, each listing the measurement qualified sets `Released`, and each it did not leaves it unset |
   | `TestCanonicalID_0041` | decoded where present; `pricesByCanonical` fills a missing `cache_read`, and fills nothing when the canonical model has none |

6. **Plants:** the guard for one constant `created` removed; a listing that
   does not qualify read anyway; the canonical lookup on the wrong section.
7. **Green.** The gate. Stage. The owner commits.

### Phase 3: the band (D4)

1. **Research.**
   * Read the ranking's eligibility, groups and orders (P10, P11, P12,
     P13), the Kilo candidate (P14) and `metadataCurate` (P24).
   * Read how the first-party providers curate (P25).
   * Read both snapshot tests (P26).
2. **Change.**
   * **"Small"** in `metadataCandidate` and `kiloCandidate`: with a known,
     positive price, `small = (3 × input + output) / 4 ≤ 1.0` per million
     tokens. Otherwise `isSmallModel` decides.
   * **Unusable** (no tools, or no text output) is never eligible.
     Deprecated was already ineligible.
   * **Groups:** in `metadataCurate`, OpenAI, Claude, Gemini and Grok group
     by `family`.
   * **First-party curation:** `modelCatalogFor`'s four first-party cases
     curate through `metadataCurate`, with their name curators as the
     fallback (P25).
3. **The snapshot tests change here, and only here.**
   * `TestListModelCatalog_Snapshot20261012`'s sixes become 0041-MADR D4's
     table, row for row.
   * The five ranked static lists become the new utility sixes, so
     `TestStatic_EqualsLatestSnapshot` holds.
   * Each changed id is recorded here against the MADR's table. Any
     difference from the table is a deviation.
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestSmallFromPrice_0041` | `gpt-6-luna` (blend 0.20) small; `gpt-5.4-mini` (1.69) not; a model with no price small by its name |
   | `TestUnusableIneligible_0041` | a `tool_call: false` model and an image-output model are never in the band, and stay in `Usable` |
   | `TestOneVendorGroups_0041` | OpenAI's band holds up to two of each family, and no `o3-pro` |
   | `TestFirstPartyRankByFacts_0041` | the four first-party providers' sixes, on their sections, equal D4's table; with metadata off they equal today's static order |
   | the two snapshot tests | as step 3 |
   | `TestEvidence0041_*` | updated where D4 changes what they measure |

5. **Plants:**
   * the price band at 2.0;
   * the regex deciding over a known price;
   * unusable models made eligible;
   * the letters group restored;
   * one first-party case still on its name curator.
6. **Green.** The gate. Stage. The owner commits.

### Phase 4: the live menu (D3, 0038-MADR A4)

1. **Research.**
   * Read `catalogFrom`, `recommend` and `fullList` (P23, P24).
   * Read the `_0038` tests that 0038-PLAN's follow-up lists, and the
     wizard's use of `Recommended`.
2. **Types.**
   * `type MenuEntry struct { ID string; Info ModelInfo; Rank int; Caveats
     []Caveat }`.
   * `type Caveat string`, with five constants: `CaveatDeprecated`,
     `CaveatNoTools`, `CaveatNoTextOutput`, `CaveatExpiring` (within 30
     days) and `CaveatUnavailable` (`excludedAtRequest`).
   * `Catalog.Menu []MenuEntry`.
3. **Build.**
   * **Entries:** for every `Usable` id, its facts (`CachedFacts`), its rank
     (its place in the band, else 0), and its caveats.
   * **The order** (D3):
     1. the band, by rank;
     2. the uncaveated entries, newest first by release date, then in
        listing order, then by id;
     3. the caveated entries, in the same order.
   * **`Recommended`** is the menu's first two groups' ids. `WithLimit(n)`
     cuts it.
   * **The static path:** `Menu` holds the static ids, with what facts are
     cached.
   * **Ollama:** listing order, with `CaveatNoTools` where its `/api/show`
     capabilities say so.
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestMenu_Order_0041` | on a fixture: band by rank, then newest first, then listing order, then id; caveated last |
   | `TestMenu_Deterministic_0041` | the same input in a shuffled listing order gives the same menu, 100 times, under `-shuffle=on` |
   | `TestMenu_Caveats_0041` | each caveat from its fact; a caveated entry is in `Menu`, not in `Recommended` |
   | `TestMenu_Snapshot_0041` | on the 2026-10-12 snapshot, each provider's `Menu` holds exactly `Usable`, and OpenAI's section-based full list holds no deprecated model (the evidence test's 11 becomes 0) |
   | `TestRecommended_Prefix_0041` | `WithLimit(n)` is a prefix of the unlimited list for n = 1 to 8; `WithLimit(MaxListed)` is the band |
   | 0038's tests | each assertion A4 changes is updated and recorded; the rest pass unchanged |
   | the wizard's tests | unchanged: it shows `Recommended` |

5. **Plants:**
   * caveated entries left in `Recommended`;
   * the id tie-break removed (caught by the determinism test);
   * the rest ordered by name score instead of date;
   * a caveat's fact ignored.
6. **Docs.** `docs/architecture.md`, the catalog section. 0038-PLAN's
   follow-up note gains the line "done in 0041-PLAN Phase 4, commit …".
   Stage after the gate. The owner commits.

### Phase 5: `ProfileAgent` (D5)

1. **Research.** Read `Profile` (P11), `WithProfile`, the capable order
   (P12), the band's loop (P13), and Phase 2's `pricesByCanonical`.
2. **Change.**
   * `ProfileAgent Profile = 2`. `ReasoningEffort` returns `""` for it.
   * **The band's members** are `ProfileCapable`'s band, computed as it is.
     **Their order:**
     1. signal;
     2. bench;
     3. preferred index;
     4. `ageDays / 90`;
     5. the cached turn;
     6. id.
   * **The cached turn:** `(1 − o) × (h × read + (1 − h) × write) + o ×
     output`.
     * `read` is `cache_read`, else the canonical model's, else `input`.
     * `write` is `cache_write`, else `input`.
     * An unknown price sorts last.
   * **The constants:** `agentCacheShare = 0.90` and `agentOutputShare =
     0.01`, each with a comment citing its source (0041-MADR §7).
   * **`WithWorkload(cacheShare, outputShare float64) llmprovider.Option`,**
     scoped like `WithProfile`. Each value must be in (0, 1), or the option
     is refused with `ErrInvalidRequest` when applied.
   * **`ObservedWorkload(usages ...llmprovider.Usage) (cacheShare,
     outputShare float64, ok bool)`:**
     * the cache share is ΣCachedTokens / ΣInputTokens;
     * the output share is ΣOutputTokens / Σ(InputTokens + OutputTokens);
     * `ok` is false when the sums are zero.
3. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestProfileAgent_Members_0041` | on the snapshot, for every provider and section, the agent band's members equal `ProfileCapable`'s band |
   | `TestProfileAgent_Order_0041` | on a fixture, a cheaper cached turn in the same quarter comes first, and a newer quarter beats a cheaper one |
   | `TestCachedTurn_0041` | the Sonnet 5 example: 2.08 uncached; in [0.47, 0.53] cached; an unknown `cache_read` with a canonical model takes its price; with none, the input price |
   | `TestWithWorkload_0041` | (0.5, 0.05) changes the order of a fixture where it should; 0, 1 and NaN are refused |
   | `TestObservedWorkload_0041` | three usages give the stated shares; no usages give `ok` false |
   | `TestProfileAgent_ReasoningEffort_0041` | `""` |
   | `TestEvidence0041_AgentTrial` | 0041-MADR D5's trial, the price choosing the members, on the snapshot with D4's rules: Zen keeps 1 of capable's 6, the Kilo section 1, OpenAI, Claude, Hugging Face and Together 6, Go 5, Gemini 4 of 5; over the 4 × 4 grid of `h` and `o`, the distinct bands are OpenAI, Claude, Gemini, Grok and Hugging Face 1, Go and Together 3, Zen 4, the Kilo section 12 |

4. **Plants:**
   * members chosen by price;
   * the bucket removed;
   * `h` and `o` swapped;
   * the canonical fill skipped;
   * the range check removed.
5. **Green.** The gate. Stage. The owner commits.

### Phase 6: steady menus (D6)

1. **Research.**
   * Read the listing cache (P15).
   * Read how each lister's error is classified, `APIError` and its kinds.
   * Read the consumer lines that depend on `Live` and `Err`: 0041-MADR E42
     to E45.
2. **Change.**
   * **The cache.** `listingEntry` gains the ordered usable ids. `List`
     serves a fresh entry, within `listingMetadataTTL`, without fetching,
     and fetches one key at a time.
   * **Stale.** On a fetch failure that matches `ErrProviderUnavailable`,
     `ErrRateLimited`, `context.DeadlineExceeded` or a network error, with
     an entry under 24 hours old, the result is that entry with:
     * `Source = ListStale`;
     * `FetchedAt` its time;
     * `Err` the failure;
     * `Live` true.
   * **No stale on authentication or invalid requests.**
     `ErrAuthFailure` and `ErrInvalidRequest` return the static floor with
     `Err`, as today.
   * **The floor.** No entry and no `WithLastGood` gives the static list,
     `Source = ListStatic` and `Live` false.
   * **`WithLastGood(ids, fetchedAt)`** supplies the entry when none is
     cached. It is used only on a failure, and only within 24 hours.
   * `Catalog.Source` and `FetchedAt` are set on every path.
3. **Red first,** each with a fake clock and a scripted server:

   | Test | Want |
   | :--- | :--- |
   | `TestList_Cached_0041` | two `List` calls within the TTL make one listing request; concurrent calls make one |
   | `TestList_StaleOn503_0041` | 200 then 503: the second result is the first's ids, `ListStale`, `Live` true, `Err` matching `ErrProviderUnavailable` |
   | `TestList_NoStaleOn401_0041` | 200 then 401: the static list, `ListStatic`, `Live` false, `Err` matching `ErrAuthFailure` |
   | `TestList_StaleExpires_0041` | 200, then 503 25 hours later: the static floor |
   | `TestList_LastGood_0041` | a cold process with `WithLastGood` and a 503 serves the given ids as stale; with a 401, the floor |
   | `TestList_SourceAlways_0041` | every path sets `Source` and `FetchedAt` |

4. **Plants:**
   * stale served on a 401;
   * the 24-hour window removed;
   * `Live` false on stale;
   * the fetch not deduplicated.
5. **Docs.** `Catalog.Live`'s doc says it covers a stale listing, and names
   `Source`. Stage after the gate. The owner commits.

### Phase 7: the refresh check (D7)

1. **`tools/catalogcheck.py`:**
   * It runs `go run ./llmprovider/internal/cmd/catalogs -snapshot DIR`
     (keys are checked for presence only).
   * It runs the agreement tests on DIR (`TestAgreement0041_*`, with
     `LLMPROVIDER_SNAPSHOT_DIR=DIR`).
   * It reports each rule and fact difference, by model.
   * It reports, per provider and profile, how many of the six changed
     against the latest committed snapshot.
   * Exit 0 is clean, 1 is a difference found, 2 is a tool error. It reports
     and never edits.
2. **Tests** in `tools/test_tools.py`: a fixture snapshot with one planted
   disagreement and one changed six, each reported; a clean fixture.
3. **Docs.** Its row in `tools/README.md`, and the refresh step in
   0033-PLAN-refresh-the-curated-catalogs.md's process, by a dated note.
   Stage after the gate. The owner commits.

### Phase 8: raw listings (D9)

1. **Research.**
   * Read each provider's listing fetch:
     * Claude's page (P32);
     * Gemini's page (P33);
     * the OpenAI-shaped fetch that OpenAI, Grok and OpenCode share (P38,
       0041-MADR E50);
     * Hugging Face (P37), Together (P36), Kilo (P35) and Ollama (P34).
   * Read `listAndRecord`'s credential renewal (P39).
   * Read the ChatGPT session's check, listing, sort and filter (P30, P31,
     P28, P29).
   * **Record an envelope per provider:**
     * Zen, Go, Kilo, Hugging Face and Together from
       `testdata/ranking-2026-10-12`;
     * the four first-party listings from Phase 2's recording;
     * a ChatGPT session's and Ollama's from the owner's own, with the
       owner's approval: each is free, and the ChatGPT one is metered on
       the subscription.

     Each is redacted, and checked by `tools/denyscan.py`. A provider whose
     envelope differs from 0041-MADR D9's table is a deviation.
   * **Check the package map.** Read `scripts/` dep-check and
     `docs/architecture.md` for whether `catalog` may import a new
     `llmprovider/internal/chatgpt`, which imports `llmprovider/auth`. If it
     may not, stop: that is a deviation, and the ChatGPT row of D9 waits on
     the owner's choice.
2. **The shared ChatGPT package.**
   * `isChatGPTSession`, `setChatGPTHeaders` and the listing request (the
     URL with `client_version`) move from `providers/openai` to
     `llmprovider/internal/chatgpt`.
   * `providers/openai` calls them, and its behaviour, sort and filter are
     unchanged. Its existing tests pass untouched; that is the proof.
3. **`ListRaw`.**
   * It resolves the provider's endpoint, credential header, base URL,
     client and user agent exactly as `modelCatalogFor` does, through
     shared unexported helpers, so the two paths cannot drift.
   * It wraps the client in a recording transport, as the snapshot command
     does (0041-MADR E49), and reads every page.
   * It splits each page's model array into `RawModel`s with
     `json.RawMessage`, so each entry's bytes are kept exactly.
   * It renews a refused credential once, as `listAndRecord` does.
   * It never touches the listing cache or the models.dev document, and
     never falls back to a static list.
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestListRaw_Verbatim_0041` (per provider, on its recorded envelope) | `Pages` equal the served bodies byte for byte; `Models` has one entry per array element, each equal to the element's bytes; `ID` is the provider's own field (`slug` for ChatGPT, `name` for Gemini, prefix included) |
   | `TestListRaw_NoCuration_0041` | a body with an embedding model, an image model, a deprecated model, a duplicate id and an unknown field returns all of them, in the body's order, the unknown field kept |
   | `TestListRaw_Pages_0041` | Claude's three pages by `after_id`, and Gemini's three by `pageToken`, all returned in order; an eleventh page is an error |
   | `TestListRaw_Bound_0041` | a page over 8 MiB is an error |
   | `TestListRaw_NoFallback_0041` | a 503 returns a zero `RawListing` and an error matching `ErrProviderUnavailable`; a 401 renews once, then returns the error |
   | `TestListRaw_NoSideEffects_0041` | a metadata server counting requests sees 0; after `ListRaw`, `CachedFacts` and `CachedModel` know nothing of the listed models |
   | `TestListRaw_SameTransportAsList_0041` | for each provider, `ListRaw` and `List` send the same method, path, query and credential header |
   | `TestListRaw_ChatGPT_0041` | a ChatGPT session lists the Codex backend with `client_version`, and returns every entry unsorted and unfiltered, `visibility` and `supported_in_api` included |
   | `TestListRaw_Ollama_0041` | `/api/tags`, `name` as `ID` |
   | `providers/openai`'s existing listing tests | unchanged, after the package move |

5. **Plants:**
   * the usability filter applied in `ListRaw`;
   * a page re-encoded instead of kept;
   * the ChatGPT sort applied;
   * the listing cache written;
   * the static fallback returned on a 503;
   * the second page skipped.
6. **Docs.** The `catalog` package doc and `docs/architecture.md` describe
   `ListRaw` beside `List`. Each provider's package doc names its raw
   endpoint (R12). Stage after the gate. The owner commits.

### Phase 9: documentation, live check and release

1. **Docs.**
   * `docs/guides/api-standards.md`: R6, for the facts at request time, and
     R10, for the menu and `ProfileAgent`, citing 0041.
   * `docs/architecture.md`.
   * `README.md`, its Status and a bullet.
2. **Live,** optional, with the owner's approval of the count: one Claude
   generation per changed thinking shape, and one Gemini level, each
   through the fact path. Up to three billed requests.
3. **Checks:**
   * `make lint vuln parity-check dep-check coverage-check api-check
     generate-check records-check gate-selftest`;
   * `tools/apidiff.py` against the last tag shows exactly "The exported
     additions";
   * the gate.
4. **Release notes,** recorded here, and the owner tags.
5. **After the tag:**
   * CI on `main` and on the tag;
   * the module proxy;
   * `tools/consumer.py` and `tools/offline.py` on the new version.
6. **The consumers.** Draft an issue for each; open them only on the owner's
   ask:
   * **gobble-cli:**
     * `ProfileAgent`, with `ObservedWorkload` from its own usage;
     * `Menu` and its caveats in the model picker;
     * `Source` in its retry logic (E42, E43).
   * **prepare-commit-msg:** the sixes it will see change (D4's table).
     `Err` on a stale list does not mean a refused key (E45).

## Verification

| 0041-MADR | Proven by |
| :--- | :--- |
| §3–§5, §7 numbers | `TestEvidence0041_*` (Phase 0), updated as phases change them |
| every E*n*, P*n* | `tools/claims.py`, in the gate |
| D1 precedence, no listing fetch | `TestCachedFacts_*` |
| D2 | the Phase 1 tests, `TestAgreement0041_*`, unchanged goldens |
| D1 dates, canonical | the Phase 2 tests |
| D4 | the Phase 3 tests, and the two snapshot tests equal to D4's table |
| D3, 0038-MADR A4 | the Phase 4 tests |
| D5 | the Phase 5 tests |
| D6 | the Phase 6 tests |
| D7 | `TestAgreement0041_*` in CI; `tools/catalogcheck.py` and its tests |
| D9 | the Phase 8 tests, and `providers/openai`'s listing tests unchanged |
| API | `tools/apidiff.py`: exactly the additions |

## Rollout and Rollback

* **Rollout.** One minor version, after Phase 9. Each phase is its own
  commit, and the tree passes the gate after each.
* **Rollback.** The facts layer turns off with `WithoutModelMetadata`:
  every request decision falls back to its rule, as it does cold. A bad
  ranking is fixed by a patch release. The menu's fields are additive, so a
  caller that ignores them sees `Recommended` and `Usable` as before, apart
  from D4's sixes. Reverting a phase's commit reverts that phase alone.
  Phases 4 to 6 depend on 2 and 3, so they revert in reverse order.
  Phase 8 depends on no other phase but Phase 2's recording, and reverts
  alone. Its package move is reverted with it.

## Evidence index

Checked against go-llmprovider-sdk `3e81e3d`, at that commit, by the
checker Phase 0 commits as `tools/claims.py`. The first run found three
rows a line off; they were corrected, and the second run passed every row.

| Id | Location | Where | Matched |
| :--- | :--- | :--- | :--- |
| P1 | Claude's thinking call | `llmprovider/providers/claude/claude.go:331` | `messages.AddThinking(body, model, string(effort), budget, maxTokens)` |
| P2 | OpenCode's Messages thinking call | `llmprovider/providers/opencode/opencode.go:523` | `messages.AddThinking(body, c.model, string(c.reasoning.Effort), c.reasoning.Budget, maxTokens)` |
| P3 | OpenCode's Google thinking call | `llmprovider/providers/opencode/opencode.go:539` | `generatecontent.ThinkingConfig(c.model, string(c.reasoning.Effort), c.reasoning.Budget)` |
| P4 | Gemini's level | `llmprovider/providers/gemini/gemini.go:220` | `if level := thinkingLevel(model, r.Effort); level != "" {` |
| P5 | Grok's clamp | `llmprovider/providers/grok/grok.go:258` | `clamped := clampReasoningEffort(model, string(effort))` |
| P6 | the output limit setting | `llmprovider/settings.go:289` | `func (st *Settings) MaxTokens() int { return st.s.cfg.MaxTokens }` |
| P7 | Kilo's replay | `llmprovider/providers/kilo/kilo.go:311` | `ReplayReasoningDetails: true,` |
| P8 | `CachedModel` | `llmprovider/catalog/lookup.go:94` | `func CachedModel(` |
| P9 | OpenCode reads the cached document | `llmprovider/providers/opencode/opencode.go:466` | `meta, cached = catalog.CachedMetadataWith(ctx, p.gateway, p.listing...)` |
| P10 | eligibility | `llmprovider/catalog/model_ranking.go:118` | `func (c rankCandidate) eligible(provider llmprovider.ProviderID) bool {` |
| P11 | `Profile` | `llmprovider/catalog/model_profile.go:5` | `type Profile int` |
| P12 | the capable order | `llmprovider/catalog/model_ranking.go:195` | `func (n rankNorm) capableOrder(a, b rankCandidate) int {` |
| P13 | the band's loop | `llmprovider/catalog/model_ranking.go:252` | `func rankRecommended(` |
| P14 | Kilo's candidate | `llmprovider/catalog/model_ranking.go:331` | `func kiloCandidate(e kiloCatalogEntry, now time.Time) rankCandidate {` |
| P15 | the listing cache entry | `llmprovider/catalog/listing_cache.go:35` | `type listingEntry struct {` |
| P16 | Gemini's listing facts | `llmprovider/catalog/listing_metadata.go:107` | `func geminiInfo(id string, m geminiModel) ModelInfo {` |
| P17 | the snapshot's ranked lists | `llmprovider/internal/cmd/catalogs/main.go:75` | `var snapshotFiles = map[llmprovider.ProviderID]string{` |
| P18 | the latest snapshot | `llmprovider/catalog/snapshot_20261012_test.go:31` | `func latestSnapshot(t *testing.T) string {` |
| P19 | `ModelInfo` | `llmprovider/catalog/model_info.go:17` | `type ModelInfo struct {` |
| P20 | `AddThinking` | `llmprovider/internal/wire/messages/messages.go:439` | `func AddThinking(body map[string]any, model, effort string, budget, maxTokens int) int {` |
| P21 | `ThinkingConfig` | `llmprovider/internal/wire/generatecontent/generatecontent.go:393` | `func ThinkingConfig(model, effort string, budget int) map[string]any {` |
| P22 | the Chat Completions body | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:344` | `func Body(model string, maxTokens int, input []llmprovider.Item, o Opts) map[string]any {` |
| P23 | `catalogFrom` | `llmprovider/catalog/discovery.go:192` | `func catalogFrom(usable []string, fetchErr error, static []string, curate curator, limit int) Catalog {` |
| P24 | `metadataCurate` | `llmprovider/catalog/model_metadata.go:684` | `func metadataCurate(provider llmprovider.ProviderID, profile Profile` |
| P25 | the first-party dispatch | `llmprovider/catalog/discovery.go:251` | `func modelCatalogFor(` |
| P26 | the static-equals-snapshot test | `llmprovider/catalog/snapshot_20261012_test.go:124` | `func TestStatic_EqualsLatestSnapshot(t *testing.T) {` |
| P27 | the wire scenarios run with metadata off | `llmprovider/internal/wirecase/wirecase.go:330` | `llmprovider.WithoutModelMetadata()` |
| P28 | the ChatGPT listing's sort | `llmprovider/providers/openai/chatgpt_listing.go:126` | `slices.SortStableFunc(listed, func(a, b ranked) int {` |
| P29 | the ChatGPT listing's filter | `llmprovider/providers/openai/chatgpt_listing.go:153` | `if model.Supported != nil && !*model.Supported {` |
| P30 | the ChatGPT session check | `llmprovider/providers/openai/chatgpt_session.go:31` | `func isChatGPTSession(src llmprovider.TokenSource) bool {` |
| P31 | the ChatGPT branch of `ListModels` | `llmprovider/providers/openai/openai.go:356` | `if p.chatGPT {` |
| P32 | Claude's listing page | `llmprovider/catalog/discovery.go:461` | `func fetchClaudePage(ctx context.Context, endpoint string` |
| P33 | Gemini's listing page | `llmprovider/catalog/discovery.go:360` | `func fetchGeminiPage(ctx context.Context, endpoint string` |
| P34 | Ollama's listing | `llmprovider/catalog/discovery.go:504` | `func fetchOllamaNames(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P35 | Kilo's listing | `llmprovider/catalog/discovery.go:914` | `func fetchKiloCatalog(ctx context.Context, token llmprovider.Token, cfg config) ([]kiloCatalogEntry, error) {` |
| P36 | Together's listing | `llmprovider/catalog/discovery.go:818` | `func fetchTogetherUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P37 | Hugging Face's listing | `llmprovider/catalog/discovery.go:718` | `func fetchHuggingFaceUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P38 | OpenCode's listing | `llmprovider/catalog/discovery.go:683` | `func fetchOpencodeUsable(ctx context.Context, gateway llmprovider.ProviderID` |
| P39 | the credential renewal | `llmprovider/catalog/discovery.go:114` | `func listAndRecord(ctx context.Context, id llmprovider.ProviderID` |

## Execution Record

No phase has run. Each phase appends its record here: what it did, its FAIL
and plant lines, its gate summary, and a dated entry for every deviation.
