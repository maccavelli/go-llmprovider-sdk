---
status: proposed
date: 2026-11-11
decision-makers: repository owner
consulted: "magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md (observed 2026-11-10, committed in that repository as 3e2f784b), its findings F3, F20 and F62 and ledger S7 and S11, and the owner's decisions of 2026-11-10; this repository at 773b84c; re-checked 2026-11-11 against gobble-cli at 17d1605 and its proposed, untracked 0051 and 0052 pairs; the owner's rulings of 2026-11-11 on the wizard's client option and on LockRefresh's two context kinds"
informed: gobble-cli, which runs this SDK's sign-ins under its own name; every caller of RevokeOAuthSession, StartDeviceOAuth and the wizard
---

<!-- markdownlint-disable MD013 MD024 -->

# Return One Unsupported Kind from Revocation, an Auth-Failure Kind from Every Device Expiry, Let a Consumer Name Itself on Its Sign-Ins and in the Wizard, and Correct Four Docs

## Context and Problem Statement

The 0245 report (magic-cli-remote
`docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md`)
lists defects in this SDK that stand on their own, apart from the redesign of
[0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md)
to [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md),
and can ship now, in a minor release (its §8.1, item 4, and step 0 of its
§8.4).

### Error kinds (F3, S11)

* **Revocation's unsupported error is the standard library's.** Three places
  return `errors.ErrUnsupported` (C1, C2, C3): a session of a provider with
  no revocation, and a Grok or plan issuer that publishes no
  `revocation_endpoint`. Every sign-in returns `llmprovider.ErrUnsupported`,
  which also matches the standard library's (C4), but not the reverse. So
  `errors.Is(err, llmprovider.ErrUnsupported)`, the test R25 tells a caller
  to use, is false for a revocation. The report names two of the three
  sites; the plan's (C3) is the third. It is reached only for an issuer
  other than OpenAI's own, whose revocation endpoint is built in and fills
  a discovery document that lacks one (C36); a Grok session reaches C2 at
  xAI's own issuer too, since its built-in endpoints fill no revocation
  endpoint (C40).
* **One OpenAI device expiry is a plain error.** The OpenAI device poll
  checks its 15-minute deadline at the top of its loop and returns "OpenAI
  device-code login timed out after 15 minutes", a plain error (C5). Every
  other expiry matches `ErrAuthFailure` (C6), and `Wait`'s doc promises that
  "the code passing its own expiry" does (C7). The line runs only when the
  clock passes the deadline between the pacer's own check (C8) and the loop's,
  so it is rare, but it breaks the documented kind when it does.

### A consumer cannot name itself, or pick its client, in the wizard (F20, S7)

* **The SDK names itself to the identity providers.** The OpenAI authorize
  URL carries `originator=go-llmprovider-sdk` (C9, C10); Grok's authorize URL
  and device request carry `referrer=go-llmprovider-sdk` (C11, C12); every
  sign-in request's User-Agent is the SDK's default identity (C13 to C16). A
  provider built with `llmprovider.WithClientInfo` names the consuming
  application (C17), and 0012-MADR §1.4 says every request should name "this
  module and the consuming application honestly" (C18). The sign-ins cannot.
* **The wizard passes only an HTTP client** to every browser and device
  sign-in (C19). `OAuthFlowOptions` has `ClientID` and `Issuer` (C20), and
  `GrokFlowFromEnv` fills them from the environment (C21), but no wizard
  caller can reach them.
* **One flat client id would be wrong.** `Issuer` means Kilo's API origin to
  the Kilo device sign-in, which ignores `ClientID` (C22); OpenAI's and Grok's
  issuers and clients differ.

The host id, the third of S7's options, is 0075-MADR D6's.

### Docs that say what the code does not (F62)

* `FileTokenStore.LockRefresh` says a waiter "gives up after 40 s" (C23); the
  wait is 60 s since 0068-MADR D3 (C24).
* `OAuthSession.GoString` prints the prefix `&llmprovider.` (C25), from
  before the type moved to `auth`.
* `RevokeOAuthSession`'s doc names OpenAI's JSON request and Grok's form
  (C26), not the plan session's form, which it sends first (C27).
* 0006-MADR §"7. xAI OAuth client id" says "Refresh and `import_vendor_cli`
  use the `oidc_client_id` stored on the session, not the compile-time
  default" (C28). The refresh does (C29). The vendor import reads only the
  default client's entry (C30), so a Grok CLI signed in under
  `GROK_OAUTH2_CLIENT_ID` cannot be imported.

### Evidence index

<!-- claims: self=773b84c -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | No revocation: the standard library's error | `llmprovider/auth/oauth_revoke.go:62` | `return fmt.Errorf("oauth: revoke: %s sessions have no revocation: %w", state.provider, errors.ErrUnsupported)` |
| C2 | Grok with no endpoint | `llmprovider/auth/oauth_revoke.go:124` | `return nil, fmt.Errorf("oauth: revoke: %s publishes no revocation_endpoint: %w", issuer, errors.ErrUnsupported)` |
| C3 | The plan with no endpoint | `llmprovider/auth/oauth_revoke.go:140` | `return nil, fmt.Errorf("oauth: revoke: %s publishes no revocation_endpoint: %w", issuer, errors.ErrUnsupported)` |
| C4 | `llmprovider.ErrUnsupported` matches the standard library's | `llmprovider/api_error.go:76` | `ErrUnsupported error = &sentinelError{msg: "llmprovider: unsupported", parent: errors.ErrUnsupported}` |
| C5 | A plain expiry | `llmprovider/auth/oauth_device.go:234` | `return nil, errors.New("oauth: OpenAI device-code login timed out after 15 minutes")` |
| C6 | Every other expiry | `llmprovider/auth/oauth_device.go:439` | `var errDeviceCodeExpired = fmt.Errorf("%w: oauth: device code expired", llmprovider.ErrAuthFailure)` |
| C7 | `Wait`'s promise | `llmprovider/auth/oauth_device.go:133` | `its own expiry, gives an error matching llmprovider.ErrAuthFailure; an` |
| C8 | The pacer's check | `llmprovider/auth/oauth_device.go:452` | `if !config.now().Before(deadline) {` |
| C9 | The originator's value | `llmprovider/auth/oauth_loopback.go:35` | `originatorValue      = "go-llmprovider-sdk"` |
| C10 | set on OpenAI's authorize URL | `llmprovider/auth/oauth_loopback.go:460` | `query.Set(oauthParamOriginator, originatorValue)` |
| C11 | Grok's authorize referrer | `llmprovider/auth/oauth_loopback.go:464` | `query.Set("referrer", "go-llmprovider-sdk")` |
| C12 | Grok's device referrer | `llmprovider/auth/oauth_device.go:300` | `"referrer":         {"go-llmprovider-sdk"},` |
| C13 | Discovery's User-Agent | `llmprovider/auth/oauth_loopback.go:321` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C14 | The token exchange's | `llmprovider/auth/oauth_loopback.go:654` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C15 | The device requests' | `llmprovider/auth/oauth_device.go:508` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C16 | Kilo's | `llmprovider/auth/kilo_device.go:136` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C17 | A provider can name the application | `llmprovider/identification.go:15` | `func WithClientInfo(name, version string) Option {` |
| C18 | 0012-MADR §1.4, as the code cites it | `llmprovider/identification.go:8` | `consuming application honestly, and never a reference client.` |
| C19 | The wizard passes only a client | `wizard/auth.go:723` | `return auth.OAuthFlowOptions{HTTPClient: o.HTTPClient}` |
| C20 | The flow options have a client and an issuer | `llmprovider/auth/oauth_loopback.go:47` | `ClientID     string` |
| C21 | `GrokFlowFromEnv` fills them | `llmprovider/auth/oauth_loopback.go:189` | `func GrokFlowFromEnv() OAuthFlowOptions {` |
| C22 | Kilo reads `Issuer` as its origin | `llmprovider/auth/kilo_device.go:34` | `origin; nothing else in opts but HTTPClient and the test clock applies.` |
| C23 | The doc's 40 s | `llmprovider/auth/tokenstore_file.go:240` | `F13, Q1 a and its amendment of 2026-10-06). A waiter gives up after 40 s` |
| C24 | The code's 60 s | `llmprovider/auth/tokenstore_file.go:26` | `const lockWait = 60 * time.Second` |
| C25 | `GoString`'s prefix | `llmprovider/auth/tokenstore.go:112` | `func (s *OAuthSession) GoString() string { return "&llmprovider." + s.String() }` |
| C26 | `RevokeOAuthSession`'s doc | `llmprovider/auth/oauth_revoke.go:26` | `// OpenAI: a JSON request to {issuer}/oauth/revoke (revoke.rs:31-53, 104-114).` |
| C27 | The plan's form goes first | `llmprovider/auth/oauth_revoke.go:55` | `case state.resource != "":` |
| C28 | 0006-MADR §7 | `docs/decisions/0006-MADR-subscription-auth-for-llm-providers.md:704` | `` * Refresh and `import_vendor_cli` use the `oidc_client_id` stored on the `` |
| C29 | The refresh sends the session's client | `llmprovider/auth/oauth_session.go:697` | `"client_id":       state.clientID,` |
| C30 | The import reads the default client's entry | `llmprovider/auth/vendor_session.go:168` | `var grokAuthScope = strings.TrimRight(DefaultGrokOAuthIssuer, "/") + "::" + DefaultGrokOAuthClientID` |
| C31 | Tests that read the standard library's error | `llmprovider/auth/oauth_revoke_test.go:92` | `if !errors.Is(err, errors.ErrUnsupported) \|\| c.path != "" {` |
| C32 | and another | `llmprovider/auth/oauth_issuer_fallback_test.go:61` | `if err := RevokeOAuthSession(context.Background(), kilo); !errors.Is(err, errors.ErrUnsupported) {` |
| C33 | and the plan's | `llmprovider/auth/plan_session_0043_test.go:335` | `if err := RevokeOAuthSession(context.Background(), s); !errors.Is(err, errors.ErrUnsupported) {` |
| C34 | The live test pins the originator | `llmprovider/live_oauth_login_test.go:19` | `const identityValue = "go-llmprovider-sdk"` |
| C35 | `NewIdentity` defaults an empty name to this module | `llmprovider/internal/transport/transport.go:194` | `func NewIdentity(name, version, session string) Identity {` |
| C36 | OpenAI's own issuer fills a missing revocation endpoint | `llmprovider/auth/oauth_plan.go:330` | `discovered.Revocation = planBuiltinEndpoints.Revocation` |
| C37 | The issuer's keys are fetched under this module's User-Agent | `llmprovider/auth/oauth_idtoken.go:207` | `transport.NewIdentity("", "", "").SetUserAgent(req)` |
| C38 | from a process-wide cache first | `llmprovider/auth/oauth_idtoken.go:180` | `entry, cached := jwksCache.entries[jwksURL]` |
| C39 | Revocation discovers Grok's endpoints as a sign-in does | `llmprovider/auth/oauth_revoke.go:119` | `endpoints, err := oauthEndpointsFor(ctx, oauthFlowConfig{provider: llmprovider.ProviderGrok, issuer: issuer, httpClient: client})` |
| C40 | The built-in endpoints fill no revocation endpoint | `llmprovider/auth/oauth_loopback.go:305` | `{&e.Authorization, from.Authorization}, {&e.Token, from.Token}, {&e.Device, from.Device}, {&e.JWKS, from.JWKS},` |
| C41 | v1.26.0 changed an error's kind in a minor | `CHANGELOG.md:136` | `no longer retries HTTP 501 or 505` |
| C42 | `RevokeOAuthSession`'s doc names the standard library's error, for Grok | `llmprovider/auth/oauth_revoke.go:28` | `revocation_endpoint; without one the result wraps errors.ErrUnsupported.` |
| C43 | The ChatGPT browser sign-in passed with this module's `originator` | `docs/decisions/0046-PLAN-v1-15-0-follow-ups.md:765` | `--- PASS: TestLive_ChatGPTBrowserLogin (30.37s)` |
| C44 | and the Grok device sign-in with its `referrer` | `docs/decisions/0046-PLAN-v1-15-0-follow-ups.md:766` | `--- PASS: TestLive_GrokDeviceLogin (25.01s)` |
| C45 | The live test checks the `originator` before anyone signs in | `llmprovider/live_oauth_login_test.go:34` | `if got := u.Query()[param]; len(got) != 1 \|\| got[0] != identityValue {` |
| C46 | `LockRefresh`: a `ctx` seen ended between attempts is also `ErrProviderUnavailable` | `llmprovider/auth/tokenstore_file.go:277` | `return nil, fmt.Errorf("%w: oauth: another process is refreshing the %s session: %w",` |
| C47 | and one that ends during the pause is `ctx`'s error alone | `llmprovider/auth/tokenstore_file.go:289` | `return nil, ctx.Err()` |
| C48 | The README names the helper with no word on where its values go | `README.md:153` | `` itself. Pass `catalog.OptionsFromEnv()`, `auth.GrokFlowFromEnv()` or `` |

## Decision Drivers

* **Landable now** (the report's §8.4 step 0; the owner's DF answer of
  2026-11-10, "now, ahead of the redesign"): each change is compatible under
  R48, needs no contract module and no engine, and ships in the next minor
  release.
* **Error kinds are tested with `errors.Is`** (R25), and a function's doc
  names the kinds it returns (R52). A kind the doc does not promise is a
  defect.
* **No caller sees a change it did not ask for.** Every existing
  `errors.Is(err, errors.ErrUnsupported)` keeps matching (C31 to C33). Every
  default identity value stays (C34). Every message's text that a test pins
  stays, except the one plain expiry, whose kind is the defect.
* **The owner's decision of 2026-11-10 on Q15:** one error vocabulary, the
  SDK's sentinels mapped to it. A sentinel that is the wrong one must be
  special-cased: 0073-MADR D3's `unsupported` row maps from
  `errors.ErrUnsupported` beside `ErrUnsupported`, and a mapper written
  from R25 alone would miss a revocation.
* **A change of behaviour ships in a minor release.** The standards guide
  has no rule for it beyond R48, which `apidiff` checks on signatures, and
  R52's named kinds. This repository has done it before: v1.26.0 made
  `WithRetry`'s 501 and 505 terminal, under **Changed** (C41; 0068-MADR D2).
  This record also adds API, so its release is a minor in any case.
* **A consumer names itself** (0012-MADR §1.4, C18), and gobble signs in
  under its own name (the report's F20).

## Considered Options

For the error kinds:

* **E1. Return the SDK's sentinels:** `llmprovider.ErrUnsupported` from the
  three revocation sites, and the device expiry's `ErrAuthFailure`.
* **E2. Leave the code, and document the standard library's error.**
* **E3. Return both** from revocation, joined, so each `errors.Is` holds.

For the consumer's identity and client in the wizard:

* **W1. Flat `wizard.Options.ClientID` and `Issuer`, and an `Identity`.**
* **W2. A per-provider map of client and issuer, and an `Identity`.**
* **W3. One `wizard.Options.GrokFlow auth.OAuthFlowOptions`** for Grok's
  override alone, and an `Identity`.
* **W4. An `Identity` only,** and no client or issuer option until a
  consumer's record names a call site for one. *(Added 2026-11-11, with the
  owner's ruling.)*

## Decision Outcome

Chosen options: "E1" and "W4", because E1 makes every auth error testable by
the one sentinel R25 names, while each existing test still holds, and W4
lets a consumer sign in under its own name while adding no option that no
consumer calls. *(2026-11-11: this read '"E1" and "W2"', "and W2 lets a
consumer set the right client and issuer for each provider without sending
one provider's to another". The owner ruled that day that W2's map,
`wizard.Options.OAuthClients`, and its type, `wizard.OAuthClient`, are
dropped: R55 asks for a consumer's call site, and there is none. W4 is what
W1 to W3 share, the `Identity`.)* The report's §8.4 places this record's
fixes (its S11 and S7) at step 0, landable now, as a recommendation. The
owner answered the timing on 2026-11-10 (DF, put after this record's first
draft): the standalone defect fixes, this record among them with
magic-cli-remote `docs/decisions/0074-PLAN-remote-provider-auth-from-phone.md`
P23 and `docs/decisions/0179-PLAN-gobble-native-acp-provider.md` P7, land
"now, ahead of the redesign", each on
its own once its record is accepted. That answer covers the whole record,
`auth.Identity` included, although it is new API rather than a defect fix
*(2026-11-11: "and the wizard's client map" removed with the map)*. This
record is `proposed`: it awaits the owner's review of this draft.

On 2026-11-11 the owner ruled on this record's two open points:
`wizard.Options.OAuthClients` and `wizard.OAuthClient` are dropped, and the
README is corrected instead (D4); and `LockRefresh`'s doc names both of its
`ctx` kinds now, their unifying left to a later record (D5). The record
stays `proposed`: the owner has answered questions, not accepted it.

### D1. Revocation returns `llmprovider.ErrUnsupported`

* The three sites (C1 to C3) wrap `llmprovider.ErrUnsupported` in place of
  `errors.ErrUnsupported`. Their own words are unchanged; the wrapped part,
  which each prints through its final `%w`, changes from "unsupported
  operation" to "llmprovider: unsupported", which no test pins (the PLAN
  checks).
* `errors.Is(err, errors.ErrUnsupported)` stays true through C4's parent;
  the three tests that read it (C31 to C33) pass unchanged.
* `ErrUnsupported`'s doc says it "fails before any network call" (C4's
  comment). For Grok and the plan, revocation reads the discovery document
  before it finds no endpoint. `RevokeOAuthSession`'s doc says so: a
  revocation may read the issuer's discovery document before it returns
  `ErrUnsupported`, and sends no token.

### D2. Every device expiry matches `ErrAuthFailure`

* The OpenAI poll's deadline check (C5) returns the pacer's expiry, the same
  error `sleepBeforeDeadline` gives (C6), with the last transient failure
  when there was one, as the pacer reports it.
* `Wait`'s doc (C7) is then true on every path.

### D3. The consumer's identity on the sign-ins

* **`auth.Identity{Name, Version string}`,** a comparable value, the
  application's name and version as `llmprovider.WithClientInfo` takes them
  (C17).
* **`OAuthFlowOptions.Identity` and `ChatGPTPlanOptions.Identity`.** When
  `Name` is set:
  * the OpenAI authorize URL's `originator` is `Name`;
  * Grok's authorize URL's and device request's `referrer` is `Name`;
  * every request of the sign-in (discovery, the token exchange, the device
    requests, Kilo's, the plan's, and the fetch of the issuer's keys that
    verifies the ID token, C37, when they are not already cached, C38) has
    the User-Agent of `transport.NewIdentity(Name, Version, "")`, which
    names the application and this module (C35). Discovery is shared with
    revocation (C39), which keeps this module's identity, so the identity
    reaches discovery through the sign-in's own configuration.

  When `Name` is empty, everything is as today, byte for byte, so the
  existing offline tests and the live test's check (C34) hold.
* **Refresh and revocation keep this module's identity.** A session carries no
  identity, and giving it one is a field on `OAuthSession` this record does
  not add. A later record may.
* **The plan's `agent_name_hint` is unchanged:** it is `AgentName`, the
  app's name OpenAI registers (0043-MADR D7), not the identity.

### D4. The wizard passes the identity, and takes no client or issuer

*(2026-11-11: this decision was "The wizard reaches the client, the issuer
and the identity". The owner's ruling of that day removed its first two
bullets, which are quoted in the second bullet below.)*

* **`wizard.Options.Identity auth.Identity`** goes to every sign-in the
  wizard runs (D3).
* **No client or issuer option** *(the owner's ruling of 2026-11-11)*. As
  first written, this decision added
  `wizard.Options.OAuthClients map[llmprovider.ProviderID]OAuthClient`, with
  a new comparable `wizard.OAuthClient{ClientID, Issuer string}`: the
  browser and device sign-ins of a provider took its entry's `ClientID` and
  `Issuer` into `OAuthFlowOptions` (C19, C20), Kilo's `Issuer` being its
  origin (C22) and the plan ignoring the map, and `GrokFlowFromEnv` reached
  the wizard through it, "The README shows it". The owner dropped both
  names. R55 asks each exported name for the call site that needs it, and no
  consumer has one: gobble-cli calls the wizard through
  `wizard.ResolveCredential` only, and no gobble-cli record names
  `OAuthClients`. The wizard goes on setting no OAuth client id and no
  issuer on a sign-in: of `OAuthFlowOptions` it fills the HTTP client (C19),
  and now the identity.
* **The README is corrected instead.** With no such option, a wizard caller
  cannot pass `auth.GrokFlowFromEnv()`'s values. They take effect on a
  caller's own `OAuthFlowOptions` only, given to `LoginBrowserOAuth` or
  `StartDeviceOAuth` (C21). So the README shows only what the wizard can
  do: its wizard section shows `Options.Identity` and says that the wizard
  sets no OAuth client or issuer; and where it names the helper, with no
  word on where its values go (C48), it says they go on a caller's own
  `OAuthFlowOptions`. The example this decision first planned, a wizard
  caller filling `OAuthClients` from `auth.GrokFlowFromEnv()`, is not
  written.

### D5. Four docs, corrected

* `FileTokenStore.LockRefresh`: "after 60 s" (C23, C24). And, by the owner's
  ruling of 2026-11-11, its doc names the two kinds a `ctx` that ends while
  it waits gives, which it does not name today (R52): an error matching both
  `ErrProviderUnavailable` and `ctx`'s when the end is seen between attempts
  (C46), and `ctx`'s error alone when it ends during the pause (C47). The
  code is unchanged: the doc says what it does, and a test shows each kind,
  as R52 asks (the PLAN's Phase 3). Making the two one kind is a later
  record's ("Found on review").
* `OAuthSession.GoString`: the prefix `&auth.` (C25). No test pins the old
  prefix (the PLAN checks); a new test pins the new one.
* `RevokeOAuthSession`: the plan session's RFC 7009 form to its discovered
  endpoint, with the issued client (C26, C27), and D1's note. Its last
  sentence names `errors.ErrUnsupported`, and only for Grok (C42); it names
  `llmprovider.ErrUnsupported` for all three cases of D1, a session of a
  provider with no revocation included.
* 0006-MADR §7 (C28): an amendment records that the refresh uses the
  session's stored client (C29) and the vendor import reads the default
  client's entry only (C30), so an import of a Grok login under another
  client is not supported. Changing the import is out of scope: no caller has
  asked for it.

### Consequences

* Good, because `errors.Is(err, llmprovider.ErrUnsupported)` holds for every
  auth error of that kind, and every device expiry is `ErrAuthFailure`, which
  0074-MADR D8 can map to one contract kind.
* Good, because a consumer can sign in under its own name. *(2026-11-11:
  "and a wizard caller can use an enterprise or test issuer and
  `GrokFlowFromEnv`" removed with `OAuthClients`.)*
* Good, because the docs say what the code does.
* Neutral, because the identity stops at the sign-in: a refresh or a
  revocation still names this module.
* Bad, because whether OpenAI's authorize endpoint accepts any `originator`,
  and xAI's any `referrer`, is not verified. The default is unchanged, so a
  caller who sets `Identity` takes that risk knowingly; the PLAN's owner-run
  probe measures it.
* Bad, because a revocation error's text changes in its wrapped part
  ("unsupported operation" becomes "llmprovider: unsupported"), which a
  caller matching text would see.
* Bad, because `wizard.Options` gains a field and `auth` a type, under the
  v1 promise. *(2026-11-11: "a map" was `OAuthClients`, now dropped.)*
* Bad, because the second half of F20 stays: a client, an issuer and
  `GrokFlowFromEnv`'s values still cannot reach a sign-in the wizard runs.
  The README now says so. *(Added 2026-11-11.)*

### Confirmation

* The PLAN's tests pass, each seen to fail on its planted break, or red at
  `773b84c` where it calls existing API only.
* The three revocation tests (C31 to C33) and every sign-in test pass
  unchanged.
* `make api-check` lists exactly the PLAN's additions.
* `tools/claims.py` passes on this record.

## Pros and Cons of the Options

### E1. The SDK's sentinels

* Good, because one `errors.Is` covers every unsupported auth error (R25).
* Good, because `llmprovider.ErrUnsupported` matches the standard library's
  too, so nothing that holds today breaks.
* Neutral, because `ErrUnsupported`'s "before any network call" is stretched
  for two of the three sites, and the doc says so.
* Bad, because the wrapped part of each revocation error's text changes
  ("unsupported operation" becomes "llmprovider: unsupported"), which a caller
  matching text, not kind, would see.

### E2. Document the standard library's error

* Good, because no code changes.
* Bad, because R25's test stays false for revocation, and Q15's mapping has
  to special-case it.

### E3. Both, joined

* Good, because every test holds.
* Bad, because `llmprovider.ErrUnsupported` already matches the standard
  library's: joining adds nothing but a longer message.

### W1. Flat fields

* Good, because two strings are simple.
* Bad, because one `ClientID` would be sent to OpenAI, Grok and Kilo alike,
  and Kilo reads `Issuer` as its origin (C22): a Grok issuer set for Grok
  would redirect a Kilo sign-in.

### W2. A per-provider map

* Good, because each provider gets its own, and an absent entry is today's
  behaviour.
* Neutral, because a map is not comparable, but `Options` already holds
  slices and functions.
* Bad, because a map field is under the v1 promise from its release: its key
  and value types cannot change, and `OAuthClient` can only grow.
* Bad, because the identity it carries (D3) is unverified at the vendors:
  whether OpenAI accepts any `originator` and xAI any `referrer` is known only
  after the PLAN's owner-run probe.

### W3. A Grok-only field

* Good, because it is all `GrokFlowFromEnv` needs.
* Bad, because OpenAI's issuer and client, and Kilo's origin, stay
  unreachable, and a second provider later needs another field.

### W4. An `Identity` only

*(Added 2026-11-11, with the owner's ruling.)*

* Good, because every name it exports has a call site (R55): gobble's
  sign-ins set the identity.
* Good, because nothing is put under the v1 promise that no consumer calls.
  W2's map could not change its key or value type once released, and its
  value type could only grow.
* Bad, because a client, an issuer and `GrokFlowFromEnv`'s values stay
  unreachable through the wizard (F20). A caller that needs them calls
  `auth`'s sign-in functions itself.
* Bad, because the identity it carries (D3) is unverified at the vendors, as
  under W2.

## More Information

### Records this decision amends, applied on acceptance

* **`0006-MADR-subscription-auth-for-llm-providers.md`, §"7. xAI OAuth
  client id"** (`:704-706`): "Refresh and `import_vendor_cli` use the
  `oidc_client_id` stored on the session, not the compile-time default, so a
  file minted by `grok login` round-trips." An amendment, by D5: true of the
  refresh; the import reads the default client's entry only.
* **`0068-MADR-five-internal-fixes-from-the-0054-audit.md`, D3:** a note that
  `LockRefresh`'s doc now says 60 s, which D3 changed and its PLAN did not
  carry into the doc.
* **`0048-MADR-adopt-go-sdk-best-practices.md`** (R52, error kinds): no
  amendment; D1 and D2 make two functions meet it.

### Where the report differs from the code

* **F3 names two revocation sites** (`oauth_revoke.go:61-62` and
  `:112-127`). There are three: the plan's (C3) is the third.
* **F3's device time-out "is effectively unreachable"** (the SDK audit). It
  is reachable when the clock passes the deadline between the pacer's check
  (C8) and the loop's (C5); a test with a stepped clock reaches it.

### Found on review, not decided here

* **`LockRefresh`'s doc names no kind for a `ctx` that ends while it
  waits,** and the code gives two: an error matching both
  `ErrProviderUnavailable` and `ctx`'s when the end is seen between
  attempts (C46), and `ctx`'s error alone when it ends during the pause
  (C47). This is drift of D5's kind, in one of its four docs, and R52 asks
  the doc to name both. The question put on 2026-11-11 was whether D5 also
  documents the two, or a later record makes them one; neither was in this
  record's PLAN. *Ruled by the owner the same day: D5 documents both kinds
  now, in the PLAN's docs phase (R52). Making them one kind is deferred to
  a later record, after the cut-over, and no record is written for it now.
  It is listed as a candidate in the fleet-level record
  `0006-MADR-sequence-the-auth-contract-effort-across-four-repositories.md`,
  which is kept outside this repository, in the fleet's docs directory, to
  be raised after gobble-cli `v0.6.0` and magic-cli-remote's cut-over have
  shipped.*

### Revisions of 2026-11-11

A review against the code at `773b84c` and the sibling records as they stood
on 2026-11-11. The decisions are unchanged except where a bullet says so.

* **Context, error kinds.** The plan's revocation site (C3) is reached only
  for an issuer other than OpenAI's, whose endpoint is built in (C36); a
  Grok session reaches C2 at xAI's own issuer (C40). The PLAN's test now
  uses a test issuer for the plan.
* **Decision Drivers.** "A sentinel that is the wrong one cannot be mapped"
  was too strong: 0073-MADR D3 maps `errors.ErrUnsupported` too. A new
  driver records that a change of an error's kind ships in a minor, with
  v1.26.0 as the precedent (C41).
* **D3.** "Every request of the sign-in" now names the fetch of the
  issuer's keys (C37, C38), which the list left out, and says that
  discovery, which revocation shares (C39), takes the identity from the
  sign-in's configuration only. The PLAN's User-Agent test covers it.
* **D5.** `RevokeOAuthSession`'s doc also names `errors.ErrUnsupported`, for
  Grok only (C42); the correction names `llmprovider.ErrUnsupported` for all
  three cases.
* **"Found on review".** `LockRefresh`'s two `ctx` kinds (C46, C47), an open
  question.
* **"Not verified".** Narrowed by the live runs 0046-PLAN recorded (C43 to
  C45).
* **The siblings.** magic-cli-remote's
  `docs/decisions/0074-PLAN-remote-provider-auth-from-phone.md` P23, which
  the DF answer names with this record, was approved on 2026-11-11 and is in
  progress there; its `docs/decisions/0179-PLAN-gobble-native-acp-provider.md`
  P7 is still proposed. gobble-cli calls the wizard through
  `wizard.ResolveCredential` only, and no gobble record names
  `wizard.Options.OAuthClients`; the PLAN's R55 table is corrected, with an
  open question.

The owner's rulings of 2026-11-11, written in after the review above.

* **D4, the wizard's client option dropped.** `wizard.Options.OAuthClients`
  and `type wizard.OAuthClient` are removed from this record and its PLAN,
  since R55 asks for a consumer's call site and there is none. The README is
  corrected instead, so that it shows only what the wizard can do. D4 is
  rewritten in place, with what it said quoted; the chosen option is W4,
  added that day, where it was W2; two Consequences change and one is
  added. `auth.Identity`, `OAuthFlowOptions.Identity`,
  `ChatGPTPlanOptions.Identity` and `wizard.Options.Identity` stay.
* **The title.** It read "… Let a Consumer Name Itself and Choose Its OAuth
  Client in the Wizard, and Correct Four Docs". The PLAN's title and both
  rows in `docs/README.md` change with it. The file's name does not.
* **D5 and "Found on review", `LockRefresh`.** The open question is
  answered: D5 documents both `ctx` kinds now (C46, C47), and the PLAN's
  docs phase carries it. Making them one kind waits for a later record,
  after the cut-over, listed in the fleet-level record named there.
* **The PLAN's order.**
  [0077-PLAN-go-1-27-4-for-go-2026-6859.md](0077-PLAN-go-1-27-4-for-go-2026-6859.md),
  which raises the Go floor to 1.27.4, lands before any code of this
  record, and its change ships in this record's `v1.30.0`.

### Not verified

* Whether OpenAI accepts any `originator`, and xAI any `referrer`, that a
  consumer names (D3; the PLAN's owner-run probe). Both already accept this
  module's own value, which is neither vendor's own client's: the ChatGPT
  browser sign-in and the Grok device sign-in passed with it on the live
  runs 0046-PLAN recorded (C43, C44), and the ChatGPT test checks the
  `originator` before anyone signs in (C45).

### Related

* PLAN: [0076-PLAN-auth-error-kinds-and-doc-drift.md](0076-PLAN-auth-error-kinds-and-doc-drift.md).
* The error vocabulary these sentinels map to:
  [0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md) D3,
  and [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md) D8.
* The host id, S7's third option:
  [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md) D6.
* The Go floor that lands before this record's code:
  [0077-MADR-go-1-27-4-for-go-2026-6859.md](0077-MADR-go-1-27-4-for-go-2026-6859.md).
