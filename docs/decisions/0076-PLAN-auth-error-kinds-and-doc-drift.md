---
status: proposed
date: 2026-11-11
associated-madr: "0076-MADR-auth-error-kinds-and-doc-drift.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement One Unsupported Kind from Revocation, an Auth-Failure Device Expiry, the Consumer's Identity on Its Sign-Ins and in the Wizard, and Four Doc Corrections

Associated MADR: [0076-MADR-auth-error-kinds-and-doc-drift.md](0076-MADR-auth-error-kinds-and-doc-drift.md)

## Goal

After this PLAN, in the next minor release (`v1.30.0` if nothing else is
tagged first):

* every revocation that cannot revoke returns an error matching
  `llmprovider.ErrUnsupported` (MADR D1);
* every device-code expiry matches `ErrAuthFailure` (D2);
* a consumer can name itself to the identity providers, in its own sign-ins
  and through the wizard (D3, D4) *(2026-11-11: "and a wizard caller can set
  each provider's OAuth client and issuer" removed by the owner's ruling)*;
* four docs say what the code does (D5), and the README shows only what the
  wizard can do (D4).

It depends on no other record of this series. *(2026-11-11: one record
outside the series' four now comes before it.
[0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md)
raises the Go floor to 1.27.4 before any code here, and this PLAN's release
ships that floor.)* The owner decided on
2026-11-10 (DF) that it lands now, ahead of the redesign, once its MADR is
accepted, so it ships before
[0073-PLAN-publish-an-auth-contract-module.md](0073-PLAN-publish-an-auth-contract-module.md)'s
tag. [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)
uses its `auth.Identity` and builds on its D2 expiry.

The MADR is `proposed`. Nothing below starts before the owner accepts it and
approves this PLAN.

## Scope

### In scope

* **D1:** `llmprovider/auth/oauth_revoke.go`.
* **D2:** `llmprovider/auth/oauth_device.go`.
* **D3:** `llmprovider/auth/oauth_loopback.go`, `oauth_device.go`,
  `oauth_plan.go`, `kilo_device.go`, `oauth_idtoken.go` (added 2026-11-11,
  MADR D3's keys' fetch), a new `llmprovider/auth/identity.go`.
* **D4:** `wizard/configure.go`, `wizard/auth.go`, `wizard/device.go`.
* **D5:** `llmprovider/auth/tokenstore_file.go`, `tokenstore.go`,
  `oauth_revoke.go`; `README.md`; `CHANGELOG.md`; the amendments the MADR
  lists.

### Exported names (R55)

| Name | Call site that needs it |
| :--- | :--- |
| `type auth.Identity` and its `Name`, `Version` | gobble's sign-ins, which run under gobble's name and version (the report's F20): its ACP sign-in sets `signin.Options.Identity` (0074-MADR D3; gobble-cli `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D3, proposed); `wizard.Options.Identity` |
| `OAuthFlowOptions.Identity` | the wizard's browser and device sign-ins (MADR D4); `x/signin`'s, which pass `signin.Options.Identity` on (0074-MADR D3) |
| `ChatGPTPlanOptions.Identity` | the wizard's plan sign-in; `x/signin`'s plan sign-in, through which gobble's ACP sign-in names itself |
| ~~`wizard.Options.OAuthClients`; `type wizard.OAuthClient` and its `ClientID`, `Issuer`~~ | *Removed 2026-11-11 by the owner's ruling (MADR D4). R55 asks for a consumer's call site, and the row had none: it gave "a wizard caller passing `auth.GrokFlowFromEnv()`'s values, as the README shows (the report's S7)", an example this PLAN was itself to write, and an open question, "no consumer's record names it". Phase 3 step 3 corrects the README instead.* |
| `wizard.Options.Identity` | gobble's terminal sign-ins, which call `wizard.ResolveCredential` (P4; 0056-MADR D1) |

No other name is exported.

*(2026-11-11: the table read "gobble's direct use of `LoginBrowserOAuth` for
Grok" and "gobble's test issuer". gobble calls neither: it signs in through
`wizard.ResolveCredential` only (P4), and no gobble-cli record names
`OAuthClients` or a test issuer. The rows now name the call sites that exist
or are proposed. Whether `OAuthClients`, with only the README's call site,
still meets R55 is the owner's question.)*

*(2026-11-11, later: the owner answered that it does not. Its row is
struck, and `auth.Identity`, `OAuthFlowOptions.Identity`,
`ChatGPTPlanOptions.Identity` and `wizard.Options.Identity` stay.)*

### Out of scope

* The identity on refresh and revocation (MADR D3).
* The vendor import of a Grok login under another client (MADR D5).
* The host id (0075-MADR D6).
* A client or issuer option in the wizard: dropped by the owner's ruling of
  2026-11-11 until a consumer's record names a call site (MADR D4).
* One kind for a `LockRefresh` whose `ctx` ends while it waits: a later
  record's, after the cut-over (MADR, "Found on review"). *(Both added
  2026-11-11.)*

## Rules for every phase

* As the "Rules for every phase" of
  [0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md](0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md):
  red at `773b84c` for a test of existing API (`tools/redgreen.py --expect
  fail --rev 773b84c`), planted breaks for a test of a new name
  (`tools/plant.py`), each failure line quoted here; pre-add; the gate;
  stage; the owner commits; stop conditions. Read that PLAN's `d62142c` as
  `773b84c` and its `v1.28.0` as `v1.29.0`. Its drift-check list of
  `APIError` test files is replaced here by: the three revocation tests
  (MADR C31 to C33), `grok_referrer_test.go`, and every other test of
  `llmprovider/auth` and `wizard`, unchanged.
* **Its stop condition "`Error()`'s text changes for an error that exists
  today"** (P6) is replaced by the third one below, since MADR D1 and D2
  change two texts on purpose.
* **Added stop conditions:**
  * any existing test needs a change;
  * an authorize URL, a request body or a User-Agent changes when no
    `Identity` is set *(2026-11-11: "and no `OAuthClients` entry" removed
    with the option)*;
  * any error text changes beyond MADR D1's wrapped part and D2's message.

*(2026-11-11: the revisions and the replaced stop condition spelled out; read
as written, 0072-PLAN's rule stopped Phase 1 at its first step.)*

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR and their index rows (`tools/records.py add`).
2. On the owner's acceptance, `tools/records.py status` sets the MADR
   `accepted` and this PLAN `in-progress`.
3. **Apply the supersession notes** the MADR lists: a dated amendment to
   `0006-MADR-subscription-auth-for-llm-providers.md` §"7. xAI OAuth client
   id", quoting `:704-706`; a note on
   `0068-MADR-five-internal-fixes-from-the-0054-audit.md` D3.
4. `python3 -B tools/gate.py --markdown`, then stage.

### Phase 1: error kinds (D1, D2)

1. `oauth_revoke.go`: the three sites wrap `llmprovider.ErrUnsupported`;
   `RevokeOAuthSession`'s doc names the plan's form and D1's note, and its
   sentence on `errors.ErrUnsupported` (`oauth_revoke.go:27-28`) becomes one
   naming `llmprovider.ErrUnsupported` for a session with no revocation and
   for a Grok or plan issuer with no `revocation_endpoint` (MADR D5).
2. `oauth_device.go`: the OpenAI poll's deadline check returns what
   `pace.wait` returns at the deadline: `errDeviceCodeExpired`, or, when
   `pace.last` is set, `fmt.Errorf("%w; the last poll failed: %w",
   errDeviceCodeExpired, pace.last)`. One unexported `devicePacer` method
   builds that error for both, so the two cannot drift.

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestRevokeUnsupportedKind_0076` | a Kilo session, a Grok session whose issuer is a test server publishing no `revocation_endpoint`, and a plan session whose issuer is another such test server (OpenAI's own issuer has a built-in endpoint, MADR C36) each give an error matching both `llmprovider.ErrUnsupported` and `errors.ErrUnsupported`; the Kilo case sends no request, and the other two send only the discovery request | red at `773b84c` (matches only the standard library's) |
| `TestOpenAIDeviceExpiryKind_0076` | with a stepped clock that passes the deadline after the pacer's check, `Wait` gives an error matching `ErrAuthFailure` | red at `773b84c` (a plain error) |
| `TestRevokeMessages_0076` | each of the three errors' text, pinned as a literal with the new wrapped part | planted: a message reworded |

*(2026-11-11: step 1 names the doc sentence to change; step 2 names the
error's exact form; the revocation test names its issuers, since a plan
session at OpenAI's own issuer never reaches C3.)*

Acceptance: the tests pass, each seen red or planted; the three existing
tests that read `errors.ErrUnsupported` pass unchanged.

### Phase 2: identity (D3, D4)

*(2026-11-11: this phase was "identity and clients". The owner's ruling
removed the clients: step 3's `Options.OAuthClients` and `OAuthClient`, step
4's `ClientID` and `Issuer`, and `TestWizardOAuthClients_0076`.)*

1. `llmprovider/auth/identity.go`: `Identity`, with its doc (a value, R51).
2. `OAuthFlowOptions.Identity` and `ChatGPTPlanOptions.Identity`, carried in
   `oauthFlowConfig`; the `originator` and both `referrer` values, and the
   User-Agent of every sign-in request, from it when `Name` is set. The
   sites: discovery (`oauth_loopback.go:321`), the token exchange (`:654`),
   `postOAuthJSON` and `postOAuthForm` (`oauth_device.go:508`, `:522`),
   Kilo's (`kilo_device.go:136`), the plan's exchange (`oauth_plan.go:302`)
   and the issuer's keys (`oauth_idtoken.go:207`, MADR C37). The refresh
   (`oauth_session.go:725`) and revocation (`oauth_revoke.go:67`) keep
   `NewIdentity("", "", "")`, and so does the discovery a revocation runs.
3. `wizard/configure.go`: `Options.Identity`, documented as MADR D4.
4. `wizard/auth.go`: `oauthFlowOptions` sets `Identity` from `Options`, and
   neither `ClientID` nor `Issuer`, as today; the plan's options take
   `Identity`.

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestIdentityUnset_Unchanged_0076` | with no `Identity`: OpenAI's authorize URL, Grok's authorize URL and device form, and each request's User-Agent equal literals pinned from `773b84c` | green at `773b84c` (proving the pins), then planted: the default `originator` read from an empty `Identity.Name` |
| `TestIdentity_Originator_0076` | `Identity{Name: "probe-app", Version: "1.2.3"}` gives `originator=probe-app` | planted: `originator` kept at the default |
| `TestIdentity_Referrer_0076` | the same name as Grok's `referrer`, on the authorize URL and the device form | planted: the device form keeps the default |
| `TestIdentity_UserAgent_0076` | discovery, the token exchange, the OpenAI and Grok device requests, Kilo's requests, the plan's exchange and the issuer's keys (at a test server's URL, which the keys' cache has not seen) each send a User-Agent beginning `probe-app/1.2.3 (` and naming `go-llmprovider-sdk/`; a revocation of the session, its discovery included, sends the default | planted: Kilo's requests keep the default; the keys' fetch keeps the default |
| ~~`TestWizardOAuthClients_0076`~~ | *Removed 2026-11-11 with `OAuthClients` (MADR D4). It asserted that a Grok entry reached Grok's sign-in and not OpenAI's, that a Kilo entry's `Issuer` was Kilo's origin, that the plan ignored the map, and that no entry gave today's options.* | — |
| `TestWizardIdentity_0076` | `Options.Identity` reaches the browser, device and plan sign-ins | planted: the plan's options omit it |

*(2026-11-11: step 2 lists its sites, the keys' fetch added after MADR D3's
revision; `TestIdentity_UserAgent_0076` covers it and revocation's
discovery.)*

Acceptance: the tests pass, each seen as stated; `grok_referrer_test.go` and
every other existing test pass unchanged; `make api-check` lists exactly the
table above.

### Phase 3: docs (D5)

1. `tokenstore_file.go`: `LockRefresh`'s doc says 60 s. It also names the
   two kinds a `ctx` that ends while it waits gives (MADR D5): an error
   matching both `ErrProviderUnavailable` and `ctx`'s, when the end is seen
   between attempts (MADR C46), and `ctx`'s error alone, when it ends during
   the pause (MADR C47). No code changes. *(Added 2026-11-11, by the owner's
   ruling: both kinds are documented now, and making them one is a later
   record's.)*
2. `tokenstore.go`: `GoString` prints `&auth.`.
3. `README.md` (MADR D4). Its "The configuration wizard" section shows
   `Options.Identity`, and says that the wizard sets no OAuth client id or
   issuer on a sign-in, so that `auth.GrokFlowFromEnv()`'s values take
   effect only on a caller's own `OAuthFlowOptions`. Its "No ambient state"
   item, which names the helper beside two others (P7), says the same of
   it. No example passes the helper's values to the wizard.
   `docs/architecture.md` needs no change: no package is added (R43).
   *(2026-11-11: the step read "shows `Options.OAuthClients` with
   `auth.GrokFlowFromEnv()`, and `Options.Identity`". The owner dropped the
   option and had the README corrected instead, so that it shows only what
   the wizard can do.)*
4. `CHANGELOG.md` `[Unreleased]`: **Added** the names above; **Fixed**
   revocation's kind, the device expiry's kind, and the four docs of MADR D5
   (`LockRefresh`'s wait and its two `ctx` kinds, `GoString`'s prefix,
   `RevokeOAuthSession`'s plan form, and 0006-MADR §7's amendment);
   **Changed** revocation's wrapped error text. The **Security** entry that
   0077-PLAN wrote there, for the Go floor, stays as it is.

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestGoStringPrefix_0076` | `fmt.Sprintf("%#v", session)` begins `&auth.OAuthSession{` | red at `773b84c` |
| `TestLockRefreshContextKinds_0076` *(added 2026-11-11)* | with the lock held by another handle: a `ctx` already done gives an error matching both `ErrProviderUnavailable` and `ctx`'s; a `ctx` that ends while `LockRefresh` pauses between attempts gives `ctx`'s error, which does not match `ErrProviderUnavailable` | green at `773b84c` (the code is unchanged, and R52 asks a test to show each kind a doc names), then planted: the pause's error wrapped as `ErrProviderUnavailable` |

`TestLockRefreshContextKinds_0076` calls existing API only: it is seen green
at `773b84c` (`redgreen.py --expect pass --rev 773b84c`), which proves the
two kinds are today's code's, and then to fail on its plant. If its second
case cannot be made to end during the pause every time without a new seam,
that is a stop. *(Added 2026-11-11, with step 1's two kinds.)*

Acceptance: the tests pass; `make docs-check changelog-check links-check
parity-check markdownlint` pass.

### Phase 4: owner-run probe

1. **The identity at the identity providers,** `TestLive_0076Identity` in
   `llmprovider/live_0076_identity_test.go` (build tag `live_gateways`),
   under `LLMPROVIDER_LIVE_0076`, with `Identity{Name: "llmprovider-probe"}`,
   a name that is no real product's, in two halves, each gated as AGENTS.md
   "Live tests" assigns the variables (`AGENTS.md:246-248`): an OpenAI
   browser sign-in, also under `LLMPROVIDER_LIVE_BROWSER_LOGIN`; and a Grok
   device sign-in, also under `LLMPROVIDER_LIVE_DEVICE_LOGIN`. It records
   whether each completes, then revokes. *Owner-run; asks first; scratch
   store; quota: none (no generation).*
2. A refusal is a deviation: the field's doc then says which provider refuses
   a custom name, and the MADR is amended. The default is unaffected.
3. `scripts/check_tables.py` passes: `LLMPROVIDER_LIVE_0076` is this record's
   variable, which that script accepts because this record names it.
4. **The checks after each sign-in** (the authorize URL's `originator`, the
   device form's `referrer`, the session's use, the revocation) live in a
   function in `llmprovider/live_checks_test.go`, run by an offline twin in
   `llmprovider/live_twins_test.go` against a fake, where each check's
   planted break is seen to fail (P5; 0046-MADR D3).
5. The Grok browser sign-in's `referrer` is not probed: the device form's is
   the same value, and one Grok sign-in is enough for the question.

*(2026-11-11: steps 4 and 5 added; AGENTS.md asks step 4 of every login
test.)*

### Phase 5: the release

As 0072-PLAN's Phase 6: the changelog section renamed for the version, the
full gate, `tools/apidiff.py v1.29.0` lists exactly the table above, the tag
on the owner's word, then `tools/consumer.py` and `tools/offline.py`. The
section carries 0077-PLAN's Security entry, and that PLAN's Phase 2 reads
the tag's Go version back. *(2026-11-11: added with 0077.)*

## Verification

Each command's exit status is captured before any filter.

* `go test -race -count=1 ./llmprovider/auth/... ./wizard/...`
* `go test -race ./...`
* `make api-check` and `python3 -B tools/apidiff.py v1.29.0`
* `make dep-check tables-check coverage-check generate-check docs-check
  changelog-check parity-check records-check fixtures-check links-check
  disclosure-check gate-selftest markdownlint`, `make lint vuln`,
  `go mod tidy -diff`
* `python3 -B tools/gate.py`
* `python3 -B tools/claims.py docs/decisions/0076-MADR-auth-error-kinds-and-doc-drift.md`
* `go vet -tags live_gateways ./...`
* Every red run and plant of Phases 1 to 3, quoted.

## Rollout and Rollback

* **Order.** First,
  [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md)
  raises the Go floor to 1.27.4: its Phase 1 lands before this PLAN's Phase
  1, the first code here, and ships in this PLAN's `v1.30.0` *(added
  2026-11-11, when the owner approved 0077)*. Then this PLAN, now, ahead of
  the redesign (the owner's DF, 2026-11-10);
  independent of 0073; before 0074, which uses `auth.Identity` and D2's
  expiry.
  gobble may then set its identity through the wizard. *(2026-11-11: "and
  its test issuer" removed; no gobble record names one.)*
* **Rollout.** Additive. A caller sees two changes it did not ask for: a
  revocation's error now also matches `llmprovider.ErrUnsupported`, with a
  different wrapped text; and a rare OpenAI device expiry now matches
  `ErrAuthFailure`. *(2026-11-11: `v1.30.0` also carries 0077's Go floor. A
  consumer at `go 1.27.3` raises its own `go` line to take it, as
  0077-MADR's Consequences say.)*
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** the names stay (R48); a provider that refuses a
  custom name is documented, and callers leave `Identity` unset for it.

## Code claims

The facts Phase 4's gating rests on, and the R55 table's gobble call site,
checked by `tools/claims.py`.

<!-- claims: self=773b84c gobble-cli=17d1605 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | The browser-login variable covers the ChatGPT and Grok browser logins | `AGENTS.md:246` | `` `LLMPROVIDER_LIVE_CHATGPT`, `LLMPROVIDER_LIVE_BROWSER_LOGIN` (the ChatGPT and `` |
| P2 | The device-login variable covers the Grok and Kilo device logins | `AGENTS.md:247` | `` Grok browser logins), `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok and Kilo `` |
| P3 | The existing Grok device live test uses it | `llmprovider/live_oauth_login_test.go:101` | `if os.Getenv("LLMPROVIDER_LIVE_DEVICE_LOGIN") != "1" {` |
| P4 | gobble signs in through the wizard | `gobble-cli:llm/provider/signin.go:176` | `res, _, err := wizard.ResolveCredential(ctx, w, wizard.Options{` |
| P5 | The checks after a login live beside their twins | `AGENTS.md:255` | `` checks after the sign-in in a function in `llmprovider/live_checks_test.go`, `` |
| P6 | 0072-PLAN's stop condition on error text | `docs/decisions/0072-PLAN-keep-absolute-reset-times-and-limit-detail-on-apierror.md:154` | `` * `Error()`'s text changes for an error that exists today; `` |
| P7 | The README's "No ambient state" item names the helper | `README.md:153` | `` itself. Pass `catalog.OptionsFromEnv()`, `auth.GrokFlowFromEnv()` or `` |

## Execution record

Not started. The MADR is `proposed`.
