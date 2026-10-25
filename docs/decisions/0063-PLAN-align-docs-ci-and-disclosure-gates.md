---
status: proposed
date: 2026-10-25
associated-madr: "0063-MADR-align-docs-ci-and-disclosure-gates.md"
---
# Implement aligned docs-only CI and disclosure gates

Associated MADR:
[0063-MADR-align-docs-ci-and-disclosure-gates.md](0063-MADR-align-docs-ci-and-disclosure-gates.md)

## Goal

A documentation-only push or pull request gets a Docs run with the common
Markdown, record/link and portable disclosure gates plus this repository's
Markdown-reading checks. Other changes retain full CI. The private pre-push
guard remains part of publication.

## Scope

Affected files: `.github/workflows/ci.yml`, `.github/workflows/docs.yml`,
`Makefile`, `tools/denyscan.py`, `tools/test_tools.py`,
`scripts/test_gates.py`, `AGENTS.md`, `docs/architecture.md`,
`docs/README.md`,
`docs/reports/0011-REPORT-provider-source-compatibility-audit.md`, and
`docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md`.

Repository-specific work: Keep the existing filters and five docs targets. Add
a pinned `make markdownlint` target and the existing link checker to both
Linux CI and Docs; preserve the docs job's golangci-lint prerequisite and its
pin parity test. Add a portable tracked-text disclosure target to both
workflows, reusing the tested `tools/denyscan.py` home-path logic but without
relying on the runner's account or host names; keep the owner-specific
pre-push guard local. Extend `scripts/test_gates.py` to assert that the
docs-sensitive target set and pins in both workflows agree. Update `AGENTS.md`
and `docs/architecture.md` to state the common contract.

Boundary: Do not remove the docs job's golangci-lint install: its gate
self-test invokes the precheck. Keep full CI for non-Markdown edits, schedule
and tags.

## Implementation Steps

### Phase 1: inventory and shared checks

1. Re-read workflow triggers, job commands, Markdownlint globs, tracked
   Markdown files and every code or script reader of tracked Markdown. Record
   the docs-safe path set in this PLAN before editing filters. Include hidden
   agent-rule trees and test fixtures. Review required-status settings; if a
   filtered job would stay pending, amend this pair with a concrete migration
   before changing filters.
2. Format the two reports named in Scope to resolve the 380 existing
   Markdownlint findings, preserving their prose, links and tables. Confirm
   the pinned linter is clean before making it a CI gate. Add missing
   record/link and Markdownlint targets, reusing current commands where they
   exist. Run identical pinned documentation commands in full CI and Docs.
   Keep Go lint, tests, vulnerability, builds and releases in full CI.
3. Implement a standard-library portable disclosure check over tracked text.
   It rejects real macOS, Linux and Windows user-home paths, accepts
   documented placeholders, prints only file, line and category, and fails on
   unreadable tracked input. Run the same target in full CI and Docs. Retain
   the private deny list and global pre-push guard locally.
4. Add fixture tests for new gates. Prove each fails on deliberately broken
   input in a scratch copy and passes on clean input. Record commands,
   statuses and redacted diagnostics. Commit the approved phase after checks
   pass.

### Phase 2: workflow routing and parity

1. Keep push and pull-request CI ignore sets identical and limited to audited
   docs-safe paths. Make Docs paths cover every ignored path and the Docs
   workflow itself. Mixed changes run full CI. Preserve tags, schedules,
   dispatch, release jobs and permissions.
2. Add or extend a Python workflow contract check. It validates actual YAML
   structure, compares event filters, confirms ignored tracked paths are in
   Docs, refuses known Markdown inputs to code tests unless their reader runs
   in Docs, and compares common docs-gate commands and pins, and fails when
   full CI loses a tag, schedule or dispatch trigger it had, or gains a
   `paths` filter. Integrate it in the workflow/tool test target.
3. Run actionlint and the contract check. In independent scratch copies, plant
   a mismatched path, a Markdown reader wrongly ignored, a missing Docs gate
   and a differing pin; require each to fail. Commit the approved phase after
   checks pass.

### Phase 3: documentation and observed rollout

1. Explain routing, docs-safe paths, common checks, local readers and the
   private pre-push guard in AGENTS.md and the architecture or CI guide.
   Verify this pair's exact filenames and current statuses in the records
   index; use the repository index generator if present.
2. Run `make parity-check records-check tables-check changelog-check
   gate-selftest markdownlint links-check disclosure-check`; `actionlint` on
   both workflows; scratch plants for a bad link, MD018, a real home path, a
   missing docs target, and a mismatched tool pin. Capture status before
   filtering output. Check this pair and edited documents for real-machine
   identifiers. Read each edited file in full and verify links.
3. After explicit authorization to push in that same turn, observe a docs-only
   push and a non-doc or mixed push. Tag, schedule and dispatch routing is not
   waited for: GitHub evaluates no path filter for a tag push, `schedule` and
   `workflow_dispatch` carry none, and the contract check of phase 2, step 2
   fails if a trigger is lost or a `paths` filter is added. Record run IDs,
   conclusions and which workflow started. Mark this PLAN complete only after
   the criteria below hold; otherwise record the outstanding observation.
   Commit at the end of each approved phase under repository rules. *Amended
   2026-10-25: the wait for the next tag or scheduled run is removed, as
   acp-go-sdk2's 0037-PLAN-align-docs-ci-and-disclosure-gates.md did in its
   step 3.*

## Verification

* Both workflows pass pinned Markdownlint, numbered-record/index and
  relative-link validation, and portable tracked-text disclosure scanning.
* Each Markdown reader runs on changes to its input, either in Docs or full
  CI.
* A docs-only push starts Docs but not full CI; a non-doc or mixed push starts
  full CI. Tags, schedules and dispatch retain existing behavior, shown by the
  contract check and the workflows' structure, not by waiting for a run.
* Scratch plants fail their intended gates; filter or gate drift fails the
  parity test.
* Before an authorized push, the local pre-push guard checks outgoing commits.
  Its private deny list is never copied into CI.

## Rollout and Rollback

Land gates before relying on the filters. If routing misclassifies a change,
revert the routing commit through a new reviewed commit so full CI runs for
every push again. Keep the shared gates while repairing the audit. This PLAN
does not authorize a push or tag.
