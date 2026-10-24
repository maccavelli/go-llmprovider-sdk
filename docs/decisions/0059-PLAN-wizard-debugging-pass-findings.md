---
status: in-progress
date: 2026-10-24
associated-madr: "0059-MADR-wizard-debugging-pass-findings.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Remediation of the Wizard's Debugging-Pass Findings

Associated MADR: [0059-MADR-wizard-debugging-pass-findings.md](0059-MADR-wizard-debugging-pass-findings.md)

## Goal

The MADR's D1, under the owner's answers of 2026-10-24, (a) to each of Q1
to Q7: each of the 11 findings that changes code (W1 to W11; W12 is
withdrawn, and W13 stays as decided) becomes a test that fails on
`2d6e7f3`, then passes. In particular:

* no key that the listing accepts is called the wrong shape, for any
  provider (W1), and no notice promises a check that does not run (W3);
* a sign-in that the listing refuses leaves the saved session as it was
  (W2);
* the masked entry types only what was pasted (W4), and the paste prompt
  does not spin (W5);
* an unusable saved session is not offered, and the sign-in menu follows
  (W6); a saved fallback is offered (W7);
* `ConfigureLLM` asks nothing on an ended context (W8), `NoEndpointPrompt`
  asks nothing (W9), raw-mode entry is tested (W10), and a pasted token's
  session keeps the caller's client (W11);
* `v1.22.1` is released.

## Scope

### In scope

* **`wizard/`:** `auth.go`, `configure.go`, `base_url.go`,
  `model_select.go`, `text_prompter.go`, and new tests named
  `*_0059_test.go`.
* **Two existing tests whose assertion the owner's answers reverse,**
  rewritten to the new behaviour, each named in its phase:
  * `TestProviderShapeWarning` (`wizard/auth_shapes_0028_test.go`), which
    asserts the warning Q1 (a) removes;
  * `TestConfigureLLM_KeepRefusesStubSession` (`wizard/auth_test.go`),
    which asserts the end of the run Q3 (a) replaces.
* **Docs:** each changed function's doc; `docs/architecture.md`, "The
  wizard", whose sign-in "saves the session" (Phase 2); `CHANGELOG.md`;
  the README's Status.
* **Records:** this pair, and its index rows.

### Out of scope

* Any exported name. The PLAN adds none and changes no signature, so R55's
  table is empty. `Options`' and `ConfigureLLM`'s docs change where a
  behaviour does (R52).
* W12, withdrawn by the MADR's amendment, and W13, kept by Q6 (a).
* Q0, the owner's key's shape, which no step needs.
* `llmprovider/`, `catalog` and the providers: no change.
* gobble-cli, which takes `v1.22.1` in its own repository.

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts after the owner has committed the last.
* **The gate, before staging:**
  * `python3 -B tools/gate.py`;
  * `make pre-add-check FILES="…"` on the phase's Go files;
  * `tools/claims.py` on the MADR;
  * `make records-check`;
  * `tools/recordlint.py --changed`;
  * `tools/denyscan.py --changed`;
  * `tools/linkcheck.py`.

  Each exit status is captured before any filter.
* **Red first.** Every new test is seen to fail on `2d6e7f3`'s code, or on
  a planted break, by `tools/plant.py` on a scratch clone, never in the
  tree. The phase's record holds each failure line. A probe of the MADR
  (P1 to P8) becomes its finding's first test.
* **The drift check.** After each phase's code and before its new tests,
  `go test -race -count=1 -shuffle=on ./wizard/` passes with every existing
  test unchanged, the two named in Scope aside.
* **No key in a test, a log or a record.** Keys are built to a shape, as
  `auth_shapes_0028_test.go` builds them.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * an existing test other than the two in Scope would change;
  * an exported name would be added or change;
  * a doc would state a behaviour no test shows.

  On any of these, stop, record a dated deviation here (with an amendment
  in the MADR when a decision or a fact changes), and wait.

## Implementation Steps

### Phase 0: records

1. `tools/records.py status`: the MADR `accepted`, this PLAN
   `in-progress`.

### Phase 1: the shape check, by S1 (W1, W3; Q1 (a))

1. **Another vendor's key, for every provider.** `foreignKeyPrefixes`'
   entries, and the OpenCode shape, name the providers that own them:
   `sk-ant-` Claude; `xai-` Grok; `tgp_` Together; `hf_` and `api_org_`
   Hugging Face; `AIza` and `AQ.` Gemini; the OpenCode shape OpenCode Zen
   and Go; `sk-or-` none. A new `foreignKeyFor(id, value)` returns the
   vendor of a value that is another provider's, or "".
   `foreignKeyVendor` keeps its answers, so the OpenAI paste
   (`resolveTokenStdin`) is unchanged.
2. **`promptAPIKey(p, d, o)`** replaces `warnKeyShape` with that check. A
   key that is another vendor's gets "this looks like <vendor> key, not a
   <Label> key", followed, only with `o.Discover`, by "; the model listing
   will check it" (W3). No other key gets a notice. `resolveAPIKey` and the
   refusal loop pass `o`.
3. **The shape, in a refusal.** In `credentialStep`, each time the listing
   refuses a typed key, the last refusal included, the wizard notices "a
   <Label> key is <shape>" before it asks again or returns. It does this
   only when `keyShapes` has a shape for the provider and the key does not
   match it. Claude's shape text becomes "sk-ant-api03- or sk-ant-admin01-
   and 95 characters ending in AA", which is what the check accepts.
4. **The drift check runs here.** `TestProviderShapeWarning` is the one
   test expected to fail. It is rewritten: a malformed key of no vendor,
   and a well-shaped one, get no notice; `TestForeignKey_Researched` is
   unchanged.
5. **The tests,** `wizard/shape_0059_test.go`:
   * `TestShape_AcceptedKeyNoWarning_0059`: P1. Claude keys `sk-ant-api04-`
     (108), `sk-ant-api03-` (109) and the standard one, each listed and
     accepted by a stubbed transport. No notice, and the key is returned.
   * `TestShape_ForeignKey_0059`: an `sk-ant-` key typed for Grok, an
     `xai-` one for Claude, and an OpenCode one for OpenAI. Each gets one
     notice naming the vendor, with the listing's promise under
     `Discover`. Claude's own key for Claude gets none.
   * `TestShape_NoDiscoverNoPromise_0059`: P2. Without `Discover`, the
     foreign-key notice has no "listing will check", and no listing call
     is made.
   * `TestShape_RefusalNamesShape_0059`: a typed Claude key without the
     shape, refused by the listing three times. Each refusal's notices
     name the shape, and no notice shows a key. A well-shaped key refused
     once gets no shape notice.
6. **Plants:**
   * the exact-shape warning put back before the listing: the first test
     fails;
   * the promise shown without `Discover`: the third fails;
   * the refusal's shape notice dropped: the fourth fails;
   * `foreignKeyFor` treating Claude's own prefix as foreign: the second
     fails.
7. **Docs:** `promptAPIKey`'s and `credentialStep`'s docs;
   `CHANGELOG.md` `[Unreleased]`, Fixed: "the wizard no longer warns that
   a key the provider accepts has the wrong shape".

### Phase 2: a fresh session saved only once accepted (W2; Q2 (a))

1. **`saveOAuthCredential` becomes `pendingOAuthCredential`:** it checks
   the session (`auth.ValidateOAuthSession`) and sets its `Provider`, but
   neither sets its `Store` nor saves it. The credential carries the store
   it is due to. A refresh during the listing then rotates the tokens in
   memory only (a session with no `Store` saves nothing,
   `llmprovider/auth/oauth_session.go`'s refresh).
2. **Its callers:** the browser and device sign-ins, the ChatGPT plan, the
   pasted ChatGPT token, and `resolveKiloDevice`, whose session is saved
   now before its profile is read.
3. **`credentialStep` saves it** once the listing has not refused it: it
   sets `Store` and calls `Save`, after the refusal loop. That covers an
   accepted listing, a listing that failed for another reason, and no
   listing (`Discover` off), as today. On a refusal the session is not
   saved, and the store keeps what it held. A failed `Save` is returned as
   today's is, "wizard: save … session".
4. **A kept session** (`keepExistingOAuth`) is not saved again: it is the
   store's own.
5. **The drift check runs here.** `store.saves` counts in existing tests
   are expected to hold, since every accepted sign-in saves once.
6. **The tests,** `wizard/session_0059_test.go`:
   * `TestSession_RefusedKeepsSaved_0059`: P8. The store still holds the
     previous session's access token after the refusal, and was not
     written.
   * `TestSession_AcceptedSavedOnce_0059`: a browser sign-in whose listing
     succeeds is saved once, with `Store` set.
   * `TestSession_NoDiscoverSaved_0059`: without `Discover`, saved once.
   * `TestSession_RotatedDuringCheckSaved_0059`: a session that refreshes
     during the listing (its access expired, its refresh token valid, a
     transport that answers the token endpoint) is saved with the new
     access token.
   * `TestSession_KiloRefusedNotSaved_0059`: Kilo's device sign-in, its
     listing refused: nothing saved.
7. **Plants:** the save moved back before the listing (the first test
   fails); the save skipped on success (the second fails); `Store` set
   before the listing (the store is written during a refusal, and the
   first fails).
8. **Docs:** `docs/architecture.md`, "The wizard": a sign-in's session is
   saved once its listing accepts it. `CHANGELOG.md` Fixed.

### Phase 3: the masked entry and the paste prompt (W4, W5; Q4 (a))

1. **W4, `readMasked`:** the escape state is kept across reads, in place
   of `skipEscape`'s look at the buffer. After an ESC, a `[` begins a CSI
   sequence, ended by a byte in 0x40–0x7E, however many reads it takes;
   an `O` takes one more byte. Any other character after an ESC that had
   bytes buffered behind it is dropped with it (Alt and a key, as today).
   One after a lone ESC is kept (the Escape key, then a key, as today:
   `TestReadMasked_Keys` holds).
2. **W5, `pastePrompt.input`:** an empty answer returned within
   `pasteQuick` (50 ms) of the read's start counts. After three in a row,
   the wizard notices "the prompter answers at once; waiting for the
   browser", stops asking, and waits for `ctx`, so the loopback or the
   login's limit ends it. An empty answer after a longer wait resets the
   count, so a person's Enter is still asked again (0057-MADR D4).
   `pasteQuick` is a variable, which a test changes.
3. **The drift check runs here.**
4. **The tests,** `wizard/entry_0059_test.go`:
   * `TestReadMasked_SplitEscape_0059`: P3. The two splits give
     `sk-ant-keyAA`, and so does an arrow key split after its `[`.
   * `TestPaste_QuickEmptiesStop_0059`: P4. At most three `Input` calls in
     100 ms, one notice, and the input returns when `ctx` ends.
   * `TestPaste_SlowEmptyAskedAgain_0059`: a `Prompter` whose empty answer
     takes longer than `pasteQuick` is asked a fourth time.
5. **Plants:** `skipEscape`'s buffer check put back (the first test
   fails); the count not reset by a slow answer (the third fails); no
   limit (the second fails).
6. **Docs:** `readMasked`'s and `pastePrompt.input`'s docs;
   `CHANGELOG.md` Fixed.

### Phase 4: the saved session and the saved fallbacks (W6, W7; Q3 (a), Q5 (a))

1. **W6, `keepExistingOAuth`:** it sets `Store` and the client, then
   checks the session with `validateKept`, before it asks. One that fails
   is not offered: a warning notice "the saved <Label> session cannot be
   used (<reason>); sign in again", where the reason is the check's own
   text, never a service's. It returns no session and no error, so the
   method menu follows.
2. **W7, `selectFallbacks`:** the blank round offers, after the
   recommended rows, each saved fallback that is not excluded and not among
   them. Each is marked "saved" and preselected. A round ends early only
   when nothing remains, saved fallbacks included.
3. **The drift check runs here.** `TestConfigureLLM_KeepRefusesStubSession`
   is the one test expected to fail. It is rewritten: no `Confirm` for the
   stub, one notice, and the method menu shown next.
4. **The tests,** `wizard/kept_0059_test.go`:
   * `TestKept_UnusableNotOffered_0059`: P7. No `Confirm`, the notice, then
     the method menu; a browser sign-in chosen there completes the run.
   * `TestKept_UsableStillOffered_0059`: a valid saved session is offered
     and kept, as today.
   * `TestFallbacks_SavedOffered_0059`: P5. The saved
     `claude-legacy-fallback` is offered, marked and preselected, and kept
     by a blank answer.
   * `TestFallbacks_SavedOnlyRound_0059`: no recommended row left, a saved
     fallback remaining: the round is still shown.
5. **Plants:** the check moved back after the `Confirm` (the first test
   fails); the saved rows dropped (the third fails); the early return when
   no recommended row is left put back (the fourth fails).
6. **Docs:** `keepExistingOAuth`'s and `selectFallbacks`' docs;
   `CHANGELOG.md` Fixed.

### Phase 5: the context, the endpoint, raw mode and the client (W8 to W11; Q7 (a))

1. **W8, `ConfigureLLM`:** it returns `fmt.Errorf("wizard: %w",
   ctx.Err())` before any question when `ctx` has ended, as
   `ResolveCredential` does.
2. **W9, `resolveBaseURL`:** under `NoEndpointPrompt`, a plain-http remote
   URL is used without a `Confirm`, with a warning notice "<URL> is plain
   http to a remote host, so the key travels unencrypted". A typed URL is
   asked about as today. `validateBaseURL` takes whether it may ask.
   `Options.NoEndpointPrompt`'s doc says so.
3. **W10, a raw-mode seam:** `isTerminal`, `makeRaw` and `restoreTerm`
   become package variables over `golang.org/x/term`'s functions, which
   tests replace. `secretOnce` reads through them.
4. **W11, the pasted token's session:** `accessOnlyOpenAISession` takes
   the caller's client and calls `UseHTTPClient`.
5. **The drift check runs here.**
6. **The tests,** `wizard/edges_0059_test.go`:
   * `TestConfigure_EndedContextAsksNothing_0059`: P6. No question asked,
     and the error matches `context.Canceled`.
   * `TestBaseURL_NoPromptPlainHTTP_0059`: under `NoEndpointPrompt`, an
     `Existing` plain-http remote URL: no `Confirm`, one warning, and the
     URL in the Result. A typed one is still asked about.
   * `TestSecret_RawMode_0059`: through the seam, over an `os.Pipe`: raw
     mode on, the masked read, and the restore; raw mode refused, the
     visible read and its notice; the restore failing, an error notice.
     `secretOnce` reaches at least 90 % coverage.
   * `TestPastedToken_KeepsClient_0059`: the pasted ChatGPT token's session
     holds `Options.HTTPClient`.
7. **Plants:** the context check removed (the first test fails); the
   `Confirm` put back (the second fails); `restoreTerm`'s failure ignored
   (the third fails); `UseHTTPClient` dropped (the fourth fails).
8. **Docs:** `ConfigureLLM`'s and `Options.NoEndpointPrompt`'s docs;
   `CHANGELOG.md` Fixed.

### Phase 6: release `v1.22.1`

1. **`CHANGELOG.md`:** `[Unreleased]` becomes `[v1.22.1] - <date>`, dated
   by the commit the tag will name. It holds the Fixed entries of Phases 1
   to 5 and no Added, since no name is added. An empty `[Unreleased]` goes
   above it, and the comparison links are updated. The README's Status
   names `v1.22.1`.
2. **Before the push:** the gate; `tools/apidiff.py v1.22.0`, listing no
   change; the notes from `scripts/check_changelog.py --section v1.22.1`
   on a local clone with a local tag, scanned; the disclosure guard.
3. **The tag,** on the owner's word: `v1.22.1`, annotated, pushed. Every
   `validate` job passes, and the `release` job's body equals step 2's
   notes.
4. **After the tag:** `gh release delete v1.19.0 --yes`, the Release that
   falls to sixth (R53). `tools/consumer.py v1.22.1` and
   `tools/offline.py v1.22.1`.
5. **Close:** the MADR's Confirmation, item by item; this PLAN `complete`.

## Verification

| # | Criterion | Phase | Check |
| :--- | :--- | :--- | :--- |
| V1 | A key the listing accepts gets no shape warning, for any provider | 1 | `TestShape_AcceptedKeyNoWarning_0059`, `TestProviderShapeWarning` rewritten |
| V2 | Another vendor's key is named; no check is promised without `Discover` | 1 | `TestShape_ForeignKey_0059`, `TestShape_NoDiscoverNoPromise_0059` |
| V3 | A refused typed key's notice names the shape | 1 | `TestShape_RefusalNamesShape_0059` |
| V4 | A refused sign-in leaves the saved session | 2 | `TestSession_RefusedKeepsSaved_0059`, `TestSession_KiloRefusedNotSaved_0059` |
| V5 | An accepted one is saved once, with any rotation | 2 | `TestSession_AcceptedSavedOnce_0059`, `…_NoDiscoverSaved_0059`, `…_RotatedDuringCheckSaved_0059` |
| V6 | A split escape sequence is not typed | 3 | `TestReadMasked_SplitEscape_0059`, `TestReadMasked_Keys` |
| V7 | The paste prompt does not spin, and a person's Enter is asked again | 3 | `TestPaste_QuickEmptiesStop_0059`, `TestPaste_SlowEmptyAskedAgain_0059` |
| V8 | An unusable saved session is not offered, and the menu follows | 4 | `TestKept_UnusableNotOffered_0059`, `TestConfigureLLM_KeepRefusesStubSession` rewritten |
| V9 | A saved fallback is offered and kept | 4 | `TestFallbacks_SavedOffered_0059`, `TestFallbacks_SavedOnlyRound_0059` |
| V10 | Nothing is asked on an ended context, nor about plain http under `NoEndpointPrompt` | 5 | `TestConfigure_EndedContextAsksNothing_0059`, `TestBaseURL_NoPromptPlainHTTP_0059` |
| V11 | Raw-mode entry is tested | 5 | `TestSecret_RawMode_0059`; `secretOnce` ≥ 90 % |
| V12 | The pasted token's session keeps the caller's client | 5 | `TestPastedToken_KeepsClient_0059` |
| V13 | No exported name added or changed | 1–6 | `tools/apidiff.py v1.22.0` |
| V14 | Every new test failed first | 1–5 | the execution record |

## Rollout and Rollback

* **Rollout.** Seven commits on `main`, staged by the agent and committed
  by the owner: the records, one per phase, and the release commit. Then
  the owner's tag.
* **Rollback.**
  * Each phase reverts on its own, newest first. No name is added, so a
    revert is allowed after the tag too, as a later patch.
  * Reverting Phase 1 brings back the false warning; Phase 2, the
    refused session saved over the good one.

## Execution record

### Phase 0: records (2026-10-24)

* **Approval.** The owner answered Q1 to Q7 ("follow recommendations and
  write the plan"), committed the pair as `a3a2d9b`, and approved this PLAN
  with "committed, proceed".
* **Step 1.** `tools/records.py status`: the MADR `accepted` and this PLAN
  `in-progress`, both dated 2026-10-24, with their index rows. The MADR's
  Decision Outcome now names R1 with S1 as chosen.
* **Not done here:** no code. Phase 1 starts after the owner has committed
  this phase.
