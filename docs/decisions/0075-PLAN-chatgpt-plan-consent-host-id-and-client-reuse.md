---
status: proposed
date: 2026-11-11
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

Not started. The MADR is `proposed`.
