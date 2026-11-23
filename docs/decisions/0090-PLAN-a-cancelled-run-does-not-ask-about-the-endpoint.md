---
status: proposed
date: 2026-11-23
associated-madr: "0090-MADR-a-cancelled-run-does-not-ask-about-the-endpoint.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement: A Cancelled Run Does Not Ask About the Endpoint

Associated MADR: [0090-MADR-a-cancelled-run-does-not-ask-about-the-endpoint.md](0090-MADR-a-cancelled-run-does-not-ask-about-the-endpoint.md)

## Goal

After this PLAN, a run whose context has ended during the Ollama endpoint
probe shows no "cannot reach" notice and asks no "Try a different endpoint?"
question, in both modes ("Stop at once, both modes (Recommended)"); the
"kept" check of the architecture-note test pins its whole sentence ("Match
the whole sentence (Recommended)"); the six test-only variables' names are
gated by `scripts/check_tables.py` and each of four test files' child branch
refuses to run unless its parent launched it ("Gate names + close the
hazard"); no exported name changes. The fix rides in v1.33.0 ("Inside
v1.33.0"): Phase 1 starts only after the SDK's merged output-limit phase
(0088/0089's build in this tree) is done, serially, and the v1.33.0 tag waits
for all three phases.

The MADR is `accepted` (2026-11-23, the owner's four answers; 0 rows open).
This PLAN is `proposed`: nothing starts until the owner approves it and says
`proceed`, `execute the plan` or `do phase N`.

## Scope

* `wizard/configure.go`: `resolveBaseURL` only (Phase 1).
* `wizard/resolve_base_url_0090_test.go` (new), `wizard/save_at_end_0081_test.go`
  (one check, Phase 2).
* `scripts/check_tables.py`, `scripts/test_gates.py`, `AGENTS.md` (Phase 3).
* The four test files that hold the six variables, edited in Phase 3 step 7:
  `wizard/save_at_end_0081_test.go`, `wizard/late_lf_0087_windows_test.go`,
  `llmprovider/providers/proxy_test.go`,
  `llmprovider/providers/claude/reauth_test.go`.
* `docs/architecture.md`, `CHANGELOG.md` (a Fixed entry), `docs/README.md`.
* Not in scope: any exported name; `catalog.ValidateOllamaURLWith`; the
  prompter; the listing warnings (0081-MADR D4); any other provider.

## Rules for Every Phase

* **Red, then green.** A test marked red is run at the phase's base and must
  fail on its assertion, not on a build error; a pin marked green passes at
  the base. After the change, all pass.
* **Plants.** Each is one production change on a scratch clone, never the
  tree; it must turn its test red; the execution record quotes the failing
  line.
* **Heavy runs.** `python3 -B tools/gate.py` and `make pre-add-check` go
  through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`,
  or another host through `run-on-host`, one at a time per host; never
  `-race` on WSL. A single `go test ./wizard` may stay on the session host.
* **Staged only.** Each phase ends with its files staged; the owner commits.
* **Stop conditions:** an existing test needs a change (except the one edit Phase 2 names, in
  `wizard/save_at_end_0081_test.go`, and the four test files Phase 3 step 7
  edits); `apidiff` reports any
  change; a red test cannot be red at the base without a production change; a
  row's outcome cannot be met as written; a third build pass of one phase.

## Implementation Steps

### Phase 0: records

1. This pair and its index rows (the coordinator adds them to
   `docs/README.md`).
2. The owner answers the MADR's four questions; the open rows become decided
   by dated amendment; the MADR becomes `accepted`.
3. The owner approves this PLAN (`tools/records.py status` to `in-progress`).

Exit criterion: no blocker, every row tested, plants caught (here: no row is
`open`, and each decided row names its test and plant).

### Phase 1: the endpoint probe (S78)

1. **Re-ground** at the phase's start commit: re-read `resolveBaseURL`, its
   two probe sites, `describe`, and 0081-MADR's S78 and D4; search this
   repository (code and tests) for a match on the text "wizard: cannot reach"
   and record the result. Record any moved line.
2. **Tests first**, in `wizard/resolve_base_url_0090_test.go`, over
   `fakePrompter` and a transport that answers or fails as the case needs:

   | Test | Rows | Base | Plant (on the fixed code) |
   | :--- | :--- | :--- | :--- |
   | `TestResolveBaseURL_UnreachableAlive_0090` | T1 | green | the new guard unconditional |
   | `TestResolveBaseURL_AnsweredAlive_0090` | T5 | green | as above |
   | `TestResolveBaseURL_NoPromptUnreachableAlive_0090` | T6 | green | the no-prompt guard unconditional |
   | `TestResolveBaseURL_EndedBeforeAsk_0090` | T4 | green | the loop-top check removed |
   | `TestResolveBaseURL_ProbeBoundAlive_0090` | T8 | green | the guard reads the error's class (`errors.Is(err, context.DeadlineExceeded)`) |
   | `TestResolveBaseURL_EndedAfterAnswer_0090` | T10 | green | the guard placed before the success check |
   | `TestResolveBaseURL_CancelledProbe_0090` | T2 | red | the guard removed |
   | `TestResolveBaseURL_DeadlineProbe_0090` | T3 | red | the guard removed |
   | `TestResolveBaseURL_NoPromptEnded_0090` | T7 | red | the no-prompt guard removed |
   | `TestResolveBaseURL_AnsweredEnded_0090` | T15 | red | the guard only in the connect-failure branch (it must turn this red) |

   Each red test asserts: the error matches the context's error, `Notify` was
   never called, and `Confirm` was never asked (the fake records both);
   `NoPromptEnded` asserts the text "wizard: endpoint check ended: ..." and
   never "cannot reach", in subtests for a connect failure and a non-200
   answer. T2 has two subtests: the cancel arrives while the probe is in
   flight, and `Input` cancels the context as it returns its answer. In both,
   the test's transport fails the request whether or not the context is done
   (an `http.Client` still calls a custom `RoundTripper` with a done
   context; one answering 200 would make the case T10, which
   `EndedAfterAnswer` covers). Ten test functions in all. The red run is at
   the base; the failure is the fake's recorded "cannot reach" notice, not a
   build error.
3. **The change** (the owner's "both modes"): after each failed probe, before any
   notice or wrapping, in the loop
   `if err := ctx.Err(); err != nil { return "", fmt.Errorf("enter base URL: %w", err) }`
   (the loop-top's wording), covering both the connect-failure and the
   non-200 branch (one guard before the `errors.As` split); and in the
   `NoEndpointPrompt` branch the same check returning
   `fmt.Errorf("wizard: endpoint check ended: %w", err)` (MADR T7; nothing was
   entered there). A comment cites 0090-MADR S78 and 0081-MADR D4.
   Re-ground also (step 1): the no-prompt wording matches no caller in
   prepare-commit-msg and gobble-cli (search both for "cannot reach").
4. **Docs step:** one sentence in the endpoint passage of
   `docs/architecture.md` (T14); a `tools/claims.py` row on it, planted false.
   `CHANGELOG.md`: a **Fixed** entry under `[Unreleased]` (T11: the fix
   ships in v1.33.0); `make changelog-check` passes.
5. **Gates:** `gofmt -l .`; `go vet ./...`; `golangci-lint run -c .golangci.yml
   ./wizard/...`; `make api-check` (no change) and `make docs-check`; the
   suite through `run-heavy -- python3 -B tools/gate.py`. Capture each
   status before any filter.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 2: the "kept" check (P-20, "Match the whole sentence")

1. **Re-ground:** re-read the bullet in `docs/architecture.md` and the test.
2. **Test first (on a scratch copy of the document):** the new check, "was
   measured kept on 2026-11-19, that is, a second sign-in did not end the
   first session", is green on the tree. Plant: the sentence deleted and the
   word "kept" added to another clause of the bullet; the old check passes
   there, the new one fails. Both outcomes are quoted.
3. **The change:** replace the one `want` entry; nothing else in the test.
4. **Gates:** as Phase 1 step 5, for `./wizard/...` and `make docs-check`.

Exit criterion: no blocker, every row tested, plants caught.

### Phase 3: test-only variables (P-16, "Gate names + close the hazard")

1. **Re-ground:** list every string literal in a `_test.go` file that matches
   `*_TEST_*` and is not `LLMPROVIDER_LIVE_*` (they are read through
   constants, so scan literals, not `os.Getenv` calls); confirm the six named
   in the MADR (`SDK_TEST_0081_STORE`, `SDK_TEST_0081_SIGNAL`,
   `WIZARD_TEST_PTY_CHILD_0087`, `WIZARD_TEST_PTY_DIR_0087`,
   `LLMPROVIDER_TEST_PROXY_CHILD`, `LLMPROVIDER_TEST_CLAUDE_KEY_HELPER`) and
   any others. None is named by a document at 6ace750: each gets a line in
   `AGENTS.md` (a record may name only a variable carrying its own number),
   before the script change.
2. **Test first:** a `scripts/test_gates.py` case in which a scratch tree
   reads `FOO_TEST_9999_X` in a test and names it nowhere: `check_tables.py`
   exits 1 naming it; with a record numbered 9999 that names it, exit 0. It is
   seen to fail before the script change and pass after.
3. **The change:** `scripts/check_tables.py` adds the rule of the MADR's
   question 4 (a string literal matching `*_TEST_*` in a test file is named
   as a whole word by `AGENTS.md` or by a record whose number is in its name),
   and updates its docstring. `make tables-check` passes on the tree.
4. **Plant** (scratch clone): `SDK_TEST_0081_STORE` renamed in the test only;
   `make tables-check` exits 1.
5. **Docs step:** the `AGENTS.md` "Live tests" or tests passage states the
   rule in one sentence.
6. **Gates:** `make tables-check`, `make gate-selftest`,
   `make docs-check`, through `run-heavy` where heavy.
7. **The shell hazard (T13):** each of the four test files
   (`wizard/save_at_end_0081_test.go`, `wizard/late_lf_0087_windows_test.go`,
   `llmprovider/providers/proxy_test.go`,
   `llmprovider/providers/claude/reauth_test.go`) makes its child branch
   refuse to run unless its parent launched it: it requires a value only its
   parent writes (a path under the parent's temporary directory holding a
   marker the parent created). Red first, a test per file: the variable is
   set to a stray value in the parent and the parent branch must run; red at
   the base (the child branch runs). Plant per file: the marker check
   removed, which turns that test red. These four are the exceptions to the
   stop on editing existing tests; the launch code of each parent test
   changes only to write the marker. Windows-only 0087 file: its test runs
   through `run-on-host --need windows`, with `sh -c` as the first word.

Exit criterion: no blocker, every row tested, plants caught.

## Verification

* Every decided row of the MADR's state table has its test green and its
  plant caught, with the output quoted in the execution record.
* The four red tests of Phase 1 (T2, T3, T7, T15) and the four stray-value
  tests of Phase 3 are seen red at the base and green after.
* `make api-check` reports no change; `go mod tidy -diff` is clean;
  `make records-check` and `make docs-check` pass; the index count in
  `docs/README.md` matches.

## Rollout and Rollback

* Rollout: `wizard` text and control flow only, plus a test and a gate; a
  consumer takes it by bumping the module. The fix ships in v1.33.0 with
  0088 and 0089 (MADR T11, "Inside v1.33.0"); it is built after their merged
  output-limit phase, in the same tree, serially, and the tag waits for it.
* Rollback: revert the phase's commit. Nothing persisted depends on it.

## Execution Record

Nothing is executed. Each phase's passes are numbered here, with the gate
output and a dated entry for every deviation.
