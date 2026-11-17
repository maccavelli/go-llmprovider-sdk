---
status: proposed
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
