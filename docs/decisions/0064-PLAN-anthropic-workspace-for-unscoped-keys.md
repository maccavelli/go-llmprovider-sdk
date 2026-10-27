---
status: in-progress
date: 2026-10-26
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
