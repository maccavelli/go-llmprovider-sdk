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
  their tests. *(2026-11-12, amended: also `llmprovider/catalog/lookup.go`
  and `config.go`, for the lookups' hand-off; Phase 3 step 3a.)*
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
    (C49), the field-copy guard (C25), and the 66 refresh tests
    *(2026-11-13, annotated: this stopped Phase 2 at
    `identity_0076_test.go:373-389`; the owner's ruling allows that one
    change and no other)*;
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
existing test of `llmprovider/auth` passes unchanged *(2026-11-13, annotated:
but `TestIdentity_UserAgent_0076`'s revocation half, which the owner ruled
rewritten; see the deviation entry)*; `tools/apidiff.py`
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
   before it reads the token. *(2026-11-12, amended (MADR, "Amendment
   2026-11-12: the catalog lookups take D4's hand-off"): the hand-off is one
   unexported `nameSession(src, cfg)` in `llmprovider/catalog`, which
   `listingToken` calls here; `config` gains the version beside
   `clientName`, from `st.ClientVersion()`.)*
   * **Step 3a, the lookups** *(added 2026-11-12 by the amendment)*:
     `llmprovider/catalog/lookup.go`: `CachedModel` calls
     `nameSession(src, cfg)` on the line directly above
     `if token, err := src.Token(ctx); err == nil {` (MADR C50), and
     `listingModel` on the line directly above `token, err := src.Token(ctx)`
     (C51). `CachedModel`'s and `LookupModel`'s docs each gain the sentence
     "Before it asks src for its token, it gives a session WithClientInfo's
     name, as List does (0079-MADR D4).", and `CachedModel`'s "It asks src
     for its token as the source gives it" (C58) stays. Nothing else in
     either lookup changes: neither reads its token through `listingToken`,
     so neither gains its client hand-off or its error text (C56, C59).
     These lines are what gobble-cli's 0051-PLAN P2a step 0 quotes at
     `v1.31.0` (P12); a different spelling is a deviation recorded here, with
     the line gobble should quote, before the tag.
     *(2026-11-12, the owner's rulings: "Same fix": `CachedFacts` calls
     `nameSession(cfg.tokens, cfg)` on the line directly above
     `if token, err := cfg.tokens.Token(ctx); err == nil {` (MADR C57), and
     its doc gains "Before it asks the credential opts give for its token,
     it gives a session WithClientInfo's name, as List does (0079-MADR
     D4)."; nothing else of it changes. "Exact code lines": the two
     `nameSession(src, cfg)` lines and `CachedModel`'s doc sentence are
     gobble's signal, and a renamed helper is a deviation recorded here with
     the replacement line. gobble's signal needs no `CachedFacts` line: at
     gobble-cli `8fa0b73` no gobble code calls it (MADR, the signal's note).
     "Keep whole": these steps stay in this phase.)*
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
| `TestLookupNamesSession_0079` *(added 2026-11-12 by the amendment)* | a session loaded from a `FileTokenStore`, past its expiry, used first, with no provider or listing before it, by `catalog.CachedModel`, and in a second case by `catalog.LookupModel`, each with `WithClientInfo("probe-app", "1.2.3")` and a cold listing cache: the refresh the lookup starts is the first request at the fake issuer, and its User-Agent begins `probe-app/1.2.3 (` and names `go-llmprovider-sdk/`; `CachedModel`'s background listing, waited for, sends the same; with no `WithClientInfo`, each refresh sends this module's User-Agent. *(2026-11-12, the ruling "Same fix": a third case, `catalog.CachedFacts` for grok with the session passed as `WithTokenSource` and `WithClientInfo("probe-app", "1.2.3")`: its refresh's User-Agent begins `probe-app/1.2.3 (`; and with the same session followed by `WithAPIKey("static")`, as a provider passes it, no request reaches the issuer.)* | red at `5d08770` (this module's); then planted: `CachedModel` reads `src.Token` directly (its `nameSession` line removed); `listingModel` reads `src.Token` directly; `CachedModel`'s `nameSession` moved below its token read; *(added 2026-11-12)* `CachedFacts` reads its token directly (its `nameSession` line removed) |
| `TestLookupHandOffOnlyNames_0079` *(added 2026-11-12 by the amendment)* | what L1 leaves alone: a session with no `HTTPClient` read by `CachedModel` or `LookupModel` is still without one afterwards; `LookupModel` with a token source that fails gives an error ending "model metadata: listing grok: acquire token: " and the source's error, not "model listing: acquire token"; each lookup asks the source for its token as many times as at `5d08770` | green at `5d08770`, then planted: `CachedModel` and `listingModel` read their token through `listingToken` (L2), which hands the client and changes the text |
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
   logout name `Options.Identity`. *(2026-11-12, amended: **Changed** also
   says that a refresh `catalog.CachedModel` or `catalog.LookupModel` starts
   names `WithClientInfo`'s application, as a listing's does.)*
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

<!-- claims: self=5d08770 gobble-cli=8fa0b73 -->

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
| P11 | gobble's agent reads its model facts through `CachedModel` with a stored session *(added 2026-11-12)* | `gobble-cli:llm/provider/model.go:157` | `info, _ := catalog.CachedModel(ctx, llmprovider.ProviderID(id), src, model, opts...)` |
| P12 | gobble's P2a step 0 quotes this line at `v1.31.0` *(added 2026-11-12)* | `gobble-cli:docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:210` | `the record quotes the line where` |
| P13 | gobble's 0051-MADR N6 leaves the session's naming to the SDK *(added 2026-11-12)* | `gobble-cli:docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:671` | `hand that name to the session before they read its token` |

*(2026-11-12: P11 to P13 added with the amendment, and `gobble-cli=8fa0b73`
added to the marker for them. P1 to P10 name no other repository.)*

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

### Phase 1 (2026-11-12): the ChatGPT originator (D1)

* **Done.** `internal/chatgpt`: `SetHeadersAs` and `ModelsRequestAs` send
  `name`, or `OriginatorValue` when it is empty; `SetHeaders` and
  `ModelsRequest` keep their signatures and call them with `OriginatorValue`;
  `PlanModelsRequest` is unchanged. The openai provider has an `originator`
  field from `st.ClientName()`, which generation's `Prepare` (through
  `setChatGPTHeaders`, now taking the name) and the ChatGPT listing use;
  `originatorValue` stays. `catalog`'s `config.clientName` is
  `st.ClientName()`, and `fetchChatGPTRaw` uses `ModelsRequestAs`. No exported
  name was added: `tools/apidiff.py v1.30.0` lists only 0074's additions
  (`SessionKind`, `OAuthSession.Kind`, `AuthMethod` fields); `make api-check`
  reports 0 incompatible changes.
* **Tests.** `providers/openai/originator_0079_test.go` holds
  `TestChatGPTOriginator_ClientInfo_0079`, `TestChatGPTOriginator_Default_0079`
  and `TestPlanSendsNoOriginator_0079`; `catalog/originator_0079_test.go` holds
  `TestListRawOriginator_0079`. No existing test or golden changed.
* **Red at `5d08770`** (`tools/redgreen.py --expect fail --rev 5d08770`, the
  two tests that name a client): "2 of 2 fail as expected".
  * `originator_0079_test.go:59: listing originator = "go-llmprovider-sdk"; want probe-app`
  * `originator_0079_test.go:59: generation originator = "go-llmprovider-sdk"; want probe-app`
  * `originator_0079_test.go:32: named: originator = "go-llmprovider-sdk"; want "probe-app"`
* **Green at `5d08770`** for `Default` and `PlanSendsNoOriginator`: "2 of 2
  pass as expected". **Green on the tree** for all four: "4 of 4 pass as
  expected".
* **Plants** (`tools/plant.py`, scratch clone):
  * generation keeps the constant: caught, `originator_0079_test.go:59: generation originator = "go-llmprovider-sdk"; want probe-app`.
  * listing keeps the constant: caught, `originator_0079_test.go:59: listing originator = "go-llmprovider-sdk"; want probe-app`.
  * `fetchChatGPTRaw` keeps `ModelsRequest`: caught, `originator_0079_test.go:32: named: originator = "go-llmprovider-sdk"; want "probe-app"`.
  * `PlanModelsRequest` sets the originator: caught, `originator_0079_test.go:105: GET /v1/models sent originator "go-llmprovider-sdk"; want none`.
  * `SetHeadersAs` sends `name` with no default: see the deviation.
* **Deviation (2026-11-12), resolved inside Phase 1's files.** The table's
  plant "`SetHeadersAs` sends `name` with no default" was first **not caught**
  by `TestChatGPTOriginator_Default_0079` as drafted: both providers pass
  `st.ClientName()`, which is never empty, so the helper's own default is
  unreachable through a provider. (The plant `name` alone also failed to
  build, an unused `cmp` import, which does not count as caught; the plant
  was changed to `cmp.Or(name, "")`.) Resolution: the same test also calls
  `chatgpt.SetHeadersAs(req, nil, "")` and `chatgpt.ModelsRequestAs(..., "", ...)`
  directly (helper `checkEmptyNameDefaults`), asserting the default. Re-run:
  caught, `originator_0079_test.go:71: SetHeadersAs("") originator = ""; want "go-llmprovider-sdk"`
  and the same for `ModelsRequestAs`. No assertion was loosened.
* **Pre-add:** `make pre-add-check` on the 8 changed Go files, rc 0.
* **Gate:** `run-heavy -- python3 -B tools/gate.py --timeout 5400`: "gate: 30
  of 30 passed", exit 0.

### Deviation (2026-11-12): the catalog lookups bypass D4's hand-off; records amended

* **Found.** gobble-cli's records agent, writing its 0051-MADR D9 (N6),
  found that `catalog.CachedModel` and `LookupModel`'s `listingModel` read the
  session's token themselves (MADR C50, C51), not through `listingToken`
  (C53), where Phase 3 step 3 puts `catalog`'s hand-off. A refresh either
  lookup starts would therefore name this module, for every consumer. The
  coordinator confirmed it at `f50466b`. The code is the same at `5d08770`
  and at `b44cf42`, where this entry was written with Phase 1 committed and a
  clean tree. Not a failure of Phase 1, which touches neither lookup.
* **The owner's ruling, 2026-11-12:** "Amend SDK 0079": the SDK fixes it at
  the source; gobble passes `WithClientInfo` on its `CachedModel` call (P11)
  and calls no `UseIdentity` for it.
* **Decided** (MADR, "Amendment 2026-11-12: the catalog lookups take D4's
  hand-off"): L1, one unexported `nameSession(src, cfg)`, called by
  `listingToken` and by both lookups before their token read, which is
  otherwise unchanged. Not L2, routing both through `listingToken`, which
  would also hand a lookup's session the listing's client and change
  `LookupModel`'s error text (C56, C59), both stop conditions.
* **Added to this PLAN:** Phase 3 step 3's note and step 3a; the scope's D4
  files, `lookup.go` and `config.go`; `TestLookupNamesSession_0079` and
  `TestLookupHandOffOnlyNames_0079`, each with its plants; Phase 4 step 3's
  changelog note; code claims P11 to P13 at gobble-cli `8fa0b73`.
* **Placement:** Phase 3, as written. The lookups' hand-off needs
  `UseIdentity` (Phase 2, after 0074-PLAN Phase 3) and `nameSession` needs
  `ClientVersion` (Phase 3 step 1), so it cannot land before Phase 2; and it
  ships in the same `v1.31.0` wherever it lands in that window, which is all
  gobble's P2a step 0 depends on. Whether Phase 3 splits so that its
  `catalog` and provider steps land before 0074-PLAN Phase 5 is put to the
  owner (MADR, "Put to the owner").
* **gobble's signal at `v1.31.0`:** in `llmprovider/catalog/lookup.go`, the
  line `nameSession(src, cfg)` directly above each of the two token reads,
  and `CachedModel`'s doc sentence "Before it asks src for its token, it
  gives a session WithClientInfo's name, as List does (0079-MADR D4)." (step
  3a; P12).
* **Not done:** no code. `CachedFacts` (MADR C57) is left to the owner.

### The owner's rulings on the amendment (2026-11-12)

* **`CachedFacts`: "Same fix".** It takes `nameSession(cfg.tokens, cfg)`
  directly above its token read (step 3a's note), with a doc sentence;
  `TestLookupNamesSession_0079` gains its case and the plant "`CachedFacts`
  reads its token directly". MADR claims C60 to C62 added: a provider passes
  it a static key after its options, and it reads a token only for a
  listing layer.
* **Phase 3: "Keep whole".** No split; step 3a stays in Phase 3, after
  0074-PLAN Phase 5.
* **gobble's check: "Exact code lines".** The two `nameSession(src, cfg)`
  lines and `CachedModel`'s doc sentence are the signal (step 3a); a
  renamed helper is a deviation, recorded here with the replacement line.
* **gobble and `CachedFacts`:** not part of the signal. `git grep -n
  CachedFacts 8fa0b73 -- '*.go'` in gobble-cli exits 1, no call; the SDK
  providers gobble builds call it with a static key (MADR C60), which never
  refreshes.
* The MADR's "Put to the owner" keeps its options as written, each with its
  dated ruling; none is open.

### Deviation (2026-11-13): Phase 2 breaks `TestIdentity_UserAgent_0076`'s revocation half

* **Found.** With Phase 2's source in place, `go test ./llmprovider/auth/`
  fails in one test, and only that one, which passed at `3f87e2b`:
  * `identity_0076_test.go:387: Grok browser: revocation's GET /.well-known/openid-configuration sent User-Agent "probe-app/1.2.3 (darwin; arm64) go-llmprovider-sdk/(devel)"; want this module's only`
  * `identity_0076_test.go:387: Grok browser: revocation's POST /revoke sent User-Agent "probe-app/1.2.3 (darwin; arm64) go-llmprovider-sdk/(devel)"; want this module's only`
  * `identity_0076_test.go:387: ChatGPT plan: revocation's GET /.well-known/openid-configuration sent User-Agent "probe-app/1.2.3 (darwin; arm64) go-llmprovider-sdk/(devel)"; want this module's only`
  * `identity_0076_test.go:387: ChatGPT plan: revocation's POST /revoke sent User-Agent "probe-app/1.2.3 (darwin; arm64) go-llmprovider-sdk/(devel)"; want this module's only`
* **Why.** Lines 373-389 assert 0076-MADR D3's old rule, that a revocation
  names this module whatever the sign-in's `Identity`. This record's D3
  reverses exactly that. The test was not named by the pair.
* **The owner's ruling, 2026-11-13:** "Assert new rule + keep old case".
* **Resolution.** `llmprovider/auth/identity_0076_test.go` joins Phase 2's
  scope, for that revocation half alone: it asserts that a signed-in
  session's revocation and its discovery send the sign-in's name first and
  this module's after it (Grok browser, ChatGPT plan), and a new case holds
  that a session built with no identity revokes with this module's
  User-Agent only. The sign-in half is unchanged. Plants, on a scratch clone:
  the revocation keeps this module's identity (fails the new half); a zero
  identity sends an empty or a named User-Agent (fails the no-identity case).
* **Claims.** The marker stays at `5d08770`. Moved lines are recorded in
  Phase 2's execution entry.

### Phase 2 (2026-11-13): the session's identity, on refresh and revocation (D2, D3)

* **Base.** `3f87e2b`, which holds 0074-PLAN Phase 3. The sites were read
  again there; every asserted text is unchanged. Lines that moved, recorded
  here and not in the evidence index, whose rows stay pinned to `5d08770`
  (the marker is not bumped, by the owner's ruling): C5 `oauth_plan.go:325`
  is now 293; C46 `oauth_loopback.go:700` is now 824; C6 and C7
  (`oauth_loopback.go:469`, `:473`) are now 566 and 570; C8
  (`oauth_device.go:299`) is now 303. C1 to C4, C20, C21, C27 and C47 hold
  their lines. **Open item for the release phase:** Phase 1's own changes
  moved or reworded C9 to C15, C22 and C33, and 0076-MADR C43 and P7 (0074's
  Phase 8 line) moved; `tools/claims.py --rev self=HEAD` lists them failing.
  They are not Phase 2's and are left for the release phase to re-pin.
* **Done.** `tokenstore.go`: `OAuthSession.identity`, in memory only.
  `oauth_session.go`: `UseIdentity` (zero changes nothing, first name wins,
  under the lock, no error); `oauthSessionState.identity`, set by
  `refreshState`; `newRefreshRequest` sends it; `adopt` and the refreshed
  session leave it alone. `oauth_revoke.go`: the revocation request, Grok's
  discovery (`identity: state.identity`) and the plan's (`planEndpointsFor`
  with the identity) send it; `planOAuthEndpoints` keeps its signature.
  `oauth_loopback.go` (`oauthSessionFromResponse`) and `kilo_device.go` set
  it from the sign-in's `Identity`. `identity.go`: the doc follows D2, D3.
* **Tests.** `session_identity_0079_test.go` holds the six tests of the
  table; `identity_0076_test.go`'s revocation half is rewritten by the
  deviation above (see its entry), its sign-in half unchanged. The Unset
  test's literals are pinned for the three hand-built sessions; its
  version-only case compares each sign-in's session to an unnamed one.
* **Plants** (`tools/plant.py`, scratch clone; spec in the session's
  scratchpad): 13 of 13 caught.
  * version read without a name: `session_identity_0079_test.go:258: OpenAI browser refresh: POST /oauth/token | ... sent User-Agent "go-llmprovider-sdk/9.9.9 (darwin; ...`
  * `newRefreshRequest` keeps this module's: `:270: OpenAI browser: POST /oauth/token | ... sent User-Agent "go-llmprovider-sdk/(devel) (darwin; arm64)`, and `Kept`'s `:405: refresh 1 sent User-Agent "go-llmprovider-sdk/(devel) ...; want the probe's first`
  * `oauthSessionFromResponse` sets no identity: the same `:270` lines, and `identity_0076_test.go:389: Grok browser: revocation's POST /revoke sent User-Agent "go-llmprovider-sdk/(devel) ...; want the probe's first, then this module`
  * Grok's discovery without the identity: `:284: Grok browser: GET /.well-known/openid-configuration | | sent User-Agent "go-llmprovider-sdk/(devel) ...; want it to begin "probe-app/1.2.3 ("`
  * the plan's revocation through `planOAuthEndpoints`: `:284: ChatGPT plan: GET /.well-known/openid-configuration | | sent User-Agent "go-llmprovider-sdk/(devel) ...`
  * last identity wins: `:300: step 2: UseIdentity({Name:second-app Version:2}) left {Name:second-app Version:2}; want {Name:first-app Version:1}`
  * zero identity clears: `:300: step 3: UseIdentity({Name: Version:}) left {Name: Version:}; want {Name:first-app Version:1}`
  * the record gains the name: `:361: the named session's record is {` (the bytes differ)
  * `String` shows it: `:392: the named session formats as ["OAuthSession{Provider:grok ...`
  * `adopt` copies a zero identity: `:405: refresh 2 sent User-Agent "go-llmprovider-sdk/(devel) ...; want the probe's first` and `:463: refresh 2 (rt-sibling) sent ...`
  * 0076 half, the revocation keeps this module's: `identity_0076_test.go:389: Grok browser: revocation's POST /revoke sent User-Agent "go-llmprovider-sdk/(devel) ...; want the probe's first, then this module`
  * 0076 no-identity case, empty User-Agent: `identity_0076_test.go:409: Grok browser: revocation's POST /revoke with no identity sent User-Agent ""; want this module's only`
  * 0076 no-identity case, a named User-Agent: `identity_0076_test.go:409: Grok browser: revocation's POST /revoke with no identity sent User-Agent "named-app/1.0"; want this module's only`
* ~~**Unset at `5d08770` not shown.** The table asks `TestSessionIdentity_Unset_0079` green at `5d08770`. `redgreen.py --rev 5d08770` reports it "missing": the tool runs the revision's own files, and the test is new, uncommitted. Its literals are the code's own bytes at the base (Phase 2 changes no unset request, and the 66 refresh tests and the gate pass), and its plant above is the proof it can fail. A deviation for the owner to accept or answer.~~
  *(2026-11-13, later: met, no deviation.)* A scratch clone at `5d08770`
  (outside the tree) holds one file with `TestSessionIdentity_Unset_0079` and
  the helpers it needs, and none of `UseIdentity`, the `identity` field or
  any other new name: it compiles and `go vet` is clean there. `go test
  -count=1 -v -run '^TestSessionIdentity_Unset_0079$' ./llmprovider/auth/`
  exits 0: "--- PASS: TestSessionIdentity_Unset_0079 (0.09s)", then "PASS".
  The table's "green at `5d08770`" holds.
* **Refresh tests unchanged.** `redgreen.py REFRESH.json --expect pass`: 65
  of 66 pass; the one "missing" is `TestOAuthSessionWait_WindowsRemoval_0071`,
  which skips off Windows, as 0074's Phase 2 recorded. `git diff --stat` over
  the 14 refresh files and `session_kind_0074_test.go` is empty.
  `TestUseIdentity_0079` passes under `-race` (one package).
* **Pre-add:** `make pre-add-check` on the 8 changed Go files, rc 0.
  `make api-check`: 0 incompatible changes. `tools/apidiff.py v1.30.0`: this
  phase's only addition is `(*auth.OAuthSession).UseIdentity`; the rest are
  0074's, and `Settings.ClientVersion` is Phase 3's.
* **Gate:** `run-heavy -- python3 -B tools/gate.py --timeout 5400`: "gate: 30 of 30 passed", exit 0, run after the last code change (the records after it are this entry and the two notes above).

### Deviation (2026-11-13): Phase 3 breaks `tables-check`, and `listerFor` is a wizard request

* **Found.** With Phase 3's source in place, `make tables-check` exits 2; it is clean at `a0012db` (47 packages, 0 problems):
  * `docs/architecture.md: llmprovider/providers/grok: imports llmprovider/internal/transport, which its row does not name`
  * `docs/architecture.md: llmprovider/catalog: imports llmprovider/auth, which its row does not name`
  * `docs/guides/api-standards.md: llmprovider/catalog: imports llmprovider/auth, which its row does not allow`
  `catalog`'s `nameSession` takes `auth.Identity` (the type of `UseIdentity`'s parameter), and `grok`'s `shareIdentity` reads `transport.DefaultClientName`. Neither pair named the import tables.
* **The owner's ruling, 2026-11-13:** "Add the rows to Phase 3."
* **Resolution.** `docs/guides/api-standards.md` and `docs/architecture.md` join Phase 3's scope, for those rows alone: `catalog`'s row allows and names `auth`, and `grok`'s row names `internal/transport`. Plants, on a scratch clone, one per row with its addition removed; `tables-check` must catch each. A dated note in the MADR's D4 records the import.
* **Also found: `listerFor`** (`wizard/configure.go`) builds the provider that lists a ChatGPT session and takes `ProviderOptions`, but the PLAN named three wizard sites. **The owner's ruling, 2026-11-13:** "Include `listerFor` in Phase 3", a wizard request under W1's "All wizard requests". Resolution: `listerFor` puts `identityOptions()` first, before `ProviderOptions`; `TestWizardOwnRequestsNamed_0079` gains a case for the ChatGPT session listing's `originator` and User-Agent; its plant leaves the identity option out of `listerFor`. A dated note in the MADR's W1 records it.

### Phase 3 (2026-11-13): the hand-offs, and the wizard (D4, D5)

* **Base.** `a0012db`, which holds 0074-PLAN Phase 5 and this PLAN's Phases 0 to 2. The sites were read again there. Every asserted text is unchanged. Lines that moved, recorded here and not in the evidence index, whose rows stay pinned to `5d08770`: C22 `openai.go:287` is now 291 at `a0012db` (292 with this phase); C30 `wizard/auth.go:369` is now 377; C31 `wizard/configure.go:813` is now 825, and the same text is also at 990, in `listerFor`. Moved by this phase's own lines in `lookup.go` and `discovery.go`: C50 138 to 142; C51 187 to 194; C57 174 to 180; C58 123 to 124; C59 189 to 196; C61 160 to 164; C62 173 to 178; C53 277 to 278; C54 259 to 260; C56 295 to 297; C24 291 to 292; C23 221 to 222. C28 and C29 hold (`wizard/logout.go` 45 and 48 with the hand-off between). The wizard's files changed shape in 0074 Phases 4 and 5; no asserted text did.
* **Done.**
  * Step 1: `Settings.ClientVersion()` returns the resolved identity's version.
  * Step 2: `shareIdentity(src, st)` in the openai and grok providers, beside `shareHTTPClient`, skips this module's default name.
  * Step 3: `catalog`'s `nameSession(src, cfg)`, with `config.clientVersion`; `listingToken` calls it before its token read.
  * Step 3a: `CachedModel`, `listingModel` and `CachedFacts` call it directly above their token reads; the doc sentences are written as separate unwrapped lines in `CachedModel`, `LookupModel` and `CachedFacts`.
  * Step 4: `Options.identityOptions()` goes first in the listing check, the Ollama check, the Kilo profile and (the deviation below) `listerFor`; the logout calls `session.UseIdentity(o.Identity)` before `RevokeOAuthSession`; `Options.Identity`'s doc says so.
* **Deviation entry above (tables, `listerFor`).** Done as ruled: the three rows edited, the MADR's D4 and W1 each gained a dated note.
* **gobble's signal, `grep -n` at this tree** (`llmprovider/catalog/lookup.go`):
  * `141: nameSession(src, cfg)` directly above `142: if token, err := src.Token(ctx); err == nil {` in `CachedModel`;
  * `193: nameSession(src, cfg)` directly above `194: token, err := src.Token(ctx)` in `listingModel`;
  * `125: // Before it asks src for its token, it gives a session WithClientInfo's name, as List does (0079-MADR D4).` in `CachedModel`'s doc (the same sentence at 86, in `LookupModel`'s);
  * `179: nameSession(cfg.tokens, cfg)` directly above `180: if token, err := cfg.tokens.Token(ctx); err == nil {` in `CachedFacts`, its doc sentence at 166.
  No other spelling was forced.
* **Tests.** `settings_client_version_0079_test.go`; `providers/openai/identity_0079_test.go` and `providers/grok/identity_0079_test.go` (`TestProviderNamesSession_0079`); `catalog/lookup_names_session_0079_test.go` (`TestListingNamesSession_0079`, `TestLookupNamesSession_0079`, `TestLookupHandOffOnlyNames_0079`); `wizard/identity_0079_test.go` (`TestWizardOwnRequestsNamed_0079`, with the ChatGPT session listing case, and `TestWizardLogoutNamed_0079`). A 401 forces the provider tests' refresh; the lookup tests use sessions past their expiry. `TestLookupHandOffOnlyNames_0079` warms the listing cache first: a listing fetch hands the session its client through `listingToken`, as before 0079, so the "no client" assertion holds only when the lookup fetches nothing.
* **Red and green at `5d08770`**, on a scratch clone with only the compilable tests (the provider tests without their `kept` and `version` subtests, which use new names): `TestProviderNamesSession_0079` (openai, grok), `TestListingNamesSession_0079`, `TestLookupNamesSession_0079` (`named=true`; `named=false` passes), `TestWizardOwnRequestsNamed_0079` and `TestWizardLogoutNamed_0079` are red; `TestLookupHandOffOnlyNames_0079` is green, which also fixes the token-read count at 2.
* **Plants** (`tools/plant.py`, scratch clone, specs in the session's scratchpad): 20 of 20 caught, each by a test failure and none by a build error. `ClientVersion` given alone: `settings_client_version_0079_test.go:42: ClientVersion() = "", want "(devel)"`. Hand-off omitted, openai and grok: `identity_0079_test.go:81: refresh User-Agent = "go-llmprovider-sdk/(devel) ...", want it to begin "probe-app/1.2.3 ("`; the version passed empty: `...User-Agent = "probe-app/(devel) ..."`. `listingToken` without it: `lookup_names_session_0079_test.go:122`. `CachedModel` and `listingModel` direct, and `CachedModel`'s moved below its read: `:141`, `:153`, `:141`. `CachedFacts` direct: `:162`. Both through `listingToken`: `:226: CachedModel gave the session a client`; `:237: LookupModel error = model metadata: listing grok: acquire token: model listing: acquire token: token source is down`. Wizard: option after `ProviderOptions` `identity_0079_test.go:85`; Kilo `:118`; Ollama `:99`; logout `:165`; `listerFor` without the option `:124` and `:128: originator "go-llmprovider-sdk", want "probe-app"`; after `ProviderOptions` `:124`, `:128: originator "probe-app", want "other-app"`. The three table rows, each with its addition removed: `tables-check` caught each (exit 1).
* **Checks.** `make pre-add-check` on the 14 Go files, rc 0. `make api-check`: 0 incompatible changes. `tools/apidiff.py v1.30.0`: this phase adds `(*llmprovider.Settings).ClientVersion` and nothing else of its own. `tables-check`, `docs-check`, `coverage-check`: rc 0 (43 packages, 0 problems).
* **Gate:** `run-heavy -- python3 -B tools/gate.py --timeout 5400`: "gate: 30 of 30 passed", exit 0, 1068 s, started after the last code change (the records after it are this entry).

### Phase 4 (2026-11-13): docs and the changelog

Base: `main` at `c5792fc` (0074-PLAN Phase 6 committed), tree clean. Phase 4,
"Rules for every phase" and every execution entry above were read first, so
the docs describe what Phases 1 to 3 built: the ChatGPT originator, `UseIdentity`
and `Settings.ClientVersion`, the hand-offs in openai, grok and catalog (the
three lookups included), the wizard's requests (`listerFor` and logout), and
the deviations ruled on 2026-11-12 and 2026-11-13. Each claim was read against
the code: `llmprovider/auth/oauth_session.go` `UseIdentity`,
`llmprovider/settings.go` `ClientVersion`, `llmprovider/internal/chatgpt`
`SetHeadersAs`, `wizard/configure.go` `identityOptions`.

* **Step 1, `README.md`.** The client-identification item says
  `WithClientInfo` leads the User-Agent, is the ChatGPT `originator` (a plan
  session sends none), and is given by the OpenAI and Grok providers and
  `catalog`'s listings and lookups to the session they use. The wizard's
  `Options.Identity` item says the session keeps the identity in memory, so
  its refresh, revocation and discovery name the app; that the wizard's own
  requests (listing check, Ollama check, Kilo profile, a ChatGPT session's
  listing) and `wizard.Logout` name it unless `ProviderOptions` carry another
  `WithClientInfo`; and that a session loaded by the caller takes
  `UseIdentity`, the first name staying. The sentence "A refresh and a
  revocation still name this module" is gone: Phase 2 made it false. No
  example passes a name the code does not send.
* **Step 2, `docs/architecture.md` "Identity".** The `originator` is
  `WithClientInfo`'s name, the Grok `referrer` the sign-in's `Identity`, each
  `go-llmprovider-sdk` without one, a plan session sends none, the trailer
  stays; a session's refresh and revocation name its identity, which no store
  keeps; the hand-offs and the wizard's requests as above.
* **Step 3, `CHANGELOG.md` `[Unreleased]`.** **Added** `UseIdentity` and
  `Settings.ClientVersion`; **Changed** the originator, the refresh and
  revocation (the `CachedModel`, `LookupModel` and `CachedFacts` refreshes
  included, per the amendment of 2026-11-12), and the wizard's requests and
  logout. Nothing is said that Phase 5's probe, not yet run, would settle.
* **Step 4:** `make docs-check`, `changelog-check`, `links-check`,
  `markdownlint`, `parity-check`, and `tables-check`, each exit 0.
* **Claims.** No row is added: README, architecture and the CHANGELOG are not
  claim-checked records, and Phase 4 makes no new code assertion in a record
  (the coordinator's decision, as for 0074 Phase 6). The open item of Phase 2,
  claims moved by Phase 1 and the release phase's re-pin, is not Phase 4's.
* **Checks,** each exit status captured before any filter, after the last
  edit including this entry: `records-check`, `recordlint --changed`, `claims` on the 0079 pair and `denyscan --changed`, each exit 0. No `.go` file changed, so the
  gate run is the Markdown gate, `python3 -B tools/gate.py --markdown`
  through `run-heavy`. No new test or gate, so no planted break.
* **Provenance.** Mainstream model, medium effort. Outcome: Phase 4 done as
  written; no deviation and no escalation to the owner; staged for the
  coordinator to commit.
