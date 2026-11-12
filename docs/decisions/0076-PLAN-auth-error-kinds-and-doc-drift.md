---
status: in-progress
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
table above. *(2026-11-11: `make api-check` cannot list it.
`scripts/check_api.py` runs apidiff with `-incompatible`, so it prints
incompatible changes only. The list of additions is read from `python3 -B
tools/apidiff.py v1.29.0`, which lists exactly the table above, and `make
api-check` shows no incompatible change. Phase 2's record has both.)*

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

The owner committed this pair, its index rows and the pairs 0073 to 0077 as
records only (`771b422`), and on 2026-11-11 accepted the MADR and approved
this PLAN.

### Phase 0 (2026-11-11)

* Step 1: done by the owner's commit `771b422`.
* Step 2: `python3 -B tools/records.py status RECORD STATUS --date
  2026-11-11` set the MADR `accepted` and this PLAN `in-progress`, each with
  its row in `docs/README.md`; each run printed "records: written;
  records-check: 0 problem(s)". The MADR's Decision Outcome records the
  acceptance and its date.
* Step 3: the two supersession notes, with no decision rewritten:
  * `0006-MADR-subscription-auth-for-llm-providers.md` §"7. xAI OAuth client
    id": a dated amendment after the section's bullets, quoting the last
    bullet. The range `:704-706` still holds at `771b422`. Its wording is
    the MADR's: true of the refresh; the import reads the default client's
    entry only.
  * `0068-MADR-five-internal-fixes-from-the-0054-audit.md` D3: a dated
    note at the end of D3. The MADR words it "`LockRefresh`'s doc now says
    60 s", which holds only after Phase 3 step 1; the note says the doc says
    40 s until Phase 3 makes it say 60 s.
* Step 4: `python3 -B tools/gate.py --markdown` exited 0, started at load
  9.3 after waiting for it to fall below 10: "gate: 11 of 11 passed"
  (parity-check, records-check, tables-check, changelog-check,
  markdownlint, links-check, disclosure-check, gate-selftest, deny-scan,
  recordlint, claims). Each also exited 0: `tools/claims.py` on the MADR,
  "48 of 48 pass", on this PLAN, "7 of 7 pass", and on 0068-MADR, "25 of
  25 pass" (0006-MADR has no claims marker); `make records-check`,
  "records-check: 162 records, 0 problem(s)"; `make links-check`,
  "links-check: 0 problem(s), 1233 relative link(s) in 174 file(s)";
  `tools/recordlint.py --changed`, "recordlint: 8 record(s); no
  rule rose"; `tools/denyscan.py --changed`, "deny-scan: 0 hit(s) in 11
  file(s)"; `tools/linkcheck.py`, "G-links: 0 problem(s)". This entry was
  added after those runs, so the record checks were run again before
  staging, each exiting 0. The changed files were then staged for the
  owner to commit.

The owner committed Phase 0 (`fab656f`), and 0077-PLAN's Go floor
(`ef432b7`), before Phase 1.

### Phase 1 (2026-11-11)

Started at `6feb801`, on `main`, with a clean tree and `go 1.27.4`. The
code of `llmprovider/` and `wizard/` is the same at `6feb801` as at
`773b84c` (`git diff --stat 773b84c HEAD -- llmprovider wizard` lists
`go.mod` only), and `tools/claims.py` on the MADR, "48 of 48 pass", and on
this PLAN, "7 of 7 pass", exited 0 before any change.

* **Step 1, `oauth_revoke.go`:** the three sites (MADR C1 to C3) wrap
  `llmprovider.ErrUnsupported`; their own words are unchanged.
  `RevokeOAuthSession`'s doc names the plan session's form, checked first,
  and its sentence on `errors.ErrUnsupported` is replaced by one naming
  `llmprovider.ErrUnsupported` for a session of a provider with no
  revocation and for a Grok or plan issuer with no `revocation_endpoint`,
  with D1's note: for an issuer, the error comes after its discovery
  document is read, and no token is sent.
* **Step 2, `oauth_device.go`:** one new unexported method,
  `(*devicePacer).expired`, returns `errDeviceCodeExpired`, or
  `fmt.Errorf("%w; the last poll failed: %w", errDeviceCodeExpired,
  p.last)` when `p.last` is set. `devicePacer.wait` returns it at the
  deadline, as its own code did, and the OpenAI poll's deadline check
  returns it in place of the plain "oauth: OpenAI device-code login timed
  out after 15 minutes". No exported name changed.
* **Tests,** in a new `llmprovider/auth/error_kinds_0076_test.go`. Each
  passes.
  * `TestRevokeUnsupportedKind_0076`: a Kilo session (a recording transport,
    no request), and a Grok and a plan session each at its own `httptest`
    issuer whose discovery document names no `revocation_endpoint` and
    whose other endpoints are on its origin. Each error matches both
    sentinels; the Kilo revocation sends nothing, and the other two send
    `GET /.well-known/openid-configuration` only.
  * `TestRevokeMessages_0076`: the three texts, the issuer's URL replaced
    by `ISSUER`: "oauth: revoke: kilo sessions have no revocation:
    llmprovider: unsupported" and, for Grok and the plan, "oauth: revoke:
    ISSUER publishes no revocation_endpoint: llmprovider: unsupported".
  * `TestOpenAIDeviceExpiryKind_0076`: the package's clock seam
    (`OAuthFlowOptions.now` and `sleep`, unexported) with a stepped clock
    that moves on a second at every read and by the whole duration at
    every sleep, and an 896 s interval. Its reads are T0 (the deadline is
    T0+900s), the loop's check at T0+1s, the pacer's remaining time at
    T0+2s, its check after the sleep at T0+899s, and the loop's next check
    at T0+900s: the deadline passes between the pacer's check (C8) and the
    loop's (C5), as MADR "Where the report differs from the code" says. Two
    cases: a pending poll (404), whose error ends "device code expired",
    and a failed poll (503), whose error ends "device code expired; the
    last poll failed: oauth: OpenAI device poll failed: Service
    Unavailable". Each matches `ErrAuthFailure`, after one poll and one
    896 s sleep.
* **Red at `773b84c`:** `python3 -B tools/redgreen.py SPEC.json --expect
  fail --rev 773b84c`, with the new test file copied in, exited 0:
  "redgreen: 2 of 2 fail as expected". The failure lines (the loopback
  ports left out):
  * `error_kinds_0076_test.go:89: a Kilo session: RevokeOAuthSession = oauth: revoke: kilo sessions have no revocation: unsupported operation; want an error matching llmprovider.ErrUnsupported and errors.ErrUnsupported`
  * `error_kinds_0076_test.go:89: a Grok session: RevokeOAuthSession = oauth: revoke: http://127.0.0.1:… publishes no revocation_endpoint: unsupported operation; want an error matching llmprovider.ErrUnsupported and errors.ErrUnsupported`
  * `error_kinds_0076_test.go:89: a plan session: RevokeOAuthSession = oauth: revoke: http://127.0.0.1:… publishes no revocation_endpoint: unsupported operation; want an error matching llmprovider.ErrUnsupported and errors.ErrUnsupported`
  * `error_kinds_0076_test.go:203: a pending poll: Wait = oauth: OpenAI device-code login timed out after 15 minutes; want an error matching llmprovider.ErrAuthFailure`
  * `error_kinds_0076_test.go:203: a failed poll: Wait = oauth: OpenAI device-code login timed out after 15 minutes; want an error matching llmprovider.ErrAuthFailure`

  The device lines carry the 15-minute text, which only the loop's check
  returns, so the stepped clock reaches C5 at `773b84c`. At `773b84c` the
  revocation test's request assertions passed: the three sites already
  sent nothing but discovery.
* **Green:** `tools/redgreen.py SPEC.json --expect pass` on the tree, the
  three tests, exited 0: "redgreen: 3 of 3 pass as expected".
* **Planted breaks:** `python3 -B tools/plant.py SPEC.json`, run
  `^TestRevokeMessages_0076$`, on a scratch clone, exited 0: "plant: all 3
  plant(s) caught". Each plant rewords one message:
  * "sessions have no revocation" as "sessions cannot be revoked":
    `error_kinds_0076_test.go:117: a Kilo session: Error() = "oauth: revoke: kilo sessions cannot be revoked: llmprovider: unsupported"; want "oauth: revoke: kilo sessions have no revocation: llmprovider: unsupported"`
  * Grok's "publishes no revocation_endpoint" as "has no
    revocation_endpoint":
    `error_kinds_0076_test.go:117: a Grok session: Error() = "oauth: revoke: ISSUER has no revocation_endpoint: llmprovider: unsupported"; want "oauth: revoke: ISSUER publishes no revocation_endpoint: llmprovider: unsupported"`
  * the plan's, the same:
    `error_kinds_0076_test.go:117: a plan session: Error() = "oauth: revoke: ISSUER has no revocation_endpoint: llmprovider: unsupported"; want "oauth: revoke: ISSUER publishes no revocation_endpoint: llmprovider: unsupported"`
* **The drift check:** before any change, `go test -count=1
  ./llmprovider/auth/... ./wizard/...` exited 0. After it, the three tests
  that read `errors.ErrUnsupported` (MADR C31 to C33),
  `grok_referrer_test.go` and every other test of `llmprovider/auth` and
  `wizard` pass unchanged; no existing test file was edited. No test
  pinned "unsupported operation" or the 15-minute text.
* **Race:** `go test -race -count=1 ./llmprovider/auth/... ./wizard/...`,
  exit 0; `go test -race -count=20 -run '_0076$' ./llmprovider/auth/`,
  exit 0.
* **Pre-add:** `make pre-add-check` on the three Go files, exit 0:
  "go-precheck: 3 file(s) clean (gofmt, golangci-lint, go vet, go test,
  govulncheck)". `revive`'s rules stand in for `golint` there (AGENTS.md).
* **The gate:** `python3 -B tools/gate.py`, started at load 8.9 after
  waiting for it to fall below 10, exit 0: "gate: 29 of 29 passed".
  `api-check`: "against v1.29.0, 0 incompatible change(s) outside
  llmprovider/x/"; `coverage-check`: `llmprovider/auth` 89.6 % against
  80.0 %, `wizard` 93.8 % against 86.4 %; `docs-check`: "sdkaudit: 0
  violation(s) of R50,R51,R52"; no data race in `test-race-cover`.
* **`tools/apidiff.py v1.29.0`:** exit 0, an empty report: no exported
  change, as this phase adds none.
* **Stop conditions:** none met. No existing test changed; no authorize
  URL, request body or User-Agent changed; the only text changes are D1's
  wrapped part, "unsupported operation" to "llmprovider: unsupported", and
  D2's message; every MADR claim held at the base; every plant was caught;
  `-race` reported no race.
* **Not done:** Phases 2 to 5. `Wait`'s doc is unchanged: D2 makes it true
  on every path.
* This entry was added after the gate ran, so the record checks were run
  again before staging.

The owner committed Phase 1 (`bc4bb21`) before Phase 2.

### Phase 2 (2026-11-11)

Started at `bc4bb21`, on `main`, five commits ahead of `origin/main`, with a
clean tree. Step 2's sites were read before any change. Each does what the
step says; three moved with Phase 1's code, which added lines above them:
`postOAuthJSON`'s and `postOAuthForm`'s User-Agent lines are
`oauth_device.go:518` and `:532` (`:508` and `:522` at `773b84c`), and the
revocation's is `oauth_revoke.go:75` (`:67`). The others are where the step
says.

* **Step 1, `llmprovider/auth/identity.go`:** `type Identity struct{ Name,
  Version string }`, documented as a comparable value, "safe for concurrent
  use while no goroutine changes it" (R51), with what `Name` sets, that the
  zero value names this module, that a refresh and a revocation name this
  module, and that the vendors' acceptance is not verified. Two unexported
  methods: `clientName`, `Name` or `originatorValue`; and `setUserAgent`,
  `transport.NewIdentity(Name, Version, "")` when `Name` is set, else
  `transport.NewIdentity("", "", "")`, so that a `Version` without a `Name`
  changes nothing.
* **Step 2:** `OAuthFlowOptions.Identity` and `ChatGPTPlanOptions.Identity`,
  each documented; `oauthFlowConfig.identity`, set by
  `resolveOAuthFlowConfig`, `resolvePlanConfig` and `startKiloDevice` (whose
  doc now names `Identity` among the options Kilo reads). The `originator`
  and both `referrer` values come from `clientName`; discovery, the token
  exchange, the OpenAI and Grok device requests, Kilo's requests, the plan's
  exchange and the keys' fetch set the User-Agent from the identity. The
  keys' fetch takes it through `idTokenCheck.identity` and `issuerKey`. The
  plan's discovery takes it through a new `planEndpointsFor(ctx, config)`,
  which `LoginChatGPTPlan` calls. The refresh (`oauth_session.go:725`) and
  the revocation (`oauth_revoke.go:75`) are unchanged, and so are the
  discoveries a revocation runs: Grok's builds its configuration with no
  identity, and the plan's calls `planOAuthEndpoints(ctx, client, issuer)`,
  now `planEndpointsFor` with no identity. Existing tests call
  `postOAuthJSON`, `postOAuthForm`, `fetchJWKS` and `planOAuthEndpoints` with
  their present signatures, so each keeps it and sends this module's
  identity; the sign-ins call `postOAuthJSONAs`, `postOAuthFormAs` and
  `fetchJWKSAs`, which take the identity. No existing test changed.
* **Step 3, `wizard/configure.go`:** `Options.Identity auth.Identity`,
  documented as MADR D4: every sign-in the wizard runs, the browser,
  device-code and plan sign-ins, Kilo's included; the zero value names this
  module; the wizard sets no OAuth client id and no issuer, so
  `auth.GrokFlowFromEnv`'s values do not reach one.
* **Step 4, `wizard/auth.go`:** `oauthFlowOptions` sets `HTTPClient` and
  `Identity`, and neither `ClientID` nor `Issuer`; the plan's options take
  `Identity`. Kilo's device sign-ins, as a session and as a key, use
  `oauthFlowOptions` too. `wizard/device.go` needed no change: it passes on
  the options it is given.
* **Tests,** in a new `llmprovider/auth/identity_0076_test.go` and
  `wizard/identity_0076_test.go`. The auth tests run six sign-ins, OpenAI's
  and Grok's browser and device sign-ins, Kilo's device sign-in and the
  ChatGPT plan's, each at its own `httptest` issuer that records every
  request's method, path and User-Agent. Its keys are at a path new for each
  issuer, so the keys' cache (MADR C38) has not seen the URL and each
  sign-in fetches them. Each passes.
  * `TestIdentityUnset_Unchanged_0076`: with no `Identity`, every request of
    the six sign-ins, in order, carries "go-llmprovider-sdk/SDK (GOOS;
    GOARCH) go-llmprovider-sdk/SDK", where SDK is the build's version of this
    module; OpenAI's and Grok's authorize URLs, with the issuer's URL, the
    loopback port, the state, the challenge and the nonce replaced, equal
    pinned literals, and so does Grok's device form,
    "client_id=test-client&referrer=go-llmprovider-sdk&scope=openid+profile+email+offline_access+grok-cli%3Aaccess+api%3Aaccess".
  * `TestIdentity_Originator_0076`: with `Identity{Name: "probe-app",
    Version: "1.2.3"}`, OpenAI's authorize URL carries
    `originator=probe-app`.
  * `TestIdentity_Referrer_0076`: the same name is Grok's `referrer` on the
    authorize URL and on the device form.
  * `TestIdentity_UserAgent_0076`: every request of the six sign-ins, the
    keys' fetch included, carries a User-Agent beginning "probe-app/1.2.3 ("
    and naming "go-llmprovider-sdk/"; a revocation of the Grok and the plan
    sessions sends its discovery and the revocation, each under this
    module's User-Agent only.
  * `TestWizardIdentity_0076`: `Options.Identity` reaches the browser
    sign-in (Grok), the device sign-ins (Grok's, and Kilo's as a key) and
    the plan's; the browser and device sign-ins get no `ClientID` and no
    `Issuer`.
* **Green at `773b84c`:** `TestIdentityUnset_Unchanged_0076` names no new
  identifier. `python3 -B tools/redgreen.py SPEC.json --expect pass --rev
  773b84c`, with the new auth test file copied in and the three tests that
  name `Identity` dropped (`TestIdentity_Originator_0076`,
  `TestIdentity_Referrer_0076`, `TestIdentity_UserAgent_0076`), exited 0:
  "redgreen: 1 of 1 pass as expected", with
  `--- PASS: TestIdentityUnset_Unchanged_0076 (0.05s)`. The pins are what
  `773b84c` sends.
* **Green:** `tools/redgreen.py SPEC.json --expect pass` on the tree, the
  five tests, exited 0: "redgreen: 5 of 5 pass as expected".
* **Planted breaks:** `python3 -B tools/plant.py SPEC.json`, one spec per
  test, each run on its test alone on a scratch clone; each exited 0, "plant:
  all N plant(s) caught". The failure lines (`plant.py` cuts a long line):
  * `TestIdentityUnset_Unchanged_0076`, the default `originator` read from
    an empty `Identity.Name` (`config.identity.Name` for
    `config.identity.clientName()`):
    `identity_0076_test.go:284: OpenAI browser: authorize URL ISSUER/oauth/authorize?client_id=test-client&code_challenge=CODE_CHALLENGE&code_challenge_method=S256&codex_cli_simplified_flow=true&id_token_add_organizations=true&originator=&redire`
  * `TestIdentity_Originator_0076`, `originator` kept at the default
    (`originatorValue`):
    `identity_0076_test.go:314: originator = ["go-llmprovider-sdk"]; want ["probe-app"]`
  * `TestIdentity_Referrer_0076`, the device form keeps the default:
    `identity_0076_test.go:342: Grok device: referrer = ["go-llmprovider-sdk"]; want ["probe-app"]`
  * `TestIdentity_UserAgent_0076`, Kilo's requests keep the default
    (`Identity{}.setUserAgent(req)` in `kiloDeviceRequest`):
    `identity_0076_test.go:370: Kilo device: POST /api/device-auth/codes sent User-Agent "go-llmprovider-sdk/(devel) (darwin; arm64) go-llmprovider-sdk/(devel)"; want the probe's first, then this module`,
    and the same for `GET /api/device-auth/codes/KILO-1234`.
  * `TestIdentity_UserAgent_0076`, the keys' fetch keeps the default (the
    same in `fetchJWKSAs`):
    `identity_0076_test.go:370: OpenAI browser: GET /keys sent User-Agent "go-llmprovider-sdk/(devel) (darwin; arm64) go-llmprovider-sdk/(devel)"; want the probe's first, then this module`,
    and the same for Grok's browser, OpenAI's and Grok's device, and the
    plan's sign-ins.
  * `TestWizardIdentity_0076`, the plan's options omit it:
    `--- FAIL: TestWizardIdentity_0076/plan (0.00s)` and
    `identity_0076_test.go:83: plan sign-ins given [{AgentName:test-app HostID: Previous:OAuthSession(nil) HTTPClient:<nil> OpenURL:0x101200c40 InputCode:<nil> Identity:{Name: Version:} issuer: now:<nil>}]; want one, with {Name:probe-app Version`
* **The drift check:** after the code change and before any new test,
  `go test -count=1 ./llmprovider/auth/... ./wizard/...` exited 0.
  `grok_referrer_test.go` and every other test of `llmprovider/auth` and
  `wizard` pass unchanged; no existing test file was edited.
* **Race:** `go test -race -count=1 ./llmprovider/auth/... ./wizard/...`,
  exit 0; `go test -race -count=20 -run '_0076$' ./llmprovider/auth/
  ./wizard/`, exit 0; no data race reported.
* **Pre-add:** `make pre-add-check` on the ten Go files, exit 0:
  "go-precheck: 10 file(s) clean (gofmt, golangci-lint, go vet, go test,
  govulncheck)".
* **The gate:** `python3 -B tools/gate.py`, which waited at load 126.3 and
  started at load 9.3, exit 0: "gate: 29 of 29 passed". `api-check`:
  "against v1.29.0, 0 incompatible change(s) outside llmprovider/x/";
  `coverage-check`: `llmprovider/auth` 89.7 % against 80.0 %, `wizard`
  93.8 % against 86.4 %; `docs-check`: "sdkaudit: 0 violation(s) of
  R50,R51,R52"; no data race in `test-race-cover`.
* **The additions:** `make api-check`, exit 0, prints the line above and no
  addition: `scripts/check_api.py` runs apidiff with `-incompatible`, so it
  lists incompatible changes only, as in Phase 1 and in 0072-PLAN. The
  additions are `tools/apidiff.py`'s. `python3 -B tools/apidiff.py v1.29.0`,
  exit 0, "Compatible changes:":
  * `./llmprovider/auth.ChatGPTPlanOptions.Identity: added`
  * `./llmprovider/auth.Identity: added`
  * `./llmprovider/auth.OAuthFlowOptions.Identity: added`
  * `./wizard.Options.Identity: added`

  These are the R55 table's rows after the owner's ruling, `Identity`'s
  `Name` and `Version` with its type, and nothing else.
* **Stop conditions:** none met. No existing test changed; with no
  `Identity`, no authorize URL, request body or User-Agent changed (the
  unset test, green at `773b84c` and in the tree); no error text changed;
  `tools/claims.py` on the MADR, "48 of 48 pass", and on this PLAN, "7 of 7
  pass"; every plant was caught; `-race` reported no race.
* **Not done:** Phases 3 to 5: the README and `CHANGELOG.md` entries for
  these names are Phase 3's. The PLAN's acceptance and MADR Confirmation
  read "`make api-check` lists exactly the table"; the tool lists no
  addition (above), so the list is `tools/apidiff.py`'s. That wording is for
  the owner to amend or keep; no code depends on it.
* This entry was added after the gate ran, so the record checks were run
  again before staging.

The owner committed Phase 2 (`f98f982`) before Phase 3.

### Phase 3 (2026-11-11)

Started at `f98f982`, on `main`, six commits ahead of `origin/main`, with a
clean tree. `tokenstore_file.go` and `tokenstore.go` are the same at
`f98f982` as at `773b84c`; `LockRefresh`'s two `ctx` returns (MADR C46, C47)
and `GoString` (C25) were read before any change. Each command's output and
exit status are in the session's scratch directory, each status captured
before any filter.

* **Step 1, `tokenstore_file.go`:** `LockRefresh`'s doc says "A waiter
  gives up after 60 s", cites 0068-MADR D3 beside 0016-MADR amendment A2,
  and adds: "A ctx that ends while it waits gives one of two kinds
  (0076-MADR D5): seen ended between attempts, an error matching both
  ErrProviderUnavailable and ctx's error; ended during the pause between
  attempts, ctx's error alone, which does not match
  ErrProviderUnavailable." No code changed; the two returns are now
  `tokenstore_file.go:281` and `:293`, four lines lower with the doc.
* **Step 2, `tokenstore.go`:** `GoString` returns `"&auth." + s.String()`.
  No test pinned `&llmprovider.`: the two tests that format a session with
  `%#v` (`redaction_test.go` and `plan_session_0043_test.go`) check what it
  hides and shows, not its prefix.
* **Step 3, `README.md`:** a new item in "The configuration wizard":
  `Options.Identity` names the app on every sign-in the wizard runs, as
  `auth.OAuthFlowOptions.Identity` does on the caller's own, and a refresh
  and a revocation still name this module; an example passes
  `Identity: auth.Identity{Name: "my-app", Version: "1.4.0"}` to
  `wizard.ConfigureLLM`; and the wizard sets no OAuth client id or issuer
  on a sign-in, so `auth.GrokFlowFromEnv()`'s values take effect only on
  the caller's own `auth.OAuthFlowOptions`, given to
  `auth.LoginBrowserOAuth` or `auth.StartDeviceOAuth`. The "No ambient
  state" item says the same of the helper. No example passes the helper's
  values to the wizard. The repository has no check that compiles the
  README's examples, so the example was compiled in a scratch module
  outside the tree, required through a `replace` of this module to the
  working tree: `go vet` exited 0, and exited 1 on a copy with the field
  misspelled ("unknown field Identities in struct literal of type
  wizard.Options").
* **Step 4, `CHANGELOG.md` `[Unreleased]`:** **Added** `auth.Identity`,
  `OAuthFlowOptions.Identity`, `ChatGPTPlanOptions.Identity` and
  `wizard.Options.Identity`; **Changed** revocation's wrapped text;
  **Fixed** revocation's kind, the device expiry's kind, and the four docs
  (`LockRefresh`'s wait and its two `ctx` kinds, `GoString`'s prefix,
  `RevokeOAuthSession`'s plan form, 0006-MADR §7's amendment). They sit
  above the **Security** entry 0077-PLAN wrote, which is unchanged.
* **The coordinator's correction to Phase 2's wording,** records only: a
  dated note beside Phase 2's acceptance line here, and beside the MADR's
  Confirmation item, that the list of additions is read from `python3 -B
  tools/apidiff.py v1.29.0` and `make api-check` shows no incompatible
  change. The original wording stands.
* **Tests,** in a new `llmprovider/auth/doc_fixes_0076_test.go`. Each
  passes.
  * `TestGoStringPrefix_0076`: `fmt.Sprintf("%#v", session)` begins
    `&auth.OAuthSession{`.
  * `TestLockRefreshContextKinds_0076`: with the lock held by another
    handle on the same directory, a `ctx` already cancelled gives an error
    matching both `ErrProviderUnavailable` and `context.Canceled`; and a
    `ctx` that ends during the pause gives `context.Canceled` itself, which
    does not match `ErrProviderUnavailable`. The second case uses no seam:
    its `ctx` is a test type that ends when it is first asked for `Done`.
    `LockRefresh` reads `Err` between attempts, then asks for `Done` only in
    the pause's `select`, so the context is live at the check and ended in
    the pause on every run. The test asserts both: at least one live `Err`
    read, and the context ended. No unexported hook was needed; the
    package's one for `LockRefresh`, `FileTokenStore.wait`, bounds the
    whole wait and cannot place an end in the pause.
* **Red at `773b84c`:** `python3 -B tools/redgreen.py SPEC.json --expect
  fail --rev 773b84c`, the new test file copied in, exited 0: "redgreen: 1
  of 1 fail as expected", with
  `doc_fixes_0076_test.go:19: %#v = &llmprovider.OAuthSession{Provider:grok Issuer:https://auth.x.ai ClientID:client-visible AccountID: FedRAMP:false TokenURL: Expiry:none Access: Refresh:}; want it to begin &auth.OAuthSession{`.
* **Green at `773b84c`:** the same spec for
  `TestLockRefreshContextKinds_0076`, `--expect pass --rev 773b84c`,
  exited 0: "redgreen: 1 of 1 pass as expected", with
  `--- PASS: TestLockRefreshContextKinds_0076/seen_between_attempts` and
  `--- PASS: TestLockRefreshContextKinds_0076/during_the_pause`. The two
  kinds are the code's before this phase.
* **Green:** `tools/redgreen.py SPEC.json --expect pass` on the tree, the
  two tests, exited 0: "redgreen: 2 of 2 pass as expected".
* **Deterministic:** `go test -race -count=50 -run
  '^TestLockRefreshContextKinds_0076$' -v ./llmprovider/auth/`, exit 0: 50
  of 50 `--- PASS` for the test and for `during_the_pause`, no `--- FAIL`,
  no data race.
* **Planted breaks:** `python3 -B tools/plant.py SPEC.json`, run
  `^TestLockRefreshContextKinds_0076$`, on a scratch clone, exited 0:
  "plant: all 2 plant(s) caught".
  * The pause's error wrapped as `ErrProviderUnavailable` (the plan's
    plant):
    `doc_fixes_0076_test.go:102: LockRefresh = llmprovider: provider unavailable: context canceled; want context.Canceled alone, not matching llmprovider.ErrProviderUnavailable`
  * The between-attempts error made `ctx`'s alone (added, for the first
    case):
    `doc_fixes_0076_test.go:84: LockRefresh = context canceled; want an error matching both llmprovider.ErrProviderUnavailable and context.Canceled`
* **Acceptance:** `make docs-check`, "sdkaudit: 0 violation(s) of
  R50,R51,R52"; `make changelog-check`, "42 release section(s), 0
  problem(s)"; `make links-check`, "0 problem(s), 1242 relative link(s) in
  174 file(s)"; `make parity-check`, "409 identifiers, 409 rows, 409 with
  an SDK equivalent, 0 problem(s)"; `make markdownlint`, "0 issues in 0
  files". Each exited 0.
* **The drift check and race:** `go test -race -count=1
  ./llmprovider/auth/... ./wizard/...`, exit 0, no data race; no existing
  test file was edited.
* **Pre-add:** `make pre-add-check` on the three Go files, exit 0:
  "go-precheck: 3 file(s) clean (gofmt, golangci-lint, go vet, go test,
  govulncheck)".
* **The gate:** `python3 -B tools/gate.py`, which waited at load 55.8 and
  started at load 9.4, exit 0: "gate: 29 of 29 passed". `api-check`:
  "against v1.29.0, 0 incompatible change(s) outside llmprovider/x/";
  `coverage-check`: `llmprovider/auth` 89.8 % against 80.0 %, `wizard`
  93.8 % against 86.4 %; `docs-check`: "sdkaudit: 0 violation(s) of
  R50,R51,R52"; no data race in `test-race-cover`.
* **Records:** `tools/claims.py` on the MADR, "48 of 48 pass", and on this
  PLAN, "7 of 7 pass"; `make records-check`, "162 records, 0 problem(s)";
  `tools/recordlint.py --changed`, "no rule rose"; `tools/denyscan.py
  --changed`, "0 hit(s)"; `tools/linkcheck.py`, "0 problem(s)". Each exited
  0.
* **Stop conditions:** none met. No existing test changed; no error text
  changed; every claim held; every plant was caught; `-race` reported no
  race; the second case ends in the pause on every run with no seam.
* **Not done:** Phases 4 and 5.
* This entry was added after the gate ran, so the record checks were run
  again before staging.

The owner committed Phase 3 (`3f77944`) before Phase 4.

### Phase 4, prepared (2026-11-11)

Started at `3f77944`, on `main`, seven commits ahead of `origin/main`, with a
clean tree. This entry writes the probe and verifies it offline only: no run
set a live variable, nothing signed in, and no request left the machine. The
owner runs the probe (below) and records what it finds. Each command's output
and exit status are in the session's scratch directory, each status captured
before any filter.

* **Step 1, `llmprovider/live_0076_identity_test.go`** (build tag
  `live_gateways`): `TestLive_0076Identity`, which skips unless
  `LLMPROVIDER_LIVE_0076=1`, with `auth.Identity{Name: "llmprovider-probe"}`,
  in two subtests, each gated as AGENTS.md "Live tests" assigns:
  * `OpenAIBrowser`, also under `LLMPROVIDER_LIVE_BROWSER_LOGIN`:
    `LoginBrowserOAuth` for OpenAI with the identity. Its `OpenURL` checks
    the authorize URL's `originator` before it logs the URL for the person
    to open, and on a wrong value ends the sign-in through `cancel` before
    anyone signs in, as `TestLive_ChatGPTBrowserLogin` does.
  * `GrokDevice`, also under `LLMPROVIDER_LIVE_DEVICE_LOGIN`:
    `LoginDeviceOAuth` for Grok with the identity, its requests through
    `identityRecorder`, logging the verification URL and the code for the
    person, as `TestLive_GrokDeviceLogin` does; then the device form's
    `referrer` is checked.
  * Each half logs `FINDING sign-in: … under "llmprovider-probe" completed`,
    or `… did not complete:` with the error and stops; then it runs the
    checks below. A session that a check stopping early leaves unrevoked is
    revoked when the half ends. Each sign-in has five minutes, as the
    existing login tests give. The only store is a `FileTokenStore` in
    `t.TempDir()`. Quota: none; the session's use is a model listing with no
    probe, and nothing is generated.
* **Step 4, the checks,** in `llmprovider/live_checks_test.go` (untagged),
  appended; no existing function changed:
  * `checkAuthorizeIdentity(t, raw, param, name)`: the authorize URL
    carries `name` as its one `param`. It fails with `t.Errorf` only, since
    `OpenURL` runs on a goroutine of its own.
  * `identityRecorder`, an `http.RoundTripper` that keeps the `referrer`
    values of each form posted with one, and nothing else of any request;
    and `checkDeviceReferrer`: exactly one such form, Grok's device form,
    whose `referrer` is the name alone.
  * `checkIdentitySession`: the session is saved to a scratch
    `FileTokenStore` and bound to it, so a refresh saves there, then
    `catalog.ListRaw` lists its provider's models with it (no probe, no
    listing cache, no models.dev, no static fallback, errors returned),
    which must hold a model; it logs the count only.
  * `checkIdentityRevoked`: `RevokeOAuthSession`, then a forced refresh,
    which must be refused; its kind is logged, as `planRevoke` does.
* **The twin,** `TestLiveTwin_Identity_0076` in
  `llmprovider/live_twins_test.go` (untagged), against a new `identityFake`;
  `twinFake` and the existing twins are unchanged. `OpenAIBrowser` checks
  the fake's authorize URL, then uses and revokes a ChatGPT session whose
  token endpoint is the fake's, listing under `/chatgpt`. `GrokDevice`
  starts a device sign-in at the fake through `identityRecorder`, as the
  live half does, cancels it, checks the form posted, then uses and revokes
  a Grok session whose issuer is the fake, listing under `/xai`. **What the
  twin does not cover:** the OpenAI authorize URL it checks is built by the
  fake, in OpenAI's shape, not by the SDK. OpenAI's browser sign-in listens
  on the registered ports 1455 and 1457, which no offline test of this
  package takes (`llmprovider/auth`'s tests move them, an unexported
  seam). Phase 2's `TestIdentity_Originator_0076` pins that the SDK puts
  the name there; the twin proves the check catches a URL without it. The
  Grok device form the twin checks is the SDK's own.
* **Planted breaks:** `python3 -B tools/plant.py SPEC.json`, run
  `^TestLiveTwin_Identity_0076$` on `./llmprovider/`, on a scratch clone,
  exited 0: "plant: all 4 plant(s) caught". One plant a check:
  * the authorize URL carries the default originator (the fake's
    `originator` set to `"go-llmprovider-sdk"`):
    `live_twins_test.go:301: authorize URL originator = ["go-llmprovider-sdk"], want ["llmprovider-probe"]`
  * the Grok device form keeps the default referrer (`originatorValue` for
    `config.identity.clientName()` in `oauth_device.go`'s device form):
    `live_twins_test.go:321: device form referrers = [["go-llmprovider-sdk"]], want one form with ["llmprovider-probe"]`
  * each listing holds no model (`listed: false`):
    `live_twins_test.go:307: FINDING use: openai listed 0 models with the session, <nil>; want one or more`,
    and the same for grok at `:325`
  * a revocation leaves the refresh token alive (`revokeKills: false`):
    `live_twins_test.go:308: FINDING revocation: a refresh after revocation succeeded`,
    and the same at `:326`
* **The twin and race:** `go test -count=1 -run 'Twin|0076'
  ./llmprovider/...`, exit 0; `go test -race -count=20 -run TestLiveTwin
  ./llmprovider/`, exit 0, no data race.
* **Compiled, not run:** `go vet -tags live_gateways ./llmprovider/...`,
  exit 0; `go build -tags live_gateways ./...`, exit 0.
* **The gate holds:** with no live variable set, `go test -tags
  live_gateways -count=1 -run TestLive_0076Identity -v ./llmprovider/`,
  exit 0, with
  `live_0076_identity_test.go:30: LLMPROVIDER_LIVE_0076 unset: this signs in at OpenAI and xAI under a custom name, with a person`
  and `--- SKIP: TestLive_0076Identity (0.00s)`.
* **Step 3:** `python3 -B scripts/check_tables.py`, exit 0,
  "check-tables: 42 packages, 0 problem(s)". The script accepts
  `LLMPROVIDER_LIVE_0076` because a record numbered 0076 names it as a
  whole word; this PLAN already does, in steps 1 and 3, so nothing was added
  for it. The test's other two variables are named in AGENTS.md.
* **Pre-add:** `make pre-add-check` on the three Go files, exit 0:
  "go-precheck: 3 file(s) clean (gofmt, golangci-lint, go vet, go test,
  govulncheck)".
* **The gate:** `python3 -B tools/gate.py`, which waited at load 23.2 and
  started at load 9.9, exit 0: "gate: 29 of 29 passed", the three
  `vet-*-live` steps included. `api-check`: "against v1.29.0, 0
  incompatible change(s) outside llmprovider/x/"; `coverage-check`:
  `llmprovider` 98.5 % against 95.9 %, `llmprovider/auth` 89.8 % against
  80.0 %; no data race in `test-race-cover`.
* **Stop conditions:** none met. Every check is expressed against the fake,
  the authorize URL's as described above; no existing test, twin or check
  changed; every plant was caught; `-race` reported no race.
* **Not done:** the live run, which is the owner's, and Phase 5.
* This entry was added after the gate ran, so the record checks were run
  again before staging.

**For the owner: the probe.** It opens a browser sign-in at OpenAI, then a
device sign-in at xAI; each needs a person within five minutes. Run it from
the repository root:

```bash
LLMPROVIDER_LIVE_0076=1 LLMPROVIDER_LIVE_BROWSER_LOGIN=1 LLMPROVIDER_LIVE_DEVICE_LOGIN=1 \
  go test -tags live_gateways -count=1 -timeout 15m -run '^TestLive_0076Identity$' -v ./llmprovider/
```

`-v` prints the authorize URL and the device code as they come, and
`-timeout 15m` covers both halves. A half whose variable is left unset
skips. Then record here, dated, for each half: whether the sign-in completed
under the custom name (its `FINDING sign-in` line); whether the session
listed models (`FINDING use`); and whether it was revoked (`FINDING
revocation`, with no "RevokeOAuthSession when the half ends" error). A
sign-in refused under the name, or a failed `originator` or `referrer`
check, is a deviation (step 2): the field's doc then says which provider
refuses a custom name, and the MADR is amended. The record quotes no URL,
code or token from the run.

### Phase 4, the owner's live run (2026-11-11)

The owner committed Phase 4's preparation as `ee7e7f3`, then ran the probe
on this host, one half at a time, from the repository root (the
coordinator started each run in the background and opened the URL it
printed in the owner's browser; the owner approved each sign-in). No live
variable other than each half's own was set.

* **OpenAI, browser:** `LLMPROVIDER_LIVE_0076=1
  LLMPROVIDER_LIVE_BROWSER_LOGIN=1 go test -tags live_gateways -count=1
  -timeout 15m -run '^TestLive_0076Identity$/^OpenAIBrowser$' -v
  ./llmprovider/`, exit 0. The authorize URL carried
  `originator=llmprovider-probe`. Its lines:
  * "FINDING sign-in: OpenAI's browser sign-in under "llmprovider-probe"
    completed"
  * "FINDING use: openai listed 6 models with the session"
  * "FINDING revocation: revoked; a refresh afterwards is refused, as
    ErrAuthFailure true"
  * "--- PASS: TestLive_0076Identity/OpenAIBrowser (11.44s)"
* **xAI, device:** `LLMPROVIDER_LIVE_0076=1 LLMPROVIDER_LIVE_DEVICE_LOGIN=1
  go test -tags live_gateways -count=1 -timeout 15m -run
  '^TestLive_0076Identity$/^GrokDevice$' -v ./llmprovider/`, exit 0. Its
  lines:
  * "FINDING sign-in: xAI's device sign-in under "llmprovider-probe"
    completed"
  * "FINDING use: grok listed 13 models with the session"
  * "FINDING revocation: revoked; a refresh afterwards is refused, as
    ErrAuthFailure true"
  * "--- PASS: TestLive_0076Identity/GrokDevice (15.97s)"
* No "RevokeOAuthSession when the half ends" error in either run.
* **Result:** neither provider refuses a custom name, and the referrer
  check passed in the xAI half. No deviation; step 2's doc change and MADR
  amendment are not needed. An earlier attempt at the OpenAI half was
  interrupted before any sign-in, when the printed URL wrapped in the
  owner's terminal and could not be copied; nothing was stored or sent.
* **Not done:** Phase 5, the release.

### Phase 5, prepared (2026-11-11)

Started at `9c34004`, on `main`, nine commits ahead of `origin/main`, with a
clean tree; the latest tag `v1.29.0`; `git ls-remote --tags origin v1.30.0`
empty. This entry prepares the release commit as 0072-PLAN's Phase 6 did,
up to and not including the tag. Each command's output and exit status are in
the session's scratch directory, each status captured before any filter.

#### Step 1: the release's section

* `CHANGELOG.md`'s `[Unreleased]` became `## [v1.30.0] - 2026-11-11`, under
  an empty `[Unreleased]`. The compare links read `[Unreleased]:
  …compare/v1.30.0...HEAD` and `[v1.30.0]: …compare/v1.29.0...v1.30.0`. The
  section is the former `[Unreleased]` unchanged: this PLAN's Added, Changed
  and Fixed, and 0077's Security entry.
* **The README.** `README.md:375` reads "The current release is `v1.30.0`.
  The module requires Go 1.27.4." No other file outside the changelog and
  the records names `v1.29.0`; `v1.28.0` appears outside them only in
  `llmprovider/budget_0072_test.go`, the base its test compiles at, as it
  did at 0072-PLAN's Phase 6. Nothing else changed.

#### Step 2: the checks

* `scripts/check_changelog.py`: "changelog-check: 43 release section(s), 0
  problem(s)"; `--section v1.30.0` printed the section, 55 lines: Added,
  Changed, Fixed and Security. Each exit 0.
* `tools/apidiff.py v1.29.0`, exit 0, "Compatible changes:" and nothing
  incompatible: `./llmprovider/auth.ChatGPTPlanOptions.Identity`,
  `./llmprovider/auth.Identity`, `./llmprovider/auth.OAuthFlowOptions.Identity`
  and `./wizard.Options.Identity`, each "added", exactly the R55 table.
  `make api-check`: "against v1.29.0, 0 incompatible change(s) outside
  llmprovider/x/", exit 0.
* **The gate:** `python3 -B tools/gate.py` waited from load 18.4, started at
  load 9.1, exit 0: "gate: 29 of 29 passed", the `vet-*-live` steps and
  `coverage-check` included (41 packages, 0 problems; `wizard` 93.8 %
  against 86.4 %).
* `go.mod` reads `go 1.27.4`, with no `toolchain` line; `go list -m` prints
  the module path.
* `tools/claims.py`: 0076-MADR 48 of 48, 0077-MADR 23 of 23, this PLAN 7 of
  7, 0077-PLAN 13 of 13. `make records-check`: "162 records, 0 problem(s)".
  `tools/recordlint.py --changed`: "0 record(s); no rule rose".
  `tools/denyscan.py --changed`: "0 hit(s) in 2 file(s)".
  `tools/linkcheck.py`: "0 problem(s), 1242 relative link(s) in 174
  file(s)". `make disclosure-check`: "0 hit(s) in 1514 file(s)". Each exit 0.
  This entry was added after the gate, so the record checks were run again
  before staging, with this entry in the tree: `tools/claims.py` on this
  PLAN, 7 of 7; `make records-check`, 0 problems; `recordlint --changed`,
  "no rule rose"; `denyscan --changed`, "0 hit(s) in 3 file(s)";
  `linkcheck`, "0 problem(s), 1243 relative link(s)"; `make
  disclosure-check`, `changelog-check`, `docs-check`, `parity-check` and
  `markdownlint` (12 files, 0 issues). Each exit 0.

#### The guard

The pre-push disclosure guard, run read-only over the commits a push would
send (`origin/main` to `9c34004`, nine commits, before the release commit,
which the push will also carry), returned exit 0 with no output.

#### The Releases

`gh release list` shows six, `v1.29.0` (Latest) to `v1.25.0`. When
`v1.30.0`'s Release appears there are seven, so R53's check (Deviation 3 of
0072-PLAN) finds `v1.25.0`'s past the sixth. Its JSON is saved to the
session's scratch directory first, and it is deleted with `gh release delete
v1.25.0 --yes` on the owner's word, keeping its tag and section. Nothing was
deleted here.

#### Not done

* The commit, which is the owner's.
* The push and the `v1.30.0` tag, which wait for the owner's word.
* CI on `main` and on the tag, and the Release.
* `tools/consumer.py v1.30.0` and `tools/offline.py v1.30.0`, after the tag.
* R53's check and `v1.25.0`'s deletion, after the Release appears.
* [0077-PLAN](0077-PLAN-go-1-27-4-for-go-2026-6859.md)'s Phase 2, the read
  back of `"GoVersion": "1.27.4"` from `@v1.30.0` through the proxy (V8).
* Any issue comments.
