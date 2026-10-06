---
status: proposed
date: 2026-10-06
decision-makers: repository owner
consulted: 0002-MADR-migrate-llmprovider-from-mcplib.md (§10–§11), 0021-MADR-harden-and-tune-after-the-v1-1-review.md (Z9)
informed: maintainers of go-llmprovider-sdk's records
---

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Check the Decision Records Mechanically, and Correct the Two Stale PLAN Statuses

## Context and Problem Statement

`AGENTS.md` ("Records") sets the rules for `docs/decisions/` and
`docs/reports/`:

* the filename pattern and the directory for each kind;
* one number sequence across all four kinds;
* a MADR and its PLAN share a number;
* `docs/README.md` indexes every record, updated in the same change.

The `madr-and-plan-writing` skill adds:

* the status vocabulary, where a PLAN is `proposed`, `in-progress`,
  `complete` or `superseded`;
* that `scripts/check_records.py --next`, where a repository has it, is the
  only source of the next number.

This repository has no such script. People follow the rules by hand, and
nothing checks them.

**Measured 2026-10-06,** by a read-only scan of all 51 records at `64b82d5`:

* **Two PLANs carry a decision's status.**
  `0004-PLAN-add-gateway-llm-providers.md` and
  `0005-PLAN-canonicalize-llm-provider-configuration.md` say
  `status: accepted`, as their index rows do. They were migrated from
  `mcplib` (its 0003 and 0004) by
  `0002-MADR-migrate-llmprovider-from-mcplib.md`, which set the transferred
  plans' statuses one by one but did not set these two.
* **They were executed.** `mcplib`'s history shows 0004's phases merged at
  `50ac165`, and 0005's at `29ef039` and `5786e6d`. Each PLAN's deviation
  log runs to its last phase (0004's D1–D4, 0005's D1–D7), and
  `0004-MADR-add-gateway-llm-providers.md` carries a "Status after
  implementation (2026-08-29)" note. Their acceptance criteria name
  `mcplib`'s tree of 2026-08-29 and three consumer repositories, so they
  cannot be re-run here.
* **The index count had drifted,** found and fixed by hand while writing
  `0022-MADR-gemini-live-instructions-test.md`: it said "43 records" and
  listed 47. Nothing would have caught it.
* **Everything else holds:**
  * every name matches its pattern and directory;
  * the numbers run 0001–0023 with no gap;
  * every PLAN names its MADR;
  * the index has one row per file, and its statuses match.

  Three REPORTs carry no `status`, and their rows say "observation", a
  convention to keep. `0012-MADR-conform-providers-to-reference-clients.md`
  has six PLANs, each with its own slug, which the skill allows.

What should keep the records consistent, and what are the two PLANs'
statuses?

## Decision Drivers

* A rule that matters, and that can be checked, is checked by a machine:
  the index count drifted once already.
* The skill names `scripts/check_records.py --next` as the source of the
  next number. Having it removes a hand scan from every new record.
* The repository's other rules are checked in CI with a planted-breach
  self-test (`make gate-selftest`, 0021-MADR Z9). A new check should be
  built the same way.
* Standard library only (`AGENTS.md`, "Dependencies"); the existing
  `scripts/` are Python and the stdlib.
* A status must rest on evidence. History is not rewritten: a dated note
  explains the correction.

## Considered Options

* A checker in `scripts/`, run by `make records-check` and CI, with a self-test breach
* The same checker, run by hand only
* Generate the index from the records
* Correct the two statuses by hand, and add no tooling

## Decision Outcome

Chosen option: "A checker in `scripts/`, run by `make records-check` and CI,
with a self-test breach", because it catches every drift found so far, and
gives the skill its `--next`. It also follows how this repository already
runs its gates.

**`scripts/check_records.py`** (Python, the standard library only) checks:

* **Names:** `NNNN-{MADR,PLAN}-slug.md` in `docs/decisions/`, and
  `NNNN-{REPORT,GATES}-slug.md` in `docs/reports/`. The slug is lowercase
  kebab-case. No other `.md` file sits in either directory.
* **Numbers:** at most one MADR per number.
* **Statuses:**
  * a MADR's front-matter `status` is `proposed`, `accepted`, `rejected`,
    `deprecated`, or begins `superseded`;
  * a PLAN's is `proposed`, `in-progress`, `complete` or `superseded`;
  * a REPORT or GATES has none, or `observation`.
* **Pairs:**
  * every PLAN has a MADR of the same number;
  * a PLAN that is its number's only one has its MADR's slug;
  * a PLAN's `associated-madr` or `parent-madr` names an existing MADR of
    its number.
* **The index:**
  * `docs/README.md` has exactly one row per record, whose number, kind and
    link match the file;
  * the row's status is the file's, or "observation" for a report with
    none;
  * its "N records." line equals the count.
* **`--next`** prints the next number: the highest across all four kinds,
  plus one.

Each problem is printed as `path: problem`. The exit code is 1 on any
problem.

**Wiring:**

* `make records-check`;
* CI's gate step gains it;
* `scripts/test_gates.py` gains a breach: a PLAN with `status: accepted` on
  a scratch copy, which must fail with exit 1;
* `AGENTS.md` ("Records") names `make records-check`, and
  `scripts/check_records.py --next` for the next number.

**The two statuses:** both PLANs become `complete`. Each gains a dated note
naming the evidence (the merge commits in `mcplib`, the deviation logs, and
0004's "Status after implementation"). It also says the acceptance criteria
were not re-run: they describe `mcplib`'s tree. The index rows follow.

### Consequences

* Good, because the drift found by hand, and the status error found today,
  now fail in CI.
* Good, because `--next` replaces a hand scan of four kinds in two
  directories.
* Good, because the checker is self-tested like the other gates.
* Neutral, because index titles stay hand-written: the checker checks the
  links, numbers, kinds and statuses, not the wording.
* Bad, because every record change must keep the index exact, or CI fails.
  That is the rule `AGENTS.md` already states.
* Bad, because the two `complete` statuses rest on history and records, not
  a re-run, which the notes say.

### Confirmation

* `make records-check` exits 1 on today's tree, naming exactly the two
  PLANs. It exits 0 after their correction.
* `scripts/check_records.py --next` prints the next free number.
* The self-test's new breach fails with exit 1, and `make gate-selftest`
  passes.
* The repository gate is clean (0025-PLAN).

## Pros and Cons of the Options

### A checker in `scripts/`, run by `make records-check` and CI, with a self-test breach

* Good, because it checks every rule found to drift, on every push.
* Good, because it is built like the five existing gates.
* Bad, because it adds a CI step and one more self-test case, about a
  second each.

### The same checker, run by hand only

* Good, because it adds no CI step.
* Bad, because a check people must remember is the hand process that let
  the count drift.

### Generate the index from the records

* Good, because a generated index cannot drift.
* Bad, because the index's titles and its other tables are hand-written,
  and a generator that owns part of the file needs markers and its own
  rules. It is a bigger change than today's drift calls for.
* Neutral, because a checker now does not rule out a generator later.

### Correct the two statuses by hand, and add no tooling

* Good, because it is two lines in two records and two in the index.
* Bad, because the next drift is found the same way: by chance.

## More Information

* **Relationship:**
  * corrects what `0002-MADR-migrate-llmprovider-from-mcplib.md`'s
    transfer left unset for `0004-PLAN-add-gateway-llm-providers.md` and
    `0005-PLAN-canonicalize-llm-provider-configuration.md`;
  * follows 0021-MADR Z9's self-tested gates.
* **Plan:** [0025-PLAN-records-consistency-check.md](0025-PLAN-records-consistency-check.md).
* **The scan** was a scratch script, read-only, on `64b82d5`.
