---
status: in-progress
date: 2026-11-18
associated-madr: "0086-MADR-golangci-lint-v2-15-0.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement golangci-lint v2.15.0

Associated MADR: [0086-MADR-golangci-lint-v2-15-0.md](0086-MADR-golangci-lint-v2-15-0.md)

## Goal

The pin is `v2.15.0` everywhere the current state names it, and the three
`//nolint:errcheck` directives of the MADR's F1 are gone, in one commit. The
owner approved this PLAN in advance, for that scope, on 2026-11-18 (the
MADR's Context).

## Scope

* `Makefile`, `scripts/go-precheck.sh`, `.github/workflows/ci.yml`,
  `.github/workflows/docs.yml`: the version in the install line.
* `llmprovider/internal/transport/offline.go` (two lines) and
  `unsendable.go` (one line): the directive removed, nothing else.
* This pair, its row in `docs/README.md`, and one dated line in
  0029-MADR.
* Out of scope: any other record, the unreleased work of 0080 and 0082, any
  tag or release, `CHANGELOG.md` (a tooling change that no consumer sees; the
  changelog gate holds release sections only and the file records consumer
  effects).

## Implementation Steps

1. Run v2.15.0 on a `git archive HEAD` copy; record the three findings (MADR F1).
2. On a scratch copy, remove the directives and run v2.14.0 and v2.15.0 (F3).
3. Move the pin in the four files; remove the three directives.
4. Plant a mismatch between the two workflows' install lines in a scratch
   clone and run `scripts/test_gates.py`'s workflow test: it fails; then pass.
5. Run the gates below; stage the records only.

## Verification

* `make lint` with v2.15.0, host and `GOOS=windows`, and in `authcontract`: 0 issues.
* `make pre-add-check FILES="llmprovider/internal/transport/offline.go llmprovider/internal/transport/unsendable.go"`.
* `make api-check`, `make records-check`, `make docs-check`, `make tables-check`,
  `make changelog-check`, `make links-check`, `make disclosure-check`,
  `make markdownlint`, `go mod tidy -diff`, `actionlint` on both workflows,
  the full suite with `-race`.

## Rollout and Rollback

No release. Rollback is a revert of the change commit; the pin returns to
`v2.14.0` and the directives return with it.
