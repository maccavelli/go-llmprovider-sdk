---
status: in-progress
date: 2026-11-20
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

> **Annotation of 2026-11-18.** The first item above concerns the wizard's
> own sessions (MADR D6: a session it did not save is not revoked) and stands.
> The probe of Phase 1a signs in twice at one provider and keeps neither
> session, which is a different thing, and the owner ruled on it: "Revoke the
> probe's own two grants at the end, after the verdict is printed
> (Recommended)" (the label, relayed by the coordinator). Phase 1a's
> annotations of that date say what it builds, and Phase 1b's amendment lists,
> per provider, what is revoked by the run, what is left and where the owner
> removes it by hand.

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

**Dated note, 2026-11-20.** The owner's ruling of that date on live runs,
relayed by the coordinator, "API keys from the environment only
(Recommended)", means a future live step that needs only environment API
keys may be run by an agent; this record's probe, which signs in through
the browser and by device code, was and remains the owner's, and nothing
in 0081 is re-run.

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

> **Dated note, 2026-11-19.** The steps below are amended by the annotations
> that follow this phase's "Acceptance" paragraph, and those win where they
> differ: step 3's output goes to a file, not the window; step 4's sequence has
> a sixth step, `last-after`; step 5's "each refresh token is sent exactly once"
> is not guaranteed; step 6's verdicts are the table in Phase 1b's amendment; and
> there is a step 9, the revocation.

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

> **Annotations of 2026-11-18 to Phase 1a** (the independent review's
> findings F3, F4, F7, F9, F11 to F13). Additions only: the steps above stand
> except where an annotation says otherwise.
>
> * **Step 2.** "reads no environment variable but its two" is read as: the
>   test calls `os.Getenv` for no variable but its two. The HTTP transport
>   reads the proxy variables, and the browser opener inherits the process's
>   environment.
> * **Step 3.** The browser's page is passed to the opener as a process
>   argument and is kept in the browser's history, and the plan's second
>   sign-in's URL carries the first session's ID token as `id_token_hint`
>   (Phase 1b's amendment says so to the owner). The opener's failure is
>   made a stop by the opener's own side effect, not by the error it returns:
>   `runBrowserLogin` calls `OpenURL` from a goroutine and ignores its error
>   (`llmprovider/auth/oauth_loopback.go:220-222`), so `grantShow.URL` records the
>   stop and cancels the sign-in's context itself. The twin shows it with a
>   stand-in that ignores the error and with the shipped `LoginBrowserOAuth`.
>   The probe's output goes to a file, so that the terminal shows only
>   Kilo's user code (Phase 1b's amendment).
> * **Step 5.** "Grok's discovery document, which comes first" is wrong for a
>   refresh: a refresh makes no discovery request. The session's token URL is
>   set at sign-in (`llmprovider/auth/oauth_loopback.go:859`), or derived
>   from the issuer when it is OpenAI's or xAI's own
>   (`llmprovider/auth/oauth_session.go:1273-1275`). The refresh's own
>   requests are all a use step makes.
> * **Step 6.** The recorder reads the body of every response of 400 or
>   more, not only a refused refresh's; it keeps only an allow-listed code,
>   and puts every body back (a reading taken). It reads the code in the three
>   shapes the SDK's own `refreshErrorCode` reads, lower-cased
>   (`llmprovider/auth/oauth_session.go:1010-1027`): `{"error": "code"}`,
>   `{"error": {"code": "code"}}` and `{"code": "code"}`, and reads the first 2048
>   bytes of the body for it, as the SDK does (`oauthErrorBodyLimit`, `:32`),
>   putting the whole body back. A use step's line now
>   ends `, requests N`, the number of round trips the step completed:
>   `"%s %s: status %d, kind %s, code %s, requests %d"`. The sign-in steps'
>   lines and the verdict line are as written.
> * **Step 4, an edge case.** A use whose one-minute bound ends while its
>   refresh is still running leaves the refresh running, detached from the
>   caller (`llmprovider/auth/oauth_session.go:229`, `complete`). The
>   plan's second sign-in reads the first session's fields
>   (`Previous`) without the session's lock, and a late reply writes them and
>   the next step's status. The refresh's own bound is three attempts of
>   fifteen seconds with a short backoff, under the use's minute, so the case
>   is not expected; if it happened, the use's step line would read
>   `canceled`, and the late round trip would be counted in the next step's
>   `requests`, which is how the count shows it.
>
> **Annotations of 2026-11-18 to Phase 1a, the owner's rulings** (relayed by
> the coordinator; each label is quoted as a label only). Additions only.
>
> * **Which verdicts pass.** "Only `kept` passes (Recommended)". As built:
>   `grantPasses`, in one line; `ended` and `indeterminate` fail the test, so
>   that `go test` says so.
> * **Step 4, one step more.** "Add one step: refresh A once more after the
>   control (Recommended)". The sequence is now sign-in-1, baseline, sign-in-2,
>   first-after, control, last-after. The baseline's rotated refresh token
>   carries through all of A's refreshes: the session adopts each rotation, and
>   no store is bound, so each token is sent as the session holds it. `kept` now
>   needs both of A's later refreshes to succeed; an ending that comes minutes
>   after the run is still not caught, and the verdict says so. The verdict for
>   every combination of the two later refreshes and the control is the table
>   in Phase 1b's amendment, written before the code; `grantVerdict` is its
>   definition. The readings it needed, each from this PLAN's own rule that no
>   refresh token is sent twice (step 5): `last-after` is made only when
>   `first-after` succeeded, because after a refusal A still holds the same
>   token, and sending it again adds nothing; and a failed `baseline` ends the
>   run at once, `indeterminate`, before the second sign-in (the earlier
>   reading, to go on to the second sign-in and `first-after`, would have sent
>   the baseline's token again). For Kilo, which does not refresh, each use of A
>   is a read of its profile.
> * **Step 6, the count.** "The run is `indeterminate`: run it once more
>   (Recommended)". `grantVerdict` returns `indeterminate` when any of the four
>   uses made more than one request, whatever the status, before it reads
>   anything else; the count stays in the step line. Revocation requests are
>   kept out of the count: revocation runs after the verdict is computed and
>   printed, each revocation begins from a count of zero (`grantRecorder.begin`),
>   and its line carries no count.
> * **Step 5, "each refresh token is sent exactly once".** Not guaranteed: the
>   SDK sends a refresh again, with the same token, on a transport error, a 429
>   or a 5xx. The guard is the count rule (Step 6, the count), and for `grok`,
>   whose issuer answers a reused token as plain `invalid_grant`, it is the only
>   one (Phase 1b's amendment).
> * **Step 6, the verdicts and the lines.** The vocabulary stands, but its
>   definitions are the table in Phase 1b's amendment, which has the sixth step
>   and the count rule; a use step's line ends `, requests N`; and each grant
>   obtained has a revocation line after the verdict line.
> * **Step 9, new: revocation.** "Revoke the probe's own two grants at the end,
>   after the verdict is printed (Recommended)". After the verdict line (or,
>   when the run was stopped, in its place), each session a sign-in returned is
>   revoked, in the order obtained, by the SDK's `auth.RevokeOAuthSession`,
>   through the session's own client, which is the recorder's
>   (`llmprovider/auth/oauth_revoke.go:53`, `:79`), on a context detached from
>   the run's (`context.WithoutCancel`) and bound at twenty seconds each
>   (`grantRevokeLimit`), so that the run's own bound having passed does not
>   stop it. It runs on every path where a grant exists: after a verdict, after
>   an `indeterminate` that follows the first sign-in (a failed second sign-in
>   or a failed baseline), and after a stop, and not when no sign-in gave a
>   session. Its line is `"%s revoke-%d: status %d, kind %s"`: the provider,
>   which grant (1, the first session; 2, the second), the status of the
>   revocation's last response (0 when it made none) and the error's kind
>   (`ok`, `unsupported`, `canceled`, `other`), through the same redaction, and
>   nothing from a body, a code or an error's text. A failed revocation never
>   changes the verdict and never fails the test by itself. What each provider
>   revokes, what is left and where to remove it by hand, and what is not
>   verified, is Phase 1b's amendment. No non-test code was added or changed.
> * **Step 9, `chatgpt` and `grok` (rulings of 2026-11-19).** "Run `chatgpt`
>   with revocation switched off; its two grants are listed for removal by hand
>   (Recommended)", and "Switch `grok`'s revocation off too; its two grants are
>   listed for removal by hand (Recommended)". For `chatgpt` and `grok` no
>   revocation request, and no discovery request for one, is made on any path (the
>   verdict, the stop, the `indeterminate` and the bound-passed paths), by
>   `grantRevokes`, a function of the provider in the checks file that is false for
>   those two only; only `plan` is revoked by the run. Their revocation lines
>   read `<provider> revoke-N: status 0, kind by_hand`: a
>   fixed word, which is not `ok`, not Kilo's `unsupported` and not a failure,
>   and is not counted as one. The measurement is unchanged. The reason is
>   Phase 1b's "Not verified": nothing read says what OpenAI's revocation ends
>   besides the token it names, and the sign-in uses the client id Codex itself
>   uses. `plan`, `grok` and `kilo` are as written above. The twin shows zero
>   requests to a revocation endpoint on the three paths
>   (`TestEarlierGrantRevocation_0081/chatgpt's_grants_are_not_revoked,_*`).
> * **Step 4, a baseline of more than one request (ruling of 2026-11-19).**
>   "Stop there: end the run as `indeterminate` before the second sign-in
>   (Recommended)". After the baseline, if its request count is above one, the
>   run ends `indeterminate` before the second sign-in, on the path a failed
>   baseline takes (`earlierGrant.run`): the first grant is revoked where the
>   provider's revocation is on, a `by_hand` line otherwise, and the baseline's
>   step line is printed with its count. The row is in the verdict table of
>   Phase 1b's amendment and in `grantVerdict`'s doc, and the twin runs it end to
>   end (`TestEarlierGrantReport_0081/*/a_baseline_that_needed_two_requests_*`).
> * **Step 4, the two readings ruled (2026-11-19).** "As built: never resend a
>   refused token; the last step runs only after a success (Recommended)", and
>   "As built now: a failed baseline ends the run as `indeterminate`
>   (Recommended)". The two readings recorded above are the owner's rulings; no
>   code changed for them.
> * **The bounds, re-derived.** Two sign-ins at their four minutes use eight
>   minutes of the run's nine; the four uses take seconds each and each is
>   bound at one minute; the revocations add at most forty seconds on their own
>   bound: nine minutes and forty seconds at the most, which `go test`'s
>   default ten minutes leaves twenty seconds, so the four commands now carry
>   `-timeout 15m`. Phase 1b says how long the owner has.
> * **Readings the owner was told of and did not change:** the four-minute
>   sign-in wait and the nine-minute run bound, and `xdg-open` on Linux beyond
>   step 3's "open on macOS".

#### Phase 1b: the owner's run

> **Amendment of 2026-11-18, as amended on 2026-11-19** (the independent
> review's findings, and the owner's rulings of both days, relayed by the
> coordinator). Additions only: the text after this block stands, except where
> this says otherwise, and the commands here replace steps 2 to 5's. **Read this
> block in order.** The first section is what a run does to your real accounts
> and what is not known about it; the commands come last on purpose, and none is
> to be run before that section has been read.
>
> **Before any command: what a run does to your accounts, and what is not
> known.** The probe signs in twice at one provider and, at its end, revokes
> the grants it obtained, with the SDK's own `auth.RevokeOAuthSession`, for
> `plan` only: the owner ruled that `chatgpt`'s and `grok`'s are left, and Kilo
> has none. That call sends the session's own refresh
> token (its access token if it has no refresh token) and its client id to its
> own issuer's revocation endpoint, through the client the probe's recorder
> wraps, and nothing to any other host. What it can do to a real account, and
> what nobody has measured:
>
> | Provider | What the run does to the real account | What is not known |
> | :--- | :--- | :--- |
> | `chatgpt` | Signs in twice with the client id Codex itself uses. Makes **no revocation request, on any path**, by the owner's ruling "Run `chatgpt` with revocation switched off; its two grants are listed for removal by hand (Recommended)". Two grants stay live on the OpenAI account until they expire or are removed by hand. | Whether OpenAI's revocation, if it were sent, would end only the token it names or also the owner's own Codex login, which uses the same client id. That is why it is not sent. |
> | `plan` | Signs in twice under the client the first sign-in registers for this app's name, `go-llmprovider-sdk`, and revokes the two refresh tokens of **that client only**, at the revocation endpoint of OpenAI's discovery document, with that client's id (`llmprovider/auth/oauth_revoke.go:144`). It touches no other client's token. The registered client itself stays (below). | Whether OpenAI's revocation of a token ends anything besides that token. |
> | `grok` | Signs in twice with the Grok CLI's own public client id (`DefaultGrokOAuthClientID`, `llmprovider/auth/oauth_constants.go:13`, set on the shipped sign-in at `oauth_loopback.go:296-301`). Makes **no revocation request, on any path, and no discovery request for one**, by the owner's ruling "Switch `grok`'s revocation off too; its two grants are listed for removal by hand (Recommended)". The SDK's call would send that same client id with the token (`oauth_revoke.go:137`, the form at `:159`), which is the condition that led the owner to switch `chatgpt`'s off. Two grants stay live on the xAI account until they expire or are removed by hand. | Whether xAI's revocation, if it were sent, would end only the token it names or also the owner's other Grok sessions that use the same client id, the Grok CLI's included. That is why it is not sent. |
> | `kilo` | Signs in twice by device code and revokes nothing: the SDK has no revocation for Kilo's device token (`oauth_revoke.go:72`; `llmprovider/x/signin/signin.go:158`), so the run sends no request. Two device tokens, which do not expire, stay on the Kilo account until removed by hand. | Nothing about revocation: none is sent. |
>
> The doubt in the last column cannot be settled from the SDK's code or the
> upstream sources read: the Codex checkout shows that logout revokes its
> refresh token, "best effort", and says nothing of what else the issuer then
> ends, and the Grok CLI's source (`xai-grok-login/src/error.rs:77`) says only
> that its issuer does not distinguish an expired, reused or revoked refresh
> token. Nothing here measured whether a revocation ends other sessions. That
> is why the run sends no revocation for `chatgpt` or `grok`, and why it sends
> `plan`'s only for the client the probe registered itself, with that client's
> tokens.
>
> One more fact about `grok`, so that it is no surprise in its file: xAI answers
> a reused refresh token as plain `invalid_grant`, which the probe reads as a
> dead grant, never as `refresh_token_reused`. For `grok`, a token sent twice
> would therefore read as `ended`; the only guard against that is the count rule
> (a use step that made more than one request makes the run `indeterminate`), so
> a `grok` `ended` is to be read with the `requests` of its lines in view.
>
> **What a run leaves behind, and what it removes.** By the owner's ruling
> "Revoke the probe's own two grants at the end, after the verdict is printed
> (Recommended)", the run revokes what it obtained, except `chatgpt`'s, `grok`'s
> and Kilo's (above): only `plan`'s are revoked. Per provider:
>
> | Provider | Revoked by the run | Left | Where the owner removes it by hand |
> | :--- | :--- | :--- | :--- |
> | `chatgpt` | nothing: no revocation request is made on any path (the SDK's call would send a JSON request to OpenAI's `/oauth/revoke` with the Codex client id, `llmprovider/auth/oauth_revoke.go:92`) | **two grants under Codex's client id**, live on the owner's OpenAI account until they expire or are removed | the owner's OpenAI account, where the apps and sessions signed in to it are managed (not verified: no page was looked up); the owner's own Codex login is not one of them, and is not to be removed |
> | `plan` | both refresh tokens, by a form request to the revocation endpoint of OpenAI's discovery document with the issued client id (`oauth_revoke.go:144`) | the client the first sign-in registered under the owner's OpenAI account for this app's name (`go-llmprovider-sdk`): no call of the SDK removes a registered client | the owner's OpenAI account, where the apps and agents signed in to it are managed (not verified: no page was looked up) |
> | `grok` | nothing: no revocation request, and no discovery request for one, is made on any path (the SDK's call would discover xAI's revocation endpoint and send a form request with the Grok CLI's client id, `oauth_revoke.go:123`) | **two grants under the Grok CLI's client id**, live on the owner's xAI account until they expire or are removed | the owner's xAI account, where the apps and sessions signed in to it are managed (not verified: no page was looked up); the owner's own Grok CLI login is not one of them, and is not to be removed |
> | `kilo` | nothing: no request is sent | both device tokens, which do not expire | the owner's Kilo account, where its signed-in devices or tokens are managed (not verified: no page was looked up) |
>
> A revocation line reads `<provider> revoke-N: status S, kind K`: `S` is the
> status of the revocation's last response (0 when it made none), and `K` is
> `ok` (the provider answered 200), `unsupported` (Kilo: nothing was sent),
> `by_hand` (`chatgpt` and `grok`: the run does not revoke them, by the owner's
> rulings, nothing was sent), `canceled` (its twenty seconds ended), or `other` (any other answer; its status says which). Only
> `status 200, kind ok` means the grant is revoked, and `unsupported` and
> `by_hand` mean the owner removes it by hand. A line that says anything else
> means the grant is still live: remove it by hand, in the place above. A
> revocation never changes the verdict, never fails the test and adds nothing
> to any count: it runs after the verdict is printed, each revocation starts
> from a count of zero, and its line carries no count. Revocation runs on every
> path where a sign-in gave a session: after a verdict, after an
> `indeterminate` that follows the first sign-in, and after a stop (the line
> that stops the run, "the browser could not be opened" or "the terminal
> device could not be opened", follows the revocation lines), and on a bound of
> its own, so that the run's bound having passed does not stop it; for
> `chatgpt` and `grok` no request is made on any of those paths. A re-run after
> an `indeterminate` obtains two more grants, each revoked in the same way (for
> `chatgpt` and `grok`, left in the same way: four grants then stand), and, for
> `plan`,
> registers one more client under the owner's account (the run's two sign-ins
> share the one it registers).
>
> **The owner's rulings of 2026-11-19** (relayed by the coordinator; each label
> is quoted as a label only), recorded here and in Phase 1a's annotations: "Run
> `chatgpt` with revocation switched off; its two grants are listed for removal
> by hand (Recommended)"; "As built: never resend a refused token; the last step
> runs only after a success (Recommended)" (`last-after` is made only after a
> successful `first-after`); "As built now: a failed baseline ends the run as
> `indeterminate` (Recommended)"; "Switch `grok`'s revocation off too; its two
> grants are listed for removal by hand (Recommended)"; "Stop there: end the
> run as `indeterminate` before the second sign-in (Recommended)" (a baseline
> that succeeded but needed more than one request).
>
> **The time the owner has.** Each sign-in must be finished, the browser's
> page approved or Kilo's code entered, within four minutes of its page
> opening. The run's sign-ins and uses are bound at nine minutes in all: two
> sign-ins of four minutes use eight, and the four uses take seconds (the
> SDK's refresh gives up after three attempts of fifteen seconds, under the
> minute each use is allowed), so a person who takes the whole four minutes
> twice is not cut off. The revocations that follow have twenty seconds each, on
> their own bound, so the whole test is at most nine minutes and forty
> seconds, and `-timeout 15m` stands over it. A sign-in left longer than four
> minutes is cancelled: its step line reads `sign-in-N: canceled`, the run
> ends with the verdict `indeterminate`, and the re-run of step 6 is two new
> sign-ins. Have the account signed in to the browser and the person ready
> before each command.
>
> **What the browser is given.** Each authorize URL, and Kilo's verification
> page, is passed to the system's opener (`open` on macOS; `xdg-open` on
> Linux, which the PLAN's step 3 did not name) as a process argument: it is in
> this machine's process list for the moment the opener runs, and the browser
> keeps it in its history. The authorize URL carries its state, nonce, PKCE
> challenge and client id; the plan's, host id; and the plan's second
> sign-in, as the shipped wizard's (0075-MADR D7), carries the first
> session's ID token as `id_token_hint`, which is identity data
> (`llmprovider/auth/oauth_plan.go:251`). That is what the shipped sign-ins
> already do with the same `OpenURL`; it is not new to the probe. None of it
> reaches the output or the file.
>
> **What the window shows, and what the file holds.** The probe's output is
> written to a file, never to the window, so that the one secret of a run,
> Kilo's user code (the path of its poll URL,
> `llmprovider/auth/kilo_device.go:80`), which the test writes to the terminal
> device (`/dev/tty`, which `go test`'s redirection does not capture), is alone
> in the window.
>
> * The window shows: for Kilo only, one line `Kilo code: <code>` per
>   sign-in, to read and type at Kilo's page; and, when each command ends, the
>   one line `go test exited N`. The browser opens by itself. `N` is not
>   proof that the run measured anything: `0` means the verdict is `kept`, and
>   also that the test skipped (a variable misspelt) or that `go test` found no
>   test to run; `1` means any other verdict, or a run that stopped, and also
>   that the redirect itself failed because `tmp/live-0081/` does not exist
>   (the shell prints its own error, `go test` never ran, and **no file was
>   written**). Only the file says which.
> * The file holds all of `go test`'s output and nothing else: a `=== RUN`
>   line, the step lines, the verdict line and the revocation lines, each
>   prefixed by `go test` with the name of the checks file and a line number,
>   then the `--- PASS:` or `--- FAIL:` line, `PASS` or `FAIL`, and the `ok`
>   or `FAIL` line with the package and the time. Nothing in it is a token, a
>   code, a URL or an id (`TestEarlierGrantReport_0081`); a build failure
>   would be in it too. The owner sends the file, whole, and nothing from the
>   window.
> * **A run that measured something has a verdict line** in its file:
>   `<provider>: kept`, `<provider>: ended` or `<provider>: indeterminate`. A
>   file with none (a `--- SKIP:` line, a stop line `<provider>: stopped: ...`,
>   a build error, or no file) is a run that did not measure; the owner says
>   which, and the run is made again after the cause is put right.
> * The file's place: `tmp/live-0081/<provider>.txt` in the checkout. The
>   directory is covered by the `tmp/` rule of the repository's `.gitignore`
>   (`git check-ignore -v tmp/live-0081/chatgpt.txt` names it), so nothing in
>   it can be staged by a plain `git add`.
>
> **The commands.** Last, on purpose: they are run only after the sections
> above have been read. Once, in the checkout, after the `git log` of step 1:
>
> ```sh
> mkdir -p tmp/live-0081
> ```
>
> Then, one provider at a time, as in steps 2 to 5 (the same sign-ins, in the
> same order of providers); each is one line, and carries `-timeout 15m`,
> because the run can now last up to nine minutes and forty seconds (above),
> too near `go test`'s default ten:
>
> ```sh
> LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=chatgpt go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -timeout 15m -v > tmp/live-0081/chatgpt.txt 2>&1; echo "go test exited $?"
> LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=plan go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -timeout 15m -v > tmp/live-0081/plan.txt 2>&1; echo "go test exited $?"
> LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=grok go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -timeout 15m -v > tmp/live-0081/grok.txt 2>&1; echo "go test exited $?"
> LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=kilo go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -timeout 15m -v > tmp/live-0081/kilo.txt 2>&1; echo "go test exited $?"
> ```
>
> Run them one at a time, each only after the one before has ended and its
> `go test exited` line is on screen. Each holds no secret, as before.
>
> **What each run prints** (in its file): seven or eight lines (three, when
> the baseline ends the run), then, for each grant the run obtained, one
> revocation line.
>
> * `chatgpt sign-in-1: ok`
> * `chatgpt baseline: status 200, kind ok, code -, requests 1`
> * `chatgpt sign-in-2: ok`
> * `chatgpt first-after: status 200, kind ok, code -, requests 1`
> * `chatgpt control: status 200, kind ok, code -, requests 1`
> * `chatgpt last-after: status 200, kind ok, code -, requests 1`, made only
>   if `first-after` succeeded: a refresh token the provider refused is not
>   sent again
> * `chatgpt: kept` (or `ended`, or `indeterminate`)
> * one revocation line per grant obtained, after the verdict line, whatever
>   the verdict: for `plan`, `plan revoke-1: status 200, kind ok` and `plan
>   revoke-2: status 200, kind ok`; for `kilo`, `kilo revoke-N: status 0, kind
>   unsupported`; for `chatgpt` and `grok`, `<provider> revoke-1: status 0, kind
>   by_hand` and `<provider> revoke-2: status 0, kind by_hand`, because the run
>   does not revoke them (above)
>
> The count is of the round trips the step made. The SDK sends a refresh
> again, with the same refresh token, up to three times in all, when its
> request fails in transport or is answered 429 or 5xx, so `requests 2` or `3`
> on a refresh says that the token was sent more than once. By the owner's
> ruling "The run is `indeterminate`: run it once more (Recommended)", any use
> step that made more than one request makes the whole run `indeterminate`,
> whatever its status was; the count stays in the step line, and the re-run is
> step 6's.
>
> **The verdict, by the owner's ruling "Add one step: refresh A once more
> after the control (Recommended)".** A is the first session, B the second.
> `baseline`, `first-after`, `last-after` and `control` are uses; "dead" is a
> refusal as the grant gone (a 400 or 401 with `invalid_grant`,
> `invalid_client`, `refresh_token_expired` or `refresh_token_invalidated`;
> Kilo's: a 401, and only a 401: a 403 is the profile not being able to say);
> "reused" is `refresh_token_reused`; "other" is any other failure. Any step
> with `requests` above 1 first makes the verdict `indeterminate`; otherwise:
>
> | baseline | control | first-after | last-after | verdict |
> | :--- | :--- | :--- | :--- | :--- |
> | not ok, or ok but with `requests` above 1 | not made | not made | not made | `indeterminate`; the run ends before the second sign-in |
> | any | not ok | any | any | `indeterminate` |
> | ok | ok | ok | ok | `kept` |
> | ok | ok | ok | dead | `ended` (the lazy case: the earlier grant answered once after the new sign-in, then did not) |
> | ok | ok | ok | reused | `indeterminate` |
> | ok | ok | ok | other | `indeterminate` |
> | ok | ok | dead | not made | `ended` |
> | ok | ok | reused | not made | `indeterminate` |
> | ok | ok | other | not made | `indeterminate` |
>
> A baseline that failed, or that succeeded but made more than one request,
> ends the run at once, `indeterminate`, before the second sign-in: its step
> line prints with its count, then the verdict line, then the first grant's
> revocation line (`by_hand` for `chatgpt` and `grok`). The owner is not asked
> for a sign-in that could not change the verdict. `kept` says nothing of a grant that ends minutes after the run.
> What a `last-after` that is made after a failed `first-after` would add is
> nothing: the same refresh token would be sent again.
>
> **The sessions** are in the test's memory only and are gone when it ends.
> Nothing is written to the owner's stored files, and the probe opens none.

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

   > **DATED NOTE, 2026-11-19: THE COMMANDS OF STEPS 2 TO 5 BELOW ARE REPLACED
   > BY THE AMENDMENT ABOVE. DO NOT RUN THEM.** Use the commands in the amendment's section "The commands" (above, in the amendment that opens Phase 1b), after
   > reading its first section, "Before any command: what a run does to your
   > accounts, and what is not known". Those commands carry `-timeout 15m` and
   > write the test's output to `tmp/live-0081/<provider>.txt`, so that the Kilo
   > code is alone in the window; run its `mkdir -p tmp/live-0081` once, after
   > this step. The old commands below, run as well, would sign in twice more at
   > the provider and leave two more grants under the owner's account (the
   > `chatgpt` and `grok` ones by hand), with Kilo's code in the same window as
   > the test's output. Steps 2 to 5 still say which sign-ins to make and in what
   > order, and nothing else of theirs stands.

2. The ChatGPT browser sign-in, twice: sign in in the browser window each
   time it opens, with the same account:

   > **REPLACED, 2026-11-19: do not run the command below.** Run the `chatgpt`
   > command of the amendment's section "The commands" (above, in the amendment that opens Phase 1b) instead.

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=chatgpt go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

3. The ChatGPT plan sign-in, twice; the second may show OpenAI's consent
   again:

   > **REPLACED, 2026-11-19: do not run the command below.** Run the `plan`
   > command of the amendment's section "The commands" (above, in the amendment that opens Phase 1b) instead.

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=plan go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

4. Grok's browser sign-in, twice:

   > **REPLACED, 2026-11-19: do not run the command below.** Run the `grok`
   > command of the amendment's section "The commands" (above, in the amendment that opens Phase 1b) instead.

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=grok go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

5. Kilo's device sign-in, twice: Kilo's page opens in the browser; enter
   the code the terminal shows as `Kilo code: ...`, and approve, each
   time:

   > **REPLACED, 2026-11-19: do not run the command below.** Run the `kilo`
   > command of the amendment's section "The commands" (above, in the amendment that opens Phase 1b) instead.

   ```sh
   LLMPROVIDER_LIVE_0081=1 LLMPROVIDER_LIVE_0081_PROVIDER=kilo go test -tags live_gateways ./llmprovider -run '^TestLive_EarlierGrant_0081$' -count=1 -v
   ```

6. A run whose verdict is `indeterminate` is run once more, the same
   command; a second `indeterminate` is a stop.

   > **Dated note, 2026-11-19.** "The same command" is the amendment's command
   > for that provider. A re-run signs in twice more and leaves two more grants
   > (by hand, for `chatgpt` and `grok`). An `indeterminate` is also any run in
   > which a use step made more than one request, or whose baseline failed or did;
   > a file with no verdict line is not an `indeterminate` but a run that did not
   > measure, to be put right and made again (the amendment says what such a file
   > holds).

7. Close the terminal window.

   > **Dated note, 2026-11-19.** Close it after the four files of
   > `tmp/live-0081/` have been sent (the amendment's "What the window shows, and
   > what the file holds").

> **Dated note, 2026-11-19: SUPERSEDED.** A run now prints seven or eight lines
> (three, when the baseline ends it) and one revocation line per grant, in its
> file, `tmp/live-0081/<provider>.txt`, which the owner sends, whole; each use
> step's line ends `, requests N`, and a `last-after` line follows `control`.
> The amendment's "What each run prints" says it, with the lines for each
> provider. The paragraph below describes the first draft.

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

> **Dated note, 2026-11-19.** The verdicts are defined by the table in the
> amendment's "The verdict", not by the first draft's: `ended` includes the lazy
> case (the earlier grant answers once after the new sign-in and then does not);
> `indeterminate` includes any run in which a use step made more than one
> request and any run whose baseline failed; and only `kept` passes the test, so
> a `go test exited 1` is any verdict but `kept`, or a run that did not measure.
> "`kept` for all four" says nothing of a grant that ends minutes after a run.

**Dated note, 2026-11-20: the outcome.** The four runs of 2026-11-19 gave
`kept` for `chatgpt`, `plan`, `grok` and `kilo` (the execution record, "Phase
1c"). The table's first row applies: "`kept` for all four | the premise of
0059-MADR Q2 (a) and of saving last holds for those sign-ins; Phase 2
proceeds". Its other three rows (`ended` for any one; `indeterminate` twice;
a provider not run) **no longer apply**, and nor does the first stop
condition of "Rules for every phase" (`ended`, `indeterminate` twice,
`refresh_token_reused`, a provider not run), nor its second (no line of the
four files shows a token, a code, a client id or a URL).

#### Phase 1c: the record (an agent)

Model: lightweight, low.

1. The execution record quotes every line the owner sent, per provider.
2. The MADR's D12 gets a dated amendment line per provider with its
   verdict; "Not verified" loses N1 for the providers measured.
3. Commit the two records' edits (records only).

Acceptance: four verdicts recorded, each `kept`; or a stop, with the
owner's resolution recorded as a dated deviation before Phase 2.

> **Dated note, 2026-11-19.** What the owner sends is a file per provider
> (`tmp/live-0081/<provider>.txt`), and step 1 quotes its lines, the revocation
> lines included, and records, for `chatgpt` and `grok`, the two grants left to be
> removed by hand, with the owner's word when they are. A file with no verdict
> line is not a verdict.

**Dated note, 2026-11-20: done, but step 3.** Steps 1 and 2 are written: the
execution record's "Phase 1c (2026-11-20)" and the MADR's amendment of
2026-11-20, "D12: the four verdicts", with "Not verified" annotated in
place. The acceptance's first branch is met: four verdicts recorded, each
`kept`. Step 3, the commit, is the coordinator's: the readiness pass that
wrote them stages and commits nothing.

### Phase 2: the tests and the code, one commit

> **Dated addition, 2026-11-20** (the readiness pass; the execution record's
> entry of that date holds the state table's test map). **Step 0, before step
> 1:** re-ground every line the MADR and this PLAN cite at the phase's start
> commit, with `python3 -B tools/claims.py` on both records at their markers
> and on a scratch copy of the re-pointed tables pinned at that commit, and
> the prose citations the entry lists; record each that moved. **The tests
> also include** the proposed P-1 to P-16 that belong to this phase (the
> entry's "Proposed tests"). **Exit criterion:** no blocker, every row tested,
> plants caught. **Before it starts**, by the owner's rule for every record
> that changes behaviour (its state table decides every case before code; the
> open cases go to the owner in one prompt), the owner's answers to the state
> table's open rows are recorded. They are: every row is decided or out of
> scope after the third pass of 2026-11-20 (the owner's rulings on S50,
> S61, S62, S48, S60, S76 and S77; S18 needs no ruling).

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

   > **Annotation, 2026-11-18** (the MADR's amendment of the same date and this
   > PLAN's deviation entry). At `a7e924f` the first four rows of this table are
   > not red: record 0082 put a context check in `resolvedCredential.write`
   > (`wizard/auth.go:1181`), so `TestCancelDuringListingSavesNothing_0081`,
   > `TestCancelAfterIssueSavesNothing_0081`,
   > `TestCancelDuringKiloProfileSavesNothing_0081` and
   > `TestDeadlineDuringListingSavesNothing_0081` are green there. They are
   > **guards**, each held by the plant "`write`'s context check removed". Their
   > red column stands as the history at `b109cea`, where they are red. The
   > tests red at the base are the other eight: `TestCancelledConfigureReturnsNoKey_0081`,
   > `TestCancelAfterSaveReturnsTheResult_0081`,
   > `TestCancelAtModelStageSavesNothing_0081`,
   > `TestCancelAtModelStageReturnsNoKey_0081`,
   > `TestCancelAtManualModelSavesNothing_0081`,
   > `TestConfigureSavesOnceAfterTheModel_0081`,
   > `TestModelStageErrorSavesNothing_0081` and
   > `TestRotationDuringQuestionsSavedAtEnd_0081`. The three guards of the table
   > are green at both revisions. `TestCancelDuringDeviceFlowSavesNothing_0081`
   > and the ChatGPT plan's cases of the first row are still owed (the deviation
   > entry says what they need).

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

   > **Annotation, 2026-11-18.** At `a7e924f` the session's windows (K2 to K4,
   > K6) are covered by `write`'s check, which 0082 built and whose place 0082's
   > N-6 fixes: after the lock, on every pass of the retry loop. Phase 2 does
   > not move, remove or repeat it. What remains to build is: `commitCredential`
   > (its check is the one a key and any no-session credential lacks; for a
   > session it is the second check, before the lock); `ResolveCredential`
   > without its own last check; `ConfigureLLM` calling the unsaved step,
   > checking after it (N3) and committing at its end (N2). Phase 3's plants
   > gain two on `write`'s check, each to name the test that fails: the check
   > moved before the lock's wait, and the check skipped on the retry pass; a
   > plant no test catches is a missing test, which Phase 2 adds. The Phase 2
   > steps otherwise stand as written, and none starts before Phase 1c.

   **Annotation, 2026-11-20** (re-grounded at `24756a6`). Phase 1c is
   recorded (`kept` for all four). `commitCredential` takes the run's
   `Prompter`: `save` has taken one since 0082 (`wizard/auth.go:1103`, for
   the lock's notice and retry question), so the step's `commitCredential(ctx,
   d, c)` and `c.save(ctx, d)` read `commitCredential(ctx, p, d, c)` and
   `c.save(ctx, p, d)`. Its own check stays its first statement, before
   `save`, so its error, `wizard: <ctx>`, is the one a run returns whenever
   the context has ended by the commit (MADR D9 as amended on 2026-11-18);
   `write`'s words are returned only for a context that ends after that check,
   during the lock's wait or its retry question. `ConfigureLLM`'s commit
   therefore runs 0082's lock wait, notice and retry question after the model
   and fallback questions (the MADR's state table, S35).

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

   **Dated addition, 2026-11-20** (the owner's rulings of that date, relayed
   by the coordinator, and the review's decidable findings; the MADR's
   "Second pass" and rows S50, S60 to S62, S67 to S77). Step 5 also writes,
   and step 4 also changes:

   * **Files added to the phase:** `wizard/kilo_verify.go` (`verifyKilo`'s
     notice, `:128`); `discoverModels` in `wizard/configure.go` (the
     listing's warnings, `:948`, `:951`, `:958`, `:967`, `:974` and `:977`);
     `resolveKiloDevice` in `wizard/auth.go` (its organizations warning,
     `:478`, now code, not only a doc comment);
     `wizard/kilo_verify_0080_test.go` (its expected words, `:362` and
     `:383`, the one existing test the ruling "Fix both" changes);
     `docs/architecture.md:1148-1149` and `CHANGELOG.md`'s unreleased 0080
     entry (lines 40-41), which quote today's Kilo words; a dated one-line
     note in 0080-MADR D6 (line 433), whose words S76 replaces.
   * **The notices** (S61, S67 to S77): Kilo's notice reads "could not
     verify the Kilo key (<class>[, HTTP <status>]); going on without
     verifying it" (S76), in the code, in
     `wizard/kilo_verify_0080_test.go:362` and `:383`, in `CHANGELOG.md`
     lines 40-41 and in `docs/architecture.md:1148-1149`, and 0080-MADR D6
     gets a dated one-line note naming the new words; every listing warning
     (`wizard/configure.go:948`, `:951`, `:958`, `:967`, `:974`, `:977`) is
     not shown once the run's context has ended, by cancel or deadline, and
     is shown as today with it alive, the listing's own limit included
     (S61, S71, S72, S77); the organizations warning without `Discover`
     likewise (S73, S74). Which
     context: the run's (`ctx`), never the listing's own (`dCtx`).
   * **`ConfigureLLM`'s doc and `Options.TokenStore`'s** (S50): one sentence,
     that two runs saving to one store leave the session of the last to
     commit.
   * **The architecture note, "When a session is saved"** (S60, S62): one
     sentence for Kilo, that an abandoned Kilo device sign-in leaves a token
     that does not expire and that the SDK cannot revoke, which the person
     removes in the Kilo account's settings; and one sentence that the
     earlier grant was measured kept on 2026-11-19 for the ChatGPT browser
     sign-in, the plan, Grok's browser sign-in and Kilo's device sign-in, the
     OpenAI and Grok device sign-ins not probed. `ConfigureLLM`'s doc says the
     Kilo sentence too. Neither copies D6's "lapses" for Kilo.
   * **`save`'s doc comment** (the review's finding 3): its last sentence,
     "The context is checked after the lock is held, before anything is
     written: a lock that outlasts the sign-in saves nothing (0081-MADR's
     single check before the save belongs here, after the lock, and not before
     the wait)", becomes: "The context is checked after the lock is held,
     before anything is written: a lock that outlasts the sign-in saves
     nothing. This is the check that covers the wait (0082-MADR N-6); the
     caller, commitCredential, has already checked the context once, before
     the wait, for every credential (0081-MADR D5)." 0082-MADR N-6 carries a
     dated note of 2026-11-20 to the same effect, written by the readiness
     pass.
   * **Overlap with 0085**, which by the owner's ruling rewords the comment at
     `wizard/kilo_verify.go:95` in its own phase: the same function,
     `verifyKilo`, different lines. 0081 builds first; 0085 re-reads
     `verifyKilo` at its base, its notice included.

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

**Dated note, 2026-11-20 (build pass 1).** Built, not yet staged: see the
execution record's "Phase 2, build pass 1". Two things it records that this
phase's text does not say: five of the tests the table calls guards are red at
the base on the words (its Deviation 1), and the existing test the acceptance
says is unchanged, `wizard/kilo_verify_0080_test.go`, changes its expected
words, as step 5's addition allows.

**Dated note, 2026-11-20 (the owner's ruling on build pass 1's Deviation 1).**
By the owner, relayed by the coordinator: "Keep as built; record them as red before the change (Recommended)". The five tests the table above calls guards, `TestCancelDuringListingSavesNothing_0081`, `TestCancelAfterIssueSavesNothing_0081`, `TestCancelDuringKiloProfileSavesNothing_0081`, `TestDeadlineDuringListingSavesNothing_0081` and `TestCancelDuringListingReturnsNoKey_0081`,
are red at the base by their words and no-model-question assertions, and
their count assertions (no save, an error matching the context's) are green
there. They are tests red before the change, by that ruling; their "guard"
labels in the table and the annotation of 2026-11-18 are superseded, and the
MADR's state table carries a dated annotation of the same date on the rows
that name them (S7, S8, S11, S24, S28, S31, S37). The other guards stand:
`TestCancelDuringRefreshSavesNothing_0081`, `TestResolveSavesInTheStep_0081`.

### Phase 3: planted breaks and the gates

Model: mainstream, low.

> **Dated addition, 2026-11-20.** **Step 0, before step 1:** re-ground the
> cited lines at the phase's start commit (Phase 2's commit), as Phase 2's step
> 0 does, and record each that moved. **The plants** are re-mapped, and gain
> one per proposed test, in the execution record's entry of 2026-11-20 ("The
> plants, re-mapped after 0082"); that list replaces step 1's test names where
> they differ. **One run on Windows:** `go test -count=1 ./wizard/` through
> `~/.agents/skills/available-dev-hosts/scripts/run-on-host` with `--need`
> pinning `windows` (P-16 kills a process, and the write is `FileTokenStore`'s
> per platform). **Exit criterion:** no blocker, every row tested, plants
> caught.

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

> **Dated addition, 2026-11-20.** **Step 0, before step 1:** re-ground the
> cited lines at the release's commit, as Phase 2's step 0 does, and record
> each that moved. **Exit criterion:** no blocker, every row tested, plants
> caught (as Phase 3 left them, re-run on the release's commit by its CI).

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

### Approval (2026-11-17)

* **Approved 2026-11-17.** This PLAN was approved through its live step or
  release by the owner's ruling of 2026-11-17, relayed by the coordinator:
  "Accept all five and run each plan up to its live step or release
  (Recommended)". Its phases are committed and pushed by the coordinator as
  each passes. Tags and releases need the owner's word.

### Deviation of 2026-11-18: 0082 already guards a session's write

Found at the start of the first build attempt, before anything was built, and
put to the owner by the coordinator; the amendment of the same date in the
MADR carries the full account.

* **Evidence.** `wizard/auth.go:1180` to `:1183`: `resolvedCredential.write`,
  which every pass of `save`'s lock loop and the no-lock path run, begins with
  `if err := ctx.Err(); err != nil { return fmt.Errorf("wizard: the new %s
  session was not saved: %w", ...) }`; the session is given its store and
  saved after it (`:1184`, `:1185`). Built by record 0082 (commits `8051b00`
  to `fd986c4`); its N-6 is the note addressed to this record. Not
  pre-existing at `b109cea`, where `save`'s only check was the conjunction with
  `Token` (C2). On a scratch copy of `a7e924f` the drafted tests run as the
  MADR's amendment tabulates: four of the fifteen that were red at `b109cea`
  are green, eight are red, three were guards.
* **What it changes.** The PLAN's Phase 2 table calls four tests red at the
  base. They are not red at `a7e924f`. The PLAN's stop condition "a new test
  cannot be red at the base without changing code that is not a test" would
  have applied to them; the owner's ruling below is the resolution.
* **Resolutions offered** (the three options of the stop report, the first
  chosen): amend the record by a dated note and keep the four tests as guards
  (recommended); build against the records as written and report each as a
  documented deviation; drop the four tests and rely on plants alone (loses
  the test for each session kind that D1 asks for).
* **The owner's ruling, 2026-11-18** (relayed by the coordinator):
  "Amend the record by dated note; keep the four tests as guards
  (Recommended)". And on the order of work: "As the plan says: build the
  probe now, I run it, then the code is built (Recommended)".
* **Consequences recorded in this PLAN.**
  * The four tests are guards, each with its plant: `write`'s context check
    (`wizard/auth.go:1181` to `:1183`) removed. Seen to fail on a scratch copy
    of `a7e924f`: `TestCancelDuringListingSavesNothing_0081` ("a sign-in
    cancelled during its listing saved 1 time(s); want none", twelve cases),
    `TestCancelAfterIssueSavesNothing_0081` ("a sign-in cancelled after the
    issue saved 1 time(s); want none", both cases),
    `TestCancelDuringKiloProfileSavesNothing_0081` ("a sign-in cancelled
    during the profile read saved 1 time(s); want none"),
    `TestDeadlineDuringListingSavesNothing_0081` ("a sign-in whose deadline
    passed during its listing saved 1 time(s); want none"). The other three
    guards are held by the plants of Phase 3.
  * Phase 2's table and steps are annotated in place (additions only). What
    remains to build is the key path (`commitCredential`'s check),
    `ResolveCredential`'s own last check removed, and `ConfigureLLM` saving at
    its very end after a check right after the step.
  * **Owed in Phase 2.** `TestCancelDuringDeviceFlowSavesNothing_0081`, which
    this PLAN adds and the drafting never wrote. It needs: Grok's device
    sign-in through the real `auth` device flow, not the wizard's `deviceLogin`
    seam (`wizard/device.go:31`); the issuer's device-code and token
    endpoints answered in process through the run's `HTTPClient`, with no
    listener; the issuer's poll interval (the auth package's own device tests
    use 1 s, `llmprovider/auth/device_login_test.go:32`), so the test waits at
    least one interval; the approval given, then the context ended during the
    listing; and the expectation of the context's error and no save. If the
    flow cannot be served in process without changing non-test code, the test
    is dropped and the execution record says why, as the table's row says.
    Likewise the ChatGPT plan's cases, with its consent scripted, which the
    first row adds in Phase 2.
* **The code claims, re-pointed at `a7e924f`** (the PLAN's own table above is
  unchanged, at `b109cea`). Each row was checked by `tools/claims.py` on a
  scratch copy with plain ids and `self=a7e924f`: 25 of 25 pass. 14
  unchanged, 9 moved with the same text, 2 moved with changed text (P3, P7),
  none no longer true.

| Id | Claim | Now | Location at `a7e924f` | Text |
| :--- | :--- | :--- | :--- | :--- |
| P1 now | The README's sign-in bullet | moved from line 388; text and meaning the same | `README.md:434` | `` - A sign-in saves its session to `Options.TokenStore`, once the model `` |
| P2 now | The architecture's save paragraph | moved from line 391; text and meaning the same | `docs/architecture.md:393` | `` sign-in. A sign-in's session is saved to `Options.TokenStore` once the `` |
| P3 now | and its "When a session is saved" | moved; the line was reflowed by 0080, same sentence | `docs/architecture.md:1120` | `- **When a session is saved.** A new session is saved once the check, the` |
| P4 now | The changelog's open section | unchanged | `CHANGELOG.md:12` | `## [Unreleased]` |
| P5 now | `Wait`'s doc | moved from line 251; text and meaning the same | `llmprovider/x/signin/signin.go:267` | `// Wait returns the credential the sign-in yields, or its error, which` |
| P6 now | `ResolveCredential`'s last check's comment | moved from line 376; text and meaning the same | `wizard/configure.go:433` | `// A listing ctx ended is not a refusal: it falls back to the static` |
| P7 now | `credentialStep`'s doc | moved; 0080 rewrote the first line: it now says the step checks, then lists | `wizard/configure.go:484` | `// credentialStep resolves d's endpoint and credential, then checks it` |
| P8 now | The test seams: a fake transport | unchanged | `wizard/auth_test.go:349` | `type roundTripFunc func(*http.Request) (*http.Response, error)` |
| P9 now | a session | unchanged | `wizard/auth_test.go:240` | `func testOAuthSession(provider llmprovider.ProviderID) *auth.OAuthSession {` |
| P10 now | a JSON answer | unchanged | `wizard/session_0059_test.go:67` | `func answer0059(r *http.Request, status int, body string) *http.Response {` |
| P11 now | a third party with no static catalog | unchanged | `wizard/describe_0056_test.go:94` | `func registry0056(t *testing.T, static []string, err error) *llmprovider.Registry {` |
| P12 now | the provider menu's index | unchanged | `wizard/configure_test.go:21` | `func providerIdx(t *testing.T, id llmprovider.ProviderID) int {` |
| P13 now | 0059's W2 | unchanged | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:88` | `* **W2. A fresh sign-in replaces the saved session before the listing` |
| P14 now | and its amendment | moved from line 490; text and meaning the same | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:494` | `## Amendment 2026-10-24: a refresh can outlast the listing (Phase 2)` |
| P15 now | 0056's D1 | unchanged | `docs/decisions/0056-MADR-wizard-credential-only-entry-point.md:136` | `` ### D1. `wizard.ResolveCredential` `` |
| P16 now | `saveTo`'s comment | unchanged | `wizard/auth.go:55` | `// saveTo is the store a new session is due to: credentialStep saves it` |
| P17 now | `Options.TokenStore`'s doc | moved from line 117; text and meaning the same | `wizard/configure.go:128` | `// TokenStore persists sessions created by the browser, device-code and` |
| P18 now | 0059's W8 | moved from line 143; text and meaning the same | `docs/decisions/0059-MADR-wizard-debugging-pass-findings.md:147` | `` * **W8. `ConfigureLLM` asks questions on an ended context** **[probed]** `` |
| P19 now | A record's own live variable | unchanged | `AGENTS.md:266` | `` switched on by `LLMPROVIDER_LIVE_<NNNN>`, with suffixes for its options, as `` |
| P20 now | The plan's live sign-in registers this module's name | unchanged | `llmprovider/live_chatgpt_plan_test.go:17` | `const planAgentName = "go-llmprovider-sdk"` |
| P21 now | auth's terminal refresh codes, the probe's allow-list | unchanged | `llmprovider/auth/oauth_session.go:62` | `"invalid_grant", "invalid_client",` |
| P22 now | A session can be made due, to force one refresh | moved from line 484; text and meaning the same | `llmprovider/auth/oauth_session.go:595` | `func (s *OAuthSession) Invalidate() {` |
| P23 now | The plan sign-in takes the earlier session as `Previous` | unchanged | `llmprovider/auth/oauth_plan.go:61` | `Previous *OAuthSession` |
| P24 now | Kilo's profile read, the probe's use of a Kilo session | moved from line 179; text and meaning the same | `llmprovider/auth/kilo_device.go:187` | `func KiloProfile(ctx context.Context, token string, opts ...llmprovider.Option) (KiloAccount, error) {` |
| P25 now | The checks a live test shares with its offline twin | unchanged | `llmprovider/live_checks_test.go:3` | `// The checks the person-gated live tests run after their sign-in, untagged so` |

* **What did not change.** Phase 1's probe is built as the PLAN writes it
  (Phase 1a, next); Phase 1b is the owner's; no code of Phase 2 is built.

### Phase 1a (2026-11-18): the probe, built and not run

Built under the owner's ruling "As the plan says: build the probe now, I run
it, then the code is built (Recommended)" (the deviation entry above). No
agent ran the live test, signed in, or held a token or a session; the live
test was compiled and vetted only.

* **Files.** `llmprovider/live_0081_grant_test.go` (new, `//go:build
  live_gateways`): `TestLive_EarlierGrant_0081`, the real sign-ins, the real
  browser opener and terminal device. `llmprovider/live_checks_test.go`
  (untagged, appended): the recorder, the step and verdict code, `earlierGrant`
  with its `run`, the two uses of a session. `llmprovider/live_twins_test.go`
  (untagged, appended): `TestEarlierGrantReport_0081` (the redaction proof and
  every verdict, twenty-two cases), `TestEarlierGrantStops_0081` (four),
  `TestGrantRecorder_0081` (four). `AGENTS.md` ("Live tests"): one sentence
  naming the probe, its two variables and where its commands are. No exported
  name; no existing test changed.
* **The PLAN's steps, and where each is.**
  * Step 1, the switch: the test's first statement skips unless
    `LLMPROVIDER_LIVE_0081` is `1`; the provider is validated against
    `chatgpt`, `plan`, `grok`, `kilo` before any request, and any other value
    fails with a fixed message that does not repeat it. Both variables are the
    record's own (`tables-check` below).
  * Step 2, memory only: no store is opened or passed; `os.Getenv` is called
    for the two variables only (a search of the new code finds no other; the
    HTTP transport reads the proxy variables, and the browser opener inherits
    the environment); the plan's two sign-ins share a random host id
    the test makes (the second takes it from `Previous`).
  * Step 3, what the person sees: `grantShow.URL` hands a URL to the system's
    opener only; `grantShow.Code` writes `Kilo code: <code>` to `/dev/tty`
    only; nothing is pasted (no `InputCode`); on either failing the run stops
    with `<provider>: stopped: the browser could not be opened` or `the
    terminal device could not be opened`, which hold neither the URL, the code
    nor the error's text. The opener and terminal are `grantIO` seams the twin
    replaces.
  * Step 4, the sequence: `earlierGrant.run`: sign-in-1, baseline, sign-in-2
    (the plan's with the first session as `Previous`), first-after, control.
  * Step 5, the use: one forced refresh (`Invalidate`, then `Token`) for
    chatgpt, plan and grok, so each refresh token is sent once; one
    `auth.KiloProfile` read for Kilo.
  * Step 6, the record: `grantRecorder`, the transport of every request;
    `"%s %s: status %d, kind %s, code %s"`, the sign-in steps' `"%s %s: %s"`,
    and `"%s: %s"` for the verdict, each line passed through
    `redact.String`.
  * Step 7, the proof: the twin, below.
  * Step 8, the read: below.
* **Where the PLAN left room, and the reading taken.** None is a deviation
  (each is inside a step as written); each is for review.
  * A sign-in that fails ends the run with the verdict `indeterminate` (the
    later steps cannot be measured); a failed baseline or control does not
    stop it, and the verdict is `indeterminate`.
  * `ended` is read as step 6's parenthesis says: kind `auth`, a status of 400
    or 401 and a terminal code other than `refresh_token_reused`; for Kilo,
    kind `auth` and a 401.
  * The recorder's code is `other` for an error field that is not on the
    allow-list, and `-` for a refusal with no OAuth error field at all (a
    plain 401, a body that is not JSON).
  * Only `kept` passes the test; `ended` and `indeterminate` fail it, so that
    `go test` says so, with the verdict line above.
  * One sign-in waits up to four minutes and the run is bound by nine, inside
    `go test`'s default ten, so the owner's commands (which carry no
    `-timeout`) stand as written.
  * The opener is `open` (macOS) or `xdg-open` (Linux), given an http or
    https URL only; elsewhere the run stops with the browser line.
* **Step 8's read.** The three files were read. Every log or failure call in
  the new code, by search: `t.Skip` and `t.Fatal` with fixed messages (the
  last, `t.Fatal(stopped)`, a line built from a validated provider name and
  one of two fixed reasons), `t.Fail()` with no message, and `earlierGrant.say`
  at three sites, each given a provider name, a fixed step name, a status, an
  error's kind word and an allow-listed code or `-`. No call is given a
  session, a token, a variable holding one, an error's text or a body.
* **Without the switch, and without the tag.** With the tag and the switch
  unset the test's first statement is `t.Skip`, before any request, any
  session and any read of a credential; the package has no `TestMain` and the
  new code has no `init`. Without the tag the file is not compiled: `go list`
  lists `live_0081_grant_test.go` among the test files only with
  `-tags live_gateways`.
* **Twin, red then green.** Each verdict branch and each guard of the
  redaction was seen to fail on a plant, on a scratch clone with
  `tools/plant.py` (17 of 17 caught, each by a test failure, none by a build
  failure; the spec and output are kept outside the tree), then pass on the
  unplanted tree:

  | Plant | A test it failed |
  | :--- | :--- |
  | the step line is given the error's text | `TestEarlierGrantReport_0081/chatgpt/ended`: "the output holds a planted value (40 characters, starting "rt00")" |
  | the recorder keeps a code outside the allow-list | `TestEarlierGrantReport_0081/chatgpt/control_fails`, and the three other providers' |
  | the verdict is computed without the control | `TestEarlierGrantReport_0081/chatgpt/control_fails`, all four providers |
  | the verdict is computed without the baseline | `TestEarlierGrantReport_0081/chatgpt/baseline_fails`, all four providers |
  | the user code is logged to the output as well as the terminal | `TestEarlierGrantReport_0081/kilo/kept`, and `TestEarlierGrantStops_0081` |
  | the opener's failure falls back to logging the URL | `TestEarlierGrantStops_0081/the_browser_fails`: "the output holds a planted value (40 characters, starting "http")" |
  | a kept verdict is reported as ended | `TestEarlierGrantReport_0081/chatgpt/kept`, all four providers |
  | an ended verdict is reported as indeterminate | `TestEarlierGrantReport_0081/chatgpt/ended`, all four providers |
  | an indeterminate verdict is reported as kept | `TestEarlierGrantReport_0081/chatgpt/reused`, and the unlisted-code and plain-401 cases |
  | `refresh_token_reused` is read as an ended grant | `TestEarlierGrantReport_0081/chatgpt/reused`, plan, grok |
  | Kilo's ended grant is not its 401 | `TestEarlierGrantReport_0081/kilo/ended` |
  | the plan's second sign-in is made without `Previous` | `TestEarlierGrantReport_0081/plan/kept`: "the plan's second sign-in was not made with the first session as Previous" |
  | the recorder reads a successful response's body | `TestGrantRecorder_0081/a_success's_body_is_not_read` |
  | the recorder does not put a refusal's body back | `TestEarlierGrantReport_0081/chatgpt/ended` |
  | the recorder reads the request's body | `TestGrantRecorder_0081` |
  | the terminal device is not checked before the first sign-in | `TestEarlierGrantStops_0081/the_terminal_device_cannot_be_opened` |
  | the browser opener is not checked before the first sign-in | `TestEarlierGrantStops_0081/there_is_no_browser_opener`: "stopped "", verdict "kept", 2 sign-ins; want "grok: stopped: the browser could not be opened", none, 0" |

  Survivors: none. The twin also shows its own precondition: the SDK's error
  for a refused refresh carries the planted echo, so keeping the error's text
  out of the output is a thing the test can fail on (the `ended` cases assert
  it).
* **Gates** (each with its status taken before any filter; heavy runs through
  `run-heavy` or `run-on-host`):

  | Gate | Where | Status |
  | :--- | :--- | :--- |
  | `make pre-add-check FILES="llmprovider/live_checks_test.go llmprovider/live_twins_test.go llmprovider/live_0081_grant_test.go"` (gofmt, lint for the host and `GOOS=windows`, vet, test, govulncheck) | session host, `run-heavy` | 0; its first run failed on one `bodyclose` finding in the new recorder test, fixed by closing the body |
  | `go vet ./...`, `GOOS=windows go vet ./...`, `go vet -tags live_gateways ./...` | session host | 0, 0, 0 |
  | `go test -tags live_gateways -c -o /dev/null ./llmprovider/` (compiled, never run) | session host | 0 |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session host | 0; the one addition listed is `llmprovider/x/signin.VerifyKilo` (0080), none from this change |
  | `go test -race -count=20 -timeout 30m ./llmprovider/` | session host (darwin/arm64), `run-heavy` | 0, 225 s |
  | `go test -race -count=1 -timeout 45m ./...` | `pick-host` chose a Windows machine, native, through `run-on-host --race` (not a WSL distro) | 0, 43 packages ok, none failing |
  | the 17 plants | scratch clone | 17 of 17 caught, twice (before and after the `bodyclose` fix) |

  The record and docs gates after the last record edit are in the handoff.
* **Not run, by design.** The live test itself; no environment variable of
  the probe was set in any command.

### Phase 1a, review fixes (2026-11-18)

The independent review of the staged Phase 1a returned PASS WITH FINDINGS:
the output's secrecy holds, and no request bypasses the injected client. The
findings below were fixed in place, before any command was handed to the
owner; nothing was committed. The live test was not run, and every experiment
ran with `LLMPROVIDER_LIVE_0081` and `LLMPROVIDER_LIVE_0081_PROVIDER` unset
and, for the twin and the plants, under a sandbox that denies every outbound
connection but loopback.

* **F1, the recorder read one shape of the code.** `grantCodeOf`
  (`llmprovider/live_checks_test.go:1420`) now reads `{"error": "code"}`,
  `{"error": {"code": "code"}}` and `{"code": "code"}`, lower-cased, in the
  SDK's own order, then keeps the code only if it is on the allow-list. Before,
  a nested `refresh_token_invalidated` printed `code other` and the verdict
  `indeterminate` where it should have been `ended`, a nested
  `refresh_token_reused` printed `other` so the stop on it could not be seen,
  and an upper-case code printed `other`. Question: can the recorder share
  `refreshErrorCode`'s logic? **No.** It is unexported in package `auth`; a
  test of package `llmprovider_test` cannot name it (a scratch file naming it
  fails: `undefined: auth.refreshErrorCode`), and `auth` has no
  `export_test.go`, which in any case reaches only `auth`'s own tests; exporting
  it is an API change that this record does not make. So it is a copy, and
  `TestGrantCodeAgreesWithTheSDK_0081` (`llmprovider/live_twins_test.go:2606`)
  fails if the two disagree: it sends 27 bodies (the five codes flat, nested,
  bare, upper and mixed case, and unlisted, empty, null, numeric and
  unparseable ones) to the SDK's real refresh as a 400, and requires that the
  SDK's verdict (an auth failure, a dead refresh token) equal "the recorder's
  code is terminal", and that the code equal the expected one. What it cannot
  see is the code's exact value, which the SDK does not expose; it sees
  whether it is terminal. Twin cases, each red first by a plant that undoes
  the fix (table below): `TestEarlierGrantReport_0081/the_shape:_*` (nested
  `invalid_grant`, `refresh_token_invalidated`, `refresh_token_expired`,
  `refresh_token_reused`, a bare code, a bare reused code, upper case, upper
  case reused).
* **F2, the time a sign-in has.** Phase 1b's amendment of the same date
  says each sign-in must finish within four minutes, how the run's nine
  minutes are spent (two full sign-ins leave a minute for four uses, which
  take seconds, and the revocations have a bound of their own, so a person who
  takes the whole four minutes twice is not cut off), and that a sign-in
  that takes longer is cancelled, the result `indeterminate` and the re-run
  two new sign-ins.
* **F3, the terminal and the output.** The four commands now write the test's
  output to `tmp/live-0081/<provider>.txt` and end with `echo "go test exited
  $?"`; the window shows only Kilo's user code (written to `/dev/tty`, which
  the redirection does not capture) and that one line, and the file holds all
  of `go test`'s output. `.gitignore`'s `tmp/` rule covers the directory
  (`git check-ignore -v tmp/live-0081/chatgpt.txt` names `.gitignore:15`); no
  rule was added. The amendment says what the window shows and what the file
  holds.
* **F4.** Phase 1b's amendment and Phase 1a's step 3 annotation say that the
  authorize URL goes to the opener as a process argument and into the
  browser's history, that the plan's second URL carries the first session's ID
  token as `id_token_hint` (`llmprovider/auth/oauth_plan.go:251`), and that the
  shipped sign-ins do the same.
* **F5.** Phase 1b's amendment says what a run leaves behind and what it
  removes (two grants per run, Kilo's without expiry and not revocable, one
  plan client registered per run, all doubled by a re-run). Whether the probe
  revokes its own grants was the owner's, and is ruled below: it does, after
  the verdict is printed.
* **F7, first half.** `grantRecorder` counts the round trips a step completes
  (`llmprovider/live_checks_test.go:1374`), and a use step's line ends `,
  requests N` (`:1690`). The SDK sends a refresh again, with the same token,
  on a transport error, a 429 or a 5xx, up to three in all; the twin shows
  `requests 3` for a 500 on a refresh, `requests 1` for Kilo's profile read
  and `requests 2` for a refresh that was answered 503 once. What a count above
  one means for the verdict was then the owner's to rule, and is ruled below
  (a use step that made more than one request makes the run `indeterminate`).
* **F8, the twin proved the code against its own assumptions.**
  * Is the stop on a failed opener reachable on the live path? **Yes, by the
    opener's own side effect.** `runBrowserLogin` calls `OpenURL` from a
    goroutine and ignores its error (`llmprovider/auth/oauth_loopback.go:220-222`),
    so the stand-ins' synchronous error proved nothing about it; but
    `grantShow.URL` records the stop and cancels the sign-in's context itself
    (`llmprovider/live_checks_test.go:1616`), not through what it returns, and
    the cancel ends the browser login through its context. Two new twin cases
    show it: a sign-in stand-in that calls the opener from a goroutine,
    ignores the error and blocks on its context, and the shipped
    `auth.LoginBrowserOAuth` itself against the fake issuer (discovery
    answered), each stopped within two seconds with the plain line. For Kilo,
    the opener is called by the probe itself, synchronously. A failed write to
    the terminal is handled the same way (its cancel is `Code`'s own).
  * Survivors of the reviewer's 14 plants and of the first pass's 17 were
    given twin cases, red first by plant (table below): A (`redact.String`
    dropped): `TestEarlierGrantSay_0081`; D (a 5xx body not put back), J (the
    first status kept), K (a transport error keeps the old status): the
    recorder cases of `TestGrantRecorder_0081`; E (the wait unbounded), H and L
    (no cancel on a failed opener or terminal write): the stop cases of
    `TestEarlierGrantStops_0081`; G (`ended` at any status) and M (a 200
    carrying an error read as `kept`): `TestGrantVerdict_0081`; I (auth before
    canceled): `TestGrantKind_0081`.
  * B and C and N lived in the tagged file. B, the scheme and host check,
    is now `grantWebURL` in the untagged file with `TestGrantWebURL_0081`. C,
    the switch, and the provider check are `grantRunTest`, which the live test
    calls with `os.Getenv`, with `TestEarlierGrantRunTest_0081` (the switch
    off skips and builds nothing; a provider that is none of the four fails
    at once with a message that does not repeat it; the verdict decides the
    failure). N, `t.Fail()`, is `grantApply`, which the live test calls with
    its `*testing.T`, with `TestGrantApply_0081` on a stand-in. What stays in
    the tagged file and has no twin: the binding of `os.Getenv`, the real
    sign-ins, the opener and terminal, the call of `grantApply(t, out)`; the
    twin cannot reach them without a live run, and what holds them is the
    compile, vet and the review's read.
* **F9.** Phase 1a's step 5 text, "Grok's discovery document, which comes
  first", is corrected by annotation: a refresh makes no discovery request.
* **F10.** `AGENTS.md`'s sentence now says the probe prints step lines and a
  verdict (a status, an error's kind, an allow-listed code and a request
  count per step), and where its output goes.
* **F11.** "Reads no environment variable but its two" is corrected in
  Phase 1a's step 2 by annotation and in the Phase 1a entry above: the test
  calls `os.Getenv` for those two; the transport reads the proxy variables and
  the opener inherits the environment.
* **F12, a reading taken.** The recorder reads the body of every response of
  400 or more, not only a refused refresh's, as step 6 says; only an
  allow-listed code is kept, and every body is put back.
* **F13, the edge case.** A use that times out at one minute while its
  refresh, detached from the caller, is still running leaves a late reply that
  can overwrite the next step's status, and the plan's second sign-in reads the
  first session's fields without the session's lock. It is not expected (the
  refresh's own bound is under the minute), and the step lines would show it
  as `canceled` for the use and an extra round trip counted in the next
  step's `requests`: the count makes it visible. Recorded in Phase 1a's
  annotation.
* **The owner's four rulings of the same day**, relayed by the coordinator, each
  label quoted as a label; Phase 1a's annotations say what each built and
  Phase 1b's amendment says what the owner is told.
  * "Only `kept` passes (Recommended)": `grantPasses`
    (`llmprovider/live_checks_test.go:1544`), as built, now with its own twin
    test (`TestGrantPasses_0081`) and a plant.
  * "Revoke the probe's own two grants at the end, after the verdict is printed
    (Recommended)": `earlierGrant.finish` (`:1724`) prints the verdict, then
    `revokeAll` (`:1747`) revokes each session obtained with the SDK's
    `auth.RevokeOAuthSession`, on a context detached from the run's and bound
    at twenty seconds. What `RevokeOAuthSession` does for each provider, with
    the lines of its code: `chatgpt`, a JSON request to OpenAI's `/oauth/revoke`
    (`llmprovider/auth/oauth_revoke.go:92`; the run does not send it, see below);
    `plan`, discovery then a form
    request to the discovered revocation endpoint with the issued client id
    (`:144`); `grok`, discovery then a form request (`:123`); `kilo`, no request,
    an error matching `ErrUnsupported` (`:72`). Each is sent through the
    session's own HTTP client (`:53`, `:79`), which the run has set to the
    recorder's. No call of the SDK removes the client a plan sign-in registers,
    and none revokes Kilo's device token; no non-test code was added or
    changed, and both are on Phase 1b's by-hand list. Whether a revocation can
    end anything besides the grant it names is not verified, and Phase 1b says
    so before the run. Twin: `TestEarlierGrantRevocation_0081` (revoked,
    refused, server error, unsupported by `TestEarlierGrantReport_0081/kilo/*`,
    bound expired, the run's bound already passed, a failed first sign-in, a
    stop after the first sign-in).
  * "Add one step: refresh A once more after the control (Recommended)": the
    sixth step `last-after`, and the verdict table in Phase 1b's amendment,
    written before the code; `grantVerdict` (`:1505`) and
    `TestGrantVerdict_0081` hold every row.
  * "The run is `indeterminate`: run it once more (Recommended)":
    `grantVerdict`'s first loop. The revocation requests are kept out of the
    count (Phase 1a's annotation).
  * Readings changed by these, ruled on 2026-11-19: "As built: never resend a
    refused token; the last step runs only after a success (Recommended)"
    (`last-after` only after a successful `first-after`), and "As built now: a
    failed baseline ends the run as `indeterminate` (Recommended)". No code
    changed for either. Readings the owner was told and did not change: the
    four-minute wait, the nine-minute bound and `xdg-open`. The commands now
    carry `-timeout 15m`.
  * The `chatgpt` revoke risk, ruled on 2026-11-19: "Run `chatgpt` with
    revocation switched off; its two grants are listed for removal by hand
    (Recommended)". `grantRevokes` (`llmprovider/live_checks_test.go`, after
    `grantRevokeLimit`) is false for `chatgpt` only, and `revokeAll` makes no
    request for it on any path; its lines read `chatgpt revoke-N: status 0, kind
    by_hand`, a fixed word that is neither `ok`, nor Kilo's `unsupported`, nor
    a failure (`grantByHand`). Phase 1b's table lists the two grants under
    Codex's client id as left, to be removed by hand, and keeps the
    not-verified statement as the reason. The twin runs `chatgpt` on the
    verdict, indeterminate and stop paths and requires zero requests to a
    revocation endpoint and the fixed lines; the revocation twin cases that used
    `chatgpt` as the provider that is revoked now use `plan`, the only provider
    the run revokes after the rulings below.
* **The second review's findings (2026-11-19).** The reviewer's final read
  found the code sound and every ruling built as labelled; its findings, fixed
  in place, nothing committed:
  * **N2, the order of Phase 1b's amendment.** The amendment now opens with
    "Before any command: what a run does to your accounts, and what is not
    known": a table, per provider, of what the run does to the real account
    and what is not known, then the not-verified statement and `grok`'s facts
    (its revocation sends the Grok CLI's own client id,
    `llmprovider/auth/oauth_constants.go:13`, set at
    `oauth_loopback.go:296-301` and sent at `oauth_revoke.go:137` and `:159`:
    the condition that led the owner to switch `chatgpt`'s off; `plan` revokes
    only the client the probe registered). The commands are the amendment's
    last section but one, with a line that says they are run only after the
    rest has been read. The owner then ruled `grok`'s revocation off (below),
    and the table and the by-hand list say so.
  * **N3, the MADR.** A dated amendment under D12 quotes the nine labels and
    says what each changes in D12; no line above it is touched.
  * **N4, a 403 is not a dead grant.** Kilo's profile answering 403 is
    classified an auth failure by the SDK (`llmprovider/api_error.go:225`) and
    is, by 0080-MADR, the profile not able to say. `grantDead`
    (`llmprovider/live_checks_test.go:1527`) takes only a 401 for Kilo, and,
    for the others, only a 400 or 401 with a terminal code. The twin cases:
    `TestEarlierGrantReport_0081/<provider>/forbidden` for all four providers
    (indeterminate), and rows of `TestGrantVerdict_0081` for Kilo's 403, 404,
    500 and 200 auth failures and for a terminal code at 403, 404, 422, 429 and
    502 for the others. Red first by the reviewer's plant P9 and by N4-1 and
    N4-2, which the twin did not notice before.
  * **N5, the recorder read more than the SDK.** The recorder read 64 KiB for
    the code where the SDK reads 2048 bytes (`oauthErrorBodyLimit`,
    `llmprovider/auth/oauth_session.go:32`), so a longer body gave the
    recorder a code and the SDK none. `grantErrorCode`
    (`llmprovider/live_checks_test.go:1402`) now reads `grantCodeLimit`
    (`:1397`, 2048) bytes for the code, and still puts the whole body back.
    The agreement table gains bodies of exactly 2048, 2049, about 3000 and
    70000 bytes, and the recorder test a long body put back whole. Red first:
    plant N5-1 (`grantCodeLimit` back to 64 KiB) fails
    `TestGrantCodeAgreesWithTheSDK_0081` with "the SDK calls the refresh token
    dead: false; the recorder's code "invalid_grant" is terminal: true", the
    reviewer's demonstration.
  * **N6 and N8, the window and the `grok` facts.** In the same amendment: `go
    test exited 0` also shows on a skip or "no tests to run", and `exited 1`
    also when the redirect failed because `tmp/live-0081/` is missing (no file);
    a run that measured has a verdict line in its file. For `grok`: xAI answers a
    reused token as plain `invalid_grant` (`xai-grok-login/src/error.rs:77`), so
    only the count rule guards a `grok` verdict. The notes about a failed `grok`
    revoke line and about a discovery that fails fell away with the ruling that
    switches `grok`'s revocation off; its by-hand entry replaces them.
  * **N1, `grok`'s revocation (ruling of 2026-11-19).** "Switch `grok`'s
    revocation off too; its two grants are listed for removal by hand
    (Recommended)". `grantRevokes` (`llmprovider/live_checks_test.go:1320`) is
    false for `grok` as for `chatgpt`; the run makes no revocation request, and
    no discovery request for one, on any path, and its lines read `grok
    revoke-N: status 0, kind by_hand`. Only `plan` is revoked by the run. The
    twin runs `chatgpt` and `grok` on the verdict, indeterminate, stop and
    bound-passed paths, requiring zero revocation requests and zero discovery
    requests (`TestEarlierGrantRevocation_0081/<provider>'s_grants_are_not_revoked,_*`);
    the revocation cases that used `grok` move to `plan`. Plants: R6-1 (the
    `grok` switch removed), R6-2 (`plan` switched off by mistake), R6-3 (a `grok`
    line reading `ok`), R6-4 (the `chatgpt` switch removed).
  * **N7, a baseline of more than one request (ruling of 2026-11-19).** "Stop
    there: end the run as `indeterminate` before the second sign-in
    (Recommended)". `earlierGrant.run` ends the run after the baseline when its
    count is above one, on the path a failed baseline takes
    (`llmprovider/live_checks_test.go:1698`, the first grant revoked where the
    revocation is on, a `by_hand` line otherwise, the baseline's line printed
    with its count). The row is in Phase 1b's verdict table and in
    `grantVerdict`'s doc; `TestGrantVerdict_0081` has a row, and
    `TestEarlierGrantReport_0081/<provider>/a_baseline_that_needed_two_requests_ends_the_run_before_the_second_sign-in`
    the end-to-end case. Plants: R7-1 (the run goes on to the second
    sign-in), R7-2 (a baseline of three requests is not read).

* **The plants** (`tools/plant.py` on a scratch clone, under a sandbox that denies
  every outbound connection but loopback, `LLMPROVIDER_LIVE_0081` and
  `LLMPROVIDER_LIVE_0081_PROVIDER` unset; each by a test failure, none by a build
  failure; the spec and output are kept outside the tree). The reviewer's 14
  (A to N), the 17 of the first pass re-expressed for the code as it now
  reads, one or more for each fix and each ruling, and the reviewer's 15 of the second review (the survivor among them, a Kilo 403 taken as dead, is caught since N4). `plant: all 88 plant(s) caught`.
  Survivors: none among them. What no twin reaches, and what holds it
  instead: in the live test, the binding of `os.Getenv` into `grantRunTest`,
  the call of `grantApply(t, out)`, the real sign-ins, the real opener and
  terminal (the compile, the vet, and the read of the file; a plant of the
  binding or the call is not caught, because the test cannot be run without a
  person). A plant of the tagged file could not be run by the twin's build,
  which does not compile it.

| Plant | A test it failed |
| :--- | :--- |
| A: redact.String dropped from the output line | `TestEarlierGrantSay_0081` |
| B: the web-URL check removed | `TestGrantWebURL_0081` |
| C: the switch made never true | `TestEarlierGrantRunTest_0081/the_switch_is_off` |
| D: a 5xx body not put back | `TestGrantRecorder_0081/a_refusal's_code_is_kept_and_its_body_put_back` |
| E: the sign-in wait unbounded | `TestEarlierGrantStops_0081/the_wait_for_the_person_is_bounded` |
| F: the baseline's rotation not carried | `TestEarlierGrantReport_0081/chatgpt/kept` |
| G: ended accepted at any status | `TestEarlierGrantReport_0081/chatgpt/forbidden` |
| H: no cancel on a browser failure | `TestEarlierGrantStops_0081/a_browser_sign-in_that_ignores_the_opener's_error_is_stopped_by_the_opener` |
| I: auth checked before canceled | `TestGrantKind_0081` |
| J: the first status kept, not the last | `TestEarlierGrantReport_0081/chatgpt/a_baseline_that_needed_two_requests_ends_the_run_before_the_second_sign-in` |
| K: a transport error keeps the old status | `TestEarlierGrantRevocation_0081/a_revocation_that_outlasts_its_bound_is_cut` |
| L: no cancel on a terminal write failure | `TestEarlierGrantStops_0081/the_terminal_device_fails_to_take_the_code` |
| M: a 200 carrying an error read as kept | `TestGrantVerdict_0081` |
| N: the live test's t.Fail() removed | `TestGrantApply_0081` |
| N2: a stop's message is not fatal | `TestGrantApply_0081` |
| N3: the switch off is not a skip | `TestGrantApply_0081` |
| 1: the step line is given the error's text | `TestEarlierGrantReport_0081/chatgpt/kept` |
| 2: the recorder keeps a code outside the allow-list | `TestEarlierGrantReport_0081/chatgpt/control_fails` |
| 3: the verdict is computed without the control | `TestEarlierGrantReport_0081/kilo/control_fails` |
| 4: the verdict is computed without the baseline | `TestGrantVerdict_0081` |
| 5: the user code is logged to the output as well as the terminal | `TestEarlierGrantReport_0081/kilo/kept` |
| 6: the opener's failure falls back to logging the URL | `TestEarlierGrantRevocation_0081/a_stop_after_the_first_sign-in_still_revokes_it,_after_no_verdict` |
| 7: a kept verdict is reported as ended | `TestEarlierGrantReport_0081/chatgpt/kept` |
| 8: an ended verdict (the first later use dead) is reported as indeterminate | `TestEarlierGrantReport_0081/chatgpt/ended` |
| 8b: the lazy ending is reported as indeterminate | `TestEarlierGrantReport_0081/chatgpt/lazy` |
| 9: an indeterminate verdict is reported as kept | `TestEarlierGrantReport_0081/chatgpt/forbidden` |
| 10: refresh_token_reused is read as an ended grant | `TestEarlierGrantReport_0081/chatgpt/reused` |
| 11: Kilo's ended grant is not its 401 | `TestEarlierGrantReport_0081/kilo/ended` |
| 12: the plan's second sign-in is made without Previous | `TestEarlierGrantReport_0081/plan/kept` |
| 13: the recorder reads a successful response's body | `TestGrantRecorder_0081/a_success's_body_is_not_read` |
| 14: the recorder does not put a refusal's body back | `TestEarlierGrantReport_0081/chatgpt/ended` |
| 15: the recorder reads the request's body | `TestGrantRecorder_0081/a_refusal's_code_is_kept_and_its_body_put_back` |
| 16: the terminal device is not checked before the first sign-in | `TestEarlierGrantReport_0081/kilo/kept` |
| 17: the browser opener is not checked before the first sign-in | `TestEarlierGrantStops_0081/there_is_no_browser_opener` |
| F1a: the code is read case-sensitively | `TestEarlierGrantReport_0081/the_shape:_an_upper-case_code` |
| F1b: the nested shape is not read | `TestEarlierGrantReport_0081/the_shape:_nested_invalid_grant` |
| F1c: the bare code is not read | `TestEarlierGrantReport_0081/the_shape:_a_bare_code` |
| F1d: refresh_token_invalidated is not terminal | `TestEarlierGrantReport_0081/the_shape:_nested_refresh_token_invalidated` |
| F7a: a transport error is not counted | `TestGrantRecorderSteps_0081/a_transport_error_forgets_the_status_of_the_response_before_it` |
| F7b: a step does not begin from a zero count | `TestEarlierGrantReport_0081/chatgpt/kept` |
| F7c: the line does not carry the count | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R1-1: grantPasses passes every verdict that is not empty | `TestEarlierGrantRunTest_0081/the_verdict_of_the_ended_run` |
| R1-2: the run fails nothing | `TestEarlierGrantRunTest_0081/the_verdict_of_the_ended_run` |
| R1-3: the provider's value is repeated in the fatal message | `TestEarlierGrantRunTest_0081/the_provider_is_none_of_the_four` |
| R1-4: the probe is built before the provider is checked | `TestEarlierGrantRunTest_0081` |
| R2-1: the revocation runs before the verdict is printed | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R2-2: a failed revocation changes the verdict | `TestEarlierGrantRevocation_0081/a_refused_revocation_is_told,_and_changes_nothing` |
| R2-3: a revocation's error text reaches the output | `TestEarlierGrantReport_0081/plan/kept` |
| R2-4: the second grant is skipped | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R2-5: the revocation runs on the run's own context | `TestEarlierGrantRevocation_0081/the_run's_own_bound_having_passed_does_not_stop_the_revocation` |
| R2-6: the revocation is not bound | `TestEarlierGrantRevocation_0081/a_revocation_that_outlasts_its_bound_is_cut` |
| R2-7: the revocation's step does not begin from nothing | `TestEarlierGrantReport_0081/kilo/kept` |
| R2-8: a session obtained is not remembered | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R2-9: a session obtained before a stop is not revoked | `TestEarlierGrantRevocation_0081/a_grant_obtained_as_the_run_is_stopped_is_still_revoked` |
| R5-1: the chatgpt switch is removed | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R5-2: the chatgpt switch is applied to the wrong provider | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R5-3: the chatgpt line reads ok | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R5-4: the chatgpt line reads unsupported | `TestEarlierGrantReport_0081/chatgpt/kept` |
| N4-1: kilo's 403 accepted as a dead grant | `TestEarlierGrantReport_0081/kilo/forbidden` |
| N4-2: a terminal code at 403 accepted as a dead grant | `TestEarlierGrantReport_0081/chatgpt/forbidden` |
| N5-1: the recorder reads 64 KiB for the code | `TestGrantCodeAgreesWithTheSDK_0081/a_body_of_exactly_2048_bytes` |
| N5-2: the recorder reads 16 bytes for the code | `TestEarlierGrantReport_0081/chatgpt/ended` |
| RV-P2: baseline's count ignored | `TestGrantVerdict_0081` |
| RV-P3: control's count ignored | `TestGrantVerdict_0081` |
| RV-P4: the count threshold is two | `TestGrantVerdict_0081` |
| RV-P5: kilo left by hand | `TestEarlierGrantReport_0081/kilo/kept` |
| RV-P7: Previous for every provider | `TestEarlierGrantReport_0081/chatgpt/kept` |
| RV-P11: the revoke index is wrong | `TestEarlierGrantReport_0081/plan/kept` |
| RV-P12: chatgpt revoked but said by hand | `TestEarlierGrantReport_0081/chatgpt/kept` |
| RV-P13: last-after uses the second session | `TestEarlierGrantReport_0081/chatgpt/lazy` |
| RV-P15: a failed second sign-in's first grant is not revoked | `TestEarlierGrantRevocation_0081/the_run's_own_bound_having_passed_does_not_stop_the_revocation` |
| RV-P30: the revocation order is reversed | `TestEarlierGrantReport_0081/plan/kept` |
| RV-P31: the by-hand line carries the last status | `TestEarlierGrantReport_0081/chatgpt/kept` |
| RV-P32: revocation before the verdict on a stop only | `TestEarlierGrantRevocation_0081/a_stop_after_the_first_sign-in_still_revokes_it,_after_no_verdict` |
| R6-1: the grok switch is removed | `TestEarlierGrantReport_0081/grok/kept` |
| R6-2: plan is switched off by mistake | `TestEarlierGrantReport_0081/plan/kept` |
| R6-3: the grok line reads ok | `TestEarlierGrantReport_0081/grok/kept` |
| R6-4: the chatgpt switch is removed | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R7-1: the run goes on to the second sign-in after a baseline of two requests | `TestEarlierGrantReport_0081/chatgpt/a_baseline_that_needed_two_requests_ends_the_run_before_the_second_sign-in` |
| R7-2: a baseline of three requests is not read | `TestEarlierGrantReport_0081/chatgpt/a_baseline_that_needed_two_requests_ends_the_run_before_the_second_sign-in` |
| R3-1: the second later use is not made | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R3-2: the second later use is made after a failed first | `TestEarlierGrantReport_0081/chatgpt/ended` |
| R3-3: the second later use is made before the control | `TestEarlierGrantReport_0081/chatgpt/kept` |
| R3-4: a kept verdict does not need the second later use | `TestEarlierGrantReport_0081/chatgpt/lazy` |
| R4-1: a request count above one is not read | `TestGrantVerdict_0081` |
| R4-2: the second later use's count is not read | `TestGrantVerdict_0081` |
| RV-P14: a failed baseline's grant is not revoked | `TestEarlierGrantReport_0081/chatgpt/baseline_fails` |
| R4-3: a failed baseline does not end the run | `TestEarlierGrantReport_0081/chatgpt/baseline_fails` |

* **Gates** (each exit status taken before any filter; heavy runs through
  `run-heavy`, or `run-on-host --race`, never a WSL distro):

  | Gate | Where | Exit |
  | :--- | :--- | :--- |
  | `make pre-add-check FILES="llmprovider/live_checks_test.go llmprovider/live_twins_test.go llmprovider/live_0081_grant_test.go"` (gofmt, lint for the host and `GOOS=windows`, vet, test, govulncheck) | session machine, `run-heavy` | 0 |
  | `go vet ./...`, `GOOS=windows go vet ./...`, `go vet -tags live_gateways ./...` | session machine | 0, 0, 0 |
  | `go test -tags live_gateways -c -o /dev/null ./llmprovider/` (compiled, never run) | session machine | 0 |
  | `make api-check`; `python3 -B tools/apidiff.py v1.31.0` | session machine | 0; 0, the one addition listed is `llmprovider/x/signin.VerifyKilo` (0080), none from this change |
  | `go test -race -count=20 -timeout 30m ./llmprovider/` | session machine (darwin/arm64), `run-heavy` | 0 |
  | `go test -race -count=1 -timeout 45m ./...` | `run-on-host --race`: `pick-host` chose the Linux machine, linux/amd64 native (not a WSL distro) | 0, 43 packages ok, none failing |
  | the 88 plants | scratch clone, no-network sandbox | 88 of 88 caught |
  | the twin tests of the probe, under the sandbox | session machine | 0 |

### Phase 1c (2026-11-20): the owner's four runs, recorded

Written by an agent, from the four files the owner's runs of 2026-11-19
produced, `tmp/live-0081/chatgpt.txt`, `plan.txt`, `grok.txt` and `kilo.txt`
(git-ignored: `git check-ignore -v` names `.gitignore:15`, `tmp/`), which the
coordinator had verified the same day. Before any line was shown, each line
of each file was scanned for anything shaped like a token, a code, a client
id or a URL with a query; the only hit in each file was the test's name. No
agent ran the probe, signed in or held a session. Each file holds one run of
13 lines. Every line is quoted below as the file holds it, except that the
module path in each file's last line is written `<module>` (an account name
is not written into a record).

`chatgpt` (the ChatGPT browser sign-in):

```text
=== RUN   TestLive_EarlierGrant_0081
    live_checks_test.go:1584: chatgpt sign-in-1: ok
    live_checks_test.go:1584: chatgpt baseline: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: chatgpt sign-in-2: ok
    live_checks_test.go:1584: chatgpt first-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: chatgpt control: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: chatgpt last-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: chatgpt: kept
    live_checks_test.go:1584: chatgpt revoke-1: status 0, kind by_hand
    live_checks_test.go:1584: chatgpt revoke-2: status 0, kind by_hand
--- PASS: TestLive_EarlierGrant_0081 (20.54s)
PASS
ok   <module>/llmprovider 21.246s
```

`plan` (the ChatGPT plan sign-in):

```text
=== RUN   TestLive_EarlierGrant_0081
    live_checks_test.go:1584: plan sign-in-1: ok
    live_checks_test.go:1584: plan baseline: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: plan sign-in-2: ok
    live_checks_test.go:1584: plan first-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: plan control: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: plan last-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: plan: kept
    live_checks_test.go:1584: plan revoke-1: status 200, kind ok
    live_checks_test.go:1584: plan revoke-2: status 200, kind ok
--- PASS: TestLive_EarlierGrant_0081 (29.80s)
PASS
ok   <module>/llmprovider 30.073s
```

`grok` (Grok's browser sign-in):

```text
=== RUN   TestLive_EarlierGrant_0081
    live_checks_test.go:1584: grok sign-in-1: ok
    live_checks_test.go:1584: grok baseline: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: grok sign-in-2: ok
    live_checks_test.go:1584: grok first-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: grok control: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: grok last-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: grok: kept
    live_checks_test.go:1584: grok revoke-1: status 0, kind by_hand
    live_checks_test.go:1584: grok revoke-2: status 0, kind by_hand
--- PASS: TestLive_EarlierGrant_0081 (5.14s)
PASS
ok   <module>/llmprovider 5.411s
```

`kilo` (Kilo's device sign-in):

```text
=== RUN   TestLive_EarlierGrant_0081
    live_checks_test.go:1584: kilo sign-in-1: ok
    live_checks_test.go:1584: kilo baseline: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: kilo sign-in-2: ok
    live_checks_test.go:1584: kilo first-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: kilo control: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: kilo last-after: status 200, kind ok, code -, requests 1
    live_checks_test.go:1584: kilo: kept
    live_checks_test.go:1584: kilo revoke-1: status 0, kind unsupported
    live_checks_test.go:1584: kilo revoke-2: status 0, kind unsupported
--- PASS: TestLive_EarlierGrant_0081 (17.55s)
PASS
ok   <module>/llmprovider 17.825s
```

| Provider | Verdict | The four uses | Revocation | Left on the owner's account, to remove by hand | Removed |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `chatgpt` | `kept` | each 200, `requests 1` | `by_hand` twice: none sent, by the owner's ruling | two grants under Codex's client id (Phase 1b's table: not the owner's own Codex login) | not yet: the owner's word is awaited |
| `plan` | `kept` | each 200, `requests 1` | `ok`, status 200, twice: both grants revoked | the client the first sign-in registered for this app's name (Phase 1b's table: no call of the SDK removes it; the files carry no line for it) | not yet: the owner's word is awaited |
| `grok` | `kept` | each 200, `requests 1` | `by_hand` twice: none sent, by the owner's ruling | two grants under the Grok CLI's client id (not the owner's own Grok CLI login) | not yet: the owner's word is awaited |
| `kilo` | `kept` | each 200, `requests 1` | `unsupported` twice: none sent | two device tokens, which do not expire | not yet: the owner's word is awaited |

* **Step 1** (every line quoted, per provider; the grants left by hand): above.
* **Step 2** (D12, per provider; "Not verified" loses N1 for the four): the
  MADR's amendment of 2026-11-20, "D12: the four verdicts", and its "Not
  verified" annotated in place. N1 stays unverified for the OpenAI and Grok
  device sign-ins and for an ending minutes after a sign-in.
* **Step 3** (commit the records' edits): not done by this pass, which stages
  and commits nothing; it is the coordinator's.
* **Acceptance:** met, its first branch: four verdicts recorded, each `kept`.
  No count above one, so the count rule made nothing `indeterminate`; no
  `refresh_token_reused`; every provider run.
* **Not in the files, so not recorded:** the commit the runs were made at
  (Phase 1b's step 1 prints it to the window, which is not sent), and whether
  an earlier run of a provider was made and its file overwritten.

**What `kept` means for Phase 2, by this PLAN's own text.**

* Phase 1: "Phase 2's code does not land before Phase 1c is recorded and no
  stop condition below is met." Phase 1c is recorded and no stop condition is
  met.
* Phase 1b, "What each outcome decides": the branch for this outcome is
  "`kept` for all four | the premise of 0059-MADR Q2 (a) and of saving last
  holds for those sign-ins; Phase 2 proceeds". The branches that no longer
  apply, `ended` for any one, `indeterminate` twice and a provider not run,
  are annotated as such under the table (2026-11-20), as are the first two
  stop conditions of "Rules for every phase".
* Phase 2, the annotation of 2026-11-18 to step 4: "none starts before Phase
  1c". Met.
* No step of Phase 2 or 3 branches on the verdict beyond "Phase 2 proceeds",
  so none changes. What the PLAN does not say is whether the docs of Phase 2's
  step 5 state the measured premise. That was the MADR state table's row S62,
  put to the owner and since ruled (step 5's addition of 2026-11-20).

### Readiness pass (2026-11-20): the test map, the plants and the re-grounding

Records only, at `24756a6`: the MADR's amendment of 2026-11-20 holds the
state table (66 rows: 52 decided, 11 out of scope, 3 open). Nothing was run
but reads, `git` reads, `tools/claims.py` and two scratch scripts that read
`git show`; no test, gate, build or live run. Nothing was staged.

**Each decided row, and its test** (phase 2 unless the row says;
"existing" is in the tree at `24756a6`, "guard" is green at `a7e924f`, "red"
is red at the base, "P-n" is proposed below):

| Rows | Test | Kind |
| :--- | :--- | :--- |
| S1 | `TestConfigure_EndedContextAsksNothing_0059` | existing |
| S2 | `TestResolveCredential_ErrorKinds_0056/ctx` | existing |
| S3, S4, S5, S6, S39 | P-1 `TestCancelBeforeIssueSavesNothing_0081` | proposed |
| S7, S31 | `TestCancelAfterIssueSavesNothing_0081`, words; P-6's key cases (S31) | guard; proposed |
| S8, S10, S73 | `TestCancelDuringKiloProfileSavesNothing_0081`, words, with P-8's cases (S10, S73) | guard; proposed |
| S9, S69, S75 | `TestKiloVerify_EndedContext_0080`; `TestKiloVerify_OutagesWarn_0080` (S75, its words changed by S76) | existing |
| S11, S28 | `TestCancelDuringListingSavesNothing_0081`, words, the plan's cases required; `TestCancelDuringDeviceFlowSavesNothing_0081`, required | guard; proposed |
| S12 | `TestCancelDuringRefreshSavesNothing_0081`, extended (the issuer holds the refresh; the error within one second of the cancel) | guard |
| S13 | `TestCancelledConfigureReturnsNoKey_0081`; P-9, words | red; proposed |
| S14, S16 | `TestCancelAtModelStageSavesNothing_0081`; P-3, words per case, and its deadline case | red; proposed |
| S15 | `TestCancelAtManualModelSavesNothing_0081`, words | red |
| S17 | `TestModelStageErrorSavesNothing_0081`; P-4 | red; proposed |
| S18 | P-18 `TestCancelBetweenTheCheckAndTheWriteSaves_0081` | proposed |
| S19, S20 | `TestCancelAfterSaveReturnsTheResult_0081`; P-11 | red; proposed |
| S21 | `TestConfigureSavesOnceAfterTheModel_0081`; `TestResolveSavesInTheStep_0081` | red; guard |
| S22 | P-5 `TestSessionDueAtTheCommit_0081`, three cases | proposed |
| S23 | `TestRotationDuringQuestionsSavedAtEnd_0081` | red |
| S24 | `TestCancelDuringListingReturnsNoKey_0081`; `TestCancelledConfigureReturnsNoKey_0081`; `TestCancelAtModelStageReturnsNoKey_0081`; `TestResolveCredential_ErrorKinds_0056/ctx_ended_during_the_listing` | guard; red; red; existing |
| S25, S26, S27 | P-6 `TestCancelledRunReturnsNoCredential_0081`, words | proposed |
| S29, S30 | P-7 `TestCancelWithKeptSessionReturnsNoResult_0081`, words | proposed |
| S32 | P-9, a Driver case of `TestCancelAfterIssueSavesNothing_0081` | proposed |
| S34 | `TestSaveTakesTheRefreshLock_0082`; `TestSaveWithoutALockerSaves_0082` | existing |
| S35 | P-10 `TestConfigureWaitsForTheLockAfterTheModel_0081` | proposed |
| S36 | `TestSaveCancelledWhileWaitingSavesNothing_0082`; `TestSaveCancelledAtAnyWaitSavesNothing_0082` | existing |
| S37 | `TestDeadlineDuringListingSavesNothing_0081` (a session), words; P-14 (a key) | guard; proposed |
| S38, S71 | P-13 `TestListingLimitIsAnOutageAndSaves_0081`; `TestListing_StaticBesideNoticed_0066` (S71) | proposed; existing |
| S40 | the context's tests above | — |
| S41 | P-16 `TestKilledAtTheModelQuestionKeepsTheStore_0081` | proposed |
| S42 | `TestDiscoverLimit_0080` | existing |
| S43, S44 | the save-counting tests above | as above |
| S45 | `TestLive_EarlierGrant_0081` (Phase 1b, the owner's) and its twins (`TestEarlierGrantReport_0081` and the others of Phase 1a) | done |
| S48 | P-15 `TestAbandonedPlanRunKeepsTheClientRecord_0081` | proposed |
| S49 | `TestSaveWaitsForARefreshInFlightInOneProcess_0082` | existing |
| S50 | P-19 `TestLastCommitWins_0081` | proposed |
| S51 | 0082's tests in `llmprovider/auth/` (`held_session_0082_test.go` and its siblings) | existing |
| S52, S53 | P-12 `TestStoreErrorAtTheCommit_0081` | proposed |
| S56 | `TestLockFileThatCanNeverBeOpenedAsksNeutrally_0082` | existing |
| S60, S62 | P-20 `TestSaveNotesInTheArchitecture_0081` | proposed |
| S61, S72 | P-17 `TestNoListingWarningOnAnEndedRun_0081` | proposed |
| S63 | Phase 3's two plants on `write`'s check; `TestSaveCancelledAtAnyWaitSavesNothing_0082` | existing, to be seen caught |
| S65 | owed by 0083-PLAN (the question does not exist at this record's base) | — |
| S67, S68 | P-21 `TestKiloUnverifiedNoticeClaimsNoSave_0081` | proposed |
| S70, S74 | `TestKiloVerify_NoDiscoverNoRead_0080` | existing |

**Revised in the second pass (2026-11-20)**, before anything was committed,
after the independent review and the owner's rulings (the MADR's "Second
pass"): the map above, P-3, P-5 to P-9, the plants' table, and P-17 to P-21.
"Words" means the test asserts the error's words of the first check that runs
after its cancel: `wizard: <ctx>` from N3's check (`ConfigureLLM`) or
`commitCredential`'s (`ResolveCredential`), compared as the whole error text,
since `write`'s error also begins with `wizard:`; `select model: <ctx>` or `select
fallbacks: <ctx>` from a model loop's top; "the new <label> session was not
saved: <ctx>" from `save` or `write`. Without it a plant on one check is
hidden by the next one. The plant of each row is in its Test cell in the
MADR's state table.

**PLAN tests that map to no row:** none. Each of Phase 2's sixteen maps
above, and the probe's tests map to S45. **S11's tests may not be dropped**
(the coordinator's instruction of 2026-11-20, from the review): the drop
allowed by Phase 2's table row for `TestCancelDuringDeviceFlowSavesNothing_0081`
and by the deviation entry of 2026-11-18 ("Owed in Phase 2") no longer
applies, to it or to the plan's cases of `TestCancelDuringListingSavesNothing_0081`;
a case that cannot be served in process without changing non-test code is a
stop and a deviation put to the owner.

**Proposed tests** (a dated addition to Phase 2; P-2 is not used, since S2
has an existing test). Each is in `wizard/`, against in-process stand-ins
(0065-MADR), lists through `testCache(t)` with model metadata off, runs both
entry points where the row names both, and is seen to fail on its plant with
`tools/plant.py` on a scratch clone, never in the tree. A plant is the single
production change that must turn the test red.

* **P-1 `TestCancelBeforeIssueSavesNothing_0081`** (S3 to S6, S39). Cases: a
  browser sign-in whose `loginBrowserOAuth` seam blocks until its context
  ends; a device sign-in whose `deviceLogin` seam does the same; the plan's
  consent question answered after the cancel; the exchange cut off, through
  the seam calling the shipped `auth.LoginBrowserOAuth` with `NoLoopback` and
  the code pasted, its token endpoint served in process through the run's
  `HTTPClient` and held until the context ends; the login's own bound, the run
  alive (the seam returns "oauth: browser login timed out" wrapping
  `context.DeadlineExceeded`). Asserts: the login's error, matching the
  context's where the context ended; no Result; no save; no client record.
  Plant: `signIn`'s browser branch returns `errors.New("wizard: sign-in
  failed")` in place of the login's error.
* **P-3, added to `TestCancelAtModelStageSavesNothing_0081`** (S14, S16). Each
  case asserts the words of the first check that runs after its cancel, by D9
  row 4: `wizard: context canceled` where the question answered was the last
  (the final check); `select model: context canceled` where a query answered
  leads to the matches' loop (`wizard/model_select.go:104`); `select
  fallbacks: context canceled` where a model chosen with `NeedFallbacks` leads
  to the fallbacks' loop. A case where the run's deadline passes while the
  model menu is on screen, answered, asserts `context.DeadlineExceeded` and
  `wizard: context deadline exceeded`. Plants: `commitCredential`'s check
  removed (the final-check cases: the words become `write`'s);
  `commitCredential` matching `context.Canceled` only (the deadline case).
* **P-4, added to `TestModelStageErrorSavesNothing_0081`** (S17). Cases: a
  blank model id in the empty-catalog branch ("no model available ... and none
  entered"); a `Prompter`'s error at the fallbacks. Asserts that error and no
  save. Plants: `commitCredential` called before the blank check in the
  model-id branch; `commitCredential` called before `selectFallbacks`.
* **P-5 `TestSessionDueAtTheCommit_0081`** (S22). `ConfigureLLM`, a Grok
  device session (the seam) made due while the model question is on screen
  (the `Prompter` calls `Invalidate`), its issuer served in process. Cases: the
  refresh answered 200 (one save, of the rotated refresh token, and the
  Result); answered 400 `invalid_grant` (an error matching
  `llmprovider.ErrAuthFailure`, no save); answered 503 on each of the SDK's
  three tries, the context alive (one save, of the session as it was, its
  refresh token unspent, and the Result). Plants: `save`'s first check
  (`wizard/auth.go:1107`) removed (the refusal case); that check refusing any
  error (the 503 case).
* **P-6 `TestCancelledRunReturnsNoCredential_0081`** (S25 to S27, S31's key).
  Cases: an environment key (`LookupEnv`), a saved key (`Existing`), Kilo's
  device key (`KiloDeviceKey`, the seam), a vendor CLI read-through (a file in
  `t.TempDir()`), no credential (a keyless provider), a Claude key with the
  cancel at the workspace question, a typed key without `Discover`. The
  context ends before the commit. Asserts: an error matching the context's, in
  the words `wizard: context canceled`; a zero Result; for `ConfigureLLM`, no
  model question. Plants: `commitCredential` checks only when the credential is
  a session or a typed key (caught by the `ResolveCredential` cases); N3's check
  removed (caught by the `ConfigureLLM` cases through the words only: without
  N3 the model loop's top refuses first, as `select model: context canceled`,
  and asks nothing either way).
* **P-7 `TestCancelWithKeptSessionReturnsNoResult_0081`** (S29, S30). Cases: a
  kept session, the context ended during the listing (the context's error, no
  Result, no save by the wizard); the same with the kept session's refresh
  completing during the listing (one save, through its own store, of the
  rotated pair, and none by the wizard); a kept session refused, a new
  sign-in, a cancel while the last model question is on screen, answered (the
  store holds the kept session; the words `wizard: context canceled`). Plants:
  `commitCredential` returns before its check for a kept credential (caught by
  the `ResolveCredential` case, which then returns the kept Result);
  `commitCredential`'s check removed (caught by the refused-kept case through
  `write`'s words); N3's check removed (caught by the `ConfigureLLM` case of the
  first, through the words).
* **P-8, cases added to `TestCancelDuringKiloProfileSavesNothing_0081`**
  (S8, S10, S73): every case asserts the words; the organization menu answered
  after the cancel, through both entry points; without `Discover`, the run
  cancelled and, in another case, its deadline passing during the profile read:
  no organizations warning. Plants: `commitCredential`'s check removed (the
  `ResolveCredential` cases: `write`'s words); N3's check removed (the
  `ConfigureLLM` cases: `select model:`); `resolveKiloDevice` keeping its
  suppression under `Discover` only (S73: the warning shown); the suppression
  matching `context.Canceled` only (S73's deadline case).
* **P-9, cases added to `TestCancelAfterIssueSavesNothing_0081`** (S13 for a
  session, S32): `ConfigureLLM` asks no model question and returns the words
  `wizard: context canceled`; a case with `Options.Driver`, through both entry
  points. Plants: N3's check removed (caught by the words: the model loop's top
  then answers `select model: context canceled`, still asking nothing);
  `commitCredential`'s check removed (the Driver case through
  `ResolveCredential`).
* **P-10 `TestConfigureWaitsForTheLockAfterTheModel_0081`** (S35). A scripted
  locker held, as 0082's tests script one, and `NeedFallbacks`. Asserts the
  order: the model and fallback questions, then the waiting notice, then, at
  the bound, the retry question; declined, the store's error, no save, no
  Result. Plant: `ConfigureLLM` saving in the step again.
* **P-11 `TestConfigureReturnsTheResultWhenCancelledInTheWrite_0081`** (S19,
  S20, `ConfigureLLM`): a store whose `Save` writes, then cancels the run;
  asserts the Result and one save, through the end and through the model-id
  branch. Plant: a context check added after the commit in `ConfigureLLM`.
* **P-12 `TestStoreErrorAtTheCommit_0081`** (S52, S53), `ConfigureLLM` after
  the model questions and `ResolveCredential`: a store whose `Save` fails before
  writing (its error, no Result, the store as it was); one that writes, then
  fails (its error, the new session in the store). Plant: `write` ignores the
  store's error.
* **P-13 `TestListingLimitIsAnOutageAndSaves_0081`** (S38): `DiscoverLimit` of
  50 ms, a listing transport that blocks until its request's context ends, the
  run's context alive; asserts a warning, one save (in the step, or at the end)
  and the Result. Plant: `discoverModels` returns a listing ended by its own
  limit as a `*checkEnded`.
* **P-14 `TestDeadlineReturnsNoKey_0081`** (S37, a key): the run's deadline
  passes during the listing of a typed key; `ResolveCredential` returns an
  error matching `context.DeadlineExceeded` and no key; `ConfigureLLM` asks no
  model. Plant: `commitCredential` matches `context.Canceled` only.
* **P-15 `TestAbandonedPlanRunKeepsTheClientRecord_0081`** (S48), a
  `FileTokenStore` in `t.TempDir()`. Cases: a first plan sign-in (the seam, its
  consent scripted) abandoned at the model question: no session saved, the
  record naming the issued client; an undecodable record and a stored plan
  session of client X: one notice, a new client Y, the run abandoned, the store
  holding X and the record Y, then a second run's plan sign-in made with no
  `Previous` and the consent asked. Plant: `signsInAgainTo` ignores the record.
* **P-16 `TestKilledAtTheModelQuestionKeepsTheStore_0081`** (S41). The test
  re-runs its own binary as a helper, told apart without a new environment
  variable, or with one named as `scripts/check_tables.py` requires; the helper
  runs `ConfigureLLM` with a `FileTokenStore` in a temporary directory that
  holds an earlier session, a stand-in sign-in, and a `Prompter` that signals
  at the model question and blocks; the test kills it (`Process.Kill`, on every
  platform) and asserts the file holds the earlier session. Plant:
  `ConfigureLLM` saving in the step again.

* **P-17 `TestNoListingWarningOnAnEndedRun_0081`** (S61, S72, S77), `Discover`,
  the listing cache cold, both entry points; corrected in the fourth pass of
  2026-11-20 (a cancel during a cold-cache `catalog.List` of a provider with a
  static catalog reaches `wizard/configure.go:974`, not `:951` or `:967`, and a
  stale listing is live and warns nothing). One case per warning site, each
  with the run cancelled during the listing and, in a second case, its
  deadline passing there; each asserts that site's warning is not shown, the
  words `wizard: <ctx>` and nothing saved:
  * `:948`: a ChatGPT session (OpenAI, no static catalog), its listing's
    transport returning when its request's context ends;
  * `:951`: a third party's descriptor with a static catalog that `catalog`
    does not list (`ErrUnsupported`, so `providerCatalog`), its `ListModels`
    returning the context's error (`registry0056` with static ids);
  * `:958`: a third party's `catalog.Lister` stand-in, no static catalog,
    whose `ListCatalog` returns a catalog not live, carrying the context's
    error, with a nil error;
  * `:967`: the same stand-in with a static catalog, returning nothing
    recommended;
  * `:974`: a built-in provider with a static catalog, listed by
    `catalog.List`: the failed listing comes back as the static catalog
    carrying the error, with a nil error
    (`llmprovider/catalog/discovery.go:353-366`, `:556-565`);
  * `:977`: a `catalog.Lister` stand-in returning a catalog not live, with
    models recommended and no error.

  Plants, one per site: that site's warning shown whatever the run's context
  (six); the suppression matching `context.Canceled` only (the deadline
  cases). P-13 holds the other side (S71): with the run alive and the
  listing's own limit passed, the warning is shown; its plant, the
  suppression read from the listing's own context (`dCtx`) instead of the
  run's.
* **P-18 `TestCancelBetweenTheCheckAndTheWriteSaves_0081`** (S18): a store
  whose `Save` cancels the run, then writes; nothing reads the context between
  `write`'s check and `Save` (`wizard/auth.go:1181` to `:1185`), so this is
  the instant. Asserts one save and the Result, through both entry points.
  Plant: a context check added after `Save` in `write`. This replaces the
  first pass's "S18 has no test"; the question put to the owner on it was
  withdrawn by the coordinator.
* **P-19 `TestLastCommitWins_0081`** (S50): two `ConfigureLLM` runs on one
  store; run A signs in and waits at its model question while run B completes
  and saves; A then answers and commits. Asserts the store holds A's session,
  saved twice in all. Plant: the commit skips the save when the store already
  holds a session.
* **P-20 `TestSaveNotesInTheArchitecture_0081`** (S60, S62): reads
  `docs/architecture.md` and requires, under "When a session is saved", the
  sentence that an abandoned Kilo device sign-in leaves a token that does not
  expire and that the person removes in the Kilo account's settings, and the
  sentence on the measured premise naming the two device sign-ins as not
  probed. Plants: each sentence removed, one at a time.
* **P-21 `TestKiloUnverifiedNoticeClaimsNoSave_0081`** (S67, S68): Kilo,
  `Discover`, no organization, the profile answering 503, the run alive; a
  typed key and a device session, through both entry points. Asserts the
  notice is shown in the words of S76, "...; going on without verifying
  it", and holds no
  "saved"; for `ConfigureLLM`, that it comes before the model question. Plant:
  the words back to "it is saved unchecked". The existing
  `TestKiloVerify_OutagesWarn_0080` (`wizard/kilo_verify_0080_test.go:362`,
  `:383`) changes its expected words with it; that change is the one the
  owner's ruling "Fix both" allows, an exception to the stop condition on an
  existing test.

**The plants, re-mapped after 0082** (reasoned from the code at `24756a6`,
not run; Phase 3 runs them). For a session, `write`'s check refuses an ended
context after `commitCredential`'s does, so a plant on `commitCredential`
alone is caught only by a test of a key or one that asserts the error's
words:

| Plant (Phase 3 step 1, the 2026-11-18 annotation, and this pass) | Caught by, at `24756a6` (reasoned) | Not caught by, though named before |
| :--- | :--- | :--- |
| the final check (`commitCredential`'s) removed | `TestCancelAtModelStageReturnsNoKey_0081`; P-3's final-check cases (words); the `ResolveCredential` cases of the guards S7, S8, S11 and of P-7, P-8, P-9 (words) | `TestCancelAtModelStageSavesNothing_0081`'s count assertions, `TestCancelDuringListingSavesNothing_0081` without words (each held by `write`'s check) |
| the check for a session only | `TestCancelAtModelStageReturnsNoKey_0081`; `TestCancelDuringListingReturnsNoKey_0081`'s `ResolveCredential` case; P-6's `ResolveCredential` cases | — |
| a cancel matched, not a deadline | P-14; P-3's deadline case | `TestDeadlineDuringListingSavesNothing_0081` without words (held by `write`'s check) |
| `ConfigureLLM` saving in the step again | `TestConfigureSavesOnceAfterTheModel_0081`; `TestModelStageErrorSavesNothing_0081`; P-10; P-16 | — |
| the model-id branch not committed | `TestCancelAtManualModelSavesNothing_0081` (words) if the plant commits without the check; P-11's model-id case if it returns without committing | — |
| `ResolveCredential`'s own last check put back | `TestCancelAfterSaveReturnsTheResult_0081` | — |
| the check after the step removed (N3) | `TestCancelledConfigureReturnsNoKey_0081` (an empty catalog: a model id asked); the `ConfigureLLM` cases of P-6, P-7, P-8, P-9 and of the guards S7, S11, S37, through the words only | P-6 and P-9 as the first pass credited them, by "no model question" alone: the model loop's top refuses first and asks nothing |
| `write`'s check removed (the 2026-11-18 guards' plant) | after Phase 2, only a cancel after the commit's check: `TestSaveCancelledAtAnyWaitSavesNothing_0082/a_context_ended_while_the_prompt_was_open` (candidate) | the guards S7, S8, S11, S37, where the commit's or N3's check now refuses first |
| `write`'s check moved before the lock's wait | `TestSaveCancelledWhileWaitingSavesNothing_0082`, `TestSaveCancelledAtAnyWaitSavesNothing_0082` (candidates) | — |
| `write`'s check skipped on the retry pass | `TestSaveCancelledAtAnyWaitSavesNothing_0082/a_context_ended_while_the_prompt_was_open` (candidate) | — |
| `save`'s `Token` on `context.WithoutCancel(ctx)` | `TestCancelDuringRefreshSavesNothing_0081`, extended (S12) | the guard as drafted, held by `write`'s check |
| `save`'s first check removed; that check refusing any error | P-5's refusal case; P-5's 503 case | — |
| a context check added after `Save` | P-18; `TestCancelAfterSaveReturnsTheResult_0081`; P-11 | — |
| the listing's warning shown whatever the run's context; the suppression on `context.Canceled` only; the suppression on `dCtx` | P-17; P-17's deadline case; P-13 | — |
| the organizations warning kept under `Discover` only; suppressed whatever the context | P-8 (S73); `TestKiloVerify_NoDiscoverNoRead_0080/device_session` (S74) | — |
| the Kilo notice's words back to "it is saved unchecked" | P-21 | — |
| the commit skipping a save when the store holds a session | P-19 | — |
| a sentence of the architecture note removed | P-20 | — |
| `signsInAgainTo` ignoring the record | P-15 | — |

**Re-grounding at `24756a6`.**

* The code this pair cites is unchanged since `a7e924f` (`git diff --stat
  a7e924f 24756a6`: `AGENTS.md`, this pair, and the probe's three files).
* `python3 -B tools/claims.py` on the MADR and this PLAN, at their markers:
  47 of 47 and 25 of 25, exit 0. The re-pointed rows ("C1 now" to "M2 now",
  "P1 now" to "P25 now") on a scratch copy with plain ids and `self=24756a6`:
  47 of 47 and 25 of 25, exit 0.
* The prose citations of both records and of this PLAN's Phase 1a annotations
  and review fixes (59 lines in `wizard/`, `llmprovider/auth/`,
  `llmprovider/x/signin/`, `llmprovider/`, and 0082-MADR line 1145), read by a
  scratch script against `git show 24756a6`: each holds, but one.
  **Moved:** "Phase 1a, review fixes", F1,
  `TestGrantCodeAgreesWithTheSDK_0081` (`llmprovider/live_twins_test.go:2606`)
  is at `:2619` at `24756a6`, its doc comment at `:2611`; the file was
  committed after F1 was written.
* "Relation to 0080": the tests that call `credentialStep` directly are still
  `TestKiloVerify_EndedContext_0080`, twice
  (`wizard/kilo_verify_ctx_0080_test.go:46`, `:62`). None of 0082's seven
  wizard test files calls `ConfigureLLM` or `credentialStep`.
* Step 4's `commitCredential` needs the `Prompter` (annotated in place).
* The MADR's sentence on the plan's client record after an abandoned run no
  longer holds in one case since 0082 (its amendment of 2026-11-20; S48).
* **Tests to watch, added to "Rules for every phase"'s list:**
  `TestResolveCredential_ErrorKinds_0056/ctx_ended_during_the_listing`, held
  today by `ResolveCredential`'s own last check, which step 4 removes, and
  after it by `commitCredential`'s; and 0082's `wizard/save_*_0082_test.go`,
  `wizard/client_*_0082_test.go`, which call `save` and `ResolveCredential`.
* **The prerequisite "no other agent holds staged or uncommitted work in the
  tree"** does not hold at the time of writing: the readiness passes of 0083,
  0084 and 0085 hold uncommitted edits to their own records. Phase 2 starts
  after they are committed.

### Second readiness pass (2026-11-20): the review and the owner's rulings

Records only, at `24756a6`, before anything of the first pass was committed;
the first pass's entry above was revised in place where this says so. Each
finding of the independent review was verified at its cited lines before it
was applied, and each held. Nothing was run but reads, `git` reads and the
records' checks; nothing staged.

**The review's findings, each applied:**

* Kilo's device token never expires (D6's "lapses"): the owner's ruling, D6
  annotated, S60 decided, step 5's Kilo sentence, P-20.
* A notice that claims a save (`wizard/kilo_verify.go:128`): the owner's
  ruling "Fix both", S67 to S75 decided, S76 open for the words, P-21.
* The commit's check before the lock (`wizard/auth.go:1100-1102`, 0082-MADR
  N-6): decided from the amendment of 2026-11-18; 0082-MADR N-6's dated note;
  step 5 gives `save`'s new last sentence.
* S22's third branch (a refresh failing with a 5xx, a 429 or a transport
  error at the commit): decided by D10; P-5's 503 case and its plant.
* S48 not decided: the owner's ruling, S48 decided, P-15.
* S13, P-6, P-9 (N3's plant hidden by the model loop's top check): words,
  and the plants' table corrected.
* S10, S29, S30 (the commit's plants caught only through
  `ResolveCredential`): P-7 and P-8 name the case that holds each plant.
* S18 needs no seam: P-18; the owner question was withdrawn.
* S12 had no plant: the guard extended, plant `save`'s `Token` detached.
* S11's tests may be dropped: no longer (the map's note).
* P-3's words contradicted D9 row 4: words per case.
* Found in the re-check, same class as the review's: the 2026-11-18 guards
  S7, S8, S11 and S37 lose their plant ("`write`'s check removed") once the
  commit's and N3's checks exist; they now assert the words, with the commit's
  and N3's plants, and the plants' table says where `write`'s plant is still
  caught.

**The re-check of the rows added or changed in this pass** (S7, S8, S10 to
S14, S18, S22, S29, S30, S37, S48, S50, S60 to S62, S67 to S77), against the
review's three questions:

* Missing case: a cancel during the listing with a listing cached within a
  day serves the catalog stale and warns nothing (`llmprovider/catalog/
  discovery.go:347-351`; a cancelled request's error is a `net.Error`, so it
  counts as transient), so P-17 needs a cold cache, as it says; the listing's
  other warnings on an ended run are not named by the ruling (S77, open). No
  other.
* Decided but not decided: S76 (the words) and S77 are `open`; every other
  changed row cites the ruling or the record that decides it.
* A test that cannot pin its row: each changed row names a plant, and each
  plant is set against the entry point where its check runs first (the
  plants' table). S65's test stays owed by 0083's PLAN, since its question
  does not exist at this record's base.

**Put to the owner** (the MADR's open rows, in one prompt):

* **S76, the Kilo notice's new words.** Today: "could not verify the Kilo
  key (<class>[, HTTP <status>]); it is saved unchecked" (0080-MADR D6).
  (a, recommended) "could not verify the Kilo key (...); going on without
  verifying it": it names no save and holds on every path, for a key and a
  session. (b) "could not verify the Kilo key (...); it is used unchecked":
  one word from 0080's, but "used" can read as already used. (c) "could not
  verify the Kilo key (...); it will be saved unchecked if the setup
  completes": keeps the save, but is wrong for a key, which the wizard never
  saves, and is longer. Each costs the same edits: 0080-MADR D6's dated note,
  `wizard/kilo_verify_0080_test.go:362` and `:383`, `CHANGELOG.md` lines 40-41
  and `docs/architecture.md:1148-1149`.
* **S77, the listing's other warnings on an ended run**
  (`wizard/configure.go:948`, `:958`, `:974`, `:977`). (a, recommended)
  Suppress every listing warning once the run's context has ended, one rule:
  four more call sites and P-17's cases. (b) Only the two "using the built-in
  catalog" warnings the ruling names: no more work, but ChatGPT's
  no-static-catalog warning (`:948`), the common one, still shows on a cancel.
  (c) A later record: a deferral, with inconsistent notices until then.

### Third readiness pass (2026-11-20): the owner's rulings on S76 and S77

Records only, at `24756a6`. The owner, relayed by the coordinator, ruled on
the two rows the second pass left open: S76, `"…; going on without verifying it" (Recommended)`;
S77, "Suppress every listing warning once the context has ended (Recommended)". Both rows are decided in
the MADR's state table (77 rows: 67 decided, 10 out of scope, none open).
Phase 2's step 5 addition names the words and every call site; P-17 gains a
case per call site of S77 and P-21 asserts S76's words. Phase 1 carries a
dated note on the owner's ruling on live runs. The second pass's "Put to
the owner" above is answered by these rulings; nothing else changes.

### Fourth readiness pass (2026-11-20): corrections

Records only, at `24756a6`, after an independent confirmation of the third
pass. Each item was verified at its cited lines before it was applied.

* **P-17's sites.** The S61 case of the second pass could not reach
  `wizard/configure.go:951` or `:967`; P-17 now names, per site, what drives
  a test to it, with its own plant (above), and the MADR's S61, S72 and S77
  say the same.
* **Whole error text.** The tests that assert the words compare the whole
  error text: `write`'s error also begins with `wizard:` (S7, S8, S11, S37 and
  the legend).
* **0082-MADR N-6's note** now says `commitCredential`'s is the only check
  at the commit for a key; in `ConfigureLLM` a key also meets N3's check
  (S24, S25).

**Found, not changed: a follow-up for the owner (MADR row S78).** In
`resolveBaseURL`, for Ollama or a local server, an endpoint question
answered after a cancel runs `catalog.ValidateOllamaURLWith` on the ended
context, shows "cannot reach <url> (...)" and asks "Try a different
endpoint?" (`wizard/configure.go:704-718`), through both entry points. The
owner's ruling "Suppress every listing warning once the context has ended
(Recommended)" names listing warnings only, so this record leaves it as it
is; whether it should change is the owner's to decide in a later record.

### Phase 2, build pass 1 (2026-11-20): the tests and the code

Built on `main` at `77c59ca` (the owner committed and pushed the records),
by a build engineer on the coordinator's brief. **The prerequisite holds:**
`git status --porcelain --untracked-files=all` was empty at the start, so no
other agent held staged or uncommitted work in the tree; no other agent worked
in it during the pass. Nothing was staged, committed, stashed or reverted. No
live test, sign-in, built binary or outside host was used; `govulncheck`, which
fetches its database, was skipped (`GO_PRECHECK_SKIP_VULN=1`).

**Step 0, the re-grounding at `77c59ca`.**

* `python3 -B tools/claims.py` on the MADR and this PLAN, at their markers:
  47 of 47 and 25 of 25, exit 0. The re-pointed rows ("C1 now" to "M2 now",
  "P1 now" to "P25 now") on a scratch copy with plain ids and `self=77c59ca`
  (the MADR's other repositories at the revisions of its marker): 47 of 47
  and 25 of 25, exit 0.
* `git diff --stat 24756a6 77c59ca` lists records and `docs/README.md` only: no
  file of `wizard/`, `llmprovider/` or `internal/` changed since the readiness
  passes read them. The step-5 addition's citations each hold at `77c59ca`:
  `wizard/auth.go:478` (the organizations warning), `wizard/kilo_verify.go:128`
  (the notice), `wizard/kilo_verify_0080_test.go:362` and `:383`,
  `docs/architecture.md:1148-1149`, `CHANGELOG.md:40-41`, 0080-MADR D6 at line
  433, 0082-MADR N-6 at line 1145. **Moved: none.** The listing warnings' lines
  (`wizard/configure.go:948`, `:951`, `:958`, `:967`, `:974`, `:977`) hold at
  the base; they are at `:1026`, `:1029`, `:1036`, `:1045`, `:1052`, `:1055` in the
  built tree. The PLAN's "Found, not mine to change" is 0080-PLAN's "Found, not
  changed by 0080" (line 1076), whose first item this pass annotates.

**Step 1, the base re-read** (`wizard/configure.go` at `77c59ca`):
`credentialStep` `:498`, its `endedCheck` `:575` and `save` `:578`, doc
`:484-497`; `ConfigureLLM` `:318`, the step `:332`, the model-id branch
`:336-352`, `selectModel` `:354`, the fallbacks `:357-361`; `ResolveCredential`
`:415`, the step `:429`, its last check `:433-437`;
`resolvedCredential.save` `wizard/auth.go:1103` (`Token` check `:1107`),
`write` `:1180`; `checkEnded` `wizard/kilo_verify.go:43`, `verifyKilo` `:82`.
The only tests that call `credentialStep` are `TestKiloVerify_EndedContext_0080`
(`wizard/kilo_verify_ctx_0080_test.go:46`, `:62`): unchanged, green.

**Step 4, the code** (`wizard/configure.go`): `unsavedStep` is `credentialStep`'s
old body returning `(Result, catalog.Catalog, resolvedCredential, error)`;
`commitCredential(ctx, p, d, c)` is the context check, then `c.save(ctx, p, d)`
(the `Prompter`, by the annotation of 2026-11-20); `credentialStep` keeps its
signature and is the two in order; `ResolveCredential` returns it, with no
check of its own; `ConfigureLLM` calls `unsavedStep`, checks the context once
(N3) and commits before each of its two successful returns. `notifyListing`
(new, unexported) shows a listing warning only while the run's `ctx` is alive,
and the six sites in `discoverModels` use it, on `ctx`, never on the listing's
`dCtx` (S61, S71, S72, S77). `resolveKiloDevice` shows its organizations warning
only while `ctx` is alive, with `Discover` or without (S73, S74). `verifyKilo`'s
notice reads "...; going on without verifying it" (S76). `write`'s check is
not moved, removed or repeated (S63). No exported name changed.

**Step 2, the tests.** `wizard/cancel_save_0081_test.go` and
`wizard/save_at_end_0081_test.go`, 32 tests, none of the existing tests
changed but `wizard/kilo_verify_0080_test.go`'s two expected words (`:362`,
`:383`; the ruling "Fix both" allows it). Each lists through `testCache(t)` with
the model metadata off, over in-process stand-ins; `TestCancelDuringDeviceFlowSavesNothing_0081`
runs Grok's real `auth` device flow against an issuer the run's `HTTPClient`
answers in process (discovery, keys, the device code, one poll of a one-second
interval, a verified ID token), so it was **not dropped** and no non-test code
changed for it. Rows to tests: as the readiness pass's map; P-1 and P-3 to P-21
are built under the names the entry gives (P-3 and P-4 inside
`TestCancelAtModelStageSavesNothing_0081` and `TestModelStageErrorSavesNothing_0081`,
P-8 inside `TestCancelDuringKiloProfileSavesNothing_0081`, P-9 inside
`TestCancelAfterIssueSavesNothing_0081`, P-14 as `TestDeadlineReturnsNoKey_0081`).

**Step 3, red at the base and green on the tree** (`tools/redgreen.py`, outputs
in the build engineer's scratch folder):

* `RED.json --expect fail --rev 77c59ca`: **27 of 27 fail as expected**, exit 0.
* `GUARDS.json --expect pass --rev 77c59ca`: **5 of 5 pass**, exit 0:
  `TestCancelDuringRefreshSavesNothing_0081` and `TestResolveSavesInTheStep_0081`
  (the PLAN's guards), and the three tests whose plants alone hold them,
  `TestCancelBeforeIssueSavesNothing_0081` (P-1; the sign-in's own errors are
  unchanged), `TestStoreErrorAtTheCommit_0081` (P-12) and
  `TestListingLimitIsAnOutageAndSaves_0081` (P-13).
* `--expect pass` on the tree: **32 of 32 pass**, exit 0.
* `--rev b109cea` cannot be run for the new files: they use helpers that are not
  at that commit (`newKilo0080`, `scriptedLock0082`, `clientNotices0082`; the
  wizard package does not build). The base of this phase is `77c59ca`.

**Deviation 1 (for the owner): five tests the PLAN calls guards are red at the
base on the words and the no-model-question assertions.** `TestCancelDuringListingSavesNothing_0081`,
`TestCancelAfterIssueSavesNothing_0081`, `TestCancelDuringKiloProfileSavesNothing_0081`,
`TestDeadlineDuringListingSavesNothing_0081` and `TestCancelDuringListingReturnsNoKey_0081`
were guards (green at `a7e924f`) as drafted. The second pass made each assert the
whole error text (`wizard: <ctx>`) and, for `ConfigureLLM`, that no model
question is asked, so that the commit's and N3's plants are caught; at the base
`write`'s words (`wizard: the new Grok (xAI) session was not saved: context
canceled`) and a model question come back instead. Their count assertions (no
save, an error matching the context's) are green at the base; the red lines are
only the words (for example `cancel_save_0081_test.go:354: err wizard: the new
Grok (xAI) session was not saved: context canceled; want "wizard: context
canceled"`) and, for `ConfigureLLM` of a typed key or an OpenAI session, `model
questions ["No models found; enter a model id"] asked; want none`. The tests are
written as the map says; the table's "guard" labels for these five are
superseded by this entry and `RED.json` lists them as red. The owner's options,
put by the coordinator: (a, recommended) keep the five as they are and record
them red at the base, as here; (b) split each into a words-free guard and a
words test, for the label's sake, five more tests; (c) drop the words from the
five, which would leave the commit's and N3's plants to other tests alone.
**Resolved, 2026-11-20:** by the owner, relayed by the coordinator: "Keep as built; record them as red before the change (Recommended)": (a). No code and no test
changed for it. `RED.json`, `GUARDS.json` and `ALL.json`, the specs given to
`tools/redgreen.py`, are scratch files of the build engineer, not in the tree;
the red list holds these five, the guard list the five that are green at the
base.

**Other differences from the PLAN as written.**

* `TestCancelDuringRefreshSavesNothing_0081` cancels 400 ms after the refresh
  arrives, not 200 ms (S12), so that the save is waiting on a loaded host; the
  assertion (the error within one second of the cancel) is as written.
* `kilo_verify_0080_test.go:383` is a prefix of the notice (`verify`); it now
  holds the full new words, which strengthens it and changes no behaviour.
* **Found, not changed:** `wizard/auth_test.go:585`'s comment says
  `credentialStep` saves a session; it still does for `ResolveCredential`, and
  `ConfigureLLM` now saves through `commitCredential` at its end. A test file
  not on the phase's list, so left as it is, for Phase 3's docs pass or a later
  record.
* `unsavedStep` is the unexported step's name; the PLAN gave no name.

**Spot-check of plants** (informal, on a scratch clone with `tools/plant.py`; the
campaign and the Windows run are Phase 3's): 19 plants, **all 19 caught**: the
commit's check removed; the check for a session only; a cancel matched, not a
deadline; the commit skipped when the store holds a session; N3's check removed;
`ConfigureLLM` saving in the step again; the model-id branch not committed;
`ResolveCredential`'s own last check put back; the listing warning shown
whatever the context; the suppression on a cancel only; a listing ended by its
own limit returned as a `*checkEnded`; the organizations warning kept under
`Discover` only; Kilo's words back; `write` ignoring the store's error;
`signsInAgainTo` ignoring the record; `save`'s `Token` on a context that never
ends; a context check added after `Save`; each of the two architecture
sentences removed.

**Docs** (step 5): doc comments of `ConfigureLLM`, `ResolveCredential`,
`Options.TokenStore`, `credentialStep`, `unsavedStep`, `commitCredential`,
`resolvedCredential.save` (its last sentence as the PLAN gives it) and `saveTo`,
`signin.SignIn.Wait`; `README.md`'s sign-in bullet; `docs/architecture.md`
("When a session is saved", with the Kilo sentence, the measured-premise
sentence and the "last commit wins" sentence; the wizard's paragraph on the
save; Kilo's words); `CHANGELOG.md` `[Unreleased]` (**Fixed**, **Changed**, and
the 0080 entry's words); dated notes in 0059-MADR (W2, W8, the amendment),
0056-MADR D1, 0080-MADR D1 and D6, and 0080-PLAN's first "Found, not changed by
0080" item.

**Gates run** (session host, each status taken before any filter): `gofmt -l .`
empty; `go vet ./...` and `go vet -tags live_gateways ./...` exit 0;
`go test -count=1 ./wizard/ ./llmprovider/x/signin/` exit 0; `run-heavy -- make
pre-add-check FILES="..."` over the seven Go files (host and `GOOS=windows`
`golangci-lint` v2.15.0, `go vet`, `go test` of the two packages) exit 0, after
a first run found one `gocognit` finding in `TestNoListingWarningOnAnEndedRun_0081`
(51 of 50), which moved its table to a helper function; `run-heavy -- go test
-race -count=1 ./...` exit 0 (43 packages ok); `run-heavy -- go test -race
-count=3 -run '_0081$' ./wizard/` exit 0; `make api-check` exit 0 and `python3
-B tools/apidiff.py v1.31.0` exit 0 (it lists only `signin.VerifyKilo`, 0080's);
`make records-check tables-check changelog-check docs-check links-check
disclosure-check` exit 0; markdownlint-cli2 0.23.2 (the cached copy, run with
`--no-globs` over `check_links.py --lint-files`) 0 issues; `tools/claims.py` on
both records 47 of 47 and 25 of 25; `tools/denyscan.py --changed` 0 hits;
`tools/recordlint.py --changed` no rule rose (after one MD012 in this entry,
fixed). Not run: `govulncheck` (the network), the Windows run and the gate
(Phase 3), `make tools-test` and the other CI targets (Phase 3's list).

**Not done in this pass:** Phase 3 (the plant campaign, the Windows run, the
gate), and the `--rev b109cea` runs above. The plan's status is
unchanged (`in-progress`); its front-matter date moved to 2026-11-20.

### Phase 2, build pass 2 (2026-11-20): the docs and S22's other failures

The independent review passed the code, every red and green claim and its own
eight plants, and failed pass 1 on one blocker, B1, which this pass fixes with
two small items. Same rules and boundary as pass 1; nothing staged.

**Deviation 2 (B1): docs said an error from a run always means nothing was
saved.** Found by the independent review, at pass 1's text. The statements:
`Options.TokenStore`'s and `ConfigureLLM`'s and `ResolveCredential`'s doc
comments, a comment in `ResolveCredential`, `docs/architecture.md`'s "When a
session is saved", `CHANGELOG.md`'s **Changed** bullet and the dated note in
0056-MADR D1. The state table's S53 (decided) says otherwise: when the store's
own `Save` fails after it has written, `FileTokenStore`'s directory sync after
the rename (`llmprovider/auth/tokenstore_file.go:244`), the run returns the
store's error with the new session already in the store; D9 row 7 ("the store's
error as today") and S52, S53 say the same, and `TestStoreErrorAtTheCommit_0081`
pins it. The wording came from D5 ("any error ... saves nothing", "its error
means nothing was saved") and step 5 of this PLAN, which read against S53 are a
contradiction inside the record, not a new decision: S53 is decided, and the
code and the test follow it. The docs now follow S53: each says the exception
(`wizard/configure.go`: `Options.TokenStore`, `ConfigureLLM`, `ResolveCredential`
and the comment in its body; `docs/architecture.md`; `CHANGELOG.md`; the
0056-MADR note), and the MADR carries a dated annotation on D5 pointing to S53.
No owner ruling was needed, and none was asked; the step 5 text above is left
as written, and this entry supersedes where it says "any error".

**Small item 1.** Pass 1's entry said the plan's date was unchanged; it moved
to 2026-11-20 (front matter). The sentence is corrected.

**Small item 2: S22's 429 and transport cases.** `TestSessionDueAtTheCommit_0081`
gains a refresh answered 429 on each of the SDK's three tries, and a refresh
whose request fails in transport (`connection reset`) on each, both with the
run alive: the session is saved as it was, its refresh token unspent, and the
Result. Each is green on the tree. Each was seen to fail under the existing
plant, `save`'s first check refusing any error (`tools/plant.py`, scratch clone,
never the tree): "err wizard: the new Grok (xAI) session was not saved:
llmprovider: rate limited: oauth: refresh failed: Too Many Requests ...; want
the Result" and "... refresh request: Post ... connection reset ...", each
with `saves 0 of []; want 1, of "oauth-refresh"`, beside the 503 case's.

**Follow-ups, listed and not changed** (the review's, and pass 1's): the stale
comment at `wizard/auth_test.go:585`; P-20's `"kept"` assertion
(`TestSaveNotesInTheArchitecture_0081`), which matches the word anywhere in the
bullet; P-16's test-only environment variables
(`SDK_TEST_0081_STORE`, `SDK_TEST_0081_SIGNAL`), which `scripts/check_tables.py`
does not cover.

**Gates, pass 2** (each status taken before any filter): `gofmt -l .` empty; `go vet ./...` 0;
`golangci-lint run -c .golangci.yml --build-tags live_gateways ./wizard/... ./llmprovider/x/signin/...`
0 issues, for the host and for `GOOS=windows`; `go test -count=1 ./wizard/
./llmprovider/x/signin/` 0; `tools/recordlint.py --changed`, `make records-check
docs-check links-check changelog-check`, `tools/denyscan.py --changed` and
`tools/claims.py` on both records as in the hand-back. No race rerun: no
existing test's timing changed, and the two new cases wait the same 0.6 s as the 503's.

### Phase 3, build pass 1 (2026-11-20): the plant campaign, the Windows run and the gates

Run on `main` at `cff8017` (Phase 2's commit), by a build engineer on the
coordinator's brief. `git status --porcelain --untracked-files=all` was empty at
the start; no other agent worked in the tree. No code and no test changed; plants
ran on scratch clones only (`tools/plant.py`, `tools/redgreen.py`); nothing was
staged, committed, stashed or reverted. No live test, sign-in, built binary or
provider was used. The specs (`PLANTS.json`, `RED.json`, `GUARDS.json`,
`ALL-tree.json`) are scratch files of the build engineer, not in the tree; this
PLAN does not ask for them to be committed.

**Step 0, the re-grounding at `cff8017`.** `python3 -B tools/claims.py` on the
MADR and this PLAN at their markers: 47 of 47 and 25 of 25, exit 0. The re-pointed
rows ("C1 now" to "M2 now", "P1 now" to "P25 now") on a scratch copy with plain
ids and `self=cff8017` (the MADR's other repositories at its marker's revisions):
MADR 30 of 47, this PLAN 18 of 25, exit 1 each. Every row that moved is a line of
a file Phase 2 was to change, and each moved because of that change, none by
another: the text of **MADR** C1, C2, C3 (`wizard/auth.go:1103`, `:1107`, `:1185`;
found at `:1111`, `:1115`, `:1193`), C8, C15, C16, C24, C29 (`wizard/configure.go`,
text found at `:345`, `:371`, `:390`, `:381`, `:394`), C11 (`:919`, found at
`:1004`), C14 (`:939`, found at `:1024`), C13 (`wizard/auth.go:478`, found at
`:484`) and C17 (0080-MADR `:329`, found at `:334`) is present at a later line;
C9 (`ResolveCredential`'s last check), C10 (the step's save), C20 (`Wait`'s
doc) and C23 (`ConfigureLLM` calls `credentialStep`; now `unsavedStep`) have no
longer the base's text, which is the change itself (D5, D7; their new words are
in `wizard/configure.go` and `llmprovider/x/signin/signin.go`); C32's text is at
`:387` and `:401`. **This PLAN's** P1 (`README.md:434`), P3
(`docs/architecture.md:1120`, text at `:1123`), P6 (`wizard/configure.go:433`), P7
(`:484`), P16 (`wizard/auth.go:55`, `saveTo`'s comment reworded), P14
(0059-MADR `:494`, found at `:501`) and P18 (`:147`, found at `:151`) moved or
were reworded by Phase 2's docs step. The rest hold. No moved line shows a
behaviour that Phase 2 did not decide: the moved code is the decided code, and
`git diff --stat 77c59ca cff8017` lists the sixteen files of Phase 2 only.

**Step 1, the plants** (`python3 -B tools/plant.py PLANTS.json`, scratch clone,
each plant run against `go test -count=1 -v ./wizard/ ./llmprovider/x/signin/`
with a runner script that prints the failing test names): **26 plants, all 26
caught**, exit 0. Phase 2's 19 (named in its entry) and seven more from "The
plants, re-mapped after 0082": `save`'s first check refusing any error (P-5, S22);
`save`'s first check removed; `write`'s check removed; `write`'s check moved
before the lock's wait (into `writeUnderLock`, before `lockToSave`); `write`'s
check skipped on the retry pass; the listing's suppression on the listing's own
`dCtx` (two of the six sites); the organizations warning suppressed whatever the
context. First failing tests, each from the quoted run: the commit's check
removed, `TestCancelDuringListingSavesNothing_0081` and the device-flow and
issue guards (`ResolveCredential` cases); the check for a session only,
`TestDeadlineReturnsNoKey_0081`, `TestCancelDuringListingReturnsNoKey_0081`,
`TestCancelledRunReturnsNoCredential_0081`; a cancel matched, not a deadline,
`TestDeadlineDuringListingSavesNothing_0081`, `TestDeadlineReturnsNoKey_0081`,
`TestCancelDuringKiloProfileSavesNothing_0081/deadline_during_the_read`;
`ConfigureLLM` saving in the step again,
`TestConfigureLLM_BrowserAndDevicePersistSessions`,
`TestCancelBetweenTheCheckAndTheWriteSaves_0081`,
`TestConfigureReturnsTheResultWhenCancelledInTheWrite_0081` (not
`TestConfigureSavesOnceAfterTheModel_0081` or `TestModelStageErrorSavesNothing_0081`
among the first twelve lines the tool prints, which cap the list); the model-id
branch not committed, `TestConfigureReturnsTheResultWhenCancelledInTheWrite_0081/through_the_model-id_branch`;
`ResolveCredential`'s own last check put back, `TestCancelAfterSaveReturnsTheResult_0081`;
N3's check removed, `TestCancelDuringListingSavesNothing_0081/.../ConfigureLLM`;
`write`'s check removed, `TestSaveCancelledAtAnyWaitSavesNothing_0082/a_context_ended_while_the_prompt_was_open`
and `TestSaveWithALockerThatIgnoresTheContext_0082`; `write`'s check moved before
the wait, `TestSaveWithALockerThatIgnoresTheContext_0082` alone; `write`'s check
skipped on the retry pass,
`TestSaveCancelledAtAnyWaitSavesNothing_0082/a_context_ended_while_the_prompt_was_open`
alone; `save`'s `Token` on a context that never ends,
`TestCancelDuringRefreshSavesNothing_0081`; `save`'s first check refusing any
error, `TestSessionDueAtTheCommit_0081` (the 503, 429 and transport cases); each
architecture sentence, `TestSaveNotesInTheArchitecture_0081`; `signsInAgainTo`
ignoring the record, `TestAbandonedPlanRunKeepsTheClientRecord_0081` and
`TestPlanPreviousOnlyForTheSameSignIn_0082`; the organizations warning,
`TestCancelDuringKiloProfileSavesNothing_0081` (kept under `Discover` only) and
`TestKiloVerify_NoDiscoverNoRead_0080/device_session`,
`TestKiloVerify_OutagesWarn_0080` (suppressed always); the Kilo words,
`TestKiloUnverifiedNoticeClaimsNoSave_0081`.

**Differences from the re-mapped table (reasoned, now run).** The plant on the
listing's own `dCtx` was caught by `TestConfigureLLM_ListingTokenFailureUsesStaticCatalog`
and `TestSession_RefreshAfterLimitSaved_0059` only; the table's P-13
(`TestListingLimitIsAnOutageAndSaves_0081`) and P-17's deadline case did not fail
under it (the plant moved two of the six sites, the two where a failed listing
has or has not a static catalog). It is caught, by tests that already existed;
this is a gap in which of the new tests cover that plant, not a plant uncaught,
and is left for the owner.

**Step 2, red and green** (`tools/redgreen.py`, outputs in the scratch folder):
`RED.json --expect fail --rev 77c59ca` **27 of 27 fail as expected**, exit 0;
`GUARDS.json --expect pass --rev 77c59ca` 5 of 5 pass, exit 0; `ALL-tree.json
--expect pass` on the tree **32 of 32 pass**, exit 0. (A first `--expect pass`
run with the spec that carries `files` was refused by the tool, "files and drop
need --rev", exit 2; the tree-form spec is the right one.) `--rev b109cea`
remains impossible for the new files (Phase 2's entry).

**Steps 3 to 7, and the Windows run** (each status taken before any filter):

* `run-heavy -- go test -race -count=1 ./wizard/... ./llmprovider/x/signin/... ./internal/signinflow/... ./llmprovider/auth/...`, session host, exit 0 (four packages ok).
* `run-heavy -- python3 -B tools/gate.py --timeout 5400`, session host, exit 0, 30 of 30 passed (589 s).
* `make lint` exit 0 (0 issues); `make vuln` exit 0 (no vulnerabilities found; the network was reachable); `go vet -tags live_gateways ./...` exit 0.
* `make parity-check dep-check coverage-check api-check contract-check generate-check records-check tables-check fixtures-check changelog-check docs-check markdownlint links-check disclosure-check gate-selftest`, each exit 0 (one run per target); `run-heavy -- make tools-test` exit 0.
* `python3 -B tools/claims.py` on the MADR and this PLAN: 47 of 47 and 25 of 25, exit 0; `tools/denyscan.py --changed` and `tools/recordlint.py --changed`, exit 0 (re-run after this entry was added, below).
* **Windows:** `run-on-host --need windows -- go test -count=1 ./wizard/`, native Windows host, remote status 0, `ok .../wizard 21.187s`. P-16's kill and `FileTokenStore`'s write ran on Windows.

**Not run / not done:** `--rev b109cea` (see above). Nothing else skipped. Phase 3's exit criterion (no blocker, every row tested, plants caught) is met; the plan's status stays `in-progress` for Phase 4.

### The release preparation (2026-11-22)

`CHANGELOG.md`'s `[Unreleased]` became the `v1.32.0` section dated 2026-11-22 (R53: a section is dated by its tag's commit, so the date is re-checked by `make changelog-check` once the owner's commit exists); `[Unreleased]` is empty. `python3 -B tools/apidiff.py v1.31.0` lists additions only (`APIError.Tool`, `ErrToolSchemaRefused`, `x/signin.VerifyKilo`, `wizard.Options.NoBrowserQuestion`, `wizard.Options.NoLoopback`): a minor version. The tag is the owner's. This record's status stays as it is and goes to `complete` when the tag exists, its CI has passed, and `tools/consumer.py` and `tools/offline.py` are recorded (`tools/records.py status`). This record's entries are under `Changed` and `Fixed`. Its stale comment at `wizard/auth_test.go` (the `TestResolveTokenStdin_ChecksTheCredential` doc named `credentialStep` where `commitCredential` saves, `ConfigureLLM` as its last act) is corrected in this preparation: comment only.
