---
status: complete
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

### 2026-11-27: the open items run, and closed

The owner's instruction, relayed by the coordinator, quoted exactly: "close out 0086 and clear the sdk follow-ups". Base `69618a4` on `main` (v1.34.0 and the records commit after it); `git status --porcelain` empty at the start and after every run. Each item the note above names as open was run on the current tree's copies, never in the tree; each exit was taken before any filter. Scripts and outputs: `<scratch>/sdkfu/` (`part_a.py`, `race.py`, `plant_workflow.json`, `out/`).

Binaries: v2.15.0, `golangci-lint has version 2.15.0 built with go1.27.4`; v2.14.0, built for this run with `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0` into the scratch directory, `golangci-lint has version 2.14.0 built with go1.27.4`. Each lint is `golangci-lint run -c .golangci.yml --build-tags live_gateways ./...` for the host and with `GOOS=windows`, and `golangci-lint run -c ../.golangci.yml ./...` in `authcontract`, as `make lint` runs them.

| Item | What ran | Exit (host, windows, authcontract) | Evidence |
| :--- | :--- | :--- | :--- |
| Step 1 (F1), at its base | v2.15.0 on `git archive a7e924f^` (`9a1a221`; its two transport files equal `fd986c4`'s, `git diff --stat` empty) | 1, 1, 0 | `3 issues: * nolintlint: 3`: `offline.go:21:52`, `:24:54`, `unsendable.go:48:65`, each ``directive `//nolint:errcheck // only whether the chain holds one matters` is unused for linter "errcheck" (nolintlint)``; `authcontract` `0 issues.` |
| Step 1, on the current tree | v2.15.0 on `git archive HEAD` with `a7e924f`'s three removals undone (each anchor asserted once before, the directive once after) | 1, 1, 0 | the same three `nolintlint` findings |
| F2 | v2.14.0 on that same copy | 0, 0, 0 | `0 issues.` each |
| Step 2 (F3) | v2.14.0 on `git archive HEAD` (the directives removed, as shipped) | 1, 1, 0 | `3 issues: * errcheck: 3`: `offline.go:21:5`, `:24:5`, `unsendable.go:48:2`, each ``Error return value of `errors.AsType` is not checked (errcheck)`` |
| Step 2 (F3), and the Verification's `make lint` | v2.15.0 on `git archive HEAD` | 0, 0, 0 | `0 issues.` each |
| Step 4, the plant | `tools/plant.py` on its scratch clone: `docs.yml`'s install line moved to `@v2.14.0`, run `python3 -B -m unittest scripts.test_gates.WorkflowsTest.test_docs_installs_what_ci_installs_before_gate_selftest` | 0 (caught) | `FAIL: test_docs_installs_what_ci_installs_before_gate_selftest` ... `AssertionError: Lists differ: ['github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0'] != []`; `plant: all 1 plant(s) caught` |
| Step 4, then pass | the same test, in the tree | 0 | `Ran 1 test ... OK` |
| `actionlint` (v1.7.12) on both workflows | `actionlint .github/workflows/ci.yml .github/workflows/docs.yml` | 0 | no output |
| `go mod tidy -diff` | `go mod tidy -diff`; `go -C authcontract mod tidy -diff` | 0; 0 | no output |
| The full suite with `-race` | `run-on-host --need linux --race -- go test -race -shuffle=on ./...`, then `-- go -C authcontract test -race -shuffle=on ./...`, as CI's race step | 0; 0 | the Linux machine, by `pick-host` (WSL excluded for a race run; "remote status 0", no fallback to this Mac); 46 packages `ok` and 3 `ok`; no `FAIL`, no `DATA RACE` |
| The MADR's Confirmation grep | `git grep -n "v2\.14\.0" -- ':!docs/decisions' ':!docs/reports'` | 1 (no match) | the control, the same grep for `v2\.15\.0`, finds the pin in five files |

Step 1 as written reads "a `git archive HEAD` copy"; that HEAD was `fd986c4`, before step 3 removed the directives. At `69618a4` no directive is left, so step 1 was run at the lint commit's parent and, on the current tree, with the directives put back as a plant; both give F1's three findings exactly. Step 3 is `a7e924f`; step 5's gates are those the note above cites (step 4a of 2026-11-27, and v1.34.0's in 0091-PLAN Phase 4), not re-run here.

The note above lists as open steps 1, 2 and 4, and the Verification's `actionlint`, `go mod tidy -diff` and the full suite with `-race`. Each now has its evidence, and every result matches the MADR's facts F1 to F3 and its Confirmation. The closing condition is met. Status `complete`, set with `tools/records.py status`.
