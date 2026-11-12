---
status: in-progress
date: 2026-11-12
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
  facts in Phase 1 (MADR D12). Any other test that needs a change is a stop.
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
   callback did (R46: same login, same checks).

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
| `TestStart_Cancel_0074` | `Cancel` during `waiting` ends with `failed` `canceled`, and `Wait`'s error matches `context.Canceled`; no goroutine is left (checked with a goroutine count before and after) | `Cancel` does not stop the poll |
| `TestStart_VendorImport_0074` | a fixture Codex `auth.json` under a temporary `CODEX_HOME` gives `done` and a `ReadThrough` | `LookupEnv` ignored |
| `TestStart_Unsupported_0074` | `api_key`, `token_stdin` and an unknown pair are `ErrUnsupported` | `api_key` started |
| `TestErrorKindOf_0074` | every row of 0073-MADR D3, by its sentinel, in order: `ErrQuotaExhausted` is `quota` not `rate_limited`, `ErrOffline` is `offline` not `unavailable`, `ErrEmptyAnswer` is `empty_answer` not `incomplete`, `ErrSignInExpired` is `expired` not `auth`; the OpenAI device poll's own 15-minute deadline, reached with a stepped clock as 0076-PLAN's `TestOpenAIDeviceExpiryKind_0076` reaches it, is `expired` | `rate_limited` tested before `quota`; `auth` before `expired`; the poll's deadline returning 0076's plain error again (MADR C51), which maps to `other` |
| `TestCredentialRedacts_0074` | an `APIKey` with a planted key prints no key in `%v`, `%+v`, `%#v`, `%s` or `slog` | `String` returns the key |
| plants of the gates | `make dep-check` with the contract's requirement removed from `REQUIRES`; `make tables-check` with `authcontract` removed from `wizard`'s R2 row | each fails, naming the module |

Acceptance: the tests pass, each plant failed; `make dep-check tables-check
coverage-check docs-check api-check` pass; `x/signin` and
`internal/signinflow` each at 80 % or more.

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
   `TestLive_OpenAIDeviceLogin`.
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

1. **A live OpenAI device sign-in,** `TestLive_OpenAIDeviceLogin` in
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

`scripts/check_tables.py` passes: `LLMPROVIDER_LIVE_0074` is this record's
variable (AGENTS.md "Live tests").

### Phase 8: the release

With 0075-PLAN's last phase: the changelog section takes the next minor
version after 0076's (`v1.31.0` if 0076 shipped as `v1.30.0`); the full gate;
`tools/apidiff.py` against the previous tag lists exactly the tables above,
with 0075-PLAN's, `OAuthFlowOptions.NoLoopback` and
`ChatGPTPlanOptions.NoLoopback` among them *(named 2026-11-11, with the
owner's reversal)*;
the tag on the owner's word; then `tools/consumer.py` and `tools/offline.py`.

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

<!-- claims: self=773b84c -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The Grok live device test Phase 3 moves | `llmprovider/live_oauth_login_test.go:100` | `func TestLive_GrokDeviceLogin(t *testing.T) {` |
| P2 | and Kilo's | `llmprovider/live_oauth_login_test.go:124` | `func TestLive_KiloDeviceLogin(t *testing.T) {` |
| P3 | Both run under the device-login variable | `llmprovider/live_oauth_login_test.go:101` | `if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {` |
| P4 | whose AGENTS.md scope Phase 6 widens | `AGENTS.md:247` | `` `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok and Kilo `` |
| P5 | R2's sentence Phase 4 amends | `docs/guides/api-standards.md:31` | `` Only `wizard` may leave the standard library, and then only for `` |
| P6 | The OpenAI poll's own deadline, mapped `other` until 0076 D2 | `llmprovider/auth/oauth_device.go:234` | `return nil, errors.New("oauth: OpenAI device-code login timed out after 15 minutes")` |

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
