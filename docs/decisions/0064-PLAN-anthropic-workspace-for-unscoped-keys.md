---
status: in-progress
date: 2026-10-27
associated-madr: "0064-MADR-anthropic-workspace-for-unscoped-keys.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Anthropic's Workspace Header for Keys Not Scoped to a Workspace

Associated MADR: [0064-MADR-anthropic-workspace-for-unscoped-keys.md](0064-MADR-anthropic-workspace-for-unscoped-keys.md)

## Goal

The MADR's D1 to D7, under the owner's answers to its Q1 to Q3. When this
PLAN is complete:

* a Claude provider built with `claude.WithWorkspace` sends
  `anthropic-workspace-id` on every request, and its listing does too;
* Anthropic's workspace refusals classify to `ErrWorkspaceRequired` and
  `ErrWorkspaceNotFound`, both still `ErrInvalidRequest`;
* the wizard asks for a workspace when a key needs one, saves nothing when
  none works, and returns it in `Result.Workspace`;
* a key that works today sends the requests it sends today.

## Scope

### In scope

* `llmprovider/api_error.go`: the two sentinels and their classification.
* `llmprovider/providers/claude/claude.go`: `WithWorkspace`, the header on
  generation, streaming and probes, and the listing options it passes on.
* `llmprovider/catalog/config.go`, `discovery.go`, `listing_cache.go`:
  `WithClaudeWorkspace`, the header on each listing page, and the cache key's
  scope.
* `wizard/configure.go`, `wizard/describe.go`, `wizard/result_format.go`: the
  Result field, the prompt, the bound, the classes.
* Tests, redacted fixtures, and the live test `LLMPROVIDER_LIVE_0064`.
* `docs/architecture.md`, `README.md`, `CHANGELOG.md`, `docs/README.md`.
* *Added 2026-10-27, by Phase 1's deviation:* `scripts/check_fixtures.py`,
  `scripts/test_gates.py`, `AGENTS.md`'s "Recorded fixtures", and an
  annotation of 0047-MADR D1.

### Out of scope

* Discovering workspaces through the Admin API (MADR D5).
* Any consumer: prepare-commit-msg's change is its own record after this
  SDK's release (MADR "After this record").
* The release itself: the tag and push are the owner's.
* The tree's uncommitted changes that predate this PLAN
  (`docs/decisions/0061-MADR-consumer-gaps-from-gobble-issue-9.md`,
  `docs/decisions/0061-PLAN-consumer-gaps-from-gobble-issue-9.md`,
  `llmprovider/catalog/cache.go`, `llmprovider/settings.go`). No phase stages
  or edits them. A phase that must edit one of those files stops and asks
  first. *Annotated 2026-10-26: those changes were 0061's and 0062's, and were
  committed and released in `v1.23.0`; the tree had no uncommitted change when
  Phase 0 resumed.*

## Rules for every phase

* Each new test is seen to fail on a planted breach first, on a scratch clone
  (`tools/plant.py`), never in the tree. The phase's record says what was
  planted and how it failed.
* `python3 -B tools/gate.py` passes before the phase is staged, and
  `make pre-add-check FILES=...` passes on each staged Go file.
* Fixtures recorded from Anthropic are redacted as AGENTS.md's "Recorded
  fixtures" says: a workspace ID becomes `wrkspc_REDACTED`.
  `make fixtures-check` passes.
  *Annotated 2026-10-27: AGENTS.md said nothing of workspace IDs, and
  `fixtures-check` did not look for them; Phase 1 adds both (its
  deviation of 2026-10-27).*
* No phase reads a key into a log, a fixture, or a record.
* The owner commits each phase (`git commit --no-edit`).

## Implementation Steps

### Phase 0: records

1. This pair, `status: proposed`, with their two rows in `docs/README.md`.
2. `make records-check` and `tools/recordlint.py --changed` pass.
3. After the owner answers Q1 to Q3: the answers are recorded in the MADR,
   the MADR's status becomes `accepted` and the PLAN's `in-progress`
   (`tools/records.py status`). A changed answer that alters a decision is
   written into that decision before Phase 1.

### Phase 1: measure and pin the replies (MADR E5, D3)

1. `llmprovider/live_0064_workspace_measure_test.go`, build tag
   `live_gateways`, switched on by `LLMPROVIDER_LIVE_0064`, reading the key
   from `ANTHROPIC_API_KEY` and the workspace from
   `LLMPROVIDER_LIVE_0064_WORKSPACE`. It sends, through `ClassifyHTTPError`'s
   own path:
   * `GET /v1/models` with no header;
   * the same with `not-a-workspace`;
   * the same with a well-formed ID of a workspace the identity may not use,
     from `LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE` when set (to measure E5's
     404), skipped otherwise;
   * the same with the workspace.
   It writes each reply's status and body, the workspace redacted, to a
   scratch directory the run names. The owner runs it.
2. The redacted bodies become fixtures under
   `llmprovider/testdata/anthropic-workspace/`.
3. If E5's 404 is measured as documented, D3 stands. If not, the MADR gets a
   dated amendment with the measured reply before Phase 2.

### Phase 2: the classes (MADR D3)

1. `ErrWorkspaceRequired` and `ErrWorkspaceNotFound` in
   `llmprovider/api_error.go`, as `sentinelError`s with `ErrInvalidRequest`
   as parent, each with a doc comment citing 0064-MADR D3.
2. Two cases in `classifyAPIError`, for the Claude service only, before the
   default, matching the error type and the fixed phrase of each fixture.
3. Tests in `llmprovider/api_error_0064_test.go`: each fixture classifies to
   its sentinel, matches `ErrInvalidRequest`, and is terminal; the same
   bodies from another service do not; a Claude 400 with another message
   stays `ErrInvalidRequest`.
4. `wizard/describe.go`: both in `errorClasses`, before
   `ErrInvalidRequest`. A test: `describe` words each as "workspace
   required, HTTP 400" and "workspace not found, HTTP 404".

### Phase 3: the header (MADR D1, D2)

1. `catalog.WithClaudeWorkspace` in `config.go`, scoped to `ProviderClaude`;
   `config.ClaudeWorkspace`; `fetchClaudePage` sets the header when it is not
   empty.
2. `listing_cache.go`: the key's `organization` field becomes `scope`,
   holding `KiloOrganization` or `ClaudeWorkspace`. Existing cache tests pass
   unchanged.
3. `claude.WithWorkspace` in `claude.go`: trimmed; a control character is
   `ErrInvalidRequest` at `New`; `Prepare` sets the header when not empty;
   `New` adds `catalog.WithClaudeWorkspace` to `p.listing` when set.
4. Tests in `llmprovider/providers/claude/workspace_0064_test.go` and
   `llmprovider/catalog/workspace_0064_test.go`, against `httptest` servers:
   * the header is present with the option and absent without it, on
     Generate, Stream, a probe, and every page of a two-page listing;
   * two workspaces with one key make two cache entries;
   * a control character fails `New` with `ErrInvalidRequest`;
   * an option scoped to Claude given to another provider is refused as
     `ScopedOption` refuses today.
5. *Added 2026-10-27, by the MADR's amendment "the probe cache":*
   `transport.ProbeKey` gains `Scope`, `wire.ProbeHealth` a `scope`
   parameter; the Claude provider passes its workspace, every other caller
   `""`. A test: one key's probes in two workspaces are two entries, and the
   same workspace's second listing sends no probe.

### Phase 4: the wizard (MADR D4)

1. `Result.Workspace`, with its doc; `result_format.go` shows it.
2. `discoverModels` passes `catalog.WithClaudeWorkspace(res.Workspace)`.
3. `credentialStep`: on either class, the notice and the prompt of D4.1,
   bounded by `maxTypedKeys`; the default of D4.3 as Q2 decides; on the
   bound, an error wrapping the last failure and no save.
4. Tests in `wizard/workspace_0064_test.go` with the scripted prompter and an
   `httptest` Anthropic that requires a workspace:
   * a key needing a workspace is asked for one, lists with it, and the
     Result carries it;
   * a wrong workspace is asked again, and three wrong ones end with an error
     matching `ErrWorkspaceNotFound` and nothing saved;
   * a scoped key is never asked;
   * `Existing.Workspace` is the default, and so is the variable when Q2 says
     so and `AllowEnv` is set;
   * no notice carries the service's message (0056-MADR D3's existing check
     covers the new notices).

### Phase 5: docs and the API check (MADR D6, D7)

1. `docs/architecture.md`, the README's Claude section, and `CHANGELOG.md`
   under Unreleased, Added.
2. `make api-check` lists exactly D6's five additions; its output goes in the
   execution record.
3. `make parity-check dep-check coverage-check api-check generate-check
   records-check tables-check fixtures-check changelog-check docs-check
   gate-selftest`, `make lint` and `make vuln` pass.

### Phase 6: live confirmation, run by the owner

1. `LLMPROVIDER_LIVE_0064=1` with key C: no header classifies to
   `ErrWorkspaceRequired`, the workspace lists and generates, a malformed one
   classifies to `ErrWorkspaceNotFound`.
   *Annotated 2026-10-27: no test did this; Phase 1's measurement sends raw
   listings only. Phase 6 adds it first (its record, "the test"), and key D,
   the organization's key of Phase 1's runs, stands for key C.*
2. The owner runs the wizard through a scratch consumer, or through
   prepare-commit-msg with a `replace` in a scratch clone, with key C: it asks
   for the workspace, lists, and returns it. Nothing in either tree changes.
3. The MADR becomes `accepted` if it is not already, and the PLAN
   `complete`.

## Verification

* Each phase's tests, seen to fail on a planted breach first.
* `python3 -B tools/gate.py` and the make targets of Phase 5, step 3.
* `go test ./...` with no `live_gateways` tag sends nothing to Anthropic.
* Phase 6's live run.

## Rollout and Rollback

* **Rollout:** additive exports and a refined classification; the next minor
  release (v1.23.0, *annotated 2026-10-26: `v1.23.0` shipped 0061 and 0062;
  this is `v1.24.0`*), tagged by the owner. Consumers adopt it by wiring
  `Result.Workspace` (prepare-commit-msg first, under its own record).
  *Annotated 2026-10-27: `v1.24.0` was tagged before its changelog section,
  so its CI made no Release; `v1.24.1` carries the release, and `v1.24.0`
  gets its Release by hand (Phase 7).*
* **Compatibility:** a caller that matches `ErrInvalidRequest` still matches.
  A key that works today sends no new header unless the caller sets one.
* **Rollback:** revert the phase's commits. A consumer that persisted a
  workspace keeps the field in its config, which an older SDK ignores; that
  consumer's key fails again exactly as it does today.

## Execution record

### Phase 0: records (2026-10-26)

* **Steps 1 and 2** were done when the pair was written (2026-10-25): its two
  rows in `docs/README.md`, `proposed`.
* **Step 3.** The owner answered Q1 to Q3 through one prompt of three options
  each, the recommended one first ("0064, prompt questions, 3 options each
  with recommendations. proceed"): Q1 two sentinels, Q2 as D4 writes, Q3
  deferred, each the decision as written, so no decision changes. The MADR
  records them, and its amendment "the findings at a878fa4" re-verifies F1 to
  F7 at the current code; `tools/records.py status`: the MADR `accepted`, this
  PLAN `in-progress`. The Out-of-scope list's uncommitted files, and the
  Rollout's `v1.23.0`, are annotated as stale.
* **For Phase 1:** none of `LLMPROVIDER_LIVE_0064_WORKSPACE`,
  `LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE` or `ANTHROPIC_WORKSPACE_ID` is
  set here, and this machine's `ANTHROPIC_API_KEY` sent Claude requests with no
  workspace header in 0061's Phase 7, so it is a scoped key, not the MADR's
  key C. Phase 1's measurement needs key C and its workspace from the owner.
* **Checks:** `make records-check`, `tools/recordlint.py --changed`,
  `tools/denyscan.py --changed`, `tools/linkcheck.py`.

### Phase 1: the measurement, written (2026-10-26)

* **Approval.** "committed proceed", after Phase 0 (`f45bd66`). The owner has
  a new key from an organization, not scoped to a workspace, and only the
  Default workspace.
* **Step 1, as built:**
  * `llmprovider/workspace_0064_record_test.go` (untagged, so its twin runs
    offline, as 0061's and 0062's live recorders are built):
    `workspaceCases` (no workspace, `not-a-workspace`, a foreign workspace
    when one is given, the key's own); `recordWorkspace`, one `GET
    /v1/models?limit=1` with `x-api-key` and `anthropic-version`
    `2023-06-01`, as the listing sends it, whose record holds the status,
    the body, the service's type and message from `ClassifyHTTPError`,
    whether the reply named a workspace, and the classes the error matches
    today; every workspace ID (`wrkspc_…`) and request ID (`req_…`)
    redacted by pattern, and the key and each workspace sent redacted by
    value; `writeWorkspaceRecord`, one `<case>.json` per case.
  * `llmprovider/live_0064_workspace_measure_test.go` (`live_gateways`):
    `TestLive_Workspace_0064`, switched by `LLMPROVIDER_LIVE_0064`; the key
    from `ANTHROPIC_API_KEY`, the workspace from
    `LLMPROVIDER_LIVE_0064_WORKSPACE`, the foreign workspace from
    `LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE` when set, and the records'
    directory from `LLMPROVIDER_LIVE_0064_OUT`, which must be outside the
    module (`outsideModule`, 0062's). It logs each case's status, type,
    classes and whether the reply named a workspace; never a body, a
    workspace or the key. Listing bills nothing.
  * Made exact: the PLAN's "a scratch directory the run names" is
    `LLMPROVIDER_LIVE_0064_OUT`, as 0062's recorder names its own; the
    recording lives in an untagged file so that its twin runs offline.
* **Tests:** `TestWorkspaceRecord_Twin_0064`, against a local server
  answering as Anthropic did on 2026-10-25 (the MADR's evidence) and as its
  documentation says of a foreign workspace (E5): each case sends the
  header it names, or none; the records' statuses (400, 400, 404, 200),
  types, classes (`ErrInvalidRequest` for the three refusals) and reply
  header; no record holds the key, a workspace ID or a request ID; a secret
  outside the pattern is redacted by value. The live test skips without
  `LLMPROVIDER_LIVE_0064`.
* **Plants** (`tools/plant.py`, on a scratch clone; all 7 caught): the
  header not sent (`foreign: … Status:400 … not scoped to a workspace`);
  workspace IDs not redacted (`workspace.json holds "wrkspc_01Other000"`);
  the secrets not redacted by value (`redactWorkspace = "echoed
  sk-ant-test-KEY0064 …"`); request IDs not redacted (`foreign.json holds
  "req_011Twin"`); the foreign case dropped; the reply's workspace header
  not read; the classes not read. A first twin did not catch the two
  redaction plants, since either redaction hid its workspace IDs alone;
  it gained a workspace ID only the pattern catches, and a secret only the
  value catches.
* **Checks:** `make pre-add-check` on the 2 Go files: clean;
  `go vet -tags live_gateways ./llmprovider/`: clean; `make tables-check`,
  once this record names `LLMPROVIDER_LIVE_0064_OUT`; `python3 -B tools/gate.py`: "gate: 28 of 28 passed".
* **Waiting on the owner:** the run, in the owner's own terminal, with the
  key never entering the session (steps 2 and 3, the fixtures and the
  check of E5, follow it).

### Phase 1: the measurement, run (2026-10-27)

* **Runs.** The owner ran `TestLive_Workspace_0064` three times, in their
  own terminal; the key never entered the session. The records were read
  through a scratchpad script that withholds a record holding `sk-ant`, or
  a `wrkspc_` or `req_` ID not redacted; it withheld none.
  * Run 1: `no-workspace` 401 `authentication_error`, "invalid x-api-key",
    `ErrAuthFailure`; `malformed` 400 `invalid_request_error`,
    `ErrInvalidRequest`; `workspace` 401, as `no-workspace`. The owner found
    the key had expired.
  * Run 2, a new key: `no-workspace` 400 `invalid_request_error`, the MADR's
    message word for word, `ErrInvalidRequest`; `malformed` 400, as run 1;
    `workspace` 200, the reply's workspace header present.
  * Run 3, with `LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE`: the same three,
    and `foreign` 200, the reply's header present, its body byte for byte
    the `workspace` case's. The owner: that workspace was new, in the key's
    organization, and never associated with the key.
* **Deviation (2026-10-27): E5's 404 not measured.** Step 1's foreign case
  needs a workspace the key's identity may not use. Asked after run 2
  (measure it, recommended; record it as unmeasured; measure it later), the
  owner chose to measure it, and run 3's workspace answered 200. Asked again
  (a workspace of another organization, recommended; record E5 as
  unmeasured; measure it later and continue), the owner chose **record E5 as
  unmeasured**. Step 2 gains `not-found.json`, written from the
  documentation; step 3 is the MADR's amendment of 2026-10-27; D3 is
  unchanged.
* **Deviation (2026-10-27): the fixture gate.** `scripts/check_fixtures.py`
  had no kind for a workspace ID, and AGENTS.md's "Recorded fixtures" named
  none, so this PLAN's rule ("as AGENTS.md's 'Recorded fixtures' says: a
  workspace ID becomes `wrkspc_REDACTED`. `make fixtures-check` passes.")
  was not checked; the gap predates this phase. Asked (add the kind in
  Phase 1, recommended; a separate record; correct the wording only), the
  owner chose to **add it in Phase 1**. Files added to the phase:
  `scripts/check_fixtures.py`, `scripts/test_gates.py`, `AGENTS.md`, and an
  annotation of `docs/decisions/0047-MADR-keep-five-more-scratch-helpers-as-tools.md`
  D1.
* **Step 2, the fixtures,** in `llmprovider/testdata/anthropic-workspace/`:
  `no-workspace.json`, `malformed.json` and `workspace.json`, run 3's;
  `unassociated.json`, run 3's `foreign` case, named for what it was; and
  `not-found.json`, written from the documentation. Each keeps the record's
  fields but `classes`, today's classification, which Phase 2 changes and
  this record lists above, and gains `source`, saying how it was made.
* **Step 3:** the MADR's amendment of 2026-10-27 (E6 to E8, E5 still
  unconfirmed, the fixture gate), and E5 annotated in place.
* **The gate:** `check_fixtures.py` gains the kind `workspace-id`,
  `wrkspc_` then letters and digits, unless a placeholder;
  `scripts/test_gates.py` gains `test_fixtures_check_workspace_id`, and its
  placeholders test a `wrkspc_REDACTED`; AGENTS.md's "Recorded fixtures"
  names the redaction.
* **Plants** (`tools/plant.py`, on a scratch clone; all 4 caught): the
  `workspace-id` kind not scanned (`FAIL: test_fixtures_check_workspace_id
  … fixtures-check exit 0, want 1`); a placeholder workspace ID reported
  (`FAIL: test_fixtures_check_placeholders_pass … placeholders were
  reported`); every workspace ID admitted (`FAIL:
  test_fixtures_check_workspace_id`); and a well-formed workspace ID put in
  `not-found.json`, which `scripts/check_fixtures.py`, scanning the tree,
  failed on (exit 1), so the new directory is scanned.
* **Checks:** `make fixtures-check`: "380 file(s), 134 hit(s), 0
  problem(s)"; `make gate-selftest`: "Ran 38 tests … OK"; `make
  records-check`, `tables-check`, `docs-check`: clean;
  `tools/recordlint.py --changed`: "no rule rose"; `tools/denyscan.py
  --changed`: "0 hit(s) in 13 file(s)"; `tools/linkcheck.py`: "0
  problem(s)"; `python3 -B tools/gate.py`: "gate: 28 of 28 passed". The
  gate's clone predates a reflow of AGENTS.md's paragraph, after which
  `tables-check`, `records-check`, `docs-check`, the link check and the deny
  scan were run again, clean.
* **Phase 1 is complete** when the owner commits it. Phase 2 classifies
  `no-workspace.json` as `ErrWorkspaceRequired`, and `malformed.json` and
  `not-found.json` as `ErrWorkspaceNotFound`.

### Phase 2: the classes, built (2026-10-27)

* **Approval.** "committed, proceed", after Phase 1 (`18ddac8`).
* **Step 1:** `ErrWorkspaceRequired` ("llmprovider: workspace required")
  and `ErrWorkspaceNotFound` ("llmprovider: workspace not found"),
  `sentinelError`s with `ErrInvalidRequest` as parent, each documented with
  0064-MADR D3 (`llmprovider/api_error.go:47`, `:51`).
* **Step 2:** `claudeWorkspaceRefusal(service, env)`
  (`llmprovider/api_error.go:475`) is nil for any service but Claude, and
  otherwise matches by type and phrase: `invalid_request_error` with "is not
  scoped to a workspace" is `ErrWorkspaceRequired`; `invalid_request_error`
  with "must be a valid workspace ID", and `not_found_error` whose message
  begins "Workspace" and ends "not found.", are `ErrWorkspaceNotFound`.
  `classifyAPIError` binds it in its switch's init statement and returns it,
  terminal, from one case after xAI's refused key (`:409`, `:431`).
  * Made exact: the step's "two cases in `classifyAPIError`" are one case
    and the helper. Written as two cases, `make pre-add-check` failed:
    `api_error.go:405:1: cyclomatic complexity 44 of func
    \`classifyAPIError\` is high (> 40) (gocyclo)`, from 39 before. The
    helper keeps D3's matching as decided; the limit is not raised and no
    `nolint` is added.
* **Step 3:** `llmprovider/api_error_0064_test.go`:
  * `TestWorkspaceClasses_0064`: `no-workspace.json` classifies to
    `ErrWorkspaceRequired`, `malformed.json` and `not-found.json` to
    `ErrWorkspaceNotFound`, each `ErrInvalidRequest`, not the other class,
    a terminal `*APIError` with the fixture's status; each sentinel is a
    child of `ErrInvalidRequest` and an `APIError` of that kind alone is
    not retried; `workspace.json` and `unassociated.json`, the 200s, are no
    error.
  * `TestWorkspaceClasses_OnlyClaude_0064`: the three refusals from
    `openai`, `kilo` and `opencode/claude` are `ErrInvalidRequest` and no
    workspace class.
  * `TestWorkspaceClasses_OtherRefusals_0064`: a Claude 400 with another
    message, a model's 404, another object's "… not found.", "Workspace …
    is archived.", the not-found phrase under `invalid_request_error`, and
    the two 400 phrases under a 401 `authentication_error`, each keep
    today's class.
  * Made exact: the sentinels are checked apart from a reply, because an
    `APIError`'s `Unwrap` also gives a 400's or 404's legacy sentinel,
    `ErrInvalidRequest`, so a reply matches it whatever the sentinel's
    parent is. A first plant making the parent `ErrProviderUnavailable` was
    caught only by Phase 1's `TestWorkspaceRecord_Twin_0064`; the check was
    added, and the four plants below are caught by this test.
* **Step 4:** `wizard/describe.go`'s `errorClasses` lists both after
  `ErrIncomplete` and before `ErrInvalidRequest`. `wizard/describe_0064_test.go`,
  `TestDescribe_0064`: "workspace required, HTTP 400", "workspace not found,
  HTTP 400" and "workspace not found, HTTP 404".
* **Plants** (`tools/plant.py`, on a scratch clone, `-run _0064` over
  `./llmprovider/` and `./wizard/`; all 19 caught): the case gone; the
  helper for every service (`TestWorkspaceClasses_OnlyClaude_0064`); each of
  the three matches gone, without its type, and without its phrase (the
  404's prefix and suffix each); the two classes swapped; either parent
  `ErrNotPermitted`; `ErrWorkspaceRequired`'s parent
  `ErrProviderUnavailable`; the wizard missing either, or listing them after
  `ErrInvalidRequest` (`describe(…) = "invalid request, HTTP 400"`). Then,
  with `-run TestWorkspaceClasses_0064$` alone, all 4 caught: each parent
  `ErrNotPermitted`, `ErrWorkspaceRequired`'s `ErrProviderUnavailable`, and
  `ErrWorkspaceNotFound`'s `ErrRateLimited` (`want a child of
  ErrInvalidRequest, never retried`).
* **Not planted, and why:** the case's `return true, workspace` as `false`.
  `terminal` is read only by `Retryable` (`api_error.go:180`); with it false,
  `Retryable` returns true only for `ErrRateLimited` or
  `ErrProviderUnavailable` (`:187`), and both classes are `ErrInvalidRequest`,
  so nothing a caller sees changes. Planted against the first, two-case
  shape, it was not caught, as expected of an equivalent change.
* **Checks:** `make pre-add-check` on the 4 Go files: "4 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)"; `make api-check`:
  "against v1.23.0, 0 incompatible change(s)"; `python3 -B tools/gate.py`: "gate: 28 of 28 passed".

### Phase 3: the header (2026-10-27)

* **Approval.** "committed, proceed", after Phase 2 (`0375b46`).
* **Deviation (2026-10-27): the probe cache.** Before the first write, the
  Claude provider's probes were found cached by provider, base URL and
  credential, not workspace (the MADR's F8). Asked (key it by the workspace
  too, recommended; record it as a limit; a separate record), the owner
  chose to **key it by the workspace**. Step 5 is added; files added to the
  phase: `llmprovider/internal/transport/probe.go`,
  `llmprovider/internal/wire/probe.go`, and the five other providers' calls
  of `ProbeHealth` (`gemini.go`, `chatcompat.go`, `grok.go`, `ollama.go`,
  `openai.go`) with their tests' calls.
* **Step 1:** `catalog.WithClaudeWorkspace` (`llmprovider/catalog/config.go:169`),
  scoped to `ProviderClaude`; `config.ClaudeWorkspace`, set in `configFor`
  (`:204`); `fetchClaudePage` sets `anthropic-workspace-id` when it is not
  empty (`discovery.go:881`), so `ListRaw`, which lists through the same
  code, sends it too, and its doc says so.
  * Made exact: the listing's option is trimmed and refuses a control
    character with `ErrInvalidRequest`, as D1 has the provider do. D2 does
    not say; the wizard gives the listing a typed value in Phase 4, and a
    header cannot carry one.
* **Step 2:** `listingKey.organization` is `scope`, `cmp.Or` of the Kilo
  organization and the Claude workspace (`listing_cache.go:40`); each option
  is scoped to its own provider, so a key holds at most one. The Kilo
  organization test, `TestList_KiloOrganizationKeyed_0041`, passes
  unchanged.
* **Step 3:** `claude.WithWorkspace` (`claude.go:169`), scoped to
  `ProviderClaude`; `New` trims it and refuses a control character
  (`:250`), and adds `catalog.WithClaudeWorkspace` to `p.listing` when it is
  set (`:257`); `Prepare` sets the header when it is not empty (`:383`), so
  Generate, Stream and each probe, a copy of the provider, carry it.
* **Step 5:** `transport.ProbeKey.Scope` (`internal/transport/probe.go:104`);
  `wire.ProbeHealth(…, baseURL, scope, src, …)` keys by it
  (`internal/wire/probe.go:30`); the Claude provider passes `p.workspace`
  (`claude.go:508`), Gemini, chatcompat, Grok, Ollama and OpenAI `""`, as
  do the two `wire` tests that call it.
* **Step 4, the tests:**
  * `llmprovider/catalog/workspace_0064_test.go`:
    `TestClaudeWorkspace_Header_0064`, a two-page listing through `List`
    and `ListRaw`, each page naming the workspace given as `" wrkspc_A\n"`,
    and none without the option or with it empty;
    `TestClaudeWorkspace_CacheKey_0064`, one key in workspaces A and B and
    with none, three listings, and each a second time from the cache;
    `TestClaudeWorkspace_Refused_0064`, a NUL and an inner tab refused, and
    the option given to OpenAI's listing refused by name, nothing sent.
  * `llmprovider/providers/claude/workspace_0064_test.go`:
    `TestWorkspace_Header_0064`, Generate, Stream, both listing pages and
    both probes naming the workspace given as `"  wrkspc_A\n"`, and none
    without it or with `" "`; `TestWorkspace_ProbeAndListingScope_0064`,
    one key and one `catalog.Cache` in workspaces A and B, each with its own
    two pages and two probes, and each again with no request;
    `TestWorkspace_Refused_0064`, a NUL, an inner tab and a CR LF refused at
    `New`, and the option refused by OpenAI's `ResolveOptions` by name.
* **Plants** (`tools/plant.py`, on a scratch clone, `-run _0064` over
  `./llmprovider/catalog/` and `./llmprovider/providers/claude/`; all 13
  caught): generation sends no workspace; an empty workspace sent
  (`TestWorkspace_Header_0064/without`); the provider's listing not given
  it; the provider's probes not scoped (`step 1 (wrkspc_B): [{list
  wrkspc_B} {list wrkspc_B}]; want its own two pages and two probes`);
  the probe key without its scope (the same); the provider, and the
  listing, not trimming; the provider, and the listing, admitting a control
  character (`New(WithWorkspace("wrkspc_\x00A")) = <nil>`); the listing
  sending no workspace; the cache key without it (`round 0: requests "";
  want "wrkspc_B,wrkspc_B"`); `WithClaudeWorkspace` for every built-in, and
  `WithWorkspace` shared (`<nil>; want ErrInvalidRequest naming …`).
* **Checks:** `make pre-add-check` on the 16 Go files: "16 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)"; `make api-check`:
  "against v1.23.0, 0 incompatible change(s)"; `go run
  ./llmprovider/internal/cmd/sdkaudit -check`: "0 violation(s) of
  R50,R51,R52"; `python3 -B tools/gate.py`: "gate: 28 of 28 passed".

### Phase 4: the wizard, built (2026-10-27)

* **Approval.** "committed, proceed", after Phase 3 (`3ef23af`).
* **Deviation (2026-10-27): D4's pre-fill.** Before the first write, D4 was
  found to read two ways (its "pre-fills" against D4.3's first-prompt
  default and D4.1's prompt only after a refusal). Asked (pre-fill and ask
  on a refusal, recommended; list without a workspace first; confirm the
  existing workspace before listing), the owner chose to **pre-fill and ask
  on a refusal**: the MADR's amendment "D4's pre-fill". Step 4's test of the
  default gains the pre-fill: an existing workspace that lists asks nothing.
* **Step 1:** `Result.Workspace` (`wizard/configure.go:60`), its doc naming
  `claude.WithWorkspace` and `catalog.WithClaudeWorkspace`;
  `result_format.go:70` shows it. `ConfigureLLM`'s and `ResolveCredential`'s
  docs name the error after the third workspace entered.
* **Step 2:** `discoverModels` lists with
  `llmprovider.For(llmprovider.ProviderClaude,
  catalog.WithClaudeWorkspace(res.Workspace))` (`:661`), beside Kilo's
  organization, and returns a workspace refusal to `credentialStep` without
  a notice, as it returns a refused credential: `stepRefusal` (`:520`) is
  `refusedCredential` or `refusedWorkspace` (`:514`), either class.
* **Step 3:** `credentialStep` pre-fills `Existing.Workspace` when the
  provider and the existing one are Claude (`:362`); one loop (`:366`)
  answers both refusals: a workspace refusal notifies "Claude (Anthropic)
  needs a workspace for this key (<class>, HTTP <status>)", asks
  `workspacePrompt` (`:526`) with `workspaceDefault` (`:534`), trims the
  answer, and lists again with it; after `maxTypedKeys` answers it returns
  "wizard: Claude (Anthropic) refused 3 workspaces entered: %w" and no
  Result. A key refusal is handled as before.
  * Made exact: an answer holding a control character, which no header can
    carry, is not sent: a notice, "a workspace ID holds no control
    character", and it counts as an answer. An empty answer lists with no
    workspace, as the amendment says.
* **Step 4:** `wizard/workspace_0064_test.go`, against a transport that
  answers as Anthropic did on 2026-10-27, and with its documented 404 for an
  unknown workspace, through `ResolveCredential` and the scripted prompter:
  * `TestWorkspace_AskedAndListed_0064`: the first answer lists, and the
    Result keeps it with the key, live; `not-a-workspace`, then a wrong ID,
    then `"  wrkspc_Good1\n"` give notices "(workspace required, HTTP 400)",
    "(workspace not found, HTTP 400)", "(workspace not found, HTTP 404)", and
    the trimmed ID lists.
  * `TestWorkspace_Bound_0064`: three wrong IDs end with
    `ErrWorkspaceNotFound`, three blank answers with `ErrWorkspaceRequired`,
    each "refused 3 workspaces entered", no Result, four listings.
  * `TestWorkspace_ScopedKeyNeverAsked_0064`: one listing, no prompt.
  * `TestWorkspace_Defaults_0064`: an existing workspace that lists asks
    nothing; one refused is the first prompt's default and not the second's
    (listings `wrkspc_Gone1,wrkspc_Gone1,-,wrkspc_Good1`); the variable is the
    default under `AllowEnv` and not without it; another provider's existing
    workspace is neither.
  * `TestWorkspace_ControlCharacter_0064`; `TestWorkspace_NoDiscover_0064`:
    nothing listed, the existing workspace kept and shown by `String`, and a
    third party's Result with none.
  * Each asking test checks that nothing shown holds Anthropic's messages
    (0056-MADR D3).
* **Plants** (`tools/plant.py`, on a scratch clone, `-run _0064` over
  `./wizard/`; all 16 caught): no pre-fill; pre-fill for every provider;
  the listing's workspace refusal not returned; the listing not given the
  workspace; one more answer than the bound; no notice; the notice showing
  the service's message; the answer not trimmed; a control character sent;
  the answer not kept; the bound's error not wrapping; no existing default;
  no variable default; the variable without `AllowEnv`; a later prompt
  keeping the default; the Result's view without it.
  * Not planted: `workspaceDefault` trimmed the variable's value at first;
    the loop trims every answer, so it was removed rather than left as a
    change no test could see.
* **Checks:** `make pre-add-check` on the 3 Go files: "3 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)"; `make api-check`:
  "against v1.23.0, 0 incompatible change(s)"; `sdkaudit -check`: "0
  violation(s) of R50,R51,R52"; `python3 -B tools/gate.py`: "gate: 28 of 28 passed".

### Phase 5: docs and the API check, done (2026-10-27)

* **Approval.** "committed, proceed", after Phase 4 (`ee834d5`).
* **Step 1:**
  * `docs/architecture.md`: the claude row names `WithWorkspace`, the
    catalog row and the listing's options `WithClaudeWorkspace`; the
    listing cache and the probe cache are said to be keyed by the workspace;
    the errors paragraph names `ErrWorkspaceRequired` and
    `ErrWorkspaceNotFound` beneath `ErrInvalidRequest`; the wizard section
    gains "Claude's workspace".
  * `README.md`: the Claude key's paragraph says when a workspace is needed
    and how to pass it; the error kinds list both beneath
    `ErrInvalidRequest`.
  * `CHANGELOG.md`, under Unreleased, Added: the two options, the two
    classes, and `wizard.Result.Workspace`.
  * Not changed: `docs/guides/migrating-from-mcplib.md`, as D7 says; and
    `docs/guides/api-standards.md` R25, which lists the kinds each error
    unwraps to, `ErrInvalidRequest` among them, as it was left for 0061's
    `ErrEmptyAnswer` and `ErrOffline`.
* **Step 2:** `python3 -B tools/apidiff.py v1.23.0`, exit 0:

  ```text
  Compatible changes:
  - ./llmprovider/catalog.WithClaudeWorkspace: added
  - ./llmprovider/providers/claude.WithWorkspace: added
  - ./llmprovider.ErrWorkspaceNotFound: added
  - ./llmprovider.ErrWorkspaceRequired: added
  - ./wizard.Result.Workspace: added
  ```

  D6's five, and nothing else. Made exact: `make api-check` reports only
  incompatible changes ("against v1.23.0, 0 incompatible change(s) outside
  llmprovider/x/"), so the list is `tools/apidiff.py`'s.
* **Step 3**, each exit 0: `parity-check` "409 identifiers, 409 rows, 409
  with an SDK equivalent, 0 problem(s)"; `dep-check` "GOOS=windows: 42
  packages, 0 problem(s)"; `coverage-check` "41 packages, 0 problem(s)";
  `api-check` as above; `generate-check` "2 generated file(s), 0
  problem(s)"; `records-check` "137 records, 0 problem(s)"; `tables-check`
  "42 packages, 0 problem(s)"; `fixtures-check` "380 file(s), 134 hit(s), 0
  problem(s)"; `changelog-check` "34 release section(s), 0 problem(s)";
  `docs-check`, its last line "sdkaudit: 0 violation(s) of R50,R51,R52";
  `gate-selftest` "OK"; `lint` "0 issues."; `vuln` "No vulnerabilities
  found.". Then `tools/linkcheck.py` "0 problem(s), 1049 relative link(s)",
  `tools/denyscan.py --changed` "0 hit(s)", and `python3 -B tools/gate.py`
  "gate: 28 of 28 passed".
* **No plant:** this phase adds no test or gate.
* **Next:** Phase 6, the owner's live confirmation.

### Phase 6: live confirmation, the test (2026-10-27)

* **Approval.** "committed, approved, proceed", after Phase 5 (`ab6b7ae`),
  with the test proposed in Phase 5's handoff.
* **Deviation (2026-10-27): step 1 named no test.** Found at Phase 5's end:
  `TestLive_Workspace_0064` sends raw `GET /v1/models` requests; it builds no
  provider, generates nothing, and checks none of the new classes. The
  owner approved adding the test step 1 needs, with an offline twin and
  plants, before the owner's run. Files added to the phase:
  `llmprovider/workspace_0064_live_check_test.go` and
  `llmprovider/live_0064_workspace_measure_test.go`.
* **The test, as built:**
  * `llmprovider/workspace_0064_live_check_test.go` (untagged, so its twin
    runs offline): `checkWorkspaceLive(ctx, key, workspace, opts...)`. With
    no workspace and with `not-a-workspace`, `Generate` must match
    `ErrWorkspaceRequired` and `ErrWorkspaceNotFound`, each an
    `ErrInvalidRequest`, and `catalog.List` the same, a static fallback's
    error read from `Catalog.Err`; with the workspace, `Generate` answers,
    `Stream` streams text, and `catalog.List` is live with models. Each
    provider and listing has a `catalog.Cache` of its own, no probes and no
    models.dev document; the model is `claude-haiku-5-5`, at most 256 tokens.
    A problem names the classes and status (`workspaceKind`), never the key,
    the workspace or the service's text.
  * `TestLive_WorkspaceConfirm_0064` (`live_gateways`, in
    `live_0064_workspace_measure_test.go`), switched by
    `LLMPROVIDER_LIVE_0064`, with `ANTHROPIC_API_KEY` and
    `LLMPROVIDER_LIVE_0064_WORKSPACE`: it runs the checks against Anthropic
    and reports each problem with `t.Error`. Four generations, two of them
    refused, and three listings.
  * Made exact: 16 tokens became 256, in case the model reasons by default,
    which the record does not establish.
* **Twin:** `TestWorkspaceLive_Twin_0064`, against a server answering as
  measured (no problem); ignoring the workspace (four: each refusal "no
  error"); refusing every request for want of one (five, by
  `[ErrWorkspaceRequired,ErrInvalidRequest] HTTP 400`); and answering every
  named workspace with the documented 404, echoing the ID (three, by
  `[ErrWorkspaceNotFound,ErrInvalidRequest] HTTP 404`). No problem may hold
  the key, the workspace or `req_`.
* **Plants** (`tools/plant.py`, on a scratch clone; all 9 caught): the
  refusals' class not checked; the refused listing not checked; a static
  fallback's error not read; the workspace's answer, stream and listing each
  not checked; a problem showing the service's text, caught by the leak
  check, whose failure held the 404's message with the twin's workspace ID
  and said the problem "names the key, the workspace or a request"; and,
  in the SDK, generation sending no
  workspace, and the refusals not classified.
* **Checks:** `make pre-add-check` on the 2 Go files: "2 file(s) clean
  (gofmt, golangci-lint, go vet, go test, govulncheck)"; `go vet -tags
  live_gateways ./llmprovider/`: clean; `make tables-check`: "42 packages, 0
  problem(s)"; `python3 -B tools/gate.py`: "gate: 28 of 28 passed".
* **Waiting on the owner:** step 1's run, in the owner's own terminal, with
  key D; then step 2, the wizard; then step 3.

### Phase 6, step 1: the owner's run (2026-10-27)

* **Commit.** The test landed as `7d1bf9f`, committed by the owner.
* **Run.** In the owner's own terminal, with key D and its Default
  workspace: `go test -tags live_gateways ./llmprovider/ -run
  '^TestLive_WorkspaceConfirm_0064$' -count=1 -v`. The output, as the owner
  pasted it:

  ```text
  === RUN   TestLive_WorkspaceConfirm_0064
      live_0064_workspace_measure_test.go:86: no workspace: ErrWorkspaceRequired; malformed: ErrWorkspaceNotFound; the workspace: generated, streamed and listed
  --- PASS: TestLive_WorkspaceConfirm_0064 (1.99s)
  PASS
  ok      github.com/maccavelli/go-llmprovider-sdk/llmprovider    2.560s
  ```

* **Result.** Step 1 holds against Anthropic: with no workspace, `Generate`
  and the listing are `ErrWorkspaceRequired`; with `not-a-workspace`,
  `ErrWorkspaceNotFound`; with the workspace, `Generate` answers, `Stream`
  streams text and the listing is live. The run does not measure E5, a
  well-formed workspace the key may not use; it stays unmeasured.
* **Next:** step 2, the wizard, run by the owner.

### Phase 6, step 2: the wizard (2026-10-27)

* **Consumer.** A scratch module outside both trees, its `go.mod` replacing
  the SDK with the working tree at `7d1bf9f`: it runs `wizard.ConfigureLLM`
  with `Providers` Claude only, `AllowEnv` with `os.Getenv`, and `Discover`;
  prints the `Result` with the workspace replaced by `wrkspc_REDACTED`;
  then builds `claude.New` from the `Result` (`WithWorkspace(res.Workspace)`)
  and generates once. An error is printed by its classes only. It built and
  vetted clean; nothing in either tree changed.
* **Run.** In the owner's own terminal, with key D, `ANTHROPIC_API_KEY`
  exported and `ANTHROPIC_WORKSPACE_ID` unset. The output, as the owner
  pasted it:

  ```text
  result: Result{Provider:claude Kind:api_key APIKey:[redacted] Model:claude-haiku-5-5 Workspace:wrkspc_REDACTED}
  workspace returned: true
  generated with the result: true
  ```

* **Result.** The wizard returned a workspace, and a provider built from the
  `Result` generated. It asked for it: `res.Workspace` is set only from
  `Existing` (`wizard/configure.go:362`), empty here, and from the prompt
  (`wizard/configure.go:382`). The pasted lines do not show the notices
  before the menu, so whether the menu came from the live listing rests on
  the owner's report; a listing that failed otherwise warns "could not list
  models for Claude (…); using the built-in catalog"
  (`wizard/configure.go:694`).
* **The listing.** Asked, the owner answered "No such warning": before the
  menu the wizard showed only the warning that Claude needs a workspace and
  the workspace prompt, so the menu came from the live listing. Step 2
  holds: the wizard asked for the workspace, listed, and returned it.

### Phase 6, step 3: the records (2026-10-27)

* **MADR.** Already `accepted` (its front matter, and its row in
  `docs/README.md`); unchanged.
* **PLAN.** `complete`, by `tools/records.py status`, which changes its row
  in `docs/README.md` with it. Every step of Phases 1 to 6 is met; what is
  not done, and stays out of this plan: E5, the documented 404 for a
  well-formed workspace the key may not use, is unmeasured (0064-MADR,
  amendment "2026-10-27: Phase 1's measurement"); the release, `v1.24.0`,
  is the owner's to tag; prepare-commit-msg wires `Result.Workspace` under
  its own record.

### Phase 7: the release (2026-10-27)

* **Deviation (2026-10-27): `v1.24.0` was tagged with no changelog
  section.** The PLAN named no release step; its Rollout said only that the
  owner tags it, and the agent's handoff after Phase 6 called `main` ready
  to tag without checking R53's changelog rule. The owner tagged `v1.24.0`,
  annotated, on `8cf739b`. Its CI run, `40574546189`, failed on
  ubuntu-24.04 at `changelog-check`: "v1.24.0 has no section" (macos-15
  and windows-2025 passed; every other check passed on ubuntu). The
  `release` job, which needs `validate`, was skipped, and `gh release view
  v1.24.0` answers "release not found". `main`'s run on the same commit,
  `40573211074`, passed on all three. R53's "a release's section is written
  before its tag" is broken for `v1.24.0` and stays broken: its tagged tree
  has no section of its own.
  * **Evidence on the proxy:** `index.golang.org`, paged from
    2026-10-26T00:00Z to 2026-10-27T20:08:51Z, lists this module's `v1.23.0`
    (seen 9 minutes after its tag) and not `v1.24.0`.
  * **Options put to the owner:** keep the tag and release `v1.24.0` by hand
    (recommended); release `v1.24.1` through CI; move the `v1.24.0` tag,
    which rewrites a published tag and races the proxy's checksum record.
  * **The owner chose:** "Release v1.24.1 through CI".
* **Deviation (2026-10-27): R53's five newest Releases.** With `v1.24.1`,
  the five newest tags are `v1.24.1`, `v1.24.0`, `v1.23.0`, `v1.22.2` and
  `v1.22.1`; `v1.24.0` would have none, against R53. Options put: give
  `v1.24.0` its Release by hand, as 0055-MADR D3 did for `v1.20.0`
  (recommended); amend R53 with an exception; drop `v1.24.1`. **The owner
  chose:** "v1.24.0 Release by hand". The MADR's decisions and facts are
  unchanged; R53 holds.
* **Steps:**
  1. A release commit, staged by the agent: `CHANGELOG.md` gets
     `[v1.24.1] - 2026-10-27` (Fixed: the release of `v1.24.0`; no code
     change) and `[v1.24.0] - 2026-10-27` (0064's entries, from
     `[Unreleased]`, which is left empty), and their compare links; the
     README's Status names `v1.24.1`; this record. The section dates are
     the tags' commit dates, as `changelog-check` requires: `8cf739b`'s
     is 2026-10-27, and the release commit's must be.
  2. Before the push: `git diff --stat v1.24.0` shows Markdown only;
     `tools/apidiff.py v1.23.0` lists D6's five additions and `v1.24.0`
     none; both tags' notes, made on a scratch clone as the `release` job
     makes them, with every link pinned to its tag and `tools/denyscan.py`
     clean; `python3 -B tools/gate.py`; the disclosure guard over the
     outgoing commits.
  3. The owner, each in its own turn: commits and pushes `main` (Docs
     runs); tags `v1.24.1`, annotated, on the release commit (CI runs in
     full and its `release` job makes the Release); runs `gh release create
     v1.24.0 --verify-tag --title v1.24.0 --notes-file` with the scanned
     notes; deletes the Releases of `v1.22.0` and `v1.21.0`, which fall to
     sixth and seventh (R53).
  4. `tools/consumer.py v1.24.1` and `tools/offline.py v1.24.1`; then this
     PLAN `complete` again.
* **Status.** The PLAN goes back to `in-progress` for this phase.
* **Step 1, done.** `CHANGELOG.md`: `[v1.24.1] - 2026-10-27`, Fixed,
  and `[v1.24.0] - 2026-10-27`, 0064's three Added entries, under an empty
  `[Unreleased]`; the links `[Unreleased]` from `v1.24.1`, `[v1.24.1]` from
  `v1.24.0` and `[v1.24.0]` from `v1.23.0`. The README's Status names
  `v1.24.1`. `make changelog-check`: "36 release section(s), 0
  problem(s)"; `v1.24.1`'s section is newer than every tag, as a release
  being prepared may be.
* **Step 2, done:**
  * `git diff --name-only v1.24.0`: `CHANGELOG.md`, `README.md`,
    `docs/README.md` and this PLAN, Markdown only;
    `tools/apidiff.py v1.24.0`: exit 0, no change; `tools/apidiff.py
    v1.23.0`: exit 0, D6's five compatible additions (`claude.WithWorkspace`,
    `catalog.WithClaudeWorkspace`, `wizard.Result.Workspace`,
    `ErrWorkspaceNotFound`, `ErrWorkspaceRequired`) and no incompatible
    change;
  * the notes, on a scratch clone with the working tree's `CHANGELOG.md`
    and `README.md`, `v1.24.0` fetched and a local lightweight `v1.24.1` on
    its `HEAD`: `scripts/check_changelog.py --section TAG --repo
    maccavelli/go-llmprovider-sdk`, exit 0 for each; `v1.24.0` 18 lines, 3
    links, `v1.24.1` 7 lines, 1 link, each pinned to its own tag, none
    relative; `tools/denyscan.py` on them, 0 hits;
  * `tools/recordlint.py --changed`: no rule rose; `tools/denyscan.py
    --changed`: 0 hits in 4 files; `tools/linkcheck.py`: "0 problem(s),
    1050 relative link(s)"; `python3 -B tools/gate.py`: "gate: 28 of 28
    passed";
  * the disclosure guard runs over the release commit once the owner has
    committed it.
