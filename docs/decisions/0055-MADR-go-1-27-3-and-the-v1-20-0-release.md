---
status: accepted
date: 2026-10-20
decision-makers: repository owner
consulted: "the Go vulnerability database's GO-2026-6809 (vuln.go.dev, read 2026-10-20); go.dev/dl's release list; CI runs 39521616762 and 39535986410; 0029-MADR-go-1-27-2-for-standard-library-fixes.md; 0048-MADR-adopt-go-sdk-best-practices.md (D5, D6, A1, A2); this repository at 5c60593"
informed: consumers of go-llmprovider-sdk v1, gobble-cli and prepare-commit-msg among them
---

<!-- markdownlint-disable MD013 -->

# Require Go 1.27.3 for GO-2026-6809, Release It as v1.20.1, and Give v1.20.0 Its GitHub Release by Hand

## Context and Problem Statement

`v1.20.0`, the release of
[0053-MADR](0053-MADR-stream-retry-and-cache-defects.md), was tagged on
`5c60593` and pushed on 2026-10-20. CI's run for the tag failed, and the
tag has no GitHub Release.

### Facts (observed 2026-10-20)

| # | Observation | Source |
| :--- | :--- | :--- |
| F1 | CI on `main` at `5c60593`, run 39521616762: all three `validate` jobs passed. The Linux job's govulncheck step passed at 00:32:03 UTC | `gh run view` |
| F2 | CI on the tag `v1.20.0`, run 39535986410, on the same commit: `validate (ubuntu-24.04)` failed at its govulncheck step (exit 3), at 03:01 UTC. Its later steps were skipped, and so was `release`, which `needs: validate` (C5). macOS and Windows passed | `gh run view --log-failed` |
| F3 | The finding: "Vulnerability #1: GO-2026-6809, Distrust compromised intermediate certificates in crypto/x509 … Found in: crypto/x509@go1.27.2, Fixed in: crypto/x509@go1.27.3", with one trace, `llmprovider/catalog/model_metadata.go:618:24: catalog.limitedReader.Read calls http.readWriteCloserBody.Read, which eventually calls x509.Certificate.Verify` (C8) | the run's log |
| F4 | The advisory was published at 2026-10-21 00:27:29 UTC, alias CVE-2026-111943. Its affected ranges are the standard library before 1.26.10, and from 1.27.0 before 1.27.3; the symbol is `crypto/x509`'s `Certificate.Verify`. It adds a denylist of compromised intermediate certificates to `Verify`, for platforms whose system trust stores are not updated | `vuln.go.dev/ID/GO-2026-6809.json` |
| F5 | On this host, `make vuln` (govulncheck `v1.8.0`, CI's pin, C3) under Go 1.27.2 exits 2, with govulncheck's exit 3 and the same finding and trace as F3 | `make vuln` |
| F6 | go.dev/dl lists `go1.27.3` and `go1.26.10` as the current releases | `go.dev/dl/?mode=json` |
| F7 | The module proxy already serves `v1.20.0`: `tools/consumer.py v1.20.0` downloaded it ("go: downloading github.com/maccavelli/go-llmprovider-sdk v1.20.0") and built a consumer of its 15 public packages | `tools/consumer.py` |
| F8 | `go.mod` says `go 1.27.2`, with no `toolchain` line (C1). Both workflows install the version it names (`go-version-file: go.mod`, C2, C21) | the files |
| F9 | The owner's Go environment sets `GOTOOLCHAIN=go1.27.2`, so the `go` command on this host runs 1.27.2 and refuses a module that requires more. *(By 2026-10-21 it names `go1.27.3`: `go version go1.27.3 darwin/arm64`.)* | `go env GOTOOLCHAIN` |

F1 passed and F2 failed on one commit, three hours apart. The first scan
ran five minutes after the advisory's publication, probably before it
reached the database govulncheck reads **[inferred]**. Nothing in the
module changed. The world changed, the case
[0052-MADR](0052-MADR-skip-full-ci-for-docs-only-pushes.md)'s Consequences
name, and CI's tag run caught it.

The trace runs through every HTTPS request the module makes: the
`catalog` package's metadata fetch reads a TLS connection, whose handshake
verifies the server's certificate chain. A program built with Go 1.27.2's
standard library lacks the denylist **[read: F4]**.

### The rules that apply

* **R54** of the standards guide: "The Go floor is `go 1.27.2`, … CI
  builds with the version `go.mod` names … The floor is raised with each Go
  security patch." (C9, C10; cited decision 0048-MADR D6 and A1).
* **R53:** "Every release has a changelog section and a GitHub Release …
  CI's `release` job makes the tag's Release with that section as its
  notes." (C11; 0048-MADR D5).
* 0029-MADR raised the floor from 1.27.1 to 1.27.2 for the same kind of
  finding, and rejected pinning CI alone (C24).
* 0048-MADR A2 made six past tags' Releases by hand, each from its
  section, with `--verify-tag --latest=false`, and read each body back
  (C23).

### The two questions

1. How the module moves to a standard library with the fix.
2. What `v1.20.0`, a published version with no Release, gets. Its tag
   cannot move: the proxy has served it (F7), so a re-tag would conflict
   with the checksum database, and the tag's CI cannot pass, since it
   installs Go 1.27.2 (F8).

## Decision Drivers

* CI is red on any new commit until its Go has the fix: every later run's
  govulncheck step fails as F2's did.
* The finding is in code every HTTPS request runs (F3, F4).
* R54 asks for the raise; 0029-MADR set the pattern, and rejected the
  alternatives below.
* R53 asks for a Release for every tag; the release job is the normal way,
  and only after `validate` passes.
* A published version cannot be changed (F7).
* A raised floor is a requirement imposed on every consumer.

## Considered Options

The floor:

* **1(a)** Raise `go.mod` to `go 1.27.3`, and release it as `v1.20.1`.
* **1(b)** Keep `go 1.27.2`, and pin CI's `setup-go` to 1.27.3.
* **1(c)** Keep everything, and exclude GO-2026-6809 from govulncheck.

`v1.20.0`'s Release:

* **2(i)** Make it by hand, from its section, once `v1.20.1` has passed CI.
* **2(ii)** Give it no Release, and record the exception to R53.
* **2(iii)** Re-run the tag's CI.

## Decision Outcome

Chosen options: **"1(a)"** and **"2(i)"**, the owner's choice on
2026-10-20 ("Follow recommendations").

* 1(a), because it is the one option where `go.mod`, CI and every
  consumer's minimum agree on a standard library with the fix, as 0029-MADR
  found for 1.27.2, and because R54 says so.
* 2(i), because `v1.20.0`'s code passed every `validate` job on `main`
  (F1), its one failure is the advisory `v1.20.1` fixes, and R53 then
  holds for every tag from `v1.10.2` on. 0048-MADR A2 is the procedure.

### D1. The floor is `go 1.27.3`

* `go mod edit -go=1.27.3`: one line, no `toolchain` line, `go.sum`
  unchanged, `go mod tidy -diff` clean.
* R54 reads "`go 1.27.3`" and names GO-2026-6809 as the patch's reason,
  keeping its rule that the floor rises with each Go security patch. Its
  row in the guide's index cites this record beside 0048-MADR A1. That is
  the amendment of R54's cited decision that AGENTS.md's "API work" asks
  for: 0048-MADR gains A3, a pointer here, and 0029-MADR an amendment
  saying its floor is raised again.
* The current-state statements say 1.27.3: `AGENTS.md` (C13), the README
  (C14, C15), `docs/architecture.md` (C16) and the migration guide (C17).
  Records, which say what ran then, are left. So is
  `scripts/test_gates.py`'s scratch module (C18), whose `go 1.27.1` states
  nothing about this module.
* **The linter.** 0029-MADR's amendment says a toolchain move checks the
  linter pin. golangci-lint `v2.14.0` (C4, C19, C20) is checked under Go
  1.27.3 before the change is staged. A patch release is not expected to
  change the export data format **[assumption, to measure]**. If it fails,
  the PLAN stops.

### D2. `v1.20.1` carries it

* `CHANGELOG.md`'s `v1.20.1` section has a **Security** entry for D1,
  which names GO-2026-6809 and what a consumer must do. It also has the
  **Fixed** entry already in `[Unreleased]`: the Docs workflow's
  golangci-lint (0052-MADR's amendment).
* `tools/apidiff.py v1.20.0` reports no change. A raised floor is not an
  exported-API change (R48), so a patch version is enough. A consumer's
  build can still break (Consequences), which the Security entry says.
* The release is the normal one: the tag's CI runs every `validate` job,
  and its `release` job makes the Release.

### D3. `v1.20.0`'s Release, by hand

After `v1.20.1`'s Release exists:

* the notes are `scripts/check_changelog.py --section v1.20.0 --repo
  maccavelli/go-llmprovider-sdk`, run on a clone at the tag with every tag
  present, as the release job runs it (C6). The identifier scan runs on
  them, and they are compared with the 2,239 bytes 0053-PLAN's Phase 3
  checked;
* `gh release create v1.20.0 --title v1.20.0 --notes-file … --verify-tag
  --latest=false`, the release job's command (C7) with A2's
  `--latest=false`, so `v1.20.1` stays Latest;
* the body is read back with `gh release view v1.20.0 --json body` and
  equals the file.

The notes are exactly the section, with no line added about the floor. The
`v1.20.1` section says that `v1.20.0` needs Go 1.27.3 to be safe.

### Consequences

* Good, because CI's govulncheck step passes with no exclusion, and the
  Release job gets its `validate` again.
* Good, because a consumer that runs `go get` on `v1.20.1` gets `go
  1.27.3`. With `GOTOOLCHAIN=auto`, the default, its `go` command then
  selects a toolchain with the fix.
* Good, because every tag from `v1.10.2` on has a Release, with its section
  as the notes.
* Bad, because a consumer at `go 1.27.2` must raise its own directive to
  take `v1.20.1`, and one built with `GOTOOLCHAIN=local` on 1.27.2 cannot
  build it.
* Bad, because `v1.20.0`'s Release did not follow a passing `validate` on
  its own tag. The record says why: it passed on `main` (F1), and failed
  only on a finding outside its code.
* Bad, because this host's `go` refuses the module until the owner's Go
  environment names 1.27.3 (F9). Until then, each check the PLAN runs sets
  `GOTOOLCHAIN=go1.27.3` itself. The environment file is the owner's to
  change. *(The owner had changed it by 2026-10-21.)*
* Neutral, because the advisory's other range, Go 1.26 before 1.26.10,
  does not apply: the module needs Go 1.27 (0048-MADR A1).
* Neutral, because `v1.20.0` stays as it is, and the proxy keeps serving
  it. Its `go 1.27.2` floor does not stop a consumer from building it with
  1.27.3.

### Confirmation

* `go.mod` says `go 1.27.3` and has no `toolchain` line. `go mod tidy
  -diff` exits 0 under `GOTOOLCHAIN=go1.27.3`.
* `GOTOOLCHAIN=go1.27.3 make vuln` exits 0, "No vulnerabilities found.";
  under Go 1.27.2 it exits 2, as F5 found.
* `GOTOOLCHAIN=go1.27.2 go build ./...` fails, naming the 1.27.3
  requirement.
* golangci-lint `v2.14.0` lints the module under 1.27.3 with no findings.
* `git grep -n "1\.27\.2" -- ':!docs/decisions' ':!docs/reports'` finds no
  current-state statement.
* CI on `main` passes all three `validate` jobs. The tag `v1.20.1`'s run
  passes them, and its `release` job makes a Release whose body equals its
  section.
* `go list -m -json github.com/maccavelli/go-llmprovider-sdk@v1.20.1`
  through the proxy gives `"GoVersion": "1.27.3"`.
* `v1.20.0`'s Release body equals its section, and `gh release list` shows
  `v1.20.1` as Latest.

## Pros and Cons of the Options

### 1(a) Raise `go.mod` to `go 1.27.3`

* Good, because one line sets CI's Go, the module's minimum and its
  consumers'.
* Good, because it is R54's rule, applied as 0029-MADR applied it.
* Bad, because every consumer must be at 1.27.3.

### 1(b) Keep `go 1.27.2`, pin CI to 1.27.3

* Good, because nothing changes for consumers.
* Bad, because `go.mod` and CI disagree, the drift 0029-MADR removed, and
  R54 says CI builds with the version `go.mod` names.
* Bad, because consumers get no signal that 1.27.2 lacks the fix.

### 1(c) Exclude the finding

* Bad, because it hides a called vulnerability to turn CI green. Not a
  fix.

### 2(i) `v1.20.0`'s Release by hand

* Good, because R53 holds, and the procedure is 0048-MADR A2's, done six
  times.
* Good, because the code it describes passed every `validate` job (F1).
* Bad, because this Release skips the `release` job's gate, which is the
  gate R53's text relies on.

### 2(ii) No Release for `v1.20.0`

* Good, because no Release is made outside the job.
* Bad, because R53 then has an exception from `v1.10.2` on, and a reader of
  the Releases page finds `v1.19.0` followed by `v1.20.1`, with
  `v1.20.0`'s notes only in `CHANGELOG.md`.

### 2(iii) Re-run the tag's CI

* Bad, because it cannot pass: the run installs the Go `go.mod` names at
  the tag, 1.27.2 (F8), and govulncheck fails on it (F5).

## More Information

* The PLAN:
  [0055-PLAN-go-1-27-3-and-the-v1-20-0-release.md](0055-PLAN-go-1-27-3-and-the-v1-20-0-release.md).
* Found at
  [0053-PLAN-stream-retry-and-cache-defects.md](0053-PLAN-stream-retry-and-cache-defects.md)'s
  Phase 3 step 3, whose deviation entry points here.
* [0029-MADR-go-1-27-2-for-standard-library-fixes.md](0029-MADR-go-1-27-2-for-standard-library-fixes.md):
  the last raise, its options and its linter amendment.
* [0048-MADR-adopt-go-sdk-best-practices.md](0048-MADR-adopt-go-sdk-best-practices.md):
  R53 (D5), R54 (D6, A1) and the hand-made Releases (A2).
* Measured on this host with govulncheck `v1.8.0` through `go run`, as
  `make vuln` runs it. The outputs are kept in the session's scratch
  directory, not committed.

### Code claims

<!-- claims: self=5c60593 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The Go floor | `go.mod:3` | `go 1.27.2` |
| C2 | CI installs the version `go.mod` names | `.github/workflows/ci.yml:42` | `go-version-file: go.mod` |
| C3 | CI's govulncheck pin | `.github/workflows/ci.yml:52` | `run: go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` |
| C4 | CI's golangci-lint pin | `.github/workflows/ci.yml:59` | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0` |
| C5 | The release job needs every `validate` job | `.github/workflows/ci.yml:72` | `needs: validate` |
| C6 | The release job's notes | `.github/workflows/ci.yml:83` | `run: python3 -B scripts/check_changelog.py --section "$GITHUB_REF_NAME" --repo "$GITHUB_REPOSITORY" > notes.md` |
| C7 | The release job's command | `.github/workflows/ci.yml:87` | `gh release create "$GITHUB_REF_NAME" --title "$GITHUB_REF_NAME" --notes-file notes.md --verify-tag` |
| C8 | The traced read | `llmprovider/catalog/model_metadata.go:618` | `if n, err := l.r.Read(one[:]); n > 0 {` |
| C9 | R54's floor | `docs/guides/api-standards.md:331` | `` - **R54. The Go floor is `go 1.27.2`,** with no `toolchain` line, and CI `` |
| C10 | R54's rule for a security patch | `docs/guides/api-standards.md:335` | `2026-10-08 security fixes. The floor is raised with each Go security` |
| C11 | R53 | `docs/guides/api-standards.md:326` | `- **R53. Every release has a changelog section and a GitHub Release.**` |
| C12 | R54's row in the index | `docs/guides/api-standards.md:356` | `` `go.mod`'s `go` line, which CI's `go-version-file` reads `` |
| C13 | AGENTS.md's requirement | `AGENTS.md:11` | `` 1.27.2: Go 1.27 for `encoding/json/v2`, and the `.2` patch for the `` |
| C14 | The README's requirement | `README.md:20` | `It requires Go 1.27.2.` |
| C15 | The README's Status | `README.md:308` | `` The current release is `v1.20.0`. The module requires Go 1.27.2. `` |
| C16 | The architecture's requirement | `docs/architecture.md:11` | `` 1.27.2, `golang.org/x/term` and, indirectly, `golang.org/x/sys`. `` |
| C17 | The migration guide's requirement | `docs/guides/migrating-from-mcplib.md:38` | `- **Go (fifth amendment; 0029-MADR).** The module requires Go 1.27.2.` |
| C18 | A gate test's scratch module, left | `scripts/test_gates.py:137` | `go 1.27.1` |
| C19 | The Makefile's install hint | `Makefile:39` | `golangci-lint@v2.14.0` |
| C20 | The precheck's install hint | `scripts/go-precheck.sh:109` | `golangci-lint@v2.14.0` |
| C21 | The Docs workflow installs the same Go | `.github/workflows/docs.yml:36` | `go-version-file: go.mod` |
| C22 | 0048-MADR's floor amendment | `docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md:411` | `### A1 (2026-10-19): the floor stays` |
| C23 | 0048's hand-made Releases | `docs/decisions/0048-PLAN-adopt-go-sdk-best-practices.md:1023` | `--verify-tag --latest=false` |
| C24 | 0029-MADR's choice | `docs/decisions/0029-MADR-go-1-27-2-for-standard-library-fixes.md:80` | `Chosen option: "Raise the` |
