---
status: accepted
date: 2026-11-17
decision-makers: repository owner
consulted: "the owner's ruling of 2026-11-17 and the owner's answers to Q1 to Q7 of the same day, relayed by the coordinator (labels quoted below); this repository at b109cea (five commits after v1.31.0, the commit before 0080's code) and 0080's staged tree as read on 2026-11-17; gobble-cli at cecc892 (branch p3-acpext-v0.5.0), its 0051-MADR and 0051-PLAN (sign-in through ACP elicitation), the PLAN's \"Review and fixes\", finding B3; prepare-commit-msg at 2715f2d; the codex checkout at 57e64682f8 (read only); an independent reviewer's reproduction, relayed by the coordinator"
informed: gobble-cli, which signs in through wizard.ResolveCredential on its terminal path and through x/signin on its ACP path; prepare-commit-msg, which configures through wizard.ConfigureLLM; every caller of wizard.ConfigureLLM and wizard.ResolveCredential
---

<!-- markdownlint-disable MD013 MD024 -->

# A Cancelled Sign-In Saves Nothing: the Wizard Checks the Run's Context Before It Saves, and ConfigureLLM Saves Last

## Context and Problem Statement

The wizard saves a new session to `Options.TokenStore`, the session's only
copy (0016-MADR A10), once the model listing has not refused it
([0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md)
W2, Q2 (a)). Before it saves, it waits for a refresh the listing left
running, and saves nothing when the issuer refuses that refresh "nor
[...] when the run's context ends first" (0059-MADR, amendment "a refresh
can outlast the listing (Phase 2)").

That last clause holds only while a refresh is in flight or due. When the
run's context ends during the listing that checks a fresh sign-in, the
wizard saves the new session and then returns the context's error. And
`ConfigureLLM` saves in its credential step, before the model and fallback
questions, so a run the person abandons at a model question has already
replaced the stored session; a cancel that arrives while the last question
is on screen even returns a Result with no error (K10).

### The owner's ruling of 2026-11-17

Relayed by the coordinator; its label, verbatim: **"A record of its own,
drafted now"**.

As it was put to the owner: a separate SDK decision record and plan for "a
cancelled sign-in saves nothing": a context check before the save in the
credential step, with tests for each session kind; it can share the next
release with
[0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md);
drafted for the owner's approval while 0080 finishes. Options shown and not
chosen: fold it into 0080; list it and fix it later.

### Decided: the owner's answers of 2026-11-17

Relayed by the coordinator. Each label is quoted verbatim; what was asked is
said beside it in this record's words.

* **Q1, the windows:** "Every window before the write, by one check just
  before the save (Recommended)". Asked: guard every window up to the
  write with one check, rather than the listing alone, or the write itself
  too. (D2)
* **Q2, the credentials:** "Every kind, typed keys included (Recommended)".
  Asked: whether a key, typed or otherwise, is covered as a session is.
  (D3)
* **Q3, deadlines:** "A cancel or the run's deadline saves nothing; the
  listing's own limit still saves (Recommended)". Asked: whether the run's
  deadline counts as a cancel, and whether the listing's own bound does.
  (D4)
* **Q4, after the write:** "The first option, and `ConfigureLLM` saves at
  its very end". Asked: what a cancel after the save means. The first
  option: `ResolveCredential`'s own last check moves before the save, so
  its error means nothing was saved and its Result means it was. Added to
  it: `ConfigureLLM`'s save leaves the credential step and becomes the last
  thing it does, after the model and fallback questions, so any cancel
  inside `ConfigureLLM` saves nothing. It was the draft's option (c), not
  the one recommended. (D5)
* **Q5, revocation:** "Out of scope; say so in the docs (Recommended)".
  Asked: whether to revoke a session the provider issued and the wizard did
  not save. (D6)
* **Q6, `x/signin` and the plan's client record:** "Leave both; add one
  sentence to `Wait`'s doc (Recommended)". Asked: whether the engine's
  `Wait`, or the plan's client record, change. (D7)
* **Q7, the release:** "All four in the next release with 0080, on my word
  (Recommended)". Asked of the four records drafted alongside 0080 together:
  whether they ship in 0080's release. The changelog section was not asked
  then; it is N4. (D8)

* **Ruling, 2026-11-17: accepted.** Accepted 2026-11-17 by the owner's ruling,
  relayed by the coordinator: "Accept all five and run each plan up to its
  live step or release (Recommended)". The option's description, as put with
  it by the coordinator (the coordinator's words, not the owner's own): "Each
  is marked accepted and built in dependency order, one at a time in the SDK
  tree, each with its independent review; 0081 and 0084 stop at the probes
  only you can run; nothing is tagged without your word. They are committed
  and pushed under your standing approval as each phase passes."

The investigation of Q4's answer raised N1 to N5. The owner answered N1 to
N4 the same day, each as the draft recommended; N5 was not put to the
owner:

* **N1, whether a new sign-in ends the earlier grant:** "A live probe by me
  before release (Recommended)". Asked: whether to measure, for the ChatGPT
  browser sign-in, the plan sign-in, Grok and Kilo, if signing in a second
  time ends the first session, by the owner signing in twice and then using
  the first session, recording status codes only, with the PLAN stopping
  and coming back to the owner if any provider ends it. (D12)
* **N2, other errors at the model stage:** "Any error from `ConfigureLLM`
  saves nothing (Recommended)". Asked: whether an error that is not the
  context's end (a Prompter's error, a closed input, no model entered) also
  leaves the store as it was. (D5)
* **N3, where `ConfigureLLM` checks:** "Right after the credential step, as
  well as at the end (Recommended)". Asked: whether a context that ended
  during the step stops the run before any model question. (D5)
* **N4, the changelog:** "Fixed for the cancel windows, plus Changed for the
  later save (Recommended)". Asked: which changelog section, or sections,
  the change goes under. (D8)
* **N5, one save path:** not put to the owner, since no caller can see it.
  The coordinator applied the draft's option (a), `credentialStep` keeping
  its signature and `ConfigureLLM` calling the unsaved half, on the
  evidence that changing the signature breaks
  `TestKiloVerify_EndedContext_0080` and that the prototype passes with
  (a) (R5). It is the coordinator's application, not the owner's ruling.
  (D5)

* **N6, how the probe shows the person what a sign-in needs:** not put to
  the owner. Raised by drafting N1's probe: a browser sign-in needs its
  authorize URL opened, and Kilo's device sign-in needs its user code read
  by the person (Kilo's own client prints it and opens its page, K12). The
  coordinator applied the draft's option (a): the test opens each URL in
  the browser itself and writes Kilo's user code to the terminal device
  (`/dev/tty`) only, never to the test's output. The coordinator's reason:
  the owner's standing constraint for live probes is that no URL with a
  code, code, pasted text, token or client id is displayed in or recorded
  from a probe's output, and (a) is the only one of the three options that
  satisfies it by construction; (b), logging them in the output, would
  need the output the owner sends back edited by hand; (c), probing Kilo
  only if its page shows the code, would leave Kilo unmeasured, against
  the ruling that named four sign-ins. It is the coordinator's
  application, not the owner's ruling; the owner is told. (D12)

**What stays open:** nothing. The record is `proposed` until the owner
approves it.

### The evidence, item by item

Each item says how it was established: "reported by the reviewer, re-run
here", "read", or "not re-run". "Re-run here" means a scratch test against a
scratch copy of the tree at `b109cea` (`git archive`), and of 0080's staged
tree as read on 2026-11-17, never in the working tree. Every sign-in in
those tests is an in-process stand-in: the wizard's own seams for the
browser and device logins and for Kilo's profile (`loginBrowserOAuth`,
`deviceLogin`, `kiloProfile`, the seams the package's tests already use),
and a `RoundTripper` for the listing and the issuer. No outside host, no
real credential. The two test files are kept for the PLAN (Phase 1).

* **R1. A cancel during the listing saves a device session.** Reported by
  the reviewer for a Kilo and a Grok device session, through both
  `ResolveCredential` and `ConfigureLLM`, at `b109cea` and on 0080's staged
  tree. **Re-run here**, and widened to six session kinds: Kilo's device
  session, Grok's device and browser sessions, OpenAI's browser and device
  sessions, and a pasted ChatGPT access token. All twelve cases (six kinds,
  two entry points) save once at `b109cea`, and again on 0080's staged tree:
  "a sign-in cancelled during its listing saved 1 time(s); want none".
* **R2. The cause.** Reported by the reviewer as `wizard/auth.go` "about
  `:1015` at `b109cea`"; **read**: at `b109cea` (and at `v1.31.0`) `save`
  is `wizard/auth.go:991`, its check `:995` (C1, C2); `:1015` is the same
  line on 0080's staged tree. The check refuses on an ended context only
  when `Token(ctx)` also fails (C2). `Token` returns a current token without
  looking at the context (C4); it reads the context only before it starts a
  refresh (C5) or while it waits for one. A fresh session is current, and a
  Kilo session has no expiry, so for them the check never fires.
  `FileTokenStore.Save` takes no notice of the context (C6). The save is then
  made (C3). It is one function for every provider, so the defect is every
  session kind's, as R1 shows.
* **R3. 0080 does not change it.** **Read**: 0080-MADR D3, "No other
  provider's check changes" (C17). On 0080's staged tree, `credentialStep`
  returns the context's error before the save only for a `*checkEnded`,
  which 0080's Kilo profile check alone returns; a listing the context ended
  is still an outage and the save follows. **Re-run here** (R1's twelve
  cases fail the same way on that tree). 0080's own record names the gap as
  found and left for the owner (0080-PLAN, execution record, "Phase 3 (with
  Phase 2's tests)", point 3, "Found, not mine to change").
* **R4. gobble's note, reconciled.** The brief relayed it as: this SDK's
  wizard "refuses to save once `ctx.Err() != nil`" at `wizard/auth.go`
  about `:995` at `v1.31.0`. That exact sentence was **not found** in
  gobble-cli at `cecc892`, nor in its working tree; the nearest statements
  are these, each **read**:
  * gobble-cli `llm/provider/signinrun.go:145` to `:146`: "A context that
    is done before the save fails the sign-in with it, as the wizard does
    not save then." (G1)
  * gobble-cli `internal/cli/agentauth.go:102` to `:103`: "the wizard saves
    nothing once its context is done" (G2).
  * gobble-cli
    `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`,
    D4: "as the wizard's `save` does (`wizard/auth.go`) [...] and it saves
    nothing once its context is done" (G3).

  **Which path has the check and which does not.** `wizard/auth.go:995` is
  the line those notes describe, and it does hold `ctx.Err() != nil`, but
  inside `err != nil && (... || ctx.Err() != nil)` (C2): it refuses on an
  ended context only when `Token` fails, which is the refresh path (in
  flight or due, 0059-MADR's amendment). The path without the check is a
  current token: every fresh sign-in, and every Kilo session. gobble copied
  the same conjunction (G4) and added its own unconditional check before its
  save (G5), which is why gobble's ACP path saves nothing after a cancel and
  this module's wizard does. The re-run shows both halves at `b109cea`: a
  session whose refresh is in flight when the run ends is not saved
  (`TestCancelDuringRefreshSavesNothing_0081` passes), and a current one is
  (R1).
* **R5. The ruled design is buildable, and breaks no existing test.** A
  prototype, on scratch copies only (never the tree), of D2 to D5 with
  N2 (a), N3 (a) and N5 (a): `credentialStep`'s body becomes an unexported
  step that returns the checked credential unsaved; `credentialStep` keeps
  its signature and is that step followed by `commitCredential` (one
  context check, then `save`); `ResolveCredential` calls `credentialStep`
  and no longer checks the context after it; `ConfigureLLM` calls the
  unsaved step, checks the context once after it (N3 (a)), and calls
  `commitCredential` as the last thing before each of its two successful
  returns. Results:
  * the fifteen tests of the reproduction: at `b109cea` twelve fail and
    three guards pass; on the prototype all fifteen pass. The same on a
    copy of 0080's staged index with the same prototype;
  * `go test -count=1 ./wizard/` passes whole, the fifteen tests
    included, on the prototype at `b109cea` ("ok", 8.5 s) and on 0080's
    staged index ("ok", 9.5 s), with no existing test changed. 0080's
    `TestKiloVerify_EndedContext_0080` calls `credentialStep` directly;
    keeping its signature (N5 (a)) is why it compiles unchanged;
  * seven planted breaks of the prototype, each caught (Confirmation).

  A prototype is evidence of size and feasibility, not the implementation;
  the PLAN writes it.

### Every window, from the user's cancel to the write

"Cancelled" here is the end of the run's context: the caller cancels it
(Ctrl-C mapped to a cancel, an ACP cancel), or its deadline passes. A
question the user declines, a Prompter's error and a `Driver`'s error are
not context ends: each returns its error from the sign-in or a menu. The
listing's own limit (`Options.DiscoverLimit`, C11) is a deadline of the
wizard's, not the run's: a listing that runs out of it is an outage, which
saves (0059-MADR's amendment; 0080-MADR D6). A run whose own deadline passes
during the listing saves today as a cancelled one does (re-run:
`TestDeadlineDuringListingSavesNothing_0081`, saves once at `b109cea`).

| # | Window | Today, at `b109cea` | How it is known |
| :--- | :--- | :--- | :--- |
| K1 | Before the provider issued anything: the browser's wait, the device poll, a consent or paste step | The login returns the context's error; nothing was issued; nothing is saved | read (`internal/signinflow` `Run.Wait`, `deviceFlowFor`); not re-run |
| K2 | The provider issued a session and the login returned it; the context ends before the check | The wizard waits for the sign-in under `context.WithoutCancel` (C12), so the issued session comes back; the listing fails as an outage; **saved**, with or without `Discover` | re-run: `TestCancelAfterIssueSavesNothing_0081`, both cases save once |
| K3 | Kilo's device session: the profile read for the organization menu | A failure is a warning and the personal account (C13); then **saved**. Under `Discover` on 0080's tree the read's end returns the context's error (0080's `checkEnded`); without `Discover` it is still K3 | re-run: `TestCancelDuringKiloProfileSavesNothing_0081` saves once |
| K4 | During the listing, the check | Not a refusal (C14): a warning, the static catalog, the loop ends, **saved** (C3). Then `ResolveCredential` returns the context's error (C9); `ConfigureLLM` returns it at the model question (C16), or, with no static catalog (a ChatGPT session), asks for a model id on the ended context (C15) and returns the Result **with no error** | re-run: twelve cases save once; the three OpenAI `ConfigureLLM` cases also return `err <nil>` |
| K5 | The listing left a refresh running, or the token is due, when `save` asks `Token(ctx)` | `Token` returns the context's error; `save` refuses; **not saved** | re-run: `TestCancelDuringRefreshSavesNothing_0081` passes |
| K6 | Between the check and the write | Nothing reads the context (C4, C6); **saved** | read; K4's runs pass through it |
| K7 | During the write | `FileTokenStore` writes a temp file, syncs it, and renames it into place (C7): the store holds the old file or the new one, never half of one. A write that has begun cannot be stopped by a context and cannot be undone by one | read |
| K8 | After the write, `ResolveCredential` | Its own last check (C9) returns the context's error with the session saved | re-run: "err wizard: context canceled, [...] saves 1" (`TestCancelAfterSaveReturnsTheResult_0081`) |
| K9 | The ChatGPT plan's issued client and host id | Saved as the plan login returns, before the wizard's check (C18), by design (0075-MADR D7): a client id and a host id, not a credential; `FileTokenStore.SaveClient` ignores the context (C19) | read |
| K10 | `ConfigureLLM`'s model stage: the manual model id, the model search, menu and id, the fallbacks | The session was saved before it (C10). Each question loop checks the context only at its top (C25, C30), so a cancel while the last question is on screen, answered anyway, returns the Result with **no error**; a Prompter's error there returns its error with the session saved | re-run: four model-stage cases "err <nil>, kind "oauth"" and "saved 1"; a typed key at the model menu "err <nil>, key returned true"; the manual model id "err <nil> [...] saves 1"; a Prompter's error "saved 1 time(s)" |

**A key.** The wizard writes no key: a typed, environment or saved key, and
Kilo's device key under `Options.KiloDeviceKey`, are returned in the
`Result` for the caller to store. After a cancel during the listing,
`ResolveCredential` returns the context's error and no Result (C9), and
`ConfigureLLM` returns it at the model question: re-run,
`TestCancelDuringListingReturnsNoKey_0081` passes at `b109cea` for a typed
Grok key through both. With no static catalog (a third party's descriptor;
every built-in key provider has one, read by listing the ten built-in
descriptors), `ConfigureLLM` asks for a model id on the ended context and
returns the key with no error: re-run, "err <nil>, key returned true, model
ids asked 1" (`TestCancelledConfigureReturnsNoKey_0081`).

**`x/signin`.** `Wait` saves nothing: "the caller saves" (C20). The engine
writes only K9's client record. A context that ends during the engine's
`Wait` returns the context's error, so a caller that then saves has a
credential only if the sign-in finished first; gobble's ACP path checks its
context before its save (G5, G6).

**A token the provider issued.** It exists at the provider whether or not it
is saved. No path revokes a new session that is not saved, neither a
refused one nor a cancelled one; `auth.RevokeOAuthSession` (C21) exists for
OpenAI's, Grok's and the plan's sessions, and Kilo's device token is not
revocable (C22).

### `ConfigureLLM` after the credential step, and what saving last changes

**What runs between the step and the return, at `b109cea`** (read; the
model stage re-run as K10):

1. The credential step, which saves (C10), then returns the Result and the
   catalog (C23 calls it).
2. With an empty catalog (no static catalog and a failed listing, or Ollama
   with nothing installed): the model-id question (C15), with no context
   check before it; a blank answer is the error "no model available"; else
   `res.Model` is set and the Result returned (C24).
3. Otherwise `selectModel` (C16): its loop checks the context at its top
   (C25), then asks the search (C26). A blank answer goes to the
   recommended menu (C27), which checks nothing before it returns; "Other"
   goes to the model id (C28), which checks nothing either. A query goes to
   the matches, a loop that checks at its top too.
4. With `Options.NeedFallbacks`, `selectFallbacks` (C29): its loop checks at
   its top (C30); a blank round's choice ends it at once (C31).
5. The return (C32). The Result was assembled in the step, from the
   session as it stood at the sign-in.

None of those steps reads the store, the session or the network: the search
is over the catalog already listed (`catalog.Catalog.Search`, C33, and
`catalog.Search`), the menus are the catalog's ids, and no model-stage code
asks the session for a token (read, `wizard/model_select.go` whole). No
endpoint or other prompt runs after the step; the endpoint is asked before
it.

**What the person and a caller can observe after the change** (D5):

* A sign-in completed at the provider, then abandoned at a model or
  fallback question, by a cancel, a Prompter's error (N2) or a killed
  process, is lost: the store keeps what it held, and the person signs in
  again. Before, the new session was kept.
* While the model questions run, the new session is in memory only. Nothing
  in that stage needs it in the store (above). `Options.Existing` on a later
  run reads what the store holds, the earlier session, which matches the
  caller's configuration, since `ConfigureLLM` returned no Result.
* The plan's client record (K9) is saved at the sign-in, as before; after an
  abandoned run it may name the client issued for the session that was not
  kept. A later plan sign-in reuses the stored plan session's own client
  first, and the record only when there is no plan session (0075-MADR D7),
  so this changes nothing a person sees; the same is true today after a
  refused sign-in.
* **A refresh during the questions.** The model stage starts none. A refresh
  the listing started, still running when the questions begin, completes
  detached (0026-MADR F3) and is adopted into the in-memory session; with no
  store bound, it saves nothing (C34). At the end, `save` waits for it, as
  today, and the store gets the rotated pair: re-run,
  `TestRotationDuringQuestionsSavedAtEnd_0081`, one save of the rotated
  refresh token, the store empty while the model was asked. **So yes, there
  is a window in which a rotated refresh token exists only in memory**: from
  the rotation to the end of the questions, human time where today it is the
  rest of the listing. If the process dies there, the rotated pair is gone.
  What it belonged to is the new session, which saving last means not to
  keep on an abandoned run; the store still holds the earlier session,
  untouched by that rotation (a different grant, never reused). No
  previously saved credential is lost by the rotation.
* **The one case that could lose a working credential** is not the
  rotation but the provider: if a provider ends the earlier grant when it
  issues a new one, a person who abandons the run at a model question keeps
  an earlier session that no longer works. Nothing in this repository
  measured that. The reference ChatGPT client revokes tokens only on logout
  (K11, read in the codex checkout), not on a new sign-in; the vendors end a
  refresh-token family on reuse (0016-MADR), which a new sign-in is not. The
  same premise already stands under 0059-MADR Q2 (a), a refused sign-in
  keeping the earlier session. Saving last makes the case reachable by an
  ordinary abandoned run. It was N1; the owner ruled it measured before
  the release (D12).

## Decision Drivers

* A cancel means the user does not want the sign-in: the store, the only
  copy of a session (0016-MADR A10), is as it was before the run, as a
  refused sign-in already leaves it (0059-MADR W2).
* An error and the store agree: a run that returns an error has saved
  nothing in that run, as far as a check can make it so (K7 says where it
  cannot); a Result means the session is saved.
* One rule for every provider and every session kind, and one save path
  for both entry points.
* No change to the v1 API (R48) and none to the `TokenStore` contract,
  which callers implement (gobble's store does).
* A refresh's rotation is never lost from what was saved (0026-MADR F3;
  0059-MADR's amendment).
* What a caller already handles stays what it sees: the context's error,
  matching `context.Canceled` or `context.DeadlineExceeded`.

## Considered Options

Where the guard goes:

* **S1.** A context check immediately before the save, in the one function both entry points save through (the ruling's mechanism).
* **S2.** `save` refuses on an ended context whatever `Token` says.
* **S3.** The store's `Save` honours the context.
* **S4.** In the sign-in engine: `x/signin`'s `Wait`, or `internal/signinflow`, refuses to yield a credential once its context has ended.
* **S5.** Nothing changes; the behaviour is documented.

Where `ConfigureLLM` saves:

* **P1.** In the credential step, right after the check, as `ResolveCredential` does (the draft's Q4 (a) and (b)).
* **P2.** At `ConfigureLLM`'s very end, after the model and fallback questions (the owner's answer to Q4).
* **P3.** In the credential step, and the earlier session put back when a later question fails.

## Decision Outcome

Chosen options: **"S1"** and **"P2"**. S1 is the ruling's mechanism and the
answers to Q1 to Q3; P2 is the owner's answer to Q4. One check right before
the save closes K2 to K4, K6 and K10 for every session kind and every key;
saving last makes `ConfigureLLM`'s error mean nothing was saved. D1 to D8
and D12 are the ruling and the owner's answers to Q1 to Q7 and N1 to N4;
D5's one save path is the coordinator's application of N5 (a); D9 to D11
follow from them. How the probe shows the person what a sign-in needs is
the coordinator's application of N6 (a) (D12).

### D1. The ruling

* A record of its own; a cancelled sign-in saves nothing; a context check
  before the save; a test for each session kind; it may share 0080's
  release.

### D2. Every window before the write (Q1, decided)

One check, the last thing before the save: K2, K3, K4, K6 and K10 save
nothing. K5 keeps `save`'s existing wait. K7 is out of reach of a context
check and is named, not closed: a write that has begun completes.

### D3. Every kind, typed keys included (Q2, decided)

On an ended context, before the save, the run returns the context's error
and no Result, whatever the credential: a new session, a key typed or read
from the environment or `Options.Existing`, Kilo's device key, a kept
session. A kept session is not saved by the run either way; its refreshes
save through its own store, unchanged.

### D4. A deadline is a cancel; the listing's own limit is not (Q3, decided)

The run's context ending by cancel or by deadline saves nothing. A listing
that runs out of `Options.DiscoverLimit` is an outage and the session is
saved, as 0059-MADR's amendment and 0080-MADR D6 decide.

### D5. Where each entry point saves (Q4, N2 and N3, decided; N5 applied)

* **`ResolveCredential`** saves inside the shared step, immediately after
  the step's own check. Its own check after the step goes, so its error
  means nothing was saved and its Result means the session is saved (the
  first option). A cancel after the write returns the Result.
* **`ConfigureLLM`** saves as the last thing it does, after the model and
  fallback questions, behind the same one check, before each of its two
  successful returns (the model-id branch and the end). Any cancel inside
  it saves nothing.
* **One save path** (N5 (a), the coordinator's application): the step's body becomes an unexported step
  that resolves and checks the credential and returns it unsaved;
  `credentialStep` keeps its signature and is that step followed by
  `commitCredential`, which is the check and `save`; `ConfigureLLM` calls
  the unsaved step and `commitCredential` at its end. `save` and its wait
  for a refresh are unchanged; there is one function that saves.
* **A check after the step in `ConfigureLLM`** (N3, decided): a context that has
  ended by the end of the step stops the run there, with the context's
  error, before any model question; it saves nothing, as the final check
  would.
* **Errors that are not a context's end** at the model stage (N2,
  decided): the run returns them and saves nothing.
* The Result's token facts (`TokenExpiry`, `Issuer`, `ClientID`,
  `AccountID`, `FedRAMP`) stay as the step assembled them, as today.

### D6. The issued token (Q5, decided)

Out of scope, and the docs say so: nothing is revoked, as for a refused
sign-in today. An unsaved session lapses at the provider by its own expiry.

*Annotation, 2026-11-20:* the last sentence is false for Kilo. Kilo's device
token has no refresh and no expiry (`wizard/auth.go:418-419`), and the SDK
cannot revoke it (C22), so an unsaved Kilo device sign-in leaves a token on
the account until the person removes it. Saving last makes that reachable by
every abandoned Kilo `ConfigureLLM`. The owner, relayed by the coordinator,
ruled "Accept it; the docs say so plainly for Kilo (Recommended)": the
behaviour does not change, and the docs say it for Kilo (the amendment of
2026-11-20, row S60).

### D7. `x/signin` and the plan's client record (Q6, decided)

Unchanged. `Wait`'s doc gains one sentence: a caller saves only while its
own context has not ended, as the wizard does. K9's client record is
0075-MADR D7's and stays.

### D8. The release (Q7, decided) and the changelog sections (N4, decided)

In the next release with 0080, on the owner's word; the tag and the GitHub
Release are each the owner's word. The sections: **Fixed**, "a
sign-in whose run ends before the wizard saves it is not saved", and
**Changed**, "`ConfigureLLM` saves a new session as its last step, after
the model and fallback questions; a run abandoned at one of them keeps the
session the store held".

### D9. The error and Result contract, after the change

| A cancel (or the run's deadline) arrives | `ConfigureLLM`: saved / returns | `ResolveCredential`: saved / returns |
| :--- | :--- | :--- |
| before the run | nothing / `wizard: <ctx>` (C8, as today) | nothing / `wizard: <ctx>` (as today) |
| during the sign-in, before the provider issued anything (K1) | nothing / the login's error, matching the context's (as today) | the same |
| after the issue, before or during the check: the listing, Kilo's profile (K2 to K4) | nothing / `wizard: <ctx>` from the check after the step (N3) | nothing / `wizard: <ctx>` from the check before the save |
| at a model or fallback question (K10) | nothing / `select model: <ctx>` or `select fallbacks: <ctx>` at a loop's top, or `wizard: <ctx>` from the final check when the question was answered | (asks none) |
| a Prompter's error, or no model entered, at the model stage | nothing / that error (N2) | (asks none) |
| between the final check and the write (K6's last instant) | saved / the Result | saved / the Result |
| during the write (K7) | saved, the write completes / the Result, or the store's error as today | the same |
| after the write | saved / the Result: nothing runs after it | saved / the Result (its own last check removed) |

`errors.Is` matches `context.Canceled` or `context.DeadlineExceeded` in
every row that returns a context's error. No new sentinel, no new name.

### D10. What does not change

* `save`'s wait for a refresh and its refusal of a refresh the issuer
  refuses (0059-MADR's amendment).
* The `TokenStore` and `ClientStore` contracts, and `FileTokenStore`.
* 0080's checks: its D3 ("No other provider's check changes") stays true.
* `x/signin`'s behaviour (D7). No exported name.

### D11. The records noted

A dated one-line note pointing here, each in the PLAN's code phase:

* [0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md):
  W2 (the save's position: `ConfigureLLM` now saves after its model
  questions), its amendment "a refresh can outlast the listing (Phase 2)"
  ("nor [...] when the run's context ends first" holds for every window
  before the write), and W8 (`ConfigureLLM` asks no model question on a
  context that ended in the step).
* [0056-MADR-wizard-credential-only-entry-point.md](0056-MADR-wizard-credential-only-entry-point.md)
  D1: `ResolveCredential`'s last context check moves into the step.
* [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  D1 ("the store is unchanged until the check passes (W2)"): for
  `ConfigureLLM`, until the run ends; and 0080-PLAN's "Found, not mine to
  change", answered here.

### D12. The earlier grant, measured before the release (N1, decided)

* Before Phase 2's code lands, the owner runs a live probe (PLAN Phase 1)
  for four sign-ins: the ChatGPT browser sign-in, the ChatGPT plan, Grok's
  browser sign-in and Kilo's device sign-in. Each signs in twice in one
  test process and then uses the first session.
* "Uses" is the request that would fail if the provider had ended the first
  grant: one refresh of the first session for the three that refresh (each
  refresh token used exactly once, so no reuse can be detected), Kilo's
  profile for Kilo, which does not refresh. A baseline use of the first
  session before the second sign-in, and a control use of the second
  session after, tell an ended grant from a failure of the probe or the
  service.
* The sessions live in the test's memory only: no store, the owner's or
  any other, is opened. The output is status codes, counts and one verdict
  per provider, never a token, a code, a URL with a code, an account or a
  client id; an offline twin proves it on planted secret-shaped values.
* What the person must see goes elsewhere (N6 (a), the coordinator's
  application): each authorize URL, and Kilo's verification page, are
  opened in the browser by the test and shown nowhere; Kilo's user code is
  written to the terminal device only. If either cannot be done, that
  provider's run stops with a plain message holding neither.
* A verdict "ended" for any provider stops the PLAN, and it comes back to
  the owner. "Indeterminate" twice in a row does the same.
* What the probe cannot avoid: if a provider does end earlier grants, the
  probe's own sign-ins end the owner's other sessions with that provider
  (in other apps, the vendor's own CLI included); the PLAN tells the owner
  before the run. The OpenAI and Grok device sign-ins are not probed; they
  reach the same issuers as the browser sign-ins, which is named in "Not
  verified".

### Consequences

* Good, because a run that ends with an error leaves the store as it was,
  for every provider, session kind and key, and a Result means the session
  is saved, through either entry point.
* Good, because `ConfigureLLM` no longer returns a Result after the run's
  context ended (K4, K10), nor asks a model question on a context that
  ended in the step.
* Good, because gobble's terminal path gets the fix by a pin bump, and
  gobble's three statements (G1 to G3) become true of the wizard.
* Good, because there is one save path and no new API.
* Neutral, because the store and the caller's configuration now change
  together: `ConfigureLLM` writes the session only when it hands back the
  Result the caller saves.
* Bad, because a person who completes a sign-in and then abandons the model
  questions, by Ctrl-C, a closed terminal or an error, must sign in again;
  the provider's grant is left to lapse (D6).
* Bad, because the rotated pair of a refresh that completes during the
  questions lives only in memory until the end; a process that dies there
  loses it, with the new session it belonged to.
* Bad, possibly, because if a provider ends the earlier grant on a new
  sign-in, an abandoned run leaves a stored session that no longer works.
  The owner measures it before the release (D12); a provider that does
  stops the PLAN.
* Bad, because the instant between the final check and the write, and the
  write itself (K7), cannot be closed by a context check; a cancel there
  ends in a saved session and a Result.
* Bad, because it is the larger change: `ConfigureLLM`'s structure, its
  docs and the README's account of when a session is saved all change.

### Confirmation

* **Tests**, in `wizard`, against in-process stand-ins (no outside host,
  0065-MADR), each red at `b109cea` and at the commit carrying 0080's code
  with `tools/redgreen.py --rev`, the guards green, and each seen to fail
  on a planted break with `tools/plant.py`, on scratch clones:
  * the eight of the first draft: a cancel during the listing for each
    session kind (Kilo device, Grok device and browser, OpenAI browser and
    device, a pasted ChatGPT access token, the ChatGPT plan) through both
    entry points; after the issue, with and without `Discover`; during
    Kilo's profile read; the run's deadline; a typed key with no static
    catalog; a cancel inside the store's `Save` (`ResolveCredential`'s
    Result); and two guards (a typed key during the listing; a refresh in
    flight);
  * seven for saving last: a cancel at the model search, the model menu,
    the model id and the fallbacks; a typed key and a cancel at the model
    menu; a cancel at the manual model id; a run
    that ends well saves exactly once, with the store empty while the model
    is asked; a Prompter's error at the model question saves nothing;
    `ResolveCredential` saves once and asks no model (a guard); a refresh
    the listing left running is saved rotated, at the end.
  * Planted breaks the prototype was seen to catch, seven of seven: the
    final check removed; the check for a session only; a cancel but not a
    deadline; `ConfigureLLM` saving in the step again; the model-id branch
    not committed; `ResolveCredential`'s own last check put back; the
    check after the step removed (N3 (a)).
* **`make api-check`** reports no change, and `tools/apidiff.py v1.31.0`
  lists nothing for this record.
* **`tools/claims.py`** on this record and its PLAN; the gates the PLAN
  names.

## Pros and Cons of the Options

### S1. A context check immediately before the save

* Good, because one check, in the one function both entry points save
  through, closes every window before the write for every kind.
* Good, because it covers a key, which `save` never sees (D3).
* Good, because it needs no API and touches no store.
* Bad, because K7 stays open, as it does in every option but S3, and S3
  closes it only for this module's store.

### S2. `save` refuses on an ended context whatever `Token` says

* Good, because it is the smallest change, in the function the defect is
  in.
* Bad, because `save` does nothing for a key: the scratch prototype of S2
  fails `TestCancelledConfigureReturnsNoKey_0081` as `b109cea` does.

### S3. The store's `Save` honours the context

* Good, because it would close the instant before the write for
  `FileTokenStore`.
* Bad, because gobble's terminal path uses its own store (gobble-cli
  `llm/provider/login.go`), so the defect stays there.
* Bad, because a change to the store's contract is one every implementer
  must follow, and a write that has begun still completes.

### S4. In the sign-in engine

* Bad, because the listing and the model stage run after the engine has
  returned; only K2 is the engine's.
* Bad, because the wizard waits for its sign-in under a detached context on
  purpose (C12); a rule in the engine would not reach the wizard's save.

### S5. Nothing changes; the behaviour is documented

* Good, because it costs nothing now.
* Bad, because a store would keep holding a session the user cancelled, and
  gobble's records state the opposite (G1 to G3).

### P1. `ConfigureLLM` saves in the step, right after the check

* Good, because a sign-in the person completed survives an abandoned model
  question, and nothing lives only in memory beyond the listing.
* Bad, because `ConfigureLLM`'s error would not mean "nothing saved": a
  cancel, or an error, at a model question returns an error with the new
  session in the store and the caller's configuration unchanged.

### P2. `ConfigureLLM` saves at its very end (chosen, Q4)

* Good, because any error from `ConfigureLLM` means nothing was saved, and
  a Result means the session is, so the store and the caller's
  configuration change together.
* Good, because it closes K10, which today returns a Result after a cancel.
* Bad, because an abandoned run loses a sign-in the provider completed, and
  a rotation during the questions lives only in memory (above).
* Bad, because it is the larger change, though the prototype shows it
  without an exported change or an existing test's change (R5).

### P3. Save in the step, and put the earlier session back on a later failure

* Good, because a crash during the questions would leave the new session.
* Bad, because it writes twice and needs the earlier session read and kept
  for the undo; a crash between the two writes leaves the new one while the
  run failed, and a `Delete` or a restore can fail. 0059-MADR Q2 (b), the
  same shape for a refusal, was not chosen.

## Open questions for the owner

None.

## More Information

### Related

* [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md):
  its D3 and D6; its PLAN's execution record names this gap (R3).
* [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)
  D7: the client record (K9).
* [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md):
  `x/signin`'s `Wait` saves nothing (D7 here).
* [0026-MADR-remediate-v1-2-debugging-pass-findings.md](0026-MADR-remediate-v1-2-debugging-pass-findings.md)
  F3: a refresh runs detached, so its rotation is adopted.
* [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md):
  both vendors end a refresh-token family on reuse (N1).
* gobble-cli
  `docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`,
  "Review and fixes" (2026-11-16), finding B3: the same defect on gobble's
  ACP path, fixed there by a context check before its save (G5) and before
  its key's store (G6).

### What the callers see

* **gobble-cli, terminal path** (read at `cecc892`): `SignIn` calls
  `wizard.ResolveCredential` (G8, G9), with `Discover`, a `TokenStore` over
  its `auth.json` and `KiloDeviceKey`, and nowhere calls `ConfigureLLM`
  (read: `git grep` at `cecc892`). So saving last changes nothing it sees.
  What changes is the first part of this record: on a cancel it returns
  the context's error, as today (G7), and its `auth.json` is no longer
  written (its store takes a free file lock whatever the context,
  `internal/filelock/lock.go`, `Lock`); a cancel after the save gives it a
  Result where it had the context's error, with the session indeed saved.
* **gobble-cli, ACP path**: unchanged by this record; it has its own checks
  (G5, G6).
* **prepare-commit-msg** (read at `2715f2d`; it pins the SDK at `v1.29.0`):
  its setup calls `wizard.ConfigureLLM` with `Discover`, `NeedFallbacks` and
  a `TokenStore` (M1), and on a `CredOAuth` Result checks the stored
  session with `config.ValidateOAuth` (M2) before it saves its own
  configuration. After a pin bump: the store is written before
  `ConfigureLLM` returns, so that check still finds the session; an error
  from `ConfigureLLM` now leaves the store as it was, which matches its
  unsaved configuration; it sets no signal context (no `NotifyContext` in
  its code), so Ctrl-C ends the process: killed during the model or
  fallback questions, the new sign-in is lost and the earlier session kept,
  where today the new one was kept. A failure in its own questions after
  `ConfigureLLM` returns is outside this record.

### Found beyond the question asked

* A cancel while the last model-stage question is on screen, answered all
  the same, returns a Result with no error today (K10): the loops check the
  context only at their tops. D2 closes it by the final check.
* The Result's `TokenExpiry` is the session's at the sign-in; a refresh
  during the listing makes it stale, today and after this change (D5). Not
  changed here.

### Evidence index

<!-- claims: self=b109cea gobble-cli=cecc892 prepare-commit-msg=2715f2d codex=57e64682f8 kilocode=58072bd696 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | The save | `wizard/auth.go:991` | `func (c resolvedCredential) save(ctx context.Context, d llmprovider.Descriptor) error {` |
| C2 | refuses on an ended context only when `Token` fails | `wizard/auth.go:995` | `if _, err := c.session.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure)` |
| C3 | and then writes | `wizard/auth.go:999` | `if err := c.saveTo.Save(ctx, d.ID, c.session); err != nil {` |
| C4 | `Token` returns a current token without reading the context | `llmprovider/auth/oauth_session.go:164` | `if current && !retrySave {` |
| C5 | it reads the context before it starts a refresh | `llmprovider/auth/oauth_session.go:182` | `if err := ctx.Err(); err != nil {` |
| C6 | `FileTokenStore.Save` | `llmprovider/auth/tokenstore_file.go:158` | `func (fs *FileTokenStore) Save(ctx context.Context, provider llmprovider.ProviderID, s *OAuthSession) (err error) {` |
| C7 | writes by rename | `llmprovider/auth/tokenstore_file.go:226` | `if err := tokenStoreRename(tmpName, final); err != nil {` |
| C8 | `ConfigureLLM`'s first context check | `wizard/configure.go:276` | `if err := ctx.Err(); err != nil {` |
| C9 | `ResolveCredential`'s last context check, after the step | `wizard/configure.go:378` | `if err := ctx.Err(); err != nil {` |
| C10 | The step's save, after the listing loop | `wizard/configure.go:514` | `if err := credential.save(ctx, d); err != nil {` |
| C11 | The listing's own limit | `wizard/configure.go:834` | `dCtx, cancel := context.WithTimeout(ctx, limit)` |
| C12 | The wizard waits for its sign-in under a detached context | `wizard/driver.go:86` | `result, err := run.Wait(context.WithoutCancel(ctx))` |
| C13 | Kilo's profile read for the menu; its failure a warning | `wizard/auth.go:403` | `could not read the Kilo account's organizations (%s); using the personal account` |
| C14 | A listing's failure that is not a refusal is not returned as one | `wizard/configure.go:870` | `if stepRefusal(err) {` |
| C15 | `ConfigureLLM` asks for a model id when the catalog is empty | `wizard/configure.go:297` | `manual, inputErr := p.Input("No models found; enter a model id", existingModel(o, d.ID))` |
| C16 | then the model question | `wizard/configure.go:311` | `if res.Model, err = selectModel(ctx, p, d, cat, o); err != nil {` |
| C17 | 0080 changes no other provider's check | `docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md:329` | `* No other provider's check changes.` |
| C18 | The engine saves the plan's issued client as the login returns | `internal/signinflow/signinflow.go:443` | `clientSaveErr = spec.ClientStore.SaveClient(ctx, spec.Provider,` |
| C19 | `FileTokenStore.SaveClient` ignores the context | `llmprovider/auth/clientstore.go:82` | `func (fs *FileTokenStore) SaveClient(_ context.Context, provider llmprovider.ProviderID, c ClientRecord) error {` |
| C20 | `x/signin`'s `Wait` saves nothing | `llmprovider/x/signin/signin.go:253` | `// saves nothing: the caller saves. It returns ctx's error when ctx ends first.` |
| C21 | A session can be revoked | `llmprovider/auth/oauth_revoke.go:37` | `func RevokeOAuthSession(ctx context.Context, session *OAuthSession) error {` |
| C22 | Kilo's device method is not marked revocable | `llmprovider/x/signin/signin.go:142` | `ClientIdentity: factNone, Slot: "kilo"}},` |
| C23 | `ConfigureLLM` calls the step | `wizard/configure.go:289` | `res, cat, err := credentialStep(ctx, p, d, o)` |
| C24 | The model-id branch returns the Result | `wizard/configure.go:307` | `res.Model = manual` |
| C25 | `selectModel` checks the context at its loop's top | `wizard/model_select.go:52` | `if err := ctx.Err(); err != nil {` |
| C26 | then asks the search | `wizard/model_select.go:55` | `q, err := p.Input(searchModelsPrompt, "")` |
| C27 | A blank search goes to the menu, which checks nothing | `wizard/model_select.go:69` | `return selectRecommended(p, d, cat.Recommended, o)` |
| C28 | The model id checks nothing | `wizard/model_select.go:208` | `manual, err := p.Input("Model id", existingModel(o, provider))` |
| C29 | `ConfigureLLM` asks the fallbacks | `wizard/configure.go:315` | `if res.Fallbacks, err = selectFallbacks(ctx, p, d, cat, res.Model, existingFallbacks(o, d.ID)); err != nil {` |
| C30 | which checks the context at its loop's top | `wizard/model_select.go:249` | `if err := ctx.Err(); err != nil {` |
| C31 | and a blank round's choice ends it at once | `wizard/model_select.go:277` | `return chosen, nil // a blank round ends the loop` |
| C32 | `ConfigureLLM` returns | `wizard/configure.go:319` | `return res, nil` |
| C33 | The search is over the catalog in hand | `llmprovider/catalog/search.go:205` | `func (c Catalog) Search(query string) []Match {` |
| C34 | A refresh saves only when a store is bound | `llmprovider/auth/oauth_session.go:994` | `if state.store != nil {` |
| K11 | The reference ChatGPT client revokes on logout | `codex:codex-rs/login/src/auth/manager.rs:1094` | `pub async fn logout_with_revoke(` |
| K12 | Kilo's own client opens its verification page (and prints the code) | `kilocode:packages/kilo-gateway/src/auth/device-auth.ts:52` | `await open(verificationUrl).catch((err) => {` |
| G1 | gobble's note: the wizard does not save after a done context | `gobble-cli:llm/provider/signinrun.go:146` | `save fails the sign-in with it, as the wizard does not save then. Wait does` |
| G2 | gobble's note, on its key | `gobble-cli:internal/cli/agentauth.go:103` | `// nothing once its context is done (0051-MADR D4 step 3).` |
| G3 | gobble's decision record says the same | `gobble-cli:docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:368` | `is not saved; and it saves nothing once its context is done.` |
| G4 | gobble copied the wizard's conjunction | `gobble-cli:llm/provider/signinrun.go:178` | `if _, err := sess.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure)` |
| G5 | and checks the context itself before it saves | `gobble-cli:llm/provider/signinrun.go:182` | `if err := ctx.Err(); err != nil {` |
| G6 | and before it stores a key | `gobble-cli:internal/cli/agentauth.go:104` | `if err := ctx.Err(); err != nil {` |
| G7 | Its terminal path returns the context's error | `gobble-cli:llm/provider/signin.go:369` | `return Credential{}, ctx.Err()` |
| G8 | Its sign-in is the wizard's `ResolveCredential` | `gobble-cli:llm/provider/signin.go:95` | `var resolveCredential = wizard.ResolveCredential` |
| G9 | called by `SignIn` | `gobble-cli:llm/provider/signin.go:345` | `res, _, err := resolveCredential(ctx, w, wizard.Options{` |
| M1 | prepare-commit-msg's setup calls `ConfigureLLM` | `prepare-commit-msg:internal/ui/setup.go:274` | `res, err := wizard.ConfigureLLM(ctx, prompter, wizard.Options{` |
| M2 | and checks the stored session afterwards | `prepare-commit-msg:internal/ui/setup.go:309` | `if err := config.ValidateOAuth(ctx, string(res.Provider), pc, store); err != nil {` |

### Not verified

* N1: whether any provider ends the earlier grant when it issues a new
  one. The owner measures it in PLAN Phase 1, for four sign-ins; the
  OpenAI and Grok device sign-ins are not probed.
* *Annotation, 2026-11-20 (PLAN Phase 1c):* N1 is no longer unverified for
  the four sign-ins probed: each was measured `kept` on 2026-11-19 (the
  amendment of 2026-11-20, "D12: the four verdicts"). It stays unverified for
  the OpenAI and Grok device sign-ins, which were not probed, and for a grant
  that ends minutes after a new sign-in, which the probe cannot catch.
* The ChatGPT plan session's cases were not run while drafting (they need
  the consent question scripted); the PLAN's tests run them.
* A sign-in through the real `auth` device and browser flows with an
  in-process issuer was not run; the reproduction stubs the wizard's login
  seams, as the package's tests do.
* K1 and K7 were read, not run; a process killed during the model questions
  was reasoned from the code, not run.
* No `-race` run of the reproduction: the session host's heavy queue held
  another agent's runs, and none was queued.
* The staged-tree runs read 0080's index on 2026-11-17 while another agent
  was still changing it; the PLAN reruns everything on the commit that
  carries 0080's code.
* The exact words the brief attributed to gobble's B3 note were not found
  (R4); the reconciliation answers the statements that were.

## Amendment of 2026-11-18: record 0082 changed the save path before this record's code was built

Added after the record was accepted and before any of its code; nothing above
is rewritten or removed, and no ruling in "Decided" changes. The facts are
those of `a7e924f` (the tip of `main` on 2026-11-18, CI green), re-read for
this amendment; the record above was written against `b109cea` (D11's notes
and the evidence index were taken there). Two rulings of the owner, relayed by
the coordinator on 2026-11-18, are recorded below; their labels are quoted.

### What record 0082 landed in the save path

[0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md](0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md)
was built in the five commits `8051b00` ("refuse to adopt another sign-in at
refresh"), `04433ae` (third-review rulings), `5f2c2ab` (fourth-review
rulings), `1545172` and `fd986c4` (the over-size client record). It changed
the very function this record's S1 puts its check in. At `a7e924f`, in
`wizard/auth.go`:

* `save` (`:1103`) takes a `Prompter`. It keeps the first check (`:1107`, the
  session's `Token`, C2), then, for a store that is an `auth.RefreshLocker`,
  loops: `writeUnderLock` (`:1142`) takes the store's refresh lock through
  `lockToSave` (`:1161`), writes with it held, and releases it. A held lock is
  announced to the person, waited for until the store's own bound or the
  context's end, and the person is offered a retry at the bound. A store that
  is not a `RefreshLocker` is written by `write` directly (`:1112`).
* `write` (`:1180`) is the one place the new session is given its store and
  saved. **Its first statement is a context check** (`:1181`): an ended
  context returns "wizard: the new <label> session was not saved: <ctx>" and
  writes nothing. The session gets its store (`UseStore`, `:1184`) only after
  that check, under the lock (0082's N-3), then `Save` (`:1185`, C3).
* A context that ends during the lock's wait is already handled: `lockToSave`
  returns the context's error, and `save` returns it as "the new session was
  not saved" (`:1119` to `:1121`) without writing.
* The doc comment on `save` (`:1100` to `:1102`) names this record: "the
  context is checked after the lock is held, before anything is written: a
  lock that outlasts the sign-in saves nothing (0081-MADR's single check
  before the save belongs here, after the lock, and not before the wait)".

0082's note addressed to this record is N-6 of its second amendment, "N-6,
0081" (`0082-MADR-refuse-to-adopt-another-sign-in-at-refresh.md`, line 1145):
the check this record adds immediately before the save must stay after the
lock is obtained, in `write`, which every pass of the retry loop runs after
taking the lock, and not before the wait, since a check before the wait would
not stop a sign-in cancelled during a wait of up to a minute from saving when
the lock comes free. At `a7e924f` that check **exists** in `write`, built by
0082 for its own reason, and the cancel windows below that lead to a
session's save are closed by it.

### What is no longer true at `a7e924f`

Each statement is of this record as written; the annotation says what holds
now. Every decision, D1 to D12, stands; what changed is the facts. The facts
not listed hold: K1, K5, K7, K8, K9, K10, "A key", and the list of what a
person and a caller observe after the change.

| Where | As written | At `a7e924f` |
| :--- | :--- | :--- |
| Context and Problem Statement | when the context ends during the listing that checks a fresh sign-in, the wizard saves the new session and then returns the context's error | for a session, no: `write`'s check refuses, nothing is saved, the context's error is returned. For a key it is as written |
| "Every window" intro | a run whose own deadline passes during the listing saves today as a cancelled one does | no longer, for a session (`TestDeadlineDuringListingSavesNothing_0081` is green) |
| R1 | all twelve cases (six kinds, two entry points) save once; the three OpenAI `ConfigureLLM` cases also return no error | none of the twelve saves; each returns the context's error. Green at `a7e924f` (measured below) |
| R2 | the check refuses only when `Token` also fails; "the save is then made (C3)"; "so the defect is every session kind's" | the first sentence holds of `save`'s first check (C2, now `:1107`), but `write` checks the context again before it writes (C3, now `:1185`), so a session's save is refused on an ended context whatever `Token` said. The defect is not a session kind's any more; a key's is as written (a key reaches no `save`) |
| R3 | 0080 does not change it; R1's twelve cases fail the same way on 0080's tree | true of 0080, which changed no save; 0082 did (above) |
| R4 | the path without the check is a current token, every fresh sign-in and every Kilo session; "which is why gobble's ACP path saves nothing after a cancel and this module's wizard does"; "a current one is [saved] (R1)" | that path has the check, in `write`; the wizard no longer saves a session on an ended context. G1 to G3 (gobble's statements that the wizard saves nothing once its context is done) are true of the wizard for a session; for a key the wizard saves nothing at all and returns it, as written |
| R5 | the prototype's fifteen tests: twelve fail and three guards pass | at `a7e924f` eight fail and seven pass (below). The prototype was written against `b109cea`; its `commitCredential` is rebuilt at the base, and its check, for a session, is the second one (below) |
| K2 | the issued session comes back and is saved, with or without `Discover` | not saved (`TestCancelAfterIssueSavesNothing_0081`, both cases green) |
| K3 | Kilo's profile read ends with the context, the personal account is used, then saved | not saved without `Discover` (`TestCancelDuringKiloProfileSavesNothing_0081` green); with `Discover` it was already the context's error (0080's `checkEnded`) |
| K4 | the listing's end: warning, static catalog, loop ends, **saved**; `ConfigureLLM` returns the Result with no error for the three OpenAI cases | not saved, for a session, in all twelve cases; each returns the context's error. For a key with no static catalog, `ConfigureLLM` still returns the key with no error (`TestCancelledConfigureReturnsNoKey_0081` red) |
| K6 | between the check and the write nothing reads the context; **saved** | the context is read in `write` immediately before `UseStore` and `Save`; what remains is the instant after that read, which is K7's |
| D2 | "One check, the last thing before the save: K2, K3, K4, K6 and K10 save nothing" | K2, K3, K4 and K6 hold for a session through `write`'s check. K10 and a key do not yet (below) |
| D9, third row | "`wizard: <ctx>` from the check before the save" for a session | a session's error is `write`'s or `save`'s ("the new <label> session was not saved: <ctx>") until the commit check is built; D9's closing sentence stands, since `errors.Is` matches the context's error in every row. The commit check's own error keeps D9's `wizard: <ctx>` |
| Confirmation, Tests | each red at `b109cea` and at the commit carrying 0080's code | four are green at `a7e924f`: they are guards (below) |

### The measurement at `a7e924f`

The drafted fifteen tests, from the drafting material, renamed with this
record's number, were placed in `wizard/` on a scratch copy of `a7e924f` (a
`git archive`, never the tree) and run with `go test -count=1 -run '_0081$'
./wizard/`. They compiled unchanged. In-process stand-ins only; no outside
host; the tree was not touched.

| Test | At `a7e924f` | Red line, or why green |
| :--- | :--- | :--- |
| `TestCancelDuringListingSavesNothing_0081` (twelve cases) | green | `write`'s check |
| `TestCancelAfterIssueSavesNothing_0081` (two cases) | green | `write`'s check |
| `TestCancelDuringKiloProfileSavesNothing_0081` | green | `write`'s check |
| `TestDeadlineDuringListingSavesNothing_0081` | green | `write`'s check |
| `TestCancelDuringListingReturnsNoKey_0081` | green (guard at `b109cea` too) | a key reaches no `save`; `ConfigureLLM` returns the context's error at the model question |
| `TestCancelDuringRefreshSavesNothing_0081` | green (guard at `b109cea` too) | `save`'s first check, `Token` |
| `TestResolveSavesInTheStep_0081` | green (guard at `b109cea` too) | none needed |
| `TestCancelledConfigureReturnsNoKey_0081` | **red** | `cancel_save_0081_test.go:346: err <nil>, key returned true, model ids asked 1; want context.Canceled, no key, none asked` |
| `TestCancelAfterSaveReturnsTheResult_0081` | **red** | `cancel_save_0081_test.go:382: err wizard: context canceled, kind "", saves 1; want the Result of the saved session` |
| `TestCancelAtModelStageSavesNothing_0081` (four cases) | **red** | `save_at_end_0081_test.go:119: err <nil>, kind "oauth"; want context.Canceled and no Result` and `:122: a run cancelled at the model search saved 1 time(s); want none`, and the same for the model menu, the model id and the fallbacks |
| `TestCancelAtModelStageReturnsNoKey_0081` | **red** | `save_at_end_0081_test.go:143: err <nil>, key returned true; want context.Canceled and no key` |
| `TestCancelAtManualModelSavesNothing_0081` | **red** | `save_at_end_0081_test.go:166: err <nil>, kind "oauth", saves 1; want context.Canceled, no Result, none saved` |
| `TestConfigureSavesOnceAfterTheModel_0081` | **red** | `save_at_end_0081_test.go:185: saves 1, saves when the model was asked 1; want one, after the model` |
| `TestModelStageErrorSavesNothing_0081` | **red** | `save_at_end_0081_test.go:204: a run that failed at the model question saved 1 time(s); want none` |
| `TestRotationDuringQuestionsSavedAtEnd_0081` | **red** | `save_at_end_0081_test.go:250: saved refresh tokens ["rt-new"], saves when the model was asked 1; want one save, of rt-new, after the model` |

**The four tests that are now guards, each held by a planted break.** On a
second scratch copy of `a7e924f` with the same tests, `write`'s context check
(`wizard/auth.go:1181` to `:1183`) was removed and nothing else changed. The
tests then fail, each for its own reason:

* `TestCancelDuringListingSavesNothing_0081`: all twelve cases fail with
  "a sign-in cancelled during its listing saved 1 time(s); want none", and
  three of them also with "err <nil>; want one matching context.Canceled"
  (the OpenAI `ConfigureLLM` cases);
* `TestCancelAfterIssueSavesNothing_0081`: "a sign-in cancelled after the
  issue saved 1 time(s); want none", in both cases;
* `TestCancelDuringKiloProfileSavesNothing_0081`: "a sign-in cancelled during
  the profile read saved 1 time(s); want none";
* `TestDeadlineDuringListingSavesNothing_0081`: "a sign-in whose deadline
  passed during its listing saved 1 time(s); want none".

The three guards of the drafting (`TestCancelDuringListingReturnsNoKey_0081`,
`TestCancelDuringRefreshSavesNothing_0081`, `TestResolveSavesInTheStep_0081`)
are not held by this break, as none of them depends on `write`'s check; the
plants that hold them are the PLAN's Phase 3 list, built with the code.

### The owner's rulings of 2026-11-18

Relayed by the coordinator, each label quoted; what was asked is in this
record's words.

* **The order of work:** "As the plan says: build the probe now, I run it,
  then the code is built (Recommended)". As asked: an agent builds only the
  PLAN's Phase 1a, the probe and its offline twin; the owner is handed the
  four commands; the code change (PLAN Phase 2) waits for the owner's
  results, so nothing is built on the unmeasured assumption of N1; 0081
  pauses on the owner while the other plans of the SDK carry on. This is the
  PLAN's own order: its Phase 2 "does not land before Phase 1c is recorded".
* **The record's facts after 0082:** "Amend the record by dated note; keep the
  four tests as guards (Recommended)". As asked: this record says plainly that
  0082 already covers the session windows; the four tests stay as regression
  guards, each proven by a planted break on 0082's check; the code change
  becomes the key path plus saving last; a records-only change now. This
  amendment and the PLAN's matching entry are that change.

### What remains to build, against "Decided"

No ruling changes. D2 to D5 and N2, N3 and N5 (a) stand as written; what has
changed is how much of them the code already meets.

* **A key, and any credential with no session** (D3, Q2). `save` returns at
  once when it has no session or no store (`wizard/auth.go:1104`), so a
  typed, environment or saved key, and Kilo's device key, reach no check at
  all in the step: `ConfigureLLM` returns such a key with no error after the
  context ended (K4's key case, K10). This needs the check this record's
  `commitCredential` makes before it calls `save` (D5's one save path), whatever
  the credential. Evidence: `TestCancelledConfigureReturnsNoKey_0081` and
  `TestCancelAtModelStageReturnsNoKey_0081`, red above.
* **`ResolveCredential`'s own last check** (D5, Q4, first option;
  `wizard/configure.go:435`) still follows the step, so a cancel after the save
  returns the context's error with the session saved. It goes, as D5 says.
  Evidence: `TestCancelAfterSaveReturnsTheResult_0081`, red above.
* **`ConfigureLLM` saves in the step** (`credentialStep`'s `credential.save`,
  `wizard/configure.go:578`), not at its very end (D5, Q4's answer, P2), and
  checks nothing right after the step (N3). It must save as the last thing it
  does, before each of its two successful returns (`:351`, `:362`), after the
  check right after the step; any error saves nothing (N2). Evidence: the six
  red tests of `save_at_end_0081_test.go` and `TestCancelledConfigureReturnsNoKey_0081`.
* **How the check composes with the lock and the retry loop.** The session's
  check stays where 0082's N-6 puts it: in `write`, after the lock, on every
  pass of the loop; this record's builder does not move it, remove it, or add
  a second wait. `commitCredential`'s check (D5) runs before `save`, hence
  before the lock and its wait; it is the check for a key and the early
  refusal for a session, and `write`'s is the one that covers a wait. A
  session therefore passes two checks; neither duplicates the other's window,
  and a cancelled wait already saves nothing and returns the context's error
  (above). The PLAN's step 4 is unchanged by this.

Because the PLAN's Phase 2 does not start before the owner's four runs
(PLAN Phase 1b), none of this is built by this amendment.

### The evidence index re-pointed at `a7e924f`

The index above stays as it was, at `b109cea`. This table is beside it, one
row per row, with the same claim; its ids end in "now" so that
`tools/claims.py`, which reads one revision marker per record, does not read
them against `b109cea`. Each self row was checked at `a7e924f` by
`tools/claims.py` on a scratch copy of this table with plain ids and its own
marker (`self=a7e924f`): 34 of 34 pass. The thirteen rows of other
repositories (K11, K12, G1 to G9, M1, M2) are pinned to those repositories'
own revisions in the marker above and pass there (47 of 47 at `b109cea`'s
marker); they are repeated, unchanged, for completeness, and the tips of
those repositories were not read.

Tally of the 47 rows: **10 unchanged** (same line, same text), **20 moved**
(same text, still true), **3 moved with changed text** (C1, C10, C11: still
true in substance), **1 no longer true as stated** (C3: "and then writes" is
now guarded by `write`'s own check), **13 pinned** to other repositories.
The 25 rows of the PLAN's code claims, below in the PLAN: 14 unchanged, 9
moved, 2 moved with changed text (P3, P7), none no longer true.

| Id | Claim | Now | Location at `a7e924f` | Text |
| :--- | :--- | :--- | :--- | :--- |
| C1 now | The save | moved; the signature now takes a Prompter (0082: the wait for the lock is announced and retried) | `wizard/auth.go:1103` | `func (c resolvedCredential) save(ctx context.Context, p Prompter, d llmprovider.Descriptor) error {` |
| C2 now | refuses on an ended context only when `Token` fails | moved; still true of `save`'s first check, but no longer the only guard on the write: `write` checks the context too | `wizard/auth.go:1107` | `if _, err := c.session.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure)` |
| C3 now | and then writes | moved into `write`, after its own context check (0082 N-6); a session is no longer written on an ended context | `wizard/auth.go:1185` | `if err := c.saveTo.Save(ctx, d.ID, c.session); err != nil {` |
| C4 now | `Token` returns a current token without reading the context | moved from line 164; text and meaning the same | `llmprovider/auth/oauth_session.go:194` | `if current && !retrySave {` |
| C5 now | it reads the context before it starts a refresh | moved from line 182; text and meaning the same | `llmprovider/auth/oauth_session.go:212` | `if err := ctx.Err(); err != nil {` |
| C6 now | `FileTokenStore.Save` | moved from line 158; text and meaning the same | `llmprovider/auth/tokenstore_file.go:173` | `func (fs *FileTokenStore) Save(ctx context.Context, provider llmprovider.ProviderID, s *OAuthSession) (err error) {` |
| C7 now | writes by rename | moved from line 226; text and meaning the same | `llmprovider/auth/tokenstore_file.go:241` | `if err := tokenStoreRename(tmpName, final); err != nil {` |
| C8 now | `ConfigureLLM`'s first context check | moved from line 276; text and meaning the same | `wizard/configure.go:319` | `if err := ctx.Err(); err != nil {` |
| C9 now | `ResolveCredential`'s last context check, after the step | moved; still true: this is the check the change removes | `wizard/configure.go:435` | `if err := ctx.Err(); err != nil {` |
| C10 now | The step's save, after the listing loop | moved; text now passes the Prompter; the step still saves after the loop | `wizard/configure.go:578` | `if err := credential.save(ctx, p, d); err != nil {` |
| C11 now | The listing's own limit | moved; text now takes the limit from `o.discoverLimit()`; the listing's own limit, as before | `wizard/configure.go:919` | `dCtx, cancel := context.WithTimeout(ctx, o.discoverLimit())` |
| C12 now | The wizard waits for its sign-in under a detached context | unchanged | `wizard/driver.go:86` | `result, err := run.Wait(context.WithoutCancel(ctx))` |
| C13 now | Kilo's profile read for the menu; its failure a warning | moved from line 403; text and meaning the same | `wizard/auth.go:478` | `could not read the Kilo account's organizations (%s); using the personal account` |
| C14 now | A listing's failure that is not a refusal is not returned as one | moved from line 870; text and meaning the same | `wizard/configure.go:939` | `if stepRefusal(err) {` |
| C15 now | `ConfigureLLM` asks for a model id when the catalog is empty | moved from line 297; text and meaning the same | `wizard/configure.go:340` | `manual, inputErr := p.Input("No models found; enter a model id", existingModel(o, d.ID))` |
| C16 now | then the model question | moved from line 311; text and meaning the same | `wizard/configure.go:354` | `if res.Model, err = selectModel(ctx, p, d, cat, o); err != nil {` |
| C17 now | 0080 changes no other provider's check | unchanged | `docs/decisions/0080-MADR-verify-a-kilo-credential-through-its-profile.md:329` | `* No other provider's check changes.` |
| C18 now | The engine saves the plan's issued client as the login returns | moved; still true | `internal/signinflow/signinflow.go:501` | `clientSaveErr = spec.ClientStore.SaveClient(ctx, spec.Provider,` |
| C19 now | `FileTokenStore.SaveClient` ignores the context | moved from line 82; text and meaning the same | `llmprovider/auth/clientstore.go:107` | `func (fs *FileTokenStore) SaveClient(_ context.Context, provider llmprovider.ProviderID, c ClientRecord) error {` |
| C20 now | `x/signin`'s `Wait` saves nothing | moved from line 253; text and meaning the same | `llmprovider/x/signin/signin.go:269` | `// saves nothing: the caller saves. It returns ctx's error when ctx ends first.` |
| C21 now | A session can be revoked | unchanged | `llmprovider/auth/oauth_revoke.go:37` | `func RevokeOAuthSession(ctx context.Context, session *OAuthSession) error {` |
| C22 now | Kilo's device method is not marked revocable | moved from line 142; text and meaning the same | `llmprovider/x/signin/signin.go:158` | `ClientIdentity: factNone, Slot: "kilo"}},` |
| C23 now | `ConfigureLLM` calls the step | moved from line 289; text and meaning the same | `wizard/configure.go:332` | `res, cat, err := credentialStep(ctx, p, d, o)` |
| C24 now | The model-id branch returns the Result | moved from line 307; text and meaning the same | `wizard/configure.go:350` | `res.Model = manual` |
| C25 now | `selectModel` checks the context at its loop's top | unchanged | `wizard/model_select.go:52` | `if err := ctx.Err(); err != nil {` |
| C26 now | then asks the search | unchanged | `wizard/model_select.go:55` | `q, err := p.Input(searchModelsPrompt, "")` |
| C27 now | A blank search goes to the menu, which checks nothing | unchanged | `wizard/model_select.go:69` | `return selectRecommended(p, d, cat.Recommended, o)` |
| C28 now | The model id checks nothing | unchanged | `wizard/model_select.go:208` | `manual, err := p.Input("Model id", existingModel(o, provider))` |
| C29 now | `ConfigureLLM` asks the fallbacks | moved from line 315; text and meaning the same | `wizard/configure.go:358` | `if res.Fallbacks, err = selectFallbacks(ctx, p, d, cat, res.Model, existingFallbacks(o, d.ID)); err != nil {` |
| C30 now | which checks the context at its loop's top | unchanged | `wizard/model_select.go:249` | `if err := ctx.Err(); err != nil {` |
| C31 now | and a blank round's choice ends it at once | unchanged | `wizard/model_select.go:277` | `return chosen, nil // a blank round ends the loop` |
| C32 now | `ConfigureLLM` returns | moved from line 319; text and meaning the same | `wizard/configure.go:362` | `return res, nil` |
| C33 now | The search is over the catalog in hand | unchanged | `llmprovider/catalog/search.go:205` | `func (c Catalog) Search(query string) []Match {` |
| C34 now | A refresh saves only when a store is bound | moved; still true | `llmprovider/auth/oauth_session.go:1144` | `if state.store != nil {` |
| K11 now | The reference ChatGPT client revokes on logout | unchanged: pinned to codex's own revision, which no change here moves | `codex:codex-rs/login/src/auth/manager.rs:1094` | `pub async fn logout_with_revoke(` |
| K12 now | Kilo's own client opens its verification page (and prints the code) | unchanged: pinned to kilocode's own revision, which no change here moves | `kilocode:packages/kilo-gateway/src/auth/device-auth.ts:52` | `await open(verificationUrl).catch((err) => {` |
| G1 now | gobble's note: the wizard does not save after a done context | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:llm/provider/signinrun.go:146` | `save fails the sign-in with it, as the wizard does not save then. Wait does` |
| G2 now | gobble's note, on its key | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:internal/cli/agentauth.go:103` | `// nothing once its context is done (0051-MADR D4 step 3).` |
| G3 now | gobble's decision record says the same | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:368` | `is not saved; and it saves nothing once its context is done.` |
| G4 now | gobble copied the wizard's conjunction | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:llm/provider/signinrun.go:178` | `if _, err := sess.Token(ctx); err != nil && (errors.Is(err, llmprovider.ErrAuthFailure)` |
| G5 now | and checks the context itself before it saves | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:llm/provider/signinrun.go:182` | `if err := ctx.Err(); err != nil {` |
| G6 now | and before it stores a key | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:internal/cli/agentauth.go:104` | `if err := ctx.Err(); err != nil {` |
| G7 now | Its terminal path returns the context's error | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:llm/provider/signin.go:369` | `return Credential{}, ctx.Err()` |
| G8 now | Its sign-in is the wizard's `ResolveCredential` | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:llm/provider/signin.go:95` | `var resolveCredential = wizard.ResolveCredential` |
| G9 now | called by `SignIn` | unchanged: pinned to gobble-cli's own revision, which no change here moves | `gobble-cli:llm/provider/signin.go:345` | `res, _, err := resolveCredential(ctx, w, wizard.Options{` |
| M1 now | prepare-commit-msg's setup calls `ConfigureLLM` | unchanged: pinned to prepare-commit-msg's own revision, which no change here moves | `prepare-commit-msg:internal/ui/setup.go:274` | `res, err := wizard.ConfigureLLM(ctx, prompter, wizard.Options{` |
| M2 now | and checks the stored session afterwards | unchanged: pinned to prepare-commit-msg's own revision, which no change here moves | `prepare-commit-msg:internal/ui/setup.go:309` | `if err := config.ValidateOAuth(ctx, string(res.Provider), pc, store); err != nil {` |

## Amendment of 2026-11-19: what the owner's nine rulings change in D12

Added after the record was accepted and while Phase 1a's probe was being
reviewed; nothing above is rewritten or removed, and no ruling in "Decided"
changes. D12 describes the probe as the draft had it: two sign-ins, a baseline
use, a control use, a verdict, no revocation, and "each refresh token used
exactly once, so no reuse can be detected". Nine rulings of the owner,
relayed by the coordinator (the first four on 2026-11-18, the last five on
2026-11-19), changed it. Each label is quoted verbatim, and what each changes
in D12 is in this record's words. The PLAN's Phase 1a annotations and Phase 1b
amendment carry the build and the text the owner reads before a run.

* **Which verdicts pass.** "Only `kept` passes (Recommended)". The probe's
  test fails on `ended` and on `indeterminate`, so that `go test` itself says
  so. D12's "a verdict `ended` for any provider stops the PLAN" is unchanged.
* **The probe's own grants.** "Revoke the probe's own two grants at the end,
  after the verdict is printed (Recommended)". D12 said the sessions end with
  the process and the probe's sign-ins may end the owner's other sessions; the
  probe now also revokes the grants it obtained, with the SDK's own
  `auth.RevokeOAuthSession`, after the verdict line and on every path where a
  grant exists, on a bound of its own, and prints one line per grant (a status
  and a kind, nothing from a body or an error). A revocation never changes the
  verdict or fails the test and is not counted in any step. What cannot be
  revoked is listed in the PLAN's Phase 1b with where to remove it by hand:
  Kilo's device tokens (the SDK has no revocation for them) and the client a
  `plan` sign-in registers, and, by the two rulings on `chatgpt` and `grok`
  below, those providers' grants. **Not verified:** whether revoking a grant can end
  anything besides it, the owner's other sessions at that provider included.
  D12's "output is status codes, counts and one verdict per provider" now
  means the step lines (a status, an error's kind, an allow-listed OAuth code
  and a request count), the verdict line and the revocation lines.
* **The sequence.** "Add one step: refresh A once more after the control
  (Recommended)". D12's baseline and control uses are joined by `first-after`
  and `last-after`: sign-in 1, baseline, sign-in 2, first-after, control,
  last-after. `kept` now needs both of the first session's later refreshes to
  succeed; a first session whose refresh then fails with a terminal refusal
  while the control succeeded is `ended` (the lazy case); a grant that ends
  minutes after the run is still not caught. The verdict for every
  combination is the table in the PLAN's Phase 1b. The baseline's rotated
  token carries through all of the first session's refreshes.
* **Retries.** "The run is `indeterminate`: run it once more (Recommended)".
  D12's "each refresh token used exactly once, so no reuse can be detected" is
  not guaranteed: the SDK sends a refresh again, with the same token, on a
  transport error, a 429 or a 5xx, up to three times. The probe counts the
  round trips of each use and prints the count in the step line; any use step
  that made more than one request makes the whole run `indeterminate`,
  whatever its status was. For `grok`, whose issuer answers a reused token as
  plain `invalid_grant`, this count is the only guard against reading a
  reuse as a dead grant.
* **`chatgpt`'s grants.** "Run `chatgpt` with revocation switched off; its
  two grants are listed for removal by hand (Recommended)". The `chatgpt`
  sign-in uses the client id Codex itself uses, and nothing read says whether
  OpenAI's revocation ends only the token it names. The run therefore makes no
  revocation request for `chatgpt` on any path, and its lines read `kind
  by_hand`; two grants stay on the owner's OpenAI account until removed or
  expired, and a re-run leaves four. The measurement is unchanged.
* **`grok`'s grants.** "Switch `grok`'s revocation off too; its two grants are
  listed for removal by hand (Recommended)". The same treatment, for the same
  reason: the revocation would send the Grok CLI's own client id with the token,
  and it is not shown whether xAI ends only the one grant. The run makes no
  revocation request for `grok`, and no discovery request for one, on any path;
  its lines read `kind by_hand`; two grants stay on the owner's xAI account
  until removed or expired. Only `plan`'s grants are revoked by the run (they
  belong to the client the probe registered itself); Kilo has no revocation.
* **No resend after a refusal.** "As built: never resend a refused token; the
  last step runs only after a success (Recommended)". `last-after` is made only
  after a successful `first-after`; after a refusal the session still holds the
  same refresh token, and sending it again adds nothing.
* **A failed baseline.** "As built now: a failed baseline ends the run as
  `indeterminate` (Recommended)". The run ends before the second sign-in, so
  the owner is not asked for a sign-in that could not be measured.
* **A baseline of more than one request.** "Stop there: end the run as
  `indeterminate` before the second sign-in (Recommended)". A baseline that
  succeeded but needed more than one request makes the verdict `indeterminate`
  by the count rule already; the run now ends there too, on the path a failed
  baseline takes, so that the owner is not asked for a sign-in that cannot
  change the verdict and one fewer grant is created.

What D12 still says, and still holds: the sessions live in the test's memory
only; no store is opened; the verdicts are `kept`, `ended` and
`indeterminate`; two `indeterminate` runs of one provider in a row stop the
PLAN; the OpenAI and Grok device sign-ins are not probed.

## Amendment of 2026-11-20: D12's verdicts, the state table, and the facts at `24756a6`

Added after the record was accepted, after the owner's four probe runs and
before any of Phase 2's code; nothing above is rewritten or removed. No
decision, no status and no owner answer changes. The code facts are those of
`24756a6` (the tip of `main` on 2026-11-20), re-read for this amendment. The
state table below is the readiness pass the owner asked of every record that
changes behaviour: each case is `decided`, `out of scope` or `open`, and the
open ones go to the owner in one prompt before Phase 2 starts.

### D12: the four verdicts (PLAN Phase 1c)

The owner ran `TestLive_EarlierGrant_0081` once for each of the four sign-ins
on 2026-11-19, with the commands of the PLAN's Phase 1b amendment. Each run
wrote its file under `tmp/live-0081/` (git-ignored); the coordinator verified
the four files, and the PLAN's execution record ("Phase 1c") quotes every
line. Per provider:

* **`chatgpt`** (the ChatGPT browser sign-in, Codex's client): **`kept`**.
  `baseline`, `first-after`, `control` and `last-after` each status 200, kind
  `ok`, `requests 1`. Revocation switched off by the owner's ruling: two
  grants left on the owner's OpenAI account, to be removed by hand.
* **`plan`** (the ChatGPT plan sign-in, the second with the first session as
  `Previous`): **`kept`**. The four uses each status 200, `requests 1`. Both
  grants revoked by the run (status 200 twice); the client the first sign-in
  registered stays, as the PLAN's Phase 1b table says.
* **`grok`** (Grok's browser sign-in): **`kept`**. The four uses each status
  200, `requests 1`, so the count rule (the only guard for `grok` against a
  reused token read as `invalid_grant`) did not apply. Revocation switched off
  by the owner's ruling: two grants left on the owner's xAI account, to be
  removed by hand.
* **`kilo`** (Kilo's device sign-in): **`kept`**. The four profile reads each
  status 200, `requests 1`. No revocation exists (`unsupported`): two device
  tokens left on the owner's Kilo account, to be removed by hand.

What this decides, by the PLAN's own table ("What each outcome decides", Phase
1b): "`kept` for all four | the premise of 0059-MADR Q2 (a) and of saving last
holds for those sign-ins; Phase 2 proceeds". No stop condition of the PLAN is
met: no `ended`, no `indeterminate`, no `refresh_token_reused`, every provider
run, and no line of the files shows a token, a code, a client id or a URL.

What it does not decide, as the record already says: the OpenAI and Grok
device sign-ins were not probed (D12); a grant that ends minutes after a new
sign-in is not caught (the amendment of 2026-11-19); and whether a
revocation ends anything besides the grant it names is not verified.

Annotations, each of a sentence above:

* Consequences, "Bad, possibly, because if a provider ends the earlier grant
  on a new sign-in, an abandoned run leaves a stored session that no longer
  works": measured not to happen, within the run, for the four sign-ins
  probed.
* "The one case that could lose a working credential" (in "`ConfigureLLM`
  after the credential step"): the same; it remains a premise for the two
  device sign-ins.
* "Not verified", N1: annotated in place, 2026-11-20.

### What the code at `24756a6` changes in this record's facts

The code this record cites did not change after `a7e924f`, the commit the
amendment of 2026-11-18 re-grounded: `git diff --stat a7e924f 24756a6` lists
`AGENTS.md`, this pair, and the probe's three files in `llmprovider/`, and no
file of `wizard/`, `internal/signinflow/` or `llmprovider/auth/`. Record
0082's five code commits (`8051b00`, `04433ae`, `5f2c2ab`, `1545172`,
`fd986c4`) all precede `a7e924f`. Checked at `24756a6`, with each status taken
before any filter:

* `tools/claims.py` on this MADR and its PLAN, at their markers (`b109cea`):
  47 of 47 and 25 of 25 pass (exit 0).
* The re-pointed rows of the amendment of 2026-11-18 ("C1 now" to "M2 now";
  the PLAN's "P1 now" to "P25 now"), copied to a scratch file with plain ids
  and `self=24756a6`: 47 of 47 and 25 of 25 pass (exit 0). Every one holds at
  `24756a6` on the line the table names.
* The lines the amendments' prose cites (`save`, `write`, `lockToSave`,
  `writeUnderLock` and their checks in `wizard/auth.go`; `configure.go:351`,
  `:362`, `:435`, `:578`; 0082-MADR's N-6 at line 1145): each holds at
  `24756a6`, read by a scratch script against `git show`.

Facts that no longer hold as written, each found in this pass:

* **The plan's client record after an abandoned run** ("What the person and a
  caller can observe after the change", third bullet: "A later plan sign-in
  reuses the stored plan session's own client first, and the record only
  when there is no plan session (0075-MADR D7), so this changes nothing a
  person sees"). At `24756a6` a stored plan session is signed in to again
  only when its client is the record's, when the store keeps a record
  (`signsInAgainTo`, `wizard/auth.go:353`, 0082-MADR D2). The record is
  written at the sign-in (K9) and normally names the same client, so the
  sentence holds in the ordinary case. It does not hold when the record could
  not be decoded (0082-MADR's third amendment, R-9): that sign-in registers a
  new client and the record is replaced with it at the sign-in, so after an
  abandoned `ConfigureLLM` the store still holds the old plan session and the
  record names another client, and the next plan sign-in does not sign in
  again to the stored session (no `Previous`, the consent asked again). The
  same follows a refused sign-in today, as the sentence says. D7 stands; the
  state table's row S48 records the outcome.
* **`commitCredential`'s call of `save`** (D5's one save path; PLAN Phase 2
  step 4: `c.save(ctx, d)`). At `24756a6` `save` takes the run's `Prompter`
  (`wizard/auth.go:1103`, 0082's lock notice and retry question), so
  `commitCredential` needs it too. The PLAN's step is annotated.
* **"Breaks no existing test" (R5)** was measured at `b109cea` and on 0080's
  staged tree, not after 0082's wizard tests (seven files,
  `wizard/*_0082_test.go`), which call `save` and `ResolveCredential`
  directly. None calls `ConfigureLLM` or `credentialStep`, and `save` keeps
  its signature under D5, so none is expected to change; Phase 2's step 6 is
  the first run of the whole package on the change after 0082. Not run in this
  pass.
* **The plants against a session's windows.** For a session, `write`'s check
  (0082) and `commitCredential`'s check both refuse an ended context, so a
  plant that removes or weakens `commitCredential`'s check alone leaves a
  session test that asserts only "matches the context, nothing saved" green;
  the two differ only in the error's words (D9 as amended on 2026-11-18:
  `wizard: <ctx>` from the commit's check, "the new <label> session was not
  saved: <ctx>" from `write`'s). Reasoned from the code at `24756a6`, not run.
  The PLAN's dated entry of 2026-11-20 maps each plant to a test that can
  catch it.

### Two consequences of saving last that the record did not spell out

Both follow from decisions already taken; they are rows of the table, not new
rulings.

* **0082's wait and retry run at the very end.** `save`'s lock wait, its
  notice and, at the store's bound, its retry question are part of the save,
  which D5 puts after the model and fallback questions. A declined retry, or
  no answer, returns the store's error and saves nothing (0082's third
  amendment, R-10), after the person has answered every model question (N2).
* **A session that falls due during the questions is refreshed by the save.**
  `save` first asks the session for a token (C2, unchanged by D10). A session
  whose access token expired while the questions were on screen is refreshed
  then: its rotated pair is what is saved, and an issuer's refusal saves
  nothing and returns an error matching `llmprovider.ErrAuthFailure`. A
  refresh that fails for another reason, a 5xx, a 429 or a transport error,
  with the run's context alive, does not stop the save (the condition at
  `wizard/auth.go:1107` refuses only an auth failure or an ended context): the
  session is saved as it was, its access token expired and its refresh token
  unspent, and `ConfigureLLM` returns the Result (D10; row S22).

### Second pass of 2026-11-20: the independent review and the owner's rulings

The state table below was revised in place on 2026-11-20, before it was ever
committed, after an independent review of this amendment and the owner's
rulings on its open rows. Each finding was verified at its cited lines at
`24756a6` before it was applied; all held.

The owner's rulings, relayed by the coordinator, each recorded as a row:

* S50: "Last commit wins; document it (Recommended)".
* S61 and the review's finding of a notice that claims a save: "Fix both" (not
  the recommended option). Rows S61 and S67 to S77.
* The review's finding that Kilo's device token never expires: "Accept it;
  the docs say so plainly for Kilo (Recommended)". Row S60, and D6's
  annotation of 2026-11-20.
* S48: "Accept it as the decided outcome, with a test (Recommended)".
* S62: "One sentence in the architecture note (Recommended)".
* S76 (third pass, 2026-11-20): `"…; going on without verifying it" (Recommended)`.
* S77 (third pass, 2026-11-20): "Suppress every listing warning once the context has ended (Recommended)".
* S18: the question put to the owner was withdrawn by the coordinator, not
  asked: the review showed that a fake store whose `Save` cancels the run and
  then writes reaches the instant with no seam in production code (nothing
  reads the context between `wizard/auth.go:1181` and `:1185`).

The review's decidable findings, applied:

* **The commit's check before the lock** (D5's `commitCredential` runs before
  `save`, hence before the lock's wait, where `save`'s comment,
  `wizard/auth.go:1100-1102`, and 0082-MADR N-6 say the single check belongs
  after the lock). Decided by the amendment of 2026-11-18, "How the check
  composes with the lock and the retry loop": `write`'s check stays and covers
  the wait; `commitCredential`'s is a second, earlier one. 0082-MADR N-6
  carries a dated note of 2026-11-20 saying so, and the PLAN's step 5 gives the
  words that replace the comment's last sentence.
* **S22's third branch** (a refresh at the commit failing for a reason that is
  not an auth failure): decided by D10, above; a case of P-5.
* **Tests that could not pin their rows** (S10, S11, S12, S13, S29, S30, P-3,
  P-6): after Phase 2 a session's cancel meets up to three checks in turn,
  N3's (in `ConfigureLLM`), `commitCredential`'s and `write`'s, and a loop's
  top check at the model stage. They differ only in the error's words (D9 as
  amended on 2026-11-18): `wizard: <ctx>` from N3's and the commit's checks;
  `select model: <ctx>` or `select fallbacks: <ctx>` from a loop's top; "the
  new <label> session was not saved: <ctx>" from `save` and `write`. So every
  cancel test asserts the words of the first check that runs after its cancel,
  and a plant on one check is named against the entry point where that check
  is the first: `commitCredential`'s through `ResolveCredential`, N3's through
  `ConfigureLLM`. The rows and the PLAN's map say which.

### The state table

Each row is one input or state the change can meet. Status: `decided` (the
outcome, where it is decided, and the test that pins it), `out of scope`
(with the reason) or `open` (not decided by this record or an owner answer:
put to the owner). A test named "P-n" is proposed in the PLAN's dated entry of
2026-11-20, in the phase it belongs to; "existing" is a test in the tree at
`24756a6`; "guard" is a drafted test green at `a7e924f` by 0082's check.

The owner answers the rows cite, by their labels (verbatim, from "Decided"):
Q1 "Every window before the write, by one check just before the save
(Recommended)"; Q2 "Every kind, typed keys included (Recommended)"; Q3 "A
cancel or the run's deadline saves nothing; the listing's own limit still
saves (Recommended)"; Q4 "The first option, and `ConfigureLLM` saves at its
very end"; Q5 "Out of scope; say so in the docs (Recommended)"; Q6 "Leave
both; add one sentence to `Wait`'s doc (Recommended)"; N1 "A live probe by me
before release (Recommended)"; N2 "Any error from `ConfigureLLM` saves nothing
(Recommended)"; N3 "Right after the credential step, as well as at the end
(Recommended)". The owner's rulings of 2026-11-20, relayed by the
coordinator, are quoted in the rows they decide (the second pass, above).
"Words" in a Test cell means the test asserts the error's words of the first
check that runs after its cancel (the second pass, above), compared as the whole
error text, since `write`'s error also begins with `wizard:`. Rows that rest on a sibling record treat its decision as
given; 0082 and 0080 are landed, 0085 and 0083 build after this record.

**The windows of one run** (every session kind, both entry points, unless the
row says)

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S1 | Context ended before `ConfigureLLM` starts | decided | no question; an error `wizard: <ctx>` matching the context's; nothing saved | D9 row 1 (C8; 0059-MADR W8) | existing `TestConfigure_EndedContextAsksNothing_0059`. Plant: `ConfigureLLM`'s first context check (`wizard/configure.go:319`) removed |
| S2 | Context ended before `ResolveCredential` starts | decided | the same | D9 row 1 | existing `TestResolveCredential_ErrorKinds_0056/ctx`. Plant: `ResolveCredential`'s first context check (`wizard/configure.go:416`) removed |
| S3 | Cancel while a question before the login is on screen (provider, endpoint, method menu, keep-existing confirm), answered anyway, then a browser or device sign-in | decided | the login starts on the ended context and returns its error, matching the context's; nothing issued, nothing saved, no Result | D9 row 2 (K1); Q1 | P-1 (Phase 2). Plant: `signIn`'s browser branch returns `errors.New("wizard: sign-in failed")` in place of the login's error |
| S4 | Cancel at the plan's consent question | decided | the engine returns the context's error; no login, no client registered, nothing saved | D9 row 2 | P-1, a case. Plant: `resolveChatGPTPlan` returns a wizard error in place of the login's |
| S5 | Cancel while the browser waits for its callback or the paste, or the device poll waits (K1) | decided | the login's error, matching the context's; nothing saved | D9 row 2 | P-1. Plant: as S3 (the browser case); `signIn`'s device branch likewise (the device case) |
| S6 | Cancel after the code arrives: during the token exchange or the ID token's check | decided | the login's error (`oauth: token request:` wraps the context's); nothing saved; a grant the issuer may have made is held by no one (S58) | D9 row 2; D6 | P-1, a case through the shipped `auth.LoginBrowserOAuth` and an in-process issuer. Plant: as S3 |
| S7 | Cancel after the login returned, before the check (K2), with and without `Discover` | decided | nothing saved; `wizard: <ctx>` from `commitCredential` (`ResolveCredential`) or from N3's check (`ConfigureLLM`) | D2 (Q1); D9 as amended on 2026-11-18 | guard `TestCancelAfterIssueSavesNothing_0081`, words; plants: `commitCredential`'s check removed (its `ResolveCredential` cases), N3's check removed (its `ConfigureLLM` cases) The comparison is of the whole error text, since `write`'s error also begins with `wizard:`. |
| S8 | Kilo device session, its profile read for the organization menu, without `Discover` (K3) | decided | nothing saved; `wizard: <ctx>` | D2 | guard `TestCancelDuringKiloProfileSavesNothing_0081`, words; plants as S7. The comparison is of the whole error text, since `write`'s error also begins with `wizard:`. |
| S9 | The same under `Discover` | decided | the context's error (0080's `checkEnded`); nothing saved | 0080-MADR D6 (given); D2 | existing `TestKiloVerify_EndedContext_0080`. Plant: `verifyKilo`'s context check (`wizard/kilo_verify.go:125`) removed (the warning is then shown) |
| S10 | Kilo device session, a cancel while the organization menu is on screen, answered anyway | decided | nothing saved; `wizard: <ctx>` | D2 | P-8, cases added to `TestCancelDuringKiloProfileSavesNothing_0081`, words; plants: `commitCredential`'s check removed (the `ResolveCredential` case: `write`'s words), N3's check removed (the `ConfigureLLM` case: `select model:`) |
| S11 | Cancel during the listing (K4), each session kind (Kilo device; Grok device and browser; OpenAI browser and device; a pasted ChatGPT access token; the ChatGPT plan) | decided | nothing saved; `wizard: <ctx>` | D2; Q1 | guard `TestCancelDuringListingSavesNothing_0081`, words, the plan's cases included; `TestCancelDuringDeviceFlowSavesNothing_0081`. Neither may be dropped: a case that cannot be served in process without changing non-test code is a stop and a deviation. Plants: `commitCredential`'s check removed (the `ResolveCredential` cases), N3's check removed (the `ConfigureLLM` cases: `select model:`, or for the three OpenAI cases a model id asked) The comparison is of the whole error text, since `write`'s error also begins with `wizard:`. |
| S12 | The run's context ends while the save waits for a refresh the listing left in flight, after the commit's check passed (K5) | decided | not saved; an error matching the context's, in `save`'s words; returned at once, without waiting for the refresh to finish | D10; 0059-MADR's amendment ("nor [...] when the run's context ends first") | guard `TestCancelDuringRefreshSavesNothing_0081`, extended: `DiscoverLimit` of 50 ms, the issuer holding the refresh for five seconds, the run cancelled 200 ms after the refresh arrives; asserts the error within one second of the cancel. Plant: `save`'s `Token` called on `context.WithoutCancel(ctx)` |
| S13 | `ConfigureLLM`, a context ended by the end of the step | decided | no model question; `wizard: <ctx>`; nothing saved | D5 (N3) | `TestCancelledConfigureReturnsNoKey_0081` (an empty catalog: no model id asked); P-9 (a session, words: without N3 the model loop's top answers `select model:`). Plant: N3's check removed |
| S14 | `ConfigureLLM`, a cancel at the model search, menu, model id or fallbacks (K10) | decided | the words of the first check after the cancel: `select model: <ctx>` or `select fallbacks: <ctx>` where a loop's top runs next (a query answered goes to the matches' loop, `wizard/model_select.go:104`; a model chosen with fallbacks on goes to their loop); `wizard: <ctx>` from the final check where the question answered was the last; nothing saved; no Result | D5 (Q4); D9 row 4 | `TestCancelAtModelStageSavesNothing_0081`; P-3, words per case. Plant: `commitCredential`'s check removed (the cases ending at the final check) |
| S15 | `ConfigureLLM`, a cancel at the model-id question of an empty catalog | decided | nothing saved; no Result | D5 (Q4) | `TestCancelAtManualModelSavesNothing_0081`. Words. Plant: `commitCredential`'s check removed (`write`'s words) |
| S16 | `ConfigureLLM`, the run's deadline at the model stage | decided | an error matching `context.DeadlineExceeded`; nothing saved | D4 (Q3) | P-3, a case. Plant: `commitCredential` matching `context.Canceled` only |
| S17 | `ConfigureLLM`, an error that is not the context's at the model stage: a `Prompter`'s error, no model entered, a choice out of range, a fallback question's error | decided | that error; nothing saved | D5 (N2) | `TestModelStageErrorSavesNothing_0081` (the search); P-4 (no model entered; the fallbacks). Plants: `ConfigureLLM` saving in the step again (the search case); P-4's own, `commitCredential` called before the blank check, and before `selectFallbacks` |
| S18 | Between the final check and the write (K6's last instant) | decided | saved; the Result | D9 row 6 | P-18 `TestCancelBetweenTheCheckAndTheWriteSaves_0081`: a store whose `Save` cancels the run, then writes. Plant: a context check added after `Save` in `write` |
| S19 | During the write (K7): the store's `Save` ends the run while it writes | decided | saved; the Result, through either entry point | D9 row 7; D5 | `TestCancelAfterSaveReturnsTheResult_0081` (`ResolveCredential`); P-11 (`ConfigureLLM`). Plants: `ResolveCredential`'s own last check put back; a context check added after the commit in `ConfigureLLM` (P-11) |
| S20 | After the write | decided | `ResolveCredential`: the Result (its own last check removed); `ConfigureLLM`: the Result, nothing runs after | D5 (Q4); D9 row 8 | as S19. Plants: as S19 |
| S21 | A run that ends well | decided | `ConfigureLLM` saves once, after the model and fallbacks, the store empty while they are asked; `ResolveCredential` saves once, in the step, and asks no model | D5 (Q4) | `TestConfigureSavesOnceAfterTheModel_0081`; guard `TestResolveSavesInTheStep_0081`. Plants: `ConfigureLLM` saving in the step again; `ResolveCredential` never committing |
| S22 | `ConfigureLLM`, a session whose access token expired during the questions | decided | the save's `Token` refreshes it. Answered: the rotated pair saved once, the Result. Refused by the issuer: nothing saved, an error matching `llmprovider.ErrAuthFailure`. Failed otherwise (a 5xx, a 429, a transport error), the context alive: the session saved as it was, the Result | D10; D5 (N2) | P-5, three cases. Plants: `save`'s first check (`wizard/auth.go:1107`) removed (the refusal case); that check refusing any error (the third case) |
| S23 | A refresh the listing left running completes during the questions | decided | adopted in memory; saved, rotated, once, at the end | "A refresh during the questions" (above); D10 | `TestRotationDuringQuestionsSavedAtEnd_0081`. Plant: `ConfigureLLM` saving in the step again |

**Credential kinds**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S24 | A typed key: the listing cancelled; a descriptor with no static catalog; a cancel at the model menu | decided | an error matching the context's; no key; no model id asked (`ConfigureLLM`) | D3 (Q2); D5 (N3) | guard `TestCancelDuringListingReturnsNoKey_0081`; `TestCancelledConfigureReturnsNoKey_0081`; `TestCancelAtModelStageReturnsNoKey_0081`; existing `TestResolveCredential_ErrorKinds_0056/ctx_ended_during_the_listing`. Plants: the check for a session only (the two `NoKey` tests' `ResolveCredential` cases); N3's check removed (`TestCancelledConfigureReturnsNoKey_0081`, a model id asked); `commitCredential`'s check removed (the `0056` subtest) |
| S25 | An environment key, a saved key (`Options.Existing`), Kilo's device key (`Options.KiloDeviceKey`), each on a context that ends before the commit, with and without `Discover` | decided | an error matching the context's; no Result; no key | D3 (Q2) | P-6. Words. Plants: `commitCredential` checks only a session or a typed key (the `ResolveCredential` cases); N3's check removed (the `ConfigureLLM` cases: `select model:`) |
| S26 | A vendor CLI read-through, and no credential (Ollama, a keyless local server) | decided | an error matching the context's; no Result | D3 ("whatever the credential"); D5 (the one check precedes `save` for every credential, PLAN Phase 2 step 4) | P-6, cases. Plants: as S25 |
| S27 | A Claude key, a cancel at the workspace question, answered anyway | decided | an error matching the context's; no key | D3 (Q2) | P-6, a case. Plants: as S25 |
| S28 | A pasted ChatGPT access token (an access-only session) | decided | nothing saved | D2, D3 | guard `TestCancelDuringListingSavesNothing_0081`, its case. Plants: as S11 |
| S29 | A kept session, the context ending during the run | decided | an error matching the context's; no Result; the run saves nothing; a rotation of the kept session during the run is saved through its own store | D3 (Q2) | P-7: its `ResolveCredential` case holds the plant `commitCredential` returns before its check for a kept credential (the Result comes back); its `ConfigureLLM` case, words, holds N3's check removed |
| S30 | A kept session refused, a new sign-in, the run abandoned at the last model question, answered anyway | decided | the store keeps the refused kept session; `wizard: <ctx>`; nothing saved | D5 (Q4); 0066-MADR V7 (given) | P-7, a case, words. Plant: `commitCredential`'s check removed (`write`'s words) |

**Modes and stores**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S31 | `Discover` off | decided | as S7 and S25 | D2, D3 | guard `TestCancelAfterIssueSavesNothing_0081` (without `Discover`); P-6, its case. Plants: as S7 and S25 |
| S32 | `Options.Driver` set | decided | as S7: nothing saved, an error matching the context's, though the Driver was given the sign-in's done step | D2; D9 row 3 | P-9 |
| S33 | `NoPastePrompt`, `NoEndpointPrompt`, `LocalServers`, `ShowDeviceCode`, `Identity`, `HostID`, `ConsentText` | out of scope | each acts before the login returns, or on what it shows; none reaches the save path, and their windows are S3 to S6 | — | — |
| S34 | The store a `RefreshLocker` (`FileTokenStore`, gobble's), not one, or nil | decided | under the lock; written directly; with no store only the methods that save no session, S24 to S26 | 0082 (given); D5 | existing `TestSaveTakesTheRefreshLock_0082`, `TestSaveWithoutALockerSaves_0082`. Plant: `save` writing without taking the lock |
| S35 | `ConfigureLLM`'s commit meets a lock another process holds | decided | the waiting notice, the wait and, at the bound, the retry question come after the model and fallback questions; a declined retry, or none, returns the store's error and saves nothing | D5 (Q4, N2); 0082 second and third amendments, R-7, R-10 (given) | P-10. Plant: `ConfigureLLM` saving in the step again |
| S36 | The context ends during the commit's lock wait, or while its retry question is on screen | decided | nothing saved; an error matching the context's | 0082 R-7, N-6 (given) | existing `TestSaveCancelledWhileWaitingSavesNothing_0082`, `TestSaveCancelledAtAnyWaitSavesNothing_0082`. Plant: `lockToSave` waiting on `context.WithoutCancel(ctx)` |

**Deadlines, interrupts and the browser**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S37 | The run's deadline during the listing | decided | an error matching `context.DeadlineExceeded`; nothing saved; no key | D4 (Q3) | guard `TestDeadlineDuringListingSavesNothing_0081` (a session), words; P-14 (a key). Plant: `commitCredential` matching `context.Canceled` only. The comparison is of the whole error text, since `write`'s error also begins with `wizard:`. |
| S38 | The listing's own limit (`Options.DiscoverLimit`) passes, the run's context alive | decided | an outage: saved (in the step for `ResolveCredential`, at the end for `ConfigureLLM`); the Result | D4 (Q3) | P-13. Plant: `discoverModels` returns a listing ended by its own limit as a `*checkEnded` |
| S39 | The login's own bound (the browser's ten minutes, a device code's expiry); the person closes the browser | decided | the login's error; nothing issued; nothing saved | D9 row 2 (K1) | P-1, cases. Plant: as S3 |
| S40 | An interrupt the caller maps to a cancel | decided | as the context's rows | "Every window" ("Cancelled" is the end of the run's context) | the context's tests. Plants: those of S3 to S37 |
| S41 | An interrupt the caller does not map: the process is killed (prepare-commit-msg sets no signal context) | decided | before the write nothing is saved, for `ConfigureLLM` even after the model questions; during it the old file or the new (K7); after it, saved | "What the person and a caller can observe after the change", first bullet; D2 (K7) | P-16. Plant: `ConfigureLLM` saving in the step again |
| S42 | Kilo's profile read past its own bound under `Discover` | decided | a warning; saved | 0080-MADR D6 (given) | existing `TestDiscoverLimit_0080`. Plant: `discoverLimit` ignoring `Options.DiscoverLimit` |

**The earlier grant**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S43 | No earlier session in the store | decided | after any cancel before the write the store stays empty | D2, D3 | the save-counting tests above (zero saves). Plants: those of the rows whose tests count the saves |
| S44 | An earlier session in the store | decided | after any cancel before the write, or any error from `ConfigureLLM`, the store keeps it, unchanged | D2; D5 (Q4, N2) | the same. Plants: as S43 |
| S45 | Whether the earlier grant still works after a new sign-in: the ChatGPT browser sign-in, the plan, Grok's browser sign-in, Kilo's device sign-in | decided | `kept` for all four, measured 2026-11-19 | D12 (N1); PLAN Phase 1c | `TestLive_EarlierGrant_0081` (owner-run, Phase 1b) and its twins. Plants: Phase 1a's 88 (for instance, a kept verdict reported as ended) |
| S46 | The same for the OpenAI and Grok device sign-ins | out of scope | not probed: N1's ruling names four sign-ins | D12 | — |
| S47 | A grant that ends minutes after a new sign-in | out of scope | the probe cannot catch it | the amendment of 2026-11-19 | — |
| S48 | The plan's client record after an abandoned `ConfigureLLM` | decided | written at the sign-in (K9) and kept; after a sign-in whose record could not be decoded, it names the new client, and the next plan sign-in does not sign in again to the stored plan session: no `Previous`, the consent asked again | the owner, relayed by the coordinator: "Accept it as the decided outcome, with a test (Recommended)"; D7 (Q6); 0082 D2 and third amendment R-9 (given) | P-15. Plant: `signsInAgainTo` ignores the record |

**Concurrency**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S49 | A refresh of the stored session, in this process or another, when the commit runs | decided | the save waits for it under the lock and lands after it | 0082 second amendment, R-1, R-7 (given) | existing `TestSaveWaitsForARefreshInFlightInOneProcess_0082`. Plant: `save` writing without taking the lock |
| S50 | Another run saves a sign-in to the same store while this `ConfigureLLM`'s model questions are on screen | decided | this run's commit, at its end, writes over it: the store holds the session of the last commit; `ConfigureLLM`'s doc and `Options.TokenStore`'s say so in one sentence | the owner, relayed by the coordinator: "Last commit wins; document it (Recommended)" | P-19 `TestLastCommitWins_0081`. Plant: the commit skips the save when the store already holds a session |
| S51 | A session held elsewhere (gobble) after this run's commit | decided | it adopts the same sign-in and refuses another | 0082 D1 to D3 (given) | 0082's tests in `llmprovider/auth`. Plant: 0082's guard in `reloadOrRefresh` removed (another sign-in adopted) |

**Failed and partial writes, and what is left behind**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S52 | The store's `Save` fails before it writes (a temp file, its restriction, the write, sync, close or rename; permission denied) | decided | the store's error; no Result; the store as it was; for `ConfigureLLM` after the model questions | D9 row 7 ("the store's error as today"); D10; D5 (N2) | P-12. Plant: `write` ignores the store's error |
| S53 | The store's `Save` fails after the rename (the directory's sync) | decided | the store's error, with the new session in the store | D9 row 7 | P-12, a case. Plant: as S52 |
| S54 | The process killed during the write | out of scope | `FileTokenStore`'s: the old file or the new, and a temp file `.tok-*.json` may stay in the store's directory | D10 | — |
| S55 | The plan's client record cannot be written | out of scope | a warning, the sign-in goes on (0075-MADR) | D7 (Q6) | — |
| S56 | The lock file cannot be opened at the commit | decided | the neutral retry question | 0082 fourth amendment, R-13 (given) | existing `TestLockFileThatCanNeverBeOpenedAsksNeutrally_0082`. Plant: a failed try that is not contention treated as contention |

**At the provider**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S57 | A session issued and not saved: any cancel after the issue, an abandoned `ConfigureLLM`, a refused sign-in | out of scope | live at the provider until it expires or is removed; nothing revoked; the docs say so | D6 (Q5) | — |
| S58 | A grant the issuer made during an exchange the cancel cut off | out of scope | held by no one; as S57 | D6 (Q5) | — |
| S59 | The client an unsaved first plan sign-in registered | out of scope | stays under the account | D6 (Q5); D7 (Q6) | — |
| S60 | Kilo's device token, unsaved: any cancel after the issue, an abandoned `ConfigureLLM`, a refused sign-in | decided | the token stays on the Kilo account: it does not expire and the SDK cannot revoke it (C22); the person removes it in the Kilo account's settings (the page is not named: not verified). `ConfigureLLM`'s doc and the architecture note say so for Kilo (PLAN Phase 2 step 5) | the owner, relayed by the coordinator: "Accept it; the docs say so plainly for Kilo (Recommended)"; D6 (Q5) and its annotation of 2026-11-20 | P-20 `TestSaveNotesInTheArchitecture_0081`, the Kilo case. Plant: the Kilo sentence removed from `docs/architecture.md` |

**What the person is shown, and the docs**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S61 | The listing's warning "could not list models for <label> (...); using the built-in catalog", the run cancelled during the listing, `Discover`, both entry points: at `wizard/configure.go:951` (a listing that returned an error, with a static catalog) and at `:967` (a catalog that is not live, nothing recommended, carrying the failure, with a static catalog) | decided | not shown; the run returns `wizard: <ctx>`; nothing saved | the owner, relayed by the coordinator: "Fix both" | P-17, one case per site: `:951` driven by a third party's descriptor with a static catalog that `catalog` does not list (`providerCatalog`), its `ListModels` returning the context's error; `:967` driven by a `catalog.Lister` stand-in with a static catalog returning nothing recommended, not live, carrying the context's error. Plants, one per site: that site's warning shown whatever the run's context. (A cancel during a cold-cache `catalog.List` of a provider with a static catalog reaches `:974`, S77, not these two: second-pass reading corrected in the fourth pass.) |
| S62 | Whether the docs state the measured premise | decided | the architecture note's "When a session is saved" gains one sentence: the earlier grant was kept, measured on 2026-11-19, for the ChatGPT browser sign-in, the plan, Grok's browser sign-in and Kilo's device sign-in; the OpenAI and Grok device sign-ins were not probed | the owner, relayed by the coordinator: "One sentence in the architecture note (Recommended)" | P-20, the premise case. Plant: the sentence removed from `docs/architecture.md` |
| S67 | Kilo's notice "could not verify the Kilo key (...); it is saved unchecked" (`wizard/kilo_verify.go:128`): `Discover`, no organization, the profile cannot say, the run's context alive, `ResolveCredential` (the save follows in the step) | decided | the notice is shown and no longer says the key is saved; its words are S76's | the owner, relayed by the coordinator: "Fix both"; 0080-MADR D6 (given, its words superseded for this notice) | P-21 `TestKiloUnverifiedNoticeClaimsNoSave_0081`, a typed key and a device session. Plant: the words back to "it is saved unchecked" |
| S68 | The same notice in `ConfigureLLM`, shown before the model questions, after which the run may be abandoned and nothing saved | decided | as S67 | as S67 | P-21, its `ConfigureLLM` cases; plant as S67 |
| S69 | The same notice, the run's context ended (cancel or deadline) at the check, either entry point | decided | not shown; the context's error | 0080-MADR D6 (given: `checkEnded`) | existing `TestKiloVerify_EndedContext_0080`. Plant: `verifyKilo`'s context check (`wizard/kilo_verify.go:125`) removed |
| S70 | The same notice without `Discover` | decided | never shown: no profile is read | 0080-MADR D3 (given) | existing `TestKiloVerify_NoDiscoverNoRead_0080`. Plant: `verifyKilo` runs without `Discover` |
| S71 | The listing's warnings with the run's context alive: an outage, or the listing's own limit (`DiscoverLimit`) | decided | shown, as today; the session is saved (in the step, or at the end) | "Fix both" covers only a run whose context has ended; D4 (Q3); 0066-MADR's notices (given) | existing `TestListing_StaticBesideNoticed_0066`; P-13 asserts the warning. Plant: the warning suppressed on the listing's own context (`dCtx`) instead of the run's |
| S72 | The run's deadline passing during the listing, at each of the six listing-warning sites (S61's two, S77's four), the listing cache cold | decided | not shown; `wizard: <ctx>` matching `context.DeadlineExceeded` (with a listing cached within a day the catalog is served stale and live, and nothing is warned, as today) | "Fix both"; the owner, relayed by the coordinator: "Suppress every listing warning once the context has ended (Recommended)"; D4 (Q3) | P-17, a deadline case per site, driven as S61's and S77's. Plant: the suppression matching `context.Canceled` only |
| S73 | Kilo's organizations warning, "could not read the Kilo account's organizations (...); using the personal account" (`wizard/auth.go:478`), without `Discover`, the run's context ended (cancel or deadline) during the read | decided | not shown; nothing saved; `wizard: <ctx>` | "Fix both" | `TestCancelDuringKiloProfileSavesNothing_0081`, the assertion added with a deadline case (P-8). Plant: `resolveKiloDevice` keeps its suppression under `Discover` only, as today |
| S74 | The organizations warning without `Discover`, the run's context alive, the read failing | decided | shown, as today | 0080-MADR D3 (given) | existing `TestKiloVerify_NoDiscoverNoRead_0080/device_session`. Plant: the warning suppressed whatever the context |
| S75 | The organizations warning under `Discover`: the context ended; or alive with a failure that is not a 401 | decided | ended: not shown (as today); alive: shown, then S67's notice | 0080-MADR D3, D6 (given); "Fix both" for S67's words | existing `TestKiloVerify_EndedContext_0080` and `TestKiloVerify_OutagesWarn_0080` (its expected words change with S76, which the ruling allows). Plants: the organizations warning suppressed with the context alive; S67's words back to the old ones |
| S76 | The new words of S67's notice | decided | the notice reads "could not verify the Kilo key (<class>[, HTTP <status>]); going on without verifying it", in place of "...; it is saved unchecked", on every path that shows it (S67, S68, S75). The same words replace today's in 0080-MADR D6 (by a dated note, line 433), `wizard/kilo_verify_0080_test.go:362` and `:383`, `CHANGELOG.md` lines 40-41 and `docs/architecture.md:1148-1149`; 0085 rewords the comment at `wizard/kilo_verify.go:95` in its own phase and builds after this record | the owner, relayed by the coordinator: `"…; going on without verifying it" (Recommended)`; "Fix both" | P-21, which asserts these words; `TestKiloVerify_OutagesWarn_0080` with its words changed. Plant: the words back to "it is saved unchecked" |
| S77 | The listing's other warnings, the run's context ended (cancel or deadline) during the listing, `Discover`, both entry points: "could not list models for <label> (...)" with no static catalog (`wizard/configure.go:948`, `:958`), and "live model listing for <label> is unavailable ...; search covers the built-in catalog only" (`:974`, `:977`) | decided | not shown, like S61's two; the run returns `wizard: <ctx>`; nothing saved. With the run's context alive they are shown, as today (S71) | the owner, relayed by the coordinator: "Suppress every listing warning once the context has ended (Recommended)" | P-17, one case per site: `:948`, a ChatGPT session (OpenAI, `CredOAuth`; no static catalog), its listing's transport returning when its request's context ends; `:951`, a third party's descriptor with a static catalog that `catalog` does not list (`ErrUnsupported`, so `providerCatalog`), its `ListModels` returning the context's error (`registry0056` with static ids); `:958`, a third party's `catalog.Lister` stand-in, no static catalog, whose `ListCatalog` returns a catalog that is not live and carries the context's error, with a nil error; `:967`, the same stand-in with a static catalog, returning nothing recommended, not live, carrying the context's error; `:974`, a built-in provider with a static catalog listed by `catalog.List`, the listing cache cold (a failed listing comes back as the static catalog carrying the error, with a nil error: `llmprovider/catalog/discovery.go:353-366`, `:556-565`); `:977`, a `catalog.Lister` stand-in returning a catalog that is not live, with models recommended and no error. Plants, one per site: that site's warning shown whatever the run's context |
| S78 | In `resolveBaseURL`, Ollama or a local server, both entry points: the endpoint question answered after a cancel, so `catalog.ValidateOllamaURLWith` runs on the ended context and the person is shown "cannot reach <url> (...)" and asked "Try a different endpoint?" (`wizard/configure.go:704-718`) | out of scope | not a listing warning: the owner's ruling "Suppress every listing warning once the context has ended (Recommended)" names listing warnings only, and this record decides nothing about the endpoint check. Listed in the PLAN as found and not changed, a follow-up for the owner | — | — |

**Sibling records and platforms**

| Row | Case | Status | Outcome | Decided by | Test |
| :--- | :--- | :--- | :--- | :--- | :--- |
| S63 | 0082's check in `write` | decided | not moved, removed or repeated; a session passes two checks, `commitCredential`'s first | the amendment of 2026-11-18; 0082 N-6 (given) | Phase 3's two plants on `write`'s check; existing `TestSaveCancelledAtAnyWaitSavesNothing_0082`. Plants: `write`'s check moved before the lock's wait; skipped on the retry pass |
| S64 | 0085 (builds after this record): a cancel while its blank-key question is asked again | out of scope | 0085's loop and its error; any key that comes out of it meets `commitCredential`'s check (S24) | 0085-MADR D2, D3 | — |
| S65 | 0083 (builds after this record): a cancel at its question where the browser is, or while a sign-in with no listener waits for the paste | decided | as S3 and S5: the login's error; nothing issued, nothing saved | D9 row 2 (K1) | none in this PLAN: the question lands with 0083, whose PLAN tests it. Owed by 0083-PLAN, since the question does not exist at this record's base; it fails on P-1's plant (S3) |
| S66 | Each platform (macOS, Linux, Windows) | out of scope | the change adds no platform-dependent code; the write's platform behaviour is `FileTokenStore`'s | D10 | — (the PLAN's entry proposes one Windows run of the tests) |

Counts, after the fourth pass of 2026-11-20 (corrections: S61's sites,
S78 added): 78 rows; **67 decided**, **11 out of scope**, **0 open**,
**0 awaiting live evidence** (D12's measurement is recorded above). Every
decided row names a test and the plant that must turn it red; S65's test
is owed by 0083's PLAN. This amendment does not change the record's
status, which is the owner's.
