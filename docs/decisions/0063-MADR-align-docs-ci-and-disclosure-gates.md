---
status: proposed
date: 2026-10-27
---
# Align docs-only CI and disclosure gates

## Context and Problem Statement

This repository needs a focused Docs run for documentation-only pushes and
pull requests. The earlier local decision is
0052-MADR-skip-full-ci-for-docs-only-pushes.md. This new decision adds a
common quality and disclosure contract across the seven repositories.

Observed in this checkout on 2026-10-25: `ci.yml` already ignores `**.md` on
pushes and pull requests; `docs.yml` runs parity, records, tables, changelog
and gate-selftest. Neither workflow runs Markdownlint or the link checker;
`tools/linkcheck.py` and `tools/denyscan.py` are local tools. `gate-selftest`
installs golangci-lint in the docs job because its precheck test needs it.

The pinned Markdownlint command currently exits 1 with 380 findings in two
unchanged reports,
`docs/reports/0011-REPORT-provider-source-compatibility-audit.md` and
`docs/reports/0054-REPORT-heuristics-and-performance-opportunities.md`.
Enabling it as a CI gate therefore requires formatting those reports while
preserving their content.
*Annotated 2026-10-27: superseded by the re-assessment below. The
findings are now 406, in three reports; 379 are MD004, the records' own
`*` list markers, which 0047-MADR A2 keeps.*

Across the repositories, the common documentation contract is pinned
Markdownlint, numbered-record and relative-link validation, a portable
disclosure check for real home paths in tracked text, and any
repository-specific check that reads Markdown. The owner's global pre-push
disclosure guard uses private deny patterns and remains mandatory before a
real push. Go lint, race, vulnerability, build and release checks remain in
full CI.

## Decision Drivers

* A documentation-only change has a successful, visible Docs run.
* A Markdown file read by code must run that reader check in Docs or full CI.
* Non-document changes, tags, schedules and manual runs keep full CI behavior.
* Both workflows use the same documentation commands and pins.
* CI disclosure checks work without private host data; the pre-push guard
  still protects publication.

## Considered Options

* Keep the current workflow split and existing checks.
* Align the documentation gates and add a portable disclosure check, with a
  path and gate parity test.
* Detect changed paths inside one full workflow and conditionally skip
  expensive jobs.

## Decision Outcome

Chosen option: "Align the documentation gates and add a portable disclosure
check, with a path and gate parity test", because it gives both workflows the
same common documentation checks, covers local Markdown readers, and makes
filter drift observable. This is proposed for the owner; it does not approve
CI implementation.

### Consequences

* Good, because documentation-only changes can complete after focused checks.
* Good, because records, links, Markdown style and generic disclosure faults
  are checked consistently.
* Neutral, because a mixed change may start both workflows.
* Bad, because private identity patterns remain enforceable only by the local
  pre-push guard; CI has narrower coverage.
* Bad, because path-filtered full-CI jobs may be pending for a required
  pull-request check; required-status settings need review before rollout.

### Confirmation

The paired
[0063-PLAN-align-docs-ci-and-disclosure-gates.md](0063-PLAN-align-docs-ci-and-disclosure-gates.md)
requires failing planted inputs for each new gate, static workflow checks, and
observed Docs-only and full-CI runs after an approved push. Local checks alone
do not prove GitHub path-filter behavior.

## Pros and Cons of the Options

### Keep the current workflow split and existing checks

* Good, because it requires little work.
* Bad, because the Docs job omits at least one common gate.

### Align the documentation gates and add a portable disclosure check, with a path and gate parity test

* Good, because it establishes the same baseline while preserving local
  Markdown readers.
* Bad, because it adds a maintained scanner and parity test and requires
  formatting two existing reports before Markdownlint can gate CI.
  *Annotated 2026-10-27: with the re-assessment's file set, no report
  changes.*

### Detect changed paths inside one full workflow and conditionally skip expensive jobs

* Good, because a single workflow can expose one stable status.
* Bad, because conditional jobs add control logic and can obscure whether
  documents passed.

## More Information

Keep the existing filters and five docs targets. Add a pinned `make
markdownlint` target and the existing link checker to both Linux CI and Docs;
preserve the docs job's golangci-lint prerequisite and its pin parity test.
Add a portable tracked-text disclosure target to both workflows, reusing the
tested `tools/denyscan.py` home-path logic but without relying on the runner's
account or host names; keep the owner-specific pre-push guard local. Extend
`scripts/test_gates.py` to assert that the docs-sensitive target set and pins
in both workflows agree. Update `AGENTS.md` and `docs/architecture.md` to
state the common contract.
*Annotated 2026-10-27: the link and disclosure checks CI runs live in
`scripts/`, not `tools/`; see the re-assessment below.*

This fleet alignment decision cites
0052-MADR-skip-full-ci-for-docs-only-pushes.md without rewriting its history.
The implementation is
[0063-PLAN-align-docs-ci-and-disclosure-gates.md](0063-PLAN-align-docs-ci-and-disclosure-gates.md).

## Re-assessment 2026-10-27

Asked by the owner to re-assess the repository, its code and its
dependencies against this record before deciding it. Measured at
`5eff09e`. The status stays `proposed`; the chosen option is unchanged.
Four facts change how it is carried out (R1 to R4).

### Facts that hold

* **The workflows are as observed.** Neither changed after `e68be51`
  (2026-10-21). `ci.yml` ignores `**.md` on push and pull request; its
  triggers are push (`main`, `v*` tags), pull request and a weekly
  schedule, with no `workflow_dispatch`. `docs.yml` runs on `**.md` and
  itself, installs golangci-lint `v2.14.0` as `ci.yml` does, and runs
  `make parity-check records-check tables-check changelog-check
  gate-selftest`. Neither runs Markdownlint, a link check or a disclosure
  check. `actionlint` v1.7.12 reports nothing on either.
* **Every reader of Markdown runs in Docs.** `parity-check` reads
  `docs/guides/migrating-from-mcplib.md`; `records-check` the records and
  `docs/README.md`; `tables-check` `AGENTS.md`, `docs/architecture.md`,
  `docs/guides/api-standards.md` and the records; `changelog-check`
  `CHANGELOG.md`; `gate-selftest` plants strings in the docs. The gates
  only full CI runs (`dep-check`, `coverage-check`, `api-check`,
  `generate-check`, `fixtures-check`, `docs-check`) read no Markdown:
  `fixtures-check` reads files under `testdata`, none of them Markdown,
  and `docs-check` reads Go. No Go file reads a Markdown file. Of the 149
  tracked Markdown files, four are outside `docs/` and the root:
  `.claude/rules/madr-and-plan-skill.md`,
  `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`
  and `tools/README.md`.
* **Routing works as 0052-MADR decided.** The docs-only push of `04dc55a`
  started Docs alone (run `40586997013`); the tag `v1.24.1` ran `ci.yml`
  in full (run `40588642904`).
* **No status check is required.** `main` has no branch protection and no
  ruleset, so a path-filtered workflow cannot leave a required check
  pending. The Consequence about required checks applies only if
  protection is added.
* **The fleet.** Seven repositories hold this decision: acp-go-sdk2 and
  go-selfupdate-lib `accepted` and `complete`, prepare-commit-msg
  `accepted` and `in-progress`, and gobble-cli, magic-cli-remote,
  go-tui-lib and this one `proposed`. go-selfupdate-lib's
  `docs/decisions/0036-MADR-align-docs-ci-and-disclosure-gates.md`
  runs `MARKDOWNLINT ?= npx --yes markdownlint-cli2@0.23.2` from both
  workflows; its Docs run on ubuntu-24.04 printed `markdownlint-cli2
  v0.23.2 (markdownlint v0.41.1)` and "0 issues". That is the version on
  this host, and the one 0052-PLAN cites.

### R1. Markdownlint: the file set, not the reports

* **Measured.** `markdownlint-cli2` with the configuration's own globs
  lints 18 files and exits 1: "406 issues in 3 files". By file:
  `0054-REPORT` 191 (MD004), `0011-REPORT` 189 (162 MD004, 26 MD032, 1
  MD013), and `0062-REPORT-thinking-refusals.md`, written after this
  record, 26 (MD004). 379 of the 406 are MD004: the configuration pins
  `-` markers, and records use `*`, which
  `0047-MADR-keep-five-more-scratch-helpers-as-tools.md` A2 keeps
  (`tools/recordlint.py` lints records with MD004 `consistent`). Under
  the records' conventions only `0011-REPORT` has findings: 26 MD032 and
  1 MD013.
* **The local gate already lints a set that is clean.**
  `tools/gate.py` runs `markdownlint-cli2 --no-globs` on every tracked
  Markdown file outside `docs/decisions/` and `docs/reports/`, and passes.
  *Annotated 2026-10-27, measured at `eaaf709`:* every tracked or
  untracked one, 12 files, all tracked; "0 issues in 0 files". They
  include `.claude/rules/madr-and-plan-skill.md` and
  `.grok/rules/madr-plan-before-mutating-work.md`, which the
  configuration's globs skip, since their names match `!**/*madr*.md`.
* **So formatting the reports is not the way.** Changing their markers
  to `-` would contradict A2. Recommended: `make markdownlint` lints
  `tools/gate.py`'s file set, with the pin above, in both workflows and the
  local gate, one list; the records stay under `tools/recordlint.py`, edit
  by edit. Rejected: the configuration's globs with the reports
  reformatted (against A2), and the configuration's globs with the
  records' conventions for the reports, which needs a second
  configuration and the 27 fixes in `0011-REPORT` for a file no edit
  changes.

### R2. The CI checks live in `scripts/`

`0035-MADR-maintainer-tools-in-python.md` keeps `scripts/` "the gates CI
runs" and `tools/` the maintainer tools, and its Q1, answered by the
owner, keeps `make tools-test` local. Running `tools/linkcheck.py` and
`tools/denyscan.py` from CI, as More Information says, would put tools CI
does not test into CI. Recommended: the link check and the disclosure
check CI runs are `scripts/` gates, each a `make` target (`links-check`,
`disclosure-check`; `docs-check` is already the API audit's), with their
plants in `scripts/test_gates.py`, which `gate-selftest` runs in both
workflows. `tools/linkcheck.py`, `tools/denyscan.py` and `tools/gate.py`
use the same code rather than a second copy.

### R3. The disclosure check covers Windows and reads nothing of the host

`tools/denyscan.py`'s home-path pattern is `/(Users|home)/` followed by
anything but `<user>`: it has no Windows form. Its other checks read the
host's account and host names at run time, which on a runner would be the
runner's. The portable check takes the home-path rule alone, adds
`C:\Users\…` in both slash forms, exempts the `<user>` placeholder, and
reads no host name. Measured with that rule over the 1,434 tracked files:
0 hits, and every file read as UTF-8.

### R4. The contract check uses the standard library

`scripts/` is Python's standard library only, which has no YAML parser.
go-selfupdate-lib's check uses PyYAML, a pinned dependency this repository
does not have. The existing check of 0052-MADR A2
(`scripts/test_gates.py`, `installs_before_selftest`) reads the workflows'
lines. Recommended: the contract check reads the triggers, filters and
`make` lines the same way, and `actionlint` validates the YAML, run
locally at v1.7.12, as the PLAN's steps already run it.

### The owner's answers (2026-10-27)

Each was put with three options, the recommended one first. The owner
chose the recommended option for all four:

* **R1, "Gate's set".** `make markdownlint` lints every Markdown file
  outside `docs/decisions/` and `docs/reports/`, the list `tools/gate.py`
  lints, moved to `scripts/`. Not chosen: the configuration's globs with the
  records' rules for the reports, and the configuration's globs with the
  reports reformatted.
* **R2, "scripts/, tools reuse".** `scripts/check_links.py` and
  `scripts/check_disclosure.py`, as `make links-check` and `make
  disclosure-check`, planted in `scripts/test_gates.py`; the tools import
  them. Not chosen: CI running `tools/`, which would amend 0035-MADR and its
  Q1, and separate copies in `scripts/`.
* **R3, "Unix + Windows homes".** Not chosen: applying the private deny
  list when present, and the Unix rule alone.
* **R4, "Stdlib lines + local actionlint".** Not chosen: `actionlint` run
  in CI as well, and PyYAML.

The record stays `proposed` until the owner accepts it.
