---
status: observation
date: 2026-10-10
subject: the v1 exported API surfaces and the protocols the built-in providers speak
examines: "the v1.4.0 tree: llmprovider and its providers, auth, catalog, llmtest, wizard, internal/wire; docs/architecture.md and docs/guides/api-standards.md"
---
# The v1 SDK API Surfaces and the Protocols the Built-in Providers Speak

This report records findings. It decides nothing. The API shape it describes
is the one
[0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
decided, on the auth baseline of
[0016-MADR-provider-auth-and-support-baseline.md](../decisions/0016-MADR-provider-auth-and-support-baseline.md).
[0015-REPORT-sdk-api-surface-assessment.md](0015-REPORT-sdk-api-surface-assessment.md)
measured the imported `mcplib` surface before that redesign; this report
measures the v1 tree as it is.

Evidence is marked **(R)** read in the source or records. Live gateway
behaviour was not re-run for this report.

## Summary

`github.com/maccavelli/go-llmprovider-sdk` is a Go library for text
generation and the credentials that sign it. It ships no binary. It depends
on the standard library and `golang.org/x/term` (wizard only). The current
release named in `README.md` is `v1.4.0`. It requires Go 1.27.2. (R)

Every built-in provider implements one contract: `ID`, `Capabilities`, and
`Generate(ctx, *Request) (*Response, error)`. Tools, reasoning and
continuation are fields of `Request`. Callers see sealed `Item` types,
never vendor JSON. (R)

Five generation wire formats sit behind that contract. Ten provider ids
speak them. Credentials are API keys, OAuth 2.0 (authorization-code with
PKCE, and device-code), vendor-CLI files, and command-backed keys. Native
token streaming is in the contract; every built-in provider declares it
`Unsupported`. (R)

## Scope and method

- **The surface.** Package comments, exported types and constructors of
  `llmprovider`, `llmprovider/providers` and each `providers/<id>`,
  `llmprovider/auth`, `llmprovider/catalog`, `llmprovider/llmtest`, and
  `wizard`. Cross-checked against
  [architecture.md](../architecture.md) and
  [api-standards.md](../guides/api-standards.md).
- **Protocols.** The paths, headers and decoders in
  `llmprovider/internal/wire/` and in each provider package; the listing
  endpoints in `llmprovider/catalog`; the OAuth flows in `llmprovider/auth`.
- **Counts.** Ten built-in ids registered in `providers.Default()`. Five
  generation wires. Four shared wire packages plus Gemini Interactions in
  `providers/gemini`. (R)

## The exported packages

Compatibility-gated packages (`make api-check` against the latest `v1.*`
tag, R48):

| Package | Holds |
| :--- | :--- |
| `llmprovider` | the contract: `Provider`, `Request`, `Response`, `Item`, `Tool`, `Capabilities`, `Event`, `Usage`, typed identifiers, options, errors, `Registry`, `WithRetry`, `Stream`, `TokenSource` |
| `llmprovider/providers` | `Default()` and `New(id, opts...)` |
| `llmprovider/providers/<id>` | one built-in family: `New` (or `NewZen` / `NewGo`), its `Descriptor`, its scoped options |
| `llmprovider/auth` | OAuth sessions and flows, `TokenStore`, `FileTokenStore`, `VendorCLISession` |
| `llmprovider/catalog` | listing, static catalogs, ranking, search, metadata |
| `llmprovider/llmtest` | `Run`, the conformance suite, and `Fake` |
| `wizard` | `ConfigureLLM`, `Logout`, `Prompter`, `TextPrompter` |

Wire formats live under `llmprovider/internal/wire` and are unexported from
the module (R3). There is no `llmprovider/x/` experimental tree. (R)

---

## F1 — One generation method, features as request fields (R)

`llmprovider.Provider` is:

```go
type Provider interface {
    ID() ProviderID
    Capabilities() Capabilities
    Generate(ctx context.Context, req *Request) (*Response, error)
}
```

Optional interfaces, found by type assertion:

- `ModelLister` — `ListModels(ctx) ([]string, error)`
- `Streamer` — `Stream(ctx, *Request) iter.Seq2[Event, error]`

Package functions over any `Provider`:

- `GenerateText` — concatenate `MessageItem` texts
- `GenerateToolCall` — force the request's single tool and return its call
- `Stream` — a provider's `Streamer` when present; otherwise `Generate`'s
  result as events
- `WithRetry(p, RetryPolicy)` — middleware; it reports
  `NativeStreaming: Unsupported` even when the inner provider streams

`Request` fields: `Model`, `Instructions`, `Input []Item`, `Tools []Tool`,
`ToolChoice`, `Reasoning *Reasoning`, `MaxOutputTokens`,
`PreviousResponseID`. Zero means the provider default or "not requested".

`Response` fields: `ID`, `Model`, `Output []Item`, a typed `FinishReason`,
`Usage`. In `Usage`, `InputTokens` includes `CachedTokens` and
`OutputTokens` includes `ReasoningTokens`.

Sealed `Item` types: `MessageItem`, `FunctionCallItem`,
`FunctionCallOutputItem`, `ReasoningItem`. A new kind of content is a new
type in this package.

`ToolChoice` values: empty/`auto`, `none`, `required`, and
`ForceTool(name)`. `Effort` values: `low`, `medium`, `high`, `xhigh`.
`FinishReason` values: `stop`, `length`, `tool_calls`, `content_filter`; a
service value with no constant is kept as sent.

`Capabilities` is data: `Tools`, `ForcedToolChoice`, `Reasoning`,
`Continuation`, `NativeStreaming`, each `Unsupported`, `BestEffort` or
`Supported`. `Capabilities.Check` refuses an unsupported need with
`ErrUnsupported` before any network call, and an invalid value with
`ErrInvalidRequest`.

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D3, D4
- [api-standards.md](../guides/api-standards.md) R5–R13

## F2 — Construction, options, registry (R)

Each provider package exports `New(opts ...Option) (Provider, error)`. The
OpenCode family exports `NewZen` and `NewGo`. Construction takes no
context and makes no network call.

Common options live in `llmprovider`: `WithAPIKey`, `WithTokenSource`,
`WithModel`, `WithBaseURL`, `WithHTTPClient`, `WithLogger`,
`WithClientInfo`, `WithSessionID`, `WithMaxTokens`, `WithReasoning`,
`WithModelProbes`, `WithModelMetadataURL`, `WithoutModelMetadata`,
`For(id, …)`, `ModelProbesFromEnv`.

Provider-scoped options: `openai.WithStore`, `gemini.WithStore`,
`grok.WithStore`, `opencode.WithRoute`, `kilo.WithOrganization`,
`kilo.WithCapabilities`, `kilo.WithDataCollection`, `catalog.WithProfile`,
`catalog.WithKiloOrganization`. A foreign option is an error from `New`.

`providers.Default()` returns a new `Registry` of the ten built-ins, in
menu order. There is no global registry and no `init` registration. A
third-party provider registers a `Descriptor` and `Factory` on a
`Registry` of its caller's; `wizard` offers that menu when
`Options.Registry` is set.

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D5, D10
- [api-standards.md](../guides/api-standards.md) R14–R20, R36–R39

## F3 — Errors (R)

`*APIError` is the only structured error. It carries `Provider`, `Status`,
`Kind`, `Code`, a redacted and bounded `Message`, `RetryAfter` and
`Reason`. Kinds, tested with `errors.Is`:

- `ErrRateLimited`, with `ErrQuotaExhausted` beneath it
- `ErrAuthFailure`
- `ErrNotPermitted`
- `ErrInvalidRequest`, with `ErrContextOverflow` and `ErrIncomplete`
  beneath it
- `ErrProviderUnavailable`
- `ErrUnsupported` (also matches `errors.ErrUnsupported`)
- `ErrInvalidProvider`

Retryability is `Retryable()`. After a 401, `wire.Reauth` invalidates an
`InvalidatingSource` and sends once more. `ClassifyHTTPError` makes a 403
`ErrNotPermitted`, which is left unrenewed; a 403 carrying a measured
credential code would stay `ErrAuthFailure`, and no code has been measured,
so that table is empty. An `*APIError` built with `Status: 403` and no
`Kind` still matches `ErrAuthFailure`, through the status-only mapping kept
from before
[0012-MADR-conform-providers-to-reference-clients.md](../decisions/0012-MADR-conform-providers-to-reference-clients.md).

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D7
- [0028-MADR-heuristics-and-performance-from-the-research-pass.md](../decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md) D-H2

## F4 — Five generation wire formats (R)

All generation is HTTP POST with `Content-Type: application/json`, through
`internal/wire.Post`. The default client connects within 30 s, waits up to
300 s for headers, has no total timeout, idles a body at 300 s, and caps a
non-stream reply at 16 MiB. HTTP/2 is kept. `HTTP_PROXY`, `HTTPS_PROXY`
and `NO_PROXY` are honoured. Every request sends
`User-Agent: <app>/<version> (<os>; <arch>) go-llmprovider-sdk/<version>`.

| Wire | Path | Speakers | Package |
| :--- | :--- | :--- | :--- |
| OpenAI Responses | `POST {base}/responses` | `openai` (platform and ChatGPT Codex), `grok`, OpenCode `responses` | `internal/wire/responses` |
| OpenAI Chat Completions | `POST {base}/chat/completions` (Ollama: `{base}/v1/chat/completions`) | `huggingface`, `kilo`, `together`, `ollama`, OpenCode `chat_completions` | `internal/wire/chatcompletions` |
| Anthropic Messages | `POST {base}/messages` with `anthropic-version: 2023-06-01` | `claude`, OpenCode `messages` | `internal/wire/messages` |
| Gemini Interactions | `POST {base}/interactions` | `gemini` | `providers/gemini` (unique to that provider) |
| Gemini generateContent | `POST {base}/models/{id}:generateContent` | OpenCode `google` | `internal/wire/generatecontent` |

ChatGPT Codex answers `/responses` as an event stream. `responses.ReadStream`
buffers the whole stream into one `Response`. That path is internal decode.
The public `Streamer` interface is unused by every built-in provider.

OpenCode is a multi-protocol gateway. The route for a request is
`WithRoute`'s value, else the model's `provider.npm` in the metadata
document, else the embedded `routes_snapshot.json`, else a prefix
heuristic. A mismatch returns HTTP 500, classified as
`ErrProviderUnavailable`. Routes: `responses`, `messages`,
`chat_completions`, `google`.

Endpoints the packages document and leave unused:

- Hugging Face `POST {base}/responses`: undocumented; on auth failure it
  returns HTTP 200 with `status:"failed"`, which status-only classification
  would treat as success (measured 2026-08-29, package comment).
- Kilo `POST {base}/responses` and `/messages`: undocumented; Kilo's
  `/responses` puts reasoning where the Responses decoder does not read it
  (package comment).

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D2
- [0012-MADR-conform-providers-to-reference-clients.md](../decisions/0012-MADR-conform-providers-to-reference-clients.md)
- [0014-MADR-gemini-wire-fidelity.md](../decisions/0014-MADR-gemini-wire-fidelity.md)

## F5 — Ten provider ids and their capabilities (R)

Menu order of `providers.Default()`:

| Id | Default base | Wire | Tools | Forced tools | Reasoning | Continuation | Native streaming |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `gemini` | `https://generativelanguage.googleapis.com/v1beta` | Interactions | Supported | Supported | BestEffort | Supported with `gemini.WithStore(true)`; otherwise Unsupported | Unsupported |
| `openai` | `https://api.openai.com/v1`, or `https://chatgpt.com/backend-api/codex` for a ChatGPT session | Responses | Supported | Supported | Supported | Supported with an API key; Unsupported for ChatGPT | Unsupported |
| `claude` | `https://api.anthropic.com/v1` | Messages | Supported | BestEffort | Supported | Unsupported | Unsupported |
| `grok` | `https://api.x.ai/v1` | Responses | Supported | Supported | BestEffort | Supported | Unsupported |
| `opencode-zen` | `https://opencode.ai/zen/v1` | four routes | Supported | BestEffort | BestEffort | Unsupported | Unsupported |
| `opencode-go` | `https://opencode.ai/zen/go/v1` | four routes | Supported | BestEffort | BestEffort | Unsupported | Unsupported |
| `huggingface` | `https://router.huggingface.co/v1` | Chat Completions | Supported | BestEffort | BestEffort | Unsupported | Unsupported |
| `kilo` | `https://api.kilo.ai/api/gateway` | Chat Completions | Supported | BestEffort | BestEffort | Unsupported | Unsupported |
| `together` | `https://api.together.ai/v1` | Chat Completions | BestEffort | BestEffort | BestEffort | Unsupported | Unsupported |
| `ollama` | `http://localhost:11434` | Chat Completions | Supported | BestEffort | BestEffort | Unsupported | Unsupported |

Each provider's package comment lists its BestEffort degradations. Forced
tool choice with Claude reasoning is sent as `"auto"`. Ollama sends no
`tool_choice`. Hugging Face keeps `ToolChoiceNone` by sending no tools
([0023-MADR-huggingface-tool-choice-none.md](../decisions/0023-MADR-huggingface-tool-choice-none.md)).
Gemini continuation needs `WithStore(true)`. ChatGPT sessions send
`store: false` and drop `max_output_tokens`. OpenCode Go requires
`x-opencode-session`.

`WithBaseURL` is accepted by every constructor. Hugging Face, Kilo,
Together, Ollama and both OpenCode gateways advertise
`SupportsBaseURL: true` on their descriptors.

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D4
- [0023-MADR-huggingface-tool-choice-none.md](../decisions/0023-MADR-huggingface-tool-choice-none.md)
- [api-standards.md](../guides/api-standards.md) R10–R12, R17

## F6 — Auth protocols (R)

`TokenSource` returns a `Token` per request. `StaticToken` is an API key.
`CommandToken` runs a caller-named command (Claude Code `apiKeyHelper`
shape): no shell, 10 s timeout, 8 KiB cap, 5 min cache, shared among
concurrent callers. Sessions are `TokenBearer`.

Default request headers when `Token.Header` is empty:

| Service | Header |
| :--- | :--- |
| OpenAI, Grok, Kilo, Hugging Face, Together | `Authorization: Bearer` |
| Claude | `x-api-key` |
| Gemini | `x-goog-api-key` (never in the URL) |
| OpenCode | the header of the request's route |
| Ollama | none; a token that names a `Header` is sent there |

OAuth, in `llmprovider/auth`:

- `LoginBrowserOAuth` — authorization code with PKCE on a loopback
  redirect. OpenAI and Grok.
- `StartDeviceOAuth` / `LoginDeviceOAuth` — device-code flow. OpenAI,
  Grok, and Kilo. Kilo's approved token never refreshes.
- Every login verifies `id_token` against the issuer's JWKS (signature,
  issuer, audience, expiry, and the Grok nonce) before any claim is used.
- Refresh is five minutes before expiry, or at half-life when shorter;
  after a 401 it runs once. OpenAI refresh is JSON; Grok refresh is form
  body plus discovery.
- `RevokeOAuthSession` ends OpenAI and Grok sessions.
- Default client ids are the vendor CLIs' public clients (Codex for
  OpenAI, Grok CLI for xAI). `OAuthFlowOptions.ClientID` overrides them.

`FileTokenStore` keeps one `0600` JSON file per provider, with an OS
exclusive lock across a refresh so a refresh token is spent once.
`VendorCLISession` reads Codex `~/.codex/auth.json` or Grok
`~/.grok/auth.json` on every request and never refreshes it.

Kilo and OpenCode send a service token when the caller gives none: Kilo
`"anonymous"`, OpenCode `"public"`. Free models answer; paid ones fail
with a typed error.

Descriptor `AuthMethods` offered by the wizard:

- OpenAI and Grok: API key, browser OAuth, device code, paste
  (`AuthTokenStdin`), vendor CLI import.
- Kilo: API key and device code.
- Claude, Gemini, Hugging Face, Together, OpenCode: API key via
  `RequiresAPIKey` (no explicit `AuthMethods` list).
- Ollama: local, no key.

Claude and Gemini refuse an `OAuthSession` or `VendorCLISession` at `New`
with `ErrUnsupported`
([0016-MADR-provider-auth-and-support-baseline.md](../decisions/0016-MADR-provider-auth-and-support-baseline.md)
D10).

**Bears on:**

- [0016-MADR-provider-auth-and-support-baseline.md](../decisions/0016-MADR-provider-auth-and-support-baseline.md)
- [0006-MADR-subscription-auth-for-llm-providers.md](../decisions/0006-MADR-subscription-auth-for-llm-providers.md)
- [0017-MADR-together-provider-and-auth-extensions.md](../decisions/0017-MADR-together-provider-and-auth-extensions.md) D2

## F7 — Listing, ranking and the wizard (R)

`catalog.List(ctx, id, src, opts...)` lists a built-in id within 10 s and
returns a `Catalog` (`Recommended`, `Usable`, `Err`). An unknown id fails
with `ErrUnsupported` before options are read.

Listing HTTP:

- OpenAI (API key), Claude, Gemini, Grok: the vendor's models list, with
  pagination, plus one billed generation probe per candidate unless
  `WithModelProbes(false)`.
- OpenAI (ChatGPT session): the Codex catalog; no probes.
- Ollama: native `GET {base}/api/tags`; `/v1/models` is unused. Probes
  unless disabled. No static catalog.
- OpenCode Zen/Go, Hugging Face, Kilo, Together: gateway or router
  catalogs; no probes.

Open catalogs (Kilo, both OpenCode ids, Hugging Face, Together) rank by
`catalog.Profile` (`ProfileUtility`, `ProfileCapable`) using
models.dev-format metadata from `https://models.opencode.ai/api.json`,
cached ten minutes with ETag revalidation. `catalog.Search` matches a
query against a list.

The listing keeps text-chat models. Catalog filters drop embeddings,
image-generation, audio-only, TTS and several vision-only ids. A
vision-language model with text+image *input* and text *output* is
admitted in ranking metadata
([0007-MADR-live-catalog-model-search.md](../decisions/0007-MADR-live-catalog-model-search.md)
§1b). Generation `Item` types
still carry text only.

`wizard.ConfigureLLM(ctx, Prompter, Options) (Result, error)` walks
provider → base URL → credential → model → fallbacks. It writes no
configuration. `Prompter` is the rendering seam. A `CredOAuth` `Result`
holds no token; the session lives in `Options.TokenStore`. `Logout`
revokes (OpenAI, Grok) and deletes the stored session.

`llmtest.Run` is the conformance suite every built-in provider passes:
capabilities, pre-network refusal, cancellation, status-to-kind,
identity headers, `Response` invariants, concurrent use under `-race`.
Optional harness checks: `Fidelity`, `Garbled`, `Truncated`,
`StrictTools`, `ReasoningCut`, `AuthFailure`, `NotPermitted`.

**Bears on:**

- [0005-MADR-canonicalize-llm-provider-configuration.md](../decisions/0005-MADR-canonicalize-llm-provider-configuration.md)
- [0007-MADR-live-catalog-model-search.md](../decisions/0007-MADR-live-catalog-model-search.md)
- [0009-MADR-use-case-aware-default-model-ranking.md](../decisions/0009-MADR-use-case-aware-default-model-ranking.md)
- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D11

## F8 — Surfaces with no type on `Request` or `Item` (R)

The generation contract as implemented covers text messages, function
tools, reasoning traces, token usage, and (where the service stores
state) continuation by response id.

These vendor capabilities have no field on `Request`, no `Item` type, and
no built-in provider method:

- Native `Streamer` implementations (the interface exists; every built-in
  `NativeStreaming` is `Unsupported`)
- Image, audio or file content as input or output
- Embeddings, TTS, STT, batch, Assistants, Live/WebSocket
- MCP as a wire protocol (the module exists so callers import neither
  `mcplib` nor the MCP go-sdk)
- Structured-output / JSON-schema response format (tools are the
  structured path)
- Sampling knobs: temperature, top_p, stop sequences, seed, penalties

A new one of these is a new field on `Request` or `Response`, or a new
sealed `Item` type, plus a MADR (R5, R9).

Streaming is the largest gap between contract and implementation.
ChatGPT already consumes SSE inside `ReadStream`. Exposing that through
`Streamer` would be an implementation change with no public API change
(R13). Other wires would need their own event-stream readers.

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D3, D4
- [architecture.md](../architecture.md) "What is not here"

## F9 — How a new protocol would plug in (R)

A provider in another module implements `Provider`, optionally
`ModelLister` and `Streamer`, constructs with `ResolveOptions`, registers
on a `Registry`, and passes `llmtest.Run`
([adding-a-provider.md](../guides/adding-a-provider.md)).

A wire format shared by more than one provider belongs under
`llmprovider/internal/wire/<format>`. A format unique to one vendor stays
in that provider package (Gemini Interactions is the existing example).
The public generation API does not grow methods when a protocol is added.
OpenCode already multiplexes four wires behind one `Generate`.

`WithBaseURL` on a Chat Completions or Responses speaker reaches an
OpenAI-compatible proxy under that speaker's capability caveats, without
a new package.

**Bears on:**

- [0015-MADR-canonical-sdk-api-and-module-layout.md](../decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) D2, D10
- [api-standards.md](../guides/api-standards.md) R3, R5, R13, R38
- [adding-a-provider.md](../guides/adding-a-provider.md)

## Not verified

- Live behaviour of each streaming endpoint, beyond the ChatGPT SSE
  decoder's unit tests and goldens.
- Whether every wire reports `Usage` on every route in production; the
  decoders map the fields they know.
- Third-party providers registered outside this module.
