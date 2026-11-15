---
status: complete
date: 2026-11-14
associated-madr: "0074-MADR-one-sign-in-handle-and-method-facts.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement Method Facts and One Sign-In Handle Under `llmprovider/x/signin`

Associated MADR: [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)

## Goal

After this PLAN:

* every built-in `AuthMethod` carries the facts of MADR D1, and
  `signin.Methods` returns them as the contract's `Method` (D2);
* `signin.Start` drives every device, browser, plan and vendor-import sign-in
  through one `*signin.SignIn` handle that emits the contract's steps and
  takes its answers (D3, D4);
* every session carries an explicit, persisted kind, and the refresh code is
  re-dispatched by it, its 66 tests unchanged (D5);
* the wizard renders the same steps, word for word as today, and hands them
  to a `Driver` when a caller has no terminal (D6);
* `signin.ErrorKindOf` maps every error to the contract's kinds (D8).

It ships in one SDK minor release with
[0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md).

The MADR is `proposed`. Nothing below starts before the owner accepts it and
approves this PLAN.

## Prerequisites

* [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md)'s
  Phase 1 has landed: the root's `go.mod` says `go 1.27.4`. It lands before
  any of 0076's code, so 0076's release, which the last bullet requires,
  carries it. *(Added 2026-11-11, when the owner approved 0077.)*
* `authcontract/v0.1.0` is tagged and resolves through the module proxy
  ([0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md)
  Phase 4). Phase 4 below requires it.
* [0076-PLAN-auth-error-kinds-and-doc-drift.md](0076-PLAN-auth-error-kinds-and-doc-drift.md)
  has shipped, in its own minor release, the owner having decided on
  2026-11-10 that it lands first (DF). This PLAN uses two of its results:
  `auth.Identity`, which `signin.Options` carries (its D3), and its D2, which
  makes the OpenAI device poll's own 15-minute deadline the shared expiry
  (MADR C51, C45), so that `ErrorKindOf` maps it to `expired` and not
  `other`. Phase 3 checks both at its base; either missing is a stop.

## Scope

### In scope

* **D1:** `llmprovider/descriptor.go`; `llmprovider/providers/openai/descriptor.go`,
  `grok/descriptor.go`, `kilo/descriptor.go`; `llmprovider/providers/descriptors_test.go`.
* **D2, D3, D8 (the engine):** a new `llmprovider/x/signin/` (`doc.go`,
  `methods.go`, `signin.go`, `credential.go`, `errorkind.go`, tests and an
  example); a new `internal/signinflow/` (the step engine both `x/signin` and
  `wizard` use).
* **D4, D5, D8, D10 (beneath, in `auth`):** `llmprovider/auth/oauth_loopback.go`,
  `oauth_plan.go`, `oauth_device.go`, `kilo_device.go`, `oauth_session.go`,
  `oauth_revoke.go`, `tokenstore.go`, `tokenstore_file.go`,
  `vendor_session.go`, a new `vendor_path.go`, and their tests;
  `llmprovider/live_oauth_login_test.go` (D7's move).
* **Phase 7 step 4, the Grok paste probe** *(added 2026-11-13, the owner's
  ruling "Probe Grok paste; keep 0247")*: a new
  `llmprovider/live_0074_grok_paste_test.go`, with its checks appended to
  `llmprovider/live_checks_test.go` and its twin to
  `llmprovider/live_twins_test.go`. Test files only. On a pass, Phase 8
  step 1 adds `llmprovider/providers/grok/descriptor.go`,
  `llmprovider/auth/oauth_loopback.go`, their tests, `CHANGELOG.md` and a
  note in `0008-MADR-repair-oauth-loopback-and-session-wiring.md`.
  *(2026-11-13, the owner's ruling 2: these test files land in Phase 3a,
  before Phase 4, and so, on a pass, does Phase 8 step 1.)*
* **D6:** `wizard/auth.go`, `wizard/device.go`, `wizard/import.go`,
  `wizard/configure.go` (`Options.Driver`), a new `wizard/driver.go`, and
  tests.
* **The second module in the gates:** `go.mod`, `go.sum`,
  `scripts/check_deps.py`, `scripts/check_tables.py`.
* **Docs:** `docs/guides/api-standards.md` (R2's sentence and rows),
  `docs/architecture.md`, `README.md`, `CHANGELOG.md`, `AGENTS.md` ("Live
  tests": `LLMPROVIDER_LIVE_DEVICE_LOGIN`'s scope), and the amendments the
  MADR lists.

### Exported names (R55)

gobble-cli's records cited below, `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
and `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`, are
proposed and untracked in gobble-cli on 2026-11-11; the owner's answers they
record (QE (a) among them, 2026-11-10) stand. A call site there is planned,
not built.

Outside `llmprovider/x/` (under the v1 promise from the release):

| Name | Call site that needs it |
| :--- | :--- |
| `AuthMethod.Kind`, `.Completion`, `.Paste`, `.FirstPaste`, `.Yields`, `.ClientIdentity`, `.Slot` (`string`), `.Refreshable`, `.Revocable` (`bool`) | `signin.Methods`; gobble's policy filter that replaces `HasLogin`, `HasKeyLogin` and `offers` (the report's G11, gobble-cli `docs/decisions/0031-MADR-sdk-current-releases-and-adoption.md` D8: "read `Descriptor.AuthMethods`") |
| `type auth.SessionKind`; `auth.SessionChatGPT`, `SessionChatGPTPlan`, `SessionGrok`, `SessionKiloDevice`; `OAuthSession.Kind` | the seven dispatch sites (MADR C19 to C25); a caller's own `TokenStore`, such as gobble's, which must persist it as `FileTokenStore` does (gobble-cli 0052-MADR D6) |
| `(*auth.OAuthSession).SessionKind` *(exported by the owner's ruling of 2026-11-11, MADR D5; listed until then as a proposed revision)* | gobble's resolver, which accepts a stored session of an offered kind, inferred when `Kind` is empty (gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D2), and `auth status --json`'s `sessionKind` (gobble-cli `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D8) |
| `auth.ErrSignInDenied`, `auth.ErrSignInExpired` | `signin.ErrorKindOf`'s `denied` and `expired`, which gobble's `provider.ErrorKind` returns for its `--json` and `auth_required` failures (gobble-cli 0051-MADR D6; Q15) |
| `OAuthFlowOptions.NoLoopback`, `ChatGPTPlanOptions.NoLoopback` *(restored 2026-11-11 by the owner's reversal, later the same day; MADR D4)* | `signin.Options.NoLoopback`, which `x/signin`'s `Start` passes to them, in another package; set by gobble's ACP `authenticate` for a sign-in started from a remote ACP client, whose person is not on the agent's host, so that only a paste completes it (gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3). *(That morning the row was struck, its names crossed out and its call site replaced by "Removed 2026-11-11 by the owner's ruling (MADR D4): neither is exported until a consumer's record sets one. The row gave `signin.Options.NoLoopback` as their call site, "which `x/signin`'s `Start` passes to them, in another package", and no consumer; before that it said "for gobble's remote sign-ins", which no gobble-cli record plans." The reversal restores it: gobble-cli's 0051-MADR D3 is the consumer's call site.)* |
| `(*auth.OAuthSession).UseStore` *(added 2026-11-14 by the owner's ruling "Locked setter `UseStore`", Phase 7 step 6's deviation)* | `wizard`'s save of a new sign-in (`resolvedCredential.save`, `wizard/auth.go`), which binds the session to its store while a listing may still be refreshing it; a consumer that does the same takes the same call |
| `auth.VendorCLIPath` | `wizard`'s vendor import (replacing `vendorAuthPath`); `x/signin`'s `import_vendor_cli` |
| `wizard.Options.Driver`; `type wizard.StepDriver` and its method `SignInStep` | gobble's `gobble auth login --json`, which runs `wizard.ResolveCredential` (0056-MADR D1) with no terminal and prints each step (gobble-cli `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3 and D5; the owner's QE (a), 2026-11-10). gobble's ACP `authenticate` does not use it: it runs `signin.Start` |

Under `llmprovider/x/signin` (outside `make api-check`, reported as
experimental):

| Name | Call site that needs it |
| :--- | :--- |
| `Methods` | gobble's policy filter, `provider.Offered` (gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D2), which feeds the ACP agent auth methods' `_meta.gobble.methods` and `--json`'s first line; Magic reads them through gobble |
| `Start`, `Options` and its fields, `type SignIn` and its `Method`, `Steps`, `Answer`, `Wait`, `Cancel` | gobble's ACP `authenticate`, through its `provider.StartSignIn` handle, relayed as request-scoped elicitation (Q7; gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3 and D4; the owner's QE (a), 2026-11-10). gobble's `--json` runs the wizard's `Driver` instead |
| `Options.NoLoopback` *(MADR D4; passed through to the two `auth` fields above)* | gobble's ACP `authenticate`, for a sign-in started from a remote ACP client (gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3); Phase 7's U2 and U3 probes (`TestLive_0074PasteBrowser`, `TestLive_0074PastePlan`). *(2026-11-11, by the owner's reversal later that day: the row was marked "experimental only, the owner's ruling of 2026-11-11; MADR D4", and its call site went on "No consumer record sets it: gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3 lists the `signin.Options` it sets without it. It is built only once MADR D4's open question is answered (Phase 3 step 2)".)* |
| `type Credential`; `APIKey`, `Session`, `AccessOnly`, `DeviceKey`, `ReadThrough` | gobble's `SignInRun.Wait`, which saves a session through its store and returns a key for its caller to store (gobble-cli `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3) |
| `ErrorKindOf` | gobble's `--json` in place of `FailureKind` (Q15) |

Deprecated (doc only, R42): `AuthMethod.Interactive`, `AuthMethod.HeadlessOK`,
`auth.LoginDeviceOAuth`, `OAuthFlowOptions.NotifyDevice`.

No other name is exported. `internal/signinflow` is internal.

### Out of scope

* Consent, the host id and the issued client: 0075.
* `auth.Identity`: 0076. *(2026-11-11: "and the wizard's client options"
  removed; 0076 no longer adds them.)*
* The Go floor: 0077.
* Value methods (`api_key`, `token_stdin`) through `Start` (MADR D3).
* Promotion into `auth` (MADR D9), and the contract's `v1.0.0`.
* Any agent CLI, and gobble's and Magic's adoption (MADR D11).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts only after the owner has committed the last one.
* **Red, then green.** A new test that calls only existing API is seen red at
  `773b84c` with `python3 -B tools/redgreen.py SPEC.json --expect fail --rev
  773b84c`, and green on the phase's tree with `--expect pass`. A test that
  calls a new name is seen to fail on each of its planted breaks with
  `python3 -B tools/plant.py SPEC.json`, on a scratch clone. This record holds
  each failure line. A break that does not fail its test is a deviation.
* **The refresh tests stay green, unchanged.** Before staging Phases 2, 3 and
  5, `python3 -B tools/redgreen.py REFRESH.json --expect pass` runs the 66
  functions below on the phase's tree, and `git diff --stat` shows none of
  their 15 files changed:
  * `oauth_refresh_test.go`: `TestOAuthRefresh_ReloadsStoreFirst`,
    `TestOAuthRefresh_StoredTokenIsTheOneRefreshed`,
    `TestOAuthRefresh_SameStoredTokenStillRefreshes`,
    `TestOAuthRefresh_OpenAISendsJSON`, `TestOAuthRefresh_GrokStaysForm`,
    `TestOAuthRefresh_TerminalFailuresAreAuthFailure`,
    `TestOAuthRefresh_RetriesTransientFailures`,
    `TestOAuthRefresh_BadRequestIsNotRetried`, `TestOAuthRefresh_ExpiryFromJWT`,
    `TestOAuthRefresh_FiveMinuteWindow`,
    `TestOAuthSession_FailedEarlyRefreshKeepsToken`,
    `TestOAuthSession_RefreshesAgainAfterBackoff`,
    `TestOAuthSession_AuthFailureStillFails`,
    `TestOAuthSession_ShortLivedTokenNotRefreshedEachCall`,
    `TestOAuthSession_RefreshAttemptBounded`;
  * `oauth_refresh_0068_test.go`: `TestOAuthRefresh_RefusedCertificateOnce_0068`;
  * `oauth_refresh_kinds_test.go`: `TestOAuthToken_NoRefreshTokenIsAuthFailure`,
    `TestOAuthRefresh_FailureKinds`;
  * `oauth_rotation_test.go`: `TestOAuthRefresh_RejectedAdoptsSiblingRotation`,
    `TestOAuthRefresh_RejectedRefreshesWithSiblingToken`,
    `TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer`,
    `TestOAuthRefresh_TolerantDecodeKeepsRotation`,
    `TestOAuthRefresh_LargeReplyIsBounded`;
  * `oauth_rotation_kept_test.go`: `TestOAuthSession_RepeatedFailedSavesKeepRotation`,
    `TestOAuthSession_CallerCancelKeepsRotation`,
    `TestOAuthSession_AbandonedWaiterDoesNotResendSpent`;
  * `oauth_spent_refresh_test.go`: `TestOAuthSession_FailedSaveNeverReusesSpentRefresh`,
    `TestOAuthSession_PendingResaveDoesNotBlock`;
  * `oauth_shared_store_stress_test.go`: `TestOAuthSession_SharedStoreStress`;
  * `oauth_session_leader_test.go`: `TestOAuthSession_WaiterOutlivesLeaderCancel`;
  * `oauth_session_test.go`: `TestValidateOAuthSession_RejectsFixture`,
    `TestExchangeOAuthCode_IncludesRedactedBody`,
    `TestRefreshOAuthSession_IncludesRedactedBody`, `TestOAuthTokenError_CapsBody`,
    `TestOAuthSession_SaveFailureKeepsRotation`, `TestOAuthSession_SingleFlight`,
    `TestOAuthSession_SkewsFiveMinutes`, `TestOAuthSession_ChatGPTDetectsIssuer`;
  * `oauth_wait_0071_test.go`: `TestOAuthSessionWait_AfterSave_0071`,
    `TestOAuthSessionWait_Deadline_0071`, `TestOAuthSessionWait_Idle_0071`,
    `TestOAuthSessionWait_TempDir_0071`, `TestOAuthSessionWait_WindowsRemoval_0071`;
  * `plan_session_0043_test.go`: `TestPlanSession_StoreRoundTrip_0043`,
    `TestPlanSession_OlderRecords_0043`, `TestPlanSession_Kind_0043`,
    `TestPlanSession_Refresh_0043`, `TestPlanSession_RefreshSaved_0043`,
    `TestPlanSession_AdoptAndPersistable_0043`,
    `TestPlanSession_CodexRefreshUnchanged_0043`, `TestPlanSession_Revoke_0043`,
    `TestPlanSession_RevokeOffOrigin_0043`, `TestPlanSession_RevokeNoEndpoint_0043`,
    `TestPlanOAuthEndpoints_BuiltinFallback_0043`, `TestRequireSameOrigin_0043`,
    `TestPlanSession_Validate_0043`, `TestPlanSession_Formatting_0043`;
  * `oauth_session_invalidate_test.go`: `TestOAuthSession_Invalidate`,
    `TestOAuthSession_InvalidateTokenIgnoresStale`, `TestOAuthSession_Account`;
  * `tokenstore_oslock_test.go`: `TestFileTokenStore_LockReleasedWhenHolderDies`,
    `TestFileTokenStore_RefreshLockExcludes`;
  * `lockwait_0068_test.go`: `TestLockWait_OutlastsLongestRefresh_0068`;
  * `session_store_kinds_0048_test.go`: `TestValidateOAuthSession_ErrorKinds_0048`,
    `TestNewFileTokenStore_ErrorKinds_0048`, `TestFileTokenStoreDelete_ErrorKinds_0048`.
* **Every other existing test passes unchanged,** in `llmprovider/auth`,
  `wizard` and `llmprovider/providers/...` (R46), except
  `TestDescriptors_OpenAIAndGrokOfferOAuth`, whose `want` table gains D1's
  facts in Phase 1 (MADR D12). Any other test that needs a change is a stop. *(2026-11-12: the owner
  ruled that Phase 3 changes two more, `llmprovider/live_0076_identity_test.go` and
  `wizard/configure_errors_0048_test.go`, each in its one call of `LoginDeviceOAuth`; see
  "Phase 3, deviations".)* *(2026-11-13, the owner's ruling on Grok's
  paste: on a pass of Phase 7 step 4, Phase 8 step 1 changes
  `TestDescriptors_OpenAIAndGrokOfferOAuth`'s Grok row, replaces
  `TestNoLoopback_GrokUnsupported_0074`, and changes the Grok rows of the
  Phase 4 tests it names. None of these is a stop.)*
* **Pre-add, the gate, staging, commits and stop conditions** are those of
  [0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md](0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md),
  "Rules for every phase", with these stop conditions added:
  * a wizard notice, prompt or question changes its text;
  * an existing error's `Error()` text changes (D8);
  * `make api-check` lists a change outside this PLAN's table;
  * coverage of a new package below 80 %, or of `llmprovider` (95.9) or
    `wizard` (86.4) below its floor.

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR and their index rows (`tools/records.py add`).
2. On the owner's acceptance, `tools/records.py status` sets the MADR
   `accepted` and this PLAN `in-progress`.
3. **Apply the supersession notes** the MADR's "Records this decision
   amends" lists, each as a dated amendment or note that quotes the text it
   amends and leaves it standing:
   `0006-MADR-subscription-auth-for-llm-providers.md` §"2. Descriptors grow
   an auth-method list"; `0016-MADR-provider-auth-and-support-baseline.md`
   D6 and D10; `0057-MADR-wizard-device-code-and-paste-prompt.md` D1 and D2;
   `0056-MADR-wizard-credential-only-entry-point.md` D2;
   `0015-MADR-canonical-sdk-api-and-module-layout.md`'s amendment "what
   `auth` holds", D9, and R4's first package;
   `0008-MADR-repair-oauth-loopback-and-session-wiring.md` F5; the standards
   guide's R2 sentence and table (Phase 4 step 4 applies them with the
   code); AGENTS.md "Live tests" (Phase 6 step 5).
4. `python3 -B tools/gate.py --markdown`, then stage.

Acceptance: `make records-check links-check` pass; each note quotes its
target; no source changed.

### Phase 1: method facts (D1, D7 first half)

1. `llmprovider/descriptor.go`: D1's nine fields on `AuthMethod`, each
   documented with the contract values it takes; `Interactive` and
   `HeadlessOK` gain `Deprecated:` naming D1's replacements.
2. The three descriptors set D1's table exactly.
3. `providers/descriptors_test.go`: `TestDescriptors_OpenAIAndGrokOfferOAuth`'s
   `want` gains the facts for OpenAI and Grok; a new
   `TestDescriptors_KiloFacts_0074` pins Kilo's two methods.
4. Run `make lint`. Record whether staticcheck's SA1019 reports the literals
   that still set `Interactive` and `HeadlessOK` (the MADR's unverified item).
   If it does, add one `//nolint:staticcheck // SA1019: kept for readers until
   v2 (0074-MADR D7)` per literal, and record the run.

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestDescriptors_OpenAIAndGrokOfferOAuth` (extended) | every fact of D1's OpenAI and Grok rows | the plan's `FirstPaste` set to `url_or_code` |
| `TestDescriptors_KiloFacts_0074` | Kilo's two rows | Kilo's `device_code` `Refreshable` true |
| `TestAuthMethodComparable_0074` | `AuthMethod` values compare with `==`, so the struct is comparable | a `[]string` field added to `AuthMethod` (compile error, recorded) |

Acceptance: the tests pass; each plant failed; `make api-check` lists the nine
fields as additions only.

### Phase 2: the session kind (D5)

1. `llmprovider/auth/tokenstore.go`: `SessionKind`, its four constants,
   `OAuthSession.Kind`; `fileRecord` gains a `Kind SessionKind` field with
   the JSON name `kind`, `omitempty`; `String`, `GoString`, `MarshalJSON` and
   `LogValue` show it when set.
2. The five copies (MADR C27 to C31) and `oauthSessionState` carry it.
3. `oauth_session.go`: an unexported `inferKind(provider, issuer, resource)`
   holds today's rules. `refreshState` sets `oauthSessionState`'s new `kind`
   to `Kind` when set, else `inferKind`'s. The sites that read the session,
   C19 (`ValidateOAuthSession`) and C20 to C22 (`Owner`, `ChatGPT`,
   `PlanUsage`, each under the session's lock), test the session's kind,
   through an unexported helper that takes the fields, so that no lock is
   taken twice. The sites that read the state, C23 (the refresh body), C24
   (`refreshTokenURL`) and C25 (revocation), test `state.kind`. No branch
   body changes. In C24 the issuer check stays as a guard (MADR D5): a
   session whose issuer is not a built-in one, and that has no token URL of
   its own, gets none, whatever its kind.
   The exported `(*OAuthSession).SessionKind() SessionKind` returns the
   same kind under the session's lock, documented with R51's and R52's
   wording (it returns no error). *(Added 2026-11-11 as a proposed
   revision; exported by the owner's ruling of that day.)* *(2026-11-11: the step named one `kind()` for all seven sites;
   three of them read `oauthSessionState`, not the session.)*
4. Each sign-in sets `Kind`: `LoginBrowserOAuth` and `StartDeviceOAuth` for
   OpenAI (`SessionChatGPT`) and Grok (`SessionGrok`), `LoginChatGPTPlan`
   (`SessionChatGPTPlan`), the Kilo device sign-in (`SessionKiloDevice`), and
   the wizard's access-only session (`SessionChatGPT`).
5. `ValidateOAuthSession` refuses a `Kind` that contradicts the fields, with a
   plain error naming both.

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestSessionKind_EachSite_0074` | for each of the seven sites, a session with `Kind` set and the same session with `Kind` empty take the same branch as at `773b84c`, for each of the four kinds, the empty-issuer OpenAI and Grok sessions, and a Grok and an OpenAI session from another issuer with no token URL, whose refresh gets no built-in token URL with `Kind` set or empty | the session's kind ignores `Kind` and always infers; the plan case of `inferKind` dropped; `refreshTokenURL` tests `state.kind` alone, so the other issuer's Grok session gets xAI's URL |
| `TestSessionKindAccessor_0074` *(added 2026-11-11, with `SessionKind`)* | `SessionKind` returns `Kind` when set, and the inferred kind of a session with `Kind` empty for each case above, empty for a session of no known kind; run under `-race` against a concurrent refresh | `SessionKind` returns `Kind` alone, so an older session reads empty |
| `TestSessionKind_SetBySignIn_0074` | each sign-in sets its kind, against the fake issuers | `LoginChatGPTPlan` leaves `Kind` empty |
| `TestSessionKind_StoreRoundTrip_0074` | `kind` round-trips through `FileTokenStore`; an older record without it loads with `Kind` empty and infers as before | `fileRecord`'s tag spelled `session_kind` |
| `TestSessionFieldsCarried_0074` | every exported stored field (all but `Store`, `HTTPClient`, `Logger`), set by reflection to a distinct value, survives `Save` and `Load`, `persistable`, `refreshState` to the refreshed session, and `adopt` | `Kind` left out of `adopt`; `Kind` left out of `persistable` |
| `TestValidateOAuthSession_KindContradiction_0074` | a `chatgpt_plan` without `Resource`, and a `kilo_device` with a refresh token, are refused | the check removed |

Acceptance: the six tests pass, each plant failed; the 66 refresh tests pass
with their files unchanged. *(2026-11-11: it read "the five tests pass (six,
with the proposed `SessionKind`)"; the owner ruled the method in.)*

### Phase 3: beneath the handle in `auth` (D4, D7 second half, D8, D10)

0. **The prerequisites, at the phase's base.** 0076's release is tagged and
   its changes are in the tree: `auth.Identity` exists, and the OpenAI device
   poll's own deadline (P6 at `773b84c`) returns an error matching the
   shared expiry, `errDeviceCodeExpired` (MADR C45; 0076-MADR D2), not the
   plain error. Either missing is
   a stop. *(2026-11-11: "Phase 3 checks both" named no step.)*
1. **One browser implementation.** An unexported `runBrowserLogin` in
   `oauth_loopback.go` holds the listener, result channel, input goroutine
   and select, with the callback parser, the CORS origin and the exchange as
   parameters. `LoginBrowserOAuth` and `LoginChatGPTPlan` call it. Every test
   in `oauth_loopback_test.go`, `oauth_callback_stray_test.go`,
   `oauth_callback_hygiene_test.go`, `oauth_redirect_test.go`,
   `browser_kinds_0048_test.go` and `plan_login_0043_test.go` passes
   unchanged.
2. **`NoLoopback`** on `OAuthFlowOptions` and `ChatGPTPlanOptions` (MADR
   D4), read by step 1's `runBrowserLogin`: with it set, no listener is
   bound, the redirect URI names the first registered port, and the
   sign-in needs an `InputCode`, else an error matching
   `ErrInvalidRequest`; for Grok it is an error matching `ErrUnsupported`,
   before any request. `StartDeviceOAuth` does not read it: a device
   sign-in binds no listener in the first place, and runs unchanged with it
   set. *(Added 2026-11-11, the owner's ruling on device sign-ins, MADR D4;
   Phase 4's `TestStart_NoLoopbackDevice_0074` holds it.)* *(Restored
   2026-11-11 by the owner's reversal, later the same day (MADR D4). That morning the step was struck through and
   followed by "Struck 2026-11-11 by the owner's ruling: `auth` exports no
   `NoLoopback` field.", with the stop below, kept as it was written. The
   reversal chose MADR D4's option (b), so the stop is lifted and the
   package of option (a) does not join the scope.)*
   * *(Lifted 2026-11-11 by the reversal; kept as written.)* **Open
     question (2026-11-11), a stop before this step:** how
     `x/signin`'s option reaches the one browser function of step 1 is MADR
     D4's open question, which names the options. No `NoLoopback` code is
     written before the owner rules.
   * *(Lifted 2026-11-11 by the reversal; kept as written.)* With the
     answer, this step is rewritten here, dated. Under option (a), the one
     proposed, `runBrowserLogin` reads the mark and binds nothing,
     and the package that holds the mark joins this PLAN's scope, R2's
     table (Phase 4 step 4) and `docs/architecture.md`'s (Phase 6).
   * *(Lifted 2026-11-11 by the reversal: nothing waits now; kept as
     written.)* What waits with it: the three `TestNoLoopback_…_0074` rows
     below,
     Phase 4's `TestStart_NoLoopback_0074`, and Phase 7's steps 2 and 3. The
     rest of this phase, and of Phases 4 to 7, does not.
3. **`ErrSignInDenied` and `ErrSignInExpired`,** each matching
   `ErrAuthFailure`, and the six sites (MADR C42 to C47) wrapped so that
   their errors also match the new one, with `Error()` unchanged.
4. **`auth.VendorCLIPath`** in a new `vendor_path.go`, with the wizard's
   rules; `wizard/import.go`'s `vendorAuthPath` and `homeDir` call it.
5. **`LoginDeviceOAuth` and `NotifyDevice`** gain `Deprecated:`;
   `TestLive_GrokDeviceLogin` and `TestLive_KiloDeviceLogin` call
   `StartDeviceOAuth` and `Wait`, logging the code as the `NotifyDevice`
   callback did (R46: same login, same checks). *(2026-11-12: the three other
   callers of `LoginDeviceOAuth` outside package `auth`, `TestLive_0076Identity`'s Grok
   device half and `TestConfigureLLM_DeviceLoginErrorBare_0048`, move to
   `StartDeviceOAuth` and `Wait` too, by the owner's ruling in "Phase 3,
   deviations"; the `Deprecated:` paragraphs of step 5 stand.)*

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestNoLoopback_OpenAI_0074` | no listener on 1455 or 1457 while the login waits (a test binds both first and the login still starts); the redirect URI names 1455; a pasted URL completes it; without `InputCode` it is `ErrInvalidRequest` | the listener still bound; `InputCode` not required |
| `TestNoLoopback_Plan_0074` | the same for `LoginChatGPTPlan` | the plan ignores `NoLoopback` |
| `TestNoLoopback_GrokUnsupported_0074` | Grok with `NoLoopback` is `ErrUnsupported` before any request | Grok binds its ephemeral port and waits |
| `TestSignInSentinels_0074` | each of the six errors matches `ErrAuthFailure` and its new sentinel, and no other kind | the browser denial wrapped as expired |
| `TestSignInMessagesUnchanged_0074` | each of the six errors' `Error()` equals a literal | a message gains the sentinel's text |
| `TestVendorCLIPath_0074` | `CODEX_HOME`, `GROK_AUTH_PATH`, `GROK_HOME` and the home default, `HOME` and `USERPROFILE` by OS, a nil `lookupEnv`, and `ErrUnsupported` for Kilo | `GROK_AUTH_PATH` read after `GROK_HOME` |

`TestSignInMessagesUnchanged_0074` calls existing API only. It is seen green
at `773b84c` (`redgreen.py --expect pass --rev 773b84c`), which proves its
literals are the old code's, and then to fail on its plant.

*(2026-11-11: the three `TestNoLoopback_…_0074` rows were written for the
exported fields, and wait with step 2. What each asserts stands: no
listener, the redirect URI's port, completion by a paste, `ErrInvalidRequest`
without `InputCode`, and `ErrUnsupported` for Grok. How each switches the
listener off follows the owner's answer.)* *(Later 2026-11-11, by the
owner's reversal: the rows no longer wait. Each switches the listener off
through the two exported fields, as first written.)*

Acceptance: the tests pass, each plant failed; the loopback, plan and
browser-kind tests pass unchanged; the 66 refresh tests pass; `make
api-check` lists `ErrSignInDenied`, `ErrSignInExpired`, `VendorCLIPath`,
`OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback` as
additions only. *(2026-11-11: the api-check clause added with the owner's
reversal, which restores the two fields.)*

### Phase 3a: the Grok paste probe, before Phase 4 (added 2026-11-13)

*(Added by the owner's ruling 2 of 2026-11-13, "Now, before Phase 4"; MADR,
"Amendment 2026-11-13", "The owner's rulings of 2026-11-13 on the probe".
Phase 7 step 4's test is built and run here, so that its outcome shapes
Phase 4's Grok rows. It starts once Phase 3 is committed, which it is
(`3f87e2b`).)*

1. **The test files of Phase 7 step 4,** exactly as that step specifies
   them, and nothing else: `llmprovider/live_0074_grok_paste_test.go`
   (`TestLive_0074GrokPaste`), the four checks appended to
   `llmprovider/live_checks_test.go`, and `TestLiveTwin_GrokPaste_0074`
   appended to `llmprovider/live_twins_test.go`. No existing function
   changes, and no file outside these three.
2. **Its five plants,** by `tools/plant.py` on a scratch clone, never in the
   tree, each failing the twin. This record quotes each failure line.
3. **Checks,** each exit status captured before any filter:
   * `make pre-add-check FILES="..."` on the three files;
   * `go vet -tags live_gateways ./...`, which builds the live test;
   * `make tables-check`: `LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE` is this
     record's variable;
   * `python3 -B tools/gate.py`, through `run-heavy`.
4. **The commit,** then the owner's run, as Phase 7 step 4's "For the
   owner" says, with the arrangement of ruling 3 (phone, send-to-self).
   This record holds every `FINDING` line and the owner's `FINDING page:`
   lines.
5. **The outcome decides what follows:**
   * **On a pass,** Phase 8 step 1 runs next, before Phase 4, in its own
     commit (rulings 1 and 2). Phase 4 then writes its Grok rows to
     `browser_paste`, `url_or_code`, once.
   * **On a fail,** no code changes, and Phase 4 starts with Grok's
     `browser_loopback` rows as written.
   * **On a run that decides nothing,** the owner reruns before Phase 4
     starts, or rules otherwise.

Acceptance: the twin passes and its five plants are caught; `go vet -tags
live_gateways ./...` and `make tables-check` pass; the gate passes; the
owner's run is recorded with its outcome.

### Phase 4: the engine, `x/signin`, and the second module in the gates (D2, D3, D4, D8)

1. **`go.mod`** requires `github.com/maccavelli/go-llmprovider-sdk/authcontract
   v0.1.0`, with no `replace`; `go mod tidy -diff` is clean.
2. **`scripts/check_deps.py`:** `REQUIRES` gains the contract module.
3. **`scripts/check_tables.py`:** `imports()` also lists the packages of
   each nested module (`go -C authcontract list …`), so a row may name
   `authcontract`.
4. **R2:** its sentence "Only `wizard` may leave the standard library, and
   then only for `golang.org/x/term`" (MADR C52) gains "`wizard`,
   `llmprovider/x/signin` and `internal/signinflow` may also import the
   nested module `authcontract`, which is itself standard library only
   (0073-MADR D1)"; its table gains rows for `llmprovider/x/signin`
   (`llmprovider`, `auth`, `authcontract`, `internal/signinflow`,
   `internal/redact`) and `internal/signinflow` (`llmprovider`, `auth`,
   `authcontract`, `internal/redact`), and `authcontract` in `wizard`'s row.
   `docs/architecture.md`'s table gains the same.
5. **`internal/signinflow/`:** the step engine. It takes the login functions
   (browser, plan, device start) as values; runs one sign-in; emits steps in
   order with `Seq`, a browser sign-in's `accepts` and `paste` step only when
   the method's `Paste` fact is set and `NoPaste` is not (MADR D3); checks
   pastes (MADR D4: state, code, `client_id` for `url_only`; at most three
   pastes, so two re-asks, and the third bad paste ends it); passes a pasted
   URL with this state and an `error` to the login, which ends the sign-in
   (MADR D4); and returns the result. *(2026-11-11: the `Paste` fact, the
   count's wording and the pasted denial added.)*
6. **`llmprovider/x/signin/`:** `Methods` (D2), `Start`, `Options`, `SignIn`
   (D3), `Credential` and its five types, `ErrorKindOf` (D8). `APIKey`'s
   `String`, `GoString` and `LogValue` redact. Package doc: experimental,
   R4, and D9's promotion.
7. **`ExampleStart`**, offline: a Kilo device sign-in against an
   `httptest` server, printing each step's kind and the credential's
   `Yields`, with `// Output:` (R50).

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| `TestMethods_0074` | each built-in descriptor's methods, field for field; Claude's implicit key as one `api_key` method; Ollama (no key required) as none | `Paste` not copied; the implicit key added for Ollama too |
| `TestMethodsUseContractValues_0074` | every string fact of every built-in descriptor is one of the contract's constants of its type | a descriptor's `Completion` set to `browser-paste` |
| `TestStart_DeviceSteps_0074` | for OpenAI, Grok and Kilo against fakes: `show_code` with URL, code and expiry, `waiting`, `done` with `Yields`; `Seq` 1, 2, 3; the channel closed | `waiting` emitted before `show_code`; the channel left open |
| `TestStart_BrowserSteps_0074` | for OpenAI: `open_url` with `accepts`, `paste`, then `done` by the loopback; the same by a paste answer; `NoPaste` emits no `paste`. For Grok's `browser_loopback`: `open_url` with no `accepts` and no `paste` step, with `NoPaste` unset, `done` by the loopback, and an answer naming the `open_url`'s `Seq` is `ErrInvalidRequest` | the paste step emitted under `NoPaste`; `accepts` set from `NoPaste` alone, so Grok's step accepts a paste |
| `TestStart_BadPaste_0074` | a paste with another state, one with no code, and a bare code where `url_only` is accepted each bring a new `paste` step whose `text` does not hold the paste; of three bad pastes the first two are re-asked and the third ends `failed` `invalid_request`; a URL with this state and `error=access_denied`, as the first paste, ends `failed` `denied` at once, with no re-ask | the paste echoed in `text`; a fourth paste allowed; the denial re-asked as a bad paste |
| `TestStart_NoLoopback_0074` | with `NoLoopback`, OpenAI's browser sign-in binds no port and completes by a paste; Grok's browser is `ErrUnsupported`; `NoLoopback` with `NoPaste` is `ErrInvalidRequest`, each before any request | `NoLoopback` not passed to `auth`; Grok started |
| `TestStart_NoLoopbackDevice_0074` *(added 2026-11-11, the owner's ruling on device sign-ins; MADR D4)* | with `NoLoopback` set, a device sign-in for OpenAI, Grok and Kilo against fakes gets no error from `Start`, emits the steps `TestStart_DeviceSteps_0074` asserts without the option (`show_code`, `waiting`, `done`; `Seq` 1, 2, 3), and completes by the poll; `auth.StartDeviceOAuth` called directly with `OAuthFlowOptions.NoLoopback` set starts and completes against the same fakes | `Start` refuses a device method when `NoLoopback` is set (`ErrUnsupported`, as for Grok's browser); `StartDeviceOAuth` refuses when `opts.NoLoopback` is set |
| `TestSessionKindsMatchContract_0074` *(added 2026-11-11, the owner's ruling on session-kind labels; 0073-MADR's amendment of that day)* | in `llmprovider/x/signin`, which imports both `auth` and `authcontract`: a table copied from 0073-MADR's amendment, one row per value (`chatgpt`, `chatgpt_plan`, `grok`, `kilo_device`), pairs each `auth` constant with the `authcontract` constant of the same name, and each pair's strings equal the row's value; the `SessionKind` constants that `go/parser` finds in `llmprovider/auth`'s non-test files are exactly the table's `auth` column. The contract's side is held to the same table by 0073-PLAN's `TestSessionKindValues_0073` | `auth.SessionKiloDevice` respelled `"kilo-device"`; a fifth constant, `SessionPlanted SessionKind = "planted"`, added to `llmprovider/auth/tokenstore.go` |
| `TestStart_Answers_0074` | an answer to another `Seq`, to each non-asking step (`open_url`, with and without `accepts`, `show_code`, `waiting`), a second answer to an answered step, or one after the end is `ErrInvalidRequest`; a browser sign-in's paste, sent for the `paste` step's `Seq`, completes it | `Answer` ignores `Seq`; an answer for the `open_url` step's `Seq` taken as the paste |
| `TestStart_PasteIsSecret_0074` | every `paste` step has `Secret` true; an answer sent with `Secret` a non-nil false to a `paste` step is masked in the engine's log, because the engine sets `Secret` from its own step (0073-MADR D6 rule 5) | the engine keeps the caller's `Secret` |
| `TestStart_Cancel_0074` | `Cancel` during `waiting` ends with `failed` `canceled`, and `Wait`'s error matches `context.Canceled`; no goroutine is left (checked with a goroutine count before and after; *2026-11-13: the count of the test's own `synctest` bubble, "Deviation (2026-11-13): two flaky tests root-caused"*) | `Cancel` does not stop the poll |
| `TestStart_VendorImport_0074` | a fixture Codex `auth.json` under a temporary `CODEX_HOME` gives `done` and a `ReadThrough` | `LookupEnv` ignored |
| `TestStart_Unsupported_0074` | `api_key`, `token_stdin` and an unknown pair are `ErrUnsupported` | `api_key` started |
| `TestErrorKindOf_0074` | every row of 0073-MADR D3, by its sentinel, in order: `ErrQuotaExhausted` is `quota` not `rate_limited`, `ErrOffline` is `offline` not `unavailable`, `ErrEmptyAnswer` is `empty_answer` not `incomplete`, `ErrSignInExpired` is `expired` not `auth`; the OpenAI device poll's own 15-minute deadline, reached with a stepped clock as 0076-PLAN's `TestOpenAIDeviceExpiryKind_0076` reaches it, is `expired` | `rate_limited` tested before `quota`; `auth` before `expired`; the poll's deadline returning 0076's plain error again (MADR C51), which maps to `other` |
| `TestCredentialRedacts_0074` | an `APIKey` with a planted key prints no key in `%v`, `%+v`, `%#v`, `%s` or `slog` | `String` returns the key |
| plants of the gates | `make dep-check` with the contract's requirement removed from `REQUIRES`; `make tables-check` with `authcontract` removed from `wizard`'s R2 row | each fails, naming the module |

Acceptance: the tests pass, each plant failed; `make dep-check tables-check
coverage-check docs-check api-check` pass; `x/signin` and
`internal/signinflow` each at 80 % or more.

*(2026-11-13, the owner's rulings; "Phase 4, deviations", at the end of this
record: `scripts/check_api.py`, `tools/apidiff.py` and their tests in
`scripts/test_gates.py` and `tools/test_tools.py` join this phase, since
`api-check` cannot pass once the root requires the nested module; the
gates' plant on `wizard`'s R2 row moves to Phase 5, and the deadline plant
of `TestErrorKindOf_0074` is replaced.)*

*(2026-11-11: `TestStart_NoLoopback_0074` waits with Phase 3 step 2. Its
plant "`NoLoopback` not passed to `auth`" then names the route the owner
chooses, not an exported field.)* *(Later 2026-11-11, by the owner's
reversal: the test no longer waits, and its plant is the option not passed
to the two exported fields, as first written.)*

*(2026-11-11: two rows added by the owner's rulings of that day.
`TestStart_NoLoopbackDevice_0074` holds MADR D4's rule that `NoLoopback`
has no effect on a device sign-in, on which gobble-cli's remote device
sign-ins rely (gobble-cli
`0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3,
and its PLAN's `TestStartSignInRemoteDevice`).
`TestSessionKindsMatchContract_0074` pins `auth.SessionKind`'s four strings
to the contract's session-kind type, which
[0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md)'s
"Amendment 2026-11-11: a session-kind type and its labels" adds before
`authcontract/v0.1.0`. It changes nothing of `auth.SessionKind`, whose v1
shape stays as Phase 2 builds it; it reads `auth`'s constants by
`go/parser`, as 0073-MADR D4's generator reads the contract's, so a value
added to either side fails one of the two tables until a record adds it to
the other, as 0073-MADR D3's table is held in acpext.)*

### Phase 5: the wizard on the step engine, and `Driver` (D6)

1. `wizard/auth.go` and `wizard/device.go`: the browser, plan and device
   paths run through `internal/signinflow`, given the wizard's existing
   seams (`loginBrowserOAuth`, `loginChatGPTPlan`, `deviceLogin`). A default
   renderer reproduces today's notices and prompts.
2. `wizard/driver.go`: `StepDriver`; `wizard/configure.go`:
   `Options.Driver`, documented as MADR D6 says.
3. An `Options.OpenURL` error becomes a warning notice: "could not open the
   browser (%s); open the URL above".

| Test | Asserts | Planted break |
| :--- | :--- | :--- |
| every existing `wizard` test | unchanged and passing: today's texts, races and drains (0057-MADR, 0059-MADR, 0066-MADR) | — (a failure is a stop condition) |
| `TestDriver_ReceivesSteps_0074` | with a `Driver`, a device sign-in's `show_code` and `waiting`, and a browser sign-in's `open_url` and `paste`, reach it in order; the `Prompter` gets no sign-in notice; `ShowDeviceCode` and `OpenURL` are not called | `ShowDeviceCode` still called |
| `TestDriver_PasteAnswerCompletes_0074` | the `Driver`'s paste answer completes a browser sign-in | the answer dropped |
| `TestDriver_ContextEndsPaste_0074` | when the loopback wins, the `Driver`'s pending `SignInStep` sees `ctx` end, and the sign-in returns without waiting for it | `ctx` not cancelled |
| `TestDriver_ErrorEnds_0074` | an error from `SignInStep` on `open_url` ends the sign-in, returned as it is | the error ignored |
| `TestOpenURLErrorWarns_0074` | an `Options.OpenURL` error gives one warning notice and the sign-in still completes by the loopback | the notice omitted |

`TestOpenURLErrorWarns_0074` calls existing API only: it is seen red at
`773b84c` (no notice there), then green.

Acceptance: every existing wizard test passes unchanged; the new tests pass,
each plant failed or seen red; `wizard`'s coverage holds its floor.

*(2026-11-13, Phase 4's Deviation 2, the owner's ruling: the plant of
Phase 4's gate row, `make tables-check` with `authcontract` removed from
`wizard`'s R2 row, is run here, once `wizard` imports `internal/signinflow`
and the contract. It could not fail in Phase 4. Add `authcontract` to
`wizard`'s row of `docs/architecture.md` in this phase too.)*

### Phase 6: docs and the changelog

1. `docs/architecture.md`: the `x/signin` and `internal/signinflow` rows and
   a "Sign-in steps" section; `auth`'s row names the kind and
   `SessionKind`, the sentinels, `NoLoopback` on `OAuthFlowOptions` and
   `ChatGPTPlanOptions`, and `VendorCLIPath`; `x/signin`'s row names its
   `NoLoopback` option, experimental, which passes through to them.
   *(2026-11-11: `NoLoopback` removed from `auth`'s row, since `auth` exports
   no such field, and `SessionKind` added. `x/signin`'s row names its
   `NoLoopback` option as experimental, once it is built.)* *(Later
   2026-11-11, by the owner's reversal: `NoLoopback` is back in `auth`'s
   row.)*
2. `README.md`'s login section: method facts, `x/signin` (experimental), and
   `Options.Driver`.
3. `CHANGELOG.md` `[Unreleased]`: **Added** each exported name above;
   **Deprecated** the four; **Changed** "an `Options.OpenURL` error is shown
   as a warning".
4. `docs/guides/migrating-from-mcplib.md`: no row changes (no `mcplib`
   identifier is replaced); `make parity-check` confirms it.
5. `AGENTS.md` "Live tests": `LLMPROVIDER_LIVE_DEVICE_LOGIN` (MADR C53)
   names "the Grok, Kilo and OpenAI device logins", for Phase 7's
   `TestLive_OpenAIDeviceLogin`. *(2026-11-13, the owner's ruling: moved to
   Phase 7 step 1; see "Phase 6, deviation".)*
6. `make docs-check changelog-check parity-check links-check tables-check`.

### Phase 7: owner-run probes (MADR D12)

Each is owner-run, asks first, uses a scratch store in a temporary directory
and a scratch home, and is never assumed done. The results go in this record.

*(2026-11-11: steps 2 and 3 run `x/signin` with `NoLoopback`, the option's
only callers, so they wait on MADR D4's open question, as Phase 3 step 2
does.)* *(Later 2026-11-11, by the owner's reversal: they no longer wait.
They run `x/signin` with `NoLoopback`, which passes it to `auth`'s two
fields; gobble-cli's remote ACP sign-ins are the option's consumer caller,
and these two its probes.)*

1. *(2026-11-13: the `AGENTS.md` "Live tests" line of Phase 6 step 5 lands in
   this step's commit; see "Phase 6, deviation".)* **A live OpenAI device sign-in,** `TestLive_OpenAIDeviceLogin` in
   `llmprovider/live_oauth_login_test.go`, under
   `LLMPROVIDER_LIVE_DEVICE_LOGIN`: start, approve, one short generation
   through the Codex backend, revoke. *Owner-run; asks first; scratch store;
   quota: one short generation on the owner's ChatGPT plan, under Codex's
   client.*
2. **U2, OpenAI's browser sign-in by paste from another device,**
   `TestLive_0074PasteBrowser`, under `LLMPROVIDER_LIVE_0074` and
   `LLMPROVIDER_LIVE_BROWSER_LOGIN`: `x/signin` with `NoLoopback`; the owner
   opens the URL on another device and pastes the address it lands on. It
   checks the session refreshes once, then revokes. *Owner-run; asks first;
   scratch store; quota: none (no generation).*
3. **U3, the plan by paste from another device,** `TestLive_0074PastePlan`,
   under `LLMPROVIDER_LIVE_0074` and `LLMPROVIDER_LIVE_CHATGPT_PLAN`: the same
   with `chatgpt_plan`; then, with ports 1455 held by the test, the port
   fallback to 1457 through the loopback. *Owner-run; asks first; scratch
   store; quota: none for the sign-in, but each run registers a new client
   under the owner's ChatGPT account, listed in its settings
   (`docs/reports/0017-REPORT-openai-sign-in-for-agents.md`, "The probe's
   runs").*
4. **The SDK side of U1, Grok's browser sign-in from another device,**
   `TestLive_0074GrokPaste`, under `LLMPROVIDER_LIVE_0074` and
   `LLMPROVIDER_LIVE_BROWSER_LOGIN`: the owner opens the URL on another
   device and records what accounts.x.ai shows when its fetch to the
   loopback fails. *Owner-run; asks first; scratch store; quota: none.* A
   pasteable result changes Grok's facts to `browser_paste` by an amendment,
   never silently.

   *(2026-11-13, the owner's ruling "Probe Grok paste; keep 0247" (MADR,
   "Amendment 2026-11-13: a live probe of Grok's paste, and what its outcome
   changes"). It is the same step and the same test, not a second one. The
   test now also completes the sign-in by the paste, and its outcome decides
   MADR D1's Grok fact and D4's Grok refusal. What the paragraph above
   records, what accounts.x.ai shows, stays one of its findings. The step
   as amended:)*

   * **The route, with shipped code only.** `auth.LoginBrowserOAuth` for
     Grok, with an `InputCode` and with `NoLoopback` unset. The shipped
     code races the `InputCode` against the bound listener (MADR C58);
     `NoLoopback` is refused for Grok (MADR C56) and is not set. No
     test-only seam, and no change to `auth`.
   * **The arrangement: a browser that cannot reach the host's loopback.**
     The listener binds the host's `127.0.0.1` only (MADR C59), so no other
     device's browser can reach it. The test runs on the host, and the
     owner opens its authorize URL on another device:
     * recommended, the phone, which is Magic's case: an SSH client on the
       phone with two sessions to the host, one running the test and one
       writing the paste file; the authorize URL is copied from the first
       session into the phone's browser;
     * or a second computer's browser, with the test on a dev host over SSH
       (the vendor's remote-VM case, MADR C64).

     No port is forwarded to the host for the session. A blocked port on
     the host itself is not used: blocking it needs a packet-filter rule
     run as root, or a seam that binds the listener elsewhere, which would
     be code in `auth`.

     *(2026-11-13, the owner's ruling 3, "Phone + send-to-self": the
     arrangement above is superseded, and kept as written. The test runs
     on the owner's Mac, which is the host, and prints the authorize URL.
     The owner sends the URL to the phone by any channel, signs in on the
     phone, and copies what the page shows. The owner sends that text back
     to the Mac by any channel, and writes it into the paste file from a
     terminal on the Mac. No SSH on the phone. The phone's browser cannot
     reach the Mac's `127.0.0.1` (MADR C59), which is the condition the
     probe needs, and the loopback line below still catches a run where
     it could.)*
   * **How the paste reaches the test.** `go test` gives the test binary no
     standard input: Go 1.27.4's `src/cmd/go/internal/test/test.go` sets
     the test command's `Dir`, `Stdout` and `Stderr`, and no `Stdin` (read
     2026-11-13). So `InputCode` reads a file instead.
     * `LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE` names a path that must not
       exist when the test starts; the test fails if it does.
     * `InputCode` polls once a second until the file holds a
       newline-terminated line, reads at most 8 KiB, removes the file and
       returns the line. It returns `ctx`'s error when the sign-in ends
       first.
     * The person writes the file in the second session: `cat >` the path,
       the paste, Enter, then Ctrl-D. *(2026-11-13, ruling 3: from a
       second terminal on the Mac, not an SSH session.)*
     * The pasted value is a secret (MADR Q11). It travels only over that
       SSH session, never through a chat or note service, and nothing logs
       it. *(2026-11-13, ruling 3: superseded, and kept as written. The
       text travels by whatever channel the owner chooses, both ways. What
       a copy of it can do is narrow: the code is used once, and its
       exchange needs the PKCE verifier, which never leaves the test
       process (RFC 7636). Nothing logs it, and this record quotes none of
       it.)*
   * **What the owner pastes, and what the owner looks at.** The owner
     pastes whatever the page or the address bar offers after consent. Both
     the SDK's parser and the vendor's take the full callback URL
     (`http://127.0.0.1:<port>/callback?code=…&state=…`), and a bare code
     (MADR C61, C65). The records do not say which of these the page
     offers, if either, so the owner records it by hand, in this record, as
     `FINDING page:` lines, quoting no URL, code or token:
     1. what the page said after consent: a success claim, an error, a URL
        or a code shown, or a copy control;
     2. what the address bar held: still `accounts.x.ai`, or a `127.0.0.1`
        callback that the browser failed to load;
     3. whether a local-network permission prompt appeared, and the answer
        given;
     4. the device class and browser (for example "phone, mobile Chrome"),
        never a device's name. *(2026-11-13, ruling 3: the device class
        is "phone".)*
   * **The test:** `TestLive_0074GrokPaste`, in a new
     `llmprovider/live_0074_grok_paste_test.go` (build tag
     `live_gateways`).
     * It skips unless `LLMPROVIDER_LIVE_0074=1`,
       `LLMPROVIDER_LIVE_BROWSER_LOGIN=1` and
       `LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE` are all set.
     * Its `OpenURL` is `openAuthorizeURL(t, "referrer", cancel)`, which
       checks the referrer and logs the URL.
     * The sign-in has the flow's own ten minutes (P11). The test's
       context has eleven.
     * When `LoginBrowserOAuth` returns, the test logs one `FINDING
       sign-in:` line from `pasteFinding`, and stops unless the run passed.
       The lines, and what each means:
       * `FINDING sign-in: completed by a paste of <shape>` is the pass.
         `<shape>` comes from `pasteShape`: "a callback URL with code and
         state" or "a bare code", with the URL's scheme, whether its host is
         a loopback address, and its path. It never holds a value.
       * `FINDING sign-in: completed by the loopback, not a paste; the
         browser reached the host, so this run decides nothing` means rerun
         with the arrangement fixed.
       * `FINDING sign-in: xAI refused the pasted <shape>: <error>` is a
         fail, when a well-formed paste was exchanged and refused.
       * `FINDING sign-in: the paste was refused before any request:
         <shape>: <error>` covers a state mismatch, a missing code or a
         pasted denial. It decides nothing, unless the page's lines show
         the page offers nothing else.
       * `FINDING sign-in: nothing was pasted before the deadline` is a fail
         when the page's lines show nothing pasteable, and otherwise a
         rerun.
     * On a pass, `checkIdentitySession` binds the session to a scratch
       `FileTokenStore` in `t.TempDir()` and lists models with it, with no
       generation (`FINDING use:`, P9). Then `checkIdentityRevoked` revokes
       it and checks that a refresh is refused (`FINDING revocation:`,
       P10).
     * A session that a failed check leaves behind is revoked when the test
       ends, as `probeSignIn` does.
   * **The checks,** appended to `llmprovider/live_checks_test.go`
     (untagged), with no existing function changed:
     * `pasteRecord`: whether `InputCode` returned a value, and its shape;
     * `pasteFromFile(path, rec)`: the `InputCode` described above;
     * `pasteShape(raw)`;
     * `pasteFinding(rec, err) (line string, passed bool)`, which decides
       the line from the record and `LoginBrowserOAuth`'s error.
   * **The twin:** `TestLiveTwin_GrokPaste_0074`, in
     `llmprovider/live_twins_test.go`. It runs:
     * `pasteFromFile` on a file in `t.TempDir()` written in two writes,
       first without its newline and then with it, and checks that the
       file is removed after the read;
     * `pasteFromFile` with `ctx` ending first;
     * `pasteShape` and `pasteFinding` over a table of every outcome above,
       with a planted code that no line may hold.

     Each planted break below, run by `tools/plant.py` on a scratch clone
     and never in the tree, must fail the twin
     ([0046-MADR-v1-15-0-follow-ups.md](0046-MADR-v1-15-0-follow-ups.md)
     D3):
     * `pasteFinding` reports a completion by a paste when `InputCode`
       returned nothing (the loopback won);
     * `pasteShape` calls a URL with a code and no `state` "a callback URL
       with code and state";
     * `pasteShape` writes the pasted value into its shape;
     * `pasteFromFile` leaves the file in place;
     * `pasteFromFile` takes a line before its newline.

     What the twin does not cover: xAI's page and its exchange, which only
     the live run shows. The session checks after a pass are 0076's. They
     are held by `TestLiveTwin_Identity_0076` (P12) and its plant, and are
     not changed.
   * **For the owner.** From the repository root on the host:

     ```bash
     export LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE="$(mktemp -d)/paste"
     echo "$LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE"
     LLMPROVIDER_LIVE_0074=1 LLMPROVIDER_LIVE_BROWSER_LOGIN=1 \
       go test -tags live_gateways -count=1 -timeout 15m -run '^TestLive_0074GrokPaste$' -v ./llmprovider/
     ```

     One package with `-v`, so the authorize URL is printed as it is
     logged; being one package, the run needs no fleet host and no queue.
     In the second session: `cat >` the echoed path, the paste, Enter,
     Ctrl-D. *(2026-11-13, ruling 3: on the owner's Mac, the second session
     is a second terminal there; the URL goes to the phone, and the page's
     text comes back, by any channel.)*
   * *Owner-run; asks first; scratch store; quota: none at xAI (a listing,
     no generation).*
   * **The outcome.** A pass, a fail or a rerun, as the lines above decide,
     is recorded in the execution record, with every `FINDING` line quoted
     and the owner's `FINDING page:` lines. A pass starts Phase 8 step 1. A
     fail changes no code: the MADR's D1 and D4 Grok bullets get a dated
     line saying so, and 0008-MADR gets its dated note (MADR, "Amendment
     2026-11-13").
   * **Acceptance:** the twin's five plants are caught; `go vet -tags
     live_gateways ./...` builds the test; `make tables-check` passes; the
     owner's run is recorded.

   *(2026-11-13, outcome: a pass. The owner's run of 2026-11-13 completed
   the sign-in by a paste of a bare code from a phone; the record is "Phase
   3a, the owner's run (2026-11-13)", at the end of this record. Phase 8
   step 1 runs next.)*

   *(2026-11-13, the owner's ruling 2, "Now, before Phase 4": this step's
   test files are built, and its probe run, in Phase 3a, before Phase 4.
   The step is done when Phase 3a's run is recorded, and Phase 7 does not
   run it again; a pass's change then follows at once (Phase 8 step 1).)*
5. **The two moved live device tests,** `TestLive_GrokDeviceLogin` and
   `TestLive_KiloDeviceLogin` in `llmprovider/live_oauth_login_test.go`,
   under `LLMPROVIDER_LIVE_DEVICE_LOGIN`, once, after Phase 3 step 5 moves
   them to `StartDeviceOAuth` and `Wait`: each completes its sign-in and
   passes the checks it passed before the move (R46: same login, same
   checks). `go vet -tags live_gateways` shows only that they build.
   *Owner-run; asks first; scratch store; quota: none (no generation).*
6. **U14, the wizard write race** (not a sign-in): on a scratch clone, a test
   that runs `ConfigureLLM` with a fake listing provider calling the new
   session's `Token` on its own goroutine while `save` sets `Store`, under
   `go test -race -count=50`. *Run by the implementer; no sign-in; no quota.*
   A race found is a deviation, fixed by setting `Store` before the listing
   or through a locked method, under an amendment.
   *(2026-11-14: two races were found. See "Phase 7, step 6, deviation
   (2026-11-14)", at the end of this record; the fix is a locked
   `UseStore` and a `Save` that reads under the session's lock.)*

`scripts/check_tables.py` passes: `LLMPROVIDER_LIVE_0074` is this record's
variable (AGENTS.md "Live tests"). So is
`LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE`, step 4's paste file *(added
2026-11-13)*: the script accepts a suffix of `LLMPROVIDER_LIVE_<NNNN>` that
a record numbered `NNNN` names.

*(2026-11-14: so is `LLMPROVIDER_LIVE_0074_PASTE_FILE`, the paste file of
steps 2 and 3, `TestLive_0074PasteBrowser` and `TestLive_0074PastePlan`, which
the two share. It must not exist when a test starts, as step 4's must not.
Step 2 also reads `LLMPROVIDER_LIVE_BROWSER_LOGIN`; step 3 also reads
`LLMPROVIDER_LIVE_CHATGPT_PLAN`. Both are named in AGENTS.md. The other
variables of this phase are `LLMPROVIDER_LIVE_DEVICE_LOGIN`, step 1's.)*

*(2026-11-14, later: every step of this phase is done, and the phase is
complete. Steps 1, 2, 3 and 5 passed on the owner's runs of 2026-11-14, step
3 on its second run, after the probe defect its first run found was fixed in
`c47602b`; step 4 passed in Phase 3a on 2026-11-13; step 6 found two races,
fixed in `67be2b7`. See "Phase 7, the owner's runs (2026-11-14)", at the end
of this record.)*

### Phase 8: the release

1. **Only on a pass of Phase 7 step 4** *(added 2026-11-13, the owner's
   ruling "Probe Grok paste; keep 0247"; MADR, "Amendment 2026-11-13: a live
   probe of Grok's paste, and what its outcome changes")*. It is its own
   commit, before the rest of this phase. If the owner has the probe run
   before Phase 4, this step runs before Phase 4 instead, and Phase 4's Grok
   rows are written to the new fact once. On a fail, or with no run, the
   step is skipped and the execution record says so. *(2026-11-13, the
   owner's rulings 1, 2 and 4: in 0074, before the tag, in its own commit;
   the probe runs in Phase 3a, so on a pass this step runs after Phase 3a
   and before Phase 4, and the table's third row, for Phase 4's tests,
   does not arise; `url_or_code` is the value whatever the page
   offers.)*
   1. `llmprovider/providers/grok/descriptor.go`: Grok's `browser_oauth`
      gets `Completion` `browser_paste` and `Paste` `url_or_code`, with no
      `FirstPaste` (MADR D1, C62).
   2. `llmprovider/auth/oauth_loopback.go`:
      * `requireNoLoopbackInput` loses its Grok case (MADR C56), so Grok
        under `NoLoopback` needs an `InputCode`, as OpenAI does;
      * `browserEndpoint` with no listener names Grok's redirect,
        `http://127.0.0.1:<port>/callback`, with the port drawn at random
        from 49152 to 65535 and never bound; OpenAI's stays as it is
        (MADR C60);
      * the doc comments of `OAuthFlowOptions.NoLoopback`,
        `LoginBrowserOAuth`, `requireNoLoopbackInput` and `browserEndpoint`
        say so.
   3. If Phase 4 has landed: `x/signin`'s `Start` no longer refuses Grok's
      browser sign-in under `NoLoopback`, because its `Paste` fact is now
      set (MADR D4); the `NoPaste` refusal stays.
   4. `CHANGELOG.md` `[Unreleased]`, **Changed**: "Grok's browser sign-in
      completes by a pasted callback URL or code, and accepts `NoLoopback`".
      `README.md`'s login section, if it names Grok's method as loopback
      only.
   5. A dated note in `0008-MADR-repair-oauth-loopback-and-session-wiring.md`
      under its 2026-11-12 amendment: F5's inference is replaced by the
      probe's observation, quoted from Phase 7 step 4's record.

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestDescriptors_OpenAIAndGrokOfferOAuth` (P8; its Grok row) | Grok's `browser_oauth` is `browser_paste`, `url_or_code`, no `FirstPaste` | the fact left `browser_loopback` |
   | `TestNoLoopback_Grok_0074` (replaces `TestNoLoopback_GrokUnsupported_0074`, P7) | with `NoLoopback`: no listener is bound; the authorize URL's `redirect_uri` is `http://127.0.0.1:<port>/callback`, with the port in 49152 to 65535; a pasted URL completes the sign-in against a fake issuer; without `InputCode` the error matches `ErrInvalidRequest`, before any request | Grok still refused; OpenAI's port and path named for Grok |
   | Phase 4's `TestStart_BrowserSteps_0074`, `TestStart_NoLoopback_0074`, `TestStart_Answers_0074` and `TestMethods_0074`, their Grok rows (when Phase 4 has landed) | Grok's `open_url` accepts `url_or_code` and is followed by `paste`, unless `NoPaste`; Grok under `NoLoopback` completes by a paste; the "`open_url` without `accepts`" case of `TestStart_Answers_0074` takes `NoPaste` instead of Grok | `x/signin` still refuses Grok; `accepts` left empty for Grok |

   Acceptance: the tests pass and each plant fails; `make api-check` lists
   no change from this step (no name is added or removed); the full gate
   passes before the tag.

With 0075-PLAN's last phase: the changelog section takes the next minor
version after 0076's (`v1.31.0` if 0076 shipped as `v1.30.0`); the full gate;
`tools/apidiff.py` against the previous tag lists exactly the tables above,
with 0075-PLAN's, `OAuthFlowOptions.NoLoopback` and
`ChatGPTPlanOptions.NoLoopback` among them *(named 2026-11-11, with the
owner's reversal)*;
the tag on the owner's word; then `tools/consumer.py` and `tools/offline.py`.

*(2026-11-12: by the owner's ruling of that day, `v1.31.0` also carries
[0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md](0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md),
proposed on this date, which has no tag of its own (its Phase 6). Once it is
approved, this phase starts after its Phases 1 to 5 are committed and its
owner-run probe is recorded; the section holds its changelog entries; and
`tools/apidiff.py` lists its two names, `(*auth.OAuthSession).UseIdentity`
and `(*llmprovider.Settings).ClientVersion`, among the tables above.)*

## Verification

Each command's exit status is captured before any filter.

* `go test -race -count=1 ./llmprovider/auth/... ./llmprovider/x/... ./internal/signinflow/... ./wizard/... ./llmprovider/providers/...`
* `go test -race -count=3 ./llmprovider/auth/` (the refresh code, three
  times)
* `python3 -B tools/redgreen.py REFRESH.json --expect pass` (the 66)
* `make api-check` and `python3 -B tools/apidiff.py v1.29.0` (or the latest
  tag): the additions only; `x/signin`'s marked experimental
* `make contract-check` (the contract module is unchanged by this PLAN)
* `make dep-check tables-check coverage-check generate-check docs-check
  changelog-check parity-check records-check links-check disclosure-check
  gate-selftest`
* `make lint vuln`, `go mod tidy -diff`
* `python3 -B tools/gate.py`
* `python3 -B tools/claims.py docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md`
* `go vet -tags live_gateways ./...` (the moved and new live tests build)
* Every plant and red run of Phases 1 to 5, quoted.

## Rollout and Rollback

* **Order.** First,
  [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md)
  raises the Go floor to 1.27.4, before any code of this series *(added
  2026-11-11, when the owner approved 0077)*. Then 0076's minor, `v1.30.0`,
  which ships that floor, now (the owner's DF, 2026-11-10);
  `authcontract/v0.1.0` (0073) before this PLAN's Phase 4; this PLAN and
  0075's ship in the next SDK minor. gobble
  pins that minor and the contract, then drives its ACP sign-ins through
  `x/signin` and its `--json` sign-ins through the wizard's `Driver` (gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3, proposed; the owner's QE (a)), and tags `acpext/v0.5.0` and then gobble-cli's own next release, `v0.6.0`.
  *(2026-11-11: the owner ruled that `v0.6.0` is the release that must pin
  the same acp-go-sdk2 tag as magic-cli-remote; both move from `v1.35.0` to
  `v1.36.0` together before it, each by a record of its own.)*
  *(2026-11-12: gobble-cli's release on the contract, named `v0.6.0` above,
  is now `v0.7.0`. The owner decided that gobble-cli releases `v0.6.0` now,
  from its main, without the sign-in redesign, and that the redesign is
  `v0.7.0` (gobble-cli
  `docs/decisions/0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`, D1
  and D2). The ruling of 2026-11-11 stays true of `v0.6.0`, which is cut on
  acp-go-sdk2 `v1.36.0` as magic-cli-remote is (0055-MADR F8); `v0.7.0` must
  meet the same lockstep when it is cut (its D2).)* Magic
  consumes gobble's steps after those tags, and the contract directly for its
  own mappers (magic-cli-remote
  `docs/decisions/0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`).
  Promotion (MADR D9) comes after both have shipped.
* **Rollout.** Additive. A caller that reads no new field sees three
  changes: a session saved by this version carries `kind` in its record; an
  `Options.OpenURL` error is shown as a warning; and nothing else, since the
  wizard's texts are unchanged.
* **Downgrade.** An older SDK reads a record with `kind` and ignores it
  (`FileTokenStore` decodes with `encoding/json`'s default, which drops an
  unknown field), and infers the kind as before.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** names outside `x/` stay (R48); a wrong fact
  is corrected by its value; `x/signin` may change in the next minor.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, checked by `tools/claims.py`.

<!-- claims: self=773b84c go-llmprovider-sdk=5e0ec2c -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The Grok live device test Phase 3 moves | `llmprovider/live_oauth_login_test.go:100` | `func TestLive_GrokDeviceLogin(t *testing.T) {` |
| P2 | and Kilo's | `llmprovider/live_oauth_login_test.go:124` | `func TestLive_KiloDeviceLogin(t *testing.T) {` |
| P3 | Both run under the device-login variable | `llmprovider/live_oauth_login_test.go:101` | `if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {` |
| P4 | whose AGENTS.md scope Phase 6 widens | `AGENTS.md:247` | `` `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok and Kilo `` |
| P5 | R2's sentence Phase 4 amends | `docs/guides/api-standards.md:31` | `` Only `wizard` may leave the standard library, and then only for `` |
| P6 | The OpenAI poll's own deadline, mapped `other` until 0076 D2 | `llmprovider/auth/oauth_device.go:234` | `return nil, errors.New("oauth: OpenAI device-code login timed out after 15 minutes")` |
| P7 | (2026-11-13, this repository at `5e0ec2c`) The Grok refusal test a pass replaces | `go-llmprovider-sdk:llmprovider/auth/no_loopback_0074_test.go:186` | `func TestNoLoopback_GrokUnsupported_0074(t *testing.T) {` |
| P8 | The descriptor pin a pass changes | `go-llmprovider-sdk:llmprovider/providers/descriptors_test.go:36` | `func TestDescriptors_OpenAIAndGrokOfferOAuth(t *testing.T) {` |
| P9 | The session check the Grok probe reuses | `go-llmprovider-sdk:llmprovider/live_checks_test.go:289` | `func checkIdentitySession(` |
| P10 | and the revocation check | `go-llmprovider-sdk:llmprovider/live_checks_test.go:309` | `func checkIdentityRevoked(` |
| P11 | The browser sign-in's own deadline | `go-llmprovider-sdk:llmprovider/auth/oauth_loopback.go:25` | `oauthBrowserTimeout         = 10 * time.Minute` |
| P12 | The twin that holds those two checks | `go-llmprovider-sdk:llmprovider/live_twins_test.go:297` | `func TestLiveTwin_Identity_0076(t *testing.T) {` |

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
* **Phase 1's code waits until `ci.yml` passes on `main`.** Run 43295522137
  on `e5eea9d` failed on Ubuntu in `TestCachedMetadataWith_Unavailable`, a
  catalog test this PLAN does not touch
  ([0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md](0078-MADR-cached-metadata-refusal-test-uses-a-refused-option.md)).
  Phase 1 starts after that record's fix has a passing `ci.yml` run on
  `main`, which its PLAN's Phase 4 records.

### Phase 0, completed (2026-11-12)

The coordinator committed Stage A as `fc699ec`, and then, the Phase 0 entry
above having left step 3 open, ruled that Phase 0 is finished before Phase 1
(this PLAN's own order; the owner's acceptance covers Phase 0).

* Step 3: six dated notes, each an appended section "Amendment 2026-11-12:
  ... (0074)" that cites 0074-MADR, quotes the text it amends exactly, and
  leaves that text standing:
  * `0006-MADR-subscription-auth-for-llm-providers.md`, §"2. Descriptors grow
    an auth-method list": `Interactive` and `HeadlessOK` deprecated, 0074-MADR
    D1's facts replace them;
  * `0016-MADR-provider-auth-and-support-baseline.md`: D6 generalised, D10
    affirmed;
  * `0057-MADR-wizard-device-code-and-paste-prompt.md`: D1 and D2 kept, as
    special cases of the step renderer;
  * `0056-MADR-wizard-credential-only-entry-point.md`: D2 kept, the Kilo
    method's `yields` is `device_key`;
  * `0015-MADR-canonical-sdk-api-and-module-layout.md`: the amendment "what
    `auth` holds" gains the session kind and `SessionKind`, the two
    sentinels, the two `NoLoopback` fields and `VendorCLIPath`; D9 governs
    D10; `llmprovider/x/signin` is the first package under R4's tree;
  * `0008-MADR-repair-oauth-loopback-and-session-wiring.md`: F5's inference
    stays until D12's Grok probe, and the method's fact is `browser_loopback`
    until then.
  The quotations were checked against each file at `HEAD` by `grep -F` on one
  line of each. The standards guide's R2 sentence and rows, and `AGENTS.md`
  "Live tests", are not edited here: they wait for Phase 4 step 4 and Phase 6
  step 5, as step 3 says. No source file changed.
* Step 4: `python3 -B tools/gate.py --markdown`, through the queue
  (`run-heavy`, which waited for load below 10 and started at 9.7): exit 0,
  "gate: 11 of 11 passed" (parity-check, records-check, tables-check,
  changelog-check, markdownlint, links-check, disclosure-check,
  gate-selftest, deny-scan, recordlint, claims), each rc=0, elapsed 647 s.
  It ran before this entry was written, so the record checks were run again
  after it, each exit 0 (below).
* Acceptance: `make records-check` and `make links-check` pass; each note
  quotes its target; no source changed. Met.
* Phase 1's wait (the last bullet of the Phase 0 entry) is met: `ci.yml` run
  43308484802 on `cae3ccb` passed on all three runners. Phase 1 starts after
  the coordinator commits this phase.

### Phase 1 (2026-11-12)

Base: `main` at `72854fd` (Phase 0 committed), tree clean. The MADR and this
PLAN were read in full first. No claim of the MADR was found false and no
existing test, gate or golden changed, but `TestDescriptors_OpenAIAndGrokOfferOAuth`'s
`want` table, which the PLAN names.

* Steps 1 and 2, `llmprovider/descriptor.go` and the three descriptors. The
  nine fields of D1 are on `AuthMethod`, each documented with the contract
  values it takes (`Paste` states that empty means nothing may be pasted,
  the contract's `PasteNone`); `Interactive` and `HeadlessOK` carry
  `Deprecated:` naming `Kind`/`Completion` and 0074-MADR D7, their values
  unchanged. The OpenAI, Grok and Kilo literals set D1's table, applied by
  script from the table and checked by the tests below (13 rows; `FirstPaste`
  only on the plan; Grok's browser method is `browser_loopback` with no
  paste). The openai and grok files each gained a small unexported constant
  block for the values `goconst` flags at three uses (see Step 4), with a
  comment that they repeat the contract's values as strings (R2).
* Step 3, `providers/descriptors_test.go`:
  `TestDescriptors_OpenAIAndGrokOfferOAuth`'s `want` gains the facts for all
  eleven OpenAI and Grok rows; new `TestDescriptors_KiloFacts_0074` pins
  Kilo's two rows whole; new `TestAuthMethodComparable_0074` compares values
  with `==` and uses one as a map key.
* Step 4, `make lint` (through `run-heavy`). **First run, exit 2, 5 issues,
  all `goconst`** in new code, none in a test: `grok/descriptor.go:25:21`
  ("string api_key has 3 occurrences, make it a constant (goconst)"), `:28:21`
  `none`; `openai/descriptor.go:28:21` `none`, `:38:21` `oauth_session`,
  `:40:21` `openai` (4 occurrences). Fixed by the unexported constants
  `factAPIKey`, `factNone` (grok) and `factOAuthSession`, `factNone`,
  `factOpenAI` (openai), not by a `nolint`; no exported name is added. Second
  run, exit 0: four "0 issues." (host and `GOOS=windows`, root and
  `authcontract`). **The MADR's unverified item: staticcheck's SA1019 does not
  report the literals** that still set `Interactive` and `HeadlessOK` (no
  SA1019 line in the output), so no `//nolint:staticcheck` is added. That the
  check is live was shown on a scratch copy of the tree, never in it, with a
  test file reading `m.Interactive` through a selector: `zz_sa_test.go:11:6:
  SA1019: (github.com/maccavelli/go-llmprovider-sdk/llmprovider.AuthMethod).Interactive
  is deprecated: read Kind and Completion. The value is unchanged and stays
  until v2 (0074-MADR D7). (staticcheck)`, exit 1.
* **Plants,** `python3 -B tools/plant.py` on a spec of three, on its scratch
  clone of the working tree, never in the tree, exit 0, "plant: all 3
  plant(s) caught":
  * the plan's `FirstPaste` set to `url_or_code`: `--- FAIL:
    TestDescriptors_OpenAIAndGrokOfferOAuth`, `descriptors_test.go:182:
    openai AuthMethods = ...`, `FAIL ... llmprovider/providers`;
  * Kilo's `device_code` `Refreshable` true: `--- FAIL:
    TestDescriptors_KiloFacts_0074`, `descriptors_test.go:219: kilo
    AuthMethods = ...`;
  * a `[]string` field added to `AuthMethod`: `FAIL ...
    llmprovider/providers [build failed]`, `descriptors_test.go:229:5:
    invalid operation: a != b (struct containing []string cannot be
    compared)`, `:233:5` the same for `a == b`, `:236:14: invalid map key
    type llmprovider.AuthMethod`.
* Checks, each exit status captured before any filter, the heavy ones through
  `run-heavy`:
  * `go test -count=1 ./llmprovider/providers/`, exit 0.
  * `make pre-add-check FILES="..."` on the five Go files, exit 0:
    "go-precheck: 5 file(s) clean (gofmt, golangci-lint, go vet, go test,
    govulncheck)."
  * `make api-check`, exit 0: "against v1.30.0, 0 incompatible change(s)
    outside llmprovider/x/". `python3 -B tools/apidiff.py v1.30.0`, exit 0:
    nine lines, all "Compatible changes", `./llmprovider.AuthMethod.` `Kind`,
    `Completion`, `Paste`, `FirstPaste`, `Yields`, `ClientIdentity`, `Slot`,
    `Refreshable`, `Revocable`: "added". Nothing else.
  * `python3 -B tools/gate.py --timeout 5400`, exit 0, 929 s: "gate: 30 of 30
    passed", every check rc=0 (the six `go vet` runs, test-race-cover,
    test-shuffle, test-repeat, lint, parity-check, dep-check, coverage-check,
    api-check, contract-check, generate-check, records-check, tables-check,
    fixtures-check, changelog-check, docs-check, gate-selftest, markdownlint,
    links-check, disclosure-check, deny-scan, tools-test and the rest). The
    full-module test runs were those of the gate; no other `go test ./...`
    was run.
  * The gate ran before this entry was written, so the record checks were run
    again after it, each exit 0 (below).
* Acceptance: the tests pass, each plant failed, and `make api-check` lists
  the nine fields as additions only. Met.
* Not done, as the phase says: no `signin` package, no `auth` change, no
  contract import (Phase 4). The files are staged for the coordinator to
  commit.

### Phase 2 (2026-11-12)

Base: `main` at `4012cf4` (Phase 1 committed), tree clean. The MADR and this
PLAN were read in full first. No claim of the MADR was found false; no
existing test, golden or gate changed. No `authcontract` import: the four
values are plain strings in the root (R2), as in Phase 1.

* Step 1, `tokenstore.go`: `type SessionKind string` and the four constants
  (`SessionChatGPT` `chatgpt`, `SessionChatGPTPlan` `chatgpt_plan`,
  `SessionGrok` `grok`, `SessionKiloDevice` `kilo_device`);
  `OAuthSession.Kind`; `fileRecord.Kind` with the JSON name `kind`,
  `omitempty`; `String`, `GoString`, `MarshalJSON` and `LogValue` show it
  when set.
* Step 2: the five copies (`Load`, `persistable`, `refreshState` and the
  refreshed session, `adopt`) and `Save` carry it. `oauthSessionState` gains
  `kind` (resolved) as the step says, and one unexported field the step does
  not name, `declared` (the session's `Kind` alone), which the refreshed
  session keeps, so that a refresh never writes an inferred kind into an
  older record.
* Step 3, `oauth_session.go`: unexported `inferKind(provider, issuer,
  resource)`, `sessionKindOf(kind, provider, issuer, resource)`, `kindLocked`
  (the session's kind, for a caller holding its lock), `openAIKind`, and
  `(oauthSessionState).kindOf`, which falls back to `inferKind` for a state
  built from fields alone (`oauth_endpoints_test.go` and
  `oauth_session_test.go` build such states and are unchanged). C19
  (`ValidateOAuthSession`), C20 to C22 (`Owner`, `ChatGPT`, `PlanUsage`) and
  C23 to C25 (the refresh body, `refreshTokenURL`, revocation in
  `oauth_revoke.go`) test the kind. Exported: `(*OAuthSession).SessionKind()
  SessionKind`, under the session's lock, documented with R51's and R52's
  wording.
  * **Guards kept so that a branch is the one taken at 773b84c.** One
    inference cannot reproduce seven rules of different shapes, so three
    sites keep the issuer test beside the kind, as C24 does by D5: C19's
    access-only case needs an OpenAI kind and OpenAI's issuer; C23's JSON body
    needs the `chatgpt` kind and OpenAI's issuer (an older session with no
    issuer is refreshed as a form, as before); C24's built-in URLs need the
    kind and the issuer, or none. C25 sends a session whose kind is empty to
    the branch of its provider, as before (a session against another issuer).
  * **Behaviour that differs, none covered by an existing test:** a session
    whose `Kind` is set by a sign-in against a non-default issuer
    (`LoginBrowserOAuth`, `LoginDeviceOAuth` on `openai`) now answers
    `ChatGPT()` true, where an empty `Kind` with that issuer answers false;
    a session with a `Resource` on a provider other than OpenAI (none is
    built by this SDK) is revoked as unsupported, where it was revoked as a
    plan session. A Kilo session with `Kind` empty infers no kind (today's
    rules name none), so `SessionKind()` of a Kilo record saved before this
    release is empty. Each is for the owner to rule on.
* Step 4: `oauthSessionFromResponse` (the browser and device flows) sets
  `SessionChatGPT` or `SessionGrok` by provider, through an unexported
  `signInKind`; `LoginChatGPTPlan` sets `SessionChatGPTPlan`; the Kilo device
  sign-in sets `SessionKiloDevice`; the wizard's access-only session sets
  `SessionChatGPT` (`wizard/auth.go`, one line, named by the step though not
  among the phase's files; no wizard test reads it).
* Step 5: `ValidateOAuthSession` refuses a `chatgpt_plan` with no `Resource`
  and a `kilo_device` with a refresh token, with plain errors naming the kind
  and the contradiction, before the other cases.
* Tests, `llmprovider/auth/session_kind_0074_test.go`: the six of the table,
  and `TestSessionKind_Revocation_0074` (Kind set and empty are revoked in
  the same place and form), with two unexported helpers for lint.
* **Plants,** `python3 -B tools/plant.py` on a spec of 17, on its scratch
  clone, never in the tree, exit 0, "plant: all 17 plant(s) caught". Failure
  lines:
  * the session's kind ignores `Kind`: `--- FAIL: TestSessionKindAccessor_0074`,
    `session_kind_0074_test.go:197: chatgpt: Kind set: SessionKind() =
    "chatgpt", want it`;
  * the plan case of `inferKind` dropped: `session_kind_0074_test.go:108:
    PlanUsage() = false, want true`, `:102: Owner() = "", want "openai"`;
  * `refreshTokenURL` tests the kind alone (Grok, and OpenAI): `--- FAIL:
    TestSessionKind_EachSite_0074/grok,_another_issuer,_no_token_URL,_Kind_set`,
    `:121: refresh request = &{POST https://auth.x.ai/oauth2/token ...`; the
    OpenAI plant the same with `https://auth.openai.com/oauth/token`;
  * `SessionKind` returns `Kind` alone: `:193: chatgpt: Kind empty:
    SessionKind() = "", want the inferred "chatgpt"`;
  * `LoginChatGPTPlan` does not set the kind: `:289: LoginChatGPTPlan: Kind =
    "chatgpt", want "chatgpt_plan"` (the plan sign-in then keeps the kind
    `oauthSessionFromResponse` gave it); the Kilo and the Grok and OpenAI
    plants: `:302: Kilo device sign-in: Kind = "", want "kilo_device"`, `:277:
    OpenAI device sign-in: Kind = "", want "chatgpt"`;
  * `fileRecord`'s tag spelled `session_kind`: `:339: a record naming a kind:
    Kind "", err <nil>; want "chatgpt_plan"`; `Save` or `Load` leaving `Kind`
    out: `:329: Load: Kind "", err <nil>; want "grok"`, `:422: Save and Load:
    field Kind = , want value-Kind`;
  * `Kind` left out of `adopt`: `:439: adopt: field Kind = , want value-Kind`;
    of `persistable`: `:423: persistable: field Kind = , want value-Kind`; of
    the refreshed session: `:435: refreshState to the refreshed session: field
    Kind = , want value-Kind`;
  * the contradiction checks removed: `:458: a chatgpt_plan without a resource:
    err = <nil>, want a refusal naming chatgpt_plan and the contradiction`; and
    for Kilo, the error was the older "no token URL" one, not the refusal;
  * revocation sends a `chatgpt` session to the wrong branch: `--- FAIL:
    TestSessionKind_Revocation_0074/chatgpt`, `:177: explicit true: err =
    oauth: revoke: openai sessions have no revocation: ... unsupported`.
  The plants were run again after the lint split of the first test, with the
  same result.
* Checks, each exit status captured before any filter, the heavy ones through
  `run-heavy`:
  * `go test -count=1 -race -run 0074 ./llmprovider/auth`, exit 0; the
    existing `./llmprovider/auth`, `./wizard` and `./llmprovider/providers/...`
    packages, exit 0, before the new tests.
  * `python3 -B tools/redgreen.py REFRESH.json --expect pass` (a spec built
    from the PLAN's 66 names): 65 pass, exit 1, one "missing":
    `TestOAuthSessionWait_WindowsRemoval_0071`, which skips on a host that is
    not Windows (`runtime.GOOS` check, line 140), so the tool sees no PASS
    line. Its file is unchanged and the Windows vet compiles. `git diff
    --stat` shows none of the 15 files changed. Not a failure of this phase;
    the tool cannot see that test pass off Windows.
  * `make pre-add-check FILES="..."` on the nine Go files, exit 0,
    "go-precheck: 9 file(s) clean". **First run, exit 2, two issues:**
    `gocognit` on `TestSessionKind_EachSite_0074` (58 of 50), fixed by
    splitting two helpers out of it; `gocritic typeDefFirst` on the
    `kindOf` method placed before `oauthSessionState`, fixed by moving it after
    the type. No `nolint`.
  * `make api-check`, exit 0: "against v1.30.0, 0 incompatible change(s)
    outside llmprovider/x/". `python3 -B tools/apidiff.py v1.30.0`, exit 0:
    Phase 1's nine `AuthMethod` fields and D5's seven additions, nothing
    else: `auth.SessionKind`, `SessionChatGPT`, `SessionChatGPTPlan`,
    `SessionGrok`, `SessionKiloDevice`, `OAuthSession.Kind`,
    `(*OAuthSession).SessionKind`.
  * `python3 -B tools/gate.py --timeout 5400`, through `run-heavy`. **First
    run, exit 1, "28 of 30 passed":** `docs-check` rc=2, "llmprovider/auth/tokenstore.go:87
    R51 type SessionKind does not say whether it is safe for concurrent use"
    (sdkaudit, 1 violation), and `gate-selftest` rc=2, which runs the same
    audit. Fixed by the type's doc sentence, as `authcontract.SessionKind`
    has it; `make docs-check` then exit 0, `make gate-selftest` exit 0 (68
    tests). **Second run, exit 0, 4048 s including the queue: "gate: 30 of 30
    passed",** every check rc=0. The full-module test runs were the gate's.
  * The gate ran before this entry was written, so the record checks were run
    again after it, each exit 0 (below).
* Acceptance: the six tests pass, each plant failed, the 66 refresh tests
  pass (65 on this host, one Windows-only, files unchanged). Met.
* Not done, as the phase says: no `signin` package, no `llmprovider/x`, no
  contract import, no change to the wizard's text.

### Phase 2, deviations (2026-11-12)

The owner ruled on three findings of the Phase 2 entry above, on
2026-11-12. The entry's "Behaviour that differs" bullet is superseded by
this one.

* **Finding 1a, ruled: keep the issuer guard in `ChatGPT()`.** `ChatGPT()`
  is true only when the kind is `SessionChatGPT` and the issuer, trimmed of
  its trailing slash, is `DefaultOpenAIIssuer` or empty, the guard D5 keeps at
  C24. A session with `Kind` set and the same session with `Kind` empty then
  take the same branch, and the doc comment ("issued by the default OpenAI
  issuer") stays true. `TestSessionKind_EachSite_0074` gains the case "openai,
  another issuer" with `Kind` set and empty, both answering `ChatGPT()`
  false. No MADR change.
* **`Owner()`, checked for the same divergence: it had it,** so the same
  guard is applied. With `Provider` empty and an issuer that is not a
  built-in one, `Kind` set answered `openai` (or `grok`) where `Kind` empty
  answered "". `Owner()` now names OpenAI for an OpenAI kind only with
  OpenAI's issuer, and Grok for `SessionGrok` only with xAI's. Two cases
  ("chatgpt, no provider, another issuer" and "grok, no provider, another
  issuer") assert "" for both.
* **Plants** (`python3 -B tools/plant.py`, scratch clone, never the tree;
  exit 0, "plant: all 20 plant(s) caught", the 17 of the entry and three
  new):
  * the issuer guard dropped from `ChatGPT()`: `--- FAIL:
    TestSessionKind_EachSite_0074/openai,_another_issuer,_Kind_set`,
    `session_kind_0074_test.go:119: ChatGPT() = true, want false`;
  * the guard dropped from `Owner()` for an OpenAI kind: `.../chatgpt,_no_provider,_another_issuer,_Kind_set`,
    `:119: Owner() = "openai", want ""`;
  * the guard dropped from `Owner()` for the Grok kind: `.../grok,_no_provider,_another_issuer,_Kind_set`,
    `:119: Owner() = "grok", want ""`.
* **Finding 1c, ruled: a Kilo record saved before v1.31.0 means one fresh
  sign-in.** No code change: Kilo gets no inference rule, and D5's "exactly
  today's rules" stands. Its `SessionKind()` is "". Recorded in the MADR as a
  dated annotation to D5, naming gobble-cli 0051-MADR D2's resolver.
* **Finding 1b accepted with no change:** a session with a `Resource` on a
  provider other than OpenAI is revoked as unsupported; this SDK builds none.

### Phase 3 (2026-11-12)

Base: `main` at `906e365` (Phase 2 committed as `5d08770`), tree clean. The
MADR (D4, D7, D8, D10) and this PLAN were read first. Step 0: `auth.Identity`
exists (`identity.go`), and the OpenAI device poll's own deadline now returns
the pacer's `errDeviceCodeExpired` (`oauth_device.go`, `pace.expired()`), so
both prerequisites hold; `v1.30.0` is tagged.

* Step 1, `oauth_loopback.go`, `oauth_plan.go`: one unexported
  `runBrowserLogin(ctx, browserLogin)` holds the listener, result channel,
  input goroutine and select. `browserLogin` carries the callback parser, the
  CORS origin, the authorize-URL builder (so Grok's nonce is still made after
  the listener binds) and the exchange. `LoginBrowserOAuth` and
  `LoginChatGPTPlan` call it. The browser's parser is `parseOAuthInput`
  wrapped, the plan's `parseOAuthCallback`, so each keeps its own. The plan's
  constant `planCallbackPath` is gone: both read `openAICallbackPath`, a new
  constant for the same string, which `browserListener` also uses.
* Step 2: `OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback`,
  carried as `oauthFlowConfig.noLoopback`. `requireNoLoopbackInput` runs after
  config resolution and before any request: Grok is `ErrUnsupported`, a
  sign-in without `InputCode` is `ErrInvalidRequest`. `browserEndpoint` binds
  nothing with it set and names `openaiLoopbackPorts[0]`. `StartDeviceOAuth`
  does not read it (`TestNoLoopback_DeviceIgnored_0074`, added).
* Step 3: `ErrSignInDenied` and `ErrSignInExpired` (each wrapping
  `ErrAuthFailure`) and an unexported `signInError` whose `Error()` is the old
  error's and whose `Unwrap() []error` adds the sentinel. The six sites (C42
  to C47: the browser `access_denied`, Grok's denial and expiry, the shared
  `errDeviceCodeExpired`, Kilo's denial and expiry) use `asSignInError`.
* Step 4: `auth.VendorCLIPath` in the new `vendor_path.go`; `wizard/import.go`
  is now `vendorAuthPath` calling it, and `homeDir` is deleted. **Error text
  differs** (not a wizard notice, prompt or question): the three errors read
  "auth: ..." where they read "wizard: ...", and the nil case says "pass a
  lookupEnv, such as os.Getenv". No wizard test reads them.
* Step 5, first half: `TestLive_GrokDeviceLogin` and `TestLive_KiloDeviceLogin`
  call `StartDeviceOAuth`, log the code, then `Wait`; `go vet -tags
  live_gateways` builds them (the gate's `vet-*-live` runs).
* **Deviation, step 5's second half not done (2026-11-12).** The `Deprecated:`
  paragraphs on `LoginDeviceOAuth` and `OAuthFlowOptions.NotifyDevice` were
  written, and `make pre-add-check` failed, exit 2, three staticcheck SA1019
  findings at the only two callers outside package `auth`:
  `llmprovider/live_0076_identity_test.go:66` (`auth.LoginDeviceOAuth`) and
  `:69` (`NotifyDevice`), and `wizard/configure_errors_0048_test.go:157`
  (`auth.LoginDeviceOAuth`). Neither file is among Phase 3's, and the PLAN's
  rule is that no other existing test changes. The markers were taken out
  again to await the owner's ruling, which came the same day; see "Phase 3,
  deviations" below.
* **Tests** (new, in `llmprovider/auth`): `no_loopback_0074_test.go`
  (`TestNoLoopback_OpenAI_0074`, `_Plan_0074`, `_GrokUnsupported_0074`,
  `_DeviceIgnored_0074`), `signin_messages_0074_test.go`
  (`TestSignInMessagesUnchanged_0074`, which owns the six-site table),
  `signin_sentinels_0074_test.go` (`TestSignInSentinels_0074`, with
  `_OpenAIDeadline_0074`, which pins 0076's expiry as `ErrSignInExpired`, and
  `_Distinct_0074`), `vendor_path_0074_test.go` (`TestVendorCLIPath_0074`).
  The existing browser, loopback, plan and callback tests pass unchanged.
* **Red, then green.** `python3 -B tools/redgreen.py --expect pass --rev
  773b84c` on `TestSignInMessagesUnchanged_0074`, exit 0, "1 of 1 pass as
  expected": its literals are the old code's.
* **Plants,** `python3 -B tools/plant.py`, scratch clone, never the tree, exit
  0, "plant: all 16 plant(s) caught", run twice (the second after the lint
  fixes below). Every one failed on an assertion, none on a build:
  * the listener still bound: `--- FAIL: TestNoLoopback_OpenAI_0074`,
    `no_loopback_0074_test.go:115: LoginBrowserOAuth = OAuthSession(nil),
    oauth: registered callback ports unavailable; use device-code login:
    listen tcp4 127.0.0.1:25201: bind: address already in use` (the plan test
    the same at `:157`);
  * `InputCode` not required: `:135: LoginBrowserOAuth = OAuthSession(nil),
    oauth: browser login timed out: context deadline exceeded; want
    ErrInvalidRequest` (`:176` for the plan);
  * the redirect names the second port: `:119: authorize redirect_uri =
    "http://127.0.0.1:22168/auth/callback", want
    "http://127.0.0.1:22143/auth/callback"`;
  * the plan ignores `NoLoopback`: `--- FAIL: TestNoLoopback_Plan_0074/a_paste_completes_it`,
    `:157: ... address already in use`;
  * Grok not refused: `--- FAIL: TestNoLoopback_GrokUnsupported_0074`, `:201:
    InputCode true: LoginBrowserOAuth = OAuthSession(nil), oauth: token
    request: Post "https://auth.x.ai/oauth2/token": no request expected; want
    ErrUnsupported`;
  * each of the six sites given the other sentinel, or none (seven plants,
    the shared expiry one failing two tests): `--- FAIL:
    TestSignInSentinels_0074/browser_denial`, `signin_sentinels_0074_test.go:33:
    err = llmprovider: authentication failed: oauth: authorization failed:
    access_denied; matches llmprovider: authentication failed: sign-in
    expired too`; Kilo's expiry unwrapped: `.../Kilo_expiry`, `:33: err = ...
    Kilo device code expired; want it to match ErrAuthFailure and ... sign-in
    expired`; the shared expiry as denied also fails
    `TestSignInSentinels_OpenAIDeadline_0074`, `:43`;
  * a message gains the sentinel's text: `--- FAIL:
    TestSignInMessagesUnchanged_0074/shared_device_expiry`,
    `signin_messages_0074_test.go:108: Error() = "llmprovider: authentication
    failed: oauth: device code expired: llmprovider: authentication failed:
    sign-in expired", want "llmprovider: authentication failed: oauth: device
    code expired"`;
  * `GROK_AUTH_PATH` read after `GROK_HOME`: `--- FAIL:
    TestVendorCLIPath_0074/grok_auth_path_before_home`,
    `vendor_path_0074_test.go:40: vendorCLIPath = "/x/grok/auth.json", <nil>;
    want "/y/login.json"`; and three of mine: Windows reading `HOME`
    (`.../codex_default_on_windows`), Kilo not `ErrUnsupported`
    (`.../Kilo_has_no_vendor_CLI`), a nil `lookupEnv` finding a home
    (`.../a_nil_lookupEnv_finds_no_home`, `VendorCLIPath(nil) =
    "/nowhere/.codex/auth.json", <nil>; want an error`).
* Checks, each exit status captured before any filter, the heavy ones through
  `run-heavy`:
  * `make pre-add-check` on the eleven Go files: **first run, exit 2:**
    errorlint twice in the new sentinels test (`==` and `!=` on errors),
    revive `confusing-naming` (`vendorCLIPath` beside `VendorCLIPath`), and
    the three SA1019 findings of the deviation. Fixed by `errors.Is`, by
    renaming the helper `vendorPathOn`, and by the deviation above; no
    `nolint`. Second run, exit 0, "go-precheck: 11 file(s) clean".
  * `python3 -B tools/redgreen.py REFRESH.json --expect pass`: 65 of 66, exit 1,
    one "missing": `TestOAuthSessionWait_WindowsRemoval_0071`, which skips off
    Windows, as in Phase 2; none of the 15 files changed.
  * `make api-check`, exit 0: "against v1.30.0, 0 incompatible change(s)
    outside llmprovider/x/". `python3 -B tools/apidiff.py v1.30.0`, exit 0,
    all "Compatible changes": the nine `AuthMethod` fields and seven session
    additions of Phases 1 and 2, and this phase's five, `auth.ErrSignInDenied`,
    `auth.ErrSignInExpired`, `auth.VendorCLIPath`,
    `auth.OAuthFlowOptions.NoLoopback`, `auth.ChatGPTPlanOptions.NoLoopback`;
    nothing else, so R55's table is met.
  * `python3 -B tools/gate.py --timeout 5400`, through `run-heavy`, after the
    last code change, exit 0: "gate: 30 of 30 passed", every check rc=0 (the
    coverage floors included). The record checks were run again after this
    entry.
* Acceptance: the tests pass and each plant failed; the loopback, plan and
  browser-kind tests pass unchanged; the refresh tests pass; `make api-check`
  lists the five names as additions only. Met, with the deviation below: the
  `Deprecated:` markers are in.
* Not done, as the phase says: no `signin` package, no `llmprovider/x`, no
  contract import, no change to the wizard's text.

### Phase 3, deviations (2026-11-12)

* **Finding.** With step 5's `Deprecated:` paragraphs on `LoginDeviceOAuth`
  and `OAuthFlowOptions.NotifyDevice`, `make pre-add-check` reported three
  staticcheck SA1019 findings at the only callers outside package `auth`:
  `llmprovider/live_0076_identity_test.go:66` (`LoginDeviceOAuth`) and `:69`
  (`NotifyDevice`), and `wizard/configure_errors_0048_test.go:157`
  (`LoginDeviceOAuth`). Both files were outside Phase 3's scope, and the
  rules said that any other test that needs a change is a stop. The markers
  were held back (the entry above).
* **Ruling, 2026-11-12, the owner: "Add the 2 tests to Phase 3".** The two
  files join Phase 3's scope. Their three call sites move to
  `StartDeviceOAuth` and `Wait`, and the `Deprecated:` paragraphs are
  restored. The 0076 live test keeps checking the device form's `referrer`;
  the wizard test keeps its assertions. The "no other existing test changes"
  rule is annotated above, not rewritten.
* **Applied (2026-11-12), after the record above.**
  * `llmprovider/live_0076_identity_test.go`, the `GrokDevice` half: `StartDeviceOAuth`,
    the code logged as the `NotifyDevice` callback did, then `Wait`; on a
    `StartDeviceOAuth` error the half still reaches `checkDeviceReferrer` and
    `probeSignIn` as before. `checkDeviceReferrer` still runs on the same
    recorder and name. `wizard/configure_errors_0048_test.go`,
    `TestConfigureLLM_DeviceLoginErrorBare_0048`: `StartDeviceOAuth`, then
    `Wait` when it starts; every assertion is unchanged.
  * `Deprecated:` paragraphs restored on `LoginDeviceOAuth` and
    `OAuthFlowOptions.NotifyDevice`. `make pre-add-check` on the thirteen Go
    files, exit 0, "go-precheck: 13 file(s) clean"; no `nolint`.
  * Callers of the two outside package `auth`, `grep -rnE
    "LoginDeviceOAuth|NotifyDevice" --include='*.go'` (exit 0) with the
    package's own files removed (exit 0): one line, the comment at
    `wizard/auth.go:722` ("It sets no NotifyDevice: ..."), no call. Docs and
    the CHANGELOG name `LoginDeviceOAuth` in prose only.
  * **The referrer check still fails when broken.** The 0076 live half cannot
    run offline; its offline twin and the shared `live_checks_test.go`
    helper can. The existing plant of 0076 (`oauth_device.go`: the device
    form's referrer kept at the default), run on the twin with `plant.py` on a
    scratch clone, exit 0, "plant: all 1 plant(s) caught": `--- FAIL:
    TestLiveTwin_Identity_0076/GrokDevice`, `live_twins_test.go:321: device
    form referrers = [["go-llmprovider-sdk"]], want one form with
    ["llmprovider-probe"]`. The auth package's own twin of it,
    `TestIdentity_Referrer_0076`, is the plant of 0076 Phase 2 and was not
    changed. The 16 plants of this phase were run again, exit 0, "plant: all
    16 plant(s) caught".
  * `make api-check`, exit 0, "0 incompatible change(s) outside
    llmprovider/x/"; `tools/apidiff.py v1.30.0`, exit 0, the same 21 additions.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after the
    last code change, exit 0: "gate: 30 of 30 passed". The record checks were
    run again after this entry.

### Amendment (2026-11-13): the owner's ruling on Grok's paste

* **The ruling, 2026-11-13, the owner: "Probe Grok paste; keep 0247".** A
  live Grok browser sign-in completed by a paste, from a browser that cannot
  reach the host's loopback, is added to the owner-run Phase 7. On a pass,
  Grok's `Paste` fact is set and D4's `NoLoopback` refusal for Grok lifts.
  On a fail, Grok's device code stays the path from a phone, and nothing
  else changes. magic-cli-remote's
  `docs/decisions/0247-MADR-replay-a-pasted-sign-in-callback-to-the-agents-loopback.md`
  is kept.
* **Applied, in the records only:**
  * the MADR gains "Amendment 2026-11-13: a live probe of Grok's paste, and
    what its outcome changes", with dated notes in D1, D3, D4 and D12, and
    claims C56 to C67;
  * Phase 7 step 4, which already named `TestLive_0074GrokPaste` and asked
    only what accounts.x.ai shows, is amended in place to complete the
    sign-in by the paste. Its route, arrangement, paste file, `FINDING`
    lines, checks, twin and plants are written there. It is not a second
    step;
  * Phase 8 gains step 1, run only on a pass;
  * "In scope" and "Rules for every phase" gain dated notes;
  * claims P7 to P12 are added.
* **Not done.** No code. The probe has not run. No other record is changed:
  [0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md](0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md)
  Phase 5's Grok half probes the name on a Grok session's refresh and
  revocation. Those do not depend on how the session was signed in, so its
  device sign-in needs no paste path. 0008-MADR's note waits for the
  probe's outcome.

### The owner's rulings on the probe (2026-11-13)

The owner answered the four questions of the amendment above on 2026-11-13.
The MADR holds each question with its three options as put ("The owner's
rulings of 2026-11-13 on the probe").

1. **Where a pass's code lands: "In 0074, before the tag".** It is Phase 8
   step 1, in its own commit. That step gains a dated note.
2. **When: "Now, before Phase 4".** Phase 7 step 4's test files become the
   new Phase 3a, which a separate code agent executes next. On a pass,
   Phase 8 step 1 runs after Phase 3a and before Phase 4. Phase 7 step 4,
   Phase 8 step 1 and the scope gain dated notes.
3. **Device: "Phone + send-to-self".** The SSH arrangement of Phase 7
   step 4 is superseded, and kept as written with a dated note:
   * the test runs on the owner's Mac and prints the authorize URL;
   * the owner sends the URL to the phone by any channel, signs in there,
     and copies what the page shows;
   * the owner sends that text back to the Mac by any channel, and writes
     it into the paste file from a terminal on the Mac;
   * there is no SSH on the phone.

   The `FINDING page:` lines stay, with the device class recorded as
   "phone".
4. **Bare code: "url_or_code".** This is the value the amendment already
   names.

Records only: no code, no commit, and the probe has not run.

### Phase 3a (2026-11-13)

Executed up to the commit. *(2026-11-13, later: the owner's run is done, a
pass; see "Phase 3a, the owner's run (2026-11-13)", at the end of this
record.)*

* **Files, and nothing else.** `llmprovider/live_0074_grok_paste_test.go`
  (new, build tag `live_gateways`: `TestLive_0074GrokPaste`);
  `llmprovider/live_checks_test.go` (the imports `fmt`, `net` and `os`, and
  appended: `pasteFileLimit`, `pasteInterval`, `pasteRecord`,
  `pasteFromFile`, `readPasteFile`, `pasteShape`, `pasteFinding`,
  `pasteRefusedLocally`); `llmprovider/live_twins_test.go` (the imports
  `errors`, `os` and `path/filepath`, and appended:
  `TestLiveTwin_GrokPaste_0074` with five subtests). No existing function
  changed. No production code, and no seam: the route is
  `auth.LoginBrowserOAuth` as shipped.
* **The paste file.** Polls every `pasteInterval` (one second; the twin sets
  10 ms) for a newline-terminated line; reads at most 8 KiB (a file with no
  newline in 8 KiB is removed and is an error); removes the file after the
  read; records the shape only; returns `ctx`'s error when the sign-in ends
  first. Nothing logs the value and no error holds it.
* **Two small additions to what Phase 7 step 4 lists.** `pasteShape` also
  names an empty line and the URL shapes "with a code and no state", "with a
  state and no code", "with an error" and "with no code and no state";
  `pasteFinding` has a sixth line, `FINDING sign-in: the sign-in ended
  before any paste: <error>`, for an error that is neither a deadline nor a
  refused paste (a listener failure, an unreadable paste file). It decides
  nothing. The test stops, as `FailNow`, on any line but the pass.
  `pasteFinding` tells "refused before any request" from "xAI refused" by
  `auth.ErrSignInDenied` and the texts "callback state mismatch", "missing
  authorization code" and "empty authorization code"; a rewording of those
  in `auth` fails the twin's table.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, exit 0, "plant:
  all 5 plant(s) caught"; the tree was not touched:
  1. `pasteFinding` reports a paste when `InputCode` returned nothing:
     `live_twins_test.go:511: loopback won: pasteFinding = "FINDING sign-in:
     completed by a paste of ", true; want "FINDING sign-in: completed by the
     loopback, not a paste; ..."`
  2. `pasteShape` calls a URL with a code and no state "a callback URL with
     code and state": `live_twins_test.go:468: pasteShape = "a callback URL
     with code and state (scheme http, a loopback host, path /callback)",
     want "a URL with a code and no state (scheme http, a loopback host, ..."`
  3. `pasteShape` writes the pasted value into its shape:
     `live_twins_test.go:468: pasteShape = "a bare code PLANTEDCODE-7f3a91",
     want "a bare code"` and `:471: pasteShape holds the pasted value: ...`
  4. `pasteFromFile` leaves the file in place: `live_twins_test.go:403: the
     paste file is still there after the read: <nil>`
  5. `pasteFromFile` takes a line before its newline:
     `live_twins_test.go:381: pasteFromFile returned "first-half", <nil>
     before the newline was written`
* **Checks.** `make pre-add-check FILES="..."` on the three files, exit 0,
  "go-precheck: 3 file(s) clean" (a first run failed `gocognit` on the twin,
  complexity 55 over 50; fixed by splitting its subtests into functions, no
  `nolint`; plants run again after). `go vet -tags live_gateways
  ./llmprovider/`, exit 0. `make tables-check`, exit 0, "check-tables: 42
  packages, 0 problem(s)". `go test -count=1 -run TestLiveTwin
  ./llmprovider/`, exit 0.
* **The gate:** `python3 -B tools/gate.py --timeout 5400` through
  `run-heavy`, after the last code change, exit 0: "gate: 30 of 30 passed".

#### For the owner: the run

From the repository root on the Mac:

```bash
export LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE="$(mktemp -d)/paste"
echo "$LLMPROVIDER_LIVE_0074_GROK_PASTE_FILE"
LLMPROVIDER_LIVE_0074=1 LLMPROVIDER_LIVE_BROWSER_LOGIN=1 \
  go test -tags live_gateways -count=1 -timeout 15m -run '^TestLive_0074GrokPaste$' -v ./llmprovider/
```

1. The test logs the authorize URL. Send it to the phone by any channel
   (send-to-self), and open it in the phone's browser, which cannot reach the
   Mac's `127.0.0.1`. Sign in and consent.
2. Note the page's text, and copy what it shows. Record the four `FINDING
   page:` lines (what the page said, what the address bar held, any
   local-network prompt and the answer, "phone" and the browser), quoting no
   URL, code or token.
3. Send the copied text back to the Mac by any channel.
4. In a second terminal on the Mac: `cat >` the echoed path, paste the text,
   Enter, Ctrl-D.
5. The test logs one `FINDING sign-in:` line, and on a pass `FINDING use:`
   and `FINDING revocation:`. Record every `FINDING` line here.

On a pass, Phase 8 step 1 runs next, before Phase 4. On a fail, no code
changes. On a run that decides nothing, rerun.

#### Phase 3a, the owner's run (2026-11-13)

The owner ran `TestLive_0074GrokPaste` twice on 2026-11-13, on the Mac, with
the arrangement of ruling 3 (phone, send-to-self). This is Phase 3a's run
record and Phase 7 step 4's outcome.

**Run 1 decided nothing.** It ended after 7.73 s:

```text
live_0074_grok_paste_test.go:55: FINDING sign-in: completed by the loopback, not a paste; the browser reached the host, so this run decides nothing
--- FAIL: TestLive_0074GrokPaste (7.73s)
```

The cause, as the coordinator reads it: a browser on the host reached the
listener. `openAuthorizeURL` only logs the URL
(`llmprovider/live_oauth_login_test.go:26`), so the link was most likely
clicked in the host's terminal. A phone cannot reach the host's `127.0.0.1`.
The test revoked its session at cleanup.

**Run 2 passed.** Its output, verbatim:

```text
live_0074_grok_paste_test.go:55: FINDING sign-in: completed by a paste of a bare code
live_0074_grok_paste_test.go:67: FINDING use: grok listed 13 models with the session
live_0074_grok_paste_test.go:68: FINDING revocation: revoked; a refresh afterwards is refused, as ErrAuthFailure true
--- PASS: TestLive_0074GrokPaste (211.72s)
PASS
```

The paste went into the paste file with `{ pbpaste; echo; } > <path>`, where
`<path>` is the paste file the run echoed. The path is not recorded here.

**The owner's page findings,** in the owner's words: "the page opened fine
in chrome, asked for permission, then provided the code. copied code,
emailed it back to myself on the mac, and did the pbpaste command."

* `FINDING page:` what the page said: it asked for permission, then showed
  a code.
* `FINDING page:` the address bar: not observed.
* `FINDING page:` a local-network prompt: not reported. The owner's "asked
  for permission" reads as xAI's consent page; that is an interpretation,
  not something the owner said.
* `FINDING page:` device class and browser: phone, Chrome.

**The outcome: a pass.** A completion by a paste of a bare code, from a
phone that cannot reach the host's loopback, then a listing and a revoked
session whose refresh is refused. By the owner's rulings:

* Phase 8 step 1 runs next, before Phase 4, in its own commit.
* Phase 4 writes its Grok rows as `browser_paste` and `url_or_code`, once.
* The MADR's D1 and D4 Grok bullets and its "Not verified" list carry a
  dated note: verified 2026-11-13, by Phase 3a.

No code changed in this record.

### Phase 8, step 1 (2026-11-13)

Run on the pass of Phase 3a's probe, before Phase 4, from `3e16417`. Steps
3 (`x/signin`, which Phase 4 has not built) and the README (it names no
loopback-only Grok method) did not apply.

* **Code.** `llmprovider/providers/grok/descriptor.go`: Grok's
  `browser_oauth` is `Completion` `browser_paste`, `Paste` `url_or_code`,
  with no `FirstPaste`. `llmprovider/auth/oauth_loopback.go`:
  `requireNoLoopbackInput` lost its Grok case; `browserEndpoint` with no
  listener names `http://127.0.0.1:<port>/callback` for Grok, the port from
  `crypto/rand` (`grokNoLoopbackPort`; `math/rand/v2` failed `gosec` G404) in 49152 to 65535 (constants `grokNoLoopbackPortMin` and
  `grokNoLoopbackPortMax`), and never bound; OpenAI's is unchanged. The doc
  comments of `OAuthFlowOptions.NoLoopback`, `LoginBrowserOAuth`,
  `requireNoLoopbackInput` and `browserEndpoint` say so. No exported name
  changed.
* **Tests.** The Grok row of `TestDescriptors_OpenAIAndGrokOfferOAuth`;
  `TestNoLoopback_GrokUnsupported_0074` replaced by `TestNoLoopback_Grok_0074`
  (a paste completes it against a fake issuer, with the redirect's scheme,
  host, path and port range asserted and the token form carrying the same
  redirect; without an `InputCode`, `ErrInvalidRequest` before any request).
* **Plants,** `tools/plant.py` on a scratch clone, "plant: all 3 plant(s)
  caught", the tree untouched: the fact left `browser_loopback` fails the
  descriptor test; Grok still refused fails `TestNoLoopback_Grok_0074` (both
  subtests, "unsupported: refused"); OpenAI's port and path named for Grok
  fails it (the authorize `redirect_uri` was OpenAI's, port 1455, path
  `/auth/callback`).
* **Records.** `CHANGELOG.md` `[Unreleased]` **Changed**; a dated note in
  0008-MADR's 2026-11-12 amendment; dated notes on the MADR's claim rows
  C56, C57, C60 and C62, whose quoted lines changed (each row holds at its
  pinned revision).
* **Checks.** `make pre-add-check` on the four Go files, exit 0 (a first run
  failed `gosec` G404 on `math/rand/v2`; fixed, plants run again after).
  `make api-check`, exit 0, 0 incompatible; `tools/apidiff.py v1.30.0`,
  exit 0, no name from this step.
* **The gate:** `python3 -B tools/gate.py --timeout 5400` through
  `run-heavy`, after the last code change, exit 0: "gate: 30 of 30 passed"
  (started 12:36, finished 12:54 local, 1085 s).

### Phase 4, deviations (2026-11-13)

Found while executing Phase 4, before any commit. Each was put to the owner,
who ruled the same day.

* **Deviation 1: `make api-check` breaks when the root requires the nested
  module (blocking).**
  * **Evidence.** After `go get github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0`
    (a module-proxy fetch), `make api-check` exits 2 with "16 incompatible
    change(s)": every root package is both "removed" and "added".
    `python3 -B tools/apidiff.py v1.30.0` exits 1 the same way, and two tests
    of `scripts/test_gates.py` fail (`test_api_check_incompatible_change`,
    `test_clean_copy_passes_every_gate` for gate `api-check`). It did not
    happen before: the nested module was not in the root's build list.
  * **Cause.** The pinned apidiff's `loadModule` loads `<module>/...` and takes
    `Module.Path` from `loaded[0]`. With the requirement, that pattern also
    matches the nested module's packages; `authcontract` sorts first, so the
    export is filed under `.../authcontract`, and nothing lines up with the
    v1.30.0 export. The fix lies in `scripts/check_api.py` (the `-w` and
    compare calls) and `tools/apidiff.py`, which Phase 4's file list does not
    name, and neither 0073 nor 0074 foresaw it.
  * **Resolution, the owner's ruling "Run apidiff per subtree".**
    `scripts/check_api.py` and `tools/apidiff.py` run apidiff once for
    `MODULE/llmprovider` and once for `MODULE/wizard` (so the pattern is
    `MODULE/llmprovider/...` and `MODULE/wizard/...`), and merge the two
    reports. The export header stays the root module's path, as checked by
    hand: `-incompatible` against v1.30.0 prints nothing for either subtree.
    Their tests in `scripts/test_gates.py` and `tools/test_tools.py` change to
    match, each new or changed check gets a planted break on a scratch clone,
    and `gate-selftest` and `tools-test` must pass. Both scripts and both
    test files join Phase 4's scope. The rejected options: a patched or newly
    pinned apidiff (a fork or a pin bump), and leaving the gate red.
  * **If every package were moved out of those two subtrees** the check would
    miss them; the scripts therefore also fail when `go list ./...` names a
    root-module package outside `llmprovider/` and `wizard/` that is not
    `internal/`.
* **Deviation 2: the wizard-row plant moves to Phase 5.** The PLAN's plant
  "`make tables-check` with `authcontract` removed from `wizard`'s R2 row" is
  not caught in Phase 4 (exit 0), because `wizard` imports nothing of the
  contract until Phase 5 (step 1). The owner ruled "Accept both". The plant
  runs on the rows that import the module now: `authcontract` removed from
  `llmprovider/x/signin`'s R2 row, from `internal/signinflow`'s R2 row, and
  from `x/signin`'s row of `docs/architecture.md`; each fails. The wizard-row
  plant is run in Phase 5, as that phase's table now says. For the same
  reason `docs/architecture.md`'s `wizard` row does not name `authcontract`
  yet (`tables-check` fails a name the package does not import); R2's `wizard`
  row does, as the PLAN says.
* **Deviation 3: the deadline plant is replaced.** The PLAN's plant for
  `TestErrorKindOf_0074`, "the poll's deadline returning 0076's plain error
  again (MADR C51)", is the loop-top `pace.expired()` in
  `llmprovider/auth/oauth_device.go`. It is not caught from `x/signin`: with
  the real clock or a `testing/synctest` bubble the deadline is always reached
  through `sleepBeforeDeadline`, and only the stepped clock inside `auth`
  reaches the loop-top check (its own test, `TestOpenAIDeviceExpiryKind_0076`,
  does). The owner ruled "Accept both": the plant is the shared expiry
  `errDeviceCodeExpired` losing its sentinel (a plain `errors.New`, which maps
  to `other`, as C51 did). It is caught by the OpenAI deadline subtest of
  `TestErrorKindOf_DeviceFailures_0074`, which runs the real poll to its
  15-minute deadline on a synctest bubble: `ErrorKindOf(oauth: device code
  expired) = "other"; want "expired"`.

### Phase 4 (2026-11-13)

Base: `main` at `5030a7d` (Phase 8 step 1 committed), tree clean. The MADR (D2,
D3, D4, D8, the amendment of 2026-11-13 and the owner's rulings on the probe)
and this PLAN were read first. Grok's browser rows are written once, as
`browser_paste` and `url_or_code`. Deviations 1 to 3 are in "Phase 4,
deviations" above, each ruled by the owner on 2026-11-13.

* **Step 1, `go.mod` and `go.sum`.** The root requires
  `github.com/maccavelli/go-llmprovider-sdk/authcontract v0.1.0`, no
  `replace`. `go get` fetched it from the module proxy (a network fetch, as
  expected). `go mod tidy -diff`, exit 0.
* **Step 2, `scripts/check_deps.py`.** `REQUIRES` gains the contract module.
* **Step 3, `scripts/check_tables.py`.** `imports()` also lists the packages
  of each nested module (`go -C <dir> list`), and returns which paths are
  nested; a row may name `authcontract`, and the nested packages need no row
  of their own, so the architecture prose ("the module has no row in either")
  holds.
* **Step 4, R2 and the architecture.** R2's sentence, the rows for
  `llmprovider/x/signin` and `internal/signinflow`, and `authcontract` in
  `wizard`'s R2 row. `docs/architecture.md`: the same two rows (listing what
  each package imports: `x/signin` does not import `internal/redact`, so its
  architecture row omits it, while R2's "may import" row keeps it as the PLAN
  gives it), the tree, the graph, the dependency rule, and the `x/` bullet.
  `wizard`'s architecture row does not name `authcontract` yet (Deviation 2).
* **Step 5, `internal/signinflow/`.** `signinflow.go` (`Start`, `Spec`, `Run`,
  `Logins`, `DeviceFlow`, `Result`), `paste.go`, `errorkind.go`. The engine
  takes the browser, plan and device-start functions as values (nil takes
  `auth`'s), emits steps in order with `Seq`, and refuses before any request
  what nothing could complete. A browser sign-in's `accepts` and `paste` step
  exist only when the method's `Paste` (or `FirstPaste` for a first sign-in)
  is set and `NoPaste` is not. A pasted value is checked before it reaches
  `auth`: state, code, and `client_id` for `url_only`; three pastes, so two
  re-asks and the third bad one ends `invalid_request`; a pasted URL with this
  state and an `error` passes to `auth`, which ends it `denied`. Because
  `auth` calls `OpenURL` on its own goroutine, the engine orders the steps
  itself: the paste step waits for `open_url`. The error mapping lives here
  and `x/signin.ErrorKindOf` returns it, so the wizard (Phase 5) can use it
  without importing `x/signin`.
* **Step 6, `llmprovider/x/signin/`.** `Methods`, `Start`, `Options`,
  `SignIn`, `Credential` and its five types (`APIKey`, `Session`,
  `AccessOnly`, `DeviceKey`, `ReadThrough`), `ErrorKindOf`, and the package
  doc (experimental, R4, D9). `APIKey` and `DeviceKey` redact in `String`,
  `GoString` and `LogValue`. **Design: the facts table (the owner's ruling
  "Keep table + equality test").** `Start(p, m, o)` takes no registry, and
  the PLAN's R2 row for `x/signin` excludes the providers. So `x/signin`
  holds an unexported table of the eight pairs it drives (`driven`), with the
  facts copied from the descriptors, and
  `TestMethods_StartFactsAreTheDescriptors_0074` is its guard: each row must
  equal `Methods(descriptor)`'s method for the same pair, so a descriptor
  change that the table does not follow fails the test. Cost: two copies of
  nine rows. `Start` returns `ErrUnsupported` for any other pair.
* **Step 7, `ExampleStart`,** with `ExampleMethods` and `ExampleErrorKindOf`:
  Kilo against an `httptest` server, about 3 s of real polling.
* **Tests.** Every row of the PLAN's table is present, as
  `_0074` tests in `llmprovider/x/signin` and `internal/signinflow`; the
  device tests run the real `auth` code against an in-memory issuer on
  `testing/synctest`, so the polls and the 15-minute deadline take no real
  time. Added beyond the table: `TestStart_PlanSteps_0074`,
  `TestStart_PastedDenial_0074`, `TestStart_NoLoopbackPassedOn_0074`,
  `TestMethods_StartFactsAreTheDescriptors_0074`,
  `TestErrorKindOf_DeviceFailures_0074`, `TestCredentialYields_0074`, and the
  engine tests of `internal/signinflow` (the loopback-only method, since no
  built-in method is one now; the `NoLoopback` refusals; the paste checks).
  `TestStart_PasteIsSecret_0074` is in `internal/signinflow`, because the
  engine's log is reachable only there (`Spec.Logger`). Grok's rows of the
  browser tests are `browser_paste`/`url_or_code`.
* **No red/green run was required:** every new test calls a new name.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, never the tree.
  `plants.json`: "plant: all 28 plant(s) caught", every failure an assertion,
  none a build failure. Quotes:
  * `start_device_0074_test.go:90: openai: steps = [waiting show_code done];
    want [show_code waiting done]` (`waiting` before `show_code`);
  * `credential_0074_test.go:108: the steps channel is still open after the
    end` (channel left open);
  * `start_browser_0074_test.go:77: open_url accepts "url_or_code" under
    NoPaste; want none` (paste under `NoPaste`);
  * `signinflow_0074_test.go:98: first step = {... Kind:open_url ...
    Accepts:url_or_code ...}` (`accepts` not following the method's `Paste`);
  * `start_browser_0074_test.go:181: re-ask text = "That is not a URL. Paste
    the whole URL the browser was sent to after you signed in.
    PLANTED-PASTE-VALUE"; want a reason with no paste` (paste echoed);
  * `start_browser_0074_test.go:203: Wait = context deadline exceeded; want
    invalid_request after three bad pastes` (a fourth paste allowed);
  * `start_browser_0074_test.go:244: Wait = context deadline exceeded; want
    denied` (the denial re-asked);
  * `start_browser_0074_test.go:275: something listens at 127.0.0.1:58903;
    want no listener under NoLoopback` (`NoLoopback` not passed to `auth`;
    the plan's: `:350: plan login was given {hasInput:true noLoopback:false
    ...}`);
  * `start_device_0074_test.go:108: Start with NoLoopback = llmprovider:
    unsupported: planted; want no error for a device sign-in`, and `:118:
    StartDeviceOAuth with NoLoopback = ... unsupported: planted; want it to
    start`;
  * `start_browser_0074_test.go:372: another Seq: Answer = <nil>; want
    ErrInvalidRequest` (`Answer` ignores `Seq`), and `:373: open_url with
    accepts: Answer = <nil>; want ErrInvalidRequest` (the `open_url` taken as
    the paste);
  * `signinflow_0074_test.go:188: the engine's log holds the paste: ...
    answer.secret=false ...` (the caller's `Secret` kept);
  * `start_device_0074_test.go:154: the poll ran 10m0s after Cancel; want it
    stopped at once` (`Cancel` does not stop the poll);
  * `start_device_0074_test.go:201: quota is not rate_limited:
    ErrorKindOf(llmprovider: quota exhausted) = "rate_limited"; want "quota"`
    and `expired is not auth: ErrorKindOf(llmprovider: authentication failed:
    sign-in expired) = "auth"; want "expired"`;
  * Deviation 3's: `start_device_0074_test.go:235: ErrorKindOf(oauth: device
    code expired) = "other"; want "expired"`;
  * `credential_0074_test.go:39: APIKey output 0 shows the key:
    sk-PLANTED-0074-KEY-VALUE`;
  * `Paste not copied`, the implicit key for Ollama, a descriptor's
    `Completion` set to `browser-paste`, `auth.SessionKiloDevice` respelled,
    a fifth `SessionKind` constant, `api_key` started, and `LookupEnv`
    ignored, each caught by its test. Three of the 28 were first caught only
    by a ten-minute test timeout (a fourth paste, the denial, `Cancel`); the
    tests' waits are now bounded to 5 s and the 28 were run again.
* **Plants of the gates.** `make dep-check` with the contract's requirement
  removed from `REQUIRES`: exit 1 (`dep-check: go.mod requires
  github.com/maccavelli/go-llmprovider-sdk/authcontract`). `make tables-check`
  with `authcontract` removed from `x/signin`'s R2 row, from
  `internal/signinflow`'s R2 row, and from `x/signin`'s architecture row:
  each exit 1, naming the package and the module. The wizard-row plant is not
  caught, and moves to Phase 5 (Deviation 2).
* **Deviation 1's change.** `scripts/check_api.py`: `SUBTREES`, `compare`
  (one apidiff run per subtree, the reports one after the other) and
  `uncovered` (a root-module package outside the subtrees that is not
  internal fails the gate, exit 1). `tools/apidiff.py`: `subtrees()` (read
  from `check_api.py`, one source), `merge` (the reports by section). Tests:
  `scripts/test_gates.py` `test_api_check_wizard_subtree` and
  `test_api_check_package_outside_the_subtrees`; `tools/test_tools.py`
  `test_the_subtrees_are_read_from_check_api` and
  `test_merge_joins_the_reports_by_section`. Plants of these, on a scratch
  clone: `plants_api.json`, "plant: all 4 plant(s) caught" (a subtree left
  out: `AssertionError: 'CredAPIKey' not found in 'api-check: package(s)
  outside llmprovider that no apidiff run compares: wizard'`; the guard
  removed: `test_api_check_package_outside_the_subtrees ... AssertionError: 0
  != 1 : api-check exit 0, want 1`; the merge keeping the first report only:
  `test_api_check_wizard_subtree ... api-check exit 0, want 1`; the new
  export read from the old tree: `test_api_check_incompatible_change ...
  api-check exit 0, want 1`), and `plants_apitools.json`, "plant: all 3
  plant(s) caught" (`merge` dropping every report but the first:
  `test_merge_joins_the_reports_by_section ... AssertionError`; compatible
  changes under the incompatible header: the same test; a subtree left out of
  `SUBTREES`: `test_the_subtrees_are_read_from_check_api ... Lists differ:
  ['llmprovider'] != ['llmprovider', 'wizard']`).
* **A stray `git stash`.** During the diagnosis of Deviation 1 I ran `git
  stash -q` by mistake, and popped it at once. `git stash pop` restored every
  modified file ("Dropped refs/stash@{0}") and `git status` listed the same
  six modified files and two untracked trees as before; nothing was lost. I
  did not use it again.
* **Checks,** each exit status captured before any filter:
  * `make pre-add-check FILES=...` on the fifteen Go files and
    `scripts/check_deps.py`, through `run-heavy`, exit 0, "go-precheck: 16
    file(s) clean". The first run failed `revive` (`start` beside `Start`,
    `confusing-naming`) and `unused` (`assertClosed`); the function was renamed
    `startWith` and the helper deleted, no `nolint`.
  * `make api-check`, exit 0: "against v1.30.0, 0 incompatible change(s)
    outside llmprovider/x/". `python3 -B tools/apidiff.py v1.30.0`, exit 0,
    all "Compatible changes": the nine `AuthMethod` fields and the session
    additions of Phases 1 to 3 (21 names) and the new package
    `llmprovider/x/signin`, which is experimental (R4); Phase 4 adds no name
    outside `x/`, as R55's table says. The experimental names are
    `Methods`, `Start`, `Options` and its nine fields, `SignIn` and its
    `Method`, `Steps`, `Answer`, `Wait`, `Cancel`, `Credential`, `APIKey`,
    `Session`, `AccessOnly`, `DeviceKey`, `ReadThrough` and `ErrorKindOf`; no
    name beyond the table.
  * `make dep-check`, `tables-check`, `docs-check`, `contract-check`,
    `generate-check`: each exit 0. `make coverage-check`: `internal/signinflow`
    88.9 %, `llmprovider/x/signin` 96.1 %, against 80.0 %.
* **The gate:** `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after the
  last code change (the only edits after it are to this record), exit 0: "gate: 30 of 30
  passed", every check rc=0, `gate-selftest` and `tools-test` included
  (started 15:39, finished 15:51 local, 717 s). `make gate-selftest` and
  `make tools-test` pass with the changed `check_api.py` and `apidiff.py`.
* Not done, as the phase says: no `wizard` change, no `Driver`, no CHANGELOG
  line (Phase 6), none of 0075's `ConsentText`, `HostID` or `ClientStore`, and
  none of 0079.

### Phase 5, deviations (2026-11-13)

Found while executing Phase 5, before any commit. The owner ruled the same
day.

* **Deviation 1: `scripts/test_gates.py` joins Phase 5 (blocking).**
  * **Evidence.** `python3 -B tools/gate.py --timeout 5400` ended "gate: 29 of
    30 passed"; the one failure was `gate-selftest`, rc=2:
    `scripts/test_gates.py:338`, `test_tables_check_architecture_stale`,
    ``AssertionError: architecture.md: '`providers/openai`, `internal/redact`,
    `golang.org/x/term` |' found 0 times``. The test anchors on `wizard`'s row of
    `docs/architecture.md`, which Phase 5's note and `make tables-check` change
    (`authcontract` and `internal/signinflow` added). It passed before this
    phase's edit, so it is not pre-existing.
  * **Resolution, the owner's ruling "Add test_gates.py to Phase 5".** The
    file joins Phase 5's scope. The test's `replace_once` anchor and its
    replacement take `wizard`'s new row text; its intent, a stale package in
    an architecture row, is unchanged. A plant on a scratch clone shows it
    still bites.
* **Deviation 2: the `Driver` is handed the terminal step (2026-11-13).**
  * **Evidence.** The first Phase 5 build handed the `Driver` `open_url`,
    `show_code`, `waiting` and `paste` only, on the reading that the wizard
    returns the outcome itself. 0074-MADR D6 says the `Driver` is "called once
    per step", `done` and `failed` are steps (D3), and gobble-cli's
    `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D5
    prints "the last `done` or `failed`" through it. Whether a `Driver`'s own
    error also gets a `failed` step was not settled by D6: the engine's step
    after such an error is `failed` with kind `canceled` (after the wizard's
    `Cancel`) or `other` (a `show_code` error through the device seam), a race,
    not the `Driver`'s own error.
  * **Resolution, the owner's rulings "No final step after its own error" and
    "Keep the credential, warn"**, recorded under 0074-MADR D6: no terminal
    step after the `Driver`'s own error; otherwise exactly one, last; an error
    on `done` leaves the credential and gives one warning notice. The
    coordinator's reading, for the owner to overturn: an error on `failed`
    leaves the sign-in's own error returned, and gives one warning notice.
  * **Tests added to Phase 5's table:** `TestDriver_TerminalDone_0074`,
    `TestDriver_TerminalFailed_0074`, `TestDriver_OwnErrorGetsNoTerminal_0074`,
    `TestDriver_TerminalErrorWarns_0074`.

### Phase 5 (2026-11-13)

Base: `main` at `d38b5c4` (Phase 4 committed), tree clean. The PLAN's Phase 5,
"Rules for every phase", the R55 table, "Phase 4, deviations" (Deviation 2
especially) and 0074-MADR D6 were read first. Deviations 1 and 2 are in
"Phase 5, deviations" above, each ruled by the owner on 2026-11-13.

* **Step 1, the browser, plan and device paths on the engine.**
  `wizard/auth.go` `browserSignIn` runs `internal/signinflow` with the
  wizard's own seams (`loginBrowserOAuth`, `loginChatGPTPlan`); the renderer
  reuses `browserFlowOptionsFor` and `pastePrompt`, so the notices, the paste
  prompts, the races and the drains are today's. The wizard builds the method
  facts itself (`url_or_code`; `url_only` for a plan's first sign-in;
  `Yields` `oauth_session`, `device_key` for `Options.KiloDeviceKey`), so a
  descriptor without facts keeps today's behaviour; `NoPastePrompt` sets
  `NoPaste`. `wizard/device.go` `deviceSignIn` runs the `deviceLogin` seam on
  a goroutine, so the engine owns the order of `show_code` and `waiting`; the
  seam's show waits for the renderer's verdict, so an error from
  `ShowDeviceCode` or the `Driver` still ends the sign-in before any poll and
  is returned as it is. A login that shows no code gets no `show_code` render.
  The three Kilo and Grok/OpenAI device call sites and the plan call site use
  the helpers.
* **Step 2, `wizard/driver.go` and `Options.Driver`.** `StepDriver` and the
  renderer loop `stepRenderer.run`. A `Driver` replaces the `Prompter`'s
  sign-in notices and paste, `ShowDeviceCode` and `OpenURL`; `NoPastePrompt`
  still decides whether a paste is asked. The Driver's context ends with the
  sign-in, so a pending paste sees the loopback's win; a Driver error ends the
  sign-in and is returned as it is. Deviation 2's rulings: the Driver is
  handed the sign-in's last step, `done` with `Yields` or `failed` with the
  error's kind, exactly one and last, except after its own error; an error on
  that step is one warning and changes nothing.
* **Step 3, an `Options.OpenURL` error** is one warning notice, "could not
  open the browser (%s); open the URL above", and the sign-in goes on. The
  text of the error is shown through `redact.String` and `redact.StripControl`,
  not `describe`, which would read a caller's error as "the request failed".
* **Behaviour change to note.** The engine's paste checks (state, a code, a
  `client_id` where a URL is required; three pastes) now apply in the wizard
  too: a paste the engine refuses is asked again, with a warning notice giving
  its reason, where `auth` once received it. No existing test changed.
* **Files.** `wizard/auth.go`, `wizard/device.go`, `wizard/configure.go`,
  `wizard/driver.go`, `docs/architecture.md` (`wizard`'s row names
  `authcontract` and `internal/signinflow`, per Phase 4's Deviation 2), and,
  by Deviation 1, `scripts/test_gates.py`. Not touched: `wizard/import.go`
  (the vendor import has no step), and none of 0075's or 0079's names.
* **Tests.** Every existing `wizard` test passes unchanged (`go test -race
  -count=3 ./wizard`). New, all `_0074`: `TestDriver_ReceivesSteps`,
  `TestDriver_PasteAnswerCompletes`, `TestDriver_ContextEndsPaste`,
  `TestDriver_ErrorEnds`, `TestOpenURLErrorWarns` (in its own file,
  `wizard/openurl_0074_test.go`, so it builds at `773b84c`), and Deviation 2's
  `TestDriver_TerminalDone`, `TestDriver_TerminalFailed`,
  `TestDriver_OwnErrorGetsNoTerminal`, `TestDriver_TerminalErrorWarns`.
* **Red, then green.** `python3 -B tools/redgreen.py --rev 773b84c ... --expect
  fail`: `TestOpenURLErrorWarns_0074` fails on its assertion at `773b84c`
  ("notices [Open https://authorize.test in your browser]; want ... could not
  open the browser (no display); open the URL above"), and passes on the tree.
  `REFRESH.json --expect pass`: 65 of 66, exit 1, the one "missing"
  `TestOAuthSessionWait_WindowsRemoval_0071`, which skips off Windows, as in
  Phases 2 and 3; none of the 15 files changed.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, every one an
  assertion, waits bounded to 5 s:
  * `ShowDeviceCode called 1 times, notices []; want neither`;
  * paste answer dropped: `InputCode() error = ... no usable callback after 3
    pastes`, and `the Driver was given [open_url paste paste paste]; want
    [open_url paste ...]`;
  * ctx not cancelled: `the sign-in did not return within 5 s of the
    loopback's win`;
  * the Driver's error on `open_url` ignored: `the sign-in did not end within
    5 s of the Driver's error`;
  * the `OpenURL` notice omitted: `notices ["Open https://authorize.test in
    your browser"]; want "could not open the browser (no display); open the
    URL above" once`;
  * the terminal step dropped: `the Driver was given [show_code waiting]; want
    [show_code waiting done]`, and `terminal steps [], last false; want one
    done, last`;
  * `failed` sent as `done`: `TestDriver_TerminalFailed_0074` fails, "terminal
    steps [... Kind:done ...]; want one failed, last";
  * a terminal step after the Driver's own error: `the Driver was handed
    [{... Kind:failed ...}] after its own error; want no terminal step`;
  * a Driver error on `done` returned, not warned: `kind "", error the last
    line could not be written; want the session kept`.
  * Phase 4's moved gate plant: `make tables-check` with `authcontract`
    removed from `wizard`'s R2 row: `check-tables:
    docs/guides/api-standards.md: wizard: imports authcontract, which its row
    does not allow`; from `wizard`'s architecture row: `check-tables:
    docs/architecture.md: wizard: imports authcontract, which its row does not
    name`. Both exit 2 through `make`.
  * Deviation 1's: with the stale-name report removed from
    `scripts/check_tables.py`, `test_tables_check_architecture_stale` fails:
    `AssertionError: 0 != 1 : tables-check exit 0, want 1:`.
* **Checks,** each exit status captured before any filter:
  * `make pre-add-check` on the seven Go files, exit 0, "go-precheck: 7
    file(s) clean". The first run failed errcheck twice and gosec G118 in
    `driver.go`; fixed by returning the `Wait` result through a channel and by
    treating a refused `Answer` as the sign-in's end, no `nolint`.
  * `make api-check`, exit 0, "0 incompatible change(s) outside
    llmprovider/x/". `python3 -B tools/apidiff.py v1.30.0`, exit 0; for the
    wizard only `wizard.Options.Driver` and `wizard.StepDriver`, as R55's table
    says.
  * `make tables-check docs-check coverage-check`, exit 0; `wizard` 94.1 %
    against 86.4 %.
  * **The gate:** `python3 -B tools/gate.py --timeout 5400` through
    `run-heavy`, after the last code change, exit 0, "gate: 30 of 30 passed",
    started 18:05:39, finished 18:22:29 local (1010 s). A first gate, before
    Deviation 1, was 29 of 30 (`gate-selftest`), and is the evidence of that
    deviation.
* Not done, as the phase says: no CHANGELOG line, README or architecture
  section (Phase 6), none of 0075's `ConsentText`, `HostID` or `ClientStore`,
  and none of 0079's `Options.Identity` on the wizard's requests.

### Phase 6, deviation (2026-11-13)

* **Finding.** Phase 6 step 5 has `AGENTS.md` "Live tests" say that
  `LLMPROVIDER_LIVE_DEVICE_LOGIN` covers "the Grok, Kilo and OpenAI device
  logins", for `TestLive_OpenAIDeviceLogin`. That test is not built: it is
  Phase 7 step 1's, owner-run, and `grep -rn OpenAIDeviceLogin` over the
  Go, Markdown, Makefile and workflow files finds it only in these records.
  `AGENTS.md` lines 259 to 261 read "(the Grok and Kilo device logins)",
  which is true today. Writing step 5 now would describe an unbuilt test.
* **Ruling, 2026-11-13, the owner: "Move it to Phase 7 step 1".** The
  `AGENTS.md` line lands in the commit that adds `TestLive_OpenAIDeviceLogin`.
  Step 5 and Phase 7 step 1 are annotated above, not rewritten. The MADR's
  C53 is unchanged: its asserted fact is the test's variable, which holds.

### Phase 6 (2026-11-13)

Base: `main` at `7415801`, tree clean. Phase 6, "Rules for every phase" and
every execution entry from Phase 0 on were read first, so the docs describe
what was built: Phase 4's deviations (`api-check` per subtree, the facts
table of `x/signin`, the deadline plant), Phase 5's (the `Driver`'s
terminal-step rulings) and Phase 8 step 1 (Grok `browser_paste`,
`url_or_code`, `NoLoopback` with a random-port redirect). Each claim was read
against the code.

* **Step 1, `docs/architecture.md`.** The `x/signin` and `signinflow` rows
  were Phase 4's. `auth`'s row now names `VendorCLIPath`, `SessionKind`,
  `OAuthSession.Kind`, the two sentinels and `NoLoopback` on both options;
  `x/signin`'s row names its experimental `NoLoopback`; a new "Sign-in steps"
  section, before "Identity".
* **Step 2, `README.md`.** A "Method facts and one sign-in handle" section
  (`x/signin` marked experimental), the `x/signin` row of the Packages table,
  and an `Options.Driver` bullet in the wizard section. The sentence "The root
  module does not require it yet", made false by Phase 4, is corrected.
* **Step 3, `CHANGELOG.md` `[Unreleased]`.** **Added** each exported name of
  R55's table; **Deprecated** the four; **Changed** the `OpenURL` warning.
  Four **Changed** lines beyond the phase's list, each describing built
  behaviour:
  * the wizard's paste checks and re-asks: Phase 5's "Behaviour change to
    note", `internal/signinflow/paste.go` `check`;
  * the vendor-path errors begin "auth:" where they began "wizard:", and the
    unsupported one matches `ErrUnsupported`: Phase 3 step 4's "Error text
    differs"; `llmprovider/auth/vendor_path.go:44` against `773b84c:wizard/import.go:29`;
  * the root requires `authcontract` v0.1.0: Phase 4 step 1, `go.mod:6`;
  * `make api-check` compares the `llmprovider` and `wizard` subtrees
    separately: Phase 4's Deviation 1, `scripts/check_api.py`.
* **Step 4:** `make parity-check`, exit 0; no `mcplib` row changed.
* **Step 5:** not done, moved to Phase 7 step 1 by the owner (above).
* **Claims.** No row is added. README, architecture and the CHANGELOG are not
  claim-checked records, and Phase 6 makes no new code assertion in a record;
  the coordinator's decision. *(Annotation, 2026-11-14: the owner ruled on this, relayed by the
  coordinator; chosen "No rows for docs edits", with "Add rows for cited
  code" and "Decide after the tag" rejected. The ruling, verbatim: "A claim
  row is added when a decision record asserts a fact about code. Edits to
  README, architecture and the changelog are covered by the docs checks
  instead. The three phases stand as recorded, re-attributed from my
  decision to your ruling, and the rule is written into the tools README so
  it is settled." The phrase "the coordinator's decision" above is the
  owner's ruling. The rule is in `tools/README.md`, beside `claims.py`.)*
* **Checks,** each exit status captured before any filter: `make docs-check`,
  `links-check`, `changelog-check`, `markdownlint`, `records-check`,
  `disclosure-check`, `tables-check`, `parity-check`, each exit 0;
  `tools/claims.py` on the 0074 pair, 0; `tools/denyscan.py --changed`, 0;
  `tools/recordlint.py --changed`, 0; `python3 -B tools/gate.py --markdown`
  through `run-heavy`, exit 0, "gate: 11 of 11 passed" (827 s). No `.go` file
  changed, so no full gate. No new test or gate, so no planted break. The
  records' checks were run again after this entry.
* **Provenance.** Mainstream model, medium effort. Outcome: Phase 6 done
  except step 5, which the owner ruled into Phase 7 step 1 on 2026-11-13
  (see "Phase 6, deviation"); staged for the coordinator to commit.

### Deviation (2026-11-13): two flaky tests root-caused

Found by 0075 Phase 2's first gate run (its PLAN, "Phase 2"): `test-shuffle`
failed on `TestStart_Cancel_0074` and on
`TestDriver_OwnErrorGetsNoTerminal_0074/show_code`. The owner's ruling
"Root-cause first, then commit": each was traced, on scratch clones at
`f64134d` and at `f64134d` with 0075 Phase 2's staged diff, before any fix was
chosen. Both predate 0075 Phase 2.

* **Flake 1, `TestStart_Cancel_0074`: test isolation, not a leak.**
  * **Failure.** `start_device_0074_test.go:164: 4 goroutines after the
    sign-in, 5 before`, always one fewer after, never more. The test runs in
    a `synctest` bubble but compares `runtime.NumGoroutine()`
    (`start_device_0074_test.go:134`, `:163`), the count of the whole
    process.
  * **The straggler.** A goroutine dump taken before and after the count, on
    an instrumented copy, shows one goroutine of the previous test, alive
    at `before` and gone at `after`:

    ```text
    goroutine 691 [runnable]:
    internal/signinflow.(*Run).askPaste             signinflow.go:433
    internal/signinflow.(*Run).runBrowser.func2     signinflow.go:399
    llmprovider/x/signin.(*loginProbe).run.func1    probe_0074_test.go:72
    created by llmprovider/x/signin.(*loginProbe).run in goroutine 690   probe_0074_test.go:71
    ```

    It is the test fake's paste read: `loginProbe.run` starts `inputCode` on
    a goroutine it does not join, as `auth.runBrowserLogin` does
    (`oauth_loopback.go:229`). The interleaving: the previous test closes
    `probe.release`; the probe returns and cancels the read's context;
    `Wait` returns and that test ends; `TestStart_Cancel_0074` reads
    `before`, counting the read still unwinding; the read returns; `after`
    is one lower. A few failures showed no straggler in the dump, or one at
    `runtime.goexit1`: a goroutine outside the bubble that ended between
    the count and the dump. A count of the whole process is unsound
    whatever the sign-in does.
  * **Which test leaves it.** Each top-level test run 200 times directly
    before `TestStart_Cancel_0074` (`-count` alternates the two): only
    `TestStart_PlanSteps_0074`, whose last subtest ends with its paste read
    pending, fails at a measurable rate. Unshuffled, the device tests run
    between the browser tests and this one, which gives the read time to
    end.
  * **No leak.** Scratch tests: 200 loopback-won sign-ins in one bubble
    leave at most one goroutine right after `Wait`, none once the bubble is
    idle; 1,000 outside a bubble peak at two and return to the baseline in
    about 120 µs. The read outlives `Wait` by microseconds and does not
    accumulate.
  * **Before 0075 Phase 2.** `TestStart_PlanSteps_0074` then
    `TestStart_Cancel_0074`, ten processes of 200, uninstrumented: 444 of
    2,000 failed at `f64134d`, 386 of 2,000 with 0075 Phase 2. 0075 Phase
    2's reading, that its tests made the flake likelier, is not borne out.
* **Flake 2, `TestDriver_OwnErrorGetsNoTerminal_0074/show_code`: a product
  defect.**
  * **Failure.** `driver_terminal_0074_test.go:151: the Driver was handed
    [{SchemaVersion:1 Seq:3 Kind:failed ...}] after its own error; want no
    terminal step`. The error returned stays the `Driver`'s in every run.
  * **Cause.** `wizard/driver.go:110-116`, from Phase 5 (`a0012db`), hands
    the `Driver`'s `show_code` error to the device login (`r.verdict <- err`)
    before it records it (`fail(err)`), and `fail` records only while
    `stepCtx` is live (`driver.go:93`). The interleaving: the renderer sends
    the verdict; the login's goroutine in `deviceFlowFor`
    (`wizard/device.go:93-103`) wakes, its `show` returns the error, it
    sends `done`; the engine's `flow.Wait` returns and `finish` emits
    `failed` #3 and closes `done`; the `Wait` goroutine
    (`driver.go:80-82`) calls `stop()`, ending `stepCtx`; then the
    renderer's `fail` finds `stepCtx` ended and records nothing, so the
    terminal step is handed to the `Driver`. That breaks 0074-MADR D6's
    ruling "No final step after its own error". Phase 5's Deviation 2 named
    this route a race; it was not closed. The `open_url`, `paste` and
    `confirm` paths cannot race: the sign-in waits on the renderer there.
  * **Counts,** `-race -count=150 -cpu 1,2` on the subtest. Twelve
    processes at once, eleven rounds, on darwin/arm64: 10 of 132
    processes failed, 10 of 39,600 `show_code` runs; one process at a time,
    0 of 6 there and 0 of 6 on linux/amd64 (WSL Ubuntu-24.04).
    It needs contention, as in the gate. A 20 ms sleep planted between the
    send and `fail`: 1,800 of 1,800 `show_code` runs failed on each host.
* **The owner's rulings (2026-11-13), verbatim.**
  * Q1, the count: "Count only its own test scope".
  * Q2, 0075 Phase 2's `settleGoroutines`: "Remove it, correct the entry".
  * Q3, where flake 2's fix is recorded: "0074 deviation, own commit first".
  * Q4, a stronger regression test: "Add the ordering check".
* **Resolution.**
  * `wizard/driver.go`: `fail(err)` before the verdict is sent, so the
    failure is recorded, and the run cancelled, before anything can end the
    sign-in.
  * `llmprovider/x/signin/start_device_0074_test.go`: `bubbleGoroutines(t)`
    counts the goroutines of the caller's `synctest` bubble from a dump of
    every goroutine (`runtime.Stack`), and calls `t.Fatal` if the caller is
    in no bubble. The row of `TestStart_Cancel_0074` in Phase 4's table
    ("checked with a goroutine count before and after") now means the
    bubble's count.
  * `wizard/driver_order_0074_test.go`, new:
    `TestDriver_ShowErrorRecordedFirst_0074`, a login whose `show` reports
    whether the sign-in was still live when it heard the `Driver`'s error.
* **Files added to scope:** `wizard/driver.go`,
  `llmprovider/x/signin/start_device_0074_test.go`,
  `wizard/driver_order_0074_test.go`, and this record and 0074-MADR's D6
  note. Landed in its own commit, before 0075 Phase 2's.
* **Not done here.** 0075 Phase 2's `settleGoroutines`
  (`llmprovider/x/signin/signin_0075_test.go`) is removed under 0075, by
  ruling Q2, and that PLAN's claim that its tests made the flake likelier is
  corrected there.

### The deviation's fix (2026-11-13)

Base: `f64134d`, on a scratch clone, not the working tree, which held 0075
Phase 2's staged work. The deviation above was recorded first, then the code.

* **Built.**
  * `wizard/driver.go`: on `show_code`, `fail(err)` before
    `r.verdict <- err`.
  * `llmprovider/x/signin/start_device_0074_test.go`: `before` and `after`
    by `bubbleGoroutines(t)`.
  * `wizard/driver_order_0074_test.go`:
    `TestDriver_ShowErrorRecordedFirst_0074`, 1,000 device sign-ins whose
    `Driver` fails `show_code`; each must return the `Driver`'s error, hand
    it no terminal step, and end the sign-in before the login hears the
    error.
* **Plants,** each on a further scratch copy, each caught:
  * the old order restored: `driver_order_0074_test.go:52: sign-in 236: the
    login heard the Driver's error while the sign-in was still live`; 8 of
    10 runs of the new test failed, and 10 of 10 under `-race`;
  * the old order with a 20 ms sleep before `fail`: the new test,
    `driver_order_0074_test.go:52: sign-in 0: the login heard the Driver's
    error while the sign-in was still live`; the `show_code` subtest,
    `driver_terminal_0074_test.go:151: the Driver was handed [{SchemaVersion:1
    Seq:3 Kind:failed ...}] after its own error; want no terminal step`;
  * a goroutine planted in `signinflow.(*Run).Cancel` that sleeps an hour:
    `start_device_0074_test.go:165: 5 goroutines in the bubble after the
    sign-in, 3 before`;
  * the bubble filter removed (`bubbleGoroutines` returning
    `runtime.NumGoroutine()`): the pair check below failed 451 of 2,000,
    `start_device_0074_test.go:165: 4 goroutines in the bubble after the
    sign-in, 5 before`.
* **Checks,** each exit status captured before any filter:
  * `make pre-add-check` with `FILES` the three Go files above: exit 0,
    `go-precheck: 3 file(s) clean`.
  * `make api-check`: exit 0, `0 incompatible change(s) outside
    llmprovider/x/`. `make tables-check`: exit 0, `47 packages, 0
    problem(s)`. `make docs-check`: exit 0, `0 violation(s) of R50,R51,R52`.
  * The pair check, `TestStart_PlanSteps_0074` then `TestStart_Cancel_0074`,
    ten processes of `-count=200`: 0 of 2,000 (444 of 2,000 at `f64134d`).
  * `-race -count=150 -cpu 1,2` on the `show_code` subtest, twelve processes
    at once, eleven rounds: 0 of 132 processes, 0 of 39,600 runs (10 of
    39,600 at `f64134d`). The new test under `-race`, `-count=10`: exit 0.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after
    the last code change: 22:18:58 to 22:39:33 (1,235 s), exit 0, `gate:
    30 of 30 passed`.
  * After this entry: `tools/claims.py` on the 0074 pair,
    `make records-check links-check disclosure-check`,
    `tools/denyscan.py --changed` and `tools/recordlint.py --changed`,
    each exit 0.
* **Provenance.** Strongest model, high effort. Root cause: flake 1, a
  count of the whole process's goroutines that another test's paste read,
  still ending, moves; flake 2, a `show_code` error handed to the login
  before it was recorded. Fix: the bubble's count, and the error recorded
  first, with an ordering test. Escalated to the owner with four questions,
  ruled 2026-11-13: "Count only its own test scope", "Remove it, correct
  the entry", "0074 deviation, own commit first", "Add the ordering check".
  Delivered as a patch on `f64134d` for the coordinator to commit before
  0075 Phase 2's commit.

### Phase 7, step 6, deviation (2026-11-14)

U14, the `-race` probe, found two races. Nothing signed in; no quota.

* **The probe.** A scratch clone at `4740e2e` with one added test: the
  wizard's credential step (`ResolveCredential`, Grok's browser sign-in
  stubbed) with a listing whose transport starts a goroutine that outlives
  it and loops `Invalidate()` then `Token()` on the new session, and with
  `-race -count=50`. A loop of `Token()` alone passed 50 of 50: a session
  that is current reads `Store` only when its rotation is unsaved.
* **Race 1, the `Store` write.** `refreshState` reads `s.Store` under the
  session's lock (`llmprovider/auth/oauth_session.go:598`, reached from
  `Token`, line 189). `resolvedCredential.save` wrote it with no lock
  (`wizard/auth.go:998`, `c.session.Store = c.saveTo`). MADR F27's
  prediction, that only the write after the listing could race, held.
  Evidence: with a store that reads no session field, the old code fails
  with exactly 1 report and the fix passes 50 of 50 with 0.
* **Race 2, the store's read.** `FileTokenStore.Save` read the live
  session's fields with no lock (`llmprovider/auth/tokenstore_file.go:166-170`
  at `4740e2e`), and the wizard's save passes it the live session
  (`wizard/auth.go:999`). `adopt` writes the same fields under the lock
  (`oauth_session.go:677-696` at `4740e2e`). Evidence: with a
  `FileTokenStore`, the old code gives 16 reports and the first fix
  (`UseStore` alone) 15. The 15 are all this race: `Invalidate`
  (`oauth_session.go:477`) against `Save` (`tokenstore_file.go:169`), and
  `adopt` against `Save` (`:167`, `:168`, `:170`).
* **The corrected premise.** The coordinator's brief for this step assumed
  that the three reports a test store's `Save` gave in the first probe were
  artifacts of a fake store that read session fields unprotected, and
  that a store's own lock would remove them. That was the coordinator's
  assumption, not the owner's, and it was wrong: those reports were this
  race, and the shipped `FileTokenStore` has it. A store's own lock cannot
  remove it, since the writer, `adopt`, holds only the session's lock.
* **The owner's rulings, 2026-11-14,** verbatim: "Locked setter `UseStore`",
  and "Save reads under the session's lock". Also: "Fix the race first":
  the race fix lands as its own commit before the rest of Phase 7's prep.
* **The lock graph,** read before any edit. Every call of a store's
  `Save` in non-test code:
  * `llmprovider/auth/oauth_session.go:343`, `persistRotation`, from
    `complete` (line 256), which holds no lock when it calls it. It passes
    `pending`, a copy `persistable()` made (line 193) with its own lock.
  * `llmprovider/auth/oauth_session.go:986`, `reloadOrRefresh`, from
    `complete` with no lock held. It passes `next`, a new session
    `refreshOAuthSession` returned.
  * `llmprovider/auth/consent.go:113`, `SetConsent`: it passes the session
    its own `Load` returned, which nothing else holds.
  * `internal/signinflow/signinflow.go:518`, through `SetConsent`'s
    `answerStore` (line 504): the same private session.
  * `wizard/auth.go:999`, `resolvedCredential.save`: the live session, with
    no lock held. This is race 2's call.
  * `adopt` (line 676) saves nothing; it runs under `s.mu` from `complete`.
  No caller holds the session's `mu` while it calls `Save`, so `Save` may
  take it. The only other `TokenStore` in the repository, `answerStore`,
  reads `Consents` of a session its own `Load` made. A test store in
  `wizard/session_0059_test.go` reads `Access` and `Refresh` unprotected;
  it is the one the first probe reported, and the new test does not use it.
* **Files added to scope:** `llmprovider/auth/oauth_session.go`
  (`UseStore`, `snapshot`), `llmprovider/auth/tokenstore_file.go` (`Save`),
  `llmprovider/auth/tokenstore.go` (`TokenStore`'s doc), `wizard/auth.go`
  (the call), `wizard/session_store_race_0074_test.go` and
  `llmprovider/auth/tokenstore_lock_0074_test.go` (new).
* **Known gap.** A `TokenStore` outside this repository cannot take the
  session's lock, so one that reads a live session's fields races with a
  refresh in the same way. `TokenStore`'s doc now says so. No exported way
  to snapshot a session was added: the ruling added no name beyond
  `UseStore`.

### Phase 7, step 6 (2026-11-14)

The fix for the two races of "Phase 7, step 6, deviation (2026-11-14)", in
its own commit before the rest of Phase 7's prep. Base `4740e2e`.

* **Files.** `llmprovider/auth/oauth_session.go`: `UseStore` and the
  unexported `snapshot`. `llmprovider/auth/tokenstore_file.go`: `Save` reads a
  `snapshot` of the session. `llmprovider/auth/tokenstore.go`: `TokenStore`'s
  doc. `wizard/auth.go`: `resolvedCredential.save` calls `UseStore`. New
  tests: `wizard/session_store_race_0074_test.go`,
  `llmprovider/auth/tokenstore_lock_0074_test.go`.
* **`UseStore(store TokenStore)`.** It sets `Store` under the session's
  lock. Its siblings differ: `UseHTTPClient`, `UseDefaultHTTPClient`,
  `UseIdentity` and `UseLogger` keep a value the session already has (first
  wins), and a nil or zero argument changes nothing. Chosen: a nil store
  changes nothing, as theirs do; the last call wins, because the call it
  replaces is `c.session.Store = c.saveTo`, which always replaced whatever
  the sign-in carried, and the wizard's save must still put the session on
  the store it is due to. The doc has R51's safe-for-concurrent-use
  sentence and R52's "It returns no error."
* **`Save`.** `FileTokenStore.Save` reads `s.snapshot()`, `persistable()`
  taken under `s.mu`. The lock graph (the deviation entry) shows no caller
  holds that session's `mu` when it saves, so the lock cannot deadlock:
  the internal saves pass private copies.
* **Fail before, pass after,** `-race -count=50`, each through `run-heavy`:
  * `TestSession_SaveBindsStoreWhileUsed_0074` on a clone at `4740e2e` with
    only the two new tests: exit 1, 16 DATA RACE reports, among them `Read
    at oauth_session.go:598 refreshState` against `Previous write at
    wizard/auth.go:998 resolvedCredential.save`, and `Write at
    oauth_session.go:477 Invalidate` against `Previous read at
    tokenstore_file.go:169 FileTokenStore.Save`. With the fix: exit 0.
  * `TestFileTokenStoreSave_RefreshSaveCompletes_0074`, same clone: exit 1,
    14 reports. With the fix: exit 0, bounded at 20 s, no deadlock.
  * With a store that reads no session field, the old code gives exactly 1
    report (the `Store` write) and `UseStore` alone gives 0, which separates
    the two races.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, exit 0, "all 5
  plant(s) caught":
  1. `Save` reads the session without its lock (`s.persistable()`): DATA
     RACE in `TestSession_SaveBindsStoreWhileUsed_0074`, which fails.
  2. `UseStore` sets `Store` without the lock: the same test fails with DATA
     RACE.
  3. the wizard assigns `Store` directly again: the same test fails with
     DATA RACE.
  4. `snapshot` does not take the lock: the same test fails with DATA RACE.
  5. `UseStore` ignores its argument: `tokenstore_lock_0074_test.go:27:
     Store = <nil> after UseStore(first); want first`.
* **Checks,** each exit status captured before any filter:
  * `make api-check`: exit 0, `0 incompatible change(s) outside
    llmprovider/x/`. `python3 -B tools/apidiff.py 4740e2e`: exit 0, one
    change, `(*OAuthSession).UseStore: added`.
  * `-race -shuffle=on -count=1` on `./llmprovider/auth/ ./wizard/
    ./internal/signinflow/ ./llmprovider/x/signin/`, on Linux through
    `run-on-host --need linux`: exit 0, four `ok`.
  * `gofmt -l`: clean. `make pre-add-check` on the fix's six Go files alone
    was not run apart from the prep: it lints the whole package, and the
    unstaged prep file `llmprovider/live_checks_test.go:630` fails
    `errorlint` (`%v` of an error in `fmt.Errorf`). That is the prep's, not
    the fix's: the fix's files add no issue.
  * The gate: with the prep in the tree, `python3 -B tools/gate.py --timeout
    5400` through `run-heavy`: exit 1, `gate: 26 of 30 passed`, 677 s. The
    four reds are `lint`, `pre-add-check` (the `errorlint` line above),
    `tables-check` (`AGENTS.md: LLMPROVIDER_LIVE_0074_PASTE_FILE: a live test
    reads it, and AGENTS.md does not name it`) and `gate-selftest` (which
    reruns `tables-check` on a clean copy). All four are the unstaged prep's.
    On a clone of `4740e2e` with only the fix's files copied in, after the
    last code change: 06:54 to 07:21, 905 s, exit 0, `gate: 30 of 30 passed`.
  * After this entry: `tools/claims.py` on the 0074 pair,
    `make records-check links-check disclosure-check docs-check`,
    `tools/denyscan.py --changed` and `tools/recordlint.py --changed`.
* **For the prep, which this commit does not carry.** Fix
  `live_checks_test.go:630`, and name `LLMPROVIDER_LIVE_0074_PASTE_FILE` in
  Phase 7's variable paragraph, as `..._GROK_PASTE_FILE` is named, and the
  prep's gate is then run again.
* **Provenance.** Mainstream model, medium effort. Escalated twice to the
  owner and ruled 2026-11-14: "Locked setter `UseStore`" for the first race
  and "Save reads under the session's lock" for the second; the coordinator's
  brief had assumed the second race was a test artifact, which the evidence
  did not support. Staged for the coordinator to commit.

### Phase 7, prep (2026-11-14)

Base `67be2b7` (the U14 race fix, committed and pushed). The code of steps 1 to
3 is built; step 4 ran in Phase 3a; step 5 needs no code; step 6 is the fix
above. **The owner's runs of steps 1, 2, 3 and 5 are pending; nothing was
signed in and no live test ran.**

* **Files.** `llmprovider/live_oauth_login_test.go`:
  `TestLive_OpenAIDeviceLogin` (step 1, `LLMPROVIDER_LIVE_DEVICE_LOGIN`; start,
  the owner approves, one generation on `gpt-6-astra`, revoke).
  `llmprovider/live_0074_paste_test.go` (new, tag `live_gateways`):
  `TestLive_0074PasteBrowser` (step 2: `x/signin` OpenAI `browser_oauth`,
  `NoLoopback`, a paste through the file, one forced refresh into a scratch
  store, revoke) and `TestLive_0074PastePlan` (step 3: `chatgpt_plan`,
  `NoLoopback`, a paste, a refresh; then the test holds port 1455 and signs
  in again to the same client with `Previous` and `NoPaste`, expecting the
  authorize URL's redirect on 1457; the owner signs in in a browser on the
  host; both sessions are revoked). `llmprovider/live_checks_test.go`
  (appended, untagged): `checkDeviceChatGPT`, `driveSteps`, `stepKinds`,
  `signInFinding`, `loopbackFinding`, `redirectPort`, `holdLoopbackPort`,
  `checkRefreshOnce`, `sameClient`. `llmprovider/live_twins_test.go`
  (appended): `TestLiveTwin_OpenAIDevice_0074` and
  `TestLiveTwin_PasteSteps_0074` (eight subtests). `AGENTS.md` "Live tests":
  `LLMPROVIDER_LIVE_DEVICE_LOGIN` covers "the Grok, Kilo and OpenAI device
  logins" (Phase 6 step 5, moved here by the owner).
* **Two things to know about the probes.** Step 3's second sign-in uses
  `Previous`, so the run registers one client under the owner's account, not
  two. Each confirm step (the plan's usage terms) is answered yes by the
  test, logging its text: the owner's run is the consent.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, exit 0, "all 14
  plant(s) caught", the tree untouched; the twins fail on each:
  1. no refresh token: `FINDING sign-in: ChatGPT session true, access set
     true, refresh set false; want a ChatGPT session with both tokens`.
  2. not a ChatGPT session (another issuer): `ChatGPT session false`.
  3. another word generated: `Generate = "OMEGA", <nil>`.
  4. a refresh after revocation succeeds: `FINDING revocation: a refresh
     after revocation succeeded`.
  5. `driveSteps` answers every confirm yes: `no: confirm answer =
     Answer{... Confirm:true}; want step 1, confirm false`.
  6. `driveSteps` answers a paste for the wrong step: `paste answer for step
     1; want step 3 with the pasted line`.
  7. `driveSteps` never shows the URL: `the URL was shown as []; want it
     once`.
  8. `signInFinding` passes a sign-in with no paste: `no paste:
     signInFinding = ..., true; want ..., false`.
  9. `loopbackFinding` passes a failed sign-in.
  10. `redirectPort` ignores the URL: `redirectPort(1455) = 1457, <nil>`.
  11. `holdLoopbackPort` holds nothing: `port <n> was free after
      holdLoopbackPort`.
  12. `sameClient` accepts another client: `another client: sameClient =
      <nil>; want ok false`.
  13. a refresh does not rotate: `FINDING refresh: forced refresh: err <nil>,
      token rotated false`.
  14. `redirectPort`'s two failures merged again into one message:
      `live_twins_test.go:697: redirectPort("https://auth.twin.test/authorize?state=x")
      error "authorize URL: no redirect_uri with a port (%!w(<nil>))" holds a
      formatting fault`.
* **A fix of mine to the prep, and the coordinator's review finding.**
  `redirectPort`'s error wrapped its cause with `%v`, which `errorlint`
  refuses; the owner's "Allow the fix" of 2026-11-14 made it `%w`. The
  coordinator's review found that one message served two failures, so an
  empty port, whose cause is nil, read `%!w(<nil>)`. They are split: a parse
  error is wrapped with `%w`, and an empty port is a plain error. The
  `redirectPort` twin checks that no error holds `%!` and that a parse error
  wraps a `*url.Error`.
* **0076 and 0079.** Their claims tables quote the old `AGENTS.md` line at
  `0076-PLAN:320` and `0079-PLAN:453`. `tools/claims.py` on both pairs: exit
  0, 48 of 48 and 7 of 7, 62 of 62 and 13 of 13. The rows are pinned to
  their revisions, so they pass; nothing changed in either record.
* **Checks,** each exit status captured before any filter:
  * `make pre-add-check` on the four Go files: exit 0, `go-precheck: 4
    file(s) clean`. `go vet -tags live_gateways ./llmprovider/`: exit 0.
    `make tables-check docs-check`: exit 0, `47 packages, 0 problem(s)` and
    `0 violation(s) of R50,R51,R52`. `go test -run TestLiveTwin
    ./llmprovider/`: exit 0.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, after the
    last code change, in the tree: 08:40:20 to 09:01:25 (1,266 s), exit 0,
    `gate: 30 of 30 passed`.
  * After this entry: `tools/claims.py` on the 0074, 0076 and 0079 pairs,
    `make records-check links-check disclosure-check markdownlint`,
    `tools/denyscan.py --changed` and `tools/recordlint.py --changed`.
* **Provenance.** Mainstream model, medium effort. Escalated once for the
  race (Phase 7, step 6, deviation), ruled and fixed in `67be2b7`. The
  owner's runs are pending: the run sheet comes after 0075 and 0079.

### Phase 5, deviation (2026-11-14)

* **Finding.** Phase 5 built the wizard's last call to the `Driver`, the
  terminal `done` or `failed` step, on `context.WithoutCancel(ctx)`
  (`wizard/driver.go`, `lastStep`). The context never ends, so a `Driver`
  that blocks there, such as one writing the last line to a reader that has
  gone, hangs the wizard forever, with the credential unsaved. Nothing bounds
  it: the sign-in has ended, and the caller's ctx is detached.
* **Ruling, 2026-11-14, the owner:** "Fixed 5 s limit".
* **The change.** `lastStep` calls the `Driver` under
  `context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)`. It stays
  detached from the sign-in's end. A `Driver` error on that step, a timeout
  included, is the one warning notice already ruled on 2026-11-13; for `done`
  the credential is kept, and for `failed` the sign-in's own error stands.
  `StepDriver`'s doc says the last step's ctx ends after 5 s. No exported
  name changes. A `Driver` that ignores its ctx still blocks; the doc already
  requires it to return soon after ctx ends.
* **Files added to scope:** `wizard/driver.go`,
  `wizard/driver_terminal_timeout_0074_test.go` (new).
* **A discrepancy noticed.** MADR D6's terminal-step note says the caller's
  own cancellation "still ends" the last step's ctx. `WithoutCancel(ctx)`
  never did, and this change keeps it so: the ruling asked for a detached
  context. The MADR note says so.

### Phase 5, the terminal step's limit (2026-11-14)

The change of "Phase 5, deviation (2026-11-14)". Base `7138882`.

* **Files.** `wizard/driver.go`: `lastStep` calls the `Driver` under
  `context.WithTimeout(context.WithoutCancel(ctx), lastStepLimit)`, with
  `lastStepLimit` 5 s; `StepDriver`'s doc says the last step's ctx ends after
  5 s and that a timeout is the one warning. The error path was already the
  single warning, with the result or the sign-in's error returned as it was.
  `wizard/driver_terminal_timeout_0074_test.go` (new).
* **The test,** `TestDriver_BlockedTerminalStepTimesOut_0074`: a Driver that
  blocks on `done` until its ctx ends. The wizard returns after 5.00 s, with
  the session kept and exactly one warning ("returned an error on the done
  step"); it fails below 4 s and above 7 s. Real time: the wizard's context
  deadline is the real clock, so the test takes 5 s.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, exit 0, "all 2
  plant(s) caught":
  1. the 5 s limit removed (`WithCancel` in its place): `--- FAIL:
     TestDriver_BlockedTerminalStepTimesOut_0074 (7.00s)`,
     `driver_terminal_timeout_0074_test.go:46: the wizard did not return
     within 7 s of a Driver that blocks on the terminal step`: it fails by
     its own bound, not a long timeout.
  2. the warning dropped: `driver_terminal_timeout_0074_test.go:55: notices
     []; want exactly one warning`, and the existing
     `driver_terminal_0074_test.go:179` and `:194`.
* **Checks,** each exit status captured before any filter: `make
  pre-add-check` on the two files, exit 0, `go-precheck: 2 file(s) clean`;
  `make api-check`, exit 0, `0 incompatible change(s) outside
  llmprovider/x/`; `tools/apidiff.py HEAD`: no change, no new name; the
  `TestDriver_` tests under `-race`: pass. `python3 -B tools/gate.py
  --timeout 5400` through `run-heavy`, after the last code change: 09:30:35
  to 09:47:10 (995 s), exit 0, `gate: 30 of 30 passed`. After this entry:
  `tools/claims.py` on the 0074 pair, `make records-check links-check
  disclosure-check markdownlint docs-check`, `tools/denyscan.py --changed`
  and `tools/recordlint.py --changed`.
* **Provenance.** Mainstream model, medium effort. Escalated once, ruled
  2026-11-14: "Fixed 5 s limit".

### Phase 7, step 3, deviation (2026-11-14): a probe asserted a parameter the plan never sends

The owner's live run of `TestLive_0074PastePlan` (step 3) failed before any
sign-in:

```text
live_0074_paste_test.go:82: consent chatgpt_plan.background_use, answered yes by the owner's run: ...
live_0074_paste_test.go:86: authorize URL originator = [], want ["go-llmprovider-sdk"]
```

* **Cause.** `pasteSignIn` checked every authorize URL for an `originator`
  naming this module (`checkAuthorizeIdentity`, `param` "originator"), and
  step 3's first call passed it for `chatgpt_plan`. `planAuthorizeURL`
  (`llmprovider/auth/oauth_plan.go:233-257`) sets no `originator`: a first
  sign-in sets `query.Set(oauthParamClientID, planDynamicClientID)` and
  `query.Set("agent_name_hint", strings.TrimSpace(opts.AgentName))`, a
  sign-in again `query.Set(oauthParamClientID, previous.ClientID)` and, when
  it has one, `query.Set("id_token_hint", previous.IDToken)`. `browser_oauth`'s
  URL does carry the `originator`, so step 2's call was right.
* **This is a probe defect found by the owner's live run, not an SDK
  defect.** The SDK built the URL it always builds. The offline twin did not
  catch it because its authorize URLs were hand-made (`twinAuthorizeURL`,
  `identityFake.authorizeURL`), never the shipped builder's.
* **The fix.** `pasteSignIn` takes an `authorizeWant`, a small spec of what a
  URL carries (parameters that equal a value, parameters that are present,
  parameters that are absent), checked by `authorizeError`, which names a
  parameter and never a value. `browserAuthorizeWant` is `browser_oauth`'s
  (`originator`, or `referrer` for Grok); `planFirstAuthorizeWant` is the
  plan's first sign-in (`agent_name_hint` equal to the app's name, a client id
  and a host id, no `originator`, no `id_token_hint`); `planAgainAuthorizeWant`
  is a sign-in again with `Previous` (its client id, its ID token as
  `id_token_hint` when it has one, no `agent_name_hint`, no `originator`),
  also checked on step 3's second sign-in.
* **The twin now drives the shipped builders.** `TestLiveTwin_AuthorizeURLs_0074`
  takes the URL of `signin.Start` (the plan with no network, so its built-in
  endpoints are used, and `browser_oauth` at a fake issuer, both with
  `NoLoopback`) and of `auth.LoginBrowserOAuth` for Grok, and runs each probe's
  assertion on it: the plan's first sign-in, the sign-in again, 0075's reuse
  with a kept client record (`reuseAuthorizeError`), OpenAI's browser URL with
  and without an identity (0079's `originator` "gobble"), Grok's `referrer`,
  and the redirect port of a `NoLoopback` plan URL. One subtest runs the old
  assertion on the plan's URL and requires it to fail.
* **The audit of the other live probes,** each asserting something of an
  authorize URL or of a device form:
  * `TestLive_0074PasteBrowser` (OpenAI `originator`): was covered only by
    `auth`'s own tests of the builder, not the probe's assertion; now by the
    twin through `signin.Start`.
  * `TestLive_0075ClientReuse` (`reuseAuthorizeError`): was run only on
    hand-made URLs; now on the URL of `signin.Start` with a kept record. It
    never asserted an `originator`, so it would not have failed this way.
  * `TestLive_0079Name`, ChatGPT half (`originator` "gobble"): was run on a
    hand-made URL; now on `signin.Start`'s `browser_oauth` URL under an
    identity. Its Grok half checks the device form's `referrer` through
    `checkDeviceReferrer`, which `TestLiveTwin_Identity_0076` runs on the
    shipped `StartDeviceOAuth` against a fake issuer: covered.
  * `TestLive_0074GrokPaste` (Grok `referrer`): the live probe passed on
    2026-11-13; now also covered by the twin on `LoginBrowserOAuth`'s URL.
  * `TestLive_OpenAIDeviceLogin`, `TestLive_GrokDeviceLogin` and
    `TestLive_KiloDeviceLogin` assert nothing of an authorize URL or a form.
  None would fail live the same way.
* **Plants,** `python3 -B tools/plant.py` on a scratch clone, exit 0, "all 6
  plant(s) caught": the plan's first sign-in asserting an `originator` again
  (`the plan's first sign-in URL: parameter originator has 0 value(s), the
  expected one among them false; want exactly the expected value`); the plan
  again asserting an `agent_name_hint`; the reuse asserting a hint; the
  browser check naming another parameter; `authorizeError` ignoring absent
  parameters (`authorizeError accepted an absent parameter that is sent`); and
  the plan's first sign-in requiring an `originator` to be present.
* **Run sheet.** A3's expected `FINDING` lines are unchanged. The sheet's
  scratch variable and `mktemp` setup are replaced by a fixed folder per
  probe, and the paste line is guarded.

### Phase 7, step 3, deviation (2026-11-14), execution

Base `b6a2125`. Files: `llmprovider/live_0074_paste_test.go`,
`llmprovider/live_checks_test.go` (appended, with the `maps` import),
`llmprovider/live_twins_test.go` (appended, with the `x/signin` import).

* **Checks,** each exit status captured before any filter: `make
  pre-add-check` on the three files, exit 0, `go-precheck: 3 file(s) clean`;
  `go vet -tags live_gateways ./llmprovider/`, exit 0; `make tables-check
  docs-check`, exit 0; `go test -run TestLiveTwin ./llmprovider/`, exit 0.
  `python3 -B tools/gate.py --timeout 5400` through `run-heavy`, in the tree, after the last code change: 12:05:27 to 12:23:58 (1,111 s), exit 0, `gate: 30 of 30 passed`. The plants: all 6 caught. After this entry: `claims.py` on the 0074 pair, `make records-check links-check disclosure-check markdownlint docs-check tables-check`, `tools/denyscan.py --changed` and `tools/recordlint.py --changed`.
* **No gate test binds 1455 or 1457.** Every test that signs in at OpenAI
  with a listener sets `openaiLoopbackPorts` to free ports first
  (`no_loopback_0074_test.go`, `identity_0076_test.go`,
  `plan_login_0043_test.go`, `oauth_loopback_test.go`); the others sign in at
  Grok. The new twins use `NoLoopback`, which binds none, or a free port.
* **Provenance.** Mainstream model, medium effort. The defect was found by
  the owner's live run, not by a check; no escalation.

### Phase 7, the owner's runs (2026-11-14)

The owner ran steps 1, 2, 3 and 5 on 2026-11-14, on the host, and all
passed. The lines below are the owner's pasted output, with any URL, code or
id removed; nothing else is changed. Step 3's pass is its second run, on the
corrected probe at `c47602b`; the owner's notes do not name the revision of
the other runs. Each run used a scratch store.

**Step 1, `TestLive_OpenAIDeviceLogin`: a pass.**

```text
FINDING sign-in: OpenAI device login approved; a ChatGPT session with an access and a refresh token
FINDING use: one short generation through the ChatGPT backend returned the requested word
FINDING revocation: revoked; a refresh afterwards is refused, as ErrAuthFailure true
--- PASS: TestLive_OpenAIDeviceLogin (105.80s)
exit status: 0
```

Quota: one short generation on the owner's ChatGPT plan, as the step says.

**Step 2, U2, `TestLive_0074PasteBrowser`: a pass.**

```text
FINDING redirect: the authorize URL's redirect_uri names port 1455, where nothing listens
FINDING sign-in: OpenAI browser sign-in completed by a paste of a callback URL with code and state (scheme http, a loopback host, another path); steps: open_url, paste, paste, done
FINDING refresh: the token rotated once and the scratch store holds it
FINDING revocation: revoked; a refresh afterwards is refused, as ErrAuthFailure true
--- PASS: TestLive_0074PasteBrowser (491.80s)
```

* **Device and browser:** the owner's phone, Chrome. No local-network
  prompt appeared; in the owner's words, "the browser on my phone is chrome
  and no prompt appeared".
* **The address bar.** After approval the phone was redirected to the
  loopback address on port 1455, which failed to load. Its address bar
  showed only the host and port until tapped; the full address, with the
  code and state, was copied from it then.
* **Two pastes.** `steps` holds two `paste` entries. The first paste was the
  authorize URL itself: the host's clipboard still held the link sent to the
  phone. The engine refused it locally and asked again. The second paste,
  the callback address, completed the sign-in.
* **Who wrote the second paste file.** The coordinator wrote it from the
  owner's clipboard, at the owner's word ("pasted"), with a guard that the
  text held `code=`. The value was not displayed or logged, and this record
  quotes none of it.
* A first attempt at this probe failed to start its log, because the run
  sheet's commands relied on a `SCRATCH` variable from another terminal. It
  was rerun with fixed paths (the run sheet's change is in "Phase 7, step 3,
  deviation (2026-11-14)").

**Step 3, U3, `TestLive_0074PastePlan`: a pass, on the second run.** The
first run stopped at the probe's own check of the authorize URL, before any
sign-in, and nothing was sent to OpenAI: the probe asserted an `originator`
that `planAuthorizeURL` never sets. That defect, its lines and its fix are
"Phase 7, step 3, deviation (2026-11-14)" and its execution entry, above;
the fix is `c47602b`. The second run, on the corrected probe:

```text
FINDING redirect: the authorize URL's redirect_uri names port 1455, where nothing listens
FINDING sign-in: OpenAI plan sign-in completed by a paste of a callback URL with code and state (scheme http, a loopback host, another path); steps: confirm, open_url, paste, done
FINDING sign-in: issued client of 32 chars; token URL <the issuer's token URL>
FINDING refresh: the token rotated and the store holds it
FINDING port: with 1455 held, the authorize URL redirects to port 1457
FINDING sign-in: the OpenAI plan again sign-in completed by the loopback, which redirected to port 1457; steps: open_url, done
FINDING sign-in: issued client of 32 chars; token URL <the issuer's token URL>
FINDING reuse: the sign-in again is under the first sign-in's client and host id
FINDING refresh after revocation: refused, ErrAuthFailure
FINDING refresh after revocation: refused, ErrAuthFailure
--- PASS: TestLive_0074PastePlan (191.36s)
```

* One paste, accepted the first time (`steps: confirm, open_url, paste,
  done`). The first sign-in was on the owner's phone, in Chrome; the second,
  the port fallback through the loopback, in a browser on the host.
* The confirm step (the plan's usage terms) was answered yes by the test, as
  "Phase 7, prep (2026-11-14)" says; the owner's run is the consent.
* One client, named `llmprovider-probe`, was registered on the owner's
  account, as the step's quota note says. The sign-in again reused it, by
  `Previous`, and registered none.

**Step 4** passed in Phase 3a on 2026-11-13 ("Phase 3a, the owner's run
(2026-11-13)"), and is not run again here.

**Step 5, the two moved device tests: both pass.**

`TestLive_GrokDeviceLogin`:

```text
--- PASS: TestLive_GrokDeviceLogin (26.61s)
exit status: 0
```

`TestLive_KiloDeviceLogin`:

```text
picked kilo-auto/free from [kilo-auto/free nvidia/nemotron-3.5-lightning:free poolside/laguna-s-2.1:free]
profile: 0 organizations, personal account true
--- PASS: TestLive_KiloDeviceLogin (21.99s)
exit status: 0
```

Kilo cannot revoke its token: it stays valid until the owner removes it in
Kilo's settings. Each test completed its sign-in on `StartDeviceOAuth` and
`Wait` and passed the checks it passed before Phase 3 moved it (R46: same
login, same checks).

**Step 6,** U14, is "Phase 7, step 6, deviation (2026-11-14)" and "Phase 7,
step 6 (2026-11-14)", above: two races, fixed in `67be2b7`.

**Phase 7 is complete.** Each step of the phase's list, read against its
text:

| Step | What the list asks | Done |
| :--- | :--- | :--- |
| 1 | an OpenAI device sign-in: start, approve, one short generation, revoke | yes, 2026-11-14 |
| 2 | U2: `x/signin` with `NoLoopback`, a paste from another device, one refresh, revoke | yes, 2026-11-14 |
| 3 | U3: the plan by paste from another device; then, with 1455 held, the fallback to 1457 by the loopback | yes, 2026-11-14, second run |
| 4 | Grok's browser sign-in by paste from another device | yes, Phase 3a, 2026-11-13 |
| 5 | the moved Grok and Kilo device tests, once | yes, 2026-11-14 |
| 6 | U14, the wizard write race, `-race -count=50` | yes, two races found and fixed, `67be2b7` |

No step is open. Phase 8, the release, is next: its step 1 ran on
2026-11-13, and the owner runs it waited on beside this phase's,
0075-PLAN Phase 6 and 0079-PLAN Phase 5, are recorded in those PLANs on this
date. No code changed in this entry.

* **Provenance.** Owner-run; recorded by the strongest model at high effort.

### Phase 8, the release prepared (2026-11-14)

Base `b39bbec`, the tree clean. This is Phase 8 up to the tag, with
0075-PLAN Phase 7 and 0079-PLAN Phase 6, which are this phase. Nothing is
committed, pushed or tagged. Each command's exit status was captured before
any filter. No Go file changed.

* **Step 1** ran on 2026-11-13 ("Phase 8, step 1 (2026-11-13)").
* **The section,** as `v1.30.0`'s was prepared (0076-PLAN Phase 5).
  `CHANGELOG.md`'s `[Unreleased]` became `## [v1.31.0] - 2026-11-14`,
  under an empty `[Unreleased]`, and the compare links read
  `[Unreleased]: …compare/v1.31.0...HEAD` and
  `[v1.31.0]: …compare/v1.30.0...v1.31.0`. The docs phases (this PLAN's
  Phase 6, 0075-PLAN Phase 5 and 0079-PLAN Phase 4) wrote the section
  before the work of 2026-11-14 below, so these entries were added from the
  code:
  * **Added:** `(*auth.OAuthSession).UseStore` (Phase 7 step 6;
    `llmprovider/auth/oauth_session.go`, `UseStore`).
  * **Added, in the `Driver` item:** the last step's 5 s limit ("Phase 5,
    deviation (2026-11-14)"; `wizard/driver.go`, `lastStepLimit`).
  * **Fixed:** the two races of Phase 7 step 6. Both are in `v1.30.0`: its
    `wizard/auth.go` assigns `c.session.Store = c.saveTo` with no lock, and
    its `FileTokenStore.Save` reads the session's fields with no lock while
    `adopt` writes them.

  Each name in the section was checked against the API diff below. The
  section names 0075's downgrade cost and the wizard's changed notice, as
  0075-PLAN Phase 7 asks. `make changelog-check`: exit 0, "45 release
  section(s), 0 problem(s)". `scripts/check_changelog.py --section
  v1.31.0`: exit 0, 129 lines, with Added, Deprecated, Changed and Fixed.
* **Version strings.** `README.md`'s status line reads "The current
  release is `v1.31.0`." Outside the changelog and the records, no other
  file names `v1.30.0`. `llmprovider/live_twins_test.go:1039` holds
  `go-llmprovider-sdk/1.31.0` inside a User-Agent fixture, and
  `chatgpt.ModelsClientVersion` is the backend's fallback `0.0.0`: neither
  is the release's version. No release phase names a migration note.
* **Docs currency** (the owner's standing requirement of 2026-11-14 that the
  docs be accurate and current). Every exported name this release adds has
  its Go doc, and is now also named in `README.md` or
  `docs/architecture.md`, besides the changelog. A read-only survey of a
  clone at `b39bbec` gave leads, each checked in this tree before any edit.
  Fixed, in the files the docs phases own (`README.md`,
  `docs/architecture.md`), where this release makes a statement false or
  leaves a name it ships unnamed:
  * `README.md`'s opening said the module depends on the standard library
    and `golang.org/x/term` only. `go.mod` now also requires `authcontract`
    `v0.1.0`, and `v1.30.0`'s did not. It now names the `authcontract`
    module.
  * `docs/architecture.md`'s issued-client item said whether OpenAI accepts
    the client with no hint "is for the owner-run probe of 0075-PLAN's
    Phase 6". It now says the probe showed it on 2026-11-14.
  * `docs/architecture.md`'s `generate-check` item named the two
    `mkwinsyscall` files only. `make generate-check` checks three, the third
    being `authcontract.schema.json` from `authcontract/doc.go`'s
    `go:generate` line, which `v1.30.0` did not have. It now names all three.
  * `UseStore` and `FileTokenStore.Save`'s lock were named in the changelog
    and their Go docs only. `docs/architecture.md` now names them in "The
    wizard" and the `TokenStore` item, and `README.md` names them in "Method
    facts and one sign-in handle".
  * `auth.SessionChatGPT`, `SessionChatGPTPlan`, `SessionGrok`,
    `SessionKiloDevice`, `llmprovider.IsBackground` and
    `(*FileTokenStore).LoadClient` and `SaveClient` were named in the
    changelog and their Go docs only. `README.md` now names each.
  * `README.md`'s package table and `docs/architecture.md`'s tree described
    `x/signin` as "one handle, `Methods` and `ErrorKindOf`". Both now name
    `Start`, `SignIn` and `Credential` too.

  Not edited, and open items for the owner, because no release phase owns
  the file or the statement predates this release:
  * `docs/guides/migrating-from-mcplib.md:24-25` says requests name
    `go-llmprovider-sdk` in the `originator` and the `referrer`. That is
    still the default, but since `v1.30.0` (`referrer`) and this release
    (`originator`) an application's name replaces it.
  * `llmprovider/auth/identity.go:19-20` says whether OpenAI and xAI accept
    every name is not verified. That stays true, but two names are now
    verified (0076-MADR, "Not verified", note of 2026-11-14). It is Go, out
    of this phase's scope.
  * `authcontract/README.md` documents the contract's six main types and its
    `Session*` constants, not its other types and constants (`Feature`,
    `FieldType`, `LinkRel`, `StepKind`, and the `Client*`, `Completion*`,
    `Failure*`, `Feature*`, `Field*`, `Kind*`, `Link*`, `Paste*`, `Step*`
    and `Yields*` constants). That module was tagged `authcontract/v0.1.0`
    by its own record.
  * From before this release: `README.md:28` lists the provider ids without
    `chatcompat`, and the package table there has no row for it or for
    `authcontract`; `README.md:158` says the library reads no environment
    variable by itself, but the default transport honours the proxy
    variables, as `README.md:121` says; `docs/architecture.md`'s tree has no
    line for `llmprovider/internal/compatid`; its `make` target list
    (`docs/architecture.md:1264-1269`) omits `markdownlint`, `links-check`,
    `disclosure-check` and `all`; `AGENTS.md`'s tools list does not name
    `tools/catalogcheck.py`; the provider key variables and `CODEX_HOME`,
    `GROK_HOME` and `GROK_AUTH_PATH` are named in no doc;
    `docs/README.md`'s "I want to…" matrix cites records up to 0046 only;
    four reports in its index are "observation" with no `status` in their
    front matter; and `docs/architecture.md:120` and `:1294` keep a
    "before the first tag" wording.
* **The API.** `make api-check`: exit 0, "against v1.30.0, 0 incompatible
  change(s) outside llmprovider/x/". `python3 -B tools/apidiff.py
  v1.30.0`: exit 0, 44 lines, each "added", nothing incompatible.

  | Record | R55 names outside `x/` | In the diff | In R55 only | In the diff only |
  | :--- | :--- | :--- | :--- | :--- |
  | this PLAN | 24: the nine `AuthMethod` facts; `SessionKind` with its four constants and `OAuthSession.Kind`; `(*OAuthSession).SessionKind`; `ErrSignInDenied`, `ErrSignInExpired`; the two `NoLoopback` fields; `(*OAuthSession).UseStore`; `VendorCLIPath`; `wizard.Options.Driver`, `wizard.StepDriver` | 24 | none | none |
  | 0075-PLAN | 17: `AuthMethod.Consent`, `ConsentText`, `ConsentURL`; `openai.ConsentBackgroundUse`; `OAuthSession.Consents`, `(*OAuthSession).Consent`, `StoredConsent`; `SetConsent`; `WithBackground`, `IsBackground`; `ClientRecord`, `ClientStore`, `(*FileTokenStore).LoadClient`, `SaveClient`; `ChatGPTPlanOptions.ClientID`; `wizard.Options.HostID`, `ConsentText` | 17 | none | none |
  | 0079-PLAN | 2: `(*OAuthSession).UseIdentity`, `(*Settings).ClientVersion` | 2 | none | none |
  | `x/signin` | the package, experimental | "package …/llmprovider/x/signin: added" | none | none |

  The diff names a type, not its fields or its interface's methods, so
  `ClientRecord`'s `ClientID` and `HostID`, `ClientStore`'s two methods and
  `StepDriver.SignInStep` are under their types. `wizard.Options.OAuthClients`
  is not in the diff: 0076-PLAN struck it. `go doc -all
  ./llmprovider/x/signin` lists `Methods`, `Start`, `Options`, `SignIn` with
  `Method`, `Steps`, `Answer`, `Wait` and `Cancel`, `Credential` and its five
  types, and `ErrorKindOf`, with `Options.NoLoopback`, `HostID`,
  `ConsentText` and `ClientStore`: the R55 rows of the three records for it.
* **The code claims: left as they are, and an open item for the owner.**
  `tools/claims.py` passes on every pair at its pinned revisions: 0074 67
  of 67 and 12 of 12, 0075 51 of 51 and 8 of 8, 0076 48 of 48 and 7 of 7,
  0079 62 of 62 and 13 of 13. Read at `HEAD` (`--rev self=HEAD --rev
  go-llmprovider-sdk=HEAD`), rows fail because the code they quote moved:
  * 0074-MADR, 53 of 67: C1–C6, C8–C11, C13–C26, C28–C31, C33, C35–C40,
    C42–C47, C51, C53–C63;
  * this PLAN, 8 of 12: P2, P4, P6, P7, P9–P12;
  * 0075-MADR, 33 of 51: C3, C5–C7, C9–C12, C15–C22, C25–C28, C30–C37,
    C40, C42–C45;
  * 0075-PLAN, 2 of 8: P2, P3;
  * 0076-MADR, 34 of 48: C1–C3, C5–C16, C19–C23, C25–C27, C29, C36–C42,
    C46–C48;
  * 0076-PLAN, 4 of 7: P1, P2, P5, P7;
  * 0079-MADR, 43 of 62: C1–C15, C20–C24, C27, C28, C30–C34, C43–C46,
    C50–C54, C56–C62;
  * 0079-PLAN, 4 of 13: P2, P5–P7.

  No record or tool says what a release does with them. What is written:
  `tools/claims.py` and 0040-PLAN's Deviation 1 pin a row to the revision
  its marker names; 0041-PLAN Phase 0 re-pinned a PLAN before its execution
  and kept its MADR at the revision the decision was made on; 0074-MADR's
  C56, C57, C60 and C62 carry "the row holds at its revision"; and
  `v1.30.0`'s release (0076-PLAN Phase 5) ran the claims at their pins and
  left them. Against these, 0079-PLAN's Phase 2 entry leaves 0079-MADR's
  C9 to C15, C22, C33 and C43 and its own P7 "for the release phase to
  re-pin", and no
  release phase lists that step. So the choice, pinned rows stand or rows
  re-pinned at the release commit, is the owner's, and no row or marker was
  changed. *(Annotation, 2026-11-14: the owner ruled on this, relayed by the
  coordinator; chosen "Pins stand", with "Re-pin 0079's deferred rows only"
  and "Re-pin all to the release" rejected. The ruling, verbatim: "A row is
  evidence for a decision at the commit it was made on and stays pinned
  there. Nothing is re-pinned for the release; this is what was done at
  v1.30.0. The one note in 0079 that deferred re-pinning to the release phase
  is annotated as superseded by this ruling. No delay to the tag." The rows
  stay as they are; nothing is re-pinned, and the question above is
  closed.)*
* **The checks of "Verification",** each exit status captured before any
  filter:
  * `go test -race -shuffle=on -count=1 ./...` on the Linux machine, through
    `run-on-host --need linux --race`, whose `target=` line named the Linux
    machine, with WSL excluded as a race target: exit 0, 42 packages `ok`,
    no `FAIL` and no `DATA RACE`, 13:41 to 13:44 (185 s).
  * `go test -race -count=1 ./...` on Windows, natively, through
    `run-on-host --need windows --race -- sh -c '...'`, whose `target=`
    line named the Windows laptop: exit 0, 42 packages `ok`, no `FAIL`,
    13:41 to 13:42 (55 s). *(Annotation, 2026-11-14: one run of the suite
    passes or fails by chance on this test; see the deviation entry named in
    the acceptance bullet below.)*
  * `go test -race -count=3 ./llmprovider/auth/` on the Linux machine, queued
    behind the run above: exit 0, `ok`, 34.7 s.
  * `python3 -B tools/redgreen.py REFRESH.json --expect pass`, the spec
    built from the 66 names in "Rules for every phase": 65 of 66, exit 1,
    the one "missing" being `TestOAuthSessionWait_WindowsRemoval_0071`,
    which skips off Windows, as in Phases 2 to 5. Its package passed in the
    Windows race run above, which is not verbose, so that run shows no
    `PASS` line of its own. None of the 15 files has changed since
    `v1.30.0`.
  * `make vuln`: exit 0, "No vulnerabilities found" for the root module and
    `authcontract`.
  * `make api-check` and `tools/apidiff.py v1.30.0`: above.
  * `tools/claims.py` on the four pairs: above, each exit 0.
* **The acceptance of this PLAN: not yet met, so it stays `in-progress`.**
  Phase 8's own acceptance waits on the tag: "the tag on the owner's word;
  then `tools/consumer.py` and `tools/offline.py`". Each criterion that does
  not wait on the tag is met: Phases 0 to 7 are complete; the changelog
  section is `v1.31.0`; the API diff lists exactly the R55 tables; the races
  pass on Linux and Windows *(annotation, 2026-11-14, after the pause: on
  native Windows `TestLiveTwin_GrokPaste_0074/FileInTwoWrites` failed about
  11 of 100 runs under `-race` and 14 of 100 without, as the release-prep
  agent measured in its diagnostic runs on native Windows (about 16:46 with
  `-race` and 17:02 without, local, 2026-11-14) and the coordinator relayed,
  so the Windows result above did not hold for every run. The owner
  chose to fix the helper; after the fix the test is 100 of 100 green on
  native Windows, with and without `-race`, and the Linux and Windows race
  passes ran again: see "Phase 8, deviation (2026-11-14): the Windows paste
  flake")*. The full gate, `run-heavy -- python3 -B
  tools/gate.py --timeout 5400`, runs after the last edit of this change,
  this entry included, so its result is not written here: it goes with the
  staged change, and the commit waits on its pass. The MADR stays
  `accepted`: no record changes its status at the release.
* **Not done, and waiting on the owner:** the commit; the push; the tag; CI
  on the tag and its Release; `tools/consumer.py v1.31.0` and
  `tools/offline.py v1.31.0`; R53's check once the Release appears, when
  seven Releases leave `v1.26.0`'s past the sixth (`gh release list` shows
  six now, `v1.30.0` to `v1.26.0`); the decision on the claims above.
* **The tag, for the owner.** `v1.30.0` and `authcontract/v0.1.0` are
  annotated, unsigned tags whose message is the tag's name. On the release
  commit: `git tag -a v1.31.0 -m v1.31.0 <release-commit>`, then `git push
  origin v1.31.0`. `ci.yml` runs on `v*` tags, in full, and its `release`
  job (`startsWith(github.ref, 'refs/tags/v')`) makes the GitHub Release
  from this section.
* **Provenance.** Prepared by the strongest model at high effort; no
  escalation to the owner.

### Phase 8, deviation (2026-11-14): a quota pause in the release prep

* **What happened.** The release prep stopped on 2026-11-14 for a quota
  pause, while its full gate waited in the heavy-run queue, and resumed the
  same day on the owner's word, "resume".
* **The tree on resume, read before any edit.** HEAD `b39bbec`. Six files
  staged and none unstaged: `CHANGELOG.md`, `README.md`,
  `docs/architecture.md` and the 0074, 0075 and 0079 PLANs. Each staged hunk
  was read whole, and none was half-applied: the entry above and the two
  release entries of 0075-PLAN and 0079-PLAN end with their provenance
  lines, and the changelog, README and architecture hunks are the edits that
  entry lists.
* **No check from before the pause is used for the release.**
  * The gate that was running started at 14:00:07, waited for the load
    (38.6, against a limit of 10), and was ended at 14:20:09 with status
    143, before it ran any check. It left no exit status.
  * Two earlier gate runs, started at 13:50 and 13:59, were stopped before
    any check ran, because docs edits followed them.
  * The race runs, the auth race run and the other checks the entry above
    quotes all ended by 13:46, before the docs edits of 13:49 to 13:59. Their results stand as observations of the code, which no edit
    since has changed, but the release does not rest on them.
* **After the pause.** Two lines that the docs edits had left over 80
  characters were reflowed, one in `README.md` and one in
  `docs/architecture.md`, with no change of wording. Then the checks ran
  again, after the last edit of this change, this entry included:
  `make api-check`, `tools/apidiff.py v1.30.0`, the record checks, the races
  on the Linux machine and on Windows through `run-on-host --race`, and the
  full gate through `run-heavy`. Their results go with the staged change,
  not into this entry, since writing them here would be an edit after the
  runs.
* **Provenance.** Recorded by the strongest model at high effort.

### Phase 8, deviation (2026-11-14): the Windows paste flake

Base `b39bbec`, with the release prep above staged and nothing committed.
Hosts are named as the plan names them: this Mac, the Linux machine and the
Windows laptop. Windows output below has its user folder replaced by
`<user>`.

* **What was found.** After the pause, native Windows runs of
  `TestLiveTwin_GrokPaste_0074/FileInTwoWrites` failed about 11 of 100 runs
  under `-race` and 14 of 100 without, at
  `llmprovider/live_twins_test.go:401`: `pasteFromFile = "", paste file:
  remove …\paste: The process cannot access the file because it is being used
  by another process.` It is not a race the detector reports, and it predates
  this release's prep: the helper and the twin were not touched by it.
  The rates were measured by the release-prep agent (the strongest model,
  high effort) in its diagnostic runs on native Windows, about 16:46 (`-race`)
  and 17:02 (no `-race`) local on 2026-11-14, and relayed by the coordinator;
  they were not measured here.
* **The cause, confirmed here before any change.** The test appends the final
  `\r\n` and then closes its handle (`live_twins_test.go`, `twinPasteTwoWrites`);
  `pasteFromFile` polls every `pasteInterval`, sees the newline and calls
  `os.Remove` at once. Go opens a file on Windows without `FILE_SHARE_DELETE`,
  so while the writer's handle is open `DeleteFile` fails with
  `ERROR_SHARING_VIOLATION` (32), the text above. The window is the time
  between the write and the close. A test that holds the handle open on
  purpose fails every time on the unfixed helper (the proof below), so the
  cause is the one named, not a guess. The same call was in the over-limit
  branch.
* **The options, and the owner's ruling.** Fix only the test (close the
  writer first, or have the test wait); or fix the helper, so that
  `pasteFromFile` retries the removal while the file is held open. The owner
  ruled on 2026-11-14, relayed by the coordinator; chosen "Fix the helper",
  and "Fix only the test" rejected. The ruling, verbatim: "pasteFromFile
  retries the delete while the file is still held open, until it succeeds or
  its context ends. This fixes the test and also the live sign-in probe, which
  has the same weakness when someone on Windows pastes with a command that
  holds the file open. Test-only change; proof is 100 Windows runs red before
  and green after; then the gate and both race passes run again."
* **Files added to this phase's scope:** `llmprovider/live_checks_test.go`,
  `llmprovider/live_twins_test.go`; comment lines of
  `llmprovider/auth/identity.go`, `docs/guides/migrating-from-mcplib.md` and
  `tools/README.md` for the rows and rulings below. No non-test Go line
  changed: the `identity.go` diff is `+6 -2` lines, every one a `//` comment
  line (re-checked after the review below: no changed line lacks `//`).
* **The fix.** `removePasteFile(ctx, path)` in `live_checks_test.go` replaces
  both `os.Remove` calls in `pasteFromFile` (the newline branch and the
  over-limit branch). It removes the file; when the error is a Windows sharing
  violation it tries again every `pasteInterval` until the removal succeeds
  or `ctx` ends, and then returns `ctx`'s error, wrapped; any other error is
  returned at once, as before. The error holds the path, never the paste.
  * **How "held open" is recognised, and the form chosen.**
    `fileHeldOpen(err)` is `runtime.GOOS == "windows"` and `errors.As(err,
    *syscall.Errno)` with the value `errSharingViolation syscall.Errno = 32`.
    `syscall.Errno` exists on every platform and `syscall.ERROR_SHARING_VIOLATION`
    does not, so the number is named in the file. The narrower-looking form
    is a `_windows.go` file with the real constant and a `_other.go` stub: two
    new build-tagged test files, a second place to keep in step, and the
    helper's behaviour split from its tests. The chosen form is in the two
    test files, with no build tag: it matches only the one Windows error, so a
    different removal error is still returned at once, and the `GOOS` test
    keeps it from matching the value 32 where that is another errno.
  * **The over-limit branch needs it.** Its `os.Remove` is the same call on
    the same file; a writer that has written past the limit and not closed has
    the file open in the same way. It uses `removePasteFile`.
  * **Other removals in the test tree.** `grep` for `os.Remove`,
    `os.RemoveAll` and `os.Rename` over every `*_test.go`: the two in
    `pasteFromFile`; `llmprovider/auth/example_test.go:25`, `defer
    os.RemoveAll(dir)` of the example's own temporary directory, which no
    writer holds as a paste; and a doc comment in
    `llmprovider/auth/session_store_kinds_0048_test.go` that names
    `os.Remove`. No other test removes a file a writer may hold. The three
    non-test callers (`auth/clientstore.go`, `auth/tokenstore_file.go`) are the
    stores' own and out of scope.
  * **The live probe** `TestLive_0074GrokPaste` and `live_0074_paste_test.go`
    call `pasteFromFile`, so they have the retry too.
* **The tests,** in `live_twins_test.go`, under
  `TestLiveTwin_GrokPaste_0074`: `FileHeldOpenThenClosed` (a whole line, held
  open, then closed), `OversizedHeldOpenThenClosed` (the over-limit branch the
  same way) and `HeldOpenUntilContextEnds` (held past `ctx`: the deadline
  error, no line, nothing recorded). On Windows the first two assert that
  the helper does not return while the handle is open, and then that it
  returns the line (or the no-newline error) once it is closed, with the file
  gone. Off Windows an open file can be removed, so those two prove only the
  line, or the error, and the removal; the third skips, with its reason. The
  retry's behaviour is proved on Windows only, and nowhere else.
* **Proof, the new tests red on native Windows.** A scratch clone of `b39bbec`
  with the new `live_twins_test.go` over it and `live_checks_test.go` as at
  `b39bbec` (never the tree), through `run-on-host --need windows`, 17:09:42 to
  17:09:44, exit 1:

  ```text
  --- FAIL: TestLiveTwin_GrokPaste_0074/FileHeldOpenThenClosed
      live_twins_test.go:474: pasteFromFile returned 0 bytes, paste file: remove C:\Users\<user>\AppData\Local\Temp\…\paste: The process cannot access the file because it is being used by another process. while the file was held open
  --- FAIL: TestLiveTwin_GrokPaste_0074/OversizedHeldOpenThenClosed
      live_twins_test.go:482: pasteFromFile returned 0 bytes, paste file: remove C:\Users\<user>\AppData\Local\Temp\…\paste: The process cannot access the file because it is being used by another process. while the file was held open
  --- FAIL: TestLiveTwin_GrokPaste_0074/HeldOpenUntilContextEnds
      live_twins_test.go:502: pasteFromFile = "", paste file: remove C:\Users\<user>\AppData\Local\Temp\…\paste: The process cannot access the file because it is being used by another process.; want the deadline error and no line
  ```

  On this Mac,
  with the fix, `go test -count=1 -run TestLiveTwin_GrokPaste_0074 -v
  ./llmprovider/`: the held-open tests pass and `HeldOpenUntilContextEnds`
  skips.
* **Proof, the reported form, `-count=100` of `TestLiveTwin_GrokPaste_0074`
  on native Windows, through `run-on-host --need windows`** (the `target=`
  line named the Windows laptop):

  | Run | Code | Started to ended | Exit | `FileInTwoWrites` |
  | :--- | :--- | :--- | :--- | :--- |
  | no `-race` | scratch clone of `b39bbec` | 17:09:57 to 17:10:25 | 1 | 5 failed, 95 passed |
  | `-race` | scratch clone of `b39bbec` | 17:10:25 to 17:10:56 | 1 | 4 failed, 96 passed |
  | no `-race` | the tree with the fix | 17:10:56 to 17:13:45 | 0 | 100 passed |
  | `-race` | the tree with the fix | 17:13:45 to 17:15:59 | 0 | 100 passed |

  Each red run's first failure was `live_twins_test.go:401: pasteFromFile = "",
  paste file: remove C:\Users\<user>\AppData\Local\Temp\…\paste: The process
  cannot access the file because it is being used by another process.; want
  the whole line and no error`, the reported line. The counts are lower than
  the 14 and 11 of 100 reported; the failure is by chance, and the fixed
  runs are green 100 of 100 for each of the four subtests (the three new
  ones included, none skipped on Windows), with no `FAIL` and no `DATA RACE`.
* **Docs rows this phase closes,** both from the earlier agent's open items
  and checked here:
  * `docs/guides/migrating-from-mcplib.md`, "Identity (§5)", said requests
    name `go-llmprovider-sdk` in `User-Agent`, `originator` and `referrer`. That
    is the default only: `auth.Identity` replaced `originator` and `referrer`
    on a sign-in in `v1.30.0` (0076-MADR D3, D4), and
    `llmprovider.WithClientInfo`'s name is the ChatGPT backend's `originator`
    and names the application on a session's refresh and revocation as of
    `v1.31.0` (0079-MADR D1 to D3). It says so.
  * `llmprovider/auth/identity.go`'s doc said whether OpenAI and xAI accept
    every name "is not verified". Two consumer names are verified, each in an
    owner-run probe: the first on 2026-11-11 (0076-PLAN, "Phase 4, the
    owner's live run (2026-11-11)"), the second on 2026-11-14 (0079-PLAN,
    "Phase 5, the owner's run (2026-11-14)"). OpenAI's browser sign-in and
    xAI's device sign-in completed under each; under the second, OpenAI's
    listing and generation and both vendors' refresh and revocation passed
    too. "Every name" stays unverified, and the comment says so. *(Review of
    2026-11-14: the coordinator's review found that this entry, and the
    comment, first dated both runs 2026-11-14; corrected in place.)*
* **The records.** The owner's three rulings are annotated, each verbatim: the
  flake (this entry, and the acceptance and Windows race bullets of "Phase 8,
  the release prepared"); "Pins stand" (the code-claims item of that entry,
  0075-PLAN Phase 7, 0079-PLAN Phase 6, and the note of 0079-PLAN Phase 2 that
  left rows "for the release phase to re-pin", now superseded); and "No rows
  for docs edits" (the three execution entries that said "the coordinator's
  decision": 0074-PLAN Phase 6, 0075-PLAN Phase 5, 0079-PLAN Phase 4), whose
  rule is in `tools/README.md` beside `claims.py`. No row or marker changed.
* **The changelog gets no entry.** `CHANGELOG.md` records "all notable
  changes to this module" by release; a change to two test files and to
  comments and guide prose alters nothing a consumer can observe, and
  `make changelog-check` asks only for a section per tag.
* **The gates,** each run after the last change to a Go or Markdown file but
  this bullet, each exit status captured before any filter, times local
  (2026-11-14):
  * `make pre-add-check` on the three Go files: exit 0, "3 file(s) clean",
    17:17:53 to 17:18:00. `go test -count=1 ./llmprovider/...`: exit 0,
    17:18:00 to 17:18:17.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`: exit 0,
    "gate: 30 of 30 passed", 17:17:45 to 17:30:19 (754 s).
  * `go test -race -shuffle=on -count=1 ./...` on the Linux machine, through
    `run-on-host --need linux --race`: exit 0, 42 packages `ok`, no `FAIL`
    and no `DATA RACE`, 17:17:45 to 17:20:46.
  * `go test -race -count=1 ./...` on the Windows laptop, natively, through
    `run-on-host --need windows --race -- sh -c '...'`: exit 0, 42 packages
    `ok`, no `FAIL` and no `DATA RACE`, 17:17:45 to 17:19:15.
  * The `-count=100` runs: in the proof above.
  * `make api-check`: exit 0, "against v1.30.0, 0 incompatible change(s)
    outside llmprovider/x/". `tools/claims.py` on the 0074, 0075, 0076 and
    0079 pairs: each exit 0 at its pins (0074 67 of 67 and 12 of 12, 0075 51
    of 51 and 8 of 8, 0076 48 of 48 and 7 of 7, 0079 62 of 62 and 13 of 13).
    `make records-check links-check disclosure-check markdownlint
    changelog-check docs-check`: each exit 0, 17:18:25 to 17:18:29.
  * After this bullet, the Markdown gates ran again, since it is an edit:
    `make records-check links-check disclosure-check markdownlint
    changelog-check docs-check`, `tools/recordlint.py --changed`,
    `tools/denyscan.py --changed` and `tools/claims.py` on the four pairs. The
    full gate, the two race passes and the `-count=100` runs did not run
    again: this bullet changes no input to them but the record's own text,
    which the gate's record checks read and the second run covers.
* **The coordinator's review (2026-11-14), and the gates again.** The review
  found that the `identity.go` comment and the docs-rows bullet above dated
  both verified names' runs 2026-11-14, when the first was run on 2026-11-11
  (0076-PLAN, "Phase 4, the owner's live run (2026-11-11)"), and that the
  failure rates were attributed to the coordinator, when the release-prep
  agent measured them. Both are corrected in place, above. The comment is
  now `+6 -2`, every changed line a `//` line. The `-count=100` runs did not
  repeat: neither test file changed after them (both last modified 17:09:23,
  before the first of those runs). The gates ran again after the last Go
  change (`identity.go`, 17:34:48), times local:
  * `make pre-add-check` on `llmprovider/auth/identity.go` and the two test
    files: exit 0, 17:34:56 to 17:36:22. `go test -count=1
    ./llmprovider/...`: exit 0, 17:36:22 to 17:36:43.
  * `python3 -B tools/gate.py --timeout 5400` through `run-heavy`: exit 0,
    "gate: 30 of 30 passed", 17:34:56 to 17:52:15 (1038 s).
  * `go test -race -shuffle=on -count=1 ./...` on the Linux machine, through
    `run-on-host --need linux --race`: exit 0, 42 packages `ok`, no `FAIL` and
    no `DATA RACE`, 17:34:56 to 17:38:08.
  * `go test -race -count=1 ./...` on the Windows laptop, natively, through
    `run-on-host --need windows --race -- sh -c '...'`: exit 0, 42 packages
    `ok`, no `FAIL` and no `DATA RACE`, 17:34:56 to 17:36:51.
  * `make api-check`: exit 0. `tools/claims.py` on the four pairs: each exit
    0, at the same counts as above. `make records-check links-check
    disclosure-check markdownlint changelog-check docs-check`, `tools/recordlint.py
    --changed` and `tools/denyscan.py --changed`: each exit 0,
    17:36:43 to 17:36:58.
  * After this bullet, the Markdown gates ran once more, as before: the six
    docs gates, `recordlint --changed`, `denyscan --changed` and `claims.py`.
* **Not verified:** the retry's behaviour on the live probe itself (a person
  pasting with a command that holds the file), which needs the owner; the
  `-count=100` rates on the Windows laptop at the 14 and 11 of 100 the
  release-prep agent measured (relayed by the coordinator), which this run
  did not reach.
* **Provenance.** Mainstream model, medium effort; the rulings are the
  owner's, relayed by the coordinator.

### Phase 8, the tag and the post-tag steps (2026-11-14)

Base `0fbb761`, the release commit "chore(release): prepare v1.31.0 changelog,
docs and session fixes"; `main` and `origin/main` both read `0fbb761`, and the
tree was clean before this entry. This entry changes records only: no Go file,
tool, workflow, Release or tag. Each command's output and exit status are in
the session's scratch folder, each status captured before any filter. Times
are local (UTC-6) unless marked UTC.

* **The owner's word.** On 2026-11-14: "You may tag and push this tag
  v1.31.0".
* **The tag.** The coordinator, on that word, created the annotated, unsigned
  tag `v1.31.0` (message `v1.31.0`) on `0fbb761` and pushed it at 18:54 on
  2026-11-14 (the tag's date, 18:54:56). `git ls-remote --tags origin
  'v1.31.0*'`: the tag object is `6127b41ac835d8554822eebb47ed4f0477655f77`
  and `v1.31.0^{}` is `0fbb7613e9ac2e3420542a75b2f4ebffbe83854b`, the release
  commit.
* **CI on the tag.** Run `43660548756`, event push, conclusion `success`,
  read here with `gh run view`: `validate (macos-15)` success (57 s),
  `validate (ubuntu-24.04)` success (8 m 40 s), `validate (windows-2025)`
  success (1 m 31 s), and `release` success (6 s). The one annotation is
  GitHub's notice that macOS arm64 jobs may queue longer.
* **The Release.** `gh release view v1.31.0 --json
  tagName,isDraft,isPrerelease,publishedAt`: `tagName` "v1.31.0", `isDraft`
  false, `isPrerelease` false, `publishedAt` "2026-11-15T01:03:48Z". That
  time is UTC, and falls on the next UTC date: it is 19:03:48 on 2026-11-14
  local, so the Release, the section's date (2026-11-14) and the tag share one
  local day. `gh release list` marks it Latest.
* **`tools/consumer.py v1.31.0`**, through `run-heavy -- python3 -B
  tools/consumer.py v1.31.0`: exit 0, 19:05:07 to 19:06:45 (98 s). Quoted:
  "`go get github.com/maccavelli/go-llmprovider-sdk@v1.31.0`: rc=0", "go:
  added github.com/maccavelli/go-llmprovider-sdk v1.31.0"; `go mod tidy` and
  the six platform builds (linux, windows and darwin, each amd64 and arm64)
  rc=0; `go run .` rc=0; "consumer: 17 packages linked"; "consumer: every
  step passed". (`v1.30.0`: 16 packages.)
* **`tools/offline.py v1.31.0`**, through `run-heavy -- python3 -B
  tools/offline.py v1.31.0`: exit 0, 19:06:49 to 19:06:59 (11 s). Quoted:
  "go get ... v1.31.0: rc=0", "gofmt -l main.go: clean", "go vet .: rc=0";
  "fields=19 window=1000000 output=128000 input=3 output=15 tiers=1
  source=models.dev", "requests=0", "offline: ok"; the six platform builds
  rc=0; "offline: every step passed". The tool's two built-in plants failed,
  as they must: "--- plant: models.dev left on" printed `FAIL: LookupModel:
  model metadata: Get "https://models.opencode.ai/api.json": offline: request
  refused`, "requests=1", `FAIL: the counting transport saw 1 request(s),
  want 0`, "exit status 1", then "caught: models.dev left on"; "--- plant: no
  caller layer" printed `FAIL: CachedModel: nothing known` and 38 further
  `FAIL: LookupModel|CachedModel: field N unset` lines, "fields=19 window=0
  output=0 input=0 output=0 tiers=0 source=", "requests=0", "exit status 1",
  then "caught: no caller layer".
* **R53's check.** The rule, in `docs/guides/api-standards.md` (R53) as
  0055-MADR's amendment of 2026-11-10 and 0048-MADR's A4 changed it: "Every
  release has a changelog section, and the six newest tags a GitHub Release."
  The release's PLAN "then deletes the Release of the tag that falls to
  seventh, keeping its tag and section, after saving its JSON". `gh release
  list` shows seven: `v1.31.0` (Latest), `v1.30.0`, `v1.29.0`, `v1.28.0`,
  `v1.27.0`, `v1.26.1`, `v1.26.0`. The six newest tags are `v1.31.0` to
  `v1.26.1`, so `v1.26.0`'s Release is past the sixth. Its JSON (every field
  `gh release view` offers) was saved to the scratch folder, read-only. **It
  is not deleted:** the deletion, `gh release delete v1.26.0 --yes`, keeps the
  tag (`git ls-remote --tags origin v1.26.0` lists it) and the changelog
  section, and waits for the owner's word, as `v1.25.0`'s did at `v1.30.0`.
  *(Annotation, 2026-11-14, later: the deletion was done. The coordinator
  asked the owner whether `v1.26.0`'s Release should be deleted under R53;
  the owner chose "Yes, delete it now": "I run `gh release delete v1.26.0
  --yes`, confirm six Releases remain and the v1.26.0 tag still exists, and
  record it in the 0074 plan. This is what the rule requires and what was
  done for v1.25.0 at the last release." Rejected: "I'll delete it myself";
  "Keep it". The coordinator ran `gh release delete v1.26.0 --yes` at about
  19:40 local: exit 0. Before: seven Releases, `v1.31.0` to `v1.26.0`.
  After, read again by the recording agent: `gh release list` shows six,
  `v1.31.0` (Latest), `v1.30.0`, `v1.29.0`, `v1.28.0`, `v1.27.0` and
  `v1.26.1`; `git ls-remote --tags origin v1.26.0` still lists the tag
  (`d31d1fd`); `CHANGELOG.md` still has `## [v1.26.0] - 2026-11-04`. The
  Release's JSON saved before the deletion (17 fields, body 2,512
  characters) is kept in the session's scratch folder. R53 is met: the
  six newest tags have a Release.)*
* **Other steps after the tag.** None in this phase's text beyond the tag, CI,
  the two tools and R53's check. 0075-PLAN Phase 7 and 0079-PLAN Phase 6 ask
  only that this phase's tag, CI and tools be recorded. Nothing read-only is
  left undone; the one open item is the Release deletion above. The claims
  question closed on 2026-11-14 ("Pins stand").
* **The acceptance of this PLAN, criterion by criterion.**
  * Phases 0 to 7 complete, and Phase 8 step 1 done on 2026-11-13: met, per
    their entries.
  * The changelog section is `v1.31.0` and `make changelog-check` passes:
    met (the prepared entry above; CI's `validate` jobs on the tag).
  * `tools/apidiff.py v1.30.0` lists exactly the R55 tables: met (the
    prepared entry's table).
  * The full gate passes before the tag, and the races pass on Linux and
    Windows: met, "gate: 30 of 30 passed" and both race passes, in the
    deviation entry "the Windows paste flake", at the last Go change.
  * "The tag on the owner's word": met, above.
  * "Then `tools/consumer.py` and `tools/offline.py`": met, each exit 0,
    above.
  * The tag's CI and its Release (0079-PLAN's condition): met, above.

  R53's Release deletion is no criterion of this phase's text, so it does not
  hold the status. **This PLAN is `complete`.** The MADR stays `accepted`: no
  record changes its status at the release.
* **Open for the owner:** `gh release delete v1.26.0 --yes`.
  *(Annotation, 2026-11-14, later: closed. The owner chose "Yes, delete it
  now" and the coordinator ran the command, exit 0; see the annotation under
  R53's check above. Nothing is open for the owner. Provenance of this
  annotation: a mainstream model at medium effort; the deletion by the
  coordinator, on the owner's word.)*
* **Provenance.** Recorded by a mainstream model at medium effort; the tag
  and its push by the coordinator, on the owner's word above.
