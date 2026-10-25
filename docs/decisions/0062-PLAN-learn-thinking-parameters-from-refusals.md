---
status: in-progress
date: 2026-10-24
associated-madr: "0062-MADR-learn-thinking-parameters-from-refusals.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Thinking Parameters Decided From Facts and Learned Refusals

Associated MADR: [0062-MADR-learn-thinking-parameters-from-refusals.md](0062-MADR-learn-thinking-parameters-from-refusals.md)

## Goal

The MADR's D1 to D11, under the owner's answers to Q1 to Q11 (2026-10-24
and 2026-10-25; Q5 to Q7 each (a); Q8 (b), Q9 (c), Q10 (a), Q11 (a)).
Concretely:

* each wire's refusals are measured before any matcher depends on them;
* every provider and route decides its thinking parameters from the memo,
  the facts and the request, in that order, and resends on a recognised
  refusal within Q5's bound of 7 sends;
* each name rule is removed or kept as D8's criterion and Phase 1's
  evidence decide;
* the exports of D9 and no other;
* every new test is seen to fail first;
* 0061-PLAN's Phase 3, committed as `dd41faf`, is reworked in this PLAN's
  Phase 2.

## Scope

### In scope

* `llmprovider`: `warning.go`, a new `refusal.go`, the options and
  settings.
* `llmprovider/internal/wire`:
  * `effort.go` (0061-PLAN Phase 3's, `dd41faf`) and `retry.go`;
  * a new `learn.go`;
  * a matcher in each of `messages`, `responses`, `chatcompletions` and
    `generatecontent`.
* `llmprovider/internal/cacheref`: the memo's place in a `catalog.Cache`.
* Every provider package under `llmprovider/providers/`. Gemini's
  Interactions matcher lives in `providers/gemini`, beside its decoder.
* `llmprovider/internal/wirecase` and the G-wire goldens that change.
* Fixtures: `llmprovider/internal/wire/testdata/refusals-0062/`.
* Docs: `CHANGELOG.md`, `docs/architecture.md`,
  `docs/guides/api-standards.md` (R56), and each changed package's doc.
* Records:
  * this pair;
  * `docs/reports/0062-REPORT-thinking-refusals.md`;
  * 0061's amendments;
  * the index rows.

### Out of scope

* Any export not in the MADR's D9 (a stop condition).
* Forced tool choice (0034-MADR), the output limit, the cache.
* `wizard` (Q6 (a)).
* Detecting an accepted-and-ignored value (MADR F4).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase, once Q5 to
  Q7 are answered. A phase starts after the owner has committed the last.
* **Red first.** Every new test is seen to fail on a scratch clone of the
  tree as it stood before the phase, with any new name declared in a shim
  file so that it
  builds; or on a planted break by `tools/plant.py`. Never in the tree. The
  phase's record holds each failure line.
* **The drift check.** After a phase's code and before its new tests,
  `go test -race -count=1 -shuffle=on ./...` passes, with every existing
  test unchanged except those the phase lists in its record before it
  changes them (Phases 2 and 3, step 1).
* **No key, token, account id or model output in a test, a log, a fixture
  or a record.** Phase 1's fixtures are redacted as AGENTS.md's "Recorded
  fixtures" says, and `make fixtures-check` passes on them.
* **The checks, before staging,** each exit status captured before any
  filter:
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `python3 -B tools/gate.py`;
  * `make api-check` and `tools/apidiff.py v1.22.2`, with the additions
    listed in the record;
  * `tools/claims.py` on the MADR;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.
* **Stage, then the owner commits** with `git commit --no-edit`. A push and a
  tag happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * an existing test would change that the phase did not list;
  * an export not in D9 would be added, or one of D9 would change;
  * a matcher would match a recorded failure of another cause;
  * Phase 1's evidence contradicts a decision rather than choosing between
    D8's outcomes.

  Each stops the phase. The deviation and its resolutions go to the owner,
  and the record is amended before work continues.

## Implementation Steps

### Phase 0: records

1. This pair, `status: proposed`; their rows in `docs/README.md` by
   `tools/records.py add`.
2. **0061-MADR** gains the amendment "D3 reworked by 0062". **0061-PLAN**
   gains, in its record, "Committed as `dd41faf` (2026-10-24)", and the note
   that Phases 4 to 8 follow 0062-PLAN.
3. **The checks:** `make records-check`, `tools/claims.py` on both MADRs,
   `tools/recordlint.py --changed`, `tools/linkcheck.py`.
4. **The commit:** the 0061 and 0062 records and `docs/README.md`, and
   nothing else.

### Phase 1: measure each wire's refusals (MADR F3, F4; D4, D8)

1. **The recorder,** ~~`llmprovider/providers/refusals_0062_live_test.go`
   (build tag `live_gateways`, package `providers`)~~
   `llmprovider/live_0062_refusals_measure_test.go` (build tag
   `live_gateways`, package `llmprovider_test`), with its case table,
   rewriter and writer in the untagged
   `llmprovider/refusals_0062_record_test.go` (the deviation of 2026-10-25):
   * It is switched on by `LLMPROVIDER_LIVE_0062=1`. A provider's cases run
     when its key variable (`ProviderEnvVars()`) is set; without it, each of
     its cases is recorded as `skipped`. Ollama is not in this phase (Q9
     (c)); Phase 5 records it.
   * Each case is a subtest named by its case id, so that one case reruns
     with `-run 'Live_Refusals_0062/<case>'`.
   * Each case builds the provider with `New` and
     `WithHTTPClient(&http.Client{Transport: rewriter})`, and sends
     `Generate` with `Reasoning{Effort: EffortHigh}` and the input "Reply
     with OK.", and no `MaxTokens`: each provider sends its own default
     output limit, as it does today (Q11 (a)).
   * The rewriter sets the case's fields in the outgoing JSON body, by
     path, and keeps the response's status and raw body. So the request
     goes out with the provider's own headers and auth, and the error
     reaches the `*APIError` exactly as a matcher will see it.
   * For each case it writes one JSON file to the directory
     `LLMPROVIDER_LIVE_0062_OUT` names, outside the tree:
     * the case id, provider, route, model and the fields sent;
     * the status;
     * the `*APIError`'s `Code` and `Message`;
     * the raw error body, redacted by `internal/redact` and cut to 4 KiB;
     * whether the reply carries usage, and the reasoning tokens of an
       accepted reply;
     * the outcome: `refused`, `accepted`, `other`, `unreachable` or
       `skipped`.
2. **The cases,** each wire's model being the static catalog's or the G-wire
   case's for that route. A case whose model is unreachable is recorded as
   `unreachable`.

   | Wire | Provider, route, model | Fields sent |
   | :--- | :--- | :--- |
   | Messages | Claude, `claude-sonnet-5` | `thinking` `{type: enabled, budget_tokens: 4096}`; adaptive with `output_config.effort` `"max"`; adaptive with effort `"bogus"` |
   | Messages | Claude, `claude-sonnet-4-5-20250929` | adaptive with effort `"high"` |
   | Messages | OpenCode Go's messages route, `qwen3.8-flash` | adaptive, no effort |
   | Messages | OpenCode Go's messages route, `minimax-m3` | adaptive with effort `"high"` |
   | Responses | OpenAI, `gpt-6-luna` | `reasoning.effort` `"max"`; `"bogus"` |
   | Responses | OpenAI, `gpt-4.1-mini` | `reasoning.effort` `"low"` |
   | Responses | Grok, `grok-4.5` | `"xhigh"`; `"max"` |
   | Responses | Grok, `grok-4.3` | `"max"` |
   | Responses | OpenCode Zen's responses route, its G-wire model | `"max"`; `"bogus"` |
   | Interactions | Gemini, `gemini-3.8-flash` | `thinking_level` `"max"`; `"xhigh"`; `"bogus"` |
   | Interactions | Gemini, `gemini-2.5-flash-lite` | `thinking_level` `"medium"`; `"low"` |
   | generateContent | OpenCode Zen's google route, its G-wire model | `thinkingLevel` `"max"`; `thinkingBudget` 4096 with no level |
   | generateContent | OpenCode Zen's google route, `gemini-2.5-flash` | `thinkingLevel` `"low"` |
   | Chat Completions | Hugging Face, `deepseek-ai/DeepSeek-V4.1-Flash` | `reasoning_effort` `"max"`; `"bogus"` |
   | Chat Completions | Together, `openai/gpt-oss-120b` | `reasoning_effort` `"max"`; `"bogus"` |
   | Chat Completions | Kilo, `openai/gpt-6-luna` | `reasoning.effort` `"max"`; `"bogus"` |
   | Chat Completions | OpenCode Zen's chat route, `glm-5.3` | `reasoning_effort` `"max"`, with no metadata |
   | every wire | one model of each row above's provider and route (Q10 (a)) | other causes: an empty input; a tool whose schema is `{"type":"string"}` |

3. **Its offline test,** ~~`llmprovider/providers/refusals_0062_test.go`~~
   `llmprovider/refusals_0062_test.go` (no
   tag), runs the rewriter against a `wiretest` server and checks that:
   * the fields arrive as set;
   * a canned 400 is recorded with its status, code, message and redacted
     body;
   * a planted secret in the body is redacted.

   Red first: without the rewriter's set, the fields assertion fails.
4. **The owner runs it,** with the key of every provider in the table set
   (Q8 (b)):

   ```bash
   LLMPROVIDER_LIVE_0062=1 LLMPROVIDER_LIVE_0062_OUT=<dir> go test -tags live_gateways -count=1 -run Live_Refusals_0062 ./llmprovider/
   ```

5. **The agent then:**
   * copies each file to
     `llmprovider/internal/wire/testdata/refusals-0062/<wire>/<case>.json`;
   * runs `make fixtures-check` and redacts until it passes;
   * writes `docs/reports/0062-REPORT-thinking-refusals.md`: a table per
     wire of the cases and outcomes, with what each refusal names
     (parameter, value, accepted values) and whether any 400 carries usage;
   * amends the MADR with "Phase 1's evidence": D8's outcome per rule,
     D4's matcher per wire (status, code, message pattern, accepted-values
     pattern), and F3 and F4 restated as measured.
   * **The gate to Phase 2 (Q8 (b)):** no case is `skipped`, and none was
     answered 401, refused for its credential (amended 2026-10-25). If one
     is, the record names it and its key variable; the owner sets the key
     and the case reruns by its subtest name; the agent then repeats this
     step for it. Phase 2 does not start while any case is `skipped` or 401.
6. **The commit:** the recorder, its offline test,
   the fixtures, the report, the amendment and the index row.

### Phase 2: the learning core, effort on every provider, and 0061's Phase 3 reworked (D2 to D6, D8, D10)

1. **Before any change,** the record lists every existing test that asserts
   a rule D8 removes from the effort path (R5, R6, Gemini's `xhigh`,
   0061's rows, and the chat-route gate (Q7 (a))), found by
   `git grep` of the rule's function and its asserted values. Only those
   change, and each is listed with its change.
2. **`llmprovider`:**
   * `warning.go` gains `WarningReasoningLearned` and `Warning.Refused`;
   * the new `refusal.go` gains `ReasoningRefusal`, `ThinkingShape`,
     `ThinkingAdaptive`, `ThinkingBudget`, `WithReasoningRefusals` and
     `Settings.ReasoningRefusals`, with `New`'s checks as D6 says.
3. **`internal/wire`:**
   * `effort.go`: `Ladder(want llmprovider.Effort, listed []llmprovider.Effort, known bool, refused []llmprovider.Effort) []llmprovider.Effort`,
     D3's rungs, in place of 0061's `ResolveMax` and `MaxEffort`;
   * `learn.go`: the memo (D5), held by `internal/cacheref` beside
     0034's `Refusals`;
   * `retry.go`: `wire.Send` and `StreamEventsRetry` consult the check
     until it declines or Q5's bound of 7 sends is met, a forced-choice refusal
     counting against the same bound;
   * one matcher per wire, as the MADR's Phase 1 amendment states it.
4. **Each provider:**
   * its `Generate` and `Stream` go through `wire.Send` and
     `StreamEventsRetry` with a check that consults its wire's matcher
     (and, where it has one, 0034's);
   * its effort comes from `Ladder` with the memo, `Settings.ReasoningRefusals`
     and the facts;
   * its response carries `WarningEffortDegraded` when the effort sent is
     not the one asked, and `WarningReasoningLearned` for each refusal
     learned.

   0061's committed rows (Grok's clamp for `max`, Gemini's and the google
   route's `high`) are replaced. R5, R6, Gemini's `xhigh` and the chat-route
   gate are removed or kept as Phase 1's amendment of D8 says. A kept one
   cites its fixture in a comment.

   **Ollama is unchanged in this phase (Q9 (c)):** it sends as `dd41faf`
   does, with no matcher and no resend, until Phase 5.
5. **The drift check.**
6. **Tests:**
   * `internal/wire`: `TestLadder_Rungs_0062` (every rung, with facts,
     refusals and none); `TestMemo_KeyAndBound_0062` (provider, base URL and
     model; the 257th model drops the first); `TestSend_Bound_0062` (the
     bound met, and a forced-choice refusal counted);
   * each wire package: `TestRefusal_Fixtures_0062`. Every recorded refusal
     of its wire matches, with its parameter, value and accepted values;
     every recorded failure of another cause does not;
   * `llmprovider`: `TestReasoningRefusals_Option_0062` (the settings
     returned; each invalid refusal refused);
   * `llmprovider/providers/learn_0062_test.go`,
     `TestLearn_Effort_0062`: for each provider and route but Ollama, a `wiretest`
     server answers the first send with that wire's recorded refusal and
     accepts the next. The test checks:
     * the request succeeds within the bound;
     * the warnings carry the `Refused` value;
     * a second request sends the learned effort first;
     * `WithReasoningRefusals` makes the first request send it first;
     * `Stream` does the same before its first event.
   * 0061's tests from `dd41faf` (`contract_0061_test.go`, `effort_0061_test.go`,
     `effortmax_0061_test.go`, the `effort-max` goldens) are reworked to D3.
     Each change is listed in the record, and every changed golden is
     diffed against D3's ladder.
7. **Plants,** all on scratch clones:
   * a matcher that never matches (its fixtures test and the learning test
     fail);
   * the memo never read (the second-request check fails);
   * the bound one higher;
   * `WithReasoningRefusals` ignored;
   * the ladder skipping rung 2.
8. **Docs:**
   * each provider's package doc, under Degradations (R12);
   * `CHANGELOG.md`: Added, the D9 exports less `ProbeReasoning`; Changed,
     the effort decided by refusals;
   * `docs/architecture.md`: the `internal/wire` row and the warnings.
9. **The commit:** this phase's paths, 0061-PLAN Phase 3's code among them,
   reworked. 0061-PLAN's record gains the commit as the one that reworks its
   Phase 3.

### Phase 3: the shapes (D2, D3, D8: R1, R2, R4)

1. **Before any change,** the record lists every existing test that asserts
   R1, R2 or R4 (`git grep` of `claudeAdaptiveOnly`,
   `minimaxAdaptiveThinking`, `geminiLegacyRE` and the shapes they assert),
   and every G-wire golden of the `thinking` and `thinking-tool` scenarios
   whose body changes.
2. **`messages.AddThinkingFor` and `generatecontent.ThinkingConfigFor`**
   take the shape from the memo and the facts, else the request's (D2);
   the provider learns a shape refusal and resends with the other shape.
   R1, R2 and R4 are removed or kept as D8's amended outcome says.
3. **The drift check.**
4. **Tests:** `TestLearn_Shape_0062`, for Claude, OpenCode's messages route
   and OpenCode's google route. The adaptive shape refused, then the budget
   accepted, and the reverse; the second request first-time right. Each
   changed golden is diffed and listed.
5. **Plants:** the shape ladder not flipped; a shape refusal not learned.
6. **Docs:** Claude's and OpenCode's package docs; `CHANGELOG.md`.

### Phase 4: the explicit probe (D7; Q6)

1. `llmprovider.ProbeReasoning`, as D7 says.
2. No caller in this repository: Q6 is (a), so `wizard` is unchanged.
3. **Tests:**
   * `TestProbeReasoning_0062`: against a fake provider whose first send is
     refused, it returns that refusal; against one that accepts, it returns
     none;
   * it makes exactly one accepted send.
4. **Plant:** the probe returns nothing.
5. **Docs:** `llmprovider`'s package doc; `CHANGELOG.md`.

### Phase 5: Ollama's refusals, and live confirmation, run by the owner

1. **Ollama's cases (Q9 (c)).** The recorder gains them, switched on by
   `LLMPROVIDER_LIVE_0062_OLLAMA` (the host's base URL) and
   `LLMPROVIDER_LIVE_0062_OLLAMA_MODEL` (a thinking model installed there):
   `reasoning_effort` `"xhigh"`; `"max"`; `"bogus"`; and the other causes of
   Phase 1's last row. The owner runs them; the agent copies, redacts,
   reports (the report gains an Ollama section) and amends the MADR's D8
   for Ollama, as Phase 1's step 5.
2. **If that amendment removes Ollama's `xhigh` mapping:** Ollama moves to
   `wire.Send` and `StreamEventsRetry` with the Chat Completions matcher, or
   with a matcher of its own in `providers/ollama` if its bodies are not the
   wire's (D4), and takes its effort from `Ladder`.
   `TestRefusal_Fixtures_0062` and `TestLearn_Effort_0062` gain Ollama's
   rows, each seen to fail first; the plant is Ollama's matcher never
   matching. If the amendment keeps the mapping, Ollama's package doc cites
   the fixture under Degradations, and nothing else changes.
3. `live_0062_refusals_measure_test.go` (the deviation of 2026-10-25) gains
   `TestLive_Learn_0062`, switched on by
   `LLMPROVIDER_LIVE_0062_CONFIRM=1`. For each removed rule's case, Ollama's
   included when step 2 removed its mapping, it sends through the provider
   with no rewriter and no `MaxTokens` (Q11 (a)), and checks:
   * that the request succeeds;
   * the warning it carries;
   * that a second request is accepted first time.
4. The owner runs it. The record holds each case's result.
5. A case that fails is a stop condition.

### Phase 6: the standard and the close

1. `docs/guides/api-standards.md` gains R56, as D11 words it, in its table
   with its check (the `grep` of Confirmation).
2. `docs/architecture.md`: the order of authority, the memo, the resend.
3. The MADR's status becomes `accepted`, and this PLAN's `complete`, by
   `tools/records.py status`.
4. 0061-PLAN resumes at its Phase 4.

## Verification

* Every phase: the rules' checks, their exit statuses in the record.
* Phase 1: `make fixtures-check` passes on the fixtures; the report covers
  every case of the table, `unreachable` included, and none is `skipped`
  or 401 (Q8 (b); amended 2026-10-25).
* Phase 5: Ollama's cases recorded and reported, and D8's Ollama row
  decided.
* Phases 2 and 3: every test named above passes; every plant is caught;
  `git grep` finds no function D8 removed.
* `tools/apidiff.py v1.22.2` lists 0061's additions and D9's, and nothing
  else.
* Phase 5: every removed rule's case confirmed live.

## Rollout and Rollback

* **Rollout:** in 0061-PLAN Phase 8's release, `v1.23.0`, which this PLAN
  precedes.
* **Rollback:** reverting Phases 2 and 3 restores the name rules; the
  exports stay compatible (R48). A caller's `WithReasoningRefusals` seeds
  only what the provider would have learned, so it can do no harm after a
  rollback.
* **The memo is never on disk,** so a restart clears anything wrongly
  learned. A caller clears its kept refusals by dropping them.

## Execution record

* **Approved 2026-10-24.** The owner answered Q5 to Q7, each (a) as
  recommended, and approved this PLAN whole. Its status became `in-progress`
  by `tools/records.py status`; the MADR stays `proposed` until Phase 6.

* **2026-10-24, before approval: 0061's Phase 3 committed.** While this PLAN
  was proposed, the owner committed 0061-PLAN's Phase 3 as `dd41faf`, the 44
  paths that were to be held, and chose to keep the commit (of: keep it and
  amend the records; revert it and hold again; reset it). The hold procedure
  and the list of held paths are removed; each phase commits its own paths.
  The MADR's amendment "0061's Phase 3 committed" records it.

* **Deviation 2026-10-25, Phase 1 step 1: the recorder's place.** Found
  before any code was written:
  * The live helpers a recorder needs (`LiveCtx`, `LiveEnvKey`,
    `LiveOpencodeKey`, `LiveKiloKey`, `SkipIfTransient`) are exported only
    to `llmprovider`'s own tests (`llmprovider/live_export_test.go:14`), so
    a test in `llmprovider/providers` cannot use them.
  * `scripts/check_tables.py:178` reads live variables only from
    `llmprovider/**/live_*_test.go`, and `llmprovider/live_hygiene_test.go:33`
    checks only files named `live_*_test.go`. A file named
    `refusals_0062_live_test.go` would escape both.

  Resolutions offered: move the recorder to `llmprovider/` as its live
  measurements are (recommended); keep it in `providers/`, renamed, with
  copies of the helpers; or a shared helper package, a refactor of about 40
  live tests needing its own record. **The owner chose the first.** The
  recorder is `llmprovider/live_0062_refusals_measure_test.go`; its case
  table, rewriter and writer are in the untagged
  `llmprovider/refusals_0062_record_test.go`, which the offline test
  `llmprovider/refusals_0062_test.go` also runs, as `live_twins_test.go` runs
  `live_checks_test.go`'s code (0046-MADR D3). Phase 1 step 1 and step 3 and
  Phase 5 step 3 are annotated. No decision changes, and the MADR names no
  path of these.

### Phase 0: records (2026-10-24)

* **Done.** Steps 1 and 2 as written: this pair and its index rows; 0061-MADR's
  amendment "D3 reworked by 0062"; 0061-PLAN's record entry "Committed as
  `dd41faf` (2026-10-24)" (step 2 as amended before approval). The owner's
  Q5 to Q7 answers are written into the MADR's D5, D7, D8, D9, Consequences
  and Owner questions, and into this PLAN's Goal, Scope, Phases 2 and 4.
* **2026-10-25, before the commit: Q8 to Q11.** The owner answered Q8 (b),
  Q9 (c), Q10 (a) and Q11 (a). They are written into the MADR's D5, D8 and
  Owner questions, and into this PLAN's Goal, Phases 1, 2 and 5, and
  Verification: no phase starts on an unrecorded wire; Ollama moves to
  Phase 5; the recorder sets no `MaxTokens`.
* **Checks** (step 3), each exit 0:
  * `make records-check`: "records-check: 130 records, 0 problem(s)";
  * `tools/claims.py`: this MADR 41 of 41 (at `1b46940`); 0061-MADR 58 of 58;
  * `tools/recordlint.py --changed`: "4 record(s); no rule rose";
  * `tools/linkcheck.py`: 0 problems; `tools/denyscan.py --changed`: 0 hits.
* **No new check or test,** so no red-first experiment.
* **Staged** (step 4): `docs/README.md`, the 0061 MADR and PLAN, this MADR
  and this PLAN, and nothing else.

### Phase 1: measure each wire's refusals (2026-10-25)

In progress: steps 1 to 3 are done; step 4, the owner's run, is next. Nothing
is staged: the phase's one commit (step 6) holds the recorder with its
fixtures and report.

* **Step 1, the recorder,** at the paths of the deviation of 2026-10-25:
  * `llmprovider/refusals_0062_record_test.go` (untagged): the case table,
    the rewriting transport, the outcome, the record and its writer, and
    `outsideModule`, which refuses an output directory inside the module.
  * `llmprovider/live_0062_refusals_measure_test.go` (`live_gateways`):
    `TestLive_Refusals_0062`, one subtest per case. A provider without its
    key variable gets a `skipped` record and no send. It logs each case's
    outcome, status and sends; never a message, a body or a key.
  * Each case is sent with `WithModelProbes(false)` and
    `WithoutModelMetadata()`, so no listing or probe is made, and with no
    `MaxTokens`: the default, 8192 (`llmprovider/settings.go:86`), which
    leaves the 4096 budget valid (Q11 (a)).
* **Step 2, the cases:** 51, as the table says, less Ollama (Q9 (c)): 29
  thinking cases and the two other causes on each of the 11 providers and
  routes, each on its route's first model (Q10 (a)). Made precise in the
  building, no decision changed:
  * **A field is measured alone.** Where the provider would send another
    field of the same kind beside it, the case drops that one: the Claude
    budget case and the qwen adaptive case drop `output_config`; the google
    route's level cases drop `thinkingBudget`, and its budget case
    `thinkingLevel`. Each record lists what was set and what was dropped.
  * **The empty input** is set by the rewriter, as the wire's `messages`,
    `input` or `contents` emptied. **The string-schema tool** is sent through
    the request, `Tools`, so each provider encodes it as a caller's would;
    the SDK refuses only a schema that does not marshal
    (`llmprovider/contract.go:269`).
  * **The outcome is the status's**, so that a record says what came back,
    not what it means: 404 `unreachable`; any other 4xx but 401, 403, 408,
    409 and 429 `refused`; anything else, a transport failure included,
    `other`. The report reads each refusal's message for its cause.
  * Each record is `<wire>/<case>.json`, the fixtures' own layout.
* **Step 3, the offline test,** `llmprovider/refusals_0062_test.go`: 8
  tests, each passing: `TestRefusalRecorder_Fields_0062` (the fields set and
  dropped arrive; a 400's status, code, message and body are kept; a planted
  key in the body is redacted), `TestRefusalRecorder_Accepted_0062`,
  `TestRefusalOutcome_0062`, `TestRefusalCases_Table_0062`,
  `TestRefusalPaths_0062`, `TestRefusalRecord_Write_0062`,
  `TestRefusalBody_Bound_0062` and `TestRefusalRecorder_OutsideModule_0062`.
* **Red first,** by `tools/plant.py` on a scratch clone: 17 plants, all
  caught, each by its test's assertion, none by a build failure. The PLAN's
  own, the rewriter's set not applied, failed with
  `refusals_0062_test.go:49: sent thinking map[type:adaptive], output_config
  map[effort:high], model claude-sonnet-5; want adaptive, effort bogus,
  claude-sonnet-5`. The others: the drop not applied; sends not counted;
  `setPath` making no objects; the body not redacted (`body holds the planted
  secret unredacted`); the code and message not kept; an accepted reply's
  usage not kept; the tool not sent; 404 and 429 as refusals; a thinking
  case lost (`28 thinking and 22 other-cause cases of 50; want 29, 22 and
  51`); one other cause per route; a skipped record's outcome; the wire's
  directory left out; the body never cut; the cut inside a rune; and
  `outsideModule` always nil.
  * The bound's test was too weak as first written: 4096 bytes of a two-byte
    rune put the bound on a rune's start, so the cut inside a rune passed.
    One byte before the runes moves each start to an odd offset; the plant
    then failed (`bound body: 4096 bytes, valid UTF-8 false`).
  * A plant renaming the recorder's `LLMPROVIDER_LIVE_0062_OUT` to a name no
    record holds failed `scripts/check_tables.py` (exit 1), so the check
    reads the recorder, as the deviation intended.
* **Checks,** each exit 0: `make pre-add-check` on the three files,
  "3 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)";
  `go vet` and `go vet -tags live_gateways` on `./llmprovider/`; the live
  test without its switch skips; `make tables-check`, "38 packages, 0
  problem(s)"; `python3 -B tools/gate.py`, "gate: 28 of 28 passed".
* **Deviation 2026-10-25, step 4: the agent ran it.** The owner asked the
  agent to run the recorder with the keys already in the host's environment,
  so that no key's value entered the session. A script printed only whether
  each of the eight variables was set (all were); the command named only
  `LLMPROVIDER_LIVE_0062` and `LLMPROVIDER_LIVE_0062_OUT`, a scratch
  directory outside the module; the log holds each case's outcome, status
  and sends. No decision changes.
* **Step 4, the first run (2026-10-25):** 51 cases in 31 s, one send each,
  none `skipped`. 35 cases on Claude, OpenAI, Grok, Gemini, Hugging Face,
  Together and Kilo recorded the wire's answer. **15 of OpenCode's 16 were
  answered 401 `api_key_expired`** ("This API key has expired"), before the
  gateway read the request, so they record the credential, not the wire; the
  16th, Zen's google route on `gemini-2.5-flash`, was 400 "Upstream request
  failed: Model is unavailable."
  * Q8 (b)'s gate names only `skipped`; a case refused for its credential
    measures the wire no more than a skipped one. Phase 1 stops here for the
    owner's resolution (the Rules' stop conditions).
  * Seen in the 35, for the report: no refusal carries usage on any wire;
    Together accepts a bogus `reasoning_effort` (38 reasoning tokens), as F4
    supposed; Grok refuses `max` with "Invalid reasoning effort." and no
    accepted values; and Gemini's Interactions answers an empty `input` with
    404 "Model … not found", so the status rule recorded it `unreachable`.
* **Resolution (owner, 2026-10-25):** the owner renews `OPENCODE_API_KEY`
  in the host's environment; OpenCode's 16 cases rerun alone, by
  `-run 'Live_Refusals_0062/opencode-'`; step 5 then reads all 51 records
  once. Step 5's gate and Verification now count a 401 as `skipped` is
  counted. Offered and not chosen: step 5 on the 35 now and OpenCode in a
  second pass; or OpenCode dropped, against Q8 (b).
* **Step 4, OpenCode's rerun (2026-10-25):** on the renewed key, the 16
  cases by `-run 'Live_Refusals_0062/opencode-'`, one send each; a script
  printed only that the variable was set. None is 401. All 51 records: 19
  accepted (200), 31 refused (400), 1 unreachable (404); none `skipped` or
  401, so the gate to Phase 2 is met.
* **Step 5:**
  * The 51 records were copied byte for byte to
    `llmprovider/internal/wire/testdata/refusals-0062/<wire>/<case>.json`.
    `make fixtures-check` passed on them as copied; every error body was
    then read whole, and Claude's five request ids and Together's one were
    redacted (`req_REDACTED`, `REDACTED`): ids of requests, not accounts, of
    no use to a matcher. `make fixtures-check` passes, "326 file(s), 134
    hit(s), 0 problem(s)". A plant of an e-mail address in one record's
    message failed it (exit 1), so it reads these files.
  * `docs/reports/0062-REPORT-thinking-refusals.md`, its tables generated
    from the fixtures, its findings E1 to E9; its row by `tools/records.py
    add`.
  * The MADR's amendment "Phase 1's evidence": F3, F4 and F6 as measured,
    D4's matchers, D8's outcomes.
* **Stop 2026-10-25, step 5: two outcomes D8 did not name.** The stop
  condition "Phase 1's evidence contradicts a decision rather than choosing
  between D8's outcomes" was met twice:
  * the value R1's non-Claude side, R2 and 0061's google-route row avoid was
    accepted, where D8 named only a recognisable refusal or none;
  * flash-lite refused `low` with a budget's message, not the level's.

  The owner chose, as the MADR's Q12 (a) and Q13 (a): such a rule is
  removed; the budget message on Interactions is the level's refusal, and R5
  is removed. Offered and not chosen: keep the rules as wire mappings, or
  decide in Phase 5; keep R5 for `low`, or whole.
* **Checks before staging,** each exit 0: `tools/recordlint.py --changed`,
  no rule rose (after four MD038 code spans in the MADR's matcher table were
  reworded and the report's three URLs bracketed); `make records-check`, 133
  records; `tools/claims.py` on the MADR, 41 of 41; `tools/linkcheck.py`;
  `tools/denyscan.py --changed`, 0 hits; `make fixtures-check`;
  `python3 -B tools/gate.py`, "gate: 28 of 28 passed".
* **Step 6, staged for the owner's commit:** the recorder's three files,
  the 51 fixtures, the report, this PLAN, the MADR and `docs/README.md`.
