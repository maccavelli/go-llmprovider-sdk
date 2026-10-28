---
status: in-progress
date: 2026-10-27
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
*Amended 2026-10-27 (the MADR's re-assessment, R1 and R2):* the two
reports leave the scope; `scripts/check_links.py` and
`scripts/check_disclosure.py` join it, new, and `tools/linkcheck.py`,
`tools/gate.py` and `tools/README.md` with `tools/denyscan.py`, to use
them.
*Amended 2026-10-27 (deviation, Phase 1's record):* `CHANGELOG.md` joins
the scope, for Phase 3.

Repository-specific work: Keep the existing filters and five docs targets. Add
a pinned `make markdownlint` target and the existing link checker to both
Linux CI and Docs; preserve the docs job's golangci-lint prerequisite and its
pin parity test. Add a portable tracked-text disclosure target to both
workflows, reusing the tested `tools/denyscan.py` home-path logic but without
relying on the runner's account or host names; keep the owner-specific
pre-push guard local. Extend `scripts/test_gates.py` to assert that the
docs-sensitive target set and pins in both workflows agree. Update `AGENTS.md`
and `docs/architecture.md` to state the common contract.
*Annotated 2026-10-27, checked at `eaaf709`:* "the existing link checker"
and "reusing the tested `tools/denyscan.py` home-path logic" are read as
R2 and R3 say: CI runs `scripts/` gates, whose code the tools reuse. "Its
pin parity test" is `scripts/test_gates.py`'s
`test_docs_installs_what_ci_installs_before_gate_selftest`, which compares
each `go install` target with its version.

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
   *Annotated 2026-10-27:* the MADR's re-assessment records this inventory
   at `5eff09e`, and that `main` requires no status check. Execution
   re-runs it and records any change.
2. Format the two reports named in Scope to resolve the 380 existing
   Markdownlint findings, preserving their prose, links and tables. Confirm
   the pinned linter is clean before making it a CI gate. Add missing
   record/link and Markdownlint targets, reusing current commands where they
   exist. Run identical pinned documentation commands in full CI and Docs.
   Keep Go lint, tests, vulnerability, builds and releases in full CI.
   *Amended 2026-10-27 (R1):* the reports are not reformatted. `make
   markdownlint` runs `npx --yes markdownlint-cli2@0.23.2 --no-globs` on
   `tools/gate.py`'s file set, every tracked Markdown file outside
   `docs/decisions/` and `docs/reports/`, from one list that the local gate
   and both workflows share; it is clean at `5eff09e`. The link check is
   `make links-check`, a `scripts/` gate (R2).
   *Annotated 2026-10-27, checked at `eaaf709`:* that list is built today in
   `tools/` (`tools/gate.py`'s `markdown_files`, from
   `tools/linkcheck.py`'s, every tracked or untracked `*.md` less the two
   record directories). Under R2 it moves to `scripts/`, and `tools/gate.py`
   imports it. At `eaaf709` it is 12 tracked files, and
   `markdownlint-cli2` v0.23.2 reports "0 issues in 0 files". `npx` is on
   the ubuntu-24.04 runner, which go-selfupdate-lib's Docs run shows, and on
   this host; the targets run on Linux only, as the other docs targets do.
3. Implement a standard-library portable disclosure check over tracked text.
   It rejects real macOS, Linux and Windows user-home paths, accepts
   documented placeholders, prints only file, line and category, and fails on
   unreadable tracked input. Run the same target in full CI and Docs. Retain
   the private deny list and global pre-push guard locally.
   *Amended 2026-10-27 (R2, R3):* it is `scripts/check_disclosure.py`,
   `make disclosure-check`: `tools/denyscan.py`'s home-path rule, the
   Windows form added, the `<user>` placeholder exempt, no host name read;
   0 hits over the 1,434 tracked files at `5eff09e`. `tools/denyscan.py`
   keeps its account and host checks, locally.
4. Add fixture tests for new gates. Prove each fails on deliberately broken
   input in a scratch copy and passes on clean input. Record commands,
   statuses and redacted diagnostics. Commit the approved phase after checks
   pass.

### Phase 2: workflow routing and parity

1. Keep push and pull-request CI ignore sets identical and limited to audited
   docs-safe paths. Make Docs paths cover every ignored path and the Docs
   workflow itself. Mixed changes run full CI. Preserve tags, schedules,
   dispatch, release jobs and permissions.
   *Annotated 2026-10-27, checked at `eaaf709`:* neither workflow has a
   `workflow_dispatch` trigger, so "dispatch" here and below preserves
   none. The triggers to keep are `ci.yml`'s push to `main` and `v*` tags,
   pull request and weekly schedule, and `docs.yml`'s push to `main` and
   pull request.
2. Add or extend a Python workflow contract check. It validates actual YAML
   structure, compares event filters, confirms ignored tracked paths are in
   Docs, refuses known Markdown inputs to code tests unless their reader runs
   in Docs, and compares common docs-gate commands and pins, and fails when
   full CI loses a tag, schedule or dispatch trigger it had, or gains a
   `paths` filter. Integrate it in the workflow/tool test target.
   *Amended 2026-10-27 (R4):* it reads the workflows' lines with the
   standard library, as `installs_before_selftest` does, and `actionlint`
   v1.7.12 validates the YAML; no YAML parser is added. Its plants go in
   `scripts/test_gates.py`, which `gate-selftest` runs in both workflows.
3. Run actionlint and the contract check. In independent scratch copies, plant
   a mismatched path, a Markdown reader wrongly ignored, a missing Docs gate
   and a differing pin; require each to fail. Commit the approved phase after
   checks pass.

### Phase 3: documentation and observed rollout

1. Explain routing, docs-safe paths, common checks, local readers and the
   private pre-push guard in AGENTS.md and the architecture or CI guide.
   Verify this pair's exact filenames and current statuses in the records
   index; use the repository index generator if present.
   *Amended 2026-10-27 (deviation, Phase 1's record):* and add an
   `[Unreleased]` Changed entry to `CHANGELOG.md` for the new gates and the
   routing, as 0052-MADR's change had in `v1.20.0`.
   *Annotated 2026-10-27, checked at `eaaf709`:* there is no CI guide;
   `docs/architecture.md`'s **CI** and **Docs** paragraphs say what the
   workflows run. The index tool is `tools/records.py`.
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

## Execution record

### Phase 1: inventory and shared checks (2026-10-27)

* **Approval.** "committed, proceed", after the owner's answers to R1 to R4
  were committed as `bd574ba`. It is taken as the MADR's acceptance and
  this PLAN's approval: the MADR is `accepted` and this PLAN `in-progress`,
  by `tools/records.py status`.
* **Step 1, the inventory, at `bd574ba`.** The MADR's re-assessment holds:
  * the workflows last changed in `e68be51` (2026-10-21);
  * 149 tracked Markdown files of 1,434 tracked files; 137 are records,
    under `docs/decisions/` and `docs/reports/`; none is under a `testdata`
    directory; the agent-rule trees hold `.claude/rules/madr-and-plan-skill.md`,
    `.grok/rules/madr-plan-before-mutating-work.md` and `.opencode/rules.md`;
  * `main` is not protected and has no branch rules, so no check is
    required and no filtered job can stay pending.
* **The docs-safe path set** is `**.md`, as `ci.yml` ignores it today: no
  tracked Markdown file is an input to a gate that only full CI runs (the
  MADR's re-assessment, "Every reader of Markdown runs in Docs"). Phase 2
  keeps it.
* **Step 2, done.**
  * `scripts/check_links.py` (`make links-check`): `tools/linkcheck.py`'s
    check, moved, with `--lint-files`, the list `make markdownlint` lints:
    every Markdown file git tracks or would add, less `docs/decisions/` and
    `docs/reports/`. It exits 2 on an empty list or a name with white
    space, which the target would split.
  * `Makefile`: `MARKDOWNLINT ?= npx --yes markdownlint-cli2@0.23.2`;
    `markdownlint` runs `files="$$(python3 -B scripts/check_links.py
    --lint-files)" && $(MARKDOWNLINT) --no-globs $$files`, so a list that
    fails stops it before the linter; `links-check` and `disclosure-check`.
  * Both workflows run `markdownlint links-check disclosure-check` on the
    `make` line of their documentation gates, before `gate-selftest`, on
    Linux; `docs.yml`'s header says so. `actionlint` v1.7.12: clean on both.
  * Run on the tree: `make markdownlint` by `npx`, "markdownlint-cli2
    v0.23.2 (markdownlint v0.41.1)", "Linting: 12 files", "Summary: 0
    issues in 0 files", and the same with the local binary; `make
    links-check`, "0 problem(s), 1050 relative link(s) in 149 file(s)".
* **Step 3, done.** `scripts/check_disclosure.py` (`make
  disclosure-check`): every file git tracks or would add, or those named;
  `/Users/<name>/` and `/home/<name>/`, and a drive letter, `Users` and a
  name after a slash or one or two backslashes; `<user>` exempt; no account
  name, host name or deny list read; a hit printed as path, line and kind;
  a file it cannot read exits 2. On the tree: "0 hit(s) in 1436 file(s)",
  the 1,434 tracked files and the two new scripts. A path such as
  `C:/Users/<name>/` is reported as both kinds, since its `/Users/<name>/`
  is a macOS one too.
* **The tools reuse the code (R2).** `tools/_common.py` appends `scripts/`
  to the import path, after the tools, so no tool is shadowed; no module
  name is shared. `tools/linkcheck.py` imports `check_links.check` and keeps
  its CLI and `--detail`; `tools/denyscan.py` takes its home-path rule from
  `check_disclosure.KINDS`, so it catches Windows paths too, and keeps its
  account and host checks. `tools/gate.py` runs `make markdownlint
  MARKDOWNLINT=markdownlint-cli2`, `make links-check` and `make
  disclosure-check` in place of its own markdownlint and `g-links` checks:
  29 checks. `tools/README.md` and `docs/architecture.md`'s layout name the
  new scripts.
* **Step 4, the tests.** In `scripts/test_gates.py`, both gates join
  `GATES`, so the clean-copy test runs them, and seven tests plant on the
  copy: a broken link; a new Markdown file in the lint list and no record
  in it; `make markdownlint` with `MARKDOWNLINT=echo LINTED`, which lints
  the list, and with a file named with a space, which fails before the
  linter; a macOS and a Linux home path; a Windows one in each separator
  form; the placeholders, which pass; and an unreadable file, exit 2. Each
  planted path is built at run time, so the file holds none. In
  `tools/test_tools.py`, the gate's check list is the new one, the test of
  the removed `gate.markdown_files` moved to `scripts/test_gates.py`, and
  `denyscan` catches a Windows home path.
* **Plants** (`tools/plant.py`, on scratch clones; all 10 caught, each by
  the test aimed at it):
  * links-check ignoring a missing target: "links-check exit 0, want 1";
  * the lint list keeping the records: the records listed, "!= []";
  * the white-space guard removed, and `&&` made `;` in the target: each
    "make markdownlint passed";
  * Linux home paths dropped: "disclosure-check exit 0, want 1"
    (`home='home'`);
  * doubled backslashes dropped: the same (`sep='\\\\'`);
  * the placeholder reported: "placeholders were reported";
  * an unreadable file skipped: "exit 0, want 2";
  * a hit printing its line: "disclosure-check printed the path it found";
  * `tools/denyscan.py` back on the Unix rule alone: its Windows test.
* **Checks.** `make tools-test`: 77 tests, OK. `python3 -B tools/gate.py`:
  "gate: 29 of 29 passed"; its `gate-selftest`, 45 tests, OK; its
  `markdownlint`, "Linting: 12 files", "0 issues"; `links-check` and
  `disclosure-check` as above.
* **Not done here:** the routing and the contract check (Phase 2); the
  contract in `AGENTS.md` and `docs/architecture.md`'s CI and Docs
  paragraphs (Phase 3).
* **Deviation (2026-10-27): `CHANGELOG.md`.** The scope left it out, but
  0052-MADR's CI change has an entry (`v1.20.0`, Changed). Put with three
  options, the owner chose "Entry in Phase 3": `CHANGELOG.md` joins the
  scope, and Phase 3 adds an `[Unreleased]` Changed entry once the routing
  is final. Not chosen: an entry now, and none.

### Phase 2: workflow routing and parity (2026-10-27)

* **Approval.** "committed proceed", after Phase 1 was committed as
  `b407f02`.
* **Step 1, the routing: unchanged.** At `b407f02`, `ci.yml` ignores
  `['**.md']` on push and on pull request alike, the docs-safe set of
  Phase 1; `docs.yml` runs on `['**.md', '.github/workflows/docs.yml']` on
  both, so it covers every ignored path and itself. A mixed change matches
  no `paths-ignore` entry for its other files, so `ci.yml` runs. `ci.yml`
  keeps its push to `main` and `v*` tags, pull request, weekly schedule,
  `release` job, `contents: read`, and the release job's own `contents:
  write`; neither workflow has
  `workflow_dispatch`. No filter is edited.
* **Step 2, the contract check.** `scripts/test_gates.py`'s
  `WorkflowContractTest`, beside 0052-MADR A2's `WorkflowsTest`, so
  `gate-selftest` runs it in both workflows. It reads the workflows' lines
  with the standard library (R4): `triggers` reads the `on:` block, each
  event, its filters as flow lists and its list items, and fails on a line
  of any other shape; `selftest_targets` reads the `make … gate-selftest`
  line. Four tests:
  * `ci.yml` keeps push, pull request and schedule, `v*` tags, `main` and a
    cron, gains no `paths` filter, and ignores the same paths on push and
    pull request;
  * `docs.yml` runs on the same paths for push and pull request, on every
    path `ci.yml` ignores and on itself, with no `paths-ignore`;
  * every gate either workflow runs is in `MARKDOWN_GATES` (the gates that
    read Markdown or plant in it, and `disclosure-check`) or `CODE_GATES`
    (those that read none, by the MADR's re-assessment); `docs.yml` runs
    every Markdown gate and no code gate, and `ci.yml` runs every Markdown
    gate. A new gate fails the test until it is placed;
  * `docs.yml`'s actions are `ci.yml`'s at the same commits, and neither
    workflow sets `MARKDOWNLINT`, whose pin the `Makefile` holds alone.
    `golangci-lint`'s pin stays with `WorkflowsTest`.
* **Step 3.** `actionlint` v1.7.12 on both workflows: clean. Plants
  (`tools/plant.py`, on a scratch clone, running `WorkflowContractTest` and
  `WorkflowsTest`; all 11 caught):
  * a mismatched path, pull requests ignoring `docs/**` too: "Lists differ:
    ['**.md'] != ['**.md', 'docs/**']";
  * a mismatched path, `docs.yml` on `docs/**.md` only: "Lists differ:
    ['**.md'] != []";
  * a Markdown reader wrongly ignored, `tables-check` out of `docs.yml`:
    "['tables-check'] != []";
  * a missing Docs gate, `disclosure-check` out of `docs.yml`:
    "['disclosure-check'] != []";
  * a differing pin, `docs.yml`'s `setup-go` at another commit: that
    `uses` listed; `docs.yml`'s `golangci-lint` at `v2.13.0`: `WorkflowsTest`;
    `MARKDOWNLINT=` set in `docs.yml`: "'MARKDOWNLINT' unexpectedly found",
    and the setting reported as an unplaced gate;
  * `ci.yml` losing its schedule: "ci.yml lost its schedule trigger";
  * `ci.yml` gaining `paths: ['**.go']`: "ci.yml's push gained a paths
    filter";
  * an unplaced gate, `new-check`: "['new-check'] != []";
  * `paths-ignore` written as a block list: "an `on:` line of a shape the
    contract does not read: '    paths-ignore:'".
* **Checks.** `python3 -B tools/gate.py`: "gate: 29 of 29 passed"; its
  `gate-selftest`, 49 tests, OK.
* **Not done here:** Phase 3, the contract in `AGENTS.md` and
  `docs/architecture.md`, the `CHANGELOG.md` entry, and the observed push.

### Phase 3: documentation and observed rollout (2026-10-27)

* **Approval.** "committed, proceed", after Phase 2 was committed as
  `0b25503`.
* **Step 1, done.**
  * `AGENTS.md`: the CI and Docs target lists with the three new gates;
    what each does, the `Makefile`'s one pin, and that a new gate goes in
    `MARKDOWN_GATES` or `CODE_GATES`; under Tools, that `tools/linkcheck.py`
    and `tools/denyscan.py` import the `scripts/` checks; under
    Identifiers, that CI's `disclosure-check` covers home paths and the
    account, host and deny list are checked only on the host.
  * `docs/architecture.md`: a bullet for the three gates; `gate-selftest`'s
    gates and its workflow contract; the local gate's checks; CI's and
    Docs' target lists, and `markdownlint` through `npx`. There is no CI
    guide (Phase 3 step 1's annotation).
  * `CHANGELOG.md`: an `[Unreleased]` Changed entry (the deviation of
    Phase 1).
  * The index: `docs/README.md` rows 144 and 145 name this pair's files,
    `accepted` and `in-progress`, set by `tools/records.py` in Phase 1.
* **Step 2, done.** `make parity-check records-check tables-check
  changelog-check gate-selftest markdownlint links-check disclosure-check`:
  exit 0; "G-parity: 409 identifiers, … 0 problem(s)"; "records-check: 137
  records, 0 problem(s)"; "check-tables: 42 packages, 0 problem(s)";
  "changelog-check: 36 release section(s), 0 problem(s)"; `gate-selftest`
  "Ran 49 tests", OK; `markdownlint` by `npx`, "Summary: 0 issues in 0
  files"; "links-check: 0 problem(s), 1051 relative link(s) in 149
  file(s)"; "disclosure-check: 0 hit(s) in 1436 file(s)". `actionlint` on
  both workflows: clean. Plants, on scratch clones (all 5 caught, each by
  its own gate):
  * a bad link in `AGENTS.md`: `make: *** [links-check] Error 1`;
  * MD018 (`#Adding a provider`): `make: *** [markdownlint] Error 1`;
  * a real home path in `tools/README.md`, built at run time: `make: ***
    [disclosure-check] Error 1`;
  * a missing docs target, `links-check` out of `docs.yml`:
    "['links-check'] != []";
  * a mismatched tool pin, `docs.yml`'s checkout at another commit: that
    `uses` listed.

  The make plants ran `make -k markdownlint links-check
  disclosure-check MARKDOWNLINT=markdownlint-cli2`, so each other target
  ran and passed. `tools/denyscan.py --changed`: "0 hit(s) in 4 file(s)";
  `tools/recordlint.py --changed`: no rule rose. Each edited file was read
  in full in its diff.
* **Step 3 waits on the owner.** Five commits are unpushed, `5eff09e` to
  `0b25503`; the disclosure guard over them exits 0. To observe both
  routes: a push of them, which changes code, should start CI and Docs;
  then a push of this phase's commit, Markdown alone, should start Docs
  alone. Each push needs the owner's word in its own turn.
* **Step 3, the code route, observed (2026-10-28).** The owner committed
  this phase as `d96d285`; the disclosure guard over the six outgoing
  commits, `5eff09e` to `d96d285`, exited 0. The owner pushed all six at
  once, not the two pushes proposed: `origin/main` went from `e1ec36d` to
  `d96d285`. The range changes code, so this observes the code route.
  * **Started:** Docs, run `40700208640`, and CI, run `40700208671`, both
    on push, both at 2026-10-28T13:50:52Z.
  * **Docs: success.** "G-parity: 409 identifiers, … 0 problem(s)";
    "records-check: 137 records, 0 problem(s)"; "check-tables: 42
    packages, 0 problem(s)"; "changelog-check: 36 release section(s), 0
    problem(s)"; `npx --yes markdownlint-cli2@0.23.2 --no-globs $files`,
    "markdownlint-cli2 v0.23.2 (markdownlint v0.41.1)", "Linting: 12
    files", "Summary: 0 issues in 0 files"; "links-check: 0 problem(s),
    1051 relative link(s) in 149 file(s)"; "disclosure-check: 0 hit(s) in
    1436 file(s)"; `gate-selftest` "Ran 49 tests", OK.
  * **CI: success.** `validate` on ubuntu-24.04, macos-15 and windows-2025
    success, `release` skipped (not a tag); its Linux gates printed the
    same lines as Docs, `gate-selftest` "Ran 49 tests", OK.
  * **The Markdown-only route** is observed by the push of the commit that
    records this: it changes this PLAN alone, so Docs alone should start.
