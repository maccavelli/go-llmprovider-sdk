---
status: accepted
date: 2026-10-18
decision-makers: repository owner
consulted: OpenAI's Sign in with ChatGPT pages for open-source agents (read 2026-10-14 and 2026-10-16) and its discovery document (read 2026-10-18); pi at 312184edb; the probe runs of 2026-10-02 and 2026-10-17; this module at v1.14.0 and aeedc28
informed: every caller of `auth`, of the `openai` provider and of `wizard`; gobble-cli and prepare-commit-msg first
---

<!-- markdownlint-disable MD013 -->

# Add OpenAI's Sign in with ChatGPT for Agents as an OpenAI Auth Method

## Context and Problem Statement

This module's only OpenAI subscription login borrows the Codex CLI's client
id (`auth.DefaultOpenAIClientID`, `llmprovider/auth/oauth_constants.go:9`),
a risk [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)
D7 records; the package doc says the ids are "borrowed, and a vendor may
restrict them at any time" (`llmprovider/auth/doc.go:8-11`).
OpenAI now publishes a flow for open-source agents: each app registers its
own client at first sign-in (`dynamic_agent_client`), and uses the token on
the public Responses API, billed to the user's ChatGPT plan.

[0017-MADR-together-provider-and-auth-extensions.md](0017-MADR-together-provider-and-auth-extensions.md)
D4 asked for a probe before any decision. Its amendment of 2026-10-17, from
0017-PLAN U5's run, decided to adopt the flow, with its design in a record of
its own: this one. The evidence is
[0017-REPORT-openai-sign-in-for-agents.md](../reports/0017-REPORT-openai-sign-in-for-agents.md)
(OpenAI's documents and terms; filed as
[issue 2](https://github.com/maccavelli/go-llmprovider-sdk/issues/2)) and
the two probe runs.

What has to be decided is where the login, the session and the request
shape go in `auth`, `providers/openai` and `wizard`, with the least new
surface and without changing any existing session's behaviour.

### What is measured

From 0017-MADR's amendments of 2026-10-03 and 2026-10-17:

| # | Fact | Source |
| :--- | :--- | :--- |
| M1 | Registration under an app's own name succeeds; the callback carries the issued client id (`oaia…`, 31 and 32 characters on two runs) | both runs |
| M2 | All six scopes asked for are granted, `chatgpt.tokens.use.direct` among them; the access token is a JWT that expires in 3600 s | both runs |
| M3 | A non-streamed request is refused, `Store must be set to false`, then `Stream must be set to true` | both runs |
| M4 | A streamed request with `store: false` is accepted: raw, and through `openai.New` with `WithStore(false)` and `Stream`, for text and a forced top-level tool call | 2026-10-17 |
| M5 | `max_output_tokens` is not refused | 2026-10-17 |
| M6 | `GET /v1/models` answers one top-level key, `models`: 9 models, 7 with `visibility: "list"` | 2026-10-17 |
| M7 | A form refresh with the issued client id, `resource` and no `scope` rotates the refresh token and keeps the scopes | 2026-10-17 |
| M8 | A form revocation at the discovery document's `revocation_endpoint` answers HTTP 200, empty; a refresh after it is refused `invalid_grant` | 2026-10-17 |
| M9 | `https://auth.openai.com/.well-known/openid-configuration`: issuer `https://auth.openai.com`; authorize `/api/accounts/authorize`; token `/api/accounts/oauth/token`; revocation `/api/accounts/oauth/revoke`; `jwks_uri` `/.well-known/jwks.json`; RS256; no `registration_endpoint`; token auth `none` allowed | read 2026-10-18 |

### What this module has (at `aeedc28`)

* **`auth.OAuthSession`** (`llmprovider/auth/tokenstore.go:21`) holds
  `Provider`, `Access`, `Refresh`, `Expiry`, `Issuer`, `ClientID`,
  `AccountID`, `FedRAMP`, `TokenURL`, `Store`, `HTTPClient` and `Logger`. It
  carries the refresh lock, the rotation and the save-retry work of
  0016-MADR, 0021-MADR and 0026-MADR.
* **`ChatGPT()`** is true for any OpenAI session whose issuer is
  `https://auth.openai.com` (`oauth_session.go:327-334`), and
  `chatgpt.IsSession` routes such a session to the Codex backend at
  `chatgpt.com` (`internal/chatgpt/chatgpt.go`). A session from the new
  flow has that issuer too (M9), so today it would be sent to the wrong
  backend.
* **The refresh** of an OpenAI-issuer session is a JSON body, with no
  `resource` (`oauth_session.go:602-633`); **the revocation** is a JSON body
  to `{issuer}/oauth/revoke` (`oauth_revoke.go:78-104`). The new flow needs
  forms, `resource`, and the discovered endpoints (M7, M8).
* **`TokenStore`** keeps one session per provider (`tokenstore.go:132`);
  `FileTokenStore`'s record has no field for `resource`, scopes, the ID
  token or a host id (`tokenstore.go:148`).
* **ID-token verification** already exists: every login checks the
  signature against the issuer's keys, the audience and the nonce
  (`oauth_idtoken.go`, 0016-MADR D7 and A3).
* **The `openai` provider** picks its backend by the credential
  (`openai.go:5-8`); a ChatGPT session already streams every request and
  reads the stream in `Generate` (`openai.go:376-396`). A key's listing
  reads the platform's `data` array and probes models with billed requests.
* **Errors:** a 429 is `ErrRateLimited`, retryable; `usage_limit_reached`
  and `insufficient_quota` are `ErrQuotaExhausted` (`api_error.go:360`).

## Decision Drivers

* **No behaviour change for existing sessions.** A Codex-id session, a key
  and a stored record keep working exactly as today (0015-MADR D1 parity;
  the G-wire goldens).
* **OpenAI's documents and terms.** The app's own name (§1), forms and
  `resource`, the issued client id saved per account, `store: false` and
  `stream: true`, the unsupported fields and tools.
* **Reuse the hardened session.** The refresh lock, rotation and
  save-retry are the most tested code in `auth`; a second session type
  would duplicate them.
* **Least new exported API** (`docs/guides/api-standards.md`), additive
  only, so the release is a minor version (R48).
* **What an SDK can and cannot do about the terms.** It can send the app's
  name, verify, store locally and revoke. It cannot know whether a request
  is the user's own activity; that is the consumer's.

## Considered Options

* **A. Extend `OAuthSession`, with a new login function.** The session
  gains the fields the flow needs and a kind that every branch reads;
  `auth.LoginChatGPTPlan` runs the flow.
* **B. A new session type**, `auth.PlanSession`, with its own refresh and
  store record.
* **C. Reuse `LoginBrowserOAuth`**, switching to the flow when
  `OAuthFlowOptions` carries an agent name.

## Decision Outcome

Chosen option: "A. Extend `OAuthSession`, with a new login function",
because it keeps one hardened refresh path, makes the kind explicit rather
than inferred from an option, and adds the fewest exported names.

### D1. The session: four fields, and an explicit kind

`auth.OAuthSession` gains:

* `Resource string`: the resource the tokens were granted for. Non-empty
  marks a plan session; the new login always sets it to
  `https://api.openai.com/v1`.
* ~~`Scopes []string`: the granted scopes.~~ `Scope string`: the granted
  scopes, space-separated as the token response sends them (A1).
* `IDToken string`: the last ID token, for a re-sign-in's `id_token_hint`.
  It is a secret: `String`, `GoString`, `MarshalJSON` and `LogValue` show
  it as they show the tokens (R33).
* `HostID string`: the `ext_agent_host_id` the client was registered with.

And one method, `PlanUsage() bool`: an OpenAI session with `Resource` set.
`ChatGPT()` becomes false for it, so `chatgpt.IsSession` no longer matches
it and no plan session reaches the Codex backend. Every other session's
`ChatGPT()` is unchanged.

### D2. Refresh and revocation follow the kind

For a plan session:

* **Refresh** posts a form to the session's `TokenURL` (the discovered
  token endpoint) with `grant_type`, `refresh_token`, the issued
  `client_id` and `resource`, and no `scope` (M7). The lock, the re-read
  after locking and the rotation are the existing ones.
* **Revocation** (`RevokeOAuthSession`) posts an RFC 7009 form to the
  discovered `revocation_endpoint` with the issued client id (M8). The
  endpoint is read from the issuer's discovery document at revocation, as
  Grok's is today (`grokRevokeRequest`), so the session keeps no field for
  it.

Every other session's refresh and revocation are unchanged.

### D3. The store: one OpenAI login per store, of either kind

`TokenStore` still keeps one session per provider. A plan sign-in saved
under `openai` replaces a Codex-id session there, and the reverse; a
person signs in with one of the two. `FileTokenStore`'s record gains
`resource`, ~~`scopes`~~ `scope` (A1), `id_token` and `host_id`, each `omitempty`, so a
record written by an older version reads as before.

* A record written by this version and read by an older one is read as a
  Codex-id session, whose refresh then fails; signing in again mends it.
  That downgrade cost is accepted and goes in the release notes.

### D4. The login: `auth.LoginChatGPTPlan`

```go
// ChatGPTPlanOptions configures LoginChatGPTPlan.
type ChatGPTPlanOptions struct {
    AgentName  string        // the app's own name, required (terms §1)
    HostID     string        // optional; a urn:uuid is made when empty
    Previous   *OAuthSession // a plan session to re-sign-in: its client id, host id and ID token
    HTTPClient *http.Client
    OpenURL    func(string) error
    InputCode  func(context.Context) (string, error)
}

func LoginChatGPTPlan(ctx context.Context, opts ChatGPTPlanOptions) (*OAuthSession, error)
```

* It reads the issuer's discovery document, and refuses one whose
  endpoints are not on the issuer's origin~~, as `LoginBrowserOAuth`
  does~~. *(A2: `LoginBrowserOAuth` does not; the check is new, and for
  plan sessions only.)*
* The first sign-in sends `client_id=dynamic_agent_client`,
  `agent_name_hint`, `ext_agent_host_id`, the six scopes, `resource`, PKCE
  `S256`, `state` and `nonce`, with an HTTP loopback redirect on
  `127.0.0.1` at `/auth/callback` (never `localhost`). With `Previous`, it
  sends that session's client id, `id_token_hint` and host id, and no
  `agent_name_hint`; a callback naming another client id is refused.
* It exchanges the code with the issued client id, verifies the ID token
  with the existing verifier (audience the issued client id, the nonce),
  and refuses a grant without `chatgpt.tokens.use.direct`.
* `AgentName` empty is an error matching `ErrInvalidRequest`. A paste of
  the callback URL (`InputCode`) is the fallback when the browser cannot
  reach the listener, as pi and `LoginBrowserOAuth` do. There is no device
  flow (OpenAI offers none).
* The host id, when not given, is a random version 4 UUID as
  `urn:uuid:…`, kept in the session (D1). OpenAI asks for one kept for the
  host's life; a caller that keeps one host-wide passes it.
* *(A3 records what the login settled that D4 left open: the callback's
  client id on a first sign-in, the pasted fallback, the host id's order,
  the grant with no scope, and discovery's fallback.)*

### D5. The `openai` provider: a third request shape

With a plan session, `openai.New`:

* sends to `https://api.openai.com/v1`, with `Authorization: Bearer` and
  none of the ChatGPT backend's headers;
* sends `store: false` and `stream: true` on every request, and reads the
  stream in `Generate`, as a ChatGPT session does (M3, M4);
* sends no `max_output_tokens`, though it was not refused (M5): OpenAI says
  to omit it, so `MaxOutputTokens` and `WithMaxTokens` have no effect there,
  as with a ChatGPT session;
* has Continuation `Unsupported`, and appends deferred tools rather than
  sending native `tool_search`, which OpenAI lists as unsupported
  (0040-MADR D6, D7 carry the warning);
* sends `prompt_cache_key` as a key does (not on OpenAI's unsupported
  list), and `include` for encrypted reasoning, as it does with
  `WithStore(false)` today (M4 sent it);
* lists from the `models` array, the entries with `visibility: "list"`,
  and never probes a model (each probe would spend the user's plan);
  `catalog.ListRaw` returns that listing as sent.

A key and a ChatGPT session send exactly what they send today; no G-wire
golden changes, and the plan shape gets its own goldens.

### D6. Errors

* `subscription_sharing_usage_limit_exceeded` joins `openAIQuotaTypes`:
  `ErrQuotaExhausted`, terminal. It is the plan's usage limit, which a
  retry within minutes does not pass, and it can arrive as
  `response.failed` mid-stream.
* `subscription_sharing_user_not_eligible` and
  `subscription_sharing_route_not_supported` are 403s, already
  `ErrNotPermitted`; `subscription_sharing_unsupported_capability` is a
  400, already `ErrInvalidRequest`. A `{"detail": …}` admission body is
  classified by its status, as today.
* `openai.ChatGPTUsageURL`, `https://chatgpt.com/settings/usage`, is the
  page a consumer's "Manage usage" opens. pi links the same page; OpenAI's
  UI guidance names the action, not the URL, so the constant cites pi.

### D7. The wizard

* The OpenAI descriptor gains `AuthChatGPTPlan` (`"chatgpt_plan"`), which
  needs a store, as the browser login does.
* `wizard.Options` gains `AgentName`. Without it the method is not
  offered: the SDK's own name is not the consumer's app (terms §1).
* Before the first plan sign-in, the wizard shows OpenAI's one-time notice,
  "You're using your ChatGPT plan", and the usage page.
* The Codex-id browser and device logins stay, unchanged. Whether they are
  ever retired is a later record's question, once plan sessions are in
  use.

### D8. What stays the consumer's

The package doc of `auth`, and the README's login section, say plainly
that the SDK cannot meet these for the consumer: requests from the user's own
runtime, consent before background use, "connected application only", no
pooling, and no charge (terms §2 and §4). No option claims to.

### D9. The probe

`TestLive_OpenAISigninProbe` is replaced by a live test of the shipped
login and provider, switched on by `LLMPROVIDER_LIVE_CHATGPT_PLAN`, which
signs in, generates, streams, refreshes, lists and revokes. The probe file
is removed in the same phase.

### Consequences

* Good, because consumers get an OpenAI subscription login under their own
  name, which removes 0016-MADR's borrowed-client-id risk for those who
  use it.
* Good, because the refresh lock, rotation and save-retry are reused, not
  copied.
* Good, because every existing session, key and record behaves as before;
  the G-wire goldens are unchanged.
* Neutral, because one store holds one OpenAI login: switching kind means
  signing in again.
* Bad, because a record written by this version breaks under an older one
  (D3).
* Bad, because OpenAI's pages are undated and titled "Preview
  limitations": the request rules may change, and the live test is the
  only early warning.
* Bad, because `OAuthSession` grows four fields and one more branch in
  refresh, revocation and `ChatGPT()`.

### Confirmation

* Fake-issuer tests for every branch of D1–D4 (with A1): the fields, `PlanUsage`,
  `ChatGPT()` false, the refresh form, the revocation form, the store
  round trip and an older record, the first sign-in and the re-sign-in, a
  callback with another client id, a grant without the direct scope, and
  a bad ID token. Each is seen to fail on a planted break first.
* Provider tests and new G-wire goldens for D5; the existing goldens
  unchanged; D6's codes classified.
* Wizard tests for D7: offered only with `AgentName`, the notice, the store.
* `make api-check`: additions only. `tools/apidiff.py` lists exactly the
  names above.
* The live test of D9 passes with the owner signing in.

## Pros and Cons of the Options

### A. Extend `OAuthSession`, with a new login function

* Good, because one refresh path serves both kinds, with its tests.
* Good, because the kind is a field, read in three places, not inferred.
* Bad, because every branch on the issuer has to learn the kind; a missed
  one sends a plan session to Codex's endpoints. The tests of D1–D2 cover
  each branch.

### B. A new session type

* Good, because the existing type is untouched.
* Bad, because the refresh lock, rotation and save-retry, and
  `TokenStore`, which returns `*OAuthSession`, would be duplicated or
  generalised.
* Bad, because every provider constructor that type-switches on
  `*auth.OAuthSession` (eight of them, `kilo.go:182` and `grok.go:151`
  among them) would learn a second type.

### C. Reuse `LoginBrowserOAuth`

* Good, because no new function.
* Bad, because one option silently changes the flow, the endpoints, the
  client id and the backend; an omitted name falls back to Codex's id,
  the very thing this removes.

## More Information

* The evidence: 0017-MADR's amendments of 2026-10-03, 2026-10-16 and
  2026-10-17; 0017-PLAN U4 and U5; 0017-REPORT-openai-sign-in-for-agents.md.
* The implementation: [0043-PLAN-openai-chatgpt-plan-sign-in.md](0043-PLAN-openai-chatgpt-plan-sign-in.md).
* Revisit when OpenAI changes the request rules, adds a device flow, or
  dates its pages; and when no consumer uses a Codex-id login.

## Amendments

### A1. The scopes are one string (2026-10-18)

* **Found** by 0043-PLAN Phase 0, step 2, before any code. D1's
  `Scopes []string` makes `OAuthSession` incomparable, and `make
  api-check` reports that as an incompatible change against `v1.14.0`
  (R48). 0021-PLAN's deviation of 2026-10-04 met the same failure with a
  func field.
* **Decided** by the owner: `Scope string`, the granted scopes
  space-separated, as OAuth's token response sends `scope`. A caller
  splits it with `strings.Fields`. The struct stays comparable.
* **Rejected:** an unexported slice with an accessor, which a caller's own
  `TokenStore` could not set; and accepting the break, which needs a major
  version.
* The store record's field is `scope`, to match.

### A2. The origin check is new, and for plan sessions only (2026-10-18)

* **Found** by 0043-PLAN Phase 1, reading `auth` before the code. D4 says
  the plan login refuses discovery endpoints off the issuer's origin "as
  `LoginBrowserOAuth` does". It does not: `requireEndpoints`
  (`llmprovider/auth/oauth_loopback.go:257`) only requires each endpoint
  to be present, and nothing in `auth` compares origins.
* **Decided** by the owner: the plan login and the plan revocation refuse
  an authorize, token, revocation or JWKS endpoint that is not on the
  issuer's origin (scheme and host), so a code or a refresh token is never
  posted to another host. The existing logins and revocations are
  unchanged. *(A3: a refresh is not checked; it trusts the session's
  `TokenURL`.)*
* **Rejected:** presence only, as today; and the check for every login,
  which changes existing sessions and needs its own record.

### A3. What the login settled that D4 left open (2026-10-18)

* **Recorded** at the owner's request after Phase 2 (`180f01e`), to make
  this record describe the code. Nothing in the code changes. Each point is
  tested in `llmprovider/auth/plan_login_0043_test.go` or
  `plan_session_0043_test.go` unless it says otherwise.
* **The callback's client id on a first sign-in.** The callback must name
  an issued client: one with no `client_id`, or one naming
  `dynamic_agent_client`, is refused before the code is exchanged. A
  re-sign-in's callback may leave `client_id` out, and the session's client
  is kept. D4 named only the re-sign-in's mismatch.
* **The pasted fallback** reads the pasted URL's `client_id` too. A bare
  code, which the other logins accept, carries none. So a bare code
  completes a re-sign-in only, and a first sign-in needs the whole URL. This
  follows from two tested pieces (a bare code parses with no client; a
  first sign-in with no client is refused); no test pastes a bare code into
  a first sign-in.
* **`id_token_hint`** is sent only when `Previous` holds an ID token.
  Untested on its own.
* **The host id** is `HostID`, trimmed; else `Previous`'s; else a new
  `urn:uuid`. D4's struct comment names only the last.
* **`AgentName` is required on a re-sign-in too,** though only a first
  sign-in sends it, so that one call serves both. A `Previous` that is not
  a plan session (`PlanUsage`) with a client id is an error matching
  `ErrInvalidRequest`, before any request.
* **The grant.** A token response with no `scope` is refused, like one
  without `chatgpt.tokens.use.direct`: it matches `ErrNotPermitted`. One
  with no ID token is refused by the existing verifier, since `openid` was
  asked for (`TestVerifiedSession_RequiresIDToken`).
* **The signature** is checked by the shared verifier, whose tests cover a
  bad one (`TestVerifyIDToken_Rejects`). The plan tests cover the audience
  and the nonce only; 0043-PLAN Phase 2 step 3 planned a signature case
  there too.
* **Discovery's fallback.** For OpenAI's own issuer only, M9's endpoints
  fill in when discovery fails or leaves one out, as the built-in values do
  for the other logins (0016-MADR D7). Any other issuer must publish them.
  D2 and D4 say only that the endpoints come from the discovery document.
  Phase 1 built this (`planOAuthEndpoints`), and the PLAN recorded it
  there, but not as a deviation.
* **What A2 checks.** The origin check runs at sign-in and at revocation. A
  refresh posts to the session's stored `TokenURL` unchecked. The login set
  that URL from a checked discovery. But a session that a caller builds, or
  loads from its own store, is trusted as given, like every other session's
  `TokenURL` (`refreshTokenURL`).
* **The listener** is the OpenAI loopback one: `127.0.0.1`, port 1455, else
  1457, path `/auth/callback`, with ten minutes to finish. When both ports
  are taken, the login fails, as `LoginBrowserOAuth` does for OpenAI.
  Untested for the plan login.
* **The account.** As with every OpenAI login, `AccountID` and `FedRAMP`
  are read from the ID token's claims, so `Account()` answers for a plan
  session. Phase 3 must not send them as the Codex backend's headers (D5).
  Untested until Phase 3.
