---
status: accepted
date: 2026-10-12
decision-makers: repository owner
consulted: 0026-PLAN-remediate-v1-2-debugging-pass-findings.md (the gate, rule 3), AGENTS.md (pre-add checks)
informed: maintainers and agents working in this repository
---

<!-- markdownlint-disable MD013 -->

# Keep the Maintainer Tools in `tools/`, Written in Python

## Context and Problem Statement

Each plan since 0026 has been checked with helper scripts that lived only in
an agent's session scratchpad:

* the local gate: 22 checks on a scratch copy of the tree;
* a relative-link checker for every Markdown file;
* the identifier scan;
* apidiff between two refs;
* plant runners;
* a classifier for live-suite output;
* a post-tag consumer check;
* exact-edit helpers for records.

**The scratchpad is pruned.** The link checker and the identifier scan were
rebuilt from the transcript on 2026-10-06. The gate itself was lost on
2026-10-12, and rebuilt from its one write and two edits in the transcript.
A maintainer without that transcript could not have rebuilt it.

**Some waits ran as bash loops:** a `sysctl` and `awk` load check, which is
BSD-specific. On 2026-10-12 the owner directed:

1. "ensure all scripts are idiomatic python, not bash";
2. "any reusable script should be saved into the repo tools/ dir for
   persistent availability".

**What the repository has today** (`main` at `10c1be3`):

* **`scripts/`** holds the CI gates, each a `make` target:
  * `check_api.py`, `check_coverage.py`, `check_deps.py`,
    `check_generated.py`, `check_parity_map.py`, `check_records.py`, and
    `test_gates.py` (`gate-selftest`);
  * all Python on the standard library, with `ROOT =
    Path(__file__).resolve().parent.parent`, a docstring stating usage and
    exit codes, and `main() -> int`.
* **One bash script:** `scripts/go-precheck.sh` (188 lines), behind
  `make pre-add-check`.
  * The machine-wide agent commit gate runs `./scripts/go-precheck.sh`
    by that path when it is executable.
  * Otherwise it falls back to `make pre-add-check`, which checks every
    tracked Go file rather than the staged ones.
* **CI** runs `make lint`, then `make parity-check dep-check
  coverage-check api-check generate-check records-check gate-selftest`.
* **`.gitignore`** already ignores `__pycache__/` and `*.py[cod]`.
* **The scratch scripts** hard-code the checkout's location under the home
  directory. The external repositories they read, such as gobble-cli, are
  machine paths too.
* **Load:** the `TestCommandToken_*` tests time out under host load
  (0034-PLAN, Phase 4e amended), so the gate has been started only once
  the load average fell below the CPU count. That rule lived only in a
  shell loop.

## Decision Drivers

* The tools survive any session, and any maintainer can run them.
* Python 3 and its standard library only, as `scripts/` is: no
  dependency, no virtual environment.
* Nothing committed carries a host name, an account name or a machine
  path. The repo root comes from the file's own location; anything outside
  the repo is an argument.
* Each tool is seen failing on a planted input before it is trusted, as
  every gate here is.
* CI's behaviour changes only by decision. Q1 keeps it unchanged.
* The machine-wide commit gate keeps working, and stays as fast.
* The tools run on macOS and Linux alike, with no BSD-only or GNU-only
  commands.

## Considered Options

* **`tools/` for the maintainer tools, beside `scripts/` for the CI
  gates**
* Everything in `scripts/`
* A Python package with a `pyproject.toml` and console entry points
* Leave the tools in each session's scratchpad

## Decision Outcome

Chosen option: "`tools/` for the maintainer tools, beside `scripts/` for the
CI gates", because:

* it is where the owner asked for them;
* it keeps `scripts/` what `docs/architecture.md` says it is, the gates
  CI runs;
* it costs no dependency.

### D1. What goes in `tools/`

| Tool | What it does | Comes from |
| :--- | :--- | :--- |
| `tools/_common.py` | the repo root; running a command with its output captured; the load wait; a scratch clone of the working tree | the scratch scripts' shared code |
| `tools/gate.py` | the local gate: 0026-PLAN's 22 checks, then `tools-test`, on a scratch clone of the working tree, staged there, started once the load is below the CPU count | `p26_gate.py` |
| `tools/linkcheck.py` | every relative link in every tracked or untracked Markdown file resolves | `g_links.py` |
| `tools/denyscan.py` | the identifier scan of given files, or of the changed files; it prints the kind of hit, never the identifier | `denyscan.py` |
| `tools/apidiff.py` | apidiff between two refs, or a ref and the tree, at `scripts/check_api.py`'s pinned version | `p10_apidiff.py` |
| `tools/plant.py` | runs plants from a JSON spec on a scratch clone; fails if any plant passes; prints each plant's FAIL lines | the plant runners |
| `tools/liveskips.py` | sorts a `go test -v` run into passes, skips with their reasons, and failures | the live-run classifier |
| `tools/consumer.py` | the post-tag check: a scratch consumer of every public package at a version, built for six platforms; optionally a scratch copy of a dependent repo, given by path, moved to that version | the consumer scripts |
| `tools/edit.py` | exact replacements from a JSON spec, each asserted to match once, all or none applied | the record-edit helpers |
| `tools/README.md` | what each tool does, its usage and exit codes | new |
| `tools/test_tools.py` | each tool on planted input, in temporary directories | new |

The record-specific scripts, one per phase, stay out. They are a phase's
record, not a tool.

### D2. Conventions

* **The standard library only.** `from __future__ import annotations`,
  `argparse`, `def main() -> int`, `sys.exit(main())`.
* **Commands are argument lists.** Never `shell=True`. No shell pipeline is
  built.
* **The repo root** is `Path(__file__).resolve().parent.parent`. Outputs go
  to `--out`, which defaults to a directory under
  `tempfile.gettempdir()`.
* **Exit codes:** 0 clean, 1 a finding, 2 a tool error, as `scripts/` uses
  them.
* **Secrets and identifiers.**
  * No tool reads a key's value, apart from `consumer.py`, which passes the
    environment through to `go`, and never prints it.
  * `denyscan.py` reads the account name and host name at run time, and
    writes neither.
* **The load wait** uses `os.getloadavg()` and `os.cpu_count()`. It
  replaces the shell loop.

### D3. Tests, and where they run

* `tools/test_tools.py` (`unittest`, the standard library) gives each tool
  a planted input it must fail on, and a clean one it must pass:
  * a broken link;
  * an identifier-shaped string built from the test's own account name;
  * a plant that passes;
  * an edit that matches twice;
  * an output with a `--- FAIL` line.
* **`make tools-test`** runs it, locally, and the local gate runs it as
  one more check. CI does not (Q1).
* `tools/gate.py` is not run by its tests, since it builds and tests the
  whole module. Its tests cover its clone and staging step, and the PLAN
  proves it once on a planted clone.

### D4. `scripts/go-precheck.sh`

It stays in bash, as it is (Q2). The new tools are Python. The pre-add
check, which the machine-wide commit gate calls by its path, is not
changed by this record.

### Consequences

* Good, because the tools no longer depend on a session, and a lost
  scratchpad costs nothing.
* Good, because the tools run on Linux too: no `sysctl`, `awk` or BSD
  `sed`.
* Good, because each tool is tested, as the gates are.
* Neutral, because `tools/` adds Python to review, about 900 lines with
  tests, on the standard library only.
* Bad, because two directories of Python now sit side by side. The README
  of each says which is which.

### Confirmation

* Each tool's test is seen failing on its plant before the tool is trusted.
* `tools/gate.py` is proved once on a planted clone: a `vet` error and a
  wrong index status each fail their checks.
* `tools/denyscan.py` runs over `tools/` itself, with no hit.
* `make records-check`, `make lint` and the gate pass at each phase.

## Pros and Cons of the Options

### `tools/` beside `scripts/`

* Good, because CI's gates and a maintainer's tools stay apart.
* Good, because it is what the owner asked for.
* Bad, because there are two places to look. Each README points at the
  other.

### Everything in `scripts/`

* Good, because there is one place.
* Bad, because `scripts/` would no longer be "what CI runs", which
  `docs/architecture.md` and the gate self-test rely on.

### A Python package with entry points

* Good, because each tool would be a command on the path.
* Bad, because it needs a `pyproject.toml`, an install step, and a place
  outside the repo to install into, for tools run a few times a phase.

### The scratchpad

* Good, because it costs nothing now.
* Bad, because it has already lost the gate once, and the owner has asked
  for the opposite.

## More Information

### Owner questions

Answered by the owner on 2026-10-12.

* **Q1 (D3). CI.** Offered:
  * `make tools-test` joins CI's check step (recommended): the tools are
    tested on every push, at a few seconds' cost;
  * local only.

  **Answer: local only.** CI is unchanged. The local gate runs
  `make tools-test` as its 23rd check.
* **Q2 (D4). `scripts/go-precheck.sh`.** Offered:
  * port it to `scripts/go_precheck.py`, and keep `scripts/go-precheck.sh`
    as a two-line `exec` shim, because the machine-wide gate calls that
    path (recommended). The checks stay the same, the logic is Python, and
    the gate keeps checking only staged files;
  * port it and delete the `.sh`. The machine-wide gate then falls back to
    `make pre-add-check`, which checks every tracked Go file on each
    commit, and is slower;
  * leave it in bash, out of this record's scope.

  **Answer: leave it in bash.** It is out of this record's scope.

### Related

* [0026-PLAN-remediate-v1-2-debugging-pass-findings.md](0026-PLAN-remediate-v1-2-debugging-pass-findings.md):
  the gate's origin, as its rule 3.
* [0034-PLAN-degrade-forced-tool-choice-per-model.md](0034-PLAN-degrade-forced-tool-choice-per-model.md),
  Phase 4e amended: the load-sensitive tests that the load wait answers.
* The plan: [0035-PLAN-maintainer-tools-in-python.md](0035-PLAN-maintainer-tools-in-python.md).
