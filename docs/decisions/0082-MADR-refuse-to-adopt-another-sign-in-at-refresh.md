---
status: proposed
date: 2026-11-17
decision-makers: repository owner
consulted: "the owner's request of 2026-11-17 (quoted below); the owner's answers of 2026-11-17 to the draft's eight questions, relayed by the coordinator (under Decided); this repository at b109cea (v1.31.0 and five records commits); gobble-cli at cecc892 on its branch p3-acpext-v0.5.0, and the records staged over it on 2026-11-17 and not yet committed (named below); the upstream checkouts codex at 57e64682f8, opencode at 4ce137b39e and kilocode at 58072bd696, read-only; scratch runs of 2026-11-17 against git-archive copies of b109cea, with in-process stores and issuers only; the third independent review of gobble's 0052-PLAN Phase 4 (2026-11-17), relayed by the coordinator as a reviewer's report and checked here against the code and run S6"
informed: gobble-cli, whose held ChatGPT plan session this decision closes; every caller that gives an auth.OAuthSession a TokenStore another process or application also writes
---

<!-- markdownlint-disable MD013 MD024 -->

# Refuse to Adopt a Stored Session That Is Not the Same Sign-In, So a Held Session Never Becomes Another Login at Its Refresh

## Context and Problem Statement

An `auth.OAuthSession` that is due for a refresh first re-reads its token
store, under the store's refresh lock, and adopts what it finds there
whenever the stored refresh token differs from its own (E1 to E9). The rule
exists for one case: two processes share one login, one of them rotates the
refresh token, and the other must pick up the rotated pair instead of
spending a refresh token the issuer has already replaced (E65; E66, E67).
The rule tests only that the refresh token *differs*. It does not test that
the stored session is the same login. `adopt` then copies every field,
the provider, the kind, the issuer, the client and the account included
(E10 to E16). So a session a program holds becomes, at its next refresh,
whatever another program has written into the store: another application's
login, another kind of session, another account, or, through
`FileTokenStore`, another provider's session (runs S1 to S3).

A provider built over the session does not notice. The OpenAI provider
decides at `New` whether the session is a ChatGPT login or a ChatGPT plan
session, and keeps that route for its life (E38 to E40; the API standards'
R20, E61). After the adoption it sends the new login's token down the old
route: the platform's plan endpoint for a session built as a plan session
(S4), the Codex backend for one built as a ChatGPT login (S5).

### The owner's answers of 2026-11-17

The owner answered the draft's questions on 2026-11-17, each as the draft
recommended, relayed by the coordinator; the labels and what was asked are
under "Decided". Q5 was not asked: it falls away by Q4's answer, as the
coordinator applied it. The record stays `proposed`: the owner accepts the
record itself once it is placed and reviewed.

### The request

The owner's request of 2026-11-17, verbatim: "analyze, research, assess and
evaluate these: - Whether to draft SDK records for the three other items on
its next release: interfaces in place of the concrete session-type checks, a
wizard option to skip the callback listener, and the typed "tool schema
refused" error. and draft the detailed, facts-grounded and complete madr docs
for review." This record is the first item. The other two items have records
of their own.

### Where the item comes from: gobble-cli

gobble-cli signs users in to OpenAI with a ChatGPT plan session through this
module, and refuses any other application's ChatGPT login it finds in its own
`auth.json` (a "foreign session": the same provider, another kind). Its
store reaches this module through an adapter that gives every entry the
provider it was asked for (G1 to G3). An independent reviewer of gobble's
0052-PLAN Phase 4 showed, with a reproduction and no network, that gobble
refuses a foreign session when it loads one, but that a session a run
already holds adopts a foreign entry written into `auth.json` meanwhile, at
its next refresh, and sends its token (the reproduction is
`TestReviewCachedSessionAdoptsAForeignEntry`, kept outside both trees because
it fails by design today).

The owner's rulings for gobble, as gobble records them (gobble-cli's
`docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`,
"Second review and fixes", lines 1532 to 1538 of the working tree, staged
over `cecc892` and not yet committed), each "the owner's ruling, relayed by
the coordinator":

1. **"Refuse at the point of use"**: a small gobble token source wraps the
   held session; on each token request it checks the session's current kind
   under the session's lock and, if gobble's policy no longer accepts it,
   returns gobble's resolver error and drops the session. Not chosen there:
   checking when the held session is handed out; accepting it and
   documenting it.
2. When the builder stopped, because this module asserts its own concrete
   session type and a wrapper would hide a plan session from it: **"Ask the
   SDK for interfaces, then build as ruled"**. Not chosen there: refusing in
   gobble's own store adapter now (the coordinator's recommendation);
   refusing in the store now and asking the SDK too.

Until a release of this module meets the request, gobble documents the limit
(its `docs/architecture.md` line 111, `docs/guides/credentials.md` lines 309
to 311, and its
`0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D2's
note at line 334, all in the same staged working tree), and names the
point-of-use refusal a step of the gobble phase that bumps its pin
(0052-PLAN P4 step 1d, line 453).

The builder's stop was put to the owner with the store guard (option D
below) as the coordinator's recommended option; the owner chose the SDK
change instead, knowingly preferring a fix in this module to a guard in one
caller (the coordinator's account of 2026-11-17).

A third independent review of gobble's Phase 4 (2026-11-17, a reviewer's
report relayed by the coordinator, checked here) found the premise behind
the request accurate, as the table below shows, and prototyped option D on a
scratch copy of gobble: its store adapter's `Load` returns gobble's
foreign-login error for an entry its policy does not accept, and its `Save`
refuses to write over such an entry. With it, a held plan session keeps its
own kind and its refreshed token and never sends the foreign one, and the
foreign entry is left as it was; gobble's affected packages' tests pass; the
probe fails without the guard. The reviewer also recorded what it does not
do: the run goes on with its own plan session rather than refusing, and the
rotated tokens are not saved, so another process, or the next start, finds
the foreign entry and not the rotated pair. Run S6 here shows the same
behaviour from this module's side.

"Interfaces in place of the concrete session-type checks" is therefore one
proposed means. The problem behind it is that a held session silently
becomes a different sign-in by adopting what its store holds. This record
decides the means.

### How a held session reloads from its store (read at `b109cea`)

* `Token` returns the held token while it is current and no save is
  pending (E17). A token that is due, or one a provider expired after a 401
  (`Invalidate`, `InvalidateToken`, E18; `wire.Reauth`, E49), starts a
  refresh. A session with no refresh token never refreshes: it fails with
  `ErrAuthFailure` before the store is read (E26). Kilo's device session is
  one (E56), and so is the access-only ChatGPT session a pasted token makes
  (E68).
* `reloadOrRefresh` takes the store's refresh lock when the store offers
  one (E2, E29), then calls `loadRotated` (E3). `loadRotated` returns the
  stored session when the store holds a refresh token that is neither the
  held one nor the one the store is known to hold while a rotation is
  unsaved (E8), and nothing when the read fails, finds nothing, or finds no
  refresh token (E7). "The stored refresh token differs" means exactly that
  comparison of two strings and nothing else.
* A returned session is used as it is when its access token is current
  (E4). Otherwise its own fields become the refresh's state (E5), so its
  refresh token is sent, as its client, to its token URL (E23). After a
  permanent refresh failure the store is read once more, and a session found
  there is adopted the same way (E6; 0016-MADR amendment A1, E66).
* `complete` adopts the result into the held session (E10) and records the
  stored refresh token as the store's (E11). `adopt` copies `Provider`,
  `Access`, `Refresh`, `Expiry`, `Issuer`, `ClientID`, `AccountID`,
  `FedRAMP`, `TokenURL`, `Resource`, `Scope`, `IDToken`, `HostID`, `Kind`,
  `Consents`, `Store`, `HTTPClient` and the issue time (E12 to E16; lines
  700 to 719 of the same file). Of what identifies the session, only its
  in-memory application identity is left as it was (0079-MADR D2); its
  logger and clock are not touched either.
* `FileTokenStore.Load(provider)` returns the record's own `Provider`, not
  the one asked for (E31), so a record of one provider in another's file is
  adopted with its provider (S3). gobble's adapter sets the asked provider
  (G3), so gobble is not exposed to that part.

The records that decided the rule: MADR 0012 §5.2, "If the stored refresh
token differs from the in-memory one, it adopts the stored session and skips
the refresh" (E65); 0016-MADR D4 and its amendments A1 (re-read after a
rejected refresh) and A2 (the cross-process refresh lock, because both
vendors revoke the whole refresh-token family when one refresh token is used
twice, E67); 0020-MADR F1 and 0026-MADR F1 (a token the session itself spent
is never adopted back, E8's second half). None of them considered a stored
session that is a different sign-in.

### What changes, and what does not, across a legitimate rotation

A legitimate rotation is the same login refreshed by another process using
this module. That process's refresh builds the next session from its own
state (lines 790 to 810 of `oauth_session.go`):

| Field | Set at sign-in from | Kept by a refresh | Differs between the held session and a sibling's rotation? |
| :--- | :--- | :--- | :--- |
| `Provider` | the sign-in's provider (E53's neighbour, line 850) | yes | no |
| `Kind` | the sign-in (E53, E55, E56) | yes, the declared kind (E21) | no; an older record with no kind infers the same one (E24, E25) |
| `Issuer` | the sign-in's issuer | yes | no |
| `ClientID` | the sign-in's client; a plan sign-in's is issued, or reused from a `ClientStore` (E54) | yes (E19) | no |
| `AccountID` | the ID token's `chatgpt_account_id` (E52) | yes (E20) | no |
| `TokenURL`, `Resource`, `HostID` | the sign-in | yes | no |
| `Scope`, `IDToken` | a plan sign-in | replaced when the refresh returns them (E22) | may |
| `Consents` | the user's answers | the store's latest (0075-MADR D3) | may |
| `Access`, `Refresh`, `Expiry` | each exchange | rotated | yes, by design |

So provider, kind, issuer, client and account identify the sign-in and stay
constant across a legitimate rotation; tokens, expiry, scope, ID token and
consents do not. The module holds no subject claim: it keeps the ID token
only for a plan session (lines 39 to 41 of `tokenstore.go`), and reads the
account id from a ChatGPT ID token's claims (E52). Whether a plan session's
ID token carries `chatgpt_account_id` has not been seen: a plan session's
`AccountID` may be empty (see "Not verified").

By provider:

* **OpenAI** has two refreshable kinds, `chatgpt` and `chatgpt_plan`, both
  offered by its descriptor (E57, E58), so both sign-ins write the same
  provider's entry. Adoption across the two kinds is possible for any caller
  that offers both, and for any store another application also writes.
* **Grok** has one kind. A foreign Grok session would differ by client or
  issuer, not by kind.
* **Kilo**'s device session has no refresh token (E56) and never reaches the
  store's re-read (E26). It is not affected.
* **The vendor CLI sessions** read the vendor's own file on every token
  request by design (E59): following whatever that file holds is their
  purpose. They are not `OAuthSession`s and are outside this record.

### What a provider needs from its token source beyond a token

Every place, at `b109cea`, where the module asserts the concrete session
type or another interface on a token source:

| Site | Assertion | What it needs | Read when |
| :--- | :--- | :--- | :--- |
| `internal/chatgpt` `IsSession` | `*auth.OAuthSession`, then `ChatGPT()` (E33) | is this a ChatGPT login | at `openai.New` (E38), and per raw listing (E48) |
| `internal/chatgpt` `IsPlanSession` | `*auth.OAuthSession`, then `PlanUsage()` (E34) | is this a plan session: the platform route, the plan listing, the no-system-message rule (E69) | the same (E39, E48) |
| `openai` `checkBackground` | `*auth.OAuthSession`, then `StoredConsent` (E42) | the stored answer to the background-use consent; a source that is not the concrete type passes the gate | each background request, only if built as a plan session (E41) |
| `openai.New`, `grok.New` | `*auth.OAuthSession`, then `Owner()` (E36, E37, E45) | R16: refuse another provider's session (E60) | at `New` |
| `kilo.New`; seven other providers | `*auth.OAuthSession`: Kilo refuses another provider's (E46); the seven refuse any (E47) | R16 | at `New` |
| `internal/chatgpt` `SetHeadersAs` | interface `accountSource` (E35) | the ChatGPT account id and FedRAMP flag | each ChatGPT request (E43) |
| `openai`, `grok`, `catalog` | interface `identityUser` and three like it (E44) | the application's identity, HTTP client and logger for the session's own refreshes | at `New`, and before a listing's token read |
| `wire.Reauth`, `catalog` | `llmprovider.InvalidatingSource`, `TokenInvalidator` (E49, E50) | expire a refused token and retry once | on a 401 |

The first five rows are concrete-type checks: those are what a wrapper
defeats. gobble's record names three of them (`chatgpt.go:50` and `:58`,
`openai.go:380`); the R16 checks at `New` are the rest, and a wrapper
defeats those too: wrapped, a plan session is no longer an
`*auth.OAuthSession`, so `openai.New` builds a provider that treats it as an
API key on the platform, and every provider that refuses an OAuth session
under R16 accepts it (inferred from E36 to E47: the `switch` has no case for
it). The other rows already take interfaces; a wrapper must implement each of
them, or the behaviour it carries stops without an error.

### The reproduction, against this module alone (ran)

Scratch tests in a `git archive` copy of `b109cea`, with an in-memory store
and in-process HTTP servers; nothing in either repository's tree changed. The
runs are listed under "Runs" below.

* **S1.** A held plan session; the store's entry is replaced by another
  application's ChatGPT login whose access token is current; the held
  session is invalidated, as a provider does after a 401. `Token` returns the
  other login's access token with no request on the network, and the held
  session is now kind `chatgpt`, `PlanUsage` false, `ChatGPT` true, with the
  other application's client and account, and no resource or host id.
* **S2.** The same with the other login's access token expired: the held
  session sends the other login's refresh token, as the other client, to the
  token URL the record names, and saves the rotated pair into its own store.
* **S3.** `FileTokenStore`: a Grok record written into `openai.json` is
  adopted by a held OpenAI session, whose `Owner()` becomes `grok`.
* **S4.** `openai.New` over a held plan session, listing models against an
  in-process server that refuses the plan token: the retry carries the other
  login's token to the plan route's `/models`, without the ChatGPT account
  header, and the listing succeeds with it.
* **S5.** The reverse: a provider built over a ChatGPT login whose store
  then holds a plan session sends the plan token to the Codex backend's
  `/models?client_version=…`.
* **S6.** Option D, a store that refuses to load another sign-in: the held
  session refreshes its own refresh token and keeps its kind. With a store
  that accepts the save, the other entry is overwritten; with one that
  refuses it, the other entry stays and the held session keeps working.
* **S7.** The baseline: the four packages below pass at `b109cea` with S1
  to S6 added.
* **S8.** A throwaway guard, option B with "same" as provider, kind, issuer,
  client and account when both are known, in a second scratch copy: S1 to S4
  now fail with `ErrAuthFailure` and no other-login token sent, and every
  existing test of `llmprovider/auth`, `providers/openai`, `x/signin` and
  `wizard` passes, the sibling-rotation tests among them.

### Who is affected

* **gobble-cli**, as described above: a running gobble whose `auth.json`
  another program rewrites. Its load-time check is unaffected and correct.
* **Any caller whose `TokenStore` another process or application writes**:
  two applications pointed at one `FileTokenStore` directory, a user who
  copies an entry between applications, or a single application that offers
  both OpenAI kinds and is signed in again by the other method in a second
  process. In the last case the adoption is the user's own intent, and still
  wrong: the built provider keeps the old route (S4, S5).
* **The other application**, when its entry is a copy: S2 spends and rotates
  its refresh token. When it next refreshes with its own copy it presents a
  spent token, and the issuer revokes the family (inferred from 0016-MADR A2's
  facts, E67; not measured).
* **The wizard and `x/signin`** hold sessions only while a sign-in runs, and
  save through the caller's store; they gain nothing and lose nothing beyond
  what the session does.

### Prior art (read; upstream checkouts, never changed)

* **Codex** reloads `auth.json` before a refresh and on a 401 "only if the
  account id matches the one the current process is running as" (X2, X3).
  On a mismatch it stops with a permanent error: "Your access token could
  not be refreshed because you have since logged out or signed in to another
  account. Please sign in again." (X1, X4). Its own persistence compares the
  login owner, the account and the ChatGPT user (X5).
* **opencode**'s ChatGPT plugin takes a stored credential only when it is an
  OAuth credential of its own sign-in method (O1): another method's
  credential is not used.
* **kilocode**'s Codex refresh takes any stored OAuth entry whose refresh
  token differs (K1, K2), as this module does today.

### The security weight

* **What it takes.** A write to the user's own credential file, or token
  store, while the program runs: a process of the same user, or one with
  more privilege, or the user by accident.
* **What follows.** The program's requests, the user's prompts and tool
  output among them, go to the provider under the other login, to the route
  the provider was built for (S4, S5); the base URL never comes from the
  record (E40), so the requests that carry the user's content do not leave
  the provider's hosts. When the other
  login's access token is due, its refresh token goes to the token URL its
  record names (S2, E23), which is the writer's choice. The other
  application's refresh token is spent (above). The background-use consent
  gate is not bypassed for a provider built as a plan session: it reads the
  answer the store holds, which another application's entry does not carry
  (E41, E51); a provider built as a ChatGPT login has no such gate (E41), so
  a plan token it adopts is sent without it, to the Codex backend, where it
  is presumably refused (inferred; not verified).
* **What does not follow.** No token of the program's own reaches the
  writer: a writer able to change the file could already read it. No
  request carrying the user's content is redirected to another host; only
  the other login's own refresh goes where its record says.
* **The limit of any fix that reads the store.** Every field the module
  could compare is in the record the writer controls. A deliberate writer
  can label a foreign token with this application's provider, kind, client
  and account, and no check of the record can tell. gobble's load-time
  check, the wrapper the owner ruled and option B below share this limit.
  What they stop is the honest record of another sign-in, written by
  another application or by mistake, which is the case the reviewer
  reproduced.

The weight is therefore correctness first, with a modest security margin:
low likelihood, a narrow precondition, and real harm to the user's
expectation and to the other application's login when it happens.

### Evidence index

<!-- claims: self=b109cea gobble-cli=cecc892 codex=57e64682f8 opencode=4ce137b39e kilocode=58072bd696 -->

All rows were read at the revisions the marker names (read).

| Id | Claim | Location | Text |
| :--- | :--- | :--- | :--- |
| E1 | `reloadOrRefresh` adopts a session another process rotated | `llmprovider/auth/oauth_session.go:957` | `// reloadOrRefresh adopts a session another process rotated, else refreshes` |
| E2 | It takes the store's refresh lock first | `llmprovider/auth/oauth_session.go:964` | `if locker, ok := state.store.(RefreshLocker); ok {` |
| E3 | and reads the store | `llmprovider/auth/oauth_session.go:971` | `stored, kept, keep := loadRotated(ctx, state)` |
| E4 | A stored session with a current token is used as it is | `llmprovider/auth/oauth_session.go:974` | `return refreshed{next: stored, token: token}, nil` |
| E5 | otherwise its fields become the refresh's state | `llmprovider/auth/oauth_session.go:976` | `state = stored.refreshState()` |
| E6 | After a permanent failure the store is read again | `llmprovider/auth/oauth_session.go:982` | `if stored, _, _ := loadRotated(ctx, state); stored != nil {` |
| E7 | A failed or empty read adopts nothing | `llmprovider/auth/oauth_session.go:1012` | `if err != nil \|\| stored == nil \|\| stored.Refresh == "" {` |
| E8 | The only test is that the refresh token differs | `llmprovider/auth/oauth_session.go:1015` | `if stored.Refresh == state.refresh \|\| (state.stored != "" && stored.Refresh == state.stored) {` |
| E9 | The stored session is returned for adoption | `llmprovider/auth/oauth_session.go:1018` | `stored.Store, stored.HTTPClient, stored.now = state.store, state.httpClient, state.now` |
| E10 | `complete` adopts it into the held session | `llmprovider/auth/oauth_session.go:273` | `s.adopt(out.next)` |
| E11 | and records its refresh token as the store's | `llmprovider/auth/oauth_session.go:279` | `s.storedRefresh = out.next.Refresh` |
| E12 | `adopt` copies the provider | `llmprovider/auth/oauth_session.go:701` | `s.Provider = next.Provider` |
| E13 | the client | `llmprovider/auth/oauth_session.go:706` | `s.ClientID = next.ClientID` |
| E14 | the account | `llmprovider/auth/oauth_session.go:707` | `s.AccountID = next.AccountID` |
| E15 | the resource | `llmprovider/auth/oauth_session.go:710` | `s.Resource = next.Resource` |
| E16 | and the kind | `llmprovider/auth/oauth_session.go:714` | `s.Kind = next.Kind` |
| E17 | A current token returns with no refresh | `llmprovider/auth/oauth_session.go:164` | `if current && !retrySave {` |
| E18 | `Invalidate` expires the token | `llmprovider/auth/oauth_session.go:486` | `s.Expiry = s.clock().Add(-time.Second)` |
| E19 | A refresh keeps the client | `llmprovider/auth/oauth_session.go:796` | `ClientID:   state.clientID,` |
| E20 | and the account | `llmprovider/auth/oauth_session.go:797` | `AccountID:  state.accountID,` |
| E21 | and the declared kind | `llmprovider/auth/oauth_session.go:805` | `Kind:       state.declared,` |
| E22 | and replaces the scope when one is returned | `llmprovider/auth/oauth_session.go:801` | `Scope:      cmp.Or(payload.Scope, state.scope),` |
| E23 | A refresh goes to the token URL the state names | `llmprovider/auth/oauth_session.go:1023` | `if state.tokenURL != "" {` |
| E24 | A session's kind is its `Kind`, else inferred | `llmprovider/auth/oauth_session.go:411` | `return sessionKindOf(s.Kind, s.Provider, s.Issuer, s.Resource)` |
| E25 | The inference from provider, issuer and resource | `llmprovider/auth/oauth_session.go:427` | `func inferKind(provider llmprovider.ProviderID, issuer, resource string) SessionKind {` |
| E26 | No refresh token: an error before the store is read | `llmprovider/auth/oauth_session.go:177` | `if !current && s.Refresh == "" {` |
| E27 | A pending save is never written over a newer stored session | `llmprovider/auth/oauth_session.go:344` | `if err == nil && loaded != nil && loaded.Refresh != stored && loaded.Refresh != next.Refresh {` |
| E28 | `TokenStore` is the caller's interface | `llmprovider/auth/tokenstore.go:229` | `type TokenStore interface {` |
| E29 | `RefreshLocker` is optional on it | `llmprovider/auth/tokenstore.go:243` | `type RefreshLocker interface {` |
| E30 | `OAuthSession` is kept comparable | `llmprovider/auth/tokenstore.go:90` | `// interface, not a func, so that OAuthSession stays comparable.` |
| E31 | `FileTokenStore.Load` returns the record's provider | `llmprovider/auth/tokenstore_file.go:81` | `Provider:  rec.Provider,` |
| E32 | and its kind | `llmprovider/auth/tokenstore_file.go:94` | `Kind:      rec.Kind,` |
| E33 | `IsSession` asserts the concrete type | `llmprovider/internal/chatgpt/chatgpt.go:50` | `session, ok := src.(*auth.OAuthSession)` |
| E34 | `IsPlanSession` too | `llmprovider/internal/chatgpt/chatgpt.go:58` | `session, ok := src.(*auth.OAuthSession)` |
| E35 | The ChatGPT headers take an interface | `llmprovider/internal/chatgpt/chatgpt.go:87` | `account, ok := src.(accountSource)` |
| E36 | `openai.New` switches on the concrete type | `llmprovider/providers/openai/openai.go:228` | `case *auth.OAuthSession:` |
| E37 | and refuses another provider's session | `llmprovider/providers/openai/openai.go:231` | `if owner := s.Owner(); owner != llmprovider.ProviderOpenAI {` |
| E38 | The route is decided at `New`: ChatGPT | `llmprovider/providers/openai/openai.go:243` | `chatGPT:    isChatGPTSession(src),` |
| E39 | and plan | `llmprovider/providers/openai/openai.go:244` | `plan:       chatgpt.IsPlanSession(src),` |
| E40 | The base URL is set at `New` | `llmprovider/providers/openai/openai.go:273` | `p.baseURL = ChatGPTBaseURL` |
| E41 | The consent gate runs only for a provider built as a plan session | `llmprovider/providers/openai/openai.go:377` | `if !p.plan \|\| !llmprovider.IsBackground(ctx) {` |
| E42 | and asserts the concrete type | `llmprovider/providers/openai/openai.go:380` | `session, ok := p.src.(*auth.OAuthSession)` |
| E43 | ChatGPT requests ask the source for its account | `llmprovider/providers/openai/openai.go:579` | `setChatGPTHeaders(r, p.src, p.originator)` |
| E44 | The identity hand-off is an interface | `llmprovider/providers/openai/openai.go:691` | `type identityUser interface {` |
| E45 | `grok.New` switches on the concrete type | `llmprovider/providers/grok/grok.go:176` | `case *auth.OAuthSession:` |
| E46 | `kilo.New` too | `llmprovider/providers/kilo/kilo.go:208` | `case *auth.OAuthSession:` |
| E47 | Seven providers refuse an OAuth session by its type, Together's for one | `llmprovider/providers/together/together.go:145` | `case *auth.OAuthSession, *auth.VendorCLISession:` |
| E48 | The raw listing decides the route per call | `llmprovider/catalog/raw.go:117` | `plan := id == llmprovider.ProviderOpenAI && chatgpt.IsPlanSession(src)` |
| E49 | A 401 expires the refused token through an interface | `llmprovider/internal/wire/reauth.go:41` | `if invalidator, ok := src.(llmprovider.TokenInvalidator); ok {` |
| E50 | `InvalidatingSource` is exported | `llmprovider/command_token.go:34` | `type InvalidatingSource interface {` |
| E51 | `StoredConsent` reads the store's record | `llmprovider/auth/consent.go:138` | `stored, err := store.Load(ctx, provider)` |
| E52 | A sign-in reads the account from the ID token | `llmprovider/auth/oauth_loopback.go:857` | `AccountID:  chatGPTAccountID(payload.IDToken),` |
| E53 | and sets the kind | `llmprovider/auth/oauth_loopback.go:849` | `Kind:       signInKind(config.provider),` |
| E54 | A plan sign-in's client is issued, or reused | `llmprovider/auth/oauth_plan.go:149` | `if config.clientID, clientErr = planIssuedClient(knownPlanClient(opts), callback.clientID); clientErr != nil {` |
| E55 | and its kind is set | `llmprovider/auth/oauth_plan.go:164` | `session.Kind = SessionChatGPTPlan` |
| E56 | Kilo's device session has no refresh token | `llmprovider/auth/kilo_device.go:123` | `Kind: SessionKiloDevice, HTTPClient: config.callerClient, identity: config.identity}, nil` |
| E57 | OpenAI offers the ChatGPT browser sign-in | `llmprovider/providers/openai/descriptor.go:39` | `ID:             llmprovider.AuthBrowserOAuth,` |
| E58 | and the plan sign-in | `llmprovider/providers/openai/descriptor.go:90` | `ID:             llmprovider.AuthChatGPTPlan,` |
| E59 | A vendor CLI session re-reads its file on each token | `llmprovider/auth/vendor_session.go:31` | `Every Token call re-reads the CLI's auth` |
| E60 | R16 | `docs/guides/api-standards.md:169` | `R16. A provider applies the token as` |
| E61 | R20 | `docs/guides/api-standards.md:196` | `a provider is immutable and safe for concurrent use.` |
| E62 | R48 | `docs/guides/api-standards.md:317` | `R48. From` |
| E63 | R55 | `docs/guides/api-standards.md:367` | `R55. A new export is justified.` |
| E64 | R4 | `docs/guides/api-standards.md:77` | `R4. Experimental API lives under` |
| E65 | MADR 0012 §5.2 decided the adoption | `docs/decisions/0012-MADR-conform-providers-to-reference-clients.md:505` | `it adopts the stored session and skips the refresh.` |
| E66 | 0016-MADR A1, the re-read after a rejected refresh | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:558` | `### A1. Re-read the store after a rejected refresh (extends D4)` |
| E67 | 0016-MADR A2: reuse revokes the family | `docs/decisions/0016-MADR-provider-auth-and-support-baseline.md:582` | `records that both vendors revoke the **whole refresh-token family** when` |
| E68 | A pasted ChatGPT token is an access-only session | `wizard/auth.go:716` | `Kind:      auth.SessionChatGPT,` |
| E69 | A plan session takes no system message in `Input` | `llmprovider/providers/openai/openai.go:362` | `if p.plan && slices.ContainsFunc(req.Input, func(item llmprovider.Item) bool {` |
| E70 | A rotation whose save failed is saved again while the token is current | `llmprovider/auth/oauth_session.go:163` | `retrySave := current && unsaved && s.Store != nil && !now.Before(s.nextResave)` |
| G1 | gobble's store adapter loads for the SDK | `gobble-cli:llm/provider/login.go:149` | `func (a sdkStore) Load(` |
| G2 | as a session built from its entry | `gobble-cli:llm/provider/login.go:154` | `return l.session(string(p), a.s, nil, nil), nil` |
| G3 | with the provider it was asked for | `gobble-cli:llm/provider/login.go:122` | `Provider: llmprovider.ProviderID(id), Access: l.Access` |
| X1 | Codex's message on an account mismatch | `codex:codex-rs/login/src/auth/manager.rs:240` | `REFRESH_TOKEN_ACCOUNT_MISMATCH_MESSAGE: &str` |
| X2 | Codex reloads only for the same account | `codex:codex-rs/login/src/auth/manager.rs:2043` | `We only reload if the account id matches` |
| X3 | the comparison | `codex:codex-rs/login/src/auth/manager.rs:2996` | `if new_account_id.as_deref() != Some(expected_account_id) {` |
| X4 | a mismatch before a refresh is a permanent error | `codex:codex-rs/login/src/auth/manager.rs:3512` | `ReloadOutcome::Skipped => {` |
| X5 | Codex's same-login test includes the account | `codex:codex-rs/login/src/auth/account_grant.rs:134` | `&& a.account_id == b.account_id` |
| O1 | opencode uses only its own method's credential | `opencode:packages/core/src/plugin/provider/chatgpt.ts:223` | `credential.methodID === methodID` |
| K1 | kilocode takes any stored OAuth entry | `kilocode:packages/opencode/src/kilocode/provider/codex-refresh.ts:56` | `auth.type !== "oauth"` |
| K2 | whose refresh token differs | `kilocode:packages/opencode/src/kilocode/provider/codex-refresh.ts:88` | `const base = current && current.refresh !== token ? current : input.auth` |

### Runs (ran, 2026-11-17)

Each in a `git archive` copy of `b109cea` in a scratch directory, with
`go test -count=1`, offline, no `-race`; each test's own log line is quoted.
No live service and no real credential.

| Id | What was run | Result |
| :--- | :--- | :--- |
| S1 | `TestScratch_HeldPlanAdoptsForeignChatGPT` in `llmprovider/auth` | "token is the foreign access: true; kind "chatgpt"; PlanUsage false; ChatGPT true; client "app_other"; account "acct-other"; resource ""; host """; no request sent |
| S2 | `TestScratch_HeldPlanRefreshesForeignRefreshToken` in `llmprovider/auth` | one refresh request, JSON, `refresh_token` the other login's, `client_id` the other client's, at the record's token URL; "store now holds refresh "foreign-refresh-2" kind "chatgpt"" |
| S3 | `TestScratch_FileStoreCrossProvider` in `llmprovider/auth` | a Grok record in `openai.json` adopted: "token is grok's: true; Owner "grok"; kind "grok"" |
| S4 | `TestScratch_ProviderSendsAdoptedForeignToken` in `providers/openai` | requests `/models` with the plan token (401), then `/models` with the other login's token and no account header; "models [gpt-scratch], err <nil>"; held kind "chatgpt" |
| S5 | `TestScratch_ProviderKeepsChatGPTRouteAfterAdoptingPlan` in `providers/openai` | second request `/models?client_version=0.0.0` with the plan token; held kind "chatgpt_plan" |
| S6 | `TestScratch_OptionD_StoreRefuses` in `llmprovider/auth` | both cases: the held session's own refresh token refreshed as its own client, kind "chatgpt_plan", one save; save accepted: the store holds the held session; save refused: the store keeps the other entry |
| S7 | `go test` of `llmprovider/auth`, `providers/openai`, `x/signin`, `wizard` at `b109cea` with S1 to S6 added | exit 0 each |
| S8 | the same four packages with a throwaway option-B guard | `auth` exit 1: S1, S2, S3 fail ("oauth: the store holds another sign-in for openai (kind chatgpt), not this session's (kind chatgpt_plan); sign in again: llmprovider: authentication failed"); `providers/openai` exit 1: S4 fails, its one request the plan token's; `x/signin` and `wizard` exit 0; every other test of the four packages passes (no other package was run), among them `TestOAuthRefresh_ReloadsStoreFirst`, `TestOAuthRefresh_RejectedAdoptsSiblingRotation`, `TestOAuthRefresh_RejectedRefreshesWithSiblingToken`, `TestOAuthRefresh_UnsavedRotationNeverOverwritesNewer` and `TestFileTokenStore_RefreshLock_OnceAcrossSessions`, run again by name with `-v` |

S8's guard is a measurement, not the implementation: it is the shape D1
describes, and it shows that the existing rotation tests do not depend on
adopting another sign-in.

## Decision Drivers

* **A held session stays the sign-in it was.** What the session is decides
  where a built provider sends its token (E38 to E40) and R20 keeps that
  fixed; a session that changes underneath is wrong whoever wrote the store.
* **Every consumer is protected by default,** not only a caller that wraps
  or adapts its store.
* **Legitimate rotation between processes keeps working:** the reason the
  adoption exists (E65 to E67), and the tests that hold it.
* **Nothing is spent that is not this sign-in's:** another login's refresh
  token is never sent (S2).
* **The least API under the v1 promise** (R48, R55, E62, E63), and no change
  that makes `OAuthSession` incomparable (E30).
* **gobble's ruling is met:** "Refuse at the point of use", with no
  documented limit left.
* **Small and testable offline,** in one release shared with the other
  items.

## Considered Options

* **A. Interfaces in place of the concrete assertions** (the item as named).
  `IsSession`, `IsPlanSession`, `checkBackground` and the `New` owner checks
  assert method sets instead of `*auth.OAuthSession`: `ChatGPT() bool`,
  `PlanUsage() bool`, `StoredConsent(ctx, id) (bool, bool, error)` and
  `Owner() llmprovider.ProviderID`, either as unexported interface types
  (the methods are the contract, as `identityUser` is today, E44) or as
  exported ones. A caller then wraps the session in its own token source,
  forwards those methods and the existing optional ones (`Account`,
  `Invalidate`, `InvalidateToken`, `UseIdentity`, `UseHTTPClient`,
  `UseDefaultHTTPClient`, `UseLogger`), and refuses in `Token` when the
  session's kind is no longer one it accepts. The providers that refuse an
  OAuth session under R16 (E46, E47) need the same recognition, or a wrapped
  session passes them. The adoption itself is unchanged.
* **B. The session refuses to adopt a stored session that is not the same
  sign-in.** In `reloadOrRefresh`, at both reads (E3, E6), a stored session
  whose provider, kind, issuer or client differs from the held one's, or
  whose account differs when both name one, is not adopted, used or
  refreshed: the token request fails with an error matching
  `ErrAuthFailure`, and the held session keeps its own fields. A rotation of
  the same sign-in is adopted as today. No new export is needed.
* **C. A caller-supplied adoption check.** An optional interface on the
  `TokenStore`, as `RefreshLocker` is (E29), say
  `AcceptRotation(ctx, provider, held, stored *OAuthSession) error`,
  consulted before adoption; a refusal fails the token request. A function
  field on the session is not possible without making it incomparable
  (E30).
* **D. The caller's store refuses** (what gobble could do today, with no
  change here). Its `Load` returns an error, or nothing, for an entry its
  policy does not accept; by the code, `loadRotated` then adopts nothing
  (E7), the held session refreshes its own refresh token and saves (S6);
  a store that also refuses that save keeps the other entry, and the held
  session keeps working on the rotated pair in memory, logging the failed
  save and retrying it at most every 30 s while its token is current (E70;
  `complete`, lines 255 to 267, and `persistRotation`, which saves after a
  failed `Load`, E27's neighbour). The issuer has rotated the held refresh
  token, so when that process ends the only valid copy is gone and the next
  start finds the other entry (inferred from the code and S6; the third
  review's prototype observed the same). `StoredConsent` sees the same `Load`
  error and refuses a background request (E51).
* **E. Do nothing in this module; document** the adoption on `TokenStore`
  and `OAuthSession.Token`.

## Decision Outcome

Chosen option: **"B"**, by the owner's answer to Q1: the session refuses to
adopt a stored session that is not the same sign-in, where "the same" is
provider, kind, issuer and client, and the account when both sessions name
one (Q2); its token requests fail with the existing auth-failure error until
the store holds its sign-in again, or nothing (Q3, Q4); no interfaces and
nothing exported (Q1, Q4; Q5 falls away); every provider's refreshable
sessions covered (Q6); a Security changelog entry (Q7); the next release,
with 0080 and the other drafts, tagged on the owner's word (Q8). The
answers and their labels are under "Decided". Because:

* B closes the problem where it arises, before anything is sent or spent
  (S8: no other-login token sent; S2's refresh never happens), for every
  consumer, with no new API, and S8 shows no existing rotation test depends
  on what it removes.
* A, the means the owner named for gobble, protects only a caller that
  wraps, and only after the adoption: by the time the wrapper sees the new
  kind, the held session has already adopted the other entry and, if its
  token was due, spent and rotated the other application's refresh token
  (S2). It needs every concrete check and every owner check replaced, and a
  wrapper that forwards eleven methods, each a silent contract under the v1
  promise. It is the wrong layer for this problem; it may be right for
  another, which would then have a record of its own.
* C and D protect only a caller that implements them. D needs no change
  here and works for gobble (S6; the third review's prototype), but it
  changes what is persisted, leaves the run diverging from its file, and
  leaves every other consumer exposed; the owner has already declined it for
  gobble. E leaves the routing defect (S4, S5) for every consumer.

**This choice differs from the means gobble's earlier ruling named,** and
was put to the owner as differing. gobble's rulings were "Refuse at the
point of use" and "Ask the SDK for interfaces, then build as ruled". B meets
the first, at the token request, inside this module, for gobble and every
other caller, and makes the wrapper of the second unnecessary. It keeps the
owner's choice of layer, a change in this module over a guard in one caller,
and differs in the mechanism: the module refuses, instead of each caller.

### D1. The guard (Q1)

* `reloadOrRefresh` compares each stored session `loadRotated` returns, at
  its first read and at the re-read after a permanent failure, with the held
  session's state, before it uses the stored token or refreshes with it.
* A stored session that is the same sign-in (D2) is adopted as today.
* One that is not: nothing is adopted, used, refreshed or saved; the token
  request fails as D3 says.
* `persistRotation` already refuses to write over a store that has moved on
  (E27); it is unchanged.
* The doc comments of `OAuthSession.Token`, `TokenStore`, `reloadOrRefresh`
  and `loadRotated` say what is adopted and what is refused.

### D2. What "the same sign-in" is (Q2)

The same `Provider` (an empty one on either side takes the held session's),
the same kind (`SessionKind`'s, so an older record with no `Kind` infers
it, E24), the same issuer (trailing slash ignored) and the same `ClientID`;
and, when both name one, the same `AccountID`. Token URL, resource and host
id follow from kind and client and add nothing; scope, ID token and consents
change in a legitimate rotation and are not compared.

### D3. What a refused adoption returns, and the held session afterwards (Q3, Q4)

* The token request returns an error matching `llmprovider.ErrAuthFailure`
  that names the provider and both kinds, never a token or an account id,
  and says to sign in again. A provider's `wire.Reauth` then returns the
  401 with it (S8's S4).
* The held session keeps its own fields and tokens. Each later token
  request re-reads the store and refuses again while it holds another
  sign-in; once the store holds this sign-in again, or nothing, the session
  refreshes and continues as today.
* No new error value is exported; a caller that must tell this refusal from
  a revoked refresh token reads its own store, as gobble's resolver does.

### D4. Interfaces and exports (Q1, Q4; Q5 falls away)

No interface replaces the concrete checks under this record, and no name is
exported: the refusal is an error matching `llmprovider.ErrAuthFailure`
(D3). A caller that needs to wrap a session for another reason asks for it
in a record of its own.

### D5. Coverage (Q6)

The guard is in `OAuthSession`, so it covers every provider's refreshable
session: OpenAI's two kinds and Grok's. Kilo's device session and access-only
sessions never reach it (E26). `FileTokenStore.Load` is unchanged; the
guard's provider comparison stops S3.

### D6. The changelog, the release and gobble afterwards (Q7, Q8)

* `CHANGELOG.md`'s `[Unreleased]` gets the entry under **Security**: a
  session no longer adopts another sign-in its token store holds; its token
  requests fail with an auth failure instead, until the store holds its
  sign-in again.
* The change ships in the next release, with the Kilo profile check
  (0080-MADR-verify-a-kilo-credential-through-its-profile.md D11) and the
  three other drafts (among them the wizard option to skip the callback
  listener and the typed "tool schema refused" error), each in its own
  record. The tag waits for the owner's word; this record's PLAN must be
  finished and its gates passed before it, or this change moves to the
  release after.
* gobble then bumps its pin, adopts the reviewer's reproduction inverted
  (the held session's token request fails with an auth failure, no
  other-login token is sent, its kind stays `chatgpt_plan`), and removes the
  documented limit. It builds no wrapper. That is gobble's record to amend,
  and its owner's ruling to revisit.

### Consequences

* Good, because a held session can no longer become another login, kind,
  client, account or provider, for any consumer, whatever writes the store.
* Good, because another application's refresh token is never spent (S2 no
  longer happens), so its login is not revoked by this module's process.
* Good, because a built provider's route and its session agree for the
  provider's life, as R20 assumes.
* Good, because no export is added and `OAuthSession` stays comparable.
* Good, because it matches Codex's behaviour on an account change (X1 to X4).
* Neutral, because the guard trusts the record's labels, as every check of
  a user-writable file must: a writer forging this sign-in's labels is not
  stopped (see "The security weight").
* Bad, because a legitimate multi-process user can now see an auth failure
  and "sign in again" in a running process when, in another process of the
  same application: they sign in by the other OpenAI method; they sign in to
  another account (Q2); or they sign in again to a plan session
  without a `ClientStore`, so the issuer issues a new client (E54). Today the
  first is adopted and misrouted, the other two adopted silently.
* Bad, because a `TokenStore` that loses `ClientID`, `Issuer` or `Provider`
  on a round trip would make every sibling rotation look like another
  sign-in. `FileTokenStore` and gobble's adapter keep them (E31, E32; G3);
  the doc of `TokenStore` must say a store keeps them.
* Bad, because gobble's earlier means, interfaces and a wrapper, is set
  aside, and gobble's recorded step 1d changes with its own record.

### Confirmation

* **Tests,** in `llmprovider/auth` and `providers/openai`, with in-memory
  stores and in-process issuers only (0065-MADR), each seen failing at
  `b109cea` first and each seen failing again with the guard planted out
  (`tools/redgreen.py`, `tools/plant.py`, on a scratch clone):
  * T1: a stored session of another kind with a current token: refused, no
    request, held kind and token unchanged (S1).
  * T2: the same with its token due: no refresh request sent (S2).
  * T3: another provider's record through `FileTokenStore`: refused (S3).
  * T4: the same kind and another client: refused.
  * T5: the same kind and client and another account: refused (Q2);
    one side with no account: adopted.
  * T6: the re-read after a permanent failure (E6) refuses the same way.
  * T7: a provider built over a plan session: the retry after a 401 never
    carries the other login's token, and the error is an auth failure (S4).
  * T8: a provider built over a ChatGPT login never sends an adopted plan
    token to the Codex backend (S5).
  * T9: once the store holds the held sign-in again, or nothing, the
    session refreshes and continues (Q3).
  * T10: a sibling's rotation of the same sign-in, one whose record has no
    `Kind`, one whose scope, ID token or consents changed, and one with no
    account on one side: adopted. The existing sibling tests stay green
    (S8's list).
  * T11: a Grok session refuses another client's Grok record (Q6).
  * T12: the refusal matches `ErrAuthFailure`, and its text carries no
    token, refresh token or account id (Q4).

  The PLAN names each test and how it is seen red. Rehearsed while
  drafting, on scratch copies: eleven red at `b109cea` and T10 green; all
  twelve green with a prototype guard; twelve planted breaks of the
  prototype each caught; `-race -count=5` clean (the PLAN's execution
  record).
* **`make api-check`** and `tools/apidiff.py v1.31.0`: no change.
* **`-race`** on `llmprovider/auth`, through the fleet's queue
  (`run-on-host --race` or `run-heavy`), since the guard runs under the
  refresh lock beside the session's own lock.
* **`tools/claims.py`** on this record and its PLAN; `tools/gate.py` before
  staging.
* **Not verifiable offline:** whether a plan session's ID token carries
  `chatgpt_account_id` (if not, D2's account test is inert for plan sessions
  and the client carries the comparison; Q2 was answered with this caveat); what the Codex backend answers to
  a plan token; whether the issuer revokes the other application's family
  after S2's rotation.

## Pros and Cons of the Options

### A. Interfaces in place of the concrete assertions

* Good, because it is the means the owner named, and lets a caller compose
  token sources around a session for other purposes.
* Good, because with unexported interface types it adds no exported name.
* Bad, because it stops nothing by itself: the adoption still happens, and
  only a caller that wraps refuses, after the fact.
* Bad, because the other application's refresh token is still spent before
  a wrapper can look (S2).
* Bad, because every concrete check and every R16 owner check must change
  (E33, E34, E36 to E47), or a wrapped session is taken for an API key, or
  passes a provider that refuses OAuth sessions.
* Bad, because a wrapper must forward eleven methods; one it misses fails
  silently (no retry after a 401, the wrong identity on a refresh), and each
  method becomes a contract under the v1 promise, named or not.
* Bad, because the routing defect (S4, S5) remains for every caller that
  does not wrap.

### B. The session refuses another sign-in

* Good, because it protects every consumer by default, at the point of
  use, before anything is sent or spent.
* Good, because the change is local (one comparison, two call sites) and
  needs no export; S8's prototype passed every existing test of the four
  packages it ran.
* Good, because it matches Codex (X1 to X4) and opencode's method check
  (O1).
* Neutral, because the comparison trusts the record's labels.
* Bad, because three legitimate multi-process actions now meet an auth
  failure in the running process (see "Consequences").
* Bad, because a lossy third-party store would see every rotation refused.

### C. A caller-supplied adoption check

* Good, because a caller can apply its own policy, such as gobble's
  `Accepts`.
* Good, because it follows `RefreshLocker`'s precedent.
* Bad, because it adds an exported interface under the v1 promise.
* Bad, because only a caller that implements it is protected; every other
  consumer keeps the defect.
* Neutral, because a caller with its own store can already do this through
  D.

### D. The caller's store refuses

* Good, because it needs nothing from this module, and gobble could ship it
  today (S6).
* Good, because the held session keeps working on its own tokens.
* Bad, because only that caller is protected.
* Bad, because it changes what is persisted, and the choice falls to each
  store: overwriting the other entry destroys it; keeping it leaves the
  rotated pair in memory only, logs a failed save every 30 s, and loses the
  held login when the process ends.
* Bad, because the run silently diverges from its own file: it goes on as
  the plan session while `auth.json` names another login.
* Bad, because for gobble it is the option the owner did not choose, put to
  the owner with the third review's evidence in view.

### E. Do nothing; document

* Good, because it costs nothing.
* Bad, because the defect, the misrouting and the spent foreign refresh
  token remain for every consumer, and gobble's limit stays documented
  indefinitely.

## Decided

The questions the draft put to the owner, in the order they were asked, each
with the owner's answer of 2026-11-17, relayed by the coordinator. Each answer
is the option the draft recommended; its label is quoted as the owner chose
it, and what was asked is beside it. The questions as drafted, with their
three options, are kept in "The questions as drafted" below.

* **Q1 (D1, D4), the means.** Answered: **"The SDK refuses to adopt another
  sign-in (Recommended)"**. As asked: option B, the session refuses a stored
  session that is not the same sign-in; no interfaces; gobble builds no
  wrapper. It was put to the owner as differing from gobble's earlier ruling
  **"Ask the SDK for interfaces, then build as ruled"**. Not chosen:
  interfaces as ruled for gobble, with gobble's wrapper and the adoption
  kept; B and the interfaces both.
* **Q2 (D2), the same sign-in.** Answered: **"Provider, kind, issuer,
  client, and account when both have one (Recommended)"**. As asked: those
  fields compared, the account only when both sessions name one; the
  question carried the caveat that whether OpenAI's plan ID token carries an
  account id is not verified. Not chosen: provider and kind only; the same
  without the account.
* **Q3 (D3), the held session after a refusal.** Answered: **"Fail its token
  requests until the store holds its sign-in again, or nothing
  (Recommended)"**. As asked: each token request fails while the store holds
  another sign-in; the session keeps its own fields and resumes once the
  store holds its sign-in again or holds nothing. Not chosen: going on in
  memory with its own refresh token and never saving; going on and saving
  over the other entry.
* **Q4 (D3), the refusal's error.** Answered: **"The existing auth-failure
  error, nothing new exported (Recommended)"**. As asked: an error matching
  `llmprovider.ErrAuthFailure`, no new sentinel or type. Not chosen: a new
  exported sentinel that also matches it; a typed error carrying both kinds.
* **Q5 (D4), the stability tier.** Not asked. It falls away because nothing
  is exported under Q1 and Q4's answers: the coordinator's application of
  Q4's answer, not a ruling of the owner's.
* **Q6 (D5), the sessions covered.** Answered: **"Every refreshable session,
  any provider (Recommended)"**. As asked: the guard sits in `OAuthSession`
  and covers OpenAI's two kinds and Grok's; Kilo's device session and
  access-only sessions never reach it. Not chosen: OpenAI's sessions only;
  only sessions whose caller opts in.
* **Q7 (D6), the changelog.** Answered: **"Security (Recommended)"**. As
  asked: the entry goes under the changelog's Security heading. Not chosen:
  Fixed; Changed.
* **Q8 (D6), the release.** Answered, in a question asked for four SDK
  drafts together: **"All four in the next release with 0080, on my word
  (Recommended)"**. As asked: this record, the three other drafts and 0080
  ship in the next release, tagged on the owner's word; each record's plan
  must finish and pass its gates before the tag, and one that is not ready
  when the owner gives the word moves to the release after. Not chosen, as
  drafted here: a patch release of its own first; a later release.

### The questions as drafted

In the order they were asked; each answer is above.

* **Q1. The means.** (a) B: the session refuses to adopt another sign-in;
  no interfaces; gobble builds no wrapper (recommended). (b) A, as ruled for
  gobble: interfaces; gobble builds its wrapper; the adoption stays. (c) B
  and A.
* **Q2. What "the same sign-in" is.** (a) Provider, kind, issuer and
  client, and the account when both name one (recommended). (b) Provider
  and kind only. (c) (a) without the account.
* **Q3. The held session after a refusal.** (a) The token request fails
  until the store holds its sign-in again, or nothing (recommended). (b) It
  goes on in memory with its own refresh token and never saves over the
  other entry; the rotated pair is lost when the process ends. (c) It goes
  on and saves over the other entry.
* **Q4. The refusal's error.** (a) An error matching `ErrAuthFailure`, no
  new export (recommended). (b) A new exported sentinel in `auth` that also
  matches it. (c) A typed error carrying both kinds.
* **Q5. The stability tier of anything exported.** (a) Nothing exported
  (recommended). (b) Under `llmprovider/x/`. (c) Stable under the v1
  promise.
* **Q6. The sessions covered.** (a) Every refreshable `OAuthSession`, any
  provider (recommended). (b) OpenAI's only. (c) Only sessions whose caller
  opts in (C).
* **Q7. What the changelog calls it.** (a) Security (recommended).
  (b) Fixed. (c) Changed.
* **Q8. The release.** (a) The next release with the other items, on the
  owner's word (recommended). (b) A patch release of its own first. (c) A
  later release.

## More Information

### Records this decision amends

* [0012-MADR-conform-providers-to-reference-clients.md](0012-MADR-conform-providers-to-reference-clients.md)
  §5.2: "it adopts the stored session" becomes "it adopts the stored session
  when it is the same sign-in".
* [0016-MADR-provider-auth-and-support-baseline.md](0016-MADR-provider-auth-and-support-baseline.md)
  D4 and amendment A1: the re-read after a rejected refresh adopts only the
  same sign-in.
* Each gets a dated note pointing here; neither is rewritten.

### Related records

* [0020-MADR-remediate-v1-debugging-pass-findings.md](0020-MADR-remediate-v1-debugging-pass-findings.md)
  F1 and
  [0026-MADR-remediate-v1-2-debugging-pass-findings.md](0026-MADR-remediate-v1-2-debugging-pass-findings.md)
  F1 and F2: the spent-token rule and R16's owner check.
* [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)
  D1 and D2: the plan session and its route.
* [0074-MADR-one-sign-in-handle-and-method-facts.md](0074-MADR-one-sign-in-handle-and-method-facts.md)
  D5: `Kind` and its inference.
* [0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md](0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md)
  D3 and D7: the consents a refresh keeps, and the plan client's reuse.
* [0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md](0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md)
  D2: an adopted rotation keeps the session's identity.
* [0080-MADR-verify-a-kilo-credential-through-its-profile.md](0080-MADR-verify-a-kilo-credential-through-its-profile.md)
  D11: the shared release.

### gobble-cli's records (another repository; cited by filename)

All in gobble-cli's working tree on branch `p3-acpext-v0.5.0`, staged over
`cecc892` and not committed on 2026-11-17:

* `docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`:
  P4 step 1d (line 453); "Second review and fixes", the owner's rulings
  (lines 1532 to 1538) and R2-B2 (lines 1548 to 1555).
* `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`:
  D2's note of 2026-11-17 (line 334).
* `docs/architecture.md` (the `llm/provider` row, line 111) and
  `docs/guides/credentials.md` (lines 309 to 311): the documented limit.

### The plan

[0082-PLAN-refuse-to-adopt-another-sign-in-at-refresh.md](0082-PLAN-refuse-to-adopt-another-sign-in-at-refresh.md):
records; the tests and the guard with its docs in one commit; planted
breaks and gates; an independent review; the release on the owner's word;
gobble's follow-up named as gobble's own phase.

### Not verified

* Whether OpenAI's plan ID token carries `chatgpt_account_id`, so whether a
  plan session's `AccountID` is set: no plan sign-in was run.
* Whether the Codex backend refuses a plan token, and whether the platform
  refuses a ChatGPT login's token on the plan route (S4 and S5 used
  in-process servers).
* That the issuer revokes the other application's token family after S2's
  rotation: inferred from 0016-MADR A2's facts (E67), not measured.
* Whether any third-party `TokenStore` in use drops `ClientID`, `Issuer` or
  `Provider`: none is known; gobble's keeps them.
* S1 to S8 ran without `-race`.
