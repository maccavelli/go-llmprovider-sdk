---
status: in-progress
date: 2026-11-12
associated-madr: "0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the ChatGPT Plan's Consent on the Session, a Host-Lifetime Host Id, and Reuse of the Issued Client

Associated MADR: [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)

## Goal

After this PLAN:

* the `chatgpt_plan` method states that it needs the consent
  `chatgpt_plan.background_use` (MADR D1);
* its sign-in asks it before the browser, as one `confirm` step carrying the
  notice, and the answer is kept on the session it returns (D2, D4);
* `auth.SetConsent` changes the answer, and no refresh in any process loses
  it (D3);
* a request marked with `llmprovider.WithBackground` on a plan session
  without the consent is refused before the network (D5);
* the host id is the consumer's, or kept by the store, and the issued client
  survives a sign-out or a switch (D6, D7), if OpenAI accepts its reuse.

It ships in the same SDK minor release as
[0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)
(`v1.31.0` if
[0076-PLAN-auth-error-kinds-and-doc-drift.md](0076-PLAN-auth-error-kinds-and-doc-drift.md)
ships as `v1.30.0`), and depends on its Phases 2 (the field copies), 4
(`x/signin`) and 5 (the wizard's renderer). So it inherits that PLAN's
prerequisites: `authcontract/v0.1.0` tagged
([0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md)
Phase 4), and 0076's minor released first.

The MADR is `proposed`. Nothing below starts before the owner accepts it and
approves this PLAN.

## Scope

### In scope

* **D2, D3:** `llmprovider/auth/tokenstore.go`, `tokenstore_file.go`,
  `oauth_session.go` (the carry-over only), a new `llmprovider/auth/consent.go`.
* **D1, D4:** `llmprovider/descriptor.go`, `llmprovider/providers/openai/descriptor.go`,
  a new `llmprovider/providers/openai/consent.go`, `internal/signinflow/`,
  `llmprovider/x/signin/`, `wizard/auth.go`, `wizard/configure.go`.
* **D5:** a new `llmprovider/background.go`; `llmprovider/providers/openai/openai.go`;
  `llmprovider/auth/consent.go`, for `StoredConsent` *(added 2026-11-11)*.
* **D6, D7:** `llmprovider/auth/oauth_plan.go`, a new
  `llmprovider/auth/clientstore.go`, `tokenstore_file.go`, `wizard/auth.go`,
  `wizard/configure.go`, `llmprovider/x/signin/`.
* **Docs:** `llmprovider/auth/doc.go`, `README.md`'s login section,
  `docs/architecture.md`, `CHANGELOG.md`, and the amendments the MADR lists.

### Exported names (R55)

Outside `llmprovider/x/`:

| Name | Call site that needs it |
| :--- | :--- |
| `AuthMethod.Consent`, `.ConsentText`, `.ConsentURL` (`string`) | `signin.Methods`'s `Consents` and `Links`; gobble's and Magic's display of the requirement before a sign-in starts |
| `openai.ConsentBackgroundUse` | the descriptor; the provider's refusal (D5); gobble's `honouredConsents` and its `gobble auth consent` word `background` (gobble-cli `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D2, `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D3; the report's G13) |
| `OAuthSession.Consents`; `(*OAuthSession).Consent` | the plan sign-in's check of whether `Previous` already holds an answer (MADR D4; Phase 2 step 3); gobble's `auth status --json` `consents` member (gobble-cli `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D8; the report's G15) *(2026-11-11: the first call site read "the provider's refusal", which now reads through `StoredConsent`, the next row)* |
| `(*OAuthSession).StoredConsent` *(added 2026-11-11 by the owner's ruling; MADR D5)* | the openai provider's check of a marked request on a plan session (Phase 3 step 3), which cannot read the session's `Store` without a data race (MADR C43 to C45) |
| `auth.SetConsent` | gobble's `provider.SetConsent` in its `llm/provider`, the one gobble package that may import this SDK, which serves `gobble auth consent <provider> [<consent> allow\|deny]` (gobble-cli `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D3, the owner's QF (a)) and its one-time migration of a stored `chatgptPlanBackground: true` (that record's D9; the report's G16) |
| `llmprovider.WithBackground`, `llmprovider.IsBackground` | gobble's session titles and auto-compaction, which it marks (gobble-cli `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D2; the report's §6.1, part 4) |
| `type auth.ClientRecord` and its `ClientID`, `HostID`; `type auth.ClientStore` and its `LoadClient`, `SaveClient`; `(*FileTokenStore).LoadClient`, `.SaveClient` | the wizard's and `x/signin`'s plan sign-in; gobble's `sdkStore`, which implements `ClientStore` over `<state>/clients.json`, a file apart from its `auth.json` (gobble-cli `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D5, the owner's QG (a)) |
| `ChatGPTPlanOptions.ClientID` | the wizard's and `x/signin`'s reuse of the issued client (D7) |
| `wizard.Options.HostID`, `wizard.Options.ConsentText` | gobble's host id from `<state>/host-id` (gobble-cli `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D4, building gobble-cli `docs/decisions/0025-MADR-provider-choice-and-auth.md` D6), and its own wording (Q2; that record's D2) |

Under `llmprovider/x/signin`: `Options.HostID`, `Options.ConsentText`,
`Options.ClientStore` (MADR D7: `x/signin`'s way to the store; the wizard
reads its `TokenStore` instead). gobble's ACP sign-in sets all three, and
`Options.Previous` from its stored session, when it calls `signin.Start`
(gobble-cli
`docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
P6 step 1). No other name is exported.

### Out of scope

* What gobble marks as background, its wording, its setting and its
  migration (its records).
* A keyed store for several OpenAI sessions (Q12 (b), not chosen).
* Any change to the refresh code beyond MADR D3's carry-over.

## Rules for every phase

* As [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)'s
  "Rules for every phase", including its 66 refresh tests, run with
  `python3 -B tools/redgreen.py REFRESH.json --expect pass` before staging
  Phases 1, 3 and 4, with their 15 files unchanged. `REFRESH.json` is the spec
  0074-PLAN's rules name: `{"tests": {"./llmprovider/auth/": [the 66
  names that PLAN lists]}}`, kept in the session's scratch directory.
  *(2026-11-11: the spec's shape spelled out, and Phase 4 added, since it
  edits `tokenstore_file.go` too. Later that day: Phase 3 added, since its
  `StoredConsent` reads under the session's lock.)*
* **Seven existing tests change by this record's decision,** and only these
  (MADR C31 to C37, the seven callers of the `stubPlanLogin` seam). Each
  keeps what it asserts:

  | Test | The change | What it keeps asserting |
  | :--- | :--- | :--- |
  | `TestPlan_FirstSignIn_0043` (`wizard/plan_0043_test.go:93`) | scripts a third `Select` (allow) for its `Confirm`; reads the notice from `seenSelect` | the notice is shown once, with "You're using your ChatGPT plan" and `openai.ChatGPTUsageURL`; one login; the plan session saved |
  | `TestPlan_NoticeDeclined_0043` (`:148`) | scripts a third `Select`, "Cancel the sign-in", for its `false` | the sign-in ends "declined", with no login and no save |
  | `TestPlan_ReSignIn_0043` (`:165`) | its stored session (`:177`) gains `Consents: "chatgpt_plan.background_use=granted"` | `Previous` is the saved plan session; no notice; one login, one save; the usual paste instruction |
  | `TestPlan_ReplacesCodexSession_0043` (`:196`) | scripts a third `Select` (allow) for its `Confirm` | no `Previous` for a Codex-id session; the plan session replaces it |
  | `TestResolveCredential_ChatGPTPlan_0056` (`wizard/resolve_0056_test.go:71`) | scripts a second `Select` (allow) for its `Confirm`; reads the notice from `seenSelect` | one plan sign-in, oauth, no model; the notice with the usage page; no input |
  | `TestNoPastePrompt_PlanSignIn_0057` (`wizard/paste_0057_test.go:55`) | scripts a second `Select` (allow) for its `Confirm` | no `InputCode` under `NoPastePrompt`; an `OpenURL`; the plan offered and signed in |
  | `TestConfigureLLM_PlanLoginErrorBare_0049` (`wizard/signin_errors_0049_test.go:40`) | scripts a third `Select` (allow) for its `Confirm` | the plan login's error returned bare |

  Each change is named in the execution record with the diff of its file.
  Any other existing test that needs a change is a stop. (A first draft named
  two of the seven; the review of 2026-11-10 found four, and a search for
  every `stubPlanLogin` caller found the other three.)

* **Phase 2's bounded amendment (2026-11-13, the owner's ruling).** Phase
  2's dry run, on a scratch clone of `f64134d`, found six more existing
  tests than the seven above that its own changes fail, which the rule above
  makes a stop. The owner ruled "Bounded amendment": an existing test may
  change only (a) by adding the consent fields to an expected value, or (b)
  by answering the consent step first, which shifts the Seq numbers of the
  steps after it to match. Every other assertion stays as it was, and a
  test that needs any other kind of change still stops the phase. The six,
  with the first failure line of the dry run:

  | Test | File and line | Kind | Dry run's failure |
  | :--- | :--- | :--- | :--- |
  | `TestDescriptors_OpenAIAndGrokOfferOAuth` | `llmprovider/providers/descriptors_test.go:36`, the plan row at `:107` | (a) its expected plan row gains `Consent`, `ConsentText` and `ConsentURL`, word for word as MADR D1 | `descriptors_test.go:183: openai AuthMethods = ...` |
  | `TestMethods_0074` | `llmprovider/x/signin/methods_0074_test.go:108`, `assertMethod` | (a) the expected plan method gains `Consents`, `Links` and the `consent` feature | `methods_0074_test.go:77: openai chatgpt_plan:` |
  | `TestStart_PlanSteps_0074` | `llmprovider/x/signin/start_browser_0074_test.go:97`, both subtests | (b) each answers the consent first; its `open_url` and `paste` steps are numbered 2 and 3 | `start_browser_0074_test.go:113: no step in 10 s` |
  | `TestStart_BadPaste_0074` | `start_browser_0074_test.go:139`, the subtest at `:166` | (b) answers the consent first; its steps are numbered one higher | `start_browser_0074_test.go:173: step = confirm #1; want open_url #1` |
  | `TestStart_NoLoopbackPassedOn_0074` | `start_browser_0074_test.go:334`, its plan half | (b) answers the consent first, so the login starts | `start_browser_0074_test.go:349: the login was never started` |
  | `TestWizardIdentity_0076` | `wizard/identity_0076_test.go:33`, the `plan` subtest at `:71` | (b) scripts a `Select` (allow) where it scripted the notice's `Confirm`, as the seven do | `identity_0076_test.go:80: fakePrompter: unexpected Select(...)` |

  The dry run's failing existing tests are these six and the seven above, and
  no other. A test the tree shows to need a change of the
  two kinds, and no other, may be added here, with its file, line and kind,
  before it is changed; any other kind is a stop.

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR and their index rows (`tools/records.py add`).
2. On the owner's acceptance, `tools/records.py status` sets the MADR
   `accepted` and this PLAN `in-progress`.
3. **Apply the supersession notes** the MADR lists: dated amendments to
   `0043-MADR-openai-chatgpt-plan-sign-in.md` for D8, D7, D3 and D4, each
   quoting the text it amends, and a note on
   `0017-REPORT-openai-sign-in-for-agents.md`'s open item "The terms in a
   consumer".
4. `python3 -B tools/gate.py --markdown`, then stage.

### Phase 1: the answer on the session, and a refresh that keeps it (D2, D3)

1. `OAuthSession.Consents`, `Consent(id)`; `fileRecord` gains `consents`,
   `omitempty`; the formatters show it; the five copies carry it (0074-PLAN
   Phase 2's guard covers it with no change to the guard).
2. `llmprovider/auth/consent.go`: `SetConsent`, as MADR D3.
3. The carry-over in `reloadOrRefresh` and `persistRotation`, as MADR D3 and
   no wider: the stored record that `loadRotated` would pass over is kept,
   in both of its cases (the session's own refresh token, and `state.stored`,
   the token an unsaved rotation spent; MADR C16, C40), and its `Consents`
   set on the session about to be saved. *(2026-11-11: the second case
   added.)*

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestConsents_ParseAndFormat_0075` | `Consent` for granted, declined and unanswered ids; entries sorted; an unknown token ignored | `declined` read as granted |
| `TestConsents_StoreRoundTrip_0075` | `consents` round-trips through `FileTokenStore`; a record without it loads unanswered | the JSON name spelled `consent` |
| `TestSetConsent_0075` | it holds the refresh lock throughout (a fake `RefreshLocker` records the order); no session and an empty id are `ErrInvalidRequest`; the store's error is returned as it is | the lock taken after `Load`; a missing session saved as a new one |
| `TestConsentSurvivesRefresh_0075` | a session loaded from a `FileTokenStore`, then `SetConsent` through a second `FileTokenStore` on the same directory, then the session's refresh against the fake issuer: the store still holds the answer, and the live session reports it | the carry-over removed from `reloadOrRefresh` |
| `TestConsentSurvivesResave_0075` | the same, through a failed save and its re-save | the carry-over removed from `persistRotation` |
| `TestConsentSurvivesUnsavedRotationRefresh_0075` | a session whose rotation is unsaved (its first save fails), then `SetConsent` through a second handle, then a refresh once the token is due: the store keeps the answer | the carry-over limited to a record holding the session's own token |
| `TestConsentAdoptedFromSibling_0075` | a sibling's rotation adopted from the store brings its consents | `adopt` drops `Consents` |

`TestConsentSurvivesRefresh_0075` is also seen to fail on one more plant,
the carry-over removed from both functions, which is `773b84c`'s refresh
code with only the field added: the record of the gap MADR C16 and C17
leave. *(2026-11-11: run as a `tools/plant.py` plant, since `redgreen.py
--rev 773b84c` cannot build a test that names the new field;
`TestConsentSurvivesUnsavedRotationRefresh_0075` added for the second case
of step 3.)*

Acceptance: the tests pass, each plant failed; the 66 refresh tests pass with
their files unchanged.

### Phase 2: the fact and the question (D1, D4)

1. `AuthMethod.Consent`, `ConsentText`, `ConsentURL`; `openai.ConsentBackgroundUse`;
   the `chatgpt_plan` descriptor's values, word for word as MADR D1.
2. `signin.Methods`: `Consents`, the usage link in `Links`, `consent` in
   `Features`.
3. `internal/signinflow`: the plan's `confirm` step before `open_url`, when
   MADR D4 says; the answer put on the returned session; a re-sign-in's
   recorded answer carried.
4. `signin.Options.ConsentText` and `wizard.Options.ConsentText`.
5. The wizard's renderer: a consent `confirm` as MADR D4's three-way
   `Select`, its title the step's `text` and the usage page's URL. The seven
   tests of "Rules for every phase" change as its table says, in this phase.

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestDescriptors_PlanConsent_0075` | the plan's three fields, word for word; no other method has a consent | the text's "background" dropped |
| `TestMethods_Consent_0075` | the plan's `Method` has the consent, required, with the usage link, and `consent` in `Features` | `Required` false |
| `TestPlanConfirmStep_0075` | a first sign-in's steps begin `confirm`, then `open_url`; the step's id, text, link and `default` false; the app's text when given | `confirm` after `open_url`; `default` true |
| `TestPlanConsentAnswer_0075` | `true` gives `chatgpt_plan.background_use=granted` on the session `Wait` returns; `false` gives `declined`; both go on to `open_url` | `false` ends the sign-in |
| `TestPlanConsentCarried_0075` | a re-sign-in whose `Previous` has an answer shows no `confirm` and carries it; one without an answer shows it | the answer not carried |
| `TestPlanConsentNotRecordedOnFailure_0075` | a sign-in that fails after the answer returns no session, and the store is untouched | the answer saved at the step |
| `TestWizardPlanConsentSelect_0075` | the wizard's three choices; the second is the default; the title holds the step's text and `openai.ChatGPTUsageURL`; "Cancel the sign-in" gives "declined", with no login and no save | the default set to the first; the usage URL left out of the title |
| `TestWizardReSignInWithoutAnswerAsks_0075` | a stored plan session with no answer: the re-sign-in shows the consent `Select` before the login, passes the session as `Previous`, and records the choice on the saved session | the `Select` skipped for any stored plan session |
| the seven tests of "Rules for every phase" (changed) | what each asserted before, as that table says | — (each change is reviewed against the table) |

Acceptance: the tests pass, each plant failed; every other wizard test passes
unchanged.

### Phase 3: the enforcement (D5)

1. `llmprovider/background.go`: `WithBackground`, `IsBackground`, an
   unexported context key type.
2. `llmprovider/auth/consent.go`: `(*OAuthSession).StoredConsent(ctx, id)
   (granted, answered bool, err error)` (MADR D5). It takes `Store` under
   the session's lock and loads from it. With no `Store` it gives the
   session's own answer. An error from the store's `Load` is returned as it
   is, as `SetConsent` returns the store's (MADR D3). Its doc says so, in
   R51's and R52's wording. *(Added 2026-11-11, by the owner's ruling.)*
3. `providers/openai/openai.go`: `Generate` and `Stream` check, before the
   network, a marked request on a plan session, through `StoredConsent`: the
   answer from the session's `Store` when set, else from the session; not
   granted is an error matching `ErrNotPermitted` naming
   `openai.ConsentBackgroundUse`, and so is a read that fails.
   * **Ruled by the owner, 2026-11-11: option (a), and a failed read is
     refused as not granted.** Until then this step carried an open
     question and a stop before it: the provider, in another package, cannot
     read the session's `Store` without a data race (MADR C43 to C45), and
     the R55 table had no name that reads the stored answer. The options, as
     they were put:
     (a) *recommended, and chosen:* one more exported method, such as
     `(*OAuthSession).StoredConsent(ctx, id) (granted, answered bool, err
     error)`, which takes `Store` under the session's lock and loads from it,
     added to the R55 table with this step as its call site; cost, one name
     under the v1 promise; (b) change D2's `Consent` to take a `ctx` and read
     the store itself; cost, a slower method that can fail, used by every
     caller that only wants the session's own answer; (c) read the
     session's field only; cost, "from the next request" holds only in the
     process that called `SetConsent`, against Q5 and MADR D5. For a failed
     read the choice was: refused as not granted (recommended, as the terms
     ask for express consent, and chosen) or sent.
     `TestOpenAIBackground_ReadsStore_0075` and a race run (`-race`) of a
     marked request beside a refresh check the choice, as before.

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestWithBackground_0075` | the mark is read back; an unmarked context reads false; a child context keeps the mark; a `context.WithValue` under the string `"background"` does not mark | `IsBackground` returns true for every context; the key made a plain string |
| `TestOpenAIBackground_Refused_0075` | a marked request on a plan session, unanswered or declined, is `ErrNotPermitted`, and the `httptest` server receives nothing; through `Generate` and `Stream` | the check placed after the request; `Stream` unchecked |
| `TestOpenAIBackground_Allowed_0075` | granted, or unmarked, the request is sent; a Codex session and a key, marked, are sent | an unmarked request refused |
| `TestOpenAIBackground_ReadsStore_0075` | a `SetConsent` through a second `FileTokenStore` handle, withdrawing the consent, refuses the very next marked request of a provider built before it | the answer read from the session only |
| `TestOpenAIBackground_ReadFails_0075` *(added 2026-11-11)* | a marked request on a plan session whose own answer is granted, and whose store's `Load` fails, is refused: the error matches `ErrNotPermitted` and names `chatgpt_plan.background_use`, and the `httptest` server receives nothing; through `Generate` and `Stream` | a failed read falls back to the session's own answer, so the request is sent |
| `TestStoredConsent_0075` *(added 2026-11-11)* | after a `SetConsent` through a second `FileTokenStore` handle, `StoredConsent` gives the store's answer while `Consent` still gives the session's own; with no `Store`, the session's own; a store whose `Load` fails gives that error | the stored record ignored, and the session's own answer returned; a `Load` error swallowed |

The race run: `go test -race -count=1 -run
'TestStoredConsent_0075|TestOpenAIBackground_ReadsStore_0075'
./llmprovider/auth/ ./llmprovider/providers/openai/`, in which each test
makes its read beside a refresh of the same session, the refresh that
rewrites `Store` (MADR C44). A race reported is a stop. *(2026-11-11: the
run was named in step 3's open question; it is kept, and written out.)*

Acceptance: the tests pass, each plant failed; the openai provider's G-wire
goldens are unchanged (R45); `llmtest.Run` passes for openai.

### Phase 4: the host id and the issued client (D6, D7)

1. `llmprovider/auth/clientstore.go`: `ClientRecord`, `ClientStore`;
   `FileTokenStore`'s `LoadClient` and `SaveClient` on `<provider>.client`,
   with `Save`'s temp file restricted through `tokenStoreRestrict` before its
   first byte (owner-only: mode 0600 on Unix, a protected DACL on Windows),
   fsync and rename; the provider id checked by `validateProviderID`; a zero
   record removes the file; `Delete` leaves it.
2. `ChatGPTPlanOptions.ClientID`, as MADR D7; `planIssuedClient` treats it as
   `Previous`'s client, and `planAuthorizeURL` names it with no
   `agent_name_hint` and no `id_token_hint`.
3. `wizard.Options.HostID`, `signin.Options.HostID` and
   `signin.Options.ClientStore`; the host id's order (MADR D6, as revised
   2026-11-11): `Options.HostID`, else `Previous`'s, else the record's, else
   a new one.
4. The wizard and `x/signin`: load the record before a plan sign-in with no
   plan session to reuse (the wizard only when its `Options.TokenStore`
   asserts to `auth.ClientStore`); save the issued client and the host id as
   soon as the sign-in returns. A sign-in with `ClientID` and no `Previous`
   still asks for the whole callback URL, as a first sign-in does, so
   0074-MADR D1's `FirstPaste` `url_only` stays true for it. *(2026-11-11:
   the paste rule and the wizard's type assertion spelled out; step 1's
   owner-only check names the seam `tokenstore_private_test.go` already
   observes.)*

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestFileTokenStore_Client_0075` | save, load, a zero record removes; the temp file passes `tokenStoreRestrict` before its first byte (the order `TestFileTokenStore_RestrictsDirAndTempBeforeWrite` checks for `Save`), and on Unix the file is mode 0600; `Delete` of the session leaves it; a bad provider id is `ErrInvalidProvider` | `Delete` removes the client file; the restriction skipped |
| `TestPlanClientID_0075` | with `ClientID`, the authorize request names it, has no `agent_name_hint` and no `id_token_hint`; a callback naming another client is refused | `agent_name_hint` still sent |
| `TestHostIDOrder_0075` | `Options.HostID`, then `Previous`'s, then the record's when there is no `Previous`, then a new one, which is saved | the record's id preferred to the option's; the record's preferred to `Previous`'s |
| `TestWizardPlanReusesClient_0075` | after `Logout`, a plan sign-in passes the stored client and host id; after a first sign-in, the issued client and the host id are saved before the listing | the record saved only after the listing |
| `TestWizardPlanSwitchKeepsClient_0075` | a plan session replaced by a Codex-id one (0043-MADR D3), then a plan sign-in, reuses the client | the client file removed on replacement |

*(2026-11-11: `TestHostIDOrder_0075` follows MADR D6 as revised, with a
second plant; `TestFileTokenStore_Client_0075`'s owner-only check and plant
name the restriction seam, since no Windows DACL helper is reachable from
`llmprovider/auth`'s tests.)*

Acceptance: the tests pass, each plant failed; the plan login tests of
`plan_login_0043_test.go` pass unchanged.

### Phase 5: docs and the changelog

1. `llmprovider/auth/doc.go`: the consent paragraph says what the SDK now
   carries, and what stays the consumer's (MADR D8).
2. `README.md`'s "Signing in with a ChatGPT plan" section, its "What stays
   yours" item: the same, with `WithBackground` and `SetConsent`.
3. `docs/architecture.md`: the consent, the client store, the mark.
4. `CHANGELOG.md` `[Unreleased]`: **Added** the names above; **Changed** the
   wizard's plan notice, now one question with three choices.
5. `make docs-check changelog-check links-check parity-check`.

### Phase 6: owner-run probes

Each is owner-run, asks first, uses a scratch store and a scratch home, and
is never assumed done. Results go in this record.

1. **Reuse of the issued client with no hint** (the owner's SDK-2; beside
   the report's U3, which probes `id_token_hint` alone),
   `TestLive_0075ClientReuse`
   in `llmprovider/live_0075_client_test.go` (build tag `live_gateways`),
   under `LLMPROVIDER_LIVE_0075` and `LLMPROVIDER_LIVE_CHATGPT_PLAN`: a first
   plan sign-in; its client saved; the session revoked and deleted, as
   `wizard.Logout` does; a second sign-in with `ClientID` and the host id
   only. It records whether OpenAI accepts it, and whether the client id is
   unchanged. It logs no token and no client id. *Owner-run; asks first;
   scratch store; quota: none (no generation); the first sign-in registers
   one client under the owner's ChatGPT account.*
   * A refusal is a deviation: MADR D7 is amended to keep the host id only,
     the wizard and `x/signin` stop passing `ClientID`, and their tests
     change by that amendment.
2. **The consent end to end,** in the same test after the second sign-in:
   decline, then a marked request is refused with no request sent; grant
   through `SetConsent`, then one short marked generation. *Quota: one short
   generation on the owner's ChatGPT plan.*
3. **The checks after the sign-ins** live in a function in
   `llmprovider/live_checks_test.go`, as AGENTS.md "Live tests" asks of every
   login test, and an offline twin in `llmprovider/live_twins_test.go` runs
   them against a fake; each check's planted break runs on the twin
   (0046-MADR D3).

`scripts/check_tables.py` passes: `LLMPROVIDER_LIVE_0075` is this record's
variable, which that script accepts because this record names it;
`LLMPROVIDER_LIVE_CHATGPT_PLAN` is AGENTS.md's.

*(2026-11-11: "U3b" was not a report id, and is renamed; step 3, the
checks' function and twin, added.)*

### Phase 7: the release

With 0074-PLAN Phase 8, in one minor release, `v1.31.0` if 0076 shipped as
`v1.30.0`. The release notes name the downgrade cost (MADR Consequences)
and the wizard's changed notice.

## Verification

Each command's exit status is captured before any filter.

* `go test -race -count=1 ./llmprovider/... ./wizard/... ./internal/signinflow/...`
* `go test -race -count=3 ./llmprovider/auth/`
* `python3 -B tools/redgreen.py REFRESH.json --expect pass` (0074-PLAN's 66)
* `make api-check` and `python3 -B tools/apidiff.py` against the previous
  tag: exactly this PLAN's and 0074-PLAN's tables
* `make contract-check` (the target 0073-PLAN adds; it is not in the
  `Makefile` at `773b84c`)
* `make dep-check tables-check coverage-check generate-check docs-check
  changelog-check parity-check records-check fixtures-check links-check
  disclosure-check markdownlint gate-selftest`, `make lint vuln`,
  `go mod tidy -diff`
* `TestWireGoldens` three times (R45)
* `python3 -B tools/gate.py`
* `python3 -B tools/claims.py docs/decisions/0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md`
* `go vet -tags live_gateways ./...`
* Every plant of Phases 1 to 4, quoted, the plant of
  `TestConsentSurvivesRefresh_0075` that restores the old refresh code
  included.

## Rollout and Rollback

* **Order.** First,
  [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md)
  raises the Go floor to 1.27.4, before any code of this series *(added
  2026-11-11, when the owner approved 0077)*. This PLAN's release comes
  after 0076's minor (`v1.30.0`), which ships that floor, with 0074, after
  `authcontract/v0.1.0`, and only after Phase 6's client-reuse probe (the
  owner's SDK-2). gobble then adopts it (gobble-cli
  `docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`):
  it marks its background requests, supplies its wording, answers the
  consent through `gobble auth consent`, implements `ClientStore` over
  `<state>/clients.json`, persists a host id in `<state>/host-id`, and
  migrates a stored `chatgptPlanBackground: true` with `SetConsent`. Magic
  relays the `confirm` step after gobble's `v0.6.0` (magic-cli-remote
  `docs/decisions/0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`).
  *(2026-11-11: the owner ruled that gobble-cli `v0.6.0` is the release that
  must pin the same acp-go-sdk2 tag as magic-cli-remote; both move from
  `v1.35.0` to `v1.36.0` together before it, each by a record of its own.)*
  *(2026-11-12: the gobble release after which Magic relays the `confirm`
  step, named `v0.6.0` above, is now `v0.7.0`. The owner decided that
  gobble-cli releases `v0.6.0` now, from its main, without the sign-in
  redesign, and that the redesign is `v0.7.0` (gobble-cli
  `docs/decisions/0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`,
  D1 and D2). The ruling of 2026-11-11 stays true of `v0.6.0`, which is cut
  on acp-go-sdk2 `v1.36.0` as magic-cli-remote is (0055-MADR F8); `v0.7.0`
  must meet the same lockstep when it is cut (its D2).)*
* **Rollout.** A plan session from an earlier release has no answer: a
  marked request on it is refused until the user answers, at the next plan
  sign-in or through the app's toggle. Unmarked requests are unchanged.
* **Downgrade.** An older SDK that refreshes and re-saves a session drops its
  consents (MADR Consequences); the next sign-in asks again. It ignores the
  `<provider>.client` file.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** the names stay (R48). If client reuse proves
  wrong after release, the wizard and `x/signin` stop passing `ClientID`,
  which needs no API change.

## Code claims

The seven tests of "Rules for every phase", where each is declared, checked by `tools/claims.py` (the lines that
script the notice are the MADR's C31 to C37).

<!-- claims: self=773b84c -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The first plan sign-in test | `wizard/plan_0043_test.go:93` | `func TestPlan_FirstSignIn_0043(t *testing.T) {` |
| P2 | The notice declined | `wizard/plan_0043_test.go:148` | `func TestPlan_NoticeDeclined_0043(t *testing.T) {` |
| P3 | The re-sign-in | `wizard/plan_0043_test.go:165` | `func TestPlan_ReSignIn_0043(t *testing.T) {` |
| P4 | A Codex session replaced | `wizard/plan_0043_test.go:196` | `func TestPlan_ReplacesCodexSession_0043(t *testing.T) {` |
| P5 | `ResolveCredential`'s plan sign-in | `wizard/resolve_0056_test.go:71` | `func TestResolveCredential_ChatGPTPlan_0056(t *testing.T) {` |
| P6 | The plan under `NoPastePrompt` | `wizard/paste_0057_test.go:55` | `func TestNoPastePrompt_PlanSignIn_0057(t *testing.T) {` |
| P7 | The plan's bare error | `wizard/signin_errors_0049_test.go:40` | `func TestConfigureLLM_PlanLoginErrorBare_0049(t *testing.T) {` |
| P8 | The seam all seven use | `wizard/plan_0043_test.go:43` | `func stubPlanLogin(t *testing.T, login func(context.Context, auth.ChatGPTPlanOptions) (*auth.OAuthSession, error)) *int {` |

## Execution record

The owner committed this pair and its index rows as records only
(`771b422`), and on 2026-11-12, in the coordinator session, accepted the
MADR and approved this PLAN.

### Phase 0 (2026-11-12)

* Step 1: done by the owner's commit `771b422`.
* Step 2: `python3 -B tools/records.py status RECORD STATUS --date
  2026-11-12` set the MADR `accepted` and this PLAN `in-progress`, each with
  its row in `docs/README.md`; each run printed "records: written;
  records-check: 0 problem(s)". The MADR's Decision Outcome records the
  acceptance and its date.
* Not done: step 3, the supersession notes, and step 4's gate for them.
  They remain Phase 0's, before Phase 1.
* **No code starts until `ci.yml` passes on `main`.** This PLAN follows
  [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)'s
  Phases 2, 4 and 5, and that PLAN's Phase 1 waits for a passing `ci.yml`
  run on `main` after
  [0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md](0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md).

### Phase 0, completed (2026-11-13)

* Step 3: done. `0043-MADR-openai-chatgpt-plan-sign-in.md` gains A5, a dated
  amendment appended after A4, quoting D8, D7 and D3 as they stood and the
  struct of D4, and saying how 0075-MADR amends or refines each (D3 kept and
  refined). No existing line of that record changed, so the line numbers the
  MADR cites (C2, C13, C23) hold.
  `0017-REPORT-openai-sign-in-for-agents.md` gains a "Follow-up (2026-11-13)"
  section appended after its earlier follow-up, noting that the open item "The
  terms in a consumer" is answered in part by 0075-MADR. No existing line of
  the report changed, so C1, C4, C8, C14 and C24 hold.
* Step 4: these ran after the last edit to the two amended records, each
  with its exit code captured before any filter:
  * `run-heavy -- python3 -B tools/gate.py --markdown`: exit 0, started
    2026-11-13T19:33:00-06:00 (it waited for the load to fall below 10),
    finished 2026-11-13T19:47:12-06:00, `gate: 11 of 11 passed`; each of its
    checks (parity, records, tables, changelog, markdownlint, links,
    disclosure, gate-selftest, deny-scan, recordlint, claims) reported rc=0.
  * `tools/claims.py` on the MADR: exit 0, `51 of 51 pass`; on this PLAN:
    exit 0, `8 of 8 pass`.
  * `make records-check links-check disclosure-check`: exit 0,
    `links-check: 0 problem(s), 1299 relative link(s) in 180 file(s)` and
    `disclosure-check: 0 hit(s) in 1622 file(s)`.
  * `tools/denyscan.py --changed`: exit 0, `deny-scan: 0 hit(s) in 3 file(s)`.
  * `tools/recordlint.py --changed`: exit 0,
    `recordlint: 3 record(s); no rule rose`.
* Provenance: mainstream model, medium effort; outcome, Phase 0 complete,
  docs only; no escalation; the coordinator's brief had missed the open steps.

### Phase 1 (2026-11-13)

* **Claims re-verified** at HEAD `cacc045` with `python3 -B tools/claims.py
  docs/decisions/0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md
  --rev self=HEAD`: exit 1, 25 rows failing at their line, 24 of them found
  elsewhere in the file with the text unchanged, and C9 not found. Those Phase 1 reads, by the new line:
  C15 (`tokenstore.go:240`, the record's `HostID`), C16 (`oauth_session.go:969`,
  `loadRotated`'s guard), C17 (`:955`, the refreshed session's save, now
  `reloadOrRefresh`), C18 (`:337`, `persistRotation`'s save), C27
  (`tokenstore.go:202`), C30 (`oauth_session.go:324`), C40 (`:970`), C41 and
  C42 (`tokenstore_file.go:295`), C44 (`oauth_session.go:680`) and C45
  (`tokenstore.go:50`). The rest (C5, C6, C7, C10, C11, C12, C19, C20, C21,
  C22, C25, C26, C28, C43) only moved and belong to later phases. **C9 is
  asserted text that changed, and is not Phase 1's:** `wizard/auth.go:291`
  no longer holds `session, loginErr := loginChatGPTPlan(ctx,
  auth.ChatGPTPlanOptions{AgentName: o.AgentName, Previous: previous,`; the
  wizard passes `Plan: auth.ChatGPTPlanOptions{...}` to the engine's spec at
  `wizard/auth.go:298`. It bears on Phases 2 and 4, whose steps name the
  wizard's call; the MADR's text of D4 and D6 holds. The MADR is not
  amended here. `claims.py` at its own `self=773b84c` still passes, 51 of 51
  and 8 of 8.
* **Step 1:** `OAuthSession.Consents` and `(*OAuthSession).Consent`, in
  `llmprovider/auth/tokenstore.go` and the new `consent.go`; `fileRecord`
  gains `consents`, `omitempty`; `String`, `MarshalJSON` and `LogValue` show
  it only when it is set, so every other session's forms are as they were;
  the five copies carry it (`persistable`, `refreshState`, the refreshed
  session, `adopt`, `FileTokenStore`'s `Load` and `Save`), and
  `TestSessionFieldsCarried_0074`, whose guard reads every exported stored
  field by reflection, passes with no change to it.
* **Step 2:** `auth.SetConsent`, as MADR D3. Two refusals beyond the MADR's
  text, both `ErrInvalidRequest`, and both needed for the entries' format to
  hold: an id holding white space or `=`, and a nil store. Nothing else
  differs.
* **Step 3:** the carry-over, no wider than MADR D3. `loadRotated` returns,
  with the rotated session it returned before, the consents of a record it
  passes over because it holds the session's own token or the token an
  unsaved rotation spent (C16, C40); `reloadOrRefresh` sets them on the state
  the refreshed session is built from, so the session it saves, and
  `adopt` puts on the live session, carries them. `persistRotation` sets the
  record's consents on the session it re-saves, in the two cases it saves
  (the record holds `stored`, or the session's own token), and `complete`
  puts them on the live session once the save succeeds. A record holding
  another token is adopted whole, consents included, as before. `loadRotated`
  is unexported and had no other caller. No refresh test changed.
* **Tests,** `llmprovider/auth/consent_0075_test.go`, the seven of the
  table and no other: `TestConsents_ParseAndFormat_0075`,
  `TestConsents_StoreRoundTrip_0075`, `TestSetConsent_0075`,
  `TestConsentSurvivesRefresh_0075`, `TestConsentSurvivesResave_0075`,
  `TestConsentSurvivesUnsavedRotationRefresh_0075` and
  `TestConsentAdoptedFromSibling_0075`. `go test -count=1 -run 0075
  ./llmprovider/auth/`: exit 0.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, one command per
  spec, `_0075$` tests only; each failure line quoted, `all 9 plant(s)
  caught`, exit 0:
  * `Declined` read as granted: `Consent("c.d") = true, true; want false,
    true` (`TestConsents_ParseAndFormat_0075`) and `the adopted session
    reports true, true; want the sibling's declined`;
  * the JSON name spelled `consent`: `TestConsents_StoreRoundTrip_0075`,
    `the record = {` (the member is missing);
  * the lock taken after `Load`: `the calls were load,lock,save,unlock, want
    lock,load,save,unlock: the lock is held throughout`;
  * a missing session saved as a new one: `no stored session: err = <nil>,
    want ErrInvalidRequest`;
  * the carry-over removed from `reloadOrRefresh`: `TestConsentSurvivesRefresh_0075:
    after the refresh: the store holds false, false; want the answer kept`,
    and the same for `TestConsentSurvivesUnsavedRotationRefresh_0075`;
  * the carry-over removed from `persistRotation`:
    `TestConsentSurvivesResave_0075: after the re-save: the store holds false,
    false; want the answer kept`;
  * the live session not told after a re-save: `after the re-save: the
    session reports false, false; want the answer`;
  * the carry-over limited to a record holding the session's own token:
    `TestConsentSurvivesUnsavedRotationRefresh_0075: after the refresh: the
    store holds false, false; want the answer kept`;
  * `adopt` dropping `Consents`: `TestConsentAdoptedFromSibling_0075: the
    adopted session reports false, false; want the sibling's declined`.
* **The plant that restores the old refresh code,** the carry-over removed
  from both functions, run as a second spec whose command first edits
  `persistRotation` in the clone and then runs `TestConsentSurvivesRefresh_0075`:
  caught, exit 1, `after the refresh: the store holds false, false; want the
  answer kept`, and `the session reports false, false`. That test's path never
  reaches `persistRotation`, so this plant and the single one above fail it
  the same way.
* **The 66 refresh tests,** `python3 -B tools/redgreen.py REFRESH.json
  --expect pass` (the spec of 0074-PLAN's list, in the scratch directory):
  65 of 66, exit 1, the one `missing`, `TestOAuthSessionWait_WindowsRemoval_0071`,
  which skips off Windows, as in 0074-PLAN's phases. `git status` shows none
  of the 15 files changed: this phase touches `oauth_session.go`,
  `tokenstore.go` and `tokenstore_file.go` and adds two files.
* **Checks, each exit status captured before any filter:**
  * `make pre-add-check FILES` on the five Go files: exit 0, `go-precheck: 5
    file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)`. No
    `nolint`.
  * `make api-check`: exit 0, `against v1.30.0, 0 incompatible change(s)
    outside llmprovider/x/`. `python3 -B tools/apidiff.py v1.30.0`: exit 0;
    this phase's three additions are `auth.OAuthSession.Consents`,
    `auth.(*OAuthSession).Consent` and `auth.SetConsent`, each a row of the
    R55 table; the rest are 0074's and 0079's.
  * `make tables-check`: exit 0, `0 problem(s)`; `make docs-check`: exit 0,
    `sdkaudit: 0 violation(s) of R50,R51,R52`; `make coverage-check`: exit 0,
    `43 packages, 0 problem(s)`.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after the
    last code change: exit 0, `gate: 30 of 30 passed`, started
    2026-11-13T19:56:07-06:00 (it waited for the load to fall below 10) and
    finished 20:26:07-06:00.
* **Not done here, by the PLAN:** the `-race` run of Phase 3, `TestWireGoldens`
  and the docs of Phase 5; `go test -race -count=3 ./llmprovider/auth/` is
  the gate's own coverage of this package.
* Provenance: mainstream model, medium effort; Phase 1 implemented, nine
  plants caught, gate 30 of 30; no escalation.

### Phase 2 (2026-11-13)

* **The owner's ruling and the dry run.** The first run of this phase stopped
  before any code, because its own changes fail six existing tests besides the
  seven. The owner ruled "Bounded amendment" (2026-11-13); the amendment is
  in "Rules for every phase", above, with each test, its file and line, and
  the kind of change. The dry run was Phase 2 implemented on a scratch clone
  of `f64134d` and `go test ./llmprovider/... ./internal/... ./wizard/...`
  there: 13 existing tests failed, the seven of "Rules for every phase" and
  the six of the amendment, and no other (one `TestStart_Cancel_0074` flake
  aside, below). No test needed a change of another kind.
* **C9.** A dated note under the evidence table of the MADR records that
  0074-PLAN Phase 5 moved the wizard's plan sign-in into the engine spec,
  `Plan: auth.ChatGPTPlanOptions{...}` at `wizard/auth.go:298`; C9's row stays
  pinned. Every step of Phases 2 and 4 that relied on C9's call site fits the
  new shape: Phase 2 moves the wizard's own `Confirm` into the engine's
  consent step, with the options still built in the spec; Phase 4 step 4 adds
  `ClientID` and `HostID` to the same `Plan:` options.
* **Moved claims,** checked with `claims.py ... --rev self=HEAD` at `f64134d`,
  each found with its text unchanged: C5 (`wizard/auth.go:278`, now `:282`)
  and C21 (`:284`, now `:288`), whose code this phase removes, since the
  notice is now the engine's consent step; C22 (`openai.go:162`, now `:163`),
  the usage page, which the descriptor's `ConsentURL` now names. C31 to C39,
  the seven tests' lines, hold where they were.
* **Step 1:** `AuthMethod.Consent`, `ConsentText` and `ConsentURL`;
  `openai.ConsentBackgroundUse` in the new `openai/consent.go`; the plan
  descriptor's three values, the text word for word as MADR D1. x/signin
  holds the same three facts for the plan (it may not import the providers,
  R2), and `TestMethods_StartFactsAreTheDescriptors_0074` holds them equal.
* **Step 2:** `signin.Methods` gives `Consents` (required, the text, the usage
  link), the usage link in `Links`, and the `consent` feature beside `paste`.
* **Step 3:** in `internal/signinflow`, a plan sign-in asks each consent of its
  method as a `confirm` step before the login starts (id, the app's text or
  the method's, the usage link, default false), unless the session being
  signed in to again holds an answer, which is carried; an answer with no
  `Confirm` is a no. The answers are put on the session the login returns,
  through `auth.SetConsent` on a one-session store of the engine's own, so
  that the entries' format stays auth's; a failed login records none. The
  wizard's `Confirm` and its decline moved here.
* **Step 4:** `signin.Options.ConsentText` and `wizard.Options.ConsentText`
  (maps from a consent id to the app's text), as `Spec.ConsentText`.
* **Step 5:** the wizard's renderer shows a consent `confirm` as a `Select` of
  "Allow background use", "Do not allow background use" (the default) and
  "Cancel the sign-in", its title the step's text and the usage page's URL;
  the last returns today's error, "wizard: ChatGPT plan sign-in declined". A
  `Driver` is given the plain step, and its `Answer.Confirm` answers it;
  `StepDriver`'s doc says so. The plan method's facts reach the wizard from
  its descriptor, else the built-in one's.
* **Existing tests changed, and only these,** by the kinds of the rule above
  (`git diff --numstat`): `descriptors_test.go` 3/0, `methods_0074_test.go`
  7/0, `start_browser_0074_test.go` 7/4, `identity_0076_test.go` 1/1,
  `paste_0057_test.go` 1/1, `plan_0043_test.go` 11/12, `resolve_0056_test.go`
  4/4, `signin_errors_0049_test.go` 1/2. In the seven wizard tests each
  scripted `Confirm` became a `Select` of the notice (allow; "Cancel the sign-in"
  for `TestPlan_NoticeDeclined_0043`), the two that read the notice read it
  from `seenSelect`, and `TestPlan_ReSignIn_0043`'s stored session gained
  `Consents: "chatgpt_plan.background_use=granted"`. Every other assertion is
  as it was. The new tests: `consent_0075_test.go` in `internal/signinflow`,
  `llmprovider/providers` and `wizard`, and `signin_0075_test.go` in
  `llmprovider/x/signin`, the eight of the table.
* **Plants,** `tools/plant.py` on a scratch clone, `_0075$` tests; each plant's
  first failure lines are those below, `all 12 plant(s) caught` and exit 0 in
  one run, and the two that first did not count (below) run again, caught:
  * the default set to yes: `TestPlanConfirmStep_0075: confirm step = {...
    Default ...}`;
  * the consent not asked before `open_url`: `first step = open_url #1; want
    confirm #1`;
  * no ends the sign-in: `step after the answer = failed #2; want open_url #2`;
  * the answer not carried: `Wait = {...}, <nil>; want Consents
    "chatgpt_plan.background_use=granted" carried`;
  * the answer put on a failed login's session: **the first form,** which
    also masked the error, was caught for the wrong reason, `Wait = nil; want
    the login's error`; the plant was made `_ = putConsents` on a failed
    login, and fails as meant: `a failed sign-in's session holds Consents
    "chatgpt_plan.background_use=granted"; want none recorded`;
  * any stored plan session skipping the question: **the first form did not
    build** (`answered` unused), which is not a catch; as `true || answered` it
    fails `first step = open_url #1; want confirm #1` and
    `TestWizardReSignInWithoutAnswerAsks_0075: 1 logins, selects [...]; want
    one login after the consent Select`;
  * the method's text before the app's: `the consent's title = "You're using
    your ChatGPT plan: ...` and the app's text not shown;
  * the consent not required: `the plan's consents = [{... Required:false ...}]`;
  * the text's "background" dropped: `TestDescriptors_PlanConsent_0075: the
    plan's consent = ...`;
  * the default set to the first choice: `the consent's default = 0; want the
    second choice`;
  * the usage URL left out of the title: `the consent's title = "You're using
    ... want the step's text and the usage page`;
  * "Cancel" read as a choice: `err <nil>, 1 logins, 1 saves; want declined,
    none, none`.
  The table's plant "`confirm` after `open_url`" is the second of these,
  since the plant keeps the consent from being asked at all.
* **Checks, each exit status captured before any filter:**
  * `make pre-add-check FILES` on the 21 Go files: exit 0, `go-precheck: 21
    file(s) clean`; again on `signin_0075_test.go` after its last change:
    exit 0. No `nolint`.
  * `make api-check`: exit 0, `0 incompatible change(s) outside
    llmprovider/x/`. `python3 -B tools/apidiff.py v1.30.0`: exit 0; this
    phase's names are `AuthMethod.Consent`, `.ConsentText`, `.ConsentURL`,
    `openai.ConsentBackgroundUse` and `wizard.Options.ConsentText`, each a row
    of the R55 table (`signin.Options.ConsentText` is under `x/`).
  * `make tables-check`, `make docs-check` (`0 violation(s) of R50,R51,R52`)
    and `make coverage-check` (`43 packages, 0 problem(s)`): exit 0.
  * **The gate,** `python3 -B tools/gate.py --timeout 5400` through
    `run-heavy`. **First run,** 20:41:55 to 20:57:22, exit 1, `29 of 30
    passed`: `test-shuffle` failed on two tests, `TestStart_Cancel_0074`
    (`start_device_0074_test.go:164: 4 goroutines after the sign-in, 5
    before`) and `TestDriver_OwnErrorGetsNoTerminal_0074/show_code`
    (`driver_terminal_0074_test.go:151: the Driver was handed [... failed ...]
    after its own error`). The second is a flake of the tree before this
    phase: at `f64134d`, six processes of `go test -race -count=150 -cpu 1,2`
    on that test failed in two of them, and the tree failed in two of them.
    The first, a count of the process's goroutines, did not fail in 85
    shuffled runs of `./llmprovider/x/signin/` at `f64134d` (25, then 60) and
    failed in 1 of 25 on the tree, ~~so this phase's tests make it likelier: the sign-in
    tests left goroutines (a login's paste read) ending after `Wait`. The new
    `TestPlanConfirmStep_0075` now waits for them in `settleGoroutines`,
    which asserts nothing less than before.~~ *(Struck 2026-11-13: see the
    correction below.)* **Second run,** 21:07:58 to
    21:28:11, after that last code change, exit 0, `gate: 30 of 30 passed`.
    `TestStart_Cancel_0074` still failed once in 60 shuffled runs of the
    package on the tree after the change, against none at `f64134d`: its
    count of every goroutine in the process is the fragile part, and any
    sign-in test ending beside it can trip it. It is reported to the
    coordinator, not changed here.
* ~~Provenance: mainstream model, medium effort; Phase 2 implemented under the
  owner's "Bounded amendment", twelve plants caught, gate 30 of 30 on its second
  run, with `TestStart_Cancel_0074`'s flake reported; no other escalation.~~
  *(Superseded 2026-11-13 by the provenance line below.)*
* **Correction (2026-11-13), the owner's ruling "Remove it, correct the
  entry".** The claim above that this phase made `TestStart_Cancel_0074`
  likelier is wrong. The test counts every goroutine in the process, a
  test-isolation defect that predates Phase 2 and is not a leak of the
  product; it fails at the same rate with and without Phase 2 (444 and 386 of
  2,000 runs). `TestDriver_OwnErrorGetsNoTerminal_0074/show_code`, which I
  called a flake, was a real ordering defect in `wizard/driver.go` from
  0074-PLAN Phase 5. Both are fixed in `eecf17e`, recorded as a deviation of
  0074-PLAN with the evidence, with a bubble-only goroutine count and
  `TestDriver_ShowErrorRecordedFirst_0074`. `settleGoroutines`, and its one
  call in `TestPlanConfirmStep_0075`, are removed.
* **Re-verification on `eecf17e` plus this phase,** after `settleGoroutines`
  was removed, with each exit status captured before any filter:
  * the plants, `tools/plant.py` on a scratch clone: the ten of the first spec
    `all 10 plant(s) caught` and the two that were redone `all 2 plant(s)
    caught`, exit 0 each, with the failure lines above unchanged;
  * `make pre-add-check FILES` on the 21 Go files: exit 0, `go-precheck: 21
    file(s) clean`; `make api-check`: exit 0; `tools/apidiff.py v1.30.0`: exit
    0, this phase's names as above;
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, started
    2026-11-13T22:47:39-06:00 (at load 9.4) and finished 22:59:48-06:00: exit 0,
    `gate: 30 of 30 passed`, with no test of this phase failing;
  * `go test -race -shuffle=on -count=3 ./llmprovider/x/signin/
    ./internal/signinflow/ ./wizard/` through `run-heavy`, 22:59:48 to
    22:59:57-06:00: exit 0, `ok` for each of the three packages.
* Provenance: mainstream model, medium effort; Phase 2 implemented under the
  owner's "Bounded amendment"; one escalation, `TestStart_Cancel_0074`'s and
  `TestDriver_OwnErrorGetsNoTerminal_0074`'s failures in the gate's
  `test-shuffle`, whose causes lay before this phase and were fixed in
  `eecf17e`, and the owner's ruling "Remove it, correct the entry"; twelve
  plants caught, gate 30 of 30, race run green.

### Phase 3 (2026-11-13)

* **Claims re-verified** with `claims.py ... --rev self=HEAD` at `8ea6485`.
  The ones Phase 3 reads only moved, each found with its text unchanged: C19
  (`openai.go:181`, now `:182`), C20 (`:351`, now `:357`), C22 (`:162`, now
  `:163`), C26 (`:303`, now `:309`), C43 (`:177`, now `:178`), C44
  (`oauth_session.go:588`, now `:691`) and C45 (`tokenstore.go:45`, now
  `:57`). Two that Phase 3 does not read changed text, both by Phase 1's
  restructure of `loadRotated` (the guard C16 quotes, and C40's), and are
  not repaired here: they stay pinned to `773b84c`, where they pass. The MADR
  is not amended. No assumption of Phase 3 about the engine, the wizard or
  the refresh code failed, and no existing test needed a change: the phase
  adds tests and changes none.
* **Step 1:** `llmprovider/background.go`: `WithBackground`, `IsBackground`
  and an unexported context key type.
* **Step 2:** `(*OAuthSession).StoredConsent(ctx, id) (granted, answered
  bool, err error)` in `llmprovider/auth/consent.go`. It takes `Store` under
  the session's lock and loads from it; with no `Store`, the session's own
  answer; a store with no session for the owner gives unanswered; an error
  from `Load` is returned as it is. Its doc says so in R51's and R52's words.
* **Step 3:** `providers/openai/openai.go`: `check` takes the context and, for
  a marked request on a plan session, reads the answer through
  `StoredConsent`; not granted is an error matching `ErrNotPermitted` that
  names `openai.ConsentBackgroundUse`, and so is a read that fails, which also
  wraps the read's error. `Generate` and `Stream` both call it before the
  network. Keys, Codex sessions and unmarked requests are untouched.
* **Tests,** the six of the table, none of them changing an existing test:
  `TestWithBackground_0075` (`llmprovider/background_0075_test.go`),
  `TestStoredConsent_0075` (appended to the auth file of Phase 1) and
  `TestOpenAIBackground_Refused_0075`, `_Allowed_0075`, `_ReadsStore_0075`
  and `_ReadFails_0075` (`providers/openai/background_0075_test.go`).
  `_ReadsStore_0075` also holds a marked request beside twenty refreshes of
  the same session, which rewrite its `Store`.
* **Plants,** `tools/plant.py` on a scratch clone, `_0075$` tests,
  `all 10 plant(s) caught`, exit 0. The first line of each failure:
  * `IsBackground` true for every context: `an unmarked context reads as
    background`;
  * the mark set under a string key: `a marked context does not read as
    background`;
  * the mark read under a string key: `a value under the string "background"
    marks a context`;
  * the check skipped for every request: `unanswered: Generate err = <nil>;
    want ErrNotPermitted naming chatgpt_plan.background_use`;
  * `Stream` unchecked: `1 request(s) sent; want none before the refusal`;
  * an unmarked request refused: `Generate err = llmprovider: not permitted:
    ... which the user has not granted`, `0 request(s) sent; want 1`;
  * the answer read from the session only: `after the withdrawal: Generate
    err = <nil>; want ErrNotPermitted naming chatgpt_plan.background_use`;
  * a failed read falling back to the session's answer: `a failed read:
    Generate err = <nil>; want ErrNotPermitted ...`;
  * the stored record ignored: `StoredConsent = false, false, <nil>; want the
    store's granted`;
  * a `Load` error swallowed: `StoredConsent with a failing Load = false,
    false, <nil>; want the store's error and no answer`.
  The table's plant "the check placed after the request" is represented by
  the check skipped for every request, since a check that never refuses is
  the extreme of one that runs too late; the mark's "plain string" key is the
  two string-key plants.
* **Checks, each exit status captured before any filter:**
  * `make pre-add-check FILES` on the six Go files: exit 0, `go-precheck: 6
    file(s) clean`. **First run, exit 2:** `errcheck` on the blank in
    `IsBackground`'s type assertion, and `staticcheck` SA1029 twice on the
    test's context keys; fixed by `ok && marked` and by keys of an unrelated
    struct type and a function-returned plain key. No `nolint`.
  * `make api-check`: exit 0. `python3 -B tools/apidiff.py v1.30.0`: exit 0;
    this phase's names are `llmprovider.WithBackground`, `IsBackground` and
    `auth.(*OAuthSession).StoredConsent`, each a row of the R55 table.
  * `make tables-check`, `make docs-check` (`0 violation(s) of R50,R51,R52`)
    and `make coverage-check` (`43 packages, 0 problem(s)`): exit 0.
  * `python3 -B tools/redgreen.py REFRESH.json --expect pass`: 65 of 66,
    exit 1, the one `missing` `TestOAuthSessionWait_WindowsRemoval_0071`,
    which skips off Windows, as in earlier phases; none of the 15 files
    changed.
  * the race run of the PLAN, `go test -race -count=1 -run
    'TestStoredConsent_0075|TestOpenAIBackground_ReadsStore_0075'
    ./llmprovider/auth/ ./llmprovider/providers/openai/`: exit 0, no race.
    `go test -count=3 -run TestWireGoldens ./llmprovider/...`: exit 0, the
    goldens unchanged (R45).
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after the
    last code change: exit 0, `gate: 30 of 30 passed`, started
    2026-11-13T23:10:51-06:00 (at load 9.8) and finished 23:29:25-06:00.
  * `go test -race -shuffle=on -count=3 ./llmprovider/ ./llmprovider/auth/
    ./llmprovider/providers/openai/` on a Linux host, a WSL distro, through
    `run-on-host --need linux --race` (which queues under `run-heavy`),
    23:29:40 to 23:30:18-06:00: exit 0, `ok` for each package.
* Provenance: mainstream model, medium effort; Phase 3 implemented, ten
  plants caught, gate 30 of 30, race runs green on macOS and Linux; no
  escalation.

### Phase 4 (2026-11-13)

* **Claims re-verified** with `claims.py ... --rev self=HEAD` at `39f1b81`.
  The ones Phase 4 reads only moved, each found with its text unchanged: C6
  (`oauth_plan.go:224`, now `:198`), C7 (`:237`, now `:211`), C10 (`:259`, now
  `:233`), C11 (`:197`, now `:169`), C25 (`:261`, now `:235`), C28 (`:60`,
  now `:58`), C12 (`wizard/logout.go:53`, now `:54`), C30
  (`oauth_session.go:318`, now `:325`) and C42 (`tokenstore_file.go:289`, now
  `:297`). C13, C24 and C29 hold where they were. **C9** is the one whose
  text is gone, and its note stands: the wizard builds the plan's options in
  the engine spec (`wizard/auth.go:291` at `39f1b81`), which is where this
  phase adds `HostID` and the client store; every step of this phase fits
  that shape, and no assumption about the engine or the wizard failed. No
  existing test changed: `plan_login_0043_test.go` and every other existing
  test pass as they were.
* **The names,** exactly as the R55 table has them, none forced to another:
  `auth.ClientRecord{ClientID, HostID}`, `auth.ClientStore` with `LoadClient`
  and `SaveClient`, `(*FileTokenStore).LoadClient` and `.SaveClient`,
  `ChatGPTPlanOptions.ClientID`, `wizard.Options.HostID`,
  `signin.Options.HostID` and `signin.Options.ClientStore`.
* **Step 1:** `llmprovider/auth/clientstore.go`, and `FileTokenStore`'s two
  methods on `<provider>.client`. Save's durable write became `writeDurable`,
  which `SaveClient` shares, so the temp file is restricted through
  `tokenStoreRestrict` before its first byte and then fsynced and renamed, as
  a session's is; the provider id is checked by `validateProviderID`; a zero
  record removes the file, and removing an absent one succeeds; `Delete` of
  the session leaves it. The 66 refresh tests, which run through `Save`, pass
  unchanged.
* **Step 2:** `ChatGPTPlanOptions.ClientID`. With no `Previous`, the authorize
  request names it and sends no `agent_name_hint` and no `id_token_hint`, and
  a callback naming another client is refused; with a `Previous`, its client
  is used and `ClientID` ignored. `planIssuedClient` now takes the known
  client (`knownPlanClient`).
* **Step 3:** `wizard.Options.HostID`, `signin.Options.HostID` and
  `signin.Options.ClientStore`; the order is `Options.HostID`, else
  `Previous`'s, else the record's, else a new one, which auth mints and the
  record then keeps.
* **Step 4:** the engine does the work for both, through a new
  `Spec.ClientStore`: before a plan sign-in with no `Previous` it loads the
  record (a load that fails ends the sign-in before the login) and passes the
  record's client and host id, the option's host id first; as the login
  returns a session it saves the client and host id the session holds, before
  anything else the caller does. The wizard passes its `Options.TokenStore`
  when it asserts to `auth.ClientStore`. A save that fails does not fail the
  sign-in: the engine's `Result.ClientSaveErr` is shown by the wizard as a
  warning notice, and `x/signin` documents that it is not reported. A
  sign-in with `ClientID` and no `Previous` still has `First` true, so 0074-MADR
  D1's `FirstPaste` `url_only` holds.
* **Tests,** besides the table's five, two the record's steps need:
  `TestFileTokenStore_Client_0075` and `TestPlanClientID_0075`
  (`llmprovider/auth/clientstore_0075_test.go`); `TestHostIDOrder_0075` and
  `TestPlanClientRecordErrors_0075` (`internal/signinflow/client_0075_test.go`;
  the order test is the engine's, since the engine does the loading);
  `TestStart_PlanClientStore_0075` (`llmprovider/x/signin`, that the two
  options reach the engine); `TestWizardPlanReusesClient_0075` and
  `TestWizardPlanSwitchKeepsClient_0075` (`wizard/client_0075_test.go`, over a
  `FileTokenStore`; "after Logout" deletes the session, which is Logout's last
  step, since revoking a plan session needs an issuer).
* **Plants,** `tools/plant.py` on a scratch clone, `_0075$` tests, exit 0
  each. The first failure line of each:
  * `Delete` removing the client file: `LoadClient after Delete of the session
    = {ClientID: HostID:}, <nil>; want the record kept`;
  * the restriction skipped: `events = ["first byte temp file"], want
    ["private temp file" "first byte temp file"]`;
  * `agent_name_hint` still sent: `the request with a ClientID =
    map[agent_name_hint:[My App] client_id:[client-0075] ...`;
  * a callback naming another client accepted: `a callback naming another
    client was accepted`;
  * the record's host id preferred to the option's: `login given host
    "urn:uuid:rec", client "client-rec", err <nil>; want the option's host id
    and the record's client`;
  * the record preferred to `Previous`'s: `the record was loaded 1 time(s)
    with a Previous; want none`;
  * the record not saved when the sign-in returns: `the record = {ClientID:
    HostID:} after 0 save(s); want the issued client and the minted host id
    saved once`;
  * the stored client not passed: `login given host "urn:uuid:opt", client "";
    want the option's host id and the record's client`;
  * `x/signin` passing no client store: `the login was given client "", host
    "urn:uuid:opt"; want the record's client and the option's host id`;
  * the wizard passing no client store, **whose first form did not build**
    (`clients` unused), which is not a catch; as the type assertion removed it
    fails `saves in the order ["Save"]; want the client record first, as the
    sign-in returns, before the session` and `a plan sign-in after a Codex-id
    session was given client ""`;
  * the table's "client file removed on replacement", as `Save` removing it:
    `TestWizardPlanSwitchKeepsClient_0075: ... was given client ""; want the
    stored client and no Previous`.
  The table's plant for `TestWizardPlanReusesClient_0075`, "the record saved
  only after the listing", is represented by the two above that fail its
  order assertion and its stored-client assertion: the engine saves as the
  login returns, and no listing is between.
* **Checks, each exit status captured before any filter:**
  * `make pre-add-check FILES` on the 11 Go files: exit 0, `go-precheck: 11
    file(s) clean`. **First run, exit 2,** three findings: `errcheck` on the
    blank of the wizard's type assertion, and `staticcheck` S1016 twice in
    `clientstore.go`; fixed by an explicit `ok` and by struct conversions. No
    `nolint`.
  * `make api-check`: exit 0. `python3 -B tools/apidiff.py v1.30.0`: exit 0;
    this phase's names are those above (the two `x/signin` options are under
    `x/`), each a row of the R55 table.
  * `make tables-check`, `make docs-check` and `make coverage-check`: exit 0.
  * `python3 -B tools/redgreen.py REFRESH.json --expect pass`: 65 of 66,
    exit 1, the one `missing` `TestOAuthSessionWait_WindowsRemoval_0071`,
    which skips off Windows; none of the 15 files changed.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after the
    last code change: exit 0, `gate: 30 of 30 passed`, started
    2026-11-13T23:43:09-06:00 (at load 9.5) and finished 23:58:30-06:00.
  * `go test -race -shuffle=on -count=3 ./llmprovider/auth/
    ./internal/signinflow/ ./llmprovider/x/signin/ ./wizard/` on a Linux host
    (a WSL distro) through `run-on-host --need linux --race`, 23:58:43 to
    23:59:26-06:00 local: exit 0, `ok` for each package.
* Provenance: mainstream model, medium effort; Phase 4 implemented, eleven
  plants caught, gate 30 of 30, race run green on Linux; no escalation.

### Phase 5 (2026-11-14)

* **What the docs say, each claim read against the code of Phases 1 to 4.**
  * `llmprovider/auth/doc.go`: the consent paragraph now says what the SDK
    carries (the method's `Consent`, the question before the browser, the
    answer on the session and in `FileTokenStore`, `SetConsent`, the refresh
    that keeps it, `WithBackground` and the openai provider's refusal through
    `StoredConsent`), the host id and the client store, and what stays the
    consumer's: which requests are background and the wording, and the rest of
    OpenAI's terms (MADR D8). The doc comment is the only Go change.
  * `README.md`: the plan section gains a "Consent" and a "Host id and
    client" item and its "What stays yours" item for consent is rewritten with
    `WithBackground` and the wording options; the wizard's list item for the
    plan sign-in describes the one question with three choices.
  * `docs/architecture.md`: the plan's consent (the carry-over in
    `loadRotated`, `persistRotation` and `complete`, `StoredConsent`, the
    provider's `check`), the issued client (`ClientStore`, `writeDurable`, the
    host-id order, the probe that gates the reuse), and the engine's consent
    step and client store beside the wizard's rendering, plus the `auth` and
    table row's new names.
  * `CHANGELOG.md` `[Unreleased]`: **Added** the names of Phases 1 to 4, each
    as the R55 table lists it; **Changed** the wizard's plan notice, now one
    question with three choices, with the rollout and downgrade costs of the
    MADR's Consequences.
  * The deviations of Phases 0 and 2 are recorded in this PLAN and in the
    amended records, and are not described as product behaviour: Phase 0's
    supersession notes (0043-MADR A5, the 0017 report's follow-up), and Phase
    2's bounded amendment of the existing tests and its note on C9. The docs
    say that 0075-MADR amends 0043-MADR D7 and D8, as the doc comment's "as
    0075-MADR amends it".
* **Claims rows, the coordinator's decision:** none are added to any record.
  Phase 5 makes no new code assertion in a record, as 0074-PLAN's Phase 6
  decided for its docs. The two records' claims run, `51 of 51` and `8 of 8`.
* **Checks, each exit status captured before any filter:**
  * `make docs-check`: exit 0, `sdkaudit: 0 violation(s) of R50,R51,R52`;
    `make links-check`: exit 0, `0 problem(s), 1301 relative link(s)`;
    `make changelog-check`: exit 0, `44 release section(s), 0 problem(s)`;
    `make markdownlint`: exit 0, `0 issues in 0 files`; `make parity-check`:
    exit 0.
  * `make pre-add-check FILES="llmprovider/auth/doc.go"`: exit 0, `1 file(s)
    clean`; `make api-check` and `tools/apidiff.py v1.30.0`: exit 0, the
    names unchanged since Phase 4.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, since a Go
    doc comment changed, after the last edit to a file but this entry's
    last bullets: exit 0, `gate: 30 of 30 passed`, started
    2026-11-14T00:03:41-06:00 (at load 9.7) and finished 00:18:16-06:00, on
    `17cfbd0`. It ran after every edit but this entry's last two bullets,
    which only record its result. `python3 -B tools/gate.py --markdown`
    refuses a change set that holds a Go file (exit 1, "llmprovider/auth/doc.go
    is not Markdown; a change to it needs the full gate"), so the final text
    was checked by the record checks instead, each exit 0 after the last
    edit: `claims.py` on both records, `make records-check links-check
    disclosure-check changelog-check markdownlint docs-check`, `denyscan
    --changed` and `recordlint --changed`.
* Provenance: mainstream model, medium effort; Phase 5 written, the docs read
  against the code of Phases 1 to 4, no claims rows added, gate 30 of 30; no
  escalation.
