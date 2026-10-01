---
status: accepted
date: 2026-09-30
decision-makers: go-llmprovider-sdk maintainers
consulted: owners of mcp-server-magicdev, mcp-server-magictools, prepare-commit-msg, pi-go
informed: every consumer of github.com/maccavelli/go-llmprovider-sdk
---
# Define a Canonical, Modular v1 API for go-llmprovider-sdk Before the First Release

## Context and Problem Statement

[0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md)
moves `llmprovider` and `wizard` out of `mcplib`. Its §3, as accepted,
required the first release's exported API to be "identical, identifier for
identifier and signature for signature" to `mcplib` `v1.6.0`.

On 2026-09-29 the owner restated the SDK's purpose:

* It must provide the current functionality: **functional parity**, not API
  identity.
* It is to be enhanced, extended, modularised and improved.
* It must be future-proofed by being canonical, defining standards, and
  keeping one consistent API as it grows.

[0015-REPORT-sdk-api-surface-assessment.md](../reports/0015-REPORT-sdk-api-surface-assessment.md)
measured the imported API against that requirement. In summary:

* **Features are method names** (F1). There are 64 `Generate*` methods on 8
  provider types and 8 generation interfaces, and streaming alone would
  double both.
* **Provider-specific settings sit in shared configuration** (F2). Seven of
  them are "Ignored by all other providers".
* **One flat package holds six concerns** (F3).
* **The same concept has several spellings.** There are 13 constructors,
  several listing functions and three error types (F4–F6).
* **The package holds global state** (F7). It has mutable exported catalogs,
  reads four environment variables and logs through the process-global
  logger.
* **There is no registry and no conformance suite** (F8).
* **The planned consumer's needs are missing** (F9). `pi-go` needs streaming,
  token usage and several tools at once, and the API has none of them.

The first tag must be `v1.x`, because the ChatGPT backend reads the SDK
version (0002-MADR §8), and `v1.0.0` fixes the compatibility promise (F11).
The shape of the API therefore has to be decided before that tag, or it
cannot change without a `/v2` module.

The question: **what is the SDK's v1 API, its package layout, and the
standards that keep it consistent as providers and features are added, given
that every behaviour of `mcplib` `v1.6.0` must remain available?**

## Decision Drivers

* **Owner requirement.** The API is canonical, consistent and extensible, and
  it defines standards.
* **Functional parity.** Every behaviour a consumer can reach through
  `mcplib` `v1.6.0` stays reachable, with the same bytes on the wire,
  except where 0002-MADR §4–§7 changes them.
* **Additive growth.** A new provider, request option or output type adds
  code. It never multiplies methods or interfaces, and never breaks v1
  callers.
* **One migration.** Consumers migrate once, from `mcplib` to the final v1
  API.
* **No lost knowledge.** The measured wire knowledge is kept, with its
  history and its 473 tests (F12).
* **Minimal dependencies.** The standard library only, except
  `golang.org/x/term` in `wizard`.
* **Checkable standards.** Every standard can be checked by a tool or a test,
  not only by review.

## Considered Options

* A. Ship the `mcplib` API as `v1.0.0`; redesign later as `/v2`.
* B. Redesign before `v1.0.0`, restructuring the imported code behind a
  canonical API.
* C. Keep the `mcplib` API in v1 and add the canonical API beside it,
  deprecating the old one.
* D. Write new providers against a new API and retire the imported
  implementation.

## Decision Outcome

Chosen option: "B. Redesign before `v1.0.0`, restructuring the imported code
behind a canonical API", because it is the only option that meets all of
these at once:

* the owner's requirement;
* a single consumer migration;
* no second module path;
* keeping the tested wire code and its history.

A leaves the combinatorial API in place for the life of v1. C ships it as
well, inside the compatibility promise. D discards measured behaviour that
took twelve records to establish.

### D1. Parity is functional

* Every capability of `mcplib` `v1.6.0` `llmprovider` and `wizard` has an
  equivalent in the SDK, or a recorded removal with its reason.
* Wire behaviour is unchanged except where 0002-MADR §4–§7 changes it. D12
  says how both are proven.
* Identifier and signature identity is not required. This replaces 0002-MADR
  §3 (see that record's amendment of the same date).

### D2. Layout: one module, packages by concern

The module stays `github.com/maccavelli/go-llmprovider-sdk`, one Go module,
with these packages:

| Package | Holds | May import |
|---|---|---|
| `llmprovider` | The contract: `Provider`, `Request`, `Response`, `Item` types, `Tool`, `Capabilities`, streaming `Event`s, `Usage`, typed identifiers, options, the error model, `Registry` and `Descriptor`, retry middleware, and the `TokenSource` / `Token` interfaces | the standard library |
| `llmprovider/auth` | OAuth sessions, refresh and revocation, the browser and device login flows, `TokenStore` and `FileTokenStore`, `StaticToken`, `VendorCLISession` | `llmprovider` |
| `llmprovider/catalog` | Static catalogs, model metadata, ranking, search, labels, profiles and the curated `Catalog` result | `llmprovider` |
| `llmprovider/providers/<id>` | One package per provider or gateway family: `openai`, `claude`, `gemini`, `grok`, `opencode`, `kilo`, `huggingface`, `ollama`. Each has its `New`, its own options and its `Descriptor` | `llmprovider`, `auth`, `catalog`, internal packages |
| `llmprovider/providers` | `Default()`, a fresh `Registry` holding every built-in provider, and `New(id, opts...)` over it | the provider packages |
| `llmprovider/llmtest` | The conformance suite and a scriptable fake provider for consumers' tests | `llmprovider` |
| `llmprovider/internal/wire/...` | The ~~four~~ five wire formats (Responses, Chat Completions, Messages, Gemini Interactions, and Gemini `generateContent` for OpenCode's Google route; *count corrected 2026-09-29*), shared by the providers and gateways | `llmprovider` |
| `llmprovider/internal/transport` | HTTP helpers, identity headers, `Retry-After` parsing, error-body classification | `llmprovider`, `internal/redact` |
| `wizard` | The configuration flow, over a `Registry` | the above, and `golang.org/x/term` |
| `internal/redact` | Redaction | the standard library |

* **Why the package names stay.** `llmprovider` and `wizard` keep their
  names, so `mcplib`'s deprecation notices and the moved records'
  `llmprovider/x.go` references still name a real package.
* **No second module.** A module is split out only when a dependency
  justifies one, and that takes its own MADR.

### D3. One generation contract

```go
type Provider interface {
	ID() ProviderID
	Capabilities() Capabilities
	Generate(ctx context.Context, req *Request) (*Response, error)
}
```

**`Request`** carries everything that is a method name today:

* `Model` (empty: the provider's configured model);
* `Instructions`;
* `Input []Item`;
* `Tools []Tool` and `ToolChoice`: auto, none, required, or one named tool;
* `Reasoning *Reasoning`: effort or budget, nil for none;
* `MaxOutputTokens`;
* `PreviousResponseID`.

**`Response`** carries:

* `ID`, `Model`, `Output []Item`;
* a typed `FinishReason`;
* `Usage`: input, output, reasoning and cached tokens, zero when the service
  reports none.

**How `mcplib`'s methods map.** Its eight generation methods and `Continue`
become `Request` values. Its convenience methods become package functions
over any `Provider`, for example a text helper and a tool-call helper. The
exact names are fixed by the standards guide (D13). Adding a request feature
later means adding a field, which is backward compatible in Go.

**`Item` stays a sealed interface.** Future content types (images, files)
are added as new types. Consumers' type switches must have a `default` case;
the standards guide says so and `llmtest` checks it for built-in code.

### D4. Streaming and capabilities are data, with universal fallbacks

* **`Capabilities`** declares, per provider, `Tools`, `ForcedToolChoice`,
  `Reasoning`, `Continuation` and `NativeStreaming`. Each is `Unsupported`,
  `BestEffort` or `Supported`.
* **A request needing an `Unsupported` capability** fails before any network
  call, with an error matching `ErrUnsupported`, which wraps
  `errors.ErrUnsupported`.
* **`BestEffort` keeps `v1.6.0`'s documented degradations.** Examples:
  Ollama offers a tool and does not force it; Hugging Face ignores reasoning
  where the routed model does. Each degradation is listed in the provider's
  package doc.
* **`llmprovider.Stream(ctx, p, req)`** returns `iter.Seq2[Event, error]`
  for every provider. It uses a provider's native `Streamer` implementation
  when it has one. Otherwise it runs `Generate` and emits the result as
  events. Callers can therefore stream against every provider from `v1.0.0`
  on, and native streaming is added provider by provider later, with no API
  change.
* **Usage is decoded** for every wire format that reports it. That is new
  functionality, not parity (F9).

### D5. Construction and options have one shape

* **Construction.** Every provider package has
  `New(opts ...llmprovider.Option) (llmprovider.Provider, error)`, and a
  `Registry` builds any provider by id with the same options. There is no
  context argument, and no network call happens at construction.
* **Credentials.** `llmprovider.WithAPIKey`, or
  `llmprovider.WithTokenSource`. This replaces every `*WithSource` twin.
* **Common options** live in `llmprovider`: model, HTTP client, base URL,
  logger, client identity, session id, max tokens and default reasoning.
* **Provider-specific options** live in their provider's package, for
  example `kilo.WithOrganization` and `opencode.WithRoute`.
* **An option given to the wrong provider is an error from `New`,** never
  silently ignored (F2).
* **Configuration structs are unexported.** There is no exported
  `ProviderConfig` or `ApplyOptions`.

### D6. Identifiers are typed

`ProviderID`, `Role`, `Effort`, `FinishReason`, `ToolChoice`,
`AuthMethodID` and `TokenType` are named types, with the canonical values as
constants. Provider ids keep the models.dev registry keys already in use.
Invalid values are rejected by `New` or `Generate` with `ErrInvalidRequest`.

### D7. One error type, classified by kind

* **`*APIError` is the only structured error.** It has `Provider`,
  `Status`, `Kind`, `Code` (the service's error type), a redacted and
  bounded `Message`, `RetryAfter`, and `Reason` (for a cut-short response).
  It absorbs `RateLimitError` and `IncompleteError`.
* **Sentinels name the kinds.** `errors.Is` works through `Unwrap` to one
  kind sentinel each: `ErrRateLimited` (with `ErrQuotaExhausted` beneath
  it), `ErrAuthFailure`, `ErrNotPermitted`, `ErrInvalidRequest`,
  `ErrProviderUnavailable`, `ErrIncomplete`, `ErrUnsupported` and
  `ErrInvalidProvider`. *Amended 2026-09-30 (accepted):* also
  `ErrContextOverflow`, beneath `ErrInvalidRequest`.
* **Retryability is a method.** `Retryable()` replaces `Terminal`.
* **Every message starts `llmprovider:`.**

### D8. Retry and cross-cutting behaviour are middleware

* `llmprovider.WithRetry(p, RetryPolicy{…}) Provider` wraps any provider. It
  honours `RetryAfter` and the error kind. It replaces the three
  `Generate*WithRetry` functions.
* The same `func(Provider) Provider` shape is the extension point for
  logging, metrics and rate limiting. There are no hooks inside the
  providers.

### D9. No ambient state

* **Nothing mutable is exported at package level.** Catalogs and
  environment-variable names are returned as copies by functions.
* **Library code reads no environment variable.** The four variables of F7
  are read only by explicit helpers a caller opts into, for example
  `catalog.OptionsFromEnv()` and `auth.GrokFlowFromEnv()`. The names stay
  those of 0002-MADR §6.
* **Logging goes only to a `*slog.Logger`** passed with `WithLogger`. With
  none, nothing is logged. The library never uses the process-global logger.
* **Providers are immutable after `New`** and safe for concurrent use. The
  race detector in the conformance suite asserts it.

### D10. Extension goes through a registry

* **`Registry`** holds `Descriptor` and `Factory` pairs. `Register` refuses
  a duplicate id.
* **`providers.Default()`** returns a new `Registry` holding the built-ins.
  There are no `init`-time side effects and no global registry.
* **`wizard`** offers what its `Options.Registry` holds, `providers.Default()`
  when nil. A third-party provider registered there is configured like a
  built-in one.
* **Adding a provider** means adding one package and one line in
  `providers.Default()`, and passing `llmtest`.

### D11. One conformance suite for every provider

* **`llmtest.Run(t, harness)`** runs the contract against a provider wired to
  a fake server the harness supplies. It checks:
  * capabilities honoured, and unsupported requests refused before the
    network;
  * cancellation;
  * status-to-kind classification;
  * identity headers;
  * the `Response` invariants;
  * concurrent use under `-race`.
* **Every built-in provider passes it,** and so must any third-party provider
  that wants to claim conformance.
* **`llmtest.Fake`** is a scriptable provider for consumers' own tests.

### D12. Functional parity is proven three ways

1. **Wire goldens (G-wire).** Before the restructure, the `mcplib` API at
   the end of 0002-PLAN Phase 7 records a golden file for each
   provider × {text, forced tool, thinking, thinking tool, items,
   continuation}. It is the HTTP request as normalised JSON: method, path,
   the non-volatile headers and the body.
   * After the restructure, the same scenario through the new API must
     produce the same golden.
   * The only allowed differences are fields a record changes, each listed.
   * Response fixtures must decode to equivalent `Output`.
2. **Ported tests.** The existing tests move with their code. An assertion
   may change its call syntax but not what it asserts.
3. **Parity map (G-parity).**
   `docs/guides/migrating-from-mcplib.md` maps every exported identifier of
   `mcplib` `v1.6.0` `llmprovider` and `wizard` to its SDK equivalent, or
   states its removal and why. A script fails when any identifier from
   `go doc -all` at `v1.6.0` is missing from the map.

### D13. Standards are written, versioned and enforced

* **`docs/guides/api-standards.md`** is the normative convention set, each
  rule citing the decision here that sets it. It covers:
  * package placement;
  * naming;
  * construction and options;
  * request fields;
  * errors;
  * context;
  * concurrency;
  * state;
  * logging;
  * documentation;
  * tests and coverage.
* **`docs/guides/adding-a-provider.md`** is the checklist for a new
  provider.
* **`docs/architecture.md`** describes the layout as built.
* **AGENTS.md** requires the standards guide for any API change.
* **Enforcement:**
  * `llmtest` for every provider;
  * a per-package dependency check: only `wizard` may leave the standard
    library;
  * `golangci-lint`;
  * from `v1.0.0` on, `apidiff` against the latest release tag. It is run
    in CI with `go run` at a pinned version. It is a tool, not a module
    requirement.
* **Compatibility.** Every non-internal package is under the v1
  compatibility promise, except packages under `llmprovider/x/`. That tree
  holds experimental API, which may change until it is promoted by a MADR.
  Deprecated identifiers carry `Deprecated:` and remain until a major
  version.
* **Coverage.** No package's statement coverage may fall below its baseline
  at the end of 0002-PLAN Phase 7, and every new package holds at least
  80 %.

### D14. Sequence

1. **0002-PLAN Phases 4–7 complete as written:** the mechanical re-home,
   records and citations. They give a green baseline for D12's goldens.
2. **Then this record's PLAN.**
3. **Then 0002-PLAN Phase 8:** live identity gates and `v1.0.0`.
4. **Then the consumer companions,** which adopt this API rather than only
   rewriting import paths.
   *Amended 2026-09-29 (0002-MADR sixth amendment):* `prepare-commit-msg`'s
   only, for now.

Release candidates (`v1.0.0-rc.N`) may be tagged from this PLAN's later
phases for consumers to trial. The ChatGPT `client_version` they send is
`1.0.0`, above the `0.155.0` threshold (0015-REPORT F11).

### Consequences

* Good, because a new feature is one `Request` or `Response` field and a
  new provider is one package, so v1 can grow without breaking callers.
* Good, because streaming, usage and multiple tools, which `pi-go` needs,
  are in the contract from `v1.0.0`.
* Good, because provider-specific behaviour is typed and scoped to its
  package, so a misapplied option fails loudly.
* Good, because the conformance suite, the dependency check and `apidiff`
  make the standards checkable, not advisory.
* Good, because consumers migrate once and get a mapping guide for every
  identifier they use.
* Neutral, because the wire code, its history and its tests are kept. They
  move into packages; they are not rewritten.
* Bad, because the restructure is large. It touches every exported
  identifier and most test call sites, and delays `v1.0.0` until it is done.
* Bad, because the consumer companions become code migrations instead of
  import-path rewrites, and must be re-scoped.
* Bad, because the wire goldens only prove the scenarios they record.
  Behaviour outside them relies on the ported tests.
* Bad, because the process-global logger is no longer used. A consumer that
  relied on the library's warnings reaching `slog.Default()` must pass
  `WithLogger`.

### Confirmation

* G-wire, G-parity, `llmtest` for all eight built-in providers, the
  per-package dependency check and the coverage floors pass, with their
  output and first-fail experiments in this PLAN's execution record.
* `docs/guides/api-standards.md`, `docs/guides/adding-a-provider.md`,
  `docs/guides/migrating-from-mcplib.md` and `docs/architecture.md` exist
  and describe the built tree.
* A third-party provider written only from `adding-a-provider.md`, in a
  scratch module, registers, is offered by `wizard` and passes `llmtest`.

## Pros and Cons of the Options

### A. Ship the `mcplib` API as `v1.0.0`; redesign later as `/v2`

* Good, because `v1.0.0` ships soonest, and 0002-PLAN runs as written.
* Bad, because v1 locks in the combinatorial API (F1). Streaming and
  multiple tools would add dozens of methods to it.
* Bad, because consumers migrate twice and a second module path follows.

### B. Redesign before `v1.0.0`, restructuring the imported code

* Good, because it meets every driver.
* Bad, because the release waits for the restructure, and the parity gates
  must be built.

### C. Keep the `mcplib` API in v1 and add the canonical API beside it

* Good, because parity is trivially true, and consumers could move at
  leisure.
* Bad, because both APIs are under the v1 promise. The old one cannot be
  removed until v2, so "one consistent API" is false for all of v1.
* Bad, because every feature has to be implemented or refused twice.

### D. New providers against a new API; retire the imported code

* Good, because it gives the cleanest internals.
* Bad, because it discards measured wire behaviour: quota classification,
  route tables, the ChatGPT rules, Gemini signature replay, identity. That
  behaviour exists because live tests found each case.
* Bad, because parity could only be asserted, not proven against the code
  that has it.

## More Information

* Evidence:
  [0015-REPORT-sdk-api-surface-assessment.md](../reports/0015-REPORT-sdk-api-surface-assessment.md).
* Implementation:
  [0015-PLAN-canonical-sdk-api-and-module-layout.md](0015-PLAN-canonical-sdk-api-and-module-layout.md).
* Changes to the migration:
  [0002-MADR-migrate-llmprovider-from-mcplib.md](0002-MADR-migrate-llmprovider-from-mcplib.md),
  "Amendment 2026-09-29 (second): functional parity and the canonical API".
* **Cross-repository, cited by name.**
  * `mcplib` `docs/decisions/0015-MADR-transfer-llmprovider-to-go-llmprovider-sdk.md`,
    whose `v1.6.1` deprecation notice points consumers here.
  * `pi-go` `docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md`, Phase 3,
    the first consumer that needs streaming.
* **Out of scope,** each taking its own record when wanted:
  * native streaming for any given provider;
  * image and file items;
  * structured-output requests;
  * new providers;
  * a second module.
* **Revisit** if G-wire cannot be made to pass for a provider without a
  behaviour change. That is a stop condition in the PLAN, not something the
  implementation decides.

## Amendment 2026-09-29: the proxy exemption, from 0016

[0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md) D8 sets `Proxy: http.ProxyFromEnvironment` on the default
transport. That is the standard library reading the proxy variables, not this
module. D9's rule stands for library code: it reads no environment variable
itself. D9's ambient check (PLAN S10) allows exactly this one call.

## Amendment 2026-09-30: a context-overflow kind, from the reference-client survey (proposed)

Status: **accepted** 2026-09-30 ("3. add it"). S6 builds it. Evidence: [0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md).

* **Gap.** Only OpenAI's `context_length_exceeded` type is recognised
  (`llmprovider/api_error.go:157`). Every other vendor's overflow is a
  plain `ErrInvalidRequest`, which a caller cannot tell from a malformed
  request. Examples:
  * Together: "The input (X tokens) is longer than the model's context
    length (Y tokens)";
  * xAI, Google and OpenRouter each word it differently.
* **What pi does.** It recognises about 24 vendor forms
  (0017-REPORT P7), because a caller's right response to overflow is to
  shorten the input and retry.
* **Proposed for D7.** A kind sentinel `ErrContextOverflow` beneath
  `ErrInvalidRequest`, as `ErrQuotaExhausted` sits beneath
  `ErrRateLimited`. It is classified from:
  * the service's error type, where there is one;
  * otherwise a per-vendor message table, each entry citing where it was
    seen.

  It is never retryable. The standards guide's R25 gains it on acceptance.

## Amendment 2026-09-30: listing, and options scoped per provider

Status: **accepted** 2026-09-30 by the owner, before the first provider moved
in 0015-PLAN S7.

### D3: `ModelLister`

* **Gap.** S7 step 2 names `ListModels`, but `Provider` has no listing
  method. `New` returns `Provider`, so a caller had no way to reach a
  provider's listing.
* **Decision.** An optional interface in `llmprovider`, which a caller
  type-asserts, as with `Streamer`:

  ```go
  type ModelLister interface {
  	ListModels(ctx context.Context) ([]string, error)
  }
  ```

  `ListModels` keeps the old `DiscoverModels` behaviour: the curated
  listing, probed where [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md) A5 says.
  `ModelDiscoverer` is removed with the old API in S8.
* **Rejected:**
  * keeping `ModelDiscoverer` in the new API, which carries an old-API
    name;
  * exported concrete types without a shared interface, which a caller
    holding several providers cannot list generically.

### D5: a common baseline, with overlays per provider

* **Question.** A5's probe options are common to five providers, and S6
  had classed them as old-API-only. The owner asked instead for "a common
  options baseline with provider scoping as extended schema attributes",
  for extensibility.
* **Decision.**
  1. **Baseline.** A common option applies to every provider.
     `WithModelProbes` and `ModelProbesFromEnv` are common options.
  2. **Overlay.** `For(id, opts...)` applies its options only when building
     `id`, after the baseline, so the more specific wins wherever it sits
     in the list. For any other id it is skipped. It is not a silent
     ignore, because the caller named the target.
  3. **Strict.** A provider-specific option given bare, such as
     `kilo.WithOrganization`, is still an error from another provider's
     `New`. Inside `For(kilo, …)` it may sit in a list shared by several
     providers.
  4. **Refused.** An old-API-only option inside `For`, and a `For` nested
     under a different id.
* **Schedule.** Step 1 is built now. Steps 2–4 are built in 0015-PLAN
  Phase S8b, once `wizard` builds through the `Registry`. The standards
  guide's R18 changes then, to read "a foreign option is an error from
  `New`, unless the caller scoped it with `For`".
* **Rejected:**
  * a scoped `WithModelProbes` in each provider package, with which
    `ModelProbesFromEnv` could not stay one function, contrary to A5;
  * an overlay decided by list order alone, with which a baseline option
    placed after an overlay would beat it.

## Amendment 2026-09-30: the OpenCode family, scoped options and the metadata URL

Status: **accepted** 2026-09-30 by the owner, before the opencode commit of
0015-PLAN S7.

The `opencode` package serves two ids, `opencode-zen` and `opencode-go`
(D2). Moving it raised three questions D5 had not settled, and one gap.

### D5: one constructor per id in a family package

* **Gap.** R14 gives each provider package one `New(opts...)`. A family
  package serves several ids, and the `Registry` needs one constructor per
  id.
* **Decision.** A package for a family of gateways has one constructor per
  id it serves, named for that id: `opencode.NewZen` and `opencode.NewGo`,
  each `func(opts ...llmprovider.Option) (llmprovider.Provider, error)`.
  R14 reads so.
* **Rejected:** a single `New` that builds Zen unless an
  `opencode.WithGateway` option says otherwise. It keeps R14's wording, but
  the `Registry`'s Go entry has to wrap it, and a forgotten option builds
  the wrong gateway silently.

### D5: an option scoped to several ids

* **Gap.** `ScopedOption` scopes an option to one id, and `New` refuses it
  for any other. `opencode.WithRoute` (named in D5) must reach both
  gateways.
* **Decision.** `llmprovider.ScopedOptionFor(ids []ProviderID, name,
  value)`: accepted by each id in the list, refused by every other.
  `ScopedOption(id, …)` is `ScopedOptionFor([]ProviderID{id}, …)`. The
  `For(id, …)` overlay of S8b is unchanged by it.
* **Rejected:** one option per gateway (`WithZenRoute`, `WithGoRoute`),
  which duplicates the API.

### D5: the metadata URL is a common option

* **Gap.** `WithModelMetadataURL` was old-API only. The open-catalog
  providers (OpenCode, Hugging Face, Kilo, Together) read the metadata
  document. OpenCode reads it on every request, for the route and the
  reasoning fields. Without the option, a new-API caller could reach the
  document only through `LLMPROVIDER_MODELS_METADATA_URL`, which D9 removes.
* **Decision.** `WithModelMetadataURL` joins the common baseline, as
  `WithModelProbes` did. A provider that reads no metadata ignores it.
  `Settings.ModelMetadataURL()` reads it.
* **Rejected:**
  * a scoped copy in each of the four packages;
  * no option until S8b, which leaves tests and callers on the environment
    variable.

### A known gap: no profile on the new API until S8b

The S7 prerequisites (0015-PLAN, amendment of 2026-09-30, decision 2) moved
the catalog extraction to S8b and rejected a scoped profile option in S7.
So from the opencode commit until S8b, a new-API open-catalog provider
lists with the default profile: `WithModelProfile` is old-API only.
`wizard` is not affected, as it lists through the old catalog functions
until its port in S8. The owner accepted the gap.

## Amendment 2026-10-01: how `llmprovider`'s coverage is measured during S7

Status: **accepted** 2026-10-01 by the owner.

* **Fact found.** D13 says no package's statement coverage may fall below
  its baseline. It does not say which tests count. During 0015-PLAN S7, the
  wire code the moved providers share stays in `llmprovider` until S7b,
  and the tests that exercise it move with the providers. After the
  opencode commit (`4adecf3`), `llmprovider` measured 86.4 % from its own
  tests, against its `P7` 89.2 %. With every test under `./llmprovider/...`
  counted (`-coverpkg=./llmprovider`), it measured 90.9 %.
* **Decided.** Until 0015-PLAN S7b has moved the shared wire, transport and
  `auth` code out of `llmprovider`, its floor is measured with
  `-coverpkg=./llmprovider` over `./llmprovider/...`. Every other package,
  and `llmprovider` after S7b, is measured by its own tests, as before.
* **Rejected:** adding tests inside `llmprovider` for code the provider
  packages already test. They would duplicate those tests, grow with every
  later move, and move again in S7b. A lower floor was not offered, as it
  loosens the check.
