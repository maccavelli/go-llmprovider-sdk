---
status: accepted
date: 2026-09-29
decision-makers: go-llmprovider-sdk maintainers
consulted: mcplib maintainers; owners of mcp-server-magicdev, mcp-server-magictools, prepare-commit-msg
informed: fleet consumers of github.com/maccavelli/mcplib
---
# Migrate `llmprovider` and `wizard` from mcplib into go-llmprovider-sdk as a standalone v1 module

## Context and Problem Statement

`mcplib` (`github.com/maccavelli/mcplib`) is the fleet's shared MCP library.
Its `llmprovider/` and `wizard/` packages are an LLM provider SDK that has
nothing to do with MCP. They carry roughly 11k lines of non-test code and
16k lines of tests. The design behind them is 26 `mcplib` records, and three
of those records still hold open work. The owner wants that SDK to live in
this repository: one scoped purpose, the fewest dependencies, no dependency
on the MCP go-sdk or on `mcplib`, and usable by any Go program.

[0001-REPORT-llmprovider-extraction-feasibility.md](../reports/0001-REPORT-llmprovider-extraction-feasibility.md)
established that the move is technically feasible. A scratch copy built,
vetted, linted and passed the full default test suite with no `mcplib`
import and a `go.mod` requiring only `golang.org/x/term`. This record
decides *how* the move is done so that nothing is left behind or
half-referenced. That covers the code, the records, the work still in
progress, the identity the library presents on the wire, versions, and the
three consumers.

Evidence gathered for this record (read-only, `mcplib` at `4e1f9a5` =
`v1.6.0`, 2026-09-29), beyond the report:

* **Coupling back into `mcplib`.** It is limited to these points:
  * `llmprovider` imports `mcplib/logging` for `RedactString` (`api_error.go:120,153`,
    `oauth_session.go:78`).
  * `wizard` imports `mcplib/logging` for `MaskSecret` (`auth.go:142,173`,
    `configure.go:246,258`).
  * `wizard` imports the `mcplib` root package for one call,
    `mcplib.IsOrchestratorOwned()` (`auth.go:51`). That is a read of
    `MCP_ORCHESTRATOR_OWNED`.
  * No `mcplib` code outside the two packages imports or names them. The one
    exception is a test comment, `backplane_test.go:254-255`, which cites
    "MADR 0012 revision 4".
* **The library names itself `mcplib` on the wire**, and the tests do not
  catch it:
  * the default client name (`identification.go:15`);
  * the `User-Agent` trailer (`:91`);
  * build-info version lookup keyed on the `mcplib` module path (`:16`, `:43-60`);
  * ChatGPT `originator: mcplib` (`openai_chatgpt.go:15`), which is also sent
    on the OpenAI OAuth authorize URL (`oauth_loopback.go:379`);
  * Grok OAuth `referrer=mcplib` (`oauth_loopback.go:387`, `oauth_device.go:157`);
  * the environment variables `MCPLIB_MODELS_METADATA_URL` and
    `MCPLIB_DISABLE_MODELS_METADATA` (`model_metadata.go:21-22`) and the
    test-only `MCPLIB_LIVE_CHATGPT`, `MCPLIB_LIVE_BROWSER_LOGIN` and
    `MCPLIB_LIVE_GROK_CLI`.

  With the module path changed and nothing else, every test still passes and
  the version silently becomes `(devel)` (report F3).
* **The version number is observable.** ChatGPT model listing sends the
  library's own release as `client_version`. `gpt-6-sol` and `gpt-6-luna`
  require `minimal_client_version` `0.155.0` (`mcplib`
  `0012-MADR-conform-providers-to-reference-clients.md` lines 904-906). A
  `v0.x` SDK would hide them.
* **Records.** 37 `mcplib` records were classified:
  * 23 target only `llmprovider/` / `wizard/`;
  * 8 target only `mcplib`: 0002, 0005, 0006 and 0007, each a MADR and PLAN;
  * 1 is `mcplib`-only by its code although it belongs to an LLM decision:
    `0012-PLAN-circuit-breaker-test.md`, which changes only `backplane_test.go`;
  * 5 are mixed: the 0004 pair, `0012-MADR`, and the `docs/decisions/0010`
    pair.

  `mcplib` has two unrelated records numbered 0009 and two numbered 0010
  (`docs/` versus `docs/decisions/`).
* **In-code citations.**
  * `llmprovider/` and `wizard/` cite `mcplib` records 261 times in the
    `MADR 00NN` form, in 90 files.
  * Every 0009 and 0010 citation resolves unambiguously:
    * a 0009 citation with a § or "Appendix" means the live-catalog record;
    * a 0009 citation with a D#, F# or "open question" means the OAuth-loopback
      record;
    * every 0010 citation means the ranking record.
  * No citation points at a missing section.
  * About 30 more are a regex hazard: split across lines, bare §, or an
    unnamed "plan §3.3" / "deviation D3".
* **Work in progress that targets this code:**
  * `docs/0010-PLAN-use-case-aware-default-model-ranking.md` (in-progress):
    * Phase 7 (README, close-out) remains.
    * The Zen DeepSeek live check is blocked on HTTP 402.
    * Its A14 untouched-files check fails since `6f06349`, with no deviation
      logged.
  * `docs/decisions/0009-PLAN-repair-oauth-loopback-and-session-wiring.md`
    (in-progress): P8 remains. That is `prepare-commit-msg` isolation and
    `ValidateOAuthSession` adoption, blocked on a `mcplib` `v1.5.1` that was
    never tagged. Its `localhost` redirect facts are stale since `a5f2460`.
  * `docs/decisions/0010-*-windows-stdio-oauth-tokenstore-ci.md` (proposed):
    * The LLM remainder is D9 (adopt a refreshed token when `Save` fails), P6
      (`FileTokenStore` sync, reserved names, Windows ACL) and part of P7.
    * P1 (Windows stdio) and P8 (CI parity) belong to `mcplib`.
    * P2, P4, P5 and most of P3 were already delivered by the 0009 plan.
  * `docs/0001-*` (grok) says `proposed` but is fully implemented.
  * No branch, stash or worktree holds unrecorded work.
* **Consumers.**
  * Only three repositories under the fleet root import the packages:
    `mcp-server-magicdev` (pins `v1.2.0`), `mcp-server-magictools`
    (`v1.4.1`) and `prepare-commit-msg` (`v1.6.0`). All three use both
    packages and all three keep needing `mcplib` for other packages.
  * The API used by the two older pins compiles unchanged at `v1.6.0`, but
    behaviour moved:
    * `ConfigureLLM` refuses when `MCP_ORCHESTRATOR_OWNED=true`.
    * OpenAI and Grok offer OAuth methods, which fail without a `TokenStore`
      (neither server passes one).
    * The vendor-CLI method returns an empty `APIKey`, which both servers
      would persist.
    * Model choice is search-first.
    * The HTTP client timeouts, `GenerateWithRetry` stop rules and Gemini
      wire changed.
  * `prepare-commit-msg` has no API gap. Its `scripts/go-precheck.py` checks
    `mcplib`'s supply-chain posture only.
* **Identifiers.** Moved text is already redacted (`/Users/<user>`). One
  commit in the code's history (`ebb93fe`) has a machine hostname in its
  author e-mail.

## Decision Drivers

* **One purpose, fewest dependencies.** `llmprovider` needs only the
  standard library. `wizard` needs only `golang.org/x/term`. No `mcplib` and
  no MCP go-sdk.
* **A consumer migration is an import-path rewrite.** The exported API does
  not change in the move, so the three consumers can migrate mechanically
  and on their own schedule.
* **Nothing half-referenced.** Every record, code citation, link, module
  path, environment name and wire identity either moves or is deliberately
  left behind with a pointer. Each is proven by a check that has been seen to
  fail.
* **The work in progress survives the move.** Open phases stay executable
  and keep their history, and they are not silently closed or restarted.
* **Honest identity.** The library identifies itself as what it is, per
  `mcplib` 0012 §1.4 "never impersonate". A wire-visible change is
  confirmed live before release.
* **History and rationale stay traceable.** `git log --follow` and
  `git blame` work in this repository, and every moved record says where it
  came from.
* **Each repository's own process is honoured.** `mcplib` and each consumer
  change only under a record in that repository.

## Considered Options

* **A. Full migration.** History-preserving import of code and LLM records,
  the API unchanged, `mcplib` identity replaced, records renumbered here,
  work in progress transferred, consumers migrated, and the packages then
  removed from `mcplib`.
* **B. Code-only copy.** Copy the tree without history. Leave all records in
  `mcplib` and cite them cross-repository. Transfer only the open work.
* **C. Forwarding shims in `mcplib`.** As A, but `mcplib/llmprovider` and
  `mcplib/wizard` stay as deprecated forwarding packages (aliases) for the
  whole v1 line.
* **D. `mcplib` v2.** As A, and signal the removal with a `mcplib` major
  version.
* **E. Import the SDK into `mcplib`, reversed.** Keep the code in `mcplib`
  and have the new repository re-export it.

## Decision Outcome

Chosen option: "A. Full migration", because it is the only option that
leaves this repository self-contained (code, history, rationale and open
work together) and leaves `mcplib` MCP-only. The consumer cost is still a
mechanical import rewrite, since all three consumers are known and migrate
before `mcplib` drops the packages.

The decision has these parts:

### 1. What moves

* `mcplib` `llmprovider/` (including `testdata/`) moves to `llmprovider/`.
* `mcplib` `wizard/` moves to `wizard/`.
* `mcplib` `logging/redact.go`, `logging/mask.go` and their tests move to
  `internal/redact/`. It is an unexported package, not a shared API:
  * It carries only `Redact`, `RedactString` and `MaskSecret`.
  * `mcplib/logging` keeps its own copy for `LogBuffer`, `SanitizingWriter`
    and the MCP log path.
  * The two pattern sets may drift. That is accepted, because the SDK's use
    (error messages and masked prompts) is narrow.
* The `mcplib` backplane client (`backplane.go`) and `IsOrchestratorOwned`
  stay in `mcplib`. They are MCP orchestration, not provider access.

### 2. Module, layout, toolchain

* Module path: `github.com/maccavelli/go-llmprovider-sdk`.
* Packages: `.../llmprovider`, `.../wizard`, `.../internal/redact`.
* The package names are unchanged, so consumer code changes only in import
  paths. Moved records' `llmprovider/x.go` references keep their meaning.
* `go 1.26.6`, the fleet floor (`mcplib` `0006-MADR-raise-go-toolchain-floor-to-1-26-6.md`).
* Requirements are pinned to what `mcplib` `v1.6.0` resolves:
  `golang.org/x/term v0.43.0` and indirect `golang.org/x/sys v0.47.0`. No
  dependency moves during the migration.
* No other module may be required without a MADR in this repository.

### 3. API parity

The exported API of `llmprovider` and `wizard` at the first release is
identical, identifier for identifier and signature for signature, to `mcplib`
`v1.6.0`. The only behaviour changes are those in §4, §5, §6 and §7. Doc
comments change only where they name `mcplib`, the MCP backplane or a record
number.

### 4. Orchestration is the caller's to say

`wizard` no longer reads `MCP_ORCHESTRATOR_OWNED`.

* A nil `Options.Orchestrated` now means "not orchestrated".
* `ErrOrchestrated` keeps its name and identity. Its message becomes
  generic: `wizard: caller reports an orchestrated process; provider
  credentials are managed by the orchestrator`.
* A consumer that wants the old refusal passes
  `Orchestrated: &owned` with `owned := mcplib.IsOrchestratorOwned()`.

### 5. Identity on the wire

| Item | `mcplib` value | SDK value |
|---|---|---|
| Default client name (`User-Agent` lead, Kilo editor name) | `mcplib` | `go-llmprovider-sdk` |
| `User-Agent` trailer | `mcplib/<mcplib version>` | `go-llmprovider-sdk/<sdk version>` |
| Version lookup module path | `github.com/maccavelli/mcplib` | `github.com/maccavelli/go-llmprovider-sdk` |
| ChatGPT `originator` header and OpenAI authorize-URL `originator` | `mcplib` | `go-llmprovider-sdk` |
| Grok OAuth `referrer` | `mcplib` | `go-llmprovider-sdk` |
| ChatGPT `client_version` | `mcplib`'s release `X.Y.Z` | the SDK's release `X.Y.Z` (same rule, `mcplib` 0012 §4.3) |

Each wire-visible change is confirmed by a live gate before release (§12).
If a service rejects the new value, execution stops for a decision; the old
value is not kept silently.

### 6. Environment names

| `mcplib` name | SDK name |
|---|---|
| `MCPLIB_MODELS_METADATA_URL` | `LLMPROVIDER_MODELS_METADATA_URL` |
| `MCPLIB_DISABLE_MODELS_METADATA` | `LLMPROVIDER_DISABLE_MODELS_METADATA` |
| `MCPLIB_LIVE_CHATGPT` (test) | `LLMPROVIDER_LIVE_CHATGPT` |
| `MCPLIB_LIVE_BROWSER_LOGIN` (test) | `LLMPROVIDER_LIVE_BROWSER_LOGIN` |
| `MCPLIB_LIVE_GROK_CLI` (test) | `LLMPROVIDER_LIVE_GROK_CLI` |

The old names are not read. No repository under the fleet root references
any of them outside `mcplib`'s own tests, and a silent fallback would keep an
`mcplib` name alive in a library that is not `mcplib`.

### 7. One behaviour fix: offer only credentials the caller can keep

At `v1.6.0` the wizard offers the non-API-key methods (browser OAuth, device
code, token paste, vendor-CLI import) whenever a provider's descriptor lists
them. Two consumers then either fail (`wizard: TokenStore is required for
OAuth`) or persist an empty key. The SDK changes this:

* It offers those methods only when `Options.TokenStore` is non-nil.
* When only the API-key method remains, it skips the method menu, as it
  already does for a provider without `AuthMethods`.
* The `Options.TokenStore` doc comment states that supplying it opts in to
  every non-API-key credential kind.

`prepare-commit-msg`, the only consumer that supplies a `TokenStore`, sees no
change.

### 8. Version

The first tag is `v1.0.0`:

* The API is the stable `mcplib` `v1.6.0` API.
* Any `v0.x` `client_version` is below the measured `0.155.0` threshold.
* Tags need the owner's explicit request.

### 9. History

The code, the redaction files and the moved records are imported with their
history:

* `git filter-repo` on a scratch clone, then an unrelated-histories merge.
* A mailmap normalises the one hostname-bearing author identity to the
  owner's standard identity.
* Renames to final paths are done afterwards with `git mv`, so
  `git log --follow` resolves.

### 10. Records move here and are renumbered locally

Records are placed under this repository's documentation standard: MADR and
PLAN in `docs/decisions/`, REPORT in `docs/reports/`. They take this
repository's sequence, and the mapping is fixed:

| SDK number and slug | From `mcplib` | Files |
|---|---|---|
| 0003 `add-grok-xai-llm-provider` | `docs/0001-*` | MADR, PLAN |
| 0004 `add-gateway-llm-providers` | `docs/0003-*` | MADR, PLAN |
| 0005 `canonicalize-llm-provider-configuration` | `docs/0004-*` (mixed; moves whole, its `mcplib`-only part is `MaskSecret`, which also moves) | MADR, PLAN |
| 0006 `subscription-auth-for-llm-providers` | `docs/0008-*` | MADR, PLAN |
| 0007 `live-catalog-model-search` | `docs/0009-*` | MADR, PLAN |
| 0008 `repair-oauth-loopback-and-session-wiring` | `docs/decisions/0009-*` | MADR, PLAN |
| 0009 `use-case-aware-default-model-ranking` | `docs/0010-*` | MADR, PLAN |
| 0010 `windows-stdio-oauth-tokenstore-ci` | `docs/decisions/0010-*` (mixed; **copied**, see below) | MADR, PLAN |
| 0011 `provider-source-compatibility-audit` | `docs/0011-REPORT-*` | REPORT |
| 0012 `conform-providers-to-reference-clients` | `docs/0012-*` except `0012-PLAN-circuit-breaker-test.md` | MADR, six PLANs |
| 0013 `remediate-debugging-pass-findings` | `docs/0013-*` | MADR, PLAN |
| 0014 `gemini-wire-fidelity` | `docs/decisions/0014-*` | MADR, PLAN |

* Numbers 0011–0014 keep their `mcplib` value. Most in-code citations
  (0012 ×108, 0013 ×44, 0014 ×17) therefore stay correct.
* Each moved record gets a provenance line: `migrated-from` in its
  frontmatter (`mcplib` path at the freeze commit), plus one sentence under
  the title.
* Record-to-record citations and relative links are renumbered in one
  mapping pass. They are never rewritten sequentially, because
  0003→0004→0005 would cascade.
* Citations of records that stay in `mcplib` become repository-named
  citations. Rationale text is otherwise untouched, and historical code line
  references stay as written.
* The mixed `docs/decisions/0010` pair is **copied**, not moved. Each copy
  gets a dated scope amendment:
  * here, D3–D13 and P2–P7 remain in scope;
  * in `mcplib`, D1–D2, D14–D18, P1 and P8 remain in scope.
* `0012-PLAN-circuit-breaker-test.md` stays in `mcplib`, and its
  `associated-madr` becomes a repository-named citation of this
  repository's 0012 MADR.
* `mcplib` deletes the moved originals. The gaps stay gaps. It gains a
  `docs/README.md` relocation table so a reader following an old number
  lands here.

### 11. Work in progress is transferred, not executed

The migration moves open work and makes it executable here: paths, module
path, commands and environment names. It does not execute the work. Each
transfer is a dated entry in the moved PLAN:

* **0009 (ranking).**
  * Phase 7 remains; its README target is now this repository's README.
  * The Zen DeepSeek live check remains blocked.
  * The A14 check is re-based and the unlogged `6f06349` deviation is
    recorded.
* **0008 (OAuth loopback).**
  * P8 is re-targeted from `mcplib` `v1.5.1` to this repository's `v1.0.0`.
  * It is executed inside `prepare-commit-msg`'s migration (§13).
  * The stale `localhost` redirect facts are amended to the shipped
    `127.0.0.1` redirect.
* **0010 (token store).** D9, P6 and the remaining P7 test stay open here.
* **0003 (grok).** The status is corrected to `accepted`, and the PLAN to
  `complete`. Its Gemini phase is annotated as completed by 0014.

From the freeze (PLAN Phase 1) until `mcplib` removes the packages, no
commit may change `mcplib` `llmprovider/` or `wizard/` except under that
removal.

### 12. Checks that prove nothing is half-referenced

These gates run on every relevant phase. Each is seen to fail on a
deliberately broken scratch input before it is trusted:

* **No `mcplib` in the module.** `go list -deps ./...` names no
  `github.com/maccavelli/mcplib` package or MCP go-sdk package, and no Go
  file contains the string `mcplib`.
* **API parity.** A normalised `go doc -all` dump of each package, diffed
  against `mcplib` `v1.6.0`, shows only the doc-comment changes this record
  allows.
* **Citations.** Every record citation in Go code and in records names a
  record that exists here, or is repository-named. No bare `mcplib` number
  remains.
* **Links.** Every relative link in `docs/` and the README resolves inside
  this repository.
* **Identity, live.** The following pass against the real services:
  * ChatGPT generation with the new `originator`;
  * OpenAI browser login with the new authorize-URL `originator`;
  * Grok device-code and loopback login with the new `referrer`;
  * ChatGPT listing from a consumer built on the `v1.0.0` tag that shows
    `gpt-6-sol`.
* **Build.** `go test`, `go vet` (for darwin, linux and windows, and with
  `-tags live_gateways`), `gofmt`, `go mod tidy -diff` and `make lint` are
  clean.

### 13. `mcplib` and the consumers change under their own records

* **`mcplib`** gets a companion pair,
  `docs/decisions/0015-{MADR,PLAN}-transfer-llmprovider-to-go-llmprovider-sdk.md`.
  It covers the freeze, the record deletions and relocation table, the 0010
  scope amendment, the circuit-breaker citation, README and AGENTS edits, a
  `v1.6.1` that marks both packages `Deprecated`, and a `v1.7.0` that removes
  them.
  * Removing packages in a v1 minor is a deliberate semantic-versioning
    exception. It is safe for the fleet, because every known importer
    migrates first. Any other importer gets a loud compile error on upgrade,
    never a silent change, and `v1.6.x` stays available.
* **Each consumer** gets a companion pair:
  * `prepare-commit-msg` `docs/decisions/0008-*`, which also folds in 0008
    P8 and extends `go-precheck.py` to this module;
  * `mcp-server-magictools` `docs/decisions/0005-*`;
  * `mcp-server-magicdev` `docs/decisions/0001-*`.

  Each consumer rewrites imports, requires `v1.0.0`, sets
  `Options.Orchestrated` explicitly, and assesses the behaviour it gains
  relative to its pinned `mcplib`. The PLAN lists those changes.

### Consequences

* Good, because this repository depends on nothing but the standard library
  and `golang.org/x/term`. The compile closure of a program that imports
  only `llmprovider` contains no module outside the Go project.
* Good, because `prepare-commit-msg` stops reaching the MCP go-sdk once
  `mcplib` removes `wizard`. Its only `mcplib` import then is `selfupdate`.
* Good, because the rationale for every line of moved code lives beside it:
  numbered locally, with `mcplib` numbers 0011–0014 preserved, and history
  and blame intact.
* Good, because the open work lands in one place with the stale facts
  corrected, instead of being split across two repositories.
* Neutral, because the ChatGPT `client_version` falls from `1.6.0` (`mcplib`)
  to `1.0.0`. Both are above the only measured threshold, `0.155.0`.
* Bad, because this is a large, mostly mechanical change: about 100 in-code
  citation rewrites, about 30 hand-fixed ones, and link repair across 28
  record files. It is only as good as the citation and link gates.
* Bad, because `mcplib` `v1.7.0` removes public packages inside v1. That is
  a recorded semantic-versioning exception.
* Bad, because the redaction patterns now exist twice (`mcplib/logging` and
  `internal/redact`) and can drift.
* Bad, because the §5 wire-identity changes could be rejected by a service.
  The live gates exist to find out before release, and a rejection stops the
  migration for a decision.
* Bad, because §7 changes the menu for any external caller that relied on
  OAuth methods appearing without a `TokenStore`. No known caller does:
  without a `TokenStore` those methods fail today.
* Bad, because four repositories each need an approved record before they
  change, so the migration takes longer.

### Confirmation

* §12's gates pass, as recorded in the PLAN's execution record with their
  output.
* The first-fail experiment for each gate is also recorded, with what the
  failure looked like.
* `v1.0.0` is tagged only after the identity gates pass.
* The three consumer PLANs are `complete`, with their test suites green on
  `v1.0.0`.
* `mcplib` `v1.7.0` builds and tests with `llmprovider/` and `wizard/` gone.
  `go list -deps ./...` in `mcplib` shows no module path of this repository,
  and the relocation table resolves every moved number.

## Pros and Cons of the Options

### A. Full migration

The chosen option, described in §1–§13 above.

* Good, because code, history, rationale and open work are together and
  self-consistent.
* Good, because `mcplib` ends MCP-only, and the SDK ends free of `mcplib`.
* Good, because consumers migrate with an import rewrite plus explicit
  orchestration.
* Bad, because it has the most steps and touches four repositories.
* Bad, because it takes a recorded semantic-versioning exception in `mcplib`.

### B. Code-only copy, records stay in `mcplib`

* Good, because it is the fewest steps: no record renumbering or link
  repair.
* Bad, because every one of 261+ citations becomes a cross-repository
  reference into a repository whose 0009 and 0010 numbers are ambiguous.
* Bad, because `git blame` here starts at the copy, and the rationale lives
  in a repository the code no longer belongs to.
* Bad, because open work would still be tracked in `mcplib` for code
  `mcplib` no longer has.

### C. Forwarding shims in `mcplib` for the whole v1 line

* Good, because there is no semantic-versioning exception, and unknown
  importers never break.
* Neutral, because it costs nothing in dependencies: `mcplib` already
  requires `x/term`.
* Bad, because `mcplib` keeps the full exported surface of both
  packages as forwarders, and a dependency on this module indefinitely. That defeats the
  one-purpose goal for `mcplib`.
* Bad, because the `wizard` shim would have to re-inject
  `IsOrchestratorOwned`, so the MCP concept leaks back.

### D. `mcplib` v2 for the removal

* Good, because it follows semantic versioning strictly.
* Bad, because every one of the 13 fleet repositories that import `mcplib`
  would have to rewrite to `/v2` for a change that affects three of them.

### E. Keep the code in `mcplib`, re-export from here

* Good, because it is trivial to do.
* Bad, because it achieves none of the drivers: this module would depend on
  `mcplib` and, through `wizard`, on the MCP go-sdk.

## More Information

* Feasibility and dependency evidence:
  [0001-REPORT-llmprovider-extraction-feasibility.md](../reports/0001-REPORT-llmprovider-extraction-feasibility.md).
* Implementation:
  [0002-PLAN-migrate-llmprovider-from-mcplib.md](0002-PLAN-migrate-llmprovider-from-mcplib.md).
* Cross-repository records, cited by name. They cannot be linked from here.
  * `mcplib` `docs/0012-MADR-conform-providers-to-reference-clients.md`:
    §1.4 identification, §4.3 `client_version`, gate G-C.
  * `mcplib` `docs/0006-MADR-raise-go-toolchain-floor-to-1-26-6.md`: the Go
    floor.
  * `mcplib` `docs/0010-PLAN-use-case-aware-default-model-ranking.md`,
    `docs/decisions/0009-PLAN-repair-oauth-loopback-and-session-wiring.md`,
    `docs/decisions/0010-PLAN-windows-stdio-oauth-tokenstore-ci.md`: the work
    in progress.
* **Out of scope.** These records move with their open questions intact and
  are not answered by this decision:
  * this repository's 0004 open questions 1–8 and 10–12;
  * 0013's accepted-not-changed items;
  * 0012's "Out of scope" list;
  * 0009's out-of-scope list;
  * 0010's "Deferred" list.

  Also out of scope:
  * adopting new `GenerateThinkingWithRetry` or `ProfileCapable` in the
    consumers;
  * `mcplib` `decisions/0010` P1 (Windows stdio) and P8 (CI parity), which
    stay with `mcplib`;
  * `prepare-commit-msg`'s proposed `0007` dependency refresh, which its
    companion record must reconcile with.
* **Revisit** if a live identity gate rejects a new value, if an unknown
  importer of `mcplib/llmprovider` surfaces before `mcplib` `v1.7.0`, or if
  the owner prefers option C over the semantic-versioning exception.

## Amendment 2026-09-29: pre-add gate and agent pointers

The owner asked for this on 2026-09-29, after the global disclosure guard
refused a push of this repository. It adds to §2 and §12 and changes nothing
above.

### Observed

* **What runs before an agent commit.** A machine-wide agent gate runs before
  every agent `git commit`. For Claude Code the chain is `PreToolUse` →
  `~/.claude/hooks/precommit-gate.sh` → `~/.agents/hooks/lib/precommit-checks.sh`.
  When Go files are staged, it runs the first of these that exists:
  1. the repository's executable `scripts/go-precheck.sh`, given the staged
     files;
  2. `make pre-add-check`;
  3. `gofmt -l` and per-file `golint` only.
* **`mcplib` and this repository get only the third option.**
  * `mcplib` ships neither of the first two.
  * `mcplib`'s AGENTS.md requires `go vet` and `go test` of the touched
    packages to exit 0, but nothing enforces that at commit time.
  * As planned before this amendment, this repository would have had the
    same gap.
* **How `magic-cli-remote` closes it:**
  * `scripts/go-precheck.sh` runs `gofmt`, per-file `golint` and
    `govulncheck ./...`. `GO_PRECHECK_SKIP_VULN=1` skips `govulncheck` for
    offline work. It exits 0 when clear, 1 when a check fails, and 2 when a
    tool is missing.
  * `make pre-add-check` (`FILES=...`) runs that same script, so the manual
    check and the enforced one cannot drift.
  * Per-agent pointer files (`.claude/rules/`, `.grok/rules/`, and
    `.opencode/rules.md`, loaded by `opencode.json`) name the skill and the
    gate, and defer to AGENTS.md for everything else.
  * The repository sets a local git identity.
* **What is not true here, or not safe to copy.**
  * **The `git add` hook.** `magic-cli-remote`'s AGENTS.md says the check
    runs at `git add` through `~/.global-agent-hooks/pre-add-go.sh`. That
    directory does not exist on this Mac. The record that would install it,
    dotfiles `0008-MADR-cross-host-agent-rules-and-hooks.md`, is still
    `proposed`. Here the check runs only at `git commit`.
  * **Vulnerability findings can be skipped.** When `govulncheck` fails, the
    script treats it as "database unreachable", and passes, if the output
    contains `proxy`, `timeout`, `dial tcp`, `connection refused` or
    `no such host`. A real finding whose trace names such a symbol (any
    `.../proxy` package, any `...Timeout` function) would be skipped.
  * **How `govulncheck` reports a finding.** Probe, 2026-09-29, with
    `govulncheck` v1.7.0: a scratch module called `language.ParseAcceptLanguage`
    from `golang.org/x/text` v0.3.7. `govulncheck` reported GO-2022-1059 and
    exited with status 3.
* **The moved code already passes these checks.** At `F`, `golint` reports
  nothing for `llmprovider/...`, `wizard/...` or the four `logging` files
  that move. `go test -count=1 ./llmprovider ./wizard` passes in about 3 s.
* **None of these layers scans for identifiers.** The global pre-push
  disclosure guard is the only check (dotfiles
  `0010-MADR-github-disclosure-guard.md`). That record rejected a
  commit-time check so that GitLab work stays outside it. This amendment
  does not change that.

### Decision

1. **`scripts/go-precheck.sh`** is the single implementation of the pre-add
   rule. It is adapted from `magic-cli-remote`'s script and committed
   executable (mode `100755`). It checks the Go files it is given, or every
   tracked Go file when given none:
   * `gofmt -l`;
   * `golint`, per file;
   * `go vet` and `go test` of the packages those files belong to.
     `magic-cli-remote`'s script has no such step. It is added because this
     repository's AGENTS.md requires it, and it costs a few seconds.
   * `govulncheck ./...`:
     * exit status 3 is always a finding;
     * any other non-zero status passes with a warning only when the output
       matches a network-failure pattern;
     * `GO_PRECHECK_SKIP_VULN=1` skips it.

   When there is no Go file to check, it prints that and exits 0. That is
   the state of the tree until Phase 3.
2. **`make pre-add-check`** (`FILES ?=`) runs the script. `make vuln` stays.
3. **Per-agent pointer files.** `.claude/rules/madr-and-plan-skill.md`,
   `.grok/rules/madr-plan-before-mutating-work.md` and `.opencode/rules.md`,
   plus an `opencode.json` whose only setting loads `.opencode/rules.md`.
   * They name the `madr-and-plan-writing` skill and the gate, and point at
     AGENTS.md.
   * They carry none of `magic-cli-remote`'s history.
   * `.claude/.gitignore` ignores `settings.local.json`.
4. **AGENTS.md** states:
   * the pre-add rule as `make pre-add-check`;
   * that the agent gate enforces it at `git commit`, with no `git add` hook
     named;
   * that agents describe identifier scans without quoting the identifiers,
     and check a push with the disclosure guard itself before making it.
5. **Identity.** This repository keeps a local `user.name` and `user.email`
   equal to the owner's standard identity. They were set on 2026-09-29.
   `.git/config` is not committed, so the PLAN verifies them instead.
6. **CI does not run the script.**
   * `magic-cli-remote`'s CI does not run it either.
   * This repository's CI already runs `gofmt`, `go vet`, `go test` and
     `golangci-lint`.
   * In CI, a newly published advisory against `x/term` or `x/sys` would
     turn unrelated pushes red. `make vuln` runs before each release
     instead.

### Consequences of the amendment

* Good, because an agent commit here runs formatting, lint, vet, tests and
  a vulnerability check over what it stages.
* Good, because the manual check and the agent gate run the same file.
* Neutral, because each agent commit that stages Go files takes longer: the
  touched packages' tests plus one `govulncheck` run. Phase 2 measures it.
* Bad, because the script is a second copy of `magic-cli-remote`'s and can
  drift from it. This record does not carry the exit-status-3 fix back to
  `magic-cli-remote`.
* Bad, because nothing yet catches an identifier before a push.

### Confirmation of the amendment

The PLAN's Phase 2 records two things:

* the script failing on each planted defect in a scratch clone;
* the agent gate denying a commit through the script.
