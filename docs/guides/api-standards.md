# API standards

The conventions every exported API in this module follows. The guide is
**normative**: an API change that breaks a rule is wrong, not a judgement call.
It changes only by amending the decision a rule cites. Each rule names that
decision, in
[0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
("0015 D*n*") or
[0016-MADR-provider-auth-and-support-baseline.md](../decisions/0016-MADR-provider-auth-and-support-baseline.md)
("0016 D*n*"), and R49 to R55 and the reasons for R14, R15 and R19 in
[0048-MADR-adopt-go-sdk-best-practices.md](../decisions/0048-MADR-adopt-go-sdk-best-practices.md)
("0048 D*n*", "0048 X*n*"), R56 in
[0062-MADR-learn-thinking-parameters-from-refusals.md](../decisions/0062-MADR-learn-thinking-parameters-from-refusals.md)
("0062 D*n*"), R57 and the output limit of R6 and R7 in
[0089-MADR-model-output-limit-discovery.md](../decisions/0089-MADR-model-output-limit-discovery.md)
("0089 D*n*", "0089 DL*n*") and
[0088-MADR-default-output-limit-from-model-facts.md](../decisions/0088-MADR-default-output-limit-from-model-facts.md)
("0088 ST*n*"), and R24's `ResetAt` and `Limit` in
[0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md](../decisions/0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md)
("0072 D*n*").

**The tree meets these rules.** Phase S11 of
[0015-PLAN-canonical-sdk-api-and-module-layout.md](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md)
checked each one against the built code. The [Checks](#checks) table names the
tool that holds a rule, where one does. For how the code is laid out, read
[architecture.md](../architecture.md); to add a provider, read
[adding-a-provider.md](adding-a-provider.md).

## Packages

- **R1. One module.** Everything is in `github.com/maccavelli/go-llmprovider-sdk`.
  A second module needs its own MADR and a dependency, or a wire contract that must version apart from the SDK, that justifies it.
  (0015 D2, and its amendment for 0073)
- **R2. A package holds one concern**, and imports only what the table allows.
  Only `wizard` may leave the standard library, and then only for
  `golang.org/x/term` and `golang.org/x/sys` (0087-MADR A6a); `wizard`, `llmprovider/x/signin` and
  `internal/signinflow` may also import the nested module `authcontract`,
  which is itself standard library only (0073-MADR D1). (0015 D2, D13; the table's `internal/wire` rows, 0026
  F62; `internal/listlimit`, 0038 A3; `internal/chatgpt`, 0041 D9; the import rows, 0046 D4; `internal/cacheref`, 0048 D1, 0049 D3, 0062 D5; `internal/nilsource`, 0050 D2;
  `internal/toolschema`, 0084 D3; `internal/compatid`, 0061 E1; `internal/compatdialect`, 0061 E2, E5; `internal/opencoderoute`, 0089 DL74;
  `internal/limitmemo`, 0089 D5, and its import by `llmprovider` (`WithoutLimitLearning`'s value, DL73) and
  `catalog` (a kept cap's evidence, DL76); `catalog`'s `internal/retired`, 0089 DL51; `llmprovider/x/signin`, `internal/signinflow` and `authcontract`, 0074 D3)

  | Package | Holds | May import |
  | :--- | :--- | :--- |
  | `llmprovider` | the contract: `Provider`, `Request`, `Response`, `Item`, `Tool`, `Capabilities`, `Event`, `Usage`, typed identifiers, options, errors and their classification, `Registry`, `Descriptor`, retry middleware, `TokenSource` and `Token`, `StaticToken` and `CommandToken` | the standard library, `internal/redact`, `internal/transport`, `internal/nilsource`, `internal/toolschema`, `internal/limitmemo` |
  | `llmprovider/auth` | OAuth sessions and flows, `TokenStore`, `FileTokenStore`, `VendorCLISession` | `llmprovider`, `internal/transport`, `internal/redact`, `internal/recorderr`, `internal/kiloendpoint`, `internal/ownerperm`, `internal/filelock` |
  | `llmprovider/catalog` | static catalogs, model metadata, a model's limits and their evidence (`Limits`), ranking, search, labels, profiles, the curated `Catalog` and its menu, raw listings, `Cache` | `llmprovider`, `auth`, `internal/transport`, `internal/kiloendpoint`, `internal/listlimit`, `internal/chatgpt`, `internal/cacheref`, `internal/nilsource`, `internal/compatid`, `internal/compatdialect`, `internal/opencoderoute`, `internal/limitmemo`, `internal/retired` |
  | `internal/kiloendpoint` | Kilo's endpoints, derived from a credential, and whether one has an organization; in the module's own `internal`, not `llmprovider`'s, so that `internal/signinflow` may import it (0080 D4) | the standard library |
  | `llmprovider/internal/cacheref` | `catalog.WithCache`'s value, for `catalog` and the providers | `internal/transport`, `internal/wire`, `internal/wire/messages`, `internal/limitmemo` |
  | `llmprovider/internal/limitmemo` | the memo of validated output caps, one per `catalog.Cache` through `internal/cacheref` (0089-MADR D5) | the standard library |
  | `llmprovider/internal/nilsource` | a typed-nil `TokenSource`'s test, for `llmprovider` and `catalog` | the standard library |
  | `llmprovider/internal/toolschema` | a service's refusal of a tool's schema, read from a reply: its form and the names it gives, for `llmprovider`'s classification and `internal/wire`'s naming (0084 D3) | the standard library |
  | `llmprovider/internal/listlimit` | `catalog.WithLimit`'s value, for `catalog` and the providers that cut their own fallback listings with it | the standard library |
  | `llmprovider/internal/compatid` | `chatcompat.WithID`'s value, for `catalog` and `providers/chatcompat` | the standard library |
  | `llmprovider/internal/compatdialect` | which server a Chat Completions listing is from, for `catalog` and `providers/chatcompat` | the standard library |
  | `llmprovider/internal/chatgpt` | a ChatGPT session's check, headers and listing request, and a ChatGPT plan session's check and listing request, for `catalog`'s `ListRaw` and `providers/openai` | `llmprovider`, `auth`, `internal/transport` |
  | `llmprovider/internal/opencoderoute` | OpenCode's route snapshot and table, for `providers/opencode`'s routing and `catalog`'s `Limits.Required` | `llmprovider` |
  | `llmprovider/internal/retired` | the retired model ids, with why (0033-MADR D4) | `llmprovider` |
  | `llmprovider/internal/catalogtool` | the maintainer tool's proposals for the static lists | `llmprovider`, `catalog`, `internal/retired` |
  | `llmprovider/internal/cmd/catalogs` | the maintainer tool's command, run with `go run` | `llmprovider`, `catalog`, `internal/catalogtool`, `internal/transport` |
  | `llmprovider/internal/cmd/sdkaudit` | the API audit's command, run with `go run` (0048) | the standard library |
  | `llmprovider/internal/ownerperm` | a directory and its files private to the current user: modes on Unix, a protected DACL on Windows through generated `syscall` bindings | the standard library |
  | `llmprovider/internal/filelock` | an exclusive OS lock on an open file: `flock` on Unix, `LockFileEx` on Windows through generated `syscall` bindings | the standard library |
  | `llmprovider/providers/<id>` | one provider or gateway family: `New`, its options, its `Descriptor` | `llmprovider`, `auth`, `catalog`, internal packages |
  | `llmprovider/providers` | `Default()`, `DefaultWith(opts...)` and `New(id, opts...)` | `llmprovider` and the provider packages |
  | `llmprovider/llmtest` | the conformance suite and `Fake` | `llmprovider` |
  | `llmprovider/internal/wire` | what the shared wire formats have in common | `llmprovider`, `internal/transport`, `internal/redact`, `internal/toolschema` |
  | `llmprovider/internal/wire/<format>` | one wire format | `llmprovider`, `internal/wire`, `internal/transport`, `internal/redact`, `internal/wire/deferral` |
  | `llmprovider/internal/wire/deferral` | how a request's deferred tools are sent, for the wire formats and the providers | `llmprovider`, `internal/wire/messages` |
  | `llmprovider/internal/transport` | the default client, identity headers, `Retry-After`, the listing probe | the standard library |
  | `llmprovider/x/signin` | experimental: one sign-in handle (`Start`), a descriptor's methods in the contract's words (`Methods`), `Credential`, `ErrorKindOf` (0074 D2, D3, D8) and `VerifyKilo`, a Kilo key's check against Kilo's profile (0080 D10) | `llmprovider`, `auth`, `authcontract`, `internal/signinflow`, `internal/redact` |
  | `internal/signinflow` | the step engine under `llmprovider/x/signin` and `wizard` (0074 D3, D4, D6), and Kilo's profile check they share (0080 D10) | `llmprovider`, `auth`, `authcontract`, `internal/redact`, `internal/recorderr`, `internal/kiloendpoint` |
  | `wizard` | the configuration flow, over a `Registry` | the above, `authcontract`, `internal/redact`, `golang.org/x/term`, `golang.org/x/sys/unix`, `golang.org/x/sys/windows` |
  | `internal/redact` | redaction | the standard library |
  | `internal/recorderr` | the marker of a client record file over its size bound, shared by `auth` and `internal/signinflow`, in an internal package so that no store of another module can return it (0082 sixth amendment) | the standard library |
  | `llmprovider/internal/wirecase` | G-wire's scenarios, for the provider packages' tests only | `llmprovider`, `internal/wiretest` |
  | `internal/wiretest` | G-wire's request recorder, for tests only | the standard library |
  | `internal/nettest` | a refused connection and a free loopback port, for tests only (0045) | the standard library |
  | `internal/localonly` | a guard that fails a test run sending a request off the machine, for tests only (0065) | the standard library |
  | `internal/ambientcheck` | the R30–R31 check, a test with no package API | the standard library |

- **R3. Wire knowledge stays internal.** A wire format shared by more than one
  provider lives under `internal/wire`, never in an exported package.
  (0015 D2)
- **R4. Experimental API lives under `llmprovider/x/`** and nowhere else. It is
  promoted by a MADR. (0015 D13)

## The contract

- **R5. One generation method.** A provider implements `ID()`, `Capabilities()`
  and `Generate(ctx, *Request) (*Response, error)`. A new feature is a new
  field on `Request` or `Response`, never a new method or interface.
  (0015 D3)
- **R6. `Request` fields.** `Model` (empty means the provider's configured
  model), `Instructions`, `Input []Item`, `Tools []Tool`, `ToolChoice`,
  `Reasoning *Reasoning` (nil for none), `MaxOutputTokens`,
  `PreviousResponseID`, and `Cache` (`CacheShort` or `CacheLong`; zero asks
  for no caching). A field's zero value means "the provider's default"
  or "not requested". A `Tool` may be `Deferred`, kept out of the model's
  context until a `ToolLoadItem` in `Input` loads it, and one tool may be
  the caller's `Search` for them; every provider honours them, in its
  service's form, through a fixed wrapper, or by appending the loaded tool,
  and says which in its package doc. A response that did not use the
  service's own form carries `WarningDeferralEmulated`. (0015 D3; 0039 D1;
  0040 D1, D5, D7)
  What a request leaves to the provider, the model's facts decide: the
  thinking shape and its budget bounds, the efforts or levels a model
  takes, and the Chat Completions field that reasoning is replayed under.
  The facts are the caller's layers, then the provider's listing, then
  the models.dev document given to the provider's `New` with
  `catalog.WithMetadata`, read only from that document and what is cached
  (`catalog.CachedFacts`). A request never waits for them or fetches them.
  Without them, and with no document, the request decides (0093 D2, Q4). A model's refusal of an
  effort or a thinking shape, learned, outranks the facts and the request.
  (0041 D1, D2, A3; 0062 D2, D3)
  The output limit follows R57. `MaxOutputTokens` is sent as given; a
  `WithMaxTokens` value is cut to the model's stated output limit and never
  raised; the ChatGPT backend and plan routes send none, whatever is given
  (0088 ST4). With neither, a Messages route (Claude, OpenCode's messages
  route), whose wire requires a limit, sends the model's output limit as
  `catalog.CachedLimits` states it: the caller's layers, then the service,
  then the registry (a layer named "models.dev", then the document given
  to `New`; the SDK embeds none, 0093 D7), then a cap passed back with `WithLimitObservations`,
  held to the stated window and to a cap the service stated; where none is
  stated, the smallest limit of the route's current models, and where none
  of those is either, `ErrInvalidRequest` before sending. Kilo sends its
  stated output limit (0089 DL65, measured 2026-11-24: with none sent, Kilo cut one model at 128,000 of 131,072 stated), and none when nothing
  states one (0093 D8, Q2); every other
  wire sends none, so the service's own default applies. A limit the SDK
  chose, the stated limit or the route's smallest, is held to the room the
  stated window (else the stated input limit) leaves the request's input,
  estimated from its content as pi does, never below the wire's minimum,
  before a thinking budget raises it; a caller's value is not held. A
  `Generate` that names no limit is read over the service's stream and
  returned whole. (0089 D1 to D7, DL46 to DL51, DL56; 0088 ST15 to ST18,
  ST87, ST88, ST105, ST106)
- **R7. `Response` fields.** `ID`, `Model`, `Output []Item`, a typed
  `FinishReason`, `Usage`, which is zero when the service reports none, and
  `Warnings`, typed non-fatal notes, nil when the request was sent as asked
  (0034 D7), and `OutputLimit`, a `SentLimit`: the output limit the request
  carried, after a thinking budget raised it, whether one was sent, its
  `LimitBasis` and its source; a stream's final response carries it too
  (0089 D7, DL54, DL55).
  An answer the output limit cut is one `Response` on every wire, by
  `Generate` and at a stream's `EventDone`: `FinishLength`, its text and its
  whole tool calls, each call the limit cut dropped and named in a
  `WarningToolCallCut` and never yielded; with nothing usable it is
  `ErrEmptyAnswer` with the service's reason (0088 ST75 to ST81, ST93,
  ST94, ST103).
  In `Usage` a total holds its part: `InputTokens` includes `CachedTokens`,
  and `OutputTokens` includes `ReasoningTokens`, whatever the service's own
  convention. (0015 D3, D4, and its amendment "what `Usage` counts")
  `InputTokens` includes `CachedTokens` and `CacheWriteTokens`, which do not
  overlap, and `CacheWriteTokens` includes `CacheWrite1hTokens`. (0036 D1,
  D3)
- **R8. Convenience is a package function over any `Provider`,** never a method
  on one. The two that replace `mcplib`'s convenience methods are
  `llmprovider.GenerateText(ctx, p, req) (string, error)`, which returns the
  text of the output, and `llmprovider.GenerateToolCall(ctx, p, req)`, which
  forces the request's single tool and returns its call. (0015 D3, which
  delegates the names to this guide)
- **R9. `Item` is sealed.** New content is a new `Item` type. Code that
  switches on `Item`, built-in or a consumer's, has a `default` case.
  (0015 D3)

## Capabilities and streaming

- **R10. Capabilities are data.** A provider declares `Tools`,
  `ForcedToolChoice`, `Reasoning`, `Continuation`, `NativeStreaming` and
  `Images`, each `Unsupported`, `BestEffort` or `Supported`. (0015 D4;
  0061 D5)
  A model's are data too: its facts, as `catalog.ModelInfo`, with each
  one's source. A `Catalog`'s `Menu` holds every usable model with its
  facts, its rank and its `Caveat`s. A caveat sinks a model and never
  removes it. Every provider's band is ranked from the facts. A profile
  orders it, and `ProfileAgent` orders `ProfileCapable`'s band by the
  price of a cached turn, with `WithWorkload`'s shares or
  `ObservedWorkload`'s. `Catalog.Source` says whether a listing is live,
  the last good one after a transient failure, or the static floor.
  `ListRaw` returns the listing with none of this. (0041 D3–D6, D9)
- **R11. An unsupported request fails before the network,** with an error
  matching `ErrUnsupported`, which wraps `errors.ErrUnsupported`. (0015 D4)
- **R12. Every `BestEffort` degradation is listed** in the provider's package
  doc. (0015 D4)
- **R13. Streaming works for every provider.** `llmprovider.Stream(ctx, p, req)`
  returns `iter.Seq2[Event, error]`. It uses a provider's `Streamer` when it
  has one, and otherwise emits `Generate`'s result as events. Native
  streaming is added without an API change. (0015 D4)

## Construction and options

- **R14. One constructor shape.** Each provider package has
  `New(opts ...llmprovider.Option) (llmprovider.Provider, error)`. A
  package for a family of gateways has one such constructor per id it
  serves, named for it, such as `opencode.NewZen` and `opencode.NewGo`. A
  constructor takes no context and makes no network call. It returns the
  `Provider` interface, not a concrete type, as a middleware returns what it
  wraps (R28): the module's product is one contract over many providers,
  and a `Registry` holds factories of one signature, as `sha256.New` returns
  a `hash.Hash`. (0015 D5, and its amendment "the OpenCode family"; 0048 X1)
- **R15. Credentials are a `TokenSource`.** `WithTokenSource`, or `WithAPIKey`
  as shorthand for a `StaticToken`. There are no `*WithSource` twins. A
  required credential is an option, not a parameter, so that one
  constructor shape serves a keyless provider, such as Ollama, and every
  kind of source; where the service needs one, `New` refuses a missing one
  at once. (0015 D5, 0016 D2; 0048 X2)
- **R16. A provider applies the token as `Token` describes it.** An empty
  `Header` means the service's own header and scheme. A non-empty one
  overrides it: a `TokenBearer` is sent as `Bearer <value>`, a `TokenAPIKey`
  or an untyped token bare. A source names a `Header` only when its caller
  set one. `New` refuses a source whose kind the service does not accept.
  (0016 D2, A6)
- **R17. Options live where they apply.** The common ones (model, HTTP client,
  base URL, logger, client identity, session id, max tokens, default
  reasoning and listing probes) are in `llmprovider`; the model metadata
  URL and turning its fetch off went with the fetch (0093 D10). A
  provider-specific one is in its provider's package, for example
  `kilo.WithOrganization`; one that several ids of a family take is scoped
  to all of them, as `opencode.WithRoute` is to both gateways. One that every
  provider takes, whatever its id, is a shared option
  (`llmprovider.SharedOption`), as `catalog.WithCache`,
  `catalog.WithModelLayer`, `catalog.WithMetadata` (0093 D1) and
  `chatcompat.WithID` are. (0015 D5, and its
  amendment "the OpenCode family"; 0061, amendments "Phase 4's options" and
  "E2 and E4's menu, made exact before Phase 4c")
- **R18. A foreign option is an error from `New`,** never ignored, unless
  the caller scoped it with `For(id, …)`: an overlay for another id is
  skipped. A shared option is foreign to no provider: one that does not read
  it ignores it. (0015 D5 and its amendment of 2026-09-30; 0061, amendment
  "Phase 4's options")
- **R19. Configuration structs are unexported.** No exported `ProviderConfig` or
  `ApplyOptions`. The module serializes no configuration of its own: a
  consumer keeps its own, and `wizard.Result` is exported for it to keep.
  (0015 D5; 0048 X3)
- **R20. After `New`, a provider is immutable and safe for concurrent use.**
  (0015 D9)

## Identifiers

- **R21. Closed sets are named types.** `ProviderID`, `Role`, `Effort`,
  `FinishReason`, `ToolChoice`, `AuthMethodID` and `TokenType` are named
  types. Each canonical value is a constant whose name starts with a short
  prefix naming its type (`RoleUser`, `EffortHigh`), and a value `mcplib`
  already named keeps that name (`ProviderGemini`, `AuthAPIKey`). (0015 D6)
- **R22. Provider ids are the models.dev registry keys** already in use;
  a provider a caller builds with `chatcompat` takes the caller's id.
  (0015 D6; 0061 D4)
- **R23. An invalid value is `ErrInvalidRequest`** from `New` or `Generate`.
  (0015 D6)

## Errors

- **R24. `*APIError` is the only structured error.** It carries `Provider`,
  `Status`, `Kind`, `Code`, a redacted and bounded `Message`, `RetryAfter`,
  `ResetAt`, `Limit`, `Reason`, `Tool`, the refused tool's name when the kind
  is `ErrToolSchemaRefused`, and `Observed`, the output cap a refusal stated
  and the provider kept, on the error that ends the call. (0015 D7; 0072 D1,
  D2; 0084 D2; 0089 DL78, DL94)
- **R25. Test errors by kind with `errors.Is`.** Each error unwraps to one of
  `ErrRateLimited` (with `ErrQuotaExhausted` beneath it), `ErrAuthFailure`,
  `ErrNotPermitted`, `ErrInvalidRequest` (with `ErrContextOverflow`,
  `ErrIncomplete` and `ErrToolSchemaRefused` beneath it), `ErrProviderUnavailable`, `ErrUnsupported` or
  `ErrInvalidProvider`. A new failure mode maps to an existing kind before it
  earns a new one. (0015 D7 and its amendments of 2026-09-30 and 2026-10-01; 0084 D1)
- **R26. Retryability is `Retryable()`.** (0015 D7)
- **R27. Every sentinel's message starts `llmprovider:`,** and so does an
  `*APIError`'s, with its kind's text. A wrapped error keeps the prefix of
  the package or provider that wrapped it, such as `oauth:` or `gemini:`.
  (0015 D7 and its amendment of 2026-10-02)

## Middleware

- **R28. Cross-cutting behaviour wraps a provider:** `func(Provider) Provider`.
  Retry is `WithRetry(p, RetryPolicy{…})`, which honours `RetryAfter` and the
  error kind; `WithRetryNotify` retries the same, and gives a callback a
  `RetryNotice` before each wait; `WithStreamRetry` streams live, retrying
  only before the first event. Logging, metrics and rate limiting take the
  same shape. There are no hooks inside providers. (0015 D8; 0044 D1; 0051
  D1)

## State, logging and the environment

- **R29. Nothing mutable is exported at package level.** A catalog or a list of
  variable names is returned as a copy by a function. Error sentinels are
  the only exported package-level variables. (0015 D9)
- **R30. Library code reads no environment variable.** A caller that wants the
  environment calls an opt-in helper, such as `llmprovider.ModelProbesFromEnv()` or
  `auth.GrokFlowFromEnv()`: an exported function whose name ends `FromEnv`.
  The one other exemption is `http.ProxyFromEnvironment` on the default
  transport, which is the standard library's read. `internal/ambientcheck`
  fails on anything else. (0015 D9 and its amendments, 0016 D8)
- **R31. Logging goes only to the `*slog.Logger` given with `WithLogger`.**
  Without one, nothing is logged. The process-global logger is never used.
  (0015 D9)
- **R32. One default HTTP client per provider instance,** serving requests,
  listing and refresh. (0016 D8)

## Secrets

- **R33. A secret-bearing value redacts itself.** `Token`, `StaticToken`,
  `CommandToken`, an OAuth session and `wizard.Result` implement `String`,
  `GoString` and `slog.LogValuer` with a masked form; a session's ID token
  is masked as its tokens are. Code that needs the secret reads the field.
  (0016 D5; 0043 D1)
- **R34. Descriptors and status values carry no key material.** (0016 D5)
- **R35. An error message is redacted and bounded** before it is returned.
  (0015 D7)

## Registry and extension

- **R36. Providers are found through a `Registry`** of `Descriptor` and
  `Factory` pairs. `Register` refuses a duplicate id. (0015 D10)
- **R37. No global registry and no `init` side effects.**
  `providers.Default()` returns a new `Registry` each call. (0015 D10)
- **R38. Adding a provider is one package, one entry in `providers.Default()`'s
  list, and a passing `llmtest` run.** A provider outside this module needs
  no change here: its caller registers it in a `Registry`
  ([adding-a-provider.md](adding-a-provider.md)); so does a `chatcompat`
  provider, whose package is here but which `providers.Default()` does not
  list. (0015 D10; 0061 D4)
- **R39. `wizard` offers what its `Options.Registry` holds,** or
  `providers.Default()` when nil. (0015 D10)

## Context

- **R40. Every call that can block or reach the network takes a
  `context.Context` first** and stops when it is cancelled. Construction
  does neither, so it takes none. (0015 D5, D11)

## Documentation

- **R41. Every exported identifier has a doc comment,** and every package a
  package comment. `golangci-lint`'s `revive` rules check both. (0015 D13)
- **R42. Deprecation is `Deprecated:`** in the doc comment. The identifier stays
  until a major version. (0015 D13) *(2026-11-24: one recorded exception:
  v1.33.0 removes eight v1 names, by
  [0093-MADR](../decisions/0093-MADR-take-model-metadata-from-the-caller.md)
  D10 and its Q1.)*
- **R43. An API change updates the user docs in the same change:**
  [migrating-from-mcplib.md](migrating-from-mcplib.md) for anything that
  replaces an `mcplib` identifier, and `docs/architecture.md` for layout.
  (0015 D12, D13)

## Tests and compatibility

- **R44. Every provider passes `llmtest.Run`:** capabilities, refusal before
  the network, cancellation, status-to-kind classification, identity
  headers, `Response` invariants, and concurrent use under `-race`.
  (0015 D11)
- **R45. The request on the wire does not change** unless a record changes it.
  G-wire's goldens prove it for the recorded scenarios. (0015 D12)
- **R46. A moved or ported test keeps its meaning.** Its call syntax may change;
  what it asserts may not. (0015 D12)
- **R47. Coverage does not fall.** No package drops below its baseline at the
  end of 0002-PLAN Phase 7, and a new package holds at least 80 %. Each
  package is measured by its own tests, except a test-support package: one
  with no test files, imported only from tests, such as
  `llmprovider/internal/wirecase`. The tests that import it measure it.
  `scripts/coverage-floors.txt` holds the baselines and names the exceptions.
  (0015 D13, its amendment of 2026-10-01, whose `-coverpkg` measurement ended
  with S7b, and its amendment "the coverage of a test-support package")
- **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is
  under the compatibility promise,** and `apidiff` against the latest `v1.*`
  tag fails an incompatible change. (0015 D13) *(2026-11-24: one recorded
  exception, 0093 D10 and Q1: against `v1.32.0`, `make api-check` accepts
  exactly the eight removals that record names; any other incompatible
  change still fails.)*

## Go SDK practice

These rules are
[0048-MADR-adopt-go-sdk-best-practices.md](../decisions/0048-MADR-adopt-go-sdk-best-practices.md)'s
("0048 D*n*"), which measured the module against Go SDK practice.

- **R49. A caller can own the caches.** What the catalog and the providers
  keep in memory between calls, the listings and their models' metadata,
  the models that answered a probe and the models a
  provider learned refuse a forced tool choice (0034 D3), is held in a
  `catalog.Cache`: its zero value is ready, and it is safe for concurrent
  use. `catalog.WithCache` gives one to a listing or to a provider's `New`,
  and `providers.DefaultWith` to every built-in provider a `Registry`
  builds; without it, the process's cache is used. A new in-memory cache
  goes into `catalog.Cache`, never into a package variable a caller cannot
  replace. `Close` ends a `Cache`'s life: it stops new background work and
  waits for running work. The process's cache has no lifetime. Background
  work a new cache starts is counted on the cache, under the lock its start
  takes, so that `Close` waits for it. It holds no models.dev document: a
  caller hands one in with `catalog.WithMetadata`. (0048 D1 and its A5;
  0049 D3; 0071 D1; 0093 D2)
- **R50. Every public package has an example** that runs offline, over
  `llmtest.Fake` or `httptest`, with an `// Output:` comment, so that `go
  test` runs it. The package comments of `llmprovider`, `providers` and
  `wizard` show the simplest call in an indented block. (0048 D2)
- **R51. Every exported type's doc says whether it is safe for concurrent
  use,** in those words: a value, while no goroutine changes it; an
  interface a caller implements, what an implementation must be, and why; a
  type with state, as a test under `-race` measures it. (0048 D3)
- **R52. Every exported function or method that returns an error names the
  kinds it returns** in its doc: a sentinel, `*APIError`, a context's
  error, or a plain error. A test through that function shows each kind it
  names. A method of a standard interface (`Error`, `String`, `GoString`,
  `Format`, `LogValue`, `MarshalJSON`, `UnmarshalJSON`, `MarshalText`,
  `UnmarshalText`) is exempt. (0048 D4)
- **R53. Every release has a changelog section, and the six newest tags a
  GitHub Release.** `CHANGELOG.md` is in the Keep a Changelog form, with a
  section for each `v*` tag, newest first, dated by the tag's commit. A
  release's section is written before its tag, and CI's `release` job makes
  the tag's Release with that section as its notes. The release then
  deletes the Release of the tag that falls to seventh, keeping the tag and
  its section. (0048 D5, A4; 0055, amendment of 2026-11-10)
- **R54. The Go floor is `go 1.27.4`,** with no `toolchain` line, and CI
  builds with the version `go.mod` names. Go 1.27 is the least the module
  builds with: it uses `encoding/json/v2` and `encoding/json/jsontext`, in
  the standard library from Go 1.27.0. The patch, `.4`, carries the
  standard library's security fixes through GO-2026-6859 (`net/http`).
  The floor is raised with each Go security patch. (0048 D6, A1; 0055; 0077)
- **R55. A new export is justified.** A PLAN that adds an exported name
  lists each one in its record with the call site that needs it; `make
  api-check` lists every addition. (0048 D7)

## Thinking parameters

This rule is
[0062-MADR-learn-thinking-parameters-from-refusals.md](../decisions/0062-MADR-learn-thinking-parameters-from-refusals.md)'s
("0062 D*n*"), as its amendment "R56's scope" words it.

- **R56. No per-model name rule decides a thinking parameter:** a thinking
  shape, an effort or a budget. The model's facts and its learned refusals
  do, then the request (`wire.Ladder`, `wire.Learner`); a rule that a
  service's refusal cannot replace stays as a wire mapping, cited and listed
  under the provider's Degradations. (0062 D1, D2, D8)

## Output limits

This rule is
[0089-MADR-model-output-limit-discovery.md](../decisions/0089-MADR-model-output-limit-discovery.md)'s
D9 ("0089 D*n*").

- **R57. No fixed number decides an output limit a caller did not give:**
  the model's evidence does (`catalog.Limits`), and a limit is sent only as
  R6's default rule decides. A number with no source is listed with its
  provenance in the record that keeps it: a listing probe's 8192, which is
  sent as given (0088 ST26; 0089 DL53); the estimate's 3.5 characters a
  token, 4,800 characters an image and 4,096 spare, pi's and unmeasured
  (0088 ST87). A refusal that states a cap or a
  window resends a limit the SDK chose once at the stated value; a stated
  cap is kept for the process and reported in a `WarningLimitLearned` or,
  on a call that fails, its `*APIError`'s `Observed`, unless
  `WithoutLimitLearning` is given, and a caller's own limit is never
  resent. (0089 D4 to D6, D9, DL78, DL94; 0088 ST99)

## Checks

| Rule | Checked by | From |
| :--- | :--- | :--- |
| R2 (leaving the standard library) | `make dep-check` | S12 |
| R30–R31 | `internal/ambientcheck` | S10 |
| R33 | a test formatting each secret-bearing type with a planted secret | S4, S8 |
| R41 | `make lint` | now |
| R43 | `make parity-check` | S1; an empty cell fails from S11 |
| R44 | `llmtest.Run` per provider | S7 |
| R45 | G-wire (`TestWireGoldens`) | now |
| R47 | `make coverage-check` | S12 |
| R48 | `make api-check` | S12 |
| R50–R52 | `make docs-check` (`sdkaudit -check`) | 0048-PLAN Phase 6 |
| R53 | `make changelog-check` | 0048-PLAN Phase 5 |
| R54 | `go.mod`'s `go` line, which CI's `go-version-file` reads | 0048-MADR A1, 0055-MADR, 0077-MADR |
| R56 | `git grep -n -w` of each thinking rule 0062 removed, which 0062-PLAN Phase 6's record names: none found | 0062-MADR Confirmation |
| R57 | `git grep -n -w` of 32000, 16384 and 8192 in the module's non-test Go code: only the listing probes' given 8192 and `MaxTokens`' default, which decides nothing unless given; 0089-PLAN's merged phase records the run | 0089-MADR D9 |

An S*n* phase is 0015-PLAN's. The other rules are held by review against this
guide.
