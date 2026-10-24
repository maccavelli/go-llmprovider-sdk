---
status: complete
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
* **Added 2026-10-24, Phase 2's deviation: two existing tests that call
  the sign-in's steps directly,** rewritten in Phase 2:
  * `TestSaveOAuthCredential_RejectsFixture` (`wizard/auth_test.go`) calls
    `pendingOAuthCredential`; its assertion, an error and no save, holds;
  * `TestResolveTokenStdin_ChecksTheCredential` (`wizard/auth_test.go`),
    whose current-token row asserts a credential due to the store and not
    yet saved. The save itself is
    `TestConfigureLLM_TokenStdinStillAcceptsChatGPTToken`'s, unchanged.
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
  test unchanged, those named in Scope aside (four since Phase 2's
  deviation).
* **No key in a test, a log or a record.** Keys are built to a shape, as
  `auth_shapes_0028_test.go` builds them.
* **Stage, then the owner commits** with `git commit --no-edit`. Push and
  tags happen on the owner's word, after the disclosure guard.
* **Stop conditions:**
  * a fact of the MADR no longer holds;
  * an existing test other than those in Scope would change;
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

   **Amended 2026-10-24 (the deviation below):** before it saves, it waits
   for a refresh the listing left running past its limit, by the session's
   `Token` with the run's context, so that the store gets the rotated
   tokens. A refresh the issuer refuses, or the run's context ending, saves
   nothing and ends the run with the error; any other failure saves the
   session as it is.
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
   * `TestSession_RefreshAfterLimitSaved_0059` (added 2026-10-24): a
     session whose refresh answers after the listing's limit is saved once,
     with the rotated refresh token.
   * `TestSession_RefreshRefusedNotSaved_0059` (added 2026-10-24): a
     session whose refresh the issuer refuses after the listing's limit is
     not saved, and the run ends with an error matching `ErrAuthFailure`.
7. **Plants:** the save moved back before the listing (the first test
   fails); the save skipped on success (the second fails); `Store` set
   before the listing (the store is written during a refusal, and the
   first fails); the wait dropped (added 2026-10-24:
   `TestSession_RefreshAfterLimitSaved_0059` fails).
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

### Phase 7: Claude's refusal hint by its prefix (W14; the MADR's amendment "Q0 answered"; proposed)

1. **Records.** This PLAN `in-progress`; the MADR's amendment carries the
   owner's answer to Q8.
2. **`keyShapes`' Claude entry** becomes `^sk-ant-[A-Za-z0-9]+-`, with the
   text "sk-ant- and a tag, such as sk-ant-api03- or sk-ant-usr-". The
   doc comment says Claude's tag and length are not fixed, so only its
   prefix and tag are checked, as Grok's prefix is.
3. **The test,** `TestShape_RefusalNamesShape_0059`, a test of this PLAN:
   its count looks for the new text; it gains a refused `sk-ant-usr-` key
   and a refused `sk-ant-api03-` key of another length, each drawing no
   shape notice; the refused key with no `sk-ant-` prefix still draws one
   per refusal. Seen to fail first on `ca51351`'s code, where the two new
   cases draw three each.
4. **The drift check,** `go test -race -count=1 -shuffle=on ./wizard/`,
   and `-count=2`: every other test unchanged.
5. **Plants:** the old exact entry put back (the `usr-` case fails); the
   prefix dropped from the check, every refused key then unnamed (the
   no-prefix case fails).
6. **Docs:** `CHANGELOG.md` `[Unreleased]`, Fixed. `docs/architecture.md`'s
   "A refused key" already says "the provider's usual shape" and holds.

### Phase 8: release `v1.22.2` (under Q8 (a))

1. Phase 6's steps 1 to 5 with `v1.22.2`: the section dated by its commit,
   the README's Status, `tools/apidiff.py v1.22.1` listing no change, the
   notes built and scanned, the disclosure guard; the tag and its push on
   the owner's word; then `gh release delete v1.20.0 --yes`, the Release
   that falls to sixth; `tools/consumer.py` and `tools/offline.py`.
2. **Not here:** prepare-commit-msg's bump, which is that repository's own
   change, under its records.

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
* **Committed** by the owner ("committed, proceed").

### Phase 1: the shape check, by S1 (2026-10-24)

* **Step 1.** `foreignKeyPrefixes` carries each prefix's owners, and the
  OpenCode shape's are `openCodeOwners`. `keyVendor` returns the vendor
  and its owners. `foreignKeyVendor` keeps its answers (it returns
  `keyVendor`'s vendor), and `foreignKeyFor(provider, value)` is new.
* **Step 2.** `promptAPIKey(p, d, o)` calls a new `warnForeignKey`, in
  place of `warnKeyShape`, which is removed.
* **Step 3.** A new `notifyShape` names the shape. `credentialStep` calls it
  after each refusal notice of a typed key, and before it returns on the
  last. Claude's shape text names `sk-ant-admin01-` too.
* **Deviation (2026-10-24): the notices' wording.** Step 2's "not a <Label>
  key" and step 3's "a <Label> key is <shape>" put "a" before labels that
  begin with a vowel ("a OpenAI"). The notices read "this looks like
  <vendor> key, not one for <Label>" and "<Label> API keys look like
  <shape>; this one does not". What each says, and when, is as written.
  No decision changes.
* **Step 4, the drift check:** `go test -count=1 ./wizard/` failed only
  `TestProviderShapeWarning`, nine lines of "malformed key: notices [];
  want one warning", as Scope foresaw. Rewritten: each key, of the shape or
  not, gets no notice, Kilo and Ollama as before. Its check that no notice
  shows the key went with it, since it now asserts there is no notice. Then
  `go test -race -shuffle=on ./wizard/`: ok.
* **Step 5, the tests,** `wizard/shape_0059_test.go`: the four as written.
  Each lists through its own `testCache(t)`, since Claude's listing is
  keyed by its default URL and the key's fingerprint, which tests would
  otherwise share (0058-MADR). Against `2d6e7f3`'s code, in a scratch
  clone, all four failed:

  ```text
  --- FAIL: TestShape_AcceptedKeyNoWarning_0059/another_version_tag
      shape_0059_test.go:59: notices ["this does not look like a Claude (Anthropic) API key (expected sk-ant-api03- and 95 characters ending in AA); the model listing will check it"]; want none for a key the listing accepted
  --- FAIL: TestShape_ForeignKey_0059
      shape_0059_test.go:89: claude: its own key: notices ["this does not look like a claude API key (expected …)…"]; want none
  --- FAIL: TestShape_NoDiscoverNoPromise_0059
      shape_0059_test.go:113: 0 listings, notices ["this does not look like a Claude (Anthropic) API key (…); the model listing will check it"]; …
  --- FAIL: TestShape_RefusalNamesShape_0059/without_the_shape
      shape_0059_test.go:140: 0 notices name the shape; want 3: …
  ```

* **Step 6, the plants** (`tools/plant.py`, a scratch clone), each caught:
  * the exact-shape warning put back: `TestShape_AcceptedKeyNoWarning_0059`
    and the others, "notices ["Claude (Anthropic) API keys look like …"];
    want none for a key the listing accepted";
  * the promise without `Discover`: `TestShape_NoDiscoverNoPromise_0059`;
  * the refusal's shape notice dropped:
    `TestShape_RefusalNamesShape_0059`, "0 notices name the shape; want 3";
  * a provider's own prefix taken as foreign. Its first form, `if vendor
    == ""`, left `owners` unused, so the build failed, which proves
    nothing. Planted again as `len(owners) == 0 && slices.Contains(…)`,
    which compiles: `TestShape_AcceptedKeyNoWarning_0059` and
    `TestShape_ForeignKey_0059` failed, "claude: its own key: notices
    ["this looks like an Anthropic key, not one for claude; …"]; want
    none".
* **Step 7, docs:** `promptAPIKey`'s, `credentialStep`'s and `keyShapes`'
  docs; `CHANGELOG.md` `[Unreleased]`, Fixed.
* **The checks:** `make pre-add-check` on the 4 Go files, "4 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `python3 -B
  tools/gate.py`, "gate: 27 of 27 passed".
* **Not done here:** Phases 2 to 6.
* **Committed** by the owner as `ba45339` ("committed, proceed").

### Phase 2: a fresh session saved only once accepted (2026-10-24)

* **Deviation (2026-10-24): a refresh can outlast the listing, and two
  existing tests change.** Found before any change to the tree, by Phase 2
  as written applied to scratch clones of `ba45339`:
  * the drift check failed on two existing tests outside Scope:
    `wizard/auth_test.go:358`, "undefined: saveOAuthCredential", and
    `TestResolveTokenStdin_ChecksTheCredential/a_current_JWT`,
    "auth_test.go:601: resolveTokenStdin = oauth, <nil> with 0 saves; want
    the session saved";
  * a probe (a Grok browser sign-in, its access expired, a listing limit of
    100 ms, a token endpoint answering at 400 ms) lost the rotation. The
    saves, by refresh token: today's code `["oauth-refresh" "rt-new"]`;
    as written `["oauth-refresh"]`, the spent token only.

  The owner chose "option 1": the save waits for the refresh (step 3's
  amendment), the probe and a refused refresh become tests (step 6), the
  wait gets a plant (step 7), and the two tests join Scope. With the wait,
  the probe's store held `["rt-new"]`, and `go test -race -shuffle=on
  ./wizard/` failed only the paste test's row. The MADR has the
  amendment "a refresh can outlast the listing (Phase 2)".
* **Records committed** by the owner as `2c52707`, and Phase 2 approved
  ("committed, proceed").
* **Steps 1 and 2.** `saveOAuthCredential` is `pendingOAuthCredential(store,
  provider, session)`: it checks the session and names its provider, and
  the credential carries the store in a new field, `saveTo`. The browser,
  device and ChatGPT plan sign-ins call it, and the pasted token through
  `pendingAccessOnlyOpenAI`, renamed from `saveAccessOnlyOpenAI` since it
  no longer saves; `resolveTokenStdin`'s context is unused, `_`.
  `resolveKiloDevice` sets `saveTo` and no longer saves.
* **Step 3.** `resolvedCredential.save(ctx, d)` runs in `credentialStep`
  after the refusal loop. It returns at once with no `saveTo`, else waits
  through the session's `Token(ctx)`, then sets `Store` and saves. A
  refused refresh or an ended context returns "wizard: the new <Label>
  session was not saved: …", and a failed save today's "wizard: save OAuth
  session" or "wizard: save Kilo login". Step 4 needed no code: a kept
  session has no `saveTo`.
* **Step 5, the drift check:** the two Scope tests rewritten, as the
  deviation above found nothing else, `go test -race -count=1 -shuffle=on
  ./wizard/`: ok.
* **Step 6, the tests,** `wizard/session_0059_test.go`, seven. A store,
  `store0059`, records what each save held, not the session's pointer,
  which the wizard goes on changing. Against `ba45339`'s code, in a scratch
  clone, five failed:

  ```text
  --- FAIL: TestSession_RefusedKeepsSaved_0059/an_empty_store (and /a_previous_session)
      session_0059_test.go:141: saves [{access:oauth-access refresh:oauth-refresh …} {access:at-new refresh:rt-new …}], the store holds OAuthSession{…}; want none, and what it held before
  --- FAIL: TestSession_RotatedDuringCheckSaved_0059
      session_0059_test.go:184: saves [{access:oauth-access refresh:oauth-refresh …} {access:at-new refresh:rt-new …}]; want one, of the rotated tokens
  --- FAIL: TestSession_KiloRefusedNotSaved_0059
      session_0059_test.go:201: saves [{access:kilo-device-token-0123456789 refresh: …}]; want none
  --- FAIL: TestSession_RefreshAfterLimitSaved_0059
      session_0059_test.go:223: saves [{access:oauth-access refresh:oauth-refresh …}]; want one, of the rotated refresh token
  --- FAIL: TestSession_RefreshRefusedNotSaved_0059
      session_0059_test.go:238: err <nil>; want one matching ErrAuthFailure, saying the session was not saved
  ```

  The first shows more than P8 did: the refused listing invalidates the
  token and refreshes it, and the old code saved that refresh too. The
  other two, `TestSession_AcceptedSavedOnce_0059` and
  `TestSession_NoDiscoverSaved_0059`, pass on the old code, which saved
  once as well; the plants are theirs.
* **Step 7, the plants** (`tools/plant.py`, a scratch clone), all four
  caught:
  * the save moved back before the listing:
    `TestSession_RefusedKeepsSaved_0059`, both cases, and three more;
  * the save skipped on success: `TestSession_AcceptedSavedOnce_0059`,
    "saves []; want one, of the new session, with its Store set", and
    `TestSession_NoDiscoverSaved_0059`, with three more;
  * `Store` set before the listing: `TestSession_RefusedKeepsSaved_0059/an_empty_store`,
    with `TestSession_RotatedDuringCheckSaved_0059` and
    `TestSession_RefreshAfterLimitSaved_0059`, "saves [{… rt-new …} {…
    rt-new …}]; want one";
  * the wait dropped: `TestSession_RefreshAfterLimitSaved_0059`, "saves
    [{access:oauth-access refresh:oauth-refresh …}]; want one, of the
    rotated refresh token", and `TestSession_RefreshRefusedNotSaved_0059`.
* **Deviation (2026-10-24): the refusal test's second case.** The first run
  of the plants caught `Store` set before the listing, but not by
  `TestSession_RefusedKeepsSaved_0059`, as step 7 says. With a previous
  session in the store, the refusal's refresh finds the store holding
  another refresh token and takes that session (0020-MADR F1's reload),
  saving nothing. The test gained a case with an empty store, where the
  refresh is saved, and the plant fails it, as step 7 says. No decision
  changes.
* **Step 8, docs:** `docs/architecture.md`, "The wizard";
  `credentialStep`'s, `resolveKiloDevice`'s and the new functions' docs;
  `CHANGELOG.md` `[Unreleased]`, Fixed.
* **The checks:** `make pre-add-check` on the 4 Go files, "4 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `python3 -B
  tools/gate.py`, "gate: 27 of 27 passed".
* **Not done here:** Phases 3 to 6.
* **Committed** by the owner as `4eff4c7` ("committed, proceed").

### Phase 3: the masked entry and the paste prompt (2026-10-24)

* **Step 1, W4.** `skipEscape` is removed. `readMasked` keeps an
  `escapeState` across reads: an ESC sets `escLone`, or `escAlt` when bytes
  are buffered behind it, and `escapeState.next` takes each rune after it.
  A `[` begins a CSI, ended by a byte in 0x40–0x7E; an `O` takes one more;
  any other rune is dropped after `escAlt` and kept after `escLone`. So the
  Escape key followed, in a later read, by a typed `[` or `O` now begins a
  sequence, as step 1 says a split sequence must; `TestReadMasked_Keys`,
  the Escape key and then `a`, holds.
* **Step 2, W5.** `pastePrompt.input` counts empty answers that came within
  `pasteQuick` (a variable, 50 ms) of the read's start; a slower one resets
  the count. At `pasteQuickEmpties` (3) it notices, at `LevelWarn`, "the
  prompter answers at once; waiting for the browser", unless drain has run,
  and returns `ctx`'s error once `ctx` ends.
* **Step 3, the drift check:** `go test -race -count=1 -shuffle=on
  ./wizard/`: ok, every existing test unchanged.
* **Step 4, the tests,** `wizard/entry_0059_test.go`, three; the split
  cases are the two of P3, an arrow split after its `[`, and an SS3 key
  split after its ESC. Against `4eff4c7`'s code, in a scratch clone, with
  `pasteQuick` and `pasteQuickEmpties` declared in a shim test file so that
  the tests build, two failed:

  ```text
  --- FAIL: TestReadMasked_SplitEscape_0059
      entry_0059_test.go:25: the end marker after its ESC: readMasked = "sk-ant-keyAA[201~", <nil>; want sk-ant-keyAA
      entry_0059_test.go:25: the start marker after its ESC: readMasked = "[200~sk-ant-keyAA", <nil>; want sk-ant-keyAA
      entry_0059_test.go:25: an SS3 key after its ESC: readMasked = "sk-ant-keyOAAA", <nil>; want sk-ant-keyAA
  --- FAIL: TestPaste_QuickEmptiesStop_0059
      entry_0059_test.go:65: input: context canceled after 100.076542ms; want ctx's deadline, once it ended
      entry_0059_test.go:68: 619407 Input calls, notices []; want 3 and one notice
  ```

  The arrow split after its `[` passed: the old code read on past the
  buffer once it had the `[`. `TestPaste_SlowEmptyAskedAgain_0059` passed,
  since the old code asked again after every empty answer; its plant is
  the second below. It runs under a 5 s context, so a plant that stops the
  asking fails it rather than hanging.
* **Step 5, the plants** (`tools/plant.py`, a scratch clone), all three
  caught:
  * `skipEscape`'s buffer check put back (a lone ESC begins no sequence):
    `TestReadMasked_SplitEscape_0059`, the three lines above;
  * the count not reset by a slow answer (the slow branch removed):
    `TestPaste_SlowEmptyAskedAgain_0059`, "input = "", context deadline
    exceeded after 3 calls, notices ["the prompter answers at once; waiting
    for the browser"]; want the code on call 4, no notice";
  * no limit: `TestPaste_QuickEmptiesStop_0059`, "425422 Input calls,
    notices []; want 3 and one notice".
* **Step 6, docs:** `readMasked`'s, `escapeState`'s and `pastePrompt.input`'s
  docs; `CHANGELOG.md` `[Unreleased]`, Fixed, two entries.
* **The checks:** `make pre-add-check` on the 3 Go files first failed on
  gocritic's `initClause`, `if quick++; quick < pasteQuickEmpties`; with
  `quick++` on its own line, "3 file(s) clean", and the no-limit plant,
  re-run against that line, caught. `scripts/check_changelog.py`, 0
  problems; `python3 -B tools/gate.py`, "gate: 27 of 27 passed".
* **Not done here:** Phases 4 to 6.
* **Committed** by the owner as `5091c40` ("committed, proceed").

### Phase 4: the saved session and the saved fallbacks (2026-10-24)

* **Step 1, W6.** `keepExistingOAuth` sets `Store` and the client, then
  runs `validateKept`, before the `Confirm`. One that fails is not offered:
  a `LevelWarn` notice "the saved <Label> session cannot be used
  (<reason>); sign in again", the reason being `validateKept`'s error, and
  no session and no error, so the method menu follows. The error "cannot
  be kept (…); sign in again" is gone.
* **Step 2, W7.** `selectFallbacks` takes, each round, `savedBeyond`: the
  saved fallbacks neither excluded nor recommended, once each, compared
  without case. The blank round offers them after the recommended rows,
  each marked `savedFallbackDetail` ("saved"), and preselects every saved
  row; both early returns also wait for them to be gone.
* **Step 3, the drift check:** `go test -race -count=1 -shuffle=on
  ./wizard/` failed only `TestConfigureLLM_KeepRefusesStubSession`, as
  Scope foresaw: "ConfigureLLM() error = select authentication method:
  fakePrompter: unexpected Select("Choose how to authenticate with
  OpenAI:") with 1 saves, want the stub refused and no save after the
  setup's", and "1 scripted confirms left unused". Rewritten: no
  `Confirm`, one notice saying to sign in again, the method menu, and an
  API key chosen there completes the run. Then the suite: ok.
* **Step 4, the tests,** `wizard/kept_0059_test.go`, the four as written.
  The fallback tests answer `MultiSelect` with its preselection, as Enter
  does. Against `5091c40`'s code, in a scratch clone, with
  `savedFallbackDetail` declared in a shim test file so that the tests
  build, three failed:

  ```text
  --- FAIL: TestKept_UnusableNotOffered_0059
      kept_0059_test.go:59: ResolveCredential = "", fakePrompter: unexpected Confirm("Keep the existing session (••••••••0059)?") with 1 saves; want the new session saved after the setup's
  --- FAIL: TestFallbacks_SavedOffered_0059
      kept_0059_test.go:114: ConfigureLLM: fallbacks [], <nil>; want claude-legacy-fallback kept
  --- FAIL: TestFallbacks_SavedOnlyRound_0059
      kept_0059_test.go:136: selectFallbacks = [], <nil> after 0 rounds; want the saved fallback, from one round
  ```

  `TestKept_UsableStillOffered_0059` passed, as it should: the old code
  kept a usable session too.
* **Step 5, the plants** (`tools/plant.py`, a scratch clone), all four
  caught:
  * the check moved back after the `Confirm`, with the old error:
    `TestKept_UnusableNotOffered_0059`, the line above;
  * the saved rows dropped: `TestFallbacks_SavedOffered_0059` and
    `TestFallbacks_SavedOnlyRound_0059`;
  * the early return when no recommended row is left put back:
    `TestFallbacks_SavedOnlyRound_0059`, "after 0 rounds";
  * every saved session refused (added; step 5 names no plant for
    `TestKept_UsableStillOffered_0059`, which the old code passes, and a
    test is unused until it has been seen to fail):
    `TestKept_UsableStillOffered_0059`, "unexpected Select("Choose how to
    authenticate with OpenAI:") with 1 saves; want the saved session
    kept", and `TestKept_UnusableNotOffered_0059`, its notice's reason
    "planted".
* **Step 6, docs:** `keepExistingOAuth`'s, `selectFallbacks`' and
  `savedBeyond`'s docs; `CHANGELOG.md` `[Unreleased]`, Fixed, two entries.
* **The checks:** `make pre-add-check` on the 4 Go files, "4 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `python3 -B
  tools/gate.py`, "gate: 27 of 27 passed". The first gate ran before
  `savedBeyond` was moved off `excludedIDs`' doc comment, where it had been
  put between the two; the pre-add check, the plants and the gate were run
  again after the move, with the same results.
* **Not done here:** Phases 5 and 6.
* **Committed** by the owner as `7f75456` ("committed, proceed").

### Phase 5: the context, the endpoint, raw mode and the client (2026-10-24)

* **Step 1, W8.** `ConfigureLLM` returns `fmt.Errorf("wizard: %w",
  ctx.Err())` before choosing a provider; its doc says so.
* **Step 2, W9.** `validateBaseURL(p, raw, ask)`: with `ask` false, a
  plain-http remote URL is used with the warning "<URL> is plain http to a
  remote host, so the key travels unencrypted". `resolveBaseURL` passes
  false under `NoEndpointPrompt` and true for a typed URL;
  `Options.NoEndpointPrompt`'s doc says so.
* **Step 3, W10.** `isTerminal`, `makeRaw` and `restoreTerm` are package
  variables over `golang.org/x/term`'s functions; `inTTY` and `secretOnce`
  read through them.
* **Deviation (2026-10-24): step 4's call site.** Step 4 has
  `accessOnlyOpenAISession` take the client. An existing test outside
  Scope, `wizard/paste_token_0026_test.go:25`, calls it with the token
  alone, so its signature is kept, and its one caller,
  `pendingAccessOnlyOpenAI`, calls `UseHTTPClient(o.HTTPClient)` on the
  session. The session holds the client as step 4 says. No decision
  changes.
* **Deviation (2026-10-24): a race in Phase 3's test.** The first drift
  check, `go test -race -count=1 -shuffle=on ./wizard/` (seed
  1792863569902308000), failed with a data race: Phase 3's
  `TestPaste_SlowEmptyAskedAgain_0059` wrote the package variable
  `pasteQuick` (`entry_0059_test.go:78`) while a paste goroutine that
  `stubLoopbackWin0057` (`inputctx_0057_test.go:91`) had started, and its
  test did not wait for, had read it (`pastePrompt.input`). Thirty runs of
  those tests on `7f75456`'s code, and that seed, did not repeat it, but
  the write is unsynchronised with any such read. The test now slows its
  empty answers to `pasteQuick` + 10 ms, rather than shortening
  `pasteQuick`, and `pasteQuick` is a constant. Phase 3's step 2 said "a
  variable, which a test changes"; no test changes it now. No decision
  changes. Phase 3's three plants were run again on the new test: all
  caught.
* **Step 5, the drift check:** with the race fixed, `go test -race
  -count=1 -shuffle=on ./wizard/`: ok, at that seed and three fresh ones,
  every existing test unchanged.
* **Step 6, the tests,** `wizard/edges_0059_test.go`, the four as written.
  The raw-mode test reads from an `os.Pipe` through the seam: raw mode,
  raw mode refused, and the restore failing. `secretOnce` is 100.0 %
  covered (`go tool cover -func`), from 26.7 %. Against `7f75456`'s code,
  in a scratch clone, with the three seam variables declared in a shim
  test file so that the tests build, all four failed:

  ```text
  --- FAIL: TestConfigure_EndedContextAsksNothing_0059
      edges_0059_test.go:30: err select provider: fakePrompter: unexpected Select("Choose an LLM provider:"), shown […]
  --- FAIL: TestBaseURL_NoPromptPlainHTTP_0059
      edges_0059_test.go:45: base URL "", confirm base URL: fakePrompter: unexpected Confirm("http://models.example.test/v1 is plain http to a remote host, so the key travels unencrypted. Use it?"), …
  --- FAIL: TestSecret_RawMode_0059/raw_mode
      edges_0059_test.go:122: secretOnce = "sk-raw-0059-keyAA", EOF; 0 raw, 0 restores; want the key, 1 raw, 1 restores
  --- FAIL: TestPastedToken_KeepsClient_0059
      edges_0059_test.go:143: resolveTokenStdin: <nil>; session OAuthSession{…}; want one holding the caller's client
  ```

* **Step 7, the plants** (`tools/plant.py`, a scratch clone), all four
  caught:
  * the context check removed: `TestConfigure_EndedContextAsksNothing_0059`,
    the line above;
  * the `Confirm` put back: `TestBaseURL_NoPromptPlainHTTP_0059`, the line
    above;
  * `restoreTerm`'s failure ignored:
    `TestSecret_RawMode_0059/the_restore_failing`, "shown "Key: \r\n";
    want "error: failed to restore terminal mode: bad descriptor"";
  * `UseHTTPClient` dropped: `TestPastedToken_KeepsClient_0059`.
* **Step 8, docs:** `ConfigureLLM`'s, `Options.NoEndpointPrompt`'s,
  `validateBaseURL`'s, the seam's and `pendingAccessOnlyOpenAI`'s docs;
  `CHANGELOG.md` `[Unreleased]`, Fixed, three entries (W10 adds tests, not
  a behaviour).
* **Found, not in this PLAN's scope:** with `-count=2`,
  `TestResolveCredential_AsksNoProviderNoModel_0056` and
  `TestResolveCredential_ErrorKinds_0056/ctx_ended_during_the_listing` fail
  on `7f75456`'s code as on this phase's: "catalog live true, usable
  ["claude-haiku-5-5"], after 0 listings" and "error <nil>; want
  context.Canceled". They list through the process cache, not
  `testCache(t)`, so a repeat finds the first run's listing (the class
  0058-MADR treated). CI and this PLAN run `-count=1`, where they pass. Left
  for the owner to place.
* **The checks:** `make pre-add-check` on the 6 Go files, "6 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `python3 -B
  tools/gate.py`, "gate: 27 of 27 passed".
* **Not done here:** Phase 6.
* **Committed** by the owner as `d4d0828` ("committed").
* **The tests found beyond this PLAN** (Phase 5's "Found, not in this
  PLAN's scope") were placed by the owner in 0058: its MADR's amendment of
  2026-10-24 and its PLAN's Phase 6, committed as `f99183a` and `858f565`.
  Ten wizard tests take their own cache, and `go test -count=2 ./...`
  passes.

### Phase 6: release `v1.22.1` (2026-10-24)

* **Approval.** "committed, proceed", after 0058's Phase 6.
* **Step 1.** `CHANGELOG.md`: `[v1.22.1] - 2026-10-24` under an empty
  `[Unreleased]`, holding Phases 1 to 5's nine Fixed entries (W1 to W9
  and W11; W10 adds tests, not a behaviour) and no Added; the comparison
  links `[Unreleased]` from `v1.22.1` and `[v1.22.1]` from `v1.22.0`. The
  README's Status names `v1.22.1`. `scripts/check_changelog.py`: "32
  release section(s), 0 problem(s)".
* **Step 2, before the push:**
  * `tools/apidiff.py v1.22.0`: no output, exit 0, so no change to the
    exported API, compatible or not;
  * the notes: on a scratch clone of the working tree with a local
    annotated `v1.22.1`, `scripts/check_changelog.py --section v1.22.1
    --repo maccavelli/go-llmprovider-sdk` printed the nine Fixed entries,
    each record link pinned to `blob/v1.22.1`; `tools/denyscan.py` on
    them: "0 hit(s) in 1 file(s)";
  * `python3 -B tools/gate.py`: "gate: 28 of 28 passed";
  * the disclosure guard runs over the outgoing commits once the owner has
    committed this step, before the tag.
* **Committed** by the owner as `41dfee4`. The disclosure guard over the
  eleven outgoing commits, `a3a2d9b` to `41dfee4`: exit 0, nothing
  reported. The owner pushed `main`; CI run 40124869685: `validate` on
  macos-15, windows-2025 and ubuntu-24.04, success; `release` skipped, as
  on a branch. Docs run 40124869681: success.
* **Step 3.** The owner tagged `v1.22.1`, annotated, on `41dfee4`, and
  pushed it; CI run 40125535484: `validate` on the three systems and
  `release`, success. The Release `v1.22.1`: not a draft or a prerelease,
  Latest. `gh release view v1.22.1 --json body` equals step 2's notes line
  for line, apart from one trailing blank line.
* **Step 4.** On the owner's word ("delete the v1.19.0 release"),
  `gh release delete v1.19.0 --yes`: exit 0. The Releases are `v1.22.1`,
  `v1.22.0`, `v1.21.0`, `v1.20.1` and `v1.20.0`, five (R53); the tag
  `v1.19.0` stays on `origin`, and its section in `CHANGELOG.md`.
  `tools/consumer.py v1.22.1`: "consumer: 15 packages linked", the six
  cross-builds and `go run` exit 0, "every step passed". `tools/offline.py
  v1.22.1`: "requests=0", "offline: ok", the six cross-builds, and its
  two plants, "models.dev left on" and "no caller layer", each caught;
  "every step passed".
* **The docs that name a release:** the README's Status names `v1.22.1`
  (step 1). The README, `CHANGELOG.md`'s preamble and
  `docs/architecture.md` state the five-Release rule, not a list, and hold
  as written. `0051-PLAN-opt-in-streaming-retry.md` links the deleted
  `v1.19.0` Release in its execution record; like the records of the
  Releases deleted before it, `v1.10.2` to `v1.18.0`, it is history and is
  not rewritten. The tag, and its changelog section, remain.
* **Step 5, the MADR's Confirmation, item by item:**
  * **"P1 to P8, made tests, each failing on `2d6e7f3` and passing after
    its phase."** Met in substance. Each was seen to fail on the commit
    before its phase, `2d6e7f3` for Phase 1, then `ba45339`, `4eff4c7`,
    `5091c40` and `7f75456`, not on `2d6e7f3` itself, and passes after it
    (each phase's record). P1 and P2 are Phase 1's, P8 Phase 2's, P3 and
    P4 Phase 3's, P5 and P7 Phase 4's, P6 Phase 5's.
  * **"For W1: a key of `sk-ant-api04-`, of 109 characters, and the
    standard one, each accepted by a stubbed listing, with no warning; a
    refused key without the shape, with the shape named in the refusal
    notice."** Met in substance by `TestShape_AcceptedKeyNoWarning_0059`:
    the other tag (`sk-ant-api04-`, 108 characters) and the other length
    (`sk-ant-api03-`, 109) are two keys, beside the standard one, each
    accepted with no notice; `TestShape_RefusalNamesShape_0059` names the
    shape on each refusal of a key without it.
  * **"`make pre-add-check`, `python3 -B tools/gate.py`, and CI on the
    three platforms."** Met: each phase's checks, the gate at 27 and then
    28 checks, and runs 40124869685 and 40125535484.
  * **"`secretOnce` covered by a test (W10)."** Met: 100.0 %, Phase 5.
* **The Verification table:** V1 to V12 by the phases' tests; V13 by
  step 2's `tools/apidiff.py v1.22.0`; V14 by each phase's red run.
* **The PLAN is `complete`.** The MADR stays `accepted`.
* **Committed and pushed** by the owner as `9ff2366`.

### Phase 7: Claude's refusal hint by its prefix (2026-10-24)

* **Approval.** The owner committed the amendment as `2897263`, answered
  Q8 (a), and approved ("q8: option a. committed, proceed.").
* **Step 1.** `tools/records.py status`: this PLAN `in-progress`; the MADR's
  amendment records Q8's answer.
* **Step 2.** `keyShapes`' Claude entry is `^sk-ant-[A-Za-z0-9]+-`, its text
  "sk-ant- and a tag, such as sk-ant-api03- or sk-ant-usr-"; the doc comment
  says Anthropic's tag and length vary, so only its prefix and a tag are
  checked (W14).
* **The drift check,** before step 3: `go test -race -count=1 -shuffle=on
  ./wizard/` failed only `TestShape_RefusalNamesShape_0059/without_the_shape`,
  the test step 3 names, "0 notices name the shape; want 3", the notices
  now holding the new text.
* **Step 3.** `TestShape_RefusalNamesShape_0059` counts "API keys look like
  sk-ant-", the stem of both texts, so that the old code's hints are
  counted too, and gains "a usr- key" and "an api03- key of another
  length", each wanting none. Against `2897263`'s code, in a scratch clone,
  the two new cases failed and the two old ones passed:

  ```text
  --- FAIL: TestShape_RefusalNamesShape_0059/a_usr-_key
      shape_0059_test.go:144: 3 notices name the shape; want 0: [… "Claude (Anthropic) API keys look like sk-ant-api0…
  --- FAIL: TestShape_RefusalNamesShape_0059/an_api03-_key_of_another_length
      shape_0059_test.go:144: 3 notices name the shape; want 0: […]
  ```

  A first form counted the new text alone, which the old code never
  prints, so the new cases would have passed on it; the stem replaced it
  before the red run.
* **Step 4.** `go test -race -count=1 -shuffle=on ./wizard/`: ok; `go test
  -count=2 ./wizard/`: ok.
* **Step 5, the plants** (`tools/plant.py`, a scratch clone), both caught:
  * the old exact entry put back: "a usr- key" and "an api03- key of another
    length", "3 notices name the shape; want 0";
  * the prefix dropped (an empty pattern, every key matching):
    "without the shape", "0 notices name the shape; want 3".
* **Step 6.** `CHANGELOG.md` `[Unreleased]`, Fixed (W14).
  `docs/architecture.md`'s "the provider's usual shape" holds.
* **The checks:** `make pre-add-check` on the 2 Go files, "2 file(s)
  clean"; `scripts/check_changelog.py`, 0 problems; `python3 -B
  tools/gate.py`: "gate: 28 of 28 passed".
* **Not done here:** Phase 8.
* **Committed** by the owner as `fb2060d` ("committed, proceed").

### Phase 8: release `v1.22.2` (2026-10-24)

* **Approval.** "committed, proceed", after Phase 7.
* **Step 1, as Phase 6's.** `CHANGELOG.md`: `[v1.22.2] - 2026-10-24`
  under an empty `[Unreleased]`, holding Phase 7's Fixed entry (W14) and no
  Added; the links `[Unreleased]` from `v1.22.2` and `[v1.22.2]` from
  `v1.22.1`. The README's Status names `v1.22.2`.
  `scripts/check_changelog.py`: "33 release section(s), 0 problem(s)".
* **Before the push:**
  * `tools/apidiff.py v1.22.1`: no output, exit 0;
  * the notes, on a scratch clone of the working tree with a local
    annotated `v1.22.2`: `--section v1.22.2 --repo
    maccavelli/go-llmprovider-sdk` printed the one Fixed entry, its link
    pinned to `blob/v1.22.2`; `tools/denyscan.py` on them, 0 hits;
  * `python3 -B tools/gate.py`: "gate: 28 of 28 passed";
  * the disclosure guard runs once the owner has committed this step.
* **Committed** by the owner as `97cc0b2`. The disclosure guard over the
  three outgoing commits, `2897263` to `97cc0b2`: exit 0, nothing reported.
  The owner pushed `main`; CI run 40139067065: `validate` on macos-15,
  windows-2025 and ubuntu-24.04, success; `release` skipped. Docs run
  40139067094: success.
* **The tag.** The owner tagged `v1.22.2`, annotated, on `97cc0b2`, and
  pushed it; CI run 40140108331: `validate` on the three systems and
  `release`, success. The Release `v1.22.2`: not a draft or a prerelease,
  Latest; its body equals the scanned notes, apart from trailing blank
  lines.
* **After the tag.** On the owner's word ("delete v1.20.0 release"),
  `gh release delete v1.20.0 --yes`: exit 0. The Releases are `v1.22.2`,
  `v1.22.1`, `v1.22.0`, `v1.21.0` and `v1.20.1`, five (R53); the tag
  `v1.20.0` stays on `origin`, and its section. Its Release URL answers 200
  with the tag's page, naming the tag and its commit `5c60593`, with no
  notes body, as the README's and `CHANGELOG.md`'s sentence says
  (0060-MADR's amendment). `tools/consumer.py v1.22.2`: "15 packages
  linked", "every step passed". `tools/offline.py v1.22.2`: "requests=0",
  "offline: ok", its two plants caught, "every step passed".
* **The amendment's decision, held:** a refused `sk-ant-usr-` key, or an
  `sk-ant-api03-` key of another length, draws no shape notice; a refused
  key without the `sk-ant-<tag>-` prefix draws one per refusal
  (`TestShape_RefusalNamesShape_0059`, Phase 7).
* **Not here:** prepare-commit-msg's bump to `v1.22.2`, its own change.
* **The PLAN is `complete` again.** The MADR stays `accepted`.
