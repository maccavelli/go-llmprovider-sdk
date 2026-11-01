---
status: proposed
date: 2026-11-01
associated-madr: "0067-MADR-keep-four-more-scratch-helpers-as-tools.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Keeping Four More Scratch Helpers: Red and Green Runs of Named Tests, a Leak Finder, Open Records, and a Markdown Gate

Associated MADR: [0067-MADR-keep-four-more-scratch-helpers-as-tools.md](0067-MADR-keep-four-more-scratch-helpers-as-tools.md)

## Goal

* **D1:** `tools/redgreen.py` runs named tests at a revision and requires
  each to fail, or each to pass.
* **D2:** `tools/leakfind.py` names the tests behind a `localonly` leak,
  on a scratch clone or on a dev host's checkout.
* **D3:** `tools/records.py open` lists the records still open.
* **D4:** `tools/gate.py --markdown` runs the checks a Markdown-only change
  needs.
* **D5:** `tools/README.md`, `AGENTS.md` and `docs/architecture.md`
  describe them.

## Scope

### In scope

* New: `tools/redgreen.py`, `tools/leakfind.py`.
* Changed: `tools/records.py`, `tools/gate.py`, `tools/test_tools.py`,
  `tools/README.md`, `AGENTS.md`, `docs/architecture.md`,
  `docs/README.md`.

### Out of scope

* Any Go file: `internal/localonly` is read, never changed (MADR option D).
* CI: the tools are not run by CI (0035-MADR).
* The 43 scratch scripts the MADR does not keep (its classes A to C).
* CHANGELOG: no tool has had an entry (0047-PLAN made none).

## Preconditions and conventions

* **Approval.** The owner answers Q1–Q3, 0067-MADR becomes `accepted`, and
  the owner approves this PLAN, whole or by phase.
* **Gate, at the end of each phase:**
  * `tools/gate.py` (its `tools-test` runs the new tests);
  * `tools/denyscan.py --changed`, `make records-check`;
  * `tools/claims.py` on this PLAN, `tools/recordlint.py --changed`.

  The phase is then staged for the owner's `git commit --no-edit`.
* **Red first.** Every new tool, mode and test is seen to fail on a planted
  break, by `tools/plant.py` with `make tools-test` as the command, on a
  scratch clone, never in the tree.
* **No identifier.** No test, spec or record names a dev host, an account
  or a machine path; `--ssh`'s alias and directory are arguments, and the
  tests use made-up ones.
* **Stop conditions:**
  * a fact of 0067-MADR, or a claim below, no longer holds;
  * a step would change a Go file in the tree;
  * a first use disagrees with the record it repeats.

## Code claims

<!-- claims: self=52241de -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The guard's message, which the leak finder reads | `internal/localonly/localonly.go:114` | `return "localonly: the tests sent requests off the machine, to " + strings.Join(hosts, ", ") +` |
| C2 | The gate's check list | `tools/gate.py:70` | `CHECKS: list[Check] = [` |
| C3 | The gate's last option today | `tools/gate.py:109` | `parser.add_argument("--timeout", type=float, default=25 * 60, help="seconds to wait for the load to fall")` |
| C4 | The Markdown gates CI runs for a Markdown-only change | `scripts/test_gates.py:533` | `MARKDOWN_GATES = ("parity-check", "records-check", "tables-check", "changelog-check", "markdownlint", "links-check",` |
| C5 | The MADR statuses | `scripts/check_records.py:42` | `MADR_STATUSES = ("proposed", "accepted", "rejected", "deprecated")` |
| C6 | The PLAN statuses | `scripts/check_records.py:43` | `PLAN_STATUSES = ("proposed", "in-progress", "complete", "superseded")` |
| C7 | `records.py`'s last subcommand today | `tools/records.py:166` | `p_status = sub.add_parser("status", help="set a record's status")` |
| C8 | The scratch clone the tools share | `tools/_common.py:78` | `def scratch_clone(src: Path, dest: Path) -> list[str]:` |
| C9 | The test that pins the gate's checks | `tools/test_tools.py:393` | `def test_the_checks_and_their_order(self) -> None:` |
| C10 | `tools/README.md`'s gate row | `tools/README.md:16` | `` \| `gate.py` \| the local gate: 28 checks `` |
| C11 | `tools/README.md`'s `records.py` row | `tools/README.md:22` | `` \| `records.py` \| a record's status and date `` |
| C12 | AGENTS.md's tools list, at `records.py` | `AGENTS.md:188` | `` - `tools/records.py add RECORD...` and `status RECORD STATUS` change a `` |
| C13 | The architecture's tree line for `tools/` | `docs/architecture.md:45` | `tools/                      maintainer tools, not run by CI: the local gate,` |
| C14 | The architecture's `tools/` paragraph | `docs/architecture.md:1146` | `` - **`tools/`** (0035-MADR) holds the maintainer tools, in Python on the `` |

## Implementation Steps

### Phase 0: records

1. 0067-MADR records the owner's answers to Q1–Q3 and becomes `accepted`;
   both records are indexed by `tools/records.py add`. `make
   records-check`.

### Phase 1: named tests, red or green (D1)

1. **`tools/redgreen.py`,** in `plant.py`'s shape: a docstring that is its
   definition, citing 0067-MADR D1; `load_spec`, `prepare` (the clone,
   `files`, `drop`), `results(output, names)` returning each name's
   `pass`, `fail` or `missing`, and `main`.
   * The spec is checked first: `tests` a non-empty map of package to
     names, each name `^(Test|Example)\w+$`; `files` and `drop` only with
     `--rev`; a bad spec is exit 2, nothing run.
   * `--rev`: `git clone --quiet --shared ROOT DEST`, `git checkout
     --quiet REV`, then the copies and drops. A drop removes `^func NAME(`
     through the next `^}` line, with the `//` lines directly above it; no
     match, or two, is exit 2.
   * Without `--rev`: `_common.scratch_clone` (C8).
   * A package whose output holds `[build failed]` or `[setup failed]` is
     a finding, each of its names reported `missing (did not build)`.
2. **Tests,** `TestRedGreen` in `tools/test_tools.py`, on a temporary Go
   module (skipped, with a reason, when `go` is not on `PATH`): a module
   with one passing and one failing test; `--expect pass` on the passing
   one, exit 0; on both, exit 1, naming the failing one; `--expect fail`
   the converse; a mistyped name, exit 1, `missing`; a module that does
   not build, exit 1, `did not build`; `--rev` at the module's first
   commit, with a test file from the tree and a `drop`, red as expected;
   a `drop` that matches nothing, exit 2; `files` without `--rev`, exit 2.
   `results` alone, on recorded `go test -v` text: a subtest's `--- PASS`
   line does not count for its parent's name.
3. **Red first,** plants with `make tools-test` as the command: `missing`
   counted as a pass; the build-failure check removed.
4. **First uses:**
   * `--expect pass` on 0066-MADR's 42 names (the PLAN's close lists
     them): 42 green.
   * `--expect fail --rev 5c11035` with `wizard/notices_0066_test.go`:
     its four tests red, as 0066-PLAN's Phase 5 record says.
5. **`tools/README.md`** gains its row.

### Phase 2: the leak finder (D2)

1. **`tools/leakfind.py`:** as MADR D2.
   * `REPORT` is the text before the hosts in C1's literal; a test reads
     C1's line and asserts the tool's constant appears in it, so a change
     to the message fails `make tools-test`.
   * The guard check: some `*_test.go` in the package's directory in the
     tree matches `localonly\.Main\(`; else exit 2.
   * A runner, local (a scratch clone, `go test` in it) or remote (`ssh -o
     BatchMode=yes -o ConnectTimeout=5 ALIAS` and one command, `cd DIR &&
     go test -count=1 -run PATTERN PACKAGE`, with `DIR` quoted by
     `shlex.quote` and each name checked against `^\w+$` before it enters
     the pattern). Both are functions the tests replace.
   * Output: each chunk's result, then each leaking test, or each set a
     leak needs, with its hosts.
2. **Tests,** `TestLeakFind`:
   * on a temporary Go module with its own `internal/localonly` whose
     `Main` prints C1's message when a test set a flag: a test that leaks
     alone is named; two tests that leak only together are named as a set,
     by leaving out; a package with no leak exits 0; a package with no
     `localonly.Main` exits 2;
   * the remote command for a made-up alias and a directory with a space,
     through a replaced runner: the options, the quoting, and no name
     unchecked;
   * C1's message holds the tool's constant.
3. **Red first,** plants: the leave-one-out step skipped (the pair is not
   found); the constant changed by a word (the C1 test fails).
4. **First use:** `tools/leakfind.py ./wizard/` on the tree: "no leak",
   exit 0, with the count of tests it listed. The Windows host is not
   used: the leak `4b511a5` fixed is gone, and a run there is not needed
   to prove the tool.
5. **`tools/README.md`** gains its row.

### Phase 3: open records (D3)

1. **`tools/records.py open`:** as MADR D3. `OPEN = {"MADR":
   ("proposed",), "PLAN": ("proposed", "in-progress")}`, each status
   asserted in C5 and C6's lists at import.
2. **Tests,** on a temporary copy of `docs/`: a `proposed` MADR and an
   `in-progress` PLAN listed in number order; an `accepted` MADR, a
   `complete` PLAN and a REPORT not; exit 0; nothing written.
3. **Red first:** `in-progress` left out of `OPEN`: its test fails.
4. **First use:** on the tree, before Phase 5's status change: this pair is
   listed, and every line agrees with the record's front matter.
5. **`tools/README.md`'s `records.py` row** (C11) gains `open`.

### Phase 4: the Markdown gate (D4)

1. **`tools/gate.py --markdown`:** as MADR D4.
   * Its checks are `[c for c in CHECKS if c.name in MARKDOWN_GATES]`, in
     `MARKDOWN_GATES`' order, asserted to find every name, then
     `deny-scan`; then, in `--src`, `tools/recordlint.py --changed` and
     `tools/claims.py --changed`.
   * After the scratch clone, a changed path not ending `.md` is reported
     and is exit 1, no check run.
   * No load wait.
2. **Tests,** in `TestGate`: the `--markdown` list is `MARKDOWN_GATES`
   then `deny-scan`, `recordlint`, `claims`; a changed `.go` file refuses
   (exit 1, the path named); C9's test unchanged.
3. **Red first:** `gate-selftest` dropped from the mode's list: the list
   test fails; the `.md` check removed: the refusal test fails.
4. **First use:** `tools/gate.py --markdown` on Phase 5's documents before
   they are staged: every check passes.
5. **`tools/README.md`'s gate row** (C10) names the mode.

### Phase 5: documents and close (D5)

1. `AGENTS.md`'s tools list (C12) names `redgreen.py`, `leakfind.py`,
   `records.py open` and `gate.py --markdown`; `docs/architecture.md`'s
   tree line (C13) and paragraph (C14) follow.
2. `tools/records.py status` sets this PLAN to `complete` once V1–V8 hold.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | `redgreen.py` fails a mistyped name, a red test, a package that does not build, and passes the clean case | 1 | `make tools-test` |
| V2 | 0066-MADR's 42 names green, and Phase 5's four red at `5c11035` | 1 | Phase 1 step 4 |
| V3 | `leakfind.py` names a lone leak and a pair's, and fails a change to the guard's message | 2 | `make tools-test` |
| V4 | `leakfind.py ./wizard/` finds no leak on the tree | 2 | Phase 2 step 4 |
| V5 | `records.py open` lists the open records and writes nothing | 3 | `make tools-test`; Phase 3 step 4 |
| V6 | `gate.py --markdown` runs `MARKDOWN_GATES`, the scan, the lint and the claims, and refuses a code change | 4 | `make tools-test`; Phase 4 step 4 |
| V7 | The full gate's checks and order are unchanged | 4 | C9's test |
| V8 | Every new tool, mode and test failed first | 1–4 | the execution record |

## Rollout and Rollback

* Tools are used as soon as they land; CI does not change.
* Rollback: revert the phase's commit. No phase changes what another
  tool's existing command does.

## Execution record

Not started.
