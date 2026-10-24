# Changelog

All notable changes to this module are recorded here, in the form of
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). The module
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html): from
`v1.0.0`, a release that is not major only adds to the exported API, which
`make api-check` enforces (the standards guide's R48). `make changelog-check`
holds this file to the release tags, and the five newest tags' GitHub Releases
carry their sections. An older tag's Release page still opens, showing the
tag and its commit without notes; its notes are its section here.

## [Unreleased]

### Fixed

- When Anthropic refuses a typed key, the wizard no longer says that an
  `sk-ant-usr-` key, or an `sk-ant-api03-` key of another length, has the
  wrong shape. Such keys work; the hint now names only the prefix and a
  tag, "sk-ant- and a tag, such as sk-ant-api03- or sk-ant-usr-", and is
  shown only for a refused key without them
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W14).

## [v1.22.1] - 2026-10-24

### Fixed

- The wizard no longer says that a key the provider accepts has the wrong
  shape. A typed key was checked against one exact shape per provider,
  such as `sk-ant-api03-` and 108 characters for Claude, before the
  listing; a key of another tag or length that the provider accepted was
  warned about, and the warning stayed. Now a key is named before the
  listing only when it is plainly another provider's, and the expected
  shape is named only once the provider has refused a typed key. Without
  `Options.Discover`, no notice promises a check
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W1, W3).
- A sign-in the model listing refuses no longer replaces the saved session.
  The wizard saved a browser, device, ChatGPT plan, pasted-token or Kilo
  sign-in to `Options.TokenStore` before the listing checked it, so a
  refused one took the place of the session that worked. Now the session
  is saved once the listing has not refused it, after any refresh the
  listing left running, so the store gets the rotated tokens; a refresh the
  issuer refuses saves nothing and ends the run
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W2).
- A key pasted into the masked entry no longer gains part of the
  terminal's paste marker. When a marker's escape arrived in one read and
  the rest in the next, the rest, such as `[201~`, was taken as typed
  text. The escape sequence is now followed across reads
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W4).
- The paste prompt of a browser sign-in no longer spins on a `Prompter`
  that answers at once with nothing. After three such answers it says
  so, stops asking, and waits for the browser or the sign-in's limit; an
  empty answer from a person, who takes longer, is still asked again
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W5).
- A saved session that cannot be used is no longer offered, then refused.
  The wizard asked whether to keep it before checking it, and a yes ended
  the run with "sign in again". Now it is checked first; one that cannot
  be used is not offered, a notice says why, and the sign-in menu follows
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W6).
- A saved fallback model the menu does not recommend is no longer
  dropped. The blank fallback round now offers it after the recommended
  models, marked "saved" and preselected, so a blank answer keeps it, and
  the round is shown while such a fallback remains
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W7).
- `ConfigureLLM` on an ended context returns its error before asking
  anything, as `ResolveCredential` does; it asked for the provider and the
  key first
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W8).
- Under `Options.NoEndpointPrompt`, a plain-http remote endpoint the
  caller holds is used with a warning notice, not a question the caller
  cannot answer; a typed one is still asked about
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W9).
- The session of a pasted ChatGPT access token keeps
  `Options.HTTPClient`, as `HTTPClient`'s doc says a sign-in's session
  does
  ([0059-MADR](docs/decisions/0059-MADR-wizard-debugging-pass-findings.md)
  W11).

## [v1.22.0] - 2026-10-22

### Added

- `wizard.Options.ShowDeviceCode` and `wizard.DeviceCode`: a device
  sign-in's provider, URL, code and expiry, given to the caller in place of
  the notice, for one that reports it as data, as
  [0057-MADR](docs/decisions/0057-MADR-wizard-device-code-and-paste-prompt.md)
  decides (D1). An error from the hook ends the sign-in before any poll.
  Without it, the notice is as before.
- `wizard.Options.NoPastePrompt`: a browser sign-in starts no paste prompt,
  for a caller with no terminal. The URL is still shown and opened, and the
  loopback alone completes the sign-in, the ChatGPT plan's included (D2).
- `wizard.InputerContext`: a `Prompter` that can give up a read implements
  `InputContext(ctx, prompt, def)`, and a browser sign-in that the loopback
  finishes then returns at once, with no Enter to press (D3). The wizard uses
  it for the paste prompt only. `TextPrompter` does not implement it, and
  keeps the Enter, since a read of stdin cannot be given up.

### Changed

- An empty answer to a browser sign-in's paste prompt is asked again,
  instead of ending the sign-in with "oauth: empty authorization code"
  (D4). At the end of the input, as from a closed pipe, the next read's
  error, which matches `io.EOF`, ends it.

## [v1.21.0] - 2026-10-21

### Added

- `wizard.ResolveCredential(ctx, p, o, id)`: the wizard's credential step
  alone, for a consumer with its own model picker, as
  [0056-MADR](docs/decisions/0056-MADR-wizard-credential-only-entry-point.md)
  decides (D1). With `id` set it asks no provider; it never asks a model.
  It runs the endpoint, every sign-in method (the ChatGPT plan sign-in with
  its notice included) and the listing that checks the credential, asking
  again for a refused typed key, and returns the `Result`, with no `Model`,
  and that listing's catalog. `ConfigureLLM` is unchanged.
- `wizard.Options.KiloDeviceKey`: Kilo's device sign-in returns the device
  token as a `CredAPIKey`, for a consumer that stores keys rather than
  sessions. It saves no session, needs no `TokenStore`, and asks no
  organization (D2). Without it, the sign-in is as before.
- `wizard.Options.SkipMethods`, which leaves sign-in methods out of every
  menu, and `wizard.Options.NoEndpointPrompt`, which asks no endpoint and
  uses `Existing`'s or the default (D4).

### Changed

- The wizard's notices that report a service's error (a refused key, the
  three model-listing warnings, Kilo's profile, a failed revocation) show
  its class and HTTP status, such as "authentication failed, HTTP 401",
  never the service's message, which can echo a key the service masked
  itself (D3). The errors the wizard returns are unchanged.

## [v1.20.1] - 2026-10-21

### Security

- The module requires Go 1.27.3, for GO-2026-6809 in `crypto/x509`
  (`Certificate.Verify`), which every HTTPS request the module makes
  reaches: Go 1.27.3 distrusts a set of compromised intermediate
  certificates that 1.27.2 accepts, as
  [0055-MADR](docs/decisions/0055-MADR-go-1-27-3-and-the-v1-20-0-release.md)
  records. A consumer at `go 1.27.2` raises its own `go` line to take this
  release; with `GOTOOLCHAIN=auto`, the default, `go` fetches the
  toolchain. `v1.20.0` and earlier build safely only with a Go 1.27.3
  toolchain. No exported name changes.

### Fixed

- The Docs workflow installs golangci-lint at CI's pin, which the gates'
  self-test needs for its precheck test; without it every Docs run failed
  that test, as
  [0052-MADR](docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md)'s
  amendment records (A1). A test now fails when the Docs workflow lacks a
  tool that CI installs before its self-test (A2). The module's code is
  unchanged.

## [v1.20.0] - 2026-10-20

### Changed

- A failure the service reports inside a 200 event stream, such as
  Responses' `response.failed` or Messages' `error` event, is retried at
  most once per call by `WithRetry`, and by `WithStreamRetry` before the
  first event, as a failure while reading a reply already was, as
  [0053-MADR](docs/decisions/0053-MADR-stream-retry-and-cache-defects.md)
  decides (D3). It was retried up to `MaxAttempts` times, each attempt
  perhaps billed.
- A stream whose tool-call, content-block, step or reasoning-entry index is
  negative or 4096 or more ends with an error matching `ErrIncomplete`
  (D2).
- A push to `main` or a pull request that changes only Markdown runs a
  one-job Docs workflow, not the full CI: the gates that read Markdown
  (`parity-check`, `records-check`, `tables-check`, `changelog-check`) and
  `gate-selftest`, as
  [0052-MADR](docs/decisions/0052-MADR-skip-full-ci-for-docs-only-pushes.md)
  decides. Every other change, every tag (and so every GitHub Release) and
  the weekly run keep the full CI. The module's code is unchanged.

### Fixed

- Chat Completions' `reasoning_details`, generateContent's text parts and
  Gemini Interactions' thought signatures accumulate in builders: a stream
  of N fragments took time and memory growing with N², 540 MiB for 16,000
  four-byte fragments (D1).
- A negative index in a Chat Completions, Messages or Gemini Interactions
  stream panicked the caller, and a large one allocated a slot for every
  index below it; both now end the stream with `ErrIncomplete` (D2).
- Concurrent `catalog.List` calls for one listing share one fetch and its
  result, a failure included, and a caller that is cancelled no longer
  restarts it; five concurrent listings answered 503 or 401 sent 5
  requests, one after another (D4).
- An Ollama `/api/show` reply is asked again ten minutes after it was
  fetched, and at once when its model's digest changes; a listing every
  few minutes kept a reply forever (D5).
- Concurrent listings share one run of the model probes `ListModels`
  sends; five cold listings sent 30 billed probes where 6 do (D6).

## [v1.19.0] - 2026-10-20

### Added

- `llmprovider.WithStreamRetry(p, policy, notify)`: live text and a retry
  together, as
  [0051-MADR](docs/decisions/0051-MADR-opt-in-streaming-retry.md) decides.
  Over a provider that streams natively, a failure before the first event
  is retried as `WithRetryNotify` retries it, and after the first event
  every event passes through, a failure ending the stream unretried.
  `WithRetry` and `WithRetryNotify` still buffer, unchanged (D1 to D4).

### Changed

- `llmtest.Run` checks a provider that streams natively through
  `WithStreamRetry`, when its `Harness` sets `StreamText` and
  `StreamFailure`: a 503 before the first event retried, a failure after
  it not, the stream live and carrying the provider's own events (D5). A
  third-party provider whose stream does not yield a failure before its
  first event as a classified first pair fails it. `StreamText` still
  needs one text delta; two are preferred, and a check that cannot be
  made is logged, not failed.

## [v1.18.0] - 2026-10-20

### Changed

- A missing token source given to `catalog.List` or `catalog.ListRaw`
  matches `ErrInvalidRequest`, with its message, as
  [0050-MADR](docs/decisions/0050-MADR-refuse-typed-nil-token-sources.md)
  decides (Q2); it was a plain error.

### Fixed

- A typed-nil token source, a nil pointer such as a nil `*CommandToken`
  held in a non-nil `TokenSource`, of any type, is refused with
  `ErrInvalidRequest` by every `New` (D3) and by `catalog.List`, `ListRaw`
  and `LookupModel`; `catalog.CachedModel` treats it as no source (D4).
  Each panicked, at once or on the first request, before.

## [v1.17.0] - 2026-10-19

### Changed

- Like failures now match like kinds, as
  [0049-MADR](docs/decisions/0049-MADR-v1-16-0-follow-ups.md) D2 decides.
  Each error gains its sentinel's text before its message:
  - `auth.LoginBrowserOAuth`: the identity provider's `access_denied`
    matches `ErrAuthFailure`, as a device-code login's denial does;
  - a device-code login whose code passes its own expiry (`DeviceLogin.Wait`,
    `auth.LoginDeviceOAuth`) matches `ErrAuthFailure`, as the service's
    `expired_token` does;
  - `catalog.ValidateOllamaURL` and `ValidateOllamaURLWith`: an unreachable
    server matches `ErrProviderUnavailable`, and a status other than 200 is
    the Ollama listing's `*APIError`, with its message;
  - `auth.StartDeviceOAuth`, `LoginDeviceOAuth` and `LoginBrowserOAuth`: a
    provider they do not serve matches `ErrUnsupported`.
- `catalog.Cache` also holds the models a provider learned refuse a forced
  tool choice, so a provider built with `catalog.WithCache` learns into its
  own cache; without it, the process's, as before (D3).

### Fixed

- A nil `*StaticToken` given to `WithTokenSource` is refused by every `New`
  with `ErrInvalidRequest`, and `Wait` on a `DeviceLogin` that
  `auth.StartDeviceOAuth` did not return gives `ErrInvalidRequest`; each
  panicked before (D1).

### Repository

- Tests of each provider's probe cache, and of every sign-in method's error
  through `wizard.ConfigureLLM`; the forced-choice tests pass at any `-count`
  (D3, D4).

## [v1.16.0] - 2026-10-19

### Added

- `catalog.Cache` and `catalog.WithCache`: a cache a caller owns, for the
  listings, the model metadata and the probes, shared only by the listings
  and providers given it; and `providers.DefaultWith`, a `Registry` whose
  built-in providers are all given the same options, such as one cache, as
  [0048-MADR](docs/decisions/0048-MADR-adopt-go-sdk-best-practices.md) D1
  decides. Without them every listing and provider of the process shares
  one cache, as before.
- A runnable example in every public package, and a code example in the
  package comments of `llmprovider`, `providers` and `wizard` (D2).

### Changed

- Every exported type's doc says whether it is safe for concurrent use, and
  every function that returns an error names the kinds it returns (D3, D4).
  `wizard.TextPrompter` is documented not safe for concurrent use: two reads
  at once race on its reader.

### Repository

- `CHANGELOG.md`, from `v1.0.0`, and a GitHub Release for each tag, made by
  CI's `release` job from the tag's section (D5). `make changelog-check`
  holds the changelog to the tags, and `make docs-check` the examples and
  the concurrency and error docs (D8).
- The standards guide's R49 to R55 (D8).

## [v1.15.0] - 2026-10-18

### Added

- OpenAI's Sign in with ChatGPT for agents, as
  [0043-MADR](docs/decisions/0043-MADR-openai-chatgpt-plan-sign-in.md)
  decides:
  - `auth.LoginChatGPTPlan` signs a user in with their ChatGPT plan, under
    your app's own name;
  - the `openai` provider sends a plan session's requests to
    `api.openai.com/v1` under OpenAI's rules for plan usage;
  - the wizard offers the method when `Options.AgentName` is set.
- `llmprovider.WithRetryNotify`, which reports each retry before its wait, as
  [0044-MADR](docs/decisions/0044-MADR-observe-retries.md) decides.

## [v1.14.0] - 2026-10-16

### Added

- Each model's behaviour is decided from its facts, and the live menu is
  ordered by them, as
  [0041-MADR](docs/decisions/0041-MADR-decide-per-model-behaviour-from-metadata.md)
  decides:
  - a request's thinking, effort and output limit follow the model's cached
    facts, read with `catalog.CachedFacts`;
  - `Catalog.Menu` lists every usable model with its facts, rank and
    caveats, and every provider's band is ranked from the facts;
  - `catalog.ProfileAgent` orders the strongest models by the price of a
    cached turn;
  - a transient listing failure serves the last good listing, and
    `Catalog.Source` says so;
  - `catalog.ListRaw` returns each provider's listing as it was sent.

### Changed

- A `catalog.List` with no base URL now warms the facts a provider's requests
  read. The recommended models change.
- Ollama leaves out models that cannot write text, and keeps recommending its
  chat models without tools, as
  [0042-MADR](docs/decisions/0042-MADR-ollama-models-that-write-text.md)
  decides.

### Fixed

- A Claude provider's `ListModels` lists live. Since `v1.0.0` it had asked
  for `/v1/v1/models` and returned the static list.

The release notes are in
[0041-PLAN-decide-per-model-behaviour-from-metadata.md](docs/decisions/0041-PLAN-decide-per-model-behaviour-from-metadata.md),
Phase 9.

## [v1.13.0] - 2026-10-13

### Added

- Deferred tools: `Tool.Deferred`, `Tool.Search` and `ToolLoadItem`, as
  [0040-MADR](docs/decisions/0040-MADR-native-tool-deferral.md) decides. A
  deferred tool stays out of the model's context until the caller's search
  loads it. Claude 4.5 and later, and OpenAI from GPT-5.4 (with every ChatGPT
  session model), get their service's native form. Grok runs them through one
  fixed `use_tool` wrapper, and every other wire appends each tool as it is
  loaded. `WarningDeferralEmulated` says when the native form was not used. A
  request with no deferred tool is sent as before.

The release notes are in
[0040-PLAN-native-tool-deferral.md](docs/decisions/0040-PLAN-native-tool-deferral.md),
Phase 7.

## [v1.12.0] - 2026-10-13

### Added

- Prompt caching on request: `Request.Cache`, `CacheShort` or `CacheLong`, as
  [0039-MADR](docs/decisions/0039-MADR-request-side-prompt-caching.md)
  decides. Claude, directly or through OpenCode, gets Anthropic's
  `cache_control` markers, with a five-minute or one-hour lifetime. OpenAI,
  xAI and Together get a `prompt_cache_key`, and Kilo's Claude models the
  top-level marker. A request that does not set it is sent as before.

The release notes are in
[0039-PLAN-request-side-prompt-caching.md](docs/decisions/0039-PLAN-request-side-prompt-caching.md),
Phase 4.

## [v1.11.0] - 2026-10-13

### Changed

- `catalog.List`, every built-in provider's `ListModels` and the wizard's
  menu recommend every known model that works, ranked, with the best six
  first, where they recommended six, as
  [0038-MADR](docs/decisions/0038-MADR-full-curated-catalog-by-default.md)
  decides. `catalog.WithLimit(catalog.MaxListed)`, or
  `wizard.Options.ModelLimit`, keeps the six. Probes still test only the
  first six.

The release notes are in
[0038-PLAN-full-curated-catalog-by-default.md](docs/decisions/0038-PLAN-full-curated-catalog-by-default.md),
Phase 4.

## [v1.10.2] - 2026-10-12

### Changed

- The curated model lists of OpenCode Zen, OpenCode Go, Kilo and Together are
  refreshed from a recorded snapshot, as
  [0033-MADR](docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md)'s
  amendment "the second refresh, and what the command reads" decides. A test
  holds each list to the snapshot.
- The ranker gains three rules
  ([0009-MADR](docs/decisions/0009-MADR-use-case-aware-default-model-ranking.md)
  §3 items 10 to 12): Together recommends only models it serves, and the
  utility profile recommends no dense model of 14B or less.

The exported API does not change. The release notes are in
[0033-PLAN-refresh-the-curated-catalogs.md](docs/decisions/0033-PLAN-refresh-the-curated-catalogs.md),
Phase 9.

## [v1.10.1] - 2026-10-12

### Fixed

- Two `Usage` counts that were missed, as
  [0037-MADR](docs/decisions/0037-MADR-reasoning-and-cache-read-spellings.md)
  decides: Anthropic's reasoning count,
  `output_tokens_details.thinking_tokens`, is now `ReasoningTokens` on every
  Messages route, in a full response and a stream; and Chat Completions reads
  a top-level `cache_read_input_tokens` as a cache read when no other
  spelling is present.

The exported API does not change. The release notes are in
[0037-PLAN-reasoning-and-cache-read-spellings.md](docs/decisions/0037-PLAN-reasoning-and-cache-read-spellings.md),
Phase 2.

## [v1.10.0] - 2026-10-12

### Added

- Cache writes in `Usage`, as
  [0036-MADR](docs/decisions/0036-MADR-cache-write-token-count.md) decides:
  - `CacheWriteTokens` is the part of `InputTokens` written to the service's
    cache, beside `CachedTokens`, the part read from it. The two do not
    overlap.
  - `CacheWrite1hTokens` is the part of the writes with a one-hour lifetime,
    which Anthropic prices at twice the input rate.
  - They are decoded from Anthropic's Messages, OpenAI's Responses, and Chat
    Completions, including Kilo's own spelling, and stay zero on Gemini's
    wires, which report no write.

  With `catalog.ModelRates`, a caller can now price a response's cache
  writes.

The release notes are in
[0036-PLAN-cache-write-token-count.md](docs/decisions/0036-PLAN-cache-write-token-count.md),
Phase 4.

## [v1.9.0] - 2026-10-12

### Added

- Model metadata, as
  [0033-MADR](docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md)
  decides:
  - `catalog.LookupModel` returns a `ModelInfo`, merged field by field from
    models.dev, the provider's own listing, and layers the caller adds with
    `WithModelLayer`. It holds a model's context window, input and output
    limits, modalities, tools, reasoning controls, prices in USD per million
    tokens with cache and reasoning rates and tiers, release date, status and
    retirement date.
  - For each field, `Has` says whether it is known, and `SourceOf` which
    layer set it, so a known zero, such as a free price, is told apart from
    an unknown value.
  - `CachedModel` answers without waiting.
  - `ParseMetadata` and `Metadata.Encode` read and write a models.dev-format
    document, losslessly, for a caller that starts offline.
  - A maintainer command proposes the next curated model lists.

The release notes are in
[0033-PLAN-public-model-metadata.md](docs/decisions/0033-PLAN-public-model-metadata.md),
Phase 5.

## [v1.8.0] - 2026-10-12

### Added

- A forced tool on the Claude models that refuse a forced tool choice
  (`claude-sonnet-5-5`, `claude-opus-5-5`, `claude-fable-5-1`,
  `claude-mythos-5-1`), as
  [0034-MADR](docs/decisions/0034-MADR-degrade-forced-tool-choice-per-model.md)
  decides:
  - a forced named tool, and so `GenerateToolCall`, is sent as structured
    outputs and comes back as its call;
  - `ToolChoiceRequired` fails before the network with `ErrUnsupported`,
    unless `WithForcedChoiceFallback(ForcedChoiceDegrade)` opts in to sending
    it as `auto` with an instruction;
  - `WithModelForcedChoice` overrides the table per model, and a refusal from
    a model no layer knows is learned for the process;
  - `Response.Warnings` reports a degraded or a learned forced choice;
  - `Tool.Strict` is sent on Messages.

### Changed

- Claude's menu adds `claude-sonnet-5-5` and `claude-opus-5-5`, and drops
  `claude-sonnet-4-6`.

The release notes are in
[0034-PLAN](docs/decisions/0034-PLAN-degrade-forced-tool-choice-per-model.md),
Phase 7.

## [v1.7.1] - 2026-10-11

### Changed

- The curated model catalogs are brought up to date with the providers'
  official pages, as
  [0033-MADR](docs/decisions/0033-MADR-public-model-metadata-and-current-catalogs.md)
  D4 decides:
  - OpenAI, Gemini, Claude and Grok's menus drop retired, scheduled,
    rerouted and restricted models, `gpt-4.1-nano` and `o4-mini` among them,
    which OpenAI shuts down on 2026-10-23;
  - each now lists only current models measured to answer;
  - labels are per provider, and the Grok effort menus cover `grok-4.7` and
    `grok-4.3`.

The exported API does not change. The release notes are in
[0033-PLAN-refresh-the-curated-catalogs.md](docs/decisions/0033-PLAN-refresh-the-curated-catalogs.md),
Phase 4.

## [v1.7.0] - 2026-10-11

### Added

- Two tool flags, as
  [0032-MADR](docs/decisions/0032-MADR-tool-strict-and-result-error-flags.md)
  decides:
  - `Tool.Strict` (`*bool`) opts a tool in or out of strict schema mode, on
    Responses and on the Chat Completions gateways that accept it;
  - `FunctionCallOutputItem.IsError` marks a failed call's result, on
    Messages, Interactions and generateContent.

  A caller that sets neither sends what it sent before. `llmtest.Harness`
  gains `SendsStrict` and `SendsIsError`.

The release notes are in
[0032-PLAN](docs/decisions/0032-PLAN-tool-strict-and-result-error-flags.md),
Phase 5.

## [v1.6.0] - 2026-10-10

### Changed

- Native streaming on every wire: Messages (`claude`), Chat Completions
  (`huggingface`, `kilo`, `together`, `ollama`), Gemini Interactions
  (`gemini`), and every OpenCode route, by
  [0031-MADR](docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md)'s
  amendment of 2026-10-10. `NativeStreaming` is `Supported` for every
  built-in provider but the OpenCode gateways, which stay `BestEffort`.
  `Generate` and `WithRetry` behave as before.

The API does not change. The release notes are in
[0031-PLAN-native-streaming-on-the-other-wires.md](docs/decisions/0031-PLAN-native-streaming-on-the-other-wires.md),
Phase 7.

## [v1.5.0] - 2026-10-10

### Added

- `llmtest.Harness`'s `StreamText` and `StreamFailure`.

### Changed

- Native streaming on the OpenAI Responses wire, as
  [0031-MADR](docs/decisions/0031-MADR-native-streaming-on-the-responses-wire.md)
  decides: `Stream` on `openai`, `grok` and OpenCode's `responses` route
  yields text as it is generated. `NativeStreaming` changes for those four
  ids; `Generate` and `WithRetry` behave as before.

The release notes are in
[0031-PLAN](docs/decisions/0031-PLAN-native-streaming-on-the-responses-wire.md),
Phase 7.

## [v1.4.0] - 2026-10-09

### Added

- `catalog.CachedMetadataWith`.

### Changed

- [0028-MADR](docs/decisions/0028-MADR-heuristics-and-performance-from-the-research-pass.md)'s
  heuristic and performance decisions. Some behaviour changes for callers:
  - a 403 is `ErrNotPermitted` and is not renewed;
  - Grok's refused key is `ErrAuthFailure`;
  - a generation that fails after its request was written is sent at most
    twice;
  - a 429 waits for the service's reset headers;
  - `ConfigureLLM` asks again for a refused typed key, and fails for a
    refused key from anywhere else;
  - default clients share connections, and listing probes are cached for ten
    minutes.

The release notes are in
[0028-PLAN](docs/decisions/0028-PLAN-heuristics-and-performance-from-the-research-pass.md),
Phase 10.

## [v1.3.2] - 2026-10-07

### Fixed

- Gemini's errors are read in the array its Interactions API sends them in,
  so a refused key renews a `CommandToken`, and a 429's delay and per-day
  quota apply. On `v1.3.0` and `v1.3.1` they did not.

## [v1.3.1] - 2026-10-07

### Changed

- The README's release notes for `v1.3.0`. No code changed.

## [v1.3.0] - 2026-10-07

### Changed

- [0026-MADR](docs/decisions/0026-MADR-remediate-v1-2-debugging-pass-findings.md)'s
  67 findings are remediated, and the API only grows. Some behaviour changes
  for callers:
  - `New` refuses inputs no request could carry;
  - a retryable error no longer matches `ErrInvalidRequest`;
  - an answer cut while reasoning is `ErrIncomplete`;
  - `FileTokenStore` locks with the OS: upgrade every process sharing a
    store;
  - `llmtest.Run` checks more.

The release notes are in
[0026-PLAN](docs/decisions/0026-PLAN-remediate-v1-2-debugging-pass-findings.md),
Phase 7.

## [v1.2.1] - 2026-10-06

### Fixed

- Hugging Face leaves the tools out of a request whose tool choice is
  `ToolChoiceNone`, as
  [0023-MADR](docs/decisions/0023-MADR-huggingface-tool-choice-none.md)
  decides.

## [v1.2.0] - 2026-10-05

### Changed

- No code changed since `v1.1.1`. The tag marks
  [0021-PLAN](docs/decisions/0021-PLAN-harden-and-tune-after-the-v1-1-review.md)
  complete, whose work `v1.1.1` carried as it landed.

## [v1.1.1] - 2026-10-05

### Added

- `ReasoningItem.Format`, `TokenInvalidator`, and new `llmtest.Harness`
  fields, from
  [0021-MADR](docs/decisions/0021-MADR-harden-and-tune-after-the-v1-1-review.md)'s
  hardening and tuning.

### Changed

- Timeouts, retries, `Search` and `Rank` behave as 0021-MADR decides.
- Harden redaction and the wizard's credential handling.
- Harden discovery and model ranking.
- Harden OAuth refresh, device polling and token storage.
- Harden answer decoding and reasoning replay.
- Harden reply handling and retry behaviour.

### Fixed

- Classify OpenCode's 403 responses as refusals.

## [v1.1.0] - 2026-10-04

### Changed

- [0020-MADR](docs/decisions/0020-MADR-remediate-v1-debugging-pass-findings.md)'s
  59 findings are remediated: provider hygiene and request handling, the
  wizard's sign-in and prompts, the catalog's metadata, client and ranking
  edge cases, reply metadata and decode failures, and OAuth sessions and
  their retries.
- Token storage permissions are hardened on Windows.
- Kilo accepts device-login sessions.

The release notes are in
[0020-PLAN](docs/decisions/0020-PLAN-remediate-v1-debugging-pass-findings.md).

## [v1.0.0] - 2026-10-02

### Added

- The v1 API that
  [0015-MADR](docs/decisions/0015-MADR-canonical-sdk-api-and-module-layout.md)
  decides, on the provider-auth baseline of
  [0016-MADR](docs/decisions/0016-MADR-provider-auth-and-support-baseline.md).
  The provider and wizard code was imported, with its history, from
  `mcplib` `v1.6.0` and re-homed here
  ([0002-PLAN](docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md),
  Phase 4), and released after the live identity gates of its Phase 8. From
  it on, `make api-check` fails on an incompatible change.

[Unreleased]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.22.1...HEAD
[v1.22.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.22.0...v1.22.1
[v1.22.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.21.0...v1.22.0
[v1.21.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.20.1...v1.21.0
[v1.20.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.20.0...v1.20.1
[v1.20.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.19.0...v1.20.0
[v1.19.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.18.0...v1.19.0
[v1.18.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.17.0...v1.18.0
[v1.17.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.16.0...v1.17.0
[v1.16.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.15.0...v1.16.0
[v1.15.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.14.0...v1.15.0
[v1.14.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.13.0...v1.14.0
[v1.13.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.12.0...v1.13.0
[v1.12.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.11.0...v1.12.0
[v1.11.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.10.2...v1.11.0
[v1.10.2]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.10.1...v1.10.2
[v1.10.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.10.0...v1.10.1
[v1.10.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.9.0...v1.10.0
[v1.9.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.8.0...v1.9.0
[v1.8.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.7.1...v1.8.0
[v1.7.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.7.0...v1.7.1
[v1.7.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.6.0...v1.7.0
[v1.6.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.5.0...v1.6.0
[v1.5.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.4.0...v1.5.0
[v1.4.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.3.2...v1.4.0
[v1.3.2]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.3.1...v1.3.2
[v1.3.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.3.0...v1.3.1
[v1.3.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.2.1...v1.3.0
[v1.2.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.2.0...v1.2.1
[v1.2.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.1.1...v1.2.0
[v1.1.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.1.0...v1.1.1
[v1.1.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/maccavelli/go-llmprovider-sdk/releases/tag/v1.0.0
