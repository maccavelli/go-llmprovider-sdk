# Changelog

All notable changes to this module are recorded here, in the form of
[Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/). The module
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html): from
`v1.0.0`, a release that is not major only adds to the exported API, which
`make api-check` enforces (the standards guide's R48). `make changelog-check`
holds this file to the release tags, and the six newest tags' GitHub Releases
carry their sections. An older tag's Release page still opens, showing the
tag and its commit without notes; its notes are its section here.

*(2026-11-24: one recorded exception to that promise. v1.33.0 removes eight
v1 names, as [0093-MADR](docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md)
D10 and its Q1 decided, an exception to the standards guide's R42 and R48;
`make api-check` accepts exactly those eight against `v1.32.0`. They are
listed under **Removed** below.)*

## [Unreleased]

### Added

- `llmtest.Harness.TruncatedCallWhole`: for a provider whose wire delivers
  a call whole, its `Truncated` reply's call with `{}` is a whole call that
  takes no arguments, and W12 wants a response finishing `FinishLength`
  that keeps it, with no `WarningToolCallCut`; unset, W12 wants
  `ErrIncomplete` as before. Gemini's and OpenCode's Google-route
  conformance harnesses set it (0088-MADR ST111).

### Changed

- An answer the output limit cut: on generateContent and an Interaction's
  plain reply, a last tool call that takes no arguments, `{}`, is now
  returned as a whole call, and on generateContent's stream yielded, with
  no `WarningToolCallCut`; so is it on an Interaction's stream when a
  non-empty piece of the call's arguments arrived, even `{}`. A last call
  on an Interaction's stream that got no `arguments_delta`, or only pieces
  that are empty, white space or `null`, and a last `{}` call on Messages,
  Responses and Chat Completions, whose arguments stream in pieces, is
  still dropped and named (0088-MADR ST107, ST110, OQ21).

### Fixed

- A Responses stream (`openai`, `grok`, OpenCode's responses route) that
  completes without an `output_item.done` for an item now takes that item
  from the terminal response's `output`, after the items delivered, and
  yields it before `EventDone`; it no longer fails as an empty answer after
  showing text, or drops a call or a reasoning item while reporting
  `stop`. A `.done` that came empty gives way to the terminal's copy. One
  `Debug` line through `WithLogger` reports the count taken of each type.
  A cut stream no longer duplicates an item whose ids disagree: the two
  sides pair by id, then by `output_index` within one type, then in order
  within one type, and take nothing of a type they cannot pair, nor
  anything past 4096 items
  ([0092-MADR](docs/decisions/0092-MADR-fill-a-completed-responses-stream-from-its-terminal-output.md)
  D1, D2, ST6, ST12, ST20).
- A completed Responses stream that announced a message, a call or a
  client tool search and delivered it neither by `output_item.done` nor in
  its terminal response, where it succeeded with the item dropped, is now
  `ErrProviderUnavailable`, marked after the reply as an early end is, its
  text naming `item_not_finished` with the item's type and id: retried at
  most once, and never after an event (0092-MADR ST11). Past 4096 items
  the check is not run (ST33).

## [v1.33.0] - 2026-11-24

### Added

- `catalog.LookupLimits` and `catalog.CachedLimits`, with `catalog.Limits`,
  `catalog.Fact` and `catalog.Evidence`: a model's output limit, window and
  input limit, each with every source's value, time and staleness, the
  chosen value by rank (the caller's layers, then the service, then the
  registry), a disagreement marked conflicting, an output above the window
  held to it, and "unknown" a state, not a number; and whether the model's
  wire requires an output limit. `LookupLimits` reads, beside what
  `LookupModel` reads, Claude's `GET /v1/models/{id}` for a model or alias
  the listing does not name, the context Ollama's `/api/ps` serves a loaded
  model (and a chatcompat server's of Ollama's dialect), the ChatGPT
  backend's `context_window`, Together's `config.max_output_length`, Hugging
  Face's per-provider `context_length` and Kilo's `per_request_limits`, and
  the models.dev document the call hands in with `catalog.WithMetadata`,
  as the registry (0093-MADR D5, D7). A
  Claude workspace is read from `catalog.WithClaudeWorkspace`. `LookupModel`
  and `ModelInfo` are unchanged (0089-MADR D1 to D3). `Limits.StreamedOutput`
  is the output limit for a streamed request beside `Output`'s for a plain
  one, each held to the cap the service stated for its own kind of call
  (0089-MADR DL76). `Limits.RouteFallback` is the Messages route's
  fallback, the smallest limit a current model on the route is stated to
  take, found with the call's own options and carried in its `Limits`
  (0093-MADR D7).
- `catalog.WithMetadata`: a shared option that hands in a models.dev
  document the caller parsed with `catalog.ParseMetadata`. Every listing,
  lookup and limits call reads the document its options hand in, and a
  provider's requests read only the one given to its `New`
  ([0093-MADR](docs/decisions/0093-MADR-take-model-metadata-from-the-caller.md) D1, D2).
- `NOTICE` and `LICENSE.models.dev`: the models.dev data the module holds
  (OpenCode's route snapshot and the test fixtures), credited under
  models.dev's MIT licence (0093-MADR D11).
- `llmprovider.Response.OutputLimit`, a `llmprovider.SentLimit`: the output
  limit a request carried, after a thinking budget raised it, whether one
  was sent, its `llmprovider.LimitBasis` (`LimitBasisRequest`,
  `LimitBasisOption`, `LimitBasisFact`, `LimitBasisRefusal`,
  `LimitBasisFallback`, `LimitBasisNone`) and its source; a stream's final
  response carries it too (0089-MADR D7).
- `llmprovider.WarningLimitLearned`, with `Warning.Observed`, a
  `llmprovider.LimitObservation`: a cap the service stated when it refused a
  limit the SDK chose, kept for the process and reported once.
  `llmprovider.WithLimitObservations` passes such caps back on the next
  start; one is used only for its base URL and scope, within 24 hours of
  when it was seen, and only where no other source states the model's limit.
  `llmprovider.WithoutLimitLearning` keeps and reports nothing, and still
  resends (0089-MADR D5, DL40 to DL42, DL49, DL73).
- `APIError.Observed`: the output cap a service stated when it refused a
  call's limit, kept by the SDK, on the error that ends the call; `WithRetry`
  carries a retried attempt's to the final result (0089-MADR DL78 to DL94).
- `llmprovider.Settings.MaxTokensGiven`: whether `WithMaxTokens` gave the
  limit, so that a provider outside this module can tell a caller's value
  from the default (0088-MADR OQ5, ST35, ST36).
- `llmprovider.WarningToolCallCut`: the output limit cut a tool call before
  its arguments were whole, so the response dropped it; its `Detail` names
  the call (0088-MADR ST77, OQ16).

### Changed

- A request that names no output limit (no `MaxOutputTokens`, no
  `WithMaxTokens`) no longer sends 8192. On a Messages route, Claude and
  OpenCode's messages route, whose wire requires one, it sends the model's
  output limit as `catalog.CachedLimits` states it, held to the stated
  window and to a cap the service stated; where nothing states it, the
  smallest limit of the route's current models, as the cached listing or
  the handed-in document states them, and where nothing states even that,
  the request is
  refused before sending with an error matching `ErrInvalidRequest` that
  says how to state the limit. Kilo sends the limit its listing states (0089-MADR DL65, measured
  2026-11-24: with none sent, Kilo cut one model at 128,000 of 131,072 stated). Every other wire sends none, so the
  service's own default applies: Gemini, Grok, OpenAI with a key, Together,
  Hugging Face, Ollama, chatcompat and OpenCode's other routes. A request's
  own `MaxOutputTokens` is sent as given; a `WithMaxTokens` value is cut to
  the model's stated limit and never raised; a listing probe still sends
  8192; the ChatGPT routes send none, as before. A caller that wants the old
  value passes `WithMaxTokens(8192)` (0089-MADR D6, DL44 to DL51, DL77;
  0088-MADR ST99).
- A limit the SDK chose (the model's stated limit, or the route's smallest)
  is now held to the room the model's stated window, else its stated input
  limit, leaves the request's input, estimated from its content as pi does:
  3.5 characters a token, each image 4,800 characters, 4,096 spare, never
  below the wire's minimum. The hold comes before a thinking budget raises
  the limit; a caller's own limit is not held (0088-MADR ST87, ST105,
  ST106; 0089-MADR DL56).
- A refusal of a limit the SDK chose that states the service's cap
  (Anthropic's `max_tokens: N > M`, OpenAI's "supports at most", Gemini's
  range, each on the wires 0089-MADR DL68 names, and on Kilo its range form,
  "The max_tokens parameter is illegal.：限制数值范围[1,M]", whose lower bound
  must be 1, 0089-MADR DL95) or a window and the tokens
  asked (OpenRouter's and Kilo's forms on Kilo) is resent once at the stated
  value, outside the bound of seven sends, once the refusal is checked
  against the request; a stated cap is kept and bounds later requests to
  that model for the process. A caller's own limit is never resent
  (0089-MADR D4 to D6, DL31 to DL39; 0088-MADR ST56, ST57).
- A `Generate` that names no output limit is now read over the service's
  stream and returned whole, on every provider, Gemini's listing probe
  included, since it keeps the provider's limit. A failure before the first event is
  retried by `WithRetry` as before; one after it is returned, not resent; the
  assembled answer is bounded at 16 MiB, past which it is `ErrIncomplete`;
  the usage is what the stream reports, zero when it reports none; and an
  error inside a 200 as the stream's first payload is classified as
  `Stream` classifies it, and resent once when it states a limit the SDK
  chose. A cap learned from such a call bounds `Limits.StreamedOutput`, not
  `Limits.Output`. A `Generate` that names its own limit stays a plain
  request (0088-MADR ST88, ST88a to ST88d).
- A `Stream` whose first payload, inside a 200, is a refusal of a limit the
  SDK chose is resent once, before any event is yielded; an error after an
  event is yielded as before (0088-MADR A5, ST55).
- Kilo's documented window refusal ("exceeds the model's context window of
  N tokens") is `ErrContextOverflow`, beneath `ErrInvalidRequest`, where it
  was `ErrInvalidRequest` alone (0088-MADR ST51).
- An answer the output limit cut is a response finishing `length` on every
  wire, by `Generate` and at a stream's `EventDone`, with its text and its
  whole tool calls, where a cut tool call, on Chat Completions, Messages and
  generateContent, and every cut Responses or Gemini Interactions answer
  was `ErrIncomplete`. A call the limit cut is dropped, never yielded, and
  named in a `WarningToolCallCut`; on Messages a `tool_use` that is the
  last block is that call. A cut answer with nothing usable, only reasoning
  or only a cut call, is `ErrEmptyAnswer`, which still matches
  `ErrIncomplete`, with the service's reason (`length`, `max_output_tokens`,
  `incomplete`). A Responses answer incomplete for another reason is
  `ErrIncomplete` as before (0088-MADR ST75 to ST81, ST93, ST94, ST103).
- Model metadata comes only from the caller. A listing's ranking and its
  menu's facts, `Catalog.Search`, `LookupModel`, `CachedModel`,
  `CachedFacts`, `LookupLimits` and `CachedLimits` read the document the
  call hands in with `catalog.WithMetadata`, and none when none is handed
  in: the curated order and the name rules then rank, as with the fetch
  off before. A provider's requests read only the document given to its
  `New`, never one handed to a lookup on the same cache. The document's
  evidence has no `Seen` and is never `Stale`: its age is the caller's
  (0093-MADR D2 to D5, Q4).
- OpenCode's request route is the pinned route, else the handed-in
  document's `provider.npm`, else the route table, else the name rule, at
  once: it never waits for a document (0093-MADR D6).
- A request on a Messages route (Claude, OpenCode's messages route) whose
  model no source states a limit for, with nothing listed, no document, no
  layer and no limit given, is refused before sending with an error
  matching `ErrInvalidRequest`, where v1.32.0 sent `max_tokens` 8192, the
  default of `WithMaxTokens`; a Kilo request in that state sends no limit,
  where v1.32.0 sent 8192. The refusal names
  `WithMaxTokens` or `catalog.WithMetadata` as the fix (0093-MADR D8, D9,
  Q2).

### Removed

- `catalog.LookupMetadata`, `catalog.LookupMetadataWith`,
  `catalog.CachedMetadataWith`, `catalog.OptionsFromEnv`,
  `llmprovider.WithModelMetadataURL`, `llmprovider.WithoutModelMetadata`,
  `(*llmprovider.Settings).ModelMetadataURL` and
  `(*llmprovider.Settings).ModelMetadataDisabled`, in a minor release, as
  the exception at the top of this file says: the SDK no longer downloads
  model metadata. Download a models.dev document, such as
  `https://models.dev/api.json`, read it with `catalog.ParseMetadata`, and
  hand it in with `catalog.WithMetadata`; `catalog.WithMetadata(catalog.Metadata{})`
  given last drops one an earlier option carried, where
  `WithoutModelMetadata` turned the fetch off (0093-MADR D10, Q1).
- The download of `https://models.opencode.ai/api.json`, its ten-minute
  cache in `catalog.Cache` and the process's, its background refresh and
  `ETag`, and its one-minute memory of a failure. The variables
  `LLMPROVIDER_MODELS_METADATA_URL` and `LLMPROVIDER_DISABLE_MODELS_METADATA`
  have no effect (0093-MADR D6, D10).

### Fixed

- `TextPrompter` drops a late LF even when a signal interrupts its wait, which then goes on for the time left ([0087-MADR](docs/decisions/0087-MADR-late-line-feed-after-a-hidden-entry.md) A-8).
- A wizard run whose context ends, by a cancel or its deadline, during the
  Ollama endpoint check no longer shows "cannot reach" or asks "Try a
  different endpoint?": it returns the context's error, as
  `enter base URL: ...`, and under `Options.NoEndpointPrompt` as
  `wizard: endpoint check ended: ...` instead of
  `wizard: cannot reach <url>: ...`; the error still matches
  `context.Canceled` or `context.DeadlineExceeded`
  ([0090-MADR](docs/decisions/0090-MADR-a-cancelled-run-does-not-ask-about-the-endpoint.md)
  S78).

## [v1.32.0] - 2026-11-22

### Added

- `wizard.Options.NoLoopback`: a browser sign-in, OpenAI's, Grok's or the
  ChatGPT plan's, binds no callback listener and asks no question, and a
  pasted callback address or code completes it, in words that say so;
  `Options.OpenURL` is still called. With `Options.NoPastePrompt` it is an
  error matching `llmprovider.ErrInvalidRequest`, before any question. And
  `wizard.Options.NoBrowserQuestion`: no question about where the browser is,
  and the listener binds, as it did (0083-MADR).
- `llmprovider.ErrToolSchemaRefused`, beneath `ErrInvalidRequest`, for a
  service's refusal of a tool's schema, and `APIError.Tool`, the refused
  tool's name when the service named exactly one tool the request sent,
  else empty. The kind is matched by service and marker, from an HTTP error
  reply, on every route of the service whose reply carries the marker: Grok's
  `[invalid_client_tool_schema]`, recorded on its Responses route; the code
  `invalid_function_parameters`, recorded on OpenAI's and OpenCode Zen's
  Responses routes, and read on OpenAI's API key, ChatGPT session and plan
  session and on every OpenCode route; Anthropic's `tools.N.` path naming
  `input_schema`, recorded on its Messages API and read for Claude alone;
  and `functionDeclaration parameters schema`, recorded on OpenCode Zen's
  Google route and read on every OpenCode route.
  The name is only ever one the request's body held, after deferral
  (Anthropic's index is into the list as sent), and the SDK still rewrites,
  drops and resends nothing. A refusal a service has reworded, or one in
  another form, stays `ErrInvalidRequest`; a failure inside a 200 or a stream
  keeps today's kind; a reply marked `x-should-retry: true` is still
  `ErrProviderUnavailable`, and an overflow still `ErrContextOverflow`. Such
  an error's text now begins "llmprovider: tool schema refused", where it
  began "llmprovider: invalid request"
  ([0084-MADR](docs/decisions/0084-MADR-typed-error-for-a-refused-tool-schema.md)
  D1 to D4).
- `signin.VerifyKilo(ctx, key, organization, opts...)`, in the experimental
  `llmprovider/x/signin` and so outside the v1 promise: it checks a Kilo key
  against Kilo's profile, for a caller that drives its own sign-in. It
  returns nil when the profile answers, when the credential has an
  organization (given, or named by a token's embedded URL or the base URL's
  path) and when the profile cannot say, a stall past its own 10 seconds
  included; an error matching `llmprovider.ErrAuthFailure` when Kilo refuses
  the key; `ctx`'s error when the caller's own `ctx` has ended; and an error
  matching `llmprovider.ErrInvalidRequest`, with no request, for options that
  are wrong
  ([0080-MADR](docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  D10).

### Changed

- Every browser sign-in the wizard runs, OpenAI's and Grok's and the ChatGPT
  plan's, first asks once per run where the browser is, with one
  `Prompter.Select` after the method menu and, for the plan, before its
  consent, unless the caller sets `Options.NoLoopback`,
  `Options.NoBrowserQuestion`, `Options.NoPastePrompt` or `Options.Driver`. A
  caller whose scripted `Prompter` refuses an unexpected `Select` scripts the
  answer (index 0, "On this machine") or sets `NoBrowserQuestion`. "On another
  device" runs the sign-in with no callback listener, to be completed by a
  paste, and does not call `Options.OpenURL`; an empty line, and an input that
  ends at the question (its first read at its end), are "On this machine",
  while an input already exhausted is an error that ends the run.
  `Options.NoLoopback` with
  `Options.NoPastePrompt` is an error matching `ErrInvalidRequest` before any
  question (0083-MADR).
- When ports 1455 and 1457 are both taken, the error keeps its prefix, "oauth:
  registered callback ports unavailable", and advises freeing the two ports,
  and for the browser sign-in the device-code login, but no longer a
  device-code login for the ChatGPT plan, which has none. A bind that fails
  for another reason, and Grok's, advise the sign-in with no listener: in the
  wizard, to answer "On another device: I will paste the address back" when it
  asks where the browser is; in code, or where `NoBrowserQuestion` or a
  `Driver` means it is not asked, to set `NoLoopback` (0083-MADR D8, ST70 as
  amended 2026-11-22).
- A pasted redirect address that begins with the callback's host and port and
  has no `http://`, as the address bar shows it, is read as the address, with
  every check an address meets, for every caller of the sign-in engine,
  `x/signin`'s included; one that does not parse is refused and asked again,
  and one that is the host and port alone is refused as cut short and asked
  again (0083-MADR D11, ST89).
- With no callback listener, a `Prompter` that answers the paste with nothing,
  three times at once, ends the sign-in with an error matching
  `ErrInvalidRequest`. The notice that asks for an Enter after a sign-in's end
  says "Sign-in cancelled" after a cancel and "Sign-in timed out" after the
  login's limit, in every flow, where it said "Browser sign-in finished"
  (0083-MADR ST38, ST72).
- `llmprovider.NewStaticToken`, and so `WithAPIKey`, store a key that is
  empty once trimmed (spaces, a tab, a line break, a non-breaking space) as
  `""`. Each provider then does what it does with an empty key: Claude,
  Gemini, Grok, Hugging Face, OpenAI and Together refuse it at `New` with an
  error matching `ErrInvalidRequest`; Kilo, OpenCode Zen and OpenCode Go send
  their anonymous or public key, keeping a caller's `Header`; Ollama and a
  Chat Completions server send no `Authorization`. What changes depends on
  the key. A key of only spaces or a non-breaking space was built and sent
  (Ollama sends a key only under a caller's `Header`): a blank environment
  variable passed to `WithAPIKey`, as this module's own example does, is no
  longer sent, and the six that refuse an empty key now refuse it at `New`
  where the spaces failed at the service; Kilo and OpenCode now give
  anonymous use where they sent the spaces. A key of only a tab, a lone CR, a
  lone LF, CR+LF or a mix holding one of them, which every provider already
  refused at `New` as holding a control character, is now an empty key too:
  Kilo, OpenCode Zen and OpenCode Go go from refused to anonymous or public
  use; Ollama and a Chat Completions server go from refused to built, with no
  `Authorization`; the six keep the same kind, refused with
  `ErrInvalidRequest`, with the text of an empty key ("<id> api key is
  required"). A key that is not blank is sent as given, padding included, and
  one holding a control character, such as `"key\n"`, is still refused at
  `New`. A `StaticToken` built as a struct literal is not changed.
  `signin.VerifyKilo`, `catalog.KiloModelCapabilities` and `auth.KiloProfile`,
  which take the key as a string, are not changed. Together's model listing
  also sends no `Authorization` for a token whose value is empty, where it
  sent a bare "Bearer", as the Chat Completions listing already did
  ([0085-MADR](docs/decisions/0085-MADR-refuse-an-empty-typed-key.md) D9,
  ST71 to ST77, ST82).
- An error reply is now read to 1 MiB for every provider, where it was read
  to 64 KiB, so a body between the two is parsed whole instead of cut inside
  its JSON, and other kinds can change too: a 200 KiB quota 429 from OpenAI
  is now `ErrQuotaExhausted`, where it was a plain rate limit. A body whose
  JSON the read cuts keeps the kind it had. The parse of an error body
  reads at most 32 `details` entries and at most 32 quota violations, so a
  large body costs time in proportion to its size and `Limit.Quotas` holds
  at most 32 entries
  ([0084-MADR](docs/decisions/0084-MADR-typed-error-for-a-refused-tool-schema.md)
  ST86, ST87).
- The wizard checks a Kilo credential against Kilo's profile, under
  `Options.Discover` and when no organization is set, and refuses it on a
  401: a wrong Kilo key is no longer saved to fail on first use, because
  Kilo's public model listing answers every credential. A typed key is asked
  for again, up to three; a key from `KILO_API_KEY`, a saved key, a device
  key or a device session ends the run naming its source; a kept session
  leads to the sign-in menu, once. The error matches
  `llmprovider.ErrAuthFailure`, as a listing's refusal does. A profile that
  cannot answer, a 403, a 5xx, a failed connection or a stall past
  `Options.DiscoverLimit`, is one warning, "could not verify the Kilo key
  (...); going on without verifying it", and the sign-in goes on. Without `Discover`,
  or with an organization, nothing is read that was not before. The
  organization is the one the listing is given, by its real order: the last
  `catalog.WithKiloOrganization` among the options, where the options outside
  `llmprovider.For` apply first, then those inside
  `For(llmprovider.ProviderKilo, ...)`, and the wizard's own `For`, with a kept
  session's organization or a device session's choice, comes before any `For`
  in `Options.ProviderOptions`; with none, a token's embedded URL's, then the
  endpoint's path's. Four limits: a key valid only for an
  organization, typed with no organization entered, is read at the profile,
  which was not verified live for such a key; with Kilo's other gateway
  address `https://ai-gateway.kilo.ai/api/v1` as the endpoint the profile
  answers 404, so every sign-in there warns and a wrong key passes with that
  warning; the organization listing that an endpoint's own organization path
  names is sent without the organization header, and that listing's refusal
  of a wrong key was not probed live; and a key that a caller's own
  `Prompter` returns empty, or of only spaces, is asked for again, and after
  three is an error, for every provider, so it never reaches the check
  ([0085-MADR](docs/decisions/0085-MADR-refuse-an-empty-typed-key.md)).
  Under
  `Options.KiloDeviceKey` the profile is now read once to check the key;
  a Kilo device session's own read of the profile is its check, with no
  second request, and under `Discover` its 401 is a refusal where it was a
  warning, and its other failures are two notices: the organization menu's
  warning that the organizations could not be read and the personal account
  is used, then the one warning above
  ([0080-MADR](docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  D1 to D9).
- `ConfigureLLM` saves a new session as its last step, after the model and
  fallback questions: a run abandoned there, or ended by an error, keeps the
  session the store held, and an error from it means nothing was saved, except
  when the store's own `Save` fails after it has written (`FileTokenStore`'s
  directory sync after the rename): that error comes back with the new session
  in the store. Two
  runs saving to one store leave the session of the last to commit. A sign-in
  abandoned at a model question is not kept, and nothing the provider issued
  is revoked: a Kilo device sign-in's token does not expire, so one abandoned
  stays on the Kilo account until the person removes it there
  ([0081-MADR](docs/decisions/0081-MADR-a-cancelled-sign-in-saves-nothing.md)
  D5, D6).

### Fixed

- The wizard saved a new sign-in when the run's context ended before the save,
  during the model listing or after the sign-in, and returned the context's
  error, or, at a model question, a Result; it now saves nothing and returns
  the context's error, for every provider, session kind and key, by cancel or
  by the run's deadline. `ResolveCredential` returns its Result when the
  context ends after the session is saved. A run whose context has ended
  shows none of the model listing's warnings and not the Kilo account's
  organizations warning, and Kilo's unverified-key warning no longer says the
  key is saved
  ([0081-MADR](docs/decisions/0081-MADR-a-cancelled-sign-in-saves-nothing.md)
  D2 to D5).
- The wizard returned a typed or pasted key that was empty once trimmed as a
  successful `CredAPIKey` with no key, for every provider that requires one,
  when a caller's `Prompter` returned it. It now gives a notice and asks
  again, up to three asks in all for each prompt, and then returns an error
  that matches no sentinel, with no Result; the pasted Grok key and OpenAI's
  paste are asked again as the typed key is, where OpenAI's blank paste was
  an error at once. A key the provider refused, followed by three blanks,
  now ends with that error, which does not match `llmprovider.ErrAuthFailure`
  where the run's error did, so a caller that reads that sentinel to report
  the keys it asked for sees none for that run. An environment value or an
  `Options.Existing` key that is empty once trimmed is no longer offered. A
  repeat ask first checks the run's context, and an ended one returns its
  error. The package's own `TextPrompter` already asked again for a blank
  entry
  ([0085-MADR](docs/decisions/0085-MADR-refuse-an-empty-typed-key.md)
  D1 to D5, D8, D10).
- `TextPrompter` let the LF of an Enter that reached it as CR then LF, in two
  reads, answer the question after a hidden entry: the model search was
  answered blank, a workspace or model question took its default, and a typed
  key asked again warned of an empty entry. A reviewer of 0085 found it. A
  hidden entry ended by a CR on a terminal now waits up to 50 ms, still in raw
  mode, and drops an LF that arrives in that time; the person's next Enter at
  the next question still answers it with its default (0020-MADR F17). The
  wait is `select(2)` on Unix and the console's pending key events on
  Windows, where the LF showed as a second hidden entry's empty-entry
  warning. `golang.org/x/sys` is now a direct requirement. Consumers:
  prepare-commit-msg takes the fix by bumping from `v1.29.0`, with no code
  change; gobble-cli and magic-cli-remote do not use `TextPrompter`; a caller
  that implements `Prompter` over a raw terminal reads the new
  `Prompter.Secret` sentence; mcplib's copy is unchanged
  ([0087-MADR](docs/decisions/0087-MADR-late-line-feed-after-a-hidden-entry.md)
  A1 to A7).

### Security

- A refreshable `OAuthSession` no longer adopts a session its token store
  holds that is another sign-in (another provider, kind, issuer or client, or
  another account when both name one), which it did at its next refresh,
  sending that login's token on the route it was built for and spending its
  refresh token. Its token requests now fail with an error matching
  `llmprovider.ErrAuthFailure`, which names the field that differs and no
  value the store holds, until the store holds its sign-in again, or nothing;
  nothing is sent or saved for the other sign-in, and the session keeps its
  own. This holds for a stored record with no refresh token too, such as a
  pasted access token. A rotation of the same sign-in by another process is
  adopted as before. An empty issuer is the kind's built-in issuer on both
  sides, as everywhere else in the module; an empty client is never the same
  as a named one. In a running process, a sign-in by the other OpenAI method,
  to another account, or to a plan session without a `ClientStore` in another
  process of the same application now ends in an auth failure that asks for a
  new sign-in, where it was adopted. Three more places follow:
  - the refresh's own save no longer overwrites a sign-in written to the store
    while the refresh request was in flight: under the store's refresh lock,
    the store is read again and written only if it still holds this sign-in's
    token or nothing; otherwise the rotation stays in memory and the next due
    refresh is refused. A store that is not an `auth.RefreshLocker` gets the
    same check, not atomic with the save;
  - the wizard saves a new sign-in under that lock when the store has one:
    it tries the lock once, tells the person it is waiting when another
    process's refresh holds it, and waits up to the store's own bound (60 s
    for `auth.FileTokenStore`); at the store's limit, whatever error it
    ends with (gobble's own included, not only one matching
    `llmprovider.ErrProviderUnavailable`), it keeps the completed sign-in
    and asks whether to try the save again, and a refusal, or a caller with
    no prompter or no answer to give, gets the store's error, with nothing
    saved; the lock is held only when the first try's error wraps
    `context.Canceled`, as `auth.RefreshLocker` documents, and any other
    failed try is not called another process: there is no wait, and the
    person is asked in neutral words, with the error shown, whether to try
    again; the sign-in's context ends the wait at any point, saving
    nothing. It signs in to a stored plan
    session again only when it is OpenAI's own and, with a client record,
    that client's, so another application's issued client is not reused. A
    client record that exists but cannot be decoded is treated as none, with
    one notice that shows nothing of the file: no plan session is then
    signed in to again, the person registers a new client, and the new
    client replaces the record. Any other error reading the record, a
    permission refused or an input or output error, fails the sign-in
    naming the file, with no notice, no registration and the record as it
    was, and the record is read once per sign-in. The sign-in engine under
    `x/signin` replaces an undecodable record silently, where it failed the
    sign-in, and fails `Wait` on any other read error; a decode failure is
    told from the rest by wrapping `encoding/json`'s error, which
    `FileTokenStore` does (`auth.ClientStore` documents it). A client record
    file over 64 KiB, which a real one never is, is corrupt too and replaced
    the same way, where it failed every sign-in; a session file over 64 KiB is
    still refused;
  - `OAuthSession.StoredConsent` counts a stored record that is another
    sign-in as unanswered, so a held plan session no longer reads another
    application's `background_use` answer.

  A custom `auth.TokenStore` must keep every field that names a sign-in
  (`Provider`, `Kind`, `Issuer`, `ClientID`, `AccountID`) as saved; one that
  drops the client, an issuer that is not OpenAI's or xAI's own, or a kind the
  other fields do not infer makes its sessions refuse each other's rotations
  with this auth failure, and its user signs in again. A session that no
  other process shares is not affected: a record holding its own refresh
  token is its own, and so is one with no refresh token that holds its own
  access token, the current one or the one the store is known to hold, so a
  store that keeps neither the refresh token nor the client still serves its
  one process, a failed save included; a record with a refresh token of its
  own, a blank one included, is judged by its labels whatever access token it
  carries. A writer of the store can plant an access-only record with the
  session's stale access token, which is taken as its own until the next save
  overwrites it; that needs write access to the session's own record
  ([0082-MADR](docs/decisions/0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md)
  D1 to D5 and its amendments of 2026-11-17 and 2026-11-18).

## [v1.31.0] - 2026-11-14

### Added

- Nine facts on `llmprovider.AuthMethod`, set by the OpenAI, Grok and Kilo
  descriptors: `Kind`, `Completion`, `Paste`, `FirstPaste`, `Yields`,
  `ClientIdentity`, `Slot` (strings), and `Refreshable` and `Revocable`
  (booleans). They spell the wire values of the new auth contract module, so
  a caller reads what a method is instead of listing providers
  ([0074-MADR](docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md)
  D1).
- `auth.SessionKind`, with `auth.SessionChatGPT`, `SessionChatGPTPlan`,
  `SessionGrok` and `SessionKiloDevice`; `OAuthSession.Kind`, saved by
  `FileTokenStore` and omitted when empty; and `(*OAuthSession).SessionKind`,
  which names the kind a session has, inferred when `Kind` is empty (D5).
- `(*auth.OAuthSession).UseStore`, which binds a session to its `TokenStore`
  under the session's lock, so it is safe while another goroutine, such as a
  listing that is refreshing it, uses the session; the wizard's save of a
  new sign-in now calls it. A nil store changes nothing, and the last call
  wins
  ([0074-PLAN](docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md)
  Phase 7 step 6).
- `auth.ErrSignInDenied` and `auth.ErrSignInExpired`, each matching
  `llmprovider.ErrAuthFailure`, which the browser, Grok, OpenAI and Kilo
  sign-ins now wrap on a denial or an expiry; no error's text changes (D8).
- `auth.OAuthFlowOptions.NoLoopback` and `auth.ChatGPTPlanOptions.NoLoopback`
  bind no callback listener, so a paste alone completes a browser sign-in;
  they need an `InputCode`, and a device sign-in ignores them (D4).
- `auth.VendorCLIPath`, the path of a vendor CLI's login file, which the
  wizard's vendor import now calls.
- `wizard.Options.Driver` and `wizard.StepDriver`, whose method `SignInStep`
  is given each step of a browser, plan or device sign-in, for a caller with
  no terminal (D6). The last step, `done` or `failed`, has 5 s: its context
  is detached from the sign-in's and ends after 5 s, and an error or a
  timeout there is one warning, with the credential kept on `done`.
- `llmprovider/x/signin`, experimental (outside the v1 promise and
  `make api-check`): `Methods`, `Start`, `Options` and its fields (`NoLoopback`
  among them), `SignIn` with `Method`, `Steps`, `Answer`, `Wait` and `Cancel`,
  `Credential` with `APIKey`, `Session`, `AccessOnly`, `DeviceKey` and
  `ReadThrough`, and `ErrorKindOf` (D2, D3, D9).
- `(*auth.OAuthSession).UseIdentity`, which gives a session the identity of
  the application using it, so that its refresh and revocation name the
  application; and `(*llmprovider.Settings).ClientVersion`, the version
  `WithClientInfo` gave, or the one the User-Agent shows. The OpenAI and
  Grok providers and `catalog` call them with `WithClientInfo`'s name and
  version
  ([0079-MADR](docs/decisions/0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md)
  D2, D4).

- The ChatGPT plan's consent, as 0075-MADR decides it
  ([0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](docs/decisions/0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)):
  `llmprovider.AuthMethod.Consent`, `ConsentText` and `ConsentURL`, set by the
  `chatgpt_plan` descriptor, and `openai.ConsentBackgroundUse`;
  `auth.OAuthSession.Consents`, saved by `FileTokenStore` and omitted when
  empty, with `(*OAuthSession).Consent`; `auth.SetConsent`; and
  `(*OAuthSession).StoredConsent`, which reads the answer the session's store
  holds (D1 to D3, D5).
- `llmprovider.WithBackground` and `llmprovider.IsBackground`, which mark a
  request the application makes on its own; the openai provider refuses a
  marked request on a ChatGPT plan session whose stored consent is not
  granted, or cannot be read, with `llmprovider.ErrNotPermitted`, before the
  network (D5).
- `auth.ClientRecord`, `auth.ClientStore` and `(*FileTokenStore).LoadClient`
  and `SaveClient`, which keep the client OpenAI issued the app and its host
  id in `<provider>.client`, apart from the session;
  `auth.ChatGPTPlanOptions.ClientID`, which signs in to an issued client with
  no `Previous`; and `wizard.Options.HostID`, `wizard.Options.ConsentText`,
  and `x/signin`'s `Options.HostID`, `Options.ConsentText` and
  `Options.ClientStore` (D4, D6, D7).

### Deprecated

- `llmprovider.AuthMethod.Interactive` and `HeadlessOK`, which `Kind` and
  `Completion` replace, `auth.LoginDeviceOAuth`, which `StartDeviceOAuth`
  replaces, and `auth.OAuthFlowOptions.NotifyDevice`. Each keeps its
  behaviour until v2 (D7).

### Changed

- The wizard's plan notice is one question with three choices, the notice and
  the background-use consent together, asked before the browser opens:
  allow, do not allow (the default) or cancel the sign-in, which ends it as
  declining the notice did. A saved plan session with no recorded answer is
  asked too. A plan session saved by an earlier release has no answer, so a
  request marked with `WithBackground` is refused until the user answers.
  An older release that refreshes and re-saves a session drops its consent,
  and ignores `<provider>.client` (0075-MADR D4 and Consequences).
- The ChatGPT backend's `originator` is `WithClientInfo`'s name, for
  generation, the model listing and `catalog.ListRaw`, where it was always
  `go-llmprovider-sdk`; with no name it is unchanged, and a ChatGPT plan
  session still sends none (0079-MADR D1).
- A session's refresh and revocation, and the discovery a revocation
  reads, lead their User-Agent with the application that signed it in, or
  that `UseIdentity` gave it, where they named this module only. A refresh
  that `catalog.CachedModel`, `LookupModel` or `CachedFacts` starts names
  `WithClientInfo`'s application, as a listing's does. With no name, every
  request is as before (0079-MADR D2 to D4).
- The wizard's own requests, the listing check, the Ollama check, the Kilo
  profile and the listing of a ChatGPT session, and its logout, name
  `Options.Identity`, unless `ProviderOptions` carry a `WithClientInfo`
  (0079-MADR D5).
- An `Options.OpenURL` error in the wizard is shown as a warning, "could not
  open the browser (...); open the URL above", and the sign-in goes on.
- The wizard's browser and plan sign-ins check a pasted callback before
  `auth` receives it: its state, a code, and a `client_id` where only a URL
  is accepted. A paste that fails is asked again with a warning that gives
  its reason, and a third bad one ends the sign-in.
- The vendor CLI's login path errors of the wizard's import begin "auth:"
  where they began "wizard:", and the error for a provider with no vendor CLI
  matches `llmprovider.ErrUnsupported`.
- The root module requires `authcontract` v0.1.0, and `make api-check`
  compares the `llmprovider` and `wizard` subtrees separately.
- Grok's browser sign-in completes by a pasted callback URL or code, and
  accepts `NoLoopback`: its `browser_oauth` method fact is `browser_paste`
  with `Paste` `url_or_code`. With `NoLoopback` and no listener its redirect
  names `http://127.0.0.1:<port>/callback`, a random port in 49152 to 65535
  that nothing binds. Verified by a live sign-in on 2026-11-13
  ([0074-PLAN](docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md)
  Phase 3a).

### Fixed

- Two data races between a session's refresh and the wizard's save of a new
  sign-in, while a listing was still using the session: the wizard set the
  session's `Store` field with no lock, and `FileTokenStore.Save` read the
  session's fields with no lock while a refresh wrote them. The wizard now
  calls `UseStore`, and `FileTokenStore.Save` reads the session under its
  lock. A `TokenStore` outside this module cannot take that lock, so its
  doc now says such a store must not be given a session in use
  ([0074-PLAN](docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md)
  Phase 7 step 6).

## [v1.30.0] - 2026-11-11

### Added

- `auth.Identity`, with `Name` and `Version`, names the consuming
  application to the identity providers on a sign-in: OpenAI's
  `originator`, Grok's `referrer`, and the User-Agent of every request of
  the sign-in, which names this module after it.
  `auth.OAuthFlowOptions.Identity` and `auth.ChatGPTPlanOptions.Identity`
  take it, and `wizard.Options.Identity` gives it to every sign-in the
  wizard runs. Its zero value names this module, as before, and a refresh
  and a revocation still do. Whether OpenAI and xAI accept every name is not
  verified
  ([0076-MADR](docs/decisions/0076-MADR-auth-error-kinds-and-doc-drift.md)
  D3, D4).

### Changed

- A revocation's unsupported error wraps `llmprovider.ErrUnsupported`, so
  its text ends "llmprovider: unsupported" where it ended "unsupported
  operation"; the words before it are unchanged (0076-MADR D1).

### Fixed

- `auth.RevokeOAuthSession` returned the standard library's
  `errors.ErrUnsupported` for a session of a provider with no revocation,
  and for a Grok or ChatGPT plan issuer that publishes no
  `revocation_endpoint`, so `errors.Is(err, llmprovider.ErrUnsupported)`
  was false. It now returns an error matching `llmprovider.ErrUnsupported`,
  which still matches `errors.ErrUnsupported` (0076-MADR D1).
- An OpenAI device-code sign-in whose deadline passed between its two
  checks returned a plain error. It now matches `llmprovider.ErrAuthFailure`,
  as every other device-code expiry does (0076-MADR D2).
- Four docs said what the code does not (0076-MADR D5):
  - `FileTokenStore.LockRefresh` gives up after 60 s, where its doc said
    40 s, and its doc now names the two kinds a `ctx` that ends while it
    waits gives: seen ended between attempts, an error matching both
    `ErrProviderUnavailable` and `ctx`'s; ended during the pause between
    attempts, `ctx`'s error alone;
  - an `OAuthSession` formatted with `%#v` begins `&auth.`, where it began
    `&llmprovider.`;
  - `RevokeOAuthSession`'s doc names the ChatGPT plan session's form, which
    it sends first;
  - [0006-MADR](docs/decisions/0006-MADR-subscription-auth-for-llm-providers.md)
    §7 is amended: the refresh uses the session's stored client, and the
    vendor import reads the default Grok client's login only.

### Security

- The module requires Go 1.27.4, for GO-2026-6859 in `net/http`, an HTTP/1
  server crash that govulncheck reports as called, through the loopback
  callback of the browser sign-ins, as
  [0077-MADR](docs/decisions/0077-MADR-go-1-27-4-for-go-2026-6859.md)
  records. A consumer at `go 1.27.3` raises its own `go` line to take this
  release; with `GOTOOLCHAIN=auto`, the default, `go` fetches the
  toolchain. `v1.29.0` and earlier build safely only with a Go 1.27.4
  toolchain. No exported name changes.

## [v1.29.0] - 2026-11-10

### Added

- `APIError.ResetAt` is the instant the service named for a limit to lift,
  in UTC: a usage limit's reset time in the error body (ChatGPT's
  `resets_at`, OpenCode's `metadata.resetsAt`), else an HTTP-date
  `Retry-After`, else, on a 429, Claude's rate-limit reset header. It is zero
  when the service named only a delay, and is kept as stated, so it may have
  passed. `RetryAfter` is unchanged
  ([0072-MADR](docs/decisions/0072-MADR-keep-absolute-reset-times-and-limit-detail-on-apierror.md)
  D1).
- `APIError.Limit`, of the new type `Limit`, is what the service stated
  about the limit it hit: OpenCode Go's limit name, a window ChatGPT gives in
  minutes, the ChatGPT plan, Gemini's per-day flag and its violated quotas
  as `Quota` values, and Kilo's link to add credits. A window is never
  inferred from how long the wait is (0072-MADR D2).
- `ClassifyStreamFailureBody` classifies a failure inside a 200 event stream
  as `ClassifyStreamFailure` does, and also reads the stream's error object
  with the HTTP body's parser. Every built-in stream reader uses it
  (0072-MADR D3).

### Fixed

- An OpenCode `BudgetLimitError` (429) is `ErrQuotaExhausted` and is no longer
  retried; `CreditLimitExceeded` is `ErrQuotaExhausted` whatever its status
  (0072-MADR D4).
- A Gemini per-day quota reported inside a stream is `ErrQuotaExhausted`, as
  in an HTTP reply, and no longer retried.
- A failure inside a stream carries the delay its error states: a
  `headers` object's `Retry-After`, a reset time, a Google `RetryInfo`, or,
  for a rate limit, "try again in 11.054s" in its message
  (0072-MADR D3).

## [v1.28.0] - 2026-11-07

### Added

- `catalog.Cache.Close` ends a cache's life. It stops the listing fetches,
  models.dev refreshes and probe runs that calls given the cache would
  start in the background, cancels those in flight, and waits for them,
  under the caller's context. After it, `List`, `LookupModel` and
  `LookupMetadataWith` given the cache refuse with an error matching
  `llmprovider.ErrInvalidRequest`. The cached reads answer from what it
  holds. The process's cache is unchanged
  ([0071-MADR](docs/decisions/0071-MADR-give-a-cache-a-lifetime-and-wait-for-a-refresh.md)
  D1, D2).
- `auth.OAuthSession.Wait` waits for a refresh that runs on after its
  caller has gone, without cancelling it, so that a test or a process that
  is ending keeps the rotated session (0071-MADR D3).

### Fixed

- Two background listing starts counted their fetch only after releasing
  their lock, so nothing could wait for them reliably. They now count it
  under the lock (0071-MADR D1).
- Concurrent `List` calls of one key could send a second listing request.
  A caller that found no fresh listing claimed the key only after the first
  fetch had stored its listing and ended. The claim now checks for the
  fresh listing again
  ([0053-MADR](docs/decisions/0053-MADR-stream-retry-and-cache-defects.md)
  D4; 0071-PLAN Deviation 4).

## [v1.27.0] - 2026-11-06

### Added

- `catalog.SortByRank` sorts a list of model ids by `Rank`, reading each
  family's version numbers from the list, and from release dates the
  caller gives. `catalog.LabelName` and `catalog.LabelNote` return a
  curated label's name and note, unpadded
  ([0070-MADR](docs/decisions/0070-MADR-place-grok-4-20-by-date-and-split-model-labels.md)
  D1, D2).

### Changed

- Eight label notes:
  - `grok-4.6`'s no longer calls it the Grok CLI's default, which is
    grok-build's compiled-in fallback;
  - `grok-4.5`'s "previous flagship" was false;
  - the notes that ranked a model in time no longer do;
  - `claude-opus-4-8`'s names its generation
    ([0070-MADR](docs/decisions/0070-MADR-place-grok-4-20-by-date-and-split-model-labels.md)
    D2).

### Fixed

- `catalog.Rank` read `grok-4.20-*` as generation 4.20, above `grok-4.7`,
  though xAI released it before `grok-4.3`. A family's two-digit minor is
  now read as an integer or a decimal, from the family's dates and
  siblings, so `grok-4.20` is 4.2 and `gemini-3.10` still follows
  `gemini-3.9`
  ([0070-MADR](docs/decisions/0070-MADR-place-grok-4-20-by-date-and-split-model-labels.md)
  D1).

## [v1.26.1] - 2026-11-04

### Fixed

- An event the stream ends inside, with no blank line after it, is no
  longer read. It was, against the server-sent events specification. The
  stream now ends with `ErrProviderUnavailable`, naming the event it
  lacked, as a stream that ends early does. A stream cut short on a
  connection's close is now `ErrProviderUnavailable`, which `WithRetry` and
  `WithStreamRetry` ask again at most once, where it was a decode error with
  no kind, never retried. Every server measured ends its events as
  the specification requires
  ([0069-MADR](docs/decisions/0069-MADR-discard-an-sse-event-the-stream-ends-inside.md)
  D1).

## [v1.26.0] - 2026-11-04

### Changed

- The shared transport, which `NewHTTPClient` returns and every provider
  built with no client uses, pings an HTTP/2 connection that has received
  nothing for 30 s, and closes it when the ping goes unanswered for 15 s.
  A pooled connection whose peer has gone, after a sleep or a network
  change, fails its next request within 45 s, where it held the request
  for the whole 300 s first-byte bound. HTTP/1.1 connections are unchanged
  ([0068-MADR](docs/decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md)
  D1).
- `WithRetry` no longer retries HTTP 501 or 505: `ClassifyHTTPError` makes
  both terminal, with `ErrProviderUnavailable`, as 525 and 526 are. An
  OAuth session's refresh retries any 5xx, as before
  ([0068-MADR](docs/decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md)
  D2).
- `FileTokenStore`'s refresh lock waits 60 s for another holder, where it
  waited 40 s
  ([0068-MADR](docs/decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md)
  D3).
- `CommandToken` returns a failed run's error for 5 s without running its
  command again, unless `Invalidate` is called; and once its output is
  four fifths of its TTL old, the next `Token` returns it and starts one
  run in the background, so a token in steady use is renewed before it
  expires; with no call in the TTL's last fifth, the next caller waits for
  the command as before
  ([0068-MADR](docs/decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md)
  D4).

### Fixed

- A TLS certificate the client refuses is no longer retried, by `WithRetry`
  or by an OAuth session's refresh, whatever kind the failure was wrapped
  with: it cost `MaxAttempts` attempts and the waits between them, and
  three refresh attempts, where none could succeed
  ([0068-MADR](docs/decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md)
  D2).
- A process waiting for another's refresh lock no longer gives up while the
  holder is within its own bounds: a refresh may hold the lock for 45.6 s,
  and the waiter gave up at 40 s
  ([0068-MADR](docs/decisions/0068-MADR-five-internal-fixes-from-the-0054-audit.md)
  D3).

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

[Unreleased]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.33.0...HEAD
[v1.33.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.32.0...v1.33.0
[v1.32.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.31.0...v1.32.0
[v1.31.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.30.0...v1.31.0
[v1.30.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.29.0...v1.30.0
[v1.29.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.28.0...v1.29.0
[v1.28.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.27.0...v1.28.0
[v1.27.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.26.1...v1.27.0
[v1.26.1]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.26.0...v1.26.1
[v1.26.0]: https://github.com/maccavelli/go-llmprovider-sdk/compare/v1.25.0...v1.26.0
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
