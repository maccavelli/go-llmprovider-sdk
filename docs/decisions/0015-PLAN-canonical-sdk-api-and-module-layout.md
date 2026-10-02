---
status: in-progress
date: 2026-10-01
associated-madr: "0015-MADR-canonical-sdk-api-and-module-layout.md"
decision-makers: go-llmprovider-sdk maintainers
---
# Implement the Canonical, Modular v1 API for go-llmprovider-sdk

Associated MADR: [0015-MADR-canonical-sdk-api-and-module-layout.md](0015-MADR-canonical-sdk-api-and-module-layout.md)

## Goal

At the end of this plan, before `v1.0.0`:

1. The module has the D2 layout:
   * `llmprovider` (the contract);
   * `auth` and `catalog`;
   * one package per provider;
   * `providers`;
   * `llmtest`;
   * `internal/wire` and `internal/transport`;
   * `wizard` and `internal/redact`.
2. Every provider implements the D3 contract. It declares its D4
   capabilities, is built through D5 options, and passes `llmtest` (D11).
3. Functional parity with `mcplib` `v1.6.0` is proven by G-wire, the ported
   tests and G-parity (D12).
4. The standards exist as `docs/guides/api-standards.md` and are enforced as
   D13 states.
5. [0002-PLAN-migrate-llmprovider-from-mcplib.md](0002-PLAN-migrate-llmprovider-from-mcplib.md)
   can resume at its Phase 8.

## Scope

### In scope

* `llmprovider/**`, `wizard/**`, `internal/**`
* `docs/guides/**`, `docs/architecture.md`, `docs/README.md`, `README.md`,
  `AGENTS.md`
* `Makefile`, `scripts/**`, `.github/workflows/ci.yml`
* this pair, and its REPORT

### Out of scope

The MADR's "Out of scope" list:

* native streaming for any given provider;
* image and file items;
* structured output;
* new providers;
* a second module.

Also out of scope:

* any dependency beyond 0002-MADR §2;
* any wire change not made by a record.

## 0. Preconditions and conventions

* **Start.** This plan starts when 0002-PLAN Phase 7 is committed. G-wire's
  baseline is the tree at that commit, called `P7` below.
* **Gate, every phase.** Each phase ends green on the gate below, then
  commits with `git commit --no-edit`:
  * `make pre-add-check`;
  * `go vet` with `GOOS` darwin, linux and windows, and with
    `-tags live_gateways`;
  * `go test -race -count=1 -cover ./...`;
  * `go mod tidy -diff`;
  * `make lint`;
  * G-wire, once it exists.
* **Intermediate phases may break the API.** No consumer imports this module
  before `v1.0.0-rc.1`, so API breaks between phases are allowed. A failing
  build or test is not.
* **Moves.** A file moves with `git mv` before it is edited, in its own
  commit where practical, so `git log --follow` and blame survive.
* **New checks.** Every new gate is seen to fail on a scratch copy first, and
  the failure is recorded in the execution record.
* **Stop conditions.** Stop, present evidence and resolutions, and record the
  chosen one here and in the MADR, when any of these happens:
  * a G-wire difference that no record explains;
  * a coverage floor broken (*amended 2026-10-01:* `llmprovider`'s is measured over
    `./llmprovider/...` until S7b; see that date's deviation);
  * a test assertion that would have to change its meaning;
  * a dependency outside the allowed set.

## Implementation Steps

### Phase S0: accept (docs only)

1. The owner decides the MADR. On acceptance:
   * set the MADR `accepted` and this PLAN `in-progress`;
   * set the 0002 amendment's items to accepted;
   * commit this pair, the REPORT, the 0002 amendments and
     `docs/README.md`.

### Phase S1: standards and the parity map, first

1. **`docs/guides/api-standards.md`.** Write it from D2–D13, one numbered
   rule per decision point, each citing its D#. It is written first so the
   later phases are built to it, and it is updated only by amending the
   MADR.
2. **`docs/guides/migrating-from-mcplib.md`.** Generate its table skeleton
   from `go doc -all` of `mcplib` `v1.6.0` `llmprovider` and `wizard`: one
   row per exported identifier. The "SDK equivalent" column is filled in as
   the phases land.
3. **`scripts/check_parity_map.py` (G-parity).** Stdlib Python. It reads the
   baseline identifier list (committed as
   `docs/guides/migrating-from-mcplib.ids`, generated at `mcplib` `v1.6.0`)
   and fails when any identifier has no row, or has a row whose equivalent
   is empty. The empty-cell check stays off until S11.
   * **First-fail:** delete one row in a scratch copy.
   * Add the Python ignore rules to `.gitignore` in the same change.
4. **`make parity-check`** runs it.

### Phase S2: wire goldens at `P7` (G-wire)

1. **The recorder.** Add `internal/wiretest`: an `httptest.Server` that
   records each request as normalised JSON:
   * method and path;
   * headers, except `User-Agent` version text, session and task ids, and
     `Content-Length`;
   * the body, with keys sorted.

   It replays a canned response from `testdata`.
2. **The scenarios.** For each provider × {text, forced tool, thinking,
   thinking tool, items, continuation where supported, model listing}, add a
   test driving the **current** API. It writes
   `llmprovider/testdata/wire/<provider>/<scenario>.json` under `-update`
   and compares against it otherwise.
3. Commit the goldens generated at `P7`.
4. **First-fail:** in a scratch copy, change one request field in one
   provider (the ChatGPT `store` flag). The golden test fails and names the
   field.
5. **Carrying the goldens forward.** These tests are rewritten against the
   new API in S7. The goldens themselves are never regenerated in this plan,
   except for a difference listed in the execution record with the record
   that causes it.

### Phase S3: transport defaults (0016 T1), in place

*Amended 2026-09-30 (see "Deviation 2026-09-30" in the execution record).
The extraction that was this phase created an import cycle, and moves to
Phase S7b.*

1. Land [0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md) T1 inside `llmprovider`:
   * the default client in `options.go` sets
     `Proxy: http.ProxyFromEnvironment`, and keeps today's timeouts and
     connection limits;
   * each provider builds one client at construction, and passes it to its
     listing and to its OAuth session when that has none.

   The client moves to `llmprovider/internal/transport` in S7b.
2. Gate, including G-wire unchanged.

The steps as first written, superseded by S7b:

1. ~~`git mv` the four wire-format files and their tests into
   `internal/wire/{responses,chatcompletions,messages,interactions}`:
   `chatcompletions.go`, the Responses encode/decode, the Messages encoding,
   and `gemini_interactions.go`.~~
2. ~~`git mv` the transport files and their tests into `internal/transport`:
   `http_helpers.go`, `identification.go`, the classification part of
   `api_error.go`, and `probe.go`.~~
3. ~~Export within `internal` only what the providers need. `llmprovider`
   keeps calling them.~~
4. ~~Gate, including G-wire unchanged.~~

### Phase S4: `auth` work, in place

*Amended 2026-09-30 (see "Deviation 2026-09-30").* The providers use the
session types, so the move into `llmprovider/auth` would create the same
cycle as S3. It moves to S7b. What lands here is the work that needs no move.

1. ~~`git mv` these into `llmprovider/auth`:
   `oauth_*.go`; `tokenstore*.go`; `vendor_session.go`; the session part of
   `token.go`; their tests.~~ *Moved to S7b.*
2. `Token`, `TokenType` and `TokenSource` stay in `llmprovider`, as D2
   says.
3. Fix the stale "Phase 2 / Phase 3" doc comments (0015-REPORT F7).
4. Land 0016-PLAN T2 in place.
5. Gate.

### Phase S5: `catalog` work, in place

*Amended 2026-09-30 (see "Deviation 2026-09-30").* Every provider lists
models through this code, so the move into `llmprovider/catalog` would create
the same cycle as S3. It moves to S7b. What lands here is the work that needs
no move.

1. ~~`git mv` these into `llmprovider/catalog`:
   `models_catalog.go`, `model_ranking.go`, `model_matcher.go`,
   `model_metadata.go`, `model_profile.go`; the curation half of
   `discovery.go`; their tests.~~ *Moved to S7b.*
2. Replace the exported mutable variables with functions returning copies
   (D9):
   * the seven `Static*` catalogs;
   * `ProviderEnvVars`.
3. Collapse the seven `Rank*Model` functions behind one function taking the
   provider id. It takes `ProviderID` once S6 defines that type, and becomes
   `catalog.Rank(ProviderID, model)` in S7b.
4. Land 0016-PLAN T3 step 2 in place: no billed probe by default. The G-wire
   listing goldens change here. Each difference is listed in the execution
   record, with 0016-MADR D9 as its record.
5. Gate.

### Phase S6: the contract, and `llmtest`

*Accepted 2026-09-30 (the MADR's amendment of that date):* S6 also adds
`ErrContextOverflow` beneath `ErrInvalidRequest`, with its vendor
message table and a red-first test for each entry.


1. **In `llmprovider`, the D3–D8 contract:**
   * `Provider`, `Request`, `Response`, `Usage`, `Capabilities` and
     `Support`;
   * `Event` and `Stream` with its fallback;
   * `Streamer`;
   * the typed identifiers;
   * `Option` and the common options, with the mechanism that lets provider
     packages define typed options and makes `New` reject foreign ones;
   * `APIError` with `Kind`, the sentinels, and `Retryable()`;
   * `WithRetry` and `RetryPolicy`;
   * `Registry`, `Descriptor` and `Factory`.

   The old types stay alongside until S8. *(2026-09-30: where a new name
   is already taken, the deviation of that date applies.)*
2. **`llmtest`:**
   * `Run(t, Harness)` with the D11 checks;
   * `Fake`, a scriptable provider.
3. **First-fail.** Run `llmtest` against a deliberately broken fake in a
   scratch copy, once for each of these defects:
   * it ignores cancellation;
   * it sends a request with an unsupported capability;
   * it mis-classifies a 429;
   * it mutates shared state without a lock, which `-race` catches.

   Each must fail with a message naming the rule.
4. **Tests for the contract itself:**
   * the `Stream` fallback;
   * the retry policy honouring `RetryAfter` and the kinds;
   * `Registry` refusing a duplicate;
   * `New` refusing a foreign option.
5. Gate.

### Phase S7: providers onto the contract, one per commit

Order: `openai` (with the ChatGPT backend), `claude`, `gemini`, `grok`,
`opencode` (Zen and Go), `kilo`, `huggingface`, `ollama`.

*Amended 2026-09-30:* until S7b, the shared wire, transport, `auth` and
`catalog` code stays in `llmprovider`. A moved provider package reaches it
through identifiers that `llmprovider` exports for the duration of S7 only.
Each commit's execution record lists the temporary exports it adds, and S7b
removes them all.

*Amended 2026-09-30 (0015-MADR, amendment "listing, and options scoped per
provider"):* before the openai commit, one commit adds `llmprovider.ModelLister`,
makes `WithModelProbes` and `ModelProbesFromEnv` common options, and gives
`Settings` a `ModelProbes()` accessor. Every moved provider that lists
implements `ModelLister`.

*Proposed 2026-09-30 ([0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md) D1):* accepted by the
owner. `together` joins the order after `ollama`. It lands in `llmprovider` first,
under 0017-PLAN U1.

*Amended 2026-09-30 (decision 1 of that date's S7 prerequisites):* S7's
first commit removes `NewProvider` and `NewProviderWithSource`, and creates
`llmprovider/providers` with `Default()` and `New(id, opts...)`, empty.
Every later S7 commit registers its provider there. The tests that build a
provider by name move to `providers.New` with their provider.

For each provider:

1. `git mv` its files and tests into `llmprovider/providers/<id>`.
   *(Amended 2026-09-30:)* register it in `providers.Default()`.
2. Implement `New`, `ID`, `Capabilities`, `Generate`, and `ListModels`
   where it has a listing. Move its provider-specific options into the
   package.
3. Document every `BestEffort` degradation in the package doc.
4. Port its tests to the new API without changing any assertion's meaning.
   Rewrite its G-wire scenarios against the new API and compare them to the
   `P7` goldens.
5. Add its `llmtest` harness.
6. Remove its old type and methods from `llmprovider`.
7. Gate. Record G-wire and `llmtest` output, and coverage against the `P7`
   baseline.

### Phase S7b: extract `internal/wire`, `internal/transport`, `auth` and `catalog`

*Amended 2026-10-01 (0015-MADR, amendment "S7b's import graph", proposed):* steps 1–7 below, as written, do not
compile: `llmprovider`'s remaining code uses the transport helpers and the
session types. They are superseded by "S7b as amended", after step 7. S7b is
two commits, wire then transport; `auth` moves in the new Phase S8c, after
S8b. The original steps are kept for the record.

*Added 2026-09-30, from the original S3, S4 step 1 and S5 step 1.* It runs
once no provider is left in `llmprovider`. *(Amended 2026-09-30, decision 2
of that date's S7 prerequisites: step 4, the `catalog` extraction, moves to
Phase S8b, after S8. S7b extracts three packages: wire, transport, `auth`.)* The four packages can then import
`llmprovider`, as D2 allows, without a cycle. Each package moves in its own
commit, in this order: wire, transport, `auth`, `catalog`.

1. `git mv` the wire-format code and its tests into
   `llmprovider/internal/wire/`, one package per format: Responses (the
   encode and decode in `http_helpers.go` and `item_convert.go`), Chat
   Completions (`chatcompletions.go`), Messages, Interactions
   (`gemini_interactions.go`), and OpenCode's `generateContent`. That is the
   five formats of D2.
2. `git mv` the transport code and its tests into
   `llmprovider/internal/transport`:
   * the transport half of `http_helpers.go`;
   * `identification.go`;
   * the classification part of `api_error.go`;
   * `probe.go`;
   * S3's default client.
3. `git mv` into `llmprovider/auth`:
   * `oauth_*.go`;
   * `tokenstore*.go`;
   * `vendor_session.go`;
   * the session part of `token.go`;
   * their tests.

   `Token`, `TokenType` and `TokenSource` stay in `llmprovider`. The S4
   doc text that describes `auth` becomes its package doc.
4. ~~`git mv` into `llmprovider/catalog`:~~ *Moved to Phase S8b.*
   * `models_catalog.go`, `model_ranking.go`, `model_matcher.go`,
     `model_metadata.go`, `model_profile.go`;
   * the curation half of `discovery.go`;
   * their tests.

   S5's ranking function becomes `catalog.Rank(ProviderID, model)`.
5. The two methods the moving code declares on types that stay become
   functions: `Response.appendOutput` and `GeminiProvider.interactionsBody`.
6. Point the provider packages at the four packages instead of S7's
   temporary exports, and remove every temporary export.
   **Check:** `go doc -all ./llmprovider` names none of the identifiers the S7
   records list.
7. Gate after each of the four commits, including G-wire unchanged.

#### S7b as amended (2026-10-01, 0015-MADR, amendment "S7b's import graph", proposed)

**Commit 1, wire.** Each moved file goes with plain `mv` where it moves
whole; a file split between packages is recorded with where each part went.

1. **`internal/wire`** (package `wire`) holds what the formats share: the
   JSON keys they use, `ToolArguments` and `SystemPrompt`
   (`item_convert.go`).
2. **One package per shared format**, each importing `internal/wire` and
   `llmprovider`. Names lose the format prefix the package now carries, so
   that none stutters under `revive`:

   | Package | From | Exported |
   |---|---|---|
   | `internal/wire/responses` | the Responses half of `http_helpers.go` | `Input` (`ItemsToInput`), `Decode` (`DecodeResponsesAPIOutput`), `ReadStream` (`ReadResponsesStream`) |
   | `internal/wire/chatcompletions` | `chatcompletions.go` | `Opts`, `Body`, `Decode` |
   | `internal/wire/messages` | `messages_wire.go`, and the Anthropic half of `thinking_wire.go` | `FromItems`, `Decode`, `AddThinking` |
   | `internal/wire/generatecontent` | `generatecontent_wire.go`, and the Gemini half of `thinking_wire.go` | `SystemInstruction`, `Contents`, `Decode`, `ThinkingConfig` |

   `Response.appendOutput` becomes a function in `responses` (original
   step 5). `GeminiProvider.interactionsBody` no longer exists: the
   Interactions wire stays in `providers/gemini`.
3. **`llmprovider` exports `ClassifyStreamFailure`**, which was
   `streamFailure`, for good. `ReadStream` calls it.
4. **Tests move with the code they test.** A test of more than one format,
   such as `item_fidelity_test.go`, goes to `internal/wire` as an external
   test package. Each test's destination is recorded.
5. **The provider packages import the new packages**, and the wire's
   temporary exports are removed.
6. **Coverage.** Each new package holds D13's 80 % from its own tests. A
   package below it stops the commit, as a deviation.
7. **Gate**, with G-wire unchanged and no `-update`.

**Commit 2, transport.**

1. **`internal/transport`** imports only the standard library and
   `internal/redact`. It holds:
   * `DefaultClient` (`defaultHTTPClient`);
   * `Identity`, with `UserAgent` and `SetUserAgent` (`clientIdentity`), and
     the build versions it reads;
   * `CloseBody` (`closeResponseBody`);
   * `ParseRetryAfter` (`parseRetryAfter`);
   * `ProbeGenerateHealth`, with its limit passed in (`probe.go`).
2. **`llmprovider` imports it.** `identityOf` stays in `llmprovider`, to
   build an `Identity` from `ProviderConfig`.
3. **R16's helper** becomes `Token.Apply(req, header, scheme)`, replacing
   `SetTokenHeader`. `ClassifyHTTPError` loses its temporary label.
4. **Tests move with the code**, as in commit 1.
5. **The provider packages** use `transport.ProbeGenerateHealth` and
   `Token.Apply`, and the transport temporary exports are removed.
6. **Check.** `go doc -all ./llmprovider` names none of the wire or
   transport temporary exports the S7 records list. The ones left are
   listed in the record: the session exports, for S8c, and the catalog's,
   for S8b.
7. **Coverage** as in commit 1, for `internal/transport`; gate.

`llmprovider` stays measured with `-coverpkg=./llmprovider` until S8c.

### Phase S8: registry, `wizard`, and removal of the old API

1. **`providers.Default()` and `providers.New`.** *(Amended 2026-09-30: S7
   creates them and fills `Default()`; this step keeps the rest.)*
   Descriptors move to their provider packages. `TestDescriptors_CoverEveryRegisteredProvider` becomes
   a test that every provider package is in `Default()`.
2. **`wizard`** takes `Options.Registry`, uses typed ids, and keeps the
   0002-MADR ~~§4 and~~ §7 behaviour. *(2026-09-29: §4 is superseded by
   0002-MADR's sixth amendment; `wizard` has no orchestration option.)*
3. **Remove what remains of the old API:**
   * `NewProvider*`;
   * the `Generate*WithRetry` functions;
   * the eight generation interfaces and `Continuer`;
   * `ProviderConfig` and `ApplyOptions`;
   * `RateLimitError` and `IncompleteError`;
   * *(added 2026-09-30, S6's deviation)* `LegacyProvider`, the
     `ProviderOption` alias, and `APIError`'s `Type` and `Terminal`.
4. *(Added 2026-09-30, S6's deviation.)* Retype the provider-id constants as
   `ProviderID` and `MessageItem.Role` as `Role`, and give the sentinels that
   still read `llm:` the `llmprovider:` prefix (R27).
5. Gate, with G-parity's empty-cell check still off.

#### S8 as amended (2026-10-01, 0015-MADR, amendment "`catalog` before the old API's removal", proposed)

Steps 1–5 above are carried out as six commits, each gated and staged for
the owner. `catalog`, which S8b was to extract after S8, is commit 2.

1. **Descriptors and the wizard menu.**
   * Each provider package declares its own `Descriptor`, and
     `providers.Default()` registers it.
   * `ProviderDescriptor`, `Descriptors()` and `DescriptorFor` are removed.
   * `wizard.Options` gains `Registry`; a nil one is `providers.Default()`.
     The menu comes from `Registry.Descriptors()`.
   * The coverage test becomes "every descriptor is in `Default()`", which
     S7 already holds.
2. **`catalog`** (S8b's steps 1–3, moved here). With plain `mv` into
   `llmprovider/catalog`, and their tests:
   * `models_catalog.go`, `model_ranking.go`, `model_matcher.go`,
     `model_metadata.go`, `model_profile.go`;
   * the listing, `discovery.go`;
   * the gateway URL files `huggingface_gateway.go`, `ollama_endpoint.go`,
     `together_endpoint.go` and `opencode_gateway.go`.

   `kilo_endpoints.go` goes to `llmprovider/internal/kiloendpoint`. The
   listing resolves its options with `ResolveOptions`. `catalog.WithProfile`
   and `catalog.WithKiloOrganization` replace `WithModelProfile` and
   `WithKiloOrganization`, which are removed. The provider packages and
   `wizard` point at `catalog`. Names lose the prefix the package gives
   (proposed):

   | Was | Is |
   |---|---|
   | `ModelCatalog` | `catalog.Catalog` |
   | `ListModelCatalog`, `ListModelCatalogWithSource`, `ListAvailableModels`, `ListAvailableModelsWithSource` | `catalog.List(ctx, id, src, opts...)`; a key is `NewStaticToken(key)`, and the recommendation is `Catalog.Recommended` |
   | `StaticModels`, `RankModel`, `SearchModels` | `catalog.Static`, `catalog.Rank`, `catalog.Search` |
   | `ModelMatch`, `ModelLabel` | `catalog.Match`, `catalog.Label` |
   | `ModelProfile`, `ProfileUtility`, `ProfileCapable` | `catalog.Profile`, `catalog.ProfileUtility`, `catalog.ProfileCapable` |
   | `MaxListedModels` | `catalog.MaxListed` |
   | `ModelMetadata`, `LookupModelMetadata` | `catalog.Metadata`, `catalog.LookupMetadata` |
   | `KiloModelCapabilities`, `ValidateOllamaURL` | `catalog.KiloModelCapabilities`, `catalog.ValidateOllamaURL` |
   | `KiloGatewayFor` | `kiloendpoint` (internal) |

   `catalog` holds 80 % from its own tests. G-wire is unchanged.
3. **Errors, D7 and R27.**
   * A 429 is an `*APIError` of kind `ErrRateLimited`, with its
     `RetryAfter`.
   * A truncated answer is an `*APIError` of kind `ErrIncomplete`, with its
     `Reason`.
   * `RateLimitError` and `IncompleteError` are removed.
   * `Terminal` becomes unexported behind `Retryable()`, and `Type` is
     removed (`Code` carries it).
   * Every sentinel reads `llmprovider:`, `ErrInvalidProvider` included.

   The tests that pin "a 429 stays a `*RateLimitError`" are rewritten to
   D7's rule, each named in the record.
4. **The old generation API**:
   * `LegacyProvider`, the eight generation interfaces, `Continuer` and
     `ModelDiscoverer`;
   * the three `Generate*WithRetry` functions and their retry loop;
   * the `ProviderOption` alias;
   * the old-API-only options left, which nothing reads;
   * `ProviderConfig` and `ApplyOptions`, made unexported.
5. **Typed ids.** The provider-id constants are `ProviderID`, and so is
   every parameter or field that holds one. A label that may carry a route,
   such as `"opencode-go/messages"`, stays a `string`. `MessageItem.Role` is
   `Role`.
6. **0016-PLAN T4 in `wizard`.**
   * D11: no token in `Result` when the session is saved to
     `Options.TokenStore`, and a stored session is reloaded from it.
   * D11: logout, which revokes and then deletes.
   * D5: `Result` redacts itself.

### Phase S8b: extract `catalog`

*Amended 2026-10-01 (0015-MADR, amendment "`catalog` before the old API's removal", proposed):* steps 1–3 are done in S8, commit 2.
Only step 4, `For(id, opts...)`, remains here, after S8.

*Added 2026-09-30, decision 2 of that date's S7 prerequisites.* It runs
after S8, once `ProviderConfig` and `WithModelProfile` are gone and nothing
in `llmprovider` refers to `ModelProfile`.

1. S7b's original step 4: `git mv` into `llmprovider/catalog` the files it
   lists, with their tests. S5's ranking function becomes
   `catalog.Rank(ProviderID, model)`, and `ModelProfile` becomes the
   catalog's.
2. Point the provider packages and `wizard` at `catalog`. `wizard`'s
   `ModelProfile` references change a second time, having changed in S8.
3. Gate, including G-wire unchanged.
4. *(Added 2026-09-30, 0015-MADR amendment "listing, and options scoped
   per provider", D5 steps 2–4.)* `For(id, opts...)`:
   * its options apply only when building `id`, after the baseline;
   * it is skipped for other ids;
   * a provider-specific option inside it is legal in a shared list;
   * it refuses an old-API-only option, and a nested `For` of another id.

     *Annotated 2026-10-01:* S8, commit 4 removed the last old-API-only
     option, so only the nested `For` is left to refuse.

   Red-first tests, and breaks for each rule. The standards guide's R18
   gains the `For` exception. `wizard` passes one option list through the
   ~~`Registry`~~ `catalog.List`.

   *Deviation, 2026-10-01:* `wizard` builds no provider: it takes
   descriptors from the `Registry` and lists with `catalog.List`
   (`wizard/configure.go:322-332`). The owner chose "overlay in the listing
   list": the Kilo organization goes in `For(kilo, …)`, so the list is the
   same whichever provider was chosen.
5. Gate.

### Phase S8c: extract `auth`

*Added 2026-10-01 (0015-MADR, amendment "S7b's import graph", proposed).* It runs after S8b, once
`discovery.go` has left `llmprovider`.

1. **The ChatGPT listing** (`listChatGPTModels`) and the ChatGPT session
   helpers move to `providers/openai`. These are `IsChatGPTSession`,
   `ChatGPTSessionAccountID`, `ChatGPTSessionFedRAMP`, `ExpireSession` and
   the four header constants. The catalog's listing stops special-casing a
   ChatGPT session; `openai`'s `ListModels` lists one itself.

   *Deviation, 2026-10-01:* three facts the step did not cover were put to
   the owner before any code, and decided as 0015-MADR amendment "the
   ChatGPT helpers leave `llmprovider`":
   * the session types gain `Account()`;
   * `OAuthSession` becomes an `InvalidatingSource`, and `ExpireSession`
     goes;
   * `wizard` lists a ChatGPT session through `openai`'s `ListModels`.
2. **S7b's original step 3:** `git mv` into `llmprovider/auth` the files it
   lists, with their tests. `auth` imports `llmprovider` and
   `internal/transport`.

   *Deviation, 2026-10-01:* a trial move in a scratch copy found three
   points the step did not settle. They were decided as 0015-MADR amendment
   "what `auth` holds":
   * `StaticToken` and `CommandToken` stay in `llmprovider`;
   * `OAuthSession.UseHTTPClient` replaces `ShareHTTPClient`;
   * the base URLs go to `openai` and `grok`.
3. **The provider packages and `wizard` import `auth`**, and the remaining
   session temporary exports are removed.
   **Check:** `go doc -all ./llmprovider` names none of the identifiers the
   S7 records list.
4. **Coverage.** The `-coverpkg` measurement of `llmprovider` ends.
   `llmprovider` is measured by its own tests against its `P7` 89.2 %, and
   `auth` holds 80 %. A shortfall stops the phase, as a deviation.
5. Gate, including G-wire unchanged.

### Phase S9: usage

*Facts from the survey, 2026-09-30 ([0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md), P7 and K2).*

* **Chat Completions cached tokens** come as
  `prompt_tokens_details.cached_tokens`, `prompt_cache_hit_tokens` or
  `cached_tokens`. Reasoning tokens come as
  `completion_tokens_details.reasoning_tokens`.
* **Together** reports `prompt_tokens`, `completion_tokens` and
  `total_tokens`.
* **Kilo's own client always sends `usage: {"include": true}`.**
  * S9 first measures, live, whether Kilo reports usage without it.
  * If it does not, sending it is a request change. That conflicts with
    step 2's "Request goldens do not change", and is put to the owner as a
    deviation before any golden changes.


1. Decode `Usage` in each wire format that reports it. Where one does not,
   write a test that asserts zero.
2. Extend the response fixtures. Request goldens do not change.
3. Gate.

### Phase S10: ambient state

1. Remove every `os.Getenv` from library code. Add the opt-in helpers
   (`catalog.OptionsFromEnv`, `auth.GrokFlowFromEnv`). Tests set options,
   not process environment.
2. Replace the three `slog` package-level calls with the injected logger.
3. **A test that fails on a new ambient read.** Add
   `internal/ambientcheck`, which walks the non-test sources and fails on
   `os.Getenv`, `os.LookupEnv` or a package-level `slog.` call.
   *Amended 2026-09-29 ([0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md) D8):* it allows exactly one
   standard-library read, `http.ProxyFromEnvironment` on the default
   transport.
   * **First-fail:** a planted `os.Getenv` in a scratch copy.
4. Gate.

### Phase S11: documentation, and the third-party proof

1. **`api-standards.md`.** Finalise it against the built tree.
2. **`adding-a-provider.md`.**
3. **`migrating-from-mcplib.md`.** Complete every row, and turn on
   G-parity's empty-cell check.
4. **`docs/architecture.md`.** The package graph and data flow, as built.
5. **AGENTS.md and `README.md`.** Point API work at the standards guide.
6. **Third-party proof.** In `SCRATCH/thirdparty`, a separate module that
   imports this one through a `replace`, write a toy provider using only
   `adding-a-provider.md`. It must:
   * register in a `Registry`;
   * be offered by `wizard`, driven through `llmtest.Fake`-style scripted
     prompts;
   * pass `llmtest.Run`.

   Record every place where the guide was insufficient, and fix the guide.
7. Gate, with G-parity fully on.

### Phase S12: enforcement in CI

1. **`make dep-check`.** `go list -deps` per package fails when a package
   other than `wizard` leaves the standard library. It runs in CI.
   * **First-fail:** a scratch copy that imports `golang.org/x/term` from
     `catalog`.
2. **`make coverage-check`.** Fails when a package drops below its `P7`
   baseline or a new package is below 80 %. The baselines are committed as
   `scripts/coverage-floors.txt`.
   *Amended 2026-10-01:* if it runs before S7b has finished, it measures
   `llmprovider` with `-coverpkg=./llmprovider` over `./llmprovider/...`.
   * **First-fail:** a scratch copy with one test removed.
3. **`make api-check`.** `go run golang.org/x/exp/cmd/apidiff@<pinned>`
   against the latest `v1.*` release tag. Before the first tag it reports
   and does not fail. From `v1.0.0` on, it fails on an incompatible change
   outside `llmprovider/x/`.
   * **First-fail:** a scratch tag and a removed exported function.
4. **CI** runs `parity-check`, `dep-check`, `coverage-check` and
   `api-check`.
5. Gate.

### Phase S13: close-out

1. Fill the execution record.
2. Set this PLAN `complete` and mark the MADR's Confirmation items met.
3. **Hand over to 0002-PLAN Phase 8.** Record the handover in 0002's
   execution record. A `v1.0.0-rc.1` tag for consumer trials needs the
   owner's request.

## Verification

| # | Criterion | Check | Phase |
|---|---|---|---|
| S-A1 | Package layout and allowed imports as D2 | `make dep-check`; package list | S4–S8 (with S7b), S12 |
| S-A2 | Every provider implements the contract and passes `llmtest` | `llmtest.Run` per provider | S7 |
| S-A3 | Requests on the wire unchanged from `P7`, except recorded differences | G-wire | S2–S10 |
| S-A4 | Every `mcplib` `v1.6.0` identifier mapped | G-parity | S1, S11 |
| S-A5 | Ported tests keep their meaning | review of each S7 commit's test diff, recorded | S7 |
| S-A6 | No ambient state | `internal/ambientcheck`; no exported package-level `var` except sentinels | S5, S10 |
| S-A7 | Coverage floors | `make coverage-check` | S3–S12 |
| S-A8 | Standards written and enforced | the four guides exist; CI runs the four checks | S1, S11, S12 |
| S-A9 | A third party can add a provider from the guide alone | the S11 scratch module | S11 |
| S-A10 | Streaming and usage available for every provider | the `Stream` fallback test; usage fixtures | S6, S9 |

## Rollout and Rollback

* **No consumer imports the module before `v1.0.0-rc.1`.** Rollback is
  `git revert` of the phase commits, and nothing outside this repository is
  affected.
* **Release candidates** are tagged only on the owner's request. A candidate
  found wanting is followed by `rc.N+1`. Tags are never moved.
* **Abandoning the plan** means reverting to `P7`. 0002-PLAN then resumes at
  Phase 8 with the `mcplib` API (option A of the MADR). That is a new
  decision, recorded as an amendment.

## Execution Record

### Phase S0: accept (2026-09-29)

* **Decision.** The owner answered "accept all" to 0002-MADR's second
  amendment together with this MADR. The MADR is `accepted`, this PLAN
  `in-progress`, and the 0002 second amendment `accepted`, as modified by
  0002-MADR's sixth amendment (no orchestration in `wizard`;
  `prepare-commit-msg` is the only consumer for now).
* **Correction at acceptance.** D2 said "four wire formats". There are
  five: OpenCode's Google route uses Gemini `generateContent`
  (`llmprovider/opencode_route.go:50-51`). The row is corrected in place,
  with the old word struck.
* **Next.** S1 starts when 0002-PLAN Phase 7 is committed (§0).

### 0016 steps in these phases (2026-09-29)

[0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md) has no phases of its own; its steps land in these phases'
commits:

| 0015 phase | 0016 step |
|---|---|
| S3 (in `llmprovider`; moves to `internal/transport` in S7b, amended 2026-09-30; only what needs no `llmprovider` type, amended 2026-10-01) | T1: proxy and one client per provider (D8) |
| S4 (in place; moves to `auth` in ~~S7b~~ S8c, amended 2026-10-01) | T2: durable writes, rotation kept, redaction, device handle, OAuth checks with `id_token` signature verification (D3–D7) |
| S5 (in place; moves to `catalog` in S7b) | T3 step 2: no billed probe by default (D9) |
| S7 (providers) | T3 step 1: every provider takes a `TokenSource` (D2) |
| S8 (`wizard`) | T4: one refresh-token copy, logout, `Result` redaction (D11, D5) |
| S10 (ambient state) | T5: `ANTHROPIC_API_KEY` only (D12) |
| S11, S13 | T6: guides and close-out |

### Phase S1: standards and the parity map (2026-09-29)

* **Start.** `P7` is `cc81adf`, the 0002-PLAN Phase 7 commit. S1 began on
  `cbb0c7a` (0016-PLAN T0), which changed records only.
* **Step 1, `docs/guides/api-standards.md`.** 48 rules, R1–R48, grouped by
  D2–D13, each citing its decision. Three things beyond a transcription:
  * **Rules from 0016-MADR.** R15–R16 (D2, token sources), R30 (the D8 proxy
    exemption), R32 (D8, one client per provider) and R33–R34 (D5,
    self-redaction) cite 0016-MADR, because they are conventions every later
    phase is built to. The guide says it changes only by amending the decision
    a rule cites, whichever record that is.
  * **R8 names the convenience helpers** `GenerateText` and
    `GenerateToolCall`, as D3 delegates to this guide. S11 may rename them
    against the built tree, as step 1 of that phase allows.
  * **A "Checks" table** names the tool or test that holds each checkable
    rule and the phase that brings it. The rest are held by review.
  * The rules are dash items labelled `R<n>`, not one ordered list, because
    markdownlint MD029 rejects numbering that runs across headings.
* **Step 2, `docs/guides/migrating-from-mcplib.md`.** The baseline is
  generated from `mcplib` at `4e1f9a5` (`v1.6.0`) with
  `scripts/check_parity_map.py generate`. It has 409 identifiers: 162
  top-level and 247 members; 349 in `llmprovider` and 60 in `wizard`. It is
  committed as `docs/guides/migrating-from-mcplib.ids`. A sample of 25
  expected identifiers (for example `llmprovider.Provider.Generate`,
  `llmprovider.ModelCatalog.Err`, `wizard.Result.RefreshToken`) were all
  present, and no line fell outside `(llmprovider|wizard).Name[.Member]`.
  The guide has one row per identifier, plus the import paths and the
  behaviour 0002-MADR has already changed. Two rows are filled, both
  "removed" by 0002-MADR's sixth amendment: `wizard.ErrOrchestrated` and
  `wizard.Options.Orchestrated`.
* **Step 3, `scripts/check_parity_map.py` (G-parity).** Stdlib Python.
  Two choices the step did not spell out:
  * it also fails on a row whose identifier is not in the baseline, so a
    typo in the guide cannot hide a missing row;
  * it reads rows only under `## Identifier map`. Its first run read the
    import-path table as two identifier rows and failed, which this
    corrected before any commit.

  The empty-cell check is `--require-equivalents`, off until S11.
  `.gitignore` gains `__pycache__/` and `*.py[cod]`.
* **First-fail,** on scratch copies of the script and `docs/guides/`, never
  the tree:

  | Case | Exit | First line |
  |---|---|---|
  | clean copy | 0 | `G-parity: 409 identifiers, 409 rows, 2 with an SDK equivalent, 0 problem(s)` |
  | the `llmprovider.Provider.Generate` row deleted | 1 | `G-parity: no row: llmprovider.Provider.Generate` |
  | that row moved above `## Identifier map` | 1 | `G-parity: no row: llmprovider.Provider.Generate` |
  | a row for `llmprovider.NotThere` added | 1 | `G-parity: row for an identifier not in the baseline: llmprovider.NotThere` |
  | `--require-equivalents` on today's guide | 1 | `G-parity: empty SDK equivalent: llmprovider.APIError` (407 problems) |
  | an unknown argument | 2 | the usage text |
* **Step 4, `make parity-check`,** with `##` help. It prints
  `G-parity: 409 identifiers, 409 rows, 2 with an SDK equivalent, 0 problem(s)`.
* **Index and architecture.** `docs/README.md` gains two "I want to…" rows
  and a pointer to `guides/`. `docs/architecture.md` lists the guides, the
  script and the target, and "What is not here" now names the code that does
  not yet meet the standards, and `adding-a-provider.md` (S11).
* **Gate,** every step exit 0:
  * `make pre-add-check`;
  * `go vet ./...` with `GOOS` darwin, linux and windows (`CGO_ENABLED=0`:
    with cgo on, the linux cross-vet fails in `runtime/cgo` on the macOS C
    compiler, before any package here);
  * `go vet -tags live_gateways ./...`;
  * `go test -race -count=1 -cover ./...`: `internal/redact` 100.0 %,
    `llmprovider` 89.2 %, `wizard` 83.4 %, unchanged from `P7`;
  * `go mod tidy -diff`, empty;
  * `make lint`, `0 issues.`;
  * `make parity-check`;
  * `markdownlint-cli2` on the two guides, `docs/README.md` and
    `docs/architecture.md`: 0 issues;
  * G-links over `docs/**` and `README.md`: 0 problems;
  * the disclosure guard's deny list over every changed file: 0 hits.
* **Not done.** `docs/reports/0011-REPORT-provider-source-compatibility-audit.md`
  has markdownlint findings (list style, line length) that predate this phase.
  CI does not run markdownlint, and the file is outside S1, so it is left as
  it is.

### Phase S2: wire goldens at `P7` (2026-09-29)

* **Baseline.** `git diff cc81adf HEAD` shows no change to any `.go` file,
  `go.mod` or `go.sum`. Only records and guides changed since `P7`, so the
  goldens recorded on this tree are `P7`'s.
* **Step 1, `internal/wiretest`.** An `httptest` server that records each
  request as method, path, query, headers and body, and serves a canned
  `Reply` chosen by a function of the request. `Compare` and `Check` handle
  the golden files, and `Diff` names each differing field by its path. The
  package's own tests cover it to 95.2 %.
  * **Normalisation.** `Content-Length` is dropped. The body is decoded JSON
    with keys sorted and numbers exact. `User-Agent` keeps its structure with
    placeholders, `app/<version> (<os>; <arch>) go-llmprovider-sdk/<version>`.
    The step named only the version text, but the platform had to go too:
    CI runs the same goldens on Linux, macOS and Windows.
  * **Session ids are pinned, not stripped.** Every scenario passes
    `WithSessionID("wire-session")`, so the goldens show every place the
    session is sent: `prompt_cache_key`, `session-id`, `x-opencode-session`
    and Kilo's `X-Kilocode-Taskid`. Nothing else on the wire was random
    (a scan of the request-building code for `rand.`, `time.Now` and
    header writes).
  * **Order.** Requests are sorted by their encoding, because the listing
    probes run concurrently.
  * **Line endings.** `Compare` accepts a CRLF golden, which a Windows
    checkout can produce. First-fail on a scratch copy without that line:
    `CRLF golden: [(root): same value, different text; the golden file was
    edited by hand]`.
  * **Limit.** Go's server canonicalises header names
    (`ChatGPT-Account-Id` is recorded as `Chatgpt-Account-Id`), so a change
    in only a header name's case is not caught.
* **Step 2, the scenarios.** `TestWireGoldens` (`llmprovider/wire_golden_test.go`)
  runs 15 cases:
  * `openai` (API key) and `chatgpt` (an `OAuthSession` on the backend's event
    stream);
  * `claude`, `gemini` and `grok`;
  * OpenCode Zen's four routes, and Go's three: responses, messages, google
    and chat;
  * `kilo`, `huggingface` and `ollama`.

  Each case runs text, forced tool, thinking, thinking tool, items (a
  conversation using every `Item` type and a replayed signature),
  continuation and listing. That is 94 files, because only `openai`,
  `chatgpt`, `gemini` and `grok` have `Continue`. Two choices:
  * **Continuation.** Gemini's is recorded with `WithStore(true)`, because
    without it `Continue` refuses before the network. ChatGPT's golden
    records its refusal before the network, and no request.
  * **Results.** Each golden also holds the decoded result, with `Response`
    items tagged by type, for D12's "response fixtures must decode to
    equivalent `Output`" in S7.
* **What the goldens show at `P7`,** worth knowing before S5:
  * the listing health-probes each model on `openai`, `claude`, `gemini`,
    `grok` and `ollama`, one billed request per model;
  * `kilo`, `huggingface`, OpenCode and `chatgpt` do not probe.

  [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)
  D9 turns the probes off by default. That is a recorded difference for S5,
  where it lands (T3 step 2).
* **Step 3.** The goldens are committed with this phase. `go test -count=3`
  and a `-race` run reproduce them byte for byte. None contains the test
  server's address, a platform name or a home path.
* **Step 4, first-fail,** on a scratch copy of the tree:
  `body["store"] = false` in `llmprovider/openai.go` (the ChatGPT branch) was
  changed to `true`. The run exited 1. Exactly the five ChatGPT generation
  scenarios failed, each with
  `requests[0].body.store: want false, got true`. The listing and the
  continuation refusal send no such body and passed. Re-run after the lint
  fixes, with the same result.
* **Lint.** The first `make lint` found seven issues in the new code:
  * two unchecked errors;
  * an `==` comparison on an error;
  * one gofmt alignment;
  * three gosec findings: golden files and directories are now written
    `0600` and `0750`, and read through `filepath.Clean`.

  All were fixed at the cause, without suppressions.
* **Gate,** every step exit 0:
  * `make pre-add-check`;
  * `go vet` for darwin, linux and windows (`CGO_ENABLED=0`), and with
    `-tags live_gateways`;
  * `go test -race -count=1 -cover ./...`: `internal/redact` 100.0 %,
    `internal/wiretest` 95.2 %, `llmprovider` 90.2 % (89.2 % at `P7`; the
    scenarios reach more code), `wizard` 83.4 %;
  * `go mod tidy -diff`, empty;
  * `make lint`, `0 issues.`;
  * `make parity-check`;
  * `markdownlint-cli2` on the guides, `docs/README.md` and
    `docs/architecture.md`;
  * G-wire three times over;
  * G-links;
  * the disclosure guard's deny list over every changed and new file,
    including the 94 goldens: 0 hits.
* **Docs.** `docs/architecture.md` lists `internal/wiretest` and describes
  G-wire. `docs/guides/api-standards.md` marks R45 as checked now.

### Deviation 2026-09-30: the extractions of S3, S4 and S5 move after S7

* **Found.** Before any change, a read-only scan of the files S3 moves out of
  package `llmprovider` showed that each uses declarations that stay there:
  * `chatcompletions.go` uses the item types, `Response`, `Tool`,
    `IncompleteError` and 19 JSON key and role constants;
  * `gemini_interactions.go` uses the same, plus `ErrProviderUnavailable`,
    and declares the method `GeminiProvider.interactionsBody`;
  * `http_helpers.go` uses the item types, `ErrProviderUnavailable` and
    `IncompleteError`, and declares the method `Response.appendOutput`;
  * the classification in `api_error.go` uses the four error sentinels,
    `RateLimitError`, `retryAfterFrom` and provider ids;
  * `identification.go` uses `ProviderConfig` and `ProviderOption`;
  * `probe.go` uses `MaxListedModels`.

  The providers, which call all of these (for example
  `llmprovider/openai.go:215-222`), stay in `llmprovider` until S7. The moved
  packages would import `llmprovider`, which imports them: a cycle Go
  rejects. Go also allows no method declarations on another package's types.
  D2's end state has no cycle, because by then the providers have left
  `llmprovider`; only this PLAN's order was wrong.
* **Options put to the owner:**
  1. extract after S7, with temporary exports during S7;
  2. wire-local types in `internal/wire`, converted in `llmprovider`;
  3. an internal `core` package holding the contract, re-exported from
     `llmprovider` as type aliases.
* **Decision.** The owner chose option 1. It ends exactly at D2's layout with
  one declaration per identifier, and changes no decision in the MADR.
  Options 2 and 3 each needed a MADR amendment. Option 3 also puts the
  contract's definitions, and every method on them, behind aliases for the
  life of v1.
* **Changed:**
  * S3 now lands only 0016 T1, in place;
  * S7 gains the temporary-export rule;
  * a new Phase S7b does the extraction, with a check that the temporary
    exports are gone;
  * S-A1 and the 0016 table follow.

  The original S3 steps are struck through, not deleted.
* **Cost accepted.** For the length of S7, `llmprovider` exports shared
  helpers it will not keep. Their call sites are edited twice: to
  `llmprovider.X` in S7, then to `wire.X` or `transport.X` in S7b.
* **Extended the same day to S4 and S5.** While writing this amendment, a
  second read-only scan found the same cycle for `auth` and `catalog`.
  Moving them before S7 would make `llmprovider` import packages that
  import it:
  * `openai_chatgpt.go` uses `OAuthSession`, `VendorCLISession` and the
    default base URLs of `oauth_constants.go`;
  * `grok.go` uses `OAuthSession`, `DefaultGrokBaseURL` and
    `oauthAuthorizationHeader`;
  * `openai.go` and `opencode.go` use `oauthAuthorizationHeader`;
  * every provider except `ollama` uses `StaticModels`;
  * each provider calls its own listing function in `discovery.go`, or
    `ListAvailableModelsWithSource`;
  * `opencode.go` uses `loadModelMetadata`;
  * `kilo.go`, `huggingface.go`, `opencode.go` and `options.go`'s
    `ProviderConfig` use `ModelProfile`.

  The owner chose to extend option 1:
  * S4 and S5 keep their in-place work, including 0016 T2 and T3 step 2;
  * their `git mv` steps join S7b, which extracts four packages in four
    commits;
  * S5's `Rank` lands under a provider-id parameter first, and becomes
    `catalog.Rank(ProviderID, model)` in S7b.

  The phase numbers are unchanged.
* **Open, to be settled by amendment before S7 starts.** The same scans
  found two more ordering conflicts in S7 and S8. Neither affects S3–S6.
  *(Settled 2026-09-30: see "Amendment 2026-09-30: the S7 prerequisites
  settled".)*
  1. `NewProvider` and `NewProviderWithSource`
     (`llmprovider/provider.go:253`, `:277`) call every provider's
     constructor. S8 removes them. But the first S7 commit that moves a
     provider out would make `llmprovider` import that provider's package.
     `wizard` does not call either.
  2. `ProviderConfig.ModelProfile` and `WithModelProfile`
     (`llmprovider/options.go:67`, `:188`), used by
     `wizard/configure.go:68`, `:300`, keep `ModelProfile` inside
     `llmprovider` until S8 removes the old API. So S7b's `catalog` commit
     would cycle unless it follows that removal.

### Phase S3: transport defaults, in place (2026-09-30)

* **Amendment first.** Records-only commit `51ebc85` holds the deviation
  above and the reordered phases, after the owner chose option 1 and its
  extension to S4 and S5.
* **Step 1, [0016-PLAN-provider-auth-and-support-baseline.md](0016-PLAN-provider-auth-and-support-baseline.md) T1, in `llmprovider`:**
  * `defaultHTTPClient` (`options.go`) sets
    `Proxy: http.ProxyFromEnvironment`. Timeouts and connection limits are
    unchanged.
  * `shareHTTPClient` (`oauth_session.go`) gives an `OAuthSession` with no
    client the provider's. `NewOpenAIWithSource` and `newGrokWithSource`
    call it. A session that has a client keeps it.
  * Listing and probes already used each provider's client. Every
    `DiscoverModels` passes `p.client`. Only the refresh did not: it built a
    fresh `defaultHTTPClient()` (`oauth_session.go`, `refreshOAuthSessionOnce`).
  * Recorded in 0016-PLAN T1 with the tests and their red-first output.
* **Step 2, gate,** every step exit 0:
  * `make pre-add-check`;
  * `go vet` for darwin, linux and windows (`CGO_ENABLED=0`), and with
    `-tags live_gateways`;
  * `go test -race -count=1 -cover ./...`: `internal/redact` 100.0 %,
    `internal/wiretest` 95.2 %, `llmprovider` 90.2 %, `wizard` 83.4 %, the
    same as S2;
  * `go mod tidy -diff`, empty;
  * `make lint`, `0 issues.`;
  * `make parity-check`;
  * markdownlint on the guides, the index and architecture;
  * G-links;
  * the deny list over the changed files: 0 hits.

  **G-wire unchanged.** It passed three times over, and no file under
  `llmprovider/testdata/wire` changed. No test sets a proxy variable for
  the goldens, and loopback is never proxied.
* **Docs.** `docs/architecture.md` describes the provider's one client.

### Survey findings for this plan (2026-09-30, accepted)

[0017-REPORT-reference-client-auth-survey.md](../reports/0017-REPORT-reference-client-auth-survey.md) proposed three changes to this plan. The owner accepted
all three on 2026-09-30:
* **S6:** `ErrContextOverflow`, from the MADR's amendment of that date;
* **S7:** `together` in the order, from [0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md) D1;
* **S9:** the usage facts, including the Kilo `usage.include` measurement
  that may become a deviation.

No phase has changed yet.

### Phase S4: `auth` work, in place (2026-09-30)

* **Step 1** (the move) is struck, and moved to S7b.
* **Step 2** holds: `Token`, `TokenType` and `TokenSource` stay in
  `llmprovider`.
* **Step 3.** The stale "Phase 2 / Phase 3" doc comments on `OAuthSession`
  and `TokenStore` (0015-REPORT F7) are replaced with what the types are and
  do.
* **Step 4, 0016-PLAN T2, with the survey's A1–A3.** Five commits, each
  through the gate, each recorded in 0016-PLAN's execution record:
  * `0cd659d`: durable writes and the cross-process refresh lock;
  * `3a627d6`: rotation kept on a failed save, and a sibling's rotation
    adopted;
  * `b63f41a`: secrets redact themselves, fully (the owner's choice);
  * `fbe8bcf`: the device-login handle;
  * this commit: `id_token` verification, and fail-closed discovery.
* **Two existing tests pinned defects the accepted MADR fixes,** and were
  replaced by the tests D4 and D7 require: M4's save-failure test, and M6's
  discovery-fallback test. No other assertion changed meaning. The login
  tests' fixtures now sign `id_token`s.
* **Open items for the owner, from this phase:**
  * ~~slog's JSON handler shows the secret of a *struct holding* a `Token`
    (step 3's record);~~ *Closed 2026-09-30: the owner chose a redacting
    `MarshalJSON` (0016-MADR A4; 0016-PLAN, "T2 step 3, addition").*
  * the S7 ordering items of the 2026-09-30 deviation (`NewProvider`, and
    `ProviderConfig.ModelProfile`).
* **Gate,** every step exit 0, at each commit. At this commit:
  * `internal/redact` 100.0 %, `internal/wiretest` 95.2 %, `llmprovider`
    90.1 % (89.2 % at `P7`), `wizard` 83.4 %;
  * `make lint` `0 issues.`;
  * G-wire three times over, unchanged;
  * G-links;
  * the deny list: 0 hits.
* `docs/architecture.md`'s Credentials section describes the result.

### Phase S5: `catalog` work, in place (2026-09-30)

* **Step 1** (the move) is struck, and moved to S7b.
* **Step 2 (D9).** The nine static catalogs, `Static*` (the eight from
  `mcplib`, and `StaticTogether`), and `ProviderEnvVars` are unexported.
  * `StaticModels(provider)` already returned a copy.
  * `ProviderEnvVars()` is now a function returning a copy.
  * The migration guide's 16 rows for these, and for the `Rank*Model`
    functions, now name their equivalents. G-parity: `18 with an SDK
    equivalent, 0 problem(s)`.
* **Step 3.** The seven `Rank*Model` functions are unexported behind
  `RankModel(provider, model)`. Both OpenCode gateways share one ranking;
  Together, Ollama and unknown providers score 0. It takes a string until
  S6's `ProviderID`, and becomes `catalog.Rank` in S7b.
* **Step 4, 0016 T3 step 2 (0016-MADR D9).** Listing no longer probes by
  default. `WithModelProbes(true)` (option, `ProviderConfig.ProbeModels`)
  turns the probe back on for OpenAI with an API key, Claude, Gemini, Grok
  and Ollama. Its doc says each probe is a billed request.
* **G-wire difference, caused by 0016-MADR D9** (S2 step 5's rule). Five
  listing goldens were regenerated with a scoped `-update`:
  * `openai`, `claude`, `grok` and `ollama` each lose 2 probe `POST`s, and
    `gemini` loses 1: 9 requests in all;
  * 188 lines were removed and none added, so every listing `GET` and every
    decoded result is unchanged. `git diff` shows only `"method": "POST"`
    requests removed.

  No other golden changed.
* **Replaced test.** `TestDiscoverModels_APIKeyOpenAIStillProbes` pinned
  probing by default, which D9 reverses. It is replaced by
  `TestDiscoverModels_ProbesOnlyWhenEnabled`: for OpenAI, Claude, Gemini and
  Grok, no generation by default, and between one and `MaxListedModels`
  with the option. Ollama's listing needs a different fake; its regenerated
  golden pins its default.
* **Tests:** `catalog_state_test.go` covers `ProviderEnvVars()` and
  `StaticModels` returning copies, and `RankModel` dispatching to each
  provider's ranking.
* **Red first** is by absence for the new API. The old default is seen in
  the five goldens' diff above. Seen to fail on deliberate breaks:

  | Break | Failure |
  |---|---|
  | Claude probes by default | `by default: 4 generation requests, want none` |
  | OpenAI ignores the option | `with WithModelProbes(true): 0 generation requests, want one per candidate` |
  | `ProviderEnvVars` returns the package's map | `a caller's change reached the package: map[… openai:CHANGED …]` |
  | `RankModel` sends Gemini to OpenAI's ranking | `RankModel(gemini, gemini-3.7-flash) = 180, want 280` |
  | `StaticModels` returns the catalog itself | `a caller's change reached the static catalog` |
* **Gate,** every step exit 0: `llmprovider` 90.1 %, `wizard` 83.4 %,
  `make lint` `0 issues.`, G-wire three times over (with the five listing
  goldens as recorded), G-parity, G-links, and the deny list at 0 hits.

### Amendment 2026-09-30: S5 step 4 reversed (0016-MADR A5)

* **Decision.** The owner reversed 0016-MADR D9's default after S5 landed:
  * probes are on by default again;
  * `WithModelProbes(bool)` sets them per provider, either way;
  * the opt-in helper `ModelProbesFromEnv()` reads `LLMPROVIDER_PROBES`.
* **The five G-wire listing goldens** return to their `P7` content, with the
  9 probe requests. That is S5's recorded difference undone, again by a
  record.
* **For S10.** `ModelProbesFromEnv` is an opt-in helper under D9, like the
  planned `catalog.OptionsFromEnv`. S10's ambient check must allow reads
  inside the named helpers, and nowhere else.

### Execution 2026-09-30: S5 step 4 reversal (0016-MADR A5)

* **Code.** `ProviderConfig.ProbeModels` is replaced by
  `DisableModelProbes`, so the zero value probes.
  * `WithModelProbes(enabled)` sets it either way.
  * `ModelProbesFromEnv()` is in `options.go`, and reads
    `LLMPROVIDER_PROBES` with `os.LookupEnv`. An unset or non-boolean value
    gives an option that changes nothing.
  * The OpenAI (API key), Claude, Gemini, Grok and Ollama constructors set
    `probeModels: !cfg.DisableModelProbes`. The ChatGPT constructor does too,
    and its session still never probes.
* **G-wire.** The five listing goldens were restored from `5b8c2fe`, their
  content before S5. `TestWireGoldens` passes against them unchanged, with
  no `-update`.
* **Replaced test.** `TestDiscoverModels_ProbesOnlyWhenEnabled` pinned S5's
  default, which A5 reverses. It is replaced by
  `TestDiscoverModels_ProbesFollowDefaultOptionAndEnv`, which covers OpenAI,
  Claude, Gemini and Grok in eight cases:
  * probes by default, and with `WithModelProbes(true)`;
  * none with `WithModelProbes(false)`;
  * `LLMPROVIDER_PROBES=false` changes nothing without the helper;
  * with the helper, `false` stops probes, `true` keeps them, and a
    non-boolean value changes nothing;
  * an explicit option after the helper wins.
* **Red first.** In a scratch copy of this change carrying S5's five
  listing goldens (from `a448143`), `TestWireGoldens` fails exactly the five
  `*/listing` cases, each with `N difference(s) from the golden file`: the
  missing probe requests. The helper is new API, so its red is by absence.
  Seen to fail on deliberate breaks, each in a scratch copy:

  | Break | Failure |
  |---|---|
  | Claude off by default | `claude: 0 generation requests, want one per candidate` |
  | Grok ignores `WithModelProbes(false)` | `grok: 6 generation requests, want none` |
  | `WithModelProbes` stores the value unnegated | `claude: 4 generation requests, want none` |
  | The helper ignores the variable | `claude: 4 generation requests, want none` |
  | A non-boolean value disables probes | `openai: 0 generation requests, want one per candidate` |
  | `ApplyOptions` reads the variable itself | `openai: 0 generation requests, want one per candidate` |

### Deviation 2026-09-30: S6's new names are already taken

* **Found.** Before any change, a read-only scan showed that S6 step 1
  cannot keep "the old types alongside": Go allows one declaration per name
  in a package, and these names are already declared.

  | New name | Existing declaration | Absorbs the new shape |
  |---|---|---|
  | `Provider` | `llmprovider/provider.go:20`, `{Name() string; Generate(ctx, string) (string, error)}` | no: same name, different `Generate` |
  | `WithHTTPClient`, `WithMaxTokens`, `WithBaseURL`, `WithClientInfo`, `WithSessionID` | `options.go:94-108`, `identification.go:27,36`, each returning `ProviderOption` | only if one option type serves both |
  | `Response` | `item.go:58` (`ID`, `Output`, `FinishReason string`) | yes, by adding fields |
  | `APIError` | `api_error.go:50` (`Type`, `Terminal`) | yes, by adding fields |

  Typing the provider-id constants as `ProviderID` breaks each of their 677
  references that is a `string` parameter, map key or switch.
* **Options put to the owner:**
  1. converge in place, and rename only the incompatible `Provider`;
  2. temporary names for the new contract, renamed in S8;
  3. the contract in a separate package until S8, which needs a D2
     amendment.

  For the constants: add the type now and retype in S8, or retype in S6.
* **Decision.** The owner chose option 1, and the type now with the
  constants retyped in S8.
* **Changed, for S6:**
  * `Response` gains `Model` and `Usage`, and its `FinishReason` becomes the
    named type. That field has one assignment site.
  * `APIError` gains `Kind` (the kind sentinel, until now the unexported
    `sentinel`), `Code`, `Reason` and `Retryable()`. `Type` duplicates
    `Code`, and with `Terminal` is marked `Deprecated:` until S8.
  * The old `Provider` interface is renamed `LegacyProvider`, marked
    `Deprecated:`, and removed in S8.
  * `Option` is the one option type. Until S8, `ProviderOption` is an alias
    of it, so the old constructors, `ApplyOptions` and `wizard` take the
    same values. An option that exists only in the old API is refused by
    the new `New`, as a foreign option is.
  * `ProviderID` is declared and used by the new contract. The constants
    stay untyped until S8.
* **Applied on the same principle, for the owner to see:**
  * `MessageItem.Role` keeps `string` until S8, for the same reason as
    the constants: 56 non-test references. `Role` and its constants are
    declared now.
  * The existing sentinels keep their `llm:` text until S8. While
    `RateLimitError` exists, its doc promises the original message
    verbatim, and `provider_test.go:208` pins that. The new sentinels start
    `llmprovider:`, as R27 requires.
* **S8** gains these removals and the retyping, as its steps 3 and 4.

### Phase S6: the contract, and `llmtest` (2026-09-30)

Executed under the deviation of the same date: the new names converge on the
old ones in place.

* **Step 1, the contract, in `llmprovider`:**
  * `contract.go`: `Provider`; `Request`; `Reasoning`; `Usage`; the typed
    `ProviderID`, `Role`, `Effort`, `ToolChoice` (with `ForceTool`) and
    `FinishReason`; `Capabilities.Check`, which validates a request (R23)
    and refuses an Unsupported need before the network (R11).
  * `capabilities.go`: `Capabilities` and `Support` (`Unsupported`,
    `BestEffort`, `Supported`).
  * `stream.go`: `Event`, `EventType`, `Streamer` and `Stream`, with the
    fallback over `Generate`.
  * `settings.go`: `Option`; the new common options `WithModel`,
    `WithAPIKey`, `WithTokenSource`, `WithLogger` and `WithReasoning`;
    `ScopedOption` for a provider package's own options; and
    `ResolveOptions`, which returns read-only `Settings` and refuses a
    foreign option or one only the old API takes. The five existing common
    options, and the ten old-API-only ones, are now `Option` values.
  * `api_error.go`: `APIError.Kind`, `Code`, `Reason` and `Retryable()`;
    `ErrContextOverflow`, `ErrIncomplete` and `ErrUnsupported`.
    `IncompleteError` also matches `ErrIncomplete`.
  * `context_overflow.go`: the three service error types and the 21 message
    forms, each citing pi `packages/ai/src/utils/overflow.ts` at `0e283203c`
    by line. It also has pi's rate-limit exclusion. A 4xx the status alone
    maps to `ErrInvalidRequest` is checked, and so is a
    `context_length_exceeded` stream failure.
  * `retry.go`: `WithRetry` and `RetryPolicy`.
  * `registry.go`: `Registry`, `Descriptor` and `Factory`.
  * `convenience.go`: `GenerateText` and `GenerateToolCall` (R8). They are
    part of D3's contract, although step 1's list does not name them.
  * `Response` gains `Model` and `Usage`. Its `FinishReason` is typed, and
    `chatcompletions.go` converts at its one assignment.
  * `Provider` was renamed `LegacyProvider` with `gopls rename`, across 9
    files.
* **Step 2, `llmprovider/llmtest`:**
  * `Run(t, Harness)` runs seven subtests, each named after its rules:
    `R10-R11-capabilities`, `R23-invalid-values`, `R40-cancellation`,
    `R25-R26-classification`, `R44-identity`, `R7-R9-response` and
    `R20-concurrency`.
  * It runs over a small `reporter` interface, so the package's own tests
    record what a flawed provider reports.
  * A panic in `Generate` is reported as an R7 failure instead of ending
    the run.
  * The classification check reads `APIError.Kind` itself. `Unwrap` also
    returns the status's own sentinel (MADR 0012 §7), which would hide a
    misclassified 429.
  * `Fake` is scriptable (`Reply`, `ReplyText`, `Fail`, `Handle`). It
    refuses what its capabilities lack, records copies of the requests, and
    is safe for concurrent use.
* **Step 3, first-fail.** A conformant reference provider (in
  `llmtest_test.go`) passes `Run`. In a scratch copy, it was switched to
  each flaw, and `Run` was run on the real `*testing.T`:

  | Flaw | Failure |
  |---|---|
  | ignores cancellation | `R40-cancellation`: `R40 (cancellation): Generate did not return within 2s of its context being cancelled` |
  | sends a request with an unsupported capability | `R10-R11-capabilities`: `R11 (refusal before the network): a request needing continuation, which is Unsupported, returned <nil>` … `sent 1 request(s); want none` |
  | misclassifies a 429 | `R25-R26-classification`: `R25 (errors by kind): HTTP 429 returned an APIError of kind llm: provider unavailable; want ErrRateLimited` |
  | mutates shared state without a lock (`-race`) | `R20-concurrency`: `race detected during execution of test` |

  The first three are also in-tree tests (`TestRun_NamesTheBrokenRule`),
  through the reporter.
* **Step 4, the contract's tests.** Breaks, each in a scratch copy, and the
  test each one failed:

  | Break | Failure |
  |---|---|
  | `Stream`'s fallback drops text deltas | `5 events …, want 6` |
  | `Stream` ignores a native `Streamer` | `Stream called Generate on a native streamer` |
  | `WithRetry` ignores `RetryAfter` | `waited 1.554291ms, want at least the 40ms the service asked for` |
  | `WithRetry` retries every kind | `2 calls, want 1` |
  | `Retryable` ignores exhausted quota | `Retryable() = true, want false` |
  | `Registry` accepts a duplicate | `a duplicate: err = <nil>, want ErrInvalidProvider` |
  | `New` accepts a foreign option | `err = <nil>, want the foreign option named` |
  | `New` accepts an old-API-only option | `WithStore: err = <nil>, want it refused` |
  | `Check` ignores continuation | `Check = <nil>, want an error matching ErrUnsupported …` |
  | validation accepts an unknown tool choice | `Check = <nil>, want an error matching ErrInvalidRequest` |
  | `GenerateToolCall` does not force the tool | `sent ToolChoice "", want ForceTool(lookup)` |
  | `WithReasoning` keeps the caller's pointer | `a caller's change to the reasoning reached the Settings` |
  | overflow without the rate-limit exclusion | `… context window exceeded: grok HTTP 400: Too many tokens, rate limit reached …; want llm: invalid request and not ErrContextOverflow` |
  | classification never checks overflow | `llm: invalid request: together HTTP 400: context length exceeded; want ErrContextOverflow, beneath ErrInvalidRequest`, for every sample |
  | `IncompleteError` loses `ErrIncomplete` | `IncompleteError must match ErrIncomplete and ErrInvalidRequest` |
  | an `APIError` without `Kind` has no kind | `"<nil>: p HTTP 401 bad_key"` |
* **`ErrContextOverflow`, a red test for each entry.** In a scratch copy,
  each of the 24 entries (3 types, 21 message forms) was deleted in turn.
  Each deletion failed exactly its own case of
  `TestContextOverflow_Types` or `TestContextOverflow_Messages`, and no
  other: `24 entries, 0 unexpected`.
  `TestContextOverflow_EveryFormHasASample` keeps the table and its
  samples in step.
* **Lint** (`make lint`) first found 12 issues. Each was fixed, none
  suppressed:
  * unclosed bodies in the new tests, which now use the existing
    `classifyBody` helper;
  * `==` on errors: `errors.Is` in `classifyAPIError`, and in the tests
    `sameError`, which keeps "returned unwrapped";
  * a `%v` for an error;
  * `llmtest`'s `run` beside `Run`, renamed `runChecks`.

  The breaks above were re-run after these fixes, and each failed again as
  shown.
* **Existing tests.** They pass unchanged, including G-wire
  (`TestWireGoldens`, no `-update`): no request changed. No assertion
  changed.
* **Coverage** (`go test -race -cover`): `llmprovider` 90.9 % (90.3 %
  before); `llmtest` 94.7 %, new.
* **Docs.**
  * The migration guide fills seven rows: `Provider`, `APIError`,
    `Response`, `ProviderOption`, and the three `Generate*WithRetry`.
  * `docs/architecture.md` gains "The contract", and `llmtest` in the tree
    and the package table.
  * The standards guide is unchanged: R8's names were kept, and R25 already
    named `ErrContextOverflow`.
* **Not done, by the deviation:**
  * the constants and `MessageItem.Role` stay untyped;
  * the old sentinels keep `llm:`;
  * `RateLimitError` and `IncompleteError` stay.

  S8 does these.

### Amendment 2026-09-30: the S7 prerequisites settled

The deviation "the extractions of S3, S4 and S5 move after S7" left two
ordering conflicts open until S7. A read-only count on 2026-09-30 found:

* only tests call `NewProvider*`: 24 call sites in 6 files, 2 of them
  live-tagged. `wizard` calls neither.
* `ModelProfile` is used by `options.go`, `discovery.go`, the open-catalog
  providers (Kilo, OpenCode, Hugging Face, Together) and `wizard/configure.go`.

The owner decided:

1. **`NewProvider` and `NewProviderWithSource`.** Chosen: start
   `llmprovider/providers` in S7.
   * S7's first commit removes both, and creates `providers` with
     `Default()` and `New`.
   * Each later S7 commit registers its provider.
   * S8 step 1 keeps the descriptors' move and the coverage test.

   Rejected:
   * shrinking `NewProvider` by one case per commit, which left no by-name
     constructor for every provider during S7;
   * removing both with nothing until S8.
2. **`ModelProfile`.** Chosen: the `catalog` extraction moves from S7b to a
   new Phase S8b, after S8 removes `ProviderConfig`.
   * S7b extracts three packages.
   * `wizard`'s `ModelProfile` references change twice.

   Rejected:
   * a scoped profile option per provider in S7, which touched `wizard`
     before its S8 port;
   * keeping `ModelProfile` in `llmprovider`, which contradicts 0015-MADR
     D2 and needed a MADR amendment.

No MADR decision changes: the end state is D2's layout.

### Phase S7, commit 1: `providers`, and `NewProvider*` removed (2026-09-30)

Executed as the S7 prerequisites amendment of the same date decided.

* **Removed:** `NewProvider` and `NewProviderWithSource`
  (`llmprovider/provider.go`).
* **Added:** `llmprovider/providers`, with `Default()` (a new, empty
  `Registry` each call) and `New(id, opts...)`.
* **Tests that used them.** Each moved without changing what it asserts:
  * `grok_oauth_test.go` (5 calls): `newGrokWithSource`, which
    `NewProviderWithSource` called.
  * `TestNewOpencode_RoutesResolved` (renamed from `TestNewProvider_…`) and
    `TestNewOpenAI_APIKeyStillPlatform` call their constructors.
  * The two live tests build the provider their table names through a local
    helper, `liveWithKey` or `liveWithSource`.
  * `TestNewProvider`'s dispatch loop, `ollama_test.go`'s by-name empty-key
    check and `TestDescriptors_EveryDescriptorIsConstructible` became one
    test in `providers`. While S7 runs, it builds a provider through
    `Default()` once it has moved, and through the old constructor in a
    table, `notYetMoved`, until then. It fails if a descriptor is in
    neither, or in both. The table is empty when S7 ends.
  * The unknown-name half of `TestNewProvider` is
    `TestNew_RefusesAnUnknownProvider`, with `ErrInvalidProvider`.
  * **Carried:** `TestNewProviderWithSource_RejectsClaude` asserted that
    Claude takes no token source. With no by-name source constructor left,
    the same property is R16's refusal of an unaccepted source kind. It
    becomes a test in Claude's S7 commit.
* **Interim gap.** Grok with a token source, such as an OAuth session, had
  one public path, `NewProviderWithSource`. Grok's own source constructor,
  `newGrokWithSource`, is unexported. So until Grok's S7 commit gives it
  `New` with `WithTokenSource`, no exported function builds Grok from a
  session.
  * §0 allows an API break between phases, because nothing imports this
    module before `v1.0.0-rc.1`.
  * `wizard` does not build Grok from a session.
  * The Grok commit closes the gap.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | Together dropped from `notYetMoved` | `descriptor "together" is offered to users but neither Default nor the old API builds it` |
  | a constructor that fails | `descriptor "claude" is offered to users but its constructor fails: claude api key is required` |
  | OpenAI registered in `Default` while still in the table | `"openai" is in Default and still in notYetMoved; remove it from the table` |
  | `Default` shares one registry | `Default returned the same Registry twice; there is no global registry (0015-MADR D10)` |

  A first "failing constructor" break used Gemini. `NewGemini` accepts an
  empty key, so it failed nothing and was replaced.
* **Coverage:** `providers` 100 %; `llmprovider` 90.8 %.
* **Docs:** the migration guide maps `NewProvider*`; `architecture.md`
  gains the package, and says how to construct a provider during S7.

### Amendment 2026-09-30: listing and option scoping decided

* **Found.** Before the openai commit, two questions no record answered:
  * how a caller reaches `ListModels` when `New` returns `Provider`;
  * where A5's probe options live in the new API. S6 had classed them as
    old-API-only; `settings.go` refuses them, and
    `TestResolveOptions_RefusesAnOldAPIOnlyOption` pins that for
    `WithModelProbes` and `ModelProbesFromEnv`.
* **Decided by the owner,** recorded in the 0015-MADR amendment of the same
  date:
  * `ModelLister`;
  * a common baseline now, with the `For(id, …)` overlay built in S8b.
* **Changed:**
  * S7 gains a commit before openai (see S7's note).
  * S8b gains steps 4 and 5.
  * S6's classification of the two probe options is corrected.
    `TestResolveOptions_RefusesAnOldAPIOnlyOption` drops them from its list
    in that commit, which is a change to its data made by this decision.
    Its meaning, that old-API-only options are refused, is unchanged.

### Phase S7, commit 2: `ModelLister`, and the probe options made common (2026-09-30)

* **`llmprovider.ModelLister`** (`contract.go`), as the 0015-MADR amendment
  of the same date decided. Each provider that moves and lists implements
  it.
* **`WithModelProbes` and `ModelProbesFromEnv` are common options.** When
  the variable is unset or not a boolean, the helper returns an option that
  changes nothing. `Settings.ModelProbes()` reports the setting, true by
  default. The old constructors are unaffected: they read the same
  `ProviderConfig` field.
* **Tests.**
  * `TestResolveOptions_RefusesAnOldAPIOnlyOption` drops the two options
    from its list, as the PLAN amendment of the same date recorded.
  * `TestResolveOptions_ModelProbesAreCommon` is new.
  * Its first draft built its options in the table literal, which reads
    `LLMPROVIDER_PROBES` before `t.Setenv` runs. The env case failed as a
    result. The options are now built inside each subtest.
* **Red first and breaks**, in scratch copies:

  | Break | Failure |
  |---|---|
  | `WithModelProbes` old-API-only, as before this commit | `ResolveOptions: llm: invalid request: option WithModelProbes belongs to the old API; openai's New does not take it` |
  | `ModelProbes()` reads the flag unnegated | `ModelProbes() = false, want true` |
  | the helper's unset case old-API-only | `… option ModelProbesFromEnv belongs to the old API …` |

### Deviation 2026-09-30: llmtest finds unclassified token errors (S7, before openai)

* **Found.** While the openai package was being written, its llmtest run
  with a ChatGPT session failed:

  ```text
  R25 (errors by kind): HTTP 401 returned llmprovider: openai: acquire token: oauth: no refresh token; want an error matching ErrAuthFailure
  ```

  `OAuthSession.Token` returned a plain `oauth: no refresh token`
  (`oauth_session.go:119`). A refresh the endpoint rejected was classified
  only for a 401 or a terminal code. Both are older than this plan, since
  the old `doGenerateItems` took the same path. The old retry helpers
  retried these errors.
* **Options put to the owner:**
  1. classify in the session, in a commit of its own before openai;
  2. classify in each provider;
  3. defer, with the ChatGPT llmtest run skipping the 401 check (not
     recommended, as it loosens a check).

  The owner chose option 1.
* **Refined before the code was written.** Option 1 as offered mapped every
  rejected refresh to `ErrAuthFailure`. But
  `TestOAuthRefresh_BadRequestIsNotRetried` asserts that a non-terminal 400
  (`invalid_request`) is not one: that 400 is a malformed request, which
  signing in again does not fix. The owner chose `ErrInvalidRequest` for
  it, which keeps that test's meaning. The map:

  | Case | Kind |
  |---|---|
  | lapsed, no refresh token | `ErrAuthFailure` |
  | 401, or a terminal code | `ErrAuthFailure` (unchanged) |
  | 429 | `ErrRateLimited` |
  | 5xx | `ErrProviderUnavailable` |
  | any other 4xx | `ErrInvalidRequest` |

  Still unclassified, and not in this change: a transport failure after the
  refresh's retries, which the default retry rule treats as a failure to
  reach the service; and a 200 that cannot be decoded or has no access
  token.
* **Execution** (the stash of the openai work was set aside for this commit
  and restored after it):
  * `oauth_session.go`: `Token` and `refreshFailure` as in the map above.
  * `oauth_refresh_kinds_test.go` adds
    `TestOAuthToken_NoRefreshTokenIsAuthFailure` and
    `TestOAuthRefresh_FailureKinds`. Existing tests pass unchanged.
  * **Red first,** on the tree before the fix: the no-refresh-token case
    and four of the six kinds failed. For example:
    `Token = oauth: refresh failed: 503 Service Unavailable: , want an error matching llm: provider unavailable`.
    The 401 and terminal-code cases passed, as they already did.
  * **Breaks,** in scratch copies:

    | Break | Failure |
    |---|---|
    | every other 4xx an auth failure, the rejected alternative | `oauth_refresh_kinds_test.go:49: Token = llm: authentication failed: … 400 Bad Request …, want an error matching llm: invalid request`, and the existing `oauth_refresh_test.go:194: … want one plain failure` |
    | a 429 an auth failure | `Token = llm: authentication failed: … 429 Too Many Requests …, want an error matching llm: rate limited` |

### Phase S7, commit 3: `openai` (2026-09-30)

* **Step 1, moved** with `git mv`:
  * `llmprovider/openai.go` to `llmprovider/providers/openai/openai.go`;
  * the goldens `testdata/wire/{openai,chatgpt}`;
  * the three `chatgpt-*.sse` fixtures.

  `openai_chatgpt.go` stays in `llmprovider` with only the session helpers.
  `discovery.go`'s ChatGPT listing uses them too, so moving them would have
  made `llmprovider` import the provider.
* **Step 2, the new API:**
  * `openai.New(opts...)`, `ID`, `Capabilities`, `Generate` and
    `ListModels` (`ModelLister`);
  * `openai.WithStore`, a scoped option;
  * a reasoning default from `WithReasoning`, which replaces
    `WithReasoningEffort`.

  Each old method is the `Request` it sent:

  | Old | New |
  |---|---|
  | `GenerateItems` | `Input` |
  | `GenerateWithTool` | one tool and `ForceTool` |
  | `GenerateThinking` | `Reasoning`, with medium when it names no effort |
  | `Continue` | `PreviousResponseID` |
  | `DiscoverModels` | `ListModels` |

  A listing probe sends the old probe provider's body: output limit 8192,
  no `store`, no reasoning. The listing gets the caller's options, the
  provider's session, client and base URL. The response body is closed
  with the failure logged to the provider's own logger (R31), where the old
  code used the global one.
* **Temporary exports** from `llmprovider`, which S7b removes:
  * the session helpers `IsChatGPTSession`, `ChatGPTSessionAccountID`,
    `ChatGPTSessionFedRAMP` and `ExpireSession`, and the header constants
    `ChatGPTAccountHeader`, `ChatGPTOriginatorHeader`,
    `ChatGPTOriginatorValue` and `ChatGPTFedRAMPHeader`;
  * `ClassifyHTTPError`, `DecodeResponsesAPIOutput`, `ReadResponsesStream`,
    `ItemsToInput`, `ShareHTTPClient` and `ProbeGenerateHealth`.

  They were renamed with `gopls rename`. Their mentions in live-tagged
  files and comments, which gopls does not see, were renamed separately.
  `CloseResponseBody` was exported, then made unexported again when openai
  stopped using it. Lint's revive rejected `ChatGPTAccountID` and
  `ChatGPTFedRAMP`, which clash with methods of `oauth_loopback.go`, so they
  took the `Session` names.
* **Step 3.** The package doc lists the degradations:
  * `Reasoning.Budget` is not sent;
  * a ChatGPT session sends no output limit and always `store: false`.
* **Step 4, the tests ported.** Assertions keep their meaning. The call
  syntax is the new API's.

  | From `llmprovider` | To `openai` |
  |---|---|
  | `openai_items_test.go`, and the OpenAI parts of `thinking_test.go`, `provider_correctness_test.go`, `api_error_message_test.go`, `identification_test.go`, `responses_store_test.go` | `openai_test.go` |
  | `openai_chatgpt_test.go`, `chatgpt_stream_test.go`, and the OpenAI parts of `chatgpt_fedramp_test.go`, `vendor_session_test.go`, `command_token_test.go` | `chatgpt_test.go` |
  | the OpenAI parts of `probe_test.go`, `probe_scope_test.go`, `transport_defaults_test.go` | `listing_test.go` |
  | the openai and chatgpt G-wire cases | `wire_test.go`, through `llmprovider/internal/wirecase` |

  * **Interface assertions**, such as `var _ ThinkingProvider = …`, are
    now assertions on `Capabilities()` and `ModelLister`, what those
    interfaces declared.
  * **Split, each half kept:**
    * FedRAMP: the claim half stays as `TestChatGPTLogin_FedRAMPClaim`,
      and the header half moved;
    * the 401 rerun: the source half stays as
      `TestCommandToken_RerunsAfterInvalidate`, and the provider half moved
      as `TestOpenAI_RetriesOnceAfterInvalidate`.
  * **One assertion changed kind, by 0015-MADR D4.**
    `TestOpenAIChatGPT_ContinueIsInvalid` is now `…ContinueIsUnsupported`.
    A ChatGPT session's continuation is still refused with no request, but
    now with `ErrUnsupported` rather than `ErrInvalidRequest`, because
    continuation is `Unsupported` there.
  * **Read differently, by D7.** `TestOpenAIChatGPT_StreamFailures` reads
    "terminal" through `Retryable()`, because `Terminal` is deprecated;
    for these kinds the two say the same.
  * **New tests** for fields the old methods could not express:
    `request_test.go` (model, instructions, the tool choices, an effort per
    request, the default reasoning, a budget alone), `TestNew_*`, and
    `TestListModels_ProbeBodyIsTheOldProbes` and
    `TestListModels_CarriesTheCallersIdentity`.
  * **Live tests.** The ChatGPT ones, and the OpenAI rows of the
    vendor-session and store tests, are external tests
    (`package llmprovider_test`) in `llmprovider`'s directory. They need its
    live helpers and its test-only build-version hook, which
    `live_export_test.go` exports to them. They build through `openai.New`.
* **G-wire, rewritten through the new API.** 13 of the 14 goldens match
  their `P7` content byte for byte.
  * The one difference is `chatgpt/continuation.json`, where only the
    `error` line changed, by D4:

    ```text
    error: want "llm: invalid request: openai: a ChatGPT session cannot continue a response; replay the items", got "llmprovider: unsupported: the request needs continuation"
    ```

    It was regenerated with a scoped `-update`. Its `requests` stay `[]`.
  * The first run also showed `result: want null, got (absent)`: the
    harness dropped a typed nil `*Response`. That was fixed in the harness,
    not the golden.
  * The gate's G-wire check now runs `./llmprovider/...`, not
    `./llmprovider`.
* **Step 5, llmtest.** `TestConformance` runs `llmtest.Run` with an API key
  and with a ChatGPT session. Both pass. The ChatGPT run first failed R25
  on a 401, which led to the deviation above and its commit.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | a ChatGPT session claims continuation | `Continuation = Supported, want Unsupported`, and the continuation golden |
  | a forced tool sends no `tool_choice` | `requests[0].body.tool_choice: want {"name":"get_weather","type":"function"}, got (absent)` |
  | probes ignore `WithModelProbes` | `1 generation requests, want none` |
  | a probe keeps the provider's limit and store | `probe body map[… max_output_tokens:77 … store:false]; want … max_output_tokens 8192, no store, no reasoning` |
  | the default effort is high | `reasoning.effort = map[effort:high], want default "medium"`, and `thinking.json` |
  | `MaxOutputTokens` ignored | `max_output_tokens = 500, want 77` |
  | a ChatGPT session sends `max_output_tokens` | `max_output_tokens = 321, want absent` |
  | `Generate` skips the capability check | `R23 (invalid values): an unknown tool choice returned <nil>; …` |
  | no retry after a 401 | `… after 0 invalidations (sent [Bearer key-1]); want one retry after the 401` |
  | the listing drops the caller's options | `listing requests ["GET go-llmprovider-sdk/(devel) …"]; want one GET naming wire-app/9.9.9` |
  | openai registered under another id | `descriptor "openai" is offered to users but neither Default nor the old API builds it` |
  | the harness drops a typed nil response | `result: want null, got (absent)` |

  Two first attempts proved nothing and were replaced:
  * dropping the listing's session id. No header on OpenAI's listing
    carries it; the option is kept, as the old code passed it.
  * removing the registration, which left an unused import and did not
    compile.
* **Coverage** (`go test -race -cover`):
  * `providers/openai` 97.8 %;
  * `providers` 80.0 %;
  * `llmprovider` 89.8 %, against its `P7` 89.2 %. The margin narrows as
    well-tested provider code leaves.
  * `internal/wirecase` has no tests of its own: 82.8 % through the openai
    tests (`-coverpkg`).
* **`providers`.** `Default()` registers openai, with the descriptor from
  `llmprovider.DescriptorFor` until S8. OpenAI left `notYetMoved`.
* **Links, the rule for every provider moved in S7.** G-links found 7
  links in `0003-PLAN-add-grok-xai-llm-provider.md` to
  `llmprovider/openai.go`, five with line anchors.
  * Each now points at the moved file.
  * The anchors are dropped. They cited a version from before the
    migration, and would point readers at the wrong lines of the new file.
  * The link text, the original citation such as `openai.go:105-116`, is
    kept.
  * The rationale around the links is unchanged.
* **History, observed after the commit (`00fe139`).**
  * The fixtures and the 13 unchanged goldens are recorded as renames
    (R100).
  * `openai.go` is not: the rewrite onto the new API changed more than git's
    rename threshold allows. `git log --follow` finds only the new commit,
    at `-M50%` and at `-M20%`. Its history stays at the old path:
    `git log -- llmprovider/openai.go`.
  * §0 asks for a move in its own commit "where practical". It was not
    practical here: moved unchanged, the file does not compile in its new
    package, and §0 allows no failing build between commits.
  * The same is expected for each provider S7 moves.

### Phase S7, commit 4: `claude` (2026-09-30)

Built on the openai commit's pattern; only what differs is recorded here.

* **Step 1, moved.** `llmprovider/claude.go` went to
  `providers/claude/claude.go`, and `testdata/wire/claude` with it, by
  `git mv`.
  * OpenCode's messages route shares the Messages wire, so it stays in
    `llmprovider`, in a new `messages_wire.go`:
    `claudeItemsToMessages`, `decodeClaudeResponse` and
    `defaultClaudeThinkingBudget`.
  * Temporary exports, for S7b to remove: `MessagesFromItems`,
    `DecodeMessagesResponse`, `AddMessagesThinking` and `SystemPrompt`.
    They were made with a reusable `gopls rename` script. Its
    word-boundary pass touched only comments beyond gopls's renames.
* **Step 2, the new API:**
  * `claude.New`, `ListModels` (through `ListAvailableModelsWithSource`,
    which runs the old listing with the same 10 s bound), and the key read
    from the source on each request.
  * `WithThinkingBudget` and `WithReasoningEffort` are `Reasoning`'s
    `Budget` and `Effort`. A request's `Reasoning` falls back to
    `WithReasoning`'s, field by field.
  * `Instructions` go before the system items in the system field.
  * `ToolChoiceRequired` is `any` and `ToolChoiceNone` is `none`. Auto sends
    no `tool_choice`.
* **Carried test.** `TestNewProviderWithSource_RejectsClaude` is
  `TestNew_RefusesAnOAuthSession`. `New` refuses an `OAuthSession` or a
  `VendorCLISession` with `ErrUnsupported` (R16, 0016-MADR D10). An empty
  static key is still refused, as `NewClaude("")` was, now with
  `ErrInvalidRequest`.
* **Step 3.** The package doc lists the degradations:
  * a forced or required tool is sent as `auto` while thinking;
  * a budget is not sent to an adaptive-only model.
* **Step 4, the tests ported.** Assertions keep their meaning.

  | From `llmprovider` | To `claude` |
  |---|---|
  | `claude_items_test.go`, and the Claude parts of `claude_system_test.go`, `thinking_test.go`, `thinking_wire_test.go`, `provider_correctness_test.go`, `api_error_message_test.go`, `identification_test.go` | `claude_test.go` |
  | the Claude parts of `probe_test.go`, `probe_scope_test.go`, `discovery_wiring_test.go` | `listing_test.go` |
  | the claude G-wire case | `wire_test.go` |

  * `TestThinkingWire_Claude` runs each case twice: with the reasoning set
    at construction, as the old options did, and on the request.
  * New: `request_test.go` (refusals, the key read per request, request
    fields); `TestListModels_ProbeBodyIsTheOldProbes`;
    `TestListModels_CarriesTheCallersIdentity`.
  * Live: `live_static_test.go` and `TestLive_ClaudeThinkingShapes` moved,
    and the Claude rows of the tool round trip and the system message went
    to `live_claude_test.go`. It is external, with `LiveEnvKey` added to
    `live_export_test.go`.
* **G-wire.** All six claude goldens match their `P7` content unchanged
  through the new API, with no `-update`. There is no continuation golden:
  the `P7` type had no `Continue` (`NoContinuation`).
* **Step 5, llmtest.** `TestConformance` passes.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an OAuth session accepted | `oauth: err = <nil>, want ErrUnsupported` |
  | an empty key accepted | `an empty key: err = <nil>, want ErrInvalidRequest` |
  | thinking keeps a forced tool | `thinking tool_choice = map[name:emit type:tool], want type=auto`, and `thinking-tool.json` |
  | the construction budget ignored | `budget_tokens = 4096, want 8000` |
  | instructions after the system items | `system = "Answer in French.\n\nBe brief."` |
  | the key read once | `x-api-key sent [key-1 key-1], want [key-1 key-2]` |
  | a probe keeps the provider's limit and reasoning | `probe body map[max_tokens:6144 … thinking:map[budget_tokens:2048 …]]; want … no thinking` |
  | probes ignore `WithModelProbes` | `1 generation requests, want none` |
  | `Generate` skips the capability check | `R11 (refusal before the network): a request needing continuation, which is Unsupported, returned <nil> …` |
  | claude registered under another id | `descriptor "claude" is offered to users but neither Default nor the old API builds it` |

  A first "forced tool" break deleted a `switch` case, which left a
  variable unused, so it did not compile. It was replaced.
* **Coverage:**
  * `providers/claude` 95.4 %;
  * `llmprovider` 89.7 %, against its `P7` 89.2 %;
  * `internal/wirecase` 74.1 % through claude's tests alone, and 82.8 %
    through openai's (no continuation scenario for claude).
* **Removed:** `listClaudeModels` in `discovery.go`. Only the old
  `DiscoverModels` called it, so lint found it unused once that was gone.
* **Links.** Six links in `0003-PLAN-add-grok-xai-llm-provider.md` to
  `llmprovider/claude.go` now point at the moved file, under the openai
  commit's rule.

### Deviation 2026-09-30: T3 step 1's header override was not built (S7, before gemini)

* **Found.** 0016-PLAN T3 step 1 lands in S7, per provider (the table in
  "0016 steps in these phases"). It asks that a non-empty `Token.Header`
  override the provider's header, and for a test that `Header: "X-Custom"`
  reaches the server. The openai and claude commits built neither:
  * `providers/openai/openai.go:259` always sent `Authorization: Bearer`;
  * `providers/claude/claude.go:141` always sent `x-api-key`;
  * `discovery.go` passed the listing only `token.Value`.

  Nothing in the module read `Token.Header`. The gap dates from `00fe139`
  (pushed) and `da4425a`.
* **Options put to the owner:**
  1. a commit of its own before gemini: a shared helper for openai and
     claude, for generation and listing, with the `X-Custom` test seen to
     fail first;
  2. the same, with listing moved to S8b;
  3. the override deferred to S7b.

  The owner chose option 1.
* **Found while building it.** The sources contradicted D2's rule:
  `StaticToken` filled in `Header: "Authorization"`, and every source
  reported `Type: bearer`. Applied as written, the rule would have sent
  `Authorization: bearer <key>` to Claude, and an override could never
  carry a bare key. The owner chose "sources say only what's set",
  recorded as 0016-MADR A6.
* **Scope.**
  * One commit: the helper, and the two sources' `Header` and `Type`.
  * openai and claude generation, and their listings, the ChatGPT listing
    included.
  * The tests, and the records.

  The providers still in `llmprovider` keep their headers. Each later S7
  commit builds the override for the provider it moves, as T3 step 1 asks.

### Phase S7, commit 5: the token header override (2026-09-30)

The deviation above, executed. 0016-PLAN T3 step 1 for openai and claude.

* **The sources** (0016-MADR A6):

  | Source | `Type` | `Header` |
  |---|---|---|
  | `StaticToken` | `TokenAPIKey` (was `TokenBearer`) | its field, empty by default (was `Authorization`) |
  | `CommandToken` | `TokenAPIKey` (was `TokenBearer`) | its field, unchanged |
  | `OAuthSession`, both paths | `TokenBearer` | none (was `Authorization`) |
  | `VendorCLISession` | `TokenBearer` | none (was `Authorization`) |

* **The rule, `token_header.go`.**
  * `tokenHeader` gives the header name and value for a token and the
    service's header and scheme.
  * `SetTokenHeader` sets them on a request. It is a temporary export, for
    S7b to move to `internal/transport`.
* **Generation.** `openai.go` sends `Authorization` with `Bearer`, and
  `claude.go` sends `x-api-key` bare, each through `SetTokenHeader`.
* **Listing.** `modelCatalogFor` takes the `Token`, not its value.
  * The OpenAI, ChatGPT and Claude listings apply the rule.
  * `fetchDataIDs` takes a header name and value in place of an
    `Authorization` value. Grok and OpenCode pass `Authorization` as
    before.
  * The other providers get the token's value, as before. Each S7 commit
    that moves one of them applies the rule to it.
* **One assertion changed meaning, by A6.** `TestStaticToken_ReturnsBearer`
  is `TestStaticToken_ReturnsAPIKey`. It expects `TokenAPIKey` and no
  `Header`, where it expected `TokenBearer` and `Authorization`. No other
  existing test changed.
* **New tests:**
  * `TestTokenHeader`, the rule's five cases;
  * `TestTokenSources_ReportOnlyWhatIsSet`, seven sources;
  * `TestOpenAI_TokenHeaderOverride` and `TestClaude_TokenHeaderOverride`.
    Each checks no header, an overriding bearer and an overriding key, in
    generation and in listing, and that the service's own header is absent
    beside an override.
* **Red first,** on a clone of `da4425a`. `TestTokenHeader` needs the new
  helper, so it was left out.
  * Both provider tests failed their four override checks. For example:
    `token_header_test.go:63: GET: X-Custom = "", want "v"`.
  * `TestTokenSources_ReportOnlyWhatIsSet` failed all seven cases. For
    example: `Type "bearer", Header "Authorization"; want "api_key", ""`.
* **G-wire.** No golden changed.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | the override ignored | `X-Custom = "", want "Bearer v"` |
  | an API-key override `Bearer`-prefixed | `X-Custom = "Bearer v", want "v"` |
  | the override keeps the service's header too | `headers map[X-Api-Key:[Bearer v] X-Custom:[Bearer v]], want only X-Custom` |
  | openai generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | openai listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | claude listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | a static key defaults to `Authorization` again | `Header = "Authorization", want none`. G-wire fails 13 cases, for example `claude/text`: `requests[0].header.Authorization: want (absent), got "sk-ant-wire"` |
  | an OAuth session names `Authorization` again | `Type "bearer", Header "Authorization"; want "bearer", ""` |

* **Coverage:** `llmprovider` 89.7 %, against its `P7` 89.2 %;
  `providers/claude` 95.4 %; `providers/openai` 97.2 %.

### Phase S7, commit 6: `gemini` (2026-09-30)

Built on the openai and claude commits' pattern; only what differs is
recorded here.

* **Staged, not committed.** Asked on 2026-09-30 about 0011's rule against
  agent commits on `main`, the owner declined a `feature/` branch and
  answered "i will commit an push. you only stage." This move is staged for
  the owner to commit. The two commits before it, `da4425a` and `9487fde`,
  were made by the agent on `main` before that was settled.
  *Annotated 2026-09-30:* the owner committed it together with the grok
  move, as `3d4aff5`.
* **Step 1, moved** with `git mv`:
  * `gemini.go` and `gemini_interactions.go` to `providers/gemini/`
    (`gemini.go`, `interactions.go`);
  * the three Gemini test files;
  * `testdata/wire/gemini`;
  * `live_gemini_interactions_test.go`, as `live_gemini_test.go`.
    `live_gemini_wire_test.go` is folded into it.

  OpenCode's google route shares Gemini's `generateContent` wire, but the
  gemini provider does not use it. So it stays in `llmprovider`, unexported,
  in a new `generatecontent_wire.go`: `geminiSystemInstruction`,
  `geminiItemsToContents`, `decodeGeminiResponse` and
  `dynamicGeminiThinkingBudget`. Its tests, `gemini_generatecontent_test.go`,
  are renamed `generatecontent_wire_test.go`.
* **Temporary export,** for S7b to remove: `ToolArguments`, which the
  Messages, `generateContent` and Interactions wires share.
  * It was renamed in `llmprovider`'s own files only: `wirecase` has an
    unrelated `toolArguments`, which the export script's word-boundary pass
    would have renamed too.
  * The placeholder signature `skip_thought_signature_validator` is a
    constant in each package, since both wires send it.
* **Step 2, the new API:**
  * `gemini.New`, `ID`, `Capabilities`, `Generate` and `ListModels`.
  * `gemini.WithStore`, a scoped option. `store` is `false` unless it is
    `true`, as before. Continuation is `Supported` only with it.
  * `WithReasoningEffort` is `Reasoning.Effort`. A request's `Reasoning`
    falls back to `WithReasoning`'s effort. Any `Reasoning` asks for thought
    summaries.
  * `Instructions` go before the system items in `system_instruction`.
  * Tool choice goes in `generation_config.tool_choice`: a named tool as
    `allowed_tools` (measured), required as `"any"` (measured), none as
    `"none"` (the API reference; the owner's decision), auto as nothing.
    Every tool in `Request.Tools` is sent.
  * The key is read from the source on each request, and sent by
    `SetTokenHeader` (0016-PLAN T3 step 1). The listing applies the same
    rule. `listGeminiModels`, which only the old type called, is removed.
  * **An empty key is still accepted.** `NewGemini` never refused one, and
    the service refuses it. An OAuth session or a CLI login is refused with
    `ErrUnsupported` (R16, 0016-MADR D10); `NewGemini` took only a key
    string.
  * **The probe** copies the old probe provider: the provider's own output
    limit (unlike claude and openai, whose probe providers used 8192),
    `store: false`, and no reasoning.
* **Step 3.** The package doc lists the degradations:
  * no budget;
  * the effort menu of `gemini-2.5-flash-lite`;
  * the unmeasured `"none"`.
* **Step 4, the tests ported.** Assertions keep their meaning.

  | From `llmprovider` | To `gemini` |
  |---|---|
  | `gemini_interactions_test.go` | `interactions_test.go` |
  | `gemini_items_test.go` | `items_test.go` |
  | `gemini_test.go`, and the Gemini parts of `provider_correctness_test.go`, `thinking_test.go`, `api_error_message_test.go`, `identification_test.go` | `gemini_test.go` |
  | the Gemini parts of `probe_test.go`, `probe_scope_test.go`, `discovery_wiring_test.go` | `listing_test.go` |
  | the gemini G-wire case | `wire_test.go` |
  | the Gemini assertions of `interface_test.go` | `TestGemini_Capabilities`, on `Capabilities()` and `ModelLister` |

  * **One assertion changed kind, by 0015-MADR D4.**
    `TestGemini_ContinueNeedsStore` still refuses a continuation without
    `WithStore(true)` before any request, now with `ErrUnsupported` rather
    than `ErrInvalidRequest`, since continuation is `Unsupported` there.
  * `TestGeminiInteractions_Thinking` runs each case twice: with the effort
    set at construction, as `WithReasoningEffort` did, and on the request.
  * `weatherTool`, which `together_test.go` and `wire_golden_test.go` also
    use, stays in `llmprovider`, in `wire_golden_test.go`.
  * `bodyCapture` in `provider_correctness_test.go` had no caller left and
    is removed.
  * **New tests:**
    * `request_test.go`: refusals, the empty key, the key read per request,
      the request fields, each tool choice, and reasoning's fallbacks;
    * `TestGemini_TokenHeaderOverride`;
    * `TestListModels_ProbeBodyIsTheOldProbes` and
      `TestListModels_CarriesTheCallersIdentity`.
  * **Live.** The Gemini live tests are external, in `live_gemini_test.go`.
    The Gemini rows of the tool round trip and the thinking shapes moved
    there. `TestLive_GeminiToolChoiceNone` is new, to measure `"none"` on
    the next live run. It has not been run.
* **G-wire.** All seven gemini goldens match their `P7` content unchanged
  through the new API, with no `-update`. Continuation is among them:
  `WithStore(true)` as the case's continuation option, as at `P7`.
* **Step 5, llmtest.** `TestConformance` passes, stored and unstored.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an OAuth session accepted | `oauth: err = <nil>, want ErrUnsupported` |
  | `WithStore` does not enable continuation | `Continuation = Unsupported, want Supported`, and `continuation.json`: 3 differences |
  | continuation claimed without a store | `Continuation = Supported, want Unsupported`; `err = <nil> after request "/interactions", want ErrUnsupported and none` |
  | a named tool sent as `"any"` | `tool_choice = any, want map[allowed_tools:…]`, and `tool.json` |
  | `ToolChoiceNone` omitted | `"none": tool_choice = <nil>, want none` |
  | the construction effort ignored | `gemini-3.7-flash/low at construction: generation_config = map[max_output_tokens:8192 thinking_summaries:auto], want … level low` |
  | instructions after the system items | `system_instruction = "Be brief.\n\nAnswer in French.", want the instructions, then the system items` |
  | `MaxOutputTokens` ignored | `max_output_tokens 500; want gemini-other, 77` |
  | the key read once | `x-goog-api-key sent [key-1 key-1], want [key-1 key-2]` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | a probe keeps the provider's store and reasoning | `probe body map[generation_config:map[max_output_tokens:77 thinking_level:high thinking_summaries:auto] … store:true]; want … store false, no thinking` |
  | probes ignore `WithModelProbes` | `1 generation requests, want none` |
  | `Generate` skips the capability check | `R11 (refusal before the network): a request needing continuation, which is Unsupported, returned <nil> …` |
  | gemini registered under another id | `descriptor "gemini" is offered to users but neither Default nor the old API builds it` |

  After the breaks, lint's gosec (G101) took the header-name constant
  `headerAPIKey` for a credential. It was renamed `googleKeyHeader`, which
  changes no behaviour.
* **Coverage:**
  * `providers/gemini` 95.6 %;
  * `llmprovider` 89.6 %, against its `P7` 89.2 %;
  * `providers` 80.0 %.
* **Links.** Four links in `0003-PLAN-add-grok-xai-llm-provider.md` to
  `llmprovider/gemini.go` were repointed under the openai commit's rule.
  One cites `gemini.go:119-137`, the `parts[]` decoding. That is the
  `generateContent` decoder, now in `llmprovider/generatecontent_wire.go`,
  so it points there. The other three name the provider's migration, and
  point at `providers/gemini/gemini.go`.

### Phase S7, commit 7: `grok` (2026-09-30)

Built on the earlier commits' pattern; only what differs is recorded here.

* **Not staged.** The gemini move was still staged, and uncommitted, when
  this one was done. So this move was left in the working tree, unstaged,
  for the owner to commit gemini's index first. It is staged once gemini is
  committed.
  *Annotated 2026-09-30:* the owner committed both moves together, as
  `3d4aff5`, so this one was never staged on its own.
* **Step 1, moved** (plain `mv`, since staging waits; git pairs the renames
  when it is staged):
  * `grok.go` and `grok_reasoning.go` to `providers/grok/` (`grok.go`,
    `reasoning.go`);
  * `grok_test.go`, `grok_oauth_test.go` (as `session_test.go`), and
    `grok_effort_menu_test.go` (as `reasoning_test.go`), into which
    `grok_reasoning_test.go` is folded;
  * `grok_tool_description_test.go`, folded into `grok_test.go`;
  * `testdata/wire/grok`;
  * `live_grok_effort_test.go`, as `live_grok_test.go`, into which
    `live_grok_tool_test.go` and `live_responses_store_test.go` are folded.
    `responses_store_test.go` is folded into `grok_test.go`.

  Kept in `llmprovider`:
  * `ItemsToInput`, the Responses wire that openai and OpenCode's responses
    route use. It moved from `grok.go` to `http_helpers.go`, beside
    `DecodeResponsesAPIOutput`.
  * `grokModel46` and `grokModel45`, which the static catalog uses. They
    moved to `models_catalog.go`. The grok package has its own copies.
  * `grok_catalog_test.go` and `live_grok_catalog_test.go`, which test the
    catalog.
* **No new temporary export.** `expireGrokSession` was `ExpireSession`
  word for word, so grok uses the export openai added.
* **Step 2, the new API:**
  * `grok.New`, `ID`, `Capabilities`, `Generate` and `ListModels`, and
    `grok.WithStore`, a scoped option.
  * `New` takes any source: a key, a command, an xAI `OAuthSession` or the
    Grok CLI's `VendorCLISession`. It shares its client with an OAuth
    session, as `newGrokWithSource` did. **This closes the interim gap
    recorded in "Phase S7, commit 1".**
  * An empty static key is refused with `ErrInvalidRequest`, as `NewGrok`
    refused it with a plain error.
  * A 401 from a source that can refresh is retried once, as before.
  * `WithReasoningEffort` is `Reasoning.Effort`, falling back to
    `WithReasoning`'s, then clamped to the CLI menu as before.
  * The token goes through `SetTokenHeader` in generation and listing
    (0016-PLAN T3 step 1).
  * **Two shapes the old methods never sent, chosen without a
    measurement.** Both have a live test that has not been run:
    * `Instructions` go in a leading system message, the shape the old API
      sent for a system item, not the Responses `instructions` field.
      `TestLive_GrokInstructions`.
    * `ToolChoiceRequired` and `ToolChoiceNone` are `"required"` and
      `"none"`, as openai sends them. `TestLive_GrokToolChoices`.

    The same rule as gemini's `"none"`: the documented form, pinned live.
  * **The probe** copies the old probe provider: the default output limit
    8192, no `store`, no reasoning.
  * `llmprovider.WithStore` now has no reader. Its doc says so; S8 removes
    it with the old API.
* **Step 3.** The package doc lists the degradations: the effort menu, no
  budget, and the two unmeasured shapes.
* **Step 4, the tests ported.** Assertions keep their meaning.

  | From `llmprovider` | To `grok` |
  |---|---|
  | `grok_test.go`, `grok_tool_description_test.go`, `responses_store_test.go`, and the Grok parts of `thinking_test.go`, `api_error_message_test.go`, `identification_test.go` (both tests), `interface_test.go` | `grok_test.go` |
  | `grok_oauth_test.go` | `session_test.go` |
  | `grok_effort_menu_test.go`, `grok_reasoning_test.go` | `reasoning_test.go` |
  | the Grok parts of `probe_test.go`, `probe_scope_test.go`, `transport_defaults_test.go` | `listing_test.go` |
  | the grok G-wire case | `wire_test.go` |

  * Tests that only Grok's rows kept alive left `llmprovider`:
    `TestDiscoverModels_ProbesFollowDefaultOptionAndEnv`,
    `TestProviderClient_SharedWithListingAndRefresh` with `pathRecorder`
    and `wireCaseNamed`, `TestProviderNamesAndConstructors`,
    `TestProvidersImplementThinkingInterfaces`, and
    `TestLive_VendorCLISession`. Its `liveVendorSession` stays, exported
    as `LiveVendorSession`.
  * `TestGrok_EmptyStaticKeyStillRejected` now also expects
    `ErrInvalidRequest`, and refuses a missing source. The old test asked
    only for an error.
  * **New tests:** `request_test.go` (the request fields, each tool choice,
    reasoning's fallbacks and clamping, every source kind accepted),
    `TestGrok_TokenHeaderOverride`, `TestListModels_ListingBounded` (Grok
    had no row in the old test), `TestListModels_ProbeBodyIsTheOldProbes`
    and `TestListModels_CarriesTheCallersIdentity`.
  * **Live.** `live_grok_test.go` is external. The Grok rows of the tool
    round trip and the CLI-login test moved there, as did the store and
    tool-description tests.
* **G-wire.** All seven grok goldens are byte-identical to `HEAD`'s,
  continuation included, with no `-update`.
* **Step 5, llmtest.** `TestConformance` passes with an API key and with an
  OAuth session.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an empty key accepted | `New(WithAPIKey("")) error = <nil>, want ErrInvalidRequest` |
  | no retry after a 401 | `Generate() error = llm: authentication failed: grok HTTP 401` |
  | the key read once | `authorizations/calls = [Bearer first-token Bearer first-token]/2` |
  | `WithStore` ignored | `store = <nil> (present false), want false (present true)` |
  | a forced tool sends no `tool_choice` | `tool_choice = <nil>, want map[name:get_weather type:function]`, and `tool.json` |
  | `ToolChoiceNone` omitted | `"none": tool_choice = <nil>, want none` |
  | instructions dropped | `input = [map[content:Be brief. role:system] …], want the instructions as a leading system message` |
  | `MaxOutputTokens` ignored | `max_output_tokens 500; want grok-other, 77` |
  | the construction effort ignored | `empty request effort takes the default: reasoning.effort = high, want medium` |
  | the effort not clamped | `reasoning must be omitted for grok-4`, and `reasoning.effort = medium, want low (clamped from medium)` |
  | grok-4.5 offered xhigh | `clampReasoningEffort("grok-4.5", "xhigh") = "xhigh", want "high"` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | the client not shared with the session | `POST /oauth/token did not go through the provider's client` |
  | a probe keeps the provider's limit, store and reasoning | `probe body map[… max_output_tokens:77 … reasoning:map[effort:high] store:false]; want … 8192, no store, no reasoning` |
  | probes ignore `WithModelProbes` | `1 generation requests, want none` |
  | `Generate` skips the capability check | `R23 (invalid values): an unknown tool choice returned <nil>` |
  | grok registered under another id | `descriptor "grok" is offered to users but neither Default nor the old API builds it` |

* **Coverage:**
  * `providers/grok` 96.1 %;
  * `llmprovider` 89.6 %, against its `P7` 89.2 %;
  * `providers` 80.0 %.
* **Links.** G-links found no link to the moved files.

### Amendment 2026-09-30: the OpenCode family (S7, before opencode)

* **Found** while preparing the opencode commit, before any write:
  * the package serves two ids, and R14 gives a package one `New`;
  * `ScopedOption` scopes to one id, so `opencode.WithRoute` could reach
    only one gateway;
  * `WithModelMetadataURL` was old-API only, and OpenCode reads the
    metadata on every request;
  * `WithModelProfile` is old-API only until S8b.
* **The owner decided,** as recorded in 0015-MADR, amendment "the OpenCode
  family, scoped options and the metadata URL":
  * `opencode.NewZen` and `opencode.NewGo`, with R14 amended;
  * `llmprovider.ScopedOptionFor`, for an option several ids take;
  * `WithModelMetadataURL` made common, with `Settings.ModelMetadataURL()`;
  * the profile gap accepted until S8b.
* **Scope.** One change before opencode adds `ScopedOptionFor` and the
  common `WithModelMetadataURL`, with their tests and R14. The opencode
  commit follows.
* **Annotated, the gemini and grok records.** Both say their move was left
  for the owner to commit separately. The owner committed them together, as
  `3d4aff5`.

### Phase S7, commit 8: `ScopedOptionFor`, and the metadata URL made common (2026-09-30)

The amendment above, executed. Staged for the owner to commit before the
opencode move, which waits unstaged.

* **`settings.go`.**
  * `Option` holds `scoped` and a list of ids in place of one `provider`.
  * `ScopedOptionFor(ids, name, value)` copies the list.
    `ScopedOption(id, …)` calls it with one id.
  * `ResolveOptions` takes a scoped option for any listed id and refuses it
    for any other. An empty list is refused by every `New`, never made
    common.
  * The refusal names one id as before (`is for provider "kilo"`), and
    several as `is for providers "opencode-zen", "opencode-go"`.
  * `Settings.ModelMetadataURL()` is new.
* **`options.go`.** `WithModelMetadataURL` is a common option. The old API
  takes it as before.
* **R14 and R17** in `docs/guides/api-standards.md` read as the amendment
  says.
* **Tests,** in `settings_family_test.go`:
  * `TestScopedOptionFor_EachListedProviderTakesIt`;
  * `TestScopedOptionFor_EmptyListIsRefusedEverywhere`;
  * `TestScopedOptionFor_CopiesTheList`;
  * `TestResolveOptions_ModelMetadataURLIsCommon`.

  `TestResolveOptions_RefusesAForeignOption` passes unchanged.
* **Red first,** on a clone of `3d4aff5`: `ResolveOptions` with
  `WithModelMetadataURL` failed with
  `option WithModelMetadataURL belongs to the old API; opencode-go's New does not take it`.
  `ScopedOptionFor` did not exist there, so its tests are proven by breaks.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | a scoped option taken by any id | `kilo: err = <nil>, want the option refused and its ids named`, and the existing `TestResolveOptions_RefusesAForeignOption` |
  | only the first listed id takes it | `opencode-go: … is for providers "opencode-zen", "opencode-go", not "opencode-go"` |
  | an empty list made common | `openai: err = <nil>, want ErrInvalidRequest` |
  | the caller's slice kept | `opencode-zen after the caller's slice changed: … is for provider "kilo"` |
  | the error names one id only | `is for provider "opencode-zen", not "kilo", want … its ids named` |
  | the metadata URL old-API only again | `option WithModelMetadataURL belongs to the old API` |
  | `ModelMetadataURL()` reads nothing | `ModelMetadataURL() = "", want the option's URL` |

### Phase S7, commit 9: `opencode` (2026-09-30; committed as `4adecf3`)

The owner committed this move together with commit 8, as `4adecf3`. It
follows the amendment "the OpenCode family".

* **Step 1, moved** (plain `mv`; git pairs the renames):
  * `opencode.go` and `opencode_route.go` to `providers/opencode/`
    (`opencode.go`, `route.go`);
  * `opencode_test.go`, `opencode_route_test.go` (as `route_test.go`),
    `opencode_conventions_test.go` (as `conventions_test.go`) and
    `opencode_metadata_route_test.go` (as `metadata_route_test.go`);
  * `testdata/opencode-routes.json`, and the seven `testdata/wire/opencode-*`
    golden directories.

  **Kept in `llmprovider`:**
  * In a new `opencode_gateway.go`, what the listing and the descriptors
    use: the two base URLs, `opencodeBaseURL` and the session header. The
    provider keeps its own copies, and `TestOpencodeBaseURLs_MatchTheDescriptors`
    checks they agree.
  * `firstFunctionCallArgs`, which four other providers use, moved to
    `http_helpers.go`.
  * The catalog filter `isUsableOpencodeModel`.

  **Removed with the old provider:**
  * `WithOpencodeRoute` and `ProviderConfig.OpencodeRoute`;
  * the `OpencodeRoute` type and its constants, which are now
    `opencode.Route` and its constants;
  * `listOpencodeModels`.

  The metadata's `opencodeRoute` method became `npm`. The npm-to-route map
  moved with the provider.
* **Temporary exports** from `llmprovider`:
  * for S7b to move to `internal/wire`:
    * the Chat Completions wire: `ChatCompletionsBody`,
      `ChatCompletionsOpts` and `DecodeChatCompletionsResponse`;
    * the `generateContent` wire: `GeminiItemsToContents`,
      `GeminiSystemInstruction`, `GeminiThinkingConfig` and
      `DecodeGeminiResponse`;
  * for S8b to move to `catalog`: `ModelMetadata` and
    `LookupModelMetadata`, the metadata as one request reads it.

  The functions were renamed with the export script. `ChatCompletionsOpts`
  was renamed with `gopls rename` under the live tag.

  `LookupModelMetadata` was first `LoadModelMetadata`. Lint's revive
  (confusing-naming) rejected that name, which differs only in case from
  the unexported loader.
* **Step 2, the new API:**
  * `opencode.NewZen` and `opencode.NewGo`, and `ListModels`;
  * `opencode.WithRoute`, scoped to both ids with `ScopedOptionFor`.

  **Credentials.**
  * With no credential, or an empty key, the gateway's `public` token is
    sent, as `NewOpencode("")` did.
  * An OAuth session or a CLI login is refused with `ErrUnsupported` (R16).
    `NewOpencode` took only a key string.
  * The key is read on each request. It is sent in each route's header, or
    in the token's own (R16), and so is the listing's (`fetchOpencodeUsable`
    takes the `Token`).

  **The route** is resolved per request:
  1. `WithRoute`'s;
  2. else the request model's `provider.npm` in the metadata;
  3. else the table, or the heuristic, for that model.

  A `Request.Model` therefore picks its own route. The old `Route()`
  accessor has no successor: the route is per request.

  **Request fields.**
  * `Instructions` go in a leading system item on every route: the
    `system` field on messages, `systemInstruction` on google, a system
    message elsewhere. These are the shapes the old API sent for a system
    item.
  * `WithReasoningEffort` and `WithThinkingBudget` are `Reasoning`'s
    `Effort` and `Budget`. A request's `Reasoning` falls back to
    `WithReasoning`'s, field by field.
  * Every tool in `Request.Tools` is offered. `ChatCompletionsBody` takes
    one tool, so the chat route's tools and `tool_choice` are built in the
    provider.

  **Tool choice:**
  * A named tool is sent as before, in each route's form.
  * `ToolChoiceRequired` and `ToolChoiceNone` use each route's documented
    form, and were not measured: `"required"`/`"none"` on responses and
    chat, `any`/`none` on messages, `ANY`/`NONE` on google. The rule is
    gemini's `"none"`'s: `TestLive_OpencodeToolChoices` pins them, and has
    not been run.

  **Capabilities.**
  * Reasoning is `BestEffort`: the chat route sends an effort only when
    the metadata lists it.
  * Continuation is `Unsupported`: the gateway rejects
    `previous_response_id`.
  * The listing never probes, whatever `WithModelProbes` says.
* **Step 3.** The package doc lists the degradations: reasoning per route;
  the messages route's forced tool sent as `auto` while thinking; the
  instructions; the unmeasured tool choices.
* **Step 4, the tests ported.**

  | From `llmprovider` | To `opencode` |
  |---|---|
  | `opencode_test.go` | `opencode_test.go` |
  | `opencode_route_test.go`, less `TestOpencodeBaseURL`, `TestProviderConstants_Distinct` and the other rows of `TestWireShapesProbedOn`, which stay in a new `llmprovider/opencode_gateway_test.go` | `route_test.go` |
  | `opencode_conventions_test.go` | `conventions_test.go` |
  | `opencode_metadata_route_test.go`, less `TestIsUsableOpencodeModel_DeniesSystemone`, which stays | `metadata_route_test.go` |
  | `claude_system_test.go`, `reasoning_replay_test.go`, the OpenCode parts of `thinking_test.go`, `thinking_wire_test.go`, `generatecontent_wire_test.go` (its decode test stays), `metadata_request_test.go`, `provider_test.go` | `wire_shapes_test.go` |
  | the OpenCode parts of `identification_test.go`, `identification_options_test.go`, `keyless_test.go`, `api_error_message_test.go`, `interface_test.go` | `identification_test.go` |
  | the OpenCode rows of `discovery_wiring_test.go`, `probe_scope_test.go` | `listing_test.go` |
  | the seven opencode G-wire cases | `wire_test.go` |

  * **Changed kind.** `TestOpencode_NoContinuer` asserted that the type
    lacked `Continuer`. It is `TestOpencode_NoContinuation`, which asserts
    `Continuation = Unsupported`, and a continuation refused with
    `ErrUnsupported` before any request (0015-MADR D4).
  * **No successor.** `TestOpencode_ConstructorErrors`'s unknown-gateway
    case: each constructor names its gateway.
  * **Half deferred, by the amendment's accepted gap.** The opencode row of
    `TestDiscoverModels_HonoursRankingOptions` switched profiles. Its
    metadata-URL half is ported as
    `TestListModels_HonoursTheMetadataURLOption`. Its profile half waits for
    S8b.
  * **Moved helper.** The opencode tests have their own `TestMain`, which
    turns the metadata fetch off, as `llmprovider`'s does.
  * **New tests:**
    * `request_test.go`: the refusals, the key read per request, each
      route's form of each tool choice with two tools, the output limit per
      route;
    * `TestOpencode_RequestModelPicksItsRoute`;
    * `TestGenerate_InstructionsAreALeadingSystemItem`;
    * `TestOpencode_TokenHeaderOverride`, on every route;
    * `TestListModels_NeverProbes`;
    * `TestListModels_CarriesTheCallersIdentity`.
  * **Live.** Every live test that built through `NewOpencode` is in the
    external `live_opencode_test.go`. `live_opencode_route_test.go`,
    `live_opencode_conventions_test.go`, `live_reasoning_replay_test.go` and
    `live_system_test.go` were OpenCode-only and are gone.
    `live_export_test.go` exports `LiveModel` and `LiveOpencodeKey`.
* **G-wire.** All 42 opencode goldens (seven cases, six scenarios each) are
  byte-identical to `3d4aff5`'s, with no `-update`, as is
  `opencode-routes.json`. There is no continuation golden: the `P7` type had
  no `Continue`.
* **Step 5, llmtest.** `TestConformance` passes on Zen's chat route and Go's
  messages route.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | `WithRoute` does not pin | `path = "/messages", want the override's /chat/completions` |
  | the metadata route ignored | `path = "/chat/completions", want "/messages"` |
  | the request's model does not pick the route | `path = "/responses", want claude-sonnet-5's /messages` |
  | the messages route reads `Authorization` | `Authorization must not be sent on this route`, and `opencode-zen-messages/text.json` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | no key sends no public token | `/messages: x-api-key = "", want "public"` |
  | an OAuth session accepted | `opencode-zen/oauth: err = <nil>, want ErrUnsupported` |
  | continuation claimed | `Continuation = Supported, want Unsupported` |
  | the chat effort sent without the metadata | `reasoning_effort = low, want absent` |
  | interleaved reasoning not replayed | `assistant reasoning_content = … (nil) …, want ["Call the tool." ""]` |
  | messages forces a tool while thinking | `thinking-tool.json`: `tool_choice.type: want "auto", got "tool"` |
  | google drops a required choice | `"required": tool choice = <nil>, want map[functionCallingConfig:map[mode:ANY]]` |
  | the chat route offers one tool | `tools = [… get_weather …], want both` |
  | instructions dropped | `claude-sonnet-5: system = SECOND, want FIRST before SECOND` |
  | the responses route sends no `store` | `store = <nil> (present false), want false`, and `requests[0].body.store: want false, got (absent)` |
  | no session header | `sessions = ["" "" ""], want 3 non-empty`, and `header.X-Opencode-Session: want "wire-session", got (absent)` |
  | the construction budget ignored | `thinking = map[budget_tokens:4096 type:enabled]` |
  | `Generate` skips the capability check | `R11 (refusal before the network): a request needing continuation … returned <nil>` |
  | `WithRoute` scoped to Zen only | `option opencode.WithRoute is for provider "opencode-zen", not "opencode-go"` |
  | opencode-go registered under another id | `descriptor "opencode-go" is offered to users but neither Default nor the old API builds it` |

* **Lint**, at the first gate, found five things, all fixed:
  * an ignored error from `resolveRoute`: a `tableRoute` helper replaces
    the call;
  * a repeated `"mode"`, which became a constant;
  * the confusing name above;
  * an unused test helper;
  * the old harness's listing variable, now unused.
* **Coverage:**
  * `providers/opencode` 97.2 %;
  * `providers` 80.0 %;
  * `llmprovider` 86.4 % from its own tests and 90.9 % from all of them,
    against its `P7` 89.2 %. See the deviation of 2026-10-01 below.
* **Links.** G-links found no link to the moved files.

### Deviation 2026-10-01: `llmprovider`'s coverage after the opencode move

* **Found** at the opencode commit's gate, a §0 stop condition:
  `llmprovider` 86.4 % (`go test -cover ./llmprovider`), against its `P7`
  89.2 %. It was 89.6 % at `3d4aff5`.
* **Cause.** The opencode tests exercised `llmprovider` code that stays
  there until S7b, because Kilo, Hugging Face and Together share it. They now
  run in `providers/opencode`. Among the functions that drop from 100 % to
  0 % on `llmprovider`'s own tests: `AddMessagesThinking`,
  `GeminiThinkingConfig`, `GeminiSystemInstruction`, `SystemPrompt`,
  `ExpireSession`, and the metadata view (`LookupModelMetadata`,
  `ReasoningEfforts`, `InterleavedField`, `NPM`).
* **Measured both ways:** 86.4 % from `llmprovider`'s tests; 90.9 % with
  `-coverpkg=./llmprovider` over `./llmprovider/...`.
* **Options put to the owner:**
  1. count every package's tests, until S7b;
  2. add tests inside `llmprovider` for the shared code.

  A lower floor was not offered. The owner chose option 1, recorded as
  0015-MADR amendment "how `llmprovider`'s coverage is measured during S7".
* **Effect.**
  * §0's stop condition reads the floor that way for `llmprovider`, until
    S7b.
  * S12 step 2's `coverage-check` measures it so, if S12 runs before S7b
    has finished.
  * R47 in the standards guide notes it.
  * Each later S7 record gives both figures.

*Annotated 2026-10-01, the commit-8 record above:* it says the opencode move
"waits unstaged". The owner committed both together, as `4adecf3`.

### Phase S7, commit 10: `kilo` (2026-10-01)

Built on the earlier commits' pattern; only what differs is recorded here. It
is left unstaged, for the owner to commit after the record changes staged
before it. *Annotated 2026-10-01:* the owner committed both together, as
`c7decee`.

* **Rebuilt tools.** The session's scratchpad had been cleared, and with it
  the gate and its helpers. The gate, the link check, the identifier scan,
  the import dropper and the break runner were rebuilt from these records.
  The link check and the identifier scan were each seen to fail on planted
  input before use. The floor check (`-coverpkg=./llmprovider`, the 2026-10-01
  amendment) failed with its floor set to 99 %.
* **Step 1, moved** (plain `mv`):
  * `kilo.go` to `providers/kilo/`;
  * `kilo_test.go`, `kilo_data_collection_test.go`,
    `kilo_data_collection_allow_test.go`, `kilo_organization_test.go` and
    `kilo_endpoints_test.go`, ported into `providers/kilo/`;
  * `testdata/wire/kilo`.

  **Kept in `llmprovider`:**
  * In a new `kilo_endpoints.go`, what the listing uses:
    * `kiloBaseURL`, the URL-prefixed token pattern, the organization
      header;
    * `resolveKiloEndpoints` with its helpers;
    * `wireShapesProbedOnKilo`, which the raw-HTTP live probes in
      `live_gateways_test.go` still check.
  * `kilo_device.go`, the device login, which `wizard` uses. It moves to
    `auth` in S7b.
  * The catalog's curation and its tests, the classification test, and
    `WithKiloOrganization`, which `wizard` passes to the old catalog
    functions. The catalog and classification tests are in a new
    `kilo_catalog_test.go`.
  * `listKiloModels`, which only the old type called, is removed. Its
    documentation of the catalog's two traps stays.
* **Temporary export,** for S8b's `catalog`: `KiloGatewayFor(baseURL, token,
  org)`, the generation base and organization `resolveKiloEndpoints`
  derives.
* **New accessor:** `Settings.ClientName()`, the name `WithClientInfo` gives,
  which Kilo sends as its editor name. `WithClientInfo` was already common.
* **Step 2, the new API:**
  * `kilo.New` and `ListModels`;
  * `kilo.WithOrganization`, `kilo.WithCapabilities` and
    `kilo.WithDataCollection`, scoped options.

  The old `WithKiloCapabilities` and `WithKiloDataCollection` lose their
  only reader. Their docs say so, as `WithStore`'s does; S8 removes them.

  **Credentials.**
  * With no credential, or an empty key, Kilo's `anonymous` token is sent,
    as `NewKilo("")` did.
  * An OAuth session or a CLI login is refused (R16). A Kilo device login
    yields a token, which is a key.

  **The endpoints are resolved per request,** from the token read for that
  request. `NewKilo` resolved them once, from the key string. A source can
  only be read at request time, since `New` makes no call. A source whose
  URL-prefixed token changes therefore moves its requests
  (`TestKilo_RotatingTokenPicksItsBase`).

  **Request fields.**
  * Every tool in `Request.Tools` is offered, as the old code offered its
    one.
  * A tool choice is sent only when the model accepts `tool_choice`, as
    before: a named tool as before, `required` and `none` as strings (not
    measured; `TestLive_KiloToolChoices`, not run).
  * `Instructions` go in a leading system message.
  * A request's `Reasoning` falls back to `WithReasoning`'s effort.

  **Capabilities.** Forced tool choice and reasoning are `BestEffort`, since
  `WithCapabilities` may withhold them. Continuation is `Unsupported`.
* **Step 4, the tests ported.**

  | From `llmprovider` | To `kilo` |
  |---|---|
  | `kilo_test.go`, less its catalog tests | `kilo_test.go` |
  | `kilo_endpoints_test.go`, `kilo_organization_test.go`, `kilo_data_collection*_test.go` (less the classification test) | `endpoints_test.go` |
  | `identification_options_test.go` and `keyless_test.go` (now Kilo's only), and the Kilo parts of `identification_test.go` (`TestIdentification_KiloHeaders`, the Kilo rows of `…_UserAgent` and of `…_NoForbiddenHeaders`, which had only that row left), `api_error_message_test.go`, `interface_test.go` | `identification_test.go` |
  | the Kilo rows of `discovery_wiring_test.go`, `probe_scope_test.go` | `listing_test.go` |
  | the kilo G-wire case | `wire_test.go` |

  * **Changed kind.** `TestKilo_NoContinuer` is `TestKilo_NoContinuation`:
    `Continuation = Unsupported`, refused with `ErrUnsupported` before any
    request.
  * **Run both ways.** `TestKilo_ReasoningEffortConfigured` runs with the
    effort at construction and on the request.
  * **Deferred, by the accepted profile gap.** The kilo row of
    `TestDiscoverModels_HonoursRankingOptions` switched profiles, and only
    that. It waits for S8b.
  * **New tests:**
    * `request_test.go`: the refusals, Kilo's options, the key per request,
      the request fields, each tool choice with and without `tool_choice`,
      reasoning's fallbacks;
    * `TestKilo_RotatingTokenPicksItsBase`;
    * `TestKilo_TokenHeaderOverride`;
    * `TestListModels_NeverProbes` and
      `TestListModels_CarriesTheCallersIdentity`;
    * in `llmprovider`, `TestKiloGatewayFor`.
  * **Live.** `live_kilo_test.go` is external:
    * the Kilo tests of `live_gateways_test.go` that build a provider;
    * `live_kilo_data_collection_test.go`, which now reads `Code`, not the
      deprecated `Type`;
    * the kilo row of the tool round trip;
    * the new `TestLive_KiloToolChoices`.

    `live_export_test.go` exports `LiveKiloKey`, `LiveKiloFreeCollecting`
    and `LiveKiloNonTraining`.
* **G-wire.** The six kilo goldens are byte-identical to `HEAD`'s, with no
  `-update`. There is no continuation golden.
* **Step 5, llmtest.** `TestConformance` passes.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | `WithCapabilities` ignored | `tool_choice present = true, want false`; `reasoning = map[enabled:true], want absent` |
  | `tool_choice` sent whatever the model accepts | `tool_choice present = true, want false` |
  | data collection always denied | `provider = map[data_collection:deny], want absent` |
  | no key sends no anonymous token | `no credential: Authorization = "Bearer", want "Bearer anonymous"` |
  | an OAuth session accepted | `oauth: err = <nil>, want ErrUnsupported` |
  | generation sends no organization | `TestKilo_OrganizationOption` and `TestKilo_URLTokenOrganization`: the requests differ |
  | the listing drops the organization | `TestKilo_OrganizationOption`: the listing URL differs |
  | the endpoints ignore the token | `TestKilo_URLTokenSelectsBase`: the requests differ |
  | no editor header | `editor/task = ""/"s-1", want pcm/s-1` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | instructions dropped | `messages = [… Be brief. …], want the instructions as a leading system message` |
  | only the first tool offered | `"auto": 1 tools sent, want 2` |
  | the construction effort ignored | `reasoning = map[enabled:true], want map[effort:low]` |
  | `Generate` skips the capability check | `R11 (refusal before the network): … returned <nil>` |
  | `KiloGatewayFor` drops the organization | `KiloGatewayFor(…, "org-1") = …, ""; want …, "org-1"` |
  | kilo registered under another id | `descriptor "kilo" is offered to users but neither Default nor the old API builds it` |

* **Lint** found one doc comment not of the form "WithDataCollection …",
  which was reworded.
* **Coverage:**
  * `providers/kilo` 95.3 %;
  * `llmprovider` 86.0 % from its own tests and 90.8 % over
    `./llmprovider/...`, the measure the 2026-10-01 amendment sets, against
    its `P7` 89.2 %.
* **Links.** G-links found no link to the moved files.

### Phase S7, commit 11: `huggingface` (2026-10-01)

Built on the earlier commits' pattern; only what differs is recorded here.
Staged for the owner to commit. *Annotated 2026-10-01:* committed as
`bea2baa`.

* **Step 1, moved** (plain `mv`): `huggingface.go` to `providers/huggingface/`,
  its tests ported there, and `testdata/wire/huggingface`.

  **Kept in `llmprovider`:**
  * In a new `huggingface_gateway.go`: `huggingFaceBaseURL`, which the
    descriptors and the listing use, and `wireShapesProbedOnHuggingFace`,
    which the raw-HTTP live probes check.
  * The catalog tests, `TestSplitHuggingFaceModelPolicy` and
    `TestStaticHuggingFace_Count`, in a new `huggingface_catalog_test.go`.
  * `listHuggingFaceModels`, which only the old type called, is removed. Its
    documentation of the measured ranking moved onto
    `fetchHuggingFaceUsable`.

  No temporary export is added.
* **Step 2, the new API:**
  * `huggingface.New` and `ListModels`.
  * No credential, or an empty one, is refused with `ErrInvalidRequest`. The
    old code refused an empty key with a plain error. An OAuth session or a
    CLI login is refused with `ErrUnsupported` (R16).
  * The key is read on each request, and sent by `SetTokenHeader`, as the
    listing's is (`fetchHuggingFaceUsable` takes the `Token`).
  * Reasoning sends `reasoning_effort`, medium with no effort, as before;
    reasoning is `BestEffort`, since the router documents it as
    model-dependent. A budget is not sent.
  * Every tool is offered. A named tool is forced as before. `required` and
    `none` are sent as strings, not measured
    (`TestLive_HuggingFaceToolChoices`, not run).
  * `Instructions` go in a leading system message.
  * The listing gets the caller's options, so `WithModelMetadataURL`
    reaches its ranking.
* **Step 4, the tests ported.**

  | From `llmprovider` | To `huggingface` |
  |---|---|
  | `huggingface_test.go`, less its catalog tests, and the Hugging Face parts of `api_error_message_test.go`, `identification_test.go`, `interface_test.go` | `huggingface_test.go` |
  | the Hugging Face rows of `discovery_wiring_test.go` and `probe_scope_test.go` | `listing_test.go` |
  | the huggingface G-wire case | `wire_test.go` |

  * **Emptied, so removed:**
    * `probe_scope_test.go`, whose metered-services test had only this
      row left;
    * `TestDiscoverModels_HonoursRankingOptions`, whose last row this was,
      with its helper `getOnly`.
  * **Half ported.** That row's metadata-URL half is
    `TestListModels_HonoursTheMetadataURLOption`. Its profile half waits for
    S8b, under the accepted gap.
  * **Changed kind.** `TestHuggingFace_NoContinuer` is
    `TestHuggingFace_NoContinuation`.
  * **Run both ways.** The effort test runs at construction and on the
    request.
  * **New tests:**
    * `request_test.go`: the refusals, the key per request, the request
      fields, each tool choice with two tools, reasoning's fallbacks;
    * `TestHuggingFace_TokenHeaderOverride`;
    * `TestListModels_NeverProbes` and
      `TestListModels_CarriesTheCallersIdentity`.
  * **`TestMain`.** The package has its own, turning the metadata fetch off.
  * **Live.** The external `live_huggingface_test.go` has
    `TestLive_HuggingFaceChatCompletions` and the new
    `TestLive_HuggingFaceToolChoices`. The raw router probes stay in
    `live_gateways_test.go`.
* **G-wire.** The six huggingface goldens are byte-identical to `HEAD`'s,
  with no `-update`.
* **Step 5, llmtest.** `TestConformance` passes.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an empty key accepted | `empty: err = <nil>, want ErrInvalidRequest` |
  | an OAuth session accepted | `oauth: err = <nil>, want ErrUnsupported` |
  | no default effort | `reasoning_effort = <nil>, want medium`, and `thinking.json` |
  | the construction effort ignored | `reasoning_effort = medium, want xhigh` |
  | a forced tool sends no `tool_choice` | `tool_choice missing`, and `tool.json` |
  | `ToolChoiceNone` omitted | `"none": tool_choice = <nil>, want none` |
  | only the first tool offered | `"auto": 1 tools sent, want 2` |
  | instructions dropped | `messages = [… Be brief. …], want the instructions as a leading system message` |
  | `MaxOutputTokens` ignored | `max_tokens 500; want other/model, 77` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | the listing drops the caller's options | `ListModels = [a/large c/flash], want the catalog's [c/flash a/large]`, and `the environment's metadata URL was fetched 1 times, want 0` |
  | `Generate` skips the capability check | `R11 (refusal before the network): … returned <nil>` |
  | huggingface registered under another id | `descriptor "huggingface" is offered to users but neither Default nor the old API builds it` |

* **Lint** found `getOnly` unused once its test had gone. It was removed.
* **Coverage:**
  * `providers/huggingface` 93.7 %;
  * `llmprovider` 85.9 % from its own tests and 90.8 % over
    `./llmprovider/...`, against its `P7` 89.2 %.
* **Links.** G-links found no link to the moved files.

### Phase S7, commit 12: `ollama` (2026-10-01)

Built on the earlier commits' pattern; only what differs is recorded here.
Staged for the owner to commit. *Annotated 2026-10-01:* committed as
`20b6b9e`.

* **Step 1, moved** (plain `mv`): `ollama.go` and `ollama_test.go` to
  `providers/ollama/`, and `testdata/wire/ollama`.

  **Kept in `llmprovider`**, in a new `ollama_endpoint.go`:
  * `ollamaBaseURL`, which the descriptor uses;
  * `wireShapesProbedOnOllama`, which `TestWireShapesProbedOn` checks.

  **The listing** stays in `llmprovider` until S8b:
  * `ollamaCatalog` and `fetchOllamaNames` now take the `Token`.
  * `listOllamaModels` is removed: only the old type called it.

  No temporary export is added.
* **Step 2, the new API:**
  * `ollama.New` and `ListModels`.
  * **No credential is needed, as before.** No source, or an empty key, is
    accepted. No `Authorization` header is sent, with or without a key.
  * **R16.** Ollama has no header of its own. A token naming a `Header` is
    sent there, by `SetTokenHeader`, in generation and in the listing. An
    OAuth session or a CLI login is refused with `ErrUnsupported`.
  * **Capabilities.**
    * Forced tool choice is `BestEffort` (0015-MADR D4's own example):
      Ollama does not support `tool_choice`. Every tool is offered, and no
      tool choice is sent for any `ToolChoice`, as Kilo does for models
      without `tool_choice`.
    * Reasoning is `BestEffort`: only thinking models reason.
    * Continuation is `Unsupported`.
  * **Reasoning** sends `reasoning_effort`, medium with no effort, as before.
    `EffortXHigh` is still sent as `max`. A budget is not sent.
  * `Instructions` go in a leading system message.
  * **Listing.**
    * The listing gets the caller's options.
    * Probes are on by default (0016-MADR A5). Each probe is the old probe
      provider's request: `max_tokens` 8192, no reasoning.
    * A failed listing still returns its error, as there is no static
      catalog.
* **Step 4, the tests ported.**

  | From `llmprovider` | To `ollama` |
  |---|---|
  | `ollama_test.go`, and the Ollama parts of `api_error_message_test.go`, `identification_test.go`, `interface_test.go` | `ollama_test.go` |
  | the Ollama row of `discovery_wiring_test.go` | `listing_test.go` |
  | the ollama G-wire case | `wire_test.go` |

  * **Emptied, so removed.** Each had only Ollama left:
    * `interface_test.go`;
    * `api_error_message_test.go`;
    * `discovery_wiring_test.go`.
  * **Narrowed.** `TestIdentification_UserAgent` keeps its listing check;
    generation's is in each provider's package.
  * **Renamed:**
    * `TestOllama_EmptyKeyAccepted` is `TestNew_NeedsNoKey`, which covers
      no key as well;
    * `TestOllama_NoContinuer` is `TestOllama_NoContinuation`.
  * **Run both ways.** The effort clamp runs at construction and on the
    request.
  * **New tests:**
    * `request_test.go`: the refusals, the request fields, each tool choice
      with two tools, reasoning's fallbacks;
    * `TestOllama_TokenHeaderOverride`;
    * in `listing_test.go`: probes and the option, the probe's request, a
      probe failure, the listing's error and an empty install, the
      caller's identity.
  * **Live.** No Ollama live test exists, and none is added: nothing this
    commit sends is unmeasured. `wireShapesProbedOnOllama` records the
    measurement.
* **G-wire.** The six ollama goldens are byte-identical to `HEAD`'s, with
  no `-update`.
* **Step 5, llmtest.** `TestConformance` passes.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | a token with no `Header` sent as a bearer | `Authorization must never be sent to Ollama, got "Bearer ignored-by-ollama"` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | no key refused | `none: New must succeed: llm: invalid request` |
  | an OAuth session accepted | `oauth: err = <nil>, want ErrUnsupported` |
  | a forced tool sends `tool_choice` | `tool_choice must NEVER be sent`, and `tool.json` |
  | only the first tool offered | `"auto": 1 tools sent, want 2` |
  | `xhigh` not clamped | `configured "xhigh" -> reasoning_effort xhigh, want "max"` |
  | no default effort | `budget alone: reasoning_effort = <nil>, want medium` |
  | the construction effort ignored | `at construction: configured "low" -> reasoning_effort medium, want "low"` |
  | instructions dropped | `messages = [… Be brief. …], want the instructions as a leading system message` |
  | `MaxOutputTokens` ignored | `max_tokens 500; want other-model, 77` |
  | listing never probes | `0 generation requests, want 2` |
  | probes ignore `WithModelProbes(false)` | `2 generation requests, want 0` |
  | a probe keeps the caller's limit and reasoning | `probes sent [… "max_tokens":500 … "reasoning_effort":"high"], want one …8192…` |
  | a failed listing swallowed | `404: ListModels = []/<nil>, want an error` |
  | the listing drops the caller's options | `want one GET /api/tags naming wire-app/9.9.9` |
  | `Generate` skips the capability check | `R11 (refusal before the network): … returned <nil>` |
  | ollama registered under another id | `descriptor "ollama" is offered to users but neither Default nor the old API builds it` |

  The "generation ignores the token's header" break did not compile as
  first written (`declared and not used: token`). It was rewritten to
  discard the token, and ran again.
* **Lint** found two helpers unused once the old type had gone:
  * `clientIdentity.options`, which only the old probe called;
  * `captureServer` in `thinking_test.go`, which only `ollama_test.go`
    called.

  Both were removed. `thinking_test.go` held nothing else, so it went too.
* **Pre-add.** `make pre-add-check` lists every tracked Go file, so it
  failed on the removed files until their removal was staged.
* **Coverage:**
  * `providers/ollama` 95.3 %;
  * `llmprovider` 85.8 % from its own tests and 90.8 % over
    `./llmprovider/...`, against its `P7` 89.2 %.
* **Links.** G-links found no link to the moved files.
* **Deviation, 2026-10-01: links to a removed file.**
  * **Found.** With `thinking_test.go` removed, G-links found 3 broken
    links in `0003-PLAN-add-grok-xai-llm-provider.md`, at lines 525, 685
    and 1031, two with line anchors. The openai commit's link rule repoints
    a moved file; a removed one has no new path.
  * **Decided.** The owner chose "unlink, keep the citation". Each link is
    now plain code text, such as `thinking_test.go:14-26`, with a note that
    this plan removed the file. `0004-PLAN-add-gateway-llm-providers.md`
    already cites the same lines that way. The rationale around them is
    unchanged.
  * **Rule.** This applies to every file S7 removes, as the openai rule
    applies to every file it moves.
  * **Scope.** `0003-PLAN-add-grok-xai-llm-provider.md` joins this commit.
    G-links then found 0 problems.
* **Next.** `notYetMoved` holds only `together`.

### Phase S7, commit 13: the proxy test over `Default()` (2026-10-01)

Staged for the owner to commit, before the together commit. *Annotated
2026-10-01:* committed as `98a3f9d`.

* **Deviation, 2026-10-01: the proxy test lost every moved provider.**
  * **Found** in the together commit's survey, before any change.
    `TestDefaultClient_HonoursProxy` (0016-PLAN T1 step 2) iterated
    `llmprovider`'s G-wire cases only
    (`llmprovider/transport_defaults_test.go:29-66` at `20b6b9e`).
  * Each S7 commit moved its provider's case to the provider's package and
    gave the proxy test no replacement. From `00fe139` (openai) on, the test
    covered fewer providers, and none of the S7 records say so. At
    `20b6b9e` it covered only `together`. After the together commit it
    would have covered nothing, and passed.
  * 0016-PLAN records V7, proxy honoured, as met on that test.
  * **Decided.** The owner chose "one test over `Default()`", in its own
    commit before the together commit.
* **The test.** `llmprovider/providers/proxy_test.go` has
  `TestDefaultClient_HonoursProxy`:
  * every provider in `Default()` is built with `providers.New`, a key, its
    first static model and `http://<id>.invalid`, and must generate "hello"
    through `HTTP_PROXY`;
  * the proxy answers with `wirecase`'s canned reply for the request's path,
    which serves every wire format;
  * the parent checks that every id's host reached the proxy, and fails if
    `Default()` is empty;
  * the child process and the cleared proxy variables are as before.

  It covers the nine ids registered at `20b6b9e`, and each provider
  registered later with no further change.
* **The old test stays** in `llmprovider` for `together`, its last case. Its
  doc comment names the new one. The together commit removes it with its
  last case.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | the default client ignores `HTTP_PROXY` | `openai: GenerateText = "", Post "http://openai.invalid/responses": dial tcp: lookup openai.invalid: no such host; want "hello" through the proxy`, and every other id |
  | ollama builds a client without a proxy | `ollama: GenerateText = "", Post "http://ollama.invalid/v1/chat/completions": … no such host` |
  | kilo builds a client without a proxy | `kilo: GenerateText = "", Post "http://kilo.invalid/chat/completions": … no such host` |
  | the child skips the first provider | `openai: no request reached the proxy (saw [claude.invalid … ollama.invalid])` |
  | `Default()` holds no provider | `Default holds no provider` |

  The fourth shows the other eight hosts reached the proxy.
* **Scope.** `llmprovider/providers/proxy_test.go` (new),
  `llmprovider/transport_defaults_test.go` (a comment), and these records.
  No library code changes.

### Phase S7, commit 14: `together`, the last provider moved (2026-10-01)

Built on the earlier commits' pattern; only what differs is recorded here.
Staged for the owner to commit.

* **Step 1, moved** (plain `mv`): `together.go` and `together_test.go` to
  `providers/together/`, and `testdata/wire/together`.

  **Kept in `llmprovider`:**
  * `togetherBaseURL`, in a new `together_endpoint.go`, which the
    descriptor and the listing use.
  * **The listing,** until S8b. `fetchTogetherUsable` now takes the `Token`
    and sends it by `SetTokenHeader`. `listTogetherModels` is removed: only
    the old type called it. `modelCatalogFor` no longer reads the token's
    value for the old providers, as there are none.

  No temporary export is added.
* **Step 2, the new API (0017-MADR D1 as decided):**
  * `together.New` and `ListModels`.
  * No credential, or an empty one, is refused with `ErrInvalidRequest`. The
    old code refused an empty key with a plain error. An OAuth session or a
    CLI login is refused with `ErrUnsupported`.
  * The key is read on each request.
  * **Capabilities,** as D1 says: tools, forced tool choice and reasoning are
    `BestEffort`, continuation `Unsupported`.
  * **Reasoning.** A request with reasoning, or a provider built with
    `WithReasoning`, sends `reasoning {"enabled": true}`, and
    `reasoning_effort` only when an effort is set. With none, neither is sent,
    as before. A budget is not sent.
  * **Tools.** Every tool is offered, and a named tool is forced, as before.
    `required` and `none` are sent as strings, not measured
    (`TestLive_TogetherToolChoices`, not run).
  * `Instructions` go in a leading system message.
  * **Listing.** The listing gets the caller's options, so
    `WithModelMetadataURL` reaches its ranking. It never probes. A failure the
    shared listing returns, a token failure, falls back to the static catalog,
    as `DiscoverModels` never failed.
* **Step 4, the tests ported.**

  | From `llmprovider` | To `together` |
  |---|---|
  | `together_test.go` | `together_test.go` |
  | `assertThinkingFields`, from `thinking_wire_test.go` | `together_test.go` |
  | the together G-wire case | `wire_test.go` |

  * **Run both ways.** `TestTogether_RequestShapes` runs each row with the
    effort at construction and on the request. Its row "effort set, plain
    path" is kept: an effort is part of `Reasoning` now, so a request with no
    `Reasoning` carries none.
  * **Renamed.** `TestTogether_RequiresKey` is `TestNew_NeedsAKey`, which
    covers no key as well.
  * **New tests:**
    * in `together_test.go`: continuation refused, the service's message,
      the User-Agent, the capabilities;
    * `request_test.go`: the refusals, the key per request, the request
      fields, each tool choice with two tools, reasoning's fallbacks;
    * `TestTogether_TokenHeaderOverride`;
    * `listing_test.go`: no probe, the 10 s bound, the metadata URL option
      (less a profile, under the accepted gap), the caller's identity, and
      the fallback on a token failure.
  * **Live.** The generation tests are external, in
    `live_together_test.go`, through `together.New`, with the new
    `TestLive_TogetherToolChoices`. `TestLive_TogetherListing` reads the
    listing directly and stays internal, in a new
    `live_together_listing_test.go`, with `togetherLiveKey`, which
    `live_export_test.go` exports as `LiveTogetherKey`. None has run.
* **Emptied, so removed:**
  * `wire_golden_test.go`, `llmprovider`'s own G-wire harness, whose last
    case this was. Every case now runs through `internal/wirecase` in its
    provider's package. `llmprovider/testdata/wire` is gone with it.
  * `TestDefaultClient_HonoursProxy` in `transport_defaults_test.go`, whose
    last case this was. The test over `Default()` (commit 13) covers
    `together` now. `transport_defaults_test.go` keeps
    `TestShareHTTPClient_KeepsTheSessionsOwn`.
  * `thinking_wire_test.go`, which held only `assertThinkingFields`.
* **`notYetMoved` is empty, and removed** (the plan's "The table is empty
  when S7 ends"). `TestDescriptors_EveryDescriptorIsConstructible` now
  requires every descriptor in `Default()`, and builds each through
  `providers.New`. All ten pass.
* **G-wire.** The six together goldens are byte-identical to `HEAD`'s, with
  no `-update`. G-wire still has 16 cases and 100 goldens, all in provider
  packages.
* **Step 5, llmtest.** `TestConformance` passes.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an empty key accepted | `empty: err = <nil>, want ErrInvalidRequest` |
  | an OAuth session accepted | `oauth: err = <nil>, want ErrUnsupported` |
  | reasoning sends no toggle | `request wins: reasoning = <nil>, want {enabled: true}` |
  | an effort sent unasked | `budget alone: reasoning_effort = medium (sent true), want <nil>`, and `reasoning_effort = medium, want it absent` |
  | the plain path reasons | `reasoning = map[enabled:true], want it absent` |
  | the construction effort ignored | `empty takes the default: reasoning_effort = <nil> (sent false), want low` |
  | a forced tool sends no `tool_choice` | `tool_choice present = false, want true` |
  | `ToolChoiceNone` omitted | `"none": tool_choice = <nil>, want none` |
  | only the first tool offered | `"auto": 1 tools sent, want 2` |
  | instructions dropped | `messages = [… Be brief. …], want the instructions as a leading system message` |
  | `MaxOutputTokens` ignored | `max_tokens 500; want other/model, 77` |
  | the key read once | `Authorization sent [Bearer key-1 Bearer key-1], want [Bearer key-1 Bearer key-2]` |
  | generation ignores the token's header | `POST: X-Custom = "", want "Bearer v"` |
  | the listing ignores the token's header | `GET: X-Custom = "", want "Bearer v"` |
  | the listing keeps non-chat models | `ListModels = [… BAAI/bge-large-en-v1.5 black-forest-labs/FLUX.2 …]`, and `ListModels = [x], <nil>; want the static catalog` |
  | no static fallback | `ListModels = [], <nil>; want the static catalog […]` |
  | the listing drops the caller's options | `the environment's metadata URL was fetched 1 times, want 0` |
  | `Generate` skips the capability check | `R11 (refusal before the network): … returned <nil>` |
  | together registered under another id | `descriptor "together" is offered to users but Default does not build it` |
  | together builds a client without a proxy | `together: GenerateText = "", Post "http://together.invalid/chat/completions": … no such host` |

  "No static fallback" first reached no assertion: the shared listing
  already falls back for a bad body, so the fallback runs only on a token
  failure. `TestListModels_FallsBackWhenTheKeyFails` was added, and the break
  ran again.
* **Lint** found three helpers that only the old providers called, unused
  once the last had gone: `boundedListing`, `firstFunctionCallArgs` and
  `clientIdentity.apply`. All three were removed.
* **Docs.** `architecture.md`: the tree and the package table list
  `together`; construction, the token rule and `providers` no longer
  describe providers still to move; G-wire is described in the provider
  packages. `providers.go`'s package doc says the same. No guide row
  changes: `mcplib` `v1.6.0` had no Together.
* **Coverage:**
  * `providers/together` 95.1 %;
  * `llmprovider` 84.9 % from its own tests and 90.9 % over
    `./llmprovider/...`, against its `P7` 89.2 %.
* **Links.** G-links found no link to a moved or removed file.
* **S7's provider moves are complete.** Every provider is in its own package
  and in `Default()`. The old API's types remain until S8. S7b, which
  removes the temporary exports, is next.

### Amendment 2026-10-01: S7b's import graph (proposed)

* **Found** in S7b's read-only survey, at `d8c6eb1`, before any change.
  The facts, with their file and line, are in 0015-MADR, amendment "S7b's import graph":
  * steps 2 and 3, transport and `auth`, would make `llmprovider` and the
    new package import each other;
  * step 1's `gemini_interactions.go` and step 5's
    `GeminiProvider.interactionsBody` moved with Gemini in S7;
  * the formats share code the step's "one package per format" leaves
    without a home.
* **Decided.** Asked on 2026-10-01, the owner chose:
  * "stdlib-only transport";
  * "after S8b, ChatGPT bits to openai";
  * "stay in `providers/gemini`" for Interactions;
  * "shared root + per format" for the wire.
* **Written from those choices,** for the owner to check:
  * `SetTokenHeader` becomes `Token.Apply`;
  * the format packages' names lose their prefix;
  * the coverage rule now runs until S8c.
* **Changed.** S7b's steps are amended, a new Phase S8c holds `auth`, and the
  0016 table's S3 and S4 rows are annotated. The MADR amendment is
  `proposed`; nothing is implemented until the owner approves it.
* **Approved** 2026-10-01: the owner answered "proceed", including the three
  points written from the choices. The MADR amendment is `accepted`.

### Phase S7b, commit 1: `internal/wire` (2026-10-01)

Under "S7b as amended". Staged for the owner to commit. *Annotated
2026-10-01:* committed with the amendment's records, as `940fee0`.

* **Moved** with plain `mv`, then rewritten in place:

  | From `llmprovider` | To |
  |---|---|
  | `item_convert.go` | `internal/wire/wire.go`, with the shared JSON keys and `LowEffortThinkingBudget` |
  | `chatcompletions.go` | `internal/wire/chatcompletions/chatcompletions.go` |
  | `messages_wire.go`, and the Anthropic half of `thinking_wire.go` | `internal/wire/messages/messages.go` |
  | `generatecontent_wire.go`, and the Gemini half of `thinking_wire.go` | `internal/wire/generatecontent/generatecontent.go` |
  | the Responses half of `http_helpers.go` | `internal/wire/responses/responses.go` (new) |

  * `http_helpers.go` keeps `closeResponseBody`, for the transport commit.
  * `thinking_wire.go` is removed, as both halves moved.
  * `Response.appendOutput` is the function `appendOutput` in `responses`.
  * The keys only one format uses are that package's own constants.
* **Renamed,** as the amended step's table says. The provider packages call
  the new names: `opencode` imports all five packages, and each other
  provider the ones it speaks.
* **`ClassifyStreamFailure`** is the exported `streamFailure`, for good;
  `responses.ReadStream` calls it.
* **Tests moved:**

  | From `llmprovider` | To |
  |---|---|
  | `chatcompletions_test.go`, less `TestClassifyHTTPStatus` | `chatcompletions/chatcompletions_test.go` |
  | `TestClassifyHTTPStatus` | `api_error_test.go`, as it tests `ClassifyHTTPError` |
  | `truncation_test.go`, `truncation_api_test.go` | `chatcompletions` (the two Chat Completions tests) and `responses/responses_test.go` (the two Responses tests); both files removed |
  | `generatecontent_wire_test.go` | `generatecontent/generatecontent_test.go` |
  | `item_signature_test.go` | `generatecontent/signature_test.go`, with its own JSON comparison |
  | `item_fidelity_test.go` | `internal/wire/fidelity_test.go`, package `wire_test`, reading Chat Completions' messages through `chatcompletions.Body` |

  `live_gateways_test.go`'s Kilo probe writes its request body out, as the
  other raw probes do. An in-package test cannot import `chatcompletions`,
  which imports `llmprovider`.
* **Deviation, 2026-10-01: the new packages' own coverage.**
  * **Found.** From their own tests, after the move:
    * `internal/wire` 36.4 %;
    * `chatcompletions` 82.9 %;
    * `generatecontent` 66.7 %;
    * `messages` 0.0 %;
    * `responses` 11.3 %.

    Counting every test under `./llmprovider/...`, they were 81.8 %,
    98.7 %, 95.7 %, 93.9 % and 90.1 %. The provider packages test the wire
    through `Generate`, and none called a wire function directly.
  * **Decided.** The owner chose "write wire unit tests". The other option
    was to count every test, as `llmprovider` does until S8c.
  * **Added:**
    * `wire/wire_test.go`: `ToolArguments`, `SystemPrompt`;
    * `responses/wire_test.go`: `Input`, `Decode`, and `ReadStream`'s
      events, early end and read error;
    * `messages/messages_test.go`: `FromItems`, `Decode`,
      `claudeAdaptiveOnly`, `AddThinking`;
    * `generatecontent/wire_test.go`: `SystemInstruction`, `Contents` for a
      result without its call, `Decode`'s errors, `ThinkingConfig`.
  * **After:** `internal/wire` 100.0 %, `chatcompletions` 82.9 %,
    `generatecontent` 91.3 %, `messages` 96.3 %, `responses` 100.0 %.
* **Breaks** for the new tests, each in a scratch copy:

  | Break | Failure |
  |---|---|
  | `ToolArguments` drops a non-object | `ToolArguments("[\"not\",\"an\",\"object\"]") = map[], want map[arguments:…]` |
  | `SystemPrompt` keeps empty system text | `SystemPrompt = "Be brief.\n\n\n\nAnswer in French."` |
  | `Input` drops the call id | `Input = [… {"arguments":…,"name":"get_weather","type":"function_call"} …]` |
  | `Decode` keeps an empty message | `Decode = &{ID:r1 … {Role:assistant Text:} …}` |
  | `Decode` ignores text-typed content | `output[1] = … Text:"hel"}, want … Text:"hello"}` |
  | no default reason for an incomplete answer | `incomplete with no reason: … response incomplete: , want reason unspecified` |
  | the stream ignores `response.created` | `ReadStream = &{ID: …}, <nil>; want r0` |
  | the stream ignores `response.failed` | `err = … stream ended before response.completed, want llmprovider: context window exceeded` |
  | an early end is a success | `err = <nil>, want the early-end error` |
  | results do not share a user turn | `FromItems = [… {"content":[{"content":"sunny",…}],"role":"user"},{"content":[{"content":"noon",…` |
  | thinking's text fallback dropped | `Decode = [{Text:plan} {Text:} …], want [{plan} {fallback} …]` |
  | Claude 4.7 takes a budget | `claudeAdaptiveOnly("claude-4.7-opus") = false, want true` |
  | low effort keeps the default budget | `low effort: body {"thinking":{"budget_tokens":4096,…}}` |
  | `max_tokens` not raised | `max_tokens raised: … max_tokens 4096; want …, 8192` |
  | `SystemInstruction` never nil | `no system items: map[parts:[map[text:]]], want nil` |
  | a result without its call has no name | `…"functionResponse":{"name":"",…` |
  | legacy Gemini takes a level | `got {"includeThoughts":true,"thinkingLevel":"low"}` |
  | an empty thought kept | `an empty thought: &{… Output:[{Text:} {Role:assistant Text:ok}] …}; want only the text` |
  | a stream overflow is not an overflow | `err = llm: provider unavailable: p stream context_length_exceeded: too long, want llmprovider: context window exceeded`, and `TestContextOverflow_StreamFailure` |

  "Results do not share a user turn" did not compile as first written. It was
  rewritten to stop the user-turn merge, and ran again.
* **Lint** found three constants in `llmprovider/constants.go` unused once
  the wire had moved: `jsonKeyToolCalls`, `geminiRoleModel` and
  `geminiSkipThoughtSignature`. All three were removed.
* **Check.** `go doc -all ./llmprovider` names none of the 15 wire
  identifiers S7 exported. It names `ClassifyStreamFailure`, which shows the
  search finds a name that is there.
* **G-wire.** All 100 goldens unchanged, with no `-update`.
* **Coverage.** `llmprovider` 88.0 % from its own tests and 90.6 % over
  `./llmprovider/...`, against its `P7` 89.2 %.
* **Links.** G-links found one link to a moved file, in
  `0003-PLAN-add-grok-xai-llm-provider.md` (line 57, to
  `generatecontent_wire.go`). It points at the moved file under the openai
  commit's rule; its text is kept.
* **Docs.** `architecture.md` lists the five packages and what each provider
  imports. Its temporary-export paragraph names only what is left.

### Phase S7b, commit 2: `internal/transport` (2026-10-01)

Under "S7b as amended". Staged for the owner to commit.

* **Moved** with plain `mv`, then rewritten in place:
  * `probe.go` to `internal/transport/probe.go`;
  * `probe_test.go` and `timeouts_test.go` to `probe_test.go` and
    `client_test.go` there;
  * `internal/transport/transport.go` is new. It holds `DefaultClient`, the
    identity, `BuildVersions` and the Retry-After parsing. It was first
    `http_helpers.go`, moved, until the body-close deviation below restored
    that file.
* **From other files:**
  * `defaultHTTPClient` (`options.go`) is `DefaultClient`;
  * `clientIdentity`, `buildVersions` and the identity constants
    (`identification.go`) are `Identity`, `BuildVersions`,
    `DefaultClientName` and `DevelVersion`;
  * `parseRetryAfter` and `retryAfterFrom` (`provider.go`) are
    `ParseRetryAfter` and `RetryAfter`.

  `RetryAfter` goes with `ParseRetryAfter`, as both are the step's
  "Retry-After parsing". `BuildVersions` reads through a new `versionsOf`, so
  that a test can give it any build info.
* **Kept in `llmprovider`:** `identityOf`, which builds a
  `transport.Identity` from `ProviderConfig`, and `WithClientInfo` and
  `WithSessionID`.
* **`Token.Apply`** replaces `SetTokenHeader`. `tokenHeader` stays, as
  `discovery.go` builds header pairs with it. `ClassifyHTTPError` loses its
  temporary label. `ShareHTTPClient`'s label now names S8c.
* **`ProbeGenerateHealth` takes its limit.** All five probing providers pass
  `llmprovider.MaxListedModels`: `openai`, `claude`, `gemini`, `grok` and
  `ollama`.
* **Call sites** in `llmprovider` and the provider packages were rewritten
  by script. The test hook `withSDKVersion` swaps
  `transport.BuildVersions`.
* **Tests moved:**
  * `TestParseRetryAfter` (`provider_correctness_test.go`) and
    `TestParseRetryAfter_HTTPDate` (`provider_test.go`) to
    `transport_test.go`;
  * `TestDefaultHTTPClient_Timeouts` as `TestDefaultClient_Timeouts`, which
    also checks the proxy;
  * `TestProbeGenerateHealth`, with a limit.

  `TestParseRetryAfter_MillisAndFractional` stays, as it goes through the
  classifier.
* **New tests:**
  * in `transport`: `TestRetryAfter`, `TestIdentity_UserAgent`,
    `TestVersionsOf`, `TestProbeGenerateHealth_Limit`;
  * in `ollama`: `TestListModels_ProbesAtMostMaxListedModels`.

  `internal/transport` holds 100.0 % from its own tests.
* **Deviation, 2026-10-01: closing a response body.**
  * **Found.** With `closeResponseBody` moved to `transport.CloseBody`,
    golangci-lint's `bodyclose` flagged its 12 call sites in `llmprovider`
    (`discovery.go`, `kilo_device.go`, `model_metadata.go`,
    `oauth_idtoken.go`). It recognises a `Body.Close()` only in the package
    that holds the response. The provider packages already close inline.
  * **Decided.** The owner chose "keep the helper in `llmprovider`". The
    other option was to inline the close at all 12 sites. Suppressing the
    linter was not offered.
  * **Changed.** `llmprovider/http_helpers.go` is restored as it was at
    `940fee0`, and `transport` has no `CloseBody`. The amended step's list
    loses one item. The MADR amendment is annotated: it said `transport`
    would hold "closing a response body".
* **Gap named.** No provider test pinned the probe's limit before this
  commit: the old function read `MaxListedModels` itself. Breaking `grok`'s
  limit to 1 failed no test, as its fake lists one model.
  `TestListModels_ProbesAtMostMaxListedModels` pins `ollama`'s, whose fake is
  the cheapest. The other four pass the same constant, read from the source
  above; no test pins them.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | the default client drops the proxy | `Proxy is nil, want http.ProxyFromEnvironment (0016-MADR D8)` |
  | `retry-after-ms` ignored | `retry-after-ms: 7s, want 1.5005s` |
  | a past date is a negative delay | `a date in the past: got -1h0m0.690362s` |
  | the User-Agent drops this module's name | `UserAgent = "app/1.2.3 (darwin; arm64) llmprovider/(devel)"` |
  | `SetUserAgent` sets another header | `User-Agent header = "", want "app/1.2.3 (darwin; arm64) go-llmprovider-sdk/(devel)"` |
  | this module as the main module reads `(devel)` | `the main module: versionsOf = "(devel)", "v1.5.0"; want "v1.5.0", "v1.5.0"` |
  | a dependency's version ignored | `a dependency: versionsOf = "(devel)", "v2.0.0"; want "v1.4.0", "v2.0.0"` |
  | the probe ignores its limit | `ProbeGenerateHealth = [a b c], want [a b]` |
  | `Token.Apply` ignores the token's header | `X-Custom = "", want "Bearer v"` |
  | `ollama`'s probe passes a smaller limit | `1 probes, want 6` |

  Two breaks were invalid as first written, and ran again:
  * a User-Agent format without the platform did not compile, as `runtime`
    was then unused. It now drops this module's name.
  * `grok`'s limit set to 1 failed no test (the gap above). It ran again on
    `ollama`.
* **Lint** found `kiloEditorHeader` and `kiloTaskHeader` unused. They had
  been unused since Kilo moved, but shared a const group with constants still
  in use, which hid them. Both were removed.
* **Check.** `go doc -all ./llmprovider` names neither `SetTokenHeader` nor
  `ProbeGenerateHealth`, and names `Token.Apply`. No wire or transport
  temporary export is left. The ones left:
  * for S8c, `IsChatGPTSession`, `ChatGPTSessionAccountID`,
    `ChatGPTSessionFedRAMP`, `ExpireSession`, the four ChatGPT header
    constants, and `ShareHTTPClient`;
  * for S8b, `ModelMetadata`, `LookupModelMetadata` and `KiloGatewayFor`.
* **Docs.**
  * `architecture.md` lists `internal/transport`, the token rule names
    `Token.Apply`, and the temporary exports are the ones above.
  * The standards guide's R2 table follows the amendment's import edges, and
    its `internal/wire` and `internal/transport` rows.
* **G-wire.** All 100 goldens unchanged, with no `-update`.
* **Coverage.** `llmprovider` 88.0 % from its own tests and 90.5 % over
  `./llmprovider/...`, against its `P7` 89.2 %.
* **S7b is complete.** The `auth` extraction is Phase S8c, after S8 and S8b.

### Amendment 2026-10-01: S8, and `catalog` before the removals (proposed)

* **Found** in S8's read-only survey at `2939d48`, before any change. The
  facts are in 0015-MADR, amendment "`catalog` before the old API's removal", and:
  * `wizard` uses `ProviderDescriptor`, `Descriptors()` and `DescriptorFor`
    for its menu, and `providers.go` builds every `Descriptor` from
    `DescriptorFor`.
  * Retry reads `RateLimitError` and `Terminal` today. Three wire paths build
    `IncompleteError`. Tests pin "a 429 stays a `*RateLimitError`".
  * Six sentinels read `llm:`, and `ErrInvalidProvider` has no prefix. No
    G-wire golden contains `llm:`.
  * Nothing outside tests uses the old generation API now.
  * About 750 uses of the id constants; several functions take a provider as
    a `string`.
  * 0016 T4 has not started: `Result` copies the session's tokens
    (`wizard/configure.go:142-150`), `keepExistingOAuth` rebuilds a session
    from them, there is no logout, and `Result` has no redaction.
* **Decided.** Asked on 2026-10-01, the owner chose:
  * "do S8b's catalog first";
  * "5 commits as proposed", with `catalog` added as commit 2;
  * "`providers.Default()`" for a nil `Registry`.
* **Written from those choices,** for the owner to check:
  * `catalog`'s two scoped options;
  * `internal/kiloendpoint`;
  * the names table in "S8 as amended".
* **Changed.** S8's steps are carried out as "S8 as amended", and S8b keeps
  step 4. Nothing is implemented until the owner approves.
* **Approved** 2026-10-01: the owner answered "proceed" after committing the
  proposed records as `9259bd9`, including the three points written from the
  choices. The MADR amendment is `accepted`.

### Phase S8, commit 1: descriptors, and the wizard menu (2026-10-01)

Under "S8 as amended". Staged for the owner to commit, with the amendment's
acceptance (its status, and the PLAN's "Approved" line).
*Annotated 2026-10-01:* committed as `5784238`.

* **Descriptors.** Each provider package has a `descriptor.go`, with
  `Descriptor()`, or `DescriptorZen()` and `DescriptorGo()` in `opencode`. It
  holds the values of `llmprovider`'s table at `9259bd9`, built from the
  package's own base URL with `ProviderEnvVars()` and `StaticModels()`.
  `kilo` gains a `defaultBaseURL` for it.
  * **Seen equal.** A scratch-only test compared all ten with the old
    `Descriptors()` through `providers.descriptor`, with `reflect.DeepEqual`,
    and passed. A Kilo note changed in the copy failed it.
* **`providers.Default()`** registers each package's descriptor.
  `providers.descriptor` is removed.
  * **Menu order kept.** `builtins` listed `openai` first, while the wizard's
    menu came from the old `Descriptors()`, which listed `gemini` first.
    `builtins` now follows the old order, so the menu does not change.
    `TestDescriptors_MenuOrder` pins it.
* **`llmprovider`** keeps `AuthMethodID` and `AuthMethod`, which `Descriptor`
  uses, in `descriptor.go`. `ProviderDescriptor`, `descriptorSpecs`,
  `Descriptors()` and `DescriptorFor` are removed.
* **`wizard`:**
  * `Options.Registry` is new; nil is `providers.Default()`, and
    `selectableDescriptors` reads it.
  * `ProviderDescriptor` is `Descriptor` throughout.
  * A descriptor's `ID` is converted with `string(...)` where the wizard
    still holds ids as strings. Commit 5 types them.
* **Tests:**
  * `llmprovider/descriptor_test.go` is removed, its tests ported to
    `providers/descriptors_test.go` over `Default()`.
  * `TestDescriptors_OpenAIAndGrokOfferOAuth` keeps its full `AuthMethod`
    comparison.
  * `DerivedFieldsMatchSource` joins `CoverEveryRegisteredProvider`.
  * `StableOrderAndDefensiveCopy` is `MenuOrder` and `DefensiveCopy`.
  * `TestDescriptorFor` is `TestDescriptor_Lookup`.
  * `TestModelLabel` stays in `llmprovider`, in `model_label_test.go`, until
    commit 2.
  * `TestDescriptors_EveryDescriptorIsConstructible` iterates `Default()`.
  * `opencode`'s `TestOpencodeBaseURLs_MatchTheDescriptors` pins the values
    only, as the descriptors are now built from the same constants.
  * In `wizard`, `providerIdx` and the other three call sites read
    `providers.Default()`.
  * **New:** `TestConfigureLLM_UsesTheRegistry`, a one-provider registry
    as the menu.
* **Lint** found `ollamaBaseURL` unused, as only the old table read it. It
  was removed. The comments of `together_endpoint.go`,
  `huggingface_gateway.go` and `opencode_gateway.go` no longer name the
  descriptors.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | the menu order changes | `menu order = [openai gemini claude …], want [gemini openai claude …]` |
  | two providers leave `Default()` | `provider "together" has a credential variable but no descriptor in Default` |
  | an auth method's detail changes | `openai AuthMethods = … want …` |
  | a descriptor names the wrong variable | `claude: EnvVar = "WRONG_API_KEY", want "CLAUDE_API_KEY"` |
  | a descriptor offers another provider's models | `gemini: StaticModels = [claude-haiku-4-5 …], want StaticModels()'s` |
  | Grok loses browser sign-in | `descriptor "grok" does not offer required auth method "browser_oauth"`, and the `AuthMethods` comparison |
  | the registry shares `AuthMethods` | `a caller's change reached the registry` |
  | the wizard ignores `Options.Registry` | `provider menu = [[{Gemini (Google) } …]], want the registry's one provider` |
  | the provider filter matches nothing | `wizard: none of the requested providers exist: [grok]` |

  Two were invalid as first written, and ran again:
  * removing `together`'s entry did not compile, its import then unused; the
    loop now skips the last two entries.
  * `"CLAUDE_API_KEY"` is Claude's variable today, so it failed no test; it
    is now `"WRONG_API_KEY"`.
* **G-parity** fills the 13 rows of `ProviderDescriptor`, its fields,
  `Descriptors` and `DescriptorFor`: `409 identifiers, … 0 problem(s)`.
* **Coverage:**
  * `llmprovider/providers` 85.7 %;
  * `wizard` 83.5 %;
  * `llmprovider` 87.9 % from its own tests and 90.4 % over
    `./llmprovider/...`, against its `P7` 89.2 %.
* **G-wire** unchanged. `architecture.md` describes the descriptors and the
  wizard's `Registry`.

### Phase S8, commit 2: `catalog` (2026-10-01)

Under "S8 as amended". Staged for the owner to commit.
*Annotated 2026-10-01:* committed as `f44ebc7`.

* **Moved** with `git mv` into `llmprovider/catalog`, then rewritten for the
  package:
  * `models_catalog.go`, `model_ranking.go`, `model_matcher.go`,
    `model_metadata.go`, `model_profile.go` and `discovery.go`;
  * `opencode_gateway.go`, as `gateways.go`, which also takes the Hugging Face
    and Together base URLs of `huggingface_gateway.go` and
    `together_endpoint.go` (both removed).
* **`kilo_endpoints.go`** is `internal/kiloendpoint`: `Resolve`, `Route`,
  `BaseURL`, `OrganizationHeader`. `KiloGatewayFor` is removed; `kilo` and the
  Kilo device login use `Resolve`.
* **The probe dates** of Kilo, Hugging Face and Ollama stay in `llmprovider`,
  in a new `wire_probes.go`; `ollama_endpoint.go` is removed. The raw live
  probes and `TestWireShapesProbedOn` read them.
* **The listing's options** (`catalog/config.go`):
  * `List` resolves its options with `ResolveOptions` for the id it lists,
    and reads them through `Settings`. A `config` holds what one listing
    reads.
  * `WithProfile` is scoped to every built-in id, and `WithKiloOrganization`
    to `kilo`. Both are read back through `Settings.Values()`.
  * `catalog` keeps its own `closeResponseBody`, as commit 2 of S7b decided,
    and builds `tokenHeader` on `Token.Apply`.
* **Renamed** as the amended step's table says. `List` replaces the four
  listing functions and takes a `ProviderID`.
* **Removed from `llmprovider`:**
  * `WithModelProfile`, `WithKiloOrganization`, and their `ProviderConfig`
    fields;
  * the 27 JSON-key constants, unused once the listing had moved.
* **Callers.** The provider packages call `catalog.List`, `catalog.Static`
  and `catalog.MaxListed`.
  * `kilo` passes its organization with `catalog.WithKiloOrganization`.
  * `opencode` uses `catalog.LookupMetadata`.
  * `wizard` lists with `catalog.List`, `catalog.WithProfile` and
    `catalog.WithKiloOrganization`, and `Options.Profile` is a
    `catalog.Profile`.
* **Tests moved** to `catalog`: 21 files whole, and two split.
  * From `opencode_gateway_test.go`, now `llmprovider/wire_probes_test.go`,
    the two OpenCode tests went to `catalog/gateways_test.go`.
  * From `catalog_state_test.go`, the static-copy and `Rank` tests went;
    `ProviderEnvVars`' stays.
  * The ranking fixtures `testdata/ranking-2026-09-26` moved with their test.
  * The listing calls became `List`, or the helper `listRecommended` for what
    `ListAvailableModels` returned.
  * `ApplyOptions` became `testConfig`.
  * `TestKiloGatewayFor` is `kiloendpoint`'s `TestResolve`, with the listing
    URLs, and a new `TestRoute`.
  * **Moved back.** `model_picker_test.go` is the live suite's model picker,
    which reads the metadata document itself. It stays in `llmprovider`,
    with its own copies of the two section keys and the URL.
  * **Live.** `TestLive_ModelMetadataDocument`, `TestLive_GrokListingTextOnly`
    and `TestLive_TogetherListing` moved to `catalog/live_listing_test.go`.
    The other raw probes stay, with literal base URLs and
    `kiloendpoint.BaseURL`. `withSDKVersion` moved to `live_export_test.go`.
    The external live tests call `catalog`.
* **Two rewriter faults,** found and fixed during the move, in a scratch
  tool, never committed: `...` was read as a selector, and a `case` label as
  a composite-literal key. The compiler caught both.
* **New tests:**
  * `TestListModels_HonoursRankingOptions` in `catalog`'s external test
    package is `TestDiscoverModels_HonoursRankingOptions` with its profile
    halves back: the original fixtures, pinned clock and rows (`kilo`,
    `huggingface`, `opencode-zen`), through `providers.New`. The deferral
    of 2026-09-30 ("no profile on the new API until S8b") ends here.
    `export_test.go` exposes the fixtures to it.
  * `TestWithProfile_EveryProviderTakesIt`: every built-in `New` takes
    `WithProfile`, and only `kilo`'s takes `WithKiloOrganization`.
  * `TestListModels_ListsTheOrganization` in `kilo`.
* **Changed assertion.** `kilo_catalog_test.go` read the deprecated
  `APIError.Terminal` and `Type`, which lint flags across packages. They are
  now `!Retryable()` and `Code`: `Code` holds the same value, and
  `!Retryable()` is weaker for this error kind. It is what commit 3 leaves,
  when `Terminal` becomes unexported.
* **Stale comments.** Nine provider structs and three listing tests no
  longer say the listing waits for S8b.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | the listing ignores `WithProfile` | `fixture does not separate the profiles: both [b/flash d/mid …]`, for each row |
  | `WithProfile` leaves out `ollama` | `ollama: New with catalog.WithProfile: … is for providers "gemini", …` |
  | `WithKiloOrganization` scoped to every id | `gemini: New with catalog.WithKiloOrganization: err = <nil>` |
  | the listing ignores the Kilo organization | `listing requests = ["/models org="], want the organization's catalog with its header` |
  | `kilo` does not pass its organization on | the same |
  | the listing ignores the base URL | `unexpected models: …`, `expected error for non-200 ollama response` |
  | the listing sends no User-Agent | `/models: User-Agent = "Go-http-client/1.1"` |
  | `tokenHeader` drops the value | `GET: Authorization = "", want "Bearer v"` (grok) |
  | an organization lists the gateway's models | `Resolve(…, "org-1") = {… Models:http://127.0.0.1:9/models …}` |
  | a token's organization is not read | `Resolve("", "https://kilo.example.test/api/organizations/org-9:secret", "") = …` |
  | `Route` keeps the path after `api` | `Gateway:https://kilo.example.test/api/organizations/org-9/api/gateway` |
  | the wizard drops the profile | `Model = "a/flash", want "b/pro"` |

  Three did not compile as first written, each leaving a variable unused,
  and were rewritten. `tokenHeader`'s break first ran against Hugging Face,
  whose listing calls `Token.Apply` directly, so it failed nothing; it ran
  again against `grok`, whose listing goes through `tokenHeader`.
* **Lint.** `golangci-lint fmt` put the moved files' imports in `goimports`
  order, and the doc comments name the new identifiers.
* **G-parity** fills the 25 rows of the moved identifiers, and the
  `WithKiloOrganization` row names both options: `409 identifiers, 409 rows,
  181 with an SDK equivalent, 0 problem(s)`.
* **G-links** found five links in `0003-PLAN-add-grok-xai-llm-provider.md`
  to the moved files. They point at the moved files, their text kept and
  their line anchors dropped, under the openai commit's rule.
* **Docs.**
  * `architecture.md` lists `catalog` and `internal/kiloendpoint`, and its
    discovery section names `catalog`.
  * The standards guide's R2 table gains `internal/kiloendpoint`.
* **Deviation, 2026-10-01: `llmprovider`'s floor.**
  * **Found.** The first gate failed `cover-floor-llmprovider`: 88.5 % over
    `./llmprovider/...`, against `P7`'s 89.2 %. It was 14 statements short,
    with 1749 of 1976 covered. Moving the listing's well-tested code out left
    the less-tested OAuth and token-store code as the bulk of `llmprovider`.
    No statement lost coverage.
  * **Decided.** The owner chose "test untested code". The other option was
    to fold commit 4's removal of the unread options into this commit too,
    which alone was 5 statements, not 14.
  * **Added**, for behaviour no test had:
    * `TestFileTokenStore_Delete` (`Delete` was at 0 %);
    * `TestFillOAuthEndpoints` (`fillOAuthEndpoints`, 0 %);
    * `TestRefreshTokenURL` (`refreshTokenURL`, 33 %).

    These move with that code to `auth` in S8c.
  * **After:** 1764 of 1976, 89.27 %.
  * **Breaks,** each in a scratch copy:

    | Break | Failure |
    |---|---|
    | `Delete` removes nothing | `Load after Delete = OAuthSession{Provider:openai …}` |
    | `Delete` hides a removal failure | `Delete that cannot remove: err = <nil>, want it returned` |
    | discovery's endpoints overwritten | `filled = {Authorization:b-auth Token:b-token …}, want {… Token:d-token …}` |
    | discovery's algorithms overwritten | `SigningAlgs = [RS256], want discovery's [ES256] kept` |
    | a session's own token URL ignored | `own token URL: refreshTokenURL = "https://auth.openai.com/oauth/token"` |
    | OpenAI's issuer not trimmed | `OpenAI: refreshTokenURL = "https://auth.x.ai/oauth2/token"` |
* **G-wire.** All 100 goldens unchanged, with no `-update`.
* **Coverage:**
  * `catalog` 88.1 %, and `internal/kiloendpoint` 100.0 %;
  * `llmprovider` 87.8 % from its own tests and 89.3 % over
    `./llmprovider/...`, against its `P7` 89.2 %.

### Phase S8, commit 3: errors, D7 and R27 (2026-10-01)

Under "S8 as amended". Staged for the owner to commit, with the amendment of
0015-MADR "`ErrIncomplete` beneath `ErrInvalidRequest`".
*Annotated 2026-10-01:* committed as `61ac6e1`.

* **One structured error.**
  * A 429 that is not exhausted quota is an `*APIError` of kind
    `ErrRateLimited`, with its `RetryAfter`. So is a stream's
    `rate_limit_exceeded` or `slow_down`, with `Status` 0.
  * A cut-short answer is an `*APIError` of kind `ErrIncomplete`, with its
    `Reason`. That covers Responses' `incomplete`
    (`internal/wire/responses`), Chat's `length` with a tool call
    (`internal/wire/chatcompletions`) and Gemini's Interaction `incomplete`.
  * `RateLimitError` and `IncompleteError` are removed, and `truncation.go`
    keeps only `finishReasonLength`.
* **`APIError`.**
  * `Type` is removed; `Code` carries the same value.
  * `Terminal` is the unexported `terminal`, read by `Retryable()`.
  * `Error()` names the retry-after and the reason, which the two removed
    types carried. It reads `<kind>: <provider> HTTP <status> <code>
    (retry-after <d>): <reason>: <message>`, leaving out what is unset.
  * `ClassifyHTTPError`'s doc no longer says a 429 stays a
    `*RateLimitError`.
* **R27.** `ErrRateLimited`, `ErrProviderUnavailable`, `ErrAuthFailure`,
  `ErrInvalidRequest`, `ErrQuotaExhausted`, `ErrNotPermitted` and
  `ErrInvalidProvider` now read `llmprovider:`. So do the three log messages
  of the old retry loop, which commit 4 removes.
  * This ends 0004-PLAN's promise (its D4) that an `Error()` with no provider
    reproduces the original message verbatim. That message read `llm:`.
  * R27 is applied here to the sentinels, as the amended step says. Errors
    wrapped with a provider prefix, such as `gemini: decode interaction:`,
    still start with it. Rate limits from token refresh and the Kilo device
    login stay wrapped sentinels; they move with the auth code in S8c.
* **The old retry loop.** `retryStops` read only `Terminal` for an
  `*APIError`, so as first written it would have retried a cut-short one;
  `IncompleteError` was not an `*APIError`. It now also stops on kind
  `ErrIncomplete`. `serverRetryAfter` reads only `*APIError`.
* **Deviation, 2026-10-01: `ErrIncomplete` and `ErrInvalidRequest`.**
  * **Found.** `IncompleteError` matched both. D7 says `errors.Is` reaches
    one kind sentinel each, and the step named only the kind. The first
    version kept the match with a case in `APIError.Unwrap`.
  * **Decided.** The owner chose "put it beneath": `ErrIncomplete` is a
    child of `ErrInvalidRequest`, as `ErrContextOverflow` is. The other
    options were to keep the `Unwrap` case, or to drop the match, a
    breaking change.
  * **Recorded** in 0015-MADR, amendment "`ErrIncomplete` beneath
    `ErrInvalidRequest`", in D7's list, and in the standards guide's R25.
  * **Effect.** The bare "returned no response" and "returned no call to
    tool" errors now also match `ErrInvalidRequest`. `WithRetry` already
    treated them as final.
* **Tests rewritten to D7's rule:**

  | Test | Was | Now |
  |---|---|---|
  | `TestClassifyHTTPError_PlainRateLimit` | a 429 stays a `*RateLimitError`, not an `*APIError` | an `*APIError` of kind `ErrRateLimited`, retryable, with retry-after, status and message |
  | `TestClassifyHTTPStatus` | a 429 is a `*RateLimitError` | an `*APIError` of kind `ErrRateLimited` |
  | `TestParseRetryAfter_MillisAndFractional` | reads `RateLimitError.RetryAfter` | reads `APIError.RetryAfter` |
  | `TestRetry_RetryAfterAboveCapReturns` | a `*RateLimitError` | an `*APIError` |
  | `TestRateLimitError_Classification` | `RateLimitError` | renamed `TestRateLimit_Classification`, an `*APIError` |
  | `TestRateLimitError_ErrorString`, `TestRateLimitError_ProviderAttribution` | `RateLimitError`'s message | removed; `TestAPIError_ErrorText` pins the message, with and without a provider |
  | `TestWithRetry_RetriesByKind` | rows for `RateLimitError` and `IncompleteError` | a stream rate limit, and a cut-short `*APIError` |
  | `TestIncompleteError_Reason` (`internal/wire/responses`) | `*IncompleteError` | renamed `TestIncomplete_Reason`, an `*APIError` of kind `ErrIncomplete` |
  | `TestDecode` (`internal/wire/responses`), `TestGeminiInteractions_Status`, `TestOpenAIChatGPT_StreamFailures/incomplete` | `*IncompleteError` | an `*APIError` of kind `ErrIncomplete`, with its reason |
  | `TestOpenCode…` 429 row (`opencode_test.go`) | `RateLimitError.RetryAfter` 0 | an `*APIError` of kind `ErrRateLimited`, `RetryAfter` 0 |
  | `TestAPIError_KindAndCodeFromClassification`, `TestClassifyHTTPError` table, `TestClassify_UsageLimitResetsAt` | `Type`, `Terminal` | `Code`, `terminal` (same package) |
  | `TestAPIError_WithoutKind` | `Code` wins over `Type`; `llm:` | no `Type`; `llmprovider:` |
  | `TestSentinels_Beneath` | `IncompleteError` matches both; three sentinels start `llmprovider:` | a cut-short `*APIError` matches both and is not retryable; a stream rate limit matches `ErrRateLimited` alone; `ErrIncomplete` is beneath `ErrInvalidRequest`; all ten sentinels start `llmprovider:` |

* **Tightened,** because neither caught a wrong kind:
  * `TestDecodeChat_LengthToolCallIsError` checked only `ErrInvalidRequest`.
    It now wants an `*APIError` of kind `ErrIncomplete` with reason `length`.
  * `TestOpenAIChatGPT_StreamFailures` read `Retryable()` only if the error
    was an `*APIError`. Its `rate_limit_exceeded` row now requires one.
* **New tests:**
  * `TestAPIError_ErrorText` covers the five shapes: a 429 with provider,
    retry-after and message; one with neither; a stream rate limit; and a
    cut-short response, with and without a provider.
  * `TestRetry_IncompleteMakesOneCall` covers the old loop, until commit 4.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | ErrIncomplete is not beneath ErrInvalidRequest | `ErrIncomplete must match ErrInvalidRequest` |
  | the old retry loop retries a cut-short response | `calls = 3, err = <nil>; want 1 call and the incomplete error` |
  | Retryable ignores terminal | `Retryable() = true, want false` |
  | a plain 429 is classified terminal | `error = &llmprovider.APIError{…terminal:true}, want a retryable *APIError of kind ErrRateLimited, with retry-after 7s and the message` |
  | a 429 loses its retry-after | `RetryAfter = 0s, want 7s` (`TestClassifyHTTPStatus`); `TestClassifyHTTPError_PlainRateLimit` and `TestParseRetryAfter_MillisAndFractional` fail too |
  | a stream rate limit is not an APIError | `err = &fmt.wrapError{…}, want an *APIError` |
  | Error drops the retry-after | `Error() = "llmprovider: rate limited: HTTP 429", want "llmprovider: rate limited: HTTP 429 (retry-after 5s)"` |
  | Error drops the provider | `Error() = "llmprovider: rate limited: stream slow_down: wait", want "llmprovider: rate limited: openai stream slow_down: wait"` |
  | Error drops the reason | `Error() = "llmprovider: incomplete response", want "llmprovider: incomplete response: length"` |
  | a cut-short response reads as a stream | `Error() = "llmprovider: incomplete response: gemini stream: incomplete", want "llmprovider: incomplete response: gemini: incomplete"` |
  | a sentinel keeps llm: | `"invalid provider id" does not start llmprovider: (R27)` |
  | Responses' incomplete has the wrong kind | `incomplete with no reason: llmprovider: invalid request: unspecified, want reason unspecified` |
  | Chat's truncation has the wrong kind | `decode = <nil>/llmprovider: invalid request: length, want an *APIError of kind ErrIncomplete, reason length` |
  | Gemini's incomplete drops its reason | `incomplete: err = llmprovider: incomplete response, want an *APIError of kind ErrIncomplete` |

* **Docs.**
  * `architecture.md`'s error line describes the one type.
  * The migration guide fills 21 rows: `APIError` and its `Terminal` and
    `Type`; the four `IncompleteError` and seven `RateLimitError` rows; and
    the seven sentinels whose message changed. G-parity: `409 identifiers,
    409 rows, 201 with an SDK equivalent, 0 problem(s)`.
* **Lint.** Clean, with no change of configuration.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:** `llmprovider` 87.8 % from its own tests, and 89.2 % over
  `./llmprovider/...` (1752 of 1964 statements, 89.21 %) against its `P7`
  89.2 %. The removed types took covered statements with them.

### Phase S8, commit 4: the old generation API (2026-10-01)

Under "S8 as amended". Staged for the owner to commit.
*Annotated 2026-10-01:* committed as `9dbdf16`.

* **Removed from `llmprovider`:**
  * `LegacyProvider` and the seven optional interfaces built on it:
    `ToolProvider`, `ThinkingProvider`, `ThinkingToolProvider`,
    `ItemProvider`, `ItemToolProvider`, `ItemThinkingProvider` and
    `ItemThinkingToolProvider`. These are the step's "eight generation
    interfaces", counting `LegacyProvider`; no other remained;
  * `Continuer` and `ModelDiscoverer`;
  * `GenerateWithRetry`, `GenerateThinkingWithRetry` and
    `GenerateItemsWithRetry`, with their loop `retryWithBackoff` and
    `retryStops`. `serverRetryAfter` and `retryBackoffCap` stay, for
    `WithRetry`;
  * the `ProviderOption` alias;
  * the old-API-only options `WithThinkingBudget`, `WithReasoningEffort`,
    `WithKiloCapabilities`, `WithKiloDataCollection` and `WithStore`, their
    `ProviderConfig` fields, `legacyOption`, `Option.legacy` and
    `ResolveOptions`' refusal of them.
* **Unexported:**
  * `ProviderConfig` is `providerConfig`, holding what the common options
    set.
  * `ApplyOptions` is `applyOptions`, read only by `KiloProfile`, which takes
    `...Option` and builds no provider. It moves with the auth code in S8c.
* **Callers.**
  * `wizard/auth.go` and its Kilo test build `[]llmprovider.Option`.
  * Nothing else outside `llmprovider` used the old API.
* **Comments.**
  * The doc comments of `commonOption`, `newOption`, `ResolveOptions`,
    `WithModelProbes`, `WithModelMetadataURL`, `retryBackoffCap`, `Tool`,
    `Response.ID` and `catalog`'s `modelListingTimeout` no longer name the
    old API.
  * `contract.go` says the ids and `MessageItem.Role` are typed in commit 5.
  * Test comments that say what a test "was" keep the old names, as history.
* **Tests removed.** Each removed test's behaviour is covered by a
  `WithRetry` or `ResolveOptions` test, or was about the removed code alone:

  | Removed | Now covered by |
  |---|---|
  | `TestGenerateWithRetry_NonRetryable`, `TestGenerateItemsWithRetry_NonRetryable`, `TestGenerateThinkingWithRetry_NonRetryable` | `TestWithRetry_RetriesByKind` (authentication, invalid request) |
  | `TestGenerateWithRetry_RetriesThenSucceeds`, `TestGenerateItemsWithRetry_RetriesThenSucceeds`, `TestGenerateThinkingWithRetry_RetriesThenSucceeds` | `TestWithRetry_RetriesByKind` and `TestWithRetry_HonoursRetryAfter` |
  | the three `…_ContextCancelled` | `TestWithRetry_StopsWhenTheContextEnds` |
  | `TestGenerateThinkingWithRetry_Exhausted` | `TestWithRetry_StopsAfterMaxAttempts` |
  | `TestRetry_TerminalMakesOneCall`, `TestRetry_IncompleteMakesOneCall` | `TestWithRetry_RetriesByKind` (exhausted quota, a cut-short response) |
  | `TestRetry_RetryAfterAboveCapReturns` | `TestWithRetry_ReturnsAtOnceWhenTheServiceAsksTooMuch` |
  | `TestRateLimit_Classification` (`provider_correctness_test.go`, removed) | `TestSentinels_Beneath` and `TestWithRetry_RetriesByKind` |
  | `TestGenerateItemsWithRetry_UsesSharedLoop` | the removed helpers alone |
  | `TestResolveOptions_RefusesAnOldAPIOnlyOption` | the removed options alone |
  | `TestLive_ToolRoundTrip` (`live_items_test.go`, removed) | its table was empty: every row had moved to a provider's own live test in S7 |

* **Tests ported,** because `WithRetry` had no test of the behaviour:

  | Was | Now |
  |---|---|
  | `TestRetry_408IsRetried` | `TestWithRetry_Retries408` |
  | `TestRetry_ShouldRetryFalseIsTerminal` | `TestWithRetry_ShouldRetryFalseIsFinal` |
  | `TestGenerateWithRetry_SubNanosNoPanic` | `TestRetryPolicy_SubNanosNoPanic` |
  | `TestRetry_NegativeRetriesMakesOneCall` | `TestRetryPolicy_NegativeIsDefault` |
  | `TestApplyOptions_DefaultTimeout`, `TestApplyOptions_WithHTTPClient` | `TestResolveOptions_DefaultTimeout`, `TestResolveOptions_WithHTTPClient` |
  | `TestApplyOptions_TakesOnlyWhatTheOldConfigHolds` | `TestApplyOptions_TakesOnlyTheConfig`, with `WithMaxTokens` for the removed `WithReasoningEffort` |
  | `TestResolveOptions_ModelMetadataURLIsCommon` | the same, less its `ApplyOptions` half |

* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | `WithRetry` stops on a 408's legacy `ErrInvalidRequest` | `calls = 1, resp = <nil>, err = llmprovider: provider unavailable: huggingface HTTP 408: timeout; want a retry that succeeds` |
  | `x-should-retry: false` is ignored | `calls = 3, err = <nil>; want 1 call` |
  | the jitter takes a quarter of a 1 ns delay | a panic in `rand.N`, from `retry_policy_test.go:51` |
  | a negative `MaxAttempts` is kept | `withDefaults = {MaxAttempts:-1 BaseDelay:1s MaxDelay:30s}, want the defaults` |
  | the default client has no timeout | `default timeout: got 0s want 330s` |
  | `WithHTTPClient` is ignored | `WithHTTPClient override not honored (err <nil>)` |
  | `applyOptions` applies nothing | `applyOptions = {… MaxTokens:8192 BaseURL: …}` |

* **S8b.** Its `For` step's rule "it refuses an old-API-only option" has no
  option left to refuse; the step is annotated.
* **Links.** G-links found two links in
  `0003-PLAN-add-grok-xai-llm-provider.md` to the removed
  `provider_correctness_test.go`, one with a line anchor. Each is now plain
  text marked "removed by" this PLAN's S8, as that record already marks
  `thinking_test.go`, removed in S7. Its rationale is unchanged.
* **Docs.**
  * `architecture.md` no longer describes the old API: `Provider` is the
    contract, its abilities are `Request` fields under `Capabilities`, and
    `ModelLister` and `Streamer` are the only optional interfaces.
  * The migration guide fills or rewrites 46 rows: the interfaces and their
    methods, the retry helpers, `ApplyOptions`, `ProviderOption`, the old
    options, `ProviderConfig` and every field of it, including the two
    commit 2 removed, and the mcplib `Provider`'s `Generate` and `Name`.
    G-parity: `409 identifiers, 409 rows, 239 with an SDK equivalent, 0 problem(s)`.
* **Lint.** Clean, with no change of configuration.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:** `llmprovider` 87.8 % from its own tests, and 89.2 % over
  `./llmprovider/...` (1710 of 1916 statements, 89.25 %) against its `P7`
  89.2 %.

### Phase S8, commit 5: typed ids (2026-10-01)

Under "S8 as amended". Staged for the owner to commit.
*Annotated 2026-10-01:* committed as `7220fdb`.

* **The constants.** The ten `Provider…` constants are `ProviderID`, and
  `MessageItem.Role` is `Role`.
* **Retyped to `ProviderID`,** as every parameter or field that holds an id:
  * `llmprovider`:
    * `OAuthSession.Provider` (its JSON is unchanged) and
      `VendorCLISession.Provider`;
    * the provider parameter of `TokenStore`'s `Load`, `Save` and `Delete`,
      `RefreshLocker.LockRefresh`, and `FileTokenStore`'s methods;
    * `StartDeviceOAuth`, `LoginDeviceOAuth` and `LoginBrowserOAuth`;
    * `ProviderEnvVars()`, now keyed by `ProviderID`;
    * the unexported OAuth state, flow configuration and helpers, and
      `validateProviderID`.
  * `catalog`: `Rank`, `Static`, `Search`, `Label`, and `Metadata`'s
    `ReasoningEfforts`, `InterleavedField` and `NPM`, with the unexported
    listers. `List`'s dispatch lower-cases the id as before.
  * `wizard`: `Result.Provider`, `Options.Providers` and the unexported
    helpers.
* **Kept as `string`, as labels that may carry a route:**
  `APIError.Provider`, `ClassifyHTTPError`'s and `ClassifyStreamFailure`'s
  `provider`, `serviceOf`, and `responses.ReadStream`'s `provider`. Each
  caller passing an id passes `string(id)`.
* **Fault found by the tests, and fixed.**
  * The Messages and Chat Completions wires put the role into a
    `map[string]any`, then compared that entry with `wire.RoleAssistant`
    to group a turn.
  * Inside `any`, a `Role("assistant")` is not equal to the string, so
    grouping stopped. The JSON looked the same, but the turns were split.
  * `TestItemFidelity_ChatGroupsCalls`,
    `TestItemFidelity_AnthropicGroupsCallsAndResults`, `TestFromItems`,
    `TestItemsToChatMessages` and `TestOpencodeChat_ReplaysInterleavedReasoning`
    failed. G-wire did not: its scenarios group no turns.
  * Every wire now stores `string(v.Role)`, Responses and generateContent
    included. A search for type switches and `.(string)` assertions found no
    other place where an id or a role reaches an `any`.
* **Deviation, 2026-10-01: an unknown role.**
  * **Found.** D6 says an invalid value of a named type is refused by `New`
    or `Generate` with `ErrInvalidRequest`. `Request.validate` checked
    `Effort` and `ToolChoice` but never a `MessageItem`'s `Role`. The step
    said only "`MessageItem.Role` is `Role`".
  * **Decided.** The owner chose "validate in this commit". The other
    options were to defer it, or to amend D6.
  * **Done.**
    * `Capabilities.Check` refuses a role other than the three constants
      and empty, naming the input's index.
    * An empty role is still the user's.
    * `llmtest`'s R23 case gains "an unknown role", so every provider's
      `Generate` is shown to refuse it before the network.
    * Written red first: `TestCapabilitiesCheck_RefusesInvalidValues`
      (new row) and `TestCapabilitiesCheck_AcceptsTheRoles` failed before
      the check.
  * **Effect.** A caller sending another role, such as `"model"`, gets
    `ErrInvalidRequest`. Until now, each wire mapped it silently.
  * The MADR is unchanged: this makes the code meet D6.
* **Tests.** Typed throughout: table fields and helper parameters that hold
  an id are `ProviderID`, slices and maps of ids are typed, and `t.Run`
  names are `string(id)`. The test token stores take `ProviderID`. The
  classification helpers `classifyBody` and `classifyFixture` keep `string`,
  as their table holds labels. No assertion was weakened.
* **Comments.** `ProviderID`'s and `Role`'s docs no longer say they wait for
  this commit, and `catalog.Rank`'s no longer says it becomes
  `Rank(ProviderID, model)`.
* **Lint.** `unconvert` found 25 conversions to `ProviderID` that the typed
  constants made redundant, in the descriptors and tests; they are removed.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an unknown role is accepted | `Check = <nil>, want an error matching ErrInvalidRequest` |
  | an empty role is refused | `role "": Check = llmprovider: invalid request: input 0 has unknown role "", want nil` |
  | a provider takes the role "model" | `R23 (invalid values): an unknown role returned <nil>; want an error matching ErrInvalidRequest` |
  | Messages keeps the typed role in its map | `FromItems` splits the assistant turn: `{"content":"Checking.","role":"assistant"}`, then the calls in a second assistant message |
  | Chat keeps the typed role in its map | `empty role should default to user, got user`: the entry holds a `Role`, not the string the test compares |

  The retyping itself is the compiler's to check.
* **Docs.**
  * `architecture.md` says the ids are `ProviderID` and the labels stay
    strings.
  * The migration guide fills 27 rows: the nine constants, the role, the
    session and token-store fields and methods, the OAuth logins, the
    catalog functions, and `wizard`'s `Result.Provider`, `Options.Providers`
    and `Options.Existing`. G-parity: `409 identifiers, 409 rows, 262 with an SDK equivalent, 0 problem(s)`.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:** `llmprovider` 87.8 % from its own tests, and 89.2 % over
  `./llmprovider/...` (1716 of 1922 statements, 89.28 %) against its `P7`
  89.2 %; `catalog` 88.1 %, `wizard` 83.5 %.

### Phase S8, commit 6: 0016-PLAN T4 in `wizard` (2026-10-01)

Under "S8 as amended". Staged for the owner to commit, with 0016-MADR's
amendment "D11 and D5 in `wizard`" (A7-A10) and 0016-PLAN's deviation entry
of the same date.
*Annotated 2026-10-01:* committed as `aa30c71`. S8 as amended is complete.

* **Deviation, 2026-10-01: T4's open points.**
  * **Found,** in the survey, before any code. D11 and D5 left four points
    open:
    * Kilo's device token was stored and also returned as `Result.APIKey`
      (`wizard/auth.go:176-180`);
    * D11 does not say where `wizard` offers logout;
    * T4.3's table checks `json.Marshal`, but the consumer persists
      `Result`, `APIKey` included;
    * keeping a session rebuilt it from `Existing`'s tokens
      (`wizard/auth.go:225-256`).
  * **Decided.** The owner chose:
    * Kilo stored only, as `CredOAuth`;
    * a `Logout` function;
    * fmt and slog redaction only;
    * keeping a session reads the store only.
  * **Recorded** as 0016-MADR A7-A10, before the code.
* **`Result`.**
  * `AccessToken` and `RefreshToken` are removed, because no path sets them
    (A7).
  * `TokenExpiry`, `Issuer`, `ClientID`, `AccountID` and `FedRAMP` stay.
  * New `result_format.go` holds `String`, `GoString` and `LogValue`. Each
    prints the set fields, and `[redacted]` for `APIKey`.
* **Sign-in.**
  * `ConfigureLLM` no longer copies a session's tokens.
  * `resolveKiloDevice` returns the stored session as a `CredOAuth`
    credential. The listing and the provider send `Authorization: Bearer
    <token>` as before: an `OAuthSession` with no `Header` and a static key
    apply the same way.
  * `keepExistingOAuth` loads the session from `Options.TokenStore` when
    `Existing` names the provider as `CredOAuth`. With none stored, it
    signs in again; a failed load is returned.
* **`Logout`** (new `logout.go`) does, in order:
  * refuses to run without a store;
  * says so when no session is stored;
  * confirms, showing the access token masked;
  * revokes OpenAI and Grok sessions, with `Options.HTTPClient` when the
    session has none, and reports a failure as a warning;
  * says Kilo and any other provider offer no revocation;
  * deletes, returning a failed delete;
  * reports "Logged out of <label>".
* **Tests.**
  * **New** (`stored_session_test.go`, `logout_test.go`):
    * `TestConfigureLLM_StoredSessionLeavesNoTokenInResult`,
      `TestConfigureLLM_KeepsTheStoredSession` and
      `TestConfigureLLM_NoStoredSessionSignsIn`;
    * `TestResult_Redacts`;
    * the four `TestLogout_…`.

    The helper `resultFieldsHolding` walks every string field of `Result`,
    so re-adding a token field fails the tests.
  * **Rewritten to the store-only rule:**
    * `TestConfigureLLM_KiloDeviceLogin` (red first);
    * six tests that kept a session from `Existing`'s tokens, which now
      save it to the store through `storedExisting`:
      `TestConfigureLLM_ChatGPTResultDoesNotPopulateAPIKey`,
      `TestConfigureLLM_KeepsFedRAMP`,
      `TestConfigureLLM_ListingTokenFailureUsesStaticCatalog`,
      `TestConfigureLLM_KeepRefusesStubSession`,
      `TestConfigureLLM_ChatGPTListingFailurePromptsForModel` and
      `TestConfigureLLM_ChatGPTNoStaticNotice`;
    * the assertions that read `Result.AccessToken`
      (`TestConfigureLLM_TokenStdinClassification`,
      `TestConfigureLLM_TokenStdinStillAcceptsChatGPTToken`,
      `TestConfigureLLM_VendorLoginReadsThrough`), which now read the store and check
      that no field of `Result` holds the token.
  * **One changed number:** `TestConfigureLLM_KeepRefusesStubSession`'s
    "no save" is now "no save after the setup's" (`saves == 1`), because
    its setup saves the stub to the store.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an OAuth Result carries the access token | `Result fields [APIKey] hold a token; the store is the only copy` |
  | Kilo's token is the API key again | `Result kind "api_key" organization ""; want oauth, ""` |
  | keeping ignores the store | `signed in again; want the stored session kept` |
  | logout does not revoke | `revoked [], want the refresh token once` |
  | a failed revocation stops the delete | `the session is still stored` |
  | logout does not ask | `declined: revoked ["rt-logout"], stored map[]; want nothing changed` |
  | logout hides a failed delete | `failed delete: Logout() error = <nil>, want it returned` |
  | logout revokes every provider | `notices ["could not revoke the Kilo Gateway session (oauth: discovery for : Get \"/.well-known/openid-confi…` |
  | String shows the API key | `Result via %+v shows the key: Result{Provider:claude Kind:api_key APIKey:key-PLANTED-TOKEN-5c1e Model:model…` |
  | GoString is the default | `Result via %#v shows the key: wizard.Result{Provider:"claude", Kind:"api_key", APIKey:"key-PLANTED-TOKEN-5c…` |
  | LogValue is the default | `Result via slog JSON shows the key: {"time":"2026-10-01T20:37:04.230205-05:00","level":"INFO","msg":"m","v"…` |
  | the JSON is redacted | `json.Marshal = {}, <nil>; want the key kept for the consumer to persist` |

* **Docs.**
  * `architecture.md`'s wizard section covers the one copy, Kilo, `Logout`
    and `Result`'s redaction, and its auth section no longer says Kilo's
    token is the API key.
  * The migration guide fills seven `wizard` rows, `AccessToken` and
    `RefreshToken` as removed. G-parity: `409 identifiers, 409 rows, 268 with an SDK equivalent, 0 problem(s)`.
* **Lint.** Clean, with no change of configuration.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:** `wizard` 84.0 % (it was 83.5 %), `Logout` 91.7 %;
  `llmprovider` 89.3 % over `./llmprovider/...`, against its `P7` 89.2 %.

### Phase S8b, step 4: `For(id, opts...)` (2026-10-01)

Staged for the owner to commit. Steps 1-3 were done in S8, commit 2.
*Annotated 2026-10-01:* committed as `79b1a64`. S8b is complete.

* **`llmprovider.For(id, opts...)`** (`settings.go`) is an `Option` holding
  its id and options (0015-MADR D5 steps 2-4).
  * `ResolveOptions` applies every option outside a `For` in order, then
    the options of each `For(id, …)` for the id being built, in order. The
    more specific wins wherever it sits in the list. A `For` of another id
    is skipped.
  * Inside an overlay an option is checked as if it were bare. A
    provider-specific option inside `For(kilo, …)` is legal in a shared
    list, and one kilo does not take is still refused when building kilo.
  * **Refused by every `New`, whichever id it builds:** a `For` with no
    id, and a `For` nested inside a `For` of another id. A `For` of the
    same id nested inside one is flattened, being only redundant.
  * Step 4's old-API-only refusal has nothing left to refuse (S8,
    commit 4).
* **`wizard`.** Its listing list puts the Kilo organization in
  `For(kilo, catalog.WithKiloOrganization(org))` unconditionally. An empty
  organization is the personal account, so the list is the same whichever
  provider was chosen.
  * **Deviation, 2026-10-01.** The step said `wizard` passes one list
    "through the `Registry`". `wizard` builds no provider: it lists with
    `catalog.List`. The owner chose "overlay in the listing list". The
    step's text is annotated.
  * The change keeps behaviour, so it cannot be shown red first. The new
    `TestConfigureLLM_KiloListsTheChosenOrganization` passed before it and
    after it. The breaks below show the test and the existing listing
    tests catch a wrong or missing overlay.
* **Red first.** `settings_for_test.go` first did not compile, `undefined:
  For`. Then it ran against a stub `For`, which ignored its options, in a
  scratch copy. Each rule failed on its assertion:
  * `kilo MaxTokens = 100, want For's 5`;
  * `MaxTokens = 100, want For's 5 over the later baseline`;
  * `kilo: err <nil>, values []; want [org-1]`;
  * `err = <nil>, want the foreign option inside For refused`;
  * `nested under a different id, building kilo: err = <nil>, …` and `no
    id, building kilo: err = <nil>, …`, for each of three ids;
  * `same id nested: MaxTokens = 8192, want 3`.
* **Tests:**
  * `llmprovider`: `TestFor_AppliesOnlyToItsID`,
    `TestFor_WinsOverTheBaselineWhereverItSits`,
    `TestFor_ScopedOptionInASharedList`, `TestFor_StaysStrictForItsOwnID`
    and `TestFor_Refusals`.
  * `providers`: `TestFor_OneListBuildsEveryProvider`. One list, with
    `kilo`, both `opencode` gateways, `openai`, `gemini` and `grok`
    overlays, builds all ten built-in providers. The same Kilo option given
    bare is refused by the other nine.
  * `kilo`: `TestListModels_ListsTheOrganization` gains a row with the
    organization inside `For`.
  * `wizard`: `TestConfigureLLM_KiloListsTheChosenOrganization`.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | an overlay applies to every id | `openai MaxTokens = 5, want the baseline's 100` |
  | an overlay applies to no id | `listing requests = ["/models org="], want the organization's catalog with its header` |
  | an overlay applies where it sits | `MaxTokens = 100, want For's 5 over the later baseline` |
  | an overlay skips the scope check | `err = <nil>, want the foreign option inside For refused` |
  | a For of another id nests | `nested under a different id, building kilo: err = <nil>, want "For(\"openai\", …) inside For(\"kilo\", …)"` |
  | a For with no id is taken | `no id, building kilo: err = <nil>, want "For needs a provider id"` |
  | only the target checks a For | `nested under a different id, building openai: err = <nil>, want "For(\"openai\", …) inside For(\"kilo\", …)"` |
  | a same-id nested For is dropped | `same id nested: MaxTokens = 8192, want 3` |
  | wizard's overlay names another id | `organization "org-1", listings ["/api/gateway/models org="]; want org-1's catalog with its header` |
  | wizard passes the organization bare | `listing requests = [], want one Codex /models call` |

* **Docs.**
  * The standards guide's R18 reads "unless the caller scoped it with
    `For(id, …)`".
  * `architecture.md`'s contract section describes `For`.
  * `For` is new, so the migration guide gains no row.
* **Lint.** Clean, with no change of configuration.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:** `llmprovider` 88.0 % from its own tests and 89.4 % over `./llmprovider/...`, against its `P7` 89.2 %; `wizard` 84.3 %.

### Phase S8c, step 1: the ChatGPT helpers move to `openai` (2026-10-01)

Staged for the owner to commit, with 0015-MADR amendment "the ChatGPT
helpers leave `llmprovider`", written before the code.
*Annotated 2026-10-01:* committed and pushed as `fdd3e17`.

* **Deviation, 2026-10-01.** The survey found three facts the step did not
  cover:
  * the helpers read unexported session state;
  * `grok` also uses `ExpireSession`;
  * `wizard` lists ChatGPT sessions through `catalog`.

  They were put to the owner and decided as the amendment says. The step is
  annotated.
* **Moved to `providers/openai`:**
  * `chatgpt_listing.go`: the Codex listing, now the method `listChatGPT`;
    `chatgptClientVersion` and its pattern; and the listed-model filter.
    `ListModels` lists a ChatGPT session itself, under the same 10 s bound
    `catalog` applies (`chatGPTListingTimeout`), with no static fallback and
    no probe.
  * `chatgpt_session.go`: `isChatGPTSession`, `setChatGPTHeaders` (the
    originator, account and FedRAMP headers, read through
    `Account()`), `invalidate`, and the four header constants, unexported.
* **`llmprovider`:**
  * `openai_chatgpt.go` is removed: `IsChatGPTSession`,
    `ChatGPTSessionAccountID`, `ChatGPTSessionFedRAMP`, `ExpireSession`, and
    `ChatGPTAccountHeader`, `ChatGPTOriginatorHeader`,
    `ChatGPTOriginatorValue` and `ChatGPTFedRAMPHeader`.
  * `(*OAuthSession).Account()` and `Invalidate()` are new, so an
    `*OAuthSession` is an `InvalidatingSource`.
  * `VendorCLISession`'s `vendorAccount` is the exported `Account()`.
  * OpenAI's authorize URL keeps its `originator` parameter, through an
    unexported constant of the sign-in code.
* **`catalog`** no longer special-cases a ChatGPT session. The listing, its
  type, filter, client-version helpers and their two imports are removed. A
  caller that passes a ChatGPT session to `catalog.List` now gets the
  Platform listing, which degrades to the static catalog. `openai`'s
  `ListModels`, and `wizard`, no longer call it for one.
* **`grok`** invalidates through `InvalidatingSource` after a 401, with its
  own `invalidate`.
* **`wizard`.** For a ChatGPT credential, `chatGPTCatalog` builds the
  `openai` provider from the run's `Registry` with the session, the HTTP
  client and the base URL. It calls `ListModels`, and returns a live
  catalog. Every other provider lists through `catalog.List`.
* **Tests.**
  * **Moved:** `catalog`'s `chatgpt_listing_version_test.go` to `openai`
    (`git mv`), with its helper building the provider.
  * **Ported** to `openai`'s `chatgpt_listing_test.go`, as
    `TestListModels_ChatGPTListsCodexCatalog`:
    * `TestListAvailableModelsWithSource_ChatGPTListsCodexCatalog`;
    * the live, one-request half of
      `TestListModelCatalogWithSource_ChatGPTListsCodexCatalog`;
    * the FedRAMP header, which the listing had no test of.

    The catalog's other half, `Recommended` and `Usable` not sharing an
    array, has no counterpart: `ListModels` returns one slice, and the
    ported test checks it is fresh on each call.
  * **Removed:** `TestListAvailableModelsWithSource_ChatGPTListingFailureIsError`,
    which `openai`'s `TestListModels_ChatGPTListingFailureIsError` already
    covers.
  * **New:**
    * `TestListModels_ChatGPTListingBounded`;
    * `TestOAuthSession_Invalidate`, and a compile-time check that
      `*OAuthSession` is an `InvalidatingSource`;
    * `TestOAuthSession_Account`.
  * **Renamed in assertions:** `TestChatGPTLogin_FedRAMPClaim` reads
    `Account()`, and `openai`'s tests use the unexported header constants.
  * `wizard`'s `TestConfigureLLM_ChatGPTNoStaticNotice` and
    `TestConfigureLLM_ChatGPTListingFailurePromptsForModel` now run through
    `chatGPTCatalog`.
* **Fault found and fixed during the move.** The script's first pass at
  removing the catalog tests used a pattern whose comment prefix could span
  lines. In `discovery_catalog_test.go` it removed 13 unrelated functions,
  among them `serveBody` and `catalogCase`. The compiler caught it. The file
  was rebuilt from `HEAD` with only the ChatGPT test removed, by a pattern
  that checks its match spans one function: 32 lines, the test and two
  imports only it used.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | a ChatGPT session lists through catalog | `models = [gpt-4.1-mini gpt-4.1-nano gpt-4o-mini gpt-4.1 gpt-4o o4-mini], want [gpt-6-astra gpt-5.6-sol gpt-…` |
  | the listing drops the account header | `ChatGPT-Account-Id = "", want "acct_live"` |
  | the listing drops the FedRAMP header | `X-OpenAI-Fedramp = "", want "true"` |
  | the listing has no bound | `the listing ran with -1ns left, want a deadline within 10s` |
  | the listing sends a fixed client_version | `client_version = "0.0.0", want "1.5.0"` |
  | Invalidate leaves the expiry | `after Invalidate: Token() = <nil>, want ErrAuthFailure: no refresh token` |
  | openai does not invalidate after a 401 | `Generate() error = llmprovider: authentication failed: openai HTTP 401` |
  | grok does not invalidate after a 401 | `Generate() error = llmprovider: authentication failed: grok HTTP 401` |
  | Account drops the FedRAMP flag | `FedRAMP = true, Account's flag = false; want both true` |
  | wizard lists ChatGPT through catalog | `ConfigureLLM() error = select model: fakePrompter: unexpected Select("Choose a OpenAI model:")` |

* **Docs.** `architecture.md` says the ChatGPT session is `openai`'s, and
  describes the 401 rule for any `InvalidatingSource`. Only
  `ShareHTTPClient` remains as a temporary export. The removed helpers were
  never in `mcplib`, so the migration guide gains no row.
* **Lint.** Clean, with no change of configuration.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:**
  * `llmprovider` 89.1 % from its own tests (88.0 % before), and 89.35 %
    over `./llmprovider/...` (1720 of 1925 statements), against its `P7`
    89.2 %;
  * `openai` 92.6 %, `catalog` 88.3 %, `wizard` 84.3 %.

### Phase S8c, steps 2-5: `auth` (2026-10-01)

Staged for the owner to commit, with 0015-MADR amendment "what `auth`
holds", written before the code. With it S8c is complete.

* **Deviation, 2026-10-01: what `auth` holds.**
  * **Found.** A trial move in a scratch copy, which the compiler checked,
    found three points the step did not settle:
    * `StaticToken` cannot leave `llmprovider` (`WithAPIKey` builds one);
    * `ShareHTTPClient` had no successor;
    * the base URL constants are the providers'.

    It also showed what crosses the new boundary: only `tokenFuture` and
    `ErrInvalidProvider` from the staying code; five unexported helpers
    from the moving code.
  * **Decided** by the owner, as the amendment says:
    * `StaticToken` and `CommandToken` stay;
    * `OAuthSession.UseHTTPClient`;
    * `openai.PlatformBaseURL`, `openai.ChatGPTBaseURL` and `grok.BaseURL`.
  * **Corrected before commit.** The amendment's first draft said `auth`
    imports only `llmprovider` and `internal/transport`. The moved code also
    imports `internal/redact` and `internal/kiloendpoint`, and the bullet
    now says so.
* **Before the move,** in `llmprovider`:
  * `transport.NewIdentity` holds the identity defaults; `identityOf` and
    `auth` both call it.
  * `ErrInvalidProvider` joins D7's sentinels in `api_error.go`.
  * `CommandToken` has its own `tokenFuture`.
  * `KiloProfile` resolves its options with `ResolveOptions(kilo, …)`, so
    `applyOptions` and its test are removed. **Changed behaviour:** an
    option for another provider is now refused, where `applyOptions`
    ignored it. The User-Agent is the same, read from `Settings`.
  * `ShareHTTPClient` is `OAuthSession.UseHTTPClient`. `openai` and `grok`
    call it through an unexported `clientUser` interface.
* **Moved with `git mv`** into `llmprovider/auth`:
  * the sources: `oauth_constants.go`, `oauth_device.go`,
    `oauth_idtoken.go`, `oauth_loopback.go`, `oauth_pkce.go`,
    `oauth_revoke.go`, `oauth_session.go`, `tokenstore.go`,
    `tokenstore_file.go` (and its `_unix` and `_windows` halves),
    `vendor_session.go` and `kilo_device.go`;
  * 29 test files, and `jwt_helper_test.go`.

  The package is renamed, and every staying `llmprovider` identifier is
  qualified.
  * The rewriter qualified struct field names that match a staying type,
    such as `Provider`. The compiler found the seven, and they were
    unqualified.
  * The rewriter now leaves a method's own name alone: `) Name(`.
* **New in `auth`:**
  * `doc.go`, the package doc: the note that stood above the client ids,
    as the step says;
  * `helpers.go`, with `auth`'s copies of `secretText`, `expiryText`,
    `closeResponseBody`, `jsonString` and `tokenFuture`. Each package closes
    its own bodies, for bodyclose.
* **Split** between the two packages:
  * `TestTokenSources_ReportOnlyWhatIsSet` keeps the key and command rows.
    The session rows are `auth`'s `TestSessions_ReportOnlyWhatIsSet`.
  * `TestSecretBearingTypesRedact` keeps `Token` and `StaticToken`. The
    session row is `auth`'s `TestSessionRedacts`, and `TestSecretText` has a
    copy in `auth` for its helpers.
  * The live session helpers, `liveChatGPTSession` and `liveVendorSession`,
    use only exported API. They are functions of the external
    `llmprovider_test` package, which imports `auth`, and leave
    `live_export_test.go`.
  * `live_oauth_issuers_test.go` and `live_oauth_revoke_test.go` moved to
    `auth`, with a `liveCtx` of their own.
    `live_oauth_login_test.go` moved, then moved back: it generates through
    `openai` with `llmprovider_test`'s helpers.
* **Callers:**
  * `llmprovider.X` is `auth.X` in nine provider packages, `wizard` and
    their tests, for the 20 names `auth` exports.
  * Imports were then grouped as the codebase does: the standard library,
    then this module.
* **Step 3's check.** `go doc -all ./llmprovider` names none of the session
  types, flows, stores, ChatGPT helpers, `ShareHTTPClient` or issuers. The
  package doc no longer says it holds OAuth sessions.
* **Step 4, coverage.** The `-coverpkg` measurement of `llmprovider` ends.
  The gate's floor check now measures each package from its own tests:
  * `llmprovider` 98.2 %, against its `P7` 89.2 %;
  * `auth` 85.3 %, against 80 %.

  Run with raised floors, the check failed: with 99 % for `llmprovider`,
  and with 90 % for `auth`.
* **New test:** `TestKiloProfile_ResolvesKilosOptions`, the caller's
  application in User-Agent, and the foreign option refused before any
  request.
* **Fault in my own check, found by the gate.** A filter on the live vet's
  output hid lines that start `vet: `, which is how vet reports a type
  error. So `liveOpenAI`, undefined in `auth`, was missed until the gate ran
  the command unfiltered. The fix is the move back above.
* **Breaks,** each in a scratch copy:

  | Break | Failure |
  |---|---|
  | UseHTTPClient replaces the session's own client | `a session's own client was replaced` |
  | openai does not share its client | `POST /oauth/token did not go through the provider's client (it carried [POST /responses GET /models])` |
  | grok does not share its client | `POST /oauth/token did not go through the provider's client (it carried [POST /responses GET /models POST /r…` |
  | KiloProfile ignores the caller's identity | `User-Agent = "go-llmprovider-sdk", want the caller's application first` |
  | NewIdentity drops the default name | `UserAgent() = "/(devel) (darwin; arm64) go-llmprovider-sdk/(devel)"; want this module by default` |
  | the ChatGPT base URL is the Platform's | `request URL = https://api.openai.com/v1/responses, want chatgpt.com/backend-api/codex/responses` |
  | a session's redaction shows the access token | `*OAuthSession via json.Marshal shows the secret: {"provider":"","issuer":"https://auth.x.ai","client_id":"c…` |
  | KiloProfile resolves as another provider | `foreign option: err = <nil>, request sent true; want ErrInvalidRequest before any request` |
  | a store accepts a traversing id | `Save("open..ai") returned nil error; want error` |

  Two were invalid as first written, and were rewritten and run again:
  * one kept the foreign option (`opts[:3]` of three options);
  * one did not compile.
* **Docs.**
  * `architecture.md`: the `auth` package's row and directory line;
    `llmprovider`'s new description; `auth` among the dependencies of the
    eight providers that import it, and of `wizard`; and the session text.
  * The standards guide's package table: `StaticToken` and `CommandToken`
    in `llmprovider`, and `auth`'s imports.
  * The migration guide maps 46 rows to `auth`, `openai` or `grok`.
    G-parity: `409 identifiers, 409 rows, 304 with an SDK equivalent, 0
    problem(s)`.
* **Lint.** Clean, with no change of configuration.
* **G-wire.** All goldens unchanged, with no `-update`.
* **Coverage:** `llmprovider` 98.2 %, `auth` 85.3 %, `openai` 92.7 %, `grok`
  95.6 %, `wizard` 84.3 %.
