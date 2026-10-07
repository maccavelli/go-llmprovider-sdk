# Architecture

How `go-llmprovider-sdk` is put together, as it is now. This file carries no
history and no rationale: the records under [decisions/](decisions/) hold the
argument, and [README.md](README.md) indexes them.

## What it is

The Go module `github.com/maccavelli/go-llmprovider-sdk`: a library for calling
LLM providers and authenticating to them. It has no binary. It requires Go
1.27.1, `golang.org/x/term` and, indirectly, `golang.org/x/sys`.

The code came from `mcplib` `v1.6.0` with its history. Its API is the v1 API
that [0015-MADR](decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
decides, and
[guides/migrating-from-mcplib.md](guides/migrating-from-mcplib.md) maps every
`mcplib` identifier to it.

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
scripts/check_deps.py       dep-check: only wizard leaves the standard library
scripts/check_coverage.py   coverage-check, with scripts/coverage-floors.txt
scripts/check_api.py        api-check: apidiff against the latest v1 tag
scripts/check_generated.py  generate-check: go:generate outputs are current
scripts/check_records.py    records-check: the decision records and their index
scripts/test_gates.py       gate-selftest: each gate fails on a planted breach
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
llmprovider/internal/ownerperm/ a directory and its files private to the current user
llmprovider/internal/filelock/ an exclusive OS lock on an open file, for the refresh lock
llmprovider/internal/wirecase/ G-wire's scenarios through the new API, for tests only
wizard/                     interactive provider configuration
internal/redact/            secret redaction and masking
internal/wiretest/          G-wire's request recorder, for tests only
internal/ambientcheck/      the no-ambient-state check, a test only
docs/
  README.md                 record index, "I want to…", migration table
  architecture.md           this file
  decisions/                MADR and PLAN records
  reports/                  numbered observations
  guides/                   API standards; adding a provider; migrating from mcplib
```

## Packages

The last column lists the module's own packages and `golang.org/x/term`; the
standard library is left out.

| Package | Holds | Depends on |
| :--- | :--- | :--- |
| `llmprovider` | the contract: request and response types, errors, options, the `Registry`, retry middleware, and the credential sources (`Token`, `TokenSource`, `StaticToken`, `CommandToken`) | `internal/transport`, `internal/redact` |
| `llmprovider/auth` | OAuth sessions and their refresh and revocation, the browser, device and Kilo device logins, `id_token` checks, `VendorCLISession`, `TokenStore` and `FileTokenStore`, and the issuers and client ids | `llmprovider`, `internal/transport`, `internal/kiloendpoint`, `internal/redact`, `internal/ownerperm`, `internal/filelock` |
| `wizard` | the configuration flow and its `Prompter` seam | `llmprovider`, `auth`, `catalog`, `providers`, `internal/redact`, `golang.org/x/term` |
| `internal/redact` | `Redact` and `String` (hide a secret completely), `MaskSecret` (show a suffix for identification), `StripControl` (remove terminal control characters) and `Field` (strip and bound a service's code or reason) | the standard library |
| `internal/wiretest` | an `httptest` server that records requests as normalised JSON, and golden-file comparison; imported only by tests | the standard library |
| `llmprovider/llmtest` | `Run`, the conformance suite, and `Fake`, a scriptable provider | `llmprovider` |
| `llmprovider/providers` | `Default()`, a new `Registry` of the built-in providers, and `New(id, opts...)`; it holds all ten provider ids | `llmprovider` and the provider packages |
| `llmprovider/providers/openai` | OpenAI through the new contract: `New`, `WithStore`, and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire/responses`, `internal/transport` |
| `llmprovider/providers/claude` | Claude through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire`, `internal/wire/messages`, `internal/transport` |
| `llmprovider/providers/gemini` | Gemini through the new contract: `New`, `WithStore`, and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire`, `internal/transport` |
| `llmprovider/providers/grok` | Grok through the new contract: `New`, `WithStore`, and `ListModels` | `llmprovider`, `catalog`, `internal/wire/responses`, `internal/transport` |
| `llmprovider/providers/opencode` | OpenCode Zen and Go through the new contract: `NewZen`, `NewGo`, `WithRoute` and `ListModels`, with the route table, built from the embedded `routes_snapshot.json` | `llmprovider`, `auth`, `catalog`, `internal/wire`, `internal/wire/responses`, `internal/wire/chatcompletions`, `internal/wire/messages`, `internal/wire/generatecontent` |
| `llmprovider/providers/kilo` | Kilo through the new contract: `New`, `WithOrganization`, `WithCapabilities`, `WithDataCollection` and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire/chatcompletions`, `internal/kiloendpoint` |
| `llmprovider/providers/huggingface` | Hugging Face through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire/chatcompletions` |
| `llmprovider/providers/together` | Together AI through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire/chatcompletions` |
| `llmprovider/providers/ollama` | Ollama through the new contract: `New` and `ListModels` | `llmprovider`, `auth`, `catalog`, `internal/wire/chatcompletions`, `internal/transport` |
| `llmprovider/internal/wire` | the JSON keys, `ToolArguments` and `SystemPrompt` that the shared formats use; `Post`, `Reauth` and `DecodeError`, the request path every provider sends through; the shared tool lists; `RemapCallIDs`, which fits another wire's call ids to a wire's rule | `llmprovider`, `internal/transport`, `internal/redact` |
| `llmprovider/internal/wire/responses` | the OpenAI Responses wire: `Input`, `Decode`, `DecodeFor`, `ReadStream` | `llmprovider`, `internal/wire`, `internal/transport`, `internal/redact` |
| `llmprovider/internal/wire/chatcompletions` | the Chat Completions wire: `Opts`, `Body`, `Decode`, `DecodeFor` | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/messages` | the Anthropic Messages wire, with its thinking shape: `FromItems`, `Decode`, `AddThinking` | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/generatecontent` | Gemini's generateContent wire, with its thinking shape: `SystemInstruction`, `Contents`, `Decode`, `ThinkingConfig` | `llmprovider`, `internal/wire` |
| `llmprovider/catalog` | `List`, `Catalog`, `Static`, `Rank`, `Search`, `Match`, `Label`, `Profile`, `Metadata`, `LookupMetadata`, `KiloModelCapabilities`, `ValidateOllamaURL`, and the options `WithProfile` and `WithKiloOrganization` | `llmprovider`, `internal/kiloendpoint` |
| `llmprovider/internal/kiloendpoint` | `Resolve`, `Route` and Kilo's base URL, for `catalog`, `providers/kilo` and the Kilo device login | the standard library |
| `llmprovider/internal/ownerperm` | `MkdirAll` and `File`, for `FileTokenStore`: modes 0700 and 0600 on Unix, where an existing directory must be the user's and not a symlink, and loses group and other write; on Windows a protected DACL, through `syscall` bindings that `mkwinsyscall` generates into `zsyscall_windows.go` | the standard library |
| `llmprovider/internal/filelock` | `TryLock`, `Unlock` and `ErrLocked`, for `FileTokenStore`'s refresh lock: `flock` on Unix, `LockFileEx` on Windows through `syscall` bindings that `mkwinsyscall` generates into `zsyscall_windows.go`; the operating system releases a lock when its file is closed or its process ends | the standard library |
| `llmprovider/internal/transport` | `DefaultClient`, `Identity` and its User-Agent, `BuildVersions`, `ParseRetryAfter`, `RetryAfter`, `ProbeGenerateHealth`; `ReplyReader`, the idle and size limits on a reply, and `AfterReply`, the mark `WithRetry` retries once | the standard library |
| `llmprovider/internal/wirecase` | G-wire's scenarios and canned replies through the new API, shared by the provider packages' tests | `llmprovider`, `internal/wiretest` |
| `internal/ambientcheck` | `TestNoAmbientState`, which parses every non-test source for environment reads and global logging; no package API | the standard library |

## Package graph

Each package imports only packages on a lower row. Every package may also use
the standard library; only `wizard` uses anything else, `golang.org/x/term`.

```text
wizard
providers                     Default(), New(id, …)
providers/<id>                one package per provider or gateway family
auth   catalog   internal/wire/<format>   llmtest
internal/wire
llmprovider                   the contract
internal/transport   internal/redact   internal/kiloendpoint   internal/ownerperm   internal/filelock
```

`internal/wiretest`, `llmprovider/internal/wirecase` and
`internal/ambientcheck` serve tests only, and nothing else imports them.

## Data flow

### A generation

1. **Construction.** The caller builds a provider with `providers.New(id,
   opts...)`, a `Registry`'s `New`, or the provider package's own `New`. It
   resolves its options with `ResolveOptions(id, opts)` into read-only
   `Settings`, and refuses a foreign or malformed option with
   `ErrInvalidRequest`. Construction makes no network call.
2. **Checks.** `Generate(ctx, req)` first runs `Capabilities.Check(req)`. An
   unsupported need fails with `ErrUnsupported`, and an invalid value with
   `ErrInvalidRequest`, before anything is sent.
3. **The request.** The provider encodes `req` in its wire format, from
   `internal/wire/<format>` or its own package. It asks its `TokenSource` for
   a `Token`, sets it with `Token.Apply`, and sends the request with the
   `Settings` client and `User-Agent`.
4. **The reply.** A failure status becomes an `*APIError` through
   `ClassifyHTTPError`. After a 401, an `InvalidatingSource` is invalidated,
   by the refused token when it is a `TokenInvalidator`, and the request sent
   once more. A success is decoded into a `Response`:
   `Output`, `FinishReason` and `Usage`.
5. **Around it.** `WithRetry` wraps a provider and retries by the error's
   kind; a provider that lists models still does, through it. `Stream` runs `Generate` and emits its result as events, since no
   built-in provider streams natively yet.

### A listing

`catalog.List(ctx, id, src, opts...)` resolves the same options, fetches the
provider's model list, and curates it into a `Catalog`. For the open catalogs
it ranks by `Profile`, with the model metadata unless `WithoutModelMetadata`
is set. A failed fetch gives the static catalog, with the failure in `Err`. A
listing in which nothing meets the profile stays live, with an empty
`Recommended` and the listing in `Usable` to search. A provider's own
`ListModels` (`ModelLister`) returns its listing, probed by default where the
provider probes.

The metadata document is fetched once per URL at a time, detached from the
callers that wait for it and bounded at 10 s; each lookup waits under its
own context, a request's for at most 5 s. Past the ten-minute cache the
cached document is served while one refresh runs in the background. A failed
fetch is remembered for a minute; a caller's own deadline is not. Listing
pages are read up to 8 MiB, and the document up to 32 MiB.

### The wizard

`ConfigureLLM` reads the menu from `Options.Registry`'s descriptors. It takes
the credential from the `Prompter`, from `Options.LookupEnv`, or from a
sign-in that saves the session to `Options.TokenStore`. It lists the models
with `catalog.List`, adding `Options.ProviderOptions`. Two cases go through
a provider's own `ListModels` instead, built from the registry: a ChatGPT
session, through `openai`, and an id that `catalog` does not list, such as a
third party's. It then asks for the model and fallbacks, and
returns a `Result`. The caller builds its provider from that `Result` and,
for a session, from the store.

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
  long as the service asks with up to a tenth more, up to a cap. A failure
  while reading a reply is retried once per call (0021-MADR D1), and a wait
  past the context's deadline returns the error at once.
- `Registry` holds `Descriptor` and `Factory` pairs and refuses a duplicate
  id. There is no global registry.
- `llmprovider/llmtest` has `Run`, the conformance suite every built-in
  provider passes, and `Fake`, a scriptable provider for tests. A `Harness`
  may switch on five more checks, and every built-in provider's does:
  - `Fidelity`: a request's `Model`, `Instructions` and a tool's output
    reach the wire;
  - `Garbled`: an undecodable 200 is `ErrIncomplete`, sent once through
    `WithRetry`;
  - `Truncated`, with `TruncatedReason`: a cut answer with a partial call
    is `ErrIncomplete`, with the reason the wire reports;
  - `StrictTools`: a call reply finishes `tool_calls`, with non-empty,
    valid JSON arguments;
  - `ReasoningCut`: an answer cut while the model was still reasoning, with
    no text or call, is `ErrIncomplete`, with the reason the wire reports.

  A harness that sets none passes as before.

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
  `FunctionCallOutputItem` and `ReasoningItem`, and `Response.Output` holds
  them.
  - A call's `Arguments` are the JSON the service sent, compacted, never
    decoded and re-encoded; no arguments are `"{}"`.
  - A `ReasoningItem`'s `Format` names the wire that produced it. Its
    `Signature` and `Encrypted` are replayed only to that wire, or from an
    item with no `Format` (0021-MADR D5).
  - A Responses request sent with `store: false` asks for encrypted
    reasoning, so a tool loop keeps it (0021-MADR D4). Kilo's
    `reasoning_details` are kept and replayed as they came (0021-MADR W9).
- **Answers:** every wire maps its finish reason the same way. A value with
  no constant is kept as sent, and a reply with a call finishes
  `tool_calls`. An answer with nothing usable is `ErrIncomplete`, with the
  service's reason (`content_filter`, a refusal) in `APIError.Reason`; so
  is an answer cut while the model was still reasoning. A call id another
  wire made, such as Gemini's `name#index`, goes to the Messages wire in a
  form Anthropic accepts, unique in the request, with each result paired to
  its call.
- **Construction:** `providers.New(id, opts...)`, or the provider package's
  own `New` (`opencode.NewZen` and `NewGo`); every provider is in its own
  package. Options are `Option` values: the common ones in `llmprovider`
  (`WithBaseURL`, `WithHTTPClient`, `WithReasoning`, …), and a provider
  package's own, scoped to it. `WithRetry` retries by the error's kind.
- **Transport:** without `WithHTTPClient`, each provider builds one client,
  `internal/transport`'s `DefaultClient`: 30 s to connect, 300 s to the
  first byte, no total timeout, HTTP/2 kept, and `HTTP_PROXY`, `HTTPS_PROXY`
  and `NO_PROXY` honoured. Every generation goes through `internal/wire`'s
  `Post`, which reads the reply under a 300 s idle limit and, unless it is
  an event stream, a 16 MiB limit, whatever client the caller gave
  (0021-MADR D2). Auth requests carry their own 30 s bound. It carries the provider's requests, its listing
  and probes, and its OAuth session's refreshes when the session has no
  client of its own.
- **Errors:** one structured error, `*APIError`, whose `Kind` is a sentinel
  such as `ErrRateLimited`, `ErrQuotaExhausted`, `ErrAuthFailure` or
  `ErrIncomplete`; a rate limit carries its `RetryAfter`, and a cut-short
  response its `Reason`. `Retryable()` says whether to try again. Every
  sentinel's message starts `llmprovider:`; a wrapped error keeps its
  package's or provider's prefix, such as `oauth:`. Error bodies pass through
  `redact.String`: the first 16 KiB of a message, control characters
  removed, is redacted, and 512 bytes of it kept; a service's error code is
  stripped the same way and kept to 128 bytes. Each redaction pattern runs
  only when the text holds one of its literal anchors.

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
  A source that is also a `TokenInvalidator`, as both are, is told which
  token was refused, and ignores a 401 on a token it has already replaced,
  so concurrent 401s renew it once (0021-MADR D5). Every built-in provider
  does it through `internal/wire`'s `Reauth`, and `llmtest` checks it
  (0020-MADR F2).
- **Sessions are `auth`'s.** A provider gives a session with no HTTP client
  its own through `OAuthSession.UseHTTPClient`, so refreshes share the
  provider's transport.
- **`OAuthSession`** (`auth`) is a refreshable `TokenSource` for ChatGPT and
  Grok.
  - **Creating one:**
    - `LoginBrowserOAuth` uses PKCE on a loopback redirect.
    - `StartDeviceOAuth` returns a `DeviceLogin` handle: the user code, the
      verification URI and the expiry, then `Wait` and `Cancel`. `Wait`
      polls through transport errors, 429 and 5xx, backing off up to 60 s,
      until the code expires.
    - `LoginDeviceOAuth` is the same flow in one call.
  - **Checked before use.** Every login verifies its `id_token` before any
    claim is used:
    - the signature against the issuer's published keys, fetched from
      discovery's `jwks_uri`, cached, and refetched once for an unknown `kid`;
    - the issuer, the audience and the expiry, and the Grok nonce.

    A missing or failing `id_token` fails the login. A caller's issuer whose
    discovery fails is an error; only the built-in issuers fall back to
    built-in endpoints and keys.
  - **Refresh.** The session refreshes before expiry and once after a 401:
    - a token is due five minutes before it expires, or at half its
      lifetime when that is shorter;
    - an early refresh that fails, other than with `ErrAuthFailure`, keeps
      the current token while it has more than 30 s left, logs the failure,
      and waits 10 to 30 s before the next;
    - each attempt is bounded at 15 s, and a reply is read up to 1 MiB;
    - a rotated refresh token is kept even when the rest of the reply cannot
      be decoded.

    A rotation is kept even when saving it fails: the failure goes to the
    session's `Logger`, and the save is retried at most every 30 s, trying
    the lock once without waiting. After a rejected refresh the session
    re-reads the store, and adopts a rotation another process saved.
  - `RevokeOAuthSession` ends it.
- **`TokenStore`** persists sessions. `FileTokenStore` keeps one `0600` JSON
  file per provider:
  - it writes through a temp file, with `fsync`, rename and a directory
    `fsync`;
  - it refuses a file over 64 KiB;
  - as a `RefreshLocker`, it holds an exclusive OS file lock on
    `<provider>.oslock` across processes for each refresh, so a refresh
    token is never spent twice. The operating system releases the lock when
    its holder closes it or dies, so nothing stale is left behind. A waiter
    gives up after 40 s.
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

  It is an `InvalidatingSource` and a `TokenInvalidator`: a provider that
  gets a 401 on its current output invalidates the source and retries
  once.
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
  `catalog.WithProfile` and `catalog.WithKiloOrganization`. It lists the ten
  built-in ids. For any other id it returns an error matching
  `ErrUnsupported`, before it reads the options.
- `catalog.Search` matches a query against a list, highest tier first and
  in the list's order within a tier: the id exactly, as a prefix or a
  substring, then without separators (`gpt41` finds `gpt-4.1`), then a
  label's display name, then token prefixes. A subsequence of the id is
  tried only when nothing else matched and the query has three or more
  characters. A glob scores an id match above a label-only one. The static
  catalogs are the fallback. `catalog.Static(provider)` returns a copy, and
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
  using models.dev-format metadata from `https://models.opencode.ai/api.json`.
  `WithModelMetadataURL` names another document, and `WithoutModelMetadata`
  turns the fetch off. `catalog.OptionsFromEnv()` sets either from
  `LLMPROVIDER_MODELS_METADATA_URL` and `LLMPROVIDER_DISABLE_MODELS_METADATA`
  for a caller who passes its result.
- **No ambient state.** Library code reads the environment only in
  exported `…FromEnv` helpers a caller opts into (`ModelProbesFromEnv`,
  `catalog.OptionsFromEnv`, `auth.GrokFlowFromEnv`). The default transport
  also reads the proxy settings. Nothing logs to the global logger. The test
  in `internal/ambientcheck` fails on any other read.

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
- **What it accepts.** A remote provider's base URL must be `http` or
  `https` with a host, and carry no userinfo, query or fragment; plain
  `http` to a host that is not loopback is used only once confirmed. A
  pasted OpenAI credential that is another vendor's key, or not a JWT, or
  an expired one, is refused before anything is saved.
- **`TextPrompter`** strips control characters from what `Notify` writes.
  Its masked entry redraws only when its input has caught up, pads rather
  than erasing the line, ignores a lone Escape, clears on Ctrl-U, and
  cancels on Ctrl-C with an error matching `context.Canceled`.

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
  `vuln`, `pre-add-check`, `parity-check`, `dep-check`, `coverage-check`,
  `api-check`, `generate-check`, `records-check`, `gate-selftest`, `help`.
  `lint` runs golangci-lint twice: for the host and with `GOOS=windows`, so the
  `_windows.go` files are linted. Its `nolintlint` requires every `//nolint`
  to name its linter and still be needed.
- **`dep-check`** (`scripts/check_deps.py`) reads every package's
  dependencies with `go list -deps`, and again with `-test` for its tests,
  for the host and with `GOOS=windows`, so an import in a `_windows.go` file
  or in a test is checked too. It fails when a package other than `wizard`
  reaches beyond the standard library and this module, or `wizard` beyond
  `golang.org/x/term` and the `golang.org/x/sys` it needs, and when `go.mod`
  requires any other module.
- **`coverage-check`** (`scripts/check_coverage.py`) measures each package
  with `go test -cover` against its floor in `scripts/coverage-floors.txt`:
  - `llmprovider` (95.9 %) and `wizard` (86.4 %), two points below their
    measure on 2026-10-05 (0021-PLAN step 6.3);
  - `internal/redact` and `llmprovider/internal/ownerperm` (100.0 %, the
    latter measured on Unix, 0010-PLAN P6b);
  - 80 % for every other package.

  `llmprovider/internal/wirecase` (89.5 %), which has no tests of its own,
  is measured by the provider packages' tests, with `-coverpkg`. A package
  with no statements has no floor.
- **`api-check`** (`scripts/check_api.py`) runs `apidiff`, at the version
  pinned in the script, with `go run`, against the latest `v1.X.Y` tag. It
  fails on an incompatible change outside `llmprovider/x/` and the internal
  packages. Before the first such tag it reports that there is nothing to
  compare, and passes.
- **`generate-check`** (`scripts/check_generated.py`) reruns each
  `//go:generate` line that names `-output` (as `-output X` or `-output=X`),
  into a temporary file, and fails when the committed file differs, or when
  it checks no file at all. Today that is `mkwinsyscall` for
  `llmprovider/internal/ownerperm` and `llmprovider/internal/filelock`, each
  pinned in its `doc.go`; the generator is not a module requirement.
- **`scripts/check_parity_map.py`** (G-parity) fails when an identifier in
  `docs/guides/migrating-from-mcplib.ids`, the exported identifiers of
  `mcplib` `v1.6.0` `llmprovider` and `wizard`, has no row in
  `docs/guides/migrating-from-mcplib.md`, a row names one that is not in
  the list, or a row's "SDK equivalent" is empty, names an identifier the
  SDK does not export, or names nothing that resolves and is not one of the
  markers `none`, `removed` or `Here`. `make parity-check` runs it.
- **`records-check`** (`scripts/check_records.py`) checks the records
  under `docs/decisions/` and `docs/reports/`:
  - names and directories;
  - statuses by kind;
  - that each PLAN has its number's MADR and names it;
  - that `docs/README.md`'s Records table has one row per record, with the
    file's status, and a count that matches.

  `--next` prints the next record number.
- **`gate-selftest`** (`scripts/test_gates.py`, standard-library
  `unittest`) copies the tree, plants one breach per gate (dep-check,
  generate-check, parity-check, coverage-check, api-check, records-check),
  and requires each gate to fail, then every gate to pass on a clean copy.
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
  `golangci-lint run -c .golangci.yml --build-tags live_gateways ./...` for
  the host and with `GOOS=windows` (so the live-tagged tests and the
  `_windows.go` files are linted too), `go vet` and `go test -race` on their
  packages, `go mod tidy -diff`, and `govulncheck ./...`. `make pre-add-check` runs it, and so does
  the machine-wide agent gate before an agent `git commit` that stages Go
  files.
- **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with
  the Go version read from `go.mod`, on each push to `main`, each pull
  request, and weekly: `go test -shuffle=on`. On Linux it also runs:
  - `go test -race -shuffle=on`;
  - `govulncheck`, pinned in the workflow;
  - `go vet`, `gofmt`, `go mod tidy -diff` and `make lint`;
  - `go vet -tags live_gateways`;
  - `make parity-check dep-check coverage-check api-check generate-check
    records-check gate-selftest`.

  It checks out the full history, so that `api-check` sees the tags. A job
  stops after 30 minutes, and a newer push to a pull request cancels its
  older run.

## What is not here

- **Native streaming.** Every provider's `NativeStreaming` is `Unsupported`,
  so `Stream` uses the `Generate` fallback.
- **`llmprovider/x/`.** No experimental API exists.
