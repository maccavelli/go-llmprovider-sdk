# go-llmprovider-sdk

A Go library for calling LLM providers and authenticating to them: API keys,
subscription OAuth (browser and device-code login, refresh, revocation), a
token store, model discovery, and an interactive configuration wizard. It
depends on the standard library and `golang.org/x/term` only, and ships no
binary.

Module: `github.com/maccavelli/go-llmprovider-sdk`

**Documentation:** [docs/](docs/README.md)

## Install

```bash
go get github.com/maccavelli/go-llmprovider-sdk
```

It requires Go 1.27.1.

## Packages

| Package | For |
| :--- | :--- |
| `llmprovider` | the contract: `Provider`, `Request`, `Response`, items, options, errors, `Registry`, `WithRetry`, `Stream`, `TokenSource` |
| `llmprovider/providers` | `Default()`, a registry of every built-in provider, and `New(id, opts...)` |
| `llmprovider/providers/<id>` | one built-in provider each: `openai`, `claude`, `gemini`, `grok`, `opencode`, `kilo`, `huggingface`, `together`, `ollama` |
| `llmprovider/auth` | OAuth sessions, browser and device-code logins, vendor CLI sessions, token stores |
| `llmprovider/catalog` | model listing, static catalogs, ranking and search |
| `llmprovider/llmtest` | the conformance suite for a provider, and `Fake` for tests |
| `wizard` | `ConfigureLLM`, the interactive configuration flow |

[docs/architecture.md](docs/architecture.md) describes how they fit
together.

## Quick start

```go
p, err := providers.New(llmprovider.ProviderClaude,
    llmprovider.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
    llmprovider.WithModel("claude-sonnet-5"))
if err != nil {
    return err
}
p = llmprovider.WithRetry(p, llmprovider.RetryPolicy{MaxAttempts: 3})
text, err := llmprovider.GenerateText(ctx, p, &llmprovider.Request{
    Input: []llmprovider.Item{llmprovider.MessageItem{Text: "Say hello in one word."}},
})
if errors.Is(err, llmprovider.ErrRateLimited) {
    return fmt.Errorf("try later: %w", err)
}
```

## LLM providers

Every provider implements one contract: `ID`, `Capabilities` and
`Generate(ctx, *Request)`. Tools, reasoning and continuation are fields of
`Request`. The sealed `Item` types (`MessageItem`, `FunctionCallItem`,
`FunctionCallOutputItem`, `ReasoningItem`) let callers switch on types
instead of parsing vendor JSON. `GenerateText` and `GenerateToolCall` are
functions over any provider, and `Stream` streams from every one.

- **Ids:** `openai`, `claude`, `gemini`, `grok`, `opencode-zen`,
  `opencode-go`, `huggingface`, `kilo`, `together` and `ollama`. Build one
  with `providers.New(id, opts...)`, or with its package's own `New`.
- **Credentials** are a `TokenSource`: `WithAPIKey(key)`, or
  `WithTokenSource` for an `auth.OAuthSession`, a vendor CLI login or a
  `CommandToken`.
  - A ChatGPT session uses `chatgpt.com/backend-api/codex`, never the
    API-key endpoint at `api.openai.com`.
  - A Grok session and a Grok key both use `api.x.ai/v1`.
  - Claude and Gemini take an API key only. Claude's is read from
    `ANTHROPIC_API_KEY`, when the caller asks for the environment.
  - Ollama needs no key. Kilo and OpenCode accept none: they then send the
    service's anonymous or public token, which free models answer and paid
    ones refuse with a typed error.
- **Errors.** A failed response is an `*APIError`, carrying the service's
  code and message (redacted, at most 512 bytes). Test it by kind with
  `errors.Is`:
  - `ErrRateLimited`, with `ErrQuotaExhausted` beneath it;
  - `ErrAuthFailure` and `ErrNotPermitted`;
  - `ErrInvalidRequest`, with `ErrContextOverflow` and `ErrIncomplete`
    beneath it;
  - `ErrProviderUnavailable` and `ErrUnsupported`.

  A truncated answer is an error of kind `ErrIncomplete`. A text answer cut
  by the token limit keeps its text and sets `Response.FinishReason`.
  `WithRetry(p, RetryPolicy{…})` retries only what can succeed later,
  honouring `Retry-After`. After a 401, an OAuth session refreshes and a
  `CommandToken` reruns its command, and the request is sent once more.
- **Transport.** The default client waits up to 300 seconds for response
  headers and 330 seconds in all; set a context deadline for less. It honours
  `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`. Every request sends
  `User-Agent: <name>/<version> (<os>; <arch>) go-llmprovider-sdk/<version>`.
  `WithClientInfo(name, version)` names the application, and
  `WithSessionID(id)` sets the conversation id that OpenCode
  (`x-opencode-session`) and Kilo (`X-KiloCode-TaskId`) receive.
- **No ambient state.** The library reads no environment variable by
  itself. Pass `catalog.OptionsFromEnv()`, `auth.GrokFlowFromEnv()` or
  `llmprovider.ModelProbesFromEnv()` to opt in, and give logging a logger
  with `WithLogger`.
- **Listing.** `catalog.List(ctx, id, src, opts...)` lists a provider's
  models within 10 seconds, curated against its static catalog, and
  `catalog.Search` matches a query against them. A provider's own
  `ListModels` returns its listing. By default OpenAI (API key), Claude,
  Gemini, Grok and Ollama send one billed generation to each candidate and
  keep those that answer; `WithModelProbes(false)` turns that off. The open
  catalogs rank with `catalog.WithProfile`, from models.dev-format metadata
  that `WithoutModelMetadata` turns off.

## The configuration wizard

`wizard.ConfigureLLM(ctx, prompter, opts)` asks for a provider, its
endpoint, a credential and a model, through a `Prompter` you implement or
`wizard.NewTextPrompter`. It never writes configuration and never logs a
key.

- A sign-in saves its session to `Options.TokenStore` and returns a
  `CredOAuth` `Result` with no token; the caller loads it from the store.
  Without a `TokenStore`, only the API key is offered.
- With `Options.Discover`, it lists the provider's models once, within
  10 seconds (`Options.DiscoverLimit` can shorten that, not extend it), and
  asks for a search before each model menu. A blank search shows the
  recommendations. A glob such as `kilo-auto/*` or `*llama*` matches whole
  ids, and other queries match loosely. Scripts that drive the wizard answer
  one extra, blank, line before each model and fallback selection.
- The menu is `Options.Registry`, every built-in provider when nil, so a
  provider of your own is offered once it is registered
  ([adding-a-provider.md](docs/guides/adding-a-provider.md)).

## Status

There is no release yet. The module requires Go 1.27.1.

- The provider and wizard code was imported, with its history, from
  `mcplib` `v1.6.0` and re-homed here
  ([0002-PLAN](docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md),
  Phase 4).
- Its API is the v1 API that
  [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
  decides, on the provider-auth baseline of
  [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md).
  [0015-PLAN](docs/decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md)
  has built it, and CI enforces its standards. The first release, `v1.0.0`,
  follows the live identity gates of
  [0002-PLAN](docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md)
  Phase 8.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| know why the code is moving here from `mcplib`, and how | [0002-MADR](docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) |
| know why the v1 API is shaped as it is | [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) |
| change or add an exported API | [api-standards.md](docs/guides/api-standards.md) |
| add a provider, in this module or my own | [adding-a-provider.md](docs/guides/adding-a-provider.md) |
| move code from `mcplib`'s `llmprovider` or `wizard` | [migrating-from-mcplib.md](docs/guides/migrating-from-mcplib.md) |
| know how provider auth is built, and what it takes from `magic-cli-remote` | [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
