---
status: in-progress
date: 2026-11-24
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
~~This PLAN is `proposed`: nothing starts until the owner approves it and says
`proceed`, `execute the plan` or `do phase N`.~~ *(2026-11-24: approved, "Approve;
build after 1b and 1c (Recommended)"; this PLAN is `in-progress`; see the
Execution Record.)*

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

~~Nothing is executed.~~ *(2026-11-24: Phase 0 step 3 to Phase 3 executed, below.)*
Each phase's passes are numbered here, with the gate output and a dated
entry for every deviation.

### 2026-11-24: Phase 0 step 3, the approval

The owner's answer, relayed by the coordinator, verbatim: **"Approve; build
after 1b and 1c (Recommended)"**. This PLAN is approved and built after
0088-PLAN's Phase 1b and Phase 1c, both committed (`f4efc5e`, `903c387`), in
the same tree, one at a time, each its own commit; the joint checks, review
and release run once over everything afterwards. `python3 -B tools/records.py
status docs/decisions/0090-PLAN-a-cancelled-run-does-not-ask-about-the-endpoint.md
in-progress --date 2026-11-24`: "records: written; records-check: 0
problem(s)" (this file's status and its `docs/README.md` row). Base:
`903c387`, `git status` clean before the first write.

Re-grounding at `903c387` is in the MADR, "Execution notes (2026-11-24)":
claims 7 of 7; two cited lines moved by one (re-pinned there); no fact
changed. The search for "wizard: cannot reach" (step 1 and step 3) is
recorded there too.

### 2026-11-24: Phase 1, pass 1 (S78)

**Tests** (`wizard/resolve_base_url_0090_test.go`, new; ten functions, six
subtests; each calls `resolveBaseURL` with a client whose transport answers
or fails as its row needs, nothing sent over the network):

| Row | Test | At the base | After |
| :--- | :--- | :--- | :--- |
| T1 | `TestResolveBaseURL_UnreachableAlive_0090` | PASS (pin) | PASS |
| T5 | `TestResolveBaseURL_AnsweredAlive_0090` | PASS (pin) | PASS |
| T6 | `TestResolveBaseURL_NoPromptUnreachableAlive_0090` | PASS (pin) | PASS |
| T4 | `TestResolveBaseURL_EndedBeforeAsk_0090` | PASS (pin) | PASS |
| T8 | `TestResolveBaseURL_ProbeBoundAlive_0090` | PASS (pin) | PASS |
| T10 | `TestResolveBaseURL_EndedAfterAnswer_0090` | PASS (pin) | PASS |
| T2 | `TestResolveBaseURL_CancelledProbe_0090` (in flight; as the endpoint is entered) | FAIL | PASS |
| T3 | `TestResolveBaseURL_DeadlineProbe_0090` | FAIL | PASS |
| T7 | `TestResolveBaseURL_NoPromptEnded_0090` (a connect failure; a non-200 answer) | FAIL | PASS |
| T15 | `TestResolveBaseURL_AnsweredEnded_0090` (cancelled; deadline) | FAIL | PASS |

The red run, `go test -count=1 -v -run '_0090$' ./wizard/` with the new file
and the base `configure.go`, exit 1, each on its assertion (a line each):

```text
resolve_base_url_0090_test.go:208: notices ["cannot reach http://127.0.0.1:11434 (provider unavailable)"]; want none after the run ended
resolve_base_url_0090_test.go:241: err = fakePrompter: unexpected Confirm("Try a different endpoint?"); want one matching context deadline exceeded
resolve_base_url_0090_test.go:272: err = wizard: cannot reach http://localhost:11434: llmprovider: provider unavailable: ollama HTTP 500; want "wizard: endpoint check ended: context canceled"
resolve_base_url_0090_test.go:297: notices ["http://127.0.0.1:11434 answered: provider unavailable, HTTP 500"]; want none after the run ended
```

In T2's second subtest the transport was called once with the context
already done, as the MADR says a custom `RoundTripper` is.

**The change** (`wizard/configure.go`, `resolveBaseURL` only): after a
failed interactive check, before the "answered" / "cannot reach" split,
`if err := ctx.Err(); err != nil { return "", fmt.Errorf("enter base URL:
%w", err) }`; under `NoEndpointPrompt`, after a failed check,
`fmt.Errorf("wizard: endpoint check ended: %w", err)`. Each carries a comment
citing 0090-MADR S78 and 0081-MADR D4. Green: the ten functions, 16 PASS
lines; `go test -count=1 ./wizard/` exit 0.

**Plants** (`tools/plant.py`, one spec per plant, each running only its own
test, on a scratch clone of the working tree): 10 of 10 caught.

| Row | Plant | The failing line |
| :--- | :--- | :--- |
| T1 | the interactive guard unconditional | `resolve_base_url_0090_test.go:86: resolveBaseURL = "", enter base URL: %!w(<nil>); want "http://127.0.0.1:11434", nil` |
| T5 | the same | `resolve_base_url_0090_test.go:105: resolveBaseURL = "", enter base URL: %!w(<nil>); want "http://127.0.0.1:11434", nil` |
| T6 | the no-prompt guard unconditional | `resolve_base_url_0090_test.go:126: err = wizard: endpoint check ended: %!w(<nil>); want one beginning "wizard: cannot reach http://localhost:11434: "` |
| T4 | the loop-top check removed | `resolve_base_url_0090_test.go:151: asked ["Ollama endpoint"] and checked 0 time(s); want neither` |
| T8 | the guard reads the error's class (`errors.Is` on `context.DeadlineExceeded` or `context.Canceled`) | `resolve_base_url_0090_test.go:165: resolveBaseURL = "", enter base URL: llmprovider: provider unavailable: could not reach Ollama at http://127.0.0.1:11434: …: the check's own bound: context deadline e` (cut by the tool) |
| T10 | the guard placed before the success check | `resolve_base_url_0090_test.go:188: resolveBaseURL = "", enter base URL: context canceled; want "http://127.0.0.1:11434", nil: the check succeeded` |
| T2 | the interactive guard removed | `resolve_base_url_0090_test.go:208: err = fakePrompter: unexpected Confirm("Try a different endpoint?"); want one matching context canceled` |
| T3 | the same | `resolve_base_url_0090_test.go:241: err = fakePrompter: unexpected Confirm("Try a different endpoint?"); want one matching context deadline exceeded` |
| T7 | the no-prompt guard removed | `resolve_base_url_0090_test.go:272: err = wizard: cannot reach http://localhost:11434: llmprovider: provider unavailable: ollama HTTP 500; want "wizard: endpoint check ended: context canceled"` |
| T15 | the guard only in the connect-failure branch | `resolve_base_url_0090_test.go:297: notices ["http://127.0.0.1:11434 answered: provider unavailable, HTTP 500"]; want none after the run ended` |

**Docs step.** `docs/architecture.md`, the `Options.NoEndpointPrompt`
bullet of the wizard's section: one sentence on a run that ends during the
Ollama endpoint check (T14), pinned by the MADR's claims row E8now (checked
and planted false there: 2 of 2 caught). `CHANGELOG.md` `[Unreleased]`
**Fixed**: one entry (T11). `make changelog-check` exit 0.

**Gates of the phase:** `gofmt -l wizard/` (nothing), `go vet ./wizard/`
exit 0, `golangci-lint run -c .golangci.yml ./wizard/...` exit 0,
`make api-check docs-check changelog-check` exit 0 ("against v1.32.0, 0
incompatible change(s) outside llmprovider/x/"). The suite and the heavy
gates ran once over all three phases' final tree, below.

### 2026-11-24: Phase 2, pass 1 (P-20, T12)

Re-grounding: the bullet (`docs/architecture.md`, "When a session is
saved") holds "measured kept on 2026-11-19, that is, a second sign-in did
not end the first session"; the check was `{"the premise was measured
kept", "kept"}` at `wizard/save_at_end_0081_test.go:695`.

The plant (`tools/plant.py`, `docs/architecture.md`, run
`^TestSaveNotesInTheArchitecture_0081$`): the sentence's "kept on
2026-11-19, that is, a second sign-in did not end the first session" made
"on 2026-11-19", and "An abandoned run keeps the session" made "An abandoned
run kept the session", so "kept" stays in another clause and the date stays.
With the old check: "NOT CAUGHT (exit 0)". The change: the one `want` entry
now reads the whole sentence, "was measured kept on 2026-11-19, that is, a
second sign-in did not end the first session"; nothing else in the test.
Green on the tree (`--- PASS: TestSaveNotesInTheArchitecture_0081`). With the
new check, the same plant: caught (exit 1):

```text
save_at_end_0081_test.go:700: "When a session is saved" does not say the premise was measured kept ("was measured kept on 2026-11-19, that is, a second sign-in did not end the first session"); it reads: A new session is saved once the check
```

Gates of the phase: `gofmt -l`, `go vet ./wizard/` exit 0; the rest with
the final tree, below.

### 2026-11-24: Phase 3, pass 1 (P-16, T13)

**Step 1.** Every string literal matching `*_TEST_*` in a test file at
`903c387`: the six the MADR names, no other. None was named by `AGENTS.md`;
0081-PLAN names the two 0081 ones; no 0087 record names 0087's. `AGENTS.md`
"Live tests" gained a paragraph naming all six, with the files that hold
them, before the script change.

**Step 2, red.** `scripts/test_gates.py`
`GateSelfTest.test_tables_check_test_variable_unnamed` (a scratch tree's
`llmprovider/zz_planted_0090_test.go` holds `"FOO_TEST_9999_X"`: exit 1
naming it; then `docs/decisions/9999-MADR-zz-planted.md` names it: exit 0).
Before the script change, FAILED: `AssertionError: 0 != 1 : tables-check
exit 0, want 1: check-tables: 51 packages, 0 problem(s)`. After: OK.

**Step 3.** `scripts/check_tables.py`: `check_test_variables`, over the
`*_test.go` files `git ls-files -co --exclude-standard` lists, by the
literal; a variable is admitted when `AGENTS.md` names it as a whole word,
or when its name carries a four-digit number (`_NNNN_` or a final `_NNNN`)
and a record of that number names it as a whole word; the docstring says
so. `make tables-check`: "check-tables: 51 packages, 0 problem(s)", exit 0.

**Step 4, plant** (`tools/plant.py`, scratch clone, `make tables-check`):
`helperStore0081` renamed to `"SDK_TEST_0081_STORAGE"` in the test only:
caught (exit 2), with

```text
check-tables: AGENTS.md: SDK_TEST_0081_STORAGE: wizard/save_at_end_0081_test.go holds it, and neither AGENTS.md nor a record of its number names it
```

**Step 5, docs.** `AGENTS.md`'s new paragraph states the rule in its last
sentence; `docs/architecture.md`'s file map says `tables-check` checks the
test-only variables too.

**Step 7, the shell hazard.** Each child part now runs only when the parent
launched it: the parent writes a marker file with a fixed text in its own
temporary directory, and the child reads it (`keyHelperLaunched`,
`proxyLaunched`, `helperLaunched0081`, `ptyLaunched0087`). The value the
child is handed is unchanged except `LLMPROVIDER_TEST_PROXY_CHILD`, which
was `1` and now names the marker. A stray-value test per file re-runs its
own binary with the variable set to a value no parent marked:

| File | Test | Red at the base (the marker check absent) | After |
| :--- | :--- | :--- | :--- |
| `llmprovider/providers/claude/reauth_test.go` | `TestHelperClaudeKeyCommand_StrayValue_0090` | `the stray value ran the key command: it wrote <tmp>/runs` | PASS (`--- SKIP: TestHelperClaudeKeyCommand` in the child) |
| `llmprovider/providers/proxy_test.go` | `TestDefaultClient_HonoursProxy_StrayValue_0090` | `the stray value ran the child part: 10 request(s) reached the shell's proxy` | PASS |
| `wizard/save_at_end_0081_test.go` | `TestKilledAtTheModelQuestion_StrayValue_0090` | `the stray value ran the child part: it reached the model question` | PASS |
| `wizard/late_lf_0087_windows_test.go` | `TestPtyChild_StrayValue_Windows_0090` | `the stray value ran the child part: it wrote ["secret.ready"] in the stray directory` | PASS |

The Windows file's red run was on a scratch clone of the working tree with
the file's three child gates and the marker write put back as at the base
and the stray test kept, through `run-on-host --need windows` with `sh -c`
as the first word, exit 1; its green run, from the tree, ran the five
Windows tests, all PASS. The parents' own tests pass with the markers
(`TestHelperClaudeKeyCommand` skips as a test,
`TestClaude_ConcurrentRefusalsRunCommandOnce`,
`TestDefaultClient_HonoursProxy`,
`TestKilledAtTheModelQuestionKeepsTheStore_0081`, the three Windows pty
tests).

Plants (`tools/plant.py`, scratch clone; each helper's last line made
`return true`, so any value is taken as the parent's; the Windows plant's
command is `run-on-host --need windows`): 4 of 4 caught, each failing only
its own stray test, with the red lines above.

**Gates of the phase:** `make tables-check` exit 0; `GOOS=windows go vet` on
the three packages exit 0; the rest with the final tree, below.

#### Deviation (2026-11-24): one lint finding, fixed

The first `make pre-add-check` failed (exit 2) on staticcheck SA4006, "this
value of err is never used", at the 60 s and 90 s give-up branch of the two
wizard stray tests (`err = <-done` before `t.Fatalf`). Fixed by draining
the channel without assigning (`<-done`); no suppression. The second run
was clean. The branch is the give-up path only; the Windows tests were run
again after it, all PASS.

### 2026-11-24: the final tree's gates

Exit statuses captured before any filter.

| Check | Host | Exit |
| :--- | :--- | :--- |
| `go build ./...`, `go vet ./...`, `go vet -tags live_gateways ./...`, `GOOS=windows go vet -tags live_gateways ./...` | this Mac | 0 |
| `run-heavy -- make pre-add-check FILES="<the six Go files>"` | this Mac | 2, then 0 after the deviation above ("6 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)") |
| `run-heavy -- go test -count=1 ./...` | this Mac | 0 (46 packages `ok`, none failed) |
| `run-on-host --need linux --race -- go test -race -count=1 ./wizard/ ./llmprovider/providers/ ./llmprovider/providers/claude/` | the Linux machine | 0 |
| `run-on-host --need windows -- sh -c 'go test -count=1 -v -run "_Windows_00(87\|90)$\|_0090$\|_0081$" ./wizard/'` | the Windows laptop | 0 (48 PASS) |
| `run-heavy -- make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest tools-test` | this Mac | 2 (markdownlint MD013 on the new CHANGELOG entry, one 513-character line; rewrapped), then 0 (gate-selftest "Ran 71 tests", OK; tools-test "Ran 116 tests", OK) |
| `run-heavy -- python3 -B tools/gate.py` | this Mac | 0 ("gate: 30 of 30 passed") |
| `python3 -B tools/apidiff.py v1.32.0` | this Mac | 0: the 21 "Compatible changes" lines of 0088 and 0089, no other; nothing in `wizard` |
| `python3 -B tools/claims.py` on the MADR | this Mac | 0 (7 of 7) |
| `python3 -B tools/recordlint.py --changed` | this Mac | 0 (both 0090 records: no new findings) |
| `python3 -B tools/denyscan.py --changed` | this Mac | 0 (0 hits in 14 files) |

### 2026-11-24: Phase 1, pass 2, after review

An independent review (relayed by the coordinator) passed the build with
notes. Its first note: T3's and T15's deadline tests (a 50 ms
`WithTimeout`) never required that the check ran, so a deadline passing
before the loop would end the run at the loop's top, T4's path, and not at
the guard after a failed check. The change: both tests count the
transport's calls and require at least one (`assertProbed0090`), as T2's
second subtest counts. Nothing else changed; `wizard/configure.go` is as
pass 1 left it.

Plants (`tools/plant.py`, scratch clone, one spec each, each running only
its own test), 4 of 4 caught:

* The guard after a failed check removed, run on T3: caught
  (`resolve_base_url_0090_test.go:255: err = fakePrompter: unexpected
  Confirm("Try a different endpoint?"); want one matching context deadline
  exceeded`); run on T15: caught (`:311` and `:324`, the notice
  "answered: provider unavailable, HTTP 500"; want none).
* The loop-top check made to fire first (`<-ctx.Done()` before the loop),
  run on T3 and on T15's `deadline` subtest: caught on the new assertion,
  `resolve_base_url_0090_test.go:254` and `:323`: "the check was made 0
  time(s); want at least 1, so the guard after a failed check is what ended
  the run". Each run also reports `fakePrompter: 1 scripted inputs left
  unused` (newFake's leftover check), which would have failed that path in
  pass 1 too; the count makes the requirement explicit and names it.

Gates: `go vet ./wizard/` exit 0; `go test -count=1 ./wizard/` exit 0;
`run-on-host --need linux --race -- go test -race -count=1 ./wizard/` on the
Linux machine, exit 0; `make records-check markdownlint` exit 0;
`tools/denyscan.py --changed` exit 0.

### Follow-ups (2026-11-24, from the review; not built)

* `scripts/check_tables.py`'s test-variable check runs one way only: a
  variable `AGENTS.md` still names after its test is gone is not reported,
  though the MADR's P-16 Context speaks of a variable "added or dropped".
* Its `TEST_VAR` matches only a double-quoted literal; a raw-string
  (backquoted) literal is not seen.
* After the guard after a failed check (`wizard/configure.go:869`), the
  run's context can still end before `Notify` or `Confirm` is called: a
  narrow window outside 0090's rows.
* `Options.NoEndpointPrompt`'s doc comment (`wizard/configure.go:245`) is
  still true but does not mention an ended run's error,
  "wizard: endpoint check ended: ..."; it is outside this PLAN's scope
  (`resolveBaseURL` only).

Status stays `in-progress`: the joint checks, the review and the release
run once over 0088, 0089 and this record afterwards.
