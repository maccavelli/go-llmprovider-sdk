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

1. **The recorder,** `llmprovider/providers/refusals_0062_live_test.go`
   (build tag `live_gateways`, package `providers`):
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

3. **Its offline test,** `llmprovider/providers/refusals_0062_test.go` (no
   tag), runs the rewriter against a `wiretest` server and checks that:
   * the fields arrive as set;
   * a canned 400 is recorded with its status, code, message and redacted
     body;
   * a planted secret in the body is redacted.

   Red first: without the rewriter's set, the fields assertion fails.
4. **The owner runs it,** with the key of every provider in the table set
   (Q8 (b)):

   ```bash
   LLMPROVIDER_LIVE_0062=1 LLMPROVIDER_LIVE_0062_OUT=<dir> go test -tags live_gateways -count=1 -run Live_Refusals_0062 ./llmprovider/providers/
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
   * **The gate to Phase 2 (Q8 (b)):** no case is `skipped`. If one is, the
     record names it and its key variable; the owner sets the key and reruns
     that case by its subtest name; the agent then repeats this step for it.
     Phase 2 does not start while any case is `skipped`.
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
3. `refusals_0062_live_test.go` gains `TestLive_Learn_0062`, switched on by
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
  (Q8 (b)).
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
