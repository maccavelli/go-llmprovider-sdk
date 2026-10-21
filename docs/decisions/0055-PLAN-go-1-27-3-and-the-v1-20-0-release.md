---
status: complete
date: 2026-10-21
associated-madr: "0055-MADR-go-1-27-3-and-the-v1-20-0-release.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Go 1.27.3 as the Floor, Release v1.20.1, and Make v1.20.0's Release

Associated MADR: [0055-MADR-go-1-27-3-and-the-v1-20-0-release.md](0055-MADR-go-1-27-3-and-the-v1-20-0-release.md)

## Goal

The MADR's D1 to D3, as the owner chose them on 2026-10-20 (1(a), 2(i)):

* `go.mod` requires Go 1.27.3, CI builds with it, and govulncheck passes
  with no exclusion;
* `v1.20.1` is released by CI's `release` job, with the floor and the Docs
  workflow's fix in its section;
* `v1.20.0` has a GitHub Release whose body is its section, made by hand
  after `v1.20.1`'s, with `v1.20.1` left Latest.

## Scope

### In scope

* **Phase 1, the floor:**
  * `go.mod`;
  * the current-state statements: `AGENTS.md:11` to `12`, `README.md:20`,
    `README.md:308` (its Go sentence), `docs/architecture.md:10` to `11`,
    `docs/guides/migrating-from-mcplib.md:38`;
  * `docs/guides/api-standards.md`: R54 (lines 331 to 336) and its row in
    the index (line 356);
  * an amendment in `0048-MADR-adopt-go-sdk-best-practices.md` (A3) and in
    `0029-MADR-go-1-27-2-for-standard-library-fixes.md`, each a pointer
    here;
  * `CHANGELOG.md`'s `[Unreleased]`: a Security entry.
* **Phase 2, `v1.20.1`:** `CHANGELOG.md`, and the README's Status.
* **Phase 3, `v1.20.0`'s Release:** a GitHub Release, and nothing in the
  tree but this PLAN's record.
* **Records:** this pair, its index rows, and the deviation entry in
  `0053-PLAN-stream-retry-and-cache-defects.md`.

### Out of scope

* **Any exported name, golden or test:** `tools/apidiff.py v1.20.0`
  reports nothing.
* **The owner's Go environment** (`GOTOOLCHAIN=go1.27.2`): it is the
  owner's. Until the owner changes it, each `go` command this PLAN runs sets
  `GOTOOLCHAIN=go1.27.3` in its own environment. Phase 1 step 7 asks the
  owner. *(2026-10-21: the owner has changed it; `go env GOTOOLCHAIN` is
  `go1.27.3`.)*
* **Records that name 1.27.2,** which say what ran then.
* **`scripts/test_gates.py:137`'s scratch `go 1.27.1`,** which states
  nothing about this module.
* **The `v1.20.0` tag,** which does not move (MADR F7).
* **gobble-cli and prepare-commit-msg moving to `v1.20.1`,** which their
  own repositories do. This PLAN names the release to them only if the
  owner asks.
* **0052's staged Docs fix,** which is 0052's. It is committed first, on
  its own, and its Fixed entry is carried into `v1.20.1`'s section.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **The toolchain.** Every `go`, `make` and `tools/gate.py` command runs
  with `GOTOOLCHAIN=go1.27.3` in its environment, until the owner's
  environment names it. The first such command downloads the toolchain to
  the module cache.
* **The gate, before staging:**
  * `GOTOOLCHAIN=go1.27.3 python3 -B tools/gate.py`;
  * `tools/claims.py` on both records;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter.
* **Stage, then the owner commits** with `git commit --no-edit`. The
  disclosure guard runs over the outgoing commits before each push. Push and
  tags happen on the owner's word.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * golangci-lint `v2.14.0` fails under Go 1.27.3;
  * govulncheck reports anything under Go 1.27.3;
  * an exported name, a golden or a test would change;
  * a CI job fails.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. This pair, and its two rows in `docs/README.md` (`tools/records.py
   add`).
2. `0053-PLAN-stream-retry-and-cache-defects.md`: a dated deviation entry
   for its Phase 3 step 3 (the tag's CI failed, the Release was not made),
   with steps 4 and 5's results, pointing here.

Before this phase is staged, 0052's Docs fix, staged now, is committed by
the owner on its own, so that this commit holds records only. *(2026-10-21:
committed as `e68be51`; 0019's third `go fix` run followed as `3c7ad7e`.)*

### Phase 1: the floor (MADR D1)

1. **Red first, already seen.** CI's run 39535986410 failed its
   govulncheck step on Go 1.27.2, and `make vuln` on this host exits 2 with
   the same finding (MADR F2, F5). Under `GOTOOLCHAIN=go1.27.3`, on the
   tree before step 2, `make vuln` is run once more and recorded. The
   expected result is a pass, since govulncheck reads the toolchain's
   standard library version, not the directive **[assumption, measured by
   this step]**.
2. **The directive.** `GOTOOLCHAIN=go1.27.3 go mod edit -go=1.27.3`.
   Assert that `git diff go.mod` is the one line `go 1.27.2` to
   `go 1.27.3`, that there is no `toolchain` line, and that `go.sum` is
   unchanged.
3. **The statements**, each asserted to land, nothing else on their lines:
   * `AGENTS.md`: "Requires Go 1.27.3: Go 1.27 for `encoding/json/v2`, and
     the `.3` patch for the standard library's security fixes, the last
     GO-2026-6809 (the standards guide's R54)."
   * `README.md:20`: "It requires Go 1.27.3."; `README.md:308`'s second
     sentence: "The module requires Go 1.27.3."
   * `docs/architecture.md`: "It requires Go 1.27.3, …".
   * `docs/guides/migrating-from-mcplib.md:38`: "… The module requires Go
     1.27.3 (0055-MADR)."
4. **R54.** "**R54. The Go floor is `go 1.27.3`,** …. The patch, `.3`,
   carries the standard library's security fixes through GO-2026-6809
   (`crypto/x509`). The floor is raised with each Go security patch. (0048
   D6, A1; 0055)". The index row's last cell becomes "0048-MADR A1,
   0055-MADR".
5. **The amendments.**
   * 0048-MADR, after A2: "### A3 (2026-10-20): the floor is `go
     1.27.3`", one paragraph: R54's version is raised by its own rule, for
     GO-2026-6809, by 0055-MADR; A1's reasons for Go 1.27 stand.
   * 0029-MADR, at its end: "## Amendment 2026-10-20: the floor is raised
     again, to `go 1.27.3`", one paragraph pointing to 0055-MADR. Neither
     record's earlier text is rewritten.
6. **`CHANGELOG.md`,** `[Unreleased]`, `### Security`, above 0052's
   `### Fixed`: the module requires Go 1.27.3, for GO-2026-6809 in
   `crypto/x509` (`Certificate.Verify`), which every HTTPS request the
   module makes reaches; a consumer at `go 1.27.2` raises its directive,
   and `GOTOOLCHAIN=auto` fetches the toolchain; `v1.20.0` and earlier need
   a Go 1.27.3 toolchain to be built safely. The MADR is linked.
7. **The checks,** all under `GOTOOLCHAIN=go1.27.3`:
   * `go mod tidy -diff`, 0;
   * `make vuln`, 0, "No vulnerabilities found.";
   * golangci-lint `v2.14.0`, `make lint` for the host and for
     `GOOS=windows`, 0 issues. If it fails, stop (0029-MADR's amendment);
   * `GOTOOLCHAIN=go1.27.2 go build ./...`, which must fail naming the
     1.27.3 requirement;
   * `git grep -n "1\.27\.2" -- ':!docs/decisions' ':!docs/reports'`, which
     must find no current-state statement;
   * the gate, and the record checks of "Rules for every phase".

   Then the owner is asked to run `go env -w GOTOOLCHAIN=go1.27.3`, since
   this host's `go` refuses the module at 1.27.2 once the commit lands. The
   PLAN does not run it. *(2026-10-21: already done by the owner.)*
8. **Push.** With the owner's word, after the disclosure guard: 0052's fix,
   Phase 0 and this phase go out together, with 0019's `go fix` commit
   (`3c7ad7e`), whose CI also needs the floor.
   * "CI" on `main` passes all three `validate` jobs, govulncheck included.
   * "Docs" passes too, since the push changes `docs.yml`. That is 0052's
     Phase 4 step 6, which 0052's PLAN records.

### Phase 2: release `v1.20.1` (MADR D2)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.20.1] - <date>`, dated
   by the commit the tag will name, with Security (Phase 1) and Fixed
   (0052's Docs workflow). An empty `[Unreleased]` goes above it, and the
   comparison links gain `v1.20.1`. The README's Status names `v1.20.1`.
2. **The notes,** in a local clone of the tree with a local `v1.20.1` tag:
   `--section v1.20.1 --repo maccavelli/go-llmprovider-sdk` exits 0, and
   the output is scanned for identifiers.
3. **Before the push:** the gate; `tools/apidiff.py v1.20.0`, no change;
   the disclosure guard.
4. **The tag,** on the owner's word: `v1.20.1`, annotated as `v1.20.0` is
   (message `v1.20.1`), on the release commit, then pushed. The tag's run
   passes every `validate` job, and its `release` job makes the Release.
   `gh release view v1.20.1 --json body` equals step 2's output.
5. **After the tag:**
   * `tools/consumer.py v1.20.1` and `tools/offline.py v1.20.1`;
   * `go list -m -json github.com/maccavelli/go-llmprovider-sdk@v1.20.1`
     through the proxy: `"GoVersion": "1.27.3"`.

### Phase 3: `v1.20.0`'s Release (MADR D3)

1. **The notes.** In a clone at the tag `v1.20.0`, with every tag:
   `python3 -B scripts/check_changelog.py --section v1.20.0 --repo
   maccavelli/go-llmprovider-sdk > v1.20.0.md`, exit 0. The file is
   compared with 0053-PLAN Phase 3's output (2,239 bytes), and
   `tools/denyscan.py` reads it, 0 hits. It is shown to the owner.
2. **The Release,** on the owner's word: `gh release create v1.20.0 --title
   v1.20.0 --notes-file v1.20.0.md --verify-tag --latest=false`.
3. **Read back:** `gh release view v1.20.0 --json body` equals the file;
   `gh release list` shows `v1.20.1` as Latest; every tag from `v1.10.2`
   on has a Release.
4. **Close:**
   * this PLAN `complete` (`tools/records.py status`);
   * `0053-PLAN-stream-retry-and-cache-defects.md` `complete`, its
     deviation entry completed with the Release's link.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | `go.mod` says `go 1.27.3`, no `toolchain` line, tidy | 1 | `git diff go.mod`; `go mod tidy -diff` |
| V2 | govulncheck passes on 1.27.3, and fails on 1.27.2 | 1 | `make vuln`, under each |
| V3 | The linter works on 1.27.3 | 1 | `make lint`, host and Windows |
| V4 | A 1.27.2 toolchain refuses the module | 1 | `GOTOOLCHAIN=go1.27.2 go build ./...` |
| V5 | No current statement names 1.27.2 | 1 | the `git grep` |
| V6 | CI and Docs pass on `main` | 1 | the run ids |
| V7 | `v1.20.1`'s Release is its section, and its `GoVersion` is 1.27.3 | 2 | `gh release view`; `go list -m -json` |
| V8 | No exported name changes | 2 | `tools/apidiff.py v1.20.0` |
| V9 | `v1.20.0`'s Release is its section, and `v1.20.1` is Latest | 3 | `gh release view`; `gh release list` |

## Rollout and Rollback

* **Rollout.** Four commits on `main`, staged by the agent and committed by
  the owner: 0052's fix, Phase 0, Phase 1, then Phase 2's release commit.
  Then the owner's tag, then Phase 3's Release.
* **Rollback.**
  * Phase 1 reverts on its own, and CI is red on govulncheck again until
    another fix lands. After `v1.20.1` is tagged, a revert ships as a new
    release.
  * Phase 3's Release can be deleted with `gh release delete v1.20.0`,
    which leaves the tag. The tag itself is never deleted.

## Execution record

### Phase 0: records (2026-10-21)

* 0052's fix was committed first, as `e68be51`, and 0019's third `go fix`
  run as `3c7ad7e`.
* The records were staged on their own: this pair, 0053-PLAN's deviation
  entry, and `docs/README.md` with this pair's rows only (`records-check:
  116 records, 0 problem(s)` on the staged snapshot). The owner committed
  them as `5eb068a`, with
  `0056-MADR-wizard-credential-only-entry-point.md` and its PLAN, still
  `proposed`, added. Records only, so the commit is still Phase 0's kind.
* Before staging, three statements that the owner's change of Go
  environment had made out of date were annotated, not rewritten: MADR F9
  and its Consequence, and this PLAN's toolchain rule and Phase 1 step 7.

### Phase 1: the floor (2026-10-21)

* **Approval.** "committed, proceed", 2026-10-21. This PLAN is
  `in-progress`. No deviation.
* **The toolchain.** The owner's environment names `go1.27.3`, so no
  command needed `GOTOOLCHAIN` set, except to name 1.27.2.

**Step 1,** on the tree before step 2 (`go 1.27.2`):

```text
$ GOTOOLCHAIN=go1.27.3 make vuln
No vulnerabilities found.                                             (exit 0)
$ GOTOOLCHAIN=go1.27.2 make vuln
Vulnerability #1: GO-2026-6809 … Found in: crypto/x509@go1.27.2      (exit 2; govulncheck's 3)
```

The step's assumption holds: govulncheck reads the toolchain's standard
library, not the directive.

**Step 2.** `go mod edit -go=1.27.3`. `git diff go.mod` is the one line
`go 1.27.2` to `go 1.27.3`. There is no `toolchain` line, `go.sum` is
unchanged, and `go mod tidy -diff` exits 0.

**Steps 3 to 6,** by `tools/edit.py`, each asserted:

* `AGENTS.md`: "Requires Go 1.27.3: Go 1.27 for `encoding/json/v2`, and the
  `.3` patch for the standard library's security fixes, the last
  GO-2026-6809 (the standards guide's R54)."
* `README.md`: both statements say 1.27.3.
* `docs/architecture.md`: "It requires Go 1.27.3, …".
* `docs/guides/migrating-from-mcplib.md`: "… The module requires Go 1.27.3
  (0055-MADR)."
* R54: "`go 1.27.3`", "The patch, `.3`, carries the standard library's
  security fixes through GO-2026-6809 (`crypto/x509`)", and "(0048 D6, A1;
  0055)". Its index row: "0048-MADR A1, 0055-MADR".
* 0048-MADR gains A3, and 0029-MADR "Amendment 2026-10-20: the floor is
  raised again, to `go 1.27.3`". Neither record's earlier text changes.
* `CHANGELOG.md` `[Unreleased]`: a Security entry above 0052's Fixed.

**Step 7, the checks,** each exit status captured before any filter:

| Check | Result |
| :--- | :--- |
| `go mod tidy -diff` | 0 |
| `make vuln` | 0, "No vulnerabilities found." |
| `make lint`, golangci-lint `v2.14.0` (built with go1.27.2), host and `GOOS=windows` | 0, "0 issues." each |
| `GOTOOLCHAIN=go1.27.2 go build ./...` | 1, "go: go.mod requires go >= 1.27.3 (running go 1.27.2; GOTOOLCHAIN=go1.27.2)" |
| `git grep -n "1\.27\.2" -- ':!docs/decisions' ':!docs/reports'` | 4 hits, none a current statement: the Security entry's own two lines, and 0029's two index titles |
| `make records-check changelog-check tables-check docs-check parity-check` | 118 records, 28 release sections, 38 packages, 0 sdkaudit violations, 409 parity rows: 0 problems each |
| `tools/claims.py` on 0055-MADR and 0048-MADR | 24 of 24 each |
| `tools/recordlint.py --changed`, `tools/linkcheck.py`, `tools/denyscan.py --changed` | no rule rose; 0 problems; 0 hits in 11 files |
| `tools/gate.py` | 27 of 27 passed, every line `rc=0` |

No Go file changed, so `make pre-add-check` has no file to check; the gate
runs it over every tracked Go file.

Step 7's ask of the owner is moot: the environment already names
`go1.27.3`. Step 8, the push, waits for the owner's word.

**Step 8, the push.** The owner committed this phase as `a3fd64a` and
pushed it with `e68be51`, `3c7ad7e` and `5eb068a` (2026-10-21).

* "CI", run 39612453007, `success`: `validate` passed on ubuntu-24.04
  (7 min 22 s, govulncheck included), windows-2025 and macos-15, and
  `release` was skipped.
* "Docs", run 39612452935, `success`: 0052's fix, live.

V1 to V6 are met.

### Phase 2: release `v1.20.1` (2026-10-21)

* **Approval.** The owner's "committed, pushed, proceed". No deviation.
* **Step 1.**
  * `CHANGELOG.md`: `[Unreleased]` became `[v1.20.1] - 2026-10-21`, with
    Security (Phase 1) and Fixed (0052's Docs workflow). An empty
    `[Unreleased]` sits above it, and the comparison links gain `v1.20.1`.
    "changelog-check: 29 release section(s), 0 problem(s)". The date must
    be that of the commit the tag names (R53).
  * The README's Status names `v1.20.1`.
* **Step 2, the notes.** In a clone of `a3fd64a` with this `CHANGELOG.md`
  and README and a local `v1.20.1` tag, `--section v1.20.1 --repo
  maccavelli/go-llmprovider-sdk` exits 0. It prints the Security and Fixed
  entries, 1,118 bytes, both MADR links made absolute at `blob/v1.20.1`.
  `tools/denyscan.py` on them: 0 hits. They are compared with the Release
  at step 4.
* **Step 3, before the push:** `tools/apidiff.py v1.20.0`, no output, exit 0; the
  gate, 27 of 27 passed; the disclosure guard, once this commit is made.

### Deviation (2026-10-21): only the five newest tags keep a Release

* **Found** at Phase 3 step 2, before the Release was made, by the owner:
  "i do not care that every tag from v1.10.x has a release. i only want the
  past 3 tags to have releases"; then "Delete them" for the older ones; then
  "change it from 3 to 5". Recorded as a standing rule of five (the MADR's
  amendment of 2026-10-21).
* **Saved first.** Every Release's metadata and body, all 11, were written
  to the session's scratch directory, none with assets. A first count of
  the older Releases, "8", was wrong; there were nine below the three
  newest, and seven below the five, which is what is deleted.
* **Phase 3, as amended:**
  * Step 2 stands: `v1.20.0`'s Release, with `--latest=false`.
  * Step 3's "every tag from `v1.10.2` on has a Release" is struck. In its
    place: `gh release delete TAG --yes`, which leaves the tag, for
    `v1.10.2`, `v1.11.0`, `v1.12.0`, `v1.13.0`, `v1.14.0`, `v1.15.0` and
    `v1.16.0`. Then `gh release list` shows exactly `v1.17.0` to `v1.20.1`,
    `v1.20.1` Latest, and `git ls-remote --tags` still lists all 29 tags.
  * New: R53, `CHANGELOG.md`'s preamble, the README's Status,
    `docs/architecture.md`'s CI passage, and 0048-MADR's A4, staged as one
    Markdown-only commit.
* **Files added to the scope:** `docs/guides/api-standards.md`,
  `CHANGELOG.md`, `README.md`, `docs/architecture.md`,
  `docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md`.

**Phase 2, steps 4 and 5, after the tag** (2026-10-21):

* The owner tagged `v1.20.1` on `4e6130f`, annotated, and pushed it.
* CI's run for the tag, 39626499816, `success`: every `validate` job
  passed, and `release` published the Release at 15:36:40 UTC. It is not a
  draft or a pre-release, and it is Latest.
* `gh release view v1.20.1 --json body` equals step 2's notes, 1,118 bytes,
  apart from the trailing newline `gh --jq` adds.
* `tools/consumer.py v1.20.1`: "consumer: 15 packages linked", "every step
  passed", built for the six platforms.
* `tools/offline.py v1.20.1`: "requests=0", "offline: ok", and both
  built-in plants caught.
* Through the proxy, `go list -m -json …@v1.20.1`: `"Version": "v1.20.1"`,
  `"GoVersion": "1.27.3"`, `Hash 4e6130f…`.

V7 and V8 are met.

### Phase 3: `v1.20.0`'s Release (2026-10-21)

As amended by the deviation entry above.

* **Step 1.** In a clone at the tag `v1.20.0` (`5c60593`), with every tag,
  `--section v1.20.0 --repo maccavelli/go-llmprovider-sdk` exits 0: 2,239
  bytes, byte for byte the notes 0053-PLAN's Phase 3 checked.
  `tools/denyscan.py`: 0 hits.
* **Step 2.** `gh release create v1.20.0 --title v1.20.0 --notes-file …
  --verify-tag --latest=false`:
  [v1.20.0](https://github.com/maccavelli/go-llmprovider-sdk/releases/tag/v1.20.0).
  Its body, read back, equals the file, apart from the trailing newline.
  It is not a draft or a pre-release.
* **Step 3, as amended.** `gh release delete TAG --yes`, each exit 0, for
  `v1.10.2`, `v1.11.0`, `v1.12.0`, `v1.13.0`, `v1.14.0`, `v1.15.0` and
  `v1.16.0`.
  * `gh release list` shows exactly `v1.20.1` (Latest), `v1.20.0`,
    `v1.19.0`, `v1.18.0` and `v1.17.0`.
  * `git ls-remote --tags origin 'v*'` still lists 29 tags, the deleted
    Releases' among them.
* **Step 4, close.** V9 is met, as amended: `v1.20.0`'s Release is its
  section, `v1.20.1` is Latest, and only the five newest tags have a
  Release. Every criterion is met, and this PLAN is `complete`.
  0053-PLAN is closed with it.
