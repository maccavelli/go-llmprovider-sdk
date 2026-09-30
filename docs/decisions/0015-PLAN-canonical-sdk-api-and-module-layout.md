---
status: in-progress
date: 2026-09-29
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

### Phase S3: extract `internal/wire` and `internal/transport`

1. `git mv` the four wire-format files and their tests into
   `internal/wire/{responses,chatcompletions,messages,interactions}`:
   `chatcompletions.go`, the Responses encode/decode, the Messages encoding,
   and `gemini_interactions.go`.
2. `git mv` the transport files and their tests into `internal/transport`:
   `http_helpers.go`, `identification.go`, the classification part of
   `api_error.go`, and `probe.go`.
3. Export within `internal` only what the providers need. `llmprovider`
   keeps calling them.
4. Gate, including G-wire unchanged.

### Phase S4: extract `auth`

1. `git mv` these into `llmprovider/auth`:
   * `oauth_*.go`;
   * `tokenstore*.go`;
   * `vendor_session.go`;
   * the session part of `token.go`;
   * their tests.
2. `Token`, `TokenType` and `TokenSource` stay in `llmprovider`, as D2
   says.
3. Fix the stale "Phase 2 / Phase 3" doc comments (0015-REPORT F7).
4. Gate.

### Phase S5: extract `catalog`

1. `git mv` these into `llmprovider/catalog`:
   * `models_catalog.go`, `model_ranking.go`, `model_matcher.go`,
     `model_metadata.go`, `model_profile.go`;
   * the curation half of `discovery.go`;
   * their tests.
2. Replace the exported mutable variables with functions returning copies
   (D9):
   * the seven `Static*` catalogs;
   * `ProviderEnvVars`.
3. Collapse the seven `Rank*Model` functions behind one
   `catalog.Rank(ProviderID, model)`.
4. Gate.

### Phase S6: the contract, and `llmtest`

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

   The old types stay alongside until S8.
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
   * `RateLimitError` and `IncompleteError`.
4. Gate, with G-parity's empty-cell check still off.

### Phase S9: usage

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
| S-A1 | Package layout and allowed imports as D2 | `make dep-check`; package list | S3–S8, S12 |
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
