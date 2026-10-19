---
status: in-progress
date: 2026-10-19
associated-madr: "0047-MADR-keep-five-more-scratch-helpers-as-tools.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Keeping Five More Scratch Helpers: a Fixture Gate, Appends for `edit.py` and `plant.py`, Record Status, Record Lint, and Claims Plants

Associated MADR: [0047-MADR-keep-five-more-scratch-helpers-as-tools.md](0047-MADR-keep-five-more-scratch-helpers-as-tools.md)

## Goal

* **D1:** `scripts/check_fixtures.py` is a CI gate, `make fixtures-check`,
  with its baseline of 134 Together model ids, its planted breaches in
  `gate-selftest`, and AGENTS.md's redaction rule.
* **D2:** `tools/edit.py` appends, creates, and reads its text from files;
  `tools/plant.py` reads `old_file` and `new_file`.
* **D3:** `tools/records.py` sets a record's status and keeps the index in
  step.
* **D4:** `tools/recordlint.py` reports an edit's new markdownlint findings
  in the records.
* **D5:** `tools/claims.py --plants` runs planted false claims.
* **D6:** `tools/README.md`, AGENTS.md and `docs/architecture.md` describe
  them.

## Scope

### In scope

* New: `scripts/check_fixtures.py`, `scripts/fixtures-baseline.txt`,
  `tools/records.py`, `tools/recordlint.py`.
* Changed: `tools/edit.py`, `tools/plant.py`, `tools/claims.py`,
  `tools/gate.py`, `tools/test_tools.py`, `scripts/test_gates.py`,
  `Makefile`, `.github/workflows/ci.yml`, `tools/README.md`, AGENTS.md,
  `docs/architecture.md`, `docs/README.md`.

### Out of scope

* Any Go file, and so the API, the coverage floors and the parity map.
* The fixtures themselves: the review found no account data to redact.
* The weaker candidates the MADR names but does not choose: the issue-draft
  extractor and the listing envelope report.

## Preconditions and conventions

* **Approval.** 0047-MADR is accepted. The owner approves this PLAN, whole
  or by phase.
* **Gate, at the end of each phase:**
  * `tools/gate.py`: 24 checks before Phase 1, 25 from its end;
  * `tools/denyscan.py --changed`, `make records-check`;
  * `tools/claims.py` on this PLAN;
  * from Phase 3, `tools/recordlint.py --changed`.

  The phase is then staged for the owner's `git commit --no-edit`.
* **Red first.** Every new check, mode and test is seen to fail on a
  planted break, by `tools/plant.py`, `scripts/test_gates.py` or
  `tools/test_tools.py`, on a scratch clone, never in the tree.
* **No value is shown.** No step prints, logs or records a matched
  fixture string. The baseline holds fingerprints only. A test's planted
  address, id and JWT are made up, and are asserted absent from the
  tool's output.
* **Stop conditions:**
  * a fact of 0047-MADR, or a claim below, no longer holds;
  * the fixture scan, with the tightened patterns, finds anything but the
    134 Together ids on the tree;
  * a step would change a Go file.

## Code claims

<!-- claims: self=41b1ca7 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `edit.py`'s spec has one entry shape | `tools/edit.py:5` | `is a list of {"file", "old", "new"} objects` |
| C2 | `plant.py`'s plants have inline text only | `tools/plant.py:9` | `"plants": [{"name", "file", "old", "new"}, ...]}` |
| C3 | The gate runs `tables-check` | `tools/gate.py:99` | `make("tables-check"),` |
| C4 | The gate's markdownlint skips the records | `tools/gate.py:58` | `def markdown_files(root: Path) -> list[str]:` |
| C5 | `gate-selftest` maps each gate to its script | `scripts/test_gates.py:38` | `"tables-check": "scripts/check_tables.py",` |
| C6 | The Makefile's `tables-check` target | `Makefile:87` | `tables-check: ## Checks the import tables and the live variables against the code` |
| C7 | CI's list of gates | `.github/workflows/ci.yml:60` | `records-check tables-check gate-selftest` |
| C8 | `claims.py` finds a sibling repository | `tools/claims.py:91` | `def find_repo(name: str, repos: dict[str, Path], search: list[Path]) -> Path \| None:` |
| C9 | The index's count line | `scripts/check_records.py:47` | `COUNT = re.compile(` |
| C10 | `tools/README.md` says 23 checks; there are 24 | `tools/README.md:16` | `the local gate: 23 checks` |
| C11 | The test that pins the gate's checks | `tools/test_tools.py:205` | `"g-wire-stable", "g-links", "deny-scan", "tools-test"]` |
| C12 | AGENTS.md's CI list | `AGENTS.md:146` | `generate-check records-check tables-check gate-selftest` |
| C13 | AGENTS.md's tools list | `AGENTS.md:159` | `` `tools/denyscan.py --changed` and `tools/linkcheck.py` are the `` |

## Implementation Steps

### Phase 0: records

1. 0047-MADR is `accepted`; this PLAN is indexed. `make records-check`.

### Phase 1: the fixture gate (D1)

1. **`scripts/check_fixtures.py`,** in `check_tables.py`'s shape: a module
   docstring that is its definition; `scan(root, files)` returning hits as
   `(path, line, column, kind, fingerprint)`; `main`.
   * Patterns, compiled once: the five kinds of MADR D1, the account id
     requiring a digit or a capital letter after its prefix.
   * Never a hit: placeholders, the nil UUID, the example domains.
   * Files: `git ls-files` under any `testdata` directory, or `argv`.
   * Output, per finding: `path:line:column kind fingerprint`.
   * The baseline: a hit not in it, or a line no hit matches, is a
     finding. `--write-baseline` writes it, with a header naming
     0047-MADR's review, and prints the count only.
2. **The baseline.** `--write-baseline` on the tree. Assert, by a count
   from the file and not a reading of it: ~~134 lines~~ 133 lines for 134
   hits *(Deviation 1)*, every line's path
   `llmprovider/catalog/testdata/ranking-2026-10-12/together.json`, every
   kind `uuid`. Any other line is a stop condition.
3. **`Makefile`:** `fixtures-check: ## Checks recorded fixtures for account
   data` running `python3 -B scripts/check_fixtures.py`, in `.PHONY`.
4. **CI:** `.github/workflows/ci.yml:60` gains `fixtures-check` after
   `tables-check`.
5. **The local gate:** `tools/gate.py` runs `make("fixtures-check")` after
   `tables-check`; `tools/test_tools.py`'s order test follows (C11).
6. **`gate-selftest`:** `scripts/test_gates.py`'s `GATES` maps
   `fixtures-check`; tests plant into a copy of a fixture, each alone:
   * an address at a real-looking domain;
   * an `org-` id with digits and capitals;
   * a JWT;
   * a baseline line no hit matches.

   Each must exit 1, name the kind, and not hold the planted value. A
   copy with `user-REDACTED`, the nil UUID and `user@example.com` added
   must pass. The clean copy passes every gate.
7. **AGENTS.md:** C12's list gains `fixtures-check`; a paragraph, "Recorded
   fixtures", gives MADR D1's redaction, and says the gate enforces it.
8. **`tools/README.md`:** C10's count becomes 25; the gate's checks list
   names `fixtures-check`.
9. **Red first,** by `tools/plant.py` on a scratch clone, `make
   gate-selftest` as the command:
   * the JWT pattern removed: the JWT test fails;
   * the baseline's stale-line check removed: its test fails;
   * the value printed with the kind: the "not in the output" assertion
     fails.

### Phase 2: appends, new files and text from files (D2)

1. **`tools/edit.py`:** an entry is one of `old`/`new` (as now), `append`,
   or `create`; each text field may be `<field>_file`, read relative to the
   spec's directory. Every entry is checked before any write, as now. Its
   docstring states the three shapes.
2. **`tools/plant.py`:** a plant's `old` and `new` may be `old_file` and
   `new_file`, the same way.
3. **Tests,** in `tools/test_tools.py`:
   * `TestEdit`: an append to a file with and without a final newline; a
     create; a create of an existing file changes nothing (exit 1); text
     from a file; a missing text file (exit 2); every existing test
     unchanged.
   * `TestPlant`: a plant from files is caught as its inline twin is.
4. **Red first:** plants of `edit.py` (the newline rule; create's
   existence check) on a scratch clone, `make tools-test` the command.
5. **`tools/README.md`'s rows** for `edit.py` and `plant.py` follow.

### Phase 3: a record's status, and the index (D3)

1. **`tools/records.py`:** `add RECORD...` and `status RECORD STATUS
   [--date]`, as MADR D3. It imports `scripts/check_records.py` for its
   status lists, its index path and its count pattern (C9), and runs its
   checks after writing.
2. **Tests,** on a temporary copy of `docs/`: `add` a new MADR between two
   rows; `status` a PLAN to `complete`; a bad status (exit 2, nothing
   written); the count; "I want to…" untouched.
3. **Red first:** the count update removed: its test fails.
4. **Used at once:** this PLAN's later status changes are made with it.
5. **`tools/README.md`** gains its row.

### Phase 4: an edit's new markdownlint findings (D4)

1. **`tools/recordlint.py`:** as MADR D4. markdownlint-cli2 runs with
   `--no-globs` on a temporary copy at `HEAD` and on the tree's file, with
   the repository's config. The findings are counted per rule.
2. **Tests:** a record with a finding added (exit 1, the line named); a
   record with only old findings (exit 0); a new record compared with
   none. They skip, with a reason, when markdownlint-cli2 is not on `PATH`.
3. **Red first:** the comparison made against the tree twice: the added
   finding passes, so its test fails.
4. **`tools/README.md`** gains its row; the PLAN's gate (Preconditions)
   uses it from here.

### Phase 5: plants of false claims (D5)

1. **`tools/claims.py --plants SPEC.json`:** as MADR D5, with the options
   the record's check takes. `check_record` is called on the planted copy,
   in-process.
2. **Tests,** in `TestClaims`'s fixtures: a plant moving a row's line is
   caught; a plant that changes nothing the check reads passes, so it is a
   finding; an `old` that matches twice is a tool error (exit 2).
3. **A first use:** 0046-MADR's five plants of `issues/plant46.py` (C1,
   C13, G2, P4, the marker's revision) as a spec in a scratch directory:
   all five caught.
4. **Red first:** the "plant must fail" inverted: the caught plant reports
   not caught.
5. **`tools/README.md`**'s `claims.py` row follows.

### Phase 6: documents and close (D6)

1. AGENTS.md's tools list (C13) names `records.py`, `recordlint.py` and
   `claims.py --plants`; `docs/architecture.md` (`:39-41`, `:873-878`)
   names the new tool and gate.
2. `tools/records.py status` sets this PLAN to `complete` once V1–V8 hold.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | The fixture gate passes on the tree, and its baseline is 133 Together `uuid` lines for 134 hits (Deviation 1) | 1 | `make fixtures-check`; the baseline's counts |
| V2 | Each planted address, id, JWT and stale line is caught, and no value is printed | 1 | `make gate-selftest` |
| V3 | CI runs `fixtures-check` | 1 | the CI run on `main` |
| V4 | `edit.py` appends and creates, and reads text from files; old specs unchanged | 2 | `make tools-test` |
| V5 | `records.py` changes a status and the index, and `check_records.py` accepts the result | 3 | `make tools-test`; `make records-check` |
| V6 | `recordlint.py` names an added finding only | 4 | `make tools-test` |
| V7 | `claims.py --plants` catches 0046's five plants | 5 | Phase 5 step 3 |
| V8 | Every new check and mode failed first | 1–5 | the execution record |

## Rollout and Rollback

* Tools are used as soon as they land; CI gains one gate in Phase 1.
* Rollback: revert the phase's commit. A revert of Phase 1 removes the
  gate from CI and the local gate together.

## Execution record

### Phase 0: records (2026-10-19)

* The MADR was accepted and this PLAN indexed, committed by the owner as
  `06de97d`; the owner then approved the PLAN ("approved, proceed").

### Deviation 1 (2026-10-19): 133 baseline lines for 134 hits

* **Found.** Step 2: `--write-baseline` wrote 133 lines for 134 hits. Two
  Together entries, `mistralai/Mistral-7B-Instruct-v0.1` and `-v0.3`, carry
  one UUID (fingerprint `2dd3fb6ddc44`). Step 2 and V1 asserted 134 lines;
  the stop condition holds (only Together's UUIDs are found).
* **Resolution, the owner's:** a line admits a value wherever it occurs in
  its file; the baseline is 133 lines. Step 2 and V1 are annotated; the
  MADR's A1 records the fact.

### Phase 1: the fixture gate (2026-10-19)

* **Deviation 1,** above, on the owner's choice.
* **Step 1.** `scripts/check_fixtures.py`: the five kinds; placeholders,
  the nil UUID and the example domains never hits; files from `git ls-files
  -co --exclude-standard` under a `testdata` directory, so a fixture not yet
  added is scanned too; a finding per unadmitted hit and per stale line.
* **Step 2.** Before the baseline: 250 files, 134 hits, every one a `uuid`
  in `llmprovider/catalog/testdata/ranking-2026-10-12/together.json`; none
  of the 54 placeholder and prose hits of the MADR's review. `--write-baseline`:
  "wrote 133 baseline line(s) for 134 hit(s)"; the file's lines, counted by
  path and kind, are 133 of that path and `uuid`. Then `make
  fixtures-check`: "250 file(s), 134 hit(s), 0 problem(s)".
* **Steps 3–8** as written: the `fixtures-check` target; CI's list
  (`.github/workflows/ci.yml:60`); the local gate after `tables-check`, and
  the order test; five `gate-selftest` cases (an address, an `org-` id, a
  JWT, each made up and asserted absent from the output; a stale baseline
  line; placeholders that must pass); AGENTS.md's CI list and its "Recorded
  fixtures" section; `tools/README.md`'s count, 25, and its checks list,
  which also gains `tables-check`, missing since 0046.
* **Step 9, red first,** `tools/plant.py` on a scratch clone, with `python3
  -B -m unittest scripts/test_gates.py -k fixtures` as the command: all five
  plants caught (exit 1 each):
  * the JWT kind removed;
  * the stale-line check removed;
  * the value printed in place of its fingerprint: caught by the address
    and id tests' "unexpectedly found" assertion. That planted run printed
    a Together model id, a public one of the baseline, as the plant meant;
  * placeholders no longer allowed;
  * lowercase-word ids reported.
* **The gate:** `tools/gate.py`, 25 of 25 passed.
* **After the commit.** The owner committed the phase as `3705255` and
  pushed. CI run `39275511985`: `success` on macOS, Windows and Linux; its
  log holds `fixtures-check: 250 file(s), 134 hit(s), 0 problem(s)`. V3 is
  met.

### Phase 2: appends, new files and text from files (2026-10-19)

* **No deviation.**
* **Steps 1–2.** `tools/_common.py` gains `has_text` and `text_field`: a
  field inline, or the text of `<field>_file` relative to the spec's
  directory; both or neither is a `ValueError`. `edit.py`'s `plan` takes
  an entry of exactly one shape (`old`/`new`, `append` or `create`), else
  exit 2; `create` of a file that exists, or that an earlier entry wrote, is
  a finding (exit 1, nothing written); a created file's directories are
  made. `plant.py` reads each plant's `old` and `new` through `text_field`
  before the clone, so a missing file is exit 2.
* **Step 3.** Seven `TestEdit` tests (append with and without a final
  newline; append after a replace; create; create of an existing file
  changes nothing; text from files; a missing text file; an entry of two
  shapes) and one `TestPlant` test (a plant from files caught as its inline
  twin). The four existing `TestEdit` tests are unchanged. `make
  tools-test`: 53 tests, OK.
* **Step 4, red first,** `tools/plant.py` on a scratch clone, the command
  the `TestEdit` and `TestPlant` tests; the third plant's text was itself
  read from files, by the new `plant.py`:
  * the append's newline rule removed: `test_an_append_adds_at_the_end`
    fails;
  * create's existence check removed:
    `test_a_create_of_an_existing_file_changes_nothing` fails;
  * a text file's name taken for its text: three tests fail, among them
    the plant-from-files test.
* **Step 5.** `tools/README.md`'s `edit.py` and `plant.py` rows.
* **The gate:** `tools/gate.py`, 25 of 25 passed.

### Phase 3: a record's status, and the index (2026-10-19)

* **No deviation.** Phase 2 was committed by the owner as `f824bc3`.
* **Step 1.** `tools/records.py`: `add` and `status`. It imports
  `scripts/check_records.py` for its name pattern, status lists, index
  path, row and count patterns, and front-matter reader, and runs its three
  checks after writing, with its `ROOT` pointed at `--root` for the call
  and restored after.
  * `add` places a row after the last row of the same or a lower number:
    the index does not order one number's kinds (0015's REPORT precedes its
    MADR), so no kind order is imposed. The title is the record's first
    heading after its front matter. The count is the table's rows.
  * `status` checks the status against the record's kind before writing
    anything, sets `status` and `date` (adding `date` after `status` when
    absent), and the row's last cell.
* **Step 2.** Six `TestRecords` tests on a temporary repository: `add` by
  number with the count and "I want to…" unchanged; a PLAN after its MADR;
  `add` of an indexed record writes nothing (exit 1); `status` with its
  date and row; a bad status writes nothing (exit 2); a PLAN without its
  MADR fails the checks run after writing (exit 1).
* **Step 3, red first,** `tools/plant.py`, the `TestRecords` tests the
  command: the count not updated (two tests fail); the row put at the
  section's end (two fail); the checks not run after writing (one fails).
* **Step 4.** On a copy of this repository's `docs/`: `records.py status`
  of this PLAN to `complete`, exit 0, "records-check: 0 problem(s)"; the
  copy's front matter and row say `complete`. The tree was not touched.
  This PLAN's own status changes are made with it from here.
* **Step 5.** `tools/README.md`'s row.
* **The gate:** `tools/gate.py`, 25 of 25 passed.

