# Architecture

How `go-llmprovider-sdk` is put together, as it is now. This file carries no
history and no rationale: the records under [decisions/](decisions/) hold the
argument, and [README.md](README.md) indexes them.

## What it is

The Go module `github.com/maccavelli/go-llmprovider-sdk`: a library for calling
LLM providers and authenticating to them. It has no binary. It requires Go
1.27.1, `golang.org/x/term` and, indirectly, `golang.org/x/sys`.

The code came from `mcplib` `v1.6.0` with its history. Its exported API is still
`mcplib`'s, apart from the removed orchestration option; the v1 API is decided by
[0015-MADR](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md).

## Tree

```text
README.md                   repository entry; links here
AGENTS.md                   rules for agents: records, checks, commits
go.mod, go.sum              the module and its two requirements
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration
.markdownlint-cli2.jsonc    Markdown lint configuration
.github/workflows/ci.yml    CI
scripts/go-precheck.sh      the pre-add check
scripts/check_parity_map.py G-parity: the mcplib migration map is complete
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
llmprovider/                providers, credentials, discovery
wizard/                     interactive provider configuration
internal/redact/            secret redaction and masking
internal/wiretest/          G-wire's request recorder, for tests only
docs/
  README.md                 record index, "I want to…", migration table
  architecture.md           this file
  decisions/                MADR and PLAN records
  reports/                  numbered observations
  guides/                   API standards; migrating from mcplib
```

## Packages

| Package | Holds | Depends on |
| :--- | :--- | :--- |
| `llmprovider` | provider adapters, request and response types, credentials, OAuth, token storage, model discovery and ranking | the standard library, `internal/redact` |
| `wizard` | the configuration flow and its `Prompter` seam | `llmprovider`, `internal/redact`, `golang.org/x/term` |
| `internal/redact` | `Redact` and `String` (hide a secret completely) and `MaskSecret` (show a suffix for identification) | the standard library |
| `internal/wiretest` | an `httptest` server that records requests as normalised JSON, and golden-file comparison; imported only by tests | the standard library |

## Providers and items

- **Ten provider ids,** constants in `llmprovider`: `openai` (API key, or a
  ChatGPT subscription through the Codex backend), `claude`, `gemini`, `grok`,
  `opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `together` and
  `ollama`.
- **Five wire formats:** OpenAI Responses (`openai`, `grok`), Anthropic Messages
  (`claude`), Gemini Interactions (`gemini`), Chat Completions (`huggingface`,
  `kilo`, `together`, `ollama`), and, per model, all of those plus Gemini
  `generateContent` for the two OpenCode gateways.
- **`Provider`** is `Name()` and `Generate(ctx, prompt)`. Further abilities are
  optional interfaces a caller checks for: `ThinkingProvider`, `ToolProvider`,
  `ThinkingToolProvider`, the four `Item*Provider` variants, `Continuer`, and
  `ModelDiscoverer`.
- **`Item`** is sealed. Its types are `MessageItem`, `FunctionCallItem`,
  `FunctionCallOutputItem` and `ReasoningItem`; item methods return a
  `*Response`.
- **Construction:** `NewProvider(name, apiKey, model, opts...)`, or a
  provider's own `New…`; options are `ProviderOption` functions (`WithBaseURL`,
  `WithHTTPClient`, `WithReasoningEffort`, …). `GenerateWithRetry` and its two
  siblings retry on typed errors.
- **Transport:** without `WithHTTPClient`, each provider builds one client:
  330 s overall, 300 s to the first byte, and `HTTP_PROXY`, `HTTPS_PROXY`
  and `NO_PROXY` honoured. It carries the provider's requests, its listing
  and probes, and its OAuth session's refreshes when the session has no
  client of its own.
- **Errors:** `*APIError` with sentinels such as `ErrRateLimited`,
  `ErrQuotaExhausted` and `ErrAuthFailure`, plus `RateLimitError` and
  `IncompleteError`. Error bodies pass through `redact.String`.

## Credentials

- **`TokenSource`** returns a `Token` for each request. `StaticToken` wraps an
  API key. `NewProviderWithSource` accepts a source for `openai` and `grok`.
- **`OAuthSession`** is a refreshable `TokenSource` for ChatGPT and Grok.
  - **Creating one:**
    - `LoginBrowserOAuth` uses PKCE on a loopback redirect.
    - `StartDeviceOAuth` returns a `DeviceLogin` handle: the user code, the
      verification URI and the expiry, then `Wait` and `Cancel`.
    - `LoginDeviceOAuth` is the same flow in one call.
  - **Checked before use.** Every login verifies its `id_token` before any
    claim is used:
    - the signature against the issuer's published keys, fetched from
      discovery's `jwks_uri`, cached, and refetched once for an unknown `kid`;
    - the issuer, the audience and the expiry, and the Grok nonce.

    A missing or failing `id_token` fails the login. A caller's issuer whose
    discovery fails is an error; only the built-in issuers fall back to
    built-in endpoints and keys.
  - **Refresh.** The session refreshes before expiry and once after a 401. A
    rotation is kept even when saving it fails: the failure goes to the
    session's `Logger`, and the save is retried. After a rejected refresh the
    session re-reads the store, and adopts a rotation another process saved.
  - `RevokeOAuthSession` ends it.
- **`TokenStore`** persists sessions. `FileTokenStore` keeps one `0600` JSON
  file per provider:
  - it writes through a temp file, with `fsync`, rename and a directory
    `fsync`;
  - it refuses a file over 64 KiB;
  - as a `RefreshLocker`, it holds a lock file across processes for each
    refresh, so a refresh token is never spent twice.
- **Formatting never shows a secret.** `Token`, `StaticToken` and
  `*OAuthSession` print `[redacted]` for every secret, under `fmt` and `slog`.
- **`VendorCLISession`** reads the Codex or Grok CLI's own login on every
  request and never refreshes it.
- **`ProviderDescriptor`** lists each provider's `AuthMethod`s; `Descriptors()`
  is the canonical menu.

## Discovery and ranking

- `ListAvailableModels` and `ListModelCatalog` (and their `…WithSource` forms)
  list a provider's models within a 10 s bound; a `ModelCatalog` carries the
  recommended six, the full usable list and, on failure, `Err`.
- `SearchModels` matches a query against a list. The static catalogs
  (`StaticModels`) are the fallback.
- `ModelProfile` (`ProfileUtility`, `ProfileCapable`) ranks the open catalogs,
  using models.dev-format metadata from `https://models.opencode.ai/api.json`
  (`LLMPROVIDER_MODELS_METADATA_URL` overrides it;
  `LLMPROVIDER_DISABLE_MODELS_METADATA` turns it off).

## The wizard

`ConfigureLLM(ctx, p, opts)` walks provider → base URL → credential → model →
fallbacks and returns a `Result`; it writes no configuration. It renders
nothing itself: everything goes through the `Prompter` interface (`Select`,
`MultiSelect`, `Confirm`, `Input`, `Secret`, `Notify`). `TextPrompter`
implements it over a terminal with `golang.org/x/term`. OAuth and other
non-API-key methods are offered only when `Options.TokenStore` is set.

## Identity

Every request names this module and the calling application:
`User-Agent: <app>/<version> (<os>; <arch>) go-llmprovider-sdk/<version>`.
`WithClientInfo` names the application and `WithSessionID` the conversation.
The ChatGPT `originator` header and the Grok OAuth `referrer` are
`go-llmprovider-sdk`.

## The dependency rule

The standard library and `golang.org/x/term` only; any other module needs a
MADR (AGENTS.md). Nothing imports `mcplib` or the MCP go-sdk.

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
  `vuln`, `pre-add-check`, `parity-check`, `help`.
- **`scripts/check_parity_map.py`** (G-parity) fails when an identifier in
  `docs/guides/migrating-from-mcplib.ids`, the exported identifiers of
  `mcplib` `v1.6.0` `llmprovider` and `wizard`, has no row in
  `docs/guides/migrating-from-mcplib.md`, or a row names one that is not in
  the list. `make parity-check` runs it.
- **G-wire** is `TestWireGoldens` in `llmprovider`, part of `go test`. It
  drives 16 provider and gateway-route cases through seven scenarios (text,
  forced tool, thinking, thinking tool, items, continuation where the
  provider has `Continue`, listing): 100 golden files. It runs them against
  `internal/wiretest`, and compares each recording with
  `llmprovider/testdata/wire/<case>/<scenario>.json`. The files were
  recorded at the end of 0002-PLAN Phase 7, except `together`'s, added with
  the provider (0017-PLAN U1). `-update` rewrites them, and is used only for
  a difference a record explains.
- **`scripts/go-precheck.sh`** runs `gofmt` on the given Go files,
  `golangci-lint run -c .golangci.yml ./...`, `go vet` and `go test` on their
  packages, and `govulncheck ./...`. `make pre-add-check` runs it, and so does
  the machine-wide agent gate before an agent `git commit` that stages Go
  files.
- **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with
  the Go version read from `go.mod`: `go test`; on Linux also `go vet`,
  `gofmt`, `go mod tidy -diff`, `make lint` and `go vet -tags live_gateways`.

## What is not here

- **Code that meets [guides/api-standards.md](guides/api-standards.md).**
  The guide states the target; the code still has `mcplib`'s API until
  [0015-PLAN](decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md)
  lands.
- **`guides/adding-a-provider.md`:** written in 0015-PLAN Phase S11.
- **The v1 package layout** (`llmprovider/auth`, `llmprovider/catalog`,
  `llmprovider/providers/…`, `llmprovider/llmtest`):
  [0015-MADR](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md),
  accepted.
- **A licence file.**
