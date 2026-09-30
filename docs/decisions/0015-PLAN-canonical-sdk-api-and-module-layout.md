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
  * a coverage floor broken;
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

*Proposed 2026-09-30 ([0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md) D1):* accepted by the
owner. `together` joins the order after `ollama`. It lands in `llmprovider` first,
under 0017-PLAN U1.

For each provider:

1. `git mv` its files and tests into `llmprovider/providers/<id>`.
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
once no provider is left in `llmprovider`. The four packages can then import
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
4. `git mv` into `llmprovider/catalog`:
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

1. **`providers.Default()` and `providers.New`.** Descriptors move to their
   provider packages. `TestDescriptors_CoverEveryRegisteredProvider` becomes
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
