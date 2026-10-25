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

### Phase 2: the learning core and effort (2026-10-25)

Done; staged for the owner's commit with step 9's paths.

* **Step 1, before any change:** the tests that assert a rule D8 removes from
  the effort path, found by searching every `_test.go` for each rule's
  function and the values it asserts (`effortMenu`, `clampReasoningEffort`,
  `thinkingLevel`, `thinkingLevelFor`, `gemini-2.5-flash-lite`, `xhigh`,
  `MaxEffort`, `ResolveMax`, `maxOnRoute`, `reasoning_effort`, the
  `effort-max` goldens). Only these change:
  * `providers/gemini/interactions_test.go`, `TestGeminiInteractions_Thinking`:
    `xhigh` is sent as `xhigh`, not `high` (Gemini's mapping removed); and
    `gemini-2.5-flash-lite` at `low` sends `low`, not nothing (R5 removed).
  * `providers/gemini/level_0041_test.go`: `TestThinkingLevel_Facts_0041`
    restated for the level D3's ladder gives; `TestAgreement0041_GeminiLevel`
    removed with R5, the rule whose agreement with the facts it checked.
  * `providers/grok/reasoning_test.go` (`TestGrokEffortMenus_ClampToCLIMenu`,
    `TestGrokEffortMenus_KeepMenuValues`, `TestGrokClampReasoningEffort`) and
    `providers/grok/reasoning_0033_test.go`: removed with the clamp (R6).
    The CLI's default, `high` for a request with no effort, which is not a
    model rule, is kept, and asserted in the new tests.
  * `providers/grok/efforts_0041_test.go`: `TestGrokEfforts_Facts_0041` keeps
    its efforts (the listing's `xhigh`; `high` from models.dev, which lists no
    `xhigh`; `xhigh` as asked with no facts) and gains the
    `WarningEffortDegraded` the models.dev case now carries;
    `TestAgreement0041_GrokEfforts` removed with R6.
  * `providers/grok/evidence_0041_test.go`, `TestEvidence0041_GrokMenus`:
    removed with `effortMenu` (R6).
  * `providers/opencode/opencode_test.go`: `TestOpencode_ChatReasoningEffort`,
    whose cases with the metadata disabled, or not listing the effort, send
    the ladder's effort instead of none (Q7 (a)); and the chat route's
    assertion in `TestOpencode_Thinking_PerRoute` that no `reasoning_effort`
    is sent without metadata (Q7 (a)).
  * `providers/opencode/wire_shapes_test.go`,
    `TestChatReasoningEffort_FailureBackoff`: with the metadata host down,
    `reasoning_effort` is sent (Q7 (a)); its one-fetch assertion stays.
  * `providers/effortmax_0061_test.go`, `TestEffortMax_Mapping_0061`: the
    rows of Grok, Gemini and OpenCode's google route follow D3's ladder;
    the warning's text keeps 0061's form.
  * The G-wire `effort-max` goldens of `grok`, `grok-stateless`, `gemini` and
    `opencode-zen-google`, by D3's ladder. Each is diffed in this record.

  Unchanged, though they name 0061's functions: `TestResolveMax_Rules_0061`
  and `TestMaxEffort_Warning_0061` (Ollama keeps `MaxEffort` until Phase 5,
  Q9 (c)), `TestEffortMax_Valid_0061`, and `TestEffortMax_ChatRoute_0061`
  (with metadata, the chat route sends what it lists, as before).
* **Stop 2026-10-25, step 4, Grok: R6's no-menu branch was not recorded.**
  `effortMenu` (`grok/reasoning.go:49`) sends no effort to a Grok model
  outside its menu ("may reject the parameter"), and Grok's catalog still
  lists two: `grok-build-0.1` and `grok-4.20-0309-non-reasoning`
  (`catalog/models_catalog.go:160`, `catalog/grok_catalog_test.go:22`).
  Phase 1 recorded only Grok's refusal of `max` on models that take efforts,
  so R6's removal would send an effort no recording covers: an unrecorded
  refusal would fail a request that succeeds today. Found in the same search:
  three tests asserting R6 that step 1's list missed, for they name no rule
  function. None had been changed.
  * Resolutions offered: record first (recommended); keep the no-menu
    branch as a wire mapping, unmeasured; remove R6 whole and return an
    unrecorded refusal. **The owner chose to record first.**
  * Phase 1's recorder gains two cases, `reasoning.effort` `"high"` on
    `grok-build-0.1` and on `grok-4.20-0309-non-reasoning`, run by the agent
    with the host's keys, as before. Their fixtures, the report's rows and
    the MADR's amendment decide the no-menu branch: removed if refused
    recognisably or accepted; kept as a wire mapping otherwise. The
    recorder's change is in Phase 2's commit.
  * **Step 1's list, amended before these tests change:**
    `grok/grok_test.go`'s `TestGrok_GenerateThinking_ReasoningEffortOmitted`
    (`grok-4` sends no reasoning) and
    `TestGrok_GenerateThinking_ReasoningEffortClamped` (`grok-3-mini`'s
    `medium` clamped to `low`), and `grok/request_test.go`'s
    `TestGenerate_Reasoning`, rows "request clamped to the menu" and "a
    model without a menu": each follows R6's outcome. And
    `TestRefusalCases_Table_0062` (Phase 1's): 31 thinking cases and 53 in
    all, for the two added.
  * **Step 1's list, amended again (2026-10-25), before the test changes:**
    `gemini/level_0041_test.go`'s `TestThinkingLevel_Wired_0041`, which the
    search missed for it names no rule function: with `low` and `high`
    listed, `medium` sent no level; by D3's ladder it sends `low`, warned
    (rung 2), as `TestThinkingLevel_Facts_0041`'s restatement does.
  * **Step 1's list, amended a third time (2026-10-25), before they
    change:** the G-wire `effort-max` goldens of `opencode-zen-chat` and
    `opencode-go-chat`, which step 1 missed for it named only the google
    route's: with no metadata, the chat route sends `reasoning_effort`
    `"max"` (Q7 (a)), where it sent none. And `TestOpencode_Thinking_PerRoute`,
    listed, does not change: its chat-route assertion is of a request with no
    reasoning.
  * **Step 1's list, amended a fourth time (2026-10-25), before it changes:**
    `internal/wire/retry_0034_test.go`'s `TestSend_Retry_0034`, its case "the
    resend refused": two refusals and a check that says resend made two sends
    and returned the second refusal, 0034-MADR D3's one resend. Under Q5 (a)
    and step 3, the check is asked after each refusal until it declines or
    `MaxSends` is met, so the third send is answered. The search of step 1
    looked for rule functions, not for `wire.Send`'s own tests.
  * **Step 1's list, amended a fifth time (2026-10-25), before they
    change:** two live tests, which step 1's search did not reach, as CI
    only vets them: `llmprovider/live_grok_test.go`'s
    `TestLive_GrokEffortMenu`, which asserted R6's clamp of `xhigh` to
    `high` on `grok-4.5`, now asserts `xhigh` sent and answered, as Phase 1
    recorded xAI accepting it there; and
    `llmprovider/live_gemini_flashlite_test.go`'s comment, which said `low`
    and `medium` send no level (R5), now says they are refused and stepped
    to `high`. The latter only logs.
* **Deviation 2026-10-25, step 8: R2's row for `internal/cacheref`.**
  `make tables-check` failed: `internal/cacheref` imports `internal/wire`,
  for the memo's type, which `docs/guides/api-standards.md` R2's row did
  not allow. Resolutions offered: amend the row (recommended); a new leaf
  package for the memo; holding it untyped. **The owner chose to amend the
  row**, citing 0062-MADR D5, recorded in the MADR's amendment "R2's row for
  internal/cacheref". `docs/guides/api-standards.md` joins this phase's
  scope for that cell and its citation; R56 stays Phase 6's.
* **Step 1's list, amended a sixth time (2026-10-25), before it changes:**
  `tools/test_tools.py`'s `TestCatalogCheck.test_a_planted_snapshot_reports_both`,
  which `tools/gate.py`'s `tools-test` found: it planted a difference for
  each of R1, R4, R5 and R6 and expected "rules: 4 differ". R5's and R6's
  agreement tests went with their rules, so it plants R1's and R4's and
  expects 2; `tools/catalogcheck.py` runs the agreement tests in `catalog`
  alone, and says why. And a test of `cacheref.LearnedThinking`, for the
  gate's `coverage-check` found `internal/cacheref` at 78.6 %, below its
  80 % floor, its nil branch untested.
* **Step 2, `llmprovider`:** `refusal.go`, `ThinkingShape` with
  `ThinkingAdaptive` and `ThinkingBudget`, `ReasoningRefusal`,
  `WithReasoningRefusals` (a later call adds) and `Settings.ReasoningRefusals`
  (a copy), with `New`'s check beside `checkForcedChoice`; `warning.go`,
  `WarningReasoningLearned` and `Warning.Refused`, tagged `omitzero` so that
  every other warning's JSON is as before, which a golden of the first build
  showed it was not. `EffortMax`'s and `WarningEffortDegraded`'s docs say an
  effort the model refused is sent lower too.
* **Step 3, `internal/wire`:** `effort.go`, `Ladder` and `DegradedWarning`;
  `ResolveMax`, `MaxEffort` and `EffortWarning` stay, for Ollama alone until
  Phase 5 (Q9 (c)), the warning's text unchanged; `learn.go`, `Learned` (the
  memo, keyed by `LearnedKey`), `Refusal`, `Matcher`, `QuotedEfforts`,
  `PlainEfforts`, `Learner` and `Retries`; `retry.go`, `MaxSends` (7) and
  `Send`'s loop; `sse.go`, `StreamEventsRetry`'s loop. `internal/cacheref`
  holds a `wire.Learned` per cache. The matchers, each the MADR's table:
  `messages.MatchRefusal`, `responses.MatchRefusal` (with the Grok cases'
  row), `chatcompletions.MatchRefusal`, and Gemini's `matchRefusal` beside its
  decoder; generateContent has none.
* **Step 4, each provider:** Claude, OpenAI, Grok, Gemini, OpenCode (every
  route), Hugging Face, Together and Kilo send through `wire.Send` and
  `StreamEventsRetry` with a per-request `wire.Learner`, joined with 0034's
  check by `wire.Retries` on the messages wire, and take their effort from
  it, the memo, `Settings.ReasoningRefusals` and the facts. Each provider's
  default for a request with no effort is kept (OpenAI's and OpenCode's
  responses route's medium, Grok's high). Removed: Grok's `reasoning.go`
  (R6); Gemini's `thinkingLevel` and `thinkingLevelFor` (R5, the xhigh
  mapping); OpenCode's `maxOnRoute` (0061's google row) and the chat-route
  gate, whose metadata now lists the facts, and whose silence sends the
  effort asked (Q7 (a)). Ollama is unchanged (Q9 (c)).
* **Step 5, the drift check:** `go test -race -count=1 -shuffle=on ./...`,
  36 packages, 0 failures. The tests that changed are step 1's, as amended
  before each changed.
* **Step 6, the tests,** each passing: `TestLadder_Rungs_0062` (10 cases),
  `TestMemo_KeyAndBound_0062`, `TestSend_Bound_0062`;
  `TestRefusal_Fixtures_0062` in `messages`, `responses`, `chatcompletions`
  and `gemini`, with `TestMatchRefusal_OtherList_0062` in `responses` and
  `chatcompletions`; `TestReasoningRefusals_Option_0062`;
  `TestLearn_Effort_0062`, over 11 providers and routes, Ollama out. The
  `effort-max` goldens, regenerated by `-update` run on those six alone, each
  differ only as D3's ladder with no facts says: `grok`, `grok-stateless` and
  `gemini` send `"max"` and no warning (were `"high"`, warned);
  `opencode-zen-google` keeps its dynamic budget, without the warning;
  `opencode-zen-chat` and `opencode-go-chat` gain `"reasoning_effort":
  "max"` (Q7 (a)). No other golden changed.
* **Step 7, the plants,** by `tools/plant.py` on scratch clones: 24, of which
  the PLAN's five (a matcher that never matches; the memo never read; the
  bound one higher; `WithReasoningRefusals` ignored; the ladder skipping rung
  2), and the ladder's rung 3; `Check` never learning; the learned warning
  without `Refused`; no degraded warning; the memo never dropping a model;
  the key keeping the model's case; `Retries` stopping at the first check; a
  stream resending once; `New` taking a refusal of both; `ReasoningRefusals`
  not a copy; each matcher row lost (the Messages effort and adaptive rows,
  the Responses list check and Grok field row, the Kilo row, the
  Interactions matcher and its budget row); Grok's `Stream` not learning; and
  the chat route sending no effort without metadata. 22 were caught. **Two
  were not**, and the tests were made to catch them:
  * the bound one higher: `TestSend_Bound_0062` read `MaxSends` itself; it
    now pins 7, Q5's answer, and the plant fails it (`MaxSends = 8; want 7`);
  * the Responses list check lost: no recording holds an `invalid_value`
    whose list is not effort words; `TestMatchRefusal_OtherList_0062` gives
    one, in `responses` and for Kilo's row in `chatcompletions`, and both
    plants fail it (`matched {Effort:max …}; the list is another
    parameter's`).

  The PLAN's red-first plant of the set, the memo never read, failed with
  `learn_0062_test.go:148: claude/claude-sonnet-5: efforts sent [max max];
  want max, then xhigh`.
* **Step 8, docs:** each changed provider's package doc under Degradations
  (and Grok's, Gemini's and OpenCode's Capabilities and facts lines);
  `CHANGELOG.md`, Added (D9's exports less `ProbeReasoning`, and 0061's
  `EffortMax` entry as it now behaves) and Changed; `docs/architecture.md`,
  the `internal/wire`, wire-format, `cacheref`, Gemini and Chat Completions
  providers' rows, the caches and the warnings; and R2's row, by the
  deviation above. Two live tests and four code comments, of step 1's fifth
  amendment.
* **Checks,** each exit 0: `make pre-add-check` on the 48 Go files, "44
  file(s) clean" and the 4 deleted; `go vet -tags live_gateways
  ./llmprovider/`; `make api-check`, "0 incompatible change(s)";
  `tools/apidiff.py v1.22.2`, 12 compatible additions, 0061's four and D9's
  eight, `ProbeReasoning` being Phase 4's; `make tables-check`, "38 packages,
  0 problem(s)"; `make changelog-check`; `make docs-check`; `make
  fixtures-check`, 328 files.
* **The gate, first run:** "gate: 25 of 28 passed". `coverage-check` and
  `gate-selftest` found `internal/cacheref` at 78.6 %, below its 80 % floor;
  `tools-test` found catalogcheck's test expecting R5's and R6's agreement
  tests. Fixed as step 1's sixth amendment says: `TestLearnedThinking_0062`
  (`internal/cacheref` 100 %), and `tools/catalogcheck.py` and its test. Each
  was seen to fail on a plant: `New` making no memo (`New's memos 0x0 and
  0x0; want one each`), and the planted snapshot losing its R4 difference
  (`'rules: 2 differ' not found`, "rules: 1 differ").
* **The gate, second run:** "gate: 28 of 28 passed"; `make pre-add-check` on
  `cacheref_0062_test.go`, clean.
* **Step 9, staged:** this phase's paths, 0061-PLAN Phase 3's code among
  them, reworked; with the two Grok fixtures, the report's, the MADR's and
  this record's changes. 0061-PLAN's record notes the rework.

### Phase 3: the shapes (2026-10-25)

Done; staged for the owner's commit with step 7's paths.

* **Stop 2026-10-25, before step 1: generateContent's request shape.** On
  OpenCode's google route, 0013-MADR B9 sends `low` as `thinkingLevel` and
  every other effort as the dynamic budget (`-1`). D2's step 3 says an effort
  means the level on generateContent, which would replace B9 for `medium`,
  `high`, `xhigh` and `max`; the MADR does not discuss B9, Phase 1 recorded
  only a level of `max` and a budget of 4096 there, both accepted, and the
  route has no matcher, so a level it refused would be returned, unlearned,
  where the request succeeds today. Resolutions offered: keep B9 and remove
  R4 (recommended); record first; follow D2 literally, superseding B9.
  **The owner chose to record first.**
  * Phase 1's recorder gains four cases, `thinkingLevel` `low`, `medium`,
    `high` and `xhigh` on Zen's google route for `gemini-3.8-flash`, each
    dropping `thinkingBudget`, run by the agent with the host's keys. Their
    fixtures, the report's rows and the MADR's amendment decide whether D2's
    level replaces B9. The recorder's change is in Phase 3's commit.
  * **Step 1's list, before it changes:** `TestRefusalCases_Table_0062`: 35
    thinking cases and 57 in all.
  * **Recorded (2026-10-25),** one send each: `low`, `medium` and `high`
    accepted; `xhigh` refused, 400 `INVALID_ARGUMENT`, "Invalid value at
    'generation_config.thinking_config.thinking_level'". Copied to the
    fixtures as before; `make fixtures-check`, "332 file(s), 0 problem(s)";
    the report's E11.
  * **Resolution (owner, 2026-10-25):** D2's level replaces 0013-MADR B9 on
    generateContent, with a matcher for the recorded refusal, as the MADR's
    amendment "Phase 3's google-route levels, and 0013's B9" records.
    Offered and not chosen: keep B9 and remove R4 alone; the level with no
    matcher. Step 2 gains `generatecontent.MatchRefusal`, and the route's
    learner uses it; step 4's google-route test learns `xhigh`.
* **Step 1, before any change:** the tests and tools that assert R1, R2, R4
  or B9, found by searching every `_test.go`, the live tests and `tools/` for
  `claudeAdaptiveOnly`, `minimaxAdaptiveThinking`, `geminiLegacyRE`,
  `ThinkingConfig`, `AddThinking`, `thinkingBudget`, `budget_tokens` and
  `"adaptive"`. Only these change:
  * `internal/wire/messages`: `TestClaudeAdaptiveOnly`, removed with R1;
    `TestAddThinking` and `TestAddThinkingFor_0041`, restated for D2's
    shape, which the caller passes: the facts', else a `Budget`'s budget,
    else adaptive, with the effort.
  * `internal/wire/generatecontent`: `TestThinkingConfig` and
    `TestThinkingConfigFor_0041`, restated: an effort as `thinkingLevel`, a
    `Budget` as `thinkingBudget`, D3's budgets (R3) for the budget shape
    otherwise; no R4, no dynamic budget.
  * `catalog`: `TestAgreement0041_MessagesThinking`,
    `TestAgreement0041_GeminiConfig`, `TestEvidence0041_ClaudeAdaptive` and
    `TestEvidence0041_GeminiLegacy`, removed with R1 and R4: each compared a
    rule with the models.dev data.
  * `tools/catalogcheck.py` and its `TestCatalogCheck`: no agreement test is
    left, so the tool reports the sixes alone, and its test, the README's row
    and its doc say so.
  * `providers/claude`: `TestClaudeThinking_RequestBody` and
    `TestThinkingWire_Claude`, whose models before Claude 4.7 got a budget by
    R1: with an effort and no facts they get adaptive thinking, and a
    refusal is learned.
  * `providers/opencode`: `TestOpencodeMessages_MinimaxM3ThinksAdaptive`
    (the effort is sent, Q12 (a)), `TestOpencodeMessages_OtherModelsKeepBudget`
    (adaptive with the effort, Q12 (a)), `TestOpencode_Thinking_PerRoute`,
    `TestGoogleRouteThinking_DynamicBudgetDefault`,
    `TestGoogleRouteThinking_RequestBody`, `TestThinkingWire_Gemini` and
    `TestThinkingWire_OpencodeRoutes` (the google route's level; the
    messages route's adaptive shape).
  * The live tests `TestLive_OpencodeMinimaxM3Adaptive` and
    `live_0041_facts_test.go`'s shape checks, where they assert a rule.
  * The G-wire goldens of the `thinking` and `thinking-tool` scenarios whose
    body changes, each listed and diffed once the code runs, before they are
    regenerated.
  * **Amended (2026-10-25), before it changes:**
    `providers/effortmax_0061_test.go`'s `TestEffortMax_Mapping_0061`, its
    google-route row with no facts: B9's dynamic budget sent no effort; the
    level sends `max`. And two tests of the list are renamed, as their names
    would say the opposite of what they assert:
    `TestOpencodeMessages_OtherModelsKeepBudget` to
    `TestOpencodeMessages_OtherModelsThinkAdaptive`, and
    `TestGoogleRouteThinking_DynamicBudgetDefault` to
    `TestGoogleRouteThinking_NoEffortNoLevel`.
  * **The goldens, listed (2026-10-25) before they are regenerated,** as the
    code's first run showed them: `opencode-zen-google`'s `thinking` and
    `thinking-tool` (B9's `thinkingBudget` `-1` gives way to the level shape
    with no level, their requests naming no effort) and `effort-max` (the
    level `max`); `opencode-go-messages`'s `thinking` and `thinking-tool`
    (qwen's 4096 budget, R1's, gives way to adaptive thinking) and
    `effort-max` (adaptive with `output_config.effort` `max`). The two
    `effort-max` goldens are another scenario than step 1 named. No other
    golden differs. The live tests `TestLive_OpencodeMinimaxM3Adaptive` and
    `live_0041_facts_test.go`'s checks do not change: the one sends no effort,
    and the other reads the facts' path.
  * **Step 1's list, amended (2026-10-25), before it changes:**
    `providers/claude/listing_0041_test.go`'s
    `TestListModels_DefaultBaseShape_0041`, which the search missed, as it
    names no rule function: it told the listing's facts from R1 by the shape,
    the model's id giving a budget by the rule and the listing's efforts
    adaptive thinking. With R1 removed both give adaptive thinking, so it
    passed without reading the listing. Its request now sets a `Budget` with
    the effort: the listing's efforts give adaptive thinking; with no facts,
    the `Budget` would give an enabled budget.
  * **Listed, and unchanged:** `TestClaudeThinking_RequestBody`,
    `TestOpencode_Thinking_PerRoute` and `TestGoogleRouteThinking_RequestBody`.
    Each sets a `Budget`, or asserts only where reasoning goes, so D2's shape
    is the one it asserted.
* **Step 2, the code:**
  * `internal/wire/learn.go`: the memo keeps each model's refused shapes;
    `Learner.Shape(preferred)` is D3's shape ladder, the preferred shape
    unless the model refused it and not the other; `Check` learns a refusal
    of the shape sent, with a `WarningReasoningLearned` whose `Refused` names
    the shape, and resends only when the other shape is not refused too. A
    passed `ReasoningRefusal` with a `Shape` seeds the ladder.
  * `internal/wire/wire.go`: `DefaultThinkingBudget` (4096, R3), which the
    messages package's default now names.
  * `internal/wire/messages`: R1 (`claudeAdaptiveOnly`) and R2
    (`minimaxAdaptiveThinking`) removed; `PreferredShape(facts, budget)`, the
    facts', else a `Budget`'s enabled budget, else adaptive; `AddThinkingFor`
    takes the shape. `ClaudeVersion` stays, for deferred tools.
  * `internal/wire/generatecontent`: R4 (`geminiLegacyRE`) and B9's dynamic
    budget removed; `PreferredShape` (listed levels, or no facts and no
    `Budget`, give the level); `ThinkingConfigFor(effort, budget, shape)`, an
    effort as `thinkingLevel`, none leaving the model's default, the budget
    shape R3's budgets; `refusal.go`, `MatchRefusal`, the amendment's row.
  * `providers/claude` and `providers/opencode`: the messages and google
    bodies take the shape from `Learner.Shape`; the google route's learner
    uses `generatecontent.MatchRefusal`.
* **Step 3, the drift check:** `go test -race -count=1 -shuffle=on ./...`,
  36 packages, exit 0. The tests that changed are step 1's, as amended before
  each changed. `git grep` finds none of `claudeAdaptiveOnly`,
  `minimaxAdaptiveThinking`, `geminiLegacyRE` and `dynamicThinkingBudget` in
  a Go file (each exit 1).
* **Step 4, the tests,** each passing:
  * `TestLearn_Shape_0062` (`providers`): Claude, adaptive refused then the
    budget accepted, and the reverse; OpenCode's messages route, both ways
    (Zen's `claude-sonnet-5`, Go's `qwen3.8-flash`); the google route,
    `xhigh` refused then `high` sent. For each, the second request is right
    the first time, and a fresh provider given the refusal by
    `WithReasoningRefusals` sends the other first. A model refusing both
    shapes has the second refusal returned after two sends; a stream learns
    before its first event. `newLearnServer` now takes a refusal per send.
  * `TestRefusal_Fixtures_0062` (`generatecontent`): of the route's four
    recorded failures, `MatchRefusal` reads only the `xhigh` level's, as
    `{Effort: xhigh}`, and not the empty contents, the string schema or
    `gemini-2.5-flash`'s `UNKNOWN` "Model is unavailable.";
    `TestMatchRefusal_Bounds_0062`, no match for another status, another
    code, or no effort sent.
  * The goldens, regenerated by `-update` on the six listed alone, differ
    only as listed: `opencode-go-messages`'s `thinking` and `thinking-tool`,
    `thinking` `{"budget_tokens": 4096, "type": "enabled"}` to
    `{"type": "adaptive"}`, and `effort-max`, that and `output_config`
    `{"effort": "max"}`; `opencode-zen-google`'s `thinking` and
    `thinking-tool`, `thinkingBudget` `-1` dropped, leaving
    `{"includeThoughts": true}`, and `effort-max`, `thinkingBudget` `-1` to
    `thinkingLevel` `"max"`. No other golden changed.
* **Step 5, the plants,** by `tools/plant.py` on scratch clones: 12, all
  caught. The PLAN's two, the shape ladder never flipping
  (`learn_0062_test.go:281: claude/claude-sonnet-4-5-20250929: sent
  [adaptive adaptive]; want [adaptive budget]`) and a shape refusal not
  learned (`claude HTTP 400 invalid_request_error: adaptive thinking is not
  supported on this model`, returned); and both shapes refused yet resent;
  the memo's shapes never read; a passed-back shape ignored;
  generateContent's matcher never matching, and ignoring the code; the
  google route with no matcher; each `PreferredShape` ignoring the
  `Budget`; the level shape dropping the effort; and Claude's request
  ignoring the facts' thinking, which the restated listing test fails
  (`thinking map[budget_tokens:2000 type:enabled]; want adaptive, from the
  listing's efforts`).
* **Step 6, docs:** Claude's and OpenCode's package docs (the shape, no
  rule of model names, the shape learned; the google route's level, its
  learned refusal, and B9 replaced), with the comments citing R1, R2 or R4
  in the two wire packages and the providers' facts helpers; `CHANGELOG.md`,
  Changed; `docs/architecture.md`, the `messages` and `generatecontent`
  rows (R43: `PreferredShape`, and `generatecontent.MatchRefusal`).
  `tools/catalogcheck.py`, its test and `tools/README.md`, by step 1's list.
* **Found, not done, for the owner:** three sentences still say that, with
  no facts, the name rules decide: `README.md`'s "Model facts and the menu",
  `docs/guides/api-standards.md`'s facts rule (`(0041 D1, D2, A3)`), and
  `docs/architecture.md`'s caches paragraph. None is in this phase's step 6;
  the last is Phase 6's step 2, the order of authority. They are false for
  the thinking shape and the effort since Phases 2 and 3.
* **Resolved (owner, 2026-10-25): "fix all docs".** Step 6 gains every doc
  outside the records that still states a removed rule, found by a search
  of `README.md`, `docs/*.md`, `docs/guides/*.md`, the unreleased
  `CHANGELOG.md` section and every Go doc comment for the rules' wording
  (name rules, version rules, the effort menu and its clamp, the dynamic
  budget, `xhigh` to `high`, MiniMax, `flash-lite`):
  * the three sentences above. `docs/architecture.md`'s states the order of
    authority, which Phase 6's step 2 named; that step keeps the memo and
    the resend path, and R56 stays Phase 6's step 1;
  * `docs/guides/migrating-from-mcplib.md`'s Grok row, which said the
    effort is "clamped to the model's menu as before": R6, removed in
    Phase 2;
  * `docs/guides/adding-a-provider.md`'s step 2, which says to encode
    `req.Reasoning` and nothing of the facts or refusals: a new provider
    would follow it into a rule of model names.

  The search's other hits are not about thinking, and stay: `README.md`'s
  "name rules" of the listing's order with `WithoutModelMetadata`, the
  catalog's ranking rules, the OpenCode routes' MiniMax rows, and the
  Gemini package doc's measured refusals.
* **Checks,** each exit 0: `make pre-add-check` on the 22 Go files, "21
  file(s) clean" and the one deleted; `go vet -tags live_gateways ./...`;
  `make api-check`, "0 incompatible change(s)"; `tools/apidiff.py v1.22.2`,
  the same 12 compatible additions as Phase 2, none new; `make
  tables-check`, "38 packages, 0 problem(s)"; `make fixtures-check`, "332
  file(s), 134 hit(s), 0 problem(s)"; `make changelog-check`; `make
  docs-check`; `make records-check`; `make tools-test`, 77 tests.
* **The gate, first run:** "gate: 26 of 28 passed". `coverage-check` and
  `gate-selftest` found `internal/wire` at 76.8 %, below its 80 % floor: the
  `Learner`, Phase 2's and this phase's shape ladder, was exercised only
  through the providers, in `learn.go`'s `Shape`, `Check`, `checkShape`,
  `NewLearner`, `QuotedEfforts` and `PlainEfforts` at 0 %. The floor
  stands. `internal/wire/learner_0062_test.go` tests the `Learner` in its own
  package: `TestLearner_Shape_0062` (the ladder, a shape refusal learned and
  warned, both refused, the memo across a model's case, the caller's
  refusals and another model's, a refusal of a shape not sent);
  `TestLearner_Effort_0062` (an effort refused with its list, the field
  refused, another effort's refusal, one learned already, nothing sent);
  `TestEffortLists_0062`. `internal/wire` is at 95.9 %. Nine plants on a
  scratch clone, all caught at last: the shape ladder never flipping; a
  shape refusal not learned; both refused yet resent; a passed-back shape
  ignored; another model's refusal taken; the field's refusal keeping a
  list; another effort's refusal learned; a list of other words read as
  efforts; and a refusal of a shape not sent learned, which the first
  version of the test did not catch, as its case refused a shape the memo
  held already. A case of a fresh memo was added, and the plant fails it
  (`learner_0062_test.go:57: a refusal of a shape not sent: budget; want it
  not learned, adaptive`). `make pre-add-check` on the file, clean.
* **The gate, second run:** "gate: 28 of 28 passed".
* **Step 7, staged:** this phase's paths, with the four google-route
  fixtures, the recorder's cases, the report's, the MADR's and this
  record's changes. `docs/decisions/0063-PLAN-align-docs-ci-and-disclosure-gates.md`
  is not this phase's: another session had staged it before this phase
  began, and this phase neither staged nor unstaged it.
