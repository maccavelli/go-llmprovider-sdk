---
date: 2026-09-29
subject: technical feasibility of extracting mcplib/llmprovider (and its configuration wizard) into go-llmprovider-sdk
examines: "mcplib at 4e1f9a5: llmprovider/, wizard/, logging/; consumers mcp-server-magicdev, mcp-server-magictools, prepare-commit-msg"
---
# LLM Provider Extraction Feasibility: mcplib/llmprovider to go-llmprovider-sdk

This report records findings. It decides nothing. Each finding names the
decision it bears on; those decisions belong in a MADR in this repository
(`docs/decisions/`), and the work in its PLAN.

## Summary

Extraction is **feasible at low technical risk**. The evidence:

- `llmprovider` already depends on the Go standard library plus exactly one
  in-repository package, `mcplib/logging`, for one function
  (`RedactString`, 3 call sites). It imports nothing from the MCP go-sdk and
  no third-party module. (M)
- The MCP go-sdk reaches LLM consumers only through `wizard`, which imports
  the `mcplib` root package for one call: `mcplib.IsOrchestratorOwned()`,
  a one-line environment check. (M, R)
- A scratch copy of `llmprovider` + `wizard` + the redaction helpers,
  re-homed under `github.com/maccavelli/go-llmprovider-sdk`, builds, vets
  (darwin, linux, windows), lints clean under the fleet `golangci-lint`
  config and passes every existing default-build test, with **zero**
  `mcplib` imports and a `go.mod` whose only requirement is
  `golang.org/x/term` (indirect `golang.org/x/sys`). Without `wizard` the
  module would have **no** requirements at all. (S)

The work that remains is not technical risk in the code; it is naming,
identity, sequencing and consumer migration: the library identifies itself
on the wire as `mcplib` (F3, F4), its comments cite `mcplib` records whose
numbers are ambiguous (F5), three `mcplib` plans touching this code are open
(F6), and three consumer repositories must move (F7).

## Scope and method

`mcplib` was read at `4e1f9a5` (clean tree, branch `main`, tags up to
`v1.6.0`). Consumers were read at their working trees under the same parent
directory. This repository was read at `fc41aeb` (README only).

Each finding carries an evidence level:

- **M:** measured by a command against `mcplib` at `4e1f9a5`
  (`go list -deps`, `go doc`, `go test`, `grep`, `git log`).
- **S:** observed in the scratch extraction experiment described under
  [Scratch extraction experiment](#scratch-extraction-experiment). No tree
  was modified; the copy lived in a session scratch directory.
- **R:** read in source, not executed.

No live request to any provider was made. The 23 `live_gateways`-tagged
test files were not run.

## What would move

### Size

| Unit | Non-test Go | Test Go | Notes |
|---|---|---|---|
| `llmprovider/` | 43 files, 9,521 lines | 107 files, 13,793 lines | 23 test files are `//go:build live_gateways`; `testdata/` holds SSE fixtures, an OpenCode route table and a dated ranking snapshot |
| `wizard/` | 6 files, 1,388 lines | 11 files, 2,092 lines | `ConfigureLLM`, `Prompter`, `TextPrompter` |
| `logging/redact.go`, `logging/mask.go` | 2 files, 125 lines | 2 files, 220 lines | only `RedactString` and `MaskSecret` are used |

History: 82 commits touch `llmprovider/` and 20 touch `wizard/`, the first
on 2026-07-30. (M)

Default-build coverage at `4e1f9a5`: `llmprovider` 88.0 %, `wizard` 82.6 %,
`logging` 85.7 %. (M)

### Feature set

| Area | What exists | Where |
|---|---|---|
| Providers | 9 ids: `openai`, `gemini`, `claude`, `grok`, `opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `ollama`; OpenAI also over the ChatGPT subscription backend | `constants.go`, one file per provider |
| Wire formats | Responses API, Chat Completions, Anthropic Messages, Gemini `generateContent` and Interactions, ChatGPT SSE, OpenCode per-model route table | `chatcompletions.go`, `gemini_interactions.go`, `openai_chatgpt.go`, `opencode_route.go` |
| Call shapes | `Provider` (text), `ToolProvider`, `ThinkingProvider`, `ThinkingToolProvider`, and the `Item*` family (typed message / reasoning / function-call items, `Continuer` for server-side continuation) | `provider.go`, `item.go` |
| Reasoning | Thinking budgets, reasoning effort, per-vendor wire mapping | `thinking_wire.go`, `grok_reasoning.go` |
| Errors and retry | `ErrRateLimited`, `ErrAuthFailure`, `ErrProviderUnavailable`, `ErrInvalidRequest`, `ErrQuotaExhausted`; `RateLimitError` with `Retry-After`; `APIError` with redacted, bounded messages; `IncompleteError`; `Generate*WithRetry` helpers | `provider.go`, `api_error.go` |
| Credentials | `TokenSource`, `StaticToken`, `OAuthSession` (refresh, reload, revoke), `VendorCLISession` (reads Codex and Grok CLI auth files), `TokenStore` / `FileTokenStore` with unix and windows variants | `tokenstore*.go`, `oauth_session.go`, `vendor_session.go` |
| OAuth login | Browser loopback (PKCE) and device-code flows for OpenAI and Grok; revocation | `oauth_loopback.go`, `oauth_device.go`, `oauth_revoke.go` |
| Discovery | Live model listing with pagination, static catalogs, labels, fuzzy search (`SearchModels`), use-case profiles and ranking, external model-metadata client, Ollama URL validation, Kilo capabilities | `discovery.go`, `models_catalog.go`, `model_matcher.go`, `model_ranking.go`, `model_metadata.go` |
| Descriptors | `ProviderDescriptor` (env var, auth methods, API-key requirement, base URL) driving any UI | `descriptor.go` |
| Configuration flow | `wizard.ConfigureLLM`: choose provider, credential, model, fallbacks; never writes config, never logs a key | `wizard/` |

The package's own doc comment already describes it as "a shared, SDK-free
LLM provider abstraction … All providers use raw net/http". (R,
`llmprovider/provider.go:1-3`)

## Dependency assessment

### Today

| Package | Non-std imports (transitive) |
|---|---|
| `mcplib/llmprovider` | `mcplib/logging` only (M) |
| `mcplib/logging` | none (M) |
| `mcplib/wizard` | `mcplib` root → MCP go-sdk, `golang.org/x/oauth2`, `google/jsonschema-go`, `segmentio/asm`, `segmentio/encoding`, `yosida95/uritemplate`; plus `golang.org/x/term`, `golang.org/x/sys` (M) |

`prepare-commit-msg` is not an MCP server, yet its `go.mod` carries the MCP
go-sdk, `x/oauth2`, `jsonschema-go`, both `segmentio` modules and
`uritemplate` as indirect requirements. It imports `mcplib/llmprovider`,
`mcplib/wizard` and `mcplib/selfupdate`; of those only `wizard` reaches the
go-sdk, through the `mcplib` root import. (M)

### After extraction

The scratch module's `go mod tidy` produced (S):

```text
module github.com/maccavelli/go-llmprovider-sdk

go 1.26.6

require golang.org/x/term v0.46.0

require golang.org/x/sys v0.48.0 // indirect
```

`golang.org/x/term` is needed only by `wizard/text_prompter.go`
(`term.IsTerminal`, `term.MakeRaw`, `term.Restore` for hidden key entry).
A consumer that imports only `llmprovider` compiles no non-std code. Test
code uses only the standard library (`net/http/httptest`, `testing`). (M)

### Further reduction examined

- **Vendor SDKs** (`openai-go`, `anthropic-sdk-go`, `google.golang.org/genai`):
  not used today, and adopting them would add large dependency trees and
  lose the raw-HTTP control the wire-fidelity work depends on. Nothing to
  remove. (R)
- **`golang.org/x/oauth2`:** not used; OAuth refresh, PKCE, device code and
  revocation are hand-written on `net/http`. Nothing to remove. (R)
- **`golang.org/x/term`:** removable only by re-implementing raw terminal
  mode per platform, or by dropping `TextPrompter` and leaving every
  consumer to supply a `Prompter`. Both trade a Go-team module for either
  platform code or a lost feature. (R)

The floor for the extracted module is therefore **zero** third-party
modules for `llmprovider`, and **one** Go-team module (`x/term`, with
`x/sys`) if `wizard` travels with it.

## Findings

### F1 — Redaction is borrowed from `mcplib/logging` (M, R)

`llmprovider` calls `logging.RedactString` at `api_error.go:120`,
`api_error.go:153` and `oauth_session.go:78`. `wizard` calls
`logging.MaskSecret` at `auth.go:142`, `auth.go:173`, `configure.go:246`
and `configure.go:258`. Both functions live in 125 lines with no imports
beyond the standard library.

`logging/redact.go` calls itself "the single source of truth for secret
redaction in mcplib", shared with `LogBuffer`, `SanitizingWriter` and the MCP
log-notification path, which stay in `mcplib`.

**Bears on:** whether the SDK carries its own copy (for example
`internal/redact`, as the scratch experiment did) and accepts two pattern
sets that can drift, or exports a redaction package that `mcplib/logging`
then delegates to (making `mcplib` depend on the SDK, which it will anyway
if any `mcplib` package keeps using `llmprovider`).

### F2 — `wizard` reaches the MCP go-sdk through one environment check (M, R)

`wizard/auth.go:51` returns `mcplib.IsOrchestratorOwned()`, which is
`os.Getenv("MCP_ORCHESTRATOR_OWNED") == "true"` (`mcplib/orchestrator.go:19-21`).
`Options.Orchestrated *bool` already overrides it when non-nil
(`wizard/configure.go:78-79`). The sentinel `ErrOrchestrated` says
"orchestrated process uses the LLM backplane", an MCP-fleet concept
(`wizard/auth.go:31-32`).

The backplane client itself (`mcplib/backplane.go`) does not import
`llmprovider` and has no reason to move. (M)

**Bears on:** whether `wizard` moves at all, and if it does, how the
orchestrated default is supplied. The scratch experiment replaced the call
with `return false`, which compiled and passed every existing test; that
changes behaviour for any consumer that relies on auto-detection without
setting `Options.Orchestrated`.

### F3 — The library identifies itself as `mcplib` on the wire (R)

Client identification was a deliberate decision (`mcplib`
`0012-MADR-conform-providers-to-reference-clients.md` §1.4, cited in
`identification.go:12`). Every one of these is `mcplib`-specific:

| Where | Value | Effect |
|---|---|---|
| `identification.go:15` | `defaultClientName = "mcplib"` | leads `User-Agent`; Kilo editor name |
| `identification.go:16`, `:43-60` | `mcplibModulePath` used to read the library version from build info | after the move this never matches, so the version reads `(devel)` |
| `identification.go:91` | `User-Agent` trailer `mcplib/<version>` | |
| `discovery.go:189-190` | ChatGPT `client_version` derived from the `mcplib` version | see F4 |
| `openai_chatgpt.go:14-15`, used at `openai.go:195`, `discovery.go:199`, `oauth_loopback.go:379` | `originator: mcplib` | sent on ChatGPT generation and listing, and on the OpenAI OAuth authorize URL; gate G-C of `mcplib` `0012-MADR-conform-providers-to-reference-clients.md` measured it "accepted; also accepted without the header" (line 889) for ChatGPT only |
| `oauth_loopback.go:387`, `oauth_device.go:157` | Grok OAuth `referrer=mcplib` | sent to xAI's authorization server |
| `model_metadata.go:21-22` | `MCPLIB_MODELS_METADATA_URL`, `MCPLIB_DISABLE_MODELS_METADATA` | operator-facing environment variables |
| `provider.go:1-3` | package doc names magictools and magicdev as the users | |

The scratch experiment's import-rewrite check flagged `identification.go`
as still containing the `mcplib` module path string, confirming this is not
caught by a build or by the existing tests: with the path unchanged, tests
pass and the version silently becomes `(devel)`. (S)

**Bears on:** the SDK's client name, its version source, whether the Grok
`referrer` value can change without a server-side effect, and whether the
environment variables are renamed with or without a compatibility read of
the old names.

### F4 — The SDK's version number is observable to the ChatGPT backend (R)

`discovery.go:15-40` sends `client_version=X.Y.Z` from the library's own
release version, and records a measurement from 2026-09-27: `0.0.0` "is
accepted but hides models whose minimal_client_version is higher
(gpt-6-sol and gpt-6-luna)". `mcplib` currently sends `1.6.0`.

The same record's gate G-C measured the threshold: `gpt-6-sol` and
`gpt-6-luna` carry `minimal_client_version` `0.155.0`
(`mcplib` `0012-MADR-conform-providers-to-reference-clients.md` lines
904-906). A new module tagged `v0.1.0` would send `0.1.0`, below that
threshold, and so would hide both models; any `v1.x` tag sends a value above
it. The current thresholds of other models were not measured.

**Bears on:** the SDK's first tag (`v0.x` or `v1.0.0`), or whether
`client_version` should be decoupled from the module version. A live
listing (`live_chatgpt_listing_test.go`) at the candidate version confirms
it.

### F5 — 261 comment citations point at `mcplib` records, some ambiguously (M)

`llmprovider/` and `wizard/` cite `mcplib` records 261 times across 90
files: MADR 0001 (1), 0004 (2), 0009 (36), 0010 (53), 0012 (108), 0013
(44), 0014 (17). Many tests are named for the section they pin
("pins MADR 0010 §2").

`mcplib` has two different records numbered 0009
(`docs/0009-MADR-live-catalog-model-search.md` and
`docs/decisions/0009-MADR-repair-oauth-loopback-and-session-wiring.md`) and
two numbered 0010 (`docs/0010-MADR-use-case-aware-default-model-ranking.md`
and `docs/decisions/0010-MADR-windows-stdio-oauth-tokenstore-ci.md`). A bare
"MADR 0009" in moved code cannot be resolved by number alone.

A relative link cannot reach another repository, so after the move each
citation needs the repository named ("mcplib 0012-MADR-… §1.4") or the
record carried into this repository under a new number.

**Bears on:** whether the rationale records migrate with the code, and the
effort of a per-citation (not pattern-based) rewrite for the 0009 and 0010
references.

### F6 — Three `mcplib` plans touching this code are open (M)

| `mcplib` record | Status | Touches |
|---|---|---|
| `docs/0010-PLAN-use-case-aware-default-model-ranking.md` | in-progress | `llmprovider/` ranking, metadata, catalogs; `wizard/configure.go` |
| `docs/decisions/0009-PLAN-repair-oauth-loopback-and-session-wiring.md` | in-progress | OAuth loopback, `wizard` session wiring |
| `docs/decisions/0010-PLAN-windows-stdio-oauth-tokenstore-ci.md` | proposed | `FileTokenStore`, CI parity |

Extracting while these are open means either finishing them in `mcplib`
first, transferring them to this repository, or porting their later commits
across. (M, R)

**Bears on:** sequencing: a freeze point for `llmprovider/` and `wizard/` in
`mcplib`.

### F7 — Three consumers, on three different `mcplib` versions (M)

| Consumer | `mcplib` pin | Packages used | Heaviest symbols |
|---|---|---|---|
| `mcp-server-magicdev` | v1.2.0 | root, `llmprovider`, `logging`, `wizard` | `Tool`, `NewProvider`, `Descriptors`, `ConfigureLLM`, `Choice` |
| `mcp-server-magictools` | v1.4.1 | root, `llmprovider`, `logging`, `selfupdate`, `wizard` | `NewProvider`, `DescriptorFor`, `Descriptors`, `NewClaude`, `GenerateWithRetry`, `ConfigureLLM` |
| `prepare-commit-msg` | v1.6.0 | `llmprovider`, `selfupdate`, `wizard` | `ProviderOpenAI`, `OAuthSession`, `TokenSource`, `NewFileTokenStore`, `VendorCLISession`, `ConfigureLLM` |

No other repository under the same parent imports `llmprovider` or
`wizard`. (M)

`mcp-server-magicdev` is four minor releases behind; moving it to the SDK
also moves it across every `llmprovider` API change since `v1.2.0`. The two
MCP servers keep `mcplib` for their MCP needs; only `prepare-commit-msg`
could stop reaching the go-sdk (it still needs `mcplib/selfupdate`, whose
own closure is `x/mod`, `x/sys`, `x/term`). (M)

**Bears on:** whether `mcplib` keeps forwarding packages
(`mcplib/llmprovider` and `mcplib/wizard` as type aliases and re-exported
functions and sentinels, marked `Deprecated`) so consumers migrate on their
own schedule, or removes them. Removing a package from a `v1` module breaks
any consumer that upgrades; semantic versioning puts that in `mcplib` v2.

Forwarding is mechanically possible for this API: types forward as
aliases, functions as package-level variables, and re-exporting the
sentinel values (not re-declaring them) keeps `errors.Is` identity across
both import paths. (R) Not built or tested.

### F8 — Platform-specific code needs a three-OS CI matrix (M)

`tokenstore_file_unix.go` (`//go:build unix`) and
`tokenstore_file_windows.go` (`//go:build windows`) split the token store.
`mcplib` CI runs `go test` on `ubuntu-24.04`, `macos-15` and
`windows-2025`, and vet / `gofmt` / `go mod tidy -diff` / `golangci-lint
v2.13.1` on Linux. The SDK needs the same, minus the self-update release
fixture and release-guard scripts, which are not `llmprovider` concerns.
This repository has no CI, `Makefile`, `.golangci.yml` or `AGENTS.md` yet.

### F9 — Live tests travel, and still need credentials (M)

23 test files behind `//go:build live_gateways` exercise real gateways,
OAuth login, revocation and vendor CLI sessions. They are excluded from the
default build and from CI. They are the only evidence that the moved code
still speaks correctly to the vendors, and the natural check for F3 and F4.

## Scratch extraction experiment

Performed in the session scratch directory; neither `mcplib` nor this
repository was modified. The script copied `llmprovider/` and `wizard/`
unchanged, copied `logging/redact.go`, `logging/mask.go` and their tests
into `internal/redact` (package clause renamed), wrote a `go.mod` for
`github.com/maccavelli/go-llmprovider-sdk` at `go 1.26.6`, rewrote the two
import paths and the `logging.RedactString` / `logging.MaskSecret`
selectors, and replaced the `mcplib.IsOrchestratorOwned()` call with
`false`.

| Step | Result |
|---|---|
| import rewrite | 14 files rewritten; a residual `mcplib` scan flagged `llmprovider/identification.go` (the module-path string of F3, not an import) |
| `go mod tidy` | exit 0; requirements `golang.org/x/term v0.46.0`, indirect `golang.org/x/sys v0.48.0` |
| `go vet ./...` (darwin; `GOOS=windows`; `CGO_ENABLED=0 GOOS=linux`) | exit 0 on all three |
| `gofmt -l .` | first run listed `wizard/auth.go` and `wizard/configure.go`: the rewritten import sorts differently inside its group; `gofmt -w` fixed both |
| `golangci-lint` v2.13.2 with `mcplib`'s `.golangci.yml` | first run: 2 `gofmt` issues (above); after `gofmt -w`: `0 issues.` |
| `go test -count=1 -cover ./...` | exit 0: `internal/redact` 100.0 %, `llmprovider` 88.1 %, `wizard` 82.6 % |
| `go list -deps` non-std | only the SDK's own packages, `golang.org/x/term`, `golang.org/x/sys/unix` |

The experiment shows the code moves; it does not show the move is
complete. It left F3's identity strings unchanged, did not port history,
did not build any consumer against the new path, and made no live request.

## Decisions a MADR here needs to make

Each is open. The evidence column says which finding bears on it.

| # | Decision | Options seen | Evidence |
|---|---|---|---|
| D1 | Does `wizard` move with `llmprovider`? | move both; move `llmprovider` only and leave `wizard` in `mcplib` importing the SDK | F2, F7: all three consumers use both; leaving `wizard` in `mcplib` keeps `prepare-commit-msg` on the go-sdk |
| D2 | How is "orchestrated" decided outside `mcplib`? | caller-supplied only (`Options.Orchestrated`), with an `mcplib` forwarding wrapper supplying `IsOrchestratorOwned()`; keep reading `MCP_ORCHESTRATOR_OWNED` inside the SDK | F2 |
| D3 | Redaction ownership | SDK-internal copy; SDK exports it and `mcplib/logging` delegates | F1 |
| D4 | Module and package layout | `llmprovider/` and `wizard/` subdirectories (package names unchanged, migration is an import-path rewrite); package at module root | S: the subdirectory layout needed no identifier changes |
| D5 | Client identity and environment names | new client name, UA trailer, Grok `referrer`, `MCPLIB_*` variables renamed with or without fallback | F3, F9 |
| D6 | First version and `client_version` source | `v0.x`; `v1.0.0`; decouple `client_version` from the module version | F4, F9 |
| D7 | Fate of `mcplib/llmprovider` and `mcplib/wizard` | forwarding shims marked `Deprecated`, removed in `mcplib` v2; remove in a v1 minor | F7 |
| D8 | History and rationale | import history with `git filter-repo` (available on this host); copy only the tree; carry `mcplib` records into this repository or cite them by repository | F5 |
| D9 | Freeze point | finish, transfer or port the open `mcplib` plans | F6 |
| D10 | Go floor | keep 1.26.6 (`mcplib` `0006-MADR-raise-go-toolchain-floor-to-1-26-6.md`); lower it for wider reach | not examined; the code uses `crypto/rand.Text` (Go 1.24+), so any lower floor needs its own evidence |

## Shape of the work

For orientation only; the PLAN owns the steps.

1. Freeze `llmprovider/` and `wizard/` in `mcplib` (D9).
2. Seed this repository: history import or copy (D8), module path, layout
   (D4), redaction (D3), orchestrated default (D2), identity (D5), repository
   scaffolding (`AGENTS.md`, `Makefile`, `.golangci.yml`, CI per F8).
3. Run the live suite at the candidate version (F4, F9); tag (D6).
4. In `mcplib`: replace the packages with forwarding shims or remove them
   (D7); release.
5. Migrate `prepare-commit-msg`, `mcp-server-magictools`,
   `mcp-server-magicdev` (the last across its `v1.2.0` API gap).
6. Rewrite or re-home the record citations (F5).

## Not verified

- No live request: F3 (Grok `referrer`) and F4 (`client_version`) effects
  are unmeasured.
- No consumer was built against the new import path.
- The forwarding-shim approach (F7) was reasoned, not compiled.
- History import with `git filter-repo` was not attempted.
- Whether `golang.org/x/term` v0.46.0 / `x/sys` v0.48.0 (chosen by the
  scratch `tidy`) versus `mcplib`'s pinned v0.43.0 / v0.47.0 matters to any
  consumer was not examined.
- The duplicate 0009 and 0010 numbers in `mcplib` were observed, not
  investigated; they are that repository's concern except where F5 needs
  them resolved.
