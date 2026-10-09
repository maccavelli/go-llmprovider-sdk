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
    *Amended 2026-09-29 (seventh amendment): `RedactString` is named
    `String`, and the package has a doc comment.*
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
* ~~`go 1.26.6`, the fleet floor (`mcplib` `0006-MADR-raise-go-toolchain-floor-to-1-26-6.md`).~~
  *Superseded 2026-09-29 by the fifth amendment: `go 1.27.1`.*
* Requirements are pinned to what `mcplib` `v1.6.0` resolves:
  `golang.org/x/term v0.43.0` and indirect `golang.org/x/sys v0.47.0`. No
  dependency moves during the migration.
* No other module may be required without a MADR in this repository.

### 3. API parity

*Superseded 2026-09-29 by [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md) D1: parity is functional. See
"Amendment 2026-09-29 (second): functional parity and the canonical API" below.*

The exported API of `llmprovider` and `wizard` at the first release is
identical, identifier for identifier and signature for signature, to `mcplib`
`v1.6.0`. The only behaviour changes are those in §4, §5, §6 and §7. Doc
comments change only where they name `mcplib`, the MCP backplane or a record
number.

### 4. Orchestration is the caller's to say

*Superseded 2026-09-29 by the sixth amendment: `wizard` has no orchestration
concept. `Options.Orchestrated`, `ErrOrchestrated` and `orchestrated()`
are removed, not kept as below.*

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
  * *Superseded 2026-09-29: the API is the one [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md) defines,
    and the tag follows its PLAN. See the second amendment below.*
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

  *Amended 2026-09-29 by the sixth amendment:* only `prepare-commit-msg`
  migrates now. `mcp-server-magictools` and `mcp-server-magicdev` stay on
  `mcplib` until the owner moves the orchestrator code out of `mcplib`.
  No consumer sets `Options.Orchestrated`, which no longer exists.

  *Amended 2026-09-29 ([0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md) D9, D11, D12):*
  `prepare-commit-msg`'s companion must also read the refresh token from
  its `TokenStore`, not from `wizard.Result`; read `claude` keys from
  `ANTHROPIC_API_KEY`, not `CLAUDE_API_KEY`; and opt in to listing
  probes if it wants them.

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

## Amendment 2026-09-29 (second): functional parity and the canonical API

Status: **accepted** 2026-09-29, with [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md),
as modified by the sixth amendment (the owner: "accept all"). The owner restated the SDK's
purpose on 2026-09-29: functional parity with `mcplib`, and an API that is
canonical, modular, extensible and consistent as the SDK grows.
[0015-REPORT-sdk-api-surface-assessment.md](../reports/0015-REPORT-sdk-api-surface-assessment.md)
measured the imported API against that. 0015-MADR decides the v1 API. This
amendment changes the migration to fit, and changes nothing else above.

### What changes

1. **§3, API parity.** Superseded by 0015-MADR D1. Parity is functional:
   * every `mcplib` `v1.6.0` capability has an equivalent or a recorded
     removal;
   * the wire is unchanged except by a record.

   §3's identity rule survives in one place only: G-api checks that PLAN
   Phases 4–5 are a **mechanical** move. It is no longer a release
   criterion.
2. **§12, the API-parity gate.** G-api runs for Phases 4–5 as written, with
   one allowance. The `var ErrOrchestrated = errors.New(…)` line may change,
   because §4 requires its new message.
   *Amended 2026-09-29 (sixth amendment):* the allowance is instead the removal
   of `ErrOrchestrated` and of the `Options.Orchestrated` field, with
   their doc comments.
   * The release criterion is 0015-MADR D12: G-wire, the ported tests and
     G-parity.
   * From `v1.0.0` on, `apidiff` guards the API (0015-MADR D13).
3. **§12, the name gate.** G-name also runs case-insensitively
   (`grep -rni`). The case-sensitive `mcplib` grep missed the test helper
   `withMcplibVersion`, which only the `live_gateways` vet caught in the
   Phase 4 dry run.
4. **§2, layout.** At release the packages are those of 0015-MADR D2. The
   module path, the `llmprovider` and `wizard` names, the Go version and
   the requirements are unchanged.
5. **§6, environment names.** The names stay. From 0015-PLAN S10 on they
   are read only by opt-in helpers (0015-MADR D9), not by library code.
6. **§8, version.** `v1.0.0` is tagged after 0015-PLAN is complete and
   Phase 8's live gates pass. Release candidates `v1.0.0-rc.N` may be
   tagged earlier, on the owner's request, for consumer trials. Their
   `client_version` is `1.0.0`.
7. **§13, the consumers.** Each companion adopts the 0015 API, not only new
   import paths, using `docs/guides/migrating-from-mcplib.md`. The companions
   are written after 0015-PLAN S11, when that guide is complete.
   *Amended 2026-09-29 (sixth amendment):* only `prepare-commit-msg`'s, for now.

### Phase 4 stop of 2026-09-29, and its proposed resolution

The Phase 4 dry run passed every check except two literal rules of the PLAN.
Approving this amendment approves the resolutions below.

* **Coverage.** `wizard` measured 82.5 % against the 82.6 % floor
  (A7).
  * §4 turns `orchestrated()` from three statements into one. The two it
    removes were covered.
  * Before: 450/545 statements covered. After: 448/543. The uncovered count
    is 95 both times, and no other file's numbers change.
  * **Resolution:** add a test for the exported, fully uncovered
    `Level.String()` in `wizard/prompter.go` (5 statements). That brings
    `wizard` to 453/543 (83.4 %). The floor stays as written.
  * *Re-measured 2026-09-29 (sixth amendment), at Go 1.27.1 with the orchestrator
    code removed:* 462/555 (83.24 %) with the test. The resolution stands.
* **G-api.** The only non-doc line in the diff is §4's `ErrOrchestrated`
  initialiser. **Resolution:** the allowance in item 2.
  *As amended by the sixth amendment: the removals.*
* **G-name.** **Resolution:** item 3.

### Consequences of the amendment

* Good, because the release ships the API the SDK will keep, so consumers
  migrate once.
* Good, because Phases 4–7 still give a mechanical, fully tested baseline.
  Every behavioural claim of 0015 is measured against it.
* Bad, because `v1.0.0`, the consumer companions and `mcplib` `v1.6.1` and
  `v1.7.0` all wait for 0015-PLAN.
* Bad, because the consumer companions become code migrations, and their
  "behaviour gained" lists grow by the API changes.

## Amendment 2026-09-29 (third): repository scaffold to standards

Status: **accepted** 2026-09-29, without Dependabot (see "Owner's
decision" below). The owner asked on 2026-09-29 for the repository to be
scaffolded "to standards", alongside
[0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md).
§2 and the first amendment scaffolded the tooling. Measured against the
documentation standard the fleet's newer repositories follow (a root
`README.md` that links `docs/README.md`; a `docs/architecture.md` describing
the tree as it is; `docs/` holding only that index, that file, and
`decisions/`, `reports/` and `guides/`), three things are missing, and one
CI practice is:

* `README.md` is one line and links nothing.
* There is no `docs/architecture.md`.
* `docs/guides/` does not exist. That is correct until the first guide:
  a placeholder would tell a reader nothing.
* `ci.yml` pins its actions to commit SHAs, as `mcplib`'s does, with
  nothing to report a stale pin. `magic-cli-remote` pairs the same pinning
  with a Dependabot configuration for GitHub Actions only.

### Decision

* The PLAN gains Phase 2b: a real `README.md`, a `docs/architecture.md`
  that describes the tree at its commit, the index row, and
  ~~`.github/dependabot.yml` for the `github-actions` ecosystem only~~
  *(struck 2026-09-29 by the owner's decision below)*.
* `docs/guides/` is created by the first guide
  ([0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md)
  S1), not by the scaffold.
* `docs/mcplib-import/` stays until Phase 6 moves its records. It is the
  one directory under `docs/` outside the standard layout, and
  `docs/architecture.md` says it is temporary.
* No `LICENSE` is added. That is the owner's choice and is not decided
  here.

### Consequences of the amendment

* Good, because a reader arriving at the repository learns what it is, what
  state it is in, and where the records are.
* ~~Good, because a stale or compromised action pin is reported monthly.~~
  *(Struck 2026-09-29: no Dependabot.)*
* Neutral, because CI stays red until Phase 4 adds `go.mod`; Phase 2b does
  not touch that.
* Bad, because `docs/architecture.md` will be rewritten at Phase 4 and again
  by 0015-PLAN, as the tree changes under it.

### Owner's decision (2026-09-29)

The owner answered "Approve 2b. No dependabot." Phase 2b runs without
`.github/dependabot.yml`. The SHA-pinned actions in `ci.yml` stay without an
automated notifier; updating them remains a manual, recorded act.

## Amendment 2026-09-29 (fourth): golangci-lint replaces golint in the pre-add gate

Status: **accepted** 2026-09-29 (the owner: "Approve 0002 for the tooling
changes"). On 2026-09-29 the owner asked for the repository to
"copy a .markdownlint config from a known good repo, like magic-cli-remote
and use golangci-lint not golint".

### Observed

* **The Markdown lint configuration is already that file.**
  `.markdownlint-cli2.jsonc` here has the same MD5 as the file in 13 of the
  14 other fleet repositories with a Markdown lint configuration,
  `magic-cli-remote` and `mcplib` among them. The fourteenth,
  `mcp-server-magictools`, has a different `.markdownlint.json`. Phase 2 of
  the PLAN copied this file verbatim. Copying it again changes nothing. The
  only failures `markdownlint-cli2` reports under `docs/` are 188 `MD004`
  findings in the imported
  `docs/mcplib-import/0011-REPORT-provider-source-compatibility-audit.md`,
  which `magic-cli-remote`'s configuration reports identically.
* **`golint` is archived**, and the gate runs it while CI runs
  `golangci-lint`. `ocp-login` and `ocp-login-macos` already replaced it in
  their `scripts/go-precheck.sh`: they run
  `golangci-lint run -c .golangci.yml ./...`, package-scoped so that the
  gate and `make lint` cannot disagree.
* **This repository's `.golangci.yml` does not carry golint's checks.** Its
  `revive` rule list omits `exported`, `package-comments` and `var-naming`.
  Measured on a scratch clone of `mcplib` at `4e1f9a5`, with
  `golangci-lint` 2.13.2:

  | Configuration | Imported `llmprovider` and `wizard` | Planted `func PlantedUndocumented() int` |
  |---|---|---|
  | `.golangci.yml` as committed | exit 0, `0 issues.` | exit 0: **not reported** |
  | the same, plus the three `revive` rules | exit 0, `0 issues.` | exit 1: `exported: exported function PlantedUndocumented should have comment or be unexported (revive)` |

  Replacing `golint` without adding the rules would silently drop the
  check that §2's first-fail table proved in its second row.

### Decision

* `scripts/go-precheck.sh` runs `golangci-lint run -c .golangci.yml ./...`
  in place of per-file `golint`. Its other steps are unchanged, including
  the stricter `govulncheck` rule of the first amendment, which
  `ocp-login`'s script does not have.
* `.golangci.yml` gains the `revive` rules `exported`, `package-comments`
  and `var-naming`. It is no longer verbatim from `mcplib`.
* `.markdownlint-cli2.jsonc` is not changed.

### Consequences of the amendment

* Good, because the gate and CI apply one linter and one configuration.
* Good, because golint's documentation and naming checks survive, now
  inside `revive`.
* Bad, because the gate lints the whole module on every commit, which is
  slower than per-file `golint`. The module is small; Phase 4 times it.
* Bad, because `.golangci.yml` now differs from the fleet copy it was taken
  from.
* Neutral, because the imported `0011` REPORT still fails `MD004`. That is
  Phase 6's to resolve when the record moves, and is not decided here.

## Amendment 2026-09-29 (fifth): `go.mod` is part of the scaffold, at Go 1.27.1

Status: **accepted** 2026-09-29 (the owner: "proceed"). On 2026-09-29 the
owner said the scaffold "should
include creating and updating go.mod as requirements are assessed,
identified, and fulfilled". Until now, `go.mod` was created only by the
PLAN's Phase 4, together with the re-home.

### Observed

Measured on scratch clones of this repository at `21f01c3`, with Go 1.27.1:

* **The tree's non-standard imports** are `golang.org/x/term` (in
  `wizard`) and three `mcplib` paths: `mcplib`, `mcplib/llmprovider` and
  `mcplib/logging`. §2 pins the first. The dependency rule forbids the
  others, which Phase 4's re-home removes.
* **`go mod tidy` cannot be run before the re-home:** it adds
  `github.com/maccavelli/mcplib v1.6.0`.
* **`go get golang.org/x/term@v0.43.0` alone is not the §2 result.** It
  resolves `golang.org/x/sys v0.44.0` (x/term's minimum) and marks x/term
  `// indirect`, because it cannot see `wizard`'s import through an
  unbuildable graph.
* **The §2 pins, written in `go mod tidy`'s layout, are exactly the Phase 4
  result,** under either `go` directive (measured at 1.26.6, and again at
  1.27.1, because Go 1.27's `go mod tidy` rewrites require blocks). That layout is one `require` for `golang.org/x/term v0.43.0`,
  then one for `golang.org/x/sys v0.47.0 // indirect`, and it passes
  `go mod verify`. Its four `go.sum` lines are identical to `mcplib`
  `4e1f9a5`'s lines for those modules. After a scratch simulation of Phase
  4's import rewrite and §4's orchestration change, `go build ./...`
  passed and `go mod tidy -diff` exited 0: nothing to change. At 1.27.1,
  `go test ./...` over the re-homed scratch tree also passed. The same
  requirements in one `require` block make `go mod tidy -diff` exit 1.
* **With that `go.mod`,** `internal/redact` builds and its tests pass.
  `go build ./...` fails only on the three `mcplib` paths
  (`no required module provides package …`).

### Observed: the `go` directive

On 2026-09-29 the owner asked: "we updated all go and go tooling to
1.27.1, why would we want to scaffold this go 1.26.6?"

* **§2's `go 1.26.6` was carried from `mcplib`** and was not re-checked.
  The installed toolchain is `go1.27.1`.
* **The fleet's toolchain rule** is `magic-cli-remote`
  `docs/decisions/0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md`
  (proposed; its PLAN is in progress). Its D2: Go 1.27.1 on every host and
  in CI, and each module's `go` directive moved to 1.27 once that
  repository passes its own gates. Until then the floor is 1.26.8, so
  1.26.6 is below it. `magic-cli-remote`'s `go.mod` says `go 1.27.1`. Every
  other fleet module, `mcplib` and the three consumers included, still says
  `go 1.26.6`.
* **The directive changes behaviour, not only the minimum.** For
  `llmprovider`'s test binary, `go 1.26.6` sets
  `tracebacklabels=0,x509sslcertoverrideplatform=0` by default, and
  `go 1.27.1` sets nothing: full Go 1.27 semantics. 0169 records four Go
  1.27 changes a module may depend on, one of them `encoding/json`'s v2
  implementation, which hung a `magic-cli-remote` test.
* **The imported code passes 0169's per-repository gate at 1.27.1.** In a
  scratch clone of `mcplib` `4e1f9a5` with its directive set to
  `go 1.27.1`:
  * `go test -count=1 -race -cover` passes, and coverage is unchanged:
    `llmprovider` 89.2 %, `wizard` 82.5 %, `logging` 85.7 %;
  * `go vet` and `go vet -tags live_gateways` are clean;
  * `govulncheck` finds no vulnerabilities, with `golang.org/x/sys v0.47.0`
    and `golang.org/x/term v0.43.0`.

### Decision

* **The `go` directive is `go 1.27.1`,** superseding §2's `go 1.26.6`.
  The requirements stay §2's pins, which are advisory-free at 1.27.1.
  Moving them is not part of this amendment.
* **The scaffold creates `go.mod` and `go.sum` now,** with exactly the §2
  pins in `go mod tidy`'s layout. No `mcplib` requirement is ever added:
  the imports that would need it are removed by the re-home, not
  fulfilled.
* **From then on, `go.mod` changes with the code that needs it.** A
  requirement is added in the commit that adds its first import, and
  removed in the commit that removes its last. A module outside §2 needs a
  MADR first (unchanged). From Phase 4 on, `go mod tidy -diff` must be
  clean at every commit.
* **Phase 4 step 1 no longer creates `go.mod`.** It asserts that the re-home
  leaves `go.mod` and `go.sum` unchanged and tidy-clean.

### Consequences of the amendment

* Good, because the module exists from the scaffold, so `internal/redact`
  and every later package can be built and tested as soon as it is
  self-contained.
* Good, because CI stops failing at `setup-go` for a missing file. It fails
  at `go test ./...` on the three unresolved `mcplib` imports, which names
  the work Phase 4 has left.
* Neutral, because the requirements are the ones Phase 4 would have
  written; the scratch tidy proves the two are the same.
* Good, because the module is on the fleet's current Go, with its gate run
  before the directive is set.
* Bad, because a module requiring the SDK must itself be at `go 1.27.1` or
  later: `go get` raises the consumer's directive. `prepare-commit-msg`,
  `mcp-server-magictools` and `mcp-server-magicdev` are at `go 1.26.6`, so
  each companion record under §13 must include its own move to 1.27.1 and
  0169's per-repository gate. Under 0169 D2 each would make that move
  anyway.
* Bad, because until Phase 4 the module has imports it cannot resolve.
  `go mod tidy` must not be run, and `make tidy` would add `mcplib`.
  AGENTS.md says so until Phase 4.

## Amendment 2026-09-29 (sixth): orchestration stays in `mcplib`; `prepare-commit-msg` migrates first

Status: **accepted** 2026-09-29. The owner:

> I do not want to bring over the orchestrator code. that is specific to
> mcplib and will remain in mcplib. we will only be migrating
> prepare-commit-msg to this new sdk initially until i can clean up the
> orchestrator stuff and move it out of mcplib.

The same day the owner accepted the second amendment and
[0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)
("accept all"). This amendment modifies the second amendment's G-api
allowance and item 7, and supersedes §4 and part of §13.

### Observed

* **The orchestrator surface is five items, all in `wizard`** (this
  repository at `08de832`, byte-identical to `mcplib` `4e1f9a5`):
  * the `Options.Orchestrated *bool` field (`wizard/configure.go:78-79`);
  * `ErrOrchestrated` (`wizard/auth.go:31-32`), whose message names "the
    LLM backplane";
  * `orchestrated()` (`wizard/auth.go:47-52`). With a nil option it calls
    `mcplib.IsOrchestratorOwned()`, which reads `MCP_ORCHESTRATOR_OWNED`.
    It is `wizard`'s only use of the `mcplib` root package;
  * the early return at the top of `ConfigureLLM`
    (`wizard/configure.go:106-108`);
  * `TestConfigureLLM_OrchestratedReturnsErr` (`wizard/auth_test.go:18`).

  `llmprovider` and `internal/redact` have none.
* **Who depends on it.**
  * `prepare-commit-msg` passes `Orchestrated: &orchestrated` with
    `orchestrated := false` (`internal/ui/setup.go:221,232`). It never
    wants the refusal.
  * `mcp-server-magictools` (`cmd/mcp-server-magictools/config.go:497-504`)
    and `mcp-server-magicdev` (`cmd/mcp-server-magicdev/configure.go:274-286`)
    do not set the option, so they rely on the environment fallback.
    `mcp-server-magicdev` also calls `mcplib.IsOrchestratorOwned()`
    directly (`cmd/mcp-server-magicdev/serve.go:51,119,344`,
    `internal/sync/standards_watcher.go:33`).
* **Coverage of `wizard`.** Measured on a scratch clone of `mcplib`
  `4e1f9a5` with its directive set to `go 1.27.1`, toolchain `go1.27.1`.
  Statements were counted from each coverage profile. Every `go vet`,
  `gofmt -l` and `go test` run exited 0.

  | State | Covered/total | Coverage |
  |---|---|---|
  | as imported | 462/560 | 82.50 % |
  | the five items removed | 457/555 | 82.34 % |
  | removed, plus a `Level.String()` table test | 462/555 | 83.24 % |

  The removal takes five covered statements and leaves the 98 uncovered
  ones. The Phase 4 dry run counted 450/545 for the same source, before its
  change. The totals differ between the two runs, and the cause was not
  established; each comparison above is within one run.
* **After the removal** no line in `wizard` mentions `orchestrat`,
  `backplane` or the `mcplib` root import.

### Decision

* **`wizard` has no orchestration concept.** Phase 4 deletes the five
  items instead of §4's rewrite. `ConfigureLLM` never refuses on
  orchestration grounds and reads no orchestration variable. A caller that
  must not configure credentials while orchestrated checks that itself,
  before calling `ConfigureLLM`.
  * Under 0015-MADR D1 this is a recorded removal:
    `docs/guides/migrating-from-mcplib.md` lists `Options.Orchestrated` and
    `ErrOrchestrated` as removed, with this reason.
  * A new test sets `MCP_ORCHESTRATOR_OWNED=true` and asserts that
    `ConfigureLLM` reaches provider selection. It is seen to fail first
    against the imported code.
* **G-api's allowance** (second amendment, item 2) becomes the removed
  `var ErrOrchestrated` and `Options.Orchestrated` lines, with their doc
  comments, in place of a changed message.
* **Coverage.** The second amendment's resolution stands: a `Level.String()`
  test, and the floor unchanged.
* **Only `prepare-commit-msg` migrates now** (PLAN Phase 10). Its companion
  deletes its `Orchestrated` lines; its behaviour does not change.
  * PLAN Phases 11 (`mcp-server-magictools`) and 12 (`mcp-server-magicdev`)
    are deferred. They are re-scoped by a later amendment once the owner has
    moved the orchestrator code out of `mcplib`, under that repository's own
    record.
  * PLAN Phase 13 (`mcplib` `v1.7.0`) is deferred with them: its
    precondition is that no fleet repository imports `mcplib/llmprovider` or
    `mcplib/wizard`.
  * `mcplib` `docs/decisions/0015-MADR-transfer-llmprovider-to-go-llmprovider-sdk.md`
    names all three consumers in its `v1.7.0` preconditions. It needs its own
    amendment, in `mcplib`; this record does not make it.
  * **PLAN Phase 9 (`mcplib` `v1.6.1` deprecation) is left open.** Tagging
    it would mark deprecated two packages that two fleet repositories must
    keep using. They see `SA1019` only if they move to `v1.6.1` or later.
    The owner decides whether it waits.

### Consequences of the amendment

* Good, because the SDK carries no MCP concept, and `wizard` loses its only
  import of the `mcplib` root package.
* Good, because the first migration touches one consumer, which never used
  the refusal.
* Neutral, because `mcp-server-magictools` and `mcp-server-magicdev` do not
  change. They keep `mcplib`'s packages at their current pins.
* Bad, because a caller that relied on the automatic refusal must now check
  for itself. No consumer migrating now does.
* Bad, because `mcplib` keeps `llmprovider` and `wizard` longer, under its
  freeze. A fix made here reaches the other two servers only when they
  migrate.
* Bad, because 0002's goals of three migrated consumers and an MCP-only
  `mcplib` `v1.7.0` are postponed without a date.

### The owner's further decisions (2026-09-29)

Asked about PLAN Phase 9 and about `mcplib`'s own record, the owner answered:
"1. wait, 2. drop prepare-commit-msg as a mcplib consumer", and confirmed
that this means removing `mcplib` from `prepare-commit-msg` entirely. On
self-update: "i am going to extract it into a new additional go shared
library package i already have the repo for as a separate project named
go-core-lib on my github".

* **PLAN Phase 9 waits,** until `mcp-server-magictools` and
  `mcp-server-magicdev` can migrate.
* **`prepare-commit-msg` drops `mcplib` entirely.** `llmprovider` and
  `wizard` come from this module. `selfupdate`, its only other `mcplib`
  import (`main.go:23`, `update.go:12`), comes from the owner's
  `go-core-lib` (`github.com/maccavelli/go-core-lib`), extracted under that repository's own records. PLAN Phase 10
  cannot complete before that release exists.
* **Not decided here:** how `mcplib`'s `selfupdate` is extracted, and when
  `magic-cli-remote` (`cmd/mcremote`, `cmd/mcrelay`,
  `internal/updateclient`) moves to it.
* **`mcplib`'s side** is recorded in `mcplib`
  `docs/decisions/0015-MADR-transfer-llmprovider-to-go-llmprovider-sdk.md`,
  "Amendment 2026-09-29 (second)".
* Bad, because Phase 10 now depends on a release in a third repository.

## Amendment 2026-09-29 (seventh): `redact.String`

Status: **accepted** 2026-09-29, by the owner's choice at a Phase 4 stop.

The Phase 4 dry run renamed the package `logging` to `redact` (PLAN step 3),
which Phase 2c's `revive` rules had not seen before. `make lint` (golangci-lint 2.13.2, the Phase 2c configuration) reported
  two `revive` findings, and nothing else:
  * `internal/redact/mask.go:1:1: package-comments: should have a package
    comment` — `mcplib`'s package doc for `logging` is in a file that was not
    imported;
  * `internal/redact/redact.go:81:6: exported: func name will be used as
    redact.RedactString by other packages, and that stutters; consider
    calling this String`.

The owner chose "Doc + rename to String": `internal/redact` gains a
`// Package redact …` doc comment, and `RedactString` becomes `String`, so a
caller writes `redact.String(s)`. The package is internal; nothing outside
this module sees the change. §1's "carries only" list is annotated. The rejected
alternative kept `RedactString` under a different package name, which would
have changed §1's `internal/redact` path.

## Amendment 2026-10-01: Apache License 2.0

Status: **accepted** 2026-10-01 (the owner: "make the license apache-2.0
for this repo, the go-llmprovider-sdk, and the go-core-lib repos").

The Phase 2b deferral ("No `LICENSE` is added. That is the owner's
choice and is not decided here.") is closed. The licence is decided by
[0018-MADR-apache-2-license.md](0018-MADR-apache-2-license.md): Apache
License 2.0, the fleet `LICENSE` copy with the appendix unfilled. This
amendment does not rewrite the Phase 2b text; it records that the
choice 0002 left open has been made.

## Amendment 2026-10-03: `go-core-lib` is now `go-selfupdate-lib`

Status: **accepted** 2026-10-03, with the owner's approval of the
outstanding rename items. It corrects a name, and changes no decision.

* **What happened.** The owner renamed `go-core-lib` to
  `go-selfupdate-lib`, both the repository and the module path
  (go-selfupdate-lib
  `docs/decisions/0009-MADR-rename-to-go-selfupdate-lib.md`).
  * `github.com/maccavelli/go-core-lib` ends at `v1.4.1`, which `go`
    reports as deprecated.
  * `github.com/maccavelli/go-selfupdate-lib` starts at `v1.5.0`
    (`6deaa52`), with `v1.4.1`'s API. It holds `selfupdate`, the canonical
    `update` command (`selfupdate/cli`) and the build stamps it reads
    (`buildinfo`).
* **What changes in "The owner's further decisions".** `prepare-commit-msg`
  takes `selfupdate` from `github.com/maccavelli/go-selfupdate-lib`, at
  `v1.5.0` or later, in place of `go-core-lib`.
* **The release PLAN Phase 10 waited for exists.** Phase 10 could not
  complete "before that release exists". `go-selfupdate-lib` `v1.5.0` is
  released, and so is this module's `v1.0.0`.
* **The text above stays as written.** Its `go-core-lib` mentions record
  the name the library had when they were written.

## Amendment 2026-10-08: the `go` directive is `go 1.27.2`

By [0029-MADR-go-1-27-2-for-standard-library-fixes.md](0029-MADR-go-1-27-2-for-standard-library-fixes.md),
accepted 2026-10-08. The fifth amendment's `go 1.27.1` is superseded:
govulncheck found ten called standard-library vulnerabilities at 1.27.1, all
fixed in 1.27.2, and CI, which installs the version `go.mod` names, failed on
them. The fifth amendment stands as the record of why the directive was set
then.
