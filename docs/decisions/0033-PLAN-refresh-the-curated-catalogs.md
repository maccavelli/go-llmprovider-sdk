---
status: complete
date: 2026-10-11
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

### Out of scope

* The public metadata (D1 to D3): the other plan.
* Forced tool choice on the newest Claude models: its own record
  (0033-MADR D6). That record adds the models it fixes to `staticClaude`.
  Until then, Phase 1 drops any Claude id that refuses forced choice.
* The maintainer tool that proposes the next lists (D4): the other plan,
  `v1.8.0`.
* The gateway lists (Zen, Go, Hugging Face, Kilo, Together). All 30 ids are
  current. Only the Go comment is re-checked.
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

## Verification

* **V1. Red, then green,** with both lines and every plant's FAIL line in
  the Execution Record.
* **V2. The gate** clean at each phase. The listing goldens change only as
  recorded.
* **V3. Measurement:** Phase 1's table recorded before Phase 2 uses it.
* **V4. API:** apidiff `v1.7.0` to `v1.7.1` empty.
* **V5. Live:** the guard and the live suites on the release commit.

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
