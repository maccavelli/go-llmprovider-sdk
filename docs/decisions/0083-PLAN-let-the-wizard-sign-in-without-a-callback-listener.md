---
status: in-progress
date: 2026-11-17
associated-madr: "0083-MADR-let-the-wizard-sign-in-without-a-callback-listener.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Browser Sign-In With No Callback Listener: the Wizard's `NoLoopback`, Its Question to the Person, and `NoBrowserQuestion`

Associated MADR: [0083-MADR-let-the-wizard-sign-in-without-a-callback-listener.md](0083-MADR-let-the-wizard-sign-in-without-a-callback-listener.md)

## Goal

After this PLAN, in the release it shares with
[0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
and the other drafts the owner's Q8 answer names (MADR D10):

* `wizard.Options.NoLoopback` runs every browser sign-in the wizard runs
  with no callback listener and no question. A paste completes the sign-in,
  under the MADR's D5 words (MADR D1, D2, D7).
* Every other browser sign-in asks the person once per run, before the
  plan's consent and the browser, where the browser is. The question is
  one `Prompter.Select` with "On this machine" as the default. "On another
  device" gives the `NoLoopback` flow, and `OpenURL` is not called (D3, D6).
* The question is never asked under `NoLoopback`, `NoBrowserQuestion`,
  `NoPastePrompt` or a `Driver`. It never loops, and an ended input means
  "On this machine" (D3, D7).
* `NoLoopback` with `NoPastePrompt` is `ErrInvalidRequest` before any
  question (D4).
* The ports-busy error keeps its prefix and gives each method's own advice
  (D8).
* Every caller can read, before bumping, what changes for it and how to
  turn the question off (D9).

The MADR is `proposed`, and so is this PLAN. All the MADR's questions are
answered ("Decided"). Nothing below Phase 0 starts before the owner accepts
the MADR and approves this PLAN, whole or by phase.

## Prerequisites

* 0080-PLAN Phase 3's code is committed on `main`, and no other agent holds
  staged or uncommitted work in the tree. This PLAN's Phase 1 starts on
  that commit, called **the base** below.
* The other drafts that Q8's answer puts in the same release (among them
  `0083-MADR-a-cancelled-sign-in-saves-nothing.md`, whose PLAN also changes
  `wizard/configure.go`) may land before this PLAN's Phase 1. The base is
  then the latest of those commits on `main`, and Phase 1 step 1 re-reads
  what they changed too. The coordinator orders the phases. No two of them
  run their gates on one host at once.
* The drafting's scripts are handed to the Phase 1 engineer by the
  coordinator. They are not committed with the records.
  * `proto.py` inserts the question's prototype into a scratch tree.
  * `overlay.py` writes a repository's staged files over a scratch copy.
  * `runwiz.py` runs the wizard's tests in scratch trees and lists the
    failures.
  * `classify.py` says which of those tests read the recorded `Select`
    calls.
  * `heldports_sdk.py` runs packages with 1455 and 1457 held.

## Relation to 0080

* **Order.** This PLAN runs after 0080-PLAN Phase 3 is committed and before
  the release that 0080-PLAN Phase 5 cuts. Its gates run after 0080's
  Phase 4 gates, never at once on one host.
* **Files both touch.** 0080's staged work changes three files this PLAN
  also changes (read on the index, 2026-11-17):
  * `wizard/auth.go`: `resolvedCredential`'s fields, `resolveKiloDevice`,
    `resolveKiloDeviceKey`'s doc and the "check" wording.
  * `wizard/configure.go`: `credentialStep`'s check and the docs of
    `Options`, `ConfigureLLM` and `ResolveCredential`.
  * `wizard/prompter.go`: a paragraph in the package doc.

  It also adds `wizard/kilo_verify.go`. It touches neither `browserSignIn`,
  nor `browserFlowOptionsFor`, nor the paste instructions, nor `Options.NoPastePrompt`.
* **What Phase 1 re-reads at the base,** recording each line in the
  execution record before any edit:
  * in `wizard/auth.go`:
    * `browserSignIn` and its doc;
    * `browserFlowOptionsFor`, `pastePrompt.show` and the four paste
      constants;
    * `resolveChatGPTPlan`, and the method switch from `methods[idx].ID`
      to the `AuthBrowserOAuth` case;
    * `resolvedCredential`'s fields as 0080 left them;
  * in `wizard/configure.go`:
    * `Options` from `OpenURL` to `NoPastePrompt`, and where the two new
      fields go;
    * the first checks of `ConfigureLLM` and `ResolveCredential`, where
      D4's error goes;
    * `credentialStep`'s refused-kept-session path (the MADR's C68), where
      a run's second sign-in comes from;
  * in `wizard/prompter.go`: the package doc and `Prompter.Select`'s doc;
  * in `llmprovider/auth`: `listenOpenAILoopback` and its caller in
    `browserListener`, and the docs of `LoginBrowserOAuth` and
    `LoginChatGPTPlan`;
  * `README.md` "The configuration wizard", `docs/architecture.md` "The
    wizard", and `CHANGELOG.md` `[Unreleased]` as 0080 and any earlier
    draft of the release filled it.
* **The census again.** Phase 1 step 2 re-runs the drafting's measurement
  at the base, because the base is not `b109cea`.

## Scope

### In scope

* `wizard/configure.go`:
  * the two fields `NoLoopback` and `NoBrowserQuestion`;
  * D4's check before any question;
  * the docs of `Options.OpenURL`, `Options.NoPastePrompt`, `ConfigureLLM`
    and `ResolveCredential`.
* `wizard/auth.go`:
  * the question and its answer, held once per run;
  * `spec.NoLoopback` set in `browserSignIn`;
  * D5's two instructions;
  * D6's `OpenURL` rule.
* `wizard/prompter.go`: `Prompter.Select`'s doc.
* `llmprovider/auth/oauth_loopback.go`: `listenOpenAILoopback` takes the
  sign-in's advice (D8), builds its text from the ports it was given, and
  keeps the prefix. Also the docs of `LoginBrowserOAuth` and, in
  `oauth_plan.go`, `LoginChatGPTPlan`.
* Tests: `wizard/browser_question_0083_test.go` and
  `llmprovider/auth/ports_advice_0083_test.go` (new). The 30 existing
  wizard tests of the table in Phase 1, changed only as that table says.
* *(Amended 2026-11-17.)* `llmprovider/auth`'s unexported listen hook and
  `llmprovider/auth/no_loopback_binds_0083_test.go`, and the repair of
  `TestStart_NoLoopback_0074` in `llmprovider/x/signin/start_browser_0074_test.go`;
  see "Amendment 2026-11-17: the no-listener test that another program's listener can fail".
* Docs:
  * `CHANGELOG.md` `[Unreleased]`: "Added" and "Changed";
  * `README.md` "The configuration wizard";
  * `docs/architecture.md` "The wizard";
  * dated one-line notes in 0074-MADR D4 and D6, 0057-MADR D2 and
    0008-MADR D3, each pointing at 0083-MADR.

### Exported names (R55)

| Name | Kind | Call site that needs it |
| :--- | :--- | :--- |
| `wizard.Options.NoLoopback` | field, `bool` | gobble-cli's `provider.SignIn` (`llm/provider/signin.go:345` at `2004980`), passing `SignInOptions.Remote`, which its two terminal tests set (MADR D10). Also this module's new tests. |
| `wizard.Options.NoBrowserQuestion` | field, `bool` | Named by the owner's N1 answer. Its first call sites are this module's 17 tests in group N of Phase 1's table. A caller that knows its browser is local, or whose scripted prompter refuses an unexpected `Select`, is its expected user (MADR D9). |

Nothing else is added, removed or changed. `Prompter` is unchanged.

### Out of scope

* `llmprovider/x/signin` and `internal/signinflow`. The engine's
  `Spec.NoLoopback` and its checks are used as they are.
* `authcontract`: the question is not a contract step (N3).
* An automatic fallback when the ports are busy (Q7, N7).
* Other repositories: gobble-cli's and prepare-commit-msg's pin bumps and
  tests, and any terminal flag (Phase 6 names them; their own records make
  them).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
* **One commit: the tests with the code.** As 0080's ruling of 2026-11-17
  had it for its own Phase 3, proposed here for the owner's approval with
  the PLAN: the new tests are seen red on a scratch clone of the base and
  land with the code in one commit, so `main` never carries a failing
  build.
* **Red, then green.**
  * Each new test is seen red at the base. A test that names a new field
    cannot compile there, so its red is the base's code with only the test
    file and the two field declarations added, on a scratch clone. That is
    declared in `RED.json` and run with
    `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>`.
  * Every new test is green on the phase's tree with `--expect pass`.
  * Every new assertion is seen to fail on a planted break of its own, with
    `python3 -B tools/plant.py PLANTS.json`, on a scratch clone, never in
    the tree.
  * The execution record quotes each failure line. A break no test catches
    is a deviation.
* **No hang.** No new test can hang the suite.
  * Where the question must not be asked, the `Prompter`'s `Select` fails
    the test and returns an error at once.
  * Where it may be asked, the run is on a goroutine, waited for with a
    bound (5 s), and a timeout fails the test.
  * Every `go test` command in this PLAN carries `-timeout`.
* **Ports.** No new test binds 1455 or 1457, or any port at all: the
  wizard's tests use the login seams (MADR C48, C49), and `auth`'s use
  `nettest.Port` with `openaiLoopbackPorts` swapped (C46, C47). Two
  existing things bind the real ports, and each says so:
  * `TestBrowserListener_OpenAIRedirectsToLoopbackIP`
    (`llmprovider/auth/oauth_redirect_test.go:16`) skips when they are busy;
  * the held-ports check of Phase 2 holds them on purpose, after checking
    them free, to show no wizard test needs them.
* **Heavy runs queue.**
  * Every `-race` run, every multi-round run and the gate run go through
    `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`
    on the session host, or through the skill's `scripts/run-on-host` on
    the host `~/.agents/skills/available-dev-hosts/scripts/pick-host`
    prints.
  * No run goes to a WSL distro. If `pick-host` prints one, the run takes
    the session host through `run-heavy`.
  * A multi-round run carries an explicit `-timeout`.
  * A single-package `go test` without `-race` may run on the session host.
* **Pre-add.** `make pre-add-check FILES="..."` on every changed Go file,
  through `run-heavy` (it lints and tests the packages).
* **No outside host in a test** (0065-MADR): logins are the seams, or an
  issuer the test serves in process.
* **Stop conditions,** each a stop and a prompt to the owner:
  * an existing test needs any change beyond what Phase 1's table gives it
    (one more scripted `0` in `selects`, or `NoBrowserQuestion: true` in
    its `Options`). An assertion that would change is a stop, never an
    edit;
  * the census at the base (Phase 1 step 2) lists a test the table does
    not. The new test is classified by the table's rule and recorded as a
    deviation; the phase goes on only if the rule places it without doubt;
  * an error text changes other than D8's advice, or D8's prefix changes;
  * `make api-check` or `tools/apidiff.py v1.31.0` reports anything but the
    two fields;
  * a new test cannot be made to fail without changing code that is not a
    test.

## Implementation Steps

### Phase 0: records

Model: lightweight, low.

1. The coordinator places this pair after the base's commit, as the
   drafting's `PLACEMENT.md` says. `place.py` writes both records with the
   number. `tools/records.py add` writes the index rows, and the commit is
   made under the bootstrap exception (records only).
2. When the owner says so, `tools/records.py status` sets the MADR
   `accepted` and this PLAN `in-progress`, each with its index row.
3. Run `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --markdown`,
   then stage and commit (`git commit --no-edit`).

Acceptance: `make records-check links-check` pass; `python3 -B
tools/claims.py` passes on both records; no source changed. Docs: the
records are the docs.

### Phase 1: the tests, the code, the 30 tests and the docs, one commit

Model: mainstream, medium. It is one question and one pass-through, but
it changes 30 existing tests, touches three files 0080 changed, and adds
a doc every caller reads.

1. **Re-read the base** ("Relation to 0080") and record the lines.
2. **The census at the base.**
   * On a scratch clone of the base, `proto.py` inserts the prototype
     question.
   * `runwiz.py` runs `go test -count=1 -v -timeout 600s ./wizard/`, first
     without the prototype (all pass) and then with it.
   * `classify.py` classifies each failure.
   * Record the list. It must equal the table in step 5, or a stop
     condition applies. At the drafting, on `b109cea` and on `b109cea`
     with 0080's staged files over it, both runs listed the same 30 tests
     (MADR RUN5, RUN6).
3. **The new tests**, in `wizard/browser_question_0083_test.go` unless the
   row says otherwise. Each uses the login seams and a fake or scripted
   `Prompter`.

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestNoLoopbackReachesTheLogin_0083` | `NoLoopback` reaches the login: `opts.NoLoopback` true for OpenAI's and Grok's `browser_oauth` and `chatgpt_plan`, first and again. No `Select` with the question's title. D5's instruction, URL-only for a first plan sign-in. | the pass-through removed |
   | `TestNoLoopbackOffByDefault_0083` | Without it, and answering "On this machine", the login gets `NoLoopback` false and today's instruction (C42). | the field's value inverted |
   | `TestQuestionAnswerElsewhere_0083` | "On another device": `NoLoopback` true at the login, D5's words, `OpenURL` not called (a recording `OpenURL`). | the answer ignored; `OpenURL` called after it |
   | `TestQuestionNoLoopbackStillOpens_0083` | Under the caller's `NoLoopback`, `OpenURL` is called once (D6). | `OpenURL` skipped under `NoLoopback` |
   | `TestQuestionForm_0083` | The question's title, its two choices word for word, its default index 0, and that it is one `Select` call. | the default set to 1; a choice's label changed |
   | `TestQuestionBeforeConsent_0083` | For a first plan sign-in, the order of the `Select` titles: method menu, the question, the consent. For Grok's browser method: the question right after the method menu. | the question asked after the consent |
   | `TestQuestionOncePerRun_0083` | A saved session refused by the listing leads to a second sign-in in the run (C68); the question is asked once. | the answer not kept for the run |
   | `TestQuestionNotAskedWhereItMustNot_0083` | Under each of `NoLoopback`, `NoBrowserQuestion`, `NoPastePrompt` and a `Driver`, a `Prompter` whose `Select` fails the test, and returns an error, on the question's title; the sign-in completes. | the question asked under one of the four, a plant each |
   | `TestQuestionNotAskedForOtherMethods_0083` | Grok's and OpenAI's device sign-ins, an API key and a vendor import ask no question. | the question asked for a device sign-in |
   | `TestQuestionEndedInputIsThisMachine_0083` | A `TextPrompter` whose input ends after the method's answer: the question takes the default, the login gets `NoLoopback` false; run on a goroutine and waited for within 5 s. | the default set to "On another device" |
   | `TestQuestionExhaustedInputEnds_0083` | A `TextPrompter` whose input is already exhausted at the question: one error that ends the run, the question's text written once to its output; waited for within 5 s, so a loop fails instead of hanging. | the question re-asked after an error |
   | `TestNoLoopbackWithNoPastePromptRefused_0083` | `NoLoopback` with `NoPastePrompt`: `ErrInvalidRequest` from `ConfigureLLM` and from `ResolveCredential`, with no `Prompter` call. | the check removed |
   | `TestNoBrowserQuestionKeepsToday_0083` | `NoBrowserQuestion` alone: no question, `NoLoopback` false at the login, today's instruction and `OpenURL` call. | `NoBrowserQuestion` ignored |
   | `TestPortsAdvicePerMethod_0083` (`llmprovider/auth/ports_advice_0083_test.go`) | Both registered ports held: two `nettest` ports bound on 127.0.0.1 and ::1 after swapping `openaiLoopbackPorts`, as `holdPorts0074` does (C46). `LoginChatGPTPlan` (its in-package fake issuer) gives an error that starts with `oauth: registered callback ports unavailable;`, says to free the two ports it was given, and does not say "device-code". `LoginBrowserOAuth` for OpenAI starts with the same prefix and says "device-code". | the old text restored for the plan (the "device-code" assertion fails); the prefix changed (both fail) |

   The existing `TestListenOpenAILoopback_ErrorsWhenAllBusy`
   (`oauth_loopback_test.go:60`) keeps its assertion that the browser's
   error names the device-code login. Its call gains the advice argument,
   a change of call syntax only (R46).
4. **Red.** `RED.json` lists every new test, with the base's code plus the
   two field declarations as the fixture, so the wizard tests compile.
   Then `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>`.
   Quote each failure. `TestPortsAdvicePerMethod_0083` is red at the base
   as it is: the plan's text still says "device-code".
5. **The 30 existing tests.**
   * **The rule, decided here.** A test whose subject is the browser or
     plan sign-in's own prompts, and whose body does not read the recorded
     `Select` calls, **scripts the answer**: one more `0` in its `selects`,
     before the sign-in's prompts.
   * Every other test **sets `NoBrowserQuestion: true`** in its `Options`.
     That covers a test whose body indexes or counts the recorded `Select`
     calls, where scripting would shift an asserted index. It also covers
     a test whose subject is what comes after the login: the save, the
     listing's refusal, a kept session, the store.
   * No assertion is changed, loosened or removed in either group. The
     question's own behaviour is held by step 3's tests.

   | File | Tests | Count | Change | Why |
   | :--- | :--- | :--- | :--- | :--- |
   | `wizard/auth_test.go` | `TestConfigureLLM_BrowserAndDevicePersistSessions`, `TestConfigureLLM_BrowserOAuthSetsInputCode`, `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt` | 3 | script `0` | the browser flow is their subject; they still hold the paste race and drain after "On this machine" |
   | `wizard/inputctx_0057_test.go` | `TestInputerContext_LoopbackWinNoEnter_0057`, `TestInputerContext_PasteWins_0057`, `TestInputerContext_IgnoresContext_0057`, `TestInputerContext_PasteOnly_0057` | 4 | script `0` | the paste read is their subject |
   | `wizard/paste_0057_test.go` | `TestPaste_EmptyAskedAgain_0057` | 1 | script `0` | the paste prompt is its subject |
   | `wizard/signin_errors_0049_test.go` | `TestConfigureLLM_BrowserLoginErrorBare_0049`, `TestConfigureLLM_PlanLoginErrorBare_0049` | 2 | script `0` | the login's error is their subject |
   | `wizard/plan_0043_test.go` | `TestPlan_NoticeDeclined_0043`, `TestPlan_ReSignIn_0043`, `TestPlan_ReplacesCodexSession_0043` | 3 | script `0` | the plan's prompts are their subject; their bodies do not read the recorded `Select`s |
   | `wizard/plan_0043_test.go` | `TestPlan_FirstSignIn_0043` | 1 | `NoBrowserQuestion` | its body reads the recorded `Select`s (`plan_0043_test.go:121`); the order with the question is `TestQuestionBeforeConsent_0083`'s |
   | `wizard/consent_0075_test.go` | `TestWizardPlanConsentSelect_0075`, `TestWizardReSignInWithoutAnswerAsks_0075` | 2 | `NoBrowserQuestion` | they index the consent's `Select` (`consent_0075_test.go:44`) |
   | `wizard/resolve_0056_test.go` | `TestResolveCredential_ChatGPTPlan_0056` | 1 | `NoBrowserQuestion` | its body reads the recorded `Select`s |
   | `wizard/kept_0059_test.go` | `TestKept_UnusableNotOffered_0059` | 1 | `NoBrowserQuestion` | a kept session is its subject, and its body reads the recorded `Select`s |
   | `wizard/credential_0066_test.go` | `TestKept_RefusedLeadsToSignIn_0066`, `TestKept_RefusedThenNewRefused_0066` | 2 | `NoBrowserQuestion` | a refused kept session is their subject |
   | `wizard/session_0059_test.go` | `TestSession_RefusedKeepsSaved_0059`, `TestSession_AcceptedSavedOnce_0059`, `TestSession_NoDiscoverSaved_0059`, `TestSession_RotatedDuringCheckSaved_0059`, `TestSession_RefreshAfterLimitSaved_0059`, `TestSession_RefreshRefusedNotSaved_0059` | 6 | `NoBrowserQuestion` | the save is their subject |
   | `wizard/session_store_race_0074_test.go` | `TestSession_SaveBindsStoreWhileUsed_0074` | 1 | `NoBrowserQuestion` | the store's race is its subject |
   | `wizard/menu_0066_test.go` | `TestSave_StoreFails_0066` | 1 | `NoBrowserQuestion` | a failing store is its subject |
   | `wizard/stored_session_test.go` | `TestConfigureLLM_NoStoredSessionSignsIn`, `TestConfigureLLM_StoredSessionLeavesNoTokenInResult` | 2 | `NoBrowserQuestion` | the store and the Result are their subject |
   | **13 files** | | **30** | 13 script `0`, 17 `NoBrowserQuestion` | |

   *(The MADR's N1 was put with "44 places", a `grep` count of the login
   stubs. The tests that meet the question are these 30: MADR RUN5,
   RUN6.)* Where a group N test builds its `Options` in a helper that other
   tests share, the field is set at the test, not in the helper, unless
   every user of the helper is in group N. The execution record says
   which.
6. **The code.**
   * `wizard/configure.go`: the two fields after `NoPastePrompt`, with
     MADR D1's and D3's doc comments.
   * D4's check, in `ConfigureLLM` and `ResolveCredential`, before any
     question: `NoLoopback && NoPastePrompt` is
     `fmt.Errorf("wizard: %w: NoLoopback with NoPastePrompt leaves nothing to complete a browser sign-in", llmprovider.ErrInvalidRequest)`.
   * `wizard/auth.go`:
     * a run-scoped answer, held by the run (not in the caller's
       `Options`, which is passed by value), asked at most once;
     * the question asked in the browser and plan branches after the
       method is chosen and before `browserSignIn` (so before the plan's
       consent), only when none of `NoLoopback`, `NoBrowserQuestion`,
       `NoPastePrompt` or `Driver` is set;
     * the answer "On another device", or `NoLoopback`, sets
       `spec.NoLoopback` and D5's instruction. That answer alone also
       leaves `OpenURL` uncalled;
     * a `Prompter` error from the question ends the run, as every
       question's does.
   * `llmprovider/auth`: `listenOpenAILoopback(ports []int, advice string)`.
     The advice is built by `browserListener` from the sign-in it serves:
     the plan's "free ports P and Q and try again", the browser's "free
     ports P and Q, or use device-code login", with P and Q the ports
     given. The text is
     `oauth: registered callback ports unavailable; <advice>: <the binds' errors>`.
7. **Docs, in the same commit**, written for a caller who reads them
   before bumping:
   * **Doc comments.**
     * `Options.NoLoopback` and `Options.NoBrowserQuestion`, as MADR D1
       and D3 give them.
     * `Options.OpenURL`: not called after the person answers "On another
       device"; called under `NoLoopback`.
     * `Options.NoPastePrompt`: no question is asked, and `NoLoopback` with
       it is an error.
     * `ConfigureLLM` and `ResolveCredential`: the question before a
       browser sign-in, and D4's error.
     * `Prompter.Select`: the wizard also asks with it, before a browser
       sign-in, where the browser is. `defaultIdx` is "On this machine",
       the answer to give when nobody can choose. A `Prompter` that cannot
       ask returns the default. The interface itself is unchanged.
     * `LoginBrowserOAuth` and `LoginChatGPTPlan`: the advice, and
       `NoLoopback` as the way past busy ports.
   * **`README.md`, "The configuration wizard".**
     * A bullet: a browser sign-in first asks where the browser is, once
       per run. "On another device" finishes by a paste with no listener;
       `Options.NoLoopback` decides that without asking;
       `Options.NoBrowserQuestion` asks nothing and keeps the listener.
     * The existing "Scripts that drive the wizard answer one extra,
       blank, line before each model…" (`README.md:471`) gains: "and
       before a browser sign-in, where a blank line is 'On this machine'".
   * **`docs/architecture.md`, "The wizard".** Beside the `NoPastePrompt`
     bullet (`docs/architecture.md:1073`), a bullet with MADR D7's table in
     prose.
   * **`CHANGELOG.md` `[Unreleased]`.**
     * **Added:** `wizard.Options.NoLoopback` (no listener, no question, a
       paste completes the sign-in) and `wizard.Options.NoBrowserQuestion`
       (no question, the listener as before) (0083-MADR).
     * **Changed:** "Every browser sign-in the wizard runs, OpenAI's and
       Grok's and the ChatGPT plan's, first asks once per run where the
       browser is, before the plan's consent, unless the caller sets
       `Options.NoLoopback`, `Options.NoBrowserQuestion`,
       `Options.NoPastePrompt` or `Options.Driver`. A caller whose scripted
       `Prompter` refuses an unexpected `Select` scripts the answer
       (index 0, 'On this machine') or sets `NoBrowserQuestion`. 'On
       another device' runs the sign-in with no callback listener, to be
       completed by a paste, and does not call `Options.OpenURL`.
       `NoLoopback` with `NoPastePrompt` is an error matching
       `ErrInvalidRequest` before any question. When ports 1455 and 1457
       are both taken, the error keeps its prefix and advises freeing the
       two ports, and for the browser sign-in the device-code login, but no
       longer a device-code login for the ChatGPT plan, which has none
       (0083-MADR)."
   * **The dated notes** in 0074-MADR D4 and D6, 0057-MADR D2 and
     0008-MADR D3.
8. **Before staging,** on a scratch clone of the phase's tree:
   * `go test -count=1 -timeout 600s ./wizard/ ./llmprovider/x/signin/ ./internal/signinflow/`
     and `go test -count=1 -timeout 600s ./llmprovider/auth/` (session
     host, no `-race`);
   * every test passes, the 30 changed only as step 5's table says.

Verification:

* the tests green;
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- make pre-add-check FILES="<every changed Go file>"`;
* `make api-check`: exactly the two fields added, `Prompter` unchanged;
* `python3 -B tools/apidiff.py v1.31.0`: the same two additions, nothing
  else;
* `make docs-check tables-check changelog-check links-check records-check markdownlint`.

Acceptance:

* every new test as its row says;
* the 30 changed exactly as the table says, with no assertion touched;
* the API change exactly the two fields;
* the docs say what a caller sees after a bump, and how to turn it off.

*(Amended 2026-11-17.)* Step 3 also writes `TestNoLoopback_BindsNothing_0083`
and repairs `TestStart_NoLoopback_0074`, as "Amendment 2026-11-17: the no-listener test that another program's listener can fail" says.

### Phase 2: planted breaks and the gates

Model: mainstream, low.

1. Run `python3 -B tools/plant.py PLANTS.json` on a scratch clone: every
   plant of Phase 1's table, each caught, each failure quoted.
2. Run `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>`,
   then `--expect pass` on the tree, quoted.
3. The held-ports check, again: `heldports_sdk.py` after checking 1455 and
   1457 free. It holds both on 127.0.0.1 and ::1 and runs the `wizard`,
   `internal/signinflow` and `x/signin` tests; every one passes, none
   skipped (the MADR's RUN1). This step binds the real ports on purpose,
   and says so in the execution record.
4. Run
   `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 -timeout 20m ./wizard/... ./internal/signinflow/... ./llmprovider/x/signin/... ./llmprovider/auth/...`.
5. Run
   `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=20 -timeout 30m -run '0083' ./wizard/ ./llmprovider/auth/`.
   This is 20 rounds of the new tests, whose question races the paste
   reader's goroutine and `Notify`.
6. Run `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`.
7. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`.
8. CI's set: `make parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest`, and `make tools-test`.
9. `python3 -B tools/claims.py` on the MADR and this PLAN;
   `python3 -B tools/denyscan.py --changed`;
   `python3 -B tools/recordlint.py --changed`.

Acceptance: every plant caught; every command exit 0, each status taken
before any filter; no run on a WSL distro. Docs: none beyond Phase 1's; a
gate that finds a docs gap is a deviation.

*(Amended 2026-11-17.)* Phase 2 also runs the 50-round `-race` run of the two
no-listener tests with 1024 listeners held, as "Amendment 2026-11-17: the no-listener test that another program's listener can fail" says.

### Phase 3: independent review

Model: strongest, high: a reviewer that has not seen the drafting or the
build, briefed with the MADR, this PLAN and the diff, not with anyone's
conclusion. The change adds a prompt to every browser sign-in of every
caller.

1. Read the diff against the base. Check each of the MADR's D1 to D10
   against the code, and each D7 row against a test.
2. Re-run Phase 2's plants and the held-ports check on a scratch clone;
   add a plant of the reviewer's own for any assertion it doubts.
3. Read the 30 test changes. Each must be one scripted `0` or
   `NoBrowserQuestion: true`, in the group the table gives, with no
   assertion changed.
4. Read what a caller reads before bumping: the CHANGELOG entry, the README
   bullets and the doc comments. Check it against what gobble-cli's
   terminal path (its `wizardPrompter`, MADR G10) and prepare-commit-msg's
   `TextPrompter` (MADR P2) would do after a bump with no code change.
5. Look for a path where the question could hang or repeat: a `Driver`, a
   `Prompter` error, an ended input, a second sign-in in the run, a
   cancelled context during the question.
6. Findings go to the coordinator. Each blocker is fixed in Phase 1's files
   by a further commit, with its plant, before Phase 4.

Acceptance: the reviewer's report in the execution record, with no blocker
open.

### Phase 4: the owner's check (optional)

Model: lightweight, low, to prepare the run sheet; the run is the owner's.

1. A run sheet for a live wizard plan sign-in in a terminal, through
   `ExampleResolveCredential`'s shape or a scratch program. The person
   answers "On another device", signs in on a phone, and pastes the
   address back. The `x/signin` path beneath it was verified live (MADR R5
   to R9).
2. If the owner runs it: the result in the execution record, with no
   credential, address or code quoted. If not: "not run, by the owner's
   choice" in "Not verified".

### Phase 5: the release (the owner's word)

Model: mainstream, medium. No step runs under a standing approval: the tag
and the GitHub Release each wait for the owner's word in the same turn.

1. By the owner's Q8 answer, this record ships in the next release with
   0080 and the other drafts that answer names. 0080-PLAN Phase 5 cuts it,
   and this PLAN adds its changelog entries, already under `[Unreleased]`.
   If this PLAN has not finished Phase 3 with its gates passed when the
   owner gives the word, it moves to the release after, and its entries
   move with it.
2. Run `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`,
   quoted.
3. `tools/records.py status` sets this PLAN `complete` once the tag's CI
   and step 2 are recorded.

Acceptance: the tag's CI passes; the release notes carry the "Changed"
entry word for word.

### Phase 6: other repositories, after the release (their own records)

Not executed under this PLAN; recorded so the hand-off is exact.

* **gobble-cli** (at `2004980`, branch `p3-acpext-v0.5.0`):
  * bump the pin from `v1.31.0` (MADR G8);
  * `provider.SignIn` passes `SignInOptions.Remote` to
    `wizard.Options.NoLoopback` (G1, G3);
  * `terminalPlan` sets `o.Remote = true` beside the HTTP client and cache
    (G15), so the two terminal tests bind no port and see no question;
  * an assertion at its `resolveCredential` seam that `NoLoopback` arrives,
    with a planted break;
  * `signInFailed`'s port branch and the developers' guide's note go
    (G18, G24);
  * any of its tests that drive the real wizard through a browser sign-in
    with a scripted prompter script the answer or set `NoBrowserQuestion`;
  * whether its user gets a terminal flag is gobble's decision.
* **prepare-commit-msg:** bump the pin from `v1.29.0` (MADR P4). Its
  `TextPrompter` (P2) asks the question. Its tests do not run a browser
  sign-in *(read 2026-11-17: `internal/ui/setup_test.go` names
  `AuthChatGPTPlan` only to list methods)*; whether it sets a field is its
  own record's choice.

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy` or `run-on-host`, never on a WSL distro.

* `go test -count=1 -timeout 600s ./wizard/ ./llmprovider/x/signin/ ./internal/signinflow/ ./llmprovider/auth/`
* `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>`, then
  `--expect pass`; `python3 -B tools/plant.py PLANTS.json`
* the held-ports check (Phase 2 step 3)
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 -timeout 20m ./wizard/... ./internal/signinflow/... ./llmprovider/x/signin/... ./llmprovider/auth/...`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=20 -timeout 30m -run '0083' ./wizard/ ./llmprovider/auth/`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`: the two
  fields only
* `make lint`, `make vuln`, `go vet -tags live_gateways ./...`
* `make parity-check dep-check coverage-check api-check contract-check
  generate-check records-check tables-check fixtures-check changelog-check
  docs-check markdownlint links-check disclosure-check gate-selftest` and
  `make tools-test`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN
* *(Amended 2026-11-17.)* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=50 -timeout 20m -run 'TestStart_NoLoopback_0074$|TestNoLoopback_BindsNothing_0083$' ./llmprovider/x/signin/ ./llmprovider/auth/`,
  with 1024 listeners held in 49152 to 65535

## Rollout and Rollback

* **Order.**
  * Phase 0 when placed.
  * Phases 1 to 3 on approval, on the base.
  * Phase 4 if the owner chooses.
  * Phase 5 with 0080's release, on the owner's word.
  * Phase 6 in the other repositories, after the tag.
* **Rollout.**
  * A caller that sets neither new field: its person is asked one question
    before a browser sign-in. An empty or ended answer keeps today's flow.
  * A caller with a scripted `Prompter` that refuses unexpected questions:
    its tests that run a browser sign-in fail until they script the answer
    or set `NoBrowserQuestion`, as the CHANGELOG says.
  * A caller that sets `NoLoopback`: no listener and no question.
  * A caller with `NoPastePrompt` or a `Driver`: no change, but
    `NoLoopback` with `NoPastePrompt` is now an error.
  * Everyone: the plan's busy-ports error no longer advises a device-code
    login.
* **Rollback before the tag:** revert Phase 1's commit (and any Phase 3
  fix); the records stay, with a dated note.
* **Rollback after the tag:** removing a v1 field is a breaking change, so
  a rollback is a patch release that keeps both fields. It makes the
  question off by default, with a dated amendment of the MADR. Nothing
  persistent changes shape, so no stored session needs migrating.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, at `b109cea`;
Phase 1 re-reads each at the base.

<!-- claims: self=b109cea -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | `Prompter.Select`'s doc | `wizard/prompter.go:72` | `// Select asks the user to choose one option, returning its index.` |
| P2 | `NoPastePrompt`'s doc | `wizard/configure.go:200` | `// NoPastePrompt starts no paste prompt in a browser sign-in, for a caller` |
| P3 | `browserSignIn`'s doc | `wizard/auth.go:812` | `// InputCode and NoLoopback itself (0074-MADR D6).` |
| P4 | where the spec's `NoPaste` is set | `wizard/auth.go:824` | `spec.NoPaste = o.NoPastePrompt` |
| P5 | the URL-only instruction | `wizard/auth.go:765` | `pasteURLInstruction  = "If the browser does not return here, paste the redirected URL"` |
| P6 | the call to the caller's `OpenURL` | `wizard/auth.go:801` | `return o.OpenURL(authorizeURL)` |
| P7 | the method switch | `wizard/auth.go:218` | `method := methods[idx].ID` |
| P8 | the ports function | `llmprovider/auth/oauth_loopback.go:490` | `func listenOpenAILoopback(ports []int) ([]net.Listener, int, error) {` |
| P9 | its error | `llmprovider/auth/oauth_loopback.go:499` | `registered callback ports unavailable; use device-code login` |
| P10 | the existing test of that error | `llmprovider/auth/oauth_loopback_test.go:60` | `if err == nil \|\| !strings.Contains(err.Error(), "device-code") {` |
| P11 | `LoginBrowserOAuth`'s doc | `llmprovider/auth/oauth_loopback.go:123` | `// LoginBrowserOAuth completes a browser authorization-code flow with PKCE. A` |
| P12 | `LoginChatGPTPlan`'s doc | `llmprovider/auth/oauth_plan.go:90` | `// LoginChatGPTPlan signs a user in with their ChatGPT plan, under the app's` |
| P13 | the test fake | `wizard/fake_prompter_test.go:39` | `type fakePrompter struct {` |
| P14 | and its refusal of an unscripted `Select` | `wizard/fake_prompter_test.go:106` | `return 0, fmt.Errorf("fakePrompter: unexpected Select(%q)", title)` |
| P15 | a test that indexes the recorded `Select`s | `wizard/plan_0043_test.go:121` | `if len(f.seenSelect) != 3 \|\| !strings.Contains(f.seenSelect[2], "You're using your ChatGPT plan") \|\|` |
| P16 | another | `wizard/consent_0075_test.go:44` | `title := f.seenSelect[2]` |
| P17 | `auth`'s held substitute ports | `llmprovider/auth/no_loopback_0074_test.go:27` | `func holdPorts0074(t *testing.T) []int {` |
| P18 | the one test that binds the real ports, and skips when busy | `llmprovider/auth/oauth_redirect_test.go:16` | `t.Skipf("OpenAI loopback ports busy: %v", err)` |
| P19 | `TextPrompter.Select`'s default on an empty line | `wizard/text_prompter.go:244` | `return defaultIdx, p.flushErr()` |
| P20 | The README's wizard section | `README.md:381` | `## The configuration wizard` |
| P21 | and its note to scripts | `README.md:471` | `drive the wizard answer one extra, blank, line before each model and` |
| P22 | The architecture's wizard section | `docs/architecture.md:1010` | `## The wizard` |
| P23 | and its `NoPastePrompt` bullet | `docs/architecture.md:1073` | `` **`Options.NoPastePrompt`**, for a `` |
| P24 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |

## Execution record

### Drafting (2026-11-17)

* **Provenance.** Drafted by a strongest-tier model at high effort, on the
  coordinator's briefs, from read-only investigation. Nothing was
  implemented or written to any repository's tree or index. The MADR and
  this PLAN are `proposed`, with `0083` for the number, which the placing
  step takes from `scripts/check_records.py --next`.
* **The rulings,** the owner's of 2026-11-17, relayed by the coordinator:
  Q1 to Q7, then N1 to N7 and Q8, each recorded by its label in the MADR's
  "Decided".
* **Verified, and how.**
  * Read at `b109cea` (`git archive` into a scratch folder): the wizard's
    browser sign-ins, the `Prompter`, `TextPrompter`, the test fake, the
    engine, `x/signin` and `auth`'s loopback.
  * Read on the index of the tree as it stood (`git show :<path>`): 0080's
    staged changes to `wizard/auth.go`, `wizard/configure.go` and
    `wizard/prompter.go`.
  * gobble-cli read at `2004980`; prepare-commit-msg at `2715f2d`.
  * Ran, on scratch copies only:
    * the held-ports check (MADR RUN1, RUN2);
    * the census of the tests that meet the question: `proto.py` on
      `b109cea` and on `b109cea` with 0080's staged files over it. The
      second copy passed whole without the prototype (305 top-level PASS).
      With it, both listed the same 30 tests in 13 files, each failing on
      the fake's scripted `Select`, none hanging (MADR RUN5, RUN6;
      Phase 1's table);
    * `classify.py` on those 30, for the rule in Phase 1 step 5.
  * The citations of both records were checked with `tools/claims.py` from
    a scratch copy, and planted false rows were each caught (the drafting's
    `PLACEMENT.md`).
* **Not verified.**
  * The decided design itself. The prototype asked per sign-in, without
    the fields, and only counted the tests it meets.
  * Any `-race` run, any gate, any host but the session's.
  * A live sign-in through the wizard.
  * How many of gobble-cli's own tests meet the question.
  * The base, which is not yet committed. Phase 1 step 2 re-runs the
    census there.

### Approval (2026-11-17)

* **Approved 2026-11-17.** This PLAN was approved through its live step or
  release by the owner's ruling of 2026-11-17, relayed by the coordinator:
  "Accept all five and run each plan up to its live step or release
  (Recommended)". Its phases are committed and pushed by the coordinator as
  each passes. Tags and releases need the owner's word.

## Amendment 2026-11-17: the no-listener test that another program's listener can fail

*(Added 2026-11-17. Nothing above is changed by it; the dated lines above
point here.)*

### The ruling

The coordinator's account: an independent review's race run of 0080's
work hit one failure, in a test that 0080 does not touch. The coordinator
told the owner, and suggested fixing it under this PLAN, since it sits in
the code this record changes. The owner's words: "yes, fold that into the
plan".

### What the test proves today, and why it can fail

* **What failed.** `TestStart_NoLoopback_0074/grok` (P25) failed once with
  "something listens at 127.0.0.1:53531; want no listener under
  NoLoopback". 0080-PLAN records it, in its execution record under "Found,
  not changed by 0080", the third item.
  * The test file is byte-identical at `b109cea` and `a3002aa`
    (`git diff` empty), and so is every other code file this amendment
    cites.
  * A rerun of the package passed ten rounds under `-race`.
* **What it checks, by subtest:**
  * **`openai`:** the sign-in completes by a paste with the real `auth`
    code, against the fake issuer, which is a `RoundTripper`, not a
    listener (P31). The redirect's port is checked by shape only, since
    OpenAI's redirect names its first registered port, which another
    program may hold (P27). Nothing in this test shows that the OpenAI
    sign-in bound no listener.
  * **`grok`:** the same, and the helper then dials the redirect's port
    (P26, P28) and fails if anything answers (P29). That port is drawn at
    random from 49152 to 65535 and never bound under `NoLoopback` (P32,
    P38).
  * **`NoPaste`:** a refusal before any request. No port is involved.
* **Why the dial can fail with no fault in the code.** The dial reaches
  any listener on that port: another package's test server in the same
  `go test`, or any program on the host.
  * On macOS the ephemeral range is 49152 to 65535 (measured on the
    session host, `sysctl net.inet.ip.portrange`, RUNF6). Every `:0` bind
    there, such as each `httptest` server this module's tests start, lands
    in the range the test draws from.
  * Linux's default ephemeral range overlaps it from 49152 to 60999
    (0045-MADR D2, P43).
  * The test therefore relies on a port it does not own staying unused,
    which is what
    [0045-MADR-tests-never-reuse-a-freed-port.md](0045-MADR-tests-never-reuse-a-freed-port.md)
    records as failing (P42). That record's two remedies cover a freed
    port and a port the code binds. Neither covers a dial meant to find
    nothing, so the test is outside 0045's letter but inside its rule.
* **How rare.** With L listeners on loopback ports in the range at the
  instant of the dial, one run of the `grok` subtest fails with
  probability L / 16384.
  * On the session host while idle, L was 5 (RUNF6), about 1 in 3300.
  * During a parallel `go test ./...` of this module, L is that baseline
    plus every test server open at that instant (the module's tests call
    `httptest.NewServer` at 361 sites, counted by `grep`).
  * L during the run that failed was not measured.
* **What the dial is the only check of.** `auth`'s own
  `TestNoLoopback_Grok_0074` (P39) checks the redirect's shape (P40), not
  whether a listener was bound. With a plant that makes Grok's sign-in
  bind a listener under `NoLoopback`, that test passed and the dial failed
  (RUNF1). So today Grok's "binds nothing" rests on the dial alone.

### Options

* **A. Leave it.**
  * Proves: Grok's redirect port has no listener at the instant of the
    dial.
  * It can be falsified by any listener on that port, ours or not
    (RUNF2), and it fails about L / 16384 of runs.
  * OpenAI's and the plan's "binds nothing" rest on `auth`'s held
    substitute ports (P41), which catch a bind for those two.
* **B. Hold the port, or reserve it.** The test cannot know Grok's random
  port before the sign-in draws it. Under the failure the dial is meant to
  catch, Grok binds `127.0.0.1:0` (P33), a different port again, so
  holding the redirect's port proves nothing about that bind. Pinning the
  port by a seam leaves the dial still answerable by a stranger.
* **C. Recommended: an unexported, per-call listen hook in `auth`, and the
  "binds nothing" assertion where every bind passes through it.**
  * `OAuthFlowOptions` and `ChatGPTPlanOptions` each gain an unexported
    field, `listen func(network, address string) (net.Listener, error)`,
    beside the unexported `now` and `sleep` they already carry for tests
    (P36, P37). It is copied to `oauthFlowConfig`; nil is `net.Listen`.
  * `browserEndpoint` (P38) and the functions under it bind through it at
    the three binds there are: P33, P34, P35. Unexported `…With` variants
    keep the existing helpers' signatures for the tests that call them.
  * A new test in `auth`, `TestNoLoopback_BindsNothing_0083`, gives each of
    OpenAI's and Grok's browser sign-ins and the plan's a hook that records
    every bind and binds nothing.
    * Under `NoLoopback` each sign-in completes by the paste with **no
      bind attempted**.
    * Without it, each sign-in's bind reaches the hook, which shows the
      hook is on the path of every bind.
  * The x/signin test keeps its paste completion and the redirect's shape,
    and drops the dial. The helper is renamed to say it checks a shape, and
    its doc points at the new test.
  * That `x/signin` hands `NoLoopback` to `auth` is
    `TestStart_NoLoopbackPassedOn_0074`'s (P30). It is already there and
    is unchanged.
  * Proves: this process's sign-in attempts no bind at all, for all three
    methods, not one port for one method. Nothing outside the process can
    pass or fail it.
* **D. A hook carried on the context, from a package under the root
  `internal/`, so x/signin's end-to-end test sees the binds itself.**
  * Proves the same, end to end.
  * Cost: a route into two exported functions that their signatures do not
    show. 0074-MADR D4's option (a) was set aside for that reason.
* **E. A package-level hook in `auth`, swapped by tests.** Simpler than C,
  but a test that swaps it races any parallel test that binds, under
  `-race`. Not recommended.
* **F. Replace the dial by binding the port ourselves.** It is as
  answerable by a stranger as the dial is (any holder makes the bind
  fail). No better than A.

### Shown on scratch copies (ran 2026-11-17; never the tree)

The runs were made on a clone of `a3002aa`, with C applied by a script and
each plant on its own copy. Single packages, no `-race`, on the session
host.

| Id | What | Result |
| :--- | :--- | :--- |
| RUNF1 | `a3002aa` unchanged but for a plant: Grok's sign-in binds under `NoLoopback` | the old x/signin test fails, `start_browser_0074_test.go:277: something listens at 127.0.0.1:54433; want no listener under NoLoopback`; `auth`'s `TestNoLoopback_Grok_0074` passes |
| RUNF2 | `a3002aa` with Grok's redirect port pinned to 61234 (scratch only), 61234 held by another listener, checked free first | the old test fails, `start_browser_0074_test.go:277: something listens at 127.0.0.1:61234; want no listener under NoLoopback`, with nothing in the sign-in bound |
| RUNF3 | C, with the same pin and the same held listener | `TestStart_NoLoopback_0074` (all 3 subtests), `TestNoLoopback_Grok_0074` and `TestNoLoopback_BindsNothing_0083` (`openai`, `grok`, `plan`) pass: exit 0 |
| RUNF4 | C with three plants, one per method: Grok's, OpenAI's and the plan's sign-in each bind under `NoLoopback` | each caught by `TestNoLoopback_BindsNothing_0083`: `no_loopback_binds_0083_test.go:97: LoginBrowserOAuth with NoLoopback = OAuthSession(nil), oauth: listen for Grok callback: listenRecorder0083: no listener is bound in this test; want a session by the paste`; for OpenAI the same line with `oauth: registered callback ports unavailable; use device-code login: listenRecorder0083: …`; for the plan, `:124`, the same. OpenAI's and the plan's are also caught by the existing held-port tests. C unplanted: `gofmt -l` clean, `go vet` exit 0, tests exit 0 |
| RUNF5 | C with the engine planted not to pass `NoLoopback` to the browser login | `TestStart_NoLoopbackPassedOn_0074` fails: `start_browser_0074_test.go:335: browser login was given {hasInput:true noLoopback:false agentName:}; want NoLoopback and an InputCode` |
| RUNF6 | the session host, idle: listening TCP ports, and the ephemeral range | 12 listening ports, 5 of them in 49152 to 65535; `net.inet.ip.portrange.first` 49152, `.last` 65535 |
| RUNF7 | `scripts/check_api.py` and `tools/apidiff.py v1.31.0` on C and on `a3002aa` | identical: "0 incompatible change(s) outside llmprovider/x/"; the one compatible change listed is 0080's `x/signin.VerifyKilo`. The hook's unexported fields add nothing: both structs already hold `func` fields, so they were not comparable before either |

### The changes to this PLAN

* **Scope.** In scope gains, in `llmprovider/auth`:
  * the unexported `listen` field on `OAuthFlowOptions` and
    `ChatGPTPlanOptions`;
  * its copy on `oauthFlowConfig`, and the binds through it (C);
  * the new test file `llmprovider/auth/no_loopback_binds_0083_test.go`;
  * one comment line on `TestNoLoopback_Grok_0074` pointing at it, with
    its assertions unchanged.

  It also gains `llmprovider/x/signin/start_browser_0074_test.go`: the dial
  removed, the helper renamed, its doc comment. These are the same `auth`
  files D8 already changes; the hook goes in beside the advice.
* **Exported names.** None added. `make api-check` still shows only this
  record's two fields; RUNF7 shows the hook adds nothing.
* **Phase 1.**
  * Step 3 gains `TestNoLoopback_BindsNothing_0083`, written from the
    scratch copy's, with four plants: Grok, OpenAI and the plan each
    binding under `NoLoopback` (RUNF4), and the hook bypassed (a bind
    through `net.Listen` directly), which the control assertion catches.
  * At the base it is seen red the way the PLAN's other new tests are: the
    test file and the field declarations alone do not route the binds
    through the hook, so its "without `NoLoopback`" assertion fails.
  * Its OpenAI subtest swaps the registered ports for `nettest.Port` ports
    for its duration, as `newPlanLogin` does for the plan. A bind that
    bypasses the hook, at the base or under a plant, then binds no
    registered port (0045-MADR D2).
  * The x/signin test's change is a removed dial. It is not a loosened
    assertion: the assertion moves to where every bind is seen, and step
    3's planted breaks show it.
  * The stop condition on existing tests applies to everything else.
* **Phase 2** gains a run of the two tests 50 rounds under `-race`, with
  listeners held:
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=50 -timeout 20m -run 'TestStart_NoLoopback_0074$|TestNoLoopback_BindsNothing_0083$' ./llmprovider/x/signin/ ./llmprovider/auth/`.
  * While it runs, a helper script holds 1024 loopback listeners spread
    over 49152 to 65535. Each is checked free and released at the end, and
    the script raises its own open-file limit.
  * Under that hold the old form would fail a round with probability
    1024 / 16384, about 96 % over 50 rounds. The new form must pass all 50.
  * The execution record names the held ports' count and range, never
    another program's.
* **Verification** gains the same command.
* **Price.** Unchanged: Phase 1 mainstream, medium; Phase 2 mainstream, low.
  The hook is three binds and two fields, and the test exists in scratch
  form.

### Citations

The marker this PLAN has (`self=b109cea`) reads the rows below. Each cited
file is byte-identical at `a3002aa`. 0080-PLAN's item is cited by filename
and section, since that text is not at `b109cea`.

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P25 | The test | `llmprovider/x/signin/start_browser_0074_test.go:259` | `func TestStart_NoLoopback_0074(t *testing.T) {` |
| P26 | calls the dial's helper with the redirect | `llmprovider/x/signin/start_browser_0074_test.go:277` | `assertNothingListens(t, p, query.Get("redirect_uri"))` |
| P27 | which checks OpenAI's by shape only | `llmprovider/x/signin/start_browser_0074_test.go:320` | `if p != llmprovider.ProviderGrok {` |
| P28 | and dials Grok's | `llmprovider/x/signin/start_browser_0074_test.go:327` | `conn, err := net.DialTimeout("tcp", target.Host, time.Second)` |
| P29 | failing if anything answers | `llmprovider/x/signin/start_browser_0074_test.go:330` | `t.Errorf("something listens at %s; want no listener under NoLoopback", target.Host)` |
| P30 | The pass-through test | `llmprovider/x/signin/start_browser_0074_test.go:336` | `func TestStart_NoLoopbackPassedOn_0074(t *testing.T) {` |
| P31 | The fake issuer is a `RoundTripper` | `llmprovider/x/signin/fake_0074_test.go:59` | `func (f *fakeIssuer) client() *http.Client { return &http.Client{Transport: f} }` |
| P32 | Grok's no-listener port is random | `llmprovider/auth/oauth_loopback.go:445` | `return grokNoLoopbackPortMin + int(n.Int64()), nil` |
| P33 | Grok's bind | `llmprovider/auth/oauth_loopback.go:477` | `listener, err := net.Listen("tcp", "127.0.0.1:0")` |
| P34 | OpenAI's bind on IPv4 | `llmprovider/auth/oauth_loopback.go:504` | `v4, err4 := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", portText))` |
| P35 | and on IPv6 | `llmprovider/auth/oauth_loopback.go:505` | `v6, err6 := net.Listen("tcp6", net.JoinHostPort("::1", portText))` |
| P36 | `OAuthFlowOptions` carries unexported test fields | `llmprovider/auth/oauth_loopback.go:71` | `now   func() time.Time` |
| P37 | and so does `ChatGPTPlanOptions` | `llmprovider/auth/oauth_plan.go:87` | `now    func() time.Time` |
| P38 | Where the sign-in binds, or not | `llmprovider/auth/oauth_loopback.go:453` | `func browserEndpoint(provider llmprovider.ProviderID, noLoopback bool) ([]net.Listener, string, string, error) {` |
| P39 | `auth`'s Grok test | `llmprovider/auth/no_loopback_0074_test.go:188` | `func TestNoLoopback_Grok_0074(t *testing.T) {` |
| P40 | checks the redirect's shape | `llmprovider/auth/no_loopback_0074_test.go:241` | `if redirect.Scheme != "http" \|\| redirect.Hostname() != "127.0.0.1" \|\| redirect.Path != "/callback" \|\| err != nil \|\| port < 49152 \|\| port > 65535 {` |
| P41 | OpenAI's and the plan's tests hold substitute ports | `llmprovider/auth/no_loopback_0074_test.go:27` | `func holdPorts0074(t *testing.T) []int {` |
| P42 | 0045-MADR: a test that relies on a port staying unused | `docs/decisions/0045-MADR-tests-never-reuse-a-freed-port.md:37` | `unused can fail, and can make another package's test fail.` |
| P43 | and the ephemeral ranges | `docs/decisions/0045-MADR-tests-never-reuse-a-freed-port.md:108` | `` macOS and Windows (`49152–65535`), so no `:0` bind in any package is `` |
| P44 | The engine passes `NoLoopback` to the browser login | `internal/signinflow/signinflow.go:448` | `opts.OpenURL, opts.InputCode, opts.NoLoopback = openURL, inputCode, spec.NoLoopback` |

### Not verified

* L during the failing run; the failure rate under a full parallel
  `go test ./...`.
* C under `-race`, on Linux, on Windows, or at 50 rounds: those are Phase
  2's.
* Option D was not built.

## Amendment 2026-11-20: the state table's tests, re-grounding at each phase's start, and the base after 0081 and 0085

*(Added 2026-11-20 by the readiness pass, before any of this PLAN's code.
Nothing above is rewritten or removed. It carries out the MADR's amendment
of the same date, "the state table, before any code", whose `ST` rows it
cites. Facts are those of `24756a6`.)*

### Where the PLAN stands

* **Phase 0 is done; it had no execution entry.** `a3002aa` placed the two
  records and their index rows, `bf8e502` folded in the amendment of
  2026-11-17, and `8f133db` set the MADR `accepted` and this PLAN
  `in-progress`, each with its index row (`docs/README.md:185`, `:186`).
* **Phases 1 to 5 have not started.** At `24756a6` no Go file names `0083`,
  `NoBrowserQuestion` or `BindsNothing_0083`.
* **Phase 1 waits for the open rows.** The MADR's amendment has seven open
  rows (ST18, ST30 to ST32, ST38, ST51, ST65), put to the owner as four
  questions in one prompt. Phase 1 does not start until each is answered.
  *(Second pass, the same day: the owner ruled on those four; seven rows are
  open now, ST68, ST70 to ST72 and ST83 to ST85, and the same condition
  applies to them.)* *(Third pass: the owner ruled on those seven too; no
  row is open, and Phase 1 waits on no record.)*
  Each answer is then recorded by a dated note in the MADR, and where it asks
  for code, by a test here with its plant, before the build.

### The base, now

* "Prerequisites" defines the base as 0080-PLAN Phase 3's commit. 0080's code
  has landed (`00757ed`), and so has 0082's (`8051b00`, `04433ae`, `5f2c2ab`,
  `1545172`, `fd986c4`).
* The coordinator's order for the three records that share the wizard's
  files is 0081, then 0085, then this PLAN. **The base is now the commit on
  `main` that carries 0081-PLAN Phase 2's code, 0085-PLAN Phase 1's code,
  and any review fix of either.**
* "Prerequisites" names `0083-MADR-a-cancelled-sign-in-saves-nothing.md`.
  The record meant is
  [0081-MADR-a-cancelled-sign-in-saves-nothing.md](0081-MADR-a-cancelled-sign-in-saves-nothing.md).
* The files those two records change before this PLAN's build are in the
  MADR's amendment, "Build order, and the files the two earlier records
  change first".

### The first step of each unbuilt phase: re-ground at the phase's start commit

Each of Phases 1 to 5 begins with a step R, before its own step 1. The
execution record quotes its output.

**Phase 1, step R, at the base.** This is redone at the base after 0081's
and 0085's code, whatever was re-read before. The re-grounding of
2026-11-20 at `24756a6` does not stand in for it.

1. Run `git diff --stat 24756a6 <base>`. Then run `git diff 24756a6 <base> --`
   on each of these paths:
   * `wizard/configure.go`, `wizard/auth.go`, `wizard/prompter.go` and
     `wizard/kilo_verify.go`;
   * `llmprovider/x/signin/signin.go`;
   * `README.md`, `docs/architecture.md`, `CHANGELOG.md` and
     `docs/guides/adding-a-provider.md`;
   * every new `wizard/*_test.go`.

   Record each hunk that touches a symbol, a line or a doc this PLAN cites.
   Say whether `wizard/kilo_verify.go` changed. The coordinator's brief
   expects it to; none of the three scopes names it.
2. Run `python3 -B tools/claims.py --rev self=<base>` on the MADR and this
   PLAN, and capture its exit status. Read each failing row against the "now"
   tables of 2026-11-20 (the MADR's, and "Citations at `24756a6`" below).
   Record each row that moved. A row whose text is gone is a stop.
3. Re-read, at the base:
   * the symbols "Relation to 0080" lists;
   * 0081's unsaved step and `commitCredential`, and where the refused
     kept session's second sign-in now comes from (ST15);
   * the first checks of `ConfigureLLM` and `ResolveCredential`, beside
     0081's check after the step. D4's check must still come before any
     question (ST7);
   * `resolveChatGPTPlan`'s reads of the session and the client record
     (ST18, as the owner answers it);
   * 0085's `secretKey` and `resolveTokenStdin` (ST24, ST62);
   * `Prompter.Secret`'s doc beside `Select`'s (`wizard/prompter.go`);
   * the `[Unreleased]` sections 0081 and 0085 filled;
   * *(second pass)* 0081-MADR's Decision Outcome as it stands at the base,
     D4, D7 and D9 above all, on which ST48, ST50, ST53, ST54 and ST69
     rest. 0081's record is being amended on 2026-11-20 by its own readiness
     pass. A changed D4, D7 or D9 is a stop;
   * *(second pass)* whether 0081's or 0085's code adds a path to a second
     browser sign-in in one run (ST15). If it does, NT13 is written on that
     path and ST15 becomes decided by a dated note; if not, ST15 stays out of
     scope and NT13 is not written;
   * *(second pass)* the sites of the second pass's plants and changes:
     `stepRenderer.run`'s call of `signinflow.Start` (`wizard/driver.go:70`
     at `24756a6`), `pastePrompt.input` and `drain` (`wizard/auth.go:961`,
     `:1018`), `pasteCheck.check` (`internal/signinflow/paste.go:48`),
     `parseOAuthCallback` (`llmprovider/auth/oauth_loopback.go:763`),
     `listenLoopbackBothFamilies` and `isAddrInUse` (`:502`, `:565`), and
     `keysServer` (`llmprovider/auth/oauth_idtoken_test.go:17`).
4. Re-run the census (step 2) at the base. Its list must equal the Phase 1
   table plus whatever ST59 to ST61 place. A test that 0081 or 0085 added
   and that meets the question is classified by step 5's rule and recorded
   as a deviation. The phase goes on only if the rule places it without
   doubt.

**Phases 2 to 5, step R, at the phase's start commit.** Run
`python3 -B tools/claims.py --rev self=<start>` on both records, with its
status captured. Re-read every line the phase's own steps cite.

### Exit criterion of each unbuilt phase

Phases 1, 2, 3, 4 and 5 each end on, besides their own "Acceptance": **no
blocker, every row tested, plants caught.** "Every row" is every decided row
of the MADR's amendment that the phase's tests pin, by the map below.

### The test map: each decided row, and the test that pins it

| Row | Test | Phase |
| :--- | :--- | :--- |
| ST1 | `TestNoLoopbackOffByDefault_0083` | 1 |
| ST2 | `TestQuestionForm_0083`; NT1 | 1 |
| ST3 | `TestQuestionEndedInputIsThisMachine_0083` | 1 |
| ST4 | `TestQuestionExhaustedInputEnds_0083` | 1 |
| ST5 | `TestQuestionAnswerElsewhere_0083`; NT2 | 1 |
| ST6 | `TestNoLoopbackReachesTheLogin_0083`, `TestQuestionNoLoopbackStillOpens_0083` | 1 |
| ST7 | `TestNoLoopbackWithNoPastePromptRefused_0083` | 1 |
| ST8 | `TestNoBrowserQuestionKeepsToday_0083` | 1 |
| ST9 | `TestQuestionNotAskedWhereItMustNot_0083`; NT3 | 1 |
| ST10 | `TestQuestionNotAskedWhereItMustNot_0083`, with NT12 (second pass) | 1 |
| ST11 | `TestQuestionNotAskedWhereItMustNot_0083`, with NT12 (second pass) | 1 |
| ST12 | NT4 | 1 |
| ST13 | `TestQuestionForm_0083` | 1 |
| ST14 | `TestQuestionBeforeConsent_0083` | 1 |
| ST15 | none at `24756a6`: out of scope since the second pass, as no run reaches it; NT13 if step R finds a path. `TestQuestionOncePerRun_0083` now pins ST67 | 1 |
| ST16 | NT5 | 1 |
| ST17 | `TestNoLoopbackWithNoPastePromptRefused_0083` (ST7's case); NT2 | 1 |
| ST19 | `TestNoLoopback_BindsNothing_0083`, `TestNoLoopbackReachesTheLogin_0083` | 1 |
| ST20 | `TestNoLoopback_BindsNothing_0083`; `TestStart_NoLoopback_0074`, repaired; its 50 rounds | 1, 2 |
| ST21 | `TestNoLoopbackReachesTheLogin_0083`; NT2 | 1 |
| ST22 | `TestNoLoopbackReachesTheLogin_0083` | 1 |
| ST23 | `TestQuestionNotAskedForOtherMethods_0083`; NT6 | 1 |
| ST24 | `TestQuestionNotAskedForOtherMethods_0083`; NT6 | 1 |
| ST25 | `TestNoLoopbackOffByDefault_0083` | 1 |
| ST27 | `TestPortsAdvicePerMethod_0083`, with NT15 (second pass); `TestListenOpenAILoopback_ErrorsWhenAllBusy` | 1 |
| ST28 | `TestPortsAdvicePerMethod_0083` | 1 |
| ST29 | NT7 | 1 |
| ST34 | `TestPortsAdvicePerMethod_0083`, on CI's three platforms | 1, 5 (the tag's CI) |
| ST47 | `TestConfigureLLM_BrowserLoopbackWinDrainsPastePrompt`, `TestInputerContext_PasteWins_0057`, each scripting the answer | 1 |
| ST48 | NT8 | 1 |
| ST50 | NT9 | 1 |
| ST55 | `TestQuestionAnswerElsewhere_0083` | 1 |
| ST56 | `TestNoLoopbackOffByDefault_0083` | 1 |
| ST57 | `TestQuestionEndedInputIsThisMachine_0083`, `TestQuestionExhaustedInputEnds_0083` | 1 |
| ST59 | the census at the base, step R.4 and step 2 | 1 |
| ST60 | the census | 1 |
| ST61 | the census | 1 |
| ST63 | `TestNoLoopback_BindsNothing_0083`; `TestStart_NoLoopback_0074`, repaired; the 50-round run with 1024 listeners held | 1, 2 |
| ST64 | NT10; 20 rounds of `llmprovider/auth` under `-race` | 1, 2 |
| ST66 | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | 1, 2 |

The 40 decided rows all appear. Every new test this PLAN names maps to at
least one row: `TestNoLoopbackReachesTheLogin_0083` (ST6, ST19, ST21, ST22),
`TestNoLoopbackOffByDefault_0083` (ST1, ST25, ST56),
`TestQuestionAnswerElsewhere_0083` (ST5, ST55),
`TestQuestionNoLoopbackStillOpens_0083` (ST6), `TestQuestionForm_0083`
(ST2, ST13), `TestQuestionBeforeConsent_0083` (ST14),
`TestQuestionOncePerRun_0083` (ST15),
`TestQuestionNotAskedWhereItMustNot_0083` (ST9 to ST11),
`TestQuestionNotAskedForOtherMethods_0083` (ST23, ST24),
`TestQuestionEndedInputIsThisMachine_0083` (ST3, ST57),
`TestQuestionExhaustedInputEnds_0083` (ST4, ST57),
`TestNoLoopbackWithNoPastePromptRefused_0083` (ST7, ST17),
`TestNoBrowserQuestionKeepsToday_0083` (ST8), `TestPortsAdvicePerMethod_0083`
(ST27, ST28, ST34), `TestNoLoopback_BindsNothing_0083` (ST19, ST20, ST63),
and the repaired `TestStart_NoLoopback_0074` (ST20, ST63). None maps to no
row. The 30 existing tests of the Phase 1 table imply no new row. They keep
their own records' assertions, and the table's rule is the PLAN's stop
condition, not a behaviour.

*(Annotated 2026-11-20, by the later passes. This paragraph is the first
pass's. There are now 62 decided rows. `TestQuestionOncePerRun_0083` pins
ST67, not ST15, which no run reaches at `24756a6`. See "Second pass" and
"Third pass" below, and the MADR's amendment.)*

### Proposed tests for decided rows with none (dated addition to Phase 1)

Each is written in `wizard/browser_question_0083_test.go` unless the row
says otherwise, under "Rules for every phase". Each is seen red as Phase 1
step 4 says, and each plant goes in `PLANTS.json` for Phase 2 step 1: the
single production change that must turn the test red, on a scratch clone.

| Id | Rows | Test | Asserts | Plant |
| :--- | :--- | :--- | :--- | :--- |
| NT1 | ST2 | a case in `TestNoLoopbackOffByDefault_0083` | a `TextPrompter` given an empty line at the question: `NoLoopback` false at the login, today's instruction | the question's default index passed as 1 |
| NT2 | ST5, ST17, ST21 | cases in `TestQuestionAnswerElsewhere_0083` | the answer "On another device" through `ConfigureLLM` and through `ResolveCredential`, for OpenAI's and Grok's `browser_oauth`, a first plan sign-in (D5's URL-only words) and a plan sign-in again (the code-or-URL words) | (a) the first plan sign-in given the code-or-URL words after the answer; (b) the question skipped when the step is entered from `ResolveCredential` |
| NT3 | ST9 | a case in `TestQuestionNotAskedWhereItMustNot_0083` | `NoLoopback` with `NoBrowserQuestion`: no question, and `NoLoopback` true at the login | `spec.NoLoopback = o.NoLoopback && !o.NoBrowserQuestion` |
| NT4 | ST12 | `TestDriverWithNoLoopback_0083` | a `Driver` and `NoLoopback`: no `Select` with the question's title; `NoLoopback` true at the login; the `Driver`'s paste completes the sign-in | `NoLoopback` not set on the spec when a `Driver` is set |
| NT5 | ST16 | `TestQuestionAskedAgainNextRun_0083` | two `ConfigureLLM` runs with one `Options` value and one store, each answered "On another device": the question asked once in each run, `NoLoopback` true at each login | the answer kept in a package-level variable |
| NT6 | ST23, ST24 | cases in `TestQuestionNotAskedForOtherMethods_0083` | Kilo's device sign-in, with and without `KiloDeviceKey`; the pasted ChatGPT token; a kept session: no question. With `NoLoopback` set, Grok's device sign-in runs its device login as without it | the question asked in the `token_stdin` branch |
| NT7 | ST29 | `TestPortsBusyEndsTheRun_0083` | the browser login's stub returns an error starting "oauth: registered callback ports unavailable;" after "On this machine": the run returns it; the question was asked once; no further `Select`, `Confirm` or `Input`; no notice naming the other answer | the question asked again when the login returns that error |
| NT8 | ST48 | `TestQuestionThenCancelledRunSavesNothing_0083` | *(second pass)* Grok's `browser_oauth` through the login seam, which has no consent step: a `Prompter` whose question `Select` cancels the run's context and then answers "On another device": the run returns an error matching `context.Canceled`; the store saved nothing; the login's stub, if called, saw an ended context | `signinflow.Start` given `context.WithoutCancel(ctx)` in `stepRenderer.run` (`wizard/driver.go:70`) |
| NT9 | ST50 | `TestNoLoopbackCancelDuringPasteSavesNothing_0083` | `NoLoopback`; an `InputerContext` `Prompter` whose paste read returns only when its context ends; the run's context cancelled during the paste, and in a second case its deadline passed: an error matching `context.Canceled`, then `context.DeadlineExceeded`; nothing saved; the run on a goroutine, done within 5 s. *(Fourth pass.)* The fake's plain `Input` blocks until the test's cleanup releases it, and asserts nothing | `pastePrompt.read` calls `Input`, not `InputContext`: the read cannot be ended, `drain` waits for an Enter that never comes, and `stepRenderer.run`'s `wg.Wait()` (`wizard/driver.go:155`) holds the run past the 5 s bound |
| NT10 | ST64, ST65 | `TestVerifyIDToken_StaleKeysForAReusedURL_0083`, in `llmprovider/auth` | *(second pass, by the owner's ruling on ST65)* a port from `nettest.Port`; a fresh entry put in `jwksCache` under `http://127.0.0.1:<port>/jwks`, as an earlier server on that port leaves it; a server built on that port by the helper beneath `keysServer`; then `TestVerifyIDToken_UnknownKidRefetchesOnce`'s sequence: one fetch to populate, one refetch for the unknown kid, the published key found, no further fetch | the helper's clearing removed: a change to a test helper, since the ruling allows no production change. Expected red then with "0 key fetches to populate the cache, want 1" (not run: this pass runs no test) |

**Phase 2 gains** a run of the package that showed the flake, as it showed
it, through the queue:
`~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=20 -timeout 45m ./llmprovider/auth/`.
Its status is captured before any filter. `TestVerifyIDToken_UnknownKidRefetchesOnce`
must not fail.

### Citations at `24756a6`

`python3 -B tools/claims.py` on this PLAN at its marker (`self=b109cea`):
44 of 44 rows pass (exit 0). With `--rev self=HEAD`: 32 of 44 (exit 1).
Every failing row's text was found on another line; none is missing. The
ids end in "now", so that `tools/claims.py` does not read them against
`b109cea`.

| Id | At `b109cea` | At `24756a6` |
| :--- | :--- | :--- |
| P1 now | `wizard/prompter.go:72` | `:78` |
| P2 now | `wizard/configure.go:200` | `:238` |
| P3 now | `wizard/auth.go:812` | `:888` |
| P4 now | `wizard/auth.go:824` | `:900` |
| P5 now | `wizard/auth.go:765` | `:841` |
| P6 now | `wizard/auth.go:801` | `:877`, the paste flow's call (`:867` is the `NoPastePrompt` branch's) |
| P7 now | `wizard/auth.go:218` | `:223` |
| P20 now | `README.md:381` | `:427` |
| P21 now | `README.md:471` | `:553` |
| P22 now | `docs/architecture.md:1010` | `:1092` (`:389` is a level-three heading of the same words) |
| P23 now | `docs/architecture.md:1073` | `:1192` |
| P44 now | `internal/signinflow/signinflow.go:448` | `:506`, the browser branch (`:475` is the plan's) |

The other 32 rows, P8 to P19 and P24 to P43, hold on their lines at
`24756a6`. The `wizard/`, `README.md`, `docs/architecture.md` and
`CHANGELOG.md` rows move again with 0081's and 0085's code (step R).

### Execution record: the readiness pass (2026-11-20)

* **Read, at `24756a6`:**
  * both records whole;
  * `AGENTS.md` and `docs/README.md`'s rows;
  * the Decision Outcome of
    [0081-MADR-a-cancelled-sign-in-saves-nothing.md](0081-MADR-a-cancelled-sign-in-saves-nothing.md)
    and its amendments, and of
    [0085-MADR-refuse-an-empty-typed-key.md](0085-MADR-refuse-an-empty-typed-key.md);
  * the scopes of both sibling PLANs, read from `HEAD` with `git show`;
  * 0082's N-5 entries;
  * the code named in the MADR's amendment.
* **Run:** `tools/claims.py` on both records, at their markers (exit 0) and
  with `--rev self=HEAD` (exit 1, every row moved, none gone).
* **Not run:** any test, gate or build. This pass is records only.
* **Changed:** this PLAN and its MADR, by these two dated additions; no
  other file.

### Second pass (2026-11-20): the review and the owner's rulings

*(Added the same day, after an independent review of this amendment and the
owner's rulings on its open rows. The MADR's amendment carries the rulings,
quoted by label, and the rows; this section carries the build.)*

**Scope gains:**

* `internal/signinflow/paste.go`: the retry of a schemeless paste (MADR D11,
  ST76 to ST82), with its test file. If the owner's answer to ST84 says so,
  `parseOAuthCallback` in `llmprovider/auth/oauth_loopback.go` too.
* `llmprovider/auth/oauth_loopback.go`: the advice for a bind failure other
  than "in use" (ST30, ST32), beside D8's.
* `wizard/auth.go`: `pastePrompt.input` (ST38); `drain` (ST51, ST69); the
  question asked through `choose` (ST73), right before `browserSignIn` in
  the plan path (ST18).
* `llmprovider/auth/oauth_idtoken_test.go`: `keysServer` and a helper beneath
  it that takes the listener (ST65).
* `llmprovider/x/signin/`: NT22's test.
* *(fourth pass)* `internal/signinflow/signinflow.go`, `askPaste`: where a
  paste passes the check, it returns the address with `http://` in front
  (`:603` to `:605` return the raw `answer.Paste` at `24756a6`). The
  corrected address comes from a second unexported function beside `check`
  in `paste.go`, which returns the text to hand on. `check` keeps its
  signature and its single return of a reason, so
  `TestEngine_PasteChecks_0074` (`internal/signinflow/signinflow_0074_test.go:230`,
  `:244`) is unchanged and the stop on existing tests is not met. A changed
  return of `check` would change that test; that design is not taken.
* *(fourth pass)* the drain's cause. `drain` takes no input
  (`wizard/auth.go:1018`; the renderer's field is `func()`,
  `wizard/driver.go:59`), and the renderer calls it before it reads the
  login's result (`wizard/driver.go:151` to `:156`). In scope, then:
  * the field's type, at `wizard/driver.go:59`, becomes a `drain` that
    takes the cause;
  * `stepRenderer.run` reads `end`, the login's result, before it calls
    `drain`, then waits on its goroutines as now;
  * `browserFlowOptionsFor`'s return (`wizard/auth.go:880`), and
    `pastePrompt.drain`, take the cause.

  The cause gives the notice:
  * the run's context ended (a cancel, or the run's deadline, which
    0081-MADR D4 counts as a cancel): "Sign-in cancelled; press Enter to
    continue";
  * otherwise, the login's error matching `context.DeadlineExceeded` (its
    own limit): "Sign-in timed out; press Enter to continue";
  * otherwise, a finished sign-in or any other end: "Browser sign-in
    finished; press Enter to continue", as today, since no ruling changes
    it.

  Step R.3 confirms at the base that `run.Steps()` closes only once the
  login has returned, so that reading `end` first cannot wait on a paste
  that `drain` alone would end. If it does not, that is a stop.
* No exported name is added; everything above is unexported.

**Step 7's docs gain:**

* `Options.Driver`'s doc: no question about the browser is asked under a
  `Driver` (N3; the review's case 3); and `StepDriver`'s doc, if it says what
  the wizard asks.
* `CHANGELOG.md` "Changed":
  * a pasted redirect address without `http://` is read as the address,
    for every caller of the engine, `x/signin`'s included (D11);
  * a bind failure other than a port in use advises the sign-in with no
    listener (ST30, ST32);
  * with no listener, a paste answered empty at once ends the sign-in
    (ST38);
  * the notice after a cancel or a timeout no longer says "finished" (ST51,
    ST69).

**The stop on error texts** ("an error text changes other than D8's advice")
also admits ST70's, ST71's and ST72's words, once the owner approves them.
*(Fourth pass.)* It also admits ST85's re-ask, "That is not a usable
address", which the owner's ruling of the third pass gives.

**Phase 2's plants** gain NT12 to NT22's, in `PLANTS.json`.

**The test map, second pass.** These rows were decided in this pass, or had
tests that could not go red:

| Row | Test | Plant that turns it red | Phase |
| :--- | :--- | :--- | :--- |
| ST10, ST11 | NT12 | `spec.NoLoopback` set true when a `Driver` is set | 1 |
| ST18 | NT14 | the question asked right after the method menu | 1 |
| ST27 | NT15 | the browser's old text restored | 1 |
| ST30, ST32 | NT16 | the in-use advice given for every failure | 1 |
| ST38 | NT17 | the wait for the login's end kept under no listener | 1 |
| ST48 | NT8, revised above | as NT8's row | 1 |
| ST50 | NT9 | as NT9's row | 1 |
| ST51 | NT18, NT19 | the old "finished" words kept after a cancel; after a timeout | 1 |
| ST65 | NT10, revised above | the helper's clearing removed (a test helper) | 1 |
| ST67 | `TestQuestionOncePerRun_0083`, re-scoped: the kept session's offer, the refusal's notice, the method menu, the question once, then the login | the question asked when the kept session is offered | 1 |
| ST69 | NT18 | *(fourth pass)* no notice shown after a cancel: `drain` returns silently. (The second pass's plant, `drain` returning without waiting for the pending read, could not turn NT18 red: `stepRenderer.run` still waits on the paste goroutine, `wizard/driver.go:155`, which stays in `Input`, `:282` to `:290`.) | 1 |
| ST73 | NT20 | the question's raw `Select` read directly, any index but 0 taken as "On another device" | 1 |
| ST76 | NT21, NT22 | the retry removed | 1 |
| ST77 | NT21 | the retry made only where the step takes a code or a URL | 1 |
| ST78 | NT21 | the retry skipped where the step takes a code only | 1 |
| ST79 | NT21 | the state check skipped after the retry | 1 |
| ST80 | NT21 | the prefix tested before the paste is trimmed | 1 |
| ST81 | NT21 | `https://` tried first | 1 |
| ST82 | NT21 | *(fourth pass)* the retry made, and its result kept, for any paste that the first parse does not read as a URL with a scheme: the host-and-port check removed, so `12:34/x` and `abc/def` are read as addresses and refused for their state. (The second pass's plant left ST83's equality check in place, which still rejects them.) | 1 |

`TestQuestionOncePerRun_0083` as first written (a second sign-in in one run,
asked once) could not go red, since no run reaches a second sign-in; it now
pins ST67, and ST15 waits on step R.

**Proposed tests of the second pass** (dated addition to Phase 1, in
`wizard/browser_question_0083_test.go` unless the row says otherwise):

| Id | Rows | Test | Asserts |
| :--- | :--- | :--- | :--- |
| NT12 | ST10, ST11 | assertions in `TestQuestionNotAskedWhereItMustNot_0083` | the login's `NoLoopback` under each setting: false under `NoBrowserQuestion`, `NoPastePrompt` and a `Driver`; true under `NoLoopback` |
| NT13 | ST15 | `TestQuestionReusedInOneRun_0083`, written only if step R finds a path to a second browser sign-in in one run | the question asked once; both logins get the first answer's `NoLoopback`; plant: the answer not kept for the run |
| NT14 | ST18 | `TestQuestionAfterThePlanReads_0083` | a first plan sign-in: (a) the client record's read refused with a permission error: the run's error, and no question asked; (b) an undecodable record: the warning notice comes before the question's `Select` |
| NT15 | ST27 | an assertion in `TestPortsAdvicePerMethod_0083` | the browser sign-in's error says "free ports P and Q, or use device-code login", with the two ports it was given |
| NT16 | ST30, ST32 | `TestPortsAdviceOtherFailures_0083`, in `llmprovider/auth` | the listen hook refuses every bind with a permission error that is not "in use": OpenAI's browser and plan sign-ins give the prefix and ST70's advice, and do not say "free ports"; Grok's gives "oauth: listen for Grok callback:" and the same advice; no bind reaches the system |
| NT17 | ST38 | `TestNoLoopbackQuickEmptiesEndAtOnce_0083` | `NoLoopback`; a `Prompter` whose paste answers come empty at once: the run returns ST71's error within 5 s, on a goroutine |
| NT18 | ST51, ST69 | `TestNoLoopbackCancelWithPlainPrompter_0083` | `NoLoopback`; a `Prompter` that is not `InputerContext`, whose paste read blocks until a line is fed; the run's context cancelled. The test waits, up to 5 s, for ST72's cancel notice, "Sign-in cancelled; press Enter to continue", and fails if it does not come; only then does it feed the line *(fourth pass)*. The run has not returned before the line; once it is fed, an error matching `context.Canceled` within 5 s; nothing saved |
| NT19 | ST51 | `TestDrainAfterTheLoginsLimit_0083` | `NoLoopback`; a `Prompter` that is not `InputerContext`; the login's stub returns the login's timeout while the paste read is pending: ST72's timeout notice, not "finished" |
| NT20 | ST73 | `TestQuestionOutOfRangeIsAnError_0083` | the question answered with index 5: a plain error naming the choice out of range; no login called |
| NT21 | ST76 to ST82 | `TestPasteSchemelessCallback_0083`, in `internal/signinflow` | an authorize URL whose `redirect_uri` is `http://127.0.0.1:1455/auth/callback`. First, at the base, `127.0.0.1:1455/auth/callback?code=c&state=<state>` is seen taken as a bare code (the review's inference shown). Then: it passes where the step takes a code or a URL (ST76); with a `client_id` it passes for a first plan sign-in, and without one it is refused (ST77); it is refused where the step takes a code only (ST78); another state, and none, are refused as "not from this sign-in" (ST79); the same with spaces and a line break around it passes (ST80); the retried URL's scheme is `http` (ST81); `12:34/x` and `abc/def` are taken as bare codes (ST82) |
| NT22 | ST76 | `TestStart_SchemelessPasteCompletes_0083`, in `llmprovider/x/signin` | a browser sign-in with `NoLoopback`, the real `auth` code against the fake issuer: a schemeless paste of the redirect completes it |

Each new test's plant is in the map above. The open rows' answers may add
rows, and with them tests; those come by a dated note before Phase 1.

**Execution record, second pass:**

* **Read, at `24756a6`:** each of the review's cited lines; `net/url`'s
  `getScheme` and `parse` in the Go 1.27.4 tree; `stepRenderer.run`;
  `credentialStep`'s refusal loop.
* **Run:** `tools/claims.py`, record lint and the identifier scan on the two
  records (statuses in the hand-back).
* **Not run:** any test.

### Third pass (2026-11-20): the last seven rows, decided

*(Added the same day, after the owner's rulings on ST68, ST70 to ST72 and
ST83 to ST85; the MADR's amendment quotes their labels. Nothing else
changes.)*

**What the code does, decided:**

* **`llmprovider/auth`, the bind failures.** When any registered port failed
  "in use", the error keeps D8's "free ports" advice, with every reason joined
  after it (ST68). When none did, it keeps its prefix and advises: "this host
  does not allow the callback listener: sign in with no listener and paste the
  address back (in the wizard, answer that the browser is on another device;
  in code, set NoLoopback)". Grok's failure gets the same advice after
  "oauth: listen for Grok callback:" (ST30, ST32, ST70).
* **`wizard/auth.go`, `pastePrompt.input`.** With no listener, three empty
  answers at once end the sign-in with "wizard: the prompter answered the
  paste with nothing, three times at once; with no callback listener only a
  paste can finish the sign-in", matching `llmprovider.ErrInvalidRequest`
  (ST38, ST71).
* **`wizard/auth.go`, `drain`, in every flow.**
  * After a cancel: "Sign-in cancelled; press Enter to continue".
  * After the login's limit: "Sign-in timed out; press Enter to continue".
  * After a finished sign-in: "Browser sign-in finished; press Enter to
    continue", as today (ST51, ST69, ST72).
* **`internal/signinflow/paste.go`, the retry.**
  * The engine alone retries a paste that begins with the registered
    callback's exact host and port, with `http://` in front. It keeps the
    result only when its host and port equal the `redirect_uri`'s.
  * It hands `auth` the address with `http://` in front.
  * A paste that begins so, but whose retried parse fails, is refused with
    "That is not a usable address", asked again, and counted among the three
    pastes.
  * `localhost:<port>` and `[::1]:<port>` are read as today (ST83 to ST85).
  * `auth`'s `parseOAuthCallback` is unchanged. The doc comments of
    `LoginBrowserOAuth` and `LoginChatGPTPlan` say that a schemeless address
    given to their `InputCode` is taken as a bare code (ST88).

**Scope, for ST72's change to the existing flow's notice** (every quote of
"Browser sign-in finished; press Enter" at `24756a6`, checked in Phase 1):

* `wizard/auth.go:1037`, the notice: changed as above.
* `wizard/auth_test.go:464` and `wizard/inputctx_0057_test.go:182` count
  "press Enter to continue" after a loopback win. That notice is unchanged,
  so they stay as they are.
* `wizard/auth_test.go:494`, `wizard/inputctx_0057_test.go:71` and `:142`:
  Prompters that react to "press Enter". The new notices hold it too, so
  their behaviour is unchanged.
* `wizard/remediation_test.go:88` sends the words as a notice in a race
  test and asserts nothing of the drain; unchanged.
* `docs/decisions/0008-MADR-repair-oauth-loopback-and-session-wiring.md:605`
  and `docs/decisions/0057-MADR-wizard-device-code-and-paste-prompt.md`
  (C19, C20) describe the notice after a loopback win, which stays. The
  dated notes this PLAN already writes in 0008-MADR D3 and 0057-MADR D2 add
  the two new notices.
* `docs/decisions/0008-PLAN-repair-oauth-loopback-and-session-wiring.md:1006`
  and `docs/decisions/0057-PLAN-wizard-device-code-and-paste-prompt.md:575`
  are execution history; not changed.
* `README.md`, `docs/architecture.md` and `CHANGELOG.md` quote it nowhere.
  The CHANGELOG "Changed" entry of the second pass names the two notices.

**Tests, adding to the second pass's** (each plant goes in `PLANTS.json`):

| Id | Rows | Test | Asserts | Plant |
| :--- | :--- | :--- | :--- | :--- |
| NT16, NT17, NT18, NT19 | ST70, ST71, ST72 | as the second pass wrote them | each asserts its row's words; NT17 also asserts that the error matches `llmprovider.ErrInvalidRequest` | the words changed; for NT17, the error made plain |
| NT21 | ST83, ST84, ST85 | cases added to `TestPasteSchemelessCallback_0083` | `127.0.0.1:14550/…` and `127.0.0.1:1457/…`, with a callback on 1455, are taken as bare codes; `localhost:1455/…` is read as today; `[::1]:1455/…` is a bare code; the text handed on for a qualifying paste begins `http://`; `127.0.0.1:1455/%zz?…` is refused with "That is not a usable address" and counted | (a) the retry kept for any port on the callback's host; (b) the address handed on without `http://` (NT22 then fails too); (c) a failed retry taken as a bare code |
| NT23 | ST68 | `TestPortsAdviceMixedFailure_0083`, in `llmprovider/auth` | the listen hook fails the first port "in use" and the second with a permission error: the error gives the "free ports" advice and holds both reasons | the other-failure advice given whenever any port failed for another reason |
| NT24 | ST72 | `TestDrainNoticeWithAListener_0083` | "On this machine": after a loopback win, "Browser sign-in finished; press Enter to continue"; after a cancel, "Sign-in cancelled; press Enter to continue"; after the login's limit, "Sign-in timed out; press Enter to continue" | the drain's notice chosen without its cause |

**Phase 4, dated note (2026-11-20):** the owner's ruling on live runs,
"API keys from the environment only (Recommended)" (relayed by the coordinator), leaves with the owner any step that
needs a browser sign-in, a phone or a real terminal, so Phase 4 stays the
owner's, and optional.

### Fourth pass (2026-11-20): corrections

*(Added the same day, after a check of the earlier passes found five
items. Corrections only; no row and no ruling changes.)*

* **NT18's plant** (ST69) is now "no notice after a cancel", and NT18 waits
  for the notice. The old plant could not turn the test red. **NT9's row**
  now says that its fake's `Input` blocks.
* **The corrected address** reaches `auth` through `askPaste` and a second
  function beside `check`. `TestEngine_PasteChecks_0074` is unchanged (scope,
  second pass).
* **ST82's plant** now removes the host-and-port check.
* **The drain's cause** reaches the notice through the renderer's `drain`
  field, read after `end` (scope, second pass).
* **The stop on error texts** admits ST85's re-ask.
* **The first pass's map paragraph** is annotated.
