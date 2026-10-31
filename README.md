# go-llmprovider-sdk

A Go library for calling LLM providers and authenticating to them: API keys,
subscription OAuth (browser and device-code login, refresh, revocation),
OpenAI's Sign in with ChatGPT for agents, a token store, model discovery, and
an interactive configuration wizard. It
depends on the standard library and `golang.org/x/term` only, and ships no
binary.

Module: `github.com/maccavelli/go-llmprovider-sdk`

**Documentation:** [docs/](docs/README.md)

## Install

```bash
go get github.com/maccavelli/go-llmprovider-sdk
```

It requires Go 1.27.3.

## Packages

| Package | For |
| :--- | :--- |
| `llmprovider` | the contract: `Provider`, `Request`, `Response`, items, options, errors, `Registry`, `WithRetry`, `WithRetryNotify` and `WithStreamRetry`, `Stream`, `TokenSource` |
| `llmprovider/providers` | `Default()`, a registry of every built-in provider, and `New(id, opts...)` |
| `llmprovider/providers/<id>` | one built-in provider each: `openai`, `claude`, `gemini`, `grok`, `opencode`, `kilo`, `huggingface`, `together`, `ollama` |
| `llmprovider/auth` | OAuth sessions, browser and device-code logins, vendor CLI sessions, token stores |
| `llmprovider/catalog` | model listing, model facts, the menu, static catalogs, ranking and search |
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
  - A ChatGPT plan session, from `auth.LoginChatGPTPlan`, uses
    `api.openai.com/v1`, billed to the user's ChatGPT plan (see below).
  - A Grok session and a Grok key both use `api.x.ai/v1`.
  - Claude and Gemini take an API key only. Claude's is read from
    `ANTHROPIC_API_KEY`, when the caller asks for the environment. A
    Claude key that is not scoped to a workspace must name one on every
    request: pass `claude.WithWorkspace(id)`, which its listing and
    probes use too, or `catalog.WithClaudeWorkspace(id)` to a listing
    alone. The ID is in Anthropic's Console, under Settings, Workspaces.
    The wizard asks for it when the key needs one, and returns it in
    `Result.Workspace`.
  - Ollama needs no key. Kilo and OpenCode accept none: they then send the
    service's anonymous or public token, which free models answer and paid
    ones refuse with a typed error.
- **Errors.** A failed response is an `*APIError`, carrying the service's
  code and message (redacted, at most 512 bytes). Test it by kind with
  `errors.Is`:
  - `ErrRateLimited`, with `ErrQuotaExhausted` beneath it;
  - `ErrAuthFailure` and `ErrNotPermitted`;
  - `ErrInvalidRequest`, with `ErrContextOverflow`, `ErrIncomplete`,
    `ErrWorkspaceRequired` and `ErrWorkspaceNotFound` beneath it;
  - `ErrProviderUnavailable` and `ErrUnsupported`.

  A truncated answer is an error of kind `ErrIncomplete`. A text answer cut
  by the token limit keeps its text and sets `Response.FinishReason`; one
  cut while the model was still reasoning, with no text, is `ErrIncomplete`.
  An empty answer is `ErrIncomplete` too, with the service's reason, such
  as a refusal's `content_filter`, in `APIError.Reason`.
  `WithRetry(p, RetryPolicy{…})` retries only what can succeed later,
  honouring `Retry-After` with a little jitter, and keeps `p`'s
  `ListModels`. A reply lost while it was being read, which may have been
  billed, is retried once at most. A wait longer than the context's deadline
  is not begun: the error is returned at once, so the caller can
  reschedule. After a 401, an OAuth session refreshes and a
  `CommandToken` reruns its command, and the request is sent once more.
  `WithRetryNotify(p, policy, notify)` retries the same, and calls `notify`
  before each wait with a `RetryNotice`: the attempt, the wait, whether the
  service asked for it, and the error. Use it to tell your user "rate
  limited, retrying in 4 s"; `notify` must not block. Both buffer a
  stream, so that it can be retried whole. `WithStreamRetry(p, policy,
  notify)` streams text as it arrives instead: it retries a failure before
  the first event as they do, and after it ends the stream with the error,
  unretried, so the caller drops the turn.
- **Transport.** The default client connects within 30 seconds and waits
  up to 300 seconds for response headers. It has no total timeout: a reply
  may take as long as it keeps sending, and is given up after 300 seconds
  with no data, as the ChatGPT client does. Set a context deadline for
  less. It honours
  `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`. Every request sends
  `User-Agent: <name>/<version> (<os>; <arch>) go-llmprovider-sdk/<version>`.
  `WithClientInfo(name, version)` names the application, and
  `WithSessionID(id)` sets the conversation id that OpenCode
  (`x-opencode-session`) and Kilo (`X-KiloCode-TaskId`) receive.
- **Prompt caching.** `Request.Cache` (`llmprovider.CacheShort`, or
  `CacheLong` for Anthropic's one-hour lifetime) asks the service to cache
  the prompt. Claude gets Anthropic's `cache_control` markers, OpenAI, xAI
  and Together a `prompt_cache_key`, and Kilo's Claude models the top-level
  marker. Without it, a request is sent as before. `Usage` reports the
  reads and writes.
- **Deferred tools.** A `Tool` with `Deferred` stays out of the model's
  context until the caller's search (the tool with `Search`) finds it, and
  a `ToolLoadItem` after that result loads it. Claude 4.5 and later,
  directly or through OpenCode, get Anthropic's `defer_loading` and
  `tool_reference`. OpenAI from GPT-5.4, and every ChatGPT session model,
  get a client `tool_search`. Grok runs them through one fixed `use_tool`
  wrapper. Every other wire appends each tool as it is loaded. A response
  that did not use the service's own form carries
  `WarningDeferralEmulated`.
- **No ambient state.** The library reads no environment variable by
  itself. Pass `catalog.OptionsFromEnv()`, `auth.GrokFlowFromEnv()` or
  `llmprovider.ModelProbesFromEnv()` to opt in, and give logging a logger
  with `WithLogger`.
- **Listing.** `catalog.List(ctx, id, src, opts...)` lists a provider's
  models within 10 seconds, curated against its static catalog, and
  `catalog.Search` matches a query against them. A provider's own
  `ListModels` returns its listing. The listing is the full curated catalog,
  every known model that works, ranked, with the best six first.
  `catalog.WithLimit(catalog.MaxListed)`, or `wizard.Options.ModelLimit`,
  keeps it to those six. By default OpenAI (API key), Claude, Gemini, Grok
  and Ollama send one billed generation to each of the first six and drop
  those that do not answer; `WithModelProbes(false)` turns that off.
- **Model facts and the menu.** What a request leaves to the provider, the
  model's facts decide: its thinking shape and budget bounds, the efforts
  or levels it takes, and an output limit that cuts the default. The facts
  are the caller's `catalog.WithModelLayer`, then the provider's listing,
  then models.dev, and a request reads only what is already cached.
  Without them the request decides: its effort is sent as asked, and a
  `Budget` asks for a fixed budget. A model's refusal of an effort or a
  thinking shape outranks both: it is remembered for the process, the
  request is sent again at the nearest setting the model takes, and the
  response carries a `WarningReasoningLearned`, which
  `WithReasoningRefusals` passes back on the next start.
  `llmprovider.ProbeReasoning` finds a model's refusals before use, with one
  billed generation.
  - A `Catalog`'s `Menu` holds every usable model with its facts, its rank
    and its caveats: deprecated, no tools, no text output, expiring, or
    failing at request. A caveat sinks a model and never removes it.
    `Recommended` is the band, then the other models without a caveat,
    newest first.
  - `catalog.ProfileAgent` suits long sessions with prompt caching: it
    orders `ProfileCapable`'s six by the price of a cached turn.
    `catalog.WithWorkload`, or `catalog.ObservedWorkload` over your own
    responses' `Usage`, sets the shares that price uses.
  - After a transient failure (a timeout, a 5xx, a rate limit or a network
    error), a listing read in the last 24 hours is served again, with
    `Source` `ListStale`, its `FetchedAt`, and the error in `Err`. An
    authentication failure never serves it, and Ollama, which lists every
    time, returns its error. `catalog.WithLastGood` hands in a listing kept
    from an earlier run.
  - `catalog.ListRaw` returns a provider's listing exactly as it was sent,
    every page and every entry, with none of the SDK's curation.
- **Ranking.** The recommended models of every provider but Ollama are
  ranked by use case, from each model's facts. The default profile,
  `catalog.ProfileUtility`, suits short, frequent tasks such as commit
  messages. It ranks recent, paid, reasoning-capable models first, at most
  two per vendor or model family (the `gpt-*` families count as one), and
  leaves out those its catalog marks as non-reasoning, free, expiring or
  preview. When fewer than six qualify, the list is filled from the curated
  order, which can add such a model, but never one a request would fail on:
  the `-contributor` models, and on OpenCode Go the region-gated DeepSeek
  ones. Kilo's `kilo-auto/*` tiers are never added under it either. A search
  still finds all of these.
  `catalog.WithProfile(catalog.ProfileCapable)`, or `wizard.Options.Profile`,
  ranks the strongest first instead. Kilo ranks from its own listing. The
  others read `https://models.opencode.ai/api.json`, cached for ten minutes
  and refreshed in the background after that; a failed fetch is not retried
  for a minute. When every model in a listing has a caveat, the
  recommendations are empty and the wizard asks for a search; a blank search
  then offers the saved model, if any, or a typed id, never the same prompt
  again. `WithModelMetadataURL` points
  elsewhere, and `WithoutModelMetadata` turns the fetch off. That restores
  the curated order for the four gateways, and the name rules for OpenAI,
  Claude, Gemini and Grok. `catalog.OptionsFromEnv()` maps
  `LLMPROVIDER_MODELS_METADATA_URL` and
  `LLMPROVIDER_DISABLE_MODELS_METADATA=1` onto those options. Search still
  covers every usable model. For commit-message-sized work, ask for the
  profile's effort and retry:

  ```go
  req.Reasoning = &llmprovider.Reasoning{
      Effort: llmprovider.Effort(catalog.ProfileUtility.ReasoningEffort()),
  }
  p = llmprovider.WithRetry(p, llmprovider.RetryPolicy{})
  ```

## Caches

A listing's models, the models.dev document and the models that answered a
probe are kept in memory, shared by every listing and provider of the
process. A `catalog.Cache` keeps them apart: the listings and providers
given `catalog.WithCache(&c)` share `c`, and no other cache. A test, or a
process with more than one configuration, gives each its own. The zero
value is ready, and a `Cache` is safe for concurrent use.

```go
var cache catalog.Cache
reg := providers.DefaultWith(catalog.WithCache(&cache))
p, err := reg.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey(key))
```

`DefaultWith`'s options come before each `New`'s own, so a later option
wins. A provider the caller registers on the `Registry` is not given them.

## Signing in with a ChatGPT plan

`auth.LoginChatGPTPlan` is OpenAI's Sign in with ChatGPT for agents
([0043-MADR](docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md)). The
user signs in in a browser, under your app's own name. At the first
sign-in OpenAI registers a client for your app under the user's account.
Pass the saved session as `Previous` to sign in to it again.

```go
session, err := auth.LoginChatGPTPlan(ctx, auth.ChatGPTPlanOptions{
    AgentName: "my-agent", // your app's own name, the same on every install
    OpenURL:   openBrowser,
})
if err != nil {
    return err
}
p, err := openai.New(llmprovider.WithTokenSource(session))
```

- **Requests.** Every request is streamed, with `store` false and no output
  limit. `PreviousResponseID`, native tool search, and a system message in
  `Input` are not supported: put system prompts in `Instructions`.
  `ListModels` lists the account's models and never sends a billed probe.
- **Usage.** A plan's usage limit is `ErrQuotaExhausted`.
  `openai.ChatGPTUsageURL` is the page your "Manage usage" action opens.
- **Store.** A `TokenStore` keeps one OpenAI login, of either kind: a plan
  sign-in replaces a Codex-id one, and the reverse. A record saved by this
  version and read by one before `v1.15.0` fails to refresh; signing in
  again mends it.
- **What stays yours.** OpenAI's terms ask things of your app that this
  module cannot do for you, and no option claims to:
  - send requests only from the user's own runtime, or one only that user
    controls, and only for that user's activity;
  - get the user's express consent before any background use;
  - use the plan only for the app the user connected, never as
    general-purpose API access for other tools;
  - never pool, share, transfer or resell plan usage or the tokens;
  - charge nothing for this use.

## The configuration wizard

`wizard.ConfigureLLM(ctx, prompter, opts)` asks for a provider, its
endpoint, a credential and a model, through a `Prompter` you implement or
`wizard.NewTextPrompter`. It never writes configuration and never logs a
key.

- A sign-in saves its session to `Options.TokenStore`, once the model
  listing has not refused it, and returns a `CredOAuth` `Result` with no
  token; the caller loads it from the store. A refused sign-in leaves the
  store as it was. With `Options.Existing` naming such a session, the
  wizard checks it and offers to keep it before asking how to sign in; one
  that cannot be used is not offered, and a notice says why. Without a
  `TokenStore`, only the methods
  that save no session are offered: the API key, a vendor CLI's login, and
  a pasted Grok key.
- With `Options.AgentName`, your app's own name, OpenAI also offers "Use
  your ChatGPT plan with this app" (`AuthChatGPTPlan`). Before the first
  plan sign-in the user confirms OpenAI's notice, "You're using your
  ChatGPT plan". A saved plan session is signed in to again, keeping the
  client OpenAI issued your app.
- A vendor CLI's login is found by default under the home directory, which
  the wizard reads only through `Options.LookupEnv` (`HOME`, or
  `USERPROFILE` on Windows). Pass `LookupEnv: os.Getenv` to find it there.
  With no `LookupEnv` and no `Existing.VendorAuthPath`, that sign-in fails
  and says so.
- `TextPrompter` answers the first prompt after its input ends with that
  prompt's default; every later prompt fails, so a script that runs out of
  answers ends instead of looping. In a hidden entry, Ctrl-U clears what was
  typed and Ctrl-C cancels.
- A base URL needs `http://` or `https://` and a host, and no user name,
  query or fragment. Plain `http` to a remote host asks first, or, under
  `Options.NoEndpointPrompt`, is used with a warning. A pasted
  OpenAI credential must be an `sk-` key or an unexpired ChatGPT access
  token; another vendor's key is refused.
- A provider with no default endpoint, a `chatcompat` server's, must be
  given one. Its key is optional: the wizard lists with `Existing`'s key,
  kept, or with none, and asks for the key when the server refuses the
  request without one.
- With `Options.Discover`, it lists the provider's models once, within
  10 seconds (`Options.DiscoverLimit` can shorten that, not extend it).
  Without it, the built-in catalog is offered. A provider that implements
  `catalog.Lister`, `chatcompat`'s among them, gives the menu its models'
  facts.
- Before each model menu it asks for a search, with or without discovery.
  A blank search shows the recommendations. A glob such as `kilo-auto/*` or
  `*llama*` matches whole ids, and other queries match loosely, in the
  listing's order among equal matches. A search that matches nothing offers
  the query itself as the model id. A saved fallback the menu does not
  recommend is offered after the recommendations, marked "saved" and
  preselected. Scripts that
  drive the wizard answer one extra, blank, line before each model and
  fallback selection.
- The menu is `Options.Registry`, every built-in provider when nil, so a
  provider of your own is offered once it is registered
  ([adding-a-provider.md](docs/guides/adding-a-provider.md)).
- With `Options.LocalServers`, the menu also offers each Chat Completions
  server running on this machine, LM Studio's or llama.cpp's for example,
  as `local-<port>` with its endpoint filled in. A `Result` whose
  `Provider` your `Registry` lacks is one of them: build it with
  `chatcompat.New(chatcompat.WithID(res.Provider),
  llmprovider.WithBaseURL(res.BaseURL))`, and `llmprovider.WithAPIKey`
  when `res.Kind` is `CredAPIKey`.

## Status

The current release is `v1.24.1`. The module requires Go 1.27.3.

What each release changed, from `v1.0.0` on, is in
[CHANGELOG.md](CHANGELOG.md); the five newest tags' GitHub Releases carry
their sections. An older tag's Release page still opens, showing the tag and
its commit without notes; its notes are its section in the changelog.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| see what changed in a release | [CHANGELOG.md](CHANGELOG.md) |
| know why the code is moving here from `mcplib`, and how | [0002-MADR](docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md) |
| know why the v1 API is shaped as it is | [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md) |
| change or add an exported API | [api-standards.md](docs/guides/api-standards.md) |
| add a provider, in this module or my own | [adding-a-provider.md](docs/guides/adding-a-provider.md) |
| move code from `mcplib`'s `llmprovider` or `wizard` | [migrating-from-mcplib.md](docs/guides/migrating-from-mcplib.md) |
| know how provider auth is built, and what it takes from `magic-cli-remote` | [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md) |
| sign a user in with their ChatGPT plan under my own app's name | [0043-MADR](docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
