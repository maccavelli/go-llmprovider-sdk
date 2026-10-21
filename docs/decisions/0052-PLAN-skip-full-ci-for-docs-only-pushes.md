---
status: in-progress
date: 2026-10-20
associated-madr: "0052-MADR-skip-full-ci-for-docs-only-pushes.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Skipping the Full CI for Docs-Only Pushes, With a Docs Workflow

Associated MADR: [0052-MADR-skip-full-ci-for-docs-only-pushes.md](0052-MADR-skip-full-ci-for-docs-only-pushes.md)
(option B, proposed; Q1 answer (a) assumed below, and changed in Phase 1
step 2 if the owner answers otherwise). Annotated 2026-10-20: the MADR is
`accepted`, and Q1 was answered (a), so Phase 1 step 2 stands as written
(execution record, "Approval (2026-10-20)").

## Goal

A push to `main` or a pull request that changes only Markdown runs one Linux
"Docs" job, which runs the gates a Markdown change can break, instead of
`ci.yml`'s three-OS `validate`. Any other change, every `v*` tag (and so
every Release) and the weekly run keep `ci.yml` exactly as it is.

## Scope

### In scope

| Phase | What | Commit |
| :--- | :--- | :--- |
| 0 | this pair's statuses and index rows, on approval | records only |
| 1 | `ci.yml`'s two `paths-ignore` lines and comment; `.github/workflows/docs.yml` | CI |
| 2 | `docs/architecture.md` (layout block and CI bullet), `AGENTS.md` (one sentence), `CHANGELOG.md` (Unreleased) | docs |
| 3 | the live checks, and closing | records |

### Out of scope

* Any change to `ci.yml`'s jobs, steps, pins, schedule or concurrency, and
  to the `release` job.
* Adding markdownlint or the link check to CI. `tools/gate.py` runs both
  locally and CI runs neither today (MADR C26); putting them in `docs.yml`
  alone would hold a docs-only push to more than a code push. A record of
  its own may add them to both.
* Rulesets and required checks: none exist. The MADR's consequences say
  what a later ruleset must respect.
* A release. The change is CI and documents only; the next release's
  changelog section carries the Unreleased entry.
* The `Makefile` and `scripts/go-precheck.sh`, which read the
  `govulncheck` pin from `ci.yml` (MADR C24, C25) and keep doing so.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase, and
  answers the MADR's Q1; a phase starts after the owner has committed the
  last.
* **The gate, before staging:** `python3 -B tools/gate.py`;
  `tools/claims.py` on both records; `make records-check`;
  `tools/recordlint.py --changed`; `tools/denyscan.py --changed`;
  `tools/linkcheck.py`; `actionlint` on each changed workflow. No Go file
  changes, so `make pre-add-check` has nothing to check. Each exit status is
  captured before any filter, and read before anything is written from it.
* **Red first.** Every check relied on is seen to fail on a planted break,
  by `tools/plant.py` on a scratch clone, or on a scratch copy of a
  workflow, never in the tree; the execution record holds each plant and
  its failure line.
* **Stage, then the owner commits** with `git commit --no-edit`; push on
  the owner's word; the disclosure guard runs over the outgoing commits
  before the owner pushes.
* **Stop conditions:** a fact of the MADR or a claim below no longer holds;
  a step would change a `ci.yml` job or step; `make vuln` would print a
  different pin; a planted break is not caught; a live check shows a
  docs-only push starting "CI", or a non-Markdown push not starting it.
  Stop, record a dated deviation here (and an amendment in the MADR when a
  decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted` with the owner's answer
   to Q1 recorded under "Owner questions", this PLAN `in-progress`, each
   with its index row.
2. `make records-check`, `tools/claims.py` on both records,
   `tools/linkcheck.py`.

### Phase 1: the filters and the docs workflow (MADR D1, D2)

1. **`ci.yml`** (claims P1 to P3):
   * `on.push` gains `paths-ignore: ['**.md']` after `tags: ['v*']`, with
     a comment above it: a change to Markdown alone runs `docs.yml`
     instead; GitHub does not evaluate path filters for a tag push, so every
     tag, and its `release` job, runs this in full
     (`docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md`);
   * `on.pull_request` gains `paths-ignore: ['**.md']`;
   * nothing else changes. `git diff` of `ci.yml` shows only those lines.
2. **`.github/workflows/docs.yml`,** new, exactly:

   ```yaml
   name: Docs

   # Checks a change to Markdown alone, which ci.yml skips: the gates that
   # read Markdown (parity, records, tables, changelog) and the gates'
   # self-test, whose plants replace exact strings in the docs and whose
   # clean test runs every gate. The steps are ci.yml's, by the same make
   # targets. A change to anything else runs ci.yml, and so does every tag
   # (docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md).
   on:
     push:
       branches: [main]
       paths: ['**.md', '.github/workflows/docs.yml']
     pull_request:
       paths: ['**.md', '.github/workflows/docs.yml']

   # A newer push to a pull request cancels its older run; main runs each push.
   concurrency:
     group: docs-${{ github.ref }}
     cancel-in-progress: ${{ github.event_name == 'pull_request' }}

   permissions:
     contents: read

   jobs:
     docs:
       runs-on: ubuntu-24.04
       timeout-minutes: 15
       steps:
         - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
           with:
             # Every tag, for changelog-check.
             fetch-depth: 0
             persist-credentials: false
         - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
           with:
             go-version-file: go.mod
         - run: go mod download
         - name: parity, records, the tables, the changelog, the gates' self-test (0052-MADR D2)
           run: make parity-check records-check tables-check changelog-check gate-selftest
   ```

   The two action SHAs are `ci.yml`'s (claims P4, P5). If the owner
   answers Q1 with (b), the last `run:` is `ci.yml`'s line 60 verbatim; with
   (c), it is `make parity-check records-check tables-check changelog-check`
   and then the `-k` command the MADR names. The header comment changes to
   match.
3. **Seen failing,** with `tools/plant.py` on a scratch clone, its
   `command` the docs job's last `run:` line, as a list. Each plant must
   fail the command, at the target named:

   | Plant | File | Change | Fails at |
   | :--- | :--- | :--- | :--- |
   | X1 | `docs/README.md` | 0051-PLAN's row: `complete` becomes `in-progress` | `records-check` |
   | X2 | `AGENTS.md` | `` `LLMPROVIDER_LIVE_CHATGPT_PLAN` (the ChatGPT plan sign-in), `` becomes `the ChatGPT plan sign-in,` | `tables-check` |
   | X3 | `docs/guides/migrating-from-mcplib.md` | the first `llmprovider.` row's SDK cell becomes `TBD` | `parity-check` |
   | X4 | `CHANGELOG.md` | `## [v1.19.0] - 2026-10-20` becomes `- 2026-10-19` | `changelog-check` |
   | X5 | `AGENTS.md` | `(the ChatGPT plan sign-in),` becomes `(the ChatGPT plan's sign-in),`: the variable is still named | `gate-selftest`, `test_tables_check_live_variable_unnamed`: "found 0 times" |

   The same command passes on an unplanted clone. X5 is the case that puts
   `gate-selftest` in the docs job: every gate passes, and only the
   self-test fails.

   `actionlint` exits 1 on a scratch copy of `docs.yml` whose concurrency
   line reads `github.event_nam`, and 0 on the file.
4. **Checks:** `actionlint .github/workflows/ci.yml
   .github/workflows/docs.yml`; `make vuln` prints `ci.yml`'s pin as before
   (or `grep -oE` with `Makefile:10`'s pattern on `ci.yml` gives
   `golang.org/x/vuln/cmd/govulncheck@v1.8.0`); `tools/gate.py`. The filters
   themselves are proven only on GitHub, in Phase 3.

### Phase 2: the documents (MADR D3)

1. **`docs/architecture.md`:**
   * the layout block gains `.github/workflows/docs.yml  Docs: the
     Markdown gates, for a Markdown-only change` after `ci.yml`'s line
     (claim P6);
   * the CI bullet (claim P7) says that a push or pull request that changes
     only Markdown skips it, and `docs.yml` runs `parity-check`,
     `records-check`, `tables-check`, `changelog-check` and `gate-selftest`
     on one Linux runner instead; and that a tag push and the weekly run are
     never filtered, so a Release still follows a full `validate`.
2. **`AGENTS.md`:** after the sentence that lists what CI runs (claim P8),
   one sentence: a Markdown-only push or pull request runs `docs.yml`
   instead, citing the MADR in backticks, as AGENTS.md cites records.
3. **`CHANGELOG.md`:** under `## [Unreleased]` (claim P9), `### Changed`:
   a Markdown-only push or pull request runs a Docs job, not the full CI,
   with the MADR linked. `make changelog-check` stays at 0 problems.
4. **Checks:** `tools/linkcheck.py`, `tools/denyscan.py --changed`,
   markdownlint through `tools/gate.py`, `make tables-check`. This commit
   changes Markdown only, so its push is Phase 3's docs-only test.

### Phase 3: live checks and closing

1. **A push with a non-Markdown change:** Phase 1's push changes the two
   workflows, so "CI" starts, and "Docs" starts too, its own file being in
   its `paths`. Both results, run ids and job counts are recorded.
2. **A docs-only push:** Phase 2's push. `gh api
   "repos/maccavelli/go-llmprovider-sdk/actions/runs?head_sha=<sha>"` lists
   one run, "Docs", `success`, with its duration; no "CI" run started.
3. **Tags:** the next release's tag push shows a full "CI" run and its
   `release` job; that release's PLAN records it, and this plan names it as
   left to that release.
4. `### Verification, at closing`, `tools/records.py status` to
   `complete`. The MADR stays `accepted`.

### Phase 4: the docs job's golangci-lint (MADR amendment A1, A2)

Added 2026-10-20, by the owner's decision ("fix it option 1 + check"),
after the docs job failed on its first two runs (execution record,
"Deviation (2026-10-20): the docs job failed on its runs").

1. **`docs.yml`:** after `go mod download`, a step that runs `go install
   github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`, the
   line `ci.yml` runs, with a comment citing A1 and A2.
2. **`scripts/test_gates.py`:** `WorkflowsTest`, as A2 gives it, with
   `installs_before_selftest`, which reads a workflow's `go install`
   targets above its `make … gate-selftest` line.
3. **The documents:** the architecture's Docs paragraph names the
   golangci-lint install and the test; `CHANGELOG.md`'s `[Unreleased]`
   gains a Fixed entry.
4. **Red first:**
   * the precheck test, on this host, with `GOLANGCI_LINT` naming a
     missing file, fails as the runs did;
   * `tools/plant.py` on a scratch clone, with `WorkflowsTest` as its
     command: docs.yml's install removed; at another version; moved after
     the `make` step; `ci.yml`'s pin raised alone; docs.yml's `make` line
     lacking `gate-selftest`. Each fails the test.
5. **Checks:** `actionlint` on both workflows; `make gate-selftest`;
   `tools/gate.py`; claims, records, recordlint, the link and identifier
   checks.
6. **Live:** after the owner pushes, "Docs" passes; its run id and time are
   recorded, and so is the install step's time.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | `ci.yml` has `paths-ignore: ['**.md']` on push and pull request, and no other change | 1 | `git diff`; `actionlint` |
| V2 | `docs.yml` runs the docs job of D2 (or Q1's answer), with `ci.yml`'s pins | 1 | the file; `actionlint` |
| V3 | Each planted break fails the docs command at its target, and the tree passes | 1 | `tools/plant.py`, X1 to X5 |
| V4 | The `govulncheck` pin is still read from `ci.yml` | 1 | `make vuln` or the `grep` |
| V5 | A docs-only push ran only "Docs"; a non-Markdown push ran "CI" | 3 | the run ids |
| V6 | The next tag ran the full CI and made its Release | later | that release's PLAN |
| V7 | The architecture, AGENTS.md and the changelog say so | 2 | the files; link, identifier and lint checks |
| V8 | The docs job installs golangci-lint at `ci.yml`'s pin, and its `gate-selftest` passes on a runner | 4 | the file; the first "Docs" run after the push |
| V9 | `WorkflowsTest` fails on each planted break, and passes on the tree | 4 | `tools/plant.py` |

## Rollout and Rollback

* **Rollout.** Three commits on `main` after the records (Phases 1, 2 and
  3's closing), staged by the agent and committed by the owner, each pushed
  on the owner's word.
* **Rollback.** `git revert` of Phase 1 restores the full CI for every push
  and removes `docs.yml`; Phase 2's sentences are then reverted with it, or
  amended. Nothing outside this repository depends on the split.

## Code claims

<!-- claims: self=cadaad5 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The tag trigger, after which `paths-ignore` goes | `.github/workflows/ci.yml:5` | `tags: ['v*']` |
| P2 | The pull-request trigger | `.github/workflows/ci.yml:6` | `pull_request:` |
| P3 | The schedule, which stays unfiltered | `.github/workflows/ci.yml:9` | `schedule:` |
| P4 | The checkout pin | `.github/workflows/ci.yml:30` | `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1` |
| P5 | The setup-go pin | `.github/workflows/ci.yml:34` | `actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0` |
| P6 | The layout block's CI line | `docs/architecture.md:29` | `.github/workflows/ci.yml    CI` |
| P7 | The CI bullet's triggers | `docs/architecture.md:923` | `` the Go version read from `go.mod`, on each push to `main`, each pull `` |
| P8 | AGENTS.md's list of what CI runs | `AGENTS.md:148` | `` CI also runs `make parity-check dep-check coverage-check api-check `` |
| P9 | The changelog's Unreleased section | `CHANGELOG.md:11` | `## [Unreleased]` |
| P10 | X2 and X5's text | `AGENTS.md:196` | `` `LLMPROVIDER_LIVE_CHATGPT_PLAN` (the ChatGPT plan sign-in), `` |
| P11 | X4's text | `CHANGELOG.md:13` | `## [v1.19.0] - 2026-10-20` |
| P12 | X1's row | `docs/README.md:121` | `(decisions/0051-PLAN-opt-in-streaming-retry.md) \| complete \|` |

## Execution record

### Approval (2026-10-20)

The owner approved this pair as recommended on 2026-10-20, answering
"go-llmprovider-sdk 0052" to "Which repos' pairs do you approve as
recommended (MADR accepted, PLAN in-progress, then Phase 0/records and
Phase 1 per its plan)?". That includes the MADR's recommended answer to Q1,
(a): the docs job runs `make parity-check records-check tables-check
changelog-check gate-selftest`, so Phase 1 step 2 stands as written.

**Deviation (2026-10-20), from "Rules for every phase":** Phases 0, 1 and 2
were executed in one working tree, by the owner's instruction to the
executing agent, without staging: the rule that a phase starts after the
owner has committed the last, and "Stage, then the owner commits", do not
hold for them. Each phase's entry below lists the files it changed, so the
owner can stage and commit each phase on its own. No decision of the MADR
changes. Phase 3 needs pushes and is not started.

### Phase 0: records (2026-10-20)

Done:

* `tools/records.py status` set the MADR `accepted` and this PLAN
  `in-progress`, each with its row in `docs/README.md`.
* The MADR's Decision Outcome reads "Chosen option: "B"", with the owner's
  words and Q1's answer; its Q1 under "Owner questions" records "Answered
  2026-10-20: (a)". This PLAN's header note is annotated, not rewritten.

Files: `docs/README.md`,
`docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md`,
`docs/decisions/0052-PLAN-skip-full-ci-for-docs-only-pushes.md`.

Evidence, each exit status captured before any filter:

```text
$ python3 -B tools/records.py status docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md accepted
records: written; records-check: 0 problem(s)          (exit 0)
$ python3 -B tools/records.py status docs/decisions/0052-PLAN-skip-full-ci-for-docs-only-pushes.md in-progress
records: written; records-check: 0 problem(s)          (exit 0)
$ make records-check
records-check: 111 records, 0 problem(s)               (exit 0)
$ python3 -B tools/claims.py <the MADR> <this PLAN>
claims: docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md: 36 of 36 pass
claims: docs/decisions/0052-PLAN-skip-full-ci-for-docs-only-pushes.md: 12 of 12 pass   (exit 0)
$ python3 -B tools/linkcheck.py
G-links: 0 problem(s), 909 relative link(s) in 123 file(s)   (exit 0)
$ python3 -B tools/recordlint.py --changed
recordlint: 2 record(s); no rule rose                  (exit 0)
$ python3 -B tools/denyscan.py --changed
deny-scan: 0 hit(s) in 3 file(s)                       (exit 0)
```

### Phase 1: the filters and the docs workflow (2026-10-20)

Done:

* `ci.yml`: `on.push` gained the comment and `paths-ignore: ['**.md']`
  after `tags: ['v*']`, and `on.pull_request` gained `paths-ignore:
  ['**.md']`. `git diff` of the file shows those six added lines and
  nothing else; the `govulncheck` pin is untouched.
* `.github/workflows/docs.yml`, new, as step 2 gives it: a script that
  strips the block's three-space indent and compares it with the file
  printed "docs.yml matches the PLAN's block: 39 lines" (exit 0).

Files: `.github/workflows/ci.yml`, `.github/workflows/docs.yml`, and this
PLAN's entry.

Seen failing (step 3). `tools/plant.py` on a scratch clone of the working
tree (Phases 0 and 1, and nothing else, in it), its `command` the docs
job's `make` line run through a wrapper that keeps each run's whole output:

```text
== X1 records-check: 0051-PLAN's index row complete -> in-progress: caught (exit 2)
   make: *** [records-check] Error 1
== X2 tables-check: LLMPROVIDER_LIVE_CHATGPT_PLAN dropped from AGENTS.md: caught (exit 2)
   make: *** [tables-check] Error 1
== X3 parity-check: the first llmprovider. row's SDK cell -> TBD: caught (exit 2)
   G-parity: SDK equivalent of llmprovider.APIError names nothing that resolves, and is not one of none, removed, Here: TBD
   make: *** [parity-check] Error 1
== X4 changelog-check: v1.19.0's date 2026-10-20 -> 2026-10-19: caught (exit 2)
   make: *** [changelog-check] Error 1
== X5 gate-selftest: AGENTS.md's plant string reworded, the variable still named: caught (exit 2)
   FAIL: test_tables_check_live_variable_unnamed (scripts.test_gates.GateSelfTest.test_tables_check_live_variable_unnamed)
   raise AssertionError(f"{path.name}: {old!r} found {text.count(old)} times")
   AssertionError: AGENTS.md: '`LLMPROVIDER_LIVE_CHATGPT_PLAN` (the ChatGPT plan sign-in),' found 0 times
   make: *** [gate-selftest] Error 1
plant: all 5 plant(s) caught                                         (exit 0)
```

Each run's failing line, from the wrapper's logs:

* X1: `docs/README.md: decisions/0051-PLAN-opt-in-streaming-retry.md
  indexed as 'in-progress'; the file says 'complete'`, then
  `records-check: 111 records, 1 problem(s)`; `parity-check` before it
  passed.
* X2: `check-tables: AGENTS.md: LLMPROVIDER_LIVE_CHATGPT_PLAN: a live test
  reads it, and AGENTS.md does not name it`, then `check-tables: 38
  packages, 1 problem(s)`.
* X3: as quoted above, `G-parity: 409 identifiers, 409 rows, 409 with an
  SDK equivalent, 1 problem(s)`.
* X4: `changelog-check: line 13: v1.19.0 is dated 2026-10-19; its tag's
  commit is 2026-10-20`, then `changelog-check: 27 release section(s), 1
  problem(s)`.
* X5: every gate passed (`check-tables: 38 packages, 0 problem(s)`,
  `changelog-check: 27 release section(s), 0 problem(s)`), and only the
  self-test failed: `Ran 36 tests in 152.372s`, `FAILED (failures=1)`.
  This is the case that puts `gate-selftest` in the docs job.

The same command on an unplanted scratch clone of the same tree: exit 0,
`G-parity: 409 identifiers, 409 rows, 409 with an SDK equivalent, 0
problem(s)`, `records-check: 111 records, 0 problem(s)`, `check-tables: 38
packages, 0 problem(s)`, `changelog-check: 27 release section(s), 0
problem(s)`, `Ran 36 tests in 155.056s`, `OK`.

`actionlint` 1.7.12 on a scratch copy of `docs.yml` whose concurrency line
reads `github.event_nam` (exit 1):

```text
.github/workflows/docs.yml:19:27: property "event_nam" is not defined in object type {action: string; …; workspace: string} [expression]
   |
19 |   cancel-in-progress: ${{ github.event_nam == 'pull_request' }}
   |                           ^~~~~~~~~~~~~~~~
```

The type list is elided here (`…`). On the tree, `actionlint
.github/workflows/ci.yml .github/workflows/docs.yml` printed nothing,
exit 0.

Checks (step 4):

```text
$ grep -oE 'golang.org/x/vuln/cmd/govulncheck@v[0-9][0-9.]*' .github/workflows/ci.yml   (Makefile:10's pattern)
golang.org/x/vuln/cmd/govulncheck@v1.8.0                              (exit 0)
$ make -n vuln
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
$ python3 -B tools/gate.py
started at load 9.0
pre-add-check: rc=0 … tools-test: rc=0
gate: 27 of 27 passed                                                 (exit 0)
```

Every one of the gate's 27 lines read `rc=0`: pre-add-check, vet for
three platforms with and without `live_gateways`, test-race-cover,
test-shuffle, tidy-diff, lint, the eleven `make` gates, gate-selftest,
markdownlint (`Summary: 0 issues in 0 files`), g-wire-stable, g-links
(`0 problem(s), 912 relative link(s) in 125 file(s)`), deny-scan (`0
hit(s) in 7 file(s)`) and tools-test. The filters themselves are proven
only on GitHub, in Phase 3.

Note (2026-10-20): while this phase ran, another session added and staged
`docs/decisions/0053-MADR-stream-retry-and-cache-defects.md`,
`docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md` and
their two rows in `docs/README.md`. They are not this plan's. The plants
and the clean run above predate them (111 records); the gate's clone holds
them (`records-check: 113 records, 0 problem(s)`, and 7 changed files
scanned).

### Phase 2: the documents (2026-10-20)

Done:

* `docs/architecture.md`: the layout block gained
  `.github/workflows/docs.yml  Docs: the Markdown gates, for a Markdown-only
  change` after `ci.yml`'s line; the CI bullet gained a paragraph: a push
  to `main` or a pull request that changes only Markdown skips CI
  (`paths-ignore: ['**.md']`), and `docs.yml` runs instead, on one Linux
  runner with `ci.yml`'s full-history checkout and Go version, `make
  parity-check records-check tables-check changelog-check gate-selftest`;
  any other change runs CI, and Docs too when it also changes Markdown or
  `docs.yml`; a tag push and the weekly run are never filtered, so a
  Release still follows a full `validate`; it links the MADR.
* `AGENTS.md`: after the sentence that lists what CI runs, one sentence: a
  Markdown-only push or pull request runs `.github/workflows/docs.yml`
  instead, with its `make` line, and a tag always runs CI in full, citing
  the MADR by full filename in backticks.
* `CHANGELOG.md`: `## [Unreleased]` gained `### Changed`, one entry with
  the MADR linked.

Each claim checked against the two workflows: `paths-ignore: ['**.md']`
on `ci.yml`'s `push` and `pull_request`; `docs.yml`'s `paths` (`**.md` and
itself), its one `ubuntu-24.04` job, `fetch-depth: 0`, `go-version-file:
go.mod` and its `make` line; the `release` job and the schedule unchanged.

Files: `docs/architecture.md`, `AGENTS.md`, `CHANGELOG.md`, and this
PLAN's entry.

Checks, each exit status captured before any filter:

```text
$ make tables-check
check-tables: 38 packages, 0 problem(s)                               (exit 0)
$ make changelog-check
changelog-check: 27 release section(s), 0 problem(s)                  (exit 0)
$ python3 -B tools/linkcheck.py
G-links: 0 problem(s), 914 relative link(s) in 125 file(s)            (exit 0)
$ python3 -B tools/denyscan.py --changed
deny-scan: 0 hit(s) in 10 file(s)                                     (exit 0)
$ python3 -B tools/claims.py <the MADR> <this PLAN>
claims: …0052-MADR…: 36 of 36 pass; …0052-PLAN…: 12 of 12 pass         (exit 0)
$ python3 -B tools/gate.py
started at load 6.6
gate: 27 of 27 passed                                                 (exit 0)
```

The gate's 27 lines all read `rc=0`; among them markdownlint (`Summary: 0
issues in 0 files`), `gate-selftest` (`OK`), so no plant string in
`AGENTS.md`, `docs/architecture.md` or `CHANGELOG.md` was reworded, and
`records-check: 113 records, 0 problem(s)`. `npx --yes
markdownlint-cli2@0.23.2 AGENTS.md CHANGELOG.md docs/architecture.md` found
nothing in those three files; its exit 1 came from the configuration's own
globs, which also took in `docs/reports/0011-REPORT-…` (unchanged at
`HEAD`) and the other session's `docs/reports/0054-REPORT-…`, neither this
plan's. This commit changes Markdown only, so its push is Phase 3's
docs-only test.

Not done: Phase 3 (the live checks and closing), which needs the owner's
pushes. The plan stays `in-progress`.

### Deviation (2026-10-20): the docs job failed on its runs

* **Found** while `0053-PLAN-stream-retry-and-cache-defects.md`'s Phase 3
  was being pushed: "Docs" failed on both of its runs, 39498970113 on
  `882272f` and 39521616742 on `5c60593`. In each, the four Markdown gates
  passed, and `gate-selftest` failed
  `PrecheckTest.test_precheck_deleted_file_checks_its_package` (`2 != 1`,
  "go-precheck: golangci-lint not found at …"): `Ran 36 tests`,
  `FAILED (failures=1)`. "CI" passed on `882272f`. The cause and the
  options are in the MADR's "Amendment 2026-10-20".
* **Why Phase 1 did not see it.** X1 to X5 and the clean run ran on this
  host, where golangci-lint is installed.
* **Neither push tested Phase 3's docs-only case.** The push of `882272f`
  also carried `90c69a6` (the workflows), and that of `5c60593` Go, so
  "CI" ran for both. Phase 3 step 2 still needs a docs-only push.
* **The decision:** the owner's, "fix it option 1 + check": the MADR's A1
  and A2, executed as Phase 4. Files added: `.github/workflows/docs.yml`,
  `scripts/test_gates.py`, `docs/architecture.md`, `CHANGELOG.md`.

### Phase 4: the docs job's golangci-lint (2026-10-20)

Done:

* `docs.yml`: after `go mod download`, a comment of four lines and `- run:
  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`.
* `scripts/test_gates.py`: `SELFTEST_STEP`, `GO_INSTALL`,
  `installs_before_selftest` and `WorkflowsTest`, before the module's
  `__main__` block.
* `docs/architecture.md`'s Docs paragraph and `CHANGELOG.md`'s
  `[Unreleased]` Fixed entry, as step 3 gives them.

Seen failing (step 4), each exit status captured before any filter:

```text
$ GOLANGCI_LINT=<scratch>/no-such-golangci-lint python3 -B -m unittest scripts.test_gates.PrecheckTest
AssertionError: 2 != 1 : go-precheck exit 2, want 1:
go-precheck: golangci-lint not found at <scratch>/no-such-golangci-lint.
FAILED (failures=1)                                                   (exit 1)
$ python3 -B -m unittest scripts.test_gates.PrecheckTest
OK                                                                    (exit 0)
$ python3 -B tools/plant.py <spec>     (command: python3 -B -m unittest scripts.test_gates.WorkflowsTest)
== docs.yml's install removed: caught (exit 1)
   AssertionError: Lists differ: ['github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0'] != []
== docs.yml's install at another version: caught (exit 1)
   AssertionError: Lists differ: ['github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0'] != []
== docs.yml's install moved after the make step: caught (exit 1)
   AssertionError: Lists differ: ['github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0'] != []
== ci.yml's pin raised alone: caught (exit 1)
   AssertionError: Lists differ: ['github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0'] != []
== docs.yml's make line without gate-selftest: caught (exit 1)
   AssertionError: unexpectedly None : docs.yml has no `make … gate-selftest` line
== ci.yml installs nothing before gate-selftest: caught (exit 1)
   AssertionError: [] is not true : ci.yml installs nothing before gate-selftest; the precheck test needs golangci-lint
plant: all 6 plant(s) caught                                          (exit 0)
$ python3 -B -m unittest scripts.test_gates.WorkflowsTest
OK                                                                    (exit 0)
```

The first line reproduces the runs' failure on this host; the scratch
path is shortened to `<scratch>`.

Checks (step 5):

```text
$ actionlint .github/workflows/ci.yml .github/workflows/docs.yml
(no output)                                                           (exit 0)
$ python3 -B tools/gate.py
gate: 27 of 27 passed                                                 (exit 0)
$ python3 -B tools/claims.py <the MADR> <this PLAN>
claims: …0052-MADR…: 36 of 36 pass; …0052-PLAN…: 12 of 12 pass         (exit 0)
$ make records-check changelog-check tables-check
records-check: 114 records, 0 problem(s)
changelog-check: 28 release section(s), 0 problem(s)
check-tables: 38 packages, 0 problem(s)                               (exit 0)
$ python3 -B tools/recordlint.py --changed
recordlint: 2 record(s); no rule rose                                 (exit 0)
$ python3 -B tools/linkcheck.py
G-links: 0 problem(s), 920 relative link(s) in 126 file(s)            (exit 0)
$ python3 -B tools/denyscan.py --changed
deny-scan: 0 hit(s) in 6 file(s)                                      (exit 0)
```

The first identifier scan found one hit, "a machine home path": the
runner's home directory, in the MADR's quotation of the run's log. It is
replaced by `<runner home>`, which the MADR says.

Step 6, the live run, follows the owner's push.
