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

It requires Go 1.27.2.

## Packages

| Package | For |
| :--- | :--- |
| `llmprovider` | the contract: `Provider`, `Request`, `Response`, items, options, errors, `Registry`, `WithRetry` and `WithRetryNotify`, `Stream`, `TokenSource` |
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
  limited, retrying in 4 s"; `notify` must not block.
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
  model's facts decide: its thinking shape and budget bounds, Gemini's
  level, Grok's efforts, and an output limit that cuts the default. The
  facts are the caller's `catalog.WithModelLayer`, then the provider's
  listing, then models.dev, and a request reads only what is already
  cached. Without them the name rules decide, as before.
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

- A sign-in saves its session to `Options.TokenStore` and returns a
  `CredOAuth` `Result` with no token; the caller loads it from the store.
  With `Options.Existing` naming such a session, the wizard offers to keep
  it before asking how to sign in. Without a `TokenStore`, only the methods
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
  query or fragment. Plain `http` to a remote host asks first. A pasted
  OpenAI credential must be an `sk-` key or an unexpired ChatGPT access
  token; another vendor's key is refused.
- With `Options.Discover`, it lists the provider's models once, within
  10 seconds (`Options.DiscoverLimit` can shorten that, not extend it).
  Without it, the built-in catalog is offered.
- Before each model menu it asks for a search, with or without discovery.
  A blank search shows the recommendations. A glob such as `kilo-auto/*` or
  `*llama*` matches whole ids, and other queries match loosely, in the
  listing's order among equal matches. A search that matches nothing offers
  the query itself as the model id. Scripts that
  drive the wizard answer one extra, blank, line before each model and
  fallback selection.
- The menu is `Options.Registry`, every built-in provider when nil, so a
  provider of your own is offered once it is registered
  ([adding-a-provider.md](docs/guides/adding-a-provider.md)).

## Status

The current release is `v1.15.0`. The module requires Go 1.27.2.

- `v1.15.0` adds OpenAI's Sign in with ChatGPT for agents, as
  [0043-MADR](docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md)
  decides:
  - `auth.LoginChatGPTPlan` signs a user in with their ChatGPT plan, under
    your app's own name;
  - the `openai` provider sends a plan session's requests to
    `api.openai.com/v1` under OpenAI's rules for plan usage;
  - the wizard offers the method when `Options.AgentName` is set.
- It also adds `llmprovider.WithRetryNotify`, which reports each retry
  before its wait, as
  [0044-MADR](docs/decisions/0044-MADR-observe-retries.md) decides.

- `v1.14.0` decides each model's behaviour from its facts, and lists a live
  menu ordered by them, as
  [0041-MADR](docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md)
  decides:
  - a request's thinking, effort and output limit follow the model's
    cached facts, read with `catalog.CachedFacts`;
  - `Catalog.Menu` lists every usable model with its facts, rank and
    caveats, and every provider's band is ranked from the facts;
  - `catalog.ProfileAgent` orders the strongest models by the price of a
    cached turn;
  - a transient listing failure serves the last good listing, and
    `Catalog.Source` says so;
  - `catalog.ListRaw` returns each provider's listing as it was sent.

  A Claude provider's `ListModels` now lists live. Since v1.0 it had asked
  for `/v1/v1/models` and returned the static list. A `catalog.List` with
  no base URL now warms the facts a provider's requests read.
  Ollama leaves out models that cannot write text, and keeps recommending
  its chat models without tools, as
  [0042-MADR](docs/decisions/0042-MADR-ollama-models-that-write-text.md)
  decides. The recommended models change, and the exported API only
  grows. The release notes are in
  [0041-PLAN-decide-per-model-behaviour-from-metadata.md](docs/decisions/0041-PLAN-decide-per-model-behaviour-from-metadata.md),
  Phase 9.
- `v1.13.0` adds deferred tools: `Tool.Deferred`, `Tool.Search` and
  `ToolLoadItem`. A deferred tool stays out of the model's context until the
  caller's search loads it. Claude 4.5 and later, and OpenAI from GPT-5.4
  (with every ChatGPT session model), get their service's native form. Grok
  runs them through one fixed `use_tool` wrapper, and every other wire
  appends each tool as it is loaded. `WarningDeferralEmulated` says when the
  native form was not used. A request with no deferred tool is sent as
  before, as
  [0040-MADR](docs/decisions/0040-MADR-native-tool-deferral.md) decides.
  The release notes are in
  [0040-PLAN-native-tool-deferral.md](docs/decisions/0040-PLAN-native-tool-deferral.md),
  Phase 7.
- `v1.12.0` lets a request ask for prompt caching: `Request.Cache`,
  `CacheShort` or `CacheLong`. Claude, directly or through OpenCode, gets
  Anthropic's `cache_control` markers, with a five-minute or one-hour
  lifetime. OpenAI, xAI and Together get a `prompt_cache_key`, and Kilo's
  Claude models the top-level marker. A request that does not set it is
  sent as before, as
  [0039-MADR](docs/decisions/0039-MADR-request-side-prompt-caching.md)
  decides. The release notes are in
  [0039-PLAN-request-side-prompt-caching.md](docs/decisions/0039-PLAN-request-side-prompt-caching.md),
  Phase 4.
- `v1.11.0` lists the full curated catalog by default: `catalog.List`,
  every built-in provider's `ListModels` and the wizard's menu recommend
  every known model that works, ranked, with the best six first, where
  they recommended six. `catalog.WithLimit(catalog.MaxListed)`, or
  `wizard.Options.ModelLimit`, keeps today's six. Probes still test only
  the first six, as
  [0038-MADR](docs/decisions/0038-MADR-full-curated-catalog-by-default.md)
  decides. The release notes are in
  [0038-PLAN-full-curated-catalog-by-default.md](docs/decisions/0038-PLAN-full-curated-catalog-by-default.md),
  Phase 4.
- `v1.10.2` refreshes the curated model lists of OpenCode Zen, OpenCode Go,
  Kilo and Together from a recorded snapshot, as
  [0033-MADR](docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md)'s
  amendment "the second refresh, and what the command reads" decides. A test
  holds each list to the snapshot. The ranker gains three rules
  ([0009-MADR](docs/decisions/0009-MADR-use-case-aware-default-model-ranking.md)
  §3 items 10 to 12): Together recommends only models it serves, and the
  utility profile recommends no dense model of 14B or less. The exported API
  does not change. The release notes are in
  [0033-PLAN-refresh-the-curated-catalogs.md](docs/decisions/0033-PLAN-refresh-the-curated-catalogs.md),
  Phase 9.
- `v1.10.1` reads two `Usage` counts it missed, as
  [0037-MADR](docs/decisions/0037-MADR-reasoning-and-cache-read-spellings.md)
  decides: Anthropic's reasoning count,
  `output_tokens_details.thinking_tokens`, is now `ReasoningTokens` on every
  Messages route, in a full response and a stream; and Chat Completions
  reads a top-level `cache_read_input_tokens` as a cache read when no other
  spelling is present. The exported API does not change. The release notes
  are in
  [0037-PLAN-reasoning-and-cache-read-spellings.md](docs/decisions/0037-PLAN-reasoning-and-cache-read-spellings.md),
  Phase 2.
- `v1.10.0` reports cache writes in `Usage`, as
  [0036-MADR](docs/decisions/0036-MADR-cache-write-token-count.md)
  decides:
  - `CacheWriteTokens` is the part of `InputTokens` written to the
    service's cache, beside `CachedTokens`, the part read from it. The two
    do not overlap.
  - `CacheWrite1hTokens` is the part of the writes with a one-hour
    lifetime, which Anthropic prices at twice the input rate.
  - They are decoded from Anthropic's Messages, OpenAI's Responses, and
    Chat Completions, including Kilo's own spelling, and stay zero on
    Gemini's wires, which report no write.

  With `catalog.ModelRates`, a caller can now price a response's cache
  writes. The exported API only grows. The release notes are in
  [0036-PLAN-cache-write-token-count.md](docs/decisions/0036-PLAN-cache-write-token-count.md),
  Phase 4.
- `v1.9.0` publishes model metadata, as
  [0033-MADR](docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md)
  decides:
  - `catalog.LookupModel` returns a `ModelInfo`, merged field by field from
    models.dev, the provider's own listing, and layers the caller adds with
    `WithModelLayer`. It holds a model's context window, input and output
    limits, modalities, tools, reasoning controls, prices in USD per million
    tokens with cache and reasoning rates and tiers, release date, status
    and retirement date.
  - For each field, `Has` says whether it is known, and `SourceOf` which
    layer set it, so a known zero, such as a free price, is told apart from
    an unknown value.
  - `CachedModel` answers without waiting.
  - `ParseMetadata` and `Metadata.Encode` read and write a models.dev-format
    document, losslessly, for a caller that starts offline.
  - A maintainer command proposes the next curated model lists.

  The exported API only grows. The release notes are in
  [0033-PLAN-public-model-metadata.md](docs/decisions/0033-PLAN-public-model-metadata.md),
  Phase 5.
- `v1.8.0` forces a tool on the Claude models that refuse a forced tool
  choice (`claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1`,
  `claude-mythos-5-1`), as
  [0034-MADR](docs/decisions/0034-MADR-degrade-forced-tool-choice-per-model.md)
  decides:
  - a forced named tool, and so `GenerateToolCall`, is sent as structured
    outputs and comes back as its call;
  - `ToolChoiceRequired` fails before the network with `ErrUnsupported`,
    unless `WithForcedChoiceFallback(ForcedChoiceDegrade)` opts in to
    sending it as `auto` with an instruction;
  - `WithModelForcedChoice` overrides the table per model, and a refusal
    from a model no layer knows is learned for the process;
  - `Response.Warnings` reports a degraded or a learned forced choice;
  - `Tool.Strict` is sent on Messages;
  - Claude's menu adds `claude-sonnet-5-5` and `claude-opus-5-5`, and drops
    `claude-sonnet-4-6`.

  The release notes are in
  [0034-PLAN](docs/decisions/0034-PLAN-degrade-forced-tool-choice-per-model.md),
  Phase 7.
- `v1.7.1` brings the curated model catalogs up to date with the
  providers' official pages, as
  [0033-MADR](docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md)
  D4 decides.
  - OpenAI, Gemini, Claude and Grok's menus drop retired, scheduled,
    rerouted and restricted models, `gpt-4.1-nano` and `o4-mini` among
    them, which OpenAI shuts down on 2026-10-23.
  - Each now lists only current models measured to answer.
  - Labels are per provider, and the Grok effort menus cover `grok-4.7`
    and `grok-4.3`.

  The exported API does not change. The release notes are in
  [0033-PLAN-refresh-the-curated-catalogs.md](docs/decisions/0033-PLAN-refresh-the-curated-catalogs.md),
  Phase 4.
- `v1.7` adds two tool flags, as
  [0032-MADR](docs/decisions/0032-MADR-tool-strict-and-result-error-flags.md)
  decides:
  - `Tool.Strict` (`*bool`) opts a tool in or out of strict schema mode,
    on Responses and on the Chat Completions gateways that accept it;
  - `FunctionCallOutputItem.IsError` marks a failed call's result, on
    Messages, Interactions and generateContent.

  A caller that sets neither sends what it sent before. `llmtest.Harness`
  gains `SendsStrict` and `SendsIsError`. The release notes are in
  [0032-PLAN](docs/decisions/0032-PLAN-tool-strict-and-result-error-flags.md),
  Phase 5.
- `v1.6` streams natively on every wire: Messages (`claude`), Chat
  Completions (`huggingface`, `kilo`, `together`, `ollama`), Gemini
  Interactions (`gemini`), and every OpenCode route, by
  [0031-MADR](docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md)'s
  amendment of 2026-10-10. The API does not change. `NativeStreaming` is
  `Supported` for every built-in provider but the OpenCode gateways, which
  stay `BestEffort`. `Generate` and `WithRetry` behave as before. The
  release notes are in
  [0031-PLAN-native-streaming-on-the-other-wires.md](docs/decisions/0031-PLAN-native-streaming-on-the-other-wires.md),
  Phase 7.
- `v1.5` streams natively on the OpenAI Responses wire, as
  [0031-MADR](docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md)
  decides: `Stream` on `openai`, `grok` and OpenCode's `responses` route
  yields text as it is generated. It adds only `llmtest.Harness`'s
  `StreamText` and `StreamFailure` to the API. `NativeStreaming` changes
  for those four ids; `Generate` and `WithRetry` behave as before. The
  release notes are in
  [0031-PLAN](docs/decisions/0031-PLAN-native-streaming-on-the-responses-wire.md),
  Phase 7.
- `v1.4` makes
  [0028-MADR](docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md)'s
  heuristic and performance decisions, and adds only
  `catalog.CachedMetadataWith` to the API. Some behaviour changes for
  callers: a 403 is `ErrNotPermitted` and is not renewed, Grok's refused key
  is `ErrAuthFailure`, a generation that fails after its request was
  written is sent at most twice, a 429 waits for the service's reset
  headers, and `ConfigureLLM` asks again for a refused typed key and fails
  for a refused key from anywhere else. Default clients share connections,
  and listing probes are cached for ten minutes. The release notes are in
  [0028-PLAN](docs/decisions/0028-PLAN-heuristics-and-performance-from-the-research-pass.md),
  Phase 10.
- `v1.3` remediates
  [0026-MADR](docs/decisions/0026-MADR-remediate-v1-2-debugging-pass-findings.md)'s
  67 findings, and only adds to the API. Some behaviour changes for callers:
  `New` refuses inputs no request could carry, a retryable error no longer
  matches `ErrInvalidRequest`, an answer cut while reasoning is
  `ErrIncomplete`, `FileTokenStore` locks with the OS (upgrade every process
  sharing a store), and `llmtest.Run` checks more. `v1.3.2` reads Gemini's
  errors in the array its Interactions API sends them in, so a refused key
  renews a `CommandToken`, and a 429's delay and per-day quota apply; on
  `v1.3.0` and `v1.3.1` they did not. The release notes are in
  [0026-PLAN](docs/decisions/0026-PLAN-remediate-v1-2-debugging-pass-findings.md),
  Phase 7.

- The provider and wizard code was imported, with its history, from
  `mcplib` `v1.6.0` and re-homed here
  ([0002-PLAN](docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md),
  Phase 4).
- Its API is the v1 API that
  [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
  decides, on the provider-auth baseline of
  [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md).
  [0015-PLAN](docs/decisions/0015-PLAN-canonical-sdk-api-and-module-layout.md)
  has built it, and CI enforces its standards. `v1.0.0` was released after
  the live identity gates of
  [0002-PLAN](docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md)
  Phase 8. From it on, `make api-check` fails on an incompatible change.

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
| sign a user in with their ChatGPT plan under my own app's name | [0043-MADR](docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
