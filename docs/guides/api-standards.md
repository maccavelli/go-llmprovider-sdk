# API standards

The conventions every exported API in this module follows. The guide is
**normative**: an API change that breaks a rule is wrong, not a judgement call.
It changes only by amending the decision a rule cites. Each rule names that
decision, in
[0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
("0015 D*n*") or
[0016-MADR-provider-auth-and-support-baseline.md](../decisions/0016-MADR-provider-auth-and-support-baseline.md)
("0016 D*n*").

**This is the target, not a description of today's tree.** The code still has
the API it was imported with from `mcplib`. It is brought to these rules phase
by phase by
[0015-PLAN-canonical-sdk-api-and-module-layout.md](../decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md),
which finalises this guide against the built tree in its phase S11. For what
exists now, read [architecture.md](../architecture.md).

## Packages

- **R1. One module.** Everything is in `github.com/maccavelli/go-llmprovider-sdk`.
  A second module needs its own MADR and a dependency that justifies it.
  (0015 D2)
- **R2. A package holds one concern**, and imports only what the table allows.
  Only `wizard` may leave the standard library, and then only for
  `golang.org/x/term`. (0015 D2, D13)

  | Package | Holds | May import |
  | :--- | :--- | :--- |
  | `llmprovider` | the contract: `Provider`, `Request`, `Response`, `Item`, `Tool`, `Capabilities`, `Event`, `Usage`, typed identifiers, options, errors, `Registry`, `Descriptor`, retry middleware, `TokenSource` and `Token` | the standard library |
  | `llmprovider/auth` | OAuth sessions and flows, `TokenStore`, `FileTokenStore`, `StaticToken`, `VendorCLISession` | `llmprovider` |
  | `llmprovider/catalog` | static catalogs, model metadata, ranking, search, labels, profiles, the curated `Catalog` | `llmprovider` |
  | `llmprovider/providers/<id>` | one provider or gateway family: `New`, its options, its `Descriptor` | `llmprovider`, `auth`, `catalog`, internal packages |
  | `llmprovider/providers` | `Default()` and `New(id, opts...)` | the provider packages |
  | `llmprovider/llmtest` | the conformance suite and `Fake` | `llmprovider` |
  | `llmprovider/internal/wire/...` | the wire formats | `llmprovider` |
  | `llmprovider/internal/transport` | HTTP helpers, identity headers, `Retry-After`, error classification | `llmprovider`, `internal/redact` |
  | `wizard` | the configuration flow, over a `Registry` | the above, `golang.org/x/term` |
  | `internal/redact` | redaction | the standard library |

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
  `PreviousResponseID`. A field's zero value means "the provider's default"
  or "not requested". (0015 D3)
- **R7. `Response` fields.** `ID`, `Model`, `Output []Item`, a typed
  `FinishReason`, and `Usage`, which is zero when the service reports none.
  (0015 D3, D4)
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
  `ForcedToolChoice`, `Reasoning`, `Continuation` and `NativeStreaming`,
  each `Unsupported`, `BestEffort` or `Supported`. (0015 D4)
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
  `New(opts ...llmprovider.Option) (llmprovider.Provider, error)`. `New`
  takes no context and makes no network call. (0015 D5)
- **R15. Credentials are a `TokenSource`.** `WithTokenSource`, or `WithAPIKey`
  as shorthand for a `StaticToken`. There are no `*WithSource` twins.
  (0015 D5, 0016 D2)
- **R16. A provider applies the token as `Token` describes it.** An empty
  `Header` means the service's own header and scheme. A non-empty one
  overrides it: a `TokenBearer` is sent as `Bearer <value>`, a `TokenAPIKey`
  or an untyped token bare. A source names a `Header` only when its caller
  set one. `New` refuses a source whose kind the service does not accept.
  (0016 D2, A6)
- **R17. Options live where they apply.** The common ones (model, HTTP client,
  base URL, logger, client identity, session id, max tokens, default
  reasoning) are in `llmprovider`. A provider-specific one is in its
  provider's package, for example `kilo.WithOrganization`. (0015 D5)
- **R18. A foreign option is an error from `New`,** never ignored. (0015 D5)
- **R19. Configuration structs are unexported.** No exported `ProviderConfig` or
  `ApplyOptions`. (0015 D5)
- **R20. After `New`, a provider is immutable and safe for concurrent use.**
  (0015 D9)

## Identifiers

- **R21. Closed sets are named types.** `ProviderID`, `Role`, `Effort`,
  `FinishReason`, `ToolChoice`, `AuthMethodID` and `TokenType` are named
  types. Each canonical value is a constant whose name starts with a short
  prefix naming its type (`RoleUser`, `EffortHigh`), and a value `mcplib`
  already named keeps that name (`ProviderGemini`, `AuthAPIKey`). (0015 D6)
- **R22. Provider ids are the models.dev registry keys** already in use.
  (0015 D6)
- **R23. An invalid value is `ErrInvalidRequest`** from `New` or `Generate`.
  (0015 D6)

## Errors

- **R24. `*APIError` is the only structured error.** It carries `Provider`,
  `Status`, `Kind`, `Code`, a redacted and bounded `Message`, `RetryAfter`
  and `Reason`. (0015 D7)
- **R25. Test errors by kind with `errors.Is`.** Each error unwraps to one of
  `ErrRateLimited` (with `ErrQuotaExhausted` beneath it), `ErrAuthFailure`,
  `ErrNotPermitted`, `ErrInvalidRequest` (with `ErrContextOverflow` beneath
  it), `ErrProviderUnavailable`, `ErrIncomplete`, `ErrUnsupported` or
  `ErrInvalidProvider`. A new failure mode maps to an existing kind before it
  earns a new one. (0015 D7 and its amendment of 2026-09-30)
- **R26. Retryability is `Retryable()`.** (0015 D7)
- **R27. Every error message starts `llmprovider:`.** (0015 D7)

## Middleware

- **R28. Cross-cutting behaviour wraps a provider:** `func(Provider) Provider`.
  Retry is `WithRetry(p, RetryPolicy{…})`, which honours `RetryAfter` and the
  error kind. Logging, metrics and rate limiting take the same shape. There
  are no hooks inside providers. (0015 D8)

## State, logging and the environment

- **R29. Nothing mutable is exported at package level.** A catalog or a list of
  variable names is returned as a copy by a function. Error sentinels are
  the only exported package-level variables. (0015 D9)
- **R30. Library code reads no environment variable.** A caller that wants the
  environment calls an opt-in helper, such as `catalog.OptionsFromEnv()` or
  `auth.GrokFlowFromEnv()`. The one exemption is `http.ProxyFromEnvironment`
  on the default transport, which is the standard library's read.
  (0015 D9 and its amendment, 0016 D8)
- **R31. Logging goes only to the `*slog.Logger` given with `WithLogger`.**
  Without one, nothing is logged. The process-global logger is never used.
  (0015 D9)
- **R32. One default HTTP client per provider instance,** serving requests,
  listing and refresh. (0016 D8)

## Secrets

- **R33. A secret-bearing value redacts itself.** `Token`, `StaticToken`, an
  OAuth session and `wizard.Result` implement `String`, `GoString` and
  `slog.LogValuer` with a masked form. Code that needs the secret reads the
  field. (0016 D5)
- **R34. Descriptors and status values carry no key material.** (0016 D5)
- **R35. An error message is redacted and bounded** before it is returned.
  (0015 D7)

## Registry and extension

- **R36. Providers are found through a `Registry`** of `Descriptor` and
  `Factory` pairs. `Register` refuses a duplicate id. (0015 D10)
- **R37. No global registry and no `init` side effects.**
  `providers.Default()` returns a new `Registry` each call. (0015 D10)
- **R38. Adding a provider is one package, one line in `providers.Default()`,
  and a passing `llmtest` run.** (0015 D10)
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
  until a major version. (0015 D13)
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
  end of 0002-PLAN Phase 7, and a new package holds at least 80 %.
  (0015 D13)
- **R48. From `v1.0.0`, every non-internal package outside `llmprovider/x/` is
  under the compatibility promise,** and `apidiff` against the latest `v1.*`
  tag fails an incompatible change. (0015 D13)

## Checks

| Rule | Checked by | From 0015-PLAN phase |
| :--- | :--- | :--- |
| R2 (leaving the standard library) | `make dep-check` | S12 |
| R30–R31 | `internal/ambientcheck` | S10 |
| R33 | a test formatting each secret-bearing type with a planted secret | S4, S8 |
| R41 | `make lint` | now |
| R43 | `make parity-check` | now; empty cells fail from S11 |
| R44 | `llmtest.Run` per provider | S7 |
| R45 | G-wire (`TestWireGoldens`) | now |
| R47 | `make coverage-check` | S12 |
| R48 | `make api-check` | S12 |

The other rules are held by review against this guide.
