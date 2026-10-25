# Architecture

How `go-llmprovider-sdk` is put together, as it is now. This file carries no
history and no rationale: the records under [decisions/](decisions/) hold the
argument, and [README.md](README.md) indexes them.

## What it is

The Go module `github.com/maccavelli/go-llmprovider-sdk`: a library for calling
LLM providers and authenticating to them. It has no binary. It requires Go
1.27.3, `golang.org/x/term` and, indirectly, `golang.org/x/sys`.

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
.github/workflows/docs.yml  Docs: the Markdown gates, for a Markdown-only change
scripts/go-precheck.sh      the pre-add check
scripts/check_parity_map.py G-parity: the mcplib migration map is complete
scripts/check_deps.py       dep-check: only wizard leaves the standard library
scripts/check_coverage.py   coverage-check, with scripts/coverage-floors.txt
scripts/check_api.py        api-check: apidiff against the latest v1 tag
scripts/check_generated.py  generate-check: go:generate outputs are current
scripts/check_records.py    records-check: the decision records and their index
scripts/check_tables.py     tables-check: the import tables and the live variables match the code
scripts/check_fixtures.py   fixtures-check: recorded fixtures hold no account data
scripts/check_changelog.py  changelog-check: CHANGELOG.md has a section for every release tag
scripts/check_docs.py       docs-check: the API audit's R50-R52, the examples and the concurrency and error docs
scripts/test_gates.py       gate-selftest: each gate fails on a planted breach
tools/                      maintainer tools, not run by CI: the local gate,
                            plants, link check, identifier scan, apidiff,
                            edits, record status and lint, claims,
                            release helpers; tools/README.md
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
llmprovider/                the contract, errors, options and credential sources
llmprovider/auth/           OAuth sessions and logins, vendor CLI sessions, token stores
llmprovider/llmtest/        the conformance suite and a fake provider
llmprovider/providers/      the built-in providers, by id
llmprovider/providers/openai/  OpenAI: the Responses API, with a key or a ChatGPT plan session, and the ChatGPT backend
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
llmprovider/internal/transport/ the shared default transports, the client identity, Retry-After and the rate-limit resets, the listing probe and its stores
llmprovider/internal/kiloendpoint/ Kilo's endpoints, derived from a credential
llmprovider/internal/listlimit/ catalog.WithLimit's value, for catalog and the providers
llmprovider/internal/cacheref/ catalog.WithCache's value, for catalog and the providers
llmprovider/internal/nilsource/ a typed-nil TokenSource's test, for llmprovider and catalog
llmprovider/internal/chatgpt/ a ChatGPT session's headers and listing request, and a plan session's, for catalog and providers/openai
llmprovider/internal/ownerperm/ a directory and its files private to the current user
llmprovider/internal/filelock/ an exclusive OS lock on an open file, for the refresh lock
llmprovider/internal/wirecase/ G-wire's scenarios through the new API, for tests only
llmprovider/internal/retired/ the retired model ids, with why, for the catalog tests and the maintainer tool
llmprovider/internal/catalogtool/ the maintainer tool's proposals for the static lists
llmprovider/internal/cmd/catalogs/ the maintainer tool: go run, prints the proposals, edits nothing
llmprovider/internal/cmd/sdkaudit/ the API audit: go run, reports or checks R50–R52, edits nothing
wizard/                     interactive provider configuration
internal/redact/            secret redaction and masking
internal/wiretest/          G-wire's request recorder, for tests only
internal/nettest/           a refused connection and a free loopback port, for tests only
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
| `llmprovider` | the contract: request and response types, errors, options, the `Registry`, retry middleware, and the credential sources (`Token`, `TokenSource`, `StaticToken`, `CommandToken`) | `internal/transport`, `internal/redact`, `internal/nilsource` |
| `llmprovider/auth` | OAuth sessions and their refresh and revocation, the browser, device and Kilo device logins, the ChatGPT plan sign-in (`LoginChatGPTPlan`, `ChatGPTPlanOptions`), `id_token` checks, `VendorCLISession`, `TokenStore` and `FileTokenStore`, and the issuers and client ids | `llmprovider`, `internal/transport`, `internal/kiloendpoint`, `internal/redact`, `internal/ownerperm`, `internal/filelock` |
| `wizard` | the configuration flow and its `Prompter` seam | `llmprovider`, `auth`, `catalog`, `providers`, `providers/openai`, `internal/redact`, `golang.org/x/term` |
| `internal/redact` | `Redact` and `String` (hide a secret completely), `MaskSecret` (show a suffix for identification), `StripControl` (remove terminal control characters) and `Field` (strip and bound a service's code or reason) | the standard library |
| `internal/wiretest` | an `httptest` server that records requests as normalised JSON, and golden-file comparison; imported only by tests | the standard library |
| `internal/nettest` | `Refusing`, a client whose every dial is refused, with `RefusedURL`; `Port`, a loopback port outside every ephemeral range; so that no test depends on a freed port (0045-MADR); imported only by tests | the standard library |
| `llmprovider/llmtest` | `Run`, the conformance suite, and `Fake`, a scriptable provider | `llmprovider` |
| `llmprovider/providers` | `Default()`, a new `Registry` of the built-in providers, `DefaultWith(opts...)`, that `Registry` with `opts` given to every built-in (0048-MADR D1), and `New(id, opts...)`; it holds all ten provider ids | `llmprovider` and the provider packages |
| `llmprovider/providers/openai` | OpenAI through the new contract: `New`, `WithStore`, `ListModels`, `Stream`, and `ChatGPTUsageURL` | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/chatgpt`, `internal/wire/responses`, `internal/wire`, `internal/wire/deferral`, `internal/cacheref`, `internal/transport` |
| `llmprovider/providers/claude` | Claude through the new contract: `New`, `ListModels` and `Stream` | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/wire`, `internal/wire/messages`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/providers/gemini` | Gemini through the new contract: `New`, `WithStore`, `ListModels`, and `Stream`, with the Interactions wire's decoders and its refusal matcher (0062-MADR D4) | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/wire`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/providers/grok` | Grok through the new contract: `New`, `WithStore`, `ListModels`, and `Stream` | `llmprovider`, `catalog`, `internal/listlimit`, `internal/wire/responses`, `auth`, `internal/wire`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/providers/opencode` | OpenCode Zen and Go through the new contract: `NewZen`, `NewGo`, `WithRoute`, `ListModels` and `Stream` (native on every route), with the route table, built from the embedded `routes_snapshot.json` | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/wire`, `internal/wire/responses`, `internal/wire/chatcompletions`, `internal/wire/messages`, `internal/wire/generatecontent`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/providers/kilo` | Kilo through the new contract: `New`, `WithOrganization`, `WithCapabilities`, `WithDataCollection`, `ListModels` and `Stream` | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/wire/chatcompletions`, `internal/kiloendpoint`, `internal/wire`, `internal/wire/deferral`, `internal/wire/messages`, `internal/cacheref` |
| `llmprovider/providers/huggingface` | Hugging Face through the new contract: `New`, `ListModels` and `Stream` | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/wire/chatcompletions`, `internal/wire`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/providers/together` | Together AI through the new contract: `New`, `ListModels` and `Stream` | `llmprovider`, `auth`, `catalog`, `internal/listlimit`, `internal/wire/chatcompletions`, `internal/wire`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/providers/ollama` | Ollama through the new contract: `New`, `ListModels` and `Stream` | `llmprovider`, `auth`, `catalog`, `internal/wire/chatcompletions`, `internal/wire`, `internal/wire/deferral`, `internal/cacheref` |
| `llmprovider/internal/wire` | the JSON keys, `ToolArguments` and `SystemPrompt` that the shared formats use; `Post`, `Reauth` and `DecodeError`, the request path every provider sends through; `Send` and `StreamEventsRetry`, the openers that build a request again and send it once more while a `Retry` check says so, at most `MaxSends` (7) times (0062-MADR D5); `Retries`, which joins checks; `OpenStream` and `Stream`, a reply held open while its events are read; `ReadEvents`, the server-sent-events reader every wire's stream decoder uses, with its `EventLimit`, which marks a decoder's error as after the reply (0053-MADR D3); `StreamIndex` and `MaxStreamIndex` (4096), the bound on every index a stream sends, past which the stream ends with `ErrIncomplete` (0053-MADR D2); `StreamEvents` and `StreamEventsFor`, the native stream every provider's `Stream` runs; the shared tool lists; `RemapCallIDs`, which fits another wire's call ids to a wire's rule; `ProbeHealth`, a listing's probes through a probe store; `Arena`, which holds the strings a request body points to; `Ladder`, the efforts a request may be sent at, and `DegradedWarning` (0062-MADR D3); `Learned`, the memo of refused thinking settings, `Learner`, one request's effort and its `Retry` check, `Matcher` and `Refusal`, what a wire's matcher reads, with `QuotedEfforts` and `PlainEfforts` (0062-MADR D4, D5) | `llmprovider`, `internal/transport`, `internal/redact` |
| `llmprovider/internal/wire/responses` | the OpenAI Responses wire: `Input`, `Decode`, `DecodeFor`, `ReadStream`; `MatchRefusal`, its refusals of an effort (0062-MADR D4); `Events`, which yields a stream's events as they arrive; for deferred tools, `InputLoading` (the native form), `InputDispatch` (Grok's wrapper) and `Decoder`, which names the Search tool and the dispatched ones | `llmprovider`, `internal/wire`, `internal/redact`, `internal/wire/deferral` |
| `llmprovider/internal/wire/chatcompletions` | the Chat Completions wire: `Opts`, `Body`, `Decode`, `DecodeFor`, `Events`; `MatchRefusal`, its refusals of an effort (0062-MADR D4) | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/messages` | the Anthropic Messages wire, with its thinking shape: `FromItems`, `Decode`, `Events`, `AddThinking`, and from a model's facts `Thinking`, `FactsThinking`, `PreferredShape` and `AddThinkingFor`; deferred tools: `FromItemsLoading`, `ClaudeVersion`; forced tool choice per model: `Policy`, `Forcing`, `CheckForcing`, `AddToolChoice`, `StructuredCall`, `StructuredStream`, `RefusalRetry`, `ToolChoiceWarning`; `MatchRefusal`, its refusals of a thinking shape or an effort (0062-MADR D4) | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/generatecontent` | Gemini's generateContent wire, with its thinking shape: `SystemInstruction`, `Contents`, `Decode`, `Events`, `ThinkingConfig`, and from a model's facts `Thinking`, `FactsThinking`, `PreferredShape` and `ThinkingConfigFor`; `MatchRefusal`, its refusal of a level (0062-MADR D4, amendment "Phase 3's google-route levels, and 0013's B9") | `llmprovider`, `internal/wire` |
| `llmprovider/internal/wire/deferral` | how a request's deferred tools are sent, decided once for every encoder: `Wire`, `Form`, `Of`, `ForRequest`, `Deferring`, `Tools`, `Loaded`, `SearchLoads`, `SearchName`; `Warning` and `Warnings`, the `WarningDeferralEmulated` a form earns; and Grok's dispatch: `UseTool`, `DispatchTool`, `Dispatched`, `Loads` and `CheckDispatch` (0040-MADR D1 to D7) | `llmprovider`, `internal/wire/messages` |
| `llmprovider/catalog` | `List`, `Catalog`, its `Menu` of `MenuEntry` with each `Caveat` (0041-MADR D3), `ListRaw` with its `RawListing` and `RawModel` (0041-MADR D9), `Static`, `Rank`, `Search`, `Match`, `Label`, `Profile`, `Metadata`, `LookupMetadata`, `LookupMetadataWith`, `CachedMetadataWith`, `KiloModelCapabilities`, `ValidateOllamaURL`, `ObservedWorkload`, `ListSource` (`ListLive`, `ListStale`, `ListStatic`), `Catalog.Source` and `Catalog.FetchedAt`, and the options `WithProfile`, `WithWorkload`, `WithLastGood` and `WithKiloOrganization`; model metadata (0033-MADR D1, D3): `ModelInfo`, `ModelReasoning`, `ModelCost`, `ModelRates`, `ModelCostTier`, `Field`, `Modality`, `ModelStatus`, `ParseMetadata`, and `Metadata`'s `Model`, `Models`, `Encode` and `Name`; the resolver: `Layer`, `LookupModel`, `CachedModel`, `CachedFacts` and the option `WithModelLayer`; `Cache` and the option `WithCache` (0048-MADR D1) | `llmprovider`, `internal/chatgpt`, `internal/kiloendpoint`, `internal/listlimit`, `internal/transport`, `internal/cacheref`, `internal/nilsource` |
| `llmprovider/internal/kiloendpoint` | `Resolve`, `Route` and Kilo's base URL, for `catalog`, `providers/kilo` and the Kilo device login | the standard library |
| `llmprovider/internal/nilsource` | `Is`, which tells a typed-nil `TokenSource`, a nil pointer, map, slice, func, chan or interface held in a non-nil interface, from a value, for `ResolveOptions` and the catalog's entry points (0050-MADR D1, D2) | the standard library |
| `llmprovider/internal/listlimit` | `Value`, `catalog.WithLimit`'s option value, `Of`, which reads it from a provider's options, and `Cut`, for `catalog` and the providers that cut their own fallback listings with it (0038-MADR A3) | the standard library |
| `llmprovider/internal/cacheref` | `Value`, `catalog.WithCache`'s option value: a `catalog.Cache`'s store, its `transport.ProbeStore`, its `messages.Refusals` and its `wire.Learned`; `New`, a whole one; `From`, which reads it from a listing's or a provider's options; `ProbeStore`, `LearnedRefusals` and `LearnedThinking` (0048-MADR D1; 0049-MADR D3; 0062-MADR D5) | `internal/transport`, `internal/wire`, `internal/wire/messages` |
| `llmprovider/internal/chatgpt` | a ChatGPT session's `BaseURL`, `IsSession`, `SetHeaders` and its header names, `ClientVersion`, and `ModelsRequest`, the Codex backend's listing request; a plan session's `IsPlanSession` and `PlanModelsRequest`; for `catalog`'s `ListRaw` and `providers/openai` (0041-MADR D9; 0043-MADR D5) | `llmprovider`, `auth`, `internal/transport` |
| `llmprovider/internal/ownerperm` | `MkdirAll` and `File`, for `FileTokenStore`: modes 0700 and 0600 on Unix, where an existing directory must be the user's and not a symlink, and loses group and other write; on Windows a protected DACL, through `syscall` bindings that `mkwinsyscall` generates into `zsyscall_windows.go` | the standard library |
| `llmprovider/internal/filelock` | `TryLock`, `Unlock` and `ErrLocked`, for `FileTokenStore`'s refresh lock: `flock` on Unix, `LockFileEx` on Windows through `syscall` bindings that `mkwinsyscall` generates into `zsyscall_windows.go`; the operating system releases a lock when its file is closed or its process ends | the standard library |
| `llmprovider/internal/transport` | `DefaultClient` and `Config`, over the transports the process keeps; `Identity` and its User-Agent, `BuildVersions`, `ParseRetryAfter`, `RetryAfter`, `RateLimitReset`; `ProbeGenerateHealth`, and `ProbeGenerateHealthCached` with its `ProbeStore`, `NewProbeStore`, `ProbeKey` and `Fingerprint`; `Unsendable`; `ReplyReader`, the idle and size limits on a reply, and `AfterReply`, the mark `WithRetry` retries once | the standard library |
| `llmprovider/internal/wirecase` | G-wire's scenarios and canned replies through the new API, shared by the provider packages' tests | `llmprovider`, `internal/wiretest` |
| `llmprovider/internal/retired` | `IDs`, the retired model ids by provider, each with why (0033-MADR D4) | `llmprovider` |
| `llmprovider/internal/catalogtool` | `Propose`, the maintainer tool's proposals: ids to remove from a static list, and models to add | `llmprovider`, `catalog`, `internal/retired` |
| `llmprovider/internal/cmd/catalogs` | the maintainer tool's command, `go run ./llmprovider/internal/cmd/catalogs`: it reads models.dev and each keyed listing, and prints the proposals; `-snapshot` records the ranked listings, and `-firstparty` the first-party listings, every page, redacted | `llmprovider`, `catalog`, `internal/catalogtool`, `internal/transport` |
| `llmprovider/internal/cmd/sdkaudit` | the API audit, `go run ./llmprovider/internal/cmd/sdkaudit`: the measures of 0048-MADR as JSON, or with `-check` the violations of R50–R52, from the source by `go/ast`; it changes nothing | the standard library |
| `internal/ambientcheck` | `TestNoAmbientState`, which parses every non-test source for environment reads and global logging; no package API | the standard library |

## Package graph

Each package imports only packages on a lower row. Every package may also use
the standard library; only `wizard` uses anything else, `golang.org/x/term`.

```text
wizard
providers                     Default(), New(id, …)
providers/<id>                one package per provider or gateway family
catalog   internal/wire/<format>   llmtest
internal/chatgpt              a ChatGPT session's headers and listing request, and a plan session's
auth
internal/wire
llmprovider                   the contract
internal/transport   internal/redact   internal/kiloendpoint   internal/listlimit   internal/ownerperm   internal/filelock
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
   kind, and `WithRetryNotify` also reports each wait to a callback; a
   provider that lists models still does, through it, and it does
   not stream natively: `Stream` over it runs its `Generate`, so a failure
   is retried whole. `WithStreamRetry` streams natively over a provider
   that does: it retries a failure before the first event, and passes
   every event after it through, a failure there ending the stream
   (0051-MADR). `Stream` uses a provider's own `Streamer`: every
   built-in provider asks its wire for an event stream and yields each event
   as it arrives (0031-MADR). A provider without one runs `Generate` and
   emits its result as events.

### A listing

`catalog.List(ctx, id, src, opts...)` resolves the same options, fetches the
provider's model list, and curates it into a `Catalog`. Every provider ranks
by `Profile`, with the model metadata unless `WithoutModelMetadata` is set;
OpenAI, Claude, Gemini and Grok rank by their name curators only with no
document (0041-MADR D4). A listing read in the last ten minutes, for the
same provider, base URL, Kilo organization and credential, is ranked again
with no request, and callers at once share one fetch, and its result, a
failure included; a caller that is cancelled ends only its own wait
(0053-MADR D4). A transient failure
(a timeout, a 5xx, a rate limit, a network error) serves the last good
listing, at most 24 hours old, or one the caller kept and hands in with
`WithLastGood`: `Source` is `ListStale`, `Live` stays true, `Err` is the
failure, and `FetchedAt` says when it was read. Any other failed fetch
gives the static catalog (`ListStatic`), with the failure in `Err`. Ollama
lists every time, and returns its errors (0041-MADR D6, A6). A
listing in which every model has a caveat stays live (on Ollama, a caveat
other than `CaveatNoTools`), with an empty `Recommended` and the listing in
`Usable` to search. A provider's own
`ListModels` (`ModelLister`) returns its listing, probed by default where the
provider probes.

`catalog.ListRaw(ctx, id, src, opts...)` sends the same requests through
the same code, with a recording transport over the caller's, and returns
the listing as the provider sent it: every page byte for byte, and each
entry of the pages' model arrays with the provider's own id (0041-MADR D9).
It filters, ranks and caveats nothing, fetches no metadata document,
neither reads nor writes the listing cache, and never falls back: a failed
page is its error. It renews a refused credential once, as `List` does.
A ChatGPT session lists the Codex backend, with the request the openai
provider sends (`internal/chatgpt`); a ChatGPT plan session reads the
`models` array of the platform's `/models`; Ollama's listing is `/api/tags`
alone.

The metadata document is fetched once per URL at a time, detached from the
callers that wait for it and bounded at 10 s; each lookup waits under its
own context, a request's for at most 5 s. Past the ten-minute cache the
cached document is served while one refresh runs in the background. A failed
fetch is remembered for a minute; a caller's own deadline is not. A refresh
sends the cached document's `ETag` as `If-None-Match`, and a 304 renews the
entry without a download or a decode. Listing pages are read up to 8 MiB,
and the document up to 32 MiB, decoded as it streams in.
`CachedMetadataWith` returns the cached document without waiting, and
starts the fetch when none is cached.

### Model metadata

`catalog.LookupModel(ctx, id, src, model, opts...)` returns a `ModelInfo`.
That holds the model's limits, modalities, capabilities, reasoning
descriptor, prices in USD per million tokens (with a list of tiers), and
lifecycle. It is merged field by field from three kinds of layer, a later one
winning (0033-MADR D1 to D3):

1. **models.dev's document,** unless `WithoutModelMetadata` is set. Nine
   sections are read: the four open catalogs', and `openai`, `anthropic`,
   `google`, `xai` and `kilo`.
   - A fetched document keeps each section's bytes, and decodes a section
     the first time it is read, once.
   - `canonical_model_id` is `CanonicalID`. A gateway model's missing
     `cache_read` comes from its canonical model's first-party entry
     (0041-MADR D1).
   - `ParseMetadata` decodes every section of a document a caller
     persisted.
   - `Metadata.Encode` writes the sections back as they were read,
     losslessly.
2. **The provider's listing,** when a `TokenSource` is given: the fields
   each listing is authoritative for, decoded from the replies `List`
   already reads.
   - Anthropic: limits, capabilities and release date, never its lifecycle.
   - Gemini: limits and thinking.
   - xAI: window, efforts, prices, from cents per 100 million tokens, and
     release date.
   - Kilo: window, output cap, prices from per-token strings, expiration
     and release date.
   - Together: window and prices.
   - OpenAI: shutdown date and release date.
   - Hugging Face: release date.

   A listing's date is read only where it was measured to be a release
   date, and never when every dated row shares one, or from a zero
   (0041-MADR D1; 0041-PLAN Phase 2).
   - Ollama: window, capabilities and what it writes, from one model's
     `/api/show`; what it writes from `/api/tags` when that reply fails
     (Ollama v0.30.0 on).
3. **The caller's layers,** given with `WithModelLayer`. Any `Layer` is
   one, `Metadata` among them.

`ModelInfo.Has` says whether a layer set a field, so a known zero, such as a
free price, is told apart from an unknown value. `SourceOf` names the layer.
`Known` is how a layer marks the fields it sets. `ModelCost.RatesFor` picks
the highest tier the input exceeds. `Cost.Tiers` merges as a whole.

A listing's metadata is cached for ten minutes, keyed by provider, the base
URL the listing reads and a fingerprint of the credential, never the
credential itself. No base URL is the provider's default, so a `List` with
none and a provider's request, which names its default, share an entry.
Claude's lister reads `/v1/models` under a base with or without `/v1`
(0041-MADR A8). `List` fills the cache too. `CachedModel` answers from the caches and the caller's
layers without waiting, and starts a refresh where one is missing or stale.
`CachedFacts` is what a request reads (0041-MADR D1, A3): the same merge,
from the caches alone, starting no fetch. A provider passes its request's
own token, so its token source is asked once per request. Every built-in
provider reads it once per request for its output limit; Claude and
OpenCode's messages route for the thinking shape; Gemini and OpenCode's
google route for the level; Grok for its efforts; and Together, Hugging
Face and Ollama for the replay field, `ModelReasoning.ReplayField`. A
refusal the model returned outranks the facts: the effort and the thinking
shape sent are those it has not refused, then those its facts list, then
the request's (0062-MADR D2, D3). With no facts, the request decides.

These caches, and the models.dev document's and the probes', are the
process's, shared by every listing and provider, unless a caller gives
`catalog.WithCache(&c)` (0048-MADR D1). A `catalog.Cache` holds its own
listings, metadata documents, probe results, learned forced-choice
refusals (0049-MADR D3) and refused thinking settings (0062-MADR D5), shared
only by the listings and providers given it;
`providers.DefaultWith(catalog.WithCache(&c))` gives it to every built-in a
`Registry` builds. `internal/cacheref` carries it in the option's value: the
catalog's store, a `transport.ProbeStore`, which each provider reads in `New`
and passes to `wire.ProbeHealth`, a `messages.Refusals`, which Claude and
OpenCode give to their forced-choice policy, and a `wire.Learned`, which
every provider gives to each request's `wire.Learner`.
A model no layer knows is not an error; a failed source is joined into
`LookupModel`'s error, beside what the other layers set.

### The wizard

`ConfigureLLM` reads the menu from `Options.Registry`'s descriptors. It takes
the credential from the `Prompter`, from `Options.LookupEnv`, or from a
sign-in. A sign-in's session is saved to `Options.TokenStore` once the
listing has not refused it, after any refresh the listing left running, so a
refused sign-in leaves the store as it was. It lists the models with
`catalog.List`, adding `Options.ProviderOptions`. Two cases go through
a provider's own `ListModels` instead, built from the registry: a ChatGPT
session, through `openai`, and an id that `catalog` does not list, such as a
third party's. The menu recommends the whole curated list, best six first,
cut on every path to `Options.ModelLimit` when it is positive; the wizard
passes it to `catalog.List` as `catalog.WithLimit`. It then asks for the
model and fallbacks, and returns a `Result`. The caller builds its provider from that `Result` and,
for a session, from the store.

## The contract

`llmprovider` holds the generation contract (0015-MADR D3–D10):

- `Provider` is `ID`, `Capabilities` and `Generate(ctx, *Request)
  (*Response, error)`. `GenerateText`, `GenerateToolCall` and
  `ProbeReasoning`, one generation at `EffortMax` that returns the refusals
  the provider learned from it (0062-MADR D7), are functions over any
  `Provider`.
- `Capabilities` gives each of `Tools`, `ForcedToolChoice`, `Reasoning`,
  `Continuation` and `NativeStreaming` as `Unsupported`, `BestEffort` or
  `Supported`. `Capabilities.Check(req)` validates a request and refuses an
  unsupported need before any network call, with `ErrUnsupported`.
- Two tool flags are sent only where a wire has the field (0032-MADR
  D1–D3). Each provider's package doc names the wires that do not send
  them.
  - `Tool.Strict` is a `*bool`; nil sends nothing.
    - Set, it is `strict` on Responses (`openai`, `grok`, OpenCode's
      `responses` route).
    - It is the tool's `strict` on Messages (`claude`, OpenCode's
      `messages` route; 0034-MADR D4).
    - It is `function.strict` on Chat Completions for `huggingface`,
      `kilo`, `together` and OpenCode's `chat_completions` route.
    - Interactions, generateContent and `ollama` do not send it.
  - `FunctionCallOutputItem.IsError` marks a failed call's result.
    - It is `is_error` on Messages and on an Interactions `function_result`.
    - On generateContent, the result is sent as `response: {"error": …}`
      in place of `{"output": …}`.
    - Responses and Chat Completions have no such field, so the `Output`
      is sent as written. No wire rewrites it.
- `Request.Cache` (`CacheShort`, `CacheLong`; zero asks for nothing) asks
  the service to cache the prompt (0039-MADR). An unknown value is refused
  before the network. Each wire asks in its service's way, and a request
  without it is sent unchanged:
  - Messages (`claude`, OpenCode's `messages` route): the top-level
    `cache_control` and one breakpoint, on the system prompt (then one text
    block) or else the last tool, both with `ttl` `5m` or `1h`.
  - Responses (`openai` with an API key, `grok`, OpenCode's `responses`
    route) and `together`: `prompt_cache_key`, the provider's session id.
    The ChatGPT backend sends it on every request.
  - `kilo`: OpenRouter's top-level `cache_control` for an `anthropic/`
    model.
  - Interactions, generateContent, `huggingface`, `ollama` and OpenCode's
    `chat_completions` route send nothing for it.
- `Stream(ctx, p, req)` streams from every provider: a provider's own
  `Streamer`, or `Generate`'s result as events. Only `EventDone` carries a
  `Response`, and it comes last; what a native stream yielded before a
  failure is not a response. Every built-in provider declares
  `NativeStreaming` `Supported`, except both OpenCode gateways, which
  declare `BestEffort`: their `google` route was measured on Google's own
  endpoint, and Zen's routes stand on Go's measurements. Each wire's
  decoder yields text deltas (a refusal's included) as `EventTextDelta`
  and, where the wire streams them, reasoning deltas as
  `EventReasoningDelta`: Responses' reasoning summary, Messages'
  `thinking_delta`, Chat Completions' `reasoning_content` or `reasoning`;
  generateContent and Interactions stream none. It yields each item as
  `EventItem`, and `EventDone` carries the `Response` that wire's `Decode`
  returns for the same answer. A call's arguments arrive only in its item
  (0031-MADR D2, E3). A Chat Completions stream asks for its usage
  (`stream_options.include_usage`, E5).
- `Response.Warnings` holds typed, non-fatal notes, nil when the request
  was sent as asked: `WarningForcedChoiceDegraded` when a forced tool choice
  went as `auto`; `WarningForcedChoiceLearned` when a model's refusal of
  forcing was learned (0034-MADR D7); `WarningDeferralEmulated` when
  deferred tools were not sent in the service's own form (0040-MADR D7);
  `WarningEffortDegraded` when an effort was not sent as asked, because the
  model's facts do not list it or the model refused it (0061-MADR D3;
  0062-MADR D3); and `WarningReasoningLearned` when a model's refusal of a
  thinking setting was learned, its `Refused` the value a caller passes back
  with `WithReasoningRefusals` (0062-MADR D6). A provider sets the last
  three on the request it builds (`wire.Call.Warnings`),
  and `Post` and the stream add it to the response. A stream carries them
  on its `EventDone` response. Errors stay errors.
- A model that refuses a forced tool choice is known by layer: the
  caller's `WithModelForcedChoice`, then the providers' dated table matched
  on a normalised id, then a refusal learned this process (0034-MADR D9).
  `ToolChoiceRequired` on such a model fails before the network, with
  `ErrUnsupported`, unless `WithForcedChoiceFallback(ForcedChoiceDegrade)`
  opts in to sending it as `auto` (D10). A forced named tool, and so
  `GenerateToolCall`, goes to such a model as structured outputs, with the
  tool's schema as `output_config.format`, its open objects closed in a
  copy, and comes back as its `FunctionCallItem`: not degraded (D8, Q9).
  A refusal from a model no layer knows is learned in `wire.Send` and
  `wire.StreamEventsRetry`, the openers beside `wire.Reauth`, through
  `messages.RefusalRetry`. The request is built again and sent once more for
  a named tool, or for `ToolChoiceRequired` under `ForcedChoiceDegrade`.
  Otherwise the refusal is returned as `ErrUnsupported`. A stream is resent
  only when it fails to open, before any event (D3).
- A request's reasoning effort is decided in order by what the model
  refused, the memo's and the caller's `WithReasoningRefusals`, then its
  facts, then the request, along `wire.Ladder` (0062-MADR D2, D3). A refusal
  of the effort sent is read by its wire's matcher, built from the refusals
  0062-PLAN Phase 1 recorded, learned by `wire.Learner` in `wire.Send` and
  `wire.StreamEventsRetry`, and the request built again, at most
  `wire.MaxSends` (7) sends in all, a forced-choice refusal counted in them
  (D4, D5). Ollama has no matcher, no refusal of an effort it can send
  having been recorded, and learns nothing; a refusal passed back still
  steers it (0062-MADR, amendment "Phase 5's Ollama cases").
- `Response.Usage` is decoded from every wire format that reports it. A
  total holds its part: `InputTokens` includes `CachedTokens`, and
  `OutputTokens` includes `ReasoningTokens`. Anthropic's cache reads and
  writes are added to the input, and Gemini's thoughts to the output.
  Anthropic's reasoning count is `output_tokens_details.thinking_tokens`,
  which a stream sends on its last `message_delta` (0037-MADR). Chat
  Completions reads the cache read from
  `prompt_tokens_details.cached_tokens`, `prompt_cache_hit_tokens`, a
  top-level `cached_tokens`, then a top-level `cache_read_input_tokens`,
  the first present.
  `CacheWriteTokens`, the part of the input written to the cache, does not
  overlap `CachedTokens` (0036-MADR):
  - **Messages:** `cache_creation_input_tokens`, with
    `cache_creation.ephemeral_1h_input_tokens` as `CacheWrite1hTokens`;
  - **Responses:** `input_tokens_details.cache_write_tokens`;
  - **Chat Completions:** `prompt_tokens_details.cache_write_tokens`, else
    Kilo's top-level `cache_creation_input_tokens`;
  - **generateContent** and **Interactions** report no write, so both
    counts stay zero there.
- `Option` configures construction. A provider package's `New` resolves its
  options with `ResolveOptions(id, opts)`, which refuses an option scoped to
  another provider (`ScopedOption`). `For(id, opts...)` scopes options to
  one provider: they apply after the rest when building `id`, and are
  skipped for any other, so one list can build every provider. The resolved
  `Settings` are read-only.
- `APIError` carries a `Kind`, one of the sentinels. `ErrContextOverflow`
  sits beneath `ErrInvalidRequest`; it is classified from the service's
  error type, or from a message table taken from pi's `overflow.ts`
  (`context_overflow.go`). An overflow whose completion alone fills the
  context is `ErrInvalidRequest`: shortening the input cannot help. A 403
  is `ErrNotPermitted`, a refusal of what the key may do, so it is neither
  renewed nor resent; every provider refuses a key with a 401 or a 400.
  Grok's 400 `invalid-argument` naming an API key is `ErrAuthFailure`. A
  failure inside a 200 with no known code is classified by its OpenAI- or
  Anthropic-style error type.
- `WithRetry(p, RetryPolicy{…})` retries what `Retryable` allows, waiting as
  long as the service asks with up to a tenth more, up to a cap. It is not a
  `Streamer`: a stream through it is buffered so that it can be retried, and
  a caller that wants text as it arrives streams from the provider itself
  (0031-MADR D6). A 429
  whose headers and body name no delay waits for the longest rate-limit
  reset header the service sent. A failure while reading a reply, a
  failure the service reports inside a stream (0053-MADR D3), or one
  after the request was written, is retried at most once per call
  (0021-MADR D1, 0028-MADR D-H1); one before the request was written, as
  often as the policy allows. A wait past the context's deadline returns the error at once.
- `WithRetryNotify(p, policy, notify)` is the same wrapper, which calls
  `notify` with a `RetryNotice` on the caller's goroutine, with its `ctx`,
  each time a wait is about to begin, and at no other time (0044-MADR).
- `WithStreamRetry(p, policy, notify)` is a `Streamer` over a provider that
  is one, with its `Capabilities`: a failure before the first event is
  retried as `WithRetryNotify` retries it, through the same decision, and
  after the first event every event passes through, a failure ending the
  stream with no `EventDone` and no retry. Over a provider that does not
  stream it is `WithRetryNotify`'s result (0051-MADR).
- `Registry` holds `Descriptor` and `Factory` pairs and refuses a duplicate
  id. There is no global registry.
- `llmprovider/llmtest` has `Run`, the conformance suite every built-in
  provider passes, and `Fake`, a scriptable provider for tests. A `Harness`
  may switch on five more checks, and every built-in provider's does:
  - `Fidelity`: a request's `Model`, `Instructions` and a tool's output
    reach the wire, and a failed result's output reaches it unchanged.
    With `SendsStrict` or `SendsIsError` set, the flag it names must
    change the request (0032-MADR D4);
  - `Garbled`: an undecodable 200 is `ErrIncomplete`, sent once through
    `WithRetry`;
  - `Truncated`, with `TruncatedReason`: a cut answer with a partial call
    is `ErrIncomplete`, with the reason the wire reports;
  - `StrictTools`: a call reply finishes `tool_calls`, with non-empty,
    valid JSON arguments;
  - `ReasoningCut`: an answer cut while the model was still reasoning, with
    no text or call, is `ErrIncomplete`, with the reason the wire reports.

  Two more fit a service that has the reply: `AuthFailure`, the service's
  own refusal of a key, for the renewal check (Gemini's 400), and
  `NotPermitted`, a refusal of a request the key may not make, which must
  be `ErrNotPermitted` (Kilo, OpenCode, OpenAI). `StreamText` and
  `StreamFailure` switch on the native stream check for a `Streamer`: one
  `EventDone`, last, whose `Response` is `Generate`'s and holds the text the
  deltas carried; a failure that ends the stream with no `EventDone`; and
  every reply body closed, after a stream that completes, stops early or
  fails (0031-MADR D7). They also switch on the check through
  `WithStreamRetry`: a 503 before the first event retried, with one
  notice; a failure after a delta not; the stream live, shown by holding
  back the part of `StreamText` after the one that gives the provider's
  own first delta; and the provider's own events, in order (0051-MADR D5).
  `StreamText` needs one delta, and two are preferred, since one cannot
  show that none is merged; a check that cannot be made, such as liveness
  for a script in one part, goes to the test log. Every built-in
  provider's harness sets them, in its wire's format, its text in two
  deltas, which each package's `TestStreamTextDeltas_0051` holds. A
  harness that sets none passes as before. Every harness also gets checks with no field: each HTTP
  failure an `*APIError`, a 429's `Retry-After` in `RetryAfter`, a token
  that names its own `Header` sent there, and, for a provider that lists,
  the listing's identity and cancellation (0026-MADR F57).

## Providers and items

- **Ten provider ids,** `ProviderID` constants in `llmprovider`: `openai`
  (API key; a ChatGPT subscription through the Codex backend; or a ChatGPT
  plan session on the platform), `claude`,
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
  them. A fifth, `ToolLoadItem`, is only ever in a request's `Input`: it
  loads one deferred tool (`Tool.Deferred`) from the result of the call it
  names, and no response holds one (0040-MADR D1, A5).
  - A call's `Arguments` are the JSON the service sent, compacted, never
    decoded and re-encoded; no arguments are `"{}"`.
  - A `ReasoningItem`'s `Format` names the wire that produced it. Its
    `Signature` and `Encrypted` are replayed only to that wire, or from an
    item with no `Format` (0021-MADR D5).
  - A Responses request sent with `store: false` asks for encrypted
    reasoning, so a tool loop keeps it (0021-MADR D4). Kilo's
    `reasoning_details` are kept and replayed as they came (0021-MADR W9).
- **Request bodies:** each shared wire encodes its messages as typed
  structs, with fields in the wire's key order and optional ones omitted
  when unset. Messages, Responses and generateContent use one union struct
  per item, block or part, whose strings are held in a per-request
  `wire.Arena`. The body around the messages is a map.
- **Answers:** every wire maps its finish reason the same way. A value with
  no constant is kept as sent, and a reply with a call finishes
  `tool_calls`. An answer with nothing usable is `ErrEmptyAnswer`, which
  also matches `ErrIncomplete`, with the service's reason (`content_filter`,
  a refusal) in `APIError.Reason`; so is an answer cut while the model was
  still reasoning (0061-MADR D1). A call id another
  wire made, such as Gemini's `name#index`, goes to the Messages wire in a
  form Anthropic accepts, unique in the request, with each result paired to
  its call.
- **Construction:** `providers.New(id, opts...)`, or the provider package's
  own `New` (`opencode.NewZen` and `NewGo`); every provider is in its own
  package. Options are `Option` values: the common ones in `llmprovider`
  (`WithBaseURL`, `WithHTTPClient`, `WithReasoning`, …), and a provider
  package's own, scoped to it. `WithRetry` retries by the error's kind.
- **Transport:** without `WithHTTPClient`, each provider instance gets its
  own client from `internal/transport`'s `DefaultClient`, over the one
  transport the process keeps for each transport configuration, so
  instances share connections, up to 16 idle per host. Closing one
  client's idle connections leaves the shared ones open. The bounds are
  30 s to connect, 300 s to the first byte, no total timeout, HTTP/2 kept,
  and `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY` honoured. `catalog`'s own
  calls use one such client. Every generation goes through `internal/wire`'s
  `Post`, which reads the reply under a 300 s idle limit and, unless it is
  an event stream, a 16 MiB limit, whatever client the caller gave
  (0021-MADR D2). Auth requests carry their own 30 s bound. It carries the provider's requests, its listing
  and probes, and its OAuth session's refreshes when the session has no
  client of its own.
- **Errors:** one structured error, `*APIError`, whose `Kind` is a sentinel
  such as `ErrRateLimited`, `ErrQuotaExhausted`, `ErrAuthFailure` or
  `ErrIncomplete`; a rate limit carries its `RetryAfter`, and a cut-short
  response its `Reason`. An answer with nothing usable is
  `ErrEmptyAnswer`, beneath `ErrIncomplete`; a request that could not be
  sent for want of a network, a name that did not resolve or a dial timed
  out, refused or unreachable, is `ErrOffline`, beneath
  `ErrProviderUnavailable`, on every send path: generation, listing,
  refresh and sign-in (`transport.Offline`; 0061-MADR D1, D2).
  `Retryable()` says whether to try again. Every
  sentinel's message starts `llmprovider:`; a wrapped error keeps its
  package's or provider's prefix, such as `oauth:`. Error bodies pass through
  `redact.String`: the first 2 KiB of a message, cut back to a separator
  so no secret is split, control characters removed, is redacted, and 512
  bytes of it kept; a service's error code is stripped the same way and kept
  to 128 bytes. Each redaction pattern runs
  only when the text holds one of its literal anchors. Redaction masks each
  vendor's published key and token shapes, reads `ſ` and the Kelvin sign
  as `s` and `k`, and keeps a `code` value made of letter-only words, such
  as `invalid-argument`.

## Credentials

- **`TokenSource`** returns a `Token` for each request. `StaticToken` wraps an
  API key. `openai.New` takes a source with `WithTokenSource`; a ChatGPT
  session selects the ChatGPT backend, and a plan session the platform with
  OpenAI's rules for plan usage. `grok.New` takes any source: a key,
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
- **The ChatGPT plan session** (`OAuthSession.PlanUsage`, 0043-MADR) is
  `openai`'s concern too. Its requests go to `api.openai.com/v1` with none
  of the Codex backend's headers: always streamed, with `store` false and
  no output limit. It has no continuation, appends deferred tools, and
  refuses a system message in the input before the network. It lists the
  account's `models` array and is never probed. Its four
  `subscription_sharing_*` codes are classified by code, in a reply or a
  stream.
- **After a 401** a provider invalidates any `InvalidatingSource` and retries
  once: a `CommandToken` reruns its command, and an `OAuthSession` refreshes.
  A source that is also a `TokenInvalidator`, as both are, is told which
  token was refused, and ignores a 401 on a token it has already replaced,
  so concurrent 401s renew it once (0021-MADR D5). Every built-in provider
  does it through `internal/wire`'s `Reauth`, and `llmtest` checks it
  (0020-MADR F2).
- **Sessions are `auth`'s.** A provider gives a session with no HTTP client
  its own, so refreshes share the provider's transport: a caller's
  `WithHTTPClient` client through `OAuthSession.UseHTTPClient`, its default
  through `UseDefaultHTTPClient`. A caller's client replaces a default,
  never another caller's, and only a caller's `WithLogger` logger is shared
  (`Settings.HTTPClientGiven` and `LoggerGiven`; 0026-MADR F37).
- **`OAuthSession`** (`auth`) is a refreshable `TokenSource` for ChatGPT and
  Grok.
  - **Creating one:**
    - `LoginBrowserOAuth` uses PKCE on a loopback redirect.
    - `LoginChatGPTPlan` is OpenAI's Sign in with ChatGPT for agents. It
      runs the same loopback flow under the consumer's app name, through
      endpoints read from discovery that must be on the issuer's origin, and
      keeps the client OpenAI issued, the resource, the scopes, the ID token
      and the host id. Its session refreshes and revokes with forms and the
      resource.
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
  `Options.Registry`, `providers.Default()` when nil. It offers OpenAI's
  ChatGPT plan sign-in (`AuthChatGPTPlan`) only with `Options.AgentName`
  and a `TokenStore`.

## Discovery and ranking

- `catalog.List(ctx, id, src, opts...)` lists a provider's models within a
  10 s bound; a `Catalog` carries the recommended list, the full usable list
  and, on failure, `Err`. The recommended list is the band, the profile's
  six, then every other usable id without a caveat, newest first by
  release date, then by id; `catalog.WithLimit(n)` cuts it to `n`, and
  `WithLimit(catalog.MaxListed)` to the band. A static fallback is cut in
  both views. Its options are `llmprovider`'s common ones, with
  `catalog.WithProfile`, `catalog.WithWorkload`, `catalog.WithLimit`,
  `catalog.WithLastGood` and `catalog.WithKiloOrganization`. It lists the ten
  built-in ids. For any other id it returns an error matching
  `ErrUnsupported`, before it reads the options.
- **`Catalog.Menu`** is every usable model as a `MenuEntry`: its id, its
  facts (`ModelInfo`, merged as `CachedFacts` merges them), its rank in the
  band (1 to 6, or 0) and its caveats (0041-MADR D3, A4):
  - the band by rank, then the uncaveated, then the caveated, each newest
    first, then by id; `Recommended` is the menu less the caveated (Ollama keeps its
    tool-less models: 0042-MADR D2);
  - a `Caveat` sinks a model, and never drops it: `CaveatDeprecated` (a
    deprecated or retired status), `CaveatNoTools`, `CaveatNoTextOutput`,
    `CaveatExpiring` (a retirement date within 30 days) and
    `CaveatUnavailable` (measured to fail at request);
  - the band is ranked from the uncaveated models only; the profile's
    exclusions and Together's ids models.dev does not list are never in
    it, and follow it;
  - a static catalog's menu is the static ids, in order, each ranked by
    its place;
  - Ollama's keeps `/api/tags` order, with no band. `List` asks
    `/api/show` for each installed model not cached, four at a time, and
    caches the replies, each for ten minutes from when it was fetched, and
    asks again at once when its model's digest in `/api/tags` changes
    (0053-MADR D5); a model it marks as taking no tools follows the
    others, and stays in `Recommended` (0042-MADR D2). A model Ollama says writes no text,
    such as an embedding model, is not listed (0042-MADR D1).
- `catalog.Search` matches a query against a list, highest tier first and
  in the list's order within a tier: the id exactly, as a prefix or a
  substring, then without separators (`gpt41` finds `gpt-4.1`), then a
  label's display name, then token prefixes. A subsequence of the id is
  tried only when nothing else matched and the query has three or more
  characters. A glob scores an id match above a label-only one. The static
  catalogs are the fallback. `catalog.Static(provider)` returns a copy, and
  `llmprovider.ProviderEnvVars()` a copy of the variable names.
  `catalog.Label` is per provider, so an id two providers offer can be
  labelled differently.
- **Two guards keep the static catalogs current** (0033-MADR D4):
  - a unit test's table of retired ids, taken from the providers'
    deprecations pages;
  - a live test, run before each release, which fails on an id the
    provider's listing lacks or marks stale.
  `catalog.Rank(provider, model)` scores a model by the provider's own
  ranking.
- A provider's `ListModels`, through `llmprovider.ModelLister`, returns the
  listing, cut by `catalog.WithLimit` given to `New`, its static fallback
  included. By default, OpenAI (API key), Claude, Gemini, Grok and Ollama
  also send one billed generation to each of the first `catalog.MaxListed`
  candidates and drop those that do not answer; the later ones are returned
  unprobed. The models that answered are cached in
  memory, the process's or a `catalog.Cache`'s, for 10 minutes, keyed by provider, base URL and the SHA-256 of
  the credential, so a listing within that time sends no probe, and
  listings that miss the cache at once share one run of probes (0053-MADR
  D6).
  `WithModelProbes(false)` turns that off. `ModelProbesFromEnv()` reads
  `LLMPROVIDER_PROBES` (`true` or `false`) for a caller who passes it; the
  package reads the variable nowhere else.
- `catalog.Profile` (`ProfileUtility`, `ProfileCapable`, `ProfileAgent`),
  passed with `catalog.WithProfile`, ranks every provider's catalog,
  using models.dev-format metadata from `https://models.opencode.ai/api.json`.
  `WithModelMetadataURL` names another document, and `WithoutModelMetadata`
  turns the fetch off. `catalog.OptionsFromEnv()` sets either from
  `LLMPROVIDER_MODELS_METADATA_URL` and `LLMPROVIDER_DISABLE_MODELS_METADATA`
  for a caller who passes its result.
- **`ProfileAgent`** (0041-MADR D5) is for long agent sessions with prompt
  caching on. Its band holds `ProfileCapable`'s six, reordered by signal,
  bench, preferred index, release quarter (90 days), then the cheaper
  cached turn, `(1 − o)(h × read + (1 − h) × write) + o × output`, an
  unknown price last, then id. `read` is the cache read price, else the
  canonical model's, else the input price; `write` is the cache write
  price, else the input price. `h` and `o` default to 0.90 and 0.01;
  `catalog.WithWorkload(h, o)` sets them, each within (0, 1), and
  `catalog.ObservedWorkload(usages...)` measures them from a caller's own
  responses, `ok` false when there is nothing to measure or a share is
  0 or 1. Its reasoning effort is the provider's default, as
  `ProfileCapable`'s is.
- **The band** (0041-MADR D4):
  - a model is small when its 3:1 blended price, `(3 × input + output) / 4`,
    is $1 or less per million tokens; the name decides only with no
    positive price;
  - a model that takes no tools, or writes no text, is never recommended,
    and stays in `Usable`;
  - on OpenAI, Claude, Gemini and Grok, the per-group cap counts models.dev's
    `family`; the gateways keep the vendor prefix.
- **No ambient state.** Library code reads the environment only in
  exported `…FromEnv` helpers a caller opts into (`ModelProbesFromEnv`,
  `catalog.OptionsFromEnv`, `auth.GrokFlowFromEnv`). The default transport
  also reads the proxy settings. Nothing logs to the global logger. The test
  in `internal/ambientcheck` fails on any other read.

## The wizard

`ConfigureLLM(ctx, p, opts)` walks provider → base URL → credential → model →
fallbacks and returns a `Result`; it writes no configuration. On a `ctx`
that has already ended it returns that error before asking anything, as
`ResolveCredential` does (0059-MADR W8). It renders
nothing itself: everything goes through the `Prompter` interface (`Select`,
`MultiSelect`, `Confirm`, `Input`, `Secret`, `Notify`). `TextPrompter`
implements it over a terminal with `golang.org/x/term`. OAuth and other
non-API-key methods are offered only when `Options.TokenStore` is set.

- **`ResolveCredential(ctx, p, opts, id)`** runs the same steps up to the
  model and stops there, for a consumer with its own model picker
  (0056-MADR D1). It asks the provider only when `id` is empty, and no
  model, fallback or model id. It returns the `Result`, with no `Model`, and
  the catalog its listing checked the credential with: the refusal loop, the
  ChatGPT plan sign-in and every other method are `ConfigureLLM`'s own.

- **One copy of a session.** A sign-in saves its session to
  `Options.TokenStore` and returns a `CredOAuth` `Result` with no token; the
  consumer loads the session from the store. Keeping an existing session
  reads it there too (0016-MADR D11, A7, A10).
- **When a session is saved.** A new session is saved once the model
  listing has not refused it, after any refresh the listing left running,
  so a refused sign-in leaves the store as it was and the store gets
  rotated tokens, never spent ones. A refresh the issuer refuses saves
  nothing and ends the run (0059-MADR W2).
- **Keeping a saved session.** With `Options.Existing` naming one, the
  session is checked before it is offered. One that cannot be used, such
  as an expired one with no refresh token, is not offered: a notice says
  why, and the sign-in menu follows (0059-MADR W6).
- **Kilo.** A Kilo device login saves its token the same way, offers the
  account's organizations, and puts the choice in `Result.Organization`.
  With `Options.KiloDeviceKey` it returns the token as a `CredAPIKey`
  instead, needing no store, saving nothing and asking no organization: one
  copy, in the `Result` (0056-MADR D2).
- **A device sign-in's code.** It is shown as a notice, or, with
  `Options.ShowDeviceCode`, given to the caller as a `DeviceCode`: the
  provider, URL, code and expiry, for a caller that reports it as data. An
  error from the hook ends the sign-in before any poll (0057-MADR D1).
- **`Options.SkipMethods`** leaves sign-in methods out of every menu, and
  a provider it leaves with none is an error. **`Options.NoEndpointPrompt`**
  asks no endpoint: `Existing`'s, else the default, is checked and used, and
  a failure is returned (0056-MADR D4). A plain-`http` URL to a remote host
  is used there with a warning notice, since nobody is asked (0059-MADR W9).
  **`Options.NoPastePrompt`**, for a
  caller with no terminal, starts no paste prompt in a browser sign-in: the
  URL is shown and opened, and the loopback alone completes it, the ChatGPT
  plan's included (0057-MADR D2).
- **The paste prompt.** A browser sign-in otherwise races the loopback
  against a paste prompt. An empty paste is asked again; when the loopback
  wins, a read still waiting is finished by one Enter, which the wizard asks
  for, before any other prompt (0008-MADR's amendment; 0057-MADR D4). A
  `Prompter` that answers at once with nothing is asked three times, each
  answer within 50 ms; then a notice says so, and the wizard waits for the
  browser or the sign-in's limit, while a person's slower Enter is still
  asked again (0059-MADR W5). A
  `Prompter` that can give up a read implements **`InputerContext`**: the
  paste prompt, and no other, goes to its `InputContext`, whose context the
  wizard ends when the loopback wins, and no Enter is asked for unless the
  read is still running after a short grace. `TextPrompter` does not
  implement it, since a read of stdin cannot be given up, and keeps the
  Enter (0057-MADR D3).
- **`Logout(ctx, p, opts, id)`** revokes the stored session (OpenAI and
  Grok), reports a failure, and deletes it.
- **`Result`** prints `[redacted]` for its API key under `fmt` and `slog`.
  Its JSON keeps the key, for the consumer to persist (0016-MADR A9).
- **What it accepts.** A remote provider's base URL must be `http` or
  `https` with a host, and carry no userinfo, query or fragment; plain
  `http` to a host that is not loopback is used only once confirmed, except
  under `NoEndpointPrompt`. A
  pasted OpenAI credential that is another vendor's key, or not a JWT, or
  an expired one, is refused before anything is saved. A typed key's shape
  is not checked before the listing, which decides: only a key that is
  plainly another vendor's is named then, with the listing's check promised
  only under `Options.Discover` (0059-MADR S1, W1, W3).
- **A refused key.** When the listing refuses a typed key, the wizard asks
  for another, up to three keys, then returns an error. Each refusal of a
  typed key without the provider's usual shape names that shape, the last
  included, and never the key (0059-MADR S1). A refused key from
  the environment or `Options.Existing`, or a refused session or CLI login,
  is an error naming that source. Any other listing failure falls back to
  the built-in catalog, with a warning. Every notice that reports a service's
  error, the refusal, the three listing warnings, Kilo's profile and a
  failed revocation, shows its class and HTTP status, never its text: a
  service can echo a key it has masked itself, which redaction leaves as it
  is. A returned error keeps its text, for `errors.Is` (0056-MADR D3).
- **`TextPrompter`** strips control characters from what `Notify` writes.
  Its masked entry redraws only when its input has caught up, pads rather
  than erasing the line, ignores a lone Escape, clears on Ctrl-U, and
  cancels on Ctrl-C with an error matching `context.Canceled`. A cursor or
  function key, or a paste's bracket marker, adds nothing, even when its
  escape sequence is split across reads (0059-MADR W4).
- **Fallbacks.** The fallback round preselects the saved fallbacks it
  offers. A saved one that the menu does not recommend is offered too,
  after the recommendations, marked "saved", so a blank answer keeps it
  (0020-MADR F50; 0059-MADR W7).

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
  `api-check`, `generate-check`, `records-check`, `tables-check`,
  `fixtures-check`, `changelog-check`, `docs-check`, `gate-selftest`,
  `tools-test`, `help`.
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
  `unittest`) copies the tree, plants breaches of each gate (dep-check,
  generate-check, parity-check, coverage-check, api-check, records-check,
  tables-check, fixtures-check, docs-check, changelog-check), and requires
  each to fail, then every gate to pass on a clean copy.
- **G-wire** is `TestWireGoldens` in each provider package, part of
  `go test`. Through `llmprovider/internal/wirecase` it drives 18 provider
  and gateway-route cases through seven scenarios (text, forced tool,
  thinking, thinking tool, items, continuation where the provider continues,
  listing): 114 golden files. It runs them against `internal/wiretest`, and
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
- **`tools/`** (0035-MADR) holds the maintainer tools, in Python on the
  standard library. `tools/gate.py` runs every local check on a scratch
  clone of the working tree, once the load average is below the CPU count:
  the checks above, `go vet` for three platforms with and without
  `live_gateways`, the suite shuffled and again with every package run
  twice in one process (`test-repeat`, which fails on a test that reads what
  its own first run left in a process-wide cache; 0058-MADR's amendment of
  2026-10-24), markdownlint, the link check, the identifier scan, and
  `tools-test`. `tools/cachecensus.py` reports the tests that write the
  process's catalog caches, `--any-url` at any URL and with the keys two
  tests share (0058-MADR D5, D7). `make tools-test`
  (`tools/test_tools.py`) proves each tool fails on a planted input. CI
  runs neither. 0047-MADR added `records.py` (a record's status and index
  row), `recordlint.py` (an edit's new markdownlint findings) and
  `claims.py --plants`, and the CI gate
  `fixtures-check` (`scripts/check_fixtures.py`).
- **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with
  the Go version read from `go.mod`, on each push to `main`, each pull
  request, and weekly: `go test -shuffle=on`. On Linux it also runs:
  - `go test -race -shuffle=on`;
  - `govulncheck`, pinned in the workflow;
  - `go vet`, `gofmt`, `go mod tidy -diff` and `make lint`;
  - `go vet -tags live_gateways`;
  - `make parity-check dep-check coverage-check api-check generate-check
    records-check tables-check fixtures-check changelog-check docs-check
    gate-selftest`.

  It checks out the full history, so that `api-check` and `changelog-check`
  see the tags. A job stops after 30 minutes, and a newer push to a pull
  request cancels its older run. On a `v*` tag, once every `validate` job
  has passed, the `release` job, the only one that may write, creates the
  tag's GitHub Release with its `CHANGELOG.md` section as the notes, each
  relative link pointed at its file at the tag (0048-MADR D5; 0048-PLAN
  Phase 5 step 7). Only the five newest tags keep a Release: each release
  deletes the sixth-newest's, leaving its tag and section (0055-MADR).

  A push to `main` or a pull request that changes only Markdown skips it
  (`paths-ignore: ['**.md']`). **Docs** (`.github/workflows/docs.yml`) runs
  instead, on one Linux runner, with `ci.yml`'s checkout of the full
  history, its Go version and its pinned golangci-lint, which the
  self-test's precheck test runs: `make parity-check records-check
  tables-check changelog-check gate-selftest`, the gates that read Markdown
  and the gates' self-test, whose plants replace exact strings in the docs.
  A test in the self-test fails when `docs.yml` lacks a tool `ci.yml`
  installs before its own self-test. A
  change to anything else runs CI, and Docs too when it also changes
  Markdown or `docs.yml`. GitHub does not filter a tag push, and the weekly
  run is not a push, so both always run CI in full, and a Release still
  follows a full `validate`
  ([0052-MADR](decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md)).

## What is not here

- **Streamed reasoning on the Gemini wires.** generateContent and
  Interactions send no thought-summary delta in their streams, as measured,
  so their reasoning arrives only as an item.
- **`llmprovider/x/`.** No experimental API exists.
