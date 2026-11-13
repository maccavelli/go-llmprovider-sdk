---
status: in-progress
date: 2026-11-12
associated-madr: "0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 MD024 -->

# Implement the Consuming Application's Name on a Session's Refresh and Revocation and on the ChatGPT Originator

Associated MADR: [0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md](0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md)

## Goal

After this PLAN, in `v1.31.0` with 0074 and 0075 (the owner's ruling 1):

* the ChatGPT backend's `originator` is `WithClientInfo`'s name, else
  `go-llmprovider-sdk` (MADR D1), proven live under "gobble" first (ruling
  2);
* a session carries its sign-in's identity in memory, and a provider, a
  listing or a caller can give one to a session loaded from a store (D2,
  D4);
* a refresh and a revocation, and a revocation's discovery, name that
  application (D3);
* the wizard names its own requests and its logout with `Options.Identity`
  (D5);
* the User-Agent keeps this module's trailer (ruling 3), and with no name
  every request is what it is today (D6).

The MADR is `proposed`. Nothing below starts before the owner accepts it and
approves this PLAN. *(2026-11-12, the owner's ruling: the MADR is accepted,
with the three choices it put to the owner decided as it recommended: S1
with `UseIdentity` and `Settings.ClientVersion`, W1 for every wizard
request, and a revocation under "gobble" in both halves of Phase 5. The
Exported names table and Phases 2, 3 and 5 below already follow them. This
PLAN stays `proposed`: nothing starts before the owner approves it.)*
*(2026-11-12, later: the owner approved this PLAN, "0079 approved"; it is
`in-progress`. See the Execution record.)*

## Prerequisites

* **For Phase 2:** [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)
  Phase 3 is committed. It reworks the sign-ins and the sentinels in the
  `auth` files Phase 2 changes (MADR, "How this relates to 0074's sign-in
  handle").
* **For Phase 3:** 0074-PLAN Phase 5 is committed. It moves the wizard's
  sign-ins onto the step engine in `wizard/auth.go`, whose Kilo-profile
  options Phase 3 changes.
* **For Phase 6:** 0074-PLAN Phase 8 is the release.

Phase 1 needs none of these.

## Scope

### In scope

* **D1:** `llmprovider/internal/chatgpt/chatgpt.go`;
  `llmprovider/providers/openai/openai.go`, `chatgpt_session.go`,
  `chatgpt_listing.go`; `llmprovider/catalog/config.go`, `raw.go`; their
  tests.
* **D2, D3:** `llmprovider/auth/tokenstore.go`, `oauth_session.go`,
  `oauth_revoke.go`, `oauth_plan.go`, `oauth_loopback.go`, `kilo_device.go`,
  `identity.go`; their tests.
* **D4:** `llmprovider/settings.go`; `llmprovider/providers/openai/openai.go`,
  `llmprovider/providers/grok/grok.go`; `llmprovider/catalog/discovery.go`;
  their tests.
* **D5:** `wizard/configure.go`, `wizard/auth.go`, `wizard/logout.go`; their
  tests.
* **Live (ruling 2):** a new `llmprovider/live_0079_name_test.go`;
  `llmprovider/live_checks_test.go`, `live_twins_test.go`.
* **Docs:** `README.md`, `docs/architecture.md`, `CHANGELOG.md`.

### Exported names (R55)

Outside `llmprovider/x/`, under the v1 promise from `v1.31.0`:

| Name | Call site that needs it |
| :--- | :--- |
| `(*auth.OAuthSession).UseIdentity` | the openai and grok providers' and `catalog`'s hand-off of `WithClientInfo` to a session they use (MADR D4); the wizard's logout (D5); gobble's `Logout`, which revokes a session it builds itself (MADR C38), by a gobble-cli record of its own |
| `(*llmprovider.Settings).ClientVersion` | the same three hand-offs, which pass `WithClientInfo`'s version with its name, so a refresh names the version the provider's requests name (MADR D4) |

No other name is exported. `internal/chatgpt`'s new functions are internal.

### Out of scope

* Grok's OAuth client id, and OpenAI's Codex client id (MADR D6).
* The ChatGPT `client_version` (MADR D1).
* gobble's own calls of `UseIdentity`, and its adoption of `v1.31.0`
  (gobble-cli's records).
* Showing the identity in a session's `String`, `GoString`, `MarshalJSON`
  or `LogValue`, or storing it (MADR D2).

## Rules for every phase

* **Approval.** The owner approves this PLAN, whole or by phase. A phase
  starts only on the commit of the phase before it, of this PLAN or of 0074's
  and 0075's, one at a time on `main`.
* **Red, then green.** A new test that calls only existing API is seen red
  at `5d08770` with `python3 -B tools/redgreen.py SPEC.json --expect fail
  --rev 5d08770`, and green on the phase's tree with `--expect pass`. A test
  that calls a new name, and a test whose red run cannot show each
  assertion, is seen to fail on each planted break with `python3 -B
  tools/plant.py SPEC.json`, on its scratch clone, never in the tree. This
  record holds each failure line. A break that does not fail its test is a
  deviation.
* **The refresh tests stay green, unchanged.** Before staging Phases 2 and 3,
  `python3 -B tools/redgreen.py REFRESH.json --expect pass` runs 0074-PLAN's
  66 refresh and store functions (its "Rules for every phase"), and `git
  diff --stat` shows none of their 15 files changed, nor
  `session_kind_0074_test.go` (MADR C25).
* **Heavy runs queue.** Every `-race` run and the gate run through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`. The
  gate is `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3
  -B tools/gate.py --timeout 5400`. Full-module test runs are the gate's.
* **Pre-add:** `make pre-add-check FILES="..."` on every changed Go file.
* **Staging and commits:** as 0074-PLAN's rules; the coordinator commits
  after the owner's approval.
* **Stop conditions:**
  * any existing test, golden or fixture needs a change: among them the
    ChatGPT wire goldens (MADR C35), `chatgpt_test.go`'s originator check
    (C49), the field-copy guard (C25), and the 66 refresh tests;
  * with no `WithClientInfo` and no `Identity`, any request's URL, body,
    `originator` or User-Agent changes;
  * `FileTokenStore`'s record bytes change for any session;
  * an error's text changes;
  * `tools/apidiff.py` lists an addition outside this table, 0074's and
    0075's;
  * a phase's prerequisite (above) is not committed at its base.

## Implementation Steps

### Phase 0: records

1. Commit this PLAN, the MADR, their index rows (`tools/records.py add`,
   done with the draft), and the two annotations written with them: 0076-MADR
   D3 and its Consequences (MADR C42, C43), and 0074-PLAN Phase 8 (P7).
2. On the owner's acceptance, `tools/records.py status` sets the MADR
   `accepted` and this PLAN `in-progress`. *(2026-11-12: the MADR's half is
   done: `tools/records.py status … accepted --date 2026-11-12` set it, with
   its index row. This PLAN's waits for the owner's approval.)* *(2026-11-12,
   later: done for this PLAN too, `in-progress`, after the owner's
   approval.)*
3. `python3 -B tools/gate.py --markdown`, through `run-heavy`, then stage.

Acceptance: `make records-check links-check` pass; the annotations quote
nothing away; no source changed.

### Phase 1: the ChatGPT originator (D1)

1. `internal/chatgpt`: `SetHeadersAs(req, src, name)` and
   `ModelsRequestAs(ctx, baseURL, userAgent, originator, token, src)`, which
   send `name`, or `OriginatorValue` when it is empty. `SetHeaders` and
   `ModelsRequest` keep their signatures and call them with
   `OriginatorValue`: `chatgpt_test.go` calls `ModelsRequest` as it is
   (P4). `PlanModelsRequest` is unchanged and sends no originator.
2. openai provider: an `originator` field from `st.ClientName()` (MADR C16);
   generation's `Prepare` (C12) and the ChatGPT listing (C13) use the `…As`
   forms. `originatorValue` stays: `chatgpt_test.go` reads it (C49).
3. `catalog`: `config.clientName` from `st.ClientName()`;
   `fetchChatGPTRaw` (C14) uses `ModelsRequestAs`.

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestChatGPTOriginator_ClientInfo_0079` | an openai provider with a ChatGPT session against an `httptest` backend, built with `WithClientInfo("probe-app", "1.2.3")`: a generation and `ListModels` each send `originator: probe-app` and a User-Agent beginning `probe-app/1.2.3 (` | red at `5d08770` (the constant); then planted: generation keeps the constant; the listing keeps it |
| `TestChatGPTOriginator_Default_0079` | with no `WithClientInfo`, and with `WithClientInfo("", "1.2.3")`: `originator: go-llmprovider-sdk` on both, and the User-Agent as at `5d08770` | green at `5d08770`, then planted: `SetHeadersAs` sends `name` with no default, so an empty name sends an empty header |
| `TestListRawOriginator_0079` | `catalog.ListRaw` for openai with a ChatGPT session: `WithClientInfo("probe-app", "1.2.3")` sends `originator: probe-app`; none sends the default | red at `5d08770` for the named case; then planted: `fetchChatGPTRaw` keeps `ModelsRequest` |
| `TestPlanSendsNoOriginator_0079` | a plan session's generation and listing, with `WithClientInfo("probe-app", "1.2.3")`, send no `originator` (0043-MADR D5; P9 holds it without the option) | green at `5d08770`, then planted: `PlanModelsRequest` sets the originator |

Acceptance: the tests pass, each seen as stated; the ChatGPT wire goldens
(MADR C35) and every existing test of `internal/chatgpt`, `providers/openai`
and `catalog` pass unchanged; `tools/apidiff.py` lists no addition.

### Phase 2: the session's identity, on refresh and revocation (D2, D3)

0. **The prerequisite at the phase's base:** 0074-PLAN Phase 3 is committed.
   The sites below are read again there before any change, and any that
   moved is recorded with its new line.
1. `tokenstore.go`: `OAuthSession` gains the unexported `identity Identity`,
   documented as MADR D2 (in memory only; never stored or shown).
2. `oauth_session.go`: `UseIdentity(id Identity)`, under the session's lock:
   a zero `id` changes nothing, and a session with an identity keeps it. Its
   doc follows `UseHTTPClient`'s and says it returns no error (R52).
   `oauthSessionState` gains `identity`, set by `refreshState`;
   `newRefreshRequest` sets the User-Agent through `state.identity`
   (`Identity.setUserAgent`, whose zero value is this module's). Neither the
   refreshed session nor `adopt` sets or clears `identity`.
3. `oauth_revoke.go`: the revocation request's User-Agent through
   `state.identity`; Grok's discovery takes `identity: state.identity` in
   its `oauthFlowConfig` (MADR C4); the plan's revocation calls
   `planEndpointsFor` with the identity (C5). `planOAuthEndpoints` keeps its
   signature for its existing callers and tests.
4. The sign-ins set `identity` from their `Identity`: `oauthSessionFromResponse`
   (MADR C46), which the browser, device and plan sign-ins share, and Kilo's
   device sign-in (C47).
5. `identity.go`: `Identity`'s doc says a session keeps the sign-in's
   identity in memory, that its refresh and revocation send it, that
   `UseIdentity` gives one to a session that has none, and that nothing
   stores it (MADR C1, as D2 and D3 change it).

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestSessionIdentity_Unset_0079` | with no identity, and with `Identity{Version: "9.9.9"}` and no name: the refresh of a ChatGPT (JSON), Grok (form) and plan (form) session, and the revocation of each with Grok's and the plan's discovery, send "go-llmprovider-sdk/SDK (GOOS; GOARCH) go-llmprovider-sdk/SDK", and each URL and body equals a literal pinned from `5d08770` | green at `5d08770` (it names no new identifier: the named-version case signs in through `OAuthFlowOptions.Identity`, which exists there), then planted: the version read without a name |
| `TestSessionIdentity_Refresh_0079` | sessions from the OpenAI browser and device, Grok browser and device, and plan sign-ins, each at a fake issuer under `Identity{Name: "probe-app", Version: "1.2.3"}`: each forced refresh's User-Agent begins `probe-app/1.2.3 (` and names `go-llmprovider-sdk/`; its body and URL equal the unnamed session's | planted: `newRefreshRequest` keeps this module's User-Agent; `oauthSessionFromResponse` does not set the identity |
| `TestSessionIdentity_Revoke_0079` | the same sessions revoked: the revocation request, and for Grok and the plan the discovery before it, carry the name; the forms and JSON body are unchanged | planted: Grok's revocation discovery without the identity; the plan's revocation through `planOAuthEndpoints` |
| `TestUseIdentity_0079` | a session with none takes the first non-zero identity; a second does not replace it; a zero one changes nothing; a sign-in's is kept; run under `-race` with `UseIdentity` concurrent with `Token`'s refresh | planted: the last identity wins; a zero identity clears one |
| `TestSessionIdentity_NotStored_0079` | a named session saved by `FileTokenStore` writes the same bytes as the same session unnamed; loaded, its refresh names this module; its `String`, `GoString`, `MarshalJSON` and `LogValue` equal the unnamed session's | planted: `fileRecord` gains the name; `String` shows it |
| `TestSessionIdentity_Kept_0079` | after a refresh, and after a rejected refresh adopts a sibling's rotation from the store (0016-MADR A1), the session's next refresh still names the application | planted: `adopt` copies the loaded session's zero identity |

Acceptance: the tests pass, each seen as stated; the 66 refresh tests and
`TestSessionFieldsCarried_0074` pass with their files unchanged; every
existing test of `llmprovider/auth` passes unchanged; `tools/apidiff.py`
against the previous tag adds `(*auth.OAuthSession).UseIdentity` to 0074's
additions and nothing else of this PLAN's.

### Phase 3: the hand-offs, and the wizard (D4, D5)

0. **The prerequisite at the phase's base:** 0074-PLAN Phase 5 is committed.
1. `llmprovider/settings.go`: `ClientVersion() string`, documented beside
   `ClientName` (MADR D4).
2. openai and grok providers: beside `shareHTTPClient` (MADR C22, C23), a
   `shareIdentity(src, st)` that calls `UseIdentity(auth.Identity{Name:
   st.ClientName(), Version: st.ClientVersion()})` on a source that takes it,
   when `st.ClientName()` is not this module's default name.
3. `catalog`'s `listingToken`: the same, beside its client hand-off (C24),
   before it reads the token.
4. `wizard`: with `Options.Identity.Name` set,
   `llmprovider.WithClientInfo(Name, Version)` goes first in the option
   lists of the listing check (C31), the Kilo profile (C30) and the Ollama
   check, before `ProviderOptions`; the logout calls
   `session.UseIdentity(o.Identity)` before `RevokeOAuthSession` (C28).
   `Options.Identity`'s doc says so.

| Test | Asserts | Seen to fail |
| :--- | :--- | :--- |
| `TestSettingsClientVersion_0079` | `WithClientInfo("a", "1.2.3")` gives "1.2.3"; `WithClientInfo("a", "")` and no option give the version `UserAgent()` shows, the main module's and this module's | planted: `ClientVersion` returns the given version alone, empty for none |
| `TestProviderNamesSession_0079` | a session loaded from a `FileTokenStore`, given to an openai provider (a ChatGPT session) and a grok provider each built with `WithClientInfo("probe-app", "1.2.3")`: a 401 from the backend forces a refresh whose User-Agent begins `probe-app/1.2.3 (`; with no `WithClientInfo`, this module's; a session given `first-app` before keeps it | red at `5d08770` (this module's); then planted: grok's hand-off omitted; the version passed empty |
| `TestListingNamesSession_0079` | `catalog.ListRaw` with `WithClientInfo("probe-app", "1.2.3")` and a session within its refresh window: the refresh it causes names the application | red at `5d08770`; then planted: `listingToken`'s hand-off omitted |
| `TestWizardOwnRequestsNamed_0079` | `Options.Identity{Name: "probe-app", Version: "1.2.3"}` with no `WithClientInfo` in `ProviderOptions`: the listing check, the Kilo profile and the Ollama check lead their User-Agent with `probe-app/1.2.3`; with `WithClientInfo("other-app", "2.0.0")` in `ProviderOptions`, with `other-app/2.0.0`; with no `Identity`, as at `5d08770` | red at `5d08770` for the first case; then planted: the option put after `ProviderOptions`; the Kilo profile without it |
| `TestWizardLogoutNamed_0079` | `wizard.Logout` of a stored Grok session with `Options.Identity` set: the discovery and the revocation carry the name | red at `5d08770`; then planted: the logout without `UseIdentity` |

Acceptance: the tests pass, each seen as stated; every existing test of
`llmprovider`, `providers/openai`, `providers/grok`, `catalog` and `wizard`
passes unchanged, the wizard's texts included (0074-PLAN's stop condition);
the 66 refresh tests pass; `tools/apidiff.py` adds
`(*llmprovider.Settings).ClientVersion`, and this PLAN's two names are all
its own.

### Phase 4: docs and the changelog

1. `README.md`: the wizard's `Options.Identity` item (MADR C45) says the
   identity reaches the session's refresh and revocation and the wizard's
   own requests, unless `ProviderOptions` name another application; the
   client-identification item (`WithClientInfo`) says it names the ChatGPT
   `originator` and the sessions a provider uses.
2. `docs/architecture.md`, "Identity" (MADR C34): the `originator` is
   `WithClientInfo`'s name, the Grok `referrer` the sign-in's `Identity`,
   each `go-llmprovider-sdk` without one; a session's refresh and revocation
   name its identity; the trailer stays.
3. `CHANGELOG.md` `[Unreleased]`: **Added** the two names; **Changed** the
   ChatGPT `originator` follows `WithClientInfo`, a session's refresh and
   revocation name the application, and the wizard's own requests and
   logout name `Options.Identity`.
4. `make docs-check changelog-check links-check markdownlint parity-check`.

Acceptance: each passes; the README and architecture say what Phases 1 to 3
built, and no example passes a name the code does not send.

### Phase 5: owner-run probe (ruling 2)

The owner's ruling: one ChatGPT model listing and one short generation under
the name "gobble" pass before the release. This phase also refreshes and
revokes under the name, at both vendors: the probe's scratch sessions must
be revoked anyway, so revoking under the name costs nothing and probes D3's
last request. *(2026-11-12, the owner's ruling: `"gobble" in both`, a
revocation under the name in each half, as written here.)*

1. **`TestLive_0079Name`** in `llmprovider/live_0079_name_test.go` (build
   tag `live_gateways`), skipped unless `LLMPROVIDER_LIVE_0079=1`, with
   `auth.Identity{Name: "gobble"}` and `WithClientInfo("gobble", "")`, in two
   halves, each gated as AGENTS.md "Live tests" assigns the variables (P1,
   P2):
   * **`ChatGPT`,** also under `LLMPROVIDER_LIVE_BROWSER_LOGIN` and
     `LLMPROVIDER_LIVE_CHATGPT` (live ChatGPT calls spend the subscription,
     P3): `LoginBrowserOAuth` for OpenAI under the identity, its `OpenURL`
     checking the authorize URL's `originator` before it logs the URL
     (`checkAuthorizeIdentity`, P5). Then an openai provider with the
     session, `WithClientInfo("gobble", "")`, no probes: `ListModels`, which
     must hold a model, and one short generation (`checkWord`). Then a
     forced refresh, which must succeed, and the revocation.
   * **`Grok`,** also under `LLMPROVIDER_LIVE_DEVICE_LOGIN`:
     `StartDeviceOAuth` and `Wait` under the identity, the device form's
     `referrer` checked (`checkDeviceReferrer`, P6); `catalog.ListRaw` with
     `WithClientInfo("gobble", "")`, which must hold a model; a forced
     refresh, which must succeed; and the revocation, after which a refresh
     must be refused.
   * Each half's requests go through a `nameRecorder` (step 2), which keeps
     only each request's `originator` value and its User-Agent's first
     product name. Each half logs `FINDING` lines, as 0076's probe did. Its
     only store is a `FileTokenStore` in `t.TempDir()`; a session a check
     leaves unrevoked is revoked when the half ends. Each sign-in has five
     minutes.
   * **A control for a refusal.** When the ChatGPT listing or the generation
     fails under the name, the same call runs once with a provider built
     without `WithClientInfo` (the default `originator`). If it passes, the
     failure is a refusal of the name, logged `FINDING refused:`; if it
     fails too, the failure is not this record's, and the half stops. A
     revocation refused under the name is logged the same way, and a copy of
     the session with no identity is then revoked, so nothing stays alive.
   * *Owner-run; asks first; scratch store; quota: one short generation on
     the owner's ChatGPT plan, under Codex's client; none at xAI.*
2. **The checks,** appended to `llmprovider/live_checks_test.go`
   (untagged); no existing function changes: `nameRecorder`;
   `checkNamedRequests` (every recorded backend request carried the name as
   its `originator`, when one was sent, and led its User-Agent);
   `checkNamedRefresh` (a forced refresh succeeds, and its request led its
   User-Agent with the name); `checkNamedRevocation` (the revocation and its
   discovery led with the name, then a refresh is refused).
3. **The twin,** `TestLiveTwin_Name_0079` in `llmprovider/live_twins_test.go`
   (untagged), against a new `nameFake` serving a Codex backend's `/models`
   and `/responses`, a token endpoint, discovery and a revocation endpoint;
   the existing twins and fakes are unchanged. Each check's planted break is
   seen to fail on it (0046-MADR D3):
   * the originator kept at the constant in `internal/chatgpt`;
   * the refresh's User-Agent kept at this module's;
   * the revocation's kept at this module's;
   * the fake refuses the refresh;
   * the fake lets a refresh after revocation succeed.
   **What the twin does not cover:** OpenAI's authorize URL, which the
   SDK builds only on the registered ports (0076-PLAN Phase 4 says why), and
   whether a vendor accepts the name.
4. `scripts/check_tables.py` passes: `LLMPROVIDER_LIVE_0079` is this
   record's variable, which that script accepts because this record names
   it.
5. **A refusal is a deviation** (ruling 2). For the `originator`: Phase 1's
   providers and `catalog` pass `OriginatorValue` whatever `WithClientInfo`
   says, by a dated amendment of the MADR's D1 and this PLAN, and the field
   docs say why. For a refresh or a revocation refused under the name: a
   dated entry and the owner's ruling before the release.

**For the owner: the probe.** It opens a browser sign-in at OpenAI, then a
device sign-in at xAI; each needs a person within five minutes. From the
repository root:

```bash
LLMPROVIDER_LIVE_0079=1 LLMPROVIDER_LIVE_BROWSER_LOGIN=1 LLMPROVIDER_LIVE_CHATGPT=1 \
  LLMPROVIDER_LIVE_DEVICE_LOGIN=1 \
  go test -tags live_gateways -count=1 -timeout 15m -run '^TestLive_0079Name$' -v ./llmprovider/
```

A half whose variables are not all set skips. The record quotes no URL, code
or token from the run.

Acceptance: the twin's plants are caught; `go vet -tags live_gateways
./...` passes; the owner's run is recorded here, each `FINDING` line quoted,
with no refusal, or the refusal's deviation resolved.

### Phase 6: the release

No tag of its own (ruling 1). It ships in 0074-PLAN Phase 8's `v1.31.0`
(P7, which carries a dated annotation saying so). Before that phase starts:

* Phases 1 to 5 are committed, and Phase 5's run is recorded;
* the `[Unreleased]` section holds Phase 4's entries.

In 0074-PLAN Phase 8, `tools/apidiff.py` against `v1.30.0` lists this
PLAN's two names among 0074's and 0075's. This PLAN is set `complete` when
that phase records the tag's CI and `tools/consumer.py`.

## Verification

Each command's exit status is captured before any filter; heavy ones run
through `run-heavy`.

* `go test -race -count=1 ./llmprovider/auth/... ./llmprovider/catalog/... ./llmprovider/providers/openai/... ./llmprovider/providers/grok/... ./llmprovider/internal/chatgpt/... ./wizard/...`
* `python3 -B tools/redgreen.py REFRESH.json --expect pass` (0074's 66)
* `make api-check` and `python3 -B tools/apidiff.py v1.30.0`
* `make tables-check docs-check changelog-check links-check records-check
  disclosure-check gate-selftest markdownlint`
* `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- python3 -B tools/gate.py --timeout 5400`
* `python3 -B tools/claims.py` on the MADR and this PLAN
* `go vet -tags live_gateways ./...`
* Every red run and plant of Phases 1 to 3 and 5, quoted.

## Rollout and Rollback

* **Order.** Phase 0 now. Phase 1 at any point after approval: it shares no
  file with 0074, and lands between 0075's phases on the commit before it.
  Phase 2 after 0074-PLAN Phase 3. Phase 3 after 0074-PLAN Phase 5. Phase 4
  after Phase 3, serial with 0074-PLAN Phase 6's changelog work. Phase 5
  after Phases 1 to 3, before 0074-PLAN Phase 8. Phase 6 is 0074-PLAN Phase
  8. gobble then adopts `v1.31.0` by its own records: its providers name its
  refreshes with no change (MADR D4), and its `Logout` calls `UseIdentity`.
* **Rollout.** Additive. A caller that names nothing sees nothing change. A
  caller with `WithClientInfo` sees its name on the ChatGPT `originator` and
  on the refreshes of the sessions its providers use; one with `Identity`
  on its sessions' refreshes and revocations and, in the wizard, on the
  wizard's own requests.
* **Rollback before the tag:** revert the phase's commit.
* **Rollback after the tag:** the two names stay (R48). A vendor that
  refuses the name later is answered by sending this module's name on that
  request again, in a patch release, by a dated amendment of this record.

## Code claims

The facts this PLAN's steps rest on beyond the MADR's index, checked by
`tools/claims.py`.

<!-- claims: self=5d08770 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| P1 | AGENTS.md assigns the browser-login variable to the ChatGPT and Grok browser logins | `AGENTS.md:259` | `(the ChatGPT and` |
| P2 | and the device-login variable to the Grok and Kilo device logins | `AGENTS.md:260` | `(the Grok and Kilo` |
| P3 | Live ChatGPT calls are gated because they spend the subscription | `llmprovider/live_chatgpt_session_test.go:27` | `LLMPROVIDER_LIVE_CHATGPT unset: live ChatGPT calls spend the subscription` |
| P4 | An existing test calls `ModelsRequest` with its signature | `llmprovider/internal/chatgpt/chatgpt_test.go:91` | `req, err := ModelsRequest(context.Background(), "http://example.test/codex/", "agent/1", llmprovider.Token{Value: "t"}, src)` |
| P5 | The authorize URL's check the probe reuses | `llmprovider/live_checks_test.go:225` | `func checkAuthorizeIdentity(t *testing.T, raw, param, name string) bool {` |
| P6 | and the device form's | `llmprovider/live_checks_test.go:275` | `func checkDeviceReferrer(t *testing.T, rec *identityRecorder, name string) {` |
| P7 | 0074-PLAN Phase 8, the release this PLAN ships in | `docs/decisions/0074-PLAN-one-sign-in-handle-and-method-facts.md:561` | `With 0075-PLAN's last phase: the changelog section takes the next minor` |
| P8 | xAI accepted a custom name on a device sign-in | `docs/decisions/0076-PLAN-auth-error-kinds-and-doc-drift.md:905` | `--- PASS: TestLive_0076Identity/GrokDevice (15.97s)` |
| P9 | An existing test holds that a plan session sends no originator | `llmprovider/providers/openai/plan_0043_test.go:214` | `sent[0].Header.Get(headerOriginator) != "" \|\| sent[0].Header.Get(headerChatGPTAccount) != "" {` |
| P10 | 0076's probe is gated by its own record's variable | `llmprovider/live_0076_identity_test.go:29` | `if os.Getenv("LLMPROVIDER_LIVE_0076") != "1" {` |

## Execution record

~~None yet: this PLAN is `proposed`.~~

### Approval (2026-11-12)

* The owner accepted the MADR on 2026-11-12, with the three rulings its
  Decision Outcome records, and approved this PLAN the same day ("0079
  approved"). `python3 -B tools/records.py status RECORD STATUS --date
  2026-11-12` set the MADR `accepted` and this PLAN `in-progress`, each with
  its row in `docs/README.md`; each run printed "records: written;
  records-check: 0 problem(s)".
* No phase has run. Phase 0 step 1, the commit of the records, is the
  coordinator's, and step 3's `gate.py --markdown` runs before it is staged.
* The phases follow the order in "Rollout and Rollback": Phase 1 at any
  point, Phase 2 after 0074-PLAN Phase 3 is committed, Phase 3 after
  0074-PLAN Phase 5, Phase 4 after Phase 3, Phase 5 before 0074-PLAN Phase
  8, and Phase 6 in 0074-PLAN Phase 8's `v1.31.0`. Each starts on the commit
  of the phase before it, of this PLAN or of 0074's and 0075's.
