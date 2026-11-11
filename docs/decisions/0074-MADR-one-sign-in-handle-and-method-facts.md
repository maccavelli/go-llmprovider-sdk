---
status: proposed
date: 2026-11-11
decision-makers: repository owner
consulted: "magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md (observed 2026-11-10, committed in that repository as 3e2f784b, unchanged at eefb7c25), its SDK findings F2, F6, F7, F12 to F14, F17, F18, F20, F21, F26, F27, F40 and F59, its ledger S1 to S6, S10, S12 and S13, and the owner's decisions of 2026-11-10; the SDK audit behind it; this repository at 773b84c; the consumers' proposed, untracked records as they stood on 2026-11-11: gobble-cli's 0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md and 0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md and their PLANs (gobble-cli origin/main 17d1605), and magic-cli-remote's 0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md and 0247-MADR-replay-a-pasted-sign-in-callback-to-the-agents-loopback.md and their PLANs (magic-cli-remote origin/master eefb7c25)"
informed: gobble-cli and magic-cli-remote, which drive sign-ins through this engine; every caller of llmprovider/auth and wizard
---

<!-- markdownlint-disable MD013 MD024 -->

# Describe Every Sign-In Method as Facts, and Drive Every Interactive Sign-In Through One Handle, Under `llmprovider/x/signin`

## Context and Problem Statement

A consumer that wants to run a sign-in somewhere other than a terminal (a
phone through magic-cli-remote, a script through gobble's `--json`, an ACP
client through gobble's `authenticate`) cannot learn from this SDK how a
method completes, and cannot hold a browser sign-in as an object.

* **The facts are missing, so every consumer guesses** (the 0245 report's
  F17, F18). `AuthMethod` has an id, a label, a detail and two booleans (C1).
  The booleans are never read outside tests: a search of the module for
  `.HeadlessOK` and `.Interactive` finds only the field and the descriptor
  literals and their test (run 2026-11-10). `HeadlessOK` is false for OpenAI's
  browser sign-in and for the ChatGPT plan (C2, C3), though both complete by a
  pasted URL (C4, C5). The wizard re-derives which methods save a session in
  a switch (C6), and its logout which providers revoke in another (C7).
* **Only the device code is a handle** (F6). `StartDeviceOAuth` returns a
  `DeviceLogin` with `Wait` and `Cancel` (C8, C9), as 0016-MADR D6 decided.
  The browser and plan sign-ins are single blocking calls whose authorize URL
  goes to an `OpenURL` callback whose error is discarded (C10, C11). This
  module's own live test says so: "LoginBrowserOAuth ignores an error OpenURL
  returns" (C12). The two browser flows build the same listener, result
  channel and select twice (C13, C14).
* **A bad paste ends the sign-in** (part of F12). A pasted URL whose state
  does not match, or that carries no code, is returned as the login's error
  (C15, C16), so a typo from a phone ends a ten-minute sign-in.
* **Two processes on one host, at most two OpenAI-style browser sign-ins**
  (F14). The OpenAI and plan sign-ins bind port 1455, else 1457 (C17), and
  fail with "registered callback ports unavailable" when both are taken
  (C18). A consumer whose browser is on another machine needs no listener at
  all, and cannot say so.
* **A session's kind is inferred from strings** (F26). Seven places decide
  between a ChatGPT, a plan, a Grok and another session from the issuer,
  `Resource` and `Provider` (C19 to C25). A new kind has to be taught to every
  one of them. Kilo's device token is an `OAuthSession` whose issuer is an API
  origin (C26).
* **Every stored field is copied by hand in five places** (C27 to C31): the
  store's `Load`, `persistable`, `refreshState`, `adopt` and the refreshed
  session. A field missed in one is lost on the first refresh. Any new field,
  this record's or 0075-MADR's, meets this.
* **The wizard speaks prose** (F7). `Prompter`'s questions have no id or kind
  (C32); only the device code reaches a caller as data, through
  `Options.ShowDeviceCode` (0057-MADR D1).
* **Vendor-file paths live in `wizard`, parsing in `auth`** (F40, C33, C34).

The owner decided on 2026-11-10 where this starts (the answer to the
report's Q1, option (a); the report's §11 holds the question and its
options, and the answer is recorded in
[0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md)'s
Decision Drivers): "the contract at v0, the engine under
`llmprovider/x/signin` (outside v1 api-check), promoted into
`llmprovider/auth` in a minor release once gobble and Magic have run it".
This record decides the engine: the facts, the
handle, the wizard on top of it, and what in `auth` changes under it.

### Evidence index

<!-- claims: self=773b84c -->

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| C1 | `AuthMethod`'s two booleans | `llmprovider/descriptor.go:34` | `HeadlessOK  bool` |
| C2 | OpenAI's browser sign-in has no `HeadlessOK` | `llmprovider/providers/openai/descriptor.go:28` | `Label:       "Sign in with ChatGPT",` |
| C3 | nor has the plan | `llmprovider/providers/openai/descriptor.go:54` | `Label:       "Use your ChatGPT plan with this app",` |
| C4 | The browser flow takes a paste | `llmprovider/auth/oauth_loopback.go:154` | `if config.inputCode != nil {` |
| C5 | and so does the plan | `llmprovider/auth/oauth_plan.go:144` | `if config.inputCode != nil {` |
| C6 | The wizard's store switch | `wizard/auth.go:120` | `func needsStore(provider llmprovider.ProviderID, method llmprovider.AuthMethodID) bool {` |
| C7 | The logout's revocation switch | `wizard/logout.go:43` | `case llmprovider.ProviderOpenAI, llmprovider.ProviderGrok:` |
| C8 | The device handle | `llmprovider/auth/oauth_device.go:83` | `type DeviceLogin struct {` |
| C9 | is started by `StartDeviceOAuth` | `llmprovider/auth/oauth_device.go:114` | `func StartDeviceOAuth(ctx context.Context, provider llmprovider.ProviderID, opts OAuthFlowOptions) (*DeviceLogin, error) {` |
| C10 | The browser flow drops `OpenURL`'s error | `llmprovider/auth/oauth_loopback.go:149` | `ignoreOAuthError(config.openURL(authorizeURL))` |
| C11 | and the plan's | `llmprovider/auth/oauth_plan.go:140` | `ignoreOAuthError(config.openURL(authorizeURL))` |
| C12 | The live test knows it | `llmprovider/live_oauth_login_test.go:24` | `LoginBrowserOAuth ignores an` |
| C13 | One listener | `llmprovider/auth/oauth_loopback.go:125` | `listeners, redirectURI, callbackPath, err := browserListener(config.provider)` |
| C14 | and a second | `llmprovider/auth/oauth_plan.go:123` | `listeners, redirectURI, _, err := browserListener(llmprovider.ProviderOpenAI)` |
| C15 | A paste is parsed in the input goroutine | `llmprovider/auth/oauth_loopback.go:160` | `input, inputErr = parseOAuthInput(input, state)` |
| C16 | and its error ends the login | `llmprovider/auth/oauth_loopback.go:179` | `return nil, callback.err` |
| C17 | Fixed ports | `llmprovider/auth/oauth_loopback.go:38` | `var openaiLoopbackPorts = []int{1455, 1457}` |
| C18 | Both taken, the login fails | `llmprovider/auth/oauth_loopback.go:369` | `registered callback ports unavailable; use device-code login` |
| C19 | Kind inferred: validation | `llmprovider/auth/oauth_session.go:85` | `if strings.TrimRight(session.Issuer, "/") != DefaultOpenAIIssuer \|\|` |
| C20 | Kind inferred: `Owner` | `llmprovider/auth/oauth_session.go:365` | `switch strings.TrimRight(s.Issuer, "/") {` |
| C21 | Kind inferred: `ChatGPT` | `llmprovider/auth/oauth_session.go:386` | `return issuer == DefaultOpenAIIssuer \|\| issuer == "" && s.Provider == llmprovider.ProviderOpenAI` |
| C22 | Kind inferred: `PlanUsage` | `llmprovider/auth/oauth_session.go:399` | `return s.Provider == llmprovider.ProviderOpenAI \|\|` |
| C23 | Kind inferred: the refresh body | `llmprovider/auth/oauth_session.go:704` | `if strings.TrimRight(state.issuer, "/") == DefaultOpenAIIssuer && state.resource == "" {` |
| C24 | Kind inferred: the token URL | `llmprovider/auth/oauth_session.go:893` | `case issuer == DefaultOpenAIIssuer, issuer == "" && state.provider == llmprovider.ProviderOpenAI:` |
| C25 | Kind inferred: revocation | `llmprovider/auth/oauth_revoke.go:57` | `case state.provider == llmprovider.ProviderOpenAI \|\| strings.TrimRight(state.issuer, "/") == DefaultOpenAIIssuer:` |
| C26 | Kilo's token as a session | `llmprovider/auth/kilo_device.go:121` | `return &OAuthSession{Provider: llmprovider.ProviderKilo, Access: approved.Token, Issuer: config.issuer,` |
| C27 | Copy 1: the store's `Load` | `llmprovider/auth/tokenstore_file.go:80` | `s := &OAuthSession{` |
| C28 | Copy 2: `persistable` | `llmprovider/auth/oauth_session.go:298` | `func (s *OAuthSession) persistable() *OAuthSession {` |
| C29 | Copy 3: `refreshState` | `llmprovider/auth/oauth_session.go:505` | `func (s *OAuthSession) refreshState() oauthSessionState {` |
| C30 | Copy 4: `adopt` | `llmprovider/auth/oauth_session.go:574` | `func (s *OAuthSession) adopt(next *OAuthSession) {` |
| C31 | Copy 5: the refreshed session | `llmprovider/auth/oauth_session.go:662` | `next = &OAuthSession{` |
| C32 | `Prompter` is prose | `wizard/prompter.go:71` | `type Prompter interface {` |
| C33 | Vendor paths in the wizard | `wizard/import.go:17` | `func vendorAuthPath(provider llmprovider.ProviderID, o Options) (string, error) {` |
| C34 | Vendor parsing in `auth` | `llmprovider/auth/vendor_session.go:103` | `func vendorCLI(provider llmprovider.ProviderID) (cli, hint string, parse func([]byte) (vendorCredential, error)) {` |
| C35 | The wizard passes only the client | `wizard/auth.go:723` | `return auth.OAuthFlowOptions{HTTPClient: o.HTTPClient}` |
| C36 | The descriptor pins compare whole structs | `llmprovider/providers/descriptors_test.go:117` | `if !reflect.DeepEqual(descriptor.AuthMethods, methods) {` |
| C37 | R4 | `docs/guides/api-standards.md:72` | `**R4. Experimental API lives under` |
| C38 | api-check allows `llmprovider/x/` | `scripts/check_api.py:36` | `EXPERIMENTAL = re.compile(r"^(?:\./\|package " + re.escape(MODULE) + r"/)llmprovider/x[/.:]")` |
| C39 | R30 names `GrokFlowFromEnv` | `docs/guides/api-standards.md:241` | `an exported function whose name ends` |
| C40 | `LoginDeviceOAuth` has a live caller | `llmprovider/live_oauth_login_test.go:106` | `session, err := auth.LoginDeviceOAuth(ctx, llmprovider.ProviderGrok, auth.OAuthFlowOptions{` |
| C41 | staticcheck runs on tests | `.golangci.yml:24` | `- staticcheck` |
| C42 | Six denials and expiries: the browser | `llmprovider/auth/oauth_loopback.go:594` | `return fmt.Errorf("%w: %w", llmprovider.ErrAuthFailure, err)` |
| C43 | Grok's device denial | `llmprovider/auth/oauth_device.go:368` | `return nil, fmt.Errorf("%w: oauth: Grok device authorization denied", llmprovider.ErrAuthFailure)` |
| C44 | Grok's device expiry | `llmprovider/auth/oauth_device.go:370` | `return nil, fmt.Errorf("%w: oauth: Grok device code expired", llmprovider.ErrAuthFailure)` |
| C45 | The shared expiry | `llmprovider/auth/oauth_device.go:439` | `var errDeviceCodeExpired = fmt.Errorf("%w: oauth: device code expired", llmprovider.ErrAuthFailure)` |
| C46 | Kilo's denial | `llmprovider/auth/kilo_device.go:106` | `Kilo device authorization denied` |
| C47 | Kilo's expiry | `llmprovider/auth/kilo_device.go:108` | `Kilo device code expired` |
| C48 | 0016-MADR D6 | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:324` | `The browser flow keeps its` |
| C49 | 0016-MADR D10 | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:367` | `**It does not drive agent CLIs.**` |
| C50 | Root `internal/` is the wizard's to import | `wizard/logout.go:8` | `"github.com/maccavelli/go-llmprovider-sdk/internal/redact"` |
| C51 | OpenAI's own device deadline is a plain error | `llmprovider/auth/oauth_device.go:234` | `return nil, errors.New("oauth: OpenAI device-code login timed out after 15 minutes")` |
| C52 | R2 lets only `wizard` leave the standard library | `docs/guides/api-standards.md:31` | `` Only `wizard` may leave the standard library, and then only for `` |
| C53 | The device-login variable is scoped to Grok and Kilo | `AGENTS.md:247` | `` `LLMPROVIDER_LIVE_DEVICE_LOGIN` (the Grok and Kilo `` |
| C54 | A pasted URL with this state and an `error` ends the login with the provider's error | `llmprovider/auth/oauth_loopback.go:619` | `if authErr := callbackError(query); authErr != nil {` |
| C55 | Without its own token URL, no other issuer's refresh token is sent | `llmprovider/auth/oauth_session.go:890` | `providers. Any other issuer's refresh token is never sent elsewhere` |

## Decision Drivers

* **The owner's decisions of 2026-11-10:**
  * **Q1:** the engine starts under `llmprovider/x/signin`, outside `make
    api-check` (C37, C38), and is promoted into `llmprovider/auth` in a minor
    release once gobble and Magic have run it.
  * **Q8:** its types are the contract module's
    ([0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md)).
  * **Q15:** the SDK maps its sentinels to the contract's `ErrorKind`.
  * **Q10:** a method's client identity is a fact a client can show.
  * **Q11:** a pasted callback is a secret.
* **The owner's standing constraints.** The reusable bulk of auth goes in
  this SDK, canonical and flexible; no gobble-specific features; Magic
  stays a relay; the agent CLIs own their own flows.
* **Refresh is re-dispatched, not rewritten** (the report's §5.4.3, and the
  owner's no-regression rule). Both vendors revoke a whole token family when
  a refresh token is reused (`llmprovider/auth/tokenstore.go:180-181`). The
  refresh, lock and rotation code keeps its 66 tests unchanged.
* **This repository's rules:** R48 for everything outside `llmprovider/x/`;
  `AuthMethod` and `OAuthSession` stay comparable, so no slice or map field
  (0043-MADR A1); R46, a moved test keeps its meaning; R50 to R52 and R55;
  R30, no ambient environment read.
* **0016-MADR D10 stays** (C49): the SDK drives no agent CLI and writes no
  vendor CLI's store.

## Considered Options

* **A. The engine under `llmprovider/x/signin`, with comparable facts on
  `AuthMethod`, a step engine in a root `internal/` package shared with the
  wizard, and `auth` changed only beneath its existing API.**
* **B. The engine added to `llmprovider/auth` at once,** under the v1 promise
  (the report's Q1 (b)).
* **C. Handles per flow, no step model:** a `BrowserLogin` beside
  `DeviceLogin`, with `AuthorizeURL`, `Complete(paste)`, `Wait` and `Cancel`
  (the SDK audit's seam (a)).
* **D. Wizard callbacks only:** an `Options.OnStep` sink in `wizard`, no
  engine (the SDK audit's seam (c)).

## Decision Outcome

Chosen option: "A", because it gives every consumer one handle and one set of
facts in the contract's terms, keeps every early mistake outside the v1
promise as the owner chose (Q1), and leaves the refresh code and every
existing exported name as they are. The owner chose this option on
2026-11-10 (Q1, with Q8, Q10, Q11 and Q15). This record is `proposed`: it
awaits the owner's review of this draft.

The owner also answered, the same day and as recommended, a point this draft
raised: `wizard.Options.Driver` may name the v0 contract module's types in
the v1 `wizard` API. The contract's own gate forbids breaking changes, so the
cost is accepted (D6; see the "Bad" consequence below).

On 2026-11-11 the owner ruled on the questions that day's review had left
open:

* `(*OAuthSession).SessionKind()` is exported (D5);
* `NoLoopback` is experimental only: `signin.Options.NoLoopback` stays, and
  no exported field is added to `auth` (D4) *(superseded later the same day
  by the owner's reversal, below)*;
* the three rules corrected that day are confirmed: a paste step follows the
  method's `Paste` fact (D3), three pastes in all and a pasted denial ends
  the sign-in (D4), and the issuer guard stays on the refresh's token URL
  (D5);
* one revocation table (the report's S12) is deferred past the cut-over
  ("Where the report and the audit differ").

Later on 2026-11-11 the owner reversed the second ruling:
`OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback` are part
of the v1 API, and `signin.Options.NoLoopback` passes through to them, as D4
first designed. gobble-cli sets it for a sign-in started from a remote ACP
client, which meets R55's call-site rule (D4). *(This paragraph read: "The
second ruling leaves one point for the owner: how `x/signin`'s option
reaches `auth` without an exported field (D4's open question)." The
reversal closes that question.)* The record stays `proposed`: the owner has
answered questions, not accepted it.

### D1. Method facts on `llmprovider.AuthMethod`

`AuthMethod` gains comparable fields, each a plain string or a boolean, so the
struct stays comparable (0043-MADR A1):

| Field | Type | Holds |
| :--- | :--- | :--- |
| `Kind` | `string` | an `authcontract.Kind` value |
| `Completion` | `string` | an `authcontract.Completion` value |
| `Paste` | `string` | an `authcontract.Paste` value: what a paste may be |
| `FirstPaste` | `string` | the same, for a first sign-in when it differs |
| `Yields` | `string` | an `authcontract.Yields` value |
| `ClientIdentity` | `string` | an `authcontract.ClientIdentity` value |
| `Slot` | `string` | the stored credential the method shares with others; empty when it stores none |
| `Refreshable` | `bool` | the credential refreshes itself |
| `Revocable` | `bool` | a logout revokes it at the issuer |

0075-MADR D1 adds the consent fields.

* **Why plain strings, not the contract's types.** `llmprovider` is under the
  v1 promise and may import only the standard library and its internal
  packages (R2). A field of a v0 module's type would tie the v1 API to it.
  `signin.Methods` (D2) converts; a test (D12) holds every built-in value to
  the contract's constants.
* **The built-in values,** set in each descriptor:

| Provider, method | Kind | Completion | Paste (first) | Yields | Client | Slot | Refr. | Rev. |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| openai `api_key` | `api_key` | `value` | | `api_key` | `none` | | no | no |
| openai `browser_oauth` | `browser` | `browser_paste` | `url_or_code` | `oauth_session` | `vendor_borrowed` | `openai` | yes | yes |
| openai `device_code` | `device_code` | `device` | | `oauth_session` | `vendor_borrowed` | `openai` | yes | yes |
| openai `token_stdin` | `token_paste` | `value` | | `access_only` | `none` | `openai` | no | yes |
| openai `import_vendor_cli` | `vendor_import` | `local_file` | | `read_through` | `none` | | no | no |
| openai `chatgpt_plan` | `browser` | `browser_paste` | `url_or_code` (`url_only`) | `oauth_session` | `app_registered` | `openai` | yes | yes |
| grok `api_key` | `api_key` | `value` | | `api_key` | `none` | | no | no |
| grok `browser_oauth` | `browser` | `browser_loopback` | | `oauth_session` | `vendor_borrowed` | `grok` | yes | yes |
| grok `device_code` | `device_code` | `device` | | `oauth_session` | `vendor_borrowed` | `grok` | yes | yes |
| grok `token_stdin` | `token_paste` | `value` | | `api_key` | `none` | | no | no |
| grok `import_vendor_cli` | `vendor_import` | `local_file` | | `read_through` | `none` | | no | no |
| kilo `api_key` | `api_key` | `value` | | `api_key` | `none` | | no | no |
| kilo `device_code` | `device_code` | `device` | | `device_key` | `none` | `kilo` | no | no |

* **Grok's browser sign-in is `browser_loopback`,** with no paste, although
  the code accepts one (C4). Whether accounts.x.ai ever shows a pasteable URL
  or code is unverified (the report's F13, U1; 0008-MADR F5 calls the
  delivery "inferred"). The fact changes to `browser_paste` only after the
  owner-run probe of D12 shows a paste completing it.
* **OpenAI's pasted token yields `access_only`,** the credential that needs a
  store and is revocable. A pasted `sk-` key is taken as an API key and saves
  nothing (`wizard/auth.go`'s `resolveTokenStdin`). The fact names the
  stronger case; the field's doc says so.
* **The two old booleans keep their values** and are deprecated (D7).

### D2. `signin.Methods`: the facts as the contract's `Method`

```go
// Methods returns d's sign-in methods as the contract describes them, in
// menu order. A descriptor that lists none but requires a key gets one
// api_key method.
func Methods(d llmprovider.Descriptor) []authcontract.Method
```

* Each `AuthMethod` maps field for field, with `SchemaVersion` set, and
  `Features` from its facts (`paste` when `Paste` is set).
* **Two contract values and one field have no source here.** No built-in
  method is `oauth` with `at_start`, or carries a `command`: each of this
  SDK's methods knows its mechanism, and it runs no host command (D11). They
  are for a consumer that describes another program's methods (0073-MADR D2),
  so `AuthMethod` gains no field for them and `Methods` never sets them.
* **An implicit API key is made explicit here, not in the descriptors.** Seven
  of the ten built-in descriptors (`llmprovider/providers/providers.go`'s
  `builtins`), and `chatcompat.Descriptor`, list no methods. Adding an `api_key` entry to them would change
  what a consumer that tests `len(d.AuthMethods) == 0` sees, though the
  wizard's own branch would not change (`wizard/auth.go`'s `signIn` and
  `skippedEvery` treat no methods and one `api_key` alike). `Methods` gives the
  contract its explicit method and leaves every descriptor as it is.

### D3. One handle: `signin.Start` and `*signin.SignIn`

```go
package signin // llmprovider/x/signin

type Options struct {
    HTTPClient *http.Client
    ClientID   string        // OpenAI's or Grok's client; ignored by Kilo and the plan
    Issuer     string        // OpenAI's or Grok's issuer; Kilo's origin
    Identity   auth.Identity // 0076-MADR D3
    AgentName  string        // the plan's app name; required for chatgpt_plan
    Previous   *auth.OAuthSession
    NoPaste    bool // no paste step: the loopback alone completes a browser sign-in
    NoLoopback bool // no listener: a paste alone completes it (D4)
    LookupEnv  func(string) string // import_vendor_cli's paths (D10)
    // 0075-MADR adds ConsentText (its D4), HostID (D6) and ClientStore (D7).
}

func Start(ctx context.Context, p llmprovider.ProviderID, m llmprovider.AuthMethodID, o Options) (*SignIn, error)

type SignIn struct{ /* unexported */ }
func (s *SignIn) Method() authcontract.Method
func (s *SignIn) Steps() <-chan authcontract.Step
func (s *SignIn) Answer(ctx context.Context, a authcontract.Answer) error
func (s *SignIn) Wait(ctx context.Context) (Credential, error)
func (s *SignIn) Cancel()

// Credential is what a sign-in yields: APIKey, Session, AccessOnly, DeviceKey or ReadThrough.
type Credential interface {
    Yields() authcontract.Yields
    TokenSource() llmprovider.TokenSource
    credential()
}
```

* **Methods it drives:** `device_code` (OpenAI, Grok, Kilo), `browser_oauth`
  (OpenAI, Grok), `chatgpt_plan`, and `import_vendor_cli`, which reads the
  file and is done at once. A value method (`api_key`, `token_stdin`) and
  any other pair give an error matching `llmprovider.ErrUnsupported`: the
  caller reads the value itself.
* **Steps:**
  * a device sign-in: `show_code` (URL, code, expiry), `waiting`, then
    `done` or `failed`;
  * a browser sign-in whose method takes a paste (its `Paste` fact is set,
    D1): `open_url` with `accepts`, the method's `Paste`, then `paste`, then
    `done` or `failed`. With `NoPaste`, `open_url` carries no `accepts` and
    no `paste` step follows;
  * a browser sign-in whose method takes no paste, Grok's `browser_loopback`
    (D1): `open_url` with no `accepts`, then `done` or `failed`, whatever
    `NoPaste` says. Only the loopback completes it, so no step asks, and an
    `Answer` is refused as any answer to a step that does not ask (below).
    `auth`'s `LoginBrowserOAuth` still takes an `InputCode` for Grok (C4);
    the engine passes none. *(2026-11-11: the rule read `NoPaste` alone and
    would have offered Grok a paste its fact denies. Confirmed by the owner
    the same day.)*;
  * the plan: 0075-MADR D4's `confirm` first, then as a browser sign-in, its
    first paste `url_only` (its `FirstPaste`, D1);
  * a vendor import: `done` or `failed`.
* **Every `paste` step carries `secret: true`** (0073-MADR D2; Q11): a pasted
  callback or code is worth a token. A `form` step carries it when any of its
  fields is of type `secret`.
* **The channel** is buffered for the whole sign-in, so the engine never
  blocks on a slow reader, and is closed after `done` or `failed`. `Seq`
  rises by one per step.
* **`Answer`** takes the answer to the step whose `Seq` it names. Only a
  `paste`, `confirm` or `form` step asks; `open_url`, `show_code`, `waiting`,
  `done` and `failed` are shown, never answered. An answer to a step that
  does not ask, to a step already answered, to any other `Seq`, or after the
  end, gives an error matching `llmprovider.ErrInvalidRequest`. A consumer
  whose transport wants a shown step acknowledged (gobble's ACP relay, where
  every step is an elicitation that the client accepts, gobble-cli
  `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D4, proposed and untracked in gobble-cli on 2026-11-11) keeps that acknowledgement to itself and calls `Answer` for nothing
  but the asking steps: an `open_url` that accepts a paste is followed by its
  own `paste` step, and the paste is that step's answer. Before it logs or keeps anything of the
  answer, the engine sets its `Secret` from the step it holds for that `Seq`,
  whatever the caller sent (0073-MADR D6 rule 5); the wizard does the same
  with a `Driver`'s answer (D6).
* **`Wait`** returns the credential, or the sign-in's error. It does not save:
  the caller saves, as the wizard does after its listing (0059-MADR W2).
* **`Cancel`** ends the sign-in: `failed` with `canceled`, and `Wait`'s error
  matches `context.Canceled`.
* **A concrete type, not an interface,** so that a method can be added after
  promotion without breaking an implementer (R48), as `DeviceLogin` is.
* **`DeviceLogin`, `LoginBrowserOAuth`, `StartDeviceOAuth` and
  `LoginChatGPTPlan` stay** as they are (R48). The engine calls them: it adds
  no second flow.

### D4. Under the handle: paste, one listener, and no listener

* **A bad paste asks again.** The engine checks a pasted value before it
  reaches `auth`: a URL must carry the `state` of the authorize URL the
  `open_url` step showed and a `code`, and, where the step accepts
  `url_only`, a `client_id`; a bare code is taken only where the step accepts
  `url_or_code`. A paste that fails is answered with a new `paste` step whose
  `text` says why, never echoing the paste. A sign-in takes at most three
  pastes: each of the first two bad ones brings a new `paste` step (two
  re-asks), and a third bad one ends the sign-in, `failed` with
  `invalid_request`. `auth`'s own parsing (C15) is unchanged. *(The count,
  three pastes in all and two re-asks, confirmed by the owner, 2026-11-11.)*
* **A pasted denial is not a bad paste.** A pasted URL that carries this
  sign-in's `state` and an `error` parameter is the identity provider's
  answer. The engine passes it to `auth`, whose parsing ends the sign-in as
  it does today (C54): an `access_denied` matches `ErrAuthFailure` (C42),
  and with D8 `ErrSignInDenied`, so the sign-in ends `failed` with `denied`;
  another code is a plain error, `other`. It is never asked again.
  *(2026-11-11: the rule above would have re-asked it, three times, and
  ended `invalid_request`, unlike today's code. Confirmed by the owner the
  same day.)*
* **One listener, paste and select in `auth`** (the report's S5). The
  listener, result channel, input goroutine and select that
  `LoginBrowserOAuth` and `LoginChatGPTPlan` each build (C13, C14) become one
  unexported function both call, with the callback parsing, the CORS origin
  and the exchange as its parameters. Every loopback and plan test passes
  unchanged.
* **No listener, when asked** *(the owner's ruling of 2026-11-11, later the
  same day, which reverses that morning's "`NoLoopback` is experimental
  only"; the note at the end of this bullet quotes what it replaces)*.
  `OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback` are part
  of the v1 API ("exported, additive"), and `signin.Options.NoLoopback`
  (experimental, in `x/signin`) passes through to them, as this bullet first
  designed. Each binds nothing: the redirect URI names the first registered
  port, and only a paste completes the sign-in. Two consumers on one host no
  longer contend for ports 1455 and 1457 (F14) when neither needs the
  loopback.
  * **In `auth`.** The one browser function of the bullet above binds no
    listener and needs an `InputCode`, the only thing that can complete the
    sign-in; without one it is an error matching `ErrInvalidRequest`. For
    Grok, whose paste is unverified (D1), it is an error matching
    `ErrUnsupported`, before any request.
  * **In `x/signin`.** With a method whose `Paste` fact is empty it is an
    error matching `ErrUnsupported`, as for Grok; with `NoPaste` as well it
    is `ErrInvalidRequest`, since nothing could complete the sign-in. Both
    are returned by `Start`, before any request.
  * **Its callers.** gobble-cli sets `signin.Options.NoLoopback` for a
    sign-in started from a remote ACP client. The person completing that
    sign-in is not on the agent's host (a phone through magic-cli-remote,
    for example), so the host's loopback listener can never receive the
    callback and only a paste completes it; setting it also frees ports
    1455 and 1457 for other agents on that host (gobble-cli
    `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
    D3, proposed and untracked in gobble-cli on 2026-11-11). gobble learns
    that a sign-in is remote from `_meta.gobble.remote: true` on
    `authenticate`, which magic-cli-remote sends on every sign-in the phone
    starts (magic-cli-remote
    `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
    D8). That field is the consumer's context: the SDK reads nothing of it.
    `x/signin`'s `Start` is the call site of the two `auth` fields. D12's
    U2 and U3 probes call `x/signin` with it too.
  * **No exported field in `auth`.** *(Superseded 2026-11-11 by the owner's
    reversal, later the same day; kept as it was written.)* As first
    written, this bullet added
    `OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback`
    ("exported, additive"), under the v1 promise, and `x/signin` passed its
    option to them. The owner ruled on 2026-11-11 that neither is added
    until a consumer's record sets one. R55 asks each exported name for the
    call site that needs it, and no decision record, plan or source file of
    gobble-cli or magic-cli-remote names `NoLoopback` (their Go, Dart and
    Markdown files, searched 2026-11-11); the one mention is the 0245
    report's own sketch of the engine's options (its `:602`).
  * **Open question (2026-11-11), for the owner: how the option reaches
    `auth`.** *(Closed 2026-11-11 by the owner's reversal, later the same
    day, which chose (b); kept as it was put.)* The listener is bound
    inside `auth` (C13, C14; after this record, in the one unexported function of the bullet above). `x/signin`
    runs a browser sign-in only by calling `LoginBrowserOAuth` and
    `LoginChatGPTPlan` (D3), and their exported option structs were the
    route. `internal/signinflow` cannot carry the option: it imports `auth`
    (the PLAN's Phase 4, step 4), so `auth` cannot import it. R37 allows no
    hook that `auth` would register at start-up. What those two functions
    take besides their option structs is a `context.Context`. The options:
    * (a) *proposed:* a mark on the context, set and read through a small
      package under `llmprovider/internal/`, which `auth` and `x/signin` may
      both import and nothing outside the module can. `x/signin` marks the
      context it gives the two login functions. The one browser function
      reads the mark and binds nothing; it then needs `InputCode`, else an
      error matching `ErrInvalidRequest`, and for Grok it is
      `ErrUnsupported`. Cost: one more internal package (an R2 row, R47's
      80 %); a value read from a context, where at `773b84c` no file
      outside the tests calls `context.WithValue`, and where the series'
      one other mark, 0075-MADR D5's `WithBackground`, is exported and
      documented; and two exported functions whose behaviour a caller
      inside the module changes by a route their signatures do not show.
    * (b) the two exported fields after all. Cost: two names under the v1
      promise with no consumer call site, against the ruling above.
    * (c) no `NoLoopback` anywhere until a consumer's record asks:
      `signin.Options.NoLoopback` is dropped too, and the U2 and U3 probes
      complete by a paste with the listener bound, which a paste already
      wins today (C4, C5). Cost: F14's contention for the two ports stays,
      and the ruling keeps the option.

    Until the owner rules, the PLAN builds no `NoLoopback` step: each one
    is marked as a stop. *(Lifted by the reversal: the PLAN's `NoLoopback`
    steps are restored.)*

  *(2026-11-11: rewritten in place by the owner's ruling. The first version
  is quoted in the first sub-bullet; its other sentences are kept above.)*

  *(2026-11-11, later the same day: rewritten again by the owner's
  reversal. Under the morning's ruling this bullet's lead read "No
  listener, when asked, in `x/signin` only" with "the owner's ruling of
  2026-11-11: `NoLoopback` is experimental only", and its body ended "Its
  callers are D12's U2 and U3 probes. No consumer record sets it:
  gobble-cli's proposed 0051-MADR D3 lists the `signin.Options` it sets
  without it." The ruling's three options were put as kept above: (a) a
  mark on the context, through a small package under `llmprovider/internal/`
  that `auth` and `x/signin` both import; (b) the two exported fields after
  all; (c) no `NoLoopback` anywhere, `signin.Options.NoLoopback` dropped
  too. The owner chose (b). Its stated cost, "two names under the v1
  promise with no consumer call site", no longer holds: R55's call-site
  rule is met, not overridden, by gobble-cli's 0051-MADR D3. The behaviour
  of the "In `auth`" and "In `x/signin`" sub-bullets is the one this
  bullet, option (a) and the PLAN's tests already specified.)*
* **An `OpenURL` that cannot deliver is seen** (F6). The engine has no
  `OpenURL`: the `open_url` step is the delivery, and a reader that cannot
  deliver it calls `Cancel`. In the wizard (D6), an error from
  `Options.OpenURL` is shown as a warning notice and the sign-in goes on,
  since the URL is already on screen; a `Driver`'s error ends it.

### D5. An explicit, persisted session kind

* **`OAuthSession.Kind`,** of the new type `auth.SessionKind`, with
  `SessionChatGPT` (`"chatgpt"`), `SessionChatGPTPlan` (`"chatgpt_plan"`),
  `SessionGrok` (`"grok"`) and `SessionKiloDevice` (`"kilo_device"`). Every
  sign-in sets it. `FileTokenStore` persists it as `kind`, `omitempty`.
* **Empty means inferred,** by exactly today's rules, gathered into one
  unexported function, so a record from an older version, or a session a
  caller builds, behaves as today.
* **`(*OAuthSession).SessionKind() SessionKind`,** exported *(the owner's
  ruling of 2026-11-11; it was added that day as a proposed revision)*:
  `Kind` when set, else the inferred kind, read under the session's lock as
  `Owner`, `ChatGPT` and `PlanUsage` read theirs; empty when neither names
  one. gobble-cli's proposed records need
  the kind of a session saved before this release, whose `Kind` is empty:
  its resolver accepts a stored session "of an offered kind", "inferred by
  the SDK's rules when empty" (gobble-cli
  `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D2), and `auth status --json` reports `sessionKind` (gobble-cli
  `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` D8).
  With the inference unexported, gobble would copy the rules, and the two
  copies could drift. *Ruled by the owner, 2026-11-11: the method is
  exported. The question put that day was whether to add it or to record
  gobble's need as unmet. The PLAN's R55 table lists it with those two call
  sites.*
* **Re-dispatched, not rewritten.** The seven sites (C19 to C25) test the
  kind instead of the issuer and fields. Each branch's body is unchanged. The
  refresh, lock, rotation and save code is not edited beyond the five field
  copies (C27 to C31), which gain the new field.
* **The issuer guard stays where it is a guard.** The refresh's token URL
  (C24) sends a refresh token to a built-in endpoint only for a session
  whose issuer is a built-in one, or empty: any other issuer's refresh token
  is sent nowhere unless the session holds its own token URL (C55; 0020-MADR
  F45). The kind chooses among the branches; it never sends a token where
  the issuer would not. A `grok` or `chatgpt` session whose issuer is a
  caller's own, such as one from `GrokFlowFromEnv` (D7), still gets no
  built-in token URL. *(2026-11-11: made explicit; "test the kind instead of
  the issuer" read alone would have sent such a token to xAI's or OpenAI's
  endpoint. Confirmed by the owner the same day.)*
* **A contradiction is refused:** `ValidateOAuthSession` refuses a session
  whose `Kind` disagrees with its fields (a `chatgpt_plan` with no
  `Resource`, a `kilo_device` with a refresh token), with a plain error.
* **The tests that must stay green, unchanged:** the 66 functions in
  `oauth_refresh_test.go`, `oauth_refresh_0068_test.go`,
  `oauth_refresh_kinds_test.go`, `oauth_rotation_test.go`,
  `oauth_rotation_kept_test.go`, `oauth_spent_refresh_test.go`,
  `oauth_shared_store_stress_test.go`, `oauth_session_leader_test.go`,
  `oauth_session_test.go`, `oauth_wait_0071_test.go`,
  `plan_session_0043_test.go`, `oauth_session_invalidate_test.go`,
  `tokenstore_oslock_test.go`, `lockwait_0068_test.go` and
  `session_store_kinds_0048_test.go`, all in `llmprovider/auth`. The PLAN
  names each one.
* **A field-copy guard.** A test sets every exported stored field of a
  session to a distinct value and passes it through each of the five copies
  and a store round trip, failing on any field that does not come through.
  It holds this record's field and 0075's.

### D6. The wizard renders steps, and hands them to a `Driver`

* **One step engine, in `internal/signinflow`** (root `internal/`, which
  `wizard` may import as it imports `internal/redact`, C50; R2 gains the
  row). It runs a sign-in from the login functions it is given and produces
  `authcontract.Step` values. `x/signin` gives it `auth`'s functions; the
  wizard gives it its own test seams, so every wizard test keeps its seams and
  its meaning (R46).
* **The wizard's renderer** shows each step as today, word for word:
  `open_url` as "Open %s in your browser" and the paste instruction,
  `show_code` through `ShowDeviceCode` or "Open %s and enter code %s",
  `paste` through `Input` or `InputContext` with today's prompts, `confirm`
  through 0075-MADR D4's rendering. Any wizard test that needs a changed
  assertion stops the work (the PLAN's stop condition).
* **`Options.Driver`,** of the new interface type `StepDriver`, hands each
  sign-in step to a caller with no terminal:

  ```go
  type StepDriver interface {
      SignInStep(ctx context.Context, provider llmprovider.ProviderID, step authcontract.Step) (authcontract.Answer, error)
  }
  ```

  * It is called once per step, on the flow's goroutine. For `paste`,
    `confirm` and `form` it returns the answer; for the others a zero
    `Answer`. An error ends the sign-in and is returned as it is.
  * It must return soon after `ctx` ends, as `InputerContext` must
    (0057-MADR D3), so that a loopback that wins ends a pending paste.
  * With a `Driver`, `ShowDeviceCode`, `OpenURL` and the `Prompter` are not
    used for sign-in steps. `NoPastePrompt` still decides whether a paste is
    asked. The menus still use the `Prompter`.
  * It speaks the contract's types only, never `x/signin`'s, so no
    experimental type enters `wizard`'s v1 API. The contract's own rules
    (0073-MADR D6, D7) forbid a break even at v0.
* **`ShowDeviceCode` and `NoPastePrompt` stay** (0057-MADR D1, D2), as
    special cases of the renderer.

### D7. What is deprecated, and what is not

* **Deprecated** (`Deprecated:` in the doc, R42; nothing removed before a v2):
  * `AuthMethod.Interactive` and `AuthMethod.HeadlessOK`, in favour of D1's
    facts. Their values stay as they are.
  * `auth.LoginDeviceOAuth` and `OAuthFlowOptions.NotifyDevice`, in favour of
    `StartDeviceOAuth` or `x/signin`. The two live tests that call it (C40)
    move to `StartDeviceOAuth` and `Wait`, the flow the wizard already uses.
* **staticcheck runs on tests** (C41). Whether its SA1019 check reports the
  descriptor literals that still set the deprecated fields is unverified;
  the PLAN's lint step shows it. If it does, each literal carries a
  `//nolint:staticcheck` naming SA1019 and this record, which `nolintlint`
  then requires to be needed.
* **Not deprecated, against the report's S2:**
  * `GrokFlowFromEnv`: R30 names it as the pattern of an opt-in environment
    helper (C39). *(2026-11-11: this went on "and 0076-MADR D4 gives its
    values a way into the wizard". The owner dropped that option from 0076
    the same day, so the helper's values reach a sign-in on a caller's own
    `OAuthFlowOptions` only, as today. R30's naming it is reason enough to
    keep it.)*
  * `ValidateOAuthSession`'s refusal of the `chatgpt-access` fixture: removing
    it changes what the function accepts, which is not this record's to
    change.

### D8. One error vocabulary: `signin.ErrorKindOf`, and two sentinels in `auth`

* **`signin.ErrorKindOf(err error) authcontract.ErrorKind`** maps an error to
  0073-MADR D3's kinds, most specific first, in that table's order. An error
  that matches none is `other`. `Failure.Retryable` is the `*APIError`'s
  `Retryable()` when there is one, else true for `rate_limited`, `offline`
  and `unavailable`.
* **Two sentinels in `auth`,** so that `denied` and `expired` are told apart:
  `ErrSignInDenied` and `ErrSignInExpired`, each matching
  `llmprovider.ErrAuthFailure` as well. The six sites that return a denial or
  an expiry (C42 to C47) wrap their error so that it also matches the new
  sentinel, and its text is unchanged, byte for byte: every existing error
  message and every existing `errors.Is` stays as it is.
* **One expiry depends on 0076.** At `773b84c` the OpenAI device poll's own
  15-minute deadline returns a plain error (C51), which matches no sentinel
  and maps to `other`. [0076-MADR-auth-error-kinds-and-doc-drift.md](0076-MADR-auth-error-kinds-and-doc-drift.md)
  D2 makes it the shared expiry, C45, which this record wraps as
  `ErrSignInExpired`. The owner decided on 2026-11-10 that 0076 lands first
  (its DF answer), so this record builds on it; the PLAN names it as a
  prerequisite and tests the kind.

### D9. Promotion, and the rules until then

* `llmprovider/x/signin` is outside `make api-check` (R4, R48; C37, C38): it
  may change in any release until promoted.
* **Promotion** is a later record's, once gobble-cli and magic-cli-remote
  have each shipped a release that drives a sign-in through it (the owner's
  Q1). It moves `Start`, `SignIn`, `Options`, `Credential`, `Methods` and
  `ErrorKindOf` into `llmprovider/auth`, in a minor release, and the contract
  module to `v1.0.0` in step (0073-MADR D1).
* Everything this record adds outside `x/signin` is under the v1 promise from
  its release: D1's fields, D4's `NoLoopback` fields on `OAuthFlowOptions`
  and `ChatGPTPlanOptions`, D5's kind and its `SessionKind` method, D6's
  `Driver` and `StepDriver`, D8's sentinels and D10's function.
  *(2026-11-11, by the owner's rulings: `SessionKind` was listed "if the
  owner adopts it", and is adopted; "D4's `NoLoopback` fields" were listed,
  and are not added.)* *(Later the same day, by the owner's reversal: D4's
  two `NoLoopback` fields are listed again, and added.)*

### D10. Vendor-file paths move into `auth`

* `auth.VendorCLIPath(provider llmprovider.ProviderID, lookupEnv func(string) string) (string, error)`
  resolves a vendor CLI's auth file as the wizard does today (C33): Codex
  from `CODEX_HOME`, Grok from `GROK_AUTH_PATH` or `GROK_HOME`, else under the
  home directory from `HOME`, or `USERPROFILE` on Windows. It reads only
  through `lookupEnv` (R30; 0015-MADR D9); a nil one finds no home, and the
  error says to pass one, as today.
* A provider with no vendor CLI gives an error matching `ErrUnsupported`.
* The wizard calls it; so does `x/signin` for `import_vendor_cli`.

### D11. What does not change: 0016-MADR D10

The engine runs this SDK's own flows only. It starts no process, drives no
agent CLI, and writes no vendor CLI's store (C49). The mappings from codex,
grok, kilo and opencode into the contract's steps stay in magic-cli-remote.

### D12. Tests and probes

* **Offline**, each seen to fail on a planted break (the PLAN names them):
  the descriptor facts, held to the contract's constants; `Methods`; each
  method's step sequence, Grok's browser with no paste step among them,
  answers, a bad paste, a pasted denial, `Cancel`, `NoPaste` and
  `NoLoopback`; the session kind on each site, set and inferred, a session
  from a caller's own issuer among them; the
  field-copy guard; the wizard's texts unchanged and its `Driver`;
  `ErrorKindOf` over every sentinel; the two new sentinels with every message
  unchanged; `VendorCLIPath`.
* **`providers/descriptors_test.go`'s pins** (C36) gain the facts. They keep
  comparing whole structs, so the deprecated booleans stay pinned at their
  values: what the test asserts does not change, only what it covers grows.
* **Live, owner-run, each asking first, each with a scratch store:**
  * a live OpenAI device sign-in (no live test has one; the report's F59),
    under `LLMPROVIDER_LIVE_DEVICE_LOGIN`, whose scope AGENTS.md names as the
    Grok and Kilo device logins (C53) and which the PLAN widens to OpenAI's;
  * the two live device tests D7 moves to `StartDeviceOAuth`
    (`TestLive_GrokDeviceLogin`, `TestLive_KiloDeviceLogin`), run once after
    the move, since R46's "same login, same checks" is shown only by a run;
  * the report's U2: OpenAI's browser sign-in completed by a URL pasted from
    another device, through `x/signin` with `NoLoopback`;
  * the report's U3: the plan sign-in completed by a pasted full URL from
    another device, and its port fallback;
  * the SDK side of U1: Grok's browser sign-in opened on another device,
    recording whether accounts.x.ai shows anything pasteable.

### Consequences

* Good, because a consumer reads how each method completes, what it yields
  and whose client it signs in as, from data, and drives every interactive
  sign-in through one handle and one set of steps.
* Good, because the wizard's non-terminal callers get the steps as data
  through `Driver`, and the terminal wizard is unchanged.
* Good, because no refresh code is rewritten: kinds re-dispatch it, and the
  field-copy guard holds every copy.
* Good, because a phone's typo no longer ends a sign-in, and two remote
  sign-ins no longer contend for the loopback ports.
* Neutral, because the engine adds no flow: it calls the existing exported
  ones, which stay.
* Neutral, because Grok's browser sign-in stays loopback-only until a probe
  shows otherwise.
* Bad, because `wizard`'s v1 API (`StepDriver`) names a v0 module's types. The
  contract's gate forbids breaking them, but a mistake there now reaches
  `wizard` too.
* Bad, because `AuthMethod` grows nine fields, and `OAuthSession` one field
  and one method (`SessionKind`, added 2026-11-11), and `OAuthFlowOptions`
  and `ChatGPTPlanOptions` one field each (`NoLoopback`, restored by the
  owner's reversal of 2026-11-11), each under the v1 promise.
* Bad, because the root module gains its first requirement outside
  `golang.org/x`, its own nested module, and three gates learn it (the PLAN's
  Phase 4).

### Confirmation

* Every test of D12 passes, each seen to fail on its planted break.
* The 66 refresh and store tests pass with no change to their files.
* `make api-check` passes: the additions only; `tools/apidiff.py` lists
  exactly the PLAN's table of exported names, with `x/signin`'s marked
  experimental.
* `make dep-check`, `make tables-check` and `make coverage-check` pass, with
  the contract module and the new packages in them.
* `tools/claims.py` passes on this record.
* The owner-run probes are recorded in the PLAN, with their outcomes, before
  D1's Grok fact or the promotion changes.

## Pros and Cons of the Options

### A. `x/signin`, facts on `AuthMethod`, a shared step engine

* Good, because early mistakes in the handle are free to fix until
  promotion (Q1).
* Good, because the wizard and `x/signin` share one engine, and the wizard's
  seams and tests stay.
* Neutral, because consumers adopt an experimental package, which may change
  between minor releases.
* Bad, because the facts on `AuthMethod` are under the v1 promise at once:
  a wrong fact is fixed by its value, never by removing its field.

### B. The engine in `auth` at once

* Good, because consumers adopt a stable API once.
* Bad, because every early mistake is frozen under `make api-check`.
* Bad, because the owner chose against it (Q1 (b) was not chosen).

### C. Handles per flow, no step model

* Good, because it is close to the code that exists (the audit's seam (a)),
  and `DeviceLogin` is the precedent.
* Bad, because each consumer still turns a handle's fields into its own
  events, and the plan's consent, a question before the browser, has no
  place in it.
* Bad, because it leaves the contract module without a producer in the SDK.

### D. Wizard callbacks only

* Good, because it is the smallest change.
* Bad, because a consumer that does not use the wizard, as gobble's ACP
  `authenticate` would not, gets nothing.
* Bad, because the facts would still be decided by each consumer.

## More Information

### Records this decision amends, applied on acceptance

These notes are applied by the PLAN's Phase 0, step 3. No existing record is
edited before this record is accepted.

* **`0006-MADR-subscription-auth-for-llm-providers.md`, "2. Descriptors grow
  an auth-method list"** (`:573-579`), whose `AuthMethod` has `Interactive
  bool // wizard may offer it` and `HeadlessOK bool // device-code, stdin,
  import`: superseded in part. Both are deprecated; D1's facts replace them.
* **`0016-MADR-provider-auth-and-support-baseline.md`, D6** ("`auth` starts a
  device login and returns a handle … The browser flow keeps its `OpenURL`
  and `InputCode` callbacks", C48): generalised. Every interactive method
  has a handle, `x/signin`'s; the device handle and the callbacks stay.
* **`0016-MADR-provider-auth-and-support-baseline.md`, D10** (C49): affirmed,
  unchanged (D11).
* **`0057-MADR-wizard-device-code-and-paste-prompt.md`, D1 and D2:** kept, as
  special cases of D6's renderer; a note says so.
* **`0056-MADR-wizard-credential-only-entry-point.md`, D2**
  (`Options.KiloDeviceKey`): kept; a note says the Kilo method's `yields` is
  `device_key`, and the option still turns it into a key.
* **`0015-MADR-canonical-sdk-api-and-module-layout.md`:** its amendment
  "what `auth` holds" gains the session kind and its `SessionKind` method,
  the two sentinels, `NoLoopback` on `OAuthFlowOptions` and
  `ChatGPTPlanOptions`, and `VendorCLIPath`; D9 ("no ambient state")
  governs D10; and a note that `llmprovider/x/signin` is the first package
  under R4's tree. *(2026-11-11: the list named `NoLoopback`, which `auth` no
  longer exports, and not `SessionKind`.)* *(Later the same day, by the
  owner's reversal: `auth` exports the two `NoLoopback` fields, and the list
  names them again.)*
* **`0008-MADR-repair-oauth-loopback-and-session-wiring.md`, F5** ("This is
  **inferred** from the official CLI source"): a note that D12's Grok probe
  will replace the inference, and that until then the method's fact is
  `browser_loopback`.
* **The standards guide's R2 table:** rows for `llmprovider/x/signin` and
  `internal/signinflow`, and `authcontract` added to `wizard`'s row. **R2's
  sentence** "Only `wizard` may leave the standard library, and then only for
  `golang.org/x/term`" (C52) gains: "`wizard`, `llmprovider/x/signin` and
  `internal/signinflow` may also import the nested module `authcontract`,
  which is itself standard library only (0073-MADR D1)." The table rows alone
  would contradict the sentence.
* **AGENTS.md "Live tests":** `LLMPROVIDER_LIVE_DEVICE_LOGIN`'s scope, "the
  Grok and Kilo device logins" (C53), becomes "the Grok, Kilo and OpenAI
  device logins", for D12's OpenAI device test.

### Where the report and the audit differ from the code

* **S2 would deprecate `GrokFlowFromEnv`.** R30 names it as its example
  (C39). D7 keeps it. *(2026-11-11: "and 0076-MADR D4 gives it a use"
  removed; 0076 no longer adds the wizard option that would have.)*
* **S2 says `LoginDeviceOAuth` has no consumer.** It has no non-test caller,
  but two live tests call it (C40); D7 moves them.
* **S3 would add explicit `api_key` entries to the descriptors.** D2 makes
  them explicit in the contract instead, so no consumer that counts
  `AuthMethods` sees a change.
* **F27's five wizard writes.** Four write a session no other goroutine holds
  yet: one just returned by a device sign-in (`wizard/auth.go:358`), one just
  loaded (`:455`), one just returned by a sign-in (`:915`), and one just loaded
  for logout (`wizard/logout.go:44-45`). Only `:933` writes a session the
  listing has already used. The race the report suspects is therefore
  narrower than it states. It stays unverified until the PLAN's targeted
  `-race` probe (the report's U14), run on a scratch clone; it needs no
  sign-in and spends no quota.
* **The report's §5.4.2 makes `SignIn` an interface.** D3 makes it a concrete
  type, which can gain a method after promotion.
* **S3 gives the facts on `AuthMethod` JSON tags.** D1 adds none:
  `AuthMethod` has no JSON tags today (C1's struct), and the facts reach a
  wire as the contract's `Method`, whose names 0073-MADR D2 fixes, through
  `Methods` (D2).
* **S5 makes an undeliverable `OpenURL` a `failed` step.** D4 does not: the
  engine has no `OpenURL`, and a reader that cannot deliver an `open_url`
  step calls `Cancel`, which ends `failed` with `canceled`. In the wizard an
  `OpenURL` error is a warning and the sign-in goes on, since the URL is
  already on screen.
* **S12 asks for one revocation table.** This record does not make one. D1's
  `Revocable` states the fact as data, and the wizard's logout switch (C7)
  and `RevokeOAuthSession`'s own dispatch (C25) stay as they are; a later
  record may make the logout read the fact. *Deferred by the owner,
  2026-11-11, past the cut-over: no record is written for it now. It is
  listed as a candidate in the fleet-level record
  `0006-MADR-sequence-the-auth-contract-effort-across-four-repositories.md`,
  which is kept outside this repository, in the fleet's docs directory, to
  be raised after gobble-cli `v0.6.0` and magic-cli-remote's cut-over have
  shipped.*
* **S13 replaces the descriptor pins ("the dead pins") with the facts.**
  D12 keeps the pins, which compare whole structs (C36), and adds the facts
  to them, so the deprecated booleans stay pinned at their values.

### Revisions of 2026-11-11

A review against the code and the consumers' records as they stood on
2026-11-11. Each fix was checked against the code first.

* 2026-11-11, D3: a browser sign-in's `accepts` and `paste` step now follow
  the method's `Paste` fact as well as `NoPaste`. As written, the rule read
  `NoPaste` alone, so Grok's `browser_loopback` method (D1), which takes no
  paste, would have had one. gobble-cli's proposed 0051-MADR D4 and
  magic-cli-remote's proposed 0247-MADR assume no paste for it. 0247-MADR
  says "the SDK refuses a paste for Grok" and cites D4. D4 refuses only
  `NoLoopback` for Grok; with this fix D3 makes the claim true, since no
  step of Grok's sign-in asks for an answer.
* 2026-11-11, D4: three pastes in all, two re-asks, said in those words, so
  that a reader cannot count three re-asks. magic-cli-remote's 0247-MADR D2
  check 6 ("asked for again … at most three times") cites this rule; that
  wording is Magic's to align.
* 2026-11-11, D4: a pasted URL with this sign-in's `state` and an `error`
  ends the sign-in as today (C54), `denied` for `access_denied`, instead of
  being re-asked as a bad paste. magic-cli-remote's 0247-MADR D2 check 3
  ends such a paste the same way.
* 2026-11-11, D4: `NoLoopback` in `x/signin` stated for a method with no
  `Paste` fact and with `NoPaste`; and its callers named. No consumer record
  sets it: the PLAN's justification, "gobble's remote sign-ins", was not
  borne out by gobble-cli's records. *(Superseded by the owner's reversal,
  the last entry below: gobble-cli's 0051-MADR D3 is now the call site.)*
* 2026-11-11, D5: `SessionKind()` added as a *proposed revision awaiting the
  owner*, for gobble-cli's need of an old session's inferred kind; and the
  issuer guard of the refresh's token URL (C55) made explicit, so that the
  kind never sends a refresh token to an issuer that did not issue it.
* 2026-11-11, D2: "Eight descriptors list no methods" made exact: seven of
  the ten built-ins, and `chatcompat.Descriptor`.
* 2026-11-11, "Where the report and the audit differ": S3's JSON tags, S5's
  `failed` step, S12's revocation table and S13's pins, which this record
  does not follow, are now listed.
* 2026-11-11, citations of gobble-cli's 0051-MADR and 0052-MADR and
  magic-cli-remote's 0246-MADR and 0247-MADR say that each is proposed and
  untracked in its repository; the owner's answers they record, such as
  gobble's QE (a), stand.

The owner's rulings of 2026-11-11, written in after the review above.

* 2026-11-11, D5: `SessionKind()` is exported. It was "a proposed revision
  awaiting the owner", with an open question; the marker and the question
  are replaced by the ruling. D9, the Consequences, the note for 0015-MADR
  and the PLAN's R55 table, Phase 2 and its tests no longer call it
  proposed.
* 2026-11-11, D4: `NoLoopback` is experimental only.
  `signin.Options.NoLoopback` stays, for the U2 and U3 probes.
  `OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback` are not
  added until a consumer's record sets one, since R55 asks for a call site
  and no consumer has one. D9, the note for 0015-MADR and the PLAN's R55
  table, Phase 3, Phase 4 and Phase 6 lose the two fields. *(Superseded by
  the owner's reversal, the last entry below.)*
* 2026-11-11, D4, a question the ruling opens: with no exported field, the
  records' design gives `x/signin` no route into `auth`'s listener, since
  `internal/signinflow` imports `auth`. D4 now puts the options, a mark on
  the context proposed among them, and the PLAN stops before each
  `NoLoopback` step until the owner rules. *(Closed by the owner's
  reversal, the last entry below.)*
* 2026-11-11, D3, D4 and D5: the three rules corrected in the review above
  are confirmed by the owner, and each says so where it is stated: a browser
  sign-in's `accepts` and `paste` step follow the method's `Paste` fact as
  well as `NoPaste`, so Grok's `browser_loopback` gets none; three pastes in
  all, two re-asks, and a pasted URL carrying this sign-in's `state` and an
  `error` ends the sign-in, `denied` for `access_denied`; and the issuer
  guard stays on the refresh's token URL.
* 2026-11-11, "Where the report and the audit differ", S12: one revocation
  table is deferred past the cut-over, with no record now, and listed as a
  candidate in the fleet-level record named there.
* 2026-11-11, D7 and "Where the report and the audit differ", S2: both said
  0076-MADR D4 gives `GrokFlowFromEnv`'s values a way into the wizard. The
  owner dropped `wizard.Options.OAuthClients` from 0076 the same day, so
  both now rest on R30 alone. "Related" names 0076 for `auth.Identity`
  only.
* 2026-11-11, the PLAN's order: 0077's PLAN, which raises the Go floor to
  1.27.4, is its first step, before 0076's `v1.30.0`.

The owner's reversal, later on 2026-11-11.

* 2026-11-11, D4: the ruling "`NoLoopback` is experimental only:
  `signin.Options.NoLoopback` stays, and no exported field is added to
  `auth`" is reversed. `OAuthFlowOptions.NoLoopback` and
  `ChatGPTPlanOptions.NoLoopback` are part of the v1 API, and
  `signin.Options.NoLoopback` passes through to them, as D4 first designed;
  of the open question's three options, the owner chose (b). R55's
  call-site rule is met: gobble-cli sets the option for a sign-in started
  from a remote ACP client, whose person is not on the agent's host, so
  only a paste can complete it (gobble-cli's 0051-MADR D3), learning that
  the sign-in is remote from magic-cli-remote's `_meta.gobble.remote`
  (magic-cli-remote's 0246-MADR D8), which the SDK does not read. The
  Decision Outcome, D4's "No listener" bullet, D9, the Consequences and the
  note for 0015-MADR name the two fields again; the PLAN restores their R55
  row, Phase 3 step 2 and its three tests, Phase 4's
  `TestStart_NoLoopback_0074`, Phase 6's row for `auth`, and Phase 7's U2
  and U3, each with a dated note.

### Not verified

* Whether SA1019 reports the deprecated fields in descriptor literals (D7).
* Whether accounts.x.ai offers a paste (D1; D12's probe).
* The F27 race (above).

### Related

* PLAN: [0074-PLAN-one-sign-in-handle-and-method-facts.md](0074-PLAN-one-sign-in-handle-and-method-facts.md).
* The contract: [0073-MADR-publish-an-auth-contract-module.md](0073-MADR-publish-an-auth-contract-module.md).
* The plan's consent, host id and client:
  [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md).
* `auth.Identity`:
  [0076-MADR-auth-error-kinds-and-doc-drift.md](0076-MADR-auth-error-kinds-and-doc-drift.md).
* The Go floor this record's code is built on:
  [0077-MADR-go-1-27-4-for-go-2026-6859.md](0077-MADR-go-1-27-4-for-go-2026-6859.md).
* magic-cli-remote `docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md`,
  §5.4, §7.2 and §8.1.
* The consumers' records, each proposed and untracked in its repository on
  2026-11-11: gobble-cli
  `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  (`x/signin` for ACP `authenticate`, `Driver` for `--json`, the policy over
  D1's facts, `NoLoopback` for a remote client's sign-in) and
  `docs/decisions/0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`
  (the session's `kind` in gobble's store); magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  and `docs/decisions/0247-MADR-replay-a-pasted-sign-in-callback-to-the-agents-loopback.md`
  (Grok's browser method host-only; a paste re-asked as D4 asks).
