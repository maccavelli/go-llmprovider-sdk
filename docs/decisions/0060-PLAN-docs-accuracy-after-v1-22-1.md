---
status: proposed
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

Not started: the pair awaits the owner's answers to Q1 to Q3 and approval.
