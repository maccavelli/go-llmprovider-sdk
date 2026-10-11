---
status: proposed
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

   Listings are not billed.
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

Not started.
