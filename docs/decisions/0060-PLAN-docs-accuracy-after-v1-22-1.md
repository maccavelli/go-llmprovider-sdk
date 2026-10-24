---
status: in-progress
date: 2026-10-24
associated-madr: "0060-MADR-docs-accuracy-after-v1-22-1.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Docs Update for v1.22.1's Wizard and the Local Gate

Associated MADR: [0060-MADR-docs-accuracy-after-v1-22-1.md](0060-MADR-docs-accuracy-after-v1-22-1.md)

## Goal

The MADR's D1 to D4, under the owner's answers to Q1 to Q3: every
finding of A, B and C resolved in the doc that holds it, each wizard
sentence traced to the test that shows it, and nothing but docs changed.

## Scope

### In scope

* `README.md`, "The configuration wizard" (D1).
* `docs/architecture.md`, "The wizard" (D1), and "Tooling" (D2, D3).
* `AGENTS.md`, "Tools" (D2), and under Q2 (a) or (b) the cache rule.
* `docs/guides/adding-a-provider.md`'s tests, under Q2 (a).
* Records: this pair, and its index rows.

### Out of scope

* Any Go file, test, `tools/` or `scripts/` file, CI file or `Makefile`
  (MADR D4).
* `CHANGELOG.md`: docs only, no release.
* Every other record, the dead Release links among them, under Q3 (a).

## Rules for every phase

* **Approval.** The owner approves this PLAN, after Q1 to Q3. A phase
  starts after the owner has committed the last.
* **The checks, before staging:** `python3 -B tools/gate.py` (its
  markdownlint covers the docs); `make docs-check`; `tools/linkcheck.py`;
  `tools/denyscan.py --changed`; `tools/claims.py` on the MADR;
  `make records-check`; `tools/recordlint.py --changed`. Each exit status is
  captured before any filter.
* **Each sentence against the code.** A wizard sentence changed is read
  against the function it describes, and names the test that shows it
  (step 1.2's table). A sentence no test shows is a stop condition.
* **Stage, then the owner commits** with `git commit --no-edit`. A push
  happens on the owner's word, after the disclosure guard; a push of
  Markdown alone runs the Docs workflow.
* **Stop conditions:** a finding of the MADR no longer holds; a change
  outside Scope is needed; a doc's sentence cannot be traced to a test.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted` with the answers, this
   PLAN `in-progress`.

### Phase 1: the docs

1. **A, the wizard (D1).** In `docs/architecture.md`, "The wizard":
   * "One copy of a session": a fresh session is saved once the listing
     has not refused it, after any refresh the listing left running; a
     saved session is checked before it is offered, and one that cannot be
     used is not, with a notice, and the sign-in menu follows (A4);
   * `Options.NoEndpointPrompt`: a plain-http URL to a remote host is used
     with a warning notice (A2);
   * "The paste prompt": three empty answers each within 50 ms stop the
     asking, with a notice, until the browser or the sign-in's limit (A3);
   * "What it accepts": plain http asked about unless under
     `NoEndpointPrompt` (A2); the shape sentence replaced: another vendor's
     key is named before the listing, and no shape is checked there (A1);
   * "A refused key": each refusal of a typed key without the provider's
     shape names the shape (A1);
   * "`TextPrompter`": it follows an escape sequence split across reads
     (A5);
   * a sentence for an ended context, which asks nothing, and one for a
     saved fallback the menu does not recommend, offered marked "saved"
     and preselected (A6).

   In `README.md`, "The configuration wizard": the session bullet (A4), the
   base-URL bullet (A2), and the search bullet's fallback sentence (A6).
2. **The trace,** recorded in the phase's record, one row per sentence:

   | Finding | Test |
   | :--- | :--- |
   | A1 | `TestShape_AcceptedKeyNoWarning_0059`, `TestShape_ForeignKey_0059`, `TestShape_RefusalNamesShape_0059` |
   | A2 | `TestBaseURL_NoPromptPlainHTTP_0059` |
   | A3 | `TestPaste_QuickEmptiesStop_0059`, `TestPaste_SlowEmptyAskedAgain_0059` |
   | A4 | `TestSession_RefusedKeepsSaved_0059`, `TestSession_AcceptedSavedOnce_0059`, `TestSession_RefreshAfterLimitSaved_0059`, `TestKept_UnusableNotOffered_0059`, `TestKept_UsableStillOffered_0059` |
   | A5 | `TestReadMasked_SplitEscape_0059`, `TestReadMasked_Keys` |
   | A6 | `TestFallbacks_SavedOffered_0059`, `TestConfigure_EndedContextAsksNothing_0059` |

3. **B, the gate and its rule (D2).** `docs/architecture.md`'s account of
   `tools/gate.py` names `test-repeat`. `AGENTS.md`'s tool list gains
   `tools/cachecensus.py [--any-url]`. Under Q2 (a) or (b), `AGENTS.md`
   states the rule: a test that lists or probes through the catalog gives
   its calls a cache of its own (`testCache(t)` in the wizard; each
   provider package's `build`), unless it tests the process's cache on
   purpose, under a credential unique to the run (0058-MADR D1, D6). Under
   Q2 (a), `docs/guides/adding-a-provider.md`'s tests say the same for a
   provider package.
4. **C, Tooling (D3), under Q1 (a).** The `make` target list matches the
   Makefile's `.PHONY` (`all` aside); the `gate-selftest` bullet names the
   ten gates `scripts/test_gates.py` plants.
5. **The checks** of the rules; the phase's record holds each result.

### Phase 2: close

1. **After the owner's commit:** a claims table in this PLAN's record,
   pinned to Phase 1's commit, holding one new sentence per finding, run
   by `tools/claims.py`, and three of its rows planted false with
   `--plants`, each caught.
2. **After the owner's push,** if asked: the Docs workflow passes.
3. **The MADR's Confirmation, item by item;** this PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | No doc states the removed shape warning, and each of A1 to A6 is described | 1 | the diff; step 2.1's claims |
| V2 | Each wizard sentence changed has its test | 1 | step 1.2's table |
| V3 | The gate's account and AGENTS' tools are complete; the rule is written (Q2) | 1 | the diff; step 2.1's claims |
| V4 | Tooling's lists match the Makefile and `scripts/test_gates.py` (Q1 (a)) | 1 | the diff against both |
| V5 | Only docs changed | 1 | `git diff --stat` |
| V6 | The checks pass | 1 | the rules' list |

## Rollout and Rollback

* **Rollout.** Commits on `main`: the records, Phase 1, Phase 2's record.
  No tag, no release.
* **Rollback.** A revert of Phase 1 restores the old text, and with it the
  inaccuracies.

## Execution record

### Phase 0: records (2026-10-24)

* **Approval.** The owner committed the pair as `5c75afc`, answered Q1
  (a), Q2 (a) and Q3 (a), and approved the PLAN ("committed, q1: a, q2: a,
  q3: a, proceed").
* **Step 1.** `tools/records.py status`: the MADR `accepted`, its Decision
  Outcome and questions with the answers; this PLAN `in-progress`; their
  index rows.
* **Not done here:** Phase 1 starts after the owner has committed this
  phase.
* **Committed** by the owner as `796f591` ("committed, proceed").

### Phase 1: the docs (2026-10-24)

* **Step 1, A (D1).** `docs/architecture.md`, "The wizard": the opening
  paragraph says an ended `ctx` returns before any question (A6, W8); two
  bullets after "One copy of a session", "When a session is saved" (A4,
  W2) and "Keeping a saved session" (A4, W6); `NoEndpointPrompt`'s
  plain-http warning (A2); the paste prompt's three quick empties (A3);
  "What it accepts": plain http "except under `NoEndpointPrompt`", and the
  shape sentence replaced (A1); "A refused key": the shape named on each
  refusal of a typed key without it (A1); "`TextPrompter`": a split escape
  sequence (A5); a new bullet, "Fallbacks" (A6, W7). `README.md`, "The
  configuration wizard": the session bullet (A4), the base-URL bullet
  (A2), the saved-fallback sentence (A6). Each sentence was written
  against the code read at `9ff2366`: `credentialStep` and
  `resolvedCredential.save`, `keepExistingOAuth`, `validateBaseURL`,
  `pastePrompt.input`, `warnForeignKey` and `notifyShape`, `escapeState`,
  `selectFallbacks`, and `ConfigureLLM`'s first check.
* **Step 2, the trace.** Each test of the table exists, once, in
  `wizard/`, and A4 also has `TestSession_RefreshRefusedNotSaved_0059`, for
  "A refresh the issuer refuses saves nothing and ends the run":

  | Finding | Sentence | Test |
  | :--- | :--- | :--- |
  | A1 | the shape not checked before the listing; another vendor's key named | `TestShape_AcceptedKeyNoWarning_0059`, `TestShape_ForeignKey_0059`, `TestShape_NoDiscoverNoPromise_0059` |
  | A1 | the shape named on each refusal | `TestShape_RefusalNamesShape_0059` |
  | A2 | plain http under `NoEndpointPrompt`, with a warning | `TestBaseURL_NoPromptPlainHTTP_0059` |
  | A3 | three quick empties, then a notice and the wait; a slow Enter asked again | `TestPaste_QuickEmptiesStop_0059`, `TestPaste_SlowEmptyAskedAgain_0059` |
  | A4 | saved once the listing has not refused it, with the rotated tokens | `TestSession_RefusedKeepsSaved_0059`, `TestSession_AcceptedSavedOnce_0059`, `TestSession_RefreshAfterLimitSaved_0059`, `TestSession_RefreshRefusedNotSaved_0059` |
  | A4 | a saved session checked first, an unusable one not offered | `TestKept_UnusableNotOffered_0059`, `TestKept_UsableStillOffered_0059` |
  | A5 | a split escape sequence adds nothing | `TestReadMasked_SplitEscape_0059`, `TestReadMasked_Keys` |
  | A6 | a saved fallback offered, marked and kept | `TestFallbacks_SavedOffered_0059`, `TestFallbacks_SavedOnlyRound_0059` |
  | A6 | an ended context asks nothing | `TestConfigure_EndedContextAsksNothing_0059` |

* **Step 3, B (D2).** `docs/architecture.md`'s account of `tools/gate.py`
  names `test-repeat`, and `tools/cachecensus.py` with `--any-url`.
  `AGENTS.md`'s tool list gains `tools/cachecensus.py [--any-url]`, and a
  paragraph after the tools states the cache rule (Q2 (a)).
  `docs/guides/adding-a-provider.md`'s tests gain the rule for a provider
  package; the nine provider packages' `helpers_test.go` each hold
  `testCache`, as it says.
* **Step 4, C (D3).** The `make` target list matches the Makefile's
  `.PHONY`, `all` aside; the `gate-selftest` bullet names the ten gates
  `scripts/test_gates.py` plants (its `assertFails` calls name dep-check,
  generate-check, parity-check, coverage-check, api-check, records-check,
  tables-check, fixtures-check, docs-check and changelog-check).
* **Step 5, the checks:** `make docs-check`, "sdkaudit: 0 violation(s) of
  R50,R51,R52"; `make tables-check`, "38 packages, 0 problem(s)";
  `tools/linkcheck.py`, 0 problems; `tools/denyscan.py --changed`, 0 hits;
  no line the change adds is longer than 82 characters; `python3 -B
  tools/gate.py`: "gate: 28 of 28 passed", markdownlint rc 0.
* **`git diff --stat`:** `AGENTS.md`, `README.md`, `docs/architecture.md`,
  `docs/guides/adding-a-provider.md` and this record; nothing else (V5).
* **Not done here:** Phase 2.
