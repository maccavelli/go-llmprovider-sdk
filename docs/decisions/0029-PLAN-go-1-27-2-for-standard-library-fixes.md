---
status: in-progress
date: 2026-10-08
associated-madr: "0029-MADR-go-1-27-2-for-standard-library-fixes.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 -->

# Implement Go 1.27.2 as the Module's Requirement

Associated MADR: [0029-MADR-go-1-27-2-for-standard-library-fixes.md](0029-MADR-go-1-27-2-for-standard-library-fixes.md)

## Goal

`go.mod` requires Go 1.27.2, CI runs 1.27.2, and govulncheck passes on CI
with no exclusion.

## Scope

### In scope

* `go.mod`: `go 1.27.1` → `go 1.27.2`.
* The current-state statements: `AGENTS.md:10-11`, `README.md:19`,
  `README.md:193`, `docs/architecture.md:10-11`, and
  `docs/guides/migrating-from-mcplib.md:38` (deviation D1).
* `docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`: an
  amendment pointing to 0029-MADR.
* `docs/README.md`: the two index rows.

### Out of scope

* 0028's uncommitted Phase 4 work (D10) in the tree: not staged with this
  change, not touched.
* Historical statements in records (measurements "with Go 1.27.1", the
  Windows run records): they describe what ran then.
* The owner's Go env file (`GOTOOLCHAIN=go1.27.2`): it is the owner's, and
  agrees with the change.
* A release or tag.

## Implementation Steps

1. **Red first, already seen.** CI's Linux job on `52709e4` failed its
   govulncheck step; `GOTOOLCHAIN=go1.27.1 go run
   golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` exits 3 with ten called
   findings (0029-MADR). No new check is added.
2. **The directive.** `go mod edit -go=1.27.2`. Assert `go.mod` reads
   `go 1.27.2`, has no `toolchain` line, and nothing else changed
   (`git diff go.mod` is one line); `go.sum` unchanged.
3. **The statements.** Each "Go 1.27.1" in the four places of the scope
   becomes "Go 1.27.2", nothing else on those lines.
4. **0002-MADR.** Amendment 2026-10-08: "The `go` directive is `go 1.27.2`,
   by 0029-MADR", one paragraph, the fifth amendment left as written.
5. **Index.** `docs/README.md` gains rows for 0029-MADR and 0029-PLAN;
   `make records-check` passes.
6. **Gate.** Under `GOTOOLCHAIN=go1.27.2`, on a scratch copy of `HEAD`
   with only this change applied, since the tree also holds 0028's
   uncommitted work: `go mod tidy -diff`; `make vuln`;
   `go build ./...` and `go vet ./...` for darwin, linux and windows;
   `go test ./...`; `make lint`; `make records-check`; markdownlint on the
   changed Markdown. And `GOTOOLCHAIN=go1.27.1 go build ./...`, which must
   fail naming the 1.27.2 requirement. The identifier scan of the changed
   files.
7. **Record and stage.** This PLAN's execution record; stage exactly the
   files of the scope. The owner commits (`git commit --no-edit`) and
   pushes; CI's Linux job must then pass govulncheck.

## Verification

* `git diff --cached --stat` lists only: `go.mod`, `AGENTS.md`,
  `README.md`, `docs/architecture.md`, `docs/README.md`,
  `docs/guides/migrating-from-mcplib.md` (D1), the 0002-MADR, and this pair.
* Step 6's commands, each exit 0, and the 1.27.1 build failing.
* After the owner's push: the CI run for that commit is green on all three
  jobs. *(Amended 2026-10-08, D2: green once 0028 Phase 4's
  lint follow-up is committed too.)*

## Rollout and Rollback

* **Q1, the release.** Decided 2026-10-08: "With 0028 in v1.4.0". The
  options were: with
  0028 in `v1.4.0`; or a `v1.3.3` holding only this change, cut from a
  branch at `v1.3.2`, since `main` already carries 0028's Phases 2 to 4
  (`276496b`, `ad92bf7`, `52709e4`), which are not patch-level. That is
  the owner's call; this PLAN tags nothing and branches nothing.
* **Approval:** "Proceed", 2026-10-08.
* **Rollback:** revert the commit. CI is red again on govulncheck until a
  different fix lands.

## Execution Record

### Deviation D1 (2026-10-08): the migration guide states the requirement

* **Found** at step 3, by the MADR's confirmation search
  (`git grep -n '1\.27\.1' -- ':!docs/decisions' ':!docs/reports'`):
  `docs/guides/migrating-from-mcplib.md:38`, "The module requires Go
  1.27.1.", a current-state statement the scope did not list. The other hit,
  `scripts/test_gates.py:133`, writes a scratch module's `go.mod` for a gate
  test and states nothing about this module; it is left.
* **Options put to the owner:** add the line to the scope; or leave it and
  narrow the confirmation to exclude guides.
* **Decision.** "Add it to scope": the line reads "The module requires Go
  1.27.2.", citing 0029-MADR beside the fifth amendment.

### Deviation D2 (2026-10-08): `make lint` fails on `HEAD`, from 0028 Phase 4

* **Found** at step 6. `make lint` exits 2 with three `goconst` findings,
  the same on scratch copies of `HEAD` (`52709e4`) under `go1.27.1` and
  `go1.27.2`, and with this change applied:
  `internal/redact/redact.go:183` (`token`, 3 occurrences),
  `internal/redact/redact.go:185` (`code`, 3) and
  `llmprovider/context_overflow.go:37` (`maximum context length`, 3). They
  are 0028 Phase 4's code, committed before that phase's lint gate ran. CI
  runs govulncheck before its lint step, so with this change alone CI fails
  at lint instead.
* **Options put to the owner:** fix them under 0028 Phase 4, in its
  follow-up; or in this change.
* **Decision.** "Fix under 0028 Phase 4". This change does not touch them.
  CI is green once both are committed; 0028-PLAN records the fix.

### Steps 2 to 6 (2026-10-08)

* **Step 2.** `go mod edit -go=1.27.2` (under `GOTOOLCHAIN=go1.27.2`):
  `git diff go.mod` is the one line `go 1.27.1` → `go 1.27.2`; no
  `toolchain` line; `go.sum` unchanged.
* **Step 3.** `AGENTS.md`, `README.md` (two), `docs/architecture.md`, and
  by D1 `docs/guides/migrating-from-mcplib.md`, say 1.27.2. The
  confirmation search then finds only `scripts/test_gates.py:133`.
* **Step 4.** 0002-MADR: "Amendment 2026-10-08: the `go` directive is
  `go 1.27.2`".
* **Step 5.** `docs/README.md`: two rows, 63 records;
  `records-check: 63 records, 0 problem(s)`.
* **Step 6,** on a scratch copy of `HEAD` with only this change, under
  `GOTOOLCHAIN=go1.27.2` (`go version go1.27.2 darwin/arm64`):

  | Check | Result |
  | :--- | :--- |
  | `go mod tidy -diff` | 0 |
  | `make vuln` | 0, "No vulnerabilities found." |
  | `CGO_ENABLED=0 go build ./...`, darwin, linux, windows | 0 each |
  | `CGO_ENABLED=0 go vet ./...`, with and without `-tags live_gateways`, darwin, linux, windows | 0 each |
  | `go test -count=1 ./...` | 0 |
  | `make lint` | 2: three `goconst` findings, present on `HEAD` (D2) |
  | `make records-check` | 0, 63 records |
  | `GOTOOLCHAIN=go1.27.1 go build ./...` | 1: "go: go.mod requires go >= 1.27.2 (running go 1.27.1; GOTOOLCHAIN=go1.27.1)" |

  A first run built and vetted for Linux with cgo on, which failed on the
  host's C headers; the gate builds with `CGO_ENABLED=0`, as 0028's did.

### Step 7 (2026-10-08)

* Staged for the owner's commit: exactly the files Verification lists. The
  tree's other changes, 0028 Phase 4's D10 work, are left unstaged.
* Pending: the owner's commit and push, and the CI run (D2).

### Deviation D3 (2026-10-08): CI's golangci-lint cannot read Go 1.27.2's export data

* **Found** in CI on `2e5e881`, after this change and 0028's follow-up were
  pushed: the Linux job passed govulncheck and failed "vet, gofmt, tidy,
  lint". `golangci-lint` `v2.13.1`, which CI installs
  (`.github/workflows/ci.yml:53`), reported four `typecheck` errors such as
  `could not import errors (… could not import internal/goarch (-: could
  not load export data: internal error in importing "internal/goarch"
  (cannot decode "internal/goarch", export data version 5 is greater than
  maximum supported version 4)))`. Reproduced on this host under
  `GOTOOLCHAIN=go1.27.2`: `v2.13.1`, installed to a scratch `GOBIN`, exits 1
  with the same errors; `v2.14.0` exits 0, `0 issues.`. The owner's local
  linter was already `v2.14.0`, so step 6's gate could not see it.
* **Options put to the owner:** pin `v2.14.0` under this PLAN; or a
  separate pair.
* **Decision.** "Pin v2.14.0 under 0029": the install in
  `.github/workflows/ci.yml:53`, and the install hints in `Makefile:39` and
  `scripts/go-precheck.sh:109`, name `v2.14.0`. Records that name `v2.13.1`
  describe what ran then, and are left. Added to the scope: those three
  files.
* **Checks** (2026-10-08, `GOTOOLCHAIN=go1.27.2`): `make lint` with
  `GOLANGCI_LINT` set to `v2.14.0` exits 0, `0 issues.` for the host and for
  `GOOS=windows`; with `v2.13.1` it exits 2, `8 issues: * typecheck: 8`.
  `make parity-check`, `make records-check` (63 records) and
  `make gate-selftest` exit 0. Staged for the owner; CI's run on the commit
  is the check that remains.
