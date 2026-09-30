---
date: 2026-09-29
subject: the imported llmprovider and wizard API measured against the owner's requirements for a canonical, extensible SDK
examines: "this repository's Phase 4 dry run of 0002-PLAN (mcplib 4e1f9a5 re-homed, identity and env renamed); consumers mcp-server-magicdev, mcp-server-magictools, prepare-commit-msg; the planned consumer pi-go"
---
# SDK API Surface Assessment: The Imported `mcplib` API Against the SDK Requirements

This report records findings. It decides nothing. The decisions it bears on
belong to
[0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md),
and to the amendment of
[0002-MADR-migrate-llmprovider-from-mcplib.md](../decisions/0002-MADR-migrate-llmprovider-from-mcplib.md)
of the same date.

Evidence is marked **(M)** measured, **(R)** read in the source or records,
or **(S)** shown in a scratch experiment.

## Summary

The owner restated the SDK's purpose on 2026-09-29:

> go-llmprovider-sdk does not have to match what is implemented in mcplib it
> needs to provide the functionality but enhance, extend, modularize, and
> improve on how that framework is constructed. […] it needs to provide
> current functional parity but the sdk will be extended, expanded, improved
> and enhanced and we need to code to future-proof this by being canonical,
> defining standards, and using a consistent API as we move forward.

The requirement changes what the migration must preserve. The code at `F`
works: 473 test functions pass, and the providers carry measured wire
knowledge that must not be lost. Its **API shape**, however, does not meet
the new requirements:

- **Features are expressed as method names, not request fields.** Every
  new request option multiplies the method count and the interface count
  (F1).
- **Everything is one flat package.** Provider-specific settings leak into
  shared configuration, and every consumer imports every provider (F2, F3).
- **The same concept has several spellings.** There are 13 constructors,
  three retry helpers, four listing functions, three error types and untyped
  string identifiers (F4, F5, F6).
- **There is global state.** The package has mutable exported catalogs, and
  it reads environment variables and logs through the process-global logger
  (F7).
- **Nothing makes a new provider consistent.** There is no registry that
  third parties can extend, and no conformance suite that every provider
  must pass (F8).
- **It lacks what the planned consumer needs.** `pi-go` needs streaming
  events, token usage and several tools at once. The API has none of these,
  and the method-per-feature shape cannot absorb them without a further
  multiplication (F9).

The three current consumers use a small part of the surface (F10), so a
redesign before `v1.0.0` breaks little that is in use. After `v1.0.0` it
would take a v2 module (F11).

## Scope and method

- **The surface.** `go doc -all` of `llmprovider` and `wizard` from the
  scratch dry run of 0002-PLAN Phase 4 (`SCRATCH/sdk-p4`). That is `mcplib`
  `4e1f9a5` with its imports, identity and environment names re-homed, and
  no API change.
- **Counts.** `go doc` output and `grep` over it.
- **Consumer use.** Every `llmprovider.X` and `wizard.X` selector in the Go
  files of `mcp-server-magicdev`, `mcp-server-magictools` and
  `prepare-commit-msg`, at their working trees under the fleet root.
- **Future use.** `pi-go`'s `0001-REPORT-go-port-feasibility.md` and
  `0002-PLAN-cli-acp-headless-mcp-v1.md`, read, not run.

## The surface today

| Element | Count | Source |
|---|---|---|
| Source files and lines (`llmprovider`, non-test) | 43 files, 9,521 lines | (M) |
| Test functions (`llmprovider`) | 473 | (M) |
| Exported package-level functions | 53, of which 13 constructors and 7 `Rank*Model` | (M) |
| Exported interfaces | 13 | (M) |
| Methods on the 8 provider types | 84, of which 64 are `Generate*` variants | (M) |
| `ProviderConfig` fields documented "Ignored by all other providers" | 7 | (M) |
| Coverage at `F` | `llmprovider` 88.0 %, `wizard` 82.6 %, redaction 85.7 % (`mcplib/logging`) | (M) |

## Findings

### F1 — A feature is a method, so the API grows combinatorially (M, R)

Each provider type implements eight generation methods: {text, items} ×
{plain, tool} × {plain, thinking}. Examples are `Generate`,
`GenerateWithToolThinking` and `GenerateItemsWithToolThinking`. Each pair of
methods has its own optional interface:

- `ThinkingProvider`, `ToolProvider`, `ThinkingToolProvider`;
- `ItemProvider`, `ItemToolProvider`, `ItemThinkingProvider`,
  `ItemThinkingToolProvider`;
- `Continuer`.

The retry helpers follow the same pattern: `GenerateWithRetry`,
`GenerateThinkingWithRetry` and `GenerateItemsWithRetry`.

The next features a consumer needs each add a dimension:

- streaming;
- more than one tool;
- a tool choice other than "forced";
- structured output;
- images.

Streaming alone would double the 64 methods and the eight generation
interfaces. The request is also limited by its shape. `GenerateWithTool`
takes exactly one `Tool` and always forces it, so "offer three tools and let
the model choose" cannot be expressed.

**Bears on:** the generation contract (0015-MADR D3).

### F2 — Provider-specific settings live in the shared configuration (M, R)

`ProviderConfig` and `ApplyOptions` are exported. Seven of its documented
fields apply to some providers and are "Ignored by all other providers":

- `OpencodeRoute`;
- the three Kilo settings;
- `Store`, which OpenAI, Grok and Gemini honour.

A Kilo option passed to Claude is silently dropped. Every new provider
feature adds a field that every other provider must ignore.

**Bears on:** options and provider packages (0015-MADR D2, D5).

### F3 — One flat package holds six concerns (M, R)

`llmprovider` holds all of these together:

- the generation types;
- eight provider implementations over four wire formats (Responses, Chat
  Completions, Messages, Gemini Interactions), plus OpenCode's Google route;
- OAuth login, refresh, revocation and token stores;
- vendor-CLI sessions;
- model catalogs, ranking, search and metadata;
- configuration descriptors.

The largest files are `discovery.go` (988 lines), `models_catalog.go` (683)
and `oauth_loopback.go` (657). A consumer that needs one provider compiles
all eight, plus the OAuth servers. The package boundary gives no signal
about which types are stable contracts and which are implementation.

**Bears on:** the module layout (0015-MADR D2).

### F4 — The same concept has several spellings (M, R)

- **Constructors.** There are 13, with four signatures:
  - `NewGemini(ctx, apiKey, model, …)` alone takes a context;
  - `NewOpencode(gateway, apiKey, model, …)` takes a gateway first;
  - `NewOpenAIWithSource` exists for OpenAI only;
  - `NewProvider` and `NewProviderWithSource` duplicate each other with the
    credential as the only difference.
- **Listing.** `ListAvailableModels`, `ListAvailableModelsWithSource`,
  `ListModelCatalog` and `ListModelCatalogWithSource`, plus
  `ModelDiscoverer.DiscoverModels`, which has a different contract: it may
  send a generation health probe.
- **Ranking.** Seven exported `Rank*Model(string) int` functions, one per
  provider.
- **Unused parameters.** `ModelLabel(provider, model)` documents its
  `provider` parameter as unused.

**Bears on:** construction and naming standards (0015-MADR D5, D6).

### F5 — Identifiers are untyped strings (R)

Provider ids (`ProviderOpenAI = "openai"`), `MessageItem.Role`, and the
reasoning effort (`"low"|"medium"|"high"`, with `"xhigh"` accepted and
clamped for Ollama) are all plain `string`. `Response.FinishReason` is a
string, "e.g. "stop" or "length"". A misspelt effort or role compiles and
fails, or is silently ignored, at the service.

**Bears on:** typed identifiers (0015-MADR D6).

### F6 — There are three error types (R)

A caller classifies a failure through three error types:

- `*APIError` (status, type, message, `RetryAfter`, `Terminal`);
- `*RateLimitError` (`RetryAfter`, status, provider, message);
- `*IncompleteError` (reason).

It also has seven sentinels to choose from. `APIError` alone unwraps to two
sentinels "so every existing errors.Is check still matches", which carries
compatibility with an earlier `mcplib` release forward. `ErrInvalidProvider`
is the one sentinel without the `llm:` prefix.

**Bears on:** the error model (0015-MADR D7).

### F7 — There is global and ambient state (M, R)

- **Mutable exported catalogs.** Seven catalog variables
  (`StaticGemini` … `StaticGrok`) and the `ProviderEnvVars` map are
  exported and mutable. `StaticModels(provider)` returns a copy, but the
  variables can be written directly.
- **Environment variables.** The library reads four:
  - `LLMPROVIDER_MODELS_METADATA_URL` and
    `LLMPROVIDER_DISABLE_MODELS_METADATA` in `model_metadata.go:148,157`;
  - `GROK_OAUTH2_ISSUER` and `GROK_OAUTH2_CLIENT_ID` in
    `oauth_loopback.go:161,167`.

  A test must set process environment (`wizard/main_test.go`) to keep unit
  tests off the network.
- **Logging.** Through the process-global `slog` default logger, in
  `provider.go:215`, `http_helpers.go:19` and `api_error.go:112`. A
  consumer cannot route or silence it per provider.
- **Stale doc comments.** `OAuthSession` and `TokenStore` still describe
  themselves as "a minimal stub" for "Phase 2" and "Phase 3" of a completed
  `mcplib` plan.

**Bears on:** state, environment and observability standards (0015-MADR
D8, D9).

### F8 — Nothing makes the next provider consistent (R)

- **What adding a provider means today:** a constant, a constructor, a
  `NewProvider` switch case, a catalog variable, a `Rank*` function and a
  descriptor entry. `TestDescriptors_CoverEveryRegisteredProvider` checks the
  descriptor list against the switch.
- **Which behaviours are shared:**
  - tool forcing or offering;
  - reasoning degradation;
  - continuation refusal;
  - error classification;
  - identity headers.

  Each provider's own tests assert these, and no single suite runs all of
  them against every provider.
- **Outside this package:** a third-party provider cannot be registered,
  offered by `wizard`, or checked against the same contract.

**Bears on:** the registry and conformance suite (0015-MADR D10, D11).

### F9 — The planned consumer needs what the API cannot express (R, M)

`pi-go`'s `0002-PLAN-cli-acp-headless-mcp-v1.md` Phase 3 names this
repository as its provider layer (its D10). The same phase needs:

- message chunks and tool calls streamed to the client;
- cancellation of an in-flight request;
- after each prompt, token usage and the context-window size.

Its feasibility report lists `pi-ai`'s streaming, several tools per turn and
image models among what a Go port must eventually cover. Measured against
this code:

- **No streaming.** The one event-stream path, the ChatGPT backend
  (`openai.go:137,193`), buffers the whole stream before returning. No wire
  decoder reads token usage.
- **One tool, always forced.** `ModelCatalog` has no context-window field.
- **Cancellation already works.** It rides on the request context and is
  already sound.

**Bears on:** streaming, usage and capability design (0015-MADR D3, D4).

### F10 — The consumers use a small, stable subset (M)

Selectors used by the three consumers:

| Area | Used |
|---|---|
| Construction | `NewProvider`, `NewProviderWithSource`, `NewClaude`, `NewOpenAI`; options `WithHTTPClient`, `WithBaseURL`, `WithMaxTokens`, `WithReasoningEffort`, `WithThinkingBudget` |
| Generation | `Provider`, `ThinkingProvider`, `ToolProvider`, `ThinkingToolProvider`, `Tool`, `GenerateWithRetry` |
| Catalog | `Descriptors`, `DescriptorFor`, `StaticModels`, `ListAvailableModels`, `ModelDiscoverer`, `ProviderEnvVars`, the provider constants |
| Auth | `OAuthSession`, `TokenSource`, `TokenStore`, `FileTokenStore`, `NewFileTokenStore`, `VendorCLISession`, `DefaultOpenAIIssuer` |
| Errors | `ErrAuthFailure`, `ErrRateLimited`, `ErrProviderUnavailable`, `ValidateOllamaURL` |
| `wizard` | `ConfigureLLM`, `Options`, `Result`, `Choice`, `Prompter`, `TextPrompter`, `NewTextPrompter`, `Level*`, `Cred*` |

The consumers use none of these:

- the item API, `Continue` or `Response`;
- `SearchModels`, `ModelCatalog` or any `Rank*`;
- the OpenCode or Kilo options.

Each area in the table has a direct equivalent under any of the layouts
0015-MADR considers, so the consumers' migration is mechanical.

**Bears on:** sequencing and the consumer companions (0015-MADR D14;
0002-MADR amendment 2026-09-29, second).

### F11 — The first tag fixes the compatibility promise, and it must be `v1.x` (R)

0002-MADR §8 makes the first tag `v1.0.0`, because the ChatGPT backend
reads the SDK's version as `client_version`. A value below `0.155.0` hides
models (0001-REPORT F4). A `v0.x` series is therefore not available for
iterating on the API. From `v1.0.0` on, semantic versioning forbids breaking
changes until a `/v2` module. Two consequences follow:

- **Before the tag.** A redesign made before `v1.0.0` costs consumers one
  migration.
- **After the tag.** A redesign made later costs them two, plus a second
  module path.

Pre-release tags are an option: `chatgptVersionRE` in `discovery.go`
accepts `v1.0.0-rc.1` and sends `1.0.0`, so release candidates can be
published for consumers to trial without committing to v1.

**Bears on:** sequencing (0015-MADR D14).

### F12 — The wire knowledge sits in functions that can be kept as they are (R)

The measured behaviour lives in unexported code:

- request builders and decoders;
- route tables;
- quota classification;
- identity headers;
- the ChatGPT store-false rule;
- Gemini `thoughtSignature` replay;
- Kilo data-collection denial.

It is covered by the 473 tests. Most of those tests call it through the
exported methods of F1. A restructure that moves unexported code into
provider and wire packages, and re-exposes it through one request type,
keeps that knowledge whole, provided the tests move with it. The request
each provider puts on the wire is observable in tests through
`httptest.Server`, and many tests already capture it.

**Bears on:** the functional-parity gate (0015-MADR D12).

## Decisions a MADR here needs to make

1. Whether `v1.0.0` ships the `mcplib` API or a redesigned one (F11).
2. Package layout and what is internal (F2, F3).
3. The generation contract, streaming and capabilities (F1, F9).
4. Construction, options, typed identifiers and errors (F4, F5, F6).
5. Rules for state, environment and logging (F7).
6. Extension: registry and conformance suite (F8).
7. How functional parity is proven when the API differs (F12).
8. Where the standards are written down, and how they are enforced.

## Not verified

- **Streaming endpoints.** Whether each provider's streaming endpoint behaves
  as documented. Only the ChatGPT event stream is exercised today.
- **Usage fields.** Whether every wire format reports token usage on every
  route. The field names were not surveyed.
- **`pi-go`'s eventual needs.** Its requirements beyond its v1 plan.
