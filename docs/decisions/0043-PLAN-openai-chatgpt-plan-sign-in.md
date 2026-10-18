---
status: in-progress
date: 2026-10-18
associated-madr: "0043-MADR-openai-chatgpt-plan-sign-in.md"
decision-makers: repository owner
---

<!-- markdownlint-disable MD013 -->

# Implement OpenAI's Sign in with ChatGPT for Agents as an OpenAI Auth Method

Associated MADR: [0043-MADR-openai-chatgpt-plan-sign-in.md](0043-MADR-openai-chatgpt-plan-sign-in.md)

## Goal

A consumer signs a user in with their ChatGPT plan under the consumer's own
app name, and uses the `openai` provider with that session: generate,
stream, tools, listing, refresh and sign-out. No existing session, key,
stored record or wire golden changes behaviour.

## Scope

### In scope

* `llmprovider/auth`: the session's fields and kind (D1), refresh and
  revocation (D2), the store record (D3), `LoginChatGPTPlan` (D4).
* `llmprovider/providers/openai`: the plan request shape, listing and
  `ChatGPTUsageURL` (D5, D6); `llmprovider/internal/chatgpt`'s session
  check; `catalog.ListRaw` for a plan session.
* `llmprovider/api_error.go`: the usage-limit code (D6).
* `wizard`: the method, `Options.AgentName`, the notice (D7).
* Docs: `auth`'s package doc and the README (D8), the API standards'
  rows, `docs/architecture.md`, the release notes.
* The live test that replaces the probe (D9).

### Out of scope

* Retiring the Codex-id logins (D7).
* A device flow; OpenAI offers none.
* Native `tool_search`, continuation, or any unsupported field for plan
  sessions.
* Any change to an existing G-wire golden.

## 0. Preconditions and conventions

* **Approval.** The owner accepts the MADR and approves this PLAN, phase by
  phase or whole.
* **Gate.** Each phase ends on `tools/gate.py` (23 checks), `make
  pre-add-check` on its Go files, `tools/apidiff.py` against `v1.14.0`,
  `tools/denyscan.py --changed` and `make records-check`, and is staged for
  the owner's `git commit --no-edit`.
* **Red first.** Every new test is seen to fail on a planted break, by
  `tools/plant.py` on a scratch clone, and the plant table goes in this
  record.
* **Stop conditions.**
  * An existing G-wire golden changes.
  * An existing session's refresh, revocation or backend changes.
  * A live run contradicts M1–M9.
  * A step needs a module outside the dependency rule.

## Implementation Steps

### Phase 0: baseline and names (records only)

1. Record `<base>`, `tools/gate.py` at `<base>`, and the coverage of
   `auth`, `providers/openai` and `wizard`.
2. Check every exported name in MADR D1, D4, D5, D6 and D7 against
   `docs/guides/api-standards.md`. A rule a name breaks becomes a deviation
   and a MADR amendment before Phase 1.
3. Re-read the discovery document and compare it with M9.

### Phase 1: the session (D1, D2, D3)

1. `OAuthSession` gains `Resource`, ~~`Scopes`~~ `Scope` (A1), `IDToken` and `HostID`;
   `PlanUsage()`; `ChatGPT()` false for a plan session. `String`,
   `GoString`, `MarshalJSON` and `LogValue` hide `IDToken`.
2. The refresh of a plan session: a form to `TokenURL` with `resource` and
   no `scope`. The lock, re-read and rotation are unchanged.
3. `RevokeOAuthSession` for a plan session: an RFC 7009 form to the
   discovered `revocation_endpoint`, with the issued client id, refused
   when it is off the issuer's origin (A2).
4. `fileRecord` gains the four fields, `omitempty`. `ValidateOAuthSession`
   requires `TokenURL` for a plan session.
5. **Tests** (fake issuer): each field round-trips the file store; an
   older record (no new fields) loads as before; `PlanUsage` and
   `ChatGPT()` for each kind; the refresh form's fields and the absence of
   `scope`; the revocation form and endpoint; the R33 formatting test with
   a planted ID token; a Codex-id session's refresh and revocation
   byte-for-byte unchanged.

### Phase 2: the login (D4)

1. `ChatGPTPlanOptions` and `LoginChatGPTPlan`, reusing the loopback
   listener, PKCE, the callback handler, discovery and the ID-token
   verifier.
2. The first sign-in's and the re-sign-in's authorize parameters; the
   issued client id from the callback; a mismatched client id refused; the
   direct scope required; `AgentName` required; the pasted-callback
   fallback.
3. **Tests** (fake issuer): both authorize URLs, parameter by parameter;
   `localhost` never used; the exchange with the issued id; a bad
   signature *(covered by the shared verifier's tests, not here; see the
   Phase 2 record)*, audience or nonce; a grant without the direct scope; an
   empty `AgentName`; a callback with no client id or another one; the
   generated host id's form.

### Phase 3: the provider (D5, D6)

1. `internal/chatgpt.IsSession` false for a plan session (it follows
   `ChatGPT()`); `providers/openai` detects `PlanUsage()` and uses the
   platform URL, `store: false`, `stream: true`, no `max_output_tokens`,
   Continuation `Unsupported`, deferred tools appended, `prompt_cache_key`
   and `include` as with `WithStore(false)`. *(A4: a system message in the
   input is refused before the network.)*
2. `ListModels` and `catalog.ListRaw` read the `models` array; no probes.
3. `subscription_sharing_usage_limit_exceeded` joins `openAIQuotaTypes`;
   `openai.ChatGPTUsageURL`. *(A4: the other three codes are classified by
   code, in a reply and inside a stream.)*
4. **Tests:** new G-wire goldens for the plan shape (text, tool call,
   reasoning, deferred tools); every existing golden unchanged; the
   listing from a `models` body and the refusal of a `data` body; no probe
   sent; the four `subscription_sharing_*` codes classified, in a reply
   and mid-stream; the capabilities.

### Phase 4: the wizard (D7)

1. `AuthChatGPTPlan` in `llmprovider/descriptor.go` and the OpenAI
   descriptor; `wizard.Options.AgentName`; the method offered only with
   it and a store; the one-time notice; save and logout through the
   store.
2. **Tests:** offered and not offered; the notice shown once; the session
   saved under `openai`, replacing a Codex-id one; logout revokes it.

### Phase 5: docs, the live test, and the release

1. `auth`'s package doc and the README: the method, and what stays the
   consumer's (D8). `docs/guides/api-standards.md` rows and
   `docs/architecture.md` for any layout change (R43).
2. `llmprovider/live_chatgpt_plan_test.go`, switched on by
   `LLMPROVIDER_LIVE_CHATGPT_PLAN`: sign in, `Generate`, `Stream`, a tool
   call, `ListModels`, a forced refresh, revoke, and a refresh after it
   refused. `live_openai_signin_probe_test.go` and its helpers are removed;
   `AGENTS.md`'s live-test list names the new variable.
3. The live run, with the owner signing in.
4. Release notes for `v1.15.0`, the downgrade cost of D3 among them. The
   owner commits, pushes and tags; after the tag: CI, the proxy,
   `tools/consumer.py` and `tools/offline.py`; issue 2 is answered and
   closed on the owner's ask.

## Verification

| # | Criterion | Check | Phase |
| :--- | :--- | :--- | :--- |
| V1 | A plan session stores, refreshes and revokes as M7 and M8 measured | fake-issuer tests; the live test | 1, 5 |
| V2 | Every existing session, record and golden behaves as before | the unchanged-behaviour tests; G-wire | 1, 3 |
| V3 | The login follows OpenAI's flow and refuses what D4 refuses | fake-issuer tests | 2 |
| V4 | The provider's plan requests match D5, and its errors D6 | goldens; classification tests | 3 |
| V5 | The wizard offers the method only with an app name and a store | wizard tests | 4 |
| V6 | The API changes are additions only, and exactly the MADR's | `make api-check`; `tools/apidiff.py v1.14.0` | every |
| V7 | The live test passes with the owner signing in | the run's log | 5 |
| V8 | Every new test failed first on a planted break | `tools/plant.py` tables | every |

## Rollout and Rollback

* Additive: a consumer that never calls `LoginChatGPTPlan` sees no change.
  The release is `v1.15.0`.
* Rollback before the tag: revert the phase's commit. After it: a patch
  release that stops offering the method in the wizard; stored plan
  records then fail to refresh, and the user signs in with the Codex-id
  login (D3).

## Execution record

The owner committed the pair as `8969f1c`, accepted the MADR and approved
this PLAN ("committed, accepted, approved to proceed", 2026-10-18).
Phase 1 is `188330e`, and Phase 2 is `180f01e`.

### Phase 0: baseline and names (2026-10-18)

* **`<base>`** is `8969f1c`, the tree clean.
* **`tools/gate.py` at `<base>`:** 23 of 23.
* **Coverage at `<base>`** (`go test -count=1 -cover`): `auth` 86.6 %,
  `providers/openai` 94.3 %, `wizard` 89.6 %, `internal/chatgpt` 96.7 %.
  `wizard`'s floor is 86.4; the other three have the 80 % floor
  (`scripts/coverage-floors.txt`).
* **Step 3, discovery.** The document fetched again is byte-for-byte the one
  read for M9 (`cmp`, exit 0).
* **Step 2, the names against `docs/guides/api-standards.md`:**
  * R2: `auth`, `internal/chatgpt` and `providers/openai` keep their
    imports; no package is added.
  * R21: `AuthChatGPTPlan` is an `AuthMethodID` constant with the type's
    prefix.
  * R29: `openai.ChatGPTUsageURL` is a constant.
  * R33: `IDToken` is hidden by the session's four formatters (Phase 1).
  * R40: `LoginChatGPTPlan` takes the context first.
  * R19 governs provider configuration; `ChatGPTPlanOptions` is a flow's
    options, as the exported `OAuthFlowOptions` already is.
  * A fake issuer for Phase 2's tests needs no exported field: `auth`'s
    tests are in package `auth`, so `ChatGPTPlanOptions` takes an
    unexported issuer, as `OAuthFlowOptions` takes its unexported clock.
  * **R48 is broken by D1's `Scopes []string`.** A slice field makes
    `OAuthSession` incomparable, which `make api-check` reports as an
    incompatible change against `v1.14.0`. 0021-PLAN's deviation of
    2026-10-04 met the same failure with a func field (`old is comparable,
    new is not`), and `tokenstore.go:58` and `:62` keep the struct
    comparable on purpose.
* **Deviation 2026-10-18: D1's `Scopes []string` breaks R48.** Found by
  step 2, before any code; nothing is changed.
  * **Resolutions offered:** `Scope string` (recommended); an unexported
    slice with an accessor; accepting the break.
  * **Decided:** the owner chose `Scope string`. MADR amendment A1 records
    it; D1 and Phase 1's step 1 are annotated. No file is added to scope.

### Phase 1: the session (2026-10-18)

* **Deviation 2026-10-18: D4's origin check has no precedent.** Found
  reading `oauth_loopback.go` before writing the revocation: D4 says the
  plan login refuses off-origin endpoints "as `LoginBrowserOAuth` does",
  and `requireEndpoints` (`:257`) only checks presence. Pre-existing in
  the MADR's text; no code was written on it.
  * **Resolutions offered:** the check for plan sessions only
    (recommended); presence only; the check everywhere, which changes
    existing sessions.
  * **Decided:** the owner chose the check for plan sessions only. MADR
    amendment A2 records it; D4 and step 3 are annotated. Phase 2's
    login uses the same helper.

* **Done.**
  * **Step 1.** `OAuthSession` (`tokenstore.go`) gains `Resource`, `Scope`,
    `IDToken` and `HostID`, and `PlanUsage()` (`oauth_session.go`).
    `ChatGPT()` is false when `Resource` is set. `String`, `GoString`,
    `MarshalJSON` and `LogValue` show the plan fields, the ID token masked,
    only on a session that has one (`planFields`), so every other session
    formats as before.
  * **Step 2.** A plan session's refresh is a form with `resource` and no
    `scope`; a reply's `scope` and `id_token` replace the session's, and a
    reply without them keeps the session's. `oauthSessionState`,
    `refreshState`, `adopt` and `persistable` carry the four fields.
  * **Step 3.** `RevokeOAuthSession` sends a plan session to
    `planRevokeRequest`: the discovered `revocation_endpoint`, an RFC 7009
    form with the issued client id. Grok's form is now `formRevokeRequest`,
    shared, and sends the same bytes.
  * **A2's helper.** `llmprovider/auth/oauth_plan.go` holds
    `planOAuthEndpoints` (the discovery document as published, the
    built-in endpoints of M9 for OpenAI's issuer only when discovery fails
    or leaves one out) and `requireSameOrigin`.
  * **Step 4.** `fileRecord` gains `resource`, `scope`, `id_token` and
    `host_id`, each `omitempty`. `ValidateOAuthSession` refuses a plan
    session without a token URL.
  * **Step 5, tests,** `llmprovider/auth/plan_session_0043_test.go`, 14
    tests: the store round trip; an older record, and a Codex-id record
    that gains no key; the kind of six sessions; the refresh form, with and
    without the reply's scope and ID token; the saved rotation; `adopt`
    and `persistable`; a Codex-id refresh still Codex's JSON; the
    revocation form; an off-origin revocation endpoint refused with no
    request sent; an issuer with none, unsupported; the built-in fallback;
    `requireSameOrigin`; validation; and the formatters, with a planted ID
    token, and a Codex-id session's `String` byte for byte.
* **Red first.** `tools/plant.py`, 21 plants on a scratch clone:

  | Plant | First run | Second run |
  | :--- | :--- | :--- |
  | Load drops the resource | caught | caught |
  | Save drops the ID token | caught | caught |
  | the record always writes resource | caught | caught |
  | ChatGPT() ignores the resource | caught | caught |
  | PlanUsage ignores the provider | caught | caught |
  | a plan refresh is sent as JSON | caught | caught |
  | a plan refresh omits the resource | caught | caught |
  | a refresh drops the reply's scope | caught | caught |
  | a refresh without an ID token loses the session's | caught | caught |
  | the refresh state drops the resource | caught | caught |
  | adopt drops the host id | **not caught** | caught |
  | persistable drops the scope | **not caught** | caught |
  | a plan session revokes as Codex | caught | caught |
  | the plan endpoints skip the origin check | caught | caught |
  | the origin check ignores the scheme | caught | caught |
  | the built-in fallback is gone | caught | caught |
  | a plan session validates without a token URL | caught | caught |
  | String shows the ID token | caught | caught |
  | MarshalJSON shows the ID token | caught | caught |
  | LogValue shows the ID token | caught | caught |
  | every session shows plan fields | caught | caught |

  The two misses: a refresh keeps the host id and scope it started with,
  so a dropped field is invisible there; `adopt` matters when another
  process's rotation is taken from the store, and `persistable` when a
  failed save is retried. `TestPlanSession_AdoptAndPersistable_0043` now
  checks each directly; the second run's failures read `adopt: … host
  "urn:uuid:4f1c…"; want next's` and `persistable: … scope ""`.
* **Gate.**
  * `make pre-add-check` on the six Go files: `6 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)`. gofmt realigned two
    struct blocks that the new commented fields split; whitespace only.
  * `tools/apidiff.py v1.14.0`: five lines, each `added`:
    `(*OAuthSession).PlanUsage`, `OAuthSession.HostID`, `.IDToken`,
    `.Resource`, `.Scope`. `api-check` passes: the struct stays comparable
    (A1).
  * `tools/gate.py`: 23 of 23. `auth` coverage 87.0 % (86.6 % at `<base>`).
  * `tools/denyscan.py --changed`: 0 hits. `make records-check`: 0
    problems.
* **Not done here:** nothing calls the new fields yet. A plan session given
  to `openai.New` now fails `chatgpt.IsSession` and would be treated as a
  platform token until Phase 3; no release carries this phase alone.

### Phase 2: the login (2026-10-18)

* **Done.**
  * **Step 1.** `ChatGPTPlanOptions` and `LoginChatGPTPlan`, in
    `llmprovider/auth/oauth_plan.go`. It reuses the loopback listener
    (`browserListener`, ports 1455 and 1457, `127.0.0.1`), PKCE, the
    callback handler, `planOAuthEndpoints` (Phase 1) and the ID-token
    verifier (`verifiedSession`). The fake issuer is an unexported
    `issuer` field, as Phase 0 planned.
  * **Step 2.** The first sign-in sends `dynamic_agent_client`,
    `agent_name_hint` and `ext_agent_host_id`; a re-sign-in sends the
    issued client, `id_token_hint` and the host id, without the name. Both
    send the six scopes, `resource`, PKCE `S256`, `state` and `nonce`.
    The callback must name an issued client on a first sign-in, and may not
    name another one on a re-sign-in. The exchange sends the issued client
    and `resource`. A grant without `chatgpt.tokens.use.direct` is refused,
    matching `ErrNotPermitted`; so is a token response with no `scope`,
    since OpenAI's page says to check the granted scopes. An empty
    `AgentName`, or a `Previous` that is not a plan session with a client,
    matches `ErrInvalidRequest` before any request. The host id is the
    option's, else `Previous`'s, else a new version 4 `urn:uuid`.
  * **The callback** (`oauth_loopback.go`): `oauthCallbackResult` carries
    the callback's `client_id`, from the listener and from a pasted URL
    (`parseOAuthCallback`, which `parseOAuthInput` now wraps with the same
    behaviour and messages). The Codex and Grok logins ignore it.
    `oauthTokenResponse` gains `scope`.
  * **Step 3, tests,** `llmprovider/auth/plan_login_0043_test.go`, 12
    tests against a fake issuer: the first sign-in's authorize query
    parameter by parameter, the `127.0.0.1` redirect, the token form and
    the session; the re-sign-in, with and without the callback repeating
    the client; three refused callbacks, none exchanged; a grant without
    plan usage, and one with no scope; an ID token for another audience,
    and one with another nonce; four invalid options with no request; an
    off-origin token endpoint, refused before the browser opens; the pasted
    callback; a given host id; and `parseOAuthCallback`. Three runs under
    `-race` are clean.
* **Deviation 2026-10-18: `unparam` on `randomBase64URL`.** Found by `make
  pre-add-check`: `oauth_pkce.go:27:22: randomBase64URL - byteCount always
  receives 32 (unparam)`. HEAD's three callers all pass 32 and lint clean;
  the login's two more cross the linter's threshold. Not pre-existing as a
  failure.
  * **Resolutions offered:** drop the parameter (recommended); a second
    helper in `oauth_plan.go`, which duplicates one to pass a lint.
  * **Decided:** the owner chose to drop it. `randomBase64URL()` makes 32
    bytes, and the five calls lose the argument; no behaviour changes.
    `llmprovider/auth/oauth_pkce.go` is added to this phase's files.
* **Red first.** `tools/plant.py`, 23 plants on a scratch clone.
  * **The first run hung** at "a Codex session is accepted as Previous":
    with that check broken, `TestLoginChatGPTPlan_InvalidOptions_0043`
    reached a login that falls back to the built-in endpoints and waits
    ten minutes for a browser. The run was stopped. The test now gives
    each case a five-second deadline, so a regression fails fast.
  * **The second run** caught all 23; the hung plant's failure read
    `Previous Codex: err = oauth: browser login timed out: context
    deadline exceeded, want ErrInvalidRequest` in 5.00 s.

  | Plant | Second run |
  | :--- | :--- |
  | the first sign-in sends Codex's client | caught |
  | the first sign-in omits the app's name | caught |
  | a re-sign-in sends the app's name | caught |
  | a re-sign-in omits the ID token hint | caught |
  | the authorize request omits the resource | caught |
  | the authorize request omits the nonce | caught |
  | the authorize request asks Codex's scopes | caught |
  | a re-sign-in makes a new host id | caught |
  | a given host id is ignored | caught |
  | a first sign-in without an issued client is accepted | caught |
  | the registration client is kept | caught |
  | a re-sign-in accepts another client | caught |
  | the exchange omits the resource | caught |
  | the exchange sends the registration client | caught |
  | a grant without plan usage is accepted | caught |
  | the ID token is not verified | caught |
  | the nonce is not checked | caught |
  | an empty app name is accepted | caught |
  | a Codex session is accepted as Previous | caught |
  | the session is not marked a plan session | caught |
  | the callback drops the issued client | caught |
  | a pasted callback drops the issued client | caught |
  | the callback client keeps its spaces | caught |

  The plants ran before the deviation's rename, which touches no planted
  line.
* **Gate.**
  * `make pre-add-check` on the four Go files: `4 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)`.
  * `tools/apidiff.py v1.14.0`: Phase 1's five lines, and
    `auth.ChatGPTPlanOptions` and `auth.LoginChatGPTPlan`, each `added`.
  * `tools/gate.py`: 23 of 23. `auth` coverage 86.9 % (87.0 % after Phase 1, 86.6 % at `<base>`).
  * `tools/denyscan.py --changed`: 0 hits. `make records-check`: 0
    problems.
* **Not done here.**
  * **No signature case in the plan tests.** Step 3 planned one. The login
    calls the shared verifier (`verifiedSession`), whose tests already
    refuse a bad signature (`TestVerifyIDToken_Rejects`, "tampered
    payload"). The plant that skips verification altogether is caught by
    the audience and nonce cases. A plant that broke only the signature
    check would be caught by the verifier's tests, not by this file's.
  * **Untested:** `id_token_hint` left out when `Previous` has no ID token;
    the login failing when both loopback ports are taken; a bare pasted
    code on a first sign-in (it follows from two tested pieces). 0043-MADR
    A3 lists these.
  * Nothing outside `auth` calls the login yet. A plan session given to
    `openai.New` is still treated as a platform token until Phase 3.
* **Carried to Phase 3.** A plan session has `AccountID` and `FedRAMP` from
  its ID token, as every OpenAI login does, so `Account()` answers. The
  provider must not send them as the Codex backend's headers (D5); Phase
  3's goldens assert their absence.
* **Records aligned (2026-10-18).** At the owner's request, after the
  commit, the code at `180f01e` was read against this record and 0043-MADR.
  The MADR gains A3, which records what the login settled that D4 left
  open. D3's store field reads `scope` (A1), and A2 says that a refresh is
  not origin-checked. Nothing in the code changed. The Phase 2 entries
  above stand as written, except for step 3's annotation.
* **Discovery's fallback accepted (2026-10-18).** The owner accepted the
  built-in endpoint fallback, as Phase 1 built it, over two alternatives:
  failing closed, and keeping the fallback for revocation only. 0043-MADR
  A3 records why. No code changed, and no file was added to any phase.

### Phase 3: the provider (2026-10-18)

The owner approved Phase 3 ("committed, proceed", 2026-10-18), after the
records alignment (`d8efb45`).

* **Deviation 2026-10-18: two rules D5 and D6 left out.** Found reading
  `llmprovider/api_error.go` and `llmprovider/internal/wire/responses`
  before any code. Neither is a failure in existing code; both concern
  plan sessions only.
  * **The codes inside a stream.** `ClassifyStreamFailure` classifies an
    unknown code as a 500. So `subscription_sharing_user_not_eligible`,
    `_route_not_supported` and `_unsupported_capability` inside a stream
    would be `ErrProviderUnavailable` and retried, though step 4 tests them
    as D6's kinds.
    * **Resolutions offered:** classify them by code (recommended); test
      them as HTTP replies only, and annotate step 4.
    * **Decided:** classify by code.
  * **A system message in the input.** OpenAI rejects it for plan
    requests (0017-REPORT F2), and the provider sends one as `role:
    "system"`.
    * **Resolutions offered:** refuse it before the network (recommended);
      fold it into `instructions`; leave it for OpenAI to refuse.
    * **Decided:** refuse it before the network.
  * 0043-MADR A4 records both, and steps 1 and 3 are annotated. Nothing
    is added to the phase's files: `llmprovider/api_error.go` was already
    in scope.
* **`<base>`** is `d8efb45`. Coverage there: `llmprovider` 97.7 %,
  `providers/openai` 94.3 %, `catalog` 95.2 %, `internal/chatgpt` 96.7 %.
* **Done.**
  * **Step 1, the session check.** `internal/chatgpt.IsSession` already
    follows `ChatGPT()`, which Phase 1 made false for a plan session.
    `internal/chatgpt.IsPlanSession` is new, and the provider and
    `catalog` share it.
  * **Step 1, the request shape** (`providers/openai/openai.go`). With a
    plan session the provider:
    * sends to `PlatformBaseURL`, with `Authorization: Bearer` and none of
      the Codex backend's headers (`session-id`, `originator`,
      `ChatGPT-Account-Id`, FedRAMP, residency), though the session holds
      an account (A3);
    * sends `stream: true`, `store: false` and `include` for encrypted
      reasoning on every request, and reads the stream in `Generate`;
    * sends no `max_output_tokens`, so `WithMaxTokens`, `MaxOutputTokens`
      and `WithStore` have no effect;
    * sends `prompt_cache_key` only for a cached request, as a key does;
    * has Continuation `Unsupported`;
    * appends deferred tools (`deferral.Append`), with
      `WarningDeferralEmulated`, and never sends `tool_search`;
    * refuses a `RoleSystem` message in `Input` before the network, matching
      `ErrUnsupported`, and names `Request.Instructions` (A4).

    The package doc lists each of these.
  * **Step 2, the listing.**
    * `ListModels` sends `internal/chatgpt.PlanModelsRequest`: GET
      `/models`, with the bearer token, `Accept: application/json`, no
      `client_version` and no Codex headers.
    * It keeps the `models` entries with `visibility` `list` (trimmed,
      any case) and a slug, in the listing's order, once each.
    * It never probes, even with `WithModelProbes(true)`. A 401 renews the
      token once, through `wire.Reauth`, as the ChatGPT listing does; that
      is untested for the plan listing.
    * A body with no `models` array (a key's `data` listing), a catalog
      that lists nothing, and a body that does not decode are each
      `ErrIncomplete`. There is no static fallback, as for a ChatGPT
      session.
    * `fetchCatalog` now holds the reading that the ChatGPT listing already
      did, with that listing's messages unchanged.
    * `catalog.ListRaw` sends the same request and splits the `models`
      array by `slug` (`fetchPlanRaw`).
  * **Step 3, the errors** (`llmprovider/api_error.go`).
    * `subscription_sharing_usage_limit_exceeded` joins `openAIQuotaTypes`.
    * Per A4, `subscription_sharing_user_not_eligible` and
      `_route_not_supported` are `ErrNotPermitted`, and
      `_unsupported_capability` is `ErrInvalidRequest`.
    * All four are terminal, in a reply or a stream, for the `openai`
      service only.
    * `openai.ChatGPTUsageURL` is `https://chatgpt.com/settings/usage`.
  * **Step 4, tests.**
    * **The wire case.** `chatgpt-plan` is a new case in
      `providers/openai/wire_test.go`. Its session has an account id, and
      it is built with `WithMaxTokens(2048)`, so the goldens show that
      neither is sent. Its 12 goldens were recorded with `-update` for this
      case only, and read in full:
      * text, tool, thinking, thinking and tool, items, flags;
      * `cache-short` and `cache-long`, which send the key;
      * `defer-offer` (no deferred tool) and `defer-load` (`lookup_stock`
        appended, no `tool_search` entry);
      * `continuation`, refused with no request sent;
      * `listing`, which keeps the two listed models.
    * **Unchanged goldens.** `git status` shows no existing golden
      changed, and the existing tests of the four packages pass.
    * **`providers/openai/plan_0043_test.go`, six tests:**
      * the request, the URL and the headers;
      * the refusals, through `Generate` and `Stream`, with no request
        sent;
      * a ChatGPT session's system message still sent;
      * deferred tools appended, with the warning;
      * the listing, its order, no probe, and three refused bodies;
      * the four codes inside a stream, each terminal.

      A seventh, `TestChatGPTUsageURL_0043`, pins the usage page.
    * **Elsewhere:**
      * `llmprovider/api_error_0043_test.go`: the four codes in a reply
        and in a stream, and another service's stream unchanged;
      * `internal/chatgpt/plan_0043_test.go`: `IsPlanSession` and
        `PlanModelsRequest`;
      * `catalog/raw_0041_test.go`: `TestListRaw_ChatGPTPlan_0043`.
* **Red first.** `tools/plant.py`, 26 plants on a scratch clone, all caught
  on the first run. Each failed on an assertion, not on a build error.

  | Plant | Result |
  | :--- | :--- |
  | the usage limit is not a quota code | caught |
  | the refusal codes are not named | caught |
  | the capability code is not named | caught |
  | no session is a plan session | caught |
  | the plan listing sends the Codex headers | caught |
  | the plan listing sends no Accept | caught |
  | the provider ignores the plan session | caught |
  | a plan session continues | caught |
  | a system message is sent | caught |
  | a plan session sends tool_search | caught |
  | a plan session sends an output limit | caught |
  | a plan session sends no store | caught |
  | a plan session always sends the cache key | caught |
  | a plan session asks no encrypted reasoning | caught |
  | Generate does not read a plan session's stream | caught |
  | a plan request asks no event stream | caught |
  | a plan session sends the Codex headers | caught |
  | a plan session lists through the key's catalog | caught |
  | the plan listing keeps hidden models | caught |
  | the plan listing repeats a model | caught |
  | the plan listing takes a data body as empty | caught |
  | the plan listing names the chatgpt catalog | caught |
  | ListRaw lists a plan session as a key | caught |
  | ListRaw sends the plan listing with Codex headers | caught |
  | ListRaw splits a plan page as data | caught |
  | the usage page moves | caught |
  | ListRaw does not classify a plan listing's failure (a second run, below) | caught |

  After the gate, `catalog` measured 95.1 %, under `<base>`'s 95.2 % but
  over its 80 % floor; the dip is `fetchPlanRaw`'s error branch.
  `TestListRaw_ChatGPTPlan_0043` gained a 503 case, and a 27th plant, which
  skips its classification, was caught alone: `a 503: err = model listing:
  no page was read; want ErrProviderUnavailable naming page 1`. `catalog`
  still rounds to 95.1 %.

  The failure of "the plan listing sends the Codex headers", for example,
  read `requests[0].header.Originator: want (absent), got
  "go-llmprovider-sdk"` in the listing golden, and `originator:
  "go-llmprovider-sdk"; want ""` in `TestPlanModelsRequest_0043`.
* **Gate.**
  * `make pre-add-check` on the ten Go files: `10 file(s) clean (gofmt,
    golangci-lint, go vet, go test, govulncheck)`. Two comment lines in
    the package doc were rewrapped, and gofmt aligned one map in a test,
    after the plants ran; neither touches a planted line.
  * `tools/apidiff.py v1.14.0`: the seven lines from Phases 1 and 2, and
    `providers/openai.ChatGPTUsageURL`, each `added`.
  * `tools/gate.py`: 23 of 23, before the 503 case; `make pre-add-check` on `catalog/raw_0041_test.go` after it: `1 file(s) clean`. Coverage after: `llmprovider` 97.7 %, `providers/openai` 95.7 %, `catalog` 95.1 %, `internal/chatgpt` 97.4 %.
  * `tools/denyscan.py --changed`: 0 hits. `make records-check`: 0
    problems.
* **Not done here.**
  * **`catalog.List`** (not `ListRaw`) with a plan session is unchanged.
    It lists the platform as for a key, as it does today for a ChatGPT
    session; D5 named only `ListModels` and `ListRaw`. The provider's
    `ListModels` does not call it for a plan session.
  * **Admission bodies** (`{"detail": …}` with 401, 403 or 503) are
    classified by status as before, per D6. No new test.
  * **`facts`** still reads the cached facts with the session's token. It
    sends nothing, and a plan session uses no fact, since it sends no
    output limit.
  * The wizard (Phase 4), and the docs, the live test and the release
    (Phase 5).
