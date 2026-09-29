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
   * It is tagged `v1.0.0`.
2. The 26 LLM records of `mcplib` live in `docs/decisions/` and
   `docs/reports/`, renumbered per MADR §10. The mixed `decisions/0010` pair
   is copied, and every citation and link resolves.
3. The open work of `mcplib` records 0010 (ranking), `decisions/0009` and
   `decisions/0010` is transferred into this repository's 0009, 0008 and
   0010, executable but not executed.
4. `prepare-commit-msg`, `mcp-server-magictools` and `mcp-server-magicdev`
   import this module, each under its own approved record.
5. `mcplib` `v1.7.0` no longer contains `llmprovider/`, `wizard/` or the
   moved records. A relocation table points readers here.

## Scope

### Repositories and the record each change is made under

| Repository | Record that authorises the change | Phases |
|---|---|---|
| go-llmprovider-sdk (this) | this PLAN | 0, 2–8, 14 |
| mcplib | `docs/decisions/0015-{MADR,PLAN}-transfer-llmprovider-to-go-llmprovider-sdk.md` (written in Phase 1) | 1, 9, 13 |
| prepare-commit-msg | `docs/decisions/0008-{MADR,PLAN}-adopt-go-llmprovider-sdk.md` | 10 |
| mcp-server-magictools | `docs/decisions/0005-{MADR,PLAN}-adopt-go-llmprovider-sdk.md` | 11 |
| mcp-server-magicdev | `docs/decisions/0001-{MADR,PLAN}-adopt-go-llmprovider-sdk.md` | 12 |

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

1. **`go.mod`:** `module github.com/maccavelli/go-llmprovider-sdk` and
   `go 1.26.6`. Then run `go get golang.org/x/term@v0.43.0 golang.org/x/sys@v0.47.0`
   and `go mod tidy`. Assert the result names exactly those two modules,
   with `x/sys` `// indirect`.
2. **Import rewrite, one pass, Python in `SCRATCH`:**
   * `github.com/maccavelli/mcplib/llmprovider` → `github.com/maccavelli/go-llmprovider-sdk/llmprovider`
   * `github.com/maccavelli/mcplib/logging` → `github.com/maccavelli/go-llmprovider-sdk/internal/redact`
   * selectors `logging.RedactString` and `logging.MaskSecret` → `redact.…`
3. **`internal/redact`:**
   * Rename the package clause to `redact`.
   * Rewrite the "single source of truth for secret redaction in mcplib"
     comment (`redact.go` lines 14-16) to describe this package's narrower
     role.
   * Keep only `Redact`, `RedactString` and `MaskSecret` with their tests.
4. **Orchestration (MADR §4):**
   * In `wizard/auth.go`, remove the `mcplib` import and make `orchestrated`
     return `o.Orchestrated != nil && *o.Orchestrated`.
   * Change the `ErrOrchestrated` message.
   * Update the `Options.Orchestrated` doc comment: nil means not
     orchestrated.
   * Tests: add `TestConfigureLLM_OrchestratedNilIsNotOrchestrated` (with
     `MCP_ORCHESTRATOR_OWNED=true` set, a nil option does not refuse). Keep
     the existing explicit-true refusal test.
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
   * **G-name:** `grep -rn --include='*.go' 'mcplib' .` is empty, except
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
     * Fail experiment: add a parameter to a scratch copy's
       `WithSessionID`.
9. **Verify:**
   * `go build ./...`;
   * `go vet ./...` with `GOOS` darwin, windows, and `CGO_ENABLED=0 GOOS=linux`;
   * `go vet -tags live_gateways ./...`;
   * `go test -count=1 -cover ./...`. Coverage must be no lower than
     `mcplib` at `F`: `llmprovider` 88.0 %, `wizard` 82.6 %;
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

1. Add a `// Deprecated: use github.com/maccavelli/go-llmprovider-sdk/<pkg>.`
   paragraph to the package doc of `llmprovider` and `wizard`. Change
   nothing else; the freeze allows this one change under 0015.
2. **Verify:**
   * `go vet ./...`, `go test ./...`, `make lint`;
   * a scratch program importing `mcplib/llmprovider` gets staticcheck
     `SA1019`.
3. With the owner's request, tag `v1.6.1`.

### Phase 10: `prepare-commit-msg` adopts the SDK (its 0008 companion)

The companion MADR and PLAN, in `prepare-commit-msg/docs/decisions/`, must
include:

1. **Import rewrite.** Replace both imports in `main.go`,
   `internal/ui/setup.go`, `internal/config/config.go` and their tests.
2. **Module requirements.** `go get github.com/maccavelli/go-llmprovider-sdk@v1.0.0`.
   Keep `mcplib` for `selfupdate` (bump to `v1.6.1` or later only if the
   companion chooses). Run `go mod tidy`.
3. **Folded-in 0008 P8.** This is this repository's `0008-PLAN` P8, which
   was re-targeted:
   * live-token-store isolation in `main_oauth_test.go`;
   * `ValidateOAuthSession` at config load in `internal/config/config.go`.

   Transfer its acceptance criteria.
4. **`scripts/go-precheck.py`.** Extend the `mcplib` supply-chain check
   (lines 121-175) to this module:
   * it must be required at a release version, not a pseudo-version;
   * no `replace`;
   * no GOPRIVATE / GONOSUMDB / GONOSUMCHECK / GOINSECURE exemption;
   * `go.sum` matches.

   See the check fail on a scratch copy with a `replace`.
5. **Docs.** Update `README.md:66` and any record text that states the
   current dependency. Historical records stay as written.
6. **Reconcile.** Reconcile with its proposed `0007` dependency refresh.
7. **Orchestration.** It already passes `Orchestrated: &false`. No change.
8. **Live check.** This is MADR §12's `client_version` gate. Built from the
   `v1.0.0` tag, a ChatGPT-session model listing shows `gpt-6-sol`. Record
   the output in this PLAN's Phase 10 entry as well.

### Phase 11: `mcp-server-magictools` adopts the SDK (its 0005 companion)

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
| A3 | Exported API equals `BASE` apart from allowed doc lines | G-api | 4, 5 |
| A4 | Every citation resolves here or is repository-named | G-cite | 7 |
| A5 | Every relative link resolves | G-links | 6, 13 |
| A6 | Builds, vets (3 OS and `live_gateways`), tests, formats, tidy, lints | Phase 4 step 9 | 4–8 |
| A7 | Coverage not lower than `mcplib` at `F` | `go test -cover` | 4–7 |
| A8 | History preserved and identifier-clean | `git log --follow`, author scan | 3 |
| A9 | Wire identity accepted live | Phase 8 step 1 | 8 |
| A10 | `client_version` shows `gpt-6-sol` from a `v1.0.0` build | Phase 10 step 8 | 10 |
| A11 | Open work transferred with dated entries | 0008, 0009, 0010 PLANs | 6 |
| A12 | Consumers green on `v1.0.0` | their companion PLANs | 10–12 |
| A13 | `mcplib` MCP-only and relocation table resolves | Phase 13 step 8 | 13 |
| A14 | The pre-add gate runs at every agent commit and fails on each planted defect | Phase 2 step 12 | 2 |

Each new gate (G-dep, G-name, G-api, G-links, G-cite, the Phase 5 tests and
the `go-precheck.py` extension, and, since 2026-09-29, `scripts/go-precheck.sh`)
is recorded with its first-fail experiment:
what was broken on a scratch copy, and what the failure looked like.

## Rollout and Rollback

**Order.** 0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → (10, 11, 12 in any
order) → 13 → 14. Phase 5 can be dropped by the owner before it starts; the
consumer companions then carry the TokenStore handling themselves.

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
* G-api shows a non-doc change.
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
