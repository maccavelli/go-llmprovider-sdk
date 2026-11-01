---
status: accepted
date: 2026-11-01
decision-makers: repository owner
consulted: docs/decisions/0035-MADR-maintainer-tools-in-python.md; docs/decisions/0047-MADR-keep-five-more-scratch-helpers-as-tools.md; a read of the 49 helper scripts left in the agent's session scratchpad from the work on records 0031 (gobble-cli), 0036 (gobble-cli), 0063 and 0066, on 2026-11-01
informed: maintainers and agents working in this repository
---

<!-- markdownlint-disable MD013 -->

# Keep Four More Scratch Helpers: Red and Green Runs of Named Tests, a Leak Finder, Open Records, and a Markdown Gate

## Context and Problem Statement

[0035-MADR](0035-MADR-maintainer-tools-in-python.md) and
[0047-MADR](0047-MADR-keep-five-more-scratch-helpers-as-tools.md) moved into
`tools/` the helpers a session scratchpad held, because the scratchpad is
pruned and "a helper that two plans needed is a tool, or it is lost"
(0047-MADR, Decision Drivers). The work on
[0066-PLAN](0066-PLAN-wizard-v1-24-debugging-pass-findings.md), and the end of
0063's and two gobble-cli records', left 49 more scripts. On 2026-11-01 each
was read in full and sorted **[read]**:

| Class | Scripts | Why not a tool |
| :--- | ---: | :--- |
| A. Assert-once replaces and appends on records, docs and CHANGELOG, each for one phase | 35 | `tools/edit.py` already does this (0047-MADR D2); the text is the phase's, written once |
| B. Covered by a tool | 4 | a copy of the tree (`_common.scratch_clone`), two claim locators (`tools/claims.py`), a dependent's bump (`tools/consumer.py --dependent`) |
| C. One-off probes, or obsolete | 4 | a gobble-cli fix probe; `./wizard/`'s guard on a scratch `TestMain`, in the tree since 0066-PLAN Phase 6; 0063's log lines; the session's own remote files |
| D. Candidates | 6 | below |

The six candidates, and what no tool does:

1. **A named test seen red at an earlier revision.** `red0066.py` cloned the
   repository at a revision, copied the new test files from the tree,
   dropped the test functions whose subject did not exist there, and ran
   the named tests. 0066-PLAN's Phases 1, 2, 4, 5 and 6 used it. 0061-PLAN
   did the same by hand: "Every new test is seen to fail on `d9b1910`'s
   code" (`0061-PLAN-consumer-gaps-from-gobble-issue-9.md:54`) **[read]**.
   `tools/plant.py` plants a break in the tree's code; it cannot run new
   tests against old code.
2. **Named tests seen green, none missing.** `verify0066.py` ran the 42
   tests 0066-MADR's Verification table names and required a `--- PASS`
   line for each, since `go test -run` passes when a mistyped name matches
   nothing **[read]**. The PLAN's close recorded it: "42 of 42 `--- PASS`,
   none missing". It is the green half of item 1: the same names, at
   another revision.
3. **Which test sends a request off the machine.** `internal/localonly`'s
   guard names each host, never the test
   (`internal/localonly/localonly.go:114`) **[read]**. 0065-MADR found its
   leaks by hand, and said "the next leak is found by hand, as these were"
   (`0065-MADR-tests-reach-no-outside-host.md:152`) **[read]**. 0066's leak
   showed only on Windows CI. `leakhunt0066.py` ran `./wizard/`'s tests in
   chunks of 25 on the Windows dev host, then one by one in a chunk that
   leaked; and `leakhunt0066b.py`, when the chunk leaked only as a whole,
   left each test out in turn, twice, to find the two tests the leak
   needed (0066-PLAN, Phase 8 deviation) **[read]**.
4. **The records still open.** `open_records.py` listed each record whose
   status is not final, with its date **[read]**. `tools/records.py` sets a
   status and indexes a record (`tools/records.py:166`), and lists nothing
   **[read]**.
5. **The checks a record edit needs.** `checks0066.py` ran
   `records-check`, `links-check`, `disclosure-check`, `claims.py`,
   `recordlint.py --changed`, `denyscan.py --changed` and `linkcheck.py`
   after each record edit **[read]**. 0062-PLAN lists three of them by hand
   (`0062-PLAN-learn-thinking-parameters-from-refusals.md:84-86`)
   **[read]**. The local gate runs every check, Go tests included
   (`tools/gate.py:70`), which a Markdown-only change does not need; CI
   runs `MARKDOWN_GATES` for one (`scripts/test_gates.py:533`) **[read]**.

So four jobs, items 1 and 2 being one: a run of named tests that must all
fail, or all pass, at a revision; a leak finder; a list of open records;
and the Markdown gate.

## Decision Drivers

* A helper that two plans needed is a tool (0047-MADR). Items 1 and 5
  served two plans each (0061 and 0066; 0062 and 0066), item 3 two
  (0065 by hand, 0066). Items 2 and 4 served one plan each, and recur:
  a green run at every PLAN's close, the open records whenever the open
  work is asked.
* 0035-MADR's rules hold for every tool: Python 3 on the standard library;
  the root found from the tool's own file; no host, account or machine
  path, anything outside the repository an argument; exit codes 0 clean, 1
  a finding, 2 a tool error; a test in `tools/test_tools.py` that fails it
  on a planted input.
* The tree is never written: a run at a revision, a red run and a leak hunt
  each work on a scratch clone, or on a checkout the caller names.
* One tool per job, and an extension keeps every command line that works
  today working (0047-MADR).

## Considered Options

* **A. Two new tools and two extensions:** `tools/redgreen.py` (items 1
  and 2), `tools/leakfind.py` (item 3), `records.py open` (item 4),
  `gate.py --markdown` (item 5).
* **B. Leave them in the scratchpad.**
* **C. Five new tools,** one per item, red and green apart.
* **D. Name the test in the guard,** in Go, instead of a leak finder.

## Decision Outcome

Chosen option: "A", because each job gets one tool under 0035-MADR's
rules, red and green share their spec and their clone, and the two
extensions sit in the tools whose job they extend.

### D1. `tools/redgreen.py`: named tests, all red or all green, at a revision

* **Usage:** `tools/redgreen.py SPEC.json --expect fail|pass [--rev REV]`.
* **The spec:** `{"tests": {"./pkg/": ["TestName", ...]}, "files":
  ["pkg/x_test.go", ...], "drop": ["TestName", ...]}`. `files` and `drop`
  are optional.
* **Where it runs:** with `--rev`, a shared clone of the repository at
  `REV`, with each of `files` copied from the working tree, and each
  function in `drop` removed from them with its doc comment, so a test
  whose subject does not exist at `REV` does not stop the package from
  building. A name in `drop` that matches no function, or more than one,
  is a tool error. Without `--rev`, `_common.scratch_clone` of the working
  tree; `files` and `drop` are then a tool error.
* **The run:** each package once, `go test -count=1 -v -run '^(names)$'`.
* **`--expect pass`:** each name must print `--- PASS: name`. **`--expect
  fail`:** each must print `--- FAIL: name`. A name with neither line, or
  a package that does not build, is a finding: a red run needs the test to
  run, and fail on its assertion.
* **Output:** per package, the exit status and each name's result; each
  package's full output in the tool's output directory.

### D2. `tools/leakfind.py`: the tests behind a leak

* **Usage:** `tools/leakfind.py PACKAGE [--chunk N] [--runs N] [--ssh ALIAS
  --dir DIR]`.
* **Only behind the guard:** the package must have a `_test.go` file that
  calls `localonly.Main`, in the tree; otherwise a tool error.
* **The search:** the package's tests, by `go test -list`, run in chunks
  of `N` (25). A run leaks when its output holds the guard's message
  (`localonly.Report`'s text). For each chunk that leaks, each test alone;
  a test that leaks alone is reported with its hosts. When a chunk leaks
  and none of its tests leaks alone, each test is left out in turn, every
  run `--runs` times (2); a test whose absence stops the leak in every run
  is one the leak needs, and those tests are then run together to confirm
  it. *Annotated 2026-11-01 (A1): the whole package runs first; a clean
  whole run is "no leak", and a whole run that leaks when no chunk does is
  a finding.*
* **Where it runs:** on a scratch clone of the working tree; or, with
  `--ssh`, in the checkout `DIR` on the host `ALIAS`, through `ssh -o
  BatchMode=yes -o ConnectTimeout=5`, for a leak that shows on one
  platform only. The tool never copies to the host, and the alias and the
  directory are arguments only.
* **Exit:** 1 when a leak is found, 0 when none is, 2 on a tool error.

### D3. `tools/records.py open`

* Lists each record whose status is not final, a MADR `proposed`, a PLAN
  `proposed` or `in-progress`, one per line: status, date, path, in number
  order.
* The open statuses are checked against `scripts/check_records.py`'s lists
  (`scripts/check_records.py:42-43`), which it imports already. It writes
  nothing, and exits 0.

### D4. `tools/gate.py --markdown`

* Runs, on the scratch clone, the gate's own checks whose names are in
  `scripts/test_gates.py`'s `MARKDOWN_GATES`, imported so the two never
  differ, then `deny-scan`; and, in the working tree, which they only read,
  `tools/recordlint.py --changed` and `tools/claims.py --changed`, since
  both compare the tree with `HEAD`, and the clone's `HEAD` already holds
  the changes. *Annotated 2026-11-01 (A2): the claims check is given the
  changed records by name, and is clean when there are none.*
* **A Markdown change only:** a changed file that is not `.md` is a
  finding, exit 1, and nothing runs: such a change needs the full gate.
* ~~It starts without the load wait: none of its checks runs a Go test.~~
  *Annotated 2026-11-01 (A2): false; `gate-selftest` runs Go tests, so the
  mode waits for the load as the full gate does.*
* The full gate is unchanged.

### D5. The documents

* `tools/README.md`'s table gains the two tools and the two modes, and
  `gate.py`'s row the mode.
* `AGENTS.md`'s "Tools" list names them.
* `docs/architecture.md`, where it lists `tools/`.

### Consequences

* Good, because a red run at an earlier revision, the close's named-test
  run, a leak hunt and a record edit's checks become calls to tested
  tools, where each plan rebuilt them.
* Good, because a mistyped test name fails a green run, where `go test
  -run` passes it.
* Good, because the next leak is found by a tool, which 0065-MADR said
  would be found by hand.
* Neutral, because `leakfind.py --ssh` runs commands on a dev host, as an
  agent's own ssh does today, and only in a checkout the caller names.
* Bad, because `tools/` grows from 14 tools to 16, and two tools gain a
  mode.

### Confirmation

* Each tool and mode has a test in `tools/test_tools.py` that fails it on
  a planted input and passes a clean one; `make tools-test` passes.
* Each is seen to fail first, by `tools/plant.py` on a scratch clone, with
  `make tools-test` as the command.
* First uses:
  * `redgreen.py --expect pass` on 0066-MADR's 42 names, 42 green;
  * `redgreen.py --expect fail --rev 5c11035` with
    `wizard/notices_0066_test.go` copied in: its four tests red, as
    0066-PLAN's Phase 5 record says they were;
  * `leakfind.py ./wizard/` on the tree: no leak;
  * `records.py open` lists what `check_records.py`'s statuses say;
  * `gate.py --markdown` on this pair: every check passes.

## Pros and Cons of the Options

### A. Two new tools and two extensions

* Good, because red and green share one spec, one clone and one parser.
* Good, because `records.py` and `gate.py` keep every existing command.
* Bad, because two tools gain a mode to test.

### B. Leave them in the scratchpad

* Good, because nothing changes.
* Bad, because the scratchpad is pruned, and 0061's red runs and 0065's
  leak hunt were already rebuilt by hand once.

### C. Five new tools

* Good, because each name says one thing.
* Bad, because a red tool and a green tool would duplicate the spec, the
  clone and the result parser, and a Markdown gate apart from `gate.py`
  would copy its check list.

### D. Name the test in the guard

* Good, because a leak would name its test in CI's own output.
* Bad, because the guard is a proxy, and a request carries no test's name;
  0066's leak came from a goroutine its test left running, which only a
  later test's lifetime let through, so no attribution at the request
  would have named the cause.
* Bad, because it changes the guard, in Go, for a maintainer's job.

## Owner questions

Answered 2026-11-01, each as recommended; each answer is in the decisions
above.

* **Q1.** Is a package that does not build at `REV` a red result? **The
  recommendation: no,** a finding (D1): a test that does not compile has
  not shown that its assertion catches the bug; `drop` makes it build.
  **The owner: no, as recommended.**
* **Q2.** Should `leakfind.py` take `--ssh`? **The recommendation: yes**
  (D2), since 0066's leak showed on Windows only; the host is an argument,
  never in the tool. **The owner: yes, as recommended.**
* **Q3.** Should `gate.py --markdown` refuse a change with a non-Markdown
  file? **The recommendation: yes** (D4), so a code change is never gated
  by the Markdown checks alone. **The owner: yes, as recommended.**

## More Information

* The classification's evidence is the scratch scripts themselves, which
  are not committed; this record names them by their scratch names only.
* Related: [0035-MADR](0035-MADR-maintainer-tools-in-python.md) (the
  tools' rules), [0047-MADR](0047-MADR-keep-five-more-scratch-helpers-as-tools.md)
  (the last such move),
  [0065-MADR](0065-MADR-tests-reach-no-outside-host.md) (the guard), and
  [0066-PLAN](0066-PLAN-wizard-v1-24-debugging-pass-findings.md) (the red
  runs, the leak hunt, the close).

### A1 (2026-11-01): the leak finder runs the whole package first

* **Found** in 0067-PLAN Phase 2, before the phase was recorded: D2's
  search starts with chunks. A leak that needs two tests in two chunks
  never shows in a chunk's run, so D2 as written reports "no leak", exit 0,
  for it. 0066's leak was of that kind in part: it showed only when a later
  test kept the process running (0066-PLAN, Phase 8 deviation).
* **The owner chose (2026-11-01):** the tool runs the whole package first.
  A whole run that does not leak is "no leak", exit 0. A whole run that
  leaks when no chunk does is exit 1, with the advice to run with a larger
  `--chunk`. The cost is up to `--runs` more runs of the package per
  search; the first use on `./wizard/`, whose two whole runs were clean,
  took 15 s in all. D2 is annotated; no other decision changes.

### A2 (2026-11-01): the Markdown gate waits for the load, and gives the claims check its records

* **Found** in 0067-PLAN Phase 5, at `gate.py --markdown`'s first use, on
  `AGENTS.md` and `docs/architecture.md` alone: "gate: 10 of 11 passed",
  exit 1, in 5 min 8 s **[probed]**.
  * `claims` exited 2, "claims: no record given": `tools/claims.py
    --changed` treats a change with no record as a tool error, where
    `tools/recordlint.py --changed` reports "0 record(s)", exit 0.
  * Timed one by one on a scratch clone, at a load average of 65:
    `gate-selftest` 308.7 s; the other ten 14.4 s together **[probed]**.
    `gate-selftest` runs every CI gate on planted trees
    (`scripts/test_gates.py:68`), `coverage-check`'s `go test -cover
    ./...` (`scripts/check_coverage.py:69`) and `scripts/go-precheck.sh`
    (`scripts/test_gates.py:111`) among them **[read]**. So D4's "none of
    its checks runs a Go test" was false, and the reason for no load wait
    with it: the full gate waits because some tests time out on a loaded
    host (0034-PLAN Phase 4e).
* **The owner chose (2026-11-01):**
  * the mode waits for the load as the full gate does, with the same
    `--max-load` and `--timeout`;
  * the claims check runs `tools/claims.py` on the changed records,
    read from the clone's commit, and is clean, "no record changed", when
    there are none. `tools/claims.py` is unchanged.
* D4 is annotated; no other decision changes.

### A3 (2026-11-01): the claims check takes only records with an evidence index

* **Found** in 0067-PLAN Phase 5, running A2's fix on that phase's
  Markdown alone: `claims: rc=1`, with "0067-MADR…: 0 of 0 pass (no
  evidence rows)" and "0067-PLAN…: 14 of 14 pass" **[probed]**.
  `tools/claims.py` counts a record with no evidence rows as a finding
  (`tools/claims.py:310`) **[read]**, and 109 of the 141 records carry no
  `<!-- claims: … -->` marker **[probed]**, so A2's check failed any change
  that touched one of them.
* **The owner chose (2026-11-01):** the check passes `tools/claims.py`
  only the changed records whose text matches its own `MARKER`
  (`tools/claims.py:70`), imported; when there are none it is clean, "no
  record with an evidence index changed". `tools/claims.py` is unchanged.
  D4's A2 annotation stands, narrowed so.
