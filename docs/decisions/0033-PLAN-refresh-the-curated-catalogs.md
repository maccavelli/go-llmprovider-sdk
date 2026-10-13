---
status: in-progress
date: 2026-10-12
associated-madr: "0033-MADR-public-model-metadata-and-current-catalogs.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Refresh the Curated Model Catalogs

Associated MADR: [0033-MADR-public-model-metadata-and-current-catalogs.md](0033-MADR-public-model-metadata-and-current-catalogs.md)

This is the first of the MADR's two plans: D4, released as `v1.7.1`. The
other is
[0033-PLAN-public-model-metadata.md](0033-PLAN-public-model-metadata.md).

## Goal

When it is done:

* **No retired or doomed id.** No static catalog names an id its provider
  has retired, scheduled for shutdown, routes to another model, or
  restricts to past users.
* **Recommendations measured.** Each menu's first entry is a current
  model, measured to answer.
* **Rankers current.** The rankers prefer current models over retired ones
  in a live listing.
* **A guard.** A live test fails when a catalog goes stale.

All of it ships as `v1.7.1` before 2026-10-23, when OpenAI shuts down
`gpt-4.1-nano` and `o4-mini`.

## Scope

### In scope

* `llmprovider/catalog/models_catalog.go`:
  * `staticOpenAI`, `staticGemini`, `staticClaude`, `staticGrok` and
    their comments;
  * `modelLabels` and `Label`;
  * `rankOpenAIModel`, `rankGeminiModel`, `rankClaudeModel` and
    `rankGrokModel`, with their anchors;
  * `staticOpencodeGo`'s comment.
* `llmprovider/providers/grok/reasoning.go`: effort menus for `grok-4.7`
  and `grok-4.3`.
* Tests:
  * a unit test that no static id is on the retired list;
  * ranker tests for the current naming;
  * the live catalog test.
* The G-wire listing goldens, only where Phase 2's change explains a
  difference.
* `README.md` Status, release notes, and the index.
* *(Amended 2026-10-12, 0033-MADR's amendment "the second refresh, and what
  the command reads"; Phases 5 to 9.)*
  * `llmprovider/internal/catalogtool` and `llmprovider/internal/cmd/catalogs`:
    the listings the command reads, the proposal for each list by its rule,
    and `-snapshot`;
  * the five ranked lists, `staticOpencodeZen`, `staticOpencodeGo`,
    `staticKilo`, `staticHuggingFace` and `staticTogether`, and their
    comments;
  * a recorded snapshot, `llmprovider/catalog/testdata/ranking-<date>/`,
    its golden test, and a test that each ranked list equals it;
  * a live measurement of each id the refresh adds;
  * dated notes in 0033-PLAN-public-model-metadata.md ~~and
    0036-PLAN-cache-write-token-count.md~~ *(corrected 2026-10-12, Phase
    5's deviation)*;
  * `README.md` Status, release notes, and the index, for `v1.10.2`.

### Out of scope

* The public metadata (D1 to D3): the other plan.
* Forced tool choice on the newest Claude models: its own record
  (0033-MADR D6). That record adds the models it fixes to `staticClaude`.
  Until then, Phase 1 drops any Claude id that refuses forced choice.
* The maintainer tool that proposes the next lists (D4): the other plan,
  `v1.8.0`.
* The gateway lists (Zen, Go, Hugging Face, Kilo, Together). All 30 ids are
  current. Only the Go comment is re-checked. *(Amended 2026-10-12: they are
  in scope for Phases 5 to 9.)*
* *(Amended 2026-10-12.)* For Phases 5 to 9:
  * the ranker's rules (0009-MADR §3 and §4). A pick the owner refuses
    becomes a ranker rule, by a 0009-MADR amendment (Q19), and is a
    deviation here;
  * the vendor lists (OpenAI, Claude, Gemini, Grok), which the command only
    checks;
  * the runtime catalog, which keeps reading the keyed listing;
  * the 2026-09-26 snapshot and its golden, which stay as they are.
* gobble-cli's own move to the new release.

## Rules for every phase

The rules of
[0031-PLAN-native-streaming-on-the-responses-wire.md](0031-PLAN-native-streaming-on-the-responses-wire.md)
apply unchanged:

1. approval, naming billed requests;
2. red first, and plants only on a scratch copy;
3. the 22-check gate;
4. coverage floors;
5. benchmarks, where a phase names them;
6. live checks: keys for presence only, nothing logged but statuses, shapes
   and redacted errors;
7. deviations stop and prompt;
8. one commit per phase, the owner's or on a same-turn ask;
9. no identifiers.

## Implementation Steps

### Phase 0: records

1. This PLAN, the MADR and
   [0033-PLAN-public-model-metadata.md](0033-PLAN-public-model-metadata.md),
   `proposed`.
2. `docs/README.md`: three rows, the count, and "I want to…" rows.
3. Records check, links and markdownlint; stage.

### Phase 1: measurement (live; billed: at most 32 requests of at most 64 output tokens)

The tests are behind `LLMPROVIDER_LIVE_0033`, in
`llmprovider/live_0033_measure_test.go`.

1. **Listings, unbilled**, with each key checked for presence only:
   * OpenAI `/v1/models`: each candidate is listed, and has no
     `shutdown_date`;
   * Anthropic `/v1/models`: each candidate is listed, with its
     `lifecycle`, `max_input_tokens`, `max_tokens` and `capabilities`;
   * Gemini `models.list`: each candidate is listed, with its token
     limits;
   * xAI `/v1/models`: each candidate is listed, with its reasoning
     efforts.
2. **One generation per candidate, billed:** 16 output tokens, text only.
   Every candidate must answer 200.
3. **Forced choice on each Claude candidate, billed:** `GenerateToolCall`.
   While Q5 is open, an id that answers 400 is dropped from `staticClaude`.
4. **Each new Grok effort menu, billed:** one request per documented
   effort, on `grok-4.7` and `grok-4.3`.
5. **The candidates**, from the official pages of 2026-10-11, ordered fast
   and cheap first, as the catalogs are. The owner approved them on
   2026-10-11 (0033-MADR Q8):

   | List | Candidates |
   | :--- | :--- |
   | `staticOpenAI` | `gpt-6-luna`, `gpt-5.6-luna`, `gpt-4.1-mini`, `gpt-5.6-terra`, `gpt-6.1-sol`, `gpt-6-astra` |
   | `staticGemini` | `gemini-3.5-flash-lite`, `gemini-3.8-flash`, `gemini-3.6-flash` |
   | `staticClaude` | `claude-haiku-5-5`, `claude-sonnet-5`, `claude-opus-4-8`, `claude-sonnet-4-6`; and, only if forced choice answers, `claude-sonnet-5-5`, `claude-opus-5-5` |
   | `staticGrok` | `grok-4.6`, `grok-4.5` (the Grok CLI's defaults, 0012-MADR §6), `grok-4.7`, `grok-4.3`, `grok-build-0.1` |

   Notes on the lists:
   * `gemini-3.1-flash-lite` is left out: its shutdown is 2027-05-07.
   * `gemini-3.1-pro-preview` is left out: it is a preview.
   * `claude-haiku-4-5` is left out: its retirement floor is 2026-10-15.
6. **Record** a table per list: status, listing fields, generation status,
   forced-choice status. The final lists are the candidates that pass. A
   list left with fewer than three ids is a deviation.

### Phase 2: the catalogs (D4)

1. **Red first:**
   * `catalog/retired_0033_test.go`: no `Static(id)` holds an id on a table
     of retired, scheduled or rerouted ids taken from the deprecations
     pages. It fails on today's lists, naming `gpt-4.1-nano`, `o4-mini`,
     `gemini-3.7-flash`, `gemini-3.5-flash`, `grok-4-fast-reasoning` and
     the rest.
   * Ranker tests: in a listing of current and retired ids, a current one
     ranks first:
     * `gpt-6-luna` over `gpt-4.1-nano`;
     * `gemini-3.8-flash` over `gemini-3.7-flash`;
     * `claude-haiku-5-5` over `claude-haiku-4-5`;
     * `grok-4.7` over `grok-3-mini-fast`.
   * `Label`: one id offered by two providers takes each provider's label.
2. **Change:**
   * the four lists, from Phase 1's table, with their check date;
   * `modelLabels` for every new id, keyed by provider;
   * the rankers and their anchors;
   * the Grok effort menus;
   * the OpenCode Go comment, corrected or confirmed by the Go docs.
3. **Goldens:**
   * The listing scenario's results may change where curation against a
     static list moved.
   * Each changed golden is listed in the record with the reason, and
     regenerated only for that reason.
   * No other golden changes.
4. **Plants:** a retired id put back into a list, and a ranker that
   prefers `mini` again, each fail their test.
5. Gate; stage.

### Phase 3: the guard (D4)

1. `llmprovider/live_catalog_0033_test.go`, run by the release's live
   suites:
   * for each provider with a key set, every `Static` id is in that
     provider's listing;
   * where the listing carries a lifecycle (Anthropic's `lifecycle`,
     OpenAI's `shutdown_date`), none is deprecated or scheduled.

   Listings are not billed. *(Amended by deviation D1, 2026-10-11:
   Anthropic's `lifecycle` lags its deprecations page. Phase 2's
   `TestStatic_NoRetiredIDs_0033` guards what no listing reports.)*
2. **Red first:** on a scratch copy with a retired id planted in a list,
   the test fails, naming the provider and the id.
3. Gate; stage.

### Phase 4: documentation and release

1. **`README.md` Status:** `v1.7.1`. **Release notes** in this PLAN.
2. **The live suites** on the release commit (billed, as the `v1.7.0`
   run), with the guard among them.
3. **The owner** commits, pushes, and tags `v1.7.1`.
4. **After the tag:**
   * CI;
   * the proxy;
   * apidiff `v1.7.0` to `v1.7.1`, which must show nothing;
   * a scratch consumer, and a scratch copy of gobble-cli on `v1.7.1`.
5. This PLAN `complete`; the index updated.

### Phase 5: records *(amended 2026-10-12)*

1. 0033-MADR's amendment "the second refresh, and what the command reads",
   with its research and Q13 to Q19, `accepted` (the owner, 2026-10-12).
2. **D4d's notes,** each dated, each saying that the removal was one
   workspace's disabled model, not a retirement (0033-MADR F1):
   * 0033-PLAN-public-model-metadata.md, Phase 4's "Found, outside this
     plan's scope";
   * the same PLAN's "Not done, and why";
   * ~~0036-PLAN-cache-write-token-count.md's "Not done, and why".~~
     *(Deviation, 2026-10-12: that PLAN never carried the finding; see
     Phase 5's record.)*
3. This PLAN `in-progress`; the index updated.
4. `make records-check`, `tools/linkcheck.py`, `tools/denyscan.py
   --changed`. Stage. The owner commits.

### Phase 6: the command *(amended 2026-10-12; D4a, D4b, D4c)*

1. **Red first** (`internal/catalogtool/propose_0033_test.go`, and
   `internal/cmd/catalogs/main_0033_test.go`):

   | Test | Asserts |
   | :--- | :--- |
   | `TestPropose_RankedListDiff` | for a ranked list, the proposals are exactly the difference between the static list and the ranked six: an id in the six and not the list is an addition whose reason names its rank; an id in the list and not the six is a removal "not in today's utility six" |
   | `TestPropose_RankedListNoBroadAdds` | a newer model with tools that the ranker did not pick is not proposed for a ranked list |
   | `TestPropose_RetiredRemovedFromAnyList` | an id on the retired table, or models.dev `deprecated`, is proposed for removal from a ranked list too |
   | `TestPropose_VendorReviewLines` | for a vendor list, a newer model is a `review` line, never an addition; a removal is unchanged |
   | `TestRun_PublicListings` | with every key set, the fetcher is asked for Zen, Go, Kilo and Hugging Face with an empty key, and for Together and the vendors with their keys |
   | `TestRun_Snapshot` | `-snapshot DIR` writes `zen.json`, `go.json`, `kilo.json`, `hf.json`, `together.json` and `metadata.json`, each byte for byte what the fetcher recorded or `Metadata.Encode` wrote, and `snapshot.json` with `recorded_at` and each file's SHA-256 |
   | `TestRun_SnapshotRefusesAnExistingDir` | `-snapshot` into a directory that exists exits 2, and writes nothing |

2. **Change:**
   * **`catalogtool`:**
     * `Sources` gains `Ranked map[llmprovider.ProviderID][]string`, each
       ranked list's utility six;
     * `Proposal`'s `Add bool` becomes `Kind`: `add`, `remove` or
       `review`. The package is internal;
     * a table of the ranked lists: Zen, Go, Kilo, Hugging Face and Together
       (Q17). Their additions come from `Ranked`, and the broad rule applies
       to the vendor lists only, as `review`.
   * **`cmd/catalogs`:**
     * the fetcher's `Listed` becomes `Catalog(ctx, provider, key)
       (catalog.Catalog, error)`. The live fetcher calls `catalog.List` with
       `WithProfile(ProfileUtility)` and metadata on for a ranked list, and
       with `WithoutModelMetadata` for a vendor list;
     * `publicListing`, the four gateways whose listing is read with no
       key. Its comment names the live guard's four constants, which it
       must match;
     * `-snapshot DIR`: the live fetcher's HTTP client records each listing
       body, and the command writes them, `Metadata.Encode`'s output and
       `snapshot.json`. It refuses a directory that exists. It edits no
       list (D4).
   * **`R30`:** the environment is still read in `KeysFromEnv` only.
3. **Plants,** each failing a named test:
   * `publicListing` without Zen;
   * a ranked list's removal dropped;
   * a vendor list's newer model proposed as an addition;
   * `-snapshot` writing a body other than the one recorded;
   * `snapshot.json` with a wrong hash;
   * `-snapshot` writing into an existing directory.
4. **Coverage:** both packages keep the default floor of 80%.
5. Gate; stage. The owner commits.

### Phase 7: measure, then record the snapshot *(amended 2026-10-12)*

1. **Run the command** (listings and models.dev: not billed). The printed
   diffs, for the five ranked lists and the vendor review lines, go in the
   Execution Record. The ids to add are those the diffs name, and nothing
   else.
2. **Measure each id to add** (D4: every candidate answers before it is
   listed):
   * `llmprovider/live_0033_refresh2_measure_test.go`,
     `TestLive_0033RefreshCandidates`, switched on by
     `LLMPROVIDER_LIVE_0033_REFRESH`. One `Generate` of at most 16 output
     tokens per id, with the provider's key. It logs the status and error
     only.
   * **The count goes to the owner before the run.** The research on
     2026-10-12 gives 9: Zen 1, Go 2, Kilo 1, Together 5. Step 1's diffs
     give the real count.
   * Each id's data policy is recorded where its provider documents one:
     Kilo's `mayTrainOnYourPrompts`, and Go's data table. `space-bunny`'s
     is undocumented (0033-MADR R6).
3. **The owner reviews the picks** with their measurements (Q15),
   `openai/gpt-6.1-sol` and `space-bunny` among them. A refusal is a
   deviation: the work stops, and a 0009-MADR amendment makes it a ranker
   rule (Q19). Then step 1 is run again.
   *(Deviation, 2026-10-12: three Together picks could not be served, and
   0009-MADR gained §3 item 10. Phase 7a builds it, and step 1 is run again.
   See Phase 7's record.)*
4. **Record the snapshot,** after the owner's approval of the picks:

   ```text
   go run ./llmprovider/internal/cmd/catalogs -snapshot llmprovider/catalog/testdata/ranking-<date>
   ```

   * `tools/denyscan.py` over each file. One hit stops the work.
   * The snapshot's sixes must equal step 1's diffs. A difference (a listing
     that moved between the two runs) is recorded, and the snapshot is
     taken again.
5. Gate; stage. The owner commits.

### Phase 7a: 0009-MADR §3 item 10 *(amended 2026-10-12, Phase 7's deviation)*

0009-MADR's amendment "Together ranks only what models.dev lists (§3 item
10)": on Together, an id that models.dev's `togetherai` section does not list
is neither ranked nor filled. It stays in `Usable`.

1. **Red first** (`llmprovider/catalog`):

   | Test | Asserts |
   | :--- | :--- |
   | `TestTogetherRanksOnlyModelsDevIDs` | a Together listing of ids models.dev lists and ids it does not, ranked with a document that covers only the first: the utility and capable sixes hold no uncovered id, neither ranked nor filled, and `Usable` keeps every id |
   | `TestTogetherWithoutMetadata` | with the document disabled, Together's curation is the fallback, unchanged by item 10 |

   The existing `TestListModelCatalog_HuggingFaceRanksWithMetadata`, which
   ranks an uncovered Hugging Face id, keeps the rule to Together.
2. **Change:** `metadataCurate` leaves Together's uncovered ids out of the
   candidates and the fill, through one predicate. Its comment cites §3
   item 10.
3. **Plants:**
   * item 10 applied to the ranking but not the fill;
   * item 10 applied to every provider: the Hugging Face test must fail;
   * item 10 removing the ids from `Usable`.
4. **The 2026-09-26 golden** is unchanged: it has no Together snapshot.
5. Gate; stage. The owner commits. Phase 7 then resumes at step 1: the
   command again, the measurement of any new addition (its count to the
   owner first), the owner's review, and the snapshot.

### Phase 7b: 0009-MADR §3 item 11 *(amended 2026-10-12, Phase 7's second deviation)*

0009-MADR's amendment "no small dense model in the utility six (§3 item
11)": under `ProfileUtility`, an id that names a dense model of 14B or less
is neither ranked nor filled, on every ranked provider.

1. **Red first** (`llmprovider/catalog`), `TestRankRecommended_SmallDenseUtility`,
   pure ranking over fixed candidates:
   * `x/model-9b` and `x/model-14B` are left out of the utility six, ranked
     or filled;
   * `x/model-15b`, `x/moe-30b-a3b` and `x/model-flash` stay eligible;
   * under `ProfileCapable`, `x/model-9b` is ranked.
2. **Change:** `utilityExcluded` gains item 11, through one predicate,
   `smallDense(id)`, with its pattern in production code. The criteria test
   keeps its own pattern, so the rule and its check do not share code.
3. **Plants:**
   * the threshold as `< 14`: `x/model-14B` stays;
   * the mixture-of-experts exemption dropped: `x/moe-30b-a3b` is left out;
   * item 11 under `ProfileCapable` too;
   * item 11 in the ranking only, not the fill.
4. **The 2026-09-26 golden** is unchanged: no small dense model was in its
   sixes.
5. Gate; stage. The owner commits. Phase 7 then resumes at step 1.

### Phase 7c: 0009-MADR §3 item 12 *(amended 2026-10-12, Phase 7's fourth deviation)*

0009-MADR's amendment "Together's ids measured not to serve (§3 item 12)":
on Together, an id on a dated list of ids measured not to answer is neither
ranked nor filled.

1. **Red first** (`llmprovider/catalog`), `TestTogetherMeasuredUnserved`:
   * a Together listing and document that cover `MiniMaxAI/MiniMax-M2.7` and
     two other models: the six holds neither ranked nor filled
     `MiniMaxAI/MiniMax-M2.7`, and `Usable` keeps it;
   * the same id on Hugging Face is ranked.
2. **Change:** `togetherUnserved`, a list beside `opencodeGoRegionGated`,
   with the date and the error each id gave. `excludedAtRequest` leaves its
   ids out on Together.
3. **Plants:**
   * the list applied on every provider: the Hugging Face case fails;
   * `excludedAtRequest` without the Together list.
4. Gate; stage. The owner commits. Phase 7 then resumes at step 1, and the
   new sixth pick is measured.

### Phase 8: the lists, and the golden *(amended 2026-10-12)*

1. **Red first** (`llmprovider/catalog`):

   | Test | Asserts |
   | :--- | :--- |
   | `TestListModelCatalog_Snapshot<date>` | the snapshot, replayed with the clock pinned to `snapshot.json`'s `recorded_at`, gives each ranked list's utility and capable sixes, as step 1 printed them |
   | `TestStatic_EqualsLatestSnapshot` | each ranked static list equals the latest snapshot's utility six. It fails until the lists change |
   | `TestStaticOpenCatalogs_UtilityCriteria` | gains `staticTogether` |

2. **Change:** the five ranked lists become the snapshot's utility sixes.
   Each comment names the snapshot, its date, and this amendment. A test
   that names an id leaving a list changes only where it asserts the list
   itself, and the record names each one. A label in `modelLabels` is
   added where a new id needs one.
3. **Plants:**
   * one id of `staticOpencodeZen` swapped for another usable id:
     `TestStatic_EqualsLatestSnapshot` fails;
   * the golden replay not pinning the clock;
   * `staticTogether` left out of the criteria test's table, with a
     planted free id.
4. **The live guard,** `TestLive_StaticCatalogsCurrent` (not billed),
   passes for all nine providers.
5. Gate; stage. The owner commits.

### Phase 9: release *(amended 2026-10-12)*

1. **`README.md` Status:** `v1.10.2`. **Release notes** in this PLAN.
2. Gate; stage. The owner commits, runs the disclosure guard, pushes, and
   tags `v1.10.2`.
3. **After the tag:**
   * CI on `main` and on the tag;
   * the proxy;
   * `tools/apidiff.py v1.10.1 v1.10.2`: no change;
   * `tools/consumer.py v1.10.2`;
   * `tools/offline.py v1.10.2`.
4. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with both lines and every plant's FAIL line in
  the Execution Record.
* **V2. The gate** clean at each phase. The listing goldens change only as
  recorded.
* **V3. Measurement:** Phase 1's table recorded before Phase 2 uses it.
* **V4. API:** apidiff `v1.7.0` to `v1.7.1` empty.
* **V5. Live:** the guard and the live suites on the release commit.
* *(Amended 2026-10-12, Phases 5 to 9.)*
  * **V6.** The command reads the guard's four public listings, and
    proposes each list by its rule. Its tests fail on each plant.
  * **V7.** Every id the refresh adds answered a live request, and the owner
    approved the picks.
  * **V8.** Each ranked list equals its recorded snapshot's utility six, by
    a test, and the snapshot's golden replays offline.
  * **V9.** apidiff `v1.10.1` to `v1.10.2`: no change.

## Rollout and Rollback

* **Rollout.** Three commits after the records, then `v1.7.1`, before
  2026-10-23.
* **Rollback, before the tag:** each phase reverts alone, newest first.
* **Rollback, after the tag:** fix forward in `v1.7.2`. A caller that
  pinned a removed id keeps working until its provider retires it. The
  SDK only stops recommending it.

## Execution Record

### Phase 0: records (2026-10-11)

* The MADR and both plans, written `proposed`. The owner answered Q1 to Q8
  with the recommended option of each, and the MADR was accepted with
  them. The owner committed the records as `9ec6571`.
* The index has three rows, 72 records, and two "I want to…" rows.

### Phase 1: measurement (2026-10-11)

`llmprovider/live_0033_measure_test.go`, behind `LLMPROVIDER_LIVE_0033`.
Each key was read from its provider's variable and sent only in its
request header. The test logs statuses, ids and listing metadata only.

**Step 1, the listings,** not billed. Each candidate, then each of today's
static ids:

| Provider | Listing | Candidates | Today's ids |
| :--- | :--- | :--- | :--- |
| OpenAI | 200, 133 models | all six listed, `shutdown_date null` | `gpt-4.1-nano` and `o4-mini`: `shutdown_date "2026-10-23"`; `gpt-4o-mini`, `gpt-4.1`, `gpt-4o`: `null` |
| Anthropic | 14 models | all six listed, `lifecycle "active"`, `max_input_tokens 1000000`, `max_tokens 128000` | `claude-haiku-4-5` not listed by that alias |
| Gemini | 62 models | all three listed, `inputTokenLimit 1048576`, `outputTokenLimit 65536`, `thinking true` | `gemini-3.7-flash`, `gemini-3.5-flash`, and both 2.5 models still listed for this key |
| xAI | 200, 14 models | all five listed | `grok-3-mini-fast`, `grok-3-mini`, `grok-4`, `grok-4-fast-reasoning` not listed |

The listings' own facts:

* **Anthropic's `capabilities`:**
  * `effort` levels (`xhigh` false on `claude-sonnet-4-6` only);
  * `image_input`;
  * `thinking.types`: `enabled` false on all but `claude-sonnet-4-6`, and
    `disabled` false on `claude-sonnet-5-5` and `claude-opus-5-5`.

  It has **no entry for forced tool choice**.
* **xAI's `capabilities.reasoning_effort`:**
  * `grok-4.7`, `grok-4.6` and `grok-4.5`: `low`, `medium`, `high`,
    `xhigh`, default `high`;
  * `grok-4.3`: `none`, `low`, `medium`, `high`, `xhigh`, default `low`;
  * `grok-build-0.1`: no `capabilities`.
* **xAI's `context_length`:**
  * 500000 for 4.7, 4.6 and 4.5;
  * 1000000 for 4.3;
  * 256000 for `grok-build-0.1`.

**Step 2, one generation per candidate,** billed: 18 requests of at most 16
output tokens, 13:52 UTC.

| Provider | Candidate | Status |
| :--- | :--- | :--- |
| OpenAI | `gpt-6-luna`, `gpt-5.6-luna`, `gpt-4.1-mini`, `gpt-5.6-terra`, `gpt-6.1-sol`, `gpt-6-astra` | 200 each |
| Gemini | `gemini-3.5-flash-lite` | 200 |
| Gemini | `gemini-3.8-flash`, `gemini-3.6-flash` | 200, `incomplete`: thinking spent the 16 tokens, as 0031-PLAN-native-streaming-on-the-other-wires.md deviation D2 found |
| Claude | `claude-haiku-5-5`, `claude-sonnet-5`, `claude-opus-4-8`, `claude-sonnet-4-6`, `claude-sonnet-5-5`, `claude-opus-5-5` | 200 each |
| Grok | `grok-4.6`, `grok-4.5`, `grok-build-0.1` | 200 each (`grok-4.7` and `grok-4.3` in step 4) |

**Step 3, forced choice,** billed: 6 requests. `GenerateToolCall` on each
Claude candidate:

| Candidate | Status |
| :--- | :--- |
| `claude-haiku-5-5`, `claude-sonnet-5`, `claude-opus-4-8`, `claude-sonnet-4-6` | 200, cut at 16 tokens (`length`): forced choice accepted |
| `claude-sonnet-5-5`, `claude-opus-5-5` | 400 `invalid_request_error`: "tool_choice: type "tool" and "any" are not supported for this model." |

**Step 4, the Grok efforts,** billed: 8 requests, `reasoning.effort` set on
the wire:

* `grok-4.7`: `low`, `medium`, `high` and `xhigh` each answered 200;
* `grok-4.3`: `low`, `medium`, `high` and `xhigh` each answered 200.

32 billed requests in all, as planned.

**The lists Phase 2 builds**, the candidates that passed:

| List | Ids, in order |
| :--- | :--- |
| `staticOpenAI` | `gpt-6-luna`, `gpt-5.6-luna`, `gpt-4.1-mini`, `gpt-5.6-terra`, `gpt-6.1-sol`, `gpt-6-astra` |
| `staticGemini` | `gemini-3.5-flash-lite`, `gemini-3.8-flash`, `gemini-3.6-flash` |
| `staticClaude` | `claude-haiku-5-5`, `claude-sonnet-5`, `claude-opus-4-8`, `claude-sonnet-4-6` |
| `staticGrok` | `grok-4.6`, `grok-4.5`, `grok-4.7`, `grok-4.3`, `grok-build-0.1` |

* `claude-sonnet-5-5` and `claude-opus-5-5` wait for 0033-MADR D6's record.
* Every list has three ids or more.
* **The Grok effort menus** Phase 2 adds, as the listing reports and step
  4 measured:
  * `grok-4.7`: `low`, `medium`, `high`, `xhigh`;
  * `grok-4.3`: `low`, `medium`, `high`, `xhigh`.

  `none` has no `Effort` constant, so an omitted effort stays omitted.

**Found, outside this plan's scope**, and not changed by it:

* **`grok-4.5`'s efforts.** The listing reports `xhigh` for `grok-4.5`.
  `providers/grok/reasoning.go` gives it `low`, `medium` and `high`, from
  the Grok CLI's catalog (0012-MADR §6), and clamps `xhigh` to `high`.
  Whether to follow the listing is the owner's call; it is not measured
  here.
* **Forced choice is not in Anthropic's metadata.** The record D6 names
  must take it from a measured table: Anthropic's `capabilities` has no
  entry for it. The MADR's measured note says so.

### Phase 2: the catalogs (2026-10-11)

**Change** (`llmprovider/catalog/models_catalog.go`):

* **The four lists, Phase 1's table, in order:**
  * `staticOpenAI`: `gpt-6-luna`, `gpt-5.6-luna`, `gpt-4.1-mini`,
    `gpt-5.6-terra`, `gpt-6.1-sol`, `gpt-6-astra`;
  * `staticGemini`: `gemini-3.5-flash-lite`, `gemini-3.8-flash`,
    `gemini-3.6-flash`;
  * `staticClaude`: `claude-haiku-5-5`, `claude-sonnet-5`,
    `claude-opus-4-8`, `claude-sonnet-4-6`;
  * `staticGrok`: `grok-4.6`, `grok-4.5`, `grok-4.7`, `grok-4.3`,
    `grok-build-0.1`.

  Each comment carries its 2026-10-11 check and what it left out.
* **Shared constants:** `modelGPT6Luna`, `modelGemini38Flash` and
  `modelGemini35Lite` name the ids the vendor lists share with OpenCode's,
  for `goconst`. Their file-wide `//nolint:goconst` became unused
  (`nolintlint`), and was removed.
* **`modelLabels`** is keyed by provider, and `Label` reads the provider's
  map, case-insensitively, as `Static` does:
  * every new id has a label;
  * the dropped ids lost theirs;
  * a gateway's fallback six keep the bare id, so Zen's
    `gemini-3.8-flash` no longer shows the Gemini label.
* **`rankOpenAIModel`:**
  * tiers: `luna`, then `nano`, `mini`, `terra`, `sol` and `astra`;
  * then a generation bonus: `gpt-6.1`, `gpt-6`, `gpt-5.6`, `gpt-5`,
    `gpt-4.1`.
* **`rankGrokModel`:**
  * it ranks by generation only: hints for 4.7, 4.6, 4.5 and 4.3, and
    anchors up to 4.7;
  * the `mini-fast`, `mini` and `fast-reasoning` bonuses are gone, with the
    tiers they favoured.
* **`rankGeminiModel` and `rankClaudeModel`** are unchanged. Their
  generation anchors already place `gemini-3.8-flash` above
  `gemini-3.7-flash`, and `claude-haiku-5-5` above `claude-haiku-4-5`: both
  cases passed in step 1's red run before any change.
* **`staticOpencodeGo`'s comment** is corrected, not confirmed:
  * the Go docs (last updated 2026-10-10) list DeepSeek V4.1 Flash, V4 Pro,
    V4 Flash and V4 Flash Vision Exp with no region limit;
  * they mark only Muse Spark 1.2 and 1.3 Contributor "limited regions";
  * the comment now names the gate (`opencodeGoRegionGated`, MADR 0009 §3
    item 8), and says that it is not re-measured.

  The gate itself is unchanged; see "Found" below.

**`llmprovider/providers/grok/reasoning.go`:**

* `grok-4.7` and `grok-4.3` take the four efforts. No effort sends `high`,
  as on every menu.
* The package doc's Degradations entry names them.

**Red first:**

* `catalog/refresh_0033_test.go`:
  * `TestStatic_NoRetiredIDs_0033` (a table of retired, scheduled, rerouted
    and restricted ids, and the three D6 waits);
  * `TestRank_PrefersCurrentModels_0033`;
  * `TestLabel_ByProvider_0033`.
* `providers/grok/reasoning_0033_test.go`: `TestGrokEffortMenus_0033`.

On the tree before the change:

```text
refresh_0033_test.go:53: openai: static catalog lists "gpt-4.1-nano", which is shuts down 2026-10-23
refresh_0033_test.go:53: openai: static catalog lists "o4-mini", which is shuts down 2026-10-23
refresh_0033_test.go:53: gemini: static catalog lists "gemini-3.7-flash", which is routed to gemini-3.8-flash
refresh_0033_test.go:53: gemini: static catalog lists "gemini-3.5-flash", which is routed to gemini-3.6-flash
refresh_0033_test.go:53: gemini: static catalog lists "gemini-2.5-flash", which is open only to past users
refresh_0033_test.go:53: gemini: static catalog lists "gemini-2.5-flash-lite", which is open only to past users
refresh_0033_test.go:53: claude: static catalog lists "claude-haiku-4-5", which is retirement floor 2026-10-15
refresh_0033_test.go:53: grok: static catalog lists "grok-3-mini-fast", which is not listed
refresh_0033_test.go:53: grok: static catalog lists "grok-3-mini", which is not listed
refresh_0033_test.go:53: grok: static catalog lists "grok-4", which is not listed; grok-4-0709 retired 2026-05-15
refresh_0033_test.go:53: grok: static catalog lists "grok-4-fast-reasoning", which is retired 2026-05-15
refresh_0033_test.go:83: openai: Rank("gpt-6-luna") = 150, not above Rank("gpt-4.1-nano") = 200
refresh_0033_test.go:83: openai: Rank("gpt-5.6-terra") = 150, not above Rank("o4-mini") = 180
refresh_0033_test.go:83: grok: Rank("grok-4.7") = 71, not above Rank("grok-3-mini-fast") = 230
refresh_0033_test.go:83: grok: Rank("grok-4.3") = 40, not above Rank("grok-4-fast-reasoning") = 140
refresh_0033_test.go:102: Label(gemini, "gemini-3.8-flash") = "gemini-3.8-flash", want a label starting "Gemini 3.8 Flash"
refresh_0033_test.go:114: Label(opencode-zen, "gemini-3.5-flash-lite") = "Gemini 3.5 Flash-Lite  [ultra-low latency]", want the bare id: a gateway's fallback six carry no label
reasoning_0033_test.go:10: clampReasoningEffort("grok-4.7", "low") = "", want "low"
reasoning_0033_test.go:10: clampReasoningEffort("grok-4.3", "") = "", want "high"
```

The record above is abridged: the full run also failed the other new
labels and the remaining Grok efforts, each in the same form.

**Existing tests whose expectations the record changes.** Each keeps its
meaning:

| Test | Was | Now | Why |
| :--- | :--- | :--- | :--- |
| `model_label_test.go`: `TestModelLabel`, `TestStaticIDs_Labelled`, `TestLabels_NoOrphans`, `unlabelledStaticIDs` | a flat map of ids | keyed by provider; `TestModelLabel` reads `claude-haiku-5-5` | labels are per provider; `gpt-6-luna` is labelled for OpenAI and excused for OpenCode Go |
| `model_matcher_test.go`: `TestSearchModels_NoSubsequenceOverLabels`, `TestSearchModels_LabelFromModelLabel`, `withLabel` | `Label("", "claude-haiku-4-5")`, a flat map | `Label(claude, "claude-haiku-5-5")`, a provider argument | a provider-less label no longer exists; the labelled id is the current Haiku |
| `model_matcher_test.go`: `TestSearch_CompactQuery` | `Static(openai)`, which held `gpt-4.1` | the former list, literally | the test is about compact queries, not the list |
| `discovery_catalog_test.go`: `TestListModelCatalog_OpenAIRecommendedIsCurated` | a listing of the old ids | a listing holding every new `staticOpenAI` id among others | `Recommended` must still equal `staticOpenAI` in catalog order |
| `models_catalog_test.go`: `TestRankGenerations` | `grok-4.7` 71, `grok-5` 90 | 70 and 100 | 4.7 is now a hint and the newest anchor |
| `models_catalog_test.go`: `TestRankGrokModel` | the mini tiers rank above their generation | generation order, 4.7 down to `grok-3-mini-fast`; no mini bonus | D4 reverses the preference: xAI no longer lists those tiers |
| `providers/grok/reasoning_test.go`: one row of `TestGrokEffortMenus_KeepMenuValues` | `grok-4.7` sends no effort | `grok-build-0.1` sends none | `grok-4.7` has a menu now; `grok-build-0.1` reports none |
| `wizard/model_select_edge_test.go`: `TestConfigureLLM_FallbackExclusionIgnoresCase` | the primary `Claude-Haiku-4-5` | `Claude-Haiku-5-5` | the primary must be a static id to be excluded |
| `wizard/model_select_test.go`: `TestConfigureLLM_SearchAgain` | the label of `claude-haiku-4-5` | of `claude-haiku-5-5` | the haiku in the list |

**The golden.** One G-wire golden changed, and the record explains it:

```text
testdata/wire/claude/listing.json: 2 difference(s) from the golden file
  result[0]: want "claude-haiku-4-5", got "claude-sonnet-5"
  result[1]: want "claude-sonnet-5", got "claude-haiku-4-5"
```

* The canned listing holds `claude-sonnet-5` and `claude-haiku-4-5`.
  `claude-haiku-4-5` left `staticClaude`, so curation puts the static
  `claude-sonnet-5` first and fills with the other.
* Only `result` changed; the requests are the same.
* It was regenerated alone (`-run 'TestWireGoldens/claude/listing$'
  -update`). Every other golden passes unchanged.

**Green:** `go test -race -count=1 ./...` passes.

**Plants** on a scratch clone, each failing:

| Plant | Test | FAIL line |
| :--- | :--- | :--- |
| `o4-mini` back in `staticOpenAI` | `TestStatic_NoRetiredIDs_0033` | `openai: static catalog lists "o4-mini", which is shuts down 2026-10-23` |
| the Grok ranker prefers `mini-fast` again | `TestRank_PrefersCurrentModels_0033`, `TestRankGrokModel` | `grok: Rank("grok-4.7") = 70, not above Rank("grok-3-mini-fast") = 200`; `a mini tier still earns a bonus over its generation` |
| the OpenAI ranker loses `luna` | `TestRank_PrefersCurrentModels_0033` | `openai: Rank("gpt-6-luna") = 200, not above Rank("gpt-4.1-nano") = 200` |
| `Label` ignores the provider | `TestLabel_ByProvider_0033` | `Label(opencode-go, "gpt-6-luna") = "GPT-6 Luna …", want the bare id` |
| `grok-4.3` has no menu | `TestGrokEffortMenus_0033` | `clampReasoningEffort("grok-4.3", "low") = "", want "low"` |

**Pre-add:** 12 files clean. The first run found the unused
`//nolint:goconst`, removed above.

**Found, outside this plan's scope**, not changed:

* **OpenCode Go's DeepSeek gate.** The docs no longer mark the four
  DeepSeek models region-limited, but `opencodeGoRegionGated` still keeps
  them out of the ranking (MADR 0009 §3 item 8). Whether a workspace
  without region `cn` is now served them needs a measurement and a record.
* **`grok-4.5`'s `xhigh`,** as Phase 1 found.

### Deviation D1 (2026-10-11): Anthropic's listing lags its deprecations page

* **Found** in Phase 3, step 2. Besides the plants the step names, a third
  plant put `claude-sonnet-4-5-20250929` in `staticClaude`, to exercise the
  guard's `lifecycle` branch. The guard passed:

  ```text
  live_catalog_0033_test.go:208: claude: 14 models listed
  --- PASS: TestLive_StaticCatalogsCurrent/claude (0.24s)
  ```

* **Why:** a scratch probe printed every entry's lifecycle fields.
  * Anthropic lists the model with `lifecycle="active"`,
    `deprecated_at=null` and `retires_at=null`.
  * Its deprecations page says it was deprecated on 2026-09-30, and
    retires on 2026-11-30.
  * models.dev's `anthropic` section has no `status` for it, nor its
    `google` section for the rerouted `gemini-3.7-flash` and
    `gemini-3.5-flash`.
  * models.dev does mark OpenAI's `gpt-4.1-nano` and `o4-mini`
    `deprecated`.
* **What it contradicts:**
  * 0033-MADR D2 makes Anthropic's listing authoritative for `Status` and
    `Retires`;
  * D4 has the guard catch what a listing's lifecycle marks.

  No source the guard reads carries Anthropic's deprecations, nor Gemini's
  reroutes. The error is in the pair, not in existing code.
* **Options put to the owner:**
  * two guards, and D2 amended (recommended);
  * the guard parsing the deprecations pages;
  * keeping D2 and noting the lag.
* **Decision.** The owner chose two guards and D2 amended. 0033-MADR records
  it as "Amendment (2026-10-11): sources that lag".
  * The live guard stays as built, and its doc comment says what it cannot
    see.
  * Phase 2's `TestStatic_NoRetiredIDs_0033` guards the rest.
  * 0033-PLAN-public-model-metadata.md takes the D2 change, and the one
    table for its tool.
* **Files added to the phase:** none beyond the guard; its doc comment
  names the lag.

### Phase 3: the guard (2026-10-11)

**`llmprovider/live_catalog_0033_test.go`:** `TestLive_StaticCatalogsCurrent`
runs with `-run Live`, in the release's live suites.

* For every provider with a static catalog, it reads that provider's
  listing:
  * with the key for OpenAI, Anthropic, Gemini and xAI;
  * without one for OpenCode Zen and Go, Hugging Face and Kilo;
  * for Together, with the key behind `LLMPROVIDER_LIVE_TOGETHER`.
* It fails on a static id the listing lacks, or one the listing marks
  stale (OpenAI's `shutdown_date`, Kilo's `expiration_date`, Anthropic's
  `lifecycle` other than `active`).
* A 429 or 5xx skips. An unset key skips.
* Its doc comment says what it cannot see (deviation D1).

**Green on the tree**, every provider passing:

| Provider | Models listed |
| :--- | :--- |
| `openai` | 133 |
| `claude` | 14 |
| `gemini` | 62 |
| `grok` | 55 (ids and aliases) |
| `opencode-zen` | 87 |
| `opencode-go` | 45 |
| `huggingface` | 126 |
| `kilo` | 390 |
| `together` | 265 |

**Red** (step 2), on a scratch clone, listings only:

| Plant | Result |
| :--- | :--- |
| `o4-mini` in `staticOpenAI` | `openai: static id "o4-mini" is listed as stale: shutdown_date "2026-10-23"`; FAIL |
| `grok-3-mini` in `staticGrok` | `grok: static id "grok-3-mini" is not in its provider's listing`; FAIL |
| `claude-sonnet-4-5-20250929` in `staticClaude` | PASS: deviation D1 |

The third plant went beyond the step, to exercise the `lifecycle` branch.
It found the lag recorded in deviation D1, and the owner's decision there.
For that model, Phase 2's unit test is the guard: its table lists it as
retiring on 2026-11-30.

**Pre-add:** clean. **Gate:** all 22 checks pass.

### Phase 4: documentation and release (2026-10-11)

**Step 1**, on `443690d` (Phase 3):

* **`README.md`:** Status names `v1.7.1`, with a `v1.7.1` bullet.
* **`docs/architecture.md`:**
  * the catalog paragraph says `Label` is per provider;
  * it names the two guards.

  `Label`'s behaviour changed, and the architecture describes the system
  as it is (R43), so this file is added to the step.
* **Package docs:** the Grok package doc changed in Phase 2.
  **`api-standards.md`:** unchanged.

**Step 2, the live suites,** on the release commit `8981532`.

* **Run:** 14:52 to 14:55 UTC.
  * Every provider's key variable was set, checked for presence only.
  * `LLMPROVIDER_LIVE_TOGETHER=1` was set, and a local Ollama server
    answered.
  * The suites that need a person, a subscription or their own switch
    stayed off.
* **Command:** `go test -tags live_gateways -count=1 -v ./llmprovider/...
  -run Live`.
* **Result:** exit 0.
  * Top level: 75 passed, 27 skipped, 0 failed.
  * Subtests: 85 passed, 5 skipped, 0 failed.

  The suites log tests, not requests, so the requests were not counted.
* **The guard,** `TestLive_StaticCatalogsCurrent`, passed for all nine
  providers.
* **The streaming tests** passed as in `v1.7.0`. The Gemini ones now run
  on `gemini-3.5-flash-lite`, the new `Static(gemini)[0]`, with 2 text
  deltas, then `EventDone`, and an interaction id on the stored one.

**The 27 top-level skips:**

* **Off by their own switch, 24:**
  * the 0028, 0031, 0031B, 0032 and 0033 measurements, 13;
  * the ChatGPT and Codex CLI subscription, 4;
  * Gemini's rate-limit drain, 1;
  * the Grok CLI, 1;
  * the browser and device sign-ins and the OpenAI sign-in probe, 5.
* **No active model:** `TestLive_OpencodeInterleavedReasoningReplay`, "no
  active opencode-go model among [kimi-k2.6]", as before.
* **Transient, 2,** each rerun once alone at 14:56 UTC:
  * `TestLive_KiloChatCompletions`: "kilo stream 503: Upstream error from
    Nvidia: Service temporarily overloaded". Its rerun passed.
  * `TestLive_KiloReasoningSpelling`: "gateway returned HTTP 429 (free tier
    limits)", on `kilo-auto/free`. Its rerun answered 429 again, so it
    stays skipped under 0027-MADR's transient rule.
    * It runs on Kilo's free tier, which nothing in this release touches.
    * It passed on its rerun in the `v1.6.0` run, and in the `v1.7.0` run.

**The 5 subtest skips** are the same five OpenCode Zen ones, for the same
account reasons, as in `v1.7.0`:

* 403 "Model access is disabled" on `responses`;
* 402 "Insufficient account funds" on `messages`, `chat_completions` and
  `google`, and on `TestLive_OpencodeGoogleTwoCallRoundTrip/zen`.

Every check that ran against a service passed on the release commit, apart
from the one transient skip above.

**Found, outside this plan's scope:** four live test files still name
`claude-haiku-4-5`, which can be retired from 2026-10-15:

* `live_claude_test.go`;
* `live_gateways_test.go`;
* `live_stream_0031b_test.go`;
* `live_0032_measure_test.go`.

They pass today. After its retirement they would fail, so the next
record that touches the live suites moves them to a current model.

**Step 3, the release.**

* The owner committed step 2's record as `0adc791`, pushed, and tagged
  `v1.7.1` on it, an annotated tag (`09c3c63`).
* From the release commit `8981532`, `0adc791` changes this PLAN only.

**Step 4, after the tag:**

| Check | Result |
| :--- | :--- |
| CI on `main` at `0adc791` (run `38149317648`) | success |
| CI on the tag (run `38149800321`, a push of `v1.7.1`) | success |
| `go list -m -json …@v1.7.1` through `proxy.golang.org` | `Version` `v1.7.1`, `GoVersion` `1.27.2`, `Origin.Hash` `0adc791…` (`refs/tags/v1.7.1`) |
| apidiff, `v1.7.0` to `v1.7.1` (both tags archived, without `-incompatible`) | no output: no change to the exported API |
| a scratch consumer of every public package on `v1.7.1` | builds with `CGO_ENABLED=0` for linux, windows and darwin, amd64 and arm64. `go run` prints each provider's first static id and its label: `gemini-3.5-flash-lite`, `gpt-6-luna`, `claude-haiku-5-5` and `grok-4.6`, each with its "★ Recommended" label; each gateway's first id, with its bare id as its label |
| a scratch copy of gobble-cli, moved from `v1.2.1` to `v1.7.1` | baseline tests pass. `go get` upgrades `v1.2.1 => v1.7.1`. Then build, vet on three platforms, `go test`, `go test -race` and golangci-lint (`0 issues.`) all pass. |

**Step 5.** Every criterion is met, so this PLAN is `complete`:

* **V1:** red then green, and each plant's FAIL line, in Phases 2 and 3.
* **V2:** the gate at each phase. Only the Claude listing golden changed,
  as Phase 2 recorded.
* **V3:** Phase 1's table, recorded before Phase 2.
* **V4:** step 4's apidiff, empty.
* **V5:** the guard and the live suites, step 2.

**Not done, as scoped:**

* **The newest Claude models** (`claude-sonnet-5-5`, `claude-opus-5-5`,
  `claude-fable-5-1`) wait for 0033-MADR D6's record.
* **The metadata and the maintainer tool:**
  [0033-PLAN-public-model-metadata.md](0033-PLAN-public-model-metadata.md),
  `v1.8.0`.
* **The findings,** each waiting for a measurement and a record:
  * OpenCode Go's DeepSeek region gate;
  * `grok-4.5`'s `xhigh`;
  * the four live test files that name `claude-haiku-4-5`.
* **gobble-cli's move to `v1.7.1`** is its own record. The scratch copy
  above shows that the move compiles and passes.

#### Release notes: `v1.7.1`

`v1.7.1` brings the curated model catalogs up to date with the providers'
official pages (0033-MADR D4). The exported API does not change.

**The menus.** `catalog.Static`, and each provider's
`Descriptor.StaticModels`:

| Provider | Now | Dropped, and why |
| :--- | :--- | :--- |
| OpenAI | `gpt-6-luna`, `gpt-5.6-luna`, `gpt-4.1-mini`, `gpt-5.6-terra`, `gpt-6.1-sol`, `gpt-6-astra` | `gpt-4.1-nano` and `o4-mini` (shut down 2026-10-23); `gpt-4o-mini`, `gpt-4.1`, `gpt-4o` (superseded) |
| Gemini | `gemini-3.5-flash-lite`, `gemini-3.8-flash`, `gemini-3.6-flash` | `gemini-3.7-flash` and `gemini-3.5-flash` (rerouted to newer models); `gemini-2.5-flash` and `gemini-2.5-flash-lite` (open only to past users) |
| Claude | `claude-haiku-5-5`, `claude-sonnet-5`, `claude-opus-4-8`, `claude-sonnet-4-6` | `claude-haiku-4-5` (retirement floor 2026-10-15) |
| Grok | `grok-4.6`, `grok-4.5`, `grok-4.7`, `grok-4.3`, `grok-build-0.1` | `grok-4-fast-reasoning` (retired 2026-05-15); `grok-4`, `grok-3-mini`, `grok-3-mini-fast` (no longer listed) |

* Every id was listed by its provider, and answered a generation, on
  2026-10-11. Each menu's first entry is its recommendation.
* `claude-sonnet-5-5`, `claude-opus-5-5` and `claude-fable-5-1` are not
  listed yet. They refuse a forced tool choice, which `GenerateToolCall`
  sends, and wait for the record 0033-MADR D6 names.
* The gateways' menus (OpenCode Zen and Go, Hugging Face, Kilo, Together)
  are unchanged: every id was current.

**Ranking.** In a live listing:

* OpenAI ranks `luna` first, then `nano`, `mini`, `terra`, `sol` and
  `astra`, and a newer generation above an older one of the same tier.
* Grok ranks by generation, and no longer favours the retired mini tiers.

**Labels.** `catalog.Label` reads the provider. An id two providers offer
takes each one's label, and a gateway's fallback models show their bare id.

**Grok efforts.** `grok-4.7` and `grok-4.3` take `low` to `xhigh`, as xAI's
listing reports. Before, they were sent no effort.

**Guards:**

* a unit test fails on a static id the deprecations pages retire;
* `TestLive_StaticCatalogsCurrent`, in the live suites, fails on one a
  listing lacks or marks stale.

**Upgrading:**

* No code change is needed.
* A caller that pinned a dropped id keeps working until its provider
  retires it. The SDK only stops recommending it.
* A caller that reads `Label` for a gateway's model now gets the bare id
  where it got a vendor's label.

### Phase 5: records (2026-10-12)

* 0033-MADR's amendment "the second refresh, and what the command reads"
  was committed as `caffd35`. Its research and Q13 to Q19, and Phases 5 to
  9 here, were committed as `f9df589`, accepted by the owner.
* **D4d's notes,** each dated, in 0033-PLAN-public-model-metadata.md: on
  Phase 4's "Found, outside this plan's scope", and on its "Not done, and
  why". Each says the removal was one workspace's disabled model, and cites
  F1.

**Deviation (2026-10-12): 0036-PLAN never carried the finding.** Step 2
planned a note in 0036-PLAN-cache-write-token-count.md's "Not done, and
why". That list names only the Anthropic reasoning count and the Zen
cache-read spelling. The amendment's claim that 0036-PLAN repeated the
finding was written without checking. The owner chose a dated correction:

* 0033-MADR's amendment strikes "0036-PLAN-cache-write-token-count.md
  repeated it" and its D4d's mention of 0036-PLAN, each with a dated note.
  The finding's repetition was in 0033-PLAN-public-model-metadata.md's own
  "Not done, and why".
* This PLAN's scope and step 2 strike the 0036-PLAN bullet, each with a
  dated note.
* 0036-PLAN is unchanged.

**Checks:** `make records-check`, `tools/linkcheck.py` and
`tools/denyscan.py --changed`, clean.

Committed by the owner as `e43f89d`.

### Phase 6: the command (2026-10-12)

**Change:**

* **`internal/catalogtool/propose.go`:**
  * `Proposal.Add bool` is now `Kind`: `add`, `remove` or `review`.
    `String` prints the kind;
  * `Sources.Ranked` is each ranked list's utility six;
  * `rankedLists`, and `Ranked(provider)`: Zen, Go, Kilo, Hugging Face and
    Together;
  * `Propose`:
    * removes from any list an id the retired table or models.dev retires,
      or the listing leaves out;
    * removes from a ranked list one its six leaves out ("not in today's
      utility six");
    * adds to a ranked list each id of its six it lacks, with its rank;
    * turns a vendor list's newer models into `review` lines, "review
      against the official page".
* **`internal/cmd/catalogs/main.go`:**
  * `publicListing`, the guard's four public gateways, read with no key.
    Its comment names the guard's constants;
  * the fetcher's `Listed` is now `Catalog(ctx, provider, key, ranked)`,
    plus `Listing(provider)`. The live fetcher ranks a ranked list with
    `WithProfile(ProfileUtility)` and metadata, and lists a vendor with
    `WithoutModelMetadata`;
  * `listingRecorder` keeps the one GET whose path ends in `/models`. None,
    or two, is an error;
  * `-snapshot DIR`:
    * it refuses a `DIR` that exists, before any listing is read;
    * it writes the five listing bodies, `Metadata.Encode`'s output and
      `snapshot.json`, with `recorded_at` and each file's SHA-256;
    * it writes into `DIR.partial`, then renames it into place, and
      removes it if a write or the rename fails;
    * it writes nothing unless every ranked listing was read;
  * `liveFetcher` gains `metadataURL` and `extra` options. Their zero values
    are the real endpoints, and a test points them at its own server.
* **`R30`:** the environment is still read only in `KeysFromEnv`.

**Tests,** as step 1 lists them, and three more:

* `catalogtool`: `TestPropose_RankedListDiff`,
  `TestPropose_RankedListNoBroadAdds`, `TestPropose_RetiredRemovedFromAnyList`,
  `TestPropose_VendorReviewLines` and `TestRankedLists`;
* `cmd/catalogs`: `TestRun_PublicListings`, `TestRun_Snapshot`,
  `TestRun_SnapshotRefusesAnExistingDir`, and also
  `TestRun_SnapshotNeedsEveryRankedListing`, `TestSnapshotFiles` (the
  snapshot names every ranked list, and nothing else), `TestListingRecorder`
  and `TestLiveFetcher`.

**Red first:**

* `catalogtool`: the build failed on `unknown field Ranked`, `unknown
  field Kind` and `undefined: KindRemove`. With the new shapes and the old
  behaviour, four tests failed:
  * the ranked diff got the broad additions, `zen-unpicked` among them;
  * the vendor test got `claude add claude-new`;
  * `Ranked` was false for all five gateways.

  `TestPropose_RetiredRemovedFromAnyList` passed before the change, since
  the retired check already applied to every list. It stays as a guard.
* `cmd/catalogs`: the build failed on `undefined: snapshotFiles`,
  `f.asked undefined` and `unknown field bodies`. The behaviour of each of
  its tests was proven by the plants below, not by a separate stub run.

**Two existing assertions changed, by Q18.** Claude's list is a vendor
list, so its newer model is now a `review` line:

* `TestPropose_AddsACurrentModel_0033` wants `Kind: KindReview`, and the
  reason prefixed "review against the official page; ";
* `TestRun_Proposes_0033` wants `claude\treview\tclaude-next\t`.

Each still pins the same model, status, date and price.

**Green:** both packages pass. Coverage is 98.5% for `catalogtool` and
94.2% for `cmd/catalogs`. It was 77.8% before `TestLiveFetcher`, under the
80% floor; step 4 is met.

**Plants,** through `tools/plant.py`:

| Plant | Caught by |
| :--- | :--- |
| `publicListing` without Zen | `TestRun_PublicListings` |
| a ranked list's removal dropped | `TestPropose_RankedListDiff` |
| a vendor list's newer model proposed as an addition | `TestPropose_VendorReviewLines`, `TestPropose_AddsACurrentModel_0033`, `TestRun_Proposes_0033` |
| `-snapshot` writing a body other than the one recorded | `TestRun_Snapshot` |
| `snapshot.json` with a wrong hash | `TestRun_Snapshot` |
| `-snapshot` writing into an existing directory | `TestRun_SnapshotRefusesAnExistingDir`: `9 listing(s) were read before the directory was refused` |
| the recorder keeping more than one listing | `TestListingRecorder` |

**A plant not caught at first.** On the first run, "`-snapshot` writing
into an existing directory" passed. With the early check gone, the rename
onto the existing directory still failed, so the test saw exit 2 and an
empty directory. But the run had read every listing first, and left
`DIR.partial` beside the target. So:

* the test now also requires that no listing was read, and that no
  `DIR.partial` is left;
* `writeSnapshot` removes `DIR.partial` when the rename fails.

On the second run, all seven plants were caught.

**The first gate failed, 18 of 23.** `TestNoAmbientState` (R30) found
`llmprovider/internal/cmd/catalogs/main.go:147: http.DefaultTransport is
shared by the process`: the recorder sent its requests through net/http's
shared transport. `pre-add-check`, `test-race-cover`, `test-shuffle`,
`coverage-check` and `gate-selftest` (which runs `coverage-check` on a
clean copy) all failed on that one test. The per-file pre-add check had
passed, since it does not run `internal/ambientcheck`. The fix, within this
phase: `newListingRecorder` builds the recorder over the SDK's bounded
default transport, `transport.DefaultClient().Transport`. The ambient check
then passed, and the seven plants were run again, and all caught.

**Checks:** `make pre-add-check` on the six Go files, clean (gofmt,
golangci-lint, go vet, go test, govulncheck).

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 8.8, after the fix.

Committed by the owner as `35b9321`.

### Phase 7: measure, then record the snapshot (2026-10-12)

**Step 1, the command,** live (listings and models.dev; not billed): 18
proposals, against 50 from the old rule. The vendor lists drew none.

| List | Add (rank in today's utility six) | Remove |
| :--- | :--- | :--- |
| OpenCode Zen | `claude-haiku-5-5` (1) | `gemini-3.8-flash` |
| OpenCode Go | `claude-haiku-5-5` (1), `space-bunny` (6) | `hy3`, `mimo-v2.6-pro` |
| Kilo | `openai/gpt-6.1-sol` (6) | `thinkingmachines/inkling` |
| Together | `zai-org/GLM-5.3-Flash` (1), `deepseek-ai/DeepSeek-V4-Flash-0731` (2), `Qwen/Qwen3.8-Flash` (4), `arcee-ai/trinity-mini` (5), `mistralai/Devstral-Small-2505` (6) | `zai-org/GLM-5.3`, `moonshotai/Kimi-K3`, `MiniMaxAI/MiniMax-M3`, `Qwen/Qwen3.6-Plus`, `openai/gpt-oss-120b` |
| Hugging Face | none | none |

**Free facts for the owner's review:**

* Kilo's listing: `openai/gpt-6.1-sol` has `mayTrainOnYourPrompts: false`,
  `preferredIndex` 5, and is priced at $2.00 and $10.00 per million. For
  `thinkingmachines/inkling` the figures are `false`, none, and $0.95 and
  $4.05.
* Together's listing: `mistralai/Devstral-Small-2505` was created on
  2025-05-21 and is priced at 0, and `arcee-ai/trinity-mini` was created on
  2026-05-15.

**Step 2, the measurement** (`llmprovider/live_0033_refresh2_measure_test.go`,
`TestLive_0033RefreshCandidates`). The owner approved 9 billed requests,
each "Say hello." with at most 16 output tokens. Every key variable was set,
checked for presence only.

| Candidate | Status |
| :--- | :--- |
| opencode-zen `claude-haiku-5-5` | 200 |
| opencode-go `claude-haiku-5-5` | 400 `ModelProtocolUnsupported`, on `/chat_completions` (Deviation 1) |
| opencode-go `space-bunny` | 200 |
| kilo `openai/gpt-6.1-sol` | 200 |
| together `zai-org/GLM-5.3-Flash` | 200; `ErrIncomplete`, its 16 tokens spent reasoning |
| together `deepseek-ai/DeepSeek-V4-Flash-0731` | 200; `ErrIncomplete`, as above |
| together `Qwen/Qwen3.8-Flash` | 400 `streaming_required` |
| together `arcee-ai/trinity-mini` | 400 `model_not_available`, "non-serverless model" |
| together `mistralai/Devstral-Small-2505` | 400 `dedicated_endpoint_not_running` |

**Deviation 1 (2026-10-12): the measurement's routing.** The test copied
0033's older probe, which turns model metadata off. Without it, OpenCode
routes a `claude-` model to Messages on Zen only (`heuristicRoute`). On Go the
route comes from models.dev's npm, `@ai-sdk/anthropic`. So Go's request went
to Chat Completions, which a Claude model does not serve.

* The owner chose to re-measure as the runtime routes it.
* The test leaves metadata on, and its comment says why.
* The re-measure was 1 billed request: opencode-go `claude-haiku-5-5`,
  status 200.

**Deviation 2 (2026-10-12): Together's unservable picks.** The three Together
picks that failed are exactly those models.dev's `togetherai` section does not
list. The two it lists answered. Together's listing has no field that tells
them apart.

* The owner chose a ranker rule (Q19): 0009-MADR's amendment "Together ranks
  only what models.dev lists (§3 item 10)".
* Phase 7a builds it, and step 1 is then run again.

**Q15, the owner's review:** `openai/gpt-6.1-sol` and `space-bunny` are
approved as the ranker chose them.

Phase 7 used 10 billed requests so far: 9, then 1.

Phase 7's record to here, 0009-MADR's amendment and the measurement test
were committed by the owner as `02873ea`.

### Phase 7a: 0009-MADR §3 item 10 (2026-10-12)

**Change** (`llmprovider/catalog/model_metadata.go`):

* `listedOnly(provider)` is true for Together alone. Its comment cites §3
  item 10 and the exception to "Unknown values never exclude".
* `metadataCurate`, once the document has the provider's section, drops
  from a clone of `usable` every id the section does not list, where
  `listedOnly` holds. Both the candidates and the fill are built from it.
  `Catalog.Usable` is built outside this function, so it keeps every id.
* With the document failed or disabled, `metadataCurate` returns the
  fallback, so item 10 does not apply.

**Tests** (`llmprovider/catalog/together_item10_test.go`):

* `TestTogetherRanksOnlyModelsDevIDs`: a listing of two covered and two
  uncovered chat models, ranked under both profiles. The six holds both
  covered ids and neither uncovered one, and `Usable` keeps all four.
* `TestTogetherWithoutMetadata`: with the document disabled, the six is
  `curateTogether`'s, and it fills from the whole listing.

**Red first:** `TestTogetherRanksOnlyModelsDevIDs` failed under both
profiles: `Recommended [t/covered-a t/covered-b t/uncovered-a t/uncovered-b]
holds t/uncovered-a, which models.dev does not list`, and the same for
`t/uncovered-b`. `TestTogetherWithoutMetadata` passed before the change, as
it must: it guards that item 10 leaves the fallback alone.

**Green:** `go test ./llmprovider/catalog/`, every test.

**Plants,** through `tools/plant.py`:

| Plant | FAIL line |
| :--- | :--- |
| item 10 applied to the ranking but not the fill | `TestTogetherRanksOnlyModelsDevIDs`: `Recommended [t/covered-a t/covered-b t/uncovered-a t/uncovered-b] holds t/uncovered-a`, under both profiles |
| item 10 applied to every provider | `TestListModelCatalog_HuggingFaceRanksWithMetadata`: `ranked = [c/flash b/mid a/large v/no-reason]`, with `d/uncovered` lost |
| the filter mutating `Usable` in place, without its clone | `TestTogetherRanksOnlyModelsDevIDs`: `Usable [t/covered-a t/covered-b  ] lacks t/uncovered-a; item 10 keeps every id searchable` |

**The 2026-09-26 golden** is unchanged; it has no Together snapshot.

**Checks:** `make pre-add-check` on the two Go files, clean (gofmt,
golangci-lint, go vet, go test, govulncheck).

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 9.7.

Committed by the owner as `09b3798`.

**Step 1 again, after Phase 7a** (listings and models.dev; not billed): 16 proposals. Item 10
held: Together's six named no id models.dev does not list. Together's diff
is now:

* add `zai-org/GLM-5.3-Flash` (1), `deepseek-ai/DeepSeek-V4-Flash-0731` (2),
  `Qwen/Qwen3.5-9B` (4) and `MiniMaxAI/MiniMax-M2.7` (6);
* remove `zai-org/GLM-5.3`, `moonshotai/Kimi-K3`, `Qwen/Qwen3.6-Plus` and
  `openai/gpt-oss-120b`.

The other lists' diffs are unchanged.

**Deviation 3 (2026-10-12): a small dense model in the utility six.**
`Qwen/Qwen3.5-9B` is a 9B dense model. 0009-MADR's Confirmation says the
utility six never holds a dense model of 14B or less, and Phase 8's criteria
test would fail on it. But no §3 item enforced it. The owner chose §3 item
11, under the utility profile only: 0009-MADR's amendment "no small dense
model in the utility six (§3 item 11)". Phase 7b builds it, and step 1 is
then run again.

The records of Deviation 3 and 0009-MADR's item 11 were committed by the
owner as `e475f08`.

### Phase 7b: 0009-MADR §3 item 11 (2026-10-12)

**Change** (`llmprovider/catalog/model_ranking.go`):

* `denseSizeRE` reads a size such as `9b` or `14B` standing alone in an id,
  and `moeActiveRE` a mixture of experts's active size, such as `-a3b`.
* `smallDense(id)` is true for a size of `smallDenseMax` (14) or less, where
  the id names no active size.
* `utilityExcluded` returns false under `ProfileCapable`. Otherwise it
  excludes Kilo's `kilo-auto/*` (item 9), or any `smallDense` id (item 11).
  `rankRecommended` already applies it to the ranking and the fill.
* The criteria test keeps its own pattern, so the rule and its check share
  no code.

**Test** (`llmprovider/catalog/small_dense_item11_test.go`),
`TestRankRecommended_SmallDenseUtility`, on Together and Kilo:

* under the utility profile, `a/model-9b`, `b/model-14B` and the fill's
  `f/fill-7b` are left out;
* `c/model-15b`, `d/moe-30b-a3b`, `h/moe-7b-a1b`, `e/model-flash` and
  `g/fill-ok` stay;
* under the capable profile, `a/model-9b` is ranked.

**Red first:** under the utility profile, `ranked = [a/model-9b b/model-14B
c/model-15b d/moe-30b-a3b e/model-flash f/fill-7b]`, against `want
[c/model-15b d/moe-30b-a3b e/model-flash g/fill-ok]`.

**Green:** `go test ./llmprovider/catalog/`, every test, the 2026-09-26
golden among them, unchanged.

**Plants,** through `tools/plant.py`:

| Plant | FAIL line |
| :--- | :--- |
| the threshold as `< 14` | `ranked = [b/model-14B c/model-15b d/moe-30b-a3b e/model-flash g/fill-ok]` |
| the mixture-of-experts exemption dropped | `ranked = [c/model-15b d/moe-30b-a3b e/model-flash g/fill-ok]`: `h/moe-7b-a1b` lost |
| item 11 under `ProfileCapable` too | `together capable: [e/model-flash c/model-15b d/moe-30b-a3b g/fill-ok], want a/model-9b ranked` |
| item 11 in the ranking only, not the fill | `ranked = [c/model-15b d/moe-30b-a3b e/model-flash f/fill-7b g/fill-ok]` |

**A plant not caught at first.** On the first run, "the mixture-of-experts
exemption dropped" passed. The test's only mixture of experts,
`d/moe-30b-a3b`, is 30B in total, so item 11 never touched it. The
exemption matters only for one of 14B or less in total, which the criteria
test also exempts. The test gained `h/moe-7b-a1b`, which must stay. On the
second run, all four plants were caught.

**Checks:** `make pre-add-check` on the two Go files, clean (gofmt,
golangci-lint, go vet, go test, govulncheck).

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 8.4.

Committed by the owner as `14f000d`.

**Step 1 again, after Phase 7b** (not billed): 14 proposals. Item 11 held: `Qwen/Qwen3.5-9B` left
the six. Together's diff:

* add `zai-org/GLM-5.3-Flash` (1), `deepseek-ai/DeepSeek-V4-Flash-0731` (2)
  and `MiniMaxAI/MiniMax-M2.7` (5);
* remove `zai-org/GLM-5.3`, `moonshotai/Kimi-K3` and `openai/gpt-oss-120b`.

The other lists' diffs were unchanged, and every addition there had
answered. The measurement test's Together list became these three, and its
count 7.

**The new id measured,** 1 billed request: together
`MiniMaxAI/MiniMax-M2.7`, HTTP 400 `model_not_available`, "Unable to access
non-serverless model".

**Deviation 4 (2026-10-12): an unservable id that models.dev lists.**
models.dev's `togetherai` section lists `MiniMaxAI/MiniMax-M2.7` as
`active`. Together's listing entry has the shape of
`MiniMaxAI/MiniMax-M3`'s, which is served. No field tells the two apart. The
owner chose a list in the manner of §3 item 8: 0009-MADR's amendment
"Together's ids measured not to serve (§3 item 12)", which also corrects
item 10's premise. Phase 7c builds it.

Phase 7 used 11 billed requests so far.

The records of Deviation 4 and item 12 were committed by the owner as
`769e2b9`.

### Phase 7c: 0009-MADR §3 item 12 (2026-10-12)

**Change** (`llmprovider/catalog/model_ranking.go`):

* `togetherUnserved`, beside `opencodeGoRegionGated`, holds
  `MiniMaxAI/MiniMax-M2.7`, with its date and error.
* `excludedAtRequest` leaves its ids out on Together. It already holds for
  the ranking, the fill and Go's fallback curation.

**Test** (`llmprovider/catalog/together_item12_test.go`),
`TestTogetherMeasuredUnserved`:

* a Together listing and document covering `MiniMaxAI/MiniMax-M2.7`, priced
  cheapest so that only item 12 can keep it out, and two other models;
* under both profiles, the six does not hold it, and `Usable` does;
* the same id, on Hugging Face, is ranked.

**Red first:** `profile 0: Recommended [MiniMaxAI/MiniMax-M2.7 t/covered-a
t/covered-b] holds MiniMaxAI/MiniMax-M2.7, which Together does not serve`,
and the same under profile 1.

**Green:** `go test ./llmprovider/catalog/`, every test.

**Plants,** through `tools/plant.py`:

| Plant | FAIL line |
| :--- | :--- |
| the list applied on every provider | `huggingface: [x/other], want MiniMaxAI/MiniMax-M2.7 ranked; item 12 is Together's alone` |
| `excludedAtRequest` without the Together list | `profile 0: Recommended [MiniMaxAI/MiniMax-M2.7 t/covered-a t/covered-b] holds MiniMaxAI/MiniMax-M2.7`, and the same under profile 1 |

**The command, after item 12** (not billed): 12 proposals. Together's diff
is now:

* add `zai-org/GLM-5.3-Flash` (1) and `deepseek-ai/DeepSeek-V4-Flash-0731`
  (2), both measured at HTTP 200;
* remove `zai-org/GLM-5.3` and `moonshotai/Kimi-K3`.

`MiniMaxAI/MiniMax-M3`, `Qwen/Qwen3.6-Plus` and `openai/gpt-oss-120b` stay.
No new id needs measuring. The measurement test's Together list became these
two, and its count 6.

**Checks:** `make pre-add-check` on the two catalog files and the
measurement test, clean (gofmt, golangci-lint, go vet, go test,
govulncheck).

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 9.9.

Phase 7c was committed by the owner as `930b74d`.

**Phase 7, step 4: the snapshot.** With every addition measured and the picks approved (Q15),
the command ran again (listings and models.dev; not billed):

```text
go run ./llmprovider/internal/cmd/catalogs -snapshot llmprovider/catalog/testdata/ranking-2026-10-12
```

* **Exit 0.** Its 12 proposals were identical to the run before it.
* **The snapshot,** `llmprovider/catalog/testdata/ranking-2026-10-12/`:

  | File | Bytes | SHA-256 |
  | :--- | :--- | :--- |
  | `zen.json` | 6,967 | `700a2a7613610dcf3b83bde13b1415ed4fc48c3e9505c40502dcf00ac9a521bf` |
  | `go.json` | 3,181 | `bcdb4c3ea52c6a8563e8ee2473a501912abe927aded6a09b14cfbd1fcec79d61` |
  | `kilo.json` | 486,974 | `133ea2d9be2ba98f1783431756b0b0b974ff0647772cc67e335d883cab44d340` |
  | `hf.json` | 83,364 | `fa23012ee0d5a838ac657664f9068b81c366a672598bc873691c630853119c37` |
  | `together.json` | 442,420 | `3c63e508a0a1f6cc2e0eb48d45ca7f976671c88b355e6211087a2b66a16ce578` |
  | `metadata.json` | 509,488 | `eda4b2e7542f376529851240835355012ba5b6a4498b44782b718062def2b417` |

  `snapshot.json` gives `recorded_at` `2026-10-12T23:54:43Z`, and those
  hashes. A check recomputed each one: no mismatch.
* **The identifier scan,** `tools/denyscan.py` over the seven files: `0
  hit(s) in 7 file(s)`.
* **Replayed offline,** a scratch program served each file from a local
  server and ran `catalog.List` under the utility profile, 31 seconds after
  the recording. Each ranked list's six gave the command's diff exactly:

  | List | The snapshot's utility six, in rank order |
  | :--- | :--- |
  | OpenCode Zen | `claude-haiku-5-5`, `deepseek-v4.1-flash`, `qwen3.8-flash`, `glm-5.3-flash`, `deepseek-v4-flash`, `gemini-3.5-flash-lite` |
  | OpenCode Go | `claude-haiku-5-5`, `mimo-v2.6-flash`, `qwen3.8-flash`, `glm-5.3-flash`, `gpt-6-luna`, `space-bunny` |
  | Kilo | `deepseek/deepseek-v4.1-flash`, `z-ai/glm-5.3-flash`, `google/gemini-3.8-flash`, `google/gemini-3.6-flash`, `meta/muse-spark-1.2`, `openai/gpt-6.1-sol` |
  | Hugging Face | `deepseek-ai/DeepSeek-V4-Flash-0731`, `zai-org/GLM-5.3-Flash`, `deepseek-ai/DeepSeek-V4.1-Flash`, `thinkingmachines/Inkling-Small`, `stepfun-ai/Step-3.7-Flash`, `stepfun-ai/Step-3.5-Flash` (unchanged) |
  | Together | `zai-org/GLM-5.3-Flash`, `deepseek-ai/DeepSeek-V4-Flash-0731`, `deepseek-ai/DeepSeek-V4.1-Flash`, `MiniMaxAI/MiniMax-M3`, `Qwen/Qwen3.6-Plus`, `openai/gpt-oss-120b` |

  Phase 8's golden pins the clock to `recorded_at`. This replay could not,
  from outside the package.

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 5.1, with the snapshot in the tree.

**Phase 7 is complete.** It used 11 billed requests: 9, then 1 to re-measure
Go's Claude model, then 1 for `MiniMaxAI/MiniMax-M2.7`.

Phase 7's snapshot was committed by the owner as `e2a0f1e`.

### Phase 8: the lists, and the golden (2026-10-12)

**Research first.** The golden needs each list's capable six as well as its
utility six, with the clock pinned. A throwaway probe test, run in a scratch
clone and not in the tree, replayed the snapshot pinned to `recorded_at`. Its
utility sixes equal the unpinned replay of Phase 7.

**Tests** (`llmprovider/catalog/snapshot_20261012_test.go`):

* `replaySnapshot(t, dir, provider, profile)` serves a snapshot's listing and
  `metadata.json` from local servers. It pins the ranking clock to
  `snapshot.json`'s `recorded_at`. `latestSnapshot` is the last
  `testdata/ranking-*` that holds a `snapshot.json`.
* `TestListModelCatalog_Snapshot20261012` requires each ranked list's utility
  and capable sixes, and its usable count: Zen 79, Go 34, Kilo 285, Hugging
  Face 119, Together 172.
* `TestStatic_EqualsLatestSnapshot`: each ranked static list equals the
  latest snapshot's utility six, in order.
* `TestReplaySnapshot_PinsTheClock` (added; see below).
* `TestStaticOpenCatalogs_UtilityCriteria` gains `staticTogether`.

**Red first:** `TestStatic_EqualsLatestSnapshot` failed for `opencode-zen`,
`opencode-go`, `kilo` and `together` (`static list … want the utility six of
testdata/ranking-2026-10-12`), and passed for `huggingface`, whose list does
not change. The golden and the criteria test passed before the lists
changed, as they must: the ranker had not changed.

**Change** (`llmprovider/catalog/models_catalog.go`): the four lists become
the snapshot's utility sixes, in its order, with the routes from api.json's
npm. Each comment, Hugging Face's included, names the snapshot and the test
that holds the list to it. `modelClaudeHaiku55` names `claude-haiku-5-5`,
which the file now uses four times (`goconst`), as `modelGPT6Luna` and
`modelGemini35Lite` are named.

**Tests that named the old lists,** each changed where it asserts the list
itself:

* `model_label_test.go`'s `unlabelledStaticIDs`: the gateway lists' ids,
  excused as `fallbackSix`, follow the new lists.
* `providers/together/together_test.go`'s `TestTogether_Listing`: "the
  static order leads". `openai/gpt-oss-120b` is still static and now leads.
  `zai-org/GLM-5.3`, no longer static, follows in the listing's order. The
  fixture is unchanged, and the test's comment says why.
* **One G-wire golden,** `providers/together/testdata/wire/together/listing.json`,
  regenerated alone (`-update -run TestWireGoldens/together/listing`), for the
  same reason. Its diff is the two result lines swapped:

  ```text
  -    "zai-org/GLM-5.3",
  -    "openai/gpt-oss-120b"
  +    "openai/gpt-oss-120b",
  +    "zai-org/GLM-5.3"
  ```

  No other golden changed.

**Green:** `go test ./...`, all 29 packages.

**A plant the plan named could not fail.** "The golden replay not pinning
the clock" would not have changed today's sixes: the replay ran within
hours of the recording, so no model's age moved enough. So
`TestReplaySnapshot_PinsTheClock` checks the clock directly. During a
replay it must read `2026-10-12T23:54:43Z`.

**Plants,** through `tools/plant.py`:

| Plant | FAIL line |
| :--- | :--- |
| one of `staticOpencodeZen`'s ids swapped for another usable one (`gemini-3.8-flash` for `gemini-3.5-flash-lite`) | `TestStatic_EqualsLatestSnapshot`: `opencode-zen: static list … want the utility six of testdata/ranking-2026-10-12` |
| the replay not pinning the clock | `TestReplaySnapshot_PinsTheClock`: `the ranking clock is` the host's local time `during the replay, want recorded_at 2026-10-12 23:54:43 +0000 UTC` |
| a free id planted in `staticTogether` | `TestStaticOpenCatalogs_UtilityCriteria`: `staticTogether: "openai/gpt-oss-120b:free" is a free model`; and `TestStatic_EqualsLatestSnapshot` |

The third plant shows that Together is in the criteria test. The plan's
form, "left out of the criteria test's table, with a planted free id", was
run as the free id alone. The table already holds Together, so the plant
must be caught.

**The live guard,** `TestLive_StaticCatalogsCurrent` (not billed), passed for
all nine providers.

**Checks:** the first `make pre-add-check` failed on `goconst`:
``string `claude-haiku-5-5` has 4 occurrences, make it a constant``. It
passed once `modelClaudeHaiku55` named it, on all five Go files (gofmt,
golangci-lint, go vet, go test, govulncheck).

**Gate:** `tools/gate.py`, 23 of 23, started at a load of 8.3, on the tree with `modelClaudeHaiku55`.
