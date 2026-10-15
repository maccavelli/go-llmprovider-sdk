---
status: in-progress
date: 2026-10-14
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
| `catalog` | `ModelReasoning.ReplayField`, `FieldReasoningReplay` (0041-MADR A3; Phase 1, deviation 3) | 1 |
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
  * the gate, `tools/gate.py`, at 23 of 23.
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
   * ~~**Arguments:** `claims.py RECORD.md [NAME=DIR@REV ...]`.~~ *(Amended
     2026-10-13 by 0040-PLAN-native-tool-deferral.md's Phase 0 deviation 1,
     which built the tool: each record names its revisions in a
     `<!-- claims: NAME=REV ... -->` marker, other checkouts are found beside
     this repository, and `--repo`, `--rev`, `--json`, `--fixes` and
     `--changed` serve people and agents alike. See `tools/README.md`.)*
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
     * ~~`gate.py` gains it as a check on the changed records, which makes 24
       checks;~~
     * ~~`make gate-selftest` covers the new check.~~

     *(Amended 2026-10-13, 0040-PLAN Phase 0 deviation 1: `gate-selftest`
     tests `scripts/`, not `tools/`; and a gate check cannot read claims
     pinned to past revisions and other checkouts. The tool is run by hand,
     by a person or an agent, before a record is staged, and `make
     tools-test` tests it. The gate stays at 23 checks.)*
   * **Plant:** the line comparison off by one, which a test must catch.
3. **`llmprovider/catalog/evidence_0041_test.go`.** Offline, on
   `testdata/ranking-2026-10-12`, with the clock pinned at its
   `recorded_at`. It asserts each number 0041-MADR states, through the
   code's own functions:

   | Test | Asserts (0041-MADR) |
   | :--- | :--- |
   | `TestEvidence0041_ClaudeAdaptive` | R1: `claudeAdaptiveOnly` equals "no `budget_tokens` option" on 34 of 34 Claude reasoning models in `anthropic`, `opencode` and `opencode-go` (§3) |
   | `TestEvidence0041_GeminiLegacy` | R4: 27 of 29, the two others being `gemini-2.5-flash-image` and `gemini-2.5-computer-use-preview-10-2025` (§3) |
   | `TestEvidence0041_GrokMenus` | R6: 5 of 7 reasoning models agree; `grok-4.3` and `grok-4.20-multi-agent-0309` differ as stated (§3). *In `llmprovider/providers/grok/evidence_0041_test.go` (Phase 0, deviation 2).* |
   | `TestEvidence0041_OutputLimits` | R7: ~~52 models below 8192 across the sections~~ 52 entries below 8192 across the sections, 10 of them image and video models at a stated 0, so 42 text models (0041-MADR A1); the smallest Claude limit is 64,000 (§3) |
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
   | `TestAgreement0041_*` (the offline half of D7) | on the committed snapshot, R1, R2, R4, R5 and R6 from facts equal the rule for every model both answer, except the listed set. ~~The set is `grok-4.3`, `grok-4.20-multi-agent-0309` and the two Gemini models of §3, each with its reason~~ *The set is 0041-MADR A2's (Phase 1, deviation 1): `grok-4.3` and `grok-4.20-multi-agent-0309` (R6); four Gemini image models (R4, R5); and the models whose facts are silent, §3's two among them, where the rule decides* |
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
   | `TestFirstPartyRankByFacts_0041` | the four first-party providers' sixes, on their sections, equal D4's table; with metadata off they equal today's static order *(Phase 3: the static list leads, in order, and the name curator fills the six, as it did before; `TestFirstPartyListRanksByFacts_0041` was added for `List` itself, deviation 2)* |
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
     2. the uncaveated entries, newest first by release date, ~~then in
        listing order,~~ then by id *(0041-MADR A4: a dated entry before
        an undated one; listing order is no key)*;
     3. the caveated entries, in the same order.
   * **`Recommended`** is the menu's first two groups' ids. `WithLimit(n)`
     cuts it. *(A4: the second group includes the profile's exclusions
     and Together's ids models.dev does not list. The band is curated
     from the uncaveated ids, so no caveated id is ranked.)*
   * **The static path:** `Menu` holds the static ids, with what facts are
     cached. *(Phase 4: in the static order, each ranked by its place,
     and `Recommended` stays equal to `Usable`, 0038-MADR D3. No static
     id has a caveat on the snapshot.)*
   * **Ollama:** listing order, with `CaveatNoTools` where its `/api/show`
     capabilities say so. *(A4: `List` asks `/api/show` for each
     installed model not cached, four at a time, within its 10 s bound,
     and caches the replies; a failed reply leaves no caveat. The entries
     with `CaveatNoTools` follow the others, in listing order, and are
     not in `Recommended`.)*
4. **Red first:**

   | Test | Want |
   | :--- | :--- |
   | `TestMenu_Order_0041` | on a fixture: band by rank, then newest first, then listing order, then id; caveated last |
   | `TestMenu_Deterministic_0041` | the same input in a shuffled listing order gives the same menu, 100 times, under `-shuffle=on` |
   | `TestMenu_Caveats_0041` | each caveat from its fact; a caveated entry is in `Menu`, not in `Recommended` |
   | `TestMenu_Snapshot_0041` | on the 2026-10-12 snapshot, each provider's `Menu` holds exactly `Usable`, and OpenAI's section-based full list holds no deprecated model (the evidence test's 11 becomes 0) *(A4: and each full list's length, measured and pinned, with the profile's exclusions and Together's unlisted ids in it)* |
   | `TestRecommended_Prefix_0041` | `WithLimit(n)` is a prefix of the unlimited list for n = 1 to 8; `WithLimit(MaxListed)` is the band |
   | 0038's tests | each assertion A4 changes is updated and recorded; the rest pass unchanged *(0038-MADR A5: `TestWithLimit_FullList`'s expected set and lengths; `fullList`, A1's builder, gives way to the menu, and `TestFullList_SixFirst` with it; the Ollama provider's test server counts generation requests only, as `/api/show` is a POST)* |
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

<!-- claims: self=2c82cf1 -->

Checked against go-llmprovider-sdk `3e81e3d`, at that commit, by the
checker Phase 0 commits as `tools/claims.py`. The first run found three
rows a line off; they were corrected, and the second run passed every row.

*Re-pinned 2026-10-13 to `2c82cf1`, in Phase 0, after 0040-PLAN moved
the lines of ten rows (P1–P5, P7, P9, P20, P27, P31). Each row's text is
unchanged; only its line number moved. See Phase 0's record.*

| Id | Location | Where | Matched |
| :--- | :--- | :--- | :--- |
| P1 | Claude's thinking call | `llmprovider/providers/claude/claude.go:353` | `messages.AddThinking(body, model, string(effort), budget, maxTokens)` |
| P2 | OpenCode's Messages thinking call | `llmprovider/providers/opencode/opencode.go:572` | `messages.AddThinking(body, c.model, string(c.reasoning.Effort), c.reasoning.Budget, maxTokens)` |
| P3 | OpenCode's Google thinking call | `llmprovider/providers/opencode/opencode.go:592` | `generatecontent.ThinkingConfig(c.model, string(c.reasoning.Effort), c.reasoning.Budget)` |
| P4 | Gemini's level | `llmprovider/providers/gemini/gemini.go:232` | `if level := thinkingLevel(model, r.Effort); level != "" {` |
| P5 | Grok's clamp | `llmprovider/providers/grok/grok.go:300` | `clamped := clampReasoningEffort(model, string(effort))` |
| P6 | the output limit setting | `llmprovider/settings.go:289` | `func (st *Settings) MaxTokens() int { return st.s.cfg.MaxTokens }` |
| P7 | Kilo's replay | `llmprovider/providers/kilo/kilo.go:325` | `ReplayReasoningDetails: true,` |
| P8 | `CachedModel` | `llmprovider/catalog/lookup.go:94` | `func CachedModel(` |
| P9 | OpenCode reads the cached document | `llmprovider/providers/opencode/opencode.go:474` | `meta, cached = catalog.CachedMetadataWith(ctx, p.gateway, p.listing...)` |
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
| P20 | `AddThinking` | `llmprovider/internal/wire/messages/messages.go:558` | `func AddThinking(body map[string]any, model, effort string, budget, maxTokens int) int {` |
| P21 | `ThinkingConfig` | `llmprovider/internal/wire/generatecontent/generatecontent.go:393` | `func ThinkingConfig(model, effort string, budget int) map[string]any {` |
| P22 | the Chat Completions body | `llmprovider/internal/wire/chatcompletions/chatcompletions.go:344` | `func Body(model string, maxTokens int, input []llmprovider.Item, o Opts) map[string]any {` |
| P23 | `catalogFrom` | `llmprovider/catalog/discovery.go:192` | `func catalogFrom(usable []string, fetchErr error, static []string, curate curator, limit int) Catalog {` |
| P24 | `metadataCurate` | `llmprovider/catalog/model_metadata.go:684` | `func metadataCurate(provider llmprovider.ProviderID, profile Profile` |
| P25 | the first-party dispatch | `llmprovider/catalog/discovery.go:251` | `func modelCatalogFor(` |
| P26 | the static-equals-snapshot test | `llmprovider/catalog/snapshot_20261012_test.go:124` | `func TestStatic_EqualsLatestSnapshot(t *testing.T) {` |
| P27 | the wire scenarios run with metadata off | `llmprovider/internal/wirecase/wirecase.go:366` | `llmprovider.WithoutModelMetadata()` |
| P28 | the ChatGPT listing's sort | `llmprovider/providers/openai/chatgpt_listing.go:126` | `slices.SortStableFunc(listed, func(a, b ranked) int {` |
| P29 | the ChatGPT listing's filter | `llmprovider/providers/openai/chatgpt_listing.go:153` | `if model.Supported != nil && !*model.Supported {` |
| P30 | the ChatGPT session check | `llmprovider/providers/openai/chatgpt_session.go:31` | `func isChatGPTSession(src llmprovider.TokenSource) bool {` |
| P31 | the ChatGPT branch of `ListModels` | `llmprovider/providers/openai/openai.go:404` | `if p.chatGPT {` |
| P32 | Claude's listing page | `llmprovider/catalog/discovery.go:461` | `func fetchClaudePage(ctx context.Context, endpoint string` |
| P33 | Gemini's listing page | `llmprovider/catalog/discovery.go:360` | `func fetchGeminiPage(ctx context.Context, endpoint string` |
| P34 | Ollama's listing | `llmprovider/catalog/discovery.go:504` | `func fetchOllamaNames(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P35 | Kilo's listing | `llmprovider/catalog/discovery.go:914` | `func fetchKiloCatalog(ctx context.Context, token llmprovider.Token, cfg config) ([]kiloCatalogEntry, error) {` |
| P36 | Together's listing | `llmprovider/catalog/discovery.go:818` | `func fetchTogetherUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P37 | Hugging Face's listing | `llmprovider/catalog/discovery.go:718` | `func fetchHuggingFaceUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {` |
| P38 | OpenCode's listing | `llmprovider/catalog/discovery.go:683` | `func fetchOpencodeUsable(ctx context.Context, gateway llmprovider.ProviderID` |
| P39 | the credential renewal | `llmprovider/catalog/discovery.go:114` | `func listAndRecord(ctx context.Context, id llmprovider.ProviderID` |

## Execution Record

~~No phase has run.~~ Each phase appends its record here: what it did, its
FAIL and plant lines, its gate summary, and a dated entry for every
deviation.

### Phase 0: records, the claims check, and the evidence tests (2026-10-13)

#### Records (step 1)

* The owner approved this PLAN ("proceed with 0041"). It is
  `in-progress`, and `docs/README.md` says so.
* **The evidence index was re-pinned** from `3e81e3d` to `2c82cf1`, as the
  rule "Ordering with 0040-PLAN" requires of the plan that runs second.
  * At `HEAD`, `tools/claims.py` found ten PLAN rows on moved lines (P1–P5,
    P7, P9, P20, P27, P31). It wrote the fixes for nine. P31's text, `if
    p.chatGPT {`, now occurs six times, and its `ListModels` branch was read
    as line 404 by hand.
  * The marker became `self=2c82cf1`, with a dated note. All 39 rows pass.
  * **0041-MADR stays at `3e81e3d`,** where its 57 rows pass. Seven of its
    rows would also have moved. But its evidence is what the decision was
    made on, and its prose names that commit, so it is not re-pinned.

#### `tools/claims.py` (step 2)

**Skipped,** as step 1 provides. 0040-PLAN Phase 0 committed the tool
(`0af9ac7`), and it runs as it stands.

#### Deviations

1. **2026-10-13: R7's count.**
   * **Found** while measuring §3 through the SDK's own functions on the
     snapshot. 52 entries in the eight sections record `limit.output` below
     8192, but 10 of them are image and video models with a stated `0`
     (five `gpt-image`/`chatgpt-image`, five `grok-imagine`). Only 42 text
     models allow less than 8192.
   * **Decided by the owner:** amend with the split. 0041-MADR gains
     amendment A1; R7's row and the Consequences line are annotated, and so
     is this PLAN's row for `TestEvidence0041_OutputLimits`. The test
     asserts 52, 10 and 42.
2. **2026-10-13: the Grok test's package.**
   * **Found:** R6's rule, `effortMenu`, is unexported in `providers/grok`,
     and `grok` imports `catalog`. A `catalog` test that imported `grok`
     would be an import cycle.
   * **Decided by the owner:** the test lives in
     `llmprovider/providers/grok/evidence_0041_test.go`. It calls
     `effortMenu` directly, and reads the committed snapshot from
     `catalog`'s testdata. The other seven stay in
     `llmprovider/catalog/evidence_0041_test.go`. The row in step 3's
     table is annotated.

#### Research (step 3)

A scratch probe on a scratch clone measured each number through the SDK's
own functions before any test was written. R1 used `messages.AddThinking`
and R4 `generatecontent.ThinkingConfig` at low effort; §4 used `curateOpenAI`
and `curateGrok`, and §5 `capableOrder`'s keys before cost:

| 0041-MADR | Measured |
| :--- | :--- |
| R1 | `anthropic` 17 of 17, `opencode` 16 of 16, `opencode-go` 1 of 1: 34 of 34 |
| R4 | 27 of 29; `gemini-2.5-computer-use-preview-10-2025` and `gemini-2.5-flash-image` differ |
| R7 | 52 below 8192, 10 at a stated 0 (deviation 1); the smallest Claude limit 64,000 |
| §4 | OpenAI: section 53, full list 42, 11 deprecated, `gpt-3.5-turbo` at #32 without tools; Grok: 8, the last `grok-4.20-multi-agent-0309` without tools |
| §5 | OpenAI 13 of 276, Claude 3 of 136, Together 0 of 78, Kilo 144 of 23,220 (216 eligible: the snapshot, not the live document) |
| §7 | `0.99 × 2 + 0.01 × 10 = 2.08`; cached, 0.476 to 0.526 |

R6 is measured in its own package (deviation 2).

#### The evidence tests (step 3)

* **`llmprovider/catalog/evidence_0041_test.go`** holds seven. Each reads
  the snapshot's models.dev document with `decodeModelMetadata`, and pins
  the ranking clock at its `recorded_at`:
  * `TestEvidence0041_ClaudeAdaptive` (R1). `messages.AddThinking` sends
    `{"type":"adaptive"}` exactly where models.dev lists no `budget_tokens`
    option: 17, 16 and 1, so 34 of 34.
  * `TestEvidence0041_GeminiLegacy` (R4). `generatecontent.ThinkingConfig`
    at low effort sends a `thinkingBudget` exactly where models.dev lists a
    budget and no effort: 27 of 29, the two named.
  * `TestEvidence0041_OutputLimits` (R7, A1): 52, 42, the ten ids at 0,
    and 64,000. The default is read from `ResolveOptions(…).MaxTokens()`,
    not written as 8192.
  * `TestEvidence0041_FullListFlags` (§4), through `curateOpenAI` and
    `curateGrok`.
  * `TestEvidence0041_ListingDates` (§4's table). It reproduces the
    research script's measure: a row with no `created`, or no models.dev
    `release_date`, is not dated, and within 31 days counts as matched.
    Every number the table states holds. The table gives no count of
    zero-dated Kilo rows; the test asserts the 5 measured, and says so.
  * `TestEvidence0041_CostKeyReach` (§5), through `metadataCandidate`,
    `eligible`, `newRankNorm` and `capableOrder`'s keys before cost.
  * `TestEvidence0041_CachedTurn` (§7): D5's formula, `cachedTurn0041`,
    gives 2.08 uncached, and 0.476 to 0.526 cached. Phase 5 replaces it
    with `ProfileAgent`'s own function.
* **`llmprovider/providers/grok/evidence_0041_test.go`** holds
  `TestEvidence0041_GrokMenus` (R6, deviation 2). `effortMenu` equals
  models.dev's effort values on 5 of 7. `grok-4.3` and
  `grok-4.20-multi-agent-0309` differ exactly as §3 states.
* **No red run.** These tests assert behaviour that exists, and no code
  changes in this phase, so there is nothing to be red against. Their
  plants are their proof of failing.
* **One expectation fixed while writing.** The first run failed only on
  Kilo's zero-dated count, which the test had assumed to be 0:

  ```text
  kilo.json: {… zeroes:5 …}; want {… zeroes:0 …}
  ```

  The table states no such count, and every number it does state held.

#### Plants (`tools/plant.py`, 8 of 8 caught)

Each plant changes one test's function, or its input, by about one model.
Every plant compiles, and each was caught by an assertion.

| Plant | FAIL line |
| :--- | :--- |
| R1: Claude 4.7 no longer adaptive (`minor >= 8`) | `anthropic/claude-opus-4-7: adaptive false, but models.dev's budget_tokens options are []`; `anthropic: 16 of 17 …` |
| R4: Gemini 2 no longer legacy | `R4: 26 of 29 agree, differing [gemini-2.5-flash gemini-2.5-flash-lite gemini-2.5-pro]` |
| R7: the default output limit 8193 | `R7: 69 below 8193, 59 of them text models …` |
| §4: `gpt-3.5-turbo` renamed in the snapshot | `OpenAI: section 53, full list 41, 10 deprecated, gpt-3.5-turbo at #0` |
| §4's table: one zero-dated Kilo row dated | `kilo.json: {… distinct:369 zeroes:4 within:365 dated:383}` |
| §5: age no longer orders | `openai: 196 of 276 pairs reach the cost key`; `claude: 94 of 136 …` |
| §7: the new input priced as a read | `cached: 0.298 to 0.298; want both within [0.47, 0.53]` |
| R6: `grok-4.5` gains `xhigh` | `R6: 4 of 7 agree, differing map[… grok-4.5:[[high low medium xhigh] [high low medium]]]` |

#### Checks (step 4)

* `go test ./llmprovider/catalog -run Evidence0041`, and the Grok test:
  all pass.
* `make pre-add-check` on the two test files: clean.
* `make tools-test`: 41 tests, OK.
* `make records-check`, and `tools/claims.py` on both records: 57 of 57
  (0041-MADR, at `3e81e3d`) and 39 of 39 (this PLAN, at `2c82cf1`).
* `tools/denyscan.py --changed` and `tools/linkcheck.py`: clean.
* **Gate:** `tools/gate.py`, 23 of 23 passed.

### Phase 1: facts at request time (2026-10-14)

#### Research

* **P8, `CachedModel`.** It merges the caller's layers, the cached listing
  and the cached document, and starts any missing fetch. `CachedFacts` is
  the same merge with no listing fetch. The listing cache is keyed by
  provider, base URL and a fingerprint of the token, so `CachedFacts` needs
  the credential. `Settings.TokenSource()` gives it from the options
  (`WithAPIKey` or `WithTokenSource`), as the PLAN's signature implies, and
  `config` now holds it.
* **P9, OpenCode's `requestRoute`,** reads the cached document
  (`CachedMetadataWith`), which starts a missing fetch for the next call
  without waiting. That is D1's "no request waits".
  * OpenCode's `chatReplayField` instead blocks on `LookupMetadataWith`.
  * Step 4 says the Chat Completions adapters replay "as OpenCode's
    `chatReplayField` does". It is read, by D1, as the same field from the
    same document, read from the cache without waiting. OpenCode's own is
    unchanged.
* **The replay field is not a `ModelInfo` field.** `Metadata`'s
  `InterleavedField` gives it. Together, Hugging Face and Ollama read it
  from `CachedMetadataWith`.
* **Every provider keeps its options** in `p.listing`, which `CachedFacts`
  takes.
* **P27:** the wire scenarios run with `WithoutModelMetadata`, and no
  scenario caches a listing at their base URL, so the goldens keep the rule
  path.
* **A module-internal field beyond the PLAN's list.**
  `messages.Thinking` gains `MaxOutput`. D2 says a thinking budget's raise
  of `max_tokens` never passes the model's limit, and Anthropic requires
  `max_tokens` above `budget_tokens`. Only the wire function sees both. It
  is not API, and decides nothing new.
* **Import cycles.**
  * `catalog`'s agreement test can call `messages` and `generatecontent`,
    which do not import `catalog`.
  * R5's and R6's rules are unexported in `gemini` and `grok`, which do
    import `catalog`. Their agreement tests live in those packages, as
    Phase 0's deviation 2 did for R6.

#### Deviations

1. **2026-10-14: the agreement set.**
   * **Found:** the agreement tests measured a set other than the one step
     5 predicts. R6 is exactly `grok-4.3` and `grok-4.20-multi-agent-0309`,
     and R1 and R2 differ on no model. But on Gemini:
     * §3's two models do not differ. Their facts are silent (reasoning,
       with no budget and no effort), so the rule decides for them.
     * Four image models do differ: `gemini-3-pro-image` (R5), and
       `gemini-3.1-flash-image`, `gemini-3.1-flash-image-preview` and
       `gemini-3.1-flash-lite-image` (R4 and R5). models.dev lists their
       efforts as `minimal` and `high`, so D2 sends no level where the rule
       sends one the model does not list.
   * **Decided by the owner:** the facts win, as D2 decides, and the
     measured set is recorded with its reasons. 0041-MADR gains amendment
     A2. Step 5's row is annotated. The tests assert the exact differing
     sets and the exact silent sets:
     * Google, silent: `deep-research-max-preview-04-2026`,
       `deep-research-preview-04-2026`,
       `gemini-2.5-computer-use-preview-10-2025`, `gemini-2.5-flash-image`,
       `gemini-3-pro-image-preview`, `gemini-3.1-flash-tts-preview`,
       `gemini-omni-flash-preview`, `gemma-4-26b-a4b-it` and
       `gemma-4-31b-it`;
     * the Messages route, silent: `opencode-go/minimax-m2.7`,
       `opencode-zen/minimax-m2.1-free` and
       `opencode-zen/minimax-m2.5-free`.
2. **2026-10-14: a request starts no fetch.**
   * **Found:** wired as step 2 says, `CachedFacts` started the models.dev
     fetch whenever the document was missing or stale. A race detector run
     caught `TestGrok_TokenSourceCalledPerRequest` recording that
     background fetch through its counting transport, from another
     goroutine:
     * `session_test.go:92`, written by `catalog/model_metadata.go:517`,
       from `lookup.go:145`;
     * read by `post.go:144`.

     It depends on test order: whichever test first meets an uncached
     document triggers the fetch.
     * The eight providers' test helpers do not turn metadata off, as
       OpenCode's do (0015-PLAN S10), so their tests would reach
       models.dev.
     * In production, a first `Generate` would fetch ~1 MB through the
       caller's client.
   * **Decided by the owner:** amend D1. A request reads the cached document
     and starts no fetch (0041-MADR A3). `CachedFacts` reads the caches
     only. Step 2's sentence "With no document cached, it starts that fetch
     only" no longer holds. Step 5's `TestCachedFacts_NoListingFetch_0041`
     asserts no document request either.
3. **2026-10-14: the replay field is a fact.**
   * **Found:** the Chat Completions replay field (`interleaved.field`) is
     not a `ModelInfo` field. `CachedMetadataWith`, its only read, starts a
     fetch, and OpenCode's route depends on that.
   * **Decided by the owner:** `ModelInfo.Reasoning.ReplayField`, with
     `FieldReasoningReplay` (0041-MADR A3). The two are added to "The
     exported additions". `Metadata.Model` sets it from the document, and
     Together, Hugging Face and Ollama read it from `CachedFacts`.

#### What was built

* **`catalog`:**
  * `CachedFacts(ctx, provider, model, opts...) (ModelInfo, bool)`. It
    merges the caller's layers, a listing already cached for the provider,
    base URL and credential, and the cached document (`peekMetadata`, which
    starts no fetch; A3). `config` holds the options' token source, so the
    listing is found by its credential.
  * `ModelReasoning.ReplayField` and `FieldReasoningReplay`, set from
    `interleaved.field` by `modelInfoFrom`. `interleavedField` reads
    through the same `interleavedFieldOf`, so the two never differ.
* **`messages`:** `Thinking` (`Known`, `Adaptive`, `EffortOnly`,
  `BudgetMin`, `BudgetMax`, and `MaxOutput`, which research added),
  `FactsThinking` and `AddThinkingFor`. `AddThinking` is `AddThinkingFor`
  with no facts.
* **`generatecontent`:** `Thinking` (`Known`, `Budget`, `Levels`),
  `FactsThinking` and `ThinkingConfigFor`.
* **The providers.** Each reads its facts once per request, with the
  request's own token: `WithAPIKey(token.Value)`, after its options.
  * Gemini, OpenAI, Together, Hugging Face, Ollama and Grok now build
    `Stream`'s call per token, through `StreamEventsFor`.
  * Claude builds through `Send` and `StreamEventsRetry`, and Kilo already
    built per token.
  * OpenCode reads its facts in `resolve`; its gateways are not listing
    layers, so no token is needed.
  * **What each reads:**
    * every provider, the output limit;
    * Claude, and OpenCode's messages route, the thinking shape;
    * Gemini's `thinkingLevelFor`, and OpenCode's google route, the level;
    * Grok's `clampReasoningEffortFor`, the efforts;
    * Together, Hugging Face and Ollama, the replay field.
* **Docs:**
  * each provider's package doc names the facts it reads and its fallback
    (R12);
  * `docs/architecture.md` names `CachedFacts` and the wire functions, and
    says what a request reads (R43).

#### Found while building

* **The token for the listing.** As first written, `CachedFacts` asked the
  options' token source for a token. That was a second call per request:
  `TestGrok_TokenSourceCalledPerRequest` panicked when its two-token source
  ran out. A rotating source would hand the request the wrong token, and a
  `CommandToken` would run its command twice. Each provider now passes the
  token it is sending with, and `CachedFacts`'s doc says so. The panic is
  the plant "Grok's facts read through its token source" below.
* **A wiring test the plants showed missing.** "Gemini's levels ignored"
  passed the first plant run only because of the race that deviation 2
  removed. On the second run it was not caught. `TestThinkingLevel_Wired_0041`
  was added, and the plant then failed it:

  ```text
  level_0041_test.go:83: facts true, effort medium: thinking_level medium; want <nil>
  ```

* **Not this phase's:** the 0034 learned-refusal tests of Claude and
  OpenCode fail under `-count=3`, since their learned memo is the
  process's. They fail the same way on a clean clone of `0eef583`, and
  pass at `-count=1`, as the gate runs them. Nothing was changed for them.

#### Tests, red and green

* **Red, the wire and the catalog** (before the code):

  ```text
  undefined: CachedFacts
  undefined: AddThinkingFor
  undefined: ThinkingConfigFor
  undefined: thinkingLevelFor
  undefined: clampReasoningEffortFor
  ```

  The undefined `messages.Thinking`, `messages.FactsThinking` and
  `generatecontent.FactsThinking` failed the same way.
* **Red, the providers**, run before any was wired: every provider sent the
  8192 default for a model whose limit is 4096, and neither Together nor
  Hugging Face replayed under the field.

  ```text
  facts_0041_test.go:97: claude/claude-sonnet-5, a limit of 4096: sent 8192 (found true); want 4096
  ```

  This failed for all 12 rows, and so did:

  ```text
  facts_0041_test.go:158: together, interleaved.field "reasoning_content": the reasoning was replayed under ""; want "reasoning_content"
  efforts_0041_test.go:92: models.dev only: sent effort xhigh; want "high"
  ```

  The 128,000 rows passed already, as they should: the default is never
  raised.
* **Fixed while writing:** `sentLimit` first read `float64`, but `wiretest`
  keeps numbers exact as `json.Number`, so every row read "found false".
* **Added after the code**, and shown failing only by the plants:
  * `TestModelReasoning_ReplayField_0041` (A3);
  * `TestThinkingLevel_Wired_0041`;
  * the A2 sets in the agreement tests.
* **Green:**
  * `TestCachedFacts_NoListingFetch_0041` and
    `TestCachedFacts_Precedence_0041`;
  * `TestAddThinkingFor_0041`, `TestThinkingConfigFor_0041` and both
    `TestFactsThinking_0041`;
  * `TestGrokEfforts_Facts_0041` and `TestThinkingLevel_Facts_0041`;
  * `TestOutputLimit_Facts_0041` and `TestReplayField_Facts_0041`;
  * the five `TestAgreement0041_*`;
  * `go test ./...`;
  * every wire golden, three runs, with no golden changed;
  * `go test -race -shuffle=on ./llmprovider/...`, once, and Grok's former
    race ten times under `-race`.

#### Plants (`tools/plant.py`, 18 of 18 caught)

Every plant compiles; two that first did not were rewritten in compiling
forms. Each was caught by an assertion, or by the source's panic for the
token plant.

| Plant | FAIL line |
| :--- | :--- |
| `CachedFacts` calls the listing fetch | `CachedFacts on a cold cache knew the model` |
| `CachedFacts` starts the document fetch | the same |
| the precedence inverted | `field 4: 128000 from "models.dev"; want 100000 from "listing:claude"` |
| efforts give a toggle's shape | `claude/claude-fable-5 at "low": rule […"output_config":{"effort":"low"}…], facts […]` |
| a budget option ignored | `claude/claude-opus-4-5 at "": rule [{"thinking":{"budget_tokens":4096…}}], facts [{"thinking":{"type":"adaptive"}}]` |
| the effort sent with a toggle | `a toggle alone: adaptive, no effort: {…"output_config":{"effort":"high"}…}` |
| the output cap on the raise removed | `the raise stops at the output limit: {…"max_tokens":10096}` |
| a level outside the list sent | `R4: facts and the rule differ on [] …` |
| the replay field not decoded | `a: "", known true; want "reasoning_content"` |
| Claude's output cut removed | `claude/claude-sonnet-5, a limit of 4096: sent 8192` |
| OpenCode's output cut removed | `opencode-zen/claude-sonnet-5, a limit of 4096: sent 8192` |
| Together's replay field ignored | `together, interleaved.field "reasoning_content": … replayed under ""` |
| Grok's efforts facts ignored | `models.dev only: sent effort xhigh; want "high"` |
| Gemini's levels ignored | `facts true, effort medium: thinking_level medium; want <nil>` |
| Grok's facts read through its token source | `TestGrok_TokenSourceCalledPerRequest` panics: the source ran out |
| an R6 exception removed | `R6: facts and the menu differ on [grok-4.20-multi-agent-0309 grok-4.3]; want only [grok-4.3]` |
| an R4 exception removed | `gemini-3.1-flash-image: rule {…"thinkingLevel":"low"}, facts {…}` |
| an R5 exception removed | `R5: facts and thinkingLevel differ on [gemini-3-pro-image …]; want only […]` |

#### Checks

* `make pre-add-check` on the 25 Go files: clean.
* `tools/apidiff.py v1.13.0`: `CachedFacts`, `FieldReasoningReplay` and
  `ModelReasoning.ReplayField`, all added, and nothing else, as amended by
  deviation 3.
* `make records-check`; `tools/claims.py` on both records, 57 of 57 and
  39 of 39; `tools/denyscan.py --changed` and `tools/linkcheck.py`: clean.
* **Gate:** `tools/gate.py`, 23 of 23 passed, `g-wire-stable` included: no
  existing golden changed.

### Phase 2: listing facts (2026-10-14)

#### Research

* **P15, P16, P19, the listing decoders.**
  * Each listing a `List` reads emits its models' facts through
    `config.emit`, and `listOnce` stores them by provider, base URL and
    credential.
  * Kilo's entry already has `created`; OpenAI's and xAI's share
    `dataModel`; Anthropic's `claudeModel` did not decode `created_at`.
  * Hugging Face's listing was no listing layer: its fetch emitted nothing.
* **P17, the snapshot command,** records the five ranked listings, one body
  each. Anthropic and Gemini page their listings, and the recorder refused
  more than one body. The fetcher now keeps every page:
  * `Listing` gives a listing's body only when it was one page, so the
    ranked snapshot keeps its rule;
  * `Pages` gives them all.
* **P18:** `latestSnapshot` reads `ranking-*` only, so
  `testdata/firstparty-*` moves no ranked list. The ranked tests are
  unchanged.
* **The four keys** were checked for presence only, and all were set.

#### Step 2: the first-party listings

`go run ./llmprovider/internal/cmd/catalogs -firstparty
llmprovider/catalog/testdata/firstparty-2026-10-14` read the four
listings. They are free; no request was billed. It wrote `openai.json`,
`claude.json`, `gemini.json` and `grok.json`, each an array of its pages,
and `firstparty.json`, the time and each file's SHA-256.

| Listing | Pages | Rows | Its date field |
| :--- | ---: | ---: | :--- |
| OpenAI | 1 | 133 | `created`, Unix |
| Anthropic | 1 | 14 | `created_at`, RFC 3339 |
| Gemini | 1 | 63 | none |
| xAI | 1 | 14 | `created`, Unix |

**Redaction.** Fine-tuned rows (`ft:…`, which name an organisation) are
dropped, and an `owned_by` that is not a vendor's becomes `<owner>`. The
recorded `owned_by` values are `system`, `openai`, `openai-internal` and
`xai`. No row holds `org-`, `user-`, `proj_` or `@`, and no fine-tuned row
was listed. `tools/denyscan.py` over the five files: 0 hits.

#### Step 3: the dates measured

Each listing's date against models.dev's `release_date`, as §4 measured
it: rows with both, and within 31 days. A listing qualifies at 90% or more,
with no single date on every row and no zero.

| Listing | Rows | Within 31 days | Share | Qualifies |
| :--- | ---: | ---: | ---: | :--- |
| OpenAI | 133 | 45 of 48 | 94% | yes |
| Anthropic | 14 | 14 of 14 | 100% | yes |
| xAI | 14 | 12 of 13 | 92% | yes |
| Gemini | 63 | no date field | — | no |
| Kilo | 391 | 365 of 382 | 96% | yes |
| Hugging Face | 125 | 66 of 69 | 96% | yes |
| Together | 265 | 11 of 16 | 69% | no |
| OpenCode Zen | 83 | 13 of 80 | 16% | no: one date on every row |
| OpenCode Go | 38 | 8 of 34 | 24% | no: one date on every row |

The Kilo, Hugging Face, Together, Zen and Go counts are §4's, unchanged.

#### Step 4: what was built

* **Release dates,** from the five that qualified:
  * OpenAI's and xAI's `created`, Anthropic's `created_at`, Kilo's
    `created` and Hugging Face's `created`, each as a UTC day, through
    `listingDay`;
  * a zero dates nothing;
  * `undatedConstant`, in `listOnce`, drops every date of a listing whose
    dated rows all share one. It needs two dated rows to tell, so a
    one-row listing keeps its date.
* **Hugging Face becomes a listing layer,** for its dates. Step 4 reads
  Hugging Face's date "from the listing layer", and Hugging Face had none.
  So `LookupModel` and `CachedModel` now read its listing too, as they do
  the others'. `List` already read it.
* **The canonical id:**
  * `canonical_model_id` is `ModelInfo.CanonicalID`, with
    `FieldCanonicalID`;
  * `pricesByCanonical(doc, canonical)` is that model's rates in its
    maker's first-party section (`openai`, `anthropic`, `google`, `xai`).
* **D1's fill is applied in `Metadata.Model` and `Models`,** whose result
  every layer reads: a missing `cache_read` comes from the canonical
  model's entry, with models.dev as its source. A helper only tests called
  would fail the linter's `unused`. On the snapshot it fills one model:
  Kilo's `openai/gpt-3.5-turbo`.
* **Expiry:** OpenAI's `shutdown_date` and Kilo's `expiration_date` were
  already `Retires`. The first-party test checks OpenAI's on the recorded
  listing.
* **`docs/architecture.md`:**
  * which listing gives a release date;
  * the canonical id;
  * the command's `-snapshot` and `-firstparty` (R43).

#### Tests, red and green

* **Red:**

  ```text
  firstparty_0041_test.go:70: exit 2: flag provided but not defined: -firstparty
  listingdates_0041_test.go:188:11: info.CanonicalID undefined (type ModelInfo has no field or method CanonicalID)
  listingdates_0041_test.go: undefined: FieldCanonicalID, pricesByCanonical
  ```

  Once the canonical id built, before the dates were decoded:

  ```text
  listingdates_0041_test.go:81: openai, two dates: Released 0001-01-01 00:00:00 +0000 UTC from ""; want from the listing true
  listingdates_0041_test.go:126: openai/gpt-4.1: Released 0001-01-01 00:00:00 +0000 UTC, known false; want known true
  ```

  The same failed for Anthropic, xAI, Kilo and Hugging Face, and for the
  recorded Anthropic and xAI listings. The rows that want no date passed
  already, as they should.
* **Fixed while writing:**
  * the command's placeholder first came out `<owner>`, since
    `json.Marshal` escapes HTML. `marshalPlain` writes it as given;
  * the first `make pre-add-check` found an unchecked type assertion
    (`errcheck`).
* **Green:**
  * `TestFirstParty_0041`;
  * `TestListingDates_0041`, `TestListingDates_FirstParty_0041` and
    `TestCanonicalID_0041`;
  * `go test ./...`;
  * every wire golden, three runs, with no golden changed, and no ranked
    list moved.

#### Plants (`tools/plant.py`, 9 of 9 caught)

| Plant | FAIL line |
| :--- | :--- |
| the guard for one date on every row removed (PLAN) | `openai, one date on every row: Released 2026-05-28 … from "listing:openai"; want from the listing false` |
| Together, which did not qualify, given a date (PLAN) | `together, two dates: Released 2026-09-02 … from "listing:together"; want from the listing false` |
| the canonical lookup on the wrong section (PLAN) | `openai/gpt-x: cache_read 0, known false; want 0.25, true` |
| Hugging Face not a listing layer | `huggingface, two dates: Released 0001-01-01 … from ""; want from the listing true` |
| Anthropic's `created_at` ignored | `claude, two dates: … want from the listing true` |
| a zero read as a date | `openai, a zero: Released 1970-01-01 … from "listing:openai"` |
| an entry's own `cache_read` overwritten | `openai/gpt-x-own: cache_read 0.25, known true; want 0.5, true` |
| an organisation owner kept | `openai.json keeps "user-xyz"` |
| a fine-tuned row kept | `openai.json keeps "ft:"` |

`plant.py` takes one edit per plant. So "Together given a date" sets a date
in `togetherInfo` itself, as a read of its listing would, rather than
adding a field and reading it. A first version gave every row the same
date, which the one-date guard rightly dropped, and it was not caught. The
plant was rewritten with a date per row, and caught. The fine-tuned-row
plant was rerun after the `errcheck` fix, and caught.

#### Checks

* `make pre-add-check` on the 9 Go files: clean after the fix above.
* `tools/apidiff.py 6253cc8`: `ModelInfo.CanonicalID` and
  `FieldCanonicalID`, added, and nothing else, as "The exported additions"
  lists for Phase 2.
* `make records-check`; `tools/claims.py` on both records, 57 of 57 and
  39 of 39; `tools/linkcheck.py`: clean.
* **The identifier scan's first run** found 6 hits, all in `catalogs`, a
  Mach-O binary that a `go build` of the command wrote into the repository
  root during this phase. It was never staged. It was removed, and the scan
  then found 0 hits in 16 files.
* **Gate:** `tools/gate.py`, 22 of 23 on the first run (`deny-scan`, the
  binary above), then 23 of 23. `g-wire-stable` was included: no existing
  golden changed.

### Phase 3: the band (2026-10-14)

#### Research

* **P10–P13, the ranking.**
  * `rankCandidate.eligible` holds 0009-MADR §3's exclusions.
  * `rankGroup` is the id's leading letters on every provider, so on
    OpenAI `o3-pro` and `o4-mini` share a group with nothing, and the
    per-group cap of two never meets them.
  * `isSmallModel` is the name regex alone.
* **P14, `kiloCandidate`,** reads Kilo's own per-token prices, which D4
  reads the same way, per million tokens.
* **P24, `metadataCurate`,** builds the candidates from the document's
  section and falls back to the curator it is given.
* **P25.** `modelCatalogFor` gave each first-party provider its name
  curator (`curateOpenAI`, `curateClaude`, `curateGemini`, `curateGrok`),
  and never fetched the document for them.
* **P26.** `TestListModelCatalog_Snapshot20261012` pins the five ranked
  sixes, and `TestStatic_EqualsLatestSnapshot` holds the ranked static
  lists to its utility sixes. `TestListModelCatalog_Snapshot20260926`
  replays 0009-MADR §7, and requires that "a rule change must update §7
  and this test together".

#### What was built

* **`model_ranking.go`:**
  * `smallBlendMax = 1.0`, and `smallByPrice(input, output)`, which is
    `(3 × input + output) / 4 ≤ 1.0`. Its `ok` is false when the blend is
    not positive and finite, and the name then decides;
  * `rankCandidate.unusable`, which `eligible` excludes first;
  * `kiloCandidate` sets small from its price when it has one.
* **`model_metadata.go`:**
  * `metadataCandidate` sets small from the price, and `unusable` when
    `tool_call` is false or the output modalities lack `text`;
  * `metadataCurate` groups by `family` on the `oneVendor` providers:
    OpenAI, Claude, Gemini and Grok.
* **`discovery.go`:** the four first-party cases are one case. It starts
  the document fetch and lists through `firstPartyUsable`, then curates
  through `metadataCurate`, with `firstPartyCurator(p)` (the name curator)
  as the fallback.
* **`models_catalog.go`:** the five ranked static lists are the new
  utility sixes. The first-party static lists are unchanged; they remain
  the fallback with no listing.
* **`docs/architecture.md`:** every provider ranks by facts, and the band's
  three rules (R43).

#### The sixes against D4's table

On the committed 2026-10-12 snapshot, measured by the tests:

| Provider | D4's table | Measured |
| :--- | :--- | :--- |
| OpenCode Zen | utility 5 of 6: `gpt-6-luna` in, `gemini-3.5-flash-lite` out; capable 6 of 6 | the same |
| OpenCode Go | utility 6 of 6; capable 3 of 6: `qwen3.8-max`, `kimi-k3`, `glm-5.2` for `space-bunny`, `mimo-v2.6-pro`, `gpt-6-luna` | the same; the utility six is reordered |
| Kilo | utility 4 of 6: `minimax/minimax-m3`, `nvidia/nemotron-3-ultra-550b-a55b` for `meta/muse-spark-1.2`, `openai/gpt-6.1-sol`; capable 6 of 6 | the same |
| Hugging Face | utility 3 of 6: `tencent/Hy3` and the two `gemma-4` models for `Inkling-Small` and the two Step models; capable 6 of 6 | the same |
| Together | utility 6 of 6; capable 5 of 6: `nemotron-3-ultra` for `MiniMax-M3` | the same; the utility six is reordered |
| OpenAI | utility `gpt-6-luna`, `gpt-5.6-luna`, `gpt-5.4-nano`, `gpt-5-nano`, `gpt-5-mini`, `gpt-6.1-sol`; capable `gpt-6.1-sol`, `gpt-6-sol`, `gpt-6-astra`, `gpt-5.6-terra`, `gpt-5.5-pro`, `gpt-5.5`; no `o3-pro` | exactly these |
| Claude | utility 4 of 5 static, adds `claude-haiku-4-5`, `claude-opus-5`; capable 3 of 5, adds `claude-fable-5-1`, `claude-opus-5`, `claude-fable-5` | utility `claude-haiku-5-5`, `claude-sonnet-5-5`, `claude-sonnet-5`, `claude-haiku-4-5`, `claude-opus-5-5`, `claude-opus-5`; capable `claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1`, `claude-opus-5`, `claude-sonnet-5`, `claude-fable-5` |
| Gemini | 3 of 3 static, filled from the section | utility `gemini-3.1-flash-lite`, `gemini-3.5-flash-lite`, `gemini-2.5-flash`, `gemini-3.8-flash`, `gemini-2.5-pro`, `gemini-3.6-flash`; capable `gemini-3.8-flash`, `gemini-3.7-flash`, `gemini-2.5-pro`, `gemini-3.5-flash-lite`, `gemini-flash-lite-latest`, `gemini-3.6-flash` |
| Grok | 5 of 5 static, filled from the section | utility `grok-build-0.1`, `grok-4.3`, `grok-4.20-0309-reasoning`, `grok-4.6`, `grok-4.5`, `grok-4.7`; capable `grok-4.7`, `grok-4.6`, `grok-build-0.1`, `grok-4.5`, `grok-4.3`, `grok-4.20-0309-non-reasoning` |

No row differs from the table. A reorder within an unchanged set is what
"kept" allows.

With metadata off, on the recorded first-party listings, each six begins
with today's static list in order. The name curator fills the rest, as it
did before this phase: Claude's sixth is `claude-haiku-4-5-20251001`;
Gemini's are `gemini-3.7-flash`, `gemini-3.5-flash` and
`gemini-3.1-flash-lite`; Grok's is `grok-4.20-0309-non-reasoning`.
OpenAI's static list is already six.

#### Tests, red and green

* **Red** (the tests written, the code not yet changed):

  ```text
  band_0041_test.go:41: gpt-6-luna: small false, want true
  band_0041_test.go:41: gpt-5.4-mini: small true, want false
  band_0041_test.go:56: kilo cheap-pro: small false, want true
  band_0041_test.go:56: kilo dear-mini: small true, want false
  band_0041_test.go:79: no tools: eligible
  band_0041_test.go:79: image output: eligible
  band_0041_test.go:96: band [a/image a/notools]; want the two usable models only
  band_0041_test.go:149: profile 0: o3-pro in [gpt-5.4-nano gpt-5.4-mini o3-pro gpt-6-luna gpt-5.6-luna gpt-4.1-mini]
  band_0041_test.go:169: openai profile 1:
       got  [gpt-6.1-sol gpt-6-sol o3-pro gpt-6-luna gpt-5.6-luna gpt-4.1-mini]
       want [gpt-6.1-sol gpt-6-sol gpt-6-astra gpt-5.6-terra gpt-5.5-pro gpt-5.5]
  band_0041_test.go:199: claude profile 1: [...] lacks claude-fable-5-1
  ```

* **Green:**
  * `TestSmallFromPrice_0041`, `TestUnusableIneligible_0041`,
    `TestOneVendorGroups_0041`, `TestFirstPartyRankByFacts_0041` and
    `TestFirstPartyListRanksByFacts_0041`;
  * both snapshot tests and `TestStatic_EqualsLatestSnapshot`, with the
    new sixes;
  * `go test ./...`;
  * `-race -shuffle=on` on `catalog` and `wizard`;
  * every wire golden, three runs, with no golden changed.
* **Pre-existing, not this phase's:** the 0034 learned-refusal tests fail
  under `-count=3` at HEAD, before any Phase 3 change. They pass at
  `-count=1`, as the gate runs them.

#### Plants (`tools/plant.py`, 7 of 7 caught)

| Plant | FAIL line |
| :--- | :--- |
| the price band at 2.0 (PLAN) | `gpt-5.4-mini: small true, want false` |
| the regex deciding over a known price (PLAN) | `gpt-6-luna: small false, want true` |
| unusable models made eligible (PLAN) | `band [a/image a/notools]; want the two usable models only` |
| the letters group restored (PLAN) | `profile 0: o3-pro in [gpt-6-luna gpt-5.6-luna o3-pro gpt-4.1-mini gpt-5.6-terra gpt-6.1-sol]` |
| one first-party case still on its name curator (PLAN) | `TestFirstPartyListRanksByFacts_0041`: `openai:` got the name curator's six |
| Kilo's small from its name only | `kilo cheap-pro: small false, want true` |
| Grok's name curator replaced | `grok with metadata off: [] does not start with the static [grok-4.6 grok-4.5 grok-4.7 grok-4.3 grok-build-0.1]` |

On the first run the PLAN's fifth plant was **not caught**: every test
ranked a section through `metadataCurate` directly, and none listed a
first-party provider with a document. `TestFirstPartyListRanksByFacts_0041`
was added (deviation 2), and the rerun caught all seven. Every plant is a
change in behaviour, and none breaks the build.

#### Deviations

1. **2026-10-14: the tests D4 moves outside the two snapshot tests.**
   * **Found:** step 3 says the snapshot tests change "here, and only
     here", and names two. D4's rules also move:
     * `TestListModelCatalog_Snapshot20260926`, which replays 0009-MADR §7
       and must change with it;
     * `TestListModelCatalog_HuggingFaceRanksWithMetadata`: `b/mid`'s 0.75
       blend makes it small, so the capable profile ranks it after
       `d/uncovered` and `c/flash`;
     * `TestListModelCatalog_OpencodeGoGates`: `gpt-6-luna`'s 0.20 blend
       makes it small, so the utility profile ranks it second;
     * `TestWithLimit_SixIsToday` (0038): OpenAI's six is no longer the
       static list. It is now the first six of the default listing, which
       is the property 0038-MADR asserts;
     * `TestEvidence0041_CostKeyReach`: OpenAI's pairs that reach the cost
       key go from 13 to 8 of 276. Kilo's eligible set goes from 216 to
       209, the unusable leaving it, and its pairs from 144 of 23220 to 111
       of 21736. 0041-MADR §5's figures are the pre-D4 measurement, and
       stand as that;
     * the wizard's `claudeRefusals` fixture. Claude now fetches the
       document, and its transport counted that fetch as a listing. It
       now counts only `/v1/models`, and answers other paths 404 with
       `{}`. The assertion (three listings for three refused keys) is
       unchanged;
     * `unlabelledStaticIDs` in `model_label_test.go`: the new ranked
       static ids join it as `fallbackSix`, the reason every other ranked
       id has, and the ids that left the lists leave it.
   * **Decided by the owner:** amend 0009-MADR with the new sixes. It gains
     "Amendment 2026-10-14: 0041-MADR D4's band", with the six rows that
     change and the price that moves each one. §7's own table is kept as
     the 2026-09-26 record, and annotated. The other changes follow the
     same rule: each test's new expectation is D4's, with a comment that
     gives the price, and no assertion was loosened or removed.
2. **2026-10-14: a test for `List` on the first-party providers.**
   * **Found:** the PLAN's fifth plant, which leaves OpenAI on its name
     curator, was not caught (above).
   * **Done:** `TestFirstPartyListRanksByFacts_0041` serves each recorded
     first-party listing with the committed document. It asserts that
     `List`'s six is `metadataCurate`'s on the same usable ids, and that
     this differs from the name curator's, so each case tests something.
     It is in `band_0041_test.go`, in this phase's scope. The metadata-off
     check of `TestFirstPartyRankByFacts_0041` also asserts that today's
     static list leads (the Grok plant above). Neither changes a decision.

#### Checks

* `make pre-add-check` on the 11 Go files: clean, after one fix (an
  unused helper in the new test, `unused`).
* A misplaced doc comment was found while reading for the plants and
  fixed: `modelCatalogFor`'s comment had been left above
  `firstPartyUsable`.
* `tools/apidiff.py 9765ffc`: no exported change, as "The exported
  additions" lists none for Phase 3.
* `make records-check`: 88 records, 0 problems. `tools/claims.py`: 57 of
  57 on the MADR and 39 of 39 on the PLAN. 0009-MADR has no evidence rows
  for it to check.
* `tools/denyscan.py --changed`: 0 hits in 14 files.
  `tools/linkcheck.py`: 0 problems.
* **Gate:** `tools/gate.py`, 23 of 23, `g-wire-stable` included.

### Phase 4: the live menu (2026-10-14)

#### Research

* **P23, `catalogFrom`,** wraps a lister's usable ids. Its `recommend`
  cuts the six from `curate(usable, MaxListed)`. Its `fullList` (0038-MADR
  A1) appends `curate(usable, 0)`'s other ids. Each curator is
  `rankRecommended` over the facts, filled from its name curator and then
  the listing. The fill skips `utilityExcluded` and `excludedAtRequest`,
  and `metadataCurate` drops Together's unlisted ids first (`listedOnly`).
* **P24, `metadataCurate`,** reads the document from a channel. The menu's
  facts are read once the curation has run: the listing's own, collected
  by `listOnce`, the cached document (`peekMetadata`), and the caller's
  layers, merged as `CachedFacts` merges them (D1).
* **Ollama:** `ollamaCatalog` lists `/api/tags` only. `/api/show` is read
  by `fetchListingModel`, for `LookupModel`, and its reply is cached by
  provider, base URL and credential.
* **The `_0038` tests** that 0038-PLAN's follow-up lists:
  * `TestWithLimit_FullList` asserts D1's exclusions, through
    `expectedFull`, and the measured lengths, `snapshotFull`;
  * `TestFullList_SixFirst` pins `fullList`;
  * `TestWithLimit_FallbackExcludes` asserts that no id measured to fail
    at request is recommended;
  * the Ollama provider's `instance` counts every POST as a generation
    probe.
* **The wizard** reads `Recommended` and `Usable` only.
* **Measured on the 2026-10-12 snapshot** (a scratch test, on a clone):
  every usable id outside today's full list is measured to fail at
  request (Zen 2, Go 6, Together 1), or is a profile exclusion (Kilo 13,
  Hugging Face 17, Together 49, under `ProfileUtility`), or is one of
  Together's ids models.dev does not list (103 utility, 150 capable).
  There are no others. No static id has a deprecated status, or no tools,
  in the document.

#### Deviations

1. **2026-10-14: D3's order, the uncaveated exclusions, and Ollama's
   facts.**
   * **Found** in research, before any code: the three gaps 0041-MADR A4
     lists. D3's listing-order key leaves its id key unreachable, and the
     determinism test as step 4 states it cannot pass. Two classes 0038-MADR
     D1 leaves out of the full list have no caveat. And `List` reads no
     `/api/show`.
   * **Decided by the owner,** of the options offered:
     * the order is release date, then id (recommended);
     * the profile's exclusions and Together's unlisted ids are admitted to
       `Recommended` past the band (offered third; the recommended option
       was two more caveats);
     * `List` fetches `/api/show` for each uncached Ollama model
       (recommended).
   * **Recorded:** 0041-MADR amendment A4, D3 annotated; 0038-MADR
     amendment A5, A4.1 and A4.2 annotated; step 3 and step 4's rows
     annotated above. No exported identifier is added.
   * **Added to the phase's scope:** the Ollama provider's test server,
     which would count `/api/show` as a probe; `fullList` and its test,
     which the menu replaces.
   * **Not decided by the owner, and recorded here:** the band is curated
     from the uncaveated ids, so that D3's rule that no caveated entry is
     in `Recommended` holds within the six too; a static catalog's menu
     keeps the static order. A status of `retired` is caveated as
     `CaveatDeprecated`, since `StatusDeprecated` already covers "no longer
     served" and a retired model is past it.
2. **2026-10-14: the tests that pinned the exclusions on the full list.**
   * **Found** on the first green run: five catalog tests and two wizard
     tests asserted, on the whole of `Recommended`, what A4 now asserts only
     for the band:
     * `TestListKiloModels_MetadataCuration` and
       `TestKiloAuto_ExcludedButChoosable`: no `kilo-auto/*` tier under
       `ProfileUtility` (0009-MADR §3 item 9);
     * `TestTogetherRanksOnlyModelsDevIDs`: no id models.dev does not list
       (item 10);
     * `TestCatalogFrom_EmptyCurationKeepsLiveListing`: Kilo's tiers alone
       recommend nothing (0021-MADR C13);
     * `TestWithLimit_FullList`: D1's exclusions and the lengths;
     * the wizard's `TestConfigure_EmptyRecommendedOpensSearch` and
       `TestConfigure_EmptyRecommendedHonoursCtx`, which reach the
       "nothing recommended" path through Kilo's tiers alone. Step 4 said
       the wizard's tests would not change.
   * **The catalog tests follow deviation 1's decision.** Each item-9 and
     item-10 assertion now holds for the band (`Rank > 0`), and each test
     also asserts that the excluded ids follow the band in `Recommended`.
     The C13 test keeps the tiers-only case, which now recommends the
     tiers after an empty band, and adds a listing whose every id fails at
     request, which recommends nothing and stays live. `TestWithLimit_FullList`
     takes the lengths measured below, and its expected set is usable less
     the caveated, which it reads from the document and the listing's
     facts itself.
   * **Decided by the owner for the wizard,** of two options: "both
     cases" (recommended). `TestConfigure_TiersOnlyAreOffered` asserts that
     the tiers alone are offered, with no notice that no model meets the
     profile. The three empty-path tests take `caveatedKiloListing`, two
     `-contributor` ids, so the path and its ctx test are still reached.
     `TestConfigure_EmptyRecommendedSearchStillWorks` passed on the tiers
     listing, but no longer on an empty `Recommended`, so it moved too.
     No assertion was removed.
3. **2026-10-14: the Ollama wire golden.**
   * **Found:** `testdata/wire/ollama/listing.json` records the listing
     and its probes. It now also records the two `/api/show` requests A4
     decided. I reported that four requests at a time would make their
     order unstable.
   * **Decided by the owner,** of two options: keep four at a time, and
     compare the `/api/show` requests as a set (offered second; the
     recommended option was one at a time).
   * **Then found, and corrected here:** the harness already does that.
     `wiretest.Server.Requests` sorts the recorded requests by their
     encoding, because a listing's probes run concurrently too. So the
     arrival order never reaches the golden, and no harness change was
     made. My report of an unstable order was wrong.
   * **Done:** the one golden was regenerated with `-update`. Its diff adds
     the two `POST /api/show` requests, for `llama3.3:latest` and
     `qwen3:8b`, and changes nothing else. `g-wire-stable` runs it three
     times.

#### What was built

* **`menu.go`** (new):
  * `Caveat` and its five constants, and `MenuEntry`, as "The exported
    additions" lists;
  * `caveatsOf`: a deprecated or retired status; `tool_call` false; an
    output without text; a retirement within 30 days, the last day
    included; `excludedAtRequest`;
  * `newestFirst`: a dated id before an undated one, newest first, then
    by id;
  * `factsOf`: the listing's facts, the document's and the caller's
    layers, merged by `mergeLayers`, as `CachedFacts` merges them;
  * `menuOf`, `staticMenu` and `ollamaMenu`, and `ollamaShows`, which asks
    `/api/show` four at a time for each model not cached.
* **`discovery.go`:**
  * `Catalog.Menu`, and `Recommended`'s doc comment as A4 states it;
  * each lister returns a `listingResult`: the usable ids, the failure,
    the static list, the curator and the document it ranks with.
    `listOnce` builds the catalog from it once the listing's facts are
    collected. A failed listing reads only what is cached, so the static
    path waits for no document;
  * `catalogFrom`, `recommend` and `fullList` gave way to it;
  * `ollamaCatalog` emits each `/api/show` reply, so that `listOnce`
    caches it with the listing.
* **`model_metadata.go`:** `startModelMetadata` returns a function that
  waits for the document once, which the curation and the menu share;
  `metadataCurate` takes it.
* **A redundant guard removed:** `menuOf` first kept only curated ids that
  were uncaveated, unique and within six. A curator returns distinct ids
  of those it is given, at most `limit`, so the guard could never fail,
  and no plant could show it working. It was removed.
* **The lint's first run** found the local name `clear`, which shadows the
  built-in (`revive`); it is `uncaveated`.
* **Docs:** `docs/architecture.md`, the listing paragraph, the
  recommended list, `Catalog.Menu` and the package table (R43); 0038-PLAN's
  follow-up note.

#### The full lists, measured

On the 2026-10-12 snapshot, the same under both profiles, pinned by
`TestMenu_Snapshot_0041` and `TestWithLimit_FullList`:

| Provider | Before, utility / capable | After | The caveated |
| :--- | ---: | ---: | :--- |
| OpenCode Zen | 77 / 77 | 77 | 2 `-contributor` ids, unavailable, one also deprecated |
| OpenCode Go | 28 / 28 | 25 | 6 unavailable; `qwen3.7-max`, `kimi-k2.6`, `qwen3.6-plus` deprecated |
| Kilo | 272 / 285 | 276 | 9 expiring by Kilo's listing, `google/gemini-2.5-pro` among them |
| Hugging Face | 102 / 119 | 118 | `zai-org/AutoGLM-Phone-9B-Multilingual`, no tools |
| Together | 19 / 21 | 166 | 5 deprecated; `MiniMaxAI/MiniMax-M2.7`, unavailable |

No six changed: `TestListModelCatalog_Snapshot20261012`,
`TestListModelCatalog_Snapshot20260926` and `TestStatic_EqualsLatestSnapshot`
pass unchanged. OpenAI's section, ranked as a menu, holds no deprecated
model in its full list, where §4's curator held 11, and `gpt-3.5-turbo`
is in the menu with `CaveatNoTools`.

#### Tests, red and green

* **Red** (the tests written first):

  ```text
  menu_0041_test.go:67:21: undefined: MenuEntry
  menu_0041_test.go:79:9: undefined: menuOf
  menu_0041_test.go:152:23: undefined: Caveat
  menu_0041_test.go:153:18: undefined: CaveatDeprecated
  ```

  The behaviour each test checks was seen failing through the plants
  below, the build intact.
* **Green:**
  * `TestMenu_Order_0041`, `TestMenu_Deterministic_0041` (100 shuffled
    listings), `TestMenu_Caveats_0041`, `TestMenu_Snapshot_0041`,
    `TestRecommended_Prefix_0041` (n = 1 to 8), `TestMenu_Static_0041`
    and `TestMenu_Ollama_0041`;
  * the tests deviation 2 lists, and the Ollama provider's four, whose
    servers answer `/api/show` apart from the generation requests they
    count. `TestListModels_CarriesTheCallersIdentity` now also requires
    the `/api/show` request to name the caller;
  * `TestFullList_SixFirst` was removed with `fullList`, the function it
    pinned. `TestRankRecommended_UncappedFillDiffers` is kept;
  * `go test ./...`; `-race -shuffle=on` on `catalog`, `wizard` and
    `providers/ollama`.

#### Plants (`tools/plant.py`, 13 of 13 caught)

| Plant | FAIL line |
| :--- | :--- |
| caveated entries left in `Recommended` (PLAN) | `deprecated: in Recommended true, rank 0; want false`; the wizard's `Model = "a/x-contributor", want b/y-contributor, found by search` |
| the id tie-break removed (PLAN) | `TestMenu_Deterministic_0041`: `listing [band-1 old-deprecated z-old …]:` a different menu |
| the rest ordered by name score instead of date (PLAN) | `tiers only: … Recommended [] …; want … [kilo-auto/balanced kilo-auto/small]` |
| a caveat's fact ignored: tools (PLAN) | `notools: caveats []; want [no-tools]` |
| the band curated from every usable id | `TestMenu_Order_0041`: `menu:` and `Recommended:` differ |
| a retired status not caveated | `retired: caveats []; want [deprecated]` |
| the expiry window's last day left out | `expiring: caveats []; want [expiring]` |
| the menu not cut by `WithLimit` | `together WithLimit(1): … want the full list's first 1` |
| a static id ranked 0 | `gpt-6-luna: rank 0; want 1` |
| Ollama's caveated left in `Recommended` | `Recommended [b:1 a:1 notools:1]; want [b:1 a:1]` |
| Ollama's cached replies asked for again | `/api/show asked 6 times after a second listing; want the cached 3` |
| Ollama's replies not cached | `menu [b:1 notools:1 a:1]; want [b:1 a:1 notools:1]` |
| the menu's facts without the document | `huggingface profile 0: the full list holds 119 ids; want 118` |

`plant.py` prints 12 failure lines a plant, so the first plant was run
again on the wizard alone, to see its tests fail.
`TestConfigure_EmptyRecommendedSearchStillWorks` passes under it, since
search works whatever is recommended, as it did before this phase.

#### Checks

* `make pre-add-check` on the 14 Go files: clean, after the rename above.
* `tools/apidiff.py 7f34e31`: `Catalog.Menu`, `Caveat`, its five constants
  and `MenuEntry`, all added, and nothing else, as "The exported
  additions" lists for Phase 4.
* `make records-check`: 88 records, 0 problems. `tools/claims.py`: 57 of
  57 on the MADR and 39 of 39 on the PLAN; 0038-MADR has no evidence rows.
* `tools/denyscan.py --changed`: 0 hits in 20 files.
  `tools/linkcheck.py`: 0 problems.
* **Gate:** `tools/gate.py`, 23 of 23. `g-wire-stable` ran every golden
  three times; the one golden that changed is deviation 3's.

*2026-10-14, a cross-reference.*
[0042-PLAN-ollama-models-that-write-text.md](0042-PLAN-ollama-models-that-write-text.md)
Phase 2 changed `TestMenu_Ollama_0041`'s `Recommended` assertion to
`[b:1 a:1 notools:1]`: on Ollama, a model whose only caveat is
`CaveatNoTools` stays recommended (0042-MADR D2; 0041-MADR A5). The
plant above, "Ollama's caveated left in `Recommended`", describes the
behaviour before that change.

### Phase 5: `ProfileAgent` (2026-10-14)

#### Research

* **P11 to P13.** `Profile` had two values, and `ReasoningEffort` tested
  for `ProfileCapable`. The capable order and the band's loop are
  `rankRecommended`'s. `WithProfile`'s doc still named only the open
  catalogs, which has been out of date since Phase 3 ranked every provider
  (D4).
* **Phase 2's `pricesByCanonical`** returned
  `c.ModelRates, slices.Contains(costFrom(&c, e.Cost), FieldCostCacheRead)`.
  * The Go spec leaves unspecified when the variable `c` is read, relative
    to the call that fills it (the spec's "Order of evaluation", and its
    example with `x[j()]` and `y`).
  * Today's compiler reads it after the call. `TestCanonicalID_0041`'s
    check of a 0.25 cache-read price passes at 35c2e47, re-run on a
    scratch clone.
  * Phase 5's `withCanonicalRead` is its second caller. That is deviation 1.
* **The trial.** 0041-MADR D5's figures came from a scratch harness this
  session wrote, and the clone it ran on is gone.
  * The harness was recovered from the session transcript.
    `TestEvidence0041_AgentTrial` reproduces it.
  * The harness has every models.dev section, and D4's family groups on
    the one-vendor sections. Its prices come from the document, with the
    canonical fill.
  * Its order is: signal, bench, preferred index, not small, the release
    quarter, the known turn first, the cheaper turn, then id.
  * The test gives exactly the MADR's numbers, below.
* **Kilo's prices.** Kilo is ranked from its own listing, not the
  document. So `kiloCandidate` reads its per-token prompt, completion,
  cache read and cache write prices, as millions. A cache price counts as
  known only when its string is present.

#### Deviations

1. **2026-10-14: `pricesByCanonical`'s order of evaluation.**
   * **Found:** in research, before Phase 5's new caller. See above.
   * **Decided by the owner** (recommended): keep the fix, and record it.
     The value is computed into `known` first, and then returned. Nothing
     measurable changes, so no MADR amendment is needed.
   * **Added to the phase's scope:** `metadata_document.go`'s two lines.
2. **2026-10-14: `ObservedWorkload`'s `ok`.**
   * **Found:** step 2 says `ok` is false "when the sums are zero". A
     measured share of exactly 0 or 1 is refused by `WithWorkload`, so an
     `ok` result would not always be usable. A caller with no cached
     tokens is one case.
   * **Decided by the owner** (recommended): `ok` means usable. It is
     false when there is nothing to measure, or when a share is 0 or 1.
     0041-MADR D5 already gives the shares "for `WithWorkload`", so the
     MADR stands.

#### What was built

* **`model_profile.go`:**
  * `ProfileAgent = 2`, documented.
  * `strongest()`, true for `ProfileCapable` and `ProfileAgent`, gives
    `ReasoningEffort` its `""`.
  * `workload{cacheShare, outputShare}`, with `defaultWorkload` from
    `agentCacheShare = 0.90` and `agentOutputShare = 0.01`. Each constant
    cites 0041-MADR §7.
  * `turnPrices`, and `workload.turn`: the cached turn as step 2 states
    it. It is unknown when the input or output price is.
  * `share`, and `ObservedWorkload`.
* **`config.go`:**
  * `WithWorkload(h, o)`, scoped to the built-in ids. `configFor` refuses
    a value outside (0, 1), NaN and infinities included, with
    `ErrInvalidRequest`.
  * `config.workload` defaults to the published workload.
  * `WithProfile`'s doc is corrected (Research).
* **`model_ranking.go`:**
  * `rankCandidate` gains `prices`, `turn` and `turnKnown`, set by
    `withTurn`.
  * `quarter()` is `ageDays / 90`, and an unknown age sorts last.
  * `agentOrder` is the order step 2 lists, with an unknown turn after a
    known one.
  * `agentBand` takes `ProfileCapable`'s ranking at no fewer than six,
    reorders the first six, and then cuts to the caller's limit.
    `rankRecommended` sends `ProfileAgent` to it.
  * Kilo's prices (Research).
* **`model_metadata.go` and `discovery.go`:**
  * `metadataCandidate` sets the turn prices from the document.
  * `metadataCurate` and `kiloCurate` take the workload. Each candidate
    gets the canonical read fill (`withCanonicalRead`), then its turn.
* **`metadata_document.go`:**
  * `withCanonicalRead`, and `pricesOf` for the evidence test;
  * deviation 1.
* **Docs:**
  * `Profile`'s doc and `WithProfile`'s;
  * `docs/architecture.md`: the package row, `List`'s options, the
    `Profile` bullet, and a `ProfileAgent` bullet (R43).

#### The agent bands, measured

On the 2026-10-12 snapshot, through `List` with `WithLimit(MaxListed)`.
`TestProfileAgent_Snapshot_0041` pins both the published and the
(0.5, 0.05) columns.

| Provider | `ProfileCapable` | `ProfileAgent`, published workload | `ProfileAgent`, (0.5, 0.05) |
| :--- | :--- | :--- | :--- |
| OpenCode Zen | mistral-large-4, gpt-6.1-sol, claude-sonnet-5-5, claude-opus-5-5, gpt-6-sol, grok-4.7 | mistral-large-4, gpt-6.1-sol, claude-sonnet-5-5, gpt-6-sol, grok-4.7, claude-opus-5-5 | mistral-large-4, grok-4.7, gpt-6.1-sol, claude-sonnet-5-5, gpt-6-sol, claude-opus-5-5 |
| OpenCode Go | grok-4.7, glm-5.3, grok-4.6, qwen3.8-max, kimi-k3, glm-5.2 | glm-5.3, qwen3.8-max, grok-4.6, grok-4.7, kimi-k3, glm-5.2 | glm-5.3, grok-4.6, grok-4.7, qwen3.8-max, kimi-k3, glm-5.2 |
| Kilo | openai/gpt-6-astra, anthropic/claude-fable-5.1, openai/gpt-5.6-sol, deepseek/deepseek-v4.1-flash, google/gemini-3.8-flash, x-ai/grok-4.6 | the same | the same |
| Hugging Face | zai-org/GLM-5.3, Qwen/Qwen3.8-27B, Qwen/Qwen3.8-2.4T-A95B, deepseek-ai/DeepSeek-V4-Pro-0813, moonshotai/Kimi-K3, thinkingmachines/Inkling | Qwen/Qwen3.8-27B, thinkingmachines/Inkling, deepseek-ai/DeepSeek-V4-Pro-0813, zai-org/GLM-5.3, Qwen/Qwen3.8-2.4T-A95B, moonshotai/Kimi-K3 | the same as published |
| Together | zai-org/GLM-5.3, deepseek-ai/DeepSeek-V4-Pro-0813, moonshotai/Kimi-K3, thinkingmachines/Inkling, zai-org/GLM-5.2, nvidia/nemotron-3-ultra-550b-a55b | deepseek-ai/DeepSeek-V4-Pro-0813, thinkingmachines/Inkling, zai-org/GLM-5.3, moonshotai/Kimi-K3, nvidia/nemotron-3-ultra-550b-a55b, zai-org/GLM-5.2 | thinkingmachines/Inkling, deepseek-ai/DeepSeek-V4-Pro-0813, zai-org/GLM-5.3, moonshotai/Kimi-K3, nvidia/nemotron-3-ultra-550b-a55b, zai-org/GLM-5.2 |

* Every agent band holds its capable band's members.
* Kilo's order is capable's at both workloads: its quality signals
  decide before the price.
* The trial (`TestEvidence0041_AgentTrial`) gives the MADR's figures.
  "Kept" is how many of the price-chosen six are in capable's six;
  "distinct" is how many different bands the 16 workloads give:

  | Section | Kept | Distinct |
  | :--- | :--- | :--- |
  | OpenAI | 6 of 6 | 1 |
  | Claude | 6 of 6 | 1 |
  | Gemini | 4 of 5 | 1 |
  | Grok | 3 of 3 | 1 |
  | Zen | 1 of 6 | 4 |
  | Go | 5 of 6 | 3 |
  | Hugging Face | 6 of 6 | 1 |
  | Together | 6 of 6 | 3 |
  | Kilo | 1 of 6 | 12 |

#### Tests, red and green

* **Red** (the tests written first):

  ```text
  profile_agent_0041_test.go:20:12: undefined: ProfileAgent
  profile_agent_0041_test.go:47:68: undefined: workload
  profile_agent_0041_test.go:50:3: unknown field prices in struct literal of type rankCandidate
  profile_agent_0041_test.go:50:11: undefined: turnPrices
  profile_agent_0041_test.go:58:7: undefined: defaultWorkload
  ```

  The behaviour each test checks was seen failing through the plants
  below, the build intact.
* **Green:**
  * Step 3's tests: `TestProfileAgent_ReasoningEffort_0041`,
    `TestProfileAgent_Members_0041`, `TestProfileAgent_Order_0041`,
    `TestCachedTurn_0041`, `TestWithWorkload_0041`,
    `TestObservedWorkload_0041` and `TestEvidence0041_AgentTrial`.
  * Beyond step 3:
    * `TestProfileAgent_Snapshot_0041`: the table above;
    * `TestKiloTurnPrices_0041`: Kilo's four prices, and no cache price
      without one.
  * `TestWithWorkload_0041` also refuses infinity.
  * `band_0041_test.go` and `menu_0041_test.go` pass the published
    workload to `metadataCurate`'s new parameter.
  * `go test ./...` passes.

#### Plants (`tools/plant.py`, 10 of 10 caught)

| Plant | FAIL line |
| :--- | :--- |
| members chosen by price (PLAN) | `opencode-go: agent band [mimo-v2.6-flash claude-haiku-5-5 …]; want capable's members [grok-4.7 glm-5.3 …]`, on every provider and section |
| the bucket removed (PLAN) | `agent order [unpriced-new dear-new cheap-new cheapest-old]; want [cheap-new dear-new unpriced-new cheapest-old]` |
| `h` and `o` swapped (PLAN) | `uncached turn 9.2, true; want 2.08` |
| the canonical fill skipped (PLAN) | `a/claude-x: read 0, known false; want the canonical model's 0.2` |
| the range check removed (PLAN) | `WithWorkload(0, 0.5): <nil>; want ErrInvalidRequest`, and five more |
| an unknown turn not last | `agent order [unpriced-new cheap-new dear-new cheapest-old]; want …` |
| Kilo's cache read ignored | `prices {… read:0 … readKnown:false …}; want 2, 10, 0.2 and 2.5, each known` |
| the curated candidates' turn not computed | `TestProfileAgent_Snapshot_0041`: `opencode-zen:` and every other priced provider |
| `ObservedWorkload`'s share check removed | `no cached tokens, a share of 0: ok true` |
| `ProfileAgent`'s effort low | `ProfileAgent.ReasoningEffort() = "low"; want ""` |

#### Checks

* `make pre-add-check` on the 10 Go files: clean (gofmt, golangci-lint,
  go vet, go test, govulncheck).
* `tools/apidiff.py 35c2e47`: `ObservedWorkload`, `ProfileAgent` and
  `WithWorkload`, all added, and nothing else.
* **Gate:** `tools/gate.py`, 23 of 23.
  * The first run gave 21 of 23. `records-check` and `gate-selftest`
    found the index at 90 records while 91 were on disk. Another session
    was adding `docs/reports/0017-REPORT-openai-sign-in-for-agents.md` and
    its index row while the gate copied the tree.
  * Once both had landed, `make records-check` gave 91 records and
    0 problems, and the second run gave 23 of 23.
  * That report is not this phase's, and is not staged with it.

### Phase 6: steady menus (2026-10-15)

#### Research

* **P15, the listing cache.**
  * `listingEntry` holds a listing's facts and when they were read, by
    `listingKey`: the provider, the base URL, and the credential's
    fingerprint.
  * `listOnce` stores the facts it emitted (`storeListing(…, false)`)
    when the catalog is live.
  * Nothing caches the listing itself: every `List` fetches.
* **The listers.**
  * `modelCatalogFor` fetches and builds in one step.
  * Most listers' curators need only the usable ids and the models.dev
    document. Kilo's, `kiloCurate(entries, profile, w)`, ranks from each
    entry's prices and dates.
  * Ollama's lister fetches `/api/show` per model, and returns its errors.
* **The errors.**
  * `ClassifyHTTPError` gives `ErrProviderUnavailable` for 5xx and 408, and
    for no response. ~~The listers wrap send failures in it too (0026-MADR
    F35).~~ *(Corrected by the network test, 2026-10-15: only Ollama's
    listing wraps a send failure. The others return the transport's error,
    a `net.Error`, which the allowlist's "a network error" covers.)*
  * It gives `ErrRateLimited` for 429.
  * Others are terminal: `ErrAuthFailure`, `ErrInvalidRequest`,
    `ErrQuotaExhausted` and `ErrNotPermitted`.
  * Step 2's allowlist (`ErrProviderUnavailable`, `ErrRateLimited`,
    `context.DeadlineExceeded`, a network error) therefore sends
    everything else to the static floor.
* **The consumers.**
  * gobble treats `!Live` as a failure and retries (E42, E43).
  * prepare-commit-msg shows `Recommended` (E44), and reports a refused
    credential (E45).
* **A scratch prototype.** A fresh listing was served without fetching,
  per key, profile, limit, workload, layer count and metadata switch.
  `go test ./llmprovider/... ./wizard/` then failed two tests, and no
  others:
  * `TestListModels_ListsTheOrganization` (`providers/kilo`):
    `listing requests = [], want the organization's catalog with its header`;
  * `TestOllamaUsable_TagsNotCached_0042`:
    `/api/show asked map[chat:1:1 granite-embedding:30m:1] after two listings; want … granite-embedding:30m:2`.

#### Deviations

1. **2026-10-15: what the cache holds.**
   * **Found:** the curators need more than ids for Kilo (research).
   * **Decided by the owner** (recommended): the cache holds the fetched
     listing, ids and Kilo's entries. Each lister is split into a fetch
     and a build. 0041-MADR A6.
   * **Added to the phase's scope:** that restructuring of `discovery.go`.
2. **2026-10-15: the cache key.**
   * **Found:** the prototype answered a Kilo organization's request from
     the personal account's listing.
   * **Decided by the owner** (recommended): `listingKey` gains the Kilo
     organization. 0041-MADR A6.
   * **Added to the phase's scope:** the facts cache, which shares the key.
3. **2026-10-15: Ollama.**
   * **Found:** a fresh cached listing breaks 0042-MADR D1's retry of a
     failed `/api/show`, hides newly pulled models, and Ollama has no
     static floor.
   * **Decided by the owner** (recommended): Ollama is outside D6. It lists
     every time, never serves stale, and returns its errors. 0041-MADR A6.

#### Deviation 1, measured

On the 2026-10-12 snapshot, a scratch probe built Kilo's list both with its
entries and from its ids alone. The ids are cheapest first, as `kiloUsable`
orders them:

| Profile | `WithLimit` | Same list without the entries? |
| :--- | :--- | :--- |
| `ProfileUtility` | `MaxListed` | yes |
| `ProfileUtility` | none | no: the full list differs from its seventh id |
| `ProfileCapable` | `MaxListed` and none | no: the six differ from the first, `openai/gpt-6-astra…` against `deepseek/deepseek-v4.1-flash…` |
| `ProfileAgent` | `MaxListed` and none | no, as for `ProfileCapable` |

`TestList_KiloStaleRanksAsLive_0041` was first written under
`ProfileUtility`'s six, and so passed with the entries dropped. That plant
was not caught. The test now checks `ProfileCapable`'s six and
`ProfileUtility`'s full list, and the plant is caught.

#### What was built

* **`discovery.go`:**
  * `Catalog.Source` and `Catalog.FetchedAt`, and `ListSource` with
    `ListLive`, `ListStale` and `ListStatic`;
  * the `Live`, `Err` and `List` docs;
  * `listOnce`, through the cache, and Ollama's `listOllamaOnce` beside
    it;
  * `cachedOrFetched` and `fetchClaimed`: one fetch per key;
  * `transientListingFailure`: the allowlist;
  * `modelCatalogFor` split into `fetchListing` and `buildListing` around
    `fetchedListing` (deviation 1). `opencodeCatalog` goes, and its doc
    moves to `fetchOpencodeUsable`.
* **`listing_cache.go`:**
  * `listingKey` gains `organization` (deviation 2);
  * `listingEntry` gains the fetched listing and when it was read;
  * `lastGoodWindow` (24 hours);
  * `storeFetched`, `freshListing`, `lastGoodListing` and
    `claimListing`.
  * `storeListing` keeps the listing when it replaces the facts.
* **`config.go`:** `WithLastGood`, and its `lastGood` value.
* **Tests:**
  * `steady_0041_test.go`: step 3's six tests, and four more for the
    deviations: `TestList_CachedRebuildsWithProfile_0041`,
    `TestList_KiloStaleRanksAsLive_0041`,
    `TestList_KiloOrganizationKeyed_0041` and
    `TestList_OllamaOutsideD6_0041`.
  * `providers/kilo`'s `TestListModels_ListsTheOrganization` gives each
    iteration its own server. Its two iterations list the same
    organization, so the second was answered from the cache, which D6
    intends. The assertion is unchanged.
* **Docs:** `docs/architecture.md`, the listing section, the package row
  and `List`'s options (R43).

#### Tests, red and green

* **Red** (the tests written first):

  ```text
  steady_0041_test.go:115:68: second.Source undefined (type Catalog has no field or method Source)
  steady_0041_test.go:115:78: undefined: ListLive
  steady_0041_test.go:116:11: second.FetchedAt undefined (type Catalog has no field or method FetchedAt)
  steady_0041_test.go:175:23: undefined: ListStale
  ```

  The behaviour each test checks was seen failing through the plants
  below.
* **Green:**
  * the `TestList_` tests;
  * `go test ./llmprovider/... ./wizard/ ./internal/...`;
  * `-race` on `catalog`'s `TestList_`, `_0041` and `_0042` tests.
  * No wire golden changed.
* **Found while going green:**
  * The network case's `Err` is `dial tcp …: connection refused`, a
    `net.Error`, and not `ErrProviderUnavailable`. That corrects the
    research note above. The test now expects a `net.Error`.
  * My `listOllama` collided with a test helper of that name, and was
    renamed `listOllamaOnce`.
  * gocritic's `deferInLoop` moved the claimed fetch into `fetchClaimed`.

#### Plants (`tools/plant.py`, 11 of 11 caught)

| Plant | FAIL line |
| :--- | :--- |
| stale served on a 401 (PLAN) | `Source stale, Live true, Err llmprovider: authentication failed: openai HTTP 401 …; want static, not live, an auth failure` |
| the 24-hour window removed (PLAN) | `Source stale, Live true, Err llmprovider: provider unavailable: openai HTTP 503 …; want the static floor` |
| `Live` false on stale (PLAN) | `Source stale, Live false, …; want stale, live, llmprovider: provider unavailable` |
| the fetch not deduplicated (PLAN) | `8 listing requests for 8 concurrent Lists; want 1` |
| the cache served past the TTL | `1 listing requests once the TTL passed; want 2` |
| a network error not transient | `Source static, Live false, Err Get "http://127.0.0.1:…/models": dial tcp …: connection refused; want stale` |
| `WithLastGood` ignored | `Source static; want stale` |
| Kilo's entries not cached (deviation 1) | `stale Kilo list (stale, 6 ids) differs from the live one (6 ids): ["deepseek/deepseek-v4.1-flash" …]` |
| the organization left out of the key (deviation 2) | `listing requests ["/models"]; want one personal, one for the organization, each once` |
| Ollama through the cache (deviation 3) | `ollama: no option recommends []; want every installed model [m1 … m8]` |
| a cache hit built without the listing's facts | `menu entry 1: cached gpt-4.1-mini released 2025-04-14 … ; live gpt-4.1-mini released 2025-04-15 …` |

The first run caught 9 of 11. "Kilo's entries not cached" is above. "A
cache hit built without the listing's facts" was not caught because the
listing carried no dates. The test now serves `created` times, and checks
each menu entry's release date against the live listing's.

The plants ran again after the `fetchClaimed` change: 11 of 11.

#### Checks

* `make pre-add-check` on the 5 Go files: clean (gofmt, golangci-lint,
  go vet, go test, govulncheck), after the `deferInLoop` change.
* `tools/apidiff.py 2e17224`: `Catalog.FetchedAt`, `Catalog.Source`,
  `ListLive`, `ListSource`, `ListStale`, `ListStatic` and `WithLastGood`,
  all added, and nothing else, as "The exported additions" lists for
  Phase 6.
* **Gate:** `tools/gate.py, 23 of 23`.
* `make records-check`, `tools/claims.py`, `tools/denyscan.py --changed`
  and `tools/linkcheck.py`: in the staging run.

### Phase 7: the refresh check (2026-10-15)

#### Research

* **The command.** `internal/cmd/catalogs -snapshot DIR` writes a new
  directory:
  * each ranked list's listing body, the four public ones read with no key
    and the others with their provider's key when it is set;
  * the document's sections the SDK reads;
  * `snapshot.json`, with `recorded_at` and each file's SHA-256.

  It prints "`<provider>: <VAR> unset; its listing is skipped`" for each
  unset key, and never prints a key. The committed
  `testdata/ranking-2026-10-12` has that layout: `zen.json`, `go.json`,
  `kilo.json`, `hf.json`, `together.json`, `metadata.json` and
  `snapshot.json`.
* **The agreement tests.**
  * `TestAgreement0041_MessagesThinking` (R1, R2) and
    `TestAgreement0041_GeminiConfig` (R4) are in `catalog`, and read
    `evidence0041Dir`.
  * `TestAgreement0041_GeminiLevel` (R5) is in `providers/gemini`, and
    `TestAgreement0041_GrokEfforts` (R6) in `providers/grok`. Each
    hard-codes `../../catalog/testdata/ranking-2026-10-12/metadata.json`
    (`level_0041_test.go:93`, `efforts_0041_test.go:101`).
  * Each reports one line per rule, "`facts and the rule differ on […];
    want only […]`", and logs each differing model with `-v`.
* **`LLMPROVIDER_SNAPSHOT_DIR`** is read nowhere. Step 1 assumes it.
* **The sixes.** Nothing prints a snapshot's sixes. `replaySnapshot(t,
  dir, provider, profile, opts…)` builds a provider's `Catalog` from any
  snapshot directory, and `latestSnapshot` picks the newest committed
  one.

#### Deviations

1. **2026-10-15: how the check reads a new snapshot.**
   * **Found:** in research, before any code. No test reads
     `LLMPROVIDER_SNAPSHOT_DIR`, and nothing prints the sixes.
   * **Decided by the owner** (recommended):
     * `evidence0041Dir` and the two provider agreement tests read
       `LLMPROVIDER_SNAPSHOT_DIR` when it is set, else the committed
       snapshot. This is test code only.
     * One printer test, `TestCatalogCheck0041_Sixes`, skipped unless the
       variable is set, prints each open catalog's six under each
       profile.
     * `catalogcheck.py` gains `--dir DIR`, to check an existing snapshot
       with no network. Its tests use it on planted fixtures.
   * **Added to the phase's scope:**
     * `llmprovider/catalog/evidence_0041_test.go`;
     * `llmprovider/providers/gemini/level_0041_test.go`;
     * `llmprovider/providers/grok/efforts_0041_test.go`;
     * the new `llmprovider/catalog/catalogcheck_0041_test.go`.
   * D7 names no variable, so the MADR stands.

#### What was built

* **`tools/catalogcheck.py`:**
  * Without `--dir`, it records a snapshot with
    `go run ./llmprovider/internal/cmd/catalogs -snapshot DIR`, into
    `--out`, which must not exist, or a new temporary directory. It
    relays the listings the command skipped. It reads no key and prints
    none.
  * With `--dir`, it checks an existing snapshot, and sends no request.
  * It runs the `TestAgreement0041_` tests in `catalog`,
    `providers/gemini` and `providers/grok`, with
    `LLMPROVIDER_SNAPSHOT_DIR`, and lists each failing one with its logged
    models.
  * It runs `TestCatalogCheck0041_Sixes` for the snapshot and for the
    baseline (`--baseline`, or the newest committed
    `testdata/ranking-*/`). It reports each open catalog's six under each
    profile, as unchanged or as "`n of 6 changed: +… -…`".
  * Exit 0 is clean, 1 a finding, 2 a tool error. It never edits.
* **Deviation 1's test changes:**
  * `evidence0041Dir` reads `LLMPROVIDER_SNAPSHOT_DIR` through
    `snapshotDirOr`;
  * `providers/gemini` and `providers/grok` each gain
    `agreementSnapshotDir`;
  * `catalogcheck_0041_test.go` holds the printer, skipped without the
    variable.
* **Tests** (`tools/test_tools.py`, `TestCatalogCheck`):
  * a clean copy of the committed snapshot gives exit 0;
  * a planted copy gives exit 1, and names each planted model:
    * Claude Haiku 4.5's budget minimum (R1, R2);
    * Gemini Flash's efforts, which move both R4 and R5;
    * Grok 4.7's efforts (R6);
    * a model removed from Kilo's six;
  * a directory without `snapshot.json`, and an `--out` that exists, each
    give exit 2.
* **Docs:**
  * `tools/README.md`'s row;
  * 0033-PLAN's Phase 7, by a dated note: the refresh runs the check, and
    its report goes in the Execution Record.

#### Runs

* **The committed snapshot:**
  `tools/catalogcheck.py --dir llmprovider/catalog/testdata/ranking-2026-10-12`
  gives `rules: 0 differ from the facts`, and each of the 15 sixes
  unchanged, exit 0.
* **The planted fixture:** `rules: 4 differ from the facts`, each with
  its model:
  * `claude/claude-haiku-4-5 at "low": rule …budget_tokens":1024…, facts …4096…`;
  * R4 and R5 on `gemini-flash-latest`;
  * R6 on `grok-4.7`.

  And `sixes …: 3 changed`:
  * `kilo utility: 1 of 6 changed: +meta/muse-spark-1.2; -deepseek/deepseek-v4.1-flash`;
  * Kilo's capable and agent sixes, each with `+moonshotai/kimi-k3`.

  Exit 1.
* **The plants, found first.** Before the fixture was written, edits were
  tried on scratch copies to find ones that move each rule:
  * a budget minimum of 4096 moves R1/R2 on that model alone;
  * `reasoning: false` moves nothing;
  * dropping the reasoning options makes the model silent instead.

#### Tests, red and green

* **Red.** The tests were written after the tool, so their red is the
  plants below. Each was seen failing on a deliberate breach, the build
  intact.
* **Green:**
  * `make tools-test`: 45 tests, OK;
  * `TestCatalogCheck`, also under the system's Python 3.9;
  * `go test ./llmprovider/...`. The variable is unset there, so the
    agreement tests read the committed snapshot, as before.

#### Plants (`tools/plant.py`, the command `unittest … -k TestCatalogCheck`, 10 of 10 caught)

| Plant | Caught by |
| :--- | :--- |
| the snapshot not passed to the agreement tests | `test_a_planted_snapshot_reports_both` |
| a failing agreement test not counted | the same |
| a changed six ignored | the same |
| the baseline compared with itself | the same |
| a failed command not a tool error | `test_an_existing_out_is_a_tool_error` |
| a directory without a snapshot accepted | `test_a_dir_without_a_snapshot_is_a_tool_error` |
| `catalog`'s agreement tests ignore the variable | `test_a_planted_snapshot_reports_both` |
| the Gemini agreement test ignores the variable | the same |
| the Grok agreement test ignores the variable | the same |
| the sixes printer ignores the variable | the same |

#### Checks

* `make pre-add-check` on the 4 Go files: clean (gofmt, golangci-lint,
  go vet, go test, govulncheck).
* `tools/apidiff.py 1d4e8ba`: no exported change.
* **Gate:** `tools/gate.py, 23 of 23`.
* `make records-check`, `tools/claims.py`, `tools/denyscan.py --changed`
  and `tools/linkcheck.py`: in the staging run.
* **Not run:** a live `tools/catalogcheck.py` without `--dir`.
  * It reads the public listings, and the keyed ones whose keys are set.
  * Phase 9's live check covers the release. A refresh runs it by
    0033-PLAN's note.
