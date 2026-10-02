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
LICENSE                     Apache License 2.0
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
llmprovider/                the contract, errors, options and credential sources
llmprovider/auth/           OAuth sessions and logins, vendor CLI sessions, token stores
llmprovider/llmtest/        the conformance suite and a fake provider
llmprovider/providers/      the built-in providers, by id
llmprovider/providers/openai/  OpenAI: the Responses API, and the ChatGPT backend
llmprovider/providers/claude/  Claude: the Anthropic Messages API
llmprovider/providers/gemini/  Gemini: the Interactions API
llmprovider/providers/grok/    Grok: the xAI Responses API
llmprovider/providers/opencode/ OpenCode Zen and Go: four wire formats, routed per model
llmprovider/providers/kilo/    Kilo Gateway: Chat Completions
llmprovider/providers/huggingface/ Hugging Face Inference Providers: Chat Completions
llmprovider/providers/together/ Together AI: Chat Completions
llmprovider/providers/ollama/  a local Ollama: Chat Completions
llmprovider/internal/wire/     what the shared wire formats have in common
llmprovider/internal/wire/*/   one shared wire format each: responses, chatcompletions, messages, generatecontent
llmprovider/catalog/          model listing, static catalogs, ranking, metadata, search, profiles
llmprovider/internal/transport/ the default client, the client identity, Retry-After, the listing probe
llmprovider/internal/kiloendpoint/ Kilo's endpoints, derived from a credential
llmprovider/internal/wirecase/ G-wire's scenarios through the new API, for tests only
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
| `llmprovider` | the contract: request and response types, errors, options, the `Registry`, retry middleware, and the credential sources (`Token`, `TokenSource`, `StaticToken`, `CommandToken`) | the standard library, `internal/redact`, `internal/transport` |
| `llmprovider/auth` | OAuth sessions and their refresh and revocation, the browser, device and Kilo device logins, `id_token` checks, `VendorCLISession`, `TokenStore` and `FileTokenStore`, and the issuers and client ids | `llmprovider`, `internal/transport`, `internal/redact`, `internal/kiloendpoint` |
| `wizard` | the configuration flow and its `Prompter` seam | `llmprovider`, `auth`, `catalog`, `providers`, `internal/redact`, `golang.org/x/term` |
| `internal/redact` | `Redact` and `String` (hide a secret completely) and `MaskSecret` (show a suffix for identification) | the standard library |
| `internal/wiretest` | an `httptest` server that records requests as normalised JSON, and golden-file comparison; imported only by tests | the standard library |
| `llmprovider/llmtest` | `Run`, the conformance suite, and `Fake`, a scriptable provider | `llmprovider`, the standard library |
| `llmprovider/providers` | `Default()`, a new `Registry` of the built-in providers, and `New(id, opts...)`; it holds all ten provider ids | `llmprovider` and the provider packages |
| `llmprovider/providers/openai` | OpenAI through the new contract: `New`, `WithStore`, and `ListModels` | `llmprovider`, `auth`, `internal/wire/responses` |
| `llmprovider/providers/claude` | Claude through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `internal/wire`, `internal/wire/messages` |
| `llmprovider/providers/gemini` | Gemini through the new contract: `New`, `WithStore`, and `ListModels` | `llmprovider`, `auth`, `internal/wire` |
| `llmprovider/providers/grok` | Grok through the new contract: `New`, `WithStore`, and `ListModels` | `llmprovider`, `internal/wire/responses` |
| `llmprovider/providers/opencode` | OpenCode Zen and Go through the new contract: `NewZen`, `NewGo`, `WithRoute` and `ListModels`, with the route table | `llmprovider`, `auth`, `internal/wire` and its four format packages |
| `llmprovider/providers/kilo` | Kilo through the new contract: `New`, `WithOrganization`, `WithCapabilities`, `WithDataCollection` and `ListModels` | `llmprovider`, `auth`, `internal/wire/chatcompletions` |
| `llmprovider/providers/huggingface` | Hugging Face through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `internal/wire/chatcompletions` |
| `llmprovider/providers/together` | Together AI through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `internal/wire/chatcompletions` |
| `llmprovider/providers/ollama` | Ollama through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `internal/wire/chatcompletions` |
| `llmprovider/internal/wire` | the JSON keys, `ToolArguments` and `SystemPrompt` that the shared formats use | `llmprovider` |
| `llmprovider/internal/wire/responses` | the OpenAI Responses wire: `Input`, `Decode`, `ReadStream` | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/chatcompletions` | the Chat Completions wire: `Opts`, `Body`, `Decode` | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/messages` | the Anthropic Messages wire, with its thinking shape: `FromItems`, `Decode`, `AddThinking` | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/generatecontent` | Gemini's generateContent wire, with its thinking shape: `SystemInstruction`, `Contents`, `Decode`, `ThinkingConfig` | `llmprovider`, `internal/wire` |
| `llmprovider/catalog` | `List`, `Catalog`, `Static`, `Rank`, `Search`, `Match`, `Label`, `Profile`, `Metadata`, `LookupMetadata`, `KiloModelCapabilities`, `ValidateOllamaURL`, and the options `WithProfile` and `WithKiloOrganization` | `llmprovider`, `internal/transport`, `internal/kiloendpoint` |
| `llmprovider/internal/kiloendpoint` | `Resolve`, `Route` and Kilo's base URL, for `catalog`, `providers/kilo` and the Kilo device login | the standard library |
| `llmprovider/internal/transport` | `DefaultClient`, `Identity` and its User-Agent, `BuildVersions`, `ParseRetryAfter`, `RetryAfter`, `ProbeGenerateHealth` | the standard library |
| `llmprovider/internal/wirecase` | G-wire's scenarios and canned replies through the new API, shared by the provider packages' tests | `llmprovider`, `internal/wiretest` |

## The contract

`llmprovider` holds the generation contract (0015-MADR D3–D10):

- `Provider` is `ID`, `Capabilities` and `Generate(ctx, *Request)
  (*Response, error)`. `GenerateText` and `GenerateToolCall` are functions
  over any `Provider`.
- `Capabilities` gives each of `Tools`, `ForcedToolChoice`, `Reasoning`,
  `Continuation` and `NativeStreaming` as `Unsupported`, `BestEffort` or
  `Supported`. `Capabilities.Check(req)` validates a request and refuses an
  unsupported need before any network call, with `ErrUnsupported`.
- `Stream(ctx, p, req)` streams from every provider: a provider's own
  `Streamer`, or `Generate`'s result as events.
- `Response.Usage` is decoded from every wire format that reports it. A
  total holds its part: `InputTokens` includes `CachedTokens`, and
  `OutputTokens` includes `ReasoningTokens`. Anthropic's cache reads and
  writes are added to the input, and Gemini's thoughts to the output.
- `Option` configures construction. A provider package's `New` resolves its
  options with `ResolveOptions(id, opts)`, which refuses an option scoped to
  another provider (`ScopedOption`). `For(id, opts...)` scopes options to
  one provider: they apply after the rest when building `id`, and are
  skipped for any other, so one list can build every provider. The resolved
  `Settings` are read-only.
- `APIError` carries a `Kind`, one of the sentinels. `ErrContextOverflow`
  sits beneath `ErrInvalidRequest`; it is classified from the service's
  error type, or from a message table taken from pi's `overflow.ts`
  (`context_overflow.go`).
- `WithRetry(p, RetryPolicy{…})` retries what `Retryable` allows, waiting as
  long as the service asks, up to a cap.
- `Registry` holds `Descriptor` and `Factory` pairs and refuses a duplicate
  id. There is no global registry.
- `llmprovider/llmtest` has `Run`, the conformance suite each provider will
  pass, and `Fake`, a scriptable provider for tests.

## Providers and items

- **Ten provider ids,** `ProviderID` constants in `llmprovider`: `openai`
  (API key, or a ChatGPT subscription through the Codex backend), `claude`,
  `gemini`, `grok`, `opencode-zen`, `opencode-go`, `huggingface`, `kilo`,
  `together` and `ollama`. A label that may carry a route, such as
  `APIError.Provider`, stays a `string`.
- **Five wire formats:** OpenAI Responses (`openai`, `grok`), Anthropic Messages
  (`claude`), Gemini Interactions (`gemini`), Chat Completions (`huggingface`,
  `kilo`, `together`, `ollama`), and, per model, all of those plus Gemini
  `generateContent` for the two OpenCode gateways.
- **`Provider`** is the contract above. Tools, reasoning and continuation are
  fields of `Request`, and `Capabilities` says which a provider supports.
  Listing (`ModelLister`) and native streaming (`Streamer`) are the only
  optional interfaces.
- **`Item`** is sealed. Its types are `MessageItem`, `FunctionCallItem`,
  `FunctionCallOutputItem` and `ReasoningItem`; item methods return a
  `*Response`.
- **Construction:** `providers.New(id, opts...)`, or the provider package's
  own `New` (`opencode.NewZen` and `NewGo`); every provider is in its own
  package. Options are `Option` values: the common ones in `llmprovider`
  (`WithBaseURL`, `WithHTTPClient`, `WithReasoning`, …), and a provider
  package's own, scoped to it. `WithRetry` retries by the error's kind.
- **Transport:** without `WithHTTPClient`, each provider builds one client,
  `internal/transport`'s `DefaultClient`:
  330 s overall, 300 s to the first byte, and `HTTP_PROXY`, `HTTPS_PROXY`
  and `NO_PROXY` honoured. It carries the provider's requests, its listing
  and probes, and its OAuth session's refreshes when the session has no
  client of its own.
- **Errors:** one structured error, `*APIError`, whose `Kind` is a sentinel
  such as `ErrRateLimited`, `ErrQuotaExhausted`, `ErrAuthFailure` or
  `ErrIncomplete`; a rate limit carries its `RetryAfter`, and a cut-short
  response its `Reason`. `Retryable()` says whether to try again. Every
  sentinel's message starts `llmprovider:`. Error bodies pass through
  `redact.String`.

## Credentials

- **`TokenSource`** returns a `Token` for each request. `StaticToken` wraps an
  API key. `openai.New` takes a source with `WithTokenSource`; a ChatGPT
  session selects the ChatGPT backend. `grok.New` takes any source: a key,
  an xAI OAuth session or the Grok CLI's login.
- **Where a token goes.** A token with no `Header` goes in the service's own
  header: `Authorization: Bearer` for OpenAI, Grok, Kilo, Hugging Face and Together, `x-api-key` for
  Claude, `x-goog-api-key` for Gemini, and on OpenCode the header of the
  request's route. Ollama has no header of its own, so it sends only a
  token that names one. One
  naming a `Header` goes there instead, prefixed `Bearer` only for a
  `TokenBearer`. Key sources (`StaticToken`, `CommandToken`) are
  `TokenAPIKey`; sessions are `TokenBearer` and name no `Header`. Every
  provider applies this to generation and listing through `Token.Apply`.
- **The shared wire formats** are in `internal/wire`, one package per format
  that more than one provider speaks; the Interactions wire is Gemini's
  alone. A Responses stream's failure is classified by
  `ClassifyStreamFailure`, beside `ClassifyHTTPError`: both are part of
  `llmprovider`'s error model.
- **The ChatGPT session** is `openai`'s concern. It lists the Codex
  catalog, and sets the originator, account and FedRAMP headers, reading
  `Account()` on an `OAuthSession` or `VendorCLISession`. `catalog` lists
  only the other providers' catalogs.
- **After a 401** a provider invalidates any `InvalidatingSource` and retries
  once: a `CommandToken` reruns its command, and an `OAuthSession` refreshes.
- **Sessions are `auth`'s.** A provider gives a session with no HTTP client
  its own through `OAuthSession.UseHTTPClient`, so refreshes share the
  provider's transport.
- **`OAuthSession`** (`auth`) is a refreshable `TokenSource` for ChatGPT and
  Grok.
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
- **Kilo device login.** `StartDeviceOAuth(ctx, "kilo", …)` runs Kilo's
  device flow on the same handle. The approved token never refreshes. It is
  stored as a session with no refresh token and no expiry, and the session
  is the Kilo credential. `KiloProfile` lists the account's organizations.
- **`CommandToken`** takes the token from a command the caller names, like
  Claude Code's `apiKeyHelper`:
  - the command runs without a shell, with a 10 s timeout;
  - its output is trimmed, capped at 8 KiB and cached for 5 minutes;
  - concurrent callers share one run;
  - a failure names the command, never its output.

  It is an `InvalidatingSource`: a provider that gets a 401 invalidates the
  source and retries once.
- **`VendorCLISession`** reads the Codex or Grok CLI's own login on every
  request and never refreshes it.
- **A `Descriptor`** lists a provider's `AuthMethod`s, among what a menu
  shows. Each provider package declares its own (`Descriptor()`, or
  `DescriptorZen()` and `DescriptorGo()`), and `providers.Default()`
  registers them in menu order. `wizard` builds its menu from
  `Options.Registry`, `providers.Default()` when nil.

## Discovery and ranking

- `catalog.List(ctx, id, src, opts...)` lists a provider's models within a
  10 s bound; a `Catalog` carries the recommended six, the full usable list
  and, on failure, `Err`. Its options are `llmprovider`'s common ones, with
  `catalog.WithProfile` and `catalog.WithKiloOrganization`.
- `catalog.Search` matches a query against a list. The static catalogs are
  the fallback. `catalog.Static(provider)` returns a copy, and
  `llmprovider.ProviderEnvVars()` a copy of the variable names.
  `catalog.Rank(provider, model)` scores a model by the provider's own
  ranking.
- A provider's `ListModels`, through `llmprovider.ModelLister`, returns the
  listing. By default, OpenAI (API key), Claude, Gemini, Grok and Ollama also
  send one billed generation to each candidate, up to `catalog.MaxListed`,
  and keep those that answer.
  `WithModelProbes(false)` turns that off. `ModelProbesFromEnv()` reads
  `LLMPROVIDER_PROBES` (`true` or `false`) for a caller who passes it; the
  package reads the variable nowhere else.
- `catalog.Profile` (`ProfileUtility`, `ProfileCapable`), passed with
  `catalog.WithProfile`, ranks the open catalogs,
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

- **One copy of a session.** A sign-in saves its session to
  `Options.TokenStore` and returns a `CredOAuth` `Result` with no token; the
  consumer loads the session from the store. Keeping an existing session
  reads it there too (0016-MADR D11, A7, A10).
- **Kilo.** A Kilo device login saves its token the same way, offers the
  account's organizations, and puts the choice in `Result.Organization`.
- **`Logout(ctx, p, opts, id)`** revokes the stored session (OpenAI and
  Grok), reports a failure, and deletes it.
- **`Result`** prints `[redacted]` for its API key under `fmt` and `slog`.
  Its JSON keeps the key, for the consumer to persist (0016-MADR A9).

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
- **G-wire** is `TestWireGoldens` in each provider package, part of
  `go test`. Through `llmprovider/internal/wirecase` it drives 16 provider
  and gateway-route cases through seven scenarios (text, forced tool,
  thinking, thinking tool, items, continuation where the provider continues,
  listing): 100 golden files. It runs them against `internal/wiretest`, and
  compares each recording with
  `llmprovider/providers/<id>/testdata/wire/<case>/<scenario>.json`. The files were
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
