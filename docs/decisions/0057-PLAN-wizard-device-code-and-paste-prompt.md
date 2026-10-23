---
status: in-progress
date: 2026-10-22
associated-madr: "0057-MADR-wizard-device-code-and-paste-prompt.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Wizard's Device Code as Data, Browser Sign-In Without a Paste Prompt, and the End of the Paste Prompt Once the Loopback Wins

Associated MADR: [0057-MADR-wizard-device-code-and-paste-prompt.md](0057-MADR-wizard-device-code-and-paste-prompt.md)

## Goal

The MADR's D1 to D5, with the owner's answers to Q1 and Q2 (2026-10-22)
and to Q3 to Q5 once given, so that each request of issues
[6](https://github.com/maccavelli/go-llmprovider-sdk/issues/6),
[7](https://github.com/maccavelli/go-llmprovider-sdk/issues/7) and
[8](https://github.com/maccavelli/go-llmprovider-sdk/issues/8) is a passing
test:

* `Options.ShowDeviceCode` gives a device sign-in's code and expiry to the
  caller as a `DeviceCode` (D1, issue 6);
* `Options.NoPastePrompt` runs a browser sign-in on the loopback alone
  (D2, issue 7);
* an empty paste is asked again (D4, issue 7's side request);
* a `Prompter` that implements `InputerContext` has its paste prompt ended
  once the loopback wins, with no Enter (D3, issue 8);
* `v1.22.0` is released (D5).

## Scope

### In scope

* **`wizard/`:** `auth.go`, `configure.go` (`Options`, and the docs of
  `ConfigureLLM` and `ResolveCredential`), `prompter.go`,
  `text_prompter.go` (its doc only), a new `device.go`, and new tests named
  `*_0057_test.go`.
* **Three existing tests, moved to the new device seam** (MADR F3, D1),
  each keeping what it asserts (R46):
  * `wizard/auth_test.go` (the stub at line 206);
  * `wizard/kilo_device_test.go` (line 54);
  * `wizard/remediation_test.go` (line 245).
* **Docs:** `docs/architecture.md` ("The wizard"), `CHANGELOG.md`, the
  README's Status. `docs/guides/migrating-from-mcplib.md` is not changed:
  no new name replaces an `mcplib` identifier (R43).
* **Records:** this pair, and its index rows.
* **The issues:** a reply to each and its closing, on the owner's word.

### Out of scope

* Any exported name other than these four:

  | New name | Kind | Call site (R55) |
  | :--- | :--- | :--- |
  | `wizard.DeviceCode` | type | gobble's `auth login --json`, whose `device_code` event it fills |
  | `wizard.Options.ShowDeviceCode` | field | the same, and gobble's `/login`, which prints the code itself |
  | `wizard.Options.NoPastePrompt` | field | gobble's `auth login` with no terminal or with `--json` |
  | `wizard.InputerContext` | interface | gobble's `/login`, first run and `gobble setup`, over its `editor.Reader` |

* Any change in `llmprovider/auth` (MADR DC3).
* `TextPrompter` implementing `InputerContext` (MADR Q2, IC5).
* A context on any prompt other than the paste prompt (MADR D3).
* gobble's own change, which its repository makes.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase, once Q3 to
  Q5 are answered. A phase starts after the owner has committed the last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `tools/claims.py` on both records;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter.
* **Red first.** Every new test is seen to fail, on today's code or on a
  planted break, by `tools/plant.py` on a scratch clone, never in the tree.
  The phase's record holds each plant and its failure line. A test that can
  hang under a plant (Phase 3) bounds its own wait, and fails with a
  message rather than by the test binary's timeout.
* **The drift check.** After each phase's code and before its new tests,
  `go test -race -count=1 ./wizard/` passes with every existing test
  unchanged. The one exception is Phase 1's three files, named above.
* **The docs.** A function whose behaviour changes has its doc changed in
  the same phase, naming only what a test shows (R52). `make docs-check`
  stays at 0 violations.
* **The API.** `tools/apidiff.py v1.21.0` lists only the names a phase adds,
  from the table above.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * a name outside the table would be added;
  * an existing test would change for a reason no step names;
  * a doc would state a behaviour no test shows.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. The owner's answers to Q3 to Q5 are written into the MADR's "Owner
   questions". Where an answer differs from the recommended one, the
   decision it bears on (D1, D2 or D4) is written to match, and so are this
   PLAN's steps that depend on it: Phase 1 step 1 (Q4), Phase 2 steps 2 and
   4 (Q3, Q5).
2. `tools/records.py status`: the MADR `accepted`, this PLAN `in-progress`.

### Phase 1: the device code as data (D1)

1. **`wizard/device.go`:**
   * `DeviceCode`, as the MADR's D1 gives it, documented as a value, safe
     for concurrent use while no goroutine changes it (R51);
   * `runDeviceLogin(ctx, id, flow auth.OAuthFlowOptions, show
     func(DeviceCode) error) (*auth.OAuthSession, error)`: call
     `auth.StartDeviceOAuth`, then `show` with the provider,
     `VerificationURI`, `UserCode` and `Expiry`. On an error from `show`,
     `login.Cancel()` and return that error as it is. Else
     `login.Wait(ctx)`; *(2026-10-22 deviation: no `Cancel`. The error is
     returned before `Wait`, so before any poll; see Phase 1's record.)*
   * `var deviceLogin = runDeviceLogin`, the test seam, in place of
     `loginDeviceOAuth`;
   * `showDeviceCode(p, o) func(DeviceCode) error`: `o.ShowDeviceCode` when
     set, else today's notice, "Open %s and enter code %s", returning nil.
2. **`Options.ShowDeviceCode func(DeviceCode) error`,** documented after
   `KiloDeviceKey`: when it is called, what replaces it when nil, that an
   error from it ends the sign-in and is returned as it is, and that
   `Expiry` is never zero and is OpenAI's own wait for OpenAI.
3. **`wizard/auth.go`:** the three device call sites (Grok's and OpenAI's,
   `resolveKiloDevice`, `resolveKiloDeviceKey`) call `deviceLogin(ctx,
   d.ID, oauthFlowOptions(p, o), showDeviceCode(p, o))`.
   `oauthFlowOptions` loses `NotifyDevice`, which nothing then reads, and
   its comment says so.
4. **The three moved tests.** Each stub of `loginDeviceOAuth` becomes a
   stub of `deviceLogin`, calling `show` where it called
   `opts.NotifyDevice`. `auth_test.go:207`'s check that `NotifyDevice` is
   set becomes a check that `show` is. Nothing else in them changes. **The
   drift check runs here.**
5. **`ConfigureLLM`'s and `ResolveCredential`'s docs** name the hook's
   error among the kinds they return, as they are (R52).
6. **The tests,** `wizard/device_0057_test.go`:
   * `TestShowDeviceCode_Data_0057`: Grok's device sign-in with the hook.
     One call, with `ProviderGrok`, the stub's URL, code and expiry; no
     notice holds the code.
   * `TestShowDeviceCode_Kilo_0057`: Kilo's device sign-in, with and
     without `KiloDeviceKey`. Each calls the hook once with `ProviderKilo`.
   * `TestShowDeviceCode_ErrorEndsSignIn_0057`: the hook returns a
     sentinel. The run's error matches it, and the store saved nothing.
   * `TestShowDeviceCode_NilKeepsNotice_0057`: no hook. The notice is
     `Open https://verify.test and enter code ABCD-EFGH`, word for word.
   * `TestRunDeviceLogin_Expiry_0057`: the real `runDeviceLogin`, for Grok,
     with an `http.Client` whose transport answers the device
     authorization with `expires_in: 600` and fails any other request
     except discovery. `show` records the `DeviceCode` and returns a
     sentinel. The expiry is within 5 s of now plus 600 s, the error
     matches the sentinel, and the token endpoint was never called.
7. **Plants** (`tools/plant.py`), each expected to fail a test above:
   * the notice shown as well as the hook;
   * `Expiry` left zero;
   * the hook's error ignored, and the wait run;
   * `Provider` left empty;
   * Kilo's key mode still on the notice;
   * ~~`login.Cancel()` dropped (the token endpoint is then polled).~~
     *(Struck 2026-10-22: there is no `Cancel` to drop. The plant of the
     hook's error ignored covers the poll.)*
8. **Docs:** "The wizard" in `docs/architecture.md` names the hook;
   `CHANGELOG.md`'s `[Unreleased]` Added.

### Phase 2: `NoPastePrompt`, and an empty paste asked again (D2, D4)

1. **`Options.NoPastePrompt bool`,** documented after `NoEndpointPrompt`:
   no paste prompt and no paste instruction; the authorize URL still shown
   and `OpenURL` still called; the loopback alone completes the sign-in;
   and Q3's answer for the ChatGPT plan.
2. **`wizard/auth.go`:**
   * `browserFlowOptionsFor`: with the field set, `OpenURL` shows "Open %s
     in your browser" and calls `o.OpenURL`. `InputCode` is nil, and the
     returned drain does nothing.
   * `offeredAuthMethods`, **under Q3 (b)**: the ChatGPT plan sign-in is
     left out when the field is set, and its comment says so. Under Q3 (a)
     it is unchanged. *(Q3 answered (a), 2026-10-22: unchanged. The plan
     sign-in's `InputCode` comes from `browserFlowOptionsFor`, so the first
     bullet covers it.)*
3. **The drift check runs here,** before step 4.
4. **The empty paste, under Q5 (a):** `pastePrompt.input` asks again while
   the answer is empty after trimming. Before each read it checks `done`
   under its lock, as the first read does, and returns `context.Canceled`
   once `drain` has run. Each read has its own `pending`. An error from
   `Input` is returned as it is. The drift check runs again here:
   `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt`, whose paste
   read is answered with Enter, passes unchanged.
5. **The tests,** `wizard/paste_0057_test.go`:
   * `TestNoPastePrompt_LoopbackOnly_0057`: Grok's browser sign-in with the
     field set. The login's `InputCode` is nil, no `Input` was shown, no
     notice says "paste", the URL notice was shown once, and the
     consumer's `OpenURL` was called once.
   * `TestNoPastePrompt_PlanSignIn_0057`: OpenAI with `AgentName`, a store
     and the field set. Under Q3 (b), the menu lacks the plan sign-in;
     under Q3 (a), it offers it, and the plan login's `InputCode` is nil.
     *(Q3 (a): the second.)*
   * `TestNoPastePrompt_OtherMethodsUnchanged_0057`: the device and API-key
     methods are in the menu with the field set.
   * `TestPaste_EmptyAskedAgain_0057`: a stub login that reads `InputCode`
     once. Answers "", then a code. Two paste prompts, and the login got
     the code.
   * `TestPaste_EndOfInputEnds_0057`: a `TextPrompter` over an empty
     reader. The run ends with an error matching `io.EOF`.
6. **Plants:**
   * `InputCode` left set under the field;
   * the paste instruction shown under the field;
   * the plan sign-in offered (Q3 (b)) or hidden (Q3 (a)), against the
     answer *(Q3 (a): hidden)*, and its `InputCode` left set;
   * the empty answer passed to the login;
   * the re-ask that ignores `done` (expected to fail
     `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt`).
7. **Docs:** `docs/architecture.md`'s entry for `SkipMethods` and
   `NoEndpointPrompt` gains `NoPastePrompt`; `CHANGELOG.md` Added and
   Changed (the empty paste).

### Phase 3: `InputerContext` (D3)

1. **`wizard/prompter.go`:** `InputerContext`, with the MADR's contract in
   its doc: give up when `ctx` ends, return soon after with an error, leave
   no read running; a `Prompter` that cannot promise it must not implement
   it; a wrapper must forward it; the wizard calls it on a goroutine of its
   own while `Notify` may run, for the paste prompt only (R51).
2. **`wizard/auth.go`, `pastePrompt`:**
   * `input` derives `readCtx, cancel` from the login's context, and keeps
     `cancel` under the lock. It calls `InputContext(readCtx, prompt, "")`
     when the `Prompter` implements it, else `Input`, as today.
   * `drain` sets `done` and calls `cancel`. With no read pending it
     returns. For an `InputerContext` it waits for the read, and if the
     read has not returned within `drainGrace` it shows today's notice and
     keeps waiting. For any other `Prompter` it does as today.
   * `var drainGrace = 2 * time.Second`, which a test shortens.
3. **`TextPrompter`'s doc:** it does not implement `InputerContext`, since
   a read of stdin cannot be given up, so the wizard asks for Enter after a
   browser sign-in that the loopback finished (0057-MADR D3).
4. **The drift check runs here.**
5. **The tests,** `wizard/inputctx_0057_test.go`, over a `ctxPrompter`
   whose `InputContext` blocks until its context ends, records the
   context's error, and fails if a second read starts while one runs:
   * `TestInputerContext_LoopbackWinNoEnter_0057`: the stub login starts
     `InputCode` and returns a session without cancelling the context it
     gave, so that the wizard's own cancel is what ends the read. No
     notice says "press Enter", the read's context ended with
     `context.Canceled`, and the next `Select` started after the read
     returned. The wait is bounded at 5 s.
   * `TestInputerContext_PasteWins_0057`: `InputContext` answers a code
     before the loopback. The login got it.
   * `TestInputerContext_IgnoresContext_0057`: `drainGrace` at 10 ms, and
     an `InputContext` that ignores its context until the notice. The
     notice was shown once, and the run finished.
   * `TestInputerContext_PasteOnly_0057`: a run that also asks for a model
     id by `Input`. Only the paste prompt went to `InputContext`.
   * `TestTextPrompter_NotInputerContext_0057`: `*TextPrompter` does not
     implement it.
6. **Plants:**
   * `drain` not cancelling, so the read waits on the login's cancel;
   * the notice shown for an `InputerContext`;
   * `InputContext` used for every `Input`;
   * the grace dropped, so an `InputContext` that ignores its context
     hangs;
   * `Input` called in place of `InputContext`.
7. **Docs:** "The wizard" in `docs/architecture.md` names `InputerContext`,
   and says that `TextPrompter` keeps the Enter; `CHANGELOG.md` Added.

### Phase 4: release `v1.22.0` (D5)

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.22.0] - <date>`, dated
   by the commit the tag will name. It holds Added (the four names) and
   Changed (the empty paste). An empty `[Unreleased]` goes above it, and
   the comparison links are updated. The README's Status names `v1.22.0`.
2. **Before the push:** the gate; `tools/apidiff.py v1.21.0`, listing
   exactly the table's four additions; the notes from `--section v1.22.0
   --repo maccavelli/go-llmprovider-sdk` on a local clone with a local tag,
   scanned; the disclosure guard.
3. **The tag,** on the owner's word: `v1.22.0`, annotated, pushed. Every
   `validate` job passes, and the `release` job's body equals step 2's
   notes.
4. **After the tag:** `gh release delete v1.18.0 --yes`, the Release that
   falls to sixth (R53). Its tag and section stay, and `gh release list`
   then shows `v1.19.0` to `v1.22.0`. `tools/consumer.py v1.22.0` and
   `tools/offline.py v1.22.0`.
5. **The issues,** on the owner's word: one reply to each of 6, 7 and 8,
   naming `v1.22.0`, each request with its test, and the answer to the
   issue's own suggestion where the MADR differs (the struct for issue 6;
   the plan sign-in for issue 7; `TextPrompter` keeping the Enter for issue
   8). Each is written to a scratch file, scanned and shown first. Then the
   three issues are closed.
6. **Close:** the MADR's Confirmation, item by item; this PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | A device sign-in's provider, URL, code and expiry reach the hook, and no notice | 1 | `TestShowDeviceCode_Data_0057`, `TestShowDeviceCode_Kilo_0057` |
| V2 | The expiry is the device authorization's | 1 | `TestRunDeviceLogin_Expiry_0057` |
| V3 | An error from the hook ends the sign-in, before any poll *(was "and cancels the login"; amended 2026-10-22)* | 1 | `TestShowDeviceCode_ErrorEndsSignIn_0057`, `TestRunDeviceLogin_Expiry_0057` |
| V4 | With `NoPastePrompt`, no paste prompt, and the loopback alone | 2 | `TestNoPastePrompt_LoopbackOnly_0057` |
| V5 | The plan sign-in under `NoPastePrompt`, as Q3 decides | 2 | `TestNoPastePrompt_PlanSignIn_0057` |
| V6 | An empty paste is asked again; the drain's Enter is not | 2 | `TestPaste_EmptyAskedAgain_0057`; `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt` |
| V7 | An `InputerContext` read overtaken by the loopback ends with no Enter, before the next prompt | 3 | `TestInputerContext_LoopbackWinNoEnter_0057` |
| V8 | Any other `Prompter`, `TextPrompter` included, keeps the Enter | 3 | the existing drain test; `TestTextPrompter_NotInputerContext_0057` |
| V9 | The existing tests pass, Phase 1's three moved files aside | 1–3 | the drift checks |
| V10 | Only the four names are added | 1–4 | `tools/apidiff.py` |
| V11 | The new names' docs meet R51 and R52 | 1–3 | `make docs-check` |
| V12 | `v1.22.0`'s Release is its section | 4 | `gh release view` |
| V13 | Every new test failed first | 1–3 | the execution record |

V1 to V3 are issue 6's request, V4 to V6 issue 7's, and V7 and V8 issue 8's.

## Rollout and Rollback

* **Rollout.** Five commits on `main`, staged by the agent and committed by
  the owner: the records, one commit per phase, then the release commit.
  Then the owner's tag.
* **Rollback.**
  * Each phase reverts on its own, newest first.
  * Reverting Phase 2 restores the empty paste's end of the sign-in.
  * Reverting any phase removes names. That is allowed only before
    `v1.22.0` is tagged (R48). After it, a fix ships forward.

## Execution record

### Phase 0: records (2026-10-22)

* **Approval.** "Q3 a, Q4 a, Q5 a; commit, then proceed", 2026-10-22. The
  records were first committed as `9d9d59c`.
* **The answers,** each the recommended one, written into the MADR's
  "Owner questions":
  * Q3 (a): the ChatGPT plan sign-in is offered under `NoPastePrompt`. It
    replaces Q1's "hide it", which is kept as given. D2 now says so, with
    the reason, and D5's test bullet with it. Phase 2's steps 2, 5 and 6
    are annotated, not rewritten.
  * Q4 (a): the hook takes a `DeviceCode` and returns an error. D1 says it
    is the owner's answer. No step changes.
  * Q5 (a): an empty paste is asked again. D4 says it is the owner's
    answer. No step changes.
* **Statuses:** the MADR `accepted`, this PLAN `in-progress`, by
  `tools/records.py status`.
* Committed by the owner as `f39372d`.

### Phase 1: the device code as data (2026-10-22)

* **Approval.** "committed", 2026-10-22, after Phase 0 was committed as
  `f39372d`, under the approval to proceed.

**Deviation (2026-10-22): no `Cancel` on the hook's error.**

* **Found** before any code, reading `auth.DeviceLogin`. It polls only in
  `Wait` (`oauth_device.go:164`), and `Cancel` sets a flag and stops a
  running `Wait` (`:179`, `:181`). Step 1's `login.Cancel()`, made before
  any `Wait`, changes nothing a test can observe, so plant 6 could fail
  none.
* **Options put to the owner:**
  * no `Cancel`: the error is returned before `Wait`, so no poll is made
    (recommended);
  * keep `Cancel` as a defence and strike plant 6, which leaves a line no
    test checks.
* **Decision.** "resolution 1". Step 1 is annotated, plant 6 struck and V3
  reworded. The MADR gains an amendment, and its D1 and D5 bullets are
  annotated. Plant 3 (the hook's error ignored and the wait run) is caught
  by `TestRunDeviceLogin_Expiry_0057`'s count of token requests. No exported
  name changes.

**Steps 1 to 3, the code:**

* **`wizard/device.go`:** `DeviceCode`; `runDeviceLogin`, which starts the
  login, gives `show` the code and returns its error before `Wait`;
  `deviceLogin`, the seam; `showDeviceCode`, the hook or today's notice.
* **`Options.ShowDeviceCode`,** after `KiloDeviceKey`, with its doc.
* **`wizard/auth.go`:** `loginDeviceOAuth` is gone, and the three device
  call sites call `deviceLogin`. `oauthFlowOptions` sets no `NotifyDevice`.
  Its unused `Prompter` parameter is dropped too, at its four callers, so
  that lint has no unused parameter to find. Nothing else in the step
  changes.

**Step 4, the three moved tests:** each stub of `loginDeviceOAuth` now stubs
`deviceLogin`, and calls `show` with a `DeviceCode` where it called
`NotifyDevice`, returning its error. `auth_test.go`'s nil check is of
`show`. **The drift check:** `gofmt -l wizard/` empty, `go vet ./wizard/` 0,
`go test -race -count=1 ./wizard/` "ok".

**Step 5:** `ConfigureLLM`'s and `ResolveCredential`'s docs name the hook's
error, returned as it is.

**Step 6, the tests,** `wizard/device_0057_test.go`, each passing under
`-race`:

| Test | Shows |
| :--- | :--- |
| `TestShowDeviceCode_Data_0057` | Grok: the hook given the code once; no notice holds it |
| `TestShowDeviceCode_Kilo_0057` | Kilo, with and without key mode: the hook given Kilo's code once; no notice holds it |
| `TestShowDeviceCode_ErrorEndsSignIn_0057` | the hook's error, as it is; nothing saved |
| `TestShowDeviceCode_NilKeepsNotice_0057` | no hook: `Open https://verify.test and enter code ABCD-EFGH` |
| `TestRunDeviceLogin_Expiry_0057` | the real sign-in for Grok: its provider, URL and code, the expiry within 5 s of now plus 600 s, the hook's error, no request beyond discovery and the device code |

`TestRunDeviceLogin_Expiry_0057` bounds its wait at 5 s, and its device
reply asks a 1 s interval: under plant 3 the wait polls, and fails there
rather than hanging (the PLAN's red-first rule).

**Step 7, red first,** `tools/plant.py` on a scratch clone. Every plant was
caught by a failing test, none by a failed build:

| Plant | Failure line |
| :--- | :--- |
| the notice shown as well as the hook | `notices ["Open https://verify.test and enter code ABCD-EFGH"] show the code; the hook replaces the notice` |
| `Expiry` left zero | "expiry 0001-01-01 00:00:00 +0000 UTC; want within 5 s of …, the device authorization's 600 s" |
| the hook's error ignored, and the wait run | "error oauth: device-code wait: context deadline exceeded; want show's error as it is" |
| `Provider` left empty | `show given {Provider: URL:https://accounts.x.ai/device Code:WXYZ-1234 …}; want Grok's provider, URL and code` |
| Kilo's key mode still on the notice | "key mode true: hook given []; want Kilo's code once" |
| Grok's and OpenAI's device sign-in still on the notice | `hook given [], kind "oauth"; want {grok https://verify.test ABCD-EFGH …} once`; "error <nil>, 1 saves; want the hook's error and nothing saved" |
| Kilo's session sign-in still on the notice | "key mode false: hook given []; want Kilo's code once" |

The PLAN listed six, the sixth struck by the deviation. Two more are added
for call sites it did not name: Grok's and OpenAI's, and Kilo's session
sign-in.

**Step 8, the docs and checks:**

* **Docs:** `docs/architecture.md`'s wizard section gains "A device
  sign-in's code", and `CHANGELOG.md`'s `[Unreleased]` an Added entry.
* **The checks,** each exit status captured before any filter:

```text
$ python3 -B tools/apidiff.py v1.21.0
Compatible changes:
- ./wizard.DeviceCode: added
- ./wizard.Options.ShowDeviceCode: added                              (exit 0)
$ make docs-check changelog-check tables-check parity-check
sdkaudit: 0 violation(s) of R50,R51,R52; changelog-check: 30 release section(s), 0 problem(s);
check-tables: 38 packages, 0 problem(s); G-parity: 409 identifiers, … 0 problem(s)   (exit 0)
$ make pre-add-check FILES="<the 7 Go files>"
go-precheck: 7 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck).   (exit 0)
$ python3 -B tools/gate.py
gate: 27 of 27 passed                                                 (exit 0)
$ python3 -B tools/claims.py docs/decisions/0057-MADR-wizard-device-code-and-paste-prompt.md
claims: … 36 of 36 pass                                               (exit 0)
```

V1 to V3 are met.

### Phase 2: `NoPastePrompt`, and an empty paste asked again (2026-10-22)

* **Approval.** "Commit to main, then proceed". Phase 1 was committed at the
  owner's ask as `29c0238`. No deviation.

**Steps 1 and 2, the field and the flow:**

* **`Options.NoPastePrompt`,** after `NoEndpointPrompt`, with its doc: no
  paste prompt, no paste instruction, no read of the `Prompter`; the URL
  shown and given to `OpenURL`; the loopback alone completes the sign-in,
  the ChatGPT plan's included.
* **`browserFlowOptionsFor`** returns, under the field, a flow whose
  `OpenURL` shows "Open %s in your browser" and calls `o.OpenURL`, with no
  `InputCode` and a drain that does nothing. No `pastePrompt` is made. The
  plan sign-in takes its `InputCode` from this flow, so it has none either.
* **`offeredAuthMethods`** is unchanged (Q3 (a)).

**Step 3, the drift check:** `gofmt -l wizard/` empty, `go vet ./wizard/` 0,
`go test -race -count=1 ./wizard/` "ok".

**Step 4, the empty paste:** `pastePrompt.input` loops over a new `read`
while the trimmed answer is empty. `read` is the old body: it checks `done`
and `ctx` under the lock, makes its own `pending`, and calls `Input`. An
error is returned as it is. **The drift check again:** "ok", with
`TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt` unchanged.

**Step 5, the tests,** `wizard/paste_0057_test.go`, each passing under
`-race`:

| Test | Shows |
| :--- | :--- |
| `TestNoPastePrompt_LoopbackOnly_0057` | Grok's browser sign-in: no `InputCode`, no `Input`, no notice saying "paste"; the URL shown once and opened once |
| `TestNoPastePrompt_PlanSignIn_0057` | OpenAI with `AgentName`: the plan sign-in in the menu, its login given `OpenURL` and no `InputCode`, the session's client |
| `TestNoPastePrompt_OtherMethodsUnchanged_0057` | every built-in provider's methods equal with and without the field |
| `TestPaste_EmptyAskedAgain_0057` | answers "" then a code: the paste prompt twice, and the login got the code |
| `TestPaste_EndOfInputEnds_0057` | a `TextPrompter` over an empty reader: an error matching `io.EOF` |

**Step 6, red first,** `tools/plant.py` on a scratch clone, all six caught
by failing tests, none by a failed build:

| Plant | Failure line |
| :--- | :--- |
| `InputCode` left set under the field | "the login was given an InputCode under NoPastePrompt"; the same for the plan login |
| the paste instruction shown under the field | "notice \"If the browser does not return here, paste the redirected URL or the authorization code\" under NoPastePrompt; want no paste instruction" |
| the plan sign-in hidden under the field, against Q3 (a) | "scripted index 5 out of range for \"Choose how to authenticate with OpenAI:\""; "openai: methods [… ] under NoPastePrompt; want …" |
| the plan login's `InputCode` left set | "the plan login was given an InputCode under NoPastePrompt" |
| the empty answer passed to the login | "InputCode = \"\", <nil>; want the code after the empty answer"; "want an error matching io.EOF" |
| the re-ask that ignores `done` | a panic in `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt`: its `drainPrompter.Input` closed `inputStarted` a second time (`auth_test.go:479`), so a second paste read started after the drain |

The last is caught by the existing test the PLAN named, as a panic, not an
assertion: a second read after the drain is what it was written to refuse.

**Step 7, the docs and checks:**

* **Docs:** `docs/architecture.md`'s wizard section names `NoPastePrompt`,
  and a new entry, "The paste prompt", says that an empty paste is asked
  again and that the Enter finishes a read the loopback overtook.
  `CHANGELOG.md` gains an Added and a Changed entry.
* **The checks,** each exit status captured before any filter:

```text
$ python3 -B tools/apidiff.py v1.21.0
Compatible changes:
- ./wizard.DeviceCode: added
- ./wizard.Options.NoPastePrompt: added
- ./wizard.Options.ShowDeviceCode: added                              (exit 0)
$ make docs-check changelog-check tables-check parity-check
sdkaudit: 0 violation(s) of R50,R51,R52; … 0 problem(s) each          (exit 0)
$ make pre-add-check FILES="wizard/auth.go wizard/configure.go wizard/paste_0057_test.go"
go-precheck: 3 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck).   (exit 0)
$ python3 -B tools/gate.py
gate: 27 of 27 passed                                                 (exit 0)
```

V4 to V6 are met.
