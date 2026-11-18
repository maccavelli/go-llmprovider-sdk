---
status: accepted
date: 2026-11-18
decision-makers: repository owner
consulted: "the owner's rulings of 2026-11-18 (their labels are quoted in the Context); golangci-lint v2.14.0 and v2.15.0, each built with go1.27.4, run on a git archive copy of this repository at fd986c4 on 2026-11-18; 0029-MADR-go-1-27-2-for-standard-library-fixes.md, which decided the current pin"
informed: gobble-cli and the other repositories that move to the same linter
---

<!-- markdownlint-disable MD013 MD024 -->

# Pin golangci-lint v2.15.0, and Drop the Three nolint Directives It Makes Dead

## Context and Problem Statement

CI, the `Makefile`, the precheck script and the Docs workflow pin golangci-lint
`v2.14.0` (C1 to C4). Its pin was decided by
[0029-MADR-go-1-27-2-for-standard-library-fixes.md](0029-MADR-go-1-27-2-for-standard-library-fixes.md)
(its amendment of 2026-10-08), moved from `v2.13.1` because that release could
not read Go 1.27.2's export data. Later records name `v2.14.0` without
deciding it: they measure it under a newer Go (0055-MADR, 0077-MADR) or
repeat CI's install line (0052-MADR, 0063-MADR).

The owner ruled on 2026-11-18, in the owner's own words, "standardize on
golangci-lint v2.15.0". Asked how the moves proceed in the repositories
still pinned lower, the owner chose **"One short record pair per repository,
approved now through CI (Recommended)"**: a record, the pin moved everywhere
it appears, the findings fixed in the same change, and no release.

### Facts (observed 2026-11-18)

| # | Observation | Source |
| :--- | :--- | :--- |
| F1 | v2.15.0, on `git archive HEAD` (fd986c4), for the host and for `GOOS=windows`, reports 3 issues, all `nolintlint`: `llmprovider/internal/transport/offline.go:21:52` and `:24:54`, and `unsendable.go:48:65`, each "directive `//nolint:errcheck // only whether the chain holds one matters` is unused for linter "errcheck"". The nested module `authcontract` has none | the run |
| F2 | v2.14.0 on the same copy reports none | the run |
| F3 | With the three directives removed, v2.14.0 reports `Error return value of errors.AsType is not checked (errcheck)` at the three sites, and v2.15.0 reports 0 issues, for the host and for `GOOS=windows` | the run on a scratch copy |
| F4 | Each call is `errors.AsType[T](err)`, which returns a value and a bool. At each site the bool, `ok`, is read: two `if` conditions and one `return ok`. No error is left unchecked. Why v2.15.0 stops flagging a call of this shape is not established here: the observed behaviour is F3 | the three sites |
| F5 | `go.mod` says `go 1.27.4` with no `toolchain` line; `authcontract/go.mod` the same | the files |

The pin appears in `Makefile` (the install hint), `scripts/go-precheck.sh`
(the install hint), `.github/workflows/ci.yml` and `.github/workflows/docs.yml`
(C1 to C4). `docs/architecture.md` and `AGENTS.md` say "pinned golangci-lint"
without a number.

## Considered Options

* A. Move the pin to `v2.15.0` and remove the three directives in the same change.
* B. Move the pin and leave the directives, or add a config exclusion.
* C. Stay at `v2.14.0`.

## Decision Outcome

Chosen option: "A", because it is the owner's ruling, and because the
directives are needed by v2.14.0 (F3) and dead under v2.15.0 (F1): removing
them in the commit that moves the pin means the pinned linter is clean at
every commit.

This record supersedes 0029-MADR's decision of the linter's version only,
not its Go-floor decision. 0029-MADR carries one dated line pointing here.

### Consequences

* Good, because the fleet lints with one version.
* Good, because the `nolint` directives, which would otherwise fail
  `nolintlint`, go; no new directive or exclusion is added.
* Bad, because a contributor with v2.14.0 installed gets `errcheck` findings
  at those three sites until they install v2.15.0; the install hint names it.
* Neutral, because no exported name, behaviour or dependency changes, and no
  release is cut.

### Confirmation

* `make lint` with a v2.15.0 binary: 0 issues for the host, for
  `GOOS=windows` and in `authcontract`.
* `git grep -n "v2\.14\.0" -- ':!docs/decisions' ':!docs/reports'` finds
  nothing.
* The Docs-installs-what-CI-installs test in `scripts/test_gates.py` fails
  when the two workflows name different linter versions (seen on a plant).
