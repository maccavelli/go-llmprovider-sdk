---
status: in-progress
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
   * *The whole package runs first; a clean whole run is "no leak"
     (Deviation 2, 0067-MADR A1).*
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
5. **`tools/README.md`'s gate row** (C10) names the mode, *and its "28
   checks" becomes 29 (Deviation 1)*.

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

### Phase 0: records (2026-11-01)

* The owner answered Q1–Q3, each as recommended, and 0067-MADR records
  them; `tools/records.py status` set it `accepted`. The pair and its
  index rows are committed as `a118c6b`. The owner approved the PLAN.
* `tools/records.py status` set this PLAN `in-progress`: "records: written;
  records-check: 0 problem(s)".
* Before any change, `make tools-test`: "Ran 77 tests", "OK", in 29 s.

### Phase 1: named tests, red or green (2026-11-01)

* **`tools/redgreen.py`:** as step 1, with `SpecError` for exit 2,
  `drop_func` for a drop, and `output_name` for each package's output
  file. A spec key outside `tests`, `files` and `drop`, and a name both run
  and dropped, are spec errors too. A missing `go` is exit 2, not a
  traceback.
* **Tests,** `TestRedGreen`, 12, on a module whose first commit has `Add`
  subtract and whose second fixes it and adds `Sub`, with `fix_test.go`:
  green on the tree, its output file read; red at the first commit with
  `TestSub` dropped; each expectation's converse a finding; the package
  that does not build without the drop, "BAD missing (did not build)",
  exit 1 (Q1); a mistyped name, "BAD missing", exit 1; a drop that
  matches nothing and `files` without `--rev`, exit 2; the tree and its
  status unchanged; a drop takes both comment lines above it; a
  subtest's line does not count for its parent; the output names.
  "Ran 12 tests", "OK".
* **Red first** (`tools/plant.py`, `make tools-test`): "all 4 plant(s)
  caught", each failing its test, `missing counted as a pass`
  (`test_a_mistyped_name_is_missing`: "0 != 1"), `the build-failure
  check removed` ("'did not build' not found"), `a drop leaves the
  comment lines` ("'TestSub' unexpectedly found"), `files and drop
  allowed without --rev` ("0 != 2"); run again after the fix below, all
  4 caught.
* **Found in the first use, and fixed in the phase:** the output files were
  named from the package with its dots kept, so `./wizard/` wrote
  `._wizard.out` and `.` wrote `.out`, hidden from `ls`. `output_name`
  drops the dots and names `.` `root.out`; `test_green_on_the_tree` reads
  `root.out`, and `test_each_package_output_has_a_visible_name` checks
  three packages. The plant restoring the old name: "all 1 plant(s)
  caught", `FileNotFoundError: … root.out` and "'._llmprovider_providers_chatcompat.out'
  != 'llmprovider_providers_chatcompat.out'".
* **First uses:**
  * `--expect pass` on 0066's 42 names, taken from the close's list, not
    retyped: "redgreen: 42 of 42 pass as expected", the four packages
    exit 0, one output file each.
  * `--expect fail --rev 5c11035` with `wizard/notices_0066_test.go`,
    unchanged since `3e48e8b` added it: "redgreen: 4 of 4 fail as
    expected", `./wizard/` exit 1, no build failure. Its output holds
    Phase 5's record's failures: "want offline", "has a query or a
    fragment", `SERVER-TEXT-0066`, "notices []; want".
* **`tools/README.md`:** the `redgreen.py` row and "Red and green specs".
* **Gate:** `tools/gate.py`, started at load 9.3: "gate: 29 of 29
  passed", exit 0; its `tools-test` "Ran 89 tests", "OK" (77 before, 12
  added).
* **Found, not in this phase's scope:** `tools/README.md:16` (C10) says
  "the local gate: 28 checks"; `gate.CHECKS` holds 29, and the gate printed
  "29 of 29". The row predates 0067; it is put to the owner, since Phase 4
  edits that row.

### Deviation 1 (2026-11-01): the gate row's count

* **Found** after Phase 1's gate: `tools/README.md:16` says "the local
  gate: 28 checks"; `tools/gate.py`'s `CHECKS` holds 29, which
  `TestGateChecks` pins, and the gate printed "29 of 29 passed". The
  uncounted check is `test-repeat`, added by 0058-MADR's amendment of
  2026-10-24. Not caused by 0067.
* **The owner chose (2026-11-01):** fix it in Phase 4, which already edits
  that row: "28 checks" becomes 29. Phase 4 step 5 is annotated, and
  `tools/README.md` was already in this PLAN's scope. No MADR decision or
  fact changes.

### Phase 2 begins (2026-11-01)

* Phase 1 is committed as `7d96582`. `tools/leakfind.py` and its tests,
  drafted while Phase 1's gate ran, are copied in.

### Deviation 2 (2026-11-01): the leak finder runs the whole package first

* **Found** while writing the tool: MADR D2's search starts with chunks, so
  a leak that needs two tests in two chunks reads "no leak", exit 0. The
  draft ran the whole package first, which D2 does not say.
* **The owner chose (2026-11-01):** keep the whole run and amend D2
  (0067-MADR A1). Phase 2 step 1 is annotated. A test of the case it
  covers, `test_a_leak_across_chunks_is_not_clean`, is added to step 2's.

### Phase 2: the leak finder (2026-11-01)

* **`tools/leakfind.py`:** as step 1 and Deviation 2. `Finder.leaks` runs
  a set up to `--runs` times and returns the hosts of the first run that
  leaks; `Finder.needed` leaves each test out in turn; `search` prints
  each step. `PACKAGE` must look like `./path/`, `--ssh` must be a plain
  alias (`^\w[\w.-]*$`, so never an option) and needs `--dir`, and
  `--chunk` and `--runs` are at least 1; each else is exit 2 before
  anything runs.
* **Tests,** 10. `TestLeakFind` (6), on a module with a stand-in
  `internal/localonly` whose `Main` prints the real report's text when a
  test called `Reach`: `--chunk 3` names `TestAlone` alone and
  `TestPairA, TestPairB` together, not `TestClean3`, "2 leak(s) found",
  exit 1; two tests that leak only together, in chunks of 1, exit 1 with
  "run with a larger --chunk"; a clean package exit 0; a package with no
  `localonly.Main` exit 2; `--ssh` without `--dir`, and an alias that is
  an option, exit 2 with the reason, with `run` replaced by a stub that
  fails the test if any command runs; the tree unchanged.
  `TestLeakFindParts` (4): C1's line holds `REPORT` in quotes; the remote
  command for a directory with a space; only `Test`, `Example` and `Fuzz`
  names leave a listing; a runner's set reaches `go test` as
  `^(TestA|TestB)$`. "Ran 10 tests", "OK".
* **Red first** (`tools/plant.py`, `make tools-test`), seven plants, all
  caught in the end:
  * `the leave-one-out step skipped`: "'LEAK TestPairA, TestPairB,
    together: pair.test:443 (1)' not found";
  * `the guard's report changed by a word`: the lone-and-pair test "0 !=
    1", and `test_the_report_is_the_guards`;
  * `any listing line taken as a name`: "7 tests" for 6, and the listing
    test;
  * `the remote directory unquoted`: "cd work dir/sdk && …" for "cd 'work
    dir/sdk' && …";
  * `any alias accepted`: **not caught** on the first run, since the test
    checked only the exit code, and the run still ended 2, later: with
    the check planted away, the code path runs `ssh` with the planted
    value in the alias's place, and its output was not kept. The test
    now asserts the reason and fails on any command run; with
    it, "a refused --ssh ran a command", caught, and `--ssh without --dir
    accepted` the same. The first five were run again with the final
    tests: "all 5 plant(s) caught";
  * for Deviation 2, `a leak only the whole run shows reported clean`
    ("0 != 1") and `the whole run skipped` (three tests fail): "all 2
    plant(s) caught".
* **First use:** `tools/leakfind.py ./wizard/` on the tree: "leakfind: 270
  tests in ./wizard/", "leakfind: no leak", exit 0, in 15 s; `go test
  -list . ./wizard/` lists 270. The dev host was not used, as step 4 says.
* **`tools/README.md`:** the `leakfind.py` row.
* **Gate:** `tools/gate.py`, started at load 9.9: "gate: 29 of 29
  passed", exit 0; its `tools-test` "Ran 99 tests", "OK" (89 before, 10
  added).

### Phase 3: open records (2026-11-01)

* Phase 2 is committed as `55d6f87`. The change was drafted as a
  `tools/edit.py` spec while Phase 2's gate ran: "edit: 9 entry(ies)
  applied to 2 file(s)".
* **`tools/records.py open`:** as step 1. `OPEN` and two `assert`s
  against `check_records.py`'s lists at import; `open_records` reads the
  records through `check_records.scan`, in its order, decisions by name,
  so by number; `open` prints and returns before the write path, so it
  runs no check. The docstring names the third command.
* **Tests,** 2, in `TestRecords`: with a `complete` PLAN, an
  `in-progress` PLAN and a REPORT added, `open` prints exactly the
  `proposed` MADR and the `in-progress` PLAN, then "records: 2 open", exit
  0, and no file changes; `OPEN`'s statuses are within
  `check_records.py`'s. `TestRecords`: "Ran 8 tests", "OK".
* **Red first** (`tools/plant.py`, `make tools-test`): "all 3 plant(s)
  caught": `in-progress left out of OPEN` and `accepted counted open`
  (the list test, "Lists differ"); `a status check_records does not
  know` (the import's `AssertionError`, so every test fails).
* **First use,** on the tree: `in-progress  2026-11-01
  docs/decisions/0067-PLAN-keep-four-more-scratch-helpers-as-tools.md`,
  "records: 1 open", exit 0. Step 4 expected "this pair": the MADR is
  `accepted` since Phase 0, so it is rightly not listed. A read of every
  record's front matter without `check_records.py` agrees: 141 records,
  63 MADRs `accepted`, 69 PLANs `complete`, this PLAN `in-progress`, 8
  REPORTs; one open.
* **`tools/README.md`:** `records.py`'s row (C11) names `open`.
* **Gate:** `tools/gate.py`, started at load 9.0: "gate: 29 of 29
  passed", exit 0; its `tools-test` "Ran 101 tests", "OK" (99 before, 2
  added).

### Phase 4: the Markdown gate (2026-11-01)

* Phase 3 is committed as `58a223b`. The change was drafted as a
  `tools/edit.py` spec while Phase 3's gate ran: "edit: 8 entry(ies)
  applied to 3 file(s)".
* **`tools/gate.py --markdown`:** as step 1. `markdown_checks(src)` takes
  the gate's own checks named in `MARKDOWN_GATES`, imported from
  `scripts/test_gates.py`, in its order, and raises if one is missing;
  then `deny-scan`; then `recordlint` and `claims`, run in `--src` by
  `in_tree`. The paths the change touches are read from the clone's
  commit, `git diff --name-only HEAD^ HEAD`, so a deleted file counts,
  which `scratch_clone`'s list leaves out. The full gate's path is
  unchanged but for `src` named once; its docstring gives the mode, its
  usage and exit 1 for a refusal.
* **Tests,** 2, in `TestGateChecks`: the mode's list is `MARKDOWN_GATES`,
  then `deny-scan`, `recordlint`, `claims`, `gate-selftest` among them; a
  repository whose change edits a `.md`, edits a `.go` and deletes a `.go`
  is refused, exit 1, naming both `.go` files and not the `.md`, "gate
  --markdown: nothing run", no `summary.out`. C9's test is unchanged.
  `TestGateChecks`: "Ran 3 tests", "OK".
* **Red first** (`tools/plant.py`, `make tools-test`): "all 3 plant(s)
  caught": `gate-selftest dropped from --markdown` (the list test, "Lists
  differ"); `the Markdown-only refusal removed` ("'gate: code.go is not
  Markdown' not found"; the checks then ran in the test's temporary
  repository and failed there, `rc=2`); `deletions left out of the
  refusal` ("'gate: gone.go is not Markdown' not found").
* **On the tree,** with this phase's Python unstaged: "gate: tools/gate.py
  is not Markdown; …", "gate: tools/test_tools.py is not Markdown; …",
  "gate --markdown: nothing run", exit 1. Step 4's first use, on Phase
  5's documents, is Phase 5's.
* **`tools/README.md`:** the gate row (C10) names the mode, and "28
  checks" is 29 (Deviation 1).
* **Gate:** `tools/gate.py`, started at load 9.1: "gate: 29 of 29
  passed", exit 0; its `tools-test` "Ran 103 tests", "OK" (101 before, 2
  added).
