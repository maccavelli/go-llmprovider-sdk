---
status: in-progress
date: 2026-10-12
associated-madr: "0035-MADR-maintainer-tools-in-python.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Maintainer Tools in `tools/`

Associated MADR: [0035-MADR-maintainer-tools-in-python.md](0035-MADR-maintainer-tools-in-python.md)

## Goal

When it is done:

* the tools of 0035-MADR D1 are in `tools/`, in Python on the standard
  library, each tested on a planted input;
* `make tools-test` runs their tests, locally and in the local gate, not
  in CI (Q1);
* no committed file names a host, an account or a machine path;
* `AGENTS.md`, `docs/architecture.md` and `tools/README.md` say where each
  tool is and how to run it.

No Go code changes, and no exported API.

## Scope

### In scope

* **`tools/`:** `_common.py`, `gate.py`, `linkcheck.py`, `denyscan.py`,
  `apidiff.py`, `plant.py`, `liveskips.py`, `consumer.py`, `edit.py`,
  `README.md` and `test_tools.py`.
* **`Makefile`:** a `tools-test` target.
* **Docs:** `AGENTS.md` ("Pre-add checks", and a pointer to `tools/`), the
  layout block and the gates section of `docs/architecture.md`,
  `docs/README.md`'s index.

### Out of scope

* Changing what any CI gate in `scripts/` checks.
* The disclosure guard, which is machine-wide, not this repository's.
* CI (0035-MADR Q1): `.github/workflows/ci.yml` is unchanged.
* `scripts/go-precheck.sh`, which stays in bash (Q2).
* The per-phase record scripts.

## Rules for every phase

* **Red first:** each tool's test is seen failing, and its FAIL lines are
  recorded.
* **Plants** run in temporary directories or a scratch clone, never in the
  tree.
* **Every script is Python,** including waits and loops (the owner's
  direction of 2026-10-12).
* **A phase is staged only after:**
  * `make tools-test`;
  * `make records-check`;
  * markdownlint;
  * `tools/denyscan.py` over the changed files;
  * the gate, started below the load limit.
* **A deviation stops the work:** evidence, options with the recommended
  one first, and a dated entry here once the owner chooses.
* **The owner commits** each phase.

## Implementation Steps

### Phase 1: the shared code, and the three small tools

1. **Red first** (`tools/test_tools.py`):

   | Test | Asserts |
   | :--- | :--- |
   | `TestRoot` | `_common.ROOT` holds `go.mod`; it is found from the file, whatever the working directory |
   | `TestWaitLoad` | with a load function that answers 50, then 1, and a limit of 10, it returns `True` after one wait; with one that always answers 50 and a timeout of 0, it returns `False` |
   | `TestLinkCheck` | in a temporary git repo: a link to a missing file is a finding (exit 1, the file and line printed); a link inside a code fence, an `https:` link and a pure `#anchor` are not checked; a clean tree exits 0 |
   | `TestDenyScan` | a file holding the test's own account name gives a hit naming the kind, and the output never holds the name; a file with `/home/<user>/x` gives none; a file with a home path under `/Users/` that names an account, built by the test, gives a home-path hit |
   | `TestEdit` | a spec whose `old` matches once is applied; one that matches twice, or not at all, exits 1 and leaves every file unchanged |

2. **Change:**
   * **`tools/_common.py`:** `ROOT`, `run(args, cwd, env)`, `wait_load(limit,
     timeout, load=os.getloadavg)`, and `scratch_clone(dest)`. The clone
     copies the working tree, deletions included, into a shared clone,
     stages it, and commits it with `git commit-tree`, as the gate did.
   * **`tools/linkcheck.py`:** `g_links.py`, with `argparse` and no
     positional repo argument (it uses `ROOT`).
   * **`tools/denyscan.py`:**
     * given files, or `--changed` for the files `git status` lists;
     * it reads no hard-coded repo path;
     * a relative name is resolved from `ROOT`.
   * **`tools/edit.py`:** reads `[{"file", "old", "new", "count": 1}]`.
     Every replacement is checked before any file is written.
   * **`Makefile`:** `tools-test: python3 -B -m unittest discover -s
     tools -p 'test_*.py'`.
3. **Plants**, each failing its test:
   * `linkcheck` skipping every link;
   * `denyscan` printing the identifier;
   * `edit` writing before it has checked every replacement;
   * `wait_load` returning at once.
4. Gate; stage.

### Phase 2: the gate, the plant runner, and apidiff

1. **Red first:**

   | Test | Asserts |
   | :--- | :--- |
   | `TestGateChecks` | `gate.CHECKS` names the 23 checks, in order: the 22 of 0026-PLAN's gate (`pre-add-check`, the six `vet` variants, `test-race-cover`, `test-shuffle`, `tidy-diff`, `lint`, `parity-check`, `dep-check`, `coverage-check`, `api-check`, `generate-check`, `records-check`, `gate-selftest`, `markdownlint`, `g-wire-stable`, `g-links`, `deny-scan`), then `tools-test` (Q1) |
   | `TestScratchClone` | in a temporary repo with a modified file, an untracked file and a staged deletion, the clone's `HEAD` tree holds the modification and the new file, and not the deleted one |
   | `TestPlant` | a spec with a plant its test catches exits 0 and prints the FAIL lines; one with a plant no test catches exits 1, naming it; a plant whose `old` does not match exits 2. A tiny Go module under a temporary directory is the target |
   | `TestApidiffArgs` | the pinned version is read from `scripts/check_api.py`, not repeated |

2. **Change:**
   * **`tools/gate.py`:**
     * `p26_gate.py`, with `CHECKS` as data;
     * `--max-load`, the CPU count by default, and `--timeout` for the
       wait;
     * `--out`;
     * the markdownlint file list read from the repo, not hard-coded;
     * it calls `linkcheck` and `denyscan` as modules.
   * **`tools/plant.py`:** a spec of `{name, file, old, new}`, the
     packages, and a `-run` pattern.
   * **`tools/apidiff.py`:** `p10_apidiff.py`, with `OLD [NEW]` refs.
3. **Proof of the gate:** `tools/gate.py` is run once on a scratch clone
   holding a `vet` error and a wrong index status. Both must fail, and the
   FAIL lines are recorded. Then it is run on the clean tree, where all 23
   checks must pass.
4. **Plants:**
   * `TestScratchClone`'s, which drops the deletion;
   * `TestPlant`'s, which reports a passing plant as caught.
5. Gate; stage.

### Phase 3: the release helpers

1. **Red first:**

   | Test | Asserts |
   | :--- | :--- |
   | `TestLiveSkips` | a canned `go test -v` output gives the right counts of top-level and subtest passes, skips and failures, and each skip's last log line as its reason; `--- FAIL` exits 1 |
   | `TestConsumerSource` | the generated `main.go` imports every public package (from `go list ./...`, without `internal`), and builds against the tree through a `replace` directive, with no network |

2. **Change:**
   * **`tools/liveskips.py`.**
   * **`tools/consumer.py`:**
     * `VERSION`;
     * `--dependent PATH`, which copies that repository, skipping
       `*.devops.*.json`, `secrets.json` and `.env*`, and moves it to the
       version;
     * `--platforms`, six by default.
3. **Plants:**
   * `liveskips` counting a subtest as top level;
   * `consumer` leaving a public package out.
4. Gate; stage.

### Phase 4: documentation

1. **Docs:**
   * `tools/README.md`, each tool with its usage and exit codes;
   * `AGENTS.md`'s "Pre-add checks" and a "Tools" paragraph;
   * `docs/architecture.md`'s layout block and gates section;
   * `scripts/` named as CI's, and `tools/` as the maintainers'.
2. **The scan of the tools:** `tools/denyscan.py tools/*.py tools/README.md`
   finds nothing.
3. This PLAN `complete`; the index updated.

## Verification

* **V1. Red, then green,** with every plant's FAIL line.
* **V2.** `make tools-test` passes, and fails on each recorded plant.
* **V3.** The gate's planted proof in Phase 2, and a clean 23 of 23.
* **V4.** No committed file in `tools/` names a host, an account or a
  machine path (`tools/denyscan.py`).

## Rollout and Rollback

* **Rollout:** the records, then one commit per phase. No release is
  needed: nothing in the module changes.
* **Rollback:** each phase reverts alone. The tools are not imported by
  the module, so a revert cannot break a consumer.

## Execution Record

### Phase 0: records (2026-10-12)

* The MADR and this PLAN were written `proposed`.
* The owner answered Q1 (local only) and Q2 (leave `go-precheck.sh` in
  bash), and committed them as `ae1a95f`.
* The owner then directed "proceed to 0035 execute the plan", which
  accepts the MADR and approves this PLAN.

### Phase 1: the shared code, and the three small tools (2026-10-12)

**Change:**

* **`tools/_common.py`:**
  * `ROOT`, from the file's own location;
  * `out_dir`, under `tempfile.gettempdir()`;
  * `run`, an argument list, never a shell;
  * `load_limit`, the CPU count;
  * `wait_load(limit, timeout, *, load, sleep, clock, interval)`;
  * `scratch_clone(src, dest)`.
* **`tools/linkcheck.py`:** `g_links.py`, with `argparse`, `--root` and
  `--detail`.
* **`tools/denyscan.py`:** files, or `--changed` from `git status
  --porcelain -z`. It reads the account and host names at run time, and
  prints only the kind of hit.
* **`tools/edit.py`:** a JSON spec of `{file, old, new, count}`. Every
  entry is checked before any file is written.
* **`tools/test_tools.py`:** 15 tests.
* **`Makefile`:** `tools-test`, local only (Q1), and in `.PHONY`. The
  Makefile edit was the edit tool's first spec.

**`TestScratchClone`,** which Phase 2 lists, is written here, because
`scratch_clone` lands here. It adds a test, and changes no scope.

**Red first:**

```text
    import _common
ModuleNotFoundError: No module named '_common'
FAILED (errors=1)
```

**Green:** `make tools-test`: `Ran 15 tests … OK`.

**The tools on the tree:**

* `tools/linkcheck.py`: `G-links: 0 problem(s), 743 relative link(s) in
  86 file(s)`, the same counts the scratch link checker gave on the same
  tree;
* `tools/denyscan.py --changed`: `deny-scan: 0 hit(s) in 6 file(s)`.

**Plants,** each on a temporary copy of `tools/`:

| Plant | FAIL lines |
| :--- | :--- |
| `linkcheck` skipping every link | `test_a_missing_target_is_a_finding`: `AssertionError: 0 != 1 : G-links: 0 problem(s), 0 relative link(s) in 2 file(s)` |
| `denyscan` printing the identifier | `test_the_account_name_is_a_hit_and_is_never_printed`: `AssertionError: '<the account name>' unexpectedly found in 'f.md:1: the local account name written by <the account name> today …'` |
| `edit` writing before it checks every entry | `test_a_double_match_changes_nothing`: `AssertionError: 'beta\n' != 'alpha\n'` |
| `wait_load` returning at once | `test_gives_up_at_the_timeout`: `True is not false`; `test_waits_until_the_load_falls`: `0 != 1` |
| `scratch_clone` dropping the deletion | `test_the_clone_commits_the_working_tree`: `Lists differ: ['changed.txt', 'deleted.txt', 'kept.txt', 'new.txt'] != ['changed.txt', 'kept.txt', 'new.txt']` |

The account name in the second plant's output is given here as a
placeholder, as AGENTS.md's "Identifiers" rule requires.

**Gate:** the 22 checks, then `make tools-test`, all passing. They were
started at a load average of 7.4 by a scratch runner that waits with
`tools/_common.wait_load`. `tools/gate.py` is Phase 2's.
