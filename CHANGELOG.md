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

## [v1.25.0] - 2026-10-31

### Added

- `catalog.Lister`, an interface with one method, `ListCatalog`: a
  provider that lists its models as a `catalog.Catalog`, its menu, facts
  and `Source` with them, where `llmprovider.ModelLister` gives ids only.
  `chatcompat`'s provider implements it, with no probe, and the wizard
  lists through it before `ListModels`, so a Chat Completions server's
  menu has its facts
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  Q1, V4).
- `wizard.Options.LocalServers`: the provider menu also offers each Chat
  Completions server `chatcompat.Discover` finds running on this machine,
  as `local-<port>` with its endpoint filled in, and a server that wants a
  key is asked for one. Ollama's port, already offered, is left out. A
  `Result` whose `Provider` the caller's `Registry` lacks is such a server,
  built with `chatcompat.New` as the field's doc shows
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V5, Q2).
- `llmprovider.NewHTTPClient`: the client a provider builds when given
  none, a new one per call over the transport the process shares, with
  its bounds (a connection within 30 s, the first byte within 300 s). A
  caller that adds request headers wraps its `Transport` and gives the
  client to `WithHTTPClient`, or to the wizard as `Options.HTTPClient`,
  keeping those bounds and the shared connections, where wrapping
  `http.DefaultTransport` lost them (issue #10;
  [0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md),
  amendment "issue #10", D2).

### Changed

- The wizard configures a Chat Completions server that wants a key: a
  remote provider whose key is optional is offered the saved key, else
  listed with none, and when the server refuses that, the key is asked for
  as a typed key is, up to three; the `Result` is a `CredAPIKey`. A refused
  request that carried no key, a local Ollama's, is worded so, not as a
  pasted credential. A provider with no default endpoint takes no blank
  one, and under `Options.NoEndpointPrompt` with no `Existing.BaseURL` is
  an error, where it returned a `Result` that could not build its provider
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V2, V3).
- A saved session the user keeps and the model listing then refuses no
  longer ends the run: a notice says so, and the sign-in menu follows,
  once, without it. The store keeps it until a new session is saved; a
  refused new one ends the run, as before. `wizard.ResolveCredential`'s
  catalog carries a failed listing's error in `Err` on every path, a
  provider's own `ListModels` and a Chat Completions server's included,
  and its doc says what `Live` and `Err` tell
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V6, V7).
- Both CI workflows run three more documentation gates: `markdownlint`,
  at one pin (`markdownlint-cli2@0.23.2`), on every Markdown file but the
  records; `links-check`, every relative link; and `disclosure-check`, a
  real macOS, Linux or Windows home path in any file, as
  [0063-MADR](docs/decisions/0063-MADR-align-docs-ci-and-disclosure-gates.md)
  decides. The gates' self-test checks the workflows too: a Markdown-only
  change must run every gate that reads Markdown. That change is to CI
  alone, not to the module's code.

### Fixed

- A listed model id holding a character a terminal acts on, a C0 control
  but tab, DEL or a C1 control, no longer reaches a menu, the terminal or a
  `wizard.Result`: every `catalog` listing drops it, Ollama's and a
  `chatcompat` server's included, as do `chatcompat.Discover`'s
  `Server.Models` and the wizard's listings through a provider's own
  `ListModels`. `wizard.TextPrompter` prints no such character from a
  menu's title, labels or details, a question, or a prompt and its
  default, and a Kilo organization's name and role reach the menu without
  them. A server that listed `\u001b]0;…\u0007\u001b[2J` in an id could set
  the terminal's title and clear its screen
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V1).
- The wizard's notices show no service's text and no key a URL carries.
  The Ollama endpoint check says `<URL> answered: <class>, HTTP <status>`
  for a server that answered, or `cannot reach <URL> (<class>)`, where it
  showed the server's reply. A refused endpoint is shown by its scheme,
  host and path, with `?…` for a query and `#…` for a fragment, and a value
  that is not an http or https URL is not shown; the plain-http warning
  shows it the same way. An offline machine is called "offline", not
  "provider unavailable". A failed listing beside a provider's static
  catalog, where its own listing offered nothing, is told as the others
  are, where it fell back with no notice
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V8, V9, V10, V17).
- An id in `wizard.Options.Providers` that the registry lacks is named in
  a notice and left out of the menu, where it was dropped without a word;
  when the registry lacks every one, the error is as before. The provider
  guide said the first six listed ids are recommended: with
  `Options.ModelLimit` at zero every one is, and a positive limit
  recommends that many. The wizard's tests run behind `internal/localonly`'s
  guard, as `catalog`'s and the providers' do
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V11, V12, V13).
- `llmprovider.WithHTTPClient`'s doc, the package doc and `README.md` say
  how to add request headers, which
  [0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  D6 said they would and none did
  ([0066-MADR](docs/decisions/0066-MADR-wizard-v1-24-debugging-pass-findings.md)
  V16).

## [v1.24.1] - 2026-10-27

### Fixed

- The release of `v1.24.0`: its tag was pushed before its section in
  `CHANGELOG.md` was written, so its CI made no GitHub Release. `v1.24.1`
  has `v1.24.0`'s code unchanged, with that section and the README's
  Status; `v1.24.0` gets its Release by hand, from its section
  ([0064-PLAN](docs/decisions/0064-PLAN-anthropic-workspace-for-unscoped-keys.md) Phase 7).

## [v1.24.0] - 2026-10-27

### Added

- `claude.WithWorkspace` and `catalog.WithClaudeWorkspace`: the Anthropic
  workspace a key not scoped to one runs in, sent as
  `anthropic-workspace-id` on generation, streaming, each probe and each
  listing page. Empty sends none; a value with a control character is
  refused with `ErrInvalidRequest`. The listing and probe caches keep each
  workspace's entries apart ([0064-MADR](docs/decisions/0064-MADR-anthropic-workspace-for-unscoped-keys.md) D1, D2).
- `llmprovider.ErrWorkspaceRequired` and `llmprovider.ErrWorkspaceNotFound`,
  both beneath `ErrInvalidRequest`: Anthropic's refusal of a key not
  scoped to a workspace that names none, and of a workspace that is
  malformed or the key may not use. A caller matching `ErrInvalidRequest`
  still matches ([0064-MADR](docs/decisions/0064-MADR-anthropic-workspace-for-unscoped-keys.md) D3).
- `wizard.Result.Workspace`: the wizard asks a Claude key that needs a
  workspace for one, up to three answers, keeps the one that lists, and
  saves nothing after three refusals. An existing Claude workspace is
  listed with first; `ANTHROPIC_WORKSPACE_ID` is the prompt's default under
  `Options.AllowEnv` ([0064-MADR](docs/decisions/0064-MADR-anthropic-workspace-for-unscoped-keys.md) D4).

## [v1.23.0] - 2026-10-26

### Added

- `llmprovider/providers/chatcompat`: a provider for any server that speaks
  OpenAI's Chat Completions API, LM Studio, llama.cpp's server, vLLM or a
  gateway, under an id the caller picks: `New`, `WithID` and `Descriptor`,
  registered in the caller's `Registry` (`providers.Default()` does not hold
  it). The key is optional; with none, no `Authorization` header is sent.
  It learns a refused effort as the built-ins do. Its `ListModels` is
  `catalog.List`'s listing of the server, cached and shared as a built-in's,
  and `catalog.List`, `LookupModel`, `CachedModel` and `CachedFacts` take its
  id when the options carry its `WithID`
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md) D4,
  amendment "`chatcompat`, provider-agnostic", E1).
- `catalog.Catalog.Search` and `catalog.Match.Why`: a listing's search, by
  each model's facts as well as its id and name: its family, and its
  features as words (`vision`, `audio`, `video`, `pdf`, `tools`,
  `reasoning`, `thinking`, `free`, the context window from `8k` to `2m`,
  each matching a window at least its size, and the release year), each
  only where the facts state it, so that `claude vision 200k` finds the
  Claude models with image input and a window of 200k or more. It searches
  `Usable`, in its order, with `Menu`'s facts by id, on the index `List`
  built once. `Why` says how a match matched (`exact id`, `id prefix`,
  `every word: vision`, `near sonnet`)
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "Model search, rebuilt", S1, S3, S5).
- `chatcompat.Discover` and `chatcompat.Server`: the local Chat Completions
  servers running now, for a configure step to offer. It asks each
  documented default port on `127.0.0.1` only, all at once, each within 300
  ms, with one `GET /v1/models` and no key: Ollama 11434, llama.cpp 9931,
  LocalAI 8080, vLLM 8000, Jan 1337, KoboldCpp 5001, LM Studio 1234. Each
  `Server` has its `BaseURL`, its `Label` (its dialect's, such as "Ollama",
  or its host and port), its `Models`, and `NeedsKey` for one that answered
  401 or 403
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  E5, amendment "E5's ports and a keyed server, confirmed before Phase 4e").
- `llmprovider.SharedOption`: an option every provider's `New` takes, whatever
  its id, for a value only some read
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "Phase 4's options").

- `llmprovider.ErrEmptyAnswer`: an answer with nothing usable in it, on
  every wire. It also matches `ErrIncomplete`, as such an answer did
  before, and `APIError.Reason` still says why (`content_filter`,
  `length`, …). The error's message is unchanged
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  D1; issue #9).
- `llmprovider.ErrOffline`: a request that could not be sent for want of a
  network: a host name that did not resolve, or a dial that timed out or
  found the port refused or the network or host unreachable. It also
  matches `ErrProviderUnavailable` and is retried. A dial the caller's own
  context ended, a TLS failure and any HTTP response are not offline
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  D2; issue #9).
- `llmprovider.EffortMax`, the highest tier a service names, above `xhigh`
  where it has both, and `WarningEffortDegraded`. Each provider sends it as
  its wire's `max` where the model's facts list `max` or list no efforts;
  on a model whose facts list efforts without it, the highest listed below
  it, warned; and on a model that refuses it, the next effort it takes,
  learned (below)
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  D3; [0062-MADR](docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md) D3; issue #9).
- `llmprovider.WarningReasoningLearned`, `ReasoningRefusal`, `ThinkingShape`
  (`ThinkingAdaptive`, `ThinkingBudget`), `Warning.Refused`,
  `WithReasoningRefusals` and `Settings.ReasoningRefusals`: a provider that
  learns a model's refusal of a thinking setting reports it in a
  `WarningReasoningLearned`, whose `Refused` a caller keeps and passes back
  with `WithReasoningRefusals` on its next start, so that the setting is
  never sent again. `Warning.Refused` is left out of a warning's JSON when
  zero, so other warnings encode as before ([0062-MADR](docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md) D6).
- `llmprovider.ProbeReasoning(ctx, p, model)`: one generation at `EffortMax`
  with a one-line prompt, which returns the refusals the provider learned
  from it, of an effort or a thinking shape, for a caller to keep and pass
  back with `WithReasoningRefusals`. The accepted send is billed. On an
  error it returns no refusals. Nothing in this repository calls it
  ([0062-MADR](docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md) D7).
- `llmprovider.ImageItem`, `Capabilities.Images` and `WarningImageDropped`:
  an image in a user turn of `Request.Input`, by its bytes (`Data`, a PNG,
  JPEG, GIF or WebP `MediaType`) or by an https `URL`, sent in each wire's
  form and joined to the user's turn before it. `Check` refuses an invalid
  image with `ErrInvalidRequest`, and any image where `Images` is
  `Unsupported` with `ErrUnsupported`, before the network. Every built-in
  provider and `chatcompat` sends images, a ChatGPT session included, except
  an OpenAI plan session; a model whose facts state its input without image is refused
  before the network; `grok` refuses a GIF or a WebP, which xAI does not
  take; `ollama` and OpenCode's `google` route drop an image by URL, with
  a `WarningImageDropped`. `llmtest.Run` checks the new capability, and
  `llmtest.Fake` records an image's bytes as they were sent
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md) D5).

### Changed

- A `chatcompat` listing knows its models' facts: where the server's
  `/models` reply is Ollama's (every id tagged, every entry with `created`
  and `owned_by`), `List` reads its `/api/tags` and `/api/show` at the
  server's root, cached by digest as the built-in Ollama's are; and below
  that, models.dev's facts of each model's canonical id (an organisation,
  an Ollama tag `latest`, and quantisation and format tokens dropped; an
  equal id only), across the nine sections the SDK keeps, the value most
  of them give. A `chatcompat` listing now fetches the models.dev
  document, as a built-in's does; `llmprovider.WithoutModelMetadata` turns
  it off. Its menu is the default rank below
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  E2, E4; amendment "E2 and E4's menu, made exact before Phase 4c").
- The built-in Ollama's menu, and a `chatcompat` listing's, follow a
  default rank where they followed the listing's order: the uncaveated
  first, then the larger context window, then the newer `created` (Ollama's
  `modified_at`), then the listing's order, each model's variants (one
  model's sizes aside, its quantisations, formats and `instruct`, `chat`,
  `it` and `base` forms) together, the plainest first. `Usable` keeps the
  listing's order. On a `chatcompat` listing only, a model whose output is
  not known and whose id names an embedding, a reranker or speech model is
  caveated as writing no text
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "E2 and E4's menu, made exact before Phase 4c").
- `catalog.WithModelLayer` is a shared option, as `catalog.WithCache` is:
  `chatcompat` reads it, so a caller's layer ranks above a server's facts,
  and a provider outside this module ignores it, where it refused it
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "E2 and E4's menu, made exact before Phase 4c", X8).
- `catalog.Search` keeps its tiers and their scores, and orders each tier by
  BM25F over the id and the label (the id weighed 3, the label 2), then by
  fzf's score, then the listing's order, where the listing's order alone
  decided. Words split also between a letter and a digit (`gpt4o` is
  `gpt`, `4`, `o`; a size such as `8b` stays whole). Below the tiers, a
  query whose every word is an id or label word, or a prefix of one, matches
  at 1800. Only when nothing above matches are the fallbacks tried, in
  turn: every word matched, one at least within an edit or two
  (`sonet` finds `claude-sonnet-5`, at 1500, where the subsequence tier
  found it at 1035); some words, more first; then a subsequence, as before.
  `catalog.Search` over a thousand ids takes about twice the time it did,
  for its index; `Catalog.Search` on `List`'s index, from three to seven
  times less
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "Model search, rebuilt", S2, S5).
- The wizard's model search asks for "a name, a family, or a feature such
  as vision, tools, 128k", searches a listing's facts too, shows each match
  with its facts and why it matched (`200k ctx · $3 / $15 per M · vision ·
  tools · near sonnet`), counts the matches in its title, and shows them
  ten at a time with a "More" row, where it cut them at twenty with a
  notice. The fallbacks' search keeps its cut. `TextPrompter` aligns each
  menu's details in a column
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "Model search, rebuilt", S4).
- `catalog.WithCache` is a shared option: every provider's `New` takes it,
  where one outside the built-ins refused it. A provider outside this module
  ignores it; `chatcompat` uses it for its probes and learned refusals
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "Phase 4's options").
- A request's reasoning effort is decided by the model's facts and by what
  the model refused, not by rules of model names: one the facts do not
  list, or the model refused, steps to the nearest effort it takes, lower
  first, with a `WarningEffortDegraded`. A refusal of the effort sent, in a
  form recorded live per wire, is remembered for the process, in memory
  only, and the request sent again: `wire.Send`'s one resend becomes at
  most 7 sends, a forced-choice refusal counted in them. Removed: Grok's
  effort menu, which clamped an effort and sent none to a model off it;
  Gemini's mapping of `xhigh` to `high` and its `gemini-2.5-flash-lite`
  exception; OpenCode's chat route sending no `reasoning_effort` without
  metadata; 0061's rows sending `max` lower on Grok, Gemini and
  OpenCode's google route; and Ollama's mapping of `xhigh` to `"max"`:
  Ollama 0.33.2 takes `xhigh`, which is now sent as asked. Ollama learns one
  refusal, a model's that does not think (`"<model>" does not support
  thinking`): the request is sent again with no effort, warned, as every
  later one to that model is, and so does `chatcompat` pointed at Ollama;
  its facts and `WithReasoningRefusals` steer it as every other provider's do
  ([0062-MADR](docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md) D2 to D8;
  [0062-REPORT](docs/reports/0062-REPORT-thinking-refusals.md);
  [0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md),
  amendment "Ollama's refusal of thinking, learned").
- A request's thinking shape is the model's facts', else an enabled budget
  when `Reasoning.Budget` is set, else adaptive thinking with the effort; a
  refused shape is remembered as an effort is, and the request sent again
  with the other. Removed: Claude's version rule, which sent models before
  Claude 4.7 a budget; MiniMax-M3's adaptive thinking with no effort; and
  Gemini's version rule on OpenCode's google route. That route now sends an
  effort as `thinkingLevel`, where it sent `low` as a level and every other
  effort, or none, as the dynamic budget `-1`
  ([0013-MADR](docs/decisions/0013-MADR-remediate-debugging-pass-findings.md) B9,
  superseded); `xhigh`, which Zen's `gemini-3.8-flash` refuses, is learned
  and sent as `high`
  ([0062-MADR](docs/decisions/0062-MADR-learn-thinking-parameters-from-refusals.md) D2, D3, D8).
- A request that could not be sent for want of a network now matches
  `ErrProviderUnavailable`, through `ErrOffline`, on every send path:
  generation, model listing, OAuth refresh and sign-in. Such a failure
  matched no kind before, apart from Ollama's listing and the OAuth
  refresh, which matched `ErrProviderUnavailable` already. A caller's
  `errors.Is(err, llmprovider.ErrProviderUnavailable)` branch now sees it;
  `WithRetry` retries it, as it did
  ([0061-MADR](docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md)
  D2).

## [v1.22.2] - 2026-10-24

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

[Unreleased]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.25.0...HEAD
[v1.25.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.24.1...v1.25.0
[v1.24.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.24.0...v1.24.1
[v1.24.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.23.0...v1.24.0
[v1.23.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.22.2...v1.23.0
[v1.22.2]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.22.1...v1.22.2
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
