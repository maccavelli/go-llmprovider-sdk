---
status: accepted
date: 2026-11-12
decision-makers: repository owner
consulted: "the owner's request and rulings of 2026-11-12 (below, verbatim); this repository at 5d08770; gobble-cli at c47f4a7, its 0005-MADR-v1-feature-scope.md (second amendment of 2026-10-14) and its 0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md"
informed: gobble-cli, which builds its providers with WithClientInfo and signs in under its own name; every caller of WithClientInfo, auth.Identity, RevokeOAuthSession and the wizard
---

<!-- markdownlint-disable MD013 MD024 -->

# Name the Consuming Application on a Session's Refresh and Revocation and on the ChatGPT Originator

## Context and Problem Statement

The owner, on 2026-11-12: "Where possible gobble auth would send gobble as
the name/referrer not the sdk".

[0076-MADR-auth-error-kinds-and-doc-drift.md](0076-MADR-auth-error-kinds-and-doc-drift.md)
D3 gave a consumer `auth.Identity{Name, Version}` for its sign-ins: OpenAI's
authorize-URL `originator`, Grok's `referrer`, and the User-Agent of every
sign-in request take it (C6 to C8). It stopped there on purpose: "Refresh and
revocation keep this module's identity" (C42), and its Consequences say "the
identity stops at the sign-in: a refresh or a revocation still names this
module" (C43). The code says the same (C1).

Every other auth-related request still names this module:

* **A refresh** sends this module's User-Agent, whatever the sign-in's
  `Identity` (C2).
* **A revocation** sends it too (C3), and so do the discovery requests a
  Grok or plan revocation reads first (C4, C5).
* **The ChatGPT `originator` header** on the Codex backend's generation and
  model-listing requests is a constant, `go-llmprovider-sdk` (C9, C10). The
  openai provider sets it on generation (C11, C12) and on its listing (C13),
  and `catalog.ListRaw` on its own listing of a ChatGPT session (C14).
  `WithClientInfo` (C17) leads those requests' User-Agent (C15) but does not
  reach the header. 0012-MADR §1.4 asked for `originator: <name>`, the
  consuming application's name (C18); the code sends this module's.
* **The wizard's own requests.** `wizard.Options.Identity` (C32) reaches its
  sign-ins only. Its listing check (C31) and the Kilo profile (C30) take
  `ProviderOptions` alone, and its logout revokes under this module's name
  (C28).

Generation and model listing by a provider already name the application
through `WithClientInfo` (C15), and gobble passes it to every provider and
listing it builds (C36, C37).

### The owner's rulings of 2026-11-12

Recorded as decided, in the owner's words:

1. **"New pair, in v1.31.0":** this is a new record, released with
   [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)
   and [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)
   in `v1.31.0`.
2. **"Include, prove it live":** the ChatGPT `originator` header carries the
   application's name when one is set. An owner-run live probe passes before
   the release: one ChatGPT model listing and one short generation under the
   name "gobble". A refusal is a deviation, and the header then falls back to
   this module's name.
3. **"Keep the SDK trailer":** the User-Agent stays
   `<app>/<ver> (<os>; <arch>) go-llmprovider-sdk/<ver>` (C19).

What this record decides is the mechanism: how a refresh and a revocation
learn the name, how the originator learns it, and what the wizard does.

### Evidence index

<!-- claims: self=5d08770 gobble-cli=c47f4a7 -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `Identity`'s doc: the identity stops at the sign-in | `llmprovider/auth/identity.go:15` | `A refresh and a revocation name this module whatever the sign-in's` |
| C2 | The refresh request's User-Agent | `llmprovider/auth/oauth_session.go:794` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C3 | The revocation request's | `llmprovider/auth/oauth_revoke.go:78` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C4 | A Grok revocation's discovery carries no identity | `llmprovider/auth/oauth_revoke.go:130` | `endpoints, err := oauthEndpointsFor(ctx, oauthFlowConfig{provider: llmprovider.ProviderGrok, issuer: issuer, httpClient: client})` |
| C5 | nor a plan revocation's | `llmprovider/auth/oauth_plan.go:325` | `module's own identity, as a revocation reads them.` |
| C6 | The sign-in's `originator` takes the identity | `llmprovider/auth/oauth_loopback.go:469` | `query.Set(oauthParamOriginator, config.identity.clientName())` |
| C7 | and Grok's authorize `referrer` | `llmprovider/auth/oauth_loopback.go:473` | `query.Set("referrer", config.identity.clientName())` |
| C8 | and Grok's device `referrer` | `llmprovider/auth/oauth_device.go:299` | `{config.identity.clientName()},` |
| C9 | The ChatGPT backend's originator is a constant | `llmprovider/internal/chatgpt/chatgpt.go:31` | `OriginatorValue = "go-llmprovider-sdk"` |
| C10 | set on every ChatGPT backend request | `llmprovider/internal/chatgpt/chatgpt.go:78` | `req.Header.Set(HeaderOriginator, OriginatorValue)` |
| C11 | The openai provider's header helper | `llmprovider/providers/openai/chatgpt_session.go:30` | `func setChatGPTHeaders(req *http.Request, src llmprovider.TokenSource) { chatgpt.SetHeaders(req, src) }` |
| C12 | used on generation | `llmprovider/providers/openai/openai.go:544` | `setChatGPTHeaders(r, p.src)` |
| C13 | The provider's ChatGPT listing | `llmprovider/providers/openai/chatgpt_listing.go:40` | `req, err := chatgpt.ModelsRequest(ctx, p.baseURL, p.userAgent, token, p.src)` |
| C14 | `catalog.ListRaw`'s ChatGPT listing | `llmprovider/catalog/raw.go:164` | `req, err := chatgpt.ModelsRequest(ctx, baseURL, cfg.userAgent, token, src)` |
| C15 | The provider's User-Agent is `WithClientInfo`'s | `llmprovider/providers/openai/openai.go:245` | `userAgent: st.UserAgent(),` |
| C16 | The provider can read the application's name | `llmprovider/settings.go:351` | `func (st *Settings) ClientName() string { return st.identity.Name }` |
| C17 | `WithClientInfo` | `llmprovider/identification.go:15` | `func WithClientInfo(name, version string) Option {` |
| C18 | 0012-MADR §1.4 asks for the application's name | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:323` | `originator: <name>` |
| C19 | The User-Agent's form, with this module's trailer | `llmprovider/internal/transport/transport.go:216` | `return fmt.Sprintf("%s/%s (%s; %s) go-llmprovider-sdk/%s", id.Name, id.Version, runtime.GOOS, runtime.GOARCH, sdk)` |
| C20 | A session takes a provider's client through a method | `llmprovider/auth/oauth_session.go:598` | `func (s *OAuthSession) UseHTTPClient(client *http.Client) {` |
| C21 | and its logger | `llmprovider/auth/oauth_session.go:630` | `func (s *OAuthSession) UseLogger(logger *slog.Logger) {` |
| C22 | The openai provider hands its client to the session | `llmprovider/providers/openai/openai.go:287` | `shareHTTPClient(src, p.client, st.HTTPClientGiven())` |
| C23 | and so does the grok provider | `llmprovider/providers/grok/grok.go:221` | `shareHTTPClient(src, p.client, st.HTTPClientGiven())` |
| C24 | and a catalog listing | `llmprovider/catalog/discovery.go:291` | `user.UseDefaultHTTPClient(cfg.HTTPClient)` |
| C25 | The field-copy guard stops on an exported field of a type it cannot set | `llmprovider/auth/session_kind_0074_test.go:403` | `t.Fatalf("field %s has type %s: teach this test to set it", field.Name, field.Type)` |
| C26 | and skips only three exported fields, all others stored | `llmprovider/auth/session_kind_0074_test.go:392` | `field.Name == "Store" \|\| field.Name == "HTTPClient" \|\| field.Name == "Logger"` |
| C27 | The refresh reads a copy of the session's fields | `llmprovider/auth/oauth_session.go:586` | `httpClient: s.HTTPClient,` |
| C28 | The wizard's logout revokes | `wizard/logout.go:47` | `if err := auth.RevokeOAuthSession(ctx, session); err != nil {` |
| C29 | after giving the session its client | `wizard/logout.go:45` | `session.HTTPClient = o.HTTPClient` |
| C30 | The wizard's Kilo profile takes `ProviderOptions` | `wizard/auth.go:369` | `account, err := kiloProfile(ctx, session.Access, opts...)` |
| C31 | and so does its listing check | `wizard/configure.go:813` | `opts = append(opts, o.ProviderOptions...)` |
| C32 | `wizard.Options.Identity` | `wizard/configure.go:143` | `Identity auth.Identity` |
| C33 | The ChatGPT listing's `client_version` is this module's release | `llmprovider/internal/chatgpt/chatgpt.go:125` | `query.Set("client_version", ClientVersion(sdk))` |
| C34 | The architecture still names this module as Grok's `referrer` | `docs/architecture.md:1116` | `header and the Grok OAuth` |
| C35 | The ChatGPT wire goldens pin the default originator | `llmprovider/providers/openai/testdata/wire/chatgpt/tool.json:12` | `"Originator": "go-llmprovider-sdk",` |
| C36 | gobble names itself to every provider it builds | `gobble-cli:llm/provider/provider.go:194` | `llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),` |
| C37 | and to the wizard's listing | `gobble-cli:llm/provider/signin.go:181` | `llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),` |
| C38 | gobble revokes a session it builds itself | `gobble-cli:llm/provider/login.go:104` | `if rerr := auth.RevokeOAuthSession(ctx, l.session(id, nil, o.HTTPClient, nil)); rerr != nil {` |
| C39 | gobble keeps a login for Grok only | `gobble-cli:llm/provider/login.go:67` | `func HasLogin(id string) bool { return id == string(llmprovider.ProviderGrok) }` |
| C40 | xAI refuses a client id it has not registered | `gobble-cli:docs/decisions/0005-MADR-v1-feature-scope.md:1096` | `refused as unregistered ("Unknown or disabled client")` |
| C41 | gobble's ACP sign-in plans its identity | `gobble-cli:docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md:350` | `Identity: auth.Identity{Name: "gobble", Version}` |
| C42 | 0076-MADR D3 | `docs/decisions/0076-MADR-auth-error-kinds-and-doc-drift.md:250` | `Refresh and revocation keep this module's identity.` |
| C43 | 0076-MADR's Consequences | `docs/decisions/0076-MADR-auth-error-kinds-and-doc-drift.md:321` | `Neutral, because the identity stops at the sign-in: a refresh or a` |
| C44 | `signin.Options` carries an `Identity` | `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md:306` | `Identity   auth.Identity // 0076-MADR D3` |
| C45 | The README says a refresh and a revocation name this module | `README.md:315` | `sign-in request. A refresh and a revocation still name this module.` |
| C46 | The browser, device and plan sign-ins build their session here | `llmprovider/auth/oauth_loopback.go:700` | `return &OAuthSession{` |
| C47 | and Kilo's device sign-in here | `llmprovider/auth/kilo_device.go:122` | `return &OAuthSession{Provider: llmprovider.ProviderKilo, Access: approved.Token, Issuer: config.issuer,` |
| C48 | 0002-MADR §5's originator row | `docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md:233` | `header and OpenAI authorize-URL` |
| C49 | An existing test pins the default originator | `llmprovider/providers/openai/chatgpt_test.go:116` | `if originator != originatorValue {` |

## Decision Drivers

* **The owner's request and rulings of 2026-11-12** (above): the
  application's name wherever a vendor allows it; the originator included,
  proven live; the User-Agent's trailer kept; released in `v1.31.0`.
* **0012-MADR §1.4:** every request names this module and the consuming
  application, and never a reference client (C18). A refresh and a
  revocation are requests.
* **No caller sees a change it did not ask for.** With no `WithClientInfo`
  and no `Identity`, every request is byte for byte what it is today: the
  wire goldens (C35) and the existing tests (C49) pin it.
* **The v1 promise (R48) and R55.** An exported name is added only with the
  call site that needs it, and stays.
* **A session loaded from a store, in a run that never signed in,** is the
  common case: gobble builds its providers from its stored login (C36). The
  name must reach that session's refresh without a sign-in.
* **A refresh shared across processes through the store** (0016-MADR A1,
  A2): the process that sends a request is the one it should name.
* **No existing test changes,** among them 0074-PLAN's field-copy guard
  (C25, C26) and its 66 refresh tests.

## Considered Options

How a refresh and a revocation learn the name:

* **S1. An unexported field on the session, set by every sign-in, and
  `UseIdentity`, which the built-in providers, `catalog` and the wizard call
  with the name they hold; never stored.**
* **S2. The same field and `UseIdentity`, with no hand-off:** only a
  sign-in, or a caller's own `UseIdentity` call, names the session.
* **S3. An exported field, `OAuthSession.Identity`, not stored.**
* **S4. An exported field, stored by `FileTokenStore`.**
* **S5. A revocation option:** a new `RevokeOAuthSessionAs(ctx, session,
  identity)`, since `RevokeOAuthSession`'s signature cannot change (R48).

How the ChatGPT originator learns the name:

* **O1. `WithClientInfo`'s name, as the provider and `catalog` hold it, else
  this module's.**
* **O2. The session's identity (S1's field).**
* **O3. A new openai option for the originator alone.**
* **O4. Keep the constant.**

What the wizard does with `Options.Identity` beyond its sign-ins:

* **W1. It names the wizard's own requests, unless `ProviderOptions` name
  another application, and the session its logout revokes.**
* **W2. The logout only;** the wizard's own requests keep taking
  `ProviderOptions` alone.
* **W3. Nothing:** a caller passes `WithClientInfo` in `ProviderOptions` and
  revokes a session itself.

## Decision Outcome

Chosen options: "S1", "O1" and "W1", because together they carry the name a
consumer already gives (`WithClientInfo`, `Identity`) to every request this
module sends for it, with no new option to learn, no stored field, and no
change for a consumer that names nothing; and O1 is what 0012-MADR §1.4
asked for in the first place (C18). The three rulings above are the owner's
and are decided. The mechanism is this draft's recommendation: the record is
`proposed`, and awaits the owner's review. *(2026-11-12, the owner's
ruling: the three choices this draft put to the owner are decided as
recommended. The mechanism is "Hidden field + UseIdentity", S1, with its
two exported names, `UseIdentity` and `Settings.ClientVersion` (D2, D4);
the wizard is "All wizard requests", W1: the listing check, the Kilo
profile, the Ollama check and the logout, the name put first so that a
caller's own `WithClientInfo` still wins (D5); and the probe's revocation
is `"gobble" in both`, a revocation under the name in each half of the
PLAN's Phase 5. O1 follows from ruling 2. The owner accepted this record on
2026-11-12. Its PLAN is not yet approved.)*

### D1. The ChatGPT originator is the application's name

* Every ChatGPT backend request this module builds, the openai provider's
  generation (C12) and listing (C13) and `catalog.ListRaw`'s listing (C14),
  carries `originator: <name>`, where `<name>` is `WithClientInfo`'s name as
  the provider's or the listing's settings hold it (C16). With no
  `WithClientInfo`, or an empty name, it is `go-llmprovider-sdk`, as today.
* `internal/chatgpt` takes the name from its callers; its constant (C9)
  stays as the default. The functions existing tests call keep their
  signatures (the PLAN).
* A plan session's requests send no `originator`, as today (0043-MADR D5).
* The ChatGPT listing's `client_version` stays this module's release (C33):
  it is a version the backend gates models on, not a name, and an
  application's own version would hide models whose
  `minimal_client_version` is higher (`internal/chatgpt`'s doc,
  `ModelsClientVersion`).
* **Ruling 2's condition.** D1 ships only after the PLAN's owner-run probe
  lists models and generates once under the name "gobble". A refusal is a
  deviation: the originator then stays `go-llmprovider-sdk` whatever
  `WithClientInfo` says, and the record is amended.

### D2. A session carries the application's identity in memory

* **`OAuthSession` gains an unexported `identity Identity`.** Every sign-in
  that returns a session sets it from its `Identity`: the browser, device
  and plan sign-ins where they build the session (C46), and Kilo's (C47).
* **`(*OAuthSession).UseIdentity(id Identity)`** gives a session an identity
  when it has none, under the session's lock, as `UseHTTPClient` and
  `UseLogger` give it a client and a logger (C20, C21). A zero `id` changes
  nothing, and a session that has one keeps it: the first name wins. It
  returns nothing.
* **Never stored.** `FileTokenStore`'s record is unchanged, and a caller's
  own `TokenStore` stores nothing new. A session loaded from a store has no
  identity until a provider, a listing or its caller gives it one (D4, D5).
  A refresh done by another process names that process, which is the one
  sending it.
* **Not shown.** `String`, `GoString`, `MarshalJSON` and `LogValue` are
  unchanged: their output is pinned by existing tests.
* **Kept.** A refresh's new session and a sibling's rotation adopted from
  the store do not touch it: the session that refreshes keeps the identity
  it had.
* `Identity` is comparable, so `OAuthSession` stays comparable (0043-MADR
  A1), and the field, being unexported, is outside the field-copy guard
  (C26).

### D3. A refresh and a revocation send it

* The refresh request (C2), the revocation request (C3), and the discovery a
  Grok or plan revocation reads first (C4, C5) carry the User-Agent of the
  session's identity, read from the copy of the session's fields the refresh
  and the revocation already take (C27). With no identity, each is this
  module's, byte for byte.
* No body, URL or header other than the User-Agent changes. A refresh sends
  no `originator` or `referrer`, and does not start to.

### D4. A provider and a listing hand `WithClientInfo` to the session

* **`(*llmprovider.Settings).ClientVersion() string`:** the version
  `WithClientInfo` gave, or the version the User-Agent shows when it gave
  none, as `ClientName` (C16) is the name. Without it a hand-off could pass
  the name but not the version, and a refresh would name another version
  than the provider's requests.
* The openai and grok providers, beside their client hand-off (C22, C23),
  and `catalog`'s listing, beside its own (C24), call `UseIdentity` with
  `ClientName()` and `ClientVersion()` on a session that takes it, when
  `WithClientInfo` named an application other than this module. A session
  already named keeps its name (D2).
* gobble then names its refreshes with no change of its own: it builds every
  provider and listing with `WithClientInfo` (C36, C37).

### D5. The wizard names its own requests and its logout

* With `Options.Identity.Name` set, the wizard puts
  `llmprovider.WithClientInfo(Name, Version)` before its own options and
  before `ProviderOptions` on the requests it makes itself: the listing
  check (C31), the Kilo profile (C30) and the Ollama check. A
  `WithClientInfo` in `ProviderOptions` comes later and wins.
* Its logout calls `UseIdentity(o.Identity)` on the stored session before it
  revokes it, beside the line that gives it the client (C28, C29).
* No exported name is added.

### D6. What stays, and what is not possible

* **The User-Agent keeps this module's trailer** (ruling 3; C19).
* **A zero `Identity` and no `WithClientInfo` change nothing** (D1 to D5).
* **Grok's OAuth client id stays the Grok CLI's.** xAI refuses a client id it
  has not registered, "Unknown or disabled client", and `auth.x.ai` offers
  no registration (C40). Not possible.
* **OpenAI's browser and device sign-ins keep Codex's client id.** The
  application-registered alternative is the plan sign-in, under
  `AgentName` (0043-MADR D7), which gobble uses. Out of scope.
* **The ChatGPT `client_version`** stays this module's release (D1).
* **The issuer's keys** are fetched under the sign-in that first misses the
  process-wide cache (0076-MADR C38): a later sign-in under another name
  does not fetch them again. Unchanged.
* **A session another process refreshes** is refreshed under that process's
  name, or this module's (D2).
* **gobble sends no ChatGPT backend request today.** It keeps a login for
  Grok only (C39) and offers OpenAI's plan sign-in, whose requests carry no
  `originator`. D1 reaches gobble only through a Codex session it does not
  offer at `c47f4a7`; it reaches every other consumer that names itself.

### D7. Released in `v1.31.0`

Ruling 1: no release of its own. It ships in
[0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md)
Phase 8's `v1.31.0`, with 0075, after its owner-run probe (ruling 2).

### Consequences

* Good, because a consumer that names itself, through `WithClientInfo` or
  `Identity`, is named on every request this module sends for it that a
  vendor lets it name: sign-in, refresh, revocation, generation, listing,
  and the ChatGPT `originator`.
* Good, because gobble's refreshes name gobble with no gobble change (D4),
  and its ACP sign-ins (C41, C44) carry the name into the session they
  return (D2).
* Good, because the code meets 0012-MADR §1.4's `originator: <name>` (C18).
* Good, because nothing is stored: a shared store's refresh names the
  process that sends it, and no record format or consumer store changes.
* Neutral, because the first name wins on a session that several providers
  share, as the first caller's client does (C20).
* Neutral, because this module does not police names: a consumer that names
  itself after a reference client sends that name, as `WithClientInfo`
  already allows on the User-Agent.
* Bad, because two exported methods are added under the v1 promise:
  `OAuthSession.UseIdentity` and `Settings.ClientVersion`.
* Bad, because a consumer that revokes or refreshes a session it builds
  itself, with no provider, must call `UseIdentity`: gobble's `Logout` does
  (C38), and its record must plan the call.
* Bad, because whether the Codex backend accepts any `originator` is known
  only after the probe; until then D1 is conditional (ruling 2).

### Confirmation

* The PLAN's tests pass, each seen red at `5d08770` where it calls existing
  API only, or to fail on its planted break.
* The wire goldens (C35), the existing originator test (C49), the field-copy
  guard (C25) and 0074-PLAN's 66 refresh tests pass unchanged.
* `python3 -B tools/apidiff.py` against the previous tag lists this PLAN's
  two methods among 0074's and 0075's additions, and `make api-check` shows
  no incompatible change.
* The owner-run probe of ruling 2 passes, or its refusal is recorded as a
  deviation with D1's fallback.
* `tools/claims.py` passes on this record.

## Pros and Cons of the Options

### S1. Unexported field, `UseIdentity`, hand-offs

* Good, because a session loaded from a store, in a run with no sign-in, is
  named by the provider or listing that uses it (D4).
* Good, because it follows the session's existing hand-offs of a client and
  a logger (C20 to C24): one rule, first caller wins.
* Good, because nothing is stored, so no record, store or format changes,
  and a refresh shared through the store names the process that sends it.
* Good, because the field-copy guard and every refresh test stay as they are
  (C25, C26).
* Bad, because it adds two exported methods, one of them on `Settings` for
  the version alone.
* Bad, because a session shared by providers with different names takes the
  first.

### S2. Unexported field and `UseIdentity`, no hand-off

* Good, because it adds one exported method.
* Bad, because a session loaded from a store refreshes under this module's
  name unless its caller calls `UseIdentity`. gobble builds its providers
  from a stored login (C36), so every gobble refresh would need a gobble
  change for what `WithClientInfo` already says.

### S3. An exported field, not stored

* Good, because a caller sets it in the literal that builds a session.
* Bad, because the field-copy guard stops on it (C25): an exported field of
  a struct type, outside its three exclusions (C26), is "teach this test to
  set it", an existing test changed.
* Bad, because a provider cannot set a field on a session other goroutines
  are using without a race; it would need a method anyway.
* Bad, because an exported field that `FileTokenStore` drops reads as stored
  data that is lost.

### S4. An exported field, stored

* Good, because a session loaded from a store keeps its sign-in's name.
* Bad, because a refresh by another process, or another application on the
  same store, sends the name that signed in, not its own.
* Bad, because the record format changes, a caller's own `TokenStore` must
  store it too, and the field-copy guard needs a change (C25).

### S5. A revocation option

* Good, because it is explicit at the one call that needs it.
* Bad, because it covers revocation only: the refresh still needs S1's
  field, so two ways to say one thing.
* Bad, because the wizard's logout and gobble's still have to know the name
  at the call.

### O1. `WithClientInfo`'s name

* Good, because the originator and the User-Agent then name the same
  application, from the option that already names it (C15, C16).
* Good, because it is what 0012-MADR §1.4 decided (C18), and no new option
  is needed.
* Good, because a Codex CLI login read through (`VendorCLISession`), which
  has no identity, is named too.
* Bad, because the vendor's acceptance is unverified until the probe.

### O2. The session's identity

* Good, because a session signed in under a name keeps it on every request.
* Bad, because a Codex CLI login has no identity, and a session loaded from
  a store has one only after a hand-off from `WithClientInfo`, which O1
  reads directly.
* Bad, because the originator and the User-Agent could then name different
  applications.

### O3. An openai option

* Good, because a caller could set the originator alone.
* Bad, because R55 asks for a call site, and no consumer has one apart from
  `WithClientInfo`'s name.

### O4. The constant

* Good, because nothing changes.
* Bad, because the owner ruled the other way (ruling 2), and the code keeps
  contradicting 0012-MADR §1.4.

### W1. The wizard names its own requests and its logout

* Good, because a consumer that sets `Identity` and nothing else is named on
  the listing check, the Kilo profile, the Ollama check and the logout.
* Good, because a `WithClientInfo` in `ProviderOptions` still wins, so a
  caller that passes both, as gobble does (C37), sees no change.
* Neutral, because it adds no exported name.
* Bad, because `Identity`'s meaning widens from "the sign-ins" to "the
  wizard's requests", which its doc and the README must say.

### W2. The logout only

* Good, because it is the smallest change that names a revocation.
* Bad, because a consumer that sets `Identity` alone is named on its
  sign-ins and its logout, but not on the listing between them.

### W3. Nothing

* Good, because the wizard does not change.
* Bad, because the wizard's own logout keeps revoking under this module's
  name whatever `Identity` says (C28).

## More Information

### Records this decision amends

* **`0076-MADR-auth-error-kinds-and-doc-drift.md`,** D3's bullet "Refresh
  and revocation keep this module's identity" (C42) and its Consequences'
  "the identity stops at the sign-in" (C43): a dated annotation on each,
  pointing here, written with this draft. Neither is rewritten. The
  annotations say that this record is proposed; when it is accepted and
  built, the PLAN's docs phase changes the code's doc (C1) and the README
  (C45). *(2026-11-12: accepted that day by the owner's ruling, above; the
  annotations, dated at the draft, stand as written, and their "until it is
  accepted and built" now waits on the build alone.)*
* **`0074-PLAN-one-sign-in-handle-and-method-facts.md`,** Phase 8: a dated
  annotation that `v1.31.0` carries this record (ruling 1).
* **`0012-MADR-conform-providers-to-reference-clients.md` §1.4:** no
  amendment. This record makes the code meet its `originator: <name>`
  (C18).
* **`0002-MADR-migrate-llmprovider-from-mcplib.md` §5** (C48): no amendment.
  Its value, `go-llmprovider-sdk`, stays the originator's default, as its
  "Default client name" row is the User-Agent's.
* **`docs/architecture.md`'s "Identity" section** (C34) says the ChatGPT
  `originator` and the Grok `referrer` are `go-llmprovider-sdk`. The
  `referrer` half has been wrong since 0076-MADR D3 (`v1.30.0`); the PLAN's
  docs phase corrects both.

### How this relates to 0074's sign-in handle

`signin.Options` carries an `Identity` (C44), which `x/signin`'s `Start`
passes to `OAuthFlowOptions.Identity` and `ChatGPTPlanOptions.Identity`.
Under D2 the session those sign-ins return carries it, so gobble's ACP
sign-in (C41) is named on that session's refreshes in the same process with
no change to 0074's Phase 4. What this record touches:

* **0074-PLAN Phase 3** reworks the browser and plan sign-ins beneath the
  handle (`runBrowserLogin`) and wraps six sentinels, in
  `oauth_loopback.go`, `oauth_plan.go`, `oauth_device.go`, `kilo_device.go`,
  `oauth_session.go` and `oauth_revoke.go`: the files D2 and D3 change. This
  record's `auth` phase lands after 0074 Phase 3 is committed.
* **0074-PLAN Phase 5** moves the wizard's sign-ins onto the step engine in
  `wizard/auth.go`. D5 changes the Kilo profile's options there and the
  listing in `wizard/configure.go`. This record's wizard phase lands after
  0074 Phase 5.
* **0075-PLAN** edits `oauth_session.go`'s carry-over (its Phase 1) and the
  openai provider (its Phase 3). Each phase of the three records lands on
  the commit before it, one at a time, as 0074-PLAN's "Rules for every
  phase" require.
* D1 touches no file of 0074 and is independent of it.

The PLAN gives the order.

### Self-naming beyond the coordinator's survey

The survey named the refresh, the revocation and the originator constant.
Also found, at `5d08770`:

* the discovery a Grok or plan revocation reads first (C4, C5);
* `catalog.ListRaw`'s ChatGPT listing, a second caller of the originator
  constant, in another package (C14);
* a refresh started by a `catalog` listing, which hands the session a client
  and no name (C24);
* the wizard's listing check, Kilo profile and Ollama check, which take
  `ProviderOptions` and not `Options.Identity` (C30, C31), and its logout
  (C28);
* `docs/architecture.md`, which still names this module as Grok's
  `referrer` (C34).

Checked and found already named, or with nothing to name: Kilo's provider
requests (its editor name is `ClientName`); `auth.KiloProfile`, which takes
the caller's options; Kilo's sessions, which neither refresh nor revoke; the
plan sign-in's refresh, which is D3's refresh request; the issuer's keys,
fetched only by a sign-in (0076-MADR D3).

### Not verified

* Whether the Codex backend accepts an `originator` other than this
  module's, for a listing and a generation (ruling 2; the PLAN's probe).
* Whether OpenAI's and xAI's token and revocation endpoints accept any
  User-Agent. Both accepted "llmprovider-probe" on a sign-in's requests
  (0076-PLAN Phase 4's live run); the PLAN's probe refreshes and revokes
  under "gobble".

### Related

* PLAN: [0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md](0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md).
* The sign-in's identity:
  [0076-MADR-auth-error-kinds-and-doc-drift.md](0076-MADR-auth-error-kinds-and-doc-drift.md) D3, D4.
* The sign-in handle and the release:
  [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md),
  [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md).
* gobble-cli's consumer:
  `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`.
