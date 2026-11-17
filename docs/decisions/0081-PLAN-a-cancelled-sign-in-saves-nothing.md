---
status: proposed
date: 2026-11-17
associated-madr: "0081-MADR-a-cancelled-sign-in-saves-nothing.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement a Cancelled Sign-In Saving Nothing, and ConfigureLLM Saving Last

Associated MADR: [0081-MADR-a-cancelled-sign-in-saves-nothing.md](0081-MADR-a-cancelled-sign-in-saves-nothing.md)

## Goal

After this PLAN, in the next release with
[0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md),
on the owner's word (MADR D8):

* a run of `ConfigureLLM` or `ResolveCredential` whose context ends, by
  cancel or by its deadline, before the save saves nothing and returns the
  context's error, for every provider, session kind and key (MADR D2 to D4);
* `ResolveCredential` saves inside the shared step right after its check,
  and its Result means the session is saved (D5);
* `ConfigureLLM` saves as the last thing it does, after the model and
  fallback questions, so any error from it means nothing was saved (D5,
  N2);
* the docs say what is and is not guaranteed (K7, D5, D6, D9).

The MADR is `proposed`, and so is this PLAN. The owner answered Q1 to Q7
and N1 to N4 on 2026-11-17, and the coordinator applied N5 (a) (the MADR's
"Decided"); the coordinator also applied N6 (a), how the probe shows the
person what a sign-in needs. The steps below are written to them; nothing
is open. Nothing below Phase 0 starts before the owner approves this
PLAN.

## Prerequisites

* 0080-PLAN Phase 3's code is committed on `main`, and no other agent holds
  staged or uncommitted work in the tree (0080's later phases included).
  This PLAN's Phase 2 starts on that commit, called **the base** below.
* The drafting's reproduction, `cancel_save_0081_test.go` (eight tests) and
  `save_at_end_0081_test.go` (seven tests), written and run against scratch
  copies while drafting, is handed to the Phase 2 engineer by the
  coordinator. It is not committed with the records: the bootstrap
  exception takes records only.

## Relation to 0080

* **Order.** This PLAN runs after 0080-PLAN Phase 3 is committed and before
  0080-PLAN Phase 5 (the release), so both ship together. Its heavy runs
  never run beside 0080's on one host.
* **Files both touch.** 0080's Phase 3 changed `wizard/configure.go`
  (`credentialStep`, with `checkCredential` and `endedCheck` before the
  save; the doc comments of `Options`, `ConfigureLLM` and
  `ResolveCredential`) and `wizard/auth.go` (`resolveKiloDevice`,
  `resolvedCredential`'s fields), and added `wizard/kilo_verify.go`
  (`checkEnded`, `verifyKilo`) and tests that call `credentialStep`
  directly. This PLAN restructures `ConfigureLLM`, `ResolveCredential` and
  `credentialStep` in `wizard/configure.go`, and changes a doc comment in
  `wizard/auth.go`.
* **What Phase 2 re-reads at the base,** recording each line in the
  execution record before any edit:
  * `credentialStep`: its signature, every return, the `endedCheck` and
    the save at its end; its doc comment's sentences on when a session is
    saved (0080 rewrote them);
  * `ConfigureLLM` and `ResolveCredential`: where each calls the step, the
    model-id branch, the model and fallback calls, every return, and
    `ResolveCredential`'s own context check after the step (MADR C9);
  * `resolvedCredential.save` (C1 to C3); `checkEnded` and `verifyKilo`;
  * every test that calls `credentialStep` directly: at 0080's staged tree,
    `TestKiloVerify_EndedContext_0080` in
    `wizard/kilo_verify_ctx_0080_test.go` (twice). Its signature stays
    (N5 (a), applied), so it must compile and pass unchanged;
  * 0080-PLAN's execution record, "Found, not mine to change", which this
    record answers.
* **The release.** "All four in the next release with 0080, on my word
  (Recommended)", the owner's answer to Q7: the tag and the Release wait for
  the owner's word.

## Scope

### In scope

* `wizard/configure.go`: the unsaved step and `commitCredential` (one
  context check, then `save`); `credentialStep` as the two in order;
  `ResolveCredential` without its own check after the step;
  `ConfigureLLM` calling the unsaved step, checking the context once after
  it (N3), and committing as the last thing before each successful
  return; the doc comments of `credentialStep`, `ConfigureLLM`,
  `ResolveCredential` and `Options.TokenStore`.
* `wizard/auth.go`: the doc comments of `resolvedCredential.save` and of
  `saveTo` (it is due to the store at the run's end, not the listing's).
* `wizard/cancel_save_0081_test.go` and `wizard/save_at_end_0081_test.go`.
* `llmprovider/x/signin/signin.go`: one sentence in `Wait`'s doc (D7).
* Phase 1's probe (N1): `llmprovider/live_0081_grant_test.go`, live-tagged,
  its checks in `llmprovider/live_checks_test.go` and their offline twin in
  `llmprovider/live_twins_test.go`.
* Docs: `README.md` (the wizard's sign-in bullet), `docs/architecture.md`
  ("When a session is saved", and the wizard's paragraph on the save),
  `CHANGELOG.md` `[Unreleased]` (N4: **Fixed** and **Changed**); dated
  one-line notes (MADR D11) in 0059-MADR W2, its amendment and W8,
  0056-MADR D1, 0080-MADR D1, and 0080-PLAN's "Found, not mine to change".

### Exported names (R55)

None added, removed or changed. The functions this PLAN adds are
unexported.

### Out of scope

* Revoking a session the provider issued (D6, the owner's answer to Q5).
* `x/signin`'s behaviour and the plan's client record (D7).
* `TokenStore`, `ClientStore` and `FileTokenStore` (D10).
* The Result's token facts being the sign-in's (MADR "Found beyond").
* gobble-cli's and prepare-commit-msg's changes (Phase 5 names them; their
  records make them).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts on the commit of the one before it, on `main`.
* **One commit: the tests with the code.** Modelled on 0080's ruling of
  2026-11-17 ("One commit: the tests with phase 3's code"), proposed here
  for the owner's approval with the PLAN: the tests are seen red on a
  scratch clone of the base and land with the code in one commit, so
  `main` never carries a failing build.
* **Red, then green.** Each new test Phase 2's table marks red is seen red
  at the base, and at `b109cea`, with `python3 -B tools/redgreen.py
  RED.json --expect fail --rev <rev>`; every new test, the guards
  included, is green on the phase's tree with `--expect pass`; each is seen
  to fail on a planted break with `python3 -B tools/plant.py PLANTS.json`,
  on a scratch clone, never in the tree. The execution record quotes each
  failure line. A break no test catches is a deviation.
* **Heavy runs queue.** Every `-race` run, the pre-add check and the gate
  run through `~/.agents/skills/available-dev-hosts/scripts/run-heavy --
  <command>` on the session host, or through the skill's
  `scripts/run-on-host` on the host
  `~/.agents/skills/available-dev-hosts/scripts/pick-host` prints. No
  `-race` run goes to a WSL distro. A single-package `go test` without
  `-race` may run on the session host.
* **No outside host in an offline test** (0065-MADR): the sign-ins are the
  wizard's seams or an issuer the run's `HTTPClient` serves in process; the
  listing is a `RoundTripper`.
* **No credential in a record or a test** beyond the package's existing
  placeholders (`testOAuthSession`'s, `jwtShapedAccess`,
  `kiloWizardToken`) and a test's own fixed made-up key; a live run prints
  status codes and counts only.
* **Stop conditions,** each a stop and a prompt to the owner:
  * Phase 1 gives `ended` for any provider, `indeterminate` twice for one,
    `refresh_token_reused` from any refresh, or a provider is not run;
  * a line the probe prints shows anything like a token, a code or a URL;
  * an existing test, golden or fixture needs a change (the ones to watch:
    the seven `TestSession_*_0059`, `TestKiloVerify_EndedContext_0080`,
    `TestConfigureLLM_BrowserAndDevicePersistSessions`, the
    `select_ctx_0026` and `entry_0059` tests);
  * an error's text changes other than as MADR D9 names;
  * `make api-check` or `tools/apidiff.py v1.31.0` reports any change of
    this record's;
  * a new test cannot be red at the base without changing code that is not
    a test.

## Implementation Steps

### Phase 0: records

Model: lightweight, low.

1. The coordinator places this pair after 0080's Phase 3 commit, as the
   drafting's `PLACEMENT.md` says (number by `python3 -B
   scripts/check_records.py --next`, rows by `python3 -B tools/records.py
   add`), under the bootstrap exception (records only).
2. `tools/records.py status` sets the MADR `accepted` and this PLAN
   `in-progress` when the owner says so.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B
   tools/gate.py --markdown`, then stage and commit (`git commit
   --no-edit`).

Acceptance: `make records-check links-check` pass; `python3 -B
tools/claims.py` passes on both records; no source changed. Docs: the
records are the docs.

### Phase 1: the earlier grant, measured by the owner (N1)

The owner's answer to N1: "A live probe by me before release
(Recommended)". No agent runs the live test, signs in, or holds a token or
a session. Phase 2's code does not land before Phase 1c is recorded and no
stop condition below is met.

**What the probe asks, for each of four sign-ins** (MADR D12): after a
person signs in to the same provider a second time, does the first session
still work? The four are the ChatGPT browser sign-in (OpenAI, Codex's
client), the ChatGPT plan sign-in (OpenAI, the app's registered client,
signed in to again with the first session as `Previous`, as the wizard
does), Grok's browser sign-in, and Kilo's device sign-in. The OpenAI and
Grok device sign-ins reach the same issuers and are not probed (named in
the MADR's "Not verified").

#### Phase 1a: the probe (an agent)

Model: mainstream, medium (a person-gated live test in the 0046 form, with
an offline twin and a redaction proof).

Files: `llmprovider/live_0081_grant_test.go` (`//go:build live_gateways`),
the checks in `llmprovider/live_checks_test.go` and their twin in
`llmprovider/live_twins_test.go` (untagged; AGENTS.md "Live tests", 0046-MADR
D3).

1. **The switch.** `TestLive_EarlierGrant_0081` skips unless
   `LLMPROVIDER_LIVE_0081` is `1`, and runs only the provider
   `LLMPROVIDER_LIVE_0081_PROVIDER` names: `chatgpt`, `plan`, `grok` or
   `kilo` (one per run, so each run is one person's sign-ins and one
   verdict). Any other value fails before any request. Both are a record's
   own variables, which `scripts/check_tables.py` accepts (P19).
2. **Where the sessions live.** In the test's memory only. The test opens no
   `TokenStore` and no `ClientStore`, passes none to any sign-in, reads no
   environment variable but its two, and never touches the owner's real
   stores (the wizard's, gobble's `auth.json`, prepare-commit-msg's, the
   vendors' CLIs' files). Each session's `Store` stays nil, so a refresh
   persists nothing (MADR C34) and takes no file lock; nothing is written to
   disk, and the sessions end with the process. The plan's two sign-ins share
   one random host id generated by the test, as one host would.
3. **What the person sees, and where** (N6 (a), the coordinator's
   application; the owner's standing constraint that no URL with a code,
   code, pasted text, token or client id is displayed in or recorded from a
   probe's output). Two writers, and nothing crosses between them:
   * **The browser.** Each authorize URL (ChatGPT, the plan, Grok) and
     Kilo's verification page are handed to the system's opener (`open` on
     macOS, through `OAuthFlowOptions.OpenURL` and the plan's `OpenURL`)
     and shown nowhere else. No paste prompt is offered: the browser's
     return to the loopback completes the sign-in, so nothing is pasted.
   * **The terminal device**, `/dev/tty`, opened by the test for writing
     only: Kilo's user code, once per sign-in, as `Kilo code: <code>`, and
     nothing else. It never receives a URL, a token or an id.
   * **The test's output** (`t.Log`, `t.Error`, `t.Fatal` and their `f`
     forms), which the owner sends back whole: the step lines and the
     verdict only (step 6).
   * **When either cannot be done**, the run of that provider stops at
     once, before any request that needs it, with one plain line in the
     output and no fallback: `<provider>: stopped: the terminal device
     could not be opened`, or `<provider>: stopped: the browser could not
     be opened`. Neither line carries the code, the URL or the error's
     text. The run fails; nothing is printed in their place.
   * The writers are seams the twin replaces (step 7): the opener and the
     terminal writer are variables of the test, the real ones in the
     live-tagged file.
4. **The sequence**, one session at a time, nothing concurrent:
   1. `sign-in-1`: the first sign-in; session A.
   2. `baseline`: A is used once (below). This shows A works before the
      second sign-in, so a later failure is not A's own.
   3. `sign-in-2`: the second sign-in; session B. For the plan, `Previous`
      is A.
   4. `first-after`: A is used once more. This is the measurement.
   5. `control`: B is used once. This shows the service answers B now, so a
      failure of A is not an outage.
5. **"Uses the first session", per provider.**
   * **ChatGPT, the plan, Grok** (refreshing sessions): one forced refresh,
      `session.Invalidate()` then `session.Token(ctx)`: the request a
      revoked grant cannot pass. Not an access-token request: an access
      token is a signed JWT that may pass until its expiry whatever the
      grant (named, not measured). Each refresh token is sent exactly once:
      `baseline` spends A's first refresh token and receives A's second;
      `first-after` spends A's second, its first use. The probe never sends
      a refresh token twice, never shares one between A and B (two separate
      sign-ins, two grants), has no store from which an old token could be
      reloaded, and runs one refresh at a time, so no issuer can see a
      reuse; an answer of `refresh_token_reused` is therefore a defect of
      the probe, a stop, never read as "ended". After `first-after`, A's
      newest refresh token is never sent again.
   * **Kilo** (a device token with no refresh and no expiry, MADR C22): one
      read of Kilo's profile with A's token (`auth.KiloProfile`), the
      authenticated request 0080-MADR made the check; it spends and rotates
      nothing.
6. **What it records.** An `http.RoundTripper` the test installs as every
   request's client (the sign-ins' `HTTPClient`, `UseHTTPClient` on each
   session, and `llmprovider.WithHTTPClient` on the profile read) records,
   per step, the HTTP status of the step's last response: the refresh, or
   the profile read, each the step's only request but for Grok's discovery
   document, which comes first. It never reads a request's body (a refresh
   request carries the refresh token). For a refused refresh only (status
   400 or more), it reads the response's OAuth `error` field and keeps it
   only when it is one of `invalid_grant`, `invalid_client`,
   `refresh_token_expired`, `refresh_token_reused`,
   `refresh_token_invalidated` (auth's own terminal codes,
   `oauthTerminalRefreshCodes`), writing `other` for any other value. It
   reads no successful body, and puts back every body it reads. Each step logs one
   line, `"%s %s: status %d, kind %s, code %s"`: the provider, the step,
   the status, the error's kind by `errors.Is` (`ok`, `auth`,
   `rate_limited`, `unavailable`, `invalid_request`, `canceled`, `other`)
   and the allow-listed code or `-`; the sign-in steps log `ok` or the kind
   only. Then one verdict line, `"%s: %s"`, the provider and `kept`, `ended`
   or `indeterminate`:
   * `kept`: `baseline`, `first-after` and `control` each succeed;
   * `ended`: `baseline` and `control` succeed, and `first-after` fails
     with kind `auth` (a 400 or 401 with a terminal code, or Kilo's 401);
   * `indeterminate`: anything else, including `refresh_token_reused`.

   It never logs a token, an authorization or user code, a URL, an account
   id, a client id, a host id, an error's text or a body.
7. **The redaction proof**, `TestEarlierGrantReport_0081` (the twin,
   offline): the checks run against an in-process issuer and Kilo stand-in
   (`httptest`) whose answers carry planted secret-shaped values: a
   JWT-shaped access token, a 40-character refresh token, an authorization
   code and a callback URL holding it, a Kilo token, an account id, a client
   id and a host id, each also echoed in a refused refresh's body and in the
   error the SDK returns. It drives each verdict (`kept`; `ended` with
   `invalid_grant`; `indeterminate` with `refresh_token_reused` and with a
   failed control) and asserts that the captured output holds each verdict
   and none of the planted values. For step 3 it also plants a user code
   and an authorize URL holding a state and a code: it asserts the code
   appears on the stand-in terminal writer, the URL reaches the stand-in
   opener, and neither appears in the captured test output; and, with a
   terminal writer that cannot be opened and an opener that fails, that the
   run stops with its plain line and the output holds neither value. Seen
   red by plants, with `tools/plant.py`, each quoted: the step line given
   the error's text; the recorder keeping a code outside the allow-list;
   the verdict computed without the control; the user code logged to the
   output instead of the terminal writer; the opener's failure falling back
   to logging the URL.
8. **Before the owner runs it,** the agent reads the three files and
   confirms in the execution record that no log or failure call is given a
   session, a token, a variable holding one, an error's text or a body;
   runs `go vet -tags live_gateways ./llmprovider/` and `go test -count=1
   -run 'EarlierGrant' ./llmprovider/` (session host, no `-race`); and
   commits the files with
   `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- make
   pre-add-check FILES="..."`.

Acceptance: the twin green, its plants caught; the live test compiles; no
token or session in any file.

#### Phase 1b: the owner's run

**Read this first.** If a provider does end earlier grants, the probe's
own two sign-ins end your other sessions with that provider, in every app
(the vendor's CLI, gobble, prepare-commit-msg), and those apps then ask you
to sign in again. That is the very thing being measured; the probe cannot
avoid it (MADR D12). The probe does not touch your stored files either way.

**Where to look.** Each sign-in opens in your browser by itself; nothing
is pasted. For Kilo, the code to enter on Kilo's page appears in the
terminal window as `Kilo code: ...`, written straight to the terminal; it
is not part of the test's output and is not sent back. If a run stops with
"the browser could not be opened" or "the terminal device could not be
opened", nothing was shown anywhere; tell the coordinator.

The owner runs it, never an agent, at the commit that adds the probe, in a
new terminal window that is closed afterwards. The commands hold no secret,
so the shell's history holds none; they work as written in zsh and bash.
Nothing else is typed in that window. The ChatGPT plan run registers this
module's app name at OpenAI as the existing plan live test does
(`planAgentName`), once per run.

1. Open the checkout and confirm the commit:

   ```sh
   cd <the go-llmprovider-sdk checkout>
   git log -1 --format=%h
   ```

2. The ChatGPT browser sign-in, twice: sign in in the browser window each
   time it opens, with the same account:

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=chatgpt go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

3. The ChatGPT plan sign-in, twice; the second may show OpenAI's consent
   again:

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=plan go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

4. Grok's browser sign-in, twice:

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=grok go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

5. Kilo's device sign-in, twice: Kilo's page opens in the browser; enter
   the code the terminal shows as `Kilo code: ...`, and approve, each
   time:

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=kilo go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

6. A run whose verdict is `indeterminate` is run once more, the same
   command; a second `indeterminate` is a stop.
7. Close the terminal window.

**What each run prints,** and what the owner sends back whole to the
coordinator: five step lines and one verdict, for example
`chatgpt sign-in-1: ok`, `chatgpt baseline: status 200, kind ok, code -`,
`chatgpt sign-in-2: ok`, `chatgpt first-after: status 200, kind ok, code
-`, `chatgpt control: status 200, kind ok, code -`, `chatgpt: kept`; then
the `--- PASS:` or `--- FAIL:` line and `ok` or `FAIL`. The `Kilo code:`
lines are written to the terminal device, not the output; they are not
sent. If any line of the
output shows anything that looks like a token, a code or a URL, the owner
sends nothing, says so, and the PLAN stops.

**What each outcome decides:**

| Verdict | Decides |
| :--- | :--- |
| `kept` for all four | the premise of 0059-MADR Q2 (a) and of saving last holds for those sign-ins; Phase 2 proceeds |
| `ended` for any one | a stop: the PLAN comes back to the owner before Phase 2 lands, with that provider's lines; saving last would leave an abandoned run with a dead stored session there |
| `indeterminate` twice | a stop: the probe or the service could not tell, with the lines |
| a provider not run | a stop: the PLAN names it and asks the owner whether to proceed without it |

#### Phase 1c: the record (an agent)

Model: lightweight, low.

1. The execution record quotes every line the owner sent, per provider.
2. The MADR's D12 gets a dated amendment line per provider with its
   verdict; "Not verified" loses N1 for the providers measured.
3. Commit the two records' edits (records only).

Acceptance: four verdicts recorded, each `kept`; or a stop, with the
owner's resolution recorded as a dated deviation before Phase 2.

### Phase 2: the tests and the code, one commit

Model: mainstream, high. *(Re-priced from the first draft's medium: the
change now restructures where `ConfigureLLM` saves, and the tests are
fifteen, across two entry points and every window.)*

Files: the two test files (new); `wizard/configure.go`; `wizard/auth.go`
(doc comments); `llmprovider/x/signin/signin.go` (a doc sentence);
`README.md`; `docs/architecture.md`; `CHANGELOG.md`; the dated notes; this
PLAN's execution record.

1. **Re-read the base** (Relation to 0080) and record the lines.
2. **The tests.** Start from the drafting's reproduction, renamed with the
   record's number. Each test lists through `testCache(t)` (0058-MADR) and
   turns model metadata off. "Drafting" quotes the failure seen at
   `b109cea`; each was also seen on a copy of 0080's staged index.

   | Test | Asserts | Red at the base and at `b109cea` |
   | :--- | :--- | :--- |
   | `TestCancelDuringListingSavesNothing_0081` | the context ends during the listing, for each session kind (Kilo device; Grok device and browser; OpenAI browser and device; a pasted ChatGPT access token; and, added in this phase, the ChatGPT plan, its consent scripted), through both entry points: the context's error, no save | drafting: twelve cases "saved 1 time(s); want none"; the three OpenAI `ConfigureLLM` cases also `err <nil>` |
   | `TestCancelAfterIssueSavesNothing_0081` | the context ends inside the login, after the session exists, with and without `Discover`: no save | drafting: both save once |
   | `TestCancelDuringKiloProfileSavesNothing_0081` | without `Discover`, the context ends during Kilo's profile read: no save | drafting: saves once |
   | `TestDeadlineDuringListingSavesNothing_0081` | the run's deadline passes during the listing: `context.DeadlineExceeded`, no save | drafting: saves once |
   | `TestCancelledConfigureReturnsNoKey_0081` | a third party's descriptor with no static catalog, a typed key, the run cancelled after the key: the context's error, no key, and no model id asked (N3) | drafting: "err <nil>, key returned true, model ids asked 1" |
   | `TestCancelAfterSaveReturnsTheResult_0081` | a store whose `Save` ends the run after writing: `ResolveCredential` returns the Result, one save | drafting: "err wizard: context canceled, [...] saves 1" |
   | `TestCancelAtModelStageSavesNothing_0081` | `ConfigureLLM`, a Grok device session, a cancel while the model search, the model menu, the model id or the fallbacks' menu is on screen, answered all the same: the context's error, no Result, no save | drafting: four cases `err <nil>, kind "oauth"` and "saved 1 time(s)" |
   | `TestCancelAtModelStageReturnsNoKey_0081` | a typed Grok key, a cancel while the model menu is on screen, answered all the same: the context's error and no key | drafting: "err <nil>, key returned true" |
   | `TestCancelAtManualModelSavesNothing_0081` | a ChatGPT session whose listing fails with a 503: a cancel at the model-id question saves nothing | drafting: "err <nil>, kind "oauth", saves 1" |
   | `TestConfigureSavesOnceAfterTheModel_0081` | a run that ends well saves once, and the store is empty while the model is asked | drafting: "saves 1, saves when the model was asked 1" |
   | `TestModelStageErrorSavesNothing_0081` | a Prompter's error at the model search: that error, no save (N2) | drafting: "saved 1 time(s); want none" |
   | `TestRotationDuringQuestionsSavedAtEnd_0081` | a refresh the listing left running is adopted during the questions and saved at the end, rotated, once; the store empty while the model is asked | drafting: "saves when the model was asked 1" (the rotated pair itself is saved today too) |
   | `TestCancelDuringListingReturnsNoKey_0081` | a typed Grok key, the listing cancelled, both entry points: the context's error and no key | green at `b109cea` (a guard) |
   | `TestCancelDuringRefreshSavesNothing_0081` | a refresh in flight when the run ends (K5): no save | green at `b109cea` (a guard) |
   | `TestResolveSavesInTheStep_0081` | `ResolveCredential` saves once and asks no model | green at `b109cea` (a guard) |
   | `TestCancelDuringDeviceFlowSavesNothing_0081` (new in this phase) | Grok's device sign-in through the real `auth` device flow, its issuer answered in process by the run's `HTTPClient`, the context ended during the listing: no save | to be seen red at the base; if the flow cannot be served in process without changing non-test code, the test is dropped and the execution record says why |

3. **Red.** `python3 -B tools/redgreen.py RED.json --expect fail --rev
   <base>` and `--rev b109cea`, for every test marked red, with the test
   files in `files`; the guards with `--expect pass`. Quote each failure.
4. **The code** (`wizard/configure.go`), as the drafting's prototype did
   (MADR R5):
   * the body of `credentialStep` becomes an unexported step returning
     `(Result, catalog.Catalog, resolvedCredential, error)`, the
     credential unsaved; its doc comment moves with it;
   * `commitCredential(ctx, d, c)`: `if err := ctx.Err(); err != nil {
     return fmt.Errorf("wizard: %w", err) }`, then `c.save(ctx, d)`;
   * `credentialStep` keeps its signature: the unsaved step, then
     `commitCredential` (N5 (a), the coordinator's application);
   * `ResolveCredential`: its own context check after the step, and that
     check's comment, removed;
   * `ConfigureLLM`: the unsaved step; then (N3) `if err :=
     ctx.Err(); err != nil { return Result{}, fmt.Errorf("wizard: %w",
     err) }`; then `commitCredential` as the last thing before each of the
     two `return res, nil` (the model-id branch and the end). Every error
     return after the step saves nothing (N2).
5. **Docs**, in the same commit:
   * doc comments: `credentialStep` (saves right after its check, for
     `ResolveCredential`); `ConfigureLLM` (saves a new session as its last
     step; any error, a context's end included, saves nothing; a sign-in
     abandoned at a model or fallback question is not kept; nothing issued
     is revoked); `ResolveCredential` (an ended context gives its error and
     saves nothing; its Result means saved); `Options.TokenStore` (written
     only when a run returns a Result); `resolvedCredential.save` and
     `saveTo`; `signin.SignIn.Wait` (a caller saves only while its own
     context has not ended, as the wizard does);
   * `README.md`: the wizard's sign-in bullet ("`ConfigureLLM` saves the
     session as its last step, `ResolveCredential` once the listing has
     not refused it; a refused or cancelled sign-in, or an abandoned
     `ConfigureLLM`, leaves the store as it was");
   * `docs/architecture.md`: "When a session is saved", and the wizard's
     paragraph on the save, with what stays open (the write; a process
     killed during the questions);
   * `CHANGELOG.md` `[Unreleased]` (N4): **Fixed** "The wizard saved a new sign-in when
     the run's context ended before the save, during the model listing or
     after the sign-in, and returned the context's error, or, at a model
     question, a Result; it now saves nothing and returns the context's
     error. `ResolveCredential` returns its Result when the context ends
     after the session is saved (0081-MADR)." **Changed** "`ConfigureLLM`
     saves a new session as its last step, after the model and fallback
     questions: a run abandoned there, or ended by any error, keeps the
     session the store held (0081-MADR)";
   * the dated one-line notes the MADR's D11 lists.
6. **Before staging,** on a scratch clone of the phase's tree: `go test
   -count=1 ./wizard/ ./llmprovider/x/signin/` (session host, no `-race`);
   every existing test passes unchanged, or the phase stops.

Verification: the tests green;
`~/.agents/skills/available-dev-hosts/scripts/run-heavy -- make
pre-add-check FILES="wizard/configure.go wizard/auth.go
wizard/cancel_save_0081_test.go wizard/save_at_end_0081_test.go
llmprovider/x/signin/signin.go"`; `make api-check`; `python3 -B
tools/apidiff.py v1.31.0`; `make docs-check tables-check changelog-check
links-check records-check markdownlint`.

Acceptance: every test as its row says; no existing test changed; no API
change; the docs say what the code does, including what stays open.

### Phase 3: planted breaks and the gates

Model: mainstream, low.

1. `python3 -B tools/plant.py PLANTS.json` on a scratch clone, each plant
   caught, each failure quoted. The seven the drafting's prototype was
   seen to catch:
   * the final check removed (`TestCancelAtModelStageSavesNothing_0081`,
     `TestCancelDuringListingSavesNothing_0081`);
   * the check for a session only
     (`TestCancelAtModelStageReturnsNoKey_0081`,
     `TestCancelDuringListingReturnsNoKey_0081`);
   * a cancel matched, not a deadline
     (`TestDeadlineDuringListingSavesNothing_0081`);
   * `ConfigureLLM` saving in the step again
     (`TestConfigureSavesOnceAfterTheModel_0081`,
     `TestModelStageErrorSavesNothing_0081`);
   * the model-id branch not committed
     (`TestCancelAtManualModelSavesNothing_0081`);
   * `ResolveCredential`'s own last check put back
     (`TestCancelAfterSaveReturnsTheResult_0081`);
   * the check after the step removed (N3)
     (`TestCancelledConfigureReturnsNoKey_0081`'s "no model id asked").
2. `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` and
   `--expect pass` on the tree, quoted.
3. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 ./wizard/... ./llmprovider/x/signin/... ./internal/signinflow/... ./llmprovider/auth/...`
4. `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
5. `make lint` and `make vuln`; `go vet -tags live_gateways ./...`.
6. CI's set: `make parity-check dep-check coverage-check api-check
   contract-check generate-check records-check tables-check fixtures-check
   changelog-check docs-check markdownlint links-check disclosure-check
   gate-selftest`, and `make tools-test`.
7. `python3 -B tools/claims.py` on the MADR and this PLAN;
   `python3 -B tools/denyscan.py --changed`; `python3 -B
   tools/recordlint.py --changed`.

Acceptance: every plant caught; every command exit 0, each status taken
before any filter. Docs: none beyond Phase 2's; a gate that finds a docs gap
is a deviation.

### Phase 4: the release (the owner's word)

Model: mainstream, medium. No step runs under a standing approval: the tag
and the GitHub Release each wait for the owner's word in the same turn.

1. 0080-PLAN Phase 5 cuts the release that carries this record (Q7's
   answer); this PLAN adds nothing but its changelog entries, already under
   `[Unreleased]`.
2. `python3 -B tools/consumer.py` and `python3 -B tools/offline.py`, quoted.
3. `tools/records.py status` sets this PLAN `complete` once the tag's CI
   and step 2 are recorded.

Acceptance: the tag's CI passes; the section and the Release name this
change.

### Phase 5: the consumers, after the release (their own records)

Not executed under this PLAN; recorded so the hand-off is exact.

* **gobble-cli** (at `cecc892`, branch `p3-acpext-v0.5.0`): the SDK pin
  from `v1.31.0` to the release, with 0080's bump. Its terminal path
  (`llm/provider/signin.go`, `SignIn`) uses `ResolveCredential`: nothing to
  change; after a cancel its `auth.json` is no longer written, and a cancel
  after the save returns a Result. Its comments that say the wizard saves
  nothing after a done context (MADR G1 to G3) are now true. ACP path:
  unchanged.
* **prepare-commit-msg** (at `2715f2d`, pinned at `v1.29.0`): a pin bump
  brings, with every release since, `ConfigureLLM` saving last; its setup
  (`internal/ui/setup.go`) needs no code change, and an abandoned or killed
  setup no longer replaces the stored session. Its record names that a
  killed setup loses a completed sign-in.

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy` or `run-on-host`.

* Phase 1: the probe's twin and plants; the owner's four runs, quoted.
* `go test -count=1 ./wizard/ ./llmprovider/x/signin/`
* `python3 -B tools/redgreen.py RED.json --expect fail --rev <base>` (and
  `--rev b109cea`), then `--expect pass`; `python3 -B tools/plant.py
  PLANTS.json`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- go test -race -count=1 ./wizard/... ./llmprovider/x/signin/... ./internal/signinflow/... ./llmprovider/auth/...`
* `make api-check` and `python3 -B tools/apidiff.py v1.31.0`
* `make lint`, `make vuln`, `go vet -tags live_gateways ./...`
* `make parity-check dep-check coverage-check api-check contract-check
  generate-check records-check tables-check fixtures-check changelog-check
  docs-check markdownlint links-check disclosure-check gate-selftest` and
  `make tools-test`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN

## Rollout and Rollback

* **Order.** Phase 0 when placed; Phase 1 (1a, the owner's 1b, 1c), and
  nothing after it until its verdicts are recorded; Phase 2 on the base; Phase 3 after it;
  Phase 4 with 0080's release, on the owner's word; Phase 5 in each
  consumer after the tag.
* **Rollout.** A caller whose run ends well sees nothing new, except that
  `ConfigureLLM` writes the store at its end. One whose run ends with an
  error finds the store as it was. `ConfigureLLM` returns an error where it
  returned a Result after a cancel (K4, K10). `ResolveCredential` returns a
  Result where a cancel landed after its save.
* **Rollback before the tag:** revert Phase 2's commit; the records stay,
  with a dated note.
* **Rollback after the tag:** a patch release that moves `ConfigureLLM`'s
  save back into the step (P1), or removes the check, by a dated amendment
  of the MADR. Nothing persistent changes shape, so no stored session needs
  migrating either way.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, at `b109cea`;
Phase 2 re-reads each at the base.

<!-- claims: self=b109cea -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The README's sign-in bullet | `README.md:388` | `` - A sign-in saves its session to `Options.TokenStore`, once the model `` |
| P2 | The architecture's save paragraph | `docs/architecture.md:391` | `` sign-in. A sign-in's session is saved to `Options.TokenStore` once the `` |
| P3 | and its "When a session is saved" | `docs/architecture.md:1038` | `- **When a session is saved.** A new session is saved once the model` |
| P4 | The changelog's open section | `CHANGELOG.md:12` | `## [Unreleased]` |
| P5 | `Wait`'s doc | `llmprovider/x/signin/signin.go:251` | `// Wait returns the credential the sign-in yields, or its error, which` |
| P6 | `ResolveCredential`'s last check's comment | `wizard/configure.go:376` | `// A listing ctx ended is not a refusal: it falls back to the static` |
| P7 | `credentialStep`'s doc | `wizard/configure.go:427` | `// credentialStep resolves d's endpoint and credential, then lists its models,` |
| P8 | The test seams: a fake transport | `wizard/auth_test.go:349` | `type roundTripFunc func(*http.Request) (*http.Response, error)` |
| P9 | a session | `wizard/auth_test.go:240` | `func testOAuthSession(provider llmprovider.ProviderID) *auth.OAuthSession {` |
| P10 | a JSON answer | `wizard/session_0059_test.go:67` | `func answer0059(r *http.Request, status int, body string) *http.Response {` |
| P11 | a third party with no static catalog | `wizard/describe_0056_test.go:94` | `func registry0056(t *testing.T, static []string, err error) *llmprovider.Registry {` |
| P12 | the provider menu's index | `wizard/configure_test.go:21` | `func providerIdx(t *testing.T, id llmprovider.ProviderID) int {` |
| P13 | 0059's W2 | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:88` | `* **W2. A fresh sign-in replaces the saved session before the listing` |
| P14 | and its amendment | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:490` | `## Amendment 2026-10-24: a refresh can outlast the listing (Phase 2)` |
| P15 | 0056's D1 | `docs/decisions/0056-MADR-wizard-credential-only-entry-point.md:136` | `` ### D1. `wizard.ResolveCredential` `` |
| P16 | `saveTo`'s comment | `wizard/auth.go:55` | `// saveTo is the store a new session is due to: credentialStep saves it` |
| P17 | `Options.TokenStore`'s doc | `wizard/configure.go:117` | `// TokenStore persists sessions created by the browser, device-code and` |
| P18 | 0059's W8 | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:143` | `` * **W8. `ConfigureLLM` asks questions on an ended context** **[probed]** `` |
| P19 | A record's own live variable | `AGENTS.md:266` | `` switched on by `LLMPROVIDER_LIVE_<NNNN>`, with suffixes for its options, as `` |
| P20 | The plan's live sign-in registers this module's name | `llmprovider/live_chatgpt_plan_test.go:17` | `const planAgentName = "go-llmprovider-sdk"` |
| P21 | auth's terminal refresh codes, the probe's allow-list | `llmprovider/auth/oauth_session.go:62` | `"invalid_grant", "invalid_client",` |
| P22 | A session can be made due, to force one refresh | `llmprovider/auth/oauth_session.go:484` | `func (s *OAuthSession) Invalidate() {` |
| P23 | The plan sign-in takes the earlier session as `Previous` | `llmprovider/auth/oauth_plan.go:61` | `Previous *OAuthSession` |
| P24 | Kilo's profile read, the probe's use of a Kilo session | `llmprovider/auth/kilo_device.go:179` | `func KiloProfile(ctx context.Context, token string, opts ...llmprovider.Option) (KiloAccount, error) {` |
| P25 | The checks a live test shares with its offline twin | `llmprovider/live_checks_test.go:3` | `// The checks the person-gated live tests run after their sign-in, untagged so` |

## Execution record

### The owner's answers (2026-11-17)

* Relayed by the coordinator; the labels are in the MADR's "Decided": N1
  ("A live probe by me before release (Recommended)"), N2, N3 and N4, each
  as the draft recommended. N5 was not put to the owner; the coordinator
  applied option (a). Phase 1 was then written out in full (the probe, the
  owner's commands, the outcomes), N6 raised by it and then applied by the
  coordinator as option (a), and the steps that named
  the options not chosen were cut to the chosen ones.

### Drafting (2026-11-17)

* **Provenance.** Drafted by a strongest-tier model at high effort, on the
  coordinator's brief, from read-only investigation; nothing implemented,
  nothing written to any repository's tree or index. The MADR and this
  PLAN are `proposed`, with `0081` for the number, which the placing step
  takes from `scripts/check_records.py --next`.
* **The ruling,** the owner's of 2026-11-17, relayed by the coordinator:
  "A record of its own, drafted now". What it settles: the MADR's "The
  owner's ruling of 2026-11-17".
* **Verified, and how.**
  * Read at `b109cea` (`git archive` into a scratch folder): the wizard's
    sign-in, check, save and model stage; `auth`'s `Token`, refresh,
    `FileTokenStore` and `ClientStore`; `catalog`'s search;
    `internal/signinflow`; `x/signin`; the records 0016, 0026, 0056, 0059,
    0074, 0075 and 0080; the API guide's R48 and R53; `CHANGELOG.md`'s
    `v1.22.1` entry for 0059's W2. Read on the index of the tree as it
    stood (`git show :<path>`): 0080's staged wizard files and tests, and
    0080-PLAN's execution record. Read at gobble-cli `cecc892`: its
    sign-in, store and lock code and its 0051 records. Read at
    prepare-commit-msg `2715f2d`: `internal/ui/setup.go`,
    `internal/config/config.go`, `go.mod`. Read at the codex checkout
    `57e64682f8`: `codex-rs/login/src/auth/manager.rs`, where tokens are
    revoked.
  * Re-run, on scratch copies only, the fifteen tests of the reproduction:
    at `b109cea` twelve fail, each as Phase 2's table quotes, and three
    guards pass; on a copy of 0080's staged index over `b109cea` (the index
    as it stood later on 2026-11-17), the same.
  * Prototypes, on scratch copies only. The first draft's (a check before
    the save in the step) is superseded by Q4's answer. The Q4 (c)
    prototype (MADR R5): all fifteen pass at `b109cea` and on 0080's
    staged copy; `go test -count=1 ./wizard/` passes whole on each ("ok",
    8.5 s and 9.5 s, no `--- FAIL`), with no existing test changed; seven
    planted breaks, each caught (Phase 3's list), with
    failure lines such as "save_at_end_0081_test.go:185: saves 1, saves
    when the model was asked 1; want one, after the model". A first try
    that changed `credentialStep`'s signature failed to compile 0080's
    `TestKiloVerify_EndedContext_0080`, which calls it ("assignment
    mismatch: 3 variables but credentialStep returns 4 values"); hence
    N5 (a), which the coordinator then applied. `gofmt -l` is clean on the test files and `go vet ./wizard/`
    exits 0 on the prototype.
  * The line citations of the MADR's evidence index (46 rows, four
    repositories) and this PLAN's code claims were taken from the cited
    revisions and checked with `tools/claims.py` from a scratch copy, all
    passing; planted false rows were each caught.
  * Placement rehearsed on a scratch copy at `b109cea`: `tools/records.py
    add`, the records check, the link check, the disclosure check and
    `tools/recordlint.py` each exit 0 (the drafting's `PLACEMENT.md`).
* **Not verified.** N1, until the owner's Phase 1b; the ChatGPT plan
  session's cases; a sign-in through
  the real `auth` flows with an in-process issuer; K1 and K7, and a process
  killed during the questions, read only; any `-race` run (the session
  host's heavy queue held another agent's runs); any gate; any host but the
  session's.
