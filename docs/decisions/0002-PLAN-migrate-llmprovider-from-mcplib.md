---
status: in-progress
date: 2026-09-29
associated-madr: "0002-MADR-migrate-llmprovider-from-mcplib.md"
decision-makers: go-llmprovider-sdk maintainers
---
# Implement the Migration of `llmprovider` and `wizard` from mcplib into go-llmprovider-sdk

Associated MADR: [0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md)

## Goal

At the end of this plan:

1. This repository is module `github.com/maccavelli/go-llmprovider-sdk`.
   * It holds `llmprovider`, `wizard` and `internal/redact` with their
     `mcplib` history.
   * It builds with no `mcplib` or MCP go-sdk dependency.
   * Its exported API equals `mcplib` `v1.6.0`, apart from the MADR's §4–§7
     changes.
     * *Superseded 2026-09-29 (second amendment):* functional parity,
       with the API of `0015-MADR-canonical-sdk-api-and-module-layout.md`, proven by its PLAN.
   * It is tagged `v1.0.0`.
2. The 26 LLM records of `mcplib` live in `docs/decisions/` and
   `docs/reports/`, renumbered per MADR §10. The mixed `decisions/0010` pair
   is copied, and every citation and link resolves.
3. The open work of `mcplib` records 0010 (ranking), `decisions/0009` and
   `decisions/0010` is transferred into this repository's 0009, 0008 and
   0010, executable but not executed.
4. `prepare-commit-msg`, `mcp-server-magictools` and `mcp-server-magicdev`
   import this module, each under its own approved record.
   *Amended 2026-09-29 (sixth amendment): `prepare-commit-msg` only, for now.*
5. `mcplib` `v1.7.0` no longer contains `llmprovider/`, `wizard/` or the
   moved records. A relocation table points readers here.
   *Deferred 2026-09-29 (sixth amendment).*

## Scope

### Repositories and the record each change is made under

| Repository | Record that authorises the change | Phases |
|---|---|---|
| go-llmprovider-sdk (this) | this PLAN | 0, 2–8, 14 |
| go-llmprovider-sdk (this) | [0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md) (added 2026-09-29) | between 7 and 8 |
| mcplib | `docs/decisions/0015-{MADR,PLAN}-transfer-llmprovider-to-go-llmprovider-sdk.md` (written in Phase 1) | 1, 9, 13 *(2026-09-29: 9 open, 13 deferred)* |
| prepare-commit-msg | `docs/decisions/0008-{MADR,PLAN}-adopt-go-llmprovider-sdk.md` | 10 |
| mcp-server-magictools | `docs/decisions/0005-{MADR,PLAN}-adopt-go-llmprovider-sdk.md` | 11 *(deferred 2026-09-29)* |
| mcp-server-magicdev | `docs/decisions/0001-{MADR,PLAN}-adopt-go-llmprovider-sdk.md` | 12 *(deferred 2026-09-29)* |

Phases 1 and 10–13 each begin by authoring that repository's companion pair
(bootstrap exception: docs only), presenting it, and waiting for approval.
This PLAN states what each companion must contain; the companion owns its
execution detail.

### In scope in this repository

* `go.mod`, `go.sum`
* `llmprovider/**`, `wizard/**`, `internal/redact/**`
* `docs/**`, `README.md`
* `AGENTS.md`, `Makefile`, `.golangci.yml`, `.markdownlint-cli2.jsonc`,
  `.gitignore`
* `.github/workflows/ci.yml`
* Added 2026-09-29 by the MADR's amendment:
  * `scripts/go-precheck.sh`;
  * `.claude/rules/madr-and-plan-skill.md`, `.claude/.gitignore`;
  * `.grok/rules/madr-plan-before-mutating-work.md`;
  * `.opencode/rules.md`, `opencode.json`.
* Added 2026-09-29 by the MADR's third amendment:
  ~~`.github/dependabot.yml`~~ *(struck 2026-09-29, no Dependabot)* and
  `docs/architecture.md` (Phase 2b).
* Added 2026-09-29 by the MADR's fifth amendment: `go.mod` and `go.sum`
  move from Phase 4 to Phase 2d (already in scope).
* Added 2026-09-29 by the MADR's fourth amendment: no new file; Phase 2c
  changes `scripts/go-precheck.sh`, `.golangci.yml`, `Makefile` and
  `AGENTS.md`, all already in scope.

### Out of scope

* Executing any transferred phase (MADR §11).
* Any new dependency or version bump.
* Any API change beyond MADR §4–§7.
* `mcplib` `decisions/0010` P1 and P8.
* Consumer feature adoption (`GenerateThinkingWithRetry`, `ProfileCapable`,
  OAuth support in the MCP servers).

### Fixed inputs

These are the values Phase 1 records:

| Name | Value |
|---|---|
| `F` (freeze commit) | `mcplib` HEAD when Phase 1 is approved. Expected: `4e1f9a5` = `v1.6.0`. If HEAD has moved, record the new hash and re-run Phase 1's check that no commit after `v1.6.0` touched `llmprovider/` or `wizard/`. |
| `BASE` (API parity baseline) | `mcplib` `v1.6.0` |
| `SCRATCH` | the session scratchpad directory; every experiment and throwaway script lives there, never in a tree |

## Implementation Steps

Every phase ends with the pre-add checks and a `git commit --no-edit` in the
repository it changed. The global hook writes the message. Before the first
commit in each repository, confirm
`git rev-parse --path-format=absolute --git-path hooks` resolves to
`~/.global-git-hooks`; this is already true for this repository and
`mcplib`. No phase pushes or tags without the owner's explicit request in
that turn.

### Phase 0: accept the records (this repository, docs only)

1. The owner decides the MADR. On acceptance, set the MADR to
   `status: accepted` and this PLAN to `in-progress`, and update the dates.
2. Commit `docs/reports/0001-REPORT-*`, this pair and `docs/README.md`. This
   is the bootstrap exception, so no other file is included.

### Phase 1: `mcplib` companion pair and freeze (`mcplib`, docs only)

1. Confirm `mcplib` is clean and on `main`. Record `F`. Run
   `git log --oneline v1.6.0..F -- llmprovider wizard logging/redact.go logging/mask.go`.
   If it prints anything, stop: the parity baseline would be wrong.
2. Author `mcplib/docs/decisions/0015-MADR-transfer-llmprovider-to-go-llmprovider-sdk.md`
   and its PLAN. `0015` is the highest `mcplib` number (0014) plus one.
   * **MADR.** Cite this repository's `0002-MADR-migrate-llmprovider-from-mcplib.md`
     by name. Decide:
     * the freeze from `F`;
     * deleting the moved records, as a deliberate exception to AGENTS.md
       "do not relocate historical files";
     * the relocation table;
     * the `decisions/0010` scope amendment;
     * the circuit-breaker citation;
     * the `v1.6.1` deprecation release and the `v1.7.0` removal, including
       the semantic-versioning exception.
   * **PLAN.** Its phases are this plan's Phases 9 and 13.
3. Present the pair and stop until the owner approves it. Then commit it.

### Phase 2: scaffold this repository

Copy from `mcplib` at `F` and adapt; nothing is copied blind.

1. **`.golangci.yml`:** verbatim.
2. **`.markdownlint-cli2.jsonc`:** verbatim. Its globs exclude MADR and PLAN
   files but not REPORT files, so reports are linted as in `mcplib`.
3. **`.gitignore`:** verbatim.
4. **`Makefile`:** targets `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
   `vuln` and `help`, from `mcplib` `Makefile` lines 1-48 plus `help`. Change
   the header comment to name this repository.
   * *Amended 2026-09-29:* also `pre-add-check`, per step 9.
5. **`.github/workflows/ci.yml`:**
   * From `mcplib` `ci.yml` lines 1-29: the same `actions/checkout` and
     `actions/setup-go` SHAs, the 3-OS matrix, `go-version-file: go.mod`,
     `go test ./...`, and on Linux `go vet`, `gofmt -l`, `go mod tidy -diff`
     and `golangci-lint@v2.13.1` via `make lint`.
   * Add one Linux step: `go vet -tags live_gateways ./...`. The migration
     rewrites the 23 live-tagged test files, and nothing else compiles them.
   * Omit `mcplib` lines 30-37, which are self-update scripts.
6. **`AGENTS.md`:** adapted from `mcplib` `AGENTS.md`.
   * Its purpose line names this module and its two packages.
   * Dependency rule: standard library plus `golang.org/x/term` only;
     anything else needs a MADR. Never import `mcplib` or the MCP go-sdk.
   * Records: `docs/decisions/` and `docs/reports/`, with one sequence
     across both.
   * Commits: `git commit --no-edit`.
   * Pre-add: `gofmt`, `go vet`, `go test` on the touched packages;
     `make lint` before a release.
     * *Superseded 2026-09-29 by step 11.*
   * Live tests: `go test -tags live_gateways ./llmprovider -run Live` with
     the `LLMPROVIDER_LIVE_*` switches.
7. **Verify:** `make help` lists the targets.

   The workflow cannot run until code exists; its first green run is Phase
   3's verification.

**Amendment 2026-09-29.** Steps 8–12 below come from the MADR's
"Amendment 2026-09-29: pre-add gate and agent pointers". Step 12 is the
phase's verification; it includes step 7's check.

8. **`scripts/go-precheck.sh`.** Adapt `magic-cli-remote`'s
   `scripts/go-precheck.sh`.
   * Keep:
     * `gofmt -l`;
     * per-file `golint`;
     * the missing-tool check with its install hints;
     * exit codes 0, 1 and 2;
     * `GO_PRECHECK_SKIP_VULN`.
   * Add `go vet` and `go test` over the unique package directories of the
     given files, or `./...` when the script is given no arguments.
   * **`govulncheck`:**
     * exit status 3 fails;
     * any other non-zero status warns and passes only when the output
       matches a network-failure pattern;
     * anything else fails.
   * **No Go file:** print that and exit 0. Never call `gofmt` with an empty
     file list, which makes it read standard input.
   * **Header comment:** name the caller that exists on this host, the agent
     gate `~/.agents/hooks/lib/precommit-checks.sh` at `git commit`, not
     `pre-add-go.sh`.
   * Make it executable before staging, so that `git ls-files -s` records
     mode `100755`. The agent gate only uses the script when it is
     executable.
9. **`Makefile`.** Add a `pre-add-check` target, with a `##` help text and
   `FILES ?=`, that runs `./scripts/go-precheck.sh $(FILES)`.
10. **Per-agent pointer files.**
    * `.claude/rules/madr-and-plan-skill.md`,
      `.grok/rules/madr-plan-before-mutating-work.md` and
      `.opencode/rules.md` each carry the same three things:
      * the `madr-and-plan-writing` skill name, and the command that checks
        it against the filesystem;
      * the read-only versus mutating gate;
      * a pointer to AGENTS.md for everything else.
    * `opencode.json` has `$schema` and
      `"instructions": [".opencode/rules.md"]` and nothing else.
    * `.claude/.gitignore` has `settings.local.json`.
11. **`AGENTS.md`**, replacing step 6's pre-add bullet:
    * run `make pre-add-check` (or `FILES=...`) before staging Go files;
    * the machine-wide agent gate runs the same script at `git commit`
      whenever Go files are staged;
    * `make lint` and `make vuln` run before each release;
    * describe identifier scans without quoting the identifiers;
    * before asking the owner to push, check the outgoing commits with the
      disclosure guard itself:
      `python3 ~/.global-git-hooks/github-disclosure.py pre-push origin <url>`,
      with the ref line on standard input.
12. **Prove the checks fail first, then verify.**
    * **First-fail.** In `SCRATCH/sdk-precheck`, a clone of this repository
      with the step 8 script, plant a one-package module. Plant each defect
      in turn, and record each outcome:

      | Planted | Expected |
      |---|---|
      | an unformatted file | exit 1; the file is named under `gofmt` |
      | an exported function with no doc comment | exit 1, from `golint` |
      | `fmt.Printf("%d", "x")` | exit 1, from `go vet` |
      | a failing test | exit 1, from `go test` |
      | a call to `language.ParseAcceptLanguage` with `golang.org/x/text` v0.3.7 | exit 1, naming GO-2022-1059 |
      | a `govulncheck` shim on `PATH` that prints `dial tcp` and `proxy` and exits 3 | exit 1: the keyword rule cannot hide a finding |
      | the same shim, exiting 1 | exit 0, with the "could not reach" warning |
      | `golint` removed from `PATH` | exit 2 |
      | no Go file | exit 0, without waiting on standard input |

    * **The gate end to end.** In the scratch clone:
      * stage the unformatted file and run
        `~/.agents/hooks/lib/precommit-checks.sh <clone>`. It exits 1, and
        its report names `scripts/go-precheck.sh`;
      * fix the file. It exits 0.
    * **In this repository:**
      * `git ls-files -s scripts/go-precheck.sh` shows `100755`;
      * `make pre-add-check` exits 0;
      * `make help` lists every target, including `pre-add-check`;
      * `markdownlint-cli2` is clean for `AGENTS.md` and the three pointer
        files;
      * `git config --local user.name` and `user.email` equal the owner's
        standard identity. The execution record says they match; it does
        not quote them;
      * the disclosure guard's pre-push check over the phase commit exits 0.
    * **Record the cost.** Time the script over `llmprovider` and `wizard`
      at Phase 4, once code exists.
    * **Where the gate starts to apply.** Phase 3's `git merge` is not a
      `git commit`, so the agent gate does not intercept it. The gate first
      applies to Phase 4's commit, where the script runs over every staged
      Go file.

**Amendment 2026-09-29 (third): Phase 2b, the documentation tree to
standards.** From the MADR's "Amendment 2026-09-29 (third): repository
scaffold to standards". *Approved 2026-09-29 without step 5.* It may run
before Phase 4, because it touches no Go file.

### Phase 2b: documentation tree and CI hygiene (this repository)

1. **`README.md`.** Replace the one-line placeholder with:
   * what the module is: a Go library for LLM provider access and provider
     authentication, standard library plus `golang.org/x/term`, no binary;
   * its status, true at the commit: the `mcplib` code is imported but not
     yet re-homed, there is no `go.mod`, CI cannot pass until Phase 4, and
     the v1 API is being decided by
     `0015-MADR-canonical-sdk-api-and-module-layout.md` and
     `0016-MADR-provider-auth-and-support-baseline.md`;
   * `**Documentation:** [docs/](docs/README.md)`;
   * an "I want to…" table of at most five rows, each also in
     `docs/README.md`.
2. **`docs/architecture.md`.** The tree as it is at the commit, with no
   history or rationale:
   * the tree diagram (root files, `llmprovider/`, `wizard/`,
     `internal/redact/`, `scripts/`, `docs/` with `decisions/`, `reports/`
     and the temporary `mcplib-import/`);
   * what each Go directory holds today, and that it still imports `mcplib`
     and does not build;
   * the tooling: `make` targets, `scripts/go-precheck.sh`, CI;
   * "What is not here": `go.mod`, `docs/guides/`, the 0015 package
     layout. Each is named with the record that will add it.
3. **`docs/guides/`** is **not** created. An empty directory is not tracked,
   and a placeholder guide would tell a reader nothing. 0015-PLAN S1 creates
   it with its first guide.
4. **`docs/README.md`.** Add an `architecture.md` row to the "I want to…"
   table.
5. ~~**`.github/dependabot.yml`.** The `github-actions` ecosystem only,
   monthly, one grouped pull request, `ci` commit prefix. It follows
   `magic-cli-remote`'s file, with a header comment saying why:~~
   ~~`ci.yml` pins every action to a commit SHA, so a pin goes stale
   silently without a notifier; `gomod` is left out, because dependency
   currency here is a recorded act and `govulncheck` runs in the pre-add
   gate.~~
   *Struck 2026-09-29: the owner decided "No dependabot."*
6. **Verify.**
   * `markdownlint-cli2 README.md docs/README.md docs/architecture.md`
     is clean.
   * **Link check.** A throwaway stdlib-Python resolver in `SCRATCH` over
     `README.md`, `docs/README.md` and `docs/architecture.md` resolves every
     relative link. **First-fail:** a scratch copy with one planted bad link
     in each file; each is reported.
   * Every path `docs/architecture.md` names exists (`test -e`).
   * ~~`dependabot.yml` loads in a YAML parser available on the host; the
     execution record names which one.~~ *(Struck with step 5.)*
   * The identifier scan and the disclosure guard's deny list find nothing
     in the changed files.
7. **Commit** with `git commit --no-edit`.

**Not in Phase 2b:**

* `go.mod`: Phase 4. Until it exists, CI fails at `actions/setup-go` with
  `The specified go version file at: go.mod does not exist` (run
  36638209549 on `4ddcb54`, all three runners).
* The records under `docs/mcplib-import/`: Phase 6.
* A `LICENSE`: the owner has not chosen one. `mcplib` has none;
  `magic-cli-remote` is Apache-2.0. It needs the owner's decision first.

**Amendment 2026-09-29 (fourth): Phase 2c, golangci-lint in the pre-add
gate.** From the MADR's "Amendment 2026-09-29 (fourth): golangci-lint
replaces golint in the pre-add gate". *Approved 2026-09-29.* It may run
before Phase 4.

### Phase 2c: golangci-lint replaces golint (this repository)

1. **`.golangci.yml`.** Append to `linters.settings.revive.rules`, after
   `var-declaration`: `exported`, `package-comments`, `var-naming`. Nothing
   else changes.
2. **`scripts/go-precheck.sh`.** Replace step 2 (per-file `golint`) with
   `ocp-login`'s step 2:
   * `GOLANGCI="${GOLANGCI_LINT:-$(go env GOPATH)/bin/golangci-lint}"`, the
     same override the `Makefile` honours;
   * `"$GOLANGCI" run -c .golangci.yml ./...`, package-scoped whether or not
     files are given, with the last 40 lines of output on failure;
   * not executable: exit 2, with the install hint pinned to CI's
     `golangci-lint@v2.13.1`.

   Remove the `golint` install hint from `need()`. Update the header
   comment (why `golangci-lint`, why `./...`, citing the fourth amendment)
   and the final "clean" line. Steps 1, 3 and 4 do not change.
3. **`Makefile`.** The `pre-add-check` help text names `golangci-lint`.
4. **`AGENTS.md`.** "Pre-add checks" names `golangci-lint` with the
   repository's `.golangci.yml`, in place of per-file `golint`.
5. **Prove it fails first.** On a fresh scratch clone of this repository
   with the Phase 2c changes and a planted one-package module (as in step
   12), record each outcome:

   | Planted | Expected |
   |---|---|
   | nothing | exit 0, the "clean" line names `golangci-lint` |
   | an unformatted file | exit 1; the file is named under `gofmt` |
   | an exported function with no doc comment | exit 1; `revive` `exported` |
   | `fmt.Printf("%d\n", "x")` | exit 1, from `go vet` |
   | a failing test | exit 1, from `go test` |
   | `GOLANGCI_LINT` pointing at a missing path | exit 2, with the install hint |
   | no Go file | exit 0, without waiting on standard input |

   Also run the agent gate end to end, as step 12 does: an undocumented
   exported function staged, `~/.agents/hooks/lib/precommit-checks.sh`
   exits 1 and names `scripts/go-precheck.sh`; fixed, it exits 0.
6. **Verify in this repository.**
   * `git ls-files -s scripts/go-precheck.sh` still shows `100755`.
   * ~~`make pre-add-check` exits 0 (`go-precheck: no Go files to check.`
     until Phase 4 adds `go.mod`; the script must not reach
     `golangci-lint` without one).~~
     *Corrected 2026-09-29 (deviation, see the execution record):* Phase 3
     imported tracked Go files, so the whole-tree run already failed before
     this phase. Until Phase 4 adds `go.mod`, `make pre-add-check` exits 2,
     and its only failures are `golangci-lint`, `go vet` and `go test`
     reporting `directory prefix . does not contain main module`, as the
     pre-change run's `go vet` and `go test` did. Phase 4 requires it to
     pass.
   * `grep -rn golint` over `scripts/`, `Makefile`, `AGENTS.md` and the
     pointer files finds only the header comment's explanation.
7. **Commit** with `git commit --no-edit`.

**Not in Phase 2c:** `.markdownlint-cli2.jsonc`, which already equals
`magic-cli-remote`'s (MADR fourth amendment). The imported `0011` REPORT's
`MD004` findings are Phase 6's.

**Amendment 2026-09-29 (fifth): Phase 2d, `go.mod` in the scaffold.**
From the MADR's "Amendment 2026-09-29 (fifth): `go.mod` is part of the
scaffold". *Approved 2026-09-29.* It runs before Phase 4.

### Phase 2d: create `go.mod` and `go.sum` (this repository)

1. **`go.mod`,** exactly:

   ```text
   module github.com/maccavelli/go-llmprovider-sdk

   go 1.27.1

   require golang.org/x/term v0.43.0

   require golang.org/x/sys v0.47.0 // indirect
   ```

   No `toolchain` line. `go mod tidy` is **not** run.
2. **`go.sum`:** run
   `go get golang.org/x/term@v0.43.0 golang.org/x/sys@v0.47.0`, then restore
   step 1's `go.mod` bytes. Assert that `go.sum` has exactly four lines,
   equal to `mcplib` `4e1f9a5`'s `go.sum` lines for those two versions.
3. **`AGENTS.md`:** the purpose paragraph says "Requires Go 1.27.1."
   **"Dependencies":** add the rule from the MADR's fifth amendment: `go.mod` changes with the import that needs it; from Phase 4
   on, `go mod tidy -diff` is clean at every commit. Until Phase 4, do not
   run `go mod tidy` or `make tidy`, which would add `mcplib`.
4. **`README.md` and `docs/architecture.md`:** replace "no `go.mod`" with
   the state after this phase. The module exists; `internal/redact` builds
   and tests; `llmprovider` and `wizard` do not build until Phase 4 removes
   their `mcplib` imports. CI and `make pre-add-check` fail on those imports.
5. **Verify.**
   * `go mod verify` prints `all modules verified`.
   * `go list -m all` prints the module, `golang.org/x/sys v0.47.0` and
     `golang.org/x/term v0.43.0`, and nothing else.
   * `go test -count=1 -race ./internal/redact/` passes.
   * `go build ./...` fails, and every error names one of
     `github.com/maccavelli/mcplib`, `…/mcplib/llmprovider` or
     `…/mcplib/logging`. The distinct messages are listed in the execution
     record.
   * **Re-home proof, in `SCRATCH`:** a clone at the Phase 2d commit gets
     the Phase 4 import rewrite and the §4 orchestration change by `sed`.
     Then `go build ./...` and `go test ./...` pass, and
     `go mod tidy -diff` exits 0.
     **First-fail:** the same clone with the requirements in one `require`
     block makes `go mod tidy -diff` exit 1.
   * The disclosure guard's deny list finds nothing in the changed files.
6. **Commit** with `git commit --no-edit`.

### Phase 3: history import (this repository)

Work in `SCRATCH`; never filter the real `mcplib` checkout.

1. Clone `mcplib` at `F` into `SCRATCH/mcplib-import`.
2. Write `SCRATCH/mailmap`, mapping the author identity of commit `ebb93fe`
   to the owner's standard identity. The standard identity is the one used
   on the other 88 commits that touch the code.
3. Run `git filter-repo` in the clone with:
   * `--path llmprovider/ --path wizard/`
   * `--path logging/redact.go --path logging/redact_test.go`
   * `--path logging/mask.go --path logging/mask_test.go`
   * one `--path` per moved record file: 26 files, per MADR §10, plus the
     two `docs/decisions/0010-*` files;
   * `--path-rename logging/:internal/redact/`
   * `--path-rename docs/:docs/mcplib-import/`
   * `--mailmap SCRATCH/mailmap`
4. **Verify the filtered clone:**
   * `git ls-files` lists exactly the expected set. Compare against a
     manifest written from MADR §10 before filtering.
   * `git log --format='%ae %ce' | sort -u` contains no hostname.
5. In this repository, run
   `git fetch SCRATCH/mcplib-import HEAD:mcplib-import` and then
   `git merge --allow-unrelated-histories --no-edit mcplib-import`. Delete
   the `mcplib-import` branch afterwards.
6. **Verify:**
   * `git log --oneline -- llmprovider | wc -l` is at least 82 (it was 82 in
     `mcplib`).
   * `git blame llmprovider/identification.go` shows `mcplib`-era commits.
   * No path outside the manifest was added.

The tree does not build yet. That is expected and is fixed within Phase 4
before its commit.

### Phase 4: re-home the code (this repository)

1. ~~**`go.mod`:** `module github.com/maccavelli/go-llmprovider-sdk` and
   `go 1.26.6`. Then run `go get golang.org/x/term@v0.43.0 golang.org/x/sys@v0.47.0`
   and `go mod tidy`. Assert the result names exactly those two modules,
   with `x/sys` `// indirect`.~~
   *Amended 2026-09-29 (fifth amendment):* Phase 2d creates `go.mod` and
   `go.sum`. Here, after steps 2–7, `go mod tidy -diff` exits 0 and
   `git diff --exit-code go.mod go.sum` shows no change. Any change stops
   the phase.
2. **Import rewrite, one pass, Python in `SCRATCH`:**
   * `github.com/maccavelli/mcplib/llmprovider` → `github.com/maccavelli/go-llmprovider-sdk/llmprovider`
   * `github.com/maccavelli/mcplib/logging` → `github.com/maccavelli/go-llmprovider-sdk/internal/redact`
   * selectors `logging.RedactString` and `logging.MaskSecret` → `redact.…`
     *(Deviation 2026-09-29: `logging.RedactString` → `redact.String`.)*
3. **`internal/redact`:**
   * Rename the package clause to `redact`.
   * Rewrite the "single source of truth for secret redaction in mcplib"
     comment (`redact.go` lines 14-16) to describe this package's narrower
     role.
   * Keep only `Redact`, `RedactString` and `MaskSecret` with their tests.
     *Deviation 2026-09-29 (see the execution record):* rename
     `RedactString` to `String`, and add a `// Package redact …` doc
     comment.
4. **Orchestration.** *Replaced 2026-09-29 by the MADR's sixth amendment; the §4
   version is kept below, struck.* Delete the orchestrator surface:
   * `wizard/configure.go`: the `Orchestrated` field of `Options` with its
     comment, and the `orchestrated(o)` check at the top of `ConfigureLLM`;
   * `wizard/auth.go`: `ErrOrchestrated` with its comment, `orchestrated()`,
     and the `mcplib` root import;
   * `wizard/auth_test.go`: `TestConfigureLLM_OrchestratedReturnsErr`, and
     any import only it used;
   * **test:** add `TestConfigureLLM_IgnoresOrchestratorEnv`. With
     `MCP_ORCHESTRATOR_OWNED=true` set, `ConfigureLLM` reaches provider
     selection. It must fail first against the imported code, which
     refuses;
   * **assert:** no Go file under `wizard/` mentions `orchestrat`,
     `backplane` or `MCP_ORCHESTRATOR_OWNED`, except the new test.

   ~~In `wizard/auth.go`, remove the `mcplib` import and make `orchestrated`
   return `o.Orchestrated != nil && *o.Orchestrated`. Change the
   `ErrOrchestrated` message. Update the `Options.Orchestrated` doc comment:
   nil means not orchestrated. Tests: add
   `TestConfigureLLM_OrchestratedNilIsNotOrchestrated` (with
   `MCP_ORCHESTRATOR_OWNED=true` set, a nil option does not refuse). Keep the
   existing explicit-true refusal test.~~
5. **Identity (MADR §5):**
   * `identification.go`: rename `defaultClientName` and `mcplibModulePath`
     to `go-llmprovider-sdk` / `sdkModulePath`, and rename the local `mcplib`
     variables to `sdk`. The `User-Agent` trailer becomes
     `go-llmprovider-sdk/%s`.
   * `openai_chatgpt.go:15`: `openAIOriginatorValue`.
   * `oauth_loopback.go:387`, `oauth_device.go:157`: `referrer`.
   * `discovery.go`: comments at lines 18-40 and the `buildVersions`
     receiver name.
   * Update every test that asserts one of these strings, for example
     `oauth_loopback_test.go:85`.
6. **Environment names (MADR §6):** rename all five, in code, tests and
   `TestMain` files.
7. **Doc comments:**
   * `provider.go` lines 1-3 (package doc names two MCP servers);
   * `wizard` comments that say "mcplib" or "LLM backplane";
   * `oauth_revoke.go:14` "mcplib-owned";
   * `configure.go:47` "mcplib adds a provider";
   * `auth.go:27` "mcplib holds no token".

   Record citations are **not** touched here (Phase 7).
8. **Gates (each first seen to fail on a scratch copy, as noted in the
   execution record):**
   * **G-dep:** `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./...`
     lists only this module's packages, `golang.org/x/term` and
     `golang.org/x/sys/unix` (`x/sys/windows` under `GOOS=windows`).
     * Fail experiment: a scratch copy with one import reverted to
       `mcplib/logging`.
   * **G-name:** `grep -rn --include='*.go' 'mcplib' .` is empty
     *(amended 2026-09-29: and so is the case-insensitive `grep -rni`)*,
     except
     lines matched by the citation rule of Phase 7, which are allowed until
     then and listed.
     * Fail experiment: the scratch copy's unchanged `identification.go`
       (already observed in 0001-REPORT).
   * **G-api:** for each package, run
     `go doc -all <pkg> | sed 's#github.com/maccavelli/mcplib#M#g; s#github.com/maccavelli/go-llmprovider-sdk#M#g'`.
     * Run it at `BASE` in a scratch worktree of `mcplib` and here, then
       `diff`.
     * The diff may contain only doc-comment lines changed by steps 4–7.
       Every changed line is listed in the execution record.
     * A changed `func`, `type`, `const`, `var` or field line fails the
       gate.
       * *Amended 2026-09-29:* except the `var ErrOrchestrated` line,
         whose message MADR §4 changes.
       * *Amended again 2026-09-29 (sixth amendment):* instead, the removed
         `var ErrOrchestrated` and `Options.Orchestrated` lines and their
         doc comments.
       * G-api checks that Phases 4–5 are mechanical. It is not a release
         criterion.
     * Fail experiment: add a parameter to a scratch copy's
       `WithSessionID`.
9. **Verify:**
   * `go build ./...`;
   * `go vet ./...` with `GOOS` darwin, windows, and `CGO_ENABLED=0 GOOS=linux`;
   * `go vet -tags live_gateways ./...`;
   * `go test -count=1 -cover ./...`. Coverage must be no lower than
     `mcplib` at `F`: `llmprovider` 88.0 %, `wizard` 82.6 %;
     * *Amended 2026-09-29:* step 4 also adds a table test for
       `wizard.Level.String()`, covering all four branches. See the
       Phase 4 stop entry.
     * *Re-measured 2026-09-29 (sixth amendment), Go 1.27.1:* `wizard` 462/560
       (82.50 %) as imported; 457/555 (82.34 %) with the orchestrator
       code removed; 462/555 (83.24 %) with the `Level.String()` test.
       The floor stays 82.6 %.
   * `test -z "$(gofmt -l .)"`, `go mod tidy -diff`, `make lint`;
   * G-dep, G-name, G-api.

   With the owner's request to push, CI goes green on all three OSes.

### Phase 5: offer only credentials the caller can keep (this repository)

Implements MADR §7.

1. Write the tests first and see them fail against Phase 4's tree:
   * `TestConfigureLLM_NoTokenStoreOffersAPIKeyOnly`. For `openai` and
     `grok` with `TokenStore` nil, no method menu is shown and the API-key
     prompt follows. Use a scripted `Prompter` that records `Select` titles.
   * `TestConfigureLLM_TokenStoreOffersAllMethods`. With a `TokenStore`, the
     menu lists every descriptor method in order. This case is unchanged
     from `v1.6.0`.
2. Change `resolveCredential` in `wizard/auth.go` (lines 55-120 at `F`) to
   filter `d.AuthMethods` to `AuthAPIKey` when `o.TokenStore == nil`. When
   the filtered list holds only `AuthAPIKey`, take the existing no-menu
   path.
3. Update the `Options.TokenStore` doc comment as MADR §7 states.
4. **Verify:** Phase 4 step 9. G-api's allowed diff grows by the
   `TokenStore` doc-comment lines only.

### Phase 6: move the records (this repository)

1. `git mv` each file from `docs/mcplib-import/` to its MADR §10 path:
   * MADR and PLAN to `docs/decisions/NNNN-{MADR,PLAN}-<slug>.md`;
   * the report to `docs/reports/0011-REPORT-provider-source-compatibility-audit.md`;
   * the six 0012 plans keep their sub-slugs.

   `docs/mcplib-import/` must then be empty. Remove it.
2. **Provenance.**
   * Add `migrated-from: "mcplib <old path> @ F"` to each record's
     frontmatter. For records without frontmatter (the 0011 REPORT has no
     status), add the frontmatter with only that key.
   * Add one line under the title: `Migrated from mcplib <old path> at <F>
     under 0002-MADR-migrate-llmprovider-from-mcplib.md; record citations
     renumbered, links repaired, content otherwise unchanged.`
3. **Renumber citations in records, one mapping pass, Python in `SCRATCH`.**
   * **Number map:**
     * 0001→0003, 0003→0004, 0004→0005, 0008→0006;
     * docs/0009→0007, decisions/0009→0008;
     * docs/0010→0009, decisions/0010→0010;
     * 0011–0014 unchanged.
   * **Resolving bare 0009 and 0010 inside records:**
     * the linked path, when there is a link;
     * otherwise the resolutions recorded for this plan: "MADR 0009" means
       decisions/0009 in `0012-PLAN-oauth-hygiene`, `0012-MADR` line 783 and
       all of `decisions/0010`, and means docs/0009 in `0010-MADR`,
       `0012-PLAN-chatgpt-backend`, `0012-PLAN-gateway-conventions` and
       `0013-PLAN` lines 1697 and 2092.
   * **Unresolved cases:** the script refuses any 0009 or 0010 it cannot
     resolve by those rules, and prints it. Each refusal is resolved by
     hand and listed in the execution record.
   * **Records that stay in `mcplib`** (0002, 0005, 0006, 0007,
     `0012-PLAN-circuit-breaker-test`): become repository-named, for example
     "`mcplib` `docs/0002-MADR-xdg-compliant-user-paths.md`". This covers
     0003-MADR line 1511, 0004-MADR line 528, 0008-MADR lines 131, 551 and
     973, 0012-MADR line 1160, and the `decisions/0010` citations of
     0002/0005/0006/0007.
4. **Link repair.**
   * Relative links from former `docs/` files gain one `../` for code
     (`](../llmprovider/x.go)` → `](../../llmprovider/x.go)`, 45 in the grok
     PLAN alone) and for `](../Makefile)`.
   * Links between records are rewritten to the new filenames.
   * Links to files that stay in `mcplib`, or into sibling repositories,
     become repository-named text; they are not links.
5. **Mixed-record amendments,** each a dated section appended; nothing
   above it is rewritten:
   * `0010-MADR/PLAN` (copy): "Scope in go-llmprovider-sdk: D3–D13, P2–P7.
     D1–D2, D14–D18, P1 and P8 remain in `mcplib`
     `docs/decisions/0010-*`." Add the same note to its Status line.
   * `0012-MADR`: "Revision 4 (circuit-breaker test race) was executed by
     `mcplib` `docs/0012-PLAN-circuit-breaker-test.md`, which remains in
     `mcplib`."
   * `0005-MADR/PLAN`: "`MaskSecret` (§4, Phase 1) exists in both `mcplib`
     `logging/mask.go` and this repository's `internal/redact/mask.go`."
6. **Transfer entries (MADR §11),** a dated deviation entry in each PLAN:
   * **0009-PLAN (ranking).**
     * Transferred at `F`.
     * Phase 7's README target is now this repository's `README.md`, and
       its `README.md:128-136` / `:138` anchors refer to `mcplib` at `F`.
     * The phase gate's `phase_gate.py` and `mutate_and_test.py` live in
       this repository's `0007-PLAN` Appendices A and B.
     * A14 is re-based on this repository's import commit instead of
       `mcplib` `5a1fc70`.
     * The unlogged `6f06349` change to `opencode_route.go` is recorded as
       a deviation.
     * The environment names are now the `LLMPROVIDER_*` names.
     * The Zen DeepSeek check stays blocked on HTTP 402.
   * **0008-PLAN (OAuth loopback).**
     * Transferred.
     * P8 is re-targeted from `mcplib` `v1.5.1` (never tagged) to this
       module's `v1.0.0`, and is executed by `prepare-commit-msg`'s 0008
       companion (Phase 10).
     * A9/A10's `originator` value is now `go-llmprovider-sdk`.
     * Add a MADR amendment recording that the OpenAI redirect is
       `http://127.0.0.1:{port}/auth/callback` since `a5f2460`, superseding
       Goal 1, C3 and D1's `localhost`.
   * **0010-PLAN (token store).**
     * Transferred.
     * P2, P4, P5 and P3-D5 were delivered by 0008's P2–P6 (`381ae8a`,
       `ed25c94`, `1245496`, `694aff7`).
     * Record the conflicts: the paste-prompt wording as shipped at
       `wizard/auth.go:244`, and 0008's relaxed `TokenURL` rule.
     * Remaining here: P3-D9, P6, and P7's
       `TestListChatGPTModels_DefaultHostIsCodexNotPlatform`.
   * **0003 (grok).**
     * MADR `status: accepted`, PLAN `status: complete`, with a dated note.
     * Phase 3's Gemini endpoint was completed by 0014 (`a26ac36`).
7. **Rewrite `docs/README.md`:**
   * the record index, with all 31 record files;
   * a "Migrated from mcplib" table (old path → new path);
   * the "I want to…" matrix, with rows for adding a provider, OAuth,
     model discovery and ranking, the wizard, and live tests.
8. **Write `docs/architecture.md`,** describing the system as it is now:
   * the packages;
   * the provider and item interfaces;
   * credentials and token sources;
   * discovery and ranking;
   * the wizard's `Prompter` seam;
   * identity;
   * the dependency rule.

   It carries no history.
9. **G-links.** A resolver in `SCRATCH` checks every relative link in
   `docs/**` and `README.md`: the target exists, and a `#fragment` matches a
   heading slug.
   * Fail experiment: a planted link to a missing file, and the pre-move
     path `../llmprovider/grok.go` evaluated from `docs/decisions/`.
   * Run `markdownlint-cli2` on the non-record docs.
10. **Verify:**
    * G-links is clean;
    * `ls docs` shows only `README.md`, `architecture.md`, `decisions/` and
      `reports/`;
    * no record sits directly in `docs/`;
    * each PLAN's `associated-madr` names a file in its own directory.

### Phase 7: rewrite in-code citations (this repository)

1. **Mechanical pass, Python in `SCRATCH`, one mapping pass.** It uses the
   Phase 6 number map, with the in-code resolution rules for 0009 and 0010:
   * `0009` with `§`, "Appendix" or `0009-PLAN` → 0007;
   * `0009` with D#, F# or "open question" → 0008;
   * every `0010` → 0009.

   The script prints a before/after line for each of the roughly 100
   rewrites; the list goes in the execution record.
2. **Hand pass.** These are the cases a regex misses:
   * The eight split-line citations:
     * `api_error.go:85-86`;
     * `models_catalog.go:79-80`;
     * `llmprovider/main_test.go:8-9`;
     * `oauth_revoke.go:15-16`;
     * `provider.go:84-85`;
     * `truncation.go:6-7`;
     * `wizard/main_test.go:8-9`;
     * `model_select_edge_test.go:46-47`;
     * plus `discovery_test.go:415-416`.
   * The ten bare `§` citations that implicitly mean 0012:
     * `api_error.go:49`, `:130`, `:143`;
     * `api_error_test.go:14`;
     * `chatgpt_stream_test.go:142`, `:144`;
     * `http_helpers.go:53`, `:131-132`;
     * `probe_scope_test.go:70`.

     Each gets its record number.
   * The unnamed-plan references, which mean 0004 (from `mcplib` 0003) and
     0005 (from `mcplib` 0004):
     * `opencode_route_test.go:50`, `:175-176`;
     * `live_gateways_test.go:16`, `:73`, `:80`, `:277-278`, `:282`;
     * `descriptor_test.go:233`.
   * The eight "MADR 0010 §7" citations mean the ranking record's
     **Context** §7. They are rewritten as "0009 Context §7".
   * `live_gateways_test.go:48` "MADR 0012 amendment" names the amendment by
     its date or revision.
   * `constants.go:17` path form `docs/0003-MADR-…` →
     `docs/decisions/0004-MADR-add-gateway-llm-providers.md`.
3. **G-cite.** A `SCRATCH` checker extracts every citation form found by the
   audit:
   * `MADR NNNN`, `NNNN-(MADR|PLAN|REPORT)`, `NNNN §`, `NNNN Q#`, `NNNN PLAN`.

   It fails if a number has no record here, or if a known `mcplib`-only
   number (0002, 0005–0007) appears without the word `mcplib`.
   * Fail experiment: a scratch copy with one citation reverted to
     `MADR 0010 §2`, which now names the wrong record (the token store). The
     checker must flag it via a per-number topic list:
     * 0009 is ranking: `§1`–`§7`, "Context §7";
     * 0010 is the token store: D/F/P only.
4. **Verify:**
   * G-cite is clean, and G-name is now fully empty: no `mcplib` in any Go
     file;
   * Phase 4 step 9 checks, since tests assert runtime strings containing
     citations;
   * spot-read 20 rewritten citations against their target sections, and
     list them in the execution record.

### Phase 8: live identity gates and `v1.0.0` (this repository, owner-run)

*Precondition, added 2026-09-29:* `0015-PLAN-canonical-sdk-api-and-module-layout.md` is `complete`. The package
paths in the commands below are those of its layout: the live tests
sit in the provider packages, so run them with `./llmprovider/...`.

1. With the owner's credentials:
   * `go test -tags live_gateways ./llmprovider -run 'Live.*ChatGPT' -v` with
     `LLMPROVIDER_LIVE_CHATGPT=1`: generation accepted with
     `originator: go-llmprovider-sdk`.
   * `LLMPROVIDER_LIVE_BROWSER_LOGIN=1 go test -tags live_gateways ./llmprovider -run 'Live.*Login' -v`
     for OpenAI browser login, whose authorize URL carries the new
     `originator`.
   * The same for Grok browser (loopback) and device-code login, with the new
     `referrer`.
   * `TestLive_KiloReasoningShapes` and the OpenCode route and conventions
     tests, to confirm the `User-Agent` and Kilo editor-name change is
     accepted.
2. Any rejection stops the plan. Follow the deviation protocol; the old
   value is not kept silently.
3. Update `README.md`:
   * the `mcplib` README's "LLM providers" section (lines 98-174 at `F`),
     adapted to this module's identity and environment names;
   * an install line;
   * the package table;
   * a pointer to `docs/README.md`.

   Phase 7 of 0009 (ranking) is **not** executed here.
4. **Verify:** Phase 4 step 9 and G-links. Then, only when the owner asks,
   push and tag `v1.0.0`, and confirm
   `go list -m github.com/maccavelli/go-llmprovider-sdk@v1.0.0` resolves
   through the proxy.

### Phase 9: `mcplib` `v1.6.1` deprecation (`mcplib`, under its 0015 PLAN)

*Open 2026-09-29 (sixth amendment):* `mcp-server-magictools` and
`mcp-server-magicdev` keep importing these packages. Whether the
deprecation is tagged before they can migrate is the owner's decision.

*Decided 2026-09-29: it waits* until `mcp-server-magictools` and
`mcp-server-magicdev` can migrate.

1. Add a `// Deprecated: use github.com/maccavelli/go-llmprovider-sdk/<pkg>.`
   paragraph to the package doc of `llmprovider` and `wizard`. Change
   nothing else; the freeze allows this one change under 0015.
2. **Verify:**
   * `go vet ./...`, `go test ./...`, `make lint`;
   * a scratch program importing `mcplib/llmprovider` gets staticcheck
     `SA1019`.
3. With the owner's request, tag `v1.6.1`.

### Phase 10: `prepare-commit-msg` adopts the SDK (its 0008 companion)

*Amended 2026-09-29:* the companion adopts the 0015 API, not only new
import paths. It maps each call through
`docs/guides/migrating-from-mcplib.md` and adds the API changes to its
"behaviour gained" list. The companion is written after 0015-PLAN
S11. It may trial against a `v1.0.0-rc.N` tag.

The companion MADR and PLAN, in `prepare-commit-msg/docs/decisions/`, must
include:

1. **Import rewrite.** Replace both imports in `main.go`,
   `internal/ui/setup.go`, `internal/config/config.go` and their tests.
2. **Module requirements.** `go get github.com/maccavelli/go-llmprovider-sdk@v1.0.0`.
   ~~Keep `mcplib` for `selfupdate` (bump to `v1.6.1` or later only if the
   companion chooses).~~ *Amended 2026-09-29 (the MADR's sixth amendment,
   further decisions):* `prepare-commit-msg` drops `mcplib` entirely.
   `selfupdate` comes from `go-core-lib` (`github.com/maccavelli/go-core-lib`); this phase cannot complete before
   that release exists. Assert that `go list -m all` names no
   `github.com/maccavelli/mcplib`. Run `go mod tidy`.
3. **Folded-in 0008 P8.** This is this repository's `0008-PLAN` P8, which
   was re-targeted:
   * live-token-store isolation in `main_oauth_test.go`;
   * `ValidateOAuthSession` at config load in `internal/config/config.go`.

   Transfer its acceptance criteria.
4. **`scripts/go-precheck.py`.** Extend the `mcplib` supply-chain check
   (lines 121-175) to this module:
   *Amended 2026-09-29:* and to `go-core-lib`; the `mcplib` entry goes with
   the requirement.
   * it must be required at a release version, not a pseudo-version;
   * no `replace`;
   * no GOPRIVATE / GONOSUMDB / GONOSUMCHECK / GOINSECURE exemption;
   * `go.sum` matches.

   See the check fail on a scratch copy with a `replace`.
5. **Docs.** Update `README.md:66` and any record text that states the
   current dependency. Historical records stay as written.
6. **Reconcile.** Reconcile with its proposed `0007` dependency refresh.
7. **Orchestration.** ~~It already passes `Orchestrated: &false`. No change.~~
   *Amended 2026-09-29 (sixth amendment):* the field no longer exists. Delete
   `orchestrated := false` and `Orchestrated: &orchestrated`
   (`internal/ui/setup.go:221,232`). Its behaviour does not change: it
   never wanted the refusal.
8. **Live check.** This is MADR §12's `client_version` gate. Built from the
   `v1.0.0` tag, a ChatGPT-session model listing shows `gpt-6-sol`. Record
   the output in this PLAN's Phase 10 entry as well.

### Phase 11: `mcp-server-magictools` adopts the SDK (its 0005 companion)

*Deferred 2026-09-29 (sixth amendment):* until the owner has moved the
orchestrator code out of `mcplib`. A later amendment re-scopes this phase
then. The text below is not executable as written: `Options.Orchestrated`
no longer exists.

*Amended 2026-09-29:* the companion adopts the 0015 API, not only new
import paths. It maps each call through
`docs/guides/migrating-from-mcplib.md` and adds the API changes to its
"behaviour gained" list. The companion is written after 0015-PLAN
S11. It may trial against a `v1.0.0-rc.N` tag.

1. **Preconditions.** The tree was dirty on 2026-09-29
   (`M internal/forks/hnsw/go.mod`). That change is not the migration's.
   * The companion's first step confirms that the owner has resolved or
     accepted it.
   * The migration never stages that file.
2. **The companion must include:**
   * the import rewrite in:
     * `cmd/mcp-server-magictools/config.go`, `pterm_prompter.go`,
       `pterm_prompter_test.go`;
     * `internal/llm/provider.go`, `pool.go`, and the `llm` tests;
     * `internal/intelligence/hydrator.go`;
     * `internal/provider/catalog.go` and its tests;
   * `logging.MaskSecret` usage stays on `mcplib/logging`, which is
     unchanged;
   * requiring `go-llmprovider-sdk v1.0.0`;
   * `Options.Orchestrated` set explicitly, with the companion MADR
     choosing between `mcplib.IsOrchestratorOwned()` and `false`.
3. **Behaviour gained relative to its `v1.4.1` pin.** For each item, the
   companion MADR records a decision and each is covered by a test:
   * search-first model selection;
   * `DiscoverLimit` 20 s is now capped at 10 s;
   * metadata fetch from `models.opencode.ai` for Kilo, Zen/Go and Hugging
     Face (disable with `LLMPROVIDER_DISABLE_MODELS_METADATA`);
   * `GenerateWithRetry` stops on a server delay over 30 s and on a
     terminal `APIError`;
   * `RateLimitError` messages may carry a service message;
   * the default HTTP client is 330 s / 300 s (hydrator);
   * Kilo `data_collection: "deny"` by default;
   * the `User-Agent` names `go-llmprovider-sdk`;
   * no OAuth methods, since it passes no `TokenStore` (MADR §7).
4. **Docs.** `docs/guides/configuration.md:63` wording. Its historical
   records (0001, 0002) are not rewritten.

### Phase 12: `mcp-server-magicdev` adopts the SDK (its 0001 companion)

*Deferred 2026-09-29 (sixth amendment):* until the owner has moved the
orchestrator code out of `mcplib`. A later amendment re-scopes this phase
then. The text below is not executable as written: `Options.Orchestrated`
no longer exists.

*Amended 2026-09-29:* the companion adopts the 0015 API, not only new
import paths. It maps each call through
`docs/guides/migrating-from-mcplib.md` and adds the API changes to its
"behaviour gained" list. The companion is written after 0015-PLAN
S11. It may trial against a `v1.0.0-rc.N` tag.

1. **Docs tree.** It has no `docs/` tree. The companion creates
   `docs/decisions/` and, per the documentation standard, `docs/README.md`.
2. **The companion must include:**
   * the import rewrite in:
     * `cmd/mcp-server-magicdev/configure.go`, `pterm_prompter.go`,
       `pterm_prompter_test.go`;
     * `internal/integration/llm_client.go`, `llm_router.go`;
     * `internal/integration/llm/client.go` and its tests;
   * `internal/config/registry.go:57`'s description string;
   * adding this module to the workspace-module lists at
     `internal/handler/enrichment.go:55` and
     `internal/config/dependencies.go:92-93`;
   * requiring `go-llmprovider-sdk v1.0.0`;
   * `Options.Orchestrated` set explicitly.
3. **Behaviour gained relative to its `v1.2.0` pin.** Phase 11's list, and:
   * `WithThinkingBudget` is ignored on Gemini (Interactions API), and
     `llm_client.go:65` passes it;
   * `WithReasoningEffort` now applies to every provider;
   * `ListAvailableModels` returns at most `MaxListedModels` ranked ids.

### Phase 13: `mcplib` `v1.7.0` removal (`mcplib`, under its 0015 PLAN)

*Deferred 2026-09-29 (sixth amendment),* with Phases 11 and 12.

1. **Preconditions:**
   * Phases 10–12 are `complete` in their repositories;
   * no repository under the fleet root imports
     `github.com/maccavelli/mcplib/(llmprovider|wizard)`. Grep them all.
2. **Delete code.** `git rm -r llmprovider wizard`. Then run
   `go mod tidy`. `golang.org/x/term` stays, because `selfupdate` uses it;
   assert the `go.mod` diff is empty or shrinks only.
3. **Delete records.** `git rm` the 26 moved records. Keep:
   * `docs/decisions/0010-*`, with the dated scope amendment from MADR §10;
   * `docs/0012-PLAN-circuit-breaker-test.md`. Its `associated-madr` and
     link become the repository-named citation of go-llmprovider-sdk
     `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md`.
4. **Add `docs/README.md`** with:
   * the remaining records;
   * a relocation table (old path → go-llmprovider-sdk path);
   * a note that 0009 and 0010 were each used twice.
5. **Edit `README.md`:**
   * remove lines 36-37 (standalone LLM wording), 57 (`wizard.Prompter`),
     75, 90-91 and 98-174;
   * edit lines 10-11, 22-24 and 27-28;
   * keep lines 34-52, 69-72, 77-83, 89 and 176-210 (backplane, logging,
     self-update, develop);
   * add one sentence pointing LLM provider access to go-llmprovider-sdk.
6. **Edit `AGENTS.md`:**
   * lines 8-9: remove the `llmprovider/` and `wizard/` sentence;
   * lines 74-77: correct the legacy range from 0001–0008 to 0001–0013;
   * lines 78-79: note the recorded relocation exception.
7. **Edit `backplane_test.go:254-255`:** cite
   "go-llmprovider-sdk `0012-MADR-conform-providers-to-reference-clients.md`
   revision 4".
8. **Verify:**
   * `go build ./...`, `go vet ./...`, `go test ./...`, `gofmt`,
     `go mod tidy -diff`, `make lint`;
   * `grep -rn 'llmprovider\|wizard' --include='*.go' .` is empty;
   * `markdownlint-cli2`;
   * a link check over `mcplib/docs` and `README.md`.
9. With the owner's request, tag `v1.7.0`.

### Phase 14: close-out (this repository)

1. Fill the execution record with every phase's commits, gate outputs and
   first-fail experiments.
2. Set this PLAN to `complete` and the MADR's Confirmation items as met.
3. `docs/README.md` index is current.

## Verification

Acceptance criteria. Each maps to MADR §12:

| # | Criterion | Command or check | Phase |
|---|---|---|---|
| A1 | No `mcplib` or go-sdk dependency | G-dep | 4 |
| A2 | No `mcplib` string in Go code | G-name | 7 |
| A3 | Exported API equals `BASE` apart from allowed doc lines *(2026-09-29: and the §4 `ErrOrchestrated` line; mechanical-move check only. Sixth amendment: the removed `ErrOrchestrated` and `Options.Orchestrated` instead)* | G-api | 4, 5 |
| A4 | Every citation resolves here or is repository-named | G-cite | 7 |
| A5 | Every relative link resolves | G-links | 6, 13 |
| A6 | Builds, vets (3 OS and `live_gateways`), tests, formats, tidy, lints | Phase 4 step 9 | 4–8 |
| A7 | Coverage not lower than `mcplib` at `F` | `go test -cover` | 4–7 |
| A8 | History preserved and identifier-clean | `git log --follow`, author scan | 3 |
| A9 | Wire identity accepted live | Phase 8 step 1 | 8 |
| A10 | `client_version` shows `gpt-6-sol` from a `v1.0.0` build | Phase 10 step 8 | 10 |
| A11 | Open work transferred with dated entries | 0008, 0009, 0010 PLANs | 6 |
| A12 | Consumers green on `v1.0.0` *(2026-09-29: `prepare-commit-msg` only; 11–12 deferred)* | their companion PLANs | 10–12 |
| A13 | `mcplib` MCP-only and relocation table resolves *(deferred 2026-09-29)* | Phase 13 step 8 | 13 |
| A14 | The pre-add gate runs at every agent commit and fails on each planted defect | Phase 2 step 12 | 2 |
| A15 | Functional parity with `mcplib` `v1.6.0` under the 0015 API | G-wire, ported tests, G-parity (`0015-PLAN-canonical-sdk-api-and-module-layout.md` S-A3–S-A5) | 0015 |
| A16 | README and `docs/architecture.md` describe the tree as it is; every link resolves ~~; Dependabot watches the pinned actions~~ *(struck 2026-09-29)* | Phase 2b step 6 | 2b |
| A17 | The gate runs `golangci-lint` with golint's checks, and fails on each planted defect | Phase 2c step 5 | 2c |
| A18 | `go.mod` and `go.sum` hold exactly the MADR §2 pins, and equal Phase 4's `go mod tidy` result | Phase 2d step 5; Phase 4 step 1 | 2d, 4 |

Each new gate (G-dep, G-name, G-api, G-links, G-cite, the Phase 5 tests and
the `go-precheck.py` extension, and, since 2026-09-29, `scripts/go-precheck.sh`)
is recorded with its first-fail experiment:
what was broken on a scratch copy, and what the failure looked like.

## Rollout and Rollback

**Order.** 0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → (10, 11, 12 in any
order) → 13 → 14. Phase 5 can be dropped by the owner before it starts; the
consumer companions then carry the TokenStore handling themselves.

*Amended 2026-09-29:* 0 → … → 7 → 0015-PLAN → 8 → 9 → (10, 11, 12) → 13 → 14.

*Amended 2026-09-29 (sixth amendment):* 0 → … → 7 → 0015-PLAN → 8 → 10 → 14.
Phase 9 is open. Phases 11, 12 and 13 are deferred; Phase 14 closes what
ran and records the deferral.

**Before `v1.0.0` is tagged (Phases 0–8).** Everything is local to this
repository and `mcplib` docs.

* Rollback here is `git revert` of the phase commits. No history rewrite is
  needed once pushed.
* In `mcplib`, revert the 0015 docs commit.
* Consumers are untouched.

**After `v1.0.0`, before `mcplib` `v1.7.0` (Phases 9–12).** A tag is never
moved or deleted.

* A defect is fixed forward as `v1.0.1`.
* A consumer can revert its own migration commit and return to `mcplib`
  imports. `mcplib` `v1.6.x` still carries the packages; `v1.6.1` only adds
  deprecation notes.

**After `mcplib` `v1.7.0` (Phase 13).**

* Restoring the packages to `mcplib` would be a new decision, not a
  rollback.
* Any importer can stay on `mcplib` `v1.6.x`, which the module proxy keeps
  serving.

**Stop conditions.** Each is handled by the deviation protocol: stop,
present evidence and resolutions, record the chosen one here and in the
MADR, then continue.

* A live identity gate rejects a value.
* G-api shows a non-doc change *(other than the allowed `ErrOrchestrated`
  line, 2026-09-29; since the sixth amendment, the removed `ErrOrchestrated`
  and `Options.Orchestrated`)*.
* Coverage drops.
* A 0009 or 0010 citation cannot be resolved by the stated rules.
* The `mcplib` freeze is broken.
* A consumer tree has uncommitted changes in a file the migration must
  touch.

## Execution Record

### Phase 0: accept the records (2026-09-29)

* **Approval.** The owner answered "proceed" to the pair as presented on
  2026-09-29. That accepts the MADR and approves this PLAN's execution. The
  MADR is now `status: accepted` and this PLAN `status: in-progress`.
* **Step 2 was done by the owner, not by the agent.** Commit `1fe7bac`
  ("docs(migration): document llmprovider extraction plan") added the
  0001 REPORT, this pair and `docs/README.md`, and was pushed to
  `origin/main` with both records still `proposed`. The commit is docs
  only, so it stays within the bootstrap exception.
* **What this phase itself changed.** This entry, the two status fields,
  and the status column of `docs/README.md`.
* **Checks run before the commit.**
  * An identifier scan of `docs/` (the local account name, the e-mail
    local part, the machine hostname domain, scratch and home paths): no
    match.
  * **Correction (2026-09-29).** The first version of the line above named
    the account name it scanned for. The global pre-push disclosure guard
    refused the push, so the unpushed commit was amended under the
    unpushed-identifier rule.
  * `git rev-parse --path-format=absolute --git-path hooks` resolved to
    the global hooks directory.

### Amendment: pre-add gate and agent pointers (2026-09-29)

* **What was found.** An assessment of how `mcplib` and `magic-cli-remote`
  gate agent commits, made after the push refusal. The findings are in the
  MADR's "Amendment 2026-09-29: pre-add gate and agent pointers".
* **What was decided.** The owner asked that 0002 "include closing the test
  gaps as per magic-cli-remote's setup". The MADR's amendment of the same
  date records the decision. This PLAN gains:
  * Phase 2 steps 8–12;
  * acceptance criterion A14;
  * six in-scope files.

  Step 4 is annotated, and step 6's pre-add bullet is marked superseded.
* **What was probed for it.** A scratch module showed that `govulncheck`
  exits with status 3 on a finding. `golint` over the moved code at `F`
  reported nothing. Both are quoted in the MADR's amendment.
* **Status.** Phase 2 has not started. It waits for the owner's approval of
  this amendment.

### Phase 2: scaffold this repository (2026-09-29)

* **Approval.** The owner committed the amendment (`d8550d4`) and approved
  execution on 2026-09-29.
* **Copied verbatim from `mcplib` at `F`,** each checked with `cmp`:
  `.golangci.yml`, `.markdownlint-cli2.jsonc`, `.gitignore`.
* **Adapted:**
  * **`Makefile`.** `mcplib`'s Makefile (lines 1-48 plus `help`), with a
    header naming this repository and a `pre-add-check` target (steps 4 and
    9).
  * **`.github/workflows/ci.yml`.** `mcplib`'s lines 1-29, plus a Linux step
    `go vet -tags live_gateways ./...` (step 5).
  * **`AGENTS.md`.** Steps 6 and 11.
* **New:**
  * `scripts/go-precheck.sh`, recorded as mode `100755` (step 8);
  * the three pointer files, `opencode.json` and `.claude/.gitignore`
    (step 10).
* **First-fail (step 12).** Run on `SCRATCH/sdk-precheck`, a copy of this
  tree with a one-package module planted in it. Every case behaved as
  expected (12 of 12):

  | Case | Exit | The output line that proves it |
  |---|---|---|
  | baseline, clean package | 0 | `go-precheck: 2 file(s) clean (gofmt, golint, go vet, go test, govulncheck).` |
  | unformatted file | 1 | `p/p.go` listed under `gofmt` |
  | exported function without doc comment | 1 | `p/p.go:3:1: exported function Add should have comment or be unexported` |
  | `fmt.Printf("%d\n", "x")` | 1 | `go vet:` |
  | failing test | 1 | `go test:` |
  | `language.ParseAcceptLanguage`, `golang.org/x/text` v0.3.7 | 1 | `Vulnerability #1: GO-2022-1059` |
  | `govulncheck` shim: network words, exit 3 | 1 | `govulncheck: vulnerabilities found:` |
  | the same shim, exit 1 | 0 | `govulncheck: could not reach the vulnerability database; skipped.` |
  | `golint` removed from `PATH` | 2 | `go-precheck: golint not found in PATH.` |
  | no Go file among the arguments, standard input left open | 0 | `go-precheck: no Go files to check.` |
  | agent gate, staged unformatted file | 1 | `Go pre-commit check failed (scripts/go-precheck.sh):` |
  | agent gate, file fixed | 0 | (no output) |

  * **Comparison with `magic-cli-remote`.** Its own `scripts/go-precheck.sh`,
    run on the same copy with the exit-3 shim, exited 0 and printed
    `govulncheck: could not reach the vulnerability database; skipped.` That
    confirms the MADR amendment's claim that it can skip a real finding.
  * **Harness note.** On the first run the three `PATH` cases did not fail.
    `BASH_ENV` points at a profile loader that rebuilds `PATH` in every
    non-interactive bash, so the script never saw the changed `PATH`. The
    cases were re-run with `BASH_ENV` removed from their environment. The
    script did not change between the runs.
* **In this repository:**
  * `git ls-files -s scripts/go-precheck.sh` shows `100755`;
  * `make pre-add-check` exits 0 (`go-precheck: no Go files to check.`);
  * `make help` lists nine targets, including `pre-add-check`;
  * the local `user.name` and `user.email` match the author identity of
    `1fe7bac` and the local identity of `mcplib`;
  * the disclosure guard's deny list finds nothing in the 12 new files.
* **Deviation, 2026-09-29: two pointer files are outside the repository
  lint.**
  * **Found.** The verbatim `.markdownlint-cli2.jsonc` excludes any file
    whose name contains `madr` or `plan`. That covers
    `.claude/rules/madr-and-plan-skill.md` and
    `.grok/rules/madr-plan-before-mutating-work.md`, so step 12's
    `markdownlint-cli2` run over the four files linted only `AGENTS.md` and
    `.opencode/rules.md`. A planted `*` bullet in each file was reported in
    those two only.
  * **Resolution.** The two excluded files were linted from a scratch
    directory holding the same `config` with no `globs`. A planted `*`
    bullet failed there first (MD004); the real files were then clean.
  * **Not done.** Neither the config nor the file names changed. The config
    is verbatim by step 2, and the names are fixed by the MADR's amendment.
    The repository's own lint will keep skipping these two files.
* **Not done in this phase:**
  * the workflow has not run; it cannot until code exists (step 7's note);
  * the script's cost over real code is timed at Phase 4, per step 12.

### Phase 3: history import (2026-09-29)

* **Commit.** `7dab7f6` is the merge, with parents `699e2b8` and the
  filtered head `9ce0f8c`. The `mcplib-import` branch was deleted after the
  merge.
* **Inputs, checked before filtering:**
  * **Manifest.** MADR §10 resolves at `F` to 28 record files (26 moved and
    the two `docs/decisions/0010-*`), plus 180 code files: `llmprovider/`,
    `wizard/`, and the four `logging` redaction files.
  * **Renames.** No imported path was ever renamed in `mcplib`, so a path
    filter keeps each file's whole history.
* **Identities.**
  * Of the 140 `mcplib` commits touching the imported paths, 136 carry the
    owner's standard identity.
  * `ebb93fe` is the only one with a different e-mail: the hostname-bearing
    one. The mailmap maps it.
  * Three more differ in the name only: `9c01081` and `6da0d10` by letter
    case, `ba92db1` by another name. All three carry the owner's e-mail,
    and the disclosure guard's identity rule does not match them. They were
    left as recorded.

  The PLAN's "other 88 commits" counted only the code paths.
* **Filter (steps 1-3).**
  * `SCRATCH/mcplib-import` was cloned with `--no-local` and reset to `F`.
    HEAD had moved past `F` to the 0015 records commit, so the reset was
    needed.
  * `git filter-repo --force` was given the paths, renames and mailmap of
    step 3. `--force` is needed because the reset means the clone is no
    longer "fresh".
* **Verify (step 4).**
  * `git ls-files` shows 208 expected, 208 present, none missing, none
    extra.
  * Author and committer e-mails: one distinct value, the owner's. The
    hostname-bearing e-mail is absent.
  * **First-fail.** The same script, without the mailmap and with one
    record left out, reported one missing file, two distinct e-mails with
    the hostname-bearing one present, and exited 1.
* **Addition, 2026-09-29: `--no-tags`.** Step 5's `git fetch` would
  auto-follow the filtered clone's six rewritten `mcplib` release tags into
  this repository. The fetch ran with `--no-tags`. `git tag` was empty
  before and after.
* **Verify (step 6):**
  * `git log --oneline -- llmprovider | wc -l` → 82;
  * `git blame llmprovider/identification.go` attributes all 108 lines to
    `7807085` (2026-09-27, `mcplib`);
  * `git diff --name-only 699e2b8 HEAD` lists 208 paths: all in the
    manifest, and the manifest is complete.
* **Disclosure.** The guard's pre-push check over all 145 outgoing commits
  exited 0.
* **State.** The tree does not build. There is no `go.mod` yet, and the
  records sit in `docs/mcplib-import/`. Phases 4 and 6 fix both, as the
  PLAN states. The merge was made by `git merge`, so the agent gate did not
  run.

### Phase 4 stop, and the second amendment (2026-09-29)

* **What was found.** A dry run of Phase 4 on `SCRATCH/sdk-p4` (steps 1–8)
  passed all of these:
  * build;
  * vet for darwin, linux, windows and `live_gateways`;
  * tests, `gofmt`, `go mod tidy -diff` and `make lint`;
  * G-dep: `golang.org/x/term`, plus `golang.org/x/sys/unix`, or
    `golang.org/x/sys/windows` under `GOOS=windows`;
  * G-name, case-insensitive: 0 lines.

  The new `TestConfigureLLM_OrchestratedNilIsNotOrchestrated` failed first
  against an env-reading `orchestrated()`, with
  `ConfigureLLM() error = wizard: caller reports an orchestrated process; …`,
  then passed. Three things did not pass as the PLAN is written:
  1. **Coverage.** `wizard` measured 82.5 % against the 82.6 % floor
     (450/545 → 448/543 statements). The two covered statements §4
     removes from `orchestrated()` account for all of the drop.
  2. **G-api.** The diff held only doc-comment lines, plus the
     `var ErrOrchestrated` initialiser that §4 changes.
  3. **G-name.** The case-sensitive grep missed `withMcplibVersion` in
     `live_chatgpt_listing_test.go`. The first dry run's rename covered one
     file only. `go vet -tags live_gateways` caught it
     (`undefined: withMcplibVersion`). The rename now runs over every Go
     file.
* **The stop.** The work was stopped and presented. Nothing was applied to
  this repository.
* **The owner's answer.** It changed the frame: functional parity, and a
  canonical, extensible API. That led to
  [0015-REPORT-sdk-api-surface-assessment.md](../reports/0015-REPORT-sdk-api-surface-assessment.md),
  [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md) and [0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md), and to the MADR's "Amendment 2026-09-29 (second): functional parity and the canonical API".
* **Resolutions, approved 2026-09-29 with the second amendment and
  modified by the sixth amendment:**
  * a `Level.String()` test (Phase 4 step 9 note);
  * ~~the `ErrOrchestrated` allowance~~ the removal allowance (step 8);
  * the case-insensitive G-name (step 8).
* **Status.** ~~Phase 4 has not been applied. It resumes on approval.~~
  See "Second and sixth amendments accepted" below.

### Amendment (third): Phase 2b proposed (2026-09-29)

* **What was found.** Measured against the repository's documentation
  standard (a `README.md` that links `docs/README.md`, and a
  `docs/architecture.md`), after
  [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)
  evaluated `mcplib` and `magic-cli-remote`:
  * `README.md` is a one-line placeholder that links nothing;
  * there is no `docs/architecture.md`;
  * `ci.yml` pins actions by SHA with nothing to report a stale pin;
  * CI on `main` has failed since Phase 2, because there is no `go.mod`.
    That is expected until Phase 4 and is not changed here.
* **What is proposed.** Phase 2b, criterion A16, and two in-scope files.
* **Status.** Not started. Waits for the owner's approval.

### Amendment (fourth): Phase 2c proposed (2026-09-29)

* **What was asked.** Copy a known-good Markdown lint configuration, such as
  `magic-cli-remote`'s, and use `golangci-lint` instead of `golint`.
* **What was found.** The Markdown configuration is already byte-identical
  to `magic-cli-remote`'s. Replacing `golint` as the configuration stands
  would lose the undocumented-export check. Both are measured in the MADR's
  fourth amendment. The scratch clone was `SCRATCH/lintexp`; this tree was
  not touched.
* **What is proposed.** Phase 2c and criterion A17.
* **Status.** Not started. Waits for the owner's approval.

### Phases 2b and 2c approved (2026-09-29)

* **Approval.** The owner answered "Approve 0002 for the tooling changes.
  Approve 2b. No dependabot." That accepts the MADR's third and fourth
  amendments and approves Phases 2b and 2c.
* **Deviation, 2026-09-29: Phase 2b step 5 is struck.** No
  `.github/dependabot.yml`. Step 6's YAML check and A16's Dependabot clause
  go with it; the MADR's third amendment records the decision.
* **Deviation, 2026-09-29: Phase 2c step 6's expectation was false.**
  * **Found.** Step 6 expected `make pre-add-check` to exit 0 with
    `go-precheck: no Go files to check.` Phase 3 had already imported
    tracked Go files. Run before any Phase 2c change, with
    `GO_PRECHECK_SKIP_VULN=1`, it exited 2: `go vet` and `go test` failed
    with `pattern ./...: directory prefix . does not contain main module or
    its selected dependencies`. It is pre-existing since `7dab7f6`.
  * **Decision.** The owner chose to correct the fact. Step 6 now expects
    exactly the missing-module failure until Phase 4, which already
    requires the check to pass. The gate's behaviour is proven by step 5 on
    a scratch clone, not by the whole-tree run.
  * **Scope.** No file added.
* **Records committed first.** `b0bcba2` (docs only) holds the 0015 and
  0016 records as `proposed`, this PLAN's amendments and the approvals. It
  approves neither 0015 nor 0016.

### Phase 2c: golangci-lint replaces golint (2026-09-29)

* **Steps 1–4, as written.**
  * `.golangci.yml`: `revive` gains `exported`, `package-comments` and
    `var-naming`, after `var-declaration`.
  * `scripts/go-precheck.sh`: step 2 runs
    `"$GOLANGCI" run -c .golangci.yml ./...`, with `GOLANGCI_LINT` as the
    override and an install hint pinned to `v2.13.1`. The `golint` hint is
    gone, and the header explains the change. The file mode stays `100755`.
  * `Makefile`: the `pre-add-check` help text names `golangci-lint`.
  * `AGENTS.md`: "Pre-add checks" names the `golangci-lint` command and
    where golint's checks went.
* **Step 5, first-fail.** Run on `SCRATCH/sdk-2c`: a fresh clone at
  `b0bcba2` with the four changed files, the imported code removed and a
  one-package module planted. `golangci-lint` was 2.13.2, with
  `GO_PRECHECK_SKIP_VULN=1` and `BASH_ENV` unset. Every case behaved as
  expected (7 of 7, and the gate 2 of 2):

  | Case | Exit | The output line that proves it |
  |---|---|---|
  | baseline, clean package | 0 | `go-precheck: 2 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck).` |
  | unformatted file | 1 | `p/extra.go` listed under `gofmt`; `golangci-lint` also reports `File is not properly formatted (gofmt)` |
  | exported function without doc comment | 1 | `exported: exported function PlantedUndocumented should have comment or be unexported (revive)` |
  | `fmt.Printf("%d\n", "x")` | 1 | `go vet:` `fmt.Printf format %d has arg "x" of wrong type string`; `golangci-lint` (`govet`) as well |
  | failing test | 1 | `go test:` `--- FAIL: TestAdd` |
  | `GOLANGCI_LINT=/nonexistent/golangci-lint` | 2 | `go-precheck: golangci-lint not found at /nonexistent/golangci-lint.` |
  | no Go file, standard input held open | 0 | `go-precheck: no Go files to check.`, after 0 s |
  | agent gate, undocumented export staged | 1 | `Go pre-commit check failed (scripts/go-precheck.sh):`, then the `revive` line above |
  | agent gate, fixed | 0 | (no output) |

  * **Harness note.** The first timing of the no-Go-file case read 20 s,
    because the command substitution waited for the `sleep` feeding the
    pipe. Re-timed with standard input from a process substitution, the
    script returned at once. The script did not change between the runs.
* **Step 6, in this repository.**
  * `git ls-files -s scripts/go-precheck.sh` shows `100755`.
  * `make help` lists `pre-add-check` as
    `Runs the pre-add checks (gofmt, golangci-lint, vet, test, govulncheck)`.
  * `GO_PRECHECK_SKIP_VULN=1 make pre-add-check` exits 2. Its failures are
    `golangci-lint`
    (`typechecking error: pattern ./...: directory prefix . does not contain main module`),
    `go vet` and `go test` with the same message: the corrected expectation
    (deviation above).
  * `grep -rn golint` over `scripts/`, `Makefile`, `AGENTS.md` and the
    pointer files finds the script's header comment (2 lines) and one
    sentence in `AGENTS.md` saying that `golint` is not used and where its
    checks went. The step expected the header only; the `AGENTS.md`
    sentence is explanatory and was kept.
  * The disclosure guard's deny list (17 rules, loaded through the guard's
    own `load_rules`) finds nothing in the four files. It was first seen to
    report a planted home path in a scratch copy.
* **Also measured, before the change.** On a scratch clone of `mcplib` at
  `4e1f9a5`, `golangci-lint` 2.13.2 over `llmprovider` and `wizard` gave
  `0 issues.` with both the committed and the extended `.golangci.yml`. A
  planted undocumented function passed the committed configuration (exit
  0) and failed the extended one.
* **Status.** Phase 2c done: `3e764d7`.

### Phase 2b: documentation tree to standards (2026-09-29)

* **Steps 1, 2 and 4, as written.**
  * `README.md`: what the module is, its status (imported, not re-homed,
    no `go.mod`, CI failing until Phase 4, v1 decided by 0015 and 0016), the
    `docs/` link, and a five-row "I want to…" table.
  * `docs/architecture.md`: the tree, the three Go directories with their
    file counts and remaining `mcplib` imports, the tooling as it now
    stands (with Phase 2c's `golangci-lint`), and "What is not here".
  * `docs/README.md`: two rows, `architecture.md` and `AGENTS.md`, so every
    README row is also in the index.
* **Step 3.** `docs/guides/` was not created.
* **Step 5.** Struck (no Dependabot).
* **Found while writing, and corrected before any commit.**
  `llmprovider` has **five** wire formats, not four: OpenCode's Google route
  uses Gemini `generateContent` (`llmprovider/opencode_route.go:50-51`),
  while the `gemini` provider uses Interactions. `docs/architecture.md` says
  five. `0016-MADR-provider-auth-and-support-baseline.md` M1 was corrected
  before it was first committed. `0015-MADR-canonical-sdk-api-and-module-layout.md`
  D2 still says "The four wire formats"; it is not this plan's record, and
  it is reported to the owner for its S0.
* **Step 6, verification.**
  * **Link check.** A stdlib-Python resolver over `README.md`,
    `docs/README.md` and `docs/architecture.md`: 0 bad links. **First-fail:**
    a scratch copy with one planted link in each file reported all three
    (`missing file: nope/missing.md`).
  * **Markdown lint.** `markdownlint-cli2` with the repository's
    configuration reports no finding in `README.md`, `docs/README.md`,
    `docs/architecture.md` or `AGENTS.md`. The run exits 1 on 188 `MD004`
    findings, all in `docs/mcplib-import/0011-REPORT-provider-source-compatibility-audit.md`
    (Phase 6). A planted `*` bullet in each of the four files, in a scratch
    copy, was reported in each, so the configuration does cover them.
  * **Paths.** Every path in the `architecture.md` tree exists (`test -e`);
    `docs/guides` does not.
  * **Identifiers.** The disclosure guard's deny list finds nothing in the
    three files.
* **Status.** Phase 2b done.

### Amendment (fifth): Phase 2d proposed (2026-09-29)

* **What was said.** The owner: the scaffold "should include creating and
  updating go.mod as requirements are assessed, identified, and
  fulfilled".
* **What was measured.** In the MADR's fifth amendment. In scratch clones:
  the §2 pins in `go mod tidy`'s layout equal the tidy result after a
  simulated re-home, and `go.sum` equals `mcplib`'s four lines.
* **What is proposed.** Phase 2d, criterion A18, and an amended Phase 4
  step 1.
* **Revised 2026-09-29, before approval: `go 1.27.1`.** The owner asked why
  the scaffold would use `go 1.26.6` when the fleet's Go and tooling are at
  1.27.1. The MADR's fifth amendment now also supersedes §2's directive,
  with the gate evidence. Step 1 says `go 1.27.1`, and step 3 updates
  AGENTS.md's "Requires Go".
* **Effect on Phase 2c step 6.** After Phase 2d, `make pre-add-check` still
  exits 2 until Phase 4. The message changes from `does not contain main
  module` to `no required module provides package
  github.com/maccavelli/mcplib…`. Phase 2d's execution record will quote
  it.
* **Status.** Approved 2026-09-29: the owner answered "proceed" to the
  revised proposal (`go 1.27.1`). That accepts the MADR's fifth amendment.

### Phase 2d: `go.mod` and `go.sum` (2026-09-29)

* **Step 1.** `go.mod` was written with exactly the five lines of step 1,
  `go 1.27.1`, and no `toolchain` line.
* **Step 2.** `go get golang.org/x/term@v0.43.0 golang.org/x/sys@v0.47.0`
  exited 0 and left `go.mod` byte-identical to step 1 (`cmp`), so there was
  nothing to restore. `go.sum` has 4 lines, identical to `mcplib`
  `4e1f9a5`'s lines for `golang.org/x/term v0.43.0` and
  `golang.org/x/sys v0.47.0` (`diff`, no output).
* **Step 3.** `AGENTS.md`: "Requires Go 1.27.1."; "Dependencies" gains the
  rule that `go.mod` changes with the import that needs it, that
  `go mod tidy -diff` is clean at every commit from Phase 4, and that
  `go mod tidy` and `make tidy` are not run before Phase 4.
* **Step 4.** `README.md` and `docs/architecture.md` state the module, its
  Go version and requirements, that `internal/redact` builds and tests, and
  that `llmprovider`, `wizard`, the pre-add check and CI fail on the
  `mcplib` imports. The tree lists `go.mod, go.sum`; "What is not here" now
  names the re-home instead of `go.mod`.
* **Step 5, verification.**
  * `go mod verify`: `all modules verified`.
  * `go list -m all`: `github.com/maccavelli/go-llmprovider-sdk`,
    `golang.org/x/sys v0.47.0`, `golang.org/x/term v0.43.0`, and nothing
    else.
  * `go test -count=1 -race ./internal/redact/`: `ok`.
  * `go build ./...` exits 1. Every error line is one of three:
    `no required module provides package github.com/maccavelli/mcplib`,
    `…/mcplib/llmprovider` and `…/mcplib/logging`. No other error line.
  * `go mod download` exits 0, so CI now passes `setup-go` and
    `go mod download` and fails at `go test`.
  * **The pre-add check** (`GO_PRECHECK_SKIP_VULN=1`): the script exits 1
    and `make pre-add-check` exits 2 (make's own status for a failed
    recipe). The `golangci-lint`, `go vet` and `go test` sections all
    report `no required module provides package
    github.com/maccavelli/mcplib…` (10 lines), and nothing else. This
    replaces the `does not contain main module` expectation of Phase 2c
    step 6, as the fifth amendment's PLAN entry said.
  * **Re-home proof.** In `SCRATCH/rehome2d`, a clone at `21f01c3` with
    this `go.mod` and `go.sum` copied in (`cmp`-equal), the Phase 4 import
    rewrite and the §4 orchestration change were applied by `sed`. Then
    `go build ./...` and `go test -count=1 ./...` exited 0, and
    `go mod tidy -diff` exited 0.
  * **First-fail.** In the same clone, the requirements in one `require`
    block made `go mod tidy -diff` exit 1.
  * **What tidy does not check.** `x/sys v0.44.0` in `go.mod` also made
    `go mod tidy -diff` exit 1, but only because `go.sum` held the
    `v0.47.0` hashes. A `go.mod` and `go.sum` that agreed on `v0.44.0` would
    pass it. The pinned versions are therefore guarded by the
    `go list -m all` check above and by §2, not by tidy.
  * The disclosure guard's deny list finds nothing in `go.mod`, `go.sum`,
    `AGENTS.md`, `README.md`, `docs/architecture.md` and the changed
    records. The link resolver finds 0 bad links in `README.md`,
    `docs/README.md`, `docs/architecture.md` and `AGENTS.md`, and
    `markdownlint-cli2` reports nothing in them.
* **Also in this commit.**
  `0016-MADR-provider-auth-and-support-baseline.md` R7 and option B strike
  "requires Go 1.27.1" as a cost of `magic-cli-remote`, with a dated note.
  That is the correction the fifth amendment made necessary.
* **Status.** Phase 2d done. The next phase of this PLAN is Phase 4, which
  waits for the owner's decision on the MADR's second amendment.

### Second and sixth amendments accepted (2026-09-29)

* **Decisions.** The owner answered "accept all" to the second amendment,
  with 0015-MADR, and asked for its orchestration code to be explained.
  Then: "I do not want to bring over the orchestrator code. that is specific
  to mcplib and will remain in mcplib. we will only be migrating
  prepare-commit-msg to this new sdk initially until i can clean up the
  orchestrator stuff and move it out of mcplib." The MADR records both: the
  second amendment is accepted as modified by the sixth, and the sixth
  amendment is accepted.
* **What changed in this PLAN.**
  * Phase 4 step 4 is replaced: delete the orchestrator surface, with a new
    first-fail test. Step 8's G-api allowance and A3 follow it. Step 9
    carries the re-measured coverage.
  * Phase 9 is open for the owner. Phases 11, 12 and 13 are deferred. Phase
    10 step 7 deletes `prepare-commit-msg`'s `Orchestrated` lines.
  * Goals 4 and 5, the scope table, A12, A13 and the order are annotated.
* **Measured for the amendment.** A stdlib-Python script in `SCRATCH` cloned
  `mcplib` at `4e1f9a5`, set `go 1.27.1`, and measured `wizard` as
  imported, with the five items removed, and with a `Level.String()` table
  test added. Every `go vet`, `gofmt -l` and `go test` exited 0. Coverage was
  counted from the profiles: 462/560 (82.50 %), 457/555 (82.34 %), and
  462/555 (83.24 %). No orchestration, backplane or `mcplib`-root line
  remained in `wizard`.
* **Correction.** Before this measurement, the owner was told the baseline
  was 82.6 %. That figure came from a `go tool cover -func` total read
  through a pipeline. The profile count is 82.50 %, which agrees with
  `go test`'s own line.
* **Status.** Phase 4 may resume under the amended step 4. It has not
  started.

### Phase 9 and `prepare-commit-msg` decided (2026-09-29)

* **Decisions.** Phase 9 waits. `prepare-commit-msg` drops `mcplib`
  entirely, taking `selfupdate` from the owner's `go-core-lib`. Recorded in
  the MADR's sixth amendment, "The owner's further decisions".
* **What changed in this PLAN.** Phase 9 is marked as waiting. Phase 10
  step 2 strikes "keep `mcplib` for `selfupdate`" and adds the no-`mcplib`
  assertion; step 4 extends the supply-chain check to `go-core-lib`.
* **In `mcplib`.** Its 0015 pair was amended to match, uncommitted there.

### Phase 4 stop: `internal/redact` lint findings (2026-09-29)

* **Found.** A dry run of Phase 4 on a scratch clone (the phase's
  transformation, then step 9's checks and the three gates) passed every
  check except `make lint`. `make lint` (golangci-lint 2.13.2, the Phase 2c configuration) reported
  two `revive` findings, and nothing else:
  * `internal/redact/mask.go:1:1: package-comments: should have a package
    comment` — `mcplib`'s package doc for `logging` is in a file that was not
    imported;
  * `internal/redact/redact.go:81:6: exported: func name will be used as
    redact.RedactString by other packages, and that stutters; consider
    calling this String`.
  * Pre-existing? No: the package rename is this phase's step 3, and the
    `revive` rules are Phase 2c's. The two had not met before.
  * Doing nothing fails step 9, and the commit gate (`golangci-lint`) refuses
    the Go files.
* **Decision.** The owner chose to add the package doc comment and rename
  `RedactString` to `String` (3 call sites and the package's tests). Steps 2
  and 3 are annotated; the MADR's seventh amendment records it.
* **Scope.** No file added beyond Phase 4's.

### Phase 4: re-home the code (2026-09-29)

* **Approval.** The owner: "3. begin the code re-home", after the second
  and sixth amendments were accepted.
* **How.** One stdlib-Python script in `SCRATCH` (`rehome.py`) applies
  steps 2–7 to a tree and runs `gofmt -w` on what it changed; every rewrite
  asserts its match count. It ran first on a scratch clone, where step 9 and
  the gates were run and the lint stop above was found. After the owner's
  decision it ran on the scratch clone again, all green, and then on this
  tree. It changed 41 files and created one.
* **Steps, as done.**
  * **2.** 16 import paths rewritten; 7 selectors, `logging.RedactString` →
    `redact.String` and `logging.MaskSecret` → `redact.MaskSecret`.
  * **3.** `package redact` in the four files; a package doc; the
    single-source comment rewritten; `RedactString` → `String` (deviation
    above), with its 5 test call sites and 2 test messages.
  * **4.** The five orchestration items deleted; the unused `errors` import
    left `wizard/auth_test.go`. `TestConfigureLLM_IgnoresOrchestratorEnv`
    replaces `TestConfigureLLM_OrchestratedReturnsErr`. `wizard/prompter_test.go`
    adds `TestLevel_String` (four branches).
  * **5.** `go-llmprovider-sdk` as client name, `User-Agent` trailer, ChatGPT
    `originator` and Grok `referrer` (loopback and device); `sdkModulePath`;
    the local `mcplib` variables are `sdk`; the tests asserting these strings,
    and `withMcplibVersion` → `withSDKVersion`,
    `TestChatGPTListing_SendsMcplibVersion` → `…SendsSDKVersion`.
  * **6.** The five `MCPLIB_*` names → `LLMPROVIDER_*`: 14 occurrences.
  * **7.** The doc comments step 7 names, and every other comment naming
    `mcplib`, since none is a record citation under Phase 7's rule:
    `wizard/prompter.go` (package doc and one line), `text_prompter.go`,
    `configure_test.go`, `live_chatgpt_test.go`, `live_chatgpt_listing_test.go`,
    `live_oauth_login_test.go`, `model_picker_test.go`, `vendor_session_test.go`.
* **Red first.** `TestConfigureLLM_IgnoresOrchestratorEnv`, appended to a
  scratch clone of `mcplib` `4e1f9a5`, failed: `auth_test.go:529: Select
  calls = 0, want 1: the flow must reach provider selection`.
* **Step 1 and step 9, on this tree.** Every command exited 0:
  * `go build ./...`; `go vet ./...` with `GOOS=darwin`, `GOOS=windows`,
    and `CGO_ENABLED=0 GOOS=linux`; `go vet -tags live_gateways ./...`;
  * `gofmt -l .` (empty); `go mod tidy -diff`;
    `git diff --exit-code go.mod go.sum`;
  * `make lint`: `0 issues.`;
  * `go test -count=1 -cover`, statements counted from the profiles:
    `llmprovider` 3152/3533 = 89.22 % (floor 88.0 %), `wizard` 463/555 =
    83.42 % (floor 82.6 %), `internal/redact` 18/18 = 100 %;
  * `make pre-add-check`: `go-precheck: 171 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck).`
* **Gates, on this tree** (`gates.py`, stdlib Python).
  * **G-dep: pass.** Non-standard dependencies outside this module:
    `golang.org/x/sys/unix` and `golang.org/x/term` for darwin and linux;
    `golang.org/x/sys/windows` and `golang.org/x/term` for windows.
  * **G-name: pass.** 0 lines, case-insensitive.
  * **G-api: pass.** `go doc -all` of `llmprovider` and `wizard`, module
    paths normalised, against `mcplib` `4e1f9a5`: 45 doc-line changes, all
    from steps 4–7, and 0 code-line violations. The two code lines removed
    are the allowed ones: `var ErrOrchestrated = errors.New(…)` and
    `Orchestrated *bool`.
* **First-fail, on scratch copies of the re-homed tree.**
  * **G-dep:** `api_error.go`'s import reverted to `mcplib/logging` →
    `go list failed (1)`: `no required module provides package
    github.com/maccavelli/mcplib/logging`.
  * **G-name:** `identification.go` restored from `mcplib` → 17 lines
    reported, from `identification.go:12`.
  * **G-api:** `WithSessionID` given a second parameter → 2 code-line
    violations: `- func WithSessionID(id string) ProviderOption` and
    `+ func WithSessionID(id string, extra int) ProviderOption`.
* **Documents made true.** `AGENTS.md` no longer forbids `go mod tidy`
  "until Phase 4". `README.md` and `docs/architecture.md` describe the
  re-homed tree: no `mcplib` imports, `package redact`, 12 `wizard` test
  files, and 0015 accepted. Their lint and links are clean.
* **Not done.**
  * CI has not run on this commit; that needs a push, which needs the
    owner's request.
  * The wire identity is not yet confirmed live: Phase 8.
* **Status.** Phase 4 done. Phase 5 is next.
