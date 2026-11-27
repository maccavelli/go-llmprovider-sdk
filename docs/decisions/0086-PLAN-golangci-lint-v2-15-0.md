---
status: in-progress
date: 2026-11-27
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

## Execution Record

### 2026-11-27: closed

Written at v1.34.0's preparation ([0091-PLAN-adopt-the-responses-api-per-provider.md](0091-PLAN-adopt-the-responses-api-per-provider.md), Phase 4, step 4b.2); this PLAN had no execution record. Its one commit is `9a1a221` (2026-11-18), "chore(lint): pin golangci-lint v2.15.0 and drop dead nolint directives". `git tag --contains 9a1a221` lists `v1.32.0` first, so the work shipped in v1.32.0. At `223bb8d` the pin is `v2.15.0` in `Makefile:42`, `scripts/go-precheck.sh:154`, `.github/workflows/ci.yml:68` and `.github/workflows/docs.yml:43`, and `offline.go` and `unsendable.go` hold no `nolint:errcheck`. The gate outputs of 2026-11-18 are not kept in this record; the step 4a gates of 2026-11-27 ran `make lint` and the pre-add check with that pin, clean. The gates over v1.34.0's tree are in [0091-PLAN-adopt-the-responses-api-per-provider.md](0091-PLAN-adopt-the-responses-api-per-provider.md)'s execution record, Phase 4 (2026-11-27), run on the tree that carries this entry. Status `complete`. *(2026-11-27: withdrawn, by the deviation of 2026-11-27 in 0091-PLAN Phase 4 (the code review's item 2). The commit cited above is wrong: `9a1a221` is the records commit; the work is `a7e924f` (2026-11-18, "chore(lint): bump golangci-lint to v2.15.0 and drop nolint directives", the six files of the Scope), first contained by `v1.32.0`. Open, with no evidence recorded: step 1 (v2.15.0 on a `git archive` copy, F1's three findings), step 2 (v2.14.0 and v2.15.0 with the directives removed, F3), step 4 (the workflow mismatch plant), and the Verification's `actionlint` on both workflows, `go mod tidy -diff` and the full suite with `-race`. Not run now. This PLAN is `in-progress` and is not one of v1.34.0's closures. The text before this note is unchanged.)*
