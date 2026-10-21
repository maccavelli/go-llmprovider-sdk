---
status: accepted
date: 2026-10-20
decision-makers: repository owner
consulted: "go-selfupdate-lib's docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md and its PLAN, at be32614; GitHub's workflow syntax reference (path filters), as that record quotes it; the Actions timing and jobs APIs for runs 39439485755, 39358411992 and 39476389239; the rulesets and branch-protection APIs; this repository at cadaad5"
informed: anyone who pushes to this repository or opens a pull request on it
---

<!-- markdownlint-disable MD013 -->

# Skip the Full CI for a Push or Pull Request That Changes Only Markdown, and Check It With a One-Job Docs Workflow

## Context and Problem Statement

Every push to `main`, every pull request and every `v*` tag runs all of
`.github/workflows/ci.yml` (C1 to C3): a `validate` job on Linux, macOS and
Windows (C6), and on a tag the `release` job after it (C10, C11). A commit
that changes only Markdown gets the same run: a record, a phase closing, an
index row, or a release's changelog section. This repository writes many
such commits, since every decision is a MADR and a PLAN.

The owner asked, on 2026-10-20, for this repository to stop running its
full CI for pushes and pull requests that change only Markdown, as
go-selfupdate-lib has just done in
`docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md` (that
repository's record, cited by repository and filename). That record chose
its option B: `paths-ignore: ['**.md']` on `ci.yml`'s `push` and
`pull_request` triggers (G1), and a one-job `docs.yml` with `paths:` (G2)
that runs, with the same commands, the Markdown checks `ci.yml` runs. It is
implemented and live there: a docs-only push ran its Docs job in 12 s,
where the full CI is 18 jobs and 15 to 20 minutes (G5).

The question is how to do the same here, given what this repository's CI
checks and what in it reads Markdown.

### Facts (observed 2026-10-20)

* **The triggers** (C1 to C5). `push` to `main` and to `v*` tags;
  `pull_request`, with no branch filter; a weekly `schedule`, Mondays at
  06:17 UTC. There is no `workflow_dispatch`, and `ci.yml` is the only
  workflow. A newer push cancels an older run only on a pull request: each
  push to `main` runs to the end (C5).
* **The jobs.** `validate` is a three-OS matrix, `ubuntu-24.04`, `macos-15`
  and `windows-2025` (C6), each running `go test -shuffle=on`; the Linux leg
  also runs the race detector, `govulncheck`, vet, `gofmt`, tidy, lint, the
  live-tagged vet, and one `make` step of eleven gates (C9). It checks out
  the full history (C7) and reads the Go version from `go.mod` (C8).
  `release` needs `validate` and runs only on a tag push (C10, C11). A run is
  four jobs, `release` skipped on a branch.
* **Minutes.** The repository is public. The timing API reports 0 billable
  ms for every job run of three recent runs: `39439485755` and
  `39358411992`, two docs-only pushes to `main`, and `39476389239`, the
  `v1.19.0` tag; each had two Ubuntu, one macOS and one Windows job run.
  This CI bills no minutes, as in go-selfupdate-lib (G4).
* **What a docs-only run costs anyway.** `39439485755` changed
  `docs/README.md` and one PLAN, and took 7 min 37 s; `39358411992`, the
  same two kinds of file, 7 min 45 s. The Linux leg is the whole wait (7 min
  32 s and 7 min 42 s); macOS and Windows finished in about 1 min 25 s.
  The eleven-gate `make` step alone took 2 min 48 s and 2 min 54 s.
* **How often.** From 2026-10-13 to 2026-10-20, CI ran 36 times: 25 pushes
  to `main`, 10 tag pushes and one scheduled run; no pull request. Of the 25
  pushes to `main`, 13 had a Markdown-only head commit; of 82 commits in the
  same week, 32 changed only `.md` files. A push is filtered only when all
  of its commits are Markdown-only, so up to 13 of the 25 runs would have
  been Docs runs.
* **The release-prep commit is docs-only.** `87b6c9c` ("prepare v1.19.0")
  and `d19f7a3` ("prepare v1.18.0") each changed `CHANGELOG.md`,
  `README.md` and a PLAN, nothing else. The owner pushed `87b6c9c` to
  `main`, waited for its full run (`39474947857`), and then pushed the tag,
  whose full run (`39476389239`) made the Release.
* **CI runs no Markdown lint and no link check.** `tools/gate.py` runs
  markdownlint and the link check locally; CI runs neither (C26). So unlike
  go-selfupdate-lib, there is no markdownlint, link or table step to move.
* **Four of CI's gates read Markdown,** all in the one `make` step (C9):
  * `parity-check` reads `docs/guides/migrating-from-mcplib.md` (C12), and
    resolves each cell against the module with `go list` and `go doc`
    (C13);
  * `records-check` reads every record and `docs/README.md` (C14);
  * `tables-check` reads `docs/architecture.md`,
    `docs/guides/api-standards.md` and `AGENTS.md` (C15), against `go list`
    (C16);
  * `changelog-check` reads `CHANGELOG.md` (C17) against the release tags,
    which the full-history checkout fetches (C7).

  A Markdown-only change can break any of the four. Two of them need Go and
  the module's dependencies, and one needs every tag.
* **`gate-selftest` depends on Markdown text.** `scripts/test_gates.py`
  plants a breach in a copy of the tree, and for these four gates it plants
  it by replacing an exact string, once, in `AGENTS.md`,
  `docs/architecture.md`, `docs/guides/api-standards.md`, `CHANGELOG.md`,
  `docs/guides/migrating-from-mcplib.md` and
  `docs/decisions/0024-PLAN-opencode-live-system-message-test.md` (C18 to
  C22). A Markdown edit that rewords such a string breaks `gate-selftest`
  while every gate still passes. Its clean test also runs every gate on an
  unplanted copy (C23), so it covers `fixtures-check` and the others too.
  Locally, the eighteen tests of the four gates took 25 s, and the whole
  self-test, 36 tests, 134 s; the four gates themselves take about 3 s,
  and the ten other gates under a minute (the first six of them, through
  `records-check`, took 37 s). On CI the whole eleven-gate step took 2 min
  48 s.
* **No Go code reads a tracked Markdown file.** The one `//go:embed` names
  `llmprovider/providers/opencode/routes_snapshot.json`; no `.go` file names
  a `.md` file; no file under a `testdata` directory is Markdown. The
  other six gates (`dep-check`, `coverage-check`, `api-check`,
  `generate-check`, `fixtures-check`, `docs-check`) and the Go steps read
  Go, `go.mod` and the fixtures, which a Markdown-only change does not
  touch.
  `docs/guides/migrating-from-mcplib.ids`, which `parity-check` reads, is
  not Markdown: a change to it keeps the full run.
* **Two scripts read `ci.yml`.** `Makefile` and `scripts/go-precheck.sh`
  take the `govulncheck` pin from `ci.yml`, the first match (C24, C25). The
  pin must stay in `ci.yml`.
* **No rule requires a check.** `rulesets` is `[]`; `branches/main/protection`
  is 404 "Branch not protected"; the rules that apply to `main` are `[]`.
* **GitHub's path filters,** as go-selfupdate-lib's record quotes the
  workflow syntax reference (read 2026-10-20):
  * "Path filters are not evaluated for pushes of tags." (G3) A `v*` tag
    keeps the full run, and so the `release` job.
  * Branch and path filters must both be satisfied; a push's changed files
    are the diff between its before and after commits.
  * A push of more than 1,000 commits, or a diff that times out, always
    runs.
  * A workflow skipped by a path filter leaves its checks "Pending", which
    blocks a pull request that requires them (G6). None is required here.
  * The weekly schedule is not a push or a pull request, and is not
    filtered.

## Decision Drivers

* A docs-only commit keeps every check that a Markdown change can break:
  the four Markdown-reading gates, and `gate-selftest`'s plants.
* A commit that changes anything else, every tag, and the weekly run keep
  the full CI, and the `release` job stays behind a full `validate`.
* GitHub's own filters, no third-party action, and nothing that hides a
  failure.
* The docs job runs `ci.yml`'s own commands, so the two cannot disagree on
  what a gate is.
* Simple enough that a later change cannot quietly disable CI for code.

## Considered Options

* **A.** `paths-ignore: ['**.md']` on `ci.yml`'s `push` and `pull_request`
  triggers, and nothing else.
* **B.** As A, plus a new `.github/workflows/docs.yml` with `paths: ['**.md',
  '.github/workflows/docs.yml']` and one Linux job that runs, with
  `ci.yml`'s commands, the gates a Markdown change can break.
* **C.** One workflow: a first job decides from `git diff --name-only`
  whether anything but Markdown changed, and `validate` carries an `if:` on
  its output, with the docs gates in a job that always runs.
* **D.** `[skip ci]` in the message of a docs-only commit.

## Decision Outcome

Chosen option: "B", because it keeps the gates that a Markdown change can
break at the cost of one small workflow, uses GitHub's own filters, and
leaves `ci.yml`'s jobs and the `release` job untouched. Tags and the weekly
run are never filtered.

The owner chose it on 2026-10-20, as recommended, answering "go-llmprovider-sdk
0052" to "Which repos' pairs do you approve as recommended (MADR accepted,
PLAN in-progress, then Phase 0/records and Phase 1 per its plan)?". That
answers Q1 with its recommended (a): the docs job runs `make parity-check
records-check tables-check changelog-check gate-selftest` (D2).

### D1. `ci.yml`'s filters

`on.push` gains `paths-ignore: ['**.md']` beside `branches` and `tags`, and
`on.pull_request` the same, with a comment that cites this record and says
that a tag push is never filtered. Nothing else in `ci.yml` changes, the
`govulncheck` pin included (C24, C25).

### D2. `docs.yml`

A new workflow, named "Docs":

* `push` to `main` and `pull_request`, each with `paths: ['**.md',
  '.github/workflows/docs.yml']`. `.markdownlint-cli2.jsonc` is not named,
  since no CI step reads it; a change to it keeps the full run.
* `permissions: contents: read`, and the concurrency rule of `ci.yml` in
  its own group: `docs-${{ github.ref }}`, cancelled only on a pull request.
* One `ubuntu-24.04` job with `timeout-minutes: 15`. Its steps are
  `ci.yml`'s, with the same pinned SHAs: checkout with `fetch-depth: 0`
  (for `changelog-check`'s tags), setup-go with `go-version-file: go.mod`,
  `go mod download`, and then
  `make parity-check records-check tables-check changelog-check
  gate-selftest`: the four Markdown-reading gates of C9 and its self-test,
  by the same targets. The checkout adds `persist-credentials: false`,
  since the job never pushes.
* A header comment says what it is for and cites this record.

### D3. The documents

`docs/architecture.md`'s layout block (C28) and CI bullet (C27) name
`docs.yml`, the Markdown-only rule, and that a tag and the weekly run always
run the full CI. `AGENTS.md`, where it lists what CI runs (C29), gains one
sentence. `CHANGELOG.md`'s `## [Unreleased]` (C30) gains a "Changed" entry.

### Consequences

* Good, because a docs-only push to `main` waits for one Linux job of
  about two to three minutes (setup, and at most the 2 min 48 s that the
  whole `make` step took on CI), instead of 7 min 30 s, and no macOS or
  Windows runner is queued.
* Good, because every gate that reads Markdown still runs on a docs-only
  commit, with `ci.yml`'s targets, and so does `gate-selftest`, which a
  reworded string would otherwise break unseen until the next code push.
* Good, because a code push, every tag and the weekly run are as before,
  and the `release` job still needs a full `validate` on the tag.
* Neutral, because a push that changes code and Markdown runs both
  workflows, and the four gates and the self-test run twice.
* Neutral, because the release-prep commit, being docs-only, runs only
  "Docs" on `main`. The tag's full run is then the code proof, and its
  `release` job does not start unless `validate` passes. The code is that
  of the last commit whose full run passed, since a docs-only commit
  changes none; what can differ is the world (a new advisory, a runner
  image), which the tag's run, and the weekly run, still catch.
* Bad, because a required status check, if a ruleset ever requires one,
  must not name a `ci.yml` job: a docs-only pull request would leave it
  "Pending". The record that adds the ruleset must say so.
* Bad, because there are two workflows to keep in step: a new gate that
  reads Markdown must be added to both.

### Confirmation

* `actionlint` passes on both workflows, and fails on a scratch copy of
  `docs.yml` with a planted expression error.
* The docs job's `make` command fails on a scratch clone for each planted
  break: a records index row, a live variable dropped from `AGENTS.md`, a
  parity cell, a changelog date, and a reworded `AGENTS.md` string that
  passes `tables-check` and fails `gate-selftest`. It passes on the tree.
* After the change lands, a docs-only push to `main` shows a "Docs" run and
  no "CI" run, and a push that changes a non-Markdown file shows "CI". The
  PLAN records both run ids.
* The next tag push shows a full "CI" run and its Release; that release's
  PLAN records it.

## Amendment 2026-10-20: the docs job installs golangci-lint

The owner decided this on 2026-10-20 ("fix it option 1 + check"), after
the docs job had failed on each of its first two runs. D2 is not rewritten;
A1 adds a step to it, and A2 a test.

### What the runs showed

* **Run 39498970113**, on `882272f`, and **run 39521616742**, on
  `5c60593`: "Docs", `failure`. In both, `parity-check`, `records-check`,
  `tables-check` and `changelog-check` passed, and `gate-selftest` failed
  one test of 36:

  ```text
  FAIL: test_precheck_deleted_file_checks_its_package (scripts.test_gates.PrecheckTest.test_precheck_deleted_file_checks_its_package)
  AssertionError: 2 != 1 : go-precheck exit 2, want 1:
  go-precheck: golangci-lint not found at <runner home>/go/bin/golangci-lint.
  ```

  The runner's home directory is replaced by `<runner home>`.

  "CI" passed on `882272f`, its `gate-selftest` included.
* **The cause.** That test runs `scripts/go-precheck.sh`, which looks for
  golangci-lint at `$(go env GOPATH)/bin/golangci-lint` and exits 2 when it
  is not there (`scripts/go-precheck.sh`, its step 2). `ci.yml` installs it,
  at `v2.14.0`, in its "vet, gofmt, tidy, lint" step, before its `make`
  step. D2 gave the docs job `ci.yml`'s checkout, setup-go, `go mod
  download` and `make` targets, but not that install.
* **Why the Confirmation did not see it.** Its plants and its clean run ran
  on this host, where golangci-lint is installed, so the precheck test
  passed there. The defect shows only on a runner without it.

### A1. The docs job installs golangci-lint at `ci.yml`'s pin

After `go mod download` and before the `make` step, `docs.yml` runs `go
install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`,
the line `ci.yml` runs, with a comment that says why. `gate-selftest` then
runs in full, as D2 meant it to.

### A2. A test keeps the two workflows' installs in step

`scripts/test_gates.py` gains `WorkflowsTest`: every `go install` target
that `ci.yml` runs before its `make … gate-selftest` line must appear in
`docs.yml` before its own `make … gate-selftest` line, at the same version.
It runs under `gate-selftest`, so both workflows run it. It fails when the
docs job lacks one of `ci.yml`'s installs, has it at another version, or
has it after the `make` step, and when either workflow has no `make …
gate-selftest` line or `ci.yml` none of its installs.

This narrows the Consequence "there are two workflows to keep in step": a
tool the self-test needs is now checked. A new gate in one `make` line and
not the other still is not.

### Options considered

* **A1 and A2** (chosen): the docs job runs what the self-test needs, and
  a test says so.
* **A1 alone:** the same fix, with nothing to catch the next tool `ci.yml`
  adds before its self-test.
* **The precheck test skipped when golangci-lint is missing:** the docs job
  would pass, but a missing tool would then hide the precheck's own
  failure, on any runner. Rejected: it removes a check to make a job pass.
* **The docs job running `ci.yml`'s whole "vet, gofmt, tidy, lint" step:**
  it would lint Go on a Markdown-only change, which is what D1 removes.

### Consequences of A1 and A2

* Good, because the docs job's `gate-selftest` passes on a runner, as on
  this host, and fails only on what it checks.
* Bad, because each docs run also builds golangci-lint; the time is
  measured on the first run and recorded in the PLAN.
* Neutral, because the pin is now in three places: `ci.yml`, `docs.yml`,
  and `scripts/go-precheck.sh`'s install hint. A2 checks the first two.

### Confirmation of A1 and A2

* The precheck test, run on this host with `GOLANGCI_LINT` naming a missing
  file, fails as the runs did.
* `tools/plant.py` on a scratch clone: A2's test fails for each planted
  break, and passes on the tree.
* After the push, "Docs" passes; the PLAN records its run id and time.

## Pros and Cons of the Options

### A. `paths-ignore` only

* Good, because it is one line per trigger.
* Bad, because a docs-only commit then gets no `records-check`,
  `tables-check`, `parity-check`, `changelog-check` or `gate-selftest`. A
  broken index row, a stale table or a changelog date reaches `main`
  unchecked, and the next code push fails on a cause it did not make.

### B. `paths-ignore` and a docs workflow

* Good, because the Markdown-reading gates keep running, and `ci.yml`'s jobs
  are unchanged.
* Good, because both filters are GitHub's own, documented behaviour, and
  the model is live in go-selfupdate-lib.
* Neutral, because the docs job needs Go, the module's dependencies and the
  full history, unlike go-selfupdate-lib's, which needs none.
* Bad, because there are two workflows to keep in step.

### C. One workflow, gated jobs

* Good, because one file holds it all, and a job skipped by `if:` reports
  "skipped", which a required check accepts, where a filtered workflow
  leaves "Pending".
* Good, because here only one job, `validate`, would need the `if:`.
* Bad, because a wrong condition skips CI for code with nothing to show it,
  and `release`, which needs `validate`, would then be skipped on a tag
  unless the condition also exempts tags.
* Bad, because the change-detection job runs on every push.

### D. `[skip ci]`

* Bad, because the global `prepare-commit-msg` hook writes the commit
  messages, so a person would have to add it each time; and it skips the
  Markdown-reading gates too.

## Owner questions

* **Q1 (D2). Which gates the docs job runs.** Offered:
  * **(a)** `make parity-check records-check tables-check changelog-check
    gate-selftest`: the four gates that read Markdown, and the whole
    self-test, by `ci.yml`'s targets (recommended). The self-test is
    most of its time (134 s locally); since its clean test runs the ten
    other gates on the tree (C23), it already shows that every gate
    passes.
  * **(b)** `ci.yml`'s whole `make` line (C9), verbatim: one command to
    keep identical, and the cost of all eleven gates, the 2 min 48 s
    measured; it adds the six other gates' direct runs, under a minute,
    for their own failure messages, and no coverage that (a) lacks.
  * **(c)** The four gates, and only their self-tests:
    `python3 -B -m unittest scripts/test_gates.py -k parity_check -k
    records_check -k tables_check -k changelog`, 25 s locally. It is a
    command `ci.yml` does not run, and a later self-test that plants into
    Markdown under another name would be missed.

  **Answered 2026-10-20: (a),** as recommended, with the owner's approval
  of this pair (Decision Outcome).

## More Information

* The PLAN:
  [0052-PLAN-skip-full-ci-for-docs-only-pushes.md](0052-PLAN-skip-full-ci-for-docs-only-pushes.md).
* The model: go-selfupdate-lib's
  `docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md` and
  `docs/decisions/0028-PLAN-skip-full-ci-for-docs-only-pushes.md`. Its facts
  relied on here are restated above and pinned below (G1 to G6).
* How this repository differs from the model: CI runs no markdownlint or
  link check, so the docs job runs the Python gates instead, which need Go
  and every tag; `gate-selftest` depends on Markdown text; there are four
  jobs and about 7 min 30 s, not 18 and 15 to 20 minutes, so the saving is
  smaller; there is no `release/**` branch, no workflow-contract script and
  no `actionlint` in CI; and the concurrency rule cancels only on a pull
  request.
* The CI being changed is described in `docs/architecture.md`'s CI bullet
  under "Tooling" (C27), and the release rule in the standards guide's R53.

### Code claims

<!-- claims: self=cadaad5 go-selfupdate-lib=be32614 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | CI runs on a push to `main` | `.github/workflows/ci.yml:4` | `branches: [main]` |
| C2 | and on a `v*` tag | `.github/workflows/ci.yml:5` | `tags: ['v*']` |
| C3 | and on every pull request | `.github/workflows/ci.yml:6` | `pull_request:` |
| C4 | and weekly | `.github/workflows/ci.yml:10` | `- cron: '17 6 * * 1'` |
| C5 | A newer push cancels only on a pull request | `.github/workflows/ci.yml:15` | `cancel-in-progress: ${{ github.event_name == 'pull_request' }}` |
| C6 | `validate` is a three-OS matrix | `.github/workflows/ci.yml:26` | `os: [ubuntu-24.04, macos-15, windows-2025]` |
| C7 | It checks out every tag | `.github/workflows/ci.yml:33` | `fetch-depth: 0` |
| C8 | Go from `go.mod` | `.github/workflows/ci.yml:36` | `go-version-file: go.mod` |
| C9 | The eleven gates | `.github/workflows/ci.yml:60` | `run: make parity-check dep-check coverage-check api-check generate-check records-check tables-check fixtures-check changelog-check docs-check gate-selftest` |
| C10 | `release` needs `validate` | `.github/workflows/ci.yml:66` | `needs: validate` |
| C11 | `release` only on a tag push | `.github/workflows/ci.yml:67` | `if: github.event_name == 'push' && startsWith(github.ref, 'refs/tags/v')` |
| C12 | `parity-check` reads the migration guide | `scripts/check_parity_map.py:32` | `GUIDE = ROOT / "docs/guides/migrating-from-mcplib.md"` |
| C13 | and runs `go doc` | `scripts/check_parity_map.py:135` | `doc = subprocess.run(["go", "doc", "-all", path]` |
| C14 | `records-check` reads the index | `scripts/check_records.py:45` | `INDEX = "docs/README.md"` |
| C15 | `tables-check` reads `AGENTS.md` | `scripts/check_tables.py:44` | `AGENTS = "AGENTS.md"` |
| C16 | and runs `go list` | `scripts/check_tables.py:64` | `proc = subprocess.run(["go", "list", "-f"` |
| C17 | `changelog-check` reads `CHANGELOG.md` | `scripts/check_changelog.py:42` | `CHANGELOG = ROOT / "CHANGELOG.md"` |
| C18 | The self-test plants in `AGENTS.md` by an exact string | `scripts/test_gates.py:269` | `replace_once(self.tree / "AGENTS.md"` |
| C19 | and in `docs/architecture.md` | `scripts/test_gates.py:249` | `replace_once(self.tree / "docs/architecture.md"` |
| C20 | and in `CHANGELOG.md` | `scripts/test_gates.py:342` | `log = self.tree / "CHANGELOG.md"` |
| C21 | and in the migration guide | `scripts/test_gates.py:189` | `guide = self.tree / "docs/guides/migrating-from-mcplib.md"` |
| C22 | and in a PLAN | `scripts/test_gates.py:232` | `plan = self.tree / "docs/decisions/0024-PLAN-opencode-live-system-message-test.md"` |
| C23 | Its clean test runs every gate | `scripts/test_gates.py:405` | `def test_clean_copy_passes_every_gate(self) -> None:` |
| C24 | `make vuln` reads the pin from `ci.yml` | `Makefile:10` | `GOVULNCHECK_PIN := $(shell grep -oE 'golang.org/x/vuln/cmd/govulncheck@v[0-9][0-9.]*' .github/workflows/ci.yml` |
| C25 | and so does the pre-add check | `scripts/go-precheck.sh:161` | `vuln_pin="$(grep -oE 'golang.org/x/vuln/cmd/govulncheck@v[0-9][0-9.]*' .github/workflows/ci.yml` |
| C26 | CI runs neither markdownlint nor the link check | `docs/architecture.md:918` | `fails on a planted input. CI runs neither.` |
| C27 | The architecture's CI bullet | `docs/architecture.md:922` | `` - **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with `` |
| C28 | Its layout block names `ci.yml` | `docs/architecture.md:29` | `.github/workflows/ci.yml    CI` |
| C29 | AGENTS.md lists what CI runs | `AGENTS.md:148` | `` CI also runs `make parity-check dep-check coverage-check api-check `` |
| C30 | The changelog's Unreleased section | `CHANGELOG.md:11` | `## [Unreleased]` |
| G1 | The model's filter | `go-selfupdate-lib:.github/workflows/ci.yml:11` | `paths-ignore: ['**.md']` |
| G2 | The model's docs workflow filter | `go-selfupdate-lib:.github/workflows/docs.yml:11` | `paths: ['**.md', '.markdownlint-cli2.jsonc', '.github/workflows/docs.yml']` |
| G3 | Tags are not filtered | `go-selfupdate-lib:docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md:45` | `"Path filters are not evaluated for pushes of tags."` |
| G4 | The model's CI billed nothing | `go-selfupdate-lib:docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md:26` | `billable ms for every job of run` |
| G5 | Its docs-only push ran Docs in 12 s | `go-selfupdate-lib:docs/decisions/0028-PLAN-skip-full-ci-for-docs-only-pushes.md:223` | `` `success`, in 12 s. `` |
| G6 | A filtered workflow leaves its checks Pending | `go-selfupdate-lib:docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md:53` | `A workflow skipped by a path filter leaves its checks "Pending", which` |
