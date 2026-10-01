---
status: in-progress
date: 2026-09-30
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

### Phase S8b: extract `catalog`

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

   Red-first tests, and breaks for each rule. The standards guide's R18
   gains the `For` exception. `wizard` passes one option list through the
   `Registry`.
5. Gate.

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
| S3 (in `llmprovider`; moves to `internal/transport` in S7b, amended 2026-09-30) | T1: proxy and one client per provider (D8) |
| S4 (in place; moves to `auth` in S7b) | T2: durable writes, rotation kept, redaction, device handle, OAuth checks with `id_token` signature verification (D3–D7) |
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
Staged for the owner to commit.

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
